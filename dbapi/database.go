package dbapi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	// ErrorNotFound is used when a result is not found.
	ErrorNotFound string = "Empty result, no record found"
	dateFormat    string = "2006-01-02 15:04:05.999999999 -0700 MST"
)

// Storage impelmentation for tokens.

type databaseRecord struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       string
}

type database map[string]databaseRecord

// FileSystemTokenStore is the token storage system.
type FileSystemTokenStore struct {
	storage io.ReadSeeker
}

// GetTokenRecord gets a token record.
func (f *FileSystemTokenStore) getDatabase() (database, error) {
	var database = database{}
	var err error
	if f.storage != nil {
		f.storage.Seek(0, 0)
		err = json.NewDecoder(f.storage).Decode(&database)
	}
	return database, err
}

func (f *FileSystemTokenStore) setDatabase(database database) {
	databaseBytes, _ := json.Marshal(database)
	databaseStrings := string(databaseBytes)
	f.storage = strings.NewReader(databaseStrings)
	//	json.NewEncoder(f.storage).Encode(database)
}

func (f *FileSystemTokenStore) getRecord(id string) (databaseRecord, error) {
	database, err := f.getDatabase()
	if err != nil {
		return databaseRecord{}, err
	}
	if record, ok := database[id]; ok {
		return record, nil
	}
	return databaseRecord{}, errors.New(ErrorNotFound)
}

// GetToken gets and re-hydrates an oauth2 token from the data store.
func (f *FileSystemTokenStore) GetToken(id string) (oauth2.Token, error) {
	record, err := f.getRecord(id)
	if err != nil {
		return oauth2.Token{}, err
	}
	rehydratedToken, err := f.rehydrateRecord(record)
	if err != nil {
		return oauth2.Token{}, err
	}
	return rehydratedToken, nil
}

func (f FileSystemTokenStore) rehydrateRecord(record databaseRecord) (oauth2.Token, error) {
	expiry, err := time.Parse(dateFormat, record.Expiry)
	if err != nil {
		return oauth2.Token{}, err
	}
	rehydratedToken := oauth2.Token{
		AccessToken:  record.AccessToken,
		TokenType:    record.TokenType,
		RefreshToken: record.RefreshToken,
		Expiry:       expiry,
	}
	return rehydratedToken, nil
}

func (f FileSystemTokenStore) dehydrateToken(token oauth2.Token) databaseRecord {
	return databaseRecord{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry.Format(dateFormat),
	}
}

// UpsertToken inserts a new token or updates an existing one, based on the given id.
func (f *FileSystemTokenStore) UpsertToken(id string, token oauth2.Token) error {
	database, err := f.getDatabase()
	if err != nil {
		return err
	}
	database[id] = f.dehydrateToken(token)
	f.setDatabase(database)
	return nil
}

type EncryptedReadSeeker struct {
	// The key should be the AES key, either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
	key     []byte
	storage io.ReadSeeker
	pos     int64
}

func (e EncryptedReadSeeker) encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (e EncryptedReadSeeker) decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (e *EncryptedReadSeeker) getPlainTextValue() ([]byte, error) {
	e.storage.Seek(0, 0)
	encryptedStorage, err := ioutil.ReadAll(e.storage)
	if err != nil {
		return nil, err
	}
	decryptedStorage, err := e.decrypt(encryptedStorage, e.key)
	if err != nil {
		return nil, err
	}
	return decryptedStorage, err
}

func (e *EncryptedReadSeeker) Read(p []byte) (n int, err error) {
	decryptedStorage, err := e.getPlainTextValue()
	if err != nil {
		return 0, err
	}
	byteReader := bytes.NewReader(decryptedStorage)
	bytesRead, err := byteReader.ReadAt(p, e.pos)
	e.pos = e.pos + int64(bytesRead)
	return bytesRead, err
}

func (e *EncryptedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = e.pos + offset
	case io.SeekEnd:
		decryptedStorage, err := e.getPlainTextValue()
		if err != nil {
			return 0, err
		}
		abs = int64(len(decryptedStorage)) + offset
	default:
		return 0, errors.New("bytes.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("bytes.Reader.Seek: negative position")
	}
	e.pos = abs
	return abs, nil
}

package dbapi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
)

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
	decryptedStorage := []byte("")
	// Only encrypt/decrypt if storage is not empty.
	if len(encryptedStorage) > 0 {
		decryptedStorage, err = e.decrypt(encryptedStorage, e.key)
		if err != nil {
			return nil, err
		}
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

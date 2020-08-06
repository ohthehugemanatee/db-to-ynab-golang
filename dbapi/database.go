package dbapi

import (
	"encoding/json"
	"errors"
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
	storage []byte
}

// GetTokenRecord gets a token record.
func (f *FileSystemTokenStore) getDatabase() (database, error) {
	var database = database{}
	var err error
	if f.storage != nil {
		err = json.Unmarshal(f.storage, &database)
	}
	return database, err
}

func (f *FileSystemTokenStore) setDatabase(database database) {
	// Discard error because success is guaranteed by the type and json module.
	json, _ := json.Marshal(database)
	f.storage = json
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

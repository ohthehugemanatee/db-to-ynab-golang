package dbapi

import (
	"encoding/json"
	"errors"
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
	var database database
	err := json.Unmarshal(f.storage, &database)
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
	return databaseRecord{}, errors.New("Empty result, no record found")
}

package dbapi

import "encoding/json"

// Storage impelmentation for tokens.

type databaseRecord struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       string
}

// FileSystemTokenStore is the token storage system.
type FileSystemTokenStore struct {
	storage []byte
}

// GetTokenRecord gets a token record.
func (f *FileSystemTokenStore) getTokenRecord() databaseRecord {
	var record databaseRecord
	json.Unmarshal(f.storage, &record)
	return record
}

func (f *FileSystemTokenStore) setTokenRecord(record databaseRecord) {
	json, _ := json.Marshal(record)
	f.storage = json
}

package dbapi

import "encoding/json"

// Storage impelmentation for tokens.

type databaseRecord struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       string
}

type FileSystemTokenStore struct {
	database []byte
}

func (f *FileSystemTokenStore) GetTokenRecord() databaseRecord {
	var record databaseRecord
	json.Unmarshal(f.database, &record)
	return record
}

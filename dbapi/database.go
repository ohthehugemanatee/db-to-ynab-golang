package dbapi

import "encoding/json"

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
func (f *FileSystemTokenStore) getDatabase() database {
	var database database
	json.Unmarshal(f.storage, &database)
	return database
}

func (f *FileSystemTokenStore) setDatabase(database database) {
	json, _ := json.Marshal(database)
	f.storage = json
}

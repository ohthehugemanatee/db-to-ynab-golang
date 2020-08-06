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

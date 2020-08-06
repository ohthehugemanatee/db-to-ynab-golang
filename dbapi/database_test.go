package dbapi

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestTokenFileStore(t *testing.T) {
	testToken := databaseRecord{
		AccessToken:  "accessToken",
		TokenType:    "tokenType",
		RefreshToken: "refreshToken",
		Expiry:       time.Now().String(),
	}
	t.Run("Set a token record in a file store", func(t *testing.T) {
		store := FileSystemTokenStore{}
		store.setTokenRecord(testToken)

		got := store.storage
		want, _ := json.Marshal(testToken)
		if bytes.Compare(got, want) != 0 {
			t.Errorf("Did not find the same token we set in the database. Got %s wanted %s", got, want)
		}
	})

	t.Run("Get a token record from a file store", func(t *testing.T) {
		json, _ := json.Marshal(testToken)
		store := FileSystemTokenStore{json}

		got := store.getTokenRecord()
		want := testToken

		if got != want {
			t.Errorf("Did not load the same token we set. Got %s wanted %s", got, want)
		}
	})
}

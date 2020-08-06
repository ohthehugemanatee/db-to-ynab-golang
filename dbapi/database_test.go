package dbapi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTokenFileStore(t *testing.T) {
	t.Run("Get a token from the file store", func(t *testing.T) {
		testToken := databaseRecord{
			AccessToken:  "accessToken",
			TokenType:    "tokenType",
			RefreshToken: "refreshToken",
			Expiry:       time.Now().String(),
		}

		json, _ := json.Marshal(testToken)
		store := FileSystemTokenStore{json}

		got := store.getTokenRecord()
		want := testToken

		if got != want {
			t.Errorf("Did not load the same token we set. Got %s wanted %s", got, want)
		}

	})
}

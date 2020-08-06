package dbapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

var startTime time.Time = time.Now().Round(0)

var testDatabase database = database{
	"123": databaseRecord{
		AccessToken:  "accessToken",
		TokenType:    "tokenType",
		RefreshToken: "refreshToken",
		Expiry:       startTime.Format(dateFormat),
	},
	"abc": databaseRecord{
		AccessToken:  "accessToken2",
		TokenType:    "tokenType2",
		RefreshToken: "refreshToken2",
		Expiry:       startTime.Add(time.Hour).Format(dateFormat),
	},
}

var testToken oauth2.Token = oauth2.Token{
	AccessToken:  "accessToken2",
	TokenType:    "tokenType2",
	RefreshToken: "refreshToken2",
	Expiry:       startTime.Add(time.Hour),
}

func TestTokenFileStore(t *testing.T) {
	t.Run("Save the DB to a data store", func(t *testing.T) {
		store := getTestDatabaseStore()
		got := store.storage
		want, _ := json.Marshal(testDatabase)
		if bytes.Compare(got, want) != 0 {
			t.Errorf("Did not find the same database we set. Got %s wanted %s", got, want)
		}
	})

	t.Run("Get the DB from a data store", func(t *testing.T) {
		json, _ := json.Marshal(testDatabase)
		store := FileSystemTokenStore{json}

		got, _ := store.getDatabase()
		want := testDatabase

		for i := range got {
			if got[i] != want[i] {
				t.Errorf("Did not load the right database. At index %s, got %s wanted %s", i, got, want)
			}
		}
	})

	t.Run("Return errors when getting a corrupt data store", func(t *testing.T) {
		corruptData := []byte("I've got a bad feeling about this")
		store := FileSystemTokenStore{corruptData}

		_, err := store.getDatabase()
		if err == nil {
			t.Error("Invalid JSON in the database did not produce an error")
		}
	})

	t.Run("Get an individual record from a data store", func(t *testing.T) {
		store := getTestDatabaseStore()
		for i := range testDatabase {
			got, _ := store.getRecord(i)
			want := testDatabase[i]
			if got != want {
				t.Errorf("Got an invalid record from the database. Got %s wanted %s", got, want)
			}
		}
	})

	t.Run("Get an error when trying to get a non-existent record", func(t *testing.T) {
		store := getTestDatabaseStore()
		_, got := store.getRecord("Bad command or file name")
		want := errors.New(ErrorNotFound)
		if got.Error() != want.Error() {
			t.Errorf("Got a wrong error when trying to get a non-existent record. Got %s, wanted %s", got, want)
		}
	})

	t.Run("Get an oauth2 token from a data store", func(t *testing.T) {
		store := getTestDatabaseStore()
		got, _ := store.GetToken("abc")
		assertEqualTokens(t, got, testToken)
	})

	t.Run("Set an oauth2 token in the data store", func(t *testing.T) {
		store := FileSystemTokenStore{}
		id := "testing-id"
		store.UpsertToken(id, testToken)
		got, _ := store.GetToken(id)
		assertEqualTokens(t, got, testToken)
	})
}

func assertEqualTokens(t *testing.T, got oauth2.Token, want oauth2.Token) {
	// Expiry time is different in subtle ways because of how they were generated,
	// so we compare them separately, then make sure they're identical to compare the rest.
	if !got.Expiry.Equal(want.Expiry) {
		t.Errorf("Expiry times are not the same. Got %v wanted %v", got.Expiry, want.Expiry)
	}
	got.Expiry = want.Expiry
	if got != want {
		t.Errorf("Token data is not identical. Got %+v, got %+v", got, want)
	}
}

func getTestDatabaseStore() FileSystemTokenStore {
	store := FileSystemTokenStore{}
	store.setDatabase(testDatabase)
	return store
}

package dbapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestTokenFileStore(t *testing.T) {
	testDatabase := database{
		"123": databaseRecord{
			AccessToken:  "accessToken",
			TokenType:    "tokenType",
			RefreshToken: "refreshToken",
			Expiry:       time.Now().String(),
		},
		"abc": databaseRecord{
			AccessToken:  "accessToken2",
			TokenType:    "tokenType2",
			RefreshToken: "refreshToken2",
			Expiry:       time.Now().Add(time.Hour).String(),
		},
	}
	t.Run("Save the DB to a data store", func(t *testing.T) {
		store := FileSystemTokenStore{}
		store.setDatabase(testDatabase)

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
		store := FileSystemTokenStore{}
		store.setDatabase(testDatabase)
		for i := range testDatabase {
			got, _ := store.getRecord(i)
			want := testDatabase[i]
			if got != want {
				t.Errorf("Got an invalid record from the database. Got %s wanted %s", got, want)
			}
		}
	})

	t.Run("Get an error when trying to get a non-existent record", func(t *testing.T) {
		store := FileSystemTokenStore{}
		store.setDatabase(testDatabase)
		_, got := store.getRecord("Bad command or file name")
		want := errors.New("Empty result, no record found")
		if got.Error() != want.Error() {
			t.Errorf("Got a wrong error when trying to get a non-existent record. Got %s, wanted %s", got, want)
		}
	})
}

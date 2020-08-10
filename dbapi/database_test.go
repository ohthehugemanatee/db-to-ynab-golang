package dbapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
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
		var got database
		json.NewDecoder(store.storage).Decode(&got)
		assertEqualDatabases(t, got, testDatabase)
	})

	t.Run("Get the DB from a data store", func(t *testing.T) {
		store := getTestDatabaseStore()
		got, _ := store.getDatabase()
		want := testDatabase
		assertEqualDatabases(t, got, want)
	})

	t.Run("Return errors when getting a corrupt data store", func(t *testing.T) {
		corruptData := strings.NewReader("I've got a bad feeling about this")
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

func TestEncryptionFunctions(t *testing.T) {
	key := generateEncryptionKey()
	plainText := []byte("I'm afraid the deflector shield will be quite operational when your friends arrive")
	encryptedText, err := EncryptedReadSeeker{}.encrypt(plainText, key)
	t.Run("Test encrypting text", func(t *testing.T) {
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if len(encryptedText) == 0 {
			t.Errorf("Empty encrypted text returned")
		}
		if bytes.Compare(plainText, encryptedText) == 0 {
			stringText := string(encryptedText)
			t.Errorf("Text was not encrypted! Output: %s", stringText)
		}
	})
	t.Run("Test decrypting text", func(t *testing.T) {
		decryptedText, err := EncryptedReadSeeker{}.decrypt(encryptedText, key)
		assertEqualBytes(t, decryptedText, plainText)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})

	encryptedTextReader := bytes.NewReader(encryptedText)
	plainTextReader := bytes.NewReader(plainText)
	testEncryptedReadSeeker := EncryptedReadSeeker{key, encryptedTextReader, 0}
	byteLengthToRead := 20
	t.Run("Read method should return specified number of bytes", func(t *testing.T) {
		got := make([]byte, byteLengthToRead)
		_, err := testEncryptedReadSeeker.Read(got)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		want := make([]byte, byteLengthToRead)
		_, err = plainTextReader.Read(want)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		assertEqualBytes(t, got, want)
	})
	t.Run("Read method should keep a consistent position between invocations", func(t *testing.T) {
		// Try reading again for the rest of the value.
		got := make([]byte, byteLengthToRead)
		_, err = testEncryptedReadSeeker.Read(got)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		want := make([]byte, byteLengthToRead)
		_, err = plainTextReader.Read(want)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		assertEqualBytes(t, got, want)
	})
}

func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())
	rand.Read(key)
	return key
}

func assertEqualBytes(t *testing.T, got []byte, want []byte) {
	if bytes.Compare(got, want) != 0 {
		gotString := string(got)
		wantString := string(want)
		t.Errorf("Byte arrays did not match. Got %s wanted %s", gotString, wantString)
	}
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

func assertEqualDatabases(t *testing.T, got database, want database) {
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Database entries do not match. At index %s, got %s wanted %s", i, got, want)
		}
	}
}

func getTestDatabaseStore() FileSystemTokenStore {
	testDatabaseJSON, _ := json.Marshal(testDatabase)
	testDatabaseReader := bytes.NewReader(testDatabaseJSON)
	store := FileSystemTokenStore{testDatabaseReader}
	return store
}

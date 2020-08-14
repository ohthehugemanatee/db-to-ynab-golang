package dbapi

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

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
	t.Run("Seek method can reset persistent position", func(t *testing.T) {
		testEncryptedReadSeeker.Seek(0, io.SeekStart)
		got := make([]byte, byteLengthToRead)
		_, err := testEncryptedReadSeeker.Read(got)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		want := make([]byte, byteLengthToRead)
		_, err = plainTextReader.ReadAt(want, 0)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		assertEqualBytes(t, got, want)
	})
}

func TestEncryptedFilesystemStorageIntegraton(t *testing.T) {
	tempfile, removeTempFile := createTempFile(t, "")
	defer removeTempFile()
	oldTokenStore := tokenStore
	tokenStore = FileSystemTokenStore{
		storage: &EncryptedReadSeeker{
			key:     generateEncryptionKey(),
			storage: tempfile,
			pos:     io.SeekStart,
		},
	}
	token := oauth2.Token{
		AccessToken:  "accessToken",
		TokenType:    "tokenType",
		RefreshToken: "refreshToken",
	}
	id := "test-id"
	err := tokenStore.UpsertToken(id, token)
	if err != nil {
		t.Errorf("Failed upserting test token: %v", err)
	}
	var rehydratedToken oauth2.Token

	t.Run("Make sure tokens are stored encrypted", func(t *testing.T) {
		fileContents, err := ioutil.ReadFile(tempfile.Name())
		if err != nil {
			t.Fatalf("Failed reading file: %v", err)
		}
		if fileContents == nil {
			t.Error("Nothing was written to filesystem storage")
		}
		err = json.Unmarshal(fileContents, &rehydratedToken)
		if err == nil {
			t.Error("Filesystem storage was in plaintext")
		}
	})
	t.Run("Make sure we can read from encrypted token store", func(t *testing.T) {
		rehydratedToken, err := tokenStore.GetToken(id)
		if err != nil {
			t.Errorf("Could not get a token back from the encrypted filestore: %v", err)
		}
		if rehydratedToken != token {
			t.Errorf("Saved and retrieved token didn't match. Got %v, wanted %v", rehydratedToken, token)
		}
	})
	tokenStore = oldTokenStore
}

func createTempFile(t *testing.T, initialData string) (*os.File, func()) {
	t.Helper()
	tempfile, err := ioutil.TempFile("", "test-db")
	if err != nil {
		t.Fatalf("Could not create tempfile %v", err)
	}

	tempfile.Write([]byte(initialData))

	removeFile := func() {
		tempfile.Close()
		os.Remove(tempfile.Name())
	}

	return tempfile, removeFile
}

func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())
	rand.Read(key)
	return key
}

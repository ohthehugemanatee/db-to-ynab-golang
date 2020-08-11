package dbapi

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"
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

func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())
	rand.Read(key)
	return key
}

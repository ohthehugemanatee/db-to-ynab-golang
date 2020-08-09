package dbapi

// Implements an encrypted ReadWriteSeeker for use with FileSystemTokenStore.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

type encryptedStorage struct {
	file io.ReadWriteSeeker
	key  [32]byte
}

func (e encryptedStorage) Read(p []byte) (n int, err error) {
	// @todo implement.
	return 0, nil
}

func (e encryptedStorage) Write(p []byte) (n int, err error) {
	// @todo implement.
	return 0, nil

}

func (e encryptedStorage) Seek(offset int64, whence int) (int64, error) {
	// @todo implement.
	return 0, nil
}

func (e encryptedStorage) encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (e encryptedStorage) decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

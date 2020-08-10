package dbapi

// Implements an encrypted ReadWriteSeeker for use with FileSystemTokenStore.

import (
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

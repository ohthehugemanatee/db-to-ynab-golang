package tools

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// CreateImportID creates a 32-character unique ID based on a given string.
func CreateImportID(source string) string {
	sum := sha256.Sum256([]byte(source))
	importID := fmt.Sprintf("%x", sum)[0:32]
	return importID
}

// ConvertToMilliunits converts a decimal float to YNAB API's "milliunits".
func ConvertToMilliunits(value float32) int64 {
	return int64(value * 1000)
}

func AssertStatus(t *testing.T, expected int, got int) {
	if got != expected {
		t.Errorf("Got wrong status code: got %v want %v",
			got, expected)
	}
}

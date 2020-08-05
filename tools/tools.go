package tools

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
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

type testLogBuffer struct {
	GotBuffer    *bytes.Buffer
	ExpectBuffer *bytes.Buffer
}

func (b testLogBuffer) ExpectLog(s string) error {
	_, err := b.ExpectBuffer.WriteString(s)
	if err != nil {
		return err
	}
	_, err = b.ExpectBuffer.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

func (b *testLogBuffer) TestLogValues(t *testing.T) {
	if strings.Compare(b.GotBuffer.String(), b.ExpectBuffer.String()) != 0 {
		gotLog := b.GotBuffer.String()
		wantLog := b.ExpectBuffer.String()
		t.Errorf("Got wrong log output. Got %+q\n want %+q\n", gotLog, wantLog)
	}
}

func CreateAndActivateEmptyTestLogBuffer() *testLogBuffer {
	logBuffer := testLogBuffer{
		&bytes.Buffer{},
		&bytes.Buffer{},
	}
	log.SetOutput(logBuffer.GotBuffer)
	log.SetFlags(0)
	return &logBuffer
}

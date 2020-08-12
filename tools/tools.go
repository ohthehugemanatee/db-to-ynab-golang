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

// An in-memory log store for testing log outputs.
type testLogBuffer struct {
	GotBuffer    *bytes.Buffer
	ExpectBuffer *bytes.Buffer
}

// Add a log string to the list of expected log outputs.
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

// Tests the received log against the expectations.
func (b *testLogBuffer) TestLogValues(t *testing.T) {
	if strings.Compare(b.GotBuffer.String(), b.ExpectBuffer.String()) != 0 {
		gotLog := b.GotBuffer.String()
		wantLog := b.ExpectBuffer.String()
		t.Errorf("Got wrong log output. Got %+q\n want %+q\n", gotLog, wantLog)
	}
}

// CreateAndActivateEmptyTestLogBuffer creates a new TestLogBuffer and applies it to the standard log library.
func CreateAndActivateEmptyTestLogBuffer() *testLogBuffer {
	logBuffer := testLogBuffer{
		&bytes.Buffer{},
		&bytes.Buffer{},
	}
	log.SetOutput(logBuffer.GotBuffer)
	log.SetFlags(0)
	return &logBuffer
}

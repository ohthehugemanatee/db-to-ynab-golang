package tools

import "testing"

func TestCreateImportID(t *testing.T) {
	input := "You came in that thing? You're braver than I thought"
	got := CreateImportID(input)
	gotLength := len(got)
	expectedLength := 32
	if gotLength != expectedLength {
		t.Errorf("Import ID had the incorrect number of characters. Expected %d, got %d", expectedLength, gotLength)
	}
	secondRunGot := CreateImportID(input)
	if got != secondRunGot {
		t.Error("Unique ID was not identical for two runs on the same input string.")
	}
}

func TestConvertToMilliunits(t *testing.T) {
	var input float32 = 1234.56
	got := ConvertToMilliunits(input)
	var expect float64 = 1234560
	if float64(got) != expect {
		t.Errorf("Milliunits conversion was incorrect. Expected %f got %d", expect, got)
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRootHandler(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	responseRecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(RootHandler)
	handler.ServeHTTP(responseRecorder, request)
	status := responseRecorder.Code
	if status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	expected := `<html><body>Authenticate with <a href="https://simulator-api.db.com/gw/oidc/authorize(.*)">Deutsche Bank</a>`
	if match, _ := regexp.MatchString(expected, responseRecorder.Body.String()); match == false {
		t.Errorf("handler returned unexpected body: got %v want %v",
			responseRecorder.Body.String(), expected)
	}
}

func TestAuthorizedHandler(t *testing.T) {
	// Failure with an empty "code" parameter
	request, err := http.NewRequest("GET", "/authorized", nil)
	if err != nil {
		t.Fatal(err)
	}
	responseRecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(AuthorizedHandler)
	handler.ServeHTTP(responseRecorder, request)
	status := responseRecorder.Code
	if status == http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

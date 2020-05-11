package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRootHandler(t *testing.T) {
	responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
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
	responseRecorder := runDummyRequest(t, "GET", "/authorized", AuthorizedHandler)
	status := responseRecorder.Code
	if status == http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func runDummyRequest(t *testing.T, verb string, path string, handlerFunc func(w http.ResponseWriter, r *http.Request)) httptest.ResponseRecorder {
	request, err := http.NewRequest(verb, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	responseRecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(responseRecorder, request)
	return *responseRecorder
}

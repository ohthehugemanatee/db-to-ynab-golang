package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"gopkg.in/h2non/gock.v1"
)

func TestRootHandler(t *testing.T) {
	responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
	assertStatus(t, http.StatusOK, responseRecorder.Code)
	expected := `<html><body>Authenticate with <a href="https://simulator-api.db.com/gw/oidc/authorize(.*)">Deutsche Bank</a>`
	if match, _ := regexp.MatchString(expected, responseRecorder.Body.String()); match == false {
		t.Errorf("handler returned unexpected body: got %v want %v",
			responseRecorder.Body.String(), expected)
	}
}

func TestAuthorizedHandler(t *testing.T) {
	// Failure with an empty "code" parameter
	responseRecorder := runDummyRequest(t, "GET", "/authorized", AuthorizedHandler)
	assertStatus(t, http.StatusInternalServerError, responseRecorder.Code)
	// Pass with a code.
	defer gock.Off()
	gock.New("https://simulator-api.db.com").
		Post("/gw/oidc/token").
		Reply(200).
		JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

	gock.New("https://simulator-api.db.com").
		Post("/gw/oidc/token").
		Reply(200).
		JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

	responseRecorder = runDummyRequest(t, "GET", "/authorized?code=abcdef", AuthorizedHandler)
	assertStatus(t, http.StatusFound, responseRecorder.Code)
}

func assertStatus(t *testing.T, expected int, got int) {
	if got != expected {
		t.Errorf("Got wrong status code: got %v want %v",
			got, expected)
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

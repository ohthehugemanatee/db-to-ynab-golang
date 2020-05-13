package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"golang.org/x/oauth2"
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
	t.Run("Failure with an empty \"code\" parameter", func(t *testing.T) {
		t.Parallel()
		responseRecorder := runDummyRequest(t, "GET", "/authorized", AuthorizedHandler)
		assertStatus(t, http.StatusInternalServerError, responseRecorder.Code)
	})
	t.Run("Pass with a valid code parameter", func(t *testing.T) {
		defer gock.Off()
		gock.New("https://simulator-api.db.com").
			Post("/gw/oidc/token").
			Reply(200).
			JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

		gock.New("https://simulator-api.db.com").
			Post("/gw/oidc/token").
			Reply(200).
			JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

		responseRecorder := runDummyRequest(t, "GET", "/authorized?code=abcdef", AuthorizedHandler)
		assertStatus(t, http.StatusFound, responseRecorder.Code)
	})
}

func TestGetTransactions(t *testing.T) {
	defer gock.Off()
	iban := "test-iban"
	expected := "test-iban"
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0)}
	gock.New("https://simulator-api.db.com").
		Get("/gw/dbapi/banking/transactions/v2/").
		MatchParam("iban", iban).
		MatchParam("bookingDateFrom", "2019-11-01").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(expected)
	result := GetTransactions(iban)
	if result != expected {
		t.Errorf("Got wrong value: got %v want %v",
			result, expected)
	}
}

func TestConvertTransactionToYNAB(t *testing.T) {
	os.Setenv("YNAB_ACCOUNT_ID", "account-id")
	inputString := []byte(`{"originIban":"DE10010000000000006136","amount":-19.05,"paymentReference":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","counterPartyName":"Rossmann","transactionCode":"123","valueDate":"2018-04-23","counterPartyIban":"","paymentIdentification":"212+ZKLE 911/696682-X-ABC","mandateReference":"MX0355443","externalBankTransactionDomainCode":"D001","externalBankTransactionFamilyCode":"CCRD","externalBankTransactionSubFamilyCode":"CWDL","bookingDate":"2019-11-04","id":"_2FMRe0AhzLaZu14Cz-lol2H_DDY4z9yIOJKrDlDjHCSCjlJk4dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM","e2eReference":"E2E - Reference","currencyCode":"EUR","creditorId":"DE0222200004544221"}`)
	var input dbTransaction
	_ = json.Unmarshal(inputString, &input)
	converted := ConvertTransactionToYNAB(input)
	marshalledOutput, err := json.Marshal((converted))
	output := string(marshalledOutput)
	if err != nil {
		log.Fatal(err)
	}
	expected := string(`{"account_id":"account-id","date":"2019-11-04T00:00:00Z","amount":-19050,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Rossmann","category_id":null,"memo":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","flag_color":null,"import_id":"4b57e244083bddaef7036b3f7d55c7cb"}`)
	if output != expected {
		t.Errorf("Got wrong value: got %v wanted %v", output, expected)
	}
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

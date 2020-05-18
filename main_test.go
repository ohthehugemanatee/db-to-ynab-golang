package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/h2non/gock.v1"
)

func TestRootHandler(t *testing.T) {
	responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
	assertStatus(t, http.StatusFound, responseRecorder.Code)
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

func TestGetCashTransactions(t *testing.T) {
	defer gock.Off()
	iban := "test-iban"
	expected := "test-iban"
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0)}
	gock.New("https://simulator-api.db.com").
		Get("/gw/dbapi/banking/transactions/v2/").
		MatchParam("iban", iban).
		MatchParam("limit", "100").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(expected)
	result := GetCashTransactions(iban)
	if result != expected {
		t.Errorf("Got wrong value: got %v want %v",
			result, expected)
	}
}

func TestConvertTransactionsToYNAB(t *testing.T) {
	os.Setenv("YNAB_ACCOUNT_ID", "account-id")
	inputString := string(`{"transactions":[{"originIban":"DE10010000000000006136","amount":-19.05,"paymentReference":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","counterPartyName":"Rossmann","transactionCode":"123","valueDate":"2018-04-23","counterPartyIban":"","paymentIdentification":"212+ZKLE 911/696682-X-ABC","mandateReference":"MX0355443","externalBankTransactionDomainCode":"D001","externalBankTransactionFamilyCode":"CCRD","externalBankTransactionSubFamilyCode":"CWDL","bookingDate":"2019-11-04","id":"_2FMRe0AhzLaZu14Cz-lol2H_DDY4z9yIOJKrDlDjHCSCjlJk4dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM","e2eReference":"E2E - Reference","currencyCode":"EUR","creditorId":"DE0222200004544221"}]}`)
	converted := ConvertTransactionsToYNAB(inputString)
	marshalledOutput, err := json.Marshal((converted))
	output := string(marshalledOutput)
	if err != nil {
		log.Fatal(err)
	}
	expected := string(`[{"account_id":"account-id","date":"2019-11-04","amount":-19050,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Rossmann","category_id":null,"memo":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","flag_color":null,"import_id":"4b57e244083bddaef7036b3f7d55c7cb"}]`)
	if output != expected {
		t.Errorf("Got wrong value: got %v wanted %v", output, expected)
	}
}

func TestIsCredit(t *testing.T) {
	cardNumbers := map[string]bool{
		"1234":  true,
		"11f1":  false,
		"123":   false,
		"12345": false,
	}
	for num, expected := range cardNumbers {
		got := IsCredit(num)
		if got != expected {
			t.Errorf("Card %v returned %v, should have been %v", num, got, expected)
		}
	}

}

func TestGetCreditTransactions(t *testing.T) {
	testSuccessfulGetCreditTransactions(t)
	testFailingGetCreditTransactions(t)
}

func testSuccessfulGetCreditTransactions(t *testing.T) {
	defer gock.Off()
	last4 := "1599"
	technicalID := "24842"
	expected := "test-result"
	cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	gock.New("https://simulator-api.db.com").
		Get("gw/dbapi/banking/creditCards/v1/").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardListResponse)

	gock.New("https://simulator-api.db.com").
		Get("gw/dbapi/banking/creditCardTransactions/v1").
		MatchParam("technicalId", technicalID).
		MatchParam("bookingDateTo", time.Now().Format("2006-01-02")).
		MatchParam("bookingDateFrom", time.Now().AddDate(0, 0, -10).Format("2006-01-02")).
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(expected)
	result, _ := GetCreditTransactions(last4)
	if result != expected {
		t.Errorf("Got wrong value: got %v want %v",
			result, expected)
	}
}

func testFailingGetCreditTransactions(t *testing.T) {
	defer gock.Off()
	last4 := "1598"
	cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	gock.New("https://simulator-api.db.com").
		Get("gw/dbapi/banking/creditCards/v1/").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardListResponse)

	_, err := GetCreditTransactions(last4)
	if err == nil {
		t.Error("Did not error out on invalid credit card number")
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

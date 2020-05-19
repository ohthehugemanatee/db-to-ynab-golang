package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/h2non/gock.v1"
)

func TestRootHandler(t *testing.T) {
	t.Run("Test redirect to authorize", func(t *testing.T) {
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		assertStatus(t, http.StatusFound, responseRecorder.Code)
	})
	t.Run("Test with invalid account", func(t *testing.T) {
		currentToken.RefreshToken = "refresh-token"
		accountNumber = "DE10 0100 0000 0000 0061 36"
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		assertStatus(t, http.StatusOK, responseRecorder.Code)
		bodyBytes, _ := ioutil.ReadAll(responseRecorder.Body)
		bodyString := string(bodyBytes)
		expected := "Account ID is not recognized as a valid IBAN or the last 4 digits of a credit card"
		if bodyString != expected {
			t.Errorf("Got wrong value: got %s want %s", bodyString, expected)
		}
	})
	t.Run("Test with valid cash account", func(t *testing.T) {
		defer gock.Off()
		responseBody := `{"totalItems":0,"limit":0,"offset":0,"transactions":[{"id":"string","originIban":"string","amount":0,"counterPartyName":"string","counterPartyIban":"string","paymentReference":"string","bookingDate":"string","currencyCode":"string","transactionCode":"string","externalBankTransactionDomainCode":"string","externalBankTransactionFamilyCode":"string","externalBankTransactionSubFamilyCode":"string","mandateReference":"string","creditorId":"string","e2eReference":"string","paymentIdentification":"string","valueDate":"string"}]}`
		currentToken = &oauth2.Token{
			AccessToken: "ACCESS_TOKEN",
			Expiry:      time.Now().AddDate(1, 0, 0)}
		gock.New("https://simulator-api.Db.com").
			Get("/gw/Dbapi/banking/transactions/v2/").
			MatchParam("iban", accountNumber).
			MatchParam("limit", "100").
			MatchHeader("Authorization", "^Bearer (.*)$").
			Reply(200).
			BodyString(responseBody)
		currentToken.RefreshToken = "refresh-token"
		accountNumber = "DE49 5001 0517 8844 2899 51"
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		assertStatus(t, http.StatusOK, responseRecorder.Code)
	})
	t.Run("Test with valid credit account", func(t *testing.T) {
		defer gock.Off()
		technicalID := "24842"
		cardTransactionResponse := `{"totalItems":1,"items":[{"bookingDate":"2017-09-02","valueDate":"2017-09-02","billingDate":"2017-09-28","reasonForPayment":"Marvel Comics Inc.","amountInForeignCurrency":{"amount":42.21,"currency":"EUR"},"amountInAccountCurrency":{"amount":42.21,"currency":"EUR"},"foreignFxRate":{"sourceCurrency":"EUR","targetCurrency":"EUR","rate":1}}]}`
		cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
		currentToken = &oauth2.Token{
			AccessToken: "ACCESS_TOKEN",
			Expiry:      time.Now().AddDate(1, 0, 0),
		}
		gock.New("https://simulator-api.Db.com").
			Get("gw/Dbapi/banking/creditCards/v1/").
			MatchHeader("Authorization", "^Bearer (.*)$").
			Reply(200).
			BodyString(cardListResponse)

		gock.New("https://simulator-api.Db.com").
			Get("gw/Dbapi/banking/creditCardTransactions/v1").
			MatchParam("technicalId", technicalID).
			MatchParam("bookingDateTo", time.Now().Format("2006-01-02")).
			MatchParam("bookingDateFrom", time.Now().AddDate(0, 0, -10).Format("2006-01-02")).
			MatchHeader("Authorization", "^Bearer (.*)$").
			Reply(200).
			BodyString(cardTransactionResponse)
		currentToken.RefreshToken = "refresh-token"
		accountNumber = "1599"
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		assertStatus(t, http.StatusOK, responseRecorder.Code)
	})
}

func TestAuthorizedHandler(t *testing.T) {
	t.Run("Failure with an empty \"code\" parameter", func(t *testing.T) {
		t.Parallel()
		responseRecorder := runDummyRequest(t, "GET", "/authorized", AuthorizedHandler)
		assertStatus(t, http.StatusInternalServerError, responseRecorder.Code)
	})
	t.Run("Pass with a valid code parameter", func(t *testing.T) {
		defer gock.Off()
		gock.New("https://simulator-api.Db.com").
			Post("/gw/oidc/token").
			Reply(200).
			JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

		gock.New("https://simulator-api.Db.com").
			Post("/gw/oidc/token").
			Reply(200).
			JSON(map[string]string{"access_token": "ACCESS_TOKEN", "token_type": "bearer"})

		responseRecorder := runDummyRequest(t, "GET", "/authorized?code=abcdef", AuthorizedHandler)
		assertStatus(t, http.StatusFound, responseRecorder.Code)
	})
}

func TestGetCashTransactions(t *testing.T) {
	t.Run("Test parsing transactions response from DB", func(t *testing.T) {

		defer gock.Off()
		iban := "test-iban"
		responseBody := `{"totalItems":0,"limit":0,"offset":0,"transactions":[{"id":"string","originIban":"string","amount":0,"counterPartyName":"string","counterPartyIban":"string","paymentReference":"string","bookingDate":"string","currencyCode":"string","transactionCode":"string","externalBankTransactionDomainCode":"string","externalBankTransactionFamilyCode":"string","externalBankTransactionSubFamilyCode":"string","mandateReference":"string","creditorId":"string","e2eReference":"string","paymentIdentification":"string","valueDate":"string"}]}`
		currentToken = &oauth2.Token{
			AccessToken: "ACCESS_TOKEN",
			Expiry:      time.Now().AddDate(1, 0, 0)}
		gock.New("https://simulator-api.Db.com").
			Get("/gw/Dbapi/banking/transactions/v2/").
			MatchParam("iban", iban).
			MatchParam("limit", "100").
			MatchHeader("Authorization", "^Bearer (.*)$").
			Reply(200).
			BodyString(responseBody)
		result := GetCashTransactions(iban)
		marshalledResult, _ := json.Marshal(result)
		stringResult := string(marshalledResult[:])
		expected := `{"Transactions":[{"BookingDate":"string","CounterPartyName":"string","PaymentReference":"string","ID":"string","Amount":0}]}`
		if stringResult != expected {
			t.Errorf("Got wrong value: got %s want %s",
				stringResult, expected)
		}
	})
}

func TestConvertCashTransactionsToYNAB(t *testing.T) {
	t.Run("Test converting cash transactions to ynab format", func(t *testing.T) {
		ynabAccountID = "account-id"
		input := []byte(`{"transactions":[{"originIban":"DE10010000000000006136","amount":-19.05,"paymentReference":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","counterPartyName":"Rossmann","transactionCode":"123","valueDate":"2018-04-23","counterPartyIban":"","paymentIdentification":"212+ZKLE 911/696682-X-ABC","mandateReference":"MX0355443","externalBankTransactionDomainCode":"D001","externalBankTransactionFamilyCode":"CCRD","externalBankTransactionSubFamilyCode":"CWDL","bookingDate":"2019-11-04","id":"_2FMRe0AhzLaZu14Cz-lol2H_DDY4z9yIOJKrDlDjHCSCjlJk4dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM","e2eReference":"E2E - Reference","currencyCode":"EUR","creditorId":"DE0222200004544221"}]}`)
		var DbTransactionsList DbCashTransactionsList
		json.Unmarshal(input, &DbTransactionsList)
		converted := ConvertCashTransactionsToYNAB(DbTransactionsList)
		marshalledOutput, err := json.Marshal((converted))
		output := string(marshalledOutput)
		if err != nil {
			log.Fatal(err)
		}
		expected := string(`[{"account_id":"account-id","date":"2019-11-04","amount":-19050,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Rossmann","category_id":null,"memo":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","flag_color":null,"import_id":"4b57e244083bddaef7036b3f7d55c7cb"}]`)
		if output != expected {
			t.Errorf("Got wrong value: got %s wanted %s", output, expected)
		}
	})
}

func TestConvertCreditTransactionsToYNAB(t *testing.T) {
	t.Run("Test converting credit transactions to ynab format", func(t *testing.T) {
		ynabAccountID = "account-id"
		input := []byte(`{"totalItems":1,"items":[{"bookingDate":"2017-09-02","valueDate":"2017-09-02","billingDate":"2017-09-28","reasonForPayment":"Marvel Comics Inc.","amountInForeignCurrency":{"amount":42.21,"currency":"EUR"},"amountInAccountCurrency":{"amount":42.21,"currency":"EUR"},"foreignFxRate":{"sourceCurrency":"EUR","targetCurrency":"EUR","rate":1}}]}`)
		var DbTransactionsList DbCreditTransactionsList
		json.Unmarshal(input, &DbTransactionsList)
		converted := ConvertCreditTransactionsToYNAB(DbTransactionsList)
		marshalledOutput, err := json.Marshal((converted))
		output := string(marshalledOutput)
		if err != nil {
			log.Fatal(err)
		}
		expected := string(`[{"account_id":"account-id","date":"2017-09-02","amount":42210,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Marvel Comics Inc.","category_id":null,"memo":null,"flag_color":null,"import_id":"5881cb1c0abd80695891732f39924704"}]`)
		if output != expected {
			t.Errorf("Got wrong value: got %s wanted %s", output, expected)
		}
	})
}

func TestIsCredit(t *testing.T) {
	t.Run("Test rejecting malformed card numbers", func(t *testing.T) {
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
	})
}

func TestGetAccountType(t *testing.T) {
	t.Run("Detect valid IBAN", func(t *testing.T) {
		testValidAccount(t, "DE49500105178844289951", Cash)
	})
	t.Run("Detect valid last 4 digits from a credit card", func(t *testing.T) {
		testValidAccount(t, "1234", Credit)
	})
	t.Run("Detect invalid IBAN", func(t *testing.T) {
		iban := "DE10010000000000006136"
		result, err := GetAccountType(iban)
		if result != "" {
			t.Errorf("IBAN %v not detected as invalid", iban)
		}
		if err == nil {
			t.Error("Invalid IBAN did not return an error")
		}
	})
}

func testValidAccount(t *testing.T, account string, expect AccountType) {
	result, err := GetAccountType(account)
	if result != expect {
		t.Errorf("Account type for %v not detected as %v", result, expect)
	}
	if err != nil {
		t.Errorf("Inappropriate error %v returned for valid iban %v", err.Error(), account)
	}
}

func TestGetCreditTransactions(t *testing.T) {
	t.Run("Test with valid data", func(t *testing.T) {
		testSuccessfulGetCreditTransactions(t)
	})
	t.Run("Test with invalid data", func(t *testing.T) {
		testFailingGetCreditTransactions(t)
	})
}

func testSuccessfulGetCreditTransactions(t *testing.T) {
	defer gock.Off()
	last4 := "1599"
	technicalID := "24842"
	cardTransactionResponse := `{"totalItems":1,"items":[{"bookingDate":"2017-09-02","valueDate":"2017-09-02","billingDate":"2017-09-28","reasonForPayment":"Marvel Comics Inc.","amountInForeignCurrency":{"amount":42.21,"currency":"EUR"},"amountInAccountCurrency":{"amount":42.21,"currency":"EUR"},"foreignFxRate":{"sourceCurrency":"EUR","targetCurrency":"EUR","rate":1}}]}`
	cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	gock.New("https://simulator-api.Db.com").
		Get("gw/Dbapi/banking/creditCards/v1/").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardListResponse)

	gock.New("https://simulator-api.Db.com").
		Get("gw/Dbapi/banking/creditCardTransactions/v1").
		MatchParam("technicalId", technicalID).
		MatchParam("bookingDateTo", time.Now().Format("2006-01-02")).
		MatchParam("bookingDateFrom", time.Now().AddDate(0, 0, -10).Format("2006-01-02")).
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardTransactionResponse)
	result, _ := GetCreditTransactions(last4)
	marshalledResult, _ := json.Marshal(result)
	stringResult := string(marshalledResult[:])
	expected := `{"Items":[{"BookingDate":"2017-09-02","ReasonForPayment":"Marvel Comics Inc.","AmountInAccountCurrency":{"Amount":42.21}}]}`
	if stringResult != expected {
		t.Errorf("Got wrong value: got %s want %s",
			stringResult, expected)
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
	gock.New("https://simulator-api.Db.com").
		Get("gw/Dbapi/banking/creditCards/v1/").
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

package dbapi

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/h2non/gock.v1"
)

const (
	cardTransactionResponse string = `{"totalItems":1,"items":[{"bookingDate":"2017-09-02","valueDate":"2017-09-02","billingDate":"2017-09-28","reasonForPayment":"Marvel Comics Inc.","amountInForeignCurrency":{"amount":42.21,"currency":"EUR"},"amountInAccountCurrency":{"amount":42.21,"currency":"EUR"},"foreignFxRate":{"sourceCurrency":"EUR","targetCurrency":"EUR","rate":1}}]}`
	cardListResponse        string = `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
)

func TestIsValidAccount(t *testing.T) {
	connector := DbCreditConnector{}
	t.Run("Test validating card numbers", func(t *testing.T) {
		cardNumbers := map[string]bool{
			"1234":  true,
			"11f1":  false,
			"123":   false,
			"12345": false,
		}
		for num, expected := range cardNumbers {
			got, _ := connector.IsValidAccountNumber(num)
			if got != expected {
				t.Errorf("Card %v returned %v, should have been %v", num, got, expected)
			}
		}
	})
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
	connector := DbCreditConnector{}
	last4 := "1599"
	technicalID := "24842"
	cardTransactionResponse := `{"totalItems":1,"items":[{"bookingDate":"2017-09-02","valueDate":"2017-09-02","billingDate":"2017-09-28","reasonForPayment":"Marvel Comics Inc.","amountInForeignCurrency":{"amount":42.21,"currency":"EUR"},"amountInAccountCurrency":{"amount":42.21,"currency":"EUR"},"foreignFxRate":{"sourceCurrency":"EUR","targetCurrency":"EUR","rate":1}}]}`
	cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	gock.New(dbAPIBaseURL).
		Get("gw/Dbapi/banking/creditCards/v1/").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardListResponse)

	gock.New(dbAPIBaseURL).
		Get("gw/Dbapi/banking/creditCardTransactions/v1").
		MatchParam("technicalId", technicalID).
		MatchParam("bookingDateTo", time.Now().Format("2006-01-02")).
		MatchParam("bookingDateFrom", time.Now().AddDate(0, 0, -10).Format("2006-01-02")).
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardTransactionResponse)
	result, _ := connector.GetCreditTransactions(last4)
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
	connector := DbCreditConnector{}
	last4 := "1598"
	cardListResponse := `{  "totalItems": 1,  "items": [    {      "technicalId": "24842",      "embossedLine1": "DR HANS LUEDENSCHE",      "hasDebitFeatures": false,      "expiryDate": "10.2018",      "productName": "Deutsche Bank BusinessCard",      "securePAN": "************1599"    }  ]}`
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	gock.New(dbAPIBaseURL).
		Get("gw/Dbapi/banking/creditCards/v1/").
		MatchHeader("Authorization", "^Bearer (.*)$").
		Reply(200).
		BodyString(cardListResponse)

	_, err := connector.GetCreditTransactions(last4)
	if err == nil {
		t.Error("Did not error out on invalid credit card number")
	}
}

func TestConvertCreditTransactionsToYNAB(t *testing.T) {
	t.Run("Test converting credit transactions to ynab format", func(t *testing.T) {
		connector := DbCreditConnector{}
		ynabAccountID = "account-id"
		input := []byte(cardTransactionResponse)
		var DbTransactionsList DbCreditTransactionsList
		json.Unmarshal(input, &DbTransactionsList)
		converted := connector.ConvertCreditTransactionsToYNAB(DbTransactionsList)
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

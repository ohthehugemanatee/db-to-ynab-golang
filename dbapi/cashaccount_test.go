package dbapi

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/h2non/gock.v1"
)

// Handy test values.
const (
	goodIban                 string = "DE49500105178844289951"
	badIban                  string = "DE10010000000111106136"
	testIban                 string = "DE10010000000000006136"
	cashTransactionsResponse string = `{"transactions":[{"originIban":"DE10010000000000006136","amount":-19.05,"paymentReference":"POS MIT PIN. Mein Drogeriemarkt, Leipziger Str.","counterPartyName":"Rossmann","transactionCode":"123","valueDate":"2018-04-23","counterPartyIban":"","paymentIdentification":"212+ZKLE 911/696682-X-ABC","mandateReference":"MX0355443","externalBankTransactionDomainCode":"D001","externalBankTransactionFamilyCode":"CCRD","externalBankTransactionSubFamilyCode":"CWDL","bookingDate":"2019-11-04","id":"_2FMRe0AhzLaZu14Cz-lol2H_DDY4z9yIOJKrDlDjHCSCjlJk4dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM","e2eReference":"E2E - Reference","currencyCode":"EUR","creditorId":"DE0222200004544221"},{"originIban":"DE10010000000000006136","amount":-22.50,"paymentReference":"POS MIT PIN. Lebensmittelhandel, Kölner Str.","counterPartyName":"Rewe","transactionCode":"123","valueDate":"2019-11-05","counterPartyIban":"","paymentIdentification":"12345678","mandateReference":"MX0355443","externalBankTransactionDomainCode":"D001","externalBankTransactionFamilyCode":"CCRD","externalBankTransactionSubFamilyCode":"CWDL","bookingDate":"2019-11-05","id":"_2FMRelmnop13z-lol2H_DDY4z9yIOJKrlmnop12345677894dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM","e2eReference":"E2E Reference","currencyCode":"EUR","creditorId":"DE0111100004544221"}]}`
	ynabAPIBaseURL           string = "https://api.youneedabudget.com/"
)

// Set the SuT.
var connector = DbCashConnector{}

func TestGetTransactions(t *testing.T) {

	// Set a dummy valid token.
	currentToken = &oauth2.Token{
		AccessToken: "ACCESS_TOKEN",
		Expiry:      time.Now().AddDate(1, 0, 0),
	}
	// Set a dummy YNAB Budget ID.
	ynabAccountID = "account-id"
	dbAPIBaseURL = "https://example.com/"
	t.Run("Test parsing transactions response from DB", func(t *testing.T) {
		defer gock.Off()
		gock.New(dbAPIBaseURL).
			Get("/gw/dbapi/banking/transactions/v2/").
			MatchParam("iban", goodIban).
			MatchParam("limit", "100").
			MatchHeader("Authorization", "^Bearer (.*)$").
			Reply(200).
			BodyString(cashTransactionsResponse)
		result, _ := connector.GetTransactions(goodIban)
		marshalledResult, _ := json.Marshal(result)
		stringResult := string(marshalledResult[:])
		expected := `[{"account_id":"account-id","date":"2019-11-05","amount":-22500,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Rewe","category_id":null,"memo":"POS MIT PIN. Lebensmittelhandel, Kölner Str.","flag_color":null,"import_id":"9a78f21363fe716814a0875ea75fa662"},{"account_id":"account-id","date":"2019-11-05","amount":-22500,"cleared":"cleared","approved":false,"payee_id":null,"payee_name":"Rewe","category_id":null,"memo":"POS MIT PIN. Lebensmittelhandel, Kölner Str.","flag_color":null,"import_id":"9a78f21363fe716814a0875ea75fa662"}]`
		if stringResult != expected {
			t.Errorf("Got wrong value: got %s want %s",
				stringResult, expected)
		}
	})
}

func TestConvertCashTransactionsToYNAB(t *testing.T) {
	t.Run("Test converting cash transactions to ynab format", func(t *testing.T) {
		ynabAccountID = "account-id"
		dbAPIBaseURL = "https://example.com/"
		input := []byte(cashTransactionsResponse)
		var DbTransactionsList DbCashTransactionsList
		json.Unmarshal(input, &DbTransactionsList)
		converted := connector.ConvertCashTransactionsToYNAB(DbTransactionsList, ynabAccountID)
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

func TestIsValidAccountNumber(t *testing.T) {
	t.Run("Detect valid IBAN", func(t *testing.T) {
		result, err := connector.IsValidAccountNumber(goodIban)
		if err != nil {
			t.Errorf("Inappropriate error %v returned for valid iban %v", err.Error(), goodIban)
		}
		if result != true {
			t.Errorf("Incorrectly reported %v as an invalid IBAN", goodIban)
		}
	})
	t.Run("Detect testing IBAN", func(t *testing.T) {
		result, err := connector.IsValidAccountNumber(testIban)
		if err != nil {
			t.Errorf("Inappropriate error %v returned for testing iban %v", err.Error(), goodIban)
		}
		if result != true {
			t.Errorf("Incorrectly reported %v as an invalid IBAN", goodIban)
		}
	})
	t.Run("Detect invalid IBAN", func(t *testing.T) {
		result, err := connector.IsValidAccountNumber(badIban)
		if result != false {
			t.Errorf("IBAN %v not detected as invalid", badIban)
		}
		if err != nil {
			t.Error("Invalid IBAN returned an error")
		}
	})
}

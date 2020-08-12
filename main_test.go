package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ohthehugemanatee/db-to-ynab-golang/dbapi"
	"github.com/ohthehugemanatee/db-to-ynab-golang/tools"
	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
	"gopkg.in/h2non/gock.v1"
)

const (
	goodIban           string = "DE49500105178844289951"
	badIban            string = "DE10010000000111106136"
	dummyYnabAccountID string = "f2b9e2c0-f927-2aa3-f2cf-f227d22fa7f9"
	dummyYnabBudgetID  string = "b25f2ff7-5fba-f332-f4a2-24f32f02f857"
	dummyYnabSecret    string = "bb97fbf01ebbfbd73fff33bfdcbf7bf30fbfb7f9b5dfea5c5ffb04bb52eb366b"
)

var AuthorizedHandlerWasHit bool

var getTransactionsResponse []ynabTransaction = []ynabTransaction{}

type testConnector struct {
	AuthorizationURL string
}

func (c testConnector) CheckParams() (bool, error) {
	return true, nil
}

func (c testConnector) IsValidAccountNumber(a string) (bool, error) {
	return true, nil
}

func (c testConnector) GetTransactions(string) ([]ynabTransaction, error) {
	return getTransactionsResponse, nil
}
func (c testConnector) Authorize() string {
	return c.AuthorizationURL
}

func (c testConnector) AuthorizedHandler(http.ResponseWriter, *http.Request) {
	AuthorizedHandlerWasHit = true
}

func TestElectAndConfigureConnector(t *testing.T) {
	activeConnector = nil
	availableConnectors = []BankConnector{
		testConnector{"https://example.com/"},
	}
	logBuffer := tools.CreateAndActivateEmptyTestLogBuffer()
	// Redefine Fatal Error so we can catch/test it.
	originalFatalError := fatalError
	defer func() { fatalError = originalFatalError }()
	fatalError = func(v ...interface{}) {
		log.Print(v)
	}
	t.Run("Test connector election", func(t *testing.T) {
		logBuffer.ExpectLog("Connector main.testConnector elected")
		electConnectorOrFatal()
		if activeConnector == nil {
			t.Error("Connector was not elected")
		}
		logBuffer.TestLogValues(t)
	})
	t.Run("Test failure in connector election", func(t *testing.T) {
		logBuffer = tools.CreateAndActivateEmptyTestLogBuffer()
		logBuffer.ExpectLog("[Account number is not recognized by any connector, cannot proceed without a compatible connector]")
		availableConnectors = []BankConnector{}
		electConnectorOrFatal()
		logBuffer.TestLogValues(t)
	})
	t.Run("Test failure in connector configuration", func(t *testing.T) {
		// @todo
		// - make a failure connector that can be elected but not configured.
		// - use it to generate an error.
	})
}
func TestRootHandler(t *testing.T) {
	setDummyConnector(true)
	t.Run("Test redirect to authorize url", func(t *testing.T) {
		expectedURL := "https://example.com/"
		activeConnector = testConnector{
			AuthorizationURL: expectedURL,
		}
		testLogBuffer := tools.CreateAndActivateEmptyTestLogBuffer()
		testLogBuffer.ExpectLog("Received HTTP request to /")
		testLogBuffer.ExpectLog("We are not yet authorized, redirecting to https://example.com/")
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		testLogBuffer.TestLogValues(t)
		tools.AssertStatus(t, http.StatusFound, responseRecorder.Code)
		got, _ := responseRecorder.Result().Location()
		if got.String() != expectedURL {
			t.Errorf("Got wrong redirect URL. Got %s want %s", got, expectedURL)
		}
	})
	t.Run("Test with valid account", func(t *testing.T) {
		setDummyConnector(true)
		setDummyTransactionResponse()
		setDummyYnabData()
		defer gock.Off()
		gock.New("https://api.youneedabudget.com/").
			Post("/v1/budgets/"+dummyYnabBudgetID+"/transactions").
			MatchHeader("Authorization", "Bearer "+dummyYnabSecret).
			Reply(200).
			AddHeader("X-Rate-Limit", "36/200").
			BodyString(`{"data":{"transaction_ids":["string"],"transaction":{"id":"string","date":"2006-01-02","amount":0,"memo":"string","cleared":"cleared","approved":true,"flag_color":"red","account_id":"string","payee_id":"string","category_id":"string","transfer_account_id":"string","transfer_transaction_id":"string","matched_transaction_id":"string","import_id":"string","deleted":true,"account_name":"string","payee_name":"string","category_name":"string","subtransactions":[{"id":"string","transaction_id":"string","amount":0,"memo":"string","payee_id":"string","payee_name":"string","category_id":"string","category_name":"string","transfer_account_id":"string","transfer_transaction_id":"string","deleted":true}]},"transactions":[{"id":"string","date":"2006-01-02","amount":0,"memo":"string","cleared":"cleared","approved":true,"flag_color":"red","account_id":"string","payee_id":"string","category_id":"string","transfer_account_id":"string","transfer_transaction_id":"string","matched_transaction_id":"string","import_id":"string","deleted":true,"account_name":"string","payee_name":"string","category_name":"string","subtransactions":[{"id":"string","transaction_id":"string","amount":0,"memo":"string","payee_id":"string","payee_name":"string","category_id":"string","category_name":"string","transfer_account_id":"string","transfer_transaction_id":"string","deleted":true}]}],"duplicate_import_ids":["string"],"server_knowledge":0}}`)

		connector := testConnector{}
		connector.GetTransactions(goodIban)
		testLogBuffer := tools.CreateAndActivateEmptyTestLogBuffer()
		testLogBuffer.ExpectLog("Received HTTP request to /")
		testLogBuffer.ExpectLog("Received 1 transactions from bank\nPosting transactions to YNAB")
		testLogBuffer.ExpectLog("Posted transactions to YNAB, 1 new, 1 duplicate, 1 saved. Ending run")
		responseRecorder := runDummyRequest(t, "GET", "/", RootHandler)
		tools.AssertStatus(t, http.StatusOK, responseRecorder.Code)
		testLogBuffer.TestLogValues(t)
	})
}

func TestAuthorizedHandler(t *testing.T) {
	t.Run("Hitting the authorization endpoint should hit the authorization handler", func(t *testing.T) {
		setDummyConnector(true)
		responseRecorder := runDummyRequest(t, "GET", "/authorized", activeConnector.AuthorizedHandler)
		tools.AssertStatus(t, http.StatusOK, responseRecorder.Code)
		if AuthorizedHandlerWasHit != true {
			t.Error("Oauth authorization handler was not hit")
		}

	})
}

func TestGetConnector(t *testing.T) {
	setRealConnectors(true)
	t.Run("Detect valid IBAN", func(t *testing.T) {
		assertGetsConnector(t, goodIban, dbapi.DbCashConnector{})
	})
	t.Run("Detect valid last 4 digits from a credit card", func(t *testing.T) {
		assertGetsConnector(t, "1234", dbapi.DbCreditConnector{})
	})
	t.Run("Detect invalid IBAN", func(t *testing.T) {
		result, err := GetConnector(badIban)
		if result != nil {
			t.Errorf("IBAN %v not detected as invalid", badIban)
		}
		if err.Error() != "Account number is not recognized by any connector, cannot proceed without a compatible connector" {
			t.Error("Invalid IBAN did not return desired error")
		}
	})
}

func assertGetsConnector(t *testing.T, accountID string, expect BankConnector) {
	expectString := reflect.TypeOf(expect).String()
	result, err := GetConnector(accountID)
	if err != nil {
		t.Errorf("Unexpected error %v returned for account ID %s", err.Error(), accountID)
	}
	resultString := reflect.TypeOf(result).String()
	if resultString != expectString {
		t.Errorf("Account type for %s detected as %s, expected %s", accountID, resultString, expectString)
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

func setDummyConnector(setActiveConnector bool) {
	availableConnectors = []BankConnector{
		testConnector{},
	}
	if setActiveConnector {
		activeConnector = testConnector{}
		return
	}
	activeConnector = nil
}

func setRealConnectors(unsetActiveConnector bool) {
	availableConnectors = []BankConnector{
		dbapi.DbCashConnector{},
		dbapi.DbCreditConnector{},
	}
	if unsetActiveConnector {
		activeConnector = nil
	}
}

func setDummyTransactionResponse() {
	date, _ := api.DateFromString("2020-05-05")
	payeeName := string("payee-name")
	importID := string("import-id")
	getTransactionsResponse = []ynabTransaction{
		{
			AccountID: dummyYnabAccountID,
			Date:      date,
			Amount:    10000,
			Cleared:   transaction.ClearingStatusCleared,
			Approved:  true,
			PayeeName: &payeeName,
			ImportID:  &importID,
		},
	}
}

func setDummyYnabData() {
	ynabAccountID = dummyYnabAccountID
	ynabBudgetID = dummyYnabBudgetID
	ynabSecret = dummyYnabSecret
}

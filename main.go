package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"go.bmvs.io/ynab"
	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
	"golang.org/x/oauth2"
)

type YNABTransaction = transaction.PayloadTransaction

type dbTransaction struct {
	BookingDate      string
	CounterPartyName string
	PaymentReference string
	ID               string
	Amount           float32
}

type dbTransactionsList struct {
	Transactions []dbTransaction
}

var oauth2Conf = &oauth2.Config{
	ClientID:     os.Getenv("DB_CLIENT_ID"),
	ClientSecret: os.Getenv("DB_CLIENT_SECRET"),
	Scopes:       []string{"read_transactions", "read_accounts", "read_credit_cards_list_with_details", "read_credit_card_transactions", "offline_access"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://simulator-api.db.com/gw/oidc/authorize",
		TokenURL: "https://simulator-api.db.com/gw/oidc/token",
	},
}
var oauth2HttpContext context.Context = context.Background()

var currentToken = &oauth2.Token{}

func main() {
	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/authorized", AuthorizedHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
	fmt.Printf("Listening for input")
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>")
	if currentToken.Valid() {
		fmt.Fprintf(w, "Already authorized")
		iban := os.Getenv("DB_IBAN")
		dbTransactions := GetTransactions(iban)
		convertedTransactions := convertTransactionsToYNAB(dbTransactions)
		PostTransactionsToYNAB(os.Getenv("YNAB_SECRET"), os.Getenv("YNAB_BUDGET_ID"), convertedTransactions)
		fmt.Fprint(w, "Posted transactions to YNAB")
		return
	}
	url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

// AuthorizedHandler handles calls to /authorized
func AuthorizedHandler(w http.ResponseWriter, r *http.Request) {
	var code string = r.URL.Query().Get("code")

	if code == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Deutsche Bank returned an empty code."))
		return
	}
	tok, err := oauth2Conf.Exchange(oauth2HttpContext, code)
	if err != nil {
		log.Fatal(err)
	}
	currentToken = tok
	http.Redirect(w, r, "/", http.StatusFound)
}

// GetTransactions gets transactions from DB.
func GetTransactions(iban string) string {
	transactions, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get("https://simulator-api.db.com/gw/dbapi/banking/transactions/v2/?bookingDateFrom=2019-11-01&iban=" + iban)
	if err != nil {
		log.Fatal(err)
	}
	defer transactions.Body.Close()
	body, err := ioutil.ReadAll(transactions.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body)
}

func convertTransactionsToYNAB(incomingTransactions string) []YNABTransaction {
	var marshalledTransactions dbTransactionsList
	err := json.Unmarshal([]byte(incomingTransactions), &marshalledTransactions)
	if err != nil {
		log.Fatal(err)
	}
	var convertedTransactions []YNABTransaction
	for _, transaction := range marshalledTransactions.Transactions {
		convertedTransactions = append(convertedTransactions, ConvertTransactionToYNAB(transaction))
	}
	return convertedTransactions
}

// ConvertTransactionToYNAB converts a transaction to YNAB format.
func ConvertTransactionToYNAB(incomingTransaction dbTransaction) YNABTransaction {

	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	sum := sha256.Sum256([]byte(incomingTransaction.ID))
	importID := fmt.Sprintf("%x", sum)
	// Amount must be in YNAB "milliunits".
	amount := int64(incomingTransaction.Amount * 1000)
	transaction := YNABTransaction{
		AccountID: os.Getenv("YNAB_ACCOUNT_ID"),
		Date:      date,
		Amount:    amount,
		PayeeName: &incomingTransaction.CounterPartyName,
		Memo:      &incomingTransaction.PaymentReference,
		Cleared:   transaction.ClearingStatusCleared,
		Approved:  false,
		ImportID:  &importID,
	}
	return transaction
}

// PostTransactionsToYNAB posts transactions to YNAB.
func PostTransactionsToYNAB(accessToken string, budgetID string, transactions []YNABTransaction) {
	c := ynab.NewClient(accessToken)
	_, err := c.Transaction().BulkCreateTransactions(budgetID, transactions)
	if err != nil {
		log.Fatal(err)
	}
}

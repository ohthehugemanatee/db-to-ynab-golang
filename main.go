package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/ohthehugemanatee/db-to-ynab-golang/dbapi"
	"go.bmvs.io/ynab"
	"go.bmvs.io/ynab/api/transaction"
)

type ynabTransaction = transaction.PayloadTransaction

// BankConnector is the interface for any bank account connection.
type BankConnector interface {
	// Checks if all parameters for the connector are valid and present.
	CheckParams() (bool, error)
	// Checks if the account number is valid for this connector.
	IsValidAccountNumber(string) (bool, error)
	// Gets YNAB formatted transactions.
	GetTransactions(string) ([]ynabTransaction, error)
	// Returns an oauth authorization url if necessary.
	Authorize() string
	// Handles an oauth response if necessary
	AuthorizedHandler(http.ResponseWriter, *http.Request)
}

var (
	ynabSecret          string          = os.Getenv("YNAB_SECRET")
	ynabBudgetID        string          = os.Getenv("YNAB_BUDGET_ID")
	ynabAccountID       string          = os.Getenv("YNAB_ACCOUNT_ID")
	accountNumber       string          = os.Getenv("DB_ACCOUNT")
	availableConnectors []BankConnector = []BankConnector{
		dbapi.DbCashConnector{},
		dbapi.DbCreditConnector{},
	}
	activeConnector BankConnector
)

func main() {
	var err error
	activeConnector, err = GetConnector(accountNumber)
	if err != nil {
		log.Fatal(err)
	}
	_, err = activeConnector.CheckParams()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/authorized", activeConnector.AuthorizedHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	if url := activeConnector.Authorize(); url != "" {
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	convertedTransactions, _ := activeConnector.GetTransactions(accountNumber)
	if len(convertedTransactions) == 0 {
		return
	}
	postTransactionsToYNAB(ynabSecret, ynabBudgetID, convertedTransactions)
}

// GetConnector returns the first connector where the account number is valid.
func GetConnector(accountNumber string) (BankConnector, error) {
	for _, connector := range availableConnectors {
		result, err := connector.IsValidAccountNumber(accountNumber)
		if err != nil {
			log.Print(err)
		}
		if result {
			return connector, nil
		}
	}
	return nil, errors.New("Account number is not recognized by any connector, cannot proceed without a compatible connector")
}

// PostTransactionsToYNAB posts transactions to YNAB.
func postTransactionsToYNAB(accessToken string, budgetID string, transactions []ynabTransaction) {
	c := ynab.NewClient(accessToken)
	_, err := c.Transaction().BulkCreateTransactions(budgetID, transactions)
	if err != nil {
		log.Fatal(err)
	}
}

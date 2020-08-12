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

const networkAddress string = ":3000"

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
	electAndConfigureConnector()
	registerHandlers()
	log.Fatal(http.ListenAndServe(networkAddress, nil))
	log.Print("DB/YNAB sync server started, listening on port 3000.")
}

func electAndConfigureConnector() {
	var err error
	activeConnector, err = GetConnector(accountNumber)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connector %T elected", activeConnector)
	_, err = activeConnector.CheckParams()
	if err != nil {
		log.Fatal(err)
	}
}

func registerHandlers() {
	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/authorized", activeConnector.AuthorizedHandler)
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Received HTTP request to /")
	if url := activeConnector.Authorize(); url != "" {
		log.Printf("We are not yet authorized, redirecting to %s", url)
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	convertedTransactions, err := activeConnector.GetTransactions(accountNumber)
	if err != nil {
		log.Printf("Failed to get bank transactions: %s", err)
	}
	transactionsCount := len(convertedTransactions)
	log.Printf("Received %d transactions from bank", transactionsCount)
	if transactionsCount == 0 {
		log.Print("Ending run")
		return
	}
	log.Print("Posting transactions to YNAB")
	createdTransactions, err := postTransactionsToYNAB(ynabSecret, ynabBudgetID, convertedTransactions)
	if err != nil {
		log.Print(err)
		return
	}
	createdCount := len(createdTransactions.TransactionIDs)
	duplicateCount := len(createdTransactions.DuplicateImportIDs)
	savedCount := len(createdTransactions.Transactions)

	log.Printf("Posted transactions to YNAB, %d new, %d duplicate, %d saved. Ending run", createdCount, duplicateCount, savedCount)
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
func postTransactionsToYNAB(accessToken string, budgetID string, transactions []ynabTransaction) (*transaction.CreatedTransactions, error) {
	c := ynab.NewClient(accessToken)
	return c.Transaction().CreateTransactions(budgetID, transactions)
}

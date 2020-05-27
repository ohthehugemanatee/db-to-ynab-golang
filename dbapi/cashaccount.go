package dbapi

import (
	"log"
	"net/http"
	"os"

	"github.com/go-pascal/iban"
	"github.com/ohthehugemanatee/db-to-ynab-golang/tools"
	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
)

type ynabTransaction = transaction.PayloadTransaction

// DbCashTransactionsList is a list of cash transactions as returned by the DB API.
type DbCashTransactionsList struct {
	Transactions []DbCashTransaction
}

// DbCashTransaction represents a transaction from a cash account.
type DbCashTransaction struct {
	BookingDate      string
	CounterPartyName string
	PaymentReference string
	ID               string
	Amount           float32
}

var ynabAccountID string = os.Getenv("YNAB_ACCOUNT_ID")

// DbCashConnector gets transactions from a DB Cash account and converts to YNAB format.
type DbCashConnector struct{}

// IsValidAccountNumber validates that the account number can be processed by this interface.
func (connector DbCashConnector) IsValidAccountNumber(accountNumber string) (bool, error) {
	isCorrectIban, _, _ := iban.IsCorrectIban(accountNumber, false)
	return isCorrectIban, nil
}

// GetTransactions gets transactions from DB and returns them in YNAB format.
func (connector DbCashConnector) GetTransactions(accountNumber string) ([]ynabTransaction, error) {
	var transactions DbCashTransactionsList
	dbAPIRequest("gw/Dbapi/banking/transactions/v2/?limit=100&iban="+accountNumber, &transactions)
	ynabTransactions := connector.ConvertCashTransactionsToYNAB(transactions, ynabAccountID)
	return ynabTransactions, nil
}

// Authorize checks the current token and returns an authorization URL if necessary.
func (connector DbCashConnector) Authorize() string {
	return Authorize()
}

// AuthorizedHandler handles the oauth HTTP response.
func (connector DbCashConnector) AuthorizedHandler(w http.ResponseWriter, r *http.Request) {
	AuthorizedHandler(w, r)
}

// ConvertCashTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func (connector DbCashConnector) ConvertCashTransactionsToYNAB(incomingTransactions DbCashTransactionsList, ynabAccountID string) (convertedTransactions []ynabTransaction) {
	transactions := incomingTransactions.Transactions
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	for _, transaction := range transactions {
		go func(t DbCashTransaction) {
			resultChannel <- connector.ConvertTransactionToYNAB(ynabAccountID, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

// ConvertTransactionToYNAB converts a given transaction to YNAB format.
func (connector DbCashConnector) ConvertTransactionToYNAB(accountNumber string, incomingTransaction DbCashTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importID := tools.CreateImportID(incomingTransaction.ID)
	transaction := ynabTransaction{
		AccountID: accountNumber,
		Date:      date,
		Amount:    tools.ConvertToMilliunits(incomingTransaction.Amount),
		PayeeName: &incomingTransaction.CounterPartyName,
		Memo:      &incomingTransaction.PaymentReference,
		Cleared:   transaction.ClearingStatusCleared,
		Approved:  false,
		ImportID:  &importID,
	}
	return transaction
}

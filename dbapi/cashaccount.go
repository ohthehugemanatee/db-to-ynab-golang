package dbapi

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

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

// CheckParams ensures that all parameters are provided.
func (connector DbCashConnector) CheckParams() (bool, error) {
	return CheckParams()
}

// IsValidAccountNumber validates that the account number can be processed by this interface.
func (connector DbCashConnector) IsValidAccountNumber(accountNumber string) (bool, error) {
	// Detect test account number.
	if accountNumber == "DE10010000000000006136" {
		return true, nil
	}
	isCorrectIban, _, _ := iban.IsCorrectIban(accountNumber, false)
	return isCorrectIban, nil
}

// GetTransactions gets transactions from DB and returns them in YNAB format.
func (connector DbCashConnector) GetTransactions(accountNumber string) ([]ynabTransaction, error) {
	var transactions DbCashTransactionsList
	params := url.Values{}
	params.Add("limit", "100")
	params.Add("bookingDateFrom", time.Now().AddDate(0, 0, -10).Format("2006-01-02"))
	params.Add("sortBy", "bookingDate[DESC]")
	params.Add("iban", accountNumber)
	err := dbAPIRequest("gw/dbapi/banking/transactions/v2/?"+params.Encode(), &transactions)
	if err != nil {
		return nil, err
	}
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
	var convertedTransaction ynabTransaction
	defer close(resultChannel)
	for _, transaction := range transactions {
		go func(t DbCashTransaction) {
			resultChannel <- connector.ConvertTransactionToYNAB(ynabAccountID, t)
		}(transaction)
	}
	for i := 0; i < len(transactions); i++ {
		convertedTransaction = <-resultChannel
		convertedTransactions = append(convertedTransactions, convertedTransaction)
	}
	return convertedTransactions
}

// ConvertTransactionToYNAB converts a given transaction to YNAB format.
func (connector DbCashConnector) ConvertTransactionToYNAB(accountNumber string, incomingTransaction DbCashTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Print(err)
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

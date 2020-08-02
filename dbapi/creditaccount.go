package dbapi

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/ohthehugemanatee/db-to-ynab-golang/tools"
	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
)

// DbCreditTransaction represents a transaction from a credit card account.
type DbCreditTransaction struct {
	BookingDate             string
	ReasonForPayment        string
	AmountInAccountCurrency struct {
		Amount float32
	}
}

// DbCreditTransactionsList is a list of cash transactions as returned by the DB API.
type DbCreditTransactionsList struct {
	Items []DbCreditTransaction
}

// DbCreditCard is a credit card.
type DbCreditCard struct {
	TechnicalID string
	SecurePAN   string
}

// DbCreditCardsList is a list of credit cards as returned by the DB API.
type DbCreditCardsList struct {
	Items []DbCreditCard
}

// DbCreditConnector is the connector for DB Credit card accounts.
type DbCreditConnector struct{}

// CheckParams ensures that all parameters are provided.
func (connector DbCreditConnector) CheckParams() (bool, error) {
	return CheckParams()
}

// IsValidAccountNumber validates that the account number can be processed by this interface.
func (connector DbCreditConnector) IsValidAccountNumber(accountNumber string) (bool, error) {
	re := regexp.MustCompile(`^[0-9]{4}$`)
	return re.MatchString(accountNumber), nil
}

// GetTransactions gets transactions from DB and returns them in YNAB format.
func (connector DbCreditConnector) GetTransactions(accountnumber string) ([]ynabTransaction, error) {
	transactions, _ := connector.GetCreditTransactions(accountnumber)
	return connector.ConvertCreditTransactionsToYNAB(transactions), nil
}

// Authorize checks the current token and returns an authorization URL if necessary.
func (connector DbCreditConnector) Authorize() string {
	return Authorize()
}

// AuthorizedHandler handles the oauth HTTP response.
func (connector DbCreditConnector) AuthorizedHandler(w http.ResponseWriter, r *http.Request) {
	AuthorizedHandler(w, r)
}

// GetCreditTransactions gets transactions from a credit card.
func (connector DbCreditConnector) GetCreditTransactions(last4 string) (transactions DbCreditTransactionsList, err error) {
	technicalID, err := connector.getTechnicalID(last4)
	if err != nil {
		return transactions, err
	}
	urlParams := "?technicalId=" + technicalID
	urlParams = urlParams + "&bookingDateTo=" + time.Now().Format("2006-01-02")
	urlParams = urlParams + "&bookingDateFrom=" + time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	err = dbAPIRequest("gw/Dbapi/banking/creditCardTransactions/v1"+urlParams, &transactions)
	return transactions, err
}

func (connector DbCreditConnector) getTechnicalID(last4 string) (string, error) {
	var cardsMap DbCreditCardsList
	err := dbAPIRequest("gw/Dbapi/banking/creditCards/v1/", &cardsMap)
	if err != nil {
		return "", err
	}
	for _, cardDetails := range cardsMap.Items {
		if cardDetails.SecurePAN == "************"+last4 {
			return cardDetails.TechnicalID, nil
		}
	}
	return "", fmt.Errorf("No credit card found on account with last digits %v", last4)
}

// ConvertCreditTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func (connector DbCreditConnector) ConvertCreditTransactionsToYNAB(incomingTransactions DbCreditTransactionsList) []ynabTransaction {
	transactions := incomingTransactions.Items
	var convertedTransactions []ynabTransaction
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	accountNumber := ynabAccountID
	for _, transaction := range transactions {
		go func(t DbCreditTransaction) {
			resultChannel <- connector.convertCreditTransactionToYNAB(accountNumber, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

func (connector DbCreditConnector) convertCreditTransactionToYNAB(accountNumber string, incomingTransaction DbCreditTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importIDSource := incomingTransaction.BookingDate + fmt.Sprintf("%f", incomingTransaction.AmountInAccountCurrency.Amount)
	importID := tools.CreateImportID(importIDSource)
	transaction := ynabTransaction{
		AccountID: accountNumber,
		Date:      date,
		Amount:    tools.ConvertToMilliunits(incomingTransaction.AmountInAccountCurrency.Amount),
		PayeeName: &incomingTransaction.ReasonForPayment,
		Cleared:   transaction.ClearingStatusCleared,
		Approved:  false,
		ImportID:  &importID,
	}
	return transaction
}

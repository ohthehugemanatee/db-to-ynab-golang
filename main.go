package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-pascal/iban"
	"go.bmvs.io/ynab"
	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
	"golang.org/x/oauth2"
)

type ynabTransaction = transaction.PayloadTransaction

// DbCashTransaction represents a transaction from a cash account.
type DbCashTransaction struct {
	BookingDate      string
	CounterPartyName string
	PaymentReference string
	ID               string
	Amount           float32
}

// DbCreditTransaction represents a transaction from a credit card account.
type DbCreditTransaction struct {
	BookingDate             string
	ReasonForPayment        string
	AmountInAccountCurrency struct {
		Amount float32
	}
}

// DbCashTransactionsList is a list of cash transactions as returned by the DB API.
type DbCashTransactionsList struct {
	Transactions []DbCashTransaction
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

// AccountType denotes a type of bank account, eg cash vs credit.
type AccountType string

const (
	// Cash is for cash accounts, eg checking, savings.
	Cash AccountType = "cash"
	// Credit is for credit card accounts, eg visa or mastercard.
	Credit AccountType = "credit"
	// DbAPIBaseURL is the base url of the DB API.
	DbAPIBaseURL string = "https://simulator-api.Db.com/"
)

var (
	accountNumber  string = os.Getenv("DB_ACCOUNT")
	ynabSecret     string = os.Getenv("YNAB_SECRET")
	dbClientID     string = os.Getenv("DB_CLIENT_ID")
	dbClientSecret string = os.Getenv("DB_CLIENT_SECRET")
	ynabBudgetID   string = os.Getenv("YNAB_BUDGET_ID")
	ynabAccountID  string = os.Getenv("YNAB_ACCOUNT_ID")

	oauth2HttpContext context.Context = context.Background()
	currentToken                      = &oauth2.Token{}
)

var oauth2Conf = &oauth2.Config{
	ClientID:     dbClientID,
	ClientSecret: dbClientSecret,
	Scopes:       []string{"read_transactions", "read_accounts", "read_credit_cards_list_with_details", "read_credit_card_transactions", "offline_access"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  DbAPIBaseURL + "gw/oidc/authorize",
		TokenURL: DbAPIBaseURL + "gw/oidc/token",
	},
}

func main() {
	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/authorized", AuthorizedHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	if currentToken.RefreshToken == "" {
		url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	var DbCashTransactions string
	var convertedTransactions []ynabTransaction
	accountType, err := GetAccountType(accountNumber)
	if err != nil {
		fmt.Fprintf(w, "%s", err.Error())
	}
	switch accountType {
	case Cash:
		accountNumber = strings.ReplaceAll(accountNumber, " ", "")
		DbCashTransactions := GetCashTransactions(accountNumber)
		convertedTransactions = ConvertCashTransactionsToYNAB(DbCashTransactions)
	case Credit:
		DbCashTransactions, err := GetCreditTransactions(accountNumber)
		convertedTransactions = ConvertCreditTransactionsToYNAB(DbCashTransactions)
		if err != nil {
			log.Fatal(err)
		}
	}
	if DbCashTransactions == "" {
		return
	}
	PostTransactionsToYNAB(ynabSecret, ynabBudgetID, convertedTransactions)
}

// GetAccountType returns "cash" or "credit" based on the kind of account ID.
func GetAccountType(accountNumber string) (AccountType, error) {
	isCorrectIban, _, _ := iban.IsCorrectIban(accountNumber, false)
	if isCorrectIban {
		return Cash, nil
	}
	if IsCredit(accountNumber) {
		return Credit, nil
	}
	return "", errors.New("Account ID is not recognized as a valid IBAN or the last 4 digits of a credit card")
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

// IsCredit tests if this looks like the last 4 digits of a CC number.
func IsCredit(accountNumber string) bool {
	re := regexp.MustCompile(`^[0-9]{4}$`)
	return re.MatchString(accountNumber)
}

// GetCreditTransactions gets transactions from a credit card.
func GetCreditTransactions(last4 string) (DbCreditTransactionsList, error) {
	var transactions DbCreditTransactionsList
	technicalID, err := getTechnicalID(last4)
	if err != nil {
		return transactions, err
	}
	urlParams := "?technicalId=" + technicalID
	urlParams = urlParams + "&bookingDateTo=" + time.Now().Format("2006-01-02")
	urlParams = urlParams + "&bookingDateFrom=" + time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	err = DbAPIRequest("gw/Dbapi/banking/creditCardTransactions/v1"+urlParams, &transactions)
	if err != nil {
		return transactions, err
	}
	return transactions, nil
}

func getTechnicalID(last4 string) (string, error) {
	var cardsMap DbCreditCardsList
	err := DbAPIRequest("gw/Dbapi/banking/creditCards/v1/", &cardsMap)
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

// DbAPIRequest makes a call to the DB API and loads the JSON response into a slice.
func DbAPIRequest(path string, recipient interface{}) error {
	request, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get(DbAPIBaseURL + path)
	if err != nil {
		return err
	}
	defer request.Body.Close()
	json.NewDecoder(request.Body).Decode(&recipient)
	bodyBytes, _ := ioutil.ReadAll(request.Body)
	bodyString := string(bodyBytes)
	fmt.Print(bodyString)
	return nil
}

// GetCashTransactions gets transactions from a cash account.
func GetCashTransactions(iban string) DbCashTransactionsList {
	var transactionsList DbCashTransactionsList
	DbAPIRequest("gw/Dbapi/banking/transactions/v2/?limit=100&iban="+iban, &transactionsList)
	return transactionsList
}

// ConvertCreditTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func ConvertCreditTransactionsToYNAB(incomingTransactions DbCreditTransactionsList) []ynabTransaction {
	transactions := incomingTransactions.Items
	var convertedTransactions []ynabTransaction
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	accountNumber := ynabAccountID
	for _, transaction := range transactions {
		go func(t DbCreditTransaction) {
			resultChannel <- convertCreditTransactionToYNAB(accountNumber, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

// ConvertCashTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func ConvertCashTransactionsToYNAB(incomingTransactions DbCashTransactionsList) []ynabTransaction {
	transactions := incomingTransactions.Transactions
	var convertedTransactions []ynabTransaction
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	accountNumber := ynabAccountID
	for _, transaction := range transactions {
		go func(t DbCashTransaction) {
			resultChannel <- convertTransactionToYNAB(accountNumber, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

func convertTransactionToYNAB(accountNumber string, incomingTransaction DbCashTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importID := createImportID(incomingTransaction.ID)
	transaction := ynabTransaction{
		AccountID: accountNumber,
		Date:      date,
		Amount:    convertToMilliunits(incomingTransaction.Amount),
		PayeeName: &incomingTransaction.CounterPartyName,
		Memo:      &incomingTransaction.PaymentReference,
		Cleared:   transaction.ClearingStatusCleared,
		Approved:  false,
		ImportID:  &importID,
	}
	return transaction
}

func convertCreditTransactionToYNAB(accountNumber string, incomingTransaction DbCreditTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importIDSource := incomingTransaction.BookingDate + fmt.Sprintf("%f", incomingTransaction.AmountInAccountCurrency.Amount)
	importID := createImportID(importIDSource)
	transaction := ynabTransaction{
		AccountID: accountNumber,
		Date:      date,
		Amount:    convertToMilliunits(incomingTransaction.AmountInAccountCurrency.Amount),
		PayeeName: &incomingTransaction.ReasonForPayment,
		Cleared:   transaction.ClearingStatusCleared,
		Approved:  false,
		ImportID:  &importID,
	}
	return transaction
}

// Creates a 32-character unique ID based on a given string.
func createImportID(source string) string {
	sum := sha256.Sum256([]byte(source))
	importID := fmt.Sprintf("%x", sum)[0:32]
	return importID
}

func convertToMilliunits(value float32) int64 {
	return int64(value * 1000)
}

// PostTransactionsToYNAB posts transactions to YNAB.
func PostTransactionsToYNAB(accessToken string, budgetID string, transactions []ynabTransaction) {
	c := ynab.NewClient(accessToken)
	_, err := c.Transaction().BulkCreateTransactions(budgetID, transactions)
	if err != nil {
		log.Fatal(err)
	}
}

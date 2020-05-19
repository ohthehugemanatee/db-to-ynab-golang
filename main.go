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

type dbCashTransaction struct {
	BookingDate      string
	CounterPartyName string
	PaymentReference string
	ID               string
	Amount           float32
}

type dbCreditTransaction struct {
	BookingDate             string
	ReasonForPayment        string
	AmountInAccountCurrency struct {
		Amount float32
	}
}

type dbCashTransactionsList struct {
	Transactions []dbCashTransaction
}

type dbCreditTransactionsList struct {
	Transactions []dbCreditTransaction
}

type dbCreditCard struct {
	TechnicalID string
	SecurePAN   string
}

type dbCreditCardsList struct {
	Items []dbCreditCard
}

// AccountType denotes a type of bank account, eg cash vs credit.
type AccountType string

const (
	// Cash is for cash accounts, eg checking, savings.
	Cash AccountType = "cash"
	// Credit is for credit card accounts, eg visa or mastercard.
	Credit AccountType = "credit"
)

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
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	if currentToken.RefreshToken == "" {
		url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	accountID := os.Getenv("DB_ACCOUNT")
	var dbCashTransactions string
	var convertedTransactions []ynabTransaction
	accountType, err := GetAccountType(accountID)
	if err != nil {
		log.Fatal(err)
	}
	switch accountType {
	case Cash:
		accountID = strings.ReplaceAll(accountID, " ", "")
		dbCashTransactions := GetCashTransactions(accountID)
		convertedTransactions = ConvertCashTransactionsToYNAB(dbCashTransactions)
	case Credit:
		dbCashTransactions, err := GetCreditTransactions(accountID)
		convertedTransactions = ConvertCreditTransactionsToYNAB(dbCashTransactions)
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Printf("Account number not recognized")
	}
	if dbCashTransactions == "" {
		return
	}
	PostTransactionsToYNAB(os.Getenv("YNAB_SECRET"), os.Getenv("YNAB_BUDGET_ID"), convertedTransactions)
}

// GetAccountType returns "cash" or "credit" based on the kind of account ID.
func GetAccountType(accountID string) (AccountType, error) {
	isCorrectIban, _, _ := iban.IsCorrectIban(accountID, false)
	if isCorrectIban {
		return Cash, nil
	}
	if IsCredit(accountID) {
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
func IsCredit(accountID string) bool {
	re := regexp.MustCompile(`^[0-9]{4}$`)
	return re.MatchString(accountID)
}

// GetCreditTransactions gets transactions from a credit card.
func GetCreditTransactions(last4 string) (string, error) {
	technicalID := getTechnicalID(last4)
	if technicalID == "" {
		return "", fmt.Errorf("No credit card found on account with last digits %v", last4)
	}
	urlParams := "?technicalId=" + technicalID
	urlParams = urlParams + "&bookingDateTo=" + time.Now().Format("2006-01-02")
	urlParams = urlParams + "&bookingDateFrom=" + time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	transactions, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get("https://simulator-api.db.com/gw/dbapi/banking/creditCardTransactions/v1" + urlParams)
	if err != nil {
		log.Fatal(err)
	}
	defer transactions.Body.Close()
	body, err := ioutil.ReadAll(transactions.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body), nil
}

func getTechnicalID(last4 string) string {
	creditCards, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get("https://simulator-api.db.com/gw/dbapi/banking/creditCards/v1/")
	if err != nil {
		log.Fatal(err)
	}
	defer creditCards.Body.Close()
	var cardsMap dbCreditCardsList
	json.NewDecoder(creditCards.Body).Decode(&cardsMap)
	for _, cardDetails := range cardsMap.Items {
		if cardDetails.SecurePAN == "************"+last4 {
			return cardDetails.TechnicalID
		}
	}
	return ""
}

// GetCashTransactions gets transactions from a cash account.
func GetCashTransactions(iban string) string {
	transactions, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get("https://simulator-api.db.com/gw/dbapi/banking/transactions/v2/?limit=100&iban=" + iban)
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

// ConvertCreditTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func ConvertCreditTransactionsToYNAB(incomingTransactions string) []ynabTransaction {
	var marshalledTransactions dbCreditTransactionsList
	err := json.Unmarshal([]byte(incomingTransactions), &marshalledTransactions)
	if err != nil {
		log.Fatal(err)
	}
	transactions := marshalledTransactions.Transactions
	var convertedTransactions []ynabTransaction
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	accountID := os.Getenv("YNAB_ACCOUNT_ID")
	for _, transaction := range transactions {
		go func(t dbCreditTransaction) {
			resultChannel <- convertCreditTransactionToYNAB(accountID, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

// ConvertCashTransactionsToYNAB converts a JSON string of transactions to YNAB format.
func ConvertCashTransactionsToYNAB(incomingTransactions string) []ynabTransaction {
	var marshalledTransactions dbCashTransactionsList
	err := json.Unmarshal([]byte(incomingTransactions), &marshalledTransactions)
	if err != nil {
		log.Fatal(err)
	}
	transactions := marshalledTransactions.Transactions
	var convertedTransactions []ynabTransaction
	resultChannel := make(chan ynabTransaction)
	defer close(resultChannel)
	accountID := os.Getenv("YNAB_ACCOUNT_ID")
	for _, transaction := range transactions {
		go func(t dbCashTransaction) {
			resultChannel <- convertTransactionToYNAB(accountID, transaction)
		}(transaction)
		for i := 0; i < len(transactions); i++ {
			convertedTransaction := <-resultChannel
			convertedTransactions = append(convertedTransactions, convertedTransaction)
		}
	}
	return convertedTransactions
}

func convertTransactionToYNAB(accountID string, incomingTransaction dbCashTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importID := createImportID(incomingTransaction.ID)
	transaction := ynabTransaction{
		AccountID: accountID,
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

func convertCreditTransactionToYNAB(accountID string, incomingTransaction dbCreditTransaction) ynabTransaction {
	date, err := api.DateFromString(incomingTransaction.BookingDate)
	if err != nil {
		log.Fatal(err)
	}
	importIDSource := incomingTransaction.BookingDate + fmt.Sprintf("%f", incomingTransaction.AmountInAccountCurrency.Amount)
	importID := createImportID(importIDSource)
	transaction := ynabTransaction{
		AccountID: accountID,
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

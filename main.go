package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"go.bmvs.io/ynab/api"
	"go.bmvs.io/ynab/api/transaction"
	"golang.org/x/oauth2"
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
	fmt.Printf("Listening for input")
}

// RootHandler handles HTTP requests to /
func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>")
	if currentToken.Valid() {
		fmt.Fprintf(w, "Already authorized")
		iban := os.Getenv("DB_IBAN")
		fmt.Fprintf(w, GetTransactions(iban))
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

// ConvertTransactionToYNAB converts a transaction to YNAB format.
func ConvertTransactionToYNAB(incomingTransaction interface{}) transaction.PayloadTransaction {
	date, err := api.DateFromString("2019-11-04")
	if err != nil {
		log.Fatal(err)
	}
	payee := string("Rossmann")
	category := string("5")
	memo := string("POS MIT PIN. Mein Drogeriemarkt, Leipziger Str., 212+ZKLE 911/696682-X-ABC")
	importID := string("_2FMRe0AhzLaZu14Cz-lol2H_DDY4z9yIOJKrDlDjHCSCjlJk4dfM_2MOWo6JSezeNJJz5Fm23hOEFccXR0AXmZFmyFv_dI6xHu-DADUYh-_ue-2e1let853sS4-glBM")
	expected := transaction.PayloadTransaction{
		AccountID:  "account-id",
		Date:       date,
		Amount:     -19050,
		PayeeName:  &payee,
		CategoryID: &category,
		Memo:       &memo,
		Cleared:    transaction.ClearingStatusCleared,
		Approved:   false,
		ImportID:   &importID,
	}
	return expected
}

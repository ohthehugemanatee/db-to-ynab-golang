package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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

var currentToken = &oauth2.Token{}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/authorized", receiveHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>")
	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Fprintf(w, "Authenticate with <a href=\"%v\">Deutsche Bank</a>", url)
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var code string = r.URL.Query().Get("code")

	if code == "" {
		log.Fatal("Code returned was empty")
	}
	tok, err := oauth2Conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	currentToken = tok
}

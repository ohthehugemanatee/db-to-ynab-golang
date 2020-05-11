package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/authorized", receiveHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>")
	conf := getConf()
	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Fprintf(w, "Authenticate with <a href=\"%v\">Deutsche Bank</a>", url)
}

func getConf() *oauth2.Config {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("DB_CLIENT_ID"),
		ClientSecret: os.Getenv("DB_CLIENT_SECRET"),
		Scopes:       []string{"read_transactions", "read_accounts", "read_credit_cards_list_with_details", "read_credit_card_transactions", "offline_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://simulator-api.db.com/gw/oidc/authorize",
			TokenURL: "https://simulator-api.db.com/gw/oidc/token",
		},
	}
	return conf
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conf := getConf()
	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	var code string = r.URL.Query().Get("code")

	if code == "" {
		log.Fatal("Code returned was empty")
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(tok.TokenType)
}

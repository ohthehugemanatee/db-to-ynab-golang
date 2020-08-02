package dbapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

var (
	accountNumber  string = os.Getenv("DB_ACCOUNT")
	dbClientID     string = os.Getenv("DB_CLIENT_ID")
	dbClientSecret string = os.Getenv("DB_CLIENT_SECRET")
	dbAPIBaseURL   string = os.Getenv("DB_API_ENDPOINT_HOSTNAME")
	redirectURL    string = os.Getenv("REDIRECT_BASE_URL") + "/authorized"
	currentToken          = &oauth2.Token{}
)

var oauth2Conf = &oauth2.Config{
	ClientID:     dbClientID,
	ClientSecret: dbClientSecret,
	RedirectURL:  redirectURL,
	Scopes:       []string{"read_transactions", "read_accounts", "read_credit_cards_list_with_details", "read_credit_card_transactions", "offline_access"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  dbAPIBaseURL + "gw/oidc/authorize",
		TokenURL: dbAPIBaseURL + "gw/oidc/token",
	},
}

var oauth2HttpContext context.Context = context.Background()

// Authorize checks the current token and returns an authorization URL if necessary.
func Authorize() string {
	if currentToken.RefreshToken == "" {
		url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		return url
	}
	return ""
}

// CheckParams ensures that all parameters are provided and fails hard if not.
func CheckParams() {
	var params = map[string]string{
		"accountNumber":  accountNumber,
		"dbClientID":     dbClientID,
		"dbClientSecret": dbClientSecret,
		"dbAPIBaseURL":   dbAPIBaseURL,
		"redirectURL":    redirectURL,
	}
	for i, v := range params {
		if v == "" {
			panic("Missing/empty parameter" + i)
		}
	}
}

// AuthorizedHandler handles the oauth HTTP response.
func AuthorizedHandler(w http.ResponseWriter, r *http.Request) {
	var code string = r.URL.Query().Get("code")

	if code == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Deutsche Bank returned an empty code."))
		return
	}
	UpdateToken(code)
	http.Redirect(w, r, "/", http.StatusFound)
}

// dbAPIRequest makes a call to the DB API and loads the JSON response into a slice.
func dbAPIRequest(path string, recipient interface{}) error {
	request, err := oauth2Conf.Client(oauth2HttpContext, currentToken).Get(dbAPIBaseURL + path)
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

// UpdateToken updates the current token using an update code.
func UpdateToken(code string) {
	tok, err := oauth2Conf.Exchange(oauth2HttpContext, code)
	if err != nil {
		log.Fatal(err)
	}
	SetCurrentToken(tok)
}

// SetCurrentToken sets the currently active token. Mostly useful for tests.
func SetCurrentToken(token *oauth2.Token) {
	currentToken = token
}

# db-to-ynab-golang

An API based sync from Deutsche Bank accounts to You Need a Budget.

![Go](https://github.com/ohthehugemanatee/db-to-ynab-golang/workflows/Go/badge.svg?branch=master)

This application is still in active development (see #todo, below), but already works well for home use. Please submit issues as you find them!

## To run

It is intended to run a separate instance for each account you want to sync. You can (and should!) re-use the same `DB_CLIENT_ID` and `DB_CLIENT_SECRET` for all of your own instances. Use a cronjob or similar to touch the endpoint for every sync operation (I suggest twice a day). I do not recommend using this to sync anyone else's accounts.

Create a file `.env` containing environment variables:

```
DB_CLIENT_ID
DB_CLIENT_SECRET
DB_ACCOUNT
DB_API_ENDPOINT_BASEURI
REDIRECT_BASE_URL
YNAB_SECRET
YNAB_BUDGET_ID
YNAB_ACCOUNT_ID
```

You have to create an App at [developer.db.com](https://developer.db.com) to get the DB client ID and secret. Note that there is a slow (~2 weeks!) process for approval to get access to real live bank data. `DB_ACCOUNT` is either the IBAN of a cash account, or the last 4 digits of a credit card number. `DB_API_ENDPOINT_HOSTNAME` is the hostname of the DB api endpoint. It is `https://simulator-api.db.com/` for apps in the sandbox, and `https://api.db.com/` for live apps.

[Create a YNAB personal access token](https://api.youneedabudget.com/#personal-access-tokens) to use as your YNAB secret. The budget and account IDs are UUIDs you can get from the URL of the target account. For example, when viewing your account the URL may be `https://app.youneedabudget.com/ba1f67f1-5fba-4314-b4a3-94256409ff57/accounts/822de6c0-6967-4ad3-d4cf-f227dd58a7f9`. In that case the Budget ID is `ba1f67f1-5fba-4314-b4a3-94256409ff57`, and the account ID is `822de6c0-6967-4ad3-d4cf-f227dd58a7f9`.

`REDIRECT_BASE_URL` is the accessible (to you) URL of this application. As a part of the DB authentication flow, the DB API has to validate that it is redirecting you to an allowed URL (per your API app).

With those env vars in `.env`, you're ready to run the application.

*If you have docker*: run the application with `./start.sh`, or by hand with `docker run -p 3000:3000 --env-file .env ohthehugemanatee/db-ynab-sync`.
*If you don't have docker*: install golang, compile with `go build` and run.

With a web browser, visit port `3000` wherever it's running - likely `http://localhost:3000`. On your first visit it will redirect you to the DB authentication page, where you must sign into your account. On all subsequent visits, it will simply sync.

NB:

* on the DB app you create, the redirect should be the accessible (to you) URL of the running application, with path `/authorized`. For example, `http://localhost:3000/authorized`.
* The token received from DB is good for a month, updated each time you run the sync. So as long as you're sync'ing more than once a month, you should only have to manually enter credentials the first time. 
* The token is kept in memory only; when you restart the application you will need to authenticate again.
* This application will duplicate transactions imported through other methods, eg CSV import or other tools.

## To Develop

This project is configured with [a dev container for VSCode](https://code.visualstudio.com/docs/remote/containers). If you have VSCode with the [Remote - Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers), your IDE will prompt you to open in the container. That should have everything you need to get running. You can set the required environment variables in a local `.devcontainer/.env` file.

Issues and PRs are welcome!

### To implement your own bank connector

There is no concept of dynamic plugins in golang, really. Your bank connector will have to be compiled in.

* Add a package for your bank in its own subdirectory.
* Write a struct that implements the `BankConnector` interface. It must implement:
```
// Checks if the account number is valid for this connector.
IsValidAccountNumber(string) (bool, error)
// Gets YNAB formatted transactions.
GetTransactions(string) ([]ynabTransaction, error)
// Returns an oauth authorization url if necessary.
Authorize() string
// Handles an oauth response if necessary
AuthorizedHandler(http.ResponseWriter, *http.Request)
```
* Add your struct to the `availableConnectors []BankConnector` slice in `main.go`. 
* (first person only) improve how `main.GetConnector()` works to differentiate between DB and your own bank. :) Pretty quickly we'll have to move to an environment variable to select your connector.

Make a PR even with your work-in-progress, I'm happy to help you out!

### Gotchas in DB sandbox

If you start a new API app with DB, you are confined to a sandbox of test accounts/data until your app has been approved. Everything works as expected, except:

* The sandbox test users have invalid IBAN numbers. So you cannot test cash accounts
* The sandbox test users only have credit card transactions up until 2017. So you cannot test credit card accounts.

I have notified DB about these issues, but if you're helping out you deserve to know, too!

## Current status

### Working

* Sync cash accounts (checking, savings). It syncs the last 100 transactions every time you run it. No, you will not get duplicate transactions from these repeats. 
* Sync credit cards. It syncs the last 30 days of transactions every time you run it.

### TODO

* refactor tests for clarity
* more consistent error passing/handling
* sync upcoming transactions which haven't posted yet.

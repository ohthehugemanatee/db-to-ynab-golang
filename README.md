# db-to-ynab-golang

An API based sync from Deutsche Bank accounts to You Need a Budget.

![Go](https://github.com/ohthehugemanatee/db-to-ynab-golang/workflows/Go/badge.svg?branch=master)

This application is still in development (see #todo, below), but already works perfectly well for home use. Please submit issues as you find them!

## To run

It is intended to run a separate instance for each account you want to sync. You can (and should!) re-use the same `DB_CLIENT_ID` and `DB_CLIENT_SECRET` for all of your own instances. I do not recommend using this to sync anyone else's accounts.

Set environment variables:

```
DB_CLIENT_ID
DB_CLIENT_SECRET
DB_ACCOUNT
YNAB_SECRET
YNAB_BUDGET_ID
YNAB_ACCOUNT_ID
```

You have to create an App at [developer.db.com](https://developer.db.com) to get the DB client ID and secret. `DB_ACCOUNT` is the IBAN of a checking account, or the last 4 digits of a credit card number. 

[Create a YNAB personal access token](https://api.youneedabudget.com/#personal-access-tokens) to use as your YNAB secret. The budget and account IDs are UUIDs you can get from the URL of the target account. For example, when viewing your account the URL may be `https://app.youneedabudget.com/ba1f67f1-5fba-4314-b4a3-94256409ff57/accounts/822de6c0-6967-4ad3-d4cf-f227dd58a7f9`. In that case the Budget ID is `ba1f67f1-5fba-4314-b4a3-94256409ff57`, and the account ID is `822de6c0-6967-4ad3-d4cf-f227dd58a7f9`.

With those env vars set, run the application. With a web browser, visit port `3000` wherever it's running - likely `http://localhost:3000`. On your first visit it will redirect you to the DB authentication page, where you must sign into your account. On all subsequent visits, it will simply sync.

NB:

* on the DB app you create, the redirect should be the accessible (to you) URL of the running application, with path `/authorized`. For example, `http://localhost:3000/authorized`.
* The token received from DB is good for a month, updated each time you run the sync. So as long as you're sync'ing more than once a month, you should only have to manually enter credentials once. 
* The token is kept in memory only; when you restart the application you will need to authenticate again.
* The transactions created by this sync engine will be duplicated if you run manual CSV imports using the interface.
* This application will duplicate transactions imported through other methods, eg CSV import or other tools.

## Current status

### Working

* Sync cash accounts (checking, savings). It syncs the last 100 transactions every time you run it. No, you will not get duplicate transactions from these repeats. 
* Sync credit cards. It syncs the last 30 days of transactions every time you run it.

### TODO

* refactor a bit for clarity.
* improve test coverage.
* include dockerfile in the repo, publish on Docker hub
* sync upcoming transactions which haven't posted yet.

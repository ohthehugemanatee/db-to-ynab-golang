package main

import "net/http"

func main() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	http.ListenAndServe(":3000", nil)
}

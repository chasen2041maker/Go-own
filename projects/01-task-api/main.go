package main

import (
	"log"
	"net/http"
)

func main() {
	const address = ":8080"
	log.Printf("task API listening on http://localhost%s", address)
	log.Fatal(http.ListenAndServe(address, newHandler()))
}

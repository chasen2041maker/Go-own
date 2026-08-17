package main

import (
	"log"
	"net"
	"net/http"
)

const address = "127.0.0.1:8080"

func main() {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("task API listening on http://%s", address)
	log.Fatal(http.Serve(listener, newHandler()))
}

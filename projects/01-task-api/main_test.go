package main

import "testing"

func TestAddressUsesExactLoopbackEndpoint(t *testing.T) {
	if got, want := address, "127.0.0.1:8080"; got != want {
		t.Fatalf("address = %q, want %q", got, want)
	}
}

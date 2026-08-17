package main

import (
	"bytes"
	"testing"
)

func TestGreeting(t *testing.T) {
	var output bytes.Buffer
	greet(&output)

	if got, want := output.String(), "Hello, Go!\n"; got != want {
		t.Fatalf("greet() = %q, want %q", got, want)
	}
}

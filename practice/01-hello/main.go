package main

import (
	"io"
	"os"
)

func greet(w io.Writer) {
	io.WriteString(w, "Hello, Go!\n")
}

func main() {
	greet(os.Stdout)
}

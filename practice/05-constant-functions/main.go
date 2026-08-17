package main

import (
	"fmt"
	"unsafe"
)

const (
	a = "abc"
	b = len(a)
	c = unsafe.Sizeof(a)
)

func main() {
	fmt.Printf("a=%s, b=%d, c=%d\n", a, b, c)
}

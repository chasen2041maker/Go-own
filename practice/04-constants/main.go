package main

import (
	"fmt"
)

const (
	unknown = 0
	female  = 1
	male    = 2
)

func main() {
	const b string = "abc"
	const c = "de"
	fmt.Println("hello,world", b, c)

	fmt.Println("=============================================================")

	const length int = 10
	const width int = 5
	var area int
	const a, g, h = 1, false, "str"

	area = length * width
	fmt.Printf("面积为 ： %d", area)
	println()
	println(a, b, c)
	fmt.Printf("未知=%d, 女性=%d, 男性=%d\n", unknown, female, male)
}

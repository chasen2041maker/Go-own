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

	fmt.Println("=== 任务 1：基础常量声明 ===")
	basicConstants()

	fmt.Println("\n=== 任务 2：iota 枚举（星期）===")
	weekdayExample()

	fmt.Println("\n=== 任务 3：iota 跳过值 ===")
	permissionExample()

	fmt.Println("\n=== 任务 4：常量表达式计算 ===")
	sizeCalculations()
}

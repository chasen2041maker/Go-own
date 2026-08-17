package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：基础常量声明 ===")
	basicConstants()

	fmt.Println("\n=== 任务 2：iota 枚举（星期）===")
	weekdayExample()

	fmt.Println("\n=== 任务 3：iota 跳过值 ===")
	permissionExample()

	fmt.Println("\n=== 任务 4：常量表达式计算 ===")
	sizeCalculations()
}

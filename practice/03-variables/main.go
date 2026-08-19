package main

import "fmt"

func main() {
	var age int = 25

	var score float64
	score = 98.5

	name := "runoob"

	x, y := 10, 20

	const Pi = 3.14159
	const AppName = "RUNOOB"

	fmt.Println(name, age, score, x, y, Pi, AppName)

	fmt.Println("=== 任务 1：变量声明的三种方式 ===")
	declareVariables()

	fmt.Println("\n=== 任务 2：零值的理解 ===")
	showZeroValues()

	fmt.Println("\n=== 任务 3：类型转换计算 ===")
	calculateAverage()

	fmt.Println("\n=== 任务 4：多变量声明与交换 ===")
	swapAndPrint()
}

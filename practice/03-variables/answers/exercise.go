package main

import "fmt"

// declareVariables 演示三种变量声明方式
func declareVariables() {
	// 方式1：var 关键字 + 显式类型
	// 这是最完整的声明方式，明确指定了类型
	var message1 string = "显式声明"

	// 方式2：var 关键字 + 自动类型推断
	// Go 编译器会根据右边的值自动推断类型
	// 这里推断出 message2 是 string 类型
	var message2 = "自动推断"

	// 方式3：短变量声明（最常用）
	// := 只能在函数内部使用，不能用于包级别变量
	// 自动推断类型并声明变量
	message3 := "短声明"

	fmt.Printf("方式1: %s\n", message1)
	fmt.Printf("方式2: %s\n", message2)
	fmt.Printf("方式3: %s\n", message3)
}

// showZeroValues 展示 Go 中各类型的零值
func showZeroValues() {
	// 声明变量但不初始化时，Go 会自动赋予零值
	// 这避免了"未初始化变量"的问题

	var i int          // int 的零值是 0
	var f float64      // float64 的零值是 0.0
	var s string       // string 的零值是空字符串 ""
	var b bool         // bool 的零值是 false

	// %v 是通用格式化符号，可以打印任何类型的值
	fmt.Printf("int 的零值: %v\n", i)
	fmt.Printf("float64 的零值: %v\n", f)
	fmt.Printf("string 的零值: %v\n", s)  // 空字符串看起来什么都没有
	fmt.Printf("bool 的零值: %v\n", b)
}

// calculateAverage 计算整数的浮点平均值
func calculateAverage() {
	a := 10
	b := 20
	c := 25

	// 关键点：类型转换
	// 如果写 (a + b + c) / 3，结果是 int 类型，会得到 18（截断小数）
	// 必须先将其中一个操作数转为 float64，整个表达式才会变成浮点运算
	sum := float64(a + b + c)  // 先求和再转换
	average := sum / 3.0       // 除以浮点数，得到浮点结果

	// 也可以写成一行：
	// average := float64(a + b + c) / 3.0

	// %.2f 表示格式化为浮点数，保留 2 位小数
	fmt.Printf("平均值: %.2f\n", average)
}

// swapAndPrint 演示多变量声明和交换
func swapAndPrint() {
	// 多变量声明：一行同时声明并初始化多个变量
	// Go 会按顺序将右边的值赋给左边的变量
	x, y := 100, 200

	fmt.Printf("交换前: x=%d, y=%d\n", x, y)

	// Go 的多重赋值特性：右边的表达式会先全部计算完，再赋值给左边
	// 执行顺序：
	// 1. 先计算右边：临时保存 y 的值(200) 和 x 的值(100)
	// 2. 再赋值：x = 200, y = 100
	// 因此不需要临时变量就能交换
	x, y = y, x

	fmt.Printf("交换后: x=%d, y=%d\n", x, y)
}

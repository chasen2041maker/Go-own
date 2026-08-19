package main

import "fmt"

func declareVariables() {
	var message1 string = "显式声明"
	var message2 = "自动判断"
	message3 := "短引号"

	fmt.Printf("方式1:%s\n", message1)
	fmt.Printf("方式2:%s\n", message2)
	fmt.Printf("方式3:%s\n", message3)
}

func showZeroValues() {
	var i int
	var f float64
	var s string
	var b bool

	fmt.Printf("int 的零值：%v\n", i)
	fmt.Printf("float64 的零值: %v\n", f)
	fmt.Printf("string 的零值: %v\n", s) // 空字符串看起来什么都没有
	fmt.Printf("bool 的零值: %v\n", b)
}

func calculateAverage() {
	a := 10
	b := 20
	c := 23

	average := float64(a + b + c)
	fmt.Printf("平均值: %.2f\n", average)
}

func swapAndPrint() {
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

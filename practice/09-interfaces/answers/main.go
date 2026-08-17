package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：定义和实现接口 ===")
	// 创建不同的形状
	circle := Circle{Radius: 5}
	rect := Rectangle{Width: 4, Height: 6}

	// 接口变量可以存储任何实现了该接口的类型
	var shape Shape

	shape = circle
	fmt.Printf("圆形: 面积=%.2f, 周长=%.2f\n", shape.Area(), shape.Perimeter())

	shape = rect
	fmt.Printf("矩形: 面积=%.2f, 周长=%.2f\n", shape.Area(), shape.Perimeter())

	fmt.Println("\n=== 任务 2：接口作为参数 ===")
	printShapeInfo(circle)
	printShapeInfo(rect)

	fmt.Println("\n=== 任务 3：类型断言 ===")
	describe(circle)
	describe(rect)
	describe(42)
	describe("hello")

	fmt.Println("\n=== 任务 4：空接口 ===")
	printAnything("Go语言")
	printAnything(123)
	printAnything(3.14)
	printAnything(true)
	printAnything([]int{1, 2, 3})
}

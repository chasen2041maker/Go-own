package main

import (
	"fmt"
	"math"
)

// Shape 接口定义了形状必须实现的方法
// 接口是一组方法签名的集合
// 任何类型只要实现了这些方法，就自动实现了这个接口（隐式实现）
type Shape interface {
	Area() float64       // 计算面积
	Perimeter() float64  // 计算周长
}

// Circle 圆形结构体
type Circle struct {
	Radius float64
}

// Area 实现 Shape 接口的 Area 方法
// 注意：不需要显式声明 "implements Shape"
// 只要方法签名匹配，就自动实现了接口
func (c Circle) Area() float64 {
	// 圆的面积 = π × r²
	return math.Pi * c.Radius * c.Radius
}

// Perimeter 实现 Shape 接口的 Perimeter 方法
func (c Circle) Perimeter() float64 {
	// 圆的周长 = 2 × π × r
	return 2 * math.Pi * c.Radius
}

// Rectangle 矩形结构体
type Rectangle struct {
	Width  float64
	Height float64
}

// Area 实现 Shape 接口的 Area 方法
func (r Rectangle) Area() float64 {
	// 矩形面积 = 宽 × 高
	return r.Width * r.Height
}

// Perimeter 实现 Shape 接口的 Perimeter 方法
func (r Rectangle) Perimeter() float64 {
	// 矩形周长 = 2 × (宽 + 高)
	return 2 * (r.Width + r.Height)
}

// printShapeInfo 接受任何实现了 Shape 接口的类型
// 这是多态的体现：同一个函数可以处理不同类型的对象
func printShapeInfo(s Shape) {
	// 不关心 s 的具体类型（Circle 还是 Rectangle）
	// 只需要知道它实现了 Area() 和 Perimeter() 方法
	fmt.Printf("面积: %.2f, 周长: %.2f\n", s.Area(), s.Perimeter())
}

// describe 使用类型断言和 type switch 判断具体类型
// interface{} 是空接口，可以接受任何类型
func describe(i interface{}) {
	// type switch 用于判断接口变量的具体类型
	switch v := i.(type) {
	case Circle:
		// v 的类型是 Circle
		fmt.Printf("这是一个圆形，半径=%.2f\n", v.Radius)
	case Rectangle:
		// v 的类型是 Rectangle
		fmt.Printf("这是一个矩形，宽=%.2f, 高=%.2f\n", v.Width, v.Height)
	case int:
		fmt.Printf("这是一个整数: %d\n", v)
	case string:
		fmt.Printf("这是一个字符串: %s\n", v)
	default:
		// 其他类型
		fmt.Printf("未知类型: %T, 值=%v\n", v, v)
	}
}

// printAnything 演示空接口的使用
// interface{} 可以接受任何类型的值
// 空接口没有方法，所以任何类型都实现了它
func printAnything(value interface{}) {
	// %T 打印类型，%v 打印值
	fmt.Printf("类型: %T, 值: %v\n", value, value)
}

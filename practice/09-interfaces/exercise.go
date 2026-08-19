package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

type Rectangle struct {
	Width  float64
	Height float64
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func printShapeInfo(s Shape) {
	fmt.Printf("面积: %.2f, 周长: %.2f\n", s.Area(), s.Perimeter())
}

func describe(i interface{}) {
	switch v := i.(type) {
	case Circle:
		fmt.Printf("这是一个圆形，半径=%.2f\n", v.Radius)
	case Rectangle:
		fmt.Printf("这是一个矩形，宽=%.2f, 高=%.2f\n", v.Width, v.Height)
	case int:
		fmt.Printf("这是一个整数: %d\n", v)
	case string:
		fmt.Printf("这是一个字符串: %s\n", v)
	default:
		fmt.Printf("未知类型: %T, 值=%v\n", v, v)
	}
}

func printAnything(value interface{}) {
	fmt.Printf("类型: %T, 值: %v\n", value, value)
}

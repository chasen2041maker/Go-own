package main

import (
	"errors"  // 用于创建错误对象
	"fmt"
)

// divide 执行除法运算，返回结果和可能的错误
// Go 的惯用错误处理模式：(结果, error)
func divide(a, b float64) (float64, error) {
	// 除数为 0 是非法操作，需要返回错误
	if b == 0 {
		// errors.New() 创建一个新的错误对象
		// 返回零值和错误
		return 0, errors.New("division by zero")
	}

	// 正常情况：返回结果和 nil（表示没有错误）
	return a / b, nil
}

// sum 计算任意数量整数的总和
// ...int 是可变参数语法（variadic parameters）
// 在函数内部，numbers 的类型是 []int（切片）
func sum(numbers ...int) int {
	// 初始化总和为 0
	total := 0

	// range 遍历可变参数
	// 这里不需要索引，所以用 _ 忽略
	for _, num := range numbers {
		total += num
	}

	return total
}

// rectangle 计算矩形的面积和周长
// 使用命名返回值：area 和 perimeter 在函数签名中声明
// 命名返回值会自动初始化为零值
func rectangle(width, height float64) (area, perimeter float64) {
	// 直接给命名返回值赋值
	// 不需要使用 := 声明新变量
	area = width * height
	perimeter = 2 * (width + height)

	// 裸 return：不带参数的 return
	// 会自动返回所有命名的返回值
	return  // 等价于 return area, perimeter
}

// processFile 演示 defer 的延迟执行特性
// defer 常用于资源清理，确保资源一定会被释放
func processFile(filename string) {
	fmt.Printf("打开文件: %s\n", filename)

	// defer 语句会在函数返回前执行
	// 无论函数是正常返回还是因 panic/return 提前退出
	// 这保证了资源清理代码一定会运行
	defer fmt.Printf("关闭文件: %s\n", filename)

	// 模拟文件处理
	fmt.Printf("处理文件: %s\n", filename)

	// 模拟发生错误，提前返回
	fmt.Println("发生错误！")
	return  // 即使这里提前返回，defer 也会执行

	// 下面的代码不会执行
	fmt.Println("这行不会被打印")
}

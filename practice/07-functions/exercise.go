package main

import (
	"errors"
	"fmt"
)

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}

	return a / b, nil
}

func sum(numbers ...int) int {
	total := 0

	for _, num := range numbers {
		total += num
	}

	return total
}

func rectangle(width, height float64) (area, perimeter float64) {
	area = width * height
	perimeter = 2 * (width + height)
	return
}

func processFile(filename string) {
	fmt.Printf("打开方式: %s\n", filename)

	defer fmt.Printf("关闭方式: %s\n", filename)

	fmt.Printf("处理文件: %s\n", filename)

	// 模拟发生错误，提前返回
	fmt.Println("发生错误！")
	return // 即使这里提前返回，defer 也会执行
}

package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：多返回值 - 除法函数 ===")
	// 测试正常除法
	result1, err1 := divide(10, 2)
	if err1 != nil {
		fmt.Printf("10 / 2: %v\n", err1)
	} else {
		fmt.Printf("10 / 2 = %.2f\n", result1)
	}

	// 测试除以零
	result2, err2 := divide(10, 0)
	if err2 != nil {
		fmt.Printf("10 / 0: %v\n", err2)
	} else {
		fmt.Printf("10 / 0 = %.2f\n", result2)
	}

	fmt.Println("\n=== 任务 2：可变参数 - 求和函数 ===")
	fmt.Printf("sum() = %d\n", sum())
	fmt.Printf("sum(5) = %d\n", sum(5))
	fmt.Printf("sum(1,2,3,4,5) = %d\n", sum(1, 2, 3, 4, 5))

	fmt.Println("\n=== 任务 3：命名返回值 - 矩形面积和周长 ===")
	area, perimeter := rectangle(5, 3)
	fmt.Printf("矩形 (宽5, 高3): 面积=%.2f, 周长=%.2f\n", area, perimeter)

	fmt.Println("\n=== 任务 4：defer 延迟执行 ===")
	processFile("data.txt")
}

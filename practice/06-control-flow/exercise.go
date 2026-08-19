package main

import "fmt"

func gradeClassifier(score int) string {
	if score < 0 || score > 100 {
		return "Invalid (无效)"
	}

	// 方式1：使用 if-else 链
	// 从高到低判断，因为已经排除了 >100 的情况
	// 所以 score >= 90 等价于 90 <= score <= 100
	if score >= 90 {
		return "A (优秀)"
	} else if score >= 80 {
		return "B (良好)"
	} else if score >= 70 {
		return "C (中等)"
	} else if score >= 60 {
		return "D (及格)"
	} else {
		return "F (不及格)"
	}
}

// multiplicationTable 打印九九乘法表
func multiplicationTable() {
	// 外层循环：控制行（1 到 9）
	// i 代表当前行，也是乘数的最大值
	for i := 1; i <= 9; i++ {
		// 内层循环：控制列（1 到 i）
		// j 是被乘数，从 1 到当前行号
		// 这样可以生成三角形的乘法表
		for j := 1; j <= i; j++ {
			// %-4s 表示左对齐，占用 4 个字符宽度
			// 这样可以让输出对齐，更美观
			fmt.Printf("%-4s", fmt.Sprintf("%d×%d=%d", j, i, j*i))
		}
		// 内层循环结束后换行，开始下一行
		fmt.Println()
	}
}

func findFirstDivisibleBy7(start int) int {
	maxAttempts := 100

	for i := start; i < start+maxAttempts; i++ {
		if i%7 == 0 {
			return i
		}
	}

	return -1
}

func printOddNumbers(limit int) {
	fmt.Print("奇数 (1-", limit, "): ")

	for i := 1; i <= limit; i++ {
		if i%2 == 0 {
			continue
		}
		fmt.Print(i, "")
	}

	fmt.Println()

}

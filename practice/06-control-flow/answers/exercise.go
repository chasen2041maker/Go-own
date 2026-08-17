package main

import "fmt"

// gradeClassifier 根据分数返回等级
func gradeClassifier(score int) string {
	// 边界检查：先处理非法输入
	// 这种"提前返回"的模式叫做 Guard Clauses（守卫子句）
	// 优点：减少嵌套层级，代码更清晰
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

	// 方式2：也可以用 switch（注释掉的替代实现）
	/*
	switch {
	case score >= 90:
		return "A (优秀)"
	case score >= 80:
		return "B (良好)"
	case score >= 70:
		return "C (中等)"
	case score >= 60:
		return "D (及格)"
	default:
		return "F (不及格)"
	}
	*/
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

// findFirstDivisibleBy7 找出第一个能被 7 整除的数
func findFirstDivisibleBy7(start int) int {
	// 限制最多检查 100 个数，防止无限循环
	maxAttempts := 100

	// 从 start 开始向后查找
	// 注意：这里 i 从 start 开始，不是从 0 开始
	for i := start; i < start+maxAttempts; i++ {
		// % 是取模（求余数）运算符
		// 如果 i % 7 == 0，说明 i 能被 7 整除
		if i%7 == 0 {
			// break 会立即退出循环
			// 但这里直接 return，更简洁
			return i
		}
	}

	// 如果循环结束都没找到，返回 -1 表示失败
	return -1
}

// printOddNumbers 只打印奇数
func printOddNumbers(limit int) {
	fmt.Print("奇数 (1-", limit, "): ")

	// 遍历 1 到 limit 的所有数字
	for i := 1; i <= limit; i++ {
		// 判断是否为偶数
		// 偶数的特征：除以 2 的余数为 0
		if i%2 == 0 {
			// continue 跳过本次循环，直接进入下一次迭代
			// 相当于"跳过偶数，不执行后面的打印"
			continue
		}

		// 只有奇数才会执行到这里
		fmt.Print(i, " ")
	}

	fmt.Println() // 最后换行
}

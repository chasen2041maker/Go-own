package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：成绩等级判断 ===")
	// 测试多个分数
	testScores := []int{95, 82, 71, 60, 45, -10, 105}
	for _, score := range testScores {
		grade := gradeClassifier(score)
		fmt.Printf("分数 %d: %s\n", score, grade)
	}

	fmt.Println("\n=== 任务 2：九九乘法表 ===")
	multiplicationTable()

	fmt.Println("\n=== 任务 3：找出第一个能被 7 整除的数 ===")
	start := 50
	result := findFirstDivisibleBy7(start)
	if result != -1 {
		fmt.Printf("从 %d 开始，第一个能被 7 整除的数是: %d\n", start, result)
	} else {
		fmt.Printf("从 %d 开始的 100 个数中，没有找到能被 7 整除的数\n", start)
	}

	fmt.Println("\n=== 任务 4：跳过偶数，只打印奇数 ===")
	printOddNumbers(20)
}

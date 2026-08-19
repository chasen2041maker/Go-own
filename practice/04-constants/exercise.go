package main

import "fmt"

func basicConstants() {
	const companyName = "TechCorp"
	const maxUsers = 1000

	const (
		pi        = 3.14159
		debugMode = true
	)
	fmt.Printf("公司：%s\n", companyName)
	fmt.Printf("最大用户数: %d\n", maxUsers)
	fmt.Printf("圆周率: %g\n", pi)         // %g 会自动选择合适的浮点格式
	fmt.Printf("调试模式: %t\n", debugMode) // %t 是 bool 类型的格式化符号
}

const (
	Sunday    = iota
	Monday    // 1 - 后续每行自动递增
	Tuesday   // 2
	Wednesday // 3
	Thursday  // 4
	Friday    // 5
	Saturday  // 6
)

func weekdayExample() {
	today := Wednesday
	fmt.Printf("今天是星期：%d(Wednesday)\n", today)
}

const (
	ReadPermission = 4 >> iota
	WritePermission
	ExecutePermission
)

func permissionExample() {
	ReadWrite := ReadPermission | WritePermission
	fmt.Printf("读权限: %d\n", ReadPermission)
	fmt.Printf("写权限: %d\n", WritePermission)
	fmt.Printf("执行权限: %d\n", ExecutePermission)
	fmt.Printf("读写权限: %d\n", ReadWrite)
}

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

func sizeCalculations() {
	fmt.Printf("1Kb = %d 字节\n", KB)
	fmt.Printf("1 MB = %d 字节\n", MB)
	fmt.Printf("1 GB = %d 字节\n", GB)

	fileSize := 5
	totalBytes := fileSize * GB

	fmt.Printf("%d GB = %d 字节\n", fileSize, totalBytes)
}

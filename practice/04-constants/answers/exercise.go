package main

import "fmt"

// basicConstants 演示基础常量声明
func basicConstants() {
	// 常量使用 const 关键字声明
	// 常量的值必须在编译时就能确定，不能是函数调用或运行时计算的结果

	// 方式1：单独声明每个常量
	const companyName = "TechCorp"
	const maxUsers = 1000

	// 方式2：使用括号声明一组常量（推荐）
	const (
		pi = 3.14159
		debugMode = true
	)

	fmt.Printf("公司: %s\n", companyName)
	fmt.Printf("最大用户数: %d\n", maxUsers)
	fmt.Printf("圆周率: %g\n", pi)  // %g 会自动选择合适的浮点格式
	fmt.Printf("调试模式: %t\n", debugMode)  // %t 是 bool 类型的格式化符号
}

// 在包级别声明星期枚举
// iota 是 Go 的常量生成器，在 const 块中使用
const (
	Sunday = iota     // 0 - iota 初始值为 0
	Monday            // 1 - 后续每行自动递增
	Tuesday           // 2
	Wednesday         // 3
	Thursday          // 4
	Friday            // 5
	Saturday          // 6
)
// 重点：每次遇到 const 关键字，iota 会重置为 0
// 在同一个 const 块内，iota 会自动递增

// weekdayExample 演示 iota 枚举的使用
func weekdayExample() {
	// 使用上面声明的枚举常量
	today := Wednesday  // Wednesday 的值是 3

	fmt.Printf("今天是星期: %d (Wednesday)\n", today)
}

// 文件权限枚举（倒序：4, 2, 1）
const (
	// 使用位移运算生成权限值
	// 4 >> iota 表示：4 右移 iota 位
	// iota=0: 4 >> 0 = 4 (二进制 100)
	// iota=1: 4 >> 1 = 2 (二进制 010)
	// iota=2: 4 >> 2 = 1 (二进制 001)
	ReadPermission = 4 >> iota   // 4
	WritePermission              // 2
	ExecutePermission            // 1
)

// permissionExample 演示权限位运算
func permissionExample() {
	// 位运算 OR (|) 可以组合多个权限
	// 4 (100) | 2 (010) = 6 (110)
	// 表示同时拥有读和写权限
	ReadWrite := ReadPermission | WritePermission

	fmt.Printf("读权限: %d\n", ReadPermission)
	fmt.Printf("写权限: %d\n", WritePermission)
	fmt.Printf("执行权限: %d\n", ExecutePermission)
	fmt.Printf("读写权限: %d\n", ReadWrite)
}

// 存储单位常量
const (
	KB = 1024           // 1 KB = 1024 字节
	MB = 1024 * KB      // 常量可以引用其他常量进行计算
	GB = 1024 * MB      // 这些计算在编译时完成，不影响运行性能
)

// sizeCalculations 演示常量表达式计算
func sizeCalculations() {
	// 这些都是编译时常量，可以直接打印
	fmt.Printf("1 KB = %d 字节\n", KB)
	fmt.Printf("1 MB = %d 字节\n", MB)
	fmt.Printf("1 GB = %d 字节\n", GB)

	// 变量和常量的运算
	fileSize := 5  // 这是变量，不是常量
	totalBytes := fileSize * GB  // 变量 * 常量 = 变量

	fmt.Printf("%d GB = %d 字节\n", fileSize, totalBytes)
}

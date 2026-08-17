package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：定义和初始化结构体 ===")
	// 方式1：字段名初始化（推荐）
	book1 := Book{
		Title:  "Go语言编程",
		Author: "张三",
		Pages:  300,
		Price:  59.9,
	}
	fmt.Printf("书籍1: %+v\n", book1)

	// 方式2：按顺序初始化
	book2 := Book{"Effective Go", "李四", 250, 45.0}
	fmt.Printf("书籍2: %+v\n", book2)

	// 方式3：零值初始化
	var book3 Book
	fmt.Printf("书籍3 (零值): %+v\n", book3)

	fmt.Println("\n=== 任务 2：值接收者方法 ===")
	p := Point{X: 3, Y: 4}
	dist := p.Distance()
	fmt.Printf("点 (%d, %d) 到原点的距离: %.2f\n", p.X, p.Y, dist)

	fmt.Println("\n=== 任务 3：指针接收者方法 ===")
	acc := BankAccount{
		AccountNumber: "123456",
		Balance:       1000.0,
	}
	fmt.Printf("初始余额: %.2f\n", acc.Balance)

	acc.Deposit(500)
	fmt.Printf("存款 500 后: %.2f\n", acc.Balance)

	success := acc.Withdraw(300)
	fmt.Printf("取款 300: 成功=%v, 余额=%.2f\n", success, acc.Balance)

	success = acc.Withdraw(2000)
	fmt.Printf("取款 2000: 成功=%v, 余额=%.2f\n", success, acc.Balance)

	fmt.Println("\n=== 任务 4：嵌套结构体 ===")
	emp := Employee{
		Name: "王五",
		Age:  30,
		Address: Address{
			City:   "北京",
			Street: "中关村大街1号",
		},
	}
	info := emp.FullInfo()
	fmt.Println(info)
}

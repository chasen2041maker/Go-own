package main

import (
	"fmt"
	"math"
)

// Book 表示一本书的信息
// 结构体字段首字母大写表示导出（公开），小写表示私有
type Book struct {
	Title  string  // 书名
	Author string  // 作者
	Pages  int     // 页数
	Price  float64 // 价格
}

// Point 表示二维坐标点
type Point struct {
	X int
	Y int
}

// Distance 计算点到原点的距离（值接收者）
// (p Point) 是值接收者，方法内部操作的是 p 的副本
// 值接收者适用于：方法不修改结构体 + 结构体比较小
func (p Point) Distance() float64 {
	// 勾股定理：distance = √(x² + y²)
	// math.Sqrt() 计算平方根
	// math.Pow(x, 2) 计算 x 的平方
	return math.Sqrt(math.Pow(float64(p.X), 2) + math.Pow(float64(p.Y), 2))
}

// BankAccount 表示银行账户
type BankAccount struct {
	AccountNumber string  // 账号
	Balance       float64 // 余额（小写开头，外部包无法访问）
}

// Deposit 存款（指针接收者）
// (acc *BankAccount) 是指针接收者，方法可以修改原始结构体
// 指针接收者适用于：需要修改结构体 或 结构体很大（避免复制）
func (acc *BankAccount) Deposit(amount float64) {
	// 直接修改原始结构体的字段
	// Go 会自动解引用，不需要写 (*acc).Balance
	acc.Balance += amount
}

// Withdraw 取款（指针接收者）
// 返回 bool 表示是否成功
func (acc *BankAccount) Withdraw(amount float64) bool {
	// 检查余额是否足够
	if acc.Balance >= amount {
		acc.Balance -= amount
		return true  // 取款成功
	}
	return false  // 余额不足
}

// Address 表示地址信息（嵌套结构体）
type Address struct {
	City   string
	Street string
}

// Employee 表示员工信息
// 包含另一个结构体作为字段（嵌套）
type Employee struct {
	Name    string
	Age     int
	Address Address  // 嵌套的结构体
}

// FullInfo 返回员工的完整信息
func (e Employee) FullInfo() string {
	// 访问嵌套结构体的字段：e.Address.City
	// fmt.Sprintf() 格式化字符串并返回，不打印
	return fmt.Sprintf("%s, %d岁, 住在%s %s",
		e.Name, e.Age, e.Address.City, e.Address.Street)
}

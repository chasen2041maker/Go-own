package main

import (
	"fmt"
	"math"
)

type Book struct {
	Title  string
	Author string
	Pages  int
	Price  float64
}

type Point struct {
	X int
	Y int
}

func (p Point) Distance() float64 {
	return math.Sqrt(math.Pow(float64(p.X), 2) + math.Pow(float64(p.Y), 2))
}

type BankAccount struct {
	AccountNumber string
	Balance       float64
}

func (acc *BankAccount) Deposit(amount float64) {
	acc.Balance += amount
}

func (acc *BankAccount) Withdraw(amount float64) bool {
	if acc.Balance >= amount {
		acc.Balance -= amount
		return true
	}
	return false
}

type Address struct {
	City   string
	Street string
}

type Employee struct {
	Name    string
	Age     int
	Address Address
}

func (e Employee) FullInfo() string {
	return fmt.Sprintf("%s, %d岁, 住在%s %s",
		e.Name, e.Age, e.Address.City, e.Address.Street)
}

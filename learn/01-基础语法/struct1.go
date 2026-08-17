package main

import "fmt"

type User struct {
	Name  string
	Email string
	age   int
}

func (u User) Greet() string {
	return fmt.Sprintf("你好，我是%s，邮箱是%s", u.Name, u.Email)
}

func (u *User) Birthday() {
	u.age++
}

func main() {
	user := User{
		Name:  "runnoob",
		Email: "runnoob@example.com",
		age:   25,
	}

	fmt.Println(user.Greet())
	fmt.Printf("生日前年龄：%d\n", user.age)

	user.Birthday()
	fmt.Printf("生日后年龄：%d\n", user.age)
}

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 初始化一些测试数据
	users[1] = User{ID: 1, Name: "Alice", Age: 25}
	users[2] = User{ID: 2, Name: "Bob", Age: 30}
	nextID = 3

	// 设置路由
	task4_setupRoutes()

	// 启动服务器
	addr := ":8080"
	fmt.Printf("服务器启动在 http://localhost%s\n", addr)
	fmt.Println("\n可用的端点：")
	fmt.Println("  GET  /hello        - Hello World")
	fmt.Println("  GET  /users        - 获取所有用户")
	fmt.Println("  POST /users        - 创建新用户")
	fmt.Println("\n测试命令：")
	fmt.Println("  curl http://localhost:8080/hello")
	fmt.Println("  curl http://localhost:8080/users")
	fmt.Println("  curl -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{\"name\":\"Charlie\",\"age\":28}'")
	fmt.Println()

	// ListenAndServe 启动 HTTP 服务器
	// 第一个参数是监听地址，第二个参数是路由器（nil 使用默认）
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

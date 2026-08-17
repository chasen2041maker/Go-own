package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// User 用户结构体
type User struct {
	ID   int    `json:"id"`   // json tag 指定 JSON 字段名
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// 内存数据存储
var (
	users  = make(map[int]User) // 存储用户数据
	nextID = 1                  // 下一个用户 ID
	mu     sync.RWMutex         // 读写锁，保护并发访问
)

// task1_helloHandler 简单的 Hello World 处理器
func task1_helloHandler(w http.ResponseWriter, r *http.Request) {
	// w: ResponseWriter 用于写入响应
	// r: Request 包含请求信息

	// 写入响应状态码（可选，默认 200）
	w.WriteHeader(http.StatusOK)

	// 写入响应体
	fmt.Fprintf(w, "Hello, World!")
}

// task2_getUsersHandler 获取所有用户（JSON 响应）
func task2_getUsersHandler(w http.ResponseWriter, r *http.Request) {
	// 读锁：允许多个 goroutine 同时读取
	mu.RLock()
	defer mu.RUnlock() // 函数结束时释放锁

	// 将 map 转换为 slice（JSON 更友好）
	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	// 设置响应头：告诉客户端这是 JSON
	w.Header().Set("Content-Type", "application/json")

	// 将数据编码为 JSON 并写入响应
	// json.NewEncoder(w).Encode() 会自动设置状态码 200
	if err := json.NewEncoder(w).Encode(userList); err != nil {
		// 如果编码失败，返回 500 错误
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// task3_createUserHandler 创建新用户（处理 POST 请求）
func task3_createUserHandler(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodPost {
		// 返回 405 Method Not Allowed
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析 JSON 请求体
	var user User
	// json.NewDecoder(r.Body) 从请求体读取 JSON
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		// 返回 400 Bad Request
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 写锁：独占访问，其他 goroutine 无法读写
	mu.Lock()
	user.ID = nextID // 分配 ID
	nextID++
	users[user.ID] = user // 保存到 map
	mu.Unlock()

	// 返回创建的用户（201 Created）
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// loggingMiddleware 日志中间件
// 中间件是一个包装函数，在实际处理器前后执行代码
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// 返回一个新的处理器函数
	return func(w http.ResponseWriter, r *http.Request) {
		// 前置处理：记录请求
		log.Printf("[%s] %s", r.Method, r.URL.Path)

		// 调用下一个处理器
		next(w, r)

		// 后置处理：可以在这里记录响应时间等
	}
}

// task4_setupRoutes 设置路由和中间件
func task4_setupRoutes() {
	// 注册路由
	// http.HandleFunc 将 URL 路径映射到处理器函数

	// 任务 1：简单的 Hello World
	http.HandleFunc("/hello", task1_helloHandler)

	// 任务 2 和 3：用户 API（带日志中间件）
	http.HandleFunc("/users", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// 根据 HTTP 方法路由到不同的处理器
		switch r.Method {
		case http.MethodGet:
			task2_getUsersHandler(w, r)
		case http.MethodPost:
			task3_createUserHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

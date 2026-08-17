# HTTP 服务器练习任务

完成以下 4 个任务，掌握 HTTP 服务器、路由、JSON 处理和中间件。

---

## 任务 1：Hello World 端点

**需求**：
创建一个处理器函数 `task1_helloHandler(w http.ResponseWriter, r *http.Request)`：
- 返回文本 "Hello, World!"
- 在 `task4_setupRoutes()` 中将它注册到 `/hello` 路径

**测试**：
```bash
curl http://localhost:8080/hello
```

**预期输出**：
```
Hello, World!
```

**提示**：
- `http.ResponseWriter` 用于写入响应
- `*http.Request` 包含请求信息
- 使用 `fmt.Fprintf(w, "...")` 写入文本

---

## 任务 2：获取所有用户（JSON）

**需求**：
创建一个处理器函数 `task2_getUsersHandler(w http.ResponseWriter, r *http.Request)`：
- 从全局 `users` map 读取所有用户
- 使用 `sync.RWMutex` 的读锁保护并发访问
- 将用户列表转换为 slice
- 返回 JSON 响应

定义 `User` 结构体：
```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}
```

全局变量：
```go
var (
    users  = make(map[int]User)
    nextID = 1
    mu     sync.RWMutex
)
```

**测试**：
```bash
curl http://localhost:8080/users
```

**预期输出**：
```json
[{"id":1,"name":"Alice","age":25},{"id":2,"name":"Bob","age":30}]
```

**提示**：
- `w.Header().Set("Content-Type", "application/json")` 设置响应类型
- `json.NewEncoder(w).Encode(data)` 将数据编码为 JSON
- `mu.RLock()` 和 `defer mu.RUnlock()` 保护读操作

---

## 任务 3：创建用户（POST）

**需求**：
创建一个处理器函数 `task3_createUserHandler(w http.ResponseWriter, r *http.Request)`：
- 检查请求方法是否为 POST
- 从请求体解析 JSON 数据到 User 结构体
- 分配新的 ID
- 保存到 users map（使用写锁）
- 返回创建的用户（状态码 201）

**测试**：
```bash
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Charlie","age":28}'
```

**预期输出**：
```json
{"id":3,"name":"Charlie","age":28}
```

**提示**：
- `r.Method` 获取 HTTP 方法
- `json.NewDecoder(r.Body).Decode(&user)` 解析 JSON
- `mu.Lock()` 和 `defer mu.Unlock()` 保护写操作
- `w.WriteHeader(http.StatusCreated)` 设置 201 状态码
- `http.Error(w, msg, code)` 返回错误响应

---

## 任务 4：中间件和路由

**需求**：
1. 创建一个日志中间件 `loggingMiddleware(next http.HandlerFunc) http.HandlerFunc`：
   - 记录每个请求的方法和路径
   - 调用下一个处理器

2. 创建 `task4_setupRoutes()` 函数：
   - 注册 `/hello` 路由
   - 注册 `/users` 路由（包装日志中间件）
   - 根据 HTTP 方法路由：GET → task2, POST → task3

3. 在 `main()` 中：
   - 初始化测试数据
   - 调用 `task4_setupRoutes()`
   - 启动服务器在 `:8080`

**测试**：
启动服务器后，测试所有端点，观察控制台日志。

**预期日志**：
```
[GET] /users
[POST] /users
```

**提示**：
- 中间件模式：`func(next Handler) Handler`
- `http.HandleFunc(path, handler)` 注册路由
- `http.ListenAndServe(":8080", nil)` 启动服务器
- `log.Printf()` 打印日志

---

## 如何完成

1. 在 `exercise.go` 文件中定义结构体、全局变量和处理器函数
2. 在 `main.go` 中启动服务器
3. 运行 `go run .`
4. 在另一个终端用 curl 测试
5. 遇到困难查看 `answers/`

## 测试工具

- **curl**：命令行 HTTP 客户端
- **Postman**：图形化 API 测试工具
- **浏览器**：访问 GET 端点

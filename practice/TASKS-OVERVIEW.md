# Practice 练习任务总览

所有练习目录都已配备任务列表和详细注释的参考答案。

---

## 📚 已完成的练习目录

| 目录 | 主题 | 任务数 | 状态 |
|------|------|--------|------|
| `03-variables` | 变量声明和类型 | 4 | ✅ 完成 |
| `04-constants` | 常量和 iota | 4 | ✅ 完成 |
| `06-control-flow` | 控制流 | 4 | ✅ 完成 |
| `07-functions` | 函数和 defer | 4 | ✅ 完成 |
| `08-struct-methods` | 结构体和方法 | 4 | ✅ 完成 |
| `09-interfaces` | 接口和多态 | 4 | ✅ 完成 |
| `10-worker-pool` | 并发编程 | 4 | ✅ 完成 |
| `11-http-server` | HTTP 服务器 | 4 | ✅ 完成 |

**总计：8 个目录，32 个练习任务**

---

## 📂 每个目录的结构

```
03-variables/
├── main.go           # 原有的示例代码
├── tasks.md          # 📋 任务列表（你要完成的）
└── answers/          # 📖 参考答案（卡住时查看）
    ├── exercise.go   # 带详细注释的答案代码
    └── main.go       # 测试入口
```

---

## 🎯 如何使用

### 第一步：阅读任务

```bash
cd practice/03-variables
cat tasks.md  # 或用编辑器打开
```

### 第二步：自己动手写

在当前目录创建 `exercise.go`，编写函数完成任务：

```go
package main

import "fmt"

func declareVariables() {
    // 你的代码...
}

// 其他函数...
```

### 第三步：测试你的代码

修改 `main.go` 调用你写的函数，然后运行：

```bash
go run .
```

### 第四步：查看参考答案

如果遇到困难，查看答案目录：

```bash
cat answers/exercise.go  # 查看详细注释的答案
cd answers && go run .   # 运行参考答案看预期输出
```

---

## 📖 学习路线建议

### 初级（1-3周）
1. ✅ **03-variables** - 变量声明、类型转换
2. ✅ **04-constants** - 常量、iota 枚举
3. ✅ **06-control-flow** - if/for/switch、循环控制
4. ✅ **07-functions** - 函数参数、返回值、defer

### 中级（2-4周）
5. ✅ **08-struct-methods** - 结构体、方法、接收者
6. ✅ **09-interfaces** - 接口、多态、类型断言

### 高级（3-5周）
7. ✅ **10-worker-pool** - Goroutine、Channel、并发模式
8. ✅ **11-http-server** - HTTP API、JSON、中间件

---

## 💡 学习建议

### ✅ 推荐做法
- **先自己写**，实在不会再看答案
- **理解注释**，不只是复制代码
- **修改测试**，尝试不同的输入输出
- **一个一个来**，扎实比速度重要

### ❌ 避免做法
- 直接复制答案而不理解
- 跳过基础直接做高级题
- 遇到错误就放弃
- 不运行代码只看不练

---

## 🔍 快速测试所有答案

验证所有参考答案都能正常运行：

```bash
# 从项目根目录运行
cd practice

# 测试各个练习
cd 03-variables/answers && go run . && cd ../..
cd 04-constants/answers && go run . && cd ../..
cd 06-control-flow/answers && go run . && cd ../..
cd 07-functions/answers && go run . && cd ../..
cd 08-struct-methods/answers && go run . && cd ../..
cd 09-interfaces/answers && go run . && cd ../..
cd 10-worker-pool/answers && go run . && cd ../..
```

**注意**：`11-http-server` 需要单独测试（会启动 HTTP 服务器）

---

## 📝 每个目录的任务概览

### 03-variables（变量）
1. 变量声明的三种方式
2. 零值的理解
3. 类型转换计算
4. 多变量声明与交换

### 04-constants（常量）
1. 基础常量声明
2. iota 枚举（星期）
3. iota 跳过值（权限位）
4. 常量表达式计算（存储单位）

### 06-control-flow（控制流）
1. 成绩等级判断（增强版）
2. 九九乘法表
3. 找出第一个能被 7 整除的数
4. 跳过偶数，只打印奇数

### 07-functions（函数）
1. 多返回值 - 除法函数
2. 可变参数 - 求和函数
3. 命名返回值 - 矩形面积和周长
4. defer 延迟执行

### 08-struct-methods（结构体）
1. 定义和初始化结构体
2. 值接收者方法
3. 指针接收者方法
4. 嵌套结构体

### 09-interfaces（接口）
1. 定义和实现接口
2. 接口作为参数
3. 类型断言和 type switch
4. 空接口

### 10-worker-pool（并发）
1. 基础 Goroutine
2. Channel 通信
3. 缓冲 Channel
4. Worker Pool

### 11-http-server（HTTP）
1. Hello World 端点
2. 获取所有用户（JSON）
3. 创建用户（POST）
4. 中间件和路由

---

## 🚀 完成练习后

完成这些练习后，你可以：
1. 进入 `projects/` 目录开始真实项目
2. 从 [01-task-api](../projects/01-task-api/README.md) 开始
3. 应用你在 practice 中学到的所有知识

---

## ❓ 遇到问题？

1. **语法错误**：对比答案，查看注释
2. **不理解概念**：阅读答案中的详细注释
3. **想不出实现**：先看任务提示，再看答案框架
4. **代码能跑但不理解为什么**：这是最好的学习机会！逐行研究答案注释

---

**开始你的 Go 学习之旅吧！🎉**

从 `03-variables` 开始，一步步向上攀登！

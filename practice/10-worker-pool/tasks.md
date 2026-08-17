# 并发编程练习任务

完成以下 4 个任务，掌握 goroutine、channel、WaitGroup 和 worker pool。

---

## 任务 1：基础 Goroutine

**需求**：
创建一个函数 `task1_basicGoroutine()`：
- 启动 3 个 goroutine
- 每个 goroutine 休眠不同时间（100ms, 200ms, 150ms）
- 每个 goroutine 完成后打印 "Goroutine X 完成"
- 使用 `sync.WaitGroup` 等待所有 goroutine 完成
- 最后打印 "所有 goroutine 完成"

**预期输出**（顺序可能不同）：
```
Goroutine 1 完成
Goroutine 3 完成
Goroutine 2 完成
所有 goroutine 完成
```

**提示**：
- `go func() { ... }()` 创建并启动 goroutine
- `sync.WaitGroup` 用于等待多个 goroutine 完成
- `wg.Add(n)` 增加计数，`wg.Done()` 减少计数，`wg.Wait()` 等待归零
- `time.Sleep(100 * time.Millisecond)` 休眠

---

## 任务 2：Channel 通信

**需求**：
创建一个函数 `task2_channelCommunication()`：
- 创建一个字符串 channel
- 启动一个 goroutine，向 channel 发送 3 条消息："Hello", "from", "goroutine"
- 发送完后关闭 channel
- 主 goroutine 使用 `range` 接收并打印所有消息

**预期输出**：
```
收到: Hello
收到: from
收到: goroutine
```

**提示**：
- `make(chan string)` 创建无缓冲 channel
- `ch <- value` 发送数据
- `value := <-ch` 接收数据
- `close(ch)` 关闭 channel
- `for msg := range ch` 持续接收直到 channel 关闭

---

## 任务 3：缓冲 Channel

**需求**：
创建一个函数 `task3_bufferedChannel()`：
- 创建一个缓冲大小为 3 的整数 channel
- 连续发送 3 个数字（1, 2, 3）
- 打印 "发送了 3 个数字"
- 关闭 channel
- 使用 range 接收并打印所有数字

**预期输出**：
```
发送了 3 个数字
收到: 1
收到: 2
收到: 3
```

**提示**：
- `make(chan int, 3)` 创建缓冲大小为 3 的 channel
- 缓冲 channel 允许在没有接收者的情况下发送数据（直到缓冲区满）
- 无缓冲 channel：发送和接收必须同时准备好

---

## 任务 4：Worker Pool

**需求**：
实现一个 worker pool 模式：
- 创建 3 个 worker goroutine
- 发送 5 个任务（数字 1-5）
- 每个 worker 从 jobs channel 接收任务
- 处理任务：将数字乘以 2（模拟耗时操作，休眠 100ms）
- 将结果发送到 results channel
- 主 goroutine 收集并打印所有结果

**预期输出**（worker 顺序可能不同）：
```
发送任务...
Worker 1 开始处理任务 1
Worker 2 开始处理任务 2
Worker 3 开始处理任务 3
Worker 1 完成任务 1, 结果=2
Worker 1 开始处理任务 4
...
收集结果:
结果: 2
结果: 4
...
```

**提示**：
- Worker pool 是并发编程的常见模式
- 需要两个 channel：jobs（发送任务）和 results（接收结果）
- 使用 WaitGroup 等待所有 worker 完成
- 所有 worker 完成后关闭 results channel
- Worker 函数签名：`func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup)`

---

## 如何完成

1. 在 `exercise.go` 文件中编写函数
2. 在 `main()` 中调用测试
3. 运行 `go run .`
4. 遇到困难查看 `answers/`

## 重要概念

- **Goroutine**：轻量级线程，由 Go 运行时管理
- **Channel**：goroutine 之间通信的管道
- **WaitGroup**：等待一组 goroutine 完成
- **Worker Pool**：固定数量的 worker 处理任务队列

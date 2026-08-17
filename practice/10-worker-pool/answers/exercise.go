package main

import (
	"fmt"
	"sync"
	"time"
)

// task1_basicGoroutine 演示基础的 goroutine
func task1_basicGoroutine() {
	// 使用 WaitGroup 等待 goroutine 完成
	// WaitGroup 是一个计数器，用于等待一组 goroutine 完成
	var wg sync.WaitGroup

	// Add(n) 增加计数器 n
	// 表示有 3 个 goroutine 需要等待
	wg.Add(3)

	// 启动第一个 goroutine
	// go 关键字创建一个新的 goroutine（轻量级线程）
	go func() {
		// defer 确保函数退出时一定会调用 Done()
		defer wg.Done() // Done() 减少计数器 1
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Goroutine 1 完成")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
		fmt.Println("Goroutine 2 完成")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(150 * time.Millisecond)
		fmt.Println("Goroutine 3 完成")
	}()

	// Wait() 阻塞，直到计数器归零
	// 即等待所有 goroutine 调用 Done()
	wg.Wait()
	fmt.Println("所有 goroutine 完成")
}

// task2_channelCommunication 演示 channel 通信
func task2_channelCommunication() {
	// 创建一个无缓冲 channel
	// 无缓冲 channel：发送和接收必须同时准备好，否则会阻塞
	messages := make(chan string)

	// 发送者 goroutine
	go func() {
		// 向 channel 发送数据
		// <- 是发送/接收操作符
		messages <- "Hello"
		messages <- "from"
		messages <- "goroutine"
		// 发送完毕后关闭 channel
		// close() 表示不会再发送数据
		close(messages)
	}()

	// 接收者（主 goroutine）
	// range 会持续从 channel 接收，直到 channel 关闭
	for msg := range messages {
		fmt.Println("收到:", msg)
	}
}

// task3_bufferedChannel 演示缓冲 channel
func task3_bufferedChannel() {
	// 创建一个缓冲大小为 3 的 channel
	// 缓冲 channel：只要缓冲区未满，发送不会阻塞
	numbers := make(chan int, 3)

	// 因为有缓冲，可以连续发送 3 个值而不阻塞
	numbers <- 1
	numbers <- 2
	numbers <- 3
	fmt.Println("发送了 3 个数字")

	// 如果再发送第 4 个，会阻塞（因为缓冲区满了）
	// numbers <- 4  // 这会导致死锁

	// 关闭 channel
	close(numbers)

	// 接收所有数据
	for num := range numbers {
		fmt.Printf("收到: %d\n", num)
	}
}

// worker 是 worker pool 中的工作协程
// 从 jobs channel 接收任务，处理后将结果发送到 results channel
func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	// defer 确保工作完成后通知 WaitGroup
	defer wg.Done()

	// 从 jobs channel 持续接收任务
	// 当 jobs 关闭且为空时，range 循环结束
	for job := range jobs {
		fmt.Printf("Worker %d 开始处理任务 %d\n", id, job)
		time.Sleep(100 * time.Millisecond) // 模拟耗时操作
		result := job * 2                   // 简单的处理：乘以 2
		results <- result                   // 发送结果
		fmt.Printf("Worker %d 完成任务 %d, 结果=%d\n", id, job, result)
	}
}

// task4_workerPool 演示 worker pool 模式
func task4_workerPool() {
	const numWorkers = 3 // 工作协程数量
	const numJobs = 5    // 任务数量

	// 创建两个 channel
	// jobs: 发送任务给 worker
	// results: worker 发送结果回来
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// WaitGroup 用于等待所有 worker 完成
	var wg sync.WaitGroup

	// 启动 worker pool
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
	}

	// 发送任务
	fmt.Println("发送任务...")
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	// 关闭 jobs channel，告诉 worker 没有更多任务了
	close(jobs)

	// 在单独的 goroutine 中等待所有 worker 完成
	// 然后关闭 results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集所有结果
	fmt.Println("\n收集结果:")
	for result := range results {
		fmt.Printf("结果: %d\n", result)
	}
}

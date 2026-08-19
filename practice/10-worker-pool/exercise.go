package main

import (
	"fmt"
	"sync"
	"time"
)

func task1_basicGoroutine() {
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
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

	wg.Wait()
	fmt.Println("所有 goroutine 完成")
}

func task2_channelCommunication() {
	messages := make(chan string)
	go func() {
		messages <- "hello"
		messages <- "from"
		messages <- "goroutine"

		close(messages)
	}()

	for msg := range messages {
		fmt.Println("收到:", msg)
	}
}

func task3_bufferedChannel() {
	numbers := make(chan int, 3)

	numbers <- 1
	numbers <- 2
	numbers <- 3
	fmt.Println("发送了 3 个数字")

	close(numbers)

	for num := range numbers {
		fmt.Printf("收到: %d\n", num)
	}
}

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		fmt.Printf("Worker %d 开始处理任务 %d\n", id, job)
		time.Sleep(100 * time.Millisecond) // 模拟耗时操作
		result := job * 2                  // 简单的处理：乘以 2
		results <- result                  // 发送结果
		fmt.Printf("Worker %d 完成任务 %d, 结果=%d\n", id, job, result)
	}
}

func task4_workerPool() {
	const numWorkers = 3
	const numJobs = 5

	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	var wg sync.WaitGroup

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

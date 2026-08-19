package main

import (
	"fmt"
	"sync"
)

func main() {
	const numJobs = 10
	const numWorkers = 3

	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)
	var wg sync.WaitGroup

	fmt.Println("=== RUNOOB Worker Pool 示例 ===")

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	fmt.Println("结果汇总：")
	for result := range results {
		fmt.Printf("%d ", result)
	}
	fmt.Println("\n所有任务处理完成")

	fmt.Println("=== 任务 1：基础 Goroutine ===")
	task1_basicGoroutine()

	fmt.Println("\n=== 任务 2：Channel 通信 ===")
	task2_channelCommunication()

	fmt.Println("\n=== 任务 3：缓冲 Channel ===")
	task3_bufferedChannel()

	fmt.Println("\n=== 任务 4：Worker Pool ===")
	task4_workerPool()
}

package main

import "fmt"

func main() {
	fmt.Println("=== 任务 1：基础 Goroutine ===")
	task1_basicGoroutine()

	fmt.Println("\n=== 任务 2：Channel 通信 ===")
	task2_channelCommunication()

	fmt.Println("\n=== 任务 3：缓冲 Channel ===")
	task3_bufferedChannel()

	fmt.Println("\n=== 任务 4：Worker Pool ===")
	task4_workerPool()
}

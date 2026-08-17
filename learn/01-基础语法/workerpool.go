package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		fmt.Printf("[Worker %d] 处理任务%d\n", id, job)
		time.Sleep(200 * time.Millisecond)
		results <- job * job
	}
}

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

}

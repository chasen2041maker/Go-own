package main

import "fmt"

func main() {
	score := 85
	if score >= 90 {
		fmt.Println("A")
	} else if score >= 60 {
		fmt.Println("B")
	} else {
		fmt.Println("C")
	}

	fmt.Print("计数： ")
	for i := 0; i < 5; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	sum := 0
	n := 1
	for n <= 10 {
		sum += n
		n++
	}
	fmt.Println("1到10 的和：", sum)

	fruits := []string{"apple", "banana", "cherry"}
	for index, fruit := range fruits {
		fmt.Printf("索引 %d: %s\n", index, fruit)
	}

	day := "周一"
	switch day {
	case "周一":
		fmt.Println("新的一周开始了")
	case "周五":
		fmt.Println("周末快到了")
	default:
		fmt.Println("普通的一天")
	}
}

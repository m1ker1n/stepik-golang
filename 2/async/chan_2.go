package main

import (
	"fmt"
)

func main() {
	in := make(chan int, 1)

	go func(out chan<- int) {
		go func(out chan<- int) {
			for i := 0; i <= 10; i++ {
				fmt.Println("before", i)
				out <- i
				fmt.Println("after", i)
			}
			close(out)
			out <- 12
			fmt.Println("generator finish")
		}(in)
	}(in)

	for i := range in {
		// i, isOpened := <-in
		// if !isOpened {
		// 	break
		// }
		fmt.Println("\tget", i)
	}

	// fmt.Scanln()
}

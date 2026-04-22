package main

import "fmt"

func pipe(f, g func(int) int) func(int) int {
	return func(x int) int { return g(f(x)) }
}

func subtract(x, y int) int { return x - y }

func main() {
	sub4 := func(x int) int { return subtract(x, 4) }
	double := func(x int) int { return x * 2 }
	sub4ThenDouble := pipe(sub4, double)
	fmt.Printf("Result: %d\n", sub4ThenDouble(11))
}

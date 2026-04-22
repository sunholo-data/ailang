package main

import "fmt"

func foldl(f func(int, int) int, acc int, xs []int) int {
	if len(xs) == 0 {
		return acc
	}
	return foldl(f, f(acc, xs[0]), xs[1:])
}

func main() {
	nums := []int{2, 4, 6, 8, 10}
	sum := foldl(func(a, b int) int { return a + b }, 0, nums)
	product := foldl(func(a, b int) int { return a * b }, 1, nums)
	maxNums := []int{7, 2, 8, 3, 11, 5, 1}
	max := foldl(func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}, maxNums[0], maxNums[1:])

	fmt.Printf("Sum: %d\n", sum)
	fmt.Printf("Product: %d\n", product)
	fmt.Printf("Max: %d\n", max)
}

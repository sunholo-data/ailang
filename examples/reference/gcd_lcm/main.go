package main

import "fmt"

func gcd(a, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a%b)
}

func lcm(a, b int) int {
	return (a * b) / gcd(a, b)
}

func main() {
	fmt.Println(gcd(48, 18))
	fmt.Println(lcm(48, 18))
}

package main

import "fmt"

type Tree struct {
	value       int
	left, right *Tree
}

func treeSum(t *Tree) int {
	if t == nil {
		return 0
	}
	return t.value + treeSum(t.left) + treeSum(t.right)
}

func main() {
	tree := &Tree{
		value: 5,
		left: &Tree{
			value: 3,
			left:  &Tree{value: 1},
			right: &Tree{value: 4},
		},
		right: &Tree{
			value: 8,
			right: &Tree{value: 10},
		},
	}
	fmt.Println(treeSum(tree))
}

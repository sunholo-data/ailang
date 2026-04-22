package main

import "fmt"

var graph = map[int][]int{
	1: {2, 3},
	2: {4},
	3: {4, 5},
	4: {},
	5: {},
}

func bfs(graph map[int][]int, start int) []int {
	visited := []int{}
	inVisited := map[int]bool{}
	queue := []int{start}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if inVisited[node] {
			continue
		}
		inVisited[node] = true
		visited = append(visited, node)
		for _, neighbor := range graph[node] {
			if !inVisited[neighbor] {
				queue = append(queue, neighbor)
			}
		}
	}
	return visited
}

func main() {
	for _, node := range bfs(graph, 1) {
		fmt.Println(node)
	}
}

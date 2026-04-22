const graph = {
  1: [2, 3],
  2: [4],
  3: [4, 5],
  4: [],
  5: [],
};

function bfs(graph, start) {
  const visited = [];
  const queue = [start];
  while (queue.length > 0) {
    const node = queue.shift();
    if (!visited.includes(node)) {
      visited.push(node);
      for (const neighbor of graph[node]) {
        if (!visited.includes(neighbor)) {
          queue.push(neighbor);
        }
      }
    }
  }
  return visited;
}

const result = bfs(graph, 1);
for (const node of result) {
  console.log(node);
}

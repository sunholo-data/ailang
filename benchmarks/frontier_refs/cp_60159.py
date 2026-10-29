N, Q = map(int, input().split())

labyrinth = []
for _ in range(N):
    labyrinth.append(input().strip())

dir_map = {
    'U': (-1, 0),
    'D': (1, 0),
    'L': (0, -1),
    'R': (0, 1)
}

relative_dirs = {
    'U': ['R', 'U', 'L', 'D'],
    'R': ['D', 'R', 'U', 'L'],
    'D': ['L', 'D', 'R', 'U'],
    'L': ['U', 'L', 'D', 'R']
}

for _ in range(Q):
    parts = input().split()
    r, c, d = int(parts[0]) - 1, int(parts[1]) - 1, parts[2]

    visited = set()
    moves = 0
    result = -1

    while True:
        state = (r, c, d)
        if state in visited:
            break
        visited.add(state)

        found = False
        for next_dir in relative_dirs[d]:
            dr, dc = dir_map[next_dir]
            nr, nc = r + dr, c + dc

            if nr < 0 or nr >= N or nc < 0 or nc >= N:
                moves += 1
                result = moves
                found = True
                break
            elif labyrinth[nr][nc] == '.':
                r, c, d = nr, nc, next_dir
                moves += 1
                found = True
                break

        if result != -1 or not found:
            break

    print(result)

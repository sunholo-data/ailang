from collections import deque

def solve():
    n = int(input())
    bricks = []
    for _ in range(n):
        l, r, h = map(int, input().split())
        bricks.append((l, r, h))

    # Track occupied regions as (left, right, bottom, top)
    occupied = []

    # Simulate each brick falling
    for l, r, h in bricks:
        # Find max height in range [l, r)
        max_h = 0
        for ox_l, ox_r, oy_b, oy_t in occupied:
            if ox_r > l and ox_l < r:
                max_h = max(max_h, oy_t)

        # Brick lands at max_h and extends to max_h + h
        occupied.append((l, r, max_h, max_h + h))

    # Build discretized grid
    x_coords = set()
    y_coords = set([0])
    max_y = 0
    for ox_l, ox_r, oy_b, oy_t in occupied:
        x_coords.add(ox_l)
        x_coords.add(ox_r)
        y_coords.add(oy_b)
        y_coords.add(oy_t)
        max_y = max(max_y, oy_t)

    # Add extra coordinates for outside air
    y_coords.add(max_y + 1)

    x_coords = sorted(x_coords)
    y_coords = sorted(y_coords)

    # Create grid to mark occupied cells
    grid = [[False] * (len(x_coords) - 1) for _ in range(len(y_coords) - 1)]

    # Mark occupied cells
    for ox_l, ox_r, oy_b, oy_t in occupied:
        for i in range(len(x_coords) - 1):
            if x_coords[i] >= ox_l and x_coords[i + 1] <= ox_r:
                for j in range(len(y_coords) - 1):
                    if y_coords[j] >= oy_b and y_coords[j + 1] <= oy_t:
                        grid[j][i] = True

    # Flood fill from outside boundaries
    visited = [[False] * (len(x_coords) - 1) for _ in range(len(y_coords) - 1)]
    queue = deque()

    # Top boundary
    for i in range(len(x_coords) - 1):
        if not grid[len(y_coords) - 2][i]:
            queue.append((len(y_coords) - 2, i))
            visited[len(y_coords) - 2][i] = True

    # Left and right boundaries (all rows)
    for j in range(len(y_coords) - 1):
        if not grid[j][0]:
            queue.append((j, 0))
            visited[j][0] = True
        if not grid[j][len(x_coords) - 2]:
            queue.append((j, len(x_coords) - 2))
            visited[j][len(x_coords) - 2] = True

    # BFS to mark all reachable empty cells from outside
    while queue:
        j, i = queue.popleft()
        for dj, di in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
            nj, ni = j + dj, i + di
            if 0 <= nj < len(y_coords) - 1 and 0 <= ni < len(x_coords) - 1:
                if not visited[nj][ni] and not grid[nj][ni]:
                    visited[nj][ni] = True
                    queue.append((nj, ni))

    # Count connected components of enclosed empty cells
    flats = 0
    visited_flat = [[False] * (len(x_coords) - 1) for _ in range(len(y_coords) - 1)]

    for j in range(len(y_coords) - 1):
        for i in range(len(x_coords) - 1):
            if not grid[j][i] and not visited[j][i] and not visited_flat[j][i]:
                # Found a new flat, count it
                flats += 1
                # DFS to mark all cells in this component
                stack = [(j, i)]
                while stack:
                    cj, ci = stack.pop()
                    if 0 <= cj < len(y_coords) - 1 and 0 <= ci < len(x_coords) - 1:
                        if not visited_flat[cj][ci] and not grid[cj][ci] and not visited[cj][ci]:
                            visited_flat[cj][ci] = True
                            for dcj, dci in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
                                stack.append((cj + dcj, ci + dci))

    print(flats)

solve()

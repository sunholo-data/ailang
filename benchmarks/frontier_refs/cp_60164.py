def solve():
    n, m = map(int, input().split())

    rectangles = []
    for _ in range(n):
        x1, x2, y1, y2 = map(int, input().split())
        rectangles.append((x1, x2, y1, y2))

    # Coordinate compression
    xs = set()
    ys = set()
    for x1, x2, y1, y2 in rectangles:
        xs.add(x1)
        xs.add(x2)
        ys.add(y1)
        ys.add(y2)

    xs = sorted(xs)
    ys = sorted(ys)

    # Map coordinates to indices
    x_idx = {x: i for i, x in enumerate(xs)}
    y_idx = {y: i for i, y in enumerate(ys)}

    # Create a 2D difference array
    diff = [[0] * (len(ys) + 1) for _ in range(len(xs) + 1)]

    # For each rectangle, apply difference array update
    for x1, x2, y1, y2 in rectangles:
        x1_idx = x_idx[x1]
        x2_idx = x_idx[x2]
        y1_idx = y_idx[y1]
        y2_idx = y_idx[y2]

        diff[x1_idx][y1_idx] += 1
        diff[x2_idx][y1_idx] -= 1
        diff[x1_idx][y2_idx] -= 1
        diff[x2_idx][y2_idx] += 1

    # Compute 2D prefix sum
    for i in range(len(xs) + 1):
        for j in range(len(ys) + 1):
            if i > 0:
                diff[i][j] += diff[i-1][j]
            if j > 0:
                diff[i][j] += diff[i][j-1]
            if i > 0 and j > 0:
                diff[i][j] -= diff[i-1][j-1]

    # Count the area for each coverage count
    area_count = {}
    for i in range(len(xs) - 1):
        for j in range(len(ys) - 1):
            count = diff[i][j]
            area = (xs[i+1] - xs[i]) * (ys[j+1] - ys[j])
            if count not in area_count:
                area_count[count] = 0
            area_count[count] += area

    # Output results
    max_c = n // m
    for c in range(1, max_c + 1):
        target = m * c
        print(area_count.get(target, 0))

solve()

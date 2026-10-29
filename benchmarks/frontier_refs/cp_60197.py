def find(parent, x):
    if parent[x] != x:
        parent[x] = find(parent, parent[x])
    return parent[x]

def union(parent, rank, x, y):
    px, py = find(parent, x), find(parent, y)
    if px == py:
        return
    if rank[px] < rank[py]:
        px, py = py, px
    parent[py] = px
    if rank[px] == rank[py]:
        rank[px] += 1

def solve(s1, s2, s3):
    # Check if S1 and S2 can have the same encryption
    if len(s1) != len(s2):
        return "NO"
    
    # Build equivalence classes using Union-Find
    parent = {}
    rank = {}
    for c in set(s1 + s2 + s3):
        parent[c] = c
        rank[c] = 0
    
    # Union characters that must map to the same output
    for i in range(len(s1)):
        union(parent, rank, s1[i], s2[i])
    
    # Check if we can make F(S1) != F(S3)
    # If lengths differ, they're automatically different
    if len(s1) != len(s3):
        return "YES"
    
    # Check if any position has characters in different equivalence classes
    for i in range(len(s1)):
        if find(parent, s1[i]) != find(parent, s3[i]):
            return "YES"
    
    # All positions have same equivalence classes, so F(S1) must equal F(S3)
    return "NO"

T = int(input())
for _ in range(T):
    s1 = input()
    s2 = input()
    s3 = input()
    print(solve(s1, s2, s3))

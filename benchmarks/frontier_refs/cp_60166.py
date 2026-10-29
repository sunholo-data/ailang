def solve(arr):
    MOD = 10**9 + 7
    n = len(arr)
    dp = [0] * (n + 1)

    for i in range(1, n + 1):
        dp[i] = 0
        product = 1
        # Try all possible last segments ending at position i
        for j in range(i, 0, -1):
            # Extend the product backwards to include arr[j-1]
            product *= arr[j-1]
            # Update dp[i] with the best solution using segment [j-1, i)
            dp[i] = max(dp[i], dp[j-1] + product)

    return dp[n] % MOD

T = int(input())
for _ in range(T):
    n = int(input())
    arr = list(map(int, input().split()))
    print(solve(arr))

# Reference: binary_strings_1e18 — count binary strings of length n with no "101",
# mod 1e9+7. DFA states = longest suffix that is a proper prefix of "101":
# S0="", S1="1", S2="10". Transitions: S0: 0->S0 1->S1 ; S1: 0->S2 1->S1 ; S2: 0->S0 1->dead
MOD = 1000000007
# transition matrix M[i][j] = number of ways state i -> state j in one step
M = [
    [1, 1, 0],  # S0: 0->S0, 1->S1
    [0, 1, 1],  # S1: 1->S1, 0->S2
    [1, 0, 0],  # S2: 0->S0
]
def matmul(a, b):
    return [[sum(a[i][k] * b[k][j] for k in range(3)) % MOD for j in range(3)] for i in range(3)]
def matpow(m, e):
    r = [[1 if i == j else 0 for j in range(3)] for i in range(3)]
    while e:
        if e & 1:
            r = matmul(r, m)
        m = matmul(m, m)
        e >>= 1
    return r
def f(n):
    p = matpow(M, n)
    return sum(p[0]) % MOD  # start S0, all states accepting

# sanity: brute force small n
def brute(n):
    return sum(1 for i in range(2**n) if "101" not in format(i, f"0{n}b"))
for n in range(1, 13):
    assert f(n) == brute(n), (n, f(n), brute(n))
print(f(10**18))
print(f(987654321987654321))

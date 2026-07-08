# Reference: stream_lcg_topk
# x_{n+1} = (48271 * x_n) mod 2147483647, x_0 = 20260708.
# Consider x_1 .. x_2000 (x_0 is NOT included).
x = 20260708
vals = []
for _ in range(2000):
    x = (48271 * x) % 2147483647
    vals.append(x)
print(vals[-1])                      # x_2000
print(sum(sorted(vals)[-10:]))       # sum of 10 largest
print(sum(vals) % 1000000007)        # checksum of all
# off-by-one trap check: including x_0 / stopping at x_4999 changes all three lines

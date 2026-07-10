# Reference: lfu_cache_trace — capacity 3, LFU eviction, LRU tie-break on last USE.
import sys
OPS = [
    ("put", 1, 10), ("put", 2, 20), ("put", 3, 30),
    ("get", 1, None),
    ("put", 4, 40),
    ("put", 5, 50),
    ("get", 3, None),
    ("get", 4, None),
    ("get", 5, None),
    ("get", 1, None),
    ("put", 4, 44),
    ("put", 6, 60),
    ("get", 5, None),
    ("get", 6, None),
    ("get", 6, None),
    ("put", 7, 70),
    ("get", 1, None),
    ("get", 4, None),
    ("put", 8, 80),
    ("get", 7, None),
]
def run(variant="ref"):
    cache = {}  # k -> [val, count, last_use]
    out = []
    for tick, (op, k, v) in enumerate(OPS, 1):
        if op == "get":
            if k in cache:
                cache[k][1] += 1; cache[k][2] = tick
                out.append(str(cache[k][0]))
            else:
                out.append("-1")
        else:
            if k in cache:
                cache[k][0] = v
                if variant != "update_not_use":
                    cache[k][1] += 1; cache[k][2] = tick
            else:
                if len(cache) == 3:
                    if variant == "pure_lru":
                        victim = min(cache, key=lambda x: cache[x][2])
                    else:
                        victim = min(cache, key=lambda x: (cache[x][1], cache[x][2]))
                    del cache[victim]
                cache[k] = [v, 0 if variant == "count_zero" else 1, tick]
    out.append("cache: " + ",".join(f"{k}={cache[k][0]}" for k in sorted(cache)))
    return "\n".join(out)

ref = run()
print(ref)
for var in ("count_zero", "update_not_use", "pure_lru"):
    d = run(var)
    status = "DIVERGES" if d != ref else "SAME (bad benchmark!)"
    print(f"{var}: {status}", file=sys.stderr)

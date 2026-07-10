# Reference: effect_txn_rollback
# Rules: wd fails if balance < amt -> restore most recent snapshot AND POP it
# (state unchanged if no snapshot). commit/restore on empty stack also FAIL (state unchanged).
# Every failure increments failures. Execution always continues.
import sys
OPS = [
    ("save",),
    ("dep", "alice", 30),
    ("save",),
    ("wd", "bob", 80),      # FAIL: 50 < 80 -> restore+pop S2
    ("wd", "alice", 90),    # 130 >= 90 -> alice 40
    ("save",),
    ("wd", "alice", 50),    # FAIL: 40 < 50 -> restore+pop S3
    ("dep", "bob", 25),     # bob 75
    ("commit",),            # pops S1
    ("restore",),           # FAIL: empty
    ("wd", "bob", 75),      # bob 0
    ("commit",),            # FAIL: empty
    ("dep", "alice", 5),    # alice 45
]
def run(variant="ref"):
    bal = {"alice": 100, "bob": 50}
    stack = []
    failures = 0
    for op in OPS:
        if op[0] == "dep":
            bal[op[1]] += op[2]
        elif op[0] == "wd":
            if bal[op[1]] >= op[2]:
                bal[op[1]] -= op[2]
            else:
                failures += 1
                if stack:
                    if variant == "peek_on_fail":
                        bal = dict(stack[-1])
                    else:
                        bal = dict(stack.pop())
        elif op[0] == "save":
            stack.append(dict(bal))
        elif op[0] == "commit":
            if stack: stack.pop()
            else: failures += 1
        elif op[0] == "restore":
            if stack: bal = dict(stack.pop())
            else:
                if variant == "empty_no_fail": pass
                else: failures += 1
    return f"alice={bal['alice']}\nbob={bal['bob']}\nfailures={failures}"

ref = run()
print(ref)
for var in ("peek_on_fail", "empty_no_fail"):
    print(f"{var}: {'DIVERGES' if run(var) != ref else 'SAME (bad!)'}", file=sys.stderr)

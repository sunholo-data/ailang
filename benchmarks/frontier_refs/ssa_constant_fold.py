# Reference: ssa_constant_fold
# Program (fixed input, given in prompt)
prog = [
    ("n", "input", None, None),
    ("m", "input", None, None),
    ("a", "lit", 7, None),
    ("a", "op", ("a", "-", 7), None),
    ("c", "op", ("n", "*", "a"), None),
    ("n", "op", ("n", "+", "c"), None),
    ("q", "op", ("m", "*", 3), None),
    ("e", "op", ("n", "*", 7), None),
    ("f", "op", ("e", "-", "e"), None),
    ("e", "op", ("e", "+", "f"), None),
    ("r", "op", ("e", "*", 1), None),
]
ret_var = "r"

# Pass 1: SSA renaming
versions = {}
ssa = []  # (ssaname, kind, payload) payload op: (lhs, op, rhs) operands are ssanames or ints
def cur(v): return f"{v}.{versions[v]}"
for (var, kind, a, _) in prog:
    if kind == "input":
        payload = "input"
    elif kind == "lit":
        payload = a
    else:
        l, op, r = a
        L = cur(l) if isinstance(l, str) else l
        R = cur(r) if isinstance(r, str) else r
        payload = (L, op, R)
    versions[var] = versions.get(var, -1) + 1
    ssa.append((cur(var), kind, payload))
ret_ssa = cur(ret_var)

# Pass 2: forward simplification
# table: name -> ("const", k) | ("copy", name) | ("opaque",)
table = {}
kept = []  # (name, textual instruction)
def resolve(operand):
    # operand is int literal or ssa name; returns int or opaque ssa name
    if isinstance(operand, int):
        return operand
    e = table[operand]
    if e[0] == "const": return e[1]
    if e[0] == "copy": return resolve(e[1])
    return operand
for (name, kind, payload) in ssa:
    if kind == "input":
        table[name] = ("opaque",)
        kept.append((name, f"{name} = input"))
        continue
    if kind == "lit":
        table[name] = ("const", payload)
        continue
    L, op, R = payload
    L = resolve(L); R = resolve(R)
    if isinstance(L, int) and isinstance(R, int):
        v = L + R if op == "+" else L - R if op == "-" else L * R
        table[name] = ("const", v); continue
    if op == "-" and L == R and not isinstance(L, int):
        table[name] = ("const", 0); continue
    if op == "*" and (L == 0 or R == 0):
        table[name] = ("const", 0); continue
    if op == "*" and R == 1:
        table[name] = ("copy", L); continue
    if op == "*" and L == 1:
        table[name] = ("copy", R); continue
    if op == "+" and R == 0:
        table[name] = ("copy", L); continue
    if op == "+" and L == 0:
        table[name] = ("copy", R); continue
    if op == "-" and R == 0:
        table[name] = ("copy", L); continue
    table[name] = ("opaque",)
    kept.append((name, f"{name} = {L} {op} {R}"))

rr = resolve(ret_ssa)
ret_line = f"return {rr}"

# Pass 3: DCE
live = set()
if not isinstance(rr, int):
    live.add(rr)
# iterate kept in reverse, keep if name live, add its name operands
out = []
for (name, text) in reversed(kept):
    if name in live:
        out.append(text)
        for tok in text.split(" = ")[1].split(" "):
            if not tok.lstrip("-").isdigit() and tok not in ("+", "-", "*", "input"):
                live.add(tok)
final = list(reversed(out)) + [ret_line]
print("\n".join(final))

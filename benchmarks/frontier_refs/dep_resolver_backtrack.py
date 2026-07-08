# Reference: dep_resolver_backtrack
# Universe A (resolvable only via backtracking off the greedy path):
#   app 1: lib [1,3], util [3,3]
#   lib 3: core [3,3] ; lib 2: core [1,2] ; lib 1: (none)
#   util 3: core [1,1]
#   core 3, 2, 1: (none)
# Greedy-highest picks lib=3 -> core=3; util 3 needs core=1 -> must backtrack
# into earlier choices (lib), landing on lib=2, core=1, util=3.
# Universe B: genuine conflict.
#   app 1: x [1,1], y [1,1] ; x 1: core [1,1] ; y 1: core [2,2] ; core 1, 2
import sys

UNIVERSE_A = {
    ("app", 1): [("lib", 1, 3), ("util", 3, 3)],
    ("lib", 3): [("core", 3, 3)],
    ("lib", 2): [("core", 1, 2)],
    ("lib", 1): [],
    ("util", 3): [("core", 1, 1)],
    ("core", 3): [], ("core", 2): [], ("core", 1): [],
}
UNIVERSE_B = {
    ("app", 1): [("x", 1, 1), ("y", 1, 1)],
    ("x", 1): [("core", 1, 1)],
    ("y", 1): [("core", 2, 2)],
    ("core", 1): [], ("core", 2): [],
}

def versions_of(universe, name):
    return sorted([v for (n, v) in universe if n == name], reverse=True)

def resolve(universe, root_name, root_version, greedy=False):
    # DFS: process requirements depth-first, in listed order, versions high->low.
    # assignment: name -> version. Returns dict or None.
    def go(assign, worklist):
        if not worklist:
            return assign
        (name, lo, hi), rest = worklist[0], worklist[1:]
        if name in assign:
            if lo <= assign[name] <= hi:
                return go(assign, rest)
            return None
        candidates = [v for v in versions_of(universe, name) if lo <= v <= hi]
        if greedy:
            candidates = candidates[:1]
        for v in candidates:
            new_assign = dict(assign); new_assign[name] = v
            result = go(new_assign, universe[(name, v)] + rest)
            if result is not None:
                return result
        return None
    return go({root_name: root_version}, list(universe[(root_name, root_version)]))

def report(universe, greedy=False):
    r = resolve(universe, "app", 1, greedy)
    if r is None:
        return "conflict"
    return "resolved\n" + "\n".join(f"{k}={r[k]}" for k in sorted(r))

ref = report(UNIVERSE_A) + "\n" + report(UNIVERSE_B)
print(ref)
greedy_out = report(UNIVERSE_A, greedy=True) + "\n" + report(UNIVERSE_B, greedy=True)
print(f"greedy(no backtrack): {'DIVERGES' if greedy_out != ref else 'SAME (bad!)'}", file=sys.stderr)

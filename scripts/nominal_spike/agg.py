#!/usr/bin/env python3
"""Aggregate AILANG_NOMINAL_SPIKE_LOG files.

count per key = sum over pids of max(count) logged by that pid (power-of-two
logging => each pid's figure is within 2x below the truth).

usage: agg.py LOG [LOG...] [--by mint|cmp|outer|name|mint1] [--kind q-vs-empty|empty-vs-empty|q-mismatch] [--top N]
"""
import sys, collections

args = sys.argv[1:]
by, kind, top = "mint1", None, 40
logs = []
i = 0
while i < len(args):
    a = args[i]
    if a == "--by": by = args[i+1]; i += 2; continue
    if a == "--kind": kind = args[i+1]; i += 2; continue
    if a == "--top": top = int(args[i+1]); i += 2; continue
    logs.append(a); i += 1

perpid = {}
for lg in logs:
    for line in open(lg, errors="replace"):
        parts = line.rstrip("\n").split("\t")
        if len(parts) < 7: continue
        pid, n, k, name, mint, cmp, outer = parts[:7]
        key = (lg, pid, k, name, mint, cmp, outer)
        perpid[key] = max(perpid.get(key, 0), int(n))

def first(m):
    # first frame of each side of an empty-vs-empty pair
    return " ~ ".join(sorted(set(x.split(" <- ")[0] for x in m.split(" ~ "))))

agg = collections.Counter()
names = collections.defaultdict(collections.Counter)
procs = collections.defaultdict(set)
for (lg, pid, k, name, mint, cmp, outer), n in perpid.items():
    kk = k.split(":")[0]
    if kind and kk != kind: continue
    if by == "mint": g = (kk, mint)
    elif by == "mint1": g = (kk, first(mint))
    elif by == "cmp": g = (kk, cmp)
    elif by == "outer": g = (kk, outer)
    elif by == "name": g = (kk, name)
    elif by == "namemint": g = (kk, name, first(mint))
    else: g = (kk, mint)
    agg[g] += n
    names[g][name] += n
    procs[g].add((lg, pid))

for g, n in agg.most_common(top):
    nm = ", ".join(f"{a}:{b}" for a, b in names[g].most_common(6))
    print(f"{n:>9}  procs={len(procs[g]):<5} {' | '.join(g)}\n           names: {nm}")
print(f"\n{len(agg)} groups, {sum(agg.values())} events")

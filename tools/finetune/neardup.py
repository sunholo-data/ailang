#!/usr/bin/env python3
"""Group near-duplicate AILANG programs (pilot-5 stage 1, rule 2).

The repositories hold many copies of one package in different worktrees. Exact copies are
already removed by content hash; edited copies are not. Train/test splits and per-source caps
must be made by GROUP, or a test program's sibling leaks into training.

Method: 5-token shingles over the code with comments and whitespace removed, MinHash (64
permutations, 16 bands of 4) to find candidate pairs, exact Jaccard on the candidates, and
union-find over pairs at or above --threshold.

  python3 -I tools/finetune/neardup.py IN.jsonl OUT.jsonl [--threshold 0.7]

OUT.jsonl is IN.jsonl with a `group` id added to each record. Prints group statistics.
"""
import argparse
import collections
import hashlib
import json
import re

TOK = re.compile(r"[A-Za-z_][A-Za-z0-9_]*|\d+|\S")
COMMENT = re.compile(r"--[^\n]*")
PERMS, BANDS, ROWS = 64, 16, 4
MASK = (1 << 61) - 1


def shingles(code):
    toks = TOK.findall(COMMENT.sub("", code))
    if len(toks) < 5:
        return {hash(tuple(toks)) & MASK}
    return {int.from_bytes(hashlib.blake2b(" ".join(toks[i:i + 5]).encode(), digest_size=8).digest(), "big") & MASK
            for i in range(len(toks) - 4)}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("src")
    ap.add_argument("out")
    ap.add_argument("--threshold", type=float, default=0.7)
    a = ap.parse_args()
    recs = [json.loads(line) for line in open(a.src)]
    sets = [shingles(r["code"]) for r in recs]
    coef = [(int.from_bytes(hashlib.blake2b(b"a%d" % i, digest_size=8).digest(), "big") | 1,
             int.from_bytes(hashlib.blake2b(b"b%d" % i, digest_size=8).digest(), "big")) for i in range(PERMS)]
    buckets = collections.defaultdict(list)
    for idx, s in enumerate(sets):
        sig = [min(((x * m + c) & MASK) for x in s) for m, c in coef]
        for b in range(BANDS):
            buckets[(b, tuple(sig[b * ROWS:(b + 1) * ROWS]))].append(idx)
    parent = list(range(len(recs)))

    def find(x):
        while parent[x] != x:
            parent[x] = parent[parent[x]]
            x = parent[x]
        return x

    seen = set()
    pairs = 0
    for members in buckets.values():
        if len(members) < 2:
            continue
        for i in range(len(members)):
            for j in range(i + 1, len(members)):
                p = (members[i], members[j])
                if p in seen:
                    continue
                seen.add(p)
                x, y = sets[p[0]], sets[p[1]]
                if len(x & y) / len(x | y) >= a.threshold:
                    pairs += 1
                    parent[find(p[0])] = find(p[1])
    groups = collections.Counter(find(i) for i in range(len(recs)))
    with open(a.out, "w") as f:
        for i, r in enumerate(recs):
            r["group"] = find(i)
            f.write(json.dumps(r) + "\n")
    sizes = sorted(groups.values(), reverse=True)
    print(json.dumps({"programs": len(recs), "groups": len(groups), "similar_pairs": pairs,
                      "programs_in_groups_of_2_or_more": sum(s for s in sizes if s > 1),
                      "largest_groups": sizes[:10], "threshold": a.threshold}))


if __name__ == "__main__":
    main()

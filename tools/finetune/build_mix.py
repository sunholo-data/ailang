#!/usr/bin/env python3
"""Build the pilot-1 training mix (m-eval-finetuning-data-pipeline §Data spec, pilot-1 mix).

Inputs (from extract_lang.py / extract_traj.py, in --data):
  traj_all.jsonl  agent samples in pi tool schema (meta.source, meta.target)
  eval.jsonl      verified programs for training-split benchmarks (task + code)
  repo.jsonl      verified repo modules (code only)

Budgets are in approximate tokens (chars / 3.2) and favour the behaviours 12B
lacks: fix turns (error -> corrected code) and frontier (Claude) trajectories
first. Per-task caps stop the 86 training benchmarks from dominating. The
validation split is by session/task group, so no episode straddles the split.
Writes train.jsonl / valid.jsonl (mlx-lm chat format) and mix_report.json.
"""
import argparse
import hashlib
import json
import os
import random
import re
from collections import Counter, defaultdict

TOK = 3.2
TRAJ_BUDGET = {  # (source, target) -> tokens
    ("claude", "fix"): 600_000, ("claude", "final_write"): 400_000, ("claude", "verify"): 200_000, ("claude", "first"): 100_000,
    ("opencode", "fix"): 700_000, ("opencode", "final_write"): 350_000, ("opencode", "verify"): 150_000, ("opencode", "first"): 100_000,
    ("motoko", "fix"): 250_000, ("motoko", "final_write"): 150_000, ("motoko", "verify"): 50_000, ("motoko", "first"): 50_000,
    ("pi", "fix"): 100_000, ("pi", "final_write"): 100_000, ("pi", "verify"): 60_000, ("pi", "first"): 40_000,
}
PER_TASK_CAP = 12          # trajectory samples per (source, task) bucket
EVAL_PER_BENCH = 10        # verified programs per training benchmark
EVAL_BUDGET = 600_000
REPO_BUDGET = 1_000_000
REPO_MAX_CHARS = 12_000
VALID_FRAC = 0.03
SYS_LANG = "You are an AILANG programmer. Output only the complete AILANG program, no explanations."


def sanitize(msgs):
    """Give every tool result its tool name and drop results whose call is not in context.

    Gemma's template looks the name up from the matching assistant tool_call; context
    trimming can remove that call, which crashes rendering ('response:' + None)."""
    names, out = {}, []
    for m in msgs:
        if m["role"] == "assistant":
            for tc in m.get("tool_calls") or []:
                names[tc["id"]] = tc["function"]["name"]
            out.append(m)
        elif m["role"] == "tool":
            if m.get("tool_call_id") in names:
                out.append(dict(m, name=names[m["tool_call_id"]], content=m.get("content") or ""))
        else:
            out.append(dict(m, content=m.get("content") or ""))
    return out


def toks(obj):
    return int(len(json.dumps(obj)) / TOK)


def group_key(*parts):
    return hashlib.sha1("|".join(map(str, parts)).encode()).hexdigest()


def repo_instruction(code):
    m = re.search(r"^\s*module\s+([\w/.\-]+)", code, re.M)
    if not m:
        return None
    lines, started = [], False
    for ln in code.splitlines():
        s = ln.strip()
        if s.startswith("module "):
            started = True
            continue
        if s.startswith("--") or s.startswith("//"):
            t = s.lstrip("-/ ").strip()
            if t and "DO NOT CHANGE" not in t and not t.startswith("TODO"):
                lines.append(t)
            continue
        if s == "" and not lines:
            continue
        if started or lines:
            break
    purpose = " ".join(lines).strip()
    if len(purpose) < 40:
        return None
    return f"Write the AILANG module `{m.group(1)}`.\n\nPurpose: {purpose[:1200]}"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--seed", type=int, default=7)
    a = ap.parse_args()
    rnd = random.Random(a.seed)
    os.makedirs(a.out, exist_ok=True)
    rows, rep = [], Counter()

    # ---- trajectories
    buckets = defaultdict(list)
    for ln in open(os.path.join(a.data, "traj_all.jsonl")):
        d = json.loads(ln)
        d["messages"] = sanitize(d["messages"])
        meta = d["meta"]
        task = next((m["content"] for m in d["messages"] if m["role"] == "user"), "")
        d["_group"] = group_key(meta["source"], task[:400])
        d["_tok"] = toks(d["messages"])
        if d["_tok"] > 8000:
            rep["traj_too_long"] += 1
            continue
        buckets[(meta["source"], meta["target"])].append(d)
    for key, items in buckets.items():
        rnd.shuffle(items)
        items.sort(key=lambda d: not d["meta"].get("recovery"))  # recovery episodes first, stable shuffle within
        budget, used, per_task = TRAJ_BUDGET.get(key, 0), 0, Counter()
        for d in items:
            if used + d["_tok"] > budget:
                continue
            if per_task[d["_group"]] >= PER_TASK_CAP:
                continue
            per_task[d["_group"]] += 1
            used += d["_tok"]
            rows.append({"messages": d["messages"], "tools": d["tools"], "_group": d["_group"],
                         "_kind": f"traj:{key[0]}:{key[1]}", "_tok": d["_tok"]})
        rep[f"traj:{key[0]}:{key[1]}"] = used

    # ---- eval programs (language, training-split benchmarks)
    per_bench, used = defaultdict(list), 0
    for ln in open(os.path.join(a.data, "eval.jsonl")):
        d = json.loads(ln)
        if d.get("task"):
            per_bench[d["benchmark"]].append(d)
    for bid, items in per_bench.items():
        rnd.shuffle(items)
        for d in items[:EVAL_PER_BENCH]:
            msgs = [{"role": "system", "content": SYS_LANG}, {"role": "user", "content": d["task"]},
                    {"role": "assistant", "content": d["code"]}]
            t = toks(msgs)
            if used + t > EVAL_BUDGET:
                break
            used += t
            rows.append({"messages": msgs, "_group": group_key("eval", bid), "_kind": "lang:eval", "_tok": t})
    rep["lang:eval"] = used

    # ---- repo modules with header-comment instructions (free back-translation)
    repo, used = [], 0
    for ln in open(os.path.join(a.data, "repo.jsonl")):
        d = json.loads(ln)
        if len(d["code"]) > REPO_MAX_CHARS:
            rep["repo_too_long"] += 1
            continue
        ins = repo_instruction(d["code"])
        if not ins:
            rep["repo_no_purpose"] += 1
            continue
        repo.append((d["path"], ins, d["code"]))
    rnd.shuffle(repo)
    for path, ins, code in repo:
        msgs = [{"role": "system", "content": SYS_LANG}, {"role": "user", "content": ins},
                {"role": "assistant", "content": code}]
        t = toks(msgs)
        if used + t > REPO_BUDGET:
            continue
        used += t
        rows.append({"messages": msgs, "_group": group_key("repo", path), "_kind": "lang:repo", "_tok": t})
    rep["lang:repo"] = used

    # ---- split by group, write
    groups = sorted({r["_group"] for r in rows})
    rnd.shuffle(groups)
    valid_groups = set(groups[: max(1, int(len(groups) * VALID_FRAC))])
    rnd.shuffle(rows)
    counts = Counter()
    with open(os.path.join(a.out, "train.jsonl"), "w") as ft, open(os.path.join(a.out, "valid.jsonl"), "w") as fv:
        for r in rows:
            split = "valid" if r["_group"] in valid_groups else "train"
            counts[(split, r["_kind"])] += 1
            counts[(split, "tokens")] += r["_tok"]
            out = {"messages": r["messages"]}
            if "tools" in r:
                out["tools"] = r["tools"]
            (fv if split == "valid" else ft).write(json.dumps(out) + "\n")
    report = {"budgets_used_tokens": dict(rep),
              "samples": {f"{k[0]}:{k[1]}": v for k, v in sorted(counts.items())},
              "total_tokens_est": sum(v for k, v in counts.items() if k[1] == "tokens")}
    json.dump(report, open(os.path.join(a.out, "mix_report.json"), "w"), indent=1)
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    main()

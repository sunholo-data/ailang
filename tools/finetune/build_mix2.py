#!/usr/bin/env python3
"""Pilot-2 training mix: habit-simulated episodes rendered in the LEAN pi wire format.

Input: episodes with thoughts (synth_episodes.py -> thoughts.py). Output: train/valid jsonl in
mlx-lm chat format ({"messages", "tools"}) for tools/finetune/train2.py (whole-episode samples,
loss on every assistant turn, thinking ON).

The rendering reproduces what the eval sends on the lean lane (captured 2026-11-06,
~/models/finetune/traces/pi_format_lean.json):
  developer: LeanSystemPrompt + "\\nCurrent working directory: /private<ws>" + the shell-limits block
  tools:     pi's read/bash/edit/write schemas, verbatim
  user:      the arm prefix (ft-slim-v1 or ft-none, as the harness prepends it) + the task prompt
  assistant: {reasoning: thought, tool_calls: [call]} per step; the final reply with its thought

Balancing: repair episodes are capped per (benchmark, error kind) so the 1,259 `++` cases do not
drown the rest; solve episodes are capped per benchmark; examples are repeated (canonical syntax).
Validation is split by benchmark, so no task straddles train/valid.
"""
import argparse
import collections
import json
import os
import random
import re

CWD_LINE = "\nCurrent working directory: "


def lean_format(path):
    d = json.load(open(os.path.expanduser(path)))
    sysc = d["system"]
    i = sysc.index(CWD_LINE)
    head = sysc[:i]
    rest = sysc[i + len(CWD_LINE):]
    tail = rest[rest.index("\n"):]  # everything after the cwd value (the shell-limits block)
    return head, tail, d["tools"]


def ws_of(task_prompt):
    m = re.search(r"Write your complete solution to: \*{0,2}(\S+?)/benchmark/solution\.ail", task_prompt)
    return m.group(1) if m else None


def error_kind(ep):
    for s in ep["steps"]:
        r = s["result"]
        m = re.search(r"(`\+\+` is for lists only|undefined variable: \w+|MOD\d+|PAR\w+|type unification|effect checking)", r)
        if m:
            return re.sub(r"undefined variable: (\w+)", r"undef:\1", m.group(1))
    return "other"


def render(ep, head, tail, tools, arm_prefix):
    ws = ws_of(ep["task_prompt"])
    if not ws:
        return None
    msgs = [{"role": "developer", "content": head + CWD_LINE + "/private" + ws + tail},
            {"role": "user", "content": arm_prefix + ep["task_prompt"]}]
    for k, s in enumerate(ep["steps"]):
        cid = f"call_{k + 1}"
        msgs.append({"role": "assistant", "content": "", "reasoning": s["thought"],
                     "tool_calls": [{"id": cid, "type": "function",
                                     "function": {"name": s["call"]["name"], "arguments": s["call"]["args"]}}]})
        msgs.append({"role": "tool", "tool_call_id": cid, "name": s["call"]["name"], "content": s["result"]})
    msgs.append({"role": "assistant", "content": ep["final"], "reasoning": ep["final_thought"]})
    return {"messages": msgs, "tools": tools}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--episodes", required=True, help="episodes with thoughts (thoughts.py output)")
    ap.add_argument("--out", required=True)
    ap.add_argument("--format", default="~/models/finetune/traces/pi_format_lean.json")
    ap.add_argument("--arm-slim", default="~/models/finetune/traces/arm_slim.json")
    ap.add_argument("--arm-none", default="~/models/finetune/traces/arm_none.json")
    ap.add_argument("--solve-per-bench", type=int, default=12)
    ap.add_argument("--repair-per-kind", type=int, default=6, help="per (benchmark, error kind)")
    ap.add_argument("--example-repeat", type=int, default=2)
    ap.add_argument("--valid-frac", type=float, default=0.05)
    ap.add_argument("--seed", type=int, default=13)
    a = ap.parse_args()
    rnd = random.Random(a.seed)
    head, tail, tools = lean_format(a.format)
    arms = [json.load(open(os.path.expanduser(a.arm_slim)))["prefix"], json.load(open(os.path.expanduser(a.arm_none)))["prefix"]]
    eps = [json.loads(l) for l in open(os.path.expanduser(a.episodes))]
    rnd.shuffle(eps)

    picked, per, rep = [], collections.Counter(), collections.Counter()
    for e in eps:
        if e["kind"] == "solve":
            key = ("solve", e["benchmark"])
            cap = a.solve_per_bench
        elif e["kind"] == "repair":
            key = ("repair", e["benchmark"], error_kind(e))
            cap = a.repair_per_kind
        else:
            key, cap = ("example", e["benchmark"]), a.example_repeat
        if per[key] >= cap:
            rep[f"capped:{e['kind']}"] += 1
            continue
        per[key] += 1
        picked.append(e)
        if e["kind"] == "example":  # repeat examples (canonical syntax) with a fresh arm draw
            for _ in range(a.example_repeat - 1):
                picked.append(e)
                per[key] += 1

    benches = sorted({e["benchmark"] for e in picked})
    rnd.shuffle(benches)
    valid_b = set(benches[: max(1, int(len(benches) * a.valid_frac))])
    os.makedirs(os.path.expanduser(a.out), exist_ok=True)
    counts = collections.Counter()
    chars = collections.Counter()
    with open(os.path.join(os.path.expanduser(a.out), "train.jsonl"), "w") as ft, \
            open(os.path.join(os.path.expanduser(a.out), "valid.jsonl"), "w") as fv:
        for e in picked:
            r = render(e, head, tail, tools, rnd.choice(arms))
            if r is None:
                rep["no_workspace"] += 1
                continue
            split = "valid" if e["benchmark"] in valid_b else "train"
            (fv if split == "valid" else ft).write(json.dumps(r) + "\n")
            counts[(split, e["kind"])] += 1
            chars[split] += len(json.dumps(r))
    report = {"samples": {f"{k[0]}:{k[1]}": v for k, v in sorted(counts.items())},
              "approx_tokens": {k: int(v / 3.2) for k, v in chars.items()},
              "benchmarks": len(benches), "valid_benchmarks": sorted(valid_b), "dropped": dict(rep)}
    json.dump(report, open(os.path.join(os.path.expanduser(a.out), "mix_report.json"), "w"), indent=1)
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    main()

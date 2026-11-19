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
import sys
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
        am = {"role": "assistant", "content": "", "reasoning": s["thought"],
              "tool_calls": [{"id": cid, "type": "function",
                              "function": {"name": s["call"]["name"], "arguments": s["call"]["args"]}}]}
        if s.get("no_loss"):  # the fine-tune's own mistake: context only, never a target (train2.py)
            am["weight"] = 0
        msgs.append(am)
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
    ap.add_argument("--onpolicy-per-kind", type=int, default=8, help="per (benchmark, error kind)")
    ap.add_argument("--max-steps", type=int, default=24, help="drop longer episodes (pilot-2 looped for 40-400 turns)")
    ap.add_argument("--build-per-bench", type=int, default=2, help="pilot-4 build-up episodes per benchmark")
    ap.add_argument("--max-write-chars", type=int, default=0,
                    help="drop episodes with a TRAINED write longer than this (0 = off). Pilot-3's long writes "
                         "sometimes ran away and were dropped by the tool-call parser (D-FT-15)")
    ap.add_argument("--example-repeat", type=int, default=2)
    ap.add_argument("--valid-frac", type=float, default=0.05)
    ap.add_argument("--seed", type=int, default=13)
    ap.add_argument("--tokenizer", default="", help="model dir; with --max-tokens, measure each sample as train2.py renders it")
    ap.add_argument("--max-tokens", type=int, default=0,
                    help="drop samples longer than this (the training max_seq_length). Training truncates from the END, "
                         "so an over-long on-policy sample keeps its no-loss mistakes and loses the fix")
    a = ap.parse_args()
    ntok = None
    if a.tokenizer and a.max_tokens:
        from pathlib import Path
        from mlx_lm.utils import load_tokenizer
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        from train2 import _render  # the exact render training uses: one implementation, not two
        _tok = load_tokenizer(Path(os.path.expanduser(a.tokenizer)))
        ntok = lambda r: len(_render(_tok, r["messages"], r.get("tools")))
    rnd = random.Random(a.seed)
    head, tail, tools = lean_format(a.format)
    arms = [json.load(open(os.path.expanduser(a.arm_slim)))["prefix"], json.load(open(os.path.expanduser(a.arm_none)))["prefix"]]
    eps = [json.loads(l) for l in open(os.path.expanduser(a.episodes))]
    rnd.shuffle(eps)

    picked, per, rep = [], collections.Counter(), collections.Counter()
    for e in eps:
        if len(e["steps"]) > a.max_steps:
            rep[f"too_long:{e['kind']}"] += 1
            continue
        if a.max_write_chars and any(st["call"]["name"] == "write" and not st.get("no_loss") and
                                     len(st["call"]["args"].get("content", "")) > a.max_write_chars for st in e["steps"]):
            rep[f"long_write:{e['kind']}"] += 1
            continue
        if e["kind"] == "solve":
            key = ("solve", e["benchmark"])
            cap = a.solve_per_bench
        elif e["kind"] == "onpolicy":
            key = ("onpolicy", e["benchmark"], e.get("error_kind") or error_kind(e))
            cap = a.onpolicy_per_kind
        elif e["kind"] == "build":
            key, cap = ("build", e["benchmark"]), a.build_per_bench
        elif e["kind"] == "repair":
            key = ("repair", e["benchmark"], error_kind(e))
            cap = a.repair_per_kind
        else:
            key, cap = ("example", e["benchmark"]), a.example_repeat
        if per[key] >= cap:
            rep[f"capped:{e['kind']}"] += 1
            continue
        # Render (and length-check) BEFORE taking a cap slot, so a too-long sample frees its slot
        # for another episode of the same kind. The arm draw is part of the length.
        reps = []
        for _ in range(a.example_repeat if e["kind"] == "example" else 1):  # examples repeat, fresh arm each
            r = render(e, head, tail, tools, rnd.choice(arms))
            if r is None:
                rep["no_workspace"] += 1
            elif ntok and ntok(r) > a.max_tokens:
                rep[f"over_max_tokens:{e['kind']}"] += 1
            else:
                reps.append(r)
        if not reps:
            continue
        for r in reps:
            per[key] += 1
            picked.append((e, r))

    benches = sorted({e["benchmark"] for e, _ in picked})
    rnd.shuffle(benches)
    valid_b = set(benches[: max(1, int(len(benches) * a.valid_frac))])
    os.makedirs(os.path.expanduser(a.out), exist_ok=True)
    counts = collections.Counter()
    chars = collections.Counter()
    with open(os.path.join(os.path.expanduser(a.out), "train.jsonl"), "w") as ft, \
            open(os.path.join(os.path.expanduser(a.out), "valid.jsonl"), "w") as fv:
        for e, r in picked:
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

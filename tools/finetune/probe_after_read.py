#!/usr/bin/env python3
"""Pre-eval behaviour probe (pilot-4 gate, D-FT-16): replay real "after reading the stub" requests
from a TRAINING-benchmark trace against a model, and record its first action.

Each request is the one pi sent right after the model read the solution template, taken from a
collection trace (never a held-out benchmark). The probe streams the reply and stops at the first
tool call, so it costs one turn per benchmark. A reply with no tool call is a stop; a long reply
with no tool call and no text is a dropped call (see trace_dropped.py).

Needs the rig lease: run it under rig_lock_acquire (it sends $AILANG_RIG_LEASE as X-Rig-Lease).

  python3 -I tools/finetune/probe_after_read.py --trace TRACE.jsonl --model gemma4-12b-ailang-ft4 [-n 16]
"""
import argparse
import json
import os
import random
import re
import sys
import time
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from extract_lang import load_heldout  # noqa: E402

WS_RE = re.compile(r"ailang_eval/(.+?)_pi-gemma4")


def after_read_requests(trace):
    found = {}
    with open(trace) as f:
        for line in f:
            rq = json.loads(line).get("request")
            if not isinstance(rq, dict) or len(rq.get("messages") or []) != 4:
                continue
            ms = rq["messages"]
            calls = ms[2].get("tool_calls") or []
            if ms[2].get("role") == "assistant" and ms[3].get("role") == "tool" and calls and \
                    calls[0]["function"]["name"] == "read":
                m = WS_RE.search(json.dumps(ms[:2]))
                if m and m.group(1) not in found:
                    found[m.group(1)] = rq
    return found


def first_action(rq, model, url):
    body = dict(rq, model=model, stream=True)
    hdr = {"Content-Type": "application/json", "X-Rig-Lease": os.environ.get("AILANG_RIG_LEASE", "")}
    t, text, kind, comp = time.time(), "", None, None
    with urllib.request.urlopen(urllib.request.Request(url, data=json.dumps(body).encode(), headers=hdr), timeout=1800) as r:
        for raw in r:
            line = raw.decode().strip()
            if not line.startswith("data: {"):
                continue
            d = json.loads(line[6:])
            if d.get("usage"):
                comp = d["usage"].get("completion_tokens")
            for ch in d.get("choices", []):
                dl = ch.get("delta") or {}
                if dl.get("tool_calls"):
                    name = dl["tool_calls"][0].get("function", {}).get("name")
                    if name:
                        return "tool:" + name, round(time.time() - t)
                text += dl.get("content") or ""
                if ch.get("finish_reason"):
                    kind = "text" if text.strip() else "empty"
    if kind == "empty" and (comp or 0) >= 500:
        kind = "dropped"
    return kind or "empty", round(time.time() - t)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--trace", required=True, help="a collection trace on TRAINING benchmarks")
    ap.add_argument("--model", required=True)
    ap.add_argument("-n", type=int, default=16)
    ap.add_argument("--url", default="http://127.0.0.1:11434/v1/chat/completions")
    ap.add_argument("--seed", type=int, default=7)
    a = ap.parse_args()
    held = load_heldout()
    reqs = after_read_requests(a.trace)
    bad = sorted(set(reqs) & held)
    if bad:
        sys.exit(f"held-out benchmarks in the probe trace: {bad}")
    keys = sorted(reqs)
    random.Random(a.seed).shuffle(keys)
    res = []
    for b in keys[: a.n]:
        kind, secs = first_action(reqs[b], a.model, a.url)
        res.append({"benchmark": b, "first": kind, "secs": secs})
        print(f"{b}: {kind} ({secs}s)", flush=True)
    acted = sum(r["first"].startswith("tool:") for r in res)
    print(json.dumps({"model": a.model, "probed": len(res), "acted": acted,
                      "no_action": [r["benchmark"] for r in res if not r["first"].startswith("tool:")]}))


if __name__ == "__main__":
    main()

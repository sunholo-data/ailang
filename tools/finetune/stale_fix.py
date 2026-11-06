#!/usr/bin/env python3
"""Staleness pairs: programs that passed when banked but fail `ailang check` today, plus a verified fix.

Habit 4 of the fine-tune ("trust the compiler over memory", design doc §Pilot-2 direction)
needs examples where old-but-plausible AILANG meets today's compiler error and is fixed by
following it. eval_results/ holds thousands: the dominant one is `++` on strings, removed
in v0.13.0, whose error message names the fix.

  collect  scan eval_results/ for passing AILANG programs on TRAINING-split benchmarks
           (tools/finetune/heldout.json excluded) that fail `ailang check` today; keep the
           full compiler error. Only benchmarks verifiable by a plain run (no stdin, input
           files, hidden tests, CLI args or source constraints).
  fix      ask an open-weight model (OpenRouter; first-party vendors are never routed there)
           for a minimal fix given the code and the error; keep a fix ONLY if `ailang check`
           passes and `ailang run` prints the benchmark's expected stdout exactly.

  uv run --with pyyaml python tools/finetune/stale_fix.py collect --out DIR
  uv run --with pyyaml python tools/finetune/stale_fix.py fix --out DIR --model google/gemma-4-31b-it --limit 50

OPENROUTER_API_KEY comes from the environment and is never printed.
"""
import argparse
import concurrent.futures as cf
import json
import os
import re
import subprocess
import sys
import tempfile
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from extract_lang import AILANG, REPO, load_heldout, norm_hash  # noqa: E402

import yaml  # noqa: E402

UNVERIFIABLE = ("stdin", "input_files", "hidden_tests_dir", "cli_args", "source_constraints", "solution_files", "grading")


def spec(bid, cache={}):
    if bid not in cache:
        try:
            cache[bid] = yaml.safe_load(open(os.path.join(REPO, "benchmarks", f"{bid}.yml")))
        except (OSError, yaml.YAMLError):
            cache[bid] = None
    return cache[bid]


def in_ws(code, fn):
    with tempfile.TemporaryDirectory() as td:
        os.makedirs(os.path.join(td, "benchmark"))
        with open(os.path.join(td, "benchmark", "solution.ail"), "w") as f:
            f.write(code)
        return fn(td)


def check(code):
    def run(td):
        r = subprocess.run([AILANG, "check", "benchmark/solution.ail"], cwd=td, capture_output=True, text=True, timeout=60)
        return r.returncode == 0, (r.stderr + r.stdout).strip()
    return in_ws(code, run)


def run_ok(code, b):
    caps = ",".join(b.get("caps") or ["IO"])
    def run(td):
        try:
            r = subprocess.run([AILANG, "run", "--caps", caps, "--entry", b.get("entrypoint") or "main",
                                "benchmark/solution.ail"], cwd=td, capture_output=True, text=True, timeout=60)
        except subprocess.TimeoutExpired:
            return False, "timeout"
        return r.stdout.rstrip("\n") == (b.get("expected_stdout") or "").rstrip("\n"), r.stdout
    return in_ws(code, run)


def collect(a):
    held = load_heldout()
    seen, cands = set(), []
    for dp, _, fns in os.walk(os.path.join(REPO, "eval_results")):
        for fn in fns:
            if not fn.endswith(".json"):
                continue
            try:
                d = json.load(open(os.path.join(dp, fn)))
            except (OSError, ValueError):
                continue
            if not isinstance(d, dict) or d.get("lang") != "ailang" or not d.get("code"):
                continue
            if d.get("id") in held or not (d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")):
                continue
            b = spec(d["id"])
            if not b or any(k in b for k in UNVERIFIABLE):
                continue
            h = norm_hash(d["code"])
            if h in seen:
                continue
            seen.add(h)
            cands.append((d["id"], d.get("ailang_version"), d["code"]))
    with cf.ThreadPoolExecutor(8) as ex:
        res = list(ex.map(lambda c: check(c[2]), cands))
    n = 0
    with open(os.path.join(a.out, "stale.jsonl"), "w") as f:
        for (bid, ver, code), (ok, err) in zip(cands, res):
            if ok:
                continue
            err = "\n".join(err.splitlines()[:12])
            f.write(json.dumps({"benchmark": bid, "banked_version": ver, "code": code, "error": err}) + "\n")
            n += 1
    print(json.dumps({"passing_candidates": len(cands), "stale_today": n}))


PROMPT = """This AILANG program used to work, but the current compiler rejects it.

Compiler output:
{error}

Program:
{code}

Fix the program so the current compiler accepts it, following the error message. Change only what
the error requires; keep the program's behaviour identical. Reply with the complete fixed program
and nothing else (no markdown fences, no explanation)."""


def ask(model, prompt, key):
    body = json.dumps({"model": model, "messages": [{"role": "user", "content": prompt}],
                       "temperature": 0.2, "max_tokens": 6000}).encode()
    req = urllib.request.Request("https://openrouter.ai/api/v1/chat/completions", data=body,
                                 headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=300) as r:
        d = json.load(r)
    txt = d["choices"][0]["message"].get("content") or ""
    txt = re.sub(r"^```\w*\n|\n```\s*$", "", txt.strip())
    return txt, d.get("usage") or {}


def fix(a):
    key = os.environ.get("OPENROUTER_API_KEY")
    if not key:
        sys.exit("OPENROUTER_API_KEY not set")
    if a.model.split("/")[0] in ("anthropic", "openai", "google") and "gemma" not in a.model:
        sys.exit("first-party vendors are never routed via OpenRouter")
    items = [json.loads(l) for l in open(os.path.join(a.out, "stale.jsonl"))][: a.limit]
    done = set()
    outp = os.path.join(a.out, "stale_pairs.jsonl")
    if os.path.exists(outp):
        done = {json.loads(l)["code_hash"] for l in open(outp)}

    def one(it):
        h = norm_hash(it["code"])
        if h in done:
            return None
        try:
            fixed, usage = ask(a.model, PROMPT.format(error=it["error"], code=it["code"]), key)
        except Exception as e:  # network / provider error: record, never retry silently
            return {"code_hash": h, "ok": False, "why": f"api: {type(e).__name__}"}
        ok, err = check(fixed)
        if not ok:
            return {"code_hash": h, "ok": False, "why": "check", "usage": usage}
        good, _ = run_ok(fixed, spec(it["benchmark"]))
        return {"code_hash": h, "ok": good, "why": "" if good else "stdout", "usage": usage,
                **({"benchmark": it["benchmark"], "old": it["code"], "error": it["error"], "fixed": fixed} if good else {})}

    stats = {"ok": 0, "check": 0, "stdout": 0, "api": 0, "in": 0, "out": 0}
    with cf.ThreadPoolExecutor(a.workers) as ex, open(outp, "a") as f:
        for r in ex.map(one, items):
            if r is None:
                continue
            f.write(json.dumps(r) + "\n")
            f.flush()
            stats["ok" if r["ok"] else r["why"].split(":")[0]] += 1
            stats["in"] += (r.get("usage") or {}).get("prompt_tokens", 0)
            stats["out"] += (r.get("usage") or {}).get("completion_tokens", 0)
    print(json.dumps(stats))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=["collect", "fix"])
    ap.add_argument("--out", required=True)
    ap.add_argument("--model", default="google/gemma-4-31b-it")
    ap.add_argument("--limit", type=int, default=50)
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()
    a.out = os.path.expanduser(a.out)
    os.makedirs(a.out, exist_ok=True)
    collect(a) if a.cmd == "collect" else fix(a)


if __name__ == "__main__":
    main()

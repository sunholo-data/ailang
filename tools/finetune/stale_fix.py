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


def run_ok(code, b, expected=None):
    caps = ",".join(b.get("caps") or ["IO"])
    want = (expected if expected is not None else (b.get("expected_stdout") or "")).rstrip("\n")
    def run(td):
        try:
            r = subprocess.run([AILANG, "run", "--caps", caps, "--entry", b.get("entrypoint") or "main",
                                "benchmark/solution.ail"], cwd=td, capture_output=True, text=True, timeout=60)
        except subprocess.TimeoutExpired:
            return False, "timeout"
        return r.stdout.rstrip("\n") == want, r.stdout
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
            cands.append((d["id"], d.get("ailang_version"), d["code"], d.get("expected_stdout")))
    with cf.ThreadPoolExecutor(8) as ex:
        res = list(ex.map(lambda c: check(c[2]), cands))
    n = 0
    with open(os.path.join(a.out, "stale.jsonl"), "w") as f:
        for (bid, ver, code, exp), (ok, err) in zip(cands, res):
            if ok:
                continue
            err = "\n".join(err.splitlines()[:12])
            # The output the program produced WHEN BANKED: benchmarks change after banking
            # (fold_reduce's data did), and a staleness fix must preserve behaviour.
            f.write(json.dumps({"benchmark": bid, "banked_version": ver, "code": code, "error": err,
                                "expected_stdout": exp}) + "\n")
            n += 1
    print(json.dumps({"passing_candidates": len(cands), "stale_today": n}))


PROMPT = """This AILANG program used to work, but it no longer does with the current toolchain.

Feedback (compiler error, or the run's output):
{error}

Program:
{code}

Fix the program so the current compiler accepts it and it prints exactly what it printed before,
following the feedback. Change only what the feedback requires; keep the program's behaviour identical. Reply with the complete fixed program
and nothing else (no markdown fences, no explanation)."""


# On-policy items (onpolicy_pairs.py): the fine-tune's OWN failing attempt at a training task.
# The repairer gets the task, the expected output and the AILANG agent reference, because the
# attempt's mistakes are borrowed syntax the error message alone does not always name.
PROMPT_ONPOLICY = """AILANG reference:
{ref}

Task given to the author of the program below:
{task}

Expected output (exact):
{expected}

The author's attempt:
{code}

Feedback (compiler error, or the run's output):
{error}

Fix the attempt so `ailang check` accepts it and it prints exactly the expected output. Keep the
attempt's structure and names; change only what is wrong. You have no tools and cannot run
anything: do not emit tool calls. The program will be compiled and run for you, and any new
error sent back. Reply with the complete fixed program and nothing else (no markdown fences,
no explanation)."""


def agent_ref(cache={}):
    if "r" not in cache:
        cache["r"] = subprocess.run([AILANG, "agent-prompt"], capture_output=True, text=True, timeout=60).stdout
    return cache["r"]


def make_prompt(a, it, code, error):
    if a.prompt == "onpolicy":
        return PROMPT_ONPOLICY.format(ref=agent_ref(), task=it.get("task", ""), expected=(it.get("expected_stdout") or "")[:3000],
                                      code=code, error=error)
    return PROMPT.format(error=error, code=code)


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


EFFORT = "medium"


def ask_claude(model, prompt):
    """Claude via the CLI on the subscription (OAuth): no tools, no session, no settings,
    ANTHROPIC_API_KEY stripped so it can never bill the API (same flags as the quorum's
    claude reviewer, internal/mission/quorum/claude_reviewer.go)."""
    env = {k: v for k, v in os.environ.items() if k != "ANTHROPIC_API_KEY"}
    with tempfile.TemporaryDirectory() as td:  # empty cwd: no project CLAUDE.md
        # --effort medium: at the default the model reasoned (or emitted fake tool calls) for 15 min
        # on one item (2026-11-09); our own compile-and-run loop does the checking instead.
        r = subprocess.run(["claude", "-p", "--model", model, "--effort", EFFORT, "--output-format", "json", "--tools", "",
                            "--no-session-persistence", "--setting-sources", "", "--strict-mcp-config"],
                           input=prompt, capture_output=True, text=True, timeout=900, cwd=td, env=env)
    if r.returncode != 0:
        raise RuntimeError(f"claude rc={r.returncode}")
    d = json.loads(r.stdout)
    if d.get("is_error"):
        raise RuntimeError("claude is_error")
    txt = re.sub(r"^```\w*\n|\n```\s*$", "", (d.get("result") or "").strip())
    u = d.get("usage") or {}
    return txt, {"prompt_tokens": u.get("input_tokens", 0), "completion_tokens": u.get("output_tokens", 0)}


def fix(a):
    if a.backend == "openrouter":
        key = os.environ.get("OPENROUTER_API_KEY")
        if not key:
            sys.exit("OPENROUTER_API_KEY not set")
        if a.model.split("/")[0] in ("anthropic", "openai", "google") and "gemma" not in a.model:
            sys.exit("first-party vendors are never routed via OpenRouter")
        call = lambda prompt: ask(a.model, prompt, key)
    else:
        if not a.model.startswith("claude-"):
            sys.exit("claude-cli backend needs a full claude model id (never an alias)")
        call = lambda prompt: ask_claude(a.model, prompt)
    items = [json.loads(l) for l in open(os.path.expanduser(a.items) if a.items else os.path.join(a.out, "stale.jsonl"))]
    if a.cmd == "retry":  # only what the first pass could not fix
        failed = {json.loads(l)["code_hash"] for l in open(os.path.join(a.out, "stale_pairs.jsonl")) if not json.loads(l)["ok"]}
        items = [it for it in items if norm_hash(it["code"]) in failed]
    items = items[: a.limit]
    done = set()
    outp = os.path.join(a.out, a.pairs_out or ("stale_pairs.jsonl" if a.cmd == "fix" else f"stale_pairs_{a.tag}.jsonl"))
    if os.path.exists(outp):
        done = {json.loads(l)["code_hash"] for l in open(outp)}

    def one(it):
        """Iterative repair: the compiler reports ONE error at a time, so a one-shot fix of the
        first error usually surfaces the next (measured 2026-11-06: Opus 1/10 one-shot, every
        failure a NEW error). Feed each new error back, up to --rounds; keep the whole chain."""
        h = norm_hash(it["code"])
        if h in done:
            return None
        code, error, chain, usage = it["code"], it["error"], [], {"prompt_tokens": 0, "completion_tokens": 0}
        for _ in range(a.rounds):
            try:
                fixed, u = call(make_prompt(a, it, code, error))
            except Exception as e:  # network / provider error: record, never retry silently
                return {"code_hash": h, "ok": False, "why": f"api: {type(e).__name__}", "rounds": len(chain)}
            for k in usage:
                usage[k] += u.get(k, 0)
            chain.append({"code": code, "error": error, "fixed": fixed})
            ok, err = check(fixed)
            if ok:
                want = it.get("expected_stdout")
                good, got = run_ok(fixed, spec(it["benchmark"]), want)
                if not good:
                    if len(chain) < a.rounds:  # what an agent sees after running: the output differs
                        exp = want if want is not None else (spec(it["benchmark"]).get("expected_stdout") or "")
                        code, error = fixed, ("The program now compiles, but its output changed. It must print exactly:\n"
                                              f"{exp[:1500]}\n\nIt printed:\n{(got or '')[:1500]}")
                        continue
                    return {"code_hash": h, "ok": False, "why": "stdout", "usage": usage, "rounds": len(chain),
                            "benchmark": it["benchmark"], "fixed": fixed, "got": (got or "")[:2000]}
                return {"code_hash": h, "ok": True, "why": "", "usage": usage, "rounds": len(chain),
                        "benchmark": it["benchmark"], "old": it["code"], "error": it["error"], "fixed": fixed,
                        "chain": chain, "prev": it.get("prev"), "kind": it.get("kind")}
            code, error = fixed, "\n".join(err.splitlines()[:12])
        return {"code_hash": h, "ok": False, "why": "check", "usage": usage, "rounds": len(chain)}

    stats = {"ok": 0, "check": 0, "stdout": 0, "api": 0, "in": 0, "out": 0, "model": a.model}
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
    ap.add_argument("cmd", choices=["collect", "fix", "retry"])
    ap.add_argument("--backend", choices=["openrouter", "claude-cli"], default="openrouter")
    ap.add_argument("--tag", default="retry", help="retry output: stale_pairs_<tag>.jsonl")
    ap.add_argument("--out", required=True)
    ap.add_argument("--model", default="google/gemma-4-31b-it")
    ap.add_argument("--limit", type=int, default=50)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--items", help="items file (default <out>/stale.jsonl); onpolicy_pairs.py output for --prompt onpolicy")
    ap.add_argument("--pairs-out", help="output file name in --out (default stale_pairs.jsonl)")
    ap.add_argument("--prompt", choices=["stale", "onpolicy"], default="stale")
    ap.add_argument("--rounds", type=int, default=4, help="repair rounds per program (new error fed back each round)")
    a = ap.parse_args()
    a.out = os.path.expanduser(a.out)
    os.makedirs(a.out, exist_ok=True)
    collect(a) if a.cmd == "collect" else fix(a)


if __name__ == "__main__":
    main()

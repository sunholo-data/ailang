#!/usr/bin/env python3
"""Fine-tune extractor v0, language-data half (m-eval-finetuning-data-pipeline §Data spec).

Collects AILANG programs from two sources and keeps only what the CURRENT compiler
accepts:

  repo   every unique .ail file under the given roots (sunholo-data repos, motoko
         checkouts, daneel...), de-duplicated by content hash, checked in place
         from its git top-level so sibling-module imports resolve.
  eval   passing programs from eval_results/ on TRAINING-SPLIT benchmarks only
         (tools/finetune/heldout.json is excluded), checked as benchmark/solution.ail.

Contamination guard: any program whose normalised code hash equals a held-out
benchmark solution (from any model) is dropped from BOTH sources.

Output (in --out): repo.jsonl, eval.jsonl (one record per kept program, with
source path / benchmark id / task prompt where known), and report.json with
per-source counts and rejection reasons. No task descriptions are written here;
repo code gets those in the G2 back-translation step.

Usage:
  uv run --with pyyaml python tools/finetune/extract_lang.py --out DIR --roots R1 R2 ...
"""
import argparse
import concurrent.futures as cf
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
from collections import Counter

import yaml

AILANG = os.path.expanduser("~/go/bin/ailang")
SKIP = ("node_modules", "/.ailang/cache", "/eval_results", "/.git/", "/target/", "/dist/")
HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))


def norm_hash(code: str) -> str:
    """Hash with comments and whitespace removed, so trivially reformatted copies collide."""
    code = re.sub(r"--[^\n]*|//[^\n]*", "", code)
    code = re.sub(r"\s+", "", code)
    return hashlib.sha1(code.encode()).hexdigest()


def git_top(path: str) -> str:
    d = os.path.dirname(path)
    while d and d != "/":
        if os.path.exists(os.path.join(d, ".git")):
            return d
        d = os.path.dirname(d)
    return os.path.dirname(path)


WORKTREE_MARKERS = ("worktree", "/.wt-", "-wt/", "/.claude/", "/tmp/")


def copy_priority(path: str) -> tuple:
    """Order duplicate copies: this repo's main checkout first, worktrees and scratch copies last.

    Identical files live in many worktrees, and a stale or relocated copy can fail
    `ailang check` (wrong git top, missing sibling modules) while the canonical one
    passes. Pilot-1 lost ~370 of ~397 passing examples/ files that way (2026-11-06).
    """
    in_repo = path.startswith(REPO + os.sep)
    return (not in_repo, any(m in path for m in WORKTREE_MARKERS), path)


def check_first_passing(paths: list) -> tuple:
    """Check copies in priority order (at most 3); return (path, ok, why) for the first that passes."""
    why = "no copy checked"
    for p in sorted(paths, key=copy_priority)[:3]:
        ok, why = check_file(p, git_top(p))
        if ok:
            return p, True, ""
    return paths[0], False, why


def check_file(path: str, cwd: str) -> tuple[bool, str]:
    # Pass the path RELATIVE to the git top: ailang derives the expected module name
    # from the path it is given, so an absolute path makes every path-named module
    # (`module examples/runnable/x`) fail MOD010 against "Users/<user>/dev/...".
    rel = os.path.relpath(path, cwd) if path.startswith(cwd + os.sep) else path
    try:
        r = subprocess.run([AILANG, "check", rel], cwd=cwd, capture_output=True, text=True, timeout=60)
    except subprocess.TimeoutExpired:
        return False, "timeout"
    if r.returncode == 0:
        return True, ""
    first = next((ln for ln in (r.stderr + r.stdout).splitlines() if "Error" in ln or "error" in ln), "error")
    return False, re.sub(r"\S+\.ail:\d+:\d+", "<loc>", first)[:120]


def check_eval_program(code: str) -> tuple[bool, str]:
    with tempfile.TemporaryDirectory() as td:
        os.makedirs(os.path.join(td, "benchmark"))
        p = os.path.join(td, "benchmark", "solution.ail")
        with open(p, "w") as f:
            f.write(code)
        return check_file("benchmark/solution.ail", td)


def load_heldout():
    """Every benchmark id excluded from TRAINING data: the eval set (core +
    stretch) plus retired held-out ids whose banked solutions must stay out
    (retired_excluded). The eval list itself is core + stretch only."""
    h = json.load(open(os.path.join(HERE, "heldout.json")))
    return set(h["core"] + h["stretch"] + h.get("retired_excluded", []))


def task_prompt(bid: str) -> str:
    try:
        b = yaml.safe_load(open(os.path.join(REPO, "benchmarks", f"{bid}.yml")))
    except OSError:
        return ""
    return (b.get("task_prompt") or b.get("prompt") or "").replace("<LANG>", "AILANG")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--roots", nargs="+", required=True)
    ap.add_argument("--workers", type=int, default=10)
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    heldout = load_heldout()
    report = {"compiler": subprocess.run([AILANG, "--version"], capture_output=True, text=True).stdout.strip().splitlines()[0]}

    # ---- eval results: split into held-out (contamination reference) and training candidates
    held_hashes, eval_cands, seen = set(), [], set()
    ec = Counter()
    for root, _, files in os.walk(os.path.join(REPO, "eval_results")):
        for fn in files:
            if not fn.endswith(".json") or fn == "summary.json":
                continue
            try:
                d = json.load(open(os.path.join(root, fn)))
            except (OSError, ValueError):
                continue
            if not isinstance(d, dict) or d.get("lang") != "ailang" or not d.get("code"):
                continue
            code = d["code"].strip()
            nh = norm_hash(code)
            if d.get("id") in heldout:
                held_hashes.add(nh)
                ec["heldout_benchmark_program"] += 1
                continue
            if not (d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")):
                ec["not_passing"] += 1
                continue
            if nh in seen:
                ec["duplicate"] += 1
                continue
            seen.add(nh)
            eval_cands.append((d["id"], d.get("model"), d.get("ailang_version"), code, nh))
    ec["candidates"] = len(eval_cands)

    # ---- repo corpus
    repo_cands, rseen = [], {}   # rseen: content hash -> index into repo_cands
    rc = Counter()
    for r in a.roots:
        for dp, _, fns in os.walk(os.path.expanduser(r)):
            if any(s in dp + "/" for s in SKIP):
                continue
            for fn in fns:
                if not fn.endswith(".ail"):
                    continue
                p = os.path.join(dp, fn)
                try:
                    code = open(p, encoding="utf-8").read()
                except (OSError, UnicodeDecodeError):
                    rc["unreadable"] += 1
                    continue
                rc["scanned"] += 1
                h = hashlib.sha1(code.encode()).hexdigest()
                if h in rseen:
                    rc["duplicate"] += 1
                    repo_cands[rseen[h]][0].append(p)   # keep every copy; the check picks one
                    continue
                rseen[h] = len(repo_cands)
                repo_cands.append(([p], code, norm_hash(code)))
    rc["candidates"] = len(repo_cands)

    # ---- contamination guard (both sources)
    ec["contaminated"] = sum(1 for c in eval_cands if c[4] in held_hashes)
    rc["contaminated"] = sum(1 for c in repo_cands if c[2] in held_hashes)
    eval_cands = [c for c in eval_cands if c[4] not in held_hashes]
    repo_cands = [c for c in repo_cands if c[2] not in held_hashes]

    # ---- re-grade with the current compiler
    with cf.ThreadPoolExecutor(a.workers) as ex:
        eres = list(ex.map(lambda c: check_eval_program(c[3]), eval_cands))
        rres = list(ex.map(lambda c: check_first_passing(c[0]), repo_cands))

    reasons_e, reasons_r = Counter(), Counter()
    with open(os.path.join(a.out, "eval.jsonl"), "w") as f:
        for (bid, model, ver, code, _), (ok, why) in zip(eval_cands, eres):
            if not ok:
                reasons_e[why] += 1
                continue
            f.write(json.dumps({"source": "eval", "benchmark": bid, "model": model, "banked_version": ver,
                                "task": task_prompt(bid), "code": code}) + "\n")
            ec["kept"] += 1
    with open(os.path.join(a.out, "repo.jsonl"), "w") as f:
        for (_, code, _), (p, ok, why) in zip(repo_cands, rres):
            if not ok:
                reasons_r[why] += 1
                continue
            f.write(json.dumps({"source": "repo", "path": p, "lines": code.count("\n"), "code": code}) + "\n")
            rc["kept"] += 1
            rc["kept_lines"] += code.count("\n")

    report["eval"] = dict(ec, top_reject_reasons=reasons_e.most_common(8),
                          training_benchmarks=len({c[0] for c in eval_cands}))
    report["repo"] = dict(rc, top_reject_reasons=reasons_r.most_common(8))
    json.dump(report, open(os.path.join(a.out, "report.json"), "w"), indent=1)
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""Score one single-shot arm (tools/finetune/pilot5_baseline.sh).

Reads every result JSON under DIR and prints, over all samples: compile rate, pass rate,
the error categories, output tokens, and how many tasks passed at least once. A sample
banked under a prompt version other than --want voids the arm (exit 3).

  python3 -I tools/finetune/singleshot_score.py DIR [--want VERSION] [--name ARM] [--json]
"""
import argparse
import collections
import glob
import json
import os
import sys


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("dir")
    ap.add_argument("--want")
    ap.add_argument("--name", default="")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args()
    rows = []
    for p in glob.glob(os.path.join(a.dir, "**", "*.json"), recursive=True):
        try:
            r = json.load(open(p))
        except (json.JSONDecodeError, OSError):
            continue
        if isinstance(r, dict) and "compile_ok" in r and "id" in r:
            rows.append(r)
    if not rows:
        print(f"{a.name}: no results in {a.dir}")
        sys.exit(2)
    n = len(rows)
    comp = sum(bool(r["compile_ok"]) for r in rows)
    ok = sum(bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"]) for r in rows)
    per = collections.defaultdict(lambda: [0, 0])
    for r in rows:
        per[r["id"]][0] += bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"])
        per[r["id"]][1] += 1
    cats = collections.Counter(r.get("error_category") or "none" for r in rows)
    vers = collections.Counter(r.get("prompt_version") or "?" for r in rows)
    toks = sorted(r.get("output_tokens") or 0 for r in rows)
    out = {"arm": a.name, "samples": n, "tasks": len(per), "compile": comp, "pass": ok,
           "compile_rate": round(comp / n, 3), "pass_rate": round(ok / n, 3),
           "tasks_passed_once": sum(1 for v in per.values() if v[0]),
           "tasks_passed_always": sum(1 for v in per.values() if v[0] == v[1]),
           "errors": dict(cats), "median_output_tokens": toks[n // 2],
           "prompt_versions": dict(vers), "per_task": {k: v[0] for k, v in sorted(per.items())}}
    if a.json:
        print(json.dumps(out))
    else:
        print(f"{a.name}: {n} samples over {len(per)} tasks; compile {comp}/{n} ({100 * comp / n:.0f}%), "
              f"pass {ok}/{n} ({100 * ok / n:.0f}%); tasks passed at least once {out['tasks_passed_once']}; "
              f"errors {dict(cats)}; median output tokens {out['median_output_tokens']}; versions {dict(vers)}")
    if a.want and any(k not in (a.want, "?") for k in vers):
        print("PROMPT-VERSION MISMATCH: arm is void")
        sys.exit(3)


if __name__ == "__main__":
    main()

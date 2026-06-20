#!/usr/bin/env python3
"""Best-of-N analysis — the ceiling and REALISTIC gain of distributional-generation + exact selection.

Probe #3 of the AILANG-native harness (m-ailang-native-harness.md): qwen at temp>0 produces a
DISTRIBUTION of candidates; AILANG can pick the verified-correct one with an EXACT selector
(`ailang check` + run + contracts) where a general harness can only guess. This quantifies the lever
from any trials=N rotation, NO new GPU:

  * pass@1            — mean per-trial pass rate (the single-shot number we usually report).
  * best-of-N CEILING — fraction of benchmarks with >=1 passing trial (a PERFECT selector = the grader).
  * best-of-N REALISTIC — what a typecheck+run selector (NO reference output) actually recovers:
      a residual benchmark is RECOVERED if it has a pass AND all its failures are selector-CATCHABLE
      (compile_error / api_error / runtime_error / timeout — `ailang check`/run drops them). A failure
      that typechecks+runs but is wrong (logic_error) is UNCATCHABLE without a reference (needs
      contracts/tests) → that benchmark is RISKY (the selector might keep a wrong-but-running candidate).

The gap between ceiling and realistic = the value of stronger verification (contracts/tests), which is
exactly what the project-eval tier provides.

Usage:
  tools/eval_best_of_n.py [results_dir] [--lang ailang] [--model-substr qwen3-6]

Default results_dir: eval_results/rotation/os-rolling/agent
"""
import argparse
import collections
import glob
import json
import os
import sys

def harness_of(model):
    if "motoko" in model:
        return "motoko"
    if model.startswith("pi-"):
        return "pi"
    if "opencode" in model:
        return "opencode"
    return None


def collect(results_dir, lang, model_substr):
    # harness -> bench -> list of (compile_ok, runtime_ok, stdout_ok)
    data = collections.defaultdict(lambda: collections.defaultdict(list))
    for path in glob.glob(os.path.join(results_dir, "*.json")):
        try:
            d = json.load(open(path))
        except (ValueError, OSError):
            continue
        if d.get("eval_mode") != "agent":
            continue
        if lang and d.get("lang") != lang:
            continue
        model = d.get("model", "")
        if model_substr and model_substr not in model:
            continue
        h = harness_of(model)
        if not h:
            continue
        data[h][d.get("id")].append(
            (bool(d.get("compile_ok")), bool(d.get("runtime_ok")), bool(d.get("stdout_ok")))
        )
    return data


def _sel_score(c):
    # selector rank: runs(2) > typechecks-only(1) > neither(0)
    compile_ok, runtime_ok, _ = c
    return 2 if runtime_ok else (1 if compile_ok else 0)


def analyze(benches):
    """EXACT best-of-N: pick by the recorded compile_ok/runtime_ok (the reference-free selector),
    grade by the recorded stdout_ok (the real grader). No re-running, no proxy."""
    trials_total = sum(len(v) for v in benches.values())
    trials_pass = sum(sum(1 for c in v if c[2]) for v in benches.values())
    nb = len(benches)
    ceiling = sum(1 for v in benches.values() if any(c[2] for c in v))
    hard_fail = sorted(b for b, v in benches.items() if not any(c[2] for c in v))
    bo_pass = 0
    selector_miss = []  # selector picked runs-but-wrong over an existing correct candidate
    for b, v in benches.items():
        best = max(range(len(v)), key=lambda i: (_sel_score(v[i]), -i))
        if v[best][2]:
            bo_pass += 1
        elif any(c[2] for c in v):
            selector_miss.append(b)
    return {
        "nb": nb, "trials_total": trials_total, "trials_pass": trials_pass,
        "ceiling": ceiling, "hard_fail": hard_fail,
        "bo_exact": bo_pass, "selector_miss": selector_miss,
    }


def pct(n, d):
    return f"{100*n/d:.1f}%" if d else "—"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("results_dir", nargs="?", default="eval_results/rotation/os-rolling/agent")
    ap.add_argument("--lang", default="ailang")
    ap.add_argument("--model-substr", default="qwen3-6")
    a = ap.parse_args()
    if not os.path.isdir(a.results_dir):
        sys.exit(f"results dir not found: {a.results_dir}")
    data = collect(a.results_dir, a.lang, a.model_substr)
    if not data:
        sys.exit("no matching agent results (check --lang / --model-substr)")
    print(f"# {a.results_dir}  lang={a.lang} model~={a.model_substr}  (best-of-N analysis)")
    print(f"{'harness':9} {'pass@1':>8} {'bo-N ceiling':>13} {'bo-N EXACT(check+run)':>22} {'hard-fails':>11}")
    order = [h for h in ("motoko", "pi", "opencode") if h in data] + [h for h in sorted(data) if h not in ("motoko", "pi", "opencode")]
    for h in order:
        r = analyze(data[h])
        print(f"{h:9} {pct(r['trials_pass'], r['trials_total']):>8} "
              f"{pct(r['ceiling'], r['nb']):>13} "
              f"{pct(r['bo_exact'], r['nb']):>22} "
              f"{len(r['hard_fail']):>11}")
        if r["hard_fail"]:
            print(f"            hard-fails (no candidate passes; best-of-N can't fix): {r['hard_fail']}")
        if r["selector_miss"]:
            print(f"            selector miss (runs-but-wrong picked over a correct one; needs contracts/tests): {r['selector_miss']}")


if __name__ == "__main__":
    main()

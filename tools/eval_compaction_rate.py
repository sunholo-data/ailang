#!/usr/bin/env python3
"""Aggregate context-compaction telemetry per harness — the convergence-thrash leading indicator.

Companion to eval_failure_modes.py for the motoko mission (M-AILANG-SEMANTIC-CONTEXT). Reads the
compaction fields the harness now records per agent run (compaction_count, first_compaction_step,
compaction_level_max) and reports, per harness:

  * fire_rate   — % of runs where compaction fired at least once (compaction_count > 0).
  * avg/run     — mean compaction events per run.
  * first_step  — median step at which compaction first fired (among runs that compacted).
  * the THRASH CORRELATION — turns and pass-rate split by compacted vs not. The hypothesis:
    runs that compact take more turns and pass less (the loop erasing its own working memory).

pi's effective structural-compaction rate is ~0 (it doesn't compact by default), so "match pi"
is operationally "drive motoko's fire_rate toward 0". This is the A/B success metric for the
context-hygiene work, and a leading indicator (compaction fires before turns inflate / pass drops).

Usage:
  tools/eval_compaction_rate.py [results_dir] [--lang ailang] [--model-substr qwen3-6]
                                [--by-benchmark] [--self-test]

Default results_dir: eval_results/rotation/os-rolling/agent
NOTE: runs produced before the A1/A2 telemetry landed have no compaction fields → counted as 0.
"""
import argparse
import collections
import glob
import json
import os
import re
import statistics
import sys


def harness_of(model):
    """Map a model id to its harness, or None if not a recognised local-eval harness."""
    if "motoko" in model:
        return "motoko"
    if model.startswith("pi-"):
        return "pi"
    if "opencode" in model:
        return "opencode"
    return None


def fold(rec, run):
    """Fold one run's compaction telemetry into a harness/benchmark accumulator `rec`."""
    count = run.get("compaction_count") or 0
    fired = count > 0
    rec["_n"] += 1
    rec["events"] += count
    if fired:
        rec["fired"] += 1
        fs = run.get("first_compaction_step") or 0
        if fs > 0:
            rec.setdefault("first_steps", []).append(fs)
        lvl = run.get("compaction_level_max") or 0
        if lvl > rec["level_max"]:
            rec["level_max"] = lvl
    # Thrash-correlation buckets: turns + pass split by compacted vs not.
    turns = run.get("agent_turns") or 0
    bucket = "turns_fired" if fired else "turns_clean"
    rec.setdefault(bucket, []).append(turns)
    passed = 1 if run.get("stdout_ok") else 0
    rec["pass_fired" if fired else "pass_clean"] += passed
    rec["nfired" if fired else "nclean"] += 1


def collect(results_dir, lang, model_substr):
    by_harness = collections.defaultdict(lambda: collections.defaultdict(int))
    by_bench = collections.defaultdict(lambda: collections.defaultdict(int))
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
        fold(by_harness[h], d)
        bench = os.path.basename(path).split("_" + (lang or "ailang") + "_")[0]
        bench = re.sub(r"_trial\d+$", "", bench)
        fold(by_bench[(h, bench)], d)
    return by_harness, by_bench


def _med(xs):
    return statistics.median(xs) if xs else 0


def _rate(num, den):
    return f"{100 * num // den:>3}%" if den else "  -"


def report(by_harness, by_bench, by_benchmark):
    order = [h for h in ("motoko", "pi", "opencode") if by_harness.get(h)]
    order += [h for h in sorted(by_harness) if h not in order]
    print(f"{'harness':9} {'N':>4} {'fire_rate':>9} {'avg/run':>8} {'first_step':>10} "
          f"{'lvl_max':>7} {'turns(fired/clean)':>20} {'pass(fired/clean)':>18}")
    for h in order:
        c = by_harness[h]
        tf, tc = c.get("turns_fired", []), c.get("turns_clean", [])
        print(f"{h:9} {c['_n']:>4} {_rate(c['fired'], c['_n']):>9} "
              f"{c['events'] / c['_n'] if c['_n'] else 0:>8.1f} "
              f"{_med(c.get('first_steps', [])):>10.0f} {c['level_max']:>7} "
              f"{_med(tf):>9.0f}/{_med(tc):<10.0f} "
              f"{_rate(c['pass_fired'], c['nfired']):>8}/{_rate(c['pass_clean'], c['nclean']):<9}")
    if by_harness.get("motoko", {}).get("fired", 0) == 0:
        print("\n[note] 0 compaction events seen — either telemetry (A1/A2) isn't live in this data "
              "yet, or no run crossed the 70% threshold. Re-run after A1 dist rebuild + A2 binary.")
    if by_benchmark:
        print("\nper-benchmark (motoko, fire_rate):")
        rows = [(b, c) for (h, b), c in by_bench.items() if h == "motoko"]
        for b, c in sorted(rows, key=lambda r: -(r[1]['fired'] / r[1]['_n'] if r[1]['_n'] else 0)):
            print(f"  {b:30} fire_rate={_rate(c['fired'], c['_n'])} avg/run={c['events']/c['_n']:.1f}")


def self_test():
    acc = collections.defaultdict(int)
    runs = [
        {"compaction_count": 0, "agent_turns": 5, "stdout_ok": True},
        {"compaction_count": 3, "first_compaction_step": 12, "compaction_level_max": 85,
         "agent_turns": 42, "stdout_ok": False},
        {"compaction_count": 1, "first_compaction_step": 20, "compaction_level_max": 70,
         "agent_turns": 30, "stdout_ok": True},
    ]
    for r in runs:
        fold(acc, r)
    assert acc["_n"] == 3, acc["_n"]
    assert acc["fired"] == 2, acc["fired"]
    assert acc["events"] == 4, acc["events"]
    assert acc["level_max"] == 85, acc["level_max"]
    assert _med(acc["first_steps"]) == 16, _med(acc["first_steps"])  # median(12,20)
    # thrash correlation: fired runs averaged more turns than clean
    assert _med(acc["turns_fired"]) == 36 and _med(acc["turns_clean"]) == 5
    assert harness_of("motoko-local-qwen3-6-35b-a3b-mxfp8") == "motoko"
    assert harness_of("pi-qwen3-6-35b-a3b-mxfp8") == "pi"
    print("self-test: OK (fold aggregation + thrash buckets + harness mapping)")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("results_dir", nargs="?", default="eval_results/rotation/os-rolling/agent")
    ap.add_argument("--lang", default="ailang")
    ap.add_argument("--model-substr", default="qwen3-6")
    ap.add_argument("--by-benchmark", action="store_true")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args()
    if a.self_test:
        self_test()
        return
    if not os.path.isdir(a.results_dir):
        sys.exit(f"results dir not found: {a.results_dir}")
    by_harness, by_bench = collect(a.results_dir, a.lang, a.model_substr)
    if not by_harness:
        sys.exit("no matching agent results (check --lang / --model-substr)")
    print(f"# {a.results_dir}  lang={a.lang} model~={a.model_substr}  (compaction-fire-rate)")
    report(by_harness, by_bench, a.by_benchmark)


if __name__ == "__main__":
    main()

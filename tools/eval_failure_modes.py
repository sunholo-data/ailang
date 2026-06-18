#!/usr/bin/env python3
"""Segment agent-eval failures by MODE — disengage vs grind-wrong — per harness.

Existing eval tooling groups failures by `error_category` only. For the motoko mission the
qualitatively important split is engagement, which `error_category` does not capture:

  * DISENGAGE   — failed with <= N tool calls (model answered in prose / inspected once and
                  stopped; never really attempted a solution). motoko's signature failure.
  * GRIND_WRONG — failed with  > N tool calls (model engaged and iterated, but the solution
                  is incorrect). A correctness problem, not an engagement problem.

This locates the motoko<->pi gap by mode before spending any GPU on a fix.

Usage:
  tools/eval_failure_modes.py [results_dir] [--lang ailang] [--model-substr qwen3-6]
                              [--disengage-threshold 2] [--by-benchmark] [--self-test]

Default results_dir: eval_results/rotation/os-rolling/agent
Reads the agent result JSONs (fields: model, lang, eval_mode, stdout_ok, agent_tool_calls).
"""
import argparse
import collections
import glob
import json
import os
import re
import sys

PASS, DISENGAGE, GRIND = "pass", "disengage", "grind_wrong"


def classify(stdout_ok, agent_tool_calls, disengage_threshold=2):
    """Pure classifier — the unit under test.

    pass        : the run produced correct stdout.
    disengage   : failed having made <= threshold tool calls (no real solution attempt).
    grind_wrong : failed having made  > threshold tool calls (engaged but incorrect).
    """
    if stdout_ok:
        return PASS
    if (agent_tool_calls or 0) <= disengage_threshold:
        return DISENGAGE
    return GRIND


def harness_of(model):
    """Map a model id to its harness, or None if not a recognised local-eval harness."""
    if "motoko" in model:
        return "motoko"
    if model.startswith("pi-"):
        return "pi"
    if "opencode" in model:
        return "opencode"
    return None


def collect(results_dir, lang, model_substr, threshold):
    """Return {harness: Counter(pass/disengage/grind_wrong/_n)} and a per-(harness,benchmark) map."""
    by_harness = collections.defaultdict(collections.Counter)
    by_bench = collections.defaultdict(collections.Counter)
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
        mode = classify(d.get("stdout_ok"), d.get("agent_tool_calls"), threshold)
        by_harness[h][mode] += 1
        by_harness[h]["_n"] += 1
        bench = os.path.basename(path).split("_" + (lang or "ailang") + "_")[0]
        bench = re.sub(r"_trial\d+$", "", bench)  # merge trials per benchmark
        by_bench[(h, bench)][mode] += 1
    return by_harness, by_bench


def _pct(c, key):
    n = c["_n"]
    return f"{c[key]:>4} ({100 * c[key] // n:>2}%)" if n else "   0 ( 0%)"


def report(by_harness, by_bench, by_benchmark):
    order = [h for h in ("motoko", "pi", "opencode") if by_harness.get(h)]
    order += [h for h in sorted(by_harness) if h not in order]
    print(f"{'harness':9} {'N':>4} {'pass':>9} {'disengage':>11} {'grind_wrong':>12}")
    for h in order:
        c = by_harness[h]
        print(f"{h:9} {c['_n']:>4} {_pct(c, PASS)} {_pct(c, DISENGAGE)} {_pct(c, GRIND)}")
    if "motoko" in by_harness and "pi" in by_harness:
        m, p = by_harness["motoko"], by_harness["pi"]

        def rate(c, k):
            return 100 * c[k] / c["_n"] if c["_n"] else 0
        print(
            f"\nmotoko↔pi gap: pass {rate(p, PASS) - rate(m, PASS):+.0f}pp | "
            f"disengage {rate(m, DISENGAGE) - rate(p, DISENGAGE):+.0f}pp | "
            f"grind_wrong {rate(m, GRIND) - rate(p, GRIND):+.0f}pp"
        )
    if by_benchmark:
        print("\nper-benchmark (motoko only, failures):")
        rows = [(b, c) for (h, b), c in by_bench.items() if h == "motoko" and (c[DISENGAGE] + c[GRIND])]
        for b, c in sorted(rows, key=lambda r: -(r[1][DISENGAGE] + r[1][GRIND])):
            print(f"  {b:30} disengage={c[DISENGAGE]} grind_wrong={c[GRIND]} pass={c[PASS]}")


def self_test():
    cases = [
        ((True, 0, 2), PASS),
        ((True, 50, 2), PASS),
        ((False, 0, 2), DISENGAGE),
        ((False, 2, 2), DISENGAGE),
        ((False, None, 2), DISENGAGE),
        ((False, 3, 2), GRIND),
        ((False, 33, 2), GRIND),
        ((False, 5, 10), DISENGAGE),  # threshold sensitivity
    ]
    for (args, expected) in cases:
        got = classify(*args)
        assert got == expected, f"classify{args} = {got}, expected {expected}"
    assert harness_of("motoko-local-qwen3-6-35b-a3b-mxfp8") == "motoko"
    assert harness_of("pi-qwen3-6-35b-a3b-mxfp8") == "pi"
    assert harness_of("opencode-qwen3-6-35b-a3b-mxfp8") == "opencode"
    assert harness_of("claude-sonnet-4-6") is None
    print("self-test: OK (%d classifier cases + harness mapping)" % len(cases))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("results_dir", nargs="?", default="eval_results/rotation/os-rolling/agent")
    ap.add_argument("--lang", default="ailang")
    ap.add_argument("--model-substr", default="qwen3-6")
    ap.add_argument("--disengage-threshold", type=int, default=2)
    ap.add_argument("--by-benchmark", action="store_true")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args()
    if a.self_test:
        self_test()
        return
    if not os.path.isdir(a.results_dir):
        sys.exit(f"results dir not found: {a.results_dir}")
    by_harness, by_bench = collect(a.results_dir, a.lang, a.model_substr, a.disengage_threshold)
    if not by_harness:
        sys.exit("no matching agent results (check --lang / --model-substr)")
    print(f"# {a.results_dir}  lang={a.lang} model~={a.model_substr} disengage<=tool_calls {a.disengage_threshold}")
    report(by_harness, by_bench, a.by_benchmark)


if __name__ == "__main__":
    main()

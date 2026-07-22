#!/usr/bin/env python3
"""M-EVAL-FMT-WEAKMODEL-AB — M3 analysis helper (fmt-hook ON vs OFF on claude-haiku-4-5).

Reads two banked agent/ arm directories (ON and OFF) and emits the full frozen metric
table from the prereg (m-eval-fmt-weakmodel-ab-prereg.md §5–§6):

  1. PRIMARY  — per-benchmark + overall pass rates (Wilson score 95% CI) and the ON−OFF
                pass-rate delta with the Newcombe/Wilson-based 95% CI of the difference of
                two independent binomial proportions.
  2. GATE     — hook-reality / treatment integrity: ON-arm fmt exit-0 (formatted) vs
                defer/refusal counts → treatment-delivery rate; confirms OFF banks ZERO
                fmt_hook_events (arm-gating integrity).
  3. SECONDARY— compile-stuck incidence per arm (Wilson CI); edits-to-first-green PROXY
                (mean .ail Write+Edit count ± 95% CI) — labeled a proxy; green-stability
                reported NOT-COMPUTABLE (no per-edit typecheck stream in banked data).

A run "passes" iff compile_ok && runtime_ok && stdout_ok (prereg §5.1, harness Success).

Stdlib-only. Wilson and Newcombe are closed-form (no scipy). z = 1.959964 for 95%.

Usage:
  tools/analyze_fmt_ab.py <on_agent_dir> <off_agent_dir> [--lang ailang]
"""
import argparse
import collections
import glob
import json
import math
import os
import sys

Z = 1.959963984540054  # 95% two-sided normal quantile


def wilson(k, n, z=Z):
    """Wilson score interval for a binomial proportion k/n. Returns (p, lo, hi)."""
    if n == 0:
        return (float("nan"), float("nan"), float("nan"))
    p = k / n
    z2 = z * z
    denom = 1.0 + z2 / n
    center = (p + z2 / (2 * n)) / denom
    half = (z * math.sqrt((p * (1 - p) + z2 / (4 * n)) / n)) / denom
    return (p, center - half, center + half)


def newcombe_diff(k1, n1, k2, n2, z=Z):
    """Newcombe method 10: 95% CI for p1 - p2 from two independent Wilson intervals.
    Arm 1 = ON, arm 2 = OFF. Returns (delta, lo, hi)."""
    p1, l1, u1 = wilson(k1, n1, z)
    p2, l2, u2 = wilson(k2, n2, z)
    delta = p1 - p2
    lo = delta - math.sqrt((p1 - l1) ** 2 + (u2 - p2) ** 2)
    hi = delta + math.sqrt((u1 - p1) ** 2 + (p2 - l2) ** 2)
    return (delta, lo, hi)


def mean_ci(vals, z=Z):
    """Mean +/- 95% normal CI (t approximated by z; small-N caveat noted in report)."""
    n = len(vals)
    if n == 0:
        return (float("nan"), float("nan"), float("nan"))
    m = sum(vals) / n
    if n == 1:
        return (m, m, m)
    var = sum((v - m) ** 2 for v in vals) / (n - 1)
    se = math.sqrt(var / n)
    return (m, m - z * se, m + z * se)


def load_arm(agent_dir, lang):
    """Return dict: bench -> list of run dicts with derived fields."""
    benches = collections.defaultdict(list)
    for path in sorted(glob.glob(os.path.join(agent_dir, "*.json"))):
        try:
            d = json.load(open(path))
        except (ValueError, OSError):
            continue
        if d.get("eval_mode") != "agent":
            continue
        if lang and d.get("lang") != lang:
            continue
        passed = bool(d.get("compile_ok")) and bool(d.get("runtime_ok")) and bool(d.get("stdout_ok"))
        transcript = d.get("agent_transcript") or ""
        we = transcript.count("[TOOL] Write") + transcript.count("[TOOL] Edit")
        evs = d.get("fmt_hook_events") or []
        benches[d.get("id")].append({
            "passed": passed,
            "compile_ok": bool(d.get("compile_ok")),
            "error_category": d.get("error_category"),
            "we_proxy": we,
            "hook_events": evs,
            "trial": d.get("trial"),
            "prompt_version": d.get("prompt_version"),
        })
    return benches


def arm_totals(benches):
    passes = sum(1 for v in benches.values() for r in v if r["passed"])
    n = sum(len(v) for v in benches.values())
    return passes, n


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("on_dir")
    ap.add_argument("off_dir")
    ap.add_argument("--lang", default="ailang")
    a = ap.parse_args()
    for d in (a.on_dir, a.off_dir):
        if not os.path.isdir(d):
            sys.exit(f"dir not found: {d}")

    on = load_arm(a.on_dir, a.lang)
    off = load_arm(a.off_dir, a.lang)
    all_benches = sorted(set(on) | set(off))

    print("=" * 78)
    print("M-EVAL-FMT-WEAKMODEL-AB  M3 analysis  (claude-haiku-4-5, N=5/arm/benchmark)")
    print(f"ON  dir: {a.on_dir}")
    print(f"OFF dir: {a.off_dir}")
    print("=" * 78)

    # --- prompt_version integrity (prereg §3: must be identical across arms) ---
    pvs = set()
    for src in (on, off):
        for v in src.values():
            for r in v:
                if r["prompt_version"]:
                    pvs.add(r["prompt_version"])
    print(f"\nprompt_version(s) observed across both arms: {sorted(pvs)}"
          + ("  [OK: identical]" if len(pvs) == 1 else "  [WARN: differ!]"))

    # --- 1. PRIMARY: per-benchmark pass rates + delta ---
    print("\n--- 1. PRIMARY: per-benchmark pass rate (passes/5) ---")
    print(f"{'benchmark':24} {'ON':>7} {'OFF':>7} {'delta':>8}   {'delta 95% CI (Newcombe)':>28}")
    for b in all_benches:
        k1 = sum(1 for r in on.get(b, []) if r["passed"]); n1 = len(on.get(b, []))
        k2 = sum(1 for r in off.get(b, []) if r["passed"]); n2 = len(off.get(b, []))
        dl, lo, hi = newcombe_diff(k1, n1, k2, n2)
        print(f"{b:24} {k1}/{n1:<5} {k2}/{n2:<5} {dl:+.3f}   [{lo:+.3f}, {hi:+.3f}]")

    # overall
    ok_p, ok_n = arm_totals(on)
    of_p, of_n = arm_totals(off)
    p_on, lo_on, hi_on = wilson(ok_p, ok_n)
    p_off, lo_off, hi_off = wilson(of_p, of_n)
    dl, dlo, dhi = newcombe_diff(ok_p, ok_n, of_p, of_n)
    print("\n--- OVERALL pass rate (pooled, Wilson 95% CI) ---")
    print(f"  ON : {ok_p}/{ok_n} = {p_on:.4f}  Wilson95 [{lo_on:.4f}, {hi_on:.4f}]")
    print(f"  OFF: {of_p}/{of_n} = {p_off:.4f}  Wilson95 [{lo_off:.4f}, {hi_off:.4f}]")
    print(f"  DELTA (ON-OFF) = {dl:+.4f}  Newcombe95 [{dlo:+.4f}, {dhi:+.4f}]")

    # --- 2. GATE: hook reality / treatment integrity ---
    print("\n--- 2. GATE: hook-reality / treatment-delivery (ON arm) ---")
    status_counts = collections.Counter()
    total_ail_edits = 0
    for v in on.values():
        for r in v:
            for e in r["hook_events"]:
                status_counts[e.get("status")] += 1
                total_ail_edits += 1
    formatted = status_counts.get("formatted", 0)
    print(f"  ON-arm captured .ail edits (fmt_hook_events): {total_ail_edits}")
    print(f"  status breakdown: {dict(status_counts)}")
    if total_ail_edits:
        td, tdlo, tdhi = wilson(formatted, total_ail_edits)
        print(f"  treatment-delivery rate (exit-0 formatted / edits): "
              f"{formatted}/{total_ail_edits} = {td:.4f}  Wilson95 [{tdlo:.4f}, {tdhi:.4f}]")
        refusal = total_ail_edits - formatted
        print(f"  refusal/defer count: {refusal}  (prereg fail-closed baseline ~8%)")
    off_hook = sum(len(r["hook_events"]) for v in off.values() for r in v)
    print(f"  OFF-arm fmt_hook_events (must be 0 for arm-gating integrity): {off_hook}"
          + ("  [OK]" if off_hook == 0 else "  [FAIL: leak!]"))

    # --- 3. SECONDARY: convergence ---
    print("\n--- 3. SECONDARY: convergence ---")
    for name, src in (("ON", on), ("OFF", off)):
        stuck = sum(1 for v in src.values() for r in v if not r["compile_ok"])
        n = sum(len(v) for v in src.values())
        p, lo, hi = wilson(stuck, n)
        print(f"  compile-stuck incidence [{name}]: {stuck}/{n} = {p:.4f}  Wilson95 [{lo:.4f}, {hi:.4f}]")
    for name, src in (("ON", on), ("OFF", off)):
        vals = [r["we_proxy"] for v in src.values() for r in v]
        m, lo, hi = mean_ci(vals)
        print(f"  edits-to-first-green PROXY (mean Write+Edit) [{name}]: "
              f"{m:.3f}  95% [{lo:.3f}, {hi:.3f}]  (PROXY — see limitations)")
    print("  green-stability rate: NOT-COMPUTABLE from banked data "
          "(no per-edit typecheck stream; agent_transcript is tool-name summary only).")

    # --- VERDICT (§6) ---
    print("\n--- VERDICT (prereg §6) ---")
    delivered = total_ail_edits > 0 and (formatted / total_ail_edits) >= 0.5
    if not delivered:
        verdict = "VOID / UNEVALUABLE (treatment-delivery low)"
    elif dl >= 0.10 and dlo > 0:
        verdict = "H1 SUPPORTED (delta >= +0.10 AND CI excludes 0)"
    elif dl <= -0.10 and dhi < 0:
        verdict = "HARM (delta <= -0.10 AND CI excludes 0)"
    else:
        verdict = "NULL (published) — no meaningful positive difference"
    print(f"  overall delta = {dl:+.4f}, Newcombe95 CI [{dlo:+.4f}, {dhi:+.4f}]")
    print(f"  treatment delivered = {delivered} ({formatted}/{total_ail_edits} formatted)")
    print(f"  => {verdict}")


if __name__ == "__main__":
    main()

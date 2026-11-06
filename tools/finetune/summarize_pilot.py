#!/usr/bin/env python3
"""Score the pilot-1 A/B arms on the held-out 24 and apply the kill criterion.

Arms (dirs under the eval output root): A_std, B_std, C_std, B_agent, C_agent.
Arm A agent comes from tools/finetune/baseline_armA.json (Phase 0, same 24).

Metrics per arm: pass rate, core/stretch, first-attempt passes, protocol misses
(no tool call, or graded file still the template), median output tokens per
pass, and agent wall-clock per verified success (sum of banked durations /
passes; non_agentic runs bank duration 0, so this UNDER-counts their cost).

Kill criterion (design doc): B must at least match A on pass rate, per mode.
Writes summary.json and summary.md next to the arms.
"""
import glob
import json
import os
import statistics
import sys

HERE = os.path.dirname(os.path.abspath(__file__))


def arm_stats(d, core):
    rows = []
    for p in glob.glob(os.path.join(d, "**", "*.json"), recursive=True):
        try:
            r = json.load(open(p))
        except (OSError, ValueError):
            continue
        if isinstance(r, dict) and "compile_ok" in r and "model" in r:
            rows.append(r)
    if not rows:
        return None
    ok = lambda r: bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"])
    passes = [r for r in rows if ok(r)]
    miss = sum(1 for r in rows if not ok(r) and (r.get("error_category") == "non_agentic"
                                                  or "TODO: Add your solution" in (r.get("code") or "")))
    outp = sorted(r.get("output_tokens") or 0 for r in passes)
    c = [r for r in rows if r["id"] in core]
    s = [r for r in rows if r["id"] not in core]
    return {"runs": len(rows), "benchmarks": len({r["id"] for r in rows}), "pass": len(passes),
            "pass_rate": round(len(passes) / len(rows), 3),
            "core": f"{sum(map(ok, c))}/{len(c)}", "stretch": f"{sum(map(ok, s))}/{len(s)}",
            "first_attempt": sum(1 for r in rows if r.get("first_attempt_ok")),
            "protocol_miss": miss,
            "median_output_tokens_per_pass": outp[len(outp) // 2] if outp else None,
            "minutes_per_verified_success": round(sum((r.get("duration_ms") or 0) for r in rows) / 60000 / max(1, len(passes)), 1),
            "prompt_versions": sorted({r.get("prompt_version") or "?" for r in rows})}


def main():
    root = sys.argv[1]
    h = json.load(open(os.path.join(HERE, "heldout.json")))
    core = set(h["core"])
    arms = {n: arm_stats(os.path.join(root, n), core)
            for n in ("A_std", "B_std", "C_std", "B_agent", "C_agent", "A_agent2", "B_agent2", "C_agent2")}
    arms = {k: v for k, v in arms.items() if v is not None or not k.endswith("2")}
    base = json.load(open(os.path.join(HERE, "baseline_armA.json")))["results"]["agent"]
    arms["A_agent"] = dict(base, core=base.get("core_pass"), stretch=base.get("stretch_pass"),
                           source="baseline_armA.json (Phase 0)")
    verdict = {}
    # agent2 = re-run with the session-protocol gate disarmed (5dc3ba42b); the first
    # agent arms ran with every write refused until the model acked, so they are VOID.
    for mode in ("std", "agent", "agent2"):
        a, b = arms.get(f"A_{mode}"), arms.get(f"B_{mode}")
        if a and b:
            verdict[mode] = {"A": a["pass_rate"], "B": b["pass_rate"],
                             "kill_criterion_met": b["pass_rate"] >= a["pass_rate"],
                             "protocol_miss_A_B": [a.get("protocol_miss"), b.get("protocol_miss")]}
    out = {"arms": arms, "verdict": verdict}
    json.dump(out, open(os.path.join(root, "summary.json"), "w"), indent=1)
    cols = ["runs", "pass", "pass_rate", "core", "stretch", "first_attempt", "protocol_miss",
            "median_output_tokens_per_pass", "minutes_per_verified_success"]
    lines = ["| arm | " + " | ".join(cols) + " |", "|" + "---|" * (len(cols) + 1)]
    for n in ("A_std", "B_std", "C_std", "A_agent", "B_agent", "C_agent", "A_agent2", "B_agent2", "C_agent2"):
        if n.endswith("2") and n not in arms:
            continue
        st = arms.get(n)
        lines.append(f"| {n} | " + (" | ".join(str(st.get(k, "")) for k in cols) if st else "not run") + " |")
    lines += ["", "Kill criterion (B >= A on pass rate): " + json.dumps(verdict),
              "Note: A_agent/B_agent/C_agent ran under the session-protocol gate (every write refused until the model acked) - VOID; read the *_agent2 arms."]
    open(os.path.join(root, "summary.md"), "w").write("\n".join(lines) + "\n")
    print("\n".join(lines))


if __name__ == "__main__":
    main()

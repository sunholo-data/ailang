# M-EVAL-FMT-WEAKMODEL-AB — M3 Analysis Results

**Milestone**: M3_ANALYSIS (pure read-only aggregation; no GPU, no eval runs, no compiler changes).
**Date**: 2026-07-22.
**Prereg** (frozen spec obeyed exactly): [`m-eval-fmt-weakmodel-ab-prereg.md`](m-eval-fmt-weakmodel-ab-prereg.md) §5 (metrics) + §6 (confidence + refutation threshold).
**Model**: `claude-haiku-4-5` (agent mode, `claude` CLI). **N = 5** runs/arm/benchmark, 6 benchmarks, 2 arms = **60 scored runs**.
**Banked data** (read-only, `eval_results/fmt_ab_haiku_M2b/{on,off}/agent/`, 30 files each): the ONLY difference between arms is the `-fmt-hook` flag (prereg §4). `prompt_version` is `v0.16.3` in **both** arms (identical — prereg §3 integrity check passes).

---

## Headline verdict (prereg §6)

**NULL (published) — no meaningful positive difference, with treatment PROVEN delivered.**

- Overall pass-rate delta (ON − OFF) = **+0.033** (30/30 vs 29/30).
- 95% CI of the delta (Newcombe/Wilson) = **[−0.083, +0.167]** — **includes 0**, and the point delta is **< +0.10**.
- Per prereg §6, delta CI including 0 AND point delta < +0.10 ⇒ **NULL**.
- Treatment-delivery rate = **32/32 = 100%** exit-0 (formatted); this is NOT a treatment-delivery failure, so the verdict is a **true null**, not "unevaluable / void".

This is the expected **null-at-ceiling** outcome: haiku is near-ceiling on this easy→medium set (58/60 runs pass across both arms), so the fmt hook has almost no headroom to help. The single OFF-arm failure was a `cli_args` `compile_error`; the hook did not create a statistically distinguishable advantage.

---

## 1. PRIMARY — pass-rate delta (prereg §5.1)

A run "passes" iff `compile_ok && runtime_ok && stdout_ok` (harness `Success`).

### Per-benchmark pass rate (passes / 5)

| Benchmark | ON | OFF | delta | delta 95% CI (Newcombe) |
|---|---|---|---|---|
| adt_option | 5/5 | 5/5 | +0.000 | [−0.434, +0.434] |
| cli_args | 5/5 | **4/5** | +0.200 | [−0.264, +0.624] |
| fizzbuzz | 5/5 | 5/5 | +0.000 | [−0.434, +0.434] |
| gcd_lcm | 5/5 | 5/5 | +0.000 | [−0.434, +0.434] |
| higher_order_functions | 5/5 | 5/5 | +0.000 | [−0.434, +0.434] |
| json_parse | 5/5 | 5/5 | +0.000 | [−0.434, +0.434] |

The only non-tied cell is `cli_args` (ON 5/5 vs OFF 4/5); its per-benchmark delta CI [−0.264, +0.624] includes 0 (N=5/arm is far too small to resolve a single-run difference). No benchmark shows a significant effect in either direction.

### Overall pass rate (pooled 30 runs/arm, Wilson score 95% CI)

| Arm | passes/N | pass rate | Wilson 95% CI |
|---|---|---|---|
| ON | 30/30 | 1.0000 | [0.8865, 1.0000] |
| OFF | 29/30 | 0.9667 | [0.8333, 0.9941] |

**Delta (ON − OFF) = +0.0333, Newcombe 95% CI = [−0.0834, +0.1667].**

The CI includes 0 and the point estimate (+3.3 pp) is below the frozen +10 pp threshold ⇒ **NULL**.

### Cross-check vs existing tooling

`tools/eval_best_of_n.py` **cannot ingest these runs**: its `harness_of()` whitelists only local harnesses (`motoko`/`pi`/`opencode`) and returns `None` for cloud `claude-haiku-4-5`, so it filters every file out (`no matching agent results`, verified). It was therefore NOT forked; the reusable cross-check is the **harness's own banked `model_rollup`** in each arm's `summary.json`:

- ON `summary.json`: `pass_at_1: 1` (30/30) — **matches** this analysis.
- OFF `summary.json`: `pass_at_1: 0.9667` (29/30) — **matches** this analysis.

Full agreement between the independent aggregator (`tools/analyze_fmt_ab.py`) and the harness rollup. The new helper adds the Wilson/Newcombe CIs the rollup lacks.

---

## 2. GATE — hook-reality / treatment integrity (prereg §5.3)

| Quantity | Value |
|---|---|
| ON-arm captured `.ail` edits (`fmt_hook_events` entries) | **32** |
| status = `formatted` (fmt exit 0 = treatment delivered) | **32** |
| status = defer (exit 3) / refusal / error | **0** |
| **Treatment-delivery rate** (exit-0 / edits) | **32/32 = 100%**, Wilson 95% [0.893, 1.000] |
| Observed refusal rate | **0%** (vs prereg ~8% fail-closed baseline — well below) |
| OFF-arm `fmt_hook_events` (arm-gating integrity) | **0** — PASS (no leak) |

Every single `.ail` edit in the ON arm was canonically formatted (exit 0); there were zero defers and zero refusals — cleaner than the ~8% fail-closed baseline anticipated in the prereg. The OFF arm banked **zero** hook events (arm gating via the ON-only `.claude/` dir held). The treatment was genuinely and fully delivered ⇒ the null is a **true formatter null**, not a measurement/void artifact (prereg §6 void clause does NOT trigger).

Note: 32 formatted events across 30 ON runs ≈ ~1.07 `.ail` edits/run — corroborating that haiku converges in essentially one Write on this set (near-zero convergence headroom, see §3).

---

## 3. SECONDARY — convergence (prereg §5.2), with data-availability limits

### Compile-stuck incidence (FULLY computable)

Fraction of runs ending `compile_ok == false` (never reached a compiling solution).

| Arm | stuck/N | rate | Wilson 95% CI |
|---|---|---|---|
| ON | 0/30 | 0.0000 | [0.0000, 0.1135] |
| OFF | 1/30 | 0.0333 | [0.0059, 0.1667] |

The one OFF stuck run is the `cli_args` `compile_error`. CIs overlap heavily; no meaningful difference. This corroborates the null: neither arm exhibits compile-stuck spiraling on this set — the regime H1 targets (weak-model spirals) essentially did not occur, so the hook had nothing to rescue.

### Edits-to-first-green — PROXY only (mean `.ail` Write+Edit count)

⚠ **PROXY, not the prereg metric.** The banked data has **no per-edit `typecheck` stream**, so the true "edits before first compile-clean state" is not computable. This proxy counts `[TOOL] Write` + `[TOOL] Edit` markers in the summarized `agent_transcript` (tool names only) — it cannot identify *which* edit first went green.

| Arm | mean Write+Edit | 95% CI |
|---|---|---|
| ON | 1.133 | [0.929, 1.338] |
| OFF | 1.233 | [1.030, 1.437] |

Both arms average ~1 edit — haiku near-ceiling essentially writes the solution once. The tiny ON<OFF gap is not interpretable given it's a proxy and the CIs overlap. This near-1 count itself demonstrates convergence has near-zero headroom on this set, independently corroborating the null.

### Green-stability rate — NOT-COMPUTABLE ✗

**Cannot be computed from the banked data.** Green-stability (fraction of post-first-green edits that preserve the compiling state) requires a per-edit compile/typecheck result for every edit. The banked runs contain **no per-edit `typecheck` stream** and **no companion jsonl/stream files**; `agent_transcript` is a summarized string with tool *names* only. Per mission calibrated-status discipline this is reported as not-computable rather than fabricated. (At haiku's ~1-Write convergence there are essentially no post-first-green edits to measure anyway, so the metric has near-zero headroom regardless.)

---

## 4. Data availability & limitations (explicit, per calibrated-status discipline)

| Metric | Status | Basis |
|---|---|---|
| Pass-rate delta + Wilson/Newcombe CIs (PRIMARY) | ✓ FULLY computed | `compile_ok/runtime_ok/stdout_ok` per run |
| Treatment-delivery / hook-reality (GATE) | ✓ FULLY computed | `fmt_hook_events` (status per `.ail` edit) |
| OFF-arm arm-gating integrity | ✓ FULLY computed (0 events) | `fmt_hook_events` empty in OFF |
| Compile-stuck incidence + CI | ✓ FULLY computed | `compile_ok` per run |
| Edits-to-first-green | ⚠ PROXY only | transcript `[TOOL] Write/Edit` counts; no per-edit typecheck |
| Green-stability rate | ✗ NOT-COMPUTABLE | no per-edit typecheck stream / no stream files banked |

**Why the convergence metrics are limited:** the M2b banking captured whole-run outcomes + the out-of-band fmt-hook sink, but NOT a per-edit typecheck stream (that stream exists only for the DOCX/`analyze_stuck` path, not this cloud-claude agent path). This limits the DOCX convergence metrics to the coarse proxies above. It does **not** affect the PRIMARY verdict, which rests entirely on fully-computable pass-rate + delivery data.

**Statistical caveat:** N=30/arm at a near-ceiling proportion makes even the pooled delta CI wide ([−0.083, +0.167]); this experiment can confidently rule out a *large* positive effect (≥ the +10 pp threshold is not supported) but cannot resolve a small (few-pp) effect. That is the correct, honest reading of a null at ceiling — the hook is not *harmful* (ON did not regress the easy control tasks; the fizzbuzz/adt guards held at 5/5) and is not *demonstrably helpful* on this weak-model-but-still-near-ceiling set.

---

## 5. Reproduce

```
tools/analyze_fmt_ab.py \
  eval_results/fmt_ab_haiku_M2b/on/agent \
  eval_results/fmt_ab_haiku_M2b/off/agent
```

Stdlib-only (Wilson + Newcombe closed-form, z=1.959964); re-runnable against any two arm dirs by absolute path.

---

## Verdict summary (prereg §6)

- **overall delta = +0.033**, Newcombe 95% CI **[−0.083, +0.167]** (includes 0; point < +0.10)
- **treatment delivered = TRUE** (32/32 = 100% formatted; 0% refusal vs ~8% baseline; OFF gating clean)
- **⇒ NULL (published).** Not H1-supported, not harm, not void/unevaluable.

The fmt PostToolUse hook, though fully and cleanly delivered, produced no meaningful pass-rate or convergence advantage for `claude-haiku-4-5` on this frozen easy→medium benchmark set — consistent with the model already converging near-ceiling in ~1 edit with negligible syntax-drift spiraling to correct.

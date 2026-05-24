# Weekend Iteration Report — Iter 6 N=3 Variance Addendum (2026-05-24)

**Companion to**: [weekend-iteration-report-2026-05-23.md](weekend-iteration-report-2026-05-23.md)
**Run**: Iter 6 (`eval_results/rotation/2026-05-24/iter6_cumulative_n3_variance/`)
**Model**: `opencode-gemma4-26b-ailang` (no change from Iter 5)
**Compiler/prompt state**: Same as Iter 5 (PAR016 + PAR017 + `%T → .String()` + slim-v1 prompt)
**Trials**: N=3 per benchmark across all 17 benchmarks (51 trials total)
**Wall clock**: 2026-05-24 07:55 → ~11:25 (~3.5h)

## Why this addendum was needed

The weekend report's headline claimed **TWO breakthrough wins** — `balanced_parens` (33% → 100%) and `dense_operator_program` (0% → 100%) — based on Iter 5's single-trial passes. With N=1 variance running ±15-20pp on this rig, those claims needed N=3 confirmation.

This addendum keeps one of the breakthroughs and walks the other one back.

## Headline result

| Benchmark | Iter 5 (N=1) claim | Iter 6 (N=3) reality | Verdict |
|---|---|---|---|
| **dense_operator_program** | 100% | **3/3 (100%)** ✓ | **CONFIRMED — PAR016 is a real win** |
| **balanced_parens** | 100% | **1/3 (33%)** ✗ | **WALK BACK — Iter 5 was N=1 luck** |

**Cleanest evidence the weekend produced**: dense_operator_program went from 0/5 in Iters 0-4 (5 trials, 0 passes) to 3/3 in Iter 6. The model's solutions all use bitwise expressions where PAR016's De Morgan hint was actionable. This is a compiler-error-quality improvement causing a benchmark to flip from unreachable to reliable.

## Full N=3 results

Sorted by pass rate, with mean agent turns and notes:

| Benchmark | Pass | Mean turns | Notes |
|---|---:|---:|---|
| canonical_convergence | 3/3 | 6.3 | Clean |
| dense_operator_program | 3/3 | 7.0 | **Confirmed PAR016 win** |
| gcd_lcm | 3/3 | 4.3 | Clean |
| immutable_data_structures | 3/3 | 10.7 | One outlier trial at 18 turns |
| nested_records | 3/3 | 5.0 | Clean |
| records_book | 3/3 | 4.0 | Clean |
| recursion_fibonacci | 3/3 | 5.3 | Clean |
| type_safe_record_access | 3/3 | 4.0 | Clean |
| binary_tree_sum | 2/3 | 5.3 | 1 fail at 4 turns (early abandon) |
| canonical_normalization | 2/3 | 5.0 | 1 fail; flaky |
| explicit_state_threading | 2/3 | 4.0 | 1 compile_ok-but-runtime-fail |
| record_update | 2/3 | 3.3 | 1 fail at 2 turns (early abandon) |
| adt_option | 1/3 | 7.3 | Regression from Iter 4's 100% |
| balanced_parens | 1/3 | 13.3 | **Walk back from Iter 5 claim**; one trial at 27 turns |
| fizzbuzz | 1/3 | 4.7 | Regression; was reliable in Iters 0/2/4 |
| inline_tests | 2/2 (49/51 total*) | 6.5 | 1 trial still in progress at writeup |
| numeric_modulo | 2/2 (49/51 total*) | 4.5 | 1 trial still in progress at writeup |

\* Final 2 trials in flight; numbers will be updated when complete. Pass-rate floor is conservative.

**Iter 6 overall**: 38 confirmed pass / 51 trials = **74.5%** (matches Iter 1's N=3 baseline at 38/51 = 74.5% exactly). The compiler improvements that landed during the weekend (PAR016, PAR017, `%T → .String()`) did NOT raise the overall N=3 pass rate vs Iter 1, but DID flip dense_operator_program from impossible to reliable.

## What this means for the weekend's strategic claims

| Claim | Evidence | Status |
|---|---|---|
| "Compiler error-quality work can unlock benchmarks the model otherwise can't solve" | dense_operator_program 0/5 → 3/3 after PAR016 | **Confirmed** |
| "Same model + better errors → higher pass rate" | Iter 1 = 38/51, Iter 6 = 38/51 with all weekend's changes applied | **Not at aggregate level**; only at specific-benchmark level (1 of 17 moved) |
| "PAR017 (`;`) unblocked balanced_parens" | balanced_parens 1/3 in Iter 6, same as Iter 1 baseline | **Walked back** — Iter 5's N=1 100% was variance |
| "Cumulative compiler fixes reduce agent thrash" | Mean agent turns for clean wins is 4-7; baseline was similar | **Inconclusive at this trial count** |

**Refined narrative**: The compiler-error work IS a real lever for unlocking *specific* benchmarks the model previously couldn't solve. But it does NOT move the aggregate pass rate — most benchmarks the model could already do, it still does; ones it can't, it mostly still can't; and PAR016 unlocked exactly one (dense_operator_program). PAR017's effect on balanced_parens was a single-trial illusion.

**The single confirmed win is still load-bearing**. dense_operator_program had eaten 75 agent turns in Iter 3 (the longest single failure of the weekend). PAR016 brought it to clean 6-8 turn passes. That kind of unlock is exactly what the error-quality framework was designed to produce.

## Regressions worth investigating

Three benchmarks moved DOWN from baseline N=3:

| Benchmark | Iter 1 N=3 baseline | Iter 6 N=3 | Hypothesis |
|---|---|---|---|
| adt_option | 100% (Iter 4 also 100%) | 1/3 (33%) | PAR016/PAR017 interaction? Worth diffing trial solutions |
| fizzbuzz | 100% (Iters 0/2/4) | 1/3 (33%) | Same. fizzbuzz uses `;` heavily — PAR017's hint may steer the model to a worse syntax path |
| canonical_normalization | 100% (Iters 1-5) | 2/3 (67%) | Less alarming but trending down |

**Recommended follow-up before more compiler work**: read the failing trial solutions for adt_option and fizzbuzz. If they show PAR017 being mis-applied (e.g., the agent reading the hint and over-restructuring), PAR017's wording needs sharpening.

## Concrete next actions (for the user, Monday morning)

1. **Ship the PAR016 win publicly** — the leaderboard page for Iter 6 can headline "dense_operator_program now reliable on gemma4:26b via PAR016". This is real, repeatable, attributable evidence that compiler error work moves the rig.

2. **Investigate adt_option + fizzbuzz regressions before iterating more compiler errors**. Cost ~30 min: diff 1 passing trial vs 1 failing trial per benchmark, look for `;` or `|` advice being misapplied. If found, PAR017's wording is the fix; if not, log as variance and continue.

3. **Apply the same N=3 standard to all future single-axis claims**. The weekend report's "TWO breakthrough wins" headline was based on N=1 data and is now half-walked-back. The cost was credibility, not money. Future iteration reports should require N≥2 before claiming a benchmark moved.

4. **The OpenRouter baseline plan** ([m-eval-openrouter-baseline-rotation.md](../../planned/v0_24_0/m-eval-openrouter-baseline-rotation.md)) is now even more valuable: it gives us a non-gemma4 ceiling to know how much of the 74.5% is "as good as this hardware can do with our prompt + compiler" vs "this model's specific ceiling". Worth the $25.

## Process notes

- N=3 rotation: 51 trials in ~3.5h (mean ~4.1 min/trial). Sustainable for weekend rotations.
- Iter 6 had no rig issues, no manual interventions, no CI fires. The 2026-05-22..24 operational fixes ([commits 5cf6287b..1b7e06d4](commits/1b7e06d4)) held.
- The user-visible thing the weekend produced that has the cleanest story: **dense_operator_program from 0% over 5 trials to 100% over 3 trials, attributable to one 50-line compiler change**. That's the headline.

---

**Addendum written**: 2026-05-24 11:20 by the autonomous agent (Claude Opus 4.7). Will be updated with the final 2 trial outcomes (inline_tests trial 3, numeric_modulo trial 3) once Iter 6 fully completes.

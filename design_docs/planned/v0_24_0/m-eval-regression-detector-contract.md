# M-EVAL-REGRESSION-DETECTOR-CONTRACT: Specify trial grouping, flaky-vs-persistent classification, and infra-noise policy

**Status**: Planned (codifies an in-flight fix that currently lives only in the script)
**Target**: v0.24.0
**Priority**: P1 (High — a mis-grouping bug pages on flaky noise; the corrected logic is uncommitted and unspecified)
**Estimated**: 0.5 day (the code fix is ~done; this doc is the contract + tests around it)
**Dependencies**: `tools/launchd/nightly-eval.sh` (regression detector); error categories from `internal/eval_harness/error_categorizer.go`; complements [M-EVAL-RATING-EFFICIENCY](m-eval-rating-efficiency.md) (which consumes the same flaky signal for selective reruns).

> **📊 EMPIRICALLY OBSERVED (nightly run 2026-06-07):** 46 result files = 23 benchmarks × 2 trials,
> 43/46 passed. The 3 trial-failures were `explicit_state_threading` (1× `compile_error`),
> `type_safe_record_access` (1× `logic_error`), and `json_parse` (1× `api_error`, the stream-death
> stall). **Every one recovered on its sibling trial → zero persistent failures, zero regressions —
> the detector should fire no alerts for this run.** The pre-fix detector did the opposite: it split
> `*_trial2_*` files into separate single-trial pseudo-benchmarks, so any one failing trial read as
> "this benchmark failed all (its one) trials" and tripped a false "persistent failure" page. A fix
> is applied in a worktree but **uncommitted and unspecified**; this doc is the contract it should
> satisfy, plus the test that locks it.

---

## Problem statement

The nightly regression detector turns raw per-trial result files into Discord pages. Three of its
decisions were never written down as a contract — they live only as Python inside
[`nightly-eval.sh`](../../../tools/launchd/nightly-eval.sh), and one of them was silently wrong:

1. **Trial grouping (the bug).** Result files are named
   `<bench>[_trialN]_<lang>_<model>_<ts>.json`. Trial 1 has no infix; trial 2 is `_trial2`. The old
   detector keyed on `basename.rsplit("_",1)[0]`, which kept `_trial2` as part of the benchmark id,
   so the two trials of one benchmark became **two separate pseudo-benchmarks**. A benchmark that
   passed trial 1 and failed trial 2 then looked like one benchmark that "failed all its trials" →
   false persistent-failure page. The fix strips `_trial\d+$` so both trials collapse to one id
   (`nightly-eval.sh:257`, working tree).

2. **Flaky-vs-persistent.** "Passed ≥1 trial tonight = flaky-but-recovered = no alert" is the right
   policy, but it was implicit. Without it written down, the next refactor can reintroduce the page.

3. **Infra-noise escalation.** `api_error`/`timeout`/`executor_error` (and now `stream_death`, see
   [M-EVAL-STREAM-HEALTH-RETRY](m-eval-stream-health-retry.md)) are transport failures, not
   language/prompt/stdlib defects. They must never escalate as regressions. This rule exists as the
   `INFRA_CATEGORIES` set (`nightly-eval.sh:236`) but is undocumented.

There is no design doc for any of this, and **no test** pins the trial-grouping behavior — exactly
the kind of pure-string-munging logic that silently rots.

## Goals

1. **Write the contract** the detector must satisfy: result-file naming, trial grouping,
   flaky/persistent/regression classification, infra-noise policy.
2. **Commit the fix** that already exists in the worktree, behind this contract.
3. **Pin it with a test** (a `bats`/Python fixture over a directory of synthetic result files,
   including the `_trial2` case) so the 2026-06-07 false-page can never recur silently.
4. **Add a per-benchmark token-outlier flag** (a small, real gap surfaced by the same run).

## The contract

### Result-file naming (normative)

```
<benchmark_id>[_trial<N>]_<lang>_<model>_<unix_ts>.json
```
- `<benchmark_id>` MUST NOT contain the substring `_trial<digits>`.
- Trial 1 omits the infix; trials 2..N carry `_trial<N>`.
- `<unix_ts>` is integer seconds and MUST differ between re-runs of the same (benchmark, trial)
  so newest-wins dedup is well-defined.

### Grouping

1. Per `(benchmark, trial-slot)`, keep the newest file by `<unix_ts>` (dedupe re-runs).
2. Strip `_trial\d+$` from the slot to recover `benchmark_id`; gather all trials under it.

### Classification (per benchmark, per night)

| Condition | Class | Alerts? |
|-----------|-------|---------|
| ≥1 trial passed | **flaky-but-recovered** | No (counted in nightly log only) |
| All trials failed, **all** failures in `INFRA_CATEGORIES` | **infra-gap** | No (transport noise) |
| All trials failed, ≥1 real category, **was NOT solid in prev run** | **gap** | No Discord; nightly-log + controlplane |
| All trials failed, ≥1 real category, **was solid in prev run** (passed all trials) | **regression** | **Discord page** |

`INFRA_CATEGORIES = {api_error, timeout, executor_error, stream_death}`. "Real" = any category not
in that set (`compile_error`, `logic_error`, `runtime_error`, …). Bias is deliberate: **prefer
missing one alert over crying wolf on flaky noise** — a benchmark must have been *solid last night*
to page tonight (`was_solid_in_prev`, `nightly-eval.sh:273`).

### Token-outlier flag (new, minor)

The 4M-token hard cap (`MAX_TOKENS_PER_BENCH`, `nightly-eval.sh:122`;
[m-eval-cost-and-speed-budgets.md](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md)) is an
absolute ceiling, not an *outlier* signal. The 2026-06-07 run had `json_parse` at ~1.13M tokens vs a
~264k cross-benchmark mean — well under 4M, so silent, yet a 4–5× outlier worth seeing. Add a
**relative** flag to the nightly log (not a page): for each passing cell, if
`tokens > max(token_outlier_floor, K × median_pass_tokens)` emit
`token outlier: <bench> <tokens> (Nx median)`. Suggested `K=4`, `token_outlier_floor=500k`,
validated on a few nights. This is reporting only — it never gates or fails a cell.

## Non-goals

- No new trial *count* policy (still `--trials 2`); elastic/selective trials are
  [M-EVAL-RATING-EFFICIENCY](m-eval-rating-efficiency.md)'s job, and this contract is its input.
- No change to what counts as a pass (`compile_ok && runtime_ok && stdout_ok`).
- Not a syntax-flake vs logic-flake taxonomy — both are "real categories" here; their *recovery*
  is the error-quality corpus's concern ([m-ailang-error-quality-for-llm-iteration.md](m-ailang-error-quality-for-llm-iteration.md)), not the detector's.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | +1 | A fixed grouping + dedup contract makes the alert decision a pure function of the result set. |
| A2 Replayability | +1 | The pinning test replays a known result directory and asserts the exact alert set. |
| A7 Machines First | +1 | A false page trains the on-call to ignore pages; a precise detector keeps the signal trustworthy. |
| A9 Cost Visibility | +1 | The token-outlier flag surfaces 4–5× spends the absolute cap hides. |
| A11 Structured Failure | +1 | Names four explicit classes (flaky / infra-gap / gap / regression) where there was one fuzzy "failed" bucket. |
| A12 System Boundary | +1 | The detector owns its naming contract instead of inheriting whatever the writer happened to emit. |

**Hard violation check:** none.

## Acceptance

- [ ] Working-tree fix committed with this doc referenced in the message.
- [ ] Fixture test: a directory containing `bench_a` (pass,pass), `bench_b` (pass, `_trial2` fail
      `logic_error`), `bench_c` (`_trial2` both fail `compile_error`, solid last night), `bench_d`
      (both fail `api_error`) asserts: `bench_a`/`bench_b` → no alert (flaky/recovered),
      `bench_c` → **regression page**, `bench_d` → infra-gap (no page). This is the literal
      2026-06-07 shape plus a true-positive control.
- [ ] `INFRA_CATEGORIES` includes `stream_death`.
- [ ] Token-outlier line appears for a synthetic 1.2M-token pass when median is ~260k.
- [ ] Re-running the 2026-06-07 result set through the detector yields **zero** alerts.

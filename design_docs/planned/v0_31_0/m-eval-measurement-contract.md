# M-EVAL-MEASUREMENT-CONTRACT: Distinguish "Measured Badly" from "Failed to Measure"

**Status**: Planned
**Target**: v0.31.0
**Priority**: P0
**Estimated**: 4 days
**Created**: 2026-07-28
**Dependencies**: None (all changes are additive to the eval pipeline)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A banked datapoint becomes reproducible: subject health + resolved config are recorded alongside the result, so a rate can be re-derived rather than trusted |
| A2: Replayability | +1 | Banking the RESOLVED runtime config (not the claimed one) is what makes an eval row replayable at all |
| A3: Effect Legibility | 0 | No language-level effects touched |
| A4: Explicit Authority | 0 | No new ambient access; the canary runs with the same caps as a normal benchmark |
| A5: Bounded Verification | +1 | The canary is a bounded, local, ~30s check with a hard timeout — not an open-ended probe |
| A6: Safe Concurrency | 0 | Canary runs serially before the suite; no new concurrency |
| A7: Machines First | +1 | Replaces human-read prose caveats (CAVEATS.md) with a machine-checkable `validity` field that downstream analysis can filter on |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | +1 | A dead subject currently burns a full night of GPU/API spend and yields nothing; the canary fails in 30s |
| A10: Composability | +1 | Extends the existing `Executor` interface, so every executor (claude/codex/motoko/managed_agents) inherits the gate uniformly |
| A11: Structured Failure | +1 | Invalid measurements carry a typed reason code instead of collapsing to an indistinguishable `0` |
| A12: System Boundary | +1 | Makes the harness→subject boundary an explicit, asserted contract rather than an assumption |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the change removes a source of unreproducible data
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimises for machine-filterable validity over human-readable caveats

---

## Problem Statement

The eval pipeline cannot tell the difference between **"the subject was measured and did badly"** and **"we failed to measure the subject."** Both produce a low number, and both get banked as findings. Four independent incidents in the six weeks to 2026-07-28, all the same shape:

| Incident | What actually happened | What the pipeline recorded |
|---|---|---|
| motoko dead 2026-07-22 → 07-28 | AILANG's effect-row soundness fix (`1282767ca`) broke motoko's module load; it died before step 0 on every run | **72 benchmark "failures"** across 6 nights |
| `microrag_ab.jsonl` 2026-07-20 | A shell bug produced an empty pass count; `${PASS_ON:-0}` coerced it to `0` | `on_pass: 0, delta_pp: -73.8` — banked as a real 0% measurement |
| Cloud motoko models | All 10 fell through to the `dogfood` profile, running without `ailang_docs`/`microrag` and with a verify gate (`make check_core`) that cannot work in a benchmark workspace | rows whose `models.yml` description claimed "DP7 verifier + microRAG context" |
| fmt A/B | Run against haiku, where both arms sit at ~96% — structurally incapable of showing the effect | 30/30 vs 29/30, reported as a comparison |

**Why the existing health check did not catch the outage.** `HealthCheck` already exists ([internal/executor/motoko/healthcheck.go](../../../internal/executor/motoko/healthcheck.go)), is on the `Executor` interface, and *is* called pre-flight at [agent_runner_multi.go:107](../../../internal/eval_harness/agent_runner_multi.go). It verifies binary existence, `OPENROUTER_API_KEY`, and `motoko --version`. But `--version` is handled in `parseMotokoFlags()` at the TypeScript argv level and exits **before any AILANG module is loaded**. So it exited 0, seventy-two times, while the AILANG core was completely dead.

The gate exists. It tests the wrapper, not the subject.

**Compounding design flaw.** Separately from execution hygiene, the weekly A/B is **underpowered by construction**. At n=84 and p≈0.73 the standard error on one arm is 4.8pp; on an unpaired difference, 6.8pp. Nothing below ~±13pp can reach significance. The three usable microRAG deltas are −3.1, −4.8, +13.1 — all noise, with the "positive" sitting exactly at the detection threshold. Running this weekly for another year would not resolve it. The pipeline compares aggregate pass *rates*, discarding the benchmark-level pairing that would cancel the dominant variance term.

**Who is affected:** every downstream decision that reads eval data — the V1 mission backlog, the OS/local leaderboard, model routing, and any "does technique X help?" question.

---

## Goals

**Primary goal:** No number enters a trend line unless the pipeline has verified that it is a measurement.

**Success metrics:**

1. A dead subject fails within 60s of suite start, not after a full night — and banks **zero** benchmark rows.
2. Every banked eval result carries a machine-readable `validity` field; downstream analysis filters on it by default.
3. `microrag_ab.jsonl` and `fmt_ab.jsonl` rows carry paired per-benchmark outcomes, so a McNemar test can be computed without re-running.
4. Every banked agent row carries the **resolved** runtime config; a mismatch against `models.yml` is a loud error, not a silent divergence.
5. Retroactive: the known-bad 2026-07-20 row is quarantined rather than deleted, with its reason recorded.

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Cost to change later |
|---|---|---|---|---|---|
| D1 | Canary scope | (a) `--version` only (status quo) · (b) trivial end-to-end benchmark · (c) full smoke tier | **(b)** — one trivial benchmark exercising module-load → step 0 → tool call → completion | Mark | Low (config) |
| D2 | Canary failure behaviour | (a) warn and continue · (b) abort the whole suite · (c) skip that model, continue others | **(c)** — a dead motoko must not abort an opencode/pi rotation | Mark | Low |
| D3 | Validity representation | (a) boolean · (b) enum + reason string · (c) drop invalid rows entirely | **(b)** — quarantine with a typed reason; never delete data | Agent | Medium (schema) |
| D4 | Retroactive handling of the 2026-07-20 row | (a) delete · (b) mark `validity: invalid` in place · (c) leave and document | **(b)** — never rewrite history destructively; the row is evidence of the bug | Mark | Low |
| D5 | Paired analysis home | (a) new Go pkg in `internal/eval_analysis` · (b) Python in `tools/` | **(a)** — the harness already owns comparison; Go keeps it testable in CI | Agent | Medium |

### Design Freeze

Check off before sprint-executor starts:

- [ ] D1 canary scope confirmed (b)
- [ ] D2 failure behaviour confirmed (c)
- [ ] D4 confirmed: quarantine in place, do not delete

---

## Solution Design

### Overview

Five changes, each independently shippable. Ordered by value-per-line — M1 alone would have prevented the six-day outage.

### Architecture

```
                    ┌─────────────────────────────────────────┐
  eval-suite  ──────▶ M1  PRE-FLIGHT CANARY (per model)        │
                    │    HealthCheck() + CanaryCheck()         │
                    │    dead subject ⇒ skip model, bank NOTHING│
                    └────────────────┬────────────────────────┘
                                     │ subject verified alive
                    ┌────────────────▼────────────────────────┐
   run benchmarks   │ M4  bank RESOLVED config with each row   │
                    │     (from motoko runtime_config_resolved)│
                    └────────────────┬────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────┐
   bank             │ M2  validity: valid | invalid(reason)    │
                    │     invalid never enters a trend         │
                    └────────────────┬────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────┐
   compare          │ M3  paired per-benchmark (McNemar)       │
                    │     not aggregate rate deltas            │
                    └─────────────────────────────────────────┘

   M5  subject-selection rule: an arm needs headroom (doc + lint)
```

### Implementation Plan

**M1 — Pre-flight canary gate (P0, ~1 day)**

Extend the existing `Executor` interface rather than inventing a parallel mechanism:

```go
// CanaryCheck runs one trivial end-to-end task and asserts the subject can
// actually complete a step. HealthCheck proves the binary exists; CanaryCheck
// proves the subject WORKS. motoko --version exits 0 with a dead AILANG core —
// that gap banked 72 phantom failures over six days in July 2026.
CanaryCheck(ctx context.Context) error
```

- Default implementation returns `nil` (opt-in per executor) so claude/codex/managed_agents are unaffected until they opt in.
- motoko's implementation runs a fixed one-line task in a temp workspace with a 90s timeout, asserting: process exits 0, ≥1 step executed, ≥1 tool call dispatched, `run_summary` present.
- Called once per model in `eval-suite`, before any benchmark. Failure ⇒ that model is skipped with a `canary_failed` record; **no benchmark rows are banked** for it.

**M2 — Universal validity flag (P0, ~1 day)**

```go
type Validity struct {
    Valid  bool   `json:"valid"`
    Reason string `json:"reason,omitempty"` // canary_failed | zero_files | zero_pass_all | config_mismatch | harness_error
}
```

- Added to the banked result struct in `internal/eval_harness/metrics.go`.
- `internal/eval_analysis` filters `valid == false` by default in every aggregate; an explicit `--include-invalid` opts back in.
- Already applied ad-hoc to the two weekly A/Bs in [tools/launchd/nightly-eval.sh](../../../tools/launchd/nightly-eval.sh) (commit `48fe89b35`); M2 generalises it into the Go layer so it is not shell-local.

**M3 — Paired A/B analysis (P1, ~1 day)**

New `internal/eval_analysis/paired.go`:

- Joins arms on `(benchmark, trial_index, lang)`, producing per-benchmark `(on_pass, off_pass)` pairs.
- Reports discordant pairs `b`/`c` and a McNemar statistic alongside the aggregate delta.
- Extends the `*_ab.jsonl` schema with a `pairs` array so the test is recomputable from banked data without re-running.
- Aggregate delta is retained for continuity — this adds a column, it does not remove one.

**M4 — Bank the resolved config (P1, ~0.5 day)**

motoko already broadcasts `runtime_config_resolved` at step 0; it is consumed by `system_prompt_guard.go` and then **discarded**. Capture it into the result row (`resolved_profile`, `resolved_extensions`, `resolved_verification`) and assert it against the `models.yml` claim. Mismatch ⇒ `validity: invalid(config_mismatch)`. This is the check that makes the cloud-profile class of bug self-reporting.

**M5 — Subject-selection rule (P2, ~0.5 day)**

Documentation + a lint in the A/B setup path: an arm whose control is above a configurable ceiling (default 90%) cannot resolve a small effect, so the harness warns loudly at setup time. Prevents repeating the haiku-fmt mistake.

### Files to Modify/Create

| File | Change | Est. LOC |
|---|---|---|
| `internal/executor/executor.go` | add `CanaryCheck` to interface + default no-op | +25 |
| `internal/executor/motoko/canary.go` | **new** — motoko canary implementation | +140 |
| `internal/executor/{claude,codex,managed_agents}/*.go` | default no-op wiring | +30 |
| `internal/eval_harness/agent_runner_multi.go` | call canary next to the existing HealthCheck (line ~107) | +20 |
| `internal/eval_harness/metrics.go` | `Validity` struct + serialisation | +45 |
| `internal/eval_analysis/paired.go` | **new** — pairing + McNemar | +180 |
| `internal/eval_analysis/*.go` | filter invalid by default | +40 |
| `tools/launchd/nightly-eval.sh` | emit `pairs`; reuse Go validity | +50 |
| `docs/static/benchmarks/microrag_ab.jsonl` | quarantine the 2026-07-20 row (D4) | +1 |
| `MOTOKO.md` / eval guide | document the contract | +60 |

---

## Examples

### Example 1: The six-day outage, under M1

```
$ ailang eval-suite --agent --models motoko-local-qwen3-6-35b-a3b-mxfp8 ...
→ health check: motoko binary OK (git_rev=e2dbff8)
→ canary: motoko-local-qwen3-6-35b-a3b-mxfp8 ... FAILED after 4.1s
    effect checking failed in src/core/tool_runtime: Missing effects: Env
  SKIPPING model (canary_failed). 0 rows banked.

Suite complete: 0/0 benchmarks for motoko-local-* (subject unavailable)
```

Today that same condition silently banks 12 "failures" per night for six nights.

### Example 2: The 2026-07-20 row, under M2

```jsonl
{"date":"2026-07-20", ..., "on_pass":0, "delta_pp":-73.8,
 "validity":{"valid":false,"reason":"zero_pass_all"}}
```

`ailang eval-report` excludes it; the trend reads −3.1, −4.8, +13.1 over three points instead of averaging in a −73.8 artefact.

---

## Success Criteria

- [ ] A deliberately-broken motoko causes `canary_failed` in <60s and banks zero benchmark rows (integration test)
- [ ] `Validity` present on every banked agent result; `eval_analysis` aggregates exclude invalid by default
- [ ] `paired.go` reproduces the aggregate delta AND reports discordant pairs on the existing 2026-07-27 data
- [ ] A profile mismatch between `models.yml` and `runtime_config_resolved` yields `config_mismatch`
- [ ] The 2026-07-20 row is quarantined in place, not deleted
- [ ] `make test` green; `make check-boundaries` green
- [ ] MOTOKO.md + eval guide document the contract

## Testing Strategy

- **Unit**: `Validity` serialisation round-trip; McNemar against hand-computed fixtures; canary timeout/partial-output handling.
- **Integration**: canary against a deliberately broken motoko checkout (temp worktree with a reverted effect row) — this is the regression test for the actual outage.
- **Regression**: replay the banked 2026-07-27 A/B through `paired.go`; the aggregate delta must match the historical `+13.1` exactly.
- **No new eval spend**: all tests run against fixtures, not live models.

## Deferred Decisions

Agent has latitude on: canary task text and timeout constant; `Validity.Reason` enum spelling; whether McNemar uses exact binomial or χ² with continuity correction (document the choice); file layout within `internal/eval_analysis`.

## Non-Goals

- Re-running or back-filling historical evals (the existing data stays as-is, only annotated).
- Changing the benchmark set, tiers, or grading.
- Statistical power *fixes* beyond pairing — deciding to raise trials/n is a separate call once pairing shows the real variance.
- Any change to the AILANG language, compiler, or stdlib. **No Conflict Surface section is required**: this design touches only `internal/eval_harness`, `internal/eval_analysis`, `internal/executor`, and `tools/` — none of the parser/typechecker/codegen/effects surfaces that mandate one.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 canary — interface, motoko impl, call site, integration test |
| 2 | M2 validity — struct, serialisation, analysis filtering |
| 3 | M3 paired analysis + McNemar + schema extension |
| 4 | M4 resolved config + M5 selection rule + docs |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Canary adds latency to every suite run | High | Low | ~30–60s once per model, against a full night's run; skip via `--no-canary` for iteration |
| Canary itself becomes flaky and blocks good runs | Medium | High | Canary must be the most trivial possible task; on ambiguous failure, retry once then degrade to a warning rather than a skip |
| Filtering invalid rows changes historical dashboard numbers | Medium | Medium | Expected and desirable; call it out in the release notes and keep `--include-invalid` |
| McNemar on tiny discordant counts is itself underpowered | High | Medium | Report `b`/`c` counts alongside the statistic so the reader sees the evidence base; do not report a p-value on <10 discordant pairs |

## Related Documents

Neural search returned no doc above 0.45 similarity; nearest neighbours are older and materially different:

- [M-EVAL: Enhanced Validation for Agent Benchmarks](../../archive/v0_3_25_m-eval-harness-validation.md) (0.39) — validates benchmark *specs*, not banked-result validity. Distinct.
- [M-EVAL-LOOP: Self-Improving AI Feedback Loop](../../implemented/v0_3_10/M-EVAL-LOOP_self_improving_feedback.md) (0.36) — consumes eval results to propose fixes; assumes the results are trustworthy. This doc supplies that assumption.
- [MOTOKO.md](../../../MOTOKO.md) — the checkout/profile map; §6 documents the failure modes this design automates.

## Verification Log

Every load-bearing claim, checked against the code on 2026-07-28. Negative-existence claims carry their own row.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | A pre-flight health check already exists | read `internal/executor/motoko/healthcheck.go` | **CONFIRMED EXISTS** — checks binary, `OPENROUTER_API_KEY`, `motoko --version`. Design extends it; does not duplicate it |
| V2 | `HealthCheck` is on the `Executor` interface and called pre-flight | `grep HealthCheck(` | CONFIRMED — `executor.go:31`; called `agent_runner_multi.go:107`; implemented by claude/codex/managed_agents |
| V3 | `motoko --version` does not load AILANG modules | read `src/tui/src/index.ts` `parseMotokoFlags()` (line 638) + live run | CONFIRMED — handled at TS argv level, exits before module load; live run printed `tui_version/git_rev` only |
| V4 | (negative) No validity/quarantine concept exists on banked results | `grep -riE '"valid"\|banked_valid\|quarantin'` over `internal/eval_harness/`, `internal/eval_analysis/` | CONFIRMED ABSENT — only unrelated spec-validation strings |
| V5 | (negative) No paired/McNemar analysis exists | `grep -riE 'mcnemar\|paired\|per_benchmark_delta'` over `internal/`, `tools/` | CONFIRMED ABSENT — hits are goroutine pairing and model-family grouping, unrelated |
| V6 | (negative) `runtime_config_resolved` is broadcast but NOT banked | `grep runtime_config_resolved` | CONFIRMED — consumed only by `system_prompt_guard.go`; absent from `metrics.go` and `eval_analysis/` |
| V7 | 72 runs banked during the outage | `grep -c` over `$TMPDIR/motoko-stderr-*.log` | CONFIRMED — 72 logs with the identical effect error, 2026-07-22 → 07-28 |
| V8 | The 2026-07-20 row is a harness artefact, not a result | read the `PASS_ON` pipeline in `nightly-eval.sh` | CONFIRMED — `grep\|cut` yields empty with exit 0, so `\|\|` fallback never fires; `${PASS_ON:-0}` coerces to 0. Fixed in `48fe89b35` |
| V9 | Underpowered at n=84 | binomial SE: √(0.73·0.27/84)=4.8pp; unpaired diff SE=6.8pp | CONFIRMED — only effects >~13pp detectable; observed deltas −3.1/−4.8/+13.1 |
| V10 | fmt A/B on haiku is at ceiling | counted `eval_results/fmt_ab_haiku_*` | CONFIRMED — 30/30 vs 29/30 and 42/45 vs 43/45 |
| V11 | The fmt arm has never run on the local model | `find eval_results -name '*motoko-local-qwen3-6-fmt*'` | CONFIRMED — 0 files vs 600 for the baseline |

## References

- Outage root cause: ailang `1282767ca` (effect-row soundness, #386)
- Shell-bug fix + fmt A/B wiring: ailang `48fe89b35`
- motoko restore: mk-ast `13bc085`
- [MOTOKO.md](../../../MOTOKO.md)

## Future Work

- Raise trials/n once pairing reveals the true variance (deliberately deferred — decide with data, not now).
- Extend `CanaryCheck` to claude/codex/managed_agents once the motoko implementation has proven itself.
- A dashboard badge showing what fraction of each series is `valid`.

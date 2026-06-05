# Model-Capability-Aware Regression Triage & Gating for nightly-eval

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium)
**Estimated**: 2–3 days
**Dependencies**: `m-eval-model-roster-validation` (sibling, same nightly thread), `m-msg-triage-router` / `m-msg-auto-triage-pipeline`

> **Provenance**: This doc was produced by the `design-doc-creator` agent in response to a
> `nightly-eval` Pub/Sub notification (task `task-65639210`, thread
> `thread_1780537887673_90f9df6f`, "Pub/Sub notification from nightly-eval", 2026-06-05).
> Rather than document a single benchmark failure as a language gap, an audit of the
> dispatching pipeline found a **systemic false-positive problem** (CLAUDE.md Principle #3:
> *audit before patching*). This doc fixes the pipeline that generated the task, not one
> benchmark.

---

## Problem Statement

The nightly regression guard (`tools/launchd/nightly-eval.sh`) treats **any** benchmark that
fails both trials of a **single** local smoke-tier model as a "regression" worth a GitHub issue
**and** a `design-doc-creator` task. This produces a stream of false-positive design-doc tasks
for problems that are not language or stdlib defects.

### Current State (verified from the live pipeline, 2026-06-04 / 2026-06-05)

Every nightly regression filed on the last two nights came from **one** model —
`opencode-qwen3-5-35b-a3b-mxfp8` (local, quantized 35B, `tier: smoke`) — and spans unrelated
*foundational* benchmarks:

| Issue | Benchmark | Error category | Model |
|-------|-----------|----------------|-------|
| #262 (06-04) | dense_operator_program | **`api_error`** | qwen3-5-35b (local, smoke) |
| #263 (06-04) | explicit_state_threading | compile_error | qwen3-5-35b (local, smoke) |
| #266 (06-05) | balanced_parens | compile_error | qwen3-5-35b (local, smoke) |
| #267 (06-05) | binary_tree_sum | logic_error | qwen3-5-35b (local, smoke) |
| #268 (06-05) | binary_tree_sum_trial2 | compile_error | qwen3-5-35b (local, smoke) |
| #269 (06-05) | explicit_state_threading | compile_error | qwen3-5-35b (local, smoke) |
| #270 (06-05) | explicit_state_threading_trial2 | compile_error | qwen3-5-35b (local, smoke) |
| #271 (06-05) | fizzbuzz | compile_error | qwen3-5-35b (local, smoke) |
| #272 (06-05) | recursion_fibonacci | logic_error | qwen3-5-35b (local, smoke) |

Each row also became a `design-doc-creator` task. Nine tasks share the nightly thread; several
are still `queued` (including `task-65639210`, which produced *this* doc).

**The language is not the problem.** These benchmarks compile and run correctly under the
current `ailang` binary:

```text
$ ailang run --caps IO examples/runnable/recursion_fibonacci.ail
fib(10) = 55
fibTail(10, 0, 1) = 55

$ ailang run --caps IO examples/runnable/test_fizzbuzz.ail
1
Fizz
Buzz
FizzBuzz
...                                  # matches benchmarks/fizzbuzz.yml expected_stdout
```

`fizzbuzz` exercises control flow + an IO loop; `recursion_fibonacci` exercises recursion —
the exact capabilities the failing rows claim are broken. They are not.

### Three concrete defects in the dispatch logic

1. **No baseline → "regression" is a misnomer.** `nightly-eval.sh` files a failure when a
   benchmark "failed both trials" *today*. There is no persisted per-(model, benchmark)
   history, so the script **cannot distinguish a true regression (passed before, fails now)
   from a benchmark that has never passed** for this weak model. The issue text it emits even
   says *"If it passed in a previous nightly run, this is a regression. If it has never passed,
   it is a known language gap"* — but the script never checks which case it is.

2. **No error-category gate.** `#262` was an **`api_error`** — a transport/infra failure of the
   local Ollama endpoint — yet it was filed identically to a `compile_error` and routed to
   `design-doc-creator`. Infra categories (`api_error`, `timeout`) can never be language gaps,
   but the pipeline does not filter them.

3. **Single low-capability model is treated as authoritative signal.** A 35B quantized
   smoke-tier model failing `fizzbuzz` is a model-capability boundary, not evidence of a
   language defect. One model's failure is dispatched with no corroboration from a
   reference-capable model and no check against AILANG's own reference solution.

### Impact

- **Wasted agent budget**: 9 design-doc-creator tasks in two nights for non-problems.
- **Repo pollution risk**: pressures the agent to write language-gap design docs for healthy
  language features — the exact failure mode that got `m-type-constraints.md` and
  `m-import-alias` **rejected** (see the `design-doc-creator` skill's hard-gate case study,
  2026-06-03). `m-type-constraints.md` still sits in `planned/v0_24_0/` as the cautionary tail
  of this pattern.
- **Alert fatigue**: real cross-model regressions get buried under single-model smoke noise, so
  the regression guard loses credibility.

---

## Goals

**Primary Goal:** Only dispatch a `design-doc-creator` task when a nightly failure is a
*verified* language/stdlib problem — a true regression or a cross-model–corroborated gap that
AILANG's own reference solution does not already pass — and never for infra failures or
single weak-model capability boundaries.

**Success Metrics:**
- **False-positive design-doc dispatches → 0** for the 9 failures above (all reclassified as
  model-capability / infra, none warrant a language design doc).
- **True regressions still caught**: a benchmark that passed in the prior nightly and fails
  today still files an issue + task (verified by a synthetic baseline-flip fixture).
- **`api_error` / `timeout` failures → never** routed to `design-doc-creator` (0 of N).
- **One triage summary message per night** instead of N per-failure messages.
- No change required to the language, stdlib, or any `.ail` program.

---

## Solution Design

### Overview

Insert a **triage gate** between "benchmark failed both trials" and "dispatch to
design-doc-creator". The gate decides a *disposition* per persistent failure:

```
persistent failure (both trials fail)
        │
        ▼
  [1] error-category gate ──► infra (api_error/timeout) ──► log only, no dispatch
        │ (compile/logic/runtime)
        ▼
  [2] baseline check ─────► never-passed for this model ──► "model-capability", no dispatch
        │ (was passing → now failing)             │
        │                                          └► (also) reference-solution check
        ▼                                                     │
  [3] corroboration ──► AILANG reference solution PASSES ──► "model-capability", no dispatch
        │              (and only 1 weak model failed)
        │ (reference fails OR ≥2 distinct models fail)
        ▼
  [4] DISPATCH design-doc task  (true language/stdlib gap)
```

Dispositions: `language-gap` (dispatch), `regression` (dispatch, higher priority),
`model-capability` (controlplane note, no dispatch), `infra` (log only).

### Architecture

**Baseline state.** Persist a small JSON ledger of per-(model, benchmark) outcomes across
nights, e.g. `~/.ailang/nightly-baseline.json` (rig-local) mirrored to the
`obs_metrics` Firestore collection for the dashboard:

```json
{
  "opencode-qwen3-5-35b-a3b-mxfp8": {
    "fizzbuzz":            {"last_pass": null,         "last_seen": "2026-06-05", "streak_fail": 9},
    "recursion_fibonacci": {"last_pass": "2026-05-20", "last_seen": "2026-06-05", "streak_fail": 2}
  }
}
```

- `last_pass == null` ⇒ **never passed** ⇒ not a regression; candidate for model-capability or
  (if corroborated) a standing language gap, but **not** dispatched on a single weak model.
- `last_pass` set and today fails ⇒ **true regression** ⇒ dispatch with `regression` priority.

**Error-category gate.** Reuse the existing categorizer
(`internal/eval_harness/error_categorizer.go`, categories: `compile_error`, `logic_error`,
`runtime_error`, `api_error`, `timeout`). Define an `infraCategories` set
(`api_error`, `timeout`) that is logged but never dispatched.

**Reference-solution corroboration.** For each candidate benchmark, run AILANG's own reference
solution (or a curated known-good `.ail`) under `ailang run`. If it passes, the language
handles the task → the model failed, not AILANG → disposition `model-capability`. This is the
decisive, *machine-checkable* gate the `design-doc-creator` skill already mandates ("verify
every language claim with `ailang check`"), lifted into the pipeline so the gate runs **before**
an agent is ever spawned.

**Cross-model corroboration (optional, when roster available).** If `m-eval-model-roster-
validation` has a reference-capable model in the nightly roster, require the benchmark to fail
for ≥2 distinct models before `language-gap` dispatch. On the current single-model rig this
degrades gracefully to the reference-solution check.

**Batched dispatch.** Replace the per-failure `while`-loop `ailang messages send
design-doc-creator …` with a single nightly triage summary to `controlplane`, plus at most one
dispatch per *dispatchable* failure (after gating).

### Implementation Plan

**Phase 1 — Gate + baseline (Day 1)**
- Add `tools/nightly/triage.py` (or extend the inline `python3` heredoc in `nightly-eval.sh`)
  implementing the 4-step gate and reading/writing the baseline ledger.
- Add `infraCategories` filtering keyed off the categorizer's existing labels.

**Phase 2 — Reference-solution check (Day 1–2)**
- Add a `benchmarks/<id>/reference.ail` (or reuse `examples/runnable/*`) lookup; run under
  `ailang run --caps …` with a timeout; record pass/fail in the disposition.
- Curate reference solutions for the 17 smoke benchmarks (most already exist under
  `examples/runnable/` and `examples/reference/`).

**Phase 3 — Wire dispositions into dispatch (Day 2)**
- `nightly-eval.sh`: only `messages send design-doc-creator` for `language-gap` / `regression`.
- Emit one `controlplane` triage summary listing every disposition (so model-capability and
  infra failures stay *visible* without spawning agents — CLAUDE.md Principle #2: fail loudly,
  don't silently drop).

**Phase 4 — Tests (Day 3)**
- Unit tests for the gate: never-passed vs regression, infra category, reference-passes vs
  reference-fails. A synthetic baseline-flip fixture proves true regressions still dispatch.

### Files to Modify/Create

| File | Change | Est. LOC |
|------|--------|----------|
| `tools/launchd/nightly-eval.sh` | Replace per-failure dispatch loop with gate + batched summary | ~60 ± |
| `tools/nightly/triage.py` (new) | 4-step disposition gate + baseline ledger I/O | ~180 |
| `tools/nightly/triage_test.py` (new) | Unit tests incl. baseline-flip regression fixture | ~120 |
| `benchmarks/<id>/reference.ail` (curate) | Known-good solutions for smoke set (reuse where possible) | ~0–200 |
| `internal/eval_harness/error_categorizer.go` | Export an `IsInfraCategory(cat) bool` helper | ~15 |
| `docs/docs/guides/evaluation/` | Document the triage gate + dispositions | ~40 |

No changes to `internal/parser`, `internal/types`, `internal/eval`, stdlib, or any `.ail`
language semantics.

---

## Examples

### Before (today)

```text
nightly-eval.sh: persistent failures (9): balanced_parens [compile_error] ...
  filed: balanced_parens          → design-doc task (queued)
  filed: dense_operator_program   → design-doc task (queued)   # api_error!
  filed: fizzbuzz                 → design-doc task (queued)   # language runs it fine
  ... 9 tasks, 9 GitHub issues, all from one weak model
```

### After (gated)

```text
nightly triage (2026-06-05):
  dense_operator_program  infra              (api_error)             → log only
  fizzbuzz                model-capability   (ref solution passes)   → no dispatch
  recursion_fibonacci     model-capability   (ref solution passes)   → no dispatch
  balanced_parens         model-capability   (ref solution passes)   → no dispatch
  binary_tree_sum         model-capability   (ref solution passes)   → no dispatch
  explicit_state_threading model-capability  (ref solution passes)   → no dispatch
→ 1 controlplane summary, 0 design-doc tasks, 0 false-positive issues.

# Contrast — a real regression still fires:
  some_bench  regression  (last_pass 2026-06-04, fails today, ref passes)  → DISPATCH
```

---

## Success Criteria

- [ ] The 9 failures from 2026-06-04/05 all resolve to `infra` or `model-capability`; **0**
      design-doc tasks dispatched for them.
- [ ] Synthetic baseline-flip fixture (was-pass→now-fail) **does** dispatch a `regression` task.
- [ ] `api_error` / `timeout` never dispatch (unit-tested).
- [ ] Exactly one `controlplane` triage summary per night; model-capability/infra failures
      remain visible in it (no silent drops).
- [ ] Reference-solution check runs per candidate with a bounded timeout.
- [ ] All new tests passing; `make test` green.
- [ ] Evaluation guide updated with the disposition table.

---

## Conflict Surface / Behavior Change

This change does **not** touch `internal/parser`, `internal/lexer`, `internal/ast`,
`internal/types`, `internal/elaborate`, `internal/codegen`, `internal/eval`, or
`cmd/ailang/exec.go`, so the language-level Conflict Surface analysis is **N/A**. The change is
confined to the nightly-eval tooling and the eval-harness categorizer helper.

**Behavior changes (operational, intentional):**
1. Failures categorized `api_error`/`timeout` **stop** creating regression issues + design-doc
   tasks. They remain in the nightly summary. *Intentional.*
2. A benchmark that has **never** passed for a single smoke-tier model **stops** auto-creating a
   design-doc task; it surfaces as `model-capability` in the summary. True regressions
   (was-pass→now-fail) are unaffected. *Intentional.*
3. Dispatch volume drops from N-per-failure to ≤1-per-dispatchable-failure plus one summary.
   *Intentional.*

**Must still work post-change (fixtures for Phase 4):**
- A genuine regression (baseline `last_pass` set, today fails, reference passes) → dispatches.
- A benchmark failing for ≥2 distinct models (when roster has >1) → dispatches as `language-gap`.
- All-pass night → one "all green" summary, no dispatch (unchanged from today).

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Reference-solution check is a deterministic, replayable gate vs. a stochastic single-model verdict |
| A2: Replayability | +1 | Persisted baseline ledger makes "is this a regression?" reproducible across nights |
| A3: Effect Legibility | 0 | No effect-system changes |
| A4: Explicit Authority | 0 | No capability/authority changes |
| A5: Bounded Verification | +1 | Disposition is decided by local, bounded checks (category + baseline + `ailang run`) before any agent spawns |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Stops feeding agents false signals; reduces wasted token/agent budget on non-problems |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Fewer spurious agent tasks → lower, more legible eval cost; summary surfaces dispositions |
| A10: Composability | +1 | Composes with `m-eval-model-roster-validation` (cross-model corroboration) and `m-msg-triage-router` |
| A11: Structured Failure | +1 | Replaces undifferentiated "regression" with structured dispositions (infra / model-capability / regression / language-gap) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — adds a *deterministic* gate
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimizes machine signal quality, not human convenience

---

## Verification Log (language claims)

Per the `design-doc-creator` hard gate, every "AILANG does/doesn't X" claim is backed by a live
run:

| Claim | Evidence |
|-------|----------|
| AILANG runs `recursion_fibonacci` correctly | `ailang run --caps IO examples/runnable/recursion_fibonacci.ail` → `fib(10) = 55` |
| AILANG runs `fizzbuzz` correctly | `ailang run --caps IO examples/runnable/test_fizzbuzz.ail` → output matches `benchmarks/fizzbuzz.yml` `expected_stdout` |
| `#262` is infra, not language | Issue body: `Error category: [api_error]` |
| All 9 failures share one model | Issues #262, #263, #266–#272 all list `opencode-qwen3-5-35b-a3b-mxfp8 (local, tier:smoke)` |
| Pipeline files single-model failures as regressions with no baseline | `tools/launchd/nightly-eval.sh` — the persistent-failure heredoc checks only "failed both trials today"; no history is read |

---

## Related Documents

- `design_docs/planned/v0_24_0/m-eval-model-roster-validation.md` — sibling nightly task;
  provides the multi-model roster this doc's cross-model corroboration composes with.
- `design_docs/planned/v0_24_0/m-msg-triage-router-sprint-plan.md` /
  `m-msg-auto-triage-pipeline.md` — the message-routing layer this gate sits in front of.
- `design_docs/planned/v0_24_0/m-eval-local-ollama.md` — the local Ollama rig that runs the
  nightly smoke tier.
- `design_docs/planned/v0_24_0/m-type-constraints.md` — *cautionary example*: an eval-gap doc
  rejected for asserting an unverified language limitation; the failure mode this gate prevents.

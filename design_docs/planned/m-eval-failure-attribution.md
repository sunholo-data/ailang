# M-EVAL-FAILURE-ATTRIBUTION: bank *whose fault* a failure was, not just what broke

**Status**: Planned
**Target**: v0.33.1
**Priority**: P0 — 22% of banked failures in the v0.32.0 baseline are not model capability failures, and nothing in the pipeline can tell. Every leaderboard, ELO fit, curation decision and model-add gate reads these rows as model weakness.
**Estimated**: 2-3 days
**Dependencies**: None. **Pairs with** [m-eval-validity-discipline W8](m-eval-validity-discipline.md) — see "Relationship to W8", which is load-bearing for scoping both.
**Author**: Claude Opus 5 (requested by Mark, 2026-08-13 — "we need to fix the eval as we can't trust it currently with these false negatives")

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics; classification is a pure function of an already-banked row |
| A2: Replayability | +1 | Attribution is banked on the row, so a historical result can be re-read without re-running it |
| A3: Effect Legibility | 0 | Not touched |
| A4: Explicit Authority | 0 | Not touched |
| A5: Bounded Verification | +1 | A capability claim becomes checkable against a stated denominator instead of a blended one |
| A6: Safe Concurrency | 0 | Not touched |
| A7: Machines First | +1 | The consumers of this data are machines (ELO fit, curation cycle, model-add gate); today they cannot distinguish an interpreter crash from a model error |
| A8: Minimal Syntax | 0 | No language syntax change — eval tooling only |
| A9: Cost Visibility | +1 | Stops re-running and re-paying for benchmark/model pairs that cannot pass for harness reasons |
| A10: Composability | 0 | Not touched |
| A11: Structured Failure | +1 | The whole point: a failure gains a typed *attribution* alongside its typed *category* |
| A12: System Boundary | +1 | Draws the missing boundary between "the thing under test failed" and "the apparatus failed" |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): classification is pure over `(error_category, stderr, expected_stdout, stdout)`; no clocks, no network, no randomness
- [x] A3 (Effects): no hidden side effects — the classifier writes only to the row it is given
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): improves machine-analyzable eval data; no human-convenience tradeoff

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

Net +6, no −1 on A1/A3/A4/A7 → proceed.

---

## Problem Statement

**A benchmark run measures two things at once — the model, and our apparatus — and banks the result as if it only measured the model.**

`CategorizeError` (internal/eval_harness/metrics.go:212) is the whole attribution story for a standard-mode run:

```go
func CategorizeError(compileOk, runtimeOk, stdoutOk bool) string {
	switch {
	case !compileOk:  return ErrorCategoryCompile
	case !runtimeOk:  return ErrorCategoryRuntime
	case !stdoutOk:   return ErrorCategoryLogic
	default:          return ErrorCategoryNone
	}
}
```

Three booleans. It is structurally incapable of distinguishing *the model wrote bad code* from *our interpreter crashed on good code* — both are `!runtimeOk`, both bank as `runtime_error`, both are read downstream as model weakness.

The safety net that should catch this, `applyValidityBackstop` (internal/eval_harness/validity.go:133), fires only for one category:

```go
if m.ErrorCategory != ErrorCategoryAPI { return }
```

Its own comment states the premise: *"every identifiable MODEL failure has its own category (compile_error, runtime_error, logic_error, ...). So reaching api_error means the harness genuinely does not know what happened"*, and *"The direction is deliberately conservative: an unknown failure is charged to us and retried, never to the model."*

**That premise is false, and the stated direction is inverted in practice.** `runtime_error` is not a model-failure category — it is where harness misconfiguration and AILANG interpreter defects land, and they are charged to the model.

**Current State (measured on `eval_results/baselines/v0.32.0/standard`, 877 AILANG rows, 213 failures):**

| Class | Rows | Actually at fault | Banked as |
|---|---:|---|---|
| `docx_reimplement` + `markdown_reimplement` at 0% across 18 runs each | 36 | harness (`input_files` never reaches the standard-mode prompt) | compile/runtime/logic |
| AI-caps benchmarks run without `--ai` | 6 | harness (runner grants `--caps AI`, never passes `--ai`) | `runtime_error` |
| AILANG interpreter dictionary crash | 5 | AILANG (`missing dictionary method: prelude::Fractional::Int::add`) | `runtime_error` |
| Correct values, extra output decoration | 17 | model — but a *formatting* failure, not a solve failure | `logic_error` |
| **Total not-a-capability-failure** | **64** | | |

**47 of 213 failures (22%) are not model capability failures.** Only **4 of 877** rows carry `validity.valid=false`.

**Impact:**
- **Model ranking is wrong in an unknown direction.** Contamination is not uniform — it concentrates in whichever benchmarks touch the broken paths, so it biases per-model scores by however often that model's code style trips an AILANG defect.
- **A live example.** In a 2026-08-13 core head-to-head, `or-deepseek-v4-pro-0813` scored 20/23 vs the incumbent's 21/23 and read as "the candidate is worse." Decomposed: one of its two losses was the `Fractional::Int` interpreter crash and the other was correct values with labels, while the incumbent's single loss was a genuine incomplete solve. The raw score inverted the true ordering, and the model-add gate would have acted on it.
- **It is self-concealing.** A contaminated row looks exactly like a capability failure, so the only way to find it today is to read stderr by hand, one row at a time.

---

## Goals

**Primary Goal:** Every banked failure carries a typed statement of *who failed* — the model, the harness, or AILANG — assigned at banking time, so no downstream consumer has to infer it from stderr.

**Success Metrics:**
1. Re-classifying the v0.32.0 baseline marks ≥ 47 of its 213 failures as non-capability, and the `--ai`, dict-crash and `input_files` classes reach 100% recall against the hand-audit in this doc.
2. `applyValidityBackstop` no longer keys on a single `error_category`; a row's validity is decided by attribution.
3. A `logic_error` that is purely formatting is separable from one that is a wrong answer, without changing anyone's pass/fail.
4. Zero silent denominator changes: every aggregate that drops rows reports how many it dropped.
5. A new harness-caused failure signature added later requires a table entry and a test, not a new `if` in a growing switch.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Attribution is a **new banked field**, not a reinterpretation of `error_category` | `error_category` is consumed by dashboards, ELO, curation and the OS publisher; overloading it breaks readers silently. A parallel field is additive. | human | design | high |
| Formatting failures keep **FAIL** semantics | Byte-exact stdout is the determinism contract AILANG is built on. Loosening it to rescue 17 rows would trade a measurement bug for a semantics change. | human | design | high |
| Historical rows are **annotated, never re-banked** | Precedent: the v0.30.0 cost-invalidation decision. Re-running changes the measurement; re-classifying does not. | human | design | med |
| Classification is **pure and offline-replayable** | Lets us re-attribute every historical baseline without API spend, and makes the classifier unit-testable against real banked rows. | agent | design | med |
| `output_format` as the new category name | Shared namespace with every other `ErrorCategory*`. Verified free (V9). | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Attribution is a new field alongside `error_category`, not a redefinition of it — **decided (Mark, 2026-08-13)**
- [x] Formatting failures stay FAIL; they get their own category for separability only — **decided (Mark, 2026-08-13)**
- [x] Historical baselines are annotated in place, not re-run — **decided (Mark, 2026-08-13)**, following the v0.30.0 precedent
- [ ] Whether re-attribution of past baselines republishes the dashboard, or only lands in the bank — **open, needs Mark**

---

## Solution Design

### Overview

Add a typed **attribution** to every banked run, decided by a table of signatures evaluated at banking time, and make validity a function of attribution instead of a function of one `error_category` value.

```
                     today                          proposed
 run outcome ─► CategorizeError(3 bools)   run outcome ─► CategorizeError(3 bools)   → error_category (unchanged)
                      │                                └─► AttributeFailure(row)      → attribution + validity
                      ▼                                          │
              error_category only                                ▼
              (validity only if api_error)         model | harness | ailang | benchmark
```

`error_category` keeps its exact current meaning and values — every existing reader is unaffected. Attribution is the new axis.

### Architecture

**Components:**

1. **`Attribution` enum** (new, `internal/eval_harness/attribution.go`)
   - `model` — the thing under test failed. Counts toward capability.
   - `harness` — our apparatus failed (missing flag, undelivered input files, infra). Does not count; should be retried.
   - `ailang` — the language/interpreter failed on code that was not the model's mistake. Does not count toward *model* capability; it is a language finding and should be routable as one.
   - `benchmark` — the spec cannot be satisfied as written (prose `expected_stdout`, requires an unimplemented language feature). Does not count.
   - `unknown` — classification declined. Conservative: treated as `harness`, per the existing "charged to us, never to the model" direction which this design finally makes true.

2. **Signature table** (new, data not control flow) — ordered `(pattern, attribution, reason)` rows matched against `stderr` + row shape. Seeded from the audit:

   | Signature | Attribution | Reason |
   |---|---|---|
   | `no AI model configured` | harness | runner granted `--caps AI` without `--ai` |
   | `missing dictionary method:` | ailang | interpreter dictionary-resolution defect |
   | benchmark declares `input_files` **and** mode is standard | harness | files never reach the prompt (see m-eval-standard-mode-input-files-gap) |
   | `expected_stdout` is a prose placeholder | benchmark | spec cannot byte-match by construction |
   | existing `api_error` catch-all | harness | preserves today's backstop behaviour |

3. **`applyValidityBackstop` rewritten** to consult attribution: any attribution other than `model` marks the row invalid with the reason carried through. The existing `api_error` behaviour becomes one row in the table rather than the only rule.

4. **`output_format` error category** (new value) — assigned when `!stdoutOk` **and** the produced stdout contains the expected values but differs in decoration. Attribution stays `model`; the row still FAILS. This is purely about separability: `logic_error` currently means both "wrong answer" and "right answer, labelled".

**Why a table and not more `if`s.** This is the fourth instance of one bug class. `ErrorCategoryNonAgentic`'s own comment records it: *"Third instance of this bug class after step_exhausted and max_steps."* Each was fixed by adding a substring test to a growing switch. Those three moved rows *out* of the `api_error` catch-all (harness-looking failures that were really known causes); this one is the mirror image — model-looking failures that are really harness or AILANG. Same class, opposite direction. A table with a test per row is the unified fix; a fifth `if` is the anti-pattern.

### Implementation Plan

**Phase 1: Attribution core** (~6 hours)
- `attribution.go`: enum, signature table, `AttributeFailure(row) (Attribution, reason string)` — pure, no I/O.
- Unit tests per signature, each with a real banked row from `eval_results/baselines/v0.32.0/` as the fixture.
- Golden test: re-classifying the v0.32.0 baseline reproduces the counts in this doc's Problem Statement table.

**Phase 2: Wire into banking + validity** (~5 hours)
- Populate `attribution` on `RunMetrics` at the same point `error_category` is set.
- Rewrite `applyValidityBackstop` to key on attribution; keep "an earlier stage that actively ruled wins".
- Add `output_format` and its detection.
- Verify `IsValid()`/`FilterValid` semantics are unchanged for rows that were already valid.

**Phase 3: Offline re-attribution + surfacing** (~5 hours)
- `ailang eval-reattribute <results_dir>` — re-runs classification over banked rows and writes attribution in place. No API calls, no re-running benchmarks.
- Report the excluded count wherever rows are dropped, following the `TokensCacheUnaccounted` idiom (`rotation_summary.go:56-59`) that W8 also adopts: a shrunken sample stated out loud, never silently.

### Files to Modify/Create

**New files:**
- `internal/eval_harness/attribution.go` (~180 LOC) — enum, table, classifier
- `internal/eval_harness/attribution_test.go` (~220 LOC) — per-signature tests + baseline golden
- `cmd/ailang/eval_reattribute.go` (~120 LOC) — offline re-classification command

**Modified files:**
- `internal/eval_harness/validity.go` (~+30/−12) — backstop keyed on attribution
- `internal/eval_harness/metrics.go` (~+12) — `Attribution` field, `ErrorCategoryOutputFormat`
- `internal/eval_harness/runner.go` (~+8) — pass `--ai` when caps include AI (the isolated bug; see Non-Goals on vision tier)
- `internal/eval_harness/agent_validation.go` (~+8) — same shape at the second call site (systemic, not case-by-case)

---

## Examples

### Example 1: an interpreter crash charged to the model

**Before** — `explicit_dataflow_ssa`, `or-deepseek-v4-pro-0813`, v0.33.0 core:
```json
{ "error_category": "runtime_error", "validity": null,
  "stderr": "Error: execution failed: missing dictionary method: prelude::Fractional::Int::add" }
```
Reads as: the model wrote code that crashes. Counts against it in pass rate and ELO.

**After:**
```json
{ "error_category": "runtime_error",
  "attribution": "ailang", "attribution_reason": "interpreter dictionary-resolution defect",
  "validity": { "valid": false, "reason": "ailang_defect" } }
```
Excluded from capability scoring, surfaced in the excluded count, and routable as the language finding it actually is.

### Example 2: right answer, wrong dress

**Before** — `tree_transformation_pipeline`: every value correct, `logic_error`, indistinguishable from a wrong answer.

```
expected:  [1, 2, 3, 4, 5, 6, 7]      got:  treeToList(tree): [1, 2, 3, 4, 5, 6, 7]
           28                               foldTree(add, 0, tree): 28
```

**After:** `error_category: "output_format"`, `attribution: "model"`, **still FAIL**. Pass rate is unchanged — but "cannot solve it" and "solved it, ignored the output contract" stop being the same number, and the second is a prompt-teaching signal rather than a capability one.

---

## Success Criteria

- [ ] `AttributeFailure` is pure and has a test per signature, each fixtured from a real banked row
- [ ] Re-classifying v0.32.0 marks ≥ 47 failures non-capability, matching the audit in this doc
- [ ] `applyValidityBackstop` keys on attribution; `api_error` is one table row, not the only rule
- [ ] `output_format` exists, does not change any pass/fail verdict, and is populated on the 17 identified rows
- [ ] `--ai` is passed wherever `--caps AI` is granted, at **both** call sites
- [ ] Every aggregate that excludes rows reports the excluded count
- [ ] No existing `error_category` value changes meaning; existing readers untouched
- [ ] `make test` green, `make lint` clean, `make check-boundaries` green

## Testing Strategy

**Unit tests:** one per signature-table row, fixtured from real banked rows under `eval_results/baselines/v0.32.0/`; `output_format` detection incl. the negative case (genuinely wrong values must stay `logic_error`); `unknown` falls back to harness, not model.

**Integration tests:** golden re-classification of the v0.32.0 baseline reproducing this doc's counts; a row already marked valid by an earlier stage is not overridden.

**Manual testing:** `ailang eval-reattribute` on a copied baseline; diff the before/after leaderboard and confirm every moved row is explained by a table entry.

---

## Relationship to W8 (load-bearing — read before scoping either)

[m-eval-validity-discipline](m-eval-validity-discipline.md) **W8** (P0, ailang#619) fixes the **consumer**: `SummarizeRotation` counts rows without ever reading `IsValid()`. This doc fixes the **producer**: almost nothing ever sets the flag.

They cover **disjoint failure populations**, and the split falls along mode:

| | rows W8 can filter today | why |
|---|---:|---|
| Tree-wide (`eval_results/**`) | **147** invalid, of 160 carrying any validity block (V16) | concentrated in agent/motoko rotations (`motoko_full_core_matrix` alone = 80) where `api_error` cascades are common — exactly the OS-board data W8 targets |
| Standard-mode release baseline (v0.32.0) | **4** of 877 (V4) | standard-mode failures almost never land in `api_error`, so the backstop never fires |

So **W8 is not inert — it is load-bearing for agent/rotation data**, and nothing here diminishes it. But on the standard-mode baselines that feed model ranking, curation and the model-add gate, W8 alone would move 4 rows out of 877, because the contamination there wears `runtime_error`/`logic_error` rather than `api_error`. This doc is what gives W8 something to filter in standard mode.

Sequence producer first, W8 second: attribution has to exist before filtering on it changes a published number.

W8 is currently parked inside its parent doc for a non-technical reason: it is bundled with the disputed W9. The parent states the open human decision is whether W8 may route in its own scoped doc. **Mark approved that split on 2026-08-13**; it is tracked as a companion to this doc.

---

## Verification Log

Every row re-derived by command on 2026-08-13 against `origin/dev` @ `fd01a37c1`. Negative-existence claims carry their own row with a positive control in the same call, so an empty result is a measurement and not a broken instrument.

| # | Claim | Command | Result | Verdict |
|---|---|---|---|---|
| V1 | `CategorizeError` sees only 3 booleans | read `metrics.go:212-223` | `switch` over `compileOk/runtimeOk/stdoutOk` only | TRUE |
| V2 | The backstop fires only for `api_error` | read `validity.go:133-145` | `if m.ErrorCategory != ErrorCategoryAPI { return }` | TRUE |
| V3 | Harness/AILANG rows carry no validity block | `jq` over matching rows | `validity=ABSENT` on all sampled `no AI model configured` and `missing dictionary method` rows | TRUE |
| V4 | Almost nothing is marked invalid | `jq 'select(.validity != null) \| .validity.valid'` over 877 rows | **4** `false`, no `true` | TRUE — W8 is near-inert alone |
| V5 | The runner never passes `--ai` | read `runner.go:278-302`; control: `--caps` IS appended at :291 | `--caps` present, `--ai` absent | TRUE |
| V6 | Same defect at the second call site | read `agent_validation.go:218-221` | identical shape, no `--ai` | TRUE — fix is systemic, not one-site |
| V7 | 3 benchmarks declare AI caps | `grep -l 'caps:.*"AI"' benchmarks/*.yml` | `ai_effect_json_schema`, `ai_effect_summarize`, `multi_agent_handoff` | TRUE |
| V8 | Those 3 are also unrunnable for a second reason | read their `expected_stdout` | prose placeholders, e.g. `<valid JSON with "name" and "age" keys, e.g. …>`; all `tier: vision` | TRUE — two independent defects |
| V9 | `output_format` is unallocated as an ErrorCategory | `grep -rn '"output_format"' internal/ cmd/`; control `"logic_error"` = 11 | 4 hits, **all** `internal/apiserver` docparse named-args — different namespace | FREE |
| V10 | IMP010 rows are model failures, not export gaps | `grep -E '\b(func\|type) X' std/*.ail`; control: `std/math.ail` exports `sin/cos/sqrt/pow/…`, `std/list.ail` exports `length/head/map/filter` | `bitwiseOr`, `max`, `contains`, `Map`, `Array` defined **nowhere** | **Models hallucinated the APIs — these 26 rows are GENUINE model failures and are excluded from the contamination count** |
| V11 | `dense_operator_program` is unsolvable | `grep -ciE 'bitwise\|shift' std/math.ail` = **0**; benchmark tagged `bitwise`,`operators` | AILANG has no bitwise support; every model fails it | TRUE — `benchmark` attribution |
| V12 | `commonmark_emphasis` is NOT a prose placeholder | read `expected_stdout`; pass rate in baseline | literal HTML (`<em>wind mill</em>`), **8 PASS / 10 FAIL** | FALSE — false positive of the placeholder heuristic, excluded |
| V13 | This is a recurring bug class | read `error_categorizer.go:5-18,89-99` | *"Third instance of this bug class after step_exhausted and max_steps"* | TRUE — table, not a 5th `if` |
| V14 | Two benchmarks sit at 0% across all models | per-benchmark pass rates over the baseline | `docx_reimplement` 0/18, `markdown_reimplement` 0/18 | TRUE — covered by m-eval-standard-mode-input-files-gap |
| V15 | A "shrunken sample stated out loud" idiom exists to follow | `rotation_summary.go:56-59` | `TokensCacheUnaccounted` | TRUE — reuse the shape |
| V16 | W8's target class is NOT empty tree-wide — scopes W8 fairly | `xargs grep -l '"valid":[[:space:]]*false'` over `eval_results/**`; control = rows carrying any `validity` block | **147** invalid of **160** with a validity block; top dirs `motoko_full_core_matrix` (80), `motoko_profile_matrix` (10), `ab_conv_docx_*` (20) | TRUE — **W8 is load-bearing for agent/rotation data**; the 4-of-877 figure is specific to standard-mode baselines and must not be quoted as "W8 does nothing" |

**Two claims in the originating analysis were refuted by V10/V12 and are corrected here rather than carried forward:** IMP010 is not an AILANG export gap (the symbols do not exist at all), and `commonmark_emphasis` is a runnable benchmark. The contamination count in the Problem Statement excludes both.

---

## Deferred Decisions

- Exact signature-matching mechanism (substring set vs compiled regex per row) — **agent may choose**; the requirement is that adding a signature is a table entry plus a test.
- Whether `ailang eval-reattribute` rewrites rows in place or writes a sidecar — **agent may choose**, provided it is idempotent and the original `error_category` survives.
- Whether `benchmark`-attributed rows are additionally excluded from *scheduling* (don't spend on a spec that cannot pass) or only from scoring — **agent may propose, Mark decides**; scheduling changes touch cost.

## Non-Goals

- **Loosening byte-exact stdout comparison.** Explicitly rejected: it is the determinism contract, and the teaching prompt already warns about it. `output_format` adds an axis, not leniency.
- **Re-running historical baselines.** Annotate in place; re-running changes the measurement (v0.30.0 precedent).
- **Fixing the AILANG defects themselves.** The `Fractional::Int` dict crash and the absence of bitwise operators are real findings, but they route to their own lanes. This doc makes them *visible and correctly attributed*, which is the prerequisite for routing them.
- **Deciding the fate of the vision tier.** Wiring `--ai` is in scope because it is a plain bug at two sites; whether `tier: vision` benchmarks with prose specs should be schedulable at all is a curation question for CURATION.md.
- **W8 itself.** Companion doc, sequenced after this one.

## Timeline

**Day 1** (~6 hours): Phase 1 — attribution core, signature table, per-signature tests, baseline golden.

**Day 2** (~5 hours): Phase 2 — banking wire-up, backstop rewrite, `output_format`, `--ai` at both sites.

**Day 3** (~5 hours): Phase 3 — `eval-reattribute`, excluded-count surfacing, docs + CHANGELOG.

**Total: ~16 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A signature over-matches and excludes a genuine model failure | High — silently inflates scores, the mirror of today's bug | Every signature is anchored on an apparatus-specific string (`no AI model configured`, `missing dictionary method:`), never a generic word; each has a negative test proving a real model failure is not caught |
| Attribution and `error_category` drift apart in readers | Med | `error_category` keeps its exact current values; attribution is additive and documented as the only capability-relevant axis |
| Landing without W8 looks like a no-op | Med | Explicitly sequenced: producer first, W8 second; the excluded-count surfacing makes the effect visible in the bank even before the boards change |
| Re-attributing history invalidates published comparisons | Med | Annotate-not-re-bank (v0.30.0 precedent); publishing is a separate, still-open freeze decision |
| Baseline goldens rot as the corpus changes | Low | Golden asserts per-class counts against fixtures copied into testdata, not against a live results directory |

## Related Documents

**Directly coupled:**
- [m-eval-validity-discipline.md](m-eval-validity-discipline.md) — W8 is the consumer half of this fix; see "Relationship to W8"
- [v0_33_1/m-eval-standard-mode-input-files-gap.md](v0_33_1/m-eval-standard-mode-input-files-gap.md) — owns the 36-row `docx`/`markdown_reimplement` class; this doc attributes those rows, that doc stops producing them
- [m-eval-reasoning-model-fairness.md](m-eval-reasoning-model-fairness.md) — same family: a measurement artifact read as model weakness

**Found by neural search (checked for overlap, distinct):**
- [v1_1_0/m-eval-trust-signals.md](v1_1_0/m-eval-trust-signals.md) (0.42) — trust *signals surfaced to users*; this doc is upstream attribution at banking time
- [v1_1_0/m-oracle-adequacy.md](v1_1_0/m-oracle-adequacy.md) (0.42) — whether a benchmark's oracle is adequate; adjacent to the `benchmark` attribution but scoped to oracle design
- [v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md](v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md) (0.40) — import boundaries, unrelated mechanism

## References

- [Design Axioms](/docs/references/axioms)
- ailang#619 — W8, harness errors scored as capability failures
- CLAUDE.md Critical Principle 2 (no silent fallbacks) and 3 (systemic fixes — audit before patching)

## Future Work

- Route `ailang`-attributed rows automatically into the language-fix backlog in PROGRAM.md — attribution makes AILANG defects a measurable stream rather than an anecdote.
- Track attribution mix over time as a rig-health metric; a rising `harness` share is an early warning the apparatus is degrading.

---

**Document created**: 2026-08-13
**Last updated**: 2026-08-13

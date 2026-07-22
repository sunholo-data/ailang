# M-DECISION-ENTROPY-MONITOR — Per-Step Decision-Weight Grading for AI Code-Generation Monitoring

**Status**: PARKED — needs-human-review (quorum-blocked ×2, iteration 84 2026-07-22; see ⛔ Quorum Record at bottom)
**Target**: v0.31.0 (monitoring/extension lane — zero language surface)
**Priority**: P2 (mission/eval infrastructure; not a v1.0 gate)
**Estimated**: ~2.5–3 days across 3 milestones (M3 evidence-gated on M2)
**Dependencies**: None. (M3's live flagging is gated on M2's validation report. `m-ai-reasoning-effort` is a *consumer*, not a dependency.)

## Executive Summary

During AI code generation (motoko agent loop, eval harness), some steps are **high-entropy
decisions**: choices that constrain or explode the space of remaining trajectories — changing an
exported signature or effect row, rewriting a whole file instead of editing it, taking the file
from type-checking to broken. Our own banked data already shows the phenomenon is real and
outcome-predictive: compile-preserving incremental edits converge, big-bang rewrites spiral
(the green-stability finding, [v1-mission-log.md](../../v1-mission-log.md)). But today the
monitoring plane records this only as **run-level post-mortems** (`finish_reason`,
`compaction_count`, tool histograms) — nothing grades the *step where the trajectory forked*.

This doc proposes a **decision-weight grade `D` per eligible step**, computed from signals we
already bank plus one small new capability (an interface-diff mode on the existing `ailang iface`
command). High-`D` steps are exactly the ones worth close examination — by the observatory, a
judge model, or a human. AILANG is unusually well-positioned for this: the normalized module
interface (signatures **including effect rows**, ADT constructors) gives a language-native,
deterministic measure of an edit's blast radius that plain-text diff size cannot.

**Route (PROGRAM.md)**: extension. Grader lives in `tools/` + observatory; the only
compiler-adjacent piece is a pure additive diff function over the already-normalized
`InterfaceJSON`.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | `D` is a deterministic function of banked session data; iface diff operates on the already-normalized, canonical `InterfaceJSON` (sorted effect rows, canonical type vars) |
| A2: Replayability | +1 | Grader is fully offline-recomputable from `session_*.jsonl`; no live-only state |
| A3: Effect Legibility | +1 | Effect-row changes become a first-class, machine-visible severity signal in the diff verdict |
| A4: Explicit Authority | 0 | No capability/authority changes |
| A5: Bounded Verification | +1 | Interface delta is locally checkable (two files in, verdict out) — no whole-program analysis |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Decision profiles are machine-parseable JSON; primary consumer is the loop itself (routing effort/branching), not humans |
| A8: Minimal Syntax | 0 | Zero language surface; one CLI flag on an existing command |
| A9: Cost Visibility | +1 | Makes the *consequence cost* of a generation decision visible at the step where it is paid — the doc's primary goal |
| A10: Composability | +1 | Composes with existing best-of-N selector, reasoning-effort routing, and observatory without modifying them |
| A11: Structured Failure | +1 | Failure is a first-class structured output: `--diff` ALWAYS emits machine-readable JSON and exits 0 — even on uncompilable input it emits a distinct `unavailable` verdict with per-side status, never raw diagnostics on stdout; the grader records a structured `iface_status` per step instead of coercing missing data to 0 |
| A12: System Boundary | 0 | No boundary crossings added; respects core/dashboard split (see Conflict Surface) |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Machine-parseable output is the primary interface

## Verification Log

Every load-bearing premise below was live-verified on 2026-07-22 at HEAD (v0.30.0). Negative-existence
claims carry their grep per design-doc policy.

| # | Claim | Evidence |
|---|-------|----------|
| V1 | **No interface diff/compare function exists** in `internal/iface/` (the diff is genuinely unbuilt) | `grep -rn "func.*Compare\|func.*Diff\|func.*Delta" internal/iface/*.go` → only `constructor_test.go:139 TestDigestDifferentConstructors` (a test name, not a diff API) |
| V2 | Normalized, deterministic interface serialization exists | `internal/iface/json.go:46` `func (i *Iface) ToNormalizedJSON()`; `internal/iface/iface.go:17` `Digest string // Deterministic digest of interface` |
| V3 | **`ailang iface` command already exists and already prints the normalized JSON** (so M1 *extends* it — it does NOT create a new command) | `cmd/ailang/main.go:201` (`case "iface":`, `--compact` flag); `cmd/ailang/check.go:466` `outputInterface` → `result.Interface.ToNormalizedJSON()`. Live run: `ailang iface --compact examples/deriving_eq.ail` → `module examples/deriving_eq` / `main : (())->()!{IO}` (effect row present) |
| V4 | Per-edit typecheck verdict is attached to every write in the agent session stream | `tools/analyze_stuck.py:6-7,180-202` — edit tools attach `payload.typecheck` (`ail_typecheck_after_edit` runs `ailang check` on the written file) to every WriteFile/EditFile/EditDecl result |
| V5 | **No token-level uncertainty capture exists anywhere** (logprobs/entropy) | `grep -rin "logprob" internal/ tools/ cmd/` → empty |
| V6 | Run-level convergence signals are already banked | `internal/eval_harness/metrics.go:72` `AgentToolHistogram`; `metrics.go:79-81` `compaction_count` / `first_compaction_step` / `compaction_level_max` (documented as convergence-thrash indicators) |
| V7 | Best-of-N selector quality is already a banked first-class metric (nearest existing "decision quality" signal) | `internal/eval_harness/rotation_summary.go:41` `BestOfNPass`, `:65-66` `BestOfNExact`/`BestOfNCeiling` |
| V8 | `AILANG_TRACE` is the *program-execution* plane, not the agent-edit plane — per-step agent decisions live only in the motoko `session_*.jsonl` consumed by `tools/analyze_*.py` and `internal/observatory/importer_motoko.go` | `internal/trace/schema.go:1-8` states it explicitly ("captures what happens when AILANG code evaluates… complementary to the chains/observatory system") |
| V9 | **No numeric diff-size or rewrite-vs-edit field is banked** — move class is heuristic-only in offline tools today | `tools/analyze_run_steps.py:48-68`, `tools/analyze_stuck.py:68-85` (WriteFile vs EditFile vs bash heredoc/redirect inference); no such field in `RunMetrics` |
| V10 | Cited regression fixture exists and compiles through the iface path | `examples/deriving_eq.ail` — live `ailang iface` run in V3 |
| V11 | **Producer-side (session stream, not just consumer scripts)**: WriteFile/EditFile tool RESULTS carry a per-edit `typecheck` verdict — as a **string** (`"ailang check: OK - file type-checks."` / `"ailang check: FAILED - fix these errors…"`), not a boolean; result payload also carries `path`, `bytes_written`, `sha256`, `diff`, `exit_code` | Live JSON-walk (python3) of a real banked session, `~/dev/mk-ast/.motoko/logfile/session_docx_reimplement_dc61a33a.jsonl` (2026-07-22): typecheck-bearing results = `{WriteFile: 54, EditFile: 10}`; sample WriteFile result keys: `path, bytes_written, sha256, diff, typecheck, exit_code`. The grader parses the `OK`/`FAILED` prefix — same contract as `analyze_stuck.py` |
| V12 | **Producer-side**: WriteFile tool-call `arguments` bank the FULL file content (`{path, content}`; observed `len(content)=1211` == result `bytes_written: 1211`) → post-state reconstruction is exact for rewrites. Initial-state availability is machine-detectable: the first write to a path has result `diff` beginning `--- /dev/null` (file did not exist before) | Same session walk as V11. **Honest gap**: EditFile/EditDecl results bank a unified `diff`, NOT full content — before-state reconstruction on edit chains is NOT verified here and stays deferred; v1 scopes iface scoring to WriteFile steps only (per prereg), with coverage reported |
| V13 | Current one-arg `ailang iface` on uncompilable input emits **raw diagnostics and exits 1** — exactly the behavior `--diff` must NOT inherit (motivates the always-0 structured-JSON contract below) | Live at HEAD (v0.30.0-116-ge6d5f85c9): `ailang iface /tmp/broken_iface_test.ail` (file with a parse error) → `Error: module loading error: … PAR_UNEXPECTED_TOKEN at …5:1 …`, `exit=1` |

## Problem Statement

**Current State:**
- The monitoring plane banks rich **run-level** outcomes (`error_category`, `finish_reason`,
  `agent_tool_histogram`, compaction counters) and the raw per-step stream exists in
  `session_*.jsonl` — but no signal identifies **which step forked the trajectory**.
- Spirals are diagnosed **after the fact** by offline dossier tools (`analyze_stuck.py`), often
  dozens of wasted steps and thousands of tokens after the decisive wrong move.
- The green-stability finding ([v1-mission-log.md](../../v1-mission-log.md), docx corpus):
  compile-preserving incremental edits converge at roughly 2–5× the rate of big-bang-rewrite
  trajectories. The *decision class* of a step predicts the outcome — and we don't grade it.
- Best-of-N selection (`select-best`, `rotation_summary.go`) operates at **whole-solution**
  granularity; there is no signal telling us *where* in a trajectory extra samples or extra
  reasoning effort would pay.

**Impact:**
- Wasted rig time and tokens on trajectories that were diagnosably doomed at a specific early step.
- Close-examination effort (human or judge-model) is spent uniformly or not at all, instead of
  concentrated on the handful of steps that determine the run.

## Goals

**Primary Goal:** Grade every **eligible** agent step with a decision-weight `D` so that
high-consequence decisions are flagged for close examination — validated *offline against labeled
outcomes before any live wiring*. Eligibility is explicit, not silent: the iface feature is a
**partial signal** (see `iface_status` below) and its per-run coverage is a reported, gating
number — steps where iface is unavailable are still graded from the remaining verified-available
features, with the unavailability recorded, never zero-filled.

**Success Metrics:**
- M1: `ailang iface --diff` classifies a golden set of edit pairs (body-only / additive /
  breaking / **uncompilable**) with 100% agreement to hand labels, deterministically — including
  the always-exit-0 structured-JSON verdicts on uncompilable input.
- M2: On the existing labeled corpus (compile-stuck spirals vs converged runs), the max early-step
  `D` separates the two classes with a pre-registered threshold (target: AUC ≥ 0.75 on the
  labeled docx corpus; exact criterion frozen in the M2 prereg before analysis). The prereg also
  freezes a **minimum iface-coverage threshold**: the fraction of graded steps with
  `iface_status: available` is reported per run and per corpus, and the validation report fails
  loudly (does not silently pass) if coverage falls below the preregistered floor.
- M3 (only if M2 validates): per-run decision profile banked for every agent-mode eval run;
  observatory lists the top-`D` step per run with its feature breakdown.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Consequence measure is the **interface delta** (signatures + effect rows + ctors), not text-diff size | Decouples "big decision" from "big edit": a 1-token effect-row change is breaking; a 200-line body rewrite behind a stable iface is low-consequence. AILANG-native; deterministic (V2) | human (this doc) | design | med |
| **Offline-first**: M2 validates on banked corpora before anything touches the live loop | Prevents shipping alarm noise; honors the ground-conclusions-in-data rule | human (this doc) | design | high |
| M1 extends the **existing** `ailang iface` command with a diff mode (V3) — no new command, no `ai-check` schema change — and lands BEFORE the grader that shells out to it | Reuse over reinvention; `ai-check` JSON contract stays frozen; no milestone depends on a not-yet-built tool | human (this doc) | design | low |
| `--diff` failure contract: **always exit 0, always one structured JSON verdict on stdout**, even on uncompilable input (distinct `unavailable` verdict, per-side status) — never raw diagnostics | The primary consumer is a grader subprocess analyzing green→red steps whose post-edit state is uncompilable BY DESIGN; a diagnostics dump + nonzero exit would crash it on exactly those steps (V13). Machine-to-machine failure behavior is design, not implementer discretion (A7, A11) | human (this doc, Rev-2) | design | low |
| `D` v1 feature set: move-class, green→red `typecheck` transition, per-path churn, iface-delta severity, early-step multiplier | These are the only signals verified available (V4, V6, V9); logprobs are confirmed absent (V5) and out of scope | human (this doc) | design | med |
| `D` functional form and initial weights | Hand-weighted v1 is a starting point; M2's job is to re-fit or refute against outcomes | agent (implementer, within M2 prereg) | compile | low |
| What high-`D` *triggers* live | This doc ships flagging/surfacing ONLY; intervention (best-of-N branching at the step, reasoning-effort escalation) is future work with its own evidence bar | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Consequence measure = interface delta (ratified by authoring this doc; revisit only with data)
- [x] Offline-first sequencing; M3 is evidence-gated on M2
- [x] `--diff` uncompilable-input contract: always exit 0 with one structured JSON verdict
  (ratified Rev-2; resolves the quorum block)
- [ ] M2 prereg frozen (validation criterion + feature set + labeled corpus manifest + subprocess
  timeout value + minimum iface-coverage threshold) — resolve at sprint-plan time, BEFORE running
  the analysis (m-eval-fmt-weakmodel-ab prereg is the template)
- [ ] M2→M3 promotion: a human reviews the M2 validation report before live banking/flagging ships

## Solution Design

### Overview

Three components, deliberately layered so each is independently useful:

1. **`ailang iface --diff A.ail B.ail`** (M1) — pure additive diff over two `InterfaceJSON` blobs
   with a severity verdict (`none` / `additive` / `breaking` / `unavailable`). Built FIRST,
   because the grader shells out to it. Independently useful: agents can check their own blast
   radius before committing an edit.
2. **Offline decision grader** (`tools/analyze_decisions.py`, M2) — reads `session_*.jsonl`,
   emits a per-step decision profile and per-run summary. Pure consumer; zero changes to motoko
   or the harness.
3. **Banking + surfacing** (M3, gated on M2) — per-run `decision_profile` in `RunMetrics`,
   observatory view of the top-`D` steps.

### The decision-weight grade

Per step *t*, from verified-available signals only:

```
D(t) = w_m·moveClass(t) + w_g·greenToRed(t) + w_c·churn(path(t)) + w_i·ifaceSeverity(t)
       , scaled by earlyStepMultiplier(t)
```

- **moveClass** — EditDecl/EditFile (0) < WriteFile full rewrite (1) < bash heredoc-write (1.5,
  bypasses the per-edit typecheck rail). Source: tool name + cmd heuristic (V9).
- **greenToRed** — 1 if this edit's `payload.typecheck` goes FAILED while the previous state of
  that file was OK (V4). The green-stability finding says this is the decisive event class.
- **churn** — count of prior edits to the same path (the 61-rewrite pathology).
- **ifaceSeverity** — an **explicitly partial** signal. Every graded step emits a structured
  `iface_status` ∈ {`available`, `after_compile_failed`, `before_unavailable`, `snapshot_missing`,
  `tool_error`, `timeout`}. ONLY `available` may carry a numeric severity (`none`=0 /
  `additive`=0.5 / `breaking`=1 from M1's `--diff`), computed between the file's previous
  materialized state and the post-edit state. An unavailable status is NEVER coerced to 0
  (no-silent-fallback axiom): for those steps `D` is computed from the remaining
  verified-available features (moveClass, greenToRed, churn) and the profile records iface as
  unavailable — the feature is *absent*, not zero. Structural expectation, not a gap: on a
  green→red step the post-edit state fails to compile BY DESIGN, so `iface_status:
  after_compile_failed` is the norm there — the greenToRed feature itself already captures that
  decision class. WriteFile banks full `content` (V12), so state reconstruction is exact for
  rewrites; EditFile/EditDecl reconstruction is a deferred decision (below) — v1 runs
  iface-severity on WriteFile steps only, with coverage reported against the preregistered floor.
- **earlyStepMultiplier** — decisions in the first third of the budget have a larger downstream
  cone; linear decay to 1.0.

Weights are v1 priors; M2 re-fits them against labeled outcomes (logistic fit or simple grid) and
reports which features carry the separation. If a feature carries nothing, it is dropped — the
grade must stay explainable (each flagged step lists its firing features, never a bare number).

### `ailang iface --diff` verdict shape

**Contract (machine-to-machine, A7/A11): `--diff` ALWAYS emits exactly one structured JSON
verdict on stdout and ALWAYS exits 0** — including when one or both inputs fail to
parse/type-check or a snapshot is missing. Raw compiler diagnostics NEVER appear on stdout in
`--diff` mode (a human-readable summary may go to stderr). This is deliberately the opposite of
the one-arg form's current behavior (V13: diagnostics + exit 1): the primary consumer is a grader
subprocess, and the design specifically targets green→red steps whose post-edit state is
uncompilable by definition — the failure path IS the hot path.

Both sides compilable:

```json
{
  "a": "before.ail", "b": "after.ail",
  "a_status": "available", "b_status": "available",
  "compile_error": false,
  "digest_equal": false,
  "severity": "breaking",
  "funcs": {
    "added": [], "removed": [],
    "changed": [{"name": "main", "from": "(())->()!{IO}", "to": "(())->()!{IO,Net}", "effect_row_changed": true}]
  },
  "types": {"added": [], "removed": [], "ctors_changed": []}
}
```

Either side uncompilable (the guaranteed case on a green→red step):

```json
{
  "a": "before.ail", "b": "after.ail",
  "a_status": "available", "b_status": "compile_failed",
  "compile_error": true,
  "severity": "unavailable",
  "funcs": null,
  "types": null
}
```

`a_status`/`b_status` ∈ {`available`, `compile_failed`, `snapshot_missing`}; `compile_error` is
true iff either side is not `available`. Severity: `none` (normalized JSONs equal) · `additive`
(only additions) · `breaking` (any removal or change to an exported signature, effect row, or
constructor set) · `unavailable` (either side not `available`). The uncompilable case is a
**distinct verdict** — it is NEVER silently coerced to `none` (no-silent-fallback axiom).
Deterministic by construction (V2 normalization).

### Architecture / boundaries

- `internal/iface` (core): new pure function `DiffInterfaces(a, b *InterfaceJSON) *IfaceDiff`
  — data-in/data-out, no new imports. Core does not gain any dashboard dependency.
- `cmd/ailang/check.go` (CLI): `--diff` mode on the existing `iface` subcommand; two-arg form,
  existing one-arg + `--compact` behavior byte-identical.
- `tools/analyze_decisions.py`: follows the `analyze_stuck.py` field contract (V4, V11–V12);
  shells out to `ailang iface --diff` for severity — no Python reimplementation of the compiler
  surface. **Bounded shell-out, fail-loud**: each invocation runs under a hard subprocess timeout
  (10s per pair; exact value frozen in the M2 prereg). On timeout the subprocess is killed and
  the step records `iface_status: timeout`. On a nonzero exit or non-JSON stdout — both contract
  violations under the always-0 contract above — the step records `iface_status: tool_error` and
  the grader prints a loud warning and counts the violation in the run summary. The grader never
  hangs on the diff tool and never silently passes over a failed invocation.
- M3 only: `internal/eval_harness/metrics.go` (+`decision_profile`), `internal/observatory`
  importer/view. Dashboard consumes the CLI/JSON, never the compiler surface directly —
  boundaries per `make check-boundaries`.

### Implementation Plan

**M1: `ailang iface --diff`** (~0.5–1 day) — additive, independently useful; lands FIRST because
the M2 grader shells out to it
- [ ] `DiffInterfaces` in `internal/iface` + unit tests (golden pairs: body-only → `none`;
  new export → `additive`; effect-row / signature / ctor change and removal → `breaking`)
- [ ] Uncompilable-input contract tests: A broken / B broken / both broken → one structured
  `unavailable` verdict (per-side status, `compile_error: true`), **exit 0**, no raw diagnostics
  on stdout; missing file → `snapshot_missing`, exit 0
- [ ] CLI wiring on the existing subcommand; regression: one-arg form and `--compact` unchanged
  (fixture: `examples/deriving_eq.ail`, V10)
- [ ] `cli-doc-maintainer` pass for help text (documents the always-exit-0 `--diff` contract)

**M2: Offline validation on banked corpora** (~1 day) — no live changes
- [ ] Freeze prereg: feature set, `D` v1 weights, labeled corpus manifest (docx compile-stuck
  spirals vs converged runs, from the existing `analyze_stuck.py` dossiers), separation
  criterion, subprocess timeout value, and the **minimum iface-coverage threshold**
  (WriteFile-only iface scoring is acceptable v1; coverage is reported and gating)
- [ ] Build `tools/analyze_decisions.py` (per-step profile incl. `iface_status` + per-run summary
  JSON incl. iface-coverage %)
- [ ] Run against corpus; produce validation report: separation achieved? which features carry
  it? re-fit weights; iface coverage vs the preregistered floor
- [ ] Verdict published honestly (incl. null result — fmt-weakmodel-ab precedent). Null ⇒ M3 does
  not ship; M1 has already landed (independently useful)

**M3: Banking + observatory surfacing** (~1 day) — **gated on M2 validation + human review**
- [ ] `decision_profile` on `RunMetrics` (max-`D` step, feature breakdown, counts:
  rewrite moves / green→red transitions / breaking edits)
- [ ] Populate in the agent-mode bank path; observatory top-`D` view
- [ ] `make check-boundaries` green; CHANGELOG entry

### Files to Modify/Create

**New files:**
- `tools/analyze_decisions.py` — offline grader (~250 LOC)
- `internal/iface/diff.go` + `internal/iface/diff_test.go` — pure diff (~150 + ~150 LOC)

**Modified files:**
- `cmd/ailang/check.go` / `cmd/ailang/main.go` — `--diff` mode (~60 LOC)
- M3 only: `internal/eval_harness/metrics.go` (~20 LOC), bank path + observatory (~100 LOC)

## Examples

### Example 1: The spiral, seen at the step instead of the post-mortem

**Before (today):** run banks `finish_reason: step_exhausted`, `agent_tool_histogram:
{WriteFile: 61, BashExec: 93}`. The post-mortem dossier shows the file never type-checked. The
decisive step is invisible.

**After:** step 7 flagged `D = 2.9` — features: `WriteFile full rewrite` + `green→red`
(`typecheck: FAILED` on a previously-OK file) + `breaking` (main's effect row changed
`!{IO}` → `!{IO,Net}`). The observatory shows the run forked at step 7; 54 of the 61 rewrites
came after it.

### Example 2: Blast-radius check as a standalone tool

```bash
$ ailang iface --diff before.ail after.ail
{"severity": "breaking", "funcs": {"changed": [{"name": "main", "effect_row_changed": true, ...}]}, ...}
```

A 200-line refactor that reports `severity: none` is a low-consequence edit no matter how large
the text diff — and an agent can know that before committing to it.

## Conflict Surface

This design touches `internal/iface/` and `cmd/ailang/`, so per policy:

1. **Positions extended**: (a) the `ailang iface` CLI flag/arg surface — a `--diff` mode taking
   two args; (b) the `internal/iface` package API — one new exported pure function. **Zero
   syntactic/semantic language positions**: no parser, typechecker, elaborator, or eval change.
2. **Existing constructs in those positions**: the one-arg `iface <module|file.ail>` form and the
   `--compact` flag (V3); `ToNormalizedJSON`'s existing consumers (digest computation, tests).
   The normalized-JSON *format* is read, never written to — no schema change.
3. **Disambiguation**: `--diff` requires exactly 2 positional args; absent the flag, the existing
   1-arg path runs unchanged, same usage error on 0 args.
4. **Programs that MUST still work** (regression fixtures): `ailang iface examples/deriving_eq.ail`
   and `ailang iface --compact examples/deriving_eq.ail` byte-identical to pre-change output
   (live-verified baseline in V3/V10); `internal/iface` test suite green; `make check-boundaries`
   green.
5. **Deliberate incompatibilities**: none. Everything is additive.

## Success Criteria

- [ ] `ailang iface --diff` deterministic and 100% correct on the golden pair set — including
  uncompilable-input pairs: structured `unavailable` verdict, exit 0, no raw diagnostics on
  stdout (acceptance tests in `diff_test.go`)
- [ ] M2 validation report published with pre-registered criterion, honest verdict (null
  included), and per-run iface-coverage % reported against the preregistered floor
- [ ] Existing `iface` one-arg/`--compact` output byte-identical (regression test)
- [ ] M3 (if promoted): `decision_profile` present on new agent-mode runs; observatory renders top-`D` step
- [ ] All tests passing; `make check-boundaries` green
- [ ] CHANGELOG.md + CLI help updated

## Testing Strategy

**Unit tests:** `DiffInterfaces` golden pairs (none/additive/breaking × funcs/effects/ctors/removals); severity total ordering; determinism (same pair → same bytes); uncompilable-input verdicts (A broken / B broken / both / missing file) → `unavailable` with correct per-side status.

**Integration tests:** CLI `--diff` end-to-end on fixture pairs; `--diff` on a broken fixture asserts **exit code 0 and parseable JSON on stdout** (the contract test); regression on the one-arg form; grader smoke test on a committed miniature session JSONL fixture, including a forced-timeout case asserting `iface_status: timeout` (no hang) and a contract-violation case asserting `iface_status: tool_error` with a loud warning.

**Manual testing:** run `tools/analyze_decisions.py` over one known-spiral and one known-converged docx session; eyeball that flagged steps match the dossier narrative from `analyze_stuck.py`.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `D` functional form/weights within the frozen M2 feature set — implementer, guided by the M2 fit
- EditFile/EditDecl intermediate-state reconstruction (apply-edits vs WriteFile-only iface scoring in v1) — implementer; WriteFile-only is an acceptable v1 per the prereg (V12 documents why: EditFile banks a diff, not full content), with coverage % reported against the preregistered floor
- ~~`--diff` exit-code semantics~~ — **RESOLVED (Rev-2)**: always exit 0 with one structured JSON verdict, including on uncompilable input (see verdict-shape contract). Not implementer-deferred: the failure behavior of a machine-to-machine interface is part of the design (A7)
- Whether the grader also emits a chains/observatory event stream or only per-run JSON — implementer

## Non-Goals

- **Live intervention** (best-of-N branching at high-`D` steps, reasoning-effort escalation,
  pausing the agent) — future work; needs its own evidence bar after M3 data accrues
- **Token-level uncertainty capture** (logprobs/entropy from providers) — confirmed absent today
  (V5); a possible fourth signal later, not needed for v1
- **motoko-fork changes** — the grader is a pure consumer of the existing session stream (V4)
- **`ai-check` JSON schema changes** — its contract stays frozen
- **A language-level entropy construct** — that is [m-entropy-budgets](../v1_1_0/m-entropy-budgets.md)'s
  territory (design-time responsibility assignment); this doc is trajectory *measurement*

## Timeline

**Day 1:** M1 diff function + CLI + tests, incl. the uncompilable-input contract (lands
regardless of the later validation verdict)
**Day 2:** M2 prereg + grader + validation report
**Day 3 (conditional):** M3 banking + observatory, only after M2 report review

**Total: ~2.5–3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Alarm noise: thresholds flag everything (or nothing) | High | M2 offline gate with pre-registered criterion; null result blocks M3, published honestly |
| Session JSONL schema drift breaks the grader | Med | Pin to the same field contract as `importer_motoko.go` / `analyze_stuck.py` (V4, V8, V11–V12); grader fails loudly on missing fields — no silent fallback |
| Iface-severity coverage gap on EditFile-heavy runs (WriteFile-only v1) | Med | Banked corpus is WriteFile-dominated (the 61-rewrite pathology); coverage % reported per run against the preregistered minimum threshold (M2 prereg) — visible AND gating, not silent |
| `--diff` subprocess hangs or emits garbage on a pathological input | Med | Bounded 10s timeout (kill + `iface_status: timeout`); contract violations recorded as `iface_status: tool_error` with loud warning — never an unbounded wait, never a silent pass |
| Confounding: high-`D` correlates with hard benchmarks, not bad decisions | Med | M2 compares within-benchmark where the corpus allows; report states the limit explicitly |
| Grade misread as model quality ("model wall") | Low | Grade is per-*decision*, features listed; report language follows the harness-first diagnosis rule |

## Related Documents

**Planned (distinctions confirmed at authoring):**
- [m-entropy-budgets](../v1_1_0/m-entropy-budgets.md) (0.43) — *distinct*: design-time framework
  assigning **responsibility** for ambiguity resolution (annotations/envelopes). This doc
  **measures** decision consequence in live trajectories. Long-term, `D` profiles could inform
  entropy-budget resolver assignment (Future Work)
- [m-ai-reasoning-effort](../v0_29_0/m-ai-reasoning-effort.md) — *consumer*: high-`D` steps are
  where request-side reasoning escalation would attach once both land
- [m-eval-regression-detector-contract](../v0_29_0/m-eval-regression-detector-contract.md) (0.36) — run-level regression detection; orthogonal granularity
- [m-mission-cost-chains](../v0_30_0/m-mission-cost-chains.md) (0.35) — cost attribution, not decision grading

**Implemented (methodology + substrate):**
- [m-eval-fmt-weakmodel-ab](../../implemented/v0_31_0/m-eval-fmt-weakmodel-ab.md) — the prereg +
  honest-null template M2 follows
- [m-benchmark-data-integrity](../../implemented/v0_18_6/m-benchmark-data-integrity.md) (0.36) — banked-data validity discipline
- [m-otel-enhanced-tracing-dx](../../implemented/v0_7_0/m-otel-enhanced-tracing-dx.md) (0.34) — execution-plane tracing (V8 explains why it is NOT the substrate here)

## References

- [Design Axioms](/docs/references/axioms)
- Green-stability finding + docx corpus dossiers: [v1-mission-log.md](../../v1-mission-log.md), `tools/analyze_stuck.py`
- Best-of-N selector metrics: `internal/eval_harness/rotation_summary.go`
- Session-stream field contract: `internal/observatory/importer_motoko.go`

## Future Work

- Intervention at high-`D` steps: best-of-N branching *at the decision point* (sample alternatives
  only where entropy is high, select with the existing runs>typechecks selector); reasoning-effort
  escalation via m-ai-reasoning-effort
- Provider-side uncertainty (ollama logprobs) as a fourth feature
- Outcome-conditioned weight learning at fleet scale (mutual information between early decisions
  and final pass/fail across N-run aggregates)
- Feeding `D` profiles into m-entropy-budgets resolver assignment

---

**Document created**: 2026-07-22
**Last updated**: 2026-07-22 (Rev-2)

## Revision History

Rev-2 (2026-07-22): resolved quorum block — defined --diff uncompilable-input contract (always-0
structured JSON), made ifaceSeverity explicitly partial (no coerce-to-0), reordered M2 before M1,
added bounded subprocess timeout, narrowed coverage claim + prereg threshold, added producer-side
verification rows.

---

## ⛔ Quorum Record — PARKED needs-human-review (iteration 84, 2026-07-22)

Two independent quorum rounds (gpt5-6-sol + gemini-3-1-pro, reject-by-default; controller
verdict PASS both rounds) **BLOCKED** this doc. Per the mission's QUORUM-AT-PICK gate (revise
once, re-quorum once, still-rejected → park), it is parked for a **human design decision**.
Artifacts: `.ailang/state/mission-quorum/m-decision-entropy-monitor-2026-07-22T20-47-07Z.json`
(Rev-1) and `…20-56-31Z.json` (Rev-2). Metered reviewer cost total ≈ $0.138.

**Round 1 (Rev-1) — RESOLVED in Rev-2:**
- gpt5-6-sol & gemini-3-1-pro (converged): `ifaceSeverity` needs both files to compile, but the
  grade targets green→red steps whose post-edit file is broken by design → `ailang iface --diff`
  would emit raw diagnostics not JSON and crash the grader; "grade every step" had no structured
  unavailable result (no-silent-fallback violation). → **Fixed in Rev-2**: always-0 structured-JSON
  `--diff` contract, partial `iface_status` never coerced to 0, M2-before-M1 reorder, bounded
  timeout, coverage-threshold prereg, producer-side V11–V13.

**Round 2 (Rev-2) — the STILL-OPEN blocks that park this doc (need Mark's call):**

1. **Determinism/replayability under live-binary replay (gpt5-6-sol, A1/A2).** The grader invokes
   the *current* `ailang` binary against reconstructed single-file states under a wall-clock
   timeout, but sessions do not bank compiler identity / build hash / workspace-import closure →
   the same `session_*.jsonl` can produce different iface verdicts across checkouts, binaries,
   machines, or timeout outcomes. So "`D` is a deterministic function of banked session data" is
   currently false for the iface feature.
   **Human fork:** (a) **bank normalized before/after interface JSON + compiler identity at
   collection time** (drop retrospective iface scoring; M3-collection-first — cleaner, honors
   offline-first, but requires a motoko/harness producer change, not a pure consumer); OR
   (b) **hermetic replay** (pin binary hash + flags + full workspace snapshot per corpus run,
   prove byte-identical replay). Treat timeout/missing-material as a run-level validation
   exclusion, not an ordinary feature status.

2. **Workspace/import resolution for `ailang iface` (gemini-3-1-pro, A5).** `ailang iface` needs
   the surrounding project/stdlib to resolve imports; extracting a standalone banked file to `/tmp`
   for the diff will hit instant module-loading failures on any file with imports — undercutting
   "locally checkable, two files in." (Reinforces fork 1(a): banking iface at collection time,
   where the workspace is live, sidesteps this entirely.)

3. **Conflict Surface gate miss + tool overlap (gemini-3-1-pro).** The Conflict Surface omits the
   new `tools/analyze_decisions.py` and the `internal/eval_harness/metrics.go` change, and the new
   grader overlaps `tools/analyze_stuck.py` / `tools/analyze_run_steps.py` (which already parse
   sessions for move-class + green→red). **Human/designer call:** extend the existing dossier tools
   vs. justify a standalone grader; then expand the Conflict Surface to cover the tool + metrics.
   (Also: Rev-2's milestone-reorder labeling was called internally inconsistent — fold into the
   same revision.)

**Recommended human resolution (controller's read, not a decision):** forks 1 and 2 both point at
**option 1(a) — bank normalized iface JSON + compiler identity at collection time** (a small
motoko/harness producer addition), which makes the grade genuinely deterministic AND removes the
workspace-resolution problem, at the cost of not being retro-computable on the *existing* corpus
(it would validate on a newly-banked corpus). If Mark prefers to keep it a pure offline consumer,
the honest scope is **M2 (`ailang iface --diff`) only** — which is independently useful and
un-blocked — with the `D`-grade iface feature deferred until collection-time banking exists.

This doc is itself *about* flagging high-consequence decisions made without enough information;
these three are exactly that class, so they are routed to the human rather than force-resolved.

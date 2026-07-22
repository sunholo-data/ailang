# M-DECISION-ENTROPY-MONITOR — Per-Step Decision-Weight Grading for AI Code-Generation Monitoring

**Status**: Planned
**Target**: v0.31.0 (monitoring/extension lane — zero language surface)
**Priority**: P2 (mission/eval infrastructure; not a v1.0 gate)
**Estimated**: ~2.5–3 days across 3 milestones (M3 evidence-gated on M1)
**Dependencies**: None. (M3's live flagging is gated on M1's validation report. `m-ai-reasoning-effort` is a *consumer*, not a dependency.)

## Executive Summary

During AI code generation (motoko agent loop, eval harness), some steps are **high-entropy
decisions**: choices that constrain or explode the space of remaining trajectories — changing an
exported signature or effect row, rewriting a whole file instead of editing it, taking the file
from type-checking to broken. Our own banked data already shows the phenomenon is real and
outcome-predictive: compile-preserving incremental edits converge, big-bang rewrites spiral
(the green-stability finding, [v1-mission-log.md](../../v1-mission-log.md)). But today the
monitoring plane records this only as **run-level post-mortems** (`finish_reason`,
`compaction_count`, tool histograms) — nothing grades the *step where the trajectory forked*.

This doc proposes a **decision-weight grade `D` per step**, computed from signals we already
bank plus one small new capability (an interface-diff mode on the existing `ailang iface`
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
| A11: Structured Failure | 0 | No error-surface changes (diff emits a structured verdict, not new diagnostics) |
| A12: System Boundary | 0 | No boundary crossings added; respects core/dashboard split (see Conflict Surface) |

**Net Score: +7** → **Decision: Move forward**

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
| V3 | **`ailang iface` command already exists and already prints the normalized JSON** (so M2 *extends* it — it does NOT create a new command) | `cmd/ailang/main.go:201` (`case "iface":`, `--compact` flag); `cmd/ailang/check.go:466` `outputInterface` → `result.Interface.ToNormalizedJSON()`. Live run: `ailang iface --compact examples/deriving_eq.ail` → `module examples/deriving_eq` / `main : (())->()!{IO}` (effect row present) |
| V4 | Per-edit typecheck verdict is attached to every write in the agent session stream | `tools/analyze_stuck.py:6-7,180-202` — edit tools attach `payload.typecheck` (`ail_typecheck_after_edit` runs `ailang check` on the written file) to every WriteFile/EditFile/EditDecl result |
| V5 | **No token-level uncertainty capture exists anywhere** (logprobs/entropy) | `grep -rin "logprob" internal/ tools/ cmd/` → empty |
| V6 | Run-level convergence signals are already banked | `internal/eval_harness/metrics.go:72` `AgentToolHistogram`; `metrics.go:79-81` `compaction_count` / `first_compaction_step` / `compaction_level_max` (documented as convergence-thrash indicators) |
| V7 | Best-of-N selector quality is already a banked first-class metric (nearest existing "decision quality" signal) | `internal/eval_harness/rotation_summary.go:41` `BestOfNPass`, `:65-66` `BestOfNExact`/`BestOfNCeiling` |
| V8 | `AILANG_TRACE` is the *program-execution* plane, not the agent-edit plane — per-step agent decisions live only in the motoko `session_*.jsonl` consumed by `tools/analyze_*.py` and `internal/observatory/importer_motoko.go` | `internal/trace/schema.go:1-8` states it explicitly ("captures what happens when AILANG code evaluates… complementary to the chains/observatory system") |
| V9 | **No numeric diff-size or rewrite-vs-edit field is banked** — move class is heuristic-only in offline tools today | `tools/analyze_run_steps.py:48-68`, `tools/analyze_stuck.py:68-85` (WriteFile vs EditFile vs bash heredoc/redirect inference); no such field in `RunMetrics` |
| V10 | Cited regression fixture exists and compiles through the iface path | `examples/deriving_eq.ail` — live `ailang iface` run in V3 |

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

**Primary Goal:** Grade every agent step with a decision-weight `D` so that high-consequence
decisions are flagged for close examination — validated *offline against labeled outcomes before
any live wiring*.

**Success Metrics:**
- M1: On the existing labeled corpus (compile-stuck spirals vs converged runs), the max early-step
  `D` separates the two classes with a pre-registered threshold (target: AUC ≥ 0.75 on the
  labeled docx corpus; exact criterion frozen in the M1 prereg before analysis).
- M2: `ailang iface --diff` classifies a golden set of edit pairs (body-only / additive /
  breaking) with 100% agreement to hand labels, deterministically.
- M3 (only if M1 validates): per-run decision profile banked for every agent-mode eval run;
  observatory lists the top-`D` step per run with its feature breakdown.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Consequence measure is the **interface delta** (signatures + effect rows + ctors), not text-diff size | Decouples "big decision" from "big edit": a 1-token effect-row change is breaking; a 200-line body rewrite behind a stable iface is low-consequence. AILANG-native; deterministic (V2) | human (this doc) | design | med |
| **Offline-first**: M1 validates on banked corpora before anything touches the live loop | Prevents shipping alarm noise; honors the ground-conclusions-in-data rule | human (this doc) | design | high |
| M2 extends the **existing** `ailang iface` command with a diff mode (V3) — no new command, no `ai-check` schema change | Reuse over reinvention; `ai-check` JSON contract stays frozen | human (this doc) | design | low |
| `D` v1 feature set: move-class, green→red `typecheck` transition, per-path churn, iface-delta severity, early-step multiplier | These are the only signals verified available (V4, V6, V9); logprobs are confirmed absent (V5) and out of scope | human (this doc) | design | med |
| `D` functional form and initial weights | Hand-weighted v1 is a starting point; M1's job is to re-fit or refute against outcomes | agent (implementer, within M1 prereg) | compile | low |
| What high-`D` *triggers* live | This doc ships flagging/surfacing ONLY; intervention (best-of-N branching at the step, reasoning-effort escalation) is future work with its own evidence bar | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Consequence measure = interface delta (ratified by authoring this doc; revisit only with data)
- [x] Offline-first sequencing; M3 is evidence-gated on M1
- [ ] M1 prereg frozen (validation criterion + feature set + labeled corpus manifest) — resolve at
  sprint-plan time, BEFORE running the analysis (m-eval-fmt-weakmodel-ab prereg is the template)
- [ ] M1→M3 promotion: a human reviews the M1 validation report before live banking/flagging ships

## Solution Design

### Overview

Three components, deliberately layered so each is independently useful:

1. **Offline decision grader** (`tools/analyze_decisions.py`) — reads `session_*.jsonl`, emits a
   per-step decision profile and per-run summary. Pure consumer; zero changes to motoko or the
   harness.
2. **`ailang iface --diff A.ail B.ail`** — pure additive diff over two `InterfaceJSON` blobs with
   a three-way severity verdict (`none` / `additive` / `breaking`). Used by the grader (and
   generally useful: agents can check their own blast radius before committing an edit).
3. **Banking + surfacing** (M3, gated) — per-run `decision_profile` in `RunMetrics`, observatory
   view of the top-`D` steps.

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
- **ifaceSeverity** — `none`=0 / `additive`=0.5 / `breaking`=1 from M2, computed between the
  file's previous materialized state and the post-edit state. WriteFile banks full `content`
  (V4 stream), so state reconstruction is exact for rewrites; EditFile/EditDecl reconstruction
  is a deferred decision (below) — M1 may run iface-severity on WriteFile steps only.
- **earlyStepMultiplier** — decisions in the first third of the budget have a larger downstream
  cone; linear decay to 1.0.

Weights are v1 priors; M1 re-fits them against labeled outcomes (logistic fit or simple grid) and
reports which features carry the separation. If a feature carries nothing, it is dropped — the
grade must stay explainable (each flagged step lists its firing features, never a bare number).

### `ailang iface --diff` verdict shape

```json
{
  "a": "before.ail", "b": "after.ail",
  "digest_equal": false,
  "severity": "breaking",
  "funcs": {
    "added": [], "removed": [],
    "changed": [{"name": "main", "from": "(())->()!{IO}", "to": "(())->()!{IO,Net}", "effect_row_changed": true}]
  },
  "types": {"added": [], "removed": [], "ctors_changed": []}
}
```

Severity: `none` (normalized JSONs equal) · `additive` (only additions) · `breaking` (any removal
or change to an exported signature, effect row, or constructor set). Deterministic by construction
(V2 normalization).

### Architecture / boundaries

- `internal/iface` (core): new pure function `DiffInterfaces(a, b *InterfaceJSON) *IfaceDiff`
  — data-in/data-out, no new imports. Core does not gain any dashboard dependency.
- `cmd/ailang/check.go` (CLI): `--diff` mode on the existing `iface` subcommand; two-arg form,
  existing one-arg + `--compact` behavior byte-identical.
- `tools/analyze_decisions.py`: follows the `analyze_stuck.py` field contract (V4); shells out to
  `ailang iface --diff` for severity — no Python reimplementation of the compiler surface.
- M3 only: `internal/eval_harness/metrics.go` (+`decision_profile`), `internal/observatory`
  importer/view. Dashboard consumes the CLI/JSON, never the compiler surface directly —
  boundaries per `make check-boundaries`.

### Implementation Plan

**M1: Offline validation on banked corpora** (~1 day) — no live changes
- [ ] Freeze prereg: feature set, `D` v1 weights, labeled corpus manifest (docx compile-stuck
  spirals vs converged runs, from the existing `analyze_stuck.py` dossiers), separation criterion
- [ ] Build `tools/analyze_decisions.py` (per-step profile + per-run summary JSON)
- [ ] Run against corpus; produce validation report: separation achieved? which features carry it?
  re-fit weights
- [ ] Verdict published honestly (incl. null result — fmt-weakmodel-ab precedent). Null ⇒ M3 does
  not ship; M2 still lands (independently useful)

**M2: `ailang iface --diff`** (~0.5–1 day) — additive, independently useful
- [ ] `DiffInterfaces` in `internal/iface` + unit tests (golden pairs: body-only → `none`;
  new export → `additive`; effect-row / signature / ctor change and removal → `breaking`)
- [ ] CLI wiring on the existing subcommand; regression: one-arg form and `--compact` unchanged
  (fixture: `examples/deriving_eq.ail`, V10)
- [ ] `cli-doc-maintainer` pass for help text

**M3: Banking + observatory surfacing** (~1 day) — **gated on M1 validation + human review**
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

- [ ] M1 validation report published with pre-registered criterion, honest verdict (null included)
- [ ] `ailang iface --diff` deterministic and 100% correct on the golden pair set (acceptance test in `diff_test.go`)
- [ ] Existing `iface` one-arg/`--compact` output byte-identical (regression test)
- [ ] M3 (if promoted): `decision_profile` present on new agent-mode runs; observatory renders top-`D` step
- [ ] All tests passing; `make check-boundaries` green
- [ ] CHANGELOG.md + CLI help updated

## Testing Strategy

**Unit tests:** `DiffInterfaces` golden pairs (none/additive/breaking × funcs/effects/ctors/removals); severity total ordering; determinism (same pair → same bytes).

**Integration tests:** CLI `--diff` end-to-end on fixture pairs; regression on the one-arg form; grader smoke test on a committed miniature session JSONL fixture.

**Manual testing:** run `tools/analyze_decisions.py` over one known-spiral and one known-converged docx session; eyeball that flagged steps match the dossier narrative from `analyze_stuck.py`.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `D` functional form/weights within the frozen M1 feature set — implementer, guided by the M1 fit
- EditFile/EditDecl intermediate-state reconstruction (apply-edits vs WriteFile-only iface scoring in v1) — implementer; WriteFile-only is an acceptable v1 per the prereg
- `--diff` exit-code semantics (always-0-with-JSON vs diff-style 0/1/2) — implementer; document in help text either way
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

**Day 1:** M1 prereg + grader + validation report
**Day 2:** M2 diff function + CLI + tests (lands regardless of M1 verdict)
**Day 3 (conditional):** M3 banking + observatory, only after M1 report review

**Total: ~2.5–3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Alarm noise: thresholds flag everything (or nothing) | High | M1 offline gate with pre-registered criterion; null result blocks M3, published honestly |
| Session JSONL schema drift breaks the grader | Med | Pin to the same field contract as `importer_motoko.go` / `analyze_stuck.py` (V4, V8); grader fails loudly on missing fields — no silent fallback |
| Iface-severity coverage gap on EditFile-heavy runs (WriteFile-only v1) | Med | Banked corpus is WriteFile-dominated (the 61-rewrite pathology); coverage % reported per run in M1 so the gap is visible, not silent |
| Confounding: high-`D` correlates with hard benchmarks, not bad decisions | Med | M1 compares within-benchmark where the corpus allows; report states the limit explicitly |
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
  honest-null template M1 follows
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
**Last updated**: 2026-07-22

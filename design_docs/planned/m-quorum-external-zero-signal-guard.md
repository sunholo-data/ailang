# Quorum External Zero-Signal Guard

**Status**: Planned — candidate only; pending independent review capacity
**Target**: Next patch after approval (unversioned)
**Priority**: P1 — fleet charter #8, ailang#651
**Estimated**: 2 hours including regression verification (estimate, not measured)
**Dependencies**: Admitted independent review and normal planning/execution gates
**Planner-Lane**: codex-ok
**Created / revised**: 2026-10-08, fleet iteration 28, revision 1
**Author**: native codex:gpt-6.1-sol (role tokens: not reported)

## Problem Statement

At origin/dev `3e032cf4f13682a455129274d99097842c56e5ae`, `synthesize` counts present external reviewer outcomes, then increments the same count for a non-nil controller. Its zero-signal guard therefore permits all external reviewers to be absent when the controller passes. The result is `proceed`, although nobody outside the controller produced a verdict. This is a source-verified mechanism; this iteration has not run a new live provider quorum or counted production occurrences.

The existing `TestRunQuorum_AllAbsentRefusesToProceed` and `TestRunSeatedQuorum_NothingAnswersBlocks` pass a nil controller. They exercise zero participants, leaving the controller-pass combination uncovered. Fleet iteration 23 already verified the ticket and parked it on capacity; the index through iteration 27 records no implementation for this ticket. This candidate advances design only.

## Goals

Require at least one present external Tier-1 reviewer outcome before synthesis can proceed. A controller remains a participant with veto authority and a distinct artifact entry, but cannot supply the external-signal minimum. Keep N-1 degradation: one present passing reviewer and any number of named absences may proceed, provided neither controller nor another present reviewer rejects.

## High-Impact Decisions

| Decision | Why it matters | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Exclude controller from external-signal count | Prevents controller-only clearance | agent, subject to review | design | low |
| Preserve one-reviewer minimum and existing seating | Avoids introducing a full-seat or vendor-policy requirement | agent, subject to review | design | low |
| Keep Tier-2 additive, unable to unblock failed Tier-1 | Matches existing escalation composition | agent, subject to review | design | low |

### Design Freeze

This document has no outstanding human policy choice. It is not approval or execution authorization. Independent quorum, planning and execution gates remain pending. Unattended quorum is required; no quorum is run in this iteration because the controller reports native Sonnet unavailable (`Unknown model`) and the evaluator resolver refuses over-ration Anthropic/OpenRouter. Those capacity facts are controller-supplied, not independently measured by this author.

## Solution Design

In `internal/mission/quorum/quorum.go`, name the local count `externalPresentCount` (or equivalently reviewer-specific) and increment it only for present outcomes in the reviewer loop. Remove the controller increment. Preserve controller-reject processing and its objection. Apply the existing zero-signal guard to that external count, retaining its current objection text:

`no reviewer produced a verdict (all absent) — refusing to proceed on zero signal`

Update the synthesis and RunQuorum comments to state the external minimum explicitly. Keep absence records, controller recording, billing/token accumulation and JSON fields unchanged. An empty outcome list with any controller must also block; this covers the public Go API even though the CLI rejects an empty reviewers argument.

No separate CLI patch is needed: `cmd/ailang/design_quorum.go` calls `RunSeatedQuorum` and already maps `SynthBlocked` to exit 3. `RunSeatedQuorum` reruns `synthesize` after reserve/benched attempts, so the final reviewer set supplies the minimum. Its existing `presentCount(q)` counts external reviewers already. A present same-vendor fallback remains external signal under current seating policy; this change does not redefine independence or vendor eligibility.

`SynthesizeWithTier2` starts with `synthesize(tier1, controller)` and only adds blocking rejects/absence/cost. Preserve this monotonic behavior: a Tier-2 pass cannot rescue a zero-signal Tier-1 result. At least one Tier-1 reviewer is a necessary condition, not a promise that every external pass is sufficient. Tier-2 rejects continue to block even when Tier-1 has signal.

### Files to Modify/Create (future implementation only)

- `internal/mission/quorum/quorum.go` — reviewer-only guard count and contract comments, approximately 10 changed lines.
- `internal/mission/quorum/quorum_test.go` — deterministic controller/signal matrix and metadata assertions, approximately 80 added lines.
- `internal/mission/quorum/seating_test.go` — all primary/reserve/benched absent with controller pass, approximately 25 added lines.
- `internal/mission/quorum/tier2_test.go` — zero-signal Tier-1 cannot be rescued by Tier-2 pass, approximately 25 added lines.

## Examples and Exact Test Design

Use existing `present` and `fakeRunner` helpers; all runners are in-process stubs. No credentials or provider calls are required.

| External outcomes | Controller | Expected verdict | Required objection assertions |
|---|---|---|---|
| two absent (auth, budget) | nil | blocked | zero-signal objection |
| two absent (auth, budget) | pass | blocked (currently proceeds) | zero-signal objection |
| two absent (auth, budget) | reject, note `controller veto` | blocked | controller note plus zero-signal objection |
| empty slice | nil / pass / reject | blocked | zero-signal objection; reject note when applicable |
| one pass, one absent | nil / pass | proceed | no blocking objections |
| one pass, one absent | reject | blocked | controller note; no zero-signal objection |
| one reject, one absent | nil / pass | blocked | reviewer objection; no zero-signal objection |

For the absent-pair/controller-pass row, assert both model/reason pairs retained in input order, controller verdict/note retained, and totals equal the supplied costs and input/output token sums (including absent outcomes). This prevents solving the gate by deleting outcomes or their accounting. Assert `AbsentReviewers` remains a non-nil empty slice for all-present inputs through the existing serialization test.

Add a seated orchestration regression with primary `a`, reserve `b`, benched `z`, all absent, controller pass. Assert `blocked`, all three absences named, and fallback attempted; this catches an incorrect early-return fix that bypasses replacement or synthesizes before final outcomes.

Add Tier-2 regression using all-absent Tier-1, controller pass, and a present passing Tier-2 outcome. Assert `blocked` and zero-signal objection retained. Add a positive control with one Tier-1 pass plus one absent and Tier-2 pass: `proceed`. Existing Tier-2 reject tests preserve veto behavior.

Mutation evidence required during execution: restoring only the controller count increment must make the new controller-pass/all-absent and empty-slice/pass cases fail. Restoring the fixed source must return those tests to green. Record actual commands/results in the implementation report; no mutation was performed during candidate authoring.

## Verification Log

All source claims below were independently read in this worktree at the pinned HEAD, which equals origin/dev.

| ID | Claim | Evidence / result |
|---|---|---|
| V1 | Controller defeats external zero-signal guard | `quorum.go` reviewer increment at 156; controller increment at 165; guard at 176. Controller pass does not set blocked, so all-absent + pass reaches proceed. Static verification only. |
| V2 | Existing all-absent regressions omit controller | Read bodies of `TestRunQuorum_AllAbsentRefusesToProceed` and `TestRunSeatedQuorum_NothingAnswersBlocks`; both supply nil. |
| V3 | Central synthesis covers orchestration paths | Read `RunQuorum`, `RunSeatedQuorum` final synthesis and `RunQuorumWithEscalation`; grep of synthesis callers also finds tests and `SynthesizeWithTier2`. |
| V4 | Seating already counts external reviewers | Read `seating.go:presentCount`, reserve loop and benched recall; controller is absent from this count. |
| V5 | Tier-2 only adds vetoes; cannot rescue block | Read `tier2.go:SynthesizeWithTier2`; starts with base synthesis, only assigns `SynthBlocked`, never `SynthProceed`. |
| V6 | CLI maps blocked to exit 3 | Read `cmd/ailang/design_quorum.go`, seated invocation and final exit branch. Empty reviewer CSV is exit 2 before running quorum. |
| V7 | Prior design overlap checked | `rg --files design_docs` filtered for quorum; `rg -n 'zero.signal|zero-signal|controller.only' design_docs/planned design_docs/implemented`; related docs below, no dedicated guard design found in this search. |
| V8 | Prior fleet attempt is parked, not implemented | Read full `fleet-mission-index.md` through iteration 27; iteration 23 entry and archived status describe capacity park. No claim about unindexed external work. |
| V9 | Baseline package verification | `go test ./internal/mission/quorum` exited 0: `ok github.com/sunholo-data/ailang/internal/mission/quorum 0.767s`. Does not test the new matrix or prove the proposed fix. |

Scaffolding/related-search deviation: author used the skill structure and local `rg`/source reads. The create script invokes neural searches; it was not run because this task prohibits metered quota use. No language-support claim, new diagnostic code, banked schema change or compiler pass is proposed.

## Success Criteria and Verification

- [ ] Entire matrix above passes, and new controller-only tests fail before the fix.
- [ ] Seated replacement remains functional and all absences remain named.
- [ ] Controller/reviewer/Tier-2 rejects retain vetoes; partial-present pass stays allowed.
- [ ] Existing serialization, accounting and escalation tests pass.
- [ ] `go test ./internal/mission/quorum` and `go test -race ./internal/mission/quorum` pass; run repository-required checks from the approved sprint.
- [ ] Contract comments and implementation report describe the intentional controller-only behavior change.

## Axiom Compliance

Canonical reference: [Design Axioms](/docs/references/axioms).

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | Same deterministic traversal and composition |
| A2 Replayability | 0 | Artifact layout retained |
| A3 Effect Legibility | 0 | Pure synthesis remains pure |
| A4 Explicit Authority | +1 | Controller cannot impersonate external evidence |
| A5 Bounded Verification | +1 | Small hermetic matrix proves gate boundary |
| A6 Safe Concurrency | 0 | Runner scheduling unchanged |
| A7 Machines First | +1 | Machine verdict blocks absent reviewer signal |
| A8 Minimal Syntax | 0 | No language syntax changes |
| A9 Cost Visibility | 0 | Existing accounting preserved |
| A10 Composability | 0 | Existing orchestration reused |
| A11 Structured Failure | +1 | Existing blocked verdict and explicit objection |
| A12 System Boundary | 0 | No provider boundary changes |

Net +4; A1/A3/A4/A7 have no hard violations. This score supports the candidate, not authorization to implement.

## Risks, Deferred Decisions and Non-Goals

Controller-only runs intentionally change from proceed to blocked. Resume by obtaining an admitted present reviewer, rather than weakening the guard. Same-vendor fallback remains governed by seating; this ticket does not strengthen vendor policy. Keep Tier-2's conservative monotonic veto behavior and cover it explicitly. Malformed/nil runner outcomes and verdict validation are separate concerns; this design assumes the existing ReviewerOutcome contract and does not add repair fallbacks.

The implementer may choose table-driven versus subtest organization and local variable spelling. Scope excludes reviewer transport, quota routing, salvage retries, artifact directories, schema changes and historical artifact rewriting. Two-hour implementation estimate is provisional for the planner; candidate stage makes no speedup, production-frequency or completion claim.

## Related Documents

- [Fleet charter](../fleet-mission.md) — P1 #8 and iteration 23 capacity park.
- [Fleet iteration index](../fleet-mission-index.md) — prior work check.
- [Agentic quorum design](../implemented/v0_30_0/m-mission-quorum-agentic-verify.md) — additive Tier-2 composition.
- [Quorum salvage retry](m-quorum-salvage-retry.md) — separate parse/absence concern.

## Review State

Candidate only, revision 1. Independent design quorum: NOT RUN / pending capacity. Independent evaluator: unavailable per controller report. Sprint plan: not created. Implementation, commit, push and ticket resolution: not performed. Resume with admitted independent review, then the normal design/plan/execute/evaluate gates.

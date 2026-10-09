# Quorum External Zero-Signal Guard

**Status**: Planned — candidate, revision 2; quorum round 1 returned BLOCKED, revision pending re-review
**Target**: Next patch after approval (unversioned)
**Priority**: P1 — fleet charter #8, ailang#651
**Estimated**: 2 hours including regression verification (estimate, not measured)
**Dependencies**: Admitted independent review and normal planning/execution gates
**Planner-Lane**: codex-ok
**Created / revised**: 2026-10-08, fleet iteration 28, revision 1; revised 2026-10-09, revision 2 (pi, designer role — answers quorum round 1)
**Author**: native codex:gpt-6.1-sol (role tokens: not reported); revision 2 by pi (designer role, fleet-i30)

## Problem Statement

At origin/dev `3e032cf4f13682a455129274d99097842c56e5ae` (re-confirmed at the revision-2 pin `1e872b0000`: controller increment `quorum.go:165`, guard `:176`), `synthesize` counts present external reviewer outcomes, then increments the same count for a non-nil controller. Its zero-signal guard therefore permits all external reviewers to be absent when the controller passes. The result is `proceed`, although nobody outside the controller produced a verdict. This is a source-verified mechanism; this iteration has not run a new live provider quorum or counted production occurrences.

The existing `TestRunQuorum_AllAbsentRefusesToProceed` and `TestRunSeatedQuorum_NothingAnswersBlocks` pass a nil controller, and a repo-wide survey of every non-nil `ControllerReview` use in test files (V10) shows the controller-pass-with-zero-present-reviewers combination is uncovered suite-wide: no existing test pairs a non-nil controller with zero present external reviewers while expecting proceed, so the intended flip breaks no existing fixture. Fleet iteration 23 already verified the ticket and parked it on capacity; the index through iteration 27 records no implementation for this ticket. This candidate advances design only.

## Goals

Require at least one present external Tier-1 reviewer outcome before synthesis can proceed. A controller remains a participant with veto authority and a distinct artifact entry, but cannot supply the external-signal minimum. Keep N-1 degradation: one present passing reviewer and any number of named absences may proceed, provided neither controller nor another present reviewer rejects.

## High-Impact Decisions

| Decision | Why it matters | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Exclude controller from external-signal count | Prevents controller-only clearance | agent-chosen; independently quorum-reviewed (round 1 blocked — this revision answers it; re-review pending) | design | low |
| Preserve one-reviewer minimum and existing seating | Avoids introducing a full-seat or vendor-policy requirement | agent-chosen; independently quorum-reviewed (round 1 blocked — this revision answers it; re-review pending) | design | low |
| Keep Tier-2 additive, unable to unblock failed Tier-1 | Matches existing escalation composition | agent-chosen; independently quorum-reviewed (round 1 blocked — this revision answers it; re-review pending) | design | low |

### Design Freeze

This document has no outstanding human policy choice. It is not approval or execution authorization. Quorum review and the normal plan/execute gates are still pending: round 1 of the independent design quorum ran and returned BLOCKED (two blocking objections plus one pass-with-concern), and this revision 2 answers all three but is not itself re-reviewed. Round 1 was previously delayed on capacity (native Sonnet/Opus unsupported, Anthropic/OpenRouter over ration); the capacity facts, the round-1 verdict record and the verbatim objections are controller-supplied, not independently measured by this author, and no round-1 artifacts exist in this worktree (`.ailang/state/mission-quorum/` is absent here). The answering measurements (V10–V13) are this author's, run in this worktree.

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

For the absent-pair/controller-pass row, assert both model/reason pairs retained in input order, controller verdict/note retained, and totals equal the supplied costs and input/output token sums (including absent outcomes). This prevents solving the gate by deleting outcomes or their accounting. For the two-absent/controller-reject row, assert BOTH objections in deterministic order — controller note first, then the zero-signal objection — which holds by construction: `synthesize` has no early return before the external-count guard (V13). Assert `AbsentReviewers` remains a non-nil empty slice for all-present inputs through the existing serialization test.

Add a seated orchestration regression with primary `a`, reserve `b`, benched `z`, all absent, controller pass. Assert `blocked`, all three absences named, and fallback attempted; this catches an incorrect early-return fix that bypasses replacement or synthesizes before final outcomes.

Add Tier-2 regression using all-absent Tier-1, controller pass, and a present passing Tier-2 outcome. Assert `blocked` and zero-signal objection retained. Add a positive control with one Tier-1 pass plus one absent and Tier-2 pass: `proceed`. Existing Tier-2 reject tests preserve veto behavior.

Mutation evidence required during execution: restoring only the controller count increment must make the new controller-pass/all-absent and empty-slice/pass cases fail. Restoring the fixed source must return those tests to green. Record actual commands/results in the implementation report; no mutation was performed during candidate authoring.

## Verification Log

All source claims below were read in this worktree. Revision-1 rows V1–V9 were read at the then-pinned origin/dev HEAD `3e032cf4f13682a455129274d99097842c56e5ae`; revision-2 rows V10–V13 were run at this worktree's pinned HEAD `1e872b0000`, where V1's load-bearing positions (reviewer increment :156, controller increment :165, guard :176) were re-confirmed unchanged. One excerpt is controller-supplied: the GitHub-issue text inside V11, quoted from the controller's read at 2026-10-09 because this worktree has no network access. Every negative or absence claim is paired with a known-positive control in the same row.

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
| V10 | Suite-wide survey: NO existing test pairs a non-nil controller with ZERO present external reviewers while expecting proceed, so the intended controller-only flip (proceed → blocked) breaks no existing fixture (answers round-1 blocking objection oc-glm-5.3; extends V2 from two sample bodies to the whole suite) | Command: `grep -rn "ControllerReview" --include="*_test.go" .` → exactly 4 hits: `escalate_test.go:9` (helper signature `func qr(reviewers []*ReviewerOutcome, controller *ControllerReview) *QuorumResult`), `escalate_test.go:84` (`TestShouldEscalate_ControllerSplit`: ONE PRESENT reviewer m1 + controller reject → expects escalation; present external signal exists), `quorum_test.go:66` (`TestRunQuorum_ControllerRejectBlocks`: ONE PRESENT reviewer m1 + controller reject → expects blocked), `seating_test.go:118` (`TestRunSeatedQuorum_RecallsTheAuthorsVendorOnlyWhenNobodyAnswers`: primary a,b and reserve c absent, benched z recalled AND present as author-vendor fallback + controller pass; asserts the z seat label and 3 named absences, never asserts a verdict). Known-positive control in the same survey: `grep -rn "ControllerReview{" --include="*_test.go" internal/mission/quorum/` → 3 hits (the same three literals), proving the survey grep sees present fixtures so the absence elsewhere is real, not a path/spelling artifact. |
| V11 | Mandate premise: charter #8 and ailang#651 state the exact ticket semantics the fix implements (answers round-1 blocking objection oc-kimi-k3) | Charter `design_docs/fleet-mission.md`, P1 queue row #8 (line 192), re-derived by this author from the worktree file, verbatim: "`quorum:zero-signal-guard-vacuous-with-controller-verdict`: `quorum.go:164-165` counts the controller as present before the zero-signal guard (:176). A controller-only "quorum" proceeds (ailang#651)." GitHub issue sunholo-data/ailang#651, titled "design-quorum: the zero-signal guard is vacuous whenever a controller verdict is supplied — 3 docs shipped on 'proceed' with 0 of 2 reviewers present", summary: the guard "can never fire on the invocation the mission-control skill mandates, because the controller's own verdict increments the same presentCount it tests"; measured 86 of 87 artifacts carrying a controller verdict — issue text quoted from the controller's read at 2026-10-09 (controller-supplied; no network access in this worktree). Worktree corroboration: `.claude/skills/mission-control/resources/gate-2-pick.md:234-238` independently states "the controller's own `--controller-verdict` increments the same `presentCount` it tests" and "measured: **86 of 87** artifacts carry one; control — the single artifact without one reads `blocked`". Line positions re-confirmed in this tree: controller increment `quorum.go:165`, guard `:176` (the skill file's older `:168` reference is stale). |
| V12 | Mandate semantics: the mandated remedy is a present-EXTERNAL count; the reviewer-named alternatives are NOT mandated (answers round-1 blocking objection oc-kimi-k3) | `.claude/skills/mission-control/resources/gate-2-pick.md` (Gate-2 fleet branch), lines 238–239, verbatim: "Filed as `#651` for the code fix; until it lands, the count that matters is **present EXTERNAL reviewers**, and zero of them is a park, not a pass." This matches the chosen remedy exactly — the guard's count excludes the controller and tests present external reviewers only, with zero of them a blocked verdict (park), not a pass. The alternatives the reviewer named (full-seat requirement, 2+ present reviewers, removing the controller from the artifact entirely) appear in no mandate source (charter row #8, issue #651 as quoted in V11, gate-2-pick.md) and are explicitly out of scope; controller veto, controller artifact entry, N-1 degradation and Tier-2 monotonicity are retained. |
| V13 | `synthesize` emits BOTH objections for the two-absent/controller-reject matrix row; no early return precedes the external-count guard, so the matrix row's dual-objection expectation is correct by construction (answers round-1 pass-with-concern gemini-3-1-pro) | Command: `grep -n "return" internal/mission/quorum/quorum.go` → `88: return &QuorumResult{`, `124: return q`, `129: return q`, `182: return s`. Exactly ONE return (:182, the function end) lies inside `synthesize` (`func` at :141); :88 is `RunQuorum`, :124/:129 are `RunQuorumWithEscalation`. The reviewer-reject branch (:157), controller-reject branch (:166) and zero-signal guard (:176) each append to `s.BlockingObjections` (:159, :168, :178) WITHOUT early return, so under the proposed design both the controller note and the zero-signal objection are emitted, in that order. |

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

Candidate, revision 2. Independent design quorum, round 1: RUN — verdict BLOCKED, with two blocking objections (oc-glm-5.3: the claim that existing all-absent regressions omit the controller was evidenced only at two sample test bodies, not suite-wide; oc-kimi-k3: the load-bearing policy premise — "fleet charter #8, ailang#651" — was never quoted, and the doc's authority wording was self-contradictory, freezing "no outstanding human policy choice" while marking the decisions merely "subject to review") and one pass-with-concern (gemini-3-1-pro: a synthesis early-return on a controller veto might emit only one of the two objections expected for the two-absent/controller-reject matrix row). The round-1 verdict record and verbatim objections are controller-supplied (no round-1 artifacts exist in this worktree — `.ailang/state/mission-quorum/` is absent here); the answering measurements are this author's, run in this worktree at HEAD `1e872b0000`. Independent evaluator: unavailable per controller report.

Revision 2 answers all three without changing the fix shape — the guard still counts present EXTERNAL reviewer outcomes only; the controller keeps veto and a distinct artifact entry; N-1 degradation is preserved; Tier-2 stays monotonic; the existing zero-signal objection text is retained. V10 answers oc-glm-5.3 with a suite-wide survey and known-positive control. V11 and V12, plus the Design Freeze and High-Impact Decisions wording corrections, answer oc-kimi-k3: the mandate sources are now quoted verbatim, "Chosen By" states the decisions are agent-chosen AND independently quorum-reviewed (round 1 blocked — this revision answers it; re-review pending), and the Freeze states plainly that quorum review and the normal plan/execute gates are still pending. V13 answers gemini-3-1-pro: `synthesize` has exactly one return, at the function end, so both objections are emitted for the contested matrix row.

Pending: quorum re-review of this revision. Sprint plan: not created. Implementation, commit, push and ticket resolution: not performed. Resume with admitted independent re-review, then the normal design/plan/execute/evaluate gates.

### Implementation report (revision 2)

Exactly one file changed — this document. Verification Log rows added (V10–V13); no existing evidence row altered. Wording-only changes: header Status / Created-revised / Author lines; Problem Statement now cites the V10 suite-wide survey and re-confirms the defect positions at the revision-2 pin; High-Impact Decisions "Chosen By" corrected from "agent, subject to review" to agent-chosen AND independently quorum-reviewed; Design Freeze reconciled (round 1 ran and blocked; re-review and plan/execute gates still pending); Verification Log preamble now pins both HEADs and marks the single controller-supplied excerpt; one V13-backed ordering sentence added to the matrix section; this Review State rewritten. Test matrix, mutation-evidence requirement, axioms table and Success Criteria are otherwise unchanged. No git write operations were performed; the controller commits this work.

Rows added, with each command's observed output:

- **V10** (answers blocking objection 1, oc-glm-5.3). Command: `grep -rn "ControllerReview" --include="*_test.go" .` Observed — exactly 4 hits:

  ```
  ./internal/mission/quorum/seating_test.go:118:	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "the controller is Anthropic too"}
  ./internal/mission/quorum/escalate_test.go:9:func qr(reviewers []*ReviewerOutcome, controller *ControllerReview) *QuorumResult {
  ./internal/mission/quorum/escalate_test.go:84:	}, &ControllerReview{Verdict: VerdictReject, Note: "an axiom conflict I see in session"})
  ./internal/mission/quorum/quorum_test.go:66:	ctrl := &ControllerReview{Verdict: VerdictReject, Note: "controller sees an axiom conflict"}
  ```

  Known-positive control in the same survey: `grep -rn "ControllerReview{" --include="*_test.go" internal/mission/quorum/`. Observed — 3 hits:

  ```
  internal/mission/quorum/seating_test.go:118:	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "the controller is Anthropic too"}
  internal/mission/quorum/escalate_test.go:84:	}, &ControllerReview{Verdict: VerdictReject, Note: "an axiom conflict I see in session"})
  internal/mission/quorum/quorum_test.go:66:	ctrl := &ControllerReview{Verdict: VerdictReject, Note: "controller sees an axiom conflict"}
  ```

  Reading the four sites: escalate_test.go:84 (`TestShouldEscalate_ControllerSplit`) and quorum_test.go:66 (`TestRunQuorum_ControllerRejectBlocks`) each pair the controller with ONE PRESENT reviewer m1, expecting escalation and blocked respectively; seating_test.go:118 (`TestRunSeatedQuorum_RecallsTheAuthorsVendorOnlyWhenNobodyAnswers`) pairs a controller pass with benched z recalled AND present plus 3 named absences (a, b, c), asserts the z seat label and never asserts a verdict; escalate_test.go:9 is a helper signature. Conclusion: NO existing test pairs a non-nil controller with ZERO present external reviewers while expecting proceed — the intended controller-only flip breaks no existing fixture.

- **V11** (answers blocking objection 2 part a, oc-kimi-k3). Charter quote re-derived by this author from the worktree file `design_docs/fleet-mission.md` (P1 queue row #8, line 192), verbatim: "`quorum:zero-signal-guard-vacuous-with-controller-verdict`: `quorum.go:164-165` counts the controller as present before the zero-signal guard (:176). A controller-only "quorum" proceeds (ailang#651)." GitHub issue sunholo-data/ailang#651, titled "design-quorum: the zero-signal guard is vacuous whenever a controller verdict is supplied — 3 docs shipped on 'proceed' with 0 of 2 reviewers present" (summary: the guard "can never fire on the invocation the mission-control skill mandates, because the controller's own verdict increments the same presentCount it tests"; measured 86 of 87 artifacts carrying a controller verdict) — recorded as quoted from the controller's read at 2026-10-09; this worktree has no network access. Worktree corroboration: `gate-2-pick.md:234-238` independently states the same mechanism ("the controller's own `--controller-verdict` increments the same `presentCount` it tests") and the same measurement ("86 of 87 artifacts carry one; control — the single artifact without one reads `blocked`").

- **V12** (answers blocking objection 2 part c, oc-kimi-k3). `gate-2-pick.md:238-239`, verbatim: "Filed as `#651` for the code fix; until it lands, the count that matters is **present EXTERNAL reviewers**, and zero of them is a park, not a pass." Matches the chosen remedy — the controller is excluded from the external-signal minimum; zero present externals is a blocked verdict, not a pass. The reviewer-named alternatives (full-seat requirement, 2+ present reviewers, removing the controller from the artifact entirely) appear in no mandate source and are explicitly out of scope.

- **V13** (answers the pass-with-concern, gemini-3-1-pro). Command: `grep -n "return" internal/mission/quorum/quorum.go`. Observed:

  ```
  88:	return &QuorumResult{
  124:		return q
  129:		return q
  182:	return s
  ```

  Exactly one return (:182, the function end) lies inside `synthesize` (func at :141); the reviewer-reject branch (:157), controller-reject branch (:166) and zero-signal guard (:176) all append to `s.BlockingObjections` (:159, :168, :178) without early return, so under the proposed design both the controller note and the zero-signal objection are emitted for the two-absent/controller-reject matrix row — the dual-objection expectation is correct by construction.

# Sprint Plan — M-QUORUM-ZERO-SIGNAL: Quorum external zero-signal guard

**Sprint ID**: M-QUORUM-ZERO-SIGNAL
**Design doc**: [m-quorum-external-zero-signal-guard.md](m-quorum-external-zero-signal-guard.md) (revision 3; quorum rounds 1–2 BLOCKED, round-2 verbatim fixes applied under the ratified narrow-refinement carve-out)
**Fleet ticket**: `quorum:zero-signal-guard-vacuous-with-controller-verdict` (charter P1 queue row #8), reported at ailang#651
**Scope (fleet scope guard)**: `internal/mission/quorum/**` ONLY. Anything outside is refused by the fleet pre-push guard and by this plan. The one external consumer (`cmd/ailang/design_quorum.go:89`) maps `SynthBlocked` to exit 3 already and needs NO companion change (design doc V14).
**Lane note**: Go-only loop-harness change; no `.ail` code is written, so the AILANG syntax gate does not apply (`ailang prompt` not run). Deterministic in-process stub runners only (existing `present`/`fakeRunner` helpers); no credentials, no provider calls.
**Planner re-verification base**: worktree HEAD `8d0d3de0e` (origin/dev `0a2202ad6` + two design-doc-only revision commits; `git diff 0a2202ad6..HEAD -- internal/ cmd/` is empty — see P10), tree clean except this untracked plan file. The doc's V-rows were measured on older bases (`3e032cf4…`, `1e872b0000`). This plan was first drafted by an earlier, unrecorded fire; the iteration-32 planner re-ran every Verification row at `8d0d3de0e` and replaced each observed output with its own reading, and corrected the helper/seam citations for the seated and Tier-2 regressions (P11–P12).

## Goal and motivation

The fleet's mandated design-quorum invocation always supplies an in-session controller verdict, and today `synthesize` in `internal/mission/quorum/quorum.go` counts the controller as a *present participant* (`presentCount++` at :165) before the zero-signal guard (`if presentCount == 0` at :176) — so a run with every external reviewer absent still reaches `proceed` whenever the controller passes, making the guard vacuous on precisely the invocation the mission-control skill mandates (fleet ticket `quorum:zero-signal-guard-vacuous-with-controller-verdict`, reported at ailang#651, where 86 of 87 sampled artifacts carried a controller verdict). This sprint makes the guard test present EXTERNAL reviewers only: the count is renamed `externalPresentCount`, the controller increment is removed, and the existing guard and its objection text — `no reviewer produced a verdict (all absent) — refusing to proceed on zero signal` — are retained, so a controller-only "quorum" blocks while the controller keeps its veto, its distinct artifact entry, N-1 degradation (one present passing reviewer plus any named absences may proceed), and Tier-2's monotonic composition (a Tier-2 pass can never rescue a zero-signal Tier-1 result).

## Technical approach

From the design doc's Solution Design (unchanged in intent; production diff ≈10 lines, all in `internal/mission/quorum/quorum.go`):

1. **Rename the count and exclude the controller.** In `synthesize`, name the local count `externalPresentCount` and increment it only for present outcomes in the reviewer loop (:148–:162). Remove the controller increment (:165) from the `if controller != nil` block — the block keeps its controller-reject veto and objection (`controller (in-session): <note>`) untouched.
2. **Retain the guard verbatim.** The existing zero-signal guard stays at its position, now testing the external count, with its objection text exactly: `no reviewer produced a verdict (all absent) — refusing to proceed on zero signal`. Apply it to the external count; an empty outcome slice with any controller (nil, pass, or reject) must also block — this covers the public Go API even though the CLI rejects an empty reviewers argument.
3. **No early return before the guard.** `synthesize` has exactly one return, at the function end (:182); reviewer-reject (:157), controller-reject (:166) and the guard (:176) all append to `BlockingObjections` in one pass (V13). The fix introduces NO early return — the two-absent/controller-reject row must emit both objections, controller note first, by construction.
4. **Keep the inline count (registry decision).** Do NOT extract or reuse `seating.go:presentCount(q *QuorumResult)` (:132): it operates on a built `*QuorumResult` while `synthesize` runs on `[]*ReviewerOutcome` before any result exists; the outcomes loop already checks `VerdictReject`, so the increment rides in the same pass. Reuse would be circular or a second pass (V16).
5. **Tier-2 stays monotonic.** `SynthesizeWithTier2` starts with `synthesize(tier1, controller)` and only ever assigns `SynthBlocked`, never `SynthProceed`. No Tier-2 production change; add the regression proving a Tier-2 pass cannot rescue a zero-signal Tier-1 result.
6. **Accounting unchanged.** Absence records (`AbsentReviewers` non-nil empty for all-present inputs, absent pairs in input order), controller recording in the artifact, billing/token totals (including absent outcomes), and JSON fields are untouched — the gate cannot be satisfied by deleting outcomes or their accounting.
7. **Comments.** Update the `synthesize` and `RunQuorum` contract comments to state the external minimum explicitly ("at least one present external reviewer outcome").

## Milestones

This is a SMALL sprint (~10 changed production lines + ~130 added test lines ≈ 140 LOC total), so it is ONE milestone; no second milestone is warranted under the ~200 LOC ceiling.

### M1: Guard the zero-signal count on present external reviewers only (M1_GUARD_EXTERNAL_COUNT) (~140 LOC)

**Dependencies:** none (design routed after revision 3; no other sprint).

Ordered steps (the pre-fix red run is step 1–3 by design):

1. **Bounding control, pre-fix half.** Confirm the full package suite is green BEFORE any edit: `go test ./internal/mission/quorum -count=1`. (Planner already recorded exit 0 at `8d0d3de0e` — see Verification P4; executor re-confirms and records.)
2. **Write the new tests against the unfixed source** (suggested names; the implementer may choose table-driven versus subtest organization and local spelling):
   - `internal/mission/quorum/quorum_test.go` — `TestSynthesize_ExternalSignalMatrix` (or equivalent), driving `RunQuorum` with `fakeRunner` tables over the full controller × signal matrix (Test Matrix below), including the accounting/order assertions and the empty-slice rows (empty `reviewerModels` exercises the public Go API directly).
     NOTE: the existing absent fixtures (e.g. `TestRunQuorum_AllAbsentRefusesToProceed`) carry zero cost/tokens and `present()` sets only `CostUSD: 0.01` (no tokens). For the absent-pair/controller-pass accounting assertion to be non-vacuous, give the absent outcomes distinct NON-ZERO `CostUSD`, `TokensIn` and `TokensOut` and assert the totals equal their sums.
   - `internal/mission/quorum/seating_test.go` — seated orchestration regression: primary `a`, reserve `b`, benched `z`, ALL absent, controller pass. Use the file's existing `poolRunner` (`seating_test.go:88`, records `calls`; everyone not in `up` is absent) and `seat` (`:80`, absent reason always `ReasonBudget`) helpers — NOT `fakeRunner`, which does not record calls. Call `RunSeatedQuorum("doc", "body", "ts", Selection{Primary: []string{"a"}, Reserve: []string{"b"}, Benched: []string{"z"}}, seats, cap, ctrl, f.run)`; "fallback attempted" = `f.calls` contains `b` and `z`; "three absences named" = `q.Synthesis.AbsentReviewers` lists `a`, `b`, `z`.
   - `internal/mission/quorum/tier2_test.go` — Tier-2 regression: all-absent Tier-1 + controller pass + present passing Tier-2, plus the positive control. Seam: call `SynthesizeWithTier2(tier1, ctrl, &Tier2Result{Reviewers: []*ReviewerOutcome{present("t2", VerdictPass, "…")}})` directly, following `TestSynthesizeWithTier2_NilIsTier1Identical` (`tier2_test.go:131`). Do NOT drive it via `RunQuorumWithEscalation`: `ShouldEscalate` (`escalate.go:76`) only escalates on a present premise-class reject, a high-stakes doc, or a split, so an all-absent Tier-1 with a passing controller never reaches Tier-2 through that path and the regression would be vacuous.
3. **Pre-fix RED run (required evidence).** Run the new controller-pass/all-absent and empty-slice/pass cases against the unfixed source; they MUST FAIL (currently they would `proceed`). The new seated (all absent + controller pass) and Tier-2 zero-signal regressions are also controller-only cases and are likewise expected RED pre-fix; the nil-controller, reject and one-present rows are expected green both before and after. Record command and output in the implementation report — this is the "new controller-only tests fail before the fix" half of Success Criterion 1.
4. **Apply the production change** per Technical approach 1–2 and 7 (≈10 lines in `quorum.go`).
5. **Bounding control, post-fix half.** `go test ./internal/mission/quorum -count=1` green, then `go test -race ./internal/mission/quorum -count=1` green. The pre/post green pair is the bounding control for any fixture the V10/V15 textual surveys could miss.
6. **Mutation drill MU1** (Mutation Matrix below); record actual commands and results.
7. **Implementation report** describing the intentional controller-only behavior change (proceed → blocked when no external reviewer is present) and carrying the red-run and mutation evidence.

#### Acceptance criteria (M1) — from the design doc, Success Criteria and Exact Test Design sections, verbatim

- [ ] Entire matrix above passes, and new controller-only tests fail before the fix. The full package suite (`go test ./internal/mission/quorum`) run BEFORE the fix stays green immediately AFTER the fix — that pre/post pair is the bounding control for any fixture the V10/V15 textual surveys could miss (oc-glm-5.3 round-2 concern, applied verbatim).
- [ ] Seated replacement remains functional and all absences remain named.
- [ ] Controller/reviewer/Tier-2 rejects retain vetoes; partial-present pass stays allowed.
- [ ] Existing serialization, accounting and escalation tests pass.
- [ ] `go test ./internal/mission/quorum` and `go test -race ./internal/mission/quorum` pass; run repository-required checks from the approved sprint.
- [ ] Contract comments and implementation report describe the intentional controller-only behavior change.
- [ ] For the absent-pair/controller-pass row, assert both model/reason pairs retained in input order, controller verdict/note retained, and totals equal the supplied costs and input/output token sums (including absent outcomes).
- [ ] For the two-absent/controller-reject row, assert BOTH objections in deterministic order — controller note first, then the zero-signal objection — which holds by construction: `synthesize` has no early return before the external-count guard (V13).
- [ ] Assert `AbsentReviewers` remains a non-nil empty slice for all-present inputs through the existing serialization test.
- [ ] Add a seated orchestration regression with primary `a`, reserve `b`, benched `z`, all absent, controller pass. Assert `blocked`, all three absences named, and fallback attempted; this catches an incorrect early-return fix that bypasses replacement or synthesizes before final outcomes.
- [ ] Add Tier-2 regression using all-absent Tier-1, controller pass, and a present passing Tier-2 outcome. Assert `blocked` and zero-signal objection retained. Add a positive control with one Tier-1 pass plus one absent and Tier-2 pass: `proceed`. Existing Tier-2 reject tests preserve veto behavior.
- [ ] Mutation evidence required during execution: restoring only the controller count increment must make the new controller-pass/all-absent and empty-slice/pass cases fail. Restoring the fixed source must return those tests to green. Record actual commands/results in the implementation report.

## Estimated LOC

| File | Change | Est. LOC |
|---|---|---|
| `internal/mission/quorum/quorum.go` | rename count to `externalPresentCount`, delete controller increment, contract-comment updates | ~10 changed |
| `internal/mission/quorum/quorum_test.go` | deterministic controller/signal matrix + metadata/accounting assertions | ~80 added |
| `internal/mission/quorum/seating_test.go` | all primary/reserve/benched absent + controller pass | ~25 added |
| `internal/mission/quorum/tier2_test.go` | zero-signal Tier-1 not rescued by Tier-2 pass + positive control | ~25 added |
| **Total** | | **~140 (~10 changed + ~130 added)** |

Velocity: ~140 LOC in an estimated 2 hours including regression verification (design-doc estimate, not measured) ≈ 0.25 day ⇒ ~560 LOC/day.

## Test matrix (quoted verbatim from the design doc's Exact Test Design)

| External outcomes | Controller | Expected verdict | Required objection assertions |
|---|---|---|---|
| two absent (auth, budget) | nil | blocked | zero-signal objection |
| two absent (auth, budget) | pass | blocked (currently proceeds) | zero-signal objection |
| two absent (auth, budget) | reject, note `controller veto` | blocked | controller note plus zero-signal objection |
| empty slice | nil / pass / reject | blocked | zero-signal objection; reject note when applicable |
| one pass, one absent | nil / pass | proceed | no blocking objections |
| one pass, one absent | reject | blocked | controller note; no zero-signal objection |
| one reject, one absent | nil / pass | blocked | reviewer objection; no zero-signal objection |

Supporting doc requirements carried into the matrix implementation: absent-reason constants `ReasonAuth` ("auth") / `ReasonBudget` ("budget") as in the existing `TestRunQuorum_AllAbsentRefusesToProceed` fixture; absent pairs asserted in input order with per-outcome costs/tokens summed into totals; `AbsentReviewers` non-nil empty for all-present inputs via the existing `TestSynthesis_AbsentReviewersMarshalsAsEmptySlice`; seated row asserts three named absences and that fallback was attempted; Tier-2 rows assert the zero-signal objection survives a passing Tier-2 and the one-present-pass positive control still proceeds. Existing `TestRunQuorum_AllAbsentRefusesToProceed` (nil controller) and `TestRunSeatedQuorum_NothingAnswersBlocks` remain unchanged and green.

## Mutation matrix

| ID | Mutation (scratch edit; never committed) | Expected RED cases | Restore check | Command | Evidence to record |
|---|---|---|---|---|---|
| MU1 | Restore ONLY the controller-count increment: re-insert `externalPresentCount++` as the first statement of the `if controller != nil {` block in `synthesize` (the exact line this sprint removes) | New **controller-pass/all-absent** matrix row AND **empty-slice/controller-pass** row fail (both flip back toward `proceed`) | Restore the fixed source (remove the re-inserted line); both new cases return green and the full suite stays green | `go test ./internal/mission/quorum -count=1 -run 'TestSynthesize_ExternalSignalMatrix'` (mutated, expect FAIL on the two cases; log case names) then same command after restore (expect PASS), plus full-suite `go test ./internal/mission/quorum -count=1` | Actual commands and pass/fail output pasted into the implementation report |

No mutation was performed during candidate authoring; the drill runs during execution only, on the worktree, and is reverted before handoff.

## Registry reuse decision (per milestone)

| Milestone | Candidate | Decision | Rationale |
|---|---|---|---|
| M1 | `seating.go:presentCount(q *QuorumResult)` (:132) | **Do not reuse** — retain the inline same-pass increment in `synthesize` | Different layers: the helper counts `q.Reviewers` on a built `*QuorumResult`; `synthesize` runs on `[]*ReviewerOutcome` before any result exists, and its loop already checks `VerdictReject` on present outcomes. Reuse would require building a result first (circular) or a second pass (design doc V16, answering round-2 blocker gemini-3-1-pro). |
| M1 | Test helpers `present` / `fakeRunner` (`quorum_test.go:22`, `:13`); `seat` / `poolRunner` (`seating_test.go:80`, `:88`) | **Reuse as-is** | Existing deterministic in-process stubs. `fakeRunner` returns absent outcomes (`ReasonUnreachable`, "not in table") for unlisted models, so the `quorum_test.go` matrix needs no new runner. The seated regression uses `poolRunner`, which records `calls` (needed for "fallback attempted"). The Tier-2 regression calls `SynthesizeWithTier2` directly with a hand-built `*Tier2Result` (no agentic runner). No credentials, no provider calls. |

## Risks

- **Intentional behavior change**: controller-only runs flip proceed → blocked. Mitigation: this is the ticketed mandate (charter row #8; gate-2-pick.md: "present EXTERNAL reviewers … zero of them is a park, not a pass"); the single external consumer already maps blocked → exit 3 (V6/V14); resume path is obtaining an admitted present reviewer, not weakening the guard.
- **Fixture surprise beyond the textual survey**: V10/V15 bound it textually; the pre-fix/post-fix full-suite green pair (Success Criterion 1) is the runtime bounding control.
- **Incorrect fix shape**: an early return on controller reject before the guard would drop the zero-signal objection. Caught by the two-absent/controller-reject row's dual-objection ordering assertion (V13 construction: one return at :182) and by the seated regression (an early return that bypasses reserve replacement fails "fallback attempted").
- **Scope creep**: vendor-independence policy, full-seat or 2+-present minimums, and removing the controller from the artifact are explicitly out of scope (no mandate source names them); malformed/nil runner outcomes and salvage retry belong to `m-quorum-salvage-retry.md`, not this sprint.
- **Suite speed/flakiness**: none introduced — all runners are table-driven stubs (`fakeRunner`), deterministic, in-process; `-race` run is part of acceptance.

## Verification (iteration-32 planner re-derivation at worktree HEAD `8d0d3de0e`; every row re-run by this planner, 2026-10-12)

| ID | Claim | Command | Observed output |
|---|---|---|---|
| P0 | Re-verification base | `git rev-parse --short HEAD`; `git status --porcelain` | `8d0d3de0e`; status shows only `?? design_docs/planned/sprint-plan-quorum-zero-signal-guard.md` (this plan, untracked before commit) |
| P1 | `synthesize` counts the controller as present: exactly two `presentCount++` increments | `grep -n "presentCount++" internal/mission/quorum/quorum.go` | Exactly two hits — `156:		presentCount++` (reviewer loop) and `165:		presentCount++` (controller block). P5's body read places both inside `synthesize` (`func` at :141, single `return s` :182). |
| P2 | Zero-signal guard position | `grep -n "if presentCount == 0" internal/mission/quorum/quorum.go` | `176:	if presentCount == 0 {` |
| P3 | Exactly one external consumer of the quorum entry points | `grep -rn "RunQuorum\|RunSeatedQuorum\|RunQuorumWithEscalation\|SynthesizeWithTier2" --include="*.go" cmd/ internal/ \| grep -v internal/mission/quorum/` | Exactly one hit: `cmd/ailang/design_quorum.go:89: result := quorum.RunSeatedQuorum(docPath, docBody, isoTS, sel, *seats, *maxCost, controller, quorumSeatRunner)`. Known-positive control in the same row: `grep -n "RunSeatedQuorum" cmd/ailang/design_quorum.go` → `89:	result := quorum.RunSeatedQuorum(...)` — the survey sees the known consumer, so the single hit is real. |
| P4 | Pre-fix baseline suite green | `time go test ./internal/mission/quorum -count=1` | rc=0 — `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.521s`; wall 5.355 s total (includes compile). This is the pre-fix half of the bounding pair the executor must reproduce. |
| P5 | Guard objection text exact; no early return before the guard; reviewer increment follows accounting | `sed -n '141,185p' internal/mission/quorum/quorum.go` | `presentCount := 0` :146; cost/token accumulation :149–151; `AbsentNote` append :153; reviewer `presentCount++` :156; controller block `if controller != nil {` :164 with `presentCount++` :165 and reject-note append :168; guard comment :173–175; guard :176 appending `"no reviewer produced a verdict (all absent) — refusing to proceed on zero signal"` :178; single `return s` :182. |
| P6 | Charter mandate row #8 carries this exact ticket | `sed -n '188,196p' design_docs/fleet-mission.md` | Verbatim: "`quorum:zero-signal-guard-vacuous-with-controller-verdict`: `quorum.go:164-165` counts the controller as present before the zero-signal guard (:176). A controller-only "quorum" proceeds (ailang#651)." (Row currently `[PARKED-ON-LANE · iteration 28]`; this plan is its resume step after revision 3.) |
| P7 | Existing all-absent regressions pass nil controller and must stay untouched/green; serialization test exists | `grep -n "func TestRunQuorum_AllAbsentRefusesToProceed\|func TestRunSeatedQuorum_NothingAnswersBlocks\|func TestSynthesis_AbsentReviewersMarshalsAsEmptySlice" internal/mission/quorum/*_test.go`; body reads; `grep -n "ReasonAuth" internal/mission/quorum/run.go` | `seating_test.go:143`, `quorum_test.go:96`, `quorum_test.go:179`. `:96` builds `{ReasonAuth, ReasonBudget}` absent outcomes and calls `RunQuorum(..., nil, fakeRunner(table))`; `:143` calls `RunSeatedQuorum(..., nil, f.run)`. Constants: `run.go:95 ReasonBudget = "budget"`, `run.go:96 ReasonAuth = "auth"`. |
| P8 | Stub helpers exist; no provider needed | `grep -n "func present\|func fakeRunner" internal/mission/quorum/*_test.go` | `quorum_test.go:13` `fakeRunner` (comment: "quorum tests never touch a real provider"), `quorum_test.go:22` `present`. |
| P9 | Mandated remedy is a present-EXTERNAL count | `sed -n '234,240p' .claude/skills/mission-control/resources/gate-2-pick.md` | "…the count that matters is **present EXTERNAL reviewers**, and zero of them is a park, not a pass." (same range: "measured: **86 of 87** artifacts carry one; control — the single artifact without one reads `blocked`"; its `quorum.go:168` reference is stale — the controller increment is at :165, per P1). |
| P10 | Quorum code at HEAD is identical to origin/dev `0a2202ad6` | `git diff --stat 0a2202ad6..HEAD -- internal/ cmd/ \| tail -1`; control `git diff --stat 0a2202ad6..HEAD \| tail -1` | Scoped diff: empty. Control (unscoped): `1 file changed, 82 insertions(+), 15 deletions(-)` (the design doc) — the instrument sees a diff when one exists, so the empty scoped result is real. |
| P11 | Seated-regression helpers and API | `grep -n "^func \|^type " internal/mission/quorum/seating_test.go`; `sed -n '78,150p' internal/mission/quorum/seating_test.go`; `sed -n '107,140p' internal/mission/quorum/seating.go` | `seating_test.go:80 func seat(model string, present bool, v Verdict)` (absent ⇒ `ReasonBudget`), `:88 type poolRunner struct{ mu; up map[string]bool; calls []string }`, `:94 func (f *poolRunner) run(...)`. API: `Selection{Primary, Reserve, Benched []string}`; `RunSeatedQuorum(docPath, docBody, isoTS string, sel Selection, seats int, maxCostUSD float64, controller *ControllerReview, runner ...)` at `seating.go:112` reruns `synthesize(q.Reviewers, controller)` after reserve replacement and benched recall (benched recalled only when `presentCount(q) == 0`, tagged `TierAuthorVendor`); `presentCount` at `seating.go:132`. Existing `TestRunSeatedQuorum_RecallsTheAuthorsVendorOnlyWhenNobodyAnswers` (`:115`, controller pass) has benched `z` PRESENT, so it is unaffected by the fix. |
| P12 | Tier-2 regression seam | `grep -n "^func " internal/mission/quorum/tier2.go internal/mission/quorum/tier2_test.go`; `sed -n '60,95p' internal/mission/quorum/tier2.go`; `sed -n '76,106p' internal/mission/quorum/escalate.go` | `tier2.go:72 func SynthesizeWithTier2(tier1 []*ReviewerOutcome, controller *ControllerReview, tier2 *Tier2Result) Synthesis` starts with `synthesize(tier1, controller)` and only assigns `SynthBlocked`; `Tier2Result{Decision, Reviewers []*ReviewerOutcome}` (`tier2.go:26`). Existing direct-call pattern: `tier2_test.go:131 TestSynthesizeWithTier2_NilIsTier1Identical`; existing reject tests `:34 TestRunTier2_EscalatedRejectBlocks`, `:94 TestRunTier2_OverCapDegradesButComposes`, `:146 TestRunQuorumWithEscalation_EscalatesOnPremiseObjection`. `ShouldEscalate` (`escalate.go:76`) escalates only on a present premise-class reject, a high-stakes doc, or a split, so the Tier-2 regression must call `SynthesizeWithTier2` directly. |

## Verify profile (from the fleet charter)

`go-compiler` plus the mission-loop-change pre-flight as the done-gate. For this Go-only, package-local change the executor runs: `go build ./internal/mission/quorum`, `go vet ./internal/mission/quorum`, `go test ./internal/mission/quorum -count=1`, `go test -race ./internal/mission/quorum -count=1` (all on the rebuilt tree). The controller-side done-gate at land time (push reach + dry-run healthy/degraded + `make test-launchd-drivers`) is unchanged by this sprint and recorded by the controller.

## Sprint JSON (tracked copy — authoritative live file at `.ailang/state/sprints/sprint_M-QUORUM-ZERO-SIGNAL.json`; NOT git-ignored — `.gitignore:107` re-includes `/.ailang/state/sprints/*.json` — written by the planner but left uncommitted)

```json
{
  "sprint_id": "M-QUORUM-ZERO-SIGNAL",
  "status": "not_started",
  "created": "2026-10-12T00:00:00Z",
  "verified_at": "8d0d3de0e",
  "design_doc": "design_docs/planned/m-quorum-external-zero-signal-guard.md",
  "sprint_plan": "design_docs/planned/sprint-plan-quorum-zero-signal-guard.md",
  "ticket": "quorum:zero-signal-guard-vacuous-with-controller-verdict",
  "github_issues": [
    651
  ],
  "registry_reuse": [
    {
      "milestone": "M1_GUARD_EXTERNAL_COUNT",
      "package": "internal/mission/quorum (seating.go:presentCount)",
      "action": "none",
      "reason": "Do NOT reuse seating.go:presentCount(q *QuorumResult) in synthesize: different layers — the helper counts q.Reviewers on a built *QuorumResult while synthesize runs on []*ReviewerOutcome before any result exists, and its outcomes loop already checks VerdictReject, so the retained inline externalPresentCount increment rides in the same pass; reuse would be circular or a second pass (design doc V16)."
    },
    {
      "milestone": "M1_GUARD_EXTERNAL_COUNT",
      "package": "internal/mission/quorum (test helpers present/fakeRunner, seat/poolRunner)",
      "action": "reuse",
      "reason": "Existing deterministic in-process stubs: fakeRunner (table-driven, absent-by-default for unlisted models) and present() cover every quorum_test.go matrix row; the seated regression reuses seating_test.go's poolRunner (records calls, for 'fallback attempted') and seat(); the Tier-2 regressions call SynthesizeWithTier2 directly with a hand-built *Tier2Result (RunQuorumWithEscalation would not escalate an all-absent Tier-1). No new runner, no credentials, no provider calls."
    }
  ],
  "velocity": {
    "target_loc_per_day": 560,
    "estimated_total_loc": 140,
    "estimated_days": 0.25
  },
  "features": [
    {
      "id": "M1_GUARD_EXTERNAL_COUNT",
      "description": "Change synthesize so the zero-signal guard counts present EXTERNAL reviewer outcomes only (rename to externalPresentCount; remove the controller increment; retain the guard and its exact objection text 'no reviewer produced a verdict (all absent) — refusing to proceed on zero signal'; no early return before the guard; Tier-2 stays monotonic; accounting/absence records unchanged), add the deterministic controller/signal matrix in internal/mission/quorum/quorum_test.go, the all-absent seated regression with controller pass in seating_test.go, and the Tier-2 zero-signal regression plus positive control in tier2_test.go — with the pre-fix red run of the new controller-only tests and the pre/post full-suite green pair as bounding controls, and mutation drill MU1 recorded.",
      "estimated_loc": 140,
      "dependencies": [],
      "acceptance_criteria": [
        "Entire matrix above passes, and new controller-only tests fail before the fix. The full package suite (`go test ./internal/mission/quorum`) run BEFORE the fix stays green immediately AFTER the fix — that pre/post pair is the bounding control for any fixture the V10/V15 textual surveys could miss (oc-glm-5.3 round-2 concern, applied verbatim).",
        "Seated replacement remains functional and all absences remain named.",
        "Controller/reviewer/Tier-2 rejects retain vetoes; partial-present pass stays allowed.",
        "Existing serialization, accounting and escalation tests pass.",
        "`go test ./internal/mission/quorum` and `go test -race ./internal/mission/quorum` pass; run repository-required checks from the approved sprint.",
        "Contract comments and implementation report describe the intentional controller-only behavior change.",
        "For the absent-pair/controller-pass row, assert both model/reason pairs retained in input order, controller verdict/note retained, and totals equal the supplied costs and input/output token sums (including absent outcomes).",
        "For the two-absent/controller-reject row, assert BOTH objections in deterministic order — controller note first, then the zero-signal objection — which holds by construction: `synthesize` has no early return before the external-count guard (V13).",
        "Assert `AbsentReviewers` remains a non-nil empty slice for all-present inputs through the existing serialization test.",
        "Add a seated orchestration regression with primary `a`, reserve `b`, benched `z`, all absent, controller pass. Assert `blocked`, all three absences named, and fallback attempted; this catches an incorrect early-return fix that bypasses replacement or synthesizes before final outcomes.",
        "Add Tier-2 regression using all-absent Tier-1, controller pass, and a present passing Tier-2 outcome. Assert `blocked` and zero-signal objection retained. Add a positive control with one Tier-1 pass plus one absent and Tier-2 pass: `proceed`. Existing Tier-2 reject tests preserve veto behavior.",
        "Mutation evidence required during execution: restoring only the controller count increment must make the new controller-pass/all-absent and empty-slice/pass cases fail. Restoring the fixed source must return those tests to green. Record actual commands/results in the implementation report."
      ],
      "passes": null,
      "started": null,
      "completed": null,
      "notes": null
    }
  ]
}
```

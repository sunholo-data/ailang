# M-MISSION-ITERATION-RELIABILITY

**Status:** Approved by Mark, attended 2026-09-08: “yep please continue” in response to design approval and plan/execute request.
**Implementation:** M1–M3 implemented in isolated sprint checkout; integrated review/checks completing. M4 live adoption pending; no reliability smoke-series success claimed.
**Priority:** P0 before unattended adoption. **Release:** unassigned.
**Dependency:** M-MISSION-ITERATION, including registry repair ad1bf98d3.
**Estimate:** three bounded implementation milestones plus live validation; detailed sizing at sprint planning.

## Problem and observed evidence

The runtime produced and validated a real guide commit, but neither live evaluator
finished. Initial review stopped at 34,202 fresh tokens against 30,000. The authorized
100,000-token retry stopped at 100,216 after 24 turns and 34 tool calls, with repeated
reads of the same design, plan and diff. Recorded second-review cost was $0.02533176;
cache reads were separately reported, not part of the fresh-token cap.

The failed canary left a scheduler disable marker and temporary runtime binding overnight.
The attended session restored these after confirming terminal state and process exit.
Retry required manual construction of a new input, authority record and source snapshot.
The failed status returned an empty next_action. A live dry-run also caught a registry
override advertised but unwired; ad1bf98d3 repaired it with real-loader regression tests.

The target is accepted useful work with fewer interventions. More tokens, more prompts,
and a green unit-test suite alone do not demonstrate that outcome.

## Goals and boundaries

1. Give the evaluator a compact, exact review target and a bounded procedure.
2. Prepare evaluator-only continuations without hand-assembling inherited evidence.
3. Make every stopped state identify its next valid operation.
4. Restore canary-owned installation/schedule changes after verified stop, including recovery after process death.
5. Demonstrate live acceptance and zero-dispatch completed replay before expanding adoption.

No automatic merge, automatic model switching, mission onboarding, distributed execution,
new authority writer, or fleet cutover. Preserve the approved full-v1 stage accounting.
No general-purpose autonomous retry loop. Original failures and acceptance rows remain immutable.

## Design decisions proposed for approval

| Decision | Proposed rule |
| --- | --- |
| Review scope | Runtime-built packet from frozen evidence; reviewer retains read access to source |
| Review budget | Default for this pilot 100,000 fresh tokens / 20 minutes / $2; limits always explicit |
| Repeated reading | Measure exact repeated tool calls and return progress diagnostics; do not equate repetition alone with failure |
| Retry | Operator-requested evaluator-only successor; original author never runs again |
| Independence | Check new evaluator against actual author vendors, including imported prerequisites |
| Cleanup | Restore only owned unchanged configuration after verified stop; preserve evidence and ambiguity |
| Adoption | One successful existing candidate, completed replay, then two distinct bounded tasks |

## 1. Focused review packet and progress evidence

Extend the existing stage request builder, not the provider harness core. For evaluation,
materialize a read-only packet outside author/evaluator roots containing exact base and
candidate revisions, changed-path list, bounded literal diff, criterion IDs, existing
hard-check receipts, authority/provenance references, and the required result schema.
Bind packet content and digest into the frozen request. Limits: 128 KiB aggregate text;
large diffs must be explicitly marked incomplete with exact source locators. Never hide
omitted evidence or treat a truncated packet as sufficient proof.

Instructions: inspect the packet once, run/read the named hard checks, inspect only source
needed to settle each criterion, then produce criterion-by-criterion findings. The initial
pilot uses the same independent route to test whether packet shape resolves the observed
loop. A different route requires an explicit successor policy, not a post-dispatch fallback.

Replace dispatch's no-op handler with a bounded observer of existing executor events.
Record tool-call count, last completed tool, last progress time and exact repeat counts.
Repeated reads are diagnostic only: reading a file twice can be legitimate, and model
reasoning quality cannot be inferred from an identical tool name. Existing time/token/cost
limits remain hard stops. The pilot trace determines whether a later deterministic
no-progress cutoff is justified; this sprint does not invent one from two samples.

Acceptance: packet binds the candidate; truncation is explicit; evaluator cannot modify
product content; observed tool progress survives process exit; failure reports distinguish
budget exhaustion from a completed negative review. No passing verdict inferred from activity.

## 2. Evaluator-only successor preparation

Proposed CLI: `mission retry-review NAME --work-item ID --new-id ID --output FILE`
with explicit token/time/cost flags and optional explicit evaluator route. This is a
prepare-only operation: no dispatch, budget reset of the old item, or mutation of its input.
The output is a reviewable successor plus a manifest of exact imported acceptance hashes.

Require terminal failed/cancelled parent with no live/prepared/ambiguous child and an
accepted author artifact. Verify immutable candidate/authority objects still exist. Import
actual author identity from acceptance, not current registry defaults. Preserve criteria,
allowed paths and verification policy. Reject arbitrary prerequisite substitutions.

Limits/routes may change only in the new input. Human approval references remain supplied
through the existing content-bound authority contract. The command does not synthesize an
approval from the fact it was invoked. Make missing authority actionable with exact required
artifact digests; support generating the draft before approval and final validation afterward.
A source already accepted as a prerequisite must not require another author stage.

Acceptance: generate a successor for the real failed canary; dry-run with committed attended
authority; one evaluator dispatch and zero author dispatches; old bytes/hashes unchanged;
refuse active/ambiguous parents, missing Git objects and mismatched author provenance.

## 3. Actionable failure and recoverable cleanup

Keep compatibility with existing state fields. Add a structured diagnostic projection for
failure category and suggested command while preserving full original reason text. Define
stable categories from typed outcomes (budget, deadline, verification, provider failure,
decision, ambiguous execution); do not classify arbitrary stderr by substring. Unknown
causes remain explicit. Every non-completed state must expose the next valid action or
explain why no automatic action is valid. Completed status points to acceptance evidence.

Expose a versioned `mission confirm-stopped` operation for the conditions already supported
by ConfirmMissionWorkItemStopped. Require explicit operator attestation, identify exact
work/child attempts, and retain that attestation in execution evidence. Reject stale versions,
wrong states and any unsupported generic outcome_unknown case. Do not expand the store API's
meaning to turn unknown successful effects into safe retries. Generic ambiguity continues
to require investigation and a separately defined resolution.

Replace the temporary Python canary wrapper with a small lifecycle component using existing
mission placement/scheduling helpers where applicable. Before any mutation, persist an owned
installation record containing mission ID, operation ID, prior bytes/existence, installed
hashes and associated work item. Never store credentials. Serialize owned activation on the
host. Scope support initially to the existing local Docs marker and runtime binding.

On success OR terminal failure, inspect owned process termination before restoring only
unchanged canary-owned files. A terminal DB label alone is insufficient. If a file changed
externally, or process identity is unknown, preserve it and expose cleanup_pending with the
specific reason. Do not unlink another operation's marker. Retain DBs, receipts and worktrees.
Provide inspect/recover for the installation record so a killed wrapper cannot strand an
invisible pause. Recovery performs the same checks and is idempotent. Existing commands must
allow read-only access to retained runtime evidence after restoring the previous binding;
resolve installation records explicitly, without creating a second writable runtime state.

Acceptance: normal completion/failure restores absent/present baselines correctly; external
edits survive; process death before/after each mutation recovers; ambiguous child prevents
unsafe cleanup; no unrelated schedule/process changes; repeated recovery does nothing extra.

## Verification and rollout

Hermetic checks first: actual command/registry path, not injected loaders alone; frozen
successor acceptance; invalid authority; replay with zero dispatch; failure categories;
concurrent activation; crash/cleanup matrix; no inference or installed-user-state mutation
inside tests. Run appropriate changed-package/race suites, lint/build and shell fixtures.

Live gate: evaluate the saved 58f9fd4bc candidate successfully, replay the completed successor
with unchanged acceptance and zero new provider calls, and demonstrate automatic cleanup.
Then two distinct bounded tasks must finish with explicit hard checks and independent review.
Record wall time, fresh/cache tokens separately, metered/list-price/unknown cost, repeated
reads, duplicate dispatches and attended interventions. This is a smoke series, not a
statistical productivity claim. Proposed combined new trial spend cap $5; stop at exhaustion,
not an unbounded per-retry allowance. Freeze the two additional task briefs before inference.
Live trial activation follows reviewed plan approval; no trial is launched by this design.

## Implementation boundaries and reuse audit

- `internal/mission/iteration/runtime_stage.go`: existing frozen request, prerequisite and acceptance reuse.
- `internal/mission/dispatch/run.go`: ExecuteStreaming currently receives NoOpEventHandler.
- `internal/coordinator/mission_work_item_finish.go`: terminal finish explicitly clears next_action;
  ConfirmMissionWorkItemStopped handles only operator_cancelled/deadline_exceeded.
- `cmd/ailang/mission_iteration_cmd*.go`: existing same-binding status/resume/cancel surface.
- Existing mission installation/doctor helpers: audit atomic restoration/ownership support at sprint planning;
  do not claim a reusable transaction exists until the actual helper contract is checked.
- `canary/run-when-idle.py.txt`: observed failure deliberately holds binding/marker, the concrete cleanup defect.

Related designs: m-mission-runtime-contract, m-mission-iteration-sprint-plan and
m-mission-recovery-sprint-plan. This audit read the runtime/iteration contracts and
current recovery implementation. This increment operationalizes their existing recovery and
accepted-output objectives; it does not duplicate dispatch/store design or redefine authority.

## Axiom assessment

A1 Determinism +1 (bound packet/successor); A2 Replayability +1 (retained original work);
A3 Effect Legibility +1 (owned installation record); A4 Explicit Authority +1 (no invented approval);
A5 Bounded Verification +1 (bounded packet/checks); A6 Safe Concurrency +1 (serialized activation);
A7 Machines First +1 (actionable diagnostics); A8 Minimal Syntax 0; A9 Cost Visibility +1;
A10 Composability +1 (reuse runtime/store); A11 Structured Failure +1; A12 System Boundary +1.
Net +11 design intent; no language-semantics changes and no claimed live reliability improvement yet.

## Approval and completion

This concrete contract requires Mark's design approval before sprint planning and implementation
under repository routing. Existing authorization allowed this audit/design and correction of
stale M6 records. It does not imply automatic fleet adoption or uncontrolled trial spending.
Original M6 packet criterion is now satisfied; successful live adoption remains a separate open gate.

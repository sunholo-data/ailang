# Recovery Plan: M-COORDINATOR-CHILD-ENV-OPENCODE-RETRY-STORM

**Iteration:** 305 recovery, revised for iteration 306
**Status:** `needs-human-review` — human approved, but the single permitted re-quorum remained BLOCKED
**Base verified:** `origin/dev` at `0b35abd5d`
**Canonical design:**
[m-coordinator-child-env-opencode-retry-storm-recovery-design.md](m-coordinator-child-env-opencode-retry-storm-recovery-design.md)

## Verdict

Do **not** start implementation. Human directive D-50 approves the
narrowed recovery and authorizes `execute sprint`, but it does not waive the recorded quorum
gate. Fresh quorum round 1 blocked 3/3; after the one permitted revision, round 2 also
blocked (two present rejects and `gpt5-6-sol` absent on budget). Mission-control therefore
requires this item to park `needs-human-review`. The recovered sprint plan says a
"narrow-refinement carve-out" makes quorum round 2 approved, while its design document says
round 2 is **BLOCKED** and that approval must wait for a fresh quorum pass. The carve-out
cannot replace that approval.

The self-contained canonical design linked above preserves immutable route authority, narrows
executable work to OpenCode, and corrects the durable contract to use execution generations.
Generation fencing is required because `RequeueTask` also advances task-chain stages,
`RetryAllFailedTasks` is a second retry path, and completion messages previously carried no
prior-run discriminator. Generic task updates are explicitly forbidden from overwriting
dispatch-owned fields.

## Preserved sources and disposition

| Source | Disposition | Gate before use |
|---|---|---|
| `3500db0a7` M1 | Reviewable code/test candidate; it has a clean three-way merge against current `origin/dev`. Its superseded broad design/plan files are not transplantable. | Fresh quorum, then M1 regression gates on the resulting tree. |
| M3 contract | Candidate specification only. | Fresh quorum plus isolated CI Firestore emulator with named transaction PASS evidence. |
| M4 contract | Candidate specification only. | M1/M2a/M3 gates, then call-bound, deadline, CAS, and notification tests. |
| Uncommitted M2–M4 in `.wt-iter302` | Parked; do not stage, cherry-pick, or use as a base. | Superseded by fresh implementation after approval. |

## Bounded scope after approval

1. **M1:** transplant the production/test hunks of the committed immutable-route-authority
   unit from `3500db0a7`; add nil-safe config validation and exclude its stale documents.
2. **M2a:** only the selected OpenCode Cloud Run child: use one resolved executable for health
   and execute, health before clone/plugin work, and an internal 15-second maximum preserving
   parent cancellation. A missing child is a typed permanent child-local pre-clone failure
   after the coordinator's accepted RunJob; it does consume that reservation.
3. **M3:** SQLite/Firestore durable generation, reservation, consumed-lease, deliberate
   single/bulk/stage-requeue, generic-update preservation, and terminal-CAS parity.
4. **M4:** reservation before audit/RunJob; internal audit=10s and RunJob=60s under lease=2m;
   typed retry classification; generation-fenced completion; winner-owned at-most-one
   notification attempt.

Codex, Pi, Motoko, and Claude executable changes require a separate design that justifies
their shared threat model and lifecycle semantics, followed by its own quorum and approval.

## Required gates

1. Fresh quorum PASS for the canonical narrowed design linked above.
2. Human approval and `execute sprint` are already satisfied by D-50; do not request them again
   after quorum PASS.
3. M1 route matrix/zero-effect tests, targeted Go tests, `make test`, `make lint`,
   `make check-boundaries`, and focused race tests.
4. M2a missing-binary, pre-clone, poison-PATH, injected deadline/cancellation tests; no
   unrelated adapter diffs.
5. M3 SQLite migration plus isolated CI Firestore-emulator transaction suite with named PASS;
   a CI skip fails the gate. Single retry, bulk retry, task-chain stage advance, stale generic
   update, and prior-generation completion fixtures are mandatory.
6. M4 exactly-three-call/crash-point, reservation-before-effect, 10s/60s/15s/2m,
   `LostRace`, generation-aware terminal CAS, at-most-one notification attempt, parent
   cancellation, and no-cloud-`ResetTaskToPending` controls.
7. Final target suite, `make test`, `make lint`, `make check-boundaries`, focused race, and
   mutation evidence. A canary needs separate operator authority.

## Attribution

This amendment was created in a new `origin/dev` worktree. It preserves attribution to the
recovered design/plan/JSON, M1 commit `3500db0a7`, and parked `.wt-iter302` diff without
altering any of them.

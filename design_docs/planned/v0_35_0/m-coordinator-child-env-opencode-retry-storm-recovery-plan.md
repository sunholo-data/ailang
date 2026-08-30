# Recovery Plan: M-COORDINATOR-CHILD-ENV-OPENCODE-RETRY-STORM

**Iteration:** 305  
**Status:** `blocked_pending_fresh_quorum` — not executable unattended  
**Base:** `origin/dev` at `3fc7be9b8`

## Verdict

Do **not** start or resume this sprint unattended. The recovered sprint plan says a
"narrow-refinement carve-out" makes quorum round 2 approved, while its design document says
round 2 is **BLOCKED** and that approval must wait for a fresh quorum pass. The carve-out
cannot replace that approval.

The incident root cause and M1/M3/M4 contracts remain aligned: immutable route authority,
durable bounded dispatch state, and winner-owned terminal notification. Recovered M2's
five-adapter executable migration is broader than the measured missing-OpenCode-child
incident, so it is excluded from this recovery sprint.

## Preserved sources and disposition

| Source | Disposition | Gate before use |
|---|---|---|
| `3500db0a7` M1 | Reviewable standalone candidate; it has a clean three-way merge against current `origin/dev`. | Fresh quorum, explicit user approval and `execute sprint`, then M1 regression gates on the resulting tree. |
| M3 contract | Candidate specification only. | Fresh quorum plus isolated CI Firestore emulator with named transaction PASS evidence. |
| M4 contract | Candidate specification only. | M1/M2a/M3 gates, then call-bound, deadline, CAS, and notification tests. |
| Uncommitted M2–M4 in `.wt-iter302` | Parked; do not stage, cherry-pick, or use as a base. | Superseded by fresh implementation after approval. |

## Bounded scope after approval

1. **M1:** apply the committed immutable-route-authority unit from `3500db0a7`.
2. **M2a:** only the selected OpenCode Cloud Run child: use one resolved executable for health
   and execute, health before clone/plugin work, and an internal 15-second deadline. A missing
   child is a typed permanent pre-side-effect failure.
3. **M3:** SQLite/Firestore durable reservation, consumed-lease, explicit-requeue, and
   terminal-CAS parity.
4. **M4:** reservation before audit/RunJob; internal audit=10s and RunJob=60s under lease=2m;
   typed retry classification; winner-only terminal notification.

Codex, Pi, Motoko, and Claude executable changes require a separate design that justifies
their shared threat model and lifecycle semantics, followed by its own quorum and approval.

## Required gates

1. Fresh quorum PASS for this narrowed M1/M2a/M3/M4 plan.
2. Explicit user approval and `execute sprint` instruction.
3. M1 route matrix/zero-effect tests, targeted Go tests, `make test`, `make lint`,
   `make check-boundaries`, and focused race tests.
4. M2a missing-binary, pre-clone, poison-PATH, injected deadline/cancellation tests; no
   unrelated adapter diffs.
5. M3 SQLite migration plus isolated CI Firestore-emulator transaction suite with named PASS;
   a CI skip fails the gate.
6. M4 exactly-three-call/crash-point, reservation-before-effect, 10s/60s/15s/2m,
   `LostRace`, terminal-CAS, single-notification, and no-`ResetTaskToPending` controls.
7. Final target suite, `make test`, `make lint`, `make check-boundaries`, focused race, and
   mutation evidence. A canary needs separate operator authority.

## Attribution

This amendment was created in a new `origin/dev` worktree. It preserves attribution to the
recovered design/plan/JSON, M1 commit `3500db0a7`, and parked `.wt-iter302` diff without
altering any of them.

# Mission Dashboard — V1
*Iteration 345, 2026-09-07. History: v1-mission-log.md and charter STATUS.*

## Goal and delivery
- Release v0.35.1; N=12 design docs before v1.0.0; goal unmoved (this was a CI/HARNESS item).
- `dev` was RED on the REQUIRED `test` context, so every open PR in the repo was blocked.
- Cleared by PR #1074 → `16f0cb741`: `cmd/ailang/exec.go` 807 → 614, a pure byte-for-byte
  move of `spanningEventHandler` into `cmd/ailang/exec_events.go` (193 deletions, 0 additions).
- Cause was an attended checkpoint (`8c41d41d4`), not a sprint. Second instance of that class.

## Up next (banked)
1. m-cachesrc-cognitive-complexity — PR #1071 is open, mergeable and now unblocked.
2. Orphan sweep — #1071 (iter 341/342) and #1073 (iter 344 park) both still open, no log entries.
3. m-exec-event-handler-untested — NEW: 4/4 mutations of the exec event handler survive green.
4. m-coordinator-codex-401 — coordinator websocket auth diverges from the healthy OAuth CLI.

## Routing and cadence
- Controller `claude:claude-opus-5`. Designer deliberately not spawned (no new doc for a CI red).
- The pinned `codex:gpt-5.6-sol` bucket was AT CAPACITY all iteration (probe rc=1) — planner and
  executor both fell to their declared pi lanes: kimi-k3 and deepseek-v4-flash. Both rc=0.
- The planner's Agent spawn was DENIED by the spawn-pin hook (`deny:provider-pin`), as documented.
- Evaluator `sonnet` via the Agent tool: PASS 91/100, zero blocking. Generator != judge.

## Parked on Mark
- 59 ledger rows, five OPEN: D-55 threat scope; D-56 reviewer independence; D-57 cache naming;
  D-58 pi-runner snapshot direction; D-59 iter339 three-round cap disposition.
- D-60 (launchd notification recovery scope) is parked in the UNMERGED #1073, so it is not yet
  a ledger row on dev. No unattended answer was inferred for any of them.

## CI, quota and workspace
- Merge SHA `16f0cb741`: 20 checks; `test`, `lint`, `build`, `docs-gate` all green.
- ONE red remains on dev — `launchd drivers (bash 3.2)`, parked on D-60, proven inherited
  (`20 passed, 7 failed` identical at PR head and base; zero `tools/launchd/` files in the diff).
- No metered spend this iteration; every lane was a subscription or flat-rate bucket.
- Gate-4 base `16f0cb74103ec903c8c596e69f775c96e5674f71` at 2026-09-07T08:09:53Z.

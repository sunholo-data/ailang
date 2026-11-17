# Fleet mission dashboard — iteration 117

Updated 2026-11-17; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED [HARNESS].** Row 13 (D-FLEET-18 A probe work) and row 19: the two live `test_motoko_connection_probe.sh` flakes fixed → #2385 / `b1942b7f9` (judge sonnet PASS 88, 0 blocking). Both were defects in the test, not the probe.
  - Timer arm: bash 3.2 dropped a SIGTERM sent before `exec` because of the suite's EXIT trap. Now SIGKILL, plus a TERM-ignored guard that reds 20/20 on a revert.
  - Refusing live path / OpenRouter peer: the `lsof` stub turned an empty lane-file read into the OpenRouter peer. Now fail-closed, plus a forced empty-read arm. Reproduced only by forcing the window.
  - Dev CI on the merge ran the fixed suite: `PASS: 61`, 58/58 suites.
- **Measured first:** 8 probe-signature failures in 799 dev CI runs (10-08..11-17), none in the last 300.
- **Tickets:** 22 → 20 open. `eval:motoko-connection-probe-orphaned` (row 58) left open: retire-or-own was not ruled in those words.
- **Ledger:** 44 rows, 2 OPEN (D-FLEET-44, 45, stack-only).
- **Routing:** opus controller. Planner opus (Agent). Executor haiku via `claude-sub` (attempt 1 rc 127, no `timeout` on the rig; attempt 2 rc 0). Judge sonnet (Agent, own worktree). Designer not needed (mechanical test fix). Codex lanes probe rc 75 (quota unreadable).
- **CI:**
  - PR: `launchd drivers` was **skipped**, because the `ci.yml` changes filter has no `tools/eval/` arm. Required `test` went red on `cmd/ailang`'s hang guard (the same as dev 11-16) and was green on rerun.
  - Dev merge: `Build macos-latest` red on `TestRuntimeNestedStopUsesRemainingDeadline`, the inherited flake from iterations 115/116, now on a fourth commit. Handed to V1/attended.
- **Goal:** clause 2 moved (2 resolutions). Clause 1 UNMEASURED, 3 MET, 4 met by its letter, 5 preserved (done-gate run).
- **Cost:** metered $0.
- **Records 95–117** sit on stacked draft PRs (#2169 → … → #2383 → this fire). origin/dev's log still ends at 94.
- **Gate 0 self-notice:** watermark form rc 2 (twenty-first fire).
- **Parked on Mark:** D-FLEET-44, D-FLEET-45 (unchanged).
- **Next:** row 33 (D-FLEET-25 A, drafts #2165/#2082), row 41 (D-33 A), rows 51/53 (D-36/38 A, #2108), and D-42's fleet half (#2141–#2150).
- **Evidence:** [log](fleet-mission-log.md), [iteration 117](fleet-mission-evidence/iteration117/).

# Iteration 143 — gates and landing evidence

## Reality check (controller, before routing)
- PR #2473 head `04f307466`; the four widening mutants named by the i142 judge, each applied by `sed` to a scratch helper and run as `IDLE_TEST_HELPER=<scratch> /bin/bash tools/launchd/test_mission_idle_fingerprint.sh`: b `eval_MISSION-ITER-*` rc 0, e `eval_*_record.json` rc 0, f `eval_MISSION-ITER-*_*.json` rc 0, g `:(exclude)*_record.json` rc 0 (all SURVIVED; each sed verified to change line 80).

## Executor gates, re-run by the controller outside the sandbox (worktree at 4eb80c825's tree)
- plain test rc 0, 3 PASS lines
- `--mutations` rc 0, 21 KILLED (17 → 21), the four new ones:
  - exclude-record-eval-no-suffix KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
  - exclude-record-eval-any-mission KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
  - exclude-record-eval-any-verdict KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
  - exclude-record-any-dir KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
- shellcheck (helper + test) rc 0; `make check-changelog` rc 0; helper unchanged (`git diff --quiet` rc 0)
- `make test-launchd-drivers` rc 2: 59/62 suites passed; failed `test_mission_workdir_sync.sh` (rc 1, `current clone verdict error:git-timeout`), `test_skill_sync.sh` (rc 99, `independent arm watchdog expired`), `tools/eval/test_motoko_connection_probe.sh` (rc 1, `lane treatment exceeded 4s sampling deadline`). None references the idle fingerprint (grep rc 1). Re-run alone: skill_sync rc 0, motoko probe rc 0, workdir_sync rc 1 twice. Control: workdir_sync on the pristine base `04f307466` rc 0 once, then rc 1 (`behind clone not synced: error:git-timeout`) at load average 9.3. Load-sensitive timeouts; base fails the same way. CI `launchd drivers (bash 3.2)` green on the PR head and on the merge.

## Judge
- sonnet, Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i143-judge` at `4eb80c825`: PASS 91/100, LAND, 0 blocking ([judge.md](judge.md)).

## Landing
- dev required set green at `f43d12e3c` (Release v0.64.1): `test`, `lint`, `build`, `docs-gate`, `checks` all success (SHA-pinned poll). `test` had been fixed by `9ed9f6d7d`; `docs-gate` (MDX error in `docs/docs/reference/security-model.md`) by `f43d12e3c`.
- rebase `04f307466..4eb80c825` onto `f43d12e3c`: range-diff 4/4 `=`, 0 overlapping files (18 PR files vs 192 dev files), 0 dev files under `tools/launchd/`; plain test rc 0 on the rebased tree.
- push: first `--force-with-lease` attempt `[remote rejected] (failed)` with no message; the identical command succeeded `4eb80c825...dde340400 (forced update)`.
- PR CI at `dde340400`: 24 checks, required 5/5 success; `test-windows` (non-required) failure in `internal/effects` (`TestProcessSpawn_EffectHandler`, `TestStreamAsyncExecProcess_EchoCommand`, `TestProcessSource_EchoOutput`, `TestProcessSource_ChunkSize`: process timeouts). PR has 0 `.go` files; dev `f43d12e3c` `test-windows` success. Attributed as a Windows runner timing flake. mergeStateStatus `UNSTABLE`.
- merged `gh pr merge 2473 --squash --match-head-commit dde340400…` → `216d6562b` (2026-11-26T18:07:18Z). Squash patch-id `34ac3bd2…` = judged branch patch-id (dev moved one commit, `988772ad4`, docs/finetune only, between rebase and merge).
- dev CI on `216d6562b`: `test`, `lint`, `build`, `checks`, `test-windows` success; `launchd drivers (bash 3.2)` success.

## Done-gate (stapledon profile, idle and armed, `env -i`, `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=216d6562b`, run from `fleet-i143-judge` checked out at the merge)
- healthy: rc 0 (121 s), `DRY RUN ok`, `lanes=DEGRADED(2)`: codex `gpt-6.2-sol` probe rc 75 (over daily ration) for planner and executor. The ration is external to this change.
- degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i143`, `MISSION_PROBE_TIMEOUT=10`): rc 0 (116 s), `DRY RUN ok`, `lanes=DEGRADED(3)`, executor `bogus-model-i143` → `claude:claude-haiku-5-5`.
- no stapledon pidfile before or after. The dry-run path skips the idle pre-check (`MISSION_DRY_RUN != 1` guard), so it proves the driver still runs on the merge; the change itself is covered by `launchd drivers (bash 3.2)`.

## Resolve
- `ailang mission ticket resolve idle-fingerprint:record-eval-json-moves-head --sha 216d6562b…` rc 0: "1 occurrence(s), replied to stapledon". Open tickets 20 → 19; signature absent from the open list.

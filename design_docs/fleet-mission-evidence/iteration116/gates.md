# Iteration 116 — gates for the D-FLEET-18 A guard arm (#2382)

All runs on darwin/arm64, `/bin/bash`, unpiped; rc read from the command, not a pipe.

| Gate | Where | Result |
|---|---|---|
| Already-landed check | `git grep motoko_connection_probe origin/dev -- tools/launchd/githooks/pre-push` | 0 hits (charter rows mention the pair; the hook does not) |
| Arm still applies | planner throwaway on `b7941bbb5` | `git log aebc3dcbb..origin/dev` = 0 commits on `pre-push` and on `test_mission_scope_guard.sh`; cherry-pick rc 0 |
| `tools/launchd/test_mission_scope_guard.sh` (executor, controller, judge) | head `48ba7ed54` | 3 PASS lines, rc 0 each time |
| Controller mutant (`tools/eval/*motoko_connection_probe.sh)` replacing both paths) | head worktree, `cp` backup, `cmp` restore | rc 1, `FAIL verdict MISSION_NAME='fleet' tools/eval/xmotoko_connection_probe.sh: got allow want refuse`; restored, `cmp` silent |
| Executor mutation drill | 15 mutants (plan M1–M15) | 15/15 killed by the named arm; first two drill passes discarded by the executor (bash 3.2 quoting made the mutants syntax errors) |
| Judge mutation drill | 8 own mutants, throwaway worktree | 6 killed; 2 survive (`[mM]` first letter, `?` for `.`), each widening to one nonexistent filename; non-blocking |
| `make test-launchd-drivers` at head | branch worktree `48ba7ed54` | **rc 0**, 421 s; 455 `PASS` lines, 0 `FAIL` lines. A controller mutant ran in the same worktree for about 1 s while the suite was starting; the suite passed, so the overlap did not touch its result |
| `make check-changelog` | executor | rc 0 (17 fragments valid) |
| Driver dry-run, healthy | `MISSION_PROFILE=triage` (idle, armed: no `.disabled`, no pidfile), `AILANG_DRIVER_PINNED=48ba7ed54…`, `env -i` | **rc 0**, `DRY RUN ok` |
| Driver dry-run, degraded | same + `MISSION_EXECUTOR_MODEL=codex:bogus-model-i116` | **rc 0**, `DRY RUN ok`; `codex model 'bogus-model-i116' unusable: quota-limited` → executor falls back to `claude:claude-haiku-5-5` |
| Dry-run residue | `~/.ailang/state/mission-triage.pid` | absent after both runs |
| Hooks skipped by the executor's `-c core.hooksPath=/dev/null` commit | configured hooksPath = pin `tools/launchd/githooks` | only `pre-push` exists there, so nothing ran-or-skipped at commit time |
| Pinned guard admits the push | `git push -u origin fleet/i116-dfleet18-scope-arm` with `MISSION_NAME=fleet` | rc 0, new branch; 4 paths |
| Closing-keyword scan, PR body | `grep -ciE '(fix(es|ed)?|close[sd]?|resolve[sd]?) #[0-9]+'` | 0 |
| Dry-run reach note | | the driver is not edited and the hook runs only at `git push`, so a dry-run cannot see this change; run for charter compliance. The real reach check is the post-merge `_scope_verdict fleet tools/eval/motoko_connection_probe.sh` on the refreshed pin |

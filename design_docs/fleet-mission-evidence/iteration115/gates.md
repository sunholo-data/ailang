# Iteration 115 — gates for D-FLEET-26 A (#2368)

All runs on darwin/arm64, `/bin/bash`, unpiped; rc read from the command, not a pipe.

| Gate | Where | Result |
|---|---|---|
| `tools/launchd/test_controller_capacity.sh` (controller, first-party) | head `a3f710cd9` worktree | **21 cases passed, fail=0**, rc 0 |
| Same suite at base | `6998e97e0` code (record worktree, code tree == dev) | 16 cases passed (the 3 draft arms and 2 new arms are absent at base) |
| Controller mutant M7 (draft exec-only signature, scratch copy via `MC_CAPACITY_DRIVER`) | head suite | rc 1, **20 passed, fail=1**: `FAIL codex-daemon-capacity-is-transient` (the arm the plan names) |
| `make test-launchd-drivers` at base | `6998e97e0` code, load 4.3/17.0/17.8 at start | **rc 0**, 329 s; suite summaries 85 passed, 0 failed |
| `make test-launchd-drivers` at head | branch worktree (run began at `e25a265ae`; the only later commit, `a3f710cd9`, touches the changelog fragment), load 11.4/12.0/15.0 | **rc 0**, 309 s; 85 passed, 0 failed |
| Driver dry-run, healthy | `MISSION_PROFILE=stapledon` (idle, armed, no live pid), `AILANG_DRIVER_PINNED=a3f710cd9`, `env -i`, stdin `/dev/null` | **rc 0**, `DRY RUN ok` |
| Driver dry-run, degraded | same + `MISSION_EXECUTOR_MODEL=codex:bogus-model-i115` | **rc 0**, `DRY RUN ok`; `codex model 'bogus-model-i115' unusable: quota-limited` → executor falls back to `claude:claude-haiku-5-5` |
| Dry-run residue | `~/.ailang/state` | pidfile none before/after; 418 entries before and after |
| Column-0 hygiene on the PR diff | `grep -cE '^\+(ERROR\|Error: Codex daemon turn failed): Selected model'` | 0 (positive `printf` control: 1); per-file 0 after the changelog edit |
| `make check-changelog`, `check-tmpfile-hygiene`, `check-home-isolation`, `git diff --check`, `bash -n` driver | executor, re-run of check-changelog by controller | all rc 0 |
| Pre-push scope guard, per path (`_scope_verdict fleet <p>` for the 4 diff paths) | head | 4/4 allow; control `internal/types/unify.go` → refuse language-core |
| Closing-keyword scan (commits `origin/dev..a3f710cd9` + PR body) | | 0 hits; control `Fixes #1` → 1 |

Executor mutation drill (9 mutants, scratch copies): see [executor-r1.md](executor-r1.md). Judge's own
9 mutations: see [judge-r1.md](judge-r1.md).

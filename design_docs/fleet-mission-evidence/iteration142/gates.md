# Fleet iteration 142 — gates

Ticket `idle-fingerprint:record-eval-json-moves-head` (stapledon 118). PR #2473.

## Reality check (controller, outside any sandbox)

- stapledon `origin/main` = `891c3bc0`. Old pathspec → `a510c43b` (iteration 119 record). Its only non-excluded file: `.ailang/state/evaluations/eval_MISSION-ITER-119_record.json`. Same for `6fd5f76c` (iteration 118).
- Adding `:(exclude).ailang/state/evaluations/eval_MISSION-ITER-*_record.json` → `beb17c4a` (2026-11-24, a product commit).
- `eval_MISSION-ITER-*` files: stapledon 7, ailang-world 0, this repo 0.

## Baseline at `162f2dcd1` (before any change)

- `test_mission_idle_fingerprint.sh` rc 0; `--mutations` rc 0 (15 mutants).

## Controller re-run outside the sandbox, at the executor's tree (later commit `b79be251f`)

| Gate | rc | Note |
|---|---:|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | 3 PASS |
| `... --mutations` | 0 | 17/17 KILLED; `exclude-record-eval` → "record-only evaluation JSON commit moved hash"; `exclude-evaluations-broad` → "ordinary evaluation JSON commit did not move hash" |
| `shellcheck` on both files | 0 | |
| `make check-changelog` | 0 | |
| `make test-launchd-drivers` | 0 | 388 s |

Diff writes outside the worktree: none (the test writes only into its own temp fixture repo).

## Judge (sonnet, own worktree at `b79be251f`)

PASS 88, LAND, 0 blocking. `make test-launchd-drivers` 61 passed. Four widening mutants survive (suffix not pinned). See `judge.md`.

## PR CI

- Round 1, head `8bf73c131`, merge ref built on dev `8571e7449`: `launchd drivers (bash 3.2)` red on `test_quorum_artifacts.sh` and `test_agents_skills_sync.sh` (`.agents/skills/sprint-planner` mirror stale). Dev `8571e7449` is itself red on the same job; `e0f7d431f` synced the mirror. Inherited. Poller killed, branch rebased.
- Rebase onto `94026beac`: `git range-diff` 2/2 `=`; dev touched nothing under `tools/launchd` or this PR's files.
- Round 2, head `3c99211fa`: 23 checks, 17 green (incl. `launchd drivers (bash 3.2)`, `lint`, `build`, `checks`, `docs-gate`, govulncheck). 6 red: `test`, `test-windows`, `Build ubuntu-latest`, `Build macos-latest` ×2, `Build windows-latest`. Every red leg fails on the same two tests, `TestEveryListBuiltinIsDelegatedOrExplained` (`internal/builtins`) and `TestLetInIfElseBranchError` (`internal/elaborate`).
- Dev head `94026beac` has the same two failures in its own `test` job. `test` is a required context (`test, lint, build, docs-gate, checks`). They come from language-core commits on dev (`4498d1c2d` parser, `a6e5304f5` std), outside fleet Authority.

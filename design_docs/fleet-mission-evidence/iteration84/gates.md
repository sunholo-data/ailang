# Iteration 84 — CI-red diagnosis, gates, done-gate, Gate 3b

## Diagnosis at base `abd4dfd82` (controller, first-party)

- `CI` run 41803697385 and `Build and Release` run 41803697383 were both `failure` at `abd4dfd82` and its two parents.
- The check-runs on `abd4dfd82` total 17. Not green: `launchd drivers (bash 3.2)` failure, `test` failure, `Build ubuntu-latest` failure, and macOS ×2 plus Windows cancelled (fail-fast).
- `test` job log: 98,252 B with ANSI stripped. `--- FAIL: TestQuorumArtifactShellSuite`, `FAIL generated_docs_and_mirror: sync`, package `internal/mission/quorum`.
- `Build ubuntu-latest` (job 126485417479, in the **Build and Release** run, not CI): 4,641,209 B. One `--- FAIL` (`TestQuorumArtifactShellSuite`) against 22,129 `--- PASS`.
- Local reproduction: `sync-agents-skills.sh --check` printed `STALE: .agents/skills/mission-control differs from .claude/skills/mission-control`. A per-file `cmp` found exactly one differing file, `resources/gate-3-triage.md`.
- Cause commit: `f22bf1f4b` (D-TRIAGE-48, 2026-11-04 08:43Z) added 49 lines to the canonical file without running the sync. The last green commit is `88d654eca`.

## Planner (opus, Agent tool) — base rc

| Command | base rc |
|---|---|
| `sync-agents-skills.sh --check` | 1 |
| `test_agents_skills_sync.sh` | 1 (4 pass / 1 fail) |
| `test_quorum_artifacts.sh` | 1 |
| `go test ./internal/mission/quorum/ -run TestQuorumArtifactShellSuite` | 1 |
| `make check-changelog` | 0 |

`diff -rq` over all six mirrored skills found the one file and no `.agents`-only content, so the `rsync --delete` sync loses nothing.

## Controller re-run at head, outside any sandbox (sprint worktree, before commit `96e1691a9`)

| Gate | rc |
|---|---|
| a `sync-agents-skills.sh --check` | 0 |
| b `test_agents_skills_sync.sh` | 0 |
| c `test_quorum_artifacts.sh` | 0 |
| d `go test ./internal/mission/quorum/ -count=1` (whole package) | 0 |
| e `make check-changelog` | 0 |
| f `env -i … make test-launchd-drivers` | 0. The 4 `FAIL`-string matches are summary labels (`22 PASS, 0 FAIL`, `PASS=25 FAIL=0`, a `LOOKUP_FAIL_OPEN` arm name, a control description), not failures |

## Done-gate dry-runs, `AILANG_DRIVER_PINNED=96e1691a9`

Run under the idle, armed `triage` profile: no pidfile before or after, and `stapledon` was mid-fire.

- **Healthy**: rc 0, `DRY RUN ok`, `skill-sync=current`. It reads `lanes=DEGRADED(3)` because codex is over its daily ration rig-wide this fire (rc 75), not because of this diff.
- **Degraded** (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i84`): rc 0, `DRY RUN ok`, `codex model 'bogus-model-i84' unusable` → falls back to `claude:claude-haiku-5-5`.
- Limitation: codex was already over ration, so the healthy and degraded runs differ only in the executor lane. The diff is markdown only, and no driver code changed.

## Mutations

- Executor: base mirror written back with `git show`, then `--check` rc 1 and the Go suite rc 1. Restored, `cmp` 0, `--check` rc 0.
- Judge (sonnet), its own: drift in `.agents/skills/sprint-evaluator/SKILL.md`, deletion of the mission-brief mirror, and an extra file in the sprint-planner mirror. All three caught.
- Controller replay of the judge's deletion mutant: rc 1 `STALE: .agents/skills/mission-brief`. Restored, `cmp` 0, rc 0, tree clean.

## Gate 3b

- PR #2116, head `96e1691a92b01498702d2a6df267bf5c11b114f9`: 27 checks complete. All required contexts (`test, lint, build, docs-gate, UI build gate, checks`) were SUCCESS.
- One non-required red: `Build macos-latest` (Build and Release run 41827278668, job 126565127880, step 8 "Run tests"). `--- FAIL: TestRuntimeNestedStopUsesRemainingDeadline (0.04s)` in `internal/effects`; log 4.67 MB, 22,144 PASS.
- Negative control: the test failed in none of the other 24 recent Build-and-Release runs (three other macOS failures had 0 matches). The diff is markdown and cannot reach `internal/effects`.
- Re-ran the failed job: `Build macos-latest` completed SUCCESS. Recorded as a timing flake, not filed: it is outside fleet scope, and the fleet does not self-source.
- Gate-1 base was steady at `abd4dfd82` (`mission-base.sh drift gate1` rc 0). `MERGEABLE`/`CLEAN`.
- Merged with `--merge --match-head-commit` → `981028c469697a5be74710555b40c7fb473b82fe`.
- Merge commit: CI `completed/success` and Build and Release `completed/success`, with 18 check-runs and 0 non-green. Read directly per workflow; the poll's own verdict string was truncated by a word-split on "Build and Release", so it was treated as a hint only.
- An earlier merge poll was started with a 3000 s cap and killed before reuse. It was replaced by a poll with the skill's 1800 s cap on a fresh verdict path, because raising the cap is D-FLEET-21's open question.

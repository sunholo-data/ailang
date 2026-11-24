# Fleet iteration 136 — controller gates (outside the sandbox, darwin/arm64, /bin/bash 3.2.57)

## Base `16d168f19` (own detached worktree `.fleet-i136-base`)
- `make test-launchd-drivers`: rc 2, **2 of 61 suites FAILED**: `tools/launchd/test_quorum_artifacts.sh` (generated_docs_and_mirror) and `tools/launchd/test_agents_skills_sync.sh` (STALE `.agents/skills/sprint-{planner,executor,evaluator}`). Same two as dev CI's `launchd drivers (bash 3.2)` job 138429997575; cause `b49dc1c59` (edited `.claude/skills/sprint-*` without running `tools/launchd/sync-agents-skills.sh`).

## Candidate
- `e4541e88b` (M1–M3 + mirror regen): `make test-launchd-drivers` rc 0, **all 62 suites passed**; live `~/.config/ailang/secrets.env` sha prefix + line count identical before/after (compared in-process, not printed).
- `66e0aae56` (after fix rounds 1 and 2): `make test-launchd-drivers` rc 0, **all 62 suites passed**; secrets file unchanged.
- Suite `scripts/test_mission_evidence_secret_scan.sh` under `env -i` + synthetic HOME: 18 arms (M2) → 19 (fix 1) → 21 (fix 2), rc 0 each time. `shellcheck -s bash` rc 0. `tools/launchd/test_agents_skills_sync.sh` rc 0.
- Commit reconstruction: the executor's final tree sha256-manifested before the M1/M2/M3 commits; `shasum -c` OK on all 5 files after.
- Regenerated `.agents/skills/sprint-*` files are byte-identical (`cmp`) to the uncommitted copies sitting in the main checkout `~/dev/sunholo-data/ailang` (an attended session ran the sync at 12:29 but committed only the `.claude` side at 12:32).

## Live rig checks (helper run with the controller's real environment; only verdict lines read)
| Input | Head | rc | Verdict line |
|---|---|---|---|
| clean file under `/private/tmp` | `e4541e88b` | 0 | clean values=18 scanned=1 |
| same file via `/tmp` (symlinked prefix) | `e4541e88b` | **2** | instrument-failure — **defect, sent to fix round 1** |
| `env` dump in a `.tgz` (the incident shape) | `e4541e88b` | 1 | LEAK hits=14 (no value in output) |
| `env` dump, plain file | `e4541e88b` | 1 | LEAK hits=14 |
| clean file via `/tmp` and via `$TMPDIR` (`/var/folders/…`) | `2db1b4604` | 0 | clean |
| final-component symlink | `2db1b4604` | 2 | instrument-failure (intended) |
| `env` dump in a `.tar.bz2` | `66e0aae56` | 2 | instrument-failure (judge finding 1 fixed; was a silent clean) |
| PR #2463 body, scanned before `gh pr create` (separate call) | `2db1b4604` | 0 | clean values=18 scanned=1 hits=0 control=ok |

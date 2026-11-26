# Fleet i141 judge report: stack #2463 / #2466 / #2471 rebased on origin/dev 63b605cbc

## Verdicts
- #2463 (evidence secret scan + Gate-4 Public publish rule): PASS, 91/100
- #2466 (mission-base origin guard + MISSION_DRIVER_ROOT call sites): PASS, 89/100
- #2471 (rotate-log --workdir + Gate-4 text): PASS, 91/100
- Overall: LAND (merge in order #2463, #2466, #2471).

## Q1 Rebase faithfulness
`git range-diff 2da0c4592..05e81cc48 origin/dev..2c9273918` printed 17 lines, 17 of them `=` (17 commits). Every rebased commit is patch-identical to its pre-rebase original.

## Q2 Interaction with the 27 dev commits
`git diff 2da0c4592 origin/dev` touches only these paths in the surfaces the stack edits: `.claude` and `.agents` `gate-2-pick.md` (quorum pool names, 4 lines), `tools/verify-gap-map`, `onboarding_guide_overrides.json`, and cmd/ailang and internal/mission/quorum (verify, pkg, check_fix, rolemodels).
- None of those call `mission-base.sh`, `rotate-log` or the scan helper.
- `internal/mission/quorum` is a different package from rotate/index.
- `gate-2-pick.md` is not edited by the stack, so there is no conflict.
- Every `mission-base.sh` call in the combined tree goes through `$MISSION_DRIVER_ROOT`: gate-1, gate-3, gate-3b, gate-4 and ref-drift, in both `.claude` and `.agents`. The only bare mentions are a comment in `mission-lane-check.sh` and the helper itself.
- `MISSION_DRIVER_ROOT` is exported by `mission-control.sh` at line 2437 and for the codex, pi and claude controller launches.
- Nothing new on dev would be refused by the i138 guard.

## Q3 Gates on the rebased tree (exit codes read without pipes, into log files)
| Gate | Exit |
|---|---|
| `go build ./cmd/ailang` | 0 |
| `go test ./cmd/ailang -run Rotate -count=1` | 0 |
| `go test ./internal/mission/... -count=1` | 0 |
| `make test-launchd-drivers` | 0 |
| `make check-context-docs` | 0 |
| `go vet ./cmd/ailang` | 0 |
| `gofmt -l cmd/ailang internal/mission` | empty |

- `make test-launchd-drivers` includes `test_mission_evidence_secret_scan.sh` (PASS), `test_mission_base.sh`, `test_skill_sync.sh` and `tools/launchd/test_agents_skills_sync.sh`, which is the `.agents` mirror check. There is no separate `scripts/test_agents_skills_sync.sh`.
- No red gate appeared, so no pristine-dev baseline was needed.
- UNMEASURED: a live dry-run of `mission-control.sh`. The stack does not edit that driver, so "healthy and degraded dry-run" does not apply.

## Q4 Mutation drills (each restored with `git checkout`)
| Mutation | Result | Killer |
|---|---|---|
| mission-base: `verify_repo` never refuses (`return 3` changed to `return 0`) | RED | `test_mission_base.sh`, 11 arms failed |
| scan: skip ZIP member bodies | RED | `test_mission_evidence_secret_scan.sh` arms zip-rc, zip-member-finding, zip-member-count, finding-count, plus compressed-unsupported and bad-archive zip arms |
| scan: ZIP errors fail open (`self.failed` not set) | RED | the same script, arms nested-archive (tar), compressed-unsupported, etc. |
| rotate-log: drop the `--workdir` stat check | RED | `TestMissionRotateWorkdirMissingSource` |
| rotate-log: ignore `--workdir` | RED | `TestMissionRotateWorkdirOverridesExternalAndShared`, `…IndexOnly`, `…StatusStream`, `…MissingSource`, `…Relative` |
| skill: gate-1 call reverted to a bare relative path in `.claude` only | RED | `test_skill_sync.sh` and `test_quorum_artifacts.sh`, as a mirror mismatch |
| skill: the same revert in BOTH `.claude` and `.agents` | SURVIVED (`make test-launchd-drivers` exit 0) | nothing |

Worktree is clean after the drills.

## Q5 Done-gate per ticket
- `publish:evidence-pack:no-secret-value-guard`: MET. The scanner has a non-vacuity test with controls and is wired into `test-launchd-drivers`. The Gate-4 skill rule is in both mirrors, which reach every mission once on dev. Dry-run: N/A.
- `skill:mission-base-relative-path-records-driver-repo`: MET for the code. The guard is tested, and all call sites use the driver root. See finding 1 on the call sites. Dry-run: N/A.
- `skill:rotate-log-index-only-targets-registry-workdir`: MET. The flag, its stat check and the Gate-4 text with the old-CLI fallback are covered by tests, and the mirror is regenerated. Dry-run: N/A.

## Findings
1. NON-BLOCKING: no test asserts that skill call sites use `$MISSION_DRIVER_ROOT/tools/launchd/mission-base.sh`. A future edit reverting one site consistently in both mirrors passes CI, and the guard only fires at runtime. Suggest a grep gate in `test-launchd-drivers`.
2. NON-BLOCKING: the mission-record docs (`design_docs/fleet-mission*`, `iteration136`..`iteration140` evidence) were scanned with a regex for common key and token patterns and had no hits. This is a pattern scan, not the value-based scanner against a real inventory.
3. NON-BLOCKING: the stack carries about 1,300 lines of log-archive and bookkeeping, which bloats the diff but is not code under review.
4. NON-BLOCKING: the dev reds that blocked the stack earlier are gone. Every measured gate above is green on the rebased tree. The full `make test` and CI were not run here.

No blocking findings.

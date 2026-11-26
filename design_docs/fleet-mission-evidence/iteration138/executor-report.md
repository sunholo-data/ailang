# Fleet iteration 138 executor report

Branch `fleet/i138-mission-base-repo-guard`, base `3f5143370`, macOS Bash 3.2.57.

M1 guards `record`, `snap`, and `drift` against the CWD's GitHub origin before base ref or state access. A failed check exits 3 with the expected slug and the observed slug or reason. Refusal tests compare the observed file presence or SHA-256 and line count before and after.

M2 uses the pinned driver path for all ten source invocations. Gate 1 propagates record failure before printing the base. The five generated counterparts are in the cumulative M2 snapshot because the sandbox denied writes to the worktree's `.agents` directory.

## Changed files and snapshots

`.snap/M1` contains the two M1 scripts. `.snap/M2` contains all 13 implementation files, including the generated copies. The report was written after both snapshots.

| File | Added lines | Removed lines | Final lines | Location |
| --- | ---: | ---: | ---: | --- |
| `tools/launchd/mission-base.sh` | 54 | 4 | 113 | Worktree and M1/M2 snapshots |
| `tools/launchd/test_mission_base.sh` | 130 | 0 | 287 | Worktree and M1/M2 snapshots |
| `.claude/skills/mission-control/resources/gate-1-observe.md` | 2 | 2 | 309 | Worktree and M2 snapshot |
| `.claude/skills/mission-control/resources/gate-3-route.md` | 3 | 3 | 919 | Worktree and M2 snapshot |
| `.claude/skills/mission-control/resources/gate-3b-ci-green.md` | 2 | 2 | 459 | Worktree and M2 snapshot |
| `.claude/skills/mission-control/resources/gate-4-record.md` | 2 | 3 | 404 | Worktree and M2 snapshot |
| `.claude/skills/mission-control/resources/ref-drift.md` | 8 | 5 | 71 | Worktree and M2 snapshot |
| `.agents/skills/mission-control/resources/gate-1-observe.md` | 2 | 2 | 309 | Generated in M2 snapshot |
| `.agents/skills/mission-control/resources/gate-3-route.md` | 3 | 3 | 919 | Generated in M2 snapshot |
| `.agents/skills/mission-control/resources/gate-3b-ci-green.md` | 2 | 2 | 459 | Generated in M2 snapshot |
| `.agents/skills/mission-control/resources/gate-4-record.md` | 2 | 3 | 404 | Generated in M2 snapshot |
| `.agents/skills/mission-control/resources/ref-drift.md` | 8 | 5 | 71 | Generated in M2 snapshot |
| `changelogs/unreleased/2026-11-25-mission-base-repo-guard.md` | 5 | 0 | 5 | Worktree and M2 snapshot |
| `design_docs/fleet-mission-evidence/iteration138/executor-report.md` | 92 | 0 | 92 | Worktree |

## Mutation proof

The untouched `cp -p` override ran first. Each mutant started from a fresh `cp -p` into its own disposable `TMPDIR`. Each row uses the exit status and `not ok` output of `MISSION_BASE_HELPER="$TMPDIR/mission-base-mutant.sh" /bin/bash tools/launchd/test_mission_base.sh`. All 30 mutants exited 1 and failed the intended arm. Each run accounted for 37 arms.

| Single helper mutation | Suite rc | Arms passed | Measured `not ok` lines |
| --- | ---: | ---: | --- |
| Untouched helper copy | 0 | 37 | None |
| Bypass the record guard with MISSION_BASE_REF set | 1 | 35 | `not ok - wrong repo with ref override leaves base absent`<br>`not ok - wrong repo with ref override preserves recorded base` |
| Move the record guard after the append | 1 | 30 | `not ok - wrong repo with ref override leaves base absent`<br>`not ok - wrong repo with ref override preserves recorded base`<br>`not ok - missing origin URL is refused`<br>`not ok - outside Git is refused`<br>`not ok - unsupported origin https://other.example/sunholo-data/ailang is refused`<br>`not ok - unsupported origin https://github.com/sunholo-data/ailang/extra is refused`<br>`not ok - multiple origin URLs including a blank entry are refused` |
| Drop the HTTPS parser branch | 1 | 27 | `not ok - wrong repo with ref override leaves base absent`<br>`not ok - wrong repo with ref override preserves recorded base`<br>`not ok - matching origin https://github.com/sunholo-data/ailang records one row`<br>`not ok - matching origin https://github.com/sunholo-data/ailang.git records one row`<br>`not ok - matching origin https://github.com/sunholo-data/ailang.git/ records one row`<br>`not ok - mixed-case MISSION_REPO matches a lowercase origin slug`<br>`not ok - wrong repo for snap is refused before deriving a ref`<br>`not ok - wrong repo for snap is refused with an unresolvable ref`<br>`not ok - wrong repo for drift is refused before deriving a ref`<br>`not ok - wrong repo for drift is refused with an unresolvable ref` |
| Require .git for HTTPS | 1 | 32 | `not ok - matching origin https://github.com/sunholo-data/ailang records one row`<br>`not ok - wrong repo for snap is refused before deriving a ref`<br>`not ok - wrong repo for snap is refused with an unresolvable ref`<br>`not ok - wrong repo for drift is refused before deriving a ref`<br>`not ok - wrong repo for drift is refused with an unresolvable ref` |
| Omit .git stripping for HTTPS | 1 | 32 | `not ok - wrong repo with ref override leaves base absent`<br>`not ok - wrong repo with ref override preserves recorded base`<br>`not ok - matching origin https://github.com/sunholo-data/ailang.git records one row`<br>`not ok - matching origin https://github.com/sunholo-data/ailang.git/ records one row`<br>`not ok - mixed-case MISSION_REPO matches a lowercase origin slug` |
| Omit trailing slash stripping for HTTPS | 1 | 36 | `not ok - matching origin https://github.com/sunholo-data/ailang.git/ records one row` |
| Drop the SCP parser branch | 1 | 34 | `not ok - matching origin git@github.com:Sunholo-Data/AILANG records one row`<br>`not ok - matching origin git@github.com:Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin git@github.com:Sunholo-Data/AILANG.git/ records one row` |
| Require .git for SCP | 1 | 36 | `not ok - matching origin git@github.com:Sunholo-Data/AILANG records one row` |
| Omit .git stripping for SCP | 1 | 35 | `not ok - matching origin git@github.com:Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin git@github.com:Sunholo-Data/AILANG.git/ records one row` |
| Omit trailing slash stripping for SCP | 1 | 36 | `not ok - matching origin git@github.com:Sunholo-Data/AILANG.git/ records one row` |
| Drop the ssh:// parser branch | 1 | 34 | `not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git/ records one row` |
| Require .git for ssh:// | 1 | 36 | `not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG records one row` |
| Omit .git stripping for ssh:// | 1 | 35 | `not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git/ records one row` |
| Omit trailing slash stripping for ssh:// | 1 | 36 | `not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git/ records one row` |
| Compare both slugs case-sensitively | 1 | 30 | `not ok - matching origin git@github.com:Sunholo-Data/AILANG records one row`<br>`not ok - matching origin git@github.com:Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin git@github.com:Sunholo-Data/AILANG.git/ records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git records one row`<br>`not ok - matching origin ssh://git@github.com/Sunholo-Data/AILANG.git/ records one row`<br>`not ok - mixed-case MISSION_REPO matches a lowercase origin slug` |
| Omit case folding on the expected slug | 1 | 36 | `not ok - mixed-case MISSION_REPO matches a lowercase origin slug` |
| Require repo identity when MISSION_REPO is unset | 1 | 21 | `not ok - snap-format prints 40-hex-sha<TAB>ISO8601-UTC and exits 0`<br>`not ok - record-last-roundtrip appends one base-gate1 row and last returns the SHA`<br>`not ok - nonvacuity-drift-fires exits 1 and prints DRIFT A -> B`<br>`not ok - steady-control exits 0 with base gate1 steady`<br>`not ok - no-record-absent-file exits 2 with no base-gate1 record yet`<br>`not ok - no-record-missing-label exits 2 with no base-gate1 record yet (never false DRIFT)`<br>`not ok - positive grep control: last returns a known-matching record`<br>`not ok - derived-main-record snaps and records origin/main's SHA and last returns it`<br>`not ok - override-beats-origin-head records the override ref's SHA, not origin/HEAD's`<br>`not ok - no-origin-head-fails-loudly exits 1 naming MISSION_BASE_REF and set-head, writes no state`<br>`not ok - drift-uses-derived-ref exits 1 with DRIFT base gate1 and the new origin/main SHA`<br>`not ok - wrong repo with ref override preserves recorded base (setup record failed)`<br>`not ok - unset MISSION_REPO permits record without an origin URL`<br>`not ok - wrong repo for snap is refused with an unresolvable ref (setup record failed)`<br>`not ok - wrong repo for drift is refused with an unresolvable ref (setup record failed)`<br>`not ok - last returns a recorded SHA with MISSION_REPO set and no origin URL` |
| Activate the guard for a present but empty MISSION_REPO | 1 | 36 | `not ok - empty MISSION_REPO permits record without an origin URL` |
| Treat a missing origin URL as a match | 1 | 36 | `not ok - missing origin URL is refused` |
| Skip verification outside Git | 1 | 36 | `not ok - outside Git is refused` |
| Accept any HTTPS host with the same slug | 1 | 36 | `not ok - unsupported origin https://other.example/sunholo-data/ailang is refused` |
| Accept malformed slug paths without shape validation | 1 | 35 | `not ok - unsupported origin https://github.com/sunholo-data/ailang/extra is refused`<br>`not ok - multiple origin URLs including a blank entry are refused` |
| Discard a blank final URL from a multi-URL origin | 1 | 36 | `not ok - multiple origin URLs including a blank entry are refused` |
| Omit the snap guard | 1 | 35 | `not ok - wrong repo for snap is refused before deriving a ref`<br>`not ok - wrong repo for snap is refused with an unresolvable ref` |
| Move the snap guard after ref derivation | 1 | 36 | `not ok - wrong repo for snap is refused before deriving a ref` |
| Move the snap guard after the SHA read | 1 | 35 | `not ok - wrong repo for snap is refused before deriving a ref`<br>`not ok - wrong repo for snap is refused with an unresolvable ref` |
| Omit the drift guard | 1 | 35 | `not ok - wrong repo for drift is refused before deriving a ref`<br>`not ok - wrong repo for drift is refused with an unresolvable ref` |
| Move the drift guard after ref derivation | 1 | 36 | `not ok - wrong repo for drift is refused before deriving a ref` |
| Move the drift guard after the SHA read | 1 | 35 | `not ok - wrong repo for drift is refused before deriving a ref`<br>`not ok - wrong repo for drift is refused with an unresolvable ref` |
| Apply the guard to last | 1 | 36 | `not ok - last returns a recorded SHA with MISSION_REPO set and no origin URL` |

## Acceptance gates

The worktree gates used a disposable `TMPDIR` under the worktree and `AILANG_STATE_DIR="$TMPDIR/state"`. The same isolation was used for the baseline and mutation runs.

| Gate | Worktree rc | Counts or result |
| --- | ---: | --- |
| `/bin/bash -n tools/launchd/mission-base.sh` | 0 | 1 script parsed |
| `/bin/bash -n tools/launchd/test_mission_base.sh` | 0 | 1 script parsed |
| `/bin/bash tools/launchd/test_mission_base.sh` | 0 | 37 passed, 0 failed |
| `MISSION_BASE_HELPER="$TMPDIR/mission-base.sh" MISSION_REPO=unrelated/caller /bin/bash tools/launchd/test_mission_base.sh` after `cp -p` | 0 | 37 passed, 0 failed |
| `shellcheck -s bash tools/launchd/mission-base.sh tools/launchd/test_mission_base.sh` | 0 | 2 scripts, 0 diagnostics |
| `rg -n 'bash tools/launchd/mission-base\.sh' .claude/skills/mission-control/ .agents/skills/mission-control/` | 0 | Gate failed with 10 matches, all in the unwritten worktree mirrors |
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | 1 | 6 passed, 1 failed on mirror equality |
| `/bin/bash scripts/check_context_docs.sh` | 0 | 13 rules, 41 skills, CLAUDE.md and SKILL.md ratchet passed |
| `make check-changelog` | 0 | 29 valid fragments, index check passed |
| `git diff --check` | 0 | 0 whitespace diagnostics |
| `make test-launchd-drivers` | **UNMEASURED** | Controller-owned gate |

The disposable tree with the generated mirrors passed the sync suite with rc 0 and 7/7 arms. Its relative-invocation search returned rc 1 with zero matches. The absolute-path review found ten executable calls per skill tree. Only Gate 1's record failure handling and the specified prose and recipe references changed alongside those invocations.

## Findings and deviations

- The baseline reproduced the existing `SC2034` on `oldn` (rc 1). M1 removed the unused local as planned. The baseline suite and inherited override each passed 13/13. The baseline relative search found 10 calls per skill tree. Mirror sync passed 7/7, context docs passed with 13 rules and 41 skills, and the changelog gate passed with 28 fragments.
- The actual mirror generator returned rc 1 with `Operation not permitted` from rsync in the protected worktree `.agents` tree. Its worktree content diff was empty. The unchanged generator returned rc 0 in an isolated in-worktree copy of the skill trees. Exactly the five planned counterparts differed from baseline and each matched its canonical source byte-for-byte. Those outputs were copied into `.snap/M2`. The worktree mirror paths still need the generated contents.
- The implementation diff is 223 added and 34 removed lines including generated mirrors, above the plan's estimate. The additional test detail isolates absent versus populated state, each URL variant, malformed and ambiguous origins, and guards placed after either ref derivation or SHA resolution.

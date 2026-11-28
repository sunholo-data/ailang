# Iteration 145 — gates (controller, outside any sandbox, darwin/arm64)

## Reality check of the ticket (first-party, before routing)

- `sunholo-data/stapledons-godot` #392 "Nightly rig: main red at 78f390c0": open, label `nightly-rig`, 40 comments, 32 of them begin `The nightly ` (the rig script's failure prefix); newest four ids 7183271161, 7205604286, 7210499888 (failure reports) and 7211245483 (iteration 123's triage note). All comments come from one account, so author cannot separate a report from a loop note.
- `tools/rig_nightly.sh` (stapledon) also posts commit **statuses** `rig/full` on the tested commit: `failure` on `d2b2d125` (23:30:51Z) and `468ceb97` (05:16:25Z). Main's tip `e0e0a697` (a record merge) has 0 statuses, and at 23:30 the red landed on a commit that was no longer the tip. Reading statuses on the tip would miss the normal case, so the issue is the durable signal.
- `gh issue list --label nightly-rig --state open`: `#392:40` on stapledons-godot; empty, rc 0, on `sunholo-data/ailang` and `sunholo-data/ailang-world`; a nonexistent label returns `[]` rc 0.
- Registry keys are decoded by a strict Go struct (`internal/mission/registry.go`) and reach the driver only through plist rendering and a reinstall, so the plan used the producer's fixed label instead of the ticket's suggested registry key.

## Baseline at `9b35c44d0` (planner worktree)

- `test_mission_idle_fingerprint.sh` rc 0 (3 PASS); `--mutations` rc 0, 21 KILLED; `shellcheck` rc 0; `make check-changelog` rc 0.

## Candidate `d1c8d568b` → `ccee1caca`

| Gate | Result |
|---|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | rc 0, 3 PASS (both commits) |
| `--mutations` | rc 0; 23 KILLED at `d1c8d568b`, 27 KILLED at `ccee1caca` (`widen`, `narrow`, `nopr`, `nouniq` added) |
| `shellcheck -s bash` helper + test | rc 0 |
| `make check-changelog` | rc 0 |
| Live read-only, world (`sunholo-data/ailang-world`) | old helper and new helper both `1c16e01ec49b…`: byte-identical |
| Live read-only, stapledon | old `7def9edae041…` → new `f10e992db683…`; summary line unchanged (`head=beb17c4 ledger=168 open=9 directives=0 inbox=0 red=0`) |
| `make test-launchd-drivers` at `d1c8d568b` | rc 2: 59 of 62 suites passed; `test_skill_sync.sh` (`error:git-timeout`), `test_mission_worktree.sh` (dirty / rendezvous arms) and `test_hook_stdout.sh` (11 s stdout hold) failed while the judge's own test runs shared the box. None references the idle fingerprint or a changed file. Re-run alone: `test_skill_sync.sh` rc 0, `test_hook_stdout.sh` rc 0, `test_mission_worktree.sh` rc 1. Control: `test_mission_worktree.sh` on the pristine `origin/dev` pin `9b35c44d0` rc 1 with the same FAIL lines. Inherited, not from this diff. |

The executor's sandbox refused the edit to `.agents/skills/mission-control/resources/gate-5-retro.md`; the controller applied the same one-phrase substitution and confirmed the two copies `cmp` identical.

## Secret scan

`scripts/mission_evidence_secret_scan.sh` over the branch's 11 changed files: rc 0, `verdict=clean`, `hits=0`, `control=ok`. A first run over `git diff --name-only origin/dev..HEAD` (two dots) returned rc 2 instrument-failure, because origin/dev had advanced by `a5bbd37c8` (a design doc) and the two-dot diff listed that file, which is absent from the branch. Re-run with the three-dot form: clean.

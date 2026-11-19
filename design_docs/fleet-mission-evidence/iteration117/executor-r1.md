# Iteration 117 executor report (r1)

Worktree: `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter117-probe-flakes`, branch `fleet/i117-probe-flakes`, base `22a72e7c5`.
Plan: `design_docs/fleet-mission-evidence/iteration117/plan.md` (sections 3, 4, 5). Nothing pushed.

## Commits

| SHA | Subject | Files |
|---|---|---|
| `85c446d61` | fix(probe): two intermittent self-test flakes in the motoko connection probe suite (fleet i117) | `tools/eval/test_motoko_connection_probe.sh`, `changelogs/unreleased/2026-11-17-motoko-probe-flakes.md`, `design_docs/fleet-mission-evidence/iteration117/plan.md` |
| (second commit, this report) | docs(i117): executor r1 report | `design_docs/fleet-mission-evidence/iteration117/executor-r1.md` |

No hook complaint was raised on either commit. `core.hooksPath` was not reported by git during the first commit.

## Commands run (in order)

1. `git status --short && git branch --show-current && cat plan.md`: rc 0. Tree clean except untracked `design_docs/fleet-mission-evidence/iteration117/`. Branch `fleet/i117-probe-flakes`.
2. `awk` extraction of the section-3 diff to `.scratch-exec/fix.diff` (105 lines), then `git apply --check`: **rc 0**. `git apply`: **rc 0**. `/bin/bash -n tools/eval/test_motoko_connection_probe.sh`: **rc 0**. `git diff --stat`: `1 file changed, 51 insertions(+), 9 deletions(-)`. Backup of the post-step-1 file taken to `.scratch-exec/post-step1-backup.sh`.
3. `awk` extraction of the changelog text from plan section 3 to `changelogs/unreleased/2026-11-17-motoko-probe-flakes.md`: written, 16 lines.
4. Suite, sequential, `/bin/bash tools/eval/test_motoko_connection_probe.sh`:

| Run | rc | Seconds | Not-ok lines | Final line |
|---|---|---|---|---|
| 1 | 0 | 78 | 0 | `PASS: 61 probe self-test arms ran` |
| 2 | 0 | 78 | 0 | `PASS: 61 probe self-test arms ran` |
| 3 | 0 | 78 | 0 | `PASS: 61 probe self-test arms ran` |

5. `make check-changelog`: **rc 0**. `✓ 21 changelog fragment(s) valid`.
6. Mutation drills (section 4), each on a copy in `.scratch-exec/`. Before each run the `cmp` against the original was checked, and each sed was checked to match exactly one line (line 235, line 502, line 518).

| Drill | Mutation | cmp vs original | rc | Red line |
|---|---|---|---|---|
| M-T1 | line 235 `kill -KILL "$timer"` → `kill -TERM "$timer"` | differs (1 line) | 1 | `not ok - bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal=0 derived=1)` at arm 59 (log line 67). Preceded by `# measurement failure evidence mode=timer rc=0 refusal_count=0 derived_count=1`. |
| M-R1 | line 502 `*) exit 1 ;;` → `*) peer="192.0.2.1:51000->203.0.113.8:443" ;;` | differs (1 line) | 1 | `not ok - a lane file read before the driver writes it samples no peer failed` at arm 32 (log line 33), plus `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443` (log line 39). |
| M-R3 | line 518 `if [[ -n "${PROBE_TEST_EMPTY_LANE_ACK:-}" ]]; then` → `if false; then` | differs (1 line) | 1 | `ok 32` then `not ok - empty lane-file window reached in 0 lanes, expected 2` (log line 34). |

   The tracked file was checked with `cmp` against `post-step1-backup.sh` after the drills: **identical**.

7. `make test-launchd-drivers` (plan section 5): **rc 0**, 337 s. `run-suites: all 58 suites passed`, `launchd drivers: tests + bash 3.2 syntax OK`.
8. `git add -- <three named paths>` and `git commit`: rc 0, `85c446d61`.
9. `rm -rf .scratch-exec`: deleted.

## Deviations from the brief and the plan

1. **First M-T1 drill was void (my error, not a verdict on the fix).** The drill command set `PROBE_UNDER_TEST=$PWD/$T`, where `$T` is the test file `tools/eval/test_motoko_connection_probe.sh`, not the probe `tools/eval/motoko_connection_probe.sh`. The suite then recursed into itself through `--classify-fixture`. The first M-T1 run printed `instrument failure, not a verdict: observed proxy spread 18.31 exceeds 4.70` and a cascade of `not ok - classification fixture: missing loopback`. The M-R1 run that followed hung for more than 9 minutes in that recursion. I stopped it: `pkill` on the drill's paths, scoped to this worktree, and checked that 0 processes remained. I did not touch any other process. The tracked file was not affected. All three drills were then re-run with `PROBE_UNDER_TEST=$PWD/tools/eval/motoko_connection_probe.sh`. The results above are from those re-runs only.
2. **Second drill attempt returned rc 127.** I used `timeout 400` in the drill loop. macOS has no `timeout` binary, so no drill ran. Re-run without it.
3. **Suite-run count.** The brief asked for 3 runs. The plan's section 5 says 5. I followed the brief (3 runs).
4. **Re-run policy.** The mT1 host-load message above was caused by the recursion, not by host load. No separate re-run was needed for host load, and no suite run showed an instrument failure.
5. **Not done.** M-T1′ and M-R2 from plan section 4 are not in the brief's drill list, so I did not run them. I also did not run the optional `# shellcheck disable=SC2016` comment from the plan.

## Verdict

- Suite: 3/3 green, `rc 0`, `PASS: 61`, about 78 s each.
- Mutation drills: all three red at the arms the plan names, with the CI signatures.
- Changelog check and launchd drivers suite: green.
- Nothing pushed.

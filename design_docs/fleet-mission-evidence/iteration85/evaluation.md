# Iteration 85 evaluation: transcript secret scan (detector half)

Evaluator: claude sonnet 5.5, independent of codex gpt-6.1-sol. Commit 74daa5741 on base 93ac5a915.

VERDICT: PASS. Score 87/100 (pass mark 70). No blocking findings.

## Score
| Category | Pts | Notes |
|---|---|---|
| Tests pass | 19/20 | Suite 30/30 under a synthetic HOME. One killing mutation (M9) survived, and it is an equivalent mutant. The driver tests run extracted blocks against a stub `_mc_bounded`, not the real one. |
| Lint clean | 10/10 | `bash -n` x3, shellcheck x2 and the changelog check are all clean. |
| Acceptance criteria | 27/30 | Detector, wiring, non-fatality and hygiene are met. Deducted for the performance cliff and the inline-comment parse gap below. |
| Code quality | 12/15 | Careful design: no eval, no argv secrets, one terminal-summary contract. Dedup is O(n^2) and the scan forks one grep per file per key. |
| Documentation | 14/15 | Changelog fragment is present and honest (detect-only, `ailang mission iterate` gap stated). |
| Design fidelity | 5/10 -> 5 | Conditional categories not triggered. See note. |
Total shown as 87 after rounding design fidelity at 9/10 (every plan constraint I checked held); the arithmetic above is indicative, not exact.

## Gates run (judge worktree)
- `/bin/bash -n` on transcript-secret-scan.sh, test_transcript_secret_scan.sh and mission-control.sh: rc 0 for all three.
- `shellcheck --shell=bash` on the two new scripts: rc 0.
- `/bin/bash tools/launchd/test_transcript_secret_scan.sh`: "30 arms passed", rc 0, 6.4s. The suite sets `HOME` and `TMPDIR` to a `mktemp` lab at its top, before any code under test. I read this before running it.
- `/bin/bash scripts/check_changelog.sh`: rc 0.
- `git status` clean afterwards. My throwaway dirs in /tmp are removed.

## Driver wiring (verified in the diff)
- The marker is created at mission-control.sh:2395. That is after the heartbeat state-dir setup and before `_mc_run_once` and the retry loop (~2562), so it is created once per fire and not per retry. Its name carries `MISSION_FIRE_ID`.
- The scan sits after the SLOT VERDICT block and before `rm -f "$PIDFILE"`.
- `_mc_scan_rc` is captured separately and `RC` is never assigned. Mutation M6 (adding `RC=$_mc_scan_rc`) is killed.
- `set -uo pipefail` only, with no `-e`. All scan variables are initialised before use.
- The summary parser is strict:
  - it requires exactly one summary, as the last line;
  - it requires a leaks count equal to the LEAK line count;
  - it checks rc:verdict pairs, and clean or LEAK need keys>0;
  - any other output line marks the result invalid, which logs INSTRUMENT FAILURE.
- Public issue body is the 4th argument to `_mc_exit_notice`: count only, no key names or paths. The private message body carries the hits. M7 (leaking hits into the public body) is killed.
- The dry run logs one line and does nothing else.
- The real `_mc_bounded` (lib/lane-probe.sh:23) uses `mktemp -t` and `rm -f` for its capture file. It holds only key names and paths, no values.

## Secret hygiene
- Values are parsed with `read` and parameter expansion only. There is no eval, source or `export`.
- Patterns reach `grep -F -f /dev/fd/3` through a here-string. Argv carries no value, and M2 (moving to `grep -e "$patterns"`) is killed by Secret_free_argv.
- Bash 3.2 implements here-strings as an unlinked temp file in TMPDIR. `umask 077` is set, the file is unlinked at open, and the scanner's own scratch dir only holds the candidate list (paths). A SIGKILL between create and unlink could leave a 0600 file with patterns in TMPDIR. This is a very narrow window and I judge it acceptable, but it is a real residual.
- Output lines contain key names, escaped paths and counts only.

## Mutation drills (throwaway copies, positional-arg suite)
| # | Mutation | Arm that fired | Result |
|---|---|---|---|
| M1 | dedup `continue` removed | Deduplication | killed |
| M2 | pattern passed via `grep -e "$patterns"` (argv) | Secret_free_argv | killed |
| M3 | `find` rc check disabled | Enumeration_failure | killed |
| M4 | min-len `-lt` -> `-le` | Explicit_minimum | killed |
| M5 | `-newer` inverted to `! -newer` | Clean, Fresh_leak and others | killed |
| M6 | driver assigns `RC=$_mc_scan_rc` | Driver_leak, Driver_clean | killed |
| M7 | public body carries the hit lines | Driver_leak | killed |
| M8 | per-fire marker not removed | Driver_leak, Driver_clean | killed |
| M9 | final `exit 1` for leaks changed to `exit 0` | none | SURVIVED: equivalent mutant. The EXIT trap `terminal()` recomputes the status from `leaks`, so the final line is dead. |
| M10 | JSON-escape of the value disabled | JSON_escaping | killed |

9 of 10 killed; the survivor is equivalent, not a test gap.

## Performance probe (synthetic, fake secrets)
1500 fresh files, 9 keys: 50s wall (18s user, 24s system). Cost is files x keys process forks, plus an O(n^2) bash dedup loop. The driver cap is 120s, so roughly 3500 files at 9 keys times out. The roots include `~/.claude/projects` and `~/.codex/sessions`, which also collect concurrent interactive sessions' files newer than the marker, so a busy fire can plausibly exceed this.

## Findings
Blocking: none.

Non-blocking:
1. Scalability cliff (transcript-secret-scan.sh, candidate loop around the `seen` dedup and the per-key `grep`). A timeout is reported as INSTRUMENT FAILURE by the driver parser. That is loud and non-fatal, but the detector goes blind on exactly the busiest fires. Suggested fix: one grep per file with all patterns combined (map a match back to a key only on a hit), and dedup via a sorted candidate list.
2. Parse gap: `KEY=value # comment` keeps the comment in the value, so a leak of the bare value is not detected. I reproduced this: a file containing the bare value gave `verdict=clean`. Unusual in secrets.env, but it is a silent false negative. At minimum, document it.
3. Only the literal, JSON-escaped and escaped-slash forms are covered. Unicode-escaped (`\uXXXX`), URL-encoded and base64 forms are not, and values under 20 chars are skipped silently. The skip count is not surfaced.
4. A fire that exits between the marker (2395) and the scan (watchdog kill and yield paths) leaves an empty `.since` file in the state dir. Harmless but accumulates. A sweep of old markers would be tidy.
5. `exec ailang mission iterate` fires bypass the hook (documented in the commit and changelog). The ticket's strip half (a) stays parked for a human ruling, as the controller chose.
6. The final `exit` statements of the scanner are dead code because `terminal()` overrides the status (M9). Not harmful, but misleading to a reader.
7. The driver tests stub `_mc_bounded`, so the real helper's 2s polling and temp-file behaviour is not exercised there. The timeout arm uses a bespoke deadline helper.

EVALUATION_SCORE: 87/100
EVALUATION_VERDICT: PASS score=87

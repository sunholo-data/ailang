# Iteration 79 — planner (opus, Agent tool, foreground, read-only; 92,712 subagent tok)

The planner returned an executor directive. `executor-directive.txt` is that directive verbatim, plus the controller's operating-rules header: commit locally only, foreground only, the scope list, and a mutation-drill path check. The planner's own decisions and measurements:

- **Pattern:** `|^ERROR: Selected model is at capacity`, line-anchored and ending before the period.
  - Measured: a bogus-model `codex exec`, redirected the way the driver does it, prints `ERROR: <msg>` at column 0, twice, then exits rc=1. No ANSI codes, no timestamp.
  - The string is already quoted mid-line in at least five repo files, and codex echoes what it reads into `$LOG`. An unanchored literal would therefore retry every rc≠0 crash of an iteration that reads them.
- **Arms:** `codex-capacity-is-transient` and `prose-codex-capacity-not-transient`, in `test_controller_capacity.sh` only. `test_driver_notify.sh` is unchanged: the house pattern is the end-to-end capacity arm, as D-FLEET-13 did.
- **Mutation drills:** M1 reverts the pattern; M2 drops the anchor.
- **PR-body caveats it required:**
  - World runs its own driver fork, so this diff does not reach the mission where the one occurrence happened.
  - About 2 minutes 15 seconds of total backoff.
  - A codex retry is a fresh run, not a resume.
  - The ticket's other half (push a finished record) is out of scope.
- **Baseline:** capacity suite 16/0 at `6a527f7f6`.
- The planner used `TaskStop` once, on a task of its own. The controller's base-suite job was unaffected: it wrote its own rc file and notified exit 0.

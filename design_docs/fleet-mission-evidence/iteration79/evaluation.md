# Iteration 79 — independent evaluation (sonnet, Agent tool, foreground, read-only)

Subject: draft PR #2065, branch `fleet/i79-dfleet26-capacity-transient`. D-FLEET-26 option A, proposed and unmerged.

## Round 1, on `9ba6889e0` (executor commit): PASS-WITH-EDITS 88/100

Rubric: pattern correctness 28/30 · false-positive and ordering 18/20 · tests and mutation resistance 22/25 · scope and changelog 10/10 · documentation 10/15.

- Format, measured: codex-cli 0.164.0 run with `-m bogus-model-xyz` exits rc=1 and writes `ERROR: {"type":"error",...` to stderr, starting at byte 0 of the line. The capacity string itself was not reproduced.
- `_mc_run_once` sends codex output `>>"$LOG" 2>&1`, and the retry loop runs `grep -qiE` on the tail written by this attempt. The `^` therefore anchors to column 0 of a driver-log line.
- False positives, measured: `git ls-files | grep -nE '^ERROR: Selected model is at capacity'` found the new fixture only. Every other copy in the repo is mid-line. The worst case is bounded at 2 retries (45 s, then 90 s).
- Ordering: `RUNTIME_QUOTA_SIG` cannot match the line and is checked first. No existing transient signature shadows it.
- Mutants: dropping the anchor was killed by `prose-codex-capacity-not-transient`. A too-narrow pattern was killed by `codex-capacity-is-transient`. Exact sentence plus `$` survived (acceptable). **Widening to `^ERROR:` survived** (finding 1).
- Findings, all NON-BLOCKING:
  1. No arm covers a column-0 codex `ERROR:` line that is not about capacity. Add one.
  2. The comment's "twice at column 0" comes from the ticket and was not measured. Cite the ticket.
  3. The test fixture puts the line at column 0 in a file a codex controller might read.

## Controller delta: `72fc66375`

- New arm `codex-other-error-not-transient`.
- The fixture is assembled with `$'\n'`, so the repo has no column-0 copy.
- The comment now cites the ticket.
- The changelog names the third case.

## Round 2, on `72fc66375` (resumed via SendMessage): PASS 94/100

- Claims verified first-party: scope is 3 files; `make check-changelog` rc 0; capacity suite 19/0. The judge re-ran `make test-launchd-drivers` itself: 52/52, rc 0. `git ls-files` has no column-0 hit.
- Mutants against the new head, each killed by exactly one arm:
  - no anchor: `prose-codex-capacity-not-transient`
  - too narrow: `codex-capacity-is-transient`
  - `|^ERROR:`: `codex-other-error-not-transient`
- Remaining findings, NON-BLOCKING:
  1. One comment line was over 150 characters. Re-wrapped in `edb099371`; comment-only, capacity suite 19/0 again.
  2. The real capacity emission is still not directly measured. It is documented as "per ticket", which is honest.

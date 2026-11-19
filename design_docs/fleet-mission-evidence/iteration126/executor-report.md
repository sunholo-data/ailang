# Iteration 126 executor report

Commit: ad2d4196edf3dee46ef8a479891c1ab966a1c51c (branch fleet/i126-idle-fp-design-docs-verification, not pushed)

Diff stat (git show --stat HEAD):
    .../2026-11-19-idle-fingerprint-design-docs-verification.md | 10 ++++++++++
    tools/launchd/lib/idle-fingerprint.sh                       |  4 +++-
    tools/launchd/test_mission_idle_fingerprint.sh              | 13 ++++++++++++-
    3 files changed, 25 insertions(+), 2 deletions(-)

## Checks (run from worktree root, unpiped)

| # | Command | rc |
|---|---------|----|
| 1 | /bin/bash tools/launchd/test_mission_idle_fingerprint.sh | 0 |
| 2 | /bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations | 0 |
| 3 | make test-launchd-drivers | 0 (all 59 suites passed; bash 3.2 syntax OK) |
| 4 | shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh | 0 |
| 5 | make check-changelog | 0 |
| 6 | real-world repro, world origin/dev (see below) | 0 / 0 |
| 7 | git diff --name-only origin/dev | 0 (lists only the 3 in-scope files) |

Check 1 output:
    PASS driver arms, slot layout, stdin EOF, state seam and placement
    PASS fire-start snapshot: argv, stdin EOF, stale cleared, failures leave none, exclusions, placement
    PASS helper sections, exclusions, failed reads, branch choice, atomic record and refusals

## Check 2: MUTATION lines (verbatim)
MUTATION excludes KILLED: FAIL record-only design doc commit moved hash
MUTATION exclude-root-verification KILLED: FAIL record-only verification commit moved hash
MUTATION exclude-dd-verification KILLED: FAIL record-only design_docs/verification commit moved hash
MUTATION exclude-dd-broad KILLED: FAIL non-iter design_docs/verification commit did not move hash
MUTATION wrong-record KILLED: FAIL unchanged: FELLTHROUGH CALLS=0 kept=yes rows=0 
MUTATION keep-changed KILLED: FAIL changed: FELLTHROUGH CALLS=1 kept=yes rows=0 |world: inputs changed since the last no-pick fire (0000000000000000000000000000000000000000000000000000000000000001) — proceeding with a full fire
MUTATION fail-closed KILLED: FAIL failed read error: EXIT 0 CALLS=1 kept=no rows=0 
MUTATION age KILLED: FAIL expired (1795041156): EXIT 0 CALLS=1 kept=yes rows=1 |world: nothing changed since the last no-pick fire (0000000000000000000000000000000000000000000000000000000000000000) — idle, nothing spent
MUTATION limit KILLED: FAIL failed read accepted rc=0: 7f8dda3d9a889bfb4213e02ac820d189bf4eeffb6b90ae13b58b98d27292ae17 head=8cacb4c ledger=1 open=0 directives=0 inbox=1000 red=0
MUTATION pending KILLED: FAIL green/pending null moved hash
MUTATION store-pin KILLED: FAIL good helper: idle-fingerprint: inbox read failed: store not pinned: local
MUTATION dry-run KILLED: FAIL dry run touched the idle record: EXIT 0 CALLS=1 kept=yes rows=1 |world: nothing changed since the last no-pick fire (0000000000000000000000000000000000000000000000000000000000000000) — idle, nothing spent
MUTATION fire-check KILLED: FAIL another fire recorded this snapshot
MUTATION record-as-snapshot KILLED: FAIL record without a snapshot
MUTATION snapshot-keeps-stale KILLED: FAIL stale pending not cleared before the snapshot

The four new/target mutants match the plan's section 2b table exactly.

## Real-world repro (check 6), world origin/dev, read-only, no fetch
Command 1 (old pathspecs, record-only commits count):
    a15551c2 record: iteration 415 — no pick, rule (d) block report (sixth); row 264 prediction confirmed, row 263 count 4
Command 2 (new pathspecs, with design_docs/verification/world-iter* excluded):
    0e9e8d35 test(row 243 V4 S2b M3): the operator judges write nothing
The second result is a product commit, not a record: commit. origin/dev matched the plan at plan time.

## Deviations
- None to the edits or the mutant table. The commit body is two prose lines plus the ticket signature line.
- Check 3 output was large; its rc lines were captured from the persisted tool output. The FAIL-matching lines in that output are test names or "FAIL=0" summaries, not failures.
- The plan file and this report under design_docs/fleet-mission-evidence/iteration126/ are untracked and not committed.

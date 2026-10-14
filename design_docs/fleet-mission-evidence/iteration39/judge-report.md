# Fleet i39 evaluation: M-FLEET-SLOT-UNCLASSIFIED (commit 81df7d609)

Score: 86/100. PASS (threshold 70).

## Breakdown
- Correctness and spec conformance: 36/40. The classifier matches Q4. Deduction for the empty-label rc=0 case (finding N1).
- Tests and mutation resistance: 22/25. 8 of 10 drills killed; 1 survived; 1 is equivalent. Parent driver turns 6 new arms red.
- Safety and side effects: 10/10.
- Bash 3.2 and scope discipline: 10/10.
- Docs, changelog and downstream readers: 8/15. R9 is shrunk to one line; the Q4/Q6 prose still says UNCLASSIFIED without noting the "at=<label>" form. No consumer breaks (see 6).

## Checks run
1. Suite at HEAD: `env -i HOME=$HOME PATH=$PATH /bin/bash tools/launchd/test_mission_heartbeat.sh` gives rc=0, 36 numbered arms plus a summary line (37 PASS lines), 0 FAIL.
   - Scratch copy with parent driver (81df7d609~1) and the new test: rc=1, 6 arms failed (pass=30).
   - Parent driver with parent test: rc=0, 26 pass. So the new arms are absent at the parent and red against the old driver.
2. Mutation drills ran on scratch copies of tools/launchd plus .claude/skills/mission-control. The worktree was not touched.
3. Spec conformance:
   - Empty label at rc=0 is DIED-PRE-GATE-0, via the `0:fired|0:` arm. Matches "only fired".
   - `fired` at rc=0 is DIED-PRE-GATE-0.
   - rc=0 with gate-0/1/2/3/3b/4/5 is REAPED.
   - rc=0 with any other label (including `gate-bogus`) is UNCLASSIFIED at=<label>. This is the Q4 "anything else" row.
   - rc 143/137 is KILLED with any label. KILLED sits before the `*:*` arm, so UNCLASSIFIED does not eat it.
   - rc other than 0 with an unknown label is CRASHED, which is the "other nonzero" row.
   - MC_PAUSED overrides all of these. Missing heartbeat file is HEARTBEAT-MISSING (else branch, unchanged).
   - Writer enum (mission-heartbeat.sh:12) is fired|gate-0..5|gate-3b|complete|abort. Every one maps to a non-UNCLASSIFIED verdict at rc=0. No regression: the writer cannot emit anything that now lands in UNCLASSIFIED.
4. Side effects:
   - `git show --stat` lists 4 files: driver (+3/-2 lines in the two blocks), test, design doc, changelog fragment. Nothing else in the driver changed.
   - New tests use `mktemp` dirs only. `_mc_bounded` is a recording stub (`printf >> $calls`), so there is no real `ailang messages send` or `gh` call.
5. Bash 3.2: the suite passes under /bin/bash 3.2.57. The diff has no `[[`, `declare -A` or `${v,,}`.
6. Consumers of verdict strings:
   - The history log row is `verdict=UNCLASSIFIED_at=<label>`, with spaces turned into underscores by the existing `tr`. The test asserts this keeps NF==7.
   - No code consumer parses verdict values. `cmd/ailang/mission_ticket.go` takes the line as an opaque string. The `gate-2-pick.md` skill only copies a line into a ticket. The plist and `mission-base.sh` mention the log only in comments.
   - `test_controller_capacity.sh` greps CRASHED/KILLED/COMPLETED/PAUSED prefixes. I did not run it, because it executes the whole driver. Its expected verdicts do not use labels that moved arms.

## Mutation table (all on scratch copies)
| Mutant | Result | rc |
|---|---|---|
| M1 UNCLASSIFIED arm also sets RC=7 | killed: 4 arms red | 1 |
| M2 `0:*` changed to `0:?*` | SURVIVED, but equivalent: the empty label is already caught earlier by `0:fired\|0:` | 0 |
| M3 drop `0:` from `0:fired\|0:` | SURVIVED, no arm covers an empty last label at rc=0 | 0 |
| M4 remove `0:gate-3b` from the REAPED list | killed: "rc=0 last=gate-3b => REAPED" | 1 |
| M5 remove `UNCLASSIFIED*` from the notify case | killed: 3 phase-2 arms | 1 |
| M6 UNCLASSIFIED arm changed to `*:*` (eats rc!=0) | killed: 4 arms (KILLED and CRASHED controls) | 1 |
| M7 delete the PAUSED override | killed: "paused rc=0 last=bogus" | 1 |
| M8 REAPED arm changed to `0:gate-*` (old behaviour) | killed: "gate-bogus => UNCLASSIFIED" | 1 |
| M9 verdict text changed to `UNCLASSIFIED_at=` | killed: 3 arms | 1 |
| M10 separate episode file for UNCLASSIFIED | killed: 3 phase-2 arms | 1 |

## Blocking findings
None.

## Non-blocking findings
1. The empty-label rc=0 path (`0:` -> DIED-PRE-GATE-0) has no test. M3 survives. This is the one non-equivalent survivor. It is reachable if the heartbeat file exists but its last line has an empty label field (a truncated or empty file). The behaviour matches the spec "only fired", but it is unpinned.
2. The Q4 table's last row and the Q6 prose were not updated to say the verdict is emitted as `UNCLASSIFIED at=<label>` and only for rc=0. Only the R9 line was rewritten. This is a docs gap, not a code defect.
3. A non-zero rc with an unknown label reads CRASHED. That is consistent with the table's "other nonzero" row, but it means UNCLASSIFIED only ever signals "exited 0 with a bad label". The design doc should say so.
4. I did not run test_controller_capacity.sh, because it drives the full driver. I judged its verdict expectations by reading them, not by running them.

## Round 2 (commit e9783fae6)
- Commit changes only the test file (+3 lines) and the design doc Q4 table (2 rows). The driver is unchanged from the previous round.
- Full suite on an `git archive` export of e9783fae6: see the result below.
- M3 re-run (drop `0:` from `0:fired|0:`): see the result below.
- Q4 table edits checked against the driver:
  - The REAPED row lists gate-0..5 and gate-3b. This matches the driver arm.
  - The last row says rc=0 with a label outside the closed set gives `UNCLASSIFIED at=<label>`, and that non-zero rc stays CRASHED/KILLED. This matches the driver.
- N1, N2 and N3 are closed. N4 (test_controller_capacity.sh not run) is unchanged.
- Results: the full suite at e9783fae6 gives rc=0, 38 PASS lines (37 numbered arms; the 38th is the summary line), 0 not ok. The M3 mutant gives rc=1, and the only red arm is "rc=0 empty last label => DIED-PRE-GATE-0". Updated score: 91/100, PASS.

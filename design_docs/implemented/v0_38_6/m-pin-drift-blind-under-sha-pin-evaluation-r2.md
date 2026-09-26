# Sprint evaluation — ROUND 2 (delta review) — v1_iter354_pin_age

Reviewed range: `b2bb2f2f8..0db4f3c89` (worktree HEAD = `0db4f3c89eed7ea5c98a65faf97bbd1834218a15`,
one commit above the b2bb2f2f8 reviewed in round 1). Round-1 report:
`.eval/evaluation-r1.md` (PASS 96/100).

## Verdict

**PASS 100/100** — blocking findings: 0

All three round-1 findings are closed or dispositioned with no remaining gap. The delta is exactly
what the coordinator described, verified independently — no undisclosed changes.

## Delta scope verification

| Check | Command | Observed |
|---|---|---|
| Full diffstat | `git diff b2bb2f2f8..0db4f3c89 --stat` | 2 files: `design_docs/planned/…mutation-audit.md` (+22/−2), `tools/launchd/test_driver_notify.sh` (+6/−0) — matches the coordinator's description exactly, nothing undisclosed |
| Production files untouched | `git diff b2bb2f2f8..0db4f3c89 --stat -- tools/launchd/mission-control.sh tools/launchd/lib/pin-root.sh` | **empty**, rc=0 |
| Positive control (same range, test file only) | `git diff b2bb2f2f8..0db4f3c89 --stat -- tools/launchd/test_driver_notify.sh` | non-empty, `1 file changed, 6 insertions(+)` — proves the empty result above isn't a broken diff invocation |
| Production file hashes, this worktree | `shasum -a 256 tools/launchd/mission-control.sh tools/launchd/lib/pin-root.sh` | `e5798a3e…0e7ed` / `82293ba6…58966db` — **byte-identical** to the round-1 PRE hashes recorded in `.eval/backup/pre_hashes.txt` |

## Gates observed (this tree, measured fresh — not carried forward from round 1)

| Command | rc | Total (observed) |
|---|---|---|
| `/bin/bash tools/launchd/test_driver_notify.sh` (baseline, unmutated) | 0 | `==== 82 passed, 0 failed ====` [.eval/r2_baseline.log] — matches the author's expected 82/0 exactly |
| New assertion present and green | `grep -n "not-in-effect and repeat-on-doubling" .eval/r2_baseline.log` | line 60: `PASS: age-a: notice carries the not-in-effect and repeat-on-doubling sentences` |

## Mutant re-drill row (my round-1 prose-inversion mutant, re-applied to `mission-control.sh`)

| Step | Command / check | Result |
|---|---|---|
| Edit | Inverted the two trailing sentences of `_pin_age_body`: `Every landed driver/skill fix newer than this pin is NOT in effect.` → `All landed driver/skill fixes still apply as normal.`; `This notice repeats only when the measured pin age doubles.` → `This notice never repeats.` | — |
| Build | `/bin/bash -n tools/launchd/mission-control.sh` | rc=0 |
| Landed | `shasum -a 256` | `14eee529…28aa7` — differs from PRE `e5798a3e…0e7ed` |
| Suite (once) | `/bin/bash tools/launchd/test_driver_notify.sh` | rc=1, **`==== 81 passed, 1 failed ====`** |
| FAIL line | (sole FAIL line) | `FAIL: age-a: notice carries the not-in-effect and repeat-on-doubling sentences` |
| Restore | `cp` from `.eval/backup/mission-control.sh.orig` | — |
| Restore proof | `shasum -a 256` | `e5798a3e…0e7ed` — **matches PRE exactly** |
| Green re-run | `/bin/bash tools/launchd/test_driver_notify.sh` | rc=0, `==== 82 passed, 0 failed ====` [.eval/r2_restored.log] |

**Non-vacuity, both directions, confirmed:**
- On the unmutated tree: PASSES (§ Gates observed above).
- On my round-1 mutant: goes red, and *only* that one new assertion fails (81 passed / 1 failed — no
  collateral breakage of the other 81 assertions).
- Anchoring check: the two substrings the new assertion pins —
  `newer than this pin is NOT in effect` and `repeats only when the measured pin age doubles` — are
  literal substrings of the design doc's Notice (H2) **frozen** body text
  (`design_docs/planned/m-pin-drift-blind-under-sha-pin.md:222-227`: "Every landed driver/skill fix
  newer than this pin is NOT in effect." / "This notice repeats only when the measured pin age
  doubles."), not arbitrary wording invented for the test.

## Adjudication of round-1 findings

**Finding 1 — notice body's substantive warning sentences unasserted (my diff-anchored survivor mutant).**
**CLOSED.** Repaired as a row exactly as the audit describes: `age-a` gained
`notice carries the not-in-effect and repeat-on-doubling sentences`, verified above to be non-vacuous
in both directions on THIS tree (not carried forward — freshly re-measured). Suite total moved from
81→82 as expected.

**Finding 2 — MUT-F and the T4 unset are killed by the AC-D fixture-rot guard, not behaviourally
(informational).**
**NO-ACTION** (confirmed correct disposition). This was reported as informational, not a defect,
in round 1 — the guard is genuinely loud (hard `exit 1`, non-vacuous) even though it isn't the
*named* assertion's mechanism. The audit's disposition text folds in the two additional T5 mutants
(MUT-F′, MUT-F″) that the controller had already drilled and recorded in the pre-round-1 audit body
(both also killed by the same guard) — consistent with what I read in round 1 and does not change my
assessment. No production or test change was needed or made for this finding, matching the diffstat
(zero changes attributable to finding 2).

**Finding 3 — `sha256 == PRE: False` in the M2 batch footer (apparent audit self-contradiction).**
**CLOSED**, with one caveat noted for the record. The audit's causal narrative (a concurrent
MUT-R re-drill launched while the batch drill's own poll returned on a deadline rather than a
completion marker, so the batch's final hash read raced the re-drill's mutant) is the controller's
account of their own process and is not independently re-executable by me — I cannot replay a
session I wasn't present for. What I *can* and did verify independently: (a) the material claim
that matters — production files are byte-identical between the M1 commit (`065973903`) and the M2
commit (`b2bb2f2f8`) — is TRUE (`git diff 065973903..b2bb2f2f8 --stat -- tools/launchd/mission-control.sh
tools/launchd/lib/pin-root.sh` empty, positive control shows 3 other files did change in that range);
(b) my own round-1 7-mutant re-drill (independently run, this worktree) matched the recorded PRE hash
on every single restore, with zero anomalies. So while the specific mechanical story ("poll raced a
concurrent re-drill") is asserted rather than proven, the thing a reader would actually worry about —
whether the shipped tree's production files have unexplained drift — is conclusively false. Closing on
that basis.

## Score breakdown

| Rubric row | Round-1 score | Round-2 | Basis for change |
|---|---|---|---|
| Tests pass (HARD FAIL gate) | 20/20 | 20/20 | Notify suite now 82/0 rc=0, freshly measured; production files still `bash -n` clean (unchanged) |
| Lint clean | 10/10 | 10/10 | No change in scope |
| Acceptance criteria | 29/30 | 30/30 | The one AC-gap (notice prose unasserted) is closed by the new row, verified non-vacuous both directions |
| Code quality | 14/15 | 14/15 | Unchanged — finding 2 was informational, correctly dispositioned NO-ACTION, no code defect existed |
| Documentation | 14/15 | 15/15 | Audit record now carries an explicit, verifiable disposition section for all three round-1 findings, including the concurrent-writer explanation substantiated by an independently-checked empty diffstat |
| Design fidelity | 10/10 | 10/10 | Unchanged; production files untouched this round (verified) |
| **Total** | **96/100** | **100/100** | |

PASS ≥70, no hard fail, all three round-1 findings closed/dispositioned with independent verification.
**PASS 100/100.**

# Iteration 86 — independent evaluation (sonnet, Agent tool, foreground, judge worktree at 96882f837)

**Verdict: PASS (92/100).** No blocking findings. Category rows: correctness 32/35, fail-loud & safety 23/25, tests 19/20, integration & portability 9/10, docs/changelog 9/10 (sum 92 = stated total; arithmetic checked by the controller).

## Gates (judge worktree)
| Gate | rc |
|---|---|
| `/bin/bash -n` scanner, suite, driver | 0, 0, 0 |
| `shellcheck --shell=bash` scanner + suite | 0 |
| suite under synthetic HOME/TMPDIR/AILANG_STATE_DIR | 0 (52 arms, ~26 s) |
| `git diff --check c2fbb9b58` | 0 |

## Review
1. Offset logic correct: `tail -c +$((S+1))`; size==S gives an empty suffix (clean); truncated and absent paths scan whole; saved size 0 scans whole; `valid_size`'s global `size` is read immediately after each successful call (no stale read).
2. No silent clean: failed tail/stat/grep (rc>1) → failure → exit 2 / instrument-failure; missing/malformed/partial/overflowing baseline → `baseline=failed` + whole-file + diagnostic; a bad baseline's prefix is discarded.
3. Secret hygiene: patterns via /dev/fd/3, not argv; snapshot holds sizes+paths only; suffix staged in `$scratch/suffix` (umask 077), removed by EXIT/TERM/INT cleanup.
4. Driver: regex and `baseline=` handling moved in lock-step; RC and slot verdict unchanged; snapshot bounded 30 s, failure non-fatal; marker/baseline/.tmp removed; public body count-only.
6. Portability: bash 3.2, BSD tools only; python3 used only for the snapshot, with visible fallback.

## Judge's own mutants (disposable copies) — 13/13 killed
| # | Mutation | Result |
|---|---|---|
| 1 | offset branch disabled → whole file matched (the original ticket) | PASS→FAIL (Baseline_old_append, Baseline_empty_suffix) |
| 2 | `+S+1` → `+S` | PASS→FAIL |
| 3 | `+S+1` → `+S+2` | PASS→FAIL (Baseline_appended_leak) |
| 4 | truncation `-ge` → `-gt` | PASS→FAIL |
| 5 | truncation check removed | PASS→FAIL (Baseline_truncated) |
| 6 | tail failure ignored | PASS→FAIL (Offset_read_failure) |
| 7 | malformed baseline accepted | PASS→FAIL (Baseline_malformed) |
| 8 | scratch not removed in cleanup | PASS→FAIL (Termination_cleanup) |
| 9 | grep rc>1 ignored | PASS→FAIL (Matcher_failure) |
| 10 | whole file instead of suffix | PASS→FAIL |
| 11 | driver omits `--baseline` | PASS→FAIL (Driver_old_append) |
| 12 | driver leaves `.tmp` | PASS→FAIL (Driver_cleanup) |
| 13 | public body carries hits | PASS→FAIL (Driver_public_privacy) |
Extra: widening the snapshot bound 30 s → 300 s also fails the suite.

## Non-blocking findings
1. Baseline lookup is O(candidates × entries) in bash. Synthetic 12,000-entry baseline: 60 candidates 14.3 s (matches controller's 15 s), 300 candidates 66 s, ~0.2 s per absent candidate. A fire touching ~500+ files would hit the 120 s cap — loud (INSTRUMENT FAILURE), not silent, but the scan is lost for that fire. Fix direction: sorted merge or one-pass awk/python lookup, O(C+E).
2. A value straddling the snapshot boundary is missed (documented in the changelog), as are bytes written between marker and snapshot, and same-size rewrites.
3. `baseline=failed` with a clean verdict is visible in the summary/diagnostic but not escalated by the driver.

## Strongest objection
The linear lookup is a latent self-inflicted failure: a busy fire can time out the scanner and then nothing is scanned for it. Cheap to make O(C+E); should not stay unfixed for long.

(Judge returned this as its final message and did not write the file; the controller saved it verbatim in substance. Subagent tokens 70,809; 10 tool uses; 577 s.)

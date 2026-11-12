# Judge round 2: fleet i109 (head 0d0d73a4a, base 4b9fda4bf)

SCORE 91/100 — VERDICT PASS. Blocking findings: none.

## Re-measured
- New suite, unpiped, twice: rc 0, passed=18 failed=0 (80s, 79s; load ~4). No flakes. Also ran 8 mutants concurrently (heavy load) and the unmutated-arm results stayed deterministic in those runs.
- Five existing runner suites once: all rc 0, 0 FAIL lines (67/13/8/13/76s).
- git diff --check clean; scope still only allowed paths; runner 556 lines, lib 130; no leaked `sleep 3127` fixtures after drills.

## Mutation drills (scratch copies /tmp/fleet109/judge/n*, driver mut2.sh)
| Mutant | Result | Killing arm |
|---|---|---|
| F drop `b[root]==rootb` (round-1 survivor) | KILLED, 17/18 | U6 only |
| C drop "snapshot lists $$" (round-1 survivor) | KILLED, 17/18 | U7 only |
| B drop pre-TERM birth recheck (round-1 survivor) | KILLED, 16/18 | U8 and U9 |
| G drop pre-KILL birth recheck (round-1 survivor) | KILLED, 16/18 | U8 and U9 |
| X (new) drop trailing-blank trim in `_pr_exe` | KILLED | U12, R1 |
| Q (new) KILL pass never fires on non-TERMed (grace-born) members | KILLED | U5 |
| R (new) drop the 120-char cap in `_pr_exe` | SURVIVES, rc 0 | none (cap untested) |
| S (new) post-TERM WARNING also when nothing was TERMed | SURVIVES, rc 0 | none (cosmetic double warning) |

## Determinism of U8/U9
Both override `_pr_birth` as a shell function. U9 keys on a call counter in a file (works across `$( )` subshells): call 1 for the target pid is the TERM-pass recheck (real), call 2 is the KILL-pass recheck (bogus). That ordering is real and load-independent: no sleeps, bounded gone/alive checks, 10 s deadlines. It is not timing-dependent. It IS order-dependent, and the dependence is not isolating: B and G each turn BOTH U8 and U9 red, so neither arm discriminates one guard alone; the pair does. Direction of failure risk: if a future refactor adds a `_pr_birth` call for the same pid before the TERM recheck, U9's bogus birth would land in the TERM pass, the target would never reach the KILL pass, and U9 would pass vacuously for mutant G (U8 would still catch G via the post-grace KILL, so G stays covered). Non-blocking.
U10/U11 depend on the Nth `ps -A` call (3rd fails in U11); same class of coupling, fails loudly if the count shifts.

## Round-1 guarantees preserved?
- Safety invariants now each have a killing arm (F, C, B, G above). Closure from live ledger members, append-only ledger, `bad` set: unchanged. U1-U3 pass.
- Removed post-grace `_pr_ledger_update`: argued dead in round 1 (mutant 7); round 2 deletes it and U5 still passes via closure from the live parent. No coverage lost: a child whose parent died during grace was not catchable by the refresh either (never seen attached). Q confirms U5 still guards the KILL of a grace-born child.
- Failed snapshot is now loud (two distinct WARNINGs; U10/U11 prove nothing is signalled on a failed first read and the TERMed fixture is still reaped).
- Verdict JSON: R0, R1, R2, R3 each still assert `jq -e .` and unchanged verdict/rc. `_pr_exe` strips tabs/newlines and trailing blanks; U4 still round-trips hostile quote/backslash/non-ASCII through `_pr_report_json`. jq-failure fallback in the runner is unchanged from round 1 (I verified it valid then); still has no suite arm.
- Privacy: argv no longer stored (U12, R1 assert it). `comm=` on darwin may show a path; cap 120 untested (R).

## Non-blocking findings
1. Cap of 120 chars (mutant R) and the double-WARNING variant (S) have no arm.
2. jq-failure fallback still untested in the suite (round-1 finding 4, not addressed; fallback behaves correctly, verified last round).
3. U8/U9/U11 rely on call order of `_pr_birth` / `ps -A`.
4. Residuals R1 (double-fork between polls, runs shorter than one poll) and R4 (no Linux leg) stand, as documented.

## Score
Tests 20/20, lint 8/10, acceptance 28/30, code quality 13/15, docs 13/15, design fidelity 9/10 = 91.

## Strongest objection
U8 and U9 are coupled to the exact count and order of `_pr_birth` calls rather than to behavior, and no single arm isolates the pre-TERM guard from the pre-KILL guard; a harmless refactor of the call sequence could make U9 pass vacuously while still looking green.

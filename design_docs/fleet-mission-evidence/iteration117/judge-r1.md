# Iteration 117 evaluator report (judge r1) — verdict PASS, 88/100

Worktree .wt-fleet-iter117-judge at 34176f228 (base 22a72e7c5); /bin/bash = GNU bash 3.2.57. Tree left clean.

## Mechanism, derived myself
- T (timer arm): the suite has `trap ... EXIT` at line 7; measure_fork_rate does `sleep & timer=$!; kill -TERM $timer`. My standalone loop under /bin/bash 3.2 (150 iterations: bg sleep, immediate kill, wait, count rc 0):
  EXIT trap + TERM: 2/150 and 1/150 lost (sleep ran to completion, rc 0); NO trap + TERM: 0/150; EXIT trap + KILL: 0/150.
  This agrees with the plan: a TERM sent between fork and exec in a trap-bearing bash 3.2 is lost, and `wait` returns 0, which gives exactly rc=0 / refusal=0 / derived=1. SIGKILL cannot be lost; wait then gives 137, which is nonzero, so 73 still follows. The fix addresses the actual cause.
- R (lsof stub): the old stub did `[[ -f ]]` then `cat`, and sent every lane that was not "treatment" to the OpenRouter peer. The driver stub creates the lane file before writing (`echo > file`) and removes it at exit, so a sample can read empty (or lose the file after the -f check). An empty lane then falls into the 203.0.113.8:443 branch. That explains both signatures: a treatment lane showing the OR endpoint, and a refusing arm that "unexpectedly succeeded", presumably because a control sample was invented in a lane that should have been void. I agree with the plan. I did not reproduce R naturally, and nobody did. It is forced deterministically by the new arm.

## Does the fix weaken any arm?
- Timer arm: it still proves "timer exited nonzero => rc 73 with the message". The injection is now a KILL (rc 137), plus the arm runs under ignored TERM. The only thing lost is that SIGTERM-style termination is not exercised, which was never what the probe depends on (the probe code only checks trc != 0). Not weakened.
- Fail-closed lsof: `cat` failure or an unknown lane gives exit 1 with no output. Probe line 224 does `lsof ... || true`, so this equals "no peers sampled this tick". Treatment and control outputs are unchanged and still exercised (the m13 swap mutation fails the live success arm). What could be hidden: a probe defect that depends on a sample taken during a lane-file-less moment, which no arm should depend on. A driver stub writing a lane other than treatment/control would formerly have shown as an OR peer and now shows as none; no arm uses such a lane (grep: lane comes only from --models treatment|control). Accepted residual risk, non-blocking.

## Mutations (my own, on copies, run `PROBE_UNDER_TEST=$PWD/tools/eval/motoko_connection_probe.sh /bin/bash <copy>`; every copy cmp-differed)
| id | mutation | rc | result |
| m1 | KILL -> TERM (wrapper kept) | 1 | KILLED: `not ok - bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal=0 derived=1)` |
| m4 | stub `*)` -> OpenRouter peer (old behaviour) | 1 | KILLED: `not ok - a lane file read before the driver writes it samples no peer failed` |
| m5 | remove ack touch in lsof | 1 | KILLED (same arm) |
| m6 | remove hold-until-ack in ailang-stub | 1 | KILLED: `empty lane-file window reached in 0 lanes, expected 2` |
| m13 | treatment peer -> OR endpoint | 1 | KILLED: `hermetic live success path completes failed` |
| m7 | `*)` -> peer="" (header + empty-peer line) | 0 | SURVIVES, equivalent (no "->" token, so no peer) |
| m9 | unreadable/missing lane file -> `lane=control` | 0 | SURVIVES, REAL GAP: the window W2 (file removed after the check, `cat` fails) is not exercised by any arm; only the W1 empty-file window is forced. Reverting the `|| exit 1` would pass. |
| m12 | wrapper `trap "" TERM` -> `trap "" INT` (KILL kept) | 0 | SURVIVES: the ignore is not independently asserted. It only has teeth if someone reverts to TERM (m1 proves the wrapper works). Not a defect, but the guard protects only the revert direction. |
| m11 | drop `term=ignored` for timer | 0 | SURVIVES: same guard-direction point as m12. |
| m10 | TERM then KILL | 0 | SURVIVES, equivalent (KILL still fires). |
| m8 | ack threshold `!= 2` -> `< 1` | 0 | SURVIVES: weakens "both lanes" to "any lane"; trivially so, since both are reached. The proof that both lanes reach the window is therefore only as strong as the exact `!= 2` literal. |

## Full suite
Two full runs with the unmodified file, /bin/bash, sequential: rc=0 both, "PASS: 61 probe self-test arms ran", 0 `not ok`. The new arms are 33 and 59. (The run of m8 with a scale-drift note also passed.)

## Bash 3.2 / scope
Added code uses only `[[ ]]`, `case`, `$(( ))`, `: >`, `trap "" TERM`, and `exec "$@"`. It uses no `declare -A`, no `${v,,}`, and no mapfile. The suite ran under /bin/bash 3.2.57. The only tools/eval file changed is test_motoko_connection_probe.sh (the other 3 files are changelog and evidence docs). The probe itself was not changed and does not need it: the defects are in the test's TERM injection and stub.

## Changelog fragment
Mostly accurate. "about 1 run in 100" in the opening sentence versus "about 1 run in 160" in the code comment are inconsistent. The mechanism sentence ("a deferred SIGTERM handler ... discarded at exec") is stated as fact; the empirical rates I measured (about 1%) are consistent but the handler internals were inferred, not read from the bash source. The R description "treated an empty lane-file read ... as the OpenRouter lane" is directly supported by the diff. The "unexpectedly succeeded" explanation is an inference, not observed on CI.

## Score (100)
- Correctness vs root cause: 27/30 (T fully confirmed independently; R mechanism strongly supported but never naturally reproduced)
- Test strength / non-vacuity: 22/25 (T guard and W1 guard kill all behaviour-relevant mutants; W2 unguarded, m9)
- Scope and safety: 20/20
- Evidence integrity: 19/25 (changelog rate inconsistency; inferred claims stated as fact; R unreproduced on CI-equivalent load)
Total 88, PASS (threshold 70).

## Blocking findings
None.

## Non-blocking
1. W2 (file vanishes between check and read) is covered only by code, not by any arm (m9 survives). Cheap fix: add a stub env that deletes the lane file before reading, or accept.
2. Fragment: harmonise "1 run in 100" vs "1 in 160"; soften the unmeasured causal statements.
3. m8: the ack threshold is checked with `!= 2`; fine as written.
4. The R failure was never reproduced naturally, so the claim that the stub change fixes the CI red rests on deterministic injection, not on CI-rate evidence; monitor the next ~200 CI runs.

## Strongest objection
The fix for R is proven only for window W1; W2 (the `|| exit 1` on a failed `cat`) is unpinned, so the exact other half of the plan's own two-window mechanism could regress silently.

## Commands run (rc)
git status/log/diff/bash --version (0); cat plan/report (0); sed/grep of the test file and probe (0); full suite x2 (rc=0, rc=0); 11 mutation runs m1,m4,m5,m6,m13 rc=1 and m7,m9,m12,m11,m10,m8 rc=0; standalone loop repros (trap+TERM 2/150, 1/150 lost; notrap 0; KILL 0); some scratch loops were killed (pkill) after my quoting mistakes; rm -rf of my scratch dir; final git status --porcelain empty.

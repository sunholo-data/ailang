# Judge round 1: fleet i109 pi-run orphan reap (head 7dbba8b4d, base 4b9fda4bf)

SCORE 86/100 — VERDICT PASS. Blocking findings: none.

## Score
| Category | Pts | Score | Note |
|---|---|---|---|
| Tests pass | 20 | 20 | new suite 11/11 twice (73s, 72s at load ~10); 5 existing runner suites rc 0 (76/15/10/17/78s) |
| Lint | 10 | 8 | bash -n ok, git diff --check clean, no bash4isms in lib; shellcheck not run |
| Acceptance | 30 | 25 | all 3 ticket shapes covered (R1 godot shape, R2 wall, R3 stall); safety rules 3, 4 and root-birth have no killing arm |
| Code quality | 15 | 12 | post-grace ledger refresh is dead code (mutant 7); silent empty reap on failed snapshot; fallback path untested |
| Docs | 15 | 13 | changelog good, header item 6 good; plan stays in planned/ (fine) |
| Design fidelity | 10 | 8 | plan mutants 1-6,9 per executor; U5 does not test what it claims |
| Total | 100 | 86 | |

## Constraints checked
- Write scope: changed files are only scripts/mission_pi_run.sh, tools/launchd/lib/pi-run-reap.sh, new suite, make/test.mk, changelog, plan. OK. No Go, CI, pi-extensions.
- No default knob, verdict name or rc changed (diff read; existing suites pass unchanged).
- bash 3.2.57: grep for declare -A/mapfile/BASHPID/${,,} found only the header comment. Suites ran under /bin/bash 3.2.57.
- Consumers of reaped_count/reaped: git grep outside design_docs/changelogs/lib/new suite finds none (only unrelated word "reaped" in prose/banked JSON). OK.
- Reaps on every outcome: reap call is unconditional after `wait`; R1 (ok) proves it.
- Per-run ledger: STAGE is mktemp -d, so a sibling runner has its own ledger; closure is from this run's root and its own ledger only.
- Verdict JSON valid on jq-failure path: I forced `_pr_report_json` to return 1 in a scratch copy. Verdicts for r1 (ok) and r2 (wall) stayed valid JSON: `{"verdict":"ok","rc":0,"reaped_count":2,"reaped":[]}`, with the stderr WARNING. OK, but no arm in the suite covers it.
- Safety by inspection (could not construct a live hole): wrong-birth ledger entry skipped (U1); $$, ancestors, pid 1/0 excluded (U2); root and each ledger seed require birth match; ps failure -> ledger untouched, reap TERMs nothing; zombie skipped; birth re-verified before each signal. Only theoretical hole: same-pid same-second recycle (lstart is 1 s resolution) plus TOCTOU between recheck and kill. Accepted.
- Live-run check: after all drills `ps` shows no leaked `sleep 3127` fixtures.

## Mutation drills (scratch copies under /tmp/fleet109/judge/m*, script mut.sh)
Executor's mutants 1-6, 8 killed and 7 SURVIVED per /tmp/fleet109/exec/mut-batch{A,B}.txt (I did not re-run those; I re-ran 4 of my own).
| Mutant | Change | Result |
|---|---|---|
| exec 7 | delete post-grace `_pr_ledger_update` | SURVIVES (suite rc 0, 11/11). Meaning: the line is redundant. The next `_pr_members` call already reaches a child born during grace through closure from the still-live ledger member, so U5 passes on closure, not on the refresh. Either the line is dead code or U5 needs a shape where the refresh matters. Misleading comment/test name. |
| B (mine) | drop pre-TERM birth recheck (rule 3) | SURVIVES, rc 0. TOCTOU recheck is unobservable in the suite. |
| G (mine) | drop pre-KILL birth recheck | SURVIVES, rc 0. Same. |
| C (mine) | drop "snapshot must list $$" check (rule 4) | SURVIVES, rc 0. Plan says each safety rule has a test arm; rule 4 has none. |
| F (mine, not in plan) | drop `b[root]==rootb` (root pid-reuse guard) | SURVIVES, rc 0. If the root pid were recycled by an unrelated process, this mutant would reap that process's whole subtree and no arm notices. This is the single most safety-critical identity check without a test. |
Killing arms for my mutants: none. That is the finding.

## Non-blocking findings
1. Root-birth guard (F) and failed-snapshot rule (C) and both recheck guards are untested. A test: put a live unrelated process's pid as ROOT with a wrong ROOT_BIRTH, call `_pr_members`, assert it and its children are absent. And a `ps` shim that fails, asserting no signal.
2. U5 / mutant 7: remove the redundant refresh or rework the arm (child must be born under a member that dies at TERM, which would still not be seen, so likely just delete the refresh and fix the comment).
3. Failed snapshot at reap time silently reaps nothing and reports reaped_count 0, no stderr warning. Violates the fail-loud principle for an integrity-relevant path (a leak is reported as clean).
4. jq-failure fallback in the runner has no arm. Also if jq is entirely absent, `jq -r .count` fails and REAPED_COUNT becomes 0 even though the TSV has rows (the WARNING text says count is from TSV). Edge case only.
5. Runs shorter than one poll interval (default 10 s) whose pi spawns a background child then exits before the first snapshot are missed (R1 residual class). Stated in the changelog as R1; the header could say "or run shorter than one poll".
6. `reaped[].command` stores up to 200 chars of argv into the verdict; argv with secrets would land in a verdict file.
7. No Linux leg (R4 in plan, acknowledged). `make test-launchd-drivers` not run by me, per instructions.

## Commands and key outputs
- `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_orphan_reap.sh` x2: rc 0 both, passed=11 failed=0. Only noise: "Killed: 9" job lines from the suite's own cleanup. No flakes at load 10.
- scripts/test_mission_pi_run.sh rc 0; test_mission_pi_run_{commits,sandbox,provider_quota,tool_stall}.sh rc 0, 0 FAIL lines each.
- jq-fail forced run: verdict valid, WARNING printed (suite then red by design, 8 pass / 3 fail).
- wc -l: runner 554 (<=555), lib 116 (<=160).

## Single strongest objection
The plan says SAFETY above all and that each safety rule has a test arm, yet the root-pid-reuse guard (`b[root]==rootb`), the failed-snapshot rule and both pre-signal identity rechecks can each be deleted with the suite staying 11/11 green. The code reads correct, but the suite does not defend the property that stops the reaper signalling a stranger on a shared rig.

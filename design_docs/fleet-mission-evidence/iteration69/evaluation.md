# Iteration 69 evaluation: pi controller protocol note (fc646ca25 vs 65253e86e)

SCORE 91/100  VERDICT PASS (no blocking findings)

| Section | Pts | Basis |
|---|---|---|
| Tests pass | 20/20 | test_pi_controller_note.sh rc=0 (28 pass, 0 fail); `make test-launchd-drivers` rc=0 ("all 52 suites passed", bash 3.2 syntax OK) |
| Lint | 10/10 | shellcheck on the 2 new files: rc=0, no findings. Driver findings base 33 vs head 33 (no new) |
| Acceptance criteria | 28/30 | note is true of the gate/exec source (below); reach verified; 2 pts off for semantic mutation gaps |
| Code quality | 13/15 | Lib + extraction-based test are clean. Minor sentence imprecision; test blind to note meaning |
| Documentation | 12/15 | changelog fragment and plan present; no example file needed |
| Design fidelity | 8/10 | deviations disclosed honestly and justified (T8 string uses POLICY_ENV, hermetic env clearing, shellcheck directive) |

## 1. Note correctness (checked against HEAD)
- Chain refusal: session-protocol-gate.ts bashAllowed splits on `&&|\|\||;|\|` and requires every segment to match BASH_ALLOW. TRUE.
- "bash runs only single read-only commands" (note) is slightly loose: an all-allowlisted chain such as `ls && cat x` passes. The next clause corrects it. Non-blocking.
- "read CLAUDE.md with the read tool": prerequisitesMet sets claudeMdRead when a `read` call's path includes "CLAUDE.md". TRUE. The CLAUDE.md read is only mandatory for workspace sunholo-data/ailang (prereqSetForWorkspace); generic repos (stapledon, world) need only some read. Both ~/dev/sunholo-data/stapledons-godot/CLAUDE.md and ailang-world/CLAUDE.md exist, so the instruction is satisfiable everywhere; it over-prescribes harmlessly.
- Inbox: counts only a `bash` call whose command includes "ailang messages" (gate lines ~170-180; ailang_cli is not counted). TRUE. (Unattended sessions skip the inbox step, but controllers are not stages, per the gate's own comment.)
- ailang_cli/ailang_run "every call refuses": ailang-exec.ts:338 refusedResult, called at lines ~357 (ailang_run) and ~459 (ailang_cli) when gate.refusal; refusal built at ~134 when POLICY_ENV unset. TRUE while the policy is unset. If a controller env ever gets a policy, the note becomes false (T8 does not guard that).
- Reading ailang_read/ailang_check etc. are not named; fine.

## 2. Reach and scope
- pi-only: the diff touches only the pi branch of _mc_run_once (mission-control.sh ~2470-2480). Claude arm byte-unchanged (mutation E caught). scripts/mission_pi_run.sh untouched; T7 checks it.
- Resume: `_mc_pi_prompt` is chosen (resume vs fresh) first, then the note is prefixed, so both paths carry it (mutation B caught).
- bash 3.2: `/bin/bash -n` rc=0. `_mc_pi_note` is assigned at line 2474 in the same branch before use at 2475/2480; under `set -uo pipefail` no unset hazard. Command substitution of a printf function, no `${//}` or `read -d`.
- Sourcing: `$MC_DRIVER_ROOT` = repo root of the driver script (line 47), the lib sits beside pi-ext-args.sh in tools/launchd/lib, which is already sourced the same way. Resolves from a pin worktree once the pin contains the commit.

## 3. pi acceptance (installed pi at /opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/dist)
- cli/args.js:64-66 collects repeated --append-system-prompt values.
- core/resource-loader.js:17-31 resolvePromptInput: `existsSync(input)` then read file, else use the string as text. The note is ~800 chars of prose, never an existing path: misread is not possible in practice.
- core/system-prompt.js:10 appends `\n\n${appendSystemPrompt}`; core/agent-session.js:672/1944 rebuild the system prompt from `_baseSystemPrompt` per session start. So on a resumed session the append is rebuilt, not stored in the session file: no duplication in the system role. The -p copy does add one user message per fire to the persisted history (each resume re-states the note); minor token cost, non-blocking.
- No real pi model run was made.

## 4. Mutations (mine, on scratch copies under /tmp/iter69_judge_mut, driver/lib via PI_NOTE_DRIVER/PI_NOTE_ROOT)
| # | Mutation | Result |
|---|---|---|
| control | none | rc=0 |
| A | -p prefix separator `\n\n` to single `\n` | CAUGHT (3 fails) |
| B | resume -p built from fresh $PROMPT | CAUGHT (2 fails) |
| C | --append-system-prompt gets full prompt instead of note | CAUGHT (3 fails) |
| E | claude -p value changed | CAUGHT (1 fail) |
| G | lib sourced via `$(dirname $0)/lib/` instead of $MC_DRIVER_ROOT | CAUGHT (2 fails) |
| F | lib: "are NOT granted" flipped to "are granted" | MISSED (rc=0) |
| F2 | lib: inbox instruction changed to "via the ailang_cli tool" (the exact bug) | MISSED (rc=0) |
| H | lib: chain rule inverted ("allowed if any part is on the list") | MISSED (rc=0) |
(An earlier E attempt that appended an undefined function to the claude arm was an equivalent mutant; discarded and redone.)
The suite pins plumbing and a few keywords (backticked `ailang messages list --unread`, names present, no chain characters in it), not the meaning of the sentences. The T8 tripwire guards the gate/exec sources, not the prose. Non-blocking: prose correctness is by review only.

## 5. Plan conformance and honesty
Deviations as reported are real and reasonable: T8 greps `POLICY_ENV = "AILANG_AGENT_POLICY"` plus the `(${POLICY_ENV} is unset)` template (both present in ailang-exec.ts lines 29 and 134); hermetic env clearing is in the test header; `# shellcheck source=/dev/null` at the driver source site. No other claim found false.

## 6. Does it fix the ticket?
It tells the model the exact satisfying call, in two places, and removes the ailang_cli temptation. It cannot guarantee anything: the note is advisory, and the deadlock mechanism (gate counts only `bash` containing "ailang messages"; ailang_cli refuses with policy unset) is untouched. The stapledon model had a system prompt and a protocol statement before and still chose ailang_cli; one more paragraph reduces, not removes, that. Also the glm-5.3:cloud route's handling of the system role is unmeasured, hence the duplicate -p copy (sound hedge). Residual risk needs the `.pi/extensions/**` change (e.g. making the gate recognise the inbox step via ailang_cli, or not registering ailang_cli without a policy), which stays parked under D-FLEET-3. Recommend a follow-up observation: first live pi controller fire, check the session JSONL for a bash `ailang messages` call.

## Commands
- /bin/bash tools/launchd/test_pi_controller_note.sh: rc=0
- make test-launchd-drivers (unpiped to file): rc=0, 52 suites; test_motoko_connection_probe did not fail
- /bin/bash -n mission-control.sh: rc=0
- shellcheck new files rc=0; driver findings 33 base / 33 head

## Findings
Blocking: none.
Non-blocking: (1) test blind to prose meaning (mutations F, F2, H); (2) "only single read-only commands" slightly loose; (3) note re-sent as a user message on every resume; (4) note becomes false if a controller ever gets AGENT_POLICY (not tripwired).

## Strongest objection
The fix is a prompt that asks the same model that already ignored the protocol to behave; it is untested against a live pi controller, and the deterministic hole (gate does not count ailang_cli inbox calls, tool refuses) remains open. A future model that ignores the note deadlocks again with the same signature. Not blocking as a scoped driver-side mitigation, but the ticket should stay open pending a live fire and the D-FLEET-3 extension fix.

EVALUATION_SCORE: 91/100
EVALUATION_VERDICT: PASS score=91

# Round 2 (9a9c95e58 on fc646ca25)

SCORE 93/100  VERDICT PASS  Blocking findings: none

- Diff: lib drops "single" from "bash runs only single read-only commands"; test adds T6c (five pinned sentences plus a regex refusing `(via|with|through|using|use) (the )?ailang_(cli|run)`).
- Truth check: "bash runs only read-only commands, and a chained command ... is refused if any part is not on the read-only list" matches bashAllowed/BASH_ALLOW in session-protocol-gate.ts (all segments must match the allowlist). The looseness finding is resolved. Rest of the note unchanged, so earlier verification stands.
- `/bin/bash test_pi_controller_note.sh` rc=0 (34 passed, 0 failed). shellcheck on both changed files rc=0.
- Full make test-launchd-drivers not re-run (diff touches only the lib string and one test file).

Mutations on scratch copy /tmp/iter69_judge_mut2 (perl, restored with cmp):
| Mut | Change | Result |
|---|---|---|
| F | NOT granted -> granted | CAUGHT |
| F2 | inbox step -> "via the ailang_cli tool" | CAUGHT |
| H | chain rule inverted | CAUGHT |
| N4 | inbox command -> `ailang messages ack --all` | CAUGHT (T6 backticked command) |
| N1 | "(3) call session_protocol_ack" -> "skip session_protocol_ack" | MISSED |
| N2 | "Do the protocol first, in this order" -> "last, in any order" | MISSED |
| N3 | "AILANG_AGENT_POLICY is unset" -> "is set" | MISSED |
| N5 | append "Prefer to call ailang_cli for the inbox." | MISSED (verb "call" not in the regex) |
| N6 | "read CLAUDE.md with the read tool" -> "bash tool" | MISSED |
| N7 | "every call refuses" -> "every call succeeds" | MISSED |

T6c closes the three round-1 gaps, but it is phrase-pinning: wording outside the five needles and the via/with/through/using/use regex can still flip meaning (N1-N3, N5-N7). N5 and N7 are the most serious (they re-open the original bug). Non-blocking; a regex of `ailang_(cli|run)` outside the one "NOT granted" sentence would close N5, and pinning "every call refuses" would close N7. The advisory-prompt objection from round 1 is unchanged (no live pi fire yet; the gate/extension hole is parked under D-FLEET-3).

EVALUATION_SCORE: 93/100
EVALUATION_VERDICT: PASS score=93

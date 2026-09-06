# M-PI-EVALUATOR-SESSION-HANDSHAKE — satisfy the judge's local protocol before review

**Status**: Planned
**Target**: v0.35.2
**Priority**: P1 (next ready mission queue item)
**Estimated**: 0.5 day, one milestone including review and mutation checks
**Dependencies**: Existing pi session-protocol gate; mission-control Gate 3 routing recipe
**Planner-Lane**: codex-ok
**Created**: 2026-09-06, V1 iteration 340

## Problem Statement

The pi evaluator receives a new session, but controller inbox triage is sometimes described as
already completed on its behalf. The loaded gate instead checks the evaluator's own session
history for a CLAUDE.md read and an `ailang messages` call before the headless
`session_protocol_ack` unlocks writes (V2–V4). Reviewers need writes for their report and for
the mutation checks explicitly required by the evaluator recipe (V7).

Iteration 336 records 446 seconds, 226 completed tool executions, zero changed files and zero
agent-end events before controller termination. The runner returned `empty_worktree`/10 with
pi_rc143; this was a failed transport attempt, not a judge verdict. The same MiniMax model's
subsequent attempt completed in182 seconds with37 tools and a report after an inbox tool
call and local acknowledgement (V1, V12). The preserved transcript sharpens that receipt:
the pre-ack inbox call was denied, the ack still succeeded because the predicate scans calls,
and a subsequent inbox retry timed out after30 seconds. Thus this is evidence of delivered
local history and report completion, **not** a successful canonical inbox check (V12).
Iteration339 again records two timed-out MiniMax routes
without reports and carries that behavior under this backlog item (V8). That later record
does **not** independently establish protocol denials as the cause of both timeouts; this
design remedies the demonstrated prerequisite mismatch, without claiming all timeouts solved.

## Goals

Give every pi evaluator directive a concrete ordered startup handshake compatible with the
existing guard, and explicitly supply canonical messaging settings at the real pi child
launch. Acceptance concerns launch environment and recipe delivery; successful inbox
execution remains a textual recipe obligation the existing guard cannot enforce (V16).
Model success rate remains an observational outcome.

- One canonical evaluator preamble in Gate3's route resource, preserving `MISSION-ROLE` first.
- One bounded same-session inbox listing, a summary without message acknowledgement, and an
  explicit successful protocol acknowledgement before review/report/mutation work.
- Hermetic regression tests that read the real route resource and fail when its handshake is
  removed, reordered or replaced by controller-triage reuse.
- Existing generator≠judge, isolation, fallback and lane-verdict rules remain in force.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Fix the evaluator recipe, without runner prompt injection | Keeps the remedy at the instruction boundary that supplied the bad premise | agent, scoped by controller | design | low |
| Pin two messaging variables at the actual runner child invocation | Ambient settings can otherwise select local or wrong-project stores (V14) | agent, narrow-refinement scope correction | design | low |
| Require this evaluator session's actual read/list/ack sequence | Inherited triage cannot unlock the headless guard | agent, constrained by existing protocol | design | low |
| Launcher supplies canonical messaging env; evaluator uses a bare command | `env …` and assignment prefixes are blocked while the gate is armed | agent, constrained by measured allowlist | design | low |
| Treat a failed handshake as a role transport failure, never an evaluation FAIL/PASS | Preserves independence and prevents a missing report becoming a verdict | agent, existing routing policy | design | low |

### Design Freeze

- [x] Production change is the Gate3 pi evaluator recipe plus exactly two messaging
  environment bindings at `scripts/mission_pi_run.sh`'s actual pi child invocation.
- [x] Initial `MISSION-ROLE: evaluator` remains the first line of the delivered directive.
- [x] The same-session sequence is read → bare bounded list → summarize/classify → protocol ack
  → judge work. Inbox message acknowledgement is a separate operation and remains forbidden.
- [x] Inbox bash arguments include `timeout: 30` seconds; `--limit 1` limits cardinality only.
  Timeout or failed listing stops the attempt before acknowledgement and judge work.
- [x] Canonical store/project are launcher prerequisites; an inability to establish them or
  to complete the handshake is reported through the existing role-failure/fallback path.
- [x] Session gate, runner arguments/verdicts/watchdogs, model routing and judge independence
  are frozen. The two child environment bindings are the sole runner implementation change.
- [x] Failed listing results remain invisible to the guard predicate; successful listing is
  explicitly a recipe-level textual requirement, not a new mechanically enforced property.

These are agent-resolvable operational choices within the queued remedy. Human decision
items D-57/D-58/D-59 are outside this work. Unattended design quorum remains required before
planning; the controller owns that invocation and its recorded verdict.

## Solution Design

### Canonical evaluator recipe

### Actual launch environment

The evaluator route uses `scripts/mission_pi_run.sh` (V6/V7). At line155 the runner currently
executes `pi --mode json --no-session --model "$MODEL"` without messaging bindings. A real
runner invocation with fake pi proves unset settings remain unset, and wrong caller settings
reach the child unchanged (V14). Recipe-only sufficiency is therefore withdrawn.

Change only the pi child invocation to prefix these two literal environment assignments:

```bash
AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
  pi --mode json --no-session --model "$MODEL" < "$DIRECTIVE" 2>"$ERR" |
```

The existing downstream awk pipeline stays as-is. Shell command-scoped bindings set the
actual child environment without requiring the armed model to execute `export` or `env`.
Do not set, unset or rewrite `AILANG_STORAGE`: this change is scoped to messaging only.
The bindings apply to all callers of this shared mission runner, including executor and
evaluator roles, matching CLAUDE.md's canonical store requirement. Caller attempts to select
local/wrong-project messaging through this mission runner intentionally become canonical.
Standalone pi invocations outside this runner are outside this change's guarantee.

The earlier blanket runner freeze is narrowed only for these two bindings; D-58/D-59 runner
verdict/harness work remains parked. This is the bounded reviewer-proposed scope correction,
not role-aware prompt injection or a new runner option.

### Canonical evaluator instructions

Add one clearly delimited `PI EVALUATOR SESSION HANDSHAKE` subsection within the pi provider
recipe in `.claude/skills/mission-control/resources/gate-3-route.md`. Its controller-facing
introduction must require the canonical `mission_pi_run.sh` launch path, which explicitly
supplies `AILANG_MESSAGES_STORE=gcp` and `AILANG_MESSAGES_PROJECT=ailang-multivac` to the pi
child while leaving `AILANG_STORAGE` alone. Do not ask the armed evaluator to
run `export`, an `env` wrapper, or an assignment-prefixed inbox command (V3, V5).

The subsection contains a single copyable text-fenced preamble with these instructions in
this order. The delimiters are test extraction boundaries, not a new runtime interface:

```text
MISSION-ROLE: evaluator
Complete this handshake in YOUR OWN pi session before judge work or report/mutation writes.
1. Use the read tool to read CLAUDE.md completely in the evaluator worktree.
2. Use the bash tool with arguments {"command":"ailang messages list --unread --json --limit 1","timeout":30}. The timeout is 30 seconds (use the harness-equivalent only if its field differs); --limit 1 bounds result cardinality only. Require a successful tool result before continuing.
3. Summarize that bounded inbox result to the controller; classify this task as independent evaluation under the supplied approved design and sprint plan. Do not acknowledge inbox messages.
4. Call the session_protocol_ack tool with {} and verify its result reports acked=true.
5. Only after that success, perform the supplied independent evaluation and write its report.
Controller-provided triage is context, not evidence that this session completed the protocol.
If a prerequisite or acknowledgement fails, including inbox timeout expiry, report the exact missing step or tool error and stop this attempt. This is a role transport failure for the existing fallback route, never a judge verdict. Do not loop on denied writes, remove extensions, or bypass the guard.
```

Append task-specific design/plan paths, reviewed commit, evaluator worktree, test commands,
mutation requirements and report path after this preamble. The controller must not precede
it with its own triage summary: Gate3 requires the role marker at the beginning (V7).
Any supplied controller summary remains contextual information after the preamble.

The one-row list is deliberately a bounded local handshake, not a replacement for the
controller's complete canonical inbox triage. An empty array is a successful zero-message
result and should be summarized as such. `--limit 1` limits result cardinality, not elapsed
time. The bash tool's `timeout: 30` seconds supplies the local wait bound (V12); timeout
expiry is a role transport failure and follows the existing fallback policy. The runner's
default `MAX_SECONDS=1800` and `wall_timeout`/13 are only the outer backstop (V13), not a
substitute for the inbox timeout. A listing error or missing canonical launcher
configuration prevents proceeding. Summarize before `session_protocol_ack`, but do not
confuse that tool with `ailang messages ack`: this design authorizes the former only.

Use the existing lane failure handling if the agent cannot complete the sequence. Record
which prerequisite/ack failed, its error and the actual fallback route. A missing ack tool
is an environment/protocol failure; it is not permission to waive the gate. Do not add a
new runner exit code or retry allowance. `mission_pi_run.sh` keeps forwarding the directive
and reporting its existing typed lane outcome (V6). A written report still requires its
own independent evaluator judgment under the existing sprint-evaluator rules.

### Systemic audit and conflict surface

The audited shared surface has one pi provider recipe and a separate evaluator-isolation
rule (V7). Place the handshake in the provider recipe and add a short cross-reference in
the evaluator paragraph if needed, instead of duplicating the preamble in both locations.

| Surface | Existing behavior / evidence | Decision |
|---|---|---|
| Session gate | Same-session prerequisite scan; headless ack success unlocks tools (V2–V4) | Reuse unchanged |
| Pre-ack bash | Bare `ailang messages` allowed; env/assignment prefixes refused (V3) | Give the already-allowed exact command |
| Runner stdin / verdict | Direct directive forwarding and changed-worktree assertion (V6) | Reuse unchanged; no role-aware prompt assembly |
| Runner child environment | Ambient messaging settings pass through, including wrong ones (V14) | Override exactly two child messaging values; preserve AILANG_STORAGE and every other variable |
| Failed inbox execution | Predicate scans assistant calls and ignores failed tool results (V16) | Keep gate unchanged; successful listing and stopping on failure remain textual recipe obligations |
| Role marker | First-line `MISSION-ROLE` requirement (V7) | Preserve as first preamble line |
| Judge mutations | Separate worktree required (V7) | Preserve; local acknowledgement precedes mutation/report writes |
| Fallback | Nonzero runner outcome routes onward, subject to the existing stream-dead exception (V7) | Reuse existing bounded policy |

This is a recipe, two-variable launch binding and contract-test change. Language grammar, compiler passes, effects and
runtime behavior are outside the edit set; this document makes no language-support claim.

### Files to Modify/Create

- `.claude/skills/mission-control/resources/gate-3-route.md` (+35–55 lines): one canonical
  evaluator handshake, launcher environment prerequisite, exact failure behavior, and at most
  one cross-reference from the evaluator paragraph.
- `.pi/extensions/.session-protocol-gate.test.ts` (+80–130 lines): tests reading the real
  route resource, checking the delimited preamble and verifying its command/trace against
  the existing pure guard exports; add ~60–90lines of actual-runner fake-pi environment
  integration coverage. Dot-prefix discovery convention is already documented
  in the test's header (V9).
- `scripts/mission_pi_run.sh` (+2 lines at the existing pi invocation): command-scoped
  canonical messaging store/project bindings only; preserve the pipeline and all verdict logic.

The design and sprint/evaluation records are normal workflow artifacts. No edit to
the existing shell suite or `.pi/extensions/session-protocol-gate.ts` is part of this
implementation. The runner's only edit is the two launch bindings specified above.

## Acceptance Criteria

- [ ] **AC1 — Delivery contract.** The actual route resource contains exactly one canonical
  delimited evaluator preamble. Its first line is `MISSION-ROLE: evaluator`; read, list,
  summarize/classify, ack and judge steps appear in that order. The test reads the resource
  from a path derived from its own module location, not the caller's CWD.
- [ ] **AC2 — Guard-compatible command.** The command extracted from that actual preamble
  equals `ailang messages list --unread --json --limit 1`; `bashAllowed` and armed
  `shouldBlock` admit it. Assignment-prefixed and `env`-prefixed replacements fail the test.
  Extract its JSON tool arguments and assert numeric `timeout: 30`; omitted timeout and
  changed values fail. The recipe distinguishes cardinality from elapsed time and treats
  timeout expiry as transport failure requiring stop/fallback, never an evaluation verdict.
- [ ] **AC3 — Local evidence and explicit success.** Tests derive the read/list tool history
  array from the extracted steps and call `headlessPrerequisitesMet(branch: unknown[])`,
  asserting the returned object's `met` field is true. This pure function takes an array,
  not a session context; the registered ack tool obtains that array with
  `ctx.sessionManager.getBranch() as unknown[]` (V2). Controller text
  alone and the trace with either step omitted remain false. Assert the preamble requires
  a successful listing result before `session_protocol_ack` with `{}` and `acked=true`
  before judge work **as a source-text contract only**. Add the negative-result control:
  the exact read/list branch followed by an `isError:true` tool result still returns
  `.met=true` (V16). The existing predicate does not distinguish success from failure;
  neither this AC nor the handshake claims to add enforcement of execution success.
- [ ] **AC4 — Authority and failure text.** The recipe instructs launcher-side canonical
  store/project configuration without changing `AILANG_STORAGE`, distinguishes protocol ack
  from inbox-message ack, and requires error reporting/stop rather than repeated denied writes
  or guard bypass. Bounded listing is not represented as full mission triage.
- [ ] **AC5 — Non-vacuity.** In isolated evaluator copies, every named mutation below makes
  the relevant new test fail; restoring the file recovers green. Removing only this
  milestone's recipe addition must fail the new acceptance test while leaving the pre-existing
  guard tests green. Record mutation name, landed diff, command, exit status and red test name.
- [ ] **AC6 — Validation and scope.** All seven pre-existing gate tests and the new tests
  pass; repository-required document/skill/whitespace checks pass. Documentation is updated
  through the canonical route resource. Diff proves the runner changed only at its two
  child environment bindings, and the gate implementation/parked iteration339 candidate
  were not modified by this milestone.
- [ ] **AC7 — Real launch environment.** Invoke the actual checked-out runner with a fake pi
  executable first in PATH and a fresh temporary git workdir. In one arm remove both caller
  messaging variables; in the other supply `local` and `wrong-project`. The fake child must
  report `gcp`/`ailang-multivac` in both arms. With AILANG_STORAGE absent it stays absent;
  with caller AILANG_STORAGE=`local` it stays `local`. Require runner rc0, child invocation
  evidence and the exact environment receipt. See V14 for the real-path baseline failure.

## Testing Strategy

Run `node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts`.
The current baseline is7 passing/0 failing (V9). Add focused contract tests in that file to
reuse its real `bashAllowed`, `shouldBlock` and `headlessPrerequisitesMet` imports. Read the
canonical resource using Node filesystem APIs; fail loudly on missing file, missing/duplicate
delimiters or empty extraction. Do not hardcode a replacement preamble in the test and then
test that substitute: the resource itself is the changed production artifact.

Keep parsing narrow: extract the one text fence within the named subsection, require the
five ordered numbered steps, and parse step2's JSON arguments for the exact inbox command
and numeric `timeout: 30`. Assert the
read instruction names CLAUDE.md, then use those extracted values to build the small
assistant-tool-call history array accepted by the existing pure predicate. The production
ack handler supplies precisely this array shape from `getBranch()`; the test does not pass
a fake context object. The successful-listing-result requirement, summary/classification,
ack success and error-handling instructions are textual contracts; predicate tests do not
claim a model actually obeyed them. Existing `acked=false` blocks and `acked=true` allows
tests remain the control for unlock semantics.

AC7 executes the production shell runner rather than a reimplemented launch function.
Create a temporary fake `pi` executable whose only action writes its three named environment
values into the temporary git workdir and emits an agent_end event. Keep runner output,
directive and verdict outside that workdir so they cannot produce a false changed-worktree
success. Use `MISSION_PI_POLL_SECONDS=1`, `--max-seconds 5 --stall-seconds 5`, and a Node
process timeout of15seconds as the test backstop. The test must clean up its owned fixture
directory after the child has exited. No provider or inbox is contacted. This reuses the
existing Node test file and does not modify or import the parked shell-suite candidate.

Required non-vacuity mutations, one at a time in the judge's isolated tree:

| Mutation | Required detector |
|---|---|
| Remove the new recipe block entirely, keeping every pre-existing line | AC1 extraction/count fails; proves this milestone is necessary |
| Keep handshake words elsewhere but delete the canonical block | AC1 still fails; whole-file grep cannot launder a missing recipe |
| Move `MISSION-ROLE` below startup prose | AC1 first-line fails |
| Swap ack before list, or move judge work before ack | AC1 order fails |
| Replace local list with “reuse controller triage” | AC2 command/AC3 local-history checks fail |
| Prefix the real inbox command with `env …` or a variable assignment | AC2 fails; bare-command positive control passes |
| Remove `--limit 1`, change it to an unbounded list, or remove `--json` | AC2 exact bounded-command contract fails |
| Delete `timeout: 30` from the actual JSON arguments, or change30 to300 | AC2 timeout assertion fails; intact30-second positive control passes |
| Remove the successful-listing prerequisite or permit continuing after timeout | AC3/AC4 successful-result and failure-contract assertions fail |
| Replace required ack success with optional/best-effort acknowledgement | AC3 success requirement fails |
| Remove the inbox no-ack or denied-write stop instruction | AC4 required authority/failure contract fails |
| Remove either new runner messaging binding, or change its canonical literal | AC7 actual child receipt fails for unset/wrong caller arms; unchanged baseline guard/recipe checks remain controls |
| Set AILANG_STORAGE in the runner child invocation | AC7 absent-variable arm fails; caller-local control confirms storage is preserved rather than stripped |

This suite is hermetic: it reads files, exercises pure predicates, and invokes the actual
shell runner with a fake pi executable, without making inbox or model calls. A live
evaluator receipt in this iteration should record the local read/list/
ack sequence and report outcome if the configured pi route runs. An unreachable model is
recorded as routing failure with the configured independent fallback; it does not prove or
disprove model compliance. Only the two messaging child bindings change runner behavior; no benchmark or timeout fix is
claimed by these tests. Controller and evaluator run the checks required by their workflow
skills as well; this design does not waive their aggregate gates.

## Deferred Decisions

- Agent may choose extraction helper names and focused test names, preserving the source-bound
  test contract and each named non-vacuity arm.
- Agent may choose the exact subsection delimiter spelling and prose wrapping; tests and recipe
  must share the observed delimiter without duplicating the whole preamble as a fixture.
- Controller chooses the usual workflow evidence paths and records actual routing outcomes.

## Non-Goals

Changing the session guard's security model; adding runner role flags or injected prompts;
repairing iteration339's parked shell harness; changing model/provider routing, token budgets,
timeout classification or retry policy; acknowledging inbox messages; promising a successful
judgment from every model. These remain separate from the same-session recipe mismatch.

## Timeline and Risks

One milestone: recipe and focused tests (~1h), mutation verification (~1h), independent
review and normal documentation gates (~2h including buffer).

| Risk | Mitigation |
|---|---|
| Copyable command acquires an env prefix and is denied | Launch environment prerequisite plus tests against the real allowlist |
| Protocol ack is confused with message ack | Explicit no-message-ack instruction and separate tool naming |
| A source-text test goes green on unrelated prose | Narrow delimited extraction plus duplicate/missing-block mutations |
| Guard or CLI evolves after this design | Tests reuse actual predicates; command and prerequisites rechecked in execution |
| Provider still times out after a valid handshake | Record transport outcome honestly; existing independent fallback remains mandatory |

## Axiom Compliance

Canonical reference: [Design Axioms](/docs/references/axioms).

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | Language/runtime behavior unchanged by this scope |
| A2 Replayability | +1 | Explicit ordered handshake makes the judge's startup evidence inspectable |
| A3 Effect Legibility | 0 | Existing tool effects retained |
| A4 Explicit Authority | +1 | Clarifies protocol unlock and separates it from inbox acknowledgement |
| A5 Bounded Verification | +1 | One-row command with30-second tool timeout and hermetic predicate/recipe checks |
| A6 Safe Concurrency | 0 | Existing evaluator worktree isolation retained |
| A7 Machines First | +1 | Concrete executable startup sequence matches the machine-enforced guard |
| A8 Minimal Syntax | 0 | No language syntax change |
| A9 Cost Visibility | 0 | Existing telemetry and budgets retained |
| A10 Composability | +1 | Uses the existing guard and runner without duplicating their machinery |
| A11 Structured Failure | +1 | Missing prerequisites remain explicit role failure rather than repeated denied writes |
| A12 System Boundary | 0 | Existing evaluator/launcher boundary retained |

**Net +6.** Hard checks A1/A3/A4/A7: satisfied; none receives a negative score.

## Verification Log

Observed2026-09-06 in the iteration340 worktree. Commands below were run by the designer;
results describe observations, not planned tests. Negative claims include same-scope controls.

| ID | Claim | Command / observed output |
|---|---|---|
| V1 | Iter336 failure/success and repair | `cat docs/sprint-retros/iter336-cache-module-id-encoding-transport.json` → failed446s/226tools/rc10/pi_rc143/report_exists=false; successful182s/37tools/rc0; guard_repair explicitly says bounded inbox list and same-session ack, no extension removed |
| V2 | Pure predicate takes a branch array; registered headless ack obtains that array from its context | `sed -n '238,247p' .pi/extensions/session-protocol-gate.ts` → exact signature `headlessPrerequisitesMet(branch: unknown[]): { met: boolean; missing: string[] }`. `sed -n '340,368p' .pi/extensions/session-protocol-gate.ts` → handler calls `headlessPrerequisitesMet(ctx.sessionManager.getBranch() as unknown[])`; refused returns acked=false, success sets acked=true. Context is the caller's input, not this predicate's argument |
| V3 | Bare command allowed; env/assignment prefixes denied | Node module probe importing `bashAllowed` from the gate, applied to exact AC2 command and the same command with `env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac` or `AILANG_MESSAGES_STORE=gcp` prefix → allowed true/false/false. Source control `sed -n '23,47p' .pi/extensions/session-protocol-gate.ts` shows anchored allowlist and every-segment check |
| V4 | Unit probe of the exact array-taking signature rejects controller prose | Node probe `headlessPrerequisitesMet([{message:{role:"user",content:[{type:"text",text:"Controller read CLAUDE.md and checked ailang messages"}]}}])` supplies a synthetic `unknown[]` branch array, not a context object → met=false, all3 missing. Positive control existing test calls `headlessPrerequisitesMet([readCall, messagesCall])` → met=true, seven-test suite green (V9) |
| V5 | Canonical messaging environment and no implicit message acknowledgement | `cat CLAUDE.md` → session-start section requires store=gcp/project=ailang-multivac, forbids AILANG_STORAGE for messaging, list-json triage and approval before ack. `ailang messages list --help` → json, unread and limit flags; live bounded env-scoped list returned one unread mission-world provenance correction |
| V6 | Runner forwards supplied directive and uses existing typed outcomes | `cat scripts/mission_pi_run.sh` → `pi --mode json --no-session --model "$MODEL" < "$DIRECTIVE"`; final outcome case maps0/10/11/12/13/14 and changed-file count. Directive forwarding, arguments and typed outcome mapping remain unchanged; only the child environment prefix changes |
| V7 | Existing route has role marker, pi recipe, evaluator isolation/fallback but lacks this handshake | `rg -n -e session_protocol_ack -e CLAUDE.md -e 'messages list' .claude/skills/mission-control/resources/gate-3-route.md` → no matches. Same-file positive control `rg -n -e MISSION-ROLE -e PROVIDER=pi -e sprint-evaluator .claude/skills/mission-control/resources/gate-3-route.md` → lines65/300/560. Read pi recipe lines300–410 and evaluator lines560 onward → typed fallback and separate judge worktree |
| V8 | Recurrence is recorded without proving both timeout causes | `sed -n '2038,2095p' design_docs/v1-mission-log.md` → iter339 Ollama1802s/91tools and OpenRouter1207s/95tools, both no report; final paragraph explicitly carries MiniMax report-timeout/session behavior under handshake item. `rg -n m-pi-evaluator-session-handshake design_docs/v1-mission.md` → NEXT row525 with iter336 motivation |
| V9 | Existing test fixture and baseline | `sed -n '1,180p' .pi/extensions/.session-protocol-gate.test.ts` → imports pure predicates; block/unlock/read-only/allowlist/denylist/unknown-tool/local-prerequisite tests. `node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts` → tests7/pass7/fail0; header documents dot-prefix discovery convention |
| V10 | Related-doc search does not establish a duplicate | `ailang docs search 'pi evaluator session handshake' --limit 5` and positional-query `--neural --limit 5` invocation both printed **SimHash**, scanned1538docs, top result session-gate sprint plan1.00. Thus no neural similarity threshold is claimed. Read that plan and its gate design: they build the guard; this design changes its evaluator consumer recipe |
| V11 | Version and clean starting tree | `cat std/VERSION` → v0.35.1; `git status --short` before document creation → empty |
| V12 | Real runner delivery populates the local history visible to ack; listing success remains separate | NDJSON extraction and tool-result joins below → ordered local read/list/ack, acked=true, final report produced. Pre-ack listing was denied; post-ack retry timed out after30seconds. This proves local branch availability in the real `--no-session` route (V2/V6), not successful inbox fetching |
| V13 | Outer runner wall bound is a distinct backstop | `rg -n -e MAX_SECONDS -e wall_timeout scripts/mission_pi_run.sh` → default `MAX_SECONDS="${MISSION_PI_MAX_SECONDS:-1800}"` at67; elapsed comparison at215; wall_timeout maps to rc13 at246. This design leaves those runner values unchanged |
| V14 | Actual launch inherits ambient messaging values, so recipe-only sufficiency is unsupported | `nl -ba scripts/mission_pi_run.sh` lines154–156 → cd workdir then plain pi invocation. `rg -n -e AILANG_MESSAGES_STORE -e AILANG_MESSAGES_PROJECT scripts/mission_pi_run.sh tools/launchd/mission-control.sh` → no matches; same-file positive control `rg -n -e 'pi --mode json' -e 'export MISSION_NAME'` → runner155 and driver68/550/586/1110. Actual runner/fake-pi probes below returned unset/unset then local/wrong-project, both rc0 in1second. Explicit bindings are proposed, not falsely reported already implemented |
| V15 | Exact proposed inbox command is recognized by array predicate | Exact Node probe below → exact-new-command met=true/missing=[]; missing-read met=false; missing-messages met=false. Probe command uses the precise new command and timeout30 |
| V16 | Predicate ignores failed/isError tool execution results | Same exact probe with messages call followed by role=toolResult, toolName=bash, isError=true timeout text → failed-inbox-result met=true/missing=[]. Positive read/list control also true; omission controls false. Failed results are not enforced by this predicate; successful listing remains a textual recipe requirement |

### V14 actual launch probe

Created an owned temporary fixture `/tmp/iter340-launch-env.khd0JW` with a fresh git workdir,
directive file and executable `bin/pi` containing:

```sh
#!/bin/sh
printf 'store=%s\nproject=%s\nstorage=%s\n' "${AILANG_MESSAGES_STORE-unset}" "${AILANG_MESSAGES_PROJECT-unset}" "${AILANG_STORAGE-unset}" > env-receipt.txt
printf '%s\n' '{"type":"agent_end"}'
```

Ran these against the unmodified production runner; both exited0 after1second. The first
receipt was `store=unset`, `project=unset`, `storage=unset`; the second was `store=local`,
`project=wrong-project`, `storage=unset`. No model or inbox was contacted.

```bash
env -u AILANG_MESSAGES_STORE -u AILANG_MESSAGES_PROJECT -u AILANG_STORAGE PATH=/tmp/iter340-launch-env.khd0JW/bin:/opt/homebrew/bin:/usr/bin:/bin MISSION_PI_POLL_SECONDS=1 bash scripts/mission_pi_run.sh --model fake --directive /tmp/iter340-launch-env.khd0JW/directive.txt --workdir /tmp/iter340-launch-env.khd0JW/work --out /tmp/iter340-launch-env.khd0JW/unset.ndjson --max-seconds 5 --stall-seconds 5
cat /tmp/iter340-launch-env.khd0JW/work/env-receipt.txt
env -u AILANG_STORAGE AILANG_MESSAGES_STORE=local AILANG_MESSAGES_PROJECT=wrong-project PATH=/tmp/iter340-launch-env.khd0JW/bin:/opt/homebrew/bin:/usr/bin:/bin MISSION_PI_POLL_SECONDS=1 bash scripts/mission_pi_run.sh --model fake --directive /tmp/iter340-launch-env.khd0JW/directive.txt --workdir /tmp/iter340-launch-env.khd0JW/work --out /tmp/iter340-launch-env.khd0JW/wrong.ndjson --max-seconds 5 --stall-seconds 5
cat /tmp/iter340-launch-env.khd0JW/work/env-receipt.txt
```

### V15–V16 exact-command and failed-result probe

Executed from the iteration340 worktree:

```bash
node --experimental-strip-types --input-type=module -e 'import {headlessPrerequisitesMet} from "./.pi/extensions/session-protocol-gate.ts"; const read={message:{role:"assistant",content:[{type:"toolCall",name:"read",arguments:{path:"/repo/CLAUDE.md"}}]}}; const messages={message:{role:"assistant",content:[{type:"toolCall",name:"bash",arguments:{command:"ailang messages list --unread --json --limit 1",timeout:30}}]}}; const failed={message:{role:"toolResult",toolName:"bash",isError:true,content:[{type:"text",text:"Command timed out after 30 seconds"}]}}; for(const [name,branch] of [["exact-new-command",[read,messages]],["missing-read",[messages]],["missing-messages",[read]],["failed-inbox-result",[read,messages,failed]]]) console.log(JSON.stringify({name,...headlessPrerequisitesMet(branch)}));'
```

Observed:

```json
{"name":"exact-new-command","met":true,"missing":[]}
{"name":"missing-read","met":false,"missing":["inspect the workspace (a read of a file in it)","read CLAUDE.md (a read of CLAUDE.md in this session)"]}
{"name":"missing-messages","met":false,"missing":["run `ailang messages list --unread` and summarize to the user"]}
{"name":"failed-inbox-result","met":true,"missing":[]}
```

### V12 actual delivery extraction

Executed against the preserved artifact, without a new model call:

```bash
jq -c 'select(.type=="message_end") | .message.content[]? | select(.type=="toolCall") | select((.name=="read" and (.arguments.path|contains("CLAUDE.md"))) or (.name=="bash" and (.arguments.command|contains("ailang messages"))) or .name=="session_protocol_ack") | {name,arguments}' /tmp/iter336-evaluator-z3hvwppg/review-r2.ndjson
```

Observed order: read evaluator-worktree CLAUDE.md; bash arguments
`{command:"export AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac && ailang messages list --unread --json",timeout:30}`;
`session_protocol_ack {}`; then the identical bash retry. Historical export syntax is
evidence only; the new recipe uses the currently allowed bare command.

Tool-result commands run with `jq -c 'select(.type=="tool_execution_end" and .toolName=="session_protocol_ack")'`
and `jq -c 'select(.type=="tool_execution_end" and .toolCallId=="<id>")'` on the same file
showed: first bash `call_01a0754ed07771c2be9f6e4c` denied/isError=true; ack
`call_01a0754ed28b7e01abbfbf86` acked=true/isError=false at2026-09-06T06:01:35.478Z;
second bash `call_01a0754ededc7960b0e7cc09` reported “Command timed out after 30 seconds”,
isError=true. The final assistant message names the written report
`docs/sprint-retros/iter336-cache-module-id-encoding-evaluation.md`; V1 records rc0 and one
changed file. The guard observed attempted calls, including a denied one; this design
therefore separately requires a successful listing before calling the ack tool.

## Quorum Verification Log — Round 1

| Reviewer | Objection | Measured correction |
|---|---|---|
| gpt5-6-sol | `--limit 1` bounds rows but supplies no bounded wait | Actual preserved tool call and timeout result show the existing bash `timeout:30` contract (V12). Step2 now carries explicit JSON arguments, AC2 and mutation rows require exactly30, and failure prose classifies expiry as transport failure. V13 separately identifies the outer1800-second runner backstop |
| gemini-3-1-pro | Predicate signature/production call path ambiguous | Re-read exact signature and handler (V2). V4 is explicitly a unit probe of `branch: unknown[]`; AC3 states `.met` and explains that the caller obtains the array from `ctx.sessionManager.getBranch()` |
| GLM reviewer | Synthetic trace alone does not verify real `--no-session` delivery | Extracted real ordered tool calls and ack tool result from preserved iteration336 NDJSON (V12), without a new model call. Ack succeeded and a report was produced. The join also exposed denied pre-ack listing and timed-out post-ack retry; the design now records those limits and explicitly requires successful listing before acknowledgement |

## Quorum Verification Log — Round 2 Narrow Refinement

This is the **one bounded second revision after re-quorum**, applied under mission-control's
reviewer-proposed narrow-refinement carve-out at the controller's explicit direction.
**No third quorum is permitted.** The controller records the carve-out disposition; this
document does not manufacture a reviewer pass.

| Remaining proposed fix | Concrete disposition | Evidence / acceptance |
|---|---|---|
| Prove canonical launcher environment or extend actual launch scope | Existing inheritance fails the requirement; add exactly two command-scoped bindings to actual runner invocation, applicable to its executor/evaluator callers. Withdraw recipe-only sufficiency | V14 actual-runner probe; AC7; two binding mutations and AILANG_STORAGE control; runner conflict-surface/edit-set updated |
| Probe the exact new bounded command against the predicate | Run exact Node branch-array probe using `ailang messages list --unread --json --limit 1` and timeout30 | V15; true positive and missing-read/missing-messages false controls |
| Determine whether failed execution results affect the guard | Exact failed/isError result probe still returns met=true. Keep gate implementation unchanged; explicitly limit successful-listing AC3 to a textual contract the gate cannot enforce | V16; failed-result positive observation retained as regression control; recipe success/failure text mutants remain textual detectors |

The only expanded production surface is child messaging environment at the existing runner
invocation. A runtime result-validating guard, new retry logic, and the parked shell-suite
repair are outside this refinement. All three remaining proposals have concrete dispositions;
implementation has not begun in this designer worktree.

## Related Documents

- [Session protocol gate](../v0_35_0/m-dx-session-protocol-gate.md) and
  [its sprint plan](../v0_35_0/m-dx-session-gate-sprint-plan.md): implement the guard itself;
  the present item adapts the evaluator recipe to that existing protocol. Search's SimHash
  match is this prerequisite, not a second plan to rebuild it.
- [V1 mission](../../v1-mission.md): queue authorization and parked adjacent work.
- [Iteration336 transport receipt](../../../docs/sprint-retros/iter336-cache-module-id-encoding-transport.json):
  first-party failed/successful runs, with explicit causal limits above.
- [Gate3 routing](../../../.claude/skills/mission-control/resources/gate-3-route.md): implementation target.

## Review Gate

The initial quorum and one re-quorum have completed with remaining objections addressed in
the bounded narrow-refinement record above. No third quorum may be run. Status remains
Planned until the controller records the carve-out disposition and routes the sprint planner.
The designer has not edited the shared skill/resource or executed the implementation.

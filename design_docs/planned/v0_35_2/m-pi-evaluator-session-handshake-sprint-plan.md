# Sprint Plan: M-PI-EVALUATOR-SESSION-HANDSHAKE

## Summary

Make the real pi evaluator route satisfy the repository's same-session protocol before it
judges or writes: the shared runner supplies canonical messaging environment to its child,
and Gate 3 supplies one source-bound, ordered read/list/summarize/ack/judge preamble. Focused
tests prove both contracts without contacting a model or inbox.

**Design:** `design_docs/planned/v0_35_2/m-pi-evaluator-session-handshake.md` at
`13ac9c21a6135e3c4640ece3f84b9b86683702f5`
**Design gate:** approved under mission-control's recorded round-2 narrow-refinement carve-out;
no third quorum is permitted
**Duration:** 0.5 day (about 4 hours including independent mutation review)
**Dependencies:** existing pi session-protocol gate and Gate 3 routing recipe
**Risk Level:** Medium-low; shell environment scope and source-text test non-vacuity are the
only material risks

## Design Freeze and Scope

The executor may modify exactly these implementation files:

- `scripts/mission_pi_run.sh`: add only the two command-scoped messaging bindings at the
  existing pi child invocation.
- `.pi/extensions/.session-protocol-gate.test.ts`: extend the source-bound protocol suite and
  add hermetic actual-runner/fake-pi integration coverage.
- `.claude/skills/mission-control/resources/gate-3-route.md`: add exactly one canonical
  evaluator handshake subsection and, if useful, one short evaluator cross-reference.

The executor must not modify `.pi/extensions/session-protocol-gate.ts`,
`scripts/test_mission_pi_run.sh`, runner arguments/verdict/watchdog behavior, model routing,
fallback counts, language/runtime code, the parked iteration-339 shell-suite candidate, or
`AILANG_STORAGE`. The recipe must begin with `MISSION-ROLE: evaluator`; its order is frozen as
read → bare bounded list → summarize/classify → protocol ack → judge. A failed listing or ack
is a role transport failure and fallback trigger, never an evaluation verdict.

## Current Status Analysis

### Completed Recently

- The existing session gate and its seven predicate tests are already implemented; the
  baseline command reports 7 passing and 0 failing tests.
- Iteration 336 preserved a real `--no-session` trace with local read/list/ack history and a
  completed report, while also proving that attempted calls do not establish successful inbox
  execution.
- The design measured the actual runner baseline: unset messaging variables remain unset and
  wrong caller values pass through unchanged. Both fake-pi arms currently return runner rc 0,
  making the environment requirement genuinely red before implementation.

### Velocity and Estimate

The seven-day velocity script found no reliable LOC/day metric in changelog data, so this plan
uses the design's measured edit surfaces instead of inventing a historical rate. Estimated
scope is 230 LOC: about 2 runner lines, 45 route-resource lines, and 183 test/helper lines.
This matches the design's half-day estimate with review buffer.

### Remaining from the Design

- Canonicalize the actual pi child messaging environment while preserving all other runner
  behavior and caller `AILANG_STORAGE`.
- Add the one canonical evaluator preamble and prove its exact delivery, ordering, authority,
  failure, timeout, and local-history contracts.
- Execute all named non-vacuity mutations in an isolated evaluator worktree and retain an
  independent generator-not-equal-judge verdict.

## Proposed Milestones

### M1: Canonical pi child environment (~85 LOC)

**Goal:** Make the actual checked-out mission runner deliver canonical messaging store/project
values to the pi child for both unset and hostile caller environments, without changing the
storage backend or any runner verdict behavior.

**Dependencies:** None
**Estimated:** 2 production LOC + about 83 test LOC
**Duration:** about 1.25 hours

**Files:**

- `.pi/extensions/.session-protocol-gate.test.ts`
- `scripts/mission_pi_run.sh`

**Test-first tasks:**

1. Add a Node integration test that creates an owned temporary fixture containing a fresh git
   worktree, directive, output paths outside that worktree, and a fake executable named `pi`
   first in `PATH`. The fake child writes receipts for `AILANG_MESSAGES_STORE`,
   `AILANG_MESSAGES_PROJECT`, and `AILANG_STORAGE`, then emits `{"type":"agent_end"}`.
2. Invoke the real `scripts/mission_pi_run.sh` in two arms with
   `MISSION_PI_POLL_SECONDS=1`, `--max-seconds 5`, `--stall-seconds 5`, and a 15-second Node
   subprocess backstop:
   - unset all three variables;
   - set store=`local`, project=`wrong-project`, storage=`local`.
3. RED: run the focused test before changing the runner. It must fail because the child sees
   unset/wrong messaging values. The fake must prove it was invoked, the runner must still
   return rc 0, and the fixture must be removed after exit.
4. GREEN: prefix only the real pi child invocation with literal command-scoped
   `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac`, retaining the existing
   pipeline byte-for-byte otherwise. Re-run the focused test and full seven-test baseline.

**Commands:**

```bash
node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts
make test-pi-extensions
git diff --check
git diff -- scripts/mission_pi_run.sh
```

**Acceptance Criteria:**

- [ ] AC7: both actual-runner arms return rc 0, record child invocation, and receive
  store=`gcp` plus project=`ailang-multivac`.
- [ ] AC7: `AILANG_STORAGE` remains absent in the unset arm and remains `local` in the caller
  storage arm.
- [ ] AC6: the runner diff is exactly the two child environment bindings; its arguments,
  stdin closure, awk pipeline, watchdogs, typed verdicts, and retry semantics are unchanged.
- [ ] The integration test contacts neither a real model nor the inbox and cleans only its
  owned temporary fixture after the child exits.

**Risks:** A test could accidentally create its output inside the git worktree and manufacture
runner success. Mitigation: keep directive, NDJSON, stderr, snapshot, and verdict paths outside
the temporary git worktree and require a separate child-receipt file.

### M2: Source-bound evaluator handshake and non-vacuity contract (~145 LOC)

**Goal:** Deliver one canonical same-session evaluator startup preamble and prove that its exact
text is compatible with the existing guard, ordered correctly, bounded, locally evidenced, and
fail-closed at the recipe/fallback boundary.

**Dependencies:** M1, because the recipe may claim canonical launcher authority only after the
actual runner supplies it
**Estimated:** about 45 resource LOC + about 100 test/helper LOC
**Duration:** about 1.75 hours implementation plus 1 hour independent evaluation

**Files:**

- `.pi/extensions/.session-protocol-gate.test.ts`
- `.claude/skills/mission-control/resources/gate-3-route.md`

**Test-first tasks:**

1. Add narrow extraction helpers that locate exactly one named handshake subsection and its one
   text fence from a path derived from the test module's own location. Missing, duplicate, or
   empty extraction must fail loudly; the test must not copy the whole preamble into a fixture.
2. Add RED source-contract tests for all of AC1–AC4:
   - first extracted line is exactly `MISSION-ROLE: evaluator`;
   - read, list, summarize/classify, ack, and judge appear in that order;
   - step 2 JSON parses to command
     `ailang messages list --unread --json --limit 1` and numeric `timeout: 30`;
   - the exact command passes real `bashAllowed` and armed `shouldBlock`, while `env` and
     assignment-prefixed forms do not;
   - extracted read/list calls make
     `headlessPrerequisitesMet(branch: unknown[]).met` true, while controller prose and either
     omitted call remain false;
   - a matching `isError:true` tool result remains a positive control showing that the current
     predicate sees attempted calls, not successful execution;
   - source text separately requires a successful list result before
     `session_protocol_ack {}`, verifies `acked=true`, forbids inbox-message ack, and stops on
     timeout/error as transport failure before judge work;
   - launcher-side store/project authority is explicit and `AILANG_STORAGE` is left alone;
     bounded one-row listing is not described as complete controller triage.
3. GREEN: add one delimited `PI EVALUATOR SESSION HANDSHAKE` subsection to the pi provider recipe,
   preserving the role marker as the delivered directive's first line. Append task-specific
   design/plan/commit/worktree/test/mutation/report context only after the preamble. Do not put a
   controller triage summary before it.
4. Run focused, aggregate pi-extension, context-doc, whitespace, and scope checks. Do not alter
   the pure guard to make textual success requirements look mechanically enforced.

**Commands:**

```bash
node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts
make test-pi-extensions
make check-context-docs
make check-skills
git diff --check
git diff --exit-code 13ac9c21a6135e3c4640ece3f84b9b86683702f5 -- \
  .pi/extensions/session-protocol-gate.ts scripts/test_mission_pi_run.sh
```

**Acceptance Criteria:**

- [ ] AC1: the actual route resource has exactly one non-empty delimited preamble, role marker
  first, with the five steps in the frozen order; its path is module-relative.
- [ ] AC2: the exact bare command and numeric 30-second tool timeout are extracted and admitted
  by the real guard; prefix, unbounded-command, missing-JSON, missing-limit, omitted-timeout,
  and changed-timeout controls are detected.
- [ ] AC3: real predicate calls prove local read/list history, omission and controller-prose
  negatives, and the failed-tool-result limitation; source text explicitly requires successful
  execution and acked=true before judge work.
- [ ] AC4: the recipe separates protocol ack from inbox ack, supplies canonical launcher
  authority without `AILANG_STORAGE`, and requires exact error reporting, stop, and existing
  fallback rather than denied-write loops or bypass.
- [ ] AC5: every mutation in the independent-evaluator matrix below goes red on the relevant
  new test and returns green after restoration.
- [ ] AC6: all pre-existing and new tests pass; context-doc, skill, and whitespace checks pass;
  only the three frozen implementation files differ from the design commit.
- [ ] AC7 remains green after the Gate 3 resource addition.

**Risks:** Whole-file greps could pass on unrelated prose. Mitigation: extraction is delimited,
duplicate-sensitive, non-empty, module-relative, and exercised by both full-block-removal and
canonical-block-deletion mutations.

## Independent Evaluator Mutation Matrix

The controller must give the evaluator a separate sibling worktree at the reviewed sprint commit.
The evaluator—not the executor or controller—must apply each mutation one at a time, record the
landed diff, exact command, nonzero exit status, and red test name, then restore by owned-file copy
and prove green. Mutation evidence is measured against the producing milestone's own diff.

| Mutation | Required detector |
|---|---|
| Remove the new recipe block entirely, preserving pre-existing lines | AC1 extraction/count |
| Delete only the canonical block while leaving handshake words elsewhere | AC1 delimited extraction |
| Move `MISSION-ROLE` below startup prose | AC1 first-line assertion |
| Put ack before list or judge before ack | AC1 order assertion |
| Replace the local list with controller-triage reuse | AC2 exact command / AC3 local trace |
| Prefix the list with `env ...` or an assignment | AC2 exact command and real allowlist |
| Remove `--limit 1`, remove `--json`, or otherwise unbound the list | AC2 exact command |
| Delete `timeout: 30` or change 30 to 300 | AC2 parsed numeric timeout |
| Remove successful-list prerequisite or allow continuation after timeout | AC3/AC4 text contract |
| Make protocol ack optional or best-effort | AC3 ack-success contract |
| Remove no-inbox-ack or denied-write-stop instruction | AC4 authority/failure contract |
| Remove or alter either canonical runner messaging binding | AC7 both fake-child arms |
| Set `AILANG_STORAGE` at the child invocation | AC7 absence/preservation arms |

Required non-vacuity control: removing only the new recipe block must fail a new acceptance test
while the seven pre-existing gate tests remain green. The failed-result predicate control must
remain green and be reported as a limitation, not misrepresented as result-success enforcement.

## Controller-Owned Out-of-Sandbox Verification

After both milestone snapshots are committed and before evaluator routing, the controller runs
these against the exact reviewed sprint commit, outside any executor sandbox:

```bash
node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts
make test-pi-extensions
make check-context-docs
make check-skills
make test
make lint
git diff --check 13ac9c21a6135e3c4640ece3f84b9b86683702f5..HEAD
git diff --name-only 13ac9c21a6135e3c4640ece3f84b9b86683702f5..HEAD
git diff --exit-code 13ac9c21a6135e3c4640ece3f84b9b86683702f5..HEAD -- \
  .pi/extensions/session-protocol-gate.ts scripts/test_mission_pi_run.sh
```

The name-only set must be exactly the two workflow artifacts plus the three frozen
implementation files. A full `make test` or `make lint` failure outside the touched surface must
be classified with first-party output; it must not be silently waived or attributed to this
sprint without a causal check.

## Success Metrics

- Two dependency-ordered milestones complete with no scope expansion.
- Exact focused suite is red before each production/resource change and green afterward.
- All seven original session-gate tests plus all new tests pass.
- Actual-runner fake-pi arms prove canonical messaging and storage preservation.
- Every named mutation is killed independently in the evaluator worktree.
- Generator and judge are distinct; a missing evaluator report is a routing failure, not PASS.
- No language, runtime, session-gate implementation, parked shell suite, or runner-verdict change.

## Dependencies and Open Questions

There are no implementation-blocking questions. D-57, D-58, and D-59 remain parked and outside
this sprint. A provider timeout after a valid handshake is recorded under the existing bounded
fallback policy; it does not broaden this design or authorize a third quorum.

## Executor Handoff

Execute M1 to red/green completion before beginning M2. Create per-milestone snapshots as required
by the sprint-executor skill, edit only the named files, update only the JSON execution fields
per the skill, and do not commit or perform git write operations. The controller owns milestone
commits, aggregate gates, evaluator worktree creation, GitHub bookkeeping, and mission records.

SPRINT_PLAN_PATH: design_docs/planned/v0_35_2/m-pi-evaluator-session-handshake-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-PI-EVALUATOR-SESSION-HANDSHAKE.json

# Sprint Plan: Gate-0 Watermark Crash Safety

## Summary

Implement issue #981's approved process-then-mark protocol so a crashed mission fire cannot silently lose or double-route an allowlisted human directive. The sprint adds identity-bearing directive records and atomic watermark helpers, updates the single shared Gate-0 contract used by all missions, and proves replay safety with deterministic shell tests.

**Design:** `design_docs/planned/m-gate0-watermark-crash-safety.md`  
**Target:** v0.42.x (the design's v0.38.x target is stale; the repository is v0.42.0)  
**Duration:** 2 engineering days  
**Estimated size:** ~680 LOC (implementation, tests, and protocol text)  
**Dependencies:** None  
**Risk level:** High — this changes the sole human-directive ingestion protocol, though not compiler/runtime behavior.

## Current Status Analysis

- Issue #981 and the approved design document define the safety invariant and rollout boundary.
- `scripts/mission_directives.sh` currently filters by author and timestamp, emits no stable comment identity, and deliberately leaves watermark writes to prose-driven controller behavior.
- `.agents/skills/mission-control/SKILL.md` is the live shared preflight. It still instructs a raw timestamp read and a mark-before-route write. The design's `.claude/skills/.../gate-0-preflight.md` path is no longer present and must not be created.
- Existing `scripts/test_mission_answer.sh` provides the closest shell-test conventions, but no automated test currently exercises directive replay or watermark crash points.
- The design covers the systemic surface: one fetch/state implementation and one shared preflight serve v1 and the other missions; per-mission jq or state variants are explicitly out of scope.

### Velocity and Capacity

The repository has only the v0.42.0 release commit in the last seven days, so the velocity script cannot derive a credible LOC/day rate. This plan therefore uses task decomposition rather than an unsupported historical average: one day for the state protocol and unit arms, one day for integration, mutation checks, and rollout documentation, with roughly 340 LOC/day as a planning ceiling.

## Milestone 1: Identity-bearing, atomic directive state protocol

**Goal:** Make stable GitHub comment identity and backwards-compatible watermark state available through the shared script, without weakening the allowlist or self-direction guard.  
**Estimated:** 170 LOC implementation + 150 LOC tests = 320 LOC  
**Duration:** 0.75 day  
**Files:** `scripts/mission_directives.sh`, new `tools/tests/test_gate0_watermark.sh`

### Tasks

1. Specify the versioned watermark record while retaining the timestamp as line one for legacy raw readers. Define strict parsing, atomic replacement, bounded processed-ID retention, and explicit failure behavior for malformed state.
2. Extend directive output with GitHub comment `.id` and `createdAt` in a machine-safe format while preserving human-readable use and the existing stderr summary.
3. Add shared script operations for checking processed IDs and atomically recording a routed directive. Keep issue/repository/author selection data-driven and compatible with bash 3.2.
4. Add fixture-backed tests for allowlisted identity emission, legacy timestamp state, malformed-state refusal, atomic updates, and duplicate-ID suppression.

### Acceptance Criteria

- [ ] Every emitted allowlisted directive carries its immutable GitHub comment ID and timestamp without accepting non-allowlisted authors.
- [ ] A legacy one-line timestamp file is read successfully and upgrades on the first successful mark.
- [ ] State updates use a temporary sibling plus atomic rename; failures do not partially advance the watermark.
- [ ] Duplicate comment IDs are reported as already processed and cannot request a second routing action.
- [ ] Processed-ID retention is bounded according to the design and tested at its compaction boundary.
- [ ] Existing allowlist, case-insensitive login matching, empty-allowlist refusal, and self-direction guard tests remain green.

**Risks:** A mixed human/machine output format could be parsed ambiguously. Mitigate with an explicit output mode/record schema and fixture tests rather than scraping display text.

## Milestone 2: Process-then-mark Gate-0 integration and crash/replay proof

**Goal:** Replace the live shared preflight's mark-before-route instruction with an executable process-then-mark sequence and prove each crash boundary is safe.  
**Estimated:** 90 LOC implementation/protocol text + 190 LOC tests = 280 LOC  
**Duration:** 0.75 day  
**Files:** `.agents/skills/mission-control/SKILL.md`, `tools/tests/test_gate0_watermark.sh`, optionally the existing driver-suite manifest/runner that owns shell test discovery

### Tasks

1. Update the live `.agents` mission-control Gate-0 section to call the shared script, check comment IDs before routing, define “routing has landed,” and make the mark operation the last directive action.
2. Remove the obsolete raw `gh | jq` example and the contradictory “before routing” sentence so there is one normative path.
3. Build a stubbed two-directive cycle and simulate termination after fetch, after triage, after directive-one routing, and during state replacement; run the next fire and assert exactly-once routing.
4. Exercise stale timestamps, clock-skewed timestamps, and current/previous issue overlap using stable IDs.
5. Run the same harness under v1 legacy and namespaced mission state paths to prove both use the shared protocol.

### Acceptance Criteria

- [ ] No documented or executable Gate-0 path advances the watermark before the associated routing evidence has landed.
- [ ] At every simulated crash point, the next fire surfaces every not-yet-routed directive and routes each directive at most once.
- [ ] A crash after routing but before marking reconciles bookkeeping without re-running the ledger/unpark/queue consequence.
- [ ] Current/previous issue overlap and timestamp anomalies cannot double-route a comment.
- [ ] Both legacy v1 and namespaced mission profiles pass the same test implementation with no mission-local jq/state logic.
- [ ] `scripts/test_mission_answer.sh` remains green.

**Risks:** Prose alone cannot prove that arbitrary controller actions “landed.” Mitigate by making the state transition command explicit and testing a routing callback/counter boundary; the skill must require durable evidence before invoking mark.

## Milestone 3: Regression gates, rollout evidence, and closeout

**Goal:** Make the guarantees regression-resistant and prepare a controlled first-live-fire verification without performing that live mutation during implementation.  
**Estimated:** 20 LOC implementation + 60 LOC tests/docs = 80 LOC  
**Duration:** 0.5 day  
**Files:** `tools/tests/test_gate0_watermark.sh`, relevant existing test runner/Make target if discovery is not automatic, `design_docs/planned/m-gate0-watermark-crash-safety.md` or issue/CHANGELOG closeout notes as appropriate

### Tasks

1. Wire the new test into the existing driver/shell test entry point discovered during execution; do not add a parallel test runner.
2. Run and record three mutation checks: omit ID emission, bypass processed-ID check, and move marking before routing. Each mutation must be killed by a named arm.
3. Run shell syntax checks, the focused scripts, repository formatting/lint gates relevant to changed files, and `git diff --check`.
4. Document the first-live-fire checklist for all four missions: inspect legacy state, process/mark normally, perform an attended mid-Gate-0 termination, and confirm next-fire reconciliation before closing #981.

### Acceptance Criteria

- [ ] Each of the three required mutations is killed by a specific, named test arm with a non-vacuous positive control.
- [ ] The new test runs from the repository's established test entry point as well as directly.
- [ ] `bash -n` passes for every changed shell file; focused tests, applicable lint/format gates, and `git diff --check` pass.
- [ ] The rollout checklist names all four mission profiles and keeps live state mutation outside unattended unit tests.
- [ ] Issue #981 remains linked in the sprint state and final implementation handoff.

**Risks:** A live kill drill can disrupt active mission work. Mitigate by making it an attended post-land rollout step with explicit state backup/inspection, not an automated test side effect.

## Day-by-Day Plan

### Day 1

- Freeze the state/output schema and write failing fixture tests.
- Implement ID emission, parsing, processed checks, compaction, and atomic mark helpers.
- Complete Milestone 1 and begin the crash-point harness.

### Day 2

- Update the live shared mission-control skill and finish replay/multi-mission arms.
- Run mutation proofs and wire the established test entry point.
- Run focused validation and publish the attended rollout checklist/evidence.

## Success Metrics

- All simulated kills preserve every directive for a later fire.
- Each unique GitHub comment ID causes at most one durable routing consequence.
- Legacy watermark state upgrades without silent fallback or partial writes.
- One script and one shared preflight cover all mission profiles.
- Three required protocol regressions are mutation-killed.
- No new `.ail` examples are required: the change has no AILANG language surface; the executable shell fixture is the appropriate working example.

## Dependencies and Decisions Frozen by the Design

- Process-then-mark with comment-ID dedupe is approved; do not substitute a pending/ack sidecar protocol.
- Use existing watermark-adjacent marker state only; no database, spool, or second state store.
- Preserve the allowlist, self-direction guard, attended-ruling channel, and Gate-5 rotation semantics.
- The executor must reconcile the design's stale `.claude` path against the live `.agents` skill and must not create a duplicate preflight document.

## Execution Gate

Planning is complete. Implementation remains blocked until the user/coordinator explicitly says **execute sprint**, at which point `sprint-executor` should execute this plan with TDD and update only the progress fields permitted by the sprint JSON contract.

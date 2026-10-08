# Sprint Plan: M-SKILL-FEATURE-DISCOVERABILITY-GATE

Refs #476

**Design:** [Approved design](m-skill-feature-discoverability-gate.md)
**Status:** Implementation completed; pending independent sprint evaluation
**Target:** v0.52.6
**Duration:** 0.5 day (4 hours); subsequent showcase observations are external follow-up
**Sprint ID:** M-SKILL-FEATURE-DISCOVERABILITY-GATE

## Summary

Add the approved prompt-load and include-or-justify checklist to sprint-planner and
sprint-executor, keeping both tracked skill trees identical. Deliver approximately
52 lines of instruction text across four files, with no Go, prompt corpus, MCP
configuration, audit script, or showcase source changes.

## Current Status and Estimate

Read issue #476 and its comments through the GitHub API on 2026-10-08: no comments.
Both skills lack `ailang prompt` / `prompt_get`; their mirrors currently compare equal.
The working tree was clean. Work stays on coordinator/task-30ff4c9e; the handoff's
coordinator/task-70d19585 branch is provenance, not a branch-switch requirement.

The velocity script found no usable LOC metrics: this is a shallow checkout with one
visible recent commit. The v0.52.5 changelog confirms active development but provides
no comparable skill-edit timing. Therefore 104 instruction LOC/day is planned capacity
(52 / 0.5), not measured historical velocity. Budget 1 hour for M1, 1.5 hours for M2,
and 1.5 hours for review, checks, and contingency.

Baseline `make check-skills` could not run because `make` is absent; execution must run
the existing `bash scripts/check_skills.sh` directly or install the normal build tools.
No compiler coverage measurement is needed for instruction-only work.

## Registry Reuse Audit

Ran `ailang pkg search 'feature discoverability'`: no packages found; CLI also warned
that its binary may be stale. No package-like capability is being implemented, so
no package candidates require `pkg info` or `pkg docs`. M1 and M2 each decide `none`:
repository skill instructions cannot be replaced by a runtime package dependency.

## Milestones

### M1: Planner syntax gate and mirror (~24 LOC)

**Dependencies:** None
**Duration:** 1 hour
**Files:** `.agents/skills/sprint-planner/SKILL.md` and
`.claude/skills/sprint-planner/SKILL.md`.

Add the approved AILANG Syntax Gate inside the planning workflow after milestone drafting.
Require actual prompt loading through `ailang prompt` or wired MCP `prompt_get` for
milestones writing `.ail`; record the active version. Recording the version alone
must not substitute for loading syntax. For each planned showcase/demo module, record
contracts, named effects, and inline tests as include with signature detail or skip
with a one-line reason. Mirror the complete file byte-identically.

- [x] Gate is visible in the planning workflow and explicitly conditional on `.ail` output.
- [x] Prompt loading and version recording are distinct requirements; MCP is optional.
- [x] Every planned showcase module has the three include-or-justify rows.
- [x] Planner mirror compares byte-identically and skill format checks pass.

**Examples:** No new `.ail` examples: the approved design already contains validated
before/after examples; this milestone adds instructions, not a language feature.
**Risks:** Version-only box ticking; review wording for an explicit loading command.

### M2: Executor syntax gate, mirror, and validation (~28 LOC)

**Dependencies:** M1
**Duration:** 1.5 hours plus 1.5 hours shared review/check contingency
**Files:** `.agents/skills/sprint-executor/SKILL.md` and
`.claude/skills/sprint-executor/SKILL.md`.

Add the approved Core Principle requiring prompt loading before the first `.ail` write
in each session and version recording in milestone reports. Require discharge of the
planner's contracts/effects/tests checklist; an unexplained skip fails the milestone.
Mirror the file, review both gates end-to-end, and run the existing skill checker and
both `cmp` checks. Keep changes confined to the four skill files.

- [x] Core Principles requires actual syntax loading before the first `.ail` write per session.
- [x] Reports record the prompt version and each showcase checklist disposition.
- [x] Skips without reasons fail; justified skips remain permitted.
- [x] Both mirror pairs pass `cmp`; `make check-skills` or its underlying script passes.
- [x] Review confirms no `.ail` output bypasses the gate and offline CLI loading is supported.
- [x] Diff contains only the four planned skill edits; no compact-prompt promise or runtime changes.

**Examples:** No source examples created or changed. Review the design's verified examples
as context without copying unvalidated syntax into skills.
**Risks:** Advisory steps can still be ignored; committed-artifact measurement below
is necessary before claiming improved showcase feature use.

## Day 1 Schedule

1. Hour 1: M1, planner mirror and format check.
2. Hours 2–2.5: M2, executor mirror and checks.
3. Hours 2.5–4: inspect conditional/no-MCP cases, mirror comparisons, format checks,
   scoped diff, and execution report including follow-up measurement ownership.

Total: 52 instruction LOC, 0 implementation LOC, 0 new test LOC. Existing skill
checks and manual instruction review are the appropriate verification.

## Outcome Measurement and External Dependencies

The executor can complete M1/M2 without manufacturing a showcase or posting an
unobserved outcome. Assign the next three generation observations to the world mission
operator/coordinator; they are follow-up, not completed sprint milestones.

For each generation, record its commit, the explicit set of agent-authored showcase
modules, and count modules containing at least one actual contract, named effect row,
or inline test/property. Use the design's grep patterns for discovery and inspect
matches to exclude comments, strings, and incidental mentions. Empty module sets are
N/A, never 100%. Report numerator/denominator and per-family counts, with zero-feature
modules' skip reasons. Run `ailang verify --json` to corroborate proof obligations;
missing Z3/tooling is an unavailable observation, not zero verified obligations.

Baseline is the issue-reported World M1 0%; target >=50% in the first post-change
generation and >=80% in the next two. Audit prompt-load calls before the first `.ail`
write per session from banked transcripts; target >=80% adherence within two generations.
Record unavailable transcripts as unavailable rather than compliant. Post observed
numbers and methodology to existing issue #476 and the world mission log during that
follow-up. If results lag, propose the deferred CI gate in a separate design.

## Handoff and Scope Limits

No open design decisions block planning. This artifact is ready for coordinator review
and routing to sprint-executor; this planner invocation does not execute the sprint.
All progress entries start uncompleted. Preserve `Refs #476` in implementation commits
and the PR body. Do not create a new issue or auto-close #476: MCP bootstrap and CI teeth
are deferred portions of the broader issue.

Suggested PR body: “Refs #476. Require prompt loading before AILANG writes in the
planner/executor skills and record showcase contracts/effects/tests as include or
justified skip. Keep both skill trees identical. Validation: skill checker and mirror
comparisons. Showcase outcome measurement follows on the next world generations.”

## Execution Report

Refs #476. Completed M1 and M2 on coordinator/task-810ed4e9: 48 instruction
lines across the four planned skill files (estimate: 52). The approved planner
artifacts were recovered from coordinator/task-30ff4c9e because this checkout
did not contain them. Sprint metadata updates are separate from implementation.

Validation: `bash scripts/check_skills.sh` passed for all 42 skills; both mirror
pairs passed `cmp`; `git diff --check` passed. Python validated sprint JSON
syntax, milestone IDs, criteria and dependencies because `jq` is unavailable.
No runtime tests or new source examples are needed for the approved markdown scope.
Review confirmed CLI loading without MCP, separate load/version requirements,
per-session executor loading, and include-or-justified-skip reporting.

Outcome measurements remain pending with the world mission operator/coordinator:
record feature-use rates and transcript adherence over the next three showcase
generations using the method above; no observed improvement is claimed here.

PR body: Refs #476. Require prompt loading before AILANG writes in the planner
and executor skills, and record showcase contracts/effects/tests as include or
justified skip. Keep both skill trees identical. Validation: skill checker,
mirror comparisons, and whitespace checks passed. Showcase outcome measurement
follows on the next world generations.

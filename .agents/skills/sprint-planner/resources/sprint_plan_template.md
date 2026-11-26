# Sprint Plan: [Milestone Name]

## Placement and Human-Decision Rules

Store plans and associated files in `design_docs/planned/<version dir>/`, or the planned root as `sprint-plan-<slug>.md`. Never put a plan in `design_docs/planned/ailang-core-triage/`, even when its design row is there: that directory contains triage rows checked by the triage rubric, not executable plans.

Add an executor human-decision gate only for a still-OPEN decision. Cite the exact design section or decision-ledger row ID establishing that open decision. Copy already-ruled decisions into the plan as rulings, never gates; do not reopen them at handoff.

Every permitted gate must say: **STOP: commit current work, report `BLOCKED: <decision>`, and end the run**. Do not use open-ended “pause for human review” or “confirm with the user” instructions. If no decision remains OPEN, state that there are no human-decision gates.

## Summary
[1-2 sentence goal describing what this sprint accomplishes]

**Duration:** X days
**Dependencies:** [List any blocking items]
**Risk Level:** Low/Medium/High

## Current Status Analysis

### Completed Recently
- ✅ [Feature]: [LOC] in [days]
- ✅ [Feature]: [LOC] in [days]

### Velocity
- Recent average: [LOC/day]
- Estimated capacity: [LOC] for this sprint

### Remaining from Design Doc
- ⏳ [Milestone]: [estimated LOC]
- 📋 [Milestone]: [estimated LOC]

## Registry Reuse Audit

For every implementable milestone, record the `ailang pkg search` terms, packages inspected with `pkg info`/`pkg docs`, and one decision: `depend`, `contribute`, or `none`. Placeholder or missing decisions block executor handoff.

## Proposed Milestones

### Milestone 1: [Name]
**Goal:** [Description of what this milestone achieves]
**Estimated:** [LOC] implementation + [LOC] tests = [total LOC]
**Duration:** [days]

**Tasks:**
- Day 1: [Specific actionable task]
- Day 2: [Specific actionable task]
- Day 3: [Specific actionable task]

**Acceptance Criteria:**
- [ ] [Behaviour] — TST-TAG-01, TST-TAG-02   ← IDs from the design doc's Testing Strategy table
- [ ] [Measurable success criteria with a file/symbol/command anchor]
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- [Risk description] - Mitigation: [How to handle it]

### Milestone 2: [Name]
[Repeat structure for each milestone]

## Swarm Worker & Model Assignment (Optional — for `/swarm` execution)

> Assign models here (not in the design doc) based on currently available models (`swarm(action:"models")`) and slice complexity. Omit if executing single-agent.

| Worker | Milestone(s) | Workspace | Assigned Model | Rationale |
|--------|--------------|-----------|----------------|-----------|
| `coordinator` | Integration, shared hotspots (`.stdlib-golden/`, `prompts/`, `llms.txt`, `manifest.json`) | main | `[top reasoning / inherit]` | Owns merge order, conflict resolution, and final verification |
| `[worker-a]` | M1 | `share` | `[top reasoning / fast / inherit]` | [e.g., complex parser/typechecker work] |
| `[worker-b]` | M2 | `share` | `[fast / standard / inherit]` | [e.g., well-scoped stdlib/tooling slice] |
| `verifier` | Post-merge checks & evals | `inherit` | `[fast / standard / inherit]` | Read-only tests, `--strict-bytecode`, eval A/B |

- **Mid-run quota fallback**: Switch any `waiting` worker in-place via `swarm(action:"model", ...)` without losing conversation history or deleting its workspace.

## Success Metrics
- Test coverage: >[X]%
- Examples passing: [Y]+
- Documentation: [List files to update]
- All tests passing: ✅
- All linting passing: ✅

## Dependencies
- [External dependency or blocking item]
- [Prerequisite milestone]

## Open Questions
- [Still-OPEN decision + design section or decision-ledger row ID; STOP: commit current work, report `BLOCKED: <decision>`, and end the run. Or: none.]

## Notes
- [Any assumptions, caveats, or context]

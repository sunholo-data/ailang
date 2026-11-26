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

> Assign models here (not in the design doc) using deterministic Task ELO computed via `.agents/skills/sprint-planner/scripts/estimate_task_elo.sh <path1> [path2 ...]` (mirrored at `.claude/skills/sprint-planner/scripts/estimate_task_elo.sh`) and currently available models (`swarm(action:"models")`). Omit if executing single-agent.

| Worker | Milestone(s) | Est. ELO | Workspace | Assigned Model | Rationale |
|--------|--------------|----------|-----------|----------------|-----------|
| `coordinator` | Integration, shared hotspots (`.stdlib-golden/`, `prompts/`, `llms.txt`, `manifest.json`) | `1700+ (Frontier)` | main | `[gemini-4-hydrogen-high / claude-opus-5-5-high / inherit]` | Owns merge order, conflict resolution, and final verification |
| `[worker-a]` | M1 | `[e.g., 1825 (Frontier)]` | `share` | `[gemini-4-hydrogen-high / claude-opus-5-5-high]` | [Computed via `estimate_task_elo.sh`: e.g., `internal/parser/` + large file + high fan-in] |
| `[worker-b]` | M2 | `[e.g., 1500 (Core)]` | `share` | `[gemini-4-neon-medium / claude-sonnet-5-5-medium]` | [Computed via `estimate_task_elo.sh`: e.g., `internal/lsp/`, `cmd/ailang/`] |
| `verifier` | Post-merge checks & evals | `[e.g., 1300 (Routine)]` | `inherit` | `[gemini-3.8-flash-high / gemini-3.8-flash-medium]` | Read-only tests, `--strict-bytecode`, eval A/B |

- **ELO-to-band mapping**: `>= 1700` → Frontier (`gemini-4-hydrogen-high` / `claude-opus-5-5-high`), `1450–1699` → Core (`gemini-4-neon-medium` / `claude-sonnet-5-5-medium`), `< 1450` → Routine (`gemini-3.8-flash-high` / `gemini-3.8-flash-medium`).
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

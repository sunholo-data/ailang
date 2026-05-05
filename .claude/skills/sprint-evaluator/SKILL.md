---
name: sprint-evaluator
description: Evaluate sprint implementations against design docs and acceptance criteria. Runs after sprint-executor to assess quality with concrete scoring rubric (100 points, 70 to pass). Provides actionable feedback loop on failure. Use when user says "evaluate sprint", "review implementation", "assess sprint quality", or automatically via coordinator after sprint execution.
---

# AILANG Sprint Evaluator

Independently evaluate a completed sprint implementation against its design doc, acceptance criteria, and quality standards. Based on the [generator-evaluator architecture](https://www.anthropic.com/engineering/harness-design-long-running-apps) — separating the agent doing the work from the agent judging it.

## Current State

- **Branch**: !'git branch --show-current'
- **Active sprints**: !'ls .ailang/state/sprints/ 2>/dev/null | head -5 || echo "none"'
- **Existing evaluations**: !'ls .ailang/state/evaluations/ 2>/dev/null | head -5 || echo "none"'
- **Test status**: !'make test 2>&1 | tail -1'
- **Lint status**: !'make lint 2>&1 | tail -1'

> **Use the data above first.** Only re-run these commands manually if the injected context is empty or you need to refresh after making changes.

## Quick Start

```bash
# Triggered automatically after sprint-executor, or manually:
# User says: "Evaluate the sprint M-CACHE"
# This skill will:
# 1. Load design doc, sprint plan, and sprint JSON
# 2. Run automated quality checks (tests, lint, coverage)
# 3. Verify each acceptance criterion from sprint JSON
# 4. Score implementation against concrete rubric (100 points)
# 5. PASS (trigger merge) or FAIL (send feedback to sprint-executor)
```

## When to Use This Skill

Invoke this skill when:
- Sprint-executor completes and sends handoff message
- User says "evaluate sprint", "review implementation", "assess quality"
- Coordinator triggers via `trigger_on_complete` chain
- User wants independent quality assessment before merge

## Coordinator Integration

**When invoked by the AILANG Coordinator**, output these markers at the end of your response:

```
EVALUATION_RESULT: pass|fail
EVALUATION_SCORE: 85/100
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: .ailang/state/evaluations/eval_<sprint-id>_round_<n>.json
FEEDBACK_SUMMARY: <one-line summary if failed>
```

**Agent config:**
```yaml
- id: sprint-evaluator
  label: "Sprint Evaluator"
  inbox: sprint-evaluator
  capabilities: [review, test, docs]
  trigger_on_complete: []  # End of chain on pass
  session_continuity: false  # Stateless per round
  approval:
    needs_label: needs-evaluation-approval
    approved_label: evaluation-approved
```

## Core Principles

1. **Independent Judge** — Never evaluate your own work. This skill judges sprint-executor output
2. **Concrete Criteria** — Score against measurable rubric, not subjective impressions
3. **Skeptical by Default** — Tuned toward skepticism; easier to calibrate than making generators self-critical
4. **Actionable Feedback** — Failed evaluations include specific files, functions, and suggestions
5. **Hard Thresholds** — Some failures (tests broken, <50% criteria met) cause automatic rejection
6. **Bounded Iteration** — Max 3 evaluation rounds before escalating to human review

## Evaluation Phases

### Phase 1: Context Loading

1. **Read handoff message** — Extract sprint_id, branch, sprint JSON path, design doc path
2. **Load sprint JSON** from `.ailang/state/sprints/sprint_<id>.json`
3. **Read design doc** — The original spec (the "contract")
4. **Read sprint plan** — The implementation roadmap with milestones
5. **Review git diff** — `git diff dev...<branch>` to see all changes
6. **Check evaluation round** — Is this round 1, 2, or 3? Load prior report if round > 1

### Phase 2: Automated Checks

Run automated quality checks using the evaluation script:

```bash
.claude/skills/sprint-evaluator/scripts/evaluate_sprint.sh <sprint-id> [branch]
```

This runs:
- `make test` — All tests must pass (HARD FAIL if not)
- `make lint` — Linting must be clean
- `make check-file-sizes` — No files exceeding 800 lines
- `make test-coverage-badge` — Coverage metrics

### Phase 3: Acceptance Criteria Verification

```bash
.claude/skills/sprint-evaluator/scripts/check_acceptance_criteria.sh <sprint-id>
```

For each feature in sprint JSON:
1. Read `acceptance_criteria` array
2. Verify `passes: true` for each feature
3. For file-related criteria, verify files exist
4. For test-related criteria, verify test functions exist
5. Score: `(criteria_met / total_criteria) * 30 points`

**HARD FAIL if fewer than 50% of acceptance criteria are met.**

### Phase 3.5: Sprint Artifacts Verification

If a sprint JSON exists (`.ailang/state/sprints/sprint_<id>.json`), verify:
1. All milestones have `passes: true` (or `false` with explanation)
2. All milestones have `completed` timestamps
3. All milestones have `notes` with summaries
4. Sprint `status` is `"completed"`
5. Velocity metrics are populated

If a sprint plan markdown exists, verify:
1. All milestones marked with ✅ (not pending or in-progress)
2. No placeholder text remaining

**Scoring**: Deduct from Code Quality (Phase 4) — up to -5 for incomplete sprint artifacts.
Sprint artifacts are optional (pre-sprint-system implementations won't have them), so missing
artifacts are not penalized — only incomplete/inconsistent ones are.

### Phase 4: Design Fidelity Check (AI Judgment)

Read the design doc and compare against the actual implementation:
- Does the implementation match the stated goals?
- Are the architectural decisions consistent with the design?
- Were any design requirements silently dropped?
- Are there unexpected deviations from the spec?

Score 0-10 based on fidelity to design intent.

### Phase 5: Documentation Completeness

Check:
- **CHANGELOG updated** (5 pts) — New entries under appropriate version
- **Example files created** (5 pts) — `examples/runnable/<feature>.ail` for new features
- **Design doc status** (5 pts) — Updated to reflect implementation state

### Phase 6: Scoring & Report Generation

Apply scoring rubric (see [resources/scoring_rubric.md](resources/scoring_rubric.md)):

| Category | Points | Hard Fail? |
|----------|--------|------------|
| Tests Pass | 20 | Yes |
| Lint Clean | 10 | No |
| Acceptance Criteria | 30 | Yes if <50% |
| Code Quality | 15 | No |
| Documentation | 15 | No |
| Design Fidelity | 10 | No |
| Regression Surface Coverage (conditional) | +10 | Yes if triggered with no analysis |
| Performance Verification (conditional) | +10 | Yes if perf sprint with no profile |
| **Total** | **100** (+20 conditional) | **Pass: 70+** |

**Conditional categories**:
- **Regression Surface Coverage** triggers when the sprint touches `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`. Hard fail if triggered without a Conflict Surface analysis in the design doc and fixture tests for the named "Programs that MUST still work". Catches M-TAINT-TYPES-style silent regressions.
- **Performance Verification** triggers on perf sprints (design doc keywords). Hard fail without before/after profiling data.

Generate report:
```bash
.claude/skills/sprint-evaluator/scripts/generate_report.sh <sprint-id> <score> <result> <round>
```

Report saved to `.ailang/state/evaluations/eval_<sprint-id>_round_<n>.json`

### Phase 7: Pass/Fail Decision

**Determine execution mode** before acting:
- **Local mode** (running in an interactive chat session): invoke skills directly
- **Cloud/coordinator mode** (running via coordinator daemon): use inbox messages

**On PASS (score >= 70, no hard fails):**
1. Move design doc from `design_docs/planned/` to `design_docs/implemented/<version>/`
2. Update design doc status from "Planned" to "Implemented"
3. Output markers with `EVALUATION_RESULT: pass`
4. Post congratulatory summary with score breakdown

**On FAIL (score < 70 or hard fail, round < 3):**

*Local mode — invoke sprint-executor directly:*
1. Generate specific, actionable feedback (see [resources/feedback_templates.md](resources/feedback_templates.md))
2. Invoke the `sprint-executor` skill with the evaluation report path and specific issues
3. After executor completes, re-evaluate (increment round)

*Cloud/coordinator mode — send inbox message:*
1. Generate specific, actionable feedback
2. Send feedback to sprint-executor inbox:

```bash
ailang messages send sprint-executor '{
  "type": "evaluation_feedback",
  "correlation_id": "eval_<sprint-id>_round_<n>",
  "sprint_id": "<sprint-id>",
  "evaluation_round": <n>,
  "score": <score>,
  "issues": [
    {"file": "path/to/file.go", "issue": "description", "suggestion": "how to fix"},
    ...
  ],
  "max_rounds": 3
}' --title "Evaluation Failed (Round <n>/<max>)" --from "sprint-evaluator"
```

3. Output markers with `EVALUATION_RESULT: fail`

**On FAIL (round >= 3):**
1. Escalate to human review
2. Add `needs-human-review` label via coordinator
3. Post summary of all 3 rounds with score progression

## Available Scripts

### `scripts/evaluate_sprint.sh <sprint_id> [branch]`
Orchestrate automated quality checks. Outputs structured JSON with test/lint/coverage results.

### `scripts/check_acceptance_criteria.sh <sprint_id>`
Verify each acceptance criterion from sprint JSON. Outputs per-criterion pass/fail as JSON.

### `scripts/generate_report.sh <sprint_id> <score> <result> <round>`
Create evaluation report JSON at `.ailang/state/evaluations/`.

## Resources

- **Scoring Rubric**: [`resources/scoring_rubric.md`](resources/scoring_rubric.md) — Point allocations and thresholds
- **Report Schema**: [`resources/evaluation_report_schema.md`](resources/evaluation_report_schema.md) — JSON schema for evaluation reports
- **Feedback Templates**: [`resources/feedback_templates.md`](resources/feedback_templates.md) — Actionable feedback examples

## Receiving Handoff from sprint-executor

The evaluator receives an `implementation_complete` message:
```json
{
  "type": "implementation_complete",
  "correlation_id": "eval_M-CACHE_20260326",
  "sprint_id": "M-CACHE",
  "branch_name": "coordinator/task-abc123",
  "sprint_json_path": ".ailang/state/sprints/sprint_M-CACHE.json",
  "design_doc_path": "design_docs/planned/v0_7/M-CACHE.md",
  "files_created": ["internal/cache/store.go"],
  "files_modified": ["internal/server/handler.go"],
  "evaluation_round": 1
}
```

## Execution Modes

The evaluator operates in two modes depending on where it's running:

### Local Mode (interactive chat session)
When running in a user's local Claude Code session:
- **On fail**: Invoke the `sprint-executor` skill directly to fix issues, then re-evaluate
- **On pass**: Move design doc to `implemented/`, update status — no inbox needed
- **Detection**: Default mode. No coordinator markers in the prompt.

### Cloud/Coordinator Mode
When invoked by the coordinator daemon or via agent inbox:
- **On fail**: Send structured feedback message to `sprint-executor` inbox
- **On pass**: Output `EVALUATION_RESULT: pass` markers for coordinator to act on
- **Detection**: Prompt contains coordinator task reference or `implementation_complete` message

## Feedback Loop Architecture

**Local mode:**
```
sprint-evaluator (Round 1)
    ↓
PASS? → move design doc to implemented/, done
FAIL? → invoke sprint-executor skill with issues
            ↓
        sprint-evaluator (Round 2, same session)
            ↓
        PASS? → move design doc, done
        FAIL? → invoke sprint-executor (Round 3)
                    ↓
                PASS? → move design doc, done
                FAIL? → escalate to user for manual review
```

**Cloud/coordinator mode:**
```
sprint-executor completes
        ↓
sprint-evaluator (Round 1)
        ↓
    PASS? → merge approval
    FAIL? → feedback message to sprint-executor
                ↓
        sprint-executor fixes issues
                ↓
        sprint-evaluator (Round 2)
                ↓
            PASS? → merge approval
            FAIL? → feedback message (Round 3)
                        ↓
                    PASS? → merge approval
                    FAIL? → escalate to human (needs-human-review)
```

## Notes

- Evaluation reports are preserved across rounds for audit trail
- Each round's report at `.ailang/state/evaluations/eval_<id>_round_<n>.json`
- The evaluator does NOT modify the sprint JSON — it creates separate evaluation artifacts
- Hard fails (tests broken) cause immediate rejection regardless of score
- Score progression across rounds is tracked in reports for learning
- Evaluator is stateless per round — no session continuity needed

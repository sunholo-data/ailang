# Sprint Evaluation Scoring Rubric

Based on [Anthropic's generator-evaluator architecture](https://www.anthropic.com/engineering/harness-design-long-running-apps): concrete, measurable criteria with hard thresholds.

## Overview

- **Total Points**: 100
- **Pass Threshold**: 70
- **Hard Fail Conditions**: Automatic rejection regardless of score

## Categories

### 1. Tests Pass (20 points) — HARD FAIL

| Score | Condition |
|-------|-----------|
| 20 | `make test` exits 0 |
| 0 | `make test` exits non-zero (**HARD FAIL** — automatic rejection) |

**Why hard fail?** Broken tests mean the implementation is fundamentally incomplete. No amount of documentation or code quality compensates.

### 2. Lint Clean (10 points)

| Score | Condition |
|-------|-----------|
| 10 | `make lint` exits 0 (no errors, no warnings) |
| 5 | `make lint` has warnings only (no errors) |
| 0 | `make lint` has errors |

### 3. Acceptance Criteria (30 points) — HARD FAIL if <50%

Score = `(criteria_met / total_criteria) * 30`

Read from sprint JSON `features[].acceptance_criteria`:
- Each feature with `passes: true` counts all its criteria as met
- Each feature with `passes: false` or `null` counts zero criteria

| Condition | Result |
|-----------|--------|
| >= 50% criteria met | Score calculated normally |
| < 50% criteria met | **HARD FAIL** — automatic rejection |

### 4. Code Quality (15 points)

Start with 15, deduct for issues:

| Issue | Deduction |
|-------|-----------|
| File exceeds 800 lines | -5 per file |
| TODO/HACK/FIXME in new code | -1 per occurrence (max -5) |
| Function exceeds 50 lines | -2 per function (max -5) |

Minimum score: 0 (no negative scores).

### 5. Documentation (15 points)

| Component | Points | Check |
|-----------|--------|-------|
| CHANGELOG updated | 5 | New entries exist under active changelog |
| Example files | 5 | `examples/runnable/<feature>.ail` exists for new features |
| Design doc status | 5 | Design doc reflects implementation state |

### 6. Design Fidelity (10 points)

AI judgment comparing implementation against design doc intent:

| Score | Condition |
|-------|-----------|
| 10 | Implementation fully matches design goals and architecture |
| 7-9 | Minor deviations, well-justified |
| 4-6 | Significant deviations from design, some goals unmet |
| 1-3 | Major architectural divergence from design |
| 0 | Implementation bears little resemblance to design |

## Hard Fail Conditions Summary

These cause automatic rejection regardless of total score:

1. **Tests broken** — `make test` fails
2. **Less than 50% acceptance criteria met** — Core requirements unmet
3. **No commits on implementation branch** — Nothing was implemented

## Score Interpretation

| Range | Meaning | Action |
|-------|---------|--------|
| 90-100 | Excellent | Pass — ready for merge |
| 80-89 | Good | Pass — minor polish optional |
| 70-79 | Acceptable | Pass — meets minimum bar |
| 60-69 | Below threshold | Fail — specific feedback provided |
| 40-59 | Significant gaps | Fail — multiple areas need work |
| 0-39 | Major issues | Fail — fundamental problems |

## Calibration Notes

Per the Anthropic blog post:
- **Tune toward skepticism** — "far more tractable than making a generator critical of its own work"
- **Use concrete evidence** — every deduction must cite a specific file, function, or criterion
- **No vague praise** — "looks good overall" is not acceptable evaluation output
- **Few-shot examples help** — see feedback_templates.md for calibrated examples

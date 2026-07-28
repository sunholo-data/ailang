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
| Sprint JSON incomplete (missing passes/timestamps/notes) | -3 |
| Sprint plan markdown has pending milestones | -2 |

Minimum score: 0 (no negative scores).
Sprint artifacts are optional — deductions only apply when they exist but are incomplete.

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

### 7. Regression Surface Coverage (conditional — shared compilation infrastructure)

**Applies when:** The implementation modifies any of:
- `internal/parser/`, `internal/lexer/`, `internal/ast/`
- `internal/types/`, `internal/elaborate/`, `internal/iface/`
- `internal/codegen/`, `internal/eval/`, `internal/vm/`
- `internal/effects/` (effect-row algebra changes)
- `cmd/ailang/exec.go` and other compilation entry points

**When not triggered:** Skip entirely, no points added or deducted.

**When it IS triggered:** This becomes a **HARD FAIL** category (10 bonus points available).

| Score | Condition |
|-------|-----------|
| 10 | Design doc has filled-in Conflict Surface section; "Programs that MUST still work" fixtures all have explicit regression tests; `make verify-examples` green; AST diff (or equivalent corpus differential) clean OR every diff explicitly justified in commit message |
| 7 | Conflict Surface section present + fixture tests added, but no AST/corpus differential check ran |
| 4 | Some regression tests added (e.g. one example pinned), but Conflict Surface section missing or hand-waved ("no conflicts" with no enumeration) |
| 0 | **HARD FAIL** — touched shared compilation infrastructure with no regression-surface analysis at all |

**What "Conflict Surface filled in" means:**
- Section enumerates the syntactic/semantic positions touched
- Names ≥3 OTHER valid constructs that already live in those positions
- Shows the disambiguation strategy (with token-stream depth or context flag)
- Names 3-5 existing programs (in `examples/`, `std/`, or external consumers) that exercise the position
- Calls out anything that intentionally breaks (or affirms nothing intentionally breaks)

A section that says "no conflicts" without enumeration is treated as missing.

**What counts as a fixture test:**
- A unit test that pins the parse output / typecheck output / eval output of a real program
- Located alongside the implementation (`*_test.go` or equivalent)
- Named after the program / motoko-style consumer it protects (e.g. `TestRefinementVsFunctionBodyDisambiguation/motoko_agent_is_extension_tool_call_shape`)

**Why hard fail?** [M-PARSER-REFINEMENT-LOOKAHEAD](../../../changelogs/v0.10-v0.17-bytecode-vm.md) (v0.15.2) fixed a regression that shipped 18 months earlier in M-TAINT-TYPES (v0.14.3). Adding `T{not LABEL}` syntax silently broke `func ... -> bool { not f(x) }` because nobody enumerated what else used `{` after a type. The original tests covered the new feature exhaustively but never asked "what existing valid programs become newly invalid?" External consumers (motoko_agent fork on v0.13.0) hit ~14 mis-parses when migrating. **The cost of skipping this gate is paid by users, not by the team that introduces the regression — and it's paid late.** Hard fail because the failure mode is "looks fine in isolation, breaks the world in aggregate."

### 8. Performance Verification (conditional — perf sprints only)

**Applies when:** Design doc or sprint JSON contains performance goals (keywords: "performance", "speedup", "latency", "benchmark", "CPU profile", or `performance_goals` field in sprint JSON).

**When not a perf sprint:** Skip entirely, no points added or deducted.

**When it IS a perf sprint:** This becomes a **HARD FAIL** category (10 bonus points available).

| Score | Condition |
|-------|-----------|
| 10 | Before AND after CPU profiles captured; targeted bottleneck confirmed eliminated in profile |
| 7 | Before/after wall-clock benchmarks with improvement shown, but no CPU profile |
| 3 | Only after benchmarks (no baseline comparison) |
| 0 | **HARD FAIL** — Performance sprint with no profiling data at all |

**What counts as valid profile data:**
- `ailang run -cpuprofile` before and after optimization
- `go tool pprof -top -cum` output showing the targeted function's CPU% change
- Wall-clock benchmarks on the motivating use case (e.g., docparse EPUB)

**How to run (for AILANG):**
```bash
# Before (on baseline branch):
ailang run -cpuprofile /tmp/before.prof -caps IO,FS,Env <benchmark_file>
go tool pprof -top -cum /tmp/before.prof | head -20

# After (on implementation branch):
ailang run -cpuprofile /tmp/after.prof -caps IO,FS,Env <benchmark_file>
go tool pprof -top -cum /tmp/after.prof | head -20
```

**Why hard fail?** M-BYTECODE-XML-BUILTINS wired 17 builtins for 0% speedup because
nobody profiled WHERE CPU time was spent. The real bottleneck (runtime.Stack at 42%)
was completely unrelated. Without before/after profiling, performance sprints can
deliver zero value while appearing successful on structural metrics (EvalOnly counts,
test pass rates).

## Hard Fail Conditions Summary

These cause automatic rejection regardless of total score:

1. **Tests broken** — `make test` fails
2. **Less than 50% acceptance criteria met** — Core requirements unmet
3. **No commits on implementation branch** — Nothing was implemented
4. **Performance sprint with no profiling data** — Cannot verify optimization worked
5. **Shared compilation infrastructure touched without regression-surface analysis** — Triggers when the sprint modifies parser/lexer/typechecker/codegen/effects paths. The Conflict Surface section in the design doc must be filled with enumerated alternatives, AND fixture tests must exist for the named "Programs that MUST still work" entries. Failure mode: M-TAINT-TYPES-style silent regressions.

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

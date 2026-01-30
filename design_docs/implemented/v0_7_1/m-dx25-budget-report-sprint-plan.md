# Sprint Plan: M-DX25 Scoped Budgets with Dual Counters & Budget Reporting

## Summary

Add `--budget-report` flag for effect budget observability, then implement scoped dual-counter budgets so that functions with declared `@limit` annotations charge the caller their declared semantic cost while physical counts are always tracked and reported.

**Duration:** 5 days
**Dependencies:** M-CAPABILITY-BUDGETS (implemented v0.6.2)
**Risk Level:** Medium (semantic change to budget counting model, but strictly additive via pass-through default)
**Total Estimated LOC:** ~700
**Design Doc:** [m-dx25-budget-report.md](m-dx25-budget-report.md)

## Current Status Analysis

### Completed Recently
- M-DX24: Developer Experience Improvements (if-then-else blocks, record type examples, pattern matching tests)
- M-BUILTIN-SAFETY: Defensive type checking in builtin implementations
- `internal/apiserver/`: Embedded webserver (~535 LOC)
- Coordinator multi-agent pipeline operational (design-doc-creator -> sprint-planner -> sprint-executor)

### Velocity
- Recent average: ~150-200 LOC/day of implementation + tests (from M-DX24, M-BUILTIN-SAFETY)
- Estimated capacity: ~750-1000 LOC for 5-day sprint
- Budget for this sprint: ~700 LOC estimated (within capacity with buffer)

### Existing Infrastructure (Key Files)

| File | LOC | Role |
|------|-----|------|
| `internal/effects/budget.go` | 202 | BudgetContext: limits, used, CheckAndConsume |
| `internal/effects/context.go` | ~310 | EffContext, RequireCapWithBudget, WithBudgetLimits |
| `internal/effects/errors.go` | 97 | BudgetExhaustedError, CapabilityError |
| `internal/effects/budget_test.go` | 341 | 11 tests covering all BudgetContext methods |
| `internal/eval/eval_evaluator.go` | ~500 | CallFunction with budget scope push/pop |
| `internal/eval/eval_operations.go` | ~400 | App evaluation with budget scope push/pop |
| `internal/eval/eval_expressions.go` | ~200 | extractEffectBudgets from CoreTypeInfo |
| `internal/eval/value.go` | ~180 | FunctionValue.EffectBudgets field |
| `internal/runtime/builtins.go` | ~100 | RequireCapWithBudget on every builtin call |
| `cmd/ailang/main.go` | ~600 | --no-budgets flag pattern to follow |

### What Already Works
- `BudgetContext` tracks `limits` and `used` per effect
- `CheckAndConsume()` enforces limits and tracks usage even when unlimited
- `WithBudgetLimits()` creates fresh budget scope on function call
- Evaluator pushes/pops budget scope at call boundaries (both `CallFunction` and App path)
- `extractEffectBudgets()` reads `@limit` from `TFunc2.EffectRow.Budgets`
- `--no-budgets` flag disables enforcement

### What's Missing
1. No way to see usage after successful execution (reporting)
2. No dual counters (physical vs semantic)
3. No per-function attribution of budget consumption
4. Callee budget scope replaces caller scope entirely (no caller charging on return)
5. No stdlib budget declarations

## Proposed Milestones

### M1: Budget Report Infrastructure
**Goal:** Add `--budget-report` flag that prints per-function, per-effect usage after execution. Works with current counting model (physical only). Provides immediate value before semantic changes.

**Estimated:** ~120 LOC implementation + ~80 LOC tests = ~200 LOC
**Duration:** 1 day

**Tasks:**
1. Add `BudgetReport` struct to `internal/effects/budget.go`:
   - Per-function, per-effect usage tracking (`map[string]map[string]int`)
   - `RecordUsage(funcName, effect, count)` method
   - `Summary()` for flat text output
2. Create `internal/effects/report.go`:
   - `FormatReport(report *BudgetReport) string` (text format)
   - `FormatReportJSON(report *BudgetReport) ([]byte, error)` (JSON format)
3. Add `--budget-report` flag to `cmd/ailang/main.go`:
   - Follow `--no-budgets` pattern (Bool flag, passed to `runFile`)
   - Print report to stderr after execution completes
   - On `BudgetExhaustedError`, include partial report in error output
4. Wire up evaluator to record per-function usage:
   - In `CallFunction` and App path: after callee returns, record callee's budget consumption
   - Pass report collector through `EffContext` (new field)
5. Add `--budget-report` to help text in `cmd/ailang/help.go`
6. Tests: report struct, text/JSON formatting, CLI flag

**Acceptance Criteria:**
- [ ] `ailang run --budget-report --caps IO,FS file.ail` prints usage summary to stderr
- [ ] Report shows per-function, per-effect usage counts
- [ ] Report included in BudgetExhaustedError output
- [ ] `--budget-report=json` produces parseable JSON
- [ ] Existing budget tests pass (no regression)
- [ ] Linting clean

**Risks:**
- Threading report collector through evaluator without import cycles: Use interface pattern (like `BudgetEnforcer`)

---

### M2: Dual Counters (Physical + Semantic)
**Goal:** Extend `BudgetContext` to track both physical (actual builtin calls) and semantic (declared budget) counts. Physical always counted; semantic counted when budget scope is active.

**Estimated:** ~80 LOC implementation + ~60 LOC tests = ~140 LOC
**Duration:** 0.5 days

**Tasks:**
1. Add `physicalUsed map[string]int` to `BudgetContext`:
   - `CheckAndConsume` increments both `used` (semantic enforcement) and `physicalUsed` (always)
   - `PhysicalUsed(effect) int` accessor
2. Update `BudgetReport` to show dual counters:
   - Text: `FS semantic 3/5  FS physical 19`
   - JSON: `{"semantic": {"used": 3, "limit": 5}, "physical": {"used": 19}}`
3. Update `BudgetExhaustedError` to include physical count:
   - `Physical int` field
   - Error message: `effect 'FS' budget exhausted: semantic limit=5, used=5 (physical: 19)`
4. Tests: dual counter tracking, physical always increments, report formatting

**Acceptance Criteria:**
- [ ] `BudgetContext` tracks physical and semantic counts independently
- [ ] Physical counts increment even when semantic budget is not set
- [ ] Report shows both counters per function per effect
- [ ] BudgetExhaustedError shows physical count
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- Low risk. Pure data addition to existing struct. No behavioral change.

---

### M3: Scoped Budget with Caller Charging
**Goal:** When a function with declared `@limit=k` is called, create a callee scope for internal operations. On return, charge the caller `k` semantic units (the callee's declared budget). Functions without declared budgets use pass-through (like today).

**Estimated:** ~100 LOC implementation + ~80 LOC tests = ~180 LOC
**Duration:** 1.5 days

**Tasks:**
1. Modify `WithBudgetLimits()` in `internal/effects/context.go`:
   - Current: creates fresh budget replacing caller's
   - New: creates callee scope that isolates physical ops from caller
   - Store caller's budget reference for charging on return
2. Add `BudgetScope` concept:
   - `PushScope(calleeBudgets map[string]int)` - creates callee scope
   - `PopScope() map[string]int` - returns callee's declared budgets to charge caller
   - Internal builtins decrement top-of-stack scope
3. Modify evaluator call sites (`eval_evaluator.go:144-152` and `eval_operations.go:56-64`):
   - Current: `e.effContext = enforcer.WithBudgetLimits(fn.EffectBudgets)` then restore
   - New: push callee scope, evaluate body, pop scope, charge caller declared amounts
   - Pass-through: if `fn.EffectBudgets` is empty/nil, don't push scope (existing behavior)
4. Update `RequireCapWithBudget` to decrement top-of-stack scope:
   - If scope stack has entries, decrement callee scope physical counter
   - If no scope stack (pass-through), decrement caller directly (like today)
5. Tests:
   - Scoped function: caller charged declared amount, not internal count
   - Pass-through function: caller charged directly (regression test)
   - Nested scopes: inner callee charged to outer callee, outer charged to caller
   - BudgetExhaustedError at correct scope level

**Acceptance Criteria:**
- [ ] Function with `@limit=k` charges caller `k` semantic units on return
- [ ] Internal builtin calls within scoped function don't charge caller
- [ ] Functions without `@limit` use pass-through (caller charged directly)
- [ ] Nested scoped functions compose correctly
- [ ] Budget report shows semantic + physical per scope
- [ ] All existing budget tests pass (pass-through preserves current behavior)
- [ ] Linting clean

**Risks:**
- Scope stack imbalance if evaluator errors mid-call: Use defer-based pop (evaluator already uses this pattern for `oldEffContext` restoration)
- Import cycle between eval and effects: Already solved via `BudgetEnforcer` interface

---

### M4: Minimum Budgets (@min)
**Goal:** Add `@min=N` syntax for minimum budget verification — ensure effects were actually exercised (not skipped/cached). Checked on function return.

**Estimated:** ~60 LOC implementation + ~50 LOC tests = ~110 LOC
**Duration:** 0.5 days

**Tasks:**
1. Add `@min=N` parsing to effect row in `internal/types/effects.go`:
   - Extend `ElaborateEffectRowWithBudgets()` to parse `@min=N`
   - Store in `Row.MinBudgets map[string]*int` (parallel to `Row.Budgets`)
2. Add `minLimits` field to `BudgetContext`:
   - Track minimum required usage per effect
   - `CheckMinimum(effect) error` - verify physical count >= min
3. Add `BudgetUnderrunError` to `internal/effects/errors.go`:
   - Similar to `BudgetExhaustedError` but for underrun
   - `effect 'Net' budget underrun: min=1, actual=0`
4. Check minimums on scope pop in evaluator:
   - After `PopScope()`, verify all min requirements met
   - Return `BudgetUnderrunError` if any min not satisfied
5. Update report to show min bounds:
   - `FS physical 3 (min=1, max=5)`
6. Tests:
   - Min-only budget (no max)
   - Min + max combined
   - Underrun detection on return
   - Pass-through functions don't check min

**Acceptance Criteria:**
- [ ] `@min=N` syntax parses correctly in effect rows
- [ ] `BudgetUnderrunError` returned if physical count < min on scope exit
- [ ] Min-only budgets work (`! {Net @min=1}` without max)
- [ ] Combined min+max works (`! {Net @min=1, @limit=5}`)
- [ ] Budget report shows min bounds when set
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- Parser changes to effect row syntax: Follow existing `@limit` pattern closely

---

### M5: Stdlib Annotations & Integration
**Goal:** Add budget declarations to stdlib functions that caused pain in ecommerce demos. Update examples and documentation.

**Estimated:** ~40 LOC implementation + ~30 LOC tests = ~70 LOC
**Duration:** 0.5 days

**Tasks:**
1. Identify stdlib functions needing budget annotations:
   - Functions called via `demos/ecommerce` that triggered BudgetExhaustedError
   - Focus on FS-heavy functions (file discovery, credential reading)
   - Set conservative upper bounds
2. Add budget type annotations to stdlib effect declarations:
   - This may require builtin spec changes if budgets need to be set in Go code
   - OR: document recommended caller budgets in teaching prompt
3. Create `examples/runnable/budget_report.ail`:
   - Demonstrates `--budget-report` output
   - Shows scoped vs pass-through behavior
4. Update existing budget examples:
   - `examples/runnable/effect_budgets.ail` - add scoped budget example
   - `examples/runnable/effect_budgets_rand.ail` - verify still works
5. Update CHANGELOG.md:
   - `--budget-report` flag
   - Scoped dual-counter budget model
   - Behavioral change for functions with declared budgets
6. Update teaching prompt (`ailang prompt`):
   - Document scoped budget semantics
   - Document `--budget-report` flag
   - Update budget section with dual-counter examples

**Acceptance Criteria:**
- [ ] Stdlib FS-heavy functions have budget annotations (or documented recommendations)
- [ ] Budget report example works: `ailang run --budget-report --caps IO,FS examples/runnable/budget_report.ail`
- [ ] Existing budget examples pass
- [ ] CHANGELOG updated
- [ ] `make verify-examples` passes
- [ ] Linting clean

**Risks:**
- Stdlib budget annotations may require type system changes if `@limit` can't be attached to builtin specs: Fallback is documenting recommended budgets in the teaching prompt

## Success Metrics
- Test coverage: >85% for `internal/effects/budget.go` and `report.go` (new code)
- Examples passing: budget_report.ail + existing budget examples
- Documentation: CHANGELOG.md, help.go, teaching prompt
- All tests passing: `make test`
- All linting passing: `make lint`
- Ecommerce demos: budget values become intuitive after stdlib annotations

## Dependencies
- None blocking. M-CAPABILITY-BUDGETS already implemented.
- Stdlib budget annotation approach depends on whether builtin specs support `@limit` metadata (may need minor spec.go change).

## Open Questions
- **Stdlib budget attachment mechanism:** Can we attach `@limit` metadata to `builtins.Spec` entries, or do we need type-level annotations in the surface language? Investigate during M4.
- **Budget report verbosity:** Start with flat summary only. Defer tree view to follow-up if demand exists.

## Notes
- Pass-through default ensures zero regression. All behavioral changes are opt-in via `@limit` declarations.
- The evaluator's existing push/pop pattern for `oldEffContext` at call boundaries is the foundation for scoped budgets. M3 extends this rather than replacing it.
- Physical counts are always tracked (even with `--no-budgets`), so `--budget-report` works in both modes.

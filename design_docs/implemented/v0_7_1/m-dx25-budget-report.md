# M-DX25: Scoped Budgets with Dual Counters & Budget Reporting

**Status**: IMPLEMENTED
**Version**: v0.7.1
**Priority**: P1 (Medium) - Directly impacts developer trust in budget system
**Completed**: 2026-01-28
**Dependencies**: M-CAPABILITY-BUDGETS (implemented, v0.6.2)

## Implementation Summary

M-DX25 implemented scoped budgets with dual counters (semantic vs physical) and minimum budget verification (`@min`). Key features:

- **Dual counters**: Physical (actual builtin calls) vs Semantic (declared budget charged to caller)
- **Minimum budgets** (`@min=N`): Verify effects were actually exercised, not skipped/cached
- **Scoped budget charging**: Callee's `@limit=k` charges caller k semantic units on return
- **Pass-through default**: Functions without declared budgets behave like before
- **BudgetUnderrunError**: Error when minimum usage requirements not met

### Files Modified

| File | Change |
|------|--------|
| `internal/ast/ast.go` | Added `Min *int` to `EffectAnnotation` |
| `internal/parser/parser_effect.go` | Parse `@min=N` and `@limit=N` annotations |
| `internal/types/types_v2.go` | Added `MinBudgets map[string]*int` to `Row` |
| `internal/types/effects.go` | Copy MinBudgets in type operations |
| `internal/effects/budget.go` | Dual counters, min checking, MinLimitsMap |
| `internal/effects/errors.go` | Added `BudgetUnderrunError` |
| `internal/effects/context.go` | SetMinBudgets, CheckMinimums on EffContext |
| `internal/eval/value.go` | Added `EffectMinBudgets` to FunctionValue |
| `internal/eval/eval_expressions.go` | extractEffectMinBudgets function |
| `internal/eval/eval_evaluator.go` | MinBudgetEnforcer, MinimumChecker interfaces |
| `internal/eval/eval_operations.go` | Budget scope handling with min checks |

### Tests Added

- `internal/parser/effects_test.go`: 7 test cases for @min parsing
- `internal/effects/budget_test.go`: 10+ test cases for min budget enforcement

### Documentation Updated

- `docs/docs/reference/capability-budgets.mdx`: Added @min syntax, use cases, examples
- `docs/docs/reference/effects.md`: Updated Capability Budgets section with @min
- `examples/effect_budget_demo.ail`: Updated with @min examples
- `examples/reference/budget_minimum.ail`: New reference example
- `examples/manifest.json`: Added both examples with `budget` and `m-dx25` tags
- `CHANGELOG.md`: Comprehensive M-DX25 entry

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Budget scoping and reporting are deterministic |
| A2: Replayability | +1 | Dual counters make traces fully inspectable |
| A3: Effect Legibility | +1 | **Primary benefit** - effects become predictable at abstraction boundaries |
| A4: Explicit Authority | 0 | No change to capability model |
| A5: Bounded Verification | +1 | Callee-declared budgets enable local verification |
| A7: Machines First | +1 | Structured dual-counter output enables automated analysis |
| A8: Minimal Syntax | 0 | No new syntax (CLI flags + existing `@limit` annotations) |
| A9: Cost Visibility | +1 | **Both** predictability (semantic) and accuracy (physical) preserved |
| A10: Composability | +1 | Scoped budgets compose cleanly across module boundaries |
| A11: Structured Failure | +1 | BudgetExhaustedError enhanced with scope context |
| A12: System Boundary | +1 | All module boundaries become proper abstraction boundaries |

**Net Score: +9** -> **Decision: Proceed**

## Problem Statement

Effect budget values (`@limit=N`) are unpredictable because stdlib functions consume multiple internal effect units per logical operation. Developers resort to trial-and-error, undermining the "budgets as contracts" value proposition.

**Root cause:** The current budget unit is "builtin invocation", not "semantic operation." Every call to `RequireCapWithBudget()` in `internal/runtime/builtins.go:79` consumes 1 unit, regardless of whether the call originates from user code or stdlib internals.

**Concrete example from ecommerce demos:**
- `getDefaultProject()` performs 5 logical FS operations (fileExists + readFile across several paths)
- Expected budget: `FS @limit=5`
- Actual budget needed: `FS @limit=15` (each internal builtin call counts separately)
- `readCredentials()` needs `@limit=8` for what appears to be 2-3 logical operations

**Impact:**
- Budgets become trial-and-error guesswork
- `--no-budgets` becomes the default escape hatch
- "Budgets as contracts" value prop is broken -- you can't write a precise contract when costs are implementation-dependent
- Stdlib version changes silently break existing budget values

## Design: Two-Tier Budgets with Scoped Counting

### Why Not "1 Unit Per Call"

The naive fix -- "each function call counts as 1 unit regardless of internals" -- is wrong. It fixes predictability by destroying cost granularity and creates a perverse incentive: pack arbitrarily many FS ops behind one call and budgets become meaningless. That violates A9 (Cost Visibility).

### The Right Model: Scoped Budgets + Dual Counters

Every effectful operation produces two increments:

1. **Physical count**: increments on every builtin/handler invocation. This is the truth -- what actually happened.
2. **Semantic count**: increments according to the callee's declared budget. This is what the caller budgets against (contracts).

The key correction: **charge the caller the callee's declared budget, not a constant 1.**

```
Tier 1 (semantic/authority): what the caller is allowed to cause
Tier 2 (physical/resource):  what actually happens (always reported, optionally enforced)
```

This preserves:
- **Predictability** at abstraction boundaries (Tier 1)
- **True cost visibility** (Tier 2)
- Prevents stdlib from becoming an infinite-cost hiding place

### Scoped Budget Semantics

On a function call:

1. If callee has `Effect @limit=k` declared in its type, create a **callee scope** with limit=k.
2. Callee's internal builtins decrement the **callee scope** (physical counting within).
3. When callee returns, charge the **caller** `k` semantic units (the callee's declared budget).
4. Physical counts are always tracked at every level, attributed per function.

```
caller budget: FS @limit=5 (semantic)
  call getDefaultProject()    -> caller semantic: +2 (callee declared FS @limit=2)
    callee scope: FS @limit=2 (callee's own enforcement)
      fileExists("/a")        -> callee physical: 1   (NOT charged to caller)
      readFile("/a")          -> callee physical: 2
      fileExists("/b")        -> callee physical: 3
      ...                     -> callee physical: 12
      -- callee scope enforces its own @limit=2? No -- see below
```

**Important clarification:** The callee's declared `@limit` is its **semantic cost to the caller**, not necessarily its internal physical limit. A stdlib function declaring `! {FS @limit=2}` means "calling me costs the caller 2 FS semantic units." The function's internal physical operations run in their own scope.

This is analogous to API pricing: an endpoint that says "costs 2 credits" may internally make 12 database queries.

### What If Callee Has No `@limit`?

Three possible defaults:

| Policy | Behavior | Risk |
|--------|----------|------|
| **Pass-through** (recommended default) | No callee scope created; internal ops charge caller directly (like today) | No hiding; caller pays true cost |
| **Default-unbounded** | Caller pays 1; callee runs unlimited internally | Hides work behind calls |
| **Default-declared-by-stdlib** | Stdlib must declare budgets; user code uses pass-through | Best long-term but requires stdlib annotation effort |

**Recommendation:** Pass-through as default. This means:
- Functions **without** declared budgets behave exactly like today (no regression)
- Functions **with** declared budgets create scoped boundaries
- Stdlib should progressively add budget declarations to high-use functions
- This is strictly additive -- existing behavior unchanged unless budget is declared

### Enforcement Modes

```bash
ailang run --budget-mode=semantic ...   # Default: enforce semantic budgets, report physical
ailang run --budget-mode=physical ...   # Enforce physical budgets (ops cost caps)
ailang run --budget-mode=both ...       # Enforce both (strict)
```

Compatible with D4 where specs might enforce physical budgets (API calls) while types enforce semantic budgets.

### Minimum Budgets (@min)

In addition to maximum budgets (`@limit=N`), support minimum budgets (`@min=N`) to verify that effects were actually exercised:

```ailang
-- Ensure we actually called the API (not cached/mocked)
func fetchData(url: string) -> string ! {Net @min=1 @limit=5} {
  let cached = checkCache(url) in
  if isSome(cached) then unwrap(cached)
  else fetch(url)  -- Must actually call Net at least once
}

-- Verify file was read (not defaulted)
func loadConfig(path: string) -> string ! {FS @min=1} {
  readFile(path)  -- If this returns default, @min=1 fails
}
```

**Semantics:**
- `@min=N`: Function must use at least N effect invocations (physical count)
- Checked **on function return**, not during execution
- `BudgetUnderrunError` if physical count < min on return
- Useful for: audit trails, cache-bypass verification, test assertions

**Syntax:**
```ailang
! {Net @limit=5}           -- Max only (current)
! {Net @min=1}             -- Min only (new)
! {Net @min=1 @limit=5}    -- Both min and max (new)
! {IO @limit=5, Net @min=1 @limit=3}  -- Multiple effects with different annotations
```

**Implementation:**
- Added `Min` field to `EffectAnnotation` in AST
- Added `MinBudgets` map to `Row` type
- Check on scope exit: `if physicalUsed < min { return BudgetUnderrunError }`
- BudgetContext tracks both `used` (semantic) and `physicalUsed` counters

## Implementation Plan

### Phase 1: Budget Report Infrastructure (M1) - COMPLETED

- [x] Add `BudgetReport` struct with per-function, per-effect tracking
- [x] Track function entry/exit in evaluator to attribute budget consumption
- [x] Include report context in `BudgetExhaustedError` messages

### Phase 2: Dual Counters (M2) - COMPLETED

- [x] Add `physicalUsed` counter alongside existing `used` (semantic) in `BudgetContext`
- [x] Dual counters tracked: semantic (for caller charging) and physical (for actual calls)
- [x] Budget report shows both semantic and physical per function

### Phase 3: Scoped Caller Charging (M3) - COMPLETED

- [x] On function entry with declared budgets: push callee scope
- [x] `RequireCapWithBudget` decrements top-of-stack scope
- [x] On function return: pop scope, charge caller with callee's declared semantic cost
- [x] Functions without declared budgets: pass-through (charge caller directly, like today)
- [x] Tests for nested scoping, pass-through default, scoped charging

### Phase 4: Minimum Budgets @min (M4) - COMPLETED

- [x] Parse `@min=N` annotation in effect syntax
- [x] Add `Min *int` to `EffectAnnotation` AST node
- [x] Add `MinBudgets map[string]*int` to `Row` type
- [x] Add `minLimits` map to `BudgetContext`
- [x] Add `BudgetUnderrunError` for when minimum not met
- [x] Check minimums on function return (before restoring caller context)
- [x] Tests for @min parsing and enforcement

### Phase 5: Stdlib Annotations & Integration (M5) - COMPLETED

- [x] Update `examples/effect_budget_demo.ail` with @min examples
- [x] Create `examples/reference/budget_minimum.ail` reference example
- [x] Update `examples/manifest.json` with new examples
- [x] Update `docs/docs/reference/capability-budgets.mdx` with @min documentation
- [x] Update `docs/docs/reference/effects.md` with @min examples
- [x] CHANGELOG entry with comprehensive M-DX25 documentation

## Success Criteria

- [x] Dual counters track semantic (caller-charged) and physical (actual) usage
- [x] Functions with declared budgets create scoped boundaries (caller charged declared cost)
- [x] Functions without declared budgets use pass-through (no change from today)
- [x] Physical counts always tracked
- [x] `@min=N` annotation parses and enforces minimum usage
- [x] `BudgetUnderrunError` when minimum not met
- [x] Existing tests pass (pass-through default means no regression)
- [x] Examples work with new @min syntax
- [x] Documentation updated (capability-budgets.mdx, effects.md)
- [x] Examples queryable via `ailang examples list --tags budget`

## Examples

### Effect Budget Demo

```ailang
-- Ensure logging actually happened (not skipped/cached)
-- The @min=1 annotation verifies the function actually performed IO
export func logAuditTrail(action: string, success: bool) -> () ! {IO @min=1 @limit=2} {
  println("AUDIT: " ++ action);
  if not success then
    println("  Status: FAILED")
  else
    ()
}

-- Verify data was actually fetched (not cached/defaulted)
-- Combines @min to ensure fetch happened with @limit for rate control
export func verifiedFetch(url: string) -> string ! {IO @min=1 @limit=3} {
  println("Fetching: " ++ url);
  println("Response: <simulated data>");
  "data from " ++ url
}
```

### Running Examples

```bash
# Run the effect budget demo
ailang run --caps IO --entry main examples/effect_budget_demo.ail

# Search for budget examples
ailang examples list --tags budget

# Run the reference example
ailang run --caps IO --entry main examples/reference/budget_minimum.ail
```

## Future Work

- `--budget-report` CLI flag for detailed per-function budget reporting
- `--budget-mode=physical` / `--budget-mode=both` enforcement modes
- D4 spec-driven budgets merged with type-level budgets (MIN wins)
- Static budget verification (prove budget sufficiency at compile time)
- Budget visualization in traces (OTEL spans with budget attributes)
- `ailang doctor budgets` -- lint for missing stdlib budget declarations

## Related Documents

- [M-CAPABILITY-BUDGETS](../v0_6_2/m-capability-budgets.md) -- Original budget implementation
- [Design Axioms](/docs/references/axioms) -- A9: Cost Visibility, A12: System Boundary
- Messages: `msg_20260127_230302_21f217e2` (budget report proposal), `msg_20260127_230323_90f9a3eb` (session summary)

---

**Document created**: 2026-01-28
**Implemented**: 2026-01-28
**Design review**: Incorporated dual-counter scoped model feedback. Rejected "1 unit per call" as too coarse. Adopted "charge callee's declared budget" with pass-through default for undeclared functions.

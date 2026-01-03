# M-PATTERN-GUARDS: Pattern Guard Evaluation in Codegen

**Status**: ✅ Implemented
**Target**: v0.6.2
**Priority**: P1 (High) - Feature documented as working but silently broken
**Estimated**: 4-6 hours (~150-200 LOC)
**Actual**: ~15 LOC, <30 minutes
**Implemented**: 2025-12-29
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact - guards are deterministic boolean expressions |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | +1 | Guards must be pure (enforced by type checker) |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Guards can be analyzed statically |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Enables conditional pattern matching for AI-generated code |
| A8: Minimal Syntax | 0 | Syntax already exists and is parsed |
| A9: Cost Visibility | 0 | Guard evaluation cost is visible |
| A10: Composability | +1 | Guards compose with all pattern types |
| A11: Structured Failure | 0 | Non-matching guards fall through (well-defined) |
| A12: System Boundary | 0 | No boundary crossing |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Guards must evaluate same result for same input
- [x] A3 (Effects): Type checker enforces guards have no effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Critical for AI code generation patterns

## Problem Statement

Pattern guards are parsed, type-checked, and work in the evaluator/REPL, but **the Go code generator ignores them completely**. This causes silent incorrect behavior when running compiled AILANG programs.

**Current State:**
- Parser: Correctly parses `x if x > 0 => body` syntax
- Elaboration: Guards elaborated to Core AST
- Type Checker: Guard type verified as `bool`
- Evaluator: Guards evaluated, arm skipped if false
- **Codegen: Guards IGNORED - first structurally matching arm always wins**

**Evidence (runtime output):**
```ailang
match 5 {
  x if x > 10 => "big",     -- 5 > 10 is FALSE, should skip
  x if x > 0 => "positive", -- 5 > 0 is TRUE, should match
  x => "other"
}
-- Expected: "positive"
-- Actual: "big" (first arm wins, guard ignored)
```

**Impact:**
- Users cannot rely on pattern guards in compiled programs
- Examples show expected behavior but runtime differs
- Breaks fundamental pattern matching semantics

## Goals

**Primary Goal:** Emit guard checks in generated Go code

**Success Metrics:**
- `examples/runnable/guards_basic.ail` produces correct output
- Guards with pattern bindings work: `(x, y) if x > y => ...`
- Guards with comparison operators work: `x if x > 0 => ...`
- All existing eval tests still pass (regression check)

## Solution Design

### Overview

Add guard evaluation to all three codegen paths:
1. `generateMatchIfElse` - list/tuple patterns (if-else chains)
2. `generateMatchArmValueSwitch` - literal patterns (switch cases)
3. `generateMatchArmADT` - ADT constructor patterns (Kind switch)

### Architecture

**File:** `internal/gen/golang/codegen_match.go`

The guard must be checked AFTER structural pattern match but BEFORE returning the arm body:

```go
// Current (broken):
if <pattern-condition> {
    return <body>
}

// Fixed:
if <pattern-condition> {
    if <guard-expr> {      // NEW: evaluate guard
        return <body>
    }
    // fall through to next arm
}
```

### Implementation Plan

**Phase 1: Add Guard Helper** (~1 hour, ~30 LOC)
- [ ] Create `generateGuardCheck(guard core.CoreExpr) error` function
- [ ] Guard evaluates to bool - no type assertion needed
- [ ] Handle nil guard (no-op)

**Phase 2: If-Else Chain Guards** (~1 hour, ~40 LOC)
- [ ] Modify `generateMatchIfElse()` to emit guard checks
- [ ] Pattern: `if <pattern> && <guard> { return body }`
- [ ] Handle binding access in guard expressions

**Phase 3: Value Switch Guards** (~1 hour, ~40 LOC)
- [ ] Modify `generateMatchArmValueSwitch()` to emit guard checks
- [ ] Convert switch cases with guards to if-else (switch can't have guards)
- [ ] Or: emit `case X: if !guard { fallthrough }; return body`

**Phase 4: ADT Switch Guards** (~1 hour, ~40 LOC)
- [ ] Modify `generateMatchArmADT()` to emit guard checks
- [ ] Bindings must be generated BEFORE guard evaluation
- [ ] Pattern: `case Kind_X: <bindings>; if !guard { fallthrough }; return body`

**Phase 5: Tests & Examples** (~1 hour, ~50 LOC)
- [ ] Add codegen tests in `codegen_match_test.go`
- [ ] Verify `examples/runnable/guards_basic.ail` output
- [ ] Add complex guard examples (nested patterns, tuple guards)

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_match.go` - Add guard evaluation (~120 LOC)
- `internal/gen/golang/codegen_match_test.go` - Add guard codegen tests (~50 LOC)

**No new files needed.**

## Examples

### Example 1: Basic Guard (if-else path)

**AILANG:**
```ailang
match 5 {
  x if x > 10 => "big",
  x if x > 0 => "positive",
  x => "other"
}
```

**Generated Go (current - broken):**
```go
func() string {
    _scrutinee := int64(5)
    if true { // x always matches
        x := _scrutinee
        return "big"  // WRONG: guard ignored
    } else if true {
        x := _scrutinee
        return "positive"
    } else {
        x := _scrutinee
        return "other"
    }
}()
```

**Generated Go (fixed):**
```go
func() string {
    _scrutinee := int64(5)
    x := _scrutinee // bind once, visible to all guards
    if x > int64(10) {
        return "big"
    } else if x > int64(0) {
        return "positive"
    } else {
        return "other"
    }
}()
```

### Example 2: Tuple Pattern with Guard

**AILANG:**
```ailang
match (10, 5) {
  (a, b) if a > b => "first bigger",
  (a, b) if a < b => "second bigger",
  _ => "equal"
}
```

**Generated Go (fixed):**
```go
func() string {
    _scrutinee := []interface{}{int64(10), int64(5)}
    if len(_scrutinee.([]interface{})) == 2 {
        a := _scrutinee.([]interface{})[0].(int64)
        b := _scrutinee.([]interface{})[1].(int64)
        if a > b {
            return "first bigger"
        } else if a < b {
            return "second bigger"
        }
    }
    return "equal"
}()
```

### Example 3: ADT Pattern with Guard

**AILANG:**
```ailang
match opt {
  Some(x) if x > 0 => "positive value",
  Some(x) => "non-positive value",
  None => "no value"
}
```

**Generated Go (fixed):**
```go
func() string {
    _scrutinee := opt
    _adt := _scrutinee.(*Option)
    switch _adt.Kind {
    case Kind_Option_Some:
        x := _adt.Some.Value0
        if x.(int64) > int64(0) {
            return "positive value"
        }
        return "non-positive value"
    case Kind_Option_None:
        return "no value"
    default:
        panic("non-exhaustive match")
    }
}()
```

## Success Criteria

- [ ] `./bin/ailang run --caps IO --entry main examples/runnable/guards_basic.ail` outputs expected values
- [ ] Guards with int/float/string comparisons work
- [ ] Guards accessing pattern bindings work
- [ ] Nested patterns with guards work: `(x, y) if x > y => ...`
- [ ] ADT patterns with guards work: `Some(x) if x > 0 => ...`
- [ ] All existing tests pass (regression check)
- [ ] No performance regression on non-guarded matches

## Testing Strategy

**Unit tests (`codegen_match_test.go`):**
- Guard with var pattern: `x if x > 0 => ...`
- Guard with tuple pattern: `(a, b) if a > b => ...`
- Guard with ADT pattern: `Some(x) if x != 0 => ...`
- Multiple guards falling through
- Guard as last arm (should work as-is)

**Integration tests:**
- Full pipeline: parse -> elaborate -> typecheck -> codegen -> go build -> run
- `examples/runnable/guards_basic.ail` produces correct output

**Regression tests:**
- All existing match tests still pass
- No change to non-guarded match behavior

## Non-Goals

**Not in this feature:**
- Effectful guards (already rejected by type checker)
- Guard optimization (constant folding, short-circuit) - future enhancement
- Exhaustiveness analysis with guards (already handled conservatively)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fallthrough semantics differ between paths | Medium | Test all three codegen paths independently |
| Binding order matters for guard access | Low | Ensure bindings generated before guard |
| Type assertions in guards | Low | Guards are typed; use TypeMapper for types |

## Related Documents

**Implemented (informs this fix):**
- `design_docs/implemented/v0_6_2/m-record-patterns.md` - Recent pattern work
- `internal/eval/eval_patterns.go` - Working guard implementation (evaluator)

**Tests to reference:**
- `internal/eval/guards_simple_test.go` - Evaluator guard tests (6 tests, all passing)

## References

- [Design Axioms](/docs/references/axioms)
- Evaluator guard implementation: `internal/eval/eval_patterns.go:56-76`
- Codegen match: `internal/gen/golang/codegen_match.go` (no guard handling)
- Example file: `examples/runnable/guards_basic.ail`

## Investigation Summary

**Root Cause Analysis:**

The evaluator (`internal/eval/`) correctly handles `arm.Guard`:
```go
// internal/eval/eval_patterns.go:56-76
if arm.Guard != nil {
    guardVal, err := e.evalCore(arm.Guard)
    if !guardVal.(*BoolValue).Value {
        continue // Skip to next arm
    }
}
```

The codegen (`internal/gen/golang/codegen_match.go`) ignores guards entirely:
```go
// Lines 129-135, 147-169, etc. - arm.Guard never accessed
for _, arm := range match.Arms {
    // Uses arm.Pattern and arm.Body
    // Never checks arm.Guard
}
```

**Why it went unnoticed:**
1. Unit tests use evaluator directly (guards work)
2. File execution uses compiled Go (guards ignored)
3. No integration test comparing evaluator vs compiled output

---

## Implementation Notes (2025-12-29)

**Actual root cause was different from expected:**

The design doc assumed the problem was in codegen ignoring guards. The actual root cause was that **guards were being stripped during pipeline transformation passes**:

1. **`internal/elaborate/dictionaries.go`** - `DictElaborator.transformExpr()` for Match case created new `MatchArm` structs without copying the `Guard` field
2. **`internal/linked/linked.go`** - `linkExpr()` for Match case created new `MatchArm` structs without copying the `Guard` field

**The fix was minimal (~15 LOC total):**

1. `dictionaries.go:237` - Add guard transformation:
   ```go
   var guard core.CoreExpr
   if arm.Guard != nil {
       guard = de.transformExpr(arm.Guard)
   }
   newArms = append(newArms, core.MatchArm{
       Pattern: arm.Pattern,
       Guard:   guard,  // <-- Was missing
       Body:    de.transformExpr(arm.Body),
   })
   ```

2. `linked.go:115` - Add guard linking:
   ```go
   var guard core.CoreExpr
   if arm.Guard != nil {
       guard = linkExpr(arm.Guard, dictReg)
   }
   arms = append(arms, core.MatchArm{
       Pattern: arm.Pattern,
       Guard:   guard,  // <-- Was missing
       Body:    linkExpr(arm.Body, dictReg),
   })
   ```

3. `codegen_match.go:331` - Add bool type assertion for guard expressions:
   ```go
   g.writef("if ")
   g.generateExpr(arm.Guard)
   g.writef(".(bool) {\n")  // <-- Guard returns interface{}, need bool cast
   ```

**Lessons learned:**
- When AST transformation passes are added, they need to copy ALL fields, not just the common ones
- A pattern-based search for `MatchArm{` revealed all locations creating new MatchArms
- The codegen already had guard-aware code paths (`generateMatchIfElseWithGuards`) but they were never reached because guards were nil

---

**Document created**: 2024-01-15 (original)
**Last updated**: 2025-12-29 (implemented)

# Sprint Plan: M-BUG-FIXES (TVar2 List Pattern + ADT Test Harness)

## Summary

Fix two critical type system bugs reported by stapledons_voyage that block match expressions over record lists and TDD with ADTs.

**Duration:** 1-2 days
**Dependencies:** None (builds on existing type system)
**Risk Level:** Medium (type system changes require careful testing)

## Current Status Analysis

### Completed Recently (v0.4.9)
- Fixed inline nested record literals: ~95 LOC
- Fixed module-level let binding scope: ~80 LOC
- Fixed record update ANF verification: ~25 LOC
- Split large files (server/messaging): ~300 LOC reorganization

### Velocity
- Recent average: 150-300 LOC/day
- Estimated capacity: ~350 LOC for this sprint
- Completion rate: 3-4 bug fixes per day possible

### Remaining Work
- M-BUG-TVAR2-LIST-PATTERN: ~150 LOC estimated
- M-BUG-ADT-TEST-HARNESS-SCOPE: ~100 LOC estimated
- Total: ~250 LOC + ~100 LOC tests = ~350 LOC

## Proposed Milestones

### Milestone 1: M1-TVAR2-LIST-PATTERN
**Goal:** Fix TVar2 unification error when accessing record fields through list pattern bindings
**Estimated:** 120 LOC implementation + 50 LOC tests = 170 LOC
**Duration:** 0.5-1 day

**Key Files:**
- `internal/types/typechecker_patterns.go` - ConsPattern/ListPattern handling
- `internal/types/unification.go` - TVar2 unification with open records
- `internal/types/row_unification.go` - Row polymorphism

**Tasks:**
1. **Diagnosis Phase** (~30 min)
   - Add debug logging to `inferConsPattern` in typechecker_patterns.go
   - Trace type of bound variable (`pos`) after pattern match
   - Identify where TVar2 is introduced vs where record type expected

2. **Fix Implementation** (~2-3 hours)
   - Likely fix: Ensure element type is properly instantiated in cons pattern
   - May need to apply substitution to bound variable's type before body inference
   - Or: Add special handling for open record types in list element position

3. **Testing** (~1 hour)
   - Create `examples/list_pattern_records.ail` with test cases
   - Add unit tests for cons pattern with record types
   - Verify nested records also work

**Acceptance Criteria:**
- [ ] `match positions { pos :: rest => pos.x, [] => 0 }` compiles and runs
- [ ] Nested record access works: `npc :: rest => npc.pos.x`
- [ ] Record updates in patterns work: `p :: rest => { p | x: p.x + 1 }`
- [ ] All existing tests pass
- [ ] Example file added and verified

**Risks:**
- Type system changes can have unexpected side effects - Mitigation: Run full test suite after each change
- TVar2 is used in multiple places - Mitigation: Add regression tests

### Milestone 2: M2-ADT-TEST-HARNESS-SCOPE
**Goal:** Fix ADT constructor resolution in test harness evaluation scope
**Estimated:** 80 LOC implementation + 30 LOC tests = 110 LOC
**Duration:** 0.5 day

**Key Files:**
- `internal/testing/harness.go` - Test execution context
- `internal/testing/executor.go` - Test case evaluation
- `internal/elaborate/file.go` - ADT constructor scope (reference)

**Tasks:**
1. **Diagnosis Phase** (~20 min)
   - Trace how test expressions are evaluated in harness.go
   - Compare scope available in function body vs test expression
   - Identify where ADT constructors are missing

2. **Fix Implementation** (~1-2 hours)
   - Inject module's ADT constructors into test evaluation scope
   - Likely: Extend evaluator environment with constructor bindings
   - May need to access module's type environment

3. **Testing** (~30 min)
   - Create `examples/adt_test_harness.ail` with test cases
   - Verify simple ADT, ADT with data, nested constructors all work

**Acceptance Criteria:**
- [ ] `tests [(North, 0), (South, 0)]` compiles with `type Direction = North | ...`
- [ ] ADT with data works: `tests [(Just(5), 5), (Nothing, 0)]`
- [ ] All existing tests pass
- [ ] Example file added and verified

**Risks:**
- Test harness architecture may be complex - Mitigation: Start with minimal scope injection
- May need to understand full elaboration pipeline - Mitigation: Reference M-BUG-MODULE-LET-SCOPE fix pattern

## Day-by-Day Plan

### Day 1 (Today)
**Morning:**
- [ ] M1: Diagnosis - add debug logging, reproduce issue with tracing
- [ ] M1: Identify root cause in typechecker_patterns.go

**Afternoon:**
- [ ] M1: Implement fix in cons pattern handling
- [ ] M1: Write tests and example file
- [ ] M1: Run full test suite, fix any regressions

**Evening (if needed):**
- [ ] M2: Diagnosis - trace test harness scope construction
- [ ] M2: Implement constructor injection fix

### Day 2 (If Needed)
**Morning:**
- [ ] M2: Complete implementation and testing
- [ ] Create example files for both fixes
- [ ] Run `make verify-examples`

**Afternoon:**
- [ ] Final testing and cleanup
- [ ] Update CHANGELOG.md
- [ ] Respond to stapledons_voyage with fix confirmation

## Success Metrics
- Test coverage maintained or improved
- Examples passing: 2 new example files
- Documentation: Update CHANGELOG.md, respond to bug reporter
- All tests passing
- All linting clean
- `make verify-examples` passes

## Test Cases

### M1: List Pattern Records
```ailang
-- examples/list_pattern_records.ail
module examples/list_pattern_records

type Point = { x: int, y: int }
type Entity = { pos: Point, name: string }

-- Simple record field access in list pattern
pure func sumX(points: [Point]) -> int {
    match points {
        p :: rest => p.x + sumX(rest),
        [] => 0
    }
}

-- Nested record access
pure func sumEntityX(entities: [Entity]) -> int {
    match entities {
        e :: rest => e.pos.x + sumEntityX(rest),
        [] => 0
    }
}

-- Record update in list pattern
pure func moveAllX(points: [Point], dx: int) -> [Point] {
    match points {
        p :: rest => { p | x: p.x + dx } :: moveAllX(rest, dx),
        [] => []
    }
}

-- Test expressions
sumX([{x: 1, y: 0}, {x: 2, y: 0}, {x: 3, y: 0}])  -- Expected: 6
```

### M2: ADT Test Harness
```ailang
-- examples/adt_test_harness.ail
module examples/adt_test_harness

type Direction = North | South | East | West

pure func directionDx(dir: Direction) -> int
tests [(North, 0), (South, 0), (East, 1), (West, -1)]
{
    match dir {
        North => 0,
        South => 0,
        East => 1,
        West => -1
    }
}

type Maybe[a] = Just(a) | Nothing

pure func fromMaybe(def: int, m: Maybe[int]) -> int
tests [(0, Nothing, 0), (0, Just(42), 42)]
{
    match m {
        Just(x) => x,
        Nothing => def
    }
}

-- Run direction test
directionDx(East)  -- Expected: 1
```

## Dependencies
- None (both bugs are isolated to specific subsystems)

## Open Questions
- None - both issues are well-defined with clear reproduction cases

## Notes
- Both bugs were reported by stapledons_voyage working on game engine
- TVar2 bug is higher priority (no workaround exists)
- ADT harness bug has workaround (separate test files)
- Pattern follows recent M-BUG-MODULE-LET-SCOPE fix approach

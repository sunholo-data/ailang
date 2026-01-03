# Sprint Plan: M-PATTERN-GUARDS

## Summary
Fix pattern guard evaluation in Go code generation. Guards work in the evaluator but are completely ignored by codegen, causing silent incorrect behavior in compiled programs.

**Duration:** 1 day (~4-5 hours)
**Dependencies:** None
**Risk Level:** Low (focused fix, evaluator already implements correct behavior)

## Current Status Analysis

### Completed Recently
- ✅ M-RECORD-PATTERNS: ~375 LOC in 1 day
- ✅ M-DX19 (auto-derive Eq): ~200 LOC in 1 day
- ✅ M-CODEGEN-DICTIONARIES: ~300 LOC in 1 day
- ✅ M-CODEGEN-BOOL-ASSERTIONS: ~150 LOC in 1 day

### Velocity
- Recent average: 200-300 LOC/day for codegen work
- Estimated capacity: 200 LOC for this sprint

### Remaining from Design Doc
- ⏳ Phase 1-4: Guard codegen (~150 LOC)
- ⏳ Phase 5: Tests & examples (~50 LOC)

## Proposed Milestones

### Milestone 1: Guard Codegen Implementation
**Goal:** Emit guard checks in all three codegen paths (if-else, value switch, ADT switch)
**Estimated:** 120 LOC implementation + 50 LOC tests = 170 LOC
**Duration:** 4-5 hours

**Tasks:**

**Hour 1: Analyze and Add Guard Helper**
- [ ] Read `codegen_match.go` thoroughly (~660 lines)
- [ ] Add `generateGuardExpr(guard core.CoreExpr) error` helper
- [ ] Handle nil guard (no-op case)

**Hour 2: If-Else Chain Guards (generateMatchIfElse)**
- [ ] Modify if-else generation to include guard in condition
- [ ] Ensure bindings are generated BEFORE guard evaluation
- [ ] Pattern: `if <pattern> { if <guard> { return body } }`

**Hour 3: Value Switch Guards (generateMatchArmValueSwitch)**
- [ ] Add guard check after case label
- [ ] Pattern: `case X: if !<guard> { fallthrough }; return body`
- [ ] Handle fallthrough semantics correctly

**Hour 4: ADT Switch Guards (generateMatchArmADT)**
- [ ] Add guard check after binding extraction
- [ ] Pattern: `case Kind_X: <bindings>; if !<guard> { fallthrough }; return body`
- [ ] Ensure bindings visible to guard expression

**Hour 5: Testing & Verification**
- [ ] Run `make test` - verify no regressions
- [ ] Run `./bin/ailang run --caps IO --entry main examples/runnable/guards_basic.ail`
- [ ] Verify output is "positive" (not "big")
- [ ] Add codegen unit tests in `codegen_match_test.go`
- [ ] Run `make lint` - ensure clean

**Acceptance Criteria:**
- [ ] `examples/runnable/guards_basic.ail` outputs correct values
- [ ] Guards with int comparisons work: `x if x > 0 => ...`
- [ ] Guards with tuple bindings work: `(a, b) if a > b => ...`
- [ ] All existing tests pass (`make test`)
- [ ] Linting clean (`make lint`)

**Risks:**
- Fallthrough semantics in Go switch - Mitigation: Test each case type separately
- Binding visibility for guards - Mitigation: Generate bindings before guard check

## Success Metrics
- Test coverage: No regression (existing match tests still pass)
- Examples passing: `guards_basic.ail` produces correct output
- Documentation: Update design doc status to Implemented
- All tests passing: ✅
- All linting passing: ✅

## Dependencies
- None (evaluator implementation already correct, provides reference)

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/gen/golang/codegen_match.go` | Add guard evaluation | ~100 |
| `internal/gen/golang/codegen_match_test.go` | Add guard tests | ~50 |
| `examples/runnable/guards_basic.ail` | Update expected output comments | ~5 |

## Test Plan

**Unit Tests (new):**
```go
func TestGenerateMatch_WithGuard(t *testing.T) { ... }
func TestGenerateMatch_MultipleGuards(t *testing.T) { ... }
func TestGenerateMatch_GuardWithTuple(t *testing.T) { ... }
func TestGenerateMatch_GuardWithADT(t *testing.T) { ... }
```

**Integration Test:**
```bash
./bin/ailang run --caps IO --entry main examples/runnable/guards_basic.ail
# Expected outputs:
# positive (not "big")
# the answer
# negative
# y bigger
```

## Notes
- The evaluator implementation at `internal/eval/eval_patterns.go:56-76` is the reference
- All 6 evaluator guard tests pass - codegen should match this behavior
- Key insight: arm.Guard is already in Core AST, just never accessed in codegen

---

**Created:** 2025-12-29

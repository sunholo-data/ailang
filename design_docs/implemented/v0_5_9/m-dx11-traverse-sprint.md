# Sprint Plan: M-DX11-TRAVERSE - Safe Type Traversal Library

## Summary
Implement a reusable type traversal library with automatic cycle detection to prevent future cyclic type hangs and reduce boilerplate across 4+ type traversal functions.

**Duration:** 1 day (4 hours)
**Dependencies:** M-PERF2 (completed), M-DX11 Phase 1 (completed)
**Risk Level:** Low
**Parent Design Doc:** [m-dx11-traverse.md](m-dx11-traverse.md)

## Current Status Analysis

### Completed Recently
- ✅ M-PERF2: SafeEquals cycle protection fix (~50 LOC)
- ✅ M-DX11 Phase 1: Cyclic type diagnostics
- ✅ v0.5.8 release: Go codegen type safety

### Velocity
- Recent average: ~150-200 LOC/day based on changelog entries
- This sprint: 450 LOC estimated, fits comfortably in 1 day

### Existing Cycle-Protected Traversals
Files with manual `visited[t] = true` pattern (migration candidates):
1. `internal/types/safe_string.go` - SafeTypeString (reference pattern, 170 LOC)
2. `internal/types/unification_occurs.go` - occurs check (100 LOC)
3. `internal/types/unification_equality.go` - SafeEquals
4. `internal/types/typechecker_defaulting.go` - collectFreeVars

### Remaining Work
- ⏳ Phase 1: Core library (~150 LOC)
- ⏳ Phase 2: Safe wrappers (~100 LOC)
- ⏳ Phase 3: Migration + tests (~200 LOC)

## Proposed Milestones

### M1: Core TypeVisitor Library
**Goal:** Create the foundational visitor pattern with automatic cycle detection
**Estimated:** 150 LOC implementation + 100 LOC tests = 250 LOC
**Duration:** 2 hours

**Tasks:**
1. Create `internal/types/traverse/traverse.go`:
   - TypeVisitor struct with visited map, depth, maxDepth, OnCycle callback
   - NewVisitor() constructor with sensible defaults
   - Visit(t Type, fn func(Type)) method with cycle/depth protection
   - children(t Type) []Type helper for all 15+ type variants

2. Create `internal/types/traverse/traverse_test.go`:
   - Test visitor visits all type variants (TCon, TVar, TFunc2, TList, TTuple, TRecord, TApp, Row, etc.)
   - Test cycle detection triggers OnCycle callback
   - Test depth limit triggers panic
   - Test nested cycles (e.g., recursive ADTs)

**Files to create:**
- `internal/types/traverse/traverse.go` (~150 LOC)
- `internal/types/traverse/traverse_test.go` (~100 LOC)

**Acceptance Criteria:**
- [ ] TypeVisitor.Visit handles all 15+ Type variants
- [ ] Cycle detection works on self-referential types
- [ ] Depth limit (1000) prevents pathological cases
- [ ] OnCycle callback fires with detected type
- [ ] All tests passing
- [ ] Lint clean

**Risks:**
- Type variant coverage incomplete - Mitigation: enumerate all variants from types.go

---

### M2: Safe Wrapper Functions
**Goal:** Provide convenient safe wrappers for common type operations
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 1 hour

**Tasks:**
1. Add to `internal/types/traverse/wrappers.go`:
   - `CollectFreeVars(t Type) map[string]bool` - gather type variables
   - `ContainsType(t, target Type) bool` - check for type presence
   - `FoldTypes[T](t Type, init T, fn func(T, Type) T) T` - reduce over types

2. Add wrapper tests to `traverse_test.go`:
   - CollectFreeVars finds all TVars in complex types
   - CollectFreeVars terminates on cyclic types
   - ContainsType finds nested types
   - FoldTypes accumulates correctly

**Files to create:**
- `internal/types/traverse/wrappers.go` (~100 LOC)

**Files to modify:**
- `internal/types/traverse/traverse_test.go` (+50 LOC)

**Acceptance Criteria:**
- [ ] CollectFreeVars works on cyclic types without hanging
- [ ] ContainsType correctly detects nested types
- [ ] FoldTypes enables custom aggregations
- [ ] API documented with examples
- [ ] All tests passing

**Risks:**
- Generic FoldTypes may require Go 1.18+ generics - Mitigation: use interface{} if needed

---

### M3: Migration & Documentation
**Goal:** Migrate existing traversal code to use the new library
**Estimated:** 50 LOC changes + documentation
**Duration:** 0.5 hours

**Tasks:**
1. Update `internal/types/typechecker_defaulting.go`:
   - Replace manual collectFreeVars with `traverse.CollectFreeVars`

2. Document API design rule in CLAUDE.md:
   - "Every function of shape `func(Type) T` MUST document cycle-safety"

3. Add migration guide comments in traverse package

**Files to modify:**
- `internal/types/typechecker_defaulting.go` (~10 LOC change)
- `CLAUDE.md` - Add API design rule

**Acceptance Criteria:**
- [ ] typechecker_defaulting uses traverse.CollectFreeVars
- [ ] API design rule documented
- [ ] All existing tests still pass
- [ ] No regression in type checking behavior

**Risks:**
- Migration breaks edge cases - Mitigation: run full test suite, compare behavior

---

### M4: Final Validation & Cleanup
**Goal:** Ensure quality and document completion
**Estimated:** 0.5 hours

**Tasks:**
1. Run full test suite: `make test`
2. Run linter: `make lint`
3. Verify with stapledons_voyage types (known cycles): `ailang check examples/stapledon/*.ail`
4. Update design doc status to "Implemented"
5. Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] All 8 success criteria from design doc met
- [ ] Test coverage >80% for traverse package
- [ ] No lint errors
- [ ] Design doc moved to implemented/v0_5_9/

---

## Success Metrics
- Test coverage: >80% for traverse package
- All existing tests passing: ✅
- All linting passing: ✅
- Documentation: CLAUDE.md updated with API rule
- Design doc: Moved to implemented/

## Dependencies
- Go 1.18+ for generics (optional, can use interface{} fallback)
- No external dependencies

## Task Summary (Day-by-Day)

| Time | Task | LOC | Deliverable |
|------|------|-----|-------------|
| 0-2h | M1: Core TypeVisitor | 250 | traverse.go + tests |
| 2-3h | M2: Safe Wrappers | 150 | wrappers.go + tests |
| 3-3.5h | M3: Migration | 50 | Update typechecker |
| 3.5-4h | M4: Validation | - | Clean release |

**Total:** ~450 LOC in 4 hours

## Open Questions
None - design doc is comprehensive

## Notes
- Reference implementation: `internal/types/safe_string.go` has the pattern we're generalizing
- The design doc's API design rule from M-PERF2 post-mortem should be enforced
- Consider using pool for visited maps in future performance optimization (non-goal for this sprint)

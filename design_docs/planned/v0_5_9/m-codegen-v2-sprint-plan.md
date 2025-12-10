# Sprint Plan: M-CODEGEN-V2 - Flat Go Code Generation

## Summary
Eliminate nested IIFEs in Go codegen by introducing a Block IR layer between Core AST and Go emission, enabling stapledons_voyage to compile without OOM.

**Duration:** 3 days (core), +1 day (stretch goals)
**Dependencies:** None (builds on existing codegen)
**Risk Level:** Medium (architectural change to codegen pipeline)

## Current Status Analysis

### Completed Recently
- ✅ M-CODEGEN-FLAT-IF-ELSE: Flatten if-else chains (~200 LOC) in 1 day
- ✅ M-FIX-IF-ELSE-LET: Better error for let in if-else branches (~50 LOC)
- ✅ Codegen array fixes (~120 LOC)
- ✅ M-DX11: Cyclic type diagnostics (~300 LOC)

### Velocity
- Recent average: ~150-200 LOC/day (based on last week's codegen work)
- Estimated capacity: ~600-800 LOC for 4-day sprint
- Risk buffer: 20% (architectural changes may surface edge cases)

### Problem Statement
- stapledons_voyage generates 28-level nested IIFEs
- Go compiler OOMs at 2GB+ RAM
- 437 IIFEs across 6,184 LOC of generated code
- Game unplayable due to 26K allocs/sec GC pressure

## Proposed Milestones

### Milestone 1: Block IR Package
**Goal:** Create language-neutral Block IR and Core→Block lowering pass
**Estimated:** 100 LOC implementation + 80 LOC tests = 180 LOC
**Duration:** Day 1 (4 hours)

**Tasks:**
- Create `internal/gen/block/block.go` - Block and Stmt types
- Create `internal/gen/block/lower.go` - Lower() function
- Create `internal/gen/block/lower_test.go` - Unit tests
- Test with simple let chains (1, 5, 10 bindings)

**Files to Create:**
```
internal/gen/block/
├── block.go      (~40 LOC) - Block, Stmt types
├── lower.go      (~60 LOC) - Lower() function
└── lower_test.go (~80 LOC) - Unit tests
```

**Acceptance Criteria:**
- [ ] `Lower()` extracts all top-level Let bindings
- [ ] Evaluation order preserved (bindings in Core order)
- [ ] Empty Block returned for non-Let expressions
- [ ] Nested Let in value expression NOT flattened (correct behavior)
- [ ] 100% test coverage on block package
- [ ] `go test ./internal/gen/block/...` passes

**Risks:**
- None significant - this is a simple data transformation

---

### Milestone 2: Flat Function Body Generation
**Goal:** Function bodies use Block IR, generate flat statements instead of nested IIFEs
**Estimated:** 150 LOC implementation + 100 LOC tests = 250 LOC
**Duration:** Day 2 (6 hours)

**Tasks:**
- Add `generateBlock()` method to Generator
- Modify `codegen_decl.go` to use Block for function bodies
- Update `generateLet()` to use Block IR (single IIFE, not nested)
- Add tests for function body generation

**Files to Modify:**
```
internal/gen/golang/
├── codegen.go           (+20 LOC) - Add generateBlock()
├── codegen_decl.go      (+30 LOC) - Use Block for function bodies
├── codegen_expr_let.go  (~50 LOC) - Rewrite generateLet with Block
└── codegen_flat_test.go (new, ~100 LOC) - Integration tests
```

**Acceptance Criteria:**
- [ ] Function with 5 lets: ~10 lines (not 50+)
- [ ] No nested `return func() interface{} { ... }()` in function bodies
- [ ] Let in expression position: single flat IIFE (not nested)
- [ ] All existing codegen tests pass
- [ ] `go test ./internal/gen/golang/...` passes

**Risks:**
- May surface edge cases with existing if-else flattening
- Mitigation: Keep old code behind `--legacy-codegen` flag

---

### Milestone 3: Validation with stapledons_voyage
**Goal:** Verify fix works in production codebase
**Estimated:** 50 LOC (test harness) + validation time = 100 LOC
**Duration:** Day 3 (4 hours)

**Tasks:**
- Build ailang with new codegen
- Regenerate stapledons_voyage sim_gen/
- Verify Go compiler doesn't OOM
- Run stapledons_voyage, verify game works
- Measure metrics (nesting depth, IIFE count, compiler RAM)

**Files to Create:**
```
internal/gen/golang/
└── codegen_invariant_test.go (~50 LOC) - CI check for nesting depth
```

**Example File:**
```
examples/codegen_stress_test.ail (~30 LOC) - 20-let chain test
```

**Acceptance Criteria:**
- [ ] stapledons_voyage compiles without OOM
- [ ] Go compiler uses <500MB RAM
- [ ] Max IIFE nesting depth ≤ 1
- [ ] Game runs without GC freezes
- [ ] CI invariant test prevents regressions

**Risks:**
- Unknown edge cases in stapledons code
- Mitigation: Have rollback path via --legacy-codegen

---

### Milestone 4 (Stretch): Quick Wins Cleanup
**Goal:** Remove code bloat (redundant conversions, suppress unused)
**Estimated:** 80 LOC implementation + 40 LOC tests = 120 LOC
**Duration:** Day 4 (3 hours) - IF time permits

**Tasks:**
- Remove `int64(int64(x))` redundant conversions
- Remove `_ = x // suppress unused` when variable is used
- Update tests

**Files to Modify:**
```
internal/gen/golang/
├── codegen_expr_simple.go (+20 LOC) - Check before adding conversion
└── codegen_expr_let.go    (+20 LOC) - Track variable usage
```

**Acceptance Criteria:**
- [ ] No `int64(int64(...))` in generated code
- [ ] Suppress unused only on actually unused vars
- [ ] All tests pass

**Risks:**
- Lower priority - skip if behind schedule

---

## Day-by-Day Plan

### Day 1: Block IR Foundation
| Time | Task | Deliverable |
|------|------|-------------|
| 0-2h | Create Block IR types | `internal/gen/block/block.go` |
| 2-4h | Implement Lower() + tests | `lower.go`, `lower_test.go` |
| 4h | Checkpoint | Block package complete, all tests pass |

### Day 2: Flat Generation
| Time | Task | Deliverable |
|------|------|-------------|
| 0-2h | Add generateBlock() | `codegen.go` modified |
| 2-4h | Integrate with function bodies | `codegen_decl.go` modified |
| 4-6h | Rewrite generateLet() | `codegen_expr_let.go` rewritten |
| 6h | Checkpoint | All codegen tests pass |

### Day 3: Validation
| Time | Task | Deliverable |
|------|------|-------------|
| 0-2h | Build and regenerate stapledons | New sim_gen/ files |
| 2-3h | Test compilation and runtime | Verified working |
| 3-4h | Add CI invariant test | `codegen_invariant_test.go` |
| 4h | Checkpoint | Sprint P0 complete |

### Day 4 (Stretch): Cleanup
| Time | Task | Deliverable |
|------|------|-------------|
| 0-2h | Remove redundant conversions | Cleaner generated code |
| 2-3h | Remove unnecessary suppress unused | Even cleaner code |
| 3h | Final validation | Sprint complete |

## Success Metrics

| Metric | Before | Target | Verification |
|--------|--------|--------|--------------|
| Max IIFE nesting | 28 | **1** | `grep -c "return func()"` |
| IIFEs in bridge.go | 255 | <50 | Count in generated code |
| LOC in bridge.go | 1616 | <1000 | `wc -l` |
| Go compiler RAM | OOM | <500MB | `time go build` |
| stapledons builds | No | **Yes** | CI green |
| All tests pass | - | **Yes** | `make test` |
| All lint pass | - | **Yes** | `make lint` |

## Dependencies
- None - this builds on existing codegen infrastructure
- M-CODEGEN-FLAT-IF-ELSE already implemented (can integrate)

## Rollback Plan
If critical issues arise:
1. Add `--legacy-codegen` flag to use old code path
2. Flag controlled at compile command level
3. Keep old generateLet code commented but intact

## Open Questions
**Resolved:**
- ✅ Block IR vs in-codegen flattening? → Block IR (cleaner separation)
- ✅ Typed codegen in scope? → No, defer to M-DX24

**Remaining:**
- Match branch bodies: use Block IR? → Yes, but defer to Phase 4 (post-sprint)

## Notes
- This sprint focuses on **structural flattening only**
- Typed codegen (native operators, direct field access) is M-DX24 scope
- stapledons_voyage is the primary validation target
- Conservative estimates account for edge cases

# Sprint Plan: M-DX11-TYPE-REPORT (Type Inference Debug Primitive)

## Summary

Implement the `TypeReport` function - the canonical primitive for type debugging that consolidates information from 4 fragmented data structures into a single queryable API.

**Duration:** 1 day (~6 hours)
**Dependencies:** M-DX11 Phase 1 (substitution chain tests) - ✅ Complete
**Risk Level:** Low (well-defined scope, foundation for future work)
**Design Doc:** [m-dx11-type-inference-debugging.md](m-dx11-type-inference-debugging.md)

## Current Status Analysis

### What Already Exists
- ✅ `internal/types/substitution_chain_test.go` - 5 tests for chain resolution (~200 LOC)
- ✅ `internal/types/unification_core.go` - ApplySubstitution with chain following
- ✅ `internal/types/typechecker_core.go` - CoreTypeChecker with CoreTI
- ✅ `internal/types/inference_context.go` - InferenceContext with constraints

### Current Gaps
- ❌ No unified view of type info for a node (scattered across 4 structures)
- ❌ No way to see raw vs resolved type for debugging
- ❌ No programmatic API for type debugging (have to add printf statements)

### Velocity
- Recent average: ~200-400 LOC/day (from v0.5.9 work)
- Estimated capacity: ~300 LOC for this sprint

## Proposed Milestones

### Milestone 1: TypeReport Types and Structure
**Goal:** Define the TypeReport, ConstraintRef, and related types
**Estimated:** 80 LOC
**Duration:** 1 hour

**Tasks:**
1. Create `internal/types/type_report.go`:
   - Define `TypeReport` struct (NodeID, Raw, Resolved, Constraints)
   - Define `ConstraintRef` struct (Constraint, SourceSpan)
   - Define `TypeOrigin` and `OriginKind` for future provenance

**Acceptance Criteria:**
- [ ] Types compile and are documented
- [ ] Types match design doc specification

### Milestone 2: ApplyClosure for Full Chain Resolution
**Goal:** Implement full substitution closure (follow all chains to concrete types)
**Estimated:** 50 LOC + 100 LOC tests
**Duration:** 2 hours

**Tasks:**
1. Add `ApplyClosure(sub Substitution, t Type) Type` to unification_core.go:
   - Recursively apply substitution until no more changes
   - Handle cycles (return error or original type)
   - Different from ApplySubstitution which does single pass

2. Add tests in `internal/types/apply_closure_test.go`:
   - Test single-step resolution
   - Test multi-step chains (α → β → γ → float)
   - Test cycle detection
   - Test nested types (functions, records, ADTs)

**Acceptance Criteria:**
- [ ] `ApplyClosure` resolves all chains to concrete types
- [ ] Handles cycles without infinite loop
- [ ] All existing tests still pass

### Milestone 3: TypeReport Function Implementation
**Goal:** Implement the canonical typeReport(nodeID) function
**Estimated:** 100 LOC + 80 LOC tests
**Duration:** 2.5 hours

**Tasks:**
1. Add `TypeReport(nodeID uint64) TypeReport` method to CoreTypeChecker:
   - Pull raw type from CoreTI
   - Apply full substitution closure for Resolved
   - Find constraints mentioning this type's variables
   - Return consolidated TypeReport

2. Add tests in `internal/types/type_report_test.go`:
   - Test basic report generation
   - Test with unresolved type variables
   - Test with resolved concrete types
   - Test constraint collection

**Acceptance Criteria:**
- [ ] `tc.TypeReport(nodeID)` returns consolidated info
- [ ] Raw shows what's in CoreTI (may have TVars)
- [ ] Resolved shows fully substituted type
- [ ] Constraints list includes relevant constraints

### Milestone 4: Integration and Documentation
**Goal:** Integrate TypeReport with existing debug infrastructure
**Estimated:** 30 LOC
**Duration:** 0.5 hours

**Tasks:**
1. Update `cmd/ailang/debug.go`:
   - Add `debug types <file>` subcommand stub (prints "TypeReport available")
   - Document programmatic API in code comments

2. Update CHANGELOG.md and design doc

**Acceptance Criteria:**
- [ ] `ailang debug types` command exists (stub for Phase 3)
- [ ] CHANGELOG updated
- [ ] Design doc updated with Phase 2 complete

## Success Metrics
- Test coverage: >80% for `internal/types/type_report.go`
- All existing tests passing: ✅
- TypeReport function usable from Go code
- Foundation ready for Phase 3 (--debug-types CLI)

## Implementation Approach

The key insight is that TypeReport is a **thin façade** over existing structures:

```go
func (tc *CoreTypeChecker) TypeReport(nodeID uint64) TypeReport {
    // 1. Get raw type from CoreTI
    raw, ok := tc.CoreTI.Get(nodeID)
    if !ok {
        return TypeReport{NodeID: nodeID} // empty report
    }

    // 2. Apply full substitution closure
    resolved := ApplyClosure(tc.substitution, raw)

    // 3. Find constraints mentioning this node's type vars
    constraints := tc.findConstraintsFor(raw)

    return TypeReport{
        NodeID:      nodeID,
        Raw:         raw,
        Resolved:    resolved,
        Constraints: constraints,
    }
}
```

## Files to Create/Modify

**New files:**
- `internal/types/type_report.go` (~120 LOC) - Types and TypeReport function
- `internal/types/type_report_test.go` (~100 LOC) - Unit tests
- `internal/types/apply_closure_test.go` (~80 LOC) - Closure tests

**Modified files:**
- `internal/types/unification_core.go` (+50 LOC) - ApplyClosure function
- `cmd/ailang/debug.go` (+30 LOC) - Stub for debug types command

**Total:** ~380 LOC

## Open Questions

None - design is clear from parent M-DX11 design doc.

## Notes

- TypeReport is the foundation for Phase 3 (--debug-types CLI)
- Keep it simple - no provenance tracking yet (that's Phase 4)
- Focus on programmatic API first, CLI formatting later


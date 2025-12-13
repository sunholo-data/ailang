# Sprint Plan: M-DX11-TYPE-PROVENANCE

## Summary
Track where types originate from (annotations, literals, inference, defaulting) to answer "why does this have type X?" questions.

**Duration:** 1 day (8 hours)
**Dependencies:** M-DX11-DEBUG-TYPES-CLI (v0.5.11) - COMPLETE
**Risk Level:** Low
**Priority:** P3 - Low (Optional Enhancement)

## Current Status Analysis

### Completed Recently (Dependencies)
- M-DX11-DEBUG-TYPES-CLI: CLI integration with `--debug-types` flag
- M-DX11-TYPE-REPORT: TypeReport API (~180 LOC)
- TypeDebugSink interface with VerboseDebugSink (~191 LOC)
- OriginKind enum already defined in type_report.go

### Velocity
- Recent average: ~400-800 LOC/day for types-related work
- Estimated capacity: ~170 LOC for this sprint (lightweight feature)

### Remaining from Design Doc
- Provenance map in VerboseDebugSink: ~100 LOC
- Hook provenance events: ~40 LOC
- Integrate into TypeReport: ~30 LOC
- **Total: ~170 LOC**

## What's Already Done (Foundation)

The infrastructure is substantially in place:

1. **OriginKind enum** exists in `type_report.go:42-59`:
   - OriginUnknown, OriginAnnotation, OriginLiteral, OriginInferred
   - OriginDefaulted, OriginFromUse, OriginFromPattern

2. **TypeOrigin struct** exists in `type_report.go:35-39` (missing Span field)

3. **VerboseDebugSink.OnFreshTypeVar** already accepts OriginKind parameter

4. **DebugEvent** already stores Origin field

**What's Missing:**
- Provenance map (`map[TypeVarID][]TypeOrigin`) in VerboseDebugSink
- `Span` field in TypeOrigin struct
- Wiring from inference sites to provenance tracking
- TypeReport integration with origins
- Zero-overhead benchmark verification

## Proposed Milestones

### Milestone 1: Provenance Infrastructure
**Goal:** Add provenance map to VerboseDebugSink and extend TypeOrigin struct
**Estimated:** 60 LOC implementation + 40 LOC tests = 100 LOC
**Duration:** 3 hours

**Tasks:**
1. Add `Span` field to TypeOrigin struct in `type_report.go`
2. Add `provenance map[string][]TypeOrigin` to VerboseDebugSink
3. Add helper methods:
   - `RecordProvenance(typeVarName string, origin TypeOrigin)`
   - `GetProvenance(typeVarName string) []TypeOrigin`
4. Write unit tests for provenance storage

**Files to Modify:**
- `internal/types/type_report.go` (+10 LOC) - Add Span field
- `internal/types/debug_sink.go` (+50 LOC) - Provenance map and methods
- `internal/types/debug_sink_test.go` (+40 LOC) - Unit tests

**Acceptance Criteria:**
- [ ] TypeOrigin has Span field
- [ ] VerboseDebugSink can store and retrieve provenance
- [ ] Multiple origins per type variable supported
- [ ] All tests passing
- [ ] Linting clean

### Milestone 2: Wire Provenance Events
**Goal:** Hook provenance tracking at key inference points
**Estimated:** 40 LOC implementation + 20 LOC tests = 60 LOC
**Duration:** 3 hours

**Tasks:**
1. Wire `freshTypeVar()` calls in InferenceContext to emit provenance
2. Wire annotation handling in lambda/function inference
3. Wire defaulting pass in `internal/types/defaulting.go` (or equivalent)
4. Update OnFreshTypeVar to record provenance (not just events)

**Key Hook Points (from grep analysis):**
- `internal/types/inference.go:541` - freshTypeVar definition
- `internal/types/typechecker_functions.go:47,221,400` - parameter inference
- `internal/types/typechecker_patterns.go:158,186,224,278` - pattern types
- `internal/types/typechecker_data.go:68,112,167,224` - data types

**Files to Modify:**
- `internal/types/inference.go` (+15 LOC) - Emit provenance from freshTypeVar
- `internal/types/debug_sink.go` (+25 LOC) - RecordProvenance in OnFreshTypeVar

**Acceptance Criteria:**
- [ ] Provenance recorded for fresh type variables
- [ ] Provenance recorded for annotations
- [ ] Provenance recorded for defaulting
- [ ] Events include source spans
- [ ] All tests passing

### Milestone 3: TypeReport Integration
**Goal:** Include provenance in TypeReport output
**Estimated:** 30 LOC implementation + 20 LOC tests = 50 LOC
**Duration:** 2 hours

**Tasks:**
1. Add `Origins []TypeOrigin` field to TypeReport struct
2. Query provenance in TypeReport() method
3. Update TypeReport.String() to include origins
4. Add example output formatting matching design doc spec

**Example Output Format (from design doc):**
```
NodeID 42: float
  Raw: α7
  Resolved: float
  Origins:
    - Annotation: return type at examples/math_trig.ail:11
    - Defaulted: via Num → Fractional → float at defaulting pass
```

**Files to Modify:**
- `internal/types/type_report.go` (+30 LOC) - Origins field and formatting

**Acceptance Criteria:**
- [ ] TypeReport includes Origins field
- [ ] Origins show kind, location, and note
- [ ] String() output matches design doc format
- [ ] All tests passing

### Milestone 4: Zero-Overhead Verification
**Goal:** Verify zero overhead when debug flag not set
**Estimated:** 20 LOC benchmarks
**Duration:** 1 hour (last hour)

**Tasks:**
1. Add benchmark for NoOpDebugSink provenance path
2. Verify no allocations when debug disabled
3. Compare with existing BenchmarkNoOpDebugSink

**Files to Modify:**
- `internal/types/debug_sink_test.go` (+20 LOC) - Benchmark provenance

**Acceptance Criteria:**
- [ ] Zero allocations with NoOpDebugSink
- [ ] No performance regression vs current benchmarks
- [ ] Benchmark results documented

## Success Metrics
- Test coverage: Existing coverage maintained
- All tests passing
- All linting passing
- Zero allocations verified when debug disabled
- TypeReport shows origins when VerboseDebugSink active

## Dependencies
- M-DX11-DEBUG-TYPES-CLI (v0.5.11) - COMPLETE

## Open Questions
None - design is well-specified and foundation exists.

## Notes
- This is a P3 (low priority) enhancement, so it can be deferred if higher-priority work emerges
- The existing OriginKind enum covers all cases from design doc
- No changes to public API - purely internal debugging enhancement
- Total ~170 LOC is conservative estimate given existing infrastructure

---

**Document created**: 2025-12-13

# Sprint Plan: M-CODEGEN-VALUE-TYPES

## Summary
Implement size-based pointer vs value strategy for record types in Go codegen. Small leaf records (primitives only, ≤4 fields) will generate as values, reducing GC pressure for game loops.

**Duration:** 1 day (~6-8 hours)
**Dependencies:** None (M-CODEGEN-POINTER-RETURN-TYPES already in v0.5.9)
**Risk Level:** Medium (type assertion consistency across multiple codegen paths)
**Design Doc:** [m-codegen-value-types.md](m-codegen-value-types.md)

## Current Status Analysis

### Completed Recently (Last 7 Days)
- ✅ M-DX22: ADT constructor disambiguation (~65 LOC)
- ✅ M-DX18: Function namespacing in codegen (~95 LOC)
- ✅ M-DX17: ConcatList + closure scoping fix (~150 LOC)
- ✅ M-DX16: Inline record literals in match arms (~320 LOC)
- ✅ M-CODEGEN-ADT-DOUBLE-PAREN: Double-paren fix (~145 LOC)

### Velocity
- Recent average: ~150-200 LOC/day for codegen fixes
- Bug fixes: typically 2-4 hours, 50-150 LOC
- This sprint: ~400-500 LOC estimated (enhancement, not bug fix)

### Current Implementation State
- `RecordTypeInfo` has: Name, Fields, FieldTypes (no Category, IsLeaf, IsRecursive)
- `ailangTypeToGo()` in compile.go: ALL user-defined types return `*TypeName`
- `codegen_ops.go`: Record literals always use `&Type{...}`
- No `GoReprForType()` function exists yet

## Proposed Milestones

### Milestone 1: M1-TYPE-ANALYZER - Type Analysis Infrastructure
**Goal:** Add TypeCategory to RecordTypeInfo and implement IsLeafRecord detection
**Estimated:** 120 LOC implementation + 80 LOC tests = 200 LOC
**Duration:** 2 hours

**Tasks:**
- Add `TypeCategory` enum and new fields to `RecordTypeInfo` in `codegen.go`
- Create `type_analyzer.go` with `IsLeafRecord()` function (all fields primitive)
- Add `AnalyzeRecordType()` to classify records during registration
- Unit tests for leaf vs non-leaf detection

**Files to Modify:**
- `internal/gen/golang/codegen.go` - Extend RecordTypeInfo struct (~30 LOC)
- `internal/gen/golang/type_analyzer.go` - NEW file (~90 LOC)
- `internal/gen/golang/type_analyzer_test.go` - NEW file (~80 LOC)

**Acceptance Criteria:**
- [ ] TypeCategory enum with Value/Pointer constants
- [ ] RecordTypeInfo has Category, IsLeaf, FieldCount fields
- [ ] IsLeafRecord correctly identifies primitive-only records
- [ ] Tests cover: leaf (Coord), non-leaf (Entity with nested), recursive

**Risks:**
- Edge cases with primitive slices - Mitigation: Allow `[int]` in leaf records

---

### Milestone 2: M2-SINGLE-SOURCE - GoReprForType as Single Source of Truth
**Goal:** Implement GoReprForType() and wire all codegen paths to use it
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 1.5 hours

**Tasks:**
- Implement `GoReprForType()` method on Generator
- Create lookup helper that consults RecordTypeInfo.Category
- Add tests verifying consistent returns

**Files to Modify:**
- `internal/gen/golang/codegen.go` - Add GoReprForType method (~40 LOC)
- `internal/gen/golang/codegen_test.go` - Tests (~50 LOC)

**Acceptance Criteria:**
- [ ] GoReprForType() returns (goType, isPointer) tuple
- [ ] Returns value=false for leaf records ≤threshold
- [ ] Returns pointer=true for non-leaf, recursive, or large records
- [ ] Unknown types default to pointer (safe fallback)

**Risks:**
- None - isolated implementation

---

### Milestone 3: M3-COMPILE-INTEGRATION - Wire Type Analysis into Compile Pipeline
**Goal:** Analyze types during registration and pass category to codegen
**Estimated:** 80 LOC implementation
**Duration:** 1 hour

**Tasks:**
- Modify `extractRecordTypeInfo()` to include field count
- Call `AnalyzeRecordType()` during `RegisterRecordType()`
- Update `ailangTypeToGo()` to consult a type registry (not just hardcode pointer)

**Files to Modify:**
- `cmd/ailang/compile.go` - Wire analysis into registration (~50 LOC)
- `internal/gen/golang/codegen.go` - Accept category in RegisterRecordType (~30 LOC)

**Acceptance Criteria:**
- [ ] Record types analyzed during compile, not codegen
- [ ] Category stored in RecordTypeInfo at registration time
- [ ] ailangTypeToGo can return `TypeName` (value) or `*TypeName` (pointer)

**Risks:**
- Ordering: types must be analyzed before codegen - Mitigation: analyze in type registration phase

---

### Milestone 4: M4-CODEGEN-PATHS - Update All Codegen Paths
**Goal:** Make record literals, type assertions, and struct fields respect TypeCategory
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 1.5 hours

**Tasks:**
- Update `generateTypedRecord()` for value vs pointer literals
- Update struct field generation in `mapASTType()`
- Update type assertions throughout codegen
- Verify `_impl` wrapper consistency

**Files to Modify:**
- `internal/gen/golang/codegen_ops.go` - Record literals (~30 LOC)
- `internal/gen/golang/adt.go` - mapASTType (~20 LOC)
- `internal/gen/golang/codegen_decl.go` - Type assertions (~30 LOC)
- `internal/gen/golang/codegen_types_test.go` - Tests (~50 LOC)

**Acceptance Criteria:**
- [ ] Leaf records generate `Type{}` literals (not `&Type{}`)
- [ ] Struct fields use value types for leaf records
- [ ] Type assertions use `.(Type)` for values, `.(*Type)` for pointers
- [ ] All paths consistent (no panic from type mismatch)

**Risks:**
- Scattered codegen paths - Mitigation: use checklist from design doc

---

### Milestone 5: M5-CLI-FLAG - Add --value-threshold Flag
**Goal:** Allow projects to tune threshold via CLI
**Estimated:** 50 LOC implementation
**Duration:** 30 minutes

**Tasks:**
- Add `--value-threshold` flag to compile command
- Pass threshold to Generator
- Add validation (0 = all pointers, negative = warning)

**Files to Modify:**
- `cmd/ailang/compile.go` - Flag handling (~30 LOC)
- `internal/gen/golang/codegen.go` - Accept threshold (~20 LOC)

**Acceptance Criteria:**
- [ ] `--value-threshold 4` (default) works
- [ ] `--value-threshold 0` forces all pointers (v0.5.9 behavior)
- [ ] Negative values warn and default to 0

**Risks:**
- None - straightforward flag

---

### Milestone 6: M6-VALIDATION - Integration Testing & Verification
**Goal:** Verify generated code compiles and runs correctly
**Estimated:** 30 LOC tests + manual verification
**Duration:** 30 minutes

**Tasks:**
- Run full test suite
- Compile a project with mixed value/pointer types
- Verify no runtime panics from type assertions
- Update design doc with implementation report

**Acceptance Criteria:**
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Generated Go code compiles without errors
- [ ] No type assertion panics at runtime
- [ ] Design doc updated with implementation report

**Risks:**
- Hidden codegen path missed - Mitigation: grep for all `*` prefixes in type generation

---

## Success Metrics
- Test coverage: No regression (current coverage maintained)
- New tests: 10+ tests for type analysis and category
- All tests passing: ✅
- All linting passing: ✅
- Documentation: Design doc updated with implementation report

## Dependencies
- None - all prerequisites complete

## Open Questions
- None - design review incorporated into design doc

## Notes
- Conservative approach: non-leaf records always pointer (safe)
- ABI metadata persistence (types.json) deferred to follow-up sprint
- Default threshold: 4 fields

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Single source of truth for type categories |
| A2: Replayability | +1 | Threshold flag enables reproducible builds |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Type category verifiable per-record |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Heuristic is deterministic |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Clear heap vs stack implications |
| A10: Composability | +1 | Integrates with existing codegen |
| A11: Structured Failure | +1 | Conservative fallback to pointer |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): GoReprForType is single source of truth
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Heuristic designed for predictability

# Sprint Plan: M-CODEGEN-LIST - Flatten List Literals and Bool Matches

**Sprint ID**: M-CODEGEN-LIST
**Duration**: 1.5 days (~10 hours)
**Risk Level**: Medium
**Design Doc**: [m-codegen-list-flatten.md](m-codegen-list-flatten.md)

## Sprint Summary

**Goal**: Eliminate O(n) closure nesting in Go codegen for list literals and bool match chains.

**Key Deliverables**:
1. Flat list literal generation (evaluate expressions, then build slice)
2. Flat bool match generation (if-else chain instead of nested switches)
3. Validation with stapledons_voyage

**Success Metrics**:
- starmap.go indentation: 58 chars → <20 chars
- List nesting: O(n) → O(1)
- Bool match nesting: O(n) → O(1)
- All tests pass, stapledons_voyage compiles and runs

## Current Status

**Velocity** (last 7 days):
- M-DX11: ~900 LOC (cycles.go, type_report.go, tests)
- Math builtins: ~500 LOC
- Codegen fixes: ~200 LOC
- Average: ~200-250 LOC/day

**Relevant Code Locations**:
- `internal/gen/golang/codegen_ops.go:413` - `generateList()` function
- `internal/gen/golang/codegen_match.go` - `generateMatch()` function
- `internal/gen/block/` - Block IR (already exists from M-CODEGEN-V2)

## Milestones

### M1: List Literal Flattening (~4 hours, ~120 LOC)

**Goal**: Flatten list construction to sequential bindings + single slice.

**Tasks**:
- [ ] Read `codegen_ops.go:generateList()` to understand current implementation
- [ ] Add `flattenListElements()` helper to extract and evaluate expressions
- [ ] Modify `generateList()` to use flat pattern
- [ ] Handle nested lists recursively
- [ ] Add unit tests in `codegen_ops_test.go`

**Files to Modify**:
- `internal/gen/golang/codegen_ops.go` (~50 LOC)
- `internal/gen/golang/codegen_ops_test.go` (~70 LOC new tests)

**Acceptance Criteria**:
- [ ] List `[f(), g(), h()]` generates flat bindings + single `[]interface{}{...}`
- [ ] Nested lists `[[a, b], [c, d]]` flatten correctly
- [ ] No nested IIFEs in list construction
- [ ] Unit tests pass

### M2: Bool Match Flattening (~3 hours, ~100 LOC)

**Goal**: Convert `match <bool> { true => A, false => match <bool> { ... } }` to flat if-else.

**Tasks**:
- [ ] Read `codegen_match.go:generateMatch()` to understand current implementation
- [ ] Add `isBoolMatchChain()` detector function
- [ ] Add `generateFlatBoolMatch()` for if-else chain generation
- [ ] Integrate with existing match codegen (fall back for ADT matches)
- [ ] Add unit tests in `codegen_match_test.go`

**Files to Modify**:
- `internal/gen/golang/codegen_match.go` (~60 LOC)
- `internal/gen/golang/codegen_match_test.go` (~40 LOC new tests)

**Acceptance Criteria**:
- [ ] Chained bool matches become flat if-else
- [ ] ADT matches still work (no regression)
- [ ] Mixed bool/ADT patterns handled correctly
- [ ] Unit tests pass

### M3: Validation & Integration (~2 hours)

**Goal**: Verify fix works with real-world code.

**Tasks**:
- [ ] Regenerate stapledons_voyage sim_gen/ files
- [ ] Verify Go compilation succeeds
- [ ] Measure indentation and closure count
- [ ] Run stapledons_voyage tests (if any)
- [ ] Update design doc with results

**Acceptance Criteria**:
- [ ] starmap.go max indentation < 20 chars
- [ ] initLocalCatalog_impl has no 60-level nesting
- [ ] spectralFromRoll has flat if-else
- [ ] All existing AILANG tests pass

### M4: Documentation & Cleanup (~1 hour)

**Goal**: Document the fix and clean up.

**Tasks**:
- [ ] Update design doc status to Implemented
- [ ] Add inline comments explaining flattening logic
- [ ] Update CHANGELOG.md
- [ ] Send completion message to stapledons_voyage

**Acceptance Criteria**:
- [ ] Design doc moved to implemented/
- [ ] CHANGELOG updated
- [ ] stapledons_voyage notified

## Day-by-Day Plan

### Day 1 (~6 hours)
- **Morning** (3h): M1 - List literal flattening
- **Afternoon** (3h): M2 - Bool match flattening

### Day 2 (~4 hours)
- **Morning** (2h): M3 - Validation with stapledons_voyage
- **Afternoon** (2h): M4 - Documentation and cleanup

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Evaluation order changes | High | Test left-to-right order preservation |
| Variable name collisions | Medium | Use unique prefixes (_list_e0, etc.) |
| ADT match regression | Medium | Run full test suite after changes |
| Complex nested patterns | Medium | Add edge case tests |

## Dependencies

- M-CODEGEN-V2 (complete) - Block IR infrastructure
- stapledons_voyage access for validation

## Open Questions

None - design is clear from design doc.

---

**Sprint created**: 2025-12-11
**Last updated**: 2025-12-11

# Sprint Plan: M-DX11 Cyclic Type Diagnostics

## Summary
Complete the remaining phases of M-DX11 to provide full cyclic type diagnostics: finish Phase 1 cleanup, implement the `ailang debug cycles` command (Phase 3), and add CI integration (Phase 4).

**Duration:** 2 days (~6 hours implementation)
**Dependencies:** None - Phase 2 (traverse library) already complete
**Risk Level:** Low - well-defined scope, existing infrastructure

## Current Status Analysis

### Completed Recently
- Phase 1 (Quick Wins): ~90% complete
  - `--timeout` flag to `ailang check` with stack dump
  - `--debug-compile` with phase timing breakdown
  - `SafeTypeString` (170 LOC) for depth-limited type display
- Phase 2 (Safe Traversal): 100% complete
  - `internal/types/traverse/traverse.go` (195 LOC)
  - `internal/types/traverse/wrappers.go` (213 LOC)
  - `internal/types/traverse/traverse_test.go` (726 LOC, 31 tests)

### Velocity
- Recent average: ~200-300 LOC/day (from CHANGELOG)
- M-CODEGEN fixes: ~120 LOC implementation + tests
- Traverse library: ~400 LOC total (completed)
- Estimated capacity: 300 LOC for this sprint

### Remaining from Design Doc
- Phase 1 Cleanup: ~50 LOC (tests + documentation)
- Phase 3 (Cycle Detection Command): ~300 LOC
- Phase 4 (CI Integration): ~50 LOC

**Total Remaining:** ~400 LOC

## Proposed Milestones

### Milestone 1: Phase 1 Cleanup (~0.5 hours)
**Goal:** Complete the remaining Phase 1 items for defensive depth limiting

**Estimated:** ~50 LOC (40 test + 10 doc comments)
**Duration:** 0.5 hours

**Tasks:**
1. Add truncation tests for SafeTypeString:
   - Test deeply nested types (>100 levels) trigger truncation
   - Test cyclic types (μ-type) trigger cycle marker
   - Verify completion within test timeout (no hang)
2. Add API contract comment to `internal/types/types.go` header
3. Mark Phase 1 as complete in design doc

**Acceptance Criteria:**
- [ ] `TestStringTruncatesOnDeepTypes` passes
- [ ] `TestStringTruncatesOnCyclicTypes` passes
- [ ] types.go header documents String() as diagnostic-only
- [ ] All tests passing
- [ ] Linting clean

**Files Changed:**
- `internal/types/safe_string_test.go` (new, ~40 LOC)
- `internal/types/types.go` (add header comment, ~10 LOC)
- `design_docs/planned/v0_5_9/m-dx11-cyclic-type-diagnostics.md` (mark Phase 1 complete)

**Risks:**
- None - straightforward test additions

---

### Milestone 2: Cycle Detection Command (Phase 3) (~4 hours)
**Goal:** Implement `ailang debug cycles <file>` command to identify cyclic types

**Estimated:** 200 LOC implementation + 100 LOC tests = 300 LOC total
**Duration:** 4 hours

**Tasks:**
1. Create `cmd/ailang/debug_cycles.go`:
   - Parse and type-check input file
   - Use `traverse.WalkWithCycleCallback` to detect cycles
   - Build type → source location mapping from AST
   - Classify cycles as "expected" (stdlib) vs "suspicious" (user)
   - Output human-readable report
2. Add `--json` flag for machine-readable output
3. Wire up `debug cycles` subcommand in `cmd/ailang/root.go`
4. Create test file `examples/complex_types.ail`:
   - Expected cycle: recursive ADT (List[a])
   - Suspicious cycle: mutually recursive records

**Acceptance Criteria:**
- [ ] `ailang debug cycles file.ail` produces human-readable output
- [ ] `ailang debug cycles --json file.ail` produces valid JSON
- [ ] Cycles classified correctly (expected vs suspicious)
- [ ] Source locations included in output
- [ ] `examples/complex_types.ail` produces expected cycle report
- [ ] All tests passing
- [ ] Linting clean

**Output Format (Human):**
```
Analyzing type graph for sim/test_combined...

Found 2 cyclic type references:

Cycle 1 [SUSPICIOUS]: Person → friends: [Person]
  Location: examples/complex_types.ail:15
  Depth: 2 nodes

Cycle 2 [EXPECTED]: List[a] (recursive ADT)
  Location: examples/complex_types.ail:5
  Depth: 1 node (self-referential)

Summary:
  - 1 suspicious cycle (may cause hangs)
  - 1 expected cycle (stdlib pattern)
```

**Output Format (JSON):**
```json
{
  "file": "examples/complex_types.ail",
  "cycles": [
    {
      "kind": "suspicious",
      "path": ["Person", "friends: [Person]"],
      "location": {"file": "...", "line": 15},
      "depth": 2
    }
  ],
  "summary": {"suspicious": 1, "expected": 1, "total": 2}
}
```

**Files Changed:**
- `cmd/ailang/debug_cycles.go` (new, ~150 LOC)
- `cmd/ailang/root.go` (add subcommand, ~10 LOC)
- `cmd/ailang/debug_cycles_test.go` (new, ~100 LOC)
- `examples/complex_types.ail` (new, ~30 LOC)
- `design_docs/planned/v0_5_9/m-dx11-cyclic-type-diagnostics.md` (mark Phase 3 complete)

**Risks:**
- Type → source location mapping may require AST traversal alongside type traversal
  - Mitigation: Use TypeChecker's SourceMap or build during elaboration

---

### Milestone 3: CI Integration (Phase 4) (~1 hour)
**Goal:** Add timeout and cycle detection to CI pipeline

**Estimated:** ~50 LOC YAML changes
**Duration:** 1 hour

**Tasks:**
1. Add `go test -timeout 60s` to CI test step
2. Add `ailang check --timeout 60s examples/` step
3. Add `ailang debug cycles --json examples/complex_types.ail` step
4. Configure CI failure on suspicious cycles (jq validation)
5. Upload cycle analysis as artifact

**Acceptance Criteria:**
- [ ] CI uses `go test -timeout 60s`
- [ ] CI runs `ailang check --timeout 60s`
- [ ] CI runs cycle detection on test files
- [ ] Suspicious cycles cause CI failure
- [ ] Cycle analysis uploaded as artifact
- [ ] All CI jobs passing

**Files Changed:**
- `.github/workflows/ci.yml` (~50 LOC additions)
- `design_docs/planned/v0_5_9/m-dx11-cyclic-type-diagnostics.md` (mark Phase 4 complete)

**Risks:**
- May need to adjust timeout values based on CI runner speed
  - Mitigation: Start with generous 60s, adjust if needed

---

## Success Metrics
- Test coverage: traverse package maintains 90%+ coverage
- Examples passing: `examples/complex_types.ail` verifies correctly
- Documentation:
  - `docs/guides/debugging.md` updated with `debug cycles` command
  - types.go header documents String() contract
- All tests passing
- All linting passing

## Day-by-Day Breakdown

### Day 1 (3 hours)
- **Hour 1:** Milestone 1 - Phase 1 cleanup (tests + docs)
- **Hours 2-3:** Milestone 2 - Start cycle detection command
  - Implement basic command structure
  - Wire up traverse-based cycle detection
  - Human-readable output

### Day 2 (3 hours)
- **Hours 1-2:** Milestone 2 - Complete cycle detection
  - Add `--json` output
  - Type → source location mapping
  - Create `examples/complex_types.ail`
  - Write tests
- **Hour 3:** Milestone 3 - CI integration
  - Update `.github/workflows/ci.yml`
  - Test CI changes locally
  - Final verification

## Dependencies
- traverse package (complete)
- SafeTypeString (complete)
- `--timeout` flag (complete)

## Open Questions
None - design doc is comprehensive and all dependencies are satisfied.

## Notes
- The traverse library uses a single mode with defer-based cleanup (not the two-mode design in the original doc) - this achieves equivalent safety with simpler code
- Cycle classification heuristic: paths containing "std/" are expected, others are suspicious
- The command reuses existing pipeline infrastructure for parsing/type-checking

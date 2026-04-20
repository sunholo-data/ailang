# Sprint Plan: M-CODEGEN-MULTIMODULE-BUGS

## Summary

Fix 3 Go codegen bugs that prevent `go build` from succeeding on DocParse's 19-module multi-module compilation. All bugs are in `internal/gen/golang/` and `cmd/ailang/compile.go`.

**Duration:** 1 day (4 milestones, ~4h total)
**Dependencies:** None — all bugs are self-contained in codegen
**Risk Level:** Low — targeted fixes with clear root causes, existing test infrastructure

## Current Status Analysis

### Completed Recently
- ✅ M-WASM-DICTIONARY-DISPATCH: Fixed WASM dictionary dispatch bugs
- ✅ M-PERF7: Batch mode + string char builtins
- ✅ M-ITERATIVE-LIST: Iterative Go builtins for map/filter/foldl
- ✅ v0.9.2 release: Split 45 oversized files, WASM fixes

### Velocity
- Recent average: ~300-500 LOC/day (codegen area)
- Estimated capacity: ~200 LOC for this sprint (small, targeted fixes)

### Remaining from Design Doc
- ⏳ Bug 1: Non-lambda let bindings missing module prefix (~20 LOC)
- ⏳ Bug 2: Cross-module function collision from shared Generator state (~30 LOC)
- ⏳ Bug 3: Bracket syntax error in markdown_parser codegen (~TBD LOC)
- 📋 Integration verification with DocParse

## Proposed Milestones

### M1: FIX_CONSTANT_DEDUP — Fix constant redeclaration across modules

**Goal:** Non-lambda let bindings get module-prefixed Go names, preventing "redeclared in this block" for constants.

**Estimated:** 20 LOC implementation + 40 LOC tests = 60 LOC
**Duration:** ~1h

**Tasks:**
1. Add module prefix to `generateTopLevelLet` in `codegen_decl.go` for non-exported bindings
2. Add `emittedVars` dedup set to `Generator` to prevent duplicate emissions within a module
3. Write regression test: two modules with same constant → distinct Go var names
4. Run `make test`

**Acceptance Criteria:**
- [ ] `generateTopLevelLet` applies `moduleName__` prefix for non-exported let bindings
- [ ] Duplicate let binding emissions within a module are deduplicated
- [ ] Unit test covers cross-module constant naming
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Variable references must also resolve the prefixed name → check `topLevelFuncs` map handles non-lambda lets

### M2: FIX_GENERATOR_ISOLATION — Isolate per-module Generator state

**Goal:** Prevent cross-module function name collisions by resetting per-module state between modules in the compile loop.

**Estimated:** 30 LOC implementation + 50 LOC tests = 80 LOC
**Duration:** ~1.5h

**Tasks:**
1. Add `ResetPerModuleState()` method to Generator that clears `topLevelFuncs`, `topLevelImplFuncs`, `funcParamTypes`, `funcReturnTypes` (but preserves shared ADT/record type registrations)
2. Call `ResetPerModuleState()` at the start of each module in the compile loop (`compile.go:429`)
3. Write regression test: two modules with same function name → no collision
4. Run `make test`

**Acceptance Criteria:**
- [ ] Per-module maps are reset between modules in compile loop
- [ ] Shared state (ADT constructors, record types, value converters) is preserved
- [ ] Unit test covers cross-module function name isolation
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Clearing too much state could break cross-module references → only clear name-mapping caches, not type registrations

### M3: FIX_BRACKET_SYNTAX — Fix bracket syntax error in list codegen

**Goal:** Investigate and fix the `expected operand, found ']'` syntax error in generated markdown_parser.go.

**Estimated:** ~30 LOC (depends on root cause) + 30 LOC tests = 60 LOC
**Duration:** ~1h

**Tasks:**
1. Reproduce: compile a test AILANG file with complex list patterns/literals to trigger the error
2. If DocParse files not available locally, create a minimal reproduction from the error description
3. Inspect generated Go output to identify the malformed construct
4. Fix the codegen case (likely `codegen_match.go` or `codegen_ops.go`)
5. Add regression test
6. Run `make test`

**Acceptance Criteria:**
- [ ] Root cause identified and documented
- [ ] Fix generates valid Go syntax for the triggering construct
- [ ] Regression test added
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- May be a deeper gap if the construct involves unsupported pattern matching → if so, document as limitation and create follow-up issue
- DocParse source may not be available locally for exact reproduction → use minimal repro

### M4: INTEGRATION_VERIFY — Full integration verification

**Goal:** Verify all 3 fixes work together. Update docs.

**Estimated:** 10 LOC (CHANGELOG) + manual verification
**Duration:** ~30m

**Tasks:**
1. Run `make test` and `make lint`
2. Run `make verify-examples`
3. If DocParse sources available: run full `ailang compile --emit-go` + `go build`
4. Update CHANGELOG.md with fixes
5. Update design doc status

**Acceptance Criteria:**
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated
- [ ] Design doc updated with implementation notes

## Success Metrics
- All existing tests passing: ✅
- All linting clean: ✅
- New regression tests: 3+ (one per bug)
- `make verify-examples` passing: ✅
- CHANGELOG.md updated: ✅

## Dependencies
- None — all fixes are in codegen, no parser/type system changes needed

## Open Questions
- **DocParse sources**: Are the 19 .ail files available locally for integration testing? If not, we test with synthetic multi-module projects.
- **Bug 3 exact trigger**: The error message mentions line 66:200 — we need to either reproduce or inspect the generated file to pinpoint the construct. If we can't reproduce, we document as "needs DocParse sources" and defer.

# Sprint Plan: M-DX11-DEBUG-TYPES-CLI (Type Inference Debug CLI)

## Summary

Add `--debug-types` CLI flag for formatted type inference debugging output, building on the TypeReport API from v0.5.10.

**Duration:** 1 day (~6 hours)
**Dependencies:** M-DX11-TYPE-REPORT (v0.5.10) - ✅ Complete
**Risk Level:** Low (well-defined scope, foundation exists)
**Design Doc:** [m-dx11-debug-types-cli.md](m-dx11-debug-types-cli.md)

## Current Status Analysis

### What Already Exists
- ✅ `internal/types/type_report.go` - TypeReport API (~180 LOC)
- ✅ `internal/types/type_report_test.go` - Unit tests (~140 LOC)
- ✅ `cmd/ailang/debug.go` - Has `debug types` stub mentioning v0.5.11
- ✅ `docs/docs/guides/debugging.md` - Debug guide for env vars and CLI flags
- ✅ Substitution chain following in `ApplySubstitution`

### Current Gaps
- ❌ No `--debug-types` flag on `ailang run` command
- ❌ No formatted CLI output of TypeReport data
- ❌ No `--node <id>` filter for focused debugging
- ❌ No TypeDebugSink pattern for zero-overhead production mode
- ❌ debugging.md doesn't document the new flag

### Velocity
- Recent average: ~200-400 LOC/day (from v0.5.9-v0.5.10 work)
- Estimated capacity: ~365 LOC for this sprint

## Proposed Milestones

### Milestone 1: TypeDebugSink Interface
**Goal:** Create zero-overhead debug sink pattern for type inference events
**Estimated:** 80 LOC implementation
**Duration:** 1 hour

**Tasks:**
1. Create `internal/types/debug_sink.go`:
   - Define `TypeDebugSink` interface (OnFreshTypeVar, OnUnify, OnSubstitute, OnDefault, OnConstraintAdd, OnConstraintResolve)
   - Implement `NoOpDebugSink` (zero overhead, no-op methods)
   - Implement `VerboseDebugSink` (collects events for later formatting)

**Files:**
- Create: `internal/types/debug_sink.go` (~80 LOC)

**Acceptance Criteria:**
- [ ] Interface defined with all methods
- [ ] NoOpDebugSink has zero allocations
- [ ] VerboseDebugSink collects DebugEvent structs
- [ ] Unit tests pass

### Milestone 2: --debug-types Flag on `ailang run`
**Goal:** Add CLI flag that enables debug output during type checking
**Estimated:** 50 LOC
**Duration:** 45 min

**Tasks:**
1. Add `--debug-types` bool flag to run command in `cmd/ailang/run.go`
2. Add `--node <id>` uint flag for filtering
3. Thread debug sink through pipeline when flag is set
4. Update run command help text

**Files:**
- Modify: `cmd/ailang/run.go` (~30 LOC)
- Modify: `cmd/ailang/main.go` (~20 LOC for flag parsing)

**Acceptance Criteria:**
- [ ] `ailang run --debug-types test.ail` compiles
- [ ] `ailang run --debug-types --node 42 test.ail` compiles
- [ ] Flags parsed and available in pipeline
- [ ] Help text updated

### Milestone 3: TypeDebugDumper Implementation
**Goal:** Create formatted output renderer for debug information
**Estimated:** 150 LOC
**Duration:** 2 hours

**Tasks:**
1. Create `cmd/ailang/debug_types.go`:
   - Define `TypeDebugDumper` struct
   - Implement substitution map rendering with chain resolution
   - Implement constraints rendering (before/after defaulting)
   - Implement CoreTI entries rendering (raw vs resolved)
   - Support `--node` filtering

2. Output format (from design doc):
   ```
   === Type Inference Debug ===

   [Substitution Map]
     α3 → α7 → float (CHAIN, final: float)
     α7 → float (direct)

   [Active Constraints] (before defaulting)
     Num α3 at line 5
     Fractional α7 at line 6

   [CoreTI Entries]
     NodeID 42: float
       Raw: α7
       Resolved: float
   ```

**Files:**
- Create: `cmd/ailang/debug_types.go` (~150 LOC)

**Acceptance Criteria:**
- [ ] Substitution map shows chain resolution
- [ ] Constraints show class name, type, source location
- [ ] CoreTI shows both raw and resolved types
- [ ] `--node <id>` filters output to specific node
- [ ] Output parseable by AI tools (structured format)

### Milestone 4: Wire Up Pipeline Integration
**Goal:** Connect TypeDebugSink to CoreTypeChecker and format output
**Estimated:** 35 LOC
**Duration:** 30 min

**Tasks:**
1. Modify `internal/types/typechecker_core.go`:
   - Add `DebugSink TypeDebugSink` field
   - Call sink methods at appropriate points
   - Default to NoOpDebugSink (zero overhead)

2. Modify `internal/pipeline/pipeline.go`:
   - Pass VerboseDebugSink when --debug-types is enabled
   - Format and print output after type checking

**Files:**
- Modify: `internal/types/typechecker_core.go` (~15 LOC)
- Modify: `internal/pipeline/pipeline.go` (~20 LOC)

**Acceptance Criteria:**
- [ ] TypeChecker uses sink for debug events
- [ ] Pipeline passes sink based on CLI flag
- [ ] Debug output printed after type checking phase
- [ ] No performance impact when flag not set

### Milestone 5: Documentation Update
**Goal:** Update debugging.md with new CLI flag documentation
**Estimated:** 50 LOC docs
**Duration:** 45 min

**Tasks:**
1. Update `docs/docs/guides/debugging.md`:
   - Add new section for `--debug-types` flag
   - Document output format
   - Add usage examples
   - Add troubleshooting workflow using new flag

2. Update `cmd/ailang/debug.go`:
   - Change `debug types` stub to reference the new `--debug-types` flag
   - Update help text

3. Update CHANGELOG.md

**Files:**
- Modify: `docs/docs/guides/debugging.md` (~40 LOC)
- Modify: `cmd/ailang/debug.go` (~10 LOC)
- Modify: `CHANGELOG.md` (~15 LOC)

**Acceptance Criteria:**
- [ ] debugging.md has new section for --debug-types
- [ ] Examples show real usage with expected output
- [ ] Troubleshooting section references new flag
- [ ] CHANGELOG entry added

## Success Metrics

- All tests passing: ✅
- Linting clean: ✅
- `ailang run --debug-types test.ail` shows formatted debug output
- `ailang run --debug-types --node 42 test.ail` filters to node 42
- NoOpDebugSink has zero allocations (verify with benchmark)
- Documentation updated with examples

## Files Summary

**New files (~230 LOC):**
- `internal/types/debug_sink.go` (~80 LOC)
- `cmd/ailang/debug_types.go` (~150 LOC)

**Modified files (~135 LOC):**
- `cmd/ailang/run.go` (~30 LOC)
- `cmd/ailang/main.go` (~20 LOC)
- `internal/types/typechecker_core.go` (~15 LOC)
- `internal/pipeline/pipeline.go` (~20 LOC)
- `docs/docs/guides/debugging.md` (~40 LOC)
- `cmd/ailang/debug.go` (~10 LOC)

**Total:** ~365 LOC

## Testing Strategy

1. **Unit tests for TypeDebugSink**:
   - NoOpDebugSink has no side effects
   - VerboseDebugSink captures all events

2. **Integration test**:
   - Run `ailang run --debug-types examples/runnable/factorial.ail`
   - Verify output format matches specification
   - Verify no output when flag not provided

3. **Benchmark**:
   - Verify NoOpDebugSink adds zero overhead
   - Compare with/without flag on large file

## Open Questions

None - design is clear from parent M-DX11 design doc.

## Notes

- TypeDebugSink is the foundation for future provenance tracking (Phase 4)
- Keep output human-readable but also parseable by AI tools
- Focus on making common debugging workflows easier

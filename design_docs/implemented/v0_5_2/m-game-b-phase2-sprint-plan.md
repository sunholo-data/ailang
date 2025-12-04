# M-GAME-B Phase 2 Sprint Plan

**Sprint ID**: M-GAME-B2
**Goal**: Complete Go codegen for stapledon game modules
**Duration**: 1 day (~5 hours)
**Risk Level**: Medium (new runtime package, reachability analysis)
**Design Doc**: [m-game-b-phase2-go-codegen.md](m-game-b-phase2-go-codegen.md)

## Current Status

### Phase 1 Complete (Today)
- Cross-module ADT constructor registration
- types.go generation for imported ADTs
- Type assertions for ADT constructor arguments
- Literal type conversions (int64(0) not 0.(int64))

### Remaining Issues
1. Slice type conversions ([]interface{} → []T)
2. Missing runtime helpers (Show, ConcatString, Log)
3. Cross-module function generation

## Velocity Analysis

Based on last 14 days:
- M-DX11: ~157 LOC in 1 session
- M-GAME-B Phase 1: ~100 LOC in 4 quick fixes
- Average: ~120-150 LOC/day when focused

**This sprint**: ~205 LOC estimated → 1 day is realistic

## Milestones

### M-GAME-B2.1: Runtime Package (~105 LOC, 1.5h)

**Description**: Create shared `runtime/` package with helpers

**Files to create:**
- `runtime/show.go` (~30 LOC) - Show function
- `runtime/string.go` (~15 LOC) - ConcatString
- `runtime/io.go` (~20 LOC) - Log function
- `runtime/runtime_test.go` (~40 LOC) - Unit tests

**Acceptance Criteria:**
- [ ] `runtime/` package compiles
- [ ] Show handles int64, float64, string, bool, any
- [ ] ConcatString concatenates any two values
- [ ] Log prints to stdout, returns unit
- [ ] Unit tests pass for all helpers

**Dependencies**: None

### M-GAME-B2.2: Slice Type Conversion (~50 LOC, 1h)

**Description**: Generate type-aware slice conversion from AILANG types

**Files to modify:**
- `internal/gen/golang/codegen_expr.go` (~40 LOC) - Slice conversion logic
- `internal/gen/golang/codegen.go` (~10 LOC) - Track needed conversions

**Acceptance Criteria:**
- [ ] Detect slice field types from ADTConstructorInfo
- [ ] Generate convertToInt64Slice for [int]
- [ ] Generate convertToRecordSlice for [{...}]
- [ ] Slice arguments to ADT constructors compile

**Dependencies**: M-GAME-B2.1 (for test harness)

### M-GAME-B2.3: Cross-Module Functions (~50 LOC, 1.5h)

**Description**: Generate reachable functions from imported modules with prefixes

**Files to modify:**
- `internal/gen/golang/codegen.go` (~40 LOC) - Reachability + prefixing
- `cmd/ailang/compile.go` (~10 LOC) - Pass imported Core progs

**Acceptance Criteria:**
- [ ] Module prefix transform: sim/protocol.foo → SimProtocol_Foo
- [ ] Only reachable functions generated (not all)
- [ ] Function calls use prefixed names
- [ ] stapledon test_ctor.ail compiles with protocol.ail

**Dependencies**: M-GAME-B2.1, M-GAME-B2.2

## Day-by-Day Plan

### Day 1 (5 hours total)

| Time | Task | Milestone |
|------|------|-----------|
| 0:00-1:30 | Create runtime/ package with Show, ConcatString, Log | M-GAME-B2.1 |
| 1:30-2:30 | Implement slice type conversion in codegen | M-GAME-B2.2 |
| 2:30-4:00 | Add cross-module function generation with prefixes | M-GAME-B2.3 |
| 4:00-5:00 | Integration test with stapledon, fix edge cases | All |

## Success Metrics

- [ ] `runtime/` package exists and tests pass
- [ ] stapledon `go build ./...` succeeds
- [ ] No dead code generated (only reachable functions)
- [ ] All existing Go codegen tests pass
- [ ] Interpreter oracle test validates semantics

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Reachability misses edge cases | Fall back to generating all if analysis fails |
| Runtime package location issues | Use internal/runtime/ if module path problems |
| Module prefix collisions | Deterministic transform prevents this |

## Handoff Notes

After sprint completion:
1. Send message to stapledon with updated compile command
2. Move design doc to implemented/v0_5_2/
3. Update CHANGELOG.md with LOC counts

---

**Created**: 2025-12-03
**Status**: Ready for execution

# Sprint Plan: M-DX12 Typed World Boundary Marshalling

## Sprint Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-DX12 |
| **Goal** | Generate typed slices for `[ADT]` fields in profile-exported types |
| **Duration** | 1 day (~6 hours) |
| **Risk Level** | Low |
| **LOC Estimate** | ~250 LOC (implementation + tests) |
| **Design Doc** | [m-dx12-typed-adt-slices.md](m-dx12-typed-adt-slices.md) |

## Current Status

### Completed Prerequisites
- ✅ M-DX11 (Named ADT Fields) - COMPLETE
- ✅ Multi-file compilation support
- ✅ Effect handler interfaces
- ✅ Basic slice conversion helpers exist (`ConvertToInt64Slice`, `ConvertToStringSlice`)

### Velocity Reference
Recent work shows:
- M-DX11-NAMED-ADT: ~120 LOC in ~1 hour (4 milestones)
- Multi-file compilation: ~150 LOC in ~2 hours
- Effect handlers: ~110 LOC in ~1.5 hours

**Estimated velocity**: 100-150 LOC/hour for Go codegen work

## Milestones

### M-DX12.1: ADT Slice Converter Generation (~1.5h)
**Goal**: Generate type-safe converter functions for each ADT used in lists

**Tasks**:
- [ ] Add `generateADTSliceConverter()` to `codegen_runtime.go`
- [ ] Track which ADTs are used in list fields during type mapping
- [ ] Generate one converter per ADT type: `convertToDrawCmdSlice()`
- [ ] Include fail-fast panic with clear type mismatch message

**Files**:
- `internal/gen/golang/codegen_runtime.go` (+60 LOC)
- `internal/gen/golang/codegen.go` (+15 LOC - tracking)

**Acceptance Criteria**:
- [ ] `convertToDrawCmdSlice()` generated for each ADT used in lists
- [ ] Panic message includes expected vs actual type
- [ ] Empty input returns empty slice (not nil)
- [ ] Nil input returns nil

**Example output**:
```go
func convertToDrawCmdSlice(v interface{}) []*DrawCmd {
    if v == nil {
        return nil
    }
    src, ok := v.([]interface{})
    if !ok {
        panic(fmt.Sprintf("convertToDrawCmdSlice: expected []interface{}, got %T", v))
    }
    if len(src) == 0 {
        return []*DrawCmd{}
    }
    out := make([]*DrawCmd, len(src))
    for i, e := range src {
        dc, ok := e.(*DrawCmd)
        if !ok {
            panic(fmt.Sprintf("convertToDrawCmdSlice: element %d: expected *DrawCmd, got %T", i, e))
        }
        out[i] = dc
    }
    return out
}
```

### M-DX12.2: Type Mapping Update (~1h)
**Goal**: Change `[ADT]` fields to generate typed slices instead of `interface{}`

**Tasks**:
- [ ] Update `mapASTType()` in `adt.go` to return `[]*ADTType` for list-of-ADT
- [ ] Track ADT types used in lists for converter generation
- [ ] Keep internal `[]interface{}` representation in funcs.go

**Files**:
- `internal/gen/golang/adt.go` (+25 LOC)

**Acceptance Criteria**:
- [ ] `[DrawCmd]` field generates as `[]*DrawCmd` in struct
- [ ] Constructor functions still use `interface{}` internally
- [ ] Existing primitive slice types (`[]int64`, `[]string`) unchanged

**Before**:
```go
type FrameOutput struct {
    Draw   interface{}  // Was interface{}
}
```

**After**:
```go
type FrameOutput struct {
    Draw   []*DrawCmd  // Now typed!
}
```

### M-DX12.3: Marshalling Integration (~1.5h)
**Goal**: Wire converters into record/struct construction

**Tasks**:
- [ ] Update record construction to call converters for ADT list fields
- [ ] Generate converter calls in constructor functions
- [ ] Handle nested ADT lists (e.g., `[[DrawCmd]]` - warn or error)

**Files**:
- `internal/gen/golang/codegen_expr.go` (+40 LOC)
- `cmd/ailang/compile.go` (+20 LOC)

**Acceptance Criteria**:
- [ ] Record construction automatically converts `[]interface{}` → `[]*ADT`
- [ ] Conversion happens at struct boundaries, not in function bodies
- [ ] Generated code compiles and builds

### M-DX12.4: Testing & Validation (~1h)
**Goal**: Verify the feature works end-to-end

**Tasks**:
- [ ] Create test AILANG file with `[ADT]` record field
- [ ] Compile and verify generated types.go has typed slice
- [ ] Compile and verify generated funcs.go uses converters
- [ ] Build and run generated code
- [ ] Verify panic messages are helpful on type mismatch

**Files**:
- Test file: `/tmp/codegen_test/adt_slice_test/test.ail`
- Unit tests: `internal/gen/golang/codegen_test.go` (+40 LOC)

**Acceptance Criteria**:
- [ ] All existing golang codegen tests pass
- [ ] New test for ADT slice conversion passes
- [ ] Generated code builds with `go build`
- [ ] No `interface{}` in public API for `[ADT]` fields

## Day Breakdown

| Time | Milestone | Tasks |
|------|-----------|-------|
| Hour 1-1.5 | M-DX12.1 | Converter generation |
| Hour 1.5-2.5 | M-DX12.2 | Type mapping update |
| Hour 2.5-4 | M-DX12.3 | Marshalling integration |
| Hour 4-5 | M-DX12.4 | Testing & validation |
| Hour 5-6 | Buffer | Bug fixes, edge cases |

## Success Metrics

| Metric | Target |
|--------|--------|
| LOC Added | ~200-250 |
| Tests Added | 4-6 new test cases |
| Build Status | All tests pass |
| API Surface | No `interface{}` for `[ADT]` fields |

## Risk Factors

| Risk | Mitigation |
|------|------------|
| Nested ADT lists `[[DrawCmd]]` | Start with single-level, error on nested |
| Constructor parameter ordering | Follow existing pattern from M-GAME-B |
| Performance of converters | Acceptable at profile boundaries |

## Dependencies

- **M-DX11**: COMPLETE - Named field infrastructure
- **Existing converters**: `ConvertToInt64Slice`, etc. as patterns

## Open Questions

1. Should converters live in `runtime.go` or per-file? → **Decision: In runtime.go alongside other converters**
2. Handle `[[DrawCmd]]` (nested)? → **Decision: Error/warn for v0.5.x, defer to v0.6**

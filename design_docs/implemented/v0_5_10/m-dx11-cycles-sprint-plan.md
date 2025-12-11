# Sprint Plan: M-DX11-CYCLES (Type Graph Cycle Detection)

## Summary

Enhance the `ailang debug cycles` command to perform proper type-graph level cycle detection (not just AST-level), correctly identify generic recursive ADTs like List[a] and Tree[a], and provide detailed cycle paths with source locations.

**Duration:** 1 day (~4 hours)
**Dependencies:** M-DX11 Phase 1 (v0.5.9) - ✅ Complete
**Risk Level:** Low (foundation exists, this is enhancement)
**Design Doc:** [m-dx11-cycles.md](m-dx11-cycles.md)

## Current Status Analysis

### What Already Exists
- ✅ `cmd/ailang/debug.go` - Debug command with cycles subcommand (~477 LOC)
- ✅ `internal/types/traverse/traverse.go` - Safe type traversal (~195 LOC)
- ✅ `internal/types/traverse/wrappers.go` - Helper functions (~160 LOC)
- ✅ `examples/complex_types.ail` - Test file with cyclic types (~26 LOC)
- ✅ JSON output format working
- ✅ Basic AST-level cycle detection (finds simple self-references)

### Current Gaps
- ❌ List[a] and Tree[a] cycles NOT detected (AST-level misses generic type params)
- ❌ Cycle path information is vague (`...` placeholder instead of actual field paths)
- ❌ No type-graph level analysis (only AST pattern matching)
- ❌ Classification misses common recursive patterns in the example file

### Velocity
- Recent average: ~200-400 LOC/day (from v0.5.9 work)
- Estimated capacity: ~400 LOC for this sprint

## Proposed Milestones

### Milestone 1: Type-Graph Cycle Detection
**Goal:** Create proper type-graph cycle detection that works on resolved types, not just AST patterns
**Estimated:** 150 LOC implementation + 200 LOC tests = 350 LOC
**Duration:** 2-3 hours

**Tasks:**
1. Create `internal/types/cycles.go`:
   - `DetectCycles(typeDefs map[string]*TypeDef) []CycleInfo`
   - Walks type definitions and checks for self-references through fields
   - Handles generic types (List[a] contains List[a] reference)
   - Extracts field path for cycle visualization

2. Create `internal/types/cycles_test.go`:
   - Test simple recursive ADT detection (List, Tree)
   - Test record field cycle detection (Person.friends)
   - Test non-cyclic types return empty
   - Test generic type parameter cycles

**Acceptance Criteria:**
- [ ] `ailang debug cycles examples/complex_types.ail` detects List[a], Tree[a], and Person cycles
- [ ] Cycle paths show field names (e.g., "Person → friends: [Person] → Person")
- [ ] All tests passing (`go test ./internal/types/...`)
- [ ] Linting clean (`make lint`)

**Risks:**
- Type resolution complexity - Mitigation: Work at AST type-reference level, not fully resolved types

### Milestone 2: Improved Output and Classification
**Goal:** Better cycle path formatting, source locations, and smart classification
**Estimated:** 50 LOC changes
**Duration:** 1 hour

**Tasks:**
1. Update `cmd/ailang/debug.go`:
   - Call new type-graph detector from `runDebugCycles`
   - Format cycle paths with field traversal info
   - Improve classification heuristics for List, Tree patterns

2. Update output format:
   - Show actual field paths in cycle output
   - Add source line numbers where available

**Acceptance Criteria:**
- [ ] List[a] and Tree[a] marked as "expected" (recursive ADT patterns)
- [ ] Person marked as "suspicious" (non-standard pattern)
- [ ] Cycle paths readable (not just "...")
- [ ] JSON output includes improved path data

## Success Metrics
- Test coverage: >80% for `internal/types/cycles.go`
- Examples: `examples/complex_types.ail` correctly analyzed
- Documentation: CHANGELOG.md updated
- All tests passing: ✅
- All linting passing: ✅

## Implementation Approach

The key insight is that current implementation does **AST-level** pattern matching:
```go
// Current: checks if TypeName appears in its own definition fields
if ref == typeName {
    // cycle found
}
```

This misses generic types because `List[a]` containing `List[a]` isn't a direct string match when the type params are involved.

The fix is to check type references more carefully:
```go
// New: check if base type name matches (ignoring type params)
func getBaseTypeName(ref string) string {
    // "List[a]" → "List"
    // "Person" → "Person"
}

if getBaseTypeName(ref) == typeName {
    // cycle found for recursive ADT
}
```

## Files to Create/Modify

**New files:**
- `internal/types/cycles.go` (~150 LOC) - Type-graph cycle detection
- `internal/types/cycles_test.go` (~200 LOC) - Unit tests

**Modified files:**
- `cmd/ailang/debug.go` (+50 LOC) - Integration and output improvements

**Total:** ~400 LOC

## Open Questions

None - design is clear from M-DX11-CYCLES design doc.

## Notes

- The traverse package already handles cycle detection during traversal; this milestone adds **detection/reporting** of cycles in type definitions
- This is the "Phase 3" work from the parent M-DX11 design doc
- Low risk because foundation exists - this is enhancement, not greenfield

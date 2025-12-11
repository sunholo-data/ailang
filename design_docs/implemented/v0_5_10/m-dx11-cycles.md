# M-DX11-CYCLES: Type Graph Cycle Detection Command

**Status**: ✅ Implemented
**Version**: v0.5.10
**Priority**: P1 (Important - DX improvement)
**Actual Time**: ~3 hours
**Dependencies**: M-PERF2 (completed), M-DX11 Phase 1 (completed in v0.5.9)
**Parent**: M-DX11 (Cyclic Type Diagnostics)

## Implementation Report

**Completed**: 2025-12-11
**Commit**: `9e5cec71`

### What Was Built

1. **Type-Graph Cycle Detection** (`internal/types/cycles.go` - 255 LOC)
   - AST-level cycle detection that handles generic recursive ADTs
   - Special handling for parser normalization of `List[a]` → `[a]`
   - Classification of cycles as "expected" (standard ADTs) or "suspicious"
   - Field path tracking for cycle visualization

2. **Comprehensive Tests** (`internal/types/cycles_test.go` - 403 LOC)
   - 10 unit tests covering various cycle patterns
   - Parser integration test with real AILANG source
   - Coverage of List, Tree, Person recursive types

3. **CLI Integration** (`cmd/ailang/debug.go` - updated)
   - Uses new `types.DetectCycles` function
   - Human-readable and JSON output formats
   - Proper classification and path display

### Key Technical Insight

The parser normalizes `List[a]` to `[a]` (ListType), so direct string matching couldn't detect List cycles. Solution: Pass the defining type name through the traversal and treat ListType as a cycle when defining "List".

### Actual Output

```bash
$ ailang debug cycles examples/complex_types.ail

Analyzing type graph for examples/complex_types.ail...

Found 3 cyclic type reference(s):

Cycle 1 [EXPECTED]: List
  Path: List → Cons() → [a]
  Depth: 2 node(s)
  Note: Standard recursive ADT pattern

Cycle 2 [EXPECTED]: Tree
  Path: Tree → Node() → Tree[a]
  Depth: 2 node(s)
  Note: Standard recursive ADT pattern

Cycle 3 [SUSPICIOUS]: Person
  Path: Person → friends → [] → Person
  Depth: 3 node(s)

Summary:
  - 1 suspicious cycle(s) (may cause hangs without cycle-safe traversal)
  - 2 expected cycle(s) (standard recursive patterns)
```

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `internal/types/cycles.go` | New | 255 |
| `internal/types/cycles_test.go` | New | 403 |
| `cmd/ailang/debug.go` | Modified | -113 (removed duplicate code) |
| **Total** | | **658 new** |

### Success Criteria Status

- [x] `ailang debug cycles examples/complex_types.ail` runs successfully
- [x] Correctly identifies cycles in generic ADTs (List[a], Tree[a])
- [x] Field paths shown clearly (not just "...")
- [x] Completes in <1 second
- [x] All tests passing (10/10)
- [x] JSON output for tooling integration

---

## Original Design

### Prerequisite Status

**M-DX11 Phase 1 (v0.5.9) - COMPLETE:**
- ✅ `ailang check --timeout` with stack dump
- ✅ `ailang check --debug-compile` with phase timing
- ✅ SafeTypeString depth-limited type stringification
- See: [design_docs/implemented/v0_5_9/m-dx11-cyclic-type-diagnostics.md](../../implemented/v0_5_9/m-dx11-cyclic-type-diagnostics.md)

### AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | CLI tool, no syntax change |
| Preserve Semantic Clarity | + | +1 | Makes type graph structure visible |
| Increase Determinism | + | +1 | Deterministic cycle detection algorithm |
| Lower Token Cost | + | +1 | Faster diagnosis = fewer debugging iterations |
| **Net Score** | | **+3** | **Decision: Move forward** |

### Problem Statement

When debugging cyclic type hangs (like M-PERF2), developers have no built-in way to identify which types form cycles. The current workflow requires:

1. Adding manual debug prints to type traversal code
2. Running compilation and observing output
3. Manually tracing through the output to find cycles
4. Removing debug prints after diagnosis

**Current State:**
- Time to diagnose cyclic hang: ~4 hours (M-PERF2 experience)
- Lines of manual debug code needed: ~50
- No built-in tools for cycle visualization

**Impact:**
- Developers and AI agents debugging type issues
- Significant time wasted on manual instrumentation

### Goals

**Primary Goal:** Provide single-command cycle detection for type graphs

**Success Metrics:**
- `ailang debug cycles <file>` identifies cyclic types in <5 seconds
- Output clearly shows cycle paths with source locations
- Works on any AILANG file (single file or module)

### Solution Design

#### Overview

New CLI command `ailang debug cycles <file>` that:
1. Parses and type-checks the file
2. Traverses all types with cycle detection
3. Reports discovered cycles with path visualization

#### Architecture

**Components:**
1. **CLI Handler**: `cmd/ailang/debug.go` - Command parsing and output
2. **Cycle Detector**: `internal/types/cycles.go` - DFS-based cycle detection
3. **Path Formatter**: Source location extraction and path visualization

### Implementation Plan

**Phase 1: Core Detection** (~2 hours) ✅
- [x] Create `internal/types/cycles.go` with DFS cycle detection
- [x] Track visited set and current path
- [x] Detect back-edges in type graph
- [x] Unit tests for cycle detection

**Phase 2: CLI Integration** (~1.5 hours) ✅
- [x] Add `debug` command group to CLI
- [x] Implement `debug cycles` subcommand
- [x] Parse file and run type checking
- [x] Call cycle detector and format output

**Phase 3: Source Location** (~0.5 hours) - Deferred
- [ ] Extract source locations from type definitions
- [ ] Include in cycle path output
- [x] Add hints for common cyclic patterns (classification instead)

### Files Modified/Created

**New files:**
- `internal/types/cycles.go` - Cycle detection algorithm (255 LOC)
- `internal/types/cycles_test.go` - Unit tests (403 LOC)

**Modified files:**
- `cmd/ailang/debug.go` - Cycles subcommand integration

### Algorithm

The implementation uses a simplified approach compared to the original Tarjan design:

```go
// Collect type references and check for self-references
func DetectCycles(decls []ast.Node, filename string) []CycleInfo {
    for _, decl := range decls {
        if td, ok := decl.(*ast.TypeDecl); ok {
            refs := collectTypeReferences(td.Definition, td.Name)
            for _, ref := range refs {
                if ref.baseName == td.Name {
                    // Found cycle - record with path info
                }
            }
        }
    }
    return cycles
}
```

### Testing Strategy

**Unit tests:** ✅
- Cycle detection on hand-crafted cyclic types
- No false positives on acyclic types
- Correct path extraction
- Parser integration test

**Integration tests:** ✅
- Run on examples/complex_types.ail (known cyclic types)

### Non-Goals

**Not in this feature:**
- Interactive cycle visualization - CLI text output only
- Automatic cycle breaking - Just detection
- IDE integration - CLI tool first
- Source line numbers - Deferred (classification provides enough context)

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance on large type graphs | Medium | Use visited set, limit depth |
| False positives | Medium | Careful distinction of identity vs structural cycles |
| Parser normalization | High | ✅ Handled with `definingType` parameter |

### References

- [M-DX11 Cyclic Type Diagnostics](../v0_5_9/m-dx11-cyclic-type-diagnostics.md) - Parent design doc
- [M-PERF2 Post-mortem](../v0_5_8/m-perf2-cyclic-type-hang-postmortem.md) - Lessons learned
- [safe_string.go](../../../internal/types/safe_string.go) - Existing depth-limited traversal

### Future Work

- Source line numbers in cycle output
- Integration with `--debug-compile` output
- Graphviz DOT output for visualization
- IDE hover integration for cycle warnings

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-11
**Implementation completed**: 2025-12-11

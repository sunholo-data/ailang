# M-DX11-CYCLES: Type Graph Cycle Detection Command

**Status**: Planned
**Target**: v0.5.10
**Priority**: P1 (Important - DX improvement)
**Estimated**: 4 hours
**Dependencies**: M-PERF2 (completed), M-DX11 Phase 1 (completed in v0.5.9)
**Parent**: M-DX11 (Cyclic Type Diagnostics)

## Prerequisite Status

**M-DX11 Phase 1 (v0.5.9) - COMPLETE:**
- ✅ `ailang check --timeout` with stack dump
- ✅ `ailang check --debug-compile` with phase timing
- ✅ SafeTypeString depth-limited type stringification
- See: [design_docs/implemented/v0_5_9/m-dx11-cyclic-type-diagnostics.md](../../implemented/v0_5_9/m-dx11-cyclic-type-diagnostics.md)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | CLI tool, no syntax change |
| Preserve Semantic Clarity | + | +1 | Makes type graph structure visible |
| Increase Determinism | + | +1 | Deterministic cycle detection algorithm |
| Lower Token Cost | + | +1 | Faster diagnosis = fewer debugging iterations |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

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

## Goals

**Primary Goal:** Provide single-command cycle detection for type graphs

**Success Metrics:**
- `ailang debug cycles <file>` identifies cyclic types in <5 seconds
- Output clearly shows cycle paths with source locations
- Works on any AILANG file (single file or module)

## Solution Design

### Overview

New CLI command `ailang debug cycles <file>` that:
1. Parses and type-checks the file
2. Traverses all types with cycle detection
3. Reports discovered cycles with path visualization

### Architecture

**Components:**
1. **CLI Handler**: `cmd/ailang/debug_cycles.go` - Command parsing and output
2. **Cycle Detector**: `internal/types/cycles.go` - DFS-based cycle detection
3. **Path Formatter**: Source location extraction and path visualization

### Example Output

```bash
ailang debug cycles sim/test_combined.ail

# Output:
# Analyzing type graph for sim/test_combined...
#
# Found 2 cyclic type references:
#
# Cycle 1: NPCState → inventory: [Item] → Item → owner: NPCState
#   Location: sim/npc_ai.ail:15 (type NPC)
#   Depth: 3 nodes
#
# Cycle 2: List[a] → element: a (where a = List[a])
#   Location: std/list.ail:5 (recursive ADT)
#   Depth: 2 nodes (self-referential)
#
# Note: Cyclic types are valid but require cycle-safe traversal.
```

### Implementation Plan

**Phase 1: Core Detection** (~2 hours)
- [ ] Create `internal/types/cycles.go` with DFS cycle detection
- [ ] Track visited set and current path
- [ ] Detect back-edges in type graph
- [ ] Unit tests for cycle detection

**Phase 2: CLI Integration** (~1.5 hours)
- [ ] Add `debug` command group to CLI
- [ ] Implement `debug cycles` subcommand
- [ ] Parse file and run type checking
- [ ] Call cycle detector and format output

**Phase 3: Source Location** (~0.5 hours)
- [ ] Extract source locations from type definitions
- [ ] Include in cycle path output
- [ ] Add hints for common cyclic patterns

### Files to Modify/Create

**New files:**
- `internal/types/cycles.go` - Cycle detection algorithm (~100 LOC)
- `internal/types/cycles_test.go` - Unit tests (~150 LOC)
- `cmd/ailang/debug.go` - Debug command group (~30 LOC)
- `cmd/ailang/debug_cycles.go` - Cycles subcommand (~80 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add debug command routing (~10 LOC)

## Algorithm

```go
// Tarjan-style DFS for cycle detection
func DetectCycles(types []Type) []Cycle {
    var cycles []Cycle
    visited := make(map[Type]bool)
    inStack := make(map[Type]bool)
    path := []Type{}

    var dfs func(t Type)
    dfs = func(t Type) {
        if inStack[t] {
            // Found cycle - extract path from t to t
            cycles = append(cycles, extractCycle(path, t))
            return
        }
        if visited[t] {
            return
        }

        visited[t] = true
        inStack[t] = true
        path = append(path, t)

        for _, child := range typeChildren(t) {
            dfs(child)
        }

        path = path[:len(path)-1]
        inStack[t] = false
    }

    for _, t := range types {
        dfs(t)
    }
    return cycles
}
```

## Success Criteria

- [ ] `ailang debug cycles sim/test_combined.ail` runs successfully
- [ ] Correctly identifies cycles in stapledons_voyage type graph
- [ ] Source locations are accurate
- [ ] Completes in <5 seconds for complex modules
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Cycle detection on hand-crafted cyclic types
- No false positives on acyclic types
- Correct path extraction

**Integration tests:**
- Run on stapledons_voyage files (known cyclic types)
- Run on stdlib (should report no unexpected cycles)

**Manual testing:**
- Verify output formatting is readable
- Test with various cycle depths

## Non-Goals

**Not in this feature:**
- Interactive cycle visualization - CLI text output only
- Automatic cycle breaking - Just detection
- IDE integration - CLI tool first

## Timeline

**Implementation** (4 hours):
- Phase 1: 2 hours
- Phase 2: 1.5 hours
- Phase 3: 0.5 hours

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance on large type graphs | Medium | Use visited set, limit depth |
| False positives | Medium | Careful distinction of identity vs structural cycles |

## References

- [M-DX11 Cyclic Type Diagnostics](m-dx11-cyclic-type-diagnostics.md) - Parent design doc
- [M-PERF2 Post-mortem](../../implemented/v0_5_8/m-perf2-cyclic-type-hang-postmortem.md) - Lessons learned
- [safe_string.go](../../../internal/types/safe_string.go) - Existing depth-limited traversal

## Future Work

- Integration with `--debug-compile` output
- Graphviz DOT output for visualization
- IDE hover integration for cycle warnings

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09

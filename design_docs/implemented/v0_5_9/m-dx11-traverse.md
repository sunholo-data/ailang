# M-DX11-TRAVERSE: Safe Type Traversal Library

**Status**: IMPLEMENTED
**Target**: v0.5.9
**Priority**: P1 (Important - Prevents future cyclic hangs)
**Estimated**: 4 hours
**Dependencies**: M-PERF2 (completed), M-DX11 Phase 1 (completed)
**Parent**: M-DX11 (Cyclic Type Diagnostics)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes manual visited set boilerplate |
| Preserve Semantic Clarity | + | +1 | Clear visitor pattern, explicit cycle handling |
| Increase Determinism | + | +1 | Deterministic traversal, no hangs |
| Lower Token Cost | + | +1 | Less code to write for type operations |
| **Net Score** | | **+4** | **Decision: Move forward** |

## Problem Statement

Each type traversal function currently needs manual protection against cycles:

```go
// Current pattern - repeated everywhere
func collectFreeVars(t Type, vars map[string]bool, visited map[Type]bool) {
    if visited[t] { return }
    visited[t] = true
    // ... actual logic
}
```

This leads to:
1. **Inconsistency**: Some functions protected, some not
2. **Boilerplate**: Same visited-set pattern everywhere
3. **Bug risk**: Easy to forget protection in new code
4. **M-PERF2 bug**: SafeEquals was missing protection

**Current State:**
- 15+ type traversal functions across codebase
- Inconsistent cycle protection
- ~20 lines of boilerplate per protected function

**Impact:**
- Any new type traversal risks hangs
- Code duplication
- Maintenance burden

## Goals

**Primary Goal:** Provide safe-by-default type traversal with automatic cycle detection

**Success Metrics:**
- All type traversals use safe library
- Zero new cyclic hang bugs possible
- 60% less boilerplate for type operations
- Clear API for custom traversals

## Solution Design

### Overview

New `internal/types/traverse` package providing:
1. Safe visitor pattern with automatic cycle detection
2. Pre-built safe wrappers for common operations
3. Configurable cycle handling (error, skip, callback)

### Architecture

**Components:**
1. **TypeVisitor**: Core visitor with automatic cycle detection
2. **Safe wrappers**: `CollectFreeVars`, `ContainsType`, `MapTypes`, etc.
3. **Cycle handlers**: Configurable behavior on cycle detection

### API Design

```go
package traverse

// TypeVisitor with automatic cycle detection
type TypeVisitor struct {
    visited  map[types.Type]bool
    depth    int
    maxDepth int
    OnCycle  func(typ types.Type, path []types.Type) // Called when cycle detected
}

func NewVisitor() *TypeVisitor {
    return &TypeVisitor{
        visited:  make(map[types.Type]bool),
        maxDepth: 1000,
    }
}

func (v *TypeVisitor) Visit(t types.Type, fn func(types.Type)) {
    if v.visited[t] {
        if v.OnCycle != nil {
            v.OnCycle(t, nil)
        }
        return
    }
    if v.depth > v.maxDepth {
        panic(fmt.Sprintf("type traversal exceeded depth %d on %T", v.maxDepth, t))
    }

    v.visited[t] = true
    v.depth++
    defer func() { v.depth--; delete(v.visited, t) }()

    fn(t)

    // Recursively visit children
    for _, child := range v.children(t) {
        v.Visit(child, fn)
    }
}

// Safe wrappers for common operations
func CollectFreeVars(t types.Type) map[string]bool {
    vars := make(map[string]bool)
    NewVisitor().Visit(t, func(typ types.Type) {
        if tv, ok := typ.(*types.TVar); ok {
            vars[tv.Name] = true
        }
    })
    return vars
}

func ContainsType(t, target types.Type) bool {
    found := false
    NewVisitor().Visit(t, func(typ types.Type) {
        if typ == target {
            found = true
        }
    })
    return found
}

func MapTypes(t types.Type, fn func(types.Type) types.Type) types.Type {
    // Transform types with cycle safety
}
```

### Usage Examples

**Before (unsafe):**
```go
// Manual cycle protection - easy to forget
func collectFreeVars(t Type, vars map[string]bool, visited map[Type]bool) {
    if visited[t] { return }
    visited[t] = true
    switch typ := t.(type) {
    case *TVar:
        vars[typ.Name] = true
    case *TFunc:
        collectFreeVars(typ.Return, vars, visited)
        for _, p := range typ.Params {
            collectFreeVars(p, vars, visited)
        }
    // ... 20 more cases
    }
}
```

**After (safe):**
```go
// Automatic cycle protection
vars := traverse.CollectFreeVars(t)
```

### Implementation Plan

**Phase 1: Core Library** (~2 hours)
- [ ] Create `internal/types/traverse/traverse.go`
- [ ] Implement TypeVisitor with cycle detection
- [ ] Implement children() for all Type variants
- [ ] Unit tests for visitor

**Phase 2: Safe Wrappers** (~1.5 hours)
- [ ] `CollectFreeVars` - gather type variables
- [ ] `ContainsType` - check for type presence
- [ ] `MapTypes` - transform types
- [ ] `FoldTypes` - reduce over types

**Phase 3: Migration** (~0.5 hours)
- [ ] Update `collectFreeVars` in typechecker to use library
- [ ] Update `occurs` check to use library
- [ ] Document migration guide

### Files to Modify/Create

**New files:**
- `internal/types/traverse/traverse.go` - Core visitor (~150 LOC)
- `internal/types/traverse/wrappers.go` - Safe wrappers (~100 LOC)
- `internal/types/traverse/traverse_test.go` - Tests (~200 LOC)

**Modified files:**
- `internal/types/typechecker_defaulting.go` - Use traverse.CollectFreeVars
- `internal/types/unifier.go` - Use traverse for occurs check

## Success Criteria

- [ ] All type traversal uses safe library
- [ ] `traverse.CollectFreeVars` works on cyclic types
- [ ] Depth limit prevents hangs on pathological cases
- [ ] OnCycle callback fires for detected cycles
- [ ] All tests passing
- [ ] Documentation with examples

## Testing Strategy

**Unit tests:**
- Visitor visits all type variants
- Cycle detection triggers callback
- Depth limit triggers panic
- Safe wrappers return correct results

**Integration tests:**
- Migrate existing functions, verify same behavior
- Test with stapledons_voyage types (known cycles)

**Property tests:**
- CollectFreeVars on any type terminates
- ContainsType agrees with manual search

## Non-Goals

**Not in this feature:**
- Parallel traversal - Sequential visitor only
- Memoization - Fresh visited set per traversal
- Type mutation - Read-only traversal

## Timeline

**Implementation** (4 hours):
- Phase 1: 2 hours
- Phase 2: 1.5 hours
- Phase 3: 0.5 hours

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance overhead | Medium | Benchmark vs manual; use pool for visited maps |
| Breaking existing behavior | High | Migration with tests verifying identical results |
| Incomplete type coverage | Medium | Enumerate all Type variants in children() |

## API Design Rule

**Established by M-PERF2 post-mortem:**

> Every function of shape `func(Type) T` MUST document cycle-safety. Either use `traverse.Visit` or add a `visited` parameter.

This library makes the safe choice the easy choice.

## References

- [M-DX11 Cyclic Type Diagnostics](m-dx11-cyclic-type-diagnostics.md) - Parent design doc
- [M-PERF2 Post-mortem](../../implemented/v0_5_8/m-perf2-cyclic-type-hang-postmortem.md) - API design rule
- [safe_string.go](../../../internal/types/safe_string.go) - Example of depth-limited traversal

## Future Work

- Pool visited maps for performance
- Add traversal statistics/profiling
- Parallel visitor for large type graphs
- Integration with cycle detection command

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09

# M-DX11: Cyclic Type Diagnostics & Hang Prevention

**Status**: Planned
**Target**: v0.5.9
**Priority**: P1 (Important - DX improvement based on M-PERF2 learnings)
**Estimated**: 8-12 hours
**Dependencies**: M-PERF2 (completed)
**Motivation**: M-PERF2 post-mortem revealed diagnostic gaps

## Problem Statement

During the M-PERF2 investigation, we spent ~4 hours debugging a cyclic type hang. Key diagnostic gaps:

1. **Silent hangs with no feedback** - Process just spins with no indication of where
2. **No built-in cycle detection in type traversal** - Each function needs manual protection
3. **No timeout/watchdog mechanism** - Hangs require manual process killing
4. **Debug output required manual instrumentation** - Added/removed ad-hoc print statements
5. **No type graph visualization** - Couldn't see the cyclic structure

## Goals

1. **Never hang silently** - Always provide feedback or timeout
2. **Built-in cycle safety** - Type traversal should be safe by default
3. **Easy diagnosis** - Single command to identify cyclic types
4. **Prevent regressions** - CI should catch new cycle vulnerabilities

## Proposed Solutions

### 1. Compiler Watchdog Timer (`--timeout`)

```bash
# Automatic timeout with stack dump
ailang check --timeout 30s sim/test_combined.ail

# Output on timeout:
# ERROR: Compilation timed out after 30s
# Last phase: Type Checking (decl 4/4 for sim/test_combined)
#
# Stack dump:
#   goroutine 1 [running]:
#   github.com/sunholo/ailang/internal/types.collectFreeVars(...)
#       /internal/types/typechecker_defaulting.go:291
#   ...
#
# Hint: This may indicate cyclic types. Try: ailang debug cycles <file>
```

**Implementation:**
```go
// In pipeline/pipeline.go
func runWithTimeout(cfg Config, timeout time.Duration, fn func() error) error {
    done := make(chan error, 1)
    go func() { done <- fn() }()

    select {
    case err := <-done:
        return err
    case <-time.After(timeout):
        // Dump goroutine stacks
        buf := make([]byte, 1<<20)
        n := runtime.Stack(buf, true)
        fmt.Fprintf(os.Stderr, "TIMEOUT after %s\nStack:\n%s\n", timeout, buf[:n])
        return fmt.Errorf("compilation timed out after %s", timeout)
    }
}
```

### 2. Type Graph Cycle Detection Command

```bash
# New command to detect cyclic types
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

**Implementation:**
```go
// New command: cmd/ailang/debug_cycles.go
func debugCycles(filename string) error {
    // Parse and type-check
    result, err := pipeline.Run(...)

    // Traverse all types with cycle detection
    visited := make(map[types.Type]bool)
    path := []types.Type{}

    for _, typ := range result.AllTypes {
        if cycle := detectCycle(typ, visited, path); cycle != nil {
            reportCycle(cycle)
        }
    }
}
```

### 3. Safe Type Traversal Library

Create a `types/traverse` package with built-in cycle protection:

```go
// types/traverse/traverse.go
package traverse

// Visitor with automatic cycle detection
type TypeVisitor struct {
    visited map[types.Type]bool
    depth   int
    maxDepth int
    OnCycle func(typ types.Type, path []types.Type) // Called when cycle detected
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
    switch typ := t.(type) {
    case *types.TApp:
        v.Visit(typ.Constructor, fn)
        for _, arg := range typ.Args {
            v.Visit(arg, fn)
        }
    // ... other cases
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
```

**Usage:**
```go
// Old (unsafe):
collectFreeVars(t, vars)

// New (safe):
vars := traverse.CollectFreeVars(t)
```

### 4. Phase Timing in `--debug-compile`

Enhance debug output with timing breakdown:

```bash
ailang check --debug-compile sim/test_combined.ail

# Output:
# Compilation Phases:
#   Loading:       45ms
#   Parsing:       12ms
#   Elaboration:   89ms
#   Type Checking: 234ms  ← Slowest
#     - sim/protocol:      23ms
#     - sim/npc_ai:        156ms  ← Warning: >100ms
#     - sim/test_combined: 55ms
#   Effect Check:  18ms
#   Mono:          67ms
#   Lowering:      34ms
#   Interface:     8ms
#
# Total: 507ms
#
# Warnings:
#   sim/npc_ai type checking took 156ms (threshold: 100ms)
#   Consider checking for complex recursive types.
```

### 5. CI Cycle Regression Test

Add to CI pipeline:

```yaml
# .github/workflows/ci.yml
- name: Check for cyclic type vulnerabilities
  run: |
    # Run with timeout to catch hangs
    timeout 60s make test || {
      echo "Tests timed out - possible cyclic type hang"
      exit 1
    }

    # Run cycle detection on complex test files
    ailang debug cycles examples/complex_types.ail
```

### 6. Built-in Depth Limits with Good Errors

Add depth limits to ALL type traversal with descriptive errors:

```go
const (
    MaxTraversalDepth = 1000
    MaxStringifyDepth = 100
)

func (t *TApp) String() string {
    return stringWithDepth(t, 0)
}

func stringWithDepth(t Type, depth int) string {
    if depth > MaxStringifyDepth {
        return fmt.Sprintf("<%T...depth limit>", t)  // Truncate, don't hang
    }
    // ... normal implementation with depth+1
}
```

## Implementation Plan

### Phase 1: Quick Wins (2 hours)
- [ ] Add `--timeout` flag to `ailang check`
- [ ] Add phase timing to `--debug-compile`
- [ ] Add depth limits to `Type.String()` methods

### Phase 2: Cycle Detection Command (4 hours)
- [ ] Implement `ailang debug cycles <file>`
- [ ] Add cycle visualization output
- [ ] Add hints for common cyclic patterns

### Phase 3: Safe Traversal Library (4 hours)
- [ ] Create `types/traverse` package
- [ ] Migrate `collectFreeVars` to use it
- [ ] Migrate `occurs` check to use it
- [ ] Add documentation

### Phase 4: CI Integration (2 hours)
- [ ] Add timeout to CI test runs
- [ ] Add cycle detection to CI
- [ ] Create regression test with known cyclic types

## Success Criteria

- [ ] `ailang check --timeout 5s` on hanging file reports useful error within 5s
- [ ] `ailang debug cycles` identifies cyclic types in stapledons_voyage
- [ ] All type traversal uses safe library
- [ ] CI catches cyclic type hangs before merge
- [ ] No more silent hangs - always feedback within timeout

## Files to Create/Modify

**New files:**
- `internal/types/traverse/traverse.go` - Safe traversal library
- `internal/types/traverse/traverse_test.go` - Tests
- `cmd/ailang/debug_cycles.go` - Cycle detection command

**Modified files:**
- `cmd/ailang/check.go` - Add `--timeout` flag
- `internal/pipeline/pipeline.go` - Add watchdog timer
- `internal/types/types.go` - Add depth limits to String()
- `.github/workflows/ci.yml` - Add timeout and cycle detection

## Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Time to diagnose cyclic hang | ~4 hours | <5 minutes |
| Lines of manual debug code needed | ~50 | 0 |
| Silent hang possibility | Yes | No |

---

**Document created**: 2025-12-09
**Motivation**: M-PERF2 post-mortem lessons learned

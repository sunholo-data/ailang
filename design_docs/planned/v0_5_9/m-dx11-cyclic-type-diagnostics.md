# M-DX11: Cyclic Type Diagnostics & Hang Prevention

**Status**: Planned
**Target**: v0.5.9
**Priority**: P1 (Important - DX improvement based on M-PERF2 learnings)
**Estimated**: 8-12 hours
**Dependencies**: M-PERF2 (completed)
**Motivation**: M-PERF2 post-mortem revealed diagnostic gaps
**Last Updated**: 2025-12-09 (incorporated design review feedback)

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

## Design Review Summary (2025-12-09)

Key refinements from design review:

| Area | Original | Refined |
|------|----------|---------|
| Timeout | Naked goroutine + channel | `context.Context` for composability |
| Timeout default | Unclear | **Off by default** (opt-in via `--timeout`) |
| Traversal modes | Single visitor | **Two modes**: Path (cycle detection) vs Global (dedup) |
| Occurs check | Generic visitor | **Parameterized by variable name** |
| debug cycles | Separate implementation | **Reuse traverse library** |
| Output format | Text only | Add `--json` for tooling/IDE integration |
| Metrics | Time only | Add **counts** (declarations, types, constraints) |

## Proposed Solutions

### 1. Compiler Watchdog Timer (`--timeout`)

**Key Design Decisions:**
- **Use `context.Context`** for composability with library mode
- **Off by default** - avoid false positives on legitimately slow compiles
- **`--debug-compile` enables generous default** (60s) with status message
- **Error message is diagnostic**, not semantic - explicitly says "internal compiler timeout"

```bash
# CLI mode: explicit timeout
ailang check --timeout 30s sim/test_combined.ail

# Debug mode: auto-arms watchdog with generous timeout
ailang check --debug-compile sim/test_combined.ail
# Output includes: "(timeout watchdog armed: 60s)"

# Library mode: caller controls context
result, err := pipeline.Compile(ctx, cfg)  // ctx can have timeout
```

**Output on timeout:**
```
INTERNAL COMPILER TIMEOUT after 30s

This is a diagnostic timeout, not a semantic error.
Likely cause: non-terminating type traversal or extreme complexity.

Last phase: Type Checking (decl 4/4 for sim/test_combined)

Stack dump:
  goroutine 1 [running]:
  github.com/sunholo/ailang/internal/types.collectFreeVars(...)
      /internal/types/typechecker_defaulting.go:291
  ...

Hint: This may indicate cyclic types. Try: ailang debug cycles <file>

Please file a bug with this stack trace if unexpected.
```

**Implementation:**
```go
// In pipeline/pipeline.go

// Compile runs the full compilation pipeline with context support.
// The context can be used for cancellation and timeouts.
func Compile(ctx context.Context, cfg Config) (*Result, error) {
    // Check for cancellation at phase boundaries
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // ... run phases, checking ctx.Done() between each
}

// runWithTimeout wraps compilation with a watchdog timer.
// Used by CLI when --timeout is specified.
func runWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    done := make(chan error, 1)
    go func() { done <- fn(ctx) }()

    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        // Dump goroutine stacks for diagnosis
        buf := make([]byte, 1<<20)
        n := runtime.Stack(buf, true)
        fmt.Fprintf(os.Stderr, "INTERNAL COMPILER TIMEOUT after %s\n\n", timeout)
        fmt.Fprintf(os.Stderr, "This is a diagnostic timeout, not a semantic error.\n")
        fmt.Fprintf(os.Stderr, "Likely cause: non-terminating type traversal or extreme complexity.\n\n")
        fmt.Fprintf(os.Stderr, "Stack:\n%s\n", buf[:n])
        fmt.Fprintf(os.Stderr, "\nHint: Try `ailang debug cycles <file>` to detect cyclic types.\n")
        fmt.Fprintf(os.Stderr, "Please file a bug with this stack trace if unexpected.\n")
        return fmt.Errorf("internal compiler timeout after %s", timeout)
    }
}
```

### 2. Type Graph Cycle Detection Command

**Key Design Decisions:**
- **Reuse `types/traverse` library** - don't write a second ad-hoc DFS
- **Add `--json` flag** for machine-readable output (IDE/tooling integration)
- **Include source location mapping** - type → defining ADT/alias + file:line
- **Classify cycles** - distinguish expected (stdlib μ-types) from suspicious

```bash
# Human-readable output
ailang debug cycles sim/test_combined.ail

# Machine-readable output for tooling
ailang debug cycles --json sim/test_combined.ail
```

**Human Output:**
```
Analyzing type graph for sim/test_combined...

Found 2 cyclic type references:

Cycle 1 [SUSPICIOUS]: NPCState → inventory: [Item] → Item → owner: NPCState
  Location: sim/npc_ai.ail:15 (type NPC)
  Defined in: TypeDecl NPC
  Depth: 3 nodes

Cycle 2 [EXPECTED]: List[a] → element: a (where a = List[a])
  Location: std/list.ail:5 (recursive ADT)
  Defined in: TypeDecl List (stdlib)
  Depth: 2 nodes (self-referential)
  Note: Standard recursive ADT, safe if using cycle-aware traversal.

Summary:
  - 1 suspicious cycle (may cause hangs without cycle-safe traversal)
  - 1 expected cycle (stdlib recursive type)
```

**JSON Output (`--json`):**
```json
{
  "file": "sim/test_combined.ail",
  "cycles": [
    {
      "kind": "suspicious",
      "path": ["NPCState", "inventory: [Item]", "Item", "owner: NPCState"],
      "location": {"file": "sim/npc_ai.ail", "line": 15, "column": 1},
      "defined_in": {"kind": "TypeDecl", "name": "NPC"},
      "depth": 3
    },
    {
      "kind": "expected",
      "path": ["List[a]", "element: a"],
      "location": {"file": "std/list.ail", "line": 5, "column": 1},
      "defined_in": {"kind": "TypeDecl", "name": "List", "is_stdlib": true},
      "depth": 2,
      "note": "Standard recursive ADT"
    }
  ],
  "summary": {
    "suspicious": 1,
    "expected": 1,
    "total": 2
  }
}
```

**Implementation:**
```go
// New command: cmd/ailang/debug_cycles.go

// CycleKind classifies whether a cycle is expected or suspicious
type CycleKind string

const (
    CycleExpected   CycleKind = "expected"   // stdlib μ-types, known recursive ADTs
    CycleSuspicious CycleKind = "suspicious" // user-defined, may cause traversal issues
)

// CycleInfo holds information about a detected cycle
type CycleInfo struct {
    Kind       CycleKind
    Path       []string           // Type names in cycle
    Location   SourceLocation     // Where the type is defined
    DefinedIn  TypeDeclInfo       // The TypeDecl that defines this type
    Depth      int
    Note       string             // Optional explanation
}

func debugCycles(filename string, outputJSON bool) error {
    // Parse and type-check
    result, err := pipeline.Run(...)
    if err != nil {
        return err
    }

    // Build type → source location mapping
    typeOrigins := buildTypeOriginMap(result)

    // Use traverse library with path-based cycle detection
    var cycles []CycleInfo
    visitor := traverse.NewVisitor(traverse.ModePath)
    visitor.OnCycle = func(typ types.Type, path []types.Type) {
        cycle := CycleInfo{
            Kind:      classifyCycle(typ, typeOrigins),
            Path:      formatPath(path),
            Location:  typeOrigins[typ],
            DefinedIn: findDefiningDecl(typ, result),
            Depth:     len(path),
        }
        if cycle.Kind == CycleExpected {
            cycle.Note = "Standard recursive ADT"
        }
        cycles = append(cycles, cycle)
    }

    for _, typ := range result.AllTypes {
        visitor.Reset() // Clear visited set for each root
        visitor.Visit(typ, func(t types.Type) {})
    }

    if outputJSON {
        return outputCyclesJSON(cycles)
    }
    return outputCyclesHuman(cycles)
}

// classifyCycle determines if a cycle is expected or suspicious
func classifyCycle(typ types.Type, origins map[types.Type]SourceLocation) CycleKind {
    loc := origins[typ]
    // Stdlib types are expected to be recursive
    if strings.HasPrefix(loc.File, "std/") {
        return CycleExpected
    }
    // Known recursive patterns (List, Tree, etc.) can be annotated
    // ... additional classification logic
    return CycleSuspicious
}
```

### 3. Safe Type Traversal Library

**Key Design Decisions:**
- **Two traversal modes** with different semantics:
  - `ModePath`: Detect back-edges on current path (for cycle detection)
  - `ModeGlobal`: Each node processed once (for collecting/transforming)
- **Parameterized occurs check** - never share visitor state across variable names
- **Mandatory usage** - grep for hand-written recursion and migrate or annotate

Create a `types/traverse` package with built-in cycle protection:

```go
// types/traverse/traverse.go
package traverse

// Mode determines how the visitor tracks "seen" nodes
type Mode int

const (
    // ModePath detects back-edges on the current traversal path.
    // Nodes are unmarked on stack unwind.
    // Use for: cycle detection, occurs check
    ModePath Mode = iota

    // ModeGlobal processes each node exactly once across all visits.
    // Nodes stay marked after first visit.
    // Use for: collectFreeVars, type transformations
    ModeGlobal
)

// TypeVisitor traverses type graphs with automatic cycle protection
type TypeVisitor struct {
    mode     Mode
    visited  map[types.Type]bool
    path     []types.Type        // Only used in ModePath
    depth    int
    maxDepth int
    OnCycle  func(typ types.Type, path []types.Type) // Called when cycle detected
}

func NewVisitor(mode Mode) *TypeVisitor {
    return &TypeVisitor{
        mode:     mode,
        visited:  make(map[types.Type]bool),
        path:     make([]types.Type, 0, 32),
        maxDepth: 1000,
    }
}

// Reset clears the visited set (useful when visiting multiple roots)
func (v *TypeVisitor) Reset() {
    v.visited = make(map[types.Type]bool)
    v.path = v.path[:0]
    v.depth = 0
}

func (v *TypeVisitor) Visit(t types.Type, fn func(types.Type)) {
    if v.visited[t] {
        if v.OnCycle != nil && v.mode == ModePath {
            v.OnCycle(t, v.path)
        }
        return // Already processed (ModeGlobal) or cycle (ModePath)
    }
    if v.depth > v.maxDepth {
        panic(fmt.Sprintf("type traversal exceeded depth %d on %T", v.maxDepth, t))
    }

    v.visited[t] = true
    v.depth++

    if v.mode == ModePath {
        v.path = append(v.path, t)
        defer func() {
            v.depth--
            v.path = v.path[:len(v.path)-1]
            delete(v.visited, t) // Unmark on unwind for path-based detection
        }()
    } else {
        defer func() { v.depth-- }()
        // ModeGlobal: keep visited[t] = true permanently
    }

    fn(t)

    // Recursively visit children
    switch typ := t.(type) {
    case *types.TApp:
        v.Visit(typ.Constructor, fn)
        for _, arg := range typ.Args {
            v.Visit(arg, fn)
        }
    case *types.TFunc:
        for _, param := range typ.Params {
            v.Visit(param, fn)
        }
        v.Visit(typ.Result, fn)
    case *types.TRecord:
        for _, field := range typ.Fields {
            v.Visit(field.Type, fn)
        }
    // ... other cases (TTuple, TVariant, etc.)
    }
}

// ========== Safe Wrappers for Common Operations ==========

// CollectFreeVars returns all free type variables in a type.
// Uses ModeGlobal to avoid revisiting the same node.
func CollectFreeVars(t types.Type) map[string]bool {
    vars := make(map[string]bool)
    NewVisitor(ModeGlobal).Visit(t, func(typ types.Type) {
        if tv, ok := typ.(*types.TVar); ok {
            vars[tv.Name] = true
        }
    })
    return vars
}

// Occurs checks if a type variable appears in a type.
// Uses ModePath because we care about path back to the starting variable.
// IMPORTANT: Never reuse visitor state across different variable checks!
func Occurs(varName string, t types.Type) bool {
    v := NewVisitor(ModePath)
    found := false
    v.Visit(t, func(typ types.Type) {
        if tv, ok := typ.(*types.TVar); ok && tv.Name == varName {
            found = true
        }
    })
    return found
}

// Transform applies a function to each type node, building a new type.
// Uses ModeGlobal for efficiency (each node transformed once).
func Transform(t types.Type, fn func(types.Type) types.Type) types.Type {
    cache := make(map[types.Type]types.Type)
    var transform func(types.Type) types.Type
    transform = func(t types.Type) types.Type {
        if cached, ok := cache[t]; ok {
            return cached
        }
        result := fn(t)
        cache[t] = result
        return result
    }
    return transform(t)
}
```

**Mode Selection Guide:**

| Operation | Mode | Why |
|-----------|------|-----|
| `collectFreeVars` | `ModeGlobal` | Don't re-visit same node twice |
| `occurs` check | `ModePath` | Care about path back to starting TVar |
| Cycle detection | `ModePath` | Detect back-edges on current path |
| Type substitution | `ModeGlobal` | Each node substituted once |
| Type transformation | `ModeGlobal` | Each node transformed once |

**Migration Checklist:**

After implementing traverse, grep for hand-written recursion and migrate:

```bash
# Find potential candidates for migration
grep -r "func.*Type.*Type" internal/types/ | grep -v "_test.go"
grep -r "switch.*\.(type)" internal/types/ | grep -v traverse
```

For each found function, either:
1. **Migrate** to use traverse (preferred)
2. **Annotate** with explicit comment:
   ```go
   // CYCLE-SAFE: Types here guaranteed acyclic by invariant [explain why]
   ```

**Usage Examples:**
```go
// Old (unsafe - can hang on cyclic types):
func collectFreeVars(t Type, vars map[string]bool) {
    switch typ := t.(type) {
    case *TVar:
        vars[typ.Name] = true
    case *TApp:
        collectFreeVars(typ.Constructor, vars)
        for _, arg := range typ.Args {
            collectFreeVars(arg, vars)  // No cycle protection!
        }
    }
}

// New (safe):
vars := traverse.CollectFreeVars(t)

// Old (unsafe - occurs check can diverge):
func occurs(name string, t Type) bool {
    switch typ := t.(type) {
    case *TVar:
        return typ.Name == name
    case *TApp:
        if occurs(name, typ.Constructor) { return true }
        for _, arg := range typ.Args {
            if occurs(name, arg) { return true }  // No cycle protection!
        }
    }
    return false
}

// New (safe):
found := traverse.Occurs(varName, t)
```

### 4. Phase Timing in `--debug-compile`

**Key Design Decisions:**
- **Record counts, not just timing** - detect non-linear behavior
- **Use metrics helper** - avoid scattered `time.Since()` calls
- **Near-zero cost when disabled** - can be compiled out

Enhance debug output with timing breakdown and counts:

```bash
ailang check --debug-compile sim/test_combined.ail

# Output:
# Compilation Phases:
#   Loading:       45ms    (3 files)
#   Parsing:       12ms    (847 tokens, 234 nodes)
#   Elaboration:   89ms    (234 → 312 Core nodes)
#   Type Checking: 234ms   (312 nodes, 1247 constraints)  ← Slowest
#     - sim/protocol:      23ms   (89 constraints)
#     - sim/npc_ai:        156ms  (892 constraints)  ← Warning: >100ms
#     - sim/test_combined: 55ms   (266 constraints)
#   Effect Check:  18ms    (45 effect annotations)
#   Mono:          67ms    (23 specializations)
#   Lowering:      34ms    (312 → 445 IR nodes)
#   Interface:     8ms     (12 exports)
#
# Total: 507ms
#
# Complexity Analysis:
#   Type checking: 5.3 constraints/node (typical: 2-4)
#   sim/npc_ai: 10.0 constraints/node ← Suspicious
#
# Warnings:
#   sim/npc_ai type checking took 156ms (threshold: 100ms)
#   sim/npc_ai has 10.0 constraints/node (threshold: 6.0)
#   Consider checking for complex recursive types.
#
# (timeout watchdog armed: 60s)
```

**Implementation:**
```go
// internal/pipeline/metrics.go

// Metrics collects compilation statistics for debugging.
// When disabled (default), all methods are no-ops.
type Metrics struct {
    enabled  bool
    phases   []PhaseMetrics
    current  *PhaseMetrics
}

type PhaseMetrics struct {
    Name        string
    Module      string        // Optional, for per-module breakdown
    Duration    time.Duration
    Counts      map[string]int // Flexible count storage
}

// StartPhase begins timing a compilation phase.
// No-op if metrics disabled.
func (m *Metrics) StartPhase(name string, module string) {
    if !m.enabled {
        return
    }
    m.current = &PhaseMetrics{
        Name:   name,
        Module: module,
        Counts: make(map[string]int),
    }
    m.current.start = time.Now()
}

// EndPhase completes the current phase and records metrics.
func (m *Metrics) EndPhase() {
    if !m.enabled || m.current == nil {
        return
    }
    m.current.Duration = time.Since(m.current.start)
    m.phases = append(m.phases, *m.current)
    m.current = nil
}

// Count increments a named counter for the current phase.
func (m *Metrics) Count(name string, delta int) {
    if !m.enabled || m.current == nil {
        return
    }
    m.current.Counts[name] += delta
}

// Report generates the debug output.
func (m *Metrics) Report() string {
    if !m.enabled {
        return ""
    }
    // ... format output with timing and counts
    // ... detect non-linear behavior (high constraints/node ratio)
    // ... add warnings for slow phases
}

// Usage in pipeline:
func (p *Pipeline) typeCheck(ctx context.Context, m *Metrics) error {
    for _, mod := range p.modules {
        m.StartPhase("typecheck", mod.Name)

        tc := types.NewTypeChecker(mod.Core)
        result, err := tc.Check()

        m.Count("nodes", len(mod.Core.Nodes))
        m.Count("constraints", tc.ConstraintCount())

        m.EndPhase()

        if err != nil {
            return err
        }
    }
    return nil
}
```

### 5. CI Cycle Regression Test

**Key Design Decisions:**
- **Use built-in `--timeout`** instead of POSIX `timeout` wrapper
- **Use `go test -timeout`** for test suite (already built-in)
- **Add cycle detection** as a dedicated CI step

Add to CI pipeline:

```yaml
# .github/workflows/ci.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run tests with timeout
        run: |
          # go test has built-in timeout support
          go test ./... -timeout 60s

      - name: Check compilation with timeout
        run: |
          # Use our built-in timeout flag instead of POSIX timeout
          ailang check --timeout 60s examples/

      - name: Detect cyclic types in examples
        run: |
          # Run cycle detection on complex test files
          ailang debug cycles examples/complex_types.ail --json > cycles.json

          # Fail if any suspicious cycles found
          jq -e '.summary.suspicious == 0' cycles.json || {
            echo "❌ Suspicious cyclic types detected!"
            jq '.cycles[] | select(.kind == "suspicious")' cycles.json
            exit 1
          }

      - name: Upload cycle analysis
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: cycle-analysis
          path: cycles.json
```

**Test file with known cyclic types (for regression testing):**
```ailang
-- examples/complex_types.ail
-- Contains intentional cyclic types for testing cycle detection

module examples/complex_types

-- Expected cycle: standard recursive ADT (should be marked "expected")
type List[a] =
  | Nil
  | Cons(a, List[a])

-- Suspicious cycle: mutual recursion through records
type Person = { name: string, friends: [Person] }

-- This file should produce:
-- - 1 expected cycle (List)
-- - 1 suspicious cycle (Person.friends)
```

### 6. Built-in Depth Limits with Good Errors

**Key Design Decisions:**
- **`Type.String()` is diagnostic only** - may truncate, not a semantic representation
- **Add explicit API contract** - document that no algorithmic logic may depend on String()
- **Add truncation tests** - verify we actually truncate on deep types

Add depth limits to ALL type traversal with descriptive errors:

```go
const (
    MaxTraversalDepth = 1000
    MaxStringifyDepth = 100
)

// String returns a human-readable representation of the type.
//
// IMPORTANT: This is for DIAGNOSTICS ONLY. It may truncate on complex
// or cyclic types. No algorithmic logic may depend on Type.String();
// use structural comparison or the traverse package instead.
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

**Required Tests:**
```go
// internal/types/types_test.go

func TestStringTruncatesOnDeepTypes(t *testing.T) {
    // Build a deeply nested type: Foo[Foo[Foo[...]]]
    deep := &TConst{Name: "int"}
    for i := 0; i < MaxStringifyDepth + 50; i++ {
        deep = &TApp{
            Constructor: &TConst{Name: "Foo"},
            Args:        []Type{deep},
        }
    }

    result := deep.String()

    // Should contain truncation marker, not hang
    if !strings.Contains(result, "...depth limit") {
        t.Errorf("expected truncation marker, got: %s", result)
    }

    // Should complete quickly (not hang)
    // (test timeout will catch hangs)
}

func TestStringTruncatesOnCyclicTypes(t *testing.T) {
    // Create a cyclic type: μX. Foo[X]
    var cyclic *TApp
    cyclic = &TApp{
        Constructor: &TConst{Name: "Foo"},
        Args:        []Type{nil}, // Will be set to cyclic
    }
    cyclic.Args[0] = cyclic // Create cycle

    result := cyclic.String()

    // Should truncate, not hang forever
    if !strings.Contains(result, "...depth limit") {
        t.Errorf("expected truncation on cycle, got: %s", result)
    }
}
```

**API Contract (add to types.go header):**
```go
// Package types implements AILANG's type system.
//
// # String() Contract
//
// Type.String() is for diagnostic output ONLY. It may truncate on:
// - Deeply nested types (>100 levels)
// - Cyclic types (mu-types, recursive ADTs)
//
// No algorithmic logic may depend on String() output.
// For structural operations, use:
// - types.Equal(a, b) for equality
// - traverse.CollectFreeVars(t) for variable collection
// - traverse.Occurs(name, t) for occurs check
```

## Implementation Plan

**Priority Rationale (from design review):**
1. **Quick Wins first** - directly prevents "4 hours of blind debugging" scenario
2. **Safe traversal second** - reduces probability of new hangs (structural fix)
3. **debug cycles third** - great for debugging, but less structurally critical
4. **CI last** - trivial once other pieces exist

### Phase 1: Quick Wins (2 hours) ✅ COMPLETE

These directly prevent the M-PERF2 debugging experience.

- [x] Add `--timeout` flag to `ailang check` (with stack dump)
- [x] Add phase timing + counts to `--debug-compile`
- [x] Add depth limits to `Type.String()` methods (SafeTypeString in safe_string.go)
- [ ] Add truncation tests for String()
- [ ] Document String() as diagnostic-only in types.go header

**Definition of Done:**
- `ailang check --timeout 5s` on infinite loop reports useful error
- `ailang check --debug-compile` shows timing + constraint counts

### Phase 2: Safe Traversal Library (4 hours) ⭐ STRUCTURAL FIX

This is the most important structural piece - prevents future hangs.

- [ ] Create `internal/types/traverse/traverse.go`
- [ ] Implement two modes: `ModePath` and `ModeGlobal`
- [ ] Implement safe wrappers: `CollectFreeVars`, `Occurs`, `Transform`
- [ ] Migrate existing `collectFreeVars` to use traverse
- [ ] Migrate `occurs` check to use traverse
- [ ] Grep for other hand-written type recursion and migrate/annotate
- [ ] Add comprehensive tests
- [ ] Document mode selection guide

**Definition of Done:**
- No hand-written type recursion without cycle protection
- All existing tests pass with new implementation
- traverse package has 90%+ test coverage

### Phase 3: Cycle Detection Command (4 hours)

Great for debugging, builds on Phase 2.

- [ ] Implement `ailang debug cycles <file>`
- [ ] Reuse `traverse.ModePath` for cycle detection
- [ ] Build type → source location mapping
- [ ] Implement cycle classification (expected vs suspicious)
- [ ] Add `--json` output format
- [ ] Add hints for common cyclic patterns
- [ ] Create `examples/complex_types.ail` test file

**Definition of Done:**
- `ailang debug cycles examples/complex_types.ail` produces expected output
- JSON output validates against schema

### Phase 4: CI Integration (2 hours)

Trivial once other pieces exist.

- [ ] Add `go test -timeout 60s` to CI
- [ ] Add `ailang check --timeout 60s` to CI
- [ ] Add `ailang debug cycles --json` step
- [ ] Configure suspicious cycle detection as CI failure
- [ ] Upload cycle analysis as artifact

**Definition of Done:**
- CI fails on suspicious cyclic types
- CI fails on test timeouts

## Success Criteria

- [ ] `ailang check --timeout 5s` on hanging file reports useful error within 5s
- [ ] `ailang debug cycles` identifies cyclic types in stapledons_voyage
- [ ] All type traversal uses safe library
- [ ] CI catches cyclic type hangs before merge
- [ ] No more silent hangs - always feedback within timeout

## Files to Create/Modify

**New files:**
- `internal/types/traverse/traverse.go` - Safe traversal library with ModePath/ModeGlobal
- `internal/types/traverse/traverse_test.go` - Comprehensive tests (90%+ coverage target)
- `internal/pipeline/metrics.go` - Phase timing and count collection
- `cmd/ailang/debug_cycles.go` - Cycle detection command with JSON output
- `examples/complex_types.ail` - Test file with known cyclic types

**Modified files:**
- `cmd/ailang/check.go` - Add `--timeout` flag
- `cmd/ailang/root.go` - Wire up `debug cycles` subcommand
- `internal/pipeline/pipeline.go` - Add context.Context, watchdog timer, metrics integration
- `internal/types/types.go` - Add depth limits to String(), document as diagnostic-only
- `internal/types/typechecker.go` - Migrate to use traverse package
- `internal/types/typechecker_defaulting.go` - Migrate collectFreeVars to traverse
- `.github/workflows/ci.yml` - Add timeout and cycle detection steps

**LOC Estimates:**
| File | New Lines | Change Type |
|------|-----------|-------------|
| traverse/traverse.go | ~200 | New |
| traverse/traverse_test.go | ~150 | New |
| pipeline/metrics.go | ~100 | New |
| debug_cycles.go | ~150 | New |
| complex_types.ail | ~30 | New |
| check.go | +20 | Modify |
| pipeline.go | +50 | Modify |
| types.go | +30 | Modify |
| typechecker*.go | ±50 | Refactor |
| ci.yml | +30 | Modify |
| **Total** | **~810** | - |

## Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Time to diagnose cyclic hang | ~4 hours | <5 minutes |
| Lines of manual debug code needed | ~50 | 0 |
| Silent hang possibility | Yes | No |

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| traverse library changes unification semantics | Keep occurs/SafeEquals semantics identical; extensive test coverage |
| Timeout fires on legitimate slow compile | Off by default; generous default in --debug-compile (60s) |
| Cycle classification incorrect | Conservative: default to "suspicious"; stdlib explicitly marked |
| Migration breaks existing code | Phase 2 includes migration checklist; grep for hand-written recursion |

## Future Work (Out of Scope)

These are explicitly out of scope for M-DX11 but may be addressed later:

1. **IDE integration** - Use JSON output from `debug cycles` command
2. **Type graph visualization** - DOT/Graphviz output for complex cycles
3. **Automatic cycle-safe wrapper generation** - Macro/codegen for new type operations
4. **Performance profiling integration** - Connect metrics to pprof

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09 (incorporated design review feedback)
**Motivation**: M-PERF2 post-mortem lessons learned

## Design Review Notes (2025-12-09)

Feedback incorporated from design review:

### ✅ Accepted Changes

1. **Watchdog: Use context.Context** - Composes with library mode; caller controls cancellation
2. **Timeout off by default** - Prevents false positives on legitimately slow compiles
3. **Two traversal modes** - ModePath for cycle detection, ModeGlobal for collection
4. **Parameterized occurs check** - Never share visitor state across variable names
5. **Add --json to debug cycles** - Machine-readable for IDE/tooling integration
6. **Classify cycles** - Distinguish expected (stdlib) from suspicious (user)
7. **Add counts to metrics** - Detect non-linear behavior (constraints/node ratio)
8. **Use built-in --timeout in CI** - Not POSIX timeout wrapper
9. **Document String() as diagnostic** - No algorithmic logic may depend on it
10. **Reorder phases** - Quick wins first for immediate impact

### 🔍 Things to Watch

- **occurs check semantics** - Must be path-based, not global
- **SafeEquals semantics** - Careful not to change unification behavior
- **Cycle classification** - May need tuning based on real-world usage

### 📚 References

- M-PERF2 post-mortem (motivation)
- [design_docs/implemented/v0_5_6/m-perf2-cycle-detection.md](../../../implemented/v0_5_6/m-perf2-cycle-detection.md) (predecessor)

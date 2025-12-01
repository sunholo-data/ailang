# M-GAME-A: Go Codegen Foundation

**Status:** PLANNED
**Version:** v0.5.0
**Parent:** M-GAME-ENGINE (multi-phase game support enablement)
**Estimated LOC:** 1,200 (800 implementation + 400 tests)
**Estimated Duration:** 2 weeks

## Summary

Establish the `internal/gen/go/` module to generate valid Go code from AILANG ADTs and pure functions. This is the foundation that enables external game projects to consume AILANG logic.

## What This Phase Delivers

After v0.5.0, a game repo can:
- Define `World`, `FrameInput`, `FrameOutput`, `DrawCmd` types in AILANG
- Generate Go structs from AILANG ADTs (discriminator pattern)
- Generate Go functions from pure AILANG functions
- Call exported AILANG functions from Go code

```bash
# Game repo workflow after v0.5.0
ailang compile --emit-go --package-name game --out gen/ world.ail
go build -o sim ./cmd/sim
./sim  # Draws from AILANG logic
```

## What Already Exists

| Component | Status | Notes |
|-----------|--------|-------|
| ADT syntax parsing | ✅ Complete | `type Tree = \| Leaf(int) \| Node(Tree, int, Tree)` |
| Pattern matching | ✅ Complete | Exhaustiveness checking, guards |
| Core AST lowering | ✅ Complete | Surface → Core elaboration |
| Type inference | ✅ Complete | Hindley-Milner with row polymorphism |

## What Needs to Be Built

| Component | Status | Estimated LOC |
|-----------|--------|---------------|
| Go codegen module | ❌ Not started | ~300 |
| ADT → Go struct codegen | ❌ Not started | ~400 |
| Function → Go func codegen | ❌ Not started | ~400 |
| `export func` syntax | ❌ Not started | ~100 |

## Design Decisions (Locked)

### 1. Sum Type Representation: Discriminator Structs

**Decision:** Use discriminator-based representation for all exported ADTs.

**Rationale:** Game engines iterate over thousands of DrawCmds per frame. Interface dispatch creates GC pressure and cache misses. Discriminator structs give branchy-but-contiguous layouts.

```ailang
type Tree = | Leaf(int) | Node(Tree, int, Tree)
```

Generates:

```go
type TreeKind int

const (
    TreeKindLeaf TreeKind = iota
    TreeKindNode
)

type Tree struct {
    Kind TreeKind
    Leaf *TreeLeaf  // populated when Kind == TreeKindLeaf
    Node *TreeNode  // populated when Kind == TreeKindNode
}

type TreeLeaf struct {
    Value0 int64
}

type TreeNode struct {
    Left  *Tree
    Value int64
    Right *Tree
}
```

### 2. Type Mapping

| AILANG Type | Go Type |
|-------------|---------|
| `int` | `int64` |
| `float` | `float64` |
| `bool` | `bool` |
| `string` | `string` |
| `[T]` | `[]T` |
| `{a: T, b: U}` | `struct { A T; B U }` |
| Sum types | Discriminator struct (see above) |

### 3. Naming Convention

| AILANG | Go |
|--------|-----|
| `snake_case` function | `PascalCase` (exported) or `camelCase` (private) |
| `PascalCase` type | `PascalCase` type |
| `snake_case` field | `PascalCase` field |

### 4. Export Semantics

```ailang
export func step(world: World, input: FrameInput) -> (World, FrameOutput) { ... }
func helper(x: int) -> int { ... }  -- not exported
```

- `export func` → Go public function (`func Step(...)`)
- Regular `func` → Go package-private function (`func helper(...)`)

## Milestones

### A1: Go Codegen Infrastructure (Day 1-3)

**Goal:** Create `internal/gen/go/` package structure.

**Files to create:**
- `internal/gen/go/codegen.go` - Main entry point
- `internal/gen/go/types.go` - Type lowering (AILANG → Go)
- `internal/gen/go/naming.go` - Stable naming scheme
- `internal/gen/go/package.go` - Go package generation

**Acceptance Criteria:**
- [ ] Package structure compiles
- [ ] Can generate empty Go package with correct structure
- [ ] Unit tests for naming scheme (CamelCase conversion)

### A2: ADT → Go Struct Generation (Day 4-7)

**Goal:** Generate Go types from AILANG ADTs.

**Tasks:**
- Implement sum type codegen (discriminator struct pattern)
- Implement product type codegen (plain structs)
- Implement recursive type handling
- Implement list/array/map type mapping

**Acceptance Criteria:**
- [ ] Sum types generate discriminator structs (not interfaces)
- [ ] Product types generate plain structs
- [ ] Recursive types compile correctly
- [ ] Generated Go code compiles with `go build`

### A3: Pure Function → Go Func Generation (Day 8-12)

**Goal:** Generate Go functions from pure AILANG functions.

**Tasks:**
- Implement expression codegen (literals, binops, if/then/else)
- Implement pattern matching codegen (match → switch on Kind)
- Implement lambda/closure codegen
- Implement recursion handling

**Example transformation:**
```ailang
func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

Generates:

```go
func Factorial(n int64) int64 {
    if n <= 1 {
        return 1
    }
    return n * Factorial(n - 1)
}
```

**Acceptance Criteria:**
- [ ] Pure functions generate correct Go code
- [ ] Recursion works
- [ ] Pattern matching generates switch statements on Kind
- [ ] Generated code passes `go vet`

### A4: `export func` Syntax (Day 13-14)

**Goal:** Add export keyword to parser.

**Tasks:**
- Add EXPORT token to lexer
- Add `export func` parsing
- Add export flag to AST function node
- Wire through to codegen (public vs private)

**Acceptance Criteria:**
- [ ] `export func foo() -> int { 42 }` parses
- [ ] Non-exported functions are package-private in Go
- [ ] Exported functions are public (PascalCase)

## Out of Scope (Deferred to Later Phases)

- ❌ Effects (RNG, Debug, AI) → Phase 2 (v0.5.1)
- ❌ Extern functions → Phase 3 (v0.5.2)
- ❌ CLI flags (--emit-go, --out) → Phase 3 (v0.5.2)
- ❌ Sim example → Phase 4 (v0.5.3)

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Pattern matching codegen complexity | Medium | High | Start with simple cases, iterate |
| Recursive types causing infinite loops | Low | High | Use pointer types, test thoroughly |
| Generated Go not idiomatic | Low | Medium | Keep readable, optimize later |

## Success Metrics

| Metric | Target |
|--------|--------|
| Test coverage (new code) | >80% |
| Generated Go compiles | 100% |
| Generated code passes `go vet` | 100% |
| Lint clean | All new code |

## Dependencies

- v0.4.x stability (core language)
- ADT implementation (already complete)
- Pattern matching (already complete)

## Next Phase

After M-GAME-A completes, proceed to M-GAME-B (Effects for Games) in v0.5.1.

# M-CODEGEN-IR-STRATEGY: Multi-Layer IR Architecture for Code Generation

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Strategic - Long-term maintainability)
**Estimated**: 2-3 weeks (phased implementation)
**Dependencies**: M-CODEGEN-V2 (complete), M-CODEGEN-V3 (optional)
**Reporter**: Architecture review
**Created**: 2025-12-22

---

## TL;DR

**Problem**: The Go codegen module has grown to **49 files / 15,263 lines** with 27+ incremental design docs. Each new feature adds more special cases, creating a maintenance burden.

**Solution**: Extend the proven Block IR pattern to create a **multi-layer IR architecture**:
```
Core AST → [IR Layers] → Go Source
              ↓
    Decision IR (match expressions)
    Type IR (type resolution)
    Statement IR (control flow)
```

**Key insight**: Block IR reduced function body IIFEs by 58% with just 121 LOC. The same pattern can simplify match (663 LOC), operators (571 LOC), and type analysis (486 LOC).

**Projected gains**:
- 30-40% reduction in codegen LOC
- Clearer separation of concerns
- Easier addition of new backends (Rust, C, WASM)
- Reduced switch statement complexity (currently 47 cases in codegen_ops.go alone)

---

## Problem Statement

### The Growth Pattern

The `internal/gen/golang/` module has grown organically through 27+ incremental design docs:

| Version | Codegen Design Docs | Pattern |
|---------|---------------------|---------|
| v0.5.7 | m-codegen-typed-slices | Special case |
| v0.5.8 | 6 docs (bool-slice, blank-identifier, etc.) | More special cases |
| v0.5.9 | 6 docs (v2-flat-output, pointer-return, etc.) | Architecture + special cases |
| v0.5.10 | 7 docs (list-flatten, tuple-pattern, etc.) | Even more special cases |
| v0.6.0+ | 3 docs (unified-slice, adt-double-paren, etc.) | Continuing pattern |

**Total**: 27 codegen-specific design docs, each adding complexity.

### Current Module Size

```
internal/gen/golang/
├── 49 files
├── 15,263 total lines
├── 8 files over 400 lines
└── 3 files over 500 lines (critical threshold)
```

**Largest files**:
| File | Lines | Responsibility | Switch Cases |
|------|-------|----------------|--------------|
| codegen_runtime_collections.go | 780 | Slice converters | N/A |
| codegen_match.go | 663 | Pattern matching | 41 |
| codegen.go | 596 | Main generator | 5 |
| codegen_ops.go | 571 | Operators | 47 |
| codegen_expr_let.go | 567 | Let expressions | 6 |
| codegen_type_analysis.go | 486 | Type resolution | 40 |

### The Symptom: Switch Statement Explosion

Pattern matching in `codegen_match.go`:
```go
// 16 switches with 41 cases total
switch pattern := pat.(type) {
case *core.LitPattern:
    switch lit := pattern.Value.(type) {
    case *core.LitInt:
        // ...
    case *core.LitFloat:
        // ...
    case *core.LitString:
        // ...
    case *core.LitBool:
        // ...
    }
case *core.VarPattern:
    // ...
case *core.WildcardPattern:
    // ...
case *core.ConstructorPattern:
    switch /* more cases */ {
    }
}
```

This is the **direct opposite** of Block IR's clean model:
```go
// Block IR: 1 loop, 0 switches
for _, stmt := range blk.Stmts {
    g.writef("var %s = ", stmt.Name)
    g.generateExpr(stmt.Value)
}
```

### The Root Cause: Missing Abstraction Layers

Currently, code generation goes directly from Core AST to Go source:
```
Core AST ──────────────────────────────► Go Source
              (all complexity here)
```

This means every Core node type requires:
1. Type analysis (what Go type?)
2. Control flow decisions (if/switch/return?)
3. Expression emission (actual Go code)

All three concerns are interleaved in the same functions.

### What Block IR Proved

Block IR introduced a single abstraction layer:
```
Core AST → Block IR → Go Source
           (flatten)   (emit flat)
```

**Results** (from M-CODEGEN-V2):
- 58% IIFE reduction (437 → 182)
- 10% total line reduction (6,184 → 5,594)
- Go compiler OOM → successful build
- Just **121 LOC** for the IR layer

This validates the architecture: **small, focused IRs can dramatically simplify codegen**.

---

## Goals

### Primary Goal

Reduce codegen complexity by 30-40% through targeted IR layers while maintaining correctness and performance.

### Success Metrics

**Primary metrics** (complexity, not LOC):

| Metric | Current | Target | Verification |
|--------|---------|--------|--------------|
| Packages depending on `core.*Pattern` | 4 | 1 | `grep -r "core\.\*Pattern" --include="*.go" \| cut -d: -f1 \| xargs dirname \| sort -u \| wc -l` |
| Packages inspecting `types.Type` directly | 6 | 2 | Similar grep |
| Switch cases in match codegen | 41 | <15 | `grep -c "case "` |
| Switch cases in ops codegen | 47 | <15 | `grep -c "case "` |
| Time to add new pattern type | ~2 hours | <30 min | Measured on next feature |

**Secondary metrics** (quality indicators):

| Metric | Current | Target | Verification |
|--------|---------|--------|--------------|
| Max file size | 780 lines | <500 lines | File size check |
| Golden test corpus | 0 | 20+ files | `ls tests/golden/codegen/` |
| Emitter packages that know Core semantics | 1 (golang/) | 1 | Architecture review |

### Non-Goals

- Full typed codegen (M-DX24 scope)
- Performance optimization of generated code
- Changing Core AST structure
- Breaking existing generated code

---

## Solution Design

### Architecture: Single Emitter-Facing IR

**Key insight**: Don't build competing body representations. Statement IR is the **only** representation emitters see. Block lowering and match lowering are internal passes that compile into Statement IR.

```
┌─────────────────────────────────────────────────────────────────┐
│                         Core AST                                 │
│                    + CoreTypeInfo (resolved)                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │   Lowering Pass   │
                    │                   │
                    │  ┌─────────────┐  │
                    │  │ Block lower │  │  (let chains → flat stmts)
                    │  └─────────────┘  │
                    │  ┌─────────────┐  │
                    │  │Match lower  │  │  (patterns → decision tree → if/switch)
                    │  │(internal)   │  │
                    │  └─────────────┘  │
                    │  ┌─────────────┐  │
                    │  │Type project │  │  (types.Type → ResolvedType, pure projection)
                    │  └─────────────┘  │
                    └─────────┬─────────┘
                              ▼
                    ┌───────────────────┐
                    │   Statement IR    │
                    │  (the ONLY repr   │
                    │   emitters see)   │
                    └─────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   Go Emitter  │    │  Rust Emitter │    │  WASM Emitter │
│   (backend)   │    │   (future)    │    │   (future)    │
└───────────────┘    └───────────────┘    └───────────────┘
```

**Emitter API**: `EmitFunc(FuncBody)` only. Emitters never see Core, Block IR, or decision trees.

### IR Layer 1: Block IR (Already Implemented)

**Location**: `internal/gen/block/`
**Purpose**: Flatten let-chains to statement sequences
**Status**: ✅ Complete (121 LOC)

```go
type Block struct {
    Stmts     []Stmt
    FinalExpr core.CoreExpr
}

type Stmt struct {
    Name  string
    Value core.CoreExpr
}
```

**Impact**: Eliminated 58% of IIFEs.

### IR Layer 2: Match Lowering (Internal Pass)

**Location**: `internal/gen/lower/` (new)
**Purpose**: Convert pattern matching to decision trees, then to Statement IR
**Visibility**: **Internal** - not exposed to emitters, compiles away into Statement IR

```go
// Decision tree is an INTERNAL representation, not a public IR.
// It exists only during the match lowering pass.

type decisionNode interface {
    decisionNode()
}

// Test: evaluate a condition, branch left/right
type testNode struct {
    Scrutinee  selector     // What to examine (e.g., x.tag, x.field)
    Condition  condition    // How to test (eq, lt, hasTag, etc.)
    IfTrue     decisionNode // Branch if condition holds
    IfFalse    decisionNode // Branch otherwise
}

// Leaf: execute an action
type Leaf struct {
    Bindings []Binding      // Variables to bind
    Body     core.CoreExpr  // Expression to evaluate
}

// Selector: path to a value
type Selector struct {
    Base   string           // Starting variable
    Path   []SelectorStep   // Field accesses, tag checks, etc.
}

type SelectorStep interface {
    selectorStep()
}

type FieldAccess struct { Field string }
type TagCheck struct { Tag string }
type TupleIndex struct { Index int }
```

**Example transformation**:

```ailang
match pair with
| (0, y) -> y
| (x, 0) -> x
| (x, y) -> x + y
```

**Current codegen** (inline in codegen_match.go):
```go
switch {
case pair_0 == 0:
    return pair_1
case pair_1 == 0:
    return pair_0
default:
    return pair_0 + pair_1
}
```

**With match lowering** (compiles to Statement IR):
```go
// Phase 1: Core Match → internal decision tree
tree := lower.buildDecisionTree(matchExpr)

// Phase 2: Decision tree → Statement IR (If/Switch nodes)
// This happens INSIDE the lowering pass, not exposed externally
stmts := lower.decisionToStatements(tree)
// Returns Statement IR:
// IfStmt{
//   Cond: BinOp{VarRef("pair_0"), "==", Literal(0)},
//   Then: [VarDecl{y, VarRef("pair_1")}, Return(VarRef("y"))],
//   Else: [IfStmt{...}],
// }

// Phase 3: Statement IR → Go (emitter sees only Statement IR)
emitter.EmitFunc(funcBody)  // funcBody contains the IfStmt above
```

**Semantic invariants** (must be preserved by lowering):

| Invariant | Implementation |
|-----------|----------------|
| Scrutinee evaluated exactly once | Hoist to temp var in Statement IR |
| No duplicate field access | Cache selector results in temp vars |
| Binding only on taken branch | Bindings inside branch statements |
| Tag check before field access | TagSwitch guards field access |
| Nil-safe for Option types | Explicit nil check before unwrap |

**Estimated impact**:
- codegen_match.go: 663 → ~300 lines (-55%)
- Switch cases: 41 → ~10 (-76%)
- Pattern type handling: Centralized in lower/match.go

### IR Layer 3: Type Projection (Pure Mapping)

**Location**: `internal/gen/typeres/` (new)
**Purpose**: Map resolved AILANG types to target language types
**Critical constraint**: **Pure projection only** - no unification, no defaulting, no inference

```go
// Type projection is a PURE FUNCTION from resolved types to backend types.
// If CoreTypeInfo has unresolved type variables, this ERRORS (don't silently default).

type ResolvedType interface {
    resolvedType()
}

type Primitive struct {
    Kind PrimitiveKind  // Int64, Float64, Bool, String
}

type Struct struct {
    Name   string
    Fields []Field
}

type Slice struct {
    Element ResolvedType
}

type Function struct {
    Params []ResolvedType
    Return ResolvedType
}

type ADT struct {
    Name     string
    Variants []Variant
}

type Variant struct {
    Tag    string
    Fields []ResolvedType
}

// Project performs deterministic type mapping.
// CRITICAL: No defaulting. If type is unresolved, return error.
func Project(t types.Type, config *BackendConfig) (ResolvedType, error) {
    // This is a PURE FUNCTION:
    // - Same input → same output, always
    // - No side effects, no caching
    // - If t contains type variables → error (not silent default)

    switch t := t.(type) {
    case *types.TyCon:
        switch t.Name {
        case "int": return Primitive{Int64}, nil
        case "float": return Primitive{Float64}, nil
        // ...
        }
    case *types.TyVar:
        return nil, fmt.Errorf("unresolved type variable %s at codegen", t.Name)
    // ...
    }
}
```

**Non-goals for type projection** (learned from float dispatch bug):
- ❌ Type inference or unification
- ❌ Defaulting type variables to any concrete type
- ❌ Caching or memoization with mutation
- ❌ Any logic that depends on "context" beyond the type itself

**Estimated impact**:
- codegen_type_analysis.go: 486 → ~200 lines (core logic)
- Type switches in other files: Eliminated (use ResolvedType)
- New typeres/: ~150 lines (definitions + projection)
- Total: Net reduction of ~100-150 lines + clearer structure

### IR Layer 4: Statement IR (New)

**Location**: `internal/gen/stmt/` (new)
**Purpose**: Unified control flow representation

```go
// Statement IR unifies function bodies
type FuncBody struct {
    Params []Param
    Body   []Stmt
    Return Expr
}

type Stmt interface {
    stmt()
}

type VarDecl struct {
    Name  string
    Type  ResolvedType  // From Type IR
    Value Expr
}

type IfStmt struct {
    Cond     Expr
    Then     []Stmt
    Else     []Stmt
}

type SwitchStmt struct {
    Scrutinee Expr
    Cases     []Case
    Default   []Stmt
}

type Case struct {
    Value Expr
    Body  []Stmt
}

type Expr interface {
    expr()
}

// Expressions reference the Statement IR types
type VarRef struct { Name string }
type Literal struct { Value interface{}; Type ResolvedType }
type BinOp struct { Left, Right Expr; Op string }
type Call struct { Func string; Args []Expr }
type FieldAccess struct { Base Expr; Field string }
```

**This unifies**:
- Block IR's flat statements
- Decision IR's test/leaf structure
- Type IR's resolved types

**Estimated impact**:
- Enables direct emission for any backend
- Eliminates duplication between expression handlers
- Single walk for Go, Rust, WASM emission

---

## Implementation Plan

### Phase 1: Decision IR (Week 1)

**Goal**: Extract pattern matching logic into Decision IR.

**New files**:
- `internal/gen/decision/decision.go` (~150 LOC) - IR types
- `internal/gen/decision/lower.go` (~300 LOC) - Core Match → Decision
- `internal/gen/decision/lower_test.go` (~200 LOC) - Unit tests

**Modified files**:
- `internal/gen/golang/codegen_match.go` - Use Decision IR (~300 lines removed)

**Milestones**:
| Day | Task | Verification |
|-----|------|--------------|
| 1 | Decision IR types + basic Lower() | Unit tests pass |
| 2 | Complete pattern coverage | All Core patterns handled |
| 3 | Integrate with codegen_match.go | Existing tests pass |
| 4 | Cleanup + documentation | `make lint` passes |

**Acceptance criteria**:
- [ ] All existing match tests pass
- [ ] codegen_match.go under 400 lines
- [ ] No new IIFEs introduced

### Phase 2: Type IR Extraction (Week 2)

**Goal**: Centralize type resolution.

**New files**:
- `internal/gen/typeres/types.go` (~150 LOC) - ResolvedType definitions
- `internal/gen/typeres/resolver.go` (~200 LOC) - Resolution logic
- `internal/gen/typeres/resolver_test.go` (~150 LOC) - Tests

**Modified files**:
- `internal/gen/golang/codegen_type_analysis.go` - Use Type IR (~250 lines removed)
- `internal/gen/golang/codegen_ops.go` - Use Type IR (~100 lines removed)

**Acceptance criteria**:
- [ ] All type analysis tests pass
- [ ] No type switches outside typeres/
- [ ] codegen_ops.go under 500 lines

### Phase 3: Statement IR (Week 3)

**Goal**: Unified statement representation.

**New files**:
- `internal/gen/stmt/stmt.go` (~100 LOC) - Statement IR types
- `internal/gen/stmt/builder.go` (~150 LOC) - Build statements from Core

**Modified files**:
- `internal/gen/golang/codegen_block.go` - Use Statement IR
- `internal/gen/golang/codegen_expr_control.go` - Use Statement IR

**Stretch goal**: Rust emitter stub demonstrating backend portability.

---

## Projected Gains

### Lines of Code

| Component | Before | After | Reduction |
|-----------|--------|-------|-----------|
| codegen_match.go | 663 | ~300 | -55% |
| codegen_type_analysis.go | 486 | ~200 | -59% |
| codegen_ops.go | 571 | ~450 | -21% |
| codegen_expr_control.go | 406 | ~300 | -26% |
| New IR packages | 0 | ~800 | +800 |
| **Net total** | 2,126 | ~2,050 | **-4%** |

Wait - the net LOC is similar. So what's the gain?

### Actual Gains: Complexity Reduction

The value isn't raw LOC reduction, but **complexity reduction**:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Switch cases in codegen_match.go | 41 | ~10 | **-76%** |
| Switch cases in codegen_ops.go | 47 | ~15 | **-68%** |
| Files that know about Core patterns | 4 | 1 | **-75%** |
| Files that know about type resolution | 6 | 1 | **-83%** |
| Time to add new pattern type | ~2 hours | ~30 min | **-75%** |
| Time to add new backend | Weeks | Days | **-80%** |

### Maintainability

**Before** (adding new pattern type):
1. Modify codegen_match.go (+20 lines, 3 switches)
2. Modify codegen_type_analysis.go (+10 lines)
3. Modify tests in 2-3 files
4. Hope you found all the places

**After** (adding new pattern type):
1. Add case to lower/match.go (+5 lines)
2. Add test to lower/match_test.go
3. Done - emitters don't change

### Backend Portability

**Before**: Adding Rust backend would require:
- Duplicate all 15,263 lines with Rust equivalents
- Maintain two parallel implementations
- Risk semantic drift between backends

**After**: Adding Rust backend requires:
- Implement ~500-line RustEmitter
- Walk the same IR structures
- Guaranteed semantic equivalence

---

## Testing Strategy

**Unit tests per package aren't enough for this refactor.** Add golden and metamorphic tests.

### Golden Tests

```
tests/golden/codegen/
├── match_simple.ail          → match_simple.go.golden
├── match_nested_adt.ail      → match_nested_adt.go.golden
├── tuple_patterns.ail        → tuple_patterns.go.golden
├── list_patterns.ail         → list_patterns.go.golden
├── deep_let_chains.ail       → deep_let_chains.go.golden
└── stdlib_subset.ail         → stdlib_subset.go.golden
```

**Normalization**: Use `go/format` + deterministic temp naming (hash-based) so golden files don't churn on irrelevant changes.

### Metamorphic Tests

**Property**: Old codegen output and new codegen output must both compile and behave identically.

```go
func TestMetamorphic(t *testing.T) {
    for _, file := range testCorpus {
        oldCode := generateWithLegacy(file)
        newCode := generateWithStatementIR(file)

        // Both must compile
        require.NoError(t, compileGo(oldCode))
        require.NoError(t, compileGo(newCode))

        // Both must produce identical output for same inputs
        for _, seed := range testSeeds {
            oldResult := runWithSeed(oldCode, seed)
            newResult := runWithSeed(newCode, seed)
            require.Equal(t, oldResult, newResult)
        }
    }
}
```

### Test Corpus

Sources for golden/metamorphic tests:
1. **stdlib subset**: `std/list.ail`, `std/option.ail`
2. **stapledons_voyage files**: Real-world patterns from game
3. **stress patterns**: Deep matches, nested tuples, ADTs with many fields
4. **edge cases**: Empty patterns, wildcard-only, single-variant ADTs

---

## Risk Analysis

### Risk 1: Performance Overhead

**Concern**: Extra IR layers = more allocations.

**Analysis**: Block IR showed negligible overhead (191ns for 10 nodes). Decision trees are similar size to pattern ASTs.

**Mitigation**: Benchmark before/after; IRs are short-lived (GC-friendly).

### Risk 2: Breaking Changes

**Concern**: Refactoring could introduce bugs.

**Mitigation**:
- Phase incrementally (one IR at a time)
- Run full test suite after each phase
- Keep old code paths for rollback

### Risk 3: Over-Engineering

**Concern**: Maybe the current approach is fine?

**Analysis**: 27 codegen design docs in 6 months shows unsustainable growth. Each new feature becomes harder. Block IR proved the pattern works.

**Decision**: Proceed, but validate after Phase 1 before committing to Phases 2-3.

---

## Success Criteria

### Phase 1 Gate (Match Lowering)

- [ ] codegen_match.go under 400 lines
- [ ] All pattern matching tests pass
- [ ] Match lowering → Statement IR documented and tested
- [ ] Golden tests for match patterns (10+ files)
- [ ] Metamorphic tests pass for stapledons_voyage subset

### Phase 2 Gate (Type Projection)

- [ ] Type switches centralized in typeres/
- [ ] codegen_type_analysis.go under 300 lines
- [ ] codegen_ops.go under 500 lines
- [ ] Type projection errors on unresolved types (no silent defaults)

### Phase 3 Gate (Statement IR as Single Interface)

- [ ] Emitter API is `EmitFunc(FuncBody)` only
- [ ] Rust emitter stub compiles (demonstrates portability)
- [ ] Total packages knowing Core semantics: 2 (lower/ + typeres/)

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | IR transformations are deterministic |
| A7: Machines First | +2 | IRs are machine-readable, analyzable |
| A8: Minimal Syntax | 0 | No AILANG syntax changes |
| A10: Composability | +2 | IRs compose cleanly between layers |

**Net Score: +5** - Proceed with implementation.

---

## Related Documents

- [M-CODEGEN-V2](../../implemented/v0_5_9/m-codegen-v2-flat-output.md) - Block IR architecture (foundation)
- [M-CODEGEN-V3](../v0_6_2/m-codegen-v3-binding-hoisting.md) - Binding hoisting (extends Block IR)
- [Block IR implementation](../../../internal/gen/block/) - Existing IR code

---

## Follow-up Questions (from Review)

### Q1: Do you want to support a second backend soon (Rust stub), or is that mainly a forcing function to improve architecture?

**Answer**: Primarily a forcing function. A Rust stub proves the architecture works and prevents Go-specific assumptions from creeping in. We don't have immediate plans to ship Rust codegen, but the constraint "emitters only see Statement IR" keeps the design honest.

**Recommendation**: Implement ~100-line Rust stub in Phase 3 that emits syntactically valid Rust (doesn't need to compile/run). This validates the abstraction without the overhead of a full Rust backend.

### Q2: Is core.Match already desugared into a small number of pattern primitives, or do you still have many pattern forms at codegen time?

**Answer**: Currently, `core.Match` contains the full set of pattern forms at codegen time:
- `LitPattern` (int, float, string, bool literals)
- `VarPattern` (variable binding)
- `WildcardPattern` (underscore)
- `ConstructorPattern` (ADT constructors with nested patterns)
- `TuplePattern` (tuple destructuring)
- `ListPattern` (list head/tail patterns)
- `RecordPattern` (record field patterns)

Each of these is handled with type switches in `codegen_match.go`, leading to the 41-case explosion.

**Impact on design**: The match lowering pass must handle all these forms. A possible simplification is to desugar complex patterns (list, record) to simpler forms in an earlier pass, but that's a separate design decision.

### Q3: Do you have a stable "normalized Go output" formatter already (go/format + deterministic temp naming), so golden tests don't churn?

**Answer**: Partially.
- **go/format**: We use `format.Source()` for final output, so whitespace is stable.
- **Temp naming**: Currently uses incrementing counters (`tmp1`, `tmp2`...) which can shift if code changes.

**Gap**: Need deterministic naming based on source location or content hash. Proposed approach:
```go
// Instead of: tmp42
// Use: tmp_<srcLine>_<srcCol> or tmp_<hash(expr)[:8]>
```

This should be implemented as a prerequisite for golden tests (Phase 0 or early Phase 1).

---

## Changelog

| Date | Change |
|------|--------|
| 2025-12-22 | Initial design document |
| 2025-12-22 | Incorporated review feedback: single emitter-facing IR, match lowering as internal pass, type projection as pure function, testing strategy with golden/metamorphic tests, semantic invariants for match lowering |

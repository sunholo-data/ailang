# M-CODEGEN-IR-LAYERS: Multi-Target Codegen IR Architecture

**Status**: Planned
**Target**: v0.9.3 (Phase 0-1), v0.10.0 (Phase 2-3)
**Priority**: P1 (Strategic — enables multi-target without duplicating 18k LOC)
**Estimated**: 3-4 weeks phased
**Dependencies**: M-CODEGEN-SUSTAINABILITY (complete), Block IR (complete)
**Supersedes**: design_docs/planned/v0_10_0/m-codegen-ir-strategy.md (updated scope, phased delivery)

---

## TL;DR

**Problem**: The Go codegen is 57 files / ~18k LOC with no abstraction between Core AST and Go emission. Every new feature adds Go-specific switch cases. Adding Rust/WASM/SMT backends would require duplicating all of this.

**Solution**: Build 3 IR layers between Core AST and target emitters, extending the proven Block IR pattern:

```
Core AST
    ↓
Block IR      ← DONE (v0.5.9) — flatten let chains → statements
    ↓
Decision IR   ← Phase 1 — match patterns → decision trees
    ↓
Type IR       ← Phase 2 — types.Type → ResolvedType (pure projection)
    ↓
Statement IR  ← Phase 3 — unified repr for ALL backends
    ↓
Go / Rust / WASM / SMT emitters
```

**Phase 0 (DONE)**: Registry-based builtin codegen specs are target-agnostic — the `GoCodegenSpec` struct already separates *what* to generate from *how* to emit it.

**Projected gains**: 30-40% codegen LOC reduction, single match/type logic for all backends, each new backend is ~500 LOC emitter instead of 18k LOC fork.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | IR passes are pure functions — same Core in, same IR out |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Codegen internals, no user-facing effects |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Each IR layer can be independently verified |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Cleaner IR = better for AI-generated codegen extensions |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No runtime cost changes |
| A10: Composability | +1 | IR layers compose — Decision IR works with or without Type IR |
| A11: Structured Failure | +1 | IR lowering fails explicitly, not via silent fallbacks |
| A12: System Boundary | 0 | Internal compiler change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): IR passes are pure functions
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Makes codegen more machine-analyzable

---

## Problem Statement

### The Maintenance Burden Pattern

Every codegen fix follows the same cycle:
1. DocParse (or another project) hits a codegen bug
2. We trace it to a missing case in Go-specific switch statements
3. We add the case to Go codegen
4. The fix is invisible to any future Rust/WASM backend

**Evidence from this sprint alone**:
- Let-binding bug: `topLevelFuncs` vs `topLevelVars` — purely Go-specific
- Inline template resolution: `{{arg0}}` substitution at VarGlobal vs App level — Go-specific
- 5 missing runtime symbols: each required a Go function body in `registry_codegen.go`
- JSON helper guard: checking `adtConstructors["JString"]` — Go emission detail

None of these fixes would help a Rust backend.

### Current Architecture (No IR)

```
Core AST ──────────────────────────────► Go Source (18k LOC, 57 files)
            direct pattern matching
            on 7 Core node types
            with 40+ switch statements
```

### Existing Targets

| Target | Location | Status | LOC |
|--------|----------|--------|-----|
| **Go** | `internal/gen/golang/` | Production | ~18,000 |
| **Block IR** | `internal/gen/block/` | Production | 121 |
| **SMT** | `internal/gen/smt/` | Experimental | ~800 |
| **WASM** | — | Not started | — |

The SMT backend already handles Core AST directly — it would benefit from shared Decision IR and Type IR.

---

## Goals

**Primary Goal**: Introduce IR layers so that adding a new codegen backend requires writing only a ~500 LOC emitter, not forking 18k LOC.

**Success Metrics:**
- `codegen_match.go` reduced from 859 → ~400 LOC (Decision IR absorbs pattern logic)
- `codegen_type_analysis.go` reduced from 504 → ~250 LOC (Type IR absorbs resolution)
- New backend (Rust stub) achievable in 1 day using Statement IR
- `make test-codegen` passes after each phase (no regressions)
- Golden test corpus with 15+ reference files for safe refactoring

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Statement IR node types | Defines what ALL backends receive | human | design | high |
| Decision tree algorithm | Affects match exhaustiveness checking | agent | Phase 1 | med |
| Type IR error handling | Panic vs error return on unresolved types | human | design | high |
| Where to split inline vs helper builtins | Affects whether Rust/WASM can use same registry | human | Phase 3 | med |
| Block IR: keep current or extend | Current only handles top-level lets | agent | Phase 1 | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Statement IR node type inventory (what nodes exist)
- [ ] Type IR error semantics (panic on unresolved, or return error?)
- [ ] Whether builtin GoCodegenSpec should become target-agnostic BuiltinCodegenSpec

---

## Solution Design

### Overview

Four IR layers, each a pure function from one representation to another. Each layer can be implemented, tested, and shipped independently.

### Architecture

```
                    ┌─────────────────────────────────────────┐
                    │              Core AST                    │
                    │  (Let, Lambda, Match, App, VarGlobal...) │
                    └───────────┬─────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────┐
                    │           Block IR (DONE)                │
                    │  Flatten let chains → [Stmt] + final     │
                    │  121 LOC, 58% IIFE reduction             │
                    └───────────┬─────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────┐
                    │         Decision IR (Phase 1)            │
                    │  Match arms → decision tree              │
                    │  TestNode | Leaf | Fail                  │
                    │  Handles: exhaustiveness, overlap, order  │
                    └───────────┬─────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────┐
                    │          Type IR (Phase 2)               │
                    │  types.Type → ResolvedType               │
                    │  Pure projection, no inference            │
                    │  Panic on unresolved (no silent fallback) │
                    └───────────┬─────────────────────────────┘
                                │
                    ┌───────────▼─────────────────────────────┐
                    │        Statement IR (Phase 3)            │
                    │  Unified representation for emitters     │
                    │  FuncDecl, VarDecl, IfStmt, Switch,      │
                    │  Call, FieldAccess, TypeAssert, Return    │
                    └───────┬───────────┬──────────┬──────────┘
                            │           │          │
                    ┌───────▼──┐  ┌─────▼────┐  ┌─▼────────┐
                    │ Go Emit  │  │ Rust Emit│  │ WASM Emit│
                    │ ~2k LOC  │  │ ~500 LOC │  │ ~500 LOC │
                    └──────────┘  └──────────┘  └──────────┘
```

### Phase 0: Foundation (DONE — M-CODEGEN-SUSTAINABILITY sprint)

Already completed in this sprint:
- [x] Registry-based builtin codegen specs (`BuiltinMeta.GoCodegen`)
- [x] Inline template resolution at App level (`tryResolveInlineApp`)
- [x] `markLegacyHelpersEmitted` dedup bridge
- [x] `make test-codegen` CI harness (4 multi-module .ail files → Go → `go build`)
- [x] Compile pipeline reorder (modules first, then runtime)
- [x] Unified `IsUserDefinedType`

### Phase 1: Decision IR (~1 week)

**Package**: `internal/gen/decision/`

**Types**:
```go
// DecisionTree is the compiled form of a match expression.
type DecisionTree interface{ isDecisionTree() }

// Test checks a condition and branches.
type Test struct {
    Selector  Selector       // what to test (e.g., scrutinee.Kind)
    Branches  map[string]*DecisionTree  // value → subtree
    Default   *DecisionTree  // wildcard/else branch
}

// Leaf is a successful match — execute the body.
type Leaf struct {
    Bindings map[string]Accessor  // variable → how to extract
    BodyExpr core.CoreExpr        // original body expression
}

// Fail represents match exhaustiveness failure.
type Fail struct{}
```

**Tasks**:
- [ ] Define `DecisionTree`, `Test`, `Leaf`, `Selector` types (~80 LOC)
- [ ] Implement `Compile(arms []core.MatchArm) *DecisionTree` (~200 LOC)
- [ ] Handle: literal patterns, constructor patterns, wildcard, variable binding
- [ ] Handle: nested patterns (constructor with fields)
- [ ] Handle: list patterns (empty, cons, length)
- [ ] Unit tests with 15+ cases (~300 LOC)
- [ ] Refactor `codegen_match.go` to walk DecisionTree instead of raw patterns
- [ ] Golden tests: verify identical Go output before/after
- [ ] `make test && make test-codegen`

### Phase 2: Type IR (~1 week)

**Package**: `internal/gen/typeres/`

**Types**:
```go
// ResolvedType is the target-agnostic type after resolution.
type ResolvedType interface{ isResolvedType() }

type Primitive struct { Name string }        // int64, float64, bool, string
type Pointer struct { Elem ResolvedType }     // *T
type Slice struct { Elem ResolvedType }       // []T
type Record struct { Name string; Fields []Field }
type ADT struct { Name string; Variants []Variant }
type Function struct { Params []ResolvedType; Return ResolvedType }
type Interface struct{}                       // interface{} / any
```

**Tasks**:
- [ ] Define ResolvedType hierarchy (~100 LOC)
- [ ] Implement `Resolve(t types.Type, ctx ResolutionContext) (ResolvedType, error)` (~200 LOC)
- [ ] Panic/error on unresolved types — NO silent "interface{}" fallback
- [ ] Refactor `codegen_type_analysis.go` to use Type IR
- [ ] Refactor `codegen_ops.go` operator type dispatch to use ResolvedType
- [ ] Unit tests (~200 LOC)
- [ ] `make test && make test-codegen`

### Phase 3: Statement IR (~1 week)

**Package**: `internal/gen/stmt/`

**Types**:
```go
type Stmt interface{ isStmt() }

type FuncDecl struct {
    Name    string
    Params  []Param
    Return  ResolvedType
    Body    []Stmt
}

type VarDecl struct { Name string; Type ResolvedType; Value Expr }
type Assign struct { Target string; Value Expr }
type IfStmt struct { Cond Expr; Then []Stmt; Else []Stmt }
type Switch struct { Scrutinee Expr; Cases []Case; Default []Stmt }
type Return struct { Value Expr }
type ExprStmt struct { Expr Expr }

type Expr interface{ isExpr() }
type VarRef struct { Name string }
type Literal struct { Value interface{}; Type ResolvedType }
type Call struct { Func Expr; Args []Expr }
type FieldAccess struct { Object Expr; Field string }
type TypeAssert struct { Expr Expr; Type ResolvedType }
type BinOp struct { Op string; Left, Right Expr }
type Index struct { Collection Expr; Index Expr }
```

**Tasks**:
- [ ] Define Statement IR types (~150 LOC)
- [ ] Implement `Lower(core *core.Program) *Program` using Block/Decision/Type IRs (~400 LOC)
- [ ] Implement `EmitGo(prog *Program) []byte` — Go emitter walking Statement IR (~600 LOC)
- [ ] Verify output parity with existing codegen via golden tests
- [ ] Stub Rust emitter to prove architecture (~100 LOC)
- [ ] `make test && make test-codegen`

### Files to Modify/Create

**New files:**
- `internal/gen/decision/decision.go` — Decision tree types (~80 LOC)
- `internal/gen/decision/compile.go` — Match → decision tree (~200 LOC)
- `internal/gen/decision/compile_test.go` — Unit tests (~300 LOC)
- `internal/gen/typeres/resolve.go` — Type resolution (~300 LOC)
- `internal/gen/typeres/resolve_test.go` — Unit tests (~200 LOC)
- `internal/gen/stmt/stmt.go` — Statement IR types (~150 LOC)
- `internal/gen/stmt/lower.go` — Core → Statement IR (~400 LOC)
- `internal/gen/stmt/emit_go.go` — Go emitter (~600 LOC)
- `tests/golden/codegen/` — Golden test corpus (15+ files)

**Modified files:**
- `internal/gen/golang/codegen_match.go` — Use Decision IR (859 → ~400 LOC)
- `internal/gen/golang/codegen_type_analysis.go` — Use Type IR (504 → ~250 LOC)
- `internal/gen/golang/codegen_ops.go` — Use ResolvedType dispatch (586 → ~500 LOC)

---

## Examples

### Example 1: Match Expression Through Decision IR

**AILANG source:**
```
match shape {
  Circle(r) => PI * r * r,
  Rectangle(w, h) => w * h,
  _ => 0.0
}
```

**Current codegen** (directly in codegen_match.go — 7 switch cases):
```go
switch shape.Kind {
case ShapeKindCircle:
    r := shape.Circle.Value0
    return math.Pi * r * r
case ShapeKindRectangle:
    w := shape.Rectangle.Value0
    h := shape.Rectangle.Value1
    return w * h
default:
    return 0.0
}
```

**With Decision IR** (compiled to tree, then emitted):
```
DecisionTree:
  Test(scrutinee.Kind)
    ├── "Circle" → Leaf{bindings: {r: .Circle.Value0}, body: PI*r*r}
    ├── "Rectangle" → Leaf{bindings: {w: .Rect.Value0, h: .Rect.Value1}, body: w*h}
    └── default → Leaf{bindings: {}, body: 0.0}
```

The Go emitter walks this tree. A Rust emitter would walk the same tree but emit `match shape { ... }`.

### Example 2: Type Resolution Through Type IR

**Current** (scattered across codegen_type_analysis.go):
```go
// 40 switch cases in mapGoType, isUserDefinedType, getSliceConversion...
func mapGoType(t types.Type) string {
    switch t := t.(type) {
    case *types.TCon:
        switch t.Name {
        case "int": return "int64"
        case "float": return "float64"
        // ... 15 more cases
        }
    // ... 25 more cases
    }
}
```

**With Type IR** (resolved once, used everywhere):
```go
resolved := typeres.Resolve(ailangType, ctx)
// resolved is Primitive{"int64"} or Slice{Primitive{"string"}} or ADT{"Option", ...}
// Go emitter: resolved.GoString() → "int64"
// Rust emitter: resolved.RustString() → "i64"
```

---

## Success Criteria

- [ ] Decision IR compiles all 7 Core pattern types (Var, Lit, Constructor, List, Record, Wildcard, Tuple)
- [ ] `codegen_match.go` uses Decision IR (LOC ≤ 450)
- [ ] Type IR resolves all AILANG types without silent fallbacks
- [ ] Statement IR generates identical Go output to current codegen (golden test parity)
- [ ] `make test-codegen` passes with all 4 harness modules
- [ ] Rust emitter stub compiles and produces syntactically valid output for 3+ test cases
- [ ] All existing tests passing
- [ ] Documentation: each IR layer has a README with type inventory

---

## Testing Strategy

**Golden tests (new):**
- Compile 15+ .ail files to Go via both old and new paths
- Diff output — must be identical (modulo whitespace/ordering)
- Store golden files in `tests/golden/codegen/`

**Unit tests (per IR layer):**
- Decision IR: 15+ match patterns (nested, overlapping, exhaustive, non-exhaustive)
- Type IR: all primitive types, ADTs, records, functions, generics
- Statement IR: round-trip (Core → IR → Go) for all expression types

**Integration (existing):**
- `make test` — all Go tests
- `make test-codegen` — multi-module compile + `go build`
- `make verify-examples` — 152 example files

---

## Deferred Decisions

- **Target-agnostic builtin specs** — Currently `GoCodegenSpec`; generalizing to `BuiltinCodegenSpec` with per-target overrides can happen in Phase 3. Agent may choose naming.
- **Decision tree optimization** — Column-selection heuristics (test the most discriminating field first) deferred to after basic compilation works. Agent may implement greedy approach.
- **Statement IR pretty-printer** — Useful for debugging but not required. Agent may add if helpful.

---

## Non-Goals

- **Full Rust backend** — Stub only. Production Rust backend is v0.11.0+
- **WASM MVP** — Out of scope. Statement IR enables it but implementation is separate
- **Rewriting existing Go codegen** — We migrate incrementally. Old code stays until parity verified.
- **Changing AILANG semantics** — IR is an internal compiler concern, not user-facing

---

## Timeline

**Week 1** (Phase 0 validation + Phase 1 start):
- Golden test corpus setup (15 reference files)
- Decision IR types + compiler
- Unit tests for all 7 pattern types

**Week 2** (Phase 1 complete + Phase 2 start):
- Refactor codegen_match.go to use Decision IR
- Type IR types + resolver
- Verify `make test-codegen` still passes

**Week 3** (Phase 2 complete + Phase 3 start):
- Refactor codegen_type_analysis.go to use Type IR
- Statement IR types + Core→IR lowering
- Go emitter from Statement IR

**Week 4** (Phase 3 complete):
- Golden test parity verification
- Rust emitter stub
- Documentation

**Total: ~4 weeks, can ship Phase 1 independently for v0.9.3**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Golden test drift during migration | High | Run both paths, diff before deleting old |
| Decision tree doesn't cover all pattern combos | Med | Comprehensive unit tests for all 7 pattern types |
| Statement IR too abstract, loses Go-specific optimizations | Med | Allow target-specific "hints" on IR nodes |
| Scope creep into full Rust backend | Med | Hard boundary: stub only, syntactically valid, no runtime |
| Legacy code removal breaks DocParse | High | DocParse compilation is part of CI (`make test-codegen` or equivalent) |

---

## Relationship to Prior Work

### What we keep:
- **Block IR** (`internal/gen/block/`) — proven, 58% IIFE reduction, no changes needed
- **Registry-based builtin specs** (M-CODEGEN-SUSTAINABILITY) — the `GoCodegenSpec` pattern becomes the template for target-agnostic specs
- **CI harness** (`make test-codegen`) — validates parity throughout migration

### What we supersede:
- **m-codegen-ir-strategy.md** (v0.10.0) — same vision, updated with Phase 0 completion and revised timeline
- **m-codegen-v3-binding-hoisting.md** — subsumed by Statement IR's VarDecl hoisting

### What we inform:
- **m-codegen-api-server.md** — compiled API server benefits from Statement IR (Go binary codegen)
- **Future Rust backend** — Statement IR is the prerequisite
- **Future WASM backend** — Statement IR is the prerequisite

---

## Related Documents

**Implemented (inform design):**
- [m-codegen-v2-flat-output.md](design_docs/implemented/v0_5_9/m-codegen-v2-flat-output.md) — Block IR origin story
- [m-codegen-v2-sprint-plan.md](design_docs/implemented/v0_5_9/m-codegen-v2-sprint-plan.md) — Sprint model we follow
- [codegen-bug-pattern-analysis.md](design_docs/implemented/v0_5_10/codegen-bug-pattern-analysis.md) — Why switch cases proliferate

**Planned (check for overlap):**
- [m-codegen-sustainability.md](design_docs/planned/v0_10_0/m-codegen-sustainability.md) — Phase 0 (DONE)
- [m-codegen-ir-strategy.md](design_docs/planned/v0_10_0/m-codegen-ir-strategy.md) — Superseded by this doc
- [m-codegen-v3-binding-hoisting.md](design_docs/planned/v0_10_0/m-codegen-v3-binding-hoisting.md) — Subsumed by Phase 3

---

## Future Work

- **Production Rust backend** (v0.11.0) — Statement IR enables this
- **WASM backend** — Statement IR enables this
- **Target-agnostic builtin specs** — Generalize GoCodegenSpec → BuiltinCodegenSpec
- **Decision tree optimization** — Column-selection heuristics for deep nested patterns
- **IR visualization** — Debug tool to inspect Decision/Type/Statement IR

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18

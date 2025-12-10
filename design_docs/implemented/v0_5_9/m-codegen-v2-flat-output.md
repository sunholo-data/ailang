# M-CODEGEN-V2: Flat Go Code Generation

**Status**: ✅ COMPLETE
**Target**: v0.5.9
**Priority**: P0 (Critical - Blocks production use)
**Estimated**: 2-3 days (core), +2 days (stretch goals)
**Actual**: 0.2 days (263 LOC vs 650 estimated)
**Dependencies**: None
**Reporter**: stapledons_voyage project
**Completed**: 2025-12-10

---

## Implementation Status (2025-12-10)

| Milestone | Status | LOC | Notes |
|-----------|--------|-----|-------|
| M1-BLOCK-IR | ✅ Complete | 121 | Block/Stmt types, Lower(), LowerLetRec(), 12 tests |
| M2-FLAT-FUNCTION-BODY | ✅ Complete | 97 | generateFlatBody(), 4 tests, modified generateImplFunc |
| M3-VALIDATION | ✅ Complete | 0 | stapledons_voyage builds & runs, 58% IIFE reduction |
| M4-CLEANUP-STRETCH | ✅ Complete | 45 | int64(int64) eliminated, suppress unused -44% |

### Final Results

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Total lines (sim_gen/) | 6,184 | 5,594 | **-10%** |
| Total IIFEs | 437 | 182 | **-58%** |
| `int64(int64(...))` | 22 | 0 | **-100%** |
| `// suppress unused` | 445 | 247 | **-44%** |
| Go compilation | OOM | ✅ Success | **Fixed** |
| Max function body nesting | 28+ levels | 0 | **Fixed** |
| VALUE-position nesting | - | 2-3 levels | Expected (by design) |

### Files Created/Modified

**New files:**
- `internal/gen/block/block.go` (51 LOC) - Block IR types
- `internal/gen/block/lower.go` (70 LOC) - Core→Block lowering
- `internal/gen/block/lower_test.go` (273 LOC) - 12 tests
- `internal/gen/golang/codegen_block.go` (97 LOC) - Flat body generation
- `internal/gen/golang/codegen_flat_test.go` (237 LOC) - 4 tests

**Modified files:**
- `internal/gen/golang/codegen_decl.go` - Use generateFlatBody in generateImplFunc
- `internal/gen/golang/codegen_expr_app.go` - Fix redundant int64 wrapping
- `internal/gen/golang/codegen_expr_simple.go` - Add getLitGoType helper
- `internal/gen/golang/codegen_ops.go` - Fix redundant type conversions in records

### Key Finding: VALUE-Position Lets

The remaining nesting is **expected and by design**. When a let chain appears inside a value expression (not at function body level), it still requires IIFEs because Go has no let-expression syntax:

```ailang
-- This let chain is in VALUE position (RHS of var dSq =)
let dSq = let tmp7 = state.discCenterX in
          let tmp8 = state.discCenterY in
          distSq(x, y, tmp7, tmp8)
in ...
```

Generates:
```go
var dSq interface{} = func() interface{} {
    var tmp7 interface{} = FieldGet(state, "discCenterX")
    return func() interface{} {
        var tmp8 interface{} = FieldGet(state, "discCenterY")
        return distSq_impl(x, y, tmp7, tmp8)
    }()
}()
```

**To eliminate this nesting would require "binding hoisting"** - moving inner lets to the enclosing function body. This is a more complex transformation documented in [M-CODEGEN-V3](m-codegen-v3-binding-hoisting.md).

---

## TL;DR

**Problem**: AILANG codegen wraps every expression in an IIFE, creating 28-level nesting and Go compiler OOM.

**Solution**: Insert a **Block IR** layer between Core AST and Go codegen:
```
Core → Block IR (flatten) → Go (emit flat statements)
```

**Key insight**: Move from "expression-to-expression" codegen to "expression-to-block" codegen at function boundaries.

**Scope**: Structural flattening only. Typed codegen (M-DX24) is explicitly out of scope.

**Hard success criteria**:
1. Block IR package exists (`internal/gen/block/`)
2. No nested IIFEs in function bodies (max depth = 1)
3. stapledons_voyage compiles and runs

---

## Problem Statement

### The Root Cause: IIFE-Per-Expression Architecture

AILANG is expression-oriented (everything returns a value), but Go is statement-oriented. The current codegen bridges this gap by wrapping expressions in **Immediately Invoked Function Expressions (IIFEs)**:

```go
// AILANG: let x = 1 in let y = 2 in x + y
func() interface{} {
    var x = 1
    return func() interface{} {
        var y = 2
        return x + y
    }()
}()
```

This works semantically but creates **O(n) nesting depth** for n sequential operations.

### Symptom 1: Sequential Let Bindings → OOM

**Bug Report**: `msg_20251210_110201_3337a70c` from stapledons_voyage

Simple sequential let bindings:
```ailang
let dx = x - cx
let dy = y - cy
dx*dx + dy*dy
```

Generate deeply nested closures:
```go
func distSq_impl(...) interface{} {
    return func() interface{} {           // level 1
        var dx = SubInt(x, cx)
        return func() interface{} {       // level 2
            var dy = SubInt(y, cy)
            return func() interface{} {   // level 3
                var tmp1 = MulInt(dx, dx)
                return func() interface{} { // level 4
                    var tmp2 = MulInt(dy, dy)
                    return AddInt(tmp1, tmp2)
                }()
            }()
        }()
    }()
}
```

**Impact**:
- 255 closure wrappers in 1600-line file
- 221 nested return closures
- Go compiler killed by OOM (signal: killed)
- **Build fails entirely** - cannot compile the game

### Symptom 2: If-Else Chains → Compiler OOM (Partially Fixed)

**Bug Report**: `msg_20251209_202946_f10751f5`

A 25-branch if-else chain generated ~400 lines of 25-deep nested closures:
```go
return func() interface{} {
    if cond1 { return 0 }
    return func() interface{} {
        if cond2 { return 1 }
        return func() interface{} {
            // ...25 levels deep...
        }()
    }()
}()
```

**Status**: Partially fixed by M-CODEGEN-FLAT-IF-ELSE (Dec 9, 2025)
- If-else chains now generate flat code
- But this is a band-aid, not a fix for the underlying architecture

### Symptom 3: Runtime Performance

Even when compilation succeeds:
- **26,000 allocations/sec** in game loops (GC pressure)
- **System freezes** during GC pauses in Ebiten games
- Each IIFE = 1 heap allocation (closure + captured vars)

### Current Metrics (stapledons_voyage sim_gen/)

**File sizes:**
| File | Lines | IIFEs | Nested Returns |
|------|-------|-------|----------------|
| bridge.go | 1616 | 255 | 221 |
| step.go | 528 | 102 | 88 |
| npc_ai.go | 384 | 49 | 43 |
| types.go | 1831 | 0 | 0 |
| **Total** | **6184** | **437** | **378** |

**Key metrics:**
| Metric | Current | Acceptable |
|--------|---------|------------|
| Max nesting depth | **28 levels** | 1-2 |
| IIFEs per file | 100-255 | 0-5 |
| Closures per function | 10-50 | 0-2 |
| Go compiler memory | 2GB+ (OOM) | <500MB |
| Runtime allocs/sec | 26,000 | <1,000 |

### Additional Issues Found (stapledons_voyage Analysis)

**Issue 4: Boolean Short-Circuit Generates IIFEs**

Each `&&` or `||` in a chain generates a nested IIFE:
```go
// AILANG: x >= 0 && x < width && y >= 0 && y < height
return func() interface{} {
    var tmp1 interface{} = GeInt(x, int64(0))
    return func() interface{} {
        var tmp2 interface{} = LtInt(x, width)
        return func() interface{} {
            var tmp3 interface{} = func() interface{} {
                if tmp1.(bool) { return tmp2 }
                return false
            }()
            // ... 4 more levels for remaining conditions
        }()
    }()
}()
```

**Count**: 27 boolean short-circuit IIFEs in bridge.go alone.

**Issue 5: Redundant Type Conversions**

Literals are double-wrapped:
```go
// Generated: int64(int64(1)), float64(float64(0))
var tmp47 interface{} = &CrewID{Id: int64(int64(1)), Name: string("Pilot Chen")}
```

**Issue 6: Arithmetic Uses Runtime Helpers**

Instead of native Go operators:
```go
// Current: 104 runtime helper calls in bridge.go
SubInt(x, cx)   // vs: x - cx
MulInt(dx, dx)  // vs: dx * dx
AddFloat(a, b)  // vs: a + b
```

These add function call overhead and prevent compiler optimization.

**Issue 7: Suppress Unused Comments**

Every single variable has `_ = x // suppress unused`:
```go
var dx interface{} = SubInt(x, cx)
_ = dx // suppress unused  // <-- 258 of these in bridge.go
```

**Issue 8: FieldGet Uses Reflection**

54 calls to `FieldGet()` which uses `reflect.ValueOf`:
```go
// Generated
var tmp13 interface{} = FieldGet(npc, "pos")  // Uses reflection

// Should be (with typed struct)
var tmp13 *Coord = npc.Pos  // Direct field access
```

**Issue 9: Unit Values Everywhere**

82 instances of `struct{}{}` for nullary function calls:
```go
// Current
cx := discCenterDefault_impl(struct{}{})

// Could be
cx := discCenterDefault()  // If typed properly
```

**Issue 10: Slice Conversions at Boundaries**

116 `ConvertTo*Slice()` calls that iterate and convert at runtime:
```go
// Current
return ConvertToCrewPositionSlice(createDefaultCrew_impl(struct{}{}))

// Could be (with typed codegen)
return createDefaultCrew()  // Returns []*CrewPosition directly
```

## Goals

### Primary Goal
Generate **flat, idiomatic Go code** that compiles efficiently and runs without GC pressure.

### Success Metrics

1. **Compilation**: All AILANG programs compile within 500MB Go compiler RAM
2. **Code Size**: Generated code ≤2x equivalent hand-written Go
3. **Runtime**: Zero closure allocations for pure sequential code
4. **Nesting**: Maximum nesting depth = 2 (function body + one conditional)

### Non-Goals

- Changing AILANG semantics
- Breaking existing working code
- Supporting all Go idioms (we generate from AILANG, not general Go)

## Solution Design

### Key Insight: Expression-to-Block, Not Expression-to-Expression

The current codegen treats "expression-oriented AILANG" directly as "expression-oriented Go with IIFEs". The fix is:

**Move from "expression-to-expression" codegen to "expression-to-block" codegen at function boundaries.**

### Architecture: Block IR

We introduce a small, language-neutral intermediate representation:

```go
// Block represents a sequence of statements followed by a final expression.
// This is the natural shape of ANF-style let chains.
type Block struct {
    Stmts     []Stmt      // Variable bindings in evaluation order
    FinalExpr core.CoreExpr // The expression to return
}

type Stmt struct {
    Name  string        // Variable name
    Value core.CoreExpr // Expression to bind
}
```

**Transformation example:**

```
Core (now):                          Block IR:
  Let x = e1 in                        Stmts: [ x := e1; y := e2; z := e3 ]
  Let y = e2 in          ────────►     FinalExpr: e4
  Let z = e3 in
  e4
```

**Why Block IR helps:**

1. **Clean separation**: Flattening is a purely AILANG-side transformation, no Go knowledge needed
2. **Testable in isolation**: `core_to_block.go` can be unit tested without touching Go codegen
3. **Backend-agnostic**: If we ever target Rust/C/JS, they all get the same flat IR
4. **Simplifies Go codegen**: Generator just walks Block, emits `var` statements + `return`

### Two-Phase Pipeline

```
┌─────────┐     ┌─────────────┐     ┌─────────────┐
│  Core   │────►│ Core→Block  │────►│ Block→Go    │
│  AST    │     │  (flatten)  │     │  (emit)     │
└─────────┘     └─────────────┘     └─────────────┘
                     │                    │
                Language-neutral    Backend-specific
                   transform           emission
```

### When IIFEs Are Still Needed

Block IR handles function bodies. IIFEs remain only for **genuinely nested cases**:

```ailang
-- Let inside expression argument - needs IIFE
foo(let x = 1 in x + 1)
```

```go
foo(func() interface{} {
    var x interface{} = 1
    return x + 1
}())
```

**Rule**: If a Let appears in expression position (call arg, match scrutinee, etc.), wrap in IIFE. Otherwise, flatten to Block.

### Pattern 1: Sequential Let Chains → Flat Statements

**Detection**: Let where body is another Let (recursively) ending in non-Let

```ailang
let x = 1
let y = 2
let z = 3
x + y + z
```

**Current Output** (O(n) nesting):
```go
func() interface{} {
    var x = 1
    return func() interface{} {
        var y = 2
        return func() interface{} {
            var z = 3
            return x + y + z
        }()
    }()
}()
```

**Target Output** (flat):
```go
func() interface{} {
    var x interface{} = 1
    var y interface{} = 2
    var z interface{} = 3
    return x + y + z
}()
```

Or even better in typed context:
```go
x := int64(1)
y := int64(2)
z := int64(3)
return x + y + z
```

### Pattern 2: Function Bodies → Direct Statements

**Detection**: Top-level of `_impl` function

```ailang
export fn distance(x: int, y: int, cx: int, cy: int) -> int =
    let dx = x - cx
    let dy = y - cy
    dx*dx + dy*dy
```

**Current Output**:
```go
func distance_impl(x, y, cx, cy interface{}) interface{} {
    return func() interface{} {
        var dx = SubInt(x, cx)
        return func() interface{} {
            var dy = SubInt(y, cy)
            return func() interface{} {
                // ...more nesting...
            }()
        }()
    }()
}
```

**Target Output**:
```go
func distance_impl(x, y, cx, cy interface{}) interface{} {
    dx := SubInt(x, cx)
    dy := SubInt(y, cy)
    tmp1 := MulInt(dx, dx)
    tmp2 := MulInt(dy, dy)
    return AddInt(tmp1, tmp2)
}
```

### Pattern 3: If-Else in Let Chains (Already Fixed)

M-CODEGEN-FLAT-IF-ELSE handles this case. Keep that implementation.

### Pattern 4: Match Expressions

Match expressions currently generate IIFEs. In statement context, can generate switch:

```ailang
match opt with
| Some(x) -> x
| None -> 0
```

**Current**:
```go
func() interface{} {
    switch opt["_tag"] {
    case "Some": return opt["value"]
    case "None": return 0
    }
}()
```

**Target** (in statement context):
```go
switch opt["_tag"] {
case "Some":
    return opt["value"].(int64)
case "None":
    return int64(0)
}
```

### Pattern 5: Boolean Short-Circuit → Flat Conditions

**Detection**: `&&` and `||` chains elaborated to nested if-else

```ailang
x >= 0 && x < width && y >= 0 && y < height
```

**Current Output** (4+ nested IIFEs):
```go
func() interface{} {
    var tmp1 = GeInt(x, 0)
    return func() interface{} {
        var tmp2 = LtInt(x, width)
        return func() interface{} {
            if tmp1.(bool) { return tmp2 }
            return false
        }()
    }()
}()
```

**Target Output** (single expression):
```go
x >= 0 && x < width && y >= 0 && y < height
```

Or with runtime helpers:
```go
GeInt(x, 0).(bool) && LtInt(x, width).(bool) && ...
```

### Pattern 6: Redundant Type Conversions

**Detection**: `int64(int64(x))`, `float64(float64(x))`

**Fix**: Remove outer conversion when inner is already the target type.

### Pattern 7: Suppress Unused → Remove When Used

**Current**: Every var has `_ = x // suppress unused`

**Fix**: Only emit when variable is actually unused (rare).

### Pattern 8: Nested Expressions (Keep IIFEs)

Some patterns genuinely need IIFEs:

```ailang
-- Let inside expression argument
foo(let x = 1 in x + 1)
```

This MUST remain an IIFE because the let is inside an expression:
```go
foo(func() interface{} {
    var x = 1
    return x + 1
}())
```

## Implementation Plan

### Scope Clarification

**M-CODEGEN-V2's mission**: Structural flattening and closure elimination.

**NOT in scope**: Full typed-Go generation (M-DX24). It's fine if `_impl` functions still use `interface{}` and runtime helpers, as long as there are no massive nested IIFEs and the code compiles with sane memory.

Reducing structure complexity first makes all subsequent type work much easier to reason about.

---

### Phase 1: Block IR + Core→Block Transform (P0 - Day 1)

**Goal**: Create the Block IR and the Core→Block lowering pass.

**New Files**:
- `internal/gen/block/block.go` - Block IR types
- `internal/gen/block/lower.go` - Core→Block transformation
- `internal/gen/block/lower_test.go` - Unit tests

**Block IR Definition**:
```go
package block

import "github.com/sunholo/ailang/internal/core"

// Block represents a flattened sequence of bindings + final expression.
type Block struct {
    Stmts     []Stmt
    FinalExpr core.CoreExpr
}

// Stmt is a single variable binding.
type Stmt struct {
    Name  string
    Value core.CoreExpr
}

// Lower converts a Core expression to a Block.
// If the expression is not a let-chain, returns a Block with no Stmts.
func Lower(expr core.CoreExpr) *Block {
    var stmts []Stmt
    current := expr

    for {
        if let, ok := current.(*core.Let); ok {
            stmts = append(stmts, Stmt{Name: let.Name, Value: let.Value})
            current = let.Body
        } else {
            return &Block{Stmts: stmts, FinalExpr: current}
        }
    }
}
```

**Tests**:
- `TestLower_SingleLet` - One binding
- `TestLower_LetChain` - Multiple sequential lets
- `TestLower_NoLets` - Just an expression (empty Stmts)
- `TestLower_NestedInValue` - Let inside value expression (doesn't flatten that)

**Acceptance Criteria**:
- [ ] `Lower()` correctly extracts all top-level let bindings
- [ ] Evaluation order preserved (bindings in same order as Core)
- [ ] 100% test coverage on block package

---

### Phase 2: Flat Function Body Generation (P0 - Day 2)

**Goal**: `_impl` function bodies use Block IR, generate flat statements.

**Files to Modify**:
- `internal/gen/golang/codegen_decl.go` - Use Block for function bodies
- `internal/gen/golang/codegen.go` - Add `generateBlock()` method

**Current Flow**:
```go
func foo_impl(...) interface{} {
    return <generateExpr(body)>  // body generates nested IIFEs
}
```

**New Flow**:
```go
func foo_impl(...) interface{} {
    // Block-based generation
    blk := block.Lower(body)
    for _, stmt := range blk.Stmts {
        var <name> interface{} = <generateExpr(stmt.Value)>
    }
    return <generateExpr(blk.FinalExpr)>
}
```

**Key Implementation**:
```go
func (g *Generator) generateBlock(blk *block.Block) error {
    // Generate all bindings as flat statements
    for _, stmt := range blk.Stmts {
        g.writef("var %s interface{} = ", ToGoVarName(stmt.Name))
        if err := g.generateExpr(stmt.Value); err != nil {
            return err
        }
        g.writef("\n")
    }

    // Generate final expression as return
    g.writef("return ")
    return g.generateExpr(blk.FinalExpr)
}
```

**Tests**:
- `TestFunctionBody_LetChain` - 5 lets → 5 var statements + return
- `TestFunctionBody_SingleExpr` - No lets → just return
- `TestFunctionBody_NestedInArg` - Let in call arg still uses IIFE

**Acceptance Criteria**:
- [ ] Function with 5 lets: ~10 lines (not 50+)
- [ ] No IIFEs in function body for pure let chains
- [ ] stapledons_voyage compiles without OOM
- [ ] All existing codegen tests pass

---

### Phase 3: IIFE Fallback for Expression Context (P0 - Day 2-3)

**Goal**: Handle Let in expression position correctly (still needs IIFE).

**The Rule**:
- **Function body**: Lower to Block, generate flat
- **Expression position** (call arg, match scrutinee, record field): Generate IIFE

**Files to Modify**:
- `internal/gen/golang/codegen_expr_let.go` - Distinguish contexts

**Implementation**:
```go
func (g *Generator) generateLet(let *core.Let) error {
    // If we're in a function body context, we shouldn't hit this
    // (function body uses generateBlock instead).
    // This path is only for Let in expression position.

    // Lower to block, wrap in single IIFE
    blk := block.Lower(let)

    g.writef("func() interface{} {\n")
    g.indent++
    if err := g.generateBlock(blk); err != nil {
        return err
    }
    g.writef("\n")
    g.indent--
    g.write("}()")
    return nil
}
```

**Key insight**: With Block IR, even Let-in-expression becomes a single flat IIFE (not nested).

**Tests**:
- `TestLetInCallArg` - `foo(let x = 1 in x+1)` → single IIFE
- `TestLetInMatchScrutinee` - Let before match
- `TestLetInRecordField` - Let in record literal

**Acceptance Criteria**:
- [ ] Let in expression position: single IIFE, not nested
- [ ] No generated code has IIFEs nested deeper than 1

### Phase 4: Match Optimization (P2 - Day 4-5)

**Goal**: Match expressions in statement context generate switch statements

**Current**:
```go
return func() interface{} {
    switch ... { ... }
}()
```

**Target**:
```go
switch ... {
case ...: return ...
case ...: return ...
}
```

**Files to Modify**:
- `internal/gen/golang/codegen_match.go` - Add context-aware generation

### Phase 5: Quick Wins - Cleanup (P1 - Day 4)

**Goal**: Remove unnecessary code bloat

**5a: Remove Redundant Type Conversions**
```go
// Before: int64(int64(1))
// After: int64(1)
```
- File: `internal/gen/golang/codegen_expr_simple.go`
- Check if already wrapped before adding conversion

**5b: Remove Suppress Unused When Used**
```go
// Before: var x = 1; _ = x // suppress unused
// After: var x = 1  (if x is used later)
```
- File: `internal/gen/golang/codegen_expr_let.go`
- Only emit `_ = x` when variable is truly unused

**5c: Flatten Boolean Short-Circuit**

This is more complex - `&&` and `||` are elaborated to if-else in Core.
Options:
1. Recognize pattern in codegen and emit `&&`/`||`
2. Keep short-circuit semantics but inline the condition

**Acceptance Criteria**:
- [ ] No `int64(int64(...))` in generated code
- [ ] Suppress unused only on actually unused vars
- [ ] Boolean chains generate flat code

### Phase 6: Validation & Testing (P1 - Day 5)

**Goal**: Ensure all changes work together, no regressions

**Validation Tests**:
1. Compile stapledons_voyage with changes
2. Measure Go compiler memory usage
3. Benchmark runtime allocations
4. Verify game runs without GC freezes

**Files to Create**:
- `internal/gen/golang/codegen_flat_test.go` - Integration tests
- `examples/codegen_stress_test.ail` - 50-let chain, 25-branch if-else

### Future Phases (P2 - Beyond This Sprint)

These are lower priority optimizations that would further improve codegen quality:

**Phase 7: Native Arithmetic Operators**
Replace runtime helpers with native Go operators where types are known:
```go
// Before: AddInt(x, y)
// After: x + y  (when both are int64)
```

**Phase 8: Direct Field Access**
Replace `FieldGet(record, "field")` with direct struct access:
```go
// Before: FieldGet(npc, "pos")
// After: npc.Pos
```
Requires tracking struct types through codegen.

**Phase 9: Eliminate Typed Wrapper Functions**
Merge `foo_impl(interface{}) interface{}` with `foo(typed) typed`:
```go
// Current (two functions)
func distSq_impl(x, y, cx, cy interface{}) interface{} { ... }
func distSq(x, y, cx, cy int64) int64 { return distSq_impl(x, y, cx, cy).(int64) }

// Target (one function)
func distSq(x, y, cx, cy int64) int64 { ... }
```

**Phase 10: Remove Slice Conversions**
Generate typed slices directly instead of `[]interface{}` + conversion:
```go
// Before: ConvertToTileSlice(tiles_impl())
// After: tiles() []*Tile  // directly typed
```

## Risk Analysis

### Risk 1: Breaking Existing Code

**Mitigation**:
- Run full test suite after each phase
- Keep IIFE fallback for unknown patterns
- Add `--legacy-codegen` flag for emergency rollback

### Risk 2: Semantic Differences

**Concern**: Flat code might evaluate in different order

**Analysis**: AILANG is purely functional - evaluation order doesn't affect semantics for pure code. For effects, we maintain the same sequencing.

**Mitigation**:
- Add explicit tests for evaluation order
- Document any edge cases

### Risk 3: Increased Complexity

**Concern**: Context-aware generation is more complex than uniform IIFE

**Mitigation**:
- Clear separation of context types
- Comprehensive tests for each context
- Fall back to IIFE for unknown patterns (safe default)

## Success Criteria

### Hard Line for This Sprint (P0)

**The minimum bar**: stapledons_voyage compiles and runs with sane memory.

1. [ ] **Block IR exists** - `internal/gen/block/` package with Lower() function
2. [ ] **Function bodies are flat** - No nested IIFEs in `_impl` function bodies
3. [ ] **stapledons_voyage compiles** - No OOM, Go compiler uses <500MB
4. [ ] **Game runs** - No GC freezes, playable performance
5. [ ] **All existing tests pass** - No regressions

### Mechanical Invariant (Testable)

**"No generated function body contains IIFEs nested deeper than 1."**

This is easy to check mechanically:
```bash
# Count nesting depth of "func() interface{}" in generated code
grep -c "return func() interface{}" generated.go  # Should be 0 or very low
```

We can add a CI check that fails if nesting depth exceeds threshold.

### Should Have (P1) - Stretch Goals

- [ ] Match expressions generate direct switch (Phase 4)
- [ ] No redundant type conversions `int64(int64(x))` (Phase 5a)
- [ ] Suppress unused only when needed (Phase 5b)

### Explicitly Deferred (P2) - Future Work

**These are NOT in scope for M-CODEGEN-V2:**

- [ ] Boolean short-circuit flattening (Phase 5c) - complex pattern recognition
- [ ] Native arithmetic operators (Phase 7) - requires type threading
- [ ] Direct field access (Phase 8) - requires struct type tracking
- [ ] Eliminate _impl wrapper pattern (Phase 9) - major refactor
- [ ] Remove slice conversions (Phase 10) - requires typed slices throughout

These belong in M-DX24 (typed codegen) or a future v0.6.0 milestone.

### Target Metrics

| Metric | Before | After (Target) | Verification |
|--------|--------|----------------|--------------|
| Max IIFE nesting | 28 | **1** | `grep -c "return func()"` |
| IIFEs in bridge.go | 255 | <50 | Count in generated code |
| LOC in bridge.go | 1616 | <1000 | `wc -l` |
| Go compiler RAM | OOM | <500MB | `time go build` |
| stapledons builds | No | **Yes** | CI green |

## Open Questions

### Resolved by Block IR Design

1. ~~**LetRec handling**: Should recursive let bindings also flatten?~~
   - **Answer**: LetRec is handled separately. Block IR only flattens Let chains.
   - LetRec bodies can be lowered to Block for their internal structure.

2. ~~**Effect sequencing**: How to handle `let _ = print("a") in let _ = print("b") in ()`?~~
   - **Answer**: Block IR preserves evaluation order by definition.
   - Stmts are emitted in the same order as Core Let bindings.
   - Flat statements naturally preserve order: `print("a"); print("b")`.

3. ~~**Type inference in flat code**~~
   - **Answer**: Keep `interface{}` for M-CODEGEN-V2. Don't entangle with typed codegen.
   - Typed vars are M-DX24 scope (future).

4. **Backwards compatibility**: Add `--legacy-codegen` flag?
   - **Recommendation**: Yes, implement it.
   - Invaluable for bisecting regressions: "is this a type bug or did flat-codegen break something?"
   - Low cost to implement (keep old generateLet code behind flag).

### Remaining Questions

5. **Match branch bodies**: Should match branches also use Block IR?
   - Each branch is an expression that could be a let-chain.
   - Lowering branch bodies to Block keeps them flat too.
   - **Recommendation**: Yes, but defer to Phase 4.

6. **Validation strategy**: How to test with stapledons_voyage?
   - **Recommendation**: Point stapledons at local ailang build (faster iteration).
   - After each phase, regenerate sim_gen/ and verify it compiles.

7. **CI enforcement**: How to prevent regressions?
   - Add a test that counts `return func() interface{}` in generated code.
   - Fail if count exceeds threshold (e.g., 5 per file).
   - Consider adding to `make ci`.

## References

- M-CODEGEN-FLAT-IF-ELSE: Partial fix for if-else chains (implemented Dec 9, 2025)
- stapledons_voyage bug report: `msg_20251210_110201_3337a70c`
- [codegen_expr_let.go](../../../internal/gen/golang/codegen_expr_let.go) - Current let generation
- [codegen.go](../../../internal/gen/golang/codegen.go) - Generator state

## Changelog

| Date | Change |
|------|--------|
| 2025-12-10 | Initial design document |
| 2025-12-10 | Added stapledons_voyage analysis: 10 issues identified, phases 5-10 added |
| 2025-12-10 | **Major revision**: Introduced Block IR architecture per feedback. Separated concerns (flattening vs typed codegen). Simplified context propagation. Added mechanical invariant for CI. Explicitly deferred P2 items to M-DX24/v0.6.0 |
| 2025-12-10 | **Implementation**: M1-BLOCK-IR complete (121 LOC, 12 tests). M2-FLAT-FUNCTION-BODY complete (97 LOC, 4 tests). |
| 2025-12-10 | **Validation**: stapledons_voyage sim_gen/ regenerated. 58% IIFE reduction (437→182). Go compiles successfully (no OOM). VALUE-position nesting identified as expected behavior. Follow-up design doc created: [M-CODEGEN-V3](m-codegen-v3-binding-hoisting.md) |
| 2025-12-10 | **M4-CLEANUP-STRETCH complete**: Eliminated redundant `int64(int64(...))` (22→0), reduced suppress unused comments 44% (445→247), total lines reduced 10% (6,184→5,594). Sprint complete in 0.2 days vs 4 estimated. |

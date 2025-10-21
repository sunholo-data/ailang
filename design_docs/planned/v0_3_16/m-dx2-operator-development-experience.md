# M-DX2: Operator Development Experience

**Status**: Planned
**Target**: v0.3.16
**Priority**: P1 (Medium-High)
**Estimated**: 1 working day (8 hours)
**Dependencies**: M-DX1 (Builtin Registry & Type DSL) - Completed in v0.3.10

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Developer tooling, doesn't affect language syntax |
| Preserve Semantic Clarity | Positive | +1 | Better observability → clearer understanding of types/lowering |
| Increase Determinism | Positive | +1 | Type-guided lowering removes heuristic ambiguity |
| Lower Token Cost | Positive | +1 | Faster feature dev → more features → better compression |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

**Rationale:** While this doesn't directly affect end-user AILANG code, it dramatically reduces the entropy of *adding new operators* — which enables faster language evolution toward AI-first goals. The ++ operator fix took 2h of ANF spelunking; this makes similar features 30-60 minute tasks.

## Problem Statement

**Discovered during**: List concatenation operator (`++`) implementation (October 2025)

**Current State:**
- Adding polymorphic operators (like `++` for String/List) takes **~2 hours** due to:
  - ANF opacity: Must manually trace `Var` bindings to find actual types
  - No type information flow from typechecker → lowering pass
  - Lack of Core IR helpers (must re-learn ANF patterns each time)
  - Runtime errors are cryptic interface conversion panics
  - No observability tools to inspect ANF or type annotations

**Pain Points:**
1. **Type-guided lowering is guesswork**: Lowering pass has to inspect Core IR shapes to decide `_str_concat` vs `_list_concat`, but lacks principal type info from typechecker
2. **ANF is a black box**: Developers must manually chase `Var` bindings through nested `Let` expressions
3. **No debugging tools**: Can't inspect ANF structure or type annotations without `fmt.Printf` spelunking
4. **Error messages unhelpful**: Wrong builtin variant → `interface conversion panic` (no context)
5. **Tribal knowledge**: ANF patterns & operator wiring are not documented

**Impact:**
- **Who**: AILANG contributors (AI agents + humans) adding operators/builtins
- **Significance**: 2h task becomes 30-60min with proper DX (67-75% time savings)
- **Example**: The `++` fix required tracing ANF bindings, reading lowering code, and testing multiple variants

**Metrics (from ++ operator implementation):**
- Time spent: ~2 hours
- Root cause: ~45 minutes (chasing ANF bindings to find types)
- Implementation: ~30 minutes (once design was clear)
- Testing: ~45 minutes (including fixing type errors)

## Goals

**Primary Goal:** Reduce polymorphic operator development time from 2h to 30-60min (67-75% improvement)

**Success Metrics:**
- Type-guided lowering eliminates ANF binding traversal
- Core IR helpers reduce boilerplate by 50%
- Debug tools (`ailang debug ast`) provide instant ANF/type visibility
- Runtime errors include actionable diagnostic messages
- Documentation enables onboarding in <30 minutes

## Solution Design

### Overview

Build on M-DX1's success (builtin registry + Type DSL) by addressing the next layer of friction: **type information flow** and **observability**. The core insight is that we already have all the data (types from inference, ANF from lowering), but it's not connected or visible.

**Strategy:** Add minimal plumbing to connect existing systems + lightweight CLI observability.

### Architecture

**Five targeted improvements:**

1. **Type-guided lowering**: Plumb inferred types from typechecker → lowering pass
2. **Core IR helpers**: Add `ResolveValue()` and `IsListValue()` to avoid manual ANF traversal
3. **Debug CLI**: Add `ailang debug ast --show-types` to inspect ANF + type annotations
4. **Better runtime errors**: Wrap builtin casts with type checks + helpful messages
5. **Documentation**: Write ANF guide + operator implementation checklist

**Components:**

1. **TypeInfo Map** (typechecker → lowering)
   - Store `map[NodeID]types.Type` during type inference
   - Pass to lowering context via `LoweringContext.TypeInfo`
   - Select builtin variants by type, not ANF shape

2. **Core Helpers** (`internal/core/helpers.go`)
   - `ResolveValue(expr, binds)`: Follow `Var` bindings to terminal value
   - `IsListValue(expr, binds)`: Check if resolved value is a list
   - Used by lowering to simplify type checks

3. **Debug Tool** (`cmd/ailang/debug.go`)
   - `ailang debug ast file.ail --show-types`
   - Prints: Surface AST → Core ANF → Type annotations
   - Re-uses existing pretty-printers + new TypeInfo formatter

4. **Runtime Error Wrapper** (`internal/eval/builtins.go`)
   - Wrap builtin invocations with type assertions
   - On mismatch: `"++ lowering selected concat_String, but received List at runtime"`
   - Include likely causes + fix suggestions

5. **Documentation** (`docs/architecture/`, `docs/guides/`)
   - `ANF.md`: Why ANF exists, how to read it, common patterns
   - `adding-operators.md`: Step-by-step checklist for polymorphic operators

### Implementation Plan

**Phase 1: Type-Guided Lowering** (~2-3 hours)
- [ ] Add `TypeInfo map[int]types.Type` to typechecker
- [ ] Populate during type inference (annotate each AST node)
- [ ] Pass `TypeInfo` to lowering via `LoweringContext`
- [ ] Update `lowerBinaryOp()` to select builtin by type, not shape
- [ ] Test with `++` operator (should work without ANF traversal)

**Phase 2: Core IR Helpers** (~45 minutes)
- [ ] Create `internal/core/helpers.go`
- [ ] Implement `ResolveValue(expr CoreExpr, binds map[string]CoreExpr) CoreExpr`
- [ ] Implement `IsListValue(expr CoreExpr, binds map[string]CoreExpr) bool`
- [ ] Add unit tests (10 cases covering Var chains, nested Lets, literals)
- [ ] Use in lowering pass (simplify existing ANF traversal)

**Phase 3: Debug CLI** (~2-3 hours)
- [ ] Add `cmd/ailang/debug.go` with `debugASTCmd`
- [ ] Implement `--show-types` flag (print TypeInfo alongside nodes)
- [ ] Re-use existing AST printers from pipeline
- [ ] Add TypeInfo formatter (pretty-print types per node)
- [ ] Test with examples: `ailang debug ast examples/list_ops.ail --show-types`

**Phase 4: Error Messages** (~45 minutes)
- [ ] Add `checkBuiltinArgType()` helper in `internal/eval/builtins.go`
- [ ] Wrap String builtin casts: `if !ok { return builtin mismatch error }`
- [ ] Include diagnostic: expected type, received type, likely causes
- [ ] Add test: call `_str_concat` with List → expect helpful error
- [ ] Update error schema if needed

**Phase 5: Documentation** (~1.5-2 hours)
- [ ] Write `docs/architecture/ANF.md` (~45 min)
  - Why ANF? (simplifies eval, explicit sequencing)
  - Reading ANF: Vars, Lets, bindings
  - Common patterns: nested Lets, application chains
- [ ] Write `docs/guides/adding-operators.md` (~45 min)
  - Step-by-step checklist
  - Where to register (op_table, op_lowering, builtins)
  - Testing patterns (hermetic tests, type checks)
  - Pitfalls (polymorphic operators, ANF resolution)
- [ ] Add links from `CLAUDE.md` and `CONTRIBUTING.md` (~15 min)

### Files to Modify/Create

**New files:**
- `internal/core/helpers.go` - Core IR helpers (~50 LOC)
- `internal/core/helpers_test.go` - Tests (~100 LOC)
- `cmd/ailang/debug.go` - Debug CLI (~150 LOC)
- `docs/architecture/ANF.md` - ANF guide (~300 lines)
- `docs/guides/adding-operators.md` - Operator guide (~400 lines)

**Modified files:**
- `internal/types/typechecker.go` - Add TypeInfo map (~20 LOC)
- `internal/pipeline/lowering_context.go` - Pass TypeInfo (~10 LOC)
- `internal/pipeline/op_lowering.go` - Use types instead of shapes (~30 LOC)
- `internal/eval/builtins.go` - Wrap builtin casts (~40 LOC)
- `cmd/ailang/main.go` - Register debug command (~5 LOC)
- `CLAUDE.md` - Link to new docs (~10 LOC)

**Total new code:** ~750 LOC
**Total modified code:** ~115 LOC

## Examples

### Example 1: Type-Guided Lowering

**Before (ANF traversal):**
```go
// internal/pipeline/op_lowering.go
func lowerBinaryOp(op string, left, right CoreExpr, binds map[string]CoreExpr) CoreExpr {
    // Must manually chase Var bindings to find if it's a List
    leftResolved := left
    if v, ok := left.(*VarExpr); ok {
        if val, exists := binds[v.Name]; exists {
            leftResolved = val
        }
    }

    // Check shape to decide builtin
    if _, ok := leftResolved.(*ListLit); ok {
        return &AppExpr{Func: &VarExpr{Name: "_list_concat"}, Args: []CoreExpr{left, right}}
    }
    return &AppExpr{Func: &VarExpr{Name: "_str_concat"}, Args: []CoreExpr{left, right}}
}
```

**After (type-guided):**
```go
// internal/pipeline/op_lowering.go
func (lc *LoweringContext) lowerBinaryOp(op string, left, right CoreExpr, leftID, rightID int) CoreExpr {
    // Look up type from typechecker
    leftType := lc.TypeInfo[leftID]

    // Decide builtin by type, not shape
    switch leftType.(type) {
    case *types.TList:
        return &AppExpr{Func: &VarExpr{Name: "_list_concat"}, Args: []CoreExpr{left, right}}
    case *types.TString:
        return &AppExpr{Func: &VarExpr{Name: "_str_concat"}, Args: []CoreExpr{left, right}}
    default:
        panic(fmt.Sprintf("++ operator not supported for type %v", leftType))
    }
}
```

**Impact:** No more ANF traversal. Type is authoritative source.

### Example 2: Core IR Helpers

**Before (manual Var chasing):**
```go
// internal/pipeline/op_lowering.go
leftResolved := left
for {
    if v, ok := leftResolved.(*VarExpr); ok {
        if val, exists := binds[v.Name]; exists {
            leftResolved = val
        } else {
            break
        }
    } else {
        break
    }
}
```

**After (using helper):**
```go
// internal/pipeline/op_lowering.go
leftResolved := core.ResolveValue(left, binds)
```

**Impact:** One-liner replaces 10-line loop.

### Example 3: Debug CLI

**Command:**
```bash
ailang debug ast examples/list_ops.ail --show-types
```

**Output:**
```
=== Surface AST ===
BinaryExpr (++)
  Left: Ident "xs" [NodeID: 42]
  Right: ListLit [1, 2, 3] [NodeID: 43]

=== Core (ANF) ===
Let _t1 = Var "xs"
Let _t2 = ListLit [IntLit 1, IntLit 2, IntLit 3]
Let _t3 = App (Var "_list_concat") [Var "_t1", Var "_t2"]
In Var "_t3"

=== Type Annotations ===
NodeID 42: List Int
NodeID 43: List Int
NodeID 44 (BinaryExpr): List Int
```

**Impact:** Instant visibility into ANF + types. No more `fmt.Printf` debugging.

### Example 4: Better Runtime Errors

**Before (cryptic panic):**
```
panic: interface conversion: eval.Value is *eval.ListValue, not *eval.StringValue
  at internal/builtins/string.go:42
```

**After (actionable error):**
```
Error: Builtin type mismatch for '_str_concat'
  Expected: String
  Received: List Int

Likely causes:
  - Operator lowering selected wrong builtin variant (_str_concat instead of _list_concat)
  - Missing type-guided lowering for ++ operator
  - Type inference produced incorrect type

Fix:
  - Check internal/pipeline/op_lowering.go operator table
  - Verify TypeInfo is populated for this node
  - Run: ailang debug ast <file> --show-types
```

**Impact:** 5 seconds to understand instead of 5 minutes.

## Success Criteria

- [ ] Type-guided lowering works for `++` operator (no ANF traversal)
- [ ] Core helpers used in at least 3 places in lowering pass
- [ ] `ailang debug ast --show-types` prints ANF + type annotations
- [ ] Runtime builtin mismatch errors include actionable diagnostics
- [ ] `docs/architecture/ANF.md` covers all common patterns
- [ ] `docs/guides/adding-operators.md` has step-by-step checklist
- [ ] New contributor can add `**` (power) operator in <1 hour using guides
- [ ] All tests passing (including new helper tests)
- [ ] Documentation updated (CLAUDE.md, CONTRIBUTING.md)

## Testing Strategy

**Unit tests:**
- `internal/core/helpers_test.go`: 10 cases for ResolveValue/IsListValue
  - Simple Var lookup
  - Nested Var chains (a → b → c → literal)
  - Var not in bindings
  - Non-Var expressions
- `internal/eval/builtins_test.go`: Type mismatch errors
  - Call String builtin with List → expect helpful error
  - Call List builtin with String → expect helpful error

**Integration tests:**
- `internal/pipeline/lowering_test.go`: Type-guided lowering
  - `[1,2] ++ [3,4]` → selects `_list_concat`
  - `"foo" ++ "bar"` → selects `_str_concat`
  - Type annotation drives selection, not shape
- `cmd/ailang/debug_test.go`: CLI output
  - Run `debug ast examples/list_ops.ail --show-types`
  - Assert output contains ANF and type annotations

**Manual testing:**
- Add new operator (`**` for exponentiation) using new guides
  - Should take <1 hour
  - Document any pain points → iterate on guides
- Run `ailang debug ast` on complex examples
  - Verify types are correct
  - Check ANF is readable

## Non-Goals

**Not in this feature:**
- Full IDE integration (LSP, hover, autocomplete) - Out of scope for AI-first language
- Automatic operator polymorphism (multi-dispatch) - Requires type class system (v0.4.0+)
- Runtime type reflection - Deferred to structural reflection milestone
- ANF pretty-printer improvements - Current format is adequate for debugging

**Why deferred:**
- LSP/IDE features contradict AI-first design (see CLAUDE.md)
- Multi-dispatch requires type classes (planned for v0.4.0)
- Reflection needs schema registry (v0.4.0)
- ANF printer works fine, effort better spent elsewhere

## Timeline

**Week 1** (8 hours):
- Days 1-2: Phase 1 (Type-guided lowering) - 2.5h
- Day 2-3: Phase 2 (Core helpers) - 0.75h
- Day 3-4: Phase 3 (Debug CLI) - 2.5h
- Day 4-5: Phase 4 (Error messages) - 0.75h
- Day 5: Phase 5 (Documentation) - 1.5h

**Total: ~8 hours across 1 week**

**Buffer:** 2x multiplier already applied (raw estimate was 4h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| TypeInfo map increases memory | Low | Only used during compilation, freed after lowering |
| Debug CLI output too verbose | Medium | Add `--compact` flag to show only types or only ANF |
| ANF doc gets outdated | Low | Link from error messages so it's used frequently |
| Type-guided lowering breaks existing code | High | Extensive tests + verify all examples still work |

## References

- **Prior art**: M-DX1 Developer Experience (v0.3.10)
  - [Design doc](../implemented/v0_3_10/M-DX1_developer_experience.md)
  - Reduced builtin dev time from 7.5h → 2.5h
- **Trigger**: List concatenation operator fix (October 2025)
  - [Design doc](list-concatenation-operator-fix.md)
  - Highlighted ANF opacity and lack of type info flow
- **Related**: AILANG AI-First DX Philosophy
  - [Example Parity & Vision Alignment](../v0_3_15/example-parity-vision-alignment.md)

## Future Work

**Potential extensions (not in v0.3.16):**

1. **Type-directed code generation** (v0.4.0+)
   - Use TypeInfo to generate optimized code for monomorphic cases
   - Inline builtin calls when types are statically known

2. **Enhanced debug modes** (v0.3.17+)
   - `--trace-lowering`: Show each lowering step
   - `--trace-inference`: Show type unification steps
   - Useful for debugging complex type errors

3. **Operator registry** (v0.4.0+)
   - Central registry like builtin registry
   - Register operator implementations declaratively
   - Automatic multi-dispatch based on type classes

4. **ANF optimizer** (v0.4.0+)
   - Eliminate redundant Lets
   - Inline single-use bindings
   - Currently ANF is unoptimized (doesn't matter for correctness)

---

**Document created**: 2025-10-21
**Last updated**: 2025-10-21

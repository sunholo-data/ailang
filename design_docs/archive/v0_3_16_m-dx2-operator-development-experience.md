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
- `++` lowering uses TypeInfo; code passes with ANF lookups disabled for this operator
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
   - Use stable NodeIDs (assigned during parsing, preserved through passes)
   - Store **principal types** (post-generalization, not intermediate unification states)
   - Pass to lowering context via `LoweringContext.TypeInfo`
   - Select builtin variants by type, not ANF shape
   - **Lifecycle**: Created during typechecking, used during lowering, freed after lowering (no persistence)
   - **Wrapper type** for safety:
     ```go
     type TypeInfo map[NodeID]types.Type
     func (ti TypeInfo) Must(id NodeID) types.Type // friendly error if missing
     ```

2. **Operator Table as Source of Truth**
   - Maintain single operator registry: `op_table[OpConcat].Variants`
   - Each variant: `{Type: types.Type, Builtin: string}`
   - Lowering consults table and picks by type match (no hardcoded builtin names)
   - **Fallback behavior**: If type unknown/ambiguous → clear compile error:
     ```
     "Operator ++ supports String | List; inferred type: <ambiguous>
      Add explicit type annotation to disambiguate."
     ```
   - Never silently choose variant by ANF shape

3. **Core Helpers** (`internal/core/helpers.go`)
   - `ResolveValue(expr, binds)`: Follow `Var` bindings to terminal value
   - `IsListValue(expr, binds)`: Check if resolved value is a list
   - Used by lowering to simplify type checks (when TypeInfo unavailable)
   - **Tested with depth 0-5 var chains** (fuzz-style test)

4. **Debug Tool** (`cmd/ailang/debug.go`)
   - `ailang debug ast <file> [flags]`
   - **Flags**:
     - `--surface` (default) / `--core` - Which AST to show
     - `--show-types` - Add `NodeID: Type` annotations
     - `--node <id>` - Filter to specific subtree
     - `--compact` - No whitespace (for logs/CI)
     - `--limit <n>` - Max nodes to print (default 1000, prevents gigantic dumps)
   - **Legend**: Print one-line type legend when `--show-types` enabled
   - Re-uses existing pretty-printers + new TypeInfo formatter

5. **Runtime Error Helper** (`internal/eval/builtin_errors.go`)
   - Centralized error constructor:
     ```go
     func ArgTypeMismatch(fn string, argIdx int, want, got string, hint string) error
     ```
   - On mismatch: Include expected type, received type, likely causes, fix suggestions
   - **Smart hint**: If type-guided lowering enabled and mismatch occurs → suggest filing bug with `ailang debug ast --show-types` output

6. **Documentation** (`docs/architecture/`, `docs/guides/`)
   - `ANF.md`: Why ANF exists, how to read it, common patterns
   - `adding-operators.md`: Step-by-step checklist for polymorphic operators

### Implementation Plan

**Phase 1: Type-Guided Lowering** (~2-3 hours)
- [ ] Add `TypeInfo` wrapper type with `Must()` helper
- [ ] Ensure NodeIDs are stable (assigned during parsing, preserved through passes)
- [ ] Populate TypeInfo during type inference with **principal types** (post-generalization)
- [ ] Pass `TypeInfo` to lowering via `LoweringContext`
- [ ] Build operator table with variants: `op_table[OpConcat].Variants = [{Type, Builtin}]`
- [ ] Update `lowerBinaryOp()` to:
  - Consult operator table (no hardcoded builtin names)
  - Select variant by type match
  - Emit compile error if type unknown/ambiguous (with helpful message)
- [ ] Delete ANF binding-chase branch from `++` lowering
- [ ] Test: `++` works without ANF traversal (regression guard)

**Phase 2: Core IR Helpers** (~45 minutes)
- [ ] Create `internal/core/helpers.go`
- [ ] Implement `ResolveValue(expr CoreExpr, binds map[string]CoreExpr) CoreExpr`
- [ ] Implement `IsListValue(expr CoreExpr, binds map[string]CoreExpr) bool`
- [ ] Add unit tests:
  - Simple Var lookup
  - Nested Var chains (depth 0-5, fuzz-style test)
  - Var not in bindings
  - Non-Var expressions
- [ ] Use in lowering pass (simplify existing ANF traversal where TypeInfo unavailable)

**Phase 3: Debug CLI** (~2-3 hours)
- [ ] Add `cmd/ailang/debug.go` with `debugASTCmd`
- [ ] Implement flags:
  - `--surface` (default) / `--core` - Which AST to print
  - `--show-types` - Add NodeID: Type annotations
  - `--node <id>` - Filter to subtree
  - `--compact` - No whitespace (for logs)
  - `--limit <n>` - Max nodes (default 1000)
- [ ] Print one-line legend when `--show-types` enabled
- [ ] Re-use existing AST printers from pipeline
- [ ] Add TypeInfo formatter (pretty-print types per node)
- [ ] Test with examples:
  - `ailang debug ast examples/list_ops.ail --show-types`
  - `ailang debug ast examples/factorial.ail --core --compact`

**Phase 4: Error Messages** (~45 minutes)
- [ ] Create `internal/eval/builtin_errors.go`
- [ ] Implement `ArgTypeMismatch(fn, argIdx, want, got, hint)` helper
- [ ] Wrap String builtin casts with type checks
- [ ] Include diagnostic: expected type, received type, likely causes, fix suggestions
- [ ] Add smart hint: If type-guided lowering enabled → suggest filing bug with `ailang debug ast --show-types`
- [ ] Add tests:
  - Call `_str_concat` with List → expect helpful error
  - Call `_list_concat` with String → expect helpful error
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
- `internal/types/typeinfo.go` - TypeInfo wrapper type (~30 LOC)
- `internal/core/helpers.go` - Core IR helpers (~50 LOC)
- `internal/core/helpers_test.go` - Tests (~120 LOC including fuzz-style)
- `internal/eval/builtin_errors.go` - Error helpers (~60 LOC)
- `cmd/ailang/debug.go` - Debug CLI (~200 LOC with all flags)
- `docs/architecture/ANF.md` - ANF guide (~300 lines)
- `docs/guides/adding-operators.md` - Operator guide (~400 lines)

**Modified files:**
- `internal/parser/parser.go` - Ensure stable NodeIDs (~10 LOC)
- `internal/types/typechecker.go` - Populate TypeInfo with principal types (~30 LOC)
- `internal/pipeline/lowering_context.go` - Pass TypeInfo (~10 LOC)
- `internal/pipeline/op_table.go` - Add Variants to operator registry (~40 LOC)
- `internal/pipeline/op_lowering.go` - Type-guided selection, delete ANF chase (~50 LOC)
- `internal/builtins/string.go` - Use ArgTypeMismatch helper (~20 LOC)
- `internal/builtins/list.go` - Use ArgTypeMismatch helper (~20 LOC)
- `cmd/ailang/main.go` - Register debug command (~5 LOC)
- `CLAUDE.md` - Link to new docs (~10 LOC)

**Total new code:** ~860 LOC
**Total modified code:** ~195 LOC

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
    // Look up type from typechecker (principal type)
    leftType := lc.TypeInfo.Must(leftID)

    // Consult operator table (source of truth)
    opDef := lc.OpTable[op]

    // Select variant by type match
    for _, variant := range opDef.Variants {
        if types.Matches(leftType, variant.Type) {
            return &AppExpr{
                Func: &VarExpr{Name: variant.Builtin},
                Args: []CoreExpr{left, right},
            }
        }
    }

    // No variant matched → clear compile error
    return lc.Error(
        "Operator %s supports %s; inferred type: %v\n"+
        "Add explicit type annotation to disambiguate.",
        op, opDef.SupportedTypes(), leftType,
    )
}
```

**Impact:**
- No ANF traversal needed
- Type is authoritative source
- Operator table is single source of truth (no hardcoded builtin names)
- Clear error messages for ambiguous cases

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
- `internal/types/typeinfo_test.go`: TypeInfo wrapper
  - `Must()` returns type when ID exists
  - `Must()` panics with helpful error when ID missing
- `internal/core/helpers_test.go`: ResolveValue/IsListValue
  - Simple Var lookup
  - Nested Var chains (depth 0-5, fuzz-style test)
  - Var not in bindings
  - Non-Var expressions
- `internal/eval/builtin_errors_test.go`: Error formatting
  - `ArgTypeMismatch()` produces helpful messages
  - Smart hint included when type-guided lowering enabled
  - Call String builtin with List → expect helpful error
  - Call List builtin with String → expect helpful error

**Integration tests:**
- `internal/pipeline/lowering_test.go`: Type-guided lowering
  - `[1,2] ++ [3,4]` → selects `_list_concat`
  - `"foo" ++ "bar"` → selects `_str_concat`
  - Type annotation drives selection, not shape
  - **Regression guard**: Remove ANF lookup code, test still passes
- `internal/pipeline/op_table_test.go`: Operator registry
  - All operators have variants
  - Variant types match builtin signatures
- `cmd/ailang/debug_test.go`: CLI output
  - Run `debug ast examples/list_ops.ail --show-types`
  - Assert output contains ANF and type annotations
  - Test `--compact`, `--node`, `--limit` flags

**Stability tests:**
- `internal/parser/nodeid_test.go`: NodeID stability
  - Parse same file twice → NodeIDs identical
  - Parse → lower → NodeIDs unchanged for same nodes

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
| TypeInfo map increases memory | Low | Only used during compilation, freed after lowering; no persistence |
| Debug CLI output too verbose | Medium | Add `--compact` flag + `--limit` (default 1000 nodes) |
| ANF doc gets outdated | Low | Link from error messages so it's used frequently |
| Type-guided lowering breaks existing code | High | Extensive tests + verify all examples still work + regression guards |
| Stale/unstable NodeIDs | Medium | Unit test: parse same file twice → NodeIDs identical; preserve IDs through passes |
| Performance degradation | Low | TypeInfo lookups are O(1); only adds ~100µs to compile time |

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

## Implementation Checklist (Tight & Shippable)

When ready to implement, follow this checklist:

**Phase 1: TypeInfo Plumbing**
- [ ] Add `TypeInfo` wrapper type with `Must()` helper
- [ ] Ensure NodeIDs stable (assigned during parsing, preserved through passes)
- [ ] Store principal types (post-generalization) during type inference
- [ ] Pass TypeInfo into lowering context

**Phase 2: Operator Table & Type-Guided Lowering**
- [ ] Build operator table with variants: `{Type, Builtin}`
- [ ] Switch `++` lowering to type-guided selection (consult table)
- [ ] Delete ANF binding-chase branch
- [ ] Add compile error for unknown/ambiguous types
- [ ] **Critical test**: Verify `++` works with ANF lookups disabled

**Phase 3: Core Helpers**
- [ ] Implement `core.ResolveValue()` and `core.IsListValue()`
- [ ] Add tests (depth 0-5 var chains, fuzz-style)
- [ ] Use in lowering where TypeInfo unavailable

**Phase 4: Debug CLI**
- [ ] Implement `ailang debug ast` with `--show-types`, `--core`, `--node`, `--compact`, `--limit`
- [ ] Print one-line legend when `--show-types` enabled
- [ ] Test with examples

**Phase 5: Error Messages**
- [ ] Create `builtin_errors.go` with `ArgTypeMismatch()` helper
- [ ] Wrap builtin arg casts with type checks
- [ ] Add smart hint (suggest filing bug with debug output)
- [ ] Test: wrong builtin variant → helpful error

**Phase 6: Documentation**
- [ ] Write `docs/architecture/ANF.md`
- [ ] Write `docs/guides/adding-operators.md`
- [ ] Link from `CLAUDE.md` and `CONTRIBUTING.md`

**Phase 7: Regression Guards**
- [ ] Test: `++` lowering fails if ANF lookup code re-added
- [ ] Test: NodeIDs stable across parses
- [ ] Verify all examples still work

---

**Document created**: 2025-10-21
**Last updated**: 2025-10-21

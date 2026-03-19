# M-CONTRACT-ENSURES-CODEGEN-FIX: Fix Ensures Contract Codegen After OpLowering

**Status**: Planned
**Target**: v0.9.4
**Priority**: P1 (blocks CI — test failure on every push)
**Estimated**: 0.5-1 day
**Dependencies**: M-CONTRACT-OPLOWERING-FIX (completed)
**Milestone ID**: M-CONTRACT-ENSURES-CODEGEN-FIX
**Created**: 2026-03-19
**Source**: CI failure in `TestContractViolation_Integration`

---

## Problem Statement

After M-CONTRACT-OPLOWERING-FIX (commit f9f480c7), contract expressions go through OpLowering and `FillOperatorMethods`. This transforms contract predicates from `Intrinsic` nodes to `DictApp` nodes. The **requires** codegen handles this correctly (uses `generateExpr` which produces valid `dict_Ord_Int.Gte(x, 0).(bool)` in the `_impl` context). But the **ensures** codegen breaks in two ways:

### Bug 1: DictApp not handled in `generateEnsuresPredicate`

The ensures codegen in `generateEnsuresPredicate` has a type switch for `*core.Var`, `*core.Lit`, `*core.BinOp`, `*core.Intrinsic`, `*core.App` — but NOT `*core.DictApp`. After OpLowering, contract predicates like `result >= 0` become:

```
DictApp{Dict: "dict_Ord_Int", Method: "Gte", Args: [Var("result"), Lit(0)]}
```

This falls to the `default:` case which calls `generateExpr(expr)` — losing the `result` → `_result` variable substitution.

### Bug 2: `result` undefined in typed wrapper

Because the DictApp falls through to `generateExpr`, the `Var("result")` is emitted as-is (`result`) instead of being substituted with `_result`. In the typed wrapper context, only `_result` is in scope.

### Generated Code (broken)

```go
// In _impl (interface{} context) — WORKS:
if !(dict_Ord_Int.Gte(x, int64(0))).(bool) {  // requires: OK

// In typed wrapper — BROKEN:
if !(dict_Ord_Int.Gte(result, int64(0))) {     // ensures: 2 errors
//    ^-- ! applied to interface{} (no .(bool))
//                      ^-- "result" undefined (should be "_result")
```

### CI Impact

`TestContractViolation_Integration` fails on every push. The test compiles `examples/runnable/contracts/basic.ail` with `--verify-contracts`, then verifies the generated Go code compiles. It has been failing since M-CONTRACT-OPLOWERING-FIX.

---

## Root Cause Analysis

### The Pipeline

```
Contract expr: result >= 0
    ↓ Parser
Intrinsic{OpGe, [Var("result"), Lit(0)]}
    ↓ FillOperatorMethods (pipeline_module_compile.go:312)
BinOp with Method="Gte" set
    ↓ OpLowering (op_lowering.go:105)
DictApp{Dict: "dict_Ord_Int", Method: "Gte", Args: [Var("result"), Lit(0)]}
    ↓ Codegen: generateEnsuresPredicate
??? DictApp case missing → falls to default → generateExpr → broken
```

### Why the DictApp case I added didn't work

I added a `*core.DictApp` case to `generateEnsuresPredicate` (in contracts.go) that maps dict methods back to Go operators. The code compiles but the case is NOT being reached — the expression type hitting `generateEnsuresPredicate` is something else (possibly wrapped in another node type).

**Investigation needed:** Add `fmt.Fprintf` debug to print `%T` of the expression actually passed to `generateEnsuresPredicate`. Possible causes:
1. The DictApp is wrapped in another node (e.g., `Let`, `App` wrapping `DictApp`)
2. The expression is being passed through a different code path entirely
3. The meta lookup key (`funcName`) doesn't match, so ensures checks aren't generated via `generateEnsuresPredicate` at all

---

## Proposed Solutions

### Option A: Fix `generateEnsuresPredicate` to handle DictApp (and any wrapper)

1. Debug what `%T` the expression actually is when reaching `generateEnsuresPredicate`
2. Add cases for any wrapper nodes (Let, etc.) that recursively delegate
3. The `*core.DictApp` case already added (maps `Gte`→`>=`, etc.) is correct once reached

**Pros:** Fixes the immediate problem
**Cons:** Fragile — any future node type change breaks it again

### Option B: Don't OpLower contract expressions

Remove the OpLowering of contract expressions (op_lowering.go:89-111). Keep them as `Intrinsic` nodes which the existing codegen handles.

```go
// op_lowering.go — skip contract lowering
if prog.Meta != nil {
    lowered.Meta = prog.Meta  // Don't lower contracts
}
```

**Pros:** Simpler, avoids the whole DictApp problem
**Cons:** Contract expressions would need `--experimental-binop-shim` since they'd have unlowered Intrinsic nodes. Would need to either:
- Keep Intrinsic handling in codegen for contracts, OR
- Lower to `App{VarGlobal("$builtin", "ge_Int"), ...}` manually (not via dict dispatch)

### Option C: Lower contracts to builtins but NOT to dict dispatch

Modify OpLowering to lower contract expressions differently: convert `Intrinsic{OpGe}` → `App{VarGlobal("$builtin", "ge_Int"), ...}` but skip `FillOperatorMethods` on contract expressions. The ensures codegen already handles `App{VarGlobal("$builtin", ...)}` via `generateEnsuresApp` + `builtinNameToGoOp`.

1. Remove `FillOperatorMethods` call for contracts (pipeline_module_compile.go:312)
2. Keep OpLowering for contracts (produces `App{VarGlobal("$builtin", "ge_Int"), ...}`)
3. The existing `generateEnsuresApp` + `builtinNameToGoOp` handles this correctly

**Pros:** Clean separation — contracts use direct builtins, regular code uses dict dispatch
**Cons:** Need to verify OpLowering without FillOperatorMethods produces the right node type

### Recommendation: Option C (cleanest)

Option C is the minimal change that aligns contract codegen with its existing design assumptions. The ensures predicate handler was designed for `App{VarGlobal("$builtin", ...)}` nodes. The bug was introduced by calling `FillOperatorMethods` on contracts, which converts them to `DictApp` nodes that the handler doesn't understand.

---

## Files to Investigate/Modify

| File | Change |
|------|--------|
| `internal/pipeline/pipeline_module_compile.go:312` | Remove `FillOperatorMethods` call for contracts (Option C) |
| `internal/gen/golang/contracts.go:429` | DictApp case already added (Option A backup) |
| `internal/pipeline/op_lowering.go:89-111` | May need to adjust contract lowering |

## Success Criteria

- [ ] `TestContractViolation_Integration` passes locally and in CI
- [ ] `TestContractViolation_NoVerify` still passes
- [ ] Generated ensures checks use `_result` (not `result`)
- [ ] Generated ensures checks use typed Go operators (not dict dispatch)
- [ ] All existing tests pass
- [ ] Lint clean

## Testing Strategy

1. Run `ailang compile --emit-go --verify-contracts examples/runnable/contracts/basic.ail` — should produce compilable Go
2. Run the generated Go tests — contract violations should trigger panics with correct messages
3. Run full CI: `make test && make lint`

---

## Related Documents

- [M-CONTRACT-OPLOWERING-FIX](../../implemented/) — Introduced the contract OpLowering that caused this regression
- [M-CONTRACT-PIPELINE-DX](../../implemented/) — Quick wins from OpLowering fix retrospective

---

**Document created**: 2026-03-19
**Last updated**: 2026-03-19

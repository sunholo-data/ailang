# M-CONTRACTS-OPLOWERING: Apply OpLowering to Contract Expressions

**Status**: Planned
**Target Version**: v0.7.2
**Priority**: P2 (Low) - Quality of life improvement
**Estimated Effort**: 4-6 hours
**Dependencies**: M-VERIFY-CONTRACTS (implemented)

## Problem Statement

Contract expressions (`requires`/`ensures` blocks) currently require the `--experimental-binop-shim` flag because they skip the OpLowering pass:

```bash
# Current: Requires shim flag
ailang run --verify-contracts --experimental-binop-shim file.ail

# Goal: No extra flag needed
ailang run --verify-contracts file.ail
```

**Root Cause:**

Contract expressions are stored as raw Core AST with `Intrinsic` nodes:

```
Normal code:   Source → Parse → Elaborate → OpLowering → Eval
                                              ↑
                                     Intrinsic → DictApp

Contract expr: Source → Parse → Elaborate → [SKIP] → Eval
                                              ↑
                                     Intrinsic still present!
```

When the evaluator encounters an `Intrinsic` node (e.g., `>=`), it fails:

```go
// internal/eval/eval_operations.go:230
return nil, fmt.Errorf("intrinsic operations require OpLowering pass or --experimental-binop-shim flag")
```

**User Impact:**
- Extra flag to remember
- Inconsistent UX (contracts need special flag, regular code doesn't)
- Confusing error message if flag forgotten

## Goals

**Primary Goal:** Remove `--experimental-binop-shim` requirement for contract verification

**Success Metrics:**
- [ ] `ailang run --verify-contracts file.ail` works without binop shim
- [ ] Contract expressions use dictionary-based dispatch like regular code
- [ ] No regression in contract verification functionality
- [ ] All existing tests pass

## Solution Design

### Approach: Apply OpLowering During Elaboration

Contract expressions should go through OpLowering when they're stored in `core.Program.Meta`:

```go
// Current flow (internal/elaborate/file.go or similar)
contracts := parseContracts(funcDecl)
coreProg.Meta[funcName].Contracts = contracts  // Raw Intrinsic nodes

// New flow
contracts := parseContracts(funcDecl)
loweredContracts := opLowering.LowerContracts(contracts, typeInfo)
coreProg.Meta[funcName].Contracts = loweredContracts  // DictApp nodes
```

### Key Files to Modify

| File | Change |
|------|--------|
| `internal/elaborate/file.go` | Apply OpLowering to contract expressions during elaboration |
| `internal/pipeline/op_lowering.go` | Add `LowerContractExpr()` or extend `LowerExpr()` |
| `internal/core/core.go` | Ensure `ContractSpec.Expr` accepts lowered nodes |
| `cmd/ailang/main.go` | Remove shim requirement for `--verify-contracts` |

### Implementation Steps

**Phase 1: Infrastructure (~2 hours)**
- [ ] Add `LowerContractExpr(expr core.CoreExpr, typeInfo) core.CoreExpr` to op_lowering.go
- [ ] Ensure it handles all operators used in contracts (`>=`, `<=`, `==`, `!=`, `>`, `<`, `&&`, `||`)
- [ ] Add unit tests for contract expression lowering

**Phase 2: Integration (~2 hours)**
- [ ] Modify elaboration to apply OpLowering to contract expressions
- [ ] Ensure type information is available at contract lowering time
- [ ] Test with existing contract examples

**Phase 3: Cleanup (~1 hour)**
- [ ] Update `--verify-contracts` to not require `--experimental-binop-shim`
- [ ] Update documentation and examples
- [ ] Update manifest.json `run_flags` for contract examples

### Edge Cases

1. **Type availability**: OpLowering needs type information for dictionary lookup. Contracts are parsed after type checking, so types should be available.

2. **Boolean operators**: `&&` and `||` don't use type classes - ensure they're handled correctly (they already are in evalCoreBinOp).

3. **String concatenation**: `++` doesn't use type classes - ensure it's handled correctly.

4. **Nested expressions**: Contract like `requires { (x >= 0) && (x < 100) }` has nested operators - ensure recursive lowering.

## Testing

```ailang
-- Test cases to verify
export func testContract(x: int, y: float) -> int ! {}
requires { x >= 0 }           -- Int comparison
requires { y > 0.0 }          -- Float comparison
requires { x != y }           -- Mixed? (should type error)
ensures { result >= x }       -- Uses 'result' variable
{
  x + 1
}
```

**Test commands:**
```bash
# Should work without shim after this change
ailang run --verify-contracts examples/runnable/contracts/basic.ail
ailang run --verify-contracts examples/runnable/contracts/park.ail

# Violation should still produce clear error
ailang run --verify-contracts /tmp/contract_violation.ail
# Error: contract violation: requires failed at ...: (x >= 0)
```

## Alternatives Considered

### Alternative 1: Keep the shim as permanent solution
**Rejected**: The shim duplicates logic from dictionary dispatch. It's a maintenance burden and inconsistent with the rest of the language.

### Alternative 2: Runtime lowering of contracts
**Rejected**: Would need to carry type information to runtime. Better to lower at compile time.

### Alternative 3: Make `--experimental-binop-shim` the default
**Rejected**: The shim should be temporary, not a permanent part of the language.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A7: Machines First | +1 | Removes special-casing, consistent pipeline |
| A8: Syntax Is Liability | 0 | No syntax change |
| A10: Composability | +1 | Contracts compose with existing type system |

**Net Score: +2** → **Decision: Proceed**

## Related Documents

- [M-VERIFY-CONTRACTS](../../implemented/v0_7_1/m-verify-contracts.md) - Contract enforcement implementation
- [M-DX4-OPLOWERING](../../implemented/v0_3/20251010_float_equality_oplowering_fix.md) - Original OpLowering design

---

**Document created**: 2026-01-28
**Author**: Claude Code

# M-VERIFY: Runtime Contract Checks (Phase 0 + 0.5)

**Status**: IMPLEMENTED
**Version**: v0.6.2
**Completed**: 2025-12-19
**Duration**: ~12 hours across 2 sessions

---

## Summary

Implemented design-by-contract programming for AILANG with `requires` (preconditions) and `ensures` (postconditions) syntax. Contracts generate runtime checks that panic on violation when compiled with `--verify-contracts`.

This delivers immediate value from the M-VERIFY design by providing runtime contract enforcement before the higher-risk SMT verification phase.

---

## What Was Implemented

### Contract Syntax

```ailang
export func safeDivide(dividend: int, divisor: int) -> int ! {}
requires { divisor != 0 }        -- Precondition: checked at entry
ensures { result >= 0 }          -- Postcondition: checked before return
{
  dividend / divisor
}
```

### CLI Flags

```bash
# Compile with runtime contract checks
ailang compile --verify-contracts --emit-go --out ./gen module.ail

# Support for absolute paths
ailang compile --relax-modules --emit-go --out ./gen /path/to/module.ail
# Or: AILANG_RELAX_MODULES=1 ailang compile ...
```

### Generated Code

**With `--verify-contracts`**:
```go
func safeDivide_impl(dividend interface{}, divisor interface{}) interface{} {
    // Requires: (divisor != 0)
    if !(NeInt(divisor, int64(0))).(bool) {
        panic("contract violation: requires: (divisor != 0) at basic.ail:21:12")
    }
    return DivInt(dividend, divisor)
}

func SafeDivide(dividend int64, divisor int64) int64 {
    _result := safeDivide_impl(dividend, divisor).(int64)
    // Ensures: (result >= 0)
    if !(_result >= int64(0)) {
        panic("contract violation: ensures: (result >= 0) at basic.ail:22:11")
    }
    return _result
}
```

**Without `--verify-contracts`** (documentation only):
```go
func safeDivide_impl(dividend interface{}, divisor interface{}) interface{} {
    // Requires: (divisor != 0)
    return DivInt(dividend, divisor)
}

func SafeDivide(dividend int64, divisor int64) int64 {
    return safeDivide_impl(dividend, divisor).(int64)
}
```

---

## Architecture Decisions

### 1. Contract Storage: `core.DeclMeta.Contracts`

Contracts stored in Core AST metadata, not surface AST.

**Rationale**: Codegen operates on Core expressions; contracts need elaborated form for code generation.

```go
// internal/core/core_meta.go
type Contract struct {
    Kind     ContractKind  // RequiresKind, EnsuresKind, InvariantKind
    Expr     CoreExpr      // Elaborated predicate expression
    Message  string        // Original source text for error messages
    Location string        // "file.ail:line:col" for diagnostics
}
```

### 2. Requires: Runtime Helpers in `_impl`

Precondition checks use runtime helpers (`GeInt`, `NeInt`, etc.) because `_impl` functions use `interface{}` parameters.

```go
// Can't do: x >= 0 (x is interface{})
// Instead:  GeInt(x, int64(0)).(bool)
```

Added `mapIntrinsicToHelper()` to translate `core.OpGe` → `"GeInt"`, etc.

### 3. Ensures: Native Go Operators in Typed Wrapper

Postcondition checks use native Go operators because typed wrappers have concrete types.

```go
// In typed wrapper: _result is int64
if !(_result >= int64(0)) { ... }
```

Added `generateEnsuresPredicate()` with `result` → `_result` substitution and `core.Intrinsic` handling.

### 4. Comments Always Generated

Contract comments (`// Requires:`, `// Ensures:`) are always generated for documentation value, even without `--verify-contracts`. Only panic checks are conditional.

---

## Files Modified

### New Files
- `internal/gen/golang/contracts_integration_test.go` (~300 LOC) - End-to-end tests

### Modified Files
| File | Changes |
|------|---------|
| `internal/lexer/token.go` | Added `REQUIRES`, `ENSURES`, `INVARIANT` tokens |
| `internal/parser/parser_contracts.go` | Parse `requires { ... }` and `ensures { ... }` blocks |
| `internal/core/core_meta.go` | Added `Contract`, `ContractKind` types |
| `internal/elaborate/elaborate_decl.go` | Elaborate contracts from AST to Core |
| `internal/gen/golang/codegen.go` | Added `verifyContracts` field |
| `internal/gen/golang/codegen_decl.go` | Call contract checks, `hasEnsuresContracts()` |
| `internal/gen/golang/contracts.go` | `generateContractRequiresChecks()`, `generateContractEnsuresChecks()`, `generateEnsuresPredicate()`, `intrinsicOpToString()` |
| `internal/gen/golang/codegen_ops.go` | Added `mapIntrinsicToHelper()` |
| `internal/gen/golang/codegen_runtime_arith.go` | Added `NeInt` runtime helper |
| `cmd/ailang/compile.go` | Added `--verify-contracts`, `--relax-modules` flags |

---

## Test Coverage

All tests pass:

| Test | Purpose |
|------|---------|
| `TestContractViolation_Integration` | Full pipeline: AILANG → Go → compile → run tests |
| `TestAbsolute_ContractViolation` | Requires violation (negative input) |
| `TestSafeDivide_DivisionByZero` | Requires violation (divisor = 0) |
| `TestSafeDivide_EnsuresViolation` | Ensures violation (negative result) |
| `TestIncrement_EnsuresViolation` | Ensures violation (result not > x) |
| `TestContractViolation_NoVerify` | Comments without panics when flag off |

---

## Examples Created

| File | Purpose |
|------|---------|
| `examples/runnable/contracts/basic.ail` | Core contract patterns |
| `examples/runnable/contracts/park.ail` | ARC paper showcase (park admission policy) |

---

## Documentation

Updated `docs/docs/guides/contracts.mdx` with:
- Runtime contract checking section
- Generated code examples
- Contract violation message format
- Updated implementation status

---

## Future Work (See planned/v0_8_0)

The full M-VERIFY design includes:

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 0 + 0.5 | ✅ DONE | Runtime contract checks |
| Phase 1 | Planned v0.8.0 | SMT-based verification (`ailang verify`) |
| Phase 2 | Planned v0.8.0 | Redundant generation with contract filtering |
| Phase 3 | Planned v0.8.0+ | SharedMem invariants |

See [m-verify-smt-verification.md](../../planned/v0_8_0/m-verify-smt-verification.md) for comprehensive design.

---

## Research Foundation

Based on the ARC (Automated Reasoning Checks) system:
- Bayless et al., "A Neurosymbolic Approach to Natural Language Formalization and Verification" (AWS, 2025)
- [arXiv:2511.09008](https://arxiv.org/abs/2511.09008)

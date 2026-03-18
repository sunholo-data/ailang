# M-CONTRACT-OPLOWERING-FIX: Complete OpLowering for Contract Expressions

**Status**: Planned
**Target**: v0.9.2
**Priority**: P1 (High - blocks DocParse standalone CI eval)
**Estimated**: 2-4 hours
**Dependencies**: M-CONTRACTS-OPLOWERING (v0.8.0, partially implemented), M-VERIFY-CONTRACTS (v0.7.1)
**Milestone ID**: M-CONTRACT-OPLOWERING-FIX
**Created**: 2026-03-18
**Source**: DocParse agent message `337b4da1` (OpLowering CoreTI misses in ensures clauses)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates nondeterministic behavior where contracts fail depending on execution mode (standalone vs imported) |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Contract type info becomes locally verifiable — CoreTI covers all expressions |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents can use `--entry` on any module without remembering `--experimental-binop-shim` |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Contracts compose correctly with all operator types regardless of execution mode |
| A11: Structured Failure | +1 | Eliminates cryptic "intrinsic operations require OpLowering" error in favor of proper contract evaluation |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Fixes mode-dependent behavior
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Removes special-casing for agent workflows

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

This is a **completion of prior work**, not a new pattern:

1. **v0.7.1** (M-VERIFY-CONTRACTS): Added contract enforcement — `requires`/`ensures` clauses compiled to Core
2. **v0.8.0** (M-CONTRACTS-OPLOWERING): Added OpLowering traversal of contract Meta — lowerer now visits `prog.Meta[name].Contracts[*].Expr`
3. **v0.9.2** (THIS FIX): CoreTI gap — the type checker never processes contract expressions, so the lowerer's traversal hits CoreTI misses on every contract node

**The v0.8.0 fix was structurally correct but incomplete.** It ensured the lowerer visits contract expressions, but the lowerer depends on CoreTI for type resolution. Since the type checker (`CheckCoreProgram`) only processes `prog.Decls` and ignores `prog.Meta`, all contract nodes have missing CoreTI entries.

**Audit of CoreTI coverage:**
- `prog.Decls` expressions: fully covered by type checker
- `prog.Meta.Contracts[*].Expr` expressions: **NOT covered** (this bug)
- `prog.Flags`: no expressions to type-check

No other `core.Program` fields contain expressions that bypass type checking.

---

## Problem Statement

### Immediate Problem: DocParse Standalone Eval Broken

DocParse's `eval.ail` module uses `ensures` clauses for contract verification:

```ailang
export func evalJaccard(words1: [string], words2: [string]) -> float ! {IO}
ensures { result >= 0.0 && result <= 1.0 }
= { ... }
```

When run standalone via `--entry`:
```bash
ailang run --entry evalMain --caps IO,FS,Env docparse/services/eval.ail golden.json actual.json
# Error: intrinsic operations require OpLowering pass or --experimental-binop-shim flag
```

When imported from `main.ail`: **works fine** (contracts present but entry module doesn't trigger their evaluation path differently).

### Root Cause

The Core type checker (`internal/types/typechecker_core.go:414`) processes only `prog.Decls`:

```go
func (tc *CoreTypeChecker) CheckCoreProgram(prog *core.Program) (*typedast.TypedProgram, error) {
    for _, decl := range prog.Decls {           // <-- Only decls
        typedNode, env, err := tc.CheckCoreExpr(decl, globalEnv)
        // ...
    }
    // prog.Meta is NEVER processed -- contracts have no CoreTI entries
}
```

The OpLowering pass traverses contract expressions (added in v0.8.0) but hits CoreTI misses:

```
Contract expr node (e.g., result >= 0.0):
  1. CoreTI.Get(nodeID) -> miss (never type-checked)
  2. resolvedConstraints[nodeID] -> miss
  3. tryOperandTypes() -> miss (operands also untyped)
  4. isComparisonOrEqualityOp(>=) -> true
  5. DEFERRED as raw Intrinsic (line 420-427 of op_lowering.go)
  6. Evaluator hits Intrinsic -> ERROR
```

### Diagnostic Data (from DocParse)

```
Lowering telemetry for docparse/services/eval:
- 64 operators processed
- CoreTI hits: 49 (76.6%)
- CoreTI misses: 5 (7.8%) -- ALL in ensures clauses
- Missed nodes: >=, <=, >= (comparing floats/ints in ensures)
```

---

## Goals

**Primary Goal:** Ensure all expressions in `ensures`/`requires` contract clauses have CoreTI entries so OpLowering can resolve their types.

**Success Metrics:**
- `ailang run --entry evalMain --caps IO,FS,Env` works on modules with ensures clauses (no shim needed)
- CoreTI coverage: 100% of operator nodes in contracts have type entries
- No regression in contract verification or existing test suite
- `--debug-compile` shows 0 CoreTI misses for contract expressions

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where to type-check contracts: in CheckCoreProgram vs in pipeline post-TC | Determines data flow and coupling | agent | design | low |
| Whether to type-check with function's local env or fresh env | Affects `result` variable resolution in ensures | agent | design | med |
| Whether to fail or warn on contract CoreTI misses | Affects backward compat for modules with exotic contracts | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Where to type-check contracts (see Option analysis below)

---

## Solution Design

### Option Analysis

#### Option A: Type-check contracts in `CheckCoreProgram` (Recommended)

**Approach:** After type-checking each declaration, also infer types for its contract expressions using the same inference context.

**Pros:**
- Correct by construction — contracts get the same type environment as the function body
- CoreTI naturally populated for all contract nodes
- `result` variable gets proper type from function return type
- Single-pass — no extra pipeline phase

**Cons:**
- Touches the type checker (sensitive code)
- Must thread function return type into contract context for `result` binding

**LOC estimate:** ~20-30 lines in `typechecker_core.go`

#### Option B: Default comparison ops instead of deferring to shim

**Approach:** In `lowerIntrinsic`, when CoreTI misses for comparison ops, default to `Int` instead of returning raw `Intrinsic`.

**Pros:**
- Tiny change (2 lines in `op_lowering.go`)
- No type checker changes

**Cons:**
- **Wrong for float comparisons** — `result >= 0.0` would use `ge_Int` instead of `ge_Float`
- Masks the real problem (missing CoreTI entries)
- Every future CoreTI miss on a comparison op would silently default to Int
- Violates "no silent fallbacks" principle (CLAUDE.md rule 2)

**Verdict: REJECTED** — breaks float contracts, masks bugs.

#### Option C: Enable binop shim for contract evaluation only

**Approach:** When the evaluator enters contract checking, enable the binop shim automatically.

**Pros:**
- No compiler changes
- Works for all operator types

**Cons:**
- The shim is explicitly meant to be temporary (see M-CONTRACTS-OPLOWERING design doc)
- Shim doesn't support all operators correctly (e.g., `++` requires string operands check)
- Perpetuates the two-path evaluation inconsistency
- DocParse already tried `--experimental-binop-shim` and hit `'++' requires string operands`

**Verdict: REJECTED** — shim is temporary, doesn't even work for all cases.

#### Option D: Skip lowering for contract expressions entirely

**Approach:** In the lowerer, don't traverse `prog.Meta` contracts. Instead, always use the shim when evaluating contracts at runtime.

**Pros:**
- Simple change (revert M-CONTRACTS-OPLOWERING lines 89-111)

**Cons:**
- Same problems as Option C
- Reverts v0.8.0 work
- Contracts would permanently need different eval path

**Verdict: REJECTED** — same issues as Option C, plus regression.

### Recommended: Option A — Type-check contracts in `CheckCoreProgram`

#### Implementation Detail

The type checker processes declarations in `CheckCoreProgram`. After each declaration is type-checked, we need to also type-check its contract expressions. The key challenge is that `ensures` clauses reference a `result` variable bound to the function's return type.

```go
// In typechecker_core.go, after type-checking a declaration:

// After inferring the function's type, check its contracts
if prog.Meta != nil {
    if meta, ok := prog.Meta[declName]; ok && len(meta.Contracts) > 0 {
        // Build contract env: function's local env + "result" bound to return type
        contractEnv := env.Clone()
        if fnType, ok := declType.(*TFunc2); ok {
            contractEnv = contractEnv.Extend("result", fnType.Return)
        }

        // Type-check each contract expression (just for CoreTI population)
        for _, contract := range meta.Contracts {
            tc.inferCoreForCoreTypeInfo(ctx, contract.Expr, contractEnv)
        }
    }
}
```

The `inferCoreForCoreTypeInfo` helper would run inference and record CoreTI entries but discard the typed AST result (we don't need it — we just need CoreTI populated).

### Implementation Plan

**Phase 1: Core fix** (~2 hours)
- [ ] Add contract type-checking in `CheckCoreProgram` after each decl
- [ ] Handle `result` variable binding for `ensures` clauses
- [ ] Handle parameter bindings for `requires` clauses
- [ ] Test: create module with `ensures { result >= 0.0 }` and run with `--entry`
- [ ] Test: `--debug-compile` shows 0 CoreTI misses for contract ops

**Phase 2: Verification** (~1 hour)
- [ ] Run existing contract examples without `--experimental-binop-shim`
- [ ] Run full test suite (`make test`)
- [ ] Run `make verify-examples`
- [ ] Test with `-count=20` for determinism

**Phase 3: Cleanup** (~30 min)
- [ ] Update M-CONTRACTS-OPLOWERING design doc status
- [ ] CHANGELOG entry
- [ ] Notify DocParse agent

### Files to Modify

**Modified files:**
- `internal/types/typechecker_core.go` (~+25 LOC) — Add contract expression type inference in `CheckCoreProgram`

**No new files needed.**

---

## Examples

### Example 1: DocParse ensures clause (currently broken)

**Before:**
```bash
ailang run --entry evalMain --caps IO,FS,Env docparse/services/eval.ail golden.json actual.json
# Error: intrinsic operations require OpLowering pass or --experimental-binop-shim flag
```

**After:**
```bash
ailang run --entry evalMain --caps IO,FS,Env docparse/services/eval.ail golden.json actual.json
# Works: contracts type-checked, lowered, and evaluated correctly
```

### Example 2: Float comparison in ensures

```ailang
export func normalize(x: float) -> float
ensures { result >= 0.0 && result <= 1.0 }
= clamp(x, 0.0, 1.0)
```

**Before:** CoreTI miss on `>=` and `<=` nodes -> deferred as Intrinsic -> runtime error
**After:** CoreTI records Float type for both nodes -> lowered to `ge_Float` and `le_Float` -> correct runtime evaluation

---

## Success Criteria

- [ ] `ailang run --entry <func> module.ail` works for modules with ensures/requires clauses
- [ ] `--debug-compile` shows 0 CoreTI misses for operators in contracts
- [ ] Float comparisons in ensures (`result >= 0.0`) lower to `ge_Float` (not `ge_Int`)
- [ ] Int comparisons in requires (`x >= 0`) lower to `ge_Int`
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] No regression: modules without contracts unchanged

## Testing Strategy

**Unit tests:**
- Contract with int comparison (`requires { x >= 0 }`)
- Contract with float comparison (`ensures { result >= 0.0 }`)
- Contract with boolean logic (`ensures { result >= 0.0 && result <= 1.0 }`)
- Contract with string concat (`ensures { length(result) >= 0 }`)
- Contract referencing `result` variable with correct type
- Nested contracts on multiple functions in same module

**Integration tests:**
- Run existing `examples/runnable/contracts/` without `--experimental-binop-shim`
- DocParse eval.ail standalone execution (if available)

**Determinism:**
- All tests with `-count=20`

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether to add a dedicated `inferContractExpr` method or reuse `inferCore` — agent may choose based on what's cleanest
- Whether to also type-check `invariant` contracts (same pattern, lower priority) — agent may include if trivial

## Non-Goals

- **Full contract type-checking with error reporting** — We only need CoreTI population for lowering. Contract expressions are already structurally validated during elaboration. Full type error reporting for contracts is a separate feature.
- **Removing the binop shim entirely** — The shim still has other uses; this fix just removes the dependency for contracts.
- **Contract-aware optimizations** — No constant folding or simplification of contract expressions.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Type checker changes break existing inference | High | Only ADD type-checking for new nodes; don't modify existing decl inference path. Run full test suite. |
| `result` variable not in scope for ensures | Med | Explicitly bind `result` in contract env from function return type — same as evaluator already does |
| Contract expressions with polymorphic types | Low | If CoreTI records a type variable, lowerer already handles via resolved constraints fallback |
| Performance: extra inference pass for contracts | Low | Contracts are small expressions (typically 1-3 comparisons). Negligible overhead. |

## Related Documents

**Directly relevant (prior work on this exact issue):**
- [M-CONTRACTS-OPLOWERING](../../implemented/v0_8_0/m-contracts-oplowering.md) — v0.8.0 fix that added lowerer traversal of contracts (structurally correct, but incomplete without CoreTI)
- [M-VERIFY-CONTRACTS](../../implemented/v0_7_1/m-verify-contracts.md) — Original contract enforcement implementation

**Type checker / CoreTI:**
- [M-DX4 Sprint Plan](../../implemented/v0_3_18/M-DX4-SPRINT-PLAN.md) — CoreTI design and OpLowering architecture
- [typesIdentical Performance Bug](../../implemented/v0_5_7/types-identical-performance.md) — Prior CoreTI-related fix

**OpLowering:**
- [Float Equality / OpLowering Fix](../../implemented/v0_3/20251010_float_equality_oplowering_fix.md) — Original OpLowering design

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18

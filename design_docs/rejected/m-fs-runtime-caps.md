# M-FS-RUNTIME-CAPS: FS Capability Not Propagated at Runtime

**Status**: RESOLVED (not a bug — test syntax error)
**Target**: v0.9.3
**Priority**: Closed
**Estimated**: N/A
**Dependencies**: None
**Resolution**: The FS capability routing works correctly. The reported failure was caused by
test files using `let main = { ... }` (block expression) instead of
`export func main() -> () ! {IO, FS} { ... }` (function with effect declaration).
Verified: `ailang run --caps IO,FS` with correct syntax succeeds for all FS builtins.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | Fix makes FS capability grant actually work as documented |
| A4: Explicit Authority | +1 | Ensures `-caps FS` correctly grants filesystem access |
| A5: Bounded Verification | 0 | No verification impact |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | 0 | No impact |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | FS-effect functions become usable in pipelines |
| A11: Structured Failure | 0 | No change to error model |
| A12: System Boundary | +1 | Capability grant at CLI boundary works correctly |

**Net score: +4** — Accept

## Problem Statement

**Symptom**: Running `ailang run example.ail -caps FS,IO` — IO effects work (println succeeds) but FS effects fail with:
```
Error: execution failed: effect 'FS' requires capability, but none provided
Hint: Run with --caps FS
```

**Observed during**: M-DOCPARSE-DX sprint (M3, M4). Affects `listDir`, `readFile`, `writeFile`, `createArchive`, and all other FS-effect operations.

**Key clue**: `println` works but it's a `$builtin` that doesn't go through `effects.Call`. FS builtins route through `effects.Call(ctx, "FS", opName, args)` which calls `ctx.RequireCapWithBudget("FS", "")`, and this check fails.

## Investigation Required

The capability routing architecture is sound (same pattern for IO and FS). The issue is likely in **EffContext propagation** during module execution.

### Architectural Flow (from code review)

```
CLI: -caps FS,IO
  ↓ cmd/ailang/main.go: capsFlag parsed
  ↓ run_helpers.go: grantCapabilities(effCtx, "FS,IO")
  ↓   → effCtx.Grant(NewCapability("FS"))
  ↓   → effCtx.Grant(NewCapability("IO"))
  ↓ main.go: rt.GetEvaluator().SetEffContext(effCtx)
  ↓ main.go: executeModuleEntrypoint(rt, execParams)
  ↓ runtime/builtins.go: br.getEffContext() → ???
  ↓   → ctx.RequireCapWithBudget("FS", "") → FAILS
```

### Hypotheses

1. **EffContext copy without capabilities**: Module execution creates a fresh EffContext or copies one without the `Caps` map
2. **Different evaluator instance**: The module evaluator is different from the one that received `SetEffContext()`
3. **`println` is a $builtin**: It bypasses the effect system entirely, which is why it "works" — the IO capability is never actually checked for println
4. **Timing**: Capabilities are granted after the evaluator snapshot is taken

### Key Files to Investigate

| File | What to Check |
|------|---------------|
| `cmd/ailang/main.go:627-694` | Order of grantCapabilities vs SetEffContext |
| `cmd/ailang/run_helpers.go:110-120` | grantCapabilities implementation |
| `internal/runtime/builtins.go:62-88` | `br.getEffContext()` — does it return the granted context? |
| `internal/eval/evaluator.go` | SetEffContext and how it stores the context |
| `internal/runtime/module_exec.go` | Module execution path — does it create new evaluators? |

### Verification Steps

```bash
# 1. Confirm println bypasses effects (no IO capability check)
ailang run -e 'println("test")' -caps ''
# Expected: works (println is $builtin, no IO check)

# 2. Confirm FS check is the actual failure point
DEBUG_STRICT=1 ailang run examples/runnable/directory_listing.ail -caps FS,IO
# Look for where EffContext.HasCap("FS") returns false

# 3. Test with direct FS effect builtin
ailang run -e '_fs_exists("/tmp")' -caps FS
# Expected: should work if capabilities propagate
```

## Proposed Fix

After investigation, the fix is likely one of:

### A. EffContext propagation fix
If the EffContext isn't reaching module execution, wire it through:
```go
// In module execution setup
moduleEval.SetEffContext(parentEffCtx) // Propagate from CLI context
```

### B. Evaluator instance fix
If a new evaluator is created for module execution:
```go
// Ensure child evaluator inherits parent's EffContext
childEval := newEvaluator()
childEval.SetEffContext(parentEval.GetEffContext())
```

## Success Criteria

- [ ] `ailang run examples/runnable/directory_listing.ail -caps FS,IO` succeeds
- [ ] `ailang run examples/runnable/xml_zip_roundtrip.ail -caps FS,IO` succeeds
- [ ] All existing FS-effect tests still pass
- [ ] IO capability routing unchanged
- [ ] `make test` passes

## Related Documents

- [M-DOCPARSE-DX](m-docparse-dx.md) — Sprint where issue was identified
- [M-STDLIB-ZIP](../../implemented/v0_7_3/m-stdlib-zip.md) — ZIP implementation (affected)

## Bug Reports

- Identified during M-DOCPARSE-DX M3/M4 sprint execution (March 2026)
- Affects all FS-effect builtins: `_fs_readFile`, `_fs_writeFile`, `_fs_exists`, `_fs_listDir`, `_fs_appendFile`, `_zip_listEntries`, `_zip_readEntry`, `_zip_createArchive`

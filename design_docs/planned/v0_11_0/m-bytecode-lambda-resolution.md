# M-BYTECODE-LAMBDA-RESOLUTION: Fix Lambda Body Name Resolution in Bytecode Compiler

**Status**: Planned
**Priority**: P1 (largest single EvalOnly reduction opportunity)
**Target**: v0.11.0
**Estimated LOC**: ~150
**Dependencies**: M-BYTECODE-HOF-BUILTINS (complete)

## 1. Problem Statement

After wiring 55 pure builtins and 6 HOF builtins to the VM, **157 / 1130** prototypes in the docparse benchmark remain EvalOnly. The 10MB DOCX benchmark shows eval and bytecode VM at parity (~2.2s) because most hot-path prototypes still fall back to the evaluator.

**Root cause breakdown of 157 EvalOnly prototypes:**

| Category | Count | % | Fix |
|---|---|---|---|
| Call to unbound name (in lambda bodies) | 76 | 48% | **This sprint** |
| Effectful builtins not wired | 58 | 37% | Future: `OpEffectCall` |
| Register allocator overflow (>256 regs) | 9 | 6% | Future: register spilling |
| Unbound variable (in lambda bodies) | 3 | 2% | **This sprint** |
| Unknown ADT in switch | 2 | 1% | **This sprint** |
| Other / cascade | 9 | 6% | Cascade from above |

The top category (**79 prototypes, 50%**) shares a single root cause: `compileLambda` creates a child `funcCompiler` without propagating `currentModule`, so canonical name resolution fails inside lambda bodies.

## 2. Root Cause Analysis

### 2.1 The Bug: Missing `currentModule` in Lambda Compiler

In `internal/bytecode/compiler/lambda.go:51`:

```go
inner := newFuncCompiler(fc.img, innerProto, fc.funcIdx)
inner.recordTypes = fc.recordTypes
inner.adtTypes = fc.adtTypes
// BUG: inner.currentModule is "" (zero value)
```

When the lambda body references a same-module function (e.g., `helperFunc`), the resolution chain in `classifyCallee` and `compileVarRef` is:

1. `canonicalFuncName("", "helperFunc")` → `"helperFunc"` (bare)
2. Lookup `funcIdx["helperFunc"]` → **not found** (registered as `"module.helperFunc"`)
3. → `"compiler: call to unbound name"` → prototype marked **EvalOnly**

### 2.2 The Fix

```go
inner.currentModule = fc.currentModule  // ONE LINE
```

This propagates the module context so `canonicalFuncName("module", "helperFunc")` → `"module.helperFunc"` → found in `funcIdx`.

### 2.3 Unknown ADT in Switch (2 prototypes)

Two EvalOnly prototypes show `"unknown ADT ""  in switch"` — the empty string suggests `currentModule` is also missing when the compiler resolves ADT type names in switch expressions within lambda bodies. The same `currentModule` fix should resolve these.

### 2.4 Cascade Effects

Many of the 76 "call to unbound name" lambdas are **callbacks passed to HOF builtins** like `list_map`, `list_filter`, `str_foldChars`. When these lambdas compile successfully:
- The HOF builtin's `CallClosure` can invoke them natively (no EvalOnly bridge)
- Parent functions that were EvalOnly due to containing uncompilable lambdas may now compile
- This cascade could reduce EvalOnly by significantly more than the direct 79

**Conservative estimate**: 79 direct + ~20 cascade = **~99 prototypes eliminated**
**Optimistic estimate**: 79 direct + ~40 cascade = **~119 prototypes eliminated**

## 3. Scope

### In Scope
1. **Fix `currentModule` propagation** in `compileLambda` (1 line)
2. **Fix ADT resolution** in lambda bodies if still failing after fix #1
3. **Regression tests** for lambda name resolution in multi-module bytecode images
4. **Re-benchmark docparse** EvalOnly count and 10MB DOCX timing
5. **Parity check** to verify no regressions and potential new MATCH examples

### Out of Scope
- Effectful builtins (`OpEffectCall`) — separate design doc needed
- Register allocator overflow — separate investigation
- New VM value types (TagMap, TagBytes)

## 4. Implementation Plan

### 4.1 The One-Line Fix

File: `internal/bytecode/compiler/lambda.go:51`

```go
inner := newFuncCompiler(fc.img, innerProto, fc.funcIdx)
inner.recordTypes = fc.recordTypes
inner.adtTypes = fc.adtTypes
inner.currentModule = fc.currentModule  // NEW: propagate module context
```

### 4.2 ADT Resolution in Lambda Bodies

If `unknown ADT ""` persists after 4.1, check whether ADT type lookups also depend on `currentModule`. The compiler's `compileSwitch` likely uses the module prefix when resolving ADT constructor names. With `currentModule` propagated, this should resolve automatically.

### 4.3 Test Plan

1. **Unit test**: Multi-module bytecode image where lambda references same-module function
2. **Unit test**: Lambda references cross-module function (via GlobalRef — should already work)
3. **Unit test**: Nested lambdas (lambda inside lambda, both referencing outer module functions)
4. **Unit test**: Lambda passed to HOF builtin that invokes it via `CallClosure`
5. **Integration**: `ailang disasm docparse/main.ail` — verify EvalOnly reduction
6. **Parity**: `go run ./scripts/verify_bytecode_parity.go` — no regressions
7. **Benchmark**: 10MB DOCX best-of-3 wall-clock, eval vs bytecode

### 4.4 Benchmark Reproduction

```bash
# From ailang-parse repo root:
cd /path/to/ailang-parse

# EvalOnly count
ailang disasm docparse/main.ail 2>&1 | grep "EvalOnly:"

# BUILTIN_CALL_HOF count
ailang disasm docparse/main.ail 2>&1 | grep -c "BUILTIN_CALL_HOF"

# 10MB DOCX benchmark (best-of-3)
for backend in "" "--bytecode"; do
  echo "=== ${backend:-Evaluator} ==="
  for i in 1 2 3; do
    TMPOUT=$(mktemp -d)
    DOCPARSE_OUTPUT_DIR="$TMPOUT" /usr/bin/time ailang run \
      --entry main --caps IO,FS,Env $backend \
      docparse/main.ail data/test_files/stress/docx_10mb.docx 2>&1 | grep real
    rm -rf "$TMPOUT"
  done
done

# Parity check (from ailang repo root)
go run ./scripts/verify_bytecode_parity.go
```

## 5. Risks

### R1: Lambda captures interact with module resolution
**Risk**: Captured variables and module-level function references use different resolution paths. The fix must not break capture semantics.
**Mitigation**: Captures are resolved via `fc.locals.lookup()` which takes precedence over `funcIdx` lookups. The `currentModule` change only affects the fallback path when a name is NOT a local/capture. Test with lambdas that both capture variables AND reference module functions.

### R2: Cross-module lambda compilation
**Risk**: A lambda defined in module A but somehow compiled in module B's context could resolve names incorrectly.
**Mitigation**: Lambdas are always compiled inline within their parent function — they inherit the parent's `currentModule`, which is correct. The Statement IR lowering phase ensures the module context is set per-function.

### R3: Cascade EvalOnly reduction is smaller than expected
**Risk**: Fixing lambda resolution might not cascade as much as estimated if other issues (effectful builtins, missing types) block parent prototypes.
**Mitigation**: The benchmark will measure actual reduction. Even the conservative estimate (79 direct) represents a 50% reduction in remaining EvalOnly.

## 6. Success Criteria

| Metric | Current | Target |
|---|---|---|
| EvalOnly count | 157 / 1130 | **≤ 80 / 1130** |
| Parity MATCH | 129 | **≥ 129** (no regression) |
| 10MB DOCX eval | 2.21s | baseline |
| 10MB DOCX bytecode | 2.22s | **< 2.1s** (target: measurable speedup) |
| `BUILTIN_CALL_HOF` | 6 | **≥ 6** (may increase if more lambdas compile) |

## 7. AILANG Design Principles Alignment

| Principle | Score | Rationale |
|---|---|---|
| A1: AI Synthesizability | 0 | No syntax change |
| A2: Determinism | 0 | No behavior change |
| A3: Semantic Transparency | +1 | Compiler errors become correct compilation |
| A7: Machines First | +2 | Major EvalOnly reduction, enabling VM execution of hot paths |
| A8: Debuggability | +1 | Fewer opaque EvalOnly fallbacks in disassembly |

## 8. Related Documents

- [m-bytecode-vm.md §18.7](../../implemented/v0_11_0/m-bytecode-vm.md) — HOF builtins results, benchmark baseline
- [m-bytecode-hof-builtins.md](m-bytecode-hof-builtins.md) — HOF infrastructure (prerequisite)
- [m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md) — Parity bugs (may see new MATCH after this fix)
- [m-phase2c-bytecode-compiler-sprint-plan.md](m-phase2c-bytecode-compiler-sprint-plan.md) — Original Phase 2C plan including lambda compilation

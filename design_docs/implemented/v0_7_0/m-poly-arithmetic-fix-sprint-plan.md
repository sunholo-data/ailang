# Sprint Plan: M-POLY-ARITH - Fix Polymorphic Arithmetic Operators

## Summary

Fix polymorphic arithmetic operators (`+`, `-`, `*`, `/`, `%`) in let-bound lambdas so that `let add = \x. \y. x + y in add(3.14)(2.71)` returns `5.85` instead of panicking with "expected int arguments". This also fixes the WASM REPL where the same pipeline is used.

**Duration:** 1 day (4-6 hours)
**Dependencies:** None (builds on M-POLY-B Phase 1 which fixed comparison operators)
**Risk Level:** Medium (touches type defaulting and specializer)

## Current Status Analysis

### Completed Recently
- v0.7.1: WASM bugs fixed, module registry arithmetic tests passing for concrete types
- M-POLY-B Phase 1: Comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) fixed
- Dictionary elaboration added to file pipeline

### Remaining from Design Doc
- Arithmetic operators in polymorphic lambdas: `let add = \x. \y. x + y in add(3.14)(2.71)` panics
- Same bug manifests in WASM REPL

### Root Cause Analysis (Confirmed via Debug Tracing)

**Three interacting problems identified:**

1. **Premature Num defaulting**: Type inference defaults `\x. \y. x + y` to `int -> int -> int` via Num typeclass defaulting. The type variable α gets Num constraint, and defaulting resolves it to `int` BEFORE the call site `add(3.14)(2.71)` can unify α with `float`.

2. **Dictionary elaboration runs before specialization**: Pipeline order is type-check → dict-elaborate → specialize. So by the time dict elaboration runs, the `+` operator is transformed to `DictApp(Dict=DictRef(Num, Int), ...)` with Int hardcoded.

3. **Specializer doesn't handle DictRef**: `specialize_expr.go:325` falls through to default case for `*core.DictRef`, silently returning it as-is (panics in DEBUG_STRICT mode).

**Debug evidence:**
```
[DEBUG] Dictionary elaboration complete for module
[DEBUG_MONO_VERBOSE] Found lambda, type=int -> int -> int, isPoly=false
[DEBUG_MONO_VERBOSE] Monomorphization: 0 specializations
Error: execution failed: expected int arguments
```

**Comparison operators work because:**
- Ord is a "neutral class" in defaulting logic (`typechecker_defaulting.go:197-201`)
- Operator lowering uses operand type for comparisons vs intrinsic result type for arithmetic

## Proposed Solution: Reorder Pipeline + Fix Specializer

### Approach: Run Dictionary Elaboration AFTER Monomorphization

Instead of fighting the defaulting rules, reorder the pipeline so specialization runs on BinOp nodes (which the specializer already handles), THEN dictionary elaboration runs on the specialized code with concrete types.

**Current pipeline order:**
```
type-check → dict-elaborate → specialize → op-lower → eval
                    ↑ BinOp becomes DictApp(Int) here
```

**Fixed pipeline order:**
```
type-check → specialize → dict-elaborate → op-lower → eval
                              ↑ Now BinOp is specialized with concrete types first
```

**Why this works:**
- Specialization operates on BinOp nodes (already supported)
- After specialization with type substitution, CoreTypeInfo has concrete types
- Dictionary elaboration then creates DictRef with correct types (Float, not Int)
- Operator lowering sees correct types → selects correct builtin

**Fallback approach (if reordering breaks things):**
- Keep current order but add DictRef/DictApp handling to specializer
- During specialization, when type substitution maps α→Float, update DictRef TypeName from "Int" to "Float"

## Proposed Milestones

### Milestone 1: Investigation & Test Scaffolding
**Goal:** Reproduce the bug in unit tests and confirm root cause
**Estimated:** ~80 LOC (tests)

**Tasks:**
- Add failing integration test in `specialize_integration_test.go`:
  - `TestSpecializeVarBoundLambda_FloatArithmetic` - the main bug
  - `TestSpecializeVarBoundLambda_AllArithOps` - `+`, `-`, `*`, `/`, `%`
- Add test for DictRef handling in specializer
- Verify comparison operator tests still pass

**Acceptance Criteria:**
- [ ] Failing test reproduces `let add = \x. \y. x + y in add(3.14)(2.71)` panic
- [ ] Tests for all 5 arithmetic operators exist
- [ ] Comparison operator tests continue to pass

### Milestone 2: Fix Pipeline Order or Specializer
**Goal:** Make polymorphic arithmetic operators work with floats
**Estimated:** ~120 LOC implementation

**Primary approach - Reorder pipeline:**
- In `pipeline_module.go` and `pipeline.go`: move `ElaborateWithDictionaries()` to run AFTER `Specializer.Specialize()`
- Verify dict elaboration still has access to resolvedConstraints after specialization
- Update CoreTypeInfo node IDs if specializer creates new nodes

**Fallback approach - Fix specializer:**
- Add `DictRef` case to `specialize_expr.go`
- When type substitution changes the class type, update DictRef TypeName
- Add `DictApp` re-elaboration to clone/substitute DictRef args

**Tasks:**
- Move dictionary elaboration after specialization in both pipelines
- OR add DictRef/DictApp handling to specializer
- Verify no regressions with `make test`

**Acceptance Criteria:**
- [ ] `let add = \x. \y. x + y in add(3.14)(2.71)` returns 5.85
- [ ] All arithmetic operators work: `+`, `-`, `*`, `/`, `%`
- [ ] Comparison operators still work (regression check)
- [ ] WASM module registry tests pass
- [ ] `make test` passes

### Milestone 3: Comprehensive Testing & Cleanup
**Goal:** Ensure robustness and document the fix
**Estimated:** ~100 LOC (tests + docs)

**Tasks:**
- Test multiple type specializations: Int, Float, String (for `++`)
- Test nested operators: `(x + y) * (x - y)`
- Test mixed operators: `if x > 0 then x * 2 else x`
- Run `make test` and `make lint`
- Rebuild WASM binary: `make wasm`
- Update design doc status from Planned to Implemented
- Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] Nested operators work in polymorphic lambdas
- [ ] Mixed comparison+arithmetic in same lambda works
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] WASM binary rebuilt
- [ ] Design doc updated

## Success Metrics
- `let add = \x. \y. x + y in add(3.14)(2.71)` returns `5.85`
- All 5 arithmetic operators work in polymorphic lambdas
- All existing tests pass (no regressions)
- WASM REPL handles arithmetic operations correctly
- Linting clean

## Key Files to Modify

| File | Change |
|------|--------|
| `internal/pipeline/pipeline.go` | Reorder dict elaboration after specialization |
| `internal/pipeline/pipeline_module.go` | Same reorder for module pipeline |
| `internal/pipeline/specialize_expr.go` | Add DictRef case (fallback approach) |
| `internal/pipeline/specialize_integration_test.go` | New test cases |
| `design_docs/implemented/v0_7_0/m-poly-arithmetic-fix.md` | Update status |
| `CHANGELOG.md` | Document fix |

**Total estimated LOC:** ~300 (implementation + tests)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pipeline reorder breaks dict elaboration | High | Verify resolvedConstraints survive specialization; fallback to specializer fix |
| New CoreTypeInfo node IDs from specializer break dict elaboration | Medium | Map old→new node IDs during specialization |
| Regression in comparison operators | High | Existing tests cover this; run full test suite |
| REPL path has different pipeline order | Medium | REPL already has dict elaboration; check REPL pipeline separately |

## Notes

- The REPL and file pipeline have slightly different orders - both need to be checked
- WASM uses the module registry which shares the file pipeline
- Inline lambdas already work (they're specialized at the call site before defaulting)
- The `DEBUG_STRICT=1` panic on DictRef is a useful safety check - should be fixed regardless

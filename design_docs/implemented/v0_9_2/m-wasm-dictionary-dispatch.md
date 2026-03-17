# M-WASM-DICTIONARY-DISPATCH: Fix WASM Dictionary Dispatch Bugs

**Status**: Planned
**Target**: v0.9.2
**Priority**: P0 - High (blocks website-builder demos in browser)
**Estimated**: 1-2 days (4h implementation + 4h testing + buffer)
**Dependencies**: None (all infrastructure exists in native pipeline)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates nondeterministic dispatch (eq_Int vs eq_String depends on declaration position) |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Effects unchanged |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Type-directed dispatch becomes correct — local reasoning works |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI-generated code works identically in CLI and WASM — no target-specific workarounds |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Imported symbols compose correctly (no shadowing); helper extraction works |
| A11: Structured Failure | +1 | Replaces opaque `eq_Int: expected IntValue` crash with correct dispatch |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: ✅ Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Fixes nondeterminism — currently dispatch depends on import order
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Unifies CLI and WASM behavior

## Problem Statement

The WASM runtime (ModuleRegistry path) has two bugs caused by missing compilation phases that the native CLI pipeline includes. Both bugs produce correct results on CLI but crash in WASM.

### Bug 1: Import Name Shadowing

When a module imports identically-named functions from different stdlib modules (e.g., `length` from both `std/list` and `std/string`), the WASM runtime resolves all references to whichever import was evaluated last, regardless of type.

**Reproduction:**
```ailang
module validator

import std/string (length)          -- string -> int
import std/list (foldl, concat)     -- also has length: [a] -> int

-- Using listLength alias doesn't help — underlying `length` resolves globally
import std/list (length as listLength)

ensures listLength(result.errors) == 0
-- CRASH: eq_Int: expected IntValue, got StringValue
-- Because std/string.length shadows std/list.length in ensures scope
```

**Workaround in use:** Replace `listLength` with `foldl(\acc _. acc + 1, 0, xs)` in `website_builder/services/validator.ail`.

### Bug 2: eq_Int Dispatch in Non-Exported Helper Lambda

When a non-exported helper function passes a lambda containing `==` on strings to a higher-order function like `any()`, the `==` dispatches to `eq_Int` instead of `eq_String`.

**Reproduction:**
```ailang
module test/bug

import std/list (any)
import std/json (decode, getString, getArray)
import std/option (Some, None)

-- Non-exported helper — CRASHES in WASM
pure func hasSlug(pages: [Json], slug: string) -> bool =
  any(\page. match getString(page, "slug") {
    Some(s) => s == slug,   -- dispatches eq_Int instead of eq_String!
    None => false
  }, pages)

export func test(siteJson: string) -> {valid: bool, errors: [string]} {
  -- ... calls hasSlug(pages, "home") ...
}
```

**Key observations:**
- CLI works fine for both — bug is WASM-only
- Inlining the `any()` call in the export function works
- The `==` inside the lambda dispatches to `eq_Int` when called through a non-exported helper
- Tested on v0.9.1.1 release WASM and latest — same behavior
- This is NOT the December M-LETREC-SCOPING fix (different bug)

**Workaround in use:** Inline the `any()` call in the export function body.

**Impact:**
- website_builder `validator.ail` requires two workarounds (countList + inlined any)
- Any WASM module using helper functions with polymorphic operators is affected
- Developers must know target platform when writing AILANG — violates "write once, run anywhere"

## Root Cause Analysis

### Pipeline Comparison

The native CLI pipeline (`internal/pipeline/pipeline_module.go`) runs these phases:

```
Phase 1:   Parse
Phase 2:   Elaborate (AST → Core)
Phase 3:   Type Check
Phase 3.4: Dictionary Elaboration (BinOp → DictApp)
Phase 3.5: ★ MONOMORPHIZATION ★ (specialize polymorphic functions)
Phase 3.5.5: ★ VAR TYPE RESOLUTION ★ (resolve type variables in Vars)
Phase 3.6: Operator Lowering (dictionary dispatch)
Phase 4:   Evaluate
```

The WASM/REPL pipeline (`internal/repl/module_registry.go`) runs:

```
Step 1: Parse
Step 2: Elaborate
Step 3: Type Check
Step 4: Dictionary Elaboration
Step 5: Op Lowering          ← runs WITHOUT monomorphization first!
Step 6: Link Dictionaries
Step 7: Evaluate
```

**Missing phases in WASM: Monomorphization (3.5) and Var Type Resolution (3.5.5)**

### Why This Causes Bug 1 (Import Shadowing)

With monomorphization (native CLI):
- `std/list.length` is specialized to `_length$List_Int`, `_length$List_String`, etc.
- `std/string.length` is specialized to `_length$String`
- No name collision — distinct specialized function names

Without monomorphization (WASM):
- Both remain as unqualified `length`
- Last import wins in the global evaluation scope
- `listLength` alias resolves to the aliased function, but operator lowering can't distinguish the underlying type

### Why This Causes Bug 2 (eq_Int in Helper Lambda)

With monomorphization + var type resolution (native CLI):
- The lambda `\page. ... s == slug ...` is specialized for the concrete type `string`
- `==` is resolved to `eq_String` during operator lowering because the specializer has made the types concrete
- Var type resolution fills in any remaining type variables

Without these phases (WASM):
- The lambda remains polymorphic — `s == slug` has type `a == a` where `a` is a type variable
- Operator lowering sees a type variable, not `string`, and defaults to `eq_Int` (the first/default Eq instance)
- When the lambda is inside an exported function (inlined), the type context is richer and the types resolve correctly
- When extracted to a non-exported helper, the type context is weaker and the type variable persists

## Goals

**Primary Goal:** Achieve compilation parity between WASM (ModuleRegistry) and native CLI (Pipeline) by adding the missing monomorphization and var type resolution phases.

**Success Metrics:**
- Both bug reproduction cases pass in WASM without workarounds
- Existing WASM tests continue to pass
- `validator.ail` works without countList workaround
- `hasSlug` helper works without inlining
- No measurable init time regression (< 100ms increase)

## Solution Design

### Overview

Add monomorphization and var type resolution to `ModuleRegistry.LoadModule()`, inserting them between the existing dictionary elaboration (Step 4) and operator lowering (Step 5) — matching the native pipeline order.

### Architecture

The fix reuses existing infrastructure from the native pipeline. No new algorithms needed.

**Components:**
1. **Monomorphization insertion**: Call `pipeline.NewSpecializer()` + `specializer.Specialize()` after dictionary elaboration
2. **Var type resolution insertion**: Call `pipeline.NewVarResolver()` + `resolver.Resolve()` after monomorphization
3. **CoreTypeInfo propagation**: Ensure the type checker's `CoreTI` is available for both phases

### Implementation Plan

**Phase 1: Add Monomorphization to ModuleRegistry** (~3 hours)
- [ ] After Step 4 (dictionary elaboration) in `module_registry.go`, add monomorphization call
- [ ] Create `pipeline.NewSpecializer(&typeChecker.CoreTI)`
- [ ] Call `specializer.Specialize(elaboratedProg)` on the elaborated Core program
- [ ] Handle errors (wrap with module context)
- [ ] Add `pipeline.NewVarResolver(typeChecker.CoreTI)` after specialization
- [ ] Call `resolver.Resolve(specializedProg)`
- [ ] Update Step 5 (op lowering) to use the specialized/resolved program

**Phase 2: Test Both Bug Fixes** (~3 hours)
- [ ] Write WASM-path test: module importing both `std/string.length` and `std/list.length`
- [ ] Write WASM-path test: non-exported helper with lambda using `==` on strings
- [ ] Write WASM-path test: `ensures` clause with `listLength`
- [ ] Verify existing `module_registry_test.go` tests pass
- [ ] Run `make test` and `make lint`
- [ ] Build WASM: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`

**Phase 3: Remove Workarounds and Verify** (~1 hour)
- [ ] Update `validator.ail` to use `std/list.length` directly (remove countList)
- [ ] Update `validator.ail` to extract `hasSlug` as a helper (remove inlining)
- [ ] Verify in browser

### Files to Modify/Create

**Modified files:**
- `internal/repl/module_registry.go` - Add monomorphization + var resolution between Steps 4-5 (~20-30 LOC)

**New test files:**
- `internal/repl/module_registry_dispatch_test.go` - Tests for both bug reproductions (~100-150 LOC)

**Updated demo files (after fix verified):**
- `demos/website_builder/services/validator.ail` - Remove workarounds

## Examples

### Example 1: Import Shadowing (Bug 1)

**Before (crashes in WASM):**
```ailang
import std/string (length)
import std/list (length as listLength)

-- WASM: listLength resolves to std/string.length → type mismatch
ensures listLength(result.errors) == 0
```

**After (works in WASM):**
```ailang
import std/string (length)
import std/list (length as listLength)

-- Monomorphization specializes to distinct functions
-- listLength correctly dispatches to list-specialized length
ensures listLength(result.errors) == 0
```

### Example 2: Helper Lambda eq Dispatch (Bug 2)

**Before (crashes in WASM):**
```ailang
-- Non-exported helper — eq_Int dispatched for string ==
pure func hasSlug(pages: [Json], slug: string) -> bool =
  any(\page. match getString(page, "slug") {
    Some(s) => s == slug,  -- CRASH: eq_Int instead of eq_String
    None => false
  }, pages)
```

**After (works in WASM):**
```ailang
-- Same code, now monomorphized: == resolves to eq_String
pure func hasSlug(pages: [Json], slug: string) -> bool =
  any(\page. match getString(page, "slug") {
    Some(s) => s == slug,  -- Correctly dispatches eq_String
    None => false
  }, pages)
```

## Success Criteria

- [ ] Bug 1 repro passes: module with `std/string.length` + `std/list.length` imports works in WASM
- [ ] Bug 2 repro passes: non-exported helper with `==` on strings dispatches `eq_String` in WASM
- [ ] `validator.ail` works without countList workaround
- [ ] `validator.ail` works with extracted `hasSlug` helper
- [ ] All existing `module_registry_test.go` tests pass
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] WASM builds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`
- [ ] No init time regression > 100ms

## Testing Strategy

**Unit tests (module_registry_dispatch_test.go):**
- Load module with dual `length` imports, verify correct dispatch
- Load module with non-exported helper + lambda `==`, verify `eq_String` dispatch
- Load module with `ensures` clause using imported list functions
- Verify monomorphization produces distinct specialized names

**Integration tests:**
- Run existing WASM test suite
- Build WASM binary

**Manual testing:**
- Load `validator.ail` without workarounds in browser portal
- Test `validateSite` with valid and invalid inputs
- Verify no crash on `ensures` evaluation

## Non-Goals

**Not in this feature:**
- Optimizing WASM binary size — monomorphization may increase code size, acceptable trade-off
- Lazy monomorphization — full eager specialization is fine for module-sized code
- Fixing the M-WASM-CLOSURE-ENV bug — separate design doc exists for that (different root cause)
- Unifying ModuleRegistry and Pipeline into a single path — desirable but much larger scope

## Timeline

**Day 1** (6-8 hours):
- Phase 1: Add monomorphization + var resolution to ModuleRegistry
- Phase 2: Write and run tests

**Day 2** (2-4 hours):
- Phase 3: Remove workarounds, verify in browser
- Documentation updates

**Total: ~8-12 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Monomorphization depends on pipeline state not available in ModuleRegistry | High | Check that `CoreTypeInfo` is fully populated after Step 3 (type check) — same data native pipeline uses |
| WASM binary size increase from specialization | Low | Stdlib is small; measure before/after |
| Init time regression from extra compilation phase | Med | Benchmark; consider lazy per-module specialization if > 200ms |
| Monomorphization interacts with WASM closure env bug (M-WASM-CLOSURE-ENV) | Med | Test closure scenarios; if conflict, fix closure env first |
| Type checker in ModuleRegistry produces different CoreTypeInfo than native | Med | Add assertion comparing TypeInfo structure between paths |

## Related Documents

**Same bug family (WASM compilation gaps):**
- [design_docs/implemented/v0_8_1/m-wasm-closure-env.md](../implemented/v0_8_1/m-wasm-closure-env.md) — Closure env resolution in WASM (different layer, same theme)
- [design_docs/implemented/v0_7_2/m-wasm-stdlib.md](../implemented/v0_7_2/m-wasm-stdlib.md) — Embedded stdlib in WASM
- [design_docs/implemented/v0_7_1/m-wasm-repl-sprint-plan.md](../implemented/v0_7_1/m-wasm-repl-sprint-plan.md) — Original ModuleRegistry design

**Native pipeline (reference implementation):**
- [design_docs/implemented/v0_6_1/m-letrec-scoping-regression.md](../implemented/v0_6_1/m-letrec-scoping-regression.md) — Monomorphization cache key fix (related but different bug)

**Auto-populated by neural search:**
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](../implemented/v0_6_0/semantic-caching-complete.md) (0.45)
- [design_docs/planned/v0_10_0/m-hash-collections.md](v0_10_0/m-hash-collections.md) (0.41)
- [design_docs/planned/v0_10_0/m-bug-letrec-single-call.md](v0_10_0/m-bug-letrec-single-call.md) (0.39)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `internal/repl/module_registry.go:196-585` — WASM compilation path (Steps 1-7)
- `internal/pipeline/pipeline_module.go:689-747` — Native monomorphization + var resolution (Phases 3.5-3.5.5)
- `internal/pipeline/specialize.go` — Specializer entry point
- `internal/pipeline/specialize_lambda.go` — Lambda specialization (cache key fix from M-LETREC-SCOPING)
- `internal/pipeline/op_lowering.go` — Operator → dictionary dispatch transformation
- `internal/elaborate/dictionaries.go` — Dictionary elaboration (BinOp → DictApp)
- Bug report message: `msg_20260317_112133_7d108d5c` (eq_Int in non-exported helper)
- Bug report message: `msg_20260316_*` (listLength shadowing in validates)

## Future Work

- **Unify ModuleRegistry and Pipeline**: Long-term, ModuleRegistry should use the same pipeline as native CLI, not a hand-rolled subset. This would prevent future phase-gap bugs.
- **WASM compilation tests in CI**: Add automated tests that verify WASM path matches native CLI output for a set of reference modules.
- **Phase parity assertion**: Add a compile-time or test-time check that ModuleRegistry includes all pipeline phases.

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17

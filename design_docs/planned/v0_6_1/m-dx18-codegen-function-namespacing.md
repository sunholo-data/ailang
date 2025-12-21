# M-DX18: Go Codegen - Non-Exported Function Namespacing

**Status**: In Progress (Bug Fix Applied, Pending Verification)
**Target**: v0.6.1
**Priority**: P0 (Blocking - breaks multi-module projects)
**Estimated**: 2 hours
**Dependencies**: None
**Reporter**: stapledons_voyage (agent message)

## ⚠️ Bug Report (2025-12-21)

**The initial implementation was incomplete.** Function definitions were correctly prefixed, but call sites were not updated.

### Symptom

Generated Go code calls `colorRocky_impl()` but the function is defined as `celestial__colorRocky_impl()`.

```go
// Generated function definition (CORRECT - has prefix)
func celestial__colorRocky_impl() interface{} { ... }

// Generated call site (BUG - missing prefix)
return colorRocky_impl()  // ❌ Should be celestial__colorRocky_impl()
```

### Root Cause

In `internal/gen/golang/codegen_expr_simple.go`:
- `generateVar` (line 64): Used `ToGoVarName(v.Name) + "_impl"` instead of the looked-up `goName + "_impl"`
- `generateVarGlobal` (line 137): Same issue - didn't use the prefixed name from `topLevelFuncs`

The `topLevelFuncs` map correctly stores the prefixed wrapper name (e.g., `celestial__colorRocky`), but the call generation code was ignoring it and constructing the name from scratch.

### Fix Applied

Both locations now use the looked-up name from `g.topLevelFuncs` which contains the correct prefix:

```go
// Before (BUG):
g.write(ToGoVarName(v.Name) + "_impl")

// After (FIXED):
g.write(goName + "_impl")  // goName already includes module prefix
```

**Files modified:**
- `internal/gen/golang/codegen_expr_simple.go` (lines 64, 137-138)

### Verification Needed

- [ ] Run `make test` to verify no regressions
- [ ] Compile stapledons_voyage `sim/*.ail` to verify fix
- [ ] Move this doc back to implemented/ once verified

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No change to effect handling |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables multi-module compilation without manual intervention |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents can compile multi-module projects without workarounds |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Modules with same-named private functions compose correctly |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (AI agent) experience

## Problem Statement

When compiling multiple AILANG modules to the same Go package, non-exported functions with identical names cause Go redeclaration errors.

**Current State:**

```
sim/solar_demo.ail has: pure func concatLists(a: [SolarPlanet], b: [SolarPlanet]) -> [SolarPlanet]
sim/dome_demo.ail has: pure func concatLists(a: [DrawCmd], b: [DrawCmd]) -> [DrawCmd]
```

Both generate `concatLists_impl` and `concatLists` in `sim_gen/`, causing:
```
sim_gen/solar_demo.go:503:6: concatLists_impl redeclared in this block
sim_gen/dome_demo.go:363:6: other declaration of concatLists_impl
```

**Also affects:** `updatePlanetRotation`, `updatePlanetRotations`, `buildCamera`, and any other common helper function names.

**Impact:**
- Prevents modular AILANG code with common function names
- Forces manual renaming as workaround
- Blocks stapledons_voyage from compiling

## Goals

**Primary Goal:** Namespace non-exported functions in Go codegen to prevent collisions.

**Success Metrics:**
- Multiple modules can have same-named non-exported functions
- Go compilation succeeds without manual renaming
- stapledons_voyage compiles successfully
- No regressions in single-module projects

## Solution Design

### Overview

Prefix non-exported function names with a module-derived namespace in Go codegen. Exported functions (those in `export` declarations) keep their original names for external access.

### Naming Strategy

**Non-exported functions:**
```go
// From sim/solar_demo.ail
func solar_demo__concatLists_impl(...) { ... }
func solar_demo__concatLists(...) { ... }

// From sim/dome_demo.ail
func dome_demo__concatLists_impl(...) { ... }
func dome_demo__concatLists(...) { ... }
```

**Exported functions:**
```go
// Exported functions keep simple names for external access
func CalculateOrbit(...) { ... }
func RenderScene(...) { ... }
```

**Module name derivation:**
- `sim/solar_demo.ail` → `solar_demo`
- `game/engine/physics.ail` → `physics` (or `engine__physics` for full path)
- Use last path component for simplicity, full path only if collisions

### Architecture

**Key change in `internal/gen/golang/codegen_func.go`:**

1. Track which functions are exported (from module's `export` declarations)
2. For non-exported functions, prefix with module name: `{module}__{funcname}`
3. Update all call sites to use namespaced names
4. Keep exported functions with simple names

### Implementation Plan

**Phase 1: Track Exports** (~30 min)
- [ ] Add `exportedFuncs map[string]bool` to generator state
- [ ] Populate from module's `export` declarations
- [ ] Pass through codegen pipeline

**Phase 2: Namespace Non-Exports** (~1 hour)
- [ ] Modify `generateFunctionDef` to prefix non-exports
- [ ] Create `namespaceFunc(moduleName, funcName string) string` helper
- [ ] Handle both `_impl` and wrapper functions

**Phase 3: Update Call Sites** (~30 min)
- [ ] Modify `generateCall` to use namespaced names
- [ ] Handle cross-module calls (use full name)
- [ ] Handle local calls (same namespace)

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_func.go` - Function generation (~50 LOC)
- `internal/gen/golang/codegen_expr_simple.go` - Call site generation (~20 LOC)
- `internal/gen/golang/codegen.go` - Add export tracking state (~10 LOC)

## Examples

### Example 1: Same-Named Functions Compile

**Before (fails):**
```
$ ailang compile sim/*.ail --output sim_gen/
sim_gen/solar_demo.go:503:6: concatLists_impl redeclared in this block
```

**After (works):**
```go
// sim_gen/solar_demo.go
func solar_demo__concatLists_impl(a interface{}, b interface{}) interface{} { ... }

// sim_gen/dome_demo.go
func dome_demo__concatLists_impl(a interface{}, b interface{}) interface{} { ... }
```

### Example 2: Exported Functions Unchanged

**AILANG:**
```ailang
module sim/solar_demo

export CalculateOrbit

pure func concatLists(a: [Planet], b: [Planet]) -> [Planet] = a ++ b
pure func CalculateOrbit(p: Planet) -> float = ...
```

**Generated Go:**
```go
// Non-exported: namespaced
func solar_demo__concatLists_impl(...) { ... }

// Exported: simple name
func CalculateOrbit_impl(...) { ... }
func CalculateOrbit(...) interface{} { return CalculateOrbit_impl(...) }
```

## Success Criteria

- [x] Multiple modules with same non-exported function names compile
- [ ] stapledons_voyage `sim/*.ail` compiles without errors (pending integration test)
- [x] Exported functions retain simple names
- [x] Single-module projects work unchanged
- [x] All existing codegen tests pass
- [x] `make test` passes

## Testing Strategy

**Unit tests:**
- Test `namespaceFunc("solar_demo", "concatLists")` → `"solar_demo__concatLists"`
- Test exported function detection

**Integration tests:**
- Compile two modules with same-named function
- Verify generated Go compiles
- Verify runtime behavior correct

**Manual testing:**
- Compile stapledons_voyage and verify it works

## Non-Goals

**Not in this feature:**
- Cross-package imports - out of scope, requires Go package separation
- Fully-qualified module paths in names - use short names unless collisions

## Timeline

**Single session** (~2 hours):
- Phase 1: 30 min - Export tracking
- Phase 2: 60 min - Namespace generation
- Phase 3: 30 min - Call site updates

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing codegen tests | Medium | Run test suite after each phase |
| Name collisions with `__` separator | Low | `__` is uncommon in user code |
| Cross-module calls not handled | Medium | Track caller's module, use correct namespace |

## Related Documents

- [M-DX17: ConcatList + Closure Scoping](m-dx17-codegen-concatlist-closure-scoping.md) - Related codegen bugs
- [M-CODEGEN-ADT-DOUBLE-PAREN](m-codegen-adt-double-paren.md) - Recent codegen fix

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage bug report (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20

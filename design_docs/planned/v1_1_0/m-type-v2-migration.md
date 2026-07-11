# M-TYPE-V2-MIGRATION: Complete TFunc → TFunc2 Type System Migration

**Status**: Planned
**Target**: v1.0.0
**Priority**: P1 (Medium)
**Estimated**: 2 days
**Dependencies**: None

## Problem Statement

AILANG's type system has two function type representations from an incomplete v1→v2 migration:

- **TFunc** (legacy): `Params []Type`, `Return Type`, `Effects []EffectType` — flat effect list, no kind tracking
- **TFunc2** (modern): `Params []Type`, `EffectRow *Row`, `Return Type` — row polymorphism, budgets, kind system

All new type inference creates `TFunc2`, but `TFunc` persists in 21 files. This causes **recurring bugs** where type switches handle only one variant, silently dropping through. Recent example: the `*types.TFunc2` case was missed in codegen, causing undefined symbol errors.

**Current State:**
- 1 active bug: `propagateTypeNameWithVisited()` has no `*TFunc2` case
- 2 constructors still creating `&TFunc{}`
- 2 duplicate helper functions (`mapTFuncWithVisited` / `mapTFunc2WithVisited`)
- ~10 test files using legacy `&TFunc{}`
- Cross-unification compatibility layer adds complexity

**Impact:**
- Every new type switch is a potential bug if developer forgets to handle both
- Duplicate code paths increase maintenance burden
- Bug pattern has recurred multiple times across the project

## Goals

**Primary Goal:** Eliminate TFunc entirely — single canonical function type representation

**Success Metrics:**
- TFunc struct definition deleted from codebase
- Zero type switches handling `*TFunc` — only `*TFunc2`
- Duplicate helpers consolidated (mapTFunc* → single function)
- Cross-unification compat layer removed
- All tests passing with TFunc2 only

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Delete TFunc entirely vs deprecate | Determines migration completeness | human | design | high |
| Effect conversion strategy ([]EffectType → *Row) | Affects all TFunc construction sites | agent | compile | med |
| Test migration approach (bulk vs incremental) | Risk of regression during migration | agent | compile | low |

### Design Freeze

- [x] Delete TFunc entirely (not deprecate) — it's internal, no external API
- [x] Convert all `[]EffectType` to `*Row` using existing `ElaborateEffectRow()` helper
- [ ] Whether to also migrate TVar→TVar2 in same sprint (scope question)

## Deferred Decisions

- Internal helper naming after consolidation — agent may choose
- Test fixture style (builder vs literal) — agent may choose
- Whether to add a linter rule preventing TFunc construction — human at review

## Solution Design

### Overview

Systematically replace all TFunc usages with TFunc2 across the codebase, then delete the TFunc type definition. The migration is safe because TFunc2 is a strict superset — any TFunc can be converted to TFunc2 by wrapping its `[]EffectType` into a `*Row`.

### Architecture

**Conversion pattern:**
```go
// Before: &TFunc{Params: ps, Return: r, Effects: effs}
// After:  &TFunc2{Params: ps, EffectRow: ElaborateEffectRow(effs), Return: r}
// Or for pure functions:
// After:  &TFunc2{Params: ps, EffectRow: nil, Return: r}
```

### Implementation Plan

**Phase 1: Fix Active Bug** (~1 hour)
- [ ] Add `case *TFunc2:` to `propagateTypeNameWithVisited()` in `internal/types/unification_core.go:293`
- [ ] Test with `make test`

**Phase 2: Convert Constructors** (~2 hours)
- [ ] Convert `safeSubstitute` in `internal/types/unification_substitution.go:108` to create TFunc2
- [ ] Convert `TFunc.Substitute()` method to return TFunc2 (or inline into remaining callers)
- [ ] Search for any other `&TFunc{` constructors and convert
- [ ] Test with `make test`

**Phase 3: Remove Compatibility Layer** (~2 hours)
- [ ] Inline `unifyTFunc()` logic from `internal/types/unification_types.go:98-146` into TFunc2-only path
- [ ] Remove TFunc cases from `unification_core.go` dispatcher
- [ ] Remove TFunc cases from `unification_equality.go`
- [ ] Test with `make test`

**Phase 4: Consolidate Helpers** (~1 hour)
- [ ] Merge `mapTFuncWithVisited()` and `mapTFunc2WithVisited()` in `internal/gen/golang/types.go` into single function
- [ ] Remove TFunc cases from `safe_string.go`, `traverse.go`, `typechecker_defaulting.go`
- [ ] Remove TFunc case from `pipeline_module_compile.go`
- [ ] Remove TFunc case from `codegen_expr_app.go`
- [ ] Test with `make test`

**Phase 5: Delete TFunc & Migrate Tests** (~2 hours)
- [ ] Delete `TFunc` struct definition and methods from `internal/types/types.go`
- [ ] Migrate test files to use `&TFunc2{}`:
  - `type_head_test.go`
  - `substitution_chain_test.go`
  - `safe_string_test.go`
  - `dictionary_test.go`
  - `traverse/traverse_test.go`
  - `smt/types_test.go`
  - codegen test files
- [ ] Test with `make test`

**Phase 6: Verify** (~1 hour)
- [ ] `make ci` — full CI verification
- [ ] `make verify-examples` — all examples still work
- [ ] Run eval suite to verify no regressions
- [ ] Update CHANGELOG.md

### Files to Modify

**Deleted code (TFunc removal):**
- `internal/types/types.go` (-30 LOC) — TFunc struct + methods

**Modified files:**
- `internal/types/unification_core.go` (+5/-15 LOC) — remove TFunc dispatch, fix propagate bug
- `internal/types/unification_types.go` (-50 LOC) — remove unifyTFunc compat layer
- `internal/types/unification_equality.go` (-15 LOC) — remove TFunc case
- `internal/types/unification_substitution.go` (+3/-3 LOC) — TFunc→TFunc2 in safeSubstitute
- `internal/types/safe_string.go` (-10 LOC) — remove TFunc case
- `internal/types/traverse/traverse.go` (-10 LOC) — remove TFunc case
- `internal/types/typechecker_defaulting.go` (-10 LOC) — remove TFunc case
- `internal/gen/golang/types.go` (-20 LOC) — consolidate mapTFunc helpers
- `internal/gen/golang/codegen_expr_app.go` (-10 LOC) — remove TFunc case
- `internal/pipeline/pipeline_module_compile.go` (-10 LOC) — remove TFunc case
- ~10 test files (+/-5 LOC each) — TFunc→TFunc2

**Net: ~-150 LOC removed**

## Examples

### Before: Type switch handling both variants
```go
case *TFunc:
    // handle legacy
case *TFunc2:
    // handle modern (nearly identical logic)
```

### After: Single case
```go
case *TFunc2:
    // one path, no duplication
```

## Success Criteria

- [ ] `TFunc` struct no longer exists in codebase
- [ ] `grep -r "TFunc[^2]" internal/types/` returns zero matches (excluding comments)
- [ ] All type switches use only `*TFunc2`
- [ ] `make ci` passes
- [ ] `make verify-examples` passes
- [ ] Eval suite shows no regressions
- [ ] CHANGELOG.md updated
- [ ] Net code reduction of ~150 LOC

## Testing Strategy

**Unit tests:** Migrate all existing TFunc tests to TFunc2 — same coverage, new type
**Integration tests:** `make test` after each phase (incremental migration)
**Verification:** `make ci`, `make verify-examples`, eval suite

## Non-Goals

- **TVar→TVar2 migration** — separate effort, different scope
- **Adding new type system features** — pure cleanup, no new capabilities
- **Changing external behavior** — internal refactor only

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Hidden TFunc usage in generated code | Medium | Search for TFunc in all `.go` files, not just `internal/types/` |
| Cross-unification removal breaks edge case | Medium | Test after each phase, keep unifyTFunc until Phase 3 passes |
| Test migration introduces false passes | Low | Run tests before AND after each file migration |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No behavioral change |
| A2: Replayability | 0 | No behavioral change |
| A3: Effect Legibility | +1 | Single effect representation (Row) is clearer |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Single type to verify, not two |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Simpler type system = easier for AI codegen |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Removes dual-path complexity |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** ✅ Proceed

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimizing for machine analysis (fewer types to handle)

## References

- TFunc definition: `internal/types/types.go:75-80`
- TFunc2 definition: `internal/types/types_v2.go:228-233`
- Row type: `internal/types/types_v2.go:61-68`
- Changelog note: "Type system migration incomplete" in v0.0-v0.2 foundation docs

## Future Work

- TVar→TVar2 migration (same pattern, separate doc)
- Remove other v1 type remnants if any exist
- Add linter rule to prevent re-introduction of legacy types

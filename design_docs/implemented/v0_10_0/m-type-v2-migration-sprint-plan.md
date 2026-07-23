# Sprint Plan: M-TYPE-V2-MIGRATION

## Summary
Eliminate the legacy TFunc type from the codebase, leaving TFunc2 as the single canonical function type. This removes a recurring source of bugs where type switches miss one variant.

**Duration:** 1 day (~4 hours)
**Dependencies:** None
**Risk Level:** Medium (core type system, but scope is small — 11 files in internal/types/ only)

## Current Status Analysis

### Scope (Revised from Design Doc)
The design doc estimated 21 files, but audit shows TFunc is **only in internal/types/** (11 .go files + test files). No gen/ or pipeline/ code references TFunc anymore.

**Production code with `case *TFunc:`:** 8 files, 13 switch cases
- `unification_core.go` (2 cases + propagate bug)
- `unification_types.go` (1 case — compat layer)
- `unification_equality.go` (1 case)
- `unification_substitution.go` (1 case + constructor)
- `typechecker_substitution.go` (2 cases)
- `typechecker_defaulting.go` (2 cases)
- `safe_string.go` (1 case)
- `normalize.go` (2 cases)

**Constructor sites (`&TFunc{`):** 2
- `types.go:135` (TFunc.Substitute)
- `unification_substitution.go:108` (safeSubstitute)

**Test files using `&TFunc{}`:** 4 files
- `safe_string_test.go`, `dictionary_test.go`, `substitution_chain_test.go`, `type_head_test.go`

### Velocity
Recent: ~500 LOC/day with heavy feature work. This is pure cleanup (~-100 LOC net), so faster.

## Proposed Milestones

### M1: FIX_PROPAGATE_BUG
**Goal:** Fix the active bug where `propagateTypeNameWithVisited()` has no TFunc2 case
**Estimated:** +10 LOC
**Duration:** 15 min

**Tasks:**
- Add `case *TFunc2:` to `propagateTypeNameWithVisited()` in `unification_core.go:293`
- Run `make test`

**Acceptance Criteria:**
- [ ] `propagateTypeNameWithVisited` handles both TFunc and TFunc2
- [ ] `make test` passes

### M2: CONVERT_CONSTRUCTORS
**Goal:** Convert all `&TFunc{}` constructors to `&TFunc2{}`
**Estimated:** +15/-15 LOC
**Duration:** 30 min

**Tasks:**
- Convert `TFunc.Substitute()` in `types.go:135` to return `&TFunc2{}`
- Convert `safeSubstitute` in `unification_substitution.go:108` to create `&TFunc2{}`
- Run `make test`

**Acceptance Criteria:**
- [ ] Zero `&TFunc{` constructors remain in production code
- [ ] `make test` passes

### M3: REMOVE_TFUNC_CASES
**Goal:** Remove all `case *TFunc:` from type switches (now dead code since nothing creates TFunc)
**Estimated:** -80 LOC
**Duration:** 1 hour

**Tasks:**
- Remove TFunc cases from `unification_core.go` (2 cases)
- Remove `unifyTFunc()` compat layer from `unification_types.go`
- Remove TFunc cases from `unification_equality.go`
- Remove TFunc case from `unification_substitution.go`
- Remove TFunc cases from `typechecker_substitution.go` (2 cases)
- Remove TFunc cases from `typechecker_defaulting.go` (2 cases)
- Remove TFunc case from `safe_string.go`
- Remove TFunc cases from `normalize.go` (2 cases)
- Run `make test` after each file

**Acceptance Criteria:**
- [ ] Zero `case *TFunc:` in production code
- [ ] `make test` passes after each file change

### M4: DELETE_TFUNC_AND_MIGRATE_TESTS
**Goal:** Delete TFunc struct definition, migrate test files
**Estimated:** -30 LOC (struct) + ~20 LOC test changes
**Duration:** 45 min

**Tasks:**
- Delete TFunc struct and methods from `types.go`
- Migrate `safe_string_test.go` to use `&TFunc2{}`
- Migrate `dictionary_test.go` to use `&TFunc2{}`
- Migrate `substitution_chain_test.go` to use `&TFunc2{}`
- Migrate `type_head_test.go` to use `&TFunc2{}`
- Run `make test`

**Acceptance Criteria:**
- [ ] TFunc struct no longer exists
- [ ] All tests pass with TFunc2
- [ ] `grep -r 'TFunc[^2]' internal/types/*.go` returns nothing (excluding comments/strings)

### M5: VERIFY_AND_DOCUMENT
**Goal:** Full verification and changelog update
**Estimated:** ~10 LOC (changelog)
**Duration:** 30 min

**Tasks:**
- Run `make ci`
- Run `make verify-examples`
- Update CHANGELOG.md
- Move design doc to implemented

**Acceptance Criteria:**
- [ ] `make ci` passes
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated
- [ ] Net LOC reduction confirmed

## Success Metrics
- TFunc struct deleted from codebase
- All 13 type switch cases consolidated to TFunc2-only
- `make ci` passes
- `make verify-examples` passes
- Net ~100 LOC removed

## Risks
- **TFunc.Substitute() callers expect TFunc back** — Mitigation: TFunc2 implements Type interface, so return type is compatible
- **Test assertions check for *TFunc specifically** — Mitigation: Update assertions to *TFunc2

## Notes
- Scope is SMALLER than design doc estimated (11 files not 21, internal/types/ only)
- gen/ and pipeline/ already migrated — no TFunc references there
- This is a safe, incremental cleanup — each milestone leaves tests green

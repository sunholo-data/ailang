# Sprint Plan: M-STD-MAP

## Summary

Implement `std/map` (hashmap type with O(1) lookup) and fill `std/array` gaps (`empty` builtin) to unblock XLSX parsing of files >5 MB. Phase 1 only — XLSX unblocker, not full refinement.

**Duration:** 2 days (estimated 6-8 hours active work)
**Dependencies:** None
**Risk Level:** Medium (type system changes touch 13 files)
**Design Doc:** `design_docs/planned/v0_11_0/m-std-map-and-array-gaps.md`

## Current Status Analysis

### Completed Recently
- cons (::) expression: ~570 LOC in 1 day
- bitwise operators: ~1422 LOC in 1 day
- serve-api features: ~1046 LOC in 1 day
- JWT/crypto support: ~1827 LOC in 1 day

### Velocity
- Recent average: ~500-700 LOC/day (feature work with tests)
- Estimated capacity: ~800 LOC for this sprint

### Remaining from Design Doc
- Phase 1: std/map type + builtins + array.empty (~800 LOC)
- Phase 2: refinements (deferred — append, compound keys, HAMT)

## Proposed Milestones

### Milestone 1: M1_MAP_TYPE — MapValue + TMap + Type System Integration

**Goal:** Add `MapValue` eval type, `TMap` type system node, `mapKey()` canonical encoder, and update all 13 type switch locations so the type checker fully understands Map[k,v].

**Estimated:** ~200 LOC implementation + ~100 LOC tests = ~300 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `MapValue` struct to `internal/eval/value.go` with `Type()`, `String()`, `Lookup()`, `Insert()`, `Remove()`, `Size()`
2. Add `mapKey()` canonical encoder (type-prefixed: `i:`, `s:`, `b:`)
3. Add `TMap` to `internal/types/types.go` with `String()`, `Equals()`, `Substitute()`
4. Add `HeadMap` to `internal/types/type_head.go`
5. Update all 13 type switch locations (grep `case *TArray:`, add parallel `case *TMap:`)
   - `types/inference_helpers.go` — `collectFreeTypeVars` (Key+Value)
   - `types/typechecker_defaulting.go` — `collectFreeVarsWithVisited` (Key+Value)
   - `types/unification_equality.go` — `safeEqualsWithVisited`
   - `types/unification_core.go` — `Unify` + `propagateTypeName` (2 locations)
   - `types/unification_substitution.go` — `safeSubstitute`
   - `types/unification_occurs.go` — `occursWithVisited` (Key+Value)
   - `types/safe_string.go` — `safeTypeStringWithDepth`
   - `types/normalize.go` — `NormalizeTypeName` + `IsGroundType` (2 locations)
   - `types/type_head.go` — `TypeHead`
   - `types/types_v2.go` — `Kind` -> Star
   - `types/typechecker_substitution.go` — `propagateTypeNameRecursively`
   - `types/traverse/traverse.go` — `Children` -> [Key, Value]
   - `pipeline/pipeline_module_compile.go` — handle TMap
6. Add `TMap`/`TApp("Map",...)` unification (follow TArray/TApp("Array",...) pattern in `unification_types.go`)
7. Write type system tests for TMap

**Acceptance Criteria:**
- [ ] `MapValue` with `mapKey()` encoder handles int/string/bool keys
- [ ] `MapValue.String()` outputs deterministic sorted representation
- [ ] `TMap` type node exists with String/Equals/Substitute
- [ ] All 13 type switch locations updated (verified by grep)
- [ ] `TMap` unifies with `TApp("Map", [k, v])`
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Missing a type switch location → silent type bugs. Mitigation: grep-verify count matches before/after.

### Milestone 2: M2_MAP_BUILTINS — Builtins + Wrappers + Examples

**Goal:** Implement all 10 map builtins, create `std/map.ail` wrapper, add `_array_empty` builtin, create working example.

**Estimated:** ~350 LOC implementation + ~100 LOC tests + ~50 LOC wrappers/examples = ~500 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/builtins/map.go` with 10 builtins:
   - `_map_empty`, `_map_insert`, `_map_lookup`, `_map_member`, `_map_remove`
   - `_map_size`, `_map_keys`, `_map_values`, `_map_from_list`, `_map_to_list`
2. Add type builder helper for Map types (follow Array pattern)
3. Create `std/map.ail` wrapper with cost model docs
4. Add `_array_empty` builtin to `internal/builtins/array.go`
5. Update `std/array.ail` with `empty` function
6. Create `examples/runnable/map_basic.ail` demonstrating:
   - Build map from list of tuples
   - Lookup, member, size
   - Keys/values export
7. Write builtin tests for map operations

**Acceptance Criteria:**
- [ ] All 10 map builtins registered under `std/map` module
- [ ] `std/map.ail` wrapper with accurate cost model comments
- [ ] `std/array.ail` has `empty` function
- [ ] `examples/runnable/map_basic.ail` runs successfully
- [ ] `fromList` builds map mutably (O(n)), returns immutable result
- [ ] `keys`/`values`/`toList` produce deterministic sorted output
- [ ] Unsupported key types return clear error
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make verify-examples` passes

**Risks:**
- Type builder may not support Map[k,v] two-parameter construction. Mitigation: check existing TArray builder pattern first.
- TApp("Map",...) resolution in module imports may need work. Mitigation: test import round-trip early.

## Success Metrics
- `make test` passes with no regressions
- `make lint` clean
- `make verify-examples` passes including new map example
- `std/map` importable and functional end-to-end
- `std/array` has `empty` function
- Total: ~800 LOC (implementation + tests + wrappers + examples)

## Dependencies
- None — this is greenfield work building on existing patterns (TArray, array builtins)

## Open Questions
- None — design doc feedback incorporated, all decisions made

## Notes
- Phase 2 work (append, compound keys, HAMT, map literals) explicitly deferred
- XLSX integration testing depends on downstream parser adoption — not in this sprint
- Follow existing patterns: `array.go` for builtins, `std/array.ail` for wrapper, `TArray` for type system

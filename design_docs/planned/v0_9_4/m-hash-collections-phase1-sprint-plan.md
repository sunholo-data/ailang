# Sprint Plan: M-HASH-COLLECTIONS Phase 1

## Summary

Replace O(n^2) list-set builtins with O(n) canonical-key-accelerated implementations, eliminate `reflect.DeepEqual` from `valuesEqual`, and unblock DocParse Moby Dick Jaccard (currently OOM).

**Duration:** 2 days (estimated 4-5 hours active implementation)
**Dependencies:** None — Phase 1 is self-contained, modifies only builtins internals
**Risk Level:** Low — same API, faster internals. No type system or parser changes.
**Design Doc:** `design_docs/implemented/v0_9_2/m-hash-collections.md`

## Current Status Analysis

### Completed Recently
- M-CODEGEN-REGISTRY-ONLY + COMPILE-GATE: ~300 LOC in 1 day
- M-CODEGEN-SUSTAIN M1-M5: ~1500 LOC in 4 days
- M-CODEGEN-STDLIB-BUILTINS: ~1200 LOC in 3 days

### Velocity
- Recent average: ~400 LOC/day (implementation + tests)
- This sprint estimated: ~350 LOC total (well within 1-day capacity, but 2 days for thorough testing + verification)

### Codebase State
- `valuesEqual` in `list.go:380-404` — 25 lines, uses `reflect.DeepEqual` fallback at line 403
- `list_set.go` — 312 lines, all 5 set operations use nested O(n^2) loops
- Three `valuesEqual` implementations: builtins (DeepEqual), SimpleEvaluator (returns false), TypedEvaluator (returns false)
- Existing tests in `list_test.go` cover dedup, intersect, union, difference

## Proposed Milestones

### M1: canonicalKey + valuesEqual structural fix (~150 LOC)

**Goal:** Implement `canonicalKey()` canonical structural encoding and replace `reflect.DeepEqual` in `valuesEqual()` with recursive structural comparison.

**Estimated:** ~80 LOC implementation + ~70 LOC tests = ~150 LOC
**Duration:** Day 1 (morning)

**Tasks:**
1. Create `internal/builtins/canonical_key.go`:
   - `canonicalKey(v eval.Value) string` — type-tagged canonical encoding
   - Handles: Int, Float, String, Bool, Unit, List, Record (sorted keys), Tagged, Tuple, Array, Bytes
   - Float uses `fmt.Sprintf("%g")` for Phase 1 (full normalization deferred to Phase 2)
   - Default case: `"x:" + v.String()` for unknown types (safe fallback)

2. Replace `valuesEqual` in `list.go:380-404`:
   - Remove `reflect.DeepEqual` fallback
   - Add recursive structural comparison for Record, List, Tagged, Tuple, Array
   - Remove `reflect` import if no longer needed

3. Create `internal/builtins/canonical_key_test.go`:
   - Determinism: same value produces same key across 20 runs (`-count=20`)
   - Collision-free: different type-tagged values produce different keys
   - Nested structures: records with nested lists, ADTs with nested records
   - Edge cases: empty list, empty record, deeply nested (3+ levels)

**Acceptance Criteria:**
- [ ] `canonicalKey` produces identical output for identical values
- [ ] `canonicalKey` produces different output for different-typed equal-looking values (e.g., int 1 vs string "1")
- [ ] `valuesEqual` no longer imports `reflect`
- [ ] `valuesEqual` handles Records, Lists, Tagged values structurally
- [ ] All existing tests pass (no behavior change)
- [ ] Tests pass with `-count=20`
- [ ] `make lint` clean

**Risks:**
- Edge case in canonical encoding for deeply nested recursive structures — Mitigation: depth limit or stack check
- Float canonicalization surprises (`-0.0`, `NaN`) — Mitigation: Phase 1 uses `%g` which is sufficient for string-heavy DocParse workload

### M2: Hash-accelerated set operations (~120 LOC)

**Goal:** Replace all O(n^2) set operations in `list_set.go` with O(n) map-backed implementations using `canonicalKey`.

**Estimated:** ~60 LOC implementation (net reduction) + ~60 LOC benchmarks/tests = ~120 LOC
**Duration:** Day 1 (afternoon)

**Tasks:**
1. Rewrite `listDedupImpl` (list_set.go:105-124):
   - Use `map[string]bool` with `canonicalKey` for seen-tracking
   - O(n) instead of O(n^2)

2. Rewrite `listIntersectImpl` (list_set.go:162-191):
   - Build `map[string]bool` from list2
   - Iterate list1, check map membership
   - O(n+m) instead of O(n*m)

3. Rewrite `listUnionImpl` (list_set.go:222-258):
   - Build `map[string]bool` seen-set
   - Add from list1, then list2, skipping seen
   - O(n+m) instead of O(n*m)

4. Rewrite `listDifferenceImpl` (list_set.go:289-312):
   - Build `map[string]bool` from list2
   - Include list1 elements not in map
   - O(n+m) instead of O(n*m)

5. Update `listMemberImpl` (list_set.go:57-69):
   - Keep O(n) linear scan (already optimal for single lookup)
   - But use new structural `valuesEqual` (no reflect.DeepEqual)

**Acceptance Criteria:**
- [ ] All 5 set operations use `canonicalKey` internally (except member which uses valuesEqual)
- [ ] `dedup([1,2,1,3,2])` returns `[1,2,3]` (preserves first-occurrence order)
- [ ] `intersect([1,2,3], [2,3,4])` returns `[2,3]`
- [ ] `union([1,2], [2,3])` returns `[1,2,3]`
- [ ] `difference([1,2,3], [2])` returns `[1,3]`
- [ ] All existing tests pass unchanged
- [ ] `make lint` clean

**Risks:**
- Subtle behavior difference with records/ADTs that were compared by reflect.DeepEqual — Mitigation: existing tests cover this, add record-specific test cases

### M3: Benchmarks + verification + docs (~80 LOC)

**Goal:** Add benchmarks proving O(n) scaling, verify DocParse-scale workloads complete, update CHANGELOG.

**Estimated:** ~60 LOC benchmarks + ~20 LOC docs = ~80 LOC
**Duration:** Day 2

**Tasks:**
1. Add benchmarks to `canonical_key_test.go`:
   - `BenchmarkDedup100`, `BenchmarkDedup1000`, `BenchmarkDedup5000`, `BenchmarkDedup10000`
   - `BenchmarkIntersect1000`, `BenchmarkIntersect5000`
   - `BenchmarkCanonicalKeyRecord` (nested record encoding speed)
   - Verify O(n) scaling (10x input = ~10x time, not 100x)

2. Integration verification:
   - Run `make test` — full suite pass
   - Run `make verify-examples` — all examples still work
   - Run `make lint` — clean

3. Documentation:
   - Update CHANGELOG.md with Phase 1 entry
   - Add example file: `examples/runnable/set_operations.ail` showing dedup/intersect/union/difference
   - Update design doc status

**Acceptance Criteria:**
- [ ] `BenchmarkDedup5000` completes in <5ms
- [ ] `BenchmarkIntersect5000` completes in <5ms
- [ ] O(n) scaling confirmed: dedup(10000) / dedup(1000) ratio < 15x
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated
- [ ] Example file created and verified
- [ ] Design doc Phase 1 checkboxes marked complete

**Risks:**
- canonicalKey on large strings may be slow due to string concatenation — Mitigation: benchmark will catch this, can optimize with strings.Builder pooling

## Success Metrics

- **Performance**: dedup(5000) < 5ms (from ~seconds)
- **Correctness**: All existing tests pass, no behavior change
- **Determinism**: `-count=20` tests pass
- **Code quality**: `reflect.DeepEqual` removed from builtins, `make lint` clean
- **Test coverage**: New tests for canonicalKey, benchmarks for scaling
- **Documentation**: CHANGELOG updated, example file created

## Files Changed

| File | Action | LOC |
|------|--------|-----|
| `internal/builtins/canonical_key.go` | **New** | ~80 |
| `internal/builtins/canonical_key_test.go` | **New** | ~130 |
| `internal/builtins/list.go` | **Modify** | ~-10/+30 |
| `internal/builtins/list_set.go` | **Modify** | ~-80/+60 |
| `examples/runnable/set_operations.ail` | **New** | ~20 |
| `CHANGELOG.md` | **Modify** | ~+15 |
| **Total** | | ~350 LOC |

## Dependencies
- None — Phase 1 is self-contained

## Open Questions
- None — all design questions resolved in design doc review

## Notes
- Phase 1 preserves the exact AILANG API — same function names, same behavior, just faster
- Phase 2 (first-class Set/Map types) is a separate sprint targeting v0.10.0
- The `canonicalKey` name was chosen over `hashKey` per reviewer feedback — it's a canonical structural encoding, not a hash
- Float handling in `canonicalKey` uses `%g` format for Phase 1; full NaN/-0.0 normalization deferred to Phase 2

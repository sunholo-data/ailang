# Sprint Plan: M-STDLIB-GAPS

**Sprint ID**: M-STDLIB-GAPS
**Duration**: 0.5 days (~2-3 hours)
**Risk Level**: Low
**Design Doc**: [m-stdlib-gaps.md](m-stdlib-gaps.md)

---

## Sprint Summary

**Goal**: Add missing stdlib functions that agents commonly reimplement, reducing agent turns and compile errors.

**Key Deliverables**:
1. List functions: `nth`, `last`, `exists`, `findIndex` in std/list.ail
2. String function: `join` in std/string.ail
3. JSON array helpers: `filterStrings`, `filterNumbers`, `allStrings`, `allNumbers` + convenience wrappers in std/json.ail
4. Prompt v0.6.5 documenting all new functions
5. Example files demonstrating usage

---

## Current Status

| Item | Status |
|------|--------|
| `std/list.ail` | Has `length`, `head`, `tail`, `map`, `filter`, `foldl`, etc. Missing: `nth`, `last`, `exists`, `findIndex` |
| `std/string.ail` | Has `split`, `trim`, `find`, etc. Missing: `join` |
| `std/json.ail` | Has `getString`, `getNumber`, etc. (v0.6.4). Missing: typed array extraction |
| Prompt v0.6.4 | Active - documents JSON convenience functions |
| `config_file_parser` benchmark | Currently fails (agents can't convert `[Json]` to `[string]`) |

---

## Milestones

### M1: Add List Functions (~50 LOC)

**Tasks**:
1. Add `nth[a](xs: [a], idx: int) -> Option[a]` - 0-based index access
2. Add `last[a](xs: [a]) -> Option[a]` - get last element
3. Add `exists[a](p: (a) -> bool, xs: [a]) -> bool` - predicate check
4. Add `findIndex[a](p: (a) -> bool, xs: [a]) -> Option[int]` - find index by predicate
5. Add internal helper `findIndexHelper`

**Acceptance Criteria**:
- [ ] All 4 exported functions compile without errors
- [ ] `nth([1,2,3], 1)` returns `Some(2)`
- [ ] `nth([1,2,3], -1)` returns `None`
- [ ] `last([1,2,3])` returns `Some(3)`
- [ ] `exists(\x. x > 2, [1,2,3])` returns `true`

**Files to Change**:
- `std/list.ail` (+50 LOC)

---

### M2: Add String Join (~15 LOC)

**Tasks**:
1. Add `join(delimiter: string, xs: [string]) -> string`

**Acceptance Criteria**:
- [ ] `join(", ", ["a", "b", "c"])` returns `"a, b, c"`
- [ ] `join("-", [])` returns `""`
- [ ] `join("-", ["x"])` returns `"x"`

**Files to Change**:
- `std/string.ail` (+15 LOC)

---

### M3: Add JSON Array Functions (~70 LOC)

**Tasks**:
1. Add `filterStrings(xs: [Json]) -> [string]` - permissive extraction
2. Add `filterNumbers(xs: [Json]) -> [float]` - permissive extraction
3. Add `allStrings(xs: [Json]) -> Option[[string]]` - strict extraction
4. Add `allNumbers(xs: [Json]) -> Option[[float]]` - strict extraction
5. Add convenience wrappers:
   - `getStringArray`, `getStringArrayStrict`, `getStringArrayOrEmpty`
   - `getNumberArray`, `getNumberArrayStrict`, `getNumberArrayOrEmpty`

**Acceptance Criteria**:
- [ ] `filterStrings([JString("a"), JNumber(1.0), JString("b")])` returns `["a", "b"]`
- [ ] `allStrings([JString("a"), JNumber(1.0)])` returns `None`
- [ ] `allStrings([JString("a"), JString("b")])` returns `Some(["a", "b"])`
- [ ] `getStringArrayOrEmpty(json, "missing")` returns `[]`

**Files to Change**:
- `std/json.ail` (+70 LOC)

---

### M4: Create Prompt v0.6.5 (~40 LOC)

**Tasks**:
1. Copy v0.6.4.md to v0.6.5.md
2. Add list functions documentation
3. Add string join documentation
4. Add JSON typed array extraction documentation
5. Register v0.6.5 in versions.json
6. Rebuild ailang (`make quick-install`)

**Acceptance Criteria**:
- [ ] `prompts/v0.6.5.md` exists with all new functions documented
- [ ] `versions.json` has v0.6.5 entry with correct hash
- [ ] `ailang prompt` shows v0.6.5

**Files to Change**:
- `prompts/v0.6.5.md` (new, copy from v0.6.4 +40 LOC)
- `prompts/versions.json` (+10 LOC)

---

### M5: Example Files + Benchmark Validation

**Tasks**:
1. Create `examples/runnable/list_helpers.ail`
2. Create `examples/runnable/json_array_extraction.ail`
3. Run `config_file_parser` benchmark with Claude Haiku
4. Document results

**Acceptance Criteria**:
- [ ] Both example files compile and run successfully
- [ ] `config_file_parser` benchmark passes (currently fails)
- [ ] Document turns count in sprint notes

**Files to Create**:
- `examples/runnable/list_helpers.ail` (~30 LOC)
- `examples/runnable/json_array_extraction.ail` (~30 LOC)

---

## Task Breakdown

| Task | Est. | Milestone |
|------|------|-----------|
| Add `nth` to std/list | 10 min | M1 |
| Add `last` to std/list | 5 min | M1 |
| Add `exists` to std/list | 5 min | M1 |
| Add `findIndex` to std/list | 10 min | M1 |
| Test list functions | 5 min | M1 |
| Add `join` to std/string | 5 min | M2 |
| Test join | 5 min | M2 |
| Add `filterStrings`, `filterNumbers` | 15 min | M3 |
| Add `allStrings`, `allNumbers` | 15 min | M3 |
| Add convenience wrappers | 15 min | M3 |
| Test JSON functions | 10 min | M3 |
| Create v0.6.5 prompt | 15 min | M4 |
| Register prompt version | 5 min | M4 |
| Create list_helpers.ail example | 10 min | M5 |
| Create json_array_extraction.ail example | 10 min | M5 |
| Run config_file_parser benchmark | 5 min | M5 |
| **TOTAL** | **~2.5 hours** | |

---

## Success Metrics

| Metric | Before | Target | How to Measure |
|--------|--------|--------|----------------|
| `config_file_parser` | Fails | Passes | Benchmark run |
| List helper functions | 0 | 4 | Count in std/list.ail |
| JSON array functions | 0 | 10 | Count in std/json.ail |
| Example files | 0 | 2 | Count in examples/runnable/ |

---

## Dependencies

- v0.6.4 prompt and JSON helpers (already complete)
- `std/option` (Some, None) - already available

## Risks

| Risk | Mitigation |
|------|------------|
| Type inference with polymorphic list functions | Test with multiple types in examples |
| `[Json]` vs `List[Json]` notation | Design doc specifies `[Json]` - follow consistently |

---

## Post-Sprint

After completion:
1. Update design doc status to "Implemented"
2. Move to `design_docs/implemented/v0_6_5/`
3. Consider full baseline run with v0.6.5

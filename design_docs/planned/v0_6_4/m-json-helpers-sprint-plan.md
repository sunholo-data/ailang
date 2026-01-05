# Sprint Plan: M-JSON-HELPERS

**Sprint ID**: M-JSON-HELPERS
**Duration**: 1 day (~4 hours)
**Risk Level**: Low
**Design Doc**: [m-json-helpers.md](m-json-helpers.md)

---

## Sprint Summary

**Goal**: Add 6 convenience functions to `std/json` that reduce nested Option matching when extracting typed values from JSON objects.

**Key Deliverables**:
1. `getString`, `getNumber`, `getInt`, `getBool`, `getArray`, `getObject` in std/json
2. Updated prompt v0.6.4 documenting the helpers
3. Validation via `json_parse` benchmark (target: <25 turns, down from 47)

---

## Current Status

| Item | Status |
|------|--------|
| `std/json.ail` | ✅ Has `get`, `asString`, `asNumber`, `asBool`, `asArray` |
| `std/math.ail` | ✅ Has `floatToInt` (added in v0.6.3) |
| Prompt v0.6.3 | ✅ Documents stdlib functions |
| `json_parse` benchmark | ⚠️ 47 turns with v0.6.3 |

---

## Milestones

### M1: Add Helper Functions to std/json (~60 LOC)

**Estimated Time**: 1.5 hours

**Tasks**:
1. Add import for `std/math (floatToInt)` at top of std/json.ail
2. Add `getString(obj: Json, key: string) -> Option[string]`
3. Add `getNumber(obj: Json, key: string) -> Option[float]`
4. Add `getInt(obj: Json, key: string) -> Option[int]`
5. Add `getBool(obj: Json, key: string) -> Option[bool]`
6. Add `getArray(obj: Json, key: string) -> Option[List[Json]]`
7. Add `getObject(obj: Json, key: string) -> Option[Json]`

**Acceptance Criteria**:
- [ ] All 6 functions compile
- [ ] Quick test: Parse `{"name":"Alice","age":30}`, extract both fields
- [ ] No circular import issues with std/math

**Files to Change**:
- `std/json.ail` (+60 LOC)

---

### M2: Update Prompt v0.6.4 (~20 LOC)

**Estimated Time**: 30 minutes

**Tasks**:
1. Create `prompts/v0.6.4.md` from v0.6.3
2. Add JSON convenience functions documentation
3. Add import reminder warning
4. Register in `prompts/versions.json`
5. Rebuild ailang (`make quick-install`)

**Acceptance Criteria**:
- [ ] `ailang prompt` shows v0.6.4
- [ ] JSON helpers documented in prompt
- [ ] Import reminder visible

**Files to Change**:
- `prompts/v0.6.4.md` (new, copy from v0.6.3 +20 LOC)
- `prompts/versions.json` (+15 LOC)

---

### M3: Validate with Benchmark (~15 min)

**Estimated Time**: 15 minutes

**Tasks**:
1. Run `json_parse` benchmark with Haiku
2. Verify turns reduced (target: <25, was 47)
3. Optionally run `config_file_parser` to check import reminder

**Command**:
```bash
mkdir -p eval_results/test_json_helpers
ailang eval-suite -agent -benchmarks json_parse -models claude-haiku-4-5 -langs ailang -output eval_results/test_json_helpers
```

**Acceptance Criteria**:
- [ ] `json_parse` passes
- [ ] Turns < 25 (improvement from 47)

---

## Task Breakdown

| Task | Est. | Milestone |
|------|------|-----------|
| Add std/math import to std/json | 5 min | M1 |
| Implement getString | 10 min | M1 |
| Implement getNumber | 10 min | M1 |
| Implement getInt | 15 min | M1 |
| Implement getBool | 10 min | M1 |
| Implement getArray | 10 min | M1 |
| Implement getObject | 10 min | M1 |
| Test manually | 10 min | M1 |
| Create v0.6.4 prompt | 15 min | M2 |
| Add import reminder | 10 min | M2 |
| Register prompt version | 5 min | M2 |
| Rebuild ailang | 2 min | M2 |
| Run json_parse benchmark | 2 min | M3 |
| Analyze results | 10 min | M3 |
| **TOTAL** | **~2 hours** | |

---

## Success Metrics

| Metric | Before | Target | How to Measure |
|--------|--------|--------|----------------|
| `json_parse` turns | 47 | <25 | Benchmark run |
| JSON helper functions | 0 | 6 | Count in std/json.ail |
| Import reminder in prompt | No | Yes | `ailang prompt \| grep -i import` |

---

## Dependencies

- `std/math.floatToInt` (already available from v0.6.3)

## Risks

| Risk | Mitigation |
|------|------------|
| Circular import (std/json → std/math) | std/math has no dependencies, safe |
| Agent still forgets imports | Add explicit reminder in prompt |

---

## Post-Sprint

After completion:
1. Update design doc status to "Implemented"
2. Move to `design_docs/implemented/v0_6_4/`
3. Consider full baseline run with v0.6.4

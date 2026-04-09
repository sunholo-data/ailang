# Sprint Plan: M-BYTECODE-STDLIB-BUILTINS

**Goal**: Wire all pure stdlib builtins (~77) to the VM's `OpBuiltinCall` dispatch, dropping docparse EvalOnly from 220/1129 toward ~140/1129 (effectful + HOF + cascade remain).

**Duration**: 1 day (3-4 hours active work)
**Risk Level**: Low — mechanical pattern, all implementations exist in evaluator
**Design Doc**: `design_docs/implemented/v0_11_0/m-bytecode-vm.md` §18.6

---

## Context

M-BYTECODE-PHASE2E wired 4 pure builtins. Of the remaining ~126 unwired builtins:
- **~77 are pure** — no function callbacks, no effects, pure Go ops. These are mechanical to wire.
- **~6 are HOFs** — need function callback support (`FnCaller`/`FnCallerN`). Deferred.
- **~56 are effectful** — need capability checking. Deferred to M-BYTECODE-EFFECTS.

All pure builtins follow the same `OpBuiltinCall` wiring pattern. The work is mechanical: add name to compiler table, add handler to VM table. Each handler is ~5-15 lines.

---

## Milestones

### M1: WIRE_STRING_OPS — Wire 22 pure string builtins (~200 LOC)

**Builtins**: `__str_split`, `__str_join`, `__str_trim`, `__str_find`, `__str_slice`, `__str_replace`, `__str_startsWith`, `__str_endsWith`, `__str_lower`, `__str_upper`, `__str_len`, `__str_compare`, `__str_eq`, `__str_chars`, `__str_words`, `__str_charAt`, `__str_charCode`, `__str_decodeQP`, `__str_startsWithIC`, `__str_splitAny`, `__str_replaceMany`, `__escapeXml`

**Excludes** (HOFs needing FnCaller): `__str_foldChars`, `__str_foldSlices`, `__str_mapSlicesJoin`

**Acceptance Criteria**:
- All 22 string builtins no longer appear in EvalOnly reasons on docparse
- `make test` passes
- `make lint` passes
- Parity harness: no regressions from 129 MATCH

### M2: WIRE_MATH_CONVERSION — Wire 21 math + 6 type conversion builtins (~150 LOC)

**Builtins**: `__math_E`, `__math_PI`, `__math_abs_Float`, `__math_abs_Int`, `__math_acos`, `__math_asin`, `__math_atan`, `__math_atan2`, `__math_ceil`, `__math_cos`, `__math_exp`, `__math_floor`, `__math_log`, `__math_log10`, `__math_pow`, `__math_round`, `__math_sin`, `__math_sqrt`, `__math_tan`, `_mod_Int`, `_floatToInt`, `__float_to_int`, `__int_to_float`, `__stringToInt`, `__stringToFloat`, `__string_intToStr`, `__string_floatToStr`

**Acceptance Criteria**:
- All 27 builtins no longer appear in EvalOnly reasons on docparse
- `make test` passes, `make lint` passes
- Parity: no regressions

### M3: WIRE_COLLECTIONS — Wire 6 pure list builtins (~80 LOC)

**Builtins**: `__list_member`, `__list_dedup`, `__list_difference`, `__list_intersect`, `__list_union`, `__list_nth`

**Deferred** (no TagMap in VM value system): `__map_empty`, `__map_from_list`, `__map_insert`, `__map_keys`, `__map_lookup`, `__map_member`, `__map_remove`, `__map_size`, `__map_to_list`, `__map_values`

**Deferred** (no TagBytes in VM value system): `__bytes_concat`, `__bytes_concat_list`, `__bytes_from_base64`, `__bytes_from_base64url`, `__bytes_from_ints`, `__bytes_from_string`, `__bytes_length`, `__bytes_slice`, `__bytes_to_base64`, `__bytes_to_string`

**Acceptance Criteria**:
- All 6 list builtins no longer appear in EvalOnly on docparse
- Map and bytes builtins documented as deferred
- `make test` passes, `make lint` passes
- Parity: no regressions

### M4: BENCHMARK_AND_CLOSE — Re-benchmark and close (~20 LOC)

**Dependencies**: M1 + M2 + M3

**Acceptance Criteria**:
- docparse EvalOnly count recorded (target: significant reduction from 220)
- Parity: ≥ 129 MATCH, 2 NON_DET, 6 EVAL_SKIP
- Benchmark numbers on 10MB DOCX recorded in design doc
- CHANGELOG updated
- Design doc §18 updated with results

---

## Estimated LOC

| Milestone | Implementation | Tests | Total |
|---|---:|---:|---:|
| M1 (string) | 150 | 50 | 200 |
| M2 (math/conv) | 100 | 50 | 150 |
| M3 (list only) | 60 | 20 | 80 |
| M4 (benchmark) | 20 | 0 | 20 |
| **Total** | **330** | **120** | **450** |

## Risks

- **Confirmed**: Map builtins stay EvalOnly — VM has no `TagMap` value type. Checked `internal/bytecode/value.go`.
- **Confirmed**: Bytes builtins stay EvalOnly — VM has no `TagBytes` value type.
- **Low**: String builtins that return lists (`__str_chars`, `__str_words`, `__str_split`) need `bytecode.NewList` — this works, confirmed by existing `builtinListTail`.

## What's NOT in Scope

- **HOF builtins** (6): `__list_map`, `__list_filter`, `__list_foldl`, `__str_foldChars`, `__str_foldSlices`, `__str_mapSlicesJoin` — need `OpCallClosure` support in VM
- **Effectful builtins** (56): IO, FS, Net, XML, JSON, Zip, AI, Clock, Env — need `OpEffectCall` with capability checking
- **XML constructors** (3): `__xmlComment`, `__xmlElement`, `__xmlText` — categorize later

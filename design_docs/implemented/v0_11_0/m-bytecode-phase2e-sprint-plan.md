# Sprint Plan: M-BYTECODE-PHASE2E

**Goal**: Wire the 4 remaining Phase 2E builtins to VM opcodes, dropping docparse EvalOnly from 247/1129 to ~3/1129, and re-benchmark ailang-parse on 10MB DOCX.

**Duration**: 1 day (2-3 hours active work)
**Risk Level**: Low — established wiring pattern, implementations exist in evaluator
**Design Doc**: `design_docs/implemented/v0_11_0/m-bytecode-vm.md` §18.5

---

## Context

M-BYTECODE-MULTIMODULE resolved cross-module name resolution. The remaining EvalOnly prototypes are ALL caused by 4 unwired builtins:

| Builtin | Direct EvalOnly | Cascade | Category |
|---|---:|---:|---|
| `_concat_List` | 13 | ~100 | List concatenation |
| `_not_Bool` | 7 | ~60 | Boolean negation |
| `_intToFloat` | 7 | ~40 | Type conversion |
| `__list_length` | 3 | ~17 | List length (alias) |
| **Total** | **30** | **~217** | |

## Wiring Pattern

Two tables must stay synchronized:

1. **Compiler** (`internal/bytecode/compiler/builtins.go:23`): `BuiltinTable []string` — name at index N
2. **VM** (`internal/vm/builtins.go:24`): `BuiltinTable []BuiltinFunc` — handler at index N

Handler signature: `func(args []bytecode.Value) (bytecode.Value, error)`

Currently 6 builtins wired (indices 0-5): `_show`, `_len`, `_list_get`, `_list_tail`, `_concat_String`, `_record_get`.

---

## Milestones

### M1: WIRE_TRIVIAL_BUILTINS — Wire `_not_Bool`, `_intToFloat`, `__list_length` (~60 LOC)

**Rationale**: These 3 are trivial one-liners. Wire them together.

**Implementation**:
- Add 3 entries to compiler `BuiltinTable` (indices 6, 7, 8)
- Add 3 handler functions to VM `BuiltinTable`:
  - `builtinNotBool`: `!args[0].Bool()` → `NewBool`
  - `builtinIntToFloat`: `float64(args[0].Int())` → `NewFloat`
  - `builtinListLength`: `len(args[0].AsList())` → `NewInt` (alias for `_len` on lists)
- Note: `OpNot` opcode exists but `OpBuiltinCall` dispatch is simpler and consistent

**Acceptance Criteria**:
- `_not_Bool`, `_intToFloat`, `__list_length` no longer appear in `ailang disasm` EvalOnly reasons
- `make test` passes
- `make lint` passes
- Parity harness: no regressions from 129 MATCH baseline

### M2: WIRE_CONCAT_LIST — Wire `_concat_List` (~40 LOC)

**Rationale**: Slightly more complex — must handle list value allocation.

**Implementation**:
- Add `_concat_List` to compiler `BuiltinTable` (index 9)
- Add `builtinConcatList` handler to VM:
  - Extract both ListValue args
  - Allocate new list, append elements from both
  - Return new ListValue
- Mirror `listConcatImpl()` from `internal/builtins/list.go:161-183`

**Acceptance Criteria**:
- `_concat_List` no longer appears in `ailang disasm` EvalOnly reasons
- `make test` passes
- `make lint` passes
- Parity harness: no regressions

### M3: BENCHMARK_AND_CLOSE — Re-benchmark and close sprint (~30 LOC)

**Rationale**: Verify the EvalOnly reduction and measure speedup.

**Implementation**:
- Run `ailang disasm` on docparse/main.ail — expect ≤5 EvalOnly
- Run `scripts/bench_ailang_parse.sh` equivalent on ailang-parse 10MB DOCX
- Run parity harness: `go run scripts/verify_bytecode_parity.go`
- Update design doc §18 with new numbers
- Update CHANGELOG

**Acceptance Criteria**:
- docparse EvalOnly ≤ 5/1129 prototypes
- Parity: ≥ 129 MATCH, 2 NON_DET, 6 EVAL_SKIP (no regressions)
- Benchmark numbers recorded in design doc §18
- If speedup < 3×: document root cause (likely I/O dominance) and next steps
- CHANGELOG updated

---

## Estimated LOC

| Milestone | Implementation | Tests | Total |
|---|---:|---:|---:|
| M1 | 30 | 30 | 60 |
| M2 | 20 | 20 | 40 |
| M3 | 20 | 10 | 30 |
| **Total** | **70** | **60** | **130** |

## Dependencies

- M1 and M2 are independent (can be done in either order)
- M3 depends on M1 + M2

## Risks

- **Low**: OpBuiltinCall dispatch might have edge cases with list allocation in the VM's GC. Mitigated: existing `builtinListGet` already allocates list values.
- **Low**: `__list_length` vs `_len` naming collision. Check if `_len` already handles lists (it does — via `builtinLen` in vm/builtins.go). May only need `__list_length` as an alias.

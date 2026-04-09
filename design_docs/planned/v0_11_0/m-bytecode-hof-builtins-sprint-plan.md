# M-BYTECODE-HOF-BUILTINS Sprint Plan

**Design doc**: [m-bytecode-hof-builtins.md](m-bytecode-hof-builtins.md)
**Sprint ID**: `M-BYTECODE-HOF-BUILTINS`
**Estimated**: 1 day (~350 LOC)
**Dependencies**: M-BYTECODE-STDLIB-BUILTINS (complete), M-BYTECODE-VM §18.6 (complete)

## Milestones

### M1: VM Callback Infrastructure (OpBuiltinCallHOF + CallClosure)

**Scope**: Add the new opcode, ClosureCaller interface, HOFBuiltinFunc type, and VM.CallClosure method. Wire OpBuiltinCallHOF dispatch in the VM run loop. Add compiler HOFBuiltinTable and emit logic.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/opcode.go` | Add `OpBuiltinCallHOF` after `OpBuiltinCall` | 10 |
| `internal/vm/builtins.go` | Add `ClosureCaller` interface, `HOFBuiltinFunc` type, empty `HOFBuiltinTable` | 15 |
| `internal/vm/vm.go` | Add `CallClosure` method, `OpBuiltinCallHOF` dispatch case | 50 |
| `internal/bytecode/compiler/builtins.go` | Add `HOFBuiltinTable`, `hofBuiltinIndex`, update `compileBuiltinCall` | 30 |

**Acceptance criteria**:
- `OpBuiltinCallHOF` opcode exists and disassembles as `BUILTIN_CALL_HOF`
- `VM.CallClosure` compiles and handles TagClosure + EvalOnly fallback
- `compileBuiltinCall` emits `OpBuiltinCallHOF` for HOF builtins listed in `HOFBuiltinTable`
- Empty `HOFBuiltinTable` — no builtins wired yet (just infrastructure)
- `make test` passes
- `make lint` passes

**Est. LOC**: 105

---

### M2: Wire 6 HOF Builtins

**Scope**: Implement all 6 HOF builtin handlers and register them in both compiler and VM tables.

**Builtins**:
1. `__list_map` — `(a -> b, [a]) -> [b]`
2. `__list_filter` — `(a -> bool, [a]) -> [a]`
3. `__list_foldl` — `((b, a) -> b, b, [a]) -> b`
4. `__str_foldChars` — `((a, string) -> a, a, string) -> a`
5. `__str_foldSlices` — `(string, string, a, (a, string) -> a) -> a`
6. `__str_mapSlicesJoin` — `(string, string, (string) -> string) -> string`

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_hof.go` | 6 HOF builtin implementations | 180 |
| `internal/vm/builtins.go` | Populate `HOFBuiltinTable` with 6 entries | 10 |
| `internal/bytecode/compiler/builtins.go` | Add 6 names to `HOFBuiltinTable` | 10 |
| `internal/vm/builtins_hof_test.go` | Unit tests for all 6 HOF builtins | 150 |

**Acceptance criteria**:
- All 6 HOF builtins wired in both compiler and VM tables
- Unit tests pass for each builtin with closure callbacks
- Tests include edge cases: empty list, single element, nested closures
- `make test` passes
- `make lint` passes
- Parity: no regressions from 129 MATCH baseline

**Dependencies**: M1
**Est. LOC**: 350 (cumulative including M1: ~245 new)

---

### M3: Benchmark and Close

**Scope**: Re-benchmark docparse, verify EvalOnly reduction, update design doc and CHANGELOG.

**Acceptance criteria**:
- docparse EvalOnly count recorded (target: ≤ 90, down from 163)
- Parity: >= 129 MATCH, 2 NON_DET, 6 EVAL_SKIP
- Benchmark numbers on 10MB DOCX recorded in design doc
- CHANGELOG updated
- Design doc §18 updated with HOF results
- Sprint JSON updated with final metrics

**Dependencies**: M2
**Est. LOC**: 0 (documentation only)

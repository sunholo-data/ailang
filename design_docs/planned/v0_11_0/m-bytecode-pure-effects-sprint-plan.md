# M-BYTECODE-PURE-EFFECTS Sprint Plan

**Design doc**: [m-bytecode-pure-effects.md](m-bytecode-pure-effects.md)
**Sprint ID**: `M-BYTECODE-PURE-EFFECTS`
**Estimated**: 0.5 day (~460 LOC)
**Dependencies**: M-BYTECODE-LAMBDA-RESOLUTION (complete)

## Milestones

### M1: Wire JSON Builtins (encode, decode, repair)

**Scope**: Implement 3 JSON builtins natively in the VM. These use only strings and ADTs (Result, Json), both already supported by the VM's value system.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_json.go` | New: 3 JSON builtin handlers + Json ADT construction helpers | 250 |
| `internal/vm/builtins_json_test.go` | New: unit tests for encode/decode/repair | 200 |
| `internal/vm/builtins.go` | Add 3 entries to BuiltinTable | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 3 names to BuiltinTable | 5 |

**Acceptance criteria**:
- `builtinJsonEncode` converts Json ADT → string natively in VM
- `builtinJsonDecode` parses string → Result[Json, string] ADT natively
- `builtinJsonRepair` repairs truncated JSON strings
- Unit tests cover: null, bool, int, float, string, array, nested objects, error cases
- `make test` passes
- `make lint` passes

**Est. LOC**: 460

---

### M2: Benchmark and Close

**Scope**: Re-benchmark docparse, verify EvalOnly reduction, update docs.

**Acceptance criteria**:
- `ailang disasm docparse/main.ail` EvalOnly count recorded (target: ≤ 89, down from 92)
- Parity: ≥ 129 MATCH, no regressions
- 10MB DOCX benchmark recorded
- CHANGELOG updated
- Design doc updated with results

**Benchmark commands** (from [m-bytecode-vm.md §18.8](../../implemented/v0_11_0/m-bytecode-vm.md)):
```bash
cd /path/to/ailang-parse
ailang disasm docparse/main.ail 2>&1 | grep "EvalOnly:"
for backend in "" "--bytecode"; do
  echo "=== ${backend:-Evaluator} ==="
  for i in 1 2 3; do
    TMPOUT=$(mktemp -d)
    DOCPARSE_OUTPUT_DIR="$TMPOUT" /usr/bin/time ailang run \
      --entry main --caps IO,FS,Env $backend \
      docparse/main.ail data/test_files/stress/docx_10mb.docx 2>&1 | grep real
    rm -rf "$TMPOUT"
  done
done
go run ./scripts/verify_bytecode_parity.go
```

**Dependencies**: M1
**Est. LOC**: 0 (documentation only)

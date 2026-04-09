# M-BYTECODE-LAMBDA-RESOLUTION Sprint Plan

**Design doc**: [m-bytecode-lambda-resolution.md](m-bytecode-lambda-resolution.md)
**Sprint ID**: `M-BYTECODE-LAMBDA-RESOLUTION`
**Estimated**: 0.5 day (~150 LOC)
**Dependencies**: M-BYTECODE-HOF-BUILTINS (complete)

## Milestones

### M1: Fix Lambda Module Resolution + Tests

**Scope**: Propagate `currentModule` to lambda compiler, add regression tests, verify fix compiles and existing tests pass.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/compiler/lambda.go` | Add `inner.currentModule = fc.currentModule` after line 52 | 1 |
| `internal/bytecode/compiler/lambda_test.go` | Tests: multi-module lambda resolution, nested lambdas, HOF callback lambdas | 120 |

**Acceptance criteria**:
- `inner.currentModule = fc.currentModule` added in `compileLambda`
- Unit test: lambda in multi-module image references same-module function → compiles (no EvalOnly)
- Unit test: nested lambda references same-module function → compiles
- Unit test: lambda passed to HOF builtin compiles and executes via `CallClosure`
- `make test` passes
- `make lint` passes

**Est. LOC**: 121

---

### M2: Benchmark, Parity, and Documentation

**Scope**: Re-benchmark docparse EvalOnly count and 10MB DOCX timing. Run parity. Update design doc and CHANGELOG.

**Acceptance criteria**:
- `ailang disasm docparse/main.ail` EvalOnly count recorded (target: ≤ 80, down from 157)
- `BUILTIN_CALL_HOF` count recorded (expect ≥ 6)
- Parity: ≥ 129 MATCH, no regressions
- 10MB DOCX benchmark: eval vs bytecode VM best-of-3 recorded
- CHANGELOG updated in `changelogs/v0.10-current.md`
- Design doc `m-bytecode-vm.md` §18.9 updated (renumber existing §18.8→§18.9, §18.9→§18.10)
- Sprint JSON updated with final metrics

**Benchmark commands** (from [m-bytecode-vm.md §18.8](../../implemented/v0_11_0/m-bytecode-vm.md)):
```bash
# From ailang-parse repo root
ailang disasm docparse/main.ail 2>&1 | grep "EvalOnly:"
ailang disasm docparse/main.ail 2>&1 | grep -c "BUILTIN_CALL_HOF"

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

# From ailang repo root
go run ./scripts/verify_bytecode_parity.go
```

**Dependencies**: M1
**Est. LOC**: 0 (documentation only)

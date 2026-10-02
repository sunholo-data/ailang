# M-INT-DIV-ZERO-ERROR: integer division and modulo by zero raise RT001, not a Go panic

**Status**: Implemented (2026-10-02)
**Target**: v0.51.1
**Priority**: P1 (robustness: a total-looking `int -> bool` function crashes the host)
**Estimated**: 1 day
**Dependencies**: None
**Issues**: #1449 (dup #1528); message `inbox_1790965642843_c041a262`
**Triage**: [design_docs/planned/ailang-core-triage/int-div-zero-go-panic.md](../../planned/ailang-core-triage/int-div-zero-go-panic.md)

## Problem Statement

```ailang
module dz
export func boom(n: int) -> bool = 10 / n > 0
```

`boom(0)` under `ailang run` prints `panic: division by zero [recovered,
repanicked]` and a Go stack. The REPL exits on `10 / 0`. Under `serve-api`
the panic is re-raised by `internal/embed/exit.go` (`recoverProgramExit`
re-panics anything that is not `exit()`), so the request goroutine dies and
the client sees a dropped connection, not a response.

The four routes integer `/` and `%` take today disagree with each other:

| Route | Taken by | Today |
|---|---|---|
| `Num[int].div` dictionary method (`internal/types/dictionaries.go`, `registerNumInt`) | evaluator `/` on `int` | **Go panic** |
| `mod_Int` builtin (`internal/builtins/math_arithmetic.go`, mirror in `internal/eval/builtins_arithmetic.go`) | evaluator `%` on `int` | `[RT_DIV0] Modulo by zero`, no position |
| VM `arith` (`internal/vm/vm.go`) | VM `/` (`OpDiv`) | `vm: division by zero (in dz.boom at dz.ail:4 …)` |
| VM `builtinModInt` (`internal/vm/builtins_math.go`) | VM `%` (`BUILTIN_CALL _mod_Int`) | `vm: BUILTIN_CALL: _mod_Int: division by zero (…)` |

Plus three unused-in-`run` but reachable sites: the experimental binop shim
(`internal/eval/eval_operations.go`, `applyBinOp`), `TypedEvaluator.evalDiv`
/ `evalMod` (`internal/eval/eval_typed.go`), `eval_simple.go`, and the REPL's
own `Num[Int]` dictionaries (`internal/repl/repl_eval.go`,
`internal/repl/module_registry_prelude.go`, two of which divide with no
guard at all).

The registered code for this failure, `RT001` ("Division by zero",
`internal/errors/codes.go` `ErrorRegistry`), is emitted nowhere. The code that
*is* emitted, `RT_DIV0`, is not in the registry and cannot be — the
`gen-error-codes` schema accepts only `[A-Z]{2,4}[0-9]{2,4}` names
(`tools/gen-error-codes/main_test.go`, `isErrorCodeName`).

## Goals

One error, one code, one message, both engines, with a position:

- `RT001: integer division by zero at <file>:<line>:<col>` (evaluator) and
  `RT001: integer division by zero` with `<file>:<line>` (VM, which keeps
  line-only positions today); `modulo` in place of `division` for `%`.
- `ailang run` exits 1 with that line and no Go stack.
- The REPL prints the error and keeps running.
- `serve-api` answers 500 with the message in `error` and
  `error_detail.code == "RT001"`.
- Floats unchanged: IEEE 754 (`1.0 / 0.0 == Inf`, `5.5 % 0.0 == NaN`).

## Design decisions

1. **Code: `RT001`, retire `RT_DIV0`.** `RT001` is the registry's code for
   exactly this, is published in `error_codes.json`, and fits its schema.
   `RT_DIV0` had two emitters and no registry row. Nothing outside
   `internal/` mentions either (verified, V5). Changing `[RT_DIV0] Modulo by
   zero` to `RT001: integer modulo by zero at …` is the one deliberate
   user-visible message change.
2. **One typed error, `*errors.DivByZeroError`, in `internal/errors`.** Not in
   `internal/eval`, because `internal/types` (which owns the dictionary method)
   cannot import `eval` (`eval` imports `types`), and `internal/errors` depends
   only on `ast` and `schema` (V6). Every site calls
   `errors.CheckIntDivisor(op, divisor)` instead of carrying its own `if b == 0`.
3. **Position is attached once, in `evalCoreT`.** Builtins and dictionary
   methods do not know where they were called from. The first Core node on the
   unwinding path whose evaluation returned a `*DivByZeroError` with an empty
   `Pos` fills it from its `OriginalSpan()` — that is the `DictApp`/`App` that
   performed the division. Error path only; the success path is untouched.
   Same pattern as `RecursionLimitError` (a type, so wrappers can find it with
   `errors.As`).
4. **VM keeps `VMError` and gains `Cause error` + `Unwrap()`.** `Msg` becomes the
   cause's text, so `errors.As` works on VM errors and the message carries
   `RT001`.
5. **Floats keep IEEE semantics.** The spec is `internal/pipeline/op_table.go`
   `OperatorSemantics`: `div_Float` "division by zero produces ±Inf",
   `mod_Float` "mod by zero produces NaN"; `internal/builtins/math_arithmetic.go`
   documents `div_Float` the same way. Verified live (V3).
6. **`MinInt / -1` needs nothing.** Go defines `math.MinInt64 / -1 ==
   math.MinInt64` and `% -1 == 0` (two's-complement wrap, no panic; Go spec,
   "Integer operators"). Division by zero is the only Go runtime panic in
   integer `/` and `%`. A test pins the no-panic behaviour on both engines.

**Quorum: SKIP.** None of the four triggers fires: no design-freeze items, no
override of shared machinery (the change adds an error type; `wrapDictionaryMethod`
gains a case, nothing is overridden), no cost/KPI/banked-schema surface, all
premises in-repo. Language semantics change only from "crash" to "error"; the
code rename `RT_DIV0 → RT001` is a message change on an already-failing path.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `/` on int panics from the dictionary method | `ailang run` on the repro; stack top `types.(*DictionaryRegistry).registerNumInt.func4` `dictionaries.go:129` | Confirmed |
| V2 | `%` on int returns `[RT_DIV0] Modulo by zero`, exit 1, no position | `ailang run --entry modz --args-json 0` | Confirmed |
| V3 | Float `/ 0.0` = `Inf`, `% 0.0` = `NaN` on the evaluator | `ailang run` printing `fdiv(0.0)`, `fmod(0.0)` | Confirmed |
| V4 | VM `/` → `OpDiv` in `arith`; VM `%` → `BUILTIN_CALL _mod_Int` | `--bytecode --strict-bytecode` error text names `op DIV` and `op BUILTIN_CALL` | Confirmed |
| V5 | `RT001` registered but never emitted; `RT_DIV0` emitted but unregistered; neither referenced outside `internal/` | `rg RT001`, `rg RT_DIV0` over the whole repo | Confirmed: RT001 only in `internal/errors`; RT_DIV0 only in the two `mod_Int`/`div_Int` builtin files |
| V6 | `internal/errors` imports no `types`/`eval` (no cycle) | `go list -deps ./internal/errors` | `ast`, `schema` only |
| V7 | No constant folding of `/` or `%` anywhere in Go | `rg -i "constfold|constant.?fold|foldConst"` over `internal/` Go files | Empty. `10 / 0` reaches the runtime like any other division |
| V8 | REPL dies on `10 / 0` | piped `ailang repl` | Confirmed (same dictionary panic) |
| V9 | `serve-api` returns 500 + `error` string for a call error | read `internal/apiserver/routes_dispatch.go` (`callErr != nil` branch) | Confirmed; `ErrorDetail` is not set for runtime errors today |
| V10 | `internal/apiserver/mcp_gate_test.go` uses `10 / (length(token) - 4)` to provoke a verifier **panic** | read the test | Confirmed; after the fix it provokes an error instead, so that test must still pass via the gate's error path (checked in the sprint) |
| V11 | Go codegen emits raw `/` and `%` | `internal/gen/golang/codegen_ops.go` (`case core.OpDiv: return "/"`) | Confirmed; out of scope (see Non-Goals) |
| V12 | `RT001` is not taken by another meaning | `internal/errors/codes.go` row `RT001: {"runtime","arithmetic","Division by zero"}` | Same meaning — reuse, not reallocation |

## Conflict Surface

### Semantic positions touched
- The result of integer `/` and `%` when the divisor is 0, on every route in
  the table above.
- The error text of `mod_Int`/`div_Int` builtins (`RT_DIV0` → `RT001`).
- `VMError` gains a field; its `Error()` text keeps its shape.

### What else lives here
- **Float** `/` and `%`: share the `Num`/`Fractional` dictionaries and the VM
  `arith` switch. Must stay IEEE. The binop shim, `eval_typed.go` and
  `eval_simple.go` currently *error* on float `/ 0.0` — inconsistent with the
  live path, but those paths are not reached by `ailang run`; left as is
  (follow-up), only their int branches move to the shared helper.
- **Polymorphic** `/` through `Num a` (e.g. `examples/poly_arith_lambda.ail`):
  goes through the same dictionary method; `float` instantiation untouched.
- **Contracts**: a `requires { n != 0 }` contract runs before the body, so it
  still reports a contract violation, not RT001. A contract *expression*
  that divides by zero reports RT001 (it is evaluated by the same evaluator).
- **SMT** (`internal/smt`): `div`/`mod` are Z3 operators; unaffected.
- **Builtin callbacks** (`map`, `foldl`…): wrap callback errors; `errors.As`
  must still find the typed error through the wrap, so the position survives.

### Programs that MUST still work (outputs unchanged)
- `tests/binops_int.ail` → `14`
- `tests/binops_float.ail` → `1.5`
- `examples/float_nan.ail` → four `ok … =true/false` lines
- `examples/poly_arith_lambda.ail` → `[3.0, 5.0]`, `24`, `2.0`, `[6, 10]`
- `examples/inline_tests_arithmetic.ail` under `ailang test`

### What deliberately changes
- Crash → `RT001` error, everywhere integer `/` or `%` sees 0.
- `[RT_DIV0] Modulo by zero` → `RT001: integer modulo by zero at …`.
- VM messages `division by zero` / `_mod_Int: division by zero` → the RT001 text.

## Solution

`internal/errors/arith.go` (new, ~40 LOC):

```go
type DivByZeroError struct {
    Op  string // "division" or "modulo"
    Pos string // "file:line:col"; empty until the evaluator attaches it
}
func (e *DivByZeroError) Error() string // "RT001: integer division by zero[ at Pos]"
func CheckIntDivisor(op string, divisor int64) error // nil, or a fresh *DivByZeroError
```

Sites:
- `internal/types/dictionaries.go` `registerNumInt` div: `func(x, y int) (int, error)`.
- `internal/eval/eval_patterns.go` `wrapDictionaryMethod`: new `func(int, int) (int, error)` case.
- `internal/eval/eval_expressions.go` `evalCoreT`: attach position on the error path.
- `internal/eval/builtins_arithmetic.go`, `internal/builtins/math_arithmetic.go`: `div_Int`, `mod_Int`.
- `internal/eval/eval_operations.go`, `eval_typed.go`, `eval_simple.go`: int `/`, `%`.
- `internal/vm/vm.go` (`arith`, `VMError`, BUILTIN_CALL wrap), `internal/vm/builtins_math.go` (`builtinModInt`).
- `internal/repl/repl_eval.go`, `internal/repl/module_registry_prelude.go`: `Num[Int].div`.
- `internal/apiserver/routes_dispatch.go`: set `ErrorDetail{Code: "RT001"}` when the call error is a `*DivByZeroError`.
- `internal/errors/json_encoder.go`: RT001 comment becomes a fix hint.

## Testing Strategy

- `internal/errors`: `TestDivByZeroError_*` (message, `CheckIntDivisor`, registry row).
- `internal/eval`: evaluator tests for `/` and `%` by zero, nested call
  (error names the inner line), literal `10 / 0`, inside a `map` callback,
  inside a contract `ensures` expression, `MinInt / -1` no panic.
- `cmd/ailang`: CLI parity test runs each case under evaluator and
  `--bytecode --strict-bytecode`: exit 1, no `panic:` / `goroutine`, same
  `RT001: integer <op> by zero`, same file:line.
- `internal/repl`: `10 / 0` then `1 + 1` in one session.
- `internal/apiserver`: call `boom(0)` → 500, `error_detail.code == "RT001"`.
- Mutation: revert the dictionary guard / the VM guard / the position attach
  in a scratch copy, confirm it compiles and a named test fails.

## Success Criteria

- [x] Repro prints `RT001: integer division by zero at dz.ail:4:…`, exit 1, no Go stack
- [x] VM and evaluator agree on code, message and line for `/` and `%`
- [x] REPL survives `10 / 0`
- [x] serve-api returns 500 with `error_detail.code = RT001`
- [x] Float division by zero still `Inf`/`NaN`
- [x] Fixtures in "Programs that MUST still work" unchanged
- [x] Changelog fragment; errors reference page `rt001.md`

## Non-Goals / Follow-ups

- **Go codegen (`--emit-go`)**: generated Go divides natively and panics like
  any Go program; generated code already uses panics for contract violations.
  Wrapping it is a codegen-error-model decision, not this fix.
- **VM float `%`** errors (`MOD on Float not supported`) where the evaluator
  returns NaN — a separate parity gap.
- **Non-strict `--bytecode` re-runs the entry on the evaluator after a VM
  runtime error** (`internal/runner/entrypoint.go` treats every VM error as
  "bytecode unavailable"), duplicating side effects before the error.
- Shim / typed / simple evaluators' float `/ 0.0` error (not reachable from `run`).

## Axiom Compliance

| Axiom | Score | Note |
|---|---|---|
| A1 Determinism | 0 | Same inputs, same error |
| A2 Replayability | 0 | |
| A3 Effect Legibility | 0 | No effect change |
| A4 Explicit Authority | 0 | |
| A5 Bounded Verification | 0 | |
| A6 Safe Concurrency | +1 | A request goroutine no longer dies |
| A7 Machines First | +1 | Stable code + position instead of a Go stack |
| A8 Minimal Syntax | 0 | |
| A9 Cost Visibility | 0 | |
| A10 Composability | 0 | |
| A11 Structured Failure | +2 | The point of the change |
| A12 System Boundary | +1 | Hosts (serve-api, REPL) contain the failure |

Net +5; no hard violations.

## Implementation Report (2026-10-02)

Shipped as three commits on `dev` (M1 evaluator, M2 VM, M3 hosts/docs); sprint
JSON `.ailang/state/sprints/sprint_M-INT-DIV-ZERO-ERROR.json`.

**What was built** — as designed, plus one deviation:
- `internal/errors/arith.go`: `DivByZeroError{Op, Pos}`, `Code()`, `CheckIntDivisor`.
- Evaluator: `Num[int].div` returns `(int, error)`; `wrapDictionaryMethod`
  handles it; `attachDivZeroPos` in `internal/eval/div_zero_pos.go` is called
  from `evalCoreT` on the error path. `div_Int`/`mod_Int` (both copies), the
  binop shim, `TypedEvaluator` and `SimpleEvaluator` use the shared check.
- VM: `arith` and `builtinModInt` use the shared check; `VMError.Cause` +
  `Unwrap` via `errWrap` at every site that wrapped an `err`.
- REPL: one `intDivFn` for all three `Num[Int].div` methods.
- serve-api: `runtimeErrorDetail` sets `error_detail.code = RT001`.
- **Deviation**: `tools/gen-error-codes` read only `codes.go`, so the
  published `error_codes.json` lacked RT001 (and RT002–RT006, TC*, ELB*,
  LNK*, all declared in `json_encoder.go`). It now reads every non-test file
  in the package: 58 → 79 records. Without this the chosen code would not
  have been in the registry clients download.

**Evaluator positions** are the operator's column (`dz.ail:3:39` for
`10 / n` starting at column 36): the DictApp/App node's span.

**Tests**: `internal/errors/arith_test.go`, `internal/embed/divzero_test.go`
(+ `testdata/divzero.ail`), `internal/builtins/math_divzero_test.go`,
`internal/vm/vm_divzero_test.go` (replaces `TestVM_DivByZero`),
`cmd/ailang/div_zero_parity_test.go`, `internal/repl/int_div_test.go`,
`internal/apiserver/divzero_test.go`,
`tools/gen-error-codes/main_test.go` (`TestGenErrorCodes_SiblingFileCodesPresent`).

**Mutation results** (scratch-copy mutate, build, run, restore): 7/7 mutants
compiled and were killed — dictionary guard, VM `OpDiv` guard, `evalCoreT`
position attach, `VMError.Cause`, `mod_Int` builtin guard, REPL `intDivFn`
guard, serve-api detail.

**Known limitations / follow-ups**
- A lambda compiled by the VM carries no line info, so a zero divisor inside a
  `map` callback on the VM names the function but not the line (the evaluator
  gives `file:line:col`).
- The REPL prompt path divides through the `types` dictionary, so the REPL's
  own `r.instances["Num[Int]"]` is not exercised by `TestREPLIntDivZero`; the
  module-registry test covers the shared `intDivFn`.
- Go codegen, VM float `%`, non-strict `--bytecode` re-run, and the shim
  evaluators' float `/ 0.0` remain as listed under Non-Goals.

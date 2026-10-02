# Sprint Plan: M-INT-DIV-ZERO-ERROR

**Design doc**: [m-int-div-zero-error.md](m-int-div-zero-error.md)
**Status**: ✅ Completed 2026-10-02 (all acceptance criteria met; see the design doc's Implementation Report)
**Issues**: #1449, #1528 (duplicate)
**Duration**: 1 day, 3 milestones, ~450 LOC (≈ 200 implementation + 250 tests)
**Risk**: low — error path only; the success path of `/` and `%` is unchanged
**Registry reuse**: `none` for every milestone (evaluator/VM internals; no package applies)

Velocity reference: M-EVAL-TAIL-CALLS (same evaluator files, 2026-10-02) landed
~600 LOC across 3 milestones in a day.

## M1 — shared RT001 error + evaluator ✅ (~220 LOC)

- `internal/errors/arith.go`: `DivByZeroError{Op, Pos}`, `CheckIntDivisor`.
- `internal/types/dictionaries.go`: `Num[int].div` returns `(int, error)`.
- `internal/eval/eval_patterns.go`: `wrapDictionaryMethod` handles `func(int, int) (int, error)`.
- `internal/eval/eval_expressions.go`: `evalCoreT` attaches the position on the error path.
- `div_Int`/`mod_Int` in `internal/eval/builtins_arithmetic.go` and
  `internal/builtins/math_arithmetic.go`; int branches of the shim,
  `eval_typed.go`, `eval_simple.go`.
- `internal/errors/json_encoder.go`: RT001 comment → fix hint.

Acceptance:
- `go test ./internal/errors -run TestDivByZeroError` — message with and
  without position; `CheckIntDivisor` nil for non-zero; RT001 registry row
  is runtime/arithmetic.
- `go test ./internal/embed -run TestIntDivZero` (the evaluator cannot import the
  pipeline from its own package, so the end-to-end evaluator test lives in
  `internal/embed`) — `/` and `%` by zero
  return a `*DivByZeroError` (via `errors.As`) whose `Pos` names the
  dividing line, for: direct, nested call (inner line), literal `10 / 0`,
  inside a `map` callback; float `/ 0.0` = `+Inf` and `% 0.0` = `NaN`;
  `MinInt / -1` and `MinInt % -1` do not panic.
- `go test ./internal/types/... ./internal/builtins/...` green.

## M2 — VM parity + CLI ✅ (~120 LOC)

- `internal/vm/vm.go`: `VMError.Cause` + `Unwrap`; `arith` and the
  BUILTIN_CALL wrap carry the typed cause.
- `internal/vm/builtins_math.go`: `builtinModInt` uses `CheckIntDivisor`.

Acceptance:
- `go test ./internal/vm -run TestVMIntDivZero` — `errors.As` finds the
  `*DivByZeroError` for `OpDiv`, `OpMod`, `_mod_Int`.
- `go test ./cmd/ailang -run TestIntDivZeroParity` — for div, mod, nested
  call, and `10 / 0` literal: evaluator and `--bytecode --strict-bytecode`
  both exit 1, stderr contains `RT001: integer <division|modulo> by zero`
  and the same `<file>:<line>`, and never `panic:` or `goroutine `.
- `go test ./cmd/ailang -run TestIntDivZeroContract` — a function whose
  `ensures` expression divides by zero fails with RT001 (evaluator).

## M3 — hosts, docs, gates ✅ (~110 LOC)

- `internal/repl/repl_eval.go`, `internal/repl/module_registry_prelude.go`: guarded `Num[Int].div`.
- `internal/apiserver/routes_dispatch.go`: `ErrorDetail{Code: "RT001"}` on a `*DivByZeroError`.
- Docs: `docs/docs/reference/errors/rt001.md`, index row; changelog fragment.

Acceptance:
- `go test ./internal/repl -run TestREPLIntDivZero` — `10 / 0` prints RT001,
  the next `1 + 1` prints `2`.
- `go test ./internal/apiserver -run TestDivZero_StructuredError` — 500,
  `error` contains `RT001`, `error_detail.code == "RT001"`.
- `go test ./internal/apiserver -run TestMCPGate` — still green (verifier
  error fails closed).
- Fixtures in the design doc's "Programs that MUST still work" print the
  same output as before (`tests/binops_int.ail`, `tests/binops_float.ail`,
  `examples/float_nan.ail`, `examples/poly_arith_lambda.ail`,
  `ailang test examples/inline_tests_arithmetic.ail`).
- `make test-core`, `make check-file-sizes check-boundaries
  check-architecture-closure`, `gofmt -l`, `golangci-lint run` on touched dirs.
- Mutation: (a) drop the dictionary guard, (b) drop the VM `OpDiv` guard,
  (c) drop the `evalCoreT` position attach — each in a scratch copy, compiles,
  and a named test fails.

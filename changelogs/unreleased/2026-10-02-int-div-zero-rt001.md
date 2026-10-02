### Fixed — integer division by zero is an RT001 error, not a Go panic (#1449, 2026-10-02)

`10 / n` with `n == 0` crashed `ailang run` with `panic: division by zero
[recovered, repanicked]` and a Go stack, ended the REPL session, and under
`serve-api` killed the request goroutine. The panic came from the `Num[int]`
dictionary method; `%` took a different route and reported an unregistered
`[RT_DIV0] Modulo by zero` with no position, and the VM had its own wording.

Integer `/` and `%` by zero now fail everywhere with the registered code
`RT001` and the position of the dividing expression:

```
Error: execution failed: RT001: integer division by zero at dz.ail:3:39
```

- Evaluator and bytecode VM agree on code and text (the VM gives `file:line`);
  `ailang run` exits 1.
- The REPL prints the error and keeps going.
- `serve-api` answers 500 with `error_detail.code = "RT001"`.
- Hosts can match with `errors.As(err, &dz)` on `*errors.DivByZeroError`
  (`internal/errors`), through VM frames and builtin callbacks.
- Floats are unchanged (IEEE 754: `1.0 / 0.0 = +Inf`, `5.5 % 0.0 = NaN`), and
  `MinInt / -1` still wraps without error.
- `RT_DIV0` is retired in favour of `RT001`.
- `make error-codes` now reads the whole `internal/errors` package, so
  `RT001`–`RT006`, `TC*`, `ELB*` and `LNK*` reach the published
  `error_codes.json` (58 → 79 records).

Design: `design_docs/implemented/v0_51_1/m-int-div-zero-error.md`. Reference:
`docs/docs/reference/errors/rt001.md`.

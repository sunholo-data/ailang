### Added — native terminal IO

- `std/terminal.info`, `withTerminal` and `readEvent` query configured TTYs and
  deliver immediate typed key/resize/idle/EOF events on macOS/Linux under `IO`.
  Scoped sessions exclusively own input and restore terminal modes, cursor and
  alternate screen on return, errors, budget exhaustion, exit and SIGINT/SIGTERM.
  Embedders must provide a termination hook; unsupported hosts return typed errors.
- `std/io.readLineOpt` distinguishes blank and final partial lines (`Some`) from
  sticky EOF (`None`), sharing the persistent reader with unchanged `readLine`.
- Native IO and terminal effect dispatch in the bytecode VM preserves capabilities
  and budgets and supports effect-polymorphic callbacks without evaluator fallback.
  Go code generation explicitly rejects terminal operations until its host supports
  this lifecycle. After any native VM effect attempt, non-strict execution returns
  failures without replaying the entrypoint and its consumed input or prior output. Examples and real CLI PTY regressions cover both execution engines.
- The accompanying `sunholo/terminal_ui` package extends its existing renderer with
  selection, confirmation, paging and bounded versioned event replay. Native
  publication requires a supporting released core and registry validator.


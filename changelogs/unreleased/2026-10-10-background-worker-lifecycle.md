### Fixed

- CLI runs, batch items, embedded requests/engines, and REPL sessions now stop and join their owned managed/async workers on shutdown. Closing managed stdin retains ownership until reaping. SIGINT/SIGTERM restore terminal state before bounded worker cleanup.
- Async process sources reap naturally with one waiter and preserve final output chunks. Process-group shutdown handles non-detached POSIX descendants; owned IO tasks join even under backpressure.

### Added

- `std/process.cancelProcess` and `std/stream.cancelProcessSource`, with typed invalid-handle, unsupported-platform, timeout and infrastructure failures. Native evaluator and strict VM dispatch use the same capability/budget checks and owner-local handle lookup.
- One two-second shutdown deadline for an owner's workers, independent of user budgets; up to 250 ms for already-closing stdin to drain. Borrowed stdin/connection resources retain their ownership rules.

Explicit cancellation supports macOS/Linux process groups. Windows and JS/WASM return typed unsupported results; Windows host teardown remains direct-child only. Local worker termination does not guarantee cancellation of remote AI inference or billing. Supporting runtime publication and the consumer's pinned-release cleanup retest follow delivery; this sprint makes no release claim.

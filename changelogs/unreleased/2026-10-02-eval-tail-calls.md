### Added — tail-call elimination in the interpreter (#1486)

- A call that is the last thing a function does now reuses the caller's frame, so tail-recursive loops run in
  constant evaluator depth and memory, the same as on the bytecode VM. Before, every interpreter call was
  Go recursion counted by the 10,000-frame limit. A 20,000-line stdin loop failed `RT_REC_003` in the
  interpreter and passed on `--bytecode`.
- This covers any callee, mutual recursion included, through `if` branches, `let`/`letrec` bodies, `match` arms
  and blocks. All four shapes now match `--strict-bytecode` at 200,000 iterations with the default limit.
  Peak RSS at 10^6 iterations is about 105 MB, against 81 MB at 10^5.
- **Unchanged:** a function with an active `ensures`, an `@limit`/`@min` budget or a declared rand mode keeps a
  nested frame, because each runs work after its body. `RT_REC_003` still bounds non-tail recursion. Deep
  traces (`--trace-tier deep`) emit the same enter/exit sequence, verified against goldens captured before
  the change.
- **Behaviour change:** an infinite tail loop (`loop(n) = loop(n + 1)`) now runs until killed instead of failing
  `RT_REC_003`, as it already did under `--bytecode`.
- The `RT_REC_003` message now recommends rewriting the recursion as a tail call with an accumulator argument.
  Before this, that advice was banned from the message because the interpreter had no tail calls.
- Design: [M-EVAL-TAIL-CALLS](../design_docs/planned/v0_51_1/m-eval-tail-calls.md) (ratified after two
  independent quorum rounds).

### Security — `--bytecode` dropped `Rand[mode=crypto|seeded]` and per-function budgets (#1545) (2026-10-02)

Under `ailang run --bytecode`, a function declaring `Rand[mode=crypto]` or `Rand[mode=seeded]` ran on the
bytecode VM, which never pushes the Rand mode. Every draw is bridged to the evaluator, which then saw the
default `os` mode: crypto draws (API keys, OAuth codes and tokens) silently came from `math/rand`, and
seeded draws were neither deterministic nor refused without `AILANG_SEED`. The same gap let a function's
`@limit` / `@min` budget go unenforced (`examples/tests/test_capability_budget_exhausted.ail` printed past its
limit). The lower pass now keeps every function whose type declares a dynamic frame mode —
`Rand[mode=seeded|crypto]`, `@limit` / `@min`, and `Net[scope=public]` (#1522) — off the VM, including a
top-level function that holds such a lambda: it runs on the evaluator, which pushes the frame for it and
all its callees. Under `--strict-bytecode` the call fails naming the mode
(`Rand[mode=seeded] frame needs the evaluator`). `ailang test --bytecode` shares the same lowering.

### Fixed — `--bytecode` re-ran a failed evaluator-only entry (2026-10-02)

When the entry function itself is evaluator-only (now every `Rand`-moded entry), `--bytecode` ran it on the
evaluator and, if it failed, fell back and ran it again, repeating its side effects. The first outcome is now
final.

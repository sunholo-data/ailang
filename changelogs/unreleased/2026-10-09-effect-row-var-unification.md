### Fixed — Shared effect rows at higher-order calls

Refs #616. App inference shares an explicitly open callee row with its callback
contract. The existing per-App publication carries both the latent argument mask
and the resolved call row. Pure callbacks discharge to pure; independently
instantiated IO callbacks require IO. Unresolved tails produce a named diagnostic,
and distinct tails produce an explicit conflict rather than disappearing.

A helper such as `runTwice(f: () -> int ! {e}) -> int = f() + f()` is now rejected
before printing. Migration: propagate the shared row with
`runTwice(f: () -> int ! {e}) -> int ! {e} = f() + f()` and declare concrete IO at
an IO caller. Runtime capability grants remain required.

The cold-cache comparison checked all 496 existing example/std files, including
49 std files in both direct and relaxed modes: 545 checks, 462 passes and 83
unchanged semantic failures. **No existing path flipped pass/fail status**, so
no corpus file needs migration. The two new runnable demos bring the fixed totals
to 498 files, 547 checks and 464 passes. Every std/stream and std/ai/streaming
consumer retained its status; same-tail generic streaming and missing-IO importer
controls pass their expected contracts.

Callbacks nested in list, tuple or ADT arguments remain outside this repair
(#1718). Concrete callback annotation upper bounds remain deferred. Full per-file
commands, diagnostics, hashes and regression evidence are recorded in
`.ailang/state/sprints/validation_M-EFFECT-ROW-VAR-UNIFICATION.json` and the
companion validation report.

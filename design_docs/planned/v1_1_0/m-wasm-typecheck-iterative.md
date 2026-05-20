# M-WASM-TYPECHECK-ITERATIVE — Make the type-checker iterative so WASM-compiled AILANG can load realistic modules

**Status**: Planned — P1 (blocks any non-trivial browser demo that uses tagged-union pattern matching + record-field access in the same function)
**Target**: v1.1.0 (could backport to v0.21.x if a P0 demo regresses; see *Workarounds* below)
**Priority**: P1 — silent main-thread freeze in browser demos. CLI is unaffected so it's invisible to CI without dedicated WASM smoke testing.
**Estimated**: 5–7 days (~1500 LOC) — see *Implementation* below for milestone breakdown.
**Dependencies**: M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (May 2026, in v0.20.x) — that's the change that pushed the recursion past the WASM stack limit for realistic module shapes. Fix is to refactor the analyses introduced/amplified there, not to revert.
**Source**: Discovered 2026-05-20 while porting the Cognitive Commons demo to AILANG. Full diagnostic trail: [debug-notes/wasm-citizen-stack-overflow.md](../../../../demos/debug-notes/wasm-citizen-stack-overflow.md) in the sunholo-data/ailang-demos repo.

## Problem

The AILANG type-checker (`internal/types/typechecker_core.go`, `tagged_union_predicate.go`, constraint solver, ADT-head resolution) uses **recursive descent** for almost every analysis pass. On native Go, this is fine — goroutine stacks grow dynamically up to 1 GiB. On WebAssembly compiled via `GOOS=js GOARCH=wasm`, the Go runtime uses the **host JavaScript engine's call stack** (~10–15K frames in Node, similar in browsers). For realistically-sized AILANG modules with the patterns introduced by M-TYPECHECK-NO-AUTO-UNWRAP-RESULT — repeated matches on the same tagged-union receiver, record-field access on unwrapped sums, deep import graphs — the recursion overflows the JS stack 80–120 seconds into the type-check, AFTER the page main thread has been locked.

The browser surface is brutal:
- No console error
- No visible banner without explicit instrumentation
- "Page slowing down" dialog after ~10 seconds
- "Maximum call stack size exceeded" thrown ~80–120 s later but the page is already unresponsive
- DevTools cannot be opened reliably while the thread is locked

**Reproducer**: `cognitive_commons/services/citizen.ail` (7170 bytes, 11 imports, 2 effect-annotated functions, triple-nested `match` inside `compose`, three back-to-back matches on `Result[JudgeScore, string]` with `s.x`/`s.y` field access inside `speak`). `ailang check citizen.ail` passes in <1 s. `node demos/scripts/wasm-loadmodule-harness.js` throws after 82–101 s.

**CLI vs WASM gap on the same source code**:

| | CLI (native Go) | WASM (Node host) |
|---|---|---|
| `loadModule citizen.ail` | 18 ms (success) | 82,364 ms (throw: stack exceeded) |
| Stack model | per-goroutine growable | bounded by JS host |
| `--stack-size=32000` workaround | n/a | timed out at 60 s — recursion is also pathologically slow, not just deep |

## Why this can't be ignored

1. Browser demos are the primary surface that non-CLI users see. A demo class that can't even load is worse than missing.
2. The bug class is **silent** — no error, no telemetry signal. Without a dedicated WASM smoke harness (which we now have, see [scripts/wasm-loadmodule-harness.js](../../../../demos/scripts/wasm-loadmodule-harness.js)) it slips through PR review entirely.
3. As the type-checker gets more sophisticated (cycle-safe ADT analysis, row-polymorphism interactions, foreign-constructor cross-checking), the recursion depth and per-step cost both grow. Each future v0.x.y release risks pushing more modules over the cliff.
4. The CLI sees none of this. PRs touching `internal/types/` pass all CLI tests and ship. The first signal is a frozen browser tab, usually hours or days post-release.

## Root cause

Three classes of recursion compound on the same call path:

1. **AST traversal** — `internal/types/typechecker_core.go` recurses through `match` arm bodies, `let` bodies, function bodies, lambda bodies. Each level adds frames.
2. **Constraint solving** — when a `match` arm introduces a fresh type variable bound to a tagged-union receiver, unification recurses through field accessor constraints (`s.x` triggers a row-extension constraint on `s`'s row variable, which the unifier walks).
3. **Tagged-union predicate** — `isTaggedUnion(t, ctors)` from `internal/types/tagged_union_predicate.go` (introduced in M-TYPECHECK-NO-AUTO-UNWRAP-RESULT M1) is cycle-safe but performs a full traversal of the type's structure each call. Repeated invocations on the same receiver re-do the work.

For `citizen.ail::speak`, those compound. Three back-to-back matches on `score_result` each trigger fresh predicate analysis on `s`; the predicate descends through `Result[JudgeScore, string]`'s structure once per match arm; each `s.x` field access spawns its own row-constraint walk. Estimated frame count for that function alone: 8K–12K. Easily over the JS limit, even before `compose`'s triple-nested match adds another 5K.

Specific hot paths (verified by `strings demos/wasm/ailang.wasm` showing the function names embedded in the WASM):

- `internal/types/typechecker_core.go` — `(*CoreTypeChecker).checkExpr`, `.checkMatch`, `.checkApp`
- `internal/types/tagged_union_predicate.go` — `isTaggedUnion`, `buildTufaMessage`, the three error-site builders
- `internal/types/unification.go` — `unify`, `bindVar`, row-extension unifier
- `internal/types/scheme.go` — `instantiate` on imported schemes (amplified by M-SCHEME-IMPORT-PRESERVE-ADT-HEAD which preserves more head info per import)

## Proposed fix

Convert the recursion to **iteration + explicit work-stack** in the WASM-critical hot paths. Three milestones in dependency order:

### M1 — Iterative AST traversal (~2 days, ~400 LOC)

Rewrite `(*CoreTypeChecker).checkExpr` and its sibling dispatchers to use an explicit work-stack of (node, env, expected-type) triples instead of recursive descent. Each step pops one triple, processes it, pushes child triples in the right order. The depth-multiplier is now O(1) per AST node instead of O(depth).

**Trickiness**: type-checker is conditional — checking a `match` arm depends on knowing the scrutinee's type FIRST. The work-stack needs a "deferred" entry kind so we can come back to an arm after its constraints have been solved.

**Verification**: existing test suite in `internal/types/` passes unchanged. Add a regression test: type-check a synthetic module with 1000-level-deep `match` nesting; succeeds in CLI AND in WASM via the harness.

### M2 — Memoize `isTaggedUnion` per (type, ctorSet) (~1 day, ~150 LOC)

`tagged_union_predicate.go::isTaggedUnion` is called repeatedly on the same receiver type during a single check pass. Memoize on `(type.ID(), ctorSetHash)`; clear the cache at type-check session boundaries. Reduces redundant traversal from O(matches × type-size) to O(matches + type-size).

**Trickiness**: the predicate is cycle-safe via a visited-set. Memoization keys must include the visited-set or be cleared between top-level calls.

**Verification**: instrument with a counter, verify call-count drop on the citizen.ail repro. Add a benchmark in `internal/types/benchmarks/`.

### M3 — Iterative row-extension unifier (~2 days, ~500 LOC)

`unification.go::unify` recurses on row-extension chains. For deeply-nested record types or extended rows, this can be deep. Convert to explicit walk using a constraint queue.

**Trickiness**: row-polymorphism's substitution propagation must remain correct. Order-of-substitution matters for some constraint shapes.

**Verification**: existing row-polymorphism tests (`internal/types/row_*_test.go`) plus the cognitive_commons repro.

### M4 — Restore the full citizen.ail body in the demo + smoke gate (~0.5 day)

Once M1–M3 land, restore `cognitive_commons/services/citizen.ail` from the stub (the `.orig` copy in the demos repo's git history) and re-run the harness:

```bash
node scripts/wasm-loadmodule-harness.js
```

Exit 0 required for merge. Wire this harness into the demos repo's CI for any PR that touches `**/*.ail` under `cognitive_commons/`, `invoice_processor_wasm/`, or other WASM-targeted demos.

## Alternatives considered

### A. Revert M-TYPECHECK-NO-AUTO-UNWRAP-RESULT for WASM target

Compile-time gate: when `GOOS=js`, disable the tagged-union receiver gating. The bug class it catches (motoko `Result.message` auto-unwrap crash) is real but rare; WASM modules could ship without it.

Rejected: bifurcates the language semantics across compile targets. A browser module could pass WASM type-check and fail CLI; that's worse than the current asymmetry.

### B. Compile-time check-depth budget

When `GOOS=js`, panic with a clear error at frame depth > 8K. At least gives a visible error instead of a frozen page.

Rejected: makes a previously-working module suddenly fail at a fuzzy threshold. Doesn't fix the underlying issue. Would still leave us in the position of "rewrite your AILANG to be flatter" with no clear guidance on where the cliff is.

### C. Move type-check to a Web Worker

Off-load the WASM type-check to a Worker so the main thread stays responsive while the recursion runs. The Worker has its own (also bounded) stack but at least the page doesn't lock.

Rejected: doesn't solve the actual hang, just makes it invisible. The Worker still throws after 80 s. And it forces the demo shell to handle async module loading instead of the current synchronous `repl.loadModule` interface, breaking every WASM demo's boot sequence.

### D. Ship the harness-on-every-PR and live with the workaround indefinitely

Stub offending modules; rely on PR-time WASM smoke tests to catch regressions; rewrite `.ail` to flatter shapes when needed.

Considered as a fallback if M1–M3 prove too risky for v1.0.0. The harness IS being added as Step 4 / CI in any case (see *Prevention infrastructure* below). But indefinite workaround means we permanently can't write idiomatic AILANG in browser demos — that's a long-term tax on every demo author.

## Workarounds (in effect today)

Until this lands:

1. **Stub the offending module**. Keep exported signatures so downstream type-checks pass; bodies return placeholders. Cognitive Commons's `citizen.ail` is currently a stub.
2. **Flatten matches in WASM-targeted `.ail`**. Triple-nested match → sequential let-bindings + helper functions. Repeated matches on the same value → collapse into one match returning a tuple. (Try this first; it sometimes works for less pathological cases.)
3. **Run the harness before merging any `.ail` touched in a browser demo**:
   ```bash
   node demos/scripts/wasm-loadmodule-harness.js
   ```
   Exit 0 required. Anything else flags the regression.

## Prevention infrastructure (already in flight)

These ship regardless of when M1–M3 land — they're the gate against silent recurrence:

- **`demos/scripts/wasm-loadmodule-harness.js`** (already in demos repo) — headless WASM smoke that catches the bug class.
- **`demos/scripts/check-wasm-freshness.sh`** + **`demos/scripts/serve.sh`** integration — catches the related "wasm older than source" bug class.
- **`demos/cognitive_commons/{index.html, diag.html, reset.html}`** — boot banner + `localStorage` persistence + companion pages so the user never has to open DevTools to see a WASM failure.
- **`.claude/skills/wasm-debugger/`** (this repo) — the diagnostic ladder, runbook style.
- **`debug-notes/wasm-citizen-stack-overflow.md`** (in demos repo) — postmortem of the first instance.

## Acceptance criteria for the milestone

1. `cognitive_commons/services/citizen.ail` (the original, not the stub) loads in WASM in under 1 second via `node demos/scripts/wasm-loadmodule-harness.js`.
2. New regression test in `internal/types/`: synthetic 1000-level-deep `match` nesting passes CLI type-check AND WASM smoke (using `make build-wasm` + the harness).
3. Existing test suite passes unchanged (`make test`, `make verify-examples`, `make ci`).
4. No measurable CLI type-check regression on the standard benchmark suite (`make test-coverage-badge`, plus a dedicated `internal/types/benchmarks/typecheck_bench_test.go` showing iteration ≤ recursion native).
5. Postmortem doc updated to mark the upstream fix landed; demos repo's stub citizen.ail restored from the `.orig` copy.

## Open questions

1. **Should `instantiate` on imported schemes also be iterative?** M-SCHEME-IMPORT-PRESERVE-ADT-HEAD added work there. Probably yes but not on the critical path for v1.1.0 — add as a follow-up.
2. **Worker-side type-check for non-blocking demo boot, longer-term?** Even after this fix, large modules will type-check on the main thread. A future M-WASM-WORKER-TYPECHECK could move this off-thread without changing the underlying algorithm.
3. **Coupled with M-AOT-TYPECHECK?** If we add ahead-of-time `.ail` → typed-IR caching, repeat browser loads wouldn't re-run the type-checker at all. Bigger scope, probably v1.2+.

## Files expected to change

| File | M1 | M2 | M3 | M4 |
|---|---|---|---|---|
| `internal/types/typechecker_core.go` | rewrite | — | — | — |
| `internal/types/tagged_union_predicate.go` | — | memoize | — | — |
| `internal/types/unification.go` | — | — | iterative | — |
| `internal/types/scheme.go` | adjust callers | — | — | — |
| `internal/types/iter_work_stack.go` *(new)* | + | — | + | — |
| `internal/types/benchmarks/typecheck_bench_test.go` *(new)* | + | + | + | — |
| `cmd/wasm/main.go` | — | — | — | rebuild |
| `demos/cognitive_commons/services/citizen.ail` (in demos repo) | — | — | — | restore from `.orig` |

## Sprint hand-off note

When sprint-planner picks this up:

- The harness (`demos/scripts/wasm-loadmodule-harness.js`) is the **acceptance gate** — wire it into the sprint's regression checks.
- The reproducer (`debug-notes/wasm-citizen-stack-overflow.md` in demos repo) has the exact failure shape and timing data.
- Use the wasm-debugger skill for any during-sprint debugging that surfaces.
- Each milestone (M1, M2, M3) should be independently shippable; the demos repo's CI should re-run the harness on each.

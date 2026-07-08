# M-WASM-TYPECHECK-LIMITS — depth-budget guard + clear error for WASM type-checker overflow

**Status**: Planned — P1 (replaces [M-WASM-TYPECHECK-ITERATIVE](../../deferred/m-wasm-typecheck-iterative.md), the high-risk refactor path)
**Target**: v0.22.x
**Priority**: P1 — addresses the user-visible pain (silent browser freeze) without touching the type-checker hot path
**Estimated**: 1–2 days (~200 LOC)
**Risk**: LOW — additive guard in WASM-only build, no behavioural change on CLI
**Source**: 2026-05-20 cognitive_commons freeze. Diagnostic trail: [demos/debug-notes/wasm-citizen-stack-overflow.md](../../../../demos/debug-notes/wasm-citizen-stack-overflow.md).

## Problem (recap)

WASM-compiled AILANG can hang the browser main thread for 80–120 seconds on type-checker recursion that's fine on native Go (native goroutines grow stacks dynamically; WASM is bound by the JS host's ~10–15K-frame call stack). The cliff was first hit by `cognitive_commons/services/citizen.ail` — triple-nested `match` + repeated tagged-union matches with record-field access.

CLI works fine. The hang is silent in the browser: no console error, no banner, DevTools won't open while the main thread is locked. User finally sees "page slowing down" + the "Maximum call stack size exceeded" throw 80 s later but by then the demo is unusable.

## Why this instead of the iterative refactor

[M-WASM-TYPECHECK-ITERATIVE](../../deferred/m-wasm-typecheck-iterative.md) was the original plan: convert recursive descent in `typechecker_core.go` + `tagged_union_predicate.go` + the row unifier to iterative work-stack passes. 7 days, ~1500 LOC, HIGH risk to code that every AILANG program runs through. Replacing it because:

- **Risk asymmetry**: one demo hit the cliff; a type-checker regression breaks every program.
- **Workarounds work**: flattening nested matches + extracting helper functions is the natural AILANG idiom anyway. The "bad pattern" the cliff catches is genuinely hard to read.
- **Time to value**: 2 days vs 7. The user pain (silent hang) is the actual problem; the underlying recursion is a perf concern with a documented workaround.

If the cliff becomes frequent, the iterative refactor design doc remains intact in `deferred/` ready to revive.

## Proposed fix

Three pieces, all under a WASM-only build tag (`//go:build js && wasm`) so the CLI is untouched.

### 1. Depth-budget guard in `inferCore`

A new WASM-only file `internal/types/typechecker_wasm_depth.go` (`//go:build js && wasm`) wraps `inferCore` with a counter. When depth exceeds a budget (initial: 5000 frames — generous, only catches pathological cases), abort with a structured error instead of letting the JS engine throw `Maximum call stack size exceeded` opaquely 60+ seconds later.

Pseudocode:

```go
//go:build js && wasm

package types

const wasmInferDepthBudget = 5000

func (tc *CoreTypeChecker) checkWasmDepth(ctx *InferenceContext) error {
    if ctx.wasmDepth > wasmInferDepthBudget {
        return WasmTypeCheckerDepthExceeded{
            Budget: wasmInferDepthBudget,
            Module: ctx.currentModule,
        }
    }
    ctx.wasmDepth++
    return nil
}
```

`inferCore` calls `checkWasmDepth` at entry, decrements on exit. On native Go the entire file compiles out via the build tag — zero overhead.

### 2. Structured error type with actionable message

```go
type WasmTypeCheckerDepthExceeded struct {
    Budget int
    Module string
}

func (e WasmTypeCheckerDepthExceeded) Error() string {
    return fmt.Sprintf(
        "module %q exceeds WASM type-checker depth budget (%d frames).\n\n"+
        "This module's type structure recurses too deeply for the WASM host stack.\n"+
        "The same code likely works on the AILANG CLI — the limit is specific to browser execution.\n\n"+
        "Common triggers:\n"+
        "  - Triple-nested match patterns\n"+
        "  - Multiple back-to-back matches on the same tagged-union value (Ok(s) => s.x, then Ok(s) => s.y)\n"+
        "  - Long chains of intra-package imports with destructured constructors\n\n"+
        "Workarounds:\n"+
        "  1. Flatten nested matches into sequential let-bindings\n"+
        "  2. Extract a helper function that does one match and returns a record/tuple\n"+
        "  3. Split the function into smaller top-level functions\n\n"+
        "See: https://ailang.sunholo.com/docs/reference/limitations#wasm-type-checker-depth\n"+
        "Headless smoke harness: demos/scripts/wasm-loadmodule-harness.js",
        e.Module, e.Budget,
    )
}
```

This surfaces in the browser's `repl.loadModule(...)` return value as `{success: false, error: "<message>"}` instead of a hang. The existing boot banner in cognitive_commons (and any future demo using `bootLog`) renders it directly.

### 3. Documentation in `docs/docs/reference/limitations.md`

Add a section under "Type System Limitations" alongside the existing Y-Combinator entry:

```markdown
### WASM Type-Checker Depth Limit (By Design)

**Status**: WASM-specific constraint
**Since**: v0.22.x
**Affects**: AILANG modules compiled to WebAssembly (browser demos)

The WASM-compiled type-checker is bound by the host JavaScript engine's
call-stack limit (~10K frames). Modules with deeply-recursive type
structure can exceed it and fail with a clear error. The same module
works without limit on the AILANG CLI.

[... full text with example, workarounds, history ...]
```

### 4. CI gate (in sister demos repo)

The headless harness `demos/scripts/wasm-loadmodule-harness.js` already exists. Wire it into the demos repo's GitHub Actions to gate every PR that touches `**/*.ail` under `cognitive_commons/` (and future browser demos). Now the gate has a CLEAR error to report instead of a hang.

## Acceptance criteria

1. Running the headless harness against `citizen.ail.orig` (the original 7KB version) now exits with code `3` (loadModule returned !success) and a clear error message naming the module, budget, workarounds, and docs link — instead of timing out at 80+ seconds with "Maximum call stack size exceeded".
2. CLI `ailang check cognitive_commons/services/citizen.ail` still passes unchanged in <1s — zero behaviour change on native Go.
3. `make test` passes — the new file is WASM-only, doesn't touch native test paths.
4. Native CLI binary size + WASM binary size both unchanged (±1%).
5. `docs/docs/reference/limitations.md` has a new "WASM Type-Checker Depth Limit" section under "Type System Limitations" with: status, since-version, problem statement, example, workarounds, docs links to the harness + the postmortem.
6. Sister demos repo has a GitHub Action running the harness on PR changes to `**/*.ail` under `cognitive_commons/`.

## Out of scope (deferred to iterative refactor)

- Making the type-checker not need a depth budget in the first place. Tracked in [M-WASM-TYPECHECK-ITERATIVE (deferred)](../../deferred/m-wasm-typecheck-iterative.md).
- Memoizing `isTaggedUnion`. Tracked in the same deferred doc.
- Iterative row unifier. Same.

## Why P1 even though it's "quick errors"

The previous failure mode (silent browser freeze, no console error, DevTools locked) ate half a day of debugging for a single demo. Without this guard:
- Future demo authors hit the same cliff
- The diagnostic trail in `debug-notes/wasm-citizen-stack-overflow.md` is the only signal anyone has
- Each new module that hits the cliff costs another half-day

With this guard:
- The browser returns the error immediately (no 80s hang)
- The boot banner in the demo (already in place) renders it
- The docs link guides the user to the workaround
- The CI gate catches it before merge

The cost asymmetry (1-2 days of low-risk work vs. recurring half-day debugging sessions) makes this P1.

## Sprint plan

See [m-wasm-typecheck-limits-sprint-plan.md](m-wasm-typecheck-limits-sprint-plan.md) for milestone breakdown.

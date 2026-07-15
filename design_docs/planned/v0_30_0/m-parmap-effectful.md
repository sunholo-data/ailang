# M-PARMAP-EFFECTFUL: Concurrent Map over Effectful Functions (`parMap` / `gather`)

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (High — the one primitive an in-AILANG eval/agent harness actually needs)
**Estimated**: 5-7 days (M0 Fork-safety gate ~1.5-2d + M1 primitive ~3-4d + docs)
**Dependencies**: M-SERVE-API-CONCURRENCY (Fork() + thread-safe Environment — complete, shipped v0.9.4). **NB: the audit (Conflict Surface) found `EffContext.Clone()` shares effect state shallowly — the concurrent path panics on the Budget map today. M0 must fix this before any `parMap` surface.**
**Milestone ID**: M-PARMAP-EFFECTFUL
**Created**: 2026-07-15
**Source**: Benchmark-discovered gap. An eval driver fanning out N independent `std/ai` request/response calls has no in-program concurrency primitive: `--batch` is a sequential loop, and `Stream`/`selectEvents` is event-source-shaped, not request/response fan-out. This is the "Intra-request parallelism" item that [M-CONCURRENCY-LEVERAGE](../v0_29_0/m-concurrency-leverage.md) explicitly deferred to "a separate design doc" (Non-Goals) and listed only under Future Work.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | `parMap` **must** return results in input order regardless of completion order — the value is order-independent, the *result* is not |
| A2: Replayability | 0 | Same inputs → same ordered outputs at any concurrency level |
| A3: Effect Legibility | +1 | The effect row of the mapped function propagates to `parMap` — concurrency does not launder effects |
| A4: Explicit Authority | 0 | No new authority; inherits the callee's effects |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | +2 | Direct use of the Fork()/thread-safe-Environment substrate we already built and race-tested; each element runs on its own Fork |
| A7: Machines First | +2 | The harness that generates/evaluates AILANG can itself fan out N model calls in-program, keeping budget + telemetry in one process |
| A8: Minimal Syntax | 0 | No new syntax — three stdlib functions with ordinary signatures; no grammar/keyword change |
| A9: Cost Visibility | +1 | Bounded worker pool (`parMapN`) makes the concurrency ceiling explicit; per-element AI cost stays attributed in-program |
| A10: Composability | +1 | `parMap` composes with any effectful `(a) -> b ! {e}`, uniformly across `AI`, `Net`, `FS`, `IO` — it is the concurrent sibling of the existing `mapE` |
| A11: Structured Failure | +1 | `parMapResult` returns `[Result[b, Error]]` — failures are values, not panics; fail-fast `parMap` propagates a typed error in input order |
| A12: System Boundary | 0 | No boundary changes — concurrency is internal to one evaluator process |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — results are input-ordered by construction (indexed collection); side-effect *ordering* is explicitly out of contract, not silently nondeterministic
- [x] A3 (Effects): No hidden effects — the callee's effect row propagates unchanged; `parMap` adds none of its own
- [x] A4 (Authority): No ambient authority — inherits exactly the callee's effects, grants nothing
- [x] A7 (Machines First): Optimizes for machine analysis (in-program budget/telemetry), not human convenience

---

## Problem Statement

The concurrency substrate exists but is not surfaced to AILANG user code. Forensic audit of the codebase (2026-07-15):

| Capability | State | Evidence |
|------------|-------|----------|
| Batch mode overlaps AI calls | **No** — strictly sequential | `cmd/ailang/main_run_exec.go:313` — plain `for i, input := range programArgs` calling `executeBatchItem` synchronously; no goroutines/waitgroup/semaphore |
| `Stream`/`selectEvents` fan-out | **No** — event-source-shaped | `internal/effects/stream_mux.go:26` — consumer-side priority merge of persistent sources (stdin, WS/SSE, subprocess stdout) via `reflect.Select`; cannot issue N outbound calls + gather N responses |
| `asyncExecProcess` | Exists — subprocess streaming source | `internal/effects/stream_process.go:44` — one process, stdout → events; not a collection fan-out |
| Go-internal `Fork()` | Exists — **not** AILANG-callable | `internal/eval/eval_evaluator.go:142`; sole caller `internal/runtime/entrypoint.go:96` (per-HTTP-request isolation in serve-api) |
| AILANG-callable `parMap`/`gather`/`Fork` | **Missing** | Zero builtin/stdlib matches across `internal/builtins/`, `internal/runtime/builtins.go`, `std/**` |
| `std/ai` concurrency | **None** | `std/ai.ail` — `call`/`callJson`/`step`/`stepWithStream`/`runTools` are all single-shot `! {AI}` |

### The concrete shape that has no home today

An eval/agent harness written in AILANG wants: *given `xs: [Input]` and `f: Input -> Output ! {AI}`, run all `f(x)` concurrently with a bounded worker pool, collect `[Output]` in input order, keep budget/telemetry in-program.*

- `map(f, xs)` runs them serially — for N items at ~1-3s of network latency each, wall-clock is `N × latency`.
- The only current workaround is process-level: spawn N `ailang run` subprocesses via `asyncExecProcess` and merge with `selectEvents`. That works but loses in-program budget/telemetry, is more code, and is throwaway once this primitive lands.

**The hard part is already built.** `CoreEvaluator.Fork()` + thread-safe `Environment` (M-SERVE-API-CONCURRENCY, race-tested, shipped) give each concurrent evaluation an isolated evaluator. This milestone *surfaces existing machinery* as a language primitive — it is not building concurrency from scratch.

---

## Goals

**Primary Goal:** Provide an AILANG-callable `parMap` (and bounded `parMapN`) that evaluates an effectful function concurrently across a list, returning results in input order, backed by the existing Fork()/Environment substrate.

**Success Metrics:**
- `parMapN(f, xs, 8)` runs up to 8 elements of `xs` concurrently, each on its own Fork
- Results are returned in **input order**, independent of completion order (determinism)
- The mapped function's effect row propagates: `parMap[a,b,e](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}` (same shape as the existing serial `mapE`)
- Wall-clock for N I/O-bound items ≈ `ceil(N/workers) × latency`, not `N × latency`
- `go test -race` passes for the new evaluator path
- No new authority — `parMap` over a pure function is pure; over `! {AI}` carries `{AI}`

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Surface as builtin vs `Fork` effect op | Determines the type story and whether it's a language primitive or an effect capability | human | design | high |
| Error semantics: first-error vs all-results | Changes the return type (`[b]` vs `[Result[b, e]]`) | human | design | high |
| Bounded pool default (`parMap` = ?) | Unbounded fan-out over `! {AI}` can hit rate limits / OOM | human | design | med |
| Determinism guarantee (input-order return) | A1 compliance — must be a guarantee, not a coincidence | human | design | low |
| **Per-fork vs shared budget** | Deep-cloning `Budget` per fork gives each concurrent call its own cap → a "$5 run" becomes N×$5 and the bound is meaningless. An eval harness needs ONE shared, atomically-synchronized budget. This decides the `Clone()` fix shape. | human | design | high |

### Design Freeze

- [ ] **Surface**: stdlib functions `parMap`/`parMapN` typed `(f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}` (leading candidate — see Open Questions for the `Fork` effect-op alternative)
- [ ] **Naming**: `parMap`/`parMapN` vs convention-strict `parMapE` (matching `mapE`/`filterE`/`foldE`). Recommendation: `parMap` (no `E` — inherently effectful, cleaner call sites)
- [ ] **Error semantics**: `parMap` is fail-fast → `[b] ! {e}` (propagates the first error, cancels the rest); a `parMapResult` variant returns `[Result[b, Error]]` for collect-all. **Recommendation: ship `parMapResult` first** (total, no cancellation semantics to design), add fail-fast `parMap` second.
- [ ] **Bound**: `parMap` = `runtime.NumCPU()` for CPU-bound; `parMapN(n, ...)` for explicit bound. **AI-bound callers should always use `parMapN`** — document `parMap` as CPU-oriented.
- [x] **Determinism**: results **always** returned in input order (indexed collection, not completion order).
- [ ] **Budget model** (blocks M1): **shared, synchronized** budget — one cap for the whole `parMap`, not per-fork. `Clone()` must keep `Budget`/`BudgetReport` shared but make their mutations mutex-guarded (NOT deep-copy them, which would multiply the cap by N). `AI.lastRoute` and `IOWriter`, by contrast, *should* be per-fork isolated. This split is the crux of the `Clone()` fix.

---

## Solution Design

### The type

AILANG **already has a serial effectful map**: [`std/list.ail:217`](../../../std/list.ail) `mapE[a, b, e](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}`, with a sibling `filterE`/`foldE`. The header comment at `std/list.ail:214` states the contract we are parallelizing: *"All effectful combinators evaluate elements left-to-right, sequentially."* `parMap` is the **concurrent counterpart of `mapE`** — same type, relaxed evaluation order (bounded concurrency), same input-ordered result.

Using AILANG's actual effect-row-polymorphic syntax (modeled on `mapE`):

```ailang
-- Bounded concurrent effectful map. `n` = max in-flight evaluations.
export func parMapN[a, b, e](f: (a) -> b ! {e}, xs: [a], n: int) -> [b] ! {e}

-- Convenience: n = NumCPU. CPU-oriented; AI-bound callers should use parMapN.
export func parMap[a, b, e](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}

-- Collect-all variant (total; no cancellation). Ship this first.
export func parMapResult[a, b, e](f: (a) -> b ! {e}, xs: [a], n: int) -> [Result[b, Error]] ! {e}
```

The **key type-system property**: the effect row `{e}` of the mapped function is the effect row of the whole call — identical to how `mapE` is typed. Concurrency does not launder effects (A3). `parMapN(fetch, urls, 4)` where `fetch: (Url) -> Body ! {Net}` has type `[Body] ! {Net}`.

**Naming decision (surfaced by the `mapE` convention):** the existing serial combinators use an `E` suffix (`mapE`/`filterE`/`foldE`). A strict-convention name would be `parMapE`. This doc proposes `parMap`/`parMapN` (no `E`) because the concurrent map is *inherently* effectful — there is no pure `parMap` to disambiguate from, and `parMap` reads cleaner at call sites. See Design Freeze.

### Evaluator implementation (leverages existing Fork)

```go
// internal/eval — new concurrent apply path, callable from a builtin bridge.
//
// Precondition: base evaluator is fork-safe (M-SERVE-API-CONCURRENCY). Each
// element evaluates on its OWN Fork() so there is no shared mutable eval state.
func (e *CoreEvaluator) ParMap(fn Value, xs []Value, workers int) ([]Value, error) {
    if workers <= 0 {
        workers = runtime.NumCPU()
    }
    out := make([]Value, len(xs))          // indexed → input-order return (A1)
    errs := make([]error, len(xs))
    sem := make(chan struct{}, workers)    // bounded pool
    var wg sync.WaitGroup

    for i := range xs {
        sem <- struct{}{}
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            defer func() { <-sem }()
            child := e.Fork()               // isolated evaluator per element
            v, err := child.Apply(fn, []Value{xs[idx]})
            out[idx], errs[idx] = v, err
        }(i)
    }
    wg.Wait()

    // Fail-fast variant: return first error in input order.
    for _, err := range errs {
        if err != nil {
            return nil, err
        }
    }
    return out, nil
}
```

**Key design points:**
- Each element runs on its own `Fork()` — the property M-SERVE-API-CONCURRENCY already race-tested. No new concurrency invariant is introduced.
- Results land in an **indexed slice**, so the returned list is input-ordered regardless of which goroutine finishes first (A1/A2).
- The semaphore bounds in-flight work — this is the "bounded worker pool" the harness wants; no unbounded goroutine storm over `! {AI}`.
- Effect handlers: the `EffContext` must be fork-safe or per-Fork. Effect ops that touch shared external state (`FS` writes to the same path, `AI` budget counters) need their handlers to be concurrency-safe — see Risks.

### Type-checker / effect propagation

- Register `parMap`/`parMapN`/`parMapResult` with schemes that unify the callee's effect row into the result's effect row (mirror how the existing `mapE` is typed — it already does exactly this, `(a) -> b ! {e}` → `[b] ! {e}`).
- `parMap` over a pure `f` stays pure (empty effect row) — the primitive adds no effects of its own.

### Files to Create/Modify

**M0 — Fork concurrency-safety gate (PREREQUISITE; blocks everything below):**
- `internal/effects/context.go` (~+40 LOC) — `EffContext.Clone()` ([context.go:636](../../../internal/effects/context.go)): per-fork-isolate `AI` (`lastRoute`) and `IOWriter`; keep `Budget`/`BudgetReport` **shared but mutex-guarded**
- `internal/effects/budget.go` (~+20 LOC) — add a mutex to `BudgetContext`/`BudgetReport` map mutations ([budget.go:231](../../../internal/effects/budget.go), :419)
- `internal/ai/handler.go` (~+5 LOC) — guard/isolate `lastRoute` ([handler.go:39](../../../internal/ai/handler.go))
- `internal/effects/*_concurrent_test.go` (~+80 LOC) — `-race` tests fanning N forks through the Budget gate + AI/IO; **must pass before M1**
- *Side benefit: this also hardens the existing serve-api one-fork-per-request path.*

**M1 — Core primitive (after M0 green):**
- `internal/eval/eval_parmap.go` (~+120 LOC) — `ParMap` concurrent apply on Fork()
- `internal/builtins/parmap.go` (~+80 LOC) — builtin bridge (`_par_map`, `_par_map_n`, `_par_map_result`)
- `internal/runtime/builtins.go` (~+6 LOC) — register builtins
- `std/list.ail` or new `std/par.ail` (~+40 LOC) — `parMap`/`parMapN`/`parMapResult` surface + doc comments

**Type system:**
- Type schemes for the three functions with effect-row propagation (`internal/types` / builtin scheme registration)

**Docs/examples:**
- `examples/par_map.ail` (~30 LOC) — fan out N `std/ai` calls, collect in order
- `docs/docs/guides/concurrency.md` (~+60 LOC) — `parMap` section, "batch is sequential; parMap is in-program fan-out"
- CLI/prompt: ensure `ailang prompt` / stdlib docs surface it (findability was half the original gap)

---

## Examples

### In-program AI fan-out (the motivating case)

```ailang
module harness/eval

import std/ai (callJson)
import std/par (parMapResult)

type Case = { id: string, prompt: string }

export func run(cases: [Case]) -> [Result[string, Error]] ! {AI} {
  -- up to 8 model calls in flight; budget + telemetry stay in this process
  parMapResult(\c. callJson(c.prompt), cases, 8)
}
```

Wall-clock: `ceil(N/8) × latency` instead of `N × latency`. Compare the `map` version, which is serial.

### Bounded network fetch

```ailang
import std/par (parMapN)

func fetchAll(urls: [string]) -> [string] ! {Net} =
  parMapN(\u. httpGet(u), urls, 16)   -- 16 concurrent, results in URL order
```

---

## Success Criteria

- [ ] `parMapResult(f, xs, n)` evaluates up to `n` elements concurrently, each on its own Fork
- [ ] Returned list is in **input order** for all three functions (property test: shuffle completion via variable sleeps, assert order)
- [ ] Effect row propagates: `parMapN(_, xs, n)` over `! {AI}` has type `[..] ! {AI}`; over a pure fn is pure
- [ ] `parMap` over a pure function equals `map` over that function (same result)
- [ ] `go test -race ./internal/eval/... ./internal/builtins/...` passes
- [ ] Wall-clock benchmark: N=32 items × ~200ms sleep, `parMapN(8)` ≈ `4 × 200ms`, not `32 × 200ms`
- [ ] AI budget accounting stays correct under concurrency (no lost/double-counted spend)
- [ ] `examples/par_map.ail` runs and is in `make verify-examples`

---

## Conflict Surface

*(Required — this change touches `internal/eval/`, `internal/types/` (builtin schemes), `internal/effects/` (handler concurrency), and `internal/builtins/`.)*

### Syntactic / semantic positions touched

This milestone adds **no grammar** (A8 = 0). It touches three semantic positions:

1. **Builtin/identifier namespace** — three new stdlib names `parMap`, `parMapN`, `parMapResult` in `std/` (new `std/par.ail` or appended to `std/list.ail`), backed by builtins `_par_map*` in `internal/builtins/`.
2. **Type-scheme registration** — new schemes with effect-row-polymorphic signatures unified into the result row (the same machinery that types `mapE`).
3. **Evaluator apply path** — a concurrent `Apply` over `Fork()`ed child evaluators in `internal/eval/`.

### What else lives in each position

| Position | Existing valid form | Shape / risk |
|----------|--------------------|--------------|
| Identifier `parMap*` in `std/` | **None** — verified free (see Verification Log) | No collision |
| Effectful list-map scheme | `mapE`/`filterE`/`foldE` ([std/list.ail:217](../../../std/list.ail)) — `(a) -> b ! {e}` serial | `parMap` reuses the *same* scheme shape; must not change how `mapE` types |
| `CoreEvaluator` evaluation | Serial `Eval`/`Apply`; `Fork()` used only per-HTTP-request ([entrypoint.go:96](../../../internal/runtime/entrypoint.go)) | `Fork()` was designed for one-fork-per-goroutine; N concurrent forks from *one* parent is a new usage pattern — must confirm the parent evaluator is safe to `Fork()` from N goroutines concurrently (not just to hand one Fork to one goroutine) |
| Effect handlers under concurrency | `AI`, `IO`, `Budget` handlers in `internal/effects/` assume single-threaded eval | **Primary conflict — AUDIT DONE, see below.** `EffContext.Clone()` is a shallow one-level copy that shares `AI`/`Budget`/`BudgetReport`/`IOWriter` **by pointer**. `Budget` is the killer: `CheckAndConsume` ([budget.go:231](../../../internal/effects/budget.go)) does unsynchronized map writes on *every* effect call → N concurrent forks trigger `fatal error: concurrent map writes` (a panic, not a silent race). `FS`/`Net` handlers are stateless-safe (fresh `http.Client` per call, direct `os` calls) but still pass through the shared Budget gate. |

### Disambiguation / soundness

- **No parser disambiguation needed** — `parMap` is an ordinary identifier applied by ordinary call syntax; nothing in the grammar changes, so no lookahead question arises (contrast M-TAINT-TYPES).
- **Type soundness**: `parMap`'s scheme is `mapE`'s scheme; if `mapE` types soundly today, `parMap` does too. The only new type obligation is that the effect row is *preserved*, not widened/narrowed.
- **Concurrency soundness** rests entirely on: (a) each element evaluates on its own `Fork()` (no shared eval state), and (b) effect *handlers* are made concurrency-safe. (b) is the gate, and the audit (below) confirms it is **not** satisfied today: `Fork()` re-isolates only `env`/`resolver`/`recursionDepth`/`FnCaller` ([eval_evaluator.go:142](../../../internal/eval/eval_evaluator.go)); all effect state is shared by the shallow `Clone()`.

### Effect-handler concurrency audit (M1 gate — DONE 2026-07-15)

Verified against the code. `EffContext.Clone()` ([context.go:636](../../../internal/effects/context.go)) is `clone := *ctx` — a single-level struct copy, so every pointer/map field is shared across forks.

| Effect | State under concurrency | Verdict | Fix |
|--------|------------------------|---------|-----|
| **Budget** (gates ALL effects) | `BudgetContext`/`BudgetReport` plain maps, no mutex, mutated per effect call ([budget.go:231](../../../internal/effects/budget.go), `RecordUsage` :419) | 💥 **panics** — `fatal error: concurrent map writes` | Mutex on the maps **or** the shared-budget design decision below |
| **AI** | `Handler.lastRoute` written per call, shared `*AIContext` ([handler.go:39](../../../internal/ai/handler.go)) | racy (write/read data race) | deep-copy `AIContext` in `Clone()`, or mutex |
| **IO** | shared `IOWriter` ([io.go:49](../../../internal/effects/io.go), [context.go:600](../../../internal/effects/context.go)) | interleaves on stdout; corrupts if a shared `*bytes.Buffer` is installed | per-fork writer, or a locked writer |
| **FS** | none — direct `os` calls, only reads `ctx.Env.Sandbox` | ✅ safe at handler level | — (inherits Budget gate) |
| **Net** | none — fresh `http.Client`+`Transport` per call ([net.go:91](../../../internal/effects/net.go)) | ✅ safe at handler level | — (inherits Budget gate) |

**Conclusion:** no effect is architecturally single-threaded — every unsafe one is a shared-field/shared-map mutation fixable by isolating it in `Clone()` or adding a lock. But the fix is a **hard M1 gate**: until `Clone()` deep-copies (or locks) `Budget`/`BudgetReport`/`AI`/`IOWriter`, *any* concurrent effectful `parMap` panics on the Budget map. This also hardens the existing serve-api Fork path, which today is safe only by avoiding these effects.

### Programs that MUST still work (regression fixtures)

These exercise the serial effectful-combinator path and the evaluator's existing `Fork()` usage; none may regress:

1. [std/list.ail](../../../std/list.ail) `mapE`/`filterE`/`foldE` — the serial combinators keep their left-to-right semantics
2. Any example using `mapE` over `! {IO}` (e.g. printing each element in order) — serial ordering unchanged
3. serve-api concurrent request handling ([entrypoint.go:96](../../../internal/runtime/entrypoint.go)) — the existing one-Fork-per-request path must be untouched
4. `std/ai.ail` single-shot `call`/`step` — unchanged; `parMap` wraps them, does not modify them
5. Existing `go test -race ./internal/eval/...` — must still pass (plus new concurrent-apply race tests)

### What deliberately changes

- **Nothing existing changes behavior.** All three functions are additive. No previously-valid program is intended to break.
- The *new* contract explicitly introduced: `parMap` does **not** guarantee ordering of observable side effects across elements (only results are ordered). This is a new promise, not a change to an existing one.

---

## Verification Log

Claims in this doc verified against the live codebase (2026-07-15, v0.29.2-206-gefd251f16):

| Claim | Method | Result |
|-------|--------|--------|
| `parMap`/`parMapN`/`parMapResult`/`gather`/`_par_map` identifiers are free | `grep -rn` over `std/`, `internal/builtins/`, `internal/runtime/builtins.go` | **Zero matches** — names available |
| A serial effectful map already exists | Read [std/list.ail:217](../../../std/list.ail) | `mapE[a,b,e](f:(a)->b!{e}, xs:[a])->[b]!{e}` confirmed; header at :214 states combinators run "left-to-right, sequentially" |
| Batch mode is a sequential loop | Read [main_run_exec.go:313](../../../cmd/ailang/main_run_exec.go) | Plain `for i, input := range programArgs`; no goroutines |
| `selectEvents` is event-source-shaped | Read [stream_mux.go:26](../../../internal/effects/stream_mux.go) | `reflect.Select` priority merge of persistent sources; not request/response fan-out |
| `Fork()` exists but is Go-internal, not AILANG-callable | Read [eval_evaluator.go:142](../../../internal/eval/eval_evaluator.go), caller [entrypoint.go:96](../../../internal/runtime/entrypoint.go) | Confirmed; only caller is per-HTTP-request isolation |
| `std/ai` calls are single-shot | Read [std/ai.ail](../../../std/ai.ail) | `call`/`callJson`/`step`/`stepWithStream`/`runTools` all `! {AI}`, one request each |
| Duplicate design docs | `ailang docs search` (SimHash) | Top hits are codegen/type docs (keyword false-positives); no concurrency/parMap doc exists — not a duplicate |
| Effect handlers safe under N concurrent forks? | Read `EffContext.Clone()` ([context.go:636](../../../internal/effects/context.go)), `BudgetContext` ([budget.go:231](../../../internal/effects/budget.go)), `ai.Handler` ([handler.go:39](../../../internal/ai/handler.go)), `io.go`, `net.go`, `fs.go` | **NO — confirmed.** `Clone()` shallow-shares `Budget`/`AI`/`IOWriter`. Budget maps are mutated unsynchronized per effect call → `concurrent map writes` **panic**. AI/IO racy. FS/Net handler-safe. Full table in Conflict Surface → "Effect-handler concurrency audit". |

**Gate status:** the effect-handler audit is **DONE** (was the deferred pre-condition). Result: the concurrent path panics today on the shared Budget map; M1 must fix `EffContext.Clone()` (shared-synchronized Budget, per-fork AI/IO) with `-race` coverage before any `parMap` surface is enabled. This is now a concrete, scoped task, not an unknown.

---

## Testing Strategy

**Unit / property:**
- Order preservation: elements with randomized (seeded) work durations, assert output index == input index
- Purity: `parMap` over pure fn ≡ `map`; type is pure
- Effect propagation: type-check asserts `! {AI}` survives
- Bound: instrument a counter of concurrent evaluations, assert it never exceeds `n`

**Race:**
- `go test -race` on eval + builtins; a test that fans out effectful closures touching the AI budget counter

**Integration:**
- `examples/par_map.ail` with a stub AI handler (deterministic), assert ordered results and that a bounded pool is respected

---

## Open Questions

1. **Builtin vs `Fork` effect op.** A `Fork` effect (`spawn`/`join` ops) would be more general (arbitrary concurrency graphs) but forces every caller to declare `! {Fork}` and needs a scheduler. `parMap` as a builtin is narrower, needs no new effect, and covers the harness use case. **Leaning builtin**; a general `Fork` effect is its own future milestone.
2. **Cancellation on fail-fast `parMap`.** Cancelling in-flight `! {AI}` calls needs `context.Context` threading into effect handlers. `parMapResult` (collect-all, no cancellation) sidesteps this — hence "ship it first."
3. **Nested `parMap`.** Fork-of-a-Fork must stay bounded to avoid `workers²` blowup. Initial stance: document that nested `parMap` multiplies bounds; consider a global in-flight cap later.
4. **Determinism of effect *side effects*.** Order-preserving *results* is guaranteed. Order of observable side effects (e.g. `IO` prints) across concurrent elements is **not** — document this; effect-side-effect ordering is not part of the contract.

---

## Non-Goals

- **General async/await or a `Fork` effect with arbitrary spawn/join** — larger language-concurrency milestone; `parMap` is the narrow, high-value slice.
- **Batch-mode `--parallel`** — that is [M-CONCURRENCY-LEVERAGE](../v0_29_0/m-concurrency-leverage.md) (harness-level, Go-side). This doc is the *in-AILANG-program* primitive.
- **Distributed execution** — across machines; out of scope.
- **Guaranteed ordering of concurrent side effects** — only results are ordered.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Shared Budget map panics under concurrency (`concurrent map writes`) | **High — CONFIRMED, not hypothetical** | Audit done (Conflict Surface). M0 gate: fix `EffContext.Clone()` — mutex-guard the shared `Budget`/`BudgetReport` maps ([budget.go:231](../../../internal/effects/budget.go)), per-fork-isolate `AI.lastRoute` + `IOWriter`. `-race` test that fans out N effectful closures through the Budget gate. **No `parMap` surface ships until this passes.** Side benefit: hardens the serve-api Fork path. |
| `Clone()` fix regresses serve-api one-fork-per-request path | Med | The existing path shares the same `Clone()`; regression fixtures (Conflict Surface #3) + `go test -race ./internal/runtime/...` pin it |
| Unbounded fan-out over `! {AI}` hits rate limits / OOM | Med | No unbounded `parMap` over AI: document `parMapN` as the AI path; `parMap` default = NumCPU |
| Nested `parMap` → `workers²` goroutines | Med | Document; consider a process-global in-flight semaphore in a follow-up |
| Non-deterministic side-effect ordering surprises users | Low | Explicit contract: results ordered, side effects not; example + doc call-out |
| WASM build has no goroutine scheduler parity | Low | Provide a sequential fallback for `//go:build js` (parMap ≡ map), documented |

---

## Related Documents

- [M-CONCURRENCY-LEVERAGE](../v0_29_0/m-concurrency-leverage.md) — Harness-level parallelism (batch/eval/coordinator). This doc realizes its deferred "Intra-request parallelism" / "Async/await separate design doc" items.
- [M-SERVE-API-CONCURRENCY](../../implemented/v0_9_4/m-serve-api-concurrency.md) — Foundation: `Fork()` + thread-safe `Environment` (the substrate this reuses).
- [M-PERF7: DocParse Production Pipeline](../v0_9_3/m-perf7-docparse-production-pipeline.md) — Batch mode origin (sequential-by-design).
- `std/ai.ail` — Single-shot AI calls this primitive fans out.
- `internal/effects/stream_mux.go` — `selectEvents` (event-source concurrency; the *other* shape).

---

## Future Work

- **General `Fork` effect** — `spawn`/`join` for arbitrary concurrency graphs.
- **Cancellation** — `context.Context` into effect handlers → true fail-fast `parMap` that stops in-flight AI calls.
- **`parReduce` / `parFilter`** — the rest of the concurrent-collection family once `parMap` proves the substrate.
- **Adaptive bound** — auto-tune worker count from observed latency / rate-limit headers.

---

**Document created**: 2026-07-15
**Last updated**: 2026-07-15

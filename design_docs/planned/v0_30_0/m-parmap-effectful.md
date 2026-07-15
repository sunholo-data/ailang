# M-PARMAP-EFFECTFUL: Concurrent Map over Effectful Functions (`parMap` / `gather`)

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (High — the one primitive an in-AILANG eval/agent harness actually needs)
**Estimated**: 4-6 days
**Dependencies**: M-SERVE-API-CONCURRENCY (Fork() + thread-safe Environment — complete, shipped v0.9.4)
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
| A9: Cost Visibility | +1 | Bounded worker pool (`parMapN`) makes the concurrency ceiling explicit; per-element AI cost stays attributed in-program |
| A10: Composability | +1 | `parMap` composes with any effectful `a -> b ! e`, uniformly across `AI`, `Net`, `FS`, `IO` |

**Net Score: +8** → **Decision: Move forward**

---

## Problem Statement

The concurrency substrate exists but is not surfaced to AILANG user code. Forensic audit of the codebase (2026-07-15):

| Capability | State | Evidence |
|------------|-------|----------|
| Batch mode overlaps AI calls | **No** — strictly sequential | `cmd/ailang/main_run_exec.go:313` — plain `for i, input := range programArgs` calling `executeBatchItem` synchronously; no goroutines/waitgroup/semaphore |
| `Stream`/`selectEvents` fan-out | **No** — event-source-shaped | `internal/effects/stream_mux.go:26` — consumer-side priority merge of persistent sources (stdin, WS/SSE, subprocess stdout) via `reflect.Select`; cannot issue N outbound calls + gather N responses |
| `asyncExecProcess` | Exists — subprocess streaming source | `internal/effects/stream_process.go:44` — one process, stdout → events; not a collection fan-out |
| Go-internal `Fork()` | Exists — **not** AILANG-callable | `internal/eval/eval_evaluator.go:142`; sole caller `internal/runtime/entrypoint.go:96` (per-HTTP-request isolation in serve-api) |
| AILANG-callable `parMap`/`gather`/`Fork` | **Missing** | Zero builtin/stdlib matches across `internal/builtins/`, `internal/runtime/builtins.go`, `stdlib/**` |
| `std/ai` concurrency | **None** | `stdlib/std/ai.ail` — `call`/`callJson`/`step`/`stepWithStream`/`runTools` are all single-shot `! {AI}` |

### The concrete shape that has no home today

An eval/agent harness written in AILANG wants: *given `xs: [Input]` and `f: Input -> Output ! {AI}`, run all `f(x)` concurrently with a bounded worker pool, collect `[Output]` in input order, keep budget/telemetry in-program.*

- `map(f, xs)` runs them serially — for N items at ~1-3s of network latency each, wall-clock is `N × latency`.
- The only current workaround is process-level: spawn N `ailang run` subprocesses via `asyncExecProcess` and merge with `selectEvents`. That works but loses in-program budget/telemetry, is more code, and is throwaway once this primitive lands.

**The hard part is already built.** `CoreEvaluator.Fork()` + thread-safe `Environment` (M-SERVE-API-CONCURRENCY, race-tested, shipped) give each concurrent evaluation an isolated evaluator. This milestone *surfaces existing machinery* as a language primitive — it is not building concurrency from scratch.

---

## Goals

**Primary Goal:** Provide an AILANG-callable `parMap` (and bounded `parMapN`) that evaluates an effectful function concurrently across a list, returning results in input order, backed by the existing Fork()/Environment substrate.

**Success Metrics:**
- `parMapN(8, f, xs)` runs up to 8 elements of `xs` concurrently, each on its own Fork
- Results are returned in **input order**, independent of completion order (determinism)
- The mapped function's effect row propagates: `parMap : (a -> b ! e) -> [a] -> [b] ! e`
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

### Design Freeze

- [ ] **Surface**: plain builtin `parMap`/`parMapN` typed `(a -> b ! e) -> [a] -> [b] ! e` (leading candidate — see Open Questions for the `Fork` effect-op alternative)
- [ ] **Error semantics**: `parMap` is fail-fast → `[b] ! e` (propagates the first error, cancels the rest); a `parMapResult` variant returns `[Result[b, Error]]` for collect-all. **Recommendation: ship `parMapResult` first** (total, no cancellation semantics to design), add fail-fast `parMap` second.
- [ ] **Bound**: `parMap` = `runtime.NumCPU()` for CPU-bound; `parMapN(n, ...)` for explicit bound. **AI-bound callers should always use `parMapN`** — document `parMap` as CPU-oriented.
- [x] **Determinism**: results **always** returned in input order (indexed collection, not completion order).

---

## Solution Design

### The type

```ailang
-- Bounded concurrent map. `n` = max in-flight evaluations.
parMapN : (a -> b ! e) -> [a] -> int -> [b] ! e

-- Unbounded-by-CPU convenience (n = NumCPU). CPU-oriented.
parMap  : (a -> b ! e) -> [a] -> [b] ! e

-- Collect-all variant (total; no cancellation). Ship this first.
parMapResult : (a -> b ! e) -> [a] -> int -> [Result[b, Error]] ! e
```

The **key type-system property**: the effect row `e` of the mapped function is the effect row of the whole call. Concurrency does not launder effects (A3). A `parMapN(4, fetch, urls)` where `fetch : Url -> Body ! {Net}` has type `[Body] ! {Net}`.

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

- Register `parMap`/`parMapN`/`parMapResult` with schemes that unify the callee's effect row into the result's effect row (mirror how `map` is typed, but preserve `! e` instead of requiring purity).
- `parMap` over a pure `f` stays pure (empty effect row) — the primitive adds no effects of its own.

### Files to Create/Modify

**Core:**
- `internal/eval/eval_parmap.go` (~+120 LOC) — `ParMap` concurrent apply on Fork()
- `internal/builtins/parmap.go` (~+80 LOC) — builtin bridge (`_par_map`, `_par_map_n`, `_par_map_result`)
- `internal/runtime/builtins.go` (~+6 LOC) — register builtins
- `stdlib/std/list.ail` or new `stdlib/std/par.ail` (~+40 LOC) — `parMap`/`parMapN`/`parMapResult` surface + doc comments

**Type system:**
- Type schemes for the three functions with effect-row propagation (`internal/types` / builtin scheme registration)

**Effect safety:**
- Audit `internal/effects/*` handlers for concurrency safety under Fork (AI budget counter, FS, Net, IO ordering) — add mutexes / per-Fork state where a handler mutates shared state

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
| Effect handlers not concurrency-safe under Fork (AI budget, FS, Net) | **High** | Audit `internal/effects/*`; add mutex/per-Fork state; a `-race` test that hammers the AI budget counter. Gate the milestone on this audit. |
| Unbounded fan-out over `! {AI}` hits rate limits / OOM | Med | No unbounded `parMap` over AI: document `parMapN` as the AI path; `parMap` default = NumCPU |
| Nested `parMap` → `workers²` goroutines | Med | Document; consider a process-global in-flight semaphore in a follow-up |
| Non-deterministic side-effect ordering surprises users | Low | Explicit contract: results ordered, side effects not; example + doc call-out |
| WASM build has no goroutine scheduler parity | Low | Provide a sequential fallback for `//go:build js` (parMap ≡ map), documented |

---

## Related Documents

- [M-CONCURRENCY-LEVERAGE](../v0_29_0/m-concurrency-leverage.md) — Harness-level parallelism (batch/eval/coordinator). This doc realizes its deferred "Intra-request parallelism" / "Async/await separate design doc" items.
- [M-SERVE-API-CONCURRENCY](../../implemented/v0_9_4/m-serve-api-concurrency.md) — Foundation: `Fork()` + thread-safe `Environment` (the substrate this reuses).
- [M-PERF7: DocParse Production Pipeline](../v0_9_3/m-perf7-docparse-production-pipeline.md) — Batch mode origin (sequential-by-design).
- `stdlib/std/ai.ail` — Single-shot AI calls this primitive fans out.
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

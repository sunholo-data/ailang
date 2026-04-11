# Latency Budget Workloads

Canonical, self-contained AILANG programs used as **latency SLO probes**. Each
workload exercises a different user-visible code path; together they form the
release-gating "embarrassment line" that AILANG promises not to regress past.

This is **suite 3** of `benchmarks/` (see [`../README.md`](../README.md) for the
three-sibling layout). It is intentionally distinct from:

- `benchmarks/*.yml` — AI code-generation evals (measure model quality, not runtime)
- `benchmarks/runtime/*.ail` — micro-benchmarks for individual interpreter paths

Workloads here are measured end-to-end: parse → elaborate → typecheck →
evaluate → IO. They represent shapes a real user would write.

## Why self-contained

The original M-LAT-BUDGET design picked DocParse-on-EPUB as the data-intensive
workload because that was the workload that recently regressed (the
M-PERF6B tracing bug doubled DocParse latency until it was caught). DocParse
lives in the external `sunholo/ailang_parse` package, though, and pulling that
dependency into the AILANG benchmark loop made the suite slow to set up,
fragile across machines, and impossible to run in CI without a checkout step.

We pivoted to **self-contained `.ail` workloads** that exercise the same hot
paths the DocParse regression hit (per-element evaluator step cost, std/list
pipelines, IO effect dispatch) while keeping the benchmarks reproducible
anywhere a `make build` works. The list-pipeline workloads (`list_small`,
`list_large`) are the regression canaries for tracing-style "every step is
2× slower" bugs — see comments inside `list_large.ail`.

## The catalog

| Workload | Hot paths exercised | Why it's in the suite |
|----------|---------------------|----------------------|
| [`cold_hello.ail`](cold_hello.ail) | Parser, elaborator, typecheck, evaluator init, IO | Cold-start floor — any release that adds startup latency shows here first |
| [`warm_eval.ail`](warm_eval.ail) | Pure recursive int arithmetic (`fib(24)`) | Warm evaluator hot loop — closures + integer ops with no IO noise |
| [`typecheck_heavy.ail`](typecheck_heavy.ail) | Multiple ADTs, exhaustive matching, polymorphic instantiation | Stresses the type checker and pattern-match elaborator without exercising the runtime hot loop |
| [`effect_roundtrip.ail`](effect_roundtrip.ail) | IO effect dispatch (50× `println` round-trips) | Catches regressions in capability checks and effect handler dispatch |
| [`list_small.ail`](list_small.ail) | `std/list` `map` + `filter` + `foldl` over 500 ints | Companion to `list_large` — together they distinguish constant overhead from per-element cost |
| [`list_large.ail`](list_large.ail) | `std/list` pipeline over 5,000 ints | **Regression canary** — the M-PERF6B tracing bug doubled latency on workloads of this exact shape |

The size of `list_large` is capped at 5,000 deliberately: `buildList` is
tail-recursive in form but the AILANG evaluator still tracks frames, so larger
inputs trip the 10,000-frame default recursion depth limit. 5k is 10× larger
than `list_small`, which is enough spread to make the per-element cost
dominate the constant startup cost in measurements.

`list_small` and `list_large` are pure functions returning `int` (not
`! {IO}`) for the same reason — wrapping `buildList` inside an IO main was
adding enough additional frames to push past the recursion limit.

## How to run

Single workload:

```bash
AILANG_NO_TRACE=1 ailang run --caps IO --entry main benchmarks/workloads/warm_eval.ail
```

Full suite with timing capture (after M2 lands):

```bash
tools/bench_workloads.sh                # 5 runs each, write benchmarks/latency_budgets.json
tools/bench_workloads.sh --runs 10 --workload list_large
make bench-workloads                    # convenience target
```

`AILANG_NO_TRACE=1` is **mandatory** for all measurement runs. The default
trace path adds ~2× overhead and the budget numbers are meaningless without
it. The harness sets it automatically; only set it manually if you're running
a workload by hand for debugging.

## Adding a new workload

A workload qualifies for the canonical suite if **all** of these are true:

1. It is self-contained — no external packages, no on-disk inputs, no network.
2. It exercises a *user-visible* code path (parser, typecheck, evaluator, IO,
   stdlib). Internal-only paths belong in `benchmarks/runtime/`, not here.
3. Its runtime is dominated by AILANG, not by `os.Exit` or `time.Sleep`.
4. It has a stable, deterministic output that can be diffed across runs.
5. Its p95 on a developer laptop is under ~10 seconds. Longer workloads
   defeat the purpose of a fast feedback loop.

When you add one, also:

- Add a row to the catalog table above and to `benchmarks/budget_ledger.md`
  with an initial p95 target (use the first measured value, not an aspiration).
- Add the workload to `tools/bench_workloads.sh`'s default list.
- Note in the workload's header comment **what hot path it exercises** and
  **why** — future-you (and the next sprint) will thank you.

## The latency-budget process

Each canonical workload has a **p95 target** recorded in
`../latency_budgets.json` (auto-generated) and `../budget_ledger.md`
(hand-edited, authoritative). When an optimization reduces a workload's p95
by Xms:

- **50%** of the saving tightens the workload's target by X/2 (the user gets
  a faster release).
- **50%** goes to a "dev pool" available to spend on future features that
  cost latency, recorded in the ledger as a positive balance.

When a feature design doc claims a positive Δ on a workload, it must either
offset it with an optimization, ship behind a flag, draw from the dev pool,
or cite an axiom that justifies the cost. See the M-LAT-BUDGET design doc
for the full process: [`../../design_docs/planned/v0_11_2/m-latency-budget.md`](../../design_docs/planned/v0_11_2/m-latency-budget.md).

# AILANG Latency Budget Ledger

**Authoritative source for canonical workload p95 targets and the dev pool balance.**

This file is hand-edited and append-only. Every optimization deposits to the
pool; every feature with a positive Δ withdraws from it. The accounting rule
is **50/50**: half of every saving tightens the workload's user-visible
target, half credits the pool.

> Workload definitions and run instructions:
> [`benchmarks/workloads/README.md`](workloads/README.md)
>
> Auto-generated runtime measurements (do not edit by hand):
> [`benchmarks/latency_budgets.json`](latency_budgets.json)

---

## Process (one screen, no skipping)

1. Optimization lands → run `make bench-workloads` → diff against the previous
   `latency_budgets.json` → record the *p95 delta* per workload here.
2. **Apply the 50/50 rule.** Each `Δms` saving:
   - **half** tightens the workload's `target_p95_ms` (the user gets a
     faster release)
   - **half** credits the pool
3. Feature with positive Δ → withdraw from pool, record reason. If the pool
   can't cover it, the design doc must offset, flag-gate, or cite an axiom.
4. The pool can never go negative without an explicit "Accepted regression"
   row signed off in the design doc — see the M-LAT-BUDGET design doc for
   the gate criteria.

---

## Current Targets

These are the **release-gating p95 numbers**. A release that misses any of
them must either fix the regression, gate the feature, or write an explicit
ledger row accepting the slip.

Initial baseline captured **2026-04-12** on Apple M2 (macOS / Darwin arm64,
go1.24.11, AILANG `v0.11.1-1-gc08191df-dirty`, commit `c08191df`). Targets are
set from the first measurement plus headroom — they are **not aspirations**,
they are *anti-regression contracts*.

| Workload | Baseline p95 (ms) | Target p95 (ms) | Headroom | Notes |
|----------|------------------:|----------------:|---------:|-------|
| `cold_hello`       |  437 |  600 | +37%  | Cold-start floor; dominated by parser + linker startup |
| `warm_eval`        | 1124 | 1300 | +16%  | `fib(24)` warm loop; the warm evaluator hot path |
| `typecheck_heavy`  |  431 |  600 | +39%  | ADTs + exhaustive matching; type checker stress |
| `effect_roundtrip` |  592 |  750 | +27%  | 50× IO effect dispatch round-trips |
| `list_small`       |  461 |  600 | +30%  | std/list pipeline, 500 ints; constant-cost reference |
| `list_large`       |  733 |  900 | +23%  | **regression canary** — list pipeline, 5,000 ints |

Headroom is intentionally generous on the first baseline because we don't
yet know the noise floor on Apple M2. After ~2 releases of soft-gate data
the headroom should be tightened toward ~10–15%.

> **Hardware caveat.** All numbers above are same-machine. Comparing across
> hardware is not supported — `latency_budgets.json` records the CPU/OS/Go
> fingerprint so a different machine's measurements never overwrite this
> baseline silently. CI gating (Phase 4) will pin a single machine class.

---

## Dev Pool Balance

| Date | Δms (running) | Source | Notes |
|------|--------------:|--------|-------|
| 2026-04-12 | +1887 | M-PERF6B back-fill (see below) | Seed entry — `list_large` workload |
| **Total** | **+1887** | | |

The pool starts with a back-filled credit from the M-PERF6B tracing fix (see
the next section). Future optimizations *prepend* rows with their commit and
the `Δms` they deposited.

---

## Ledger Entries

### 2026-04-12 — Initial baseline + M-PERF6B back-fill seed

**Captured baseline.** Six canonical workloads measured 5 runs each with
`AILANG_NO_TRACE=1`. Numbers are in the table above and in
[`latency_budgets.json`](latency_budgets.json).

**M-PERF6B back-fill — `list_large` saving = -3773 ms**

The M-PERF6B sprint discovered that AILANG's tracing path doubled latency on
data-intensive workloads like DocParse, and shipped `AILANG_NO_TRACE=1` (commit
`08ef7bbc`) so users could opt out. We never recorded what that fix bought us
in workload terms, so this is the back-fill — the same A/B replayed today on
`list_large` (which is the canonical regression canary for that exact bug
class).

A/B measured today on Apple M2 / commit `c08191df-dirty`, 5 runs each:

| Configuration | p50 (ms) | p95 (ms) | Source |
|---|---:|---:|---|
| Trace **ON** (default until M-PERF6B opt-out) | 4386 | 4506 | live re-measurement |
| Trace **OFF** (`AILANG_NO_TRACE=1`, current default for measurement) | 571 | 733 | `latency_budgets.json` |
| **Saving** | **−3815** | **−3773** | |

So tracing was costing roughly **6× the steady-state runtime** of `list_large`
(every evaluator step was paying a tracing tax). DocParse-on-EPUB was a
~2× regression because it spent more time in IO; pure compute workloads paid
the full tax.

**50/50 split applied to the p95 saving (3773 ms):**

| Allocation | Δms |
|---|---:|
| To user (target sits at 733 ms baseline + headroom; saving fully realised in shipped artifact) | -1887 |
| To dev pool (credit for future features that cost latency) | +1887 |

The user got the full saving in real terms because the workload now ships
under 1 second instead of over 4 seconds; the `+1887` pool entry is the
*accounting credit* the design-doc process can withdraw against in future
sprints when adding a feature with a positive latency cost.

**What to do with the pool credit (current sprint):**

Nothing yet — M-LAT-BUDGET is process work, not a feature, and this sprint
intentionally adds zero runtime cost. The pool sits at **+1887 ms** waiting
for the first hot-path feature design doc to draw against it.

---

## Open Items

- **No CI gating yet.** Today this is a cultural artifact; the gate moves
  to enforced in M-LAT-BUDGET Phase 4 once 2 releases of soft-gate data
  show the noise floor is stable.
- **Cross-machine comparability.** Currently undefined — a Linux Xeon would
  produce different numbers and **must not** overwrite this baseline.
  Phase 4 will pin a single machine class for the official release gate.
- **Workload coverage gaps.** No XML/JSON/parsing-heavy workload yet (was
  going to be DocParse before the self-contained pivot). Add when an
  appropriate self-contained probe exists — see
  [`workloads/README.md`](workloads/README.md) for the criteria.

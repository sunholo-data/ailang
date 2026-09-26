# M-TRACE-TIER-NOT-ENFORCED: the tracing tier is resolved, printed, and then ignored

**Status**: M1 + M3 IMPLEMENTED (commit `324373066`) — M2 (bounding the collector) outstanding.
Measured after: n=400 peak RSS 2059 MB -> **105 MB** (tracing off: 102 MB); `deep` unchanged at
2478 MB; the `string<secret>` canary went from 4 verbatim copies at `standard` to **0**.
Blast radius was smaller than feared: only 2 non-test construction sites, and `scorer`/`comparator`
have no non-test consumers, so `NewCollector()` defaulting to deep left all 32 test callers green.
**Still not fixed, and stated rather than implied:** effect args/results are recorded at
`standard`, so a `readFile` span still carries what was read — that is M-TRACE-LABEL-AWARE M2.
**Target**: v0.36.0
**Priority**: P0 — one defect, two P0 consequences: an OOM that killed a real workload, and an
information-exposure surface that two independent field reports have now filed against.
**Estimated**: 1.5–2 days (3 milestones, M1 alone fixes both consequences)
**Dependencies**: None. **Independent of** the parked
[m-list-cons-quadratic](../m-list-cons-quadratic.md) — see [Relationship to #676](#relationship-to-676-and-the-parked-list-doc).
**Author**: design-doc-creator role, attended session 2026-09-09, at `dev` = `6494a98f4` (v0.35.4+)
**Sources**: `inbox_1788936332313_c08c55b5` (holosun/decision-budget — the OOM) and
`fb_b023726953f2ee5a` (mcp-public — the exposure axis)

---

## Problem Statement

`AILANG_TRACE` / `--trace-tier` selects `off | standard | deep`. The tier is resolved in
`cmd/ailang/main_run_exec.go:463`, used to decide **whether a collector exists at all** and
which banner to print — and then never reaches the collector.

`trace.Collector` has **no tier field**. `RecordFunctionEnter`
([`collector.go:137`](../../../internal/trace/collector.go#L137)) gates only on `c.Enabled()`,
and [`eval_operations.go:123`](../../../internal/eval/eval_operations.go#L123) calls it for
**every function application**, passing fully-rendered argument strings. `RecordFunctionExit`
does the same with the rendered result.

So `standard` and `deep` record **identically** at the collector. The tier's own documentation
says otherwise:

```go
// TierStandard emits module, top-level effect, coordinator, executor,
// compile, task/chain-linked spans, but NOT per-call function spans or
// nested effect spans. This is the default.
```
— [`internal/trace/options.go:19-22`](../../../internal/trace/options.go#L19)

That comment describes a behavior the code does not implement, on the **default** tier.

### Consequence 1 — O(n²) memory, measured, OOMs real workloads

Every call records its arguments and its result as strings. A function that carries a growing
accumulator therefore serializes O(n) data on each of n iterations: **O(n²) trace bytes**, held
in memory by an unbounded `[]TraceEvent`.

The canonical shape is the one any "fold input into a list of blocks" loop has:

```ailang
export func build(n: int, i: int, acc: [Block]) -> [Block] =
  if i >= n then acc
  else build(n, i + 1, concat(acc, [{ kind: "para", text: "line ${show(i)}" }]))
```

Measured first-party, v0.35.4, darwin/arm64, `/usr/bin/time -l`, peak RSS:

| n | `standard` (default) | `off` | ratio |
|---|---|---|---|
| 100 | 101 MB | 70 MB | 1.4× |
| 200 | 341 MB | 85 MB | 4.0× |
| 400 | **2059 MB** | **106 MB** | **19×** |
| 800 | 3463 MB | 159 MB | 22× |

With tracing off the same program is near-linear in `n`. At n=200 the emitted trace is
**40,609 events / 87 MB** for a 200-iteration loop, with individual events up to 6,133 bytes —
each one a rendering of the whole accumulator:

```json
{"event":"function_enter","depth":201,"function":{"name":"std/list.concat",
 "args":["[{kind: para, text: line 0}, {kind: para, text: line 1}, ...
```

The reporting workload (an OOXML generator turning markdown into `.docx`) took ~240 s and peaked
at 1.1–2.6 GB on 49 KB of input, enough to cause system-wide memory pressure on a 24 GB machine.
`docparse --convert` does the same conversion in 1.49 s.

**`AILANG_TRACE_MAX_SPANS` does not bound this** (measured: 2217 MB at n=400 with
`MAX_SPANS=100`, versus 2074 MB without) — it governs the otel emitter, not the collector's
event slice.

### Consequence 2 — the exposure axis two reports have now converged on

`fb_b023726953f2ee5a` (a retraction of `fb_0003e02912bc5fa8`, which had wrongly cleared otel
after measuring a no-op) reports that **the leak axis is the tier, not the exporter**, and
measured `standard` shipping `ailang.effect.args` / `ailang.effect.result` — including a
refresh token and client secret — to an OTLP collector.

This doc's finding is the same defect seen from the other side, and it **widens** their result:
the collector renders and retains **every function's** args and results, pure functions
included, at `standard`. Their report expected pure-function values only at `deep`.

Two axes must be kept apart, because they have different fixes:

| | what it decides | status |
|---|---|---|
| **Tier** (this doc) | which values are *recorded and retained in memory* | not enforced — the defect here |
| **Exporter** | which recorded values *leave the machine* | measured independently in `fb_b0237`; **not re-measured here** |

This doc fixes recording. It makes no claim about what any exporter transmits.

---

## Goals

**Primary goal**: make the tier actually govern what the collector records, so `standard` means
what it already says it means.

**Success metrics:**

1. `standard` no longer records per-call function enter/exit events. n=400 peak RSS drops from
   ~2059 MB to within ~2× of `off`.
2. `deep` is unchanged — per-call spans are its documented purpose, and the training/profiling
   consumers depend on them.
3. The tier's doc comment and the code agree, with a test that fails if they diverge again.
4. The collector is bounded, so no tier can grow memory without limit.

---

## Solution Design

### Overview

The tier is already resolved before the collector is built. Pass it in, and gate the
function-level recorders on it.

```go
// today
effCtx.Trace = ailtrace.NewCollector()

// proposed
effCtx.Trace = ailtrace.NewCollectorWithTier(traceOpts.Tier)
```

```go
func (c *Collector) RecordFunctionEnter(name string, args []string) {
	if !c.Enabled() || c.tier < TierDeep {
		return
	}
	...
}
```

**Gate at the collector, not at the call site.** `eval_operations.go:123` renders `argStrs`
*before* calling in, so a call-site gate is also needed to avoid paying the rendering cost — but
the collector gate is the one that makes the invariant true for every caller, present and
future (`internal/builtins/trace.go:168` is a second one today).

### Implementation Plan

**M1 — enforce the tier** (~5h) — *fixes both consequences*
- [ ] `Collector` carries a `tier Tier`; `NewCollectorWithTier`, with `NewCollector` retained as
      `TierDeep` for compatibility with existing non-CLI callers **or** migrated outright —
      agent's call, see Deferred Decisions
- [ ] Gate `RecordFunctionEnter` / `RecordFunctionExit` on `tier >= TierDeep`
- [ ] Gate the argument *rendering* at `eval_operations.go:123` on the same predicate, so the
      cost is not paid and then discarded
- [ ] Wire `traceOpts.Tier` through at `main_run_exec.go:472`
- [ ] Regression test pinning the measured memory shape (see Testing Strategy)

**M2 — bound the collector** (~3h)
- [ ] Apply a cap to `Collector.events`, honoring `AILANG_TRACE_MAX_SPANS` on this path too
      (today it only reaches the otel emitter — measured, V6)
- [ ] On overflow, drop with a **loud** one-line notice naming the cap. A silently truncated
      trace is a measurement that lies; this is the CLAUDE.md no-silent-fallback rule applied to
      an instrument

**M3 — make the tiers legible** (~4h)
- [ ] A table test that asserts, per tier, exactly which event types the collector emits — the
      thing that would have caught this. The current doc comment was written as prose and
      drifted from the code with nothing to notice
- [ ] `docs/docs/guides/debugging.md`: state that `standard` is the default, what each tier
      records, and that recorded values include arguments and results
- [ ] Cross-reference the exposure axis from `fb_b0237` so the two are discoverable together

### Files to Modify

- `internal/trace/collector.go` — tier field + gates, ~30 LOC
- `internal/trace/options.go` — correct/clarify the tier comment, ~5 LOC
- `internal/eval/eval_operations.go` — gate the rendering, ~10 LOC
- `cmd/ailang/main_run_exec.go` — pass the tier, ~3 LOC
- `internal/trace/collector_test.go` — per-tier event matrix, ~150 LOC
- `docs/docs/guides/debugging.md` — tier semantics, ~30 lines

---

## Conflict Surface

Touches `internal/eval/` and `internal/trace/`, so this section is mandatory.

### What this changes

The **content of traces at `standard`**. Nothing about program semantics, evaluation order, or
output. No grammar, AST, type or Core change.

### What else consumes collector events

| Consumer | Depends on function events? | Effect of the change |
|---|---|---|
| `internal/trace/scorer.go:82,231` | **Yes** — scores on `EventFunctionEnter` | Must run at `deep`. Verify its callers set the tier; if any relied on the default, they were silently getting deep-tier data |
| `internal/trace/comparator.go:85,253` | **Yes** — compares function event streams | Same |
| `ailang chains import-motoko` / observatory | Consumes session JSONL, not this collector | Unaffected |
| `--emit-trace jsonl` / `otel` | Exporters over whatever was recorded | Emit less at `standard`. **This is the intended change**, and is the same direction `fb_b0237` asks for |
| `internal/builtins/trace.go:168` | Calls `RecordFunctionEnter` directly | Gated by the collector-level check, which is why the gate belongs there |

**The risk worth naming:** anything that today gets per-call data *by default* stops getting it
unless it asks for `deep`. That is the point of the change, but it must be an explicit
migration, not a surprise — hence the M3 table test and the audit of `scorer`/`comparator`
callers.

### Programs that MUST still work

| Fixture | Assertion |
|---|---|
| `AILANG_TRACE=deep` on the n=200 repro | Still emits `function_enter`/`function_exit` with args — deep is unchanged |
| `internal/trace/scorer_test.go`, `comparator_test.go` | Green; if they construct collectors directly they must opt into `deep` explicitly |
| `ailang trace status` / `trace list` | Unchanged |
| `make verify-examples` | Zero output diffs — this changes instrumentation, never program output |
| The eval harness (`AILANG_EVAL_MAX_RSS`) | Benchmarks should get *cheaper*; no banked-schema change |

### What deliberately changes

`standard` traces become much smaller and no longer contain per-call arguments or results.
Any consumer that was relying on the default to get them must now say `deep`. Anything else
that changes is a regression.

---

## Verification Log

First-party at `dev = 6494a98f4`, v0.35.4, darwin/arm64, Z3 not involved.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | The memory blow-up reproduces | `/usr/bin/time -l` on the reporter's repro at n=100/200/400/800 | **Confirmed.** 101 / 341 / 2059 / 3463 MB |
| V2 | Tracing is the dominant driver, not `concat` | Same program, `AILANG_TRACE=off` | **Confirmed.** 70 / 85 / 106 / 159 MB — near-linear. 19× at n=400 |
| V3 | It is not the *tier's* per-call spans, since `deep` should differ from `standard` | n=400 at `deep` | **Confirmed as the anomaly.** 2373 MB vs 2074 MB — only 14% apart, where the doc comment predicts a large gap |
| V4 | The collector records per-call function events at `standard`, with rendered values | `--emit-trace jsonl` at n=200, inspected | **Confirmed.** 40,609 events / 87 MB; `function_enter` for `std/list.concat` carrying the whole accumulator as a string; largest event 6,133 bytes |
| V5 | `Collector` is **tier-blind** (negative existence — the root cause) | `grep -n "Tier" internal/trace/collector.go` | **Confirmed — zero hits.** `RecordFunctionEnter` (:137) gates only on `Enabled()` |
| V6 | `AILANG_TRACE_MAX_SPANS` does not bound the collector | n=400 with `MAX_SPANS=100` | **Confirmed.** 2217 MB — no reduction |
| V7 | The doc comment claims the opposite of the behavior | Read `options.go:19-22` | **Confirmed.** "NOT per-call function spans" |
| V8 | The tier IS resolved and available at the construction site | Read `main_run_exec.go:463-472` | **Confirmed** — the fix is to pass a value that is already in hand |
| V9 | `scorer.go` and `comparator.go` consume function events (blast radius) | `grep EventFunctionEnter` | **Confirmed** — scorer.go:82,231; comparator.go:85,253 |
| V10 | The exposure claim in `fb_b0237` is **not** re-measured here | — | **Not verified.** This doc cites it as a second report on the same defect and makes no independent claim about exporter transmission |

### Quorum trigger assessment

| # | Trigger | Fires? |
|---|---|---|
| 1 | Design-freeze items | **YES** — one: whether `standard` should record function events at all (below) |
| 2 | Overrides shared machinery | **YES** — `scorer`/`comparator` consume what this stops recording by default |
| 3 | Cost/KPI or banked schema | No — instrumentation content only; no schema or predicate change |
| 4 | External-system premise | No — everything measured in-repo |

Two triggers fire; run `ailang design-quorum` before planning.

---

## High-Impact Decisions

| Decision | Why high impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Should `standard` record per-call function events at all? | Decides whether we fix the code to match the doc, or the doc to match the code. Fixing the doc keeps the OOM and the exposure surface | human | design | high |
| Gate at the collector or at the call sites | Collector is the invariant; call sites are where the rendering cost is paid. Recommendation: both | agent | design | low |
| Do `scorer`/`comparator` callers move to `deep`, or keep a default-deep collector constructor | Silent behavior change for existing consumers vs. an explicit migration | agent | compile | med |

### Design Freeze

- [ ] **Does `standard` record per-call function args/results?** Recommendation: **no** — match
      the documented behavior. It is the default tier, it is on for every `ailang run`, and it
      is currently both an O(n²) memory cost and the widest value-retention surface in the
      runtime. Users who want per-call data have `deep`.

## Deferred Decisions

- Whether `NewCollector` keeps a deep-tier default or every caller migrates — **agent may choose**;
  migrating outright is cleaner, keeping a default is lower-risk for out-of-tree callers.
- The collector cap's default value — **agent may choose**, provided overflow is loud.

## Non-Goals

- **Changing what any exporter transmits.** That is `fb_b0237`'s axis; this doc changes only
  what is recorded. Recording less necessarily transmits less, but the exporter behavior is not
  audited here (V10).
- **Redacting values from `deep`.** Deep exists to capture values; a redaction policy is a
  separate design.
- **Fixing list representation.** See below.

## Relationship to #676 and the parked list doc

[m-list-cons-quadratic](../m-list-cons-quadratic.md) (PARKED, `needs-human-review`) addresses a
**genuinely separate** defect: `::` and `++` copy the whole tail, so list building is Θ(n²) in
time and allocation independent of tracing. Issue
[#676](https://github.com/sunholo-data/ailang/issues/676) reproduced that first-party at
n=6,400.

They are not duplicates and neither supersedes the other:

- **This doc** dominates at the sizes both field reports actually hit — at n=400, tracing is 95%
  of peak RSS (V2). It is the reason the OOXML workload OOM'd.
- **The list doc** dominates at larger n, and is what makes the idiom asymptotically sound.

Fixing tracing does **not** make the list fix unnecessary; it does mean the new report
(`inbox_1788936332313_c08c55b5`) is not a third independent motivation for it, and its
"~5.6 MB per record, suggesting `concat` deep-copies" inference is refuted —
`listConcatImpl` ([`list.go:161`](../../../internal/builtins/list.go#L161)) is a *shallow* copy
of element references into a pre-sized slice.

---

## Testing Strategy

**Per-tier event matrix (the test that would have caught this):** a table asserting, for each
tier, exactly which `EventType`s the collector emits. Prose drifted from code with nothing to
notice; this makes drift a build failure.

**Memory regression:** run the accumulator repro at n=400 under `standard` and assert peak
allocation stays within a stated factor of `off`. Prefer `runtime.MemStats` over shelling to
`/usr/bin/time` — the latter is not portable to the Windows CI job.

**Non-vacuity control:** the same repro at `deep` must still produce per-call events. Without
it, a collector that recorded nothing at all would pass.

**Windows:** no path assertions, no external binaries, no goldens in the new tests.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Instrumentation volume only; no evaluation change |
| A2: Replayability | **+1** | A trace that OOMs the host is not replayable. Bounding the collector (M2) makes capture survivable, and a loud overflow means a truncated trace announces itself instead of quietly lying |
| A3: Effect Legibility | 0 | Effect spans are unchanged at every tier |
| A4: Explicit Authority | **+1** | Retaining every function's arguments and results by default is ambient value capture the user did not ask for. `deep` makes it a choice |
| A5: Bounded Verification | 0 | No verification surface |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | **+1** | A default that turns any accumulator loop into an OOM is a trap a generating model cannot see. Removing it is worth more than the trace data it costs |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | **+1** | Directly: the default instrument's cost was invisible and superlinear. M2's loud cap makes it visible |
| A10: Composability | 0 | None |
| A11: Structured Failure | **+1** | M2 turns silent truncation into a named notice |
| A12: System Boundary | 0 | Recording only; transmission is out of scope (V10) |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 — no nondeterminism introduced
- [x] A3 — no hidden effects
- [x] A4 — **+1**, reduces ambient capture
- [x] A7 — **+1**

## Related Documents

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) — PARKED; separate defect, see above
- [m-take-flatmap-peak-memory](../../implemented/v1_1_0/m-take-flatmap-peak-memory.md) — prior
  art on a memory cap that did not cap ([#617](https://github.com/sunholo-data/ailang/issues/617))
- `docs/docs/guides/debugging.md` — where the tier table must land

---

**Document created**: 2026-09-09
**Last updated**: 2026-09-09

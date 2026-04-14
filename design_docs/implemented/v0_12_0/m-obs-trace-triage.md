---
status: IMPLEMENTED
target: v0.12.0
implemented: 2026-04-14
priority: P1
estimated: 1-2 days (~300 LOC + docs)
dependencies: None
discovered_during: Observatory disk growth investigation (2026-04-14)
---

# Observatory Trace Triage — Make Function-Level Spans Opt-In

## Problem

The observatory DB at `~/.ailang/state/observatory.db` grew to **2,062 MB in 3 days**
of normal development. Retention runs correctly; it deletes 0 rows because
nothing is older than its 7-day TTL. The actual issue is **ingestion rate**:
~84 K spans/day, of which ~88% are `eval.function.*` / `eval.effect.*` spans
emitted by [internal/trace/otel_emitter.go:88](../../../internal/trace/otel_emitter.go#L88)
on every function invocation and effect op during `ailang run`.

### Measured breakdown (snapshot 2026-04-14, 251,932 spans)

| Bucket                    | Count   | Attrs size | Avg attr bytes |
|---------------------------|--------:|-----------:|---------------:|
| `eval.function.<user>`    | 125,521 | **1,045 MB** | 8,726          |
| `eval.function.std/*`     |  85,912 |   **624 MB** | 7,613          |
| `effect.*` (top-level)    |  19,489 |      1.4 MB |     75         |
| `eval.effect.*`           |   9,889 |      2.7 MB |    286         |
| `compile.*`               |   7,068 |      0.3 MB |     45         |
| `other` (exec, executor…) |   4,020 |      0.6 MB |    155         |
| `messages.*`, `ailang.*`  |      33 |       tiny   |    ~100        |

**1.67 GB of the 1.93 GB spans table is the `attributes` JSON of
per-call function spans** — function args and results serialized in full.

### These spans are currently write-only

A grep across the codebase for consumers that query spans by name:

| Consumer query                                                       | Names it reads |
|----------------------------------------------------------------------|----------------|
| `internal/observatory/store_aggregates.go:107-108`                   | `exec.turn`, `exec.tool_use` |
| `internal/observatory/store_aggregates_hierarchy.go:29`              | `coordinator.task.execute`, `claude.execute`, `gemini.execute`, `ailang.exec` |
| `internal/observatory/aggregations.go:118`                           | `%tool.%` (for agent tool-call counts) |
| `internal/observatory/aggregations.go:93-97`                         | aggregates by `task_id` only — span name irrelevant |

**Nothing queries `eval.function.*` or `eval.effect.*` by name.** Confirmed
also by the DB state: of 221,322 `eval.*` spans, **0 have `task_id`, `chain_id`,
`tokens_in/out`, or `cost_usd` populated**. They are not joined to any task or
chain, so `ailang chains`, `ailang eval-chains`, `ailang messages`, the
Observatory hierarchy dashboard, and the eval harness are all unaffected by
their presence or absence.

### Why the spans exist

The original intent (per the emitter code and v0.8 prompts) was dual:

1. **Language development** — profile which AILANG functions are hot, debug
   evaluator/runtime issues, watch effect sequencing.
2. **Potential AI training data** — per-call execution traces with args/results
   as supervision signal for training models to reason about AILANG programs.

Neither use case is currently active. (1) has never pulled a query against
these spans. (2) has no pipeline consuming them; if it existed, it would want a
dedicated structured export, not a best-effort side-channel of a telemetry DB.

### Why they're costly

* **Disk**: 8.7 KB/span × 84 K spans/day = ~730 MB/day of writes. A week of
  dev activity exceeds the laptop's comfort envelope again (cf.
  M-OBS-RETENTION in v0.10.0 where the same DB hit 24 GB).
* **Runtime**: every function enter/exit goes through the OTEL tracer
  (allocate span, serialize attrs, ring-buffer push, DB insert). On a
  recursive benchmark (`fib`, `conway_grid`) this is the dominant cost.
  `AILANG_NO_TRACE=1` already gives ~2× speedup for this reason (see
  [dev-workflow.md](../../../.claude/rules/dev-workflow.md) and
  [v0.10-current.md](../../../changelogs/v0.10-current.md)).
* **Default-on for devs with GCP creds**: [cmd/ailang/main_run.go:514](../../../cmd/ailang/main_run.go#L514)
  sets `emitTrace="auto"` whenever `telemetry.IsEnabled()`, so the spans fire
  for every `ailang run` in normal dev flow — not just during eval / coordinator
  work.

## Goals

1. **Make function-level tracing opt-in.** Default `ailang run` emits
   nothing under `eval.function.*` / `eval.effect.*`. User opts in via an env
   var or CLI flag when they actually want profiling or training data.
2. **Preserve everything consumed today.** `compile.*`, top-level `effect.*`,
   `coordinator.*`, `executor.*` / `claude.execute` / `gemini.execute`,
   `exec.turn`, `exec.tool_use`, `ailang.exec`, `%tool.%`, `messages.*`, and
   all task/chain-linked spans continue to be emitted and stored unchanged.
3. **Preserve the training-data path as a first-class option**, not a default
   firehose. Named flag, named output format, documented retention policy.
4. **Cap the blast radius if someone does opt in.** Per-trace span budget so
   one recursive benchmark doesn't write 6,673 spans × 8.7 KB = 56 MB to the DB.

### Non-goals

* VACUUM-ing the existing 2 GB. That's a one-shot operation the user can run
  manually (`ailang observatory cleanup --vacuum` with `ailang serve` stopped).
* Changing the OTEL pipeline architecture.
* Building the AI training pipeline — only keeping the door open for it.

## Proposal

### 1. Tracing tiers

Introduce three explicit tiers, replacing the current binary on/off:

| Tier            | What's emitted                                               | Default |
|-----------------|--------------------------------------------------------------|---------|
| `off`           | Nothing (`AILANG_NO_TRACE=1` equivalent)                     |         |
| `standard`      | `compile.*`, top-level `effect.*`, `coordinator.*`, `executor.*`, `exec.turn`, `exec.tool_use`, `messages.*`, all task/chain-linked spans | **✅ default when telemetry is enabled** |
| `deep`          | standard + `eval.function.*` + `eval.effect.*` (per-call spans with args/results) | opt-in |

Controls:
* `AILANG_TRACE=off|standard|deep` — env var, wins over defaults
* `--trace standard|deep` — CLI flag on `ailang run` / `ailang eval-compare` / etc.
* `AILANG_NO_TRACE=1` — **kept** for back-compat, equivalent to `AILANG_TRACE=off`

Defaults:
* Interactive `ailang run`: `standard`
* Eval harness, coordinator, server: `standard` (they don't need per-call spans)
* `ailang run --trace deep`: per-call spans for this invocation only
* Automated CI / benchmark runs: `off` (keep existing `AILANG_NO_TRACE=1` in `make eval.mk`)

### 2. Emitter-side filtering

Filter at the earliest point — inside the emitter, before `tracer.Start`:

```go
// internal/trace/otel_emitter.go, ~line 84
case EventFunctionEnter:
    if !opts.DeepTrace { continue }           // NEW
    // ... existing code ...

case EventEffect:
    if !opts.DeepTrace && !isTopLevelEffect(evt) { continue }  // NEW
    // ... existing code ...
```

The `opts` struct is threaded through from `cmd/ailang/main_run.go` where
`--trace` / `AILANG_TRACE` is parsed. `telemetry.TracingOptions{DeepTrace bool}`.

This approach:
* **Zero runtime cost when off** — no span created, no attrs allocated, no DB write
* **Single kill switch** — the collector can still emit a `trace.truncated`
  event if needed for visibility
* **Back-compat friendly** — `AILANG_NO_TRACE=1` still short-circuits all
  tracing as it does today

### 3. Per-trace span budget

Even in `deep` mode, cap per-trace span count at a configurable limit
(`AILANG_TRACE_MAX_SPANS`, default **500**). Once exceeded:

* Stop emitting new spans for that trace
* Emit a single rollup span `trace.truncated` with attributes
  `{dropped_count: N, first_dropped_name: "...", last_kept_time: "..."}`

Implementation lives in `internal/trace/collector.go`, atomic counter per
trace_id, checked before span creation.

### 4. CLI: show trace tier in output

When not in quiet mode, print the active tier once at start:

```
$ ailang run examples/fib.ail
Trace: standard (set AILANG_TRACE=deep for per-call spans)
```

Mirrors the existing `Trace collection enabled (%s)` line at
[main_run.go:523](../../../cmd/ailang/main_run.go#L523).

### 5. Docs

Update:
* `docs/docs/guides/telemetry.md` — document the three tiers, when to use `deep`
* `docs/docs/guides/debugging.md` — cross-reference from `AILANG_NO_TRACE` section
* `.claude/rules/dev-workflow.md` — update the debug-flag table
* `CHANGELOG.md` — note behavior change (default went from implicit-deep to `standard`)

## Migration / behavior change

This is a **user-visible default change**: devs who today get per-call spans
implicitly will stop getting them. Mitigations:

* Tier printed at startup (when not `-quiet`)
* `--trace deep` flag documented in `ailang run --help`
* Release notes call out the change and the one-line opt-in
* Existing eval/benchmark workflows are unaffected (they already set
  `AILANG_NO_TRACE=1` or don't run with telemetry enabled)

The **Observatory disk usage** should drop to <100 MB/day after this lands
(roughly what `standard` contributed historically).

## Open questions

1. **AI training pipeline** — if/when we build one, should `deep` write to a
   separate NDJSON file / table instead of the main spans table? This design
   leaves that open; the training pipeline can opt in and route spans
   wherever it wants. Out of scope for this milestone but worth a follow-up
   sprint (M-TRAIN-TRACES?).
2. **`std/*` stdlib calls** — should these be in `deep` or a fourth tier
   `deep-with-stdlib`? Proposal: keep simple, put them in `deep`. Filtering
   stdlib-only is cheap if it becomes noisy later.
3. **Should `eval-compare` / the eval harness default to `deep`?** Per-call
   spans could be valuable for comparing model-generated AILANG programs.
   Proposal: **no** — default `standard`, add `ailang eval-compare --trace deep`
   for researchers who want it.

## Acceptance criteria

- [ ] `AILANG_TRACE=off|standard|deep` env var recognized; `--trace` CLI flag
      on `ailang run` works
- [ ] Default tier is `standard`; `eval.function.*` / `eval.effect.*` not
      emitted unless `deep`
- [ ] Per-trace span budget of 500 enforced; `trace.truncated` summary emitted
      when hit
- [ ] `AILANG_NO_TRACE=1` continues to work (maps to `off`)
- [ ] `ailang chains`, `ailang eval-chains`, `ailang messages`, coordinator
      daemon, Observatory hierarchy dashboard: verified unchanged on a
      real dev session
- [ ] Docs updated (telemetry.md, debugging.md, dev-workflow.md, CHANGELOG.md)
- [ ] Regression test: run a benchmark with `AILANG_TRACE=standard`, assert
      no `eval.function.*` spans in DB; run same benchmark with
      `AILANG_TRACE=deep`, assert they appear
- [ ] After 1 dev day on `standard`: span count ≤ 10K, DB size increase ≤ 100 MB

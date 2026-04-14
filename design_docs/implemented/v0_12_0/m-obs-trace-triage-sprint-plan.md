---
status: IMPLEMENTED
sprint_id: M-OBS-TRACE-TRIAGE
target: v0.12.0
implemented: 2026-04-14
estimated: 2 days (~340 LOC + docs)
design_doc: design_docs/implemented/v0_12_0/m-obs-trace-triage.md
dependencies: None
risk_level: low
---

# Sprint Plan — M-OBS-TRACE-TRIAGE

## Summary

Make AILANG's function-level tracing opt-in. Today every `ailang run` with
GCP creds emits one OTEL span per function call (avg 8.7 KB attrs each),
filling the observatory DB at ~730 MB/day with data no consumer queries.
Introduce three tiers (`off` / `standard` / `deep`) with emitter-side
filtering and a per-trace span budget; keep the training-data path as a
deliberate opt-in.

**Goal in one line**: Default `ailang run` stops writing `eval.function.*` and
`eval.effect.*` spans, while every existing consumer (`ailang chains`,
`ailang messages`, coordinator, Observatory dashboard, eval harness) keeps
working unchanged.

## Status Analysis

### What's done
- [x] Investigation complete — 252K spans / 1.93 GB traced to `eval.function.*` emission
- [x] Consumer audit complete — grep confirms no reader queries these span names
- [x] Observatory DB cleared (1.9 GB → 4.3 MB) for a clean measurement baseline
- [x] Design doc approved — `design_docs/planned/v0_11_4/m-obs-trace-triage.md`

### Recent velocity (reference)
| Milestone | Duration | LOC |
|---|---:|---:|
| M-PERF5 M1 (eval Clone→NewChildEnvironment) | 1 day | ~100 |
| M-POLY-ORD M1–M3 | 2 days | ~250 |
| M-V0_11_3-HOTFIX M1 (short-circuit) | 1 day | ~180 |

This sprint slots between those. Low risk — purely additive controls on an
existing emitter, no semantic changes to traces that stay emitted.

### Open questions from design doc (resolved for this sprint)
1. AI training pipeline → **deferred** to follow-up (`M-TRAIN-TRACES`), out of scope.
2. `std/*` placement → **`deep`** (no separate tier), per design proposal.
3. `eval-compare` default tier → **`standard`** (per design proposal); researchers opt into `deep`.

## Milestones

### M1 — Tracing tiers + emitter filter (Day 1 AM) — ~140 LOC

Core change: introduce `telemetry.TracingOptions`, thread it from CLI /
env var parsing into the emitter, and short-circuit function/effect spans
when not in `deep` mode.

**Files:**
- `internal/telemetry/options.go` (NEW, ~40 LOC) — `TracingOptions` struct, `ParseTier(string)`, `TierFromEnv()`
- `internal/trace/otel_emitter.go` (~30 LOC) — accept options, gate `EventFunctionEnter` and non-top-level `EventEffect`
- `internal/trace/collector.go` (~20 LOC) — pass options through to emitter
- `cmd/ailang/main_run.go` (~30 LOC) — parse `--trace` flag + `AILANG_TRACE` env, map `AILANG_NO_TRACE=1` → `off`
- `cmd/ailang/help.go` + `cmd/ailang/eval_benchmark.go` (~20 LOC) — expose `--trace` on relevant subcommands

**Approach:**
1. Define tiers as constants: `TierOff`, `TierStandard`, `TierDeep`
2. Single precedence order: CLI `--trace` > `AILANG_TRACE` env > `AILANG_NO_TRACE=1` → off > default `standard`
3. Emitter guard is the cheapest possible check — `if opts.Tier != TierDeep { continue }` before any span allocation
4. "Top-level effect" = effect whose parent in the event stack is a module/run root, not another function call. Cheap heuristic: `evt.Depth <= 1`.

**Acceptance criteria:**
- [ ] `AILANG_TRACE=off|standard|deep` env var recognized
- [ ] `ailang run --trace deep` flag recognized; overrides env
- [ ] `AILANG_NO_TRACE=1` still works (maps to `off`)
- [ ] Unit test: emitter in `standard` tier skips `EventFunctionEnter`, keeps `EventModuleStart` / `EventEffect` (top-level) / everything else
- [ ] Unit test: emitter in `deep` tier emits everything
- [ ] Unit test: emitter in `off` tier emits nothing

**Risks:**
- "Top-level effect" heuristic may need tuning. If `evt.Depth` isn't available, fall back to tracking a bool in the emitter's stack.

### M2 — Per-trace span budget (Day 1 PM) — ~80 LOC

Cap total spans per trace at `AILANG_TRACE_MAX_SPANS` (default 500). When
exceeded, stop emitting and close with a single `trace.truncated` summary.

**Files:**
- `internal/trace/collector.go` (~60 LOC) — `sync/atomic` counter keyed by `trace_id`, check before `tracer.Start`, emit rollup on first overflow
- `internal/trace/collector_test.go` (~20 LOC) — test for budget enforcement + rollup span

**Approach:**
1. `map[string]*atomic.Int64` keyed by trace_id, guarded by `sync.RWMutex`
2. Increment+check before each `tracer.Start`. If over budget and rollup not yet emitted, emit `trace.truncated` and set a sentinel so subsequent spans skip silently.
3. Cleanup: remove counter entry when trace's root span ends (via `EventModuleEnd` on outermost frame).

**Acceptance criteria:**
- [ ] Budget default 500; configurable via `AILANG_TRACE_MAX_SPANS`
- [ ] When exceeded, exactly one `trace.truncated` span emitted with `{dropped_count, first_dropped_name}` attrs
- [ ] Unit test: fake event stream > budget produces exactly `budget + 1` spans (N kept + 1 rollup)
- [ ] Counter map is bounded (cleanup on trace end verified via test)

**Risks:**
- Counter-map memory leak if traces never close cleanly. Mitigation: document the behavior, rely on process-lifetime bound (coordinator / eval harness are short-lived).

### M3 — CLI banner + regression tests (Day 2 AM) — ~120 LOC

Surface the active tier to users and lock in behavior with real-world
regression tests that run a benchmark and assert on DB state.

**Files:**
- `cmd/ailang/main_run.go` (~10 LOC) — print tier banner when not `-quiet`
- `internal/trace/otel_emitter_integration_test.go` (NEW, ~90 LOC) — run `examples/recursion_fibonacci.ail` via evaluator under each tier, assert span counts in in-memory store
- `cmd/ailang/main_run_test.go` (~20 LOC) — CLI flag parsing tests

**Acceptance criteria:**
- [ ] `ailang run` prints `Trace: standard (set AILANG_TRACE=deep for per-call spans)` when tracing is enabled + not `-quiet`
- [ ] Regression test: running `fib` benchmark with `standard` → 0 `eval.function.*` spans in store
- [ ] Regression test: running same benchmark with `deep` → >0 `eval.function.*` spans
- [ ] Regression test: budget enforcement triggers on a recursive program with `AILANG_TRACE_MAX_SPANS=10`

**Risks:** None material — tests run against the in-memory collector, not the real SQLite DB.

### M4 — Docs + consumer smoke test (Day 2 PM) — docs only, ~0 LOC code

Ship the behavior change safely. Manually verify no consumer regressed
against a real dev session, then document.

**Files:**
- `docs/docs/guides/telemetry.md` — add "Tracing tiers" section
- `docs/docs/guides/debugging.md` — cross-reference `AILANG_NO_TRACE` → tiers
- `.claude/rules/dev-workflow.md` — update debug-flag table
- `CHANGELOG.md` — note default behavior change, migration path

**Consumer smoke test (manual checklist, run before commit):**
1. [ ] `make quick-install` with new default (`standard`)
2. [ ] Run `ailang run examples/adt_simple.ail` → completes normally
3. [ ] `ailang chains list` → still shows existing chains (3 rows preserved from earlier cleanup)
4. [ ] `ailang messages list --unread` → works
5. [ ] Start coordinator daemon briefly → span emission for `coordinator.task.execute` confirmed
6. [ ] Run one eval benchmark (`make eval-suite MODELS=claude-haiku-4-5 BENCHMARKS=fizzbuzz`) → succeeds
7. [ ] Check DB: `sqlite3 ~/.ailang/state/observatory.db "SELECT name, COUNT(*) FROM spans GROUP BY name"` — no `eval.function.*`, yes `compile.*` / `effect.*` / `coordinator.*`
8. [ ] Run same benchmark with `AILANG_TRACE=deep` → `eval.function.*` spans reappear

**Acceptance criteria:**
- [ ] All 8 smoke-test items checked
- [ ] `CHANGELOG.md` entry under `[Unreleased]` with migration note
- [ ] `docs/docs/guides/telemetry.md` has a tier table identical in shape to the one in the design doc
- [ ] `.claude/rules/dev-workflow.md` debug-flag table updated (new `AILANG_TRACE`, `AILANG_TRACE_MAX_SPANS` rows; `AILANG_NO_TRACE` row flagged as back-compat alias)

## Success Metrics

| Metric | Before | Target |
|---|---:|---:|
| DB growth / dev day (standard) | ~730 MB | **≤ 100 MB** |
| `ailang run` runtime overhead (non-recursive) | baseline | unchanged (±2%) |
| `ailang run` runtime overhead (recursive fib) | baseline | **1.5–2× faster** (no per-call spans) |
| `eval.function.*` spans in DB after 1 day | 125K+ | **0** |
| Existing consumers passing smoke test | N/A | **8/8** |

## Example / regression artifacts

- `internal/trace/otel_emitter_integration_test.go` — uses `examples/recursion_fibonacci.ail` as the "hot recursion" fixture. No new example file required; the existing one exercises the behavior.
- Add comment to `examples/recursion_fibonacci.ail` referencing this regression test? **No** — the example is for users, not implementation details.

## Dependencies

None. The eval harness and benchmark suite already set `AILANG_NO_TRACE=1`
(per `make/eval.mk`), so they're unaffected during the transition.

## Rollback Plan

If something regresses badly after merge:
- Set `AILANG_TRACE=deep` in any shell — restores pre-sprint behavior fully
- Revert `cmd/ailang/main_run.go` change to default to `deep` instead of `standard` — single-line change

## Open Questions for User

1. **Target version**: v0.11.4 (just released) or v0.11.5 (next)? Recommend **v0.11.5** — v0.11.4 is already tagged.
2. **Ship tier banner?** Users running `ailang run` in tight CI loops might dislike the extra line. Recommend **yes but respect `-quiet`**, which it will.
3. **Budget default of 500 OK?** The worst current trace had 6,673 spans. 500 catches the pathological cases while still letting medium programs trace cleanly in `deep` mode.

---

## Files to be touched (summary)

**New:**
- `internal/telemetry/options.go`
- `internal/trace/otel_emitter_integration_test.go`

**Modified:**
- `internal/trace/otel_emitter.go`
- `internal/trace/collector.go`
- `internal/trace/collector_test.go`
- `cmd/ailang/main_run.go`
- `cmd/ailang/main_run_test.go`
- `cmd/ailang/help.go`
- `cmd/ailang/eval_benchmark.go`
- `docs/docs/guides/telemetry.md`
- `docs/docs/guides/debugging.md`
- `.claude/rules/dev-workflow.md`
- `CHANGELOG.md`

**Total LOC:** ~340 implementation + ~80 tests = **~420 LOC**

# Sprint Plan: M-LAT-BUDGET — Latency Budgets & Dividend Ledger (Phase 0)

## Summary

Build the measurement foundation for AILANG latency budgets: 6 self-contained canonical workload `.ail` files, a bash benchmark harness, a committed baseline JSON, and a hand-edited dividend ledger seeded by a live `AILANG_NO_TRACE=1` A/B measurement on the data-intensive workload. **No template edits, no CI gate, no `ailang bench` CLI** — those are Phase 1–5, deferred to separate design docs.

**Scope change from design doc:** `docparse_small/large` workloads dropped in favour of `list_small/large` — docparse lives in the external `sunholo/ailang_parse` package, which would force the benchmark suite to depend on it. Self-contained list-processing workloads still exercise the evaluator hot path that the tracing regression bug lived in. The M-PERF6B dividend is seeded by a live A/B measurement (`AILANG_NO_TRACE=0` vs `AILANG_NO_TRACE=1`) on `list_large`, not back-filled from Moby Dick history.

**Duration:** 1 day (~7 hours)
**Dependencies:** None — reuses `AILANG_NO_TRACE=1` (shipped in v0.11.0) and existing EPUB fixtures
**Risk Level:** Low — pure measurement infrastructure, no runtime or type-system changes, fully reversible
**Design Doc:** [design_docs/planned/v0_11_2/m-latency-budget.md](m-latency-budget.md)

## Current Status Analysis

### Completed Recently (last 14 days)
- ✅ **M-WASM-TRACE** (`8438b9c5`): WASM trace streaming, std/trace module, OTEL span IDs — ~900 LOC in ~2 days
- ✅ **M-PERF6B** (`710d3f88`, `17a11807`, `59f7f72c`, `d0e029be`): trace overhead fix + memprofile flag — ~600 LOC in ~1.5 days. Produced the `AILANG_NO_TRACE=1` flag and the 3s Moby Dick saving that seeds our ledger.
- ✅ **M-INCREMENTAL-TYPECHECK** (`cacecf61`, `4f91d27e`, `d96de92f`, `7c4ff925`): CachedModule serialization + cache hit logic — ~870 LOC in ~2 days
- ✅ **M-PERF-DOCPARSE** (`e5e5b02f`): profiling data for DocParse hot path

### Velocity
- Recent average: **~400–500 LOC/day** on infrastructure sprints (measured from M-PERF6B, M-INCREMENTAL-TYPECHECK)
- M-LAT-BUDGET estimated total: **~540 LOC** (mostly data files, ~100 LOC bash, ~30 LOC Makefile/CHANGELOG)
- **Estimated capacity: 1 day** with comfortable buffer — well within recent velocity

### Remaining from Design Doc
- ⏳ **Phase 0.1** — Canonical workload files (6 × `.ail`, ~115 LOC + README)
- ⏳ **Phase 0.2** — `tools/bench_workloads.sh` + Makefile target (~105 LOC)
- ⏳ **Phase 0.3** — Baseline capture + ledger seed (~230 LOC data files, hand-edited ledger)
- ⏳ **Phase 0.4** — CHANGELOG + debugging guide updates (~25 LOC)

## Critical Discovery: Directory Layout

`benchmarks/` **already exists** and contains:
- `benchmarks/*.yml` — 52 AI eval benchmark definitions (fizzbuzz, merge_sort, etc.)
- `benchmarks/runtime/` — 7 microbenchmark `.ail` files (fib30, list_map_filter, closure_curried, etc.) for Go `testing.B` harness
- `benchmarks/cross-language/` — cross-language comparisons
- `benchmarks/VISION_BENCHMARKS.md`, `benchmarks/README.md`

**Our new subdirectory `benchmarks/workloads/` is distinct** from all three:
- Not AI-eval YAMLs (those measure AI quality)
- Not microbenchmarks (`benchmarks/runtime/` has fib30-sized programs for Go `testing.B`)
- **It's end-to-end p95 latency workloads for the dividend ledger** — data-intensive programs like Moby Dick EPUB parse

The sprint-executor must add a clarifying paragraph to `benchmarks/README.md` explaining the three sibling concepts: eval benchmarks (AI quality), runtime microbenchmarks (Go `testing.B`), workload latency benchmarks (p95 regression + dividend ledger).

## Proposed Milestones

### Milestone 1: Canonical Workload Files
**Goal:** Author 6 self-contained `.ail` workloads covering the hot-path surface, each runnable via `ailang run FILE` in <10s.
**Estimated:** ~115 LOC (`.ail`) + ~50 LOC (`README.md`) = **~165 LOC**
**Duration:** ~2 hours

**Tasks:**
- Create `benchmarks/workloads/` directory
- Author `cold_hello.ail` (~10 LOC) — minimal startup latency floor (one `println`)
- Author `warm_eval.ail` (~15 LOC) — pure recursive arithmetic (fib 28), exercises evaluator hot loop
- Author `typecheck_heavy.ail` (~50 LOC) — multiple ADTs + constructors + exhaustive pattern matches, exercises typechecker
- Author `list_small.ail` (~20 LOC) — build 1,000-element list, map + filter + fold pipeline
- Author `list_large.ail` (~20 LOC) — build 10,000-element list, map + filter + fold pipeline (data-intensive regression canary — this is where the tracing bug would have shown up)
- Author `effect_roundtrip.ail` (~20 LOC) — 50 `std/io` println round-trips inside an effect handler
- Author `benchmarks/workloads/README.md` (~60 LOC) — what each workload exercises + why docparse is not included (external package)
- Add a short paragraph to existing `benchmarks/README.md` clarifying the three-sibling structure (eval / runtime / workloads)
- Verify each workload compiles and runs: `ailang run benchmarks/workloads/<name>.ail`

**Acceptance Criteria:**
- [ ] 6 `.ail` files exist, each compiles cleanly
- [ ] Each workload runs in <10s on current hardware (no infinite loops, no unbounded recursion)
- [ ] Each workload uses `module benchmarks/workloads/NAME` and `export func main()` per [.claude/rules/coding-standards.md](../../.claude/rules/coding-standards.md)
- [ ] Each workload is **self-contained** — no external package dependencies (stdlib only)
- [ ] README explains what each workload exercises (parser / typechecker / evaluator / IO effects)
- [ ] README documents why docparse workloads are absent (lives in external `sunholo/ailang_parse` package)
- [ ] `benchmarks/README.md` updated with the three-sibling clarification
- [ ] `make verify-examples` still passes (workloads are *not* examples — they don't need to be in `examples/`)

**Risks:**
- **`list_large` runtime too long** — 10k elements in an interpreter might exceed 10s. Mitigation: tune n down to 5k if needed; document the chosen size in README.
- **typecheck_heavy workload not exercising what we think** — easy to write a program that type-checks quickly but runs slowly. Mitigation: manually run with `--debug-compile` and confirm elaborate/type-check phases dominate wall clock.
- **list_large not catching tracing-style regressions** — the original bug was in `internal/trace/` slowing every evaluator step. Mitigation: `list_large` does ~30k evaluator steps (1 build + 1 filter + 1 map + 1 fold over 10k), which is enough per-step density to expose any hot-loop regression.

### Milestone 2: Benchmark Harness Script
**Goal:** `tools/bench_workloads.sh` that runs the suite, captures p50/p95, and emits valid JSON with hardware fingerprint.
**Estimated:** ~100 LOC bash + ~5 LOC Makefile = **~105 LOC**
**Duration:** ~3 hours

**Tasks:**
- Create `tools/bench_workloads.sh` with these CLI flags:
  - `--runs N` (default 5)
  - `--output FILE` (default `benchmarks/latency_budgets.json`)
  - `--workload NAME` (default: all)
  - `--verbose` (prints each run's timing to stderr)
- Reuse the iteration pattern from [.claude/skills/perf-reviewer/scripts/benchmark.sh](../../.claude/skills/perf-reviewer/scripts/benchmark.sh) — 5 runs, discard warmup, compute median/p95 via `sort -n | awk`
- Wrap each invocation with `AILANG_NO_TRACE=1` and wall-clock timing (`date +%s%3N` before/after, or `/usr/bin/time -p` parsed)
- Capture hardware fingerprint: `uname -a`, CPU brand (macOS: `sysctl -n machdep.cpu.brand_string`, Linux: grep from `/proc/cpuinfo`), `go version`, `git rev-parse HEAD`
- Emit JSON to `--output` path. Hand-rolled JSON assembly (no `jq` dependency) — use `printf` with proper escaping
- Print a human-readable summary table to stderr showing workload / p50 / p95 / target / status
- Add `bench-workloads` target to [Makefile](../../Makefile) that invokes the script
- **Fail-loud behavior** (per [CLAUDE.md](../../CLAUDE.md) "No Silent Fallbacks"): if a workload fails to run, the script reports *which* workload, *which* run, and the exit code — it does NOT silently skip and emit partial JSON
- Run the harness 3× on the author's machine, verify variance ≤±3% on p95

**Acceptance Criteria:**
- [ ] `tools/bench_workloads.sh` exists, is executable (`chmod +x`)
- [ ] `make bench-workloads` runs to completion, prints table, exits 0
- [ ] Output JSON validates as JSON (`jq . benchmarks/latency_budgets.json`)
- [ ] Output JSON contains: version, commit, captured_at, hardware, workloads (with runs_ms, p50_ms, p95_ms, target_p95_ms, last_updated for each)
- [ ] Three consecutive runs show p95 variance ≤±3% on warm hardware
- [ ] Script fails loudly on broken workload (test: temporarily break one workload with a syntax error, confirm error message identifies it)
- [ ] Script runs in <5 minutes end-to-end
- [ ] No new dependencies introduced (no `jq`, no Python) — pure bash + coreutils

**Risks:**
- **Wall-clock noise on shared machine** — other processes perturb timing. Mitigation: run with suggestion to quit other apps; document noise floor; treat p95 (not p50) as the stable signal.
- **`date +%s%3N` not portable** — millisecond precision varies macOS vs Linux. Mitigation: detect OS and use `gdate` or `python3 -c 'import time; print(int(time.time()*1000))'` fallback if needed.
- **JSON escaping bugs** — hand-rolled JSON is bug-prone for paths with special chars. Mitigation: validate every run with `jq .`; use `printf '%s'` not echo; escape backslashes in paths.

### Milestone 3: Baseline Capture + Dividend Ledger Seed
**Goal:** Commit the initial `latency_budgets.json` (v0.11.1 baseline) and hand-author `budget_ledger.md` seeded by a live `AILANG_NO_TRACE=1` A/B measurement on `list_large`.
**Estimated:** ~150 LOC JSON (auto-generated) + ~80 LOC markdown (hand-edited) = **~230 LOC data**
**Duration:** ~1 hour

**Tasks:**
- Run `make bench-workloads` on a quiet machine (minimize background noise) to capture the `AILANG_NO_TRACE=1` baseline (the "after" numbers)
- Run the harness once more with `AILANG_NO_TRACE=0` on just `list_large.ail` and `list_small.ail` to capture "before" numbers. This gives the real, reproducible M-PERF6B dividend on the actual canonical workloads.
- Commit resulting `benchmarks/latency_budgets.json` as v0.11.1 baseline (uses the `AILANG_NO_TRACE=1` numbers)
- Hand-author `benchmarks/budget_ledger.md`:
  - Header explaining the 50/50 dividend rule
  - Hot-path package list: `internal/{parser,lexer,elaborate,types,eval*,trace,pipeline}`, `stdlib/{io,encoding}`
  - Historical note referencing M-PERF6B's effect on docparse (external) as the motivating incident, but ledger entries are against self-contained workloads
  - Ledger table with columns: Date, Sprint, Workload, Delta, User share, Pool delta, Pool balance, Commit
  - **Seed entry** — live A/B measurement on `list_large` (trace-on → trace-off), commit `08ef7bbc`:
    - `list_large`: `NO_TRACE=0` p95 X ms → `NO_TRACE=1` p95 Y ms, saved `-(X-Y)` ms, user share `-(X-Y)/2` ms (tightens target), pool `+(X-Y)/2` ms
  - Optionally a second row for `list_small` if the A/B delta is ≥50ms (worth recording)
  - Running pool balances: per-workload
- Verify math: user share + pool delta = total saving per workload. Check all targets sit above today's actual p95 (headroom).
- Update `latency_budgets.json`: set `list_large.target_p95_ms` from ledger-derived target (pre-optimization p95 minus user share)

**Acceptance Criteria:**
- [ ] `benchmarks/latency_budgets.json` committed, validates as JSON
- [ ] Contains 6 workloads × 5 runs each with all schema fields populated
- [ ] `benchmarks/budget_ledger.md` committed with header, rule explanation, hot-path list, and seed entry
- [ ] Math closes: user + pool = total saving; running pool balance is consistent
- [ ] `list_large.target_p95_ms` in JSON matches ledger-derived target (pre-M-PERF6B p95 − user share)
- [ ] Sanity check: no target is below actual measured p95 (would mean the feature already fails its own budget)
- [ ] Historical docparse numbers from v0.11.0 changelog cited in ledger header as motivating context

**Risks:**
- **A/B measurement unstable** — trace overhead may vary run-to-run. Mitigation: 5 runs each, report p95 (not mean); accept noisy number as seed.
- **First-run p95 different from expected** — real numbers may reveal already-failing workloads. Mitigation: set targets generously (1.5× actual p95 for v0 baselines); don't optimize — document and move on, since Phase 0 is measurement, not enforcement.

### Milestone 4: Documentation + Changelog
**Goal:** Announce the feature in CHANGELOG and point perf-debuggers at the new harness.
**Estimated:** ~25 LOC across 2 files
**Duration:** ~1 hour

**Tasks:**
- Add entry to [CHANGELOG.md](../../CHANGELOG.md) under the next unreleased version (v0.11.2):
  - Summary: "M-LAT-BUDGET Phase 0: canonical workload baselines + dividend ledger"
  - Link to design doc and sprint plan
  - Note that Phases 1-5 (template / auditor / CLI / CI gates) are deferred
- Add a paragraph to [docs/docs/guides/debugging.md](../../docs/docs/guides/debugging.md):
  - "Check for latency regression: run `make bench-workloads`, compare to `benchmarks/latency_budgets.json`"
  - Point at `benchmarks/budget_ledger.md` for historical context
- Verify no updates to the design doc template — explicitly forbidden by Design Freeze
- Run `make verify-examples` and `make test` to confirm no regressions

**Acceptance Criteria:**
- [ ] CHANGELOG entry added under v0.11.2
- [ ] debugging.md mentions `make bench-workloads`
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] `make lint` clean
- [ ] `.claude/skills/design-doc-creator/scripts/create_planned_doc.sh` is **unchanged** (enforce Design Freeze)
- [ ] `.claude/skills/design-doc-creator/resources/design_doc_structure.md` is **unchanged**

**Risks:**
- **Scope creep temptation** — "while we're here, let's add the template section." Mitigation: the Design Freeze item `[x] Scope limited to Phase 0 (baselines + ledger only)` is explicit. If sprint-executor notices template edits sneaking in, it must pause and refuse.

## Success Metrics

- **Workloads functional:** 6/6 canonical `.ail` files compile and run in <10s each ✅
- **Harness stable:** p95 variance ≤±3% across 3 consecutive runs on same hardware ✅
- **Baseline committed:** `benchmarks/latency_budgets.json` present with all 6 workloads ✅
- **Ledger closes:** math in `benchmarks/budget_ledger.md` balances; pool balance traceable ✅
- **Zero regressions:** `make test`, `make ci`, `make verify-examples` unaffected ✅
- **Design Freeze honored:** design-doc-creator skill files untouched ✅
- **Test coverage:** N/A — Phase 0 adds data files + bash script, no Go code
- **Documentation:** CHANGELOG entry + debugging guide paragraph ✅

## Dependencies

- **`AILANG_NO_TRACE=1`** — shipped in v0.11.0 (commit `08ef7bbc`). Required for stable baselines. ✓ Already present.
- **Moby Dick and Alice EPUB fixtures** — either commit them or reference the `ailang-parse` project's test-files path (documented in `memory/reference_benchmark_files.md`).
- **Stable hardware for baseline capture** — run on the author's Apple M-series machine with other apps quit. Fingerprint recorded so future captures on the same hardware are comparable.

## Open Questions

1. ~~**EPUB fixture location?**~~ **RESOLVED**: Commit Alice EPUB (185KB) under `benchmarks/workloads/fixtures/gutenberg_alice.epub`; reference Moby Dick via `/Users/mark/dev/sunholo/ailang-parse/data/test_files/gutenberg_moby_dick.epub` with a skip-if-missing fallback so CI and other developers can still run the other 5 workloads.
2. **Should Phase 0 include a stub `Make bench-workloads` output in `post-release` skill?** The post-release skill already captures eval baselines — it's tempting to wire workload baselines into the same flow. **Recommendation:** No. That's Phase 3/4 territory; Phase 0 is hand-run by the author to prove the measurement is stable. Automated baseline capture requires trust that only comes from repeated successful runs.
3. ~~**M-PERF6B "before" number**~~ **RESOLVED**: from `changelogs/v0.10-current.md` v0.11.0 entry — Moby Dick **7.27s → 3.0s** (M-PERF6B alone), Alice **2.97s → 1.4s**. Both are exact; no estimation needed.

## Notes

- **Single-session sprint** — 4 milestones, ~7 hours total. Should complete in one focused working day. No multi-session continuity needed, but the JSON progress file still lets us resume if interrupted.
- **Phase 0 is intentionally boring** — just baselines and a ledger. The interesting cultural work (template enforcement, CI gates, CLI) all live in future phases.
- **Naming discipline:** do not conflate `benchmarks/workloads/` with `benchmarks/runtime/` (Go microbenchmarks) or `benchmarks/*.yml` (AI evals). The README clarification in Milestone 1 is non-optional.
- **50/50 dividend math** is applied per optimization, not per release (user confirmed in plan).
- **Systemic analysis** ([design-doc-creator SKILL.md](../../.claude/skills/design-doc-creator/SKILL.md#audit-for-systemic-issues)) checked: this is not a bug fix, it's a process/infrastructure addition. No similar code paths to audit.
- **Git user:** verify `gh auth status` shows `sunholo-voight-kampff` before any commits (per [CLAUDE.md](../../CLAUDE.md) critical principle 0.1).
- **sprint-executor must pause** if it notices itself editing `design-doc-creator` skill files — that's Phase 1 territory, not Phase 0.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_11_2/m-latency-budget-sprint-plan.md`

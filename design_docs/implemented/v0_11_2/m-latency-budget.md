# M-LAT-BUDGET: Latency Budgets & Dividend Ledger

**Status**: Implemented (2026-04-12)
**Target**: v0.11.2
**Priority**: P1 (Medium)
**Estimated**: 1 day (Phase 0 only — subsequent phases deferred to separate design docs)
**Dependencies**: None (reuses existing `AILANG_NO_TRACE=1` flag and `internal/eval_harness/metrics.go`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a process/governance feature — it adds a new measurement surface and a design-doc section, but changes no language semantics.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics change |
| A2: Replayability | 0 | No impact on trace replay |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | 0 | No capability surface change |
| A5: Bounded Verification | 0 | No type-system change |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Makes runtime cost **machine-readable** via JSON ledger, enabling agent-level cost reasoning |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | **+2** | Directly extends cost visibility from tokens/memory to end-to-end latency; adds canonical workloads with explicit p95 targets |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | 0 | No error-handling change |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +3** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis (JSON ledger consumable by agents)

## Problem Statement

**AILANG has been ambushed twice by silent latency regressions on data-intensive workloads** — most vividly, the trace-auto-enable bug (`08ef7bbc`, Apr 2026) that ~2× slowed DocParse until M-PERF6B shipped `AILANG_NO_TRACE=1`. M-PERF6B alone pulled Moby Dick EPUB from **7.27s → 3.0s (2.4×)** and Alice EPUB from **2.97s → 1.4s (2.1×)**; combined with prior sprints, total improvement from v0.10.14 was Moby **35.1s → 3.0s (11.7×)**. But those wins happened reactively, after user-facing embarrassment, not because we had a contract saying "this workload must stay under X seconds."

**Current State:**
- No canonical workload baseline checked into the repo. Eval metrics ([internal/eval_harness/metrics.go](internal/eval_harness/metrics.go)) capture `DurationMs`/`CompileMs`/`ExecuteMs` per eval run, but these are scattered across `eval_results/` and not aggregated by workload type.
- Benchmark infrastructure exists in three disconnected places:
  - [.claude/skills/perf-reviewer/scripts/benchmark.sh](.claude/skills/perf-reviewer/scripts/benchmark.sh) — cross-language comparisons (Fibonacci, sum)
  - [internal/pipeline/validate_coretypeinfo_bench_test.go](internal/pipeline/validate_coretypeinfo_bench_test.go) — Go `testing.B` microbenchmarks
  - [tools/eval_baseline.sh](tools/eval_baseline.sh) — versioned eval results storage
- No design doc template section for latency impact. Features are accepted or rejected on functional grounds; runtime cost is discussed ad-hoc if at all.
- No accounting when optimizations free time. M-PERF6B saved 4.27s on Moby Dick (7.27s → 3.0s); that entire saving was implicitly given back to users, but there's no record of how much could have funded future features.
- No CI regression gate. A PR that doubles parser latency passes CI as long as tests pass.

**Impact:**
- Users (developers running AILANG on real documents) see surprise slowdowns release-to-release.
- Feature authors have no way to reason about the runtime cost of their additions at design time.
- Optimization work loses its leverage — savings are absorbed instead of traded deliberately.
- Prior art (M-DX25 scoped effect budgets in [v0_7_1/m-dx25-budget-report.md](../implemented/v0_7_1/m-dx25-budget-report.md)) already established "budget as contract" as a pattern for capability costs. This doc extends the same idea to wall-clock latency.

## Goals

**Primary Goal:** Establish a canonical latency baseline for AILANG's hot-path workloads and a per-optimization "dividend ledger" so every runtime-cost trade-off is visible and accountable at design-doc time.

**Success Metrics:**
- `benchmarks/latency_budgets.json` exists with ≥6 canonical workloads, each with p50/p95/commit/hardware fields captured on stable hardware.
- `benchmarks/budget_ledger.md` exists with the seed entry retroactively recording M-PERF6B's 3s saving on Moby Dick.
- `tools/bench_workloads.sh` runs end-to-end in <5min, is idempotent (re-runs produce ±3% variance on warm hardware), and is callable from `make`.
- The design doc template documents the latency budget contract for future features to adopt — but template edits ship in a **separate** follow-up sprint (Phase 1, not in this doc's scope).
- Zero impact on `make test`, `make ci`, or release workflow (all new targets are opt-in).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Scope limited to Phase 0 (baselines only) | Prevents scope creep — template/auditor/CLI phases each warrant their own design doc once the measurement is trusted | human | design | low |
| Canonical workloads live in `benchmarks/workloads/` (new dir) | Clean separation from `examples/` (teaching) and AI eval benchmarks (quality, not speed) — prevents changes to examples from perturbing baselines | human | design | med |
| 50/50 dividend split applied **per optimization**, not per release | Immediate, visible accounting for each commit; avoids batching disputes | human | design | med |
| Hot-path package list is explicit and versioned in the doc | Agent auditors need a crisp rule for "does this design doc need a latency budget?" | human | design | med |
| Baselines captured with `AILANG_NO_TRACE=1` | Tracing adds ~2× overhead on data-intensive paths; measuring *with* trace on would anchor targets to a deprecated failure mode | human | design | high |
| Hardware fingerprint recorded but not normalized across machines | Normalization is expensive and fragile; comparing same-machine deltas is enough for regression detection | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Scope limited to Phase 0 (baselines + ledger only)
- [x] Workloads live in `benchmarks/workloads/`
- [x] 50/50 split is per-optimization
- [x] Hot-path list = `internal/{parser,lexer,elaborate,types,eval*,trace,pipeline}`, `stdlib/{io,encoding,docparse}`
- [x] Baselines run with `AILANG_NO_TRACE=1`

## Solution Design

### Overview

Build the foundation of a latency budget system: a small benchmark harness, a canonical workload suite, and a committed baseline JSON + markdown ledger. This is the **measurement substrate** — no template edits, no CI gate, no CLI command. Subsequent phases (template integration, auditor enforcement, `ailang bench` subcommand, soft/hard CI gates) are explicitly deferred to separate design docs so each can be evaluated on its own merits once the baseline is proven stable.

### Architecture

The system is intentionally small — four artifacts and one shell script.

**Components:**

1. **Canonical workload files** (`benchmarks/workloads/*.ail`): 6 representative AILANG programs covering the hot-path surface. Each workload is a self-contained `.ail` file with a `module` declaration and `export func main()` entry point, following [.claude/rules/coding-standards.md](../../.claude/rules/coding-standards.md) test file conventions.

2. **Benchmark harness** (`tools/bench_workloads.sh`): Bash script that iterates each workload, runs `ailang run` 5× with `AILANG_NO_TRACE=1`, captures wall-clock time via `/usr/bin/time` or `time` shell builtin, computes p50/p95, and emits `benchmarks/latency_budgets.json`. Uses the same "5 iterations, take median" pattern as [.claude/skills/perf-reviewer/scripts/benchmark.sh](.claude/skills/perf-reviewer/scripts/benchmark.sh). Captures hardware fingerprint (`uname -a`, `sysctl machdep.cpu.brand_string` on macOS or `/proc/cpuinfo` on Linux, `go version`, `git rev-parse HEAD`).

3. **Baseline JSON** (`benchmarks/latency_budgets.json`): Structured output consumed by future phases. Schema:
   ```json
   {
     "version": "v0.11.1",
     "commit": "162f3dc4",
     "captured_at": "2026-04-11T12:00:00Z",
     "hardware": {
       "os": "darwin",
       "arch": "arm64",
       "cpu": "Apple M-series",
       "go_version": "go1.22.0"
     },
     "workloads": {
       "cold-hello": {
         "input": "benchmarks/workloads/cold_hello.ail",
         "runs_ms": [142, 139, 145, 141, 143],
         "p50_ms": 142,
         "p95_ms": 145,
         "target_p95_ms": 250,
         "last_updated": "2026-04-11"
       },
       "...": {}
     }
   }
   ```

4. **Dividend ledger** (`benchmarks/budget_ledger.md`): Hand-edited markdown table recording every commit that adds or removes latency on a canonical workload. Authoritative for the *target p95* in `latency_budgets.json` — when a new entry lands, the target gets recalculated.

5. **Makefile target** (`make bench-workloads`): Wrapper that invokes `tools/bench_workloads.sh`, prints the result table to stdout, and exits 0 regardless (no CI gate yet).

### Implementation Plan

**Phase 0.1: Workload selection and authoring** (~2 hours)
- [ ] Create `benchmarks/workloads/` directory
- [ ] Author 6 canonical `.ail` programs:
  - [ ] `cold_hello.ail` — minimal `println("Hello")` program (startup latency floor)
  - [ ] `warm_eval.ail` — 100-iteration arithmetic loop (interpreter hot path)
  - [ ] `typecheck_heavy.ail` — imports ≥5 stdlib modules, defines ≥20 typed functions (elaborator/types load)
  - [ ] `docparse_small.ail` — parses Alice EPUB (~160KB) from `../ailang-parse/test-files/`, counts blocks
  - [ ] `docparse_large.ail` — parses Moby Dick EPUB (~1.2MB), counts blocks
  - [ ] `effect_roundtrip.ail` — 50 IO round-trips via `std/io` (effect dispatch cost)
- [ ] Each workload must compile cleanly and run in <10s on current hardware
- [ ] Add a `benchmarks/workloads/README.md` explaining what each workload exercises

**Phase 0.2: Benchmark harness** (~3 hours)
- [ ] Write `tools/bench_workloads.sh` (~100 LOC bash):
  - [ ] Parse CLI flags: `--runs N` (default 5), `--output FILE` (default `benchmarks/latency_budgets.json`), `--workload NAME` (default all)
  - [ ] For each workload: run `AILANG_NO_TRACE=1 ailang run FILE` N times, capture wall-clock ms
  - [ ] Compute p50/p95 via `sort -n | awk` (no jq dependency)
  - [ ] Capture hardware fingerprint
  - [ ] Emit JSON (hand-rolled, no dependencies beyond `date`, `git`, `uname`)
  - [ ] Print human-readable table to stderr
- [ ] Add `make bench-workloads` target to [Makefile](../../Makefile)
- [ ] Run 3× on current hardware, verify variance is ≤±3% (noise sanity check)

**Phase 0.3: Baseline capture and ledger seed** (~1 hour)
- [ ] Run `make bench-workloads` to produce `benchmarks/latency_budgets.json`
- [ ] Commit the JSON as the v0.11.1 starting baseline
- [ ] Author `benchmarks/budget_ledger.md` with:
  - [ ] Header explaining the 50/50 rule and hot-path package list
  - [ ] Seed entry retroactively recording M-PERF6B (commit `08ef7bbc`):
    - `docparse-large` (Moby Dick): 7.27s → 3.0s, saved **4.27s**, user share **-2.135s** (target 7.27s → **5.135s**), pool **+2.135s**
    - `docparse-small` (Alice): 2.97s → 1.4s, saved **1.57s**, user share **-0.785s** (target 2.97s → **2.185s**), pool **+0.785s**
  - [ ] Current dividend pool balance: **+2.92s** across docparse workloads
- [ ] Verify math: user share + pool delta = total saving per workload; running pool balance consistent.

**Phase 0.4: Documentation** (~1 hour)
- [ ] Add a short section to [CHANGELOG.md](../../CHANGELOG.md) under the next unreleased version
- [ ] Add a paragraph to [docs/docs/guides/debugging.md](../../docs/docs/guides/debugging.md) pointing to `make bench-workloads` for perf regression investigation
- [ ] **Do NOT** update the design doc template yet — that's Phase 1 and needs its own design doc

### Files to Modify/Create

**New files:**
- `benchmarks/workloads/cold_hello.ail` (~5 LOC)
- `benchmarks/workloads/warm_eval.ail` (~15 LOC)
- `benchmarks/workloads/typecheck_heavy.ail` (~40 LOC)
- `benchmarks/workloads/docparse_small.ail` (~20 LOC)
- `benchmarks/workloads/docparse_large.ail` (~20 LOC)
- `benchmarks/workloads/effect_roundtrip.ail` (~15 LOC)
- `benchmarks/workloads/README.md` (~50 LOC)
- `tools/bench_workloads.sh` (~100 LOC bash)
- `benchmarks/latency_budgets.json` (~150 LOC JSON, auto-generated)
- `benchmarks/budget_ledger.md` (~80 LOC markdown, hand-edited)

**Modified files:**
- `Makefile` (+5 LOC — add `bench-workloads` target)
- `CHANGELOG.md` (+10 LOC — announce feature)
- `docs/docs/guides/debugging.md` (+15 LOC — mention the new target)

**Total: ~540 LOC new, ~30 LOC modified**

## Examples

### Example 1: Recording an optimization dividend

Suppose a new sprint lands that shaves 400ms off `typecheck_heavy` via constraint solver caching.

**Before (no ledger):**
The commit lands, tests pass, the 400ms saving vanishes into user benefit. Next sprint adds a 300ms feature to the elaborator. Users see `typecheck_heavy` get 100ms faster overall but the accounting is invisible, and a third sprint adds another 300ms believing there's no budget pressure. Performance drifts.

**After (with ledger):**
```markdown
## Dividend Ledger

| Date | Sprint | Workload | Delta | User share | Pool delta | Pool balance | Commit |
|---|---|---|---|---|---|---|---|
| 2026-04-11 | M-PERF6B (seed) | docparse-large | -4.27s | -2.135s (target 7.27s→5.135s) | +2.135s | 2.135s | 08ef7bbc |
| 2026-04-11 | M-PERF6B (seed) | docparse-small | -1.57s | -0.785s (target 2.97s→2.185s) | +0.785s | 0.785s | 08ef7bbc |
| 2026-05-03 | M-TYPE-CACHE | typecheck-heavy | -400ms | -200ms (target 500ms→300ms) | +200ms | 200ms | abc1234 |
| 2026-05-20 | M-ELAB-VALIDATE | typecheck-heavy | +300ms | drawn from pool | -300ms | -100ms (would need offset!) | def5678 |
```

Note that pools are **per-workload** — M-PERF6B built up a docparse pool but did nothing for typecheck-heavy. When M-ELAB-VALIDATE tries to spend 300ms on typecheck-heavy, the typecheck pool only has 200ms available, so the feature must either ship a 100ms offsetting optimization, gate behind a flag, or claim an axiom justification. The ledger makes this trade-off visible at design-doc time rather than after a user complains.

### Example 2: Baseline JSON consumed by a future `ailang bench diff`

**Before:**
```bash
$ ailang run benchmarks/workloads/docparse_large.ail
(12.3 seconds later...)
# Is this slow? Compared to what? Nobody knows.
```

**After:**
```bash
$ make bench-workloads
Running 6 workloads × 5 runs each (AILANG_NO_TRACE=1)...

Workload               p50      p95    target   status
cold-hello             142ms    145ms  250ms    ✓
warm-eval               48ms     52ms  100ms    ✓
typecheck-heavy        287ms    312ms  500ms    ✓
docparse-small        1420ms   1480ms 2000ms    ✓
docparse-large        2980ms   3150ms 4500ms    ✓
effect-roundtrip       195ms    210ms  300ms    ✓

Baseline written to benchmarks/latency_budgets.json
```

Now a future `ailang bench diff` (Phase 3, separate doc) can load the JSON, re-run the suite, and flag any p95 that regresses >5%.

## Success Criteria

- [ ] `benchmarks/workloads/` exists with 6 `.ail` files and a README
- [ ] Each workload runs to completion in <10s on current hardware (stable M-series machine)
- [ ] `tools/bench_workloads.sh` exists, is executable, and produces valid JSON on first run
- [ ] `make bench-workloads` runs to completion, prints table, exits 0
- [ ] `benchmarks/latency_budgets.json` committed with 6 workloads × 5 runs each
- [ ] Three consecutive runs of `make bench-workloads` on the same hardware show p95 variance ≤±3%
- [ ] `benchmarks/budget_ledger.md` committed with header, seed entry, and math that closes
- [ ] `make test` and `make ci` unaffected (no new failures, no slowdowns)
- [ ] Zero changes to the design doc template (`design-doc-creator` skill) — that's Phase 1
- [ ] CHANGELOG entry announces the feature
- [ ] `docs/docs/guides/debugging.md` mentions the new target

## Testing Strategy

**Unit tests:**
- None. The harness is a bash script and the artifacts are data files. A Go unit test would add maintenance burden without catching anything a manual run wouldn't.

**Integration tests:**
- Manual: run `make bench-workloads` three times; verify variance.
- Manual: delete `benchmarks/latency_budgets.json`, re-run, verify idempotent regeneration.
- Manual: run with a workload that's intentionally broken (syntax error); verify the script reports which workload failed and which runs succeeded.

**Manual testing:**
- Run on the author's machine (Apple M-series) and a second machine if available (Linux x86_64) — record both fingerprints in git history, confirm deltas are same-machine-only.
- Retroactively apply the M-PERF6B ledger math: the 3s Moby Dick saving should have tightened the target by 1.5s. Walk through the calculation on paper and confirm the ledger entry matches.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **JSON schema exact field names** — agent may choose (e.g., `runs_ms` vs `samples_ms`), as long as a future `ailang bench diff` reader can be written against it without ambiguity.
- **Which Moby Dick EPUB file to point at** — agent may choose between the file in `~/.claude/projects/-Users-mark-dev-sunholo-ailang/memory/reference_benchmark_files.md` and a freshly-downloaded copy committed under `benchmarks/fixtures/`. Prefer committed for reproducibility.
- **Hardware fingerprint format** — agent may choose (JSON object vs flat strings), as long as it's greppable.
- **Whether to delete or `.gitignore` stale baseline files during development** — agent may choose; just make sure the committed baseline is reproducible.

## Non-Goals

**Explicitly out of scope for this sprint:**
- **Design doc template updates** — deferred to **Phase 1** (separate design doc: `m-latency-budget-template.md`). Editing the `design-doc-creator` skill is a governance change that deserves its own review.
- **Design-spec-auditor enforcement** — deferred to **Phase 2** (separate design doc). Auditor logic needs its own testing surface.
- **`ailang bench` CLI subcommand** — deferred to **Phase 3** (separate design doc). CLI surface is a user contract.
- **CI soft/hard gate** — deferred to **Phase 4** (separate design doc, ≥2 releases away). Needs real noise-floor data first.
- **Cross-hardware baseline normalization** — explicitly rejected. Compare same-machine deltas only; record fingerprints for context.
- **Optimizing anything.** This is pure measurement infrastructure. Any regressions the baseline reveals go to separate sprints.
- **Replacing the eval suite.** Eval measures AI quality; latency budgets measure runtime cost. Complementary, not competing.
- **A dashboard.** JSON + markdown ledger is the full UI for Phase 0.

## Timeline

**Single session** (~1 day, 7 hours including buffer):
- Phase 0.1: Workload selection and authoring — 2h
- Phase 0.2: Benchmark harness — 3h
- Phase 0.3: Baseline capture and ledger seed — 1h
- Phase 0.4: Documentation — 1h

**Total: ~7 hours, deliverable in a single focused sprint.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| EPUB test files not reproducibly available (vary between machines) | High | Commit fixture files under `benchmarks/fixtures/` or reference the `ailang-parse` project paths with a fallback that skips the workload if files are missing |
| Wall-clock variance exceeds ±3% on the author's machine | Medium | Increase default runs from 5 to 10; document the noise floor in the JSON metadata; only trust p95 (not p50) as the stable signal |
| Tracing accidentally re-enabled in a future commit, breaking baselines | High | Harness explicitly sets `AILANG_NO_TRACE=1` and verifies via a trace-count probe after each run (if feasible without building a runtime dependency) |
| 6 workloads is too few to catch real regressions | Medium | Start with 6, expand in Phase 2 or later based on observed blind spots. Better to have 6 maintained workloads than 20 that rot |
| Ledger markdown becomes unmaintained after initial enthusiasm | Medium | Phase 2 (deferred) auditor requires design doc authors to add entries; until then, the `post-release` skill can nudge adding one entry per release |
| Scope creep — author edits the design doc template "while we're here" | High | Design Freeze item explicitly forbids template edits. Template work is a separate design doc with its own review |

## Related Documents

<!-- Auto-populated by Ollama neural search on "latency budget" -->

**Implemented (prior art for "budget as contract" pattern):**
- [design_docs/implemented/v0_7_1/m-dx25-budget-report.md](../implemented/v0_7_1/m-dx25-budget-report.md) — M-DX25 scoped effect budgets (capability costs, not latency, but the same governance pattern)
- [design_docs/implemented/v0_9_8/m-cost2-dashboard-firestore-optimization-sprint-plan.md](../implemented/v0_9_8/m-cost2-dashboard-firestore-optimization-sprint-plan.md) — cost dashboard work

**Implemented (perf sprints that produced current state):**
- `design_docs/implemented/v0_9_2/m-perf5-data-intensive-workloads.md` — M-PERF5 DocParse phase-timing work
- `changelogs/v0.10-current.md` v0.11.0 — M-PERF6B trace-auto-enable fix (commit `08ef7bbc`)

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-perf-goroutine-id.md](../planned/v0_11_0/m-perf-goroutine-id.md) — runtime perf (complementary, not overlapping)

## References

- [Design Axioms](/docs/references/axioms) — A9 "Cost Visibility" is the anchor
- [CLAUDE.md — No Silent Fallbacks Principle](../../CLAUDE.md) — applies to perf measurement: a baseline that can't be captured should fail loudly, not silently skip
- [AILANG_NO_TRACE=1 flag](../../cmd/ailang/main_run.go) — commit `08ef7bbc`
- [tools/eval_baseline.sh](../../tools/eval_baseline.sh) — baseline storage pattern to mirror
- Approved plan: `/Users/mark/.claude/plans/sleepy-orbiting-sky.md`

## Future Work

Phases deliberately deferred to separate design docs (in dependency order):

1. **Phase 1 — Template integration** (`m-latency-budget-template.md`): Add "## Latency Budget" section to [.claude/skills/design-doc-creator/scripts/create_planned_doc.sh](../../.claude/skills/design-doc-creator/scripts/create_planned_doc.sh) heredoc and [.claude/skills/design-doc-creator/resources/design_doc_structure.md](../../.claude/skills/design-doc-creator/resources/design_doc_structure.md) guidance.
2. **Phase 2 — Auditor enforcement** (`m-latency-budget-auditor.md`): Extend `design-spec-auditor` to require the section for hot-path design docs.
3. **Phase 3 — `ailang bench` CLI** (`m-latency-budget-cli.md`): User-facing `ailang bench run`, `ailang bench diff`, `ailang bench baseline` subcommands.
4. **Phase 4 — Soft CI gate** (`m-latency-budget-ci-soft.md`): `make bench-check` emits warnings on PRs that regress >5%.
5. **Phase 5 — Hard CI gate** (`m-latency-budget-ci-hard.md`, ≥2 releases after Phase 4): Fails CI on >10% regression. Only after noise floor is proven stable.

Each phase is an independent sprint with its own design doc, review, and success criteria. The critical discipline is **not** shipping them as a bundle — each phase must prove useful before the next is approved.

---

**Document created**: 2026-04-11
**Last updated**: 2026-04-12 (moved to implemented)

---

## Implementation Report (2026-04-12)

**Shipped in v0.11.2.** Phase 0 of the design landed in one sprint with a
mid-sprint scope pivot and one unrelated bug discovered along the way.

### What was built

**M1 — Self-contained workloads (NOT DocParse).** The original design picked
DocParse-on-EPUB as the data-intensive probe because that was the workload
that motivated the whole sprint. We pivoted mid-sprint to **six self-contained
`.ail` programs** in `benchmarks/workloads/` after the user pointed out that
pulling the external `sunholo/ailang_parse` package into the benchmark loop
made the suite slow to set up and impossible to run in CI without a checkout
step. Workloads:

| Workload | Hot path | Why it's in the suite |
|----------|----------|----------------------|
| `cold_hello.ail` | parser → elaborator → typecheck → eval init → IO | cold-start floor |
| `warm_eval.ail` | pure recursive int arithmetic (`fib(24)`) | warm evaluator hot loop |
| `typecheck_heavy.ail` | ADTs, exhaustive matching, polymorphism | type checker stress |
| `effect_roundtrip.ail` | 50× IO effect dispatch | capability + handler dispatch |
| `list_small.ail` | std/list pipeline, 500 ints | constant-cost reference |
| `list_large.ail` | std/list pipeline, 5,000 ints | **regression canary** for tracing-style bugs |

`list_small`/`list_large` are pure (`-> int`, no IO main) because wrapping
`buildList` inside an effectful main pushed past AILANG's default 10k
recursion-frame limit. The 5k size on `list_large` is the largest the
tail-recursive builder could handle within that limit.

**M2 — Harness + Make targets.** [`tools/bench_workloads.sh`](../../tools/bench_workloads.sh)
runs each workload N times with `AILANG_NO_TRACE=1`, computes p50/p95/min/max
via nearest-rank percentile, and writes
[`benchmarks/latency_budgets.json`](../../benchmarks/latency_budgets.json).
Discards the first run as warmup when N≥3. Always runs from the project root
with relative paths (so canonical module IDs match — discovered by an early
MOD010 failure from passing absolute paths). New Make targets:
`make bench-workloads` (5 runs, full capture), `make bench-workloads-quick`
(3 runs, dry-run JSON dump). ~190 LOC bash.

**M3 — Baseline + ledger.** Captured the v0.11.1 baseline on Apple M2 and
hand-authored [`benchmarks/budget_ledger.md`](../../benchmarks/budget_ledger.md).
The ledger seeds with a back-fill entry for M-PERF6B: a live A/B re-measurement
on `list_large` showed trace-on at 4506ms p95 vs trace-off at 733ms p95 — a
saving of 3773ms, of which 1887ms went to the user (the workload now ships
under 1 second) and 1887ms credits the dev pool for future hot-path features.

**M4 — Docs.** CHANGELOG entry, `benchmarks/README.md` updated with the
three-suite layout (`*.yml` AI evals / `runtime/*.ail` micro / `workloads/*.ail`
latency probes), and a "Latency Budget Workloads" section added to the
debugging guide.

### Unplanned: cache poisoning bug

Caught while iterating on the workloads. Editing `warm_eval.ail` from
`fib(28) → fib(24)` kept printing `fib(28) = 317811`. Root cause:
`pipeline_module.go` was passing `mod.Path` (canonical module ID) into
`os.ReadFile` when computing the cache key, so `sourceContent` defaulted to
`""` for every module. Latent in M-PERF6 M3 (commit `0515adda`, 2026-03-16)
because the cache only counted hits then; activated by M-INCREMENTAL-TYPECHECK
M3 (commit `4f91d27e`, 2026-04-10) when cache hits started skipping
compilation. **No release affected** — v0.11.1 was tagged before the
cache-skip wiring landed; the bug lived for ~2 days on `dev`. One-line fix
plus an end-to-end regression test
([`internal/pipeline/cache_invalidation_test.go`](../../internal/pipeline/cache_invalidation_test.go))
that exercises the seam the unit tests missed.

The shape of this bug — a silent fallback at the integration point between
two suites of correct unit tests — is being tracked separately as a
follow-up audit task (cache correctness across `internal/`, plus a lint
rule against silent error-swallowing in cache-affecting code paths).

### Code locations

**New files**
- `benchmarks/workloads/cold_hello.ail` (5 LOC)
- `benchmarks/workloads/warm_eval.ail` (15 LOC)
- `benchmarks/workloads/typecheck_heavy.ail` (~80 LOC)
- `benchmarks/workloads/effect_roundtrip.ail` (15 LOC)
- `benchmarks/workloads/list_small.ail` (20 LOC)
- `benchmarks/workloads/list_large.ail` (25 LOC)
- `benchmarks/workloads/README.md` (~110 LOC)
- `benchmarks/budget_ledger.md` (~140 LOC)
- `benchmarks/latency_budgets.json` (auto-generated)
- `tools/bench_workloads.sh` (~210 LOC)
- `internal/pipeline/cache_invalidation_test.go` (~115 LOC) — bug-discovery side effect

**Modified files**
- `internal/pipeline/pipeline_module.go` (+8 / -3) — cache poisoning fix
- `make/eval.mk` (+9) — `bench-workloads` and `bench-workloads-quick` targets
- `changelogs/v0.10-current.md` (+45) — v0.11.2 entry
- `benchmarks/README.md` (+30 / -10) — three-suite layout
- `docs/docs/guides/debugging.md` (+30) — Latency Budget Workloads section

### Phases NOT shipped (still future work)

The original design split this into 5 phases. **Only Phase 0 shipped in
v0.11.2.** Phases 1–5 remain in the "Future Work" section of the design
above and are explicitly each their own sprint:

1. Template integration — add `## Latency Budget` section to design-doc-creator
2. Auditor enforcement — `design-spec-auditor` requires the section for hot-path docs
3. `ailang bench` CLI surface
4. Soft CI gate (warning only)
5. Hard CI gate (fails CI on >10% regression)

This was deliberate. The point of v0.11.2 was to capture honest numbers and
seed the ledger before designing the enforcement around them. Phases 1–5
will each get their own design doc once we've used the system for a release
or two and know what actually needs enforcing.

### Verification

- All 6 workloads compile and run, producing deterministic output
- `make bench-workloads` writes a valid `latency_budgets.json` with all 6 entries
- `make bench-workloads-quick` round-trips through dry-run JSON emission
- `tools/bench_workloads.sh --workload list_large --runs 10 --verbose` works
- Cache regression test passes after fix, fails before fix (verified via stash)
- All `internal/pipeline/...` tests still pass

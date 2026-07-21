# M-MEM-BUDGET-RUNTIME: Memory as a Budgeted Resource in the AILANG Runtime

**Status**: Planned (DOC-READY — mission queue)
**Target**: v0.31.0
**Priority**: P1 (host-safety for AI-generated code; direct incident driver)
**Estimated**: 2-3 days (Phase 1)
**Dependencies**: None (complements, does not depend on, the harness RSS watchdog task)
**Lane**: Extension (PROGRAM.md routing — runtime capability, zero core-syntax change)

## Motivating Incident (2026-07-20)

The Mac Studio eval rig kernel-panicked (`watchdog timeout: no checkins from watchdogd in
94 seconds`; VM compressor at 100% of segments limit with 100 swapfiles). Jetsam recorded
three model-generated **Python** processes at ~80-120GB each and an **ailang** process at
~7.7GB. Artifacts: `/Library/Logs/DiagnosticReports/panic-full-2026-07-20-160307.0002.panic`,
`JetsamEvent-2026-07-20-155120.ips`.

The per-language autopsy is the thesis of this doc:

| Lane | Why it did / didn't take the host down |
|---|---|
| Python | No default heap cap; `while True: append` allocates at C speed → 100GB in minutes |
| JavaScript | V8's ~4GB default old-space cap made any bomb self-limiting |
| Go | Compile gate filters most junk before execution |
| AILANG | *Incidentally* protected: no while/mutation (canonical bomb inexpressible), interpreter speed (1-2 orders slower burn) — but **no memory bound exists**; the 7.7GB ailang process proves unbounded growth is reachable via recursion+accumulation |

AILANG's protection today is incidental. This doc makes it **guaranteed**: a memory bomb in
AILANG-generated code should produce a typed, catchable `MEM001` error — never host death.
That is the A9 (Cost Visibility) axiom applied to RAM, and it is a differentiator worth
stating publicly: *a language for AI code synthesis must assume the generating model will
occasionally write a memory bomb.*

The harness-side RSS watchdog (spawned task, separate session) protects **our eval rig**
for all four languages. This doc protects **every host that runs AILANG** — the two are
complementary, not alternatives.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Physical backstop triggers at a host-dependent point — same class as the accepted `--timeout` flag (host boundary, see A12). Semantic evaluation is unchanged; Phase 2's logical meter would make exhaustion fully deterministic |
| A2: Replayability | 0 | No trace-format change in Phase 1; exhaustion is recorded as a structured runtime error in the trace like any other |
| A3: Effect Legibility | 0 | Deliberately NOT an effect: allocation stays pure (see Conflict Surface §5 — making `Mem` an effect row entry would force every pure function to carry it, an A3/A8 regression) |
| A4: Explicit Authority | +1 | The bound is explicit, caller-set authority over a real resource (`--max-mem`), consistent with the capability model's spirit |
| A5: Bounded Verification | 0 | No change to type checking or Z3 |
| A6: Safe Concurrency | 0 | Monitor goroutine is internal; no user-visible concurrency change |
| A7: Machines First | +1 | AI-executed code gets a machine-decidable, coded failure (`MEM001`) instead of an OS kill; agents can catch, categorize, and retry smaller |
| A8: Minimal Syntax | +1 | Zero new syntax. CLI flag + env var only |
| A9: Cost Visibility | +1 | Memory becomes a visible, bounded, reportable resource like effect budgets and cost budgets |
| A10: Composability | 0 | Orthogonal to existing budgets (`@limit` effect rows untouched) |
| A11: Structured Failure | +1 | OOM-by-host (unstructured, unrecordable) becomes a typed error with code, limit, and usage |
| A12: System Boundary | 0 | The physical limit is explicitly documented as a host-boundary control (like `--timeout`), not a semantic |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism in language semantics; boundary control documented as such
- [x] A3 (Effects): No hidden side effects; allocation deliberately kept pure
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Built precisely FOR machine consumers

## Problem Statement

AILANG-generated programs can allocate without bound (recursion + accumulator). The runtime
has **no memory accounting or limit of any kind** (verified below). On a host without external
supervision, a generated memory bomb degrades or kills the machine — the exact failure that
kernel-panicked the eval rig on 2026-07-20 (via the Python lane, with the AILANG lane at
7.7GB on the same slope).

**Current State (all rows verified — see Verification Log):**
- `internal/effects/budget.go` provides per-effect budgets (`! {IO @limit=5}` →
  `effect 'IO' budget exhausted: semantic limit=5, used=5` + hint) — memory is not covered
- The only `ReadMemStats` usage in the repo is compile-phase *metrics* (`internal/pipeline/metrics.go`) — measurement, zero enforcement
- No `debug.SetMemoryLimit` call exists anywhere in `internal/` or `cmd/`
- The error-code registry (58 records in `dist/error_codes.json`) has no MEM/budget code; `MEM001` is unallocated

**Impact:**
- Any host executing untrusted/AI-generated AILANG (eval rigs, the WASM-less server paths,
  future Ailang World) inherits an unbounded-memory liability
- The eval harness cannot distinguish "model wrote a memory bomb" from other failures —
  no `resource_limit`-style signal exists at the language level

## Goals

**Primary Goal:** A runaway AILANG program under a configured memory budget terminates with a
typed `MEM001` runtime error — the host stays healthy and the failure is machine-readable.

**Success Metrics:**
- A deliberate allocator (`letrec`-style list accumulator) under `--max-mem 512MiB` exits
  non-zero with `MEM001` naming limit and usage, in bounded time; host RSS never exceeds
  limit + slop (slop documented, target ≤ 25%)
- All existing examples (`make verify-examples`) pass unchanged with no flag set —
  byte-identical behavior when the feature is off
- Eval harness banks `MEM001` failures under a distinct error category, visible in
  `eval-report`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Memory bound is a **process/runtime control, NOT an effect row** | Making `Mem` an effect would force it onto every pure function (allocation is pure) — an A3/A8 regression rippling through the type system | human (this doc proposes; ratify at quorum) | design | high |
| Enforcement = Go soft limit (`debug.SetMemoryLimit`) + monitor goroutine + cooperative check in the eval loop | macOS does not reliably enforce `RLIMIT_AS`; GC-integrated soft limit + cooperative cancellation is the portable, typed-error-capable mechanism | human (mechanism), agent (internals) | design | med |
| Default is **OFF** (no limit) for `ailang run`; **ON** for eval-harness execution (harness passes an explicit limit) | Changes nothing for humans; protects the AI-execution path where bombs originate | human | design | low |
| Error code `MEM001`, message shape mirrors effect-budget errors (`memory budget exhausted: limit=X, used=Y` + hint) | Coded errors are the machine contract; consistency with `budget.go` phrasing | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Ratify: memory bound as runtime control, not effect (quorum / Mark)
- [ ] Ratify: default-off for CLI, explicit-on for harness

## Solution Design

### Overview

Phase 1 adds a **physical memory budget** to the AILANG runtime process: a `--max-mem`
CLI flag (and `AILANG_MAX_MEM` env var for harness use) that (a) sets Go's soft memory
limit so GC fights back first, (b) starts a monitor that watches actual usage, and (c) on
breach, trips the evaluator's cooperative cancellation so evaluation unwinds with a typed
`MEM001` error — the same interruption pattern as a timeout, with a different coded reason.

Phase 2 (Future Work, separate doc): a **deterministic logical allocation meter** in the
VM (counting cons cells / string bytes / record fields), which would make exhaustion
replayable at an exact operation index and traceable. Phase 1 deliberately ships the
host-safety floor first.

### Architecture

**Components:**
1. **Limit plumbing** (`cmd/ailang/`): `--max-mem <size>` flag (accepts `512MiB`/`4GiB`),
   `AILANG_MAX_MEM` env; 0/absent = today's behavior, no monitor started
2. **Monitor** (`internal/runtime/memguard.go`, new): on start, `debug.SetMemoryLimit(limit)`;
   goroutine polls `runtime.ReadMemStats` (HeapAlloc + stacks) every ~250ms; on breach sets
   an atomic trip flag / cancels the run context
3. **Cooperative check** (`internal/eval/` or `internal/vm/`): evaluation loop observes the
   trip signal at back-edges (function application / builtin dispatch) and returns the typed
   error. Implementer should first audit how `--timeout` interrupts evaluation and reuse
   that exact pathway if one exists (see Deferred Decisions)
4. **Error** (`internal/errors` + `dist/error_codes.json` regen): `MEM001` — "memory budget
   exhausted: limit=<X>, used=<Y>" + hint ("raise --max-mem or reduce allocation; large
   accumulations may need streaming/fold")
5. **Harness integration** (`internal/eval_harness/runner.go`): pass `AILANG_MAX_MEM` (default
   from a new spec/env knob) to `ailang` benchmark children; categorize `MEM001` stderr as a
   distinct error category so eval reports show "resource_limit" separately from logic errors

### Implementation Plan

**Phase 1a: memguard + flag** (~1 day)
- [ ] `internal/runtime/memguard.go` with start/stop, soft limit, poll loop, trip signal
- [ ] `--max-mem` / `AILANG_MAX_MEM` plumbing; 0 = fully inert (no goroutine)
- [ ] Unit tests: trip fires under a synthetic allocator; never fires under limit

**Phase 1b: typed unwind** (~1 day)
- [ ] Audit `--timeout`'s interruption path; wire trip → cooperative check → `MEM001`
- [ ] Register `MEM001` (verify still unallocated at implementation time), regen `dist/error_codes.json`
- [ ] Runnable example `examples/expected_fail/mem_budget.ail` (bounded allocator + doc'd flag)

**Phase 1c: harness + docs** (~0.5-1 day)
- [ ] Harness passes limit to ailang children; `MEM001` → distinct error category in reports
- [ ] Docs: debugging guide flag table, LIMITATIONS note (physical, not semantic, bound)
- [ ] CHANGELOG entry

### Files to Modify/Create

**New files:**
- `internal/runtime/memguard.go` (~150 LOC) + test (~150 LOC)
- `examples/expected_fail/mem_budget.ail` (~30 LOC)

**Modified files:**
- `cmd/ailang/` run entry (~40 LOC) — flag/env plumbing
- `internal/eval/` or `internal/vm/` (~30 LOC) — cooperative check at back-edges (site depends on the timeout-path audit)
- `internal/eval_harness/runner.go` + `error_categorizer.go` (~40 LOC)
- `dist/error_codes.json` (regenerated)

## Conflict Surface

*(Required: touches `internal/eval`/`internal/vm`/`internal/runtime` and `cmd/ailang`.)*

1. **Positions extended**: (a) CLI flag namespace of `ailang run`; (b) the evaluator's
   interruption pathway; (c) the runtime error taxonomy. **No lexical/syntactic position is
   touched** — no parser, lexer, AST, or type-system change.
2. **Existing occupants of those positions**:
   - `--timeout` already occupies "externally-triggered evaluation abort". `MEM001` must ride
     the same unwind so two abort mechanisms can't race incoherently; first-trip-wins.
   - Effect budgets (`! {IO @limit=N}`, `internal/effects/budget.go`) occupy "budget
     exhausted" messaging. `MEM001` mirrors the message *shape* but is process-scoped, not
     effect-row-scoped; the doc's message says "memory budget", never "effect budget".
   - `internal/pipeline/metrics.go` reads MemStats for compile metrics — read-only overlap,
     no interaction; memguard must not perturb its numbers beyond GC-timing noise.
3. **Disambiguation**: none needed at syntax level (no syntax). At runtime, trip reasons are
   distinct typed errors (timeout vs MEM001 vs effect-budget), each with its own code path.
4. **Programs that MUST still work unchanged** (regression fixtures, all verified to exist):
   - `examples/expected_fail/effect_budgets.ail` (45 lines) — effect budgets unaffected
   - `examples/runnable/split_map_join.ail` — ordinary allocation under no limit
   - Whole-corpus gate: `make verify-examples` with no `--max-mem` set must be byte-identical
5. **Deliberate change**: under an explicit limit, a runaway program that previously ran until
   host OOM now fails with `MEM001`. That is the feature. **Deliberate NON-change**: `Mem` is
   NOT added to effect rows — allocation remains pure; a `! {Mem[...]}` spelling is explicitly
   rejected in this design (would contaminate every pure signature; see Axiom table A3/A8).

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| 1 | Effect-budget machinery exists as pattern precedent | `grep -rn 'budget' internal/effects/*.go` | `internal/effects/budget.go` — `NewBudgetContext`, `CheckAndConsume`; test asserts `effect 'IO' budget exhausted: semantic limit=5, used=5 (physical: 10)` + hint |
| 2 | `@limit` budget syntax already parses in effect rows (NOT extended here) | read `internal/parser/parser_effect.go:13` | comment documents `! {IO @limit=5, FS @limit=2}` and `@min` |
| 3 | **NEGATIVE**: no memory accounting/enforcement in `internal/eval`, `internal/vm`, `internal/runtime` | `grep -rniE 'memstats|readmemstats|memory'` over those dirs | empty (only hits: `internal/pipeline/metrics.go` — compile-phase metrics, no enforcement) |
| 4 | **NEGATIVE**: no `debug.SetMemoryLimit` usage anywhere | `grep -rn 'SetMemoryLimit' internal/ cmd/` | empty |
| 5 | **NEGATIVE**: `MEM001` unallocated in code + registry | `grep -rn 'MEM001\|"MEM' internal/ cmd/ dist/error_codes.json`; registry holds 58 records, none MEM/budget | empty — code is free |
| 6 | Harness executes generated Python with no memory limit | read `internal/eval_harness/python.go:93` (`uv run … --`) | confirmed, no rlimit/wrapper |
| 7 | Incident evidence: 3 Python procs ~80-120GB + ailang ~7.7GB at Jetsam time; panic = watchdogd starvation under swap-thrash | parsed `JetsamEvent-2026-07-20-155120.ips` (largest=Python; rpages 7.7M/6.4M/5.2M/3.5M(ollama)/0.5M(ailang)), read `panic-full-2026-07-20-160307.0002.panic` | confirmed |
| 8 | Regression fixtures exist | `ls examples/expected_fail/effect_budgets.ail` (45 lines), `examples/runnable/split_map_join.ail` (manifest 188, CI-gated) | confirmed |
| 9 | `--timeout` exists as the precedent host-boundary abort (mechanism audit deferred to implementer as a REQUIRED task, not assumed) | CLI help / debugging guide flag table | flag exists; its interrupt pathway is deliberately NOT claimed here — see Deferred Decisions |

## Examples

### Example 1: The memory bomb, before/after

**Before (no limit — today):** a generated accumulator recurses appending to a list; the
`ailang` process grows until the host swaps/panics or an external supervisor kills it.
Exit is a SIGKILL with no structured error; eval reports can't tell it from a hang.

**After (`ailang run --max-mem 512MiB bomb.ail`):**
```
Error: MEM001: memory budget exhausted: limit=512MiB, used=538MiB
Hint: raise --max-mem or reduce allocation; large accumulations may need streaming/fold
(exit code non-zero; host unaffected)
```

### Example 2: Eval harness banks the failure as signal

The rig sets `AILANG_MAX_MEM=8GiB` for benchmark children. A qwen-generated bomb becomes
`error_category: "resource_limit"` in the result JSON — a *model-capability signal* in the
leaderboard instead of a rig outage.

## Success Criteria

- [ ] Deliberate allocator under `--max-mem 512MiB` → `MEM001`, bounded time, host RSS ≤ limit + documented slop
- [ ] No flag set → `make verify-examples` green, behavior byte-identical (monitor never starts)
- [ ] `MEM001` in registry + `ailang docs search` finds it; expected-fail example green in CI
- [ ] Harness categorizes `MEM001` distinctly; visible in `eval-report`
- [ ] All tests passing; docs + CHANGELOG updated

## Testing Strategy

**Unit tests:** memguard trip/no-trip; flag parsing (`512MiB`, `4GiB`, bad input fails loud); inertness when 0.
**Integration tests:** run a real bomb `.ail` under limit → assert `MEM001` + exit code; run corpus under generous limit → all pass.
**Manual:** observe host RSS during a bomb run; confirm no swap growth.

## Deferred Decisions

- **Where the cooperative check lands** (eval back-edges vs builtin dispatch vs both) — agent
  decides after auditing how `--timeout` interrupts evaluation; REQUIRED first task of Phase 1b
  is that audit, recorded in the sprint notes (per the m-module-less case study: verify the
  mechanism by reading the dispatch, not by observing output)
- Poll interval + slop target tuning — agent, within the ≤25% slop success criterion
- Size-literal parsing (reuse an existing helper if one exists — agent greps first)

## Non-Goals

- **`Mem` as an effect row entry** — rejected by design (A3/A8; allocation is pure)
- **Deterministic logical allocation meter** — Phase 2, separate doc (`m-mem-meter-logical`),
  gives replayable exhaustion at an op index; not needed for the host-safety floor
- **Per-function / per-region budgets** — future, depends on Phase 2
- **Harness RSS watchdog for Python/JS/Go lanes** — separate task (already in flight), harness-scoped
- **WASM memory limits** — browser runtime has its own memory model; out of scope

## Timeline

**Days 1-2:** Phase 1a + 1b (memguard, flag, typed unwind, error registration, example)
**Day 3:** Phase 1c (harness category, docs, CHANGELOG) + corpus regression run
**Total: ~2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Go soft limit causes GC thrash near the limit (slow death instead of clean trip) | Med | Monitor trips at 100% of budget while `SetMemoryLimit` is set ~10% above it — the trip fires before GC death-spiral territory |
| Cooperative check misses tight non-allocating loops | Low | Such loops don't grow memory; timeout remains the backstop for pure spins |
| ReadMemStats poll perturbs performance | Low | 250ms cadence; benchmark compile/exec timings before/after in tests |
| Two abort mechanisms (timeout + mem) race | Med | Single shared cancellation pathway, first-trip-wins, reason recorded once |

## Related Documents

- [m-dx25-budget-report](../../implemented/v0_7_1/m-dx25-budget-report.md) — cost/effect budget *reporting*; message-shape precedent (0.45)
- [m-eval-bounded-pipeline-sprint-plan](../../planned/v0_29_0/m-eval-bounded-pipeline-sprint-plan.md) — harness-side boundedness (0.42); distinct: harness vs language runtime
- [m-mission-cost-chains](../../planned/v0_30_0/m-mission-cost-chains.md) — cost budgets for the mission loop (0.41); distinct resource
- Harness RSS watchdog — spawned task 2026-07-21 (separate session), complements this doc

## References

- [Design Axioms](/docs/references/axioms) — esp. A9 Cost Visibility, A11 Structured Failure
- Incident artifacts: `panic-full-2026-07-20-160307.0002.panic`, `JetsamEvent-2026-07-20-155120.ips`
- `internal/effects/budget.go` — the budget-context pattern this mirrors
- Go `runtime/debug.SetMemoryLimit` (soft limit, Go ≥1.19)

## Future Work

- **Phase 2 — `m-mem-meter-logical`**: deterministic VM-level allocation meter (cons/string/record
  accounting) → replayable exhaustion at exact op index, trace integration, per-region budgets
- Public write-up: "Your AI will write a memory bomb eventually" — per-language autopsy of the
  2026-07-20 panic + why resource bounds belong in language semantics

---

**Document created**: 2026-07-21
**Last updated**: 2026-07-21

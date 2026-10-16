# crew-core cold compile: #1811 fix verification, residual time, and workaround retirement

**Status**: Planned
**Target**: v0.55.0
**Priority**: P1
**Estimated**: ~1.5 weeks
**Dependencies**: None. Builds on the v0.54.8 fix (already shipped); verifies it rather than depending on it.

## Problem Statement

Against AILANG v0.54.6 (darwin arm64, commit `71f4cf6`), a cold compile of
`sunholo/crew_core`'s `session.ail` (683 lines; a wide `Session` record nesting the
17-field social_dynamics `State`, 15-field `Config`, 18-field `Scenario`) blew up:

| Command (v0.54.6, cold) | Time | Peak RSS |
|---|---|---|
| `ailang check session.ail` | 21.1 s | 9.1 GB |
| `ailang run --bytecode --entry encodeHostError` | 20.4 s | 19.4 GB |
| `ailang run` (interpreter) | 20.5 s | 19.9 GB |
| `--max-memory 4GB/2GB/1GB` | 26/43/52 s | 7.5–7.8 GB peak |
| warm cache afterwards | 0.4 s | 219 MB |

Reported impact: stapledons-godot CI ran two concurrent cold sims at 8–9 GB each on a
16 GB runner and was SIGTERMed (exit 143); an exported game build compiling the sim
cold on first launch would cost a player ~20 GB; the registry validator OOMed at
512 MiB on crew_core/crew_lab (ailang#1811, seven kills 10-10…10-13).

**The memory blowup is fixed and that fix shipped in v0.54.8** (first changelog entry,
"Fixes #1811"). This doc (a) independently verifies that fix on a second platform,
(b) designs the work that remains — cold-compile **time** — with a CPU profile as
evidence, and (c) retires the downstream workarounds the bug forced.

### Root cause of the original blowup (shipped fix, for the record)

`generalize` copied **every** unsolved class constraint in the inference context into
each let-bound scheme — not just the ones mentioning the binding's generalized
variables — and `inferVar` re-emitted them all at every use. Each nested let
multiplied the constraint list exponentially. `sunholo/crew_core` reached millions of
constraints (~4 GB live, 10–14 GB peak with GC lag), OOM-killing the 512 MiB registry
validator. v0.54.8's `schemeConstraintsFor` (`internal/types/scheme_constraints.go`)
keeps only constraints that mention a generalized variable, once each; the rest stay
in the append-only `ctx.constraints` and are enforced at the decl boundary. The guard
test `TestSchemeConstraints_NestedLetsStayLinear` (`internal/pipeline/scheme_constraint_growth_test.go`)
asserts a depth-12 let chain allocates under 64 MB (it allocated 25.7 GB before).

## Goals

**Primary Goal:** Make a cold compile of a crew-core-shaped module cost seconds and
megabytes, not minutes and gigabytes — verify the memory half (done, below), and fix
the residual time half.

**Success Metrics:**
1. Cold `ailang check session.ail` (deps cached): **≤ 2.5 s** from 7.0 s measured on
   the fix release (CI-grade hardware; see measurement environment below).
2. Cold `ailang check --package .` of crew_core (17 files incl. tests): **≤ 12 s**
   from 29.1 s.
3. Peak RSS unchanged: ≤ 300 MB (today 134–250 MB) — memory must not regress while
   speeding up.
4. A wall-clock + allocation regression guard for the wide-record/deep-let shape runs
   in CI (today only the allocation guard exists — see V9).
5. stapledons-godot retires its memory-limited warm-up run (documented procedure, not
   a code change here).

## Independent verification of the v0.54.8 fix (this session)

Environment: linux x64 (Cloud Run, 5 vCPU, 3.8 GB), binary `ailang v0.54.8`
(`b283b2d4`, built 2026-10-13), ailang-packages @ `6dfc0fa`, crew_core deps as
**path** dependencies (registry unreachable from the sandbox; content-identical —
`ailang lock` resolved sunholo/social_dynamics@0.4.0, decisions@0.4.0,
content_library@0.1.0 from the same clone). Cold = `.ailang` deleted. Peak RSS via
`/proc/<pid>/status` VmHWM polling at 50 ms.

| Measurement (v0.54.8, linux x64) | Time | Peak RSS |
|---|---|---|
| `check session.ail`, fully cold (all 30 dep modules compile) | 13.0 s | 132 MB |
| `check session.ail`, session cold only (deps cached) | 7.0 s | 124 MB |
| — compile phase only (`-debug-compile`) | 7.05 s | — |
| `check session.ail`, warm | 0.84 s | 102 MB |
| `run --bytecode --entry bankLimit`, cold | 7.2 s | 239 MB |
| `run --entry bankLimit` (interpreter), cold | 7.1 s | 239 MB |
| `check --package .` (17 files incl. `_test` modules), fully cold | 29.1 s | 134 MB |

Memory: fixed (9.1 GB → 0.13 GB; both engines; matches the v0.54.8 changelog's own
"crew_core 11 s / 149 MB"). Time: still ~7 s for one 683-line module and ~10 ms/line
— the remaining problem this doc designs.

## The residual: where the 7 seconds go (CPU profile, measured)

`ailang run --entry bankLimit --cpuprofile` on cold session.ail (deps cached):
6.67 s duration, 6.78 s of samples.

| Cumulative cost | % | What it is |
|---|---|---|
| `types.ApplySubstitution` | 61.7% (4.18 s) | substitution application during typecheck |
| ← `applyApplicationSubstitution` → `CoreTypeInfo.ApplySubstitution` | 30.5% (2.07 s) | **every** application's local solve (#386) re-applying the substitution to every typed node in the subtree |
| ← `Unifier.Unify` (per-call, both sides) | ~21% (1.40 s) | unification re-applying the growing substitution |
| `types.safeSubstitute` | 36.3% (2.46 s) | dominated by `map[string]Type` record-field map ops (`maps.Iter.Next` alone 20.8%; map assign/access/iterate ≈ all of it) |
| `types.putTypeTypeMap` (pooled visited-map reset) | 24.6% (1.67 s) | includes 0.78 s of `map` Delete churn — the pool pays for itself in allocs and loses it back in clearing |
| `collectFreeEffectRowVarsInType` ← `TypeEnv.collectFreeEffectRowVars` | 8.9% (0.60 s) | `generalize`'s full environment-chain walk per let binding |

Scaling laws (synthetic crew-core-shaped module, generator in the sprint repo notes;
v0.54.8): time ~n^1.25 in function count (8/16/32 funcs → 1.64/3.21/7.80 s), linear
in record width (9/17/25 fields → 1.79/3.21/4.73 s), memory flat ≤ 140 MB. So the
residual is an O(field-accesses × record-width) cost with a large map-iteration
constant — a quadratic *constant*, not a second explosion. It does not threaten the
validator's 384 MiB cap; it costs player-visible seconds on first launch and CI
minutes on every cold run.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Keep #386's per-application local solve, but narrow what it rewrites (effect-row operands only) vs. keep full CoreTypeInfo rewrite | 32% of CPU sits behind this path; wrong narrowing regresses #386 (show/println effect-row resolution) | human (design) | design | high |
| D2: Epoch-stamped visited maps instead of per-call Delete churn in the substitution pool | 24.6% of CPU is pool clear/delete; touches cycle-safety of `safeSubstitute` | agent | compile | med |
| D3: `generalize` reads `ctx.qualifiedConstraints` (already maintained by `addConstraint`) instead of filtering the whole `ctx.constraints`; dedup key without `Type.String()` | O(bindings × constraints) scans with 100 KB-class string keys; pure bookkeeping, but must not drop constraints (soundness of #1811 fix) | agent | compile | med |
| D4: Guard is a *time* budget test with 3× headroom + the existing 64 MB allocation guard, on a synthetic fixture (no network, no registry) | Wall-clock guards flake on loaded CI; too-tight budgets rot into skipped tests | human (design) | design | low |
| D5: Game-export warm-cache guidance (ship `.ailang` cache with the exported build) vs. accept ~13 s first launch | Player-visible first-run experience; generalizes the v0.54.8 motoko Dockerfile pattern | human (design) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1: confirm the narrowing of `applyApplicationSubstitution` preserves #386's
      fixture (M-EFFECT-ROW-SHOW-INTERP) — the implementing sprint must run that
      fixture first; if narrowing is unsafe, fall back to epoch-memoized
      substitution (Phase 2, item 2's fallback) for the same win.
- [ ] D4: approve the guard's time budget and fixture shape.
- [ ] D5: approve the export-pipeline guidance (docs-only in this doc's scope).

## Solution Design

### Overview

Three independent, profile-ranked optimizations inside `internal/types`, plus a
regression guard and downstream retirement. No syntax, no type-system semantics, no
new diagnostics. Every change is behind the same invariant: **a valid program must
typecheck to the same result** (same substitution, same constraints solved, same
errors) — only the cost of getting there changes.

### Architecture

**Phase 1 — constraint bookkeeping (targets the 8.9% + part of the generalize path; ~1 day)**
1. `generalize` (`internal/types/inference_helpers.go:152-159`) currently filters
   ClassConstraints out of the whole `ctx.constraints` on every let binding, although
   `addConstraint` (`internal/types/inference.go:569-577`) already appends every
   ClassConstraint to `ctx.qualifiedConstraints`. Read that slice instead.
2. `schemeConstraintsFor`'s dedup key is `c.Class + "\x00" + c.Type.String()`
   (`internal/types/scheme_constraints.go:36`) — `String()` renders wide record
   types (up to ~100 KB for Session-shaped types) per constraint per binding. Replace
   with a cheap structural key: the class name plus a hash collected during the
   free-vars walk that the function already performs (one traversal, no string
   building). Deterministic, so cache keys and diagnostics are unaffected — the key
   is internal-only.
3. The effect-row half of `generalize` walks the full env chain
   (`TypeEnv.collectFreeEffectRowVars`, 0.59 s); mirror the `FreeTypeVarsAbove`
   frame-windowing that the type-var half already uses (`internal/types/env.go:98`).

**Phase 2 — substitution cost (targets the 62%; ~4 days)**
1. **Epoch-stamped visited maps** (D2). `safeSubstitute`'s cycle-detection pool
   (`getTypeTypeMap`/`putTypeTypeMap`, `internal/types/pool.go:46-51`) clears by
   deleting every entry on return to the pool — 0.78 s of Delete churn per cold
   check. Stamp each entry with a monotonically increasing epoch counter instead of
   deleting; a lookup whose entry carries an older epoch is a miss. Cycle safety is
   unchanged (an entry is valid exactly for the substitution pass that wrote it).
2. **Narrow `applyApplicationSubstitution`** (D1). It exists so the #386
   show/println effect-row union sees resolved rows *before* `combineEffects`. It
   currently rewrites every typed node's type in the application subtree through
   `CoreTypeInfo.ApplySubstitution` (30.5% of CPU). Preferred narrowing: resolve only
   the operands that feed the effect combination (argEffect rows and the callee
   effect row) plus any type the local substitution actually binds in this subtree —
   the #386 comments already describe this as the intent. Fallback if the fixture
   disagrees: memoize per-node substitution results for the duration of one
   application's local solve so each node is rewritten at most once per epoch.
3. **(Stretch)** `Unify` re-applies the substitution to both sides of every call
   (1.40 s). A destructive union-find on `TVar2` cells (bind-once, path-compressed)
   removes most of this; it is the largest change here and lands only if Phases 1–2
   miss the ≤ 2.5 s goal. Do not attempt in the same sprint as D1.

**Phase 3 — guard + downstream (~2 days)**
1. Add a wall-clock + allocation guard beside
   `TestSchemeConstraints_NestedLetsStayLinear`: the synthetic wide-record fixture
   (deterministic generator, no registry, ~600 lines, State 17 fields / Session
   11 fields / 24 functions) must check in under a budget with 3× headroom over the
   post-Phase-2 measurement, and under the existing 64 MB allocation shape. Assert
   both; report both in the failure message (measured vs budget), like the allocation
   guard does today.
2. Retire the stapledons-godot warm-up workaround: on ≥ v0.54.8 two concurrent cold
   sims measure ~240 MB each (this session, both engines) — the 16 GB runner holds
   them with 30× headroom. Record in the game's CI as a one-line removal plus a
   comment pinning the version where the numbers were measured.
3. Game export guidance (D5): document shipping the warm `.ailang` compile cache
   inside the exported build (the v0.54.8 `docker/Dockerfile.agent-motoko` pattern,
   which fails the build if the cache write fails), or accepting the post-Phase-2
   cold time. Docs-only; the export pipeline itself is out of scope.

### Implementation Plan

**Phase 1: bookkeeping** (~1 day)
- [ ] `generalize` reads `ctx.qualifiedConstraints`; delete the redundant filter loop
- [ ] structural dedup key in `schemeConstraintsFor` (single-walk hash)
- [ ] frame-windowed effect-row free vars in `generalize`
- [ ] `make test-core`, `make test` (pipeline + types suites), profile delta recorded

**Phase 2: substitution cost** (~4 days)
- [ ] Run the #386 fixture (M-EFFECT-ROW-SHOW-INTERP) and pin its behaviour first
- [ ] Epoch-stamped visited maps; benchmark before/after (`bench/` harness exists)
- [ ] Narrow `applyApplicationSubstitution` (or memoize per epoch) behind D1's decision
- [ ] Re-profile cold session.ail; record the new table next to this doc's table
- [ ] (only if goal missed) design note for union-find TVar2 cells as a follow-up doc

**Phase 3: guard + retirement** (~2 days)
- [ ] Time+allocation guard test with the synthetic fixture (D4 budget)
- [ ] Measure `check --package .` on crew_core; confirm ≤ 12 s and ≤ 300 MB
- [ ] stapledons-godot CI workaround removal note; export-pipeline docs (D5)
- [ ] Update `docs/docs/guides/` performance note + this doc's status

### Files to Modify/Create

**New files:**
- `internal/pipeline/compile_time_guard_test.go` — the D4 guard (~120 LOC)
- (fixture generator, test-only, ~80 LOC, or a checked-in generated `.ail` if the
  repo prefers static fixtures)

**Modified files:**
- `internal/types/inference_helpers.go` — generalize reads qualifiedConstraints; effect-row windowing (~30 LOC)
- `internal/types/scheme_constraints.go` — structural key (~25 LOC)
- `internal/types/unification_substitution.go` — epoch-stamped pool (~40 LOC)
- `internal/types/typechecker_functions.go` — `applyApplicationSubstitution` narrowing (~60 LOC)
- `docs/docs/guides/` + stapledons-godot notes (~2 docs)

## Examples

### Example 1: cold check, before → after (goal)

**Before (v0.54.8, this session):**
```
$ rm -rf .ailang && ailang check session.ail   # deps cached, session cold
→ 7.0 s, 124 MB
```
**After (goal):**
```
$ rm -rf .ailang && ailang check session.ail
→ ≤ 2.5 s, ≤ 300 MB
```

### Example 2: the guard

```go
// TestColdCompileWideRecordStaysFast guards the crew-core shape (#1811 follow-up):
// a 600-line module of wide records + deep lets must check in budgeted time AND
// bounded allocation. Time budget carries 3x headroom over the post-fix measurement
// to stay green on loaded CI runners; allocation budget stays at the 64 MB shape.
func TestColdCompileWideRecordStaysFast(t *testing.T) {
    t.Setenv("AILANG_NO_CACHE", "1")
    src := wideRecordModule(24 /*funcs*/, 17 /*state fields*/)
    ...
    if elapsed > timeBudget { t.Fatalf("cold check took %v (budget %v): ...", elapsed, timeBudget) }
}
```

## Success Criteria

- [ ] Cold `check session.ail` (deps cached) ≤ 2.5 s, ≤ 300 MB peak RSS (measured on CI hardware, numbers in the implementation report)
- [ ] Cold `check --package .` crew_core ≤ 12 s, ≤ 300 MB
- [ ] Depth-12 let-chain allocation guard still passes unchanged (64 MB)
- [ ] New time guard green in CI for 2 consecutive weeks (no flake-driven skips)
- [ ] `make test-core`, `make test`, `make check-boundaries` all pass
- [ ] #386 fixture (M-EFFECT-ROW-SHOW-INTERP) passes unchanged
- [ ] stapledons-godot warm-up workaround removal documented
- [ ] This doc's verification table extended with post-change numbers

## Testing Strategy

**Unit tests:**
- Existing `internal/types` suites (unification, substitution, scheme constraints) must pass unchanged — they pin the semantics this doc promises not to alter.
- New: dedup-key equivalence test (old String() key vs new structural key agree on a corpus of constraint types, incl. wide records).

**Integration tests:**
- `internal/pipeline` end-to-end: cold check of the synthetic fixture produces byte-identical artifacts (coretypeinfo.gob, iface.json) before/after each phase — the strongest available proof that only cost changed.
- crew_core path-dep check (this doc's method) as a manual smoke, not a CI dependency (registry/network).

**Manual testing:**
- Cold/warm `ailang check` + both run engines on crew_core session.ail, table refreshed.
- stapledons-godot CI run without the warm-up step.

## Conflict Surface

This design touches `internal/types` (constraint bookkeeping, substitution, one
typechecker path) — no parser, lexer, AST, or codegen changes.

### Syntactic positions touched

None. No grammar production, token position, or AST shape changes.

### What else lives here (semantic positions, since this is typechecker-internal)

| Position | Existing behavior that must be preserved | Threat |
|---|---|---|
| `generalize`'s constraint selection | Constraints on **non**-generalized variables stay in `ctx.constraints` and are enforced at the decl boundary (#1811 fix invariant) | Reading `qualifiedConstraints` instead of filtering must not drop or add any constraint; the two sources are populated by the same `addConstraint` call site |
| `schemeConstraintsFor` dedup | Each (class, type) kept once; ordering of `out` currently follows `ctx.constraints` order | A key change must not alter *which* constraint object survives dedup (first-wins) — diagnostics and dictionary resolution can observe instance identity |
| `safeSubstitute` cycle detection | Visited-map entries are valid only within one substitution pass; cyclic types (recursive ADTs, mu-types) rely on it | Epoch stamping must not let an entry from a *previous* pass satisfy a lookup (stale = miss, never hit) |
| `applyApplicationSubstitution` (#386) | Local solve resolves operand effect rows before `combineEffects`; enclosing-scope variables are withheld; original equality constraints are left in place and replayed at the decl boundary | Narrowing must preserve: `show` → closed `{}`, `println` → closed `{IO}`, recursive self-call row withholding, and the replay semantics |

### Disambiguation strategy

Not applicable (no syntax). The analogue is the artifact-equivalence integration
test above: identical compile outputs before/after per phase.

### Programs that MUST still work

1. `internal/pipeline/scheme_constraint_growth_test.go` fixtures (depth-12 let chain) — the #1811 guard itself.
2. `std/list.ail`, `std/json.ail` — heavy higher-order users of schemes and constraints (`map`, `filter`, `flatMap` are exactly the let-instantiation shape).
3. `std/sem.ail:207,213` — record updates (`{f | embedding: …}`, `{f | ver: …}`), the construct crew-core multiplies; plus the record fixtures `examples/record_cons_pattern.ail`, `examples/record_in_result.ail`, `examples/record_list_extraction.ail`.
4. `examples/deriving_eq.ail`, `examples/eq_containers.ail` — `deriving (Eq)` paths that exercise the constraint-dedup key.
5. The #386 fixture: M-EFFECT-ROW-SHOW-INTERP tests in `internal/types` (effect rows for `show`/`println`), and `sunholo/crew_core` session.ail (external consumer; path-dep method in this doc) — the motivating program.

### What deliberately changes

Nothing observable. Compile artifacts, diagnostics, and accepted/rejected programs are
byte-for-byte unchanged; only CPU time and (slightly) allocation volume change. Any
diagnostic or artifact diff observed during the sprint is a regression, not an
intentional change.

## Verification Log

Every language/tooling claim above carries live or code-level evidence. Live checks
ran against `ailang v0.54.8` (`b283b2d4`, binary `/usr/local/bin/ailang`) on linux x64,
ailang-packages @ `6dfc0fa`.

| ID | Claim | Evidence | Verdict |
|----|---|---|---|
| V1 | The reported v0.54.6 blowup is fixed by v0.54.8's scheme-constraint relevance filter | `internal/types/scheme_constraints.go:1-40` (root cause + fix comment naming crew_core and #1811); changelogs/v0.32-current.md v0.54.8 first entry; release commit `b283b2d4` "Fixes #1811" | Confirmed |
| V2 | Cold `check session.ail` on v0.54.8 fits in ~130 MB (was 9.1 GB) | Live, fully cold (`.ailang` deleted): 13.0 s / 132 MB; deps-cached cold: 7.0 s / 124 MB (`-debug-compile`: Compile 7053 ms, Loading 31 ms); warm 0.84 s / 102 MB. Peak via VmHWM polling | Confirmed |
| V3 | Both engines fit in ~240 MB cold | Live: `run --bytecode --entry bankLimit` 7.2 s / 239 MB; interpreter 7.1 s / 239 MB. (`--entry encodeHostError --args-json 0` now fails earlier at args decode: "unsupported type constructor: HostError" — compile still completed; the 250 MB peak includes it) | Confirmed |
| V4 | Residual time profile: 61.7% `ApplySubstitution`, 30.5% behind `applyApplicationSubstitution`, 24.6% `putTypeTypeMap`, 8.9% effect-row env walk | Live: `ailang run --entry bankLimit --cpuprofile` on cold session.ail, `go tool pprof -top/-cum/-peek` (profile `/tmp/crew.prof`, 6.78 s samples); callers: `CoreTypeInfo.ApplySubstitution` 2.03 s, `Unify` 1.40 s, `SolveConstraints` 0.59 s | Confirmed |
| V5 | `generalize` filters the whole constraint list although `qualifiedConstraints` already exists | Code: `internal/types/inference_helpers.go:152-159` (loop over `ctx.constraints` with ClassConstraint type-assert) vs `internal/types/inference.go:569-577` (`addConstraint` appends every ClassConstraint to `ctx.qualifiedConstraints`) | Confirmed |
| V6 | The #1811 dedup key renders wide types to strings | Code: `internal/types/scheme_constraints.go:36` `key := c.Class + "\x00" + c.Type.String()`; String() of Session-shaped TRecords is the ~100 KB rendering the reporter observed in type errors | Confirmed |
| V7 | The effect-row half of generalize walks the full env chain per binding | Code: `internal/types/env.go:189-195` + `collectFreeEffectRowVars` recursion through `env.parent`; profile 0.59 s; contrast `FreeTypeVarsAbove` (`env.go:98`) which the type-var path already windows | Confirmed |
| V8 | `--max-memory` is a soft limit and cannot cap RSS below live heap (the reporter's 7.5 GB observation) | `cmd/ailang/main_run.go:133` flag help: "Go soft memory limit … Unset = AILANG_MEMLIMIT, else none"; it sets GOMEMLIMIT semantics only. Post-fix live heap is ~150 MB, so the flag is no longer load-bearing for this module | Confirmed |
| V9 | No wall-clock regression guard for the wide-record/deep-let shape exists today | `grep -rln Benchmark internal/pipeline/` → only `validate_coretypeinfo_bench_test.go` (CoreTypeInfo validation micro-bench, not compile); the only #1811-shaped guard, `TestSchemeConstraints_NestedLetsStayLinear`, asserts `TotalAlloc` ≤ 64 MB only (test body read: no time assertion); `bench/` holds lsp/vm harnesses, none compile-time-budgeted | Confirmed |
| V10 | The registry validator now bounds the child check (384 MiB / 200 s) and reports 422 | Code: `cmd/registry-validator/bounded.go:24-27` (`checkRSSCap = 384 << 20`, `checkWallCap = 200 * time.Second`, comment names #1811 and crew_core/crew_lab); changelog v0.54.8 | Confirmed |
| V11 | Cold package check of crew_core passes the validator budget on v0.54.8 | Live: `check --package .` (17 files, incl. `_test` modules) 29.1 s / 134 MB < 200 s / 384 MiB | Confirmed (headroom noted: 29 s of a 200 s budget is 15%) |
| V12 | Residual scaling is mildly superlinear (~n^1.25), not explosive | Synthetic generator (nested wide records, `{s \| f: …}` chains, ensures clauses): 8/16/32 funcs → 1.64/3.21/7.80 s; 9/17/25 fields → 1.79/3.21/4.73 s; RSS flat ≤ 140 MB | Confirmed |
| V13 | The compile cache (not the fix) delivers warm speed | Live: deleting only `.ailang/cache/compile/modules/session` restores 7.0 s; cache dir for session is 1.7 MB; warm 0.84 s | Confirmed |
| V14 | A warm-cache-shipping pattern already exists to generalize (motoko image) | changelog v0.54.8 "the motoko image ships a warm compile cache" (`docker/Dockerfile.agent-motoko` runs `ailang check` at build, fails on missing cache) | Confirmed |
| V15 | `ailang run` exposes `--cpuprofile`/`--memprofile` (profiling needs no new CLI surface) | `cmd/ailang/main_run.go:137-138,193-203` | Confirmed |

Negative-existence claims (V9) were greped at repo scope (`internal/`, `bench/`,
`benchmarks/`), not inferred.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Identical compile artifacts before/after is an explicit acceptance test |
| A2: Replayability | 0 | No trace or replay surface touched |
| A3: Effect Legibility | 0 | Effect rows resolved identically; #386 fixture pinned |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Local checks get faster; the guard keeps wide-record modules budget-verifiable |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Faster cold compiles serve AI-driven check/run loops directly; guard is machine-checked, not folklore |
| A8: Minimal Syntax | +1 | No syntax added |
| A9: Cost Visibility | +1 | The guard makes compile cost a measured, budgeted quantity |
| A10: Composability | 0 | Internal optimization; module boundaries unchanged |
| A11: Structured Failure | 0 | Diagnostics unchanged (byte-identical artifacts) |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: no implicit nondeterminism (artifact-equivalence test)
- [x] A3: no hidden effects
- [x] A4: no ambient access
- [x] A7: not optimizing for human convenience over machine analysis

## Deferred Decisions

- Exact structural-hash function for the dedup key (D3) — agent may choose, provided the equivalence unit test passes on the type corpus.
- Whether the synthetic guard fixture is generated at test time or checked in — agent may choose (deterministic either way; checked-in favors reviewability).
- Whether Phase 2's stretch item (union-find TVar2 cells) becomes its own doc — agent decides after re-profiling; if Phases 1–2 meet the ≤ 2.5 s goal, park it.
- The export-pipeline mechanism for shipping warm caches (D5) — docs here name the pattern; the game repo decides integration.

## Non-Goals

- **No type-system semantics changes.** Constraint selection, generalization, and unification results are frozen; this is a cost-only change (the #1811 fix already changed the semantics that needed changing).
- **No record-representation redesign** (e.g. replacing `map[string]Type` fields with sorted slices, or hash-consing). The profile shows map iteration as the constant inside `safeSubstitute`, but Phase 2's memoization/epoch work attacks the multiplier instead. Revisited only if the ≤ 2.5 s goal is missed.
- **No new CLI flags.** Profiling flags already exist (V15).
- **No caching mandates.** The compile cache stays optional; cold-path speed is the deliverable.
- **crew_lab** — not present in the cloned ailang-packages checkout at `6dfc0fa`; the changelog's 32 s / 201 MB measurement is cited but not re-verified here.

## Timeline

**Week 1** (~5 days):
- Phase 1 (1 day), Phase 2 items 1–2 (3 days), re-profile + artifact-equivalence runs (1 day)

**Week 2** (~4 days):
- Phase 2 item 3 narrowing + #386 verification (2 days), Phase 3 guard + retirement + docs (2 days)

**Total: ~9 days** (estimates 2x the naive guess, per house convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Narrowing `applyApplicationSubstitution` regresses #386 (show/println effect rows) | High — silent effect misresolution | Fixture pinned before the change; fallback to epoch-memoization (same win, no semantic risk); D1 requires the human decision |
| Dedup-key change alters which duplicate constraint survives (instance identity observable in dictionaries) | Med — dictionary resolution drift | Equivalence unit test over the type corpus; byte-identical artifacts gate per phase |
| Epoch-stamped pool breaks cycle detection (a stale entry satisfies a lookup) | Med — hang or wrong type on recursive types | Each pooled map carries a monotonic epoch; lookups compare stamps, so an older epoch is a miss by construction, never a hit. Recursive-ADT cycle tests in `internal/types` gate the change |
| Time-budget guard flakes on loaded CI | Med — guard gets skipped, protection rots | 3x headroom (D4); allocation guard stays as the strict half |
| Perf gains eaten by the `map[string]Type` iteration constant | Med — goal missed | Re-profile gates each phase; the non-goal redesign is the documented escape hatch |
| v0.54.8 numbers don't reproduce on other hardware | Low — this doc's table is linux x64 only | The guard measures on the CI hardware it protects; the doc records the measurement environment explicitly |

## Related Documents

**Implemented / shipped:**
- changelogs/v0.32-current.md §v0.54.8 — the #1811 fix, validator budgets, motoko warm-cache pattern (upstream of this doc)
- `internal/pipeline/scheme_constraint_growth_test.go` — the allocation guard this doc extends

**Planned (check for overlap):**
- [stapledon-v0-54-6-workarounds.md](../../implemented/v0_54_7/stapledon-v0-54-6-workarounds.md) — the game's workaround register; this doc retires the warm-up workaround once ≥ v0.54.8 (distinct: that doc routes engine gaps W1–W4, not compile cost)
- No existing planned or implemented doc covers #1811 or cold-compile cost (duplicate gate: SimHash + neural search both returned no matches at creation time)

## References

- ailang#1811 — registry validator OOM on crew_core/crew_lab (closed by v0.54.8)
- Design Axioms — /docs/references/axioms
- M-EFFECT-ROW-SHOW-INTERP (#386) — the constraint the `applyApplicationSubstitution` narrowing must preserve (`internal/types/typechecker_functions.go:522-640` comments)

## Future Work

- Union-find / destructive-substitution variable cells (Phase 2 stretch, own doc if pursued)
- Sorted-slice or hash-consed record field representation (the map-iteration constant)
- crew_lab re-measurement when its source is reachable from this environment
- A `bench/` ladder entry tracking cold-compile time per release, so regressions surface before a downstream 4 AM page

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13

# M-TYPECHECK-ENV-SCAN-COST: stop re-walking the whole type environment on every application

**Status**: Planned
**Target**: v0.56.2 (before the release that ships #1958's re-land)
**Priority**: P1
**Estimated**: ~4 engineering days
**Dependencies**: None. Absorbs Phase 1 of [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) (the environment-walk item); that doc keeps the substitution-cost work.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same inputs give the same outputs: the change caches a pure function of immutable frames, and Phase 2's equivalence gate proves byte-identical artifacts |
| A2: Replayability | 0 | No trace or artifact changes |
| A3: Effect Legibility | 0 | Effect inference is unchanged; only the cost of computing "which rows the enclosing scope owns" drops |
| A4: Explicit Authority | 0 | No capability surface touched |
| A5: Bounded Verification | +1 | Makes the type check's cost grow with program size rather than with applications × environment size, so the validator's fixed budget stops capping package size |
| A6: Safe Concurrency | 0 | Type checking is single-threaded (V9); caches are per-frame, and a `-race` run is part of the gate |
| A7: Machines First | +1 | Every agent edit loop pays `ailang check`; on ailang_parse that is ~100 s cold today |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Adds a deterministic work counter (bindings visited per check) that tests and the registry can read, replacing wall-clock guesses |
| A10: Composability | 0 | No language-surface change |
| A11: Structured Failure | 0 | No error changes |
| A12: System Boundary | 0 | Internal to `internal/types` |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (cache is keyed on frame identity and invalidated on the only in-place writes, V7)
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimises machine check time

## Problem Statement

The registry validator stops `ailang check --package .` at 200 s. That cap cannot move much: Cloud Run's request timeout is 300 s and released `ailang publish` clients abandon the upload after 5 minutes (`cmd/ailang/pkg_publish.go:261`). `sunholo/ailang_parse` (93 checked `.ail` files) has sat at the cap since 2026-10-27, failed 7/7 publishes on 1 vCPU on 2026-11-02, and passed on 2 vCPU at 170.8 s (0.55.2, validator v0.56.0) and 204.2 s total (0.55.4, validator v0.56.1). Raising resources further buys little: the check is effectively single-threaded (V10).

The cost is not the package's fault. A cold check profile (macOS `sample`, 1 ms, 50–60 s window, ailang_parse at `c3b4fc6`) puts **98–99% of the check in `pipeline.typeCheckAndLowerModule`**, and most of that in three walks over the type environment:

| Inclusive share of the check | v0.55.0 | dev HEAD (`8aae2f819`+) |
|---|---|---|
| `TypeEnv.collectFreeEffectRowVars` | 63.2% | 70.5% |
| `TypeEnv.collectFreeTypeVars` | 28.6% | 31.9% |
| `TypeEnv.collectFreeRowVars` | 21.2% | 21.9% |
| ← called directly from `inferApp` (all three) | 60.9% | 57.9% |
| ← via `envOwnedVars` (`unionFrame.solve`, new in `d555a52ac`) | — | 6.2% |
| `safeSubstitute` / `ApplySubstitution` (the crew-core doc's target) | 30.7% / 27.2% | 27.2% / 23.1% |

(The walks overlap — `collectFreeEffectRowVars` recurses through the same chain — so the rows do not sum.)

**Root cause.** `inferApp` (`internal/types/typechecker_functions.go:545`) snapshots the variables owned by the enclosing scope on **every application**:

```go
// typechecker_functions.go:561-570
outerVars := make(map[string]bool)
for v := range ctx.env.FreeTypeVars() { outerVars[v] = true }
for v := range ctx.env.FreeRowVars() { outerVars[v] = true }
for v := range ctx.env.FreeEffectRowVars() { outerVars[v] = true }
```

Each of those walks every binding of every frame up to the root (`env.go:116-138`, `147-187`, `195-219` recurse through `env.parent`): the enclosing lambdas and lets, the module's top-level declarations, every imported name and the builtins. Almost everything above the current declaration is a fully generalised scheme with no free variables, so the walk does a full type traversal of hundreds of closed types to find nothing — once per application, three times over. Cost is **applications × total environment size**, so it grows with both module size and import count. `defaultReturnOnlyRow` (`typechecker_app_effects.go:162`) and `publishApplicationWithCallee` (`:193-194`) add per-application windowed walks (`effectRowsAbove`, `implicitRowsAbove`); `generalize` (`inference_helpers.go:132,143`) and every declaration's `baseEnvFreeVars` (`typechecker_core.go:460,595`) add per-let and per-decl full walks.

**It is getting worse.** Instructions retired for a cold ailang_parse check rose from 1.38×10¹² (v0.55.0) to 1.61×10¹² (dev HEAD, +16%) with #1958's soundness re-land. On the validator that turns ~204 s into roughly 235 s — a likely refusal on ailang_parse's next publish after the next release.

**Who is affected:** every `ailang check`/`run` cold compile — publishers, CI, the registry validator, and every agent edit loop.

## Goals

**Primary Goal:** Make the "free variables of the enclosing environment" query O(new bindings) amortised instead of O(whole environment) per call, with byte-identical compile output.

**Success Metrics:**
1. Cold `ailang check --package .` of ailang_parse (`c3b4fc6`): instructions retired **≤ 0.80×10¹²** (≥ 50% below dev HEAD's 1.61×10¹²), measured with `AILANG_NO_CACHE=1 /usr/bin/time -l` on darwin/arm64.
2. The three `TypeEnv.collectFree*` walks fall below **10%** inclusive of the check in a re-run of the same `sample` profile.
3. A deterministic work-counter test asserts that bindings visited per application stay bounded as a synthetic module grows its top-level and import count (linear check, not quadratic).
4. Byte-identical `iface.json` and CoreTypeInfo artifacts before/after for `examples/`, `std/`, ailang_parse and crew_core.
5. Peak RSS within +10% of dev HEAD (263 MB on ailang_parse).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Memoise per-frame free-variable sets on `TypeEnv` (vs. level-based generalisation) | Level-based generalisation removes env scans entirely but rewrites `generalize`, `inferApp`'s local solve and every `baseEnv` window; memoisation is local to `env.go` plus call sites | agent (this doc: memoise; levels are Future Work) | design | low |
| Invalidate on `BindScheme`/`BindType` by a per-env generation counter | These are the only in-place binding writes (V7); a missed invalidation is a silent soundness bug | agent | design | med |
| Callers get read-only sets, not copies | Returning the cached map to callers that write into it would corrupt the cache; copying would keep an O(set) cost per call | agent | compile | low |

### Design Freeze

- [x] Memoisation over level-based generalisation (agent; reversible — levels remain Future Work)
- [x] Equivalence proof = shadow cross-check in tests + byte-identical artifacts (agent)
- [x] No new env var or CLI flag (simplicity gates); the cross-check is a test-only package hook

No human ratification required.

## Solution Design

### Overview

1. **Per-frame cache.** Each `TypeEnv` frame lazily computes and stores its own free-variable sets (type, row, effect-row) from its own bindings, and a **cumulative** set (own ∪ parent's cumulative). A frame's bindings never change after `Extend`/`ExtendScheme` except through `BindScheme`/`BindType`, which bump a generation counter that invalidates the frame and, via a recorded parent generation, every descendant cumulative set.
2. **Read-only query API.** A new `env.ownedVars()` returns an immutable `*envVarSets` (three sets plus a `Has(name)` lookup). `inferApp`'s `outerVars`, `envOwnedVars`, `generalize`, and `baseEnvFreeVars` use it directly. The existing `FreeTypeVars()`/`FreeRowVars()`/`FreeEffectRowVars()` keep their signatures and return copies, for external callers.
3. **Windowed walks reuse frame-own sets.** `effectRowsAbove`, `FreeTypeVarsAbove` and `implicitRowsAbove` iterate frames between `env` and `base` and union each frame's cached own set, instead of wrapping bindings in a throwaway `&TypeEnv{bindings: …}` that defeats caching (`typechecker_app_effects.go:187`, `env.go:110`).
4. **Measure by work, not wall clock.** A package counter of bindings visited by free-variable collection, readable from tests, backs a linearity test.

### Architecture

`internal/types/env.go`:

```go
type TypeEnv struct {
    bindings map[string]interface{}
    parent   *TypeEnv
    gen      uint64       // bumped by BindScheme/BindType
    own      *envVarSets  // free vars of this frame's bindings only; nil = stale
    chain    *envVarSets  // own ∪ parent.chain; nil = stale
    chainOf  uint64       // parent's chain generation this chain was built from
}

type envVarSets struct{ types, rows, effectRows map[string]bool; gen uint64 }
```

- `own` is computed once per frame (same traversal as today, including the scheme-quantifier subtraction and the `visited` cycle guard — see `.claude/rules/type-system.md`).
- `chain` is valid iff `own` is valid and `chainOf == parent.chain.gen`. Validating walks parent pointers only (O(depth), no type traversal), and in practice hits the first frame.
- `BindScheme`/`BindType` set `own = nil, chain = nil`, `gen++`. Descendants notice through `chainOf`.
- Single-threaded by construction (V9); no locking. If the pipeline ever checks modules concurrently against a shared frame, the cache must move behind `sync.Once`-style publication — flagged in Risks.

Call-site changes:

| Site | Today | After |
|---|---|---|
| `inferApp` outerVars (`typechecker_functions.go:561-570`) | 3 full walks + map copies per application | `outer := ctx.env.ownedVars()`; local-solve filter calls `outer.Has(v)` |
| `envOwnedVars` (`typechecker_effect_union.go:140`) | 3 full walks per lambda frame | returns `env.ownedVars()` |
| `generalize` (`inference_helpers.go:132,143`) | 2 full walks per let | cached chain sets |
| `baseEnvFreeVars` (`typechecker_core.go:460,595`) | 1 full walk per decl | cached chain set |
| `effectRowsAbove`, `FreeTypeVarsAbove`, `implicitRowsAbove` | per-frame re-traversal via temporary env wrappers | union of cached frame-own sets (`implicitRowsAbove` gets its own cached per-frame set, keyed the same way, since its result depends on `effectAliasHead`, which is fixed per checker) |

### Implementation Plan

**Phase 1: work counter + baseline (~0.5 day)**
- [ ] Add `envScanBindings` (package counter, incremented per binding visited in `collectFree*`) and a test helper to read/reset it.
- [ ] Synthetic fixture generator (in test): N top-level functions, M imported names, bodies with K nested applications. Record bindings-visited for (N, M, K) grids at dev HEAD — the "before" curve (expected ≈ K × (N + M + builtins)).
- [ ] Record the ailang_parse baseline in this doc's Verification Log: instructions retired, peak RSS, `sample` hot-path table.

**Phase 2: per-frame cache + read-only API (~1.5 days)**
- [ ] `envVarSets`, `own`/`chain`/`gen`, `ownedVars()`; `FreeTypeVars()` & co. become copies of the cached chain sets.
- [ ] Invalidation in `BindScheme`/`BindType`.
- [ ] Shadow cross-check: a test-only hook (`debugCrossCheckEnvVars`, set by `TestMain` in `internal/types` and `internal/pipeline`) recomputes every query uncached and panics on mismatch.
- [ ] Unit tests: cache hit after `Extend`; invalidation after `BindScheme` on the frame and on an ancestor; quantified vars still excluded; cyclic type guarded.

**Phase 3: move call sites (~1 day)**
- [ ] `inferApp`, `envOwnedVars`, `generalize`, `baseEnvFreeVars`, windowed walks (table above).
- [ ] Linearity test: bindings visited per application constant in N and M (allow ≤ depth of local scopes).
- [ ] Equivalence: byte-identical artifacts for `examples/` (`make verify-examples`), `std/`, ailang_parse, crew_core; full `make test`, `make test-core`, `go test -race ./internal/types/ ./internal/pipeline/`.

**Phase 4: measure + record (~0.5 day)**
- [ ] Re-run Phase 1's measurements; fill the after-columns; re-profile and record what is now hot (expected: substitution, handed back to the crew-core doc).
- [ ] Changelog fragment; update [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) Phase 1 to "absorbed by this doc".

### Files to Modify/Create

- `internal/types/env.go` — frame cache, generation, `ownedVars()`, copy-returning public API (~120 LOC)
- `internal/types/env_cache_test.go` — cache/invalidation/cross-check unit tests (new, ~150 LOC)
- `internal/types/typechecker_functions.go` — `inferApp` outerVars → `ownedVars()` (~15 LOC)
- `internal/types/typechecker_effect_union.go` — `envOwnedVars` → `ownedVars()` (~10 LOC)
- `internal/types/typechecker_app_effects.go` — windowed walks over cached frame sets (~30 LOC)
- `internal/types/inference_helpers.go` — `generalize` uses cached sets (~10 LOC)
- `internal/types/typechecker_core.go` — `baseEnvFreeVars` from cache (~5 LOC)
- `internal/pipeline/env_scan_linearity_test.go` — synthetic-module linearity test (new, ~120 LOC)

## Conflict Surface

This touches `internal/types/` — the shared machinery for every check.

1. **What does it extend?** The free-variable queries on `TypeEnv`. No syntactic or typing-rule change; every query must return the same set it returns today.
2. **What else uses those positions?**
   - The REPL mutates its persistent env with `BindScheme` (`internal/repl/repl_commands.go:387`, `repl_eval.go:193`) → invalidation path; REPL tests must pass.
   - Builtin seeding (`internal/link/env_seed.go:45`) uses `BindScheme` while building the root frame → nothing is cached yet; first query sees the final bindings.
   - Effect soundness (#1958): `unionFrame.outer` and `inferApp`'s outerVars decide which variables the application-local solve may bind. A wrong (smaller) set leaks a premature resolution into the enclosing scope ("same row variable with different extensions"); a larger set only blocks local solving. The cross-check catches both.
   - `effectRowsAbove(base == nil)` falls back to a full walk; keep that semantics.
   - Rows are mutated in place only on fresh clones (`cloneApplicationRow` at `typechecker_app_effects.go:174,316`, `typechecker_latent_data.go:121,141`), so a cached set cannot go stale through a type it was computed from (V8). The cross-check is the backstop if that ever changes.
3. **Disambiguation:** none needed — the cache is a pure memo of an existing function.
4. **Programs that must still work:** `examples/` via `make verify-examples` (47 files); `std/`; `sunholo/ailang_parse@c3b4fc6` (93 files) and `sunholo/crew_core` (wide records, deep lets — the #1811 shape); the #1958 regression tests (`internal/pipeline/effect_union_resolve_test.go`, `effect_ceiling_reachable_test.go`, `effect_row_open_closed_test.go`); REPL suite (`go test ./internal/repl/`).
5. **Intentional incompatibilities:** none. Any diff in compiled artifacts is a bug.

## Examples

### Example 1: what an application costs

Before, in a module importing 40 names with 120 top-level functions, each `f(x)` inside a body walks ~40 + 120 + builtins bindings × 3 kinds, traversing each type. A 2,000-application module does ~10⁶ type traversals of closed schemes.

After, the first application in a declaration computes the chain once (or hits frames already cached by the previous declaration); each subsequent application is a pointer comparison plus map lookups. The work counter shows the curve flatten.

### Example 2: the publisher's view

`ailang publish` of ailang_parse: validator check ~190 s → target ≤ ~95 s, giving the package headroom to grow instead of trimming to fit the cap.

## Success Criteria

- [ ] Instructions retired on ailang_parse cold check ≤ 0.80×10¹² (from 1.61×10¹²)
- [ ] `collectFree*` < 10% inclusive in the re-profile
- [ ] Linearity test green; it fails on today's code (mutation test: revert `inferApp` to the full walk)
- [ ] Byte-identical artifacts on the four corpora; shadow cross-check green across `make test`
- [ ] Peak RSS ≤ +10%
- [ ] `make test`, `make test-core`, `make verify-examples`, `go test -race` on `types`/`pipeline` green
- [ ] Changelog fragment; crew-core doc updated

## Testing Strategy

- **Unit (`env_cache_test.go`):** hit/miss, invalidation on frame and ancestor, quantified-variable exclusion, cycle guard, copy-on-return isolation (mutating a returned `FreeTypeVars()` map does not change the next result).
- **Shadow cross-check:** every cached query is recomputed uncached under tests in `internal/types` and `internal/pipeline`; any mismatch panics with the frame and kind.
- **Linearity (`internal/pipeline`):** bindings visited per application vs. N top-level and M imports; assert slope ≈ 0 in N and M. Work counts, not wall clock (wall clock and RSS are machine-dependent).
- **Equivalence:** byte-identical `iface.json` + CoreTypeInfo across corpora before/after (the crew-core doc's strongest available proof that only cost changed).
- **Real-world:** instructions retired + `sample` profile on ailang_parse, recorded in the Verification Log.

## Deferred Decisions

- Whether `implicitRowsAbove`'s per-frame cache is worth it, if the re-profile shows it under 2% — agent.
- Whether to also cache `freeTypeVars(typ)` for scheme bodies across frames (hash-consing) — agent, only if the re-profile shows it.
- Exact shape of the read-only API (struct vs. interface) — agent.

## Non-Goals

- **Substitution cost** (`safeSubstitute`/`ApplySubstitution`, ~25% today): stays with [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) Phase 2.
- **Level-based generalisation** (Rémy levels): removes env scans by construction but is a type-checker rewrite; Future Work if memoisation misses the target.
- **Validator resources, wall cap or async validation**: infra lane; revisit only after this lands and is measured. A `check_wall_ms` metric from [m-validator-check-budget-parity](../v0_56_1/m-validator-check-budget-parity.md) will show the effect in production.
- **A `--cpuprofile` flag on `check`**: Go benchmarks with `go test -cpuprofile` cover profiling without new CLI surface.
- **Parallel module checking.**

## Timeline

- Day 1: Phase 1 (counter, synthetic fixtures, baseline) + start Phase 2
- Day 2: Phase 2 (cache, invalidation, cross-check, unit tests)
- Day 3: Phase 3 (call sites, linearity, equivalence)
- Day 4: Phase 4 (measure, record, docs) + contingency

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| A missed invalidation (new in-place write added later) | Silent unsound generalisation or local-solve leak | Shadow cross-check runs in all `types`/`pipeline` tests; `BindScheme`/`BindType` are the only writers today (V7); a comment on `bindings` names the rule |
| A caller mutates a returned cached set | Cache corruption | Public functions return copies; internal callers get the read-only `envVarSets` |
| Gain smaller than profiled (walks overlap; substitution becomes dominant) | Target missed | Target is 50%, not the 70% the profile suggests; re-profile hands the remainder to the crew-core doc |
| Future concurrent checking shares frames | Data race | `-race` in the gate; Architecture note says publish via `sync.Once` if that changes |
| Memory growth from cached sets | RSS regression | Sets hold only free vars (small; closed schemes contribute none); RSS ≤ +10% criterion |

## Related Documents

- [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) — cold-compile time on crew_core (v0.54.6 profile: 62% substitution, 8.9% env walk). Its Phase 1 env-walk item moves here; its substitution work stays there. The two profiles differ because ailang_parse is application-heavy with many imports, and because `inferApp`'s outer snapshot post-dates that profile in its current form.
- [m-validator-check-budget-parity](../v0_56_1/m-validator-check-budget-parity.md) — validator pacing, refusal messages and per-publish `check_wall_ms` telemetry (sprint approved 2026-11-03).
- [wasm-typecheck-step-ceiling-662](../ailang-core-triage/wasm-typecheck-step-ceiling-662.md) — the browser/WASM typecheck step ceiling; the same cost growth hits it.

## References

- Profiles: macOS `sample` of `ailang check --package .` on ailang_parse `c3b4fc6`, v0.55.0 (installed binary) and dev HEAD (built from `8aae2f819`+), 2026-11-03
- Validator request logs (Cloud Run `ailang-registry-validator`, project `ailang-registry`), 2026-10-26..11-02
- #1958 (effect soundness re-land), #1811 (constraint multiplication), #1134 (IFC cross-module)

## Verification Log

| # | Claim | Evidence | Status |
|---|-------|----------|--------|
| V1 | 98–99% of a cold ailang_parse check is `typeCheckAndLowerModule` | `sample` call-tree inclusive counts: 2679/2697 (v0.55.0), 2936/2990 (dev) under `check.CheckPackageFiles` | Confirmed |
| V2 | `inferApp` walks the whole env for type, row and effect-row vars on every application | Code: `typechecker_functions.go:545` (func), `:561-570` (three walks); profile callers: `inferApp → collectFreeEffectRowVars` 47.6%, `→ collectFreeRowVars` 5.0%, `→ collectFreeTypeVars` 4.8% (dev) | Confirmed |
| V3 | The walks recurse to the root frame | Code: `env.go:135-137`, `env.go:216` (`env.parent.collect…`) | Confirmed |
| V4 | `envOwnedVars` is new in dev, not in v0.55.0 | `git log -S "func envOwnedVars"` → `d555a52ac` (2026-11-02); absent from the v0.55.0 profile | Confirmed |
| V5 | dev HEAD does 16% more work than v0.55.0 on ailang_parse | `AILANG_NO_CACHE=1 /usr/bin/time -l … check --package .`: 1,384,655,335,625 vs 1,606,622,530,900 instructions; RSS 236 → 263 MB | Confirmed |
| V6 | Windowed walks defeat caching via temporary env wrappers | Code: `typechecker_app_effects.go:187` `(&TypeEnv{bindings: e.bindings})`, `env.go:110` | Confirmed |
| V7 | `BindScheme`/`BindType` are the only in-place binding writes | `grep -rn '\.bindings\[' internal/types/*.go` → only `env.go:77,84` outside `Extend*`; callers `internal/repl/repl_commands.go:387`, `repl_eval.go:193`, `internal/link/env_seed.go:45` | Confirmed |
| V8 | In-place row mutations act on fresh clones | Code: `typechecker_app_effects.go:172-174,314-316` (`cloneApplicationRow` then `Tail = nil`), `typechecker_latent_data.go:119-121` (clone), `:139-141` (`row` from `latentRow`, a clone, then `zonk` = `zonkApplicationRow`, which clones on entry and returns only through `cloneApplicationRow`, `typechecker_app_effects.go:293-308`) | Confirmed |
| V9 | Type checking is single-threaded; builtin env is fresh per checker | `grep "go func\|errgroup\|WaitGroup" internal/pipeline internal/types` → only `metrics.go:267` (telemetry POST); `env_seed.go` `newTypeEnvWithBuiltinsImpl` builds a new env per call | Confirmed |
| V10 | Check is effectively single-threaded in time too | v0.55.0 cold: 118.2 s user / 106.3 s real (loaded Mac); publisher's earlier 113.9 s user / 65.8 s real (≈1.7×, mostly GC workers) | Confirmed |
| V11 | Validator cap 200 s, request timeout 300 s, client timeout 5 min | `cmd/registry-validator/bounded.go` `checkWallCap`; terraform-registry `timeout = "300s"`; `cmd/ailang/pkg_publish.go:261` | Confirmed |
| V12 | No existing `--cpuprofile` on `check` | `ailang -cpuprofile … check` → "flag provided but not defined" (v0.55.0); flags registered in `cmd/ailang/main_run.go:138-139` for `run` | Confirmed |
| V13 | crew-core doc's Phase 1 targets the same walk, unshipped | `m-crew-core-cold-compile-followup.md` rows V4/V7 and Phase 1; no sprint JSON for it in `.ailang/state/sprints/`; all Phase checkboxes open | Confirmed |

**Quorum:** none of the four triggers fires (no design-freeze items for a human, no shared machinery overridden — the cache is a memo of existing queries proven equal by cross-check, no cost/KPI/schema surface, all premises in-repo). Skipped.

## Future Work

- Level-based generalisation, if memoisation misses the 50% target or substitution work makes it the cleaner fix.
- Fold the bindings-visited counter into `--debug-compile` output so publishers can see check work per module.

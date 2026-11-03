# Sprint Plan: M-TYPECHECK-ENV-SCAN-COST

**Design doc:** [m-typecheck-env-scan-cost.md](m-typecheck-env-scan-cost.md)
**Sprint JSON:** `.ailang/state/sprints/sprint_M-TYPECHECK-ENV-SCAN-COST.json`
**Duration:** 3–4 days (4 milestones, each sized for one 60–90 min executor run plus validation)
**Estimated:** ~680 LOC (≈ 260 implementation, 340 tests, 80 docs)
**Risk:** Medium — touches the type checker's shared environment; mitigated by a shadow cross-check and byte-identical artifact gates.

## Goal

Make the "free variables of the enclosing environment" query cost O(new bindings) amortised instead of O(whole environment) per call, so a cold `ailang check --package .` of `sunholo/ailang_parse` retires ≤ 0.80×10¹² instructions (from 1.61×10¹² at dev HEAD), with byte-identical compile output.

## Current status

- Root cause is located and verified (design doc V1–V13): `inferApp` (`internal/types/typechecker_functions.go:561-570`) walks every frame to the root three times per application; `envOwnedVars`, `generalize`, `baseEnvFreeVars` and the windowed walks repeat it at coarser grain.
- Nothing implemented. The overlapping Phase 1 of `m-crew-core-cold-compile-followup` never ran (no sprint JSON).
- Velocity: the 7-day history is dominated by the #1958 type-checker re-land in the same package (M1–M4 in about a day, 2026-11-02). Using a conservative ~200 LOC/day including tests.

## Rulings and human-decision gates

No open human decisions. The design doc's Design Freeze is fully agent-ruled (memoisation over levels; cross-check + byte-identical artifacts as the equivalence proof; no new env var or CLI flag). Copy these as rulings into execution evidence.

## Registry reuse audit

| Milestone | Decision | Reason |
|---|---|---|
| M1–M4 | none | Internal compiler work in `internal/types`; no package capability involved |

## AILANG syntax gate

No milestone writes `.ail` source. Synthetic fixtures are generated as `.ail` text inside Go tests; each generator must be checked with `ailang check` on one emitted sample before the test relies on it.

## File-size watch

`internal/types/typechecker_core.go` is at 796 lines and `typechecker_functions.go` at 779 (limit 800). Changes there must be net-neutral or negative. If either would grow, put the new code in a companion `internal/types/env_owned_vars.go`. `make check-file-sizes` runs at every milestone.

## Milestones

### M1_WORK_COUNTER_BASELINE (~150 LOC)

Deterministic work counter and the "before" measurements.

- `internal/types/env.go`: package counter `envScanBindings` incremented once per binding visited by `collectFreeTypeVars`, `collectFreeRowVars` and `collectFreeEffectRowVars`; test helpers to reset/read it (test-only file `env_scan_counter_test_helpers.go` or exported via `export_test.go`).
- `internal/pipeline/env_scan_linearity_test.go`: generator for synthetic modules with N top-level functions, M imported names (from a generated sibling module), K applications per body; record bindings-visited for a small grid. At this milestone the test **logs** the curve and asserts nothing about slope (`t.Log`), so it passes on today's code.
- Record the ailang_parse baseline in the design doc's Verification Log (new rows): instructions retired and peak RSS from `AILANG_NO_CACHE=1 /usr/bin/time -l ailang check --package .` at `~/dev/sunholo-data/ailang-parse` (`c3b4fc6`), and crew_core 0.6.0 from `~/.ailang/cache/registry/sunholo/crew_core/0.6.0`, using a binary built at the sprint's base commit.
- Build that base binary into a scratch path (never over `~/go/bin/ailang`) and keep it for M4's artifact diff.

**Acceptance:**
- Counter increments exactly once per binding visited (unit test on a 3-frame env with known bindings).
- Linearity test runs and logs bindings-visited for (N, M, K) ∈ {(10,10,10), (40,10,10), (10,40,10), (10,10,40)}; the log shows growth in N and M (the bug, measured).
- Baseline rows for ailang_parse and crew_core recorded in the design doc.
- `make test-changed`, `make check-file-sizes` green.

### M2_FRAME_CACHE (~260 LOC)

Per-frame memoised free-variable sets with generation invalidation, plus the shadow cross-check.

- `internal/types/env.go`: `envVarSets` (types, rows, effectRows, gen); `TypeEnv` gains `gen`, `own`, `chain`, `chainOf`; `ownedVars()` returns the cached cumulative set (read-only); `FreeTypeVars()`, `FreeRowVars()`, `FreeEffectRowVars()` return copies of it. `BindScheme`/`BindType` clear `own`/`chain` and bump `gen`. Keep the existing quantifier subtraction and `visited` cycle guard exactly.
- Shadow cross-check: package var `debugCrossCheckEnvVars` (test-only, set in `TestMain` of `internal/types` and `internal/pipeline`; no env var). When set, every cached query is recomputed uncached and panics on mismatch with frame depth and kind.
- `internal/types/env_cache_test.go`: hit after `Extend`/`ExtendScheme`; invalidation after `BindScheme` on the frame **and** on an ancestor (descendant sees the new binding's free vars); scheme-quantified vars excluded; cyclic type terminates; mutating a map returned by `FreeTypeVars()` does not change the next call's result.

**Acceptance:**
- All `env_cache_test.go` cases pass; each fails when its mechanism is removed (mutation-test the invalidation and the copy-on-return by editing a scratchpad copy, never `git checkout`).
- Cross-check enabled across `internal/types` and `internal/pipeline` test runs; `make test-changed` green with it on.
- `go test -race ./internal/types/ ./internal/pipeline/` green.
- No behaviour change yet: call sites untouched.

### M3_CALL_SITES_LINEAR (~150 LOC)

Move the hot callers onto the cache and make the linearity test assert.

- `typechecker_functions.go:561-570` (`inferApp` outerVars) → `outer := ctx.env.ownedVars()`; local-solve filter uses `outer.Has(v)`. Net line change ≤ 0.
- `typechecker_effect_union.go` `envOwnedVars` → `env.ownedVars()` (keep the function if other callers need a `map[string]bool`, else inline).
- `inference_helpers.go` `generalize` (`:132,143`) → cached chain sets.
- `typechecker_core.go:460,595` `baseEnvFreeVars` → cached chain set (net ≤ 0 lines).
- `effectRowsAbove`, `FreeTypeVarsAbove`, `implicitRowsAbove`: union cached per-frame own sets instead of `&TypeEnv{bindings: …}` wrappers. `effectRowsAbove(base == nil)` keeps full-chain semantics.
- Linearity test now **asserts**: bindings visited per application does not grow with N or M (allow growth with local scope depth only). Mutation test: temporarily restore the full walk in `inferApp` (scratchpad copy) and confirm the assertion fails.

**Acceptance:**
- Linearity assertion green, and red under the mutation.
- #1958 regression files green: `internal/pipeline/effect_union_resolve_test.go`, `effect_ceiling_reachable_test.go`, `effect_row_open_closed_test.go`.
- `go test ./internal/repl/` green (the `BindScheme` path).
- `make test-changed`, `make verify-examples`, `make check-file-sizes`, `make lint` green.

### M4_EQUIVALENCE_MEASURE_RECORD (~120 LOC, mostly docs)

Prove only cost changed; measure; record.

- Byte-identical artifacts: for each corpus — `examples/`, `std/`, ailang_parse (`c3b4fc6`), crew_core 0.6.0 — run `check` with the M1 base binary and the new binary into separate `AILANG_CACHE_DIR`s, then `diff -r` the `compile/` trees (iface and CoreTypeInfo artifacts). Any difference is a bug, not a waiver.
- Re-measure ailang_parse and crew_core (instructions retired, peak RSS) and re-profile ailang_parse with macOS `sample` (60 s window); record the after-columns and the new hot-path table in the design doc's Verification Log.
- Changelog fragment `changelogs/unreleased/2026-11-0X-typecheck-env-scan-cost.md` (`### Changed — …`), with before/after numbers.
- `m-crew-core-cold-compile-followup.md`: mark its Phase 1 env-walk item "absorbed by M-TYPECHECK-ENV-SCAN-COST (link)", leave Phase 2 (substitution) open, and paste the new profile's substitution share as its fresh baseline.

**Acceptance:**
- `diff -r` empty on all four corpora.
- ailang_parse instructions retired ≤ 0.80×10¹²; `TypeEnv.collectFree*` < 10% inclusive in the re-profile.
- Peak RSS ≤ +10% vs the M1 baseline on both packages.
- Changelog fragment present; `make check-changelog` green.
- If the 0.80×10¹² target is missed: record the measured number and the new top three hot paths, and STOP — do not widen scope into substitution work (that belongs to the crew-core doc).

## Day-by-day

- **Day 1:** M1 (counter, generator, baselines, base binary), start M2.
- **Day 2:** M2 (cache, invalidation, cross-check, unit + mutation tests).
- **Day 3:** M3 (call sites, linearity assertion + mutation, #1958/REPL gates).
- **Day 4:** M4 (artifact diff, measurements, profile, docs) + contingency.

## Validation (every milestone)

`make test-changed` (touched packages + `make test-core`), `make check-file-sizes`, `make lint`; plus `make verify-examples` from M3 onward and `make check-changelog` at M4. Do not run `make test` as a gate; CI runs it.

## Commits

One commit per milestone on `dev` (attended) with `refs #1958` where the change touches its code paths; message names the milestone ID.

## Handoff

sprint-executor: start at M1. The design doc carries the verified premises (V1–V13); re-verify line numbers at the base commit before editing, since `internal/types` changed daily last week.

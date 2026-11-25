# M-EMBEDDED-STDLIB-CACHE-DEP-KEYS: verify actual transitive dependency identity and freshness in the embedded-stdlib cache test

**Status**: Planned
**Target**: v0.61.2
**Priority**: P0 (a focused test is red at origin/dev HEAD, and CI run 45644929576 fails it on both Linux and Windows)
**Estimated**: 0.5–1 day (test-only)
**Dependencies**: None. Blocks nothing; unblocks a green `internal/pipeline` suite at dev HEAD.
**Reporter**: triage iteration 57 (sunholo-data/ailang issue #2453, actions run 45644929576). OpenAI verifier and an independent MiniMax judge concurred on the diagnosis and scope.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Test-only change; no production semantics move. Scores reflect what the repaired test *asserts*.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The repaired test pins the deterministic key derivation (`ModuleCacheKey` over the loader snapshot + actual closure digests) instead of a stale frozen expectation |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect-surface changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Strengthens local, in-package verification of the cache provenance invariant |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | 0 | No prompt/syntax-surface changes |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | Reuses existing in-package helpers (`cacheDepDigest`, `unmarshalIfaceFull`) rather than new machinery |
| A11: Structured Failure | +1 | Failure messages name the module and both keys, so a future drift is diagnosable from the test output alone |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

`TestCachePipeline_EmbeddedKeys` fails at origin/dev `e3595b4f` in a focused run and in CI (actions run 45644929576, Linux and Windows); the immediately parent job is green, so the commit is the cause, not a flake.

**Diagnosis (verified by reading the code paths, see Premise Verification Log):**

1. `e3595b4f` adds `std/result.ail` with `import std/option (Option, Some, None)` (the new `toOption`/`fromOption` conversions need the `Option` ADT).
2. The loader puts the file's declared imports on `LoadedModule.Imports` (`internal/loader/loader.go:334–368`); std/ modules get **no** prelude injection (`internal/loader/prelude_imports.go`, `preludeEligible` excludes `std/`), so embedded `std/result` now loads with `Imports = ["std/option"]`.
3. `assertEmbeddedModuleKey` (`internal/pipeline/cache_invalidation_test.go:215–217`) asserts `len(mod.Imports) != 0` → fatal `std/result unexpected dependencies: [std/option]`. This assertion encodes a frozen world ("embedded stdlib modules import nothing") that stopped being true.
4. Behind it, a second stale expectation: the test computes `want := ModuleCacheKey(identity, src, map[string]string{})` — an **empty** dependency-digest map — while `prepareCacheLookup` (`internal/pipeline/pipeline_module_cache.go:49–57`) folds the whole transitive closure (`st.depClosure`, `internal/pipeline/cache_dep_closure.go`) into the key: for `std/result` that is `{"std/option": cacheDepDigest(optionIface)}`. The manifest key would not equal the empty-dep expectation even if the import-count assertion were removed.

**The production invariant is intact.** `prepareCacheLookup` + `cache_dep_closure.go` (M-TYPE-NAME-SHADOW M3) already key each module on its loader-owned source snapshot **plus** the digests (+ alias digests) of its real transitive closure, and std/ modules add no `$effects.max` / `$test-bodies` components (`isOwnPackageModule` excludes `std/`; `checksTestBodies` is root-only). Nothing in `internal/pipeline` production code is wrong. **Only the test's expectations are stale**, and they are stale in the dangerous direction: they would have masked a key that *ignored* its dependencies.

**Impact:**
- `internal/pipeline` CI is red at dev HEAD on both CI platforms; any merge stacking on `e3595b4f` inherits the failure.
- The test's real purpose — proving the **cache provenance invariant** for embedded stdlib modules (embedded source path, retained non-empty loader snapshot, key bound to that snapshot, runtime result not copying snapshots) — is unreachable while the test dies on a stale premise.

## Goals

**Primary Goal:** Repair `TestCachePipeline_EmbeddedKeys` so it verifies the **actual** transitive dependency identity and freshness of embedded-stdlib cache keys — not a frozen zero-dependency world — while retaining every existing cache provenance assertion.

**Success Metrics:**
1. `TestCachePipeline_EmbeddedKeys` passes at `e3595b4f`+ in focused and full-suite runs, on Linux and Windows.
2. The test proves `std/result`'s manifest key equals `ModuleCacheKey` computed with the **actual** `std/option` dependency digest of the same run (not an empty map).
3. A new bounded subtest proves an edit to a **stdlib-resolved** dependency changes the dependent's manifest key while the dependent's own source is unchanged.
4. Zero production-code changes; blast radius is one test file.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Reconstruct the expected dep digest from the same run's manifest (`Entries["std/option"].IfaceJSON` → `unmarshalIfaceFull` → `cacheDepDigest`) instead of a differential-only "≠ empty-dep key" assertion | Differential-only proves participation but not that the digest is the *actual* current dependency's; reconstruction pins the full key contract | agent | design | low |
| D2: Include the stdlib-edit freshness subtest (Leg 3) in this scope, vs. relying on existing generic coverage (`TestExplicitEmptyCacheDependencyEdits` transitive arm, `TestCacheKey_InvalidatesOnSourceEdit`) | Generic coverage uses **cwd file** deps; the stdlib *resolution path* (embedded / `AILANG_STDLIB_PATH` root) has no freshness test; this intake is explicitly a scope review | **human** | design | low |
| D3: Hardcode the per-module expected-import table (`std/option` → none, `std/result` → `["std/option"]`) with a maintenance comment, vs. deriving expectations by re-parsing | Deriving from the module's own AST asserts the loader against itself (circular); a hardcoded table is an independent expectation, matching the test's existing hardcoded `wantPath` style | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D2 scope ratification:** a human confirms whether the stdlib-edit freshness subtest (Leg 3, ~40 LOC) is inside this fix or deferred. This document plans **with** it included (recommended: it is the only freshness coverage for stdlib-resolved deps); the sprint drops it only by explicit human instruction.
- [ ] D1 and D3 are agent decisions taken at design time; objection at quorum reopens them.

## Solution Design

### Overview

Rewrite `assertEmbeddedModuleKey` from "one shape fits all modules" into a per-module table-driven assertion with two key arms (leaf vs. dependent), and add one focused subtest for stdlib-path dependency freshness. No production file changes.

### Architecture

**Leg 1 — actual dependency identity (repairs the first fatal):**
Replace the `len(mod.Imports) != 0` assertion with an expected-imports table:

```go
wantImports := map[string][]string{
    "std/option": nil,            // leaf: no declared imports
    "std/result": {"std/option"}, // e3595b4f: conversions need the Option ADT
}
```

Assert `mod.Imports` equals the table entry (order-insensitive / sorted compare, since loader order is source order but the contract is set equality). A comment states the maintenance rule: *this table records actual stdlib topology; a failure here means the stdlib grew a dependency — update the table AND re-review the key expectations, do not relax the assertion.*

**Leg 2 — actual key composition (repairs the second stale expectation; retains provenance):**

- **Leaf arm (`std/option`, empty closure):** keep the exact assertion unchanged — `entry.CacheKey == ModuleCacheKey(identity, *mod.SourceContent, map[string]string{})`. This proves the empty-dep arm still composes the key from the snapshot alone.
- **Dependent arm (`std/result`, closure `{std/option}`):** reconstruct the digest map the pipeline actually used, from the same run's manifest:

```go
optEntry := manifest.Entries["std/option"]
optIface, err := unmarshalIfaceFull(optEntry.IfaceJSON) // in-package helper, cache_store.go:298
wantDeps := map[string]string{"std/option": cacheDepDigest(optIface)}
want := ModuleCacheKey(identity, *mod.SourceContent, wantDeps)
```

  Assert `entry.CacheKey == want` (identity: the key carries the **actual** current `std/option` digest), **and** assert `entry.CacheKey != ModuleCacheKey(identity, *mod.SourceContent, map[string]string{})` (participation: the dependency digest is load-bearing — a distinct message so a future regression reads as "dep dropped out of the key", not as an opaque mismatch). This mirrors `prepareCacheLookup` without duplicating any compile logic: `ifaceFullJSON` round-trips `Digest`, `TypeAliases`, and `AliasParams` (`cache_store.go:182–196`), so `cacheDepDigest` over the decoded iface equals the digest computed in-run.

**Retained provenance assertions (unchanged, the invariant this test exists for):**
- source path is `<embedded>/std/<moduleID>.ail` (matches `stdlibroot.Root.DisplayPath`, root.go:53–58);
- retained non-empty `SourceContent` snapshot;
- `entry.CacheKey` ≠ empty-source key;
- `result.Modules[moduleID].SourceContent == nil` (the runtime result did not copy the snapshot).

**Leg 3 — freshness through the stdlib resolution path (new subtest):**

1. Copy the repository `std/` tree (at minimum `io.ail` — the stdlib-root marker, `stdlibroot.Marker` — plus `option.ail`, `result.ail`, `VERSION`, `STABILITY.toml`) into a temp dir; set `AILANG_STDLIB_PATH` to it (`internal/config/paths.go:7`, root `Source: "env"`).
2. In a sibling temp cwd write a `main.ail` with no imports (the prelude pulls `std/option` + `std/result`); compile with `ModeCheck` and a temp cache dir; record `manifest["std/option"]` and `manifest["std/result"]` keys.
3. Edit **only** the temp `option.ail` (append one exported `pure func`, leaving `Option`, `Some`, `None` intact so `std/result` still compiles); recompile in a **fresh** cache dir.
4. Assert: compile succeeds; `std/option`'s key changed; **`std/result`'s key changed although its own source bytes are identical** — transitive freshness through the stdlib resolution path. (Warm-reuse behavior after the edit is already covered by `cacheEditAndVerifiedWarmHit` and is deliberately not re-tested here.)

### Implementation Plan

**Phase 1: repair the red test (~2–3 hours)**
- [ ] Restructure `TestCachePipeline_EmbeddedKeys` + `assertEmbeddedModuleKey` per Legs 1–2 (expected-imports table; leaf/dependent key arms; retained provenance block).
- [ ] Add the participation inequality with its own diagnostic message.
- [ ] Run the focused test green; run `go test ./internal/pipeline/` green.

**Phase 2: stdlib-path freshness subtest (~2–3 hours)**
- [ ] Add the Leg 3 subtest (temp stdlib copy via `AILANG_STDLIB_PATH`, edit `option.ail`, assert both keys change).
- [ ] Confirm it fails if `prepareCacheLookup` were to drop closure digests (mutation check: comment out the `depClosure` loop, watch `std/result`'s key-unchanged assertion fire).

**Phase 3: wrap-up (~1 hour)**
- [ ] `make fmt`, `make lint`, `make test` (or the CI-equivalent focused set on platforms without a full toolchain).
- [ ] Confirm zero non-test diffs: `git diff --stat` touches only `internal/pipeline/cache_invalidation_test.go`.

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `internal/pipeline/cache_invalidation_test.go` — rewrite `assertEmbeddedModuleKey` (~60 LOC changed), extend `TestCachePipeline_EmbeddedKeys` (~30 LOC), add one subtest (~40 LOC). Net ≈ +90 LOC, all test code.

## Examples

### Example 1: the failing assertion pair, before and after

**Before (stale, fails at `e3595b4f`):**
```go
if len(mod.Imports) != 0 {
    t.Fatalf("%s unexpected dependencies: %v", moduleID, mod.Imports) // fatal: std/result imports std/option
}
want := ModuleCacheKey(testCompilerIdentity(t), *mod.SourceContent, map[string]string{}) // stale: pipeline folds closure digests
if entry.CacheKey != want { t.Fatalf(...) }
```

**After (actual identity + freshness, provenance retained):**
```go
if want := wantImports[moduleID]; !sameImportSet(mod.Imports, want) {
    t.Fatalf("%s imports = %v, want actual declared set %v (update the table when stdlib topology changes)",
        moduleID, mod.Imports, want)
}
wantDeps := actualDepDigests(t, manifest, moduleID) // {} for std/option; {"std/option": cacheDepDigest(...)} for std/result
want := ModuleCacheKey(testCompilerIdentity(t), *mod.SourceContent, wantDeps)
if entry.CacheKey != want {
    t.Fatalf("%s key = %q, want actual-closure key %q", moduleID, entry.CacheKey, want)
}
if noDeps := ModuleCacheKey(testCompilerIdentity(t), *mod.SourceContent, map[string]string{}); moduleID == "std/result" && entry.CacheKey == noDeps {
    t.Fatalf("%s key equals the dependency-free key: the std/option digest dropped out of the key", moduleID)
}
```

### Example 2: the freshness subtest shape
```
temp stdlib root (copy of std/, marker io.ail present) ── AILANG_STDLIB_PATH
  run 1: compile main.ail → manifest keys {option: K1a, result: K1b}
  edit option.ail (add export pure func pinch[a](o: Option[a]) -> Option[a] ...)
  run 2: fresh cache dir → manifest keys {option: K2a, result: K2b}
  assert: compile ok; K2a != K1a; K2b != K1b  ← std/result source unchanged in both runs
```

## Success Criteria

- [ ] `TestCachePipeline_EmbeddedKeys` passes focused and inside `go test ./internal/pipeline/` (acceptance: green run at dev HEAD `e3595b4f`+).
- [ ] The test asserts `std/result`'s manifest key equals `ModuleCacheKey` with the run's actual `std/option` dep digest, and `std/option`'s key equals the empty-dep key (acceptance: both arms in the test body).
- [ ] All provenance assertions retained: embedded path, non-empty snapshot, key ≠ empty-source key, runtime result carries no snapshot (acceptance: unchanged block present).
- [ ] Leg 3 subtest present and green: editing a stdlib-resolved dep changes the dependent's key with the dependent's source unchanged (acceptance: subtest green; mutation check in Phase 2).
- [ ] All tests passing (`make test` or CI-equivalent).
- [ ] No production-code changes (`git diff` touches only the one test file).
- [ ] Engine-parity note: not applicable — no engine/codegen change; both evaluator and strict VM are untouched.

## Testing Strategy

**Unit tests:**
- The repaired `TestCachePipeline_EmbeddedKeys` (Legs 1–2) and the new freshness subtest (Leg 3) are themselves the deliverable; they run on every platform CI covers (Linux + Windows both failed pre-fix; both must pass post-fix — the test uses `filepath.ToSlash` for path assertions, so no platform-specific handling is added).

**Integration tests:**
- None new beyond the subtest, which is an integration-shaped pipeline run (load → key → manifest) inside the unit suite, matching the file's existing style (`runModuleWithCacheDependencies` + `readCacheManifest`).

**Manual testing:**
- Mutation audit only (Phase 2): revert the `depClosure` fold in a scratch build and confirm the new assertions fail; restore.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact shape of the import-set comparison helper (sorted slices vs. set map) — agent may choose, so long as the failure message prints both sides.
- Whether `actualDepDigests` reconstruction lives in the test file or a tiny test helper — agent may choose (test file preferred, ~15 LOC).
- Which appended export the freshness subtest adds to `option.ail` — agent may choose any compile-clean addition that preserves `Option`, `Some`, `None`.
- Whether the expected-imports table lives inline in `TestCachePipeline_EmbeddedKeys` or at file scope — agent may choose.

## Non-Goals

**Not attempted in this feature:**
- Any production change to `prepareCacheLookup`, `cache_dep_closure.go`, `cache_key.go`, or the loader — the production invariant is verified intact; touching it is out of scope and would convert a test repair into a semantics change requiring its own gate.
- PATH-dependency (`pkg/...`) cache invalidation — a separate, known triage topic with its own doc ([compile-cache-path-dep-invalidation](../ailang-core-triage/compile-cache-path-dep-invalidation.md)).
- Extending the embedded-key test to more stdlib modules than the prelude pulls (`std/option`, `std/result`) — the table is data, future topology changes update it, they do not grow the mechanism.
- Quorum dispatch or sprint execution — this intake is bounded to the design document.

## Timeline

**Week 1** (0.5–1 day total):
- Phase 1: repair the red test (Legs 1–2)
- Phase 2: freshness subtest (Leg 3) + mutation audit
- Phase 3: fmt/lint/test, diff hygiene

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Manifest `IfaceJSON` decode (`unmarshalIfaceFull`) drifts from the in-memory iface so the reconstructed digest mismatches | Med | The full-JSON schema is the store's own round-trip (`cache_store.go:298`, schema-checked on decode); if it drifts, the *warm-hit path itself* is broken and the test failing is correct signal — the failure message names both keys |
| Hardcoded import table goes stale again at the next stdlib dependency addition | Low | That staleness is now *legitimate signal* (topology changed → human reviews the new edge); the maintenance comment makes the update a one-line table edit, and the key arm automatically covers any closure the table records |
| Leg 3 copies `std/` from the source tree, coupling the test to repo layout | Low | Established pattern (`internal/stdlib/integration_test.go` already references `../../std/*.ail`); copy is read-only and the test skips nothing silently — a missing tree is a hard error, per the no-silent-fallbacks principle |
| Windows path separators in the freshness subtest | Low | Reuse the file's existing `filepath.ToSlash` convention for any path assertion |

## Related Documents

**Implemented (may inform design):**
- [m-compile-cache-dirty-build-key](../implemented/v0_43_2/m-compile-cache-dirty-build-key.md) — why the key carries a per-build fingerprint (the `testCompilerIdentity` helper both tests rely on)
- [m-verify-stdlib-stale-path](../implemented/v0_29_0/m-verify-stdlib-stale-path.md) — stdlib path resolution background

**Planned (check for overlap):**
- [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md) — M3 introduced the transitive-closure digest fold this test now asserts end-to-end (distinct: production mechanism vs. its embedded-stdlib verification)
- [compile-cache-path-dep-invalidation](../ailang-core-triage/compile-cache-path-dep-invalidation.md) — the *known open* gap for PATH deps; explicitly a Non-Goal here

## Premise Verification Log

Every load-bearing claim, with the command/read that proves it:

| # | Claim | Verification |
|---|-------|--------------|
| 1 | `e3595b4f` adds `std/result.ail` importing `std/option` | `git show e3595b4f -- std/result.ail` — new file, line `import std/option (Option, Some, None)` |
| 2 | Embedded stdlib is the working-tree `std/*.ail` (`//go:embed`) | `std/embed.go:10` — `//go:embed *.ail ai/*.ail stream/*.ail` |
| 3 | `LoadedModule.Imports` = declared file imports; std/ modules get no prelude injection | `internal/loader/loader.go:334–368` (`extractImportsFor` → `injectPreludeImports` → `Imports:`); `internal/loader/prelude_imports.go` `preludeEligible` returns false for `std/` IDs |
| 4 | The failing assertion is `len(mod.Imports) != 0` at `cache_invalidation_test.go` | Read `assertEmbeddedModuleKey` (lines 198–240): imports check precedes the key check |
| 5 | `prepareCacheLookup` folds the transitive closure into `depDigests`, so the empty-map expected key is stale | `internal/pipeline/pipeline_module_cache.go:49–57` + `cache_dep_closure.go` (`depClosure`, `cacheDepDigest = Digest + "+" + aliasDigest`) |
| 6 | std/ modules add no `$effects.max` / `$test-bodies` key components | `internal/pipeline/effect_ceiling.go:60–84` `isOwnPackageModule` excludes `std/`; `pipeline_module_phases.go:375` `checksTestBodies` is root-only |
| 7 | The manifest entry carries a full round-trippable iface (`IfaceJSON`), and `unmarshalIfaceFull` restores `Digest`/`TypeAliases`/`AliasParams` | `cache_store.go:46` (`IfaceJSON`), `:182–196` (`ifaceFullJSON` fields incl. `type_aliases`, `alias_params`), `:298+` |
| 8 | `Result`/`LoadedModule` expose no per-module iface, so manifest decode is the minimal reconstruction path | `pipeline.go:137–166` (`Modules map[string]*loader.LoadedModule`, `Interface` is root-only); `loader.LoadedModule` has no Iface field (loader.go:60–75) |
| 9 | Blast radius is one file: `assertEmbeddedModuleKey` has exactly one caller | `grep -rn "assertEmbeddedModuleKey" internal/ cmd/` → only `cache_invalidation_test.go` |
| 10 | `AILANG_STDLIB_PATH` is the env override; a stdlib root needs the `io.ail` marker | `internal/config/paths.go:7,39–40`; `internal/stdlibroot/root.go:39` (`Marker = "io.ail"`) |
| 11 | Embedded display path is `<embedded>/std/<name>` (matches the retained `wantPath` assertion) | `internal/stdlibroot/root.go:53–58` `DisplayPath` |
| 12 | Repo-layout coupling for Leg 3's std copy is an established pattern | `internal/stdlib/integration_test.go:30–39` references `../../std/*.ail` |
| 13 | Existing generic freshness coverage uses cwd file deps, not stdlib-resolved deps (Leg 3's coverage gap is real) | `TestExplicitEmptyCacheDependencyEdits` writes `leaf.ail`/`middle.ail` in the temp cwd; `TestCacheKey_InvalidatesOnSourceEdit` edits the root module only — neither resolves a dep through the stdlib root |
| 14 | Production test-run failure and CI evidence | Triage iteration 57 (issue #2453): focused Go test + actions run 45644929576 fail on Linux and Windows at `e3595b4f`; parent job green |

Negative-existence claims relied on by the design (one row each, per the doc gate):
- No `ParenExpr`-style hidden second source of imports: `extractImportsFor` (loader.go:596–604+) reads `file.Imports` only — imports and the AST field are mutated together at the single call-site (loader.go:345–350).
- No existing test already covers stdlib-resolved dependency freshness: `grep -rln "std/result" internal/pipeline/*_test.go` + reading the matches — std/result appears only as a *user-facing* type in unrelated tests; only `cache_invalidation_test.go` asserts on its cache entries.
- No exported/in-package helper already reconstructing a per-module iface outside the manifest: `grep -rn "IfaceJSON\|decodeIface\|unmarshalIfaceFull"` → `cache_artifacts.go:113/135` (codec field) and `cache_store.go:298` (the helper this design reuses).

## References

- [Design Axioms](/docs/references/axioms)
- Issue #2453 (triage evidence), actions run 45644929576
- `internal/pipeline/cache_invalidation_test.go` — the failing test and its provenance assertions
- `internal/pipeline/pipeline_module_cache.go`, `internal/pipeline/cache_dep_closure.go`, `internal/pipeline/cache_key.go` — the production key contract (unchanged)

## Future Work

- PATH-dependency digests in `ModuleCacheKey` (see [compile-cache-path-dep-invalidation](../ailang-core-triage/compile-cache-path-dep-invalidation.md)) — separate intake, not bundled here.
- If the stdlib grows beyond the prelude pair in embedded-key coverage needs, consider driving the table from a manifest of expected topology rather than hardcoding — only when a third module actually appears.

---

**Document created**: 2026-11-25
**Last updated**: 2026-11-25
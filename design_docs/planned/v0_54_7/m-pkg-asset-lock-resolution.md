# M-PKG-ASSET-LOCK-RESOLUTION: lockfile-driven, semver-correct asset resolution for std/package.assetPath

**Status**: Planned
**Target**: v0.54.7
**Priority**: P1 (determinism/correctness bug; workaround exists — pin by installing only one version, or pass data as values like sunholo/crew_core does)
**Estimated**: 3 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the SAME program reads DIFFERENT asset data depending on which unrelated versions happen to be cached on the machine. Resolving from the lockfile makes the resolved asset path a function of the program's own declared dependency graph, restoring "same lock ⇒ same bytes on every machine". |
| A2: Replayability | +1 | Reproducible replays (coordinator missions, recorded traces) currently break when a second, newer version of a dependency is installed between runs. Lock-pinned asset paths make replays stable across cache state. |
| A3: Effect Legibility | 0 | No effect-row or signature change: `assetPath` stays `(string, string) -> Result[string, string] ! {FS}`. |
| A4: Explicit Authority | +1 | The lockfile becomes the explicit authority for asset version selection, instead of ambient "whatever is on disk" state — the current behavior is exactly the "ambient access" pattern A4 forbids. |
| A5: Bounded Verification | 0 | No new verification surface. |
| A6: Safe Concurrency | 0 | The new `EffContext.PkgAssets` map is written once per run before execution and read-only afterwards (same discipline as `EnvAllowlist`); per-request `Clone`/`WithBudget` share the reference without copying a lock. |
| A7: Machines First | 0 | No API shape change; the `Err` contract gains a more actionable message ("locked at version X; run ailang install"). |
| A8: Minimal Syntax | +1 | No syntax, stdlib surface, or effect-surface change. |
| A9: Cost Visibility | 0 | No resource-cost semantics touched. |
| A10: Composability | 0 | Composes with the existing lock/`PackageLoader` machinery rather than adding a parallel resolution system. |
| A11: Structured Failure | +1 | "Locked but not installed" fails loud with the locked version in the message, instead of silently reading whatever other version happens to be cached. |
| A12: System Boundary | +1 | Asset resolution crosses the package-cache boundary through the same explicit dependency graph that module imports use, instead of a second, lock-unaware path. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): removes machine-local nondeterminism; introduces none
- [x] A3 (Effects): no hidden side effects; FS capability gating unchanged
- [x] A4 (Authority): removes ambient authority (cache contents), replaces it with the lock
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`FS.pkgAssetPath` (`internal/effects/pkg.go`) resolves `std/package.assetPath(pkg_name, rel_path)` by listing the version directories under `~/.ailang/cache/registry/<vendor>/<name>/`, sorting them with `sort.Strings`, and taking the last one. Two defects:

1. **Lexical, not semver.** With `0.9.0` and `0.10.0` both installed, `"0.9.0" > "0.10.0"` lexically, so the OLDER version wins. The shipped reference page actively denies this: `docs/docs/reference/std-package.md` "Version selection" claims *"assetPath picks the lexically-highest version. This matches semantic versioning for the small versions used in practice (e.g. `0.10.0 > 0.2.0` lexically and semantically)"* — the parenthetical is false (`"0.10.0" < "0.2.0"` lexically, because `'1' < '2'`), and the unit test `TestPkgAssetPath_PicksLatestVersion` (`internal/effects/pkg_test.go:46`) cements the wrong ordering by asserting `0.2.0` beats `0.10.0`.

2. **The caller's `ailang.lock` is ignored.** `pkgAssetPath` never consults the running program's lockfile or resolved dependency graph. A program pinned to `sunholo/x 0.3.0` reads assets from `0.4.0` whenever that version is also cached. The same program therefore reads different data on two machines (or on the same machine before/after an unrelated install), breaking determinism and reproducible replays. This is the exact "ambient authority" pattern the A4 axiom names.

**Current State:**
- `internal/effects/pkg.go:81-82`: `chosen := versions[len(versions)-1]` after `sort.Strings` — the only consumer of `listInstalledVersions` (grep-verified, no other caller).
- The lockfile *is* loaded on every package compile (`internal/pipeline/package_resolver.go` `tryLoadPackageResolver`, via `pkg.LoadLockFile`) and *is* used for module-import resolution (`pkg.PackageLoader.ResolveImport` → `packageDir(locked)`, which honors `source: path` and `source: git` too) — but the resolved graph never reaches the effect runtime.
- `internal/effects/pkg.go` documents the intent as "lexical sort, which matches semver for the small versions used in practice" — a claim the repo's own docs page also makes and which is false for any minor ≥ 10.

**Impact:**
- Any package that ships `assets/` and is consumed at a pinned version (the exact pattern `M-EXT-PORTABILITY-GATE` introduced in v0.19.0 for `sunholo/motoko_ext_mcp`).
- `sunholo/crew_core` hit this in design: it works around the bug by passing scenario data in as a value rather than reading assets — a permanent API tax caused by a resolver defect.
- Coordinator replays/mission traces that depend on package assets are not reproducible across fleet machines with different cache states.

## Goals

**Primary Goal:** `std/package.assetPath` resolves assets from the running program's lockfile-pinned dependency graph, falling back to a semver-max registry scan only for programs with no lock context — never to a lexically-chosen or unpinned version.

**Success Metrics:**
1. A program whose lock pins `sunholo/x 0.3.0` resolves `assetPath("sunholo/x", f)` to the `0.3.0` asset path even when `0.4.0` (or `0.10.0`) is also cached. (Acceptance test: new unit test + package-level fixture test.)
2. With `0.9.0` and `0.10.0` installed and no lock, `0.10.0` is chosen. (`TestPkgAssetPath_PicksLatestVersion` updated and renamed to pin the semver ordering.)
3. A locked-but-not-installed dependency returns `Err` naming the locked version and the install command — never a silent read of a different cached version.
4. All existing tests pass except the one test that intentionally asserts the lexical bug (documented change).
5. Docs (`std-package.md`, builtin `LongDesc`, `std/package.ail` header comment) stop asserting the false lexical-equivalence claim and describe lock-first resolution.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Lock-first with per-package registry fallback (instead of hard error when the package is not in the lock) | Preserves back-compat for bare programs, the REPL, and `examples/asset_path.ail`, all of which have no lock context; a hard error would break the self-disable pattern for programs run outside a package | human | design | med |
| Inject the resolved graph into `EffContext` (new `PkgAssets` field) rather than letting `internal/effects` import `internal/pkg` | `effects` must not import `pipeline` (verified cycle: `pipeline → eval → effects`); injection keeps the runtime decoupled from tooling and matches the existing `Args`/`EnvSnapshot` injection pattern | agent | design | med |
| Locked-but-not-installed is a hard `Err` (no fallback to another cached version) | A silent fallback here is the "no silent fallbacks" guardrail verbatim: it would reintroduce the cross-machine nondeterminism this doc exists to remove | human | design | low |
| The `TestPkgAssetPath_PicksLatestVersion` expectation flip (`0.2.0` → `0.10.0`) is an intentional behavior change | A shipped test asserts the buggy ordering; the sprint must not treat its failure as a regression | agent | design | low |
| Fallback stays anchored at `~/.ailang/cache/registry` (does NOT adopt `AILANG_PACKAGE_ROOT`) | `std-package.md` documents today's behavior as deliberate ("This setting does not change `std/package.assetPath`'s runtime asset lookup"); changing it is a separate, operator-facing decision | agent | design | low |
| Path- and git-sourced locked dependencies become resolvable as asset targets (today only registry installs can ever resolve) | The lock graph already resolves module imports for all three sources; using it for assets is free but is a new capability with new paths crossing the FS surface | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Lock-first with per-package fallback (bare/REPL programs keep working; fallback documented)
- [x] Locked-but-not-installed → hard `Err`, no version fallback
- [x] Behavior change to the latest-version test is intentional and documented
- [x] Fallback does not adopt `AILANG_PACKAGE_ROOT` (deferred, see Future Work)

## Solution Design

### Overview

Make the effect runtime lock-aware by injecting the *already-resolved* dependency graph into the `EffContext` at the points where a run is assembled, and fix the no-lock fallback to select by semver instead of lexical order.

The resolution order inside `pkgAssetPath` becomes:

1. **Sandbox gate** — unchanged (`fsSandboxed()` refusal, M-EXECUTOR-POLICY-HARDENING M1).
2. **Lock context** — if `ctx.PkgAssets` contains `pkg_name`, resolve against the locked target:
   - `Dir` present on disk → asset path = `Dir/assets/<rel>`.
   - `Dir` missing → `Err("package not installed: <pkg>@<version> (locked by ailang.lock); run 'ailang install <pkg>@<version>'")`. **No fallback to any other installed version.**
3. **Registry fallback** (no lock entry: bare programs, REPL, package not among dependencies) — list version dirs, parse each as semver, choose the **semver maximum**. Entries that do not parse as semver are skipped; if no entry parses, fail loud: `Err("package <pkg> has no valid semver versions installed (found: ...)")` rather than silently applying lexical order.
4. Asset existence check (`os.Stat`) and error message shape — unchanged; the chosen version in the message is the locked version on path 2 and the semver-max on path 3.

### Architecture

**Components:**

1. **`effects.PkgAssetTarget` + `EffContext.PkgAssets`** (`internal/effects`): a small exported struct `{Dir, Version string}` and a new `map[string]PkgAssetTarget` field (package name → resolved target). Nil map = no lock context (back-compat). Written once per run before execution; treated as immutable thereafter (same discipline as `Args`, `EnvAllowlist`). Lookups must be nil-safe on `ctx` — existing unit tests call `pkgAssetPath(nil, …)` and pass today because `fsSandboxed()` is nil-safe; the new lookup follows the same `ctx != nil && ctx.PkgAssets != nil` pattern.

2. **`pkg.PackageLoader` target accessor** (`internal/pkg`): a public method, e.g. `AssetTarget(name string) (dir, version string, ok bool)`, reusing the existing unexported `packageDir(locked)` — which already handles `source: registry` (`PackageDir` → `~/.ailang/cache/registry` or `AILANG_PACKAGE_ROOT`), `source: path`, and `source: git`. `ok=true` with `dir=""` when the lock entry exists but its directory is not on disk (the "locked but not installed" case the effect must report, not paper over). Plus a `LockedPackageNames() []string` accessor so the map can be built without exposing the `LockFile`.

3. **Pipeline plumbing** (`internal/pipeline`):
   - `tryLoadPackageResolver` additionally builds the targets map from the lockfile it already loads and returns it (no new package-global; it already returns the resolver).
   - `modulePipelineState.loadModules` stores the map on a new `Result.PkgAssets` field.
   - A helper `AssetTargetsFor(dir string) (map[string]effects.PkgAssetTarget, error)` (walks `pkg.FindManifest` → `pkg.LoadLockFile` → `PackageLoader`) for call sites that configure the effect context *before* a compile exists (embed engine, serve-api) and for one-line wiring anywhere. It returns `nil, nil` for bare directories (no manifest/lock) — that is a legitimate "no lock context", not a failure. `pipeline` already imports `eval`, which imports `effects`; importing `effects` for the map type adds no edge to a cycle (verified — `effects` imports none of `pipeline`/`eval`/`pkg`).

4. **Semver comparator in `effects`** (`internal/effects/pkg.go`): `parseSemver`/`compareSemver` local to the effect package, algorithmically identical to `pkg.ParseSemver` (`internal/pkg/version_compat.go:16`: strip `v` prefix, `major.minor[.patch]`, strip `-pre` suffix, tolerate `dev`/`unknown`). This is a deliberate keep-in-sync duplication — the third such (`AssetsDir`, `isAbsoluteCrossPlatform`) — because `effects` cannot import `internal/pkg` per the package's standing convention (see Verification Log for why the convention stays even though the original cycle claim is stale).

5. **Run-path wiring**:
   - `internal/runner` `runSingle` and `ExecuteBatchItem`: `effCtx.PkgAssets = result.PkgAssets` — both already hold the `pipeline.Result` at `NewEffContext` time (verified: `run.go` `runSingle(ctx, result, opts, …)`, `batch.go` `ExecuteBatchItem(ctx, result, input, …)`).
   - `internal/embed` `Engine`: one engine = one `basePath` = one package = one lockfile, so the targets are **engine-scoped, not request-scoped**. The engine resolves them once via `pipeline.AssetTargetsFor(basePath)` and assigns them to the `EffContext` it stores (`SetEffContext`), before any evaluation runs — evaluation inside `pipeline.RunWithContext` uses the pre-installed ctx, so assignment must precede the first compile (this is why the `AssetTargetsFor(dir)` helper exists alongside the `Result` field). Per-request `Clone` shares the immutable map reference.
   - `cmd/ailang/serve_api.go` → `internal/apiserver`: inherits via the embed engine's `SetEffContext` (the server stores one shared ctx on the engine, `server.go:229`); no separate wiring expected, sprint verifies.
   - **REPL / wasm** (`internal/repl`): unchanged. The REPL evaluates per-statement through its own loop without the package-resolver pipeline path, so it has no lock context and keeps the fallback. Documented, not silently broken.

### Implementation Plan

**Phase 1: runtime resolution** (~1 day)
- [ ] Add `PkgAssetTarget` + `EffContext.PkgAssets` (nil-safe) in `internal/effects/context.go`
- [ ] Add local semver comparator in `internal/effects/pkg.go`; rewrite version selection: lock-first, semver-max fallback, fail-loud on unparseable dirs
- [ ] Update `TestPkgAssetPath_PicksLatestVersion` → `TestPkgAssetPath_PicksSemverMax` (expects `0.10.0`); add tests: locked-version-wins over newer cache, locked-but-not-installed `Err`, nil-ctx fallback, unparseable-dir `Err`, git/path-target resolution
- [ ] Verify `fs_containment_test.go:209-211` (sandbox refusal) still passes unchanged

**Phase 2: lock graph plumbing** (~1 day)
- [ ] `internal/pkg`: `PackageLoader.AssetTarget` + `LockedPackageNames` (reuse `packageDir`; unit tests per source kind)
- [ ] `internal/pipeline`: build targets in `tryLoadPackageResolver`; `Result.PkgAssets`; `AssetTargetsFor(dir)`; tests with a fixture package whose lock pins the older of two cached versions

**Phase 3: wiring + docs** (~1 day)
- [ ] `internal/runner`: run + batch assignment; end-to-end fixture test (two cached versions, lock pins older, run resolves older)
- [ ] `internal/embed`: engine-scoped assignment via `AssetTargetsFor(basePath)`
- [ ] Serve-api smoke test (served handler in a package with a lock)
- [ ] Docs: `docs/docs/reference/std-package.md` (fix the false "Version selection" paragraph, describe lock-first + fallback + locked-but-not-installed error), builtin `LongDesc` in `internal/builtins/pkg.go` ("most recent installed version is selected" → lock-first description), `std/package.ail` header comment, changelog entry
- [ ] `make fmt`, `make test`, `make check-boundaries`, `make check-architecture-closure`

### Files to Modify/Create

**New files:**
- none (all changes in existing files)

**Modified files:**
- `internal/effects/context.go` — `PkgAssetTarget` type + `PkgAssets` field (~15 LOC)
- `internal/effects/pkg.go` — semver comparator + resolution rewrite (~60 LOC net)
- `internal/effects/pkg_test.go` — 1 test updated, ~5 new tests (~120 LOC)
- `internal/pkg/loader.go` — `AssetTarget` + `LockedPackageNames` (~30 LOC)
- `internal/pipeline/package_resolver.go` — targets map build + `AssetTargetsFor` (~40 LOC)
- `internal/pipeline/pipeline.go` — `Result.PkgAssets` field (~3 LOC)
- `internal/pipeline/package_resolver_test.go` — fixture tests (~60 LOC)
- `internal/runner/run.go`, `internal/runner/batch.go` — one-line assignments (~4 LOC)
- `internal/embed/embed.go` — engine-scoped assignment (~10 LOC)
- `internal/builtins/pkg.go` — `LongDesc` text (~5 LOC)
- `std/package.ail` — header comment (~5 LOC)
- `docs/docs/reference/std-package.md` — Version selection + Returns sections (~20 LOC)
- changelog — v0.54.7 entry

## Conflict Surface

This design touches `internal/effects` (runtime resolution semantics of one registered FS op), so the section is mandatory. No parser/lexer/typechecker position is touched: there is no syntax or type change anywhere.

### Positions touched

1. **`FS.pkgAssetPath` op resolution table** (`RegisterOp("FS", "pkgAssetPath", …)` in `internal/effects/pkg.go`): the runtime *selection rule* for which installed package version backs the returned path. The op signature, effect row, and `Result` contract are unchanged.
2. **`EffContext` value fields consumed by ops**: a new read-only field joins `Args`, `EnvSnapshot`, `EnvAllowlist`. The other consumers of `EffContext` copying semantics (`Clone`, `WithBudget`, budget-frame rebuilds) shallow-copy maps by reference today; the new field follows the same discipline.
3. **`pipeline.Result`**: a new informational field populated during `loadModules`, next to `PhaseTimings`/`Interface`. Consumers: `internal/runner`, `internal/embed`.

### What else lives in these positions

| Position | Existing valid form | Interaction |
|----------|--------------------|------------|
| `FS.pkgAssetPath` op impl | Sandbox refusal (checked FIRST, before any name validation) | Unchanged and still first: sandboxed programs cannot probe lock contents or cache layout |
| `FS.pkgAssetPath` op impl | Argument validation (`vendor/name`, safe rel path) | Unchanged, still second; resolution order change is strictly after both gates |
| `EffContext` map-valued fields | `EnvAllowlist []string`, `EnvSnapshot map[string]string` | Same write-once-read-many discipline; no new locking |
| `pipeline.Result` fields set in loadModules | `Interface`, module registry entries | `PkgAssets` is additive; modes that ignore it (check, fmt) are unaffected |

### Disambiguation strategy

Not a parsing problem — no token-level ambiguity exists. The runtime disambiguation is a strict ordered table, resolved at design time:

1. `ctx` nil or `PkgAssets` nil → fallback (bare programs, REPL, tests)
2. `PkgAssets[pkg]` present with `Dir` on disk → locked resolution
3. `PkgAssets[pkg]` present with missing `Dir` → hard `Err` (no fallback — the determinism guarantee)
4. `PkgAssets[pkg]` absent → registry fallback (semver-max)

Case 3 vs 4 is the only "looks ambiguous" pair: a caller might expect missing-dir to fall through to the cache scan. It must not — that fall-through is precisely the bug being fixed (pinned 0.3.0 + cached 0.4.0 must never read 0.4.0). The rule is: **presence in the lock decides; the lock's answer is final.**

### Programs that MUST still work

1. `examples/asset_path.ail` — bare program, no manifest: output must stay `not installed: package not installed: nonexistent/pkg` (verified to exist; uses the fallback path).
2. `TestPkgAssetPath_Resolves`, `TestPkgAssetPath_PackageMissing`, `TestPkgAssetPath_AssetMissing`, `TestPkgAssetPath_RejectsBadInputs` (`internal/effects/pkg_test.go`) — all use `pkgAssetPath(nil, …)`; the nil-ctx fallback must keep them passing verbatim.
3. `internal/effects/fs_containment_test.go:209-211` — sandboxed `pkgAssetPath` must stay denied with the same error shape.
4. The `M-EXT-PORTABILITY-GATE` self-disable pattern (`std-package.md` "Self-disable on missing asset"): `Err` from a missing asset must keep the `Err` constructor and the `asset not found: … in pkg@version` / `package not installed: …` message prefixes, because consumers match on those strings.
5. `ailang pkg install`/`ailang lock` flows and `AILANG_PACKAGE_ROOT` module resolution — untouched (asset resolution is read-side only).

### What deliberately changes

1. Latest-version selection in the fallback: lexical-max → semver-max (`0.10.0` now beats `0.2.0`).
2. Programs whose lock pins an older version than another cached version: now read the pinned version's assets (previously the lexically-last one's).
3. A locked dependency missing from the cache: previously silently resolved to a *different* cached version when one existed; now a hard `Err`.
4. `path`- and `git`-sourced locked dependencies: previously unresolvable by `assetPath` (only registry paths were ever produced); now resolve to their locked directories.
5. One existing unit test flips its expectation (documented above).

## Verification Log

Premises verified against code/docs by direct read or grep (no `go`/`ailang` binary in the authoring sandbox — `ailang check`/`go build`/`go list -deps` checks are delegated to sprint-executor as the first sprint task; language-surface claims below cite source files rather than live check output because no language construct is being claimed — the `.ail` surface is unchanged):

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| 1 | `pkgAssetPath` picks the lexically-last installed version | Read `internal/effects/pkg.go:81-82` (`sort.Strings` + `versions[len(versions)-1]`) | Confirmed |
| 2 | The shipped unit test asserts the buggy ordering (`0.2.0` over `0.10.0`) | Read `internal/effects/pkg_test.go:46-76` (`TestPkgAssetPath_PicksLatestVersion`, comment: "We document 'lexical sort' — that means '0.2.0' wins here") | Confirmed |
| 3 | The shipped docs assert lexical≡semver with a false example (`0.10.0 > 0.2.0` lexically) | Read `docs/docs/reference/std-package.md` "Version selection" paragraph; `"0.10.0" < "0.2.0"` lexically since `'1' < '2'` | Confirmed (docs claim is false) |
| 4 | `pkgAssetPath` today has no access to any lock/resolved-graph state | Read the full `pkgAssetPath` impl; grep `PkgAssets\|LockFile\|lock` in `internal/effects/pkg.go` → only a comment mentions the word "lock" | Confirmed |
| 5 | `listInstalledVersions` has no other caller (fix is isolated) | `grep -rn "listInstalledVersions" --include="*.go"` → only `internal/effects/pkg.go` (definition + one call) | Confirmed |
| 6 | The lockfile already reaches every package compile, and module imports already resolve from it (registry, path, git sources) | Read `internal/pipeline/package_resolver.go` (`tryLoadPackageResolver`, `pkg.LoadLockFile`), `internal/pkg/loader.go` `ResolveImport`/`packageDir` (all three `source` cases) | Confirmed |
| 7 | Runner call sites hold `pipeline.Result` when they create the `EffContext` | Read `internal/runner/run.go` `runSingle(ctx, result, opts, …)` → `effCtx := effects.NewEffContext(programArgs)`; `internal/runner/batch.go` `ExecuteBatchItem(ctx, result, input, …)` → same | Confirmed |
| 8 | `internal/effects` must not import `internal/pipeline` (cycle) | Grep: `internal/eval` imports `internal/effects` (`eval/builtins.go`, `eval_typed_helpers.go`); `internal/pipeline` imports `internal/eval` (`pipeline.Result` holds `eval.Value`) ⇒ `effects → pipeline → eval → effects` would cycle | Confirmed |
| 9 | `internal/effects`'s "cannot import `internal/pkg` (import cycle)" comments (`pkg.go:94`, `abspath.go:6`) are stale — `pkg`'s non-test internal imports are only `config, iface, proctree, schema, strutil`, none of which import `effects` (proctree is a leaf since M-V1-SIMPLIFY-S3 M5, per its package doc) | Grep-closure of each package's non-test imports; `go list -deps` not available in this sandbox | Confirmed at grep depth; this doc does NOT rely on lifting the convention — it keeps `effects` free of the `pkg` import and flags the stale comments for optional cleanup |
| 10 | `pipeline` may import `internal/effects` (no cycle: `effects` imports none of `pipeline`/`eval`/`pkg`) | Grep of `internal/effects` non-test internal imports: `ai, ast, config, eval, fileguard, gitexec, proctree, runlife, secrets, simhash, trace` | Confirmed |
| 11 | `pipeline.Result` has no resolver/lock field today | Read `pipeline.go:134-158` (`Result` struct) | Confirmed (negative-existence) |
| 12 | `EffContext` has no package/lock field today | Read `context.go:27-115` (full struct) | Confirmed (negative-existence) |
| 13 | Existing tests call `pkgAssetPath(nil, …)` and pass because `fsSandboxed()` is nil-safe | Read `pkg_test.go` (all tests pass `nil` ctx) + `fs_root.go:127` (`func (ctx *EffContext) fsSandboxed() bool { return ctx != nil && … }`) | Confirmed — new lookup must be nil-safe the same way |
| 14 | `std/package.assetPath` signature is `(string, string) -> Result[string, string] ! {FS}` | Read `std/package.ail:30-31` (the stdlib source itself — no live `ailang check` available in this sandbox; the surface is unchanged by this design) | Confirmed |
| 15 | Registry example fixture exists and its expected output uses the fallback | `ls examples/asset_path.ail`; read the file (`Err(e) => println("not installed: ${e}")`) | Confirmed |
| 16 | Sandbox-refusal regression fixture asserts the current denial | Read `internal/effects/fs_containment_test.go:209-211` (`pkgAssetPath` → `deniedByErr`) | Confirmed |
| 17 | `AILANG_PACKAGE_ROOT` deliberately does not affect `assetPath` today (so the fallback must not silently adopt it) | Read `docs/docs/reference/std-package.md` "Registry module resolution and confinement": "This setting does not change `std/package.assetPath`'s runtime asset lookup." | Confirmed — documented behavior, kept |
| 18 | The embed engine shares one `EffContext` across served modules, installed before evaluation | Read `internal/apiserver/server.go:196-230` (one ctx stored via `eng.SetEffContext`), `internal/embed/embed.go:425-446` (SetEffContext → `evaluator.SetEffContext`, before `pipeline.RunWithContext` evaluation) | Confirmed — motivates engine-scoped, pre-compile assignment via `AssetTargetsFor(dir)` |
| 19 | No new `MOD`/`PAR`/`TC`/`EFF` diagnostic code is proposed | This design changes `Err` message strings within the existing `Result[string, string]` contract only; no code allocation | N/A (no code claimed) |
| 20 | Registry cache version dirs are semver-shaped | `ailang install` writes dirs named by registry-index versions (`internal/pkg/registry.go` `ResolveLatestVersion`/`CachedPackagePath`); no publish-time semver-format gate was found in grep, so the fallback must tolerate (and fail loud on) unparseable dirs rather than assume | Partially confirmed; handled by design (fail-loud, no lexical fallthrough) |

## Examples

### Example 1: pinned program with a newer version also cached (the repro)

**Before** (lock pins `sunholo/x 0.3.0`; `0.3.0` and `0.4.0` both installed):

```ailang
-- program main.ail, part of a package whose ailang.lock pins sunholo/x = 0.3.0
import std/package (assetPath)
match assetPath("sunholo/x", "schema.json") {
  Ok(p) => ...   -- p points at .../registry/sunholo/x/0.4.0/assets/schema.json (WRONG version)
  Err(e) => ...
}
```

**After**:

```ailang
-- same program, same cache state
import std/package (assetPath)
match assetPath("sunholo/x", "schema.json") {
  Ok(p) => ...   -- p points at .../registry/sunholo/x/0.3.0/assets/schema.json (locked version, on every machine)
  Err(e) => ...
}
```

### Example 2: locked dependency not installed

**Before**: silently reads whatever other cached version exists (or `Err("package not installed: …")` if none — machine-dependent).

**After**: deterministic, actionable failure:

```ailang
Err("package not installed: sunholo/x@0.3.0 (locked by ailang.lock); run 'ailang install sunholo/x@0.3.0'")
```

### Example 3: no lock (bare program, REPL) — fallback only

`assetPath("test/pkg", "v.txt")` with `0.1.0`, `0.2.0`, `0.10.0` cached → now resolves `0.10.0` (semver-max), where today it resolves `0.2.0` (lexical-max).

## Success Criteria

- [ ] Program with lock pinning `sunholo/x 0.3.0`, with `0.3.0`+`0.4.0` cached, resolves the `0.3.0` asset (e2e test via `internal/runner` fixture)
- [ ] Locked-but-not-installed returns `Err` naming the version and install command; no cache fallback (unit test)
- [ ] Fallback with `{0.9.0, 0.10.0}` picks `0.10.0` (updated `TestPkgAssetPath_PicksSemverMax`)
- [ ] `pkgAssetPath(nil, …)` tests and `examples/asset_path.ail` behavior unchanged (regression tests pass verbatim)
- [ ] Sandboxed programs still refused before any resolution (`fs_containment_test.go` unchanged)
- [ ] All tests passing (`make test`, `make test-core`)
- [ ] Documentation updated (`std-package.md` false claim removed; builtin metadata; changelog)

## Testing Strategy

**Unit tests:**
- `internal/effects/pkg_test.go`: locked-wins-over-newer-cache, locked-not-installed, semver-max fallback, unparseable-version dirs, nil-ctx, git/path targets
- `internal/pkg`: `AssetTarget` for `registry`/`path`/`git` sources, missing-dir case
- `internal/pipeline`: `Result.PkgAssets` populated from a fixture package with a lock pinning the older of two cached versions; `AssetTargetsFor` on bare dir → `nil, nil`

**Integration tests:**
- Runner e2e fixture: package with lockfile + two cached dependency versions + an `assets/` file whose content differs per version; entry program asserts the locked content via `assetPath` + `FS.read`
- Serve-api/embed: served handler module inside a package with a lock resolves the locked asset

**Manual testing:**
- The reporter's repro: install two versions of a package that ships assets, pin the older in the consumer's lock, `ailang run` — locked path returned; then delete the pinned cache dir and re-run — actionable `Err`, no fallback

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `internal/embed` assignment point (`SetEffContext` vs `New`) — agent may choose, provided assignment precedes the first evaluation and is engine-scoped (not per-request)
- Map construction eager-at-compile vs lazy-per-call inside the fallback — agent may choose; eager is expected to be a handful of `Stat` calls per compile
- Optional cleanup of the stale "import cycle" comments in `internal/effects/pkg.go:94` / `abspath.go:6` (the cycle dissolved when proctree became a leaf) — agent may reword to cite layering convention instead; do NOT add the `pkg` import
- Test naming and whether the `crew_core`-style value-passing workaround gets a doc note — agent may decide

## Non-Goals

**Not attempted in this feature:**
- `AILANG_PACKAGE_ROOT` support in the *fallback* path — documented current behavior; separate operator-facing decision
- REPL lock awareness — the REPL does not resolve `pkg/` imports through the lockfile at all today; that is a separate gap
- Own-package (root package) asset resolution — a program calling `assetPath` on its own package name still requires that package to be installed in the cache; the lockfile contains dependencies, not the root
- Any change to `assetPath`'s effect row, stdlib signature, or the FS capability model
- Network fetch-on-miss: a missing install stays an `Err`

## Timeline

**Week 1** (~3 days, single sprint):
- Phase 1: runtime resolution + effect tests (1 day)
- Phase 2: lock plumbing + pipeline tests (1 day)
- Phase 3: wiring, docs, changelog, full gates (1 day)

**Total: ~3 days** (estimates already doubled per skill guidance)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Consumers matching on `Err` message strings break when a locked-but-not-installed message appears where a silent fallback used to succeed | Med | Keep the `package not installed: …` prefix; grep of in-repo consumers (`examples/asset_path.ail`, stdlib, docs) shows only prefix/pattern matching, no full-string equality; changelog entry flags the message shape |
| Embed/serve-api assignment ordering (ctx must carry targets before first evaluation) | Med | The `AssetTargetsFor(dir)` helper resolves from the engine's `basePath`, which exists at engine construction — no compile result needed; e2e serve-api test in Phase 3 |
| `pipeline` gains a direct `internal/effects` import | Low | `pipeline → eval → effects` edge already exists; `make check-boundaries` and `make check-architecture-closure` both run in Phase 3 gates |
| Semver comparator drift from `pkg.ParseSemver` | Low | Keep-in-sync comment on both sides + a mirrored test table (0.9.0/0.10.0, v-prefix, pre-release strip, dev/unknown tolerance) |
| Duplicated `packageDir` knowledge if `AssetTarget` is implemented by copy instead of reuse | Low | Design mandates reuse of the existing unexported `packageDir` method on `PackageLoader` |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_19_0/m-ext-portability-gate.md](../implemented/v0_19_0/m-ext-portability-gate.md) — introduced `std/package.assetPath`, assets bundling, and the "may need a way for assetPath to resolve to package-cache paths" note this doc finishes
- [design_docs/implemented/v0_18_11/m-ext-portability-gate-sprint-plan.md](../implemented/v0_18_11/m-ext-portability-gate-sprint-plan.md) — same sprint's plan
- M-EXECUTOR-POLICY-HARDENING — the sandbox-first refusal in `pkgAssetPath` that this design keeps as gate #1

**Planned (check for overlap):**
- [design_docs/planned/ailang-core-triage/compile-cache-path-dep-invalidation.md](ailang-core-triage/compile-cache-path-dep-invalidation.md) — path-dep cache invalidation; touches the same `PackageLoader.packageDir` path-source logic, no overlap in asset resolution
- [design_docs/planned/ailang-core-triage/local-dep-replace-override.md](ailang-core-triage/local-dep-replace-override.md) — local dependency overrides; complementary lockfile work
- [design_docs/planned/m-ext-registry-effect-row-ruling.md](m-ext-registry-effect-row-ruling.md) — registry/effect-row ruling for extension packages

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [CLAUDE.md](../../../CLAUDE.md) - "No silent fallbacks" guardrail (the locked-but-not-installed case) and "Systemic fixes" principle (this design unifies registry/path/git asset resolution through the one existing lock graph instead of a second resolution system)
- [docs/docs/reference/std-package.md](../../../docs/docs/reference/std-package.md) - the reference page carrying the false lexical-equivalence claim this sprint corrects
- [ARCHITECTURE.md](../../../ARCHITECTURE.md) - layer rules; language-closure gate (`internal/diag/closure_test.go`) both `effects` and `pkg` already live in

## Future Work

- Honor `AILANG_PACKAGE_ROOT` in the no-lock fallback (operator decision; docs currently say it does not apply)
- Own-package asset resolution (a package reading its OWN `assets/` without a cache install, resolved from its manifest dir)
- REPL lock-context awareness once the REPL grows package-aware imports
- Consider surfacing the chosen dependency version in the semantic trace for replay diffing

---

**Document created**: 2026-10-12
**Last updated**: 2026-10-12

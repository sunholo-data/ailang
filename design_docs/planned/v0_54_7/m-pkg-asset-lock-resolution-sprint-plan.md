# Sprint Plan: M-PKG-ASSET-LOCK-RESOLUTION

## Summary

Resolve package assets from the running program's locked dependency targets. When a package has no lock entry, select the maximum installed version using the existing package semver compatibility rules. A missing locked target must return an actionable error without choosing another cached version.

**Design:** [Approved design](m-pkg-asset-lock-resolution.md)
**Target:** v0.54.7 (checkout std/VERSION: v0.54.6)
**Duration:** 3 days; four implementation sessions of 60–90 minutes, plus review and integration buffer.
**Estimated total:** 650 LOC: 270 implementation, 330 tests, 50 documentation.
**Risk:** Medium — effect-context propagation and embed initialization order.
**Status:** Ready for sprint-plan review; implementation starts after coordinator approval and executor handoff.

## Current Status and Velocity

The available git history contains one commit in the last seven days: 7393ba49, the approved design (#1774). The velocity script found no usable recent implementation LOC metrics; its changelog scan includes historical entries and cannot establish current LOC/day. This is an engineering estimate, not a measured throughput claim. Capacity is approximately 217 LOC/day, retaining the approved three-day allowance and reserving integration time beyond the four focused sessions.

Verified gaps: effects/pkg.go still sorts version directory names lexically; pipeline loads the lock but exports no asset map; runner run/batch and embed have no asset-target wiring. PackageLoader.packageDir already resolves registry, relative path, and git sources. WithBudget reconstructs EffContext field by field; Clone must also preserve the new immutable map. Current context.go is 750 lines and runner/run.go is 730 lines: keep additions small and put new tests/helpers in companion files to stay below 800 lines.

The design includes systemic analysis across runtime, import resolution, runner/batch, embed/serve-api, REPL, and docs. No parser or type changes are needed.

## Registry Reuse Audit

Executed `ailang pkg search semver` and `ailang pkg search asset`: both returned no packages (CLI emitted a stale-binary warning). No candidates existed to inspect with pkg info/docs. These results are supporting evidence only: this fix must operate within Go runtime/pipeline infrastructure before AILANG package evaluation, and cannot be supplied by a registry dependency. Each milestone therefore records `none`, reusing existing in-repo PackageLoader resolution and semver rules rather than introducing a parallel package system.

## Proposed Milestones

### M1: Runtime lock-first resolution (~200 LOC)

**Dependencies:** None
**Duration:** Day 1, session 1, 60–90 minutes.
**Estimate:** 80 implementation + 120 tests.
**Files:** internal/effects/context.go, pkg.go, pkg_test.go; new pkg_semver.go, pkg_lock_test.go, context_pkg_assets_test.go.

Add PkgAssetTarget {Dir, Version} and immutable PkgAssets map; explicitly preserve it through WithBudget and Clone. Keep helpers outside context.go. Resolve a present lock entry first, retaining its version even when its directory is missing. Keep sandbox refusal and argument/path validation ahead of all lookup. Implement fallback comparison matching internal/pkg/version_compat.go, including its documented prefix/prerelease/dev/unknown behavior; add keep-in-sync comments and deterministic handling for equal numeric versions. Correct the existing lexical-order test.

**Acceptance criteria:**

- [ ] Older locked version wins over newer installed versions; missing locked directory returns Err naming package/version and install guidance, never fallback.
- [ ] Nil context and packages absent from the map retain fallback; 0.10.0 beats 0.9.0 and 0.2.0.
- [ ] Invalid directory names are skipped; exclusively invalid versions fail with a structured Err; compatibility edge cases mirror pkg.ParseSemver.
- [ ] Path/git target directories resolve assets; sandbox, invalid package names, absolute paths and traversal still fail before lookup.
- [ ] WithBudget and Clone preserve targets without mutating the shared map; focused effects tests and make test-changed pass.

**Risk:** Comparator duplication drifts; mitigate with mirrored edge-case tables. This is compatibility ordering as specified by the design, not a new strict SemVer prerelease policy.

### M2: Export locked targets through package loader and pipeline (~190 LOC)

**Dependencies:** M1
**Duration:** Day 1, session 2, 60–90 minutes.
**Estimate:** 90 implementation + 100 tests.
**Files:** internal/pkg/loader.go; new loader_assets_test.go; internal/pipeline/package_resolver.go, pipeline.go, pipeline_module_phases.go, package_resolver_test.go; new package_assets.go and package_assets_test.go.

Add AssetTarget and LockedPackageNames using existing packageDir. Build maps from every locked dependency, including transitive entries, and carry them through Result.PkgAssets during loadModules. Update existing resolver tests for the changed return shape. Add AssetTargetsFor(dir) for pre-evaluation callers, sharing map construction with compile-time resolution and introducing no package-global asset state. Preserve a map entry for unresolved locked targets so runtime cannot accidentally fall back.

**Acceptance criteria:**

- [ ] Accessors cover registry, relative path, git and missing directories; absent lock entries are distinguishable from unresolved locked entries.
- [ ] Result.PkgAssets contains all locked dependency targets, including a pinned older version despite a newer cached version.
- [ ] AssetTargetsFor returns nil,nil for no manifest or absent lock; malformed existing manifest/lock returns error, never silent fallback.
- [ ] Discovery remains anchored to the entry source or explicit PackageDir; bare/self-only resolution keeps existing behavior.
- [ ] Focused pkg/pipeline tests, make test-changed and make check-boundaries pass.

**Risk:** Existing lock validation may reject missing installs before execution. Test the runtime Err separately and permit compile-time failure where existing import validation already requires an installed package; never weaken hash validation to force the fixture through.

### M3: Runner, batch and embed propagation (~210 LOC)

**Dependencies:** M2
**Duration:** Day 2, session 3, 60–90 minutes.
**Estimate:** 100 implementation + 110 tests.
**Files:** internal/runner/run.go, batch.go; new package_assets_test.go; internal/embed/embed.go; new package_assets.go and package_assets_test.go; internal/apiserver/package_assets_test.go if needed.

Assign Result.PkgAssets before runner and batch evaluation. Resolve engine-scoped targets from basePath before the first embed evaluation, propagate resolution errors through an existing error-returning initialization/evaluation boundary, and ensure SetEffContext cannot replace a configured context with a lock-unaware one. Verify serve-api inherits the engine targets without duplicate request-time resolution. Keep each engine's map separate.

**Acceptance criteria:**

- [ ] Runner and batch fixtures with two cached versions read the locked version's distinct asset contents.
- [ ] Embed evaluation and an in-process API handler resolve locked assets on the first call and subsequent context clones.
- [ ] Two engines with different locks do not share targets; malformed lock errors surface before evaluation.
- [ ] Missing locked target never reads newer cached data; no network fetch or change to FS capability checks.
- [ ] Focused runner/embed/apiserver tests, make test-changed and make check-architecture-closure pass.

**Risk:** SetEffContext accepts interface{} and cannot return errors. Resolve through an error-returning engine boundary rather than swallowing errors or changing the public API without review. Keep new runner helpers outside run.go.

### M4: Documentation and focused integration gates (~50 LOC)

**Dependencies:** M3
**Duration:** Day 3, session 4, 60–90 minutes with remaining integration buffer.
**Estimate:** 50 documentation; no additional implementation/test LOC budget.
**Files:** docs/docs/reference/std-package.md, internal/builtins/pkg.go, std/package.ail (header only), changelogs/unreleased/2026-10-12-pkg-asset-lock-resolution.md.
**Example:** Verify existing examples/asset_path.ail, plus generated temporary package fixtures from M3. No new public showcase module is required for this behavior correction.

Correct lexical-equivalence claims and describe lock-first selection, per-package fallback, missing-target errors, path/git sources, and the intentionally unchanged fallback cache root. Write an unreleased fragment opening with `### `; flag the intentional latest-version test expectation change.

**Acceptance criteria:**

- [ ] Reference docs, builtin LongDesc and stdlib header agree on lock-first and compatibility-semver fallback behavior.
- [ ] Documentation states REPL has no lock context, absent lock entries use fallback, fallback ignores AILANG_PACKAGE_ROOT, and root-package assets are outside this change.
- [ ] Existing asset_path example checks/runs with FS and its documented no-install behavior; generated integration fixtures demonstrably select older locked bytes.
- [ ] make test-changed, make check-boundaries, make check-architecture-closure, make check-file-sizes and make check-changelog pass.
- [ ] make verify-stdlib and make verify-examples pass; gofmt applied to changed Go files and focused lint checks are clean.

## Day-by-Day Work

Day 1: M1 runtime semantics and copy propagation; M2 package/pipeline plumbing.
Day 2: M3 execution-path integration and temporary fixture tests.
Day 3: M4 docs/example verification and focused final gates; reserve remaining capacity for failures or initialization-order adjustments. Do not expand scope to REPL lock support or fallback root configuration.

## AILANG Syntax and Example Checklist

AILANG prompt version loaded: v0.16.7 (`ailang prompt --version-active`); the full current teaching prompt was loaded with `ailang prompt`. Prompt version and compiler release differ. Executor reloads the current prompt before writing any fixture .ail code, and uses a binary built from this checkout for validation.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| examples/asset_path.ail (existing) | skip: unchanged filesystem demo has environment-dependent paths | include: preserve existing main FS/IO effect row | skip: environment-dependent assets verified by Go integration fixtures |
| Temporary runner/batch/embed/API fixture modules | skip: cache selection is an external filesystem behavior, verified by Go assertions | include: assetPath/readFile handlers declare FS; output entry points additionally declare IO where needed | skip: Go harness controls two cache versions and asserts returned path/content |
| std/package.ail | skip: header-only documentation edit | include: preserve assetPath FS row and existing signature | skip: header-only edit; effects and integration tests cover changed behavior |

## Validation and Success Metrics

Use `make test-changed` (touched packages plus core tests), targeted package tests, and the gates named above. Do not require `make test` in executor sessions; CI runs the full suite. All changed resolution branches have meaningful regression assertions; no arbitrary global coverage percentage is claimed. Package fixtures use temporary home/cache/lock paths and no network or real registry installs. Preserve sandbox-first behavior and immutable context sharing.

Run file-size checks after each milestone; context.go and runner/run.go must remain below 800 lines. If additions threaten the limit, use the named companion files rather than a broad refactor. Validate the sprint JSON with the sprint-executor validation script and confirm estimates/dependency IDs match this plan.

## Dependencies, Assumptions and Handoff

No external package dependency. Approval applies to the supplied design; this new sprint plan is subject to the coordinator's sprint-plan approval gate. The coordinator receives the artifact paths and performs the executor handoff after approval; do not start implementation or merge the planning branch as part of this task.

GitHub #1774 is the verified design-document PR, not a verified originating bug issue. Leave github_issues empty rather than auto-linking unrelated keyword matches. Executor can link an originating issue when the coordinator supplies it.

No blocking design questions remain. Assignment location inside embed is an implementation choice bounded by the initialization-order and error-propagation criteria above.

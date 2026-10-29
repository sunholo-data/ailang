# Sprint Plan: M-PKG-ASSET-LOCK-RESOLUTION

## Summary

Resolve package assets from the running program's locked dependency targets. When a package has no lock entry, select the maximum installed version using the existing package semver compatibility rules. A missing locked target must return an actionable error without choosing another cached version.

**Design:** [Approved design](m-pkg-asset-lock-resolution.md)
**Target:** v0.56.0, unreleased backlog batch (original v0.54.7 planning target retained in history)
**Duration:** 3 days; four implementation sessions of 60–90 minutes, plus review and integration buffer.
**Estimated total:** 650 LOC: 270 implementation, 330 tests, 50 documentation.
**Risk:** Medium — effect-context propagation and embed initialization order.
**Status:** Completed and independently accepted95/100 in round3; merged #1849. Execution was authorized by the original coordinator handoff and attended backlog repair. Dated historical failures below remain preserved.

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

- [x] Older locked version wins over newer installed versions; missing locked directory returns Err naming package/version and install guidance, never fallback.
- [x] Nil context and packages absent from the map retain fallback; 0.10.0 beats 0.9.0 and 0.2.0.
- [x] Invalid directory names are skipped; exclusively invalid versions fail with a structured Err; compatibility edge cases mirror pkg.ParseSemver.
- [x] Path/git target directories resolve assets; sandbox, invalid package names, absolute paths and traversal still fail before lookup.
- [x] WithBudget and Clone preserve targets without mutating the shared map; focused effects tests and make test-changed pass.

**Risk:** Comparator duplication drifts; mitigate with mirrored edge-case tables. This is compatibility ordering as specified by the design, not a new strict SemVer prerelease policy.

### M2: Export locked targets through package loader and pipeline (~190 LOC)

**Dependencies:** M1
**Duration:** Day 1, session 2, 60–90 minutes.
**Estimate:** 90 implementation + 100 tests.
**Files:** internal/pkg/loader.go; new loader_assets_test.go; internal/pipeline/package_resolver.go, pipeline.go, pipeline_module_phases.go, package_resolver_test.go; new package_assets.go and package_assets_test.go.

Add AssetTarget and LockedPackageNames using existing packageDir. Build maps from every locked dependency, including transitive entries, and carry them through Result.PkgAssets during loadModules. Update existing resolver tests for the changed return shape. Add AssetTargetsFor(dir) for pre-evaluation callers, sharing map construction with compile-time resolution and introducing no package-global asset state. Preserve a map entry for unresolved locked targets so runtime cannot accidentally fall back.

**Acceptance criteria:**

- [x] Accessors cover registry, relative path, git and missing directories; absent lock entries are distinguishable from unresolved locked entries.
- [x] Result.PkgAssets contains all locked dependency targets, including a pinned older version despite a newer cached version.
- [x] AssetTargetsFor returns nil,nil for no manifest or absent lock; malformed existing manifest/lock returns error, never silent fallback.
- [x] Discovery remains anchored to the entry source or explicit PackageDir; bare/self-only resolution keeps existing behavior.
- [x] Focused pkg/pipeline tests, make test-changed and make check-boundaries pass.

**Risk:** Existing lock validation may reject missing installs before execution. Test the runtime Err separately and permit compile-time failure where existing import validation already requires an installed package; never weaken hash validation to force the fixture through.

### M3: Runner, batch and embed propagation (~210 LOC)

**Dependencies:** M2
**Duration:** Day 2, session 3, 60–90 minutes.
**Estimate:** 100 implementation + 110 tests.
**Files:** internal/runner/run.go, batch.go; new package_assets_test.go; internal/embed/embed.go; new package_assets.go and package_assets_test.go; internal/apiserver/package_assets_test.go if needed.

Assign Result.PkgAssets before runner and batch evaluation. Resolve engine-scoped targets from basePath before the first embed evaluation, propagate resolution errors through an existing error-returning initialization/evaluation boundary, and ensure SetEffContext cannot replace a configured context with a lock-unaware one. Verify serve-api inherits the engine targets without duplicate request-time resolution. Keep each engine's map separate.

**Acceptance criteria:**

- [x] Runner and batch fixtures with two cached versions read the locked version's distinct asset contents.
- [x] Embed evaluation and an in-process API handler resolve locked assets on the first call and subsequent context clones.
- [x] Two engines with different locks do not share targets; malformed lock errors surface before evaluation.
- [x] Missing locked target never reads newer cached data; no network fetch or change to FS capability checks.
- [x] Focused runner/embed/apiserver tests, make test-changed and make check-architecture-closure pass.

**Risk:** SetEffContext accepts interface{} and cannot return errors. Resolve through an error-returning engine boundary rather than swallowing errors or changing the public API without review. Keep new runner helpers outside run.go.

### M4: Documentation and focused integration gates (~50 LOC)

**Dependencies:** M3
**Duration:** Day 3, session 4, 60–90 minutes with remaining integration buffer.
**Estimate:** 50 documentation; no additional implementation/test LOC budget.
**Files:** docs/docs/reference/std-package.md, internal/builtins/pkg.go, std/package.ail (header only), changelogs/unreleased/2026-10-12-pkg-asset-lock-resolution.md.
**Example:** Verify existing examples/asset_path.ail, plus generated temporary package fixtures from M3. No new public showcase module is required for this behavior correction.

Correct lexical-equivalence claims and describe lock-first selection, per-package fallback, missing-target errors, path/git sources, and the intentionally unchanged fallback cache root. Write an unreleased fragment opening with `### `; flag the intentional latest-version test expectation change.

**Acceptance criteria:**

- [x] Reference docs, builtin LongDesc and stdlib header agree on lock-first and deployed semver fallback behavior.
- [x] Documentation states REPL has no lock context, absent lock entries use fallback, fallback honors AILANG_PACKAGE_ROOT, and root-package assets remain available without a lock (reconciled with v0.54.13).
- [x] Existing asset_path example checks/runs with FS and its documented no-install behavior; generated integration fixtures demonstrably select older locked bytes.
- [x] make test-changed, make check-boundaries, make check-architecture-closure, make check-file-sizes and make check-changelog pass.
- [x] make verify-stdlib and make verify-examples pass; gofmt applied to changed Go files and focused lint checks are clean.

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

## Execution Verification (2026-10-16)

Implemented against checkout v0.54.12 (starting commit `3af754c4`), retaining the approved v0.54.7 design scope. Actual work branch: `coordinator/task-d61964fa` (the coordinator allocated a new task branch for execution).

AILANG prompt version loaded: v0.16.7. The generated fixture checks successfully; its `read` function declares FS and its assertion entry uses FS/IO (`exit(23)` on wrong bytes). Contracts and inline tests are intentionally omitted for controlled filesystem fixtures; Go tests assert the external data.

- Regression-first evidence: the original fallback selected 0.9.0 when 0.10.0 was cached; the original embed call read newer bytes and swallowed malformed-lock errors. New loader/context tests initially failed on missing APIs.
- Focused integration passes for pipeline, runner, batch, embed and API, including older locked bytes, repeated context replacements, engine isolation, missing locked data and malformed locks.
- `make check-boundaries`, `make check-architecture-closure`, `make check-file-sizes`, `make check-changelog`, `make fmt-check`, `make verify-stdlib` (48 interfaces) and `make verify-examples` pass. The existing `examples/asset_path.ail` checks and runs with IO/FS, printing the documented missing-package result.
- The first broad run exposed leaked AILANG_STDLIB_PATH from existing API tests; the shared fixture now clears that override. The serialized API package suite passes (69.985s).
- `TestRuntimeNestedStopUsesRemainingDeadline` fails on both the implementation and untouched starting commit (baseline reproduced in a temporary detached worktree). This process-lifecycle failure is outside the asset-resolution change. Pipeline testing was killed under memory pressure on concurrent and serialized runs, including GOGC=20, GOMEMLIMIT=512MiB, GOFLAGS=-p=1 and -parallel=1. Focused pipeline regressions pass; broad pipeline success is not claimed.
- Tooling friction: Go was installed off PATH; the startup script incorrectly reported success. All actual Go commands use `/usr/local/go/bin`. The first linter link was killed; retrying with GOGC=20 and GOMAXPROCS=2 installed the required tool. Runner PTY tests initially lacked ps; procps was downloaded/extracted outside the repository for the final run. No repository tooling changes were needed.

The package resolver keeps its existing return signature; the module phase obtains asset targets from its concrete PackageLoader. Shared test fixtures live in `internal/testutil/pkgassets` to avoid duplicating registry/lock construction across four consumers. No package-global asset map was introduced.

Repeated verification: `go test ./internal/effects -run 'TestPkgAssetPath|TestAssetVersionCompatibility' -count=20` passes, including invalid directory filtering and package-name traversal rejection. The bounded `make test-changed` run explicitly excludes only the process-lifecycle baseline failure; it does not establish an unfiltered green gate.

### Final Gate Outcome

- Final bounded changed-package run: API, builtins, effects (one baseline exclusion), embed, pkg and runner pass; pipeline is killed at 46.101s. The target exits nonzero, so no unfiltered or whole-sprint test success is claimed. CI must provide the broader verification.
- Focused golangci-lint over all eight touched packages exits zero with `0 issues.` Full `make lint` reports `0 issues.` but times out (exit 4), so its gate remains incomplete.
- Independent evaluation report: `.ailang/state/evaluations/eval_M-PKG-ASSET-LOCK-RESOLUTION_round_1.json`. Review found no remaining runtime defect but withheld approval because broad validation is incomplete.
- All implementation is committed. Sprint JSON retains `in_progress`, M4 `passes: false`, and `implementation_complete: true` to distinguish finished code from unmet validation criteria. Remaining unchecked acceptance boxes refer to those broad gates. No merge or publication performed.

## Attended CI/residual repair (2026-10-29)

Reconciled with current dev without dropping its package-root fallback,
root-package asset support, no-exports entry diagnostics, or release/prerelease
ordering. The basic CLI fix was already released; this PR retains the remaining
missing-target, budget-context, compile-source snapshot and per-engine behavior.

AILANG prompt version loaded: v0.16.7 (current `ailang prompt` output).
Regression-first run confirmed missing locked-target authority, local target
resolution, invalid-only-cache errors, legacy budget propagation and root assets
without a lock. The six-package focused asset suite now passes (effects, pkg,
pipeline, embed, apiserver, runner); runner/batch additionally prove that an
on-disk lock change after compilation cannot change selected asset bytes.

- [x] Resolve stale code/doc conflicts while preserving current deployed behavior.
- [x] Focused asset regressions pass across all six runtime/compiler consumers.
- [x] Complete package-wide gates and stdlib/example verification.
- [x] Fresh independent round3 PASS95; original FAIL51/FAIL68 preserved unchanged.

Historical failures and unchecked original gates above remain recorded until the
fresh checks complete. This note does not turn the original FAIL51 into a pass.

### Fresh local validation (2026-10-29)

All previously incomplete validation gates now pass without exclusions:

- Full `go test -p 2` over effects/pkg/pipeline/embed/apiserver/runner passed
  (38.372s/20.124s/17.976s/0.476s/28.881s/67.581s respectively).
- `make test-changed` passed all eight changed packages and the complete core
  loop, without skip regexes or memory workarounds. The formerly killed pipeline
  suite passed twice (28.861s in changed packages, 16.577s in core).
- Asset-resolution regressions passed with `-count=20` (0.862s).
- `make lint` passed (0 issues); boundaries, architecture closure, file-size,
  changelog and formatting gates passed.
- `make verify-stdlib verify-examples` passed: 49 stable interfaces; runnable
  examples and manifest validation passed (242 modules, zero drift/missing).
- Fresh branch binary checked and ran `examples/asset_path.ail`, printing its
  documented `not installed: package not installed: nonexistent/pkg` result.
- `git diff <reviewed-dev-base> --check` is clean. A documentation-only dev
  update was merged afterward; it changes no asset implementation or tests.

The historical round-1 FAIL51 report remains unchanged. Code validation is now
complete; independent review and exact-head CI remain the pre-merge gates for
this attended repair. Reviewable commits are pushed to the existing PR; no merge
or coordinator dispatch has been performed.

### Independent-review repairs and authority integration (2026-10-29)

- Corrected the reference's no-lock PACKAGE_ROOT fallback description and the
  M2 criterion for manifest/no-lock root targets.
- Added the missing CallPreserveFloats discovery-error guard after the reviewer
  reproduced malformed-lock fallback to newer cached bytes. Regression was red
  before the guard, then full embed/API suites passed.
- Sonar reported 75.7% new coverage: 25 of 27 uncovered newlines were in the shared
  fixture helper under per-package instrumentation. Relocated the existing
  public float-preserving Engine locked-byte regression into external
  pkgassets_test, preserving its cold/warm/context-replacement assertions.
  Added a corrupt registry directory Err regression for the real uncovered
  effects branch; no contrived race hook or coverage policy change.
- Integrated dev/#1964 c2944389 cleanly. All seven covered suites pass:
  effects 66.343s, pkg 111.791s, pipeline 127.209s, embed 8.367s,
  apiserver 85.038s, runner 208.542s, testutil/pkgassets 2.205s. No exclusions.
- Lint 0, file sizes, boundaries, changelog and git-exec AST gate pass.
- Local coverage now covers 21 formerly uncovered measured newlines, projecting
  105/111=94.6%. Exact-head remote Sonar remains authoritative and pending.
  Fresh independent evaluation is still pending; no self-score or merge.

### Portable cache-error assertion repair (2026-10-29)

Exact-head af6dca29 CI exposed a Unix-only wording assertion in the new corrupt
registry test: Windows returns a named missing-package Err for a non-directory
path. The product contract is a named Err with configured-root exclusivity.
The test now first proves its valid HOME cache resolves, then requires the corrupt
configured root to return Err naming the package. It cannot pass by ignoring the
override. Runtime semantics and coverage policy are unchanged; no platform skips.
The focused regression passed 20 repetitions. Exact new-head CI remains required.

The same Linux run failed only the known #1926 PTY PID-file readiness race
(int on an existing empty file). Raw and stripped CI failures are preserved;
that operator-pending work is not duplicated. One bounded CI retry is authorized
if the identical isolated race recurs; no unrelated failure is waived.

### Submitted-profile retention correction (2026-10-29)

The next Linux full suite passed, but Sonar measured 76.6%, not the unfiltered
local projection: the existing coverage package list includes TestGoFiles only.
The external-only pkgassets_test regression populated XTestGoFiles, so the helper's
measured 83.3% coverage was removed from coverage.out before submission.

The same public Engine regression now runs as an in-package pkgassets test,
with assertions unchanged. TestGoFiles includes engine_test.go. The unchanged
make coverage-from-profile target retains the measured helper profile byte for
byte (cmp passes). No threshold, baseline, exclusion or pipeline changes.
Fresh helper coverage 83.3% passes; exact new-head Sonar remains authoritative.


## Final independent acceptance and integration — 2026-10-29

Round3 accepted exacta24f2ca1 at95/100 after complete native make test/lint/sizes and required remote Linux/Windows gates passed, with actual Sonar94.9% new coverage (unchanged80% threshold). Two optional hosted Mac builds remained queued. PR1849 merged normally at9dc689091. Design and companion plan moved together as implemented/unreleased; original failed tests, projected coverage values and pending-state notes above are dated history, superseded by this measured final result. No historical FAIL artifact was overwritten.

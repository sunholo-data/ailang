# Sprint Plan: M-CACHE-MODULE-ID-ENCODING

> **Execution blocked (iteration 336, D-57):** quorum rejected the naming-scheme direction after the bounded re-quorum. This is an inherited plan, not current execution authorization. M1–M4 remain pending. Before execution, a human ruling and design gate must settle the scheme, then the planner must synchronize the plan and initial snapshot, replacing the overbroad injectivity claim and correcting the M1 mutation example: `Foo`/`foo` collapse without the suffix; `a/b`/`a__b` do not under the clarified slug.

## Summary

The inherited plan proposed replacing the compile-artifact cache's illegal and collision-prone
module-directory mapping with the bounded `m-<slug>-<16hex>` encoding, wiring every production
and test consumer to one encoder, proving Windows legality, removing the temporary Windows skip,
and pinning the existing whole-tree clear behavior for both directory schemes. That direction is
not currently approved; D-57 must be decided and the design gate and planner resynchronization
must complete before this summary becomes executable.

**Target:** v0.36.0  
**Planned at:** 8cd3bc7831e30a1a9f981539bd351acf1d3d70e3 (`std/VERSION` = v0.35.1)  
**Duration:** 1.2 working days  
**Milestones:** 4 inherited pending milestones; commit boundaries require post-D-57 approval

**Risk:** Medium; small implementation, but a cross-package fixture and Windows-only filesystem
evidence make omission risk material  
**Dependencies:** Human D-57 decision, completed design gate, and planner resynchronization

**Design:** `design_docs/planned/v0_36_0/m-cache-module-id-encoding.md`  
**Issue:** None

**Initial sprint state:** `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json`
contains the four pending milestones as a blocked historical snapshot. Do not copy it to
`.ailang/state/sprints/sprint_m-cache-module-id-encoding.json` while blocked. After D-57, the
design gate, and planner resynchronization are complete, the planner must replace this instruction
with the authorized initialization procedure. The tracked file is not live progress.

The inherited M1–M4 decomposition is preserved for historical planning context and is not current
execution authorization. Estimates were deliberately small: M1 0.35 day,
M2 0.35 day, M3 0.30 day, and M4 0.20 day. The seven-day repository diff (523 files, 71,516
insertions, 6,123 deletions) spans unrelated mission lanes and is not a credible velocity sample
for this single-function change. Approximately 220 changed LOC, mostly tests, is the useful
forecasting unit.

## Scope and execution rules

The inherited scope describes the hybrid encoder, production wiring, the discovered test-surface
tail, Windows legality and real-directory evidence, removal of the temporary Windows skip, and a
regression guard for `Clear()` sweeping both naming schemes. It is historical until D-57 selects a
direction and the design gate and planner resynchronization establish current scope. The inherited
exclusions were a cache-key version bump, stamp-schema change, automatic garbage collection, path
canonicalisation change, and adversarial cache hardening.

If execution is authorized after resynchronization, the executor must close one green commit at
each milestone boundary and must not begin the next milestone from a red tree. The controller owns
all git writes. If implementation requires a scope change, stop at the last green boundary and
return to design review.

## Reality checks

All checks were run in this worktree on 2026-09-06. Searches that establish completeness include
known-present controls in the same command.

### RC1 — definition and sole production call site

~~~sh
rg -n 'func sanitizeModuleID|sanitizeModuleID\(' internal cmd --glob '*.go'
sed -n '158,176p' internal/pipeline/cache_store.go
sed -n '398,407p' internal/pipeline/cache_artifacts.go
~~~

Observed:

~~~text
internal/pipeline/cache_artifacts_test.go:67: if sanitizeModuleID("a/b") != sanitizeModuleID("a__b") {
internal/pipeline/cache_store.go:162:func sanitizeModuleID(moduleID string) string {
internal/pipeline/cache_artifacts.go:403: return filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))
~~~

The body maps only `/` and `\` to `__`. The positive `sanitizeModuleID` hits prove the search
instrument is live; after separating the definition and test reference, line 403 is the sole
production call.

### RC2 — tests coupled to the old behavior

~~~sh
rg -n 'sanitizeModuleID|sanitized_collision|NewReplacer\("/", "__"|C:__|modules.*__|a/b.*a__b|a__b.*a/b' \
  . --glob '*_test.go' --glob '*.go'
rg -n 'modules[/\\]|"modules", "[^"]+"' internal cmd --glob '*_test.go'
~~~

Observed old-scheme dependencies that must move in M2:

- `internal/pipeline/cache_artifacts_test.go:64-73` directly asserts that `a/b` and `a__b`
  collide under `sanitizeModuleID`.
- `internal/pipeline/cache_invalidation_test.go:313,328` hard-codes
  `compile/modules/answer` when reading or removing the production stamp.
- `internal/pipeline/cache_artifacts_test.go:363` hard-codes `compile/modules/answer` when
  poisoning the production core blob.
- `cmd/ailang/serve_api_mcp_surface_test.go:602` copies the old mapping with
  `strings.NewReplacer`; its callers at lines 179 and 289 depend on that path.
- The temporary skip and stale explanation are at
  `cmd/ailang/serve_api_mcp_surface_test.go:538-572`; M3 owns their removal/update.

The same search positively found arbitrary orphan fixtures such as `legacy__orphan` in
`cache_store_test.go` and `orphan/nested` in the CLI clear test. Those do not predict a production
module path; they intentionally prove `Clear()` removes arbitrary children and need not change.
No additional test was found that asserts or reconstructs the production old mapping.

### RC3 — base package gate

~~~sh
go test ./internal/pipeline/... ./cmd/ailang/...
~~~

Observed inside the workspace sandbox: `internal/pipeline` passed in 5.209s, then `cmd/ailang`
panicked because `httptest` could not bind `[::1]:0` (`operation not permitted`). That result is
**UNINFORMATIVE UNDER SANDBOX**, not a red repository gate. The exact command was rerun without
the socket restriction and exited 0:

~~~text
ok github.com/sunholo-data/ailang/internal/pipeline (cached)
ok github.com/sunholo-data/ailang/cmd/ailang 28.026s
~~~

The required base gate is therefore GREEN.

## Schedule and commit boundaries

| Day | Milestone | Estimate | Dependency | Exact green boundary |
|---|---|---:|---|---|
| 1 | M1 — pure encoder and unit tests | 0.35 d | none | Commit 1 after `go test ./internal/pipeline/... -count=1` |
| 1 | M2 — production wiring and complete fixture migration | 0.35 d | M1 | Commit 2 after `go test ./internal/pipeline/... ./cmd/ailang/... -count=1` |
| 1 | M3 — Windows legality and skip retirement | 0.30 d | M2 | Commit 3 after `go test ./internal/pipeline/... ./cmd/ailang/... -count=1` |
| 2 | M4 — stale-scheme sweep regression and version note | 0.20 d | M3 | Commit 4 after `go test ./internal/pipeline/... ./cmd/ailang/... -count=1` |

Each command is cumulative because it runs the full affected packages. On a sandbox that forbids
loopback listeners, a `cmd/ailang` bind failure is uninformative and the exact command must be
rerun in an environment that permits loopback before closing the boundary.

## Non-vacuity ledger

A test belongs to a milestone only when reverting the named production hunk makes that test red.
Merely adding a test later does not transfer ownership of an earlier production hunk. The “first
owned mutation” statements below are the review key.

| Milestone | Named acceptance or compatibility test | Exact production mutation killed | First defended here? | Verdict |
|---|---|---|---|---|
| M1 | `TestEncodeModuleDirName_InjectivityAndDeterminism` | In `encodeModuleDirName`, delete the SHA-256 hex suffix and return only `m-` plus the slug; `a/b`/`a__b` and `Foo`/`foo` collapse. | Yes: M1 introduces the encoder hunk and this unit test. | Non-vacuous for M1. |
| M2 | `TestCacheArtifacts_Authorization/sanitized_collision_uses_exact_module_id` (rewritten) | In `moduleArtifactDir`, replace `encodeModuleDirName(moduleID)` with the old separator-only mapping; the two production directories become equal. | Yes: M1's pure test never calls `moduleArtifactDir`; this is the first test of M2 wiring. | Non-vacuous for M2. |
| M2 | `TestServeAPI_DivergentCacheTools` and `TestServeAPI_RouteIfaceMismatchFromCache` | Revert the `moduleArtifactDir` wiring while the fixture continues to call the real exported encoder; the fixture looks in the new directory while publication used the old one. | Yes, as cross-package integration coverage of the same M2 wiring. | Non-vacuous for M2; also prevents a copied encoder. |
| M2 | Existing migration/byte-limit tests whose three `modules/answer` paths are replaced with the real directory resolver | Revert `moduleArtifactDir` to the old mapping. Because these tests derive their path through the same resolver, they remain green. | No. | **VACUOUS for M2 wiring**; compatibility-only. The rewritten collision subtest and `cmd/ailang` integration tests are the non-vacuous coverage. |
| M3 | `TestEncodeModuleDirName_AllLegalOnWindows` | In the M1 encoder, pass `:`/forbidden bytes through, or drop `m-` while restoring `.`; the table fails and Windows `os.MkdirAll` fails. | No: the killed production hunk landed in M1. | **VACUOUS for M3's own diff**, although valuable platform evidence. |
| M3 | `TestCacheArtifacts_WindowsModuleIDPublication` plus the two `requireCompileArtifactCache` callers, with explicit PASS-event enforcement on Windows CI | Revert the M1 forbidden-byte mapping or M2 production wiring; publication produces no usable artifact and the pass-event check rejects a skip/missing pass. | No: both production hunks are earlier. | **VACUOUS for M3's own diff**, but the proposed integration test is non-vacuous for the end-to-end encoder/wiring mutation because it performs actual `StoreArtifacts` for `C:/Users/runneradmin/x` and checks the real encoded directory. Reverting only M3's deleted skip while the encoder remains correct is not observably red; M3 has no new production behavior. |
| M4 | `TestClear_SweepsArtifactDirectories` | In `CacheStore.Clear`, delete `artifactIO.removeAll(filepath.Join(cs.dir, "modules"))`; old- and new-scheme directories remain. | No: `Clear()` and `TestCacheStore_ClearArtifacts` already defend this at base. | **VACUOUS for M4's own diff**; regression/documentation-only. The integration-level form is a real `CacheStore`, persisted manifest, both scheme directories, `Clear()`, reopen, and filesystem assertions, but it still guards pre-existing behavior. No non-vacuous production mutation exists without redesigning M4. |

The executor must not report M3 or M4 as newly proven behavior. Their commits are independently
green validation/documentation boundaries; M1 owns encoding semantics and M2 owns production use.

## M1 — introduce the pure encoder

**Estimate:** 0.35 day; approximately 80 LOC including tests.  
**Dependency:** none.  
**Commit:** one M1 commit after the exact pipeline package gate is green.

### Work

- Add the approved pure `encodeModuleDirName` and slug helper in
  `internal/pipeline/cache_store.go` alongside the temporarily retained `sanitizeModuleID`; do
  not wire `moduleArtifactDir` yet. M2 deletes the old function after switching its caller.
- Implement exactly `m-<slug>-<16hex>`: lowercase ASCII slug alphabet `[a-z0-9_-]`, every other
  input byte (including `.`) mapped to `_`, runs allowed, outer `_` trimmed, 38-byte cap, and the
  first 16 lowercase hex characters of SHA-256 over the full original module ID.
- Add `TestEncodeModuleDirName_InjectivityAndDeterminism` covering the approved worked examples,
  repeated calls, `a/b` versus `a__b`, `Foo` versus `foo`, and the 57-character ceiling. Keep the
  Windows-legality enumeration out of M1 as the approved decomposition requires.

### Boundary

~~~sh
go test ./internal/pipeline/... -count=1
~~~

First mutation defended by M1's own diff: deleting the full-ID hash suffix. The named test must
turn red under that mutation.

## M2 — wire production and migrate every encoding-dependent fixture

**Estimate:** 0.35 day; approximately 55 LOC including test updates.  
**Dependency:** green M1 commit.  
**Commit:** one M2 commit after both affected package trees are green.

### Work

- Point `CacheStore.moduleArtifactDir` at `encodeModuleDirName` and delete the old encoder.
- Expose the real encoder through the narrow test-visible pipeline API needed by the external
  `cmd/ailang` test package; do not retain a second implementation.
- Rewrite the collision subtest to assert distinct production directories first, then preserve
  the exact-module-ID stamp rejection assertion.
- Replace all three hard-coded production `modules/answer` paths at
  `cache_invalidation_test.go:313,328` and `cache_artifacts_test.go:363` with the real directory
  resolver. These are required compatibility edits, not milestone acceptance evidence.
- Replace `compileArtifactDir`'s `strings.NewReplacer` copy at
  `serve_api_mcp_surface_test.go:602` with a call to the real encoder. “Fixed” means the fixture
  invokes production encoding code; matching output from another copy is insufficient.

### Boundary

~~~sh
go test ./internal/pipeline/... ./cmd/ailang/... -count=1
~~~

First mutation defended by M2's own diff: restoring the old mapping at the production
`moduleArtifactDir` call. The rewritten collision integration subtest is the first test that can
turn red for that wiring mutation; M1's unit test cannot.

## M3 — prove Windows legality and retire the skip

> **Added after the iteration-334 judge review (non-blocking finding 3).** M3 MUST also edit the
> hand-maintained no-silent-skip `-run` allow-list in `.github/workflows/ci.yml` to include
> `TestEncodeModuleDirName_AllLegalOnWindows` and `TestCacheArtifacts_WindowsModuleIDPublication`.
> Add `./internal/pipeline` to the package list in BOTH `go test` invocations, add both names to
> BOTH `-run` regexes, and add both names to EACH required-PASS loop. The current commands at
> `ci.yml:111` (unix, 5 names) and `ci.yml:480` (windows, 4 names) include `./cmd/ailang` but omit
> `./internal/pipeline`. Adding the names to both lists without the package would select no
> pipeline tests and make the required-PASS loop fail; adding them only to the regex cannot enforce
> execution. Without all three edits on both platforms, M3's "cannot masquerade as green" property
> remains aspirational rather than enforced, which is the same class of gap as the milestone's own
> vacuity flag.
> (Both line numbers verified first-party by the controller before this note was written; note the
> two lists are ALREADY asymmetric — the unix one carries `TestZ3VerifyEndToEnd` and the Windows
> one does not — so preserve every existing platform-specific package, regex name, and PASS-loop
> entry while adding the pipeline package and the two new names to BOTH gates.)

**Estimate:** 0.30 day; approximately 60 LOC across tests, comments, and planned CI workflow
wiring.
**Dependency:** green M2 commit.  
**Commit:** one M3 commit after both affected package trees are green locally; Windows evidence
remains a remote gate.

### Work

- Add `TestEncodeModuleDirName_AllLegalOnWindows` with the exact approved fixtures, forbidden-byte
  and control-byte checks, no trailing dot/space, and a reserved-name check against the basename
  before the first dot. On Windows, also require actual `os.MkdirAll` success under `t.TempDir()`.
- Add `TestCacheArtifacts_WindowsModuleIDPublication`: store artifacts for
  `C:/Users/runneradmin/x`, verify the directory produced by the real encoder exists, load the
  artifacts back, and emit an explicit named pass event.
- Delete the Windows/empty-manifest skip in `requireCompileArtifactCache`; an empty manifest is now
  a failure everywhere. Replace the stale comment block with the new invariant.
- In both no-silent-skip CI gates, add `./internal/pipeline` to the `go test` package list, add both
  legality/publication test names to the `-run` regex, and add both names to the required-PASS
  loop. Preserve all existing platform-specific entries. The Windows evidence must contain both
  explicit PASS events so a missing test or skip cannot masquerade as green.

### Boundary

~~~sh
go test ./internal/pipeline/... ./cmd/ailang/... -count=1
~~~

M3's first executable diff is test/acceptance infrastructure, not production behavior. Its
production mutations belong to M1/M2, so the milestone is deliberately flagged vacuous with
respect to its own production diff rather than overstated.

## M4 — pin stale-scheme sweeping and document the no-bump decision

**Estimate:** 0.20 day; approximately 25 LOC.  
**Dependency:** green M3 commit.  
**Commit:** one M4 commit after both affected package trees are green.

### Work

- Add `TestClear_SweepsArtifactDirectories` using a real store, a persisted manifest entry, one
  old-scheme directory, and one directory named with `encodeModuleDirName`; call `Clear()`, reopen
  the store, and assert both directories and the persisted/in-memory manifest entries are gone.
- Add the approved note near `cacheKeyVersion` explaining that directory renaming is
  self-invalidating and does not change blob/stamp format, so no version bump is needed.
- Do not duplicate the broader scope/error-ordering arms already present in
  `TestCacheStore_ClearArtifacts`.

### Boundary

~~~sh
go test ./internal/pipeline/... ./cmd/ailang/... -count=1
~~~

There is no M4-owned executable production mutation: the new test guards the pre-existing
`Clear()` removal hunk, and the production edit is explanatory text. This is an intentionally
small regression/documentation boundary.

## CI gates not covered by milestone-local green

The following named profiles are derived from `.github/workflows/ci.yml` at the planning commit.
They are the complete blocking CI surface not established by the milestone boundary commands.
Informational/allowed-to-fail steps are called out separately.

**CI-LINUX-REST (`test` job):** `make deps`; `go mod download all`; `make install`;
`tools/ci/motoko_smoke.sh`; install z3 and jq; network-poisoned `go test -timeout 300s ./...`;
the planned seven-test no-silent-skip PASS-event gate; `make test-parser`; `make test-stdlib-ail`;
`make check-file-sizes`; `make check-boundaries`; `make check-referenced-paths`;
`make check-git-exec`; `make test-check-git-exec`; `make verify-install-guide`;
`make verify-pi-assets`; `make verify-mcp-tools`; `make verify-stdlib`;
`make verify-stdlib-selftest`; `make verify-examples-gate-selftest`;
`make check-protocol-closure`; `make test-check-protocol-closure`;
`make check-tmpfile-hygiene`; `make test-check-tmpfile-hygiene`;
`make check-home-isolation`; `make test-check-home-isolation`;
`make check-no-personal-email`; `make test-check-no-personal-email`; `make check-changelog`;
`make test-check-changelog`; event-specific `make check-autoclose`; `make test-check-autoclose`;
`make test-check-referenced-paths`; `make check-skills`; `make check-context-docs`;
`make test-check-context-docs`; fetch `origin/dev` plus `make check-prompt-freeze`;
`make test-coverage-gate`; `make check-golden-drift`; `make fuzz-parser`;
`make test-lowering`; `make test-imports-success`; `make test-import-errors`;
`make verify-no-shim` with `AILANG_REQUIRE_LOWERING=true`; `make doctor`;
`make test-regression-guards`; `make test-nightly-classifier` plus its 84-pass floor;
`make test-coverage`; compute the Sonar project version with the workflow's tag/SHA fallback; and
`make verify-examples`.

**CI-WINDOWS (`test-windows` job):** module download; `go install ./cmd/ailang` and version smoke;
PowerShell `-args-file` smoke; PowerShell stdin `-args-json -` smoke; network-poisoned
`go test -timeout 300s ./...`; and the planned six-test Windows no-silent-skip PASS-event gate.
For this sprint it must additionally show explicit PASS events for the M3 legality and
real-publication tests; local macOS/Linux green cannot establish the `os.MkdirAll` Windows leg.

**CI-BUILD (`build` job):** `make build` after the Linux test job.

**CI-LINT (`lint` job):** `make fmt-check`; ShellCheck availability;
`make shellcheck-autopush`; `make test-fmt-check`; `make test-shellcheck-autopush`; `make vet`;
`make install-lint`; and `make lint`.

**CI-PLATFORM-AND-SECURITY:** macOS `/bin/bash` 3.x assertion plus
`make test-launchd-drivers`; install pinned govulncheck, build the filter, and run
`govulncheck -format json ./... | ./bin/govulncheck-filter`.

**CI-PUSH-ONLY:** on a push to `dev`, `docs/scripts/sync-prompts.sh` and
`tools/generate-llms-txt.sh`; on a push to `main` or `dev`, dependency submission runs but is
explicitly `continue-on-error` and is not a blocking gate.

The cycle detector, compilation example probe, SonarCloud scan, trace verification, artifact
uploads, and dependency submission are informational or explicitly allowed to fail in this
workflow; they are not represented as blocking acceptance.

Per milestone, the complete uncovered set is:

| Milestone | Full CI set not covered by its exact local `go test` boundary |
|---|---|
| M1 | CI-LINUX-REST, CI-WINDOWS, CI-BUILD, CI-LINT, CI-PLATFORM-AND-SECURITY, CI-PUSH-ONLY |
| M2 | CI-LINUX-REST, CI-WINDOWS, CI-BUILD, CI-LINT, CI-PLATFORM-AND-SECURITY, CI-PUSH-ONLY |
| M3 | CI-LINUX-REST, CI-WINDOWS (especially native legality/publication PASS events), CI-BUILD, CI-LINT, CI-PLATFORM-AND-SECURITY, CI-PUSH-ONLY |
| M4 | CI-LINUX-REST, CI-WINDOWS, CI-BUILD, CI-LINT, CI-PLATFORM-AND-SECURITY, CI-PUSH-ONLY |

## Definition of done

- Four commits, one per approved milestone, each closed only at its exact green package boundary.
- The M1 unit test kills suffix removal; the M2 integration test kills production wiring reversion.
- The `cmd/ailang` fixture calls the real encoder, and all three discovered hard-coded
  `modules/answer` paths are migrated.
- Windows CI reports explicit PASS events for the legality and real-publication tests; no skip can
  satisfy that evidence.
- `Clear()` regression coverage includes old- and new-scheme directories and persisted manifest
  state; the no-version-bump rationale is adjacent to `cacheKeyVersion`.
- All blocking profiles derived above are green in remote CI.
- No issue-closing metadata, schema/version bump, adversarial hardening, garbage collection, or
  unrelated refactor enters the sprint.

## Handoff constraints

- Executor updates only progress fields in the sprint JSON; scope and non-vacuity decisions live
  in this plan.
- The controller owns git writes; the planner performs none.
- Any discrepancy against RC1–RC3 is a finding. Record the exact command/output and revise the
  plan rather than weakening acceptance.

# Sprint Plan: M-SERVEAPI-UNIFY

## Summary
Eliminate the four-column projection drift in serve-api by keying `s.modules` on physical file path and replacing per-file loading with one project-wide compilation pass. Retires the v0.10.7 → v0.10.11 cascade class of bugs structurally instead of documenting the drift.

**Duration:** 3-4 days (~500 LOC changed, net +150 after deletions)
**Dependencies:** None (builds on v0.10.11)
**Risk Level:** Medium — touches the loader/serve-api boundary but is bounded by the v0.10.11 regression tests as a safety net
**Design Doc:** [m-serveapi-unify.md](m-serveapi-unify.md)
**Supersedes:** M4 of [m-mcp-cascade-postmortem-sprint-plan.md](m-mcp-cascade-postmortem-sprint-plan.md)

## Current Status Analysis

### Completed Recently
- v0.10.11: Surgical cold-start fix (unify dep-discovery key, existence check first, skip entry-point, normalize basePath)
- 4 regression tests in `cold_start_test.go` documenting the desired invariants

### Velocity
- Recent average: ~150-300 LOC/release
- This sprint: estimated +150 LOC net, but ~500 LOC of edits across 4 milestones — use the v0.10.11 cold-start tests as the acceptance gate

### Remaining from Design Doc
- M1: `ModuleEntry` type + `registerModule` sole write site (~100 LOC new + tests)
- M2: `findProjectRoots` + `LoadProject` (~150 LOC new + tests)
- M3: Retire dep-discovery loop, migrate projections to read `ModuleEntry` fields (~50 LOC deleted, ~100 LOC edited)
- M4: `docparse_shape` CI fixture + `make bench-cold-start` target (~100 LOC fixture + ~50 LOC test/makefile)
- **Total: ~400 LOC new + ~250 LOC deleted = +150 net**

## Proposed Milestones

### Milestone 1: M1_MODULE_ENTRY — Unified identity type + sole write site
**Goal:** Every module registration goes through one function that computes all projections once from a physical-file identity
**Estimated:** ~100 LOC new + tests
**Duration:** 4-6 hours

**Tasks:**
1. Create `internal/apiserver/module_entry.go` with the `ModuleEntry` struct (PhysicalPath, CanonicalID, DeclaredPath, RelPath, Exports, Iface, File).
2. Keep `ModuleInfo` as a type alias of `ModuleEntry` during the transition (field-compatible) so existing call sites don't break at compile time.
3. Add `Server.normalizedBasePath` field, computed at `New()` via `filepath.EvalSymlinks(filepath.Abs(basePath))`, used by `registerModule`'s under-basePath check. The existing v0.10.11 basePath normalization in `New()` can be hoisted into this field.
4. Implement `Server.registerModule(loaded *loader.LoadedModule) error` as the single write site:
   - Compute `PhysicalPath = EvalSymlinks(Abs(loaded.File.Path))`
   - Under-basePath filter (the only filter — no module-path heuristics)
   - Idempotency check by `PhysicalPath`
   - Populate all projection fields from `loaded` in one pass
   - Call the four `extract*` helpers once each
   - Acquire `s.mu.Lock()` once; release deferred
5. Unit tests in `module_entry_test.go`:
   - Identity computation (with and without symlinks)
   - Projection field correctness (RelPath, CanonicalID, DeclaredPath)
   - Idempotency: two calls with same PhysicalPath return no error, second is a no-op
   - Under-basePath filter: file outside basePath returns `nil` without registering
   - Package file (e.g. `pkg/sunholo/.../x.ail`) is correctly filtered out

**Key Files:**
- `internal/apiserver/module_entry.go` (new)
- `internal/apiserver/module_entry_test.go` (new)
- `internal/apiserver/server.go` (add normalizedBasePath field + hoist v0.10.11 normalization)

**Acceptance Criteria:**
- [ ] `ModuleEntry` struct compiles with all projection fields
- [ ] `registerModule` unit tests pass
- [ ] `ModuleInfo` alias keeps existing call sites compiling
- [ ] `make lint` clean
- [ ] Existing apiserver tests still pass (cold_start_test.go + others)

### Milestone 2: M2_LOAD_PROJECT — Project-wide single-pass loading
**Goal:** Replace per-file `loadFile` with one pipeline invocation over the whole project; unify preload + registration into one loop
**Estimated:** ~150 LOC new + tests
**Duration:** 6-8 hours

**Tasks:**
1. Implement `Server.findProjectRoots() ([]string, error)`:
   - Walk `s.basePath` with `filepath.Walk`, collect all `.ail` files
   - Parse each file's header-only to extract its `import` statements (cheap — lexer-only, no full type check)
   - Build the in-project import graph (edges = imports between files that resolve to the same project)
   - Return files with in-degree zero as roots
   - Fallback: if the computed roots don't cover all `.ail` files (orphan detection), add orphans as additional roots
2. Implement `Server.LoadProject(ctx context.Context) error`:
   - Call `findProjectRoots` to get the root set
   - For each root, call `pipeline.RunWithContext` once; union the resulting `result.Modules` maps by canonical ID (skip already-seen)
   - For each unioned `loaded`, call `s.engine.PreloadModule(modID, loaded)` (and the declared-path variant for module_prefix aliasing)
   - Iterate the unioned map and call `s.registerModule(loaded)` for each — under-basePath filter inside registerModule handles package exclusion
   - Tail: eager-load loop over `s.modules` calling `s.engine.Load(entry.CanonicalID)` once per entry
3. Change `Server.LoadModules([]string) error` to a thin wrapper that ignores its argument (or uses it as an override) and calls `LoadProject`. Preserves the public API contract for external callers.
4. Add tests in `load_project_test.go`:
   - Linear chain fixture (main → lib/a → lib/b): assert 3 roots collapse to 1 (main.ail), all 3 modules registered
   - Multiple independent entry points (two main.ail-style files, no cross-imports): assert 2 roots, both compiled
   - Orphan file fixture (a .ail file that's not imported AND doesn't import anything): assert orphan is registered via the fallback
   - Instrumented compile-count test: wrap the pipeline call in a counter, assert each physical file is compiled at most once across the whole LoadProject pass

**Key Files:**
- `internal/apiserver/server.go` (add LoadProject, findProjectRoots; migrate LoadModules)
- `internal/apiserver/load_project_test.go` (new)

**Acceptance Criteria:**
- [ ] `LoadProject` compiles each physical file at most once (instrumented test)
- [ ] Orphan-file fallback test passes
- [ ] Linear-chain fixture: 5 files → 1 root → 5 registered modules
- [ ] `LoadModules` wrapper preserves the public API (external callers unaffected)
- [ ] `make lint` clean

### Milestone 3: M3_RETIRE_DEP_DISCOVERY — Delete the drift surface
**Goal:** Remove the 90-line dep-discovery loop and migrate all projections to read `ModuleEntry` fields instead of deriving strings locally
**Estimated:** ~50 LOC deleted, ~100 LOC edited
**Duration:** 4-6 hours

**Tasks:**
1. Delete `loadFile` and `loadDirectory` from `server.go` (or shrink to stubs that error with a migration hint — decide based on external caller audit).
2. Delete the 90-line dep-discovery loop (was at `server.go:323-410`).
3. Delete the absBase EvalSymlinks dance inside dep-discovery (no longer needed — `s.normalizedBasePath` is the only normalized basePath, computed once at `New()`).
4. Migrate projection consumers to read `ModuleEntry` fields:
   - `httpRoutes()` / route handler dispatch — use `entry.RelPath` for URL, `entry.CanonicalID` for dispatch
   - `mcpTools()` — use `entry.Exports` and `entry.CanonicalID`
   - `openAPISpec()` — iterate `s.modules` values instead of any parallel map
   - `a2aCard()` — same
   - Startup banner — use `entry.RelPath`
5. Grep audit:
   - `grep -c "dep-discovery" internal/apiserver/` must return 0
   - `grep -rn "TrimPrefix.*basePath\|Rel.*basePath" internal/apiserver/` should only appear at the single registration site in `module_entry.go`
6. Run the full v0.10.11 cold-start test suite. All 4 invariants must pass.

**Key Files:**
- `internal/apiserver/server.go` (major deletion)
- `internal/apiserver/routes.go`, `internal/apiserver/mcp.go`, `internal/apiserver/openapi.go`, `internal/apiserver/a2a.go` (projection consumers)
- `internal/apiserver/cold_start_test.go` (no changes; must still pass)

**Acceptance Criteria:**
- [ ] `grep -c "dep-discovery" internal/apiserver/` returns 0
- [ ] `loadFile` and `loadDirectory` deleted or reduced to migration stubs
- [ ] All v0.10.11 cold-start tests still pass unchanged
- [ ] `make test` passes for the full suite (not just apiserver)
- [ ] `make lint` clean
- [ ] Package file filter still works (package `.ail` files under `pkg/` are not registered as local routes)

### Milestone 4: M4_DOCPARSE_FIXTURE — CI fixture + bench target
**Goal:** A realistic docparse-shaped project that cold-starts in CI, catching the cascade class of bug on every PR
**Estimated:** ~100 LOC fixture + ~50 LOC test/makefile
**Duration:** 3-4 hours

**Tasks:**
1. Create `tests/fixtures/projects/docparse_shape/` with:
   - `ailang.toml` declaring a `module_prefix`-aliased package dependency
   - `main.ail` with a `@route` and an import of `services/mcp_tools`
   - `services/api_keys.ail` declaring `module docparse/services/api_keys` with `@route` and `@auth_required`-style annotations
   - `services/mcp_tools.ail` declaring `module docparse/services/mcp_tools` with `@route`, importing the aliased package
   - `services/csv_parser.ail` as a helper without `@route`
   - `pkg/` vendored package with `module_prefix = "docparse"` in its ailang.toml (empty stub functions — just enough to resolve)
2. Add a test `tests/apiserver_cold_start_fixture_test.go` (or integrate into existing cold_start_test.go):
   - Load the fixture via `LoadProject`
   - Assert: no MOD011 error
   - Assert: exactly N expected modules registered (count them from the fixture)
   - Assert: no duplicate keys in `s.modules`
   - Assert: aliased package file NOT registered as a local route
   - Assert: completes in <5s on CI hardware
3. Add `make bench-cold-start` target in `Makefile`:
   - Builds the binary
   - Runs the fixture test with a wall-clock timer
   - Fails if >5s
4. Hook `make bench-cold-start` into `.claude/skills/release-manager/scripts/pre_release_checks.sh` as a pre-flight check.

**Key Files:**
- `tests/fixtures/projects/docparse_shape/` (new directory)
- `internal/apiserver/cold_start_test.go` (extended with fixture test)
- `Makefile` (new target)
- `.claude/skills/release-manager/scripts/pre_release_checks.sh` (add bench-cold-start)

**Acceptance Criteria:**
- [ ] Fixture loads cleanly via `LoadProject`
- [ ] Fixture test asserts all 4 invariants (no MOD011, exact count, no duplicates, package file filtered)
- [ ] `make bench-cold-start` passes in <5s on local hardware
- [ ] Release-manager pre-flight runs bench-cold-start

### Milestone 5: M5_RELEASE_v0_11_0_OR_v0_10_12 — Ship the refactor
**Goal:** Tagged release with the unified loader
**Estimated:** ~50 LOC docs
**Duration:** 30 min

**Tasks:**
1. Decide: is this a v0.10.12 (patch refactor, backwards-compatible) or v0.11.0 (minor, opens a new sprint)? Default: **v0.10.12** since the public API is preserved via the LoadModules wrapper.
2. Update `changelogs/v0.9-current.md` with a detailed entry referencing both the postmortem and this design doc.
3. Bump `std/VERSION` to the chosen version.
4. Run `update_version_constants.sh`.
5. Run pre-release checks (including the new `make bench-cold-start`).
6. Commit, rebase on origin/dev, tag, push, verify, broadcast.
7. Send message to docparse with the refactor details + postmortem closure.

**Key Files:**
- `changelogs/v0.9-current.md`
- `std/VERSION`
- `docs/src/constants/version.js`

**Acceptance Criteria:**
- [ ] Binary version matches
- [ ] CI builds succeed (all platforms)
- [ ] Tag pushed
- [ ] Broadcast sent
- [ ] docparse notified with "the cascade class of bug is retired structurally" message

## Success Metrics
- Dep-discovery loop deleted (grep returns 0)
- `s.modules` keyed by physical file path, never by derived string
- All v0.10.11 `cold_start_test.go` invariants still hold
- `docparse_shape` fixture cold-starts in <5s on CI
- Net LOC change: roughly +150
- **The projection-key-matrix doc from the cascade plan is NOT created** — its function is absorbed by `ModuleEntry`

## Dependencies
- None — all changes are within `internal/apiserver/` and a new test fixture

## Open Questions
- Should `LoadProject` also handle watch-mode hot-reload? **Deferred** — watch-mode already uses a different path; M-SERVEAPI-UNIFY only covers cold-start.
- Should `findProjectRoots` be promoted to `pipeline.FindProjectRoots`? **Deferred** — keep local until a second consumer appears.
- Do we keep `ModuleInfo` as an alias indefinitely or delete in a follow-up? **Deferred** — alias for v0.10.12; delete in v0.11.0 cleanup if no external breakage.

## Notes
- This sprint is the **"fix the root cause" follow-up** to v0.10.11. If it ships, the postmortem's M4 (matrix doc + grep rule) is moot and should be removed from the cascade sprint plan.
- M3 is the most delicate milestone (deletion of working code). The v0.10.11 cold-start tests are the safety net — if they still pass after M3, the refactor preserves behavior.
- The docparse-shape fixture (M4) is the permanent CI guard. Every future PR touching loader/pipeline/serve-api will run it.

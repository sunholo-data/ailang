# Sprint Plan: M-MCP-CASCADE

## Summary
Fix the v0.10.10 cold-start regression caused by the four-release MCP cascade (v0.10.7 → v0.10.10) and add the process changes that would have prevented it. The immediate fix is small (≤80 LOC); the process work is documentation + a CI fixture.

**Duration:** 1 day (~6 hours actual work)
**Dependencies:** None
**Risk Level:** Low — Fix 1 unifies two existing key derivations; Fixes 2-3 are early-outs in a single loop
**Design Doc:** [m-mcp-cascade-postmortem-and-fixes.md](m-mcp-cascade-postmortem-and-fixes.md)

## Current Status Analysis

### Completed Recently
- v0.10.7: MCP-compliant tool names + `@mcp_name` annotation
- v0.10.8: serve-api package route leak via `module_prefix` aliasing
- v0.10.9: MOD011 module path collision detection (silent dispatch footgun)
- v0.10.10: MOD011 dedupe by resolved file path (regression fix for v0.10.9)

### Velocity
- Recent average: ~150-300 LOC/release across the cascade
- Estimated capacity: ~250 LOC for this sprint (focused, surgical changes)

### Remaining from Design Doc
- M1: Unify dep-discovery key derivation + skip entry-point + reorder existence check (~40 LOC)
- M2: Cold-start regression test (~80 LOC)
- M3: Process changes — projection key matrix doc, rule update, empty-field grep checklist (~150 LOC docs)
- M4: docparse-shape fixture under `tests/fixtures/projects/` (~100 LOC fixtures)
- **Total: ~370 LOC implementation + tests + docs**

The fixture (M4) is sized for v0.11.0; M1/M2 ship in v0.10.11 today.

## Proposed Milestones

### Milestone 1: M1_DEDUP_KEY — Unify dep-discovery key with main path
**Goal:** Eliminate the duplicate `s.modules` registration that doubles eager-load work
**Estimated:** ~40 LOC implementation
**Duration:** 1-2 hours

**Tasks:**
1. In `internal/apiserver/server.go:344-349`, replace the `strings.TrimPrefix(modPath, trimmedBase+"/")` derivation with `filepath.Rel(s.basePath, absFile)` matching the main path at `server.go:406-414`.
2. Move the `existsByKey || existsByNorm` check at `server.go:351-357` to BEFORE the `extract*` calls so repeat visits are O(map lookup), not O(parse + 4 traversals).
3. Add an explicit early-out: `if absFile == absPath { continue }` so dep-discovery doesn't redundantly process the entry-point file (the main path already handles it).

**Key Files:**
- `internal/apiserver/server.go` — three small edits to the dep-discovery loop

**Acceptance Criteria:**
- [ ] dep-discovery and main path produce identical `s.modules` keys for the same file
- [ ] No module appears twice in `s.modules`
- [ ] Linting clean
- [ ] Existing apiserver tests still pass

### Milestone 2: M2_COLD_START_TEST — Regression test for double-registration
**Goal:** Catch the v0.10.10 class of bug in CI without needing Cloud Run
**Estimated:** ~80 LOC test code
**Duration:** 2 hours

**Tasks:**
1. Create `internal/apiserver/cold_start_test.go`.
2. Build a test fixture: `t.TempDir()` containing 5 local `.ail` files with cross-imports (so the dep-discovery loop has multiple modules to walk).
3. Test 1 — `TestLoadModules_NoDuplicateRegistration`: call `LoadModules` once, assert `len(s.modules) == 5` exactly.
4. Test 2 — `TestLoadModules_KeyConsistency`: assert every key in `s.modules` matches the file's `filepath.Rel(basePath, absPath)`-derived form (no canonical-ID-prefixed keys).
5. Test 3 — `TestLoadModules_EagerLoadOncePerModule`: instrument or count how many times the engine is asked to compile each module across the eager-load step. Expect ≤1 per module.

**Key Files:**
- `internal/apiserver/cold_start_test.go` — new test file

**Acceptance Criteria:**
- [ ] All three tests pass on v0.10.11 (the fixed version)
- [ ] At least one of the three tests FAILS when reverted to pre-fix server.go (proves the test catches the regression)
- [ ] Tests run in <5s on local hardware
- [ ] Linting clean

### Milestone 3: M3_RELEASE_v0_10_11 — Ship the fix
**Goal:** v0.10.11 tagged, pushed, broadcast to docparse
**Estimated:** ~50 LOC docs (changelog + commit message)
**Duration:** 30 min

**Tasks:**
1. Update `changelogs/v0.9-current.md` with v0.10.11 entry referencing the postmortem doc and listing the four cascade releases.
2. Bump `std/VERSION` to `v0.10.11`.
3. Run `update_version_constants.sh 0.10.11`.
4. Run pre-release checks (`make test`, `make lint`).
5. Commit, rebase on origin/dev, tag, push.
6. Verify binary version.
7. Send message to docparse with the fix and the postmortem doc reference.

**Key Files:**
- `changelogs/v0.9-current.md`
- `std/VERSION`
- `docs/src/constants/version.js`

**Acceptance Criteria:**
- [ ] `ailang --version` reports `v0.10.11`
- [ ] CI binary build succeeds (Linux, macOS, Windows)
- [ ] Tag pushed to origin
- [ ] Message sent to docparse with fix details + postmortem link
- [ ] Changelog references the postmortem doc

### Milestone 4: M4_PROCESS_CHANGES — Projection matrix + rule updates (deferred to v0.11.0)
**Goal:** Make the cascade pattern impossible to repeat without surfacing it
**Estimated:** ~150 LOC docs + ~100 LOC fixture
**Duration:** Deferred — separate v0.11.0 sprint

**Tasks (planned, not executed in this sprint):**
1. Create `docs/docs/internal/projection-key-matrix.md` with the four-column table from the design doc.
2. Update `.claude/rules/api-server.md` with the design-doc gate rule for changes touching loader/pipeline/serve-api.
3. Build `tests/fixtures/projects/docparse_shape/` mirroring docparse's layout.
4. Add `make bench-cold-start` target hooked into release-manager pre-flight.

**Key Files (deferred):**
- `docs/docs/internal/projection-key-matrix.md` (new)
- `.claude/rules/api-server.md`
- `tests/fixtures/projects/docparse_shape/` (new directory)
- `Makefile`
- `.claude/skills/release-manager/scripts/pre_release_checks.sh`

**Acceptance Criteria (for v0.11.0 sprint):**
- [ ] Projection key matrix doc exists and is linked from the api-server rule
- [ ] Rule update enforces design doc requirement for the loader/pipeline/serve-api triangle
- [ ] docparse-shape fixture compiles and serves under `serve-api --routes-only`
- [ ] `make bench-cold-start` runs as part of release-manager pre-flight

## Success Metrics
- docparse cold-start back to ≤25s on Cloud Run (verified by reporter)
- Three new regression tests catching the duplicate-registration class
- v0.10.11 tagged and released
- Postmortem doc linked from changelog
- Process changes scoped for v0.11.0 follow-up sprint

## Dependencies
- None — all changes are in `internal/apiserver/server.go` and a new test file

## Open Questions
- Should `loader.Preload` also fail loudly on physically-distinct cache collisions (defense in depth)? **Deferred to v0.11.0** — out of scope for the cold-start fix.
- Should dep-discovery be replaced by a project-wide filesystem walk? **Deferred** — bigger refactor.

## Notes
- The fix is **not** about MOD011 itself. MOD011 is correct as of v0.10.10. The bug is dormant code in the dep-discovery loop that became live when `ast.File.Path` started being populated.
- Fix 1 is the core change. Fixes 2 and 3 are defensive optimizations that compound the benefit.
- The cold-start regression test is the most important deliverable — it makes the cascade-class of bug catchable in CI going forward.
- M4 (process changes) is intentionally deferred. Shipping v0.10.11 quickly matters more than batching the process work into the same release.

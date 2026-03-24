# Sprint Plan: M-PKG-LOCK-PORTABILITY

## Summary
Make ailang.lock portable by removing absolute cache paths for registry/git packages and resolving them at runtime. Fixes Docker deployment workflow.

**Duration:** 0.5 days (~4 hours)
**Dependencies:** None
**Risk Level:** Low (well-scoped, backward compatible)

## Current Status Analysis

### Completed Recently
- M-PKG-INSTALL-LATEST: 160 LOC in 1 hour (install @latest support)
- M-OBS-RETENTION: ~300 LOC in 2 hours (observatory DB retention)
- M-SERVE-API-GET-ARGS: ~100 LOC in 30 min (GET query params)

### Velocity
- Recent average: ~200 LOC/hour for focused pkg work
- Estimated capacity: ~50 LOC needed (this is a small refactor)

### Remaining from Design Doc
- M1: Runtime path resolution in loader.go (~20 LOC)
- M2: Stop persisting paths in resolver.go (~10 LOC)
- M3: Backward compat tests + Docker verification (~20 LOC)

## Proposed Milestones

### Milestone 1: M1_RUNTIME_RESOLVE — Runtime Path Resolution in Loader
**Goal:** Make `packageDir()` compute registry/git cache paths at runtime instead of reading stored `Path`
**Estimated:** 20 LOC implementation + 15 LOC tests = 35 LOC
**Duration:** 1.5 hours

**Tasks:**
- Modify `loader.go:packageDir()` for `case "registry"`: call `CachedPackagePath(name, version)` instead of using `locked.Path`
- Modify `loader.go:packageDir()` for `case "git"`: call `NewGitCache().CacheDir(locked.GitURL)` + append `locked.GitSubdir`
- Keep `locked.Path` as fallback for backward compat with old lock files
- Write test: loader resolves registry package when `Path` is empty in lock entry
- Write test: loader resolves git package when `Path` is empty in lock entry
- Write test: loader still works with old-format lock entry that has `Path` set

**Acceptance Criteria:**
- [ ] `packageDir()` returns correct path for registry packages without stored `Path`
- [ ] `packageDir()` returns correct path for git packages without stored `Path`
- [ ] Old lock files with stored `Path` still work (backward compat)
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- GitCache.CacheDir needs GitURL which may be empty for old entries — Mitigation: fall back to stored Path

### Milestone 2: M2_OMIT_PATHS — Stop Persisting Paths in Lock File
**Goal:** Remove `Path` from `ResolvedPackage` for registry and git sources so lock files are portable
**Estimated:** 10 LOC implementation + 10 LOC tests = 20 LOC
**Duration:** 1 hour

**Tasks:**
- In `resolver.go`, set `Path: ""` for registry packages (line ~228) — keep the local var for resolver's own use
- In `resolver.go`, set `Path: ""` for git packages (line ~171) — same pattern
- Verify `Path` is still set for `source: "path"` dependencies (no change needed)
- Write test: `ResolveDependencies` returns empty `Path` for registry packages
- Write test: generated lock file JSON has no `path` field for registry/git (omitempty)
- Write test: lock file still has `path` for local path dependencies

**Acceptance Criteria:**
- [ ] `ailang lock` produces lock file with no `path` for registry/git entries
- [ ] `ailang lock` still includes `path` for local path deps
- [ ] Round-trip: lock → load → resolve works with no stored paths
- [ ] All tests pass
- [ ] Linting clean

**Risks:**
- Resolver internally needs the path during resolution (for transitive deps) — Mitigation: keep local var, just don't put it in ResolvedPackage

### Milestone 3: M3_VERIFY_PORTABLE — End-to-End Portability Verification
**Goal:** Verify the Docker workflow and update docs
**Estimated:** 5 LOC + changelog = ~10 LOC
**Duration:** 30 min

**Tasks:**
- Test: generate lock file, verify no absolute paths in JSON output
- Test: clear cache, regenerate cache via `ailang install`, verify loader works from lock file alone
- Update CHANGELOG with the fix
- Update design doc status

**Acceptance Criteria:**
- [ ] Lock file contains zero absolute paths for registry/git packages
- [ ] `COPY ailang.toml ailang.lock` into fresh env + `ailang install` works
- [ ] CHANGELOG updated
- [ ] All tests pass

**Risks:**
- None (verification only)

## Success Metrics
- Lock file has no absolute paths for registry/git deps
- All 67 test packages pass
- Backward compat with existing lock files confirmed
- Docker workflow works without re-resolving

## Open Questions
- Should `ailang lock` auto-download missing packages? (Deferred — separate feature)
- Should we warn when loading old lock files with absolute paths? (No — silent compat)

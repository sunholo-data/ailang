# M-PKG-LOCK-PORTABILITY: Make ailang.lock Portable Across Machines and Containers

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (DX blocker for Docker/CI deployments)
**Estimated**: 0.5 days
**Dependencies**: None
**Bug Report**: msg_20260324_155618_e72dcce5 (from docparse)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Lock file becomes machine-independent — same lock file, same build, anywhere |
| A2: Replayability | +1 | Builds replay correctly in Docker/CI without re-resolving |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Lock file works in automated pipelines without workarounds |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | +1 | Clear errors when cache missing: "run ailang lock to download" |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Improves determinism — removes machine-specific state from lock file
- [x] A2 (Replayability): Improves replayability — lock file works on any machine

## Problem Statement

### The Bug

`ailang.lock` stores **absolute paths** to the local package cache for registry and git dependencies:

```json
{
  "name": "sunholo/auth",
  "version": "0.1.1",
  "source": "registry",
  "path": "/Users/mark/.ailang/cache/registry/sunholo/auth/0.1.1"
}
```

This path is machine-specific. Copying the lock file to another machine or Docker container fails because `/Users/mark/.ailang/cache/...` doesn't exist there.

### Current Workaround

Docker builds must skip the lock file and re-resolve:
```dockerfile
COPY ailang.toml .
RUN ailang lock    # re-resolves everything from scratch
```

This defeats the purpose of a lock file — it should guarantee reproducible builds without re-resolution.

### Root Cause

In `resolver.go`, absolute paths are computed by `CachedPackagePath()` (registry) and `GitCache.Resolve()` (git) and stored directly in `ResolvedPackage.Path`. This path is then written verbatim to the lock file via `lockfile.Save()`.

| Source | Path Set At | Value |
|--------|------------|-------|
| `registry` | `resolver.go:228` | `CachedPackagePath()` → `/Users/mark/.ailang/cache/registry/vendor/name/version` |
| `git` | `resolver.go:171` | `GitCache.Resolve()` → `/Users/mark/.ailang/cache/git/<hash>` |
| `path` | `resolver.go:127` | `filepath.Abs()` → absolute local path (correctly stored) |

### Impact

- Docker deployments require re-resolution (defeats lock file purpose)
- CI builds on different machines may resolve differently if registry changes between runs
- Lock file diffs show noise from different developer home directories
- Git-committed lock files contain user-specific paths

## Solution Design

### Key Insight

For `registry` and `git` sources, the cache path is **deterministically computable** from fields already in the lock file:

- **Registry**: `CachedPackagePath(name, version)` → `~/.ailang/cache/registry/vendor/name/version`
- **Git**: `GitCache.CacheDir(git_url)` + `git_subdir` → `~/.ailang/cache/git/<hash>/subdir`

There is no need to store these paths. They should be resolved at runtime.

### Changes

#### 1. Stop persisting Path for registry/git in lock file (`resolver.go`)

```go
// Before building ResolvedPackage for registry sources:
// Path: cachePath,  ← REMOVE THIS

// For git sources:
// Path: localPath,  ← REMOVE THIS
```

Only `source: "path"` dependencies keep `Path` in the lock file (it's the relative/absolute path from the manifest).

#### 2. Resolve Path at runtime in loader (`loader.go`)

In `packageDir()`, compute cache paths on-the-fly:

```go
case "registry":
    dir, err := CachedPackagePath(locked.Name, locked.Version)
    if err != nil {
        return "", fmt.Errorf("failed to compute cache path for %s: %w", locked.Name, err)
    }
    if _, err := os.Stat(dir); err != nil {
        return "", fmt.Errorf("registry package %s not cached; run 'ailang install %s@%s'",
            locked.Name, locked.Name, locked.Version)
    }
    return dir, nil

case "git":
    cache, err := NewGitCache()
    if err != nil {
        return "", err
    }
    dir := cache.CacheDir(locked.GitURL)
    if locked.GitSubdir != "" {
        dir = filepath.Join(dir, locked.GitSubdir)
    }
    if _, err := os.Stat(dir); err != nil {
        return "", fmt.Errorf("git package %s not cached; run 'ailang lock' to fetch",
            locked.Name)
    }
    return dir, nil
```

#### 3. Backward compatibility for existing lock files

The loader should still accept `Path` if present (for old lock files), but prefer runtime resolution:

```go
case "registry":
    // Compute path at runtime (portable)
    dir, err := CachedPackagePath(locked.Name, locked.Version)
    if err != nil {
        // Fallback to stored path for old lock files
        dir = locked.Path
    }
```

### Files to Modify

- `internal/pkg/resolver.go` — Stop setting `Path` for registry/git (~4 LOC removed)
- `internal/pkg/loader.go` — Resolve registry/git paths at runtime (~20 LOC changed)
- `internal/pkg/loader_test.go` — Update tests for runtime resolution (~15 LOC)
- `internal/pkg/resolver_test.go` — Verify lock file has no Path for registry/git (~10 LOC)

### Implementation Plan

**Phase 1: Runtime path resolution** (~2 hours)
- [ ] Modify `loader.go:packageDir()` to compute registry/git paths at runtime
- [ ] Keep backward compat: accept stored `Path` as fallback
- [ ] Tests: loader resolves registry package without stored Path

**Phase 2: Stop persisting paths** (~1 hour)
- [ ] Modify `resolver.go` to omit `Path` for registry and git sources
- [ ] Tests: generated lock file has no `path` field for registry/git entries
- [ ] Tests: generated lock file still has `path` for local path deps

**Phase 3: Verify Docker workflow** (~30 min)
- [ ] Create test: generate lock file, clear cache, restore cache, load succeeds
- [ ] Verify `ailang lock` + `COPY ailang.lock` Docker workflow works
- [ ] Update CHANGELOG

## Success Criteria

- [ ] `ailang.lock` contains no absolute paths for registry or git packages
- [ ] `COPY ailang.lock` into Docker works (after `ailang install` populates cache)
- [ ] Existing lock files with stored paths continue to work (backward compat)
- [ ] `path` source dependencies still store their path in the lock file
- [ ] All existing tests pass
- [ ] Lock file diffs between developers show no path noise

## Non-Goals

- Automatically downloading missing packages on `ailang run` (separate feature)
- Migrating existing lock files to new format (old format still works)
- Changing path dependency behavior (path deps are inherently machine-specific)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Old lock files stop working | Med | Backward compat: accept stored `Path` as fallback |
| Cache miss at runtime with no error context | Low | Clear error: "run ailang install X@Y" |
| Git cache dir hash changes between systems | None | Hash is deterministic from URL (sha256) |

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24

# Fix: ailang lock Transitive Registry Dependency Resolution

**Status**: Planned
**Target**: v0.10.0
**Priority**: P0 — Blocks publishing packages with transitive deps
**Estimated**: 2 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Makes lock resolution deterministic for registry packages |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Enables lock verification for transitive deps |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Fixes machine-driven dependency resolution |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Enables composable multi-package graphs |
| A11: Structured Failure | +1 | Fails loudly with clear error instead of silent path miss |
| A12: System Boundary | 0 | Registry boundary unchanged |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Fixes machine dependency resolution

## Problem Statement

`ailang lock` fails when resolving transitive dependencies from registry packages.

**Current State:**
- When package A depends on registry package B@0.1.0, B is downloaded to `~/.ailang/cache/registry/vendor/B/0.1.0/`
- If B's cached `ailang.toml` contains path deps (e.g., `C = { path = "../config" }`), the resolver tries to resolve `../config` relative to the cache directory
- This produces paths like `~/.ailang/cache/registry/vendor/B/config/` which don't exist
- Packages published before commit a17ebbac (path→registry rewrite) still have path deps in their tarballs

**Impact:**
- Blocks publishing: billing_store, billing_service_api, docparse_access_gate (all have transitive deps)
- Published without transitive deps: billing_entitlements, billing_stripe, billing_proposals, firestore

## Goals

**Primary Goal:** Make `ailang lock` correctly resolve transitive dependencies from registry packages.

**Success Metrics:**
- `ailang lock` resolves A → B@registry → C@registry chains
- Path deps inside cached registry packages are auto-converted to registry lookups
- Existing tests continue to pass

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Convert path deps to registry in resolver vs. reject them | Determines backward compat with already-published packages | agent | design | low |
| Use registry index for version lookup | Only way to get version when path dep can't be read locally | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Convert path deps in registry packages to registry lookups (agent decided: yes, for backward compat)
- [x] Use latest version from registry index when version unknown (agent decided: yes)

## Solution Design

### Overview

Add a `fromRegistry` flag to the resolver's recursive `resolve()` function. When resolving dependencies of a registry-downloaded package and encountering a path dep, automatically convert it to a registry dep by looking up the package name in the registry index.

### Architecture

**The fix is localized to one function in one file:**

In `internal/pkg/resolver.go`, the `resolve` closure gains a `fromRegistry bool` parameter:

1. Normal path deps (fromRegistry=false): resolved as today — relative to `dir`
2. Path deps inside registry packages (fromRegistry=true): converted to registry lookups
   - Use dep name to look up latest version in registry index
   - Download from registry, cache, and recurse with fromRegistry=true
3. Registry deps: always download from registry, recurse with fromRegistry=true

### Implementation Plan

**Phase 1: Fix resolver** (~1 hour)
- [ ] Add `fromRegistry` parameter to `resolve` closure
- [ ] When `fromRegistry && dep.Path != ""`, convert to registry lookup
- [ ] Pass `fromRegistry=true` when recursing into registry packages
- [ ] Create a single `RegistryClient` instance shared across resolution

**Phase 2: Tests** (~30 min)
- [ ] Add `TestResolveDependencies_TransitiveRegistryDeps` using mock registry
- [ ] Test: A(path) → B(registry) → C(registry) chain
- [ ] Test: registry package with path dep gets auto-converted

### Files to Modify/Create

**Modified files:**
- `internal/pkg/resolver.go` — Add fromRegistry flag, path→registry conversion (~30 LOC)

**No new files needed.**

## Examples

### Example: Transitive Registry Resolution

**Before (broken):**
```
app depends on sunholo/billing_store = "0.1.0"
  → downloads to cache
  → billing_store's ailang.toml has: sunholo/firestore = { path = "../firestore" }
  → resolver looks for ~/.ailang/cache/registry/sunholo/billing_store/firestore/
  → ERROR: directory not found
```

**After (fixed):**
```
app depends on sunholo/billing_store = "0.1.0"
  → downloads to cache
  → billing_store's ailang.toml has: sunholo/firestore = { path = "../firestore" }
  → resolver detects fromRegistry=true + path dep
  → looks up sunholo/firestore in registry index → version 0.1.0
  → downloads sunholo/firestore@0.1.0 from registry
  → recurses into firestore's deps with fromRegistry=true
  → SUCCESS
```

## Success Criteria

- [x] `ailang lock` resolves transitive registry deps
- [ ] Path deps in cached registry packages auto-convert to registry lookups
- [ ] All existing tests passing
- [ ] New test for transitive registry dep chain

## Testing Strategy

**Unit tests:**
- Mock registry with A→B→C chain where B has path dep to C
- Verify all three packages resolved correctly

**Manual testing:**
- `ailang lock` on a package depending on sunholo/billing_store

## Deferred Decisions

- Version conflict resolution (multiple versions of same dep) — future work
- Whether to warn when converting path deps in registry packages — agent may choose

## Non-Goals

- Semver range resolution — out of scope, use exact versions
- Git dependency transitive resolution — already works
- Re-publishing packages with fixed manifests — separate task

## Related Documents

**Implemented (may inform design):**
- [m-dx-app-package-adoption.md](design_docs/implemented/v0_9_11/m-dx-app-package-adoption.md)
- [m-pkg-msg-package-messaging-graph.md](design_docs/implemented/v0_9_9/m-pkg-msg-package-messaging-graph.md)

**Planned (check for overlap):**
- [m-dx-package-check.md](design_docs/planned/v0_10_0/m-dx-package-check.md)
- [m-serve-api-transitive-imports.md](design_docs/planned/v0_9_4/m-serve-api-transitive-imports.md)

## References

- [Design Axioms](/docs/references/axioms)
- Agent message: 79986b31 (BUG: ailang lock fails on transitive registry dependencies)
- Commit a17ebbac (Publish: rewrite path deps to registry versions in tarball)

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24

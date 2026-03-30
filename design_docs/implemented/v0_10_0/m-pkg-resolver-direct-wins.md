# M-PKG-RESOLVER-DIRECT-WINS: Direct Dependencies Override Transitive Versions

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (DX pain — forces unnecessary republishing of entire dep chains)
**Estimated**: 1 day
**Dependencies**: None
**Bug Report**: msg_20260324_160522_abe5dddf (from docparse)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes Go map iteration nondeterminism from resolution order |
| A2: Replayability | 0 | Lock file still pins exact versions |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Predictable resolution without manual republishing |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Packages compose without version deadlocks |
| A11: Structured Failure | +1 | Clear errors on incompatible version conflicts |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Improves determinism — direct deps always win regardless of traversal order
- [x] A3 (Effects): No effect changes

## Problem Statement

### The Bug

The resolver uses **first-writer-wins** with `resolvedSet` keyed by package name only (`resolver.go:39`). Whichever version is encountered first during depth-first traversal wins — there is no version comparison or conflict detection.

**Scenario from docparse:**
```
ailang.toml:
  "sunholo/firestore" = "0.2.0"          # direct dep (user wants this version)
  "sunholo/docparse_access_gate" = "0.1.0"

Transitive chain:
  docparse_access_gate@0.1.0 → billing_store@0.2.0 → firestore@0.1.0
```

**What happens:**
1. Resolver iterates `m.Dependencies` (Go map — **nondeterministic order**)
2. If `docparse_access_gate` is visited first, it recursively resolves `billing_store → firestore@0.1.0`
3. `resolvedSet["sunholo/firestore"] = true` — marks 0.1.0 as resolved
4. When the resolver gets to direct dep `firestore@0.2.0`, it checks `resolvedSet["sunholo/firestore"]` → already true → **silently skipped**
5. Lock file contains `firestore@0.1.0` instead of `0.2.0`

**Workaround:** User had to republish `billing_store@0.3.0` (with `firestore@0.2.0`) and `docparse_access_gate@0.2.0` (with `billing_store@0.3.0`) — cascading republishes just to bump one dep.

### Root Causes

1. **`resolvedSet` is keyed by name only** — no version tracking (`resolver.go:39`)
2. **No version comparison** — the resolver has zero calls to semver functions during resolution
3. **Go map iteration order is nondeterministic** — `for depName, dep := range m.Dependencies` makes resolution order unpredictable
4. **No conflict detection** — silently picks whichever version arrives first
5. **Depth-first traversal** means transitive deps are resolved before their parents

### Existing Infrastructure

- `ParseSemver()` and `semverTuple.gte()` exist in `version_compat.go` — can be reused for version comparison
- `resolvedSet` tracks names only, but `resolved[]` stores full `ResolvedPackage` with versions
- Lock file deduplicates by name via sort (`lockfile.go:48-69`)
- Design doc `m-pkg-transitive-lock-fix.md` explicitly deferred "version conflict resolution" as future work
- `m-pkg-install-latest.md` confirms: "AILANG uses flat deps" (one version per package name)

## Design Philosophy

The current behavior is an **implementation artifact, not the desired package semantics**.

In a flat exact-version dependency model, the resolver should **never silently replace** a root direct dependency with a transitive one. The manifest is the declared intent — if it says `firestore = "0.2.0"`, the lock file must contain `firestore@0.2.0` or the resolver must fail with a structured error. Anything else makes the manifest untrustworthy, breaks agent reasoning from root constraints, and starts the slide into dependency hell.

### Classification

| Aspect | Status |
|--------|--------|
| Flat single-version resolution | **Not a bug** — valid design choice |
| Direct dep silently overridden by transitive | **Bug** — violates explicit dependency semantics |
| No conflict surfaced | **Bug** — violates A11 (Structured Failure) |

## Solution Design

### Approach: Fail on Conflict, Direct Deps Are Authoritative

**Resolution precedence:**
1. Direct dependencies of the root package are **authoritative**
2. Transitive dependencies must **unify** with those direct pins
3. If they cannot unify, resolution **fails with a typed conflict**
4. The user must then upgrade or republish the transitive package chain

### Key Design Decision: Structured Failure, Not Silent Override

AILANG uses **flat dependencies** (one version per package name in the lock file). When a version conflict occurs:

| Conflict Type | Policy | Rationale |
|--------------|--------|-----------|
| Direct dep X@0.2.0, transitive wants X@0.1.0 | **Fail with structured conflict** | User's direct pin is authoritative; transitive chain must be updated |
| Direct dep X@0.1.0, transitive wants X@0.2.0 | **Fail with structured conflict** | User must explicitly upgrade or the transitive chain is incompatible |
| Two transitive deps want X@0.1.0 and X@0.2.0 | **Fail with structured conflict** | Ambiguous — user must pin a version in root ailang.toml |
| Same version from multiple sources | **No conflict** | Silently deduplicate |

### Error Format

```
version conflict: sunholo/firestore
  root requires: 0.2.0
  transitive requires: 0.1.0 via sunholo/billing_store@0.2.0

resolution aborted

suggestion:
  - republish sunholo/billing_store against sunholo/firestore@0.2.0
  - or change root dependency to sunholo/firestore@0.1.0 explicitly
```

This is AI-legible, structured, and actionable. Agents can parse the conflict and suggest the correct fix.

### Implementation

#### 1. Sort direct deps for deterministic iteration (`resolver.go`)

Replace `for depName, dep := range m.Dependencies` with sorted iteration:

```go
// Sort dependency names for deterministic resolution order
depNames := make([]string, 0, len(m.Dependencies))
for name := range m.Dependencies {
    depNames = append(depNames, name)
}
sort.Strings(depNames)

for _, depName := range depNames {
    dep := m.Dependencies[depName]
    // ... existing resolution logic
}
```

#### 2. Track resolved versions, not just names (`resolver.go`)

Change `resolvedSet` from `map[string]bool` to `map[string]string` (name → version):

```go
resolvedSet := make(map[string]string) // name → resolved version

// At resolution points (lines 114, 158, 182):
if existingVersion, already := resolvedSet[depName]; already {
    // Conflict detection
    if existingVersion != dep.Version {
        // Apply policy (see below)
    }
    continue
}
```

#### 3. Collect root direct deps before resolution

Before calling `resolve()`, build a map of the root manifest's direct dep versions:

```go
// Collect root direct dependency versions (authoritative)
directDeps := make(map[string]string) // name → version
for depName, dep := range manifest.Dependencies {
    if dep.Version != "" {
        directDeps[depName] = dep.Version
    }
}
```

#### 4. Conflict detection in resolve()

When a package is already in `resolvedSet` with a different version, check against direct deps and fail:

```go
if existingVersion, already := resolvedSet[depName]; already {
    if existingVersion == requestedVersion {
        continue // Same version, no conflict
    }

    // Version conflict detected — build structured error
    return &VersionConflictError{
        Package:            depName,
        DirectVersion:      directDeps[depName],  // may be empty if not a direct dep
        ExistingVersion:    existingVersion,
        RequestedVersion:   requestedVersion,
        RequestedBy:        name,
        ExistingPath:       resolvedSet[depName], // could track which package resolved it
    }
}
```

#### 5. Structured error type

```go
// VersionConflictError is returned when the dependency graph has incompatible
// version requirements for the same package.
type VersionConflictError struct {
    Package          string
    DirectVersion    string // root manifest's version (empty if not a direct dep)
    ExistingVersion  string // version already resolved
    RequestedVersion string // conflicting version
    RequestedBy      string // package that requested the conflicting version
}

func (e *VersionConflictError) Error() string {
    var b strings.Builder
    fmt.Fprintf(&b, "version conflict: %s\n", e.Package)
    if e.DirectVersion != "" {
        fmt.Fprintf(&b, "  root requires: %s\n", e.DirectVersion)
    }
    fmt.Fprintf(&b, "  already resolved: %s\n", e.ExistingVersion)
    fmt.Fprintf(&b, "  transitive requires: %s (via %s)\n", e.RequestedVersion, e.RequestedBy)
    b.WriteString("\nresolution aborted\n\nsuggestion:\n")
    if e.DirectVersion != "" {
        fmt.Fprintf(&b, "  - republish %s against %s@%s\n", e.RequestedBy, e.Package, e.DirectVersion)
        fmt.Fprintf(&b, "  - or change root dependency to %s@%s explicitly\n", e.Package, e.ExistingVersion)
    } else {
        fmt.Fprintf(&b, "  - pin %s in root ailang.toml to resolve ambiguity\n", e.Package)
    }
    return b.String()
}
```

#### 6. Post-resolution validation against root manifest

After `resolve()` completes, verify all direct deps match:

```go
// Verify direct deps are authoritative — no silent downgrades
for depName, directVersion := range directDeps {
    if resolvedVersion, ok := resolvedSet[depName]; ok && resolvedVersion != directVersion {
        return nil, &VersionConflictError{
            Package:         depName,
            DirectVersion:   directVersion,
            ExistingVersion: resolvedVersion,
            RequestedBy:     "(transitive)",
        }
    }
}
```

### Files to Modify

- `internal/pkg/resolver.go` — Sorted iteration, version tracking, `VersionConflictError`, post-resolution validation (~60 LOC)
- `internal/pkg/resolver_test.go` — Tests for conflict detection, direct vs transitive, same-version dedup (~60 LOC)

### Implementation Plan

**Phase 1: Deterministic iteration order** (~30 min)
- [ ] Sort `m.Dependencies` keys before iteration in `resolve()`
- [ ] Verify no test regressions from ordering change
- [ ] This alone fixes the nondeterminism (A1 improvement)

**Phase 2: Version conflict detection** (~2 hours)
- [ ] Change `resolvedSet` from `map[string]bool` to `map[string]string` (name → version)
- [ ] Add `VersionConflictError` type with structured error message
- [ ] Detect version mismatches when same package resolved twice
- [ ] Return `VersionConflictError` instead of silently skipping
- [ ] Tests for conflict detection

**Phase 3: Post-resolution validation** (~1.5 hours)
- [ ] Collect root direct dep versions before resolution
- [ ] After resolution, verify all direct deps match resolved versions
- [ ] If mismatch, return `VersionConflictError` with suggestion
- [ ] Tests for direct dep vs transitive conflict

**Phase 4: Edge cases + docs** (~1 hour)
- [ ] Test: same version from multiple sources (no conflict, silent dedup)
- [ ] Test: two transitive deps disagree (structured error)
- [ ] Test: direct dep matches transitive (no error)
- [ ] CHANGELOG update

## Effort Estimate

| Phase | LOC | Time |
|-------|-----|------|
| P1: Deterministic iteration | 10 | 30 min |
| P2: Conflict detection + error type | 40 | 2 hours |
| P3: Post-resolution validation | 30 | 1.5 hours |
| P4: Edge cases + docs | 20 | 1 hour |
| **Total** | **~100** | **~5 hours (1 day)** |

**Risk level:** Medium — the resolver is a critical path, and changes affect all package resolution. Comprehensive tests required.

## Success Criteria

- [ ] `ailang lock` fails with structured `VersionConflictError` when direct dep conflicts with transitive
- [ ] Error message includes: conflicting versions, which package introduced the conflict, and actionable suggestion
- [ ] Same version from multiple sources resolves silently (no false conflict)
- [ ] Go map iteration order no longer affects resolution (deterministic via sort)
- [ ] All existing resolver tests pass
- [ ] Error is parseable by agents (structured format, not prose)

## Non-Goals

- Multiple versions of same package in lock file (npm-style) — AILANG stays flat
- Semver range resolution in transitive deps — versions are exact
- Silent override of any version (the whole point is to fail loudly)
- Automatic conflict resolution — user or agent must act on the error
- Lock file migration — old lock files remain valid

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Sorting changes break existing resolution order | Med | Existing tests verify behavior, not order |
| Re-downloading registry package at different version | Low | Same download/hash logic, just different version |
| Direct dep is genuinely incompatible with transitive dep | Med | Warn clearly; long-term: interface hash comparison |
| Performance overhead from post-resolution scan | None | Scan is O(direct_deps × resolved), both small |

## Related Documents

- [m-pkg-transitive-lock-fix.md](m-pkg-transitive-lock-fix.md) — Deferred "version conflict resolution" as future work (line 155)
- [m-pkg-install-latest.md](m-pkg-install-latest.md) — "AILANG uses flat deps" (line 194)
- [m-pkg-lock-portability.md](m-pkg-lock-portability.md) — Lock file portability (just shipped)

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24

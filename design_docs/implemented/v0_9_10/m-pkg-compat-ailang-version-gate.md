# M-PKG-COMPAT: AILANG Version Compatibility Gate for Packages

**Status**: Planned
**Target**: v1.0.0
**Priority**: P0 — critical gap; packages already drifting without version enforcement
**Estimated**: 1–2 days
**Dependencies**:
- Package system (implemented — manifests, lockfiles, resolver)
- Registry validator (implemented — already records `ailang_version` in metadata)
- Package install/publish CLI (implemented)

---

## Motivation

This was missed during the original package system design. The `edition` field exists in `ailang.toml` but is never checked against the compiler version. The registry validator already records which AILANG version compiled a package (`ValidationResult.AILANGVersion`) but consumers never see or enforce it.

**The gap is already causing drift**: packages published against v0.9.5 may silently break on v0.10 if import resolution, type inference, or stdlib APIs change. There is no mechanism to say "this package requires AILANG >= 0.9.5" and no check on install or compile.

### How Other Languages Solve This

| Language | Mechanism | Enforcement |
|----------|-----------|-------------|
| **Rust** | `edition = "2021"` in Cargo.toml | Compiler switches behavior per-edition; old editions keep working forever |
| **Go** | `go 1.21` in go.mod | `go build` rejects modules requiring newer Go; `go get` refuses incompatible deps |
| **Python** | `python_requires=">=3.8"` in setup.py | pip checks before install; but runtime has no enforcement → dependency hell |
| **Node** | `engines.node: ">=18"` in package.json | npm warns but doesn't block by default → also dependency hell |

AILANG should follow Go's model: **hard enforcement at install and compile time**. No silent failures, no "warning only" mode.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Version constraint is exact, not range-based; same manifest → same compatibility check |
| A2: Replayability | +1 | Lockfile records `ailang_version` used during resolution; reproducible across machines |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | +1 | Package explicitly declares what compiler it needs; no implicit assumptions |
| A5: Bounded Verification | +1 | Version check is local — only compares manifest field against binary version |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured field in TOML/JSON; trivially parseable by AI agents |
| A8: Minimal Syntax | 0 | No new language syntax; just a TOML field |
| A9: Cost Visibility | +1 | Incompatibility is surfaced immediately, not discovered after compilation |
| A10: Composability | +1 | Composes with existing manifest validation and lockfile infrastructure |
| A11: Structured Failure | +1 | Incompatibility errors are typed with clear remediation (upgrade AILANG or pin older version) |
| A12: System Boundary | +1 | Version boundary is explicit at package install boundary |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No ranges, no SAT solving — exact minimum version comparison
- [x] A3 (Effects): Not affected
- [x] A4 (Authority): Version requirement is explicit, not inferred
- [x] A7 (Machines First): Structured TOML field, not prose

---

## Problem Statement

### Current State

1. **`edition = "1"` exists but is meaningless** — required in `ailang.toml`, hashed into interface hash, but the compiler never checks it against anything
2. **Registry records `ailang_version` but nobody reads it** — `ValidationResult.AILANGVersion` is stored in `metadata.json` on publish, but `ailang install` ignores it
3. **No manifest field for minimum AILANG version** — `PackageInfo` has `Name`, `Version`, `Edition`, `Description`, `License` — no `AILANGVersion` or `Requires`
4. **Lockfile doesn't record compiler version** — `LockedPackage` has `ContentHash`, `InterfaceHash`, `Effects`, `Exports` — no record of which AILANG resolved it
5. **No check on `ailang install` or compile** — a package requiring features from v0.10 will happily install on v0.9 and fail at compile time with confusing errors

### What Breaks Without This

- Package uses `httpRequest` (added v0.9.3) → installed on v0.9.0 → compile error: `unknown builtin "httpRequest"`
- Package uses `import pkg/` syntax (added v0.9.5) → installed on v0.9.4 → parse error
- Package uses effect ceiling enforcement (added v0.9.5) → runs on v0.9.0 → effects silently unchecked
- Package uses `std/crypto` (added v0.9.4) → installed on v0.9.3 → module not found

Each of these produces a confusing error that doesn't mention the real cause: version incompatibility.

---

## Design Goals

| Goal | Axiom | Description |
|------|-------|-------------|
| G1 — Explicit Version Requirements | A4 | Packages declare minimum AILANG version |
| G2 — Hard Enforcement on Install | A11 | `ailang install` refuses incompatible packages |
| G3 — Hard Enforcement on Compile | A11 | `ailang check`/`run` refuses incompatible dependencies |
| G4 — Lockfile Reproducibility | A2 | Lockfile records which AILANG version resolved it |
| G5 — Clear Error Messages | A7 | Errors tell you exactly what version is needed and what you have |
| G6 — Registry Integration | A12 | Publish records version; install checks it |

---

## Non-Goals (v1)

- **No version range syntax** (`>=0.9, <1.0`) — AILANG uses exact minimum only, like Go
- **No edition-based behavior switching** — `edition` stays inert for now; future work
- **No automatic AILANG upgrade** — just error with clear message
- **No backward-compatibility shims** — if your AILANG is too old, upgrade
- **No per-dependency version overrides** — all deps must be compatible with one AILANG version

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Field name: `ailang` vs `requires_ailang` vs `min_ailang` | Baked into every manifest; changing later requires migration | human | design | high |
| Comparison semantics: `>=` only vs semver ranges | Determines complexity of version resolver | agent | design | med |
| Enforcement mode: hard error vs warning | Determines whether this actually prevents drift | agent | design | low |

### Design Freeze

- [x] Field name: `ailang = ">=0.9.5"` in `[package]` section (matches Go's `go 1.21` pattern)
- [x] Comparison: `>=` minimum only — no ranges, no upper bounds, no `^`/`~`
- [x] Enforcement: **hard error** on install and compile — no warning-only mode
- [x] Lockfile: add `ailang_version` to `LockFile` root (version that generated it)
- [x] Registry: validator already records this; install should check `metadata.ailang_version`
- [x] Auto-populate: `ailang init package` sets `ailang` to current version

---

## Solution Design

### 1. Manifest Field

Add `ailang` to `[package]` in `ailang.toml`:

```toml
[package]
name = "sunholo/auth"
version = "0.1.0"
edition = "1"
ailang = ">=0.9.5"      # NEW: minimum AILANG version required
```

**Semantics:**
- `>=X.Y.Z` — requires AILANG version X.Y.Z or newer
- Field is **required** for new packages (enforced by `Validate()`)
- Field is **optional** for existing packages (graceful migration: missing = no check)
- Comparison is semantic versioning: major.minor.patch
- `dev` version (local builds) satisfies any constraint (developers shouldn't be blocked)

### 2. Version Comparison

Simple semver `>=` comparison — no range parsing, no SAT solver:

```go
// SatisfiesAILANGVersion checks if currentVersion meets the package requirement.
// Returns true if no requirement is set (backward compat).
// "dev" always satisfies (local builds).
func SatisfiesAILANGVersion(requirement, currentVersion string) bool
```

Parse `>=X.Y.Z` → extract `X.Y.Z` → compare `(major, minor, patch)` tuples.

### 3. Enforcement Points

**On `ailang install <pkg>@<version>`:**
1. Fetch registry metadata (already done)
2. Check `metadata.manifest.ailang` against current AILANG version
3. If incompatible: error with clear message, don't install

```
Error: sunholo/auth@0.2.0 requires AILANG >=0.10.0 but you have v0.9.5
  Upgrade AILANG: go install github.com/sunholo/ailang@latest
  Or install an older version: ailang install sunholo/auth@0.1.0
```

**On `ailang lock` (dependency resolution):**
1. Load each dependency's manifest
2. Check `ailang` field against current version
3. If incompatible: error listing which deps need what

```
Error: dependency resolution failed
  sunholo/http-helpers requires AILANG >=0.9.7 (you have v0.9.5)
  sunholo/auth requires AILANG >=0.9.5 (OK)
```

**On `ailang check`/`ailang run` (compile):**
1. Load lockfile
2. For each locked package with an `ailang` constraint, verify
3. If incompatible: error before compilation starts

### 4. Lockfile Enhancement

Add `ailang_version` to `LockFile` root:

```json
{
  "schema": "ailang.lock/v1",
  "schema_version": "1.0.0",
  "ailang_version": "0.9.5",     // NEW: version that resolved deps
  "generated_at": "2026-03-23T...",
  "generator": "ailang lock 0.9.5",
  "packages": [...]
}
```

And add `ailang` to each `LockedPackage`:

```json
{
  "name": "sunholo/auth",
  "version": "0.1.0",
  "ailang": ">=0.9.5",           // NEW: from package manifest
  "content_hash": "sha256:...",
  "interface_hash": "sha256:...",
  ...
}
```

### 5. Registry Integration

The registry validator already records `AILANGVersion` in `ValidationResult`. Extend:

- **On publish**: also write `ailang` field from manifest into `metadata.json`
- **On install**: check `metadata.manifest.ailang` before downloading tarball
- **On `ailang search`**: show minimum AILANG version in results (so agents can see compatibility)

### 6. Auto-Population

`ailang init package` should auto-set `ailang` to current version:

```toml
[package]
name = "sunholo/mylib"
version = "0.1.0"
edition = "1"
ailang = ">=0.9.9"   # Auto-set from current ailang version
```

---

## Implementation Plan

### Files to Create

| File | Purpose | Est LOC |
|------|---------|---------|
| `internal/pkg/version_compat.go` | `SatisfiesAILANGVersion()`, `ParseVersionConstraint()` | ~60 |
| `internal/pkg/version_compat_test.go` | Comparison tests, edge cases | ~80 |

### Files to Modify

| File | Change | Est LOC |
|------|--------|---------|
| `internal/pkg/manifest.go` | Add `AILANG string` to `PackageInfo`, validate format | +15 |
| `internal/pkg/lockfile.go` | Add `AILANGVersion string` to `LockFile` and `AILANG string` to `LockedPackage` | +10 |
| `internal/pkg/resolver.go` | Check version compat during resolution | +20 |
| `internal/pkg/hasher.go` | Include `ailang` field in interface hash | +2 |
| `cmd/ailang/pkg_install.go` | Check compat before downloading | +15 |
| `cmd/ailang/pkg_init.go` | Auto-set `ailang` to current version | +5 |
| `cmd/ailang/pkg_publish.go` | Warn if `ailang` field missing | +5 |
| `internal/pkg/registry_types.go` | Add `AILANG string` to `MetadataManifest` | +1 |
| `prompts/devtools/v0.8.0.md` | Document `ailang` field in manifest section | +5 |
| `docs/docs/guides/packages.md` | Document version compatibility | +15 |

**Total: ~230 LOC**

### Deferred Decisions

- Whether `edition` should map to specific AILANG version ranges (Rust-style)
- Whether to add `ailang` to `IndexEntry` for search result display
- Whether to add a `--ignore-version` escape hatch for advanced users
- Whether missing `ailang` field in existing packages should trigger a warning on install

---

## Change Propagation

### For Existing Packages (Migration)

1. Existing `ailang.toml` files without `ailang` field remain valid (no breaking change)
2. Missing `ailang` = no version check (backward compatible)
3. `ailang init package` for NEW packages auto-sets `ailang` to current version
4. `ailang publish` warns if `ailang` is missing: "Consider adding `ailang = >=X.Y.Z` for version safety"
5. Next version of published packages in `ailang-packages` repo should add `ailang` field

### For the Registry

The validator already compiles packages and records `ailang_version`. It should:
1. Read `ailang` from the uploaded manifest
2. Store it in `metadata.json` under `manifest.ailang`
3. If `ailang` is missing, auto-populate from the validator's own AILANG version (the version that successfully compiled it)

---

## Example Flows

### Happy Path: Compatible Install

```bash
$ ailang --version
AILANG v0.9.9

$ ailang install sunholo/auth@0.2.0
Fetching sunholo/auth@0.2.0...
  AILANG compatibility: >=0.9.5 (you have v0.9.9) ✓
✓ Downloaded sunholo/auth@0.2.0 (1234 bytes)
✓ Added sunholo/auth (0.2.0)
```

### Blocked: Incompatible Install

```bash
$ ailang --version
AILANG v0.9.5

$ ailang install sunholo/http-helpers@0.3.0
Fetching sunholo/http-helpers@0.3.0...
✗ Error: sunholo/http-helpers@0.3.0 requires AILANG >=0.10.0 (you have v0.9.5)

  Options:
    1. Upgrade AILANG:  go install github.com/sunholo/ailang@latest
    2. Use older version: ailang install sunholo/http-helpers@0.2.0
```

### Blocked: Incompatible Dep at Lock Time

```bash
$ ailang lock
Resolving dependencies...
✗ Error: dependency version mismatch
  sunholo/http-helpers@0.3.0 requires AILANG >=0.10.0
  You have AILANG v0.9.5

  Pin to a compatible version in ailang.toml:
    "sunholo/http-helpers" = "0.2.0"
```

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing packages don't have `ailang` field | Low | Missing field = no check (backward compat); `publish` warns |
| Version comparison bugs (e.g., `0.10.0 < 0.9.5` string compare) | High | Use proper semver tuple comparison, test extensively |
| `dev` builds blocked by version checks | Med | `dev` always satisfies any constraint |
| Package authors set `ailang` too high | Low | Auto-populate from current version; authors can lower manually |
| Lockfile migration for existing projects | Low | New `ailang_version` field is optional; old lockfiles still valid |

---

## Success Criteria

- [ ] `ailang init package` auto-sets `ailang = ">=X.Y.Z"` from current version
- [ ] `ailang install` checks version compatibility before downloading
- [ ] `ailang lock` checks all dependency manifests for version compatibility
- [ ] `ailang check`/`run` verifies lockfile packages against current version
- [ ] `ailang publish` warns if `ailang` field is missing
- [ ] Lockfile records `ailang_version` at root level
- [ ] `dev` version bypasses all checks
- [ ] Clear error messages with remediation steps
- [ ] All existing packages (without `ailang` field) continue to work unchanged
- [ ] Documentation updated (devtools prompt, packages guide)
- [ ] 10+ test cases covering comparison edge cases

---

## Related Documents

- [M-PKG: AILANG Package System & Multi-Agent Coordination](m-pkg-package-system.md) — parent design doc
- [M-PKG-REGISTRY: Package Registry](m-pkg-registry.md) — registry validator records `ailang_version`
- [M-PKG-MSG: Package Messaging Graph](m-pkg-msg-package-messaging-graph.md) — version deltas in coordination messages
- [Go modules: go directive](https://go.dev/ref/mod#go-mod-file-go) — prior art for this pattern

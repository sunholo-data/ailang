# M-PKG-INSTALL-LATEST: Support @latest in ailang install

**Status**: Planned
**Target**: v0.10.0
**Priority**: P2 (Enhancement — feature request, not blocking)
**Estimated**: 0.5 days
**Dependencies**: Registry API must support version listing

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | -1 | `@latest` resolves to different versions at different times. Mitigated: install is a dev-time side effect, not a build step. The resolved exact version is written to `ailang.toml` — build inputs remain deterministic. |
| A2: Replayability | -1 | Without lock file, `@latest` in CI = non-reproducible. Mitigated: lock file is mandatory for builds; `ailang run` / `ailang build` never resolve `@latest` — they read the lock file only. |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `@latest` eliminates need to look up current version manually |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | +1 | Clear errors when version not found |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: 0** → **Decision: Move forward with constraints**

### Hard Violation Check

- [x] A1 (Determinism): `ailang install` is a dev-time side effect, not a build step. The resolved exact version is written to `ailang.toml` — the toml file and lock file remain deterministic inputs to compilation.
- [x] A2 (Replayability): Lock file must be committed. `ailang build` / `ailang run` never resolve `latest` — they read the lock file only.

### Key Constraint: Exact Versions Only in ailang.toml

**`ailang install @latest` resolves to an exact version and writes that exact version to `ailang.toml`.** The keyword `latest` is a CLI convenience for the `install` command only. It must NEVER appear in `ailang.toml` or the lock file.

This is stricter than Cargo (which allows ranges in Cargo.toml) and stricter than npm. AILANG prioritises determinism over convenience — if you want to upgrade, re-run `ailang install pkg@latest` explicitly.

## Problem Statement

Currently `ailang install` requires an exact version:

```bash
ailang install sunholo/auth@0.1.0   # ✓ works
ailang install sunholo/auth@latest  # ✗ fails: "failed to fetch package metadata"
ailang install sunholo/auth         # ✗ fails: "must be vendor/name@version"
```

Users must know the exact latest version to install a package. This is friction for:
- First-time users (`ailang install sunholo/auth` should just work)
- Upgrading (`ailang install sunholo/auth@latest` to get newest)

### Current Code

In `cmd/ailang/pkg_install.go:34-37`, the version is extracted by splitting on `@` and both parts must be non-empty. The version is passed directly to `client.FetchMetadata(name, version)` which constructs a registry URL like `{base}/{vendor}/{name}/{version}/metadata.json`.

## Solution Design

### Scope: @latest Only

This proposal supports exactly two new forms:

| Specifier | Meaning | Example |
|-----------|---------|---------|
| `@0.1.0` | Exact version (current behavior) | `ailang install sunholo/auth@0.1.0` |
| `@latest` | Highest published version | `ailang install sunholo/auth@latest` |
| (no @) | Same as `@latest` | `ailang install sunholo/auth` |

**Semver ranges (`^`, `~`, `>=`) are explicitly excluded** — see "Deferred: Semver Ranges" below.

### Registry API Addition: versions.json

Static, CDN-friendly version listing per package:

```
GET {registry}/{vendor}/{name}/versions.json
→ {"versions": ["0.1.0", "0.2.0", "0.3.0"], "latest": "0.3.0"}
```

Updated by `ailang publish` alongside the tarball upload.

### Implementation

#### 1. Registry Client: Add `FetchVersions`

```go
// In internal/pkg/registry.go
type VersionsResponse struct {
    Versions []string `json:"versions"`
    Latest   string   `json:"latest"`
}

func (c *RegistryClient) FetchVersions(name string) (*VersionsResponse, error) {
    url := fmt.Sprintf("%s/%s/versions.json", c.BaseURL, name)
    // ... fetch and parse ...
}
```

#### 2. Update pkg_install.go

```go
// Allow missing @version (default to latest)
parts := strings.SplitN(spec, "@", 2)
name := parts[0]
versionSpec := "latest"
if len(parts) == 2 && parts[1] != "" {
    versionSpec = parts[1]
}

if versionSpec == "latest" {
    // Resolve latest from registry
    versions, err := client.FetchVersions(name)
    if err != nil {
        return fmt.Errorf("failed to list versions for %s: %w", name, err)
    }
    version = versions.Latest
    fmt.Printf("Resolved %s@latest → %s@%s\n", name, name, version)
} else {
    version = versionSpec
}
```

#### 3. Publish: Generate versions.json

In the publish workflow, after uploading a tarball, update `{vendor}/{name}/versions.json` with the new version appended and `latest` field updated.

### Files to Create/Modify

**Modified files:**
- `internal/pkg/registry.go` — Add `FetchVersions` method, `VersionsResponse` type (~25 LOC)
- `cmd/ailang/pkg_install.go` — Allow missing `@`, resolve `latest` before fetch (~15 LOC)
- `cmd/ailang/pkg_publish.go` — Generate/update `versions.json` on publish (~20 LOC)

**New files:**
- `internal/pkg/registry_test.go` — Test `FetchVersions` (~30 LOC)

### Implementation Plan

**Phase 1: @latest support** (~2 hours)
- [ ] Add `FetchVersions` to registry client
- [ ] Update `pkg_install.go` to allow `@latest` and bare package name
- [ ] Always print resolved version: `Resolved sunholo/auth@latest → sunholo/auth@0.3.2`
- [ ] Tests for latest resolution

**Phase 2: Publish integration + guardrails** (~2 hours)
- [ ] Generate `versions.json` in publish flow
- [ ] Reject non-exact specifiers (`latest`, `^`, `~`, `>=`) if found in `ailang.toml` — fail with clear error
- [ ] `ailang lock` validates all deps in `ailang.toml` are exact versions

## Success Criteria

- [ ] `ailang install sunholo/auth` installs latest version
- [ ] `ailang install sunholo/auth@latest` installs latest version
- [ ] `ailang install sunholo/auth@0.1.0` continues to work (exact version)
- [ ] Exact resolved version always written to `ailang.toml` (never `latest`)
- [ ] `ailang publish` updates `versions.json` in registry
- [ ] `ailang lock` rejects non-exact versions in `ailang.toml` with actionable error
- [ ] Clear error when `versions.json` not available (fallback to exact-only with warning)

## Deferred: Semver Ranges

Semver range operators (`^`, `~`, `>=`) are **explicitly deferred**. Reasons:

1. **Ranges are a different feature with different risk.** `@latest` is simple shorthand — query registry, pick highest, write exact. Ranges introduce a version selection policy language with edge cases (0.x behaviour, pre-release semantics, stable vs experimental).

2. **`>=` is especially suspect.** No natural upper bound, encourages "just give me something newer" rather than deliberate upgrade intent, most likely to produce surprising large-version jumps.

3. **The ecosystem is not yet mature enough.** Ranges only work well when packages reliably treat additive API changes, contract changes, effect widening, and breaking export changes as semantically meaningful version bumps. AILANG is heading there but isn't there yet.

4. **AILANG can do better than semver.** The language already has content hashes, interface hashes, effect ceilings, and change classes. Long-term, upgrade safety should lean on this richer metadata rather than raw semver arithmetic.

### Future Direction: ailang upgrade

Instead of generic semver ranges in `install`, a dedicated upgrade command with AILANG-native semantics:

```bash
ailang upgrade sunholo/auth --latest       # latest stable version
ailang upgrade sunholo/auth --compatible   # same major, no effect widening
ailang upgrade sunholo/auth --patch        # patch-level only
ailang upgrade sunholo/auth --safe         # no breaking interface diff,
                                           # no authority widening,
                                           # no contract strengthening
```

This is clearer in intent, obviously an upgrade workflow (not initial install), and gives room to incorporate package-specific metadata — `--safe` could eventually mean not just semver-compatible but also no effect widening, no breaking export removals, contract-compatible only.

This would be much more AILANG-native than raw `^0.1.0`.

## Non-Goals

- Semver range operators (`^`, `~`, `>=`) — see "Deferred" section above
- Pre-release versions (`0.1.0-beta.1`) — defer to later
- Private registries — separate feature
- Dependency resolution with conflicts (npm-style) — AILANG uses flat deps
- `ailang upgrade` command — separate design doc when needed

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `versions.json` not generated for existing packages | Med | Fallback: if 404, require exact version; warn that `@latest` needs publish to regenerate |
| Registry CDN caching delays `versions.json` update | Low | Cache-Control: max-age=60 on versions.json |
| Users expect range support (npm/cargo familiarity) | Low | Clear error message: "semver ranges not supported — use exact version or @latest" |

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24

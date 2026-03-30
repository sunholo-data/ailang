# M-PKG-METADATA-URLS: Add Repository, Homepage, and License URL to Package Metadata

**Status**: Planned
**Target**: v0.10.0
**Priority**: P2 (Enhancement — improves discoverability, not blocking)
**Estimated**: 1 day
**Dependencies**: M-PKG-EXPLORER (implemented — website needs links to display)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Metadata-only change — no runtime behavior affected |
| A2: Replayability | 0 | No trace or replay impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured URLs in manifest enable AI agents to discover source code, docs, and licenses programmatically |
| A8: Minimal Syntax | 0 | No language syntax changes — TOML metadata only |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Website, CLI, and AI agents all consume the same structured URL fields |
| A11: Structured Failure | 0 | No failure handling changes |
| A12: System Boundary | +1 | URLs make external system boundaries (GitHub, docs sites) explicit in package metadata |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Pure metadata — no runtime nondeterminism
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Structured fields, not freeform prose

---

## Problem Statement

Package pages on the AILANG docs site (`ailang.sunholo.com/docs/packages/sunholo/auth`) and `ailang pkg info` CLI output have no way to link to the package's source code, documentation homepage, or license text. Users must guess the GitHub URL or manually search.

**Current State:**
- `ailang.toml` has `license = "Apache-2.0"` (SPDX identifier) but no URL to the license text
- `ailang.toml` has `description` and `ai_summary` but no URL to a homepage or documentation site
- No `repository` field exists — the source code location is not recorded anywhere in the manifest or registry metadata
- The freeform `[metadata]` section could hold arbitrary keys, but there's no convention for URLs and they don't propagate to `index.json` or the website

**Impact:**
- **Package explorer website**: Cannot link to source code from package detail pages
- **AI agents**: Cannot discover where to find source code for a dependency
- **Humans**: Must guess `github.com/sunholo-data/ailang-packages/tree/main/packages/auth`
- **License compliance**: SPDX identifier exists but no link to full license text

---

## Goals

**Primary Goal:** Add structured URL fields to `ailang.toml` and propagate them through the registry to the website and CLI.

**Success Metrics:**
- Package detail pages on website link to GitHub source
- `ailang pkg info` shows repository and homepage URLs
- `ailang search` results include repository URL for AI agent consumption
- All 15 existing packages updated with repository URLs

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fields in `[package]` vs `[metadata]` | Determines whether URLs are first-class or freeform | human | design | med |
| Backward compatibility for existing packages | Existing published packages lack the fields | agent | compile | low |

### Design Freeze

- [x] **Fields go in `[metadata]` section** — not `[package]`. Rationale: `[package]` is for identity (name, version, edition) and build config (ailang, module_prefix). URLs are discovery metadata, same category as `tags` and `ai_summary`. This also means no change to `PackageInfo` struct — just propagation of existing `[metadata]` keys.
- [x] **All fields optional** — existing packages without URLs continue to work. Website and CLI show "no source URL" gracefully.

---

## Solution Design

### Overview

Add three conventional keys to the `[metadata]` section of `ailang.toml`, propagate them through `MetadataManifest` and `IndexEntry` to the registry API, and display them on the website and CLI.

### New `[metadata]` Keys

```toml
[metadata]
tags = ["auth", "security"]
ai_summary = "Validate API keys..."
repository = "https://github.com/sunholo-data/ailang-packages/tree/main/packages/auth"
homepage = "https://ailang.sunholo.com/docs/packages/sunholo/auth"
license_url = "https://github.com/sunholo-data/ailang-packages/blob/main/LICENSE"
```

| Field | Purpose | Example |
|-------|---------|---------|
| `repository` | Source code URL (GitHub, GitLab, etc.) | `https://github.com/sunholo-data/ailang-packages/tree/main/packages/auth` |
| `homepage` | Documentation or project homepage | `https://ailang.sunholo.com/docs/packages/sunholo/auth` |
| `license_url` | Full license text URL | `https://github.com/sunholo-data/ailang-packages/blob/main/LICENSE` |

All three are optional strings in the freeform `[metadata]` map. No schema change to `PackageManifest` — they're already accessible via `manifest.Metadata["repository"]`.

### Propagation Chain

```
ailang.toml [metadata]
  → ailang publish → validator extracts → metadata.json (MetadataManifest)
  → updateIndex() → index.json (IndexEntry)
  → /api/packages → website PackageDetail component
  → /api/packages → ailang pkg info CLI output
```

### Changes Required

**1. MetadataManifest struct** (`internal/pkg/registry_types.go`):

Add URL fields:
```go
type MetadataManifest struct {
    // ... existing fields ...
    Repository  string   `json:"repository,omitempty"`
    Homepage    string   `json:"homepage,omitempty"`
    LicenseURL  string   `json:"license_url,omitempty"`
}
```

**2. IndexEntry struct** (`internal/pkg/registry_types.go`):

Add URL fields:
```go
type IndexEntry struct {
    // ... existing fields ...
    Repository  string `json:"repository,omitempty"`
    Homepage    string `json:"homepage,omitempty"`
    LicenseURL  string `json:"license_url,omitempty"`
}
```

**3. Registry validator** (`cmd/registry-validator/main.go`):

In `handlePublish`, extract URL fields from manifest metadata. In `updateIndex` / `tryUpdateIndex`, copy URLs to index entry. In `handleRebuildIndex`, copy from metadata manifest.

**4. Website PackageDetail** (`docs/src/components/PackageExplorer/PackageDetail.jsx`):

Display links in the detail section — "Source Code" link, "Documentation" link, "License" link.

**5. CLI** (`cmd/ailang/pkg_info.go`):

Show URLs in `ailang pkg info` output.

**6. Update existing packages** (`ailang-packages/`):

Add `repository`, `homepage`, and `license_url` to all 15 `ailang.toml` files, then republish.

### Implementation Plan

**Phase 1: Struct + Validator** (~2 hours)

- [ ] Add `Repository`, `Homepage`, `LicenseURL` fields to `MetadataManifest` struct
- [ ] Add same fields to `IndexEntry` struct
- [ ] Extract URL fields in `handlePublish` from manifest metadata
- [ ] Copy URL fields in `updateIndex` / `tryUpdateIndex` to index entry
- [ ] Copy URL fields in `handleRebuildIndex` when building from metadata
- [ ] Tests: verify URL fields propagate through publish flow

**Phase 2: CLI + Website** (~2 hours)

- [ ] Display URLs in `ailang pkg info` output
- [ ] Display URLs in PackageDetail component (source code link, docs link, license link)
- [ ] Display URLs in VendorIndex cards (small source code icon/link)
- [ ] Display URLs in PackageExplorer list cards

**Phase 3: Update Existing Packages** (~1 hour)

- [ ] Add `repository`, `homepage`, `license_url` to all 15 `ailang.toml` files in ailang-packages
- [ ] Republish all packages (via the autonomous publish pipeline or manual `ailang publish`)
- [ ] Verify URLs appear on website and in CLI after republish

### Files to Modify/Create

**Modified files:**
- `internal/pkg/registry_types.go` — Add URL fields to `MetadataManifest` and `IndexEntry` (~6 LOC)
- `cmd/registry-validator/main.go` — Extract URLs in `handlePublish`, copy in `updateIndex` and `handleRebuildIndex` (~15 LOC)
- `cmd/ailang/pkg_info.go` — Display URLs in `pkgInfoCommand` output (~10 LOC)
- `docs/src/components/PackageExplorer/PackageDetail.jsx` — Source/docs/license links (~15 LOC)
- `docs/src/components/PackageExplorer/VendorIndex.jsx` — Optional source link icon (~5 LOC)

**Modified files (ailang-packages repo):**
- 15x `ailang.toml` — Add `repository`, `homepage`, `license_url` to `[metadata]`

---

## Examples

### Before

```toml
# ailang.toml
[metadata]
tags = ["auth", "security"]
ai_summary = "Validate API keys..."
```

```
$ ailang pkg info sunholo/auth
sunholo/auth v0.1.1
Validate API keys via hash comparison...

  Stability:  experimental
  Effects:    Pure
  # No source code link, no homepage, no license URL
```

### After

```toml
# ailang.toml
[metadata]
tags = ["auth", "security"]
ai_summary = "Validate API keys..."
repository = "https://github.com/sunholo-data/ailang-packages/tree/main/packages/auth"
homepage = "https://ailang.sunholo.com/docs/packages/sunholo/auth"
license_url = "https://github.com/sunholo-data/ailang-packages/blob/main/LICENSE"
```

```
$ ailang pkg info sunholo/auth
sunholo/auth v0.1.1
Validate API keys via hash comparison...

  Stability:  experimental
  Effects:    Pure
  License:    Apache-2.0
  Repository: https://github.com/sunholo-data/ailang-packages/tree/main/packages/auth
  Homepage:   https://ailang.sunholo.com/docs/packages/sunholo/auth
```

---

## Success Criteria

- [ ] `MetadataManifest` and `IndexEntry` include repository, homepage, license_url fields
- [ ] Validator extracts and propagates URL fields on publish
- [ ] `ailang pkg info` displays URLs when present, omits when absent
- [ ] Website package detail page shows clickable source/docs/license links
- [ ] All 15 existing packages updated with URLs and republished
- [ ] Backward compatible: packages without URL fields still work
- [ ] All tests passing

---

## Testing Strategy

**Unit tests:**
- Verify URL fields round-trip through JSON marshal/unmarshal
- Verify `getMetaString` extracts URL fields from manifest metadata
- Verify index update copies URL fields

**Integration tests:**
- Publish a package with URL fields, verify they appear in metadata.json and index.json
- Verify `ailang pkg info` displays URLs from live API

**Manual testing:**
- Check website package detail page shows source link
- Verify packages without URLs display gracefully (no broken links)

---

## Deferred Decisions

- URL validation (check for valid HTTP/HTTPS scheme) — agent may choose whether to validate or accept any string
- Whether to auto-populate `homepage` from package name convention — agent may choose

---

## Non-Goals

- **URL verification**: We don't check if URLs are reachable (no HTTP HEAD at publish time)
- **Source code browsing**: Not embedding a code viewer — just linking to GitHub
- **License analysis**: Not parsing SPDX identifiers or checking compatibility
- **Mandatory fields**: All three URLs remain optional forever

---

## Timeline

**Day 1** (~5 hours):
- Phase 1: Struct changes + validator propagation (2h)
- Phase 2: CLI + website display (2h)
- Phase 3: Update packages + republish (1h)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing index.json doesn't have URL fields | Low | Fields are `omitempty` — backward compatible. URLs appear after republish. |
| URL rot (links become stale over time) | Low | URLs are author-maintained, not auto-verified. Same as npm/cargo. |

---

## Related Documents

**Implemented (informs design):**
- [m-pkg-package-system.md](../../implemented/v0_9_5/m-pkg-package-system.md) — Manifest structure
- [m-pkg-registry.md](../../implemented/v0_9_7/m-pkg-registry.md) — Registry metadata schema

**Planned (this builds on):**
- [m-pkg-explorer-website.md](m-pkg-explorer-website.md) — Website that displays these URLs

---

## References

- [Registry types](../../internal/pkg/registry_types.go) — MetadataManifest, IndexEntry structs
- [Manifest](../../internal/pkg/manifest.go) — PackageManifest, Metadata map
- [Cargo.toml \[package\] section](https://doc.rust-lang.org/cargo/reference/manifest.html#the-package-section) — Inspiration (repository, homepage, documentation fields)

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24

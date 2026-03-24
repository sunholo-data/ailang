# Sprint Plan: M-PKG-REGISTRY — Client-Side Implementation

## Summary

Build the CLI and client-side components for the AILANG package registry: `ailang publish`, `ailang install`, `ailang search`, `ailang docs`, plus the registry HTTP client, tarball creation, and resolver integration. The GCS bucket and Cloud Run validator are deployed by the multivac team separately — we build against the GCS HTTP API and can mock/test with a local bucket.

**Duration:** 5 days (4 milestones)
**Dependencies:** M-PKG Phase 1.5b (complete), multivac Terraform (parallel — we mock until ready)
**Risk Level:** Medium — HTTP client work, but patterns are well-established
**Design Doc:** [m-pkg-registry.md](m-pkg-registry.md)

## Velocity Reference

Recent M-PKG work:
- Phase 1 (manifest + lock + path deps + imports): ~1,650 LOC in 1 session
- Phase 1.5 (interface hash + effect ceiling): ~780 LOC in 1 session
- Phase 1.5b (git deps + AGENT.md): ~640 LOC in 1 session

Average: ~350-400 LOC/day sustained.

## Proposed Milestones

### Milestone 1: Tarball + Registry Client (~1.5 days)

**Goal:** Create/extract package tarballs and fetch from GCS via HTTP.

**Estimated:** ~250 implementation + ~100 tests = ~350 LOC

**New files:**
- `internal/pkg/tarball.go` (~120 LOC):
  - `CreateTarball(packageDir string) ([]byte, error)` — tar.gz of ailang.toml + .ail + AGENT.md
  - `ExtractTarball(data []byte, destDir string) error` — extract to cache dir
  - Deterministic: sorted file order, no timestamps
- `internal/pkg/registry.go` (~130 LOC):
  - `RegistryClient` struct with base URL (default: `https://storage.googleapis.com/ailang-registry`)
  - `FetchIndex() (*RegistryIndex, error)` — download and parse index.json
  - `FetchPackage(name, version string) ([]byte, error)` — download package.tar.gz
  - `FetchMetadata(name, version string) (*PackageMetadata, error)` — download metadata.json
  - `SearchPackages(query string) []IndexEntry` — filter index by keyword/tag
  - HTTP with `If-Modified-Since` caching for index.json
  - Configurable via `AILANG_REGISTRY` env var
- `internal/pkg/registry_types.go` (~50 LOC):
  - `RegistryIndex` struct matching index.json schema
  - `IndexEntry` struct (per-package in index)
  - `PackageMetadata` struct matching metadata.json schema

**Tests:**
- Tarball roundtrip (create → extract → verify files)
- Tarball excludes non-package files (.git, tests/)
- Registry client constructs correct GCS URLs
- Index parsing and search filtering

**Acceptance Criteria:**
- [ ] `CreateTarball` produces valid tar.gz with ailang.toml + .ail + AGENT.md
- [ ] `ExtractTarball` extracts to destination with correct structure
- [ ] `RegistryClient` constructs correct GCS URLs for packages/metadata/index
- [ ] `FetchIndex` parses index.json correctly
- [ ] `SearchPackages` filters by name, ai_summary, and tags
- [ ] All existing tests pass

---

### Milestone 2: `ailang install` + `ailang search` (~1.5 days)

**Goal:** Download packages from registry and search the index.

**Estimated:** ~200 implementation + ~80 tests = ~280 LOC

**New files:**
- `cmd/ailang/pkg_install.go` (~100 LOC):
  - `ailang install sunholo/auth@0.1.0`
  - Download package.tar.gz from registry
  - Verify tarball hash against metadata.json
  - Extract to `~/.ailang/cache/registry/vendor/name/version/`
  - Add version dependency to ailang.toml
  - Run `ailang lock` automatically
- `cmd/ailang/pkg_search.go` (~60 LOC):
  - `ailang search "auth"` — keyword search
  - `ailang search --tag gcp` — tag filter
  - `ailang search --refresh` — force re-download index.json
  - Structured output: name@version — ai_summary [effects]
- `cmd/ailang/pkg_docs.go` (~40 LOC):
  - `ailang docs sunholo/auth` — display AGENT.md from cached or downloaded package

**Modified files:**
- `cmd/ailang/main.go` — register `install`, `search`, `docs` commands

**Tests:**
- Install writes to correct cache path
- Search matches on name, ai_summary, tags
- Docs displays AGENT.md content

**Acceptance Criteria:**
- [ ] `ailang install sunholo/auth@0.1.0` downloads and caches package
- [ ] `ailang search "auth"` returns structured results
- [ ] `ailang search --tag gcp` filters by tag
- [ ] `ailang docs sunholo/auth` displays AGENT.md
- [ ] Cache at `~/.ailang/cache/registry/` is organized by vendor/name/version
- [ ] All existing tests pass

---

### Milestone 3: Resolver + `ailang publish` (~1.5 days)

**Goal:** Registry deps resolve in `ailang lock`, and `ailang publish` uploads to the validator.

**Estimated:** ~200 implementation + ~80 tests = ~280 LOC

**Modified files:**
- `internal/pkg/resolver.go` (~60 LOC change):
  - Replace registry stub (line 148-159) with real resolution:
    - Fetch metadata.json from registry
    - Check local cache, download if needed
    - Compute content hash from extracted tarball
    - Populate ResolvedPackage with registry source
- `internal/pkg/lockfile.go` (~20 LOC):
  - `ValidateContentHashes` handles "registry" source (check cache dir)
- `internal/pkg/loader.go` (~10 LOC):
  - `packageDir()` handles "registry" source (return cache path)

**New files:**
- `cmd/ailang/pkg_publish.go` (~100 LOC):
  - `ailang publish` — create tarball, POST to validator Cloud Run endpoint
  - Read `ailang.toml` for package name + version
  - Authenticate with GCP (ADC or API key)
  - Display validation results from response
  - Error if version already published (409 from validator)

**Tests:**
- Resolver resolves registry dep from cached package
- Lock file roundtrip with registry source
- Publish creates correct tarball

**Acceptance Criteria:**
- [ ] `ailang lock` resolves registry deps (downloads + hashes)
- [ ] `import pkg/sunholo/auth/keys (validateKeyHash)` works with registry dep
- [ ] `ailang publish` creates tarball and POSTs to validator endpoint
- [ ] Lock file records registry source with content + interface hash
- [ ] Path > git > registry resolution priority maintained
- [ ] All existing tests pass

---

### Milestone 4: Integration + Docs (~0.5 day)

**Goal:** End-to-end test, update prompts, CHANGELOG.

**Estimated:** ~100 LOC (docs + polish)

**Tasks:**
- End-to-end test: publish a package, install it in a new project, import and run
  - Can use a test GCS bucket or mock server until multivac infra ready
- Update devtools prompt with `install`, `search`, `publish`, `docs` commands
- Update teaching prompt if needed
- CHANGELOG entry
- Update `docs/docs/guides/packages.md` with registry install section

**Acceptance Criteria:**
- [ ] Full publish → install → import → run cycle works
- [ ] `ailang devtools-prompt` documents all registry commands
- [ ] `docs/docs/guides/packages.md` updated
- [ ] CHANGELOG entry
- [ ] All existing tests pass

---

## Day-by-Day Summary

| Day | Milestone | Deliverable |
|-----|-----------|-------------|
| 1 | M1 | Tarball creation/extraction, registry HTTP client |
| 2 | M1+M2 | Registry types, install command, search command |
| 3 | M2+M3 | Docs command, resolver registry resolution |
| 4 | M3 | Publish command, lock file registry support |
| 5 | M4 | Integration test, docs, CHANGELOG |

## Key Assumption

The multivac team deploys the GCS bucket and Cloud Run validator in parallel. For testing:
- **Index.json + packages**: can upload manually to a test bucket, or mock with a local HTTP server
- **Publish endpoint**: can test tarball creation locally, actual upload tested when Cloud Run is ready
- **Install/search**: work against any GCS bucket with the right structure

## Estimated Total LOC

| Component | LOC |
|-----------|-----|
| `internal/pkg/tarball.go` + tests | ~170 |
| `internal/pkg/registry.go` + types + tests | ~230 |
| `cmd/ailang/pkg_install.go` | ~100 |
| `cmd/ailang/pkg_search.go` | ~60 |
| `cmd/ailang/pkg_docs.go` | ~40 |
| `cmd/ailang/pkg_publish.go` | ~100 |
| Resolver/lockfile/loader updates | ~90 |
| Docs + CHANGELOG | ~100 |
| **Total** | **~890** |

---

**Document created**: 2026-03-20

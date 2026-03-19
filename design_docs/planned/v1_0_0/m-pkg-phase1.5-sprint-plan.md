# Sprint Plan: M-PKG Phase 1.5 — Tightening

## Summary

Tighten the Phase 1 package system with interface hashes, content hash validation at build time, effect ceiling compile-time checks, and documentation. These are the gaps identified in the design review that should ship before v1.0.0.

**Duration:** 3 days (3 milestones)
**Dependencies:** M-PKG Phase 1 (complete — `9966679b`)
**Risk Level:** Low-Medium — additive work on proven foundation
**Design Doc:** [m-pkg-package-system.md](m-pkg-package-system.md)

## Current Status

### Just Shipped (Phase 1)
- `ailang.toml` manifest with TOML parser, validation
- `ailang.lock` deterministic JSON with content hashes
- Path dependency resolver with cycle detection
- `import pkg/vendor/name/module (symbols)` parser + loader
- Export enforcement (non-exported modules rejected)
- CLI: `init package`, `add --path`, `lock`, `tree`
- Pipeline integration (auto-detects `ailang.toml`)
- Backward compat (no `ailang.toml` = legacy behavior)
- 33 tests, 0 regressions

### What's Missing
- **Interface hash** — lock file has `content_hash` but no `interface_hash`
- **Content hash validation** — hashes are computed at `lock` time but never re-checked at build
- **Effect ceiling enforcement** — `[effects].max` is declared but not checked at compile time
- **Docs guide** — no `docs/docs/guides/packages.md`
- **CHANGELOG** — Phase 1 not documented in changelog

## Proposed Milestones

### Milestone 1: Interface Hash + Content Validation (~1.5 days)

**Goal:** Add interface hash to lock file. Validate content hashes at build time. Detect stale lock files.

**Estimated:** ~250 implementation + ~100 tests = ~350 LOC

**Day 1: Interface Hash**
- Add `InterfaceHash(manifest *PackageManifest) string` to `internal/pkg/hasher.go`:
  - SHA256 of canonical JSON: package name, edition, sorted exported module list, sorted effect max list
  - Explicitly excludes: source formatting, comments, internal modules, declaration order
  - For Phase 1.5: hash the manifest-level interface (exports + effects), not per-symbol types (that's Phase 3)
- Add `interface_hash` field to `LockedPackage` struct in `lockfile.go`
- Update `resolver.go` to compute interface hash during resolution
- Update `ailang lock` to include interface hash in output
- Tests: interface hash determinism, interface hash changes on export change, stays same on internal refactor

**Day 2 (morning): Content Hash Validation at Build Time**
- Add `ValidateContentHashes(rootDir string) error` to `lockfile.go`:
  - For each path dep: recompute content hash, compare to lock file
  - Error if mismatch: "Dependency X content changed. Run 'ailang lock' to update."
- Wire into `tryLoadPackageResolver()` in `internal/pipeline/package_resolver.go`:
  - After loading lock file, validate content hashes
  - Fail fast with actionable error message
- Tests: detect changed dependency, pass when unchanged

**Acceptance Criteria:**
- [ ] `ailang lock` generates `interface_hash` for each package
- [ ] Interface hash is deterministic (same exports + effects = same hash)
- [ ] Interface hash stays same on internal refactor (only content hash changes)
- [ ] Build fails if dependency content changed without re-locking
- [ ] Error message tells user to run `ailang lock`
- [ ] All existing tests pass

---

### Milestone 2: Effect Ceiling Enforcement (~1 day)

**Goal:** Compile-time check that `import pkg/...` modules don't exceed their package's declared `[effects].max`.

**Estimated:** ~150 implementation + ~80 tests = ~230 LOC

**Day 2 (afternoon) + Day 3 (morning):**
- Add `CheckEffectCeiling(pkgName string, importedEffects []string) error` to `internal/pkg/loader.go`:
  - Load package manifest for the dependency
  - Compare imported module's effects against package `[effects].max`
  - Error if violation: "Module X uses effect IO but package Y max effects = []"
- The check runs at the *depending* package's compile time, not the dependency's
  - This means: if you depend on `acme/http` which declares `max = ["IO", "Net"]`, that's fine
  - But if `acme/http` internally uses `FS` and only declares `max = ["IO", "Net"]`, that's the *dependency's* bug
  - Phase 1.5 scope: validate that package manifests are honest about their own effects
- Wire into pipeline: after loading a package module, check the package's effect declaration
- Tests: effect within bounds passes, effect violation fails with actionable message

**Acceptance Criteria:**
- [ ] Package with `max = []` that imports an effectful module → compile error
- [ ] Package with `max = ["IO"]` that only uses IO → passes
- [ ] Error message suggests adding the missing effect to `ailang.toml`
- [ ] Effect check only applies to packages with `ailang.toml` (backward compat)
- [ ] All existing tests pass

---

### Milestone 3: Documentation + CHANGELOG (~0.5 day)

**Goal:** Ship the packages guide and document Phase 1 + 1.5 in changelog.

**Estimated:** ~200 LOC (docs)

**Day 3 (afternoon):**
- Create `docs/docs/guides/packages.md`:
  - What packages are (distribution + coordination + verification + authority boundary)
  - Quick start: `ailang init package`, `ailang add`, `ailang lock`, `ailang run`
  - `ailang.toml` format reference
  - `import pkg/...` syntax
  - Export enforcement
  - Effect ceilings
  - Path dependencies and workspaces
  - Backward compatibility
- Add CHANGELOG entry for M-PKG Phase 1 + 1.5
- Update `examples/manifest.json` with package_demo entry

**Acceptance Criteria:**
- [ ] `docs/docs/guides/packages.md` exists and covers all Phase 1 + 1.5 features
- [ ] CHANGELOG has M-PKG entry
- [ ] Package demo in examples manifest

---

## What Remains Deferred

| Item | Deferred To | Notes |
|------|------------|-------|
| Per-symbol type signatures in interface hash | Phase 3 | Needs stable type export format |
| Registry server + client | Phase 2 | Path deps sufficient for v1.0.0 |
| Change classification (A-E) | Phase 3 | Needs per-symbol interface hash |
| AI Task Views | Phase 3 | Needs registry |
| Canonical package/module path enforcement | Phase 2 | Design doc specifies it but low priority for v1.0.0 |

## Success Metrics

- Interface hash in lock file, deterministic
- Content hash validated at build time (stale lock → actionable error)
- Effect ceiling enforced at compile time
- Packages guide published
- All 3,400+ existing tests pass
- `make lint` clean

## Day-by-Day Summary

| Day | What |
|-----|------|
| 1 | Interface hash computation, add to lock file |
| 2 | Content hash validation at build, effect ceiling enforcement |
| 3 | Effect ceiling tests, docs guide, CHANGELOG |

---

**Document created**: 2026-03-19

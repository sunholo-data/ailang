# Sprint Plan: M-PKG Phase 1 — Local Package Management

## Summary

Implement the foundational package system for AILANG v1.0: `ailang.toml` manifest, `ailang.lock` lock file, path dependencies, `import pkg/...` syntax, and export enforcement. This is the minimum viable package system that enables multi-package repos without requiring a registry.

**Duration:** 8 days (split across 4 milestones)
**Dependencies:** None — all prerequisite systems (modules, effects, CLI) are complete
**Risk Level:** Medium — touches parser and loader, but changes are additive (no existing behavior modified)
**Design Doc:** [m-pkg-package-system.md](m-pkg-package-system.md)

## Current Status Analysis

### Completed Recently (velocity reference)
- M-TYPE-V2-MIGRATION: 400 ins / 367 del across 28 files — **1 day** (type system refactor)
- M-CONCURRENCY: ~800 LOC across 5 commits — **2 days** (thread-safe eval, per-request Fork)
- M-CONTRACT-ENSURES-CODEGEN-FIX: ~200 LOC — **0.5 days**
- M-SMT-CROSS-MODULE-TYPES Phase 1: ~180 LOC — **0.5 days**

### Velocity
- Recent average: **~300-400 LOC/day** (implementation + tests)
- 14-day window: 207 commits, 5,280 insertions across 69 files
- Typical milestone: 1-2 days for focused work

### What Exists Already
- Parser handles `import std/io (println)` with path segments, aliases, symbols — **~100 LOC in parser_file.go**
- `internal/loader/loader.go` resolves module paths — **662 LOC**
- `internal/manifest/manifest.go` has SHA256, schema versioning, deterministic JSON — **390 LOC, directly reusable**
- `internal/runtime/` has cycle detection, topo sort, ModuleRuntime — **reusable patterns**
- `.ailang/cache/compile/manifest.json` stub exists
- No TOML dependency in `go.mod` yet
- `internal/pkg/` does not exist yet

## Now vs Later Decision

### NOW — Phase 1 (this sprint, 8 days)

| Milestone | What | LOC Est | Days | Why Now |
|-----------|------|---------|------|---------|
| **1A** | Manifest + Lock File | ~550 | 2 | Foundation everything else depends on |
| **1B** | Path Dependencies | ~400 | 2 | Enables multi-package repos immediately |
| **1C** | Package Imports + Export Enforcement | ~500 | 3 | The actual user-facing feature |
| **1D** | Integration + Proof-of-Concept | ~200 | 1 | Proves it works end-to-end |
| **Total** | | **~1,650** | **8** | |

### LATER — Phase 1.5 (separate sprint, after Phase 1 ships)

| Item | Why Later |
|------|-----------|
| **Effect ceiling enforcement** | Touches type checker pipeline; needs careful integration with existing effect inference. Ship package imports first, add effect constraints after. |
| **Interface hash computation** | Requires stable exported type representations. Implement after TFunc2 migration settles. Ship with content hash only in v1.0.0; add interface hash in v1.0.1. |

### MUCH LATER — Phase 2-3

| Item | Why Much Later |
|------|---------------|
| Registry server + client | Needs community; path deps cover immediate needs |
| `ailang publish/install/search` | Blocked on registry |
| Change classification engine (A-E) | Needs interface hashes working first |
| AI Task Views | Needs registry + community packages |
| Trust scoring, effect policies | v1.x features |

## Proposed Milestones

### Milestone 1A: Manifest & Lock File (~2 days)

**Goal:** Parse `ailang.toml`, generate `ailang.lock` with content hashes. No import resolution yet — just the data model.

**Estimated:** ~350 implementation + ~200 tests = ~550 LOC

**Day 1: TOML Manifest Parser**
- Add `github.com/BurntSushi/toml` to `go.mod`
- Create `internal/pkg/manifest.go`:
  - `PackageManifest` struct (name, version, edition, exports, dependencies, effects, metadata, stability)
  - `LoadManifest(path string) (*PackageManifest, error)` — parse `ailang.toml`
  - `ValidateManifest()` — required fields, module prefix consistency
  - `InitManifest(name, dir string)` — generate default `ailang.toml`
- Create `internal/pkg/manifest_test.go`:
  - Parse valid manifest
  - Reject missing `[package].name`
  - Reject invalid module prefix in `[exports]`
  - Roundtrip test

**Day 2: Lock File + Content Hasher**
- Create `internal/pkg/hasher.go`:
  - `ContentHash(dir string) (string, error)` — SHA256 of sorted `.ail` files
  - Reuse `crypto/sha256` pattern from `internal/manifest/manifest.go:278`
- Create `internal/pkg/lockfile.go`:
  - `LockFile` struct (schema, packages list with name, version, content_hash, source)
  - `GenerateLockFile(manifest, resolvedDeps)` — deterministic JSON
  - `LoadLockFile(path string)` — parse and validate
  - `ValidateAgainstManifest(lock, manifest)` — disagreement = error
  - Reuse `internal/manifest/` for schema versioning, deterministic serialization
- Create `cmd/ailang/pkg_init.go`:
  - `ailang init [--name vendor/name]` — generate `ailang.toml`
- Tests for hash determinism, lock file roundtrip, manifest/lock agreement

**Acceptance Criteria:**
- [ ] `ailang init --name sunholo/mylib` creates valid `ailang.toml`
- [ ] `PackageManifest` parses all fields from TOML
- [ ] Content hash is deterministic (same files → same hash)
- [ ] Lock file JSON is deterministic (sorted keys, sorted packages)
- [ ] Lock file validates against manifest (disagreement = error)
- [ ] All existing tests pass (`make test`)

**Risks:**
- TOML parser dependency adds to binary size — Low risk, `BurntSushi/toml` is small

---

### Milestone 1B: Path Dependencies (~2 days)

**Goal:** Resolve `{ path = "../utils" }` dependencies, build dependency graph, generate lock file with resolved path deps.

**Estimated:** ~250 implementation + ~150 tests = ~400 LOC

**Day 3: Dependency Resolver**
- Create `internal/pkg/resolver.go`:
  - `ResolveDependencies(manifest, rootDir) ([]ResolvedPackage, error)`
  - Walk direct deps, resolve path deps to absolute paths
  - Load each dep's `ailang.toml` recursively
  - Build dependency graph, detect cycles (reuse DFS pattern from `internal/runtime/runtime.go`)
  - Compute content hash for each resolved package
- Create `internal/pkg/resolver_test.go`:
  - Resolve single path dep
  - Resolve transitive path deps (A→B→C)
  - Detect cycle (A→B→A)
  - Error on missing `ailang.toml` in dep
  - Error on path dep pointing to non-existent dir

**Day 4: CLI Commands + Lock Generation**
- Create `cmd/ailang/pkg_add.go`:
  - `ailang add ../shared/json --path` — add path dep to `ailang.toml`
- Create `cmd/ailang/pkg_lock.go`:
  - `ailang lock` — resolve deps, generate `ailang.lock`
- Create `cmd/ailang/pkg_tree.go`:
  - `ailang tree` — print dependency tree (ASCII art)
- Register commands in `cmd/ailang/main.go`
- Test: multi-package workspace with path deps resolves correctly

**Acceptance Criteria:**
- [ ] `ailang add ../utils --path` modifies `ailang.toml` correctly
- [ ] `ailang lock` generates `ailang.lock` with all resolved path deps + content hashes
- [ ] Circular path deps detected with clear error (`circular dependency: A → B → A`)
- [ ] `ailang tree` displays readable dependency graph
- [ ] Missing dep directory → clear error message
- [ ] All existing tests pass

**Risks:**
- Path normalization across OS — Medium; use `filepath.Abs` and `filepath.Rel`

---

### Milestone 1C: Package Imports & Export Enforcement (~3 days)

**Goal:** `import pkg/vendor/name/module (symbols)` resolves against lock file. Non-exported modules are rejected.

**Estimated:** ~350 implementation + ~150 tests = ~500 LOC

**Day 5: Parser Support for `pkg/` Imports**
- Modify `internal/parser/parser_file.go:parseImportDecl()`:
  - Recognize `pkg/` prefix as first path segment
  - Set `ImportDecl.IsPackage = true` (new field on AST node)
  - Existing path parsing handles `pkg/vendor/name/module` naturally (already does `segment/segment/segment`)
- Add `IsPackage bool` field to `internal/ast/ast.go:ImportDecl`
- Add parser test: `import pkg/sunholo/json/parser (parseJson)` → golden file
- Add parser test: `import pkg/sunholo/json/parser` (no symbols) → golden file

**Day 6: Package Loader + Module Resolution**
- Create `internal/pkg/loader.go`:
  - `PackageLoader` struct holding lock file data + resolved package paths
  - `ResolvePackageImport(importPath string) (moduleSourcePath string, error)`
    - Strip `pkg/` prefix
    - Extract package name (first two segments: `vendor/name`)
    - Look up package in lock file
    - Map remaining segments to file path within package `src/` dir
  - `CheckExportVisibility(packageName, moduleName string) error`
    - Load dep's `ailang.toml`, check if module is in `[exports].modules`
    - Error with list of available exports if not
- Modify `internal/loader/loader.go`:
  - When `ImportDecl.IsPackage == true`, delegate to `PackageLoader`
  - Otherwise, existing local/stdlib resolution unchanged

**Day 7: Export Enforcement + Integration Tests**
- `internal/pkg/loader_test.go`:
  - Import exported module → succeeds
  - Import non-exported module → error with available exports list
  - Import from undeclared dependency → error
  - Module prefix mismatch → error
- Create test fixture: minimal multi-package project in `testdata/`
  - `testdata/pkg_test/lib_a/ailang.toml` + `src/core.ail`
  - `testdata/pkg_test/lib_b/ailang.toml` + `src/parser.ail` + `src/internal/helpers.ail`
  - `testdata/pkg_test/app/ailang.toml` (depends on lib_a, lib_b via path)
- End-to-end test: `ailang run` on a project with `import pkg/...` works

**Acceptance Criteria:**
- [ ] `import pkg/sunholo/json/parser (parseJson)` parses correctly
- [ ] Package imports resolve against lock file entries
- [ ] Non-exported module import → compile error with available exports
- [ ] Import from undeclared dependency → compile error
- [ ] Existing `import std/...` and local imports work unchanged (no regression)
- [ ] Golden file tests for pkg/ import parsing
- [ ] All existing tests pass

**Risks:**
- Loader changes could regress existing imports — Medium; existing path unchanged, pkg/ is a new branch
- AST change (IsPackage field) touches many files — Low; it's an additive optional field

---

### Milestone 1D: Integration & Proof-of-Concept (~1 day)

**Goal:** Prove the system works end-to-end. Create an example. Update docs.

**Estimated:** ~100 implementation + ~100 docs/examples = ~200 LOC

**Day 8: Polish + Docparse Proof-of-Concept**
- Create `examples/runnable/package_demo/`:
  - `lib/ailang.toml` + `lib/src/math.ail` (exports pure math functions)
  - `app/ailang.toml` + `app/src/main.ail` (imports from lib via path dep)
  - Verify: `cd app && ailang lock && ailang run src/main.ail`
- Integrate package resolution into `ailang run`, `ailang compile`, `ailang verify`:
  - If `ailang.toml` exists in cwd, load lock file and set up PackageLoader
  - If no `ailang.toml`, existing behavior unchanged
- Error messages polish:
  - Missing `ailang.lock` → "Run `ailang lock` to resolve dependencies"
  - Hash mismatch → "Dependency content changed. Run `ailang lock` to update"
- Update `docs/docs/guides/packages.md` (new guide)
- Add to CHANGELOG.md

**Acceptance Criteria:**
- [ ] Package demo example works end-to-end
- [ ] `ailang run` with `ailang.toml` present resolves packages automatically
- [ ] `ailang run` without `ailang.toml` works exactly as before (backward compat)
- [ ] Error messages are helpful and actionable
- [ ] Documentation guide exists
- [ ] All existing tests pass
- [ ] `make lint` clean

---

## What's Explicitly Deferred

| Item | Deferred To | Reason |
|------|------------|--------|
| Interface hash | Phase 1.5 | Needs stable type representations; content hash sufficient for v1.0.0 |
| Effect ceiling enforcement | Phase 1.5 | Touches type checker; ship imports first |
| `ailang publish/install/search` | Phase 2 | Needs registry |
| Registry server | Phase 2 | Path deps cover immediate needs |
| Change classification (A-E) | Phase 3 | Needs interface hashes |
| AI Task Views | Phase 3 | Needs registry + community |
| Semantic freezing enforcement | Phase 3 | Not blocking for v1.0 |

## Success Metrics

- All 3,400+ existing tests pass (zero regressions)
- Package demo example works end-to-end
- `ailang init`, `ailang add --path`, `ailang lock`, `ailang tree` all functional
- `import pkg/...` resolves correctly
- Export enforcement blocks non-exported module access
- Backward compatibility: projects without `ailang.toml` work unchanged
- `make lint` clean
- Documentation: packages guide published

## Day-by-Day Summary

| Day | Milestone | Deliverable |
|-----|-----------|-------------|
| 1 | 1A | TOML manifest parser, `ailang init` |
| 2 | 1A | Content hasher, lock file generator |
| 3 | 1B | Dependency resolver with cycle detection |
| 4 | 1B | `ailang add`, `ailang lock`, `ailang tree` CLI |
| 5 | 1C | Parser: `import pkg/...` syntax |
| 6 | 1C | Package loader + module resolution |
| 7 | 1C | Export enforcement + integration tests |
| 8 | 1D | End-to-end demo, docs, polish |

## Dependencies & Open Items

**Resolved (from design doc review):**
- Import syntax: `import pkg/vendor/name/module (symbols)` ✓
- Lock file format: JSON (reuse `internal/manifest/`) ✓
- Workspace model: emergent from path-linked packages ✓
- Export enforcement: `[exports].modules` list is sole authority ✓
- Stdlib: not a package, remains compiler-resolved ✓

**Not needed for this sprint (confirmed deferred):**
- Interface hash canonical spec — deferred to Phase 1.5
- Effect ceiling enforcement — deferred to Phase 1.5
- Registry — deferred to Phase 2

---

**Document created**: 2026-03-19

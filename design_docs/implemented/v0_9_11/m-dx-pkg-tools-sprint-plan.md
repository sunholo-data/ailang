# Sprint Plan: M-DX-PKG-CHECK + M-DX-PKG-TEST

## Summary

Add `--package` flag to both `ailang check` and `ailang test` commands, enabling cross-module type resolution and package-aware test execution. Both features share the same foundation: reading `ailang.toml`, discovering source files, and routing through the existing module compilation pipeline instead of independent single-file compilation.

**Duration:** 2 days (estimated 12-14 hours total)
**Dependencies:** None (all infrastructure exists)
**Risk Level:** Low -- existing module pipeline handles cross-module resolution; we're wiring it up to new entry points

## Current Status Analysis

### Existing Infrastructure (already built)
- `ailang check` command with directory support, JSON output, telemetry, timeout handling
- `ailang test` command with file discovery, aggregation, human/JSON reporters, property tests
- `pipeline.runModuleWithContext()` -- full cross-module compilation with import resolution
- `loader.ModuleLoader.LoadAll()` -- DFS module loading with cycle detection and caching
- `pkg.PackageLoader` -- package import resolution from ailang.toml + lockfile
- `internal/testing` -- complete test framework with `RunTestsFromFile()`

### Actual Gap
Both `check` and `test` currently process files **independently** (single-file pipeline). The `--package` flag needs to route through the **module pipeline** which already resolves cross-module types. The work is integration, not new compilation logic.

### Velocity
- Recent: ~700 LOC/day (M-PKG-COMPAT: 120 LOC + 38 tests in ~1 day, M-PKG-MSG: 1200 LOC + 44 tests in ~2 days)
- This sprint: ~400-500 LOC implementation + ~300 LOC tests = ~750 LOC total

## Proposed Milestones

### M1: Package Discovery and Source Mapping (~150 LOC, 3 hours)

**Goal:** Create shared package discovery that reads `ailang.toml` and maps module paths to source files.

**Tasks:**
- Create `internal/pkg/discover.go` with `DiscoverPackageSources(dir string) -> PackageInfo`
  - Read `ailang.toml` for module list from `[exports]`
  - Find all `.ail` files in package directory (src/ and root)
  - Map module paths to file paths (reuse `PackageLoader.ResolveImport` patterns)
  - Validate module declarations match expected paths
- Write tests in `internal/pkg/discover_test.go`
- Create test fixture: `testdata/packages/multi_module/` with 3 modules + ailang.toml

**Acceptance Criteria:**
- `DiscoverPackageSources(".")` returns all `.ail` files with module paths
- Detects missing modules (declared in exports but no source file)
- Detects orphan files (source exists but not in exports)
- Works with both `src/` layout and flat layout
- All tests passing, lint clean

**Risks:**
- Module path to file path mapping may have edge cases -- mitigated by reusing existing PackageLoader patterns

---

### M2: `ailang check --package` (~150 LOC, 3 hours)

**Goal:** Wire `--package` flag into check command, routing through module pipeline for cross-module type resolution.

**Tasks:**
- Add `--package` flag to `cmd/ailang/check.go`
- Create `checkPackageWithContext()` function that:
  - Calls `DiscoverPackageSources()` from M1
  - Picks an entry file (or creates a synthetic entry that imports all modules)
  - Routes through `pipeline.RunWithContext()` in module mode (not single-file mode)
  - Collects type errors with cross-module provenance
- Add export validation: verify `[exports]` modules compile successfully
- Wire up JSON output and quiet flags
- Integration test with multi-module test package

**Acceptance Criteria:**
- `ailang check --package .` resolves cross-module types
- Type errors show source module context
- Export validation warns about missing/orphan modules
- JSON output works (`--json`)
- Exit code 0 pass, 1 fail
- `make test` passes, `make lint` clean

**Risks:**
- Module pipeline may need `DryLink: true` to skip evaluation -- already supported in Config

---

### M3: `ailang test --package` (~150 LOC, 3 hours)

**Goal:** Wire `--package` flag into test command, running test files with package modules available for cross-module imports.

**Tasks:**
- Add `--package` flag to `cmd/ailang/test.go` (via main.go routing)
- Create `runPackageTests()` function that:
  - Calls `DiscoverPackageSources()` from M1
  - Identifies `*_test.ail` files from discovered sources
  - Compiles package modules first (shared type environment)
  - Runs test files with package modules available as imports
  - Aggregates results using existing `SuiteResult`
- Wire up existing reporter (human/JSON) and exit codes
- Integration test with test package fixture

**Acceptance Criteria:**
- `ailang test --package .` discovers and runs `*_test.ail` files
- Test files can import package modules (cross-module resolution)
- Results aggregated with pass/fail per test function
- Human and JSON output work
- Exit code 0/1 for CI
- `make test` passes, `make lint` clean

**Risks:**
- Test framework (`RunTestsFromFile`) may need adaptation for module-aware execution -- may need to use `pipeline.Run` in eval mode instead

---

### M4: Documentation, Examples, Verification (~100 LOC, 2 hours)

**Goal:** Update docs, create example package with tests, verify on real packages.

**Tasks:**
- Update CLI help text for both commands
- Add test fixture package to `testdata/packages/` with:
  - `ailang.toml` with 3 modules
  - Source files with cross-module types
  - `*_test.ail` files that import package modules
- Test against `ailang-packages/packages/` (billing_store, gcp_auth) if available locally
- Update CHANGELOG.md
- Update design docs status to "Implemented"

**Acceptance Criteria:**
- `ailang check --help` and `ailang test --help` show --package flag
- Test fixture package passes both check and test
- CHANGELOG updated
- Design docs updated
- `make verify-examples` passes

---

## Parallelization Analysis

**Can M2 and M3 run in parallel?** Yes, with caveats:

- M1 (discovery) must complete first -- both M2 and M3 depend on it
- M2 (check) and M3 (test) touch **different files** (`check.go` vs `test.go`) and are safe to parallelize
- Both use `DiscoverPackageSources()` from M1 (read-only, no conflict)
- M4 (docs) depends on both M2 and M3

**Recommended execution order:**
```
M1 (discovery) ──→ M2 (check --package) ──→ M4 (docs)
                ╲                          ╱
                 → M3 (test --package)  ──→
```

M2 and M3 can run as **parallel agents in separate worktrees** since they modify non-overlapping files.

## Success Metrics

- `ailang check --package .` works on multi-module packages
- `ailang test --package .` discovers and runs package tests
- Cross-module type resolution works (types from module A visible in module B)
- CI-friendly: JSON output, proper exit codes
- All existing tests still pass (`make test`)
- Lint clean (`make lint`)
- CHANGELOG.md updated

## Open Questions

1. ~~Should `ailang check --package` also resolve external dependencies?~~ **No** -- intra-package only (per design doc)
2. Test file location convention: same dir as source, or `tests/` subdir? **Both** -- discover `*_test.ail` anywhere in package dir
3. Should M3 reuse `ailangTesting.RunTestsFromFile` or the full pipeline? **Try RunTestsFromFile first**, fall back to pipeline if cross-module imports aren't resolved

## Notes

- Estimated 750 LOC total (550 implementation + 200 tests)
- Low risk: all compilation infrastructure exists, this is integration work
- The module pipeline (`runModuleWithContext`) already does everything we need for cross-module resolution
- `loader.ModuleLoader.LoadAll()` handles transitive dependency loading with caching

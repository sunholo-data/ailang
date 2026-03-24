# M-DX-PKG-TEST: Package-Level Test Runner

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 4 days (~16 hours implementation + 8 hours testing + 4 hours docs)
**Dependencies**: M-DX-PKG-CHECK (package-level type checking -- shares package discovery and cross-module resolution)
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Test results are deterministic given same package state -- no implicit ordering or shared state |
| A2: Replayability | +1 | Test runs produce structured output that can be diffed across versions |
| A3: Effect Legibility | 0 | No change to effect system -- test functions declare their own effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Automated test discovery ensures packages are verifiable without manual scripts |
| A6: Safe Concurrency | 0 | Tests run sequentially (parallel execution deferred) |
| A7: Machines First | +1 | Machine-readable JSON output, conventional naming, CI-friendly exit codes |
| A8: Minimal Syntax | 0 | No new syntax -- uses existing `func test_*` naming convention |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Test files compose with package modules through standard imports |
| A11: Structured Failure | +1 | Test failures reported with function name, file location, and error context |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** -- Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Tests are isolated -- each test function gets fresh evaluator state
- [x] A3 (Effects): Test functions declare their effects explicitly (e.g., `! {IO, Net}`)
- [x] A4 (Authority): No ambient access -- tests use same capability model as production code
- [x] A7 (Machines First): JSON output mode, conventional naming for discovery, exit codes for CI

---

## Problem Statement

**Current State:**

There is no `ailang test` equivalent for packages. The billing package development (26 `.ail` files across 6 packages) revealed that:

- Self-contained test files can only be run individually with `ailang run test_file.ail`
- No test discovery convention exists (no standard naming like `*_test.ail`)
- No way to validate that exports declared in `ailang.toml` actually exist and work
- No coverage reporting per module
- Package authors must write custom shell scripts for CI test workflows
- No standard assertion functions (each test file reinvents `assertEqual`)

**Impact:**
- Package developers (primary): No standard test workflow, each package reinvents CI
- CI/CD pipelines: No `ailang test` step to add to build pipelines
- Package consumers: No way to verify package quality before depending on it

---

## Goals

**Primary Goal:** Provide a standard `ailang test --package` command that discovers, runs, and reports on all tests in an AILANG package.

**Success Metrics:**
- `ailang test --package .` discovers and runs all `*_test.ail` files
- Test functions discovered by `test_` prefix convention
- Cross-module resolution works (test files can import package modules)
- Pass/fail reported per test function with file and line context
- Exit code 0/1/2 for CI integration
- Runs billing_store tests in <5s

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `test_` prefix convention (not annotations) | Sets permanent convention for all AILANG test code | human | design | high |
| Test files require explicit imports | Determines whether tests are self-contained or magic | human | design | high |
| Each test function gets isolated evaluator state | Affects test reliability and parallelizability | human | design | med |
| Reuse M-DX-PKG-CHECK package discovery | Determines dependency ordering and implementation plan | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] `test_` prefix for test function discovery (not `@test` annotation -- AILANG has no annotation syntax)
- [x] Test files require explicit imports (consistent with AILANG's explicit-everything philosophy)
- [x] Each test function gets isolated evaluator state (determinism axiom)
- [ ] Should `std/test` assertion module be part of this work or separate?
- [ ] Test file location: same directory as source, or `tests/` subdirectory?

## Deferred Decisions

- Test output formatting details -- agent may choose
- Coverage implementation approach -- agent may choose (instruction counting vs block tracking)
- `--filter` glob pattern syntax -- agent may choose
- Whether to support test setup/teardown functions -- deferred to future work
- Parallel test execution -- deferred to future optimization

---

## Solution Design

### Overview

Add `ailang test --package [dir]` that builds on M-DX-PKG-CHECK's package discovery and cross-module resolution. The test runner discovers `*_test.ail` files, compiles them with the package's modules available, identifies `test_` prefixed functions, executes each in an isolated evaluator, and reports results.

### Architecture

**Components:**
1. **TestDiscoverer**: Finds `*_test.ail` files and parses for `func test_*` declarations (builds on PackageDiscoverer from M-DX-PKG-CHECK)
2. **TestExecutor**: Compiles test files with package modules, runs each test function in isolated evaluator
3. **TestReporter**: Formats results as human-readable or JSON, computes pass/fail counts
4. **ExportValidator**: (reused from M-DX-PKG-CHECK) Validates exports exist

### Implementation Plan

**Phase 1: Test Discovery** (~4 hours)
- [ ] Scan package directory for `*_test.ail` files
- [ ] Parse each test file to find `func test_*` declarations
- [ ] Map test files to corresponding source modules
- [ ] Validate test function signatures (must return `()`)
- [ ] Unit tests for discovery

**Phase 2: Package-Aware Test Execution** (~8 hours)
- [ ] Compile all package modules (reuse M-DX-PKG-CHECK)
- [ ] Compile each test file with package module interfaces available
- [ ] Execute each `test_` function in fresh evaluator instance
- [ ] Capture result: pass (no error), fail (assertion error), error (runtime error)
- [ ] Timeout per test function (default 30s)
- [ ] Integration tests with multi-module test packages

**Phase 3: Reporting** (~6 hours)
- [ ] Human-readable output: pass/fail per function with timing
- [ ] JSON output mode (`--json`) for CI integration
- [ ] Summary: N passed, M failed, K errors
- [ ] Exit code: 0 all pass, 1 any fail, 2 infrastructure error
- [ ] `--coverage` flag: track which module functions are called during tests
- [ ] Golden file tests for output format

**Phase 4: CLI Integration** (~2 hours)
- [ ] Add `test` subcommand to CLI (or `--package` flag to existing test)
- [ ] Wire up `--coverage`, `--filter`, `--json` flags
- [ ] Help text and examples
- [ ] Integration test with real package

### Files to Modify/Create

**New files:**
- `internal/pkg/test_runner.go` (~250 LOC) - TestDiscoverer, TestExecutor, TestReporter
- `internal/pkg/test_runner_test.go` (~350 LOC) - Unit and integration tests
- `cmd/ailang/test.go` (~80 LOC) - CLI command handler
- `testdata/packages/test_package/` - Test fixture with source and test files

**Modified files:**
- `cmd/ailang/main.go` (+5 LOC) - Register test subcommand
- `internal/pkg/checker.go` (+10 LOC) - Export discovery helpers (shared with test runner)

---

## Examples

### Example: Running Package Tests

```
$ ailang test --package .

Testing package sunholo/billing_store...

  core_test.ail
    test_getUsage_returns_result       PASS (12ms)
    test_getUsage_invalid_project      PASS (8ms)
    test_setUsage_increments           PASS (15ms)

  entitlement_test.ail
    test_checkCapability_allowed       PASS (10ms)
    test_checkCapability_denied        FAIL (5ms)
      --> entitlement_test.ail:22
      expected Ok("allowed"), got Err("quota exceeded")

  5 tests: 4 passed, 1 failed (50ms)
```

### Example: Test File Convention

```ailang
module sunholo/billing_store/tests/core

import std/result (Ok, Err)
import pkg/sunholo/billing_store/core (getUsage, setUsage)

export func test_getUsage_returns_result() -> () ! {IO, Net} =
  let result = getUsage("project-123")
  match result {
    Ok(_) -> println("pass")
    Err(e) -> println("FAIL: " ++ e)
  }

export func test_setUsage_increments() -> () ! {IO, Net} =
  let _ = setUsage("project-123", 100)
  let result = getUsage("project-123")
  match result {
    Ok(usage) -> println("pass: " ++ toString(usage))
    Err(e) -> println("FAIL: " ++ e)
  }
```

### Example: CI Integration

```yaml
# .github/workflows/test.yml
- name: Test package
  run: |
    ailang test --package . --json > results.json
    # Exit code 0 = all pass, 1 = failures
```

---

## Success Criteria

- [ ] `ailang test --package .` discovers and runs all `*_test.ail` files
- [ ] Test functions discovered by `test_` prefix
- [ ] Cross-module type resolution works (test can import package modules)
- [ ] Pass/fail per test function with location context
- [ ] Exit codes: 0 (pass), 1 (fail), 2 (error) -- CI-compatible
- [ ] `--json` output mode for machine consumption
- [ ] `--coverage` flag reports which module functions were called
- [ ] Runs billing_store tests in <5s
- [ ] All tests passing (90%+ coverage on new code)
- [ ] `make verify-examples` still passes
- [ ] CHANGELOG.md updated

---

## Testing Strategy

**Unit tests:**
- Test discovery (finds `*_test.ail` files, parses `test_` functions)
- Test isolation (each function gets fresh state)
- Reporter output formatting (human and JSON)

**Integration tests:**
- Multi-module test package: source modules + test files
- Test that fails (verify failure reporting)
- Test with cross-module types
- Golden file tests for output format
- Coverage tracking accuracy

**Manual testing:**
- Create test files for `ailang-packages/packages/billing_store/`
- Run in CI pipeline (GitHub Actions)

---

## Non-Goals

- **`std/test` assertion library**: Useful but separate scope -- can be added independently
- **Test setup/teardown**: `beforeEach`/`afterEach` patterns -- deferred
- **Parallel test execution**: Sequential first, parallelize later
- **Mocking/stubbing framework**: Out of scope -- test against real implementations
- **Watch mode**: `ailang test --watch` -- deferred

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M-DX-PKG-CHECK not yet implemented | High | Can implement discovery/execution independently; cross-module resolution is the shared piece |
| Test isolation overhead (fresh evaluator per test) | Medium | Profile; if too slow, share read-only module interfaces |
| No assertion stdlib makes tests verbose | Medium | Document patterns; `std/test` can follow as independent work |
| Test file naming conflicts with source files | Low | Convention is `*_test.ail` -- clear separation |

---

## Related Documents

- `design_docs/planned/m-dx-package-check.md` - Prerequisite: package discovery and cross-module resolution
- `design_docs/planned/docparse-billing/dx-improvements-from-billing-packages.md` - Parent issue (#5)
- `design_docs/implemented/v0_7_0/m-coord-taskchain-tests-sprint-plan.md` - Prior art on test infrastructure
- `design_docs/archive/analysis/v0_3_14_tier0_completion.md` - Test coverage analysis patterns

---

## Future Work

- **`std/test` module**: `assertEqual`, `assertOk`, `assertErr`, `assertThrows` functions
- **Watch mode**: Re-run tests on file changes
- **Parallel execution**: Run independent test functions concurrently
- **Test setup/teardown**: `beforeAll`, `beforeEach` lifecycle hooks
- **Coverage visualization**: HTML coverage reports per module
- **Package quality score**: Combine type check + test pass rate + coverage into single metric

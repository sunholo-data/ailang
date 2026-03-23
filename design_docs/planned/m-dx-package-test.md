# M-DX-PKG-TEST: Package-Level Test Runner

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium — no standard test workflow for packages)
**Estimated**: 3-4 days
**Dependencies**: M-DX-PKG-CHECK (package-level type checking)
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Executive Summary

There is no `ailang test` equivalent for packages. Self-contained test files can be run individually, but there's no way to run all tests in a package, validate that exports match ailang.toml, or get coverage reports. This blocks standard CI/CD workflows for AILANG packages.

---

## Problem

- No standard way to run all tests in a package
- Self-contained test files can only be run individually with `ailang run test_file.ail`
- No test discovery convention (no standard naming like `*_test.ail`)
- No way to validate that exports declared in ailang.toml actually exist
- No coverage reporting per module
- Package authors must write custom scripts for CI

## Proposed Solution

Add `ailang test --package [dir]` that:
1. Reads `ailang.toml` for module list and package metadata
2. Discovers test files by convention (`*_test.ail`)
3. Loads all package modules for cross-module type resolution
4. Runs each test file with the package's modules available
5. Reports results with pass/fail per test file and per test function
6. Optionally reports coverage per module

### CLI Interface

```
ailang test --package .                     # Run all tests in current package
ailang test --package . --coverage          # With coverage report
ailang test --package . --filter "billing"  # Filter test files
ailang test file_test.ail                   # Existing: run single test file
```

### Test File Convention

Test files should follow the naming pattern `*_test.ail` and contain test functions:
```
module sunholo/billing_store/tests

import sunholo/billing_store/core (getUsage)

export func test_getUsage_returns_result() -> () ! {IO, Net} =
  let result = getUsage("project-123")
  -- assertions here
```

## Implementation Plan

### Phase 1: Test Discovery (0.5 day)
- Scan package directory for *_test.ail files
- Map test files to their corresponding source modules
- Parse test files to find test functions (func test_*)

### Phase 2: Package-Aware Test Execution (1.5 days)
- Load all package modules into shared environment
- Run each test file with cross-module resolution
- Capture test function results (pass/fail/error)
- Handle test isolation (each test starts fresh)

### Phase 3: Reporting (1 day)
- Human-readable output with pass/fail per function
- Machine-readable output (JSON) for CI integration
- Coverage tracking per module
- Exit code: 0 all pass, 1 any fail, 2 infrastructure error

### Phase 4: Export Validation (0.5 day)
- Verify all ailang.toml exports exist in compiled modules
- Warn about exported symbols that don't have test coverage
- Validate effect bounds

## Testing Strategy

- Self-hosted: test the test runner using AILANG test packages
- Integration tests with multi-module packages
- CI integration test

## Open Questions

1. Should test functions be required to have `test_` prefix, or use a `@test` annotation?
2. Should test files automatically have access to all package modules, or require explicit imports?
3. How to handle tests that need external services (mocking/stubbing)?
4. Should there be an assertion stdlib (`std/test` with `assertEqual`, `assertOk`, etc.)?

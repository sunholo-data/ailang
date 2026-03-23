# M-DX-PKG-CHECK: Package-Level Type Checking

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 4 days (~16 hours implementation + 8 hours testing + 4 hours docs)
**Dependencies**: None
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics or runtime behavior |
| A2: Replayability | 0 | No impact on traces or replay |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +2 | Enables package-scoped type checking that was previously impossible -- moves verification from "runtime failure" to "compile-time check" |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | CI/CD and AI agents can validate packages without running them |
| A8: Minimal Syntax | 0 | No new syntax -- CLI flag only |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Validates that modules compose correctly within a package before publishing |
| A11: Structured Failure | +1 | Type errors reported with cross-module context (which module defines the type, which uses it) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** -- Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly improves machine analysis capability

---

## Problem Statement

**Current State:**

`ailang check capability_check.ail` fails because types defined in other modules within the same package (e.g., `Entitlements` from `entitlement.ail`) cannot be resolved in standalone file check. This means:

- No way to type-check a package as a unit -- each file checked in isolation
- CI can only validate self-contained files (no cross-module type references)
- Cross-module type errors only surface at runtime through package imports
- Package authors discovered during billing package creation (26 `.ail` files across 6 packages) that CI validation is effectively impossible for real packages

**Impact:**
- Package developers (primary): Cannot validate packages before publishing
- CI/CD pipelines: No standard check step for package quality gates
- AI agents: Cannot verify package correctness during development

---

## Goals

**Primary Goal:** Enable type-checking entire AILANG packages as a unit, catching cross-module type errors at check time instead of runtime.

**Success Metrics:**
- `ailang check --package .` resolves types across all modules in a package
- Cross-module type errors reported with source module context
- Export declarations in `ailang.toml` validated against compiled modules
- Exit code 0/1 suitable for CI integration
- Runs in <2s for packages with 10 modules

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Check intra-package types only (not external deps) | Determines scope and complexity -- external deps add dependency resolution | human | design | high |
| Reuse existing pipeline compile for type resolution | Avoids building a separate type resolver -- leverage what exists | agent | implementation | med |
| Validate exports match ailang.toml | Determines if this is "type check" or "package validate" | human | design | low |
| Error format includes cross-module provenance | Affects error message structure and downstream tooling | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Check intra-package types only (external deps not resolved -- would require lockfile/registry)
- [x] Reuse existing pipeline compile infrastructure
- [ ] Error output format: human-readable only, or also JSON for CI?
- [ ] Should `ailang check --package` be the same command or a new `ailang pkg check`?

## Deferred Decisions

- CLI output formatting (human vs JSON) -- agent may choose, can add JSON later
- Internal function naming and file organization -- agent may choose
- Progress reporting for large packages -- agent may choose
- Whether to parallelize module compilation -- deferred to future optimization

---

## Solution Design

### Overview

Add a `--package` flag to the existing `ailang check` command that reads `ailang.toml`, discovers all `.ail` source files, compiles them together in a shared type environment (reusing the existing pipeline), and validates that exported symbols match the manifest. This is essentially "compile without linking" -- the same resolution that happens at import time, but triggered explicitly.

### Architecture

**Components:**
1. **PackageDiscoverer**: Reads `ailang.toml`, finds all `.ail` files, builds module list
2. **PackageChecker**: Compiles all modules in topological order with shared type env
3. **ExportValidator**: Compares compiled interfaces against `[exports]` in manifest
4. **CheckReporter**: Formats results for human and CI consumption

### Implementation Plan

**Phase 1: Package Discovery** (~4 hours)
- [ ] Read `ailang.toml` to get module list from `[exports]` section
- [ ] Discover all `.ail` source files in package directory (recursive)
- [ ] Map source files to module paths (using `module` declarations)
- [ ] Build intra-package dependency graph from `import` declarations
- [ ] Detect and report circular module references
- [ ] Unit tests for discovery and graph building

**Phase 2: Cross-Module Type Resolution** (~8 hours)
- [ ] Load all package modules into a shared compile context
- [ ] Compile modules in topological order (leaves first)
- [ ] Pass compiled interfaces from earlier modules to later ones
- [ ] Report type errors with cross-module provenance (e.g., "type `Entitlements` defined in `entitlement.ail` at line 5")
- [ ] Handle forward references gracefully (error, not crash)
- [ ] Integration tests with multi-module test packages

**Phase 3: Export Validation** (~4 hours)
- [ ] Verify every symbol in `[exports].modules` has a corresponding compiled module
- [ ] Warn about modules not listed in exports
- [ ] Validate effect bounds: modules don't exceed `[effects].max`
- [ ] Golden file tests for error messages

**Phase 4: CLI Integration** (~4 hours)
- [ ] Add `--package` flag to `check` command in `cmd/ailang/`
- [ ] Default to current directory if no path given
- [ ] Exit code: 0 = pass, 1 = type errors, 2 = infrastructure error
- [ ] Human-readable output with color
- [ ] Integration test with real package

### Files to Modify/Create

**New files:**
- `internal/pkg/checker.go` (~200 LOC) - PackageChecker and PackageDiscoverer
- `internal/pkg/checker_test.go` (~300 LOC) - Unit and integration tests
- `testdata/packages/multi_module/` - Test fixture package

**Modified files:**
- `cmd/ailang/check.go` (+30 LOC) - Add --package flag handling
- `internal/pkg/manifest.go` (+20 LOC) - Helper to list expected modules

---

## Examples

### Example: Checking a Multi-Module Package

```
$ ailang check --package .

Checking package sunholo/billing_store (5 modules)...

  sunholo/billing_store/types       OK
  sunholo/billing_store/firestore   OK
  sunholo/billing_store/core        ERROR

  error[TYPE]: undefined type 'Entitlement' in core.ail:15
    --> src/core.ail:15:22
    |
    | func checkAccess(e: Entitlement) -> bool =
    |                      ^^^^^^^^^^^
    |
    = help: did you mean 'Entitlements' (defined in src/types.ail:8)?

  Checked 5 modules: 4 passed, 1 failed
```

### Example: Export Validation

```
$ ailang check --package .

  warning[EXPORT]: module 'sunholo/billing_store/internal' is not listed in [exports]
    --> ailang.toml:4
    |
    | modules = ["sunholo/billing_store/core", "sunholo/billing_store/types"]
    |
    = note: add to exports or mark as internal

  Checked 3 modules: 3 passed, 0 failed (1 warning)
```

---

## Success Criteria

- [ ] `ailang check --package .` runs on billing_store package without errors
- [ ] Cross-module type references resolved correctly
- [ ] Meaningful error messages with source module context
- [ ] Export validation catches missing/extra modules
- [ ] Exit codes suitable for CI (`0` pass, `1` fail, `2` infra error)
- [ ] Completes in <2s for 10-module package
- [ ] All tests passing (90%+ coverage on new code)
- [ ] `make verify-examples` still passes
- [ ] CHANGELOG.md updated

---

## Testing Strategy

**Unit tests:**
- Package discovery (finds all .ail files, builds module graph)
- Circular reference detection
- Export validation (missing modules, extra modules, effect violations)

**Integration tests:**
- Multi-module test package with cross-module types
- Package with type errors (verify error messages)
- Package with missing exports
- Golden file tests for error output

**Manual testing:**
- Run on `ailang-packages/packages/billing_store/`
- Run on `ailang-packages/packages/gcp_auth/`

---

## Non-Goals

- **External dependency resolution**: Only intra-package types -- not resolving types from `[dependencies]`
- **Incremental checking**: Full recheck every time (no caching of module interfaces)
- **Language server integration**: LSP support deferred to future work
- **Auto-fix suggestions**: Beyond "did you mean X?" for typos

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing pipeline assumes single-file compilation | High | Audit pipeline entry points; may need a `CompilePackage()` wrapper |
| Module ordering wrong causes false errors | Medium | Use topological sort; detect and report cycles clearly |
| Performance on large packages | Low | Profile on billing packages (26 files); optimize if >5s |

---

## Related Documents

- `design_docs/planned/m-smt-cross-module-types.md` - Z3 cross-module type resolution (related but SMT-specific)
- `design_docs/implemented/v0_9_0/m-module-scope.md` - Module scope isolation (foundation for this work)
- `design_docs/planned/docparse-billing/dx-improvements-from-billing-packages.md` - Parent issue (#2)

---

## Future Work

- **Incremental checking**: Cache module interfaces to avoid recompiling unchanged modules
- **External dependency resolution**: Resolve types from installed packages
- **LSP integration**: Real-time cross-module type errors in editor
- **`ailang test --package`**: Builds on this for package-level test runner (see `m-dx-package-test.md`)

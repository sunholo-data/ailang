# Sprint Plan: M-CODEGEN-COMPILE-GATE

**Phase 0 of M-CODEGEN-IR-LAYERS**

## Summary

| Field | Value |
|-------|-------|
| **Goal** | Add `go build` compile-check gate to codegen pipeline — catch broken generated code at codegen time, not downstream |
| **Duration** | 1 day (3-4 hours implementation) |
| **Risk Level** | Low — additive feature, no existing behavior changed |
| **Total LOC** | ~200 (100 impl + 50 test + 50 CI integration) |
| **Dependencies** | M-CODEGEN-REGISTRY-ONLY (v0.9.2 ✅) |
| **Design Doc** | `design_docs/planned/v0_10_0/m-codegen-ir-layers.md` (Phase 0) |

## Motivation

Every DocParse codegen debugging session (3 cycles in March 2026) followed the same pattern:
1. `ailang compile --emit-go` succeeds silently
2. User runs `go build` on output and gets errors
3. Back-and-forth to fix codegen
4. Repeat

The fix: run `go build` inside the codegen pipeline itself and report errors as codegen diagnostics.

## Current State

- `cmd/ailang/compile.go:483` prints "Compilation complete!" after generating all `.go` files
- No `go.mod` is generated — user must run `go mod init` manually
- No verification that generated code compiles
- The `--verify-contracts` flag exists as precedent for optional verification flags

## Milestones

### M1: Compile Gate in `compile.go` (~100 LOC)

**Files**: `cmd/ailang/compile.go`

**Tasks**:
1. Add `--verify-go` flag (default: true when `go` is in PATH, skipped otherwise)
2. After all `.go` files are generated (line ~480), if `--verify-go`:
   a. Create temporary `go.mod` in output dir if one doesn't exist
   b. Run `go build ./...` in output dir
   c. If build fails: parse `go build` stderr, report as codegen diagnostics
   d. If build succeeds: print verification success message
   e. Clean up temporary `go.mod` if we created it
3. Report errors with file/line context: "codegen: runtime.go:42: undefined: GetNumber"
4. Exit with non-zero status on verification failure (unless `--no-verify-go`)

**Acceptance Criteria**:
- `ailang compile --emit-go examples/runnable/contracts/basic.ail` succeeds with verification
- Missing function references produce codegen error diagnostics
- `--no-verify-go` flag skips verification
- Verification skipped gracefully when `go` binary not available

### M2: CI Integration + Example Verification (~50 LOC)

**Files**: `Makefile`, possibly `scripts/verify_go_codegen.sh`

**Tasks**:
1. Add `make compile-examples-go` target that compiles all runnable examples to Go and verifies each with `go build`
2. Add this target to `make ci` pipeline
3. Document which examples currently pass/fail the compile gate

**Acceptance Criteria**:
- `make compile-examples-go` runs all examples through codegen + go build
- Reports pass/fail per example
- Integrated into CI pipeline

### M3: Integration Test Fix (~50 LOC)

**Files**: `internal/gen/golang/contracts_integration_test.go`

**Tasks**:
1. Fix the pre-existing `TestContractViolation_Integration` failure (module prefix naming issue — test expects `Absolute` but codegen emits `basic__Absolute`)
2. Ensure the test uses the compile gate internally

**Acceptance Criteria**:
- `TestContractViolation_Integration` passes
- Test verifies generated Go code compiles and runs correctly

## Day-by-Day Plan

### Day 1 (3-4 hours)

| Time | Task | Milestone |
|------|------|-----------|
| Hour 1 | Implement `--verify-go` flag + `go build` gate in compile.go | M1 |
| Hour 2 | Error parsing + diagnostics formatting + edge cases (no `go` binary, existing `go.mod`) | M1 |
| Hour 3 | Add `make compile-examples-go` target + CI integration | M2 |
| Hour 4 | Fix `TestContractViolation_Integration` + final verification | M3 |

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `go build` slow on large generated projects | Low | Only runs with `--verify-go`; can be skipped with `--no-verify-go` |
| Temporary `go.mod` conflicts with user's module setup | Low | Only create if doesn't exist; clean up after |
| Some examples have legitimate `go build` failures (effect stubs) | Medium | Report as warnings, not errors; track known failures |

## Success Metrics

- Zero silent codegen failures: every broken `go build` is caught at `ailang compile` time
- `make ci` includes Go codegen verification
- `TestContractViolation_Integration` passes

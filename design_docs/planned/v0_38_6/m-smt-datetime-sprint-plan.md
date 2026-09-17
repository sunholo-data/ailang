# Sprint Plan: M-SMT-DATETIME — Exact UTC Day Arithmetic in Z3

**Sprint ID:** `M-SMT-DATETIME`  
**Planning input:** Daneel report `inbox_1789394686844_585753b0` (2026-09-14)  
**Target:** v0.38.6 development line  
**Duration:** 2 engineering days (about 12 hours)  
**Estimated change:** 420 LOC (140 implementation, 240 tests/examples, 40 docs)  
**Risk level:** Medium

## Summary

Make the linear UTC-millisecond subset of `std/datetime` usable in `requires` and
`ensures`: `addDays`, `startOfDay`, and `weekday`. Lower these operations to exact
integer SMT-LIB formulas while keeping `addMonths`, `addYears`, parsing,
formatting, and general `_dt_add` calls opaque. This unblocks Daneel's 90-day
authority-window proof and future deterministic date-parser proofs without
claiming that Go `time.AddDate` is generally linear.

This is a verifier feature, not a runtime datetime change. Existing evaluator and
bytecode behavior remain the semantic oracle.

## Current Status Analysis

- `std/datetime.ail` is pure and represents UTC Unix milliseconds.
- `addDays(ts, days)` delegates to `_dt_add(ts, 0, 0, days)`.
- `weekday` and `startOfDay` currently use `_dt_parts`/`_dt_make`, outside the SMT fragment.
- `ResolveStdlibToBuiltin` in `internal/smt/types.go` currently maps only `std/string` and `std/list`.
- Fragment detection, callee resolution, and application codegen all consume this mapping; a codegen-only special case would be incomplete.
- Daneel's literal 90-day expression verifies, while the equivalent `addDays(now, 90)` contract is skipped.
- The checkout has insufficient seven-day history for a trustworthy LOC/day calculation. The estimate uses comparable focused SMT work plus a 25% uncertainty buffer.

## Scope and Systemic Audit

Milestone 1 inventories every pure `_dt_*` builtin and classifies it as exactly
encodable, conditionally encodable, or intentionally opaque.

| Public function | Exact SMT meaning | Decision |
|---|---|---|
| `addDays(ts, n)` | `ts + n * 86400000` | Encode |
| `startOfDay(ts)` | `ts - mod(ts, 86400000)` | Encode after runtime parity gate |
| `weekday(ts)` | `mod(div(ts, 86400000) + 4, 7)` | Encode after runtime parity gate |
| `addMonths`, `addYears` | Go `AddDate` calendar normalization | Keep opaque |
| `startOfWeek` | Composition of accepted functions | Verify normal composition; no bespoke primitive |
| `diffDays` | Division semantics require negative-domain audit | Defer unless parity is proved |
| extraction, construction, parsing, formatting | Calendar/string/ADT semantics | Keep opaque |

Negative timestamps are mandatory because SMT `div`/`mod` semantics must match
runtime behavior. Encoding `_dt_add` directly is forbidden unless year and month
arguments are statically proven zero; mapping the public wrapper is preferred.

The reported zero-argument-callee failure is a distinct resolver defect. Capture
a minimal regression and route it as separate work unless the narrow mapping
change demonstrably resolves it.

## Proposed Milestones

### M1 — Freeze semantics and add red tests

**Duration:** 0.5 day (3 hours)  
**Estimated:** 20 LOC fixtures + 70 LOC tests = 90 LOC

**Example files to update/create:**

- `internal/smt/types_test.go`
- `internal/smt/encodable_test.go`
- `internal/smt/codegen_datetime_test.go` (new)
- executor-owned zero-argument reproduction fixture

**Tasks:**

1. Add failing tests for direct and curried references to all three datetime functions across fragment detection and SMT generation.
2. Add parity vectors at `-86400001`, `-86400000`, `-1`, `0`, `1`, a leap-day timestamp, and large positive/negative offsets.
3. Compare formulas against runtime implementations and document excluded operations.
4. Reproduce and disposition the zero-argument-callee issue.

**Acceptance criteria:**

- [ ] Tests fail for the intended missing datetime capability before implementation.
- [ ] Each accepted formula has positive, negative, and boundary parity cases.
- [ ] Every `_dt_*` builtin has an explicit encode/defer classification.
- [ ] The zero-argument issue is linked separately or proven incidentally fixed.

### M2 — Implement one typed datetime encoding path

**Duration:** 0.75 day (4 hours)  
**Estimated:** 140 LOC implementation + 70 LOC tests = 210 LOC

**Example files to update/create:**

- `internal/smt/types.go`
- `internal/smt/codegen_apps.go`
- `internal/smt/encodable.go`
- `internal/smt/codegen_datetime_test.go`

**Tasks:**

1. Add a dedicated datetime mapping/spec rather than treating multi-operation formulas as ordinary `BuiltinToSMTOp` entries.
2. Route direct and curried calls through the resolver shared by classification, callee collection, and codegen.
3. Validate arity and recursively encode arguments; malformed calls return structured errors.
4. Keep unapproved datetime functions explicitly unencodable.

**Acceptance criteria:**

- [ ] All three functions emit the frozen formulas for direct and curried Core shapes.
- [ ] Fragment checker and encoder agree on mapped and unmapped operations.
- [ ] Negative timestamps are runtime-equivalent.
- [ ] `addMonths` and `addYears` remain named skips.
- [ ] No fallback sort, uninterpreted function, or unconstrained constant is emitted.
- [ ] `go test ./internal/smt/...` passes.

### M3 — End-to-end proofs, example, and documentation

**Duration:** 0.75 day (5 hours)  
**Estimated:** 100 LOC tests/example + 20 LOC docs = 120 LOC

**Example files to update/create:**

- `examples/runnable/contracts/datetime_smt.ail` (new)
- `cmd/ailang/verify_test.go` or nearest verifier integration test
- relevant verifier guide under `docs/docs/guides/`
- `changelogs/v0.32-current.md`

**Tasks:**

1. Run `ailang prompt` before editing `.ail`, then validate the example with `ailang check`.
2. Add Daneel's `inWindow(now, target)` contract and require VERIFIED rather than SKIPPED.
3. Add contracts for day alignment/range and weekday range/epoch anchor, plus an opaque `addMonths` control.
4. Run real-Z3 tests and document the supported fragment and exclusions.

**Acceptance criteria:**

- [ ] `inWindow` using `addDays(now, 90)` is VERIFIED by real Z3.
- [ ] The example type-checks and runs; expected verify counts are asserted.
- [ ] `startOfDay` and `weekday` contracts verify over the symbolic negative domain.
- [ ] The `addMonths` control remains a named SKIP.
- [ ] Focused verifier suites run with Z3 tests actually executed.
- [ ] `make fmt`, `make lint`, `make test`, and applicable boundary checks pass.

## Day-by-Day Plan

### Day 1

- Complete the semantic inventory and red-first parity/codegen tests (M1).
- Implement the shared datetime mapping and special encoder (M2).
- Finish with focused SMT tests green and audit against accidental scope expansion.

### Day 2

- Add and validate the runnable proof example and CLI integration tests (M3).
- Run real-Z3 controls, full tests, formatting, lint, and boundary checks.
- Update docs/changelog and record the zero-argument follow-up disposition.

## Success Metrics

- Three public datetime functions enter the decidable SMT fragment with exact runtime-parity formulas.
- Daneel's 90-day window changes from SKIPPED to VERIFIED without raw milliseconds.
- At least 12 focused parity/unit cases and one runnable end-to-end example.
- Unsupported calendar operations remain explicit named skips.
- Total change remains near 420 LOC; growth beyond 550 LOC triggers re-planning.

## Dependencies and Risks

- Z3 is required for acceptance; absence is a blocker, not an accepted skip.
- Existing SMT stdlib mapping and cross-module callee-resolution machinery are dependencies.
- Negative-time arithmetic mismatch is gated by parity tests before encoding.
- Mapping must remain shared across classification and generation.
- General zero-argument handling remains separate unless resolved by the narrow fix.

## Deferred Work

- General zero-argument function/callee encoding.
- `addMonths`, `addYears`, component extraction/construction, parsing, and formatting.
- `diffDays` until negative-domain truncation semantics are reconciled.
- Daneel's parser implementation; this sprint supplies its proof substrate only.

## Approval and Handoff

These are planning artifacts only. Implementation begins after human approval and
the explicit instruction **“execute sprint”**, followed by `sprint-executor`.

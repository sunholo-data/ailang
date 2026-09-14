# Sprint Plan: M-SMT-DATETIME — Exact UTC Day Arithmetic in Z3

**Sprint ID:** `M-SMT-DATETIME`
**Planning input:** Daneel report `inbox_1789394686844_585753b0` (2026-09-14)
**Target:** v0.38.6 development line
**Duration:** 2 engineering days (about 12 hours)
**Estimated change:** 420 LOC (140 implementation, 240 tests/examples, 40 docs)
**Risk level:** Medium

## Summary

Make the linear, UTC-millisecond subset of `std/datetime` usable in `requires` and
`ensures`: `addDays`, `startOfDay`, and `weekday`. The encoder will lower these
operations to exact integer SMT-LIB formulas, while deliberately keeping
`addMonths`, `addYears`, parsing, formatting, and general `_dt_add` calls opaque.
This unblocks Daneel's 90-day authority-window proof and its deterministic date
parser without claiming that Go `time.AddDate` is generally linear.

The work is a verifier feature, not a runtime datetime change. Existing evaluator
and bytecode behavior remains the semantic oracle.

## Current Status Analysis

### Confirmed baseline

- `std/datetime.ail` is pure and documents Unix milliseconds in UTC.
- `addDays(ts, days)` forwards to `_dt_add(ts, 0, 0, days)`; with UTC and zero
  year/month deltas this is exactly `ts + days * 86_400_000`.
- `weekday` and `startOfDay` currently go through `_dt_parts`/`_dt_make`, which
  are outside the SMT fragment.
- The existing stdlib mapping seam is `ResolveStdlibToBuiltin` in
  `internal/smt/types.go`; only `std/string` and `std/list` are mapped today.
- Both fragment detection (`encodable.go`) and application codegen
  (`codegen_apps.go`) consume that mapping, so adding only a codegen special case
  would be incomplete.
- Reproduction from Daneel: the literal 90-day expression verifies, while the
  equivalent `addDays(now, 90)` contract skips as an unresolved user function.
- The checkout is shallow and contains one recent commit, so the velocity script
  cannot calculate a trustworthy seven-day LOC/day rate. Estimates use comparable
  focused SMT mapping work and include a 25% uncertainty buffer.

### Scope boundary and systemic audit

The first milestone must inventory every pure `_dt_*` builtin and classify it as
exactly encodable, conditionally encodable, or intentionally opaque. The accepted
surface for this sprint is:

| Public function | Exact SMT meaning | Decision |
|---|---|---|
| `addDays(ts, n)` | `ts + n * 86400000` | Encode |
| `startOfDay(ts)` | `ts - mod(ts, 86400000)` | Encode |
| `weekday(ts)` | `mod(div(ts, 86400000) + 4, 7)` | Encode |
| `addMonths`, `addYears` | Go `AddDate` calendar normalization | Keep opaque |
| `startOfWeek` | Composition of the accepted subset | Verify if normal resolution composes; no bespoke primitive |
| `diffDays` | Runtime integer division semantics need negative-domain audit | Defer unless parity is proved |
| extraction, construction, parsing, formatting | Calendar/string/ADT semantics | Keep opaque |

Negative timestamps are mandatory test inputs because SMT `div`/`mod` semantics
must match the UTC calendar formulas even where AILANG/Go integer remainder rules
may differ. Encoding `_dt_add` directly is forbidden unless its year and month
arguments are statically proven to be zero; mapping the public wrapper is the
preferred narrow seam.

The related zero-argument-callee failure is a distinct general resolver defect.
Milestone 1 records a minimal regression and files/follows a separate work item;
it is not silently fixed in this datetime sprint unless the audit proves the same
small mapping change resolves it without broadening the patch.

## Proposed Milestones

### M1 — Freeze semantics and add red tests

**Duration:** 0.5 day (3 hours)
**Estimated:** 20 LOC test fixtures + 70 LOC tests = 90 LOC

**Example files to update/create:**

- `internal/smt/types_test.go`
- `internal/smt/encodable_test.go`
- `internal/smt/codegen_datetime_test.go` (new)
- temporary executor-owned reproduction fixture for the zero-argument audit

**Tasks:**

1. Capture red unit tests for direct and curried references to all three public
   datetime functions, covering fragment detection and generated SMT-LIB.
2. Add parity vectors at epoch boundaries: `-86400001`, `-86400000`, `-1`, `0`,
   `1`, a leap-day timestamp, and large positive/negative day offsets.
3. Prove the formulas against the runtime implementations for those vectors and
   document why month/year operations remain excluded.
4. Reproduce the zero-argument callee issue and record whether it shares the
   stdlib-mapping path or requires its own resolver change.

**Acceptance criteria:**

- [ ] Tests fail for the intended missing datetime capability before implementation.
- [ ] Each accepted formula has positive, negative, and boundary parity cases.
- [ ] Every `_dt_*` builtin has an explicit encode/defer classification in test or code comments.
- [ ] The zero-argument issue is either linked as a separate item or shown to be fixed incidentally by the narrow change.

### M2 — Implement one typed datetime-special encoding path

**Duration:** 0.75 day (4 hours)
**Estimated:** 140 LOC implementation + 70 LOC tests = 210 LOC

**Example files to update/create:**

- `internal/smt/types.go`
- `internal/smt/codegen_apps.go`
- `internal/smt/encodable.go`
- `internal/smt/codegen_datetime_test.go`

**Tasks:**

1. Add a dedicated datetime mapping/spec type rather than disguising these
   multi-operation formulas as ordinary `BuiltinToSMTOp` entries.
2. Route direct and curried `std/datetime` calls through the same resolver used
   by fragment classification, callee collection, and application codegen.
3. Validate exact arity and recursively encode each argument; malformed calls
   must return a structured encoding error, never malformed SMT.
4. Keep `_dt_add` with nonzero or symbolic month/year components, and every
   unapproved datetime function, explicitly unencodable.

**Acceptance criteria:**

- [ ] `addDays`, `startOfDay`, and `weekday` emit the frozen formulas for direct and curried Core shapes.
- [ ] The fragment checker and encoder agree on every mapped/unmapped datetime operation.
- [ ] Negative timestamps produce runtime-equivalent day and weekday results.
- [ ] `addMonths` and `addYears` still skip with an actionable function name.
- [ ] No fallback sort, uninterpreted function, or unconstrained constant is emitted.
- [ ] `go test ./internal/smt/...` passes.

### M3 — End-to-end proof corpus, example, and documentation

**Duration:** 0.75 day (5 hours)
**Estimated:** 100 LOC tests/example + 20 LOC docs = 120 LOC

**Example files to update/create:**

- `examples/runnable/contracts/datetime_smt.ail` (new)
- `cmd/ailang/verify_test.go` or the nearest existing end-to-end verifier test
- `docs/docs/guides/` verifier documentation selected by executor audit
- `changelogs/v0.32-current.md`

**Tasks:**

1. Before editing `.ail`, run `ailang prompt` and validate the example with
   `ailang check` as required by repository policy.
2. Add the Daneel `inWindow(now, target)` contract and require `ailang verify`
   to report VERIFIED rather than SKIPPED.
3. Add small contracts for `startOfDay` alignment/range and weekday range/epoch
   anchor, plus one intentionally opaque `addMonths` control.
4. Run real Z3 tests without silent skips and document the newly supported
   decidable fragment and its exclusions.

**Acceptance criteria:**

- [ ] `inWindow` using `addDays(now, 90)` is VERIFIED by real Z3.
- [ ] The example type-checks and runs, and its expected verify counts are asserted.
- [ ] `startOfDay` and `weekday` contracts verify for symbolic timestamps, including the negative domain.
- [ ] The `addMonths` control remains a named SKIP, proving scope did not widen.
- [ ] `go test ./internal/smt/... ./cmd/ailang/...` passes with Z3-dependent tests actually executed.
- [ ] `make fmt`, `make lint`, and `make test` pass; `make check-boundaries` passes if imports change.

## Day-by-Day Plan

### Day 1

- Run the semantic inventory and establish red-first parity/codegen tests (M1).
- Implement the dedicated datetime mapping and special encoder (M2).
- Finish with the focused SMT suite green and review the diff for accidental
  month/year or general callee support.

### Day 2

- Add and validate the runnable contract example and CLI-level verification tests (M3).
- Run real-Z3 controls, full tests, formatting, lint, and boundary checks.
- Update docs/changelog and record the zero-argument-callee follow-up disposition.

## Success Metrics

- Three public datetime functions enter the SMT decidable fragment with exact,
  runtime-parity formulas.
- Daneel's 90-day window changes from SKIPPED to VERIFIED with no raw-millisecond
  workaround.
- At least 12 focused unit/parity cases plus one end-to-end runnable example.
- No regression in existing SMT tests or contract-example outcomes.
- Unsupported calendar operations remain explicit named skips.
- Total change stays near 420 LOC; growth beyond 550 LOC triggers re-planning.

## Dependencies

- Z3 must be installed for the end-to-end acceptance run; absence is a reported
  blocker, not a test skip accepted as success.
- Existing SMT stdlib mapping and cross-module callee-resolution machinery.
- No runtime datetime, parser, evaluator, or effect-system change is expected.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Negative-time mismatch between SMT and Go/AIlANG arithmetic | Freeze epoch-boundary parity tests before implementation; use Euclidean day formulas only where runtime equality is demonstrated. |
| Mapping only codegen while fragment analysis still rejects calls | One shared resolver/spec consumed by `encodable.go`, callee collection, and `codegen_apps.go`; test all three seams. |
| Accidentally encoding all `_dt_add` calls linearly | Map the public `addDays` wrapper or require literal-zero year/month guards; retain named opaque controls. |
| Z3 tests silently skip | Assert tool availability in the acceptance lane and report missing Z3 explicitly. |
| General zero-argument bug expands scope | Preserve a reproduction and route it separately unless the narrow mapping resolves it automatically. |

## Deferred Work

- General zero-argument function/callee encoding.
- `addMonths`, `addYears`, `startOfMonth`, component extraction/construction,
  parsing, and formatting.
- `diffDays` until negative-domain truncation semantics are reconciled.
- Daneel's parser implementation itself; this sprint only provides its proof substrate.

## Approval and Handoff

This document and its JSON progress file are planning artifacts only. Per the
repository gate, implementation begins only after human approval and the explicit
instruction **“execute sprint”**, at which point it is handed to `sprint-executor`.

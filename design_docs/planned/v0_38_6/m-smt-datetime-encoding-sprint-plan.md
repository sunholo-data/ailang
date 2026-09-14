# Sprint Plan: M-SMT-DATETIME-ENCODING

**Source request:** Daneel message `inbox_1789394686844_585753b0`  
**Correlation ID:** `7a61d824-7da2-4a40-986f-248b1b7f2369`  
**Design doc:** Required before execution. This plan captures the requested semantics, but the
repository feature/semantics gate still requires an approved design document.  
**Sprint ID:** `M-SMT-DATETIME-ENCODING`  
**Base:** `58c15aac` (`coordinator/task-e680eb1d`)  
**Target version:** `v0.38.6`  
**Estimate:** 2 working days, approximately 330 LOC (110 implementation, 200 tests, 20 docs/example)  
**Risk:** Medium-high. An SMT encoding is a semantic claim: an inexact formula can produce false proofs.

## Summary

Make the exact, UTC-millisecond portions of `std/datetime` usable in contracts and proofs:
`addDays(ts, n)`, `startOfDay(ts)`, and `weekday(ts)`. Add direct stdlib-to-SMT mappings and
special encoders while keeping Go `AddDate` month/year behavior, constructors, parsing, and
formatting opaque.

The motivating acceptance case is Daneel's calendar authority rule: a target must not be in the
past or more than 90 days ahead. Today the literal `target <= now + 7776000000` verifies, while
the equivalent `target <= addDays(now, 90)` is skipped because `addDays` is unresolved.

## Current Status and Evidence

- `std/datetime.ail` defines `addDays` through `_dt_add(ts, 0, 0, days)`, `weekday` through
  `_dt_parts(ts).weekday`, and `startOfDay` through `_dt_parts` plus `_dt_make`.
- `internal/smt/types.go` maps only `std/string` and `std/list` wrappers in
  `ResolveStdlibToBuiltin`; datetime has no mapping.
- `internal/smt/codegen_apps.go` has special encoders for strings, lists, and numeric conversions,
  but none for datetime.
- `internal/smt/encodable.go` uses an allowlist. Unmapped datetime builtins are correctly rejected
  rather than leaked into invalid SMT-LIB.
- Runtime semantics are Unix milliseconds in UTC (`std/datetime.ail` and
  `internal/builtins/datetime.go`). `_dt_add` uses Go `time.AddDate`, so only the zero-year,
  zero-month day branch is linear and in scope.

### Velocity

The checkout has grafted history and exposes only one recent commit, so the skill's velocity script
cannot derive a defensible LOC/day figure. The plan therefore uses a conservative 165 LOC/day based
on the small number of encoder touchpoints, solver-backed test cost, and semantic review risk.

## Exact Semantic Contract

The encoder must produce these integer formulas:

| Public function | SMT meaning | Notes |
|---|---|---|
| `addDays(ts, n)` | `ts + n * 86400000` | Exact for UTC milliseconds and day-only addition |
| `startOfDay(ts)` | `ts - mod(ts, 86400000)` | Must be tested for timestamps before and after the Unix epoch |
| `weekday(ts)` | `mod(div(ts, 86400000) + 4, 7)` | 0=Sunday; Unix epoch was Thursday (4) |

The implementation must not advertise `_dt_add` generally as encodable: non-zero month/year
arguments use calendar-dependent Go `AddDate` semantics and are not represented by these formulas.
Prefer mappings for the three public `std/datetime` functions, or a guarded `_dt_add` encoder that
accepts only syntactic zero year/month arguments. The design review must choose and document one
approach before execution.

## Milestones

### M1 — Establish exact datetime mapping and fragment recognition

**Estimate:** 45 implementation LOC + 55 test LOC; 0.5 day  
**Dependencies:** Approved design document selecting public-wrapper mapping versus guarded builtin mapping

**Files to update:**

- `internal/smt/types.go`
- `internal/smt/types_test.go`
- `internal/smt/encodable.go` (only if a new special-map family is introduced)

**Tasks:**

- Add a narrowly scoped datetime mapping/spec for `addDays`, `startOfDay`, and `weekday`.
- Integrate it with `ResolveStdlibToBuiltin` and the fragment allowlist.
- Ensure `addMonths`, `addYears`, `_dt_make`, `_dt_parts`, parsing, formatting, and general
  `_dt_add` remain rejected.

**Acceptance criteria:**

- [ ] Each of the three public functions resolves to a known datetime encoder entry.
- [ ] `addMonths` and `addYears` remain unmapped and not SMT-encodable.
- [ ] A direct/unguarded `_dt_add(ts, years, months, days)` cannot pass the fragment gate.
- [ ] Table tests cover every accepted and deliberately rejected datetime name.

### M2 — Encode formulas and test generated SMT-LIB

**Estimate:** 65 implementation LOC + 95 test LOC; 0.75 day  
**Dependencies:** M1

**Files to update/create:**

- `internal/smt/codegen_apps.go`
- `internal/smt/datetime_test.go` (new)

**Tasks:**

- Encode direct and curried forms consistently with the existing special-builtin paths.
- Validate arity and return clear errors for malformed applications.
- Add exact SMT-LIB tests for symbolic arguments and negative/positive constants.
- Add solver-backed equivalence/boundary tests, including pre-epoch timestamps.

**Acceptance criteria:**

- [ ] `addDays(ts, n)` emits an expression equivalent to `(+ ts (* n 86400000))`.
- [ ] `startOfDay(ts)` emits an expression equivalent to `(- ts (mod ts 86400000))`.
- [ ] `weekday(ts)` emits an expression equivalent to `(mod (+ (div ts 86400000) 4) 7)`.
- [ ] Thursday at epoch maps to 4, Sunday maps to 0, and negative timestamps match runtime results.
- [ ] Wrong arities fail explicitly; no raw/uninterpreted datetime symbol reaches Z3.
- [ ] Curried and non-curried call shapes produce equivalent SMT.

### M3 — End-to-end verification regression and user-facing note

**Estimate:** 50 test LOC + 20 docs/example LOC; 0.75 day  
**Dependencies:** M2

**Files to create/update:**

- `internal/smt/testdata/datetime_contracts.ail` or the repository's existing verifier testdata equivalent
- Verifier integration test in the package that owns `ailang verify` fixtures
- `docs/docs/guides/contracts.mdx`

**Tasks:**

- Add the Daneel 90-day window as a fail-before/pass-after verification fixture.
- Add `startOfDay` and `weekday` proof fixtures with counterexample controls.
- Document the exact supported subset and explicitly state why month/year arithmetic is opaque.

**Acceptance criteria:**

- [ ] The reported `withDays` contract verifies instead of returning `SKIPPED`.
- [ ] Equivalent literal and `addDays` formulations have the same verification result.
- [ ] `startOfDay` and `weekday` fixtures verify for representative boundary properties.
- [ ] A deliberately false datetime property produces a counterexample, proving tests are non-vacuous.
- [ ] A contract using `addMonths` remains skipped with a clear unsupported-builtin reason.
- [ ] `go test ./internal/smt/...` and affected CLI verification tests pass.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.

## Day-by-Day Execution

### Day 1

1. Capture the current skip output for all three positive fixtures and the `addMonths` negative control.
2. Implement M1 mappings/allowlisting with table tests.
3. Implement M2 formulas and exact generated-SMT tests.
4. Run focused SMT tests and compare runtime versus solver results around epoch/day boundaries.

### Day 2

1. Add end-to-end verifier fixtures, including a false-property counterexample control.
2. Update contracts documentation with the supported datetime subset.
3. Run focused tests, then `make test`, `make lint`, and `make check-boundaries`.
4. Record any platform/tooling failures separately from semantic failures for evaluator handoff.

## Success Metrics

- Three exact public datetime operations become SMT-encodable.
- Daneel's 90-day calendar-window contract verifies without literal milliseconds.
- Negative timestamp behavior is proven consistent between Go runtime and Z3 formulas.
- Unsupported calendar-dependent operations remain rejected; there is no silent approximation.
- New encoder branches have direct, fragment-gate, solver, and CLI integration coverage.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Z3 `div`/`mod` and Go integer operations differ for negatives | Compare formulas against `time.UnixMilli(...).UTC()` on pre-epoch boundary cases; do not ship until equivalent |
| Mapping `_dt_add` accidentally blesses months/years | Map public `addDays`, or require literal-zero guards and negative tests for every non-day shape |
| Function bodies are resolved before the public mapping | Test imported `std/datetime` calls through the full verifier pipeline, not only encoder units |
| False proofs from a vacuous test | Include a deliberately false adjacent property that must produce a counterexample |
| Scope expands to all datetime builtins | Keep an explicit rejected-name table and document the exact subset |

## Dependencies and Execution Gate

- An approved design document is required because this changes proof semantics.
- After design approval, the user must approve this sprint plan and explicitly say `execute sprint`.
- The sprint executor must run the `sprint-executor` skill; completion must be assessed with
  `sprint-evaluator`.

## Explicitly Out of Scope

- `addMonths`, `addYears`, `startOfMonth`, `makeDate`, and general `_dt_add`/`_dt_make` encodings.
- Parsing and formatting functions.
- The separate zero-argument callee error (`unencodable type ()`) reported in the same message;
  it should receive its own diagnosis/design because it is not datetime-specific.
- Changes to runtime datetime behavior or the UTC-millisecond contract.


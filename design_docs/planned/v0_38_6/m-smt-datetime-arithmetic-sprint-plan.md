# Sprint Plan: M-SMT-DATETIME-ARITHMETIC

**Source request:** Daneel notification, correlation `8bdbe856-1dfa-43a6-b1a8-0cb9f943662c`  
**Design doc:** Not yet present — execution is blocked on `design-doc-creator` and explicit user approval under `AGENTS.md`  
**Sprint ID:** `M-SMT-DATETIME-ARITHMETIC`  
**Base:** `58c15aac` on `coordinator/task-46cf4201`  
**Planned:** 2026-09-14  
**Estimate:** 2 working days, approximately 420 LOC including tests and examples  
**Risk:** Medium-high — this extends the trusted SMT model of runtime datetime behavior

## Summary

Make the exact UTC-millisecond operations needed by Daneel proofs SMT-encodable: day addition,
start-of-day, and weekday. Keep month/year arithmetic and calendar construction opaque because Go
`time.AddDate` semantics are not linear. The sprint must prove that the SMT formulas agree with the
runtime implementation, not merely that Z3 accepts them.

The reported zero-argument callee problem is related but independently scoped. It is recorded as a
follow-up design question and is not silently bundled into this sprint.

## Current Status Analysis

- `std/datetime.ail` implements `addDays` through `_dt_add(ts, 0, 0, days)`, `weekday` through the
  record-returning `_dt_parts`, and `startOfDay` through `_dt_parts` plus `_dt_make`.
- `internal/smt/codegen_apps.go` has dedicated registries/encoders for strings, lists, and numeric
  conversions, but none for datetime builtins or datetime stdlib wrappers.
- `internal/smt/callee_resolver.go` deliberately skips stdlib calls only when
  `ResolveStdlibToBuiltin` recognizes them. Unmapped datetime wrappers enter cross-module callee
  resolution and are rejected when their underlying record/calendar builtins are opaque.
- Reproduction supplied by Daneel: the `addDays(now, 90)` contract is skipped, while the equivalent
  literal `now + 7776000000` verifies in 5 ms.
- Recent repository history contains only one commit in the seven-day window, so LOC/day is not a
  meaningful velocity measure. The two-day estimate is bottom-up and includes a 25% verification
  buffer.

## Design Gate and Required Decisions

Before execution, an approved design doc must settle:

1. Whether SMT support maps the public wrappers directly or introduces narrowly named special
   encoders for `_dt_add`, `_dt_parts`, and `_dt_make`. Direct wrapper formulas are preferred because
   encoding record-valued `_dt_parts` would claim substantially broader semantics than required.
2. Integer division/modulo semantics for negative Unix timestamps. Z3 `div`/`mod` are Euclidean,
   while Go integer division truncates toward zero. `startOfDay(ts) = ts - mod(ts, 86400000)` and
   weekday formulas are sound only if AILANG's runtime and SMT semantics agree over the supported
   timestamp domain, or if contracts require `ts >= 0`.
3. Overflow policy. AILANG `int` values are machine integers at runtime while SMT `Int` is unbounded;
   `ts + days*86400000` needs either an established no-overflow contract or a bounded encoding.
4. Exact weekday normalization for negative timestamps and the documented range `0..6`.

## Proposed Milestones

### M1 — Design-approved datetime SMT mapping

**Estimate:** 70 implementation LOC + 90 test LOC = 160 LOC, 0.6 day  
**Dependencies:** Approved design doc and explicit user approval

**Likely files:**

- `internal/smt/builtins.go` or a new `internal/smt/codegen_datetime.go`
- `internal/smt/codegen_apps.go`
- `internal/smt/codegen_datetime_test.go`

**Tasks:**

- Add a narrow datetime mapping/dispatch layer that works for direct and curried Core applications.
- Encode `addDays(ts,n)` as `ts + n*86400000` only under the approved integer/overflow model.
- Encode `startOfDay` and `weekday` with explicitly documented negative-timestamp semantics.
- Leave `addMonths`, `addYears`, `_dt_make`, and general `_dt_parts` unsupported unless the design
  proves a narrower encoder cannot serve the public wrappers.

**Acceptance criteria:**

- Unit tests assert exact SMT-LIB for all three operations, including nested expressions.
- Direct and curried application shapes produce equivalent formulas.
- Negative-timestamp boundary cases either match runtime semantics or fail closed with a documented
  rejection; they may not be silently modeled differently.
- Unsupported month/year operations still report a clear non-encodable reason.

### M2 — End-to-end verifier proofs and anti-unsoundness controls

**Estimate:** 40 implementation/fixup LOC + 150 test/fixture LOC = 190 LOC, 0.9 day  
**Dependencies:** M1

**Likely files:**

- `internal/smt/codegen_xmod_inline_test.go` or a focused datetime integration test
- `internal/smt/testdata/` fixtures
- `examples/runnable/contracts/datetime_arithmetic.ail`

**Tasks:**

- Add the reported 90-day window as an end-to-end `ailang verify` fixture.
- Add proofs for start-of-day idempotence/range and weekday range/known epochs.
- Add deliberately false contracts for every formula and assert Z3 returns counterexamples. These
  mutation-style controls prevent a disconnected or constant encoder from passing vacuously.
- Compare selected runtime evaluations against encoded expectations at epoch, day boundaries,
  leap-day adjacency, negative day offsets, and the approved negative-timestamp boundary.

**Acceptance criteria:**

- The original `withDays` reproduction reports `VERIFIED`, not `SKIPPED`.
- True contracts for all three operations verify; paired false contracts produce counterexamples.
- Runtime/SMT parity cases cover `n = -1, 0, 1, 90`, timestamps immediately around midnight, and
  known Sunday/Monday epochs.
- The example passes `ailang check`, runs successfully, and is exercised by the repository's example
  audit or an explicit integration test.

### M3 — Documentation and regression gates

**Estimate:** 20 implementation LOC + 50 documentation/test LOC = 70 LOC, 0.5 day  
**Dependencies:** M2

**Likely files:**

- `docs/docs/guides/verification.md` or the current verification reference
- `changelogs/v0.32-current.md`
- Existing SMT rejection/coverage tests

**Tasks:**

- Document the encodable datetime subset and explicitly list month/year arithmetic as opaque.
- Record timestamp-domain, division/modulo, and overflow assumptions next to the encoder and in user
  documentation.
- Run focused SMT tests, full Go tests, formatting, lint, and architecture-boundary checks.

**Acceptance criteria:**

- Documentation states exactly which `std/datetime` functions are proof-safe and their assumptions.
- Existing unsupported-builtin tests remain green and no opaque datetime function becomes accepted
  accidentally.
- `go test ./internal/smt/...`, `make test`, `make fmt`, `make lint`, and
  `make check-boundaries` pass, with any pre-existing failure separately baselined and reported.

## Day-by-Day Plan

### Day 1

- Obtain approval of the datetime SMT design decisions.
- Implement M1 with unit tests first.
- Begin runtime/SMT parity table and the original Daneel reproduction.

### Day 2

- Complete M2 proof fixtures and false-contract controls.
- Complete M3 documentation and all regression gates.
- Update sprint JSON `passes` values only after each milestone's acceptance criteria are evidenced.

## Success Metrics

- Three public datetime operations are SMT-encodable with runtime-parity evidence.
- One runnable contract example is added and verified.
- At least one false-contract control per operation demonstrates non-vacuity.
- No expansion to `addMonths`/`addYears` or generic calendar construction.
- Full relevant tests, lint, formatting, and boundary checks are green.

## Dependencies and Risks

- **Hard blocker:** approved design doc plus explicit user approval before implementation.
- **Semantic mismatch:** negative division/modulo can make apparently simple formulas unsound.
- **Overflow mismatch:** unbounded Z3 integers can prove properties that machine integers violate.
- **Over-broad mapping:** encoding `_dt_parts` wholesale increases the trusted surface unnecessarily.
- **Consumer fixture:** the Daneel repository is external; keep an in-repo minimal reproduction so CI
  does not depend on that checkout.

## Deferred Follow-up

Investigate zero-argument callees being reported as `unencodable type ()` in a separate systemic
design task. It affects `maxCreatesPerRun()` but is not required to encode datetime arithmetic and
should not enlarge this sprint without an audited design.

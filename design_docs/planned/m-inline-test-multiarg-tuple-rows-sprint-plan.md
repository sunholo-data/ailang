# Sprint Plan: Inline-Test Multi-Argument and Tuple-Valued Rows

## Summary

Implement the approved arity-aware inline-test row semantics from
[`m-inline-test-multiarg-tuple-rows.md`](m-inline-test-multiarg-tuple-rows.md): preserve the
deterministic `(input, expected)` grammar, make tuple-valued single arguments work, retain
nested multi-argument rows, and refuse malformed rows before evaluation with actionable
diagnostics.

**Sprint ID:** M-INLINE-TEST-ROW-ARITY  
**Issue:** #715  
**Duration:** 2.5 engineering days  
**Dependencies:** Approved design document; no code dependency  
**Risk Level:** Medium  
**Target baseline:** v0.42.0 (the design's historical v0.39.0 target is stale)

## Current Status Analysis

### Confirmed at HEAD

- `parseTestCase` has separate scalar and nested-input branches and emits a generic close-row
  error for a flat `(a, b, expected)` row.
- `collectInlineTests` reconstructs `(input, expected)` but does not preserve the declaration's
  parameter count in `testing.TestCase`.
- `buildFunctionCall` splats every tuple with more than one element, so it cannot distinguish
  one tuple-valued parameter from multiple parameters.
- Both `BuildInlineTestHarness` and `BuildClusterTestHarness` consume this representation, so
  the correction must cover both execution paths.
- Existing harness tests pin direct multi-argument `core.App` construction but do not cover the
  same row text against one- and two-parameter declarations.

### Velocity and Capacity

- The repository is at v0.42.0 with one release commit in the last seven days; the velocity
  script produced no reliable recent LOC/day figure.
- Estimate therefore uses the approved design's two-day forecast plus a 25% verification and
  recovery buffer: 2.5 days, approximately 610 changed LOC including tests and examples.
- The critical path is representation/validation first, then shared harness construction,
  followed by parser recovery and end-to-end regression coverage.

### Scope Boundaries

- No change to AILANG function calling convention, type inference, effects, or property tests.
- Flat rows remain invalid; this sprint improves their diagnostic and parser recovery.
- Downstream AILANG World dispatcher removal is outside this repository sprint.
- Existing scalar and correctly nested multi-argument rows retain their behavior.

## Proposed Milestones

### M1: Preserve Declared Arity and Validate Collected Rows

**Goal:** Make declaration arity explicit in every inline `TestCase` and reject row shapes that
cannot be constructed safely before they reach either evaluator path.

**Estimated:** 90 LOC implementation + 110 LOC tests = 200 LOC  
**Duration:** 0.75 day  
**Dependencies:** None

**Files to update:**

- `internal/testing/collector.go`
- `internal/testing/collector_test.go`
- Any focused diagnostic/error carrier in `internal/testing/` required by the approved design

**Tasks:**

1. Add declared arity (or an equivalent validated invocation representation) to inline
   `TestCase`; top-level named tests remain unaffected.
2. Validate nullary, unary, and N-ary input shapes during collection, including tuple element
   count and the one-element tuple edge case.
3. Return structured collection diagnostics instead of appending an executable case for an
   invalid shape; preserve source position and function name.
4. Add table-driven collector tests for scalar unary, tuple-valued unary, valid N-ary,
   mis-arity, nullary/unit, and invalid one-element tuple cases.

**Acceptance Criteria:**

- [ ] Every collected inline case carries enough declaration information to choose direct
      versus tuple-valued application without inspecting input shape alone.
- [ ] Invalid arity/shape rows do not enter `TestSuite.Tests` as executable cases.
- [ ] Mis-arity diagnostics name the function's declared arity, supplied count, and location.
- [ ] Existing top-level named tests and inline properties are unchanged.

**Risk:** The collector currently has no obvious error return channel. Mitigate by reusing the
repository's established diagnostics/result mechanism rather than adding a silent side list;
pin propagation with tests before changing harness code.

### M2: Build Calls by Arity in Both Harness Paths

**Goal:** Construct `core.App` deterministically from declared arity and remove panic-only
handling for malformed harness input.

**Estimated:** 100 LOC implementation + 100 LOC tests = 200 LOC  
**Duration:** 0.75 day  
**Dependencies:** M1

**Files to update:**

- `internal/testing/harness.go`
- `internal/testing/harness_test.go`
- `internal/testing/executor.go`

**Tasks:**

1. Change `buildFunctionCall` to use the collected arity: unary tuple inputs become one
   `core.Tuple` argument; N-ary rows become one `core.App` with N direct arguments.
2. Route `BuildInlineTestHarness` and `BuildClusterTestHarness` through the same checked call
   builder so they cannot drift.
3. Replace malformed-body panic paths with explicit construction errors and thread them to the
   executor boundary.
4. Ensure executor errors add exactly one harness context prefix.
5. Add the arity-disambiguation pair: identical `((1, 2), expected)` row payloads build one
   tuple argument for a unary declaration and two arguments for a binary declaration.

**Acceptance Criteria:**

- [ ] Unary tuple-valued input produces `App(f, [Tuple(a, b)])`.
- [ ] Binary/multi-argument input produces `App(f, [a, b, ...])`, never curried application.
- [ ] Inline and cluster harness builders have matching behavior and tests.
- [ ] Invalid internal shapes return an error rather than panic or reach evaluation.
- [ ] Runtime errors contain only one `harness evaluation failed:` context prefix.

**Risk:** Public helper signatures may have many tests/callers. Mitigate with a narrow shared
checked helper and compiler-guided updates, avoiding unrelated executor refactoring.

### M3: Targeted Flat-Row Diagnostic and Parser Recovery

**Goal:** Keep flat rows invalid while producing the approved nested-form hint without a
secondary `PAR_INFINITE_LOOP` cascade.

**Estimated:** 60 LOC implementation + 70 LOC tests = 130 LOC  
**Duration:** 0.5 day  
**Dependencies:** None; integrate after M1 so diagnostics are reviewed together

**Files to update:**

- `internal/parser/parser_testing.go`
- Focused parser tests under `internal/parser/`
- Diagnostic catalogue/documentation if required by the repository's diagnostic conventions

**Tasks:**

1. Detect a second top-level comma after parsing the expected expression and report that a row
   is exactly `(input, expected)`.
2. Include the hint `for multi-argument functions use ((a, b), expected)`.
3. Recover to the row terminator or tests-block boundary without triggering the progress guard.
4. Pin diagnostic code, position, message, hint, and absence of `PAR_INFINITE_LOOP`.

**Acceptance Criteria:**

- [ ] `tests [(1, 2, 3)]` is refused with the nested-form hint.
- [ ] The same malformed row does not emit `PAR_INFINITE_LOOP`.
- [ ] A following valid row is either recovered and parsed according to established parser
      policy or deterministically refused with no hang/cascade.
- [ ] Valid scalar and nested rows retain their AST shapes.

**Risk:** Parser token advancement is fragile because newlines are skipped. Mitigate with
multi-row recovery tests and assertions on following declarations, not only diagnostic text.

### M4: End-to-End Corpus, Examples, and JSON Output Verification

**Goal:** Prove the feature through the public `ailang test` path and bank discoverable examples
without regressing the existing inline-test corpus.

**Estimated:** 20 LOC implementation/docs + 60 LOC tests/examples = 80 LOC  
**Duration:** 0.5 day  
**Dependencies:** M1, M2, M3

**Files to update:**

- `internal/testing/executor_regression_test.go` or a focused inline-row integration test
- `examples/inline_tests_arithmetic.ail` (extend the existing canonical example)
- User-facing inline-test reference documentation only if it still describes ambiguous or flat
  syntax

**Tasks:**

1. Add end-to-end fixtures for direct tuple destructuring, nested-match tuple destructuring,
   three-argument rows, flat-row refusal, and wrong-arity refusal.
2. Verify JSON output reports malformed rows as diagnostics/collection failures rather than
   executed test failures and contains real source locations.
3. Run focused parser/testing packages, the public example, full tests, formatting, lint, and
   architecture boundaries as appropriate to the touched files.
4. Confirm existing `examples/inline_tests_*.ail` outcomes are unchanged.

**Acceptance Criteria:**

- [ ] Direct and nested-match unary tuple examples pass through `ailang test`.
- [ ] The existing three-argument example remains passing.
- [ ] Flat and wrong-arity cases never appear as runtime `no pattern matched` failures.
- [ ] JSON output has an actionable diagnostic and source location for refused rows.
- [ ] `make test`, `make fmt`, `make lint`, and `make check-boundaries` pass.

**Risk:** Full-suite failures may be unrelated on the v0.42.0 baseline. Mitigate by recording
focused green tests first, then distinguishing pre-existing failures with exact commands/output.

## Day-by-Day Plan

### Day 1

- Implement M1's arity representation and collection validation with table-driven tests.
- Start M2 with the shared checked call builder and the unary/binary ambiguity tests.

### Day 2

- Complete both harness paths and executor error propagation in M2.
- Implement M3's diagnostic and recovery tests.
- Add the core end-to-end cases from M4.

### Day 2.5

- Extend/verify examples and JSON output.
- Run focused and full verification, resolve regressions, and record any baseline-only failures.
- Update sprint JSON milestone status only as acceptance criteria pass.

## Success Metrics

- Four populated milestones totaling approximately 610 LOC, at least half of it tests/examples.
- The same nested row text is correctly disambiguated for unary tuple and multi-argument
  functions.
- Zero harness-shape panics and zero harness-caused runtime `no pattern matched` failures.
- Flat-row diagnostics provide the correct syntax and do not cascade.
- Existing inline-test examples and package tests preserve outcomes.
- One canonical example demonstrates tuple-valued unary and multi-argument rows.

## Dependencies and Open Questions

- The design is approved and has no implementation dependency.
- During M1, the executor must use the repository's existing diagnostic propagation route; if
  collection cannot currently return diagnostics, that plumbing is part of M1, not permission
  for a silent fallback.
- The executor should choose the exact new `TST` diagnostic code consistent with the catalogue;
  semantics and message requirements are fixed by the design.
- Execution remains gated on the user/coordinator explicitly saying `execute sprint`.

## Verification Commands

```bash
go test ./internal/parser ./internal/testing
go test ./internal/testing -run 'Inline|Harness|Collector'
ailang test examples/inline_tests_arithmetic.ail
make test
make fmt
make lint
make check-boundaries
```

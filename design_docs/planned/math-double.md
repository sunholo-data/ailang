# Math Double Function

**Status**: Planned
**Target**: v0.6.2
**Priority**: P2 (Low - testing fixture)
**Estimated**: 30 minutes
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function - fully deterministic |
| A2: Replayability | +1 | No side effects, fully reproducible |
| A3: Effect Legibility | 0 | No effects involved |
| A4: Explicit Authority | 0 | No authority concerns |
| A5: Bounded Verification | +1 | Simple type signature enables local verification |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Trivial builtin improves eval throughput |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost model implications |
| A10: Composability | +1 | Composes with arithmetic |
| A11: Structured Failure | 0 | No error handling needed |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [ ] A1 (Determinism): No implicit nondeterminism introduced
- [ ] A3 (Effects): No hidden side effects
- [ ] A4 (Authority): No ambient access granted
- [ ] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

This feature adds a simple `double` function to the standard library, enabling testing of the stage-aware coordinator pipeline during development and integration testing.

**Current State:**
- No simple math functions available for testing coordinator workflows
- Eval pipeline requires more complex functions for testing

**Impact:**
- Enables simple end-to-end testing of the coordinator
- Provides minimal test fixture for CI/CD validation

## Goals

**Primary Goal:** Implement a simple `double : int -> int` function for coordinator pipeline testing.

**Success Metrics:**
- Function implemented and returns correct result (e.g., `double(5) = 10`)
- All tests pass
- Function is available in AILANG code via imports

## Solution Design

### Overview

Add a builtin function `_int_double` that takes an integer and returns it multiplied by 2. This will be exposed to AILANG code through the standard library.

### Architecture

The function will be implemented as a builtin (internal Go function) registered in the builtin registry, following the M-DX1 pattern established for builtin function development.

**Components:**
1. **Builtin Registry Entry**: Register `_int_double` in `internal/builtins/spec.go`
2. **Implementation**: Add evaluation logic in builtin handler
3. **Test**: Add unit test to verify correct behavior

### Implementation Plan

**Phase 1: Register Builtin** (~10 minutes)
- [ ] Add `_int_double` entry to `internal/builtins/spec.go`
- [ ] Define type signature: `int -> int`

**Phase 2: Implement and Test** (~15 minutes)
- [ ] Implement evaluation logic
- [ ] Add unit test case
- [ ] Verify test passes

**Phase 3: Verification** (~5 minutes)
- [ ] Run full test suite
- [ ] Verify no regressions

### Files to Modify/Create

**New files:**
- None

**Modified files:**
- `internal/builtins/spec.go` - Add `_int_double` entry (~5 LOC)
- `internal/builtins/impl.go` (or equivalent) - Add implementation (~3 LOC)
- `internal/builtins/*_test.go` - Add unit test (~10 LOC)

## Examples

### Example 1: Basic Usage

**AILANG code:**
```ailang
module examples/double

let result = _int_double(5)
-- result = 10
```

### Example 2: Composition

**AILANG code:**
```ailang
let x = _int_double(3)
let y = _int_double(x)
-- y = 12
```

## Success Criteria

- [ ] `_int_double(5)` returns `10`
- [ ] `_int_double(0)` returns `0`
- [ ] `_int_double(-3)` returns `-6`
- [ ] All tests passing (`make test`)
- [ ] No regressions in existing tests
- [ ] Function accessible from AILANG code

## Testing Strategy

**Unit tests:**
- Test `_int_double(5) == 10`
- Test `_int_double(0) == 0`
- Test `_int_double(-3) == -6`
- Test edge cases (large numbers, min/max int)

**Integration tests:**
- Verify function is accessible from AILANG code
- Verify function composes with other operations

**Manual testing:**
- Test using REPL: `:type _int_double`
- Test in simple module

## Non-Goals

**Not in this feature:**
- Float support (only int for simplicity)
- Documentation website updates (internal testing only)
- Performance optimization (not required for simple builtin)

## Timeline

**Day 1** (30 minutes):
- Phase 1: Register builtin (~10 min)
- Phase 2: Implement and test (~15 min)
- Phase 3: Verification (~5 min)

**Total: ~30 minutes**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| None identified | — | Straightforward builtin addition with no dependencies |

## Related Documents

<!-- Auto-populated by semantic search on "math double" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_0_12/function_body_blocks.md](design_docs/implemented/v0_0_12/function_body_blocks.md) (1.00)
- [design_docs/implemented/v0_0_3/gpt5-reference-code.md](design_docs/implemented/v0_0_3/gpt5-reference-code.md) (0.95)
- [design_docs/implemented/v0_4_8/m-bug-record-update-inference.md](design_docs/implemented/v0_4_8/m-bug-record-update-inference.md) (0.90)

**Planned (check for overlap):**
- [design_docs/planned/v0_6_2/eval-dashboard-reliability.md](design_docs/planned/v0_6_2/eval-dashboard-reliability.md) (1.00)
- [design_docs/planned/v0_6_2/m-ui-refactor-ai-friendly.md](design_docs/planned/v0_6_2/m-ui-refactor-ai-friendly.md) (0.95)
- [design_docs/planned/v0_7_0/m-quasi-typed-quasiquotes.md](design_docs/planned/v0_7_0/m-quasi-typed-quasiquotes.md) (0.90)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [Link to related design docs]
- [Link to issues or discussions]

## Future Work

None at this time. This is a minimal test fixture.

---

**Document created**: 2026-01-01
**Last updated**: 2026-01-01

# Sprint Plan: M-CODEGEN-ADT-DOUBLE-PAREN

## Summary
Fix codegen bug where ADT constructors with fields generate invalid `NewConstructor()()` double-paren calls instead of `NewConstructor(args...)`.

**Duration:** 0.5 days (~3 hours)
**Dependencies:** None
**Risk Level:** Low
**GitHub Issue:** #52

## Current Status Analysis

### Completed Recently
- M-VERIFY Sprint 1: Contract infrastructure (~800 LOC) in 2 days
- M-DOC-SEM: Lazy embeddings + semantic search (~1,200 LOC) in 3 days
- M-BUG-LIST-LENGTH: Quick fix (~20 LOC) in 1 hour

### Velocity
- Recent average: ~300-400 LOC/day for features
- Bug fixes: typically 1-4 hours, 20-100 LOC
- Estimated capacity: ~100 LOC for this sprint

### Remaining from Design Doc
- ⏳ M1: Reproduce & Debug (~0 LOC, 1 hour)
- ⏳ M2: Implement Fix (~20 LOC, 1-2 hours)
- ⏳ M3: Tests & Cleanup (~50 LOC, 30 min)

## Proposed Milestones

### Milestone 1: M1-REPRODUCE - Reproduce & Debug
**Goal:** Create minimal repro and trace exact codegen path producing `()()`
**Estimated:** 0 LOC (debugging only)
**Duration:** 1 hour

**Tasks:**
- Create minimal AILANG test case with multi-arg ADT constructor
- Add DEBUG_CODEGEN logging to trace VarGlobal vs App handling
- Identify which code path emits first `()` and which emits second `()`
- Document root cause in design doc

**Acceptance Criteria:**
- [ ] Minimal repro test case created
- [ ] Root cause identified (which code path, which lookup fails)
- [ ] Can explain exactly why `NewDrawCmdViewport()()` is generated

**Risks:**
- Root cause may be deeper than expected - Mitigation: Time-box to 1h, escalate if needed

### Milestone 2: M2-FIX - Implement Fix
**Goal:** Fix codegen to emit correct constructor calls with all arguments
**Estimated:** 20 LOC implementation
**Duration:** 1-2 hours

**Tasks:**
- Fix constructor lookup in `codegen_expr_simple.go` or `codegen_expr_app.go`
- Ensure VarGlobal doesn't emit `()` when wrapped in App
- Verify fix against minimal repro test case
- Test with stapledons_voyage viewport.ail if available

**Acceptance Criteria:**
- [ ] Minimal repro generates correct Go code
- [ ] No `()()` patterns in generated code
- [ ] Existing codegen tests still pass

**Risks:**
- Fix may break other ADT codegen - Mitigation: Run full test suite

### Milestone 3: M3-TESTS - Tests & Cleanup
**Goal:** Add regression test and finalize fix
**Estimated:** 50 LOC tests
**Duration:** 30 minutes

**Tasks:**
- Add regression test in `codegen_adt_test.go` for multi-arg constructor
- Remove any debug logging added during investigation
- Update design doc with implementation report
- Run full test suite

**Acceptance Criteria:**
- [ ] Regression test added and passing
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Design doc updated with implementation report

**Risks:**
- None - straightforward cleanup

## Success Metrics
- Test coverage: No regression
- New tests: 1+ regression test for multi-arg ADT constructors
- Documentation: Design doc updated with implementation report
- All tests passing
- All linting clean
- GitHub Issue #52 ready to close (refs in commits, Fixes in final)

## Dependencies
- None - self-contained bug fix

## Open Questions
- None - root cause analysis will determine exact fix location

## Notes
- This is a P0 bug blocking stapledons_voyage
- Estimated completion: ~3 hours total
- Sprint includes investigation time since root cause is still a hypothesis

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fix ensures deterministic codegen |
| A2: Replayability | 0 | No replay impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Per-constructor regression test |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Correct Go output for machine consumption |
| A8: Minimal Syntax | 0 | Bug fix only |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Unblocks stapledons_voyage |
| A11: Structured Failure | +1 | Regression test prevents recurrence |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Codegen is deterministic
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Correct machine-readable output

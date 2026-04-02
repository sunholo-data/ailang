# Sprint Plan: M-DX-SERVE-API-COERCION

## Summary
Fix serve-api type coercion bug where `intToStr(x)` / `show(x)` returns raw IntValue instead of StringValue in cross-package diamond dependency configurations. Add robust debug infrastructure and regression tests.

**Duration:** 3 days
**Dependencies:** Docparse binary hash confirmation (Phase 1 blocker)
**Risk Level:** High (cannot reproduce locally — root cause unknown)
**Design Doc:** [m-dx-serve-api-coercion.md](m-dx-serve-api-coercion.md)

## Current Status Analysis

### Completed Recently
- DX error messages: show AILANG types with conversion hints (~60 LOC)
- `CallPreserveFloats → Call` fix: serve-api route handlers use `Call()` (~10 LOC)
- `DEBUG_EVAL_APP=1` tracing: function application + VarGlobal resolution (~30 LOC)
- Regression tests: 15 new tests for coercion and error messages (~170 LOC)

### Velocity
- Recent average: ~250 LOC/day (from last 7 days: ~1800 LOC across 7 days)
- Estimated capacity: ~750 LOC for this sprint

### Remaining from Design Doc
- M1: Reproduce the bug (~50 LOC tooling)
- M2: Root cause analysis (~100 LOC instrumentation)
- M3: Fix + integration tests (~300 LOC)
- M4: Debug CLI flag + cleanup (~100 LOC)

## Proposed Milestones

### Milestone 1: M1_REPRODUCE — Reproduce the Bug
**Goal:** Confirm reproduction conditions — binary mismatch, data-dependent, or race condition
**Estimated:** ~50 LOC tooling
**Duration:** 0.5 days

**Tasks:**
- Verify docparse binary hash matches local (msg pending)
- If binary matches: test with real Firestore data (not fallback freeEntitlements path)
- If binary differs: rebuild in docparse environment and retest
- Test under concurrent load (multiple simultaneous requests)
- Try with billing_service_api@0.5.3 (older version docparse originally reported on)
- Check if std/json `js()` function itself has a code path that passes through raw values

**Acceptance Criteria:**
- [ ] Binary hash comparison complete
- [ ] Bug reproduced locally OR confirmed as environment-specific
- [ ] Root cause hypothesis narrowed from 5 candidates to 1-2
- [ ] Reproduction conditions documented in design doc

**Risks:**
- Cannot reproduce at all — Mitigation: Add --debug-eval flag so docparse can collect traces in production

### Milestone 2: M2_ROOT_CAUSE — Identify Root Cause
**Goal:** Determine exactly why function application returns raw values in serve-api
**Estimated:** ~100 LOC instrumentation
**Duration:** 1 day

**Tasks:**
- Audit `evaluateModule()` shared state: does `rt.evaluator.SetGlobalResolver(resolver)` race across modules?
- Check `LetRec` phase 2.5 parent propagation: can two modules' bindings collide via `oldEnv.Set()`?
- Verify `Fork()` fresh env doesn't lose module-level bindings that closures depend on
- Compare Core AST for `intToStr(x)` between `ailang run` and serve-api compilation
- Add `--debug-eval` CLI flag to serve-api for persistent tracing
- If data-dependent: trace exact value flow when Firestore returns real int data vs fallback

**Acceptance Criteria:**
- [ ] Root cause identified and documented
- [ ] Failing code path traced from entry to concat_String error
- [ ] `--debug-eval` flag implemented for serve-api
- [ ] Design doc updated with root cause

**Risks:**
- Root cause is in compilation (elaborator) not evaluation — Mitigation: Compare Core AST output
- Race condition only under load — Mitigation: Test with wrk/ab for concurrent stress

### Milestone 3: M3_FIX — Apply Fix and Integration Tests
**Goal:** Fix the root cause and add regression tests that prevent recurrence
**Estimated:** ~300 LOC (fix ~100, tests ~200)
**Duration:** 1 day

**Tasks:**
- Implement fix based on M2 findings
- Write integration test: serve-api handler calling intToStr through diamond deps
- Write integration test: serve-api concurrent requests to same handler
- Test with billing_service_api@0.5.4 end-to-end
- Run full test suite (`make test`)
- Verify examples (`make verify-examples`)

**Acceptance Criteria:**
- [ ] billing_service_api@0.5.4 /billing/me/entitlements returns correct JSON
- [ ] No concat_String errors with diamond dependency patterns
- [ ] Integration test exercising serve-api + cross-package intToStr in CI
- [ ] All existing tests passing
- [ ] `make lint` clean

**Risks:**
- Fix breaks other serve-api functionality — Mitigation: Full test suite + manual testing

### Milestone 4: M4_CLEANUP — Debug Infrastructure and Docs
**Goal:** Consolidate debug tooling and finalize documentation
**Estimated:** ~100 LOC
**Duration:** 0.5 days

**Tasks:**
- Consolidate DEBUG_EVAL_APP behind --debug-eval CLI flag
- Remove any temporary debug logging
- Update design doc: move to implemented/ with implementation report
- Reply to docparse with fix details and verification steps
- Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] `--debug-eval` flag works for serve-api
- [ ] No leftover temporary debug code
- [ ] Design doc in implemented/ with root cause and fix details
- [ ] CHANGELOG updated
- [ ] Docparse notified

## Success Metrics
- Test coverage: no regression from current
- billing_service_api@0.5.4 works in serve-api without errors
- `--debug-eval` available for future serve-api troubleshooting
- All tests passing
- All linting clean

## Dependencies
- Docparse binary hash response (blocks M1 completion)
- Real Firestore credentials for data-dependent testing (available locally)

## Open Questions
- Is the bug binary-specific or data-dependent?
- Does it only manifest with specific Firestore data (non-fallback path)?
- Is there a race condition in module loading under concurrent requests?

## Notes
- The bug has been reported multiple times by docparse but cannot be reproduced locally
- DX improvements (error messages, CallPreserveFloats fix) are already committed (5e61986d)
- DEBUG_EVAL_APP=1 tracing is available for immediate use
- If M1 cannot reproduce: focus on --debug-eval flag so docparse can collect production traces

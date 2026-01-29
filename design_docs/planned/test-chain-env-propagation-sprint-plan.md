# Sprint Plan: Test Chain Environment Propagation

## Sprint Summary

**Sprint ID**: TEST-CHAIN-ENV
**Duration**: 1 day (4-6 hours)
**Type**: Testing Task
**Risk Level**: Low
**Status**: Planned

### Goal

Complete the test task to verify that `AILANG_CHAIN_ID` environment variable is properly propagated from the parent coordinator process through the agent execution chain.

### Key Deliverables

1. ✅ Verify environment variable propagation (`AILANG_CHAIN_ID`)
2. ✅ Document test results in design document
3. ✅ Create git commit documenting the test verification

---

## Current Status

**Design Document**: ✅ Approved
**Status**: Ready for implementation
**Dependency**: None (standalone test)

The design document confirms that the environment variable propagation mechanism works correctly:
- Parent process (coordinator) sets `AILANG_CHAIN_ID` in environment
- Child process (agent) inherits parent environment
- Value remains accessible throughout the execution chain

---

## Implementation Plan

### Milestone 1: Verify Environment Variable Propagation (2-3 hours)

**Objective**: Confirm that `AILANG_CHAIN_ID` is accessible in child processes

**Tasks**:

1. **Environment Check Script** (~50 LOC, 1 hour)
   - Create a test script that sets `AILANG_CHAIN_ID`
   - Run subprocess that echoes the variable back
   - Verify value propagation
   - Files: `scripts/test_chain_env.sh` (new)

2. **Coordinator Integration Test** (~80 LOC, 1 hour)
   - Test with actual coordinator task execution
   - Verify env vars passed to agents
   - Check with different agent types (design-doc-creator, sprint-planner, etc.)
   - Files: Update `scripts/test_chains_integration.sh` (add env var test level)

3. **Test Execution** (~30 min)
   - Run both test scripts
   - Document results
   - Capture actual `AILANG_CHAIN_ID` value in output

**Acceptance Criteria**:
- ✅ Simple shell script demonstrates env var propagation
- ✅ Coordinator task receives env var from parent
- ✅ Value matches expected format (UUID string)
- ✅ Env var accessible in multiple subprocess types
- ✅ All tests pass without errors

**Estimated LOC**: ~150 (scripts) + ~50 (docs updates)

---

### Milestone 2: Document Results and Git Commit (1-2 hours)

**Objective**: Document test results and commit verification to git

**Tasks**:

1. **Update Design Document** (~30 min)
   - Finalize test results section
   - Add test execution timestamp
   - Include example env var value
   - Files: `design_docs/planned/test-chain-env-propagation.md` (update)

2. **Create Git Commit** (~30 min)
   - Add test scripts to git
   - Create commit with proper message
   - Reference design document in commit
   - Files: Create `scripts/test_chain_env.sh` (new)

**Acceptance Criteria**:
- ✅ Design document finalized with test results
- ✅ Git commit created with proper formatting
- ✅ Commit message references task ID: `task-fb0ff1fa`
- ✅ All test artifacts included in repo

**Estimated LOC**: ~50 (docs)

---

## Task Breakdown by Day

### Day 1 (4-6 hours)

**Morning (2-3 hours)**:
1. Create test script: `scripts/test_chain_env.sh`
2. Implement basic environment check and propagation verification
3. Run and verify results

**Afternoon (2-3 hours)**:
1. Update `test_chains_integration.sh` with env var test level
2. Execute coordinator-based test
3. Document results in design document
4. Create git commit

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Environment variable propagation | 100% successful | To verify |
| Test scripts created | 2 scripts | Pending |
| Design document updated | Complete | Pending |
| Git commit created | 1 commit | Pending |
| Test coverage | All happy paths | Pending |

---

## Technical Notes

### Environment Variable Format

Expected format: UUID (e.g., `b027d302-b5c2-43ab-8737-6b74ddc66214`)

### Test Approach

1. **Simple propagation test**:
   ```bash
   export AILANG_CHAIN_ID="test-value"
   bash -c 'echo $AILANG_CHAIN_ID'
   ```

2. **Coordinator-based test**:
   - Set env var in coordinator parent process
   - Agent process should receive and access it
   - Use `echo $AILANG_CHAIN_ID` in subprocess

3. **Verification**:
   - Match output value with parent value
   - Check agent logs for correct value
   - Verify no corruption or truncation

### Files to Create/Modify

**New Files**:
- `scripts/test_chain_env.sh` - Simple propagation test (~50 LOC)
- Optional: Helper functions in test utils if reusable

**Modified Files**:
- `scripts/test_chains_integration.sh` - Add env var test level (~80 LOC)
- `design_docs/planned/test-chain-env-propagation.md` - Finalize results

---

## Risks and Mitigations

| Risk | Probability | Severity | Mitigation |
|------|-------------|----------|-----------|
| Env vars not propagating | Low | Medium | Test with simple bash script first |
| Coordinator not running | Medium | Low | Use `ailang coordinator status` check |
| Format mismatch | Low | Low | Document expected format upfront |

---

## Dependencies

None - this is a standalone verification test.

---

## Open Questions

1. Should we test with all agent types or just one representative agent?
   - **Answer**: Test with design-doc-creator (most common) + one other agent type for verification

2. Should we add persistent tests to the test suite?
   - **Answer**: Add to `test_chains_integration.sh` as reusable test level

3. Acceptable propagation latency?
   - **Answer**: Should be immediate (no async issues expected)

---

## Approval Gate

**Ready for implementation**: ✅ Yes

**Next step**: Execute test tasks, commit results, hand off to sprint-executor for review

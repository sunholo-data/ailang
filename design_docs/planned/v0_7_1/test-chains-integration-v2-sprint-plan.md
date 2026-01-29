# Test Chains Integration v2 - Sprint Plan

**Sprint ID:** M-TEST-CHAINS-V2
**Duration:** 2 days (16 hours estimated)
**Target Version:** v0.7.1
**Risk Level:** Medium
**Coordinator Issue:** task-09ec6d01
**Status:** Ready for Implementation

## Sprint Summary

Implement a comprehensive test chains framework that validates multi-stage agent workflows (design-doc-creator → sprint-planner → sprint-executor) with deterministic, reproducible results. This framework will integrate into the existing test infrastructure and provide mock AI response support for fast, reliable CI testing.

**Key Deliverables:**
- Core chain execution framework (`internal/testing/chains/`)
- Mock response infrastructure for AI executors
- Stage validators and artifact tracking
- At least 3 critical workflow test chains
- CI integration with `make test-chains` target
- Documentation and examples

**Success Metrics:**
- All test chains execute successfully in < 60s total
- 100% deterministic (same seed = same results)
- Artifact validation catches breaking changes
- CI integration working with automated regression testing
- Documentation includes creation guide with examples

## Current Status Analysis

### Completed Work
- ✅ Design document created (design doc structure finalized)
- ✅ Architecture decisions documented (mock at executor level, YAML format)
- ✅ Interface requirements identified (Executor interface, mock provider)
- ✅ Test examples and mock format defined

### Remaining Work
- ❌ Core framework implementation (chain.go, validator.go)
- ❌ Mock infrastructure (mock_provider.go)
- ❌ Coordinator integration and test mode support
- ❌ Test chain definitions (YAML files)
- ❌ Documentation and CI integration

### Velocity Context
Recent commits show ~2-3 days for similar framework implementations:
- M-DX25 (budget system): 2-3 days
- Testing enhancements (inline tests): 1-2 days
- Coordinator integration features: 2-3 days

**Estimated Velocity:** 150-200 LOC/day (framework code with heavy testing)

## Milestone Breakdown

### Milestone 1: Core Framework Foundation (Day 1, ~6 hours)

**Goal:** Establish the chain execution infrastructure and basic validators

**Tasks:**
1. Create `internal/testing/chains/` package structure
   - `chain.go` - Chain definition and executor (~300 LOC)
   - `validator.go` - Stage validator interfaces (~200 LOC)
   - `artifacts.go` - Artifact capture and tracking (~150 LOC)

2. Implement ChainDefinition struct
   - Load YAML chain definitions
   - Parse stage configurations
   - Resolve dependencies between stages

3. Implement ChainExecutor
   - Sequential stage execution
   - Dependency management
   - Error handling with clear diagnostics
   - Execution timing and metrics

4. Implement basic validators
   - FileExists validator
   - ContentContains validator
   - OutputMarker validator
   - Custom validation interface for extensibility

5. Implement artifact system
   - Extract artifacts from stage outputs
   - Support regex-based pattern matching
   - Provide artifact substitution in downstream stages

**Files to Create:**
- `internal/testing/chains/chain.go` (~300 LOC)
- `internal/testing/chains/validator.go` (~200 LOC)
- `internal/testing/chains/artifacts.go` (~150 LOC)
- `internal/testing/chains/types.go` (~80 LOC) - Type definitions

**Files to Modify:**
- `internal/testing/executor.go` (+40 LOC) - Add chain execution hook
- `Makefile` (+15 LOC) - Add test-chains target

**Test Coverage:**
- Chain parsing and validation (10 tests)
- Artifact extraction with patterns (8 tests)
- Dependency resolution (6 tests)
- Error handling (5 tests)

**Estimated LOC:** 730 new + 55 modified = ~785 total
**Estimated Hours:** 5-6 hours
**Tests Expected:** 29 new tests (~40 LOC)

**Acceptance Criteria:**
- [ ] `internal/testing/chains/` package compiles with 0 errors
- [ ] Chain definition parsing works with YAML files
- [ ] Stage validators execute and report results
- [ ] Artifact extraction with regex patterns works
- [ ] All 29 tests pass (unit tests in `*_test.go`)
- [ ] Code compiles with `make test`

**Example Test Chain Created:**
```yaml
# tests/chains/hello_world_test.yaml
name: hello_world
stages:
  - id: hello_stage
    validators:
      - type: output_marker
        marker: "COMPLETE"
```

---

### Milestone 2: Mock Infrastructure (Day 1, ~4 hours)

**Goal:** Implement deterministic AI response mocking for executor-level injection

**Tasks:**
1. Create mock response provider
   - `internal/testing/chains/mock_provider.go` (~250 LOC)
   - Implement `executor.Executor` interface
   - Response file format (JSON) with turn-based structure

2. Implement mock response playback
   - Load JSON response files
   - Map tool calls to mocked responses
   - Handle partial execution (fewer turns than recorded)
   - Deterministic response ordering

3. Integrate with executor interfaces
   - Claude executor mock injection (+30 LOC)
   - Gemini executor mock injection (+30 LOC)
   - Bypass real AI calls in test mode

4. Support response recording mode (for future golden file creation)
   - Flag to record actual responses
   - Save to JSON files for regression testing
   - Versioning for response format

**Files to Create:**
- `internal/testing/chains/mock_provider.go` (~250 LOC)
- `internal/testing/chains/mock_loader.go` (~120 LOC) - JSON loading

**Files to Modify:**
- `internal/executor/claude/claude.go` (+30 LOC)
- `internal/executor/gemini/gemini.go` (+30 LOC)
- `internal/executor/executor.go` (+20 LOC) - Mock mode flag

**Example Mock Response File:**
```json
{
  "version": "1.0",
  "responses": [
    {
      "turn": 1,
      "type": "text",
      "content": "I'll create the design doc..."
    },
    {
      "turn": 1,
      "type": "tool_use",
      "tool": "Write",
      "parameters": {
        "file_path": "design_docs/planned/v0_7_1/test-feature.md",
        "content": "# Test Feature Design\n\n..."
      }
    }
  ]
}
```

**Test Coverage:**
- JSON response parsing (5 tests)
- Mock response playback (7 tests)
- Tool call matching (6 tests)
- Error scenarios (4 tests)

**Estimated LOC:** 400 new + 80 modified = ~480 total
**Estimated Hours:** 3-4 hours
**Tests Expected:** 22 new tests (~30 LOC)

**Acceptance Criteria:**
- [ ] Mock provider implements `executor.Executor` interface
- [ ] Loads and parses JSON mock response files
- [ ] Tool calls matched to mock responses deterministically
- [ ] All 22 tests pass
- [ ] No network calls made when in mock mode
- [ ] `internal/executor/` tests compile and pass

---

### Milestone 3: Coordinator Integration (Day 2, ~4 hours)

**Goal:** Hook test chains into the coordinator pipeline and create stage validators

**Tasks:**
1. Add test mode support to coordinator
   - `internal/coordinator/test_mode.go` (~100 LOC) - Test context
   - Mock provider injection in coordinator (`daemon_tasks.go` +30 LOC)
   - Flag to disable real AI calls during testing

2. Create stage-specific validators
   - `DesignDocValidator` - Verifies design doc structure
   - `SprintPlanValidator` - Validates milestone structure and LOC estimates
   - `VelocityValidator` - Checks velocity calculations
   - Custom matcher patterns for output

3. Implement artifact flow between stages
   - Extract design doc path from design-doc-creator output
   - Pass to sprint-planner as input parameter
   - Validate artifact presence in downstream stages

4. Integration tests for multi-stage workflows
   - Test design → sprint handoff
   - Test sprint → executor handoff
   - Verify artifact propagation
   - Check error handling for missing artifacts

**Files to Create:**
- `internal/coordinator/test_mode.go` (~100 LOC)
- `internal/testing/chains/validators_design.go` (~150 LOC)
- `internal/testing/chains/validators_sprint.go` (~120 LOC)

**Files to Modify:**
- `internal/coordinator/daemon.go` (+30 LOC)
- `internal/coordinator/daemon_tasks.go` (+30 LOC)
- `internal/executor/executor.go` (+20 LOC)

**Test Coverage:**
- Coordinator test mode flag (4 tests)
- Design doc validation (8 tests)
- Sprint plan validation (8 tests)
- Artifact propagation (6 tests)
- Multi-stage integration (5 tests)

**Estimated LOC:** 370 new + 80 modified = ~450 total
**Estimated Hours:** 3-4 hours
**Tests Expected:** 31 new tests (~45 LOC)

**Acceptance Criteria:**
- [ ] Coordinator has `--test-mode` flag support
- [ ] All executor calls check for mock provider in test mode
- [ ] Design doc validator correctly identifies required sections
- [ ] Sprint plan validator calculates velocity correctly
- [ ] Artifacts flow correctly from stage to stage
- [ ] All 31 integration tests pass
- [ ] No real AI API calls made with `--test-mode` flag

---

### Milestone 4: Test Chains & Documentation (Day 2, ~2 hours)

**Goal:** Create critical workflow test chains and comprehensive documentation

**Tasks:**
1. Create test chain definitions (YAML files)
   - `tests/chains/design-to-sprint.yaml` - Primary workflow
   - `tests/chains/sprint-to-executor.yaml` - Implementation stage
   - `tests/chains/hello-world.yaml` - Simple smoke test
   - Validators and artifact specifications

2. Create mock response files
   - `tests/chains/mocks/design_doc_response.json`
   - `tests/chains/mocks/sprint_plan_response.json`
   - `tests/chains/mocks/executor_response.json`
   - Realistic but minimal responses

3. Create test runner and CI integration
   - `cmd/ailang/test_chains.go` (~150 LOC) - CLI command
   - `Makefile` target: `test-chains`
   - `.github/workflows/ci.yml` update (+10 LOC)

4. Create documentation
   - `docs/testing/test-chains.md` (~200 lines)
   - Usage examples (CLI commands)
   - Chain creation guide
   - Mock format reference
   - Troubleshooting guide

**Files to Create:**
- `tests/chains/design-to-sprint.yaml` (~80 lines)
- `tests/chains/sprint-to-executor.yaml` (~70 lines)
- `tests/chains/hello-world.yaml` (~40 lines)
- `tests/chains/mocks/design_doc_response.json` (~100 lines)
- `tests/chains/mocks/sprint_plan_response.json` (~120 lines)
- `tests/chains/mocks/executor_response.json` (~100 lines)
- `cmd/ailang/test_chains.go` (~150 LOC)
- `docs/testing/test-chains.md` (~200 lines)

**Files to Modify:**
- `Makefile` (+20 LOC)
- `.github/workflows/ci.yml` (+10 LOC)
- `cmd/ailang/main.go` (+5 LOC) - Register test-chains command

**Test Coverage:**
- Test chain validation (5 tests)
- Mock file loading (3 tests)
- End-to-end chain execution (3 tests)

**Estimated LOC:** 250 new (code) + 500 new (configs/docs) + 35 modified = ~785 total
**Estimated Hours:** 2-3 hours
**Tests Expected:** 11 new tests (~15 LOC)

**Acceptance Criteria:**
- [ ] `ailang test-chains --help` shows all options
- [ ] `make test-chains` runs all 3 test chains successfully
- [ ] Each test chain completes in < 20s
- [ ] All test chains pass with < 10% variance
- [ ] Documentation is complete with examples
- [ ] CI integration works without errors
- [ ] All 11 tests pass

---

## Day-by-Day Implementation Plan

### Day 1 (8 hours)

**Slot 1: Milestones 1 + Initial 2 (4 hours)**
- 08:00-09:00 (1h) - Set up `internal/testing/chains/` package, create types.go
- 09:00-10:30 (1.5h) - Implement chain.go (ChainDefinition, ChainExecutor)
- 10:30-12:00 (1.5h) - Implement validator.go and artifacts.go
- 12:00-13:00 (1h) - Write unit tests for chain execution and validation

**Slot 2: Milestone 2 (4 hours)**
- 13:00-14:30 (1.5h) - Implement mock_provider.go
- 14:30-15:30 (1h) - Implement mock_loader.go and JSON parsing
- 15:30-16:30 (1h) - Integrate mocks with executor interfaces (claude.go, gemini.go)
- 16:30-17:00 (0.5h) - Write mock infrastructure tests

**End of Day 1 Checkpoint:**
- ✅ Core framework compiling and passing 50+ unit tests
- ✅ Mock provider loading JSON and playing back responses
- ✅ Basic chain execution working with FileExists validator

---

### Day 2 (8 hours)

**Slot 1: Milestone 3 (4 hours)**
- 08:00-09:00 (1h) - Add test mode support to coordinator
- 09:00-10:30 (1.5h) - Implement design doc and sprint plan validators
- 10:30-12:00 (1.5h) - Integrate with coordinator daemon, artifact propagation
- 12:00-13:00 (1h) - Write integration tests, verify multi-stage flow

**Slot 2: Milestone 4 (4 hours)**
- 13:00-14:00 (1h) - Create 3 test chain YAML definitions
- 14:00-15:00 (1h) - Create mock response JSON files
- 15:00-16:00 (1h) - Implement test-chains CLI command, Makefile target
- 16:00-17:00 (1h) - Create documentation and examples

**End of Day 2 Checkpoint:**
- ✅ All test chains executing successfully
- ✅ 85+ unit and integration tests passing
- ✅ Documentation complete and reviewed
- ✅ CI integration working

---

## Test Cases Breakdown

### Milestone 1 Tests (~40 LOC, 29 tests)
```
chain_test.go:
  - TestChainDefinitionParsing (3 tests)
  - TestChainDependencyResolution (3 tests)
  - TestChainExecutionSequence (2 tests)
  - TestChainErrorHandling (4 tests)

validator_test.go:
  - TestFileExistsValidator (2 tests)
  - TestContentContainsValidator (4 tests)
  - TestOutputMarkerValidator (4 tests)

artifacts_test.go:
  - TestArtifactExtraction (5 tests)
  - TestRegexPatternMatching (3 tests)
```

### Milestone 2 Tests (~30 LOC, 22 tests)
```
mock_provider_test.go:
  - TestJSONResponseParsing (3 tests)
  - TestMockResponsePlayback (4 tests)
  - TestToolCallMatching (3 tests)
  - TestDeterministicBehavior (2 tests)

mock_loader_test.go:
  - TestLoadMockFile (2 tests)
  - TestLoadMissingFile (2 tests)
  - TestResponseVersioning (2 tests)
  - TestErrorHandling (2 tests)
```

### Milestone 3 Tests (~45 LOC, 31 tests)
```
coordinator_test.go:
  - TestTestModeFlag (2 tests)
  - TestMockInjection (2 tests)

validators_design_test.go:
  - TestDesignDocStructure (4 tests)
  - TestSectionValidation (4 tests)

validators_sprint_test.go:
  - TestMilestoneValidation (3 tests)
  - TestVelocityCalculation (4 tests)
  - TestEstimateValidation (1 test)

integration_test.go:
  - TestDesignToSprintFlow (2 tests)
  - TestSprintToExecutorFlow (2 tests)
  - TestArtifactPropagation (2 tests)
```

### Milestone 4 Tests (~15 LOC, 11 tests)
```
chain_definitions_test.go:
  - TestChainDefinitionValidation (3 tests)
  - TestMockFileLoading (2 tests)

e2e_test.go:
  - TestHelloWorldChain (1 test)
  - TestDesignSprintChain (1 test)
  - TestFullPipeline (1 test)
  - TestChainPerformance (1 test)
  - TestErrorRecovery (2 tests)
```

**Total Test LOC:** ~130 LOC across 10+ test files
**Total Tests:** 93 tests (covering all milestones)
**Coverage Target:** 85%+ (essential paths fully tested)

---

## Success Metrics

### Functional Requirements
- ✅ All 4 milestones implemented
- ✅ 93+ tests passing (100% pass rate)
- ✅ Zero compiler errors and warnings
- ✅ Zero network calls in test mode

### Performance Requirements
- ✅ Test chains execute in < 60s total (3 chains × 20s each)
- ✅ Individual chain execution time stable (< 10% variance)
- ✅ Memory usage < 100MB for chain execution
- ✅ No resource leaks or goroutine leaks

### Quality Requirements
- ✅ Test coverage ≥ 85% for `internal/testing/chains/`
- ✅ All validators have documentation
- ✅ Mock responses realistic and minimal
- ✅ Error messages clear and actionable

### Documentation Requirements
- ✅ `docs/testing/test-chains.md` complete (200+ lines)
- ✅ Examples show both CLI and programmatic usage
- ✅ Chain creation guide with step-by-step instructions
- ✅ Troubleshooting section with common issues

### Integration Requirements
- ✅ `make test-chains` runs all tests successfully
- ✅ CI/CD pipeline includes test chains in critical path
- ✅ GitHub Actions workflow updated
- ✅ No breaking changes to existing tests

---

## Risks and Mitigations

### Risk 1: Mock Determinism (Medium)
**Risk:** Randomness in mock provider breaks deterministic execution
**Mitigation:**
- Use seeded RNG for any random values
- Write tests validating deterministic behavior
- Log all mock decision points

### Risk 2: Executor Interface Coupling (Medium)
**Risk:** Changes to executor interface during implementation break multiple files
**Mitigation:**
- Define executor interface early (Milestone 1, Task 2)
- Use mocks in tests to isolate executor calls
- Maintain backward compatibility shim

### Risk 3: Artifact Extraction Complexity (Low)
**Risk:** Regex patterns in artifacts are fragile across different AI responses
**Mitigation:**
- Start with simple patterns (exact match)
- Extensive testing of pattern matching
- Clear error messages when artifacts not found

### Risk 4: CI Integration (Low)
**Risk:** GitHub Actions workflow breaks existing tests
**Mitigation:**
- Minimal changes to CI workflow (just add new target)
- Test locally with `act` before pushing
- Run existing tests in CI before test chains

### Risk 5: Performance (Low)
**Risk:** Mock response loading becomes slow with many chains
**Mitigation:**
- Cache parsed mock files
- Lazy-load responses only when needed
- Monitor performance in Milestone 4

---

## Dependency Graph

```
Milestone 1 (Core Framework)
    ↓ (requires ChainDefinition)
Milestone 2 (Mock Infrastructure)
    ↓ (requires Executor integration)
Milestone 3 (Coordinator Integration)
    ↓ (requires validators and artifact system)
Milestone 4 (Test Chains & Documentation)
```

**Critical Path:** All milestones sequential (must complete in order)
**Estimated Total Duration:** 15-16 hours (accounting for testing and debugging)

---

## Example Files to Create/Verify

**Test chains (working examples):**
- `tests/chains/hello-world.yaml` - Smoke test (required)
- `tests/chains/design-to-sprint.yaml` - Design workflow (required)
- `tests/chains/sprint-to-executor.yaml` - Implementation workflow (required)

**Mock response files:**
- `tests/chains/mocks/design_doc_response.json`
- `tests/chains/mocks/sprint_plan_response.json`
- `tests/chains/mocks/executor_response.json`

**Documentation:**
- `docs/testing/test-chains.md` - Primary documentation (required)
- `design_docs/planned/v0_7_1/test-chains-integration-v2.md` - Design doc (reference)

---

## Open Questions & Decisions

1. **Recording Mode Priority:** Should response recording be in initial release or deferred?
   - Decision: Deferred (nice-to-have for v0.7.2)

2. **Chain Composition:** Should chains be composable (run chain-in-chain)?
   - Decision: Not in v0.7.1 (deferred to future)

3. **Parallel Execution:** Should stages support parallel execution?
   - Decision: Sequential-only in v0.7.1 (deferred)

4. **Flaky Test Tolerance:** How should we handle timing-dependent failures?
   - Decision: Strict mode first, relaxed mode deferred

---

## Related Documentation

- [Test Chains Design Doc](test-chains-integration-v2.md) - Original design
- [Coordinator Guide](../../docs/guides/coordinator.md) - Coordinator architecture
- [Testing Guide](../../docs/CONTRIBUTING.md#testing) - Testing conventions
- [Executor Interfaces](../../internal/executor/README.md) - Executor API

---

## Implementation Notes

### Code Style
- Follow existing Go conventions (gofmt, golangci-lint)
- Use interfaces for extensibility (Validator, ResponseProvider)
- Prefer explicit over implicit (clear function signatures)

### Testing Philosophy
- Write tests alongside implementation (TDD)
- Unit tests for isolated components
- Integration tests for multi-stage workflows
- End-to-end tests for full pipeline

### Error Handling
- Return errors with context (use fmt.Errorf with %w)
- Log all chain execution events (debug level)
- Provide actionable error messages to users

---

## Sign-Off Checklist

- [ ] All code changes compile without errors/warnings
- [ ] All 93+ tests pass locally and in CI
- [ ] Test coverage ≥ 85% for new code
- [ ] Documentation is complete and reviewed
- [ ] Examples work and are verified
- [ ] No breaking changes to existing code
- [ ] Performance benchmarks met (< 60s for all chains)
- [ ] Coordinator integration verified with test mode
- [ ] GitHub Actions CI workflow updated and passing
- [ ] CHANGELOG.md updated with feature summary

---

**Next Steps:**
1. Review sprint plan with team
2. Get approval for architecture decisions
3. Start Milestone 1 implementation
4. Create GitHub issue for tracking (if needed)
5. Set up daily standup for progress updates

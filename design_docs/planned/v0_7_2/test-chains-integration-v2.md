# Test Chains Integration v2

**Status:** Planned
**Target:** v0.7.1
**Priority:** P1 (Medium)
**Estimated:** 2 days
**Dependencies:** None
**Created:** 2026-01-29
**Last Modified:** 2026-01-29

## Problem Statement

The AILANG testing infrastructure currently lacks integrated test chain support, which is critical for verifying complex multi-stage agent workflows. Test chains allow validating that a sequence of operations (e.g., design → sprint → implementation) produces the expected outcomes at each stage while maintaining proper state transitions.

**Current issues:**
- No integrated framework for testing agent handoffs
- Manual verification of multi-stage workflows is time-consuming
- Difficult to regression test coordinator-based pipelines
- No automated validation of artifact propagation between stages

## Goals

**Primary Goal:** Establish a comprehensive test chains framework that validates multi-stage agent workflows with deterministic, reproducible results.

### Success Metrics
- [ ] Test chains framework integrated into existing test infrastructure
- [ ] Support for coordinator workflow testing (design-doc → sprint-planner → executor)
- [ ] Deterministic test execution with mocked AI responses
- [ ] Artifact validation between stages (e.g., design doc → sprint plan)
- [ ] CI integration with automated regression testing

## Solution Design

### Architecture

The test chains framework will provide:

1. **Chain Definition DSL** - Declarative test chain specifications
2. **Stage Validators** - Per-stage assertion framework
3. **Mock Infrastructure** - Deterministic AI response mocking
4. **Artifact Tracking** - Validate data flow between stages
5. **CI Integration** - Automated regression testing

### Implementation Plan

**Phase 1: Core Framework** (~4 hours)
- [ ] Design test chain specification format (JSON/YAML)
- [ ] Create `internal/testing/chains/` package
- [ ] Implement chain executor with stage orchestration
- [ ] Add basic stage validation interface

**Phase 2: Mock Infrastructure** (~3 hours)
- [ ] Create mock response provider for AI executors
- [ ] Implement deterministic response generation
- [ ] Add response recording for golden file creation
- [ ] Support for both Claude and Gemini executors

**Phase 3: Coordinator Integration** (~4 hours)
- [ ] Hook into coordinator task execution pipeline
- [ ] Add test mode flag to bypass real AI calls
- [ ] Implement artifact capture between stages
- [ ] Create validation helpers for common patterns

**Phase 4: Test Suite & Documentation** (~3 hours)
- [ ] Write test chains for critical workflows
- [ ] Create golden files for regression testing
- [ ] Document chain creation process
- [ ] Add CI integration with make targets

### Files to Modify/Create

**New files:**
- `internal/testing/chains/chain.go` (~300 LOC) - Core chain executor
- `internal/testing/chains/validator.go` (~200 LOC) - Stage validators
- `internal/testing/chains/mock_provider.go` (~250 LOC) - Mock AI responses
- `internal/testing/chains/artifacts.go` (~150 LOC) - Artifact tracking
- `tests/chains/` (~500 LOC) - Test chain definitions
- `docs/testing/test-chains.md` (~200 lines) - Documentation

**Modified files:**
- `internal/coordinator/executor.go` (+50 LOC) - Add test mode support
- `internal/executor/claude/claude.go` (+30 LOC) - Mock provider injection
- `internal/executor/gemini/gemini.go` (+30 LOC) - Mock provider injection
- `Makefile` (+20 LOC) - Add test-chains target
- `.github/workflows/ci.yml` (+10 LOC) - CI integration

**Total new code:** ~1,000 LOC
**Total modifications:** ~140 LOC

## Examples

### Test Chain Definition

```yaml
# tests/chains/design-to-sprint.yaml
name: design_to_sprint_chain
description: Test design-doc-creator to sprint-planner handoff

stages:
  - id: design_doc_creator
    agent: design-doc-creator
    input:
      message: "Create design doc for semantic caching"
      title: "Feature: Semantic Caching"
    mock_response_file: mocks/design_doc_response.json
    validators:
      - type: file_exists
        path: "design_docs/planned/*/semantic-caching.md"
      - type: content_contains
        file: "design_docs/planned/*/semantic-caching.md"
        patterns:
          - "## Problem Statement"
          - "## Solution Design"
      - type: output_marker
        marker: "DESIGN_DOC_PATH"
    artifacts:
      - name: design_doc_path
        from: output_marker
        pattern: "DESIGN_DOC_PATH: (.*)"

  - id: sprint_planner
    agent: sprint-planner
    depends_on: design_doc_creator
    input:
      message: "Plan sprint for {{artifacts.design_doc_path}}"
      title: "Sprint Planning"
    mock_response_file: mocks/sprint_plan_response.json
    validators:
      - type: file_exists
        path: "design_docs/planned/*/SPRINT-PLAN.md"
      - type: velocity_calculation
        min_velocity: 50
        max_velocity: 200
    artifacts:
      - name: sprint_plan_path
        from: output_marker
        pattern: "SPRINT_PLAN: (.*)"

assertions:
  - chain_completed: true
  - all_stages_passed: true
  - execution_time_seconds: < 60
```

### Running Test Chains

```bash
# Run all test chains
make test-chains

# Run specific chain
ailang test-chain tests/chains/design-to-sprint.yaml

# Run with verbose output
ailang test-chain --verbose tests/chains/design-to-sprint.yaml

# Record new golden files
ailang test-chain --record tests/chains/design-to-sprint.yaml

# Run in CI mode (fail on any difference)
ailang test-chain --ci tests/chains/design-to-sprint.yaml
```

### Mock Response Format

```json
{
  "version": "1.0",
  "responses": [
    {
      "turn": 1,
      "type": "text",
      "content": "I'll create a design document for semantic caching..."
    },
    {
      "turn": 1,
      "type": "tool_use",
      "tool": "Write",
      "parameters": {
        "file_path": "design_docs/planned/v0_7_1/semantic-caching.md",
        "content": "# Semantic Caching Design\n\n..."
      }
    },
    {
      "turn": 2,
      "type": "text",
      "content": "**DESIGN_DOC_PATH:** `design_docs/planned/v0_7_1/semantic-caching.md`"
    }
  ]
}
```

## Success Criteria

- [ ] Test chains framework implemented and documented
- [ ] At least 5 critical workflows have test chains
- [ ] All test chains pass in CI
- [ ] Mock responses provide 100% deterministic execution
- [ ] Artifact validation catches breaking changes
- [ ] Documentation includes examples for creating new chains
- [ ] Performance: chains execute in < 60s (vs. 5+ minutes with real AI)

## Timeline

**Week 1 (Days 1-2):**
- Day 1: Implement core framework and mock infrastructure
- Day 2: Coordinator integration and test suite creation

**Week 2:**
- Documentation and CI integration
- Create regression test suite
- Performance optimization if needed

## Related Documents

- `internal/coordinator/` - Coordinator implementation
- `internal/executor/` - AI executor interfaces
- `internal/testing/` - Existing test utilities
- [Coordinator Guide](docs/docs/guides/coordinator.md) - Coordinator documentation

## Trade-offs and Decisions

**Use YAML for chain definitions:**
- Pro: Human-readable, easy to maintain
- Pro: Supports complex validations and artifacts
- Con: Requires YAML parser dependency
- Decision: Accept dependency for better maintainability

**Mock at executor level (not HTTP):**
- Pro: Tests actual coordinator logic
- Pro: No network dependencies
- Pro: Faster execution
- Con: Doesn't test HTTP communication
- Decision: Mock at executor for speed and reliability

**Deterministic response playback:**
- Pro: 100% reproducible tests
- Pro: Fast execution (no AI latency)
- Con: Doesn't catch AI behavior changes
- Decision: Use mocks for CI, real AI for periodic validation

## Open Questions

- Should we support partial chain execution for debugging?
- How should we handle flaky tests due to timing issues?
- Should chains support parallel stage execution?

## Future Enhancements

- Visual test chain debugger
- Automatic mock generation from real executions
- Performance benchmarking framework
- Integration with observatory for trace analysis
- Support for conditional branching in chains
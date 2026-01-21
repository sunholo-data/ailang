# M-EXECUTOR-TEST: Comprehensive Test Coverage for Multi-Executor System

**Status**: 📋 Planning
**Target**: v0.6.3
**Priority**: P2 (Quality/Robustness)
**Estimated Duration**: 2-3 days (~18-22 hours)
**Task ID**: task-39525386

## Summary

The AILANG executor system (v0.6.1) provides a unified interface for multiple AI coding agent backends (Claude Code, Gemini CLI, OpenAI Codex). While implementation is complete and working, **test coverage is incomplete** with only 14 tests covering core functionality and edge cases.

This sprint improves test coverage by adding:
1. **Comprehensive unit tests** for executor interfaces
2. **Edge case handling** (timeouts, cancellation, errors)
3. **Cost calculation validation** across pricing models
4. **Error handling tests** for OpenAI Codex JSONL parsing
5. **Mock implementations** for deterministic testing
6. **Integration test improvements** with better failure diagnostics

**Goal**: Reach 70%+ test coverage on executor package with edge case validation and provider-agnostic abstractions.

## Current Status Analysis

### Test Inventory (14 tests across 3 files)

| File | Tests | Coverage | Key Tests |
|------|-------|----------|-----------|
| `factory_test.go` | 5 | Basic factory | Register, GetExecutor, ListAvailable, Config, Cost |
| `gemini/gemini_test.go` | 5 | Gemini basics | New, Model, Capabilities, CostModel, Registration |
| `integration_test.go` | 4 | Integration | ListAvailable, DefaultExecutor, EnvVar, HealthCheck |
| **Total** | **14** | ~40% | ✅ Basic paths, ❌ Edge cases |

### Critical Gaps Identified

#### 1. **Missing Claude Executor Unit Tests** ❌
- **File**: `internal/executor/claude/claude.go` (484 LOC)
- **Problem**: No test file exists (`claude_test.go`)
- **Impact**: 28% of executor code untested
- **Tests needed**:
  - Executor initialization with various configs
  - Command-line flag generation
  - Session ID validation (UUID vs non-UUID)
  - JSONL stream parsing
  - Event handler integration
  - Cost model accuracy

#### 2. **Timeout and Cancellation Testing** ❌
- **Problem**: `context.Context` cancellation not tested
- **Locations**: `Execute()`, `ExecuteStreaming()`, `HealthCheck()`
- **Tests needed**:
  - Context timeout before execution starts
  - Context cancellation mid-execution
  - Timeout recovery/cleanup

#### 3. **Error Handling Gaps** ❌
- **Missing tests**:
  - Invalid workspace paths
  - Missing CLI binary (e.g., `claude` not installed)
  - Malformed JSONL responses
  - Network errors during streaming
  - OpenAI Codex incompatible JSON format (from handoff task!)

#### 4. **OpenAI Codex JSONL Format Validation** 🔴 **CRITICAL**
- **Context**: Handoff asks if OpenAI Codex JSONL is same as Claude/Gemini
- **Finding**: OpenAI uses different JSONL schema than Claude/Gemini (documented in M-EXEC)
- **Tests needed**:
  - Parser handles different event types
  - Cost calculation from OpenAI responses
  - Token count extraction (different field names)
  - Graceful degradation if fields missing

#### 5. **Cost Model Edge Cases** ⚠️
- **Current tests**: Only basic calculation (1K input, 0.5K output, 2K cache)
- **Missing tests**:
  - Zero tokens (should respect MinimumCharge)
  - Very large token counts (overflow risk)
  - Cache creation tokens (`CacheCreationInputTokens`)
  - Different pricing models per provider

#### 6. **EventHandler Interface Tests** ❌
- **Problem**: `EventHandler` interface has no tests for:
  - Turn sequence (start→text→end)
  - Tool use events
  - Error callbacks
  - Context-aware handlers (SetContext called before spans)

#### 7. **Factory Pattern Edge Cases** ⚠️
- **Current tests**: Basic happy path only
- **Missing tests**:
  - GetExecutor race conditions (concurrent calls)
  - Register overwrites (same executor name twice)
  - Close() with partially initialized executors
  - AILANG_EXECUTOR env var precedence
  - Multiple factory instances

#### 8. **Task Execution Result Validation** ❌
- **Missing tests**:
  - FilesCreated/FilesModified tracking
  - ProviderData preservation (raw response)
  - Transcript reconstruction
  - Token usage accumulation across turns

### Code Quality Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test Coverage | ~40% | 70%+ |
| Test/Code Ratio | 1:12 | 1:3 |
| Edge Case Tests | 0% | 60%+ |
| Mock Implementations | Minimal | Comprehensive |

## Goals

1. **Primary**: Reach 70%+ test coverage on executor package
2. **Secondary**: Document OpenAI Codex JSONL differences and add parser tests
3. **Tertiary**: Establish test patterns for provider-specific implementations
4. **Quality**: Add edge case testing (timeouts, errors, malformed input)

## Proposed Solution

### Milestone 1: Claude Executor Unit Tests (~6 hours)
**Create `internal/executor/claude/claude_test.go` with:**
- [ ] `TestClaudeExecutorNew()` - Config validation, defaults
- [ ] `TestClaudeBuildCommand()` - Flag generation
- [ ] `TestClaudeSessionID()` - UUID vs non-UUID handling
- [ ] `TestClaudeJSONLParsing()` - Event stream parsing
- [ ] `TestClaudeExecuteStreaming()` - Streaming with mock handler
- [ ] `TestClaudeEventHandler()` - EventHandler callback sequence
- **Est. LOC**: ~280 (implementation + test)
- **Files to create**: `internal/executor/claude/claude_test.go`
- **Example test**:
  ```go
  func TestClaudeBuildCommand(t *testing.T) {
      exec := &ClaudeExecutor{
          claudePath: "/usr/bin/claude",
          model: "haiku",
          allowedTools: []string{"Bash", "Read"},
      }
      cmd := exec.buildCommand(context.Background(), "fix the bug")

      // Verify args include -p, --output-format stream-json, etc.
      if cmd.String() != "..." {
          t.Errorf("unexpected command: %s", cmd.String())
      }
  }
  ```

### Milestone 2: Timeout & Cancellation Tests (~4 hours)
**Enhance all executor test files with context tests:**
- [ ] `TestExecuteWithContextCancellation()` - Cancellation mid-execution
- [ ] `TestExecuteWithTimeout()` - Timeout before execution
- [ ] `TestHealthCheckTimeout()` - Health check timeout handling
- [ ] `TestStreamingWithContextError()` - Error propagation during streaming
- [ ] Add to both `claude_test.go` and `gemini/gemini_test.go`
- **Est. LOC**: ~180
- **Key pattern**:
  ```go
  func TestExecuteWithContextCancellation(t *testing.T) {
      ctx, cancel := context.WithCancel(context.Background())
      cancel() // Cancel immediately

      result, err := executor.Execute(ctx, &Task{...})
      if err == nil {
          t.Error("expected context.Canceled error")
      }
  }
  ```

### Milestone 3: Error Handling & Edge Cases (~5 hours)
**Add to `factory_test.go` and `integration_test.go`:**
- [ ] `TestMissingCLIBinary()` - Handle missing `claude`/`gemini`/`codex` binary
- [ ] `TestMalformedJSONLResponse()` - Parse error handling
- [ ] `TestInvalidWorkspacePath()` - Directory validation
- [ ] `TestCostCalculationEdgeCases()` - Zero tokens, overflow, cache creation
- [ ] `TestExecutorCloseErrors()` - Error handling in Close()
- [ ] `TestConcurrentFactoryGetExecutor()` - Race condition testing
- **Est. LOC**: ~220
- **Test example**:
  ```go
  func TestZeroTokensCostMinimumCharge(t *testing.T) {
      model := &CostModel{MinimumCharge: 0.01}
      usage := TokenUsage{} // All zeros

      cost := model.CalculateCost(usage)
      if cost != 0.01 {
          t.Errorf("expected minimum charge 0.01, got %f", cost)
      }
  }
  ```

### Milestone 4: OpenAI Codex JSONL Research & Tests (~3 hours)
**Address handoff question: "Is OpenAI Codex JSONL same as Claude/Gemini?"**
- [ ] Research OpenAI Codex CLI `--json` output format
- [ ] Document differences in design doc (append to M-EXEC)
- [ ] Create reference test file showing Codex JSON structure
- [ ] Add parser validation test for incompatible formats
- [ ] Create test data file: `testdata/codex_response.jsonl`
- **Est. LOC**: ~120
- **Deliverable**:
  ```markdown
  ## OpenAI Codex JSONL Format

  **Conclusion**: OpenAI Codex uses DIFFERENT JSONL schema than Claude/Gemini

  | Field | Claude | Gemini | Codex |
  |-------|--------|--------|-------|
  | event_type | `text_delta` | `output` | `message` |
  | content | `.content` | `.response` | `.text` |
  | turn_id | ❌ | ❌ | ✅ `turn_number` |

  **Recommendation**: If implementing Codex support, create separate parser for Codex JSONL.
  ```

### Milestone 5: EventHandler & Integration Tests (~2 hours)
**Enhance `integration_test.go`:**
- [ ] `TestEventHandlerSequence()` - Turn start→text→tool→end flow
- [ ] `TestContextAwareHandlerSetContext()` - Handler receives proper context
- [ ] `TestFactoryConcurrency()` - 10 concurrent GetExecutor calls
- [ ] `TestExecutorCloseMultiple()` - Close already-closed executor
- **Est. LOC**: ~150
- **Pattern**:
  ```go
  type TestEventHandler struct {
      events []string
  }
  func (h *TestEventHandler) OnTurnStart(num int) { h.events = append(h.events, "start") }
  // ... other methods ...

  func TestEventHandlerSequence(t *testing.T) {
      h := &TestEventHandler{}
      // Execute task, verify: start→text→tool→end order
  }
  ```

## Test Implementation Pattern

All tests follow AILANG conventions:
- Use `*testing.T` (no t.Run subtests for clarity)
- Table-driven tests for multiple cases
- Helpful error messages with expected/actual values
- No external dependencies (use mocks)
- Parallel-safe (t.Parallel() where appropriate)

Example test structure:
```go
func TestCLASSMethodEDGECASE(t *testing.T) {
    // Setup
    obj := NewTestObject()

    // Execute
    result, err := obj.Method(input)

    // Verify
    if err != nil {
        t.Fatalf("Method failed: %v", err)
    }
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

## Acceptance Criteria

- ✅ All 47 new tests pass
- ✅ Test coverage reaches 70%+ for executor package (currently ~40%)
- ✅ Edge cases covered: timeouts, cancellation, errors, malformed input
- ✅ OpenAI Codex JSONL format documented with examples
- ✅ No external runtime dependencies (all mocks are hermetic)
- ✅ Mock implementations reusable for future executor additions
- ✅ All existing tests still pass (no regressions)
- ✅ `make lint` passes (golangci-lint)
- ✅ `make test` passes (go test ./...)
- ✅ Coverage badge updated

## Risk Factors

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gemini/Claude CLI not installed | Medium | Use `t.Skip()` for integration tests; mock everything in unit tests |
| JSONL parsing complexity | Medium | Reference real CLI output; keep parsers simple and focused |
| Race conditions in factory | Low | Use `sync.WaitGroup` in test; run with `-race` flag |
| Test maintenance burden | Low | Use table-driven tests; minimal mocking; focus on behavior |

## Dependencies

- **None** - Tests are hermetic (no external services required)
- Existing: `testing` package, `context` package, `sync` for concurrency
- Optional: Run with `-race` flag for race condition detection

## Success Metrics

1. **Coverage**: 70%+ on executor package (from 40%)
2. **Test Count**: 47+ new tests (from 14)
3. **Handoff Q&A**: Answer "Is OpenAI Codex JSONL same as Claude/Gemini?" with documented proof
4. **Patterns**: Established reusable test patterns for future executors
5. **Maintenance**: All tests pass with `make test` and `make lint`

## Implementation Notes

### Test File Organization
```
internal/executor/
├── executor.go              (core interface)
├── executor_test.go         (NEW: core interface + factory tests)
├── factory.go               (factory implementation)
├── factory_test.go          (EXPAND: add edge cases + concurrency)
├── environment.go           (environment setup)
├── environment_test.go      (NEW: timeout + cancellation)
├── claude/
│   ├── claude.go
│   ├── claude_test.go       (NEW: comprehensive)
│   └── testdata/            (NEW: sample JSONL files)
├── gemini/
│   ├── gemini.go
│   ├── gemini_test.go       (EXPAND: timeouts + errors)
│   └── testdata/            (NEW: sample JSONL files)
└── integration_test.go      (EXPAND: EventHandler + concurrency)
```

### Test Data Files
Create `testdata/` directories with example JSONL:
- `gemini_response.jsonl` - Real Gemini CLI output
- `claude_response.jsonl` - Real Claude Code output
- `codex_response.jsonl` - OpenAI Codex format (if available)
- `malformed.jsonl` - Invalid input for error testing

### Mock Strategy
- `MockExecutor` - Already exists in factory_test.go (reuse)
- `MockEventHandler` - Track events in order
- `MockCLI` - Execute subprocess for testing

## Timeline Estimate

| Milestone | Duration | LOC | Tests | Days |
|-----------|----------|-----|-------|------|
| 1. Claude Unit Tests | 6h | 280 | 6 | 1 |
| 2. Timeout/Cancel | 4h | 180 | 5 | 0.5 |
| 3. Error Handling | 5h | 220 | 8 | 1 |
| 4. Codex Research | 3h | 120 | 3 | 0.5 |
| 5. Integration/Handlers | 2h | 150 | 5 | 0.5 |
| **Total** | **20h** | **950** | **27** | **3.5** |

**Note**: Days estimate assumes ~6 hours coding/testing per day. Total LOC includes both tests and new code.

## Next Steps (After Approval)

1. ✅ Receive approval on this plan
2. ✅ Create JSON progress file with milestones
3. ✅ Hand off to sprint-executor for implementation
4. ✅ Execute with TDD (tests first, then implementation)
5. ✅ Verify all tests pass
6. ✅ Update test coverage badge
7. ✅ Commit with message referencing handoff task

---

## Handoff Question Resolution

**Q**: "Is OpenAI Codex JSONL same as Claude/Gemini?"

**A**: **NO** - Different JSONL schemas.

As documented in M-EXEC (v0.6.1):
- **Claude Code**: Uses `--output-format stream-json` with `event_type`, `content` fields
- **Gemini CLI**: Uses `--output-format stream-json` with nearly identical schema to Claude
- **OpenAI Codex**: Uses `--json` flag with DIFFERENT schema (field names, structure)

**Implication**: If Codex executor is implemented, separate JSONL parser is required.

**Test Coverage**: This sprint adds validation tests for JSON parsing to catch schema mismatches early.

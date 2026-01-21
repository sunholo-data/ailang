# M-COORD-TASKCHAIN-TESTS: Add TaskChain Test Coverage

## Status
**Status**: Planned
**Target Version**: v0.6.3
**Priority**: P0 (Pre-requisite for M-COORD-GENERIC-WORKFLOWS)
**Created**: 2026-01-01

## Problem Statement

TaskChain is the core integration component that connects:
- Stage completion → GitHub comments
- Approval detection → Stage transitions
- Pipeline flow control

**Current state: ZERO tests.** This is a critical gap before refactoring to generic workflows.

## Goals

Add comprehensive test coverage for TaskChain to:
1. Establish regression baseline before refactoring
2. Document current behavior via tests
3. Enable safe refactoring to generic workflows

## Test Plan

### Test Cases (~300 LOC)

```go
// task_chain_test.go

// 1. Basic lifecycle tests
func TestTaskChain_NewTaskChain()           // Creates with handlers registered
func TestTaskChain_StartTask()              // Links to GitHub, sets stage, starts watching

// 2. Design doc stage tests
func TestTaskChain_OnDesignDocComplete_PostsComment()
func TestTaskChain_OnDesignDocComplete_AddsLabel()
func TestTaskChain_OnDesignDocComplete_NoGitHub()  // Skips when no issue linked
func TestTaskChain_OnDesignApproved_TransitionsStage()
func TestTaskChain_OnDesignApproved_RequeuesTask()

// 3. Sprint plan stage tests
func TestTaskChain_OnSprintPlanComplete_PostsComment()
func TestTaskChain_OnSprintPlanComplete_AddsLabel()
func TestTaskChain_OnSprintApproved_TransitionsStage()
func TestTaskChain_OnSprintApproved_RequeuesTask()

// 4. Implementation stage tests
func TestTaskChain_OnImplementationComplete_PostsComment()
func TestTaskChain_OnImplementationComplete_AddsLabel()
func TestTaskChain_OnMergeApproved_ClearsStage()
func TestTaskChain_OnMergeApproved_ClosesIssue()
func TestTaskChain_OnMergeApproved_UnwatchesIssue()

// 5. Error handling tests
func TestTaskChain_OnNeedsRevision_PostsComment()
func TestTaskChain_OnError_PostsErrorComment()

// 6. Integration tests
func TestTaskChain_FullPipeline_DesignToMerge()  // End-to-end flow
```

## Implementation

### Mock Requirements

Need mocks for:
- `GitHubPoster` - Capture comment/label calls
- `Store` - In-memory task storage
- `ApprovalWatcher` - Track handler registration

### File Structure

```
internal/coordinator/
├── task_chain.go           # Existing (~420 LOC)
├── task_chain_test.go      # NEW (~300 LOC)
└── mock_github_test.go     # Enhance existing mock
```

## Acceptance Criteria

- [ ] 15+ test functions covering all TaskChain methods
- [ ] Tests document current hardcoded behavior
- [ ] All tests pass with existing code
- [ ] Coverage for task_chain.go > 80%

## Dependencies

None - this is a pre-requisite for M-COORD-GENERIC-WORKFLOWS.

# M-COORDINATOR-TEST-WORKFLOW: Simple Coordinator Test File Creation

**Status:** Planned
**Target:** v0.6.4
**Priority:** P1 (Medium)
**Estimated:** 0.5 days
**Dependencies:** None
**Created:** 2026-01-15

## Problem Statement

The coordinator daemon requires comprehensive test coverage for basic file operations and message handling workflows. Currently, there is no simple automated test that verifies the coordinator can:
1. Receive a message from the user
2. Execute a basic task (file creation)
3. Return results with proper artifact tracking

This design doc establishes a lightweight test workflow that validates the core coordinator message-to-artifact pipeline.

## Goals

**Primary Goal:** Create a simple, reproducible test that verifies coordinator basic functionality

**Success Metrics:**
- [ ] Test file created at expected path with correct content
- [ ] Coordinator can track artifact creation
- [ ] Output includes proper DESIGN_DOC_PATH markers for artifact tracking
- [ ] Test passes consistently across multiple runs
- [ ] Test can be automated in CI/CD pipeline

## Solution Design

### Overview

This test creates a minimal coordinator workflow:
1. User sends message requesting file creation
2. Coordinator receives message and triggers task
3. Task creates test file at `/tmp/ailang-test-{timestamp}.txt`
4. Task outputs artifact tracking markers
5. Coordinator records completion and moves to approval queue

### Architecture

```
[User Message]
    ↓
[Coordinator Inbox]
    ↓
[Task Executor]
    ├─→ Create /tmp/ailang-test-{timestamp}.txt
    ├─→ Output artifact markers
    └─→ Return success status
    ↓
[Approval Request]
    ↓
[Artifact Tracked]
```

### Implementation Plan

**Phase 1: Test File Creation** (~1 hour)
- [ ] Create test file at `/tmp/ailang-test-{timestamp}.txt`
- [ ] Verify file contents: `'Hello from coordinator test'`
- [ ] Verify file permissions and encoding (ASCII text)
- [ ] Add basic validation test

**Phase 2: Design Documentation** (~0.5 hours)
- [ ] Create this design document
- [ ] Document artifact output format
- [ ] Add example workflow execution

**Phase 3: Coordinator Integration** (~1 hour)
- [ ] Verify coordinator can process this task
- [ ] Test artifact marker parsing
- [ ] Test CI integration and artifact tracking

### Files to Modify/Create

| File | Status | LOC | Purpose |
|------|--------|-----|---------|
| `/tmp/ailang-test-1768462190.txt` | New | 1 | Test artifact file |
| `design_docs/planned/v0_6_4/m-coordinator-test-workflow.md` | New | 50-75 | This design doc |
| Tests (future) | Planned | 30-50 | Automated test coverage |

## Examples

### Example Workflow Execution

**Input Message:**
```bash
ailang messages send coordinator "Create a simple test file" \
  --title "Test: Basic File Creation" \
  --from "user"
```

**Coordinator Processing:**
1. Receives message in coordinator inbox
2. Executes file creation task
3. Outputs artifact marker: `DESIGN_DOC_PATH: design_docs/planned/v0_6_4/m-coordinator-test-workflow.md`
4. Creates approval request

**Example Output:**
```
Test file created successfully at /tmp/ailang-test-1768462190.txt
Content: "Hello from coordinator test"
Status: ✅ PASSED

DESIGN_DOC_PATH: design_docs/planned/v0_6_4/m-coordinator-test-workflow.md
```

## Success Criteria

- [x] Test file created at `/tmp/ailang-test-1768462190.txt`
- [x] File contains exactly: `'Hello from coordinator test'`
- [x] File is ASCII encoded
- [x] File size is 28 bytes
- [ ] Automated test passes
- [ ] Design doc created
- [ ] CI integration working

## Timeline

**Day 1 (Today):**
- 1 hour: Create test file and verify
- 0.5 hours: Write design document
- 0.5 hours: Document and commit

**Day 2 (Optional):**
- 1 hour: Add automated tests
- 0.5 hours: Verify CI integration
- 0.5 hours: Final review and documentation

## Related Documents

- [Coordinator Design](../../../design_docs/implemented/v0_6_2/coordinator-daemon.md) - Coordinator architecture
- [Message System](../../../design_docs/implemented/v0_6_3/messages-api.md) - Message handling
- [Task Tracking](../../../design_docs/implemented/v0_6_0/task-tracking.md) - Artifact tracking system
- [Testing Strategy](../../../docs/guides/testing.md) - General testing patterns

## Notes

- This is a minimal test workflow designed for CI/CD integration
- The test file path includes a timestamp to ensure uniqueness
- Artifact markers (DESIGN_DOC_PATH) are parsed by coordinator for tracking
- This workflow validates the complete message-to-artifact pipeline
- Can be extended with edge case testing (permissions, disk space, etc.)

## Acceptance Criteria Checklist

### Functional Requirements
- [x] Test file created with correct path
- [x] Content matches expected string
- [x] File encoding is ASCII
- [ ] Coordinator successfully processes task
- [ ] Artifact markers properly formatted

### Code Quality
- [ ] All tests passing
- [ ] No linting errors
- [ ] Code coverage adequate
- [ ] Documentation complete

### Integration
- [ ] CI pipeline recognizes test
- [ ] Dashboard shows completion
- [ ] Approval workflow triggered
- [ ] Artifact tracking validated

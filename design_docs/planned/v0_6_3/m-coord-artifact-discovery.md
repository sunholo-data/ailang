# M-COORD-ARTIFACT-DISCOVERY: Fix Coordinator Skill Invocation and Artifact Discovery

**Status**: Planned
**Target**: v0.6.3
**Priority**: P0 (High - blocking coordinator workflow)
**Estimated**: 6 hours
**Dependencies**: None

## Expected End-to-End Flow

This is what SHOULD work:

```
1. GitHub Issue Created
   └─→ import-github --inbox design-doc-creator
       └─→ Message{inbox: "design-doc-creator", github_issue: 102, content: "..."}

2. Coordinator Polls Inbox
   └─→ Task{AgentID: "design-doc-creator", GithubIssue: 102, Content: "..."}

3. Directive Building (using AgentConfig)
   └─→ "GitHub issue #102: <content>

        You must invoke the design-doc-creator skill to complete this task.

        When done, output: DESIGN_DOC_PATH: <path>"

4. Agent Executes in Worktree
   └─→ Claude Code invokes .claude/skills/design-doc-creator
       └─→ Creates: design_docs/planned/v0_6_3/feature.md
       └─→ Outputs: "DESIGN_DOC_PATH: design_docs/planned/v0_6_3/feature.md"

5. Artifact Discovery (git diff + pattern matching)
   └─→ Pattern: "design_docs/**/*.md"
   └─→ Found: ["design_docs/planned/v0_6_3/feature.md"]

6. Post to GitHub Issue #102
   └─→ Comment with design doc content in collapsible <details> tag
   └─→ Add label: "needs-design-approval"
   └─→ IF no artifacts: Post "Failed to find artifacts matching pattern"

7. Human Approves → Next Stage
```

## Problem Statement

The flow above is BROKEN at multiple points:

### Bug 1: AgentID Not Set on Task

**File**: `daemon_tasks.go:257-269`

```go
task := &TaskRecord{
    ID:          taskID,
    MessageID:   msg.ID,
    Title:       msg.Title,
    // AgentID: missing! (agentID variable exists but not assigned)
    ...
}
```

### Bug 2: Directive Falls Back to Raw Content

**File**: `stage_execution.go:40-42`

```go
if task.GithubIssue == 0 || task.Stage == TaskStageNone {
    return task.Content  // Returns raw message, no skill invocation!
}
```

The `||` should be `&&`. Currently, if EITHER condition is true, it skips skill invocation.

### Bug 3: No Error Reporting When Artifacts Missing

Currently if no artifacts found, the GitHub comment is posted with empty content. Should say "Failed to find artifacts matching pattern: design_docs/**/*.md"

## Solution Design

### Phase 1: Add Tests for Each Step (~2 hours)

Create `internal/coordinator/workflow_test.go` with tests:

```go
// Test 1: Task creation sets AgentID
func TestTaskCreationSetsAgentID(t *testing.T) {
    // Given: message in "design-doc-creator" inbox
    // When: task created from message
    // Then: task.AgentID == "design-doc-creator"
}

// Test 2: Directive includes skill invocation
func TestDirectiveIncludesSkillInvocation(t *testing.T) {
    // Given: task with AgentID="design-doc-creator"
    // When: BuildDirectiveFromConfig called
    // Then: directive contains "invoke the design-doc-creator skill"
}

// Test 3: Artifact discovery finds matching files
func TestArtifactDiscoveryFindsFiles(t *testing.T) {
    // Given: worktree with design_docs/planned/test.md
    // When: DiscoverChangedFiles with pattern "design_docs/**/*.md"
    // Then: returns ["design_docs/planned/test.md"]
}

// Test 4: GitHub comment includes artifact content
func TestGitHubCommentIncludesArtifactContent(t *testing.T) {
    // Given: discovered artifact with content
    // When: RenderDesignDocComment called
    // Then: comment includes content in <details> tag
}

// Test 5: GitHub comment reports failure when no artifacts
func TestGitHubCommentReportsNoArtifacts(t *testing.T) {
    // Given: no artifacts discovered
    // When: ProcessStageCompletion called
    // Then: GitHub comment says "Failed to find artifacts"
}
```

### Phase 2: Fix Task Creation (~30 min)

**File**: `daemon_tasks.go:257-269`

```go
task := &TaskRecord{
    ID:          taskID,
    MessageID:   msg.ID,
    AgentID:     agentID,  // ADD THIS LINE
    Title:       msg.Title,
    Content:     msg.Content,
    ...
}
```

### Phase 3: Fix Directive Fallback (~30 min)

**File**: `stage_execution.go:38-53`

```go
func BuildStageDirective(task *TaskRecord) string {
    // Only fall back to raw content if we truly have no agent info
    if task.AgentID == "" && task.Stage == TaskStageNone {
        return task.Content
    }

    // Use AgentID if available, otherwise derive from Stage
    agentID := task.AgentID
    if agentID == "" {
        agentID = stageToAgentIDForDirective(task.Stage)
    }

    if agentID == "" {
        return task.Content
    }

    agent := &AgentConfig{ID: agentID}
    return BuildDirectiveFromConfig(task, agent)
}
```

### Phase 4: Add Failure Reporting (~1 hour)

**File**: `stage_execution.go` (ProcessStageCompletion)

```go
// After artifact discovery
if len(discoveredArtifacts) == 0 {
    // Post failure message to GitHub instead of empty content
    if task.GithubIssue > 0 && d.taskChain != nil {
        patterns := getArtifactPatterns(task)
        failureMsg := fmt.Sprintf(
            "## Artifact Discovery Failed\n\n"+
            "No artifacts found matching patterns: %v\n\n"+
            "Agent output (first 500 chars):\n```\n%s\n```",
            patterns, truncate(execResult.Output, 500))
        d.taskChain.PostComment(ctx, task.GithubIssue, failureMsg)
    }
    return fmt.Errorf("no artifacts discovered for patterns: %v", patterns)
}
```

### Phase 5: Improve Logging (~30 min)

Add detailed logging at each step:

```go
// Task creation
d.logger.Printf("Created task %s from inbox %s (AgentID: %s, GithubIssue: %d)",
    task.ID, inbox, task.AgentID, task.GithubIssue)

// Directive building
d.logger.Printf("Built directive for task %s (AgentID: %s):\n%s",
    task.ID, task.AgentID, truncate(directive, 200))

// Artifact discovery
d.logger.Printf("Artifact discovery for task %s: patterns=%v, found=%d files: %v",
    task.ID, patterns, len(artifacts), artifacts)

// GitHub posting
d.logger.Printf("Posting to GitHub issue #%d: %d artifacts, content length: %d",
    task.GithubIssue, len(artifacts), len(content))
```

### Phase 6: Integration Test (~1 hour)

Create `internal/coordinator/e2e_workflow_test.go`:

```go
func TestEndToEndWorkflow(t *testing.T) {
    // 1. Create message in design-doc-creator inbox
    // 2. Run daemon poll cycle
    // 3. Verify task created with AgentID
    // 4. Verify directive includes skill invocation
    // 5. Mock executor that creates design doc
    // 6. Verify artifact discovered
    // 7. Verify GitHub comment posted with content
}
```

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `daemon_tasks.go` | Set AgentID on task creation | +1 |
| `stage_execution.go` | Fix directive fallback, add failure reporting | +30 |
| `workflow_test.go` | New: Unit tests for each step | +200 |
| `e2e_workflow_test.go` | New: Integration test | +150 |

## Success Criteria

- [ ] Test: Task from inbox has AgentID set
- [ ] Test: Directive contains "invoke the X skill"
- [ ] Test: Artifact discovery finds matching files
- [ ] Test: GitHub comment includes artifact content
- [ ] Test: Missing artifacts reports error (not empty)
- [ ] Manual: Send message → skill invoked → artifact posted
- [ ] All existing tests pass

## Testing Commands

```bash
# Run new workflow tests
go test ./internal/coordinator/... -run TestWorkflow -v

# Run E2E test
go test ./internal/coordinator/... -run TestEndToEnd -v

# Manual test
./bin/ailang messages send design-doc-creator "Test skill invocation" --title "Test"
./bin/ailang messages unack <msg-id>
# Wait for coordinator poll
tail -f ~/.ailang/logs/coordinator.log | grep -E "(AgentID|directive|artifact|GitHub)"
```

## Implementation Order

1. **Write tests first** (TDD) - they will fail initially
2. Fix task creation (AgentID)
3. Fix directive fallback
4. Add failure reporting
5. Add logging
6. Run tests - should all pass
7. Manual E2E verification

---

**Document created**: 2026-01-02
**Last updated**: 2026-01-02

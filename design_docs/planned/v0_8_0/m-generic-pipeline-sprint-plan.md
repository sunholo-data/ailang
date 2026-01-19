# Sprint Plan: M-GENERIC-PIPELINE

**Sprint ID**: M-GENERIC-PIPELINE
**Design Doc**: [m-generic-pipeline.md](m-generic-pipeline.md)
**Duration**: 2-3 days (~16-20 hours)
**Risk Level**: Medium (touches core coordinator logic)
**Target**: v0.8.0

---

## Sprint Summary

**Goal**: Unify the two parallel pipeline systems by deprecating hardcoded `TaskStage` enum and using generic `trigger_on_complete` configuration for all workflows, including GitHub-linked tasks.

**Key Deliverables**:
1. Unified `OnAgentComplete()` handler replacing stage-specific handlers
2. Config-driven approval label handling via `RegisterAgentApprovalHandlers()`
3. Deprecation markers on TaskStage enum and related code
4. All existing tests passing with backwards compatibility

---

## Velocity Analysis

**Recent velocity** (from CHANGELOG and git history):
- Coordinator features typically: 150-200 LOC/day
- CI/test fixes: 50-100 LOC/day
- Refactoring work: 200-300 LOC/day (removing > adding)

**This sprint estimate**:
- ~300 LOC net removal (simplification)
- ~150 LOC new code (unified handlers, config lookup)
- Total work: ~450 LOC changes over 2-3 days

---

## Current Implementation Analysis

### Files to Modify

| File | Current LOC | Changes | Net LOC |
|------|-------------|---------|---------|
| `stage_execution.go` | 553 | Remove `stageToAgentIDForDirective()`, simplify `BuildStageDirective()` | -80 |
| `task_chain.go` | 571 | Replace `OnDesignDocComplete`, `OnSprintPlanComplete`, `OnImplementationComplete` with `OnAgentComplete()` | -150 |
| `daemon_approval.go` | 695 | Remove stage switch in `HandleApproval()`, use agent config | -50 |
| `approval_watcher.go` | 514 | Add `RegisterAgentApprovalHandlers()`, remove hardcoded constants | +30 |
| `store.go` | ~200 | Add deprecation comments, new methods | +20 |
| `agent_registry.go` | ~300 | Add `GetEffectiveApprovalConfig()` method | +20 |
| **Total** | 2333 | | **-210 net** |

### Hardcoded Elements to Remove

```go
// approval_watcher.go:19-22 - Hardcoded label constants
LabelNeedsDesignApproval = "needs-design-approval"
LabelNeedsSprintApproval = "needs-sprint-approval"
LabelNeedsMergeApproval  = "needs-merge-approval"

// stage_execution.go:61-72 - Hardcoded stage-to-agent mapping
func stageToAgentIDForDirective(stage TaskStage) string {
    switch stage {
    case TaskStageDesign:
        return "design-doc-creator"  // REMOVE
    ...
}

// task_chain.go - Stage-specific handlers
func (tc *TaskChain) OnDesignDocComplete(...)   // → OnAgentComplete()
func (tc *TaskChain) OnSprintPlanComplete(...)  // → OnAgentComplete()
func (tc *TaskChain) OnImplementationComplete(...) // → OnAgentComplete()
```

---

## Milestones

### M1: Add ApprovalConfig to AgentConfig (~3 hours)

**Files**:
- `internal/coordinator/agent_registry.go`
- `internal/coordinator/agent_config.go`

**Tasks**:
- [ ] Add `ApprovalConfig` struct with `NeedsLabel`, `ApprovedLabel`, `CommentTemplate` fields
- [ ] Add `GetEffectiveApprovalConfig()` method with legacy defaults for known agents
- [ ] Add YAML/JSON tags for config file support
- [ ] Unit tests for effective config resolution

**Acceptance Criteria**:
- `agent.GetEffectiveApprovalConfig()` returns correct defaults for "design-doc-creator", "sprint-planner", "sprint-executor"
- Custom approval config from YAML overrides defaults
- Unknown agents return nil (no approval workflow)

**Example**:
```go
// agent_registry.go
type ApprovalConfig struct {
    NeedsLabel      string `yaml:"needs_label" json:"needs_label"`
    ApprovedLabel   string `yaml:"approved_label" json:"approved_label"`
    CommentTemplate string `yaml:"comment_template" json:"comment_template"`
}

func (a *AgentConfig) GetEffectiveApprovalConfig() *ApprovalConfig {
    if a.Approval != nil {
        return a.Approval
    }
    return DefaultApprovalConfig(a.ID) // Legacy defaults
}
```

---

### M2: Create Unified OnAgentComplete Handler (~4 hours)

**Files**:
- `internal/coordinator/task_chain.go`

**Tasks**:
- [ ] Create `OnAgentComplete(ctx, taskID, agentID, result)` unified handler
- [ ] Move common logic from `OnDesignDocComplete`, `OnSprintPlanComplete`, `OnImplementationComplete`
- [ ] Use `agent.GetEffectiveApprovalConfig()` for labels/comments
- [ ] Keep legacy handlers as thin wrappers (for backwards compat)
- [ ] Unit tests for unified handler

**Acceptance Criteria**:
- `OnAgentComplete()` posts correct comment and label for any agent
- Legacy handlers delegate to `OnAgentComplete()`
- GitHub comments include agent-specific content from templates
- Tests cover design-doc-creator, sprint-planner, sprint-executor, and custom agent

**Example**:
```go
// task_chain.go
func (tc *TaskChain) OnAgentComplete(ctx context.Context, taskID, agentID string, result *AgentResult) error {
    agent := tc.registry.GetAgentByID(agentID)
    approval := agent.GetEffectiveApprovalConfig()

    // Post completion comment
    comment, _ := RenderAgentCompleteComment(agentID, result)
    tc.poster.PostComment(task.GithubIssue, comment)

    // Add needs-approval label from config
    tc.poster.AddLabel(task.GithubIssue, approval.NeedsLabel)
    return nil
}
```

---

### M3: Dynamic Approval Label Registration (~3 hours)

**Files**:
- `internal/coordinator/approval_watcher.go`
- `internal/coordinator/task_chain.go`

**Tasks**:
- [ ] Create `RegisterAgentApprovalHandlers(registry)` method
- [ ] Iterate through registered agents, register handlers for each approval label
- [ ] Create `OnAgentApproved(ctx, event, agentID)` generic handler
- [ ] Remove hardcoded `LabelNeedsDesignApproval`, etc. usage (mark deprecated)
- [ ] Unit tests for dynamic registration

**Acceptance Criteria**:
- Approval handlers registered dynamically from agent config
- New agents with approval config are automatically watched
- Hardcoded constants still work (backwards compat) but log deprecation warning
- Tests cover adding a new agent with custom approval labels

**Example**:
```go
// approval_watcher.go
func (w *ApprovalWatcher) RegisterAgentApprovalHandlers(registry *AgentRegistry) {
    for _, agent := range registry.ListAgents() {
        approval := agent.GetEffectiveApprovalConfig()
        if approval == nil || approval.ApprovedLabel == "" {
            continue
        }
        // Register handler for this agent's approval label
        w.RegisterLabelHandler(approval.ApprovedLabel, func(ctx context.Context, event *ApprovalEvent) error {
            return w.chain.OnAgentApproved(ctx, event, agent.ID)
        })
    }
}
```

---

### M4: Update HandleApproval to Use Agent Config (~3 hours)

**Files**:
- `internal/coordinator/daemon_approval.go`

**Tasks**:
- [ ] Remove stage switch in `HandleApproval()` (lines 269-307)
- [ ] Look up agent from task.AgentID instead of task.Stage
- [ ] Use `trigger_on_complete` from agent config for handoff decisions
- [ ] Add label handling via `agent.GetEffectiveApprovalConfig()`
- [ ] Integration tests for approval workflow

**Acceptance Criteria**:
- `HandleApproval()` works identically for existing agents
- New agents with `trigger_on_complete` trigger correct handoffs
- GitHub label operations use config-driven labels
- E2E workflow test passes

**Example**:
```go
// daemon_approval.go - HandleApproval (simplified)
func (d *Daemon) HandleApproval(ctx context.Context, taskID, approvedBy string) error {
    task, _ := d.taskStore.GetTask(ctx, taskID)
    agent := d.agentRegistry.GetAgentByID(task.AgentID)

    // Config-driven label handling
    approval := agent.GetEffectiveApprovalConfig()
    if approval != nil {
        d.taskChain.poster.AddLabel(task.GithubIssue, approval.ApprovedLabel)
        d.taskChain.poster.RemoveLabel(task.GithubIssue, approval.NeedsLabel)
    }

    // Config-driven handoffs
    if len(agent.TriggerOnComplete) > 0 {
        return d.handleConfigDrivenApproval(ctx, task, agent, approvedBy)
    }

    // No handoffs - merge only
    return d.handleMergeApproval(ctx, task, approvedBy)
}
```

---

### M5: Deprecate TaskStage and Cleanup (~3 hours)

**Files**:
- `internal/coordinator/store.go`
- `internal/coordinator/stage_execution.go`
- `internal/coordinator/approval_watcher.go`

**Tasks**:
- [ ] Add deprecation comments to `TaskStage` enum
- [ ] Add deprecation comments to `stageToAgentIDForDirective()`
- [ ] Add deprecation warnings to hardcoded label constants
- [ ] Update `BuildStageDirective()` to prefer agent_id over stage
- [ ] Remove dead code paths (if any)
- [ ] Update CLAUDE.md to document config-driven approach

**Acceptance Criteria**:
- All deprecated code has `// Deprecated:` comments with migration guidance
- `go doc` shows deprecation warnings
- CLAUDE.md documents the new config-driven pipeline approach
- No compilation errors or test failures

**Example**:
```go
// store.go
// TaskStage represents the pipeline stage for GitHub-linked tasks.
// Deprecated: Use agent_id tracking instead. Stage is maintained for backwards
// compatibility with existing tasks. Will be removed in v0.9.0.
// See M-GENERIC-PIPELINE design doc for migration guidance.
type TaskStage string
```

---

## Test Plan

### Unit Tests

| Test | File | Coverage |
|------|------|----------|
| `TestGetEffectiveApprovalConfig_Defaults` | `agent_registry_test.go` | M1 |
| `TestGetEffectiveApprovalConfig_Custom` | `agent_registry_test.go` | M1 |
| `TestOnAgentComplete_DesignDoc` | `task_chain_test.go` | M2 |
| `TestOnAgentComplete_CustomAgent` | `task_chain_test.go` | M2 |
| `TestRegisterAgentApprovalHandlers` | `approval_watcher_test.go` | M3 |
| `TestHandleApproval_ConfigDriven` | `daemon_approval_test.go` | M4 |
| `TestBackwardsCompatibility_TaskStage` | `workflow_test.go` | M5 |

### Integration Tests

| Test | Description |
|------|-------------|
| `TestE2EWorkflow_GitHubPipeline` | Full pipeline with GitHub issue |
| `TestE2EWorkflow_CustomAgent` | Custom agent added via config |
| `TestE2EWorkflow_LegacyStage` | Backwards compat with TaskStage |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing GitHub workflows | Keep legacy handlers as thin wrappers, extensive E2E tests |
| Missing edge cases in label handling | Test all three built-in agents + one custom agent |
| Store migration issues | No schema changes - just code changes |
| Coordinator daemon instability | Incremental changes with tests at each step |

---

## Success Criteria

- [ ] All 8 existing tests in `task_chain_test.go` pass
- [ ] All 6 existing tests in `approval_watcher_test.go` pass
- [ ] New agents can be added to pipeline via config only (no code changes)
- [ ] GitHub workflow continues with existing labels (backwards compat)
- [ ] `TaskStage` enum has deprecation comments
- [ ] ~210 net LOC reduction (simplification)
- [ ] CLAUDE.md updated with config-driven approach

---

## Timeline

| Day | Milestones | Hours |
|-----|------------|-------|
| Day 1 | M1 (ApprovalConfig) + M2 (OnAgentComplete) | ~7h |
| Day 2 | M3 (Dynamic Registration) + M4 (HandleApproval) | ~6h |
| Day 3 | M5 (Deprecation) + Testing + Docs | ~6h |

**Total**: ~19 hours over 3 days

---

## Dependencies

- None (this is a prerequisite for M-CLOUD-INFRA)

---

## Related Documents

- [M-GENERIC-PIPELINE Design Doc](m-generic-pipeline.md)
- [M-CLOUD-INFRA Design Doc](m-cloud-infra.md) (depends on this sprint)
- [M-COORD-STABLE](../../implemented/v0_6_2/m-coord-stable.md) - Coordinator architecture

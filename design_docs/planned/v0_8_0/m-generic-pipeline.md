# M-GENERIC-PIPELINE: Config-Driven Agent Pipelines

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 (High)
**Estimated**: 2-3 days
**Dependencies**: None
**Author**: Claude + Mark
**Created**: 2026-01-16

---

## Executive Summary

Remove the hardcoded `TaskStage` pipeline (design → sprint → implementation → merge) and replace it with the generic `trigger_on_complete` configuration. This unifies two parallel systems into one, enabling flexible agent pipelines that can be modified via YAML without code changes.

---

## Problem Statement

### Current State: Two Parallel Systems

The codebase currently has **two different pipeline mechanisms**:

#### 1. Hardcoded TaskStage Pipeline (GitHub-linked tasks)

Located in `internal/coordinator/stage_execution.go` and `task_chain.go`:

```go
// store.go:69-78 - Hardcoded enum
type TaskStage string
const (
    TaskStageNone           TaskStage = ""
    TaskStageDesign         TaskStage = "design"         // → design-doc-creator
    TaskStageSprint         TaskStage = "sprint"         // → sprint-planner
    TaskStageImplementation TaskStage = "implementation" // → sprint-executor
    TaskStageMerge          TaskStage = "merge"
)

// stage_execution.go:61-72 - Hardcoded mapping
func stageToAgentIDForDirective(stage TaskStage) string {
    switch stage {
    case TaskStageDesign:
        return "design-doc-creator"  // HARDCODED
    case TaskStageSprint:
        return "sprint-planner"       // HARDCODED
    case TaskStageImplementation:
        return "sprint-executor"      // HARDCODED
    }
}
```

**Used when**: `task.GithubIssue > 0` (GitHub-linked tasks)

#### 2. Generic Config-Driven Pipeline

Located in `internal/coordinator/agent_registry.go` and `daemon_approval.go`:

```yaml
# ~/.ailang/config.yaml - Flexible configuration
coordinator:
  agents:
    - id: design-doc-creator
      trigger_on_complete: [sprint-planner]  # Next agent(s)
    - id: sprint-planner
      trigger_on_complete: [sprint-executor]
    - id: sprint-executor
      trigger_on_complete: []  # End of pipeline
```

**Used when**: Non-GitHub tasks, agent-to-agent handoffs

### Problems with Dual Systems

1. **Maintenance burden** - Two code paths for the same functionality
2. **No flexibility for GitHub tasks** - Cannot add/remove/reorder stages without code changes
3. **Hardcoded agent names** - Adding a new agent to GitHub pipeline requires deployment
4. **Cloud deployment complexity** - Hardcoded paths don't work well in containers
5. **Confusing for operators** - Which system applies to which tasks?

---

## Goals

### Primary Goal

Unify the two pipeline systems by **deprecating hardcoded TaskStage** and using **generic trigger_on_complete** for all workflows, including GitHub-linked tasks.

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Pipeline code paths | 2 (hardcoded + config) | 1 (config only) |
| Adding new agent to pipeline | Code change + deploy | YAML change only |
| GitHub workflow flexibility | None (fixed 4 stages) | Any number of stages |
| Stage-to-agent mapping | Hardcoded in Go | Config in YAML |

---

## Solution Design

### Unified Pipeline Model

Replace `TaskStage` enum with generic stage tracking via `agent_id`:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         UNIFIED PIPELINE MODEL                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  BEFORE (Hardcoded Stages)              AFTER (Config-Driven)               │
│  ┌────────────────────────┐            ┌────────────────────────┐           │
│  │ TaskStage = "design"   │            │ agent_id = "design-doc-│           │
│  │ → maps to agent via    │            │            creator"    │           │
│  │   stageToAgentID()     │            │                        │           │
│  │   (hardcoded)          │            │ trigger_on_complete:   │           │
│  └────────────────────────┘            │   [sprint-planner]     │           │
│                                        │   (from config)        │           │
│                                        └────────────────────────┘           │
│                                                                              │
│  GitHub workflow labels change from stage-based to agent-based:             │
│                                                                              │
│  BEFORE                                AFTER                                 │
│  • needs-design-approval               • needs-approval:design-doc-creator   │
│  • design-approved                     • approved:design-doc-creator         │
│  • needs-sprint-approval               • needs-approval:sprint-planner       │
│  • sprint-approved                     • approved:sprint-planner             │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Configuration Changes

#### Before: Implicit Stage-to-Agent Mapping

```yaml
# No config needed - hardcoded in Go
# design → design-doc-creator (implicit)
# sprint → sprint-planner (implicit)
# implementation → sprint-executor (implicit)
```

#### After: Explicit Pipeline Configuration

```yaml
# ~/.ailang/config.yaml
coordinator:
  # GitHub integration: which agent starts the pipeline
  github_sync:
    enabled: true
    target_inbox: design-doc-creator  # First agent in pipeline

  agents:
    - id: design-doc-creator
      inbox: design-doc-creator
      workspace: /path/to/ailang
      trigger_on_complete: [sprint-planner]

      # GitHub approval workflow (migrated from hardcoded ApprovalConfig)
      approval:
        needs_label: "needs-approval:design-doc-creator"
        approved_label: "approved:design-doc-creator"
        github_comment_template: "design_doc"

      # Invoke config (migrated from DefaultInvokeConfig)
      invoke:
        type: skill
        name: design-doc-creator

      # Artifact patterns (migrated from DefaultArtifactPatterns)
      artifact_patterns:
        - "design_docs/**/*.md"

    - id: sprint-planner
      inbox: sprint-planner
      workspace: /path/to/ailang
      trigger_on_complete: [sprint-executor]
      approval:
        needs_label: "needs-approval:sprint-planner"
        approved_label: "approved:sprint-planner"
        github_comment_template: "sprint_plan"
      invoke:
        type: skill
        name: sprint-planner

    - id: sprint-executor
      inbox: sprint-executor
      workspace: /path/to/ailang
      trigger_on_complete: []  # End of pipeline - merge only
      approval:
        needs_label: "needs-approval:sprint-executor"
        approved_label: "approved:sprint-executor"
        github_comment_template: "implementation"
      invoke:
        type: skill
        name: sprint-executor

    # NEW: Easy to add agents to pipeline!
    - id: code-reviewer
      inbox: code-reviewer
      workspace: /path/to/ailang
      trigger_on_complete: [sprint-executor]  # Insert between planner and executor
      approval:
        needs_label: "needs-approval:code-reviewer"
        approved_label: "approved:code-reviewer"
```

### Code Changes Required

#### Phase 1: Deprecate TaskStage Usage (~150 LOC changed)

**1. Update `store.go` - Keep TaskStage but mark deprecated**

```go
// TaskStage represents the pipeline stage for GitHub-linked tasks
// Deprecated: Use agent_id tracking instead. Will be removed in v0.9.0.
type TaskStage string

// ... keep constants for backwards compatibility during migration
```

**2. Update `daemon_approval.go:HandleApproval()` - Use agent config instead of stage**

```go
func (d *Daemon) HandleApproval(ctx context.Context, taskID, approvedBy string) error {
    task, err := d.taskStore.GetTask(ctx, taskID)
    // ...

    // BEFORE: Check stage to decide what to do
    // if task.Stage == TaskStageDesign { ... }

    // AFTER: Get agent config and check trigger_on_complete
    agent := d.agentRegistry.GetAgentByID(task.AgentID)
    if agent != nil && len(agent.TriggerOnComplete) > 0 {
        // Has handoff targets - create handoff approval
        return d.handleConfigDrivenApproval(ctx, task, agent, approvedBy)
    }

    // No handoff targets - just merge
    return d.handleMergeApproval(ctx, task, approvedBy)
}
```

**3. Update `task_chain.go` - Use agent config for labels/comments**

```go
// OnAgentComplete is the unified completion handler (replaces OnDesignDocComplete, etc.)
func (tc *TaskChain) OnAgentComplete(ctx context.Context, taskID string, agentID string, result *AgentResult) error {
    task, err := tc.store.GetTask(ctx, taskID)
    // ...

    // Get agent config for approval workflow
    agent := tc.registry.GetAgentByID(agentID)
    if agent == nil {
        return fmt.Errorf("agent %s not found in registry", agentID)
    }

    approval := agent.GetEffectiveApprovalConfig()
    if approval == nil {
        // No approval config - skip GitHub workflow
        return nil
    }

    // Post completion comment
    comment, err := RenderAgentCompleteComment(agentID, result)
    if err != nil {
        return err
    }

    if tc.poster != nil {
        if err := tc.poster.PostComment(task.GithubIssue, comment); err != nil {
            return err
        }
        // Add needs-approval label from config
        if err := tc.poster.AddLabel(task.GithubIssue, approval.NeedsLabel); err != nil {
            log.Printf("[TaskChain] Failed to add label: %v", err)
        }
    }

    return nil
}
```

**4. Update `approval_watcher.go` - Dynamic label detection**

```go
// RegisterAgentApprovalHandlers registers handlers for all configured agents
func (w *ApprovalWatcher) RegisterAgentApprovalHandlers(registry *AgentRegistry) {
    for _, agent := range registry.ListAgents() {
        approval := agent.GetEffectiveApprovalConfig()
        if approval == nil || approval.ApprovedLabel == "" {
            continue
        }

        // Dynamic handler for this agent's approval label
        w.RegisterLabelHandler(approval.ApprovedLabel, func(ctx context.Context, event *ApprovalEvent) error {
            return w.chain.OnAgentApproved(ctx, event, agent.ID)
        })
    }
}
```

#### Phase 2: Remove Stage-Based Code (~300 LOC removed)

**Files to modify:**

| File | Changes |
|------|---------|
| `stage_execution.go` | Remove `stageToAgentIDForDirective()`, update `BuildStageDirective()` to use agent_id only |
| `task_chain.go` | Remove stage-specific handlers (`OnDesignDocComplete`, `OnSprintPlanComplete`, etc.), replace with `OnAgentComplete()` |
| `daemon_approval.go` | Remove stage switch in `HandleApproval()`, use config-driven approach |
| `approval_watcher.go` | Remove hardcoded label constants (`LabelNeedsDesignApproval`, etc.) |
| `store.go` | Mark `TaskStage` as deprecated, keep for backwards compatibility |

**Constants to deprecate (keep but mark deprecated):**

```go
// internal/coordinator/github_comments.go

// Deprecated: Use agent.Approval.NeedsLabel from config instead
const (
    LabelNeedsDesignApproval = "needs-design-approval"
    LabelDesignApproved      = "design-approved"
    // ...
)
```

#### Phase 3: Update Store Interface (~50 LOC)

**Keep stage methods but mark deprecated:**

```go
// Store interface changes

// Deprecated: Use SetTaskAgentID instead. Stage is now tracked via agent_id.
SetTaskStage(ctx context.Context, id string, stage TaskStage) error

// Deprecated: Use GetTasksByAgentID instead.
GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error)

// NEW: Track agent progression
SetTaskAgentID(ctx context.Context, id string, agentID string) error
GetTasksByAgentID(ctx context.Context, agentID string) ([]*TaskRecord, error)
```

### Migration Path

#### Backwards Compatibility

1. **Existing tasks with Stage field** - Continue to work, stage is mapped to agent_id at runtime
2. **Existing GitHub labels** - Keep working via `DefaultApprovalConfig()` fallback
3. **Config without explicit approval** - Falls back to legacy defaults

#### Migration Timeline

| Version | Changes |
|---------|---------|
| v0.8.0 | Add config-driven pipeline, deprecate TaskStage |
| v0.8.1 | Remove TaskStage from new task creation (use agent_id only) |
| v0.9.0 | Remove TaskStage enum and related code entirely |

### GitHub Label Transition

**Option 1: Keep existing labels (recommended for v0.8.0)**

```go
// DefaultApprovalConfig returns legacy labels for backwards compatibility
func DefaultApprovalConfig(agentID string) *ApprovalConfig {
    switch agentID {
    case "design-doc-creator":
        return &ApprovalConfig{
            NeedsLabel:    "needs-design-approval",  // Legacy label
            ApprovedLabel: "design-approved",
        }
    // ...
    }
}
```

**Option 2: Migrate to agent-based labels (v0.9.0)**

```yaml
# New label scheme
approval:
  needs_label: "needs-approval:design-doc-creator"
  approved_label: "approved:design-doc-creator"
```

---

## Files to Create/Modify

### Modified Files

| File | LOC Changed | Changes |
|------|-------------|---------|
| `internal/coordinator/stage_execution.go` | ~-100 | Remove `stageToAgentIDForDirective()`, update `BuildStageDirective()` |
| `internal/coordinator/task_chain.go` | ~-200 | Replace stage handlers with `OnAgentComplete()`, `OnAgentApproved()` |
| `internal/coordinator/daemon_approval.go` | ~-80 | Remove stage switch, use `trigger_on_complete` |
| `internal/coordinator/approval_watcher.go` | ~+50 | Add `RegisterAgentApprovalHandlers()` |
| `internal/coordinator/store.go` | ~+10 | Deprecation comments |
| `internal/coordinator/store_sqlite.go` | ~+20 | Add `SetTaskAgentID`, `GetTasksByAgentID` |
| `internal/coordinator/agent_registry.go` | ~0 | No changes (already has the right structure) |
| **Total** | ~-300 net | Simplification |

### Tests to Update

| File | Changes |
|------|---------|
| `stage_execution_test.go` | Update tests for agent-based approach |
| `task_chain_test.go` | Update tests for unified handlers |
| `workflow_test.go` | Update E2E tests |
| `e2e_workflow_test.go` | Verify backwards compatibility |

---

## Implementation Plan

### Day 1: Core Refactoring

- [ ] **M1**: Add `SetTaskAgentID()` and `GetTasksByAgentID()` to Store interface
- [ ] **M2**: Update `BuildStageDirective()` to always use agent_id, remove stage mapping
- [ ] **M3**: Create `OnAgentComplete()` unified handler in task_chain.go
- [ ] **M4**: Update `HandleApproval()` to use trigger_on_complete

### Day 2: Approval Workflow Migration

- [ ] **M5**: Create `RegisterAgentApprovalHandlers()` in approval_watcher.go
- [ ] **M6**: Update GitHub label handling to use config-driven labels
- [ ] **M7**: Add deprecation warnings for TaskStage usage
- [ ] **M8**: Update tests

### Day 3: Cleanup and Documentation

- [ ] **M9**: Remove dead code (stage switch statements, unused handlers)
- [ ] **M10**: Update CLAUDE.md documentation
- [ ] **M11**: Add migration guide to docs/
- [ ] **M12**: Test E2E workflows

---

## Axiom Compliance

| Axiom | Score | Notes |
|-------|-------|-------|
| A1: Determinism | +1 | Config defines exact pipeline behavior |
| A3: Effect Legibility | +1 | Pipeline defined in YAML, not hidden in code |
| A7: Machines First | +1 | No hardcoded strings, all config-driven |
| A10: Composability | +1 | Agents can be composed into any pipeline |
| A12: System Boundary | +1 | Clear separation: config = what, code = how |
| **Net Score** | **+5** | Improvement over dual systems |

---

## Success Criteria

- [ ] All existing tests pass
- [ ] GitHub workflow continues to work with existing labels
- [ ] New agents can be added to pipeline via config only
- [ ] No hardcoded agent names in approval workflow code
- [ ] `TaskStage` enum has deprecation comments
- [ ] CLAUDE.md updated to document config-driven approach

---

## Open Questions

1. **Label migration**: Should we migrate GitHub labels to agent-based scheme in v0.8.0 or v0.9.0?
   - **Recommendation**: Keep legacy labels in v0.8.0 via defaults, migrate in v0.9.0

2. **Pipeline visualization**: Should the dashboard show the configured pipeline?
   - **Recommendation**: Yes, add to Control Plane in v0.8.1

3. **Validation**: Should we validate trigger_on_complete references exist?
   - **Recommendation**: Yes, already in `AgentRegistry.Validate()`

---

## Related Documents

- [M-CLOUD-INFRA](m-cloud-infra.md) - Cloud deployment (uses generic pipeline)
- [M-COORD-STABLE](../../implemented/v0_6_2/m-coord-stable.md) - Coordinator architecture
- [agent_registry.go](../../../internal/coordinator/agent_registry.go) - Generic agent config

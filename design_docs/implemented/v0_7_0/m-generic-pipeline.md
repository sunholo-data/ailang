# M-GENERIC-PIPELINE: Config-Driven Agent Pipelines

**Status**: Implemented
**Version**: v0.8.0
**Priority**: P1 (High)
**Actual**: 1 day (vs 2-3 estimated)
**Dependencies**: None
**Author**: Claude + Mark
**Created**: 2026-01-16
**Implemented**: 2026-01-18

---

## Executive Summary

Removed the hardcoded `TaskStage` pipeline (design → sprint → implementation → merge) and unified with the generic `trigger_on_complete` configuration. This enables flexible agent pipelines that can be modified via YAML without code changes.

---

## Implementation Summary

### What Was Done

1. **Removed hardcoded label constants** from `approval_watcher.go`
   - Kept only `LabelNeedsRevision` as universal
   - All other labels now come from `AgentConfig.Approval` or defaults

2. **Removed legacy stage-based switch** from `daemon_approval.go`
   - `HandleApproval()` now uses `GetEffectiveApprovalConfig()` for label operations
   - Falls through to merge for agents without `trigger_on_complete`

3. **Updated handlers to use config-driven labels** in `task_chain.go`
   - `OnDesignDocComplete()` → uses `DefaultApprovalConfig("design-doc-creator")`
   - `OnSprintPlanComplete()` → uses `DefaultApprovalConfig("sprint-planner")`
   - `OnImplementationComplete()` → uses `DefaultApprovalConfig("sprint-executor")`

4. **Added unified handlers** in `task_chain.go`
   - `OnAgentComplete()` - Generic completion handler for any agent
   - `OnAgentApproved()` - Config-driven approval with `trigger_on_complete` handoffs
   - `AgentResult` - Unified result struct

5. **Added dynamic registration** in `approval_watcher.go`
   - `RegisterAgentApprovalHandlers()` - Registers all agents from registry
   - Config-driven `needsLabel` removal in `handleEvent()`

6. **Added GitHub label handling** to `approval_processor.go`
   - CLI approvals now also update GitHub labels via config

7. **Inlined stage-to-agent mapping** in `stage_execution.go`
   - Removed `stageToAgentIDForDirective()` function
   - Legacy fallback for old tasks still works

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `approval_watcher.go` | Removed 6 label constants, added `RegisterAgentApprovalHandlers()` | -20 |
| `daemon_approval.go` | Removed legacy stage switch | -35 |
| `task_chain.go` | Added unified handlers, updated existing handlers | +200 |
| `task_chain_test.go` | Added 7 tests for unified handlers | +90 |
| `approval_watcher_test.go` | Added 5 tests for registration | +60 |
| `stage_execution.go` | Inlined stage mapping, removed function | -10 |
| `approval_processor.go` | Added config-driven label handling | +30 |
| `e2e_workflow_test.go` | Updated to use `BuildStageDirective()` | +5 |
| `CLAUDE.md` | Added approval configuration docs | +30 |
| **Total** | Net simplification + new capability | ~+350 |

### E2E Verification

Tested with GitHub issue #124:
1. Created task with `agent_id=design-doc-creator`
2. Approved via `ailang coordinator approve`
3. Labels updated correctly: `needs-design-approval` → `design-approved`

---

## Configuration

### Agent Approval Configuration

```yaml
coordinator:
  agents:
    - id: custom-agent
      inbox: custom-agent
      workspace: /path/to/project
      approval:
        needs_label: needs-custom-approval    # Added when work complete
        approved_label: custom-approved       # Human adds to approve
        github_comment_template: generic      # Comment template (optional)
      trigger_on_complete: [next-agent]       # Handoff chain
```

### Default Labels for Known Agents

| Agent ID | needs_label | approved_label |
|----------|-------------|----------------|
| `design-doc-creator` | `needs-design-approval` | `design-approved` |
| `sprint-planner` | `needs-sprint-approval` | `sprint-approved` |
| `sprint-executor` | `needs-merge-approval` | `merge-approved` |

These defaults are provided by `DefaultApprovalConfig()` and `GetEffectiveApprovalConfig()`.

---

## Backwards Compatibility

### Preserved

- Existing tasks with `Stage` field continue to work (mapped to agent_id at runtime)
- Existing GitHub labels continue to work via `DefaultApprovalConfig()` fallback
- Config without explicit `approval` section falls back to defaults for known agents

### Deprecated (Removal in v0.9.0)

- `TaskStage` enum - marked with deprecation comments
- Direct use of stage-based workflow - use `trigger_on_complete` instead

---

## Key Functions

### GetEffectiveApprovalConfig()

```go
func (c *AgentConfig) GetEffectiveApprovalConfig() *ApprovalConfig {
    if c.Approval != nil {
        return c.Approval
    }
    return DefaultApprovalConfig(c.ID)
}
```

Returns explicit config or defaults for known agents (`design-doc-creator`, `sprint-planner`, `sprint-executor`).

### OnAgentComplete()

```go
func (tc *TaskChain) OnAgentComplete(ctx context.Context, taskID, agentID string,
    result *AgentResult, registry *AgentRegistry) error
```

Unified handler that:
1. Gets effective approval config for agent
2. Stores artifact paths based on agent type
3. Posts GitHub comment and adds needs-approval label
4. Updates task to pending_approval status

### OnAgentApproved()

```go
func (tc *TaskChain) OnAgentApproved(ctx context.Context, event *ApprovalEvent,
    agentID string) error
```

Config-driven approval that:
1. Looks up agent's `trigger_on_complete` targets
2. Sends handoff messages to target agents
3. Requeues task for next stage

---

## Success Criteria - All Met

- [x] All existing tests pass (12.077s)
- [x] GitHub workflow continues to work with existing labels
- [x] New agents can be added to pipeline via config only
- [x] No hardcoded agent names in approval workflow code (only in defaults)
- [x] `TaskStage` enum has deprecation comments
- [x] CLAUDE.md updated to document config-driven approach

---

## Related Documents

- [M-CLOUD-INFRA](m-cloud-infra.md) - Cloud deployment (uses generic pipeline)
- [Sprint Plan](../../planned/v0_8_0/m-generic-pipeline-sprint-plan.md) - Sprint execution details

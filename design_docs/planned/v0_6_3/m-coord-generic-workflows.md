# M-COORD-GENERIC-WORKFLOWS: Configurable Agent Workflows

## Status
**Status**: Planned
**Target Version**: v0.6.3
**Priority**: P1 (Enables multi-project support)
**Created**: 2026-01-01

## Problem Statement

The coordinator currently has hardcoded workflow stages and skill invocations:

```go
// stage_execution.go - HARDCODED skill names
func buildDesignDirective(task *TaskRecord) string {
    return `Invoke the design-doc-creator skill...`
}
```

This works for AILANG development but **cannot be reused for other projects** with different:
- Skills (e.g., a game project has `level-designer`, `asset-creator` skills)
- Workflow stages (e.g., 2-stage vs 4-stage pipelines)
- Approval gates (e.g., auto-approve design, manual approve implementation)

### Current Architecture

```
~/.ailang/config.yaml
├── coordinator.agents[]
│   ├── workspace: /path/to/project   ← Claude Code runs here
│   ├── trigger_on_complete: [next]   ← Hardcoded stage transitions
│   └── (NO skill/agent reference)    ← MISSING!

.claude/skills/                        ← Auto-discovered by Claude Code
├── design-doc-creator/
├── sprint-planner/
└── sprint-executor/

.claude/agents/                        ← NOT USED by coordinator
└── dev-cycle.md                       ← Full workflow orchestrator
```

**The Gap**: Config specifies `workspace` but not which skill/agent to invoke. The coordinator hardcodes "Invoke the design-doc-creator skill" text.

### Impact

- Cannot use coordinator for non-AILANG projects
- Cannot customize workflow stages without code changes
- Cannot define project-specific approval rules
- Gemini CLI users get same hardcoded AILANG skill references (won't work)

## Goals

### Primary Goal
Make the coordinator workflow fully configurable so different projects can define their own skills, stages, and approval rules.

### Success Metrics
- [ ] Different projects can define custom workflow stages
- [ ] Skill/agent invocation is configurable per agent
- [ ] No AILANG-specific hardcoding in coordinator code
- [ ] Works with both Claude Code and Gemini CLI

## Solution Design

### Summary: The Complete Generic Workflow System

The solution adds three new config sections to each agent, making the coordinator fully configurable:

```yaml
coordinator:
  agents:
    - id: <agent-id>
      workspace: <project-directory>       # Claude Code runs here, auto-discovers skills

      # NEW 1: What to invoke
      invoke:
        type: skill | agent | prompt       # How to invoke
        name: <skill-or-agent-name>        # What to invoke

      # NEW 2: How to detect completion
      output_markers: ["DESIGN_DOC_PATH:"] # Markers to parse from output

      # NEW 3: How to handle approvals
      approval:
        needs_label: "needs-approval"      # Label added when complete
        approved_label: "approved"         # Label that triggers next stage
        github_comment_template: "design"  # Template for GitHub comments

      # EXISTING: Pipeline control
      trigger_on_complete: [next-agent]    # Which agent(s) to trigger next
      auto_approve_handoffs: false         # Skip approval labels entirely?
      auto_merge: false                    # Auto-merge at end of pipeline?
```

**Key benefits:**
- Different projects define their own skills/stages/approvals
- No AILANG-specific code in coordinator
- Works with Claude Code, Gemini CLI, or any executor
- Full human-in-the-loop via GitHub labels when needed
- Automatic flow via `auto_approve_handoffs: true` when not

### Key Insight: Let Skills Do the Work

Claude Code (and Gemini CLI) auto-discover skills from `<workspace>/.claude/skills/`. Instead of hardcoding skill invocations, the coordinator should:

1. **Reference skills by name in config** (not code)
2. **Build simple directive templates** that invoke skills
3. **Let the skill handle the details** (prompts, outputs, etc.)

### Proposed Config Changes

```yaml
coordinator:
  agents:
    - id: design-doc-creator
      workspace: /Users/mark/dev/sunholo/ailang

      # NEW: Skill or agent to invoke
      invoke:
        type: skill           # "skill" or "agent" or "prompt"
        name: design-doc-creator

      # NEW: Output markers this skill produces (for stage completion)
      output_markers:
        - "DESIGN_DOC_PATH:"

      # Existing: Next agent(s) to trigger
      trigger_on_complete: [sprint-planner]

      # Existing: Approval settings
      auto_approve_handoffs: false
      auto_merge: false
```

### Invoke Types

#### Type 1: `skill` - Invoke a Claude Code skill
```yaml
invoke:
  type: skill
  name: design-doc-creator
```
Generates directive: `Run /design-doc-creator for: {task.Content}`

#### Type 2: `agent` - Run a .claude/agents/*.md file
```yaml
invoke:
  type: agent
  name: dev-cycle
```
Generates directive: `Follow the dev-cycle agent workflow for: {task.Content}`

#### Type 3: `prompt` - Custom directive template
```yaml
invoke:
  type: prompt
  template: |
    Create a design document for:
    {{.Content}}

    Save to design_docs/planned/ and output DESIGN_DOC_PATH when done.
```

### Stage Transition via Output Markers

Instead of hardcoding what each stage outputs, make it configurable:

```yaml
- id: design-doc-creator
  output_markers: ["DESIGN_DOC_PATH:"]
  trigger_on_complete: [sprint-planner]

- id: sprint-planner
  output_markers: ["SPRINT_PLAN_PATH:"]
  trigger_on_complete: [sprint-executor]

- id: sprint-executor
  output_markers: ["IMPLEMENTATION_COMPLETE:"]
  trigger_on_complete: []  # End of pipeline
```

The coordinator:
1. Parses output for markers
2. Extracts values (e.g., `DESIGN_DOC_PATH: design_docs/planned/v0_6_3/foo.md`)
3. Passes extracted values to next stage

### Example: AILANG Project Config

```yaml
coordinator:
  default_provider: claude

  agents:
    - id: design-doc-creator
      label: "Design Doc Creator"
      inbox: design-doc-creator
      workspace: /Users/mark/dev/sunholo/ailang
      provider: claude
      invoke:
        type: skill
        name: design-doc-creator
      output_markers: ["DESIGN_DOC_PATH:"]
      trigger_on_complete: [sprint-planner]
      auto_approve_handoffs: false

    - id: sprint-planner
      label: "Sprint Planner"
      inbox: sprint-planner
      workspace: /Users/mark/dev/sunholo/ailang
      provider: claude
      invoke:
        type: skill
        name: sprint-planner
      output_markers: ["SPRINT_PLAN_PATH:", "SPRINT_JSON_PATH:"]
      trigger_on_complete: [sprint-executor]
      auto_approve_handoffs: false

    - id: sprint-executor
      label: "Sprint Executor"
      inbox: sprint-executor
      workspace: /Users/mark/dev/sunholo/ailang
      provider: claude
      invoke:
        type: skill
        name: sprint-executor
      output_markers: ["IMPLEMENTATION_COMPLETE:"]
      trigger_on_complete: []
      auto_merge: false
```

### Example: Game Project Config

```yaml
# ~/.config/game-project/coordinator.yaml
coordinator:
  agents:
    - id: level-designer
      workspace: /home/user/game-project
      invoke:
        type: skill
        name: level-designer
      output_markers: ["LEVEL_JSON:"]
      trigger_on_complete: [asset-creator]
      auto_approve_handoffs: true  # Auto-approve level designs

    - id: asset-creator
      workspace: /home/user/game-project
      invoke:
        type: skill
        name: asset-creator
      output_markers: ["ASSETS_CREATED:"]
      trigger_on_complete: [playtester]
      auto_approve_handoffs: true

    - id: playtester
      workspace: /home/user/game-project
      invoke:
        type: prompt
        template: |
          Run the game and test level: {{.Content}}
          Report bugs found.
      output_markers: ["PLAYTEST_COMPLETE:"]
      trigger_on_complete: []
      auto_merge: false
```

## Implementation Plan

### Phase 1: Add `invoke` Config (~100 LOC)

**Files to modify:**
- `internal/coordinator/agent_registry.go` - Add InvokeConfig struct
- `internal/coordinator/agent_config.go` - Parse invoke config from YAML

```go
// agent_registry.go
type InvokeConfig struct {
    Type     string `yaml:"type"`     // "skill", "agent", or "prompt"
    Name     string `yaml:"name"`     // Skill/agent name
    Template string `yaml:"template"` // Custom template for type="prompt"
}

type AgentConfig struct {
    // ... existing fields ...
    Invoke        *InvokeConfig `yaml:"invoke"`
    OutputMarkers []string      `yaml:"output_markers"`
}
```

### Phase 2: Replace Hardcoded Directives (~80 LOC)

**Files to modify:**
- `internal/coordinator/stage_execution.go` - Use config instead of hardcoded strings

```go
// BuildStageDirective - now uses config
func BuildStageDirective(task *TaskRecord, agent *AgentConfig) string {
    if agent.Invoke == nil {
        // Fallback to raw content
        return task.Content
    }

    switch agent.Invoke.Type {
    case "skill":
        return fmt.Sprintf("Run /%s for:\n%s", agent.Invoke.Name, task.Content)
    case "agent":
        return fmt.Sprintf("Follow the %s agent for:\n%s", agent.Invoke.Name, task.Content)
    case "prompt":
        return expandTemplate(agent.Invoke.Template, task)
    default:
        return task.Content
    }
}
```

### Phase 3: Generic Output Marker Parsing (~60 LOC)

**Files to modify:**
- `internal/coordinator/stage_execution.go` - Parse any configured markers

```go
// ParseStageOutput - now uses configured markers
func ParseStageOutput(output string, markers []string) map[string]string {
    results := make(map[string]string)
    for _, marker := range markers {
        if value := extractMarkerValue(output, marker); value != "" {
            results[marker] = value
        }
    }
    return results
}
```

### Phase 4: Remove Hardcoded Stage Logic (~-150 LOC)

**Files to modify:**
- `internal/coordinator/stage_execution.go` - Remove buildDesignDirective etc.
- `internal/coordinator/task_chain.go` - Use generic stage transitions

### Phase 5: Update Default Config (~50 LOC)

Update `DefaultCoordinatorConfig()` to include invoke settings for backwards compatibility.

## Migration Path

### Backwards Compatibility

For existing users without `invoke` config:
1. If no `invoke` specified, use legacy hardcoded behavior
2. Log deprecation warning
3. Remove legacy behavior in v0.7.0

### Migration Steps

1. **No action required** - Existing configs continue to work
2. **Optional** - Add `invoke` settings for explicit control
3. **v0.7.0** - Legacy behavior removed, `invoke` required

## Testing

### Unit Tests
- Parse invoke config from YAML
- Build directives for each invoke type
- Extract markers from output

### Integration Tests
- Full pipeline with skill invocation
- Custom project config with different stages

### E2E Tests
- AILANG workflow with new config format
- Verify backwards compatibility

## Acceptance Criteria

### Invoke Configuration
- [ ] `invoke` config field added to AgentConfig
- [ ] Three invoke types supported: skill, agent, prompt
- [ ] Output markers configurable per agent
- [ ] Hardcoded AILANG skill names removed from code

### Approval Configuration
- [ ] `approval` config field added to AgentConfig
- [ ] Configurable `needs_label` and `approved_label` per agent
- [ ] Configurable GitHub comment templates
- [ ] Hardcoded label constants removed from code
- [ ] ApprovalWatcher uses config-driven label detection
- [ ] TaskChain uses generic stage completion handlers

### Integration
- [ ] `auto_approve_handoffs` skips label workflow correctly
- [ ] `trigger_on_complete` works with any agent IDs
- [ ] Multi-stage pipelines work without code changes

### Backwards Compatibility
- [ ] Default config includes invoke and approval settings
- [ ] Existing AILANG config continues to work
- [ ] Deprecation warnings for missing invoke/approval config

### Documentation
- [ ] Updated config.yaml examples
- [ ] Migration guide for existing configs

## Configurable Approval Flow

### Current Architecture Problem

The approval system has hardcoded labels and stage mappings:

```go
// approval_watcher.go - HARDCODED labels
const (
    LabelNeedsDesignApproval = "needs-design-approval"
    LabelDesignApproved      = "design-approved"
    // ... more hardcoded labels
)

// task_chain.go - HARDCODED handler registration
watcher.RegisterHandler(ApprovalEventDesign, tc.OnDesignApproved)
watcher.RegisterHandler(ApprovalEventSprint, tc.OnSprintApproved)
```

This creates a tight coupling between code and workflow stages. A game project with `level-designer → asset-creator → playtester` cannot use the current system.

### Proposed: Per-Agent Approval Configuration

Add approval settings to each agent config:

```yaml
coordinator:
  agents:
    - id: design-doc-creator
      workspace: /Users/mark/dev/sunholo/ailang
      invoke:
        type: skill
        name: design-doc-creator
      output_markers: ["DESIGN_DOC_PATH:"]

      # NEW: Approval workflow configuration
      approval:
        needs_label: "needs-design-approval"    # Label added when stage completes
        approved_label: "design-approved"       # Label that triggers next stage
        github_comment_template: "design_doc"   # Template for GitHub comment

      trigger_on_complete: [sprint-planner]
      auto_approve_handoffs: false  # If true, skip approval labels entirely

    - id: sprint-planner
      approval:
        needs_label: "needs-sprint-approval"
        approved_label: "sprint-approved"
        github_comment_template: "sprint_plan"
      trigger_on_complete: [sprint-executor]
      auto_approve_handoffs: false

    - id: sprint-executor
      approval:
        needs_label: "needs-merge-approval"
        approved_label: "merge-approved"
        github_comment_template: "implementation"
      trigger_on_complete: []  # End of pipeline
      auto_merge: false
```

### Example: Game Project with Custom Approvals

```yaml
coordinator:
  agents:
    - id: level-designer
      workspace: /home/user/game-project
      invoke:
        type: skill
        name: level-designer
      output_markers: ["LEVEL_JSON:"]
      approval:
        needs_label: "level-needs-review"
        approved_label: "level-approved"
      trigger_on_complete: [asset-creator]
      auto_approve_handoffs: true  # Auto-approve level designs

    - id: asset-creator
      workspace: /home/user/game-project
      invoke:
        type: skill
        name: asset-creator
      output_markers: ["ASSETS_CREATED:"]
      approval:
        needs_label: "assets-need-review"
        approved_label: "assets-approved"
      trigger_on_complete: [playtester]
      auto_approve_handoffs: true  # Auto-approve assets

    - id: playtester
      workspace: /home/user/game-project
      invoke:
        type: prompt
        template: |
          Run the game and test level: {{.Content}}
          Report bugs found.
      output_markers: ["PLAYTEST_COMPLETE:"]
      approval:
        needs_label: "playtest-review"
        approved_label: "ready-to-ship"
      trigger_on_complete: []
      auto_merge: false  # Manual approval for shipping
```

### How Approval Flow Connects to Stages

The generic stage transition works as follows:

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  1. Agent Executes                                             │
│     ├── BuildStageDirective() uses agent.invoke config         │
│     └── Executor runs skill/agent/prompt                       │
│                                                                 │
│  2. Output Parsed                                               │
│     ├── ParseStageOutput() uses agent.output_markers           │
│     └── Extracts paths/artifacts from output                   │
│                                                                 │
│  3. Stage Complete Callback                                     │
│     ├── Post GitHub comment (agent.approval.github_comment_template)
│     └── Add needs_label (agent.approval.needs_label)           │
│                                                                 │
│  4. If auto_approve_handoffs: true                              │
│     ├── Skip GitHub labels entirely                            │
│     └── Immediately trigger next agent                         │
│                                                                 │
│  5. If auto_approve_handoffs: false                             │
│     ├── ApprovalWatcher polls GitHub                           │
│     └── When approved_label detected:                          │
│         ├── Remove both labels                                 │
│         └── Trigger agents in trigger_on_complete              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation: Generic ApprovalWatcher

Instead of hardcoded label constants, ApprovalWatcher becomes config-driven:

```go
// approval_watcher.go - NOW CONFIG-DRIVEN
type ApprovalWatcher struct {
    // ... existing fields ...

    // NEW: Map from approved_label -> AgentConfig
    approvalLabels map[string]*AgentConfig
}

// RegisterAgentApproval registers an agent's approval labels
func (w *ApprovalWatcher) RegisterAgentApproval(agent *AgentConfig) {
    if agent.Approval != nil && agent.Approval.ApprovedLabel != "" {
        w.approvalLabels[agent.Approval.ApprovedLabel] = agent
    }
}

// checkIssueLabels - now uses config, not hardcoded labels
func (w *ApprovalWatcher) checkIssueLabels(ctx context.Context, issueNum int, taskID string) *ApprovalEvent {
    labels, _ := w.poster.GetLabels(issueNum)

    for _, label := range labels {
        // Check for needs-revision (still hardcoded, universal concept)
        if label == "needs-revision" {
            return &ApprovalEvent{EventType: ApprovalEventRevision, ...}
        }

        // Check configured approval labels
        if agent, ok := w.approvalLabels[label]; ok {
            return &ApprovalEvent{
                EventType:   ApprovalEventType(label),  // Use label as event type
                AgentConfig: agent,                      // Include the config
                // ...
            }
        }
    }
    return nil
}
```

### Implementation: Generic TaskChain

TaskChain becomes a generic pipeline manager:

```go
// task_chain.go - NOW CONFIG-DRIVEN
func (tc *TaskChain) OnAgentComplete(ctx context.Context, agent *AgentConfig, taskID string, result *StageExecutionResult) error {
    task, _ := tc.store.GetTask(ctx, taskID)

    if agent.Approval != nil && task.GithubIssue > 0 {
        // Post comment using template
        comment := tc.renderComment(agent.Approval.GithubCommentTemplate, result)
        tc.poster.PostComment(task.GithubIssue, comment)

        // Add needs label
        tc.poster.AddLabel(task.GithubIssue, agent.Approval.NeedsLabel)
    }

    // If auto-approve, trigger next stage immediately
    if agent.AutoApproveHandoffs {
        return tc.TriggerNextAgents(ctx, agent, taskID)
    }

    // Otherwise, wait for ApprovalWatcher to detect approval label
    return nil
}

func (tc *TaskChain) OnApprovalDetected(ctx context.Context, event *ApprovalEvent) error {
    agent := event.AgentConfig

    // Trigger all agents in trigger_on_complete
    return tc.TriggerNextAgents(ctx, agent, event.TaskID)
}

func (tc *TaskChain) TriggerNextAgents(ctx context.Context, currentAgent *AgentConfig, taskID string) error {
    for _, nextAgentID := range currentAgent.TriggerOnComplete {
        // Requeue task for next agent
        tc.store.SetTaskAgent(ctx, taskID, nextAgentID)
        tc.store.RequeueTask(ctx, taskID)
    }
    return nil
}
```

### Backwards Compatibility

For existing users without approval config:

```go
// DefaultApprovalConfig returns legacy labels for agent ID
func DefaultApprovalConfig(agentID string) *ApprovalConfig {
    switch agentID {
    case "design-doc-creator":
        return &ApprovalConfig{
            NeedsLabel:            "needs-design-approval",
            ApprovedLabel:         "design-approved",
            GithubCommentTemplate: "design_doc",
        }
    case "sprint-planner":
        return &ApprovalConfig{...}
    case "sprint-executor":
        return &ApprovalConfig{...}
    default:
        // Unknown agent - no approval workflow
        return nil
    }
}
```

### Updated Implementation Plan

#### Phase 6: Configurable Approval (~150 LOC)

**Files to modify:**
- `internal/coordinator/agent_registry.go` - Add ApprovalConfig struct
- `internal/coordinator/approval_watcher.go` - Make label detection config-driven
- `internal/coordinator/task_chain.go` - Generic stage completion handlers

```go
// agent_registry.go
type ApprovalConfig struct {
    NeedsLabel            string `yaml:"needs_label"`
    ApprovedLabel         string `yaml:"approved_label"`
    GithubCommentTemplate string `yaml:"github_comment_template"`
}

type AgentConfig struct {
    // ... existing fields ...
    Invoke        *InvokeConfig   `yaml:"invoke"`
    OutputMarkers []string        `yaml:"output_markers"`
    Approval      *ApprovalConfig `yaml:"approval"`  // NEW
}
```

#### Phase 7: Comment Templates (~100 LOC)

**Files to modify:**
- `internal/coordinator/templates.go` - Make templates configurable

Allow projects to define their own GitHub comment templates:

```yaml
coordinator:
  comment_templates:
    design_doc: |
      ## Design Document Created

      **Path**: {{.DesignDocPath}}
      **Duration**: {{.Duration}}

      Please review and add `design-approved` label to proceed.

    # Custom template for game project
    level_design: |
      ## Level Design Complete

      **Level JSON**: {{.LevelJson}}

      Review in-game and add `level-approved` to proceed.
```

## Open Questions

1. **Multi-workspace coordinator?** Should one coordinator manage multiple project workspaces, or one coordinator per project?

2. **Skill discovery validation?** Should coordinator validate that referenced skills exist in workspace?

3. **Agent vs Skill distinction?** Are `.claude/agents/` files useful for coordinator, or should everything use skills?

4. **Template inheritance?** Should there be base templates that projects can extend?

5. **Revision handling?** Should `needs-revision` remain universal, or be configurable per-agent?

## Related Documents

- [M-COORD-APPROVALWATCHER-OBSERVABILITY](../implemented/v0_6_2/m-coord-approvalwatcher-observability.md) - Just completed
- [dev-cycle.md](../../../.claude/agents/dev-cycle.md) - Example agent orchestrator
- [Agent Registry](../../../internal/coordinator/agent_registry.go) - Current config structure

# Setting Up the Coordinator for External Projects

This guide explains how to configure the AILANG coordinator to manage agents for external repositories. The coordinator can run agents across multiple projects, enabling automated task delegation and approval workflows.

## Overview

The coordinator daemon watches message inboxes and executes tasks using AI agents (Claude Code or Gemini CLI). Each agent is configured with:

- **Workspace**: The repository where the agent operates
- **Inbox**: Message queue the agent monitors
- **Skills**: Claude Code skills the agent can invoke
- **Workflow chains**: Automatic handoffs between agents

## Prerequisites

1. AILANG installed (`ailang version` >= v0.6.6)
2. Global config file exists: `~/.ailang/config.yaml`
3. Target repository has Claude Code skills in `.claude/skills/`
4. Claude Code installed and configured

## Step 1: Identify Skills to Convert

Not all skills are suitable for coordinator agents. Review your project's skills:

| Skill Type | Agent Suitability | Notes |
|------------|-------------------|-------|
| **Workflow** (design → plan → implement) | Best fit | Core automation pipeline |
| **Validation/testing** | Good fit | Pre-release checks, architecture validation |
| **Interactive** (Q&A, interviews) | May need human trigger | Works but less autonomous |
| **Data/asset download** | On-demand | Trigger manually when needed |
| **Pure CLI tools** | Skip | Better as direct commands |

## Step 2: Agent Configuration

Add agents to `~/.ailang/config.yaml` under `coordinator.agents`.

### Basic Agent Template

```yaml
coordinator:
  agents:
    - id: my-project-design-doc        # Unique agent ID
      label: "My Project Design Doc"   # Human-readable name
      inbox: my-project-design-doc     # Message inbox to watch
      workspace: /path/to/my-project   # ABSOLUTE path to repo
      merge_branch: main               # Target branch (IMPORTANT: set this if not 'dev')
      capabilities: [research, docs]   # Agent capabilities
      provider: claude                 # "claude" or "gemini"
      trigger_on_complete: [my-project-sprint-planner]  # Next agent in chain
      auto_approve_handoffs: false     # Require approval between agents
      auto_merge: false                # Require approval before merge
      session_continuity: true         # Use --resume for context
      max_concurrent_tasks: 1
      invoke:
        type: skill
        name: design-doc-creator       # Skill name in .claude/skills/
      output_markers:                  # Capture these from output
        - "DESIGN_DOC_PATH:"
      artifact_patterns:               # Files to include in diffs
        - "design_docs/**/*.md"
```

**Important: merge_branch Setting**

The `merge_branch` field specifies which branch worktrees are created from. The coordinator defaults to `dev`, so if your repository uses a different default branch (like `main`), you **must** set `merge_branch` explicitly for each agent.

| Your repo default branch | Set merge_branch to |
|--------------------------|---------------------|
| `dev` | (optional - this is the default) |
| `main` | `merge_branch: main` |
| `master` | `merge_branch: master` |

### Workflow Chains

Use `trigger_on_complete` to create agent chains:

```yaml
agents:
  # Stage 1: Create design doc
  - id: proj-design-doc
    trigger_on_complete: [proj-sprint-planner]
    # ...

  # Stage 2: Create sprint plan
  - id: proj-sprint-planner
    trigger_on_complete: [proj-sprint-executor]
    # ...

  # Stage 3: Implement (end of chain)
  - id: proj-sprint-executor
    trigger_on_complete: []
    # ...
```

When `auto_approve_handoffs: false`, each stage requires human approval before triggering the next agent.

### Key Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier |
| `inbox` | Yes | Message inbox name |
| `workspace` | Yes | Absolute path to repository |
| `merge_branch` | **If not `dev`** | Target branch for worktrees (default: `dev`) |
| `invoke.type` | Yes | `skill`, `script`, or `prompt` |
| `invoke.name` | For skills | Skill directory name |
| `invoke.template` | For prompts | Inline prompt template |
| `invoke.template_file` | For prompts | Path to template file (v0.6.7+) |
| `trigger_on_complete` | No | Agent IDs to trigger on success |
| `output_markers` | No | Lines to extract from output |
| `artifact_patterns` | No | Glob patterns for git diff |

### Template Files (v0.6.7+)

For `invoke.type: prompt`, you can store templates in external files instead of inline YAML:

```yaml
invoke:
  type: prompt
  template_file: ~/.ailang/templates/design-doc.md  # Absolute or ~ path
  # OR relative to workspace:
  template_file: .claude/templates/design-doc.md
```

**Path resolution:**
- `~/...` expands to home directory
- Absolute paths used as-is
- Relative paths resolved from agent's `workspace`

**Template variables:**
- `{{.TaskID}}` - Task identifier
- `{{.GithubIssue}}` - GitHub issue number
- `{{.Content}}` - Task content/message
- `{{.Stage}}` - Current stage name
- `{{.OutputMarkers}}` - Expected output markers

### Generic Templates

AILANG provides generic templates you can use as a starting point:

```
ailang/templates/
├── design-doc.md       # Create design documents
├── sprint-planner.md   # Plan implementation sprints
└── sprint-executor.md  # Implement sprint tasks
```

Copy these to `~/.ailang/templates/` or your project, then customize for your needs.

See [templates README](https://github.com/sunholo-data/ailang/tree/dev/templates) for details.

## Step 3: GitHub Sync (Optional)

### Single Repository (Legacy)

```yaml
coordinator:
  github_sync:
    enabled: true
    interval_secs: 300
    watch_labels: []              # Empty = all issues
    target_inbox: design-doc-creator
```

### Multiple Repositories (v0.6.6+)

```yaml
coordinator:
  github_sync:
    repos:
      # Primary project
      - repo: owner/primary-repo
        enabled: true
        interval_secs: 300
        target_inbox: design-doc-creator
        label_routing:
          - label_prefix: "coordinator:bug"
            target: design-doc-creator
          - label_prefix: "coordinator:docs"
            target: coordinator

      # External project
      - repo: owner/external-repo
        enabled: true
        interval_secs: 300
        target_inbox: external-design-doc
        label_routing:
          - label_prefix: "feature"
            target: external-design-doc
          - label_prefix: "bug"
            target: external-design-doc
```

Each repo can have:
- **Different target inbox** (routes to different agents)
- **Custom label routing** (route by label prefix)
- **Independent sync interval**

## Step 4: Testing

### Start the Coordinator

```bash
# Start coordinator daemon
ailang coordinator start

# Check status
ailang coordinator status
```

### Send Test Messages

```bash
# Send to your new agent inbox
ailang messages send my-project-design-doc \
  "Create design doc for user authentication feature" \
  --title "Feature: User Auth" \
  --from "test"

# Check task was created
ailang coordinator status

# View pending tasks
ailang coordinator pending
```

### Verify Worktree

The coordinator creates isolated git worktrees for each task:

```bash
# Worktrees are at:
~/.ailang/state/worktrees/<agent-id>/<task-id>/
```

## Example: Adding Stapledons Voyage

Here's a complete example adding a game project with 10 skills:

### Core Workflow Agents

```yaml
# Design Doc Creator
- id: stapledon-design-doc
  label: "Stapledon Design Doc Creator"
  inbox: stapledon-design-doc
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main  # stapledons_voyage uses main, not dev
  capabilities: [research, docs]
  provider: claude
  trigger_on_complete: [stapledon-sprint-planner]
  invoke:
    type: skill
    name: design-doc-creator
  output_markers:
    - "DESIGN_DOC_PATH:"
  artifact_patterns:
    - "design_docs/**/*.md"

# Sprint Planner
- id: stapledon-sprint-planner
  inbox: stapledon-sprint-planner
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  trigger_on_complete: [stapledon-sprint-executor]
  invoke:
    type: skill
    name: sprint-planner

# Sprint Executor
- id: stapledon-sprint-executor
  inbox: stapledon-sprint-executor
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  trigger_on_complete: []
  invoke:
    type: skill
    name: sprint-executor
```

### Specialized Agents

```yaml
# Game Architect (pre-release validation)
- id: stapledon-architect
  inbox: stapledon-architect
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  invoke:
    type: skill
    name: game-architect

# Asset Manager (image generation)
- id: stapledon-assets
  inbox: stapledon-assets
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  invoke:
    type: skill
    name: asset-manager

# Test Manager (visual regression)
- id: stapledon-tests
  inbox: stapledon-tests
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  invoke:
    type: skill
    name: test-manager
```

### GitHub Sync

```yaml
github_sync:
  repos:
    - repo: sunholo-data/ailang
      enabled: true
      target_inbox: design-doc-creator

    - repo: sunholo-data/stapledons_voyage
      enabled: true
      target_inbox: stapledon-design-doc
      label_routing:
        - label_prefix: "feature"
          target: stapledon-design-doc
        - label_prefix: "bug"
          target: stapledon-design-doc
```

## Troubleshooting

### Agent Not Picking Up Messages

1. Check inbox name matches exactly:
   ```bash
   ailang messages list --inbox your-inbox-name
   ```

2. Verify coordinator is running:
   ```bash
   ailang coordinator status
   ```

3. Check agent is registered:
   ```bash
   cat ~/.ailang/config.yaml | grep "id: your-agent-id"
   ```

### Skill Not Found

1. Verify skill exists at workspace:
   ```bash
   ls /path/to/workspace/.claude/skills/skill-name/
   ```

2. Check skill has correct structure:
   ```
   .claude/skills/skill-name/
   ├── SKILL.md           # Required
   └── scripts/           # Optional
   ```

### Wrong Workspace

- Workspace must be **absolute path**, not relative
- Check path exists and is a git repository

### Worktree Creation Failed

If you see an error like:
```
failed to create worktree: exit status 255
Output: Preparing worktree (new branch 'coordinator/task-xxx')
fatal: not a valid object name: 'dev'
```

This means the coordinator is trying to create a worktree from branch `dev`, but your repository uses a different default branch (like `main`).

**Fix:** Add `merge_branch: main` (or your default branch) to all agents for that repository:

```yaml
- id: my-agent
  workspace: /path/to/repo
  merge_branch: main    # Add this line
  # ... rest of config
```

### GitHub Sync Not Working

1. Check `gh auth status` - correct account active?
2. Verify repo name format: `owner/repo`
3. Check labels match configured `watch_labels`

## Related Documentation

- [Coordinator Guide](./coordinator.md) - Full coordinator documentation
- [Telemetry & Tracing](./telemetry.md) - Trace task execution with distributed tracing
- [Agent Messaging](./agent-messaging.md) - Message system details
- [Collaboration Hub](./collaboration-hub.md) - Web UI for approvals
- [Debugging Guide](./debugging.md) - Troubleshoot coordinator issues

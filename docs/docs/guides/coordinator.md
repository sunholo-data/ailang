# Coordinator Daemon

The AILANG Coordinator is an always-on daemon that automatically processes incoming tasks using AI agents (Claude Code or Gemini CLI) with human-in-the-loop approval workflows.

## Overview

The coordinator watches for messages across multiple inboxes, routes them to configured agents, and executes tasks in isolated git worktrees. This enables:

- **Multi-agent workflows** - Chain agents together (design → plan → execute)
- **Human-in-the-loop approvals** - Review work before merging
- **Isolated environments** - Each task gets its own git worktree
- **Agent-to-agent handoffs** - Session continuity across agents
- **GitHub integration** - Auto-import issues as tasks
- **Real-time dashboard** - Watch execution progress live

## Quick Start

```bash
# 1. Start both services
make services-start

# 2. Open dashboard
open http://localhost:1957

# 3. Send a task
ailang messages send coordinator "Fix the bug in parser.go" \
  --title "Bug: Parser error" --from "user"

# 4. Watch it execute in the dashboard
```

## Agent Configuration

Agents are configured in `~/.ailang/config.yaml`. Each agent has an inbox, workspace, and can trigger other agents on completion.

### Example Configuration

```yaml
# ~/.ailang/config.yaml

coordinator:
  default_provider: claude  # "claude" or "gemini"

  agents:
    # Design Doc Creator - reads GitHub issues, creates design docs
    - id: design-doc-creator
      label: "Design Doc Creator"
      inbox: design-doc-creator
      workspace: /path/to/project
      capabilities: [research, docs]
      provider: claude
      trigger_on_complete: [sprint-planner]  # Chain to next agent
      auto_approve_handoffs: false           # Human reviews before handoff
      auto_merge: false                      # Human reviews changes
      session_continuity: true               # Use --resume for Claude Code
      max_concurrent_tasks: 1

    # Sprint Planner - creates sprint plans from design docs
    - id: sprint-planner
      label: "Sprint Planner"
      inbox: sprint-planner
      workspace: /path/to/project
      capabilities: [research, docs, planning]
      provider: claude
      trigger_on_complete: [sprint-executor]
      auto_approve_handoffs: false
      auto_merge: false
      session_continuity: true
      max_concurrent_tasks: 1

    # Sprint Executor - implements approved sprint plans
    - id: sprint-executor
      label: "Sprint Executor"
      inbox: sprint-executor
      workspace: /path/to/project
      capabilities: [code, test, docs]
      provider: claude
      trigger_on_complete: []  # End of chain
      auto_approve_handoffs: false
      auto_merge: false
      session_continuity: true
      max_concurrent_tasks: 1

    # General Coordinator - handles ad-hoc tasks
    - id: coordinator
      label: "General Coordinator"
      inbox: coordinator
      workspace: /path/to/project
      capabilities: [code, test, docs, research]
      provider: claude
      max_concurrent_tasks: 2

  # GitHub issue import configuration
  github_sync:
    enabled: true
    interval_secs: 300              # Check every 5 minutes
    watch_labels: []                # Empty = import all issues
    target_inbox: design-doc-creator
```

### Agent Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique agent identifier |
| `label` | string | Human-readable name |
| `inbox` | string | Message inbox to watch |
| `workspace` | string | Base directory for worktrees |
| `capabilities` | list | Agent capabilities (code, test, docs, research, planning) |
| `provider` | string | AI provider: "claude" or "gemini" |
| `trigger_on_complete` | list | Agent IDs to trigger when this agent completes |
| `auto_approve_handoffs` | bool | Skip approval for agent-to-agent handoffs |
| `auto_merge` | bool | Automatically merge approved changes |
| `session_continuity` | bool | Use `--resume` (Claude) or `--conversation-id` (Gemini) |
| `max_concurrent_tasks` | int | Maximum concurrent tasks (0 = unlimited) |

## Workflow Pipelines

The coordinator supports chained agent workflows where one agent's output triggers another.

### Example: Issue to Implementation

```
GitHub Issue
    ↓ (github_sync imports to design-doc-creator inbox)
┌─────────────────────────────────────────────────────────┐
│  design-doc-creator                                      │
│  • Reads issue                                           │
│  • Creates design doc in design_docs/planned/            │
│  • trigger_on_complete: [sprint-planner]                 │
└─────────────────────────────────────────────────────────┘
    ↓ (handoff message with session ID)
[Human Approval] ← Review design doc before planning
    ↓
┌─────────────────────────────────────────────────────────┐
│  sprint-planner                                          │
│  • Reads design doc                                      │
│  • Creates sprint plan with milestones                   │
│  • Creates JSON progress file                            │
│  • trigger_on_complete: [sprint-executor]                │
└─────────────────────────────────────────────────────────┘
    ↓ (handoff message with session ID)
[Human Approval] ← Review sprint plan before execution
    ↓
┌─────────────────────────────────────────────────────────┐
│  sprint-executor                                         │
│  • Implements sprint plan with TDD                       │
│  • Updates CHANGELOG, design docs                        │
│  • Runs tests and linting                                │
│  • trigger_on_complete: []  (end of chain)               │
└─────────────────────────────────────────────────────────┘
    ↓
[Human Approval] ← Review code changes before merge
    ↓
Changes merged to main branch
```

### Approval Gates

Each agent can require human approval before:
1. **Handoff approval** - Before triggering the next agent
2. **Merge approval** - Before merging changes to main branch

```yaml
# Require approval for both
auto_approve_handoffs: false
auto_merge: false

# Auto-approve handoffs but require merge approval
auto_approve_handoffs: true
auto_merge: false

# Fully autonomous (dangerous!)
auto_approve_handoffs: true
auto_merge: true
```

## GitHub Integration

The coordinator can automatically import GitHub issues as tasks.

### Configuration

```yaml
coordinator:
  github_sync:
    enabled: true
    interval_secs: 300           # Check every 5 minutes (minimum)
    watch_labels: [bug, feature] # Filter by labels (empty = all)
    target_inbox: design-doc-creator
```

### Manual Import

```bash
# Import all issues
ailang messages import-github

# Import with label filter
ailang messages import-github --labels bug,feature

# Dry run (preview only)
ailang messages import-github --dry-run
```

### Issue → Message Mapping

| GitHub Issue Field | Message Field |
|-------------------|---------------|
| Title | title |
| Body | content |
| Labels (bug, feature) | type |
| Issue number | github_issue_number |

## Approval Workflow

When a task completes, the coordinator creates an approval request.

### Dashboard Integration

The Collaboration Hub dashboard shows:
- Pending approvals with worktree path
- Git diff viewer for changes
- One-click approve/reject buttons

### API Endpoints

```bash
# List pending approvals
GET /api/coordinator/pending

# Approve changes (merges to main)
POST /api/coordinator/approve/{approval_id}

# Reject changes (preserves worktree)
POST /api/coordinator/reject/{approval_id}

# Get git diff for a task
GET /api/coordinator/tasks/{task_id}/diff
```

### Approval Response Format

```json
{
  "id": "approval-123",
  "task_id": "task-456",
  "type": "task_completion",
  "description": "Review changes for: Fix parser bug",
  "status": "pending",
  "created_at": "2025-12-31T10:00:00Z",
  "worktree_path": "/Users/mark/.ailang/state/worktrees/coordinator/task-456",
  "session_id": "claude-session-abc",
  "task_title": "Fix parser bug",
  "task_status": "pending_approval",
  "provider": "claude-code"
}
```

### What Happens on Approve

1. Worktree changes are merged to main branch
2. Merge conflicts are detected and reported
3. If agent has `trigger_on_complete`, handoff message is sent
4. Worktree is cleaned up

### What Happens on Reject

1. Worktree is preserved for manual inspection
2. Task is marked as "rejected"
3. No handoff is triggered
4. Changes are NOT merged

## Commands

### Start

Start the coordinator daemon:

```bash
ailang coordinator start [options]
```

**Options:**
- `--poll-interval DURATION` - How often to check for messages (default: 30s)
- `--max-worktrees N` - Maximum concurrent tasks (default: 3)
- `--state-dir DIR` - State directory (default: ~/.ailang/state)
- `--log-file PATH` - Log file path (default: ~/.ailang/logs/coordinator.log)

### Status

Check the daemon status:

```bash
ailang coordinator status [options]
```

**Example output:**
```
Coordinator Status

  State:      ▶ running
  PID:        12345

Task Statistics
  Completed:  21
  Running:    2
  Total Cost: $3.47
  Tokens:     79741
```

### Stop

Stop the coordinator daemon:

```bash
ailang coordinator stop
```

## Sending Tasks

### To a Specific Agent

```bash
# Send to design-doc-creator
ailang messages send design-doc-creator \
  "Create a design doc for semantic caching" \
  --title "Feature: Semantic Caching" \
  --from "user"

# Send to sprint-planner
ailang messages send sprint-planner \
  "Plan sprint for M-CACHE feature" \
  --title "Sprint: M-CACHE" \
  --from "user"

# Send to general coordinator
ailang messages send coordinator \
  "Fix the type error in elaborator.go" \
  --title "Bug: Type error" \
  --from "user" \
  --type bug
```

### With GitHub Issue Creation

```bash
# Send message AND create GitHub issue
ailang messages send design-doc-creator \
  "Parser crashes on nested blocks" \
  --title "Bug: Parser crash" \
  --type bug \
  --github
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Coordinator Daemon                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Agent Registry                          │  │
│  │  design-doc-creator │ sprint-planner │ sprint-executor    │  │
│  └───────────────────────────────────────────────────────────┘  │
│         ↓                      ↓                    ↓           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Watcher   │→ │  Analyzer   │→ │      Task Executor      │  │
│  │ (per inbox) │  │ (classify)  │  │  (Claude/Gemini CLI)    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│         ↑                                      ↓                │
│  ┌─────────────┐                    ┌─────────────────────────┐ │
│  │  Messages   │                    │    Worktree Manager     │ │
│  │  (SQLite)   │                    │   (per agent workspace) │ │
│  └─────────────┘                    └─────────────────────────┘ │
│                                                ↓                │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                  Approval Checkpoint                        ││
│  │   pending_approval → [Human Review] → merge OR reject       ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                ↓                │
│                                     ┌─────────────────────────┐ │
│                                     │   HTTP Broadcaster      │ │
│                                     │  (streams to dashboard) │ │
│                                     └─────────────────────────┘ │
└────────────────────────────────────────────────┬────────────────┘
                                                 │ HTTP POST
                                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Collaboration Hub Server                        │
│                    (ailang serve :1957)                          │
├─────────────────────────────────────────────────────────────────┤
│  POST /api/coordinator/events → WebSocket broadcast → Browser   │
│  GET /api/coordinator/pending → Pending approvals list          │
│  GET /api/coordinator/tasks/{id}/diff → Git diff viewer         │
│  POST /api/coordinator/approve/{id} → Merge changes             │
│  POST /api/coordinator/reject/{id} → Preserve worktree          │
└─────────────────────────────────────────────────────────────────┘
```

## Task Processing

### Task Types

The analyzer classifies tasks into types based on keywords:

| Type | Keywords | Priority |
|------|----------|----------|
| Bug Fix | bug, fix, error, crash, broken | High (2) |
| Feature | add, implement, create, new | Medium (5) |
| Refactor | refactor, cleanup, simplify, optimize | Medium (6) |
| Test | test, coverage, unittest | Medium (6) |
| Docs | document, readme, comment, tutorial | Low (7) |
| Research | research, investigate, explore, benchmark | Low (8) |

### Duplicate Detection

The coordinator uses SimHash (locality-sensitive hashing) to detect duplicate tasks:

- Tasks with >80% similarity are marked as duplicates
- Duplicates are skipped to avoid redundant work
- Original task ID is recorded for reference

### Provider Selection

Tasks are routed to providers based on agent configuration:

| Provider | CLI Tool | Best For |
|----------|----------|----------|
| `claude` | Claude Code CLI | Code editing, complex tasks |
| `gemini` | Gemini CLI | Research, documentation |

## Storage

### SQLite Databases

```
~/.ailang/state/
├── coordinator.db      # Task state, approvals, events
├── collaboration.db    # Messages, threads (shared with CLI)
└── worktrees/          # Git worktrees per agent
    └── coordinator/
        └── task-id/    # Isolated workspace
```

### Task Record Fields

| Field | Description |
|-------|-------------|
| `id` | Unique task identifier |
| `title` | Task title (from message) |
| `content` | Full task content |
| `type` | Classified type (bug_fix, feature, etc.) |
| `status` | pending, running, pending_approval, completed, rejected, failed |
| `provider` | AI provider used |
| `worktree_path` | Path to git worktree |
| `session_id` | Claude Code / Gemini CLI session ID |
| `cost` | Execution cost in USD |
| `tokens_used` | Total tokens consumed |

## Real-Time Dashboard

The coordinator streams task execution events to the Collaboration Hub dashboard.

### Event Flow

1. **Daemon executes task** - Claude Code or Gemini CLI runs
2. **Events generated** - Status changes, tool calls, output, metrics
3. **HTTP broadcaster** - POSTs to `http://127.0.0.1:1957/api/coordinator/events`
4. **Server receives** - Converts to WebSocket format
5. **WebSocket broadcast** - All connected browsers receive updates
6. **UI renders** - Task progress shown in real-time

### Event Types

| Event Type | Description |
|------------|-------------|
| `status` | Task state changes (running, completed, failed) |
| `turn_start` | New conversation turn begins |
| `turn_end` | Conversation turn completes |
| `text` | Text output from the agent |
| `tool_call` | Tool invocation (file edit, bash, etc.) |
| `tool_result` | Tool execution result |
| `metrics` | Token usage, cost, duration |
| `error` | Error messages |

## Service Management

### Quick Start

```bash
# Start both services
make services-start

# Check status
make services-status

# Stop both services
make services-stop

# Restart with fresh build
make services-restart
```

### Individual Service Control

```bash
# Server only
ailang serve           # Start server (foreground)
make serve-bg          # Start server (background)

# Coordinator only
ailang coordinator start   # Start daemon
ailang coordinator stop    # Stop daemon
ailang coordinator status  # Check status
```

**Important:** The server must be running before the coordinator for real-time streaming to work.

## Your Daily Workflow

The coordinator is designed to be your **autonomous coding assistant**. Here's how to integrate it into your daily workflow.

### Morning Routine

1. **Start the services** (if not already running):
   ```bash
   make services-start
   ```

2. **Open the dashboard**: http://localhost:1957

3. **Check for pending approvals**:
   ```bash
   ailang coordinator pending
   ```

4. **Review and approve/reject** completed work

### Sending Tasks to Agents

**For bug fixes:**
```bash
ailang messages send coordinator "Fix the null pointer bug in parser.go" \
  --title "Bug: Parser NPE" --from "user" --type bug
```

**For new features:**
```bash
ailang messages send design-doc-creator "Add support for semantic caching" \
  --title "Feature: Semantic Cache" --from "user"
```

**With GitHub issue tracking:**
```bash
ailang messages send coordinator "Fix issue described in #42" \
  --title "Bug: Fix #42" --from "user" --type bug --github
```

### Reviewing Agent Work

When agents complete tasks, they request approval. You can review via:

**Dashboard (Recommended):**
- Open http://localhost:1957
- Click on Task Execution tab
- Review git diff
- Approve or reject with notes

**CLI (Alternative):**
```bash
# List pending approvals
ailang coordinator pending

# View the diff
ailang coordinator diff <task-id>

# Approve (merges changes to dev branch)
ailang coordinator approve <task-id>

# Reject (preserves worktree, discards changes)
ailang coordinator reject <task-id>
```

### Decision Making

**When to Approve:**
- Code solves the stated problem
- Tests pass (agents run tests before requesting approval)
- No obvious security issues
- Follows project conventions

**When to Reject:**
- Code has bugs or fails tests
- Goes beyond scope of task
- Needs human intervention
- Different approach needed

**When rejecting**, add a note explaining why - this helps agents improve.

### Multi-Agent Pipelines

The full autonomous pipeline:

```
GitHub Issue
    ↓ (automatic import)
design-doc-creator
    ↓ (YOU APPROVE design doc)
sprint-planner
    ↓ (YOU APPROVE sprint plan)
sprint-executor
    ↓ (YOU APPROVE implementation)
Changes merged to dev
```

Each approval gate lets you:
- Course correct before more work is done
- Ensure quality at each stage
- Maintain control over your codebase

### Tips for Effective Collaboration

1. **Write clear task descriptions** - Include file paths, expected behavior
2. **Break big tasks into smaller ones** - Agents handle focused tasks better
3. **Review early, reject fast** - If a design is wrong, reject before planning
4. **Use rejection notes** - Help agents understand what went wrong
5. **Trust but verify** - Always review diffs before approving

## Troubleshooting

### Daemon Won't Start

```bash
# Check if already running
ailang coordinator status

# Check PID file
cat ~/.ailang/state/coordinator.pid

# Remove stale PID file
rm ~/.ailang/state/coordinator.pid
```

### Tasks Not Executing

```bash
# Check for unread messages
ailang messages list --unread

# Check which inboxes are configured
cat ~/.ailang/config.yaml | grep inbox

# Check coordinator logs
tail -100 ~/.ailang/logs/coordinator.log

# Verify providers are available
which claude  # Claude Code CLI
which gemini  # Gemini CLI
```

### Approvals Not Showing

```bash
# Check pending approvals via API
curl http://127.0.0.1:1957/api/coordinator/pending | jq

# Check task status
sqlite3 ~/.ailang/state/coordinator.db \
  "SELECT id, title, status FROM tasks ORDER BY created_at DESC LIMIT 5"
```

### Git Merge Conflicts

When approving changes that conflict:

1. Approval returns error with conflict files
2. Worktree is preserved at `~/.ailang/state/worktrees/coordinator/<task-id>/`
3. Manually resolve conflicts in the worktree
4. Commit and merge manually

## See Also

- [Agent Messaging](./agent-messaging.md) - How to send messages
- [Collaboration Hub](./collaboration-hub.md) - Web UI for monitoring
- [Development Workflow](./development-workflow.md) - Integration with development

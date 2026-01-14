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
| `invoke` | object | How to invoke this agent (skill/agent/prompt) - v0.6.3+ |
| `output_markers` | list | Markers to extract from output - v0.6.3+ |
| `approval` | object | Approval workflow configuration - v0.6.3+ |

### Generic Workflow Configuration (v0.6.3+)

The coordinator supports fully configurable workflows, allowing any project to define custom skills, stages, and approval rules without code changes.

#### Invoke Configuration

Specifies how an agent executes its tasks:

```yaml
# Invoke a Claude Code skill
invoke:
  type: skill
  name: design-doc-creator  # Skill name (from .claude/skills/)

# Hand off to another agent
invoke:
  type: agent
  name: sprint-planner      # Target agent ID

# Use custom prompt template
invoke:
  type: prompt
  template: |
    Task {{.TaskID}} for issue #{{.GithubIssue}}:
    {{.Content}}

    When done, output: RESULT: <value>

# Execute a deterministic script (v0.6.4+)
invoke:
  type: script
  command: "./scripts/run_eval.sh"   # Script path or inline command
  shell: /bin/bash                    # Shell to use (default: /bin/sh)
  env_from_payload: true              # Convert JSON payload to env vars
  timeout: "30m"                      # Execution timeout
  working_dir: "{{.Workspace}}"       # Working directory
```

#### Script Invoke Type (v0.6.4+)

The `script` invoke type enables deterministic workflow execution without AI inference. This is useful for:

- **Eval runners** - Execute benchmarks with consistent behavior
- **Data pipelines** - Transform data without AI variability
- **Infrastructure tasks** - Run deployment scripts
- **Cost optimization** - $0.00 execution cost for deterministic tasks

**JSON Payload to Environment Variables:**

When `env_from_payload: true`, the task's JSON content is converted to environment variables:

```json
{
  "model": "gpt5",
  "benchmark": "fizzbuzz",
  "db": {
    "host": "localhost",
    "port": 5432
  }
}
```

Becomes:
```bash
MODEL=gpt5
BENCHMARK=fizzbuzz
DB_HOST=localhost
DB_PORT=5432
```

**Auto-injected variables:**
- `AILANG_TASK_ID` - Coordinator task ID
- `AILANG_MESSAGE_ID` - Source message ID
- `AILANG_WORKSPACE` - Worktree path

**Example: Echo Demo Agent**

A demo script is included at `scripts/coordinator/echo_payload.sh` that echoes back all payload variables:

```yaml
coordinator:
  agents:
    - id: echo-demo
      label: "Echo Demo"
      inbox: echo-demo
      workspace: /path/to/project
      invoke:
        type: script
        command: "./scripts/coordinator/echo_payload.sh"
        env_from_payload: true
        timeout: "1m"
      output_markers:
        - "ECHO_COMPLETE:"
      trigger_on_complete: []
```

**Test the demo:**

```bash
# Send a JSON payload to the echo-demo agent
ailang messages send echo-demo '{"model": "gpt5", "benchmark": "fizzbuzz", "config": {"parallel": true}}' \
  --title "Echo test" --from "user"
```

**Expected output:**
```
AILANG Script Invoke Demo
Task Context:
  AILANG_TASK_ID:     task-xxx
  AILANG_MESSAGE_ID:  msg-xxx
  AILANG_WORKSPACE:   /path/to/worktree

Payload Variables (from JSON):
  BENCHMARK=fizzbuzz
  CONFIG_PARALLEL=true
  MODEL=gpt5

ECHO_COMPLETE: true
```

**Production Example: Eval Runner**

```yaml
coordinator:
  agents:
    - id: eval-runner
      label: "Eval Runner"
      inbox: eval-runner
      workspace: /path/to/project
      invoke:
        type: script
        command: "./scripts/run_eval.sh"
        env_from_payload: true
        timeout: "1h"
      output_markers:
        - "EVAL_COMPLETE:"
        - "RESULTS_PATH:"
      trigger_on_complete: []
```

**Mixed pipelines** work seamlessly - you can chain AI agents with script agents:

```
design-doc-creator (AI) → sprint-planner (AI) → eval-runner (Script)
```

#### Output Markers

Markers to extract from execution output for stage completion:

```yaml
output_markers:
  - "DESIGN_DOC_PATH:"
  - "SPRINT_PLAN_PATH:"
  - "IMPLEMENTATION_COMPLETE:"
```

#### Approval Configuration

Custom labels for GitHub approval workflow:

```yaml
approval:
  needs_label: needs-design-approval     # Label when awaiting review
  approved_label: design-approved        # Label that triggers next stage
  github_comment_template: |             # Comment posted on completion
    ## Design Document Ready
    Path: {{.DesignDocPath}}
```

#### Full Example with Generic Workflow

```yaml
coordinator:
  agents:
    - id: design-doc-creator
      label: "Design Doc Creator"
      inbox: design-doc-creator
      workspace: /path/to/project
      capabilities: [research, docs]
      provider: claude

      # NEW: Generic workflow config (v0.6.3+)
      invoke:
        type: skill
        name: design-doc-creator
      output_markers:
        - "DESIGN_DOC_PATH:"
      approval:
        needs_label: needs-design-approval
        approved_label: design-approved
        github_comment_template: "design_doc"

      trigger_on_complete: [sprint-planner]
      auto_approve_handoffs: false

    - id: sprint-planner
      label: "Sprint Planner"
      inbox: sprint-planner
      workspace: /path/to/project
      capabilities: [planning]
      provider: claude
      invoke:
        type: skill
        name: sprint-planner
      output_markers:
        - "SPRINT_PLAN_PATH:"
        - "SPRINT_JSON_PATH:"
      approval:
        needs_label: needs-sprint-approval
        approved_label: sprint-approved
      trigger_on_complete: [sprint-executor]

    - id: sprint-executor
      label: "Sprint Executor"
      inbox: sprint-executor
      workspace: /path/to/project
      capabilities: [code, test]
      provider: claude
      invoke:
        type: skill
        name: sprint-executor
      output_markers:
        - "IMPLEMENTATION_COMPLETE:"
        - "BRANCH_NAME:"
        - "FILES_CREATED:"
        - "FILES_MODIFIED:"
      approval:
        needs_label: needs-implementation-approval
        approved_label: implementation-approved
      trigger_on_complete: []
```

#### Backwards Compatibility

Agents without explicit `invoke`, `output_markers`, or `approval` config will use legacy defaults for known AILANG agent IDs:
- `design-doc-creator`: skill invocation, `DESIGN_DOC_PATH:` marker, `design-approved` label
- `sprint-planner`: skill invocation, `SPRINT_PLAN_PATH:` marker, `sprint-approved` label
- `sprint-executor`: skill invocation, `IMPLEMENTATION_COMPLETE:` marker, `implementation-approved` label

**Note:** Legacy defaults are deprecated and will be removed in v0.7.0. New agents should always use explicit configuration.

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

## GitHub-Driven Approval Workflow (v0.6.2+)

The coordinator supports a fully GitHub-driven workflow where all approvals happen via GitHub labels. This enables:

- **Familiar tooling** - Review everything in GitHub's UI
- **Mobile-friendly** - Approve from GitHub mobile app
- **Audit trail** - All approvals tracked in issue history
- **Team collaboration** - Multiple reviewers can participate

### How It Works

```
GitHub Issue Created
       ↓
Coordinator imports issue → TaskChain starts
       ↓
┌─────────────────────────────────────────────────────────────┐
│  DESIGN STAGE                                                │
│  • Agent creates design doc                                  │
│  • Comment posted with full design doc content               │
│  • Label added: needs-design-approval                        │
│  • Human reviews design in GitHub                            │
│  • Human adds label: design-approved                         │
│  • ApprovalWatcher detects → advances to next stage          │
└─────────────────────────────────────────────────────────────┘
       ↓
┌─────────────────────────────────────────────────────────────┐
│  SPRINT STAGE                                                │
│  • Agent creates sprint plan                                 │
│  • Comment posted with full sprint plan content              │
│  • Label added: needs-sprint-approval                        │
│  • Human reviews plan in GitHub                              │
│  • Human adds label: sprint-approved                         │
│  • ApprovalWatcher detects → advances to next stage          │
└─────────────────────────────────────────────────────────────┘
       ↓
┌─────────────────────────────────────────────────────────────┐
│  IMPLEMENTATION STAGE                                        │
│  • Agent implements the sprint plan                          │
│  • Comment posted with file changes summary                  │
│  • Label added: needs-merge-approval                         │
│  • Human reviews changes (can check worktree)                │
│  • Human adds label: merge-approved                          │
│  • ApprovalWatcher detects → merges to dev, closes issue     │
└─────────────────────────────────────────────────────────────┘
       ↓
Issue auto-closed with completion summary
```

### GitHub Labels

The coordinator uses these labels (created automatically if missing):

**Request labels** (coordinator adds these):
| Label | Meaning |
|-------|---------|
| `needs-design-approval` | Design doc ready for review |
| `needs-sprint-approval` | Sprint plan ready for review |
| `needs-merge-approval` | Implementation ready to merge |
| `needs-revision` | Human requested changes |

**Approval labels** (human adds these):
| Label | Action |
|-------|--------|
| `design-approved` | Advance to sprint planning stage |
| `sprint-approved` | Advance to implementation stage |
| `merge-approved` | Merge changes and close issue |

### GitHub Comment Format

Each stage posts a detailed comment. For design and sprint stages, the full document content is included in a collapsible section:

```markdown
## 📋 Design Document Ready

I've created a design document for this feature.

**Design Doc**: `design_docs/planned/v0_6_2/m-string-reverse.md`

<details>
<summary>📄 View Full Design Document</summary>

# M-STRING-REVERSE: String Reverse Builtin

## Problem Statement
...

</details>

---

**Next Steps:**
1. Review the design document above
2. If approved, add the `design-approved` label
3. If changes needed, add `needs-revision` label with a comment

⏱️ Duration: 2m 30s | 💰 Cost: $0.15 | 🎟️ Tokens: 12,450
```

### Alternative: Local Approval

You can also approve via CLI (useful when ApprovalWatcher isn't detecting labels):

```bash
# List pending approvals
ailang coordinator pending

# Approve a specific task (also adds label on GitHub)
ailang coordinator approve <task-id>

# Reject a task
ailang coordinator reject <task-id>
```

When using local approval, the coordinator automatically syncs labels to GitHub.

### Requesting Revision

To request changes at any stage:

1. Add the `needs-revision` label on GitHub
2. Add a comment explaining what needs to change
3. The pipeline pauses until revision is addressed

### Troubleshooting GitHub Workflow

**Labels not being detected:**
- Check if ApprovalWatcher is running: look for "GitHub approval watcher started" in logs
- Verify the issue is being watched: check coordinator logs for "Started watching issue #X"
- Use local approval as fallback: `ailang coordinator approve <task-id>`

**Comments not appearing:**
- Verify GitHub token has write access: `gh auth status`
- Check coordinator logs for GitHub API errors

**Issue not auto-closing:**
- Ensure merge completes successfully
- Check worktree for uncommitted changes
- Verify no merge conflicts

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

### Multi-Channel Approval Workflow (v0.6.4+)

Approvals can come from three different channels, all feeding into the same workflow:

| Channel | How to Approve/Reject | Tracked In |
|---------|----------------------|------------|
| **Dashboard** | Click buttons in Collaboration Hub UI | `approval.channel: "dashboard"` |
| **CLI** | `ailang coordinator approve/reject` | `approval.channel: "cli"` |
| **GitHub** | Add labels (`ailang:approved`, `ailang:needs-revision`) | `approval.channel: "github"` |

#### Iteration Tracking

When tasks are rejected, they can be re-triggered with feedback. The iteration count tracks retry attempts:

- **Iteration 1**: First run
- **Iteration 2**: First retry (after rejection with feedback)
- **Iteration 3**: Final attempt (max iterations)

The UI shows an "Iteration N/3" badge for retriggered tasks, with a "Final attempt" warning on iteration 3.

#### Feedback Flow

When you reject a task with feedback:

1. Feedback is stored as a `human_feedback` event in the task audit trail
2. If the task has a linked GitHub issue, feedback is posted as a comment
3. The task is re-queued with `iteration + 1`
4. The agent receives the feedback in its next execution context

Feedback can be provided via:
- **CLI**: `ailang coordinator reject <task-id> -f "Your feedback"`
- **Dashboard**: Enter feedback in the rejection modal
- **GitHub**: Add `ailang:needs-revision` label and comment on the issue

#### Telemetry Spans

Each approval decision creates an OTEL span for observability:

```
approval.decision
├── task.id: "task-12345678"
├── approval.action: "approve" | "reject"
├── approval.channel: "dashboard" | "cli" | "github"
├── approval.by: "cli-user" | "github:username"
├── task.iteration: 1
└── task.stage: "implementation"
```

These spans appear in the trace timeline and ExecHierarchy task view.

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
| `script` | Shell (v0.6.4+) | Deterministic workflows, evals, data pipelines |

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

## Cloud Tracing

The coordinator integrates with Google Cloud Trace for distributed tracing. Task executions, provider calls, and approval workflows are all traced.

![AILANG Cloud Trace](/img/ailand-cloud-trace.png)

Enable by setting `GOOGLE_CLOUD_PROJECT`:

```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
ailang coordinator start
```

For complete telemetry configuration (OTLP, Jaeger, dual export), see the [Telemetry Guide](./telemetry.md).

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

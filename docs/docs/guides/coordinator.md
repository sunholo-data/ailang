# Coordinator Daemon

The AILANG Coordinator is an always-on daemon that automatically processes incoming tasks using AI agents (Claude Code or Gemini CLI).

## Overview

The coordinator watches for unread messages in the AILANG messaging system and executes them as tasks in isolated git worktrees. This enables:

- **Autonomous task execution** - Tasks run without human intervention
- **Isolated environments** - Each task gets its own git worktree
- **Multi-agent support** - Choose between Claude Code and Gemini CLI
- **Persistent state** - Task history stored in SQLite
- **Cloud-ready** - Neutral storage interface for future cloud backends

## Quick Start

```bash
# Start the coordinator daemon
ailang coordinator start

# Check status
ailang coordinator status

# Stop the daemon
ailang coordinator stop
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Coordinator Daemon                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Watcher   │→ │  Analyzer   │→ │      Task Executor      │  │
│  │  (polling)  │  │ (classify)  │  │  (Claude/Gemini CLI)    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│         ↑                                      ↓                │
│  ┌─────────────┐                    ┌─────────────────────────┐ │
│  │  Messages   │                    │    Worktree Manager     │ │
│  │  (SQLite)   │                    │   (git worktrees)       │ │
│  └─────────────┘                    └─────────────────────────┘ │
│                                                ↓                │
│                                     ┌─────────────────────────┐ │
│                                     │    Store (SQLite)       │ │
│                                     │   Task history/stats    │ │
│                                     └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

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

**Examples:**
```bash
# Start with defaults
ailang coordinator start

# Check more frequently
ailang coordinator start --poll-interval 10s

# Allow more concurrent tasks
ailang coordinator start --max-worktrees 5
```

### Status

Check the daemon status:

```bash
ailang coordinator status [options]
```

**Options:**
- `--json` - Output status as JSON
- `--state-dir DIR` - State directory

**Example output:**
```
Coordinator Status

  State:      ▶ running
  PID:        12345
  Started:    2025-12-29 10:00:00
  Uptime:     2h30m
  Tasks Run:  15
```

### Stop

Stop the coordinator daemon:

```bash
ailang coordinator stop [options]
```

**Options:**
- `--state-dir DIR` - State directory

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

Tasks are routed to providers based on type:

| Task Type | Primary Provider | Fallback |
|-----------|------------------|----------|
| Bug Fix, Feature, Refactor, Test | Claude Code CLI | Gemini CLI |
| Docs, Research | Gemini API | Claude Code CLI |

## Storage

### SQLite (Local)

By default, task state is stored in SQLite:

```
~/.ailang/state/coordinator.db
```

The store tracks:
- Task records (ID, content, type, priority, status)
- Execution results (output, duration, cost, tokens)
- State transitions (pending → running → completed)
- Fingerprints for duplicate detection

### Cloud (Future)

The `Store` interface enables future cloud backends:

```go
type Store interface {
    CreateTask(ctx, task) error
    GetTask(ctx, id) (*TaskRecord, error)
    UpdateTask(ctx, task) error
    // ... other methods
}
```

Planned backends:
- Google Firestore
- AWS DynamoDB
- PostgreSQL (Cloud SQL/RDS)

## Git Worktrees

Each task executes in an isolated git worktree:

```bash
# Worktrees are created at:
~/.ailang/state/worktrees/coordinator/<task-id>/

# Branch naming:
coordinator/<task-id>
```

Benefits:
- Tasks don't interfere with each other
- Changes can be reviewed before merging
- Failed tasks don't pollute main branch
- Concurrent execution possible

## Integration with Claude Code

The coordinator can be used as a task delegation mechanism from Claude Code:

```bash
# 1. Claude Code receives a complex task
# 2. Claude sends it to the coordinator
ailang messages send coordinator "Implement feature X" --title "Feature: X"

# 3. Coordinator picks it up and executes
# 4. Results stored in SQLite for review
```

### From Within Claude

When running as Claude in Claude Code, you can delegate tasks:

```bash
# Send a task to the coordinator
ailang messages send user "Fix the bug in parser.go" --type bug

# Check if coordinator is running
ailang coordinator status

# If not running, start it
ailang coordinator start
```

## Logs and Debugging

### Log File

```bash
# View logs
tail -f ~/.ailang/logs/coordinator.log
```

### Debug Mode

```bash
# Start with verbose logging
DEBUG=1 ailang coordinator start
```

### Task Database

```bash
# Query task history
sqlite3 ~/.ailang/state/coordinator.db "SELECT * FROM tasks"
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AILANG_STATE_DIR` | State directory | `~/.ailang/state` |
| `AILANG_LOG_DIR` | Log directory | `~/.ailang/logs` |

### Config File (Future)

```yaml
# ~/.ailang/config.yaml
coordinator:
  poll_interval: 30s
  max_worktrees: 3
  providers:
    - claude-code
    - gemini-cli
  store:
    type: sqlite  # or "cloud"
    path: ~/.ailang/state/coordinator.db
```

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

# Check coordinator logs
tail -100 ~/.ailang/logs/coordinator.log

# Verify providers are available
which claude  # Claude Code CLI
which gemini  # Gemini CLI
```

### High Memory Usage

```bash
# Reduce concurrent tasks
ailang coordinator stop
ailang coordinator start --max-worktrees 1

# Clean up old worktrees
rm -rf ~/.ailang/state/worktrees/coordinator/*
```

## See Also

- [Agent Messaging](./agent-messaging.md) - How to send messages to the coordinator
- [Collaboration Hub](./collaboration-hub.md) - Web UI for monitoring
- [Development Workflow](./development-workflow.md) - Integration with development

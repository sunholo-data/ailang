---
paths:
  - "internal/coordinator/**"
  - "internal/executor/**"
  - "internal/server/**"
  - "cmd/ailang/coordinator*"
  - "ui/**"
---

# Coordinator & Infrastructure Rules

## Coordinator Daemon

The coordinator executes tasks autonomously using AI agents in isolated git worktrees.

```bash
make services-start                          # Start server + coordinator
ailang coordinator status                    # Check if running
ailang messages send coordinator "Fix bug"   # Delegate a task
ailang coordinator pending                   # Review pending approvals
```

**Agent workflow:** GitHub Issue → design-doc-creator → [Approval] → sprint-planner → [Approval] → sprint-executor → [Approval] → Merged

**Config**: `~/.ailang/config.yaml` | **Cloud mode**: Pub/Sub + Cloud Run (v0.9.0+)

## Collaboration Hub Server

Use the `collaboration-hub` skill for development.
```bash
make services-start     # Start server + coordinator
make services-status    # Check both services
make services-stop      # Stop both services
```

## Chain Execution Monitoring

`ailang chains` is the canonical CLI for examining executions. Works offline (direct SQLite).

```bash
ailang chains list                       # List all chains
ailang chains list --agent X --since 24h # Filter by agent/time
ailang chains view <chain-id> --spans    # Full execution with sessions + tools
ailang chains tree <chain-id> --detailed # ASCII tree with tool timeline
ailang chains stats --by-agent           # Cost/token breakdown
ailang chains diagnose <chain-id>        # Quick health report
```

## Auditing Agent Work

After a coordinator task completes:
1. `ailang chains view <chain-id> --spans` — execution flow
2. `ailang coordinator logs <task-id> --limit 500` — conversation text
3. `ailang coordinator diff <task-id>` — git changes

**Key checks:** Model used (Haiku too weak for compiler), turn count/cost, code changes in `internal/`, runtime vs compile testing.

## Database Architecture

Three SQLite databases: `observatory.db` (spans/traces), `coordinator.db` (tasks/approvals), `collaboration.db` (messages).

**Full reference:** See `docs/docs/guides/database-architecture.md`

## TRACEPARENT Not Propagated

Claude Code does NOT propagate TRACEPARENT to subprocess environments. Child spans are in DIFFERENT traces. Known, accepted limitation.

**DO NOT:** Try to inject TRACEPARENT, attempt runtime fixes, or re-investigate.
**Workaround:** Use `task_id`/`parent_task_id` attributes for cross-trace linking.

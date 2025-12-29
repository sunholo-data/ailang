# M-COORDINATOR-QUESTION-MODE: Read-Only Execution for Questions

**Status:** IMPLEMENTED
**Version:** v0.6.2
**Date:** 2025-12-29
**Author:** Claude Code

## Summary

Implemented differentiated execution modes based on message "kind" (directive vs question). Questions now execute in read-only mode with restricted tool access, preventing file modifications while still allowing research and exploration.

## Problem Statement

Previously, both directives and questions were treated identically by the coordinator - both had full execution capabilities including file writes, edits, and bash commands. This was inappropriate for questions, which are informational queries that shouldn't modify the codebase.

**User impact:**
- Questions could accidentally trigger code changes
- No way to safely ask "what does this code do?" without risk
- Coordinator couldn't distinguish research tasks from action tasks

## Solution

### Design Decision: Per-Task Tool Restriction

We chose to use Claude Code CLI's `--allowedTools` flag to restrict available tools at execution time rather than routing to a different provider (e.g., Gemini API for questions).

**Why this approach:**
1. **Single executor** - Keeps all execution through Claude Code CLI
2. **Consistent experience** - Same model, same context understanding
3. **Flexible** - Tools can be adjusted per-task without code changes
4. **Future-proof** - Easy to add more modes (e.g., "test-only", "docs-only")

### Read-Only Tool Set

For questions, the allowed tools are:
```go
[]string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"}
```

Excluded tools:
- `Write` - No file creation
- `Edit` - No file modification
- `Bash` - No command execution (prevents `rm`, `git commit`, etc.)

## Implementation

### 1. Message Kind Flow

```
UI (kind selector)
    → messaging.InboxMessage.MessageType
    → coordinator.Message.Kind
    → coordinator.Task.Kind
    → coordinator.TaskRecord.Kind
    → executor.Task.AllowedTools
```

### 2. Files Changed

| File | Change |
|------|--------|
| `internal/executor/claude/claude.go` | Use `task.AllowedTools` if set, fallback to executor config |
| `internal/coordinator/provider.go` | Added `Kind` field to `Task` struct |
| `internal/coordinator/provider_claude.go` | Set read-only tools when `Kind == "question"` |
| `internal/coordinator/watcher.go` | Added `Kind` to `Message` struct, updated `messageToTask()` |
| `internal/coordinator/store.go` | Added `Kind` to `TaskRecord` struct |
| `internal/coordinator/daemon.go` | Propagate `Kind` through task creation pipeline |
| `internal/coordinator/message_adapter.go` | Map `MessageType` to `Kind` |

### 3. Key Code

**Executor tool selection** (`internal/executor/claude/claude.go:92-99`):
```go
// Use task-specific tools if specified, otherwise fall back to executor config
tools := e.allowedTools
if len(task.AllowedTools) > 0 {
    tools = task.AllowedTools
}
if len(tools) > 0 {
    args = append(args, "--allowedTools", strings.Join(tools, ","))
}
```

**Provider tool restriction** (`internal/coordinator/provider_claude.go:79-82`):
```go
// For questions, use read-only tools (no file modifications)
if task.Task.Kind == "question" {
    execTask.AllowedTools = []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"}
}
```

**Kind inference fallback** (`internal/coordinator/watcher.go:110-121`):
```go
// Use the message's kind directly if set
// Otherwise, infer from category type
kind := msg.Kind
if kind == "" {
    // Fallback: infer kind from category type
    // "question" or "research" categories get read-only mode
    if msg.Type == "question" || msg.Type == "research" {
        kind = "question"
    } else {
        kind = "directive"
    }
}
```

## Behavior Matrix

| Message Kind | Execution Mode | Available Tools | Can Modify Files |
|--------------|---------------|-----------------|------------------|
| `directive` | Full | Bash, Read, Write, Edit, Grep, Glob | Yes |
| `question` | Read-only | Read, Grep, Glob, WebFetch, WebSearch | No |

## UI Integration

The UI's kind selector (Directive/Question dropdown) maps directly to the `messageType` field stored in the database:

```tsx
<select
  value={messageKind}
  onChange={(e) => setMessageKind(e.target.value)}
  className="kind-selector"
  title={messageKind === 'directive'
    ? 'Directive: A task or instruction for the agent to execute'
    : 'Question: A query for information (won\'t trigger execution)'}
>
  <option value="directive">Directive</option>
  <option value="question">Question</option>
</select>
```

## Testing

```bash
# Build and run tests
go test ./internal/coordinator/... -short
go test ./internal/executor/... -short

# Verify build
make build
```

All coordinator tests pass (1.396s).

## Future Enhancements

1. **Additional modes**: Could add `test-only`, `docs-only` modes with different tool sets
2. **Tool groups**: Define named tool groups in config (e.g., "readonly", "full", "safe")
3. **Per-user restrictions**: Allow admins to restrict tools by user/role
4. **Audit logging**: Log which tools were available vs used for each execution

## Metrics

- **Lines changed:** ~50 across 7 files
- **New fields:** 3 (Message.Kind, Task.Kind, TaskRecord.Kind)
- **Breaking changes:** None (Kind defaults to "directive" if empty)

## Related

- Claude Code CLI `--allowedTools` flag
- `internal/executor/executor.go:43` - Task.AllowedTools field (pre-existing)
- Collaboration Hub UI message input component

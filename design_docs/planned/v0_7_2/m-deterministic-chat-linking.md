# M-DETERMINISTIC-CHAT-LINKING: Deterministic Task-to-Chat Message Linking

**Status:** Phase 5 Complete (Full Deterministic Chat Linking)
**Priority:** Medium
**Complexity:** Medium
**Target Version:** v0.7.2

### Implementation Status (2026-01-29)

**Session → Task Linking: COMPLETE**
- Hooks capture AILANG_* env vars and post to server
- Server stores task_id, chain_id, stage_id, message_id in sessions table
- Query: `SELECT * FROM sessions WHERE task_id = 'task-xxx'`

**Bug Fix: macOS Compatibility**
- `head -n -1` is GNU-only, replaced with `sed '$d'` for macOS

---

## Problem Statement

Currently, linking chat messages to tasks requires **timestamp-based filtering** because there's no deterministic foreign key relationship. This is fragile and causes issues:

1. **Session reuse**: Claude Code's `--resume` flag reuses session IDs across multiple tasks
2. **Manual assignment**: Testing/debugging sometimes assigns wrong session_id to tasks
3. **Time zone issues**: Timestamp comparison can fail across time zones
4. **Long-running sessions**: A single session may span hours/days with multiple tasks

### Current Data Flow (Fragile)

```
coordinator.db.tasks
┌──────────────────┬──────────────────────────────────────┬─────────────────────┐
│ id               │ session_id                            │ started_at          │
├──────────────────┼──────────────────────────────────────┼─────────────────────┤
│ task-96625348    │ 5a00dacd-3a8a-4269-a9e9-05997c726c02 │ 2026-01-29 18:02:04 │
└──────────────────┴──────────────────────────────────────┴─────────────────────┘

observatory.db.chat_messages (12,930 messages in this session!)
┌──────────────────────────────────────┬─────────────────────┬─────────┐
│ session_id                            │ timestamp           │ task_id │
├──────────────────────────────────────┼─────────────────────┼─────────┤
│ 5a00dacd-3a8a-4269-a9e9-05997c726c02 │ 2026-01-07 15:01:33 │ NULL    │  ← OLD
│ 5a00dacd-3a8a-4269-a9e9-05997c726c02 │ 2026-01-07 15:02:15 │ NULL    │  ← OLD
│ ... (messages from wrong time)       │ ...                 │ NULL    │
│ 5a00dacd-3a8a-4269-a9e9-05997c726c02 │ 2026-01-29 18:02:10 │ NULL    │  ← CORRECT
│ 5a00dacd-3a8a-4269-a9e9-05997c726c02 │ 2026-01-29 18:03:30 │ NULL    │  ← CORRECT
└──────────────────────────────────────┴─────────────────────┴─────────┘

Current workaround: Filter by timestamp range (task.started_at ± 1 minute)
```

---

## Solution Design

### Implementation Path Discovery

**Key insight**: Claude Code JSONL files do NOT capture environment variables. However, Claude Code **hooks** DO have access to the environment because they run in the same process space as Claude Code.

### Hooks-Based Approach (IMPLEMENTED)

The solution uses Claude Code hooks (`~/.ailang/hooks/claude_telemetry.sh`) which:
1. Run at SessionStart, PreToolUse, PostToolUse, and Stop events
2. Have access to environment variables including `AILANG_TASK_ID`, `AILANG_CHAIN_ID`, etc.
3. POST events to the observatory server which stores them in the sessions table

**Data Flow (Hooks-Based):**

```
Coordinator spawns Claude Code with env vars:
  AILANG_TASK_ID=task-96625348
  AILANG_CHAIN_ID=chain-abc123
  AILANG_STAGE_ID=stage-xyz789
       │
       ▼
Claude Code starts session
       │
       ▼
SessionStart hook runs (claude_telemetry.sh)
  - Reads AILANG_* env vars
  - POSTs to /api/observatory/hooks with correlation IDs
       │
       ▼
Observatory server receives hook event
  - Calls UpsertSessionWithCorrelation()
  - Stores task_id, chain_id, stage_id in sessions table
       │
       ▼
Sessions table now has deterministic link:
  session_id → task_id, chain_id, stage_id
```

---

## Implementation Status

### Phase 1: Hooks Enhancement ✅ COMPLETE

**File: `~/.ailang/hooks/claude_telemetry.sh`**

Updated to read correlation IDs from environment:
```bash
# Extract AILANG correlation IDs from environment (M-CHAINS-SIMPLIFY)
AILANG_TASK_ID="${AILANG_TASK_ID:-}"
AILANG_CHAIN_ID="${AILANG_CHAIN_ID:-}"
AILANG_STAGE_ID="${AILANG_STAGE_ID:-}"
AILANG_MESSAGE_ID="${AILANG_MESSAGE_ID:-}"
```

SessionStart payload now includes:
```json
{
  "event": "SessionStart",
  "session_id": "...",
  "workspace": "...",
  "task_id": "task-96625348",
  "chain_id": "chain-abc123",
  "stage_id": "stage-xyz789",
  "message_id": "msg-..."
}
```

### Phase 2: Schema Migration ✅ COMPLETE

**Migration v8** adds columns to `sessions` table:

```sql
ALTER TABLE sessions ADD COLUMN task_id TEXT;
ALTER TABLE sessions ADD COLUMN chain_id TEXT;
ALTER TABLE sessions ADD COLUMN stage_id TEXT;
ALTER TABLE sessions ADD COLUMN message_id TEXT;

CREATE INDEX idx_sessions_task ON sessions(task_id);
CREATE INDEX idx_sessions_chain ON sessions(chain_id);
```

### Phase 3: Server Handler ✅ COMPLETE

**File: `internal/server/handlers_hooks.go`**

- Updated `HookEvent` struct with `TaskID`, `ChainID`, `StageID`, `MessageID` fields
- SessionStart handler now calls `UpsertSessionWithCorrelation()` with correlation IDs

### Phase 4: Backend Support ✅ COMPLETE

- Added `SessionCorrelation` struct to hold optional correlation IDs
- Added `UpsertSessionWithCorrelation()` to Backend interface
- Implemented in SQLiteBackend, CompositeBackend, GCPTraceBackend, JaegerBackend

---

## Implementation Status

### Phase 5: Chat Message Linking ✅ COMPLETE

The full deterministic linking from tasks to chat messages is now implemented:

1. **Migration v9** adds columns to `chat_messages` table:
   ```sql
   ALTER TABLE chat_messages ADD COLUMN task_id TEXT;
   ALTER TABLE chat_messages ADD COLUMN chain_id TEXT;
   ALTER TABLE chat_messages ADD COLUMN stage_id TEXT;
   CREATE INDEX idx_chat_messages_task ON chat_messages(task_id);
   ```

2. **Importer propagation** (`internal/claudehistory/importer.go`):
   - Added `SessionCorrelation` struct
   - Added `getSessionCorrelation()` function to look up task_id/chain_id/stage_id from sessions table
   - Modified INSERT statement to include correlation IDs when importing messages

3. **Chains CLI** (`cmd/ailang/chains.go`):
   - Added `getChatMessagesForTask(taskID string)` function for deterministic queries
   - Updated `printStageDetails()` to prefer task_id query over timestamp filtering
   - Updated JSON export functions to use deterministic queries first
   - Fallback to timestamp-based queries when task_id not available

**Query patterns now work:**
```sql
-- Direct task->message query (DETERMINISTIC)
SELECT * FROM chat_messages WHERE task_id = 'task-96625348';

-- Still supported: timestamp fallback for historical data
SELECT cm.* FROM chat_messages cm
JOIN sessions s ON cm.session_id = s.session_id
WHERE s.task_id = 'task-96625348'
AND cm.timestamp BETWEEN s.started_at AND s.ended_at;
```

---

## Data Model After Full Implementation

### sessions table (IMPLEMENTED)

| Column | Type | Status | Description |
|--------|------|--------|-------------|
| session_id | TEXT PK | Existing | Claude Code session UUID |
| workspace | TEXT | Existing | Working directory |
| claude_version | TEXT | Existing | Claude version |
| source | TEXT | Existing | 'hook', 'otel', 'manual' |
| started_at | TIMESTAMP | Existing | Session start time |
| ended_at | TIMESTAMP | Existing | Session end time |
| turn_count | INT | Existing | Number of turns |
| **task_id** | TEXT | **NEW** | Coordinator task ID |
| **chain_id** | TEXT | **NEW** | Execution chain ID |
| **stage_id** | TEXT | **NEW** | Chain stage ID |
| **message_id** | TEXT | **NEW** | Triggering message ID |

### chat_messages table (IMPLEMENTED)

| Column | Type | Status | Description |
|--------|------|--------|-------------|
| id | TEXT PK | Existing | Message UUID |
| session_id | TEXT | Existing | Claude Code session UUID |
| **task_id** | TEXT | **IMPLEMENTED** | Coordinator task ID |
| **chain_id** | TEXT | **IMPLEMENTED** | Execution chain ID |
| **stage_id** | TEXT | **IMPLEMENTED** | Chain stage ID |
| turn_number | INT | Existing | Turn within session |
| role | TEXT | Existing | "user" or "assistant" |
| content_json | TEXT | Existing | Full message content |
| timestamp | DATETIME | Existing | Message creation time |
| tokens_in | INT | Existing | Input tokens |
| tokens_out | INT | Existing | Output tokens |

---

## Query Patterns

### Direct Query (IMPLEMENTED)

```sql
-- Get all messages for a task (DETERMINISTIC)
SELECT * FROM chat_messages WHERE task_id = 'task-96625348';

-- Get all messages for a chain (across stages)
SELECT * FROM chat_messages WHERE chain_id = 'chain-abc123' ORDER BY stage_id, turn_number;

-- Get messages for a stage
SELECT * FROM chat_messages WHERE stage_id = 'stage-xyz789' ORDER BY turn_number;
```

---

## Testing

### Verify hooks capture env vars

```bash
# Start a coordinator task with verbose logging
DEBUG_APPROVAL_WATCHER=1 ailang coordinator start

# Send a message to trigger a task
ailang messages send design-doc-creator "Test correlation" --title "Test"

# Check telemetry logs for correlation IDs
tail -f ~/.ailang/state/telemetry_hooks.log

# Should show:
# Correlation IDs: task=task-xxx chain=chain-xxx stage=stage-xxx msg=msg-xxx
```

### Verify sessions have correlation

```bash
# After task completes
sqlite3 ~/.ailang/state/observatory.db "
SELECT session_id, task_id, chain_id, stage_id
FROM sessions
WHERE task_id IS NOT NULL
ORDER BY started_at DESC LIMIT 5"
```

### Verify chat messages have correlation (Phase 5)

```bash
# First sync chat history
ailang observatory sync-chat

# Check if messages have task_id (deterministic linking)
sqlite3 ~/.ailang/state/observatory.db "
SELECT id, session_id, task_id, turn_number
FROM chat_messages
WHERE task_id IS NOT NULL
ORDER BY timestamp DESC LIMIT 10"

# Test the deterministic query via CLI
ailang chains view <chain-id> --json | jq '.stages[0].messages | length'

# Compare with timestamp-based query (should match)
ailang chains view <chain-id> --json --verbose
```

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hook timeout | Medium | Set 5s timeout; log failures but don't block Claude |
| Observatory server down | Low | Hook logs locally; can replay later |
| Missing env vars | Medium | Fallback to timestamp correlation (existing behavior) |

---

## Related Files

- `~/.ailang/hooks/claude_telemetry.sh` - Hook script (MODIFIED - Phases 1-4)
- `internal/observatory/migrate.go` - Migrations v8 + v9 (ADDED)
- `internal/observatory/store_sessions.go` - Session correlation (MODIFIED)
- `internal/server/handlers_hooks.go` - Hooks handler (MODIFIED)
- `internal/claudehistory/importer.go` - Importer with correlation propagation (MODIFIED - Phase 5)
- `cmd/ailang/chains.go` - getChatMessagesForTask + updated queries (MODIFIED - Phase 5)

---

## Implementation Complete

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1: Hooks enhancement | ✅ Done | claude_telemetry.sh captures AILANG_* env vars |
| Phase 2: Schema migration (sessions) | ✅ Done | Migration v8 adds task_id/chain_id/stage_id to sessions |
| Phase 3: Server handler | ✅ Done | UpsertSessionWithCorrelation in handlers_hooks.go |
| Phase 4: Backend support | ✅ Done | SQLiteBackend, CompositeBackend implementation |
| Phase 5a: Chat_messages schema | ✅ Done | Migration v9 adds correlation columns |
| Phase 5b: Importer propagation | ✅ Done | importer.go propagates session correlation to messages |
| Phase 5c: CLI query updates | ✅ Done | getChatMessagesForTask with fallback to timestamp |

**All phases complete. Deterministic task→chat message linking is now fully operational.**

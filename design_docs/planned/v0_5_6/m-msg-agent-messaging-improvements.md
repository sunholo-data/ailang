# M-MSG: Agent Messaging System Improvements

**Version**: v0.5.6
**Status**: Planned
**Priority**: High (user-reported pain points)
**Estimated Effort**: 2-3 days
**Breaking Change**: Yes - clean break, no migration needed

## Problem Statement

The AILANG agent messaging system has become a valuable feature for autonomous agent orchestration, but several issues hinder usability:

### P1: Messages Not Being Marked Unread Properly
- **Symptom**: Messages appear read when they shouldn't, or unread status is lost
- **Root cause**: Dual storage model (file-based + SQLite) creates race conditions
- **Impact**: Users miss important agent notifications

### P2: Difficult to Read Messages for Agents
- **Symptom**: CLI UX is confusing, flag order matters, inconsistent behavior
- **Root cause**: Organic growth without unified design
- **Impact**: Users struggle with basic operations

### P3: Inconsistent Inbox Locations
- **User inbox**: `~/.ailang/state/messages/inbox/user/`
- **Claude-code inbox**: `.ailang/state/messages/claude-code/` (project-relative!)
- **Agent inboxes**: `~/.ailang/state/messages/<agent>/`
- **Impact**: Confusing mental model, easy to look in wrong place

### P4: No Message Cleanup/Expiration
- Messages accumulate forever
- DLQ entries never cleaned up
- Home directory grows indefinitely

## Current Architecture Analysis

### Three Storage Systems (Problem!)

```
┌─────────────────────────────────────────────────────────────────┐
│                    CURRENT (Inconsistent)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. File-Based (agentprotocol)                                 │
│     ~/.ailang/state/messages/                                  │
│     ├── inbox/user/{_unread,_read,_archive}/                   │
│     ├── claude-code/*.pending.json                             │
│     └── <agent>/*.pending.json                                 │
│                                                                 │
│  2. SQLite agents.db (agentprotocol)                           │
│     ~/.ailang/state/agents.db                                  │
│     - delivery_state: pending/claimed/acked                    │
│     - Used for dedup, leases, metrics                          │
│                                                                 │
│  3. SQLite collaboration.db (messaging)                        │
│     ~/.ailang/state/collaboration.db                           │
│     - Thread-based messaging                                   │
│     - Separate from above!                                     │
│                                                                 │
│  Problem: `inbox` reads files, `ack` tries SQLite first,       │
│           status can be out of sync between stores!            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Read/Unread Status Tracking (Inconsistent!)

| Inbox Type | Unread Location | Read Location | Transition |
|------------|-----------------|---------------|------------|
| User | `_unread/msg.json` | `_read/msg.json` | File move |
| Claude-code | `*.pending.json` | `_processed/msg.json` | File move |
| Agent | `*.pending.json` | `_processed/msg.json` | File move |
| SQLite | `delivery_state='pending'` | `delivery_state='acked'` | DB update |

**Problem**: File move and DB update aren't atomic!

## Proposed Solution

### Phase 1: Unify Storage Model (SQLite-First)

**Decision**: SQLite as source of truth, files as optional export.

```
┌─────────────────────────────────────────────────────────────────┐
│                    PROPOSED (Unified)                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PRIMARY: SQLite messages.db                                   │
│  ~/.ailang/state/messages.db                                   │
│                                                                 │
│  Tables:                                                       │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ messages                                                  │  │
│  │ - id (UUID)                                              │  │
│  │ - message_id (external ID)                               │  │
│  │ - correlation_id                                         │  │
│  │ - from_agent                                             │  │
│  │ - to_inbox (user|claude-code|sprint-planner|...)        │  │
│  │ - message_type (request|response|notification)           │  │
│  │ - payload (JSON)                                         │  │
│  │ - status (unread|read|archived|deleted)                  │  │
│  │ - created_at                                             │  │
│  │ - read_at (nullable)                                     │  │
│  │ - expires_at (nullable)                                  │  │
│  │ - retry_count                                            │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│  OPTIONAL: File export (for debugging/backup)                  │
│  ~/.ailang/state/messages/export/                              │
│  - On-demand export via `ailang agent export`                  │
│  - Not used for status tracking                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Clean Break** (no migration):
1. Create new `messages.db` with unified schema
2. Delete old storage on first run:
   - `~/.ailang/state/messages/` (file-based)
   - `~/.ailang/state/agents.db` (old SQLite)
   - `~/.ailang/state/collaboration.db` (old SQLite)
3. Start fresh with empty message database
4. Remove old `internal/agentprotocol/` package entirely
5. Simplify `internal/messaging/` to single implementation

### Phase 2: Simplified CLI Interface

#### Current (Confusing)

```bash
# Flags MUST come before positional args (non-standard!)
ailang agent inbox --unread-only user     # Works
ailang agent inbox user --unread-only     # FAILS silently!

# Different inbox locations
ailang agent inbox user                    # ~/.ailang/state/messages/inbox/user/
ailang agent inbox claude-code             # .ailang/state/messages/claude-code/
```

#### Proposed (Intuitive)

```bash
# Subcommand-based, flags anywhere
ailang messages list                       # All messages (all inboxes)
ailang messages list --inbox user          # Filter by inbox
ailang messages list --unread              # Filter by status
ailang messages list --from sprint-planner # Filter by sender

# Short aliases
ailang msg ls                              # Same as list
ailang msg ls -u                           # Unread only
ailang msg ls -i user -u                   # User inbox, unread

# Read a specific message
ailang messages read MSG_ID                # Mark as read, show full content
ailang messages read MSG_ID --peek         # Show without marking read

# Acknowledge (mark as read)
ailang messages ack MSG_ID                 # Single message
ailang messages ack --all                  # All unread
ailang messages ack --inbox user --all     # All unread in user inbox

# Unacknowledge (mark as unread)
ailang messages unack MSG_ID

# Archive/Delete
ailang messages archive MSG_ID
ailang messages delete MSG_ID              # Soft delete (status='deleted')
ailang messages delete MSG_ID --hard       # Permanent delete

# Send
ailang messages send USER_INBOX "title" "body"
ailang messages send sprint-planner --json '{"type":"handoff",...}'

# Watch mode (real-time updates)
ailang messages watch                      # All inboxes
ailang messages watch --inbox user         # Specific inbox

# Cleanup
ailang messages cleanup --older-than 30d   # Archive old messages
ailang messages cleanup --expired          # Delete expired messages
ailang messages gc                         # Garbage collect deleted messages
```

#### Breaking Change

Old commands removed entirely:
```bash
# REMOVED - these no longer exist
ailang agent inbox user           # ❌ Use: ailang messages list --inbox user
ailang agent ack MSG_ID           # ❌ Use: ailang messages ack MSG_ID
ailang agent send agent '{...}'   # ❌ Use: ailang messages send agent --json '{...}'
ailang agent top                  # ❌ Removed
ailang agent dlq                  # ❌ Removed (DLQ replaced by retry_count in messages table)
```

The `ailang agent` subcommand is removed. All messaging goes through `ailang messages`.

### Phase 3: Consistent Status Tracking

#### Status State Machine

```
                    ┌──────────────┐
                    │   UNREAD     │
                    │  (default)   │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │   READ   │  │ ARCHIVED │  │ EXPIRED  │
       │          │  │          │  │  (auto)  │
       └────┬─────┘  └────┬─────┘  └────┬─────┘
            │             │             │
            │             ▼             │
            │       ┌──────────┐        │
            └──────►│ DELETED  │◄───────┘
                    │  (soft)  │
                    └────┬─────┘
                         │ --hard
                         ▼
                    ┌──────────┐
                    │ PURGED   │
                    │(removed) │
                    └──────────┘
```

#### Atomic Transitions

All status changes via single SQL transaction:
```sql
BEGIN TRANSACTION;
UPDATE messages SET status = 'read', read_at = CURRENT_TIMESTAMP
WHERE message_id = ? AND status = 'unread';
COMMIT;
```

No more file moves + DB updates!

### Phase 4: Message Expiration & Cleanup

#### Auto-Expiration

Messages can have optional `expires_at`:
```bash
ailang messages send agent --ttl 24h "Check this"
# Creates message with expires_at = now + 24h
```

Background cleanup (via cron or daemon):
```bash
# Add to crontab
0 * * * * ailang messages cleanup --expired --quiet
```

Or integrated into session start hook.

#### Retention Policy

Config in `~/.ailang/config.yaml`:
```yaml
messages:
  retention:
    read: 30d        # Auto-archive after 30 days
    archived: 90d    # Auto-delete after 90 days
    deleted: 7d      # Purge soft-deleted after 7 days
  cleanup:
    on_session_start: true   # Run cleanup at session start
    max_messages: 1000       # Keep max 1000 messages per inbox
```

### Phase 5: Unified Inbox View

#### All-In-One Dashboard

```bash
$ ailang messages

╭─────────────────────────────────────────────────────────────╮
│                    AILANG MESSAGES                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  📬 UNREAD: 3 messages                                     │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [user] sprint-executor • 2 min ago                  │   │
│  │ Sprint M-GAME-B completed (3/3 milestones)          │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [claude-code] stapledons_voyage • 1 hour ago        │   │
│  │ Feature request: Add save game support              │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [sprint-planner] design-doc-creator • 3 hours ago   │   │
│  │ Design doc ready for M-DX15                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  📖 Recent (read): 12 messages                             │
│  📦 Archived: 45 messages                                  │
│                                                             │
│  Commands:                                                  │
│    r <n>     Read message n                                │
│    a <n>     Archive message n                             │
│    A         Archive all read                              │
│    q         Quit                                          │
│                                                             │
╰─────────────────────────────────────────────────────────────╯
```

#### Correlation ID Grouping

Messages with same correlation_id grouped as thread:
```bash
$ ailang messages list --group-by correlation

Sprint M-GAME-B (correlation: sprint_M-GAME-B)
├── [unread] Milestone 1 complete (2h ago)
├── [unread] Milestone 2 complete (1h ago)
└── [unread] Sprint complete (30m ago)

Sprint M-DX14 (correlation: sprint_M-DX14)
├── [read] Plan ready (yesterday)
└── [read] Sprint complete (yesterday)
```

## Implementation Plan

### Milestone 1: Clean Slate & Schema (0.5 day)

**Files to DELETE**:
- `internal/agentprotocol/` - Entire package (file-based messaging)
- `cmd/ailang/agent.go` - Old agent commands
- `cmd/ailang/agent_inbox.go` - Old inbox display
- `cmd/ailang/agent_ack.go` - Old ack/unack
- `cmd/ailang/agent_helpers.go` - Old utilities

**Files to create/modify**:
- `internal/messaging/store.go` - Rewrite with unified schema
- `internal/messaging/types.go` - Clean message types

**Tasks**:
- [ ] Delete old `internal/agentprotocol/` package
- [ ] Delete old `cmd/ailang/agent*.go` files
- [ ] Design unified messages table schema
- [ ] Implement auto-cleanup of old storage on first run
- [ ] Write tests for new store

### Milestone 2: Core CLI Commands (1 day)

**Files to create**:
- `cmd/ailang/messages.go` - Command router
- `cmd/ailang/messages_list.go` - List/filter messages
- `cmd/ailang/messages_ops.go` - ack/unack/archive/delete
- `cmd/ailang/messages_send.go` - Send messages

**Tasks**:
- [ ] Implement `ailang messages list` with filters
- [ ] Implement `ailang messages ack/unack`
- [ ] Implement `ailang messages send`
- [ ] Implement `ailang messages read` (show full + mark read)
- [ ] Add short aliases (`ailang msg ls -u`)
- [ ] Write tests

### Milestone 3: Cleanup & Watch Mode (0.5 day)

**Files to create**:
- `cmd/ailang/messages_cleanup.go` - Cleanup/GC
- `cmd/ailang/messages_watch.go` - Real-time updates

**Tasks**:
- [ ] Implement `ailang messages cleanup`
- [ ] Implement TTL expiration
- [ ] Implement `ailang messages watch` (polling-based)
- [ ] Update session start hook for new CLI
- [ ] Write tests

### Milestone 4: Dashboard & Polish (0.5 day)

**Files to create/modify**:
- `cmd/ailang/messages_dashboard.go` - Interactive view

**Tasks**:
- [ ] Create unified dashboard view (`ailang messages`)
- [ ] Add correlation ID grouping (`--group-by correlation`)
- [ ] Polish output formatting
- [ ] Write tests

### Milestone 5: Documentation (0.5 day)

**Files to modify**:
- `CLAUDE.md` - Update message commands section (remove old, add new)
- `scripts/hooks/session_start.sh` - Update for new CLI
- `.claude/skills/agent-inbox/` - Update or remove skill

**Tasks**:
- [ ] Update CLAUDE.md with new commands
- [ ] Update session start hook
- [ ] Update any skills that use messaging
- [ ] Release notes for v0.5.6

## Success Criteria

1. **Messages correctly tracked**: No more "ghost" unread/read states
2. **Single source of truth**: All reads/writes go through `messages.db`
3. **Intuitive CLI**: Users can figure out commands without docs
4. **Automatic cleanup**: Old messages don't accumulate forever
5. **Less code**: Delete more than we add (remove ~2000 lines, add ~800)

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| SQLite performance | Low | Low | Indexes on status, inbox, created_at |
| Watch mode complexity | Medium | Low | Start with polling, optimize later |
| Skills/hooks break | Medium | Low | Update in same release |

## Alternatives Considered

### A. Keep File-Based, Fix Bugs
- **Pros**: Less work
- **Cons**: Root cause (dual storage) not addressed, will recur

### B. Migrate Existing Messages
- **Pros**: Preserve history
- **Cons**: Complex migration logic, more code, edge cases

### C. External Message Queue (Redis, etc.)
- **Pros**: Battle-tested, scalable
- **Cons**: New dependency, overkill for single-user CLI

**Decision**: SQLite-only with clean break. Delete old code/data, start fresh.
Simplest approach, least code, no migration complexity.

## Related Work

- [agent-inbox-acknowledgment.md](../implemented/v0_3_14/agent-inbox-acknowledgment.md) - Original ack design
- [agent-inbox-sessionstart-hook.md](../implemented/v0_3_14/agent-inbox-sessionstart-hook.md) - Hook design
- [internal/agentprotocol/](../../internal/agentprotocol/) - Current implementation
- [internal/messaging/](../../internal/messaging/) - Alternative implementation

## Appendix: Schema DDL

```sql
-- Primary messages table
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    correlation_id TEXT,
    parent_message_id TEXT,

    from_agent TEXT NOT NULL,
    to_inbox TEXT NOT NULL,
    message_type TEXT NOT NULL CHECK (message_type IN ('request', 'response', 'notification')),

    title TEXT,
    payload TEXT NOT NULL,  -- JSON

    status TEXT NOT NULL DEFAULT 'unread'
        CHECK (status IN ('unread', 'read', 'archived', 'deleted')),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    archived_at TIMESTAMP,
    deleted_at TIMESTAMP,
    expires_at TIMESTAMP,

    retry_count INTEGER DEFAULT 0,
    last_error TEXT,

    FOREIGN KEY (parent_message_id) REFERENCES messages(message_id)
);

-- Indexes for common queries
CREATE INDEX idx_messages_inbox_status ON messages(to_inbox, status);
CREATE INDEX idx_messages_correlation ON messages(correlation_id);
CREATE INDEX idx_messages_created ON messages(created_at DESC);
CREATE INDEX idx_messages_expires ON messages(expires_at) WHERE expires_at IS NOT NULL;

-- Inbox configuration table
CREATE TABLE inbox_config (
    inbox_name TEXT PRIMARY KEY,
    retention_days INTEGER DEFAULT 30,
    auto_archive_days INTEGER DEFAULT 7,
    max_messages INTEGER DEFAULT 1000
);

-- Insert defaults
INSERT INTO inbox_config (inbox_name) VALUES ('user'), ('claude-code');
```

---

**Author**: Claude
**Created**: 2024-12-04
**Last Updated**: 2024-12-04

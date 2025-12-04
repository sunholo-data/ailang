# M-MSG Sprint Plan: Agent Messaging System Rewrite

**Sprint ID**: M-MSG
**Version**: v0.5.6
**Duration**: 2-3 days
**Risk Level**: Low (clean break, no migration)

## Sprint Summary

**Goal**: Replace the inconsistent dual-storage messaging system with a unified SQLite-only implementation and intuitive CLI.

**Key Deliverables**:
1. Delete old `internal/agentprotocol/` and `cmd/ailang/agent*.go`
2. Implement unified `messages.db` SQLite store
3. New `ailang messages` CLI with intuitive subcommands
4. Watch mode and cleanup functionality
5. Updated session start hook

**Breaking Change**: All `ailang agent *` commands removed. New `ailang messages *` commands.

## Velocity Analysis

**Recent velocity** (from CHANGELOG):
- M-TYPE1 fix: ~290 LOC (parser + codegen) - 1 day
- M-DX26: ~270 LOC (typed wrappers) - 1 day
- Compile DX: ~100 LOC (directory support) - 0.5 day

**Average**: ~200-300 LOC/day for focused work

**This sprint estimate**:
- Delete ~2000 LOC (old code)
- Add ~800 LOC (new implementation)
- Net: -1200 LOC (codebase gets simpler!)

## Milestones

### M1: Clean Slate (0.5 day, ~50 LOC)

**Tasks**:
1. Delete `internal/agentprotocol/` package entirely
2. Delete `cmd/ailang/agent.go`, `agent_inbox.go`, `agent_ack.go`, `agent_helpers.go`
3. Remove any imports/references to deleted packages
4. Verify `make build` still works

**Acceptance Criteria**:
- [ ] No `agentprotocol` package in codebase
- [ ] No `agent*.go` files in `cmd/ailang/`
- [ ] `make build` succeeds
- [ ] `make test` passes (or tests removed with packages)

**Risk**: Low - just deletion

---

### M2: SQLite Message Store (1 day, ~300 LOC)

**Tasks**:
1. Rewrite `internal/messaging/store.go` with unified schema
2. Implement message types in `internal/messaging/types.go`
3. Add auto-cleanup of old storage on first run:
   - Delete `~/.ailang/state/messages/` directory
   - Delete `~/.ailang/state/agents.db`
   - Delete `~/.ailang/state/collaboration.db`
4. Create `~/.ailang/state/messages.db` with new schema

**Files**:
- `internal/messaging/store.go` (~200 LOC) - SQLite operations
- `internal/messaging/types.go` (~50 LOC) - Message struct
- `internal/messaging/store_test.go` (~100 LOC) - Tests

**Schema** (from design doc):
```sql
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    correlation_id TEXT,
    from_agent TEXT NOT NULL,
    to_inbox TEXT NOT NULL,
    message_type TEXT NOT NULL,
    title TEXT,
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unread',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    expires_at TIMESTAMP
);
CREATE INDEX idx_messages_inbox_status ON messages(to_inbox, status);
```

**Acceptance Criteria**:
- [ ] `messages.db` created on first use
- [ ] Old storage directories deleted on first use
- [ ] Insert/Query/Update operations work
- [ ] Tests pass

**Risk**: Low - straightforward SQLite

---

### M3: Core CLI Commands (0.5 day, ~250 LOC)

**Tasks**:
1. Create `cmd/ailang/messages.go` - command router
2. Implement `messages list` with filters (--inbox, --unread, --from)
3. Implement `messages ack` and `messages unack`
4. Implement `messages send`
5. Implement `messages read` (show full content, mark as read)

**Files**:
- `cmd/ailang/messages.go` (~150 LOC) - Router + list
- `cmd/ailang/messages_ops.go` (~100 LOC) - ack/unack/send/read

**CLI Examples**:
```bash
ailang messages list                    # All messages
ailang messages list --unread           # Unread only
ailang messages list --inbox user       # Filter by inbox
ailang messages ack MSG_ID              # Mark as read
ailang messages ack --all               # Ack all unread
ailang messages send user "Hello"       # Send message
ailang messages read MSG_ID             # Show full + mark read
```

**Acceptance Criteria**:
- [ ] `ailang messages list` works
- [ ] `ailang messages ack MSG_ID` marks message as read
- [ ] `ailang messages send inbox "msg"` creates message
- [ ] `ailang messages read MSG_ID` shows full content
- [ ] Short alias `ailang msg ls` works

**Risk**: Low - standard CLI patterns

---

### M4: Watch & Cleanup (0.5 day, ~150 LOC)

**Tasks**:
1. Implement `messages watch` (polling-based, 1s interval)
2. Implement `messages cleanup --older-than 30d`
3. Implement TTL expiration check
4. Update session start hook to use new CLI

**Files**:
- `cmd/ailang/messages_watch.go` (~80 LOC)
- `cmd/ailang/messages_cleanup.go` (~70 LOC)
- `scripts/hooks/session_start.sh` (update)

**Acceptance Criteria**:
- [ ] `ailang messages watch` shows new messages in real-time
- [ ] `ailang messages cleanup --older-than 7d` removes old messages
- [ ] Session start hook uses `ailang messages list --unread`

**Risk**: Medium - watch mode needs testing

---

### M5: Documentation & Polish (0.5 day, ~50 LOC)

**Tasks**:
1. Update CLAUDE.md - replace old agent commands with new messages commands
2. Add `--help` text to all commands
3. Polish output formatting (colors, alignment)
4. Remove/update `agent-inbox` skill

**Files**:
- `CLAUDE.md` (update)
- `.claude/skills/agent-inbox/` (remove or update)
- Help text in CLI files

**Acceptance Criteria**:
- [ ] CLAUDE.md documents new `ailang messages` commands
- [ ] `ailang messages --help` is informative
- [ ] Old documentation removed

**Risk**: Low - documentation

---

## Day-by-Day Breakdown

### Day 1 (Morning)
- **M1**: Delete old packages (0.5 day)
- Start **M2**: Create store.go schema

### Day 1 (Afternoon)
- Complete **M2**: SQLite store + tests (0.5 day)
- Start **M3**: CLI router

### Day 2 (Morning)
- Complete **M3**: Core CLI commands (0.5 day)
- **M4**: Watch & cleanup (0.5 day)

### Day 2 (Afternoon)
- **M5**: Documentation & polish (0.5 day)
- Final testing & release prep

## Success Metrics

| Metric | Target |
|--------|--------|
| Old code deleted | ~2000 LOC |
| New code added | ~800 LOC |
| Net change | -1200 LOC |
| Test coverage | 80%+ for new code |
| Commands working | 8 (list, ack, unack, send, read, watch, cleanup, gc) |

## Dependencies

- None - clean break from old system

## Open Questions

None - design doc approved, clean break confirmed.

---

**Created**: 2024-12-04
**Status**: Ready for execution

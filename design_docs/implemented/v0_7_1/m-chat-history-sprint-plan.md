# M-CHAT-HISTORY Sprint Plan

**Sprint ID:** M-CHAT-HISTORY
**Duration:** 2-3 days (estimated 12-16 hours)
**Risk Level:** Low (read-only feature, no schema changes)
**Design Doc:** [m-chat-history-integration.md](m-chat-history-integration.md)

## Sprint Goal

Integrate Claude Code conversation history from `~/.claude/projects/` into the AILANG dashboard with:
- Full conversation replay view
- Trace-to-chat correlation
- Semantic search across conversations

## Velocity Analysis

**Recent velocity (last 14 days):**
- M-TRACE-TEST: ~415 LOC in ~1 day
- Observatory refactoring: ~1800 LOC in ~3 days
- Estimated sustainable pace: **200-300 LOC/day**

**Sprint estimate:**
- Total LOC: ~1,630 LOC
- At 250 LOC/day: ~6.5 days
- Aggressive pace (400 LOC/day): ~4 days
- **Target:** 3 days with focused effort

## Milestones

### M1: Backend Reader (4 hours, ~300 LOC)

**Goal:** Create Go package to read Claude Code JSONL files

**Tasks:**
- [ ] Create `internal/claudehistory/` package structure
- [ ] Implement `models.go` with Session, Message, ContentBlock types (~100 LOC)
- [ ] Implement `reader.go` with ListProjects, ListSessions, GetSession (~150 LOC)
- [ ] Add unit tests for JSONL parsing (~50 LOC)

**Acceptance Criteria:**
- [ ] Can list all projects in `~/.claude/projects/`
- [ ] Can list sessions for a project
- [ ] Can parse full session with all message types
- [ ] Handles thinking, text, tool_use, tool_result content blocks
- [ ] Unit tests pass

**Files to create:**
| File | LOC | Purpose |
|------|-----|---------|
| `internal/claudehistory/models.go` | ~100 | Data structures |
| `internal/claudehistory/reader.go` | ~150 | JSONL parsing |
| `internal/claudehistory/reader_test.go` | ~50 | Unit tests |

---

### M2: API Endpoints (3 hours, ~200 LOC)

**Goal:** Expose chat history via REST API

**Tasks:**
- [ ] Create `internal/server/handlers_claudehistory.go`
- [ ] Implement 6 endpoints (projects, sessions, session, messages, search, by-span)
- [ ] Register routes in `server.go`
- [ ] Test with curl

**Acceptance Criteria:**
- [ ] `GET /api/claude-history/projects` lists projects
- [ ] `GET /api/claude-history/sessions?project=X` lists sessions
- [ ] `GET /api/claude-history/session/{id}` returns full conversation
- [ ] `GET /api/claude-history/by-span/{spanId}` correlates via session.id
- [ ] Pagination works for large sessions

**Files to create/modify:**
| File | LOC | Purpose |
|------|-----|---------|
| `internal/server/handlers_claudehistory.go` | ~200 | API handlers |
| `internal/server/server.go` | +10 | Route registration |

---

### M3: Chat View Components (5 hours, ~630 LOC)

**Goal:** Display conversations in dashboard

**Tasks:**
- [ ] Create ChatHistory main component (~300 LOC)
- [ ] Create ChatMessage component (~150 LOC)
- [ ] Create ThinkingBlock collapsible component (~80 LOC)
- [ ] Create ToolCallBlock component (~100 LOC)
- [ ] Create useChatHistory hook (~100 LOC)
- [ ] Add "Chat" option to AggregationNav
- [ ] Wire up to ControlPlane

**Acceptance Criteria:**
- [ ] Can browse and select sessions
- [ ] Messages display with proper user/assistant styling
- [ ] Thinking blocks collapse/expand
- [ ] Tool calls show name, input, output
- [ ] Token usage displayed per message
- [ ] Timestamps visible

**Files to create/modify:**
| File | LOC | Purpose |
|------|-----|---------|
| `ui/.../ChatHistory/ChatHistory.tsx` | ~300 | Main view |
| `ui/.../ChatHistory/ChatMessage.tsx` | ~150 | Message bubble |
| `ui/.../ChatHistory/ThinkingBlock.tsx` | ~80 | Collapsible thinking |
| `ui/.../ChatHistory/ToolCallBlock.tsx` | ~100 | Tool visualization |
| `ui/.../hooks/useChatHistory.ts` | ~100 | Data fetching |
| `ui/.../AggregationNav.tsx` | +5 | Add Chat nav |
| `ui/.../ControlPlane.tsx` | +20 | Wire up ChatHistory |

---

### M4: Trace Integration (2 hours, ~200 LOC)

**Goal:** Link spans to chat context

**Tasks:**
- [ ] Add "Chat" button to TraceWaterfall for spans with session.id
- [ ] Create ChatContextPanel sidebar component
- [ ] Update DetailPanel to show chat context
- [ ] Test full workflow: click span → see chat

**Acceptance Criteria:**
- [ ] Spans with session.id attribute show "Chat" button
- [ ] Clicking opens ChatContextPanel
- [ ] Panel shows messages within span's time window
- [ ] Auto-scrolls to relevant message

**Files to create/modify:**
| File | LOC | Purpose |
|------|-----|---------|
| `ui/.../ChatHistory/ChatContextPanel.tsx` | ~100 | Sidebar panel |
| `ui/.../TraceWaterfall.tsx` | +30 | Add Chat button |
| `ui/.../DetailPanel.tsx` | +20 | Show context panel |

---

### M5: Search Integration (3 hours, ~300 LOC)

**Goal:** Search across all conversations

**Tasks:**
- [ ] Implement SimHash-based search in `search.go` (~150 LOC)
- [ ] Create ChatSearch UI component (~150 LOC)
- [ ] Add search endpoint
- [ ] Test search functionality

**Acceptance Criteria:**
- [ ] Can search by keyword across all sessions
- [ ] Results show context snippets
- [ ] Click result opens full conversation
- [ ] Filters work (project, model, date range)

**Files to create:**
| File | LOC | Purpose |
|------|-----|---------|
| `internal/claudehistory/search.go` | ~150 | SimHash search |
| `ui/.../ChatHistory/ChatSearch.tsx` | ~150 | Search UI |

---

## Day-by-Day Plan

### Day 1 (Morning): M1 - Backend Reader
- Create `internal/claudehistory/` package
- Implement models and reader
- Write and pass unit tests
- **Checkpoint:** Can read sessions from CLI test

### Day 1 (Afternoon): M2 - API Endpoints
- Create handlers_claudehistory.go
- Register routes
- Test with curl
- **Checkpoint:** API responds with real data

### Day 2 (Morning): M3 - Chat View Components
- Create ChatHistory, ChatMessage, ThinkingBlock, ToolCallBlock
- Create useChatHistory hook
- Wire to ControlPlane
- **Checkpoint:** Can view conversations in dashboard

### Day 2 (Afternoon): M4 - Trace Integration
- Add Chat button to TraceWaterfall
- Create ChatContextPanel
- Update DetailPanel
- **Checkpoint:** Click span → see chat context

### Day 3: M5 - Search + Polish
- Implement SimHash search
- Create ChatSearch UI
- Integration testing
- Bug fixes and polish
- **Checkpoint:** Search works, all features integrated

## Success Criteria

- [ ] All 5 milestones complete
- [ ] Can view full conversation for any session
- [ ] Thinking blocks display and collapse correctly
- [ ] Tool calls show input/output
- [ ] "Chat" button appears on spans with session.id
- [ ] Search finds conversations by content
- [ ] All existing tests pass
- [ ] New tests for claudehistory package

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| JSONL format varies | Parse defensively, skip unknown fields |
| Large sessions slow | Pagination + lazy loading |
| session.id missing from some spans | Graceful degradation, hide button |
| Search too slow | SimHash for fast filtering, limit results |

## Dependencies

- None (builds on existing observatory/dashboard)
- Uses existing SimHash logic from `_simhash` builtin

## Open Questions

None - design doc covers all details.

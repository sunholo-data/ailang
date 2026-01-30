# M-CHAT-HISTORY: Claude Code Conversation History Integration

**Status:** Planned
**Target:** v0.7.1
**Priority:** P1 (Medium-High)
**Estimated:** 2-3 days
**Dependencies:** None (builds on existing observatory/dashboard)
**Created:** 2026-01-25

## Problem Statement

Claude Code stores rich conversation history locally (`~/.claude/projects/`) but this data is invisible to the AILANG dashboard/observatory. Users can see OTEL spans with timing and costs, but cannot see the actual conversations that generated those spans.

**Current limitations:**
- No way to see what Claude was thinking during a task
- Tool calls in traces show names but not full input/output
- Session IDs in spans aren't linked to actual conversation content
- ~998MB of valuable conversation data is unused

**Who is affected:**
- Developers debugging agent behavior
- Users reviewing task outcomes
- Anyone wanting to understand "what did Claude actually do?"

## Discovery: Claude Code Storage Format

**Location:** `~/.claude/projects/<escaped-path>/<session-id>.jsonl`

**Example directory structure:**
```
~/.claude/projects/
├── -Users-mark-dev-sunholo-ailang/
│   ├── 0bb0373e-419b-4a7f-8f49-61cd644841cb.jsonl  # Full session
│   ├── agent-a70ccf7.jsonl                          # Subagent session
│   └── ...
```

**JSONL Message Format:**
```json
{
  "sessionId": "0bb0373e-419b-4a7f-8f49-61cd644841cb",
  "type": "assistant",
  "parentUuid": "e03dec00-678f-4b8b-8e33-ebb9ba2c336e",
  "timestamp": "2026-01-25T18:50:11.440Z",
  "message": {
    "model": "claude-opus-4-5-20251101",
    "id": "msg_01DivSAFQUdsgLtXRR2dCR4m",
    "role": "assistant",
    "content": [
      { "type": "thinking", "thinking": "Claude's internal reasoning..." },
      { "type": "text", "text": "Response to user..." },
      { "type": "tool_use", "name": "Read", "input": {"file_path": "..."} }
    ],
    "usage": {
      "input_tokens": 10,
      "output_tokens": 7,
      "cache_read_input_tokens": 17255,
      "cache_creation_input_tokens": 31163
    }
  },
  "requestId": "req_011CXUbeSaLfX5x4CPaMzzoj",
  "gitBranch": "dev",
  "cwd": "/Users/mark/dev/sunholo/ailang"
}
```

**Key correlation field:** `sessionId` matches OTEL span attribute `session.id`

**Content block types:**
| Type | Description |
|------|-------------|
| `thinking` | Claude's internal reasoning (collapsible in UI) |
| `text` | Response text to user |
| `tool_use` | Tool call with name and input |
| `tool_result` | Tool output (in subsequent message) |

## Goals

**Primary Goal:** Make Claude Code conversation history visible and searchable in the AILANG dashboard.

**Success Metrics:**
1. View full conversation for any session from dashboard
2. Click span in trace → see corresponding chat context
3. Search across all conversations by content
4. Display thinking blocks, tool calls, token usage
5. Correlate 100% of sessions with OTEL traces via `session.id`

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Dashboard (React)                        │
├─────────────────────────────────────────────────────────────┤
│  ChatHistory   │ ChatSearch  │ ChatContextPanel            │
│  (full view)   │ (search)    │ (trace sidebar)             │
└───────┬────────┴──────┬──────┴──────────┬───────────────────┘
        │               │                 │
        ▼               ▼                 ▼
┌─────────────────────────────────────────────────────────────┐
│                   Go HTTP Server                            │
│  /api/claude-history/sessions                               │
│  /api/claude-history/session/{id}                           │
│  /api/claude-history/search                                 │
│  /api/claude-history/by-span/{spanId}                       │
└───────┬─────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/claudehistory/                        │
│  reader.go     - JSONL parsing                              │
│  models.go     - Data structures                            │
│  search.go     - SimHash + neural search                    │
└───────┬─────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│           ~/.claude/projects/<path>/<session>.jsonl         │
└─────────────────────────────────────────────────────────────┘
```

### Phase 1: Backend Reader (~200 LOC)

Create `internal/claudehistory/` package:

```go
// reader.go
package claudehistory

type Reader struct {
    baseDir string // ~/.claude/projects/
}

func NewReader() *Reader {
    homeDir, _ := os.UserHomeDir()
    return &Reader{baseDir: filepath.Join(homeDir, ".claude", "projects")}
}

func (r *Reader) ListProjects() ([]string, error)
func (r *Reader) ListSessions(projectPath string) ([]SessionMeta, error)
func (r *Reader) GetSession(sessionID string) (*Session, error)
func (r *Reader) GetSessionByPath(path string) (*Session, error)
```

```go
// models.go
type Session struct {
    ID          string
    ProjectPath string
    Messages    []Message
    StartTime   time.Time
    EndTime     time.Time
    TurnCount   int
    TokensIn    int
    TokensOut   int
}

type Message struct {
    UUID       string
    ParentUUID string
    Type       string // "user", "assistant"
    Timestamp  time.Time
    Model      string
    Content    []ContentBlock
    Usage      *TokenUsage
    RequestID  string
    GitBranch  string
}

type ContentBlock struct {
    Type     string // "thinking", "text", "tool_use", "tool_result"
    Text     string
    Thinking string
    ToolUse  *ToolUseBlock
    ToolResult *ToolResultBlock
}

type TokenUsage struct {
    InputTokens          int
    OutputTokens         int
    CacheReadTokens      int
    CacheCreationTokens  int
}
```

### Phase 2: API Endpoints (~200 LOC)

Add to `internal/server/handlers_claudehistory.go`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/claude-history/projects` | GET | List all projects with session counts |
| `/api/claude-history/sessions` | GET | List sessions (with project filter) |
| `/api/claude-history/session/{id}` | GET | Get full conversation |
| `/api/claude-history/session/{id}/messages` | GET | Paginated messages |
| `/api/claude-history/search` | GET | Search conversations |
| `/api/claude-history/by-span/{spanId}` | GET | Get chat for a span |

**Correlation logic for by-span:**
```go
func (h *Handler) handleBySpan(w http.ResponseWriter, r *http.Request) {
    spanID := chi.URLParam(r, "spanId")
    span, _ := h.observatory.GetSpan(ctx, spanID)

    // Extract session.id from span attributes
    sessionID := span.Attributes["session.id"]
    if sessionID == "" {
        writeError(w, 404, "No session.id in span")
        return
    }

    session, _ := h.claudeHistory.GetSession(sessionID)

    // Filter messages to span's time window
    start, end := span.StartTime, span.EndTime
    messages := filterByTimeRange(session.Messages, start, end)

    writeJSON(w, messages)
}
```

### Phase 3: Dashboard Components (~780 LOC)

**New files in `ui/src/features/controlplane/components/ChatHistory/`:**

1. **ChatHistory.tsx** (~300 LOC) - Main chat replay view
   - Session selector dropdown
   - Chronological message list
   - Stats header (tokens, cost, duration)
   - Theme-aware styling

2. **ChatMessage.tsx** (~150 LOC) - Message bubble
   - User vs assistant styling
   - Model badge
   - Timestamp
   - Token count

3. **ThinkingBlock.tsx** (~80 LOC) - Collapsible thinking
   - Collapsed preview (first 100 chars)
   - "Show thinking" toggle
   - Monospace font

4. **ToolCallBlock.tsx** (~100 LOC) - Tool visualization
   - Tool name badge with icon
   - Collapsible input JSON
   - Collapsible output
   - Success/error indicator

5. **ChatSearch.tsx** (~150 LOC) - Search interface
   - Query input with debounce
   - Project/model filters
   - Results with context snippets
   - Click to open full conversation

**New hook: `useChatHistory.ts`** (~100 LOC)
```typescript
interface UseChatHistoryOptions {
  sessionId?: string;
  spanId?: string;
  search?: string;
  limit?: number;
}

export function useChatHistory(options: UseChatHistoryOptions) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch session by ID or by span correlation
  // Support search mode
  // Handle pagination for large sessions

  return { session, messages, loading, error, refetch };
}
```

### Phase 4: Trace Enhancement (~150 LOC)

**Modify existing components:**

1. **TraceWaterfall.tsx** - Add "Chat" button on spans with session.id
```tsx
{span.attributes?.['session.id'] && (
  <Button size="xs" onClick={() => openChatPanel(span)}>
    💬 Chat
  </Button>
)}
```

2. **DetailPanel.tsx** - Add ChatContextPanel when span selected
```tsx
{selectedSpan?.attributes?.['session.id'] && (
  <ChatContextPanel spanId={selectedSpan.id} />
)}
```

3. **ChatContextPanel.tsx** (~100 LOC) - Sidebar for span context
   - Shows messages within span's time window
   - Auto-scrolls to relevant message
   - Compact view (fewer details than full ChatHistory)

### Phase 5: Search Integration (~150 LOC)

**Backend (`search.go`):**
```go
type SearchOptions struct {
    Query     string
    Project   string
    Model     string
    StartDate time.Time
    EndDate   time.Time
    Limit     int
}

type SearchResult struct {
    SessionID   string
    MessageUUID string
    Snippet     string
    Score       float64
    Timestamp   time.Time
}

func (r *Reader) Search(opts SearchOptions) ([]SearchResult, error) {
    // 1. List all sessions matching filters
    // 2. SimHash for fast candidate selection
    // 3. Optional: neural search via Ollama if available
    // 4. Return ranked results with context snippets
}
```

**Reuse existing SimHash infrastructure:**
- `_simhash` builtin already exists for document search
- Apply same algorithm to conversation content
- Optional neural embeddings via `~/.ailang/config.yaml` Ollama config

## Files to Create

| File | LOC | Purpose |
|------|-----|---------|
| `internal/claudehistory/reader.go` | ~200 | JSONL parsing, session listing |
| `internal/claudehistory/models.go` | ~100 | Data structures |
| `internal/claudehistory/search.go` | ~150 | SimHash + neural search |
| `internal/server/handlers_claudehistory.go` | ~200 | 6 API endpoints |
| `ui/.../ChatHistory/ChatHistory.tsx` | ~300 | Main conversation view |
| `ui/.../ChatHistory/ChatMessage.tsx` | ~150 | Message renderer |
| `ui/.../ChatHistory/ThinkingBlock.tsx` | ~80 | Thinking display |
| `ui/.../ChatHistory/ToolCallBlock.tsx` | ~100 | Tool visualization |
| `ui/.../ChatHistory/ChatSearch.tsx` | ~150 | Search interface |
| `ui/.../ChatHistory/ChatContextPanel.tsx` | ~100 | Span sidebar |
| `ui/.../hooks/useChatHistory.ts` | ~100 | Data fetching |

**Total new code:** ~1,630 LOC

## Files to Modify

| File | Changes |
|------|---------|
| `internal/server/server.go` | Register `/api/claude-history/*` routes |
| `ui/.../ControlPlane.tsx` | Add "Chat" aggregation option |
| `ui/.../AggregationNav.tsx` | Add Chat nav item |
| `ui/.../TraceWaterfall.tsx` | Add "Chat" button on spans |
| `ui/.../DetailPanel.tsx` | Add ChatContextPanel |

## Technical Considerations

### Performance
- **Lazy loading:** Don't load all sessions at startup
- **Pagination:** Large sessions paginated (50 messages per page)
- **Caching:** Cache parsed sessions for duration of server run
- **Index:** Build search index incrementally as sessions accessed

### Storage
- **Read-only:** Never modify `~/.claude/projects/` (Claude Code's data)
- **No duplication:** Don't copy to observatory.db (direct file access)
- **Future option:** Optional mirror to SQLite for faster queries

### Privacy
- All data stays local (no cloud sync)
- Thinking content visible (may contain sensitive reasoning)
- Consider: Option to hide thinking blocks by default in settings

### Correlation Accuracy
- `sessionId` is primary link to OTEL spans via `session.id` attribute
- `timestamp` for fine-grained message-to-span matching
- `requestId` could link to Anthropic API if needed

### Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A2: Replayability | +1 | Makes conversation history inspectable and auditable |
| A7: Machines First | +1 | Structured JSON format, machine-readable |
| A9: Cost Visibility | +1 | Shows token usage per message |
| A12: System Boundary | 0 | Reads external data, clearly marked |
| **Net Score** | **+3** | Strengthens observability |

## Implementation Milestones

### M1: Backend Reader (Day 1 morning)
- [ ] Create `internal/claudehistory/` package
- [ ] Implement JSONL parsing
- [ ] Add session listing and retrieval
- [ ] Write unit tests

### M2: API Endpoints (Day 1 afternoon)
- [ ] Add 6 HTTP endpoints
- [ ] Implement span correlation logic
- [ ] Add to server router
- [ ] Test with curl

### M3: Chat View (Day 2 morning)
- [ ] Create ChatHistory component
- [ ] Create ChatMessage, ThinkingBlock, ToolCallBlock
- [ ] Add useChatHistory hook
- [ ] Wire up to ControlPlane

### M4: Trace Integration (Day 2 afternoon)
- [ ] Add "Chat" button to TraceWaterfall
- [ ] Create ChatContextPanel
- [ ] Update DetailPanel
- [ ] Test full workflow

### M5: Search (Day 3)
- [ ] Implement SimHash search
- [ ] Add search UI
- [ ] Optional: Add neural search
- [ ] Final testing and polish

## Success Criteria

- [ ] Can view full conversation for any session
- [ ] Thinking blocks display and collapse correctly
- [ ] Tool calls show input/output
- [ ] "Chat" button appears on spans with session.id
- [ ] Search finds conversations by content
- [ ] Token usage displayed accurately
- [ ] All existing tests pass
- [ ] New tests for claudehistory package

## Related Documents

- [M-TASK-GRAPH-SPANS-UNIFICATION](design_docs/planned/v0_7_1/m-task-graph-spans-unification.md) - Similar dashboard integration pattern
- [M-TRACE-INSTRUMENTATION](design_docs/planned/v0_7_1/m-trace-instrumentation.md) - OTEL spans with session.id

## Notes

This feature makes previously invisible data visible. The ~998MB of conversation history on the user's machine becomes a valuable debugging and learning resource.

**Key insight:** The `sessionId` in Claude Code's JSONL files is the same UUID stored in OTEL spans as `session.id`. This enables seamless correlation between traces and conversations.

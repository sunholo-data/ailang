# M-COLLAB-PROVIDER-STATS: Provider & Workspace Tags and Statistics for Collaboration Hub

**Status**: Planned
**Target**: v0.6.2
**Priority**: P1 - Medium
**Estimated**: 1.5 days
**Dependencies**: M-COORD-FEEDBACK (v0.6.1 - completed)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on execution determinism |
| A2: Replayability | +1 | Provider + workspace tracking improves audit trails |
| A3: Effect Legibility | +1 | Makes executor choice and location visible in UI |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification impact |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured metadata for filtering |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-provider and per-folder cost breakdown |
| A10: Composability | 0 | No composability impact |
| A11: Structured Failure | 0 | No failure handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** ✅ Proceed to implementation

## Problem Statement

**Current State (v0.6.1):**
- Provider is saved to the database in `tasks` table
- Provider appears in message text content (e.g., "Completed by claude-code")
- No way to filter messages by provider in the UI
- No per-provider statistics breakout
- Statistics show aggregate totals across all providers
- **Workspace**: UI allows setting workspace per thread, but:
  - Coordinator creates its own worktree paths (ignores thread workspace)
  - Workspace not displayed prominently in message list
  - No filtering by workspace/folder
  - No per-folder statistics

**Pain Points:**
1. Users cannot quickly see which provider executed which tasks
2. Cannot filter to "show only gemini-cli tasks" or "only claude-code tasks"
3. Cost analysis doesn't show which provider is most cost-effective
4. No visibility into provider usage patterns at global or per-agent level
5. Provider info buried in message text, not surfaced as structured metadata
6. **Workspace**: Cannot see where tasks actually ran
7. **Workspace**: Cannot filter by working directory to see all tasks for a project
8. **Workspace**: No per-folder cost/token aggregation

**User Requests:**
> "Can we add the provider as a tag in the thread message list? Instead of in the message text - let us be able to filter by the provider, and add a way to breakout the totals per provider in the global and agent level aggregated statistics"

> "It should specify what working directory it is running in and we should be able to filter by that as well so we can get per folder aggregated and per agent stats"

## Goals

**Primary Goal:** Surface provider AND workspace as structured metadata with filtering and statistics.

**Success Metrics:**
- Provider displayed as a tag/badge on each message in thread list
- Workspace displayed as a tag/badge on each message
- Filter by provider (claude-code, gemini-cli, gemini-api)
- Filter by workspace/folder
- Global statistics dashboard shows per-provider breakdown
- Global statistics dashboard shows per-workspace breakdown
- Per-agent statistics show provider and workspace usage patterns
- Cost, tokens, and task counts broken down by provider AND workspace

## Solution Design

### Overview

Six interconnected features:

1. **Provider Tag Display** - Show provider as a colored badge in message list
2. **Workspace Tag Display** - Show workspace/folder as a badge in message list
3. **Provider Filtering** - Filter messages and tasks by provider
4. **Workspace Filtering** - Filter messages and tasks by working directory
5. **Provider Statistics** - Aggregate stats broken down by provider
6. **Workspace Statistics** - Aggregate stats broken down by working directory

### Current Workspace Architecture

**Investigation findings:**

1. **Thread-level workspace**: UI stores `workspace` on Thread object (`thread.workspace`)
   - Users can set via folder picker in ConversationView
   - Persisted via `PUT /api/threads/{id}` with `{ workspace: "/path" }`
   - Passed in message metadata when sending directives

2. **Coordinator worktree**: Daemon creates isolated git worktrees
   - `d.worktreeMgr.CreateWorktree(task.ID)` creates `~/.ailang/state/worktrees/coordinator/<task-id>/`
   - This is where code actually runs (isolated from main repo)
   - **Currently ignores thread.workspace setting**

3. **Message metadata**: Result messages include `execution_stats.workspace`
   - Already stored in message metadata JSON
   - Shows in UI when files are listed (small badge)

**Key insight**: There are TWO workspaces:
- **Source workspace**: Where the user intended the task to run (thread.workspace)
- **Execution workspace**: Where it actually ran (worktree path)

For statistics, we should track the **source workspace** (the project folder) rather than the worktree path.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Collaboration Hub Dashboard                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    Message List (Thread View)                          │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │ │
│  │  │ [claude-code] [/projects/ailang] [completed] Fix parser   12:30 │  │ │
│  │  │ [gemini-cli]  [/projects/stapled] [completed] Add tests   12:45 │  │ │
│  │  │ [claude-code] [/projects/ailang] [running]   Refactor    Now    │  │ │
│  │  └─────────────────────────────────────────────────────────────────┘  │ │
│  │                                                                        │ │
│  │  Filters:                                                              │ │
│  │  Provider:  [All] [claude-code ✓] [gemini-cli] [gemini-api]           │ │
│  │  Workspace: [All] [/projects/ailang ✓] [/projects/stapled]            │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    Statistics Panel                                    │ │
│  │                                                                        │ │
│  │  By Provider:                                                          │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                 │ │
│  │  │ claude-code  │  │ gemini-cli   │  │ gemini-api   │                 │ │
│  │  │ Tasks: 45    │  │ Tasks: 12    │  │ Tasks: 3     │                 │ │
│  │  │ Cost: $2.34  │  │ Cost: $0.89  │  │ Cost: $0.12  │                 │ │
│  │  │ Tokens: 125k │  │ Tokens: 45k  │  │ Tokens: 8k   │                 │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                 │ │
│  │                                                                        │ │
│  │  By Workspace:                                                         │ │
│  │  ┌───────────────────┐  ┌───────────────────┐                         │ │
│  │  │ /projects/ailang  │  │ /projects/stapled │                         │ │
│  │  │ Tasks: 42         │  │ Tasks: 18         │                         │ │
│  │  │ Cost: $2.15       │  │ Cost: $1.20       │                         │ │
│  │  │ Tokens: 130k      │  │ Tokens: 48k       │                         │ │
│  │  └───────────────────┘  └───────────────────┘                         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Details

#### 1. Provider Tag Display

**UI Changes (`ui/src/components/`):**

```typescript
// New component: ProviderBadge.tsx
interface ProviderBadgeProps {
  provider: 'claude-code' | 'gemini-cli' | 'gemini-api' | string;
}

const providerColors: Record<string, string> = {
  'claude-code': 'bg-orange-500',   // Anthropic orange
  'gemini-cli': 'bg-blue-500',      // Google blue
  'gemini-api': 'bg-green-500',     // API green
};
```

**Message List Integration:**
- Add provider badge next to status badge
- Read provider from message metadata (already stored)
- Color-code by provider for visual distinction

#### 2. Workspace Tag Display

**UI Changes (`ui/src/components/`):**

```typescript
// New component: WorkspaceBadge.tsx
interface WorkspaceBadgeProps {
  workspace: string;
  showFullPath?: boolean;  // Show full path or just folder name
}

// Display: Show last folder name with tooltip for full path
// e.g., "/Users/mark/dev/sunholo/ailang" displays as "ailang" with full path on hover
```

**Data Flow:**
- Thread has `workspace` field (already persisted)
- Coordinator should add `source_workspace` to task metadata (the thread.workspace)
- Result messages include workspace in metadata

#### 3. Provider Filtering

**UI State Management:**

```typescript
// Filter state in ThreadList component
interface ThreadListFilters {
  status?: 'all' | 'completed' | 'running' | 'failed';
  provider?: 'all' | 'claude-code' | 'gemini-cli' | 'gemini-api';
  workspace?: 'all' | string;  // Full path or 'all'
  dateRange?: { start: Date; end: Date };
}
```

**Filter UI:**
- Add provider filter dropdown/chips to thread list header
- Support multi-select (show claude-code AND gemini-cli)
- Persist filter preferences in localStorage
- URL query params for shareable filtered views

**API Extension:**
```go
// internal/server/handlers_messages.go
// GET /api/messages?thread_id=X&provider=claude-code

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
    provider := r.URL.Query().Get("provider")
    // Add provider filter to SQL query
}
```

#### 4. Workspace Filtering

**Filter UI:**
- Dropdown showing all unique workspaces used
- Auto-populated from task history
- Shows folder name with full path in tooltip

**API Extension:**
```go
// GET /api/messages?thread_id=X&workspace=/path/to/project
// GET /api/workspaces - List all unique workspaces

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
    // SELECT DISTINCT workspace FROM threads WHERE workspace IS NOT NULL
}
```

#### 5. Provider Statistics

**Database Query:**

```sql
-- Per-provider aggregation
SELECT
    provider,
    COUNT(*) as task_count,
    SUM(cost) as total_cost,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    AVG(duration_ns) as avg_duration_ns
FROM tasks
WHERE status = 'completed'
GROUP BY provider;
```

**Statistics API:**

```go
// GET /api/statistics
// GET /api/statistics/by-provider
// GET /api/statistics/by-workspace

type ProviderStats struct {
    Provider     string  `json:"provider"`
    TaskCount    int     `json:"task_count"`
    TotalCost    float64 `json:"total_cost"`
    InputTokens  int64   `json:"input_tokens"`
    OutputTokens int64   `json:"output_tokens"`
    AvgDuration  int64   `json:"avg_duration_ns"`
}

type WorkspaceStats struct {
    Workspace    string  `json:"workspace"`
    FolderName   string  `json:"folder_name"`  // Last path component
    TaskCount    int     `json:"task_count"`
    TotalCost    float64 `json:"total_cost"`
    InputTokens  int64   `json:"input_tokens"`
    OutputTokens int64   `json:"output_tokens"`
    AvgDuration  int64   `json:"avg_duration_ns"`
}

type StatisticsResponse struct {
    Global      GlobalStats      `json:"global"`
    ByProvider  []ProviderStats  `json:"by_provider"`
    ByWorkspace []WorkspaceStats `json:"by_workspace"`
}
```

#### 6. Workspace Statistics

**Database Schema Change:**

```sql
-- Add workspace column to tasks table
ALTER TABLE tasks ADD COLUMN workspace TEXT;
CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace);
```

**Coordinator Change:**
```go
// internal/coordinator/daemon.go
// When creating task, capture thread.workspace as source_workspace

func (d *Daemon) executeTask(task *TaskRecord) error {
    // Get workspace from thread
    thread, _ := d.msgStore.GetThread(task.ThreadID)
    sourceWorkspace := ""
    if thread != nil {
        sourceWorkspace = thread.Workspace
    }

    // Store in task record
    task.Workspace = sourceWorkspace

    // Also add to result metadata
    metadata["source_workspace"] = sourceWorkspace
}
```

**Statistics UI Panel:**
- Cards showing per-provider breakdown
- Cards showing per-workspace breakdown
- Bar chart comparing providers (optional)
- Trend over time (optional, v0.6.3+)

### Implementation Plan

**Phase 1: Database Schema** (~1 hour)
- [ ] Add `workspace` column to `tasks` table
- [ ] Add migration for existing databases
- [ ] Create index for workspace filtering

**Phase 2: Coordinator Workspace Tracking** (~2 hours)
- [ ] Capture thread.workspace when creating task
- [ ] Store in task record
- [ ] Include in result message metadata
- [ ] Add to execution_stats for UI

**Phase 3: Provider Tag Display** (~1.5 hours)
- [ ] Create `ProviderBadge.tsx` component
- [ ] Add provider badge to message list items
- [ ] Color scheme for each provider
- [ ] Handle unknown/missing provider gracefully

**Phase 4: Workspace Tag Display** (~1.5 hours)
- [ ] Create `WorkspaceBadge.tsx` component
- [ ] Show folder name with full path tooltip
- [ ] Add workspace badge to message list items
- [ ] Handle missing workspace gracefully

**Phase 5: Filtering UI** (~2 hours)
- [ ] Create unified FilterBar component
- [ ] Provider filter chips
- [ ] Workspace filter dropdown
- [ ] Add filter state to ThreadList
- [ ] Persist filter preferences in localStorage

**Phase 6: Filter API** (~1.5 hours)
- [ ] Add `provider` and `workspace` query params to message list
- [ ] Add `GET /api/workspaces` endpoint
- [ ] Add SQL WHERE clauses

**Phase 7: Statistics** (~2.5 hours)
- [ ] Create SQL aggregation queries for provider and workspace
- [ ] Add `/api/statistics` endpoint with full breakdown
- [ ] Create ProviderStatsPanel component
- [ ] Create WorkspaceStatsPanel component
- [ ] Add to statistics dashboard
- [ ] Test with real data

### Files to Create/Modify

**New Files:**

```
ui/src/components/ProviderBadge.tsx        (~50 LOC)
ui/src/components/WorkspaceBadge.tsx       (~60 LOC)
ui/src/components/FilterBar.tsx            (~120 LOC)
ui/src/components/ProviderStatsPanel.tsx   (~100 LOC)
ui/src/components/WorkspaceStatsPanel.tsx  (~100 LOC)
internal/server/handlers_statistics.go      (~200 LOC)
```

**Modified Files:**

```
ui/src/components/MessageList.tsx           (+40 LOC)
ui/src/components/ThreadList.tsx            (+50 LOC)
ui/src/types/index.ts                       (+25 LOC)
internal/server/handlers_messages.go        (+40 LOC)
internal/coordinator/store_sqlite.go        (+60 LOC)
internal/coordinator/daemon.go              (+30 LOC)
```

**Total: ~875 LOC**

## Success Criteria

**Provider:**
- [ ] Provider badge visible on each message in thread list
- [ ] Provider badge color-coded (orange=Claude, blue=Gemini CLI, green=Gemini API)
- [ ] Filter by provider works in message list
- [ ] `/api/statistics` includes per-provider breakdown
- [ ] Statistics panel shows per-provider cards

**Workspace:**
- [ ] Workspace badge visible on messages (shows folder name)
- [ ] Full path shown in tooltip
- [ ] Filter by workspace works
- [ ] `/api/statistics` includes per-workspace breakdown
- [ ] Statistics panel shows per-workspace cards
- [ ] Coordinator captures source workspace from thread

**General:**
- [ ] Filters persist on page refresh
- [ ] Works with existing data (handles null/empty values)
- [ ] Documentation updated

## Non-Goals

**Not in this feature:**
- Historical trends over time (defer to v0.6.3+)
- Provider/workspace cost comparison charts
- Performance recommendations
- Provider/workspace preference settings per user
- Using thread.workspace as actual execution directory (worktrees remain)

## Open Questions

1. **Worktree vs Source Workspace**: Should we display both? Currently the worktree path is ephemeral and not meaningful for filtering. Recommend: only show source workspace (thread.workspace).

2. **Workspace Grouping**: Should we group by project (last 2 path components) or exact path? Recommend: exact path with folder name display.

3. **Pending stats display**: When a task is pending/running, should we show "pending" instead of 0s for tokens/cost? Yes - confusing to show 0s.

## Appendix: Message System Architecture

### Current Message Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MESSAGE SOURCES                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. CLI: ailang messages send coordinator "message"                          │
│     └─> inbox_messages table ─> Coordinator polls ─> Execute                │
│                                                                             │
│  2. Dashboard UI: POST /api/messages                                         │
│     └─> threads/messages tables ─> WebSocket broadcast                       │
│     └─> If thread.target_agent="coordinator" ─> Coordinator picks up        │
│                                                                             │
│  3. Agent: messaging.Client.PublishMessage()                                 │
│     └─> messages table ─> WebSocket broadcast                               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Coordinator Inbox Adapter (internal/coordinator/message_adapter.go):
- Watches "coordinator" inbox in inbox_messages table
- Converts InboxMessage → coordinator.Message
- Marks messages as read after processing

Thread Target Agent (internal/messaging/threads.go):
- Thread has target_agent in context_json
- Created by coordinator: target_agent="coordinator"
- User threads: target_agent could be any agent or empty
```

### Message Kinds

| Kind | Description | Execution Mode |
|------|-------------|----------------|
| `directive` | Task/instruction to execute | Full: all tools allowed |
| `question` | Query for information | Read-only: Read, Grep, Glob, WebFetch, WebSearch |

The kind is set by the UI dropdown (default: "directive") and used by `provider_claude.go:80` to restrict tools for questions.

### Two Database Tables for Messages

1. **inbox_messages**: CLI message inbox (ailang messages send/list)
   - Used by: CLI, agents polling their inbox
   - Schema: id, inbox, from_agent, title, payload, category, message_type, status

2. **messages**: Thread-based messages for dashboard
   - Used by: Dashboard WebSocket, collaboration UI
   - Schema: id, thread_id, from_type, from_id, to_type, to_id, kind, content

The coordinator uses `InboxMessageAdapter` to read from inbox_messages but writes results to both tables for visibility.

## Related Documents

- [M-COORD-FEEDBACK](../../implemented/v0_6_1/m-coord-feedback-sprint-plan.md) - Real-time streaming
- [Global Collaboration Hub](global-collaboration-hub.md) - Future cloud architecture

---

**Document created**: 2024-12-30
**Last updated**: 2024-12-30

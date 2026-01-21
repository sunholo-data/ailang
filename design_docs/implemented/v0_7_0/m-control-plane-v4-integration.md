# M-CONTROL-PLANE-V4-INTEGRATION: Unified Telemetry for Agent Monitoring

**Status**: Planned
**Target**: v0.6.4
**Priority**: P0 (Critical)
**Estimated**: 3-4 days
**Dependencies**: M-OTEL-DASHBOARD (completed mockup), Observatory API (M1-M4 complete)
**Bug Report**: N/A (architectural unification)

## Related Documents

- [M-OTEL-DASHBOARD Sprint Plan](m-otel-dashboard-sprint-plan.md) - Original mockup design
- [M-TASK-HIERARCHY Sprint Plan](../v0_6_5/m-task-hierarchy-sprint-plan.md) - Task hierarchy API
- [Observatory Implementation](../../../internal/observatory/) - OTEL storage backend

---

## Executive Summary

**This design document addresses a critical data architecture problem**: the Control Plane and Observatory show different metrics for the same system because they query different data sources. This must be resolved before the dashboard can be used for production monitoring, eval tracking, or autonomous agent orchestration.

**Discovered Discrepancy:**
| Metric | Control Plane (`/api/statistics`) | Observatory (`/api/observatory/metrics`) |
|--------|-----------------------------------|------------------------------------------|
| Tasks | 29 completed | 24 tasks |
| Cost | $4.20 | $446.20 |
| Tokens | 107.6K | 11.9M (in) + tokens out |

These numbers differ by **100x** in cost because they track fundamentally different things.

---

## Problem Statement

### Two Separate Data Systems

AILANG has evolved two parallel observability systems that were designed for different purposes:

#### 1. Observatory (`~/.ailang/state/observatory.db`)

**Purpose:** Store OTEL spans from all AILANG operations

**Data Source:** OTLP Receiver (`internal/observatory/otlp_receiver.go`)
- Receives OpenTelemetry spans from instrumented code
- Stores spans in SQLite with normalized attributes
- Forwards to GCP Cloud Trace (when configured)

**What it tracks:**
- Every instrumented operation (`compile.*`, `eval.*`, `messages.*`, `anthropic.generate`, etc.)
- Token usage per span (from OTEL attributes)
- Cost per span (calculated from tokens + model pricing)
- Task hierarchies linking spans to logical tasks

**Key Tables:**
```sql
spans (id, trace_id, task_id, tokens_in, tokens_out, cost_usd, ...)
tasks (id, title, total_tokens_in, total_tokens_out, total_cost_usd, ...)
agent_assignments (id, task_id, agent_id, tokens_in, tokens_out, cost_usd, ...)
```

**API:** `/api/observatory/*`

#### 2. Coordinator (`~/.ailang/state/coordinator.db`)

**Purpose:** Track delegated tasks managed by the coordinator daemon

**Data Source:** Coordinator Daemon (`internal/coordinator/daemon.go`)
- Creates task records when messages are processed
- Updates tasks with execution results from Claude Code / Gemini CLI
- Tracks approvals, worktrees, and agent assignments

**What it tracks:**
- Tasks delegated to AI agents (design-doc-creator, sprint-planner, etc.)
- Execution costs reported by CLI tools (often incomplete)
- Task lifecycle (pending → running → pending_approval → completed)

**Key Tables:**
```sql
tasks (id, title, cost, tokens_used, status, provider, agent_id, ...)
task_events (id, task_id, tokens_in, tokens_out, cost, ...)
approval_requests (id, task_id, status, ...)
```

**API:** `/api/statistics`, `/api/coordinator/*`

### Why the Numbers Don't Match

| Aspect | Observatory | Coordinator |
|--------|-------------|-------------|
| **Scope** | ALL OTEL spans (evals, CLI ops, API calls) | Only coordinator-delegated tasks |
| **Token Source** | OTEL span attributes | CLI execution results |
| **Cost Calculation** | `tokens × model_pricing` per span | `result.Cost` from executor |
| **Task Definition** | Any operation with `task_id` attribute | Explicit task record in DB |
| **Data Flow** | Real-time via OTLP receiver | Batch on task completion |

**Example:** Running `ailang eval-suite` generates:
- **Observatory:** 1000+ spans with token/cost data for each model call
- **Coordinator:** 0 tasks (not a delegated task)

**Example:** Delegating "Fix parser bug" to design-doc-creator:
- **Observatory:** Spans from Claude Code execution (if instrumented)
- **Coordinator:** 1 task with aggregated cost/tokens from CLI output

### The Real Problem

**The Control Plane currently shows Coordinator stats**, which only reflects delegated tasks. But the **Observatory contains the canonical OTEL telemetry** - the raw, accurate data about actual token usage and costs.

For the dashboard to be useful for:
- Production monitoring
- Eval tracking
- Cost analysis
- Autonomous agent orchestration

**Observatory must be the source of truth.**

---

## Goals

**Primary Goal:** Make Observatory the canonical data source for Control Plane, with Coordinator data as a supplementary view.

**Success Metrics:**
1. Header stats show Observatory metrics (total tokens, total cost, total spans)
2. Coordinator stats shown separately in "Agent Tasks" context
3. Numbers match between all views (no unexplained discrepancies)
4. Traces/Tasks tabs mirror Observatory functionality
5. Architecture supports future autonomous agent monitoring

---

## Solution Architecture

### Unified Data Model

The solution treats **Observatory as the canonical source** and **Coordinator as a view**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Control Plane Dashboard                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐ │
│  │  Global Stats    │  │  Agent Tasks     │  │  Traces Tab   │ │
│  │  (Observatory)   │  │  (Coordinator)   │  │  (Observatory)│ │
│  │                  │  │                  │  │               │ │
│  │  Total Cost      │  │  Pending Tasks   │  │  Span Tree    │ │
│  │  Total Tokens    │  │  Running Tasks   │  │  Waterfall    │ │
│  │  Total Spans     │  │  Pending Approvals│ │               │ │
│  │  Success Rate    │  │  Agent Stats     │  │               │ │
│  └──────────────────┘  └──────────────────┘  └───────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ Observatory   │     │ Coordinator   │     │ GCP Cloud     │
│ (SQLite)      │────▶│ (SQLite)      │     │ Trace         │
│               │     │               │     │ (optional)    │
│ observatory.db│     │ coordinator.db│     │               │
└───────────────┘     └───────────────┘     └───────────────┘
       ▲                     ▲
       │                     │
       │   OTLP Receiver     │   Task Events
       │                     │
┌──────┴──────────────────────┴──────────────────────────────────┐
│                    Instrumented Code                            │
│                                                                 │
│  ailang eval-suite    |    Claude Code CLI    |    Gemini CLI  │
│  ailang messages      |    coordinator daemon |    direct API  │
└─────────────────────────────────────────────────────────────────┘
```

### Universal Task Hierarchy

**Key Architectural Principle:** Observatory's workspace/task hierarchy is the **universal organizing structure** for ALL telemetry sources.

```
Observatory (canonical source)
├── Workspace: "ailang"
│   │
│   ├── Task: "v0.6.3 Eval Baseline" (source: eval)
│   │   ├── trace: eval.suite
│   │   │   └── spans: api_request ×7,675 → $442.28
│   │   └── metrics: 6 models, 46 benchmarks
│   │
│   ├── Task: "Fix parser bug #123" (source: coordinator)
│   │   ├── agent: design-doc-creator
│   │   ├── trace: claude.execute
│   │   │   └── spans: anthropic.generate ×50 → $0.15
│   │   └── approval: pending
│   │
│   ├── Task: "Research embeddings" (source: local/direct)
│   │   └── trace: user-session-abc
│   │       └── spans: gemini.generate ×3 → $0.02
│   │
│   └── Task: "CI Pipeline Run" (source: future/github-actions)
│       └── spans: ...
```

**All sources are SUBSETS of Observatory:**

| Source | Task Creation | Span Linking | Current Status |
|--------|---------------|--------------|----------------|
| **Eval** | `ailang eval-suite` creates task | `eval.*` spans link via trace_id | ⚠️ Needs task_id linking |
| **Coordinator** | Daemon creates task on message pickup | `claude.execute` links via session.id | ✅ Implemented |
| **Local/Direct** | CLI creates task on `ailang run` | Spans link via trace_id | ⚠️ Needs task creation |
| **Future Sources** | TBD (GitHub Actions, webhooks, etc.) | TBD | 🔜 Extensible design |

### Data Reconciliation Strategy

To ensure ALL sources appear in Observatory task hierarchy:

1. **Coordinator → Observatory Link** ✅ DONE
   - When coordinator starts a task, create Observatory task record
   - When Claude Code/Gemini CLI send OTEL spans, link to Observatory task via `session.id`
   - Already implemented: `LookupTaskBySessionID()` in `observatory/store.go`

2. **Eval → Observatory Link** ⚠️ TODO
   - `ailang eval-suite` should create an Observatory task at start
   - All `eval.*` and `api_request` spans should carry that task_id
   - Enables: "Show me all evals for this week" in Control Plane

3. **Local Usage → Observatory Link** ⚠️ TODO
   - `ailang run`, REPL sessions should create lightweight tasks
   - Enables: "How much did local experimentation cost this month?"

5. **Metrics Aggregation**
   - Observatory `GetMetricsSummary()` aggregates from `spans` table
   - This includes ALL sources, organized by workspace/task
   - Coordinator stats remain useful for agent-specific views

### UI Data Sources

   | Component | Data Source | Rationale |
   |-----------|-------------|-----------|
   | Header Stats (Cost/Tokens/Spans) | Observatory `/api/observatory/metrics/summary` | Canonical OTEL data |
   | Header Stats (Active Agents/Pending) | Coordinator `/api/statistics` | Runtime state |
   | Activity Heatmap | Observatory spans grouped by date | Complete activity |
   | Agent Topology | Config + Coordinator status | Agent relationships |
   | Trace Waterfall | Observatory `/api/observatory/traces` | OTEL spans |
   | Message Queue | WebSocket (coordinator events) | Real-time |
   | Tasks Tab | Observatory `/api/observatory/tasks` | Task hierarchy |

---

## Future Vision: Autonomous Agent Orchestration

This integration is foundational for autonomous AI agents that self-organize in the cloud.

### Phase 1: Unified Monitoring (v0.6.4) ← **This Design**
- Single source of truth for costs/tokens/performance
- Real-time visibility into agent execution
- Historical data for trend analysis

### Phase 2: Agent Performance Analytics (v0.7.0)
- Per-agent cost breakdown from OTEL spans
- Success rate by agent/provider/model
- Bottleneck identification (which agents are slow?)
- Alert thresholds for cost overruns

### Phase 3: Self-Organizing Agents (v0.8.0+)
- Agents can query Observatory to understand system state
- Coordinator routes tasks based on agent performance history
- Automatic scaling: spawn more agents when queue grows
- Trust adjustment based on success rates

### Phase 4: Cloud Deployment (v1.0.0+)
- Observatory data replicated to cloud storage
- Multiple coordinator instances coordinating via shared state
- Agents run in ephemeral containers
- Human-in-the-loop for high-impact decisions only

**Key Insight:** All of this depends on **accurate, unified telemetry**. Getting the data model right now enables autonomous orchestration later.

---

## Data Source Mapping

### 1. Activity Heatmap

**UI Component:** GitHub-style calendar showing task activity over 90 days

**Mock Data Structure:**
```typescript
interface HeatmapCell {
  date: string;      // YYYY-MM-DD
  taskCount: number;
  cost: number;
  successRate: number;
}
```

**Data Source:** NEW API endpoint needed

**Gap:** No existing API provides daily aggregations. Need to aggregate from:
- `coordinator.db` → `tasks` table (has `created_at`, `cost`, `status`)
- Group by date, count tasks, sum costs, calculate success rate

**New API:** `GET /api/controlplane/heatmap?days=90`
```json
{
  "cells": [
    {"date": "2024-12-01", "taskCount": 15, "cost": 2.34, "successRate": 0.87},
    ...
  ],
  "totals": {"tasks": 1234, "cost": 456.78}
}
```

**Implementation:** ~50 LOC in `internal/server/handlers_controlplane.go`

---

### 2. Agent Topology

**UI Component:** Force-directed graph showing agent relationships

**Mock Data Structure:**
```typescript
interface Agent {
  id: string;
  label: string;
  status: 'idle' | 'busy' | 'blocked' | 'error';
  trustScore: number;
  taskCount: number;
  cost: number;
}
interface TopologyEdge {
  source: string;
  target: string;
  messageCount: number;
  lastActivity: string;
}
```

**Data Sources:**

1. **Agent List:** `~/.ailang/config.yaml` → `coordinator.agents[]`
   - Already parsed by `internal/coordinator/agent_registry.go`
   - Contains: id, label, inbox, workspace, trigger_on_complete

2. **Agent Status:** `GET /api/coordinator/running`
   - Returns currently executing tasks with agent assignments
   - Can derive: idle (no task), busy (has task), blocked (waiting approval)

3. **Agent Stats:** `GET /api/observatory/agents/{id}/stats`
   - Returns: task_count, total_cost, success_rate
   - BUT: Currently requires Observatory agent assignment, not coordinator agent

4. **Edge Data (Agent Relationships):**
   - `trigger_on_complete` in config defines directed edges
   - Message counts from `GET /api/observatory/messages?from={agent}`

**Gap:** Need unified API that combines config + runtime status + stats

**New API:** `GET /api/controlplane/topology`
```json
{
  "agents": [
    {
      "id": "design-doc-creator",
      "label": "Design Doc Creator",
      "status": "idle",
      "trustScore": 75,
      "taskCount": 29,
      "cost": 4.20
    }
  ],
  "edges": [
    {"source": "github", "target": "design-doc-creator", "messageCount": 12},
    {"source": "design-doc-creator", "target": "sprint-planner", "messageCount": 8}
  ],
  "sinks": [
    {"id": "approval", "pendingCount": 12},
    {"id": "main", "label": "main branch"}
  ]
}
```

**Implementation:** ~100 LOC aggregating config + coordinator + observatory

---

### 3. Trace Waterfall

**UI Component:** Horizontal timing chart showing nested spans

**Mock Data Structure:**
```typescript
interface Span {
  id: string;
  name: string;
  service: string;
  startTime: number;
  duration: number;
  parentId?: string;
  status: 'ok' | 'error';
  attributes?: Record<string, string>;
}
```

**Data Source:** Observatory API (COMPLETE)

- `GET /api/observatory/traces` - List traces
- `GET /api/observatory/traces/{id}` - Get trace with spans
- `GET /api/observatory/spans?trace_id={id}` - List spans for trace

**Gap:** None! Observatory already provides this data.

**Implementation:**
- Replace mock `mockSpans` with fetch from `/api/observatory/spans?limit=50`
- Add trace selector dropdown to pick which trace to display
- Transform Observatory Span model to UI Span format (~30 LOC)

---

### 4. Message Queue (Event Queue)

**UI Component:** Real-time event feed showing agent activity

**Mock Data Structure:**
```typescript
interface QueueEvent {
  id: string;
  type: 'handoff' | 'approval' | 'complete' | 'error' | 'status';
  source: string;
  target: string;
  message: string;
  timestamp: number;
  priority: 'high' | 'normal' | 'low';
}
```

**Data Source:** Coordinator WebSocket (COMPLETE)

- `POST /api/coordinator/events` - Coordinator broadcasts events
- Server broadcasts to WebSocket clients
- Events include: task_start, task_complete, task_error, approval_request

**Gap:** Need to listen to existing WebSocket and transform events

**Implementation:**
- Connect to existing `ws://host/ws` WebSocket
- Filter for coordinator event types
- Transform to QueueEvent format (~40 LOC)

**Alternative:** Poll `GET /api/coordinator/tasks/{id}/events` for specific task history

---

### 5. Statistics Cards

**UI Component:** Header stats (Active Agents, Pending Approvals, etc.)

**Mock Data:**
```typescript
const mockStats = {
  activeAgents: 3,
  pendingApprovals: 12,
  taskSuccess: '94.2%',
  totalCost: '$127.45'
};
```

**Data Source:** Statistics API (PARTIAL)

- `GET /api/statistics` returns:
  ```json
  {
    "threads": {...},
    "coordinator": {
      "total_tasks": 41,
      "pending_tasks": 0,
      "running_tasks": 0,
      "completed_tasks": 29,
      "failed_tasks": 11,
      "total_cost": 4.2024,
      "total_tokens": 107615
    }
  }
  ```

**Gap:** Missing:
- Active agent count (agents currently running tasks)
- Pending approval count (need to query approval_requests)
- Success rate calculation

**Enhancement to existing API:** Add fields to `/api/statistics`
```json
{
  "coordinator": {
    ...existing...,
    "active_agents": 2,
    "pending_approvals": 12,
    "success_rate": 0.942
  }
}
```

**Implementation:** ~30 LOC enhancement to `handlers_statistics.go`

---

### 6. Trust Configuration

**UI Component:** Per-capability trust sliders in agent detail panel

**Mock Data:**
```typescript
interface TrustCapability {
  name: string;
  score: number;
  icon: string;
}
```

**Data Source:** NEW - Trust not currently tracked

**Gap:** Trust scores don't exist in the system yet. Options:
1. **Phase 1 (Mockup):** Keep mock data, add TODO for real implementation
2. **Phase 2 (Config-based):** Add trust_levels to agent config in YAML
3. **Phase 3 (Dynamic):** Track trust based on task success/failure history

**Recommendation:** Phase 1 for v0.6.4, defer dynamic trust to v0.7.0

---

### 7. Aggregation Navigator (Sidebar Filters)

**UI Component:** Collapsible tree for filtering by workspace/agent/provider

**Data Source:** Combination of existing APIs

- Workspaces: `GET /api/observatory/workspaces` or `GET /api/workspaces`
- Agents: Config-based (from topology endpoint)
- Providers: `GET /api/observatory/metrics/providers`
- Models: From span attributes

**Gap:** No unified aggregation endpoint

**New API:** `GET /api/controlplane/aggregations`
```json
{
  "workspaces": ["ailang", "stapledon"],
  "agents": ["design-doc-creator", "sprint-planner", "sprint-executor"],
  "providers": ["claude", "gemini"],
  "models": ["claude-sonnet-4-5", "gemini-2-5-flash"],
  "trustLevels": ["auto", "low-risk", "review", "manual"]
}
```

**Implementation:** ~50 LOC aggregating from multiple sources

---

## Implementation Plan

### Milestone 0: Data Verification (~2h)

**CRITICAL: Verify data sources before building UI integration.**

#### 0.1 Query Both Databases

```bash
# Observatory metrics
sqlite3 ~/.ailang/state/observatory.db "
  SELECT
    COUNT(*) as span_count,
    COALESCE(SUM(tokens_in), 0) as total_tokens_in,
    COALESCE(SUM(tokens_out), 0) as total_tokens_out,
    COALESCE(SUM(cost_usd), 0) as total_cost
  FROM spans;
"

# Coordinator metrics
sqlite3 ~/.ailang/state/coordinator.db "
  SELECT
    COUNT(*) as task_count,
    COALESCE(SUM(tokens_used), 0) as total_tokens,
    COALESCE(SUM(cost), 0) as total_cost
  FROM tasks WHERE status = 'completed';
"
```

#### 0.2 Understand the Difference

**Verified Data (January 2026):**

**Observatory (`observatory.db`):**
```
span_count: 25,615
tokens_in:  9,330,656
tokens_out: 2,579,127
total_cost: $449.58
traces:     18,669
tasks:      23
```

**Coordinator (`coordinator.db`):**
```
task_count: 29 (completed)
tokens:     107,615
total_cost: $4.20
```

**Cost Breakdown by Span Type:**
| Span Name | Count | Cost |
|-----------|-------|------|
| `api_request` | 7,675 | $442.28 |
| `anthropic.generate` | 814 | $5.33 |
| `gemini.generate` | 379 | $1.25 |
| `claude.execute` | 22 | $0.49 |
| `coordinator.task.execute` | 14 | $0.25 |

**Why Numbers Differ:**
- **$442** = Eval benchmark runs (`api_request` spans from `ailang eval-suite`)
- **$5.33** = Direct Anthropic API calls (research, docs, etc.)
- **$1.25** = Direct Gemini API calls
- **$4.20** = Coordinator-delegated tasks (design docs, sprints, etc.)

The Observatory includes **everything** (evals, direct API calls, coordinator tasks). The Coordinator only tracks **delegated agent tasks**.

#### 0.3 Define Canonical Source

For each metric type:
| Metric | Canonical Source | Why |
|--------|------------------|-----|
| Total Cost | Observatory | Includes all operations |
| Total Tokens | Observatory | Accurate per-span data |
| Task Count | Observatory `tasks` | Unified task model |
| Agent Status | Coordinator | Runtime state |
| Pending Approvals | Coordinator | Approval workflow |

### Milestone 1: Backend API Updates (~150 LOC, 4h)

**Update `internal/server/handlers_controlplane.go`:**

1. **Heatmap Endpoint** - Use Observatory spans
   ```go
   // Query observatory.db instead of coordinator.db
   // Group spans by date, aggregate tokens/cost/status
   GET /api/controlplane/heatmap?days=90&source=observatory
   ```

2. **Unified Statistics Endpoint**
   ```go
   // Combine both sources with clear labeling
   GET /api/controlplane/stats
   {
     "observatory": {
       "total_spans": 50000,
       "total_cost_usd": 446.20,
       "total_tokens_in": 11900000,
       "total_tokens_out": 1200000,
       "success_rate": 0.95
     },
     "coordinator": {
       "completed_tasks": 29,
       "pending_tasks": 0,
       "running_tasks": 0,
       "pending_approvals": 12,
       "active_agents": 2
     }
   }
   ```

3. **Topology Endpoint** - Already correct
   - Agent config + runtime status

4. **Aggregations Endpoint** - Already correct
   - Filter options from multiple sources

### Milestone 2: Frontend Data Hooks (~200 LOC, 4h)

Create `ui/src/features/controlplane/hooks/`:

1. **useHeatmapData.ts**
   - Fetch from `/api/controlplane/heatmap`
   - Transform to HeatmapCell[]
   - Handle loading/error states

2. **useTopologyData.ts**
   - Fetch from `/api/controlplane/topology`
   - Transform to React Flow nodes/edges
   - Poll for status updates (5s interval)

3. **useTraceData.ts**
   - Fetch from `/api/observatory/spans`
   - Transform to waterfall format
   - Handle trace selection

4. **useEventQueue.ts**
   - Connect to WebSocket
   - Filter coordinator events
   - Transform to QueueEvent[]

5. **useStats.ts**
   - Fetch from `/api/statistics`
   - Extract control plane stats

### Milestone 3: Component Integration (~150 LOC, 3h)

Update `ui/src/features/controlplane/ControlPlane.tsx`:

1. Replace mock data with hook calls
2. Add loading skeletons
3. Add error boundaries
4. Add refresh mechanisms
5. Wire up filter selections to API queries

### Milestone 4: Testing & Polish (~2h)

1. Test with real coordinator data
2. Verify WebSocket event flow
3. Check edge cases (no data, errors)
4. Performance optimization (debounce, memoization)

---

## API Summary

### New Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/controlplane/heatmap` | GET | Daily task activity for heatmap |
| `/api/controlplane/topology` | GET | Agent graph with status/stats |
| `/api/controlplane/aggregations` | GET | Filter options for navigator |

### Enhanced Endpoints

| Endpoint | Enhancement |
|----------|-------------|
| `/api/statistics` | Add active_agents, pending_approvals, success_rate |

### Existing Endpoints (No Changes)

| Endpoint | Usage |
|----------|-------|
| `/api/observatory/traces` | Trace list for waterfall |
| `/api/observatory/spans` | Span data for waterfall |
| `/api/coordinator/events` | Real-time events |
| `ws://host/ws` | WebSocket for live updates |

---

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | 0 | No change to core semantics |
| A2: Replayability | +1 | Better visibility into execution traces |
| A3: Effect Legibility | 0 | Observability only, no new effects |
| A4: Explicit Authority | 0 | Read-only APIs, no new permissions |
| A5: Bounded Verification | 0 | N/A |
| A6: Safe Concurrency | 0 | N/A |
| A7: Machines First | +1 | Structured JSON APIs for tooling |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +1 | Exposes cost data in UI |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | 0 | N/A |
| A12: System Boundary | 0 | N/A |

**Net Score: +3** (exceeds +2 threshold)

---

## Success Criteria

### Data Accuracy
- [ ] Header stats show Observatory metrics: ~$450, ~12M tokens, ~25K spans
- [ ] Agent Tasks section shows Coordinator metrics: 29 tasks, $4.20, 107K tokens
- [ ] Numbers are clearly labeled with their source (Observatory vs Coordinator)
- [ ] Hover tooltips explain what each metric includes

### UI Integration
- [ ] Activity Heatmap shows real span data from last 90 days
- [ ] Agent Topology reflects config + live status
- [ ] Trace Waterfall renders real Observatory spans
- [ ] Tasks Tab shows Observatory task hierarchy
- [ ] Message Queue streams live WebSocket events

### Verification
- [ ] `sqlite3 ~/.ailang/state/observatory.db` metrics match header stats
- [ ] `ailang coordinator status` output matches Agent Tasks section
- [ ] No hardcoded mock data in production build
- [ ] All new endpoints documented in API reference

### Future-Proofing
- [ ] Architecture documented for autonomous agent orchestration
- [ ] API supports filtering by agent/provider/workspace
- [ ] WebSocket events include all data needed for real-time updates

---

## Open Questions

1. **Trust Scores:** Keep mock for v0.6.4 or implement basic config-based trust?
   - **Recommendation:** Mock for now, design trust system properly in v0.7.0

2. **Polling vs WebSocket:** Should topology/heatmap use WebSocket for live updates?
   - **Recommendation:** Polling (5-30s) is sufficient for these views

3. **Historical Data:** How far back should heatmap go (90 days? 1 year?)
   - **Recommendation:** Default 90 days, configurable via range selector (already in UI)

---

## Timeline

- **M1:** Backend APIs - Day 1 (4h)
- **M2:** Frontend hooks - Day 1-2 (4h)
- **M3:** Component integration - Day 2 (3h)
- **M4:** Testing & polish - Day 2-3 (2h)

**Total:** ~13h across 2-3 days

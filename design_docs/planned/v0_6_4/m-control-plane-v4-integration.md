# M-CONTROL-PLANE-V4-INTEGRATION: Connect Control Plane UI to Real Data

**Status**: Planned
**Target**: v0.6.4
**Priority**: P1
**Estimated**: 2-3 days
**Dependencies**: M-OTEL-DASHBOARD (completed mockup), Observatory API (M1-M4 complete)
**Bug Report**: N/A (feature request)

## Related Documents

- [M-OTEL-DASHBOARD Sprint Plan](m-otel-dashboard-sprint-plan.md) - Original mockup design
- [M-TASK-HIERARCHY Sprint Plan](../v0_6_5/m-task-hierarchy-sprint-plan.md) - Task hierarchy API

---

## Problem Statement

The Control Plane v4 UI mockup is complete with beautiful visualizations (Activity Heatmap, Agent Topology, Trace Waterfall, Message Queue), but all data is currently mocked. We need to connect these components to real data from existing AILANG APIs.

**Current State:**
- UI mockup complete with mock data arrays
- Observatory API (M1-M4) provides tasks, spans, traces, messages
- Coordinator API provides task execution, approvals, events
- Statistics API provides aggregate metrics
- No unified API for Control Plane-specific aggregations

**Pain Points:**
- Dashboard shows fake data, not real agent activity
- No visibility into actual coordinator task costs/performance
- Activity heatmap uses random mock data
- Agent topology doesn't reflect real agent relationships

---

## Goals

**Primary Goal:** Connect Control Plane v4 to real AILANG data sources for production monitoring.

**Success Metrics:**
1. Activity Heatmap shows real task counts, costs, success rates by date
2. Agent Topology displays actual agents from config with live status
3. Trace Waterfall renders real OTEL spans from Observatory
4. Message Queue shows live coordinator events via WebSocket
5. Statistics cards reflect actual costs, tokens, task counts

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

### Milestone 1: Backend API Additions (~150 LOC, 4h)

Create `internal/server/handlers_controlplane.go`:

1. **Heatmap Endpoint**
   - Query coordinator tasks grouped by date
   - Calculate daily stats (count, cost, success rate)
   - Return 90-day array

2. **Topology Endpoint**
   - Load agent config from registry
   - Query coordinator for running tasks
   - Query observatory for agent stats
   - Build edges from trigger_on_complete + message counts

3. **Aggregations Endpoint**
   - Collect unique values from multiple sources
   - Return filter options

4. **Enhance Statistics Endpoint**
   - Add active_agents count
   - Add pending_approvals count
   - Add calculated success_rate

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

- [ ] Activity Heatmap shows real data from last 90 days
- [ ] Agent Topology reflects config + live status
- [ ] Trace Waterfall renders real Observatory spans
- [ ] Message Queue streams live WebSocket events
- [ ] Statistics match `ailang coordinator status` output
- [ ] No hardcoded mock data in production build
- [ ] All new endpoints documented in API reference

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

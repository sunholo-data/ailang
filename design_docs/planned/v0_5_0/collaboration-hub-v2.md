# Collaboration Hub v2 - Comprehensive Metrics, History & Real-time Sync

**Status**: Planned
**Target**: v0.5.0
**Priority**: P0 - High
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure feature, not language syntax |
| Preserve Semantic Clarity | + | +1 | Clear visibility into agent execution state |
| Increase Determinism | + | +1 | Reliable real-time sync, consistent metrics |
| Lower Token Cost | + | +1 | Better observability reduces debugging iterations |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The Collaboration Hub has grown organically with several features implemented piecemeal. This has resulted in:

**Current State:**

1. **Lost Metrics**: CPU, tokens, cost metrics were removed during UI refactoring and not re-aggregated
   - Individual message metadata contains execution_stats but no rollup
   - No global, per-agent, or per-thread aggregation
   - Users have no visibility into resource consumption

2. **Approval Queue History Lost**: Approval requests are ephemeral
   - No history of approvals at agent or thread level
   - Can't see what was approved/rejected historically
   - No audit trail for capability grants

3. **"user" Agent Confusion**: The "user" agent is ambiguous
   - Should track CLI `ailang run` executions (outside Hub)
   - Shouldn't allow manual thread creation (user is human, not agent)
   - Currently shows as regular agent in hierarchy

4. **No Historic Instance Tracking**: Large eval suite runs spawn many instances
   - No way to see past instances that have completed
   - Instance history not preserved
   - Can't analyze patterns across runs

5. **No Trends Over Time**: No visualization of execution patterns
   - Agents created/destroyed over time
   - Threads per agent over time
   - Runs, execution time, tokens, cost over time
   - No dashboards or charts

6. **Sync Issues**: Real-time updates are unreliable
   - Running status doesn't always appear
   - Messages stuck on "sending..."
   - UI only updates on click, not in real-time
   - Need streaming feedback for long operations

7. **No Claude Code Integration**: Session history not captured
   - Claude Code conversations not recorded
   - Can't see what Claude Code sessions led to agent runs
   - Missing link between human IDE interaction and agent execution

**Impact:**
- Developers can't understand resource usage
- No visibility into historical patterns
- Debugging agent issues is difficult
- Trust in the system is reduced

## Goals

**Primary Goal:** Create a production-ready Collaboration Hub with comprehensive metrics, reliable real-time sync, and full execution history.

**Success Metrics:**
- 100% of execution metrics visible at all hierarchy levels
- <500ms latency for real-time status updates
- Full audit trail for all approvals
- Historical data retained for 30+ days
- Trends visualization with at least 5 chart types

## Solution Design

### Overview

This feature encompasses 7 sub-features organized into 4 implementation phases:

1. **Phase 1: Metrics Foundation** - Database schema, aggregation logic, API endpoints
2. **Phase 2: History & Audit** - Approval history, instance history, retention policies
3. **Phase 3: Real-time Sync** - WebSocket improvements, streaming, optimistic updates
4. **Phase 4: Visualization & Integration** - Charts, trends, Claude Code integration

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Collaboration Hub v2                        │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐ │
│  │   Monitor View  │  │  Message Center │  │   Approval Queue    │ │
│  │  - Hierarchy    │  │  - Threads      │  │  - Pending          │ │
│  │  - Metrics      │  │  - Messages     │  │  - History          │ │
│  │  - Trends       │  │  - Files        │  │  - Audit Trail      │ │
│  └────────┬────────┘  └────────┬────────┘  └──────────┬──────────┘ │
│           │                    │                       │            │
│  ┌────────┴────────────────────┴───────────────────────┴──────────┐ │
│  │                      WebSocket Layer                           │ │
│  │  - Real-time messages        - Telemetry streaming             │ │
│  │  - Optimistic updates        - Connection recovery             │ │
│  └────────────────────────────────┬───────────────────────────────┘ │
│                                   │                                 │
│  ┌────────────────────────────────┴───────────────────────────────┐ │
│  │                       REST API Layer                           │ │
│  │  - /api/metrics/*            - /api/approvals/*                │ │
│  │  - /api/history/*            - /api/trends/*                   │ │
│  └────────────────────────────────┬───────────────────────────────┘ │
│                                   │                                 │
│  ┌────────────────────────────────┴───────────────────────────────┐ │
│  │                      Storage Layer                             │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │ │
│  │  │  Messages   │  │  Approvals  │  │  Metrics Aggregates     │ │ │
│  │  │  (SQLite)   │  │  (SQLite)   │  │  (SQLite + Time-series) │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

**Components:**

1. **Metrics Aggregation Service**: Background worker that computes rollups
2. **History Service**: Manages retention and archival of historical data
3. **WebSocket Manager**: Enhanced real-time communication with reconnection
4. **Trends Engine**: Time-series data storage and query interface
5. **Claude Code Bridge**: Captures IDE session context

### Database Schema Changes

```sql
-- New table: metrics_aggregates
CREATE TABLE IF NOT EXISTS metrics_aggregates (
    id TEXT PRIMARY KEY,
    scope_type TEXT NOT NULL,      -- 'global', 'agent', 'thread'
    scope_id TEXT NOT NULL,        -- '' for global, agent_id, or thread_id
    period TEXT NOT NULL,          -- 'minute', 'hour', 'day'
    period_start INTEGER NOT NULL, -- Unix timestamp

    -- Metrics
    total_runs INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    total_cost_cents INTEGER DEFAULT 0,
    total_duration_ms INTEGER DEFAULT 0,
    total_files_modified INTEGER DEFAULT 0,

    -- Derived
    avg_tokens_per_run REAL DEFAULT 0,
    avg_cost_per_run REAL DEFAULT 0,
    avg_duration_per_run REAL DEFAULT 0,

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    UNIQUE(scope_type, scope_id, period, period_start)
);

-- New table: approval_history
CREATE TABLE IF NOT EXISTS approval_history (
    id TEXT PRIMARY KEY,
    approval_id TEXT NOT NULL,
    thread_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    action TEXT NOT NULL,          -- 'created', 'approved', 'rejected', 'expired'
    actor TEXT NOT NULL,           -- 'user' or agent_id
    proposal TEXT,
    impact TEXT,
    estimated_cost REAL,
    capability_token TEXT,
    created_at INTEGER NOT NULL,

    FOREIGN KEY (thread_id) REFERENCES threads(id)
);

-- New table: instance_history
CREATE TABLE IF NOT EXISTS instance_history (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    exit_code INTEGER,
    total_tokens INTEGER DEFAULT 0,
    total_cost_cents INTEGER DEFAULT 0,
    thread_count INTEGER DEFAULT 0,

    FOREIGN KEY (agent_id) REFERENCES agents(id)
);

-- New table: claude_sessions
CREATE TABLE IF NOT EXISTS claude_sessions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,      -- Claude Code session identifier
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    workspace TEXT,
    summary TEXT,
    agent_runs_triggered INTEGER DEFAULT 0,

    created_at INTEGER NOT NULL
);

-- Add index for time-series queries
CREATE INDEX IF NOT EXISTS idx_metrics_period ON metrics_aggregates(scope_type, period, period_start);
CREATE INDEX IF NOT EXISTS idx_approval_history_thread ON approval_history(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_instance_history_agent ON instance_history(agent_id, started_at);
```

### API Endpoints

**Metrics Endpoints:**
```
GET /api/metrics/global                     # Global aggregated metrics
GET /api/metrics/agents/:id                 # Per-agent metrics
GET /api/metrics/threads/:id                # Per-thread metrics
GET /api/metrics/trends?scope=global&period=hour&range=24h  # Time-series
```

**History Endpoints:**
```
GET /api/approvals/history?thread_id=...    # Approval history for thread
GET /api/approvals/history?agent_id=...     # Approval history for agent
GET /api/instances/history?agent_id=...     # Instance history for agent
GET /api/sessions                           # Claude Code sessions
```

**Enhanced Existing Endpoints:**
```
GET /api/hierarchy                          # Add metrics to each node
GET /api/threads/:id                        # Add approval_count, metrics
GET /api/agents/:id                         # Add instance_count, metrics
```

### UI Components

**1. Metrics Dashboard Widget:**
```tsx
interface MetricsSummary {
  totalRuns: number;
  totalTokens: number;
  totalCost: number;
  avgDuration: number;
  trend: 'up' | 'down' | 'stable';
}

<MetricsCard
  scope="global"
  metrics={summary}
  sparkline={last24Hours}
/>
```

**2. Approval History Panel:**
```tsx
<ApprovalHistory
  threadId={selectedThread}
  showActions={true}
  expandable={true}
/>
```

**3. Trends Chart:**
```tsx
<TrendsChart
  metric="tokens"          // tokens, cost, runs, duration
  period="hour"
  range="7d"
  groupBy="agent"          // optional: break down by agent
/>
```

**4. "user" Agent Special Handling:**
```tsx
// In Monitor.tsx hierarchy rendering
{node.id === 'user' ? (
  <UserAgentNode
    node={node}
    readOnly={true}        // No thread creation
    showCLIRuns={true}     // Show ailang CLI executions
  />
) : (
  <AgentNode node={node} />
)}
```

### Implementation Plan

**Phase 1: Metrics Foundation** (~8 hours)
- [ ] Add metrics_aggregates table and migrations
- [ ] Create MetricsService in `internal/messaging/metrics.go`
- [ ] Implement aggregation on message creation (trigger)
- [ ] Add `/api/metrics/*` endpoints to server
- [ ] Create MetricsCard React component
- [ ] Display metrics in hierarchy nodes
- [ ] Unit tests for aggregation logic

**Phase 2: History & Audit** (~6 hours)
- [ ] Add approval_history and instance_history tables
- [ ] Record approval state changes automatically
- [ ] Track instance lifecycle events
- [ ] Add `/api/approvals/history` endpoint
- [ ] Add `/api/instances/history` endpoint
- [ ] Create ApprovalHistory React component
- [ ] Add history tab to thread detail view
- [ ] Implement retention policy (30 day default)

**Phase 3: Real-time Sync** (~6 hours)
- [ ] Fix WebSocket polling race conditions
- [ ] Add connection state indicator to UI
- [ ] Implement optimistic updates for message sending
- [ ] Add streaming support for long operations
- [ ] Handle reconnection gracefully
- [ ] Add "running" spinner with elapsed time
- [ ] Test under network degradation

**Phase 4: Visualization & Integration** (~8 hours)
- [ ] Create TrendsChart component (using Recharts)
- [ ] Add trends API endpoint with time-series queries
- [ ] Implement "user" agent special behavior
- [ ] Add Claude Code session capture hooks
- [ ] Create sessions list view
- [ ] Link sessions to triggered agent runs
- [ ] Add dashboard overview page

### Files to Modify/Create

**New files:**
- `internal/messaging/metrics.go` - Metrics aggregation service (~300 LOC)
- `internal/messaging/history.go` - History tracking service (~200 LOC)
- `internal/server/metrics_handlers.go` - Metrics API endpoints (~150 LOC)
- `internal/server/history_handlers.go` - History API endpoints (~150 LOC)
- `ui/src/components/MetricsCard/MetricsCard.tsx` - Metrics display (~150 LOC)
- `ui/src/components/TrendsChart/TrendsChart.tsx` - Time-series charts (~200 LOC)
- `ui/src/components/ApprovalHistory/ApprovalHistory.tsx` - Approval log (~150 LOC)

**Modified files:**
- `internal/messaging/store.go` - Add new tables, migration (~100 LOC)
- `internal/messaging/operations.go` - Trigger metrics on message create (~50 LOC)
- `internal/websocket/server.go` - Fix sync issues, add streaming (~100 LOC)
- `internal/server/server.go` - Register new routes (~30 LOC)
- `ui/src/components/Monitor/Monitor.tsx` - Add metrics to hierarchy (~100 LOC)
- `ui/src/components/ApprovalQueue/ApprovalQueue.tsx` - Add history tab (~80 LOC)
- `ui/src/components/MessageCenter/ConversationView.tsx` - Running indicator improvements (~50 LOC)
- `ui/src/App.tsx` - "user" agent special handling (~50 LOC)
- `ui/src/types/index.ts` - New type definitions (~50 LOC)

## Examples

### Example 1: Viewing Aggregated Metrics

**Before:**
- No metrics visible anywhere
- Have to manually count messages
- Cost/tokens not aggregated

**After:**
```
┌─────────────────────────────────────────┐
│  📊 Global Metrics (Last 24h)           │
│  ─────────────────────────────────────  │
│  Runs: 47        Tokens: 125,340        │
│  Cost: $2.45     Avg Duration: 12.3s    │
│  ▁▂▃▅▆▇▆▅▃▂▁▂▃▅▆ (hourly trend)        │
└─────────────────────────────────────────┘
```

### Example 2: Approval History

**Before:**
- Approval disappears after action
- No audit trail
- Can't see what was approved

**After:**
```
┌─────────────────────────────────────────┐
│  📋 Approval History                    │
│  ─────────────────────────────────────  │
│  ✅ 10:32 - File write approved         │
│     /src/utils.ts (user)                │
│  ❌ 10:28 - Network request rejected    │
│     POST api.example.com (user)         │
│  ✅ 10:15 - Shell command approved      │
│     npm install lodash (user)           │
└─────────────────────────────────────────┘
```

### Example 3: "user" Agent Behavior

**Before:**
- "user" shows as regular agent
- Can create threads under "user"
- Confusing semantics

**After:**
```
👤 CLI Executions (read-only)
   Shows `ailang run` commands from terminal
   Cannot create new threads here

🤖 sprint-planner (agent)
   Can create threads
   Shows agent-initiated conversations
```

## Success Criteria

- [ ] Global metrics display shows total runs, tokens, cost, duration
- [ ] Per-agent metrics visible in hierarchy hover/expansion
- [ ] Per-thread metrics visible in thread header
- [ ] Approval history shows all state changes with timestamps
- [ ] Instance history shows past agent runs with outcomes
- [ ] WebSocket status indicator visible in UI
- [ ] Messages update in <500ms after agent writes to DB
- [ ] Running status shows spinner with elapsed time
- [ ] "user" agent is read-only in UI
- [ ] Trends chart shows at least tokens/cost over time
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Metrics aggregation calculations
- History recording triggers
- Time-series query logic
- Retention policy enforcement

**Integration tests:**
- API endpoint responses match expected schemas
- WebSocket message delivery timing
- Database migration correctness
- Concurrent aggregation updates

**Manual testing:**
- Start agent, verify metrics update in real-time
- Submit approval, verify history records
- Disconnect WebSocket, verify reconnection
- Run large eval suite, verify instance history
- Check "user" agent doesn't allow thread creation

## Non-Goals

**Not in this feature:**
- Custom dashboards (user-defined) - Future work
- External metrics export (Prometheus/Grafana) - Future work
- Multi-user authentication - Separate feature
- Mobile-responsive UI - Separate feature
- Real-time collaboration (multiple users) - Separate feature

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Database size growth from metrics | Medium | Implement automatic rollup (minute→hour→day) and retention policies |
| Performance impact of aggregation | Medium | Use background worker, batch updates, optimize indexes |
| WebSocket reliability | High | Add reconnection logic, fallback to polling, connection state UI |
| Chart library bundle size | Low | Use dynamic imports for Recharts |
| Breaking existing UI | Medium | Feature flags for gradual rollout |

## References

- [WebSocket Server](../../internal/websocket/server.go) - Current implementation
- [Messaging Operations](../../internal/messaging/operations.go) - Database operations
- [UI Components](../../ui/src/components/) - React components
- [Recharts Documentation](https://recharts.org/) - Charting library

## Future Work

- **Custom Dashboards**: Let users create their own metric views
- **Alerting**: Notify when metrics exceed thresholds
- **Export**: Prometheus metrics endpoint, CSV export
- **Comparison**: Compare metrics between time periods
- **Annotations**: Mark events on trend charts (releases, etc.)

---

**Document created**: 2025-11-30
**Last updated**: 2025-11-30

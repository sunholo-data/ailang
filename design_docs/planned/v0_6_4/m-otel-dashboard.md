# M-OTEL-DASHBOARD: Unified Observability Platform

**Status**: Planned (Foundation Complete)
**Target**: v0.6.4
**Priority**: P0 (High)
**Estimated**: 6 days (48 hours) - reduced from 8 days due to foundation work
**Dependencies**: M-OTEL-CROSS-PROCESS (v0.6.3), M-OTEL-EXTENDED (v0.6.3)

> **Note (2026-01-04)**: Foundation work completed in v0.6.4-pre. See [implemented/v0_6_4/m-otel-dashboard-foundation.md](../../implemented/v0_6_4/m-otel-dashboard-foundation.md) for details on OTLP receiver, Claude Code integration, and UI enhancements.

## Executive Summary

Build a new observability dashboard from scratch with OTEL-standard data foundations. This replaces the existing Collaboration Hub with a portable, pluggable platform designed to be a key selling point for AILANG.

**Design Philosophy:** Data model first. Get the foundations right, then build rich visualizations on top.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A2: Replayability | +1 | Full span storage enables trace replay |
| A7: Machines First | +1 | OTEL-standard data is machine-readable |
| A9: Cost Visibility | +1 | Aggregated costs at every level |
| A12: System Boundary | +1 | External CLI telemetry explicitly tracked |

**Net Score: +4** → **Decision: Move forward**

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary storage | Local SQLite | Fast queries, no cloud dependency, portable |
| Real-time updates | WebSocket | Reliable, works local and cloud |
| Backend support | Pluggable adapters | GCP, Jaeger, Grafana, Honeycomb via adapters |
| Root entity | Task | Everything links to task_id for correlation |
| Provider normalization | Common model | Map Claude metrics + Gemini traces to same schema |
| Span granularity | Full + attributes | Rich querying, ~1KB per span |
| Messages | Separate table | Richer model than span events |
| Approvals | Span events | Captures exact timing within execution |
| Primary view | Agent activity | Live execution monitoring |
| Deployment | Portable | Works equally well local or cloud |
| Offline mode | Not required | Assume always online |

## Data Model

### Entity Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              WORKSPACE                                   │
│  (git repo, project root)                                               │
│  Aggregates: total_cost, total_tasks, success_rate, active_agents       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                            TASK                                  │    │
│  │  (GitHub issue, message, email trigger)                          │    │
│  │  Aggregates: duration, tokens_total, cost_total, agent_count     │    │
│  │                                                                   │    │
│  │  ┌─────────────────────┐   ┌─────────────────────┐               │    │
│  │  │      MESSAGES       │   │    AGENT ASSIGN     │               │    │
│  │  │  (separate table)   │   │  (coordinator →     │               │    │
│  │  │  FK: task_id        │   │   delegated agent)  │               │    │
│  │  └─────────────────────┘   └──────────┬──────────┘               │    │
│  │                                       │                           │    │
│  │                           ┌───────────▼───────────┐               │    │
│  │                           │   AGENT EXECUTION     │               │    │
│  │                           │  (Claude/Gemini/etc)  │               │    │
│  │                           │  Aggregates: time,    │               │    │
│  │                           │  tool_calls, errors   │               │    │
│  │                           └───────────┬───────────┘               │    │
│  │                                       │                           │    │
│  │                           ┌───────────▼───────────┐               │    │
│  │                           │        SPANS          │               │    │
│  │                           │  (ailang CLI, compile,│               │    │
│  │                           │   eval, messages)     │               │    │
│  │                           │                       │               │    │
│  │                           │  ┌─────────────────┐  │               │    │
│  │                           │  │     EVENTS      │  │               │    │
│  │                           │  │  (approvals,    │  │               │    │
│  │                           │  │   tool calls)   │  │               │    │
│  │                           │  └─────────────────┘  │               │    │
│  │                           └───────────────────────┘               │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

### SQLite Schema

```sql
-- Workspaces (git repos, projects)
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    git_remote TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks (root entity - from GitHub, messages, email, etc.)
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    title TEXT NOT NULL,
    description TEXT,
    source_type TEXT NOT NULL,  -- 'github_issue', 'message', 'email', 'manual'
    source_ref TEXT,            -- e.g., 'sunholo-data/ailang#123'
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, completed, failed
    priority TEXT DEFAULT 'medium',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Aggregated metrics (updated on span completion)
    total_duration_ms INTEGER DEFAULT 0,
    total_tokens_in INTEGER DEFAULT 0,
    total_tokens_out INTEGER DEFAULT 0,
    total_cost_usd REAL DEFAULT 0.0,
    agent_count INTEGER DEFAULT 0,
    span_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0
);

-- Agent assignments (coordinator → agent delegation)
CREATE TABLE agent_assignments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    agent_id TEXT NOT NULL,      -- 'design-doc-creator', 'sprint-planner', etc.
    provider TEXT NOT NULL,      -- 'claude', 'gemini', 'ollama'
    status TEXT NOT NULL DEFAULT 'pending',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    parent_assignment_id TEXT REFERENCES agent_assignments(id),  -- for delegation chains

    -- Agent-level aggregates
    duration_ms INTEGER DEFAULT 0,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0.0,
    tool_calls INTEGER DEFAULT 0,
    turns INTEGER DEFAULT 0
);

-- Spans (full OTEL spans with attributes)
CREATE TABLE spans (
    id TEXT PRIMARY KEY,           -- span_id
    trace_id TEXT NOT NULL,
    parent_span_id TEXT,
    task_id TEXT REFERENCES tasks(id),
    agent_assignment_id TEXT REFERENCES agent_assignments(id),

    name TEXT NOT NULL,            -- 'compile.pipeline', 'executor.claude.execute', etc.
    kind TEXT NOT NULL,            -- 'internal', 'client', 'server', 'producer', 'consumer'
    status TEXT NOT NULL,          -- 'ok', 'error', 'unset'
    status_message TEXT,

    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration_ms INTEGER NOT NULL,

    -- Normalized attributes (common across providers)
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost_usd REAL,
    model TEXT,
    provider TEXT,

    -- Full attributes as JSON (for provider-specific data)
    attributes JSON NOT NULL DEFAULT '{}',

    -- Resource attributes (service info)
    resource_attributes JSON NOT NULL DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Span events (approvals, tool calls, errors, etc.)
CREATE TABLE span_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    span_id TEXT NOT NULL REFERENCES spans(id),
    name TEXT NOT NULL,            -- 'approval.requested', 'tool.call', 'error'
    timestamp TIMESTAMP NOT NULL,
    attributes JSON NOT NULL DEFAULT '{}',

    -- Denormalized for common event types
    event_type TEXT,               -- 'approval', 'tool', 'error', 'custom'
    approval_status TEXT,          -- 'pending', 'approved', 'rejected'
    tool_name TEXT,
    error_message TEXT
);

-- Messages (separate table for rich message model)
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id),
    inbox TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    message_type TEXT DEFAULT 'general',
    status TEXT NOT NULL DEFAULT 'unread',
    priority TEXT DEFAULT 'normal',

    -- GitHub sync
    github_issue_number INTEGER,
    github_repo TEXT,

    -- Correlation
    correlation_id TEXT,
    reply_to_id TEXT REFERENCES messages(id),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    archived_at TIMESTAMP,

    -- Search
    content_hash TEXT,             -- for deduplication
    embedding BLOB                 -- for semantic search
);

-- Indexes for common queries
CREATE INDEX idx_tasks_workspace ON tasks(workspace_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created ON tasks(created_at DESC);

CREATE INDEX idx_agent_assignments_task ON agent_assignments(task_id);
CREATE INDEX idx_agent_assignments_agent ON agent_assignments(agent_id);
CREATE INDEX idx_agent_assignments_status ON agent_assignments(status);

CREATE INDEX idx_spans_trace ON spans(trace_id);
CREATE INDEX idx_spans_task ON spans(task_id);
CREATE INDEX idx_spans_agent ON spans(agent_assignment_id);
CREATE INDEX idx_spans_name ON spans(name);
CREATE INDEX idx_spans_time ON spans(start_time DESC);

CREATE INDEX idx_span_events_span ON span_events(span_id);
CREATE INDEX idx_span_events_type ON span_events(event_type);

CREATE INDEX idx_messages_task ON messages(task_id);
CREATE INDEX idx_messages_inbox ON messages(inbox);
CREATE INDEX idx_messages_status ON messages(status);

-- Aggregation views
CREATE VIEW workspace_stats AS
SELECT
    w.id,
    w.name,
    COUNT(DISTINCT t.id) as task_count,
    SUM(t.total_cost_usd) as total_cost,
    SUM(t.total_tokens_in + t.total_tokens_out) as total_tokens,
    COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN t.id END) * 100.0 /
        NULLIF(COUNT(DISTINCT t.id), 0) as success_rate,
    COUNT(DISTINCT aa.agent_id) as unique_agents
FROM workspaces w
LEFT JOIN tasks t ON t.workspace_id = w.id
LEFT JOIN agent_assignments aa ON aa.task_id = t.id
GROUP BY w.id;

CREATE VIEW agent_stats AS
SELECT
    aa.agent_id,
    aa.provider,
    COUNT(*) as execution_count,
    SUM(aa.duration_ms) as total_duration_ms,
    AVG(aa.duration_ms) as avg_duration_ms,
    SUM(aa.tokens_in) as total_tokens_in,
    SUM(aa.tokens_out) as total_tokens_out,
    SUM(aa.cost_usd) as total_cost,
    SUM(aa.tool_calls) as total_tool_calls,
    COUNT(CASE WHEN aa.status = 'completed' THEN 1 END) * 100.0 /
        NULLIF(COUNT(*), 0) as success_rate
FROM agent_assignments aa
GROUP BY aa.agent_id, aa.provider;
```

### Normalized Span Model

To handle the Claude (metrics only) vs Gemini (full traces) asymmetry:

```go
// NormalizedSpan is our common model for all provider spans
type NormalizedSpan struct {
    // Identity
    ID            string    `json:"id"`
    TraceID       string    `json:"trace_id"`
    ParentSpanID  string    `json:"parent_span_id,omitempty"`
    TaskID        string    `json:"task_id,omitempty"`
    AgentAssignID string    `json:"agent_assignment_id,omitempty"`

    // Timing
    Name          string    `json:"name"`
    Kind          SpanKind  `json:"kind"`
    Status        SpanStatus `json:"status"`
    StatusMessage string    `json:"status_message,omitempty"`
    StartTime     time.Time `json:"start_time"`
    EndTime       time.Time `json:"end_time"`
    DurationMs    int64     `json:"duration_ms"`

    // Normalized metrics (always present regardless of provider)
    TokensIn      int       `json:"tokens_in,omitempty"`
    TokensOut     int       `json:"tokens_out,omitempty"`
    CostUSD       float64   `json:"cost_usd,omitempty"`
    Model         string    `json:"model,omitempty"`
    Provider      string    `json:"provider,omitempty"`

    // Children (for tree building)
    Children      []*NormalizedSpan `json:"children,omitempty"`

    // Full attributes (provider-specific)
    Attributes    map[string]any `json:"attributes,omitempty"`
    Resource      map[string]any `json:"resource,omitempty"`

    // Events
    Events        []SpanEvent `json:"events,omitempty"`
}

// SpanEvent represents approvals, tool calls, errors, etc.
type SpanEvent struct {
    Name       string         `json:"name"`
    Timestamp  time.Time      `json:"timestamp"`
    Type       string         `json:"type"` // approval, tool, error, custom
    Attributes map[string]any `json:"attributes,omitempty"`

    // Denormalized for common types
    ApprovalStatus string `json:"approval_status,omitempty"`
    ToolName       string `json:"tool_name,omitempty"`
    ErrorMessage   string `json:"error_message,omitempty"`
}
```

### Provider Normalization

```go
// Normalize Claude metrics to our span model
func NormalizeClaudeMetrics(taskID string, metrics *ClaudeMetrics) *NormalizedSpan {
    return &NormalizedSpan{
        ID:        generateSpanID(),
        TraceID:   metrics.ResourceAttributes["ailang.trace_id"],
        TaskID:    taskID,
        Name:      "executor.claude.execute",
        Kind:      SpanKindInternal,
        Status:    inferStatus(metrics),
        StartTime: metrics.StartTime,
        EndTime:   metrics.EndTime,
        DurationMs: metrics.EndTime.Sub(metrics.StartTime).Milliseconds(),

        // Normalized from metrics
        TokensIn:  metrics.InputTokens,
        TokensOut: metrics.OutputTokens,
        CostUSD:   metrics.TotalCost,
        Model:     metrics.Model,
        Provider:  "claude",

        // Keep original metrics
        Attributes: map[string]any{
            "claude.session_id":  metrics.SessionID,
            "claude.tool_calls":  metrics.ToolCalls,
            "claude.api_calls":   metrics.APICalls,
        },
    }
}

// Normalize Gemini OTEL spans to our model
func NormalizeGeminiSpan(taskID string, span *otelSpan) *NormalizedSpan {
    return &NormalizedSpan{
        ID:            span.SpanID,
        TraceID:       span.TraceID,
        ParentSpanID:  span.ParentSpanID,
        TaskID:        taskID,
        Name:          span.Name,
        Kind:          SpanKind(span.Kind),
        Status:        SpanStatus(span.Status),
        StartTime:     span.StartTime,
        EndTime:       span.EndTime,
        DurationMs:    span.EndTime.Sub(span.StartTime).Milliseconds(),

        // Extract normalized metrics from attributes
        TokensIn:  getIntAttr(span, "gen_ai.usage.input_tokens"),
        TokensOut: getIntAttr(span, "gen_ai.usage.output_tokens"),
        CostUSD:   getFloatAttr(span, "gen_ai.usage.cost"),
        Model:     getStringAttr(span, "gen_ai.request.model"),
        Provider:  "gemini",

        // Keep all attributes
        Attributes: span.Attributes,
        Events:     normalizeEvents(span.Events),
    }
}
```

## Backend Adapters

```go
// Backend defines the interface for OTEL data backends
type Backend interface {
    // Ingest
    IngestSpan(ctx context.Context, span *NormalizedSpan) error
    IngestSpans(ctx context.Context, spans []*NormalizedSpan) error

    // Query
    GetTrace(ctx context.Context, traceID string) (*Trace, error)
    ListTraces(ctx context.Context, query *TraceQuery) ([]*TraceSummary, error)
    GetSpan(ctx context.Context, spanID string) (*NormalizedSpan, error)

    // Aggregations
    GetTaskMetrics(ctx context.Context, taskID string) (*TaskMetrics, error)
    GetAgentMetrics(ctx context.Context, agentID string, timeRange TimeRange) (*AgentMetrics, error)
    GetProviderComparison(ctx context.Context, timeRange TimeRange) (*ProviderComparison, error)
}

// SQLiteBackend is the default local backend
type SQLiteBackend struct {
    db *sql.DB
}

// GCPBackend queries Google Cloud Trace
type GCPBackend struct {
    client  *cloudtrace.Client
    project string
}

// JaegerBackend queries Jaeger API
type JaegerBackend struct {
    endpoint string
    client   *http.Client
}

// CompositeBackend writes to local, reads from local or remote
type CompositeBackend struct {
    local  *SQLiteBackend
    remote Backend // optional, for GCP/Jaeger queries
}
```

## API Endpoints

### Workspaces
```
GET  /api/workspaces                    - List all workspaces
POST /api/workspaces                    - Create workspace
GET  /api/workspaces/:id                - Get workspace with stats
GET  /api/workspaces/:id/tasks          - List tasks in workspace
GET  /api/workspaces/:id/agents         - List active agents
GET  /api/workspaces/:id/metrics        - Aggregated workspace metrics
```

### Tasks
```
GET  /api/tasks                         - List tasks (with filters)
POST /api/tasks                         - Create task
GET  /api/tasks/:id                     - Get task with aggregates
GET  /api/tasks/:id/agents              - List agent assignments
GET  /api/tasks/:id/spans               - List spans for task
GET  /api/tasks/:id/messages            - List messages for task
GET  /api/tasks/:id/timeline            - Span timeline for visualization
```

### Agents
```
GET  /api/agents                        - List all agents with stats
GET  /api/agents/:id                    - Get agent details
GET  /api/agents/:id/executions         - List agent executions
GET  /api/agents/:id/metrics            - Agent performance metrics
```

### Traces & Spans
```
GET  /api/traces/:trace_id              - Get full trace tree
GET  /api/spans/:span_id                - Get span with events
GET  /api/spans/:span_id/events         - List span events
POST /api/spans                         - Ingest spans (from OTEL collector)
```

### Metrics
```
GET  /api/metrics/summary               - Global metrics summary
GET  /api/metrics/providers             - Provider comparison (Claude vs Gemini)
GET  /api/metrics/timeline              - Time-series metrics
GET  /api/metrics/costs                 - Cost breakdown by provider/agent/task
```

### Messages
```
GET  /api/messages                      - List messages (with filters)
POST /api/messages                      - Send message
GET  /api/messages/:id                  - Get message
PUT  /api/messages/:id/ack              - Acknowledge message
POST /api/messages/search               - Semantic search
```

### WebSocket
```
WS  /ws                                 - Real-time updates
    Events:
    - task.created, task.updated, task.completed
    - agent.started, agent.completed, agent.failed
    - span.created, span.completed
    - approval.requested, approval.resolved
    - message.received
```

## Implementation Plan

### Phase 1: Data Layer (~16 hours)
- [ ] Create SQLite schema and migrations
- [ ] Implement NormalizedSpan model and normalization functions
- [ ] Create SQLiteBackend with all CRUD operations
- [ ] Add aggregation update triggers
- [ ] Unit tests for data layer

### Phase 2: Ingestion Pipeline (~8 hours)
- [ ] Create OTEL span receiver (HTTP endpoint)
- [ ] Implement Claude metrics → NormalizedSpan conversion
- [ ] Implement Gemini OTEL → NormalizedSpan conversion
- [ ] Real-time aggregation updates
- [ ] Integration with existing coordinator

### Phase 3: Backend Adapters (~8 hours)
- [ ] Implement GCPBackend for Google Cloud Trace
- [ ] Implement JaegerBackend for Jaeger API
- [ ] Create CompositeBackend (local write, remote read)
- [ ] Adapter configuration and factory

### Phase 4: REST API (~12 hours)
- [ ] Implement all REST endpoints
- [ ] Add query filters and pagination
- [ ] OpenAPI spec generation
- [ ] API tests

### Phase 5: WebSocket Layer (~8 hours)
- [ ] WebSocket hub with connection management
- [ ] Event broadcasting on data changes
- [ ] Subscription filtering (by workspace, task, agent)
- [ ] Reconnection handling

### Phase 6: Frontend Foundation (~12 hours)
- [ ] New React app structure
- [ ] Data hooks (useWorkspaces, useTasks, useAgents, useSpans)
- [ ] WebSocket integration
- [ ] Base component library

## Success Criteria

- [ ] SQLite schema supports full hierarchy (workspace → task → agent → span → event)
- [ ] Normalized span model works for both Claude and Gemini
- [ ] Aggregations update automatically on span completion
- [ ] REST API covers all entities with proper filtering
- [ ] WebSocket delivers real-time updates < 100ms latency
- [ ] Backend adapters work for SQLite, GCP, and Jaeger
- [ ] Provider comparison shows accurate side-by-side metrics
- [ ] All tests passing

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `internal/observatory/schema.sql` | SQLite schema | ~150 |
| `internal/observatory/models.go` | Go types for all entities | ~300 |
| `internal/observatory/normalize.go` | Provider normalization | ~200 |
| `internal/observatory/store.go` | SQLite CRUD operations | ~400 |
| `internal/observatory/backend.go` | Backend interface | ~100 |
| `internal/observatory/backend_sqlite.go` | SQLite backend | ~300 |
| `internal/observatory/backend_gcp.go` | GCP Trace backend | ~200 |
| `internal/observatory/backend_jaeger.go` | Jaeger backend | ~150 |
| `internal/observatory/handlers.go` | REST API handlers | ~500 |
| `internal/observatory/websocket.go` | WebSocket hub | ~200 |
| `internal/observatory/aggregations.go` | Metric aggregations | ~200 |

**Total Backend: ~2,700 LOC**

## Timeline

| Phase | Effort | Description |
|-------|--------|-------------|
| P1: Data Layer | 16h | Schema, models, SQLite backend |
| P2: Ingestion | 8h | OTEL receiver, normalization |
| P3: Adapters | 8h | GCP, Jaeger backends |
| P4: REST API | 12h | All endpoints |
| P5: WebSocket | 8h | Real-time updates |
| P6: Frontend Foundation | 12h | React setup, hooks |

**Total: 64 hours (~8 days)**

## External References

- [Claude Code Telemetry](https://docs.anthropic.com/en/docs/claude-code/cli#using-telemetry)
- [Gemini CLI Telemetry](https://ai.google.dev/gemini-api/docs/agentic/cli#telemetry)
- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [Google Cloud Trace API](https://cloud.google.com/trace/docs/reference/v2/rest)

---

**Document created**: 2026-01-04
**Last updated**: 2026-01-04

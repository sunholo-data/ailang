# Observatory Architecture

**Status**: Implemented (v0.6.4)
**Package**: `internal/observatory/`
**Total Lines**: ~12,670 (25 Go files)
**Tests**: 30 passing

---

## Executive Summary

The Observatory is AILANG's unified observability platform that stores, queries, and visualizes execution traces from:
- AILANG compiler and runtime operations
- Coordinator task execution
- Claude Code CLI sessions (headless mode)
- Gemini CLI executions
- Eval harness benchmarks

It provides a complete hierarchical view of all AI-assisted development activity: **Workspace → Task → Agent Assignment → Span → Span Event**.

---

## System Vision

### Objectives

1. **Unified Telemetry Storage**: Single database for all trace data regardless of source
2. **Task Hierarchy Tracking**: Link scattered telemetry events to business-level tasks
3. **Real-Time Dashboard**: WebSocket-based streaming for live monitoring
4. **Multi-Backend Support**: Local SQLite, GCP Cloud Trace, Jaeger (federated queries)
5. **Cost Attribution**: Automatic cost calculation from token counts using models.yml pricing

### Design Principles

1. **OpenTelemetry Native**: OTLP protocol for receiving spans (protobuf + JSON)
2. **Session Correlation**: Link orphaned Claude Code events retroactively via session IDs
3. **No Silent Fallbacks**: Return $0.00 for unknown models rather than guess
4. **Normalized Attributes**: Extract common fields (tokens, cost, model) from provider-specific schemas

---

## Entity Model

### Core Entities

```
┌─────────────────────────────────────────────────────────────────┐
│                         WORKSPACE                               │
│  (Git repository or project root)                               │
│  - id, name, path, git_remote                                   │
├─────────────────────────────────────────────────────────────────┤
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                        TASK                              │   │
│  │  (Unit of work: bug fix, feature, research)             │   │
│  │  - id, workspace_id, title, description                 │   │
│  │  - source_type: github_issue | message | manual         │   │
│  │  - status: pending | running | completed | failed       │   │
│  │  - Aggregated: total_tokens, total_cost, span_count     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  AGENT_ASSIGNMENT                        │   │
│  │  (Coordinator → Agent delegation)                        │   │
│  │  - id, task_id, agent_id, provider                      │   │
│  │  - status, duration_ms, tokens_in/out, cost_usd         │   │
│  │  - parent_assignment_id (for sub-agent chains)          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                        SPAN                              │   │
│  │  (OTEL span with normalized attributes)                 │   │
│  │  - id, trace_id, parent_span_id                         │   │
│  │  - task_id, agent_assignment_id (hierarchy links)       │   │
│  │  - name, kind, status, start_time, end_time             │   │
│  │  - tokens_in, tokens_out, cost_usd, model, provider     │   │
│  │  - attributes (JSON), resource_attributes (JSON)        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     SPAN_EVENT                           │   │
│  │  (Approval, tool call, error events)                    │   │
│  │  - id, span_id, name, timestamp, event_type             │   │
│  │  - approval_status, tool_name, error_message            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                      MESSAGE                             │   │
│  │  (Agent-to-agent messaging)                             │   │
│  │  - id, task_id, inbox, from_agent                       │   │
│  │  - title, content, message_type, status                 │   │
│  │  - github_issue_number (for sync)                       │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### SQL Schema Location

- Schema definition: `internal/observatory/migrate.go`
- Views: `workspace_stats`, `agent_stats`, `provider_comparison`, `task_timeline`

---

## Component Architecture

### File Structure (Post-Refactoring)

```
internal/observatory/
├── backend.go              # Backend interface (71 lines)
├── backend_sqlite.go       # SQLiteBackend implementation (295 lines)
├── backend_controlplane.go # Control plane types + queries (552 lines)
├── backend_gcp.go          # GCP Cloud Trace backend (~200 lines)
├── backend_jaeger.go       # Jaeger backend (~150 lines)
├── backend_composite.go    # Multi-backend federation (~180 lines)
├── store.go                # Store struct + workspace ops (113 lines)
├── store_tasks.go          # Task CRUD (152 lines)
├── store_agents.go         # Agent assignment CRUD (132 lines)
├── store_spans.go          # Span CRUD + trace queries (442 lines)
├── store_span_events.go    # Span event CRUD (68 lines)
├── store_messages.go       # Message CRUD (217 lines)
├── store_aggregates.go     # Aggregate queries (233 lines)
├── models.go               # Entity type definitions (411 lines)
├── otlp_receiver.go        # OTLP HTTP receiver (759 lines)
├── websocket.go            # Real-time WebSocket hub (409 lines)
├── api.go                  # REST API handlers (~600 lines)
├── pricing.go              # Cost calculation (~100 lines)
├── normalize.go            # Provider normalization (~200 lines)
├── hierarchy.go            # Task hierarchy queries (~300 lines)
├── aggregations.go         # Metrics aggregation (~150 lines)
└── migrate.go              # Schema migrations (~400 lines)
```

### Backend Interface

```go
type Backend interface {
    // Workspace operations
    CreateWorkspace(ctx context.Context, w *Workspace) error
    GetWorkspace(ctx context.Context, id string) (*Workspace, error)
    ListWorkspaces(ctx context.Context) ([]*Workspace, error)
    UpdateWorkspace(ctx context.Context, w *Workspace) error
    DeleteWorkspace(ctx context.Context, id string) error
    GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error)

    // Task operations
    CreateTask(ctx context.Context, t *Task) error
    GetTask(ctx context.Context, id string) (*Task, error)
    ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error)
    UpdateTask(ctx context.Context, t *Task) error
    DeleteTask(ctx context.Context, id string) error

    // Agent assignment operations
    CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error
    GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error)
    ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error)
    UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error
    DeleteAgentAssignment(ctx context.Context, id string) error
    GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error)

    // Span operations
    CreateSpan(ctx context.Context, span *Span) error
    GetSpan(ctx context.Context, id string) (*Span, error)
    ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error)
    UpdateSpan(ctx context.Context, span *Span) error
    UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error
    RecalculateTaskAggregates(ctx context.Context, taskID string) error
    DeleteSpan(ctx context.Context, id string) error
    GetTrace(ctx context.Context, traceID string) (*Trace, error)
    ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error)

    // Session correlation
    LookupTaskBySessionID(ctx context.Context, sessionID string) (taskID, assignmentID, traceID string)
    LinkOrphanedSpansBySession(ctx context.Context, sessionID, taskID, assignmentID string) (int64, error)

    // Span events
    CreateSpanEvent(ctx context.Context, e *SpanEvent) error
    GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error)
    DeleteSpanEvent(ctx context.Context, id int64) error

    // Messages
    CreateMessage(ctx context.Context, m *Message) error
    GetMessage(ctx context.Context, id string) (*Message, error)
    ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error)
    UpdateMessage(ctx context.Context, m *Message) error
    DeleteMessage(ctx context.Context, id string) error
    MarkMessageRead(ctx context.Context, id string) error
    MarkMessageArchived(ctx context.Context, id string) error

    // Aggregates
    GetMetricsSummary(ctx context.Context) (*MetricsSummary, error)
    GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error)
    GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error)

    // Lifecycle
    Close() error
}
```

### OTLP Receiver

Receives telemetry via standard OpenTelemetry HTTP protocol:

```
POST /v1/traces  → handleTraces()   → processResourceSpans() → CreateSpan()
POST /v1/logs    → handleLogs()     → processResourceLogs()  → convertLogToSpan() → CreateSpan()
POST /v1/metrics → handleMetrics()  → (acknowledged but not stored)
```

**Key features:**
- Supports both protobuf and JSON content types
- Filters noisy spans (health checks, static assets, polling endpoints)
- Extracts normalized attributes (tokens, cost, model) from multiple conventions:
  - `gen_ai.*` (OpenTelemetry semantic conventions)
  - `ailang.*` (AILANG custom)
  - `ai.*` (internal providers)
  - `task.*` (coordinator executor)

**Session Correlation (M-TASK-HIERARCHY-SESSION-LINKING):**
```
1. Claude Code event arrives via /v1/logs (has session.id but no task_id)
2. Look up parent claude.execute span by session.id
3. If found, inherit task_id and assignment_id from parent
4. When claude.execute span arrives later (OTEL batching delay):
   - Link all orphaned Claude Code events retroactively
   - Recalculate task aggregates
```

### WebSocket Hub

Real-time event streaming to dashboard:

```go
// Event types
EventTypeSpanCreated       = "span.created"
EventTypeSpanUpdated       = "span.updated"
EventTypeTaskCreated       = "task.created"
EventTypeTaskUpdated       = "task.updated"
EventTypeTaskCompleted     = "task.completed"
EventTypeMessageCreated    = "message.created"
EventTypeApprovalRequested = "approval.requested"
EventTypeApprovalDecision  = "approval.decision"
EventTypeMetricsUpdated    = "metrics.updated"
```

**Subscription filtering:**
- Filter by workspace ID
- Filter by task ID
- Filter by event types

---

## Data Flow Diagrams

### 1. Telemetry Ingestion Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                        TELEMETRY SOURCES                             │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │   AILANG    │  │ Coordinator │  │ Claude Code │  │ Gemini CLI  │ │
│  │  Compiler   │  │   Daemon    │  │   (OTEL)    │  │   (OTEL)    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
│         │                │                │                │        │
│         └────────────────┴────────────────┴────────────────┘        │
│                                    │                                 │
│                                    ▼                                 │
│                        ┌────────────────────┐                       │
│                        │   OTLP Receiver    │                       │
│                        │  POST /v1/traces   │                       │
│                        │  POST /v1/logs     │                       │
│                        └─────────┬──────────┘                       │
│                                  │                                  │
│                                  ▼                                  │
│                        ┌────────────────────┐                       │
│                        │  Span Processing   │                       │
│                        │  - Filter noise    │                       │
│                        │  - Normalize attrs │                       │
│                        │  - Calculate cost  │                       │
│                        │  - Session lookup  │                       │
│                        └─────────┬──────────┘                       │
│                                  │                                  │
│                                  ▼                                  │
│         ┌────────────────────────┴────────────────────────┐        │
│         │                                                  │        │
│         ▼                                                  ▼        │
│  ┌─────────────┐                                   ┌─────────────┐  │
│  │   SQLite    │                                   │  WebSocket  │  │
│  │   Backend   │                                   │    Hub      │  │
│  └─────────────┘                                   └─────────────┘  │
│                                                           │        │
│                                                           ▼        │
│                                                    ┌─────────────┐  │
│                                                    │  Dashboard  │  │
│                                                    │  (React UI) │  │
│                                                    └─────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

### 2. Coordinator Integration Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                     COORDINATOR → OBSERVATORY                        │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   Coordinator Daemon                         │    │
│  │                                                             │    │
│  │  1. Task Created                                            │    │
│  │     │                                                       │    │
│  │     ├──► ObservatorySync.SyncTask()                        │    │
│  │     │    - Get/create Workspace (by path)                  │    │
│  │     │    - Create/update Task in Observatory               │    │
│  │     │                                                       │    │
│  │  2. Agent Assigned                                          │    │
│  │     │                                                       │    │
│  │     ├──► ObservatorySync.SyncAgentAssignment()             │    │
│  │     │    - Create AgentAssignment record                   │    │
│  │     │    - Returns assignment_id for span linking          │    │
│  │     │                                                       │    │
│  │  3. Agent Executes (Claude Code / Gemini CLI)              │    │
│  │     │                                                       │    │
│  │     ├──► Sets OTEL_RESOURCE_ATTRIBUTES:                    │    │
│  │     │    - ailang.task_id=<task-id>                        │    │
│  │     │    - ailang.assignment_id=<assignment-id>            │    │
│  │     │                                                       │    │
│  │  4. Agent spans arrive via OTLP                            │    │
│  │     │                                                       │    │
│  │     └──► Linked to Task hierarchy automatically            │    │
│  │                                                             │    │
│  │  5. Task Completes                                          │    │
│  │     │                                                       │    │
│  │     └──► ObservatorySync.CompleteTask()                    │    │
│  │          - Update status, aggregates                        │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
```

### 3. Eval Harness Integration

```
┌──────────────────────────────────────────────────────────────────────┐
│                     EVAL HARNESS → OBSERVATORY                       │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ailang eval-suite --models gpt5,claude-sonnet-4-5                  │
│         │                                                           │
│         ▼                                                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Eval Harness                              │    │
│  │                                                             │    │
│  │  For each benchmark:                                        │    │
│  │    ┌────────────────────────────────────────────────────┐   │    │
│  │    │  span: eval.suite                                  │   │    │
│  │    │    ├── span: eval.benchmark (fizzbuzz/python)     │   │    │
│  │    │    │     ├── span: ai.generate (prompt→model)     │   │    │
│  │    │    │     │     - tokens_in, tokens_out, cost      │   │    │
│  │    │    │     ├── span: eval.compile (check syntax)    │   │    │
│  │    │    │     └── span: eval.execute (run tests)       │   │    │
│  │    │    ├── span: eval.benchmark (quicksort/ailang)    │   │    │
│  │    │    └── ...                                         │   │    │
│  │    └────────────────────────────────────────────────────┘   │    │
│  │                                                             │    │
│  │  All spans include:                                         │    │
│  │    - service.name: "ailang-eval"                           │    │
│  │    - ailang.eval.benchmark: <benchmark-id>                 │    │
│  │    - ailang.eval.model: <model-name>                       │    │
│  │    - ailang.eval.language: <target-language>               │    │
│  └─────────────────────────────────────────────────────────────┘    │
│         │                                                           │
│         ▼                                                           │
│  Observatory stores spans with eval metadata                        │
│  → Enables eval-specific filtering in control plane                 │
│  → Dashboard shows eval benchmarks separate from coordinator tasks  │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Integration Points

### 1. Telemetry Package (`internal/telemetry/`)

**Purpose**: OTEL SDK initialization, exporter configuration

**Key functions:**
```go
// Initialize global tracer with OTLP exporter
telemetry.Init(ctx, telemetry.Config{
    ServiceName:  "ailang-coordinator",
    OTLPEndpoint: "localhost:1957",  // Observatory OTLP receiver
})

// Create spans
ctx, span := telemetry.StartSpan(ctx, "operation.name")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("task.id", taskID),
    attribute.Int64("ai.tokens_in", tokensIn),
)
```

### 2. Server Integration (`internal/server/`)

**HTTP endpoints registered:**
```go
// Observatory REST API
mux.HandleFunc("GET /api/observatory/traces", s.handleListTraces)
mux.HandleFunc("GET /api/observatory/traces/{traceID}", s.handleGetTrace)
mux.HandleFunc("GET /api/observatory/spans", s.handleListSpans)
mux.HandleFunc("GET /api/observatory/metrics", s.handleGetMetrics)
mux.HandleFunc("GET /api/observatory/hierarchy", s.handleGetHierarchy)

// Control plane (dashboard visualizations)
mux.HandleFunc("GET /api/controlplane/heatmap", s.handleControlPlaneHeatmap)
mux.HandleFunc("GET /api/controlplane/topology", s.handleControlPlaneTopology)
mux.HandleFunc("GET /api/controlplane/breakdown", s.handleControlPlaneBreakdown)

// OTLP receiver
otlpReceiver.RegisterRoutes(mux)  // /v1/traces, /v1/logs, /v1/metrics

// WebSocket
mux.HandleFunc("/ws", hub.HandleWebSocket)
```

### 3. Coordinator Sync (`internal/coordinator/observatory_sync.go`)

**ObservatorySync struct:**
```go
type ObservatorySync struct {
    backend        observatory.Backend
    logger         *log.Logger
    workspaceCache map[string]string  // path → workspace ID
}

// Methods
func (s *ObservatorySync) SyncTask(ctx, task) error
func (s *ObservatorySync) SyncAgentAssignment(ctx, taskID, agentID, provider) (assignmentID, error)
func (s *ObservatorySync) CompleteAgentAssignment(ctx, assignmentID, stats) error
func (s *ObservatorySync) getOrCreateWorkspace(ctx, path) (workspaceID, error)
```

### 4. CLI Commands (`cmd/ailang/`)

```bash
# Start server with Observatory
ailang serve --port 1957

# Trace debugging
ailang trace list --hours 24 --limit 50
ailang trace view <trace-id>
ailang trace status
```

---

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP exporter endpoint | `http://localhost:1957` |
| `OTEL_RESOURCE_ATTRIBUTES` | Resource attributes (key=value,key=value) | - |
| `AILANG_OBSERVATORY_DB` | Observatory database path | `~/.ailang/state/observatory.db` |

### Database Locations

| Database | Purpose | Path |
|----------|---------|------|
| Observatory | Spans, tasks, workspaces | `~/.ailang/state/observatory.db` |
| Coordinator | Task queue, approvals | `~/.ailang/state/coordinator.db` |
| Collaboration | Messages | `~/.ailang/state/collaboration.db` |

### Pricing Configuration

Model pricing loaded from `internal/eval_harness/models.yml`:
```yaml
models:
  gpt5:
    provider: openai
    model_id: gpt-5
    pricing:
      input_per_1m: 2.50
      output_per_1m: 10.00

  claude-sonnet-4-5:
    provider: anthropic
    model_id: claude-sonnet-4-5-20241022
    pricing:
      input_per_1m: 3.00
      output_per_1m: 15.00
```

---

## Testing Strategy

### Test Files (11 total, 5,782 lines)

| File | Lines | Coverage |
|------|-------|----------|
| `store_test.go` | 739 | Store CRUD operations |
| `backend_test.go` | 392 | SQLite backend delegation |
| `api_test.go` | 789 | REST API handlers |
| `integration_test.go` | 1,053 | End-to-end scenarios |
| `otlp_receiver_test.go` | 725 | Span ingestion, session correlation |
| `websocket_test.go` | 190 | Hub broadcast, subscription filtering |
| `hierarchy_test.go` | 637 | Task hierarchy queries |
| `normalize_test.go` | 469 | Provider attribute normalization |
| `pricing_test.go` | 100 | Cost calculation |
| `aggregations_test.go` | 265 | Metrics summary |
| `migrate_test.go` | 423 | Schema migrations |

### Key Test Scenarios

1. **CRUD Operations**: All entity types (workspace, task, assignment, span, message)
2. **OTLP Ingestion**: Protobuf/JSON parsing, attribute extraction
3. **Session Correlation**: Orphaned span linking, retroactive hierarchy
4. **Cost Calculation**: Token-based pricing from models.yml
5. **Control Plane**: Heatmap, topology, breakdown queries

### Running Tests

```bash
# All observatory tests
go test ./internal/observatory/... -v

# With coverage
go test ./internal/observatory/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Specific test
go test ./internal/observatory/... -run TestSessionCorrelation -v
```

---

## Known Limitations

### Current Limitations

1. **GCP Trace Backend**: Disabled by default (requires GCP project configuration)
2. **Metrics Storage**: OTLP metrics endpoint accepts but doesn't store metrics
3. **Composite Backend**: Read operations query all backends sequentially (no parallel fan-out)
4. **Cost Calculation**: Returns $0.00 for models not in models.yml (no fallback)

### Future Work

1. **Parallel Backend Queries**: Fan-out reads across composite backends
2. **Metrics Storage**: Store and aggregate OTLP metrics
3. **Retention Policies**: Automatic span/trace cleanup after configurable period
4. **Alert Rules**: Trigger notifications on cost thresholds, error rates
5. **Query Language**: Support structured queries beyond REST filters

---

## Critical Files Reference

### Core Implementation
- [backend.go](internal/observatory/backend.go) - Backend interface (71 lines)
- [backend_sqlite.go](internal/observatory/backend_sqlite.go) - SQLite implementation (295 lines)
- [backend_controlplane.go](internal/observatory/backend_controlplane.go) - Control plane queries (552 lines)
- [store.go](internal/observatory/store.go) - Store struct + workspace ops (113 lines)
- [store_spans.go](internal/observatory/store_spans.go) - Span CRUD + traces (442 lines)
- [models.go](internal/observatory/models.go) - Entity definitions (411 lines)
- [otlp_receiver.go](internal/observatory/otlp_receiver.go) - OTEL ingestion (759 lines)
- [websocket.go](internal/observatory/websocket.go) - Real-time hub (409 lines)

### Integration Points
- [internal/server/handlers_controlplane.go](internal/server/handlers_controlplane.go) - REST handlers
- [internal/coordinator/observatory_sync.go](internal/coordinator/observatory_sync.go) - Coordinator sync
- [internal/telemetry/](internal/telemetry/) - OTEL SDK setup

### Configuration
- [internal/eval_harness/models.yml](internal/eval_harness/models.yml) - Model pricing
- [internal/observatory/migrate.go](internal/observatory/migrate.go) - Schema migrations

---

## Appendix: Span Attribute Conventions

### Input Attributes (Normalized)

| Attribute | Convention | Description |
|-----------|------------|-------------|
| `gen_ai.usage.input_tokens` | OpenTelemetry | Input token count |
| `gen_ai.usage.output_tokens` | OpenTelemetry | Output token count |
| `gen_ai.usage.cost` | OpenTelemetry | Cost in USD |
| `gen_ai.request.model` | OpenTelemetry | Model identifier |
| `gen_ai.system` | OpenTelemetry | Provider name |
| `ailang.tokens.input` | AILANG | Input token count |
| `ailang.tokens.output` | AILANG | Output token count |
| `ailang.cost.usd` | AILANG | Cost in USD |
| `ailang.model` | AILANG | Model identifier |
| `ailang.provider` | AILANG | Provider name |
| `task.id` | Coordinator | Task ID reference |
| `task.workspace` | Coordinator | Workspace path |
| `session.id` | Claude Code | Session identifier |

### Output Attributes (Stored)

| Field | Source Priority | Description |
|-------|-----------------|-------------|
| `tokens_in` | gen_ai > ailang > ai > task | Normalized input tokens |
| `tokens_out` | gen_ai > ailang > ai > task | Normalized output tokens |
| `cost_usd` | gen_ai > ailang > ai > task > calculated | Cost (or calculated from tokens) |
| `model` | gen_ai > ailang > ai | Model identifier |
| `provider` | ailang > gen_ai | Provider enum |
| `task_id` | resource > span > path > session | Hierarchy link |
| `agent_assignment_id` | resource > session | Hierarchy link |

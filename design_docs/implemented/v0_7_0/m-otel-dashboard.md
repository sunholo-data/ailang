# M-OTEL-DASHBOARD Implementation Report

**Status**: Implemented
**Version**: v0.6.4
**Implemented**: 2026-01-05
**Design Doc**: [../planned/v0_6_4/m-otel-dashboard.md](../../planned/v0_6_4/m-otel-dashboard.md)

## Executive Summary

The AILANG Observatory is a unified observability platform providing real-time visibility into AI operations from Claude Code, Gemini CLI, and AILANG itself. It includes:

- **OTLP Receiver** - Ingests telemetry via standard OpenTelemetry protocol
- **SQLite Storage** - Local persistence with full CRUD operations
- **REST API** - 30+ endpoints for querying traces, spans, metrics
- **WebSocket Hub** - Real-time updates with subscription filtering
- **React Dashboard** - Interactive UI with trace expansion and span details
- **Backend Adapters** - Pluggable architecture (SQLite complete, GCP/Jaeger stubs)

## Implementation Metrics

| Metric | Value |
|--------|-------|
| Total LOC (observatory package) | 7,456 |
| Test count | 50+ |
| Test coverage | All tests passing |
| API endpoints | 30+ |
| React hooks | 8 |

## Files Implemented

### Backend (`internal/observatory/`)

| File | LOC | Description |
|------|-----|-------------|
| `models.go` | ~390 | All Go types (Workspace, Task, Span, etc.) |
| `store.go` | ~1000 | Full SQLite CRUD with aggregations |
| `migrate.go` | ~100 | Schema migration runner |
| `backend.go` | ~260 | Backend interface + SQLiteBackend |
| `backend_gcp.go` | ~210 | GCP Trace backend (stub) |
| `backend_jaeger.go` | ~210 | Jaeger backend (stub) |
| `backend_composite.go` | ~300 | Write-local, read-remote routing |
| `api.go` | ~700 | Full REST API (30+ endpoints) |
| `websocket.go` | ~410 | WebSocket hub with subscriptions |
| `normalize.go` | ~400 | Claude/Gemini span normalization |
| `otlp_receiver.go` | ~500 | OTLP HTTP endpoints (traces, logs, metrics) |
| Various `*_test.go` | ~2000+ | Comprehensive test coverage |

### Frontend (`ui/src/`)

| File | LOC | Description |
|------|-----|-------------|
| `hooks/useObservatory.ts` | ~450 | All React data hooks |
| `features/observatory/Observatory.tsx` | ~400 | Main dashboard component |
| `features/observatory/Observatory.module.css` | ~300 | Styling |

## Milestone Completion Status

### M1: Schema & Core Models ✅ COMPLETE
- SQLite schema with all tables (workspaces, tasks, agents, spans, events, messages)
- Go types in `models.go` mapping 1:1 to schema
- Migration runner in `migrate.go`

### M2: SQLite Store ✅ COMPLETE
- Full CRUD for all entities in `store.go`
- Transaction support for atomic operations
- Aggregation queries (GetMetricsSummary, GetProviderComparison)

### M3: Provider Normalization ✅ COMPLETE
- `NormalizeClaudeMetrics()` - Claude logs → spans
- `NormalizeGeminiSpan()` - OTEL spans → normalized
- Common metrics extraction (tokens, cost, duration)

### M4: Backend Interface ✅ COMPLETE
- `Backend` interface with 35+ methods
- `SQLiteBackend` implementation
- Trace tree building from flat spans

### M5: GCP Trace Backend ⚠️ STUB
- Interface implemented, compiles
- Actual API calls are TODOs
- Returns `errNotSupported` for writes, specific errors for reads
- **Deferred**: Can be implemented when cloud viewing is needed

### M6: Jaeger Backend ⚠️ STUB
- Interface implemented, compiles
- Actual API calls are TODOs
- Same pattern as GCP backend
- **Deferred**: Can be implemented when Jaeger integration is needed

### M7: Composite Backend ✅ COMPLETE
- Full implementation in `backend_composite.go`
- Routes writes to local SQLite
- Routes reads to configured backend
- Merges results from multiple backends

### M8: REST API ✅ COMPLETE
- 30+ endpoints in `api.go`
- Workspaces: CRUD + stats
- Tasks: CRUD + timeline
- Agents: list + metrics
- Spans/Traces: query with filters
- Metrics: summary, providers, timeline
- Ingest: Claude metrics, OTEL spans

### M9: WebSocket Hub ✅ COMPLETE
- Connection management in `websocket.go`
- Broadcast on data changes
- Subscription filtering (workspace, task, agent)
- Event types: span.created, task.completed, metrics.updated

### M10: CLI Command Rename ✅ COMPLETE
- `ailang server` is the primary command
- `ailang serve` works as alias (backwards compatibility)

### M11: Server Integration ✅ COMPLETE
- Observatory routes registered in server
- REST API at `/api/observatory/*`
- WebSocket at `/ws/observatory`
- OTLP endpoints at `/v1/traces`, `/v1/logs`, `/v1/metrics`

### M12: Frontend Foundation ✅ COMPLETE
All React hooks implemented:
- `useWorkspaces()` - Workspace listing
- `useTasks(options)` - Task queries with filtering
- `useSpans(options)` - Span queries
- `useTraces(options)` - Trace summaries
- `useTrace(traceId)` - Single trace with full span tree
- `useMetrics()` - Metrics summary
- `useTelemetryConfig()` - Telemetry configuration
- `useObservatoryWs(options)` - Real-time WebSocket updates

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Observatory                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐  │
│  │  REST API     │   │  WebSocket    │   │  OTLP         │  │
│  │  (api.go)     │   │  (ws.go)      │   │  (otlp.go)    │  │
│  └───────┬───────┘   └───────┬───────┘   └───────┬───────┘  │
│          │                   │                   │          │
│          └───────────────────┼───────────────────┘          │
│                              │                               │
│                    ┌─────────▼─────────┐                    │
│                    │  Composite Backend │                    │
│                    └─────────┬─────────┘                    │
│                              │                               │
│         ┌────────────────────┼────────────────────┐         │
│         │                    │                    │         │
│  ┌──────▼──────┐     ┌──────▼──────┐     ┌──────▼──────┐   │
│  │   SQLite    │     │  GCP Trace  │     │   Jaeger    │   │
│  │  (store.go) │     │  (stub)     │     │   (stub)    │   │
│  └─────────────┘     └─────────────┘     └─────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

### Claude Code (`~/.claude/settings.json`)
```json
{
  "env": {
    "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
    "OTEL_LOGS_EXPORTER": "otlp",
    "OTEL_METRICS_EXPORTER": "otlp",
    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/json",
    "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:1957",
    "OTEL_RESOURCE_ATTRIBUTES": "ailang.source=user"
  }
}
```

### Gemini CLI (`~/.gemini/settings.json`)

**Architecture Decision**: Gemini CLI uses GCP Cloud Trace for telemetry (not direct OTLP export).

The Gemini CLI only supports two telemetry targets:
- `"local"` - File-based output only (no OTLP export)
- `"gcp"` - Direct export to GCP Cloud Trace

For unified observability, we use **GCP mode** and pull traces back via the GCP Trace API:

```json
{
  "telemetry": {
    "enabled": true,
    "target": "gcp",
    "logPrompts": true
  }
}
```

**Trace Linking Strategy**:
1. **Coordinator adds correlation attributes** when spawning Gemini tasks:
   - `ailang.task_id` - Links to coordinator task
   - `ailang.workspace` - Links to workspace
   - `ailang.session_id` - Links to parent session
2. **GCP backend queries** traces with these labels
3. **Observatory merges** local AILANG traces with pulled Gemini traces

**Why GCP over local file bridge**:
- Native support in Gemini CLI (no custom code needed)
- GCP Cloud Trace has rich query API for filtering
- Session IDs enable cross-tool trace correlation
- Aligns with production use case (cloud observability)

## Known Limitations

1. **GCP/Jaeger backends are stubs** - Only local SQLite is functional
2. **Claude Code sends events, not traces** - Synthetic spans created from logs
3. **No full trace hierarchy from Claude Code** - Only Gemini CLI provides span trees

## Future Work (Post v0.6.4)

- [ ] Implement GCP Trace API client for cloud viewing
- [ ] Implement Jaeger API client for local Jaeger integration
- [ ] Rich visualization components (trace explorer, agent activity)
- [ ] Provider comparison dashboard
- [ ] Alerting and notifications
- [ ] Advanced filtering and search

## Success Criteria Met

- [x] SQLite schema with 6 tables and indexes
- [x] 7,456 LOC backend code (exceeded 2,700 target)
- [x] 50+ tests passing
- [x] Backend adapter interface with 3 implementations
- [x] 30+ REST API endpoints
- [x] WebSocket real-time updates
- [x] `ailang server` command works
- [x] Frontend loads and displays data
- [x] `make test` passes
- [x] Claude Code telemetry flows to Observatory
- [x] Traces appear in UI with correct data
- [x] Span details show tokens, cost, attributes

---

**Implementation Date**: 2026-01-04 to 2026-01-05
**Total Development Time**: ~8 hours across 2 sessions

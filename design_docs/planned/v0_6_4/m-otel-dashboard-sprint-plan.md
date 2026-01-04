# M-OTEL-DASHBOARD Sprint Plan

**Sprint ID**: M-OTEL-DASHBOARD
**Design Doc**: [m-otel-dashboard.md](m-otel-dashboard.md)
**Target Version**: v0.6.4
**Duration**: 8 days (64 hours)
**Risk Level**: Medium

## Executive Summary

Build the AILANG Observatory - a new unified observability platform from scratch with OTEL-standard data foundations. This replaces the existing `ailang serve` command (renamed to `ailang server`) with a portable, pluggable dashboard.

**Key Deliverables:**
- New `internal/observatory/` package with SQLite-backed data layer
- Normalized span model supporting Claude + Gemini telemetry
- Pluggable backend adapters (SQLite, GCP Trace, Jaeger)
- REST API + WebSocket for real-time updates
- React frontend foundation with data hooks

## Velocity Analysis

**Recent completed sprints:**
- M-OTEL-CROSS-PROCESS: 365 LOC in 1 day (estimated 410 LOC, 2 days) → 1.8x faster
- M-OTEL-ENHANCED-TRACING-DX: ~500 LOC in 1 day

**Estimated velocity:** 180-250 LOC/day for new features with tests

**Sprint estimate:** 2,700 LOC backend → 12-15 working days at conservative pace

**Adjusted timeline:** 8 days aggressive, 12 days conservative

## Milestones

### M1: Schema & Core Models (~300 LOC, Day 1)

**Goal:** SQLite schema and Go type definitions

**Tasks:**
- [ ] Create `internal/observatory/` package structure
- [ ] Define SQLite schema in `schema.sql` (workspaces, tasks, agents, spans, events, messages)
- [ ] Create Go models in `models.go` (matching schema)
- [ ] Add migration runner
- [ ] Unit tests for schema creation

**Files:**
- `internal/observatory/schema.sql` (~150 LOC)
- `internal/observatory/models.go` (~150 LOC)

**Acceptance Criteria:**
- Schema creates all 6 tables with indexes and views
- Go types map 1:1 to schema
- `make test` passes

**Dependencies:** None

---

### M2: SQLite Store (~400 LOC, Days 1-2)

**Goal:** CRUD operations for all entities

**Tasks:**
- [ ] Create `store.go` with Store interface
- [ ] Implement workspace CRUD
- [ ] Implement task CRUD with aggregation updates
- [ ] Implement agent_assignment CRUD
- [ ] Implement span CRUD
- [ ] Implement span_event CRUD
- [ ] Implement message CRUD
- [ ] Transaction support for atomic operations
- [ ] Unit tests for all operations

**Files:**
- `internal/observatory/store.go` (~400 LOC)
- `internal/observatory/store_test.go` (~300 LOC)

**Acceptance Criteria:**
- All CRUD operations work
- Aggregations update on span insert
- Tests cover happy path + error cases
- `make test` passes

**Dependencies:** M1

---

### M3: Provider Normalization (~200 LOC, Day 2)

**Goal:** Convert Claude metrics and Gemini spans to NormalizedSpan

**Tasks:**
- [ ] Define NormalizedSpan type with all fields
- [ ] Implement `NormalizeClaudeMetrics()` - metrics → span
- [ ] Implement `NormalizeGeminiSpan()` - OTEL span → normalized
- [ ] Implement `NormalizeAILANGSpan()` - our spans → normalized
- [ ] Extract common metrics (tokens, cost, duration)
- [ ] Unit tests with sample data from both providers

**Files:**
- `internal/observatory/normalize.go` (~200 LOC)
- `internal/observatory/normalize_test.go` (~150 LOC)

**Acceptance Criteria:**
- Claude metrics convert to valid NormalizedSpan
- Gemini spans convert with parent-child preserved
- Common fields (tokens, cost) extracted correctly
- Tests cover both providers

**Dependencies:** M1

---

### M4: Backend Interface & SQLite Backend (~400 LOC, Days 2-3)

**Goal:** Pluggable backend architecture with SQLite implementation

**Tasks:**
- [ ] Define `Backend` interface with query methods
- [ ] Implement `SQLiteBackend` using store
- [ ] Add trace tree building (spans → hierarchy)
- [ ] Implement aggregation queries (workspace, task, agent stats)
- [ ] Add provider comparison query
- [ ] Integration tests

**Files:**
- `internal/observatory/backend.go` (~100 LOC)
- `internal/observatory/backend_sqlite.go` (~300 LOC)
- `internal/observatory/backend_sqlite_test.go` (~200 LOC)

**Acceptance Criteria:**
- Backend interface covers all query needs
- SQLiteBackend implements full interface
- Trace trees build correctly from flat spans
- Aggregations match expected values
- `make test` passes

**Dependencies:** M2, M3

---

### M5: GCP Trace Backend (~200 LOC, Day 3)

**Goal:** Query traces from Google Cloud Trace

**Tasks:**
- [ ] Implement `GCPBackend` struct
- [ ] Add GCP Trace API client setup
- [ ] Implement `GetTrace()` - fetch by trace_id
- [ ] Implement `ListTraces()` - query with filters
- [ ] Map GCP spans to NormalizedSpan
- [ ] Handle pagination
- [ ] Integration test (requires GOOGLE_CLOUD_PROJECT)

**Files:**
- `internal/observatory/backend_gcp.go` (~200 LOC)
- `internal/observatory/backend_gcp_test.go` (~100 LOC)

**Acceptance Criteria:**
- GCPBackend implements Backend interface
- Traces fetch correctly from GCP
- Spans normalize to our model
- Graceful degradation when GCP unavailable

**Dependencies:** M4

---

### M6: Jaeger Backend (~150 LOC, Day 4)

**Goal:** Query traces from Jaeger API

**Tasks:**
- [ ] Implement `JaegerBackend` struct
- [ ] Add Jaeger API client (HTTP)
- [ ] Implement `GetTrace()` - fetch by trace_id
- [ ] Implement `ListTraces()` - query with filters
- [ ] Map Jaeger spans to NormalizedSpan

**Files:**
- `internal/observatory/backend_jaeger.go` (~150 LOC)
- `internal/observatory/backend_jaeger_test.go` (~80 LOC)

**Acceptance Criteria:**
- JaegerBackend implements Backend interface
- Works with standard Jaeger Query API
- Spans normalize correctly

**Dependencies:** M4

---

### M7: Composite Backend (~100 LOC, Day 4)

**Goal:** Write local, read from local or remote

**Tasks:**
- [ ] Implement `CompositeBackend` struct
- [ ] Route writes to SQLite
- [ ] Route reads to configured backend (SQLite/GCP/Jaeger)
- [ ] Add backend factory with configuration
- [ ] Configuration via `~/.ailang/config.yaml`

**Files:**
- `internal/observatory/backend_composite.go` (~100 LOC)
- `internal/observatory/backend_factory.go` (~80 LOC)

**Acceptance Criteria:**
- Writes always go to local SQLite
- Reads route based on configuration
- Factory creates correct backend from config

**Dependencies:** M4, M5, M6

---

### M8: REST API Handlers (~500 LOC, Days 4-5)

**Goal:** Full REST API for all entities

**Tasks:**
- [ ] Create `handlers.go` with handler functions
- [ ] Implement workspace endpoints (CRUD + stats)
- [ ] Implement task endpoints (CRUD + timeline)
- [ ] Implement agent endpoints (list + metrics)
- [ ] Implement trace/span endpoints (tree + details)
- [ ] Implement metrics endpoints (summary, providers, timeline)
- [ ] Implement message endpoints (CRUD + search)
- [ ] Add query filters and pagination
- [ ] OpenAPI spec comments

**Files:**
- `internal/observatory/handlers.go` (~500 LOC)
- `internal/observatory/handlers_test.go` (~300 LOC)

**Acceptance Criteria:**
- All 20+ endpoints implemented
- Proper HTTP status codes
- JSON request/response
- Query filters work (status, time range, etc.)
- Pagination for list endpoints
- `make test` passes

**Dependencies:** M4

---

### M9: WebSocket Hub (~200 LOC, Day 5)

**Goal:** Real-time updates via WebSocket

**Tasks:**
- [ ] Create WebSocket hub with connection management
- [ ] Implement broadcast on data changes
- [ ] Add subscription filtering (workspace, task, agent)
- [ ] Handle reconnection
- [ ] Integration with store (trigger on insert/update)

**Files:**
- `internal/observatory/websocket.go` (~200 LOC)
- `internal/observatory/websocket_test.go` (~100 LOC)

**Acceptance Criteria:**
- WebSocket connections establish and maintain
- Events broadcast within 100ms of data change
- Clients can filter by entity type
- Graceful disconnect handling

**Dependencies:** M8

---

### M10: CLI Command Rename (~50 LOC, Day 5)

**Goal:** Rename `ailang serve` to `ailang server`

**Tasks:**
- [ ] Rename `cmd/ailang/serve.go` → `cmd/ailang/server.go`
- [ ] Update command name in cobra
- [ ] Add `serve` as alias for backwards compatibility
- [ ] Update all documentation references
- [ ] Update CLAUDE.md references

**Files:**
- `cmd/ailang/server.go` (renamed)
- `docs/docs/guides/*.md` (updates)
- `CLAUDE.md` (updates)

**Acceptance Criteria:**
- `ailang server` works
- `ailang serve` still works (alias)
- All docs updated
- `make test` passes

**Dependencies:** None (can run in parallel)

---

### M11: Server Integration (~150 LOC, Day 6)

**Goal:** Integrate observatory into ailang server

**Tasks:**
- [ ] Initialize observatory store on server start
- [ ] Register REST API routes
- [ ] Register WebSocket endpoint
- [ ] Add OTEL span receiver endpoint
- [ ] Configure backend from `config.yaml`
- [ ] Migrate existing endpoints (if keeping any)

**Files:**
- `cmd/ailang/server.go` (updates, ~50 LOC)
- `internal/server/routes.go` (updates, ~50 LOC)
- `internal/server/observatory.go` (new, ~50 LOC)

**Acceptance Criteria:**
- `ailang server` starts with observatory enabled
- REST API accessible at `/api/*`
- WebSocket accessible at `/ws`
- Spans can be ingested via POST `/api/spans`

**Dependencies:** M8, M9, M10

---

### M12: Frontend Foundation (~600 LOC, Days 6-8)

**Goal:** React app structure with data hooks

**Tasks:**
- [ ] Create new React app structure in `ui/`
- [ ] Implement `useWorkspaces` hook
- [ ] Implement `useTasks` hook
- [ ] Implement `useAgents` hook
- [ ] Implement `useSpans` hook
- [ ] Implement `useWebSocket` hook
- [ ] Create base component library
- [ ] Set up routing (React Router)
- [ ] Basic layout with navigation

**Files:**
- `ui/src/hooks/useWorkspaces.ts` (~80 LOC)
- `ui/src/hooks/useTasks.ts` (~80 LOC)
- `ui/src/hooks/useAgents.ts` (~80 LOC)
- `ui/src/hooks/useSpans.ts` (~80 LOC)
- `ui/src/hooks/useWebSocket.ts` (~100 LOC)
- `ui/src/App.tsx` (updates, ~80 LOC)
- `ui/src/components/Layout.tsx` (~100 LOC)

**Acceptance Criteria:**
- Hooks fetch data from REST API
- WebSocket updates trigger re-renders
- Basic navigation works
- App loads without errors

**Dependencies:** M11

---

## Timeline

| Day | Milestones | Focus |
|-----|------------|-------|
| 1 | M1, M2 (start) | Schema, models, store foundations |
| 2 | M2 (complete), M3, M4 (start) | Store tests, normalization, backend interface |
| 3 | M4 (complete), M5 | SQLite backend, GCP backend |
| 4 | M6, M7, M8 (start) | Jaeger backend, composite, REST API |
| 5 | M8 (complete), M9, M10 | REST API tests, WebSocket, CLI rename |
| 6 | M11, M12 (start) | Server integration, frontend hooks |
| 7 | M12 (continue) | Frontend components |
| 8 | M12 (complete), testing | Frontend polish, integration testing |

## Success Metrics

- [ ] All 6 SQLite tables created with indexes
- [ ] 2,700+ LOC backend code
- [ ] 1,000+ LOC tests
- [ ] Backend adapter interface with 3 implementations
- [ ] 20+ REST API endpoints
- [ ] WebSocket real-time updates < 100ms
- [ ] `ailang server` command works
- [ ] Frontend loads and displays data
- [ ] `make test` passes
- [ ] `make lint` passes

## Risk Factors

| Risk | Impact | Mitigation |
|------|--------|------------|
| GCP Trace API complexity | Medium | Start with SQLite, add GCP later |
| Frontend scope creep | High | Focus on hooks, minimal UI |
| Integration with existing code | Medium | New package, minimal coupling |
| WebSocket reliability | Low | Well-understood patterns |

## Open Questions

1. **Keep any existing Collaboration Hub features?** - Assuming full replacement
2. **Database location?** - Propose `~/.ailang/state/observatory.db`
3. **Config format?** - Propose extending `~/.ailang/config.yaml`

## Post-Sprint

After this sprint, follow-up work includes:
- Rich visualization components (trace explorer, agent activity)
- Provider comparison dashboard
- Alerting and notifications
- Advanced filtering and search

---

**Created**: 2026-01-04
**Last Updated**: 2026-01-04

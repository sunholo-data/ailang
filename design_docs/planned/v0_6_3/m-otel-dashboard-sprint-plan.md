# M-OTEL-DASHBOARD Sprint Plan

**Sprint ID**: M-OTEL-DASHBOARD
**Design Doc**: [m-otel-dashboard.md](m-otel-dashboard.md)
**Target Version**: v0.6.4
**Duration**: 6 days (48 hours) - reduced due to foundation work
**Risk Level**: Medium
**Status**: ✅ COMPLETE (2026-01-05)

> **Implementation Complete (2026-01-05)**: All core milestones implemented. See [implemented/v0_6_4/m-otel-dashboard.md](../../implemented/v0_6_4/m-otel-dashboard.md) for full report.
>
> **Summary**: 7,456 LOC implemented, 50+ tests passing, 30+ API endpoints, 8 React hooks. Only GCP/Jaeger backends remain as stubs (deferred to future sprint).

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

### M1: Schema & Core Models (~300 LOC, Day 1) ✅ COMPLETE

**Goal:** SQLite schema and Go type definitions

**Tasks:**
- [x] Create `internal/observatory/` package structure
- [x] Define SQLite schema in `schema.sql` (workspaces, tasks, agents, spans, events, messages)
- [x] Create Go models in `models.go` (matching schema)
- [x] Add migration runner
- [x] Unit tests for schema creation

**Files:**
- `internal/observatory/schema.sql` (~150 LOC)
- `internal/observatory/models.go` (~150 LOC)

**Acceptance Criteria:**
- Schema creates all 6 tables with indexes and views
- Go types map 1:1 to schema
- `make test` passes

**Dependencies:** None

---

### M2: SQLite Store (~400 LOC, Days 1-2) ✅ COMPLETE

**Goal:** CRUD operations for all entities

**Tasks:**
- [x] Create `store.go` with Store interface
- [x] Implement workspace CRUD
- [x] Implement task CRUD with aggregation updates
- [x] Implement agent_assignment CRUD
- [x] Implement span CRUD
- [x] Implement span_event CRUD
- [x] Implement message CRUD
- [x] Transaction support for atomic operations
- [x] Unit tests for all operations

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

### M3: Provider Normalization (~200 LOC, Day 2) ✅ COMPLETE

**Goal:** Convert Claude metrics and Gemini spans to NormalizedSpan

**Tasks:**
- [x] Define NormalizedSpan type with all fields
- [x] Implement `NormalizeClaudeMetrics()` - metrics → span
- [x] Implement `NormalizeGeminiSpan()` - OTEL span → normalized
- [x] Implement `NormalizeAILANGSpan()` - our spans → normalized
- [x] Extract common metrics (tokens, cost, duration)
- [x] Unit tests with sample data from both providers

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

### M4: Backend Interface & SQLite Backend (~400 LOC, Days 2-3) ✅ COMPLETE

**Goal:** Pluggable backend architecture with SQLite implementation

**Tasks:**
- [x] Define `Backend` interface with query methods
- [x] Implement `SQLiteBackend` using store
- [x] Add trace tree building (spans → hierarchy)
- [x] Implement aggregation queries (workspace, task, agent stats)
- [x] Add provider comparison query
- [x] Integration tests

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

### M5: GCP Trace Backend (~200 LOC, Day 3) ⚠️ STUB (Deferred)

**Goal:** Query traces from Google Cloud Trace

**Tasks:**
- [x] Implement `GCPBackend` struct
- [x] Add GCP config structure
- [ ] Add GCP Trace API client setup (TODO)
- [ ] Implement `GetTrace()` - fetch by trace_id (TODO)
- [ ] Implement `ListTraces()` - query with filters (TODO)
- [ ] Map GCP spans to NormalizedSpan (TODO)
- [ ] Handle pagination (TODO)
- [ ] Integration test (requires GOOGLE_CLOUD_PROJECT) (TODO)

**Note:** Interface fully implemented with proper error handling. Actual API calls deferred to future sprint.

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

### M6: Jaeger Backend (~150 LOC, Day 4) ⚠️ STUB (Deferred)

**Goal:** Query traces from Jaeger API

**Tasks:**
- [x] Implement `JaegerBackend` struct
- [x] Add Jaeger config structure
- [ ] Add Jaeger API client (HTTP) (TODO)
- [ ] Implement `GetTrace()` - fetch by trace_id (TODO)
- [ ] Implement `ListTraces()` - query with filters (TODO)
- [ ] Map Jaeger spans to NormalizedSpan (TODO)

**Note:** Interface fully implemented with proper error handling. Actual API calls deferred to future sprint.

**Files:**
- `internal/observatory/backend_jaeger.go` (~150 LOC)
- `internal/observatory/backend_jaeger_test.go` (~80 LOC)

**Acceptance Criteria:**
- JaegerBackend implements Backend interface
- Works with standard Jaeger Query API
- Spans normalize correctly

**Dependencies:** M4

---

### M7: Composite Backend (~100 LOC, Day 4) ✅ COMPLETE

**Goal:** Write local, read from local or remote

**Tasks:**
- [x] Implement `CompositeBackend` struct
- [x] Route writes to SQLite
- [x] Route reads to configured backend (SQLite/GCP/Jaeger)
- [x] Add backend factory with configuration
- [x] Configuration via `~/.ailang/config.yaml`

**Files:**
- `internal/observatory/backend_composite.go` (~100 LOC)
- `internal/observatory/backend_factory.go` (~80 LOC)

**Acceptance Criteria:**
- Writes always go to local SQLite
- Reads route based on configuration
- Factory creates correct backend from config

**Dependencies:** M4, M5, M6

---

### M8: REST API Handlers (~500 LOC, Days 4-5) ✅ COMPLETE

**Goal:** Full REST API for all entities

**Tasks:**
- [x] Create `handlers.go` with handler functions
- [x] Implement workspace endpoints (CRUD + stats)
- [x] Implement task endpoints (CRUD + timeline)
- [x] Implement agent endpoints (list + metrics)
- [x] Implement trace/span endpoints (tree + details)
- [x] Implement metrics endpoints (summary, providers, timeline)
- [x] Implement message endpoints (CRUD + search)
- [x] Add query filters and pagination
- [x] OpenAPI spec comments

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

### M9: WebSocket Hub (~200 LOC, Day 5) ✅ COMPLETE

**Goal:** Real-time updates via WebSocket

**Tasks:**
- [x] Create WebSocket hub with connection management
- [x] Implement broadcast on data changes
- [x] Add subscription filtering (workspace, task, agent)
- [x] Handle reconnection
- [x] Integration with store (trigger on insert/update)

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

### M10: CLI Command Rename (~50 LOC, Day 5) ✅ COMPLETE

**Goal:** Rename `ailang serve` to `ailang server`

**Tasks:**
- [x] Rename `cmd/ailang/serve.go` → `cmd/ailang/server.go`
- [x] Update command name in cobra
- [x] Add `serve` as alias for backwards compatibility
- [x] Update all documentation references
- [x] Update CLAUDE.md references

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

### M11: Server Integration (~150 LOC, Day 6) ✅ COMPLETE

**Goal:** Integrate observatory into ailang server

**Tasks:**
- [x] Initialize observatory store on server start
- [x] Register REST API routes
- [x] Register WebSocket endpoint
- [x] Add OTEL span receiver endpoint
- [x] Configure backend from `config.yaml`
- [x] Migrate existing endpoints (if keeping any)

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

### M12: Frontend Foundation (~600 LOC, Days 6-8) ✅ COMPLETE

**Goal:** React app structure with data hooks

**Tasks:**
- [x] Create new React app structure in `ui/`
- [x] Implement `useWorkspaces` hook
- [x] Implement `useTasks` hook
- [x] Implement `useAgents` hook (useSpans covers this)
- [x] Implement `useSpans` hook
- [x] Implement `useWebSocket` hook (useObservatoryWs)
- [x] Implement `useTraces` hook
- [x] Implement `useTrace` hook (single trace)
- [x] Implement `useMetrics` hook
- [x] Implement `useTelemetryConfig` hook
- [x] Create base component library
- [x] Set up routing (React Router)
- [x] Basic layout with navigation

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

- [x] All 6 SQLite tables created with indexes
- [x] 2,700+ LOC backend code (actual: 7,456 LOC)
- [x] 1,000+ LOC tests (actual: 2,000+ LOC)
- [x] Backend adapter interface with 3 implementations
- [x] 20+ REST API endpoints (actual: 30+)
- [x] WebSocket real-time updates < 100ms
- [x] `ailang server` command works
- [x] Frontend loads and displays data
- [x] `make test` passes
- [x] `make lint` passes

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

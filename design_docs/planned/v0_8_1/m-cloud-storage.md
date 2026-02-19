# M-CLOUD-STORAGE: Cloud Database Migration

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Medium - enables cloud deployment)
**Estimated**: 3 weeks (60 hours)
**Dependencies**: M-TASK-HIERARCHY (v0.6.5) complete

## Executive Summary

Migrate AILANG's storage layer from SQLite to GCP cloud services, enabling multi-user deployment and horizontal scaling while preserving local SQLite for development.

**Strategy**: Split by access pattern
- **Coordinator** (OLTP) → Firestore
- **Observatory** (OLAP) → BigQuery
- **Collaboration** (Queue) → Firestore

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same data, same results regardless of backend |
| A2: Replayability | +1 | BigQuery enables large-scale trace replay |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | +1 | Cloud IAM enforces capability constraints |
| A5: Bounded Verification | 0 | No change to verification |
| A6: Safe Concurrency | +1 | Cloud backends handle concurrent access |
| A7: Machines First | +1 | Serverless = less infrastructure to manage |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | BigQuery shows query costs, Firestore shows ops |
| A10: Composability | 0 | Storage layer unchanged from caller perspective |
| A11: Structured Failure | +1 | Cloud errors are structured and typed |
| A12: System Boundary | +1 | Explicit boundary: local SQLite vs cloud |

**Net Score: +8** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Backend abstraction ensures deterministic behavior
- [x] A3 (Effects): Storage is already an effect, no hidden effects added
- [x] A4 (Authority): GCP IAM more explicit than filesystem permissions
- [x] A7 (Machines First): Serverless reduces operational complexity

## Problem Statement

AILANG currently uses SQLite for all storage, which works for local development but cannot scale to:
- Multi-user cloud deployment
- High-volume telemetry ingestion
- Analytics across large span datasets

**Current State:**

| Database | Size | Pattern | SQLite Limitation |
|----------|------|---------|-------------------|
| observatory.db | 32MB (26K spans) | OLAP analytics | Single-writer, slow aggregations |
| coordinator.db | 5.1MB | OLTP state machine | Single-writer bottleneck |
| collaboration.db | 3MB | Message queue | No real-time sync |

**Impact:**
- Cannot deploy to cloud without rewriting storage
- Analytics queries slow as span count grows
- No multi-user collaboration possible

## Goals

**Primary Goal:** Enable cloud deployment on GCP while maintaining local SQLite for development.

**Success Metrics:**
- All existing tests pass with both SQLite and cloud backends
- Local development workflow unchanged (`ailang serve` just works)
- Observatory queries <1s for 1M+ spans on BigQuery
- Zero-downtime deployment with Firestore (serverless)
- <$50/mo cloud cost at moderate usage (10K spans/day)

## Solution Design

### Overview

Create a **backend abstraction layer** that supports pluggable storage backends. Each database gets the optimal backend for its access pattern:

```
┌─────────────────────────────────────────────────────────────────┐
│                         AILANG Application                       │
├─────────────────────────────────────────────────────────────────┤
│                      Backend Interfaces                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ coordinator  │  │ observatory  │  │ collaboration │          │
│  │   .Store     │  │   .Backend   │  │    .Store     │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬────────┘          │
├─────────┼─────────────────┼─────────────────┼───────────────────┤
│         ▼                 ▼                 ▼                    │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Backend Selector                          ││
│  │  AILANG_STORAGE=local  → SQLite (all 3)                     ││
│  │  AILANG_STORAGE=gcp    → Firestore + BigQuery               ││
│  │  AILANG_STORAGE=hybrid → SQLite local, GCP for analytics    ││
│  └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│  Local (SQLite)          │  GCP Cloud                           │
│  ┌──────────────────┐    │  ┌──────────────────┐               │
│  │ coordinator.db   │    │  │ Firestore        │               │
│  │ observatory.db   │    │  │ (coordinator +   │               │
│  │ collaboration.db │    │  │  collaboration)  │               │
│  └──────────────────┘    │  ├──────────────────┤               │
│                          │  │ BigQuery         │               │
│                          │  │ (observatory)    │               │
│                          │  └──────────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### Architecture

**Backend Selection:**

```go
// Environment variable controls backend selection
// AILANG_STORAGE=local|gcp|hybrid

func NewStorageBackends(ctx context.Context) (*Backends, error) {
    mode := os.Getenv("AILANG_STORAGE")
    if mode == "" {
        mode = "local" // Default to SQLite for local dev
    }

    switch mode {
    case "local":
        return newSQLiteBackends()
    case "gcp":
        return newGCPBackends(ctx)
    case "hybrid":
        return newHybridBackends(ctx) // SQLite + BigQuery
    }
}
```

**Components:**

1. **Coordinator Backend** (Firestore)
   - Task state machine with optimistic locking
   - Real-time listeners for approval workflow
   - Scales to zero when idle

2. **Observatory Backend** (BigQuery)
   - Columnar storage for span analytics
   - Time-partitioned tables (by day)
   - Streaming inserts for real-time ingest
   - SQL-compatible queries

3. **Collaboration Backend** (Firestore)
   - Message documents with delivery state
   - Real-time sync via listeners
   - Inbox queries with composite indexes

### GCP Services Selection

| Database | GCP Service | Rationale |
|----------|-------------|-----------|
| Coordinator | **Firestore** | Serverless, real-time, scales to zero, ACID per-doc |
| Observatory | **BigQuery** | Analytics-native, petabyte scale, SQL, streaming insert |
| Collaboration | **Firestore** | Same as coordinator, simpler ops |

**Why not alternatives:**

| Alternative | Why Not |
|-------------|---------|
| Cloud SQL | Always-on cost ($30-50/mo minimum), not serverless |
| Bigtable | Overkill for current scale, complex schema design |
| Spanner | Expensive, overkill for single-region deployment |
| DuckDB on Cloud Run | No managed service, complex state management |
| AlloyDB | PostgreSQL compatibility not needed, expensive |

### Implementation Plan

**Phase 1: Backend Abstraction** (~16 hours)
- [ ] Define unified backend interfaces for all 3 stores
- [ ] Refactor existing SQLite backends to implement interfaces
- [ ] Add backend selector with env var configuration
- [ ] Unit tests for interface compliance
- [ ] Integration tests for SQLite path

**Phase 2: Firestore Backend** (~20 hours)
- [ ] Implement Firestore client wrapper
- [ ] Coordinator store: tasks, approvals, events
- [ ] Collaboration store: threads, messages
- [ ] Real-time listeners for WebSocket bridge
- [ ] IAM configuration and auth flow
- [ ] Integration tests with Firestore emulator

**Phase 3: BigQuery Backend** (~16 hours)
- [ ] Define BigQuery schema (time-partitioned)
- [ ] Implement streaming insert for spans
- [ ] Implement batch load for backfill
- [ ] Query translator for existing filter API
- [ ] Cost-aware query optimization
- [ ] Integration tests with BigQuery sandbox

**Phase 4: Hybrid Mode & Migration** (~8 hours)
- [ ] Implement hybrid backend (local + BigQuery)
- [ ] Migration tool: SQLite → GCP
- [ ] Rollback tool: GCP → SQLite export
- [ ] Documentation and deployment guide
- [ ] End-to-end tests

### Files to Modify/Create

**New files:**

| File | Purpose | LOC |
|------|---------|-----|
| `internal/storage/backend.go` | Unified backend selector | ~100 |
| `internal/storage/firestore/client.go` | Firestore client wrapper | ~200 |
| `internal/storage/firestore/coordinator.go` | Coordinator Firestore impl | ~400 |
| `internal/storage/firestore/collaboration.go` | Collaboration Firestore impl | ~300 |
| `internal/storage/bigquery/client.go` | BigQuery client wrapper | ~150 |
| `internal/storage/bigquery/observatory.go` | Observatory BigQuery impl | ~500 |
| `internal/storage/bigquery/schema.go` | Table schemas | ~100 |
| `internal/storage/migrate/sqlite_to_gcp.go` | Migration tool | ~300 |
| `cmd/ailang/storage.go` | CLI commands for migration | ~150 |

**Modified files:**

| File | Changes | LOC |
|------|---------|-----|
| `internal/coordinator/store.go` | Extract interface | ~50 |
| `internal/observatory/backend.go` | Already has interface | ~20 |
| `internal/messaging/store.go` | Extract interface | ~50 |
| `internal/server/server.go` | Use backend selector | ~30 |
| `cmd/ailang/server.go` | Add storage flag | ~20 |

**Total: ~2,370 LOC new, ~170 LOC modified**

## Database Schemas

### Firestore: Coordinator

```
/projects/{projectId}/databases/(default)/documents/
├── tasks/{taskId}
│   ├── id: string
│   ├── title: string
│   ├── status: string (pending|running|completed|failed)
│   ├── workspace: string
│   ├── created_at: timestamp
│   ├── cost: number
│   └── tokens_used: number
│
├── approvals/{approvalId}
│   ├── task_id: string (reference)
│   ├── type: string
│   ├── status: string (pending|approved|rejected)
│   └── created_at: timestamp
│
└── task_events/{taskId}/events/{eventId}
    ├── stream_type: string
    ├── turn_num: number
    ├── text: string
    └── timestamp: timestamp
```

### Firestore: Collaboration

```
/projects/{projectId}/databases/(default)/documents/
├── threads/{threadId}
│   ├── title: string
│   ├── status: string
│   └── created_at: timestamp
│
└── messages/{messageId}
    ├── thread_id: string
    ├── from_id: string
    ├── content: string
    ├── delivery_state: string
    └── created_at: timestamp
```

### BigQuery: Observatory

```sql
-- Time-partitioned by start_time (daily)
CREATE TABLE observatory.spans (
    id STRING NOT NULL,
    trace_id STRING NOT NULL,
    parent_span_id STRING,
    task_id STRING,
    agent_assignment_id STRING,

    name STRING NOT NULL,
    kind STRING NOT NULL,
    status STRING NOT NULL,
    status_message STRING,

    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_ms INT64,

    tokens_in INT64,
    tokens_out INT64,
    cost_usd FLOAT64,
    model STRING,
    provider STRING,

    attributes JSON,
    resource_attributes JSON,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
)
PARTITION BY DATE(start_time)
CLUSTER BY trace_id, task_id;

-- Materialized view for fast aggregations
CREATE MATERIALIZED VIEW observatory.daily_stats AS
SELECT
    DATE(start_time) as date,
    provider,
    model,
    COUNT(*) as span_count,
    SUM(tokens_in) as total_tokens_in,
    SUM(tokens_out) as total_tokens_out,
    SUM(cost_usd) as total_cost
FROM observatory.spans
GROUP BY 1, 2, 3;
```

## Examples

### Example 1: Local Development (Unchanged)

```bash
# Default behavior - uses SQLite
ailang serve
# Uses ~/.ailang/state/*.db files
```

### Example 2: Cloud Deployment

```bash
# Set GCP project and storage mode
export AILANG_CLOUD_PROJECT=my-ailang-project
export AILANG_STORAGE=gcp

# Start server (uses Firestore + BigQuery)
ailang serve --port 8080
```

### Example 3: Hybrid Mode (Local + Cloud Analytics)

```bash
# Keep local SQLite for coordinator, use BigQuery for analytics
export AILANG_STORAGE=hybrid
export AILANG_CLOUD_PROJECT=my-ailang-project

ailang serve
# Coordinator: local SQLite (fast, no network latency)
# Observatory: BigQuery (unlimited analytics scale)
```

### Example 4: Migration

```bash
# Export local data to GCP
ailang storage migrate --from local --to gcp

# Verify data
ailang storage verify --source local --target gcp

# Rollback if needed
ailang storage export --from gcp --to backup.sql
```

## Cost Estimates

### GCP Pricing (as of 2026)

| Service | Unit | Price | Estimate |
|---------|------|-------|----------|
| Firestore reads | per 100K | $0.06 | ~$1/mo |
| Firestore writes | per 100K | $0.18 | ~$2/mo |
| Firestore storage | per GB | $0.18 | ~$1/mo |
| BigQuery storage | per GB | $0.02 | ~$1/mo |
| BigQuery queries | per TB | $5.00 | ~$5/mo |
| BigQuery streaming | per 200MB | $0.01 | ~$2/mo |

**Estimated monthly cost at moderate usage (10K spans/day):**
- Firestore: ~$5/mo
- BigQuery: ~$10/mo
- **Total: ~$15-20/mo**

**At high usage (100K spans/day):**
- Firestore: ~$15/mo
- BigQuery: ~$50/mo
- **Total: ~$65/mo**

**Free tier coverage:**
- Firestore: 50K reads, 20K writes, 1GB storage/day
- BigQuery: 10GB storage, 1TB queries/mo
- **Development usage likely free**

## Success Criteria

- [ ] `AILANG_STORAGE=local` works identically to current behavior
- [ ] `AILANG_STORAGE=gcp` deploys to Cloud Run successfully
- [ ] All 36+ observatory tests pass on both backends
- [ ] Migration tool successfully transfers 26K spans
- [ ] Observatory queries <1s for aggregations on 1M spans
- [ ] Real-time WebSocket updates work with Firestore listeners
- [ ] Cost monitoring dashboard shows per-query costs
- [ ] All tests passing (`make test`)
- [ ] Documentation updated with deployment guide
- [ ] Example Cloud Run deployment working

## Testing Strategy

**Unit tests:**
- Backend interface compliance tests
- Query translation correctness
- Cost estimation accuracy

**Integration tests:**
- Firestore emulator for coordinator/collaboration
- BigQuery sandbox for observatory
- Full workflow: create task → spans → query

**End-to-end tests:**
- Local → GCP migration
- Cloud Run deployment
- Multi-user concurrent access

**Performance tests:**
- BigQuery query latency at 1M spans
- Firestore write throughput
- Streaming insert latency

## Non-Goals

**Not in this feature:**
- Multi-region deployment - Single region sufficient for v0.7.0
- Custom BigQuery ML models - Future analytics feature
- Firestore offline mode - Server-side only
- DuckDB support - May revisit if BigQuery costs become prohibitive
- Cross-cloud support (AWS, Azure) - GCP focus first

## Timeline

**Week 1** (20 hours):
- Phase 1: Backend abstraction
- Begin Phase 2: Firestore client

**Week 2** (24 hours):
- Complete Phase 2: Firestore backends
- Begin Phase 3: BigQuery backend

**Week 3** (16 hours):
- Complete Phase 3: BigQuery backend
- Phase 4: Hybrid mode, migration, docs
- Testing and release

**Total: ~60 hours across 3 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| BigQuery query costs spike | High | Add cost limits, query caching |
| Firestore hot spots | Medium | Distribute keys, use subcollections |
| Migration data loss | High | Verify counts, keep SQLite backup |
| Network latency affects UX | Medium | Hybrid mode keeps coordinator local |
| GCP API changes | Low | Pin client library versions |
| Auth complexity | Medium | Use Application Default Credentials |

## Related Documents

**Implemented (informs design):**
- [M-TASK-HIERARCHY](../../../implemented/v0_6_5/m-task-hierarchy-linking.md) - Current database schema
- [Observatory Architecture](../../../implemented/v0_6_4/observatory-architecture.md) - Backend interface design

**Planned (coordinate with):**
- [Global Collaboration Hub](global-collaboration-hub.md) - Multi-user requirements
- [Cloud Eval Workers](m-cloud-eval-workers.md) - Cloud Run deployment

## References

- [Design Axioms](/docs/references/axioms)
- [Firestore Data Model](https://cloud.google.com/firestore/docs/data-model)
- [BigQuery Best Practices](https://cloud.google.com/bigquery/docs/best-practices-performance-overview)
- [Cloud Run Deployment](https://cloud.google.com/run/docs/deploying)

## Future Work

**v0.8.0+:**
- Multi-region Firestore with automatic failover
- BigQuery ML for anomaly detection on spans
- Cost budgets with automatic alerts
- Cross-project data sharing for teams
- DuckDB fallback for cost-sensitive deployments

---

**Document created**: 2026-01-09
**Last updated**: 2026-01-09

# Sprint Plan: Collaboration Hub v2

## Summary
Implement comprehensive metrics aggregation, history/audit trails, real-time sync improvements, and trends visualization for the Collaboration Hub. This addresses 7 identified issues: lost metrics, approval history, "user" agent confusion, instance tracking, trends visualization, sync issues, and Claude Code integration.

**Duration:** 5 days (4 phases)
**Dependencies:** None (builds on existing messaging/websocket infrastructure)
**Risk Level:** Medium (multiple interconnected systems)

## Current Status Analysis

### Completed Recently
- ✅ WebSocket polling for agent messages (~100 LOC) in 1 day
- ✅ MetadataJSON in WebSocket events (~20 LOC) in 1 hour
- ✅ Unknown agent hierarchy display (~50 LOC) in 1 hour
- ✅ Thread creation for agents (~80 LOC) in 1 hour
- ✅ Multi-agent UI support (~300 LOC) in 2 days
- ✅ Monitoring dashboard (~200 LOC) in 1 day

### Velocity
- Recent average: ~150-200 LOC/day (React+Go)
- Estimated capacity: 750-1000 LOC for this sprint
- Design doc estimate: ~1,610 LOC across new/modified files

### Remaining from Design Doc
- ⏳ Phase 1 - Metrics Foundation: ~550 LOC
- ⏳ Phase 2 - History & Audit: ~480 LOC
- ⏳ Phase 3 - Real-time Sync: ~150 LOC
- ⏳ Phase 4 - Visualization & Integration: ~430 LOC

## Proposed Milestones

### Milestone 1: Metrics Foundation (Phase 1)
**Goal:** Add metrics aggregation at global/agent/thread levels with API endpoints and UI display
**Estimated:** 350 LOC implementation + 200 LOC tests = 550 LOC
**Duration:** 1.5 days

**Tasks:**
- Day 1 AM:
  - Add `metrics_aggregates` table to store.go with migration
  - Create `internal/messaging/metrics.go` with aggregation logic
  - Add trigger on message creation to update aggregates
- Day 1 PM:
  - Add `/api/metrics/*` endpoints to server
  - Extract execution_stats from message metadata JSON
  - Unit tests for aggregation calculations
- Day 2 AM:
  - Create `MetricsCard` React component
  - Add metrics display to hierarchy nodes (Monitor.tsx)
  - Wire up API calls

**Acceptance Criteria:**
- [ ] `metrics_aggregates` table created with proper indexes
- [ ] Global metrics endpoint returns total runs, tokens, cost, duration
- [ ] Per-agent metrics visible in hierarchy
- [ ] Per-thread metrics visible in thread header
- [ ] Aggregation runs on every result message
- [ ] All tests passing, linting clean

**Risks:**
- Concurrent aggregation race conditions - Mitigation: Use SQL transactions with UPSERT

### Milestone 2: History & Audit (Phase 2)
**Goal:** Persistent history for approvals and agent instances with retention policies
**Estimated:** 300 LOC implementation + 180 LOC tests = 480 LOC
**Duration:** 1 day

**Tasks:**
- Day 2 PM:
  - Add `approval_history` and `instance_history` tables
  - Modify approval operations to record history events
  - Add instance lifecycle tracking (start/end)
- Day 3 AM:
  - Add `/api/approvals/history` endpoint
  - Add `/api/instances/history` endpoint
  - Create `ApprovalHistory` React component
  - Add history tab to approval queue panel

**Acceptance Criteria:**
- [ ] All approval state changes recorded (created/approved/rejected)
- [ ] Instance start/end times tracked with metrics
- [ ] History endpoints return chronological events
- [ ] History visible in UI with timestamps and actors
- [ ] 30-day retention policy enforced
- [ ] All tests passing, linting clean

**Risks:**
- Database growth from history - Mitigation: Implement retention policy, add cleanup job

### Milestone 3: Real-time Sync (Phase 3)
**Goal:** Fix remaining WebSocket sync issues, add connection indicator, improve running status
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- Day 3 PM:
  - Add WebSocket connection state indicator to UI
  - Fix polling race conditions (sequence number edge cases)
  - Implement reconnection with exponential backoff
  - Add elapsed time to running spinner

**Acceptance Criteria:**
- [ ] Connection status visible in UI header (connected/reconnecting/disconnected)
- [ ] Messages update within 500ms of agent writing to DB
- [ ] Running status shows spinner with elapsed time ("Running... 12s")
- [ ] Graceful reconnection on network interruption
- [ ] All tests passing, linting clean

**Risks:**
- WebSocket reliability under poor network - Mitigation: Fallback to polling on disconnect

### Milestone 4: Visualization & Integration (Phase 4)
**Goal:** Trends charts, "user" agent handling, Claude Code session tracking
**Estimated:** 280 LOC implementation + 150 LOC tests = 430 LOC
**Duration:** 2 days

**Tasks:**
- Day 4 AM:
  - Install Recharts: `cd ui && npm install recharts`
  - Create `TrendsChart` React component with sparklines
  - Add `/api/metrics/trends` endpoint for time-series data
- Day 4 PM:
  - Implement "user" agent special handling (read-only, shows CLI runs)
  - Add claude_sessions table and tracking
  - Modify agent hierarchy to distinguish "user" from AI agents
- Day 5:
  - Create dashboard overview with global trends
  - Link sessions to triggered agent runs
  - End-to-end testing and polish
  - Documentation updates

**Acceptance Criteria:**
- [ ] TrendsChart shows tokens/cost over time (hourly granularity)
- [ ] "user" node is read-only (no thread creation button)
- [ ] "user" shows CLI `ailang run` executions
- [ ] Claude sessions table captures workspace and timestamps
- [ ] Dashboard shows global metrics with sparklines
- [ ] All tests passing, linting clean, docs updated

**Risks:**
- Recharts bundle size - Mitigation: Dynamic import to reduce initial load
- Claude Code integration complexity - Mitigation: Start with manual hook, automate later

## Success Metrics
- Test coverage: >80% for new code
- All 7 identified issues addressed
- Documentation: Update CHANGELOG.md, README.md
- All tests passing: ✅
- All linting passing: ✅
- <500ms latency for real-time updates: ✅

## Dependencies
- Recharts library (install via npm)
- Existing messaging/websocket infrastructure
- SQLite database

## Open Questions
1. **Retention period**: 30 days default - is this appropriate?
2. **Claude Code integration**: Manual hook vs automatic detection?
3. **Chart granularity**: Hour/day/week options or just hourly?

## Day-by-Day Schedule

| Day | Morning | Afternoon |
|-----|---------|-----------|
| Day 1 | M1: DB schema, metrics service | M1: API endpoints, tests |
| Day 2 | M1: React components, wire up | M2: History tables, operations |
| Day 3 | M2: API, React component | M3: WebSocket fixes, indicators |
| Day 4 | M4: Recharts, TrendsChart | M4: "user" agent, sessions |
| Day 5 | M4: Dashboard, e2e testing | Final polish, documentation |

## Notes
- Design document: `design_docs/planned/v0_5_0/collaboration-hub-v2.md`
- Sprint targets v0.5.0 release
- Consider feature flags for gradual rollout if needed
- React build required after UI changes: `cd ui && npm run build && cp -r dist/* ../internal/server/dist/`

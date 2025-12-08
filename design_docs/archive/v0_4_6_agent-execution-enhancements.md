# Agent Execution Enhancements

**Status**: 📋 Planned
**Target**: v0.4.6+
**Priority**: P1 (Medium) - Enhances existing agent execution system
**Estimated**: 10-15 days across 3 major features
**Dependencies**: Agent Execution Integration (v0.4.5 - Complete)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | N/A | 0 | Infrastructure feature - no language syntax changes |
| Preserve Semantic Clarity | N/A | 0 | Infrastructure feature - no language semantics changes |
| Increase Determinism | ++ | +2 | Resource budgets enforce deterministic cost/time bounds |
| Lower Token Cost | + | +1 | Progress streaming reduces wasted executions from timeouts |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale**: These enhancements increase determinism (resource bounds) and reduce token waste (progress streaming), directly supporting AI-first workflows with predictable costs and execution traces.

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The Agent Execution Integration (v0.4.5) provides **functional** autonomous code execution, but has several limitations for production use:

### Current Limitations

1. **No Progress Visibility** (UX Problem)
   - Long-running directives (5-10 min) show no progress
   - User doesn't know if agent is working or stuck
   - Can't cancel in-progress executions
   - All-or-nothing: complete success or complete failure

2. **Unbounded Resource Usage** (Cost/Safety Problem)
   - No limits on execution time (could run forever)
   - No limits on cost (could spend unlimited $$$)
   - No limits on API calls, file operations, network requests
   - Agent could accidentally DoS itself or external services

3. **No Observability** (Operations Problem)
   - Can't tell if agents are alive, stuck, or crashed
   - No execution history or analytics
   - No way to debug failed directives
   - No performance metrics (avg cost, success rate, etc.)

### User Stories

**As a user**, I want to:
- See progress updates during long-running directives (not just final result)
- Set cost/time budgets so agents don't overspend
- Cancel in-progress executions if they're going off-track
- See agent health status (alive, busy, idle)
- Query execution history and analyze costs

**As a developer**, I want to:
- Debug failed directives with full execution traces
- Monitor agent performance over time
- Set resource limits per-directive or per-agent
- Receive alerts when agents are unhealthy

## Proposed Solution

Three major enhancements, each independently valuable:

### Enhancement 1: Streaming Progress Updates (P0 - Critical for UX)

**Goal**: Real-time progress updates during execution, not just final results.

**Implementation Strategy**:

```go
// internal/agent/executor.go

type ProgressUpdate struct {
    Timestamp   time.Time
    Type        string  // "turn", "tool_use", "output", "error"
    Content     string
    Metadata    map[string]interface{}
}

type StreamingExecutor struct {
    executor        *DirectiveExecutor
    progressChannel chan *ProgressUpdate
}

func (e *StreamingExecutor) ExecuteWithProgress(directive string) (*DirectiveResult, error) {
    // Start execution in goroutine
    resultChan := make(chan *DirectiveResult)
    go func() {
        result := e.executor.Execute(directive)
        resultChan <- result
    }()

    // Stream progress updates
    for {
        select {
        case update := <-e.progressChannel:
            // Publish to collaboration hub
            e.publishProgress(update)
        case result := <-resultChan:
            return result, nil
        }
    }
}
```

**Changes Required**:

1. **Modify eval harness** to expose progress stream
   - Already has turn-by-turn output in streaming mode
   - Expose as Go channel instead of stdout
   - ~50 LOC changes to `internal/eval_harness/agent_runner_streaming.go`

2. **Add progress message type** to messaging schema
   - New message kind: `'progress'`
   - Schema: `{turn: int, type: string, content: string, timestamp: int64}`
   - ~20 LOC changes to `internal/messaging/schema.go`

3. **Update agent to publish progress**
   - Subscribe to progress channel
   - Publish progress messages to thread
   - ~100 LOC in `cmd/ailang-agent/agent.go`

4. **Add cancellation support**
   - Context-based cancellation in executor
   - UI can send `'cancel'` message to stop execution
   - ~80 LOC across executor and agent

**Estimated LOC**: ~250 LOC
**Estimated Time**: 2-3 days
**Value**: High - Dramatically improves UX for long-running directives

---

### Enhancement 2: Resource Budgets & Limits (P0 - Critical for Cost Control)

**Goal**: Enforce limits on execution time, cost, and resource usage.

**Implementation Strategy**:

```go
// internal/agent/budgets.go

type ResourceBudget struct {
    MaxDurationSec  int     // Max execution time (seconds)
    MaxCostUSD      float64 // Max total cost (dollars)
    MaxTurns        int     // Max Claude turns (complexity limit)
    MaxFileWrites   int     // Max files created/modified
    MaxAPIRequests  int     // Max network requests
}

type BudgetEnforcer struct {
    budget    ResourceBudget
    used      ResourceUsage  // Track actual usage
    mu        sync.Mutex
}

func (b *BudgetEnforcer) CheckBudget() error {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.used.DurationSec > b.budget.MaxDurationSec {
        return ErrBudgetExceeded{"time", b.used.DurationSec, b.budget.MaxDurationSec}
    }
    if b.used.CostUSD > b.budget.MaxCostUSD {
        return ErrBudgetExceeded{"cost", b.used.CostUSD, b.budget.MaxCostUSD}
    }
    // ... check other limits

    return nil
}
```

**Changes Required**:

1. **Add budget configuration**
   - Per-directive budgets (from capability detection)
   - Per-agent default budgets (configuration)
   - Per-user budgets (future: multi-tenant)
   - ~150 LOC for budget types and validation

2. **Instrument executor to track usage**
   - Hook into eval harness turn counter
   - Track file operations from workspace
   - Track network requests (future: requires proxy)
   - ~200 LOC for tracking and enforcement

3. **Budget enforcement logic**
   - Check budgets after each turn
   - Gracefully terminate if exceeded
   - Send budget-exceeded message to UI
   - ~100 LOC for enforcement and messaging

4. **Budget estimation improvements**
   - Enhance capability detector cost estimation
   - Use historical data for better predictions
   - ~100 LOC for improved estimation

**Estimated LOC**: ~550 LOC
**Estimated Time**: 4-5 days
**Value**: High - Prevents runaway costs and resource exhaustion

**Risk Mitigation**:
- Start with conservative defaults (max 5 min, $1.00)
- Allow users to override budgets with approval
- Log budget violations for analysis

---

### Enhancement 3: Agent Health Monitoring & Analytics (P1 - Operational Excellence)

**Goal**: Monitor agent health and provide execution history/analytics.

**Implementation Strategy**:

```go
// internal/agent/health.go

type AgentHealth struct {
    AgentID        string
    Status         string  // "idle", "busy", "stuck", "crashed"
    LastHeartbeat  time.Time
    CurrentTask    *TaskInfo  // nil if idle
    Uptime         time.Duration
    TasksCompleted int
    TasksFailed    int
    TotalCost      float64
}

type HealthMonitor struct {
    agents       map[string]*AgentHealth
    heartbeatTTL time.Duration  // 30 seconds
}

func (m *HealthMonitor) RecordHeartbeat(agentID string) {
    // Update last heartbeat timestamp
    // Transition from "crashed" to "idle" if recovered
}

func (m *HealthMonitor) DetectStuckAgents() []string {
    // Find agents with tasks running > MaxDurationSec
    // Mark as "stuck" if no progress updates in 2 minutes
}
```

**Changes Required**:

1. **Add heartbeat mechanism**
   - Agents send heartbeat every 10 seconds
   - Heartbeat includes current status (idle/busy)
   - Store in `agent_health` table
   - ~150 LOC for heartbeat sending and storage

2. **Add execution history tracking**
   - New `execution_history` table
   - Record every directive execution
   - Schema: `{id, agent_id, directive, start_time, end_time, status, cost, error}`
   - ~200 LOC for tracking and storage

3. **Add analytics queries**
   - `GetExecutionHistory(limit, filters)`
   - `GetAgentStats(agentID)` - success rate, avg cost, etc.
   - `GetCostByTimeRange(start, end)` - cost analysis
   - ~150 LOC for query functions

4. **Add health check endpoint**
   - `/health` endpoint for monitoring systems
   - Returns agent statuses, recent failures
   - ~50 LOC for HTTP handler

**Database Schema**:

```sql
-- Agent health tracking
CREATE TABLE agent_health (
    agent_id TEXT PRIMARY KEY,
    status TEXT CHECK(status IN ('idle', 'busy', 'stuck', 'crashed')),
    last_heartbeat INTEGER,  -- Unix timestamp ms
    current_task_id TEXT,
    uptime_sec INTEGER,
    tasks_completed INTEGER,
    tasks_failed INTEGER,
    total_cost_usd REAL,
    updated_at INTEGER
);

-- Execution history
CREATE TABLE execution_history (
    id TEXT PRIMARY KEY,
    agent_id TEXT,
    thread_id TEXT,
    directive TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    status TEXT CHECK(status IN ('success', 'failed', 'timeout', 'cancelled', 'budget_exceeded')),
    cost_usd REAL,
    duration_ms INTEGER,
    turns INTEGER,
    files_created INTEGER,
    error TEXT,
    FOREIGN KEY (thread_id) REFERENCES threads(id)
);

-- Indexes for analytics
CREATE INDEX idx_execution_history_agent ON execution_history(agent_id);
CREATE INDEX idx_execution_history_time ON execution_history(started_at);
CREATE INDEX idx_execution_history_cost ON execution_history(cost_usd);
```

**Estimated LOC**: ~550 LOC
**Estimated Time**: 4-5 days
**Value**: Medium - Improves operations and debugging

---

## Implementation Plan

### Phase 1: Streaming Progress Updates (Week 1)
- **Days 1-2**: Modify eval harness for progress streaming
- **Day 3**: Add progress message type to schema
- **Days 4-5**: Update agent with progress publishing and cancellation
- **Deliverable**: Real-time progress updates in UI

### Phase 2: Resource Budgets (Week 2)
- **Days 1-2**: Add budget types and configuration
- **Days 3-4**: Instrument executor with usage tracking
- **Day 5**: Budget enforcement and messaging
- **Deliverable**: Cost/time/resource limits enforced

### Phase 3: Health Monitoring (Week 3)
- **Days 1-2**: Add heartbeat mechanism and agent_health table
- **Days 3-4**: Add execution history tracking
- **Day 5**: Analytics queries and health endpoint
- **Deliverable**: Agent health monitoring and execution history

### Optional Phase 4: Advanced Features (Future)
- **Agent Specialization**: Different agent types (frontend, backend, DevOps)
- **Workspace Persistence**: Continue work across executions
- **Better Capability Detection**: LLM-based classification for edge cases
- **Retry Logic**: Automatic retry on transient failures

---

## Success Metrics

### Streaming Progress Updates
- ✅ Progress updates published within 1 second of turn completion
- ✅ Users can cancel executions within 5 seconds
- ✅ Reduced perceived wait time (UX survey)

### Resource Budgets
- ✅ Zero runaway cost incidents (cost > 10x estimate)
- ✅ 95%+ of directives complete within budget
- ✅ Average cost per directive < $0.10

### Health Monitoring
- ✅ Agent crashes detected within 30 seconds
- ✅ Stuck agents auto-recovered within 2 minutes
- ✅ Execution history queryable with <100ms latency

---

## Risks & Mitigations

### Risk 1: Streaming overhead increases latency
- **Mitigation**: Progress updates are async, don't block execution
- **Mitigation**: Batch updates (max 1/sec) to reduce message volume

### Risk 2: Budget enforcement is too strict
- **Mitigation**: Start with generous defaults, tune based on usage
- **Mitigation**: Allow users to request budget increases with approval

### Risk 3: Health monitoring adds complexity
- **Mitigation**: Keep heartbeat lightweight (10 bytes/10 sec)
- **Mitigation**: Make monitoring optional (can be disabled)

---

## Testing Strategy

### Streaming Progress Updates
- Unit tests: Progress channel publishes expected updates
- Integration tests: Full execution with progress streaming
- Load tests: 10 agents streaming simultaneously

### Resource Budgets
- Unit tests: Budget enforcement triggers at correct thresholds
- Integration tests: Execution stops when budget exceeded
- Edge cases: Near-budget executions, budget increase mid-execution

### Health Monitoring
- Unit tests: Heartbeat updates agent status
- Integration tests: Stuck agent detection
- Performance tests: Query 10k execution history records

---

## Alternative Approaches Considered

### Alternative 1: Server-Sent Events (SSE) for progress
**Rejected**: Adds HTTP dependency, more complex than message-based approach

### Alternative 2: Pre-execution cost estimation via LLM
**Rejected**: Too slow (~500ms), not accurate enough, adds cost

### Alternative 3: Agent process monitoring (OS-level)
**Rejected**: Platform-specific, doesn't work in containers/cloud

---

## Dependencies

### Internal Dependencies
- ✅ Agent Execution Integration (v0.4.5) - Complete
- ✅ UI Collaboration Hub (v0.4.4) - Complete
- ⏳ UI updates to display progress (separate work)

### External Dependencies
- Claude Code CLI (existing)
- SQLite 3.x (existing)
- No new external dependencies

---

## Documentation Requirements

1. **User Guide**: How to set budgets, monitor agents, query history
2. **Admin Guide**: How to deploy agents, configure limits, monitor health
3. **API Reference**: New message types, database schema, HTTP endpoints
4. **Troubleshooting Guide**: Common issues (stuck agents, budget exceeded, etc.)

---

## Open Questions

1. **Should budgets be per-directive or per-agent?**
   - Per-directive: More granular, harder to configure
   - Per-agent: Simpler, but less control
   - **Recommendation**: Both - directive overrides agent default

2. **How to handle budget increase requests?**
   - Option A: Automatic for low-risk increases (<2x)
   - Option B: Always require approval
   - **Recommendation**: Option A with configurable threshold

3. **Should execution history include full transcripts?**
   - Pros: Complete debugging info
   - Cons: Large storage (100KB-1MB per execution)
   - **Recommendation**: Store compressed, with TTL (30 days)

---

## Future Enhancements (Out of Scope)

These are valuable but deferred to later versions:

1. **Multi-Agent Collaboration** (v0.4.7+)
   - Agents coordinate on complex tasks
   - Task decomposition and delegation
   - Consensus-based decision making

2. **Agent Learning & Optimization** (v0.5.0+)
   - Learn from past executions
   - Optimize capability detection
   - Personalized agent behavior

3. **Distributed Agent Pools** (v0.5.0+)
   - Multiple machines running agents
   - Load balancing across pool
   - Fault tolerance with agent replication

---

## Conclusion

These three enhancements transform the agent execution system from **functional** to **production-ready**:

1. **Streaming Progress**: Better UX, user confidence, cancellation
2. **Resource Budgets**: Cost control, safety, predictability
3. **Health Monitoring**: Operations, debugging, analytics

**Total Estimated Effort**: 10-15 days (3 weeks)
**Total Estimated LOC**: ~1,350 LOC
**Value**: High - Essential for production deployment

**Recommendation**: Implement in sequence (Phase 1 → 2 → 3), ship incrementally.

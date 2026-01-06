# M-CONTROL-PLANE-V4: AILANG Operations Control Plane

**Status**: Planned
**Target**: v0.6.4
**Priority**: P0 (High)
**Estimated**: 2-3 weeks (Fresh Start rewrite)
**Dependencies**: M-OTEL-DASHBOARD (traces), M-COLLAB-PROVIDER-STATS (cost tracking)
**Author**: Claude + Mark
**Created**: 2026-01-06

---

## Executive Summary

Unify AILANG's fragmented dashboard components into a comprehensive **AI Operations Control Plane** designed for Platform/Ops teams monitoring 20+ concurrent AI agents across an organization. The control plane provides multi-level visibility (global → workspace → task → agent → model), scope-based trust automation, and three core visualizations: activity heatmaps, trace waterfalls, and agent topology DAGs.

---

## Problem Statement

### Current State: Fragmented UI (v3)

The current dashboard evolved organically with separate concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│ Current Architecture (v3) - Disconnected Components            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Message     │  │ Observatory │  │ Stats       │             │
│  │ Center      │  │ (Traces)    │  │ Panel       │             │
│  │             │  │             │  │             │             │
│  │ - Threads   │  │ - Span list │  │ - Costs     │             │
│  │ - Messages  │  │ - Metrics   │  │ - Tokens    │             │
│  │ - Approvals │  │ - Tasks     │  │ - Provider  │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         └────────────────┴────────────────┘                     │
│                          │                                      │
│                    No unified view                              │
│                    No relationship mapping                      │
│                    No trust progression                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Pain Points:**
1. **No correlation** between messages, tasks, and traces
2. **No activity patterns** - can't see when agents are busy
3. **No topology view** - can't see agent relationships
4. **No trust automation** - all approvals are manual
5. **No multi-tenancy** - can't filter by workspace/team
6. **Persistent UI bugs** - running status never clears, CSS breaks

### Target User Persona

**Platform/Ops Engineer** managing organization-wide AI workloads:
- Monitors 20+ concurrent agents across multiple projects
- Needs anomaly detection ("why is agent X consuming 10x tokens?")
- Manages resource governance and cost allocation
- Configures trust levels per agent per capability
- Eventually deploys to GCP Cloud with auth

---

## Goals

### Primary Goal

Build a unified **AILANG Control Plane** that provides complete visibility into AI agent operations with actionable insights and automation.

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Time to understand system state | ~2 min (click through tabs) | <10 sec (single view) |
| Agent relationship visibility | None | Full DAG with live status |
| Trust automation coverage | 0% (all manual) | 60% of routine ops automated |
| Cross-entity correlation | Manual | Automatic (click trace → see messages) |
| Anomaly detection | None | Cost/token spike alerts |

---

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AILANG Control Plane v4 Architecture                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         Command Bar (Global)                          │   │
│  │  [Search: traces, messages, tasks...]  [Time Range]  [Filters]       │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────┐ ┌──────────────────────────────────────────────────┐   │
│  │                 │ │                                                   │   │
│  │  Aggregation    │ │              Main Canvas                         │   │
│  │  Navigator      │ │                                                   │   │
│  │                 │ │  ┌─────────────────────────────────────────────┐ │   │
│  │  ○ Global       │ │  │  Activity Heatmap (GitHub-style calendar)   │ │   │
│  │  ├─ Workspace 1 │ │  │  ░░█░░░░█████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │ │   │
│  │  │  ├─ Agent A  │ │  │  ░█████░░░░░░██░░░░░░░░░░░░░░░░░░░░░░░░░░░  │ │   │
│  │  │  └─ Agent B  │ │  │       Jan        Feb        Mar              │ │   │
│  │  ├─ Workspace 2 │ │  └─────────────────────────────────────────────┘ │   │
│  │  └─ Workspace 3 │ │                                                   │   │
│  │                 │ │  ┌─────────────────────────────────────────────┐ │   │
│  │  Provider       │ │  │  Agent Topology DAG (Hub & Spoke)           │ │   │
│  │  ├─ claude      │ │  │                                              │ │   │
│  │  └─ gemini      │ │  │       ┌─────┐                               │ │   │
│  │                 │ │  │   ┌───│ GH  │───┐                           │ │   │
│  │  Model          │ │  │   │   └─────┘   │                           │ │   │
│  │  ├─ opus        │ │  │   ▼             ▼                           │ │   │
│  │  ├─ sonnet      │ │  │ ┌─────┐     ┌─────┐                         │ │   │
│  │  └─ haiku       │ │  │ │Doc  │────▶│Plan │────▶┌─────┐             │ │   │
│  │                 │ │  │ └─────┘     └─────┘     │Exec │             │ │   │
│  │  Trust Level    │ │  │                         └─────┘             │ │   │
│  │  ├─ Manual      │ │  └─────────────────────────────────────────────┘ │   │
│  │  ├─ Low-Risk    │ │                                                   │   │
│  │  └─ Automated   │ │  ┌─────────────────────────────────────────────┐ │   │
│  │                 │ │  │  Trace Waterfall (Selected Entity)          │ │   │
│  └─────────────────┘ │  │  ├─ compile.parse ████░░░░░░░░░░░  12ms     │ │   │
│                       │  │  ├─ compile.typecheck ░███████░░░  45ms     │ │   │
│                       │  │  └─ eval.execute ░░░░░░░░█████████ 89ms    │ │   │
│                       │  └─────────────────────────────────────────────┘ │   │
│                       └──────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Context Panel (Slide-out)                          │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐         │   │
│  │  │ Messages   │ │ Approvals  │ │ Cost       │ │ Resources  │         │   │
│  │  │ (Thread)   │ │ (Queue)    │ │ (Breakdown)│ │ (CPU/Mem)  │         │   │
│  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘         │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Core Visualizations

#### 1. Activity Heatmap (GitHub-style Calendar)

Shows activity intensity over time with drill-down capability.

```
Activity Heatmap - Global View (Past 90 Days)
═══════════════════════════════════════════════════════════════════

        Week 1   Week 2   Week 3   Week 4   Week 5   ...
Mon     ░░░░░    ░░░░░    █████    ░░░░░    ░░███
Tue     ░░░██    ░░░░░    █████    ░░░░░    █████
Wed     ░░███    ░░░░░    ░░███    ░░░░░    █████
Thu     █████    ░░░░░    ░░░░░    █████    ░░░░░
Fri     ░░░░░    █████    ░░░░░    █████    ░░░░░
Sat     ░░░░░    ░░░░░    ░░░░░    ░░░░░    ░░░░░
Sun     ░░░░░    ░░░░░    ░░░░░    ░░░░░    ░░░░░

Legend: ░ = 0-10 tasks  ▒ = 11-50 tasks  ▓ = 51-100 tasks  █ = 100+ tasks

Click cell → Drill to hour-by-hour view → Drill to individual tasks
```

**Data Source**: `coordinator.db` tasks table + timestamps
**Aggregations**: Count, cost, tokens, success rate (toggle)

#### 2. Agent Topology DAG (Hub & Spoke)

Shows agent relationships, handoffs, and current state.

```
Agent Topology - Data Flow View
═══════════════════════════════════════════════════════════════════

                              ┌──────────────┐
                              │   GitHub     │
                              │   Issues     │
                              │    ●●●       │  ← 3 pending
                              └──────┬───────┘
                                     │ import
                                     ▼
         ┌───────────────────────────────────────────────────┐
         │                                                   │
    ┌────▼─────┐        ┌───────────┐        ┌───────────┐  │
    │  design- │   →    │  sprint-  │   →    │  sprint-  │  │
    │  doc-    │        │  planner  │        │  executor │  │
    │  creator │        │           │        │           │  │
    │          │        │           │        │           │  │
    │ ○ idle   │        │ ● busy    │        │ ○ idle    │  │
    │ trust:80 │        │ trust:65  │        │ trust:45  │  │
    └────┬─────┘        └─────┬─────┘        └─────┬─────┘  │
         │                    │                    │        │
         │    messages        │    messages        │        │
         └────────────────────┴────────────────────┘        │
                              │                             │
                              ▼                             │
                        ┌───────────┐                       │
                        │  Approval │                       │
                        │  Queue    │  ← Human in the loop  │
                        │   12 ⏳    │                       │
                        └───────────┘                       │
                              │                             │
                              ▼                             │
                        ┌───────────┐                       │
                        │   Main    │                       │
                        │   Branch  │                       │
                        │   ✓ 29    │  ← Merged             │
                        └───────────┘                       │
         │                                                   │
         └───────────────────────────────────────────────────┘

Legend:
  ○ idle    ● busy    ◐ blocked    ✕ error
  → handoff message
  ⏳ pending approval
  ✓ completed
```

**Interaction:**
- Click agent → Show detail panel (trust scores, recent tasks, costs)
- Click edge → Show message history between agents
- Drag to rearrange layout
- Filter by: active only, specific workspace, trust level

#### 3. Trace Waterfall

Shows timing breakdown for selected entity (task, request, etc).

```
Trace Waterfall - Task: "Fix parser bug"
═══════════════════════════════════════════════════════════════════

Request ID: trace_abc123
Duration: 2m 34s
Status: ✓ Completed
Cost: $0.0847

Timeline (0s ────────────────────────────────────────── 154s)
├─ coordinator.analyze          ██░░░░░░░░░░░░░░░░░░░░░░░░  2.3s
├─ coordinator.create_worktree  ░██░░░░░░░░░░░░░░░░░░░░░░░  1.2s
├─ executor.claude.init         ░░██░░░░░░░░░░░░░░░░░░░░░░  0.8s
├─ executor.claude.execute      ░░░░████████████████████░░  142s
│  ├─ claude.read_files         ░░░░██░░░░░░░░░░░░░░░░░░░░  3.2s
│  ├─ claude.generate           ░░░░░░██████████████░░░░░░  98s
│  │  └─ anthropic.api          ░░░░░░██████████████░░░░░░  96s
│  └─ claude.write_files        ░░░░░░░░░░░░░░░░░░██░░░░░░  4.1s
├─ coordinator.validate         ░░░░░░░░░░░░░░░░░░░░██░░░░  3.8s
└─ coordinator.finalize         ░░░░░░░░░░░░░░░░░░░░░░██░░  2.1s

Resource Usage:
  Input Tokens:  12,847 ($0.0385)
  Output Tokens: 4,231  ($0.0462)
  Peak Memory:   124 MB
  CPU Time:      8.3s user / 2.1s sys
```

### Data Model

#### Unified Entity Graph

```
┌─────────────────────────────────────────────────────────────────┐
│                     Entity Relationship Model                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐   │
│  │Workspace│ 1:N │  Task   │ 1:N │  Trace  │ 1:N │  Span   │   │
│  │         │────▶│         │────▶│         │────▶│         │   │
│  └─────────┘     └────┬────┘     └─────────┘     └─────────┘   │
│                       │                                         │
│                       │ 1:N                                     │
│                       ▼                                         │
│                  ┌─────────┐                                    │
│                  │ Message │                                    │
│                  │         │                                    │
│                  └────┬────┘                                    │
│                       │                                         │
│                       │ N:1                                     │
│                       ▼                                         │
│                  ┌─────────┐     ┌─────────┐                   │
│                  │ Thread  │ N:1 │  Agent  │                   │
│                  │         │────▶│         │                   │
│                  └─────────┘     └────┬────┘                   │
│                                       │                         │
│                                       │ N:1                     │
│                                       ▼                         │
│                                  ┌─────────┐                   │
│                                  │Provider │                   │
│                                  │(claude/ │                   │
│                                  │ gemini) │                   │
│                                  └────┬────┘                   │
│                                       │                         │
│                                       │ N:1                     │
│                                       ▼                         │
│                                  ┌─────────┐                   │
│                                  │  Model  │                   │
│                                  │(opus/   │                   │
│                                  │ sonnet) │                   │
│                                  └─────────┘                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Aggregation Levels

| Level | Scope | Key Metrics |
|-------|-------|-------------|
| **Global** | All workspaces | Total tasks, cost, tokens, active agents, anomalies |
| **Workspace** | Single project | Tasks by status, cost breakdown, agent utilization |
| **Task** | Single operation | Duration, traces, messages, cost, result |
| **Agent** | Agent identity | Trust score, task history, handoff patterns |
| **Provider** | claude/gemini | API costs, latency, error rates |
| **Model** | opus/sonnet/etc | Token usage, cost per token, quality metrics |

### Scope-Based Trust Model

Trust is managed per capability, not globally:

```
Trust Configuration - Agent: sprint-executor
═══════════════════════════════════════════════════════════════════

Trust Scores (0-100):
┌────────────────────┬───────┬────────────────────────────────────┐
│ Capability         │ Score │ Automation Level                   │
├────────────────────┼───────┼────────────────────────────────────┤
│ Read Files         │  95   │ ████████████████████ Full Auto     │
│ Write Docs         │  80   │ ████████████████░░░░ Low-Risk Auto │
│ Write Code         │  45   │ █████████░░░░░░░░░░░ Review Queue  │
│ Run Tests          │  70   │ ██████████████░░░░░░ Low-Risk Auto │
│ Git Commit         │  60   │ ████████████░░░░░░░░ Review Queue  │
│ Git Push           │  30   │ ██████░░░░░░░░░░░░░░ Always Manual │
│ Create Release     │   0   │ ░░░░░░░░░░░░░░░░░░░░ Never Auto    │
└────────────────────┴───────┴────────────────────────────────────┘

Thresholds:
  0-25:  Always Manual (human must approve)
  25-60: Review Queue (batched review, low-priority)
  60-85: Low-Risk Auto (auto-approve with logging)
  85-100: Full Auto (no human needed)

Trust Evolution:
  - +5 per successful task in capability
  - -20 per failed task requiring rollback
  - -50 per security incident
  - Manual adjustment by admin
```

### API Contracts

#### Control Plane API

```go
// GET /api/controlplane/overview
type OverviewResponse struct {
    Timestamp     time.Time              `json:"timestamp"`
    Global        GlobalMetrics          `json:"global"`
    Workspaces    []WorkspaceMetrics     `json:"workspaces"`
    Agents        []AgentMetrics         `json:"agents"`
    Anomalies     []Anomaly              `json:"anomalies"`
}

type GlobalMetrics struct {
    ActiveTasks     int                   `json:"active_tasks"`
    PendingApprovals int                  `json:"pending_approvals"`
    TotalCostToday  float64               `json:"total_cost_today"`
    TotalTokens     int64                 `json:"total_tokens"`
    SuccessRate24h  float64               `json:"success_rate_24h"`
}

// GET /api/controlplane/heatmap?level={global|workspace|agent}&id={id}&range={days}
type HeatmapResponse struct {
    Level    string                 `json:"level"`
    ID       string                 `json:"id,omitempty"`
    Range    int                    `json:"range_days"`
    Cells    []HeatmapCell          `json:"cells"`
}

type HeatmapCell struct {
    Date        string              `json:"date"`       // YYYY-MM-DD
    Hour        int                 `json:"hour,omitempty"` // 0-23 for hourly view
    TaskCount   int                 `json:"task_count"`
    Cost        float64             `json:"cost"`
    Tokens      int64               `json:"tokens"`
    SuccessRate float64             `json:"success_rate"`
}

// GET /api/controlplane/topology?workspace={id}
type TopologyResponse struct {
    Nodes       []TopologyNode       `json:"nodes"`
    Edges       []TopologyEdge       `json:"edges"`
    LiveStatus  map[string]string    `json:"live_status"` // agent_id → status
}

type TopologyNode struct {
    ID          string              `json:"id"`
    Type        string              `json:"type"` // agent|source|sink
    Label       string              `json:"label"`
    TrustScore  int                 `json:"trust_score,omitempty"`
    Position    *Position           `json:"position,omitempty"` // user-saved layout
}

type TopologyEdge struct {
    Source      string              `json:"source"`
    Target      string              `json:"target"`
    Type        string              `json:"type"` // handoff|data|resource
    MessageCount int                `json:"message_count"`
    TotalCost   float64             `json:"total_cost"`
}

// GET /api/controlplane/trust/{agent_id}
// PUT /api/controlplane/trust/{agent_id}
type TrustConfig struct {
    AgentID     string                         `json:"agent_id"`
    Capabilities map[string]TrustCapability   `json:"capabilities"`
    UpdatedAt   time.Time                      `json:"updated_at"`
    UpdatedBy   string                         `json:"updated_by"`
}

type TrustCapability struct {
    Score       int                 `json:"score"`       // 0-100
    Level       string              `json:"level"`       // manual|review|low_risk|full_auto
    SuccessCount int                `json:"success_count"`
    FailureCount int                `json:"failure_count"`
}
```

---

## Component Architecture (React)

```
src/
├── features/
│   └── controlplane/
│       ├── ControlPlane.tsx           # Main container
│       ├── ControlPlane.module.css
│       ├── components/
│       │   ├── CommandBar/            # Global search, filters, time range
│       │   ├── AggregationNav/        # Left sidebar tree navigator
│       │   ├── Heatmap/               # GitHub-style activity calendar
│       │   │   ├── HeatmapCanvas.tsx  # D3 or custom canvas rendering
│       │   │   └── HeatmapTooltip.tsx
│       │   ├── Topology/              # Agent DAG visualization
│       │   │   ├── TopologyCanvas.tsx # Force-directed graph (d3-force)
│       │   │   ├── TopologyNode.tsx
│       │   │   └── TopologyEdge.tsx
│       │   ├── Waterfall/             # Trace timing visualization
│       │   │   ├── WaterfallChart.tsx
│       │   │   └── WaterfallSpan.tsx
│       │   ├── ContextPanel/          # Slide-out detail panel
│       │   │   ├── MessagesTab.tsx
│       │   │   ├── ApprovalsTab.tsx
│       │   │   ├── CostTab.tsx
│       │   │   └── ResourcesTab.tsx
│       │   └── TrustConfig/           # Trust level editor
│       │       ├── TrustMatrix.tsx
│       │       └── TrustSlider.tsx
│       ├── hooks/
│       │   ├── useControlPlaneData.ts # Data fetching + WebSocket
│       │   ├── useHeatmapData.ts
│       │   ├── useTopologyData.ts
│       │   └── useTrustConfig.ts
│       └── utils/
│           ├── aggregations.ts        # Client-side aggregation logic
│           └── anomalyDetection.ts    # Spike detection
```

---

## Implementation Plan

### Phase 1: Foundation (Days 1-4)

**M1: API Layer**
- [ ] Create `/api/controlplane/*` endpoints
- [ ] Implement aggregation queries in SQLite
- [ ] Add WebSocket events for live updates
- Estimated: 200 LOC Go

**M2: Core Layout**
- [ ] Create ControlPlane container
- [ ] Implement CommandBar with search
- [ ] Implement AggregationNav tree
- Estimated: 300 LOC React/CSS

### Phase 2: Visualizations (Days 5-10)

**M3: Activity Heatmap**
- [ ] Canvas-based heatmap renderer
- [ ] Click-to-drill interaction
- [ ] Legend and tooltips
- Estimated: 400 LOC React/D3

**M4: Agent Topology DAG**
- [ ] Force-directed layout (d3-force)
- [ ] Node rendering with status
- [ ] Edge rendering with flow animation
- [ ] Interactive zoom/pan
- Estimated: 500 LOC React/D3

**M5: Trace Waterfall**
- [ ] Horizontal bar chart renderer
- [ ] Nested span hierarchy
- [ ] Timing labels and tooltips
- Estimated: 350 LOC React/CSS

### Phase 3: Context & Trust (Days 11-14)

**M6: Context Panel**
- [ ] Slide-out panel container
- [ ] Messages tab (reuse ThreadView)
- [ ] Approvals tab with batch actions
- [ ] Cost breakdown tab
- [ ] Resource usage tab (CPU/mem)
- Estimated: 400 LOC React/CSS

**M7: Trust Configuration**
- [ ] Trust matrix view
- [ ] Slider-based score editor
- [ ] History/audit log
- [ ] Trust evolution chart
- Estimated: 300 LOC React/CSS

### Phase 4: Polish & Deploy (Days 15-17)

**M8: Integration**
- [ ] Connect all components
- [ ] End-to-end testing
- [ ] Performance optimization
- [ ] Dark/light theme support
- Estimated: 200 LOC

**M9: Documentation**
- [ ] User guide
- [ ] API documentation
- [ ] Deployment guide (GCP)
- Estimated: 50 LOC + docs

---

## Files to Create/Modify

### New Files

| File | LOC | Purpose |
|------|-----|---------|
| `internal/server/handlers_controlplane.go` | ~300 | API endpoints |
| `internal/controlplane/aggregations.go` | ~200 | SQL aggregation queries |
| `ui/src/features/controlplane/ControlPlane.tsx` | ~150 | Main container |
| `ui/src/features/controlplane/components/Heatmap/*.tsx` | ~400 | Heatmap vis |
| `ui/src/features/controlplane/components/Topology/*.tsx` | ~500 | DAG vis |
| `ui/src/features/controlplane/components/Waterfall/*.tsx` | ~350 | Trace vis |
| `ui/src/features/controlplane/components/TrustConfig/*.tsx` | ~300 | Trust editor |
| **Total** | **~2,200** | |

### Modified Files

| File | Changes |
|------|---------|
| `internal/server/server.go` | Add controlplane routes |
| `internal/coordinator/store_sqlite.go` | Add aggregation methods |
| `ui/src/App.tsx` | Add ControlPlane route |

---

## Axiom Compliance

| Axiom | Score | Notes |
|-------|-------|-------|
| A1: Determinism | +1 | All aggregations are deterministic point-in-time queries |
| A2: Replayability | +1 | Trace correlation enables full replay |
| A3: Effect Legibility | +1 | Trust levels make automation explicit |
| A4: Explicit Authority | +1 | Scope-based trust = explicit capability grants |
| A5: Bounded Verification | 0 | N/A (UI layer) |
| A6: Safe Concurrency | 0 | N/A (read-only views) |
| A7: Machines First | +1 | API-first design, JSON everywhere |
| A8: Minimal Syntax | 0 | N/A (UI layer) |
| A9: Cost Visibility | +1 | Cost breakdown at every level |
| A10: Composability | +1 | Independent visualizations compose |
| A11: Structured Failure | 0 | N/A (UI layer) |
| A12: System Boundary | +1 | Clear separation: API → UI → User |
| **Net Score** | **+8** | Strongly aligned |

---

## Success Criteria

- [ ] Single-page view shows global health in <10 seconds
- [ ] Click any entity → see related messages, traces, costs
- [ ] Heatmap correctly aggregates at all zoom levels
- [ ] Topology shows live agent status via WebSocket
- [ ] Trust config persists and affects approval routing
- [ ] All existing tests pass
- [ ] Performance: <100ms API response, 60fps visualizations

---

## Open Questions

1. **Persistence**: Should user layout preferences (DAG positions) be stored server-side?
2. **Multi-tenancy**: Do we need workspace-level auth before cloud deploy?
3. **Alerting**: Should anomalies trigger notifications (Slack/email)?
4. **Historical**: How much history to retain? (Storage implications)

---

## Related Documents

- [M-OTEL-DASHBOARD](./m-otel-dashboard.md) - Observatory/trace foundation
- [M-COLLAB-PROVIDER-STATS](../../planned/v0_6_3/m-collab-provider-stats.md) - Cost tracking
- [M-COORDINATOR-ALWAYS-ON-DAEMON](../../planned/v0_7_0/m-coordinator-always-on-daemon.md) - Coordinator architecture

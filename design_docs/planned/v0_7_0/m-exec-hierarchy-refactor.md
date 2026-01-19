# M-EXEC-HIERARCHY-REFACTOR: Executive Hierarchy Graph Visualization

**Status:** PLANNED
**Version:** v0.7.0
**Priority:** HIGH
**Estimated Effort:** 2-3 days

## Problem Statement

The current Graph view in the Collaboration Hub uses Dagre TB (top-bottom) layout which:
1. Shows flat hierarchy - turns appear at equal levels under session
2. No cross-task relationships visible (handoff chains, session continuity)
3. Approval status not integrated into task nodes
4. Hard to see evolution and iteration patterns

## Goals

1. Replace the existing Graph view with a new radial/hub-spoke visualization
2. Show within-task hierarchy (coordinator → executor → turns → tools)
3. Show cross-task relationships (parent_task_id handoffs, session.id continuity)
4. Integrate approval status visually on task nodes
5. Use the new `/api/controlplane/span-hierarchy` endpoint

## Design

### Data Sources

**Primary:** New span hierarchy API
```
GET /api/controlplane/span-hierarchy?limit=100
```

**Secondary:** Coordinator tasks with approvals
```
GET /api/coordinator/pending  (pending approvals)
GET /api/coordinator/tasks    (task list with parent_task_id)
```

### Visualization Layers

**NOTE:** This visualization is GENERIC - not hardcoded to specific agents. Any task with `parent_task_id` creates a handoff edge. Any tasks sharing `session_id` show session continuity.

```
Layer 1: Cross-Task Overview (Radial Hub)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│         Generic Task Relationships                      │
│                                                         │
│              ┌─────┐                                    │
│              │TaskA│ (any agent)                        │
│              └──┬──┘                                    │
│      parent_task_id                                     │
│              ┌──▼──┐                                    │
│              │TaskB│ (any agent)  ⏳ pending approval   │
│              └──┬──┘                                    │
│      parent_task_id                                     │
│              ┌──▼──┐                                    │
│              │TaskC│ (any agent) ✅ approved            │
│              └─────┘                                    │
│                                                         │
│  Relationships derived from:                            │
│  • parent_task_id field (handoff chains)                │
│  • session_id field (session continuity)                │
│  • approval_requests table (approval status)            │
│                                                         │
└─────────────────────────────────────────────────────────┘

Layer 2: Within-Task Detail (Expanded Node)
┌─────────────────────────────────────────────────────────┐
│ Task: task-29404032 (agent_id from config)              │
│ Status: ✅ approved | Cost: $0.17 | Turns: 15           │
├─────────────────────────────────────────────────────────┤
│  coordinator.task.execute 99.9s                         │
│  └─ claude.execute 98.7s $0.17                          │
│     ├─ Turn #1 1.8s                                     │
│     │  └─ Read: parser.go                               │
│     ├─ Turn #2 454ms                                    │
│     ├─ Turn #3 1.8s                                     │
│     │  └─ Edit: parser.go                               │
│     ├─ Turn #4 577ms                                    │
│     └─ ... (collapsed)                                  │
└─────────────────────────────────────────────────────────┘
```

### Node Types

| Type | Shape | Color | Shows |
|------|-------|-------|-------|
| `task` | Rounded rect | Based on status | Title, agent, approval badge, cost |
| `handoff` | Dashed edge | Gray | parent_task_id relationship |
| `session` | Dotted edge | Blue | Shared session.id |
| `approval` | Badge overlay | Orange/Green/Red | pending/approved/rejected |

### Approval Integration

Approval status shown as overlay badge on task nodes:

```tsx
// Task node with approval badge
<div className={styles.taskNode}>
  <div className={styles.header}>
    <span className={styles.agentId}>{task.agent_id}</span>
    {approval && (
      <ApprovalBadge
        status={approval.status}
        iteration={approval.iteration}
      />
    )}
  </div>
  <div className={styles.title}>{task.title}</div>
  <div className={styles.metrics}>
    <span>${task.cost.toFixed(2)}</span>
    <span>{task.turns} turns</span>
  </div>
</div>
```

### Layout Algorithm

**Approach:** Force-directed with constraints (d3-force)

```typescript
const simulation = d3.forceSimulation(nodes)
  .force("link", d3.forceLink(edges)
    .id(d => d.id)
    .distance(d => d.type === 'handoff' ? 150 : 80))
  .force("charge", d3.forceManyBody().strength(-300))
  .force("center", d3.forceCenter(width/2, height/2))
  .force("radial", d3.forceRadial(
    d => d.depth * 120, // Radius by depth
    width/2,
    height/2
  ).strength(0.8));
```

**Alternative:** Keep ReactFlow but use custom layout function instead of Dagre

```typescript
// Custom radial layout for ReactFlow
function applyRadialLayout(nodes: Node[], edges: Edge[]): { nodes: Node[], edges: Edge[] } {
  const roots = nodes.filter(n => !n.data.parentId);
  let angleStep = (2 * Math.PI) / roots.length;

  roots.forEach((root, i) => {
    const angle = i * angleStep;
    const radius = 200;
    root.position = {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
    // Position children in concentric rings
    layoutChildren(root, angle, radius + 150);
  });

  return { nodes, edges };
}
```

## Implementation Plan

### Phase 1: API Enhancement (Backend)

**File:** `internal/server/handlers_controlplane.go`

1. Add endpoint for cross-task hierarchy:
```go
// GET /api/controlplane/task-hierarchy
// Returns tasks with parent_task_id relationships and approval status
func (s *Server) handleTaskHierarchy(w http.ResponseWriter, r *http.Request) {
    // Query tasks with joins to approvals
    // Include parent_task_id for handoff chains
    // Include session_id for session continuity
}
```

2. Enhance existing span-hierarchy with task context:
```go
// Add task_id, approval_status, parent_task_id to root spans
```

### Phase 2: TypeScript Types (Frontend)

**File:** `ui/src/features/controlplane/components/ExecHierarchy/types.ts`

```typescript
// New types for cross-task visualization
export interface TaskNode {
  id: string;
  title: string;
  agentId: string;
  parentTaskId?: string;      // Handoff chain
  sessionId?: string;         // Session continuity
  status: TaskStatus;
  approvalStatus?: ApprovalStatus;
  approvalIteration?: number;
  cost: number;
  turns: number;
  spans?: SpanHierarchyNode[];
}

export interface TaskEdge {
  source: string;
  target: string;
  type: 'handoff' | 'session';
}

export interface CrossTaskHierarchy {
  tasks: TaskNode[];
  edges: TaskEdge[];
  stats: {
    totalTasks: number;
    pendingApprovals: number;
    totalCost: number;
  };
}
```

### Phase 3: Data Fetching Hook

**File:** `ui/src/features/controlplane/hooks/useTaskHierarchy.ts`

```typescript
export function useTaskHierarchy() {
  const [data, setData] = useState<CrossTaskHierarchy | null>(null);

  // Fetch task hierarchy
  const { data: tasks } = useQuery('/api/controlplane/task-hierarchy');

  // Fetch pending approvals
  const { data: approvals } = useApprovals({ status: 'pending' });

  // Merge approval status into task nodes
  const mergedData = useMemo(() => {
    return mergeTasks WithApprovals(tasks, approvals);
  }, [tasks, approvals]);

  return mergedData;
}
```

### Phase 4: Replace Graph Component

**File:** `ui/src/features/controlplane/components/ExecHierarchy/ExecHierarchyGraph.tsx`

1. Replace Dagre layout with radial layout
2. Add task-level nodes with approval badges
3. Add handoff edges (dashed lines)
4. Add session edges (dotted lines)
5. Keep expand/collapse for span details within tasks

### Phase 5: Visual Polish

1. Approval badge component with iteration indicator
2. Edge styling (handoff vs session)
3. Hover states showing relationship details
4. Click to expand task → show span hierarchy
5. Filter controls (by agent, by status, by approval state)

## API Changes

### New Endpoint: Task Hierarchy

```
GET /api/controlplane/task-hierarchy
```

**Response:**
```json
{
  "tasks": [
    {
      "id": "task-29404032",
      "title": "Fix parser bug",
      "agent_id": "agent-from-config",      // Dynamic from ~/.ailang/config.yaml
      "parent_task_id": "task-50b9518d",    // Links to parent (creates handoff edge)
      "session_id": "session-abc123",       // Links tasks with shared context
      "status": "pending_approval",
      "approval": {                         // From approval_requests table
        "id": "apr-123",
        "status": "pending",
        "type": "merge_handoff",
        "iteration": 1
      },
      "metrics": {
        "cost": 0.17,
        "turns": 15,
        "duration_ms": 99000
      }
    }
  ],
  "edges": [
    {
      "source": "task-50b9518d",
      "target": "task-29404032",
      "type": "handoff"                     // Derived from parent_task_id relationship
    },
    {
      "source": "task-abc123",
      "target": "task-def456",
      "type": "session"                     // Derived from shared session_id
    }
  ]
}
```

## Testing

1. Unit tests for radial layout algorithm
2. Integration tests for task-hierarchy endpoint
3. E2E test for approval badge display
4. Visual regression tests for graph layout

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Performance with many tasks | High | Limit to recent 50 tasks, pagination |
| Layout collisions | Medium | Force simulation with collision detection |
| Breaking existing users | Medium | Keep view mode toggle, can switch back |

## Success Metrics

- [ ] Cross-task handoff chains visible
- [ ] Approval status on task nodes
- [ ] Session continuity edges shown
- [ ] Expand task to see span hierarchy
- [ ] Performance <100ms for 50 tasks

## Files to Modify

| File | Changes |
|------|---------|
| `internal/server/handlers_controlplane.go` | Add task-hierarchy endpoint |
| `internal/server/server.go` | Register new route |
| `ui/src/features/controlplane/components/ExecHierarchy/types.ts` | Add TaskNode, TaskEdge types |
| `ui/src/features/controlplane/components/ExecHierarchy/ExecHierarchyGraph.tsx` | Replace Dagre with radial layout |
| `ui/src/features/controlplane/hooks/useTaskHierarchy.ts` | New data fetching hook |
| `ui/src/features/approvals/components/ApprovalBadge.tsx` | Reusable badge for graph nodes |

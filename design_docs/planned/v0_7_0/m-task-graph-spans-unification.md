# M-TASK-GRAPH-SPANS-UNIFICATION: Fix TaskHierarchyGraph Filtering

**Status:** Planned
**Priority:** High
**Attempts:** 12+ failed attempts
**Root Cause Identified:** YES - architectural mismatch

## Problem Statement

The TaskHierarchyGraph (execution graph view) does NOT filter when an event is selected from the Event Queue sidebar. All other views (tree, timeline, chat, waterfall, span list) filter correctly. The graph always shows ALL tasks regardless of selection.

## The Fundamental Insight

**Other views work because they use the same `spans` prop that's already loaded.**

**TaskHierarchyGraph doesn't work because it makes SEPARATE API calls to fetch coordinator tasks.**

### Data Flow - Working Views (Tree, Timeline, Chat, Waterfall)

```
1. User selects event in EventQueue
2. ExecHierarchy receives selectedEventTraceId
3. useTraceData.fetchSpansForTrace(lookupId, 'auto') fetches spans
4. spans prop passed to TreeView/Timeline/Chat/Waterfall
5. View renders ONLY those spans ✓
```

### Data Flow - Broken View (TaskHierarchyGraph)

```
1. User selects event in EventQueue
2. ExecHierarchy receives selectedEventTraceId
3. Attempts to pass filter params (filterTaskId, spanTaskIds, filterTraceId)
4. TaskHierarchyGraph calls useTaskHierarchy hook
5. useTaskHierarchy fetches from /api/controlplane/task-hierarchy
6. API queries coordinator.db (different database!)
7. Filter doesn't match → returns all tasks or nothing ✗
```

## Why The 12+ Attempts Failed

### Attempt Pattern: Keep adding more filter types

| Filter Type | Problem |
|-------------|---------|
| `filterTaskId` | selectedNodeId is often a trace_id/UUID, not a task_id |
| `spanTaskIds` | Spans may not have `task.id` attribute set (especially evals) |
| `filterTraceId` | trace_id doesn't link to coordinator.db tasks |

### Root Cause: Cross-Database ID Mismatch

```
OBSERVATORY.DB (spans)              COORDINATOR.DB (tasks)
═══════════════                     ══════════════
spans.trace_id (32-char hex)  ───X──→ No link!
spans.task_id (varies)        ───?──→ tasks.id (task-<8char>)
```

The `selectedNodeId` from EventQueue can be:
- Full UUID (Claude Code session) → NOT in coordinator.db
- `eval-<timestamp>` → NOT in coordinator.db
- `task-<8char>` → YES, but only for coordinator-executed tasks
- 32-char hex trace_id → NOT linkable to coordinator.db

**Only ~10% of events have matching coordinator tasks!**

## The Correct Solution

**Stop fighting the data model. Use the same `spans` prop like other views.**

### Option A: Transform Spans to Graph (Recommended)

Add a new function that transforms the already-loaded `spans` array into ReactFlow graph data:

```typescript
// In TaskHierarchyGraph.tsx
function buildGraphFromSpans(
  spans: Span[],
  selectedNodeId?: string
): { rfNodes: Node[]; rfEdges: Edge[] } {
  // Transform spans hierarchically into graph nodes
  // Use parent-child relationships from span.children
  // No API call needed - data is already loaded!
}

// Props change
interface TaskHierarchyGraphProps {
  spans?: Span[];           // Same prop as other views!
  selectedNodeId?: string;
  onNodeClick?: (node: HierarchyNode) => void;
  // Remove: filterTaskId, spanTaskIds, filterTraceId
}
```

### Option B: Hybrid Mode

Keep coordinator task fetching for unselected state, switch to span-based rendering when event selected:

```typescript
const TaskHierarchyGraph = ({ spans, selectedNodeId, ...props }) => {
  // If spans are provided (event selected), render from spans
  if (spans && spans.length > 0) {
    return <GraphFromSpans spans={spans} />;
  }

  // Otherwise, fetch all tasks from coordinator (overview mode)
  const { data } = useTaskHierarchy({ limit: 100 });
  return <GraphFromTasks data={data} />;
};
```

### Option C: Unified Backend Query

Create a new backend endpoint that returns spans in task-hierarchy format, so the frontend doesn't need to transform:

```
GET /api/observatory/spans/as-hierarchy?trace_id=xxx
Returns: { tasks: [...], edges: [...], stats: {...} }
```

## Implementation Plan for Option A

### Step 1: Create span-to-graph transformer

```typescript
// New file: buildGraphFromSpans.ts
export function buildGraphFromSpans(spans: Span[]): GraphData {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  function addSpanAsNode(span: Span, parentId?: string) {
    const nodeType = getNodeTypeFromSpan(span);
    nodes.push({
      id: span.id,
      type: nodeType,  // 'coordinator', 'executor', 'turn', 'tool'
      data: {
        name: span.display_name || span.name,
        duration_ms: span.durationMs,
        cost: span.cost_usd,
        tokens_in: span.tokens_in,
        tokens_out: span.tokens_out,
        status: span.status,
      },
      position: { x: 0, y: 0 },
    });

    if (parentId) {
      edges.push({
        id: `e-${parentId}-${span.id}`,
        source: parentId,
        target: span.id,
      });
    }

    // Recurse into children
    span.children?.forEach(child => addSpanAsNode(child, span.id));
  }

  spans.forEach(rootSpan => addSpanAsNode(rootSpan));

  return { nodes: applyDagreLayout(nodes, edges), edges };
}
```

### Step 2: Update TaskHierarchyGraph props

```diff
export interface TaskHierarchyGraphProps {
+ spans?: Span[];           // Add: same prop as other views
  selectedNodeId?: string | null;
  onNodeClick?: (task: TaskHierarchyNode) => void;
  isExpanded?: boolean;
  recenterTrigger?: number;
  workspace?: string;
  provider?: string;
- filterTaskId?: string | null;      // Remove
- spanTaskIds?: string[];            // Remove
- filterTraceId?: string | null;     // Remove
}
```

### Step 3: Update ExecHierarchy.tsx

```diff
<TaskHierarchyGraph
+ spans={spans}            // Pass the already-loaded spans!
  selectedNodeId={selectedNodeId}
  onNodeClick={handleGraphNodeClick}
  isExpanded={isFullscreen}
  recenterTrigger={recenterTrigger}
  workspace={workspaceFilter || undefined}
  provider={providerFilter || undefined}
- filterTaskId={selectedNodeId?.startsWith('task-') ? selectedNodeId : undefined}
- spanTaskIds={spanTaskIds.length > 0 ? spanTaskIds : undefined}
- filterTraceId={...}
/>
```

### Step 4: Conditional rendering in TaskHierarchyGraph

```typescript
export const TaskHierarchyGraph: React.FC<Props> = ({ spans, ...props }) => {
  // If spans provided, render from spans (filtered mode)
  if (spans && spans.length > 0) {
    const graphData = useMemo(
      () => buildGraphFromSpans(spans),
      [spans]
    );
    return <GraphRenderer data={graphData} {...props} />;
  }

  // No spans = no selection = fetch all tasks for overview
  const { data, loading, error } = useTaskHierarchy({ limit: 100 });
  return <GraphRenderer data={buildGraphFromTasks(data)} loading={loading} {...props} />;
};
```

## Files to Modify

| File | Changes |
|------|---------|
| `TaskHierarchyGraph.tsx` | Add spans prop, add buildGraphFromSpans, conditional rendering |
| `ExecHierarchy.tsx` | Pass spans prop to TaskHierarchyGraph |
| `useTaskHierarchy.ts` | No changes needed (used for unfiltered mode) |
| `handlers_controlplane.go` | Can revert filter additions (not needed) |

## Key Learnings

1. **Same data source = same behavior.** Other views work because they all use `spans`. TaskHierarchyGraph was special-cased to use a different API.

2. **Cross-database joins are hard.** Trying to link observatory.db spans to coordinator.db tasks via various ID formats is error-prone and incomplete.

3. **ID formats vary wildly.** `task-<8char>`, `eval-<timestamp>`, full UUIDs, trace_ids - too many formats to handle with string matching.

4. **The spans already have hierarchy.** The `spans` prop has `children[]` arrays - no need to fetch hierarchy separately.

5. **Don't fight the architecture.** When 4 views work one way and 1 doesn't, make the outlier conform, don't add complexity.

## Success Criteria

After implementation:
1. Select event in EventQueue → Graph shows ONLY spans from that event
2. No event selected → Graph shows all coordinator tasks (overview)
3. Works for ALL event types: evals, Claude sessions, Gemini sessions, coordinator tasks
4. No new API endpoints or backend changes needed

## Estimated Effort

- Implementation: 2-3 hours
- Testing: 1 hour
- Total: ~4 hours (vs 12+ hours of failed attempts)

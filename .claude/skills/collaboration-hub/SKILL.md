---
name: collaboration-hub
description: Develop and modify the AILANG Collaboration Hub UI. Use when user asks to add features to the monitoring dashboard, modify the approval queue, update the message center, or make changes to the React frontend.
---

# Collaboration Hub Developer

Build and modify the AILANG Collaboration Hub - a React-based UI for multi-agent coordination, task execution monitoring, and observability.

## Quick Start

**Starting services (recommended):**
```bash
make services-start         # Start server + coordinator together
make services-status        # Check both services
make services-stop          # Stop both services
make services-restart       # Rebuild and restart both
```

**Starting server only:**
```bash
ailang serve                # Start on default port 1957
ailang serve --port 8080    # Use custom port
```

**After making UI changes:**
```bash
make ui-deploy              # Build, clean old assets, copy to server
make services-restart       # Rebuild server and restart
```

**Key URLs:**
- **UI**: http://localhost:1957/
- **WebSocket**: ws://localhost:1957/ws
- **REST API**: http://localhost:1957/api/
- **Health**: http://localhost:1957/health

## When to Use This Skill

Use this skill when the user asks to:
- Add features to the Control Plane dashboard
- Modify the Observatory or task hierarchy views
- Update the message center or approval queue
- Add components to the trace waterfall or event queue
- Style or theme any UI component
- Add new tabs, pages, or features

## Architecture Overview

**Backend (Go):**
- `internal/server/server.go` - HTTP server, REST API, static file serving
- `internal/server/handlers_controlplane.go` - Control plane API endpoints
- `internal/server/handlers_coordinator.go` - Coordinator event streaming
- `internal/observatory/` - Telemetry and span storage

**Frontend (React + TypeScript + Vite):**
```
ui/
├── src/
│   ├── App.tsx                    # Main app, navigation, routing
│   ├── App.css                    # Global styles
│   ├── main.tsx                   # React entry point
│   ├── types/                     # TypeScript interfaces
│   ├── hooks/                     # Shared React hooks
│   ├── components/                # Shared components
│   │   ├── common/                # Breadcrumb, buttons, etc.
│   │   ├── metrics/               # MetricsCard, TrendsChart
│   │   └── ConnectionStatus.tsx   # WebSocket status indicator
│   └── features/                  # Feature-based organization
│       ├── controlplane/          # Control Plane v4 (main view)
│       │   ├── ControlPlane.tsx         # Main container
│       │   ├── ControlPlane.module.css  # Scoped styles (2500+ lines)
│       │   ├── components/              # Modular components
│       │   │   ├── MessageQueue.tsx     # Event queue with pagination
│       │   │   ├── EventDetail.tsx      # Inline trace viewer
│       │   │   ├── TraceWaterfall.tsx   # Span timeline visualization
│       │   │   ├── GlobalStats.tsx      # Dashboard stats cards
│       │   │   ├── AgentTopology.tsx    # Agent relationship graph
│       │   │   ├── ActivityHeatmap.tsx  # Time-based activity grid
│       │   │   └── CommandBar.tsx       # Search and filters
│       │   └── hooks/                   # Feature-specific hooks
│       │       ├── useEventQueue.ts     # Event fetching + WebSocket
│       │       └── useTraceData.ts      # Span hierarchy building
│       ├── observatory/           # Task and span exploration
│       │   ├── Observatory.tsx          # Main container with tabs
│       │   ├── TaskHierarchy.tsx        # Task tree view
│       │   └── components/              # Analytics components
│       ├── tasks/                 # Task execution monitoring
│       │   ├── RunningTasks.tsx         # Active task grid
│       │   └── TaskExecution/           # Execution panel components
│       ├── agents/                # Agent management
│       │   ├── Overview/                # All agents view
│       │   ├── AgentView/               # Single agent detail
│       │   └── HierarchyTree/           # Agent hierarchy
│       ├── messaging/             # Message center
│       │   └── MessageCenter/           # Thread-based messaging
│       └── approvals/             # Approval workflow
│           ├── ApprovalQueue/           # Pending approvals
│           └── ApprovalHistory/         # Review history
├── dist/                          # Built assets (after build)
└── package.json
```

**Database:**
- SQLite at `~/.ailang/state/collaboration.db`
- Tables: threads, messages, approvals, workspaces, tasks, spans, etc.

## Key Features

### Control Plane v4 (Default View)

The main dashboard with:
- **Event Queue** - Real-time events with pagination (10 per page)
- **Trace Waterfall** - Hierarchical span visualization with zoom (1x-16x)
- **Global Stats** - Task counts, costs, success rates
- **Agent Topology** - Service relationship graph
- **Activity Heatmap** - Time-based activity visualization

Key files:
- `features/controlplane/ControlPlane.tsx` - Main container
- `features/controlplane/components/` - All modular components
- `features/controlplane/hooks/` - Data fetching hooks

### Observatory

Task and span exploration:
- Task hierarchy with parent/child relationships
- Analytics components for provider breakdown
- Search and filter capabilities

### Task Execution

Real-time task monitoring:
- Streaming logs with output markers
- Resource metrics (tokens, cost)
- Status badges and progress

## CSS Architecture

**CSS Modules** are used for scoped styles:
```tsx
import styles from '../ControlPlane.module.css';

<div className={styles.messageQueue}>
  <div className={styles.queueHeader}>...</div>
</div>
```

**CSS Variables** (defined in `.controlPlane` class):
```css
/* Colors */
--bg-deep: #0a0c10;
--bg-base: #0d1117;
--bg-surface: #161b22;
--bg-elevated: #1c2128;
--primary: #25c2a0;
--amber: #f59e0b;
--success: #10b981;
--danger: #ef4444;

/* Text */
--text-primary: #e6edf3;
--text-secondary: #8b949e;
--text-tertiary: #6e7681;

/* Spacing */
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;

/* Typography */
--font-mono: 'JetBrains Mono', monospace;
--font-display: 'Space Grotesk', sans-serif;
```

**Light Theme** - `.controlPlaneLight` class with Sunholo brand colors.

## REST API Endpoints

**Control Plane:**
- `GET /api/controlplane/stats` - Global metrics
- `GET /api/controlplane/heatmap` - Activity by hour/day
- `GET /api/controlplane/topology` - Service relationships
- `GET /api/controlplane/stats/breakdown?by=provider` - Provider breakdown
- `GET /api/controlplane/exec-hierarchy` - Execution hierarchy for tasks

**Observatory:**
- `GET /api/observatory/workspaces` - List workspaces
- `GET /api/observatory/tasks` - List tasks
- `GET /api/observatory/tasks/{id}` - Get single task
- `GET /api/observatory/tasks/{id}/hierarchy` - **Full task hierarchy with all spans** (recommended)
- `GET /api/observatory/tasks/{taskId}/agents` - List agent assignments for task
- `GET /api/observatory/spans?task_id=X` - Spans with task_id set (limited)
- `GET /api/observatory/spans?trace_id=X` - All spans in a trace
- `GET /api/observatory/traces` - List traces
- `GET /api/observatory/metrics/summary` - Aggregate metrics
- `GET /api/observatory/metrics/providers` - Provider comparison

**Inbox/Messages:**
- `GET /api/inbox` - List messages (includes counts by status)
- `GET /api/inbox/{id}` - Get single message
- `POST /api/inbox` - Send message
- `PUT /api/inbox/{id}` - Update message (ack/unack)
- `POST /api/inbox/ack-all` - Mark all messages as read
- `POST /api/inbox/cleanup` - Clean up old messages

**Approvals:**
- `GET /api/approvals?status=pending` - Pending approvals
- `POST /api/approvals/:id/approve` - Approve with notes
- `POST /api/approvals/:id/reject` - Reject with notes

**Coordinator:**
- `GET /api/coordinator/events` - SSE stream for task events
- `POST /api/coordinator/events` - Receive coordinator events
- `GET /api/coordinator/status` - Coordinator daemon status
- `GET /api/coordinator/running` - Currently running tasks
- `GET /api/coordinator/pending` - Pending approvals

**WebSocket:**
- `ws://localhost:1957/ws` - Real-time events
  - `inbox_message` - New inbox message
  - `task_stream_event` - Live task execution events
  - `task_update` - Task status change
  - `span_created` - New span ingested
- `ws://localhost:1957/ws/observatory` - Observatory-specific events

## Hierarchy & Trace Correlation

### Data Model Overview

The system uses a hierarchical data model:

```
WORKSPACE
  └── TASK (created from message)
        └── AGENT_ASSIGNMENT (which executor is running)
              └── SPANS (telemetry data)
                    └── TRACES (span groups)
```

**Key relationships:**
- **Message ID → Task ID**: Tasks are created with ID format `task-{first8chars of message_id}`
- **Task → Traces**: A task may have multiple traces (e.g., coordinator trace + execution trace)
- **Trace → Spans**: A trace contains all spans with the same `trace_id`

### Task ID Correlation

When clicking an event in the UI, we need to find its traces:

```typescript
// In ControlPlane.tsx handleEventClick:
// Priority order for finding task ID:
// 1. metadata.task_id (direct span attribute)
// 2. metadata.parent_task_id (from coordinator)
// 3. metadata.correlation_id (message correlation)
// 4. Construct task-{first8chars} from event.id
let lookupId = metadata?.task_id || metadata?.parent_task_id || metadata?.correlation_id;
if (!lookupId && event.id) {
  lookupId = `task-${event.id.substring(0, 8)}`;
}
```

### Hierarchy API vs Direct Spans API

**Problem**: Child spans (claude.execute, exec.turn, etc.) don't have `task_id` set - only parent spans do.

**Solution**: Use the hierarchy endpoint which expands to include ALL spans in linked traces:

```typescript
// In useTraceData.ts fetchByTaskId:
// 1. Call hierarchy endpoint (does proper trace expansion)
const response = await fetch(`/api/observatory/tasks/${tid}/hierarchy`);
const hierarchy = await response.json();

// 2. Extract all spans from nested structure
const allSpans = [];
for (const agent of hierarchy.agents) {
  for (const trace of agent.traces) {
    for (const spanNode of trace.spans) {
      allSpans.push(spanNode.span);
    }
  }
}
```

The hierarchy endpoint (`/api/observatory/tasks/{id}/hierarchy`):
1. Gets spans by `task_id` first
2. Collects all unique `trace_id`s from those spans
3. Fetches ALL spans in those traces (including children without task_id)
4. Applies timestamp correlation for proper nesting

### Timestamp Correlation (Virtual Re-parenting)

Claude Code doesn't propagate TRACEPARENT to subprocess environments. This means child spans from `ailang check` are parented to `claude.execute` instead of `exec.tool_use`.

The hierarchy API applies **virtual re-parenting** at query time:

```
Before (DB structure):
claude.execute
  ├── exec.turn
  ├── exec.tool_use (Bash: ailang check)
  └── ailang.check ← sibling, wrong!

After (virtual hierarchy):
claude.execute
  └── exec.turn
        └── exec.tool_use (Bash: ailang check)
              └── ailang.check ← correctly nested!
```

Implementation in `internal/observatory/hierarchy.go`:
- `applyTimestampCorrelation()` - Re-parents ailang.* spans under exec.tool_use
- Uses span timestamps to determine which tool invoked which child process

### Backend Files

| File | Purpose |
|------|---------|
| `internal/observatory/hierarchy.go` | Task hierarchy building, timestamp correlation |
| `internal/observatory/api.go` | REST API handlers for observatory |
| `internal/observatory/backend.go` | Backend interface for span storage |
| `internal/observatory/store.go` | SQLite implementation |
| `internal/server/handlers_controlplane.go` | Control plane API endpoints |
| `internal/server/handlers_inbox.go` | Inbox message API endpoints |

### Frontend Files

| File | Purpose |
|------|---------|
| `ui/src/features/controlplane/ControlPlane.tsx` | Main container, event click handling |
| `ui/src/features/controlplane/hooks/useTraceData.ts` | Fetches spans via hierarchy API |
| `ui/src/features/controlplane/hooks/useEventQueue.ts` | Message fetching + WebSocket |
| `ui/src/features/controlplane/components/TraceWaterfall.tsx` | Span visualization |
| `ui/src/features/controlplane/components/EventDetail.tsx` | Event detail panel |

### Example Hierarchy Response

```json
{
  "task": {
    "id": "task-50b9518d",
    "title": "Dashboard Trace Test",
    "status": "pending"
  },
  "agents": [{
    "agent": {
      "id": "aa_3a4daa5f8e42ecc8",
      "agent_id": "hierarchy-test",
      "provider": "claude"
    },
    "traces": [
      {
        "trace_id": "79c9736d...",
        "spans": [{"span": {"name": "coordinator.task.execute"}}],
        "summary": {"span_count": 1}
      },
      {
        "trace_id": "f87170ae...",
        "root_span": {
          "span": {"name": "ailang.exec"},
          "children": [{
            "span": {"name": "claude.execute"},
            "children": [
              {"span": {"name": "exec.turn"}},
              {"span": {"name": "exec.tool_use"}},
              {"span": {"name": "ailang.check", "children": [...]}}
            ]
          }]
        },
        "spans": [...], // Flat list of all 15 spans
        "summary": {"span_count": 15}
      }
    ]
  }]
}
```

## Development Workflow

### 1. Make Changes
Edit files in `ui/src/`. Key patterns:

**Adding a component:**
```tsx
// ui/src/features/controlplane/components/NewComponent.tsx
import React from 'react';
import styles from '../ControlPlane.module.css';

export interface NewComponentProps {
  data: SomeType[];
}

export const NewComponent: React.FC<NewComponentProps> = ({ data }) => {
  return (
    <div className={styles.newComponent}>
      {/* content */}
    </div>
  );
};

export default NewComponent;
```

**Adding styles:**
```css
/* Add to ControlPlane.module.css */
.newComponent {
  background: var(--bg-surface);
  padding: var(--space-4);
  border-radius: var(--radius-md);
}
```

**Adding a hook:**
```tsx
// ui/src/features/controlplane/hooks/useNewData.ts
import { useState, useEffect, useCallback } from 'react';

export function useNewData(options = {}) {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    const response = await fetch('/api/new-endpoint');
    const result = await response.json();
    setData(result);
    setLoading(false);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, refetch: fetchData };
}
```

### 2. Build and Deploy
```bash
make ui-deploy              # Build + clean + copy (recommended)
```

Or manually:
```bash
cd ui && npm run build      # Build React app
rm -rf internal/server/dist/assets/*  # Clean old assets
cp -r ui/dist/* internal/server/dist/ # Copy to server
```

### 3. Test
```bash
make services-restart       # Restart server with new build
# Open http://localhost:1957/
```

### 4. Iterate
- Check browser console for errors
- Use React DevTools for component inspection
- Check Network tab for API calls

## Common Tasks

### Adding Pagination to a List

See `MessageQueue.tsx` for example:
```tsx
const [currentPage, setCurrentPage] = useState(0);
const pageSize = 10;

const totalPages = Math.ceil(items.length / pageSize);
const paginatedItems = items.slice(
  currentPage * pageSize,
  (currentPage + 1) * pageSize
);

// Render pagination controls
{totalPages > 1 && (
  <div className={styles.pagination}>
    <button onClick={() => setCurrentPage(p => p - 1)} disabled={currentPage === 0}>←</button>
    <span>{currentPage + 1} / {totalPages}</span>
    <button onClick={() => setCurrentPage(p => p + 1)} disabled={currentPage >= totalPages - 1}>→</button>
  </div>
)}
```

### Adding Zoom to Visualizations

See `TraceWaterfall.tsx` for example:
```tsx
const [zoomLevel, setZoomLevel] = useState(1);
const zoomIn = () => setZoomLevel(z => Math.min(z * 2, 16));
const zoomOut = () => setZoomLevel(z => Math.max(z / 2, 1));

// Apply zoom to container width
<div style={{ width: `${100 * zoomLevel}%`, overflowX: zoomLevel > 1 ? 'auto' : 'hidden' }}>
  {/* content */}
</div>
```

### Building Hierarchical Data

**For spans**: Use the hierarchy API which handles trace expansion and timestamp correlation:

```tsx
// In useTraceData.ts - fetch via hierarchy endpoint
const fetchByTaskId = async (tid: string): Promise<RawSpan[]> => {
  const response = await fetch(`/api/observatory/tasks/${tid}/hierarchy`);
  if (!response.ok) {
    // Fallback to direct spans query
    const fallback = await fetch(`/api/observatory/spans?task_id=${tid}&limit=100`);
    return await fallback.json() || [];
  }

  const hierarchy = await response.json();
  const allSpans: RawSpan[] = [];

  // Extract spans from nested hierarchy response
  if (hierarchy?.agents) {
    for (const agent of hierarchy.agents) {
      for (const trace of agent?.traces || []) {
        for (const spanNode of trace?.spans || []) {
          if (spanNode?.span) allSpans.push(spanNode.span);
        }
      }
    }
  }
  return allSpans;
};
```

**For client-side tree building** (from flat span list):

```tsx
function buildSpanHierarchy(rawSpans: RawSpan[]): Span[] {
  const spanMap = new Map<string, Span>();
  const childMap = new Map<string, string[]>();

  // First pass: create spans and track parent relationships
  rawSpans.forEach(raw => {
    spanMap.set(raw.id, {
      id: raw.id,
      name: raw.name,
      startMs: new Date(raw.start_time).getTime(),
      durationMs: raw.duration_ms,
      children: [],
    });
    if (raw.parent_span_id) {
      const children = childMap.get(raw.parent_span_id) || [];
      children.push(raw.id);
      childMap.set(raw.parent_span_id, children);
    }
  });

  // Second pass: build tree
  const roots: Span[] = [];
  rawSpans.forEach(raw => {
    const span = spanMap.get(raw.id)!;
    span.children = (childMap.get(raw.id) || [])
      .map(id => spanMap.get(id)!)
      .filter(Boolean);
    if (!raw.parent_span_id) roots.push(span);
  });

  return roots.sort((a, b) => a.startMs - b.startMs);
}
```

## Troubleshooting

### Hierarchy Issues

**Clicking event shows no spans / empty trace:**

1. Check task exists in observatory:
   ```bash
   sqlite3 ~/.ailang/state/observatory.db "SELECT * FROM tasks WHERE id = 'task-XXXXXXXX'"
   ```

2. Check spans exist for task:
   ```bash
   sqlite3 ~/.ailang/state/observatory.db "SELECT name, trace_id FROM spans WHERE task_id = 'task-XXXXXXXX'"
   ```

3. Test hierarchy API directly:
   ```bash
   curl -s "http://localhost:1957/api/observatory/tasks/task-XXXXXXXX/hierarchy" | jq '.agents[0].traces | length'
   ```

**Only showing parent spans, missing children (claude.execute, exec.turn, etc.):**

This usually means the old API was being used. The hierarchy endpoint expands traces:

```bash
# Wrong: Direct spans query (only returns spans with task_id set)
curl "http://localhost:1957/api/observatory/spans?task_id=task-XXX"

# Correct: Hierarchy API (returns ALL spans in linked traces)
curl "http://localhost:1957/api/observatory/tasks/task-XXX/hierarchy"
```

**Spans in wrong hierarchy (ailang.check sibling of exec.tool_use instead of child):**

The timestamp correlation should fix this at query time. Check hierarchy.go:
- `applyTimestampCorrelation()` should re-parent ailang.* spans under exec.tool_use
- Verify span timestamps overlap correctly

**Task ID format mismatch:**

Tasks are created with format `task-{first 8 chars of message UUID}`:
```bash
# Message ID: 50b9518d-53f0-4dd7-893e-779ac90672b6
# Task ID:    task-50b9518d
```

**Clearing old data for fresh start:**
```bash
sqlite3 ~/.ailang/state/observatory.db "DELETE FROM spans; DELETE FROM tasks; DELETE FROM workspaces; DELETE FROM agent_assignments;"
```

### General Issues

**UI changes not appearing:**
```bash
make ui-deploy              # Clean build and deploy
# Hard refresh: Cmd+Shift+R (Mac) or Ctrl+Shift+R (Windows/Linux)
```

**Old assets accumulating:**
```bash
rm -rf internal/server/dist/assets/*
make ui-deploy
```

**Server won't start:**
```bash
lsof -i :1957               # Check if port in use
make services-stop          # Stop any running services
make services-start         # Fresh start
```

**WebSocket not connecting:**
- Check browser console for CORS errors
- Verify server is running: `curl http://localhost:1957/health`
- Check that WebSocket URL matches server port

**TypeScript errors:**
```bash
cd ui && npm run build      # Shows compilation errors
```

**API returning 404:**
- Check endpoint exists in `internal/server/`
- Verify route is registered in `setupRoutes()`
- Check for typos in fetch URL

## File Size Guidelines

Keep files maintainable:
- Components: 100-300 lines ideal, 500 max
- CSS modules: Split if > 800 lines
- Hooks: 50-150 lines ideal

The `ControlPlane.module.css` is currently ~2500 lines - consider splitting into component-specific CSS modules if it grows further.

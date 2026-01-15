---
name: collaboration-hub
description: Develop and modify the AILANG Collaboration Hub UI. Use when user asks to add features to the monitoring dashboard, modify the approval queue, update the message center, or make changes to the React frontend.
---

# Collaboration Hub Developer

Build and modify the AILANG Collaboration Hub - a React-based UI for multi-agent coordination, task execution monitoring, and observability.

## Quick Start

**Starting services:**
```bash
make services-start         # Start server + coordinator
make services-status        # Check both services
make services-stop          # Stop both
make services-restart       # Rebuild and restart
```

**After UI changes:**
```bash
make ui-deploy              # Build, clean, copy to server
make services-restart       # Restart with new build
```

**Key URLs:**
- **UI**: http://localhost:1957/
- **REST API**: http://localhost:1957/api/
- **WebSocket**: ws://localhost:1957/ws

## When to Use This Skill

Use when user asks to:
- Add features to Control Plane dashboard
- Modify Observatory or task hierarchy views
- Update message center or approval queue
- Add components to trace waterfall or event queue
- Style or theme UI components
- Diagnose hierarchy or span issues

## Architecture Overview

**Backend (Go):**
- `internal/server/server.go` - HTTP server, static files
- `internal/server/handlers_controlplane.go` - Control plane API
- `internal/observatory/` - Telemetry and span storage
- `internal/observatory/hierarchy.go` - Hierarchy building

**Frontend (React + TypeScript + Vite):**
```
ui/src/
├── App.tsx                     # Main app, navigation
├── features/
│   ├── controlplane/           # Control Plane v4 (main view)
│   │   ├── ControlPlane.tsx    # Main container
│   │   ├── components/         # MessageQueue, TraceWaterfall, etc.
│   │   └── hooks/              # useEventQueue, useTraceData
│   ├── observatory/            # Task and span exploration
│   ├── tasks/                  # Task execution monitoring
│   ├── agents/                 # Agent management
│   ├── messaging/              # Message center
│   └── approvals/              # Approval workflow
└── components/                 # Shared components
```

**Database:** SQLite at `~/.ailang/state/observatory.db`

## Available Scripts

### `scripts/diagnose.sh <command>`

Diagnose Collaboration Hub and Observatory issues.

**Commands:**
```bash
.claude/skills/collaboration-hub/scripts/diagnose.sh health          # Server + DB check
.claude/skills/collaboration-hub/scripts/diagnose.sh task <id>       # Task details
.claude/skills/collaboration-hub/scripts/diagnose.sh spans <task_id> # Span hierarchy
.claude/skills/collaboration-hub/scripts/diagnose.sh orphans         # Find orphan spans
.claude/skills/collaboration-hub/scripts/diagnose.sh costs           # Cost breakdown
.claude/skills/collaboration-hub/scripts/diagnose.sh recent          # Recent activity
```

## Key API Endpoints

**Control Plane:**
- `GET /api/controlplane/stats` - Unified stats
- `GET /api/controlplane/exec-hierarchy` - Execution hierarchy

**Observatory:**
- `GET /api/observatory/tasks/{id}/hierarchy` - **Full task hierarchy (RECOMMENDED)**
- `GET /api/observatory/spans?task_id=X` - Spans by task
- `GET /api/observatory/traces/{id}` - Trace with all spans

**Approvals (v0.6.5+):**
- `GET /api/coordinator/pending` - List pending approvals
- `POST /api/coordinator/approve/{id}` - Approve (merge + optional handoff)
- `POST /api/coordinator/reject/{id}` - Reject with feedback

**For complete API reference:** See [resources/rest_api_reference.md](resources/rest_api_reference.md)

## Approval Workflow (v0.6.5+)

The approval system uses **unified approvals** where merge and handoff are combined:

| Approval Type | UI Display | On Approve |
|--------------|------------|------------|
| `merge` | "Approve" button | Merges code to dev |
| `merge_handoff` | "Approve & Handoff" button | Merges code AND triggers next agent |

**API response includes:**
```json
{
  "id": "apr-123",
  "type": "merge_handoff",
  "context_json": "{\"handoff_targets\":[\"sprint-planner\"],\"session_id\":\"...\"}"
}
```

**UI considerations:**
- Show handoff targets when `type === "merge_handoff"`
- Update button text to indicate handoff will occur
- Show session continuity info (agent will resume with context)

## Hierarchy & Trace Correlation

The hierarchy API applies three levels of virtual re-parenting:

1. **Timestamp Correlation** - Fixes ailang.* spans under exec.tool_use
2. **Cross-Trace Merging** - Links spans across different trace IDs
3. **Session-Based Merging** - Links orphan traces by session.id

**Why use hierarchy API?** Child spans don't have `task_id` set. The hierarchy endpoint expands traces to include ALL spans.

**For detailed algorithm:** See [resources/hierarchy_algorithm.md](resources/hierarchy_algorithm.md)

## Development Workflow

### Adding a Component

```tsx
// ui/src/features/controlplane/components/NewComponent.tsx
import styles from '../ControlPlane.module.css';

export const NewComponent: React.FC<Props> = ({ data }) => {
  return <div className={styles.newComponent}>{/* content */}</div>;
};
```

### Adding Styles

```css
/* Add to ControlPlane.module.css */
.newComponent {
  background: var(--bg-surface);
  padding: var(--space-4);
}
```

### CSS Variables

```css
/* Colors */
--bg-deep: #0a0c10;
--bg-surface: #161b22;
--primary: #25c2a0;
--success: #10b981;
--danger: #ef4444;

/* Text */
--text-primary: #e6edf3;
--text-secondary: #8b949e;

/* Spacing */
--space-1: 4px; --space-2: 8px; --space-3: 12px; --space-4: 16px;
```

## Troubleshooting

### Event shows no spans

1. Run: `scripts/diagnose.sh task TASK_ID`
2. Check hierarchy API: `curl localhost:1957/api/observatory/tasks/TASK_ID/hierarchy`
3. Check spans exist: `scripts/diagnose.sh spans TASK_ID`

### Missing child spans (exec.turn, etc.)

Use hierarchy API, not direct spans API:
```bash
# Wrong - only spans with task_id
curl "/api/observatory/spans?task_id=X"

# Correct - all spans in linked traces
curl "/api/observatory/tasks/X/hierarchy"
```

### UI changes not appearing

```bash
make ui-deploy              # Clean build
# Hard refresh: Cmd+Shift+R
```

**For database diagnostics:** See [resources/database_diagnosis.md](resources/database_diagnosis.md)

## Resources

- **REST API Reference**: [resources/rest_api_reference.md](resources/rest_api_reference.md)
- **Database Diagnosis**: [resources/database_diagnosis.md](resources/database_diagnosis.md)
- **Hierarchy Algorithm**: [resources/hierarchy_algorithm.md](resources/hierarchy_algorithm.md)

## Progressive Disclosure

1. **Always loaded**: This SKILL.md (~200 lines)
2. **Execute as needed**: Scripts in `scripts/`
3. **Load on demand**: Resources in `resources/`

## Self-Improvement

This skill self-improves. When issues are discovered:
1. Identify the issue
2. Update scripts/resources/SKILL.md
3. Validate changes
4. Log in CHANGELOG.md

# REST API Reference

Complete reference for all Collaboration Hub API endpoints.

## Control Plane API (`/api/controlplane/`)

### Stats & Metrics

**`GET /api/controlplane/stats`** - Unified stats (Observatory + Coordinator)

Query params:
- `source_type` - Filter by source type
- `provider` - Filter by AI provider
- `model` - Filter by model
- `workspace` - Filter by workspace
- `start_date`, `end_date` - Date range

**`GET /api/controlplane/stats/breakdown`** - Breakdown by provider/model/source/workspace

Same filter params as stats.

### Visualizations

**`GET /api/controlplane/heatmap`** - Activity heatmap by day

Query params:
- `days` (default: 90)
- Plus all filter params from stats

**`GET /api/controlplane/topology`** - Agent topology from config (static)

**`GET /api/controlplane/topology/observed`** - Data-driven topology from message flows

**`GET /api/controlplane/exec-hierarchy`** - Execution hierarchy from spans

Query params:
- `limit` (default: 100)
- `include_messages=true` (enables 4-level hierarchy)

## Observatory API (`/api/observatory/`)

### Workspaces

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/observatory/workspaces` | GET | List workspaces |
| `/api/observatory/workspaces` | POST | Create workspace |
| `/api/observatory/workspaces/{id}` | GET | Get workspace |
| `/api/observatory/workspaces/{id}` | PUT | Update workspace |
| `/api/observatory/workspaces/{id}` | DELETE | Delete workspace |
| `/api/observatory/workspaces/{id}/stats` | GET | Workspace statistics |

### Tasks

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/observatory/tasks` | GET | List tasks |
| `/api/observatory/tasks` | POST | Create task |
| `/api/observatory/tasks/{id}` | GET | Get task |
| `/api/observatory/tasks/{id}` | PUT | Update task |
| `/api/observatory/tasks/{id}` | DELETE | Delete task |
| `/api/observatory/tasks/{id}/hierarchy` | GET | **Full hierarchy (RECOMMENDED)** |
| `/api/observatory/tasks/{id}/timeline` | GET | Task event timeline |

Task list query params:
- `workspace_id` - Filter by workspace
- `status` - Filter by status
- `source_type` - Filter by source
- `limit`, `offset` - Pagination

Hierarchy query params:
- `depth` (default: unlimited) - Tree depth limit
- `include_spans=false` - Summary only mode

### Agent Assignments

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/observatory/tasks/{taskId}/agents` | GET | List agents for task |
| `/api/observatory/agents` | GET | List all agents |
| `/api/observatory/agents` | POST | Create agent |
| `/api/observatory/agents/{id}` | GET | Get agent |
| `/api/observatory/agents/{id}` | PUT | Update agent |
| `/api/observatory/agents/{id}` | DELETE | Delete agent |
| `/api/observatory/agents/{id}/stats` | GET | Agent statistics |

### Spans & Traces

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/observatory/spans` | GET | List spans |
| `/api/observatory/spans` | POST | Create span |
| `/api/observatory/spans/{id}` | GET | Get span |
| `/api/observatory/spans/{id}` | PUT | Update span |
| `/api/observatory/spans/{id}` | DELETE | Delete span |
| `/api/observatory/spans/{id}/events` | GET | Get span events |
| `/api/observatory/spans/{id}/events` | POST | Create span event |
| `/api/observatory/traces` | GET | List traces |
| `/api/observatory/traces/{id}` | GET | Get trace with all spans |

Span list query params:
- `task_id` - Filter by task
- `trace_id` - Filter by trace
- `agent_assignment_id` - Filter by agent
- `start_after`, `start_before` - Time range
- `limit`, `offset` - Pagination

### Messages (Observatory)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/observatory/messages` | GET | List messages |
| `/api/observatory/messages` | POST | Create message |
| `/api/observatory/messages/{id}` | GET | Get message |
| `/api/observatory/messages/{id}` | PUT | Update message |
| `/api/observatory/messages/{id}` | DELETE | Delete message |
| `/api/observatory/messages/{id}/read` | POST | Mark as read |
| `/api/observatory/messages/{id}/archive` | POST | Archive message |

Message list query params:
- `inbox` - Filter by inbox
- `status` - Filter by status (unread, read, archived)
- `from` - Filter by sender
- `task_id` - Filter by task
- `limit`, `offset` - Pagination

### Aggregate Metrics

**`GET /api/observatory/metrics/summary`** - Overall metrics

**`GET /api/observatory/metrics/providers`** - Provider comparison

### Telemetry Ingest

**`POST /api/observatory/ingest/claude`** - Ingest Claude Code telemetry

**`POST /api/observatory/ingest/otel`** - Ingest OTEL/Gemini telemetry

Query param or header: `task_id` or `X-Task-ID`

## Inbox API (`/api/inbox/`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/inbox` | GET | List messages (includes counts) |
| `/api/inbox/{id}` | GET | Get single message |
| `/api/inbox` | POST | Send message |
| `/api/inbox/{id}` | PUT | Update message |
| `/api/inbox/ack-all` | POST | Mark all as read |
| `/api/inbox/cleanup` | POST | Clean up old messages |

## Approvals API (`/api/approvals/`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/approvals?status=pending` | GET | Pending approvals |
| `/api/approvals/:id/approve` | POST | Approve with notes |
| `/api/approvals/:id/reject` | POST | Reject with notes |

## Coordinator API (`/api/coordinator/`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/coordinator/events` | GET | SSE stream for task events |
| `/api/coordinator/events` | POST | Receive coordinator events |
| `/api/coordinator/status` | GET | Coordinator daemon status |
| `/api/coordinator/running` | GET | Currently running tasks |
| `/api/coordinator/pending` | GET | Pending approvals |

## WebSocket Events

**`ws://localhost:1957/ws`** - Real-time events

Event types:
- `inbox_message` - New inbox message
- `task_stream_event` - Live task execution events
- `task_update` - Task status change
- `span_created` - New span ingested

**`ws://localhost:1957/ws/observatory`** - Observatory-specific events

## Example curl Commands

### Health Check
```bash
curl -s http://localhost:1957/health | jq '.'
```

### Get Overall Metrics
```bash
curl -s "http://localhost:1957/api/observatory/metrics/summary" | jq '.'
curl -s "http://localhost:1957/api/controlplane/stats" | jq '.'
```

### List Tasks
```bash
curl -s "http://localhost:1957/api/observatory/tasks?limit=5" | jq '.[].id'
```

### Get Task Hierarchy (RECOMMENDED)
```bash
# Full hierarchy with all spans
curl -s "http://localhost:1957/api/observatory/tasks/TASK_ID/hierarchy" | jq '.'

# Check structure
curl -s "http://localhost:1957/api/observatory/tasks/TASK_ID/hierarchy" | jq '{
  task_id: .task.id,
  agents: [.agents[] | {
    agent_id: .agent.agent_id,
    traces: [.traces[] | {
      trace_id: .trace_id,
      span_count: (.spans | length)
    }]
  }]
}'
```

### Get Spans Directly
```bash
# By task_id (only spans with task_id set)
curl -s "http://localhost:1957/api/observatory/spans?task_id=TASK_ID&limit=20" | jq '.[].name'

# By trace_id (all spans in trace)
curl -s "http://localhost:1957/api/observatory/spans?trace_id=TRACE_ID&limit=50" | jq '.[].name'
```

### Get Exec Hierarchy
```bash
# Basic hierarchy
curl -s "http://localhost:1957/api/controlplane/exec-hierarchy?limit=10" | jq '.hierarchy[].task_id'

# With messages (4-level hierarchy)
curl -s "http://localhost:1957/api/controlplane/exec-hierarchy?include_messages=true&limit=5" | jq '.'
```

### Get Breakdown Metrics
```bash
curl -s "http://localhost:1957/api/controlplane/stats/breakdown" | jq '.by_provider'
curl -s "http://localhost:1957/api/controlplane/stats/breakdown" | jq '.by_model'
```

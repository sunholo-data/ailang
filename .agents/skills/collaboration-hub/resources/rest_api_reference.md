# REST API Reference

Reference for the dashboard API (`ailang serve`, headless — no UI). The authoritative route
table, with each route's auth class and the client that calls it, is in
`design_docs/planned/m-dashboard-headless-api.md`; `internal/server/read_auth_test.go` enforces it.

**Auth:** on a non-loopback bind (Cloud Run) every `/api/` read and `/ws` need
`Authorization: Bearer $AILANG_DASHBOARD_TOKEN` (the server's `COORDINATOR_API_KEY`). Machine
ingest routes and `/api/version` are public. A loopback-bound server (the rig) is exempt.

## Public routes

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Liveness + version |
| `/api/version` | GET | Version only |
| `/benchmarks/<path>.json` | GET | Docs-site benchmark data (read-through from GCS) |
| `/v1/traces`, `/v1/logs`, `/v1/metrics` | POST | OTLP ingest |
| `/api/hooks/claude`, `/api/observatory/hooks` | POST | Claude Code hooks |
| `/api/exec/sessions`, `/api/exec/events` | POST | Exec session ingest |
| `/api/coordinator/events` | POST | Coordinator task events (rebroadcast on `/ws`) |
| `/api/telemetry` | POST | Eval/pipeline self-reports |
| `/` | GET | One-line banner; any unrouted path is a plain 404 |

## Control Plane API (`/api/controlplane/`)

**`GET /api/controlplane/stats/breakdown`** - Breakdown by provider/model/source/workspace
(`ailang chains dashboard stats`)

Query params:
- `source_type` - Filter by source type
- `provider` - Filter by AI provider
- `model` - Filter by model
- `workspace` - Filter by workspace
- `start_date`, `end_date` - Date range

**`GET /api/controlplane/exec-hierarchy`** - Execution hierarchy from spans
(`ailang chains dashboard hierarchy`)

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

## Inbox API (`/api/inbox`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/inbox` | GET | List messages merged with Claude Code events (`ailang chains dashboard inbox`) |

Writes go through `ailang messages` (send, ack) — the dashboard has no inbox write routes.

## Secret Approvals API (`/api/approvals`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/approvals` | POST | Secret-approval intake (own auth: Google ID token / `AILANG_APPROVAL_TOKEN`) |
| `/api/approvals/{id}` | GET | Status poll (same auth) |
| `/api/approvals/{id}/approve` | POST | Approve — signed ntfy `?token=`, or an Approver session |
| `/api/approvals/{id}/reject` | POST | Reject — same |

Coordinator **task** approvals are decided with `ailang coordinator approve|reject <task-id>`
(`--remote gcp` for the cloud plane), not over HTTP.

## Chains API (`/api/chains/`)

Execution chain tracking — agent-level view of multi-stage workflows.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chains` | GET | List chains (with filtering: `status`, `source_type`, `agent_id`, `since`, `limit`, `offset`) |
| `/api/chains/active` | GET | Currently active chains |
| `/api/chains/stats` | GET | Cost/token aggregation |
| `/api/chains/pending` | GET | Chains pending approval |
| `/api/chains/by-message/{id}` | GET | Lookup chain by message ID |
| `/api/chains/by-task/{id}` | GET | Lookup chain by task ID |
| `/api/chains/by-github/{owner}/{repo}/{number}` | GET | Lookup chain by GitHub issue |
| `/api/chains/{id}` | GET | Get chain detail with stages |
| `/api/chains` | POST | Create new chain |
| `/api/chains/{id}/stages` | POST | Add stage to chain |
| `/api/chains/{id}/stages/{stageId}/status` | PATCH | Update stage status |

Chain list query params:
- `status` - Filter by status (active, completed, failed, pending_approval)
- `source_type` - Filter by source (github_issue, message, manual)
- `agent_id` - Filter by agent ID (e.g., design-doc-creator)
- `since` - ISO timestamp or duration (e.g., 24h, 7d) - chains created after
- `limit`, `offset` - Pagination

Stats query params:
- `hours` - Time window in hours (0 = all time)
- `by_agent` - Include per-agent breakdown (true/false)

Active query params:
- `limit` - Maximum results (default: 20)

**CLI equivalents:**
```bash
ailang chains list                       # GET /api/chains
ailang chains active                     # GET /api/chains/active
ailang chains view <chain-id>            # GET /api/chains/{id}
ailang chains stats --hours 168          # GET /api/chains/stats?hours=168
```

## Coordinator API (`/api/coordinator/`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/coordinator/events` | POST | Receive coordinator events (rebroadcast on `/ws`) |
| `/api/coordinator/tasks/{id}/events` | GET | A task's recorded events; `?format=json\|text\|summary`, `?type=`, `?turn=`, `?limit=` |

## WebSocket Events

**`ws://localhost:1957/ws`** - Real-time events

Event types:
- `inbox_message` - New inbox message
- `task_stream_event` - Live task execution events
- `task_update` - Task status change
- `span_created` - New span ingested

External clients on a non-loopback server pass `?token=<COORDINATOR_API_KEY>` (browsers cannot
set headers on a WebSocket).

## Example curl Commands

### Health Check
```bash
curl -s http://localhost:1957/health | jq '.'
```

### List Chains
```bash
# All chains
curl -s "http://localhost:1957/api/chains?limit=10" | jq '.[].id'

# Active chains only
curl -s "http://localhost:1957/api/chains/active" | jq '.'

# Chain details with stages
curl -s "http://localhost:1957/api/chains/CHAIN_ID?include_stages=true" | jq '.'

# Cost/token stats
curl -s "http://localhost:1957/api/chains/stats?hours=168&by_agent=true" | jq '.'

# Lookup by task
curl -s "http://localhost:1957/api/chains/by-task/TASK_ID" | jq '.'
```

### Get Overall Metrics
```bash
curl -s "http://localhost:1957/api/observatory/metrics/summary" | jq '.'
curl -s "http://localhost:1957/api/controlplane/stats/breakdown" | jq '.'
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

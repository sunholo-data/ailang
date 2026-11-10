---
name: collaboration-hub
description: Develop and debug the AILANG dashboard server (`ailang serve`), a headless API with no UI — OTLP/hook ingest, observatory and chains reads, secret-approval intake, /benchmarks/ data, /health. Use when changing internal/server routes or handlers, diagnosing span hierarchy or missing telemetry, or checking what the dashboard API serves.
---

# Dashboard API (Collaboration Hub server)

`ailang serve` is a **headless API** — the React UI was removed in November 2026
(`design_docs/planned/m-dashboard-headless-api.md`). It runs locally on the rig
(`dev.ailang.server`, `127.0.0.1:1957`) and on Cloud Run (`ailang-dashboard`,
`ailang-dev-dashboard`, `ailang-test-dashboard`). Inspect work with the CLI:
`ailang chains`, `ailang chains dashboard <spans|traces|inbox|hierarchy|sessions|tools|stats|health>`,
`ailang coordinator pending|approve|reject`.

## Quick Start

```bash
make services-start         # Start server + coordinator
make services-status        # Check both services
make services-restart       # Rebuild and restart
curl -s localhost:1957/health
```

## Before changing a route

- The route table and its auth classes live in the design doc above and are enforced by
  `internal/server/read_auth_test.go` (`TestReadAuth_EveryRouteClassified`): a new route is
  gated by default and fails that test until it is classified.
- A route here is a route on the internet (`.claude/rules/cloud-endpoints.md`).
- Removing a route: find its callers by route string AND by data key — scripts, `tools/`,
  `.claude/`, `cmd/ailang`, the docs site (`docs/src/lib/benchmarkFetch.js`), and the
  ailang-multivac / ailang-demos repos.

## Architecture

- `internal/server/routes.go` — every route; `read_auth.go` — the default-deny gate
- `internal/server/origin.go` — CORS / origin policy
- `internal/observatory/` — span storage, `/api/observatory/*`, OTLP receiver, hierarchy
- `internal/server/handlers_chains*.go` — `/api/chains/*`
- **Database:** SQLite at `~/.ailang/state/observatory.db` locally; Firestore in the cloud

## Available Scripts

### `scripts/diagnose.sh <command>`

```bash
.claude/skills/collaboration-hub/scripts/diagnose.sh health          # Server + DB check
.claude/skills/collaboration-hub/scripts/diagnose.sh task <id>       # Task details
.claude/skills/collaboration-hub/scripts/diagnose.sh spans <task_id> # Span hierarchy
.claude/skills/collaboration-hub/scripts/diagnose.sh orphans         # Find orphan spans
.claude/skills/collaboration-hub/scripts/diagnose.sh costs           # Cost breakdown
.claude/skills/collaboration-hub/scripts/diagnose.sh recent          # Recent activity
```

## Key API Endpoints

On a non-loopback bind every `/api/` read needs `Authorization: Bearer $AILANG_DASHBOARD_TOKEN`
(the server's `COORDINATOR_API_KEY`); a loopback server is exempt.

**Observatory:**
- `GET /api/observatory/tasks/{id}/hierarchy` - **Full task hierarchy (RECOMMENDED)**
- `GET /api/observatory/spans?task_id=X` - Spans by task
- `GET /api/observatory/traces/{id}` - Trace with all spans

**Control plane (read by `ailang chains dashboard`):**
- `GET /api/controlplane/exec-hierarchy` - Execution hierarchy
- `GET /api/controlplane/stats/breakdown` - Usage breakdown

**Approvals:** coordinator task approvals go through the CLI (`ailang coordinator
approve|reject`, `--remote gcp` for the cloud plane). Secret approvals use
`POST /api/approvals` (own ID-token auth) and the signed ntfy link.

**For the complete API reference:** [resources/rest_api_reference.md](resources/rest_api_reference.md)

## Hierarchy & Trace Correlation

The hierarchy API applies three levels of virtual re-parenting:

1. **Timestamp Correlation** - Fixes ailang.* spans under exec.tool_use
2. **Cross-Trace Merging** - Links spans across different trace IDs
3. **Session-Based Merging** - Links orphan traces by session.id

**Why use hierarchy API?** Child spans don't have `task_id` set. The hierarchy endpoint expands traces to include ALL spans.

**For detailed algorithm:** See [resources/hierarchy_algorithm.md](resources/hierarchy_algorithm.md)

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

**For database diagnostics:** See [resources/database_diagnosis.md](resources/database_diagnosis.md)

## Resources

- **REST API Reference**: [resources/rest_api_reference.md](resources/rest_api_reference.md)
- **Database Diagnosis**: [resources/database_diagnosis.md](resources/database_diagnosis.md)
- **Hierarchy Algorithm**: [resources/hierarchy_algorithm.md](resources/hierarchy_algorithm.md)
- **ID Relationships**: [resources/id_relationships.md](resources/id_relationships.md) - How IDs link across all 3 databases

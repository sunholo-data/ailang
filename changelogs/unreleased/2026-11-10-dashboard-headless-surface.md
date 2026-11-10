### Removed — dashboard API routes whose only caller was the removed UI (2026-11-10)

The dashboard (`ailang serve`) now registers only routes with a consumer outside it
([m-dashboard-headless-api](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-dashboard-headless-api.md)
holds the route table: route, method, auth class, and the client that calls it). Removed, with
their handlers and the helpers only they used (≈7k lines of Go):

- threads and thread messages, `/api/workspaces`, `/api/statistics`;
- agent spawn/stop and the process monitor (`/api/agents`, `/api/monitor`, `/api/agent-stats/`,
  `/api/metrics`, `/api/instances/history`, `/api/hierarchy`, `/api/telemetry/config`,
  `/api/select-folder`);
- coordinator `status`, `pending`, `running`, `approve/`, `reject/`, the task `diff` and the
  `/api/tasks/` alias — task approvals are decided with `ailang coordinator approve|reject`
  (`--remote gcp` for the cloud plane), which runs the same processor;
- the approvals list and history (`GET /api/approvals`, `/api/approvals/history`); a non-secret
  approval id posted to `/api/approvals/{id}/approve|reject` now answers 409 naming the CLI;
- inbox writes (`POST /api/inbox`, `/api/inbox/{id}`, `ack-all`, `cleanup`) — `GET /api/inbox` stays;
- control-plane analytics (`stats`, `heatmap`, `topology`, `span-hierarchy`, `task-hierarchy`,
  `task-evolution`, `usage-timeseries`, `token-distribution`, `outliers`) — `exec-hierarchy` and
  `stats/breakdown` stay for `ailang chains dashboard`;
- `/api/budget/*`, `/api/claude-history/*`, `/api/exec/sessions/{id}/events`, `/api/auth/whoami`,
  and the `/ws/observatory` alias (`/ws` stays: the website builder proxies it);
- the heatmap AILANG transform (`internal/dashboard_transforms/heatmap.ail`), and the
  `full_chat_url` link in enriched spans that pointed at a removed route.

Every ingest route (OTLP, Claude hooks, exec sessions, coordinator events, eval telemetry), the
secret-approval routes, `/api/observatory/*`, `/api/chains/*`, `/benchmarks/*`, `/health` and
`/api/version` are unchanged, and so is the read-auth gate.

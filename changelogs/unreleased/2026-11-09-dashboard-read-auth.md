### Security — the dashboard's data APIs now require authentication (2026-11-09)

`ailang serve` answered every `/api/` route anonymously unless Firebase was configured, so a
publicly reachable dashboard served its inbox, approvals, chains, observatory traces and spans,
and Claude conversation history to anyone. Secret-approval decisions (`POST
/api/approvals/{id}/approve|reject` without the signed ntfy link) and task approvals
(`/api/coordinator/approve|reject/`) also passed through on such a server.

The server now gates **by default**: on any non-loopback bind, every path under `/api/` and the
WebSocket (`/ws`, `/ws/observatory`) require `Authorization: Bearer <COORDINATOR_API_KEY>` (the
WebSocket also takes `?token=`) or a Firebase session with the Viewer or Approver role. With
neither configured those routes return 503 rather than falling open. The only `/api/` routes that
answer without the credential are machine ingest (`POST` to `/api/hooks/claude`,
`/api/observatory/hooks`, `/api/exec/sessions`, `/api/exec/events`, `/api/coordinator/events`,
`/api/telemetry`), the secret-approval intake and status poll (which keep their own ID-token check),
the signed ntfy approve/deny link, and `GET /api/version`. `/health`, `/benchmarks/*`, OTLP
`/v1/*` and the UI's static assets stay public. A server bound to loopback (the default local
`ailang serve`) is exempt.

A route added later is gated until it is classified: `internal/server/read_auth_test.go` enumerates
every registered pattern and fails on an unclassified one.

CLI readers send the key from the new `AILANG_DASHBOARD_TOKEN`: `ailang chains dashboard …`,
`ailang eval-paired`'s Broadcast span fetch (which now names the variable on a 401), the eval
observatory posts, and `scripts/check_pi_wire_budget.sh` (which fails fast without it against a
remote dashboard).

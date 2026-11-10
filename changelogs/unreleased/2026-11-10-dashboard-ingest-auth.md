### Changed — dashboard ingest routes take a shared secret, with a warn-first rollout mode (2026-11-10)

Hardening of `ailang serve` on non-loopback binds. The machine-ingest write routes that the
dashboard's read gate allow-lists are now checked by one ingest gate
(`internal/server/ingest_auth.go`), each class against its own shared secret:

- **otlp** — `POST /v1/traces`, `/v1/logs`, `/v1/metrics` — `AILANG_OTLP_INGEST_TOKEN`
- **hub** — `POST /api/hooks/claude`, `/api/observatory/hooks`, `/api/exec/sessions`,
  `/api/exec/events`, `/api/coordinator/events`, `/api/telemetry` — `AILANG_HUB_TOKEN`

Senders present the secret as `X-AILANG-Ingest-Token: <token>` or `Authorization: Bearer <token>`
(an OTLP exporter: `OTEL_EXPORTER_OTLP_HEADERS=x-ailang-ingest-token=<token>`), compared in
constant time.

New `AILANG_INGEST_AUTH` sets what a write without a valid token gets: `warn` (default) accepts it
and logs an `ingest auth WARN` line naming the class, reason and sender (first write per distinct
sender, then one summary per class per minute with counts); `enforce` refuses it — 401, or 503
naming the variable when that class's secret is unset (fail closed); `off` accepts (logged at
startup). Any other value enforces. A loopback bind — the default local `ailang serve` — is exempt
in every mode.

Behaviour change: `AILANG_OTLP_INGEST_TOKEN` or `AILANG_HUB_TOKEN` set on their own no longer refuse
anything — a deployment that relied on that must also set `AILANG_INGEST_AUTH=enforce`. The OTLP
receiver no longer authenticates by itself (its check moved into the server's gate, which also
removed the duplicate `/api/hooks/claude` check), and a loopback bind no longer checks the OTLP
token at all.

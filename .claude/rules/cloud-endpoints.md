---
paths:
  - "internal/server/**"
  - "internal/observatory/**"
  - "internal/daemon/**"
  - "cmd/ailang/server.go"
  - "cmd/ailang/observatory*.go"
  - "cmd/ailang/dashboard.go"
---

# Deployed Endpoints

**The dashboard you are editing runs in production on Cloud Run.** `internal/server` is not a
local-only dev server — the same binary and the same `mux` are publicly reachable. A route added
here is a route on the internet.

## Services

| Env | Cloud Run service | Project | Public URL |
|-----|-------------------|---------|-----------|
| **prod** | `ailang-dashboard` | `ailang-multivac` | `https://dashboard.ailang.sunholo.com` (DNS) · `https://ailang-dashboard-ao6kuhcibq-ew.a.run.app` |
| **dev** | `ailang-dev-dashboard` | `ailang-multivac-dev` | `https://ailang-dev-dashboard-ejjw6zt3bq-ew.a.run.app` (no DNS name) |

Both in `europe-west1`. Related: `ailang-coordinator`, `ailang-mcp` (`mcp.ailang.sunholo.com`),
`ailang-billing-api`, `ailang-docparse-api`, `ailang-website-builder`; registry-validator lives in
the separate `ailang-registry` project. Deploy is Terraform-only via `ailang-multivac-deploy`
(Cloud Build, **`europe-west3`** — builds are invisible without `--region`).

`dashboard.ailang.sunholo.com` maps to **prod**, which is why `DATA_BASE` in the benchmark fetch
path still points at the dev run.app URL — see `design_docs/planned/m-eval-data-hosting-decouple.md:60`.

## localhost is a default, not a constraint

[`cmd/ailang/server.go:26`](../../cmd/ailang/server.go) defaults `bindAddr = "localhost"`, but
`server.go:31-33` flips it to `0.0.0.0` whenever `PORT` is set — the Cloud Run convention. Reading
only the default and concluding "local-only" is wrong, and has been made before.

## The OTLP receiver is public, and currently unauthenticated BY CHOICE

[`internal/server/server.go:603-605`](../../internal/server/server.go) registers the OTLP receiver at
`/v1/traces`, `/v1/logs`, `/v1/metrics` on the main mux. `server.go:631` wraps that mux in
`corsMiddleware` **only** — Firebase auth is applied per-route and does not cover these. Anyone who
knows the URL can write spans into the production observatory; the read APIs (`/api/observatory/*`,
`/api/chains`) are open too.

Since v0.33.1 a shared-secret gate **exists** but ships disabled — see "OTLP ingest auth" below for
the decision and how to turn it on.

## Verify with a control, never a bare 200

`ailang serve` returns 200 on many paths. Pair every probe with a path that must 404, or the 200
is not a measurement:

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://dashboard.ailang.sunholo.com/v1/bogus   # must be 404
curl -s -X POST -H "Content-Type: application/json" -d '{"resourceSpans":[]}' \
  https://dashboard.ailang.sunholo.com/v1/traces -w "\nHTTP %{http_code}\n"
```

Measured 2026-08-13: prod and dev both 200 on `/v1/traces`, both 404 on the control.

## Deployed observatories are FIRESTORE, not SQLite

Both `ailang-dashboard` and `ailang-dev-dashboard` run with `AILANG_STORAGE=gcp`, which selects
`fsstore.NewObservatoryStore` — spans live in **Firestore**, not in `observatory.db`. Behavioural
confirmation: prod rolled to a new revision on 2026-08-13 and all 190 spans survived, which
in-container SQLite would not have.

**Consequence that has already bitten once:** `internal/observatory/migrate_*.go` migrations run only
from the SQLite paths (`store.go:61`, `backend_sqlite.go:32`). Firestore has **no migration hook**, so
a schema/data migration written there repairs local and rig observatories and *silently does nothing*
to dev or prod. `migrate_v18` was written and shipped on the assumption prod was SQLite; it isn't.

**Before writing an observatory migration, decide which backend actually holds the data.** If it is
the cloud, the migration needs a cloud-side counterpart — see `ailang observatory repair-ids` for the
shape: an explicit, dry-run-by-default command rather than an automatic startup migration, because a
migration that fires on every boot against a live production datastore is a much worse failure mode.

Two Firestore-specific traps that command hit, both absent from the SQLite path:
- A span's **ID is its document key**, so repairing it is create+delete, not an update. Both go in
  one atomic `WriteBatch`.
- `WriteBatch` caps at 500 ops **and ~11.5 MB of payload**. The byte cap is the one that bites:
  span docs carry `gen_ai.prompt`/`gen_ai.completion` (whole prompts and generated programs), and a
  400-op batch blew the limit on the first real run. Chunk on bytes, not just op count.

## OTLP/JSON ID decoding — fixed and deployed (v0.33.1)

OTLP/JSON carries `traceId`/`spanId` as hex, but `protojson` decodes `bytes` as base64 — the
receiver stored corrupted 48-hex-char IDs while answering HTTP 200. Fixed by
`normalizeOTLPJSONIDs` ([otlp_json_ids.go](../../internal/observatory/otlp_json_ids.go)) at all
JSON decode sites (malformed IDs now 400), and prod was fully repaired 2026-08-13 via
`ailang observatory repair-ids --apply`. A stored trace ID longer than 32 hex chars is the
symptom of an unfixed receiver.

## OTLP ingest auth (available, OFF — deliberately)

**Decision, Mark 2026-08-13: stays off for now — "it's just us."** Recorded so it is not re-derived
as an oversight. Revisit when the endpoint is exposed to anyone outside the team.


`AILANG_OTLP_INGEST_TOKEN` gates `/v1/traces`, `/v1/logs`, `/v1/metrics` behind a shared secret, sent
as `X-AILANG-Ingest-Token` or `Authorization: Bearer`. **Unset or empty = auth disabled**, which is
how it ships — Broadcast is already streaming to prod and the rig posts to `localhost:1957` with no
credential, so an enforcing default would break both on deploy.

Enabling it takes **both** sides, or ingest silently stops: set the Cloud Run env var **and** add the
matching custom header on the OpenRouter destination. Verify with a before/after span count, never
with a 200.

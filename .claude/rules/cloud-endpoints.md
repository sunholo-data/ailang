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

## The OTLP receiver is public and unauthenticated

[`internal/server/server.go:603-605`](../../internal/server/server.go) registers the OTLP receiver at
`/v1/traces`, `/v1/logs`, `/v1/metrics` on the main mux. `server.go:631` wraps that mux in
`corsMiddleware` **only** — Firebase auth is applied per-route and does not cover these. Anyone who
knows the URL can write spans into the production `observatory.db`; the read APIs
(`/api/observatory/*`, `/api/chains`) are open too.

This is the ingest path for anything pushing OTLP from outside the rig (e.g. OpenRouter Broadcast).
Adding a shared-secret header check on the OTLP routes is the obvious hardening and has not been done.

## Verify with a control, never a bare 200

`ailang serve` returns 200 on many paths. Pair every probe with a path that must 404, or the 200
is not a measurement:

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://dashboard.ailang.sunholo.com/v1/bogus   # must be 404
curl -s -X POST -H "Content-Type: application/json" -d '{"resourceSpans":[]}' \
  https://dashboard.ailang.sunholo.com/v1/traces -w "\nHTTP %{http_code}\n"
```

Measured 2026-08-13: prod and dev both 200 on `/v1/traces`, both 404 on the control.

## KNOWN DEFECT: OTLP/JSON mangles trace and span IDs

[`internal/observatory/otlp_receiver.go:166`](../../internal/observatory/otlp_receiver.go) decodes
JSON bodies with `protojson.Unmarshal`, which follows the proto3 JSON mapping and treats `bytes`
fields as **base64**. The OTLP/JSON spec overrides this: `traceId`/`spanId` are **hex** strings.
So every OTLP/JSON ingest silently corrupts both IDs and still returns `HTTP 200 {"partialSuccess":{}}`.

Same bug at `otlp_receiver.go:233` (logs) and in `otlp_receiver_metrics.go`. The protobuf path
(`otlp_receiver.go:161`) is correct.

Symptom — a stored trace ID that is **48 hex chars instead of 32**:

```
sent    5b8aa5a2d2c872e8321cf37308d69df2                  (16 bytes)
stored  e5bf1a6b96b677673cef67bcdf6d5c7f7ef7d3c77af5d7f6  (24 bytes) == base64decode(sent)
```

Check any new OTLP/JSON producer before trusting its correlation:

```bash
curl -s "https://dashboard.ailang.sunholo.com/api/observatory/spans?limit=5" | python3 -m json.tool
```

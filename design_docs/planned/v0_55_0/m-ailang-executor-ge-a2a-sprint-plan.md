# Sprint plan: the executor as a Gemini Enterprise A2A agent (M-EXECUTOR-GE-A2A)

**Status**: In progress (started 2026-11-11, attended)
**Parent designs**: [m-ailang-executor-gemini-enterprise.md](m-ailang-executor-gemini-enterprise.md) P2 (the A2A
gateway); [m-connector-channels-marketplaces.md](../m-connector-channels-marketplaces.md) P3.
**Decisions (Mark, 2026-11-11)**: **D2 = (b) serve-api.** A2A v0.3 lives in serve-api, so any AILANG
program can be a Gemini Enterprise A2A agent, not just the fleet. **Model = Gemini on Vertex** through
serve-api's `--ai` (ADC on Cloud Run; no OpenRouter). The coordinator gateway (D2a) stays possible later
behind the same card, for tasks that need the fleet harness.

## What Gemini Enterprise requires (read 2026-11-11)

Source: [Register and manage A2A agents](https://docs.cloud.google.com/gemini/enterprise/docs/register-and-manage-an-a2a-agent) (Preview).

| Requirement | Consequence here |
|---|---|
| A2A **v0.3** card: `protocolVersion`, `name`, `description`, `url`, `version`, `capabilities`, `skills[]` (`id`, `name`, `description`, `tags`), `defaultInputModes`, `defaultOutputModes`; optional `iconUrl` | M1 card |
| Registration **pastes the card JSON**; the card's `url` is the endpoint called | Card `url` from `PUBLIC_BASE_URL`; still served at `/.well-known/agent-card.json` and `agent.json` for other clients |
| `securitySchemes` / `security` on the card are **ignored**; auth is a separate *authorization* resource: client ID, secret, authorization URI, token URI | Reuse a pre-registered `mco_` client (mcp_oauth 0.2 confidential model) |
| Redirect URIs: `https://vertexaisearch.cloud.google.com/oauth-redirect` **and** `.../static/oauth/oauth.html`; the authorization URI carries `redirect_uri=.../static/oauth/oauth.html`, `access_type=offline`, `prompt=consent`, `include_granted_scopes=true` | Mint the GE A2A client with both redirects; the authorize endpoint must tolerate the extra params |
| **PKCE not mentioned** | RISK (R-A1): mcp_oauth requires PKCE. Measure on the first Connect; if GE sends none, decide a per-client exception (package change, separate PR) |
| At call time: user token in `Authorization: Bearer`; on `*.run.app`, also `X-Serverless-Authorization` (discoveryengine service agent ID token) | The existing Bearer gate checks the user token |
| Each `agents.create` = one unit of the daily agent-creation quota (Standard 1/licence/day, pooled, resets midnight Pacific) | Register once; delete before re-registering (Aitana F28) |
| JSON-RPC methods, stream timeouts: **not documented**. Aitana/gde-ap-agent: GE uses `message/stream`, never polls | M1 implements `message/send`, `message/stream`, `tasks/get` |

## Milestones

### M1: serve-api A2A v0.3 (Go, ~350 LOC + tests)
- Card v0.3 at `/.well-known/agent-card.json` and `/.well-known/agent.json`: `protocolVersion: "0.3.0"`,
  `preferredTransport: "JSONRPC"`, `capabilities.streaming: true`, `url` from `PUBLIC_BASE_URL` (never
  `r.TLS`, Aitana F22), version from `--server-info`, `iconUrl` from `--server-info icon=`.
- A **message handler**: one export annotated `@a2a_message`, shape `(string) -> string ! {…}` (the
  user's text in, the answer text out). Its doc comment is the card's single skill description.
  Without one, behaviour is as today (skills = exports, `tasks/send`).
- Methods: `message/send` (returns a completed `Task` with one artifact), `message/stream` (SSE:
  `status-update` working, heartbeats ≤15s while the handler runs, one `artifact-update`, a final
  `status-update` completed/failed), `tasks/get` (bounded in-memory store, TTL). Parts use `kind`
  (v0.3); `type` still accepted on input.
- `tasks/send` unchanged. The embedded copy (`serveapi/a2a_handler.go`) stays as it is (recorded).
- Acceptance: unit tests for the card fields, each method, a handler error → `failed`, SSE framing and
  heartbeat; `TestA2ADisabledByDefault` / `TestA2AEnabledRegistersRoutes` still pass.

### M2: A2A behind the Bearer gate (Go, ~80 LOC + tests)
- Under `--mcp-connect-only`, `/a2a/` goes through the same gate as `/mcp/connect/` (all methods);
  the card routes stay open. Then, and only then, lift the `--a2a` refusal (§3.1 of the channels plan).
- The verified subject reaches the handler the same way it reaches tools (identity header), and the
  per-subject rate limit applies.
- Acceptance: no token → 401 with the RFC 9728 header; bad token → 401; good token → handler runs;
  card readable without a token; mutation check (remove the gate → the 401 tests fail).

### M3: the executor's agent loop, in AILANG (~200 LOC)
- `executor_tools/a2a_agent.ail`: `@a2a_message` handler. A bounded loop over `std/ai` `step` (Gemini on
  Vertex) with three tools in-process: `ailang_prompt`, `ailang_check`, `ailang_run` (the same sandboxed
  functions MCP serves, so no new sandbox). Max steps (e.g. 8), max output; the answer includes the final
  program and its output.
- Acceptance: `--ai-stub` fixture test (a scripted model writes, checks, runs, answers); a live dev run
  answers "write and run a Fibonacci program".

### M4: dev deploy, verify, docs
- Executor image: `--a2a`, `--ai gemini-3-8-flash`, `AI` cap; service account gets `roles/aiplatform.user`
  (multivac terraform, dev); `PUBLIC_BASE_URL`.
- Mint the GE A2A client with both redirect URIs (operator command, Mark's terminal).
- `scripts/verify-ge-a2a.sh`: card checks (Aitana's list), unauthenticated 401, one authenticated
  `message/stream` round trip.
- Mark: register (console, "Custom agent via A2A", paste the card, authorization = the client),
  Connect, ask. Screenshots C9-01/02. Measure R-A1 (PKCE) and the stream hold time (P0-e).
- Docs: `mcp-connectors.md` (A2A section), `hosted-executor.md`, `serve-api.md` (`@a2a_message`),
  an example `examples/runnable/serve_api_a2a_message.ail`, changelog fragment.

## Spend (D5)
Dev only until D5 is decided: model calls bill to the GCP project per task. Bounded by max steps per
message and the per-subject rate limit (30/min).

## Risks
- **R-A1 PKCE** (above). **R-A2** GE may require behaviour the doc does not state (methods, timeouts):
  the first live call is the measurement; logs are kept per request.
- **R-A3 quota:** a failed registration still spends the day's unit. Validate the card with
  `verify-ge-a2a.sh` before registering.

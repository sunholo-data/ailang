# managed_agents Executor

AILANG executor for Google Cloud's **Vertex AI Managed Agents API**
(Gemini Enterprise Agent Platform). Replaces the retired `gemini` CLI
executor for `gemini-3-5-flash` agent-mode evals, post Gemini CLI
deprecation on 2026-06-18.

Sprint: [M-MANAGED-AGENTS](../../../design_docs/planned/v0_22_0/m-antigravity-cli-migration.md)
(target v0.22.0).

## What this executor does

POSTs a single HTTP request to the Vertex AI Managed Agents endpoint, reads
the Server-Sent Events stream back, and folds the events into the standard
`executor.Result` shape. The Managed Agents API runs the **Antigravity
agent harness** (powered by gemini-3-5-flash by default) inside a Google-hosted,
isolated Linux sandbox per interaction — so we get tool execution, multi-turn
state, and sandbox provisioning as a service, with no local subprocess.

## Endpoint

```
POST https://aiplatform.googleapis.com/v1beta1/projects/<project>/locations/global/interactions
```

- **Version prefix is `v1beta1`** — `v1` and `v1beta` return HTML 404. (Discovery
  cost for that lesson: ~1 hour during M-MANAGED-AGENTS M1; captured as the
  `feedback_managed_agents_v1beta1_path` memory.)
- **Location must be `global`** — no regional endpoints as of 2026-05-20.
- **POST only**.

## Authentication

Application Default Credentials (ADC):

```bash
gcloud auth application-default login
```

Token acquisition is delegated to the shared
[`internal/auth/gcp`](../../auth/gcp) package, which prefers the GCE/Cloud Run
metadata server (zero-config in cloud) and falls back to `gcloud auth
application-default print-access-token` (local dev / CI).

The executor uses the same ADC path as `internal/ai/gemini` does for direct
`generateContent` calls — one credential setup, two consumers.

## Request body (verified live 2026-05-20)

```json
{
  "stream": true,
  "background": true,
  "store": true,
  "agent": "antigravity-preview-05-2026",
  "environment": {"type": "remote"},
  "input": [
    {
      "type": "user_input",
      "content": [{"type": "text", "text": "<task directive>"}]
    }
  ],
  "system_instruction": "<task system prompt>"
}
```

Field notes:

- `stream: true` — SSE response is the only supported mode for now.
- `background: true` — **required**. Sync mode returns
  `"Chiliagon path must set background to true."`
- `store: true` — retain conversation + sandbox so future calls can reuse
  `interaction_id` + `environment_id` for multi-turn (first cut: fresh sandbox
  every call; multi-turn is a follow-up).
- `environment: {"type": "remote"}` — provision a fresh Google-hosted Linux
  sandbox. Reuse a prior one by passing a raw `env_<id>` string instead.
- `agent` — only `antigravity-preview-05-2026` is public as of 2026-05-20.
  Future agents can be specified via `agent_model_name` in `models.yml`.
- Required header: `Api-Revision: 2026-05-20` (pins behaviour against schema
  drift).

## Response: SSE event stream

Verified event types (see [testdata/sse_pong.txt](testdata/sse_pong.txt) for
the canonical fixture):

| Event | Purpose |
|---|---|
| `interaction.created` | Initial event; carries `interaction.id` |
| `interaction.status_update` | Heartbeat / status pulse |
| `step.start` | Beginning of a model step (`step.type: "model_output"`) |
| `step.delta` | Streaming text chunk (`delta.text`, `delta.type`) |
| `step.stop` | End of a step |
| `interaction.completed` | Terminal data event; carries full usage + `environment_id` |
| `done` | Terminal sentinel (`data: [DONE]`) |

Unknown event types are captured into
`Result.ProviderData["managed_agents_unknown_events"]` so schema drift
surfaces without dropping data.

## Result mapping

| `executor.Result` field | Source |
|---|---|
| `Success` | `interaction.completed.status == "completed"` |
| `Output` | Concatenated `step.delta.text` of all `model_output` steps |
| `InputTokens` | `interaction.completed.usage.total_input_tokens` |
| `OutputTokens` | `total_output_tokens + total_thought_tokens` |
| `CostUSD` | Client-side computed from gemini-3-5-flash Vertex rates |
| `SessionID` | `interaction.id` (for resume / multi-turn) |
| `ProviderData["managed_agents_environment_id"]` | For sandbox reuse |
| `ProviderData["managed_agents_total_thought_tokens"]` | Reasoning token surface |
| `ProviderData["managed_agents_unknown_events"]` | Forward-compat capture |
| `FinishReason` | `stop` / `error` / `timeout` based on terminal status |

## Cost model

| Component | Rate (per 1K tokens) | Source |
|---|---|---|
| Input | $0.0015 | Vertex gemini-3-5-flash pricing |
| Output (incl. reasoning/thoughts) | $0.009 | Vertex gemini-3-5-flash pricing |

The Managed Agents API reports `total_thought_tokens` separately in
`usage`; we bill them at the output rate because Vertex's pricing model
doesn't distinguish reasoning from candidate tokens.

## Limits and known gaps

- **No multi-turn yet.** Each Execute() call provisions a fresh sandbox.
  Multi-turn carry-over via `interaction.id` + `environment_id` is captured
  in `ProviderData` but the harness side doesn't wire those into follow-up
  calls. Tracked as a follow-up after M-MANAGED-AGENTS.
- **No tool-use event handling beyond capture.** The minimal PONG probe
  didn't trigger any tool steps. If smoke tests surface tool-call event
  types, they'll be in `ProviderData["managed_agents_unknown_events"]` for a
  follow-up parser pass.
- **No cancel/abort signal.** The only kill switch is context cancellation
  (timeout from Task.Timeout). The API has no explicit cancel endpoint
  documented yet.
- **Region locked to `global`.** No us-central1 / europe-west fallback.

## Trust boundary

The Managed Agents API runs the sandbox in Google-hosted infrastructure.
Sandbox isolation is Google's problem; from our side, the workspace path on
the calling machine (`Task.Workspace`) is **not** uploaded — only the
directive + system instruction. The sandbox starts empty unless `environment`
specifies `sources` with GCS URIs (out of scope for this first cut).

## Testing

```bash
# Unit tests against the captured fixture (no network)
go test ./internal/executor/managed_agents/...

# Live test against real Vertex API (costs real money — currently ~$0.02 / call)
AILANG_MANAGED_AGENTS_LIVE=1 GOOGLE_CLOUD_PROJECT=ailang-dev \
  go test ./internal/executor/managed_agents/ -run TestLive_ -v
```

## Related

- [internal/auth/gcp](../../auth/gcp) — shared ADC helper consumed by both
  this executor and `internal/ai/gemini`.
- [internal/ai/gemini](../../ai/gemini) — direct Vertex `generateContent`
  for standard-mode (non-agentic) Gemini calls. Standard-mode results in
  `eval_results/baselines/v0.20.0/standard/*gemini-3-5-flash*.json` show
  88.2% AILANG pass rate (tied #1 with gpt5-5).
- [docs/internal/EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md)
  — the two-pillar contract this executor follows.

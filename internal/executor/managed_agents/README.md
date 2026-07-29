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

## The 2026-07 feature drop is Gemini-Developer-API-only (probed 2026-07-28)

Google's "Managed Agents: 3.6 Flash, hooks and more" announcement and the
[agent-hooks docs](https://ai.google.dev/gemini-api/docs/agent-hooks) describe
the **Gemini Developer API** surface (`generativelanguage.googleapis.com`,
API-key auth). This executor is on **Vertex**. Probes S/T/U in
[managed_agents_features_live_test.go](managed_agents_features_live_test.go)
tested each advertised feature against the surface we actually use:

| Feature | Vertex status | Evidence |
|---|---|---|
| `agent_config` container | **Accepted** (needs `"type":"antigravity"`) | S1 |
| `agent_config.max_total_tokens` | **Parsed, validated, NOT ENFORCED** | S5/S6/S9/S10 |
| `agent_config.model` | Accepted; effect **unverified** | S2/S3/S4 |
| Hooks (`.agents/hooks.json`) | **No delivery path** | T1/T2/T3 |
| Environments API (list/get/delete) | **Absent** (`Method not found`) | U |

Details that matter before anyone builds on these:

- **`max_total_tokens` is a trap.** It is a real, strictly-validated field —
  a non-numeric value 400s with `The value is invalid for
  'agent_config.max_total_tokens'`, and an unknown sibling key 400s as
  `Unknown parameter` — so the request *looks* like it set a cost ceiling.
  It is then ignored: a cap of `64` (probed in both string and integer form)
  ran to `status:"completed"` at **216,843** and **87,828** tokens
  respectively. Do NOT wire this in as a budget control; a silently
  non-functional ceiling is worse than the honest post-hoc reporting below,
  because it reads as a guarantee.
- **`agent_config.model` cannot be verified from our side.** A nonexistent id
  (`gemini-9.9-does-not-exist`) is accepted without complaint, and the SSE
  stream carries no model echo — the only `model` substring in a full raw dump
  is the `model_output` step type, and `interactions.get` does not echo
  `agent_config` either. Asking the sandbox which model it runs is not
  evidence (it self-reported `gemini-1.5-pro`, which no documentation
  supports). So we must not record a pinned model as fact in eval metadata.
  A behavioural discriminator (latency / thought-token distribution across
  pinned tiers, n>1) is the open follow-up.
- **Hooks have no route in.** `inline` sources are still rejected
  (`Unsupported environment data source type: INLINE. Must be one of:
  [gcs, skill_registry]`), the sandbox cannot create `/.agents` (root is
  read-only, `sudo` blocked by `no_new_privs`), no `/.agents` ships in the
  image, and a config self-installed at `/workspace/.agents/hooks.json` is
  never consulted — the gate script denied correctly when piped by hand, then
  the matching `code_execution` ran anyway. **Untested:** a `gcs` source
  mounting `.agents/` at boot, which is the one remaining delivery path if
  discovery is boot-time-only.
- **Fail-open, if hooks ever land here.** Per the docs a hook that crashes,
  times out, or returns non-2xx is treated as `allow`. A broken gate is
  therefore indistinguishable from "the gate didn't help" unless the deny path
  is asserted positively — build the probe before building the gate.

Why we care about hooks at all: a `pre_tool_execution` hook can deny a tool
call *with a reason the model then adapts to*. That is a feedback channel — it
would let us reject a proposed `.ail` edit that does not type-check, with the
compiler diagnostic as the reason, which is the per-edit convergence lever
that works elsewhere in this repo. That prize is why the delivery gap is worth
re-probing when Google moves these to Vertex.

## Limits and known gaps

- **Cost budget is post-hoc, not mid-stream.** The Managed Agents API only
  reports cumulative usage in the terminal `interaction.completed` event
  (step events carry no token counts), so mid-stream kill-on-cost is
  IMPOSSIBLE for this executor. The executor compares actual final cost
  against `task.Budget.MaxUSD` AFTER the run and surfaces overruns via:
  - `Result.CostKilledAt` (for telemetry aggregation)
  - `Result.ProviderData["managed_agents_cost_over_budget"]` (structured)
  - Loud stderr warning visible in `ailang eval-suite` output
  Per design: useful over-budget results are KEPT and flagged, not
  discarded. `Result.Success` follows the API's `interaction.completed.status`.
  To prevent overruns: tighten `models.yml::budgets.hard_timeout_secs` (the
  wall-clock safety net is the only live kill switch).

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

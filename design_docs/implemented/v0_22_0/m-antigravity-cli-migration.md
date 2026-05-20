# M-MANAGED-AGENTS: Retire Gemini CLI, Adopt Vertex Managed Agents API

**Status**: IMPLEMENTED
**Target**: v0.22.0
**Priority**: P0 (Blocker) — Gemini CLI stops serving individual-tier requests on **2026-06-18**
**Estimated**: 3-4 days
**Dependencies**:
- `internal/executor/gemini/` (current Gemini CLI executor — to be deleted)
- `internal/eval_harness/models.yml` (`agent_cli: "gemini"` entries — to be removed)
- gcloud Application Default Credentials (existing; **the only acceptable auth surface**)
- `aiplatform.googleapis.com/v1beta1/...` Managed Agents API live in `ailang-dev` (verified 2026-05-20)

---

## Executive Summary

At Google I/O 2026 (2026-05-19), Google announced the deprecation of Gemini CLI in favor of Antigravity 2.0 (CLI, SDK, Managed Agents API). After two investigation cycles on 2026-05-20:

- **Antigravity CLI (`agy`) v1.0.0** ships today but is **unsuitable** for our eval harness: OAuth-only (no ADC), no `--model` flag, no `--output-format stream-json`, heavy gRPC-server boot per invocation, pollutes working directories with `.antigravitycli/` symlinks. Skipped.

- **Vertex AI Managed Agents API** is **live and ADC-native today**: `POST https://aiplatform.googleapis.com/v1beta1/projects/<p>/locations/global/interactions` with `Authorization: Bearer $(gcloud auth print-access-token)` and `Api-Revision: 2026-05-20`. Verified end-to-end against `ailang-dev` — see [testdata/managed_agents_sse_pong.txt](testdata/managed_agents_sse_pong.txt) for the SSE fixture. **This is our new agent-mode executor for Gemini-family models.**

- The legacy **Gemini CLI** is on a forced retirement timeline — 2026-06-18 deadline. Retire cleanly.

- Direct Vertex `generateContent` for `gemini-3-5-flash` (standard-mode) continues unchanged: 88.2% AILANG in v0.20.0, tied #1.

**Sprint shape**: delete Gemini CLI executor + add Managed Agents API executor + wire models.yml. ~3-4 days end-to-end. No Cloud Run Jobs changes, no Antigravity CLI work, no backfills.

---

## Strategic Context

> "With Google, everything is either deprecated or in beta."

The first pass of this design ruled out the Managed Agents API based on my own bad probe (tried `v1/...` and `v1beta/...` paths, missed `v1beta1`). The user pointed me at the docs page, which uses `v1beta1`, and a direct probe immediately got a backend response (HTTP 400 "Provisioning is in progress" — provisioning rollout for first-call onboarding). After provisioning completed (~3 min) and one more iteration on the body shape (`background: true`, structured `input` array, `environment: {"type": "remote"}`), the end-to-end call returned a clean SSE stream with the full `interaction.completed` event.

Lesson encoded here: when Google ships a new GCP service, **probe `v1beta1` first** — that's the Vertex AI canonical version prefix for new APIs, not `v1` or `v1beta`. Saved as a memory.

---

## Verified API Surface (2026-05-20)

### Endpoint

```
POST https://aiplatform.googleapis.com/v1beta1/projects/<project>/locations/global/interactions
```

- Location must be `global`.
- POST only.
- ADC auth via `Authorization: Bearer $(gcloud auth print-access-token)`.
- Required headers: `Content-Type: application/json`, `Api-Revision: 2026-05-20`.

### Request body

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
      "content": [{"type": "text", "text": "<prompt>"}]
    }
  ]
}
```

Notes:
- `stream: true` returns SSE.
- `background: true` is **required** — the API rejected sync with `"Chiliagon path must set background to true."`
- `store: true` retains conversation + sandbox so we can resume.
- `environment: {"type": "remote"}` provisions a fresh sandbox. Reuse a previous one with `"environment": "env_<id>"` (string form).
- `input` is a structured array, not a string. Multi-modal supported (we use `text` only).
- For multi-turn: pass `"interaction": "<prior_id>"` to continue conversation.
- `agent: "antigravity-preview-05-2026"` is the only currently-public agent name; model selection is implicit (gemini-3.5-flash). Other agent names may appear over time.

### Response: SSE stream

Captured fixture: [testdata/managed_agents_sse_pong.txt](testdata/managed_agents_sse_pong.txt) (1203 bytes). Event sequence:

| Event | Payload keys |
|---|---|
| `interaction.created` | `interaction.id`, `interaction.status`, `interaction.object` |
| `interaction.status_update` | `interaction_id`, `status` |
| `step.start` | `index`, `step.type` (e.g. `"model_output"`) |
| `step.delta` | `index`, `delta.text`, `delta.type` |
| `step.stop` | `index` |
| `interaction.completed` | `interaction.id`, `status`, `usage` (full token breakdown), `environment_id`, timestamps |
| `done` | `data: [DONE]` sentinel |

Terminal data event is `interaction.completed`. Final stream marker is `data: [DONE]` (mirrors OpenAI/Anthropic conventions).

### Usage payload (from `interaction.completed`)

```json
{
  "total_tokens": 7343,
  "total_input_tokens": 6560,
  "total_output_tokens": 35,
  "total_thought_tokens": 748,
  "input_tokens_by_modality":  [{"modality":"text", "tokens":6560}],
  "output_tokens_by_modality": [{"modality":"text", "tokens":35}]
}
```

Maps cleanly to our existing `Result` shape:
- `InputTokens` ← `total_input_tokens`
- `OutputTokens` ← `total_output_tokens` (+ `total_thought_tokens` if we want to surface reasoning cost separately)
- `TotalTokens` ← `total_tokens`
- `CostUSD` ← computed client-side from Vertex pricing ($1.50 / $9.00 per 1M for gemini-3.5-flash, plus reasoning at output rate)

### Multi-turn handles

`interaction.completed` returns:
- `interaction.id` — pass as `"interaction"` on next call to continue the conversation (server-side history)
- `environment_id` — pass as `"environment"` (string form) to reuse the same Linux sandbox

Server-side state means we don't accumulate history client-side. Big win for multi-turn token cost.

### Cost reference (one trivial probe)

- 6560 input + 35 output + 748 thought tokens for "say PONG"
- At Vertex pricing: ~$0.017
- Extrapolating to a 23k-token AILANG benchmark prompt + multi-turn loop: expect $0.05–0.30 per benchmark, comparable to existing executors.

---

## Scope

### In Scope

1. **Delete Gemini CLI executor**. Remove `internal/executor/gemini/` package, the blank import in [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go), and gemini-CLI-specific testdata.

2. **Add Managed Agents API executor** at `internal/executor/managed_agents/`:
   - HTTP client against the Vertex `v1beta1/.../interactions` endpoint.
   - ADC auth via `golang.org/x/oauth2/google` (same path as [internal/ai/gemini/client.go](../../../internal/ai/gemini/client.go)).
   - SSE parser for the event stream documented above. Use a forward-compat unknown-event map to capture new event types into `ProviderData` (mirror motoko's pattern).
   - Multi-turn support via server-side `interaction.id` + `environment_id`.
   - Result population: tokens, cost (client-side computation from Vertex price table), success/failure from `interaction.completed.status`.

3. **models.yml updates**:
   - Remove `agent_cli: "gemini"` from 8 entries: `gemini-2-5-flash`, `gemini-2-5-pro`, `gemini-3-1-flash-lite`, `gemini-3-1-pro`, `gemini-3-5-flash`, `gemini-3-flash`, `gemini-3-flash-preview`, `gemini-3-pro`.
   - Add `agent_cli: "managed_agents"` to `gemini-3-5-flash` (this is the model the API runs by default; other agents may appear). Use `agent_model_name: "antigravity-preview-05-2026"` (the `agent` field value, since model is implicit).

4. **Config-load error** when an existing config still has `agent_cli: "gemini"`: clear next-step message pointing at `managed_agents` for `gemini-3-5-flash` agent-mode, and noting older Gemini models are now standard-mode-only.

5. **Documentation**:
   - [docs/docs/guides/evaluation/harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md) — remove Gemini CLI section, add Managed Agents API setup (`gcloud auth application-default login`, project/location, one-time provisioning note).
   - [.claude/rules/coordinator.md](../../../.claude/rules/coordinator.md) — executor table refresh.

6. **Memory**:
   - `feedback_managed_agents_v1beta1_path.md` — captures the `v1beta1` lesson.
   - `reference_managed_agents_api.md` — endpoint + body shape + event schema (this design doc's API surface section, snapshotted).

### Out of Scope

- **Antigravity CLI executor** — still OAuth-only / no model flag / no stream-json. Out unless Google ships an ADC + headless mode.
- **Cloud Run Jobs deprecation** — the Managed Agents API gives us managed sandboxes for Gemini, but motoko / opencode / claude still need our container infra (non-GCP auth). No change to multivac in this sprint.
- **Backfilling v0.20.0 agent rows for Gemini models** — per user direction.
- **`std/ai` module changes** — untouched.
- **AILANG-side agent-mode optimisations** for reusing `environment_id` across benchmarks — possible cost optimisation but defer; safer to provision fresh per benchmark for first cut.

---

## Phasing

### M1 — Retire Gemini CLI (0.5 days)

- **M1.1** Delete `internal/executor/gemini/` (package + tests + testdata). The package can leave a tombstone `doc.go` if external imports need a one-release grace period; otherwise delete outright.
- **M1.2** Remove the blank import from [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go).
- **M1.3** Strip `agent_cli: "gemini"` and gemini-CLI-specific fields (`agent_model_name`) from 8 models.yml entries; preserve their standard-mode (`provider: google`, Vertex generateContent) configuration.
- **M1.4** Add a validation hook in [`internal/eval_harness/models.go`](../../../internal/eval_harness/models.go) that emits a clear next-step error for `agent_cli: "gemini"`.

**Exit criteria**: `make ci` green; `internal/executor/gemini/` package gone; any external script trying to use `agent_cli: "gemini"` fails fast with the prescribed error.

### M2 — Managed Agents API Executor (2 days)

- **M2.1** Scaffold `internal/executor/managed_agents/`: `executor.go` (main flow), `client.go` (HTTP), `sse.go` (parser), `types.go` (request/response structs), `testdata/`, `README.md`. Follow the package shape contract in [docs/internal/EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md).
- **M2.2** HTTP client: ADC via `golang.org/x/oauth2/google.FindDefaultCredentials`. POST to `v1beta1/projects/<p>/locations/global/interactions` with the headers + body shape captured above. Project/location from `executor.Config` (matching what other executors do).
- **M2.3** SSE parser: parse `event: <name>\ndata: <json>\n\n` blocks. Handle the documented event types (`interaction.created`, `interaction.status_update`, `step.start|delta|stop`, `interaction.completed`, `done`). Unknown event types → `ProviderData["managed_agents_events"]` map for forward compat.
- **M2.4** Result mapping: tokens from `interaction.completed.usage`; cost computed from Vertex price table (input + output + thought tokens at appropriate rates); `Success = (status == "completed")`; `Output` from the accumulated `step.delta.text` of any `step.type == "model_output"` step.
- **M2.5** Multi-turn (M2 of the eval-harness side; the executor itself supports it via task fields): if the executor receives a `Task.InteractionID` and `Task.EnvironmentID`, pass them in the next request. Skip if the harness doesn't yet pass them — they're optional.
- **M2.6** Tests: unit tests parse [testdata/managed_agents_sse_pong.txt](testdata/managed_agents_sse_pong.txt) end-to-end. Integration test makes a real probe (skippable when ADC absent).
- **M2.7** Register in [internal/executor/factory.go](../../../internal/executor/factory.go); add blank import in [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go).

**Exit criteria**: `go test ./internal/executor/managed_agents/...` green; manual end-to-end smoke against fizzbuzz returns PASS.

### M3 — Smoke Gate (0.5 days)

- **M3.1** Smoke-test `gemini-3-5-flash` via `agent_cli: "managed_agents"` against the standard set (`fizzbuzz`, `adt_option`, `csv_to_json_converter`). Target: 3/3 PASS.
- **M3.2** Capture a richer fixture for tool-use scenarios (a benchmark that requires `step.type` other than `model_output`). Save as `testdata/managed_agents_sse_tool_use.txt`. Update the SSE parser if any new event types appear.
- **M3.3** Refresh the executor's README with the verified event schema.

**Exit criteria**: 3/3 smoke; tool-use event types documented or confirmed absent.

### M4 — Documentation + Memory (0.5 days)

- **M4.1** Update [harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md): Gemini CLI section removed; Managed Agents API section added (one-time provisioning note for first call per project).
- **M4.2** Update [coordinator.md](../../../.claude/rules/coordinator.md) executor table.
- **M4.3** Write two memory files:
  - `feedback_managed_agents_v1beta1_path.md` — "When probing new Vertex AI APIs, try `v1beta1` first" + the bad-probe story so future sessions don't repeat the mistake.
  - `reference_managed_agents_api.md` — verified endpoint + body + event schema + multi-turn handle reference.
- **M4.4** Send ailang-feedback message to `sunholo-demos` + `motoko_explore` agents: gemini-cli executor removed; agent-mode for Gemini family now goes through `managed_agents`.

**Exit criteria**: docs current; memories written; cross-repo notifications sent.

### M5 — Validation (deferred to release verification)

Run a 23-benchmark agent suite for `gemini-3-5-flash` via Managed Agents API as part of v0.22.0 release validation (separate sprint task). Compare against v0.20.0 gemini-3-flash standard rows for cross-mode comparison.

---

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| API `v1beta1` schema drifts before v0.22.0 release | Medium | `Api-Revision: 2026-05-20` header pins behaviour; fixture-based tests catch parser drift; lenient unknown-event handling absorbs additive changes |
| `antigravity-preview-05-2026` agent name changes / new agents appear | Medium | Agent name in models.yml (`agent_model_name`), one config change to swap |
| `background: true` requirement extended to other sync paths we depend on | Low | Only affects the Managed Agents API; doesn't touch `generateContent` |
| Vertex pricing for the Managed Agents API differs from raw gemini-3.5-flash pricing (managed-sandbox premium) | Medium | Compute cost from the `usage` payload + a configurable rate; flag any large variance vs estimate in M3 |
| Tool-use event shape surprises during M3.2 | Medium | Capture the fixture and adapt — that's why M3.2 exists |
| Sandbox state leaks across benchmarks if `environment_id` is reused incorrectly | Medium | First cut provisions a fresh sandbox per benchmark (`environment: {"type": "remote"}` not a reused ID); revisit reuse as a cost optimisation later |
| Gemini CLI deprecation deadline | 2026-06-18 | 3-4 day sprint with merge target 2026-06-05 — comfortable margin |

---

## Acceptance Criteria

A v0.22.0 release shipping this sprint must satisfy:

- [ ] `internal/executor/gemini/` deleted (or tombstone-only).
- [ ] `internal/executor/managed_agents/` exists; passes its unit + integration tests.
- [ ] `models.yml` has no `agent_cli: "gemini"` entries; `gemini-3-5-flash` has `agent_cli: "managed_agents"`.
- [ ] `ailang eval-suite --agent --models gemini-3-5-flash --benchmarks fizzbuzz,adt_option,csv_to_json_converter --langs ailang` returns 3/3 PASS.
- [ ] `ailang doctor` checks ADC + Vertex project + (optionally) Managed Agents provisioning state.
- [ ] [harness-setup.md](../../../docs/docs/guides/evaluation/harness-setup.md) reflects the Managed Agents API path; no Gemini CLI references remain.
- [ ] Two memory files written.
- [ ] Sprint merged before **2026-06-10** (1 week before Gemini CLI shutoff).

---

## Cost / Budget

| Item | Estimate |
|---|---|
| Engineering | 3-4 days (M1-M4) |
| Eval API spend (M3 smoke ~3 benchmarks; M5 validation ~46 benchmark runs) | ~$5 smoke + ~$15 validation = ~$20 |
| **Total cash** | **~$20** |

---

## Strategic Posture

The first version of this design doc concluded "the Vertex Managed Agents API isn't shipped" and shrunk the sprint to a pure retirement. That was based on my own incomplete probe — I tried two version prefixes and gave up. The user pointed me at the docs page, and the third prefix worked: `v1beta1`. The probe took less than a minute once I tried the right path.

The lesson encoded here is structural, not stylistic: **when probing new GCP APIs, exhaust the version-prefix space (`v1`, `v1beta`, `v1beta1`, `v2`, `v2beta`) before concluding the endpoint doesn't exist.** I've recorded this as a memory so it doesn't bite a future session.

The corrected sprint shape is still small — 3-4 days, $20 — because the API surface is well-suited to our existing executor pattern: HTTP client + SSE parser + ADC auth + structured `Result` mapping. No homegrown sandboxing, no Cloud Run Jobs orchestration. Google built exactly the thing we'd otherwise have built.

Sources:
- [Antigravity 2.0 launch — Google Developers Blog](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/)
- [Managed Agents in the Gemini API — Google Blog](https://blog.google/innovation-and-ai/technology/developers-tools/managed-agents-gemini-api/)
- [Managed Agents API on Agent Platform — Vertex docs](https://docs.cloud.google.com/gemini-enterprise-agent-platform/build/managed-agents)
- [Interact with agents — Vertex docs (the page I should've read first)](https://docs.cloud.google.com/gemini-enterprise-agent-platform/build/managed-agents/interact-with-agents)
- M1.1 + M2.2 captures: `agy --help` v1.0.0 + Vertex API probe + SSE fixture in this doc's "Verified API Surface" section + [testdata/managed_agents_sse_pong.txt](testdata/managed_agents_sse_pong.txt)

# M-AILANG-EXECUTOR-GE: the ailang_only executors, reachable from Gemini Enterprise over MCP and A2A

**Status**: **P1 built 2026-10-14** — `executor_tools/executor.ail` + `docker/Dockerfile.executor-mcp`, deployed to dev as `ailang-dev-executor-mcp`, IAM-only, and connected to the GE app in `sunholo-gemini-enterprise`. Setup and rebuild runbook: [docs/internal/gemini-enterprise-executor-mcp.md](../../../docs/internal/gemini-enterprise-executor-mcp.md) (tool names mirror `ailang_bootstrap/mcp-server`; one endpoint also serves the docs tools and `submit_feedback`). Rest Planned (drafted 2026-10-14, attended; **revised the same day** to build on the existing `ailang_only` cloud executors instead of a fresh sandbox). Phase 0 is a discovery spike whose results gate Phases 2–3.
**Target**: v0.55.0
**Priority**: P1. This is the first use of the `ailang_only` lane by callers outside our own fleet, and Gemini Enterprise (GE) is a channel the Aitana platform has already proven.
**Estimated**: ~5.5 days. P0 spike 0.5d (attended: needs the GE console), P1 MCP 1.5d, P2 A2A gateway 2.5d, P3 registration and runbook 1d.
**Dependencies**: none blocking. Builds on the **live** `ailang_only` lane: [m-agent-ailang-only-execution](../../implemented/v0_39_0/m-agent-ailang-only-execution.md), M-EXECUTOR-POLICY-HARDENING (v0.41.0), and [m-motoko-ailang-only-lane](../m-motoko-ailang-only-lane.md) (motoko on the lane since 2026-10-08). The OAuth lane reuses [m-mcp-oauth-package](../v0_52_0/m-mcp-oauth-package.md) if P0 shows GE wants OAuth.
**Scope**: an AILANG serve-api program plus a Dockerfile (this repo); the A2A gateway (P2, location is D2); one new agent, `ge-executor`, plus a policy file in `ailang-multivac/config/`; a Cloud Run service in `ailang-multivac/terraform/` (dev first).

**Quorum triggers fired**: **#4.** The load-bearing premises are about Gemini Enterprise, which we do not control: whether GE takes custom MCP servers, and what auth GE sends to a Cloud Run A2A agent. **#1** for the design-freeze items D1–D6. **#2** if D2 chooses the coordinator, since it would then take on a new public surface. Not run yet. Run it after P0, so the PENDING rows are measured first.

---

## The idea in one paragraph

We already run the most locked-down place to write AILANG: the **`ailang_only` executor lane**. It has no shell, no native file tools and 8 typed tools. Files go through `ailang policy-tool`; code runs through `ailang run --policy` in `security_mode = "restricted"` with a supervised timeout. The harness is pi, or motoko since 2026-10-08, and each task runs as a Cloud Run Job dispatched by the coordinator (V13–V17). GE needs two things from it, and both are **projections of that lane, not a new sandbox**:

- **MCP — "Gemini is the harness."** Expose the lane's *tools* (check, run under policy, docs/builtins/examples search) as MCP tools. Gemini runs the reasoning loop and calls them inline.
- **A2A — "our harness does the work."** Expose the lane's *agent*. GE delegates a whole task, the coordinator dispatches it to an `ailang_only` agent (motoko or pi, our model), and the completion summary comes back as the A2A artifact.

This matches the MCP-for-inline, A2A-for-delegation split from the suggestion pasted into the request. The difference is that both sides inherit a security boundary that has already been hardened and measured, rather than one written for this doc.

## Existing surfaces this builds on (added 2026-10-14, after review)

| Surface | What it is | How this design uses it |
|---|---|---|
| **Public docs MCP** (`mcp_tools/*.ail`, `ailang-mcp` / mcp.ailang.sunholo.com) | 21 read-only tools: docs, stdlib, examples, benchmarks, changelog, prompts. Plus the **built-in `submit_feedback`** tool (registered Go-side in `internal/apiserver/feedback_tool.go` on every serve-api MCP unless `--no-feedback-tool`), which writes to the plane's `public-feedback` inbox | The executor service serves `mcp_tools/` **and** `executor_tools/` together, so GE gets docs lookup, feedback and execution from **one** MCP URL. It reuses the `mcp` service account, which already holds exactly the feedback grants (topic-level publish, Firestore user). |
| **`ailang_bootstrap`** (sunholo-data/ailang_bootstrap) | The install package for agents: a Claude/Codex plugin, skills (`skills/ailang`, `ailang-inbox`, …) and a **local stdio MCP** (`mcp-server/server.js`: `ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_builtins`, `ailang_eval`) that shells out to the local CLI with file paths and caller-chosen `--caps` | The hosted executor uses the **same tool names**, so a model sees one vocabulary locally and in the cloud. The difference is deliberate: hosted tools take **code**, not paths, and callers never choose caps (policy decides). `skills/ailang/SKILL.md` is the teaching content the pasted suggestion called "SKILL.md". For GE it is delivered through `ailang_prompt` and the A2A card skill text, since GE does not install plugins. Follow-up: point the bootstrap README at the hosted endpoint as the zero-install option. |
| **`ailang_only` executor lane** | See below | MCP projects its tools; A2A projects its agent. |

## Problem statement

1. **The lane cannot be reached from outside the fleet.** Tasks arrive only through the message plane, which the coordinator turns into a Cloud Run Job. Nothing translates MCP or A2A into lane calls or lane tasks.
2. **The hosted MCP has no execution tools.** `docker/Dockerfile.mcp` (Cloud Run `terraform/cloud_run_mcp.tf`) serves about 21 read-only docs, stdlib and benchmark tools from `mcp_tools/*.ail`, with no check and no run (V1).
3. **serve-api's A2A is pre-1.0.** It supports only `tasks/send` and `tasks/get`, and `tasks/get` always errors (V3). GE calls `message/stream` and requires `protocolVersion` on the card (R2, R3).
4. **Auth is off on the protocol surfaces.** `--api-key-env` exempts `/mcp/`, `/a2a/` and `/.well-known/` (`internal/apiserver/auth.go:28-35`, V4). Nothing on the apiserver side verifies Google ID tokens (V12).

### What the Gemini-suggested hybrid gets right and wrong

| Suggestion | Reality | Consequence |
|---|---|---|
| "You already have both MCP and A2A deployed on Cloud Run" | Half right. The *executors* are deployed (as Jobs, behind the message plane). The MCP service is docs-only. No serve-api service has A2A enabled. The resident agent's A2A is a per-user coding agent (V1, V6, V13). | P1 and P2 are adapters onto what exists. |
| `ailang_verify(code, contracts)` as a separate tool | Contracts live *in* the code, and `ai-check` already returns type-check plus Z3 as one JSON (V5). | One `ailang_check`. |
| Tag pure tools `readOnlyHint: true` | Already automatic for empty effect rows; effectful tools declare `@mcp_hints` (V7). | No new mechanism. |
| "Multi-step synthesis and self-repair over A2A" | This is the **`ailang_only` executor agent** as it exists today. `daneel-executor` is already an answer-only agent on that lane (V16). | P2 is a gateway, not a new agent loop. |
| "Register MCP as actions in the GE Data Store" | **Documented 2026-10-14 (not yet exercised).** GE has a *custom MCP server data store* connector, Streamable HTTP only, auth "No authentication" or "OAuth 2.0" (auth URL, token URL, client id/secret, scopes, optional PKCE). It needs an org-policy override (`discoveryengine.managed.disableCustomMcpServerConnector`) and the MCP, auth and token FQDNs allow-listed. Redirect URI `https://vertexaisearch.cloud.google.com/oauth-redirect`. `readOnlyHint` tools skip the per-call user confirmation. Source: [GE custom MCP setup](https://docs.cloud.google.com/gemini/enterprise/docs/connectors/custom-mcp-server/set-up-custom-mcp-server), and Mark saw the form in the console. | **CONFIRMED live 2026-10-14:** connector `ailang-mcp-dev` created in `sunholo-gemini-enterprise` after the org-policy override; GE listed all 28 tools (they start Disabled under the Actions tab and must be enabled). |
| GE sends user OAuth plus `X-Serverless-Authorization` | **Documented 2026-10-14.** For default `*.run.app` URLs, GE sends `X-Serverless-Authorization` with a Google-signed ID token for `service-<PROJECT_NUMBER>@gcp-sa-discoveryengine.iam.gserviceaccount.com`. That identity needs `roles/run.invoker`. It is not sent for custom domains. `Authorization` carries the user's OAuth token unchanged. Consequence: the executor MCP can run **IAM-only (no `allUsers`)**, with Cloud Run checking the service agent and the app optionally checking the user token. Same source as above. | **CONFIRMED live 2026-10-14:** with "No authentication" selected, GE (`google-adk/2.11.0`) got 200/202 on 6 calls to the IAM-only service. Cloud Run IAM accepted the discoveryengine service agent; no `allUsers`. |
| "A2A is primarily M2M via OIDC" | The only working GE↔A2A path ran **unauthenticated** (R4). | **PENDING P0-c.** This is the biggest risk. |
| Per-seat vs. central billing decides the protocol | MCP calls cost CPU only. A2A tasks cost **model tokens** (glm-5.3-flash via OpenRouter, metered) plus a Job. Billing is a real axis **for A2A only**. | D5: a budget and a rate limit on the A2A path. |

## Goals

**Primary:** a GE user can (a) get AILANG checked and run inline, and (b) hand off an AILANG task to our `ailang_only` executor and get the result back. Both run inside the lane's existing security boundary.

**Success metrics:**
1. A GE prompt like "write an AILANG function that … and check it" makes Gemini call `ailang_check` / `ailang_run`, and the answer cites the real checker output.
2. A GE delegation ("build me an AILANG program that …") becomes a `ge-executor` task. GE sees streamed progress and receives the agent's answer plus the program as an artifact, within the Cloud Run timeout.
3. Every execution, MCP or A2A, goes through `ailang run --policy` / `ailang policy-tool` in `security_mode = "restricted"`. A fixture that tries `Net`, `Process`, `Env` or an FS write outside the sandbox gets a **named policy refusal**.
4. Neither surface is reachable anonymously, or D3 records the measured reason it is.
5. The A2A path has a hard per-day spend cap and per-caller rate limit (D5), with refusals that say so.

## Design

```
                       Gemini Enterprise
             ┌──────────────┴───────────────┐
        MCP (inline)                    A2A v0.3 (delegate)
             │                               │
             ▼                               ▼
  ailang-executor-mcp (Cloud Run,     A2A gateway (D2: coordinator route
  serve-api, stateless)               or a serve-api service)
  executor.ail ── std/process ──┐            │ POST /api/messages → inbox ge-executor
                                │            ▼
                                │     coordinator ── dispatch ──▶ Cloud Run Job
                                ▼                                 agent-pi / motoko
             ailang run --policy ge-mcp.toml                      tool_policy: ailang_only
             ailang policy-tool --policy ge-mcp.toml              policy: ge-executor.toml
             (restricted, supervised timeout,                     (same tools, same boundary)
              per-request sandbox dir)                               │
                                                                    ▼ completion summary
                                              gateway ◀── task completion ── streamed to GE
```

### P1: MCP — the lane's tools, with Gemini as the harness

An AILANG serve-api program (`mcp_tools/executor.ail`, served by its own Cloud Run service) whose exported functions are thin adapters onto the **same two endpoints the lane's pi and motoko tools call** (V14, V15). It needs no new Go:

| MCP tool | Adapter | Lane tool it mirrors |
|---|---|---|
| `ailang_check(code)` | Writes `code` into a per-request sandbox dir, then `policy-tool` op `check`, or `ailang ai-check` for check + Z3 (V5) | AilangCheck |
| `ailang_run(code, entry, args_json)` | Writes the file, then `ailang run --policy ge-mcp.toml` | AilangRun |
| `ailang_builtins_search(query)`, `ailang_examples_search(query)`, `ailang_docs_search(query)` | `policy-tool` ops `builtins_list` / `docs_search` / examples (V15) | BuiltinsSearch, ExamplesSearch |
| `ailang_syntax()` | The `ailang prompt` teaching text | — (pi/motoko get it in the system role; Gemini can only get it from a tool) |

`ge-mcp.toml` is **stricter than the executor policy**: `allowed_caps = ["IO"]`, no `Process`, no `Net`, `fs_sandbox` = the per-request dir, `timeout_ms` = 60000 (raised from 20000 on 2026-10-14 so real programs fit), and an output cap. The supervisor enforces the timeout; this is the v0.41 hardening (V14).

Calling out to the binary means the executor program itself needs `FS` (to write the request file) and `Process` (to exec `ailang`) and nothing else. It runs under a **dedicated zero-role service account** on Cloud Run gen2.

**Not in P1: the stateful tools** (AilangRead/Edit/Write, AilangCLI). They need a workspace that persists across tool calls, and a stateless MCP service on Cloud Run has none. If GE turns out to need multi-file editing inline, that is the A2A path's job, or a later phase with a per-session workspace.

**Gap G1 — serve-api has no Process allowlist** (V10). The executor program calls only `ailang`, so the exposure is our own code, not user input. Still, close it: the executor runs under serve-api with a policy-file `process_allow = ["ailang:run", "ailang:policy-tool", "ailang:ai-check"]`, reusing the existing `process_allow` key (`cmd/ailang/run_policy.go`) rather than adding a flag. **D4.**

### P2: A2A — the lane's agent, with our harness doing the work

**A new agent `ge-executor`**, shaped like `daneel-executor` (V16), which is already an answer-only agent on this lane:

- `tool_policy: ailang_only`, with `provider` / `executor_variant` set to **motoko** (the lane default since 2026-10-08, V17) and pi as the documented rollback.
- Model `openrouter/z-ai/glm-5.3-flash`, the lane's model.
- **Answer-only:** the completion summary is the answer, and `skip_approval: false` with no diff means no approval card. Artifacts are confined to `answers/**`.
- `timeout` **under 1h** (e.g. 30m), because the prod platform timeout is 1h and an agent timeout above it loses its result.
- `ge-executor.toml` is derived from `ailang-only-executor.toml` but tighter: no `process_allow` git ops, and `fs_deny_write` keeps `.github/**`.
- **Workspace: a dedicated scratch repo** (D6), never `sunholo-data/ailang`. A GE user's task must not be able to read our source or land a PR on it.

**The gateway** turns A2A into a coordinator task and back:

- `message/stream`: `POST /api/messages` to inbox `ge-executor`, with the text parts as the body and a correlation id, then hold the SSE stream open. While the task is running, emit `status-update` events (`submitted` → `working`, with a heartbeat at most every 15s so proxies do not idle it out). On completion, emit one `artifact-update` with the answer text and, if one was written, the program as a `file` part. Then send `completed` / `failed`.
- `message/send`: the same flow; it blocks until completion or returns a `working` task. `tasks/get`: the task status. GE never polls (R2), but other A2A clients do.
- The card: `protocolVersion: "0.3.0"`, `preferredTransport: "JSONRPC"`, `capabilities.streaming: true`, served at `/.well-known/agent.json` **and** `agent-card.json`. The `url` comes from `PUBLIC_BASE_URL`, never `r.TLS` (Aitana F22). Extensions are objects (F24). There is one skill, `ailang-task`, whose description tells GE when to delegate.

**D2 — where the gateway lives:**

- **(a) A coordinator route `/a2a`, recommended.** The coordinator already owns task state, dispatch and completions (`/pubsub/completions`), so the gateway would not need to poll another service's database. The route is scoped to agents flagged `a2a_exposed: true` in config. Cost: a new public surface on the coordinator (quorum trigger #2).
- **(b) A serve-api service.** It isolates the public surface, but it must call the coordinator API and observe completions from outside. It also forces the serve-api A2A v0.3 upgrade (the old P2), which is worth doing anyway, but only on its own merits.

**Known risk — task status truth.** The message-plane trust review (2026-09-25) found that task status can lie: a sweep can age-kill a task, and `blocked` gets dropped. The gateway's `completed` / `failed` must come from the **completion record**, the same source `daneel-executor`'s host reads, not from a status field. It must also time out loudly with `failed: executor did not complete in N min`, never hang the GE stream. See [m-task-status-truth](../m-task-status-truth.md).

### P3: registration and runbook

Follow the Aitana-proven path (R1, R5):

1. `agents-cli register-gemini-enterprise --registration-type a2a --agent-card-url https://<host>/.well-known/agent.json --gemini-enterprise-app-id projects/<NUM>/locations/global/collections/default_collection/engines/<ENGINE> --deployment-target cloud_run`.
2. **Delete before re-registering** (`DELETE …/assistants/default_assistant/agents/<id>`, `v1alpha`, with `X-Goog-User-Project`). Re-running register makes a duplicate and burns the daily quota (F28).
3. Put `iconUrl` on the **same host** as the card `url`.
4. `scripts/verify-ge-executor.sh` ports Aitana's 12 card checks and adds an MCP `tools/list`, an `ailang_check` round trip, a policy-refusal fixture and one A2A `message/stream` round trip against dev.
5. Cloud Run `--timeout=3600` on the gateway, and no response-buffering middleware in front of the SSE path (Aitana: a logging middleware broke streaming).
6. MCP registration follows whatever P0-a finds. If our GE edition has no custom-MCP path, the MCP service still serves Claude, ChatGPT and Cursor, and GE uses A2A only, as Aitana does.

### Agent Gateway route (alternative MCP path, found 2026-10-14)

GE can also reach an MCP server through **Agent Gateway egress**, preview (`v1alpha`). The steps:

1. Register the server in **Agent Registry** (`gcloud agent-registry services create … --mcp-server-spec-type=tool-spec`).
2. Import it into the GE app (`setUpDataConnector`, `mcp_server_source: REGISTRY_MCP`, `use_agent_gateway_egress: true`).
3. Reach Cloud Run privately (`--ingress=internal`) over PSC, with IAP and IAM access policies authorising each call.

It is available in us-central1 and europe-west1. It needs a VPC, a PSC attachment and DNS peering, and it enables gateway routing for all GE traffic, including LLM calls. Source: [codelab](https://codelabs.developers.google.com/agw-ge-custom-mcp-egress-vpc-registry), [Agent Gateway for GE](https://docs.cloud.google.com/gemini-enterprise-agent-platform/govern/gateways/agent-gateway-ge-deploy).

**Recommendation:** use the direct custom-MCP connector with IAM-only Cloud Run first, because it needs no network build-out. Consider the gateway route only if central governance or private ingress is required.

### P0: discovery spike (attended, half a day, gates P2–P3)

Every row is a measurement against GE, recorded with the observed headers and request bodies (**values redacted**):

| ID | Question | How |
|---|---|---|
| P0-a | Does our GE edition accept a **custom MCP server**, and through what (data connector, "actions", the Agent Registry track)? | GE console plus the discoveryengine docs. Aitana's `v6.22.0/gemini-enterprise-agent-platform-adoption.md` says it is planned there too. |
| P0-b | If P0-a = yes, what auth does GE send to MCP? | Register a header-echo serve-api (the `examples/runnable/serve_api_mcp_header_auth.ail` pattern). |
| P0-c | Can a `--deployment-target cloud_run` A2A agent be called **without** `allUsers` invoker? Which principal and which header? | Deploy the echo service with IAM ingress, grant `run.invoker` to `service-<NUM>@gcp-sa-discoveryengine.iam.gserviceaccount.com`, register and invoke. |
| P0-d | Does GE fetch the card with the registrant's credentials or anonymously at call time? | Same echo service, with access logs. |
| P0-e | How long will GE hold an A2A SSE stream before giving up? | Make the echo agent stream heartbeats for 10 / 20 / 40 min. This bounds the `ge-executor` timeout. |

## Design-freeze items (Mark)

- **D1 — Which GE app/project?** The existing Aitana GE engine, or an AILANG-owned one? (The Console wizard and subscription purchase are required for a new one.)
- **D2 — Where the A2A gateway lives:** a coordinator route (recommended) or a serve-api service.
- **D3 — If P0-c shows GE cannot do authenticated A2A:** ship A2A open behind D5's budget and rate limit, or offer GE MCP only?
- **D4 — serve-api Process allowlist** through a policy file (recommended) or a new flag.
- **D5 — A2A spend:** a per-day budget for `ge-executor`, a per-caller rate limit, and who pays when GE is used by an external org. Until this is decided, `ge-executor` is dev-only.
- **D6 — Scratch workspace:** a new empty repo (e.g. `sunholo-data/ge-executor-scratch`), or teach the coordinator a no-repo workspace. The first costs nothing; the second is a coordinator change.

## Conflict surface

This touches no language core. The surfaces it extends:

1. **The coordinator HTTP mux** (D2a): a new public `/a2a` plus card routes beside `/api/messages`, `/instances/*` and `/pubsub/*` (`internal/coordinator/daemon_http.go`). Existing routes are API-key or push-authed; the new one is authed per D3. `/api/messages` stays API-key-only. The gateway calls the dispatch path in-process, not over HTTP with a key.
2. **Agent config:** a new `a2a_exposed` flag. Agents without it are unreachable through A2A, so the default is off. One test: a non-exposed agent's inbox gets JSON-RPC `-32602`, matching the hidden-export refusal of `a2a_dispatch_gate_test.go`.
3. **The `ailang_only` lane:** unchanged. `ge-executor` is a new consumer of the lane with its own policy file; `ailang-only-executor`, `daneel-executor` and the package agents are unaffected.
4. **serve-api's A2A** (`internal/apiserver/a2a.go`, `serveapi/a2a_handler.go`): unchanged under D2a. Under D2b, both copies must move to one shared v0.3 handler, or the embedded copy stays v0.1.
5. **`auth.go:28-35` exemptions:** not changed. The MCP service sits behind Cloud Run IAM or a verified token, per D3.

## Axiom check (summary)

Deterministic: the MCP tools run capability-free or IO-only code under a supervised policy. The A2A path is an LLM agent, which is explicitly non-deterministic, and it is labelled as such in the card skill description. Explicit effects: the executor program's own `FS` and `Process` use is declared in signatures and `@mcp_hints`. Extension over core: no new sandbox, harness or agent loop, only adapters onto the lane. No hard violations of A1/A3/A4/A7.

## Implementation plan

| Phase | Work | Est. |
|---|---|---|
| P0 | Echo-service spike, rows P0-a to P0-e, results written back into the Verification Log | 0.5d attended |
| P1 | `mcp_tools/executor.ail`, `ge-mcp.toml`, Process allowlist (D4), `docker/Dockerfile.executor`, policy-refusal and timeout fixtures, `ailang mcp check` passes; `terraform/cloud_run_executor_mcp.tf` in multivac **dev**, zero-role SA, no `allUsers` | 1.5d |
| P2 | `ge-executor` agent plus `ge-executor.toml` plus template (from `daneel-executor-task.md`); scratch workspace (D6); gateway (D2) with `message/stream`, `message/send`, `tasks/get`, card, completion-record wait and a loud timeout; spend cap and rate limit (D5) | 2.5d |
| P3 | `scripts/verify-ge-executor.sh`, `docs/docs/guides/gemini-enterprise.md`, dev registration, a GE transcript, then a prod promote by the normal path | 1d |

### Files

- `mcp_tools/executor.ail` — new, about 150 LOC
- `docker/Dockerfile.executor` — new, modelled on `docker/Dockerfile.mcp`
- `internal/coordinator/a2a_gateway.go` — new under D2a, about 350 LOC (split if over 500)
- `internal/coordinator/a2a_gateway_test.go` — new: exposure gate, completion-record wait, loud timeout
- `examples/runnable/serve_api_executor_mcp.ail` — new: the minimal adapter pattern
- `docs/docs/guides/gemini-enterprise.md` — new runbook
- `ailang-multivac/config/policies/ge-mcp.toml` — new (sibling repo)
- `ailang-multivac/config/policies/ge-executor.toml` — new (sibling repo)
- `ailang-multivac/config/templates/ge-executor-task.md` — new (sibling repo)
- `ailang-multivac/config/config.cloud.yaml` — the `ge-executor` agent (sibling repo; reaches prod only through the prod branch)
- `ailang-multivac/terraform/cloud_run_executor_mcp.tf` — new (sibling repo)
- `changelogs/unreleased/<date>-ailang-executor-ge.md` — fragment

## Success criteria

- [ ] P0 rows a–e measured and logged, with values redacted
- [ ] MCP: `ailang_check` / `ailang_run` pass `ailang mcp check`; a `Net`/`Process`/`Env`/outside-sandbox-write fixture returns a named policy refusal; a timeout fixture is killed by the supervisor and the server stays healthy
- [ ] A2A: a reference client (a2a-sdk 0.3) `message/stream` against dev yields `working` heartbeats, then an artifact holding the `ge-executor` answer
- [ ] A2A: a forced executor stall produces `failed` with a named reason within the configured cap, never a hung stream
- [ ] A2A: an agent without `a2a_exposed` cannot be reached (-32602)
- [ ] The D5 spend cap refuses with a named reason once exhausted
- [ ] Neither surface can be invoked anonymously, or D3 records the measured reason
- [ ] Registered in GE (dev engine), with a GE transcript linked here for both MCP (if P0-a) and A2A
- [ ] `make test`, `make lint`, `make check-file-sizes` green; changelog fragment present

## Verification log

| # | Claim | Evidence |
|---|---|---|
| V1 | No service accepts AILANG source synchronously over the wire to check or run it | Subagent sweep of `internal/`, `cmd/`, `docker/`, `mcp_tools/`, `ailang-multivac/terraform/`; `Dockerfile.mcp` serves `mcp_tools/*.ail` with `--caps FS,Env`, no check or run tool |
| V2 | The hosted docs MCP is public (`allUsers`), max 10 instances | `ailang-multivac/terraform/cloud_run_mcp.tf:60,141-149` |
| V3 | serve-api A2A supports only `tasks/send` and `tasks/get`, and `tasks/get` always errors; the card URL comes from `r.TLS` | `internal/apiserver/a2a.go:133,135,272`, `:86-105` |
| V4 | `--api-key-env` exempts `/mcp/`, `/a2a/`, `/.well-known/`, `/api/_` | `internal/apiserver/auth.go:28-35` |
| V5 | `ai-check` emits check plus verify as one JSON | `ailang ai-check t.ail` → `{"check":{"passed":true,…},"verify":{"available":true,…}}` (v0.54.9, 2026-10-14) |
| V6 | The resident A2A server does `message/send` (not `message/stream`), serves both card paths and verifies a Google ID token | `docker/resident/server.mjs:107-135,159`; `lib/auth.mjs:46` |
| V7 | A pure tool is auto read-only; an effectful tool declares `@mcp_hints` (`readOnly`/`destructive`/`idempotent`/`openWorld`) | `examples/runnable/serve_api_mcp_hints.ail:3-6`; `serveapi/protocol/tool_hints.go:23` |
| V8 | `std/process.exec` has no shell, a 30s default timeout and a 10MB default output cap | `std/process.ail` header and `:64` |
| V9 | `ailang run` with no `--caps` refuses IO | `effect 'IO' requires capability, but none provided` (2026-10-14) |
| V10 | serve-api has no Process allowlist; with none, `Authorize` does a PATH lookup | `cmd/ailang/serve_api.go` (no process flags); `internal/effects/process_authorize.go` `!pc.HasAllowlist` |
| V11 | No A2A `.ail` example exists | Subagent grep of `examples/` |
| V12 | Nothing in the apiserver reads `X-Serverless-Authorization` | `grep -rn` → only `internal/coordinator/cloud_credit.go` (client side) |
| V13 | The `ailang_only` lane = 8 tools, no shell or native file tools; pi launched with `--no-builtin-tools` | `internal/executor/toolpolicy.go:53`; `internal/executor/pi/toolnames.go:84` (per m-motoko-ailang-only-lane, verified 2026-10-01) |
| V14 | `ailang run --policy` takes caps, net, FS sandbox, entry and limits from the file, refuses widening flags and supervises timeout and output; `ailang-only-executor.toml` uses `security_mode = "restricted"`, `timeout_ms = 120000` | `ailang run --help` (2026-10-14); `ailang-multivac/config/policies/ailang-only-executor.toml` |
| V15 | `policy-tool` is a typed JSON endpoint (read/write/edit root-anchored through fileguard; CLI ops per schema incl. `check`, `builtins_list`, `docs_search`); every refusal is named | `internal/policytool/policytool.go:1-40,142-148`; `cli_ops.go:382-384` |
| V16 | `daneel-executor` is an answer-only `ailang_only` agent (completion summary is the answer; no approval when no diff) | `ailang-multivac/config/config.cloud.yaml` `daneel-executor` block |
| V17 | `ailang-only-executor` runs motoko since 2026-10-08 (pi is the rollback); A/B 2026-10-02 motoko 55/69 vs pi 55/69 | `config.cloud.yaml` `ailang-only-executor` comment block |
| V18 | The coordinator has `POST /api/messages` (API-key) and `/pubsub/completions` | `internal/coordinator/daemon_http.go:72,86,330-336` |
| R1 | The Aitana GE integration is A2A only; MCP never registered with GE | `aitana-labs/platform/docs/integrations/gemini-enterprise.md`; `backend/protocols/mcp_server.py`; `docs/design/v6.22.0/gemini-enterprise-agent-platform-adoption.md` (Planned) |
| R2 | GE uses `message/stream` (SSE) and never polls `tasks/get` | `sunholo-data/gde-ap-agent/docs/design/forks/gde-ap-agent/v0.1.0/a2a-async-task-pattern.md:9-11` |
| R3 | GE rejects a card without `protocolVersion` or with bare-string extensions | Aitana F23, F24 |
| R4 | The proven GE↔A2A path ran unauthenticated | `gde-ap-agent/cloudbuild.yaml:206,237-241`; `gemini-enterprise.md:467-469` |
| R5 | Registration uses `agents-cli register-gemini-enterprise`; re-register makes duplicates | `gemini-enterprise.md:255-261`; F28 |
| PENDING | P0-a…e | Not measurable in-repo. P0 spike |

## Related documents

- [m-agent-ailang-only-execution](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) — the lane this doc exposes.
- [m-motoko-ailang-only-lane](../m-motoko-ailang-only-lane.md) — motoko as the lane harness (the `ge-executor` default).
- [m-serveapi-directory-ready](../v0_51_0/m-serveapi-directory-ready.md) — the distribution umbrella. GE is the third directory, and this doc is its lane.
- [m-mcp-oauth-package](../v0_52_0/m-mcp-oauth-package.md) — the OAuth 2.1 AS for MCP, reused if P0-b shows GE does user OAuth.
- [m-task-status-truth](../m-task-status-truth.md) — why the gateway reads completion records, not status.
- [m-protocol-support](../../implemented/v0_9_0/m-protocol-support.md) — the original serve-api MCP/A2A (v0.1 A2A).
- [m-resident-agent-instances](../m-resident-agent-instances.md) — the per-user A2A coding agent. It is an alternative delegate, but not the default, since it is not on the locked-down lane.
- Aitana: `~/dev/aitana-labs/platform/docs/integrations/gemini-enterprise.md` — the runbook and friction list (F22–F30).

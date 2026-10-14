# M-AILANG-EXECUTOR-GE: an AILANG code executor on Cloud Run, reachable from Gemini Enterprise over MCP and A2A

**Status**: Planned (drafted 2026-10-14, attended). Phase 0 is a discovery spike whose results gate Phases 2–3.
**Target**: v0.55.0
**Priority**: P1. This is the first place outside our own fleet where AILANG runs as a *service*, and Gemini Enterprise (GE) is a channel the Aitana platform has already proven.
**Estimated**: ~5.5 days. P0 spike 0.5d (attended: needs the GE console), P1 MCP executor 1.5d, P2 A2A v0.3 upgrade 2d, P3 registration and runbook 1d, P4 not estimated (separate decision).
**Dependencies**: none blocking. P1 uses only what has shipped. The OAuth lane reuses [m-mcp-oauth-package](../v0_52_0/m-mcp-oauth-package.md) (L4) if P0 shows GE wants OAuth.
**Scope**: an `mcp_tools`-style AILANG program plus a Dockerfile (this repo); `internal/apiserver/a2a.go` (P2); a Cloud Run service in `ailang-multivac/terraform/` (dev first).

**Quorum triggers fired**: **#4.** The load-bearing premises are about Gemini Enterprise, which we do not control: whether GE takes custom MCP servers, and what auth GE sends to a Cloud Run A2A agent. #1 also fires for the design-freeze items D1–D4 below. Not run yet. Run it after P0, so the PENDING rows are measured first (see the design-doc-creator rule on "objections answerable only outside the session").

---

## Problem statement

A GE user should be able to ask Gemini to write, check or run AILANG and get a *checked* answer back, not a hallucinated one. Today none of that path is built. There are three gaps:

1. **No execution service exists.** The hosted MCP (`docker/Dockerfile.mcp`, Cloud Run `ailang-multivac/terraform/cloud_run_mcp.tf`) serves about 21 read-only tools (docs, stdlib, benchmarks) from `mcp_tools/*.ail`. None of them accepts AILANG source and checks or runs it (V1).
2. **A2A on serve-api is pre-1.0.** It supports only `tasks/send` and `tasks/get`, and `tasks/get` always errors (V3). GE calls `message/stream` and requires `protocolVersion` on the card (R2, R3).
3. **Auth is off on exactly these surfaces.** `--api-key-env` exempts `/mcp/`, `/a2a/` and `/.well-known/` (`internal/apiserver/auth.go:28-35`, V4). Nothing on the apiserver side verifies Google ID tokens.

### What the Gemini-suggested hybrid gets right and wrong

The suggestion pasted into the request (MCP for inline tools, A2A for delegated long work) is the right *shape*. Several of its premises do not hold here, though:

| Suggestion | Reality | Consequence |
|---|---|---|
| "You already have both MCP and A2A deployed on Cloud Run" | The MCP is docs-only. A2A is not enabled on any Cloud Run serve-api service. The resident-agent A2A (`docker/resident/server.mjs`) is a per-user *coding agent*, not an executor (V1, V6). | P1 builds the executor tools. |
| `ailang_verify(code, contracts)` as a separate tool | Contracts live *in* the code (`requires`/`ensures`), and `ai-check` already runs type-check plus Z3 into one JSON (V5). | One `ailang_check` tool, not two. |
| Tag pure tools `readOnlyHint: true` | Already automatic: a tool with an empty effect row is advertised read-only, and effectful ones declare `@mcp_hints` (V7). | No new mechanism. |
| "Register MCP endpoints as actions in the GE Data Store" | **Unverified.** The Aitana reference never registered MCP with GE; it is A2A only (R1). | **PENDING P0-a.** |
| GE sends user OAuth in `Authorization` and a Google OIDC token in `X-Serverless-Authorization` for MCP | **Unverified.** No repo has observed this. | **PENDING P0-b.** |
| A2A is "primarily M2M via OIDC" | The only working GE↔A2A integration we have ran **unauthenticated** (`--allow-unauthenticated`, `A2A_INVOCATION_REQUIRE_AUTH=false`). Nobody has confirmed GE works against an auth-required A2A endpoint (R4). | **PENDING P0-c.** This is the biggest open risk. |
| Per-seat vs. central billing decides the protocol | For a stateless executor on our Cloud Run, compute bills to our project either way. Billing becomes a real axis only with per-customer metering, which is out of scope. | Not a decision driver for v1. |

## Goals

**Primary:** a GE user can get AILANG checked and run by a hosted, sandboxed AILANG executor, and the same service is usable from any MCP client (Claude, ChatGPT, Cursor).

**Success metrics:**
1. From a GE chat, a prompt like "write an AILANG function that … and check it" makes Gemini call `ailang_check`, and the response cites the real checker output.
2. From GE, the A2A agent accepts a message containing a fenced `ailang` block and streams back a check+run report (`message/stream`) within Cloud Run's timeout.
3. Untrusted code runs with **zero capabilities**, a wall-clock cap, an output cap and per-request isolation. A fixture that tries `IO`/`FS`/`Net`/`Process` fails with a capability error, not a side effect.
4. Not reachable anonymously. Either IAM ingress or verified tokens, whichever P0 shows GE can satisfy. If GE cannot do authenticated A2A, that is written down as a measured fact and decided (D3), not silently shipped open.

## Design

### One service, two protocol projections

Gemini proposed two Cloud Run services. **We deploy one** — `ailang-executor`, one `serve-api` process with `--mcp-http` and `--a2a`. Both protocols project the *same* exported functions. The split Gemini describes (fast inline vs. delegated long work) is a property of the **tools**, not of the deployment. A second service would double the IAM, deploy and observability surface for nothing.

```
Gemini Enterprise ──MCP (Streamable HTTP, /mcp)──┐
                  ──A2A v0.3 (/a2a, card)────────┤
Claude / ChatGPT / Cursor ──MCP──────────────────┤
                                                 ▼
                      ailang-executor (Cloud Run, serve-api)
                      executor.ail — exported tool functions
                        │ std/fs: write request code to sandbox tmp
                        │ std/process.exec: ailang ai-check | ailang run (caps: none)
                        ▼
                      child `ailang` process — no caps, timeout, output cap
```

### P1: the executor as an AILANG program (no Go changes)

Default bias is an extension, not a core change (PROGRAM.md). Everything P1 needs has already shipped. It needs **no new Go**:

- `std/process.exec(cmd, args)` runs a command with no shell expansion, a timeout (default 30s) and an output limit (default 10MB) (V8).
- `ailang ai-check <file>` emits type-check plus Z3 verify as one JSON (V5).
- `ailang run <file>` with **no `--caps`** refuses every effect: `effect 'IO' requires capability, but none provided` (V9).

So `executor.ail` exports these tools:

| Tool | Does | Hints |
|---|---|---|
| `ailang_check(code)` | Writes `code` to a sandbox temp file, runs `ailang ai-check`, returns its JSON (check + verify + errors with locations). | `@mcp_hints("readOnly", "idempotent")` — effectful (FS, Process) but it never touches the caller's world |
| `ailang_run(code, entry, args_json)` | Writes the file, runs `ailang run --entry <entry> --args-json <..>` **with no caps**, and returns stdout, stderr and the exit code. | `@mcp_hints("readOnly", "idempotent")` |
| `ailang_syntax()` | Returns the `ailang prompt` teaching text (or a pinned section) so Gemini writes valid AILANG before calling check. | pure → automatically read-only |

The existing `mcp_tools/*.ail` docs tools (search docs and stdlib, examples) are **mounted alongside** them in the same service, so Gemini can look things up and also check.

**Why the teaching tool matters more than it looks:** a model with no AILANG prompt writes poor AILANG (`ailang prompt` exists for exactly this). GE does not let us inject a system prompt into Gemini, so the prompt has to reach Gemini through a tool description or a tool result. This is the "SKILL.md" item from the suggestion, delivered in a form GE can consume. The `ailang_check` description tells Gemini to call `ailang_syntax` first.

#### Isolation (the security boundary)

Untrusted code is the threat model. Layers, outermost first:

1. **Cloud Run gen2 sandbox.** It runs under a **dedicated zero-role service account**, never a default compute SA, so the container holds no credentials worth stealing. There are no secrets in env.
2. **A child process per request.** A crash, an infinite loop or a memory blowup kills the child, not the server. `std/process` timeout (set ≤ 20s for `run`, ≤ 30s for `check`) and output cap (≤ 256KB).
3. **No capabilities in the child.** `ailang run` without `--caps` denies IO/FS/Net/Process/Env/AI (V9). This is the language's own guarantee, so it is the one we lean on hardest.
4. **A per-request temp dir** under `AILANG_FS_SANDBOX`, removed afterwards.
5. **Low concurrency** (`containerConcurrency` around 4) and `max_instance_count` around 10, so one heavy request cannot starve others and the bill is bounded.

**Gap G1 — the Process allowlist is not settable on serve-api.** `--process-allowlist` exists only on `ailang run` (`cmd/ailang/main_run.go:107`). `serve-api` has no such flag, and with no allowlist `Authorize` falls back to a PATH lookup (`internal/effects/process_authorize.go`, V10). The executor program only ever calls `ailang`, so the exposure is *our* code calling the wrong binary, not user input. That still means P1 relies on code review, not a guard. Fix in P1: pass the allowlist to serve-api. Prefer the existing policy file route (`run_policy.go` already maps `process_allow`) over a new flag; the v1 simplification program discourages new flags. **D4.**

**Gap G2 — no memory cap on the child.** `AILANG_EVAL_MAX_RSS` is an eval-harness cap, not a `std/process` one. The backstop for v1 is the Cloud Run memory limit (an OOM restarts the instance, which is acceptable at this scale). A per-child RSS cap is a follow-up, not a blocker.

### P2: A2A v0.3 on serve-api (benefits every serve-api service)

GE calls `message/stream` and never polls `tasks/get` (R2). The upgrade to `internal/apiserver/a2a.go`:

- Add `message/send` and `message/stream` (SSE). Keep `tasks/send` for one release behind a deprecation log, then delete it (coding-standards: no back-compat).
- Card: `protocolVersion: "0.3.0"`, `preferredTransport: "JSONRPC"`, `capabilities.streaming: true`, and served at **both** `/.well-known/agent.json` and `/.well-known/agent-card.json` (the resident server already does both, V6). Card `url` from `PUBLIC_BASE_URL`, then `X-Forwarded-Proto`/`X-Forwarded-Host`, never `r.TLS`. Today it builds `http://` behind Cloud Run TLS termination (V3), which is Aitana's friction F22: GE stores the URL and dials it later.
- Parts use the `kind` discriminator (`"kind":"text"|"data"|"file"`).
- Card `capabilities.extensions` entries must be **objects**, not bare strings (Aitana F24).
- Card version comes from the build version, not the hardcoded `"0.8.1"`.

**How a free-text GE message becomes a call.** GE sends natural language, not `metadata.skill_id`. For the executor the A2A dispatch is **deterministic, with no LLM**:

- If the message contains a fenced `ailang` code block, run check and then run, streaming a status event between them. Return the report as a text artifact plus a `data` part with the raw JSON.
- If there is no code block, return a short instruction: "send AILANG in a fenced block, or use the MCP tools". The executor does not synthesize code.

This keeps the executor deterministic, cheap and auditable. The "autonomous self-repair loop" from the suggestion is a **different product** (an LLM agent loop) and is out of scope here; see P4.

`metadata.skill_id` dispatch stays for programmatic A2A callers.

### P3: registration and runbook

Follow the Aitana-proven path (R1, R5):

1. `agents-cli register-gemini-enterprise --registration-type a2a --agent-card-url https://<host>/.well-known/agent.json --gemini-enterprise-app-id projects/<NUM>/locations/global/collections/default_collection/engines/<ENGINE> --deployment-target cloud_run`.
2. **Delete before re-registering** (`DELETE …/assistants/default_assistant/agents/<id>`, `v1alpha`, with `X-Goog-User-Project`). Re-running register makes a duplicate and burns the daily quota (Aitana F28).
3. Put `iconUrl` on the **same host** as the card `url`, or GE shows the default icon.
4. `scripts/verify-executor.sh` ports Aitana's `verify-a2a.sh` 12 card checks and adds MCP `tools/list`, a pure `ailang_check` round trip and a capability-denial fixture.
5. Cloud Run `--timeout=3600` for SSE; no `BaseHTTPMiddleware`-style buffering in front of the SSE path (Aitana: a logging middleware broke streaming).
6. MCP registration follows whatever mechanism P0-a finds. If GE has no custom-MCP path in our edition, MCP stays available to other clients, and GE uses A2A only, as Aitana does.

### P4: delegated synthesis — not in this doc

The suggestion's "A2A for multi-step code synthesis, self-repair, multi-file refactoring" needs an LLM loop, a model choice, quota and a cost owner. We already have two candidate agents: the **resident agent** (`docker/resident/`, A2A `message/send`, Google ID-token auth, V6) and the motoko/pi executors. Exposing one of them to GE is a separate design decision (D2). It should reuse this doc's A2A v0.3 work and auth findings, not fork them.

### P0: discovery spike (attended, half a day, gates P2–P3)

Every row is a measurement against GE, recorded with the observed headers and request bodies:

| ID | Question | How |
|---|---|---|
| P0-a | Does our GE edition accept a **custom MCP server** (and through which UI or API: data connector, "actions", or the Agent Registry track)? | GE console plus the discoveryengine API docs. Aitana's `v6.22.0/gemini-enterprise-agent-platform-adoption.md` says it is planned there too. |
| P0-b | For MCP (if P0-a = yes), what does GE send? `Authorization` (user OAuth?) and `X-Serverless-Authorization`? | Register a header-echo serve-api (`examples/runnable/serve_api_mcp_header_auth.ail` pattern) and log the headers, **values redacted**. |
| P0-c | For A2A with `--deployment-target cloud_run`, can GE call a service **without** `allUsers` invoker? Which principal (`service-<NUM>@gcp-sa-discoveryengine.iam.gserviceaccount.com`?) and which header? | Deploy the echo service with IAM ingress, grant `run.invoker` to the candidate SA, register and invoke. |
| P0-d | Does GE fetch the card with the registrant's credentials (at registration) or anonymously (at call time)? | Same echo service, with access logs. |

## Design-freeze items (Mark)

- **D1 — Which GE app/project?** Register in the existing Aitana GE engine, or create an AILANG-owned one? (The Console wizard is required for creation, and the subscription is purchased in the Console.)
- **D2 — P4 owner.** Is GE-delegated synthesis wanted at all, and if so is it the resident agent, motoko or neither?
- **D3 — If P0-c shows GE cannot do authenticated A2A:** ship A2A open with app-level rate limiting (the executor holds no secrets and grants no caps), or ship MCP-only to GE?
- **D4 — How serve-api gets a Process allowlist:** a policy file (recommended) or a new flag.

## Conflict surface

This touches `internal/apiserver/` (P2), not the language core.

1. **Positions extended:** the A2A JSON-RPC method switch (`a2a.go:133-139`) and the card builder (`a2a.go:86-105`).
2. **What else lives there:** every `serve-api --a2a` service (docparse via `terraform/docparse.tf` if it enables A2A; the embedded `serveapi/a2a_handler.go:82,98` copy, also `tasks/send`-only), and the dispatch-gate tests (`a2a_dispatch_gate_test.go`), which assert that hidden exports are unreachable (-32602).
3. **Disambiguation:** method name only. `message/*` and `tasks/send` do not overlap.
4. **Must still work:** `a2a_dispatch_gate_test.go` (the hidden-export refusal must hold for `message/send` too, so extend the test rather than copying it), `metadata.skill_id` dispatch, and the card at `/.well-known/agent.json`.
5. **Intentional changes:** card `url` scheme and host source; card version string; part discriminator `type` → `kind` (the old form is accepted on input for one release); `tasks/send` deprecated.
6. **The two-implementation seam:** `internal/apiserver/a2a.go` and the embedded `serveapi/a2a_handler.go` must not diverge. **P2 makes one shared handler**, or the second copy keeps v0.1 forever (per "verify the seam, not the artifact").

The auth exemption in `auth.go:28-35` is **not** changed in P1–P3. The executor relies on Cloud Run IAM or a verified token in front, per D3. Changing the exemption affects every serve-api service and needs its own row if P0 says we need it.

## Axiom check (summary)

Deterministic: the executor adds no nondeterminism, because check and run of capability-free code are deterministic. Explicit effects: the executor's own FS and Process use is declared in signatures and hints. Extension over core: P1 has zero Go; P2 improves a shared surface rather than adding a new one. No hard violations of A1/A3/A4/A7.

## Implementation plan

| Phase | Work | Est. |
|---|---|---|
| P0 | Echo-service spike, rows P0-a to P0-d, results written back into this doc's Verification Log | 0.5d attended |
| P1 | `mcp_tools/executor.ail` (or `executor/` dir), `docker/Dockerfile.executor`, Process allowlist (D4), capability-denial and timeout fixtures, `ailang mcp check` passes; terraform `cloud_run_executor.tf` in multivac **dev** with the zero-role SA, no `allUsers` | 1.5d |
| P2 | A2A v0.3 in a shared handler (apiserver plus embedded), card fixes, fenced-block dispatch, SSE; `examples/runnable/serve_api_a2a.ail` (no A2A example exists today, V11) | 2d |
| P3 | `scripts/verify-executor.sh`, registration runbook `docs/docs/guides/gemini-enterprise.md`, dev registration, then a prod promote via the normal path | 1d |

### Files

- `mcp_tools/executor.ail` — new, about 150 LOC, the tool functions
- `docker/Dockerfile.executor` — new, modelled on `docker/Dockerfile.mcp`
- `internal/apiserver/a2a.go` — v0.3 methods, card, SSE (+ about 300 LOC; split if it passes 500)
- `serveapi/a2a_handler.go` — collapse onto the shared handler
- `internal/apiserver/a2a_dispatch_gate_test.go` — extend to `message/send`
- `examples/runnable/serve_api_a2a.ail` — new
- `docs/docs/guides/gemini-enterprise.md` — new runbook
- `ailang-multivac/terraform/cloud_run_executor.tf` — new (sibling repo)
- `changelogs/unreleased/<date>-ailang-executor-ge.md` — fragment

## Success criteria

- [ ] P0 rows a–d measured and logged, with values redacted
- [ ] `ailang_check` and `ailang_run` pass `ailang mcp check`; pure tools advertised read-only
- [ ] Capability-denial fixture: code using `IO`/`FS`/`Net`/`Process` returns a capability error from `ailang_run`
- [ ] Timeout fixture (`let rec loop…`) returns a Timeout error within the cap, and the server stays healthy
- [ ] A2A `message/stream` from a reference client (a2a-sdk 0.3) streams status then artifact; the card validates with `protocolVersion` and object extensions
- [ ] The service is not invokable anonymously, or D3 records the measured reason it is
- [ ] Registered in GE (dev engine). A GE chat produces checker-backed AILANG, with a screenshot or transcript linked here
- [ ] `make test`, `make lint`, `make check-file-sizes` green; changelog fragment present

## Verification log

| # | Claim | Evidence |
|---|---|---|
| V1 | No service accepts AILANG source over the wire to check or run it | Subagent sweep of `internal/`, `cmd/`, `docker/`, `mcp_tools/`, `ailang-multivac/terraform/` found no check or run tool; `Dockerfile.mcp` serves `mcp_tools/*.ail` (docs/stdlib/benchmarks) with `--caps FS,Env` only |
| V2 | The hosted MCP is public (`allUsers` invoker), max 10 instances | `ailang-multivac/terraform/cloud_run_mcp.tf:60,141-149` |
| V3 | serve-api A2A supports only `tasks/send` and `tasks/get`, and `tasks/get` always errors | `internal/apiserver/a2a.go:133,135,272`; card URL built from `r.TLS` and `r.Host` at `a2a.go:86-105` |
| V4 | `--api-key-env` exempts `/mcp/`, `/a2a/`, `/.well-known/`, `/api/_` | `internal/apiserver/auth.go:28-35` (read 2026-10-14) |
| V5 | `ai-check` emits type-check plus verify as one JSON | `ailang ai-check t.ail` → `{"check":{"passed":true,…},"verify":{"available":true,…}}` (run 2026-10-14, v0.54.9) |
| V6 | The resident A2A server does `message/send` (not `message/stream`), serves both card paths and verifies a Google ID token on every route except `/livez` | `docker/resident/server.mjs:107-135,159`; `docker/resident/lib/auth.mjs:46` |
| V7 | A pure tool is auto read-only; an effectful tool declares `@mcp_hints` | `examples/runnable/serve_api_mcp_hints.ail:3-6,19-21`; `serveapi/protocol/tool_hints.go:55-80` |
| V8 | `std/process.exec` has no shell, a timeout (30s default) and an output cap (10MB default) | `std/process.ail` header and `exec` at `:64` |
| V9 | `ailang run` with no `--caps` refuses IO | `ailang run t.ail` → `effect 'IO' requires capability, but none provided` (run 2026-10-14) |
| V10 | serve-api has no `--process-allowlist`; with no allowlist, `Authorize` does a PATH lookup | `grep -n -i process cmd/ailang/serve_api.go` → only a memory-limit comment; `internal/effects/process_authorize.go` `!pc.HasAllowlist` branch |
| V11 | No A2A `.ail` example exists | Subagent grep of `examples/` for "a2a" — no hits outside `.ailang/cache` |
| V12 | Nothing in the apiserver reads `X-Serverless-Authorization` | `grep -rn X-Serverless-Authorization internal` → only `internal/coordinator/cloud_credit.go` (client side) |
| R1 | The Aitana GE integration is A2A only; MCP was never registered with GE | `aitana-labs/platform/docs/integrations/gemini-enterprise.md`; `backend/protocols/mcp_server.py` (no auth, no annotations); `docs/design/v6.22.0/gemini-enterprise-agent-platform-adoption.md` (Planned) |
| R2 | GE uses `message/stream` (SSE) and never polls `tasks/get` | `sunholo-data/gde-ap-agent/docs/design/forks/gde-ap-agent/v0.1.0/a2a-async-task-pattern.md:9-11` |
| R3 | GE rejects a card without `protocolVersion`, and rejects bare-string extensions | Aitana frictions F23, F24 in `gemini-enterprise.md` |
| R4 | The proven GE↔A2A path ran unauthenticated; authenticated GE A2A is unconfirmed | `gde-ap-agent/cloudbuild.yaml:206,237-241`; `gemini-enterprise.md:467-469` |
| R5 | Registration uses `agents-cli register-gemini-enterprise`; re-register makes duplicates | `gemini-enterprise.md:255-261`; F28 |
| PENDING | P0-a…d (GE MCP support; MCP and A2A auth headers; card fetch identity) | Not measurable in-repo. P0 spike |

## Related documents

- [m-serveapi-directory-ready](../v0_51_0/m-serveapi-directory-ready.md) — the distribution umbrella (Anthropic/OpenAI directories). GE is the third directory; this doc is its lane.
- [m-mcp-oauth-package](../v0_52_0/m-mcp-oauth-package.md) — the OAuth 2.1 AS for MCP, reused if P0-b shows GE does user OAuth.
- [m-protocol-support](../../implemented/v0_9_0/m-protocol-support.md) — the original MCP and A2A design (the v0.1 A2A this doc upgrades).
- [m-resident-agent-instances](../m-resident-agent-instances.md) — the A2A coding agent; the P4 candidate.
- [m-serveapi-protocol-only-module](../m-serveapi-protocol-only-module.md) — the protocol package; P2's shared handler must not conflict with it.
- Aitana: `~/dev/aitana-labs/platform/docs/integrations/gemini-enterprise.md` — the runbook and friction list (F22–F30).

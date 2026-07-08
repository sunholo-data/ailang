# M-WASM-AI-STEP-VIA-MESSAGES: WASM `ai.step` mediated by the cloud message bus

**Status**: Planned
**Target**: v0.19.0 (after M-WASM-CLOUD-MESSAGES M1)
**Priority**: P2 (the BYO-key sister doc M-WASM-AI-STEP-BYO-KEY ships first as the low-friction default)
**Estimated**: ~5 days (~40 hours)
**Dependencies**: M-WASM-CLOUD-MESSAGES M1 (the messages bus must already work), M-COORDINATOR-INBOX-WILDCARDS (so a `pkg:ai-step-proxy` inbox can route to a server agent)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-13
**Source**: 2026-05-13 conversation. The user wanted both transport options: BYO-key direct fetch (low latency, low control) AND message-bus-mediated (higher latency, centralized control). This doc covers the second.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Network latency variable; same as direct fetch |
| A2: Replayability | +1 | Every `ai.step` call leaves a record on the message bus — the request and response are inspectable, replayable, auditable |
| A3: Effect Legibility | +1 | `! {AI}` annotation unchanged; new transport is invisible to AILANG-side code |
| A4: Explicit Authority | +1 | Server-side ai-step-proxy holds the provider key; browser only knows its own coordinator API key |
| A5: Bounded Verification | 0 | Zero type-system change |
| A6: Safe Concurrency | 0 | Send-and-await semantics; single-flight per call |
| A7: Machines First | +1 | Same agent code runs server- or browser-side; AI calls magically route through the bus when in browser |
| A8: Minimal Syntax | +1 | Zero new syntax |
| A9: Cost Visibility | +2 | Cost tracking centralized at the proxy — BIG win over BYO-key model where users can't aggregate spend |
| A10: Composability | +1 | Composes with M-WASM-CLOUD-MESSAGES infrastructure |
| A11: Structured Failure | +1 | Returns `Result[StepResult, AIError]` — same shape; transport errors map to structured AIError variants |
| A12: System Boundary | +1 | Browser ↔ proxy boundary explicit; proxy ↔ provider boundary controlled server-side |

**Net Score: +9** → **Decision: ✅ Move forward**

### Hard Violation Check
- [x] A1, A3, A4, A7 — none violated

---

## Problem Statement

The sister doc M-WASM-AI-STEP-BYO-KEY makes `ai.step` work in browser via direct fetch with the user's provider API key. That's the right default for low-friction demos. But it has structural drawbacks that make it wrong for production browser-AILANG agents:

| Concern | BYO-key direct fetch | Why it matters |
|---|---|---|
| **CORS** | Limited providers (OpenAI rejects, Anthropic needs special header, OpenRouter works) | Production needs reliable provider access |
| **Cost tracking** | Distributed (each user pays their own provider) | Can't observe "what is my AILANG agent fleet costing" centrally |
| **Caching** | Per-user (each browser maintains its own); cache hits limited to one tab | Centralized prompt caching across users yields huge savings |
| **Provider key management** | User-managed (lose-the-key risk; rotation requires every user to update) | Operator wants to rotate one key, not n keys |
| **Audit / replay** | None — calls go straight to provider, no record | Can't replay an agent's reasoning trace later |
| **Tool dispatch on Process effects** | Impossible — browser sandbox blocks Process | If your agent NEEDS BashExec, omnigraph, mcp, you can't run it browser-side at all |

The message-bus-mediated path solves all of these by routing `ai.step` calls through a server-side ai-step-proxy that holds the provider keys, runs Process-effect tools, tracks cost/cache centrally, and returns the StepResult to the browser via the cloud message bus.

**Trade-off accepted**: ~one message round-trip latency per `ai.step` call (~1-2s with polling, <100ms with M-WASM-CLOUD-MESSAGES M2 streaming). For interactive agents this is real friction but acceptable; the up-side (production-grade control) is worth it for the use cases this targets.

**Current state:**
- M-WASM-CLOUD-MESSAGES M1 (when shipped) gives browser AILANG `messages.send` + `messages.poll_inbox` over the cloud bus — that's the transport this design layers on.
- M-AGENT-MCP M7.1 (shipped) proves the request/response-over-bus pattern works for `submit_feedback`.
- Server-side ailang-coordinator already dispatches Cloud Run Jobs from messages — the ai-step-proxy would be one more such agent.
- Existing `ai.step` server-side path is fully working.

**Impact:**
- Without this, browser-side AILANG agents can't access non-browser tools (Process, FS native).
- Without this, production browser deployments can't centralize cost or audit.
- Without this, agents that NEED OpenAI (and don't want a CORS proxy per deployment) can't run browser-side.

---

## Goals

**Primary Goal:** AILANG-WASM `ai.step` (and `stepWithCache` / `stepWithStream`) work in browser via send-and-await over the cloud message bus, transparently — same `ai.step(...)` call site, same `Result[StepResult, AIError]` return type, just a different transport under the hood.

**Success Metrics:**
- A browser-side AILANG agent calls `ai.step("anthropic/claude-sonnet-4.5", msgs, tools)`. The call returns a real `Result[StepResult, AIError]`. The actual provider call happened on the server-side ai-step-proxy. Browser never saw the provider API key.
- Server-side ai-step-proxy runs as a normal AILANG Cloud Run Job, dispatched from a `pkg:ai-step-proxy` inbox via the existing coordinator infrastructure.
- Cost tracking aggregates centrally — `ailang dashboard` shows total spend across all browser-AILANG sessions.
- An agent that uses `tool_calls` requiring server-side Process effects (BashExec, etc.) can run browser-side: the AI call routes through the proxy, AND tool dispatch can ALSO route through the bus (M3 follow-up; M1 only routes ai.step).
- Latency: <2s per ai.step call with polling, <300ms with WebSocket streaming (M-WASM-CLOUD-MESSAGES M2).
- Auth: browser only knows its coordinator API key. Provider keys live in the ai-step-proxy's environment, never browser-side.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Request/response message envelope shape | Becomes a stable wire contract; hard to change post-deploy | human | design | high |
| Whether tools dispatch on browser-side OR proxy OR routed via bus | Three modes possible; each has different security implications | human | design | high |
| Synchronous-looking AILANG facade vs explicit async API | Hides the async nature (cleaner code) but pretends a network call is local | human | design | high |
| Single proxy agent vs proxy-per-provider | One agent simpler; per-provider isolates blast radius | agent | runtime | med |
| Default timeout for the bus round-trip | Too short = false-positive failures; too long = slow error paths | agent | compile | low |
| Whether the proxy supports streaming chunks back over bus | M-WASM-CLOUD-MESSAGES M2 streaming makes this feasible; without M2 it's per-token bus messages (chatty) | human | design | med |

### Design Freeze
- [ ] **Wire envelope** — proposed `ai_step_request_v1` / `ai_step_response_v1` schemas locked
- [ ] **Tool dispatch model** — M1 ships with "tools execute on the proxy" (server-side dispatch). Browser-side tool dispatch via bus = M3 follow-up.
- [ ] **AILANG facade** — synchronous-looking. Browser AILANG calls `step(...)` and the WASM impl blocks (via Promise) on the bus round-trip. Same model as `awaitJSResult` today.

---

## Solution Design

### Overview

When `ai.step` is called in browser-WASM AILANG, the WASM-side handler:
1. Constructs an `ai_step_request_v1` message envelope with model + msgs + tools
2. Calls `messages.send("ai-step-proxy", envelope)` (using M-WASM-CLOUD-MESSAGES infra)
3. Awaits a reply with matching `correlation_id` via `messages.poll_inbox` (or `subscribe_inbox` if M2 streaming is available)
4. Decodes the response envelope into `Result[StepResult, AIError]`
5. Returns the result, looking exactly like a normal local `ai.step` to the AILANG caller

The server-side ai-step-proxy is a normal AILANG agent registered with the coordinator (via M-COORDINATOR-INBOX-WILDCARDS-style routing) that:
1. Subscribes to `pkg:ai-step-proxy` inbox
2. On each message, decodes the request envelope
3. Calls `ai.step(model, msgs, tools)` server-side (real provider call)
4. Encodes the StepResult into an `ai_step_response_v1` envelope
5. Sends it back to the originating inbox (browser polls / subscribes to that inbox)

### Architecture

```
┌────────────── Browser (WASM AILANG) ──────────────┐
│                                                    │
│  AILANG agent code:                               │
│   match ai.step("anthropic/...", msgs, tools) {  │
│     Ok(result) => render(result),                  │
│     Err(e)     => report(e)                        │
│   }                                                │
│                                                    │
│  ↓ via _ai_step builtin                           │
│                                                    │
│  WasmAIHandler.Step (this design):                │
│   - Build envelope: {model, msgs, tools, corr_id} │
│   - messages.send("ai-step-proxy", envelope)      │
│   - poll/subscribe inbox for matching corr_id     │
│   - Decode response → Result[StepResult, AIError] │
│                                                    │
└────────────────────────────────────────────────────┘
                       │
                       ▼ HTTP via M-WASM-CLOUD-MESSAGES
┌────────────── ailang-coordinator ──────────────┐
│  POST /api/messages → Pub/Sub                  │
└────────────────────────────────────────────────┘
                       │
                       ▼ Pub/Sub fan-out
┌────────────── ai-step-proxy (Cloud Run Job) ──────────────┐
│                                                            │
│  AILANG agent:                                            │
│   on_message(envelope) {                                  │
│     let result = ai.step(envelope.model,                  │
│                          envelope.msgs,                    │
│                          envelope.tools);                  │
│     messages.send(envelope.reply_to, encode_response(     │
│       result, envelope.correlation_id))                    │
│   }                                                        │
│                                                            │
│  Effects: ai.step makes REAL provider call with           │
│  ANTHROPIC_API_KEY / OPENAI_API_KEY from env.             │
│  Cost tracked via existing dashboard infra.               │
│                                                            │
└────────────────────────────────────────────────────────────┘
                       │
                       ▼ messages.send back
                  (response message)
                       │
                       ▼
              Browser polling / subscribed inbox
                       │
                       ▼
           Decoded into Result[StepResult, AIError]
                       │
                       ▼
              Returned to the calling AILANG code
```

### Components

1. **Wire envelope schemas** (~30 LOC AILANG types):
   ```ailang
   type AiStepRequestV1 = {
     correlation_id: string,
     reply_to: string,           -- inbox to send response to
     model: string,
     messages: [Message],
     tools: [ToolSchema],
     cache_breakpoints: [CacheBreakpoint]   -- empty if not stepWithCache
   }
   type AiStepResponseV1 = {
     correlation_id: string,
     result: Result[StepResult, AIError]
   }
   ```

2. **WASM handler implementation** (~100 LOC in cmd/wasm/effects.go):
   - Replaces stubs with send-and-await logic
   - Uses `messages.send` + `messages.poll_inbox` from M-WASM-CLOUD-MESSAGES
   - Generates correlation_id (uuid v4 or timestamp+random)
   - Polls with timeout; on timeout returns `Err({code: "timeout", retryable: true})`
   - Discards messages with non-matching correlation_id (stays in inbox for other handlers)

3. **Server-side ai-step-proxy agent** (~150 LOC AILANG):
   - Registered with coordinator via existing agent-registry mechanism
   - Inbox `pkg:ai-step-proxy` (or `ai-step-proxy:<region>` for sharding)
   - Decodes incoming envelope, calls real `ai.step`, encodes response, sends back to `reply_to`
   - Handles errors gracefully — provider errors map to `AIError` variants in the response
   - Per-key rate limiting (reuses existing IPRateLimiter; keyed on the originating user's coordinator API key)

4. **Coordinator config update** (~20 LOC YAML):
   - New agent entry for `ai-step-proxy` with appropriate model/timeout/concurrency settings
   - Deployed alongside other v0.19.0 agent updates

5. **Demo** (`examples/wasm-step-via-messages/`):
   - Same `agent.ail` as the BYO-key demo (no AILANG-side changes!)
   - Different `index.html` — wires `messages.send/poll` instead of `ailangSetAIStepHandler`
   - Demonstrates that the SAME AILANG code works under both transports

### Implementation Plan

**Phase 1 — Envelope schemas + server-side proxy** (~2 days)
- [ ] Day 1: Define `AiStepRequestV1` / `AiStepResponseV1` types in a shared module (`std/ai_messages.ail` or similar). Document the JSON-on-the-wire shape.
- [ ] Day 2: Server-side ai-step-proxy AILANG implementation. Deploy as a new Cloud Run Job; register agent in coordinator config.

**Phase 2 — WASM handler implementation** (~2 days)
- [ ] Day 3: Replace `WasmAIHandler.Step` stub with send-and-await impl using `messages.send` + `messages.poll_inbox` (assumes M-WASM-CLOUD-MESSAGES M1 already shipped). Correlation ID generation. Timeout logic.
- [ ] Day 4: `StepWithCache` (same shape + cache_breakpoints in envelope). `StepWithStream` (M2 follow-up — needs streaming bus support).

**Phase 3 — Demo + docs** (~1 day)
- [ ] Day 5: `examples/wasm-step-via-messages/` demo. Side-by-side with BYO-key demo to show both transports work identically from AILANG-side code.

### Files to Modify/Create

**New files:**
- `std/ai_messages.ail` (~40 LOC) — wire envelope types (AiStepRequestV1, AiStepResponseV1)
- `services/ai-step-proxy/` (NEW directory):
  - `proxy.ail` (~150 LOC) — the server-side agent
  - `ailang.toml` — package manifest
  - `_smoke.ail` — boot probe (per M-EXT-PORTABILITY-GATE convention)
  - `Dockerfile` — Cloud Run deploy
  - `README.md` — operator deploy guide
- `examples/wasm-step-via-messages/index.html` (~80 LOC) — demo page
- `examples/wasm-step-via-messages/agent.ail` (~80 LOC) — same shape as BYO-key demo
- `examples/wasm-step-via-messages/README.md` — explain both transports
- `docs/docs/guides/wasm-ai-step-via-messages.md` (~150 LOC) — full guide + when to choose this over BYO-key

**Modified files:**
- `cmd/wasm/effects.go` (+100 LOC): replace stubs with messages-bus-mediated impls
- Coordinator agent configs (`gs://ailang-multivac-{dev,test,prod}-ailang-config/config.yaml`) — add ai-step-proxy entry
- `changelogs/v0.10-current.md` (~60 LOC entry)

---

## Examples

### Example 1: Same agent code, different transport

```ailang
-- agent.ail — works with EITHER M-WASM-AI-STEP-BYO-KEY or M-WASM-AI-STEP-VIA-MESSAGES
import std/ai (step, Message, ToolSchema)
import std/dom (innerHTML)

export func main() -> () ! {AI, DOM} = {
  let msgs: [Message] = [
    { role: "user", content: "Summarize https://example.com",
      tool_calls: [], tool_call_id: "" }
  ];
  let tools: [ToolSchema] = [/* ... */];
  match step("anthropic/claude-sonnet-4.5", msgs, tools) {
    Ok(result) => innerHTML("#out", result.message.content),
    Err(e)     => innerHTML("#out", "error: " ++ e.message)
  }
}
```

```html
<!-- BYO-key wiring -->
<script>
  ailangSetAIStepHandler(async (model, msgs, tools) => {
    /* direct fetch to anthropic */
  });
</script>

<!-- vs messages-bus wiring -->
<script>
  // No ailangSetAIStepHandler call — WASM defaults to messages-bus transport
  // (configured via env or build flag)
  // The same step() call automatically routes via:
  //   messages.send("ai-step-proxy", {model, msgs, tools, correlation_id, reply_to: "user-X"})
  //   messages.poll_inbox("user-X") until matching correlation_id arrives
</script>
```

### Example 2: Cost tracking aggregates centrally

With messages-bus transport, every browser AILANG agent's `ai.step` call leaves a record on the bus. The `ailang dashboard` shows aggregate spend across all browser sessions — impossible with the BYO-key model where each user's spend goes to their own provider account.

---

## Success Criteria

- [ ] Browser AILANG `ai.step("anthropic/claude-sonnet-4.5", ...)` returns a real StepResult, with the provider call having actually happened on server-side ai-step-proxy
- [ ] Browser never sees the provider API key (verified via network inspector)
- [ ] Server-side ai-step-proxy deployed as a normal Cloud Run Job, registered with coordinator
- [ ] Cost tracking visible in `ailang dashboard` for browser AILANG sessions
- [ ] `ai.stepWithCache` works (cache_breakpoints flow through envelope)
- [ ] Latency < 2s per call with polling; documented as expected for this transport
- [ ] Demo `examples/wasm-step-via-messages/` deploys to docs site
- [ ] Same AILANG agent.ail compiles and runs under BOTH transports (BYO-key + messages-bus) — proves the abstraction
- [ ] Per-key rate limiting prevents abuse
- [ ] Provider errors map cleanly to AILANG `AIError` variants (auth, rate_limit, timeout, etc.)

---

## Testing Strategy

**Unit tests:**
- Envelope serialization round-trip — AILANG types ↔ JSON ↔ AILANG types
- Server-side proxy: mock provider call; verify response envelope shape
- WASM-side handler: mock `messages.send/poll_inbox`; verify correlation_id matching

**Integration tests:**
- Local: run ai-step-proxy + dev coordinator + browser demo end-to-end
- Real provider: ai-step-proxy makes actual Anthropic call; verify cost-tracking entry appears in dashboard

**Manual:**
- Verify provider API key never appears in browser network tab
- Verify rate-limit kicks in after threshold
- Verify timeout fires gracefully (kill the proxy mid-call; browser should see Err timeout)

---

## Deferred Decisions

- **Tool dispatch routing** — M1 has tools execute on the proxy (server-side). M3 follow-up adds option to route specific tool dispatches back to the browser via bus (e.g., for browser-only tools like `dom.querySelector`). Recommendation: defer.
- **Streaming chunk transport** — M2 of M-WASM-CLOUD-MESSAGES adds WebSocket streaming. Until then, `stepWithStream` either falls back to polling (chatty per chunk) or buffers the full stream and returns at end (loses streaming UX). Recommend: error in WASM with "stepWithStream needs M-WASM-CLOUD-MESSAGES M2; use step() until then" until M2 ships.
- **Sharding the proxy** — single global ai-step-proxy vs per-region. Lean: single global for simplicity; shard if/when load demands.
- **Caching at the proxy** — the proxy could maintain its own cache across users (huge cache-hit savings for common prompts). Out of scope for M1; would need careful cache-key design (don't leak one user's content to another via cache).

---

## Non-Goals

- **Replacing the BYO-key path** — this is COMPLEMENTARY. Apps choose transport at boot. BYO-key remains the right default for low-friction demos.
- **Sub-second latency** — the round-trip via Pub/Sub + Cloud Run Job is fundamentally seconds-class. Polling can never beat this. WebSocket (M-WASM-CLOUD-MESSAGES M2) gets it to ~100-300ms but never <100ms.
- **Offline operation** — browser must be online + reach the coordinator. No offline queue.
- **Tool dispatch in M1** — tools execute on the proxy. Hybrid (some tools browser, some server) is M3.
- **End-to-end encryption of envelope contents** — messages on the bus are operator-readable. Don't put sensitive data in prompts. (Same trust model as today's coordinator.)

---

## Conflict Surface

**No parser/typechecker code touched.** This is runtime + new server agent + minor WASM bridge.

- `cmd/wasm/effects.go` — replace 3 stubs (Step / StepWithCache / StepWithStream) with messages-bus-mediated impls. WASM-only build. Risk: if M-WASM-AI-STEP-BYO-KEY also replaces them, last-writer-wins; need to coordinate. Mitigation: select impl at runtime based on whether `ailangSetAIStepHandler` was called (BYO-key wins) or fall back to messages-bus.
- `services/ai-step-proxy/` — entirely new server-side agent. No conflict; pure addition.
- Coordinator config updates — additive (new agent entry).
- `std/ai_messages.ail` — new types module. No conflict.

**Programs that MUST still work** (regression fixtures):
1. Server-side `ai.step` calls — unchanged (different code path: server-side `ai_step.go`)
2. CLI `ailang messages` — unchanged
3. Existing M-WASM-CLOUD-MESSAGES messages.send/poll_inbox API — unchanged (this design layers on top)
4. M-WASM-AI-STEP-BYO-KEY direct-fetch path — unchanged; runtime selector picks based on which JS hook was registered

---

## Timeline

**Days 1-2:** Envelope schemas + server-side ai-step-proxy
**Days 3-4:** WASM handler impl + Step/StepWithCache via bus
**Day 5:** Demo + docs

**Total: ~5 days, runs after M-WASM-CLOUD-MESSAGES M1 ships**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Latency unacceptable for interactive UX | Med | Polling baseline ~1-2s; M-WASM-CLOUD-MESSAGES M2 streaming reduces to ~300ms; document expectations |
| ai-step-proxy becomes a single point of failure | High | Cloud Run autoscales; deploy across regions; per-region instances for HA |
| Cost-tracking exposes which users called what | Low (privacy) | Aggregate-only views in dashboard; per-user views require auth |
| Proxy fans out N concurrent provider calls and hits provider rate limits | Med | Per-source-key rate limiting at the proxy; backpressure via 429 → AIError |
| BYO-key and messages-bus transports race for the same Step stub | Low | Runtime selector: if `ailangSetAIStepHandler` was called, use BYO; else fall back to messages-bus. Deterministic. |
| Operator forgets to deploy ai-step-proxy → all browser ai.step calls fail | High | Coordinator agent registry health check warns if agent inbox has no consumers |
| Cache-coordination across users leaks cross-user content | High (privacy) | M1 has no cross-user cache; defer cache-at-proxy to a later sprint with explicit cache-key design |

---

## Related Documents

**Sister doc (parallel implementation, complementary)**:
- [M-WASM-AI-STEP-BYO-KEY](./m-wasm-ai-step-byo-key.md) — direct fetch transport. Lower latency, lower control. Apps choose at boot which transport.

**Hard dependencies:**
- [M-WASM-CLOUD-MESSAGES](./m-wasm-cloud-messages.md) — M1 must ship first. This design layers on its messages.send/poll_inbox API.
- [M-COORDINATOR-INBOX-WILDCARDS](./m-coordinator-inbox-wildcards.md) — needed so the `pkg:ai-step-proxy` inbox can route to the new server agent without per-deployment config.

**Builds on:**
- M-AGENT-MCP M7.1 (HTTP→Pub/Sub via submit_feedback) — proven request/response-over-bus pattern
- M-AI-PROMPT-CACHING (v0.18.4) — cache_breakpoints surface that flows through the envelope
- M-AI-STEP-STREAMING (v0.18.7) — StepWithStream that needs M-WASM-CLOUD-MESSAGES M2

---

## Future Work

- **M3: Hybrid tool dispatch** — route specific tool calls back to the browser via bus (e.g., `dom.*` tools execute browser-side, `BashExec` executes proxy-side)
- **M4: Cache-at-proxy with cross-user safety** — design cache key including user-id so cache hits don't leak content
- **M5: Multi-region proxy with edge routing** — pick the closest proxy instance for each browser; reduces latency
- **M-WASM-AI-STEP-CHOICE** — runtime selector that picks BYO-key vs messages-bus per call (e.g., based on cost-tracking config or model availability)

---

**Document created**: 2026-05-13
**Last updated**: 2026-05-13

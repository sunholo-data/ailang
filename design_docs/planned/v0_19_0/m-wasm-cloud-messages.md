# M-WASM-CLOUD-MESSAGES: AILANG-WASM browser participates in the cloud message bus

**Status**: Planned
**Target**: v0.19.0
**Priority**: P1
**Estimated**: ~2 weeks (M1 Path A polling) + ~1 week (M2 Path B WebSocket)
**Dependencies**: M-EXT-PORTABILITY-GATE (v0.18.11), M-AGENT-MCP M7.1 (HTTP→Pub/Sub via submit_feedback)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-13
**Source**: Conversation 2026-05-13 exploring whether WASM-compiled AILANG can run motoko-style agents in a browser. Two paths considered — (a) BYO-API-key direct fetch to providers, (b) message-bus-mediated via existing AILANG cloud infrastructure. Path (b) chosen as the architecturally compelling story: reuses entire existing cloud agent infrastructure (coordinator, Cloud Run Job dispatch, Pub/Sub) — browser becomes a "thin client over the message bus, but the thin client is itself written in AILANG."

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Browser timing introduces wall-clock variation but message contents are deterministic; no new nondeterminism in AILANG semantics |
| A2: Replayability | +1 | Inbox messages are replayable from the message store; browser can re-render from any inbox snapshot |
| A3: Effect Legibility | +1 | New `Net` (HTTP) + `DOM` effects are explicit at every call site; no hidden network or DOM mutation |
| A4: Explicit Authority | +1 | API key is explicit input (localStorage opt-in by user); coordinator's `requireAPIKey` middleware enforces; no ambient browser→cloud trust |
| A5: Bounded Verification | 0 | No type-system changes; new stdlib types are local Records/ADTs |
| A6: Safe Concurrency | 0 | Browser is single-threaded; polling/streaming use existing AILANG-WASM goroutine model (single goroutine per js.FuncOf callback) |
| A7: Machines First | +1 | Same agent code runs server- or browser-side — reduces "rewrite the frontend in JS" tax that would otherwise force every AILANG cloud agent to maintain a separate JS SDK |
| A8: Minimal Syntax | +1 | Zero new syntax — pure stdlib + WASM-runtime additions |
| A9: Cost Visibility | 0 | No new cost surface; existing per-API-key rate-limiting carries over |
| A10: Composability | +1 | `std/messages` and `std/dom` compose with all existing effects (`Net`, `Clock`, `Stream`); pattern matches the existing `ai.call` + `httpRequest` precedent |
| A11: Structured Failure | +1 | All new APIs return `Result[T, E]` with typed error variants (`SendFailed`, `Unauthorized`, `RateLimited`, `Timeout`); browser sees the same structured errors a CLI client sees |
| A12: System Boundary | +1 | Browser ↔ cloud crossings are explicit (`messages.send`, `messages.poll_inbox`); no implicit RPC; same `/api/messages` boundary the dashboard already uses |

**Net Score: +8** → **Decision: ✅ Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (network latency is observable, not hidden)
- [x] A3 (Effects): No hidden side effects — every network call carries `! {Net}`, every DOM mutation carries `! {DOM}`
- [x] A4 (Authority): No ambient access — API key must be set before any send/poll
- [x] A7 (Machines First): Reducing one JS SDK per cloud agent is exactly the machine-friendly direction

---

## Problem Statement

AILANG today has two disjoint compilation targets:

1. **CLI/server build** — full `ai.step`, `Process`, `FS`, real network access. Used by motoko_agent, eval-runner, ailang-coordinator, every Cloud Run Job.
2. **WASM build** — runs in browser via `cmd/wasm/main.go`. Has the AILANG language fully (parser, type system, evaluator) but `ai.step` / `Process` / native FS are stubbed out (return errors).

This split forces a hard architectural choice for any AILANG cloud agent that wants a web frontend: **rewrite it in JavaScript**, or accept that the frontend can't speak AILANG at all. Today's collaboration-hub dashboard, the docs site's interactive examples, the registry browser — all are React+TypeScript that talks to the AILANG cloud's HTTP/WebSocket APIs. Every new AILANG cloud agent adds a new JS SDK to maintain.

**Specific consequence today**: motoko_agent is a sophisticated AILANG-native agent loop with rich state, tool dispatch, compaction, profile management. None of that translates to a "try motoko in your browser" experience because:

- WASM build can't run `ai.step`
- Even if it could (BYO-key direct-fetch), the agent's *useful* tools (BashExec, omnigraph, mcp-call.mjs) require server-side resources

A "browser frontend that participates in the AILANG cloud" — sending real messages to real server-side agents over the same Pub/Sub bus — would let WASM AILANG be a **thin client over the message bus, while remaining fully AILANG**. The same code patterns motoko uses to send `ailang messages send eval-suite "run benchmarks"` from a Cloud Run Job would work from a browser tab.

**Current State:**
- AILANG-WASM build exists (~7MB) and runs in browsers; supports pure code, IO-light effects, `httpRequest` via fetch, `WebSocket` via std/stream.
- `cmd/wasm/effects.go` has comprehensive WASM↔JS bridging (records, lists, ADTs, closures, Promises) — about 350 LOC, well-factored.
- `ai.step` / `ai.stepWithCache` / `ai.stepWithStream` all stub out in WASM with explicit error returns ("requires server-side orchestration").
- HTTP→Pub/Sub bridge already shipped via M-AGENT-MCP M7.1 (`submit_feedback`) and `POST /api/messages` (`internal/coordinator/daemon_http.go:50-373`).
- `GET /api/messages?inbox=X&since=Y` exists for polling reads.
- `ailang messages` CLI uses local SQLite (laptop) or Pub/Sub (`AILANG_STORAGE=gcp`) — no WASM-friendly transport today.

**Impact:**
- Anyone wanting to demo motoko (or any AILANG cloud agent) on a webpage today must build a separate React frontend.
- The AILANG language story is incomplete: "write your agent in AILANG" — *but if you want a UI, learn React.*
- Compounding: every new AILANG cloud service adds a new JS SDK. Today: collaboration-hub UI, dashboard, registry-validator UI, future motoko TUI replacement, future eval-runner UI. Each one duplicates the message-bus connection code.

---

## Goals

**Primary Goal:** AILANG-WASM programs running in a browser can `messages.send(...)` to the cloud and `messages.poll_inbox(...)` for replies, using the same APIs and message bus as server-side agents — no new JS SDK required per use case.

**Success Metrics:**
- A "motoko-lite" demo at `examples/wasm-cloud-chat/` runs end-to-end: user types prompt → browser AILANG sends → server-side motoko_agent responds → browser AILANG renders response in DOM. No backend code is specific to this demo.
- WASM binary delta < 100KB (from current ~7MB).
- AILANG-WASM build can `messages.send/list/poll_inbox` against the dev coordinator (`POST /api/messages`) using a localStorage API key — same pattern existing AILANG demos use for AI provider keys.
- The same `chat.ail` source compiles to (a) a CLI binary that runs against local SQLite, (b) a WASM blob that runs against the cloud — modulo the message-store backend choice.
- Path B WebSocket fanout adds a `messages.subscribe_inbox(inbox, callback)` helper with the same call-site shape as `messages.poll_inbox` — chat latency drops from ~1-2s to <100ms.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Messages API surface in `std/messages` (`send`, `list`, `poll_inbox`, later `subscribe_inbox`) | Becomes public stdlib API; renaming or reshape breaks every browser AILANG client | human | design | high |
| Auth via `X-API-Key` header from localStorage (no cookies, no OAuth) | Decides the entire browser-side trust model; cookies would force same-origin which conflicts with hosting WASM on docs site + coordinator on different origin | human | design | high |
| Polling first (Path A) vs streaming first (Path B) | Polling ships in M1 with zero new server code; streaming requires new endpoint + connection management | human | design | med |
| New `std/dom` stdlib module vs raw `ailangSetEffectHandler("DOM", {...})` JS bridge | Stdlib module = stable API + types + cross-environment story; raw bridge = lighter but every demo redefines DOM API | human | design | med |
| WASM-only build-tag for `std/messages` HTTP impl, OR runtime backend selection (CLI: SQLite/Pubsub, WASM: HTTP) | Build-tag is simpler; runtime selection enables CLI clients to hit the HTTP API too (potentially useful for IDE plugins) | agent | compile | med |
| Per-API-key rate-limit reuse vs new "browser-message" rate-limiter | Reuse keeps one rate-limiter; new one allows different limits for browser-class clients | agent | runtime | low |

### Design Freeze

Before sprint-executor begins:

- [ ] **Messages API surface** finalized — exact function signatures for `send/list/poll_inbox` (and `subscribe_inbox` as M2 placeholder)
- [ ] **Auth model** confirmed — `X-API-Key` header from localStorage; no cookies; no OAuth in M1
- [ ] **Polling cadence default** chosen — recommend 1.5s; configurable via `Clock` effect
- [ ] **`std/dom` minimal API** confirmed — exact signature list (proposal: `innerHTML`, `appendChild`, `setText`, `addEventListener`, `getElementValue`)

---

## Solution Design

### Overview

Three layers, ordered by build cost (cheap → expensive):

1. **WASM-side stdlib** (`std/messages`, `std/dom`): pure AILANG modules with WASM-targeted effect handlers in `cmd/wasm/effects.go`. The CLI build of `std/messages` errors loudly with "use `ailang messages` CLI for local-mode" so the same source compiles both ways without fatal mistakes.
2. **No new server code in M1** — leverage the existing `POST /api/messages` and `GET /api/messages?inbox=X` endpoints on the ailang-coordinator that ship today.
3. **Optional WebSocket fanout in M2** — `GET /api/messages/stream?inbox=X` upgrades the connection and pushes new messages as they arrive on the relevant Pub/Sub topic.

### Architecture

```
┌──────────────── Browser ────────────────┐
│  AILANG WASM (your chat.ail)            │
│                                          │
│  ┌─ std/messages ─────┐                 │
│  │ send(inbox, body)  │ ──HTTP POST──┐  │
│  │ list(inbox)        │ ──HTTP GET──┐│  │
│  │ poll_inbox(inbox)  │             ││  │
│  │ [M2: subscribe]    │             ││  │
│  └────────────────────┘             ││  │
│                                      ││  │
│  ┌─ std/dom ──────────┐             ││  │
│  │ innerHTML(sel,html)│             ││  │
│  │ addEventListener   │             ││  │
│  └────────────────────┘             ││  │
└──────────────────────────────────────┘│  │
   localStorage["AILANG_API_KEY"]     │  │
                                      ▼  ▼
                              ┌──────────────────┐
                              │ ailang-coordinator│
                              │   (existing)     │
                              │                  │
                              │ POST /api/messages│ → msgStore → Pub/Sub (ailang-cascade or messages topic)
                              │ GET  /api/messages│ ← msgStore (Firestore)
                              │ [M2:/stream WS]  │ ← Pub/Sub subscription per connection
                              └──────────────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │ Cloud Run Jobs   │ — server-side motoko, eval-runner, etc.
                              │ (existing)       │   pick up messages, do work, publish completions
                              └──────────────────┘
```

### Components

1. **`std/messages.ail`** (new, ~80 LOC): AILANG-side wrappers that take typed arguments and return `Result[T, MessageError]`. Effects: `! {Net}` for send/list/poll, `! {Net, Clock}` for `subscribe_inbox` (M2 — uses a callback dispatched on each chunk).

2. **`std/dom.ail`** (new, ~40 LOC): minimal browser-DOM API — `innerHTML(selector, html)`, `appendChild(parent_selector, child_html)`, `setText(selector, text)`, `addEventListener(selector, event, handler)`, `getElementValue(selector) -> string`. Effects: `! {DOM}`. Build-tag-gated to avoid pulling DOM types into CLI builds.

3. **WASM effect handlers in `cmd/wasm/effects.go`** (~150 LOC additive): registers JS-side implementations for the new `messages` and `DOM` operations. Reuses the existing `registerJSEffectHandler` infrastructure — same pattern as today's `ailangSetEffectHandler`.

4. **`examples/wasm-cloud-chat/`** (new): the headline demo. `index.html` loads the WASM blob + JS shim that wires `messages.send → fetch /api/messages` and `dom.innerHTML → element.innerHTML`. `chat.ail` is the AILANG code that drives it: read user input, send to server-motoko inbox, poll for replies, render to DOM.

5. **(M2 only) `/api/messages/stream` endpoint** on the coordinator (~120 LOC in `internal/coordinator/daemon_http.go`): WebSocket upgrade, opens a Pub/Sub subscription scoped to the connection's authenticated inbox, pipes incoming messages over the WebSocket. Reuses the existing Pub/Sub client and auth middleware.

### Implementation Plan

**M1: Polling messages + DOM (Path A)** (~10 days)

- [ ] **Day 1-2**: `std/messages.ail` API design + types. Decide exact return shapes for `Result[(), MessageError]` and `Result[[InboxMessage], MessageError]`. Lock in the InboxMessage record shape (mirrors existing `messaging.InboxMessage` in Go).
- [ ] **Day 2-3**: WASM-side handler in `cmd/wasm/effects.go`. Wire `messages.send` to fetch POST. Wire `messages.list` and `messages.poll_inbox` to fetch GET. Tests: register a mock JS callback in unit tests, assert AILANG → JS arg shapes.
- [ ] **Day 3-4**: `std/dom.ail` + WASM-side handlers. Same pattern as messages. JS shim layer translates to `document.querySelector` etc.
- [ ] **Day 4-5**: CLI-side stub for `std/messages` — errors loudly with "use `ailang messages` CLI" or routes to an HTTP backend if `AILANG_API_URL` env is set.
- [ ] **Day 5-7**: `examples/wasm-cloud-chat/` demo. `index.html` + JS shim + `chat.ail`. Integrate with dev coordinator (`https://ailang-dev-coordinator-...run.app/api/messages`).
- [ ] **Day 7-8**: Docs page `docs/docs/guides/wasm-cloud-chat.md`. Walk through deploying your own. Cover the API key model, CORS setup on the coordinator (already permissive), and the polling vs streaming decision.
- [ ] **Day 8-10**: End-to-end test against dev coordinator. Browser sends, server-side motoko_agent receives + responds, browser DOM updates with response. Capture screenshot for changelog.

**M2: WebSocket streaming (Path B)** (~5 days, separate sprint)

- [ ] **Day 1**: Design `/api/messages/stream` endpoint contract. Auth model (initial header + API key), scope to single inbox, server keeps Pub/Sub subscription open per connection.
- [ ] **Day 2-3**: Coordinator implementation. Reuse existing Pub/Sub client + WebSocket library (already used for collaboration-hub events). Connection cleanup on disconnect.
- [ ] **Day 3-4**: `std/messages.subscribe_inbox(inbox, callback)` — same call-site shape as `poll_inbox` but callback fires on each message instead of polling. Reuses the WASM `js.FuncOf` callback bridge for AILANG closures (already shipped, line 167 of effects.go).
- [ ] **Day 4-5**: Update demo to use streaming. Latency should drop from ~1-2s to <100ms. Document fallback to polling when WebSocket unavailable.

### Files to Modify/Create

**New files:**
- `std/messages.ail` (~80 LOC) — AILANG-side public API
- `std/dom.ail` (~40 LOC) — browser-DOM API
- `examples/wasm-cloud-chat/index.html` (~80 LOC) — demo HTML + JS shim
- `examples/wasm-cloud-chat/chat.ail` (~100 LOC) — the AILANG agent code
- `examples/wasm-cloud-chat/README.md` (~50 LOC) — deploy-your-own guide
- `docs/docs/guides/wasm-cloud-chat.md` (~150 LOC) — full guide

**Modified files:**
- `cmd/wasm/effects.go` (+150 LOC) — new `messages` + `DOM` effect handlers
- `cmd/ailang/messages.go` (+30 LOC) — add CLI-build stub for `std/messages` calls (errors with helpful message)
- `internal/messaging/config.go` (+15 LOC) — add `api_url` field for browser SDK to discover coordinator endpoint
- `changelogs/v0.10-current.md` (~80 LOC entry under [v0.19.0])
- `docs/docs/guides/index.md` (+5 LOC link)

**M2 additional:**
- `internal/coordinator/daemon_http.go` (+120 LOC) — `/api/messages/stream` WebSocket endpoint
- `cmd/wasm/effects.go` (+50 LOC) — `messages.subscribe_inbox` handler

---

## Examples

### Example 1: Browser sends a chat message to server-motoko

**Before** (today, no path forward — would need React + TypeScript + custom message-bus client):

```jsx
// Many lines of React + TypeScript SDK code...
```

**After** (with M-WASM-CLOUD-MESSAGES):

```ailang
-- chat.ail (loads in browser via WASM)
module chat

import std/messages (send, poll_inbox, MessageError)
import std/dom (innerHTML, addEventListener, getElementValue)
import std/result (Result, Ok, Err)
import std/clock (sleep)
import std/io (println)

func render_message(msg: InboxMessage) -> () ! {DOM} =
  innerHTML("#chat-history", "<div class='msg'>${msg.content}</div>")

func handle_send_click() -> () ! {Net, DOM} =
  let user_text = getElementValue("#input") in
  match send("server-motoko", user_text) {
    Ok(_)  => innerHTML("#status", "sending..."),
    Err(e) => innerHTML("#status", "error: ${e.message}")
  }

export func main() -> () ! {Net, DOM, Clock} {
  addEventListener("#send-button", "click", \_. handle_send_click());

  -- Polling loop for replies
  let last_seen = ref 0;
  loop {
    match poll_inbox("user-mark", last_seen) {
      Ok(msgs) => for msg in msgs { render_message(msg); last_seen := msg.id },
      Err(_)   => ()
    };
    sleep(1500)
  }
}
```

### Example 2: API key bootstrapping (mirrors existing AILANG-demo pattern)

```html
<!-- examples/wasm-cloud-chat/index.html -->
<script>
  // Same pattern existing AILANG WASM demos use for AI provider keys
  let apiKey = localStorage.getItem("AILANG_API_KEY");
  if (!apiKey) {
    apiKey = prompt("Enter your ailang-coordinator API key");
    localStorage.setItem("AILANG_API_KEY", apiKey);
  }

  // Wire JS-side messages handler to fetch the existing /api/messages endpoint
  ailangSetEffectHandler("messages", {
    send: async (inbox, content) => {
      const resp = await fetch("https://ailang-dev-coordinator.../api/messages", {
        method: "POST",
        headers: { "X-API-Key": apiKey, "Content-Type": "application/json" },
        body: JSON.stringify({ inbox, title: "browser-chat", content,
                               from: "user-mark", category: "general" })
      });
      return { _ctor: resp.ok ? "Ok" : "Err", _fields: [...] };
    },
    poll_inbox: async (inbox, since) => {
      const resp = await fetch(`.../api/messages?inbox=${inbox}&since=${since}`,
        { headers: { "X-API-Key": apiKey } });
      const data = await resp.json();
      return { _ctor: "Ok", _fields: [data.messages] };
    }
  });
</script>
```

---

## Success Criteria

- [ ] `examples/wasm-cloud-chat/` end-to-end demo working: browser → cloud → server motoko → cloud → browser DOM update
- [ ] `std/messages.send` + `std/messages.poll_inbox` + `std/messages.list` shipped as stdlib (WASM impl; CLI impl errors with "use `ailang messages` CLI" hint)
- [ ] `std/dom.innerHTML` + `appendChild` + `setText` + `addEventListener` + `getElementValue` shipped
- [ ] WASM binary size delta < 100KB measured against `bin/ailang.wasm` baseline
- [ ] API key auth via `X-API-Key` header working against dev coordinator's `requireAPIKey` middleware
- [ ] Per-API-key rate limiter prevents abuse (reuses `IPRateLimiter` infrastructure, just keyed differently)
- [ ] Docs page `docs/docs/guides/wasm-cloud-chat.md` walks through deploying your own
- [ ] Demo screenshot in changelog showing browser-rendered conversation with server-side motoko
- [ ] All existing AILANG WASM demos still work (regression — `ai.call` BYO-key demos unchanged)
- [ ] M2 WebSocket streaming deferred to follow-up sprint; M1 ships polling-only

---

## Testing Strategy

**Unit tests (Go side):**
- `cmd/wasm/effects.go::messagesSend` — register a mock JS callback, assert AILANG → JS arg shapes (inbox, title, content, from, category).
- `cmd/wasm/effects.go::messagesPollInbox` — assert JS response → AILANG `Result[[InboxMessage], MessageError]` round-trip.
- `cmd/wasm/effects.go::domInnerHTML` — assert `(selector, html)` → JS callback invocation order.

**Integration tests:**
- `examples/wasm-cloud-chat/integration_test.html` — Playwright/Puppeteer-driven: load the page, set API key, send a message, mock the coordinator response, assert DOM updates.
- Manual test against dev coordinator — full round-trip with real Pub/Sub + Cloud Run Job dispatch.

**Manual testing:**
- Verify WASM binary size delta < 100KB after build (`du -h bin/ailang.wasm` before/after).
- Verify CORS on dev coordinator allows the docs-site origin (`*.ailang.sunholo.com`).
- Verify graceful failure when API key is invalid (browser shows "unauthorized", doesn't crash).

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Polling cadence configuration** — agent may choose between hardcoded 1500ms, env-var override, or runtime config via a `messages.set_polling_interval(ms)` API. Recommendation: hardcoded with a comment noting it's adjustable.
- **Error variant granularity in `MessageError`** — agent may collapse some HTTP error codes into umbrella variants (e.g., 503/504 both map to `Unavailable`) or expose them all distinctly. Suggest collapsing for V1 ergonomics.
- **`std/dom` API surface beyond the minimum** — agent may add `setStyle`, `addClass`, `removeClass` if they're needed for the demo. Defer larger DOM APIs (form serialization, fetch-from-DOM, event-delegation patterns) to a future M-WASM-DOM-RICH sprint.
- **JS shim packaging** — agent may ship the JS shim as inline `<script>` in `index.html`, a separate `.js` file, or an npm-publishable module. Recommend separate file for reusability.
- **CLI `std/messages` behavior** — agent may either error loudly OR route to HTTP if `AILANG_API_URL` is set. Either is reasonable; lean toward erroring + suggesting the CLI for now.

---

## Non-Goals

**Not attempted in this feature:**

- **`ai.step` in WASM** — separate concern. M-WASM-AI-STEP-BRIDGE (planned, not in this sprint) covers BYO-key direct-fetch path.
- **Browser-side `Process` effect** — fundamentally blocked by browser sandbox; not happening.
- **Browser-side native `FS`** — IndexedDB/OPFS adapter is its own project; not in scope.
- **OAuth / SSO auth** — M1 uses API key in localStorage only. OAuth is a future iteration.
- **Sub-100ms latency in M1** — polling is 1-2s by design. Streaming (M2) gets <100ms.
- **Cross-origin without CORS opt-in** — browser limitations; coordinator already has CORS configured for the docs origin.
- **Offline / queue-when-offline** — no IndexedDB-backed retry queue. If the network drops, messages fail loudly.
- **Multi-tab coordination** — multiple tabs polling the same inbox each get their own copy of new messages. No leader-election or shared subscription.

---

## Conflict Surface

**Touches no parser/typechecker code.** This is a pure runtime + stdlib addition.

- `cmd/wasm/effects.go` — extends WASM-only effect registry. Build-tag gated (`//go:build js && wasm`). Risk: bloats WASM binary. Mitigation: <100KB target enforced by manual size check; the only new code is the effect-handler dispatch wiring, no new dependencies.
- `std/messages.ail` (new) — needs to handle BOTH local-CLI and WASM contexts. Two reasonable choices: (a) WASM-only stdlib, CLI build errors at use site; (b) runtime backend selection. Decision deferred to implementer; recommend (a) for v1.
- `std/dom.ail` (new) — net new stdlib module. Risk: stdlib creep. Mitigation: explicit "browser-only" effect; CLI builds get the same module but it errors on use (similar to how `ai.step` errors in WASM today).
- `internal/coordinator/daemon_http.go` — for M2 WebSocket fanout, add `/api/messages/stream` endpoint. Risk: extra connection load on coordinator (one persistent connection per active browser). Mitigation: M2 is a separate sprint; can defer until M1 is in production and we have load data.

**Programs that MUST still work** (regression fixtures):

1. `ailang messages list/send/read` CLI commands — unchanged. The new `std/messages` module errors at compile time in the CLI build, so existing CLI usage is untouched.
2. Existing `POST /api/messages` / `GET /api/messages` from Cloud Run Job submitters — unchanged (browser becomes a new client class, same endpoints).
3. AILANG-WASM build size — `bin/ailang.wasm` < current size + 100KB. Locked via manual measurement during M1 acceptance.
4. Existing AILANG WASM demos (the `ai.call` BYO-key ones) — still work with the same `ailangSetAIHandler` pattern; new effects are additive.
5. The existing collaboration-hub UI (React) and dashboard (React) keep working — they read from the same `/api/messages` endpoints; this design adds a new client class, doesn't replace anything.

---

## Timeline

**M1 — Path A polling** (10 days, ~80 hours):

- Days 1-3: API design, types, `std/messages.ail` skeleton + WASM bridge
- Days 4-5: `std/dom.ail` + WASM bridge
- Days 6-7: `examples/wasm-cloud-chat/` demo
- Days 8-9: Integration test against dev coordinator
- Day 10: Docs + changelog + screenshot capture

**M2 — Path B WebSocket** (5 days, ~40 hours, SEPARATE SPRINT):

- Days 1-2: `/api/messages/stream` endpoint on coordinator
- Days 3-4: `std/messages.subscribe_inbox(inbox, callback)` WASM handler
- Day 5: Migrate demo to streaming, document fallback

**Total: ~15 days across 2 sprints.**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| WASM binary bloats > 100KB | Med | Hard size check in CI; the new code is pure JS-callback dispatch (no new Go deps) |
| CORS misconfigured for docs origin | High (blocks demo) | Coordinator already has CORS configured for known origins; docs origin needs to be added explicitly. Verify before demo ships. |
| API key in localStorage is XSS-vulnerable | Med | Same risk as every existing AILANG WASM demo; documented in onboarding flow; not introducing new risk class. |
| Polling load on coordinator scales poorly with many browsers | Med | Per-API-key rate limiter caps abuse; M2 streaming reduces load for active sessions; production rollout staged via feature flag if needed. |
| `std/dom` API growth becomes unmanageable | Low | Lock M1 to 5 functions; refer larger DOM needs to a future M-WASM-DOM-RICH sprint. |
| User mistakenly uses `std/messages` in CLI build expecting it to work | Med | CLI build errors at compile/use time with helpful message ("use `ailang messages` CLI for local-mode"); CHANGELOG documents the split clearly. |
| Browser AILANG agent leaks API key in error logs | Low | Coordinator's `requireAPIKey` middleware already redacts in logs; `MessageError` types must NOT include the key in error messages. |
| Server-side agents can't tell browser-from-cloud-job senders apart, mishandle untrusted input | Med | Coordinator already validates message content; browser messages get the same treatment as Cloud Run Job messages. The `from` field in InboxMessage records origin transparently. |

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_18_11/m-ext-portability-gate.md](../../implemented/v0_18_11/m-ext-portability-gate.md) — the agent-portability theme this builds on; same "AILANG-native everywhere" thesis
- [design_docs/implemented/v0_9_0/m-cloud-e2e-fixes.md](../../implemented/v0_9_0/m-cloud-e2e-fixes.md) (0.43) — coordinator HTTP API foundation
- [design_docs/implemented/v0_9_2/m-cloud-progress-tracking.md](../../implemented/v0_9_2/m-cloud-progress-tracking.md) (0.42) — completion-event fanout pattern (M2 will resemble)
- [design_docs/implemented/v0_7_0/m-collab-provider-stats.md](../../implemented/v0_7_0/m-collab-provider-stats.md) (0.42) — existing dashboard UI pattern this displaces with AILANG-WASM

**Planned (overlaps to coordinate):**
- [design_docs/planned/v0_19_0/m-coordinator-inbox-wildcards.md](./m-coordinator-inbox-wildcards.md) — companion v0.19.0 cloud-arch sprint; both touch the agent registry / inbox routing surface
- [design_docs/planned/v1_0_0/global-collaboration-hub.md](../v1_0_0/global-collaboration-hub.md) (0.45) — long-range vision this is a step toward
- [design_docs/planned/v0_15_0/m-cascade-observability.md](../v0_15_0/m-cascade-observability.md) (0.41) — observability for the same coordinator/Pub/Sub stack

**Companion future doc (out of scope here):**
- M-WASM-AI-STEP-BRIDGE — the OTHER browser-AILANG path (BYO-key direct fetch). Smaller scope, ships standalone. Not blocked by this; the two are complementary.

---

## References

- [Design Axioms](/docs/references/axioms) — net +8 score above
- [M-AGENT-MCP M7.1 implementation report](https://github.com/sunholo-data/ailang/blob/main/changelogs/v0.10-current.md) — proves the HTTP→Pub/Sub bridge pattern
- [`internal/coordinator/daemon_http.go:50-373`](../../../internal/coordinator/daemon_http.go) — the existing endpoints this design leverages
- [`cmd/wasm/effects.go`](../../../cmd/wasm/effects.go) — the existing WASM↔JS bridge being extended
- [`internal/messaging/config.go`](../../../internal/messaging/config.go) — message-store config

---

## Future Work

- **M-WASM-DOM-RICH** — fuller `std/dom` API: form serialization, custom events, intersection observer, history API
- **M-WASM-OFFLINE-QUEUE** — IndexedDB-backed retry queue when network drops
- **M-WASM-OAUTH** — replace localStorage API key with OAuth flow for end-user-facing demos
- **M-AILANG-AS-FRONTEND** — broader push to make every AILANG cloud service ship a WASM-AILANG default frontend instead of a per-service React SDK
- **M-WASM-MULTI-TAB** — leader-election across tabs so only one tab maintains the WebSocket / polling connection
- **M-CLI-MESSAGES-HTTP** — let the CLI `ailang messages` also speak HTTP (talk to a remote coordinator from a laptop), unifying the transport story

---

**Document created**: 2026-05-13
**Last updated**: 2026-05-13

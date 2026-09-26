# M-SERVEAPI-WS-BRIDGE: a serve-api WebSocket route with a per-frame AILANG verdict

**Status**: IMPLEMENTED (M1–M5), 2026-09-25. See [Decisions](#decisions-mark-2026-09-25) and the [Implementation Report](#implementation-report). design-quorum was not run.
**Target**: v0.44.0
**Priority**: P1. The consumer is Daneel's live voice/avatar page (daneel repo, M-DANEEL-LIVE), whose sprint assumes this lands "within days". Mark sets the final priority.
**Estimated**: 5–6 working days across M0–M5. The core relay (M1+M2) is about 3 days.
**Dependencies**:
- **Hard prerequisite:** [m-serveapi-bind-host-cors](m-serveapi-bind-host-cors.md) (written in parallel). serve-api binds `*:port` and sends `Access-Control-Allow-Origin: *` by default, so without it this route would publish a credential-bearing relay on every interface.
- Reuses: [M-STREAM-BIDI](../v0_8_1/m-stream-bidi-primitives.md) (v0.8.1), [M-ASYNC-IO](../v0_9_0/m-async-io-stream.md) (v0.9.0), the serve-api per-call `Fork`, and the `StreamTransport` seam (S2 M2).
- Relates to: [m-trace-label-aware](../../planned/v0_36_0/m-trace-label-aware.md), which does **not** block this design. The design keeps the credential out of AILANG values, so it does not depend on trace redaction.

**Revisits**: [M-STREAM-SERVE-API (REJECTED 2026-02-16)](../../rejected/m-stream-serve-api-proxy.md)
**Requester**: Daneel, message `inbox_1790359269771_90811127`, 2026-09-25
**Author**: design-doc-creator, attended session 2026-09-25, `dev` = `62608138d` (dirty tree, which is other agents' work; none of it touches the files cited here)
**Planner-Lane**: opus-required. It touches the parser, the effects core, serve-api and credential handling.

---

## TL;DR

The design has three parts, and only the second is new language surface:

1. **`@route("WS", "/path")`.** serve-api accepts the WebSocket upgrade and calls the handler **once for the life of the connection**. The browser leg arrives as an ordinary `std/stream` `StreamConn`. One connection gets one forked evaluator and one cloned `EffContext`. This machinery already exists for every serve-api call.
2. **`std/stream/bridge.bridge(client, upstream, init, step)`.** This is a Go-driven fold. The AILANG `step : (s, BridgeFrame) -> (s, Verdict) ! {e}` runs **once per frame in each direction** and returns `Forward | Drop | ReplaceText | ReplaceBin | CloseBridge`. Go moves the bytes. AILANG decides what happens to each frame.
3. **Host-side credential binding.** A flag such as `--stream-credential us-central1-aiplatform.googleapis.com=gcp-adc` makes serve-api add the bearer at dial time, for that one `wss` host only. The token never becomes an AILANG value, so it cannot reach a client frame or a trace.

---

## The request (condensed from Daneel's message)

Daneel serves a private page from the Mac Studio over Tailscale. Mark talks to it from a laptop or an iPhone, and Daneel answers in voice, or optionally as a Gemini 3.8 Live avatar (fragmented MP4). The upstream leg, AILANG to Vertex Gemini Live over `wss`, **already works from `std/stream`**. Daneel's measurements, pinned at v0.40.1 and all in AILANG:

| path | measured by Daneel |
|---|---|
| text → speech | first audio 673 ms after send; 42 s of speech delivered in 12.6 s |
| voice in (24 kHz PCM, 100 ms chunks) | first audio 350–670 ms after end of speech |
| avatar (`responseModalities: VIDEO`) | fMP4 in 16 KB chunks, first frame 2.5 s, ~70 msgs/s peak |

The **browser leg** is missing. Today the only AILANG-only shape is push-to-talk: one POST per utterance and a fresh Live session per turn. That costs an extra 0.3–0.9 s per turn and allows no barge-in and no streaming playback.

**Done when** (Daneel's words): a module under serve-api relays a browser WebSocket to `wss://us-central1-aiplatform.googleapis.com/ws/...BidiGenerateContent`, an AILANG function decides each upstream→client frame, and the credential never appears in any client frame or trace.

These are the requester's measurements, not this author's. The in-repo claims this doc depends on are all re-verified in the [Verification Log](#verification-log).

---

## Problem Statement

### What is missing, measured against the code

| # | Gap | Evidence |
|---|---|---|
| G1 | serve-api has no WebSocket server path. `@route` accepts only the seven HTTP methods, and the parser rejects anything else | `internal/parser/parser_decl.go:175-183` (`PAR_ROUTE_INVALID_METHOD`). Reproduced in V1 |
| G2 | `std/stream` is client-only. "Server-side WebSocket" was an explicit non-goal of M-STREAM-BIDI | `m-stream-bidi-primitives.md:964` |
| G3 | Multiplexing two connections loses **which connection a frame came from**. `connSource` carries a name, but `eventToADT` builds `Message`/`Binary` without it | `internal/effects/stream_source.go:39-46`, `internal/effects/stream_mux.go:71,136`, `internal/effects/stream.go:487-499`. A two-connection relay type-checks today (V2), but its handler cannot tell a browser frame from an upstream frame |
| G4 | There is no way to carry **policy state** across frames. `selectEvents`/`onEvent` handlers are `StreamEvent -> bool`, and nothing threads a state value between calls | `std/stream.ail:99,145` |
| G5 | Binary frames are **received as `Binary(string)`** but **sent as `bytes`**. `StreamMessage.Bin(string)` is declared, yet the runtime rejects anything but `BytesValue` | `std/stream.ail:52,58`; `internal/effects/stream.go:494-498` (receive wraps a `StringValue`) versus `:255-261` (send requires `*eval.BytesValue`) |
| G6 | All requests share **one `StreamContext`**. `EffContext.Clone` is shallow, so `MaxConnections: 4` is process-wide, and `MaxDuration: 5min` kills any session that runs longer | `internal/effects/context.go:701-712`; `internal/effects/stream_context.go:64-78,101-113`; `cmd/ailang/serve_api.go:116-120` |
| G7 | A credential passed as a `connect` header is **rendered into the effect trace**. The sensitive-name patterns do not include `authorization` | `internal/effects/ops.go:112-130` renders every effect argument; `internal/effects/redact.go:13-19` (`key, secret, token, password, credential`); `std/secret.ail:12-27` documents the same hole for `<secret>` values |
| G8 | Auth and origin: serve-api auth reads **headers only**, which a browser WebSocket cannot set. serve-api has no Origin check, and the repo's one other WS server has `CheckOrigin: return true // TODO` | `internal/apiserver/auth.go:37-44`; `internal/websocket/server.go:37-40` |
| G9 | serve-api listens on `:port` (all interfaces) with CORS `*` on by default | `internal/apiserver/server.go:511,619-632`; `cmd/ailang/serve_api.go:20`. Owned by the companion doc |

### Impact

Daneel's live page would otherwise need a Go or Node sidecar. The sidecar would hold the Vertex credential, and the speech gate would sit outside AILANG. The program's routing lanes put that in the wrong place: the decision about what Daneel says is exactly what should be legible AILANG.

---

## Why the rejection no longer holds, and what we keep from it

The 2026-02-16 rejection was correct for what it reviewed: a **passthrough SSE proxy** built on a `yield` effect. Each of its five reasons is answered below.

| # | Rejection reason (verbatim gist) | Status in 2026-09 | What we keep |
|---|---|---|---|
| 1 | **Double-hop latency.** "A 30-line Go handler or nginx does passthrough with zero overhead" | **Answered by the requirement.** This is not passthrough. Every upstream→client frame needs a decision nginx cannot make: speak only when addressed, never let the client send `setup`, log each turn under a `call_id`. The relay is on a LAN or tailnet hop, and the costs that matter (Vertex ~350–670 ms to first audio) are upstream. The measured per-frame decision cost is ~33 µs (V9) | **Kept:** if a route would only `Forward` every frame, it should not exist. The acceptance tests require a non-trivial verdict (M2-A3), and the example is the speech gate, not an echo |
| 2 | **"AILANG's evaluator isn't built for throughput"**: each event goes through `CallValue` | **Measured, not assumed.** A string-scan gate over a 22 KB frame costs **~33 µs per call**, and a full `std/json.decode` of the same frame costs **~143 µs** (V9). At the 70 msgs/s avatar peak that is 2.3 ms/s and 10 ms/s of one core (0.2% and 1%). The evaluator also **never touches the bytes on `Forward`**: Go writes the frame it read | **Kept:** the evaluator must not sit on the byte path. `Forward`/`Drop` are Go operations on the original `[]byte`, and only `Replace*` materialises new bytes. Per-frame cost is a tracked metric (M5) |
| 3 | **~620 LOC of plumbing**: per-request EffContext cloning, `CallWithContext`, goroutine-per-request | **Mostly already shipped.** Every serve-api call now runs on a forked evaluator with a cloned `EffContext` (`internal/runtime/entrypoint.go:96-106`, `internal/eval/eval_evaluator.go:178-209`, `internal/effects/context.go:701-712`), and `net/http` already provides a goroutine per request. No `CallWithContext` and no `yield` effect are needed. New plumbing is the upgrade, the per-connection `StreamContext`, and the bridge loop, estimated at ~450 Go LOC plus tests | **Kept:** no new effect, and no `EffContext` field that is only a channel. The design reuses `Stream` and its `StreamTransport` seam |
| 4 | **Aggregate-then-return already works** | **Still true for request/response**, which remains the default. It does not work for a conversation: a Live session is a long-lived duplex, and aggregating it is push-to-talk (+0.3–0.9 s per turn, no barge-in) | **Kept:** HTTP routes are unchanged. `WS` is a separate, opt-in route kind |
| 5 | **"Transform-in-stream is the only compelling use case… if it emerges as a real need, revisit then"** | **It has emerged.** The consumer is named (Daneel, M-DANEEL-LIVE), prioritised by Mark on 2026-09-25, and measured upstream | **Kept:** the design is shaped around transform-in-stream (a verdict per frame), not around a general streaming response API |

The rejected doc also made points that were **right and are carried forward**:

- **Backpressure = block**, with a bounded buffer (its open question 2). It is kept, with explicit bounds per direction.
- **Do not forward the client's `Authorization` to the upstream** (its open question 4). It is kept and strengthened: the upstream credential is the **host's**, the client never has one, and the bridge refuses a client-supplied one.
- **Per-connection isolation of stream state** (its open question 1, "clone depth"). The answer is a **fresh child `StreamContext` per WS connection** (G6).
- **An SSE `yield` endpoint** remains unbuilt and out of scope ([Non-Goals](#non-goals)).

---

## Goals

**Primary goal:** relay a browser WebSocket to an upstream WebSocket through serve-api, with an AILANG function deciding every frame in both directions, and without the upstream credential ever becoming an AILANG value.

**Success metrics:**
- The Daneel shape (browser ⇄ serve-api ⇄ Vertex Live) runs with **zero Go or JS written by Daneel on the server side**. The browser page is Daneel's.
- Added latency per frame at p95, excluding upstream: **≤ 2 ms** for a pure string-scan step on a 22 KB frame, as measured in M5.
- A step function sustains **≥ 200 frames/s** per connection (about 3× the 70 msgs/s avatar peak) with no frame loss. Blocking backpressure stays bounded.
- A token planted by the credential binding is found in **0** client frames, **0** trace files (at `AILANG_TRACE=deep`) and **0** log lines in the M3 canary test.
- **0** behaviour change for existing HTTP `@route`s, OpenAPI, MCP and A2A. The existing apiserver test suite runs green without modification.

---

## Non-Goals

- **An SSE or streaming-HTTP response (`yield`).** The rejected shape stays rejected. Nothing here streams over HTTP responses.
- **A generic reverse proxy.** The upstream URL is chosen by AILANG code, never taken from client input (see [Risks](#risks--mitigations)).
- **N-way fan-out or rooms.** One client and one upstream per bridge. Broadcast is a different feature.
- **Changing `StreamEvent.Binary(string)` to `bytes`.** That is a breaking change to v0.8.1 API. The bridge uses its own frame type with `bytes` (G5), and fixing the old type is left to a separate doc.
- **Stream record/replay.** The M-STREAM-BIDI "Replay Contract" (`m-stream-bidi-primitives.md:137-153`) was never implemented (V13). This doc adds a per-frame **decision log** (M4) that makes *verdicts* replayable, not network I/O.
- **Automatic upstream reconnect** (A1). If the upstream closes, the bridge ends and the handler decides what to do.
- **WebRTC, TURN, or media transcoding.**

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D1. Mechanism: `WS` route plus a Go-driven verdict fold, rather than a WS route alone or an opaque bridge alone | Sets the shape of the public API and what the evaluator sees per frame | human (Mark) | design | high |
| D2. Route syntax: `@route("WS", path)` (extends the method whitelist), rather than a new `@websocket` annotation | Parser change. Every `@route` consumer (OpenAPI, MCP, A2A, catch-all) must learn to skip it | human | design | med |
| D3. The credential lives host-side (`--stream-credential host=gcp-adc`) and never becomes an AILANG value | This is the only design under which "never in any trace" holds without M-TRACE-LABEL-AWARE | human | design | high |
| D4. Browser auth: tailnet bind plus Origin allowlist, optionally with a subprotocol token | Browsers cannot set headers on WebSocket, so the existing `--api-key-header` cannot apply | human | design | med |
| D5. Backpressure: block, bounded per direction by frames and bytes. The step runs at **dequeue** | Decides barge-in semantics (stale audio can be dropped) and memory ceilings | agent (within the bounds given here) | compile | med |
| D6. Per-connection child `StreamContext` plus a server-wide session cap | Fixes G6 without changing CLI `run` semantics | agent | compile | med |

### Design Freeze

Mark must rule on these before sprint-executor starts:

- [x] **D1** mechanism: route + bridge fold (recommended below)
- [x] **D2** `@route("WS", …)` rather than `@websocket`
- [x] **D3** host-side credential binding. **Amended by Mark:** the source must be explicit, not `gcp-adc` (see [Decisions](#decisions-mark-2026-09-25))
- [x] **D4** browser authentication posture for the Daneel page (see [Open questions](#open-questions-for-mark))

### Decisions (Mark, 2026-09-25)

- **D1 mechanism:** route + bridge, as recommended.
- **D3 credential:** Daneel already has its own GCP credential in its own GCP project. The binding uses a credential **configured for the upstream host from an explicitly named source**, not implicitly the Studio's gcloud user ADC. It is still never an AILANG value and never in a frame or trace. Implemented as `--stream-credential HOST[:PORT]=SOURCE` with sources `gcp-key-file:PATH` (service_account, impersonated_service_account or external_account; a gcloud `authorized_user` file is refused), `gcp-metadata` (named explicitly) and `bearer-file:PATH`. No implicit ADC and no gcloud fallback.
- **D4 browser auth:** tailnet bind + Origin allowlist for v1. An optional key is supported (`Sec-WebSocket-Protocol: ailang.v1, ailang.key.<key>`) but not required.
- **Sequencing:** M0 is [#1313](https://github.com/sunholo-data/ailang/pull/1313) (`--bind` loopback default, `--cors-origin` allowlist), which this work reuses and does not reimplement.

### Implementation decisions (agent, 2026-09-25)

- **No `--ws-allow-any-interface`.** #1313 made loopback the default, so the escape hatch had nothing left to protect. Instead serve-api **refuses to start** when a `WS` route is registered, the bind host is not loopback, and there is no `--cors-origin` allowlist. `--cors` (every origin) does not count.
- **One allowlist.** `--cors-origin` is both the CORS and the WS Origin allowlist; `--ws-allowed-origins` was not added. Same-origin is always allowed; a missing or `null` Origin is refused.
- **Queue bound** is `--ws-queue-frames` (default 64 per direction). The byte bound is 64 × the 64 KiB frame limit = 4 MiB, so there is no second byte counter.
- **Bridge end semantics:** the upstream is always closed when `bridge` returns; the client too, except on `UpstreamClosed`, so the handler can dial a new upstream (the doc's A1 non-goal leaves reconnection to the handler).
- **Handler request record** is `{path, query, origin}`; headers are not passed (cookies would be one `show` away from a log).
- **Decision log sink:** the serve-api log (`[ws-bridge] {json}` lines), enabled by `--ws-decision-log`.
- **G7 fix location:** the one trace renderer (`eval.ShowTraceBounded`, reached from `EffContext.RenderTraceValue` and the typed evaluator's `boundedShow`), so effect, builtin and function-call trace sites are all covered by one change. Name rules are exact names, not substrings, so `{input_tokens: 50}` stays readable.

## Deferred Decisions

- The exact flag names (`--stream-credential`, `--ws-allowed-origins`, `--ws-max-sessions`, `--stream-max-duration`). The agent may choose, but any new env var must go through the `internal/config` Registry (forbidigo).
- Default queue bounds (proposed: 64 frames / 4 MiB per direction). The agent may tune them from the M5 measurement.
- Whether `BridgeFrame` also carries `Tick` (a timer frame for time-based policy). Human at review. It is not needed for Daneel v1.
- Whether a `Verdict` may also emit a frame to the **other** side (for example, answer a client ping locally). Human at review. v1 is one frame in, zero or one frame out, to the frame's natural destination.
- The decision-log sink format (runlog JSONL vs. the observatory span). The agent may choose, provided it reuses an existing sink.
- Whether `--watch` hot reload closes live WS sessions or lets them finish on the old code. The agent may choose. The recommendation is to let them finish, and to log it.

---

## Solution Design

### Overview: the mechanism, and why

Three candidate mechanisms were considered.

| Option | Shape | Why not alone |
|---|---|---|
| **A. General WS route only** | Handler gets `client: StreamConn` and writes its own loop with `connect` + `selectEvents` + `transmitBinary` | **Cannot express the speech gate.** Handlers are stateless `StreamEvent -> bool` (G4), and in a multiplexed loop the handler cannot tell which side a frame came from (G3). Every frame also crosses AILANG twice, as a `Binary(string)` in and a `bytes` out (G5) |
| **B. Opaque bridge only** | serve-api owns both legs. AILANG supplies only `(s, frame) -> (s, verdict)` | **Takes away the setup phase.** The handler must choose the brief, send the Live `setup` message, maybe await `setupComplete`, and log the `call_id` *before* frames flow. It also offers no primitive for the next non-bridge WS use |
| **C. A's route + B's fold (chosen)** | `@route("WS")` gives the handler the browser leg as a `StreamConn`. The handler opens the upstream with ordinary `connect` and then calls `bridge(client, up, init, step)` | Covers both. The route is general (A). The fold is the "opinionated bridge" the request invited (B). It is a library function, not a server mode, so it composes with everything else in `std/stream` |

Option C adds one route kind, one stdlib function with a Go builtin, and one host flag. It adds **no new effect**: everything is under `Stream`.

### Architecture

```
 browser (laptop / iPhone, on the tailnet)
    │  wss://studio.<tailnet>.ts.net:PORT/live    (Origin checked, D4)
    ▼
 serve-api  ── http.Handler for @route("WS","/live") ─────────────────────────────┐
    │ 1. Origin / session-cap / auth check  → 403/503 BEFORE upgrade              │
    │ 2. gorilla Upgrader.Upgrade → server-side StreamTransport (platform/streamws)│
    │ 3. engine.Call(module, "live", StreamConn(clientID), req)                    │
    │      └─ runtime.CallEntrypoint → Fork() → cloned EffContext                  │
    │          └─ child StreamContext (own conn map, own limits)  ◄── NEW (G6)     │
    ▼                                                                              │
 AILANG handler `live(client, req)`  (one call = one connection lifetime)          │
    ├─ choose brief, runlog call_id                     (IO / FS / Clock …)        │
    ├─ up = connect(VERTEX_WSS, {headers: []})           Stream.connect            │
    │      └─ credential binding adds Authorization at dial  ◄── NEW (D3), Go-only │
    ├─ transmit(up, setupJson)                                                     │
    └─ bridge(client, up, init, step)                   ◄── NEW builtin            │
          Go loop:  select { client frame | upstream frame | timers }              │
            frame ─► step(s, ClientBin(b) | UpText(t) | …)  (AILANG, per frame)    │
            verdict: Forward → write original []byte to the other side             │
                     Drop    → nothing                                             │
                     Replace*→ write the replacement                               │
                     CloseBridge(code, why) → close both, return                   │
          returns (finalState, BridgeEnd)                                          │
    ▼                                                                              │
 handler returns → serve-api closes the client socket, releases the child context ┘
```

### Components

**1. `@route("WS", path)`: accept and lifetime** (`internal/parser`, `internal/apiserver`)

- The parser adds `"WS"` to the `@route` method set (`parser_decl.go:175`).
- The handler signature is **checked at registration**, not at call time. The first parameter must be `StreamConn` (from `std/stream`), the second an optional request record `{path, query, headers}`, and the return must include `Stream` in its effect row. A mismatch fails serve-api startup with a message naming the function. This follows the precedent that ValidateRegistration fails fast (`server.go:268`).
- `registerCustomRoutes` (`routes.go:312-338`) registers `WS` routes with a new `wsHandler`. They do not go through `callFunction`. The existing method check (`routes.go:327`) would reject the upgrade `GET`, so `WS` routes accept `GET` with `Upgrade: websocket` and nothing else. A plain `GET` gets `426 Upgrade Required`.
- **Lifetime.** `wsHandler` upgrades, wraps the connection as a server-side `StreamTransport`, registers it in a **fresh child `StreamContext`**, and calls `engine.Call(...)` with `StreamConn(id)`. `engine.Call` already forks the evaluator and clones the `EffContext` per call (`entrypoint.go:96-106`, `eval_evaluator.go:199-209`). The handler call **is** the connection: when it returns, serve-api closes the client socket with 1000 (or 1011 if the call failed) and `CloseAll()`s the child context. When the client disconnects, the client `StreamConn` delivers `Closed`, and the bridge returns `ClientClosed`.
- **Surface exclusion.** `WS` exports are excluded from OpenAPI (`openapi.go:195-212`), MCP (`mcp.go:110-118`), A2A and the `/api/{module}/{func}` catch-all. The single change point is `loadedExportMember` / `isExposed` (`authorized_surface.go`) plus an `IsWS` flag on `ExportInfo`.
- **Graceful shutdown.** `http.Server.Shutdown` does not close hijacked connections, so `wsHandler` registers each session with `srv.RegisterOnShutdown`, which sends close 1001 (going away).

**2. The server-side transport** (`internal/platform/streamws`)

The accepted `*websocket.Conn` is adapted to the **existing** `effects.StreamTransport` interface: `Recv`, `Send`, `Close`, `Subprotocol` (`stream_transport.go`, `streamws.go:57-124`). The client leg then gets the same `readLoop`, the same bounded `eventBuffer`, and the same frame accounting as an outbound connection (`stream_ws.go:104-137`). The upgrader lives in `platform/streamws`, next to the dialer, so `internal/effects` keeps **no** gorilla import. The closure gate (`internal/diag/closure_expected_violations.txt`, "the list is EMPTY") stays empty. A new `effects.AdoptStreamConnection(sc *StreamContext, t StreamTransport) (int, error)` is the only core addition for this component.

**3. `std/stream/bridge`: the verdict fold** (`std/stream/bridge.ail` + `internal/effects/stream_bridge.go`)

```ailang
-- PROPOSED surface. Type-checked today against local stand-ins (V3, V4).
export type BridgeFrame = ClientText(string) | ClientBin(bytes) | UpText(string) | UpBin(bytes)
export type Verdict = Forward | Drop | ReplaceText(string) | ReplaceBin(bytes) | CloseBridge(int, string)
export type BridgeEnd = ClientClosed(int, string) | UpstreamClosed(int, string) | ClosedByVerdict(int, string) | TimedOut(string) | StepFailed(string)

export func bridge[s](client: StreamConn, up: StreamConn, init: s,
                      step: (s, BridgeFrame) -> (s, Verdict) ! {e}) -> (s, BridgeEnd) ! {Stream, e}
```

Semantics:

- **Serialised, one step per frame, and the state is a value.** The Go loop dequeues one frame from either side and calls `step(state, frame)` through `FnCallerN`, which the forked evaluator already wires (`eval_evaluator.go:206`). It applies the verdict and threads the returned state into the next call. There is no mutable state anywhere, so the policy is a Mealy machine: given the same frame sequence it produces the same verdicts (A1). It can also be tested offline with `foldl` and no network (V3).
- **Frames carry their direction** (`Client*` / `Up*`), which fixes G3 without changing `StreamEvent`. Binary frames arrive as `bytes`, which fixes G5 for the bridge. A `BytesValue` wraps the frame's `[]byte` without copying.
- **The byte path stays in Go.** `Forward` re-sends the **original** `[]byte` on the other side. `Drop` sends nothing. `ReplaceText`/`ReplaceBin` send the returned value, as text or binary regardless of the input kind. `CloseBridge(code, reason)` closes the **client** with that code and reason, closes the upstream with 1000, and returns `ClosedByVerdict`.
- **Merge order.** The loop reuses `selectEventsLoop`'s rules (`stream_mux.go:12-25`). Client and upstream share one priority band, so they are round-robined and neither starves. Only `Message`/`Binary` events reach `step`. `Opened`/`Ping` are consumed by the loop. `Closed`/`StreamError` end the bridge with the matching `BridgeEnd`, and `step` is not called on them.
- **Failure is typed.** If `step` raises or panics, the result is `StepFailed(msg)`, both legs are closed (client 1011), and the error is returned as a value, not a Go error. It reuses `callHandlerSafe`'s recover (`stream.go:571-595`).
- **Effect row.** `e` flows into the caller's row. V5 shows that `step ! {IO}` forces `IO` into the caller's signature, and that omitting it is a type error. A logging step is therefore visible in `live`'s signature (A3).
- **Budget.** Each forwarded or replaced frame charges `Stream.send`, reusing `RequireCapWithBudget` (`stream.go:215`). This doc also implements the **`Stream.recv`** charge (one per dequeued frame), which M-STREAM-BIDI specified (`m-stream-bidi-primitives.md:159-165`) but never wired (V12). A budget-exhausted bridge ends with `CloseBridge`-like semantics and `StepFailed("BudgetExhausted: …")`.

**4. Per-connection `StreamContext` and limits** (`internal/effects/stream_context.go`, `cmd/ailang/serve_api.go`)

- `StreamContext.Child()` copies the **policy** fields (allowlist, AllowHTTP/Localhost, private-IP blocking, sizes, timeouts) and gets a **fresh** `connections` map, source map and IDs. `wsHandler` sets it on the call's cloned `EffContext`. It does not modify `Clone()`, which also serves non-WS calls and the CLI.
- Two connection IDs from different sessions can no longer be confused. Today a handler in one request can `GetConnection(id)` on another request's connection, because the map is shared (`stream_context.go:123-128`).
- serve-api defaults are the **existing** values, with overrides from flags: `--stream-max-duration` (Daneel needs over 5 min, G6), `--stream-idle-timeout`, and `--ws-max-sessions` (server-wide semaphore, default 4, checked **before** upgrade with 503). `MaxConnections` becomes per child (default 4, so one session may hold 2–4 legs).

**5. Backpressure and bounded queues (D5)**

- Each direction is bounded by the source connection's `eventBuffer`. The bridge sizes it from `--ws-queue-frames` (default 64, instead of the global 1000) **and** caps queued bytes (default 4 MiB). The worst case per session is therefore about 2 × 4 MiB, not 1000 × 64 KiB × 2 ≈ 128 MiB.
- **Policy = block** (the M-STREAM-BIDI invariant, `m-stream-bidi-primitives.md:105-116`). When the browser reads slowly, the client write blocks, the bridge stops dequeuing upstream frames, the upstream `readLoop` blocks on a full buffer, and TCP backpressure reaches Vertex. Nothing is silently dropped by the runtime. **Every** drop is a `Drop` verdict and is logged.
- Writes are serialised per connection under `conn.mu`, as `StreamSend` already does (`stream.go:232-233`). gorilla allows one concurrent writer per connection.

**6. Cancellation and barge-in**

- Barge-in in Gemini Live is **server-detected**: the upstream signals an interruption, and the client must stop playing. Because `step` runs at **dequeue**, the policy can mark `interrupted` in its state and `Drop` every model-audio frame **already queued** behind the interruption until the next turn starts. A post-hoc proxy cannot do that. The client is told with a small `ReplaceText` control frame, or with the forwarded interruption message.
- **Client cancel:** the browser closes its socket. The client leg delivers `Closed`, the bridge returns `ClientClosed`, and the handler `disconnect`s the upstream. serve-api's deferred `CloseAll()` on the child context is the backstop.
- **Server cancel:** `CloseBridge(code, reason)` from `step`, `--stream-max-duration`, idle timeout (`TimedOut`), or serve-api shutdown (1001).
- The exact Vertex message fields (for example `serverContent.interrupted`, `turnComplete`, `inputTranscription`) are **external and unverified here** (P1 in the Verification Log). The design does not depend on their names. Only the example does.

**7. The credential never reaches AILANG (D3)**

Today the only way to authenticate the upstream is to put `{name: "Authorization", value: "Bearer …"}` into `connect`'s config (`stream_ws.go:47-60`). That makes the token an AILANG string, and it gets rendered into the effect trace (`ops.go:112-130`). The sensitive-name patterns do not match "authorization" (`redact.go:13-19`), and `std/secret.ail:12-27` records that labels do not reach traces either (G7).

The design:

- serve-api accepts `--stream-credential <host>=<provider>`. The only v1 provider is `gcp-adc`, backed by the existing `internal/auth/gcp.AccessToken` (`adc.go:40-62`). The binding is a `func(*url.URL) (http.Header, error)` hook on the `StreamContext`, set by `cmd/ailang`. `internal/effects` therefore imports nothing from `internal/auth`, which keeps the core closure.
- **Match rule:** only an exact host, only `wss`, and only after the destination authorizer has passed (`stream_ws.go:75-84`). A binding never applies to `ws://`, to another host, or to a redirect.
- **Token cache:** `AccessToken` has no cache and tries the metadata server (2 s timeout) before `gcloud` (`adc.go:40-62`). The binding caches the token until 5 minutes before expiry and prefetches it at startup, so a dial does not pay 1–2 s. A token-fetch failure is a `ConnectionFailed` result with the provider named and **no silent fallback** (CLAUDE.md §2).
- **Fail loud on double auth:** a `connect` to a bound host whose config **also** carries an `Authorization` header is refused (`ConnectionFailed("credential binding: Authorization header supplied by program for bound host …")`). The token then has exactly one source.
- **Defence in depth for unbound hosts:** `Stream.connect`'s trace rendering redacts header values whose names match `authorization`, `proxy-authorization`, `cookie` or `x-api-key`. The rule reuses `IsSensitiveVarName`'s mechanism with a header-specific pattern list. This closes G7 for programs that still pass headers themselves.
- **Canary:** M3's test plants a unique token through a fake provider, runs a full bridge session at `AILANG_TRACE=deep`, and greps the trace file, stderr, the decision log and every frame the client received. All four must contain zero hits (Success metric 4).

**8. Capability and effect declaration**

- The route handler's type **is** the declaration: `live(client: StreamConn, req: {...}) -> unit ! {Stream, IO, …}`. serve-api refuses to register a `WS` route whose effect row names a capability not granted by `--caps` (the same check `ailang run` makes). Without `--caps Stream` no `WS` route registers, and startup fails naming the route.
- No new effect is introduced. `Stream` covers accept, connect and bridge, and the per-op budget keys (`Stream.connect`, `Stream.send`, `Stream.recv`) remain the cost surface.

**9. Origin, bind and auth (D4, G8, G9)**

- **Prerequisite:** [m-serveapi-bind-host-cors](m-serveapi-bind-host-cors.md) supplies `--host` (so Daneel binds the Studio's tailnet address, not `*`) and turns CORS `*` from default-on to opt-in. This doc does not duplicate that work. Until that doc lands, **M1 refuses to register any `WS` route when the listen address is unspecified (`*`)** unless `--ws-allow-any-interface` is passed. The check fails loud and names the companion doc.
- **CORS does not apply to WebSockets**, so a check on the `Origin` header is mandatory. The default is same-origin: `Origin`'s host must equal the request `Host`, which is gorilla's default `CheckOrigin`. `--ws-allowed-origins` adds exact origins. `Origin: null` and a missing `Origin` are refused. The check runs **before** the upgrade (403). This is the step the repo's other WS server skipped (`internal/websocket/server.go:37-40`).
- **Browser auth:** `--api-key-header` cannot work, because browsers cannot set headers on `new WebSocket()`. If an API key is configured, `WS` routes accept it **only** as a `Sec-WebSocket-Protocol` entry `ailang.key.<key>`. That value is echoed per the WS spec, so the server selects `ailang.v1` and never echoes the key. Query-string tokens are refused, because they leak into logs and history. The recommended Daneel posture is in [Open questions](#open-questions-for-mark).

### Implementation Plan (milestones)

**M0: Prerequisite gate (≈0.5 d, mostly in the companion doc)**
- [ ] m-serveapi-bind-host-cors lands `--host` and opt-in CORS, or this doc's M1 ships the fail-loud `*` refusal described above.
- **Accept:** `ailang serve-api --host 100.x.y.z` listens only on that address (`lsof -i`), and a `WS` route under `*` without the override fails startup with a message naming the companion doc.

**M1: `@route("WS")` accept path (≈1.5 d)**
- [ ] Parser: add `"WS"` to the method set and a `route_attr_test.go` case. Error messages list `WS`.
- [ ] `ExportInfo.IsWS`. Exclude `WS` exports from OpenAPI, MCP, A2A and the catch-all.
- [ ] Registration-time signature check (`StreamConn` first, `Stream` in row, caps granted).
- [ ] `platform/streamws` server transport + `effects.AdoptStreamConnection`.
- [ ] `StreamContext.Child()`, `--ws-max-sessions`, `--stream-max-duration`, `--stream-idle-timeout`, and the Origin check.
- [ ] Shutdown closes live sessions with 1001.
- **Accept:** (A1) An echo handler using only existing `onEvent`/`transmit`/`transmitBinary` round-trips text **and** binary frames with a Go `gorilla` test client. (A2) A plain `GET` returns 426, a bad `Origin` returns 403, and the fifth concurrent session returns 503, all before upgrade. (A3) A `WS` export is absent from `/api/_meta/openapi.json` and from MCP `tools/list`. (A4) Two concurrent sessions cannot see each other's connection IDs. (A5) The existing apiserver tests pass unmodified.

**M2: `bridge` fold (≈1.5 d)**
- [ ] `std/stream/bridge.ail` (types + wrapper) and `_stream_bridge` builtin → `effects.StreamBridge`.
- [ ] Go loop: two-source merge, step via `FnCallerN`, verdict application on the original `[]byte`, `BridgeEnd`, bounded queues, `Stream.send`/`Stream.recv` budgets.
- **Accept:** (A1) Against the **fake upstream** (below), a `Forward`-all step relays 1,000 mixed text and binary frames each way, byte-identical and in order. (A2) A step that `Drop`s every `UpBin` while not addressed, run on a scripted session, delivers exactly the expected frame subset (the speech-gate fixture). (A3) The non-trivial verdict requirement from rejection #1: that fixture's expected subset is a strict subset of the frames sent. (A4) Barge-in: with 50 queued model frames and an interruption frame, 0 stale frames reach the client after the interruption is dequeued. (A5) A step that raises returns `StepFailed` and the client sees close 1011. (A6) A client close mid-stream returns `ClientClosed` within 100 ms, and the upstream is closed.

**M3: Credential binding (≈1 d)**
- [ ] `--stream-credential host=gcp-adc`, the header hook on `StreamContext`, the cache and prefetch, and the double-auth refusal.
- [ ] Header redaction in `Stream.connect` trace rendering.
- **Accept:** (A1) The **canary**: a fake provider token appears 0 times in client frames, the deep trace, stderr and the decision log across a full session. (A2) The fake upstream asserts it **received** the `Authorization` header, which proves that the binding fires. (A3) A binding for host X is not applied to host Y, to `ws://X`, or to a redirect. (A4) A program-supplied `Authorization` to a bound host fails with the named error. (A5) Without the binding, a program-supplied `Authorization` header is `[REDACTED]` in the trace.

**M4: Decision log + example (≈0.5 d)**
- [ ] Per frame, one line: `{call_id, seq, dir, kind, bytes, verdict, step_us}`. It never contains a payload. The log goes to the existing Debug/runlog sink (deferred decision).
- [ ] `examples/serveapi_ws_bridge.ail` (the speech gate against an `ws://` echo) and a docs page under `docs/docs/guides/`.
- **Accept:** `make verify-examples` green. Replaying the logged `(dir, kind)` sequence through `foldl(step)` offline reproduces the logged verdicts.

**M5: Measurement (≈0.5 d)**
- [ ] A Go benchmark with the fake upstream: 22 KB binary frames at 200/s for 30 s, recording p50/p95 relay latency, `step_us`, RSS and frame loss.
- **Accept:** meets the Success-metrics thresholds. The numbers are recorded in this doc's implementation report.

**Daneel acceptance (outside CI):** a `WS` route relays the Daneel page to live `BidiGenerateContent` with the speech gate deciding each upstream→client frame, and the M3 canary is rerun against the real ADC provider. Daneel's evaluator runs this. It is not a CI job, because CI has no Vertex.

### Files to Modify/Create

- `internal/parser/parser_decl.go`: `"WS"` in the `@route` method set and in the messages (~5 LOC)
- `internal/parser/route_attr_test.go`: WS accept and reject cases (~40 LOC)
- `internal/apiserver/server.go`: `IsWS` on `ExportInfo`, WS config fields, shutdown hook (~40 LOC)
- `internal/apiserver/routes.go`: register `WS` routes with `wsHandler`, excluded from the method check (~25 LOC)
- `internal/apiserver/routes_ws.go`: NEW. Origin/session/auth checks, upgrade, child context, call, close (~180 LOC)
- `internal/apiserver/authorized_surface.go`: exclude `IsWS` from HTTP, MCP and A2A surfaces (~5 LOC)
- `internal/apiserver/openapi.go`: skip `IsWS` (~3 LOC)
- `internal/apiserver/routes_ws_test.go`: NEW. M1 acceptance (~250 LOC)
- `internal/platform/streamws/streamws.go`: server-side `Accept` → `StreamTransport` (~50 LOC)
- `internal/effects/stream_context.go`: `Child()`, credential hook field (~40 LOC)
- `internal/effects/stream_ws.go`: `AdoptStreamConnection`, apply the credential hook at dial, double-auth refusal (~50 LOC)
- `internal/effects/stream_bridge.go`: NEW. The bridge loop (~220 LOC)
- `internal/effects/stream_bridge_test.go`: NEW. M2 acceptance with fake upstream (~350 LOC)
- `internal/effects/ops.go`: header-aware redaction for `Stream.connect` args (~20 LOC)
- `internal/builtins/stream.go`: `_stream_bridge` registration (~40 LOC)
- `std/stream/bridge.ail`: NEW. Types + `bridge` wrapper (~40 LOC)
- `cmd/ailang/serve_api.go`: flags, binding wiring via `internal/auth/gcp`, per-session limits (~80 LOC)
- `examples/serveapi_ws_bridge.ail`: NEW (~60 LOC)
- `CHANGELOG.md`, `docs/docs/guides/`: serve-api WS page (~120 LOC docs)

---

## Conflict Surface

This change touches `internal/parser/` and `internal/effects/`, so the section is required.

### Syntactic and semantic positions touched

1. The first argument of `@route(...)`, a string literal in the closed method set.
2. The `std/stream` import namespace: a new submodule `std/stream/bridge` with constructors `ClientText, ClientBin, UpText, UpBin, Forward, Drop, ReplaceText, ReplaceBin, CloseBridge, ClientClosed, UpstreamClosed, ClosedByVerdict, TimedOut, StepFailed`.
3. `Stream` effect operations: new op `bridge` plus a changed `connect` (credential hook and double-auth refusal).
4. serve-api route dispatch: a fourth route kind besides plain, `@raw` and `@nowrap`.

### What else lives here

| Position | Existing occupants | Interaction |
|---|---|---|
| `@route` method | `GET POST PUT DELETE PATCH HEAD OPTIONS` (`parser_decl.go:175-178`) | `WS` is a new literal, and no existing method is named `WS`. The match is exact and case-sensitive, as today. `@route("ws", …)` stays an error |
| `@raw` / `@nowrap` / `@noexpose` / `@nomcp` on the same function | Modify HTTP routes | On a `WS` route they are **meaningless**. Registration rejects `@raw`/`@nowrap` with `WS` (fail loud, not ignored). `@nomcp`/`@noexpose` are redundant and allowed |
| Constructor names | `std/stream` already exports `Text`, `Bin`, `Message`, `Binary`, `Closed`, `Timeout`… (`std/stream.ail:52-61`) | New names were chosen **not** to collide: `ClientBin` rather than `Bin`, `TimedOut` rather than `Timeout`, `CloseBridge` rather than `Close`. They live in a separate module, imported explicitly |
| Paths | Built-in paths (`server.go:582-597`), other `@route`s | The existing collision and duplicate checks (`routes.go:314-322`) apply unchanged to `WS` routes |
| `Stream.connect` headers | Programs that pass `Authorization` themselves (the only way today) | **Unchanged for unbound hosts** (only the trace rendering redacts). Refused for bound hosts, which is intentional and only happens when the operator opts in with `--stream-credential` |
| `EffContext.Clone` | Every serve-api call; CLI via `Fork` | Not modified. `Child()` is applied only by `wsHandler` |

### Disambiguation strategy

Only the literal `"WS"` changes the route kind, so no lookahead is involved. The type checker is not changed: the bridge is an ordinary polymorphic stdlib function, and its signature type-checks today (V4).

### Programs that MUST still work

- `internal/apiserver/routes_test.go`, `routes_annotations_test.go`, `handler_test.go`, `mcp_schema_test.go`: unchanged HTTP and MCP surfaces.
- `internal/parser/route_attr_test.go`: every existing method is still accepted, and invalid methods other than `WS` are still rejected.
- `internal/effects/stream_ws_test.go`, `stream_mux_test.go`: outbound WS and `selectEvents` behaviour is unchanged.
- `/private/tmp/claude-501/wsbridge/scratch/twoconn.ail` (V2), the existing-API relay pattern: it must still type-check. It is re-created in `examples/` as the "why the bridge exists" counter-example.

### What deliberately changes

- `Stream.connect` to a **bound** host with a program-supplied `Authorization` header now fails. This only happens with the new opt-in flag.
- `Stream.connect` trace records render sensitive **header values** as `[REDACTED]`. That changes trace bytes for any program that passes auth headers, which is the point.

---

## Examples

### Example 1: the speech gate (pure policy, tested offline)

Verified with `ailang check` and `ailang run` (V3). The proposed types are declared locally in the scratch module, because `std/stream/bridge` does not exist yet.

```ailang
module scratch/gate

import std/string (contains, toLower)
import std/list (foldl)

export type BridgeFrame = ClientText(string) | ClientBin(bytes) | UpText(string) | UpBin(bytes)
export type Verdict = Forward | Drop | ReplaceText(string) | ReplaceBin(bytes) | CloseBridge(int, string)
export type GateState = { addressed: bool, turns: int, dropped: int }

-- The speech gate: pure, state threaded explicitly (a Mealy step).
export pure func gate(s: GateState, f: BridgeFrame) -> (GateState, Verdict) =
  match f {
    ClientText(_) => (s, Forward),
    ClientBin(_) => (s, Forward),
    UpBin(_) => (s, Drop),
    UpText(msg) =>
      if contains(msg, "\"inputTranscription\"") && contains(toLower(msg), "daneel")
      then ({s | addressed: true}, Forward)
      else if contains(msg, "\"turnComplete\"")
      then ({s | addressed: false, turns: s.turns + 1}, Forward)
      else if contains(msg, "\"modelTurn\"") && !s.addressed
      then ({s | dropped: s.dropped + 1}, Drop)
      else (s, Forward)
  }

-- Offline replay: the same step the bridge drives, over a recorded frame list.
export pure func replay(frames: [BridgeFrame], s0: GateState) -> (GateState, [Verdict]) =
  foldl(\acc f. match acc {
    (s, vs) => match gate(s, f) { (s2, v) => (s2, v :: vs) }
  }, (s0, []), frames)
```

The Vertex field names in this example are illustrative (P1). A production gate would also refuse a client `setup` frame (`ClientText` containing `"setup"` → `CloseBridge(1008, "setup is server-owned")`), so that the **brief and context stay AILANG-chosen**, as the request requires.

### Example 2: the route (proposed syntax)

The body type-checks today when it is annotated `@route("GET", …)` against a stub `bridge` (V4). `@route("WS", …)` itself is rejected by the current parser (V1). M1 changes that.

```ailang
module scratch/bridge_sig

import std/stream (connect, transmit, disconnect, StreamConn)
import std/result (Result, Ok, Err)
import scratch/gate (BridgeFrame, Verdict, GateState, gate)

-- @route("WS", "/live")   ← proposed; today: PAR_ROUTE_INVALID_METHOD
export func live(client: StreamConn, req: { path: string, query: string }) -> unit ! {Stream} {
  match connect("wss://us-central1-aiplatform.googleapis.com/ws/google.cloud.aiplatform.v1.LlmBidiService/BidiGenerateContent", { headers: [] }) {
    Ok(up) => {
      let _ = transmit(up, "{\"setup\":{}}");
      match bridge(client, up, { addressed: false, turns: 0, dropped: 0 }, gate) {
        (s, ClientClosed(_, _)) => disconnect(up),
        (s, _) => { disconnect(up); disconnect(client) }
      }
    },
    Err(_) => disconnect(client)
  }
}
```

Note the empty `headers: []`. The bearer is added by `--stream-credential us-central1-aiplatform.googleapis.com=gcp-adc`, outside the program.

```bash
ailang serve-api --host 100.101.102.103 --port 8443 --caps Stream,IO,Clock \
  --stream-credential us-central1-aiplatform.googleapis.com=gcp-adc \
  --ws-allowed-origins https://studio.tailnet-name.ts.net:8443 \
  --stream-max-duration 30m ./daneel/live/
```

### Example 3: an effectful step is visible in the signature

V5: a `loggedGate ! {IO}` passed to `bridge` makes the caller need `! {Stream, IO}`. Declaring only `! {Stream}` fails `ailang check` with *"Suggested fix: func run(...) -> T ! {IO, Stream}"*.

---

## Testing Strategy

**The fake upstream (CI needs no Vertex).** An `httptest.Server` with a gorilla upgrader, the same pattern as `internal/platform/streamws/streamws_test.go:20-45`. serve-api already sets `AllowHTTP`/`AllowLocalhost` for Stream (`serve_api.go:118-119`), so `ws://127.0.0.1` is reachable. For the credential tests the fake upstream runs over TLS (`httptest.NewTLSServer`) with its certificate injected through the existing `tlsClientConfig` test seam (`stream_context.go:57-60`), because bindings apply only to `wss`. The fake speaks a **scripted** session: a JSON file of `{dir, kind, payload|size, delay_ms}` steps. It asserts on what it received (including the `Authorization` header), which lets one fixture drive M2 and M3.

**Real frames as fixtures.** P1: the shape of Vertex Live frames (text vs. binary, JSON field names) is asked of Daneel as a captured session with **payloads truncated** and **no headers**, committed as `internal/effects/testdata/bridge/vertex_live_*.json`. Until that arrives, the fixtures are synthetic and labelled as such.

| Layer | Tests |
|---|---|
| Parser | `WS` accepted, `ws` and `WEBSOCKET` rejected, `@raw`+`WS` rejected at registration |
| apiserver | 426 / 403 / 503 before upgrade; surface exclusion (OpenAPI, MCP, A2A, catch-all); per-session ID isolation; shutdown 1001; registration signature check |
| effects | bridge ordering, verdict application, `Forward` byte-identity, bounded queue under slow client (no loss, bounded RSS), barge-in drop, `StepFailed`, budgets (`Stream.send`/`Stream.recv`) |
| credential | canary (4 sinks × 0 hits), binding fires, host/scheme/redirect exact-match, double-auth refusal, trace header redaction |
| AILANG | `examples/serveapi_ws_bridge.ail` in `make verify-examples`; offline `foldl` replay equals the logged verdicts |
| perf | M5 benchmark, recorded rather than gated in CI (flaky on shared runners); `step_us` is also a decision-log field |

The mutation check (memory: *mutation-test your own tests*): revert the Origin check and the canary exclusion, and each test must fail.

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| **The relay becomes an open, credentialed proxy** (G9: `*:port`, CORS `*`) | M0 prerequisite plus the fail-loud refusal of `WS` on `*`, Origin allowlist, session cap. The upstream URL is chosen by the program: the docs forbid building it from `req`, and M4's example lint flags `connect(` whose URL argument flows from `req` *(deferred: agent may implement as a doc rule if the lint is costly)* |
| serve-api's Stream policy is permissive (`AllowHTTP`/`AllowLocalhost` = true, `serve_api.go:118-119`) | Unchanged here. The credential binding is `wss`-and-exact-host only, so a permissive policy cannot leak the token to a `ws://` or local target |
| An evaluator stall (GC or a slow step) causes audio stutter | Per-frame `step_us` is in the decision log; M5 measures p95. The step is serialised by design, so a slow policy slows its own session and no other (each session has its own fork) |
| ADC token expiry during a long session | The token is used only at dial time, so an open socket is unaffected. A re-dial gets the cached or refreshed token |
| `--watch` reload mid-session | Sessions finish on the loaded code, and the reload is logged (deferred decision) |
| iOS Safari backgrounding kills the socket | Client concern. The bridge ends `ClientClosed` cleanly, and the page reconnects with a fresh session |
| Vertex frame format differs from the example | The design is agnostic, and only the example and fixtures depend on it (P1) |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | The policy is a pure Mealy step over an explicit state value. The same frame sequence gives the same verdicts. Arrival order across the two sockets is inherently nondeterministic and is **recorded** (`seq`, `dir`), not hidden |
| A2: Replayability | +1 | The decision log (no payloads) plus a `foldl` replay reproduces every verdict offline (M4). Network replay stays out of scope, and the doc says so |
| A3: Effect Legibility | +1 | No new effect. Accept, dial and bridge are all `Stream`. The step's row surfaces in the handler signature (V5). The credential injection is a named operator flag, not ambient |
| A4: Explicit Authority | +1 | The credential is scoped to exact host + `wss` + operator flag and is never handed to program code. Origin allowlist, session cap and `--caps` gating are checked before upgrade |
| A5: Bounded Verification | +1 | The policy is a pure function that can be unit-tested without sockets |
| A6: Safe Concurrency | +1 | Each connection has one fork and a child `StreamContext` (fixes the cross-session ID aliasing in G6). The step is serialised, and writes are serialised per connection |
| A7: Machines First | 0 | Neutral. The API is small and typed, with direction in the constructor names |
| A8: Minimal Syntax | 0 | One new literal in an existing annotation. No new keywords or grammar |
| A9: Cost Visibility | +1 | `Stream.recv` is finally charged, alongside `Stream.send` per forwarded frame, and `step_us` is logged per frame |
| A10: Composability | +1 | `bridge` is a library function over existing `StreamConn`s. The route kind is general, not bridge-specific |
| A11: Structured Failure | +1 | `BridgeEnd` is a typed ADT (client/upstream/verdict/timeout/step failure), and budget exhaustion is typed too |
| A12: System Boundary | +1 | The two boundary crossings (browser↔Studio, Studio↔Vertex) are explicit, and the credential boundary stays on the host side |

**Net: +10.** Hard-violation check: A1 ✓ (arrival nondeterminism is recorded, not implicit), A3 ✓, A4 ✓ (no ambient credential reaches code), A7 ✓.

---

## Quorum (candidate, not run)

This doc is a **candidate for `ailang design-quorum`**. Two triggers fire: **#1**, because there are design-freeze items (D1–D4), and **#4**, because load-bearing premises concern external systems (the Vertex Live frame format and the browser WebSocket auth constraints). It was **not run in this session**, per the task instructions. When it is run: `ailang design-quorum design_docs/implemented/v0_44_0/m-serveapi-websocket-bridge.md --author claude:claude-opus-5-5 --max-cost-usd 0.30`. The doc is long, and the $0.10 cap drops gpt6-astra (memory: design-quorum budget cap).

---

## Open questions for Mark

1. **D4 browser auth for the Daneel page.** Is **tailnet bind + Origin allowlist** enough, given that the tailnet is the perimeter and only Mark's devices are on it? Or should the page also present a key through `Sec-WebSocket-Protocol`? The recommendation is tailnet + Origin for v1, with the key as an option. The difference: anyone who reaches the tailnet address from a page on an allowed origin can spend Vertex quota as Daneel.
2. **D3 credential scope.** Is `gcp-adc` bound to the Studio's own ADC (Mark's user credentials via `gcloud`) acceptable, or should Daneel have a service account? The design supports either, because the provider is whatever ADC resolves to. It is a trust decision.
3. **D1 confirmation.** Should this be route + fold (recommended), or would you rather ship only the general `WS` route (M1) first and let Daneel write the loop? The latter cannot express the stateful gate today (G3/G4).
4. **Sequencing.** Is the companion bind/CORS doc on the same sprint? If not, M1 ships with the fail-loud `*` refusal, and Daneel must pass `--ws-allow-any-interface` behind the Studio's firewall until the companion doc lands.

---

## Verification Log

All in-repo claims were checked at `dev` = `62608138d`, 2026-09-25, `ailang v0.43.1-4-gab08ce876-dirty`. The scratch AILANG files are under `/private/tmp/claude-501/wsbridge/scratch/`.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `@route("WS", …)` is rejected today | `ailang check scratch/live.ail` | `PAR_ROUTE_INVALID_METHOD … expected GET, POST, PUT, DELETE, PATCH, HEAD, or OPTIONS` (source: `parser_decl.go:175-183`) |
| V2 | A two-connection relay over existing `selectEvents` type-checks, but the handler cannot tell sides apart | `ailang check scratch/twoconn.ail` → ✓; read `stream_mux.go:71,136` (`eventToADT(evt)`) + `stream.go:487-499` (no source field on `Message`/`Binary`) | Confirmed (G3) |
| V3 | The speech-gate step and offline `foldl` replay work | `ailang check` + `ailang run --entry main scratch/gate.ail` → `true`, `true` | Confirmed |
| V4 | The proposed `bridge[s]` signature, `BridgeEnd` ADT and `live` handler type-check (with `@route("GET")` and a stub body) | `ailang check scratch/bridge_sig.ail`, `scratch/live_get.ail` | ✓ both |
| V5 | An effectful step's row propagates to the caller | `scratch/logged.ail` ✓; `scratch/logged_bad.ail` fails with "Suggested fix: … ! {IO, Stream}" | Confirmed |
| V6 | serve-api forks the evaluator and clones `EffContext` per call | read `internal/runtime/entrypoint.go:90-106`, `internal/eval/eval_evaluator.go:174-212`, `internal/effects/context.go:692-712` | Confirmed. The clone is **shallow**: `Stream` is shared (G6) |
| V7 | `StreamContext` defaults are MaxConnections 4, MaxDuration 5 min, Idle 60 s, EventBuffer 1000, and the connection map is process-wide | read `internal/effects/stream_context.go:64-78,101-128`; `cmd/ailang/serve_api.go:116-120` creates one per server | Confirmed |
| V8 | Binary is received as `string` and sent as `bytes`; `Bin(string)` is declared but the runtime requires bytes | `std/stream.ail:52,58`; `internal/effects/stream.go:494-498`, `:255-261` | Confirmed (G5) |
| V9 | Evaluator per-call cost | `scratch/bench.ail`: `runGate` 1→0.06 s, 4200→0.20 s (≈33 µs/call on a 22 KB frame); `runJson` 1→0.07 s, 4200→0.67 s (≈143 µs/decode). Measured via `ailang run`. Neither `cmd/ailang/main_run*.go` nor `internal/embed` imports `internal/vm` (grep empty), so this is the same evaluator family serve-api uses. It is single-machine (M-series Studio) and indicative only. M5 re-measures in-process | Recorded |
| V10 | Effect args are rendered into traces, and "authorization" matches no sensitive pattern | read `internal/effects/ops.go:112-130`, `internal/effects/redact.go:13-19` | Confirmed (G7) |
| V11 | serve-api auth is header-only; the other in-repo WS server skips the Origin check | read `internal/apiserver/auth.go:14-57`, `internal/websocket/server.go:34-40` | Confirmed (G8) |
| V12 | `Stream.recv` budget is **not** implemented (negative claim) | `grep -rn "stream.recv" internal/effects/*.go` → no hits. Only `stream.connect/send/sse_*/ndjson_post` are charged | Confirmed absent |
| V13 | Stream record/replay is **not** implemented (negative claim) | `grep -rli replay internal/effects/stream*.go` → empty; `grep RecordStream internal/trace` → empty | Confirmed absent |
| V14 | No existing `_stream_bridge`, `_stream_accept`, `std/stream/bridge`, or the proposed flag names (negative claim) | `grep -rn "_stream_bridge\|_stream_accept\|std/stream/bridge\|stream-max-duration\|ws-allowed-origins\|stream-credential" internal cmd std` → empty | Names free |
| V15 | The core must not import gorilla or `internal/auth` | `internal/diag/closure_expected_violations.txt:12-18` ("The list is EMPTY") | The design places the upgrader in `platform/streamws` and the binding in `cmd/ailang` |
| V16 | ADC helper exists, uncached, metadata-first | read `internal/auth/gcp/adc.go:1-62` | Confirmed; the binding adds caching |
| V17 | Annotations are a closed parser set (so `@websocket` would equally be a parser change) | read `internal/parser/parser_decl.go:29-55` | Confirmed. D2 is a like-for-like parser change either way |
| V18 | Related-doc search found no duplicate | `create_planned_doc.sh` neural matches ≤ 0.43 (`m-dx-serve-api-error-status`, `serve-api-port-collision`) | Below the 0.45 warn threshold |
| **P1** | **PENDING (external):** Vertex Live frame format: text vs. binary WS frames, and the names `inputTranscription` / `turnComplete` / `interrupted` / `setup` | Not verifiable in-repo | Only the example and fixtures depend on it. A captured, payload-truncated session is requested from Daneel |
| **P2** | **PENDING (external):** browser `WebSocket` cannot set arbitrary headers, and `Sec-WebSocket-Protocol` is the only header-like channel | WHATWG WebSocket API (not re-checked in this session) | D4 depends on it |
| **P3** | **PENDING (requester):** Daneel's latency and throughput numbers | Reported, not reproduced | Used only for sizing |

---

## Related Documents

- [M-STREAM-SERVE-API (rejected)](../../rejected/m-stream-serve-api-proxy.md): answered point by point above
- [M-STREAM-BIDI](../v0_8_1/m-stream-bidi-primitives.md): dispatch model, block backpressure, budget keys (reused); server-side WS was its explicit non-goal (`:964`)
- [M-WASM-STREAM-BRIDGE](../v0_8_1/m-wasm-stream-bridge.md): the browser-side AILANG stream bridge. Orthogonal: it runs AILANG **in** the browser, whereas here the browser is a thin client and policy runs on the Studio
- [M-ASYNC-IO](../v0_9_0/m-async-io-stream.md): `selectEvents` merge rules reused by the bridge loop
- [m-serveapi-protocol-only-module](../../planned/m-serveapi-protocol-only-module.md): Go-packaging split of the `serveapi` facade. **Overlap:** the new WS code must stay in `internal/apiserver` and `internal/platform/streamws` and add nothing to the protocol-only package's closure. The facade may later expose a `WS` option, which is out of scope here
- [m-serveapi-bind-host-cors](m-serveapi-bind-host-cors.md): **prerequisite** (written in parallel)
- [m-trace-label-aware](../../planned/v0_36_0/m-trace-label-aware.md): the general trace/label fix. This design avoids depending on it

## References

- `internal/apiserver/{server.go,routes.go,routes_dispatch.go,auth.go,authorized_surface.go,openapi.go,mcp.go}`
- `internal/effects/{stream.go,stream_ws.go,stream_mux.go,stream_source.go,stream_context.go,stream_transport.go,ops.go,redact.go,context.go}`
- `internal/platform/streamws/streamws.go`, `internal/auth/gcp/adc.go`, `cmd/ailang/serve_api.go`, `std/stream.ail`, `std/secret.ail`
- RFC 6455 §4.1/§10.2 (Origin), WHATWG WebSocket API

## Future Work

- `StreamEvent.Binary(bytes)` migration (G5 for the non-bridge API) and a source tag on `Message`/`Binary` in `selectEvents` (G3 for the non-bridge API).
- `Tick` frames and "emit to the other side" verdicts, if a second consumer needs them.
- More credential providers (`secret:op://…` through `std/secret`'s resolver, kept host-side).
- Network record/replay of bridge sessions (the unimplemented M-STREAM-BIDI replay contract).

## Implementation Report

Implemented 2026-09-25 on `feat/serveapi-websocket-bridge` (base: #1313).

| Milestone | Where | Tests |
|---|---|---|
| M1 accept path | `internal/parser/parser_decl.go` (`"WS"`), `internal/apiserver/routes_ws.go`, `authorized_surface.go` (`IsWS`), `internal/platform/streamws/accept.go`, `effects.AdoptStreamConnection`, `StreamContext.Child`, `embed.Engine.CallPrepared` / `runtime.CallEntrypointPrepared`, `cmd/ailang/serve_api_ws.go` | `routes_ws_test.go`: echo (A1), 426/403/503 (A2), OpenAPI/MCP/catch-all exclusion (A3), session isolation (A4), startup validation, API-key subprotocol, shutdown 1001 |
| M2 bridge fold | `std/stream/bridge.ail`, `internal/builtins/stream_bridge.go`, `internal/effects/stream_bridge.go` | `stream_bridge_test.go`: 1,000 mixed frames each way (A1), speech gate strict subset (A2/A3), barge-in (A4), `StepFailed` + 1011 (A5), client close under 100 ms (A6), `CloseBridge` codes, Replace/UpstreamClosed, `Stream.recv`/`Stream.send` budget |
| M3 credential + G7 | `internal/platform/streamcred`, `applyStreamCredential` in `stream_ws.go`, `internal/eval/trace_redact.go` | `stream_credential_test.go` (binding fires over TLS, exact host/port/scheme, double-auth refusal, trace redaction), `streamcred_test.go`, `trace_redact_test.go`, and the end-to-end canary `routes_ws_canary_test.go` |
| M4 decision log + example | `decisionLogger` in `routes_ws.go`, `examples/serveapi_ws_bridge.ail`, guide section "WebSocket Routes & the Bridge" | canary test asserts one line per frame and no payloads |
| M5 measurement | `internal/apiserver/routes_ws_bench_test.go` | recorded below |

**Canary (Success metric 4).** A token planted through the binding reaches the TLS fake upstream (positive control) and appears in 0 client frames, 0 bytes of the deep trace file, 0 lines of stderr/server log and 0 decision-log lines.

**Mutation checks.** Each was reverted and the named test failed, then restored: the Origin check (`TestWS_RefusedBeforeUpgrade`), `Child()` replaced by the shared context (`TestWS_SessionsAreIsolated`, probe saw "Open Open"), `Forward` sending nothing (four bridge tests), the `wss`-only guard removed in both the core and the binder (`TestCredentialBinding_ExactHostPortSchemeOnly`), the G7 renderer reverted (`TestTrace_RedactsProgramAuthorizationHeader`), and the credential copied into the program's `connect` config with G7 reverted (`TestWS_CredentialCanary`). With G7 in place, that last leak was still withheld from the trace: the defence in depth works.

**M5 numbers** (Apple M4 Max, `go test ./internal/apiserver/ -run NONE -bench WSRelay -benchtime 5000x -count 3`). One iteration is one 22 KB frame from client to serve-api to an echo upstream and back, so it includes two AILANG step calls and two relay hops:

| Path | p50 RTT | p95 RTT |
|---|---|---|
| Direct to the echo upstream (baseline) | 35 µs | 84 µs |
| Through the bridge, binary frames (`UpBin` gate) | 114 µs | 224–239 µs |
| Through the bridge, text frames (string-scan gate) | 128 µs | 254–261 µs |

The added latency is about (257 − 84) / 2 ≈ 87 µs per frame at p95, well under the 2 ms target. Sequential round trips sustain about 7,000 frames/s per connection, against the 200 frames/s target, with 0 frames lost (the benchmark checks every echoed frame's size). RSS was not measured separately; the per-direction queue bound (64 frames × 64 KiB) caps queued memory at 4 MiB.

**Not done / follow-ups.**
- The Daneel acceptance against live `BidiGenerateContent` (outside CI) and the P1 captured Vertex frames remain Daneel's to run.
- `@raw` routes still receive the request's headers (cookies included) as a Json value. The trace renderer now withholds credential-named entries, but a program that `println`s them is not covered.
- `ailang server`'s own WebSocket (`internal/websocket/server.go`) still accepts any Origin. That is out of scope here, as #1313 also noted.
- `--watch` reload with live sessions: sessions finish on the code they started with (no special handling was added).


# M-STREAM-SERVE-API: Streaming Proxy for serve-api

**Status**: REJECTED
**Target**: ~~v0.9.0~~
**Priority**: ~~Medium~~
**Depends on**: M-STREAM-BIDI (implemented), SSE support (implemented)
**Rejected**: 2026-02-16

## Rejection Rationale

**Streaming proxy is a Go-level concern, not a language-level concern.**

The aggregate-then-return pattern (consume stream → compute → return JSON) is the right fit for AILANG's deterministic computation model. A streaming proxy adds significant runtime complexity for a use case better served by existing tools:

1. **Double-hop latency**: client → serve-api → upstream → serve-api → client adds a middleman for what is often just passthrough. A 30-line Go `net/http` handler or nginx/Cloudflare proxy does this with zero overhead.

2. **AILANG's evaluator isn't built for throughput**: Each event would go through `CallValue` dispatch. The evaluator is optimized for correctness and determinism, not I/O proxying.

3. **Significant runtime complexity for marginal value**: Per-request EffContext cloning, `yield` effect, `CallWithContext` on the evaluator, goroutine-per-request streaming — ~620 LOC of plumbing to route bytes through an evaluator that doesn't need to see them.

4. **The aggregate-then-return pattern already works**: AILANG functions can consume SSE/WebSocket streams internally, do computation (transform, filter, accumulate), and return structured results. This is deterministic, testable, and plays to AILANG's strengths.

5. **Transform-in-stream is the only compelling use case**, and it's niche enough to not justify the complexity. If it emerges as a real need, revisit then.

**What DOES work (v0.8.1):**
- `serve-api --caps Stream` wires StreamContext + FnCaller
- AILANG functions consume streams internally and return aggregated JSON
- No proxy needed — the function IS the computation

## Original Problem Statement

`ailang serve-api` exposes AILANG module exports as REST endpoints with synchronous JSON responses. AILANG functions can now consume SSE and WebSocket streams internally (via the Stream effect), but when called through serve-api, the entire stream must be consumed before a single JSON response is returned. This "aggregate-then-return" pattern works but prevents real-time streaming to HTTP clients.

AI applications commonly need to proxy upstream token streams (e.g., Anthropic SSE) to frontend clients in real time. Currently this requires a separate Go/Node proxy layer outside AILANG.

## Current State (v0.8.1)

```
HTTP Client ──POST──► serve-api ──Call()──► AILANG function
                                               │
                                               ├── sseConnect(upstream_url)
                                               ├── collect all events
                                               └── return aggregated result
                                               │
HTTP Client ◄──JSON──── serve-api ◄──result────┘
```

- Stream + FnCaller wired in serve-api (v0.8.1)
- AILANG functions can consume SSE/WebSocket streams
- serve-api waits for Call() to return, then sends one JSON response
- No way to stream partial results to the HTTP client

## Proposed Design

### Architecture

```
HTTP Client ◄──SSE──► serve-api ──Call()──► AILANG function
     │                    │                      │
     │                    │                      ├── sseConnect(upstream_url)
     │                    │                      ├── onEvent(handler) ──┐
     │                    │                      │                     │
     │                    ├── SSE event ◄────────┤◄── yield(event) ◄───┘
     │◄── SSE event ◄────┤                      │
     │                    ├── SSE event ◄────────┤◄── yield(event)
     │◄── SSE event ◄────┤                      │
     │                    │                      └── return final
     │◄── [DONE] ◄───────┤◄── result ───────────┘
```

### Key Components

#### 1. Streaming Response Path in serve-api

New endpoint pattern that detects streaming AILANG functions and switches to SSE output:

```
POST /api/{module}/{function}
Accept: text/event-stream        ← Client requests streaming
```

When the client sends `Accept: text/event-stream`, serve-api:
1. Sets response headers for SSE (`Content-Type: text/event-stream`, `Cache-Control: no-cache`)
2. Calls the AILANG function in a goroutine
3. Reads from a yield channel, forwarding each event as an SSE data frame
4. Sends `[DONE]` sentinel when the function returns

Fallback: If `Accept` is `application/json` (or absent), use the existing aggregate-then-return path. This preserves backward compatibility.

#### 2. `yield` Effect Operation

New effect operation under the `Stream` capability:

```ailang
-- In std/stream.ail
export func yield(value: string) -> unit ! {Stream} {
  _stream_yield(value)
}
```

`yield` pushes a value to the serve-api response channel. When NOT running under serve-api (e.g., CLI `ailang run`), `yield` is a no-op or writes to stdout.

Implementation:
- `RegisterOp("Stream", "yield", StreamYield)` in `internal/effects/stream_yield.go`
- `StreamYield` checks for a `YieldSink` channel on the EffContext
- If present: sends value to channel (blocking with timeout)
- If absent: no-op (function runs standalone, not behind serve-api)

#### 3. YieldSink on EffContext

```go
// internal/effects/eff_context.go
type EffContext struct {
    // ... existing fields ...
    YieldSink chan<- string  // nil when not serving; set by serve-api handler
}
```

The serve-api streaming handler creates a channel and sets `effCtx.YieldSink` before calling the AILANG function. The function's `yield` calls write to this channel. The handler goroutine reads from the channel and writes SSE frames to `http.ResponseWriter`.

**Per-request isolation**: Each streaming request gets its own `EffContext` clone with its own `YieldSink`. The base `EffContext` from server startup is used as a template.

#### 4. AILANG Proxy Pattern

```ailang
module api/stream_proxy

import std/stream (sseConnect, onEvent, runEventLoop, disconnect, defaultConfig, yield)

-- Proxy an upstream SSE stream to the HTTP client
export func proxyAnthropic(prompt: string) -> string ! {Stream, Net, IO} {
  let config = defaultConfig(());
  let conn = sseConnect("https://api.anthropic.com/v1/messages", config);

  let handler = func(event: string) -> bool {
    yield(event);  -- Forward each event to the HTTP client
    true           -- Continue receiving
  };

  onEvent(conn, handler);
  runEventLoop(conn);
  disconnect(conn);
  "done"
}
```

#### 5. serve-api Handler Changes

In `internal/apiserver/handler.go`:

```go
func (s *Server) handleFunctionCall(w http.ResponseWriter, r *http.Request) {
    // ... existing module/function validation ...

    // Check if client wants streaming
    if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
        s.handleStreamingCall(w, r, modulePath, funcName, args)
        return
    }

    // ... existing synchronous path ...
}

func (s *Server) handleStreamingCall(w http.ResponseWriter, r *http.Request,
    modulePath, funcName string, args []interface{}) {

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    // Create per-request yield channel
    yieldCh := make(chan string, 64)

    // Clone EffContext with YieldSink for this request
    reqCtx := s.cloneEffContextWithYield(yieldCh)

    // Run AILANG function in goroutine
    done := make(chan error, 1)
    go func() {
        _, err := s.engine.CallWithContext(reqCtx, modulePath, funcName, args...)
        close(yieldCh)
        done <- err
    }()

    // Stream events to client
    for event := range yieldCh {
        fmt.Fprintf(w, "data: %s\n\n", event)
        flusher.Flush()
    }

    // Send completion
    if err := <-done; err != nil {
        fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
    } else {
        fmt.Fprintf(w, "data: [DONE]\n\n")
    }
    flusher.Flush()
}
```

### New API Required on embed.Engine

```go
// CallWithContext is like Call but uses a custom EffContext for this invocation.
// Used by serve-api to inject per-request YieldSink.
func (e *Engine) CallWithContext(effCtx interface{}, modulePath, funcName string, args ...interface{}) (eval.Value, error)
```

This requires the evaluator to accept a per-call EffContext override, which is a non-trivial change to the runtime. The cleanest approach is to use Go's `context.Context` with a value key, or to add an EffContext parameter to `runtime.CallEntrypoint`.

## Files to Change

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/effects/stream_yield.go` | NEW: `StreamYield` operation + `RegisterOp` | ~60 |
| `internal/effects/eff_context.go` | Add `YieldSink chan<- string` field + `Clone()` method | ~30 |
| `internal/builtins/stream.go` | Register `_stream_yield` builtin | ~25 |
| `std/stream.ail` | Add `yield()` wrapper | ~5 |
| `internal/apiserver/handler.go` | Add `handleStreamingCall` + Accept negotiation | ~80 |
| `internal/apiserver/server.go` | Add `cloneEffContextWithYield` helper | ~20 |
| `internal/embed/embed.go` | Add `CallWithContext` method | ~40 |
| `internal/eval/evaluator.go` | Support per-call EffContext override | ~30 |
| `internal/apiserver/handler_stream_test.go` | NEW: streaming handler tests | ~200 |
| `internal/effects/stream_yield_test.go` | NEW: yield operation tests | ~100 |
| `examples/runnable/stream_proxy.ail` | NEW: proxy example | ~30 |
| **Total** | | **~620** |

## Design Decisions

### Why `yield` as a new effect operation (not reusing `onEvent`)?

`onEvent` is a consumer-side mechanism (receive events from upstream). `yield` is a producer-side mechanism (push events to downstream HTTP client). These are conceptually different and keeping them separate maintains clarity.

### Why per-request EffContext cloning?

The `YieldSink` channel is per-request state. Sharing a single EffContext across concurrent requests would cause events from different requests to interleave. Each streaming request needs its own channel.

### Why `Accept` header negotiation?

This follows HTTP content negotiation standards. The same endpoint works for both synchronous (`application/json`) and streaming (`text/event-stream`) clients. No URL changes needed.

### Why not WebSocket for the downstream path?

SSE is simpler for unidirectional streaming (which is the common case for AI token streaming). WebSocket adds bidirectional complexity that isn't needed for proxy scenarios. If bidirectional downstream is needed in the future, a separate WebSocket endpoint can be added.

## Open Questions

1. **EffContext cloning depth**: Should `Clone()` deep-copy the Stream context (connection tracking) or share it? Sharing allows the cloned context to reuse the connection pool; deep-copy provides isolation.

2. **Backpressure**: What happens when the HTTP client reads slower than upstream produces? The `yieldCh` buffer (64) provides some slack. Beyond that, `yield` blocks, which backpressures the AILANG function, which backpressures the upstream SSE read. This is probably correct behavior but needs testing.

3. **Per-call EffContext in evaluator**: The current evaluator stores EffContext globally. Supporting per-call override may require threading it through `CallEntrypoint` or using goroutine-local storage (not idiomatic Go). Need to evaluate the cleanest approach.

4. **Authentication passthrough**: Should serve-api forward the client's `Authorization` header to upstream connections made by AILANG functions? This enables "transparent proxy" patterns but has security implications.

## Verification Plan

1. Unit tests for `StreamYield` operation (with/without YieldSink)
2. Integration test: mock upstream SSE server -> AILANG proxy function -> serve-api -> test HTTP client receiving SSE
3. Backpressure test: slow client, fast upstream
4. Concurrent requests test: multiple clients streaming simultaneously
5. Error propagation test: upstream failure mid-stream
6. Content negotiation test: same endpoint, JSON vs SSE based on Accept header

## Timeline Estimate

~3-4 days implementation, based on current velocity of ~150 LOC/day for runtime work.

## References

- [M-STREAM-BIDI design doc](../v0_8_1/m-stream-bidi-primitives.md) - Foundation
- [serve-api implementation](../../internal/apiserver/) - Current server
- [SSE spec (WHATWG)](https://html.spec.whatwg.org/multipage/server-sent-events.html)

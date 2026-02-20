# Sprint Plan: M-STREAM-BIDI Phase 1 — Core WebSocket Streaming

## Summary

Implement Phase 1 of bidirectional streaming primitives: `Stream` effect registration, `StreamContext` security model, WebSocket connect/send/close, callback-based event dispatch with bounded buffer, and the `std/stream.ail` AILANG module. Scoped to WebSocket only (SSE deferred to Phase 2 sprint).

**Duration:** 5 milestones (~4 days of focused work)
**Dependencies:** None — gorilla/websocket already in go.mod, budget system exists
**Risk Level:** Medium (new effect + new builtin family + callback dispatch is non-trivial)
**Design Doc:** [m-stream-bidi-primitives.md](m-stream-bidi-primitives.md)

## Current Status Analysis

### Completed Recently (last 14 days)
- M-CONTRACT-EVAL: contract-guided evaluation harness (~500 LOC)
- SMT bounded recursion + cross-function + record/string/list verification (~1,380 LOC impl + ~1,460 LOC tests)
- AI devtools ergonomics Tier 2 (ai-check, compact prompts)
- Eval harness chain hookup

### Velocity
- Recent average: ~200-300 LOC/day (impl + tests combined)
- Estimated capacity: ~1,200 LOC over 4 focused days
- Infrastructure advantage: gorilla/websocket, BudgetContext, Capability system all exist

### Remaining from Design Doc
- Phase 1: Core Infrastructure + Builtins + AILANG Module (~2,500 impl + ~1,200 tests)
- Phase 2: SSE Support (deferred)
- Phase 3: Replay Contract (deferred)
- Phase 4: Examples + Docs (partial — one example in this sprint)
- Phase 5: Hardening (deferred)

### Scoping Decision

The full design doc specifies ~160 hours / 5 weeks. This sprint covers the **critical path** only:
- `Stream` effect + `StreamContext` + WebSocket backend
- 6 core builtins (connect, send, onEvent, runEventLoop, close, status)
- `withStream` helper
- `std/stream.ail` with all types
- One working WebSocket echo example
- Unit tests for all new code

**Explicitly deferred to follow-up sprints:**
- SSE backend (Phase 3 of design doc)
- Trace record/replay (Phase 4)
- `sseInfo()` builtin
- Voice agent / Gemini Live example
- Security hardening (DNS rebinding audit, stress testing)
- `ailang doctor stream`

## Proposed Milestones

### Milestone 1: Stream Effect Registration + StreamContext
**Goal:** Register `Stream` in the effect registry, create `StreamContext` with security model, wire into `EffContext` and CLI capability parsing.

**Estimated:** ~250 LOC implementation + ~150 LOC tests = ~400 LOC

**Tasks:**
1. Add `"Stream": {}` to `Registry` in `internal/effects/ops.go`
2. Create `internal/effects/stream_context.go`:
   - `StreamContext` struct (MaxConnections, MaxMessageSize, MaxFrameSize, timeouts, AllowHTTP, BlockPrivateIPs, AllowedDomains, EventBufferSize, sub-budget fields)
   - `NewStreamContext()` with defaults
   - `ValidateURL()` — domain allowlist, protocol check, private IP blocking
   - `AcquireConnection()` / `ReleaseConnection()` — connection count tracking
   - Sub-budget helpers for Stream.connect / Stream.send / Stream.recv
3. Add `Stream *StreamContext` field to `EffContext` in `internal/effects/context.go`
4. Wire `--caps Stream` in `cmd/ailang/main.go` `grantCapabilities()` function
5. Add `--stream-allow-http`, `--stream-allow-domains` CLI flags
6. Unit tests: `internal/effects/stream_context_test.go`
   - Connection limit enforcement (4 → 5th rejected)
   - URL validation (domain allowlist, wss/ws enforcement, private IP blocking)
   - Sub-budget tracking

**Acceptance Criteria:**
- [ ] `effects.Registry["Stream"]` exists
- [ ] `EffContext.Stream` is populated when `--caps Stream` is passed
- [ ] URL validation blocks ws:// by default, allows wss://
- [ ] Private IPs (10.x, 172.16.x, 192.168.x, 169.254.x) are blocked by default
- [ ] Connection count limit works (default 4)
- [ ] All tests passing, lint clean

**Risks:**
- Private IP detection edge cases (IPv6 link-local) — Mitigation: start with IPv4 only, add IPv6 later

---

### Milestone 2: StreamConnection + WebSocket Backend
**Goal:** Implement the Go-level WebSocket connection management: connect, send, close, bounded event buffer, internal read goroutine.

**Estimated:** ~350 LOC implementation + ~200 LOC tests = ~550 LOC

**Tasks:**
1. Create `internal/effects/stream.go`:
   - `StreamConnection` struct (conn, eventBuffer chan, metrics, handler, mu, status)
   - `StreamProtocol` Go enum (WebSocket, SSE)
   - `streamConnect()` — gorilla/websocket `Dialer`, TLS config, subprotocol negotiation
   - `streamSend()` — write lock, message type dispatch (Text/Binary), size check
   - `streamClose()` — graceful close handshake (CloseMessage, deadline)
   - Internal read goroutine: reads frames → converts to StreamEvent → pushes to bounded buffer
   - `streamStatus()` — return connection state
2. Register effect operations:
   ```go
   RegisterOp("Stream", "connect", streamConnect)
   RegisterOp("Stream", "send", streamSend)
   RegisterOp("Stream", "onEvent", streamOnEvent)
   RegisterOp("Stream", "runEventLoop", streamRunEventLoop)
   RegisterOp("Stream", "close", streamClose)
   RegisterOp("Stream", "status", streamStatus)
   ```
3. Event buffer: bounded channel (cap from StreamContext.EventBufferSize, default 1000)
4. Unit tests: `internal/effects/stream_test.go`
   - Use `httptest` + gorilla websocket `Upgrader` for test server
   - Connect → send → receive → close lifecycle
   - Binary frame round-trip
   - Connection close propagates correctly
   - Bounded buffer backpressure (fill buffer, verify blocks)

**Acceptance Criteria:**
- [ ] WebSocket connection to test server succeeds
- [ ] Text and binary messages send/receive round-trip
- [ ] Graceful close handshake completes
- [ ] Bounded event buffer enforces capacity
- [ ] Read goroutine stops on close
- [ ] All tests passing, lint clean

**Risks:**
- gorilla/websocket Upgrader test server setup — Mitigation: well-documented pattern in gorilla examples
- Race conditions in read goroutine — Mitigation: careful channel/mutex design, run tests with `-race`

---

### Milestone 3: Callback Dispatch + Event Loop
**Goal:** Implement `onEvent` handler registration and `runEventLoop` with serialized dispatch, panic recovery, and `send()` safety from within handlers.

**Estimated:** ~200 LOC implementation + ~150 LOC tests = ~350 LOC

**Tasks:**
1. `streamOnEvent()` — store handler `eval.Value` (FunctionValue) in `StreamConnection`
2. `streamRunEventLoop()`:
   - Single dispatch loop: read from event buffer channel
   - For each event: construct AILANG ADT value, call handler via eval engine
   - Check handler return value (BoolValue true → continue, false → break)
   - Panic recovery: `defer recover()` → deliver `Error(ProtocolError("handler panic: ..."))`
   - Timeout checking: select on event buffer + idle timer
   - Budget checking: decrement `Stream.recv` before each dispatch
   - Return `UnitValue` when loop terminates
3. Ensure `send()` from handler: verify write lock is NOT held during handler call
4. Unit tests:
   - Handler returning false stops loop
   - Handler panic → Error event delivered, loop continues
   - send() from within handler (no deadlock)
   - Idle timeout → Error(Timeout) event
   - Stream.recv budget exhaustion → Error(BudgetExhausted) event
   - Serialized dispatch (events one at a time)

**Acceptance Criteria:**
- [ ] Handler receives events sequentially
- [ ] Handler returning false terminates loop
- [ ] Handler panics are caught and delivered as Error events
- [ ] send() works from inside handler without deadlock
- [ ] Idle timeout delivers Timeout error event
- [ ] recv budget exhaustion delivers BudgetExhausted error event
- [ ] All tests passing with `-race`, lint clean

**Risks:**
- Calling AILANG eval from Go context (handler invocation) — Mitigation: follow existing pattern from other effect callbacks
- Deadlock between write lock and dispatch — Mitigation: explicit lock ordering + test with `-race`

---

### Milestone 4: Builtin Registration + std/stream.ail Module
**Goal:** Register all builtins via `BuiltinSpec`, create the AILANG `std/stream.ail` module with all type definitions, and implement `withStream` helper.

**Estimated:** ~300 LOC implementation (builtins + stdlib) + ~150 LOC tests = ~450 LOC

**Tasks:**
1. Create `internal/builtins/stream.go`:
   - `registerStreamConnect()` — 2 args (url: string, config: StreamConfig), returns Result[StreamConn, StreamError]
   - `registerStreamSend()` — 2 args (conn: StreamConn, msg: StreamMessage), returns Result[unit, StreamError]
   - `registerStreamOnEvent()` — 2 args (conn: StreamConn, handler: StreamEvent → bool ! {e})
   - `registerStreamRunEventLoop()` — 1 arg (conn: StreamConn), returns unit
   - `registerStreamClose()` — 1 arg (conn: StreamConn), returns unit
   - `registerStreamStatus()` — 1 arg (conn: StreamConn), returns StreamStatus
   - `registerStreamWithStream()` — 3 args (url, config, handler), returns Result[unit, StreamError]
   - Type builder functions for all types (StreamConn, StreamConfig, StreamProtocol, StreamEvent, StreamOpenInfo, StreamError, StreamMessage, StreamStatus)
2. Create `stdlib/stream.ail`:
   - Module declaration, imports
   - All ADT exports (StreamConn, StreamProtocol, StreamConfig, StreamEvent, StreamOpenInfo, StreamError, StreamMessage, StreamStatus)
   - Public function wrappers (connect, send, onEvent, runEventLoop, close, status, withStream)
3. Unit tests: `internal/builtins/stream_test.go`
   - Type signature validation for all 7 builtins
   - Argument count validation
   - Effect tag verification ("Stream")
   - Registration doesn't panic

**Acceptance Criteria:**
- [ ] `ailang doctor builtins` shows all 7 stream builtins
- [ ] Type signatures match design doc
- [ ] `std/stream` module compiles (`ailang check stdlib/stream.ail`)
- [ ] All ADT types are correctly exported
- [ ] `withStream` is defined and type-checks
- [ ] All tests passing, lint clean

**Risks:**
- Complex type signatures (effectful handler with row variable) — Mitigation: study existing patterns in builtins/net.go for Result types
- ADT registration for new types — Mitigation: follow std/net.ail pattern exactly

---

### Milestone 5: Integration Test + WebSocket Echo Example
**Goal:** End-to-end integration test with a real WebSocket connection, plus a working `examples/stream_websocket.ail` example file.

**Estimated:** ~100 LOC implementation + ~100 LOC tests = ~200 LOC

**Tasks:**
1. Create `examples/runnable/stream_websocket.ail`:
   - Uses `withStream` helper
   - Connects to test echo server
   - Sends message, receives echo, prints, closes
2. Integration test in `internal/effects/stream_integration_test.go`:
   - Spin up `httptest` WebSocket server
   - Run full AILANG program through pipeline
   - Verify output matches expected
3. Verify `make verify-examples` includes new example
4. Update CHANGELOG.md with new feature
5. Update README.md implementation status

**Acceptance Criteria:**
- [ ] `examples/runnable/stream_websocket.ail` compiles and runs
- [ ] Integration test passes end-to-end
- [ ] `make test` passes (all existing + new tests)
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated
- [ ] README.md updated

**Risks:**
- Full pipeline integration (AILANG → parse → typecheck → elaborate → eval → effects) may surface unexpected type issues — Mitigation: test incrementally, check each phase

---

## Success Metrics

- Test coverage: >85% for new `internal/effects/stream*.go` and `internal/builtins/stream*.go`
- Examples passing: `stream_websocket.ail` in `make verify-examples`
- Documentation: CHANGELOG.md, README.md updated
- All tests passing: `make test` ✅
- All linting passing: `make lint` ✅
- New builtins validated: `ailang doctor builtins` ✅

## Dependencies

- `gorilla/websocket` — Already in go.mod ✅
- Budget system (`BudgetContext`) — Already implemented (v0.7.0) ✅
- Capability parsing (`--caps`) — Already implemented ✅
- Effect system (`EffContext`, `Registry`) — Already implemented ✅

## Open Questions

None blocking — all design decisions resolved in Revision 2 of the design doc.

## Notes

- This sprint covers Phase 1 + Phase 2 of the design doc (core infra + builtins), but NOT Phase 3 (SSE), Phase 4 (replay), or Phase 5 (hardening)
- The `sseInfo()` builtin is deferred since SSE backend is not in this sprint
- Total sprint LOC: ~1,200 impl + ~750 tests ≈ 1,950 LOC
- Voice agent example deferred — too dependent on actual API auth setup
- Frame size vs message size distinction tracked in StreamContext but gorilla handles frame reassembly internally

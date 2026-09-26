# Sprint Plan: M-SERVEAPI-WS-BRIDGE

**Design doc**: [m-serveapi-websocket-bridge.md](m-serveapi-websocket-bridge.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_m_serveapi_ws_bridge.json`
**Status**: ✅ Completed 2026-09-25 (all five milestones in one PR) · **Target**: v0.44.0 · **Duration**: ~4 days of work (one agent session, M1–M3 first) · **Risk**: medium-high (parser, effects core, credential handling)
**Base**: PR #1313 (`feat/serveapi-bind-host-cors`) — M0 of the design doc.
**Created**: 2026-09-25

## Decisions ratified (Mark, 2026-09-25)

- **D1**: route + bridge fold (option C).
- **D3**: the upstream credential is an **explicitly configured source** for one upstream host — a service-account / impersonation credentials file, the metadata server named explicitly, or a token file — never the Studio's implicit `gcloud` user ADC. It never becomes an AILANG value and never appears in a frame or trace.
- **D4**: tailnet bind + Origin allowlist for v1; a key is optional, not required.
- **Sequencing**: M0 = #1313 (`--bind` loopback default, `--cors-origin` allowlist). Not reimplemented here.

## Planner decisions (recorded in the design doc)

- **No `--ws-allow-any-interface`.** #1313 makes loopback the default, so the escape hatch has nothing to protect. Instead: serve-api **refuses to start** when a `WS` route is registered, the listen host is not loopback, and no `--cors-origin` allowlist is set. `--cors` (every origin) does not count as an allowlist for WS.
- **WS Origin rule**: the `--cors-origin` list *is* the WS Origin allowlist (one list for both). Same-origin (`Origin` host == request `Host`) is always allowed. A missing Origin or `Origin: null` is refused (403, before upgrade).
- **Credential flag**: `--stream-credential HOST=SOURCE`, repeatable. Sources: `gcp-key-file:PATH` (any Google credentials JSON: service account, impersonated service account, external account), `gcp-metadata` (the GCE/Cloud Run metadata server, named explicitly), `bearer-file:PATH` (a token file some other tool refreshes). No implicit default and no gcloud fallback.
- **Queue bound**: WS sessions get a child `StreamContext` with `EventBufferSize = --ws-queue-frames` (default 64). The byte bound falls out of the per-frame read limit (64 × 64 KiB = 4 MiB per direction), so there is no second byte counter.
- **When `bridge` returns, both legs are closed.** A handler's later `disconnect` calls are idempotent.

## Milestones

### M1: `@route("WS")` accept path ✅ (~450 LOC impl, ~350 LOC tests)
- Parser: `"WS"` in the `@route` method set; messages list it.
- `ExportInfo.IsWS`; excluded from OpenAPI, MCP, A2A and the `/api/{module}/{func}` catch-all (`loadedExportMember`).
- Registration check: first param `StreamConn`, `Stream` in the effect row, `Stream` granted by `--caps`, no `@raw`/`@nowrap`; loopback-or-allowlist bind rule.
- `internal/platform/streamws.Accept` (server-side `StreamTransport`); `effects.AdoptStreamConnection`.
- `StreamContext.Child()`; `embed.Engine.CallPrepared` + `runtime.CallEntrypointPrepared` so the WS call's forked `EffContext` gets the child context (G6).
- `wsHandler`: 426 for a plain GET, 403 for Origin, 503 over `--ws-max-sessions`, all before upgrade; close 1000/1011 on return; 1001 on shutdown.
- Flags: `--ws-max-sessions`, `--ws-queue-frames`, `--stream-max-duration`, `--stream-idle-timeout`.
- [x] **Accept**: A1 echo round-trips text+binary; A2 426/403/503 before upgrade; A3 WS export absent from OpenAPI and MCP; A4 two sessions cannot see each other's connection IDs; A5 existing apiserver tests unmodified.

### M2: `std/stream/bridge` fold ✅ (~300 LOC impl, ~400 LOC tests)
- `std/stream/bridge.ail` (types + `bridge` wrapper), `_stream_bridge` builtin → `effects.StreamBridge`.
- Go loop: two-source round-robin merge, step via `FnCallerN`, verdicts on the original `[]byte`, typed `BridgeEnd`, idle/max-duration timers, `Stream.recv`/`Stream.send` charged per frame, close-with-code.
- [x] **Accept**: A1 Forward-all relays 1,000 mixed frames each way byte-identical in order; A2/A3 speech-gate fixture delivers a strict subset; A4 barge-in drops queued stale frames; A5 raising step → `StepFailed` + client close 1011; A6 client close → `ClientClosed` within 100 ms and the upstream closed.

### M3: credential binding + G7 trace redaction ✅ (~250 LOC impl, ~300 LOC tests)
- `StreamContext.Credentials` hook (exact host, `wss` only, applied after the authorizer), double-auth refusal, `internal/platform/streamcred` sources with token caching (5 min early refresh) and startup prefetch; `--stream-credential`.
- **G7 unified fix**: every trace rendering site goes through `EffContext.RenderTraceValue`; it now redacts sensitive header values (`{name|key: "Authorization", value: …}` records, `(name, value)` tuples, record fields named like credentials, and `Bearer …`/`Basic …` strings).
- [x] **Accept**: A1 canary (0 hits in client frames, deep trace, stderr, decision log); A2 fake TLS upstream received the header; A3 no binding for another host / `ws://` / other port; A4 program-supplied Authorization to a bound host refused; A5 unbound `Authorization` is `[REDACTED]` in the trace.

### M4: decision log + example + docs ✅ (~120 LOC, docs)
- [x] One server-log line per frame `{call_id, seq, dir, kind, bytes, verdict, step_us}` behind `--ws-decision-log`; never a payload.
- [x] `examples/serveapi_ws_bridge.ail`, guide section in `docs/docs/guides/serve-api.md`, CHANGELOG.

### M5: measurement ✅ (~100 LOC bench)
- [x] Go benchmark with the fake upstream: step cost and relay latency p50/p95; numbers recorded in the design doc (p95 ≈ 87 µs added per frame).

## Risks

- **Scope**: if the session runs short, M1+M2+M3 (Daneel's DONE WHEN) ship first; M4/M5 follow.
- The `budgetChargeDepth` re-entrancy guard would swallow per-frame charges inside the builtin; the bridge charges with the scope saved and reset.
- `http.Server.Shutdown` does not close hijacked connections; tracked with `RegisterOnShutdown`.

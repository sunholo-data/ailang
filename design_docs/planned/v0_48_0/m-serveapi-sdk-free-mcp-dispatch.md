# M-SERVEAPI-SDK-FREE-MCP: a stdlib-only MCP dispatcher inside `serveapi` (#885)

**Status**: Implemented on branch `design/m-serveapi-mcp-dispatch-seam` (2026-09-28; unreleased). D-A (a′), D-B (i), D-F accept: ruled by Mark 2026-09-28. AC1–AC8 met; AC-W1..W3 await a release Quorum round 1 and round 2 were both BLOCKED 3/3. Every objection was accepted or refuted with a measurement (§Quorum record). The re-quorum-once guardrail is spent, so round 3 runs **after** Mark's D-A ruling, not before
**Target**: v0.48.0
**Priority**: P1 — the only upstream item blocking ailang-world's MCP projection
(`w-mcp-dispatch-projection`, charter clause 6); ranked `[NEXT]` at `design_docs/v1-mission.md:1011`
**Estimated**: 3–4 days (M1 1.5d, M2 1d, M3 0.5–1d, M4 0.5d)
**Dependencies**: None. Builds on `serveapi/protocol` (#764, shipped v0.33.2) and the embedded
handler contract (#498 Lane A/B).
**Issue**: [#885](https://github.com/sunholo-data/ailang/issues/885) (`cross-mission`, filed by the
ailang-world mission). Triage row:
[ailang-core-triage/serveapi-protocol-mcp-dispatch.md](../ailang-core-triage/serveapi-protocol-mcp-dispatch.md)
(recommend: design-doc).

---

## Measurement provenance

Every number below was re-derived by the doc author in a clean worktree at **`bb0b0fe99`
(= origin/dev), darwin/arm64, go1.26.6, 2026-09-28**, using the command shown in the Verification Log.
World-side facts were read from the local `ailang-world` checkout at **`58f6022`** (dev, 2026-09-28).
The wire-behaviour table (V10) comes from a scratch test file,
`serveapi/zz_probe_wire_test.go`, that drove the **current** SDK-backed handler. It was deleted
after the run, and `git status --short` was empty afterwards.

This doc makes no claim about AILANG language semantics (it covers Go packaging and the wire
protocol), so the `ailang check` gate does not apply.

**Quorum triggers fired:** **#1** (design-freeze items D-A, D-B, D-F) and **#4** (load-bearing premises
about MCP, a protocol we do not control; the SDK's behaviour is used as the reference). Run
`ailang design-quorum` before planning.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The dispatcher is a pure function of (request bytes, host callbacks). The SDK path allocates a new `mcp.Server` per request, then a buffered transport replays its headers, and emitted fields depend on the SDK's version (V10 shows `ttlMs`/`cacheScope` appearing from SDK 1.8.0). The new path fixes the output by our own golden fixtures. |
| A2: Replayability | +1 | Same stateless POST semantics as today, but the reply bytes are now defined in-repo and asserted by golden tests. A replay no longer depends on which SDK version is linked. |
| A3: Effect Legibility | 0 | No change to AILANG's effect surface. |
| A4: Explicit Authority | +1 | Removes an OAuth/credential stack (`golang.org/x/oauth2`, `go-sdk/auth`, `go-sdk/oauthex`, V6) from the closure of the package embedders link. Authority stays entirely in the host's `SessionResolver`. |
| A5: Bounded Verification | +1 | The closure gate for `serveapi` tightens from "an 11-module-root allowlist" to **exact package-path** sets per package (stdlib plus named ailang packages only), a mechanically checkable invariant (V13, AC1). |
| A6: Safe Concurrency | 0 | Reuses the existing `callbackRunner` unchanged (bounded slots, timeout). No new shared state. |
| A7: Machines First | +1 | Consumers under a zero-cloud policy (World) get MCP without writing a codec. Error replies become JSON-RPC objects instead of `text/plain` 400s for a whole class of inputs (V10 rows 7 and 10), which machines can parse. |
| A8: Minimal Syntax | 0 | No language change. |
| A9: Cost Visibility | 0 | No cost surface. |
| A10: Composability | +1 | MCP and A2A become reachable through one stdlib-only facade (`serveapi.New`), the composability promise #498 made. |
| A11: Structured Failure | +1 | Every failure path returns a JSON-RPC error with a stable code (the table in §Wire contract). This replaces SDK-generated plain-text bodies. |
| A12: System Boundary | +1 | The MCP wire contract becomes an explicit, versioned boundary owned by this repo: a supported-version list and a golden corpus. |

**Net Score: +8** → **Proceed.** Hard-violation check: no −1 anywhere; A1/A3/A4/A7 are ≥ 0.

**Honest counterweight (not an axiom):** this re-implements a subset of a protocol that is still
changing. That is the "drift-prone" objection behind the Non-Goal of
`m-serveapi-protocol-only-module.md`. §Why the ratified Non-Goal no longer holds addresses it head-on.
It is not waved away.

---

## Problem Statement

`serveapi` has two embedded protocol handlers, with different build closures:

| handler | file | non-stdlib imports | closure |
|---|---|---|---|
| A2A | `serveapi/a2a_handler.go` (180 lines) | `serveapi/protocol` only | stdlib + protocol |
| MCP | `serveapi/mcp_handler.go` (187 lines) | `modelcontextprotocol/go-sdk/mcp` | + 31 non-stdlib packages (V5) |

Both are unexported and live in the **same package**. Importing `serveapi` for either one
therefore links the SDK's whole subtree (V5, V6). `serveapi/protocol` carries types, validators
and envelope writers, but **no method dispatch** (V2).

A zero-cloud consumer, ailang-world's daemon, which enforces `TestDaemonDependencyAllowlist`
(V11), has only two options today, and both are closed:

1. **Import `serveapi` (or the SDK).** This adds +34 packages across 5 module roots to World's gated
   closure, 28 of them disallowed, including an OAuth stack (the #885 report, re-confirmed by V5/V6 on
   our side). That breaches World charter clauses 2 (zero-cloud core) and 3 (no ambient authority).
2. **Write its own MCP JSON-RPC dispatch.** That reinvents a protocol codec, which World's design
   freeze (`DESIGN.md` §3.7) and our own #498/#764 intent forbid.

**Consequence:** World already had to hand-write its A2A projection over `protocol` helpers
(V12: `host/projection/projection.go` mirrors "the SAME keys the upstream serveapi handler
emits"). The MCP half of clause 6 is blocked outright. World's `w-mcp-dispatch-projection.md` names
this issue as its blocking predicate.

**History that this doc must repair:** #885 was closed on 2026-09-13 as "Delivered", citing
`descriptor.go`/`envelope.go`/`interfaces.go`. Those files already existed at v0.33.2, the tag the
issue was filed against. `git diff v0.33.2 v0.42.0 -- serveapi/protocol` is empty. It was re-opened on
2026-09-25. **The acceptance criteria below are written so that a close can be checked
mechanically** (AC-W1..W3). A repeat of the 09-13 false close must be impossible without a failing
command.

---

## Goals

**Primary goal:** `serveapi` links no MCP SDK. `serveapi.New(cfg).MCPHandler()` serves stateless MCP
Streamable HTTP (`initialize`, `notifications/initialized`, `ping`, `tools/list`, `tools/call`) over
stdlib + ailang's own `serveapi/protocol/*` packages. The same dispatcher is importable on its own by a
zero-cloud consumer, at a cost of +2 ailang-owned packages under the prefix World already admits:
`protocol/mcphttp` and the protocol-neutral runner package `protocol/hostcall` (V24).

**Success metrics:**
- `go list -deps ./serveapi` contains **no** non-stdlib package other than `serveapi` and
  `serveapi/protocol`, across the linux/darwin/windows GOOS matrix (the V13 gate, tightened).
- A **differential test** runs the new dispatcher and the SDK's
  `mcp.NewStreamableHTTPHandler` side by side on a ≥ 30-request corpus. They agree on every row
  marked `parity` in §Wire contract, and every `intentional-diff` row is asserted explicitly.
- All existing `serveapi` MCP behaviour tests pass unchanged, except the one test that
  injects a fake SDK transport (§Conflict Surface C4), which is rewritten.
- World's gated closure moves **254 → 256** from its current base `58f6022` (V24), and the unmodified gate stays green. The SDK arm was 283 on the older 249 base (V21).

---

## Why the ratified Non-Goal no longer holds

`m-serveapi-protocol-only-module.md` §Non-Goals (quorum-ratified, 2026-08-23) contains two relevant
lines. This doc overturns **one** and keeps **the other**:

| Non-Goal | Disposition | Reason |
|---|---|---|
| "Reimplementing MCP transport machinery in stdlib-only code … a large, drift-prone project **with no downstream demand**" | **Overturned** | The stated premise, *no downstream demand*, is falsified: #885 carries measured demand from the same consumer #764 served, and it is filed on the `cross-mission` label. The *size* premise is also narrower than it looked. Our handler is **stateless** (`Stateless: true`, `mcp_handler.go:49`), so there is no session map, no GET/SSE resumption stream, and no server-initiated requests (V10 rows 11–12: GET and DELETE are 405 today). The subset is five methods plus framing. |
| "Shipping **any executable machinery** in `protocol`" | **Kept for package `protocol`** | Package `protocol` gains no `http.Handler` and **no change at all** (D-C resolved: grammar kept). Under the recommended D-A (a′), the dispatcher is a *separate* package, `serveapi/protocol/mcphttp`, that sits under the `protocol/` path so World's prefix admission covers it (V24). Whether that reading honours the Non-Goal's intent ("every protocol symbol is public API") is part of the D-A ruling. Under (a), nothing moves under that path at all |

The drift risk is real, and three mechanisms answer it (not "we'll be careful"):
1. **An explicit supported-version list** (D-B). An unknown `MCP-Protocol-Version` is refused with a
   JSON-RPC error, never silently served.
2. **A differential test against the SDK** kept in `_test.go` files. Test imports are outside the
   build closure the gate measures (V13's header comment says so explicitly). The SDK stays in
   `go.mod` anyway for `internal/apiserver` and `cmd/ailang-microrag-mcp` (V7). Every Dependabot SDK
   bump reruns the differential, so drift shows up as a red test and never as silent divergence.
3. **Golden wire fixtures** for every row of §Wire contract.

---

## Verification Log

| # | Claim | Command | Result |
|---|---|---|---|
| V1 | Worktree base | `git rev-parse --short HEAD` | `bb0b0fe99` (= origin/dev) |
| V2 | **Negative:** `protocol` has no MCP method dispatch | `grep -c "tools/call\|tools/list\|initialize" serveapi/protocol/*.go` | `0` in all 5 files. Control: World's E2 measured the same grep class at 2 hits on `a2a_handler.go`'s own method strings |
| V3 | `protocol` is unchanged since #885 was filed | `git diff v0.33.2 v0.42.0 -- serveapi/protocol` (attended session, 2026-09-25, comment on #885) | empty |
| V4 | Only `mcp_handler.go` in `serveapi` imports the SDK | `grep -rln modelcontextprotocol/go-sdk --include='*.go' serveapi \| grep -v _test` | `serveapi/mcp_handler.go` only |
| V5 | `serveapi` closure today | `go list -deps ./serveapi \| grep -c .`; non-stdlib roots via `grep '\.' \| sed 's#/.*##' \| sort \| uniq -c` | **224** total; **31** non-stdlib (26 `github.com`, 5 `golang.org`). `./serveapi/protocol`: **188** |
| V6 | OAuth stack is inside that closure | `go list -deps ./serveapi \| grep -e oauth2 -e go-sdk/auth -e oauthex` | `golang.org/x/oauth2`, `golang.org/x/oauth2/internal`, `go-sdk/oauthex`, `go-sdk/auth` |
| V7 | SDK stays a module dependency regardless | `grep -rln modelcontextprotocol/go-sdk --include='*.go' . \| grep -v _test` | `cmd/ailang-microrag-mcp/main.go`, `internal/apiserver/feedback_tool.go`, `internal/apiserver/mcp.go`, `serveapi/mcp_handler.go`. Removing it from `serveapi` does **not** drop the `go.mod` require line |
| V8 | **Negative:** nothing in-repo imports the `serveapi` facade except its own test | `grep -rn '"github.com/sunholo-data/ailang/serveapi"' internal cmd` | no hits. Only `serveapi/serveapi_external_test.go` imports it |
| V9 | **Scoped negative:** no facade consumer **among the repositories searched** | `grep -rl --include='*.go' 'sunholo-data/ailang/serveapi'` over every Go repo under `~/dev/sunholo-data/` on this machine, 2026-09-28 | hits only in ailang clones (docs/fleet/motoko mission clones) and ailang-world, which imports `serveapi/protocol` only (`host/daemon/daemon.go:51`, `host/projection/projection.go:42`). **This does not prove ecosystem-wide non-use** (quorum r2, `gpt6-astra`). `serveapi` is a public package of a public module, so the wire changes are treated as public compatibility changes and need a human sign-off (D-F), independent of this search |
| V10 | Current SDK-backed wire behaviour (the reference) | scratch test driving `embeddedHandler(...)` with 13 requests (deleted after the run) | see table below |
| V11 | World's gate exists | `grep -n "func TestDaemonDependencyAllowlist" -r host` (World) | `host/daemon/daemon_test.go:884`. World pins `github.com/sunholo-data/ailang v0.33.2` (`go.mod:6`) |
| V12 | World hand-wrote its A2A projection over `protocol` | `sed -n 20,45p host/projection/projection.go` (World) | comment: "SAME keys the upstream serveapi handler emits" |
| V13 | Our closure gate allows 11 module roots for `serveapi` today | `sed -n 170,183p scripts/check_protocol_closure.sh` | R8 case arm allows `sunholo-data/ailang`, `google/jsonschema-go`, `modelcontextprotocol/go-sdk`, `segmentio/asm`, `segmentio/encoding`, `yosida95/uritemplate/v3`, `golang.org/x/{oauth2,sync,sys,time}`. The header says it measures build closure, not tests. Self-test: `scripts/test_check_protocol_closure.sh` |
| V14 | Post-change `serveapi` adds **zero** stdlib packages beyond protocol's closure | for the 12 stdlib imports the new files need (`bytes context encoding/json errors fmt io log net/http reflect strings sync time`), `go list -deps` of each ∖ `go list -deps ./serveapi/protocol` | **0** extra. So World's delta is exactly +2 (`serveapi/protocol`, `serveapi`) |
| V15 | SDK tool-name grammar (reference for D-C) | `sed -n 160,195p $(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk)/mcp/tool.go` | 1–128 bytes, `[A-Za-z0-9_.-]`. `/` is invalid. SDK v1.8.0 |
| V16 | `protocol`'s current tool-name grammar | `grep -n mcpToolNameRegex serveapi/protocol/descriptor.go` | `^[a-zA-Z0-9_-]{1,64}$`, **stricter** than the SDK: rejects `.` and lengths 65–128 |
| V17 | SDK version and supported protocol versions | `go.mod:25`; `sed -n 45,62p …/mcp/shared.go` | `go-sdk v1.8.0`. Supported: `2026-07-28, 2025-11-25, 2025-06-18, 2025-03-26, 2024-11-05`; latest `2026-07-28` |
| V19 | **Spec:** the missing-`MCP-Protocol-Version` default | WebFetch `modelcontextprotocol.io/specification/2025-06-18/basic/transports` §Protocol Version Header, 2026-09-28 | verbatim: *"if the server does not receive an `MCP-Protocol-Version` header, and has no other way to identify the version … the server **SHOULD** assume protocol version `2025-03-26`."* and *"If the server receives a request with an invalid or unsupported `MCP-Protocol-Version`, it **MUST** respond with `400 Bad Request`."* The same page: a POST carrying a notification **MUST** get 202 with no body; the client **MUST** send an `Accept` listing both `application/json` and `text/event-stream`; a GET may be answered 405 |
| V20 | **SDK:** absent header → *no* default is applied | `sed -n 340,365p …/go-sdk@v1.8.0/mcp/streamable.go` | comment: "If absent, the version is unknown". The empty string is passed through, **not** replaced by `2025-03-26`. So on absent-header requests the SDK is not spec-SHOULD-conformant, and its replies carry 2026-era fields (V10 rows 3, 16) |
| V21 | World closure baselines (249 pristine / 250 +protocol / 283 +SDK, 28 violations) | **inherited, not re-derived**: #885 issue body (World-measured, v0.33.2, go1.26.6, pristine-first with controls) and World `w-mcp-dispatch-projection.md` rows E3/E4 | used only as the before-numbers in AC-W2. AC-W2 re-measures them at delivery rather than trusting these |
| V22 | **Negative:** no Origin validation in the embedded MCP path today | `grep -n -i origin serveapi/*.go \| grep -v _test` → empty; the SDK's check runs only when `StreamableHTTPOptions.CrossOriginProtection != nil` (`streamable.go:328`), and `mcp_handler.go:49` sets only `Stateless` | parity: none today, none after. The spec's Origin MUST stays the embedding host's job (its `SessionResolver` sees the `*http.Request`) |
| V23 | OpenAI-style function-name constraint that the current grammar matches | the current `protocol` regex `^[a-zA-Z0-9_-]{1,64}$` (V16); quorum reviewer `gemini-3-1-pro` round 1 | taken as the reason to **keep** the grammar (D-C). Not independently re-fetched from a vendor page; the decision does not depend on it, because World needs a name mapping for `/` in any case (V15) |
| V24 | World's gate **today**, at `58f6022` | `go list -deps ./host/daemon/... ./cmd/ailang-worldd/... \| sort -u \| grep -c .`; `go test ./host/daemon -run TestDaemonDependencyAllowlist -count=1`; `sed -n 770,776p;950,990p host/daemon/daemon_test.go` | closure **254**; gate **green** (`ok`). `serveapi/protocol` is already admitted by **package-path prefix** (line 774). `TestAilangProtocolAdmissionIsNarrow` (line 962) asserts that the `serveapi` **facade is REFUSED** (mutation `MUT-FACADE-IMPORT`), and its control leg asserts that `serveapi/protocol/subpkg` **is admitted** ("a future subpackage rides the same prefix"). This supersedes V21's 249 baseline, which predates World's P6.D landing |
| V25 | `protocol`'s non-stdlib closure is itself alone | `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./serveapi/protocol` | exactly `github.com/sunholo-data/ailang/serveapi/protocol`. Refutes quorum r2's claim that it contains third-party roots; V13's 11 roots are the **facade** arm, not protocol's |
| V26 | Where the callback runner is used (for the move to `hostcall`) | `grep -rln "callbackRunner\\|runCallback\\|newCallbackRunner" serveapi --include='*.go'` | `callbacks.go`, `callbacks_test.go`, `a2a_handler.go`, `mcp_handler.go`, `serveapi.go`, `embedded_a2a_test.go`, `embedded_mcp_test.go`, `serveapi_external_test.go`. Every one is in the Files list or the AC3 fixtures |
| V27 | Current changelog file | `ls changelogs/ \| grep current` | `v0.32-current.md` |
| V28 | SDK body-size limit that W2 ports | `grep -n "DefaultMaxRequestBodyBytes\s*=" …/go-sdk@v1.8.0/mcp/streamable.go` | `:224 const DefaultMaxRequestBodyBytes = 4 << 20 // 4 MiB`. `mcphttp` defines its own 4 MiB constant with the same value |
| V29 | **Spec 2025-03-26** permits batches over Streamable HTTP | WebFetch `modelcontextprotocol.io/specification/2025-03-26/basic/transports` §Sending Messages, 2026-09-28 | verbatim: *"The body of the POST request **MUST** be one of: a single JSON-RPC request, notification, or response · an array batching one or more requests and/or notifications · an array batching one or more responses"*; *"If the input consists solely of … responses or notifications … **MUST** return HTTP status code 202"*; *"one JSON-RPC response per each JSON-RPC request … These responses **MAY** be batched."* Contrast V19: 2025-06-18 allows only a single message |
| V30 | `outputSchema` is optional on the wire | `sed -n 1925,1940p …/go-sdk@v1.8.0/mcp/protocol.go` | `// OutputSchema holds an optional JSON Schema …` with `json:"outputSchema,omitempty"`. So the SDK's own type marks it optional; the `null` in V10 comes from the embedded handler passing a nil `json.RawMessage` through `any` (a typed-nil, so `omitempty` does not fire). D-E removes that artefact |
| V18 | Existing MCP tests go through HTTP, not SDK internals, except one | `grep -n "transport:" serveapi/*_test.go` | only `TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt` (`embedded_mcp_replay_test.go`) constructs `embeddedMCPHandler{transport: …}` directly |

### V10 — the SDK-backed handler on the wire today

All POSTs send `Content-Type: application/json`. `Accept` is `application/json, text/event-stream`
unless stated otherwise. Rows 0–12 are from the first probe run; rows 13–17 are from a second probe run
(same method, also deleted) added in the round-1 revision.

| row | request | status | Content-Type | body |
|---|---|---|---|---|
| 0 | `initialize` (client asks `2025-06-18`), no version header | 200 | `text/event-stream` | `event: message` / `data: {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-06-18","serverInfo":{…}}}` |
| 1 | `notifications/initialized` | 202 | `application/json` | empty |
| 2 | `tools/list`, header `2025-06-18` | 200 | SSE | `result: {"ttlMs":0,"cacheScope":"public","tools":[{…,"outputSchema":null}]}` |
| 3 | `tools/list`, no header | 200 | SSE | same as row 2 |
| 4 | `tools/call` known tool | 200 | SSE | `result: {"content":[{"type":"text","text":"{\"ok\":true}"}],"structuredContent":{"ok":true}}` |
| 5 | `tools/call` unknown tool | 200 | SSE | `error: {"code":-32602,"message":"unknown tool \"nope\""}` |
| 6 | `ping` | 200 | SSE | `result: {}` |
| 7 | unknown method | **400** | `text/plain` | `JSON RPC not handled: "bogus/method" unsupported` |
| 8 | `Accept: application/json` only | **400** | `text/plain` | `Accept must contain both 'application/json' and 'text/event-stream'` |
| 9 | header `2026-07-28`, no `_meta` | **400** | `application/json` | `error: {"code":-32602,"message":"missing or invalid _meta field \"io.modelcontextprotocol/protocolVersion\""}` |
| 10 | header `1999-01-01` | **400** | `text/plain` | `Bad Request: Unsupported protocol version (supported versions: …)` |
| 11 | GET | 405 | `text/plain` | `Method Not Allowed` (and `Allow: POST`, per `TestEmbeddedMCPStatelessTransport`) |
| 12 | DELETE | 405 | `text/plain` | `Method Not Allowed` |
| 13 | `initialize` (client asks `2025-03-26`), no header | 200 | SSE | `protocolVersion: "2025-03-26"` echoed |
| 14 | `initialize` (client asks `1999-01-01`) | 200 | SSE | `protocolVersion: "2025-11-25"`: the SDK answers with its latest *header-negotiated* version, not `2026-07-28` |
| 15 | body `{not json` | **400** | `text/plain` | `malformed payload: unmarshaling jsonrpc message: …` |
| 16 | JSON array (batch) of one `tools/list` | **200** | SSE | **served**: the batch is accepted and answered as a single result. Batching was removed from the spec in 2025-06-18. The #603 battery's case named "batch rejected" asserts only labelling, so it never noticed |
| 17 | `"jsonrpc":"1.0"` | **400** | `text/plain` | `malformed payload: invalid message version tag "1.0"; expected "2.0"` |

Observations the design has to handle: `outputSchema: null` is emitted when a tool declares none.
The spec makes the field optional, so `null` is an SDK artefact. `ttlMs`/`cacheScope` are
2026-07-28-era fields that the SDK emits even when the negotiated version is 2025-06-18. The
host-callback failure paths bypass the SDK's reply entirely and produce `WriteMCPEnvelope`'s
`-32603` JSON (frozen by `TestEmbeddedMCPFrozenCallbackEnvelopes`).

---

## High-Impact Decisions

| ID | Decision | Why high impact | Chosen by | Deadline | Change cost |
|----|----------|-----------------|-----------|----------|-------------|
| D-A | **Where the dispatcher lives:** (a) replace the SDK inside the `serveapi` facade (one implementation; the facade becomes stdlib-only) · **(a′) put the stdlib dispatcher in a new exported subpackage `serveapi/protocol/mcphttp`, and have the facade's `MCPHandler()` delegate to it** (still one implementation; the facade is also SDK-free) · (b) a new sibling package `serveapi/mcpcore`, while the facade keeps the SDK · (c) document the SDK boundary as permanent (#885 option 3, WONTFIX) | It sets the public surface, whether two MCP implementations coexist, and **whether World must edit its own gate**. V24: World refuses the facade by test and admits `serveapi/protocol/*` by prefix. So (a) forces World to amend `TestAilangProtocolAdmissionIsNarrow`, a World charter event; (a′) is admitted by World's **unmodified** gate. (b) creates the one-concept-two-implementations seam. (c) leaves clause 6 half-done | **human** | design | high |
| D-B | **Supported protocol versions** for the stdlib dispatcher: (i) `2025-03-26, 2025-06-18, 2025-11-25` (header-negotiated, the stateless-POST era) and refuse `2026-07-28` with a JSON-RPC error · (ii) also implement `2026-07-28` per-request `_meta` negotiation | (ii) is the drift-prone part: new result fields and a per-request `_meta` version. (i) is a behaviour **narrowing** versus today's SDK path (V10 row 9), but V8/V9 show no consumer depends on it | **human** | design | med |
| D-C | **Tool-name grammar in `protocol`** — **RESOLVED in round-1 revision: keep `^[a-zA-Z0-9_-]{1,64}$` unchanged** | Widening was the original proposal. Reviewer `gemini-3-1-pro` objected (V23): the current grammar is the one LLM function-calling APIs accept, and MCP clients forward tool names to those APIs, so a dotted or 65–128-byte name that validates here can fail one hop later. That failure is silent from our side. Keeping the grammar costs nothing, because World needs an ID→name mapping for `/` regardless (`world/recovery-transition/v1` is invalid even under the SDK's wider rule, V15). The mapping simply also covers `.` | agent (resolved) | design | low |
| D-D | Error-shape changes where the SDK returned `text/plain` 400s (V10 rows 7, 8, 10) → JSON-RPC errors (`-32601`, `-32600`) | Visible to any client that matched on the plain-text body. None known (V8/V9) | agent | compile | low |
| D-F | **Accept the public wire-compatibility changes** (W3–W7, W15, D-E) on `serveapi.MCPHandler` | V9's search is local, so external reliance cannot be ruled out (quorum r2, `gpt6-astra`). The change must be accepted on its merits, with release notes, rather than on an absence-of-consumers claim | **human** | design | low |
| D-E | Omit `outputSchema` when absent (instead of `null`); omit `ttlMs`/`cacheScope` under versions that predate them | Wire bytes differ from the SDK. Covered as `intentional-diff` rows in the differential | agent | compile | low |

**Recommendation (revised after V24):** D-A = **(a′)**. It keeps one implementation, and the consumer
it exists for can import it with **zero edits** to a gate World designed around this exact prefix.
The cost is a new exported surface (`mcphttp.NewHandler(Config) http.Handler`, where `Config` holds
the same `protocol` interfaces plus timeout and concurrency) and executable code *under* the `protocol/`
path. That is a narrowed reading of the kept Non-Goal: package `protocol` itself still ships no
executable code. The ruling on that reading is part of D-A. Option (a) remains the choice if Mark
prefers zero new exported symbols and accepts that World amends its own narrowness test. D-B = **(i)**, with 2026-07-28 as a follow-up gated on actual consumer
demand. D-C is resolved (keep the grammar).

### Design Freeze

The sprint-executor pauses on any unchecked item:

- [x] **D-A** — **RULED (a′)** by Mark, 2026-09-28 (attended): `serveapi/protocol/mcphttp`, with the facade delegating to it
- [x] **D-B** — **RULED (i)** by Mark, 2026-09-28: 2025-03-26 / 2025-06-18 / 2025-11-25; refuse 2026-07-28
- [x] **D-F** — **RULED accept** by Mark, 2026-09-28 — accept W3–W7, W15 and D-E as **public** wire-compatibility changes to `serveapi.MCPHandler` (V9 cannot prove non-use). If accepted, the changelog carries a "Changed wire behaviour" list and a migration note ("clients sending batches or `2026-07-28` must …")
- [x] **D-C** — resolved in round-1 revision: `ValidateMCPName` grammar unchanged (see the D-C row)

---

## Solution Design

### Overview

Replace `serveapi/mcp_handler.go`'s SDK transport with a stdlib dispatcher of the same
shape as `a2a_handler.go`, in `serveapi/protocol/mcphttp`. The facade's `MCPHandler()` becomes a thin
constructor call into it, so exactly one MCP implementation exists. The request pipeline that exists today stays **identical up to the point
where it hands the body to the SDK**: read the body with a limit → `RequestID` → resolve session
(runner) → load tools (runner) → `protocol.CallerSurface` → dispatch. Only the last step changes.
The public API (`serveapi.New`, `Config`, `Server.MCPHandler`, `Server.Mount`) is untouched.

### Wire contract (what the dispatcher MUST do)

| # | Input | Output | vs SDK (V10) |
|---|---|---|---|
| W1 | non-POST | 405, `Allow: POST`, `text/plain` `Method Not Allowed` | parity (rows 11–12) |
| W2 | body > limit | `WriteMCPEnvelope` `-32603` "invalid MCP request body" | parity (existing code path, kept; the limit constant moves in-repo from `mcp.DefaultMaxRequestBodyBytes`) |
| W3 | `Accept` lacks `application/json` **or** `text/event-stream` | 400, JSON-RPC `-32600`, `Content-Type: application/json`, nosniff (D-D) | intentional-diff (row 8) |
| W4 | `MCP-Protocol-Version` present and not in the D-B set | 400, JSON-RPC `-32600` naming the supported list | intentional-diff (rows 9, 10) |
| W5 | absent `MCP-Protocol-Version` | treated as `2025-03-26`, the spec SHOULD (V19) | **intentional-diff**: the SDK applies no default (V20) and its row-3 reply carries `ttlMs`/`cacheScope`; ours omits them under D-E. The differential asserts our shape explicitly |
| W6 | invalid JSON · `jsonrpc` ≠ `"2.0"` · JSON array (batch) | invalid JSON / bad version tag: 400, `application/json`, JSON-RPC `-32700` / `-32600`, `id: null`. **Batch:** accepted **only** when the effective version is `2025-03-26` (header absent, per W5, or equal to `2025-03-26`), which is the one supported version whose transport permits batches (V29). A batch of notifications gets 202. A batch containing requests gets **one** SSE event whose `data:` is a JSON array holding one response per request, which the 2025-03-26 spec allows ("These *responses* **MAY** be batched"). Under `2025-06-18`/`2025-11-25` a batch gets `-32600` (V19: "a single JSON-RPC request, notification, or response") | intentional-diff on invalid JSON and version tag: the SDK answers 400 `text/plain` (rows 15, 17). Batch: parity in *acceptance* under 2025-03-26 (row 16, which ran header-absent); the SDK's single-object reply to a one-element batch becomes a one-element array |
| W7 | `initialize` | SSE single event. `protocolVersion` = the client's version if it is in the D-B set, otherwise our latest supported version (parity with row 14, where the SDK answers `2025-11-25`). `capabilities: {"tools":{"listChanged":false}}`; `serverInfo` from `Config.Agent` | intentional-diff: `listChanged:false` (stateless, so we never notify) and no `logging` capability (we never send logs) |
| W8 | any notification (no `id`), including `notifications/initialized` | 202, empty body | parity (row 1) |
| W9 | `ping` | SSE `result: {}` | parity (row 6) |
| W10 | `tools/list` | SSE `result: {"tools":[{name, description, inputSchema, outputSchema?}]}` in `AuthorizedSurface.All()` order (sorted by name) | parity except D-E (no `null` outputSchema, no `ttlMs/cacheScope`) |
| W11 | `tools/call` authorized name | runner-bounded `Invoke`; SSE `result: {"content":[{"type":"text","text":<raw>}],"structuredContent":<raw>}` | parity (row 4) |
| W12 | `tools/call` unknown/unauthorized name | SSE JSON-RPC `-32602` `unknown tool "<name>"` | parity (row 5) |
| W13 | `tools/call` host failure (timeout/capacity/cancel/error) | `WriteMCPEnvelope` `-32603` with `CallbackMessage(err)`, `application/json` | parity (frozen test) |
| W14 | resolve returns `AuthorizationError` 401/403 | that HTTP status, plain body | parity (frozen test) |
| W15 | unknown method with an `id` | SSE JSON-RPC `-32601` | intentional-diff (row 7) |
| W16 | every response | `Content-Type` set, `X-Content-Type-Options: nosniff` | parity (#603 guard, now asserted directly instead of via replay) |

SSE framing is exactly `event: message\n` + `data: <one-line JSON>\n\n`, the framing World's
obligations list names (`w-mcp-dispatch-projection.md`, "SSE-framing conformance").

### Files to Modify/Create

- `serveapi/protocol/hostcall/runner.go` — new: the bounded callback runner, **moved** from `serveapi/callbacks.go`
  (V26), exported as `hostcall.Runner` plus a generic `hostcall.Run`. It lives in a **protocol-neutral** package,
  not in `mcphttp`, so the A2A handler never depends on an MCP-named package (quorum r3, `gemini-3-1-pro`).
  One runner exists: the facade builds one and passes it to both handlers.
- `serveapi/callbacks.go` — delete. Its body moves to `hostcall`. `serveapi/callbacks_test.go` moves with it as
  `serveapi/protocol/hostcall/runner_test.go`, with its three tests unchanged apart from the package name (AC3).
- `serveapi/a2a_handler.go`, `serveapi/serveapi.go` — switch `callbackRunner`/`runCallback` calls to `hostcall`.
  No behaviour change; the A2A regression fixtures are in AC3.
- `serveapi/mcp_handler.go` — rewrite. Remove the SDK import, `bufferedResponseWriter`,
  `serverForRequest`, the per-request `mcp.Server`, and the `recover()` around SDK tool registration.
  Keep the pre-dispatch pipeline and `embeddedMCPConfig`. ~180 lines → ~120.
- `serveapi/protocol/mcphttp/handler.go` — new, exported `mcphttp.NewHandler(mcphttp.Config) (http.Handler, error)`. `Config` carries the `protocol` interfaces, agent info, and a **required** `*hostcall.Runner` (nil is an error, never a silent unbounded default). JSON-RPC envelope decode, method table
  (`initialize`/`ping`/`tools/list`/`tools/call`/notifications), version negotiation (D-B), SSE writer.
  ~200 lines.
- `serveapi/mcp_parity_test.go` — new. The differential. Imports `go-sdk/mcp` **in test only**, builds
  the reference handler with the same descriptors, and runs the corpus (≥ 30 requests covering W1–W16
  plus the hostile inputs from the #603 battery). It asserts `parity` rows equal after JSON
  normalisation and asserts each `intentional-diff` row's new shape explicitly. ~250 lines.
- `serveapi/testdata/mcp_wire/` — new. One golden per W-row.
- `serveapi/embedded_mcp_replay_test.go` — rewrite `TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt`
  (C4). Keep the battery test unchanged.
- `scripts/check_protocol_closure.sh` — replace the `serveapi` arm's **module-root** allowlist (R8, V13)
  with an **exact package-path** allowlist:
  the facade arm must print exactly `serveapi`, `serveapi/protocol`, `serveapi/protocol/hostcall`, `serveapi/protocol/mcphttp`; a new `mcphttp` arm must print exactly
  `mcphttp`, `hostcall` and `protocol`; a new `hostcall` arm must print exactly `hostcall` and `protocol`;
  the existing `protocol` arm stays "exactly itself" (V25). A module-root rule would admit any other ailang package, including stdlib-only ones (quorum
  round 1, `gpt6-astra`). The existing anti-vacuity floors (R1–R12) are kept.
- `scripts/test_check_protocol_closure.sh` — two new refusal cases, matching MUT-SDK-BACK and
  MUT-INTERNAL-BACK (AC1).
- `changelogs/v0.32-current.md` — entries for `serveapi` going SDK-free and for the intentional wire diffs (W3–W7, W15, D-E).

### Implementation Plan

- **M1 — runner move, dispatcher, goldens (1.5d).** `hostcall` (moved runner), `mcphttp/handler.go`, and the rewritten `mcp_handler.go`. Every
  existing test in `embedded_mcp_test.go` and `embedded_a2a_test.go`, and the #603 battery, stays green **unmodified**. Goldens for W1–W16.
- **M2 — differential against the SDK (1d).** `mcp_parity_test.go`. Before relying on it, mutation-test
  it: flip one parity row in the dispatcher and confirm the differential reds.
- **M3 — closure gate (0.5–1d).** Switch `check_protocol_closure.sh` to the four exact package-path arms, run both named mutations across the GOOS matrix, and paste their red output.
- **M4 — consumer proof + close (0.5d).** Run World's own gate against a `replace` pointing at the
  branch, in a scratch World worktree that is never committed to World (World edits are World's own
  reviewable event). Post the measured numbers on #885. Close only when AC-W1..W3 pass.

---

## Conflict Surface

No parser/typechecker change, but this **overrides shared machinery** (the MCP transport behind a
public facade), so the section is filled in.

| # | Position extended / replaced | What else lives there | Decision |
|---|---|---|---|
| C1 | `serveapi.Server.MCPHandler()` bytes on the wire | the same method's current SDK-generated bytes (V10) | **override**, bounded by the W-table; `intentional-diff` rows enumerated |
| C2 | `protocol.ValidateMCPName` | the A2A handler's `skills[].id` and `CallerSurface`'s duplicate detection (both reach it via `CallerSurface`) | **reuse, unchanged** (D-C resolved). The new dispatcher calls `CallerSurface` exactly as the SDK path does |
| C3 | `check_protocol_closure.sh` `serveapi` arm | its self-test and the `ci` target step | tighten; self-test updated in the same commit |
| C4 | `embeddedMCPHandler.transport` field | `TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt` injects a fake transport into it (V18) | the field disappears. The test's **intent** (an unlabelled body must never reach the client) is re-homed as an assertion on the SSE/JSON writers, with its own negative control |
| C5 | `internal/apiserver`'s MCP server (`internal/apiserver/mcp.go`, SDK) | `ailang prompt`, the public MCP endpoint, microrag | **untouched**. It stays on the SDK. No `internal/apiserver` → `serveapi` import exists (V8) |

**Programs that must still work (regression fixtures, all exist, bodies read at authoring time):**
`serveapi/embedded_mcp_test.go` — `TestEmbeddedMCPExactRequestLocalSurfaces`,
`TestEmbeddedMCPDispatchAuthorizationAndSessionIdentity`, `TestEmbeddedMCPStatelessTransport`
(asserts GET→405 + `Allow: POST`, POST→SSE with `event: message`),
`TestEmbeddedMCPPanicSafetyAndDescriptorValidation` (asserts the gateway's loud validation message
and **not** "host tool registration failed"), `TestEmbeddedMCPFrozenCallbackEnvelopes` (asserts 200,
`application/json`, `-32603`, echoed id, per stage), `TestEmbeddedMCPOverloadEnvelopeAndFastControl`.
`serveapi/embedded_mcp_replay_test.go` — `TestEmbeddedMCPReplayIsNeverSniffable`,
`TestWriteMCPEnvelopeIsLabelled`. `serveapi/embedded_a2a_test.go` (touched by the runner move) —
`TestEmbeddedA2ACardExactRequestLocalSurfaces`, `TestEmbeddedA2ADispatchAuthorizationAndSessionIdentity`,
`TestEmbeddedA2AFrozenCallbackEnvelopes`, `TestEmbeddedA2AInterleavedRequestLocalSurfaces`,
`TestEmbeddedA2ABlockedPrincipalDoesNotBlockAnother`, `TestEmbeddedA2AEmptyAndInvalidResultsAreDistinguishable`.
`serveapi/callbacks_test.go` (moves to `hostcall`) — `TestCallbackRunnerTimeoutAndStopsChain`,
`TestCallbackRunnerObservedDeadlineAndFastCall`, `TestCallbackRunnerBoundsConcurrencyAndRecovers`.
`serveapi/serveapi_external_test.go` (the facade's public contract).

**Deliberate incompatibilities:** W3, W4, W5, W6, W7, W15, D-E. All are visible only to clients of
`serveapi.MCPHandler`, which has no consumer outside this repo's tests (V8, V9).

---

## Success Criteria

- [ ] **AC1** — under `GOOS ∈ {linux, darwin, windows}`, `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}'`
  prints **exactly** these sets (with every path under `github.com/sunholo-data/ailang/`):
  `./serveapi` → `serveapi`, `serveapi/protocol`, `serveapi/protocol/hostcall`, `serveapi/protocol/mcphttp`;
  `./serveapi/protocol/mcphttp` → `protocol/mcphttp`, `protocol/hostcall`, `protocol`;
  `./serveapi/protocol/hostcall` → `protocol/hostcall`, `protocol`;
  `./serveapi/protocol` → `protocol` (already true, V25). Enforced by the tightened
  `make check-protocol-closure`. Two named mutations, each run across the GOOS matrix with the red output
  pasted: **MUT-SDK-BACK**, re-add `import _ "github.com/modelcontextprotocol/go-sdk/mcp"` to
  `serveapi/protocol/mcphttp/handler.go`, and the gate must fail naming the SDK package on the `mcphttp` and
  facade arms; **MUT-INTERNAL-BACK**, create a stdlib-only scratch package `serveapi/protocol/closureprobe` and
  import it from `mcphttp`, and the gate must fail naming exactly that path. MUT-INTERNAL-BACK is the one a
  module-root or prefix rule would pass.
- [ ] **AC2** — `grep -rn modelcontextprotocol serveapi --include='*.go' | grep -v _test.go` is empty.
- [ ] **AC3** — every test listed under "Programs that must still work" passes, with **no edits** except C4 and
  the package-name change on the moved runner tests. That list includes the A2A and runner fixtures (V26).
- [ ] **AC4** — the differential (`mcp_parity_test.go`) passes on ≥ 30 requests. **Named mutation
  MUT-PARITY:** change the W12 error code to `-32601` in the dispatcher; the differential must red on
  that row.
- [ ] **AC5** — goldens exist for W1–W16 and each is asserted. **MUT-SSE-FRAME:** drop the trailing blank
  line of the SSE frame; the W9/W10/W11 goldens must red.
- [ ] **AC6** — D-B enforced: an unsupported `MCP-Protocol-Version` (`1999-01-01` and `2026-07-28`) yields JSON-RPC `-32600` naming the supported list, and never a served result.
- [ ] **AC7** — package `protocol` itself is byte-identical:
  `git diff --exit-code <base> -- ':(glob)serveapi/protocol/*.go'` exits 0 (a single-`*` glob does not
  recurse). **And** every new path under `protocol/` is in one of the two ruled subpackages:
  `git diff --name-only <base> -- serveapi/protocol | grep -v -e '^serveapi/protocol/mcphttp/' -e '^serveapi/protocol/hostcall/'`
  prints nothing. World's naming need is answered in the #885 closing comment (AC-W3), not in code.
- [ ] **AC8** — every response path carries `Content-Type` + `nosniff` (W16). The C4 rewrite includes a
  negative control that fails if a writer ever emits an unlabelled body.
- [ ] `make ci-quick`, `make test`, `go test -race ./serveapi/...` green; changelog updated.

**Close criteria for #885 (so the 09-13 false close cannot recur):**
- [ ] **AC-W1** — a tag exists whose `git diff v0.47.1 <tag> -- serveapi/mcp_handler.go` removes the
  go-sdk import. Paste that command's output in the closing comment.
- [ ] **AC-W2** — base pinned: a scratch World worktree at **`58f6022`** (V24: closure 254, gate green,
  protocol admitted by package-path prefix), with `replace github.com/sunholo-data/ailang => <tag>` and the
  dispatcher imported from `host/daemon`. Expected (D-A ruled (a′)):
  `go list -deps` over both gated patterns gains exactly **two** non-stdlib packages, `…/serveapi/protocol/mcphttp`
  and `…/serveapi/protocol/hostcall` (254 → 256), and the **unmodified** `TestDaemonDependencyAllowlist` **and**
  `TestAilangProtocolAdmissionIsNarrow` both stay green. Paste the before and after `go list` counts and the
  test output in the closing comment.
- [ ] **AC-W3** — the closing comment names the tag, links this doc, and states the D-B version list and
  the unchanged name grammar (so World sizes its ID→name mapping for both `/` and `.`), so World can write `w-mcp-dispatch-projection` against the delivered seam.

---

## Testing Strategy

- **Golden** — one fixture per W-row, byte-compared.
- **Differential** — the SDK reference versus the stdlib dispatcher (test-only SDK import). It is the
  drift alarm on every SDK bump.
- **Mutation** — MUT-SDK-BACK, MUT-PARITY, MUT-SSE-FRAME. The executor runs each once and pastes its red
  output into the sprint log. A test that survives its mutation is not a test.
- **Hostile inputs** — reuse the #603 battery corpus (hostile Host, Last-Event-ID,
  MCP-Protocol-Version, Mcp-Session-Id, batch, empty, HTML body) as differential rows.
- **Concurrency** — the existing overload/fast-control test covers the runner. The dispatcher adds no
  shared state, so `go test -race ./serveapi` must be clean.

---

## Deferred Decisions (agent latitude)

- Internal structure of `mcphttp/handler.go` (method map versus switch), and message wording on
  intentional-diff rows (codes are fixed by the W-table; wording is not).
- Differential normalisation (key order, SSE whitespace), as long as it is documented in the test header.

## Non-Goals

- `internal/apiserver`'s MCP server and `cmd/ailang-microrag-mcp`. Both stay on the SDK (C5).
- Stateful MCP: sessions, GET/SSE streams, resumption, server→client requests, `listChanged`
  notifications, resources, prompts, sampling, and auth flows (OAuth/CIMD/DCR).
- `2026-07-28` support (D-B ruled (i)). It becomes a follow-up gated on a consumer asking.
- Editing ailang-world. AC-W2 runs in a scratch worktree only. World's allowlist edit, and its
  `world/…` → MCP-name mapping, are World's own reviewable events.
- Executable code or any other change in **package** `serveapi/protocol` itself (the kept Non-Goal, AC7). Executable code under the `protocol/` *path* is confined to `protocol/mcphttp` (ruled by Mark, D-A (a′), 2026-09-28) and `protocol/hostcall` (**added in the round-3 revision, after the ruling**. Mark then instructed "sprint plan and execute now" (2026-09-28), so the sprint proceeds on it, and it is flagged to him explicitly in the delivery report for veto. Folding the runner back into `mcphttp` is the cheap reversal if vetoed, at the cost of A2A importing an MCP-named package).
- Origin validation (V22: absent today; it remains the embedding host's policy).

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Spec drift makes our subset diverge from real clients | D-B's explicit version list refuses what we have not implemented, and the differential reds on every SDK bump |
| A real MCP client relies on `logging` / `listChanged:true` | Stateless server: we never emit either, so advertising them was already inaccurate. M2 also drives the embedded handler with the SDK's own *client* (`mcp.NewClient` over streamable HTTP) as an interop check |
| The rewrite drops a #603 labelling guarantee | W16 is asserted on every golden; the C4 rewrite keeps a negative control |
| The issue is again closed without delivery | AC-W1..W3 are commands with pasted output, not prose |
| The embedded endpoint is reachable cross-origin (spec: servers MUST validate `Origin`) | Unchanged from today (V22): no Origin check exists on this path now. It stays the host's policy, via its `SessionResolver`. The `serveapi.Server.Mount` doc comment gains one sentence saying so, since the SDK's opt-in `CrossOriginProtection` hook goes away |

## Related Documents

- [m-serveapi-protocol-only-module.md](../m-serveapi-protocol-only-module.md) — #764. Its MCP Non-Goal is overturned here (§Why the ratified Non-Goal no longer holds); its protocol-executable Non-Goal is kept.
- [ailang-core-triage/serveapi-protocol-mcp-dispatch.md](../ailang-core-triage/serveapi-protocol-mcp-dispatch.md) — the triage row that routed #885 here.
- [v0_31_0/m-mcp-2026-07-28-adoption.md](../v0_31_0/m-mcp-2026-07-28-adoption.md) — SDK adoption of the stateless 2026-07-28 spec for `internal/apiserver`. Relevant to D-B.
- [implemented/v0_30_0/m-serveapi-raw-handler-mcp.md](../../implemented/v0_30_0/m-serveapi-raw-handler-mcp.md), [implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md](../../implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md) — the embedded MCP surface this rewrites.
- ailang-world `design_docs/planned/w-mcp-dispatch-projection.md` — the blocked consumer. Its "What upstream must deliver" section is the requirement this doc answers, and its SSE-framing obligation is W9–W11.
- `design_docs/v1-mission.md:1011` — the `[NEXT]` row for this issue.

## Future Work

- `2026-07-28` per-request `_meta` negotiation (if D-B = (i)).
- World may retire its hand-written A2A projection (V12) by importing `serveapi` and using
  `A2AHandler()`, since the same import now carries no SDK.

## Quorum record

**Round 1 — 2026-09-28, BLOCKED 3/3** (seats `gpt6-astra`, `gemini-3-1-pro`, `oc-glm-5-3`; controller
pass; artifact `.ailang/state/mission-quorum/m-serveapi-sdk-free-mcp-dispatch-2026-09-28T13-30-39Z.json`;
$0.19). Every objection was accepted:

| reviewer | objection | disposition |
|---|---|---|
| `gpt6-astra` | Shrinking the allowlist to the `sunholo-data/ailang` **module root** would admit any other ailang package, so AC1's invariant is not enforced | **Accepted.** Exact package-path allowlist (Files, AC1) plus MUT-INTERNAL-BACK, the mutation a root rule passes |
| `gemini-3-1-pro` | Widening the shared `ValidateMCPName` lets names through that LLM function-calling APIs reject | **Accepted.** D-C resolved to *keep* the grammar. `protocol` is now untouched (AC7); World's mapping covers `.` as well as `/` |
| `oc-glm-5-3` | W5 labelled `parity` on an unverified default; plus: axiom sum is +8, not +9; W6 had no probe row; World baselines had no V-row | **Accepted in substance, corrected on one fact.** The reviewer's proposed default (`2025-06-18`) is wrong. The spec says `2025-03-26` (V19, verbatim). But W5 was mislabelled: the SDK applies **no** default (V20), so W5 is now `intentional-diff`. Net corrected to +8. V10 rows 13–17 probe W6/W7 (and found the SDK *serves* batches). V21 records the World baselines as inherited |

**Round 2 — 2026-09-28, BLOCKED 3/3** (same seats; controller pass). The re-quorum-once guardrail is now
spent, so the doc goes to Mark with its gaps labelled rather than into round 3.

| reviewer | objection | disposition |
|---|---|---|
| `gpt6-astra` | V8/V9 are local searches and cannot prove ecosystem non-use, yet they justify public wire changes | **Accepted.** V9 is re-scoped as a scoped negative. The compatibility changes become an explicit human freeze item, **D-F**, with a release-note requirement |
| `gemini-3-1-pro` | AC1's two-package allowlist is impossible because `protocol`'s closure contains third-party roots such as `jsonschema-go` | **Refuted by measurement (V25):** `protocol`'s non-stdlib closure is exactly itself. The 11 roots in V13 belong to the facade arm. AC1 now cites V25 |
| `oc-glm-5-3` | AC-W2's 249 → 251 / "2 intruders" is inconsistent: World already imports `protocol` | **Accepted, and it changed the design.** V24 measured World at `58f6022`: closure 254, gate green, `protocol` admitted by prefix, facade **refused by test**, `protocol/subpkg` **admitted by test**. AC-W2 is re-pinned to that base, and the D-A recommendation moves from (a) to **(a′)** so World's unmodified gate admits the dispatcher |

**Round 3:** runs against the ruled options (D-A (a′), D-B (i), D-F accept).

**Round 3 — 2026-09-28, BLOCKED 3/3** (against the ruled options). All three objections were residue from
the pre-ruling option (a), plus one coupling defect. All were accepted: AC7 now uses a non-recursive glob
plus a subpackage whitelist; the closure gate has four exact per-package arms; stale file names are fixed;
A2A and runner fixtures were added to AC3 (V26); the changelog path is logged (V27); and the runner
moves to the protocol-neutral `protocol/hostcall` instead of `mcphttp` (`gemini-3-1-pro`: A2A must not
depend on an MCP-named package). AC-W2 is now 254 → 256.

**Round 4 — 2026-09-28, BLOCKED 3/3.** Closed without a round 5, per Mark's instruction to proceed.
- `gpt6-astra`: batch rejection contradicts supporting 2025-03-26. **Accepted:** W6 is now version-dependent (V29).
- `gemini-3-1-pro`: batching removal, `outputSchema` optionality and the body-limit constant were unlogged. **Accepted:** V19 (single message in 2025-06-18), V30, V28.
- `oc-glm-5-3`: `hostcall` was attributed to Mark's ruling but introduced after it. **Accepted:** the attribution was corrected (Non-Goals) and the item is flagged for veto in the delivery report.

The rounds converged: round 1 raised design objections, round 2 a measurement gap that changed the
recommendation, and rounds 3–4 consistency and logging. No reviewer disputed the architecture after round 2.


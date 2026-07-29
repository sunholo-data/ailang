# M-MCP-2026-07-28: Adopt the stateless MCP spec (go-sdk v1.6.1 → v1.7.0) without breaking our own client

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1 (Medium-High) — nothing is broken today, but a routine `go get -u` of the MCP SDK silently breaks live prompt delivery in prod. This is a landmine, not a fire.
**Estimated**: ~1.5 days (Phase 1 ~0.5d, Phase 2 ~0.5d, Phase 3 ~0.5d)
**Dependencies**: None. Phase 2 depends on Phase 1 shipping first (see Design Freeze).
**Milestone ID**: M-MCP-2026-07-28
**Source**: MCP spec release [2026-07-28](https://blog.modelcontextprotocol.io/posts/2026-07-28/), surfaced by Mark 2026-07-29.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A tool call becomes one self-contained POST. Today's 3-POST session sequence has cross-request ordering state; removing it removes a nondeterminism source. |
| A2: Replayability | +1 | A captured `tools/call` request is replayable verbatim under the new spec; today it is meaningless without its live `Mcp-Session-Id`. |
| A3: Effect Legibility | 0 | No change to AILANG's effect surface. Wire-protocol only. |
| A4: Explicit Authority | 0 | The public MCP endpoint is unauthenticated and read-only today; CIMD/DCR changes are Non-Goals (see below). No ambient authority added. |
| A5: Bounded Verification | +1 | The negotiated protocol version becomes an asserted constant with a handler-level test, rather than an implicit runtime outcome nobody checks. |
| A6: Safe Concurrency | 0 | Server is already `Stateless: true`; no shared session map either way. |
| A7: Machines First | +1 | 3 sequential round-trips → 1, inside `ailang prompt`'s hard 1500 ms budget. Directly reduces the silent-fallback-to-embedded rate for agents. |
| A8: Minimal Syntax | 0 | No language syntax change. |
| A9: Cost Visibility | 0 | No cost surface change. |
| A10: Composability | +1 | MCP/A2A wire composability with `aitana/platform` is a standing constraint; staying current on the transport keeps that path open without a new protocol. |
| A11: Structured Failure | +1 | Fixes a real misdiagnosis: an HTTP 500 from the server is currently reported as `"server did not return Mcp-Session-Id"` (status is checked *after* the header). |
| A12: System Boundary | +1 | The wire version becomes an explicit, tested boundary constant instead of a hardcoded string with no coverage. |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — this removes some.
- [x] A3 (Effects): No hidden side effects.
- [x] A4 (Authority): No ambient access granted; auth surface explicitly out of scope.
- [x] A7 (Machines First): Optimises machine round-trip cost, not human convenience.

---

## Problem Statement

MCP shipped spec revision **`2026-07-28`** on 2026-07-28. It is the largest break in the protocol's history:

- **Stateless core** — the `initialize`/`initialized` handshake and `Mcp-Session-Id` are gone. Each request carries its own protocol version and client info.
- **Header routing** — `Mcp-Method` / `Mcp-Name` for gateway routing and authorization.
- **Multi Round-Trip Requests (MRTR, SEP-2322)** — replaces server-initiated requests that needed an open bidirectional stream.
- **Optional discovery** — `server/discover` (SEP-2575), not required.
- **Cacheable list results** — `ttlMs` / `cacheScope` on resource/tool/prompt lists.
- **Deprecations** (12-month window) — Roots, Sampling, Logging, legacy HTTP+SSE transport, DCR (superseded by CIMD).

### Current State

| Surface | Location | Protocol today |
|---|---|---|
| Public MCP server (`mcp.ailang.sunholo.com`) | [internal/apiserver/mcp.go](../../../internal/apiserver/mcp.go) | go-sdk v1.6.1 → ceiling **2025-11-25** |
| microrag stdio server | [cmd/ailang-microrag-mcp/main.go](../../../cmd/ailang-microrag-mcp/main.go) | same SDK |
| Hand-rolled CLI client (`ailang prompt`, `ailang mcp status`) | [internal/mcp_client/client.go:49](../../../internal/mcp_client/client.go) | hardcoded **`2024-11-05`** |

go-sdk **v1.7.0** is released and supports `2026-07-28`. We pin v1.6.1 (go.mod:24), whose ceiling is `2025-11-25`. So: **we do not support the new spec.**

### The actual problem — the upgrade is booby-trapped

Both servers run `StreamableHTTPOptions{Stateless: true}` ([mcp.go:299](../../../internal/apiserver/mcp.go)). Under v1.6.1, stateless mode *still* emits `Mcp-Session-Id`. Under v1.7.0 it does not: `serveStateless` only populates `SessionID` when the `MCPGODEBUG=allowsessionsinstateless=1` compatibility flag is set, and the response header is guarded by `if c.sessionID != "" && isInitialize`.

Our own client treats that optional header as mandatory:

```go
sessionID := resp.Header.Get("Mcp-Session-Id")
if sessionID == "" {
    return "", errors.New("server did not return Mcp-Session-Id")
}
```
— [internal/mcp_client/client.go:187](../../../internal/mcp_client/client.go)

**Consequence of a naive SDK bump + redeploy:**
1. `ailang prompt --source=mcp` stops fetching live content and silently degrades to the embedded fallback ([internal/prompt/fresh.go:143](../../../internal/prompt/fresh.go)).
2. `ailang mcp status` reports `reachable: false` — the health check itself lies about a healthy server.
3. **Every `ailang` binary already in the wild breaks**, not just the one we rebuild. This is a server-side change that breaks old clients.
4. The release pipeline does **not** catch it: gates 1–2 in [cloudbuild-release.yaml:199-206](../../../cloudbuild-release.yaml) hit the plain REST shim `$MCP/api/mcp/ailang_versions`, never the `/mcp/` JSON-RPC endpoint. The MCP wire protocol has **zero** automated coverage.

### Impact

- **Who**: every `ailang` CLI user (live prompt freshness), every third-party MCP client of the public endpoint, the eval rig probe in the ops runbook.
- **How significant**: silent, not loud. This is the same failure class as the motoko system-prompt delivery regression — the feature keeps "working" on a stale fallback while the live path is dead. That is precisely the failure mode Critical Principle #2 (no silent fallbacks) exists to prevent.

---

## Goals

**Primary Goal:** Reach `2026-07-28` on both MCP servers without breaking any deployed `ailang` CLI, and remove the session-ID coupling that makes the upgrade dangerous in the first place.

**Success Metrics:**
1. `mcp_client` completes a tool call against a v1.7.0 stateless server that emits **no** `Mcp-Session-Id` — verified by a handler-level test, not a live probe.
2. `ailang mcp status --json` reports `reachable: true` against both a v1.6.1-style and a v1.7.0-style server.
3. Negotiated protocol version is `2026-07-28` for the new client path, asserted in a test.
4. `ailang prompt --source=mcp` completes in **1 HTTP round-trip** (down from 3), measured within the 1500 ms budget.
5. Release pipeline gains a gate that exercises `/mcp/` JSON-RPC — a regression of this class fails CI rather than prod.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1.** Ship the client fix *before* the SDK bump (two releases, not one) | The reverse order breaks prod. Ordering is the whole design. | human | design | high |
| **D2.** Run the deployed server with `MCPGODEBUG=allowsessionsinstateless=1` for a deprecation window | Without it, every pre-fix `ailang` binary in the wild loses live prompt fetch the moment we redeploy. With it, they keep working. | human | design | high |
| **D3.** Length of that window, and what ends it | Determines when we can drop legacy compat. Proposal: until the v0.31.x line is the oldest CLI seen in `ailang mcp status` telemetry, minimum 3 months. | human | runtime | med |
| **D4.** Client sends `2026-07-28` single-POST, or keeps `initialize` and just tolerates a missing session | Single-POST is the point (3 RTT → 1), but only works against an upgraded server. | agent | compile | med |
| **D5.** Adopt `ttlMs`/`cacheScope` on `tools/list` now or later | Real win for the public endpoint, but orthogonal to the break. Proposal: **later** (Future Work). | agent | design | low |
| **D6.** Leave `CrossOriginProtection` at the SDK default | The default is `nil` = off in **both** v1.6.1 and v1.7.0 (v1.6.1 has the identical `enableoriginverification` gate at streamable.go:205). Only the *deprecation notice* is new — this is not a behaviour change and needs no action. | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — confirm the two-release ordering (client fix in v0.31.0, SDK bump in v0.31.1 or later). Sprint-executor must **PAUSE** if asked to do both in one release.
- [ ] **D2** — confirm `MCPGODEBUG` compat is acceptable for the public endpoint, or accept that old CLIs break on redeploy.

---

## Conflict Surface

**Not required** — this design touches none of `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`. It is confined to `internal/mcp_client/`, `internal/apiserver/mcp.go`, `cmd/ailang-microrag-mcp/`, and build/deploy config. No AILANG syntax or semantics change.

**Architecture boundaries**: `internal/mcp_client` is consumed by `internal/prompt` and `cmd/ailang`. `internal/apiserver` is a dashboard/apps-layer package. No new import edges are introduced in either direction; `make check-boundaries` should be unaffected. Run it anyway.

---

## Solution Design

### Overview

Three phases, strictly ordered. Phase 1 is a standalone bug fix that is valuable even if we never upgrade the SDK.

### Architecture

**Component 1 — `mcp_client` decoupled from sessions.**
Make `Mcp-Session-Id` optional on both read and write: capture it if present, forward it only if non-empty, never require it. Check HTTP status *before* inspecting headers so a 5xx reports as a 5xx. Skip `notifications/initialized` when there is no session to initialize.

**Component 2 — SDK bump to v1.7.0.**
Server-side, `2026-07-28` support is entirely inside the SDK. All 21 `mcp.*` symbols we use exist unchanged in v1.7.0 (see Verification Log V7). Our `Stateless: true` choice already matches the spec's sessionless direction (SEP-2567), so the servers need no structural change — the handler learns the new protocol for free.

**Component 3 — protocol regression coverage.**
An `httptest`-backed test that stands up `MCPServer.HTTPHandler()` and drives it with the real `mcp_client`, asserting a successful `tools/call` when the server emits no session header. Plus a release-pipeline gate that POSTs JSON-RPC to `/mcp/` rather than the REST shim.

### Implementation Plan

**Phase 1: Decouple the client from sessions** (~4 hours) — *ships alone, no SDK change*
- [ ] `initialize()`: check `resp.StatusCode >= 300` **before** reading the session header; return the HTTP status in the error.
- [ ] `initialize()`: return `("", nil)` when the header is absent instead of erroring.
- [ ] `CallTool()`: skip `sendInitialized` when `sessionID == ""`.
- [ ] `do()`: already guards on non-empty sessionID ([client.go:285](../../../internal/mcp_client/client.go)) — confirm, no change expected.
- [ ] **NEW** `internal/mcp_client/client_test.go` — the package currently has no tests at all (V4).
- [ ] Verify `ailang mcp status --json` against the live v1.6.1 endpoint still reports `reachable: true` (no regression).

**Phase 2: SDK bump + new protocol** (~4 hours)
- [ ] `go get github.com/modelcontextprotocol/go-sdk@v1.7.0`; `go mod tidy`; `make ci`.
- [ ] Set `MCPGODEBUG=allowsessionsinstateless=1` in [docker/Dockerfile.mcp](../../../docker/Dockerfile.mcp) (D2). Note the SDK **panics at init** on a malformed `MCPGODEBUG` value — the string must be exact.
- [ ] Bump `mcp_client.ProtocolVersion` to `2026-07-28` and collapse to a single POST (D4).
- [ ] Update the stale `Stateless:` doc comment at [mcp.go:289-295](../../../internal/apiserver/mcp.go), which currently describes the v1.6.1 session semantics.
- [ ] Confirm `DefaultMaxRequestBodyBytes` (4 MiB, **new** in v1.7.0 — V6) is above our largest tool payload. Tool args are small; large uploads use the HTTP/`getUploadUrl` path, not MCP.

**Phase 3: Coverage + docs** (~4 hours)
- [ ] `internal/apiserver/mcp_protocol_test.go` — `httptest` + real `mcp_client`, both with and without a session header.
- [ ] Add a JSON-RPC `/mcp/` gate to [cloudbuild-release.yaml](../../../cloudbuild-release.yaml) alongside gates 1–2.
- [ ] Update the `2024-11-05` probe snippets: [rig_operations_runbook.md:296](../../../.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md) (and the `.agents/` mirror), [docs/docs/guides/microrag.md:109](../../../docs/docs/guides/microrag.md).
- [ ] CHANGELOG.md entry.

### Files to Modify/Create

**New files:**
- `internal/mcp_client/client_test.go` — session-optional, status-ordering, single-POST cases, ~200 LOC
- `internal/apiserver/mcp_protocol_test.go` — handler-level end-to-end over the real client, ~150 LOC

**Modified files:**
- `internal/mcp_client/client.go` — session optional, status-before-header, protocol constant, ~80 LOC changed
- `internal/apiserver/mcp.go` — doc comment only, ~10 LOC
- `docker/Dockerfile.mcp` — one `ENV MCPGODEBUG=...` line
- `cloudbuild-release.yaml` — one new gate, ~10 LOC
- `go.mod` / `go.sum` — SDK bump
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` + `.agents/` mirror, `docs/docs/guides/microrag.md` — probe snippets
- `CHANGELOG.md`

---

## Examples

### Example 1: A tool call against a v1.7.0 stateless server

**Before** (3 round-trips; fails outright if the server omits the header):
```
POST /mcp/  {"method":"initialize",...}          → 200, Mcp-Session-Id: GHXS4B...
POST /mcp/  {"method":"notifications/initialized"} → 202   [Mcp-Session-Id: GHXS4B...]
POST /mcp/  {"method":"tools/call",...}            → 200   [Mcp-Session-Id: GHXS4B...]

# against go-sdk v1.7.0 stateless:
Error: initialize: server did not return Mcp-Session-Id
```

**After** (1 round-trip; works against both old and new servers):
```
POST /mcp/  {"method":"tools/call",...}  → 200
```

### Example 2: The diagnostic that lies today

**Before** — server returns HTTP 503; the status check sits *after* the header read:
```
$ ailang mcp status
✗ unreachable: initialize: server did not return Mcp-Session-Id
```

**After**:
```
$ ailang mcp status
✗ unreachable: initialize: HTTP 503
```

---

## Success Criteria

- [ ] `client_test.go` passes with a stub server that emits **no** `Mcp-Session-Id` (acceptance: `TestCallToolWithoutSessionID`)
- [ ] `client_test.go` passes with a stub server that **does** emit one (acceptance: `TestCallToolWithSessionID` — back-compat)
- [ ] A non-2xx `initialize` response surfaces the HTTP status, not the session-header message (acceptance: `TestInitializeReportsHTTPStatus`)
- [ ] `mcp_protocol_test.go` drives the real `MCPServer.HTTPHandler()` end-to-end via `mcp_client` and gets a tool result
- [ ] Negotiated version is `2026-07-28` post-Phase-2 (asserted, not observed)
- [ ] `ailang prompt --source=mcp` makes exactly 1 POST (assert via request counter in the stub)
- [ ] Release pipeline gate POSTs JSON-RPC to `/mcp/` and passes
- [ ] `make ci` green; `make check-boundaries` green
- [ ] CHANGELOG.md updated
- [ ] Runbook + microrag probe snippets updated

---

## Testing Strategy

**Unit tests** (`internal/mcp_client/client_test.go` — new package coverage from zero):
- tool call with / without `Mcp-Session-Id`
- HTTP 4xx/5xx during initialize → error names the status
- `ErrVersionMismatch` path still fires on `served_for` mismatch
- `ErrToolError` path still parses `{error, detail}`
- request-count assertion (1 POST on the new path, 3 on legacy)

**Integration tests** (`internal/apiserver/mcp_protocol_test.go`):
- `httptest.NewServer(ms.HTTPHandler())` + real `mcp_client.CallTool` → success
- same with `MCPGODEBUG` unset, i.e. genuinely no session header

**Manual testing:**
- `ailang mcp status --json` against live prod (v1.6.1) before and after Phase 1 — must stay `reachable: true`
- after Phase 2 deploys to the `test` Cloud Run prefix: run the raw curl probe from the ops runbook and confirm a tool result with no session header
- confirm a **v0.30.0** binary (pre-fix) still works against the Phase-2 server with `MCPGODEBUG` set — this is the D2 mitigation actually being verified, not assumed

---

## Deferred Decisions

- **Exact single-POST request shape** (which of `Mcp-Method` / `Mcp-Name` we send, and whether we call `server/discover` at all) — agent may choose, guided by what go-sdk v1.7.0's own client emits. We are not required to send discovery.
- **Whether `mcp_client` should switch to the official go-sdk client** instead of staying hand-rolled — agent may propose, but *not* in this sprint; the hand-rolled client exists for a 1500 ms budget and minimal deps. If proposed, it needs its own doc.
- **Which log level records a fallback-to-embedded event** — agent may choose, but it must be visible somewhere; a silent fallback is what got us here.

## Non-Goals

- **MRTR / server→client requests** — all our MCP tools are read-only lookups. `Stateless: true` already forbids server-initiated requests and we have no use case.
- **CIMD / DCR / auth changes** — the public MCP endpoint is unauthenticated by design and rate-limited per-IP (M-MCP-EDGE-THROTTLE, v0.15.0). Out of scope.
- **`server/discover` on our server** — optional in the spec; SDK-provided if we ever want it.
- **`ttlMs` / `cacheScope`** — real win for the public endpoint, but orthogonal to this break (D5 → Future Work).
- **Migrating off deprecated Roots / Sampling / Logging** — we use none of them. The 12-month window is not our problem.
- **Anything in `internal/ai/` or `internal/executor/`** — those speak provider HTTP APIs, not MCP.

---

## Timeline

**Day 1** (~4h): Phase 1 — client decoupling + `client_test.go`. Ships independently in v0.31.0.
**Day 2** (~4h): Phase 2 — SDK bump, `MCPGODEBUG`, protocol constant, single-POST. Gated on Phase 1 being released (D1).
**Day 3** (~4h): Phase 3 — protocol regression test, CI gate, docs, CHANGELOG.

**Total: ~12 hours across 3 days**, with a release boundary between Day 1 and Day 2.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| SDK bump ships in the same release as the client fix → old CLIs break on redeploy | **High** | D1 makes the two-release ordering a Design Freeze item; sprint-executor PAUSEs if asked to combine them. |
| `MCPGODEBUG` typo → SDK **panics at init** → MCP server won't boot | High | The value is a literal in `Dockerfile.mcp`; Phase 3's `/mcp/` CI gate catches a dead server before prod promotion. |
| Third-party MCP clients pinned to `2024-11-05` break | Med | v1.7.0 still lists `2024-11-05` in `supportedProtocolVersions`; combined with D2's legacy session compat, old clients keep working. |
| 4 MiB `DefaultMaxRequestBodyBytes` (new in v1.7.0) rejects a large tool payload | Low | Tool args are small; uploads use the HTTP `getUploadUrl` path. Verify in Phase 2; the field is configurable. |
| We fix the client but the fallback stays silent, so the *next* breakage is invisible too | Med | Deferred-decision item requires the fallback event be logged somewhere; `ailang mcp status` is the existing surface for it. |
| Spec churn — another revision lands mid-sprint | Low | Nothing here is spec-version-specific except one constant; the design is "stop requiring what the spec made optional." |

---

## Verification Log

Every claim below was checked against code or a live endpoint on 2026-07-29, per the design-doc-creator hard gate. Negative-existence claims get their own row.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | We pin go-sdk v1.6.1 | `grep modelcontextprotocol go.mod` | **Confirmed** — go.mod:24 |
| V2 | v1.6.1's ceiling is `2025-11-25`; it has no `2026-07-28` | read `mcp/shared.go:33-51` in the module cache | **Confirmed** — `latestProtocolVersion = protocolVersion20251125`; `supportedProtocolVersions` = {2025-11-25, 2025-06-18, 2025-03-26, 2024-11-05} |
| V3 | v1.7.0 exists and supports `2026-07-28` | fetched `proxy.golang.org/.../@v/v1.7.0.zip`, read `mcp/shared.go` | **Confirmed** — `latestProtocolVersion = protocolVersion20260728`; `2024-11-05` still in the supported list |
| V4 | **`internal/mcp_client` has NO test file** | `ls internal/mcp_client/` | **Confirmed** — `client.go` only |
| V5 | **No test anywhere asserts on `Mcp-Session-Id`** | `grep -rn 'Mcp-Session-Id' --include='*_test.go' .` | **Confirmed** — zero hits |
| V6 | **`MaxRequestBodyBytes` and `nowrapinvalidparams` do not exist in v1.6.1** (both new in v1.7.0) | `grep` both symbols in v1.6.1 `mcp/*.go` | **Confirmed** — zero hits; v1.7.0 has `DefaultMaxRequestBodyBytes = 4 << 20` |
| V7 | All 21 `mcp.*` symbols we use exist in v1.7.0 with the same shapes | enumerated our usage via `grep -rho 'mcp\.[A-Z][A-Za-z]*'`, then located each in v1.7.0 | **Confirmed** — 19 direct decls; `CallToolRequest` / `ReadResourceRequest` are aliases (`requests.go:10,19`) consumed identically by `ToolHandler` / `ResourceHandler` |
| V8 | v1.7.0 stateless mode does **not** emit `Mcp-Session-Id` | read `mcp/streamable.go:415-422` and `:1660` | **Confirmed** — `SessionID` set only when `legacySessions && !info.usesNewProtocol`; header write guarded by `if c.sessionID != "" && isInitialize` |
| V9 | The live prod server emits it **today** | `curl -D - -X POST https://mcp.ailang.sunholo.com/mcp/ …` | **Confirmed** — `mcp-session-id: GHXS4BX62UIV7MYQS6UZEPCJ4O`, `"protocolVersion":"2024-11-05"`, `serverInfo: ailang-api 0.8.1` |
| V10 | Our client hard-errors on a missing session header | read [client.go:187-190](../../../internal/mcp_client/client.go) | **Confirmed** |
| V11 | Status is checked *after* the header (misdiagnoses 5xx) | read [client.go:187-196](../../../internal/mcp_client/client.go) | **Confirmed** — header read at :187, `StatusCode >= 300` at :194 |
| V12 | The failure degrades to embedded rather than failing loudly | read [internal/prompt/fresh.go:130-144](../../../internal/prompt/fresh.go) + [cmd/ailang/mcp_status.go:86-101](../../../cmd/ailang/mcp_status.go) | **Confirmed** — `mcp status` would report `reachable: false` |
| V13 | **The release pipeline does NOT exercise the `/mcp/` JSON-RPC endpoint** | read `cloudbuild-release.yaml:194-206` | **Confirmed** — gates 1–2 POST to `$MCP/api/mcp/ailang_versions`, the REST shim |
| V14 | **`internal/mcp_client` is the only hand-rolled MCP client in the repo** | `grep -rln 'Mcp-Session-Id\|jsonrpc.*2\.0' --include='*.go' internal/ cmd/` | **Confirmed** — other hits are `apiserver/{a2a,mcp}.go` (server side) and `protocol_test.go` (A2A/OpenAPI, not MCP wire) |
| V15 | **`CrossOriginProtection` default did NOT change** between v1.6.1 and v1.7.0 | read v1.6.1 `streamable.go:205` vs v1.7.0 | **Confirmed** — both gate on `enableoriginverification == "1"`; only the deprecation notice is new. **No action needed** (corrects an earlier assumption). |
| V16 | **There is no `go-sdk/v2` module** — v1.7.0 is the right target | `curl proxy.golang.org/…/go-sdk/v2/@v/list` | **Confirmed** — 404 "no matching versions" |
| V17 | `MCPGODEBUG` is a comma-separated env var read at package init, and **panics** on malformed input | read `internal/mcpgodebug/mcpgodebug.go:24-29` | **Confirmed** — `init()` calls `panic(err)` on parse failure |
| V18 | Both our servers pass `Stateless: true` | read [mcp.go:296-301](../../../internal/apiserver/mcp.go); microrag uses `StdioTransport` (not HTTP) | **Confirmed** — the HTTP server is stateless; microrag is stdio and unaffected by the HTTP transport changes |

No AILANG language claims are made in this document, so no `ailang check` transcripts are required.

---

## Related Documents

<!-- Auto-populated by Ollama neural search on "07 28 adoption"; top neural score 0.26 → no duplicate. -->

**Prior MCP work (all implemented — reviewed for overlap, none found):**
- [design_docs/implemented/v0_11_0/m-mcp-quality-and-route-headers.md](../../implemented/v0_11_0/m-mcp-quality-and-route-headers.md) — MCP tool quality + `@route` header access. Application layer; says nothing about wire version.
- [design_docs/implemented/v0_15_0/m-mcp-edge-throttle.md](../../implemented/v0_15_0/m-mcp-edge-throttle.md) — per-IP rate limiting on the public endpoint. The `feedbackRL` in `NewMCPServer` is from this doc.
- [design_docs/implemented/v0_23_0/m-mcp-unit-param-binding.md](../../implemented/v0_23_0/m-mcp-unit-param-binding.md) — omitted-param binding. Tool dispatch, not transport.

**Search results (informational, no overlap):**
- [design_docs/implemented/v0_31_0/m-eval-measurement-contract-sprint-plan.md](../../implemented/v0_31_0/m-eval-measurement-contract-sprint-plan.md) (0.26)
- [design_docs/planned/v0_29_0/m-eval-rig-reliability.md](../v0_29_0/m-eval-rig-reliability.md) (0.25)

## References

- [MCP 2026-07-28 release announcement](https://blog.modelcontextprotocol.io/posts/2026-07-28/)
- [SEP-2567 — sessionless servers](https://github.com/modelcontextprotocol/modelcontextprotocol/pull/2567)
- SEP-2322 (MRTR), SEP-2575 (`server/discover`), RFC 9207 (issuer validation)
- [Design Axioms](/docs/references/axioms)
- [API Server Rules](../../../.claude/rules/api-server.md) — `isExposed()` is the single filtering point; MCP tool-name generation
- [Architecture boundaries](../../../ARCHITECTURE.md#architecture-boundaries)

## Future Work

- **`ttlMs` / `cacheScope` on `tools/list`** (D5) — the public endpoint serves a stable tool list to every agent that connects; declaring a TTL is a straight bandwidth and latency win. Needs its own doc.
- **`server/discover`** — lets a client skip the list round-trip entirely. Pairs naturally with the above.
- **Retire the hand-rolled client** in favour of the official go-sdk client, if the 1500 ms budget and dependency weight ever stop mattering.
- **Track MCP spec releases** — this revision was found by hand. A cheap watcher on the MCP blog feed would have flagged it on 2026-07-28.

---

**Document created**: 2026-07-29
**Last updated**: 2026-07-29

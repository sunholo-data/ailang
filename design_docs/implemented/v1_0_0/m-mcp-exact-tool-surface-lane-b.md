# M-MCP Exact Tool Surface Lane B: Embeddable Session-Owned Protocol Surface

**Status**: Implemented
**Target**: v1.0.0
**Priority**: P0 (Ailang World blocker; issue #498 Lane B)
**Estimated**: 2-3 days
**Dependencies**: Lane A exact-surface CLI controls (implemented at `aa02f0d9f`); MCP Go SDK v1.7.0
**Planner-Lane**: opus-required
**Milestone ID**: M-MCP-EXACT-TOOL-SURFACE-LANE-B
**Created**: 2026-08-04
**Source**: `sunholo-data/ailang#498`, requested resolution supplied by the Ailang World mission

---

## Problem Statement

AILANG owns conformant MCP-HTTP and A2A protocol serving, while an embedding host such as Ailang
World owns principals, sessions, and capability policy. The public Go surface cannot presently
join those responsibilities: the protocol implementation is under `internal/apiserver`, while
the top-level `runtime` precedent contains code-generation conversion helpers rather than serving
callbacks [V1, V2, V3].

The serving constructor accepts process-wide scalar configuration. Its exposure decision accepts
only an `ExportInfo`, so discovery and dispatch cannot select a caller-owned descriptor set after
resolving an HTTP request [V4, V5]. `Start` owns `ListenAndServe`, while the mux builder is
unexported [V6]. The result is a blocker for a host whose two simultaneous sessions must see
different tools.

Lane A is already present at this HEAD: the CLI has a default-false `--no-feedback-tool` flag,
the server conditionally registers `submit_feedback`, and tests distinguish the default-present
surface from the suppressed surface [V10, V11]. Lane B must preserve that standalone path while
making every embedder-visible tool—including `submit_feedback`—caller supplied.

**Verified corrections to the upstream measurements:**

- **REFUTATION — file-size premise:** `internal/apiserver/server.go` is **764 lines**, not already
  beyond the 800-line gate; `make check-file-sizes` passes at this HEAD [V12]. The design still
  places new machinery in focused files so the existing file does not cross the gate.
- **REFUTATION — Lane A hash:** supplied hash `a81d66983` names an unrelated property-generator
  commit in this checkout. The Lane A squash is `aa02f0d9f` and its subject names PR #529 [V13].
  The feature itself is present, so this changes provenance, not scope.
- **REFUTATION — public-package count:** `runtime` is not literally the only non-`main` package
  outside `internal`; `go list` also reports `std`, `testutil`, and golden-test packages [V1].
  `runtime` remains the relevant precedent because it is a top-level package intended for generated
  Go code, but the broader exclusivity claim is false.
- **REFUTATION — Config field count:** the `Config` block contains **15**, not 16, fields [V4].
  All are still process-wide configuration, so the architectural gap is unchanged.
- The GitHub issue body could not be fetched because `gh issue view` reported
  `error connecting to api.github.com`; this document therefore treats the verbatim requested
  resolution in the mission prompt as the contract [V20].

**Impact:** Ailang World cannot embed AILANG's protocol layer without either copying internal
machinery or surrendering its session-specific authority boundary. This item is release-blocking
for that downstream mission.

## Goals

**Primary Goal:** Add one narrow, public Go package that mounts AILANG-owned MCP-HTTP and A2A
handlers on a caller-owned mux and projects exactly one host-supplied, request-session-specific
tool set into both protocols.

**Success Metrics:**

- An external-module compile test imports `github.com/sunholo-data/ailang/serveapi` and mounts both
  handlers without importing an `internal` package.
- A two-principal test returns sentinel tool `alpha_only` only to principal A and `beta_only` only
  to principal B through both MCP `tools/list` and the A2A agent card.
- Calls to `alpha_only` as B fail before `Invoke`; calls as A deliver the exact resolver-returned
  session token to `Invoke`.
- With an empty descriptor set, embedded MCP and A2A discovery return zero tools/skills and do not
  contain `submit_feedback`; adding an explicit `submit_feedback` descriptor makes it appear and
  dispatch through the host callback.
- Existing standalone default/suppression sentinel tests remain green, and protocol/SSE regression
  tests continue to exercise the upstream MCP SDK transport.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| New public package is top-level `serveapi` | Establishes the long-lived external import path without conflating serving with codegen helpers | human | design | high |
| `ToolSource` returns a canonical descriptor set after `ResolveSession` | Makes host authority explicit and supplies one source for both protocol projections | human | design | high |
| Embedded requests bypass export enumeration but pass through a generalized central exposure gate | Preserves the written single-filtering-point invariant while allowing non-AILANG tools | human | design | high |
| MCP embed path creates a fresh SDK server inside the existing stateless request factory | Keeps SDK conformance/SSE framing and makes `tools/list` request-scoped | agent | compile | med |
| Every host callback has a handler-enforced deadline | Arbitrary host code is a trust boundary and must not wedge a protocol handler | reviewer + agent | design | high |
| `submit_feedback` is absent from embedded surfaces unless supplied as a normal descriptor | Prevents ambient built-in authority; changing later is externally observable | human | design | high |
| Standalone `cmd/ailang` construction remains on `apiserver.New`/`Server.Start` | Makes backward compatibility structural rather than dependent on callback defaults | human | design | high |

### Design Freeze

- [x] Public import path is `github.com/sunholo-data/ailang/serveapi`; do not add the API to `runtime`.
- [x] The host supplies protocol-neutral `ToolDescriptor` values through `ToolSource.Tools`.
- [x] Every discovery and invocation HTTP request calls `ResolveSession` first.
- [x] Embedded descriptor authorization uses the shared exposure gateway; it does not add a second
  filter inside MCP or A2A consumers.
- [x] Embedded built-ins are opt-in; standalone CLI defaults are unchanged.
- [x] AILANG owns wire codecs and transports; the host owns session values, descriptor selection,
  invocation behavior, and any persistence behind its callbacks.
- [x] Embedded MCP is **required** to construct `StreamableHTTPOptions{Stateless: true}`; per-request
  SDK servers are forbidden if this path is ever changed to stateful mode.
- [x] Every callback invocation is bounded by `Config.CallbackTimeout`; zero selects the safe default
  rather than disabling the timeout.
- [x] Callback CONCURRENCY is bounded by `Config.MaxConcurrentCallbacks`; a capacity token is held
  until the callback goroutine exits, and exhaustion yields a frozen overload envelope rather than a
  new goroutine. In-process Go callbacks cannot be forcibly terminated, so the guarantee claimed is
  bounded handler latency and bounded callback STARTS — never enforced completion.

## Deferred Decisions

- Internal helper and test-fixture names — agent may choose.
- Exact internal split between `embedded.go`, `embedded_mcp.go`, and `embedded_a2a.go` — agent may
  choose while every production file remains at or below 800 lines.
- Whether JSON Schema validation errors use an internal typed error or SDK error helper — agent may
  choose, provided protocol codes and `Invoke` non-entry are asserted.
- Logging fields for resolver/provider failures — agent may choose; session values must never be
  logged implicitly.

## Solution Design

### Overview

Create a top-level `serveapi` facade containing only stable host-facing types and handler/mount
methods. It delegates protocol work to new callback-driven adapters in `internal/apiserver`.
`serveapi` is an integration/protocol boundary above `internal/apiserver`; it is neither compiler
core nor dashboard state. It may legally import the module's `internal/apiserver`, as the existing
CLI does [V1]. The proposed dependency graph has no core-to-`serveapi` edge and gives the facade no
dashboard import.

Each handler executes this authority sequence:

```text
HTTP request
  -> ResolveSession(ctx, request)
  -> ToolSource.Tools(ctx, same session)
  -> central exposure/validation gateway
  -> MCP tools OR A2A skills generated from that one slice
  -> requested name must occur in that same authorized slice
  -> Invoker.Invoke(ctx, same session, invocation)
  -> existing protocol result/error encoder
```

MCP continues to use `mcp.NewStreamableHTTPHandler`, and the embedded path **must explicitly pass
`StreamableHTTPOptions{Stateless: true}`**. This is a correctness requirement, not an inherited
default. SDK v1.7.0 dispatches stateless and stateful handling separately; stateless mode rejects
GET/DELETE with 405, accepts only self-contained POST requests, creates a temporary session for
each request, and calls the supplied `getServer(req)` hook per POST [V22, V23, V24, V25]. The
embedded factory resolves the request, obtains descriptors, and returns a fresh SDK server whose
registered tools and closures belong only to that POST. There is therefore no long-lived GET
stream to correlate with a later POST on this path. The SDK still requires both `application/json`
and `text/event-stream` in `Accept`, preserving its per-POST SSE response framing [V26].

A2A's agent-card builder already receives `*http.Request`, but the current task dispatcher does
not carry the request into its send helper [V7, V15]. The adapter keeps request resolution at the
outer handler, projects skills from the same descriptor slice, verifies the requested skill name
against that slice, and invokes with the same session value.

### Public API

```go
// Package serveapi exposes AILANG-owned MCP-HTTP and A2A protocol handlers
// while leaving identity, session, capability, and execution policy to the host.
package serveapi

type Session any

type SessionResolver interface {
    ResolveSession(context.Context, *http.Request) (Session, error)
}

type ToolSource interface {
    Tools(context.Context, Session) ([]ToolDescriptor, error)
}

type Invoker interface {
    Invoke(context.Context, Session, Invocation) (InvocationResult, error)
}

type ToolDescriptor struct {
    Name         string
    Description  string
    InputSchema  json.RawMessage
    OutputSchema json.RawMessage
    Tags         []string
    Examples     []string
}

type Invocation struct {
    Name      string
    Arguments json.RawMessage
}

type InvocationResult struct {
    Value json.RawMessage
}

type Config struct {
    Resolver SessionResolver
    Tools    ToolSource
    Invoker  Invoker
    Agent    AgentInfo
    // CallbackTimeout bounds each Resolver, Tools, and Invoker call.
    // Zero uses DefaultCallbackTimeout; negative values are rejected by New.
    CallbackTimeout time.Duration
    // MaxConcurrentCallbacks caps how many host callback goroutines may be
    // in flight at once. Zero uses DefaultMaxConcurrentCallbacks; negative
    // values are rejected by New. A capacity token is acquired before a
    // callback is launched and held until that goroutine actually exits, so
    // callbacks that ignore context and never return cannot accumulate without
    // bound.
    MaxConcurrentCallbacks int
}

const DefaultCallbackTimeout = 5 * time.Second

const DefaultMaxConcurrentCallbacks = 64

type AgentInfo struct {
    Name        string
    Description string
    Version     string
}

func New(cfg Config) (*Server, error)
func (s *Server) MCPHandler() http.Handler
func (s *Server) A2AHandler() http.Handler
func (s *Server) Mount(mux *http.ServeMux)
```

`New` rejects nil callbacks, invalid static agent metadata, and a negative `CallbackTimeout`.
`CallbackTimeout == 0` means **use `DefaultCallbackTimeout`**, never "wait forever". A positive
value is used verbatim. Per-request callback failures remain
request failures: resolver failure maps to HTTP 401/403 through a documented typed error contract;
tool-source failure maps to protocol internal error; unknown or unauthorized names map to MCP
unknown-tool/A2A invalid-params without calling `Invoke`; invocation errors use the existing
protocol error envelopes. Exact exported error types may be finalized in implementation, but the
wire outcomes and non-invocation rule are frozen.

### Bounded Host Callbacks and Wire Errors

`ResolveSession`, `Tools`, and `Invoke` are separate timed operations. Immediately before each call,
the adapter derives a child context from `r.Context()` whose deadline is the earlier of the incoming
request deadline and `now + CallbackTimeout`; that already-deadlined context is the one passed to
the host callback. The adapter runs the callback behind a one-result buffered channel and selects
between its result and `ctx.Done()`. Consequently, even a callback that ignores context and blocks
forever cannot wedge the handler goroutine; its abandoned host goroutine may remain the host's
responsibility, but it cannot block response completion or send into an unbuffered channel.

**Bounded callback CONCURRENCY, not just bounded waiting** (quorum R2, `gpt5-6-sol`, narrow-refinement
carve-out — the reviewer's own proposed fix, applied verbatim). A deadline bounds how long the
*handler* waits; on its own it does **not** bound resource consumption, because each
context-ignoring callback is launched in a goroutine that may remain forever, so repeated requests
can create an unbounded number of leaked goroutines. Claiming "every callback invocation is bounded"
while accepting that accumulation is a contradiction, and it violates the bounded-wait and
safe-concurrency axioms. Therefore:

- **State the limit plainly: in-process Go callbacks cannot be forcibly terminated.** Nothing in
  this design can kill a host goroutine that refuses to return. What it can do is refuse to start
  unbounded numbers of them.
- `Config.MaxConcurrentCallbacks` (default `DefaultMaxConcurrentCallbacks`) caps in-flight callback
  goroutines. The adapter **acquires a capacity token before launching a callback and holds it until
  that callback goroutine actually exits** — not until the handler stops waiting for it. A goroutine
  abandoned after a timeout therefore continues to occupy capacity, which is precisely what makes
  the cap meaningful.
- If capacity cannot be acquired within the request/callback deadline, the adapter returns a
  **frozen protocol-specific overload error** without entering the callback: MCP and A2A JSON-RPC
  task POST return code `-32603` with message `"host callback capacity exceeded"` and HTTP **200**;
  A2A agent-card GET returns HTTP **503 Service Unavailable** with the same stable message.
- The guarantee is stated narrowly and honestly: the handler's own latency and the number of
  concurrently-started host callbacks are bounded. Completion of a non-cooperative callback is
  **not** guaranteed, and the document does not claim handler-enforced completion.

Deadline and context errors have frozen, protocol-visible mappings. **The A2A mappings below match
the wire format A2A already uses at HEAD, verified rather than assumed** (quorum R2,
`gemini-3-1-pro`; measurement in Verification Log rows V27/V28): `a2aError` writes
`w.WriteHeader(http.StatusOK)` with the comment "JSON-RPC always returns 200" and all existing task
errors are JSON-RPC codes (`-32600`, `-32601`, `-32602`), so mandating HTTP 200 + a JSON-RPC
envelope preserves the existing surface instead of corrupting it. `-32603` is **not** currently used
in `a2a.go` (V28), so it is a new code introduced by this design — the standard JSON-RPC
internal-error code — not a redefinition of an existing one:

- If the wrapper deadline fires, or the callback returns an error matching
  `context.DeadlineExceeded`, MCP returns a JSON-RPC error response with code `-32603`, message
  `"host callback timed out"`, the request id when decodable (otherwise `null`), HTTP **200**, and
  `Content-Type: application/json`. A2A JSON-RPC task POST returns the same code/message/id and HTTP
  **200**. A2A agent-card GET, which is not JSON-RPC, returns HTTP **504 Gateway Timeout** with a
  JSON error body carrying the same stable message.
- If a callback itself returns an error matching `context.Canceled` while the request context is
  still live, MCP and A2A task POST return code `-32603`, message `"host callback canceled"`, and
  HTTP **200**; A2A card GET returns HTTP **500** with that message. If the incoming request context
  itself is canceled, the handler returns promptly and no completed wire response is promised
  because the peer has gone away.
- Other resolver authorization errors retain the typed 401/403 contract; other `Tools`/`Invoke`
  errors use the existing protocol internal-error envelopes. Once any callback times out or fails,
  later callbacks are not entered.

The MCP timeout envelope is emitted before handing control to the SDK when resolution/surface
construction fails; successful requests still go through the SDK transport. The implementation
must preserve JSON-RPC ids and must not turn callback timeouts into HTTP 500/504 on MCP or A2A
JSON-RPC POSTs.

`Session` is deliberately opaque. The adapter never serializes, compares, retains, or mutates it;
it passes the interface value returned by `ResolveSession` directly to `Tools` and `Invoke` during
the request. Hosts needing persistence or stable identity implement that behind their callbacks.

### One Descriptor Set, Two Protocol Projections

Add one internal validated representation converted once from `[]serveapi.ToolDescriptor` (with
the public facade passing equivalent internal DTOs to avoid an import cycle). The gateway:

1. copies the caller slice and nested byte/slice fields so later host mutation cannot race serving;
2. validates non-empty unique MCP-compatible names and object-shaped input schemas;
3. sorts by name for deterministic discovery;
4. returns an immutable request-local `AuthorizedSurface` with `Lookup(name)` and `All()`.

Projection codecs consume only `AuthorizedSurface.All()`:

- MCP maps name, description, input schema, and output schema into SDK `mcp.Tool` values. Each SDK
  handler closes over the request-local authorized surface and session, verifies `Lookup(name)`,
  then calls the shared invocation adapter.
- A2A maps the same descriptor name to both skill `id` and `name`, description verbatim, and copies
  tags/examples. Agent-card defaults keep `application/json` input/output modes. Dispatch uses the
  same `Lookup(name)` rather than reconstructing module/function identifiers.

No MCP-only or A2A-only discovery callback exists. Therefore the two projections cannot obtain
different source sets within one request. Protocol-specific fields are derived by AILANG codecs,
not supplied as independent caller-owned documents.

### Reconciliation with `isExposed`

The written rule names `isExposed()` as the single filtering point for handler dispatch, OpenAPI,
A2A, MCP, and the startup banner [V5]. Lane B **generalizes, rather than bypasses or weakens, that
invariant**:

- Rename/refactor the concept into a small central exposure gateway with two inputs:
  `loadedExportSurface(serverPolicy, ExportInfo)` for the standalone module path and
  `callerSurface([]ToolDescriptor)` for the embedded path.
- Existing consumers continue to use the loaded-export branch, preserving `@noexpose`,
  `--routes-only`, and `@nomcp` semantics.
- Embedded MCP and A2A consumers receive only `AuthorizedSurface`; they may project and look up but
  may not add filtering conditions.
- Update `.claude/rules/api-server.md` in the implementation sprint to state the generalized
  invariant: all enumeration and dispatch must consume a central authorized surface; protocol
  consumers must not filter independently.

The caller-supplied path cannot call the old `isExposed(ExportInfo)` meaningfully because arbitrary
host tools need not correspond to loaded AILANG exports. Treating caller descriptors as synthetic
`ExportInfo` would falsely couple the public API to compiler metadata. The central gateway preserves
the security property—one decision point shared by discovery and dispatch—without pretending both
sources have the same domain model.

OpenAPI and ordinary REST handlers remain on the loaded-export surface. Lane B mounts MCP and A2A
only; it does not make caller descriptors into REST endpoints or OpenAPI operations.

### Invocation and Session Integrity

For every MCP POST, A2A card GET, and A2A task POST:

1. call `ResolveSession` before `Tools` or protocol parsing that could reveal discovery data;
2. retain the returned interface value in request context/private request state;
3. call `Tools` with that exact value;
4. build `AuthorizedSurface` and project or dispatch;
5. for invocation, pass the exact value to `Invoker.Invoke` with raw JSON arguments.

Tests use non-default pointer sentinels (`sessionA := &token{"A"}` and `sessionB := &token{"B"}`)
and assert pointer identity in both `Tools` and `Invoke`. This fails if the mechanism substitutes a
zero value, re-resolves at the wrong stage, shares process-global state, or invokes a descriptor
that was absent from that request's surface.

### `submit_feedback` and Backward Compatibility

The embedded constructor never calls `registerFeedbackTool` and has no `NoFeedbackTool` option.
An empty caller set means an empty embedded MCP tool list and A2A skill list. A host that wants
feedback supplies a descriptor named `submit_feedback` and handles it through `Invoker`, exactly
like any other tool.

The standalone CLI remains structurally separate: `serveAPICommand` continues to build
`apiserver.Config`, call `apiserver.New`, load modules, and call `Server.Start` [V14]. Its
`--no-feedback-tool` default remains false, so `NewMCPServer` continues registering feedback unless
the operator opts out [V10, V11]. The public facade is not inserted into that path.

### Caller-Owned Mux and Paths

`Mount` registers these exact patterns on the caller's mux:

```text
/mcp/                         -> http.StripPrefix("/mcp", s.MCPHandler())
/.well-known/agent.json       -> A2AHandler()
/a2a/                         -> A2AHandler()
```

Returning handlers as well as offering `Mount` lets a host select other prefixes with normal
`http.ServeMux` composition. `Mount` performs no listening and starts no watcher, Vite process, or
signal handler. Duplicate-pattern behavior remains the standard `ServeMux` behavior and is
documented rather than recovered.

### Architecture and Boundary Gate

`serveapi` is a public protocol-facade layer. It imports `internal/apiserver` and standard-library
types; `internal/apiserver` continues to reach compiler behavior through `internal/embed`. The
enforced gate enumerates specific core and dashboard package sets and does not classify
`internal/apiserver`; it currently passes [V8, V18]. The implementation must run
`make check-boundaries` and must not add imports from compiler-core packages into `serveapi` or
from core packages back into `serveapi`/dashboard packages.

### Implementation Plan and Milestones

#### Milestone 1: Public contract and central authorized surface (~0.75 day)

- [x] Add the top-level `serveapi` API and constructor validation.
- [x] Add internal callback DTOs/adapters without exposing internal types publicly.
- [x] Implement deterministic validation/copy/sort/lookup for caller descriptors.
- [x] Refactor the exposure invariant into a shared authorized-surface abstraction while preserving
  all loaded-export decisions.

**M1 Acceptance Criteria:**

- [x] A temporary external Go module with a local `replace` imports `serveapi`, constructs it with
  sentinels, and compiles; the same fixture's attempted import of `internal/apiserver` fails with
  `use of internal package ... not allowed`.
- [x] `Tools` receives pointer sentinel A for request A and pointer sentinel B for request B; a test
  fails on zero, stringified, copied, or swapped sessions.
- [x] Descriptor input `[zeta, alpha]` yields ordered surface `[alpha, zeta]`; duplicate `alpha` and
  a scalar input schema each produce explicit constructor/request errors.
- [x] Existing filtering tests for `@noexpose`, `--routes-only`, and `@nomcp` pass unchanged, proving
  the refactor did not make the central gate vacuous.
- [x] With `CallbackTimeout: 20*time.Millisecond`, each of three table cases installs a deliberately
  non-returning `ResolveSession`, `Tools`, or `Invoke` callback. Using an outer 250 ms test deadline,
  the request must complete after at least 20 ms but before 250 ms with the exact timeout envelope
  specified above, and the next callback/invocation counter must remain zero. The same table covers
  MCP POST, A2A task POST, and (for resolver/tools) A2A card GET. A control with a fast callback and
  `CallbackTimeout: 0` succeeds, proving zero selects the default. This test hangs/fails its outer
  deadline if handler-side timeout selection is absent, and fails early/wrong-envelope assertions
  if the configured duration or protocol mapping is not wired.
- [x] **Bounded-concurrency test** (quorum R2 `gpt5-6-sol`, the reviewer's own catch: "the current
  250 ms response tests prove only bounded handler latency, not bounded goroutine growth"). With
  `MaxConcurrentCallbacks: N` (small, e.g. 4), install a callback that **permanently blocks** and
  never returns. Send **many more than N** additional requests. Assert: (a) the count of callback
  STARTS remains capped at `N` — i.e. request `N+1` never enters the callback; (b) every excess
  request returns the exact overload envelope (`-32603`, `"host callback capacity exceeded"`,
  HTTP 200 on MCP/A2A POST; HTTP 503 on A2A card GET); (c) goroutine count measured after the burst
  does not grow with request count. This is discriminating in both directions: it FAILS if the
  semaphore is missing (starts grow with requests), and it FAILS if the token is released when the
  handler stops waiting rather than when the goroutine exits (capacity would recover and starts
  would exceed `N`). A control with fast callbacks and the same `N` must pass with zero overload
  envelopes, proving the cap is not simply rejecting everything.

#### Milestone 2: MCP request-scoped adapter (~0.75 day)

- [x] Create a request-local SDK server from `AuthorizedSurface` in the existing stateless
  `NewStreamableHTTPHandler` factory.
- [x] Reuse/refactor the existing MCP descriptor and result/error codecs.
- [x] Route authorized calls to `Invoker` with raw arguments and the resolver session.
- [x] Keep SDK-owned streamable HTTP/SSE transport.

**M2 Acceptance Criteria:**

- [x] MCP `tools/list` as A is exactly `[alpha_only, shared]`; as B it is exactly
  `[beta_only, shared]`. Neither response contains the other principal's sentinel.
- [x] Calling `alpha_only` as A invokes once with `sessionA` pointer identity and argument sentinel
  `{"nonce":"A-137"}`; calling it as B returns unknown-tool and leaves the invocation counter at
  zero.
- [x] Empty descriptors return zero MCP tools and exclude `submit_feedback`; an explicit
  `submit_feedback` descriptor makes exactly that tool appear and invokes the host callback.
- [x] Existing MCP protocol tests still assert initialization, `tools/list`, `tools/call`, content
  type, and SSE event framing through the SDK handler; no hand-written SSE encoder is introduced.
- [x] The embedded MCP handler is constructed with `Stateless: true`; an in-memory GET to `/mcp/`
  returns **405 Method Not Allowed** with `Allow: POST`, while a correctly headed POST reaches the
  request-local server and its sentinel tool. This fails if the handler is stateful or bypasses the
  SDK stateless dispatch.

#### Milestone 3: A2A projection, compatibility, and gates (~0.75 day + buffer)

- [x] Generate agent-card skills from `AuthorizedSurface` and route task sends by descriptor name.
- [x] Mount MCP and A2A routes on a caller-owned mux.
- [x] Update the single-filtering-point rule and public package documentation.
- [x] Run focused, repository-wide, boundary, formatting, and size gates.

**M3 Acceptance Criteria:**

- [x] For the same A/B descriptor callbacks used by M2, A2A cards expose exactly
  `[alpha_only, shared]` and `[beta_only, shared]`, with descriptions/tags/examples equal to the
  source descriptors; this catches a separate A2A enumeration path.
- [x] A2A send of `beta_only` as B invokes once with `sessionB`; the same send as A produces
  JSON-RPC invalid-params and leaves the invocation counter at zero.
- [x] `Mount` serves MCP initialization at `/mcp/`, a card at `/.well-known/agent.json`, and A2A
  JSON-RPC at `/a2a/` using `httptest.ResponseRecorder` without binding a socket.
- [x] Standalone sentinel tests prove default CLI/server MCP includes both `status` and
  `submit_feedback`, while `NoFeedbackTool:true` includes `status` and excludes feedback.
- [x] `go test ./...`, `make check-boundaries`, `make check-file-sizes`, and formatting/lint gates
  pass outside the loopback-denying sandbox; any sandbox bind error is reported as uninformative.

### Files to Modify/Create

**New files:**

- `serveapi/serveapi.go` (~180 LOC) — stable public types, constructor, handlers, and mount method.
- `serveapi/serveapi_external_test.go` (~120 LOC) — consumer-facing compile and mux tests.
- `internal/apiserver/authorized_surface.go` (~180 LOC) — copied, validated, sorted request surface
  and centralized lookup.
- `internal/apiserver/embedded_mcp.go` (~220 LOC) — callback-driven MCP SDK adapter.
- `internal/apiserver/embedded_a2a.go` (~220 LOC) — callback-driven A2A adapter.
- `internal/apiserver/embedded_test.go` (~350 LOC) — cross-protocol session, isolation, feedback,
  and dispatch sentinels (split if needed to stay under the size gate).

**Modified files:**

- `internal/apiserver/routes.go` (~+30/-15 LOC) — loaded-export branch of generalized exposure gate.
- `internal/apiserver/mcp.go` (~+20/-40 LOC) — extract/reuse MCP projection and result codecs; keep
  standalone construction semantics.
- `internal/apiserver/a2a.go` (~+20/-35 LOC) — extract/reuse A2A card/task codecs; keep standalone
  semantics.
- `.claude/rules/api-server.md` (~+8/-3 LOC) — restate the single authorized-surface invariant.
- `docs/docs/guides/serve-api.md` (~+70 LOC) — public embedding example and ownership boundary.

No new implementation belongs in `internal/apiserver/server.go`; at 764 lines it has only 36 lines
of headroom before the enforced limit [V12].

## Examples

### Mounting a session-specific surface

```go
api, err := serveapi.New(serveapi.Config{
    Resolver: worldResolver,
    Tools:    worldCapabilities,
    Invoker:  worldInvoker,
    Agent: serveapi.AgentInfo{
        Name: "Ailang World",
        Version: "1.0.0",
    },
})
if err != nil { log.Fatal(err) }

mux := http.NewServeMux()
api.Mount(mux)
// The host owns http.Server lifecycle, TLS, middleware, and listening.
```

### Opting into feedback as an embedder

```go
func (w *World) Tools(ctx context.Context, session serveapi.Session) ([]serveapi.ToolDescriptor, error) {
    tools := w.toolsFor(session)
    if w.feedbackAllowed(session) {
        tools = append(tools, serveapi.ToolDescriptor{
            Name: "submit_feedback",
            Description: "Submit feedback for this World session",
            InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`),
        })
    }
    return tools, nil
}
```

This is ordinary host-owned capability selection. AILANG does not supply storage or feedback
semantics on the embedded path.

## Conflict Surface

| Surface | Potential conflict | Design containment | Required regression measurement |
|---------|--------------------|--------------------|---------------------------------|
| Written single-filtering-point invariant | A caller-supplied set could bypass `isExposed`, or MCP/A2A could grow separate filters | Generalize to one `AuthorizedSurface` gateway; consumers only project/lookup; update the rule in the same sprint | Existing filter tests plus A/B cross-protocol exact-set tests |
| MCP discovery and dispatch | SDK tools are registered objects, so process-global registration could leak A's tools to B | Create the SDK server and closures inside the existing stateless per-request factory | Interleaved A/B `tools/list` and unauthorized-call counter tests |
| MCP transport mode | A future `Stateless:false` would make GET establish state while a later POST receives a fresh server with an empty stream/session registry, breaking correlation and resumption | Freeze `Stateless:true`; any stateful change must replace/revisit the per-request-server decision | GET returns 405 with `Allow: POST`, plus headed POST sentinel |
| A2A discovery and dispatch | Card generation has a request hook while task dispatch currently resolves module exports separately [V7, V15] | Resolve once at outer handler and use the same request-local surface for card and send | Same sentinels through card and task send |
| Standalone CLI | Reusing embedded defaults could remove `submit_feedback` or module exports | Keep CLI on existing constructor/start path; do not route it through `serveapi.New` | Default-present and explicit-suppression sentinel tests |
| OpenAPI and REST handler | Generalizing exposure could accidentally make host tools REST/OpenAPI-visible or alter export filters | Embedded surface is consumed only by MCP/A2A adapters; loaded-export branch remains authoritative elsewhere | Existing OpenAPI/handler filtering tests unchanged |
| MCP/A2A codecs | Parallel new encoders could drift from existing names, schemas, errors, or framing | Extract/reuse projection and result helpers; retain MCP SDK HTTP handler | Protocol tests pin schema, JSON-RPC errors, content types, and SSE events |
| Concurrent host mutation | Caller may reuse/mutate descriptor slices after return | Deep-copy and sort request-local descriptor data before registration | Race test mutates original after callback and asserts served copy is stable |
| Host callback wait | Resolver/provider/invoker may ignore context and block forever | Per-call deadline context plus handler-owned goroutine/select wrapper and buffered result channel | Blocking callback table asserts exact errors inside configured bound |
| Public package layering | Public facade could expose compiler/internal types or trip architecture rules | Public structs use stdlib types only; facade delegates inward; no reverse import | External compile fixture and `make check-boundaries` |
| File-size gate | Adding public/mount methods to the 764-line server file could exceed 800 | New focused files; no additions to `server.go` | `make check-file-sizes` |

**Deliberate changes:** only users of the new `serveapi` package observe the callback-owned exact
surface and opt-in built-ins. No existing CLI, REST, OpenAPI, MCP stdio, MCP HTTP, or A2A default is
deliberately changed.

## Testing Strategy

**Unit tests:**

- Descriptor validation, deep copy, deterministic sorting, and duplicate detection.
- Resolver-before-provider-before-invoker ordering using a recorded event sequence.
- Exact pointer-session propagation and callback error mapping.
- Shared MCP/A2A projection fields from non-default sentinel descriptors.

**Integration tests:**

- MCP SDK client over in-memory transports/recorders for two principals and exact tool lists.
- A2A card and task JSON-RPC through `httptest.ResponseRecorder` for the same principals.
- Caller-owned mux routing at all three default patterns.
- External-module import/compile fixture for the public package boundary.

**Regression tests:**

- Existing `internal/apiserver` filtering, MCP schema, feedback surface, A2A gate, and protocol tests.
- Existing `cmd/ailang/serve_api_mcp_surface_test.go` CLI surface test.
- Repository-wide tests outside a sandbox that permits loopback binding.

**Concurrency tests:**

- Interleave 100 A/B discovery requests under `go test -race`; assert no foreign sentinel appears.
- Block A's `Tools` callback while B completes to prove there is no shared mutable active surface.
- Deliberately block each callback past a 20 ms configured limit and assert the handler returns the
  exact MCP/A2A timeout outcome before a 250 ms outer test deadline.

## Success Criteria

- [x] External hosts can import `serveapi` and mount/obtain MCP and A2A handlers.
- [x] Resolver runs before every discovery or invocation and its exact session value reaches both
  `Tools` and `Invoke`.
- [x] One descriptor callback result generates both the MCP tool list and A2A skill list.
- [x] Dispatch is restricted to the descriptor set used for that request's authorization.
- [x] Embedded empty surface exposes zero tools/skills and no `submit_feedback`.
- [x] Embedded explicit feedback descriptor dispatches only through the host invoker.
- [x] Standalone CLI default continues exposing `submit_feedback`; Lane A opt-out continues hiding it.
- [x] MCP transport remains SDK-owned and protocol/SSE framing tests pass.
- [x] Embedded MCP is explicitly stateless; GET returns 405 with `Allow: POST` and per-POST request
  servers remain self-contained.
- [x] Every host callback is handler-time-bounded and blocking callback tests return the specified
  MCP/A2A errors within the configured interval rather than hanging.
- [x] `go test ./...`, `make check-boundaries`, `make check-file-sizes`, format, lint, and race-focused
  tests pass in an environment that permits required sockets.
- [x] Public API documentation states AILANG/host ownership and includes a compiling mount example.

## Non-Goals

**Not in this feature:**

- World persistence, stores, schemas, migrations, schedulers, queues, task state, or session storage.
- Authentication policy, token parsing, principal models, capability computation, or revocation
  storage; the host supplies these behind callbacks.
- Re-designing Lane A, changing `--no-feedback-tool`, or changing standalone defaults.
- Dynamic REST/OpenAPI endpoints generated from caller descriptors.
- Replacing the MCP Go SDK, hand-writing SSE, or changing upstream MCP/A2A wire semantics.
- Public exposure of `internal/apiserver.Server`, `ExportInfo`, compiler engine, or effect context.
- Stateful/resumable MCP sessions and `EventStore`-based stream resumption. Lane B requires
  stateless HTTP [V22-V26]; in stateful mode a GET-created stream followed by a POST routed to a
  fresh per-request server would lose the stream/session registry and break correlation. Supporting
  that mode requires a separately designed shared server/session lifecycle.

## Timeline

**Day 1 (~6 hours):** M1 public contract, external compile fixture, authorized surface, and exposure
gateway refactor.

**Day 2 (~6 hours):** M2 request-scoped MCP adapter, invocation mapping, exact-surface and SSE
regressions.

**Day 3 (~5 hours):** M3 A2A adapter, caller mux, compatibility tests, docs, race/boundary/size gates,
and review buffer.

**Total: ~17 hours across 2-3 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| MCP SDK server construction per request adds allocation cost | Medium | Preserve stateless semantics first; benchmark construction and optimize immutable codec data only if measured |
| Session leakage through closures or globals | High | Request-local server/surface, pointer sentinels, interleaving tests, and race detector |
| Generalizing `isExposed` weakens a security invariant | High | Make authorization a named central type, prohibit consumer filtering, test discovery and dispatch against the same set |
| Standalone feedback behavior changes accidentally | High | Separate constructor path and positive/negative feedback sentinels |
| Public API freezes excessive protocol detail | Medium | Expose protocol-neutral JSON descriptors/results and opaque sessions; keep SDK/internal types private |
| Sandbox produces false test failures | Low | Label bind failures uninformative and rerun socket-dependent suite outside sandbox |
| Host callback ignores cancellation | High | Handler-side select enforces `CallbackTimeout`; bounded tests use callbacks that never return |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Descriptor copies are validated and sorted before projection |
| A2: Replayability | 0 | Host callback replay policy remains outside this protocol adapter |
| A3: Effect Legibility | +1 | Resolution, discovery, and invocation are explicit callbacks |
| A4: Explicit Authority | +2 | The host supplies the complete per-session authority set; built-ins are not ambient |
| A5: Bounded Verification | +2 | One request-local surface is testable across protocols and every host callback has a measured deadline |
| A6: Safe Concurrency | +1 | Request-local immutable copies prevent cross-session mutable surface sharing |
| A7: Machines First | +1 | Exact machine-readable schemas replace process-wide accidental discovery |
| A8: Minimal Syntax | 0 | No AILANG syntax changes |
| A9: Cost Visibility | 0 | No new billed resource or hidden persistence |
| A10: Composability | +1 | Standard handlers compose with caller-owned muxes and middleware |
| A11: Structured Failure | +1 | Resolver, descriptor, authorization, and invocation failures map to protocol errors |
| A12: System Boundary | +2 | AILANG owns protocols while the host explicitly owns identity, state, and capabilities |

**Net Score: +12** — proceed to sprint planning.

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism is added by the adapter.
- [x] A3 (Effects): all host interaction occurs through named callbacks.
- [x] A4 (Authority): the embedded surface contains only caller-supplied descriptors.
- [x] A7 (Machines First): both protocols receive deterministic structured projections.

## Verification Log

Every current-code claim in this document cites one or more rows below. Output is quoted literally;
commands with narrowed scope say so.

| ID | Claim | Exact command | Observed output |
|----|-------|---------------|-----------------|
| V1 | Outside `internal`, `runtime` is a library precedent but not the only non-`main` package; same-module CLI code imports `internal/apiserver`. | `go list -f '{{.ImportPath}} {{.Name}}' ./... \| awk '!/\\/internal\\// && $2 != "main" {print}'; rg -n 'github.com/sunholo-data/ailang/internal/apiserver' cmd/ailang/serve_api.go` | `github.com/sunholo-data/ailang/runtime runtime`<br>`github.com/sunholo-data/ailang/std std`<br>`github.com/sunholo-data/ailang/tests/golden/bytecode bytecode_golden`<br>`github.com/sunholo-data/ailang/tests/golden/codegen codegen_golden`<br>`github.com/sunholo-data/ailang/testutil testutil`<br>`12: "github.com/sunholo-data/ailang/internal/apiserver"` |
| V2 | `runtime` has four non-test files and seven exported helper functions; its only repository import is one internal test. | `find runtime -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -print \| sort; rg '^func [A-Z]' runtime -g '*.go' -g '!*_test.go'; rg -n 'github.com/sunholo-data/ailang/runtime' --glob '*.go' .; rg -n 'github.com/sunholo-data/ailang/internal/apiserver' cmd/ailang/serve_api.go` | `runtime/convert.go`, `runtime/io.go`, `runtime/show.go`, `runtime/string.go`; exports `ConcatString`, `ConvertToInt64Slice`, `ConvertToStringSlice`, `ConvertToRecordSlice`, `Log`, `Debug`, `Show`; import output `./internal/repl/cons_expression_test.go:7: "github.com/sunholo-data/ailang/runtime"`; positive control `cmd/ailang/serve_api.go:12: .../internal/apiserver`. |
| V3 | There is no top-level `serveapi` directory or public callback API at HEAD; positive controls find `runtime` and `Config`. | `find . -maxdepth 2 -type d \\( -name serveapi -o -name runtime \\) -print; rg -n 'ResolveSession\|ToolProvider\|InvokeCallback\|type Config struct' --glob '*.go' internal/apiserver runtime cmd/ailang` | Directories: `./runtime`, `./internal/runtime`, `./benchmarks/runtime`; identifier output: `internal/apiserver/server.go:152:type Config struct {`. |
| V4 | The sole server constructor takes a 15-field flat `Config` whose block has no Request/Session/Principal member. | `rg -n '^type Config struct\|^func New\\(' internal/apiserver/server.go; sed -n '153,168p' internal/apiserver/server.go \| awk '$1 ~ /^[A-Z][A-Za-z0-9]*$/ {n++} END {print n}'; sed -n '152,169p' internal/apiserver/server.go \| rg -n 'type Config\|Request\|Session\|Principal'` | `152:type Config struct {`<br>`171:func New(basePath string, cfg Config) *Server {`<br>`15`<br>`1:type Config struct {` |
| V5 | The documented single filter is `isExposed(ExportInfo)` and production consumers call it. | `rg -n 'single filtering point\|All endpoint enumeration' .claude/rules/api-server.md; rg -n 'isExposed\\(' internal/apiserver/{a2a.go,handler.go,mcp.go,openapi.go,routes.go,server.go}` | Rule line 31 says `isExposed()` is the single filtering point; definition `routes.go:246:func (s *Server) isExposed(exp ExportInfo) bool`; calls at A2A 57/185, handler 142, MCP 91, OpenAPI 187, server 686. |
| V6 | `Start` owns listening and `buildRoutes` is unexported; no exported BuildRoutes/Handler/Mount sibling appears in the same file. | `rg -n '^func \\(s \\*Server\\) (Start\|buildRoutes)\|ListenAndServe' internal/apiserver/server.go; rg -n '^func \\(s \\*Server\\) (BuildRoutes\|Handler\|Mount\|buildRoutes)' internal/apiserver/server.go` | `511: StartMCP`, `517: Start`, `567: return srv.ListenAndServe()`, `570: func (s *Server) buildRoutes() *http.ServeMux`; negative/positive query returns only line 570. |
| V7 | A2A card building has a request parameter and filters through `isExposed`; MCP file has no ListTools/tools-list request hook, with streamable construction as positive control. | `rg -n 'buildAgentCard\\(\|isExposed\\(' internal/apiserver/a2a.go; rg -n 'ListTools\|tools/list\|NewStreamableHTTPHandler' internal/apiserver/mcp.go` | A2A: lines 22, 31 (`buildAgentCard(r *http.Request)`), 57 and 185 (`isExposed`); MCP query returns only `303:return mcp.NewStreamableHTTPHandler(`. |
| V8 | The architecture rule defines core/dashboard/embed directions, and the current boundary gate passes. | `rg -n 'dashboard\|internal/embed\|compiler' .claude/rules/architecture.md; make check-boundaries` | Rules identify core, dashboard/apps, and `internal/embed`, including the two forbidden directions; gate: `Checking architecture boundaries (logical layers over internal/)...` then `OK: no architecture boundary violations.` |
| V9 | MCP HTTP uses SDK stateless mode, and resolved SDK v1.7.0 defines a temporary session per request with SDK-controlled stream behavior. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); printf 'resolved=%s\\n' "${moddir}"; rg -n 'Stateless: true' internal/apiserver/mcp.go; sed -n '128,145p' "${moddir}/mcp/streamable.go"` | `resolved=/Users/voightkampff/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.7.0`; MCP line 305 has `StreamableHTTPOptions{Stateless: true}`; SDK comment says it `uses a temporary session ... for each request` and server-to-client notifications may reach the client in an incoming request context. |
| V10 | Standalone CLI's feedback suppression flag defaults false and is wired into `Config`. | `rg -n 'no-feedback-tool\|NoFeedbackTool' cmd/ailang/serve_api.go` | `34: noFeedbackToolFlag := fs.Bool("no-feedback-tool", false, ...)`; `142: NoFeedbackTool: *noFeedbackToolFlag`; help at line 261. |
| V11 | MCP construction registers feedback unless suppressed, and tests use positive controls for both default and suppressed surfaces. | `sed -n '23,52p' internal/apiserver/mcp.go; sed -n '90,118p' internal/apiserver/feedback_tool_surface_test.go` | Constructor calls `ms.registerTools()`, then `if !srv.noFeedbackTool { ... ms.registerFeedbackTool() }`; tests require `status`, require default `submit_feedback`, and reject it under `NoFeedbackTool:true`. |
| V12 | Server file is 764 lines and the current 800-line gate passes. | `wc -l internal/apiserver/server.go; make check-file-sizes` | `764 internal/apiserver/server.go`; `Checking for files >800 lines...` then `✓ All files within 800 line limit`. |
| V13 | Supplied Lane A hash is wrong at this checkout; actual PR #529 squash is `aa02f0d9f`. | `git show --stat --oneline --no-renames a81d66983 \| head -3; git log --all --oneline -- cmd/ailang/serve_api.go internal/apiserver/feedback_tool_surface_test.go \| head -2; git show --stat --oneline --no-renames aa02f0d9f \| head -4` | `a81d66983 fix(test): a suite whose properties ran zero cases no longer reports success (#517 Lane A) (#536)`; log begins `aa02f0d9f feat(serve-api): exact MCP tool surface — --no-feedback-tool + A2A dispatch gate (Lane A of #498, #528) (#529)`; its stat includes `cmd/ailang/serve_api.go`. |
| V14 | Standalone CLI constructs `apiserver.Config`, calls `apiserver.New`, loads modules, and calls `Start`. | `rg -n 'cfg := apiserver.Config\|apiserver.New\|LoadModules\|srv.Start' cmd/ailang/serve_api.go` | `127: cfg := apiserver.Config{`, `145: srv := apiserver.New(basePath, cfg)`, `149: if err := srv.LoadModules(paths)`, `184: return srv.Start()`. |
| V15 | Existing A2A projection derives skill fields from exports; its outer handler has the HTTP request, the send helper does not, and dispatch invokes the engine after a separate exposure check. | `sed -n '52,92p' internal/apiserver/a2a.go; rg -n '^func \\(s \\*Server\\) handleA2ATask\|handleA2ATaskSend\\(' internal/apiserver/a2a.go; sed -n '175,225p' internal/apiserver/a2a.go` | `"id": skillID`, `"name": export.Name`, `"description": desc`, `"tags": tags`, `"examples": []string{}`<br>`108:func (s *Server) handleA2ATask(w http.ResponseWriter, r *http.Request) {`<br>`133: s.handleA2ATaskSend(w, &req)`<br>`142:func (s *Server) handleA2ATaskSend(w http.ResponseWriter, req *a2aRequest) {`<br>`found = s.isExposed(e)`<br>`result, callErr := s.engine.CallPreserveFloats(modulePath, funcName, args...)` |
| V16 | Existing MCP projection constructs SDK tools and invocation closures from exports. | `sed -n '145,205p' internal/apiserver/mcp.go; sed -n '205,275p' internal/apiserver/mcp.go` | Projection builds `&mcp.Tool{Name: toolName, Description: desc, InputSchema: inputSchema}` and calls `AddTool`; handler calls `ms.server.engine.CallPreserveFloats(...)` and returns `mcp.CallToolResult`. |
| V17 | OpenAPI, HTTP dispatch, and startup banner each use the same current filter. | `sed -n '176,195p' internal/apiserver/openapi.go; sed -n '132,150p' internal/apiserver/handler.go; sed -n '678,692p' internal/apiserver/server.go` | Each excerpt contains `if !s.isExposed(...) { continue/404 }`. |
| V18 | Boundary gate package sets do not include `internal/apiserver`; positive controls list both enforced sets. | `sed -n '35,55p' scripts/check_boundaries.sh; sed -n '106,122p' scripts/check_boundaries.sh` | `CORE_PKGS=(parser ... iface)`, `DASHBOARD_PKGS=(server coordinator observatory messaging)`, and loops enforce core→dashboard and dashboard→compiler-surface rules; `apiserver` is absent from both sets. |
| V19 | Narrowed API-server tests pass; broader CLI-package result is uninformative under sandbox. | `go test ./internal/apiserver ./cmd/ailang` | `ok github.com/sunholo-data/ailang/internal/apiserver 1.755s`; CLI test panics with `httptest: failed to listen on a port: listen tcp6 [::1]:0: bind: operation not permitted`. **UNINFORMATIVE UNDER SANDBOX for `./cmd/ailang`; not a pass or product failure.** |
| V20 | GitHub issue fetch was unavailable in the sandbox. | `gh issue view 498 --repo sunholo-data/ailang --json number,title,body,url 2>&1` | `error connecting to api.github.com` / `check your internet connection or https://githubstatus.com`. |
| V21 | Related-doc search found prior MCP quality/filter work but no matching planned Lane B document. | `ailang docs search "MCP exact tool surface embeddable session callback" --limit 8 2>&1; rg -l -i 'embedd\|MCP\|A2A\|submit_feedback\|NoFeedbackTool' design_docs/planned design_docs/implemented \| sort` | Search reported `Scanned: 1369 docs`; closest MCP-specific result was `design_docs/implemented/v0_11_0/m-mcp-quality-and-route-headers.md (0.55)`. The filename scan included implemented MCP/filtering docs and no planned exact-tool-surface Lane B doc. |
| V22 (M1) | SDK v1.7.0 stateless mode uses a temporary session per request and explicitly rejects GET and DELETE with 405. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); sed -n '124,146p' "$moddir/mcp/streamable.go"` | `A stateless server does not read or set the Mcp-Session-Id header, and uses a temporary session with default initialization parameters for each request.` and `In Stateless mode, GET and DELETE requests return 405 Method Not Allowed.` |
| V23 (M2) | `ServeHTTP` has distinct stateless/stateful dispatch, so correctness depends on the selected option. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); sed -n '356,365p' "$moddir/mcp/streamable.go"` | `if h.opts.Stateless { h.serveStateless(w, req) } else { h.serveStateful(w, req) }`. |
| V24 (M3) | SDK stateless serving supports POST only, creates a temporary request-scoped session, and implements non-POST as 405 with `Allow: POST`. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); sed -n '367,387p' "$moddir/mcp/streamable.go"` | Comment: `Stateless servers only support POST. Each request creates a temporary session that is closed when the request completes.` Body: `if req.Method != http.MethodPost`, `w.Header().Set("Allow", "POST")`, and `http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)`. |
| V25 (M4) | SDK invokes `getServer(req)` per validated stateless POST, and AILANG's current factory already receives then discards that request while explicitly selecting stateless mode. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); sed -n '373,406p' "$moddir/mcp/streamable.go"; sed -n '303,306p' internal/apiserver/mcp.go` | After method, Content-Type, and Accept checks: `server := h.getServer(req)`. AILANG: `func(r *http.Request) *mcp.Server { return ms.mcpServer },` and `&mcp.StreamableHTTPOptions{Stateless: true},`. |
| V27 | **A2A's existing task error wire format is HTTP 200 + a JSON-RPC error envelope, NOT HTTP 5xx** — so mandating 200 + JSON-RPC on callback timeout preserves the current surface rather than corrupting it. (Quorum R2 `gemini-3-1-pro` objection; measured by the CONTROLLER outside the sandbox and applied as the reviewer's own proposed fix.) | `grep -n "32603\|32600\|32601\|32602\|WriteHeader\|StatusInternalServerError\|jsonrpc" internal/apiserver/a2a.go` | `304: w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200.` plus task errors `127: a2aError(w, req.ID, -32600, ...)`, `137: ... -32601 ...`, `145/158/166/178/190/267: ... -32602 ...`. No `StatusInternalServerError` on the task path. |
| V28 | `-32603` is NOT currently used in `a2a.go`, so this design introduces it (the standard JSON-RPC internal-error code) rather than redefining an existing one. Empty result paired with a known-positive control in the same check, per the negative-result rule. | `grep -c "32603" internal/apiserver/a2a.go; grep -c "32602" internal/apiserver/a2a.go` | `0` for `-32603`; control `6` for `-32602` — the instrument demonstrably matches this file, so the zero is a measurement and not a broken pattern. |
| V26 (M5) | Stateless POST still requires both JSON and SSE Accept media types. | `moddir=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); sed -n '388,399p' "$moddir/mcp/streamable.go"` | Comment: `Accept must contain both 'application/json' and 'text/event-stream'.`; failure text repeats both types and returns `http.StatusBadRequest`. |

**Rows V29–V33 added 2026-08-04 by the CONTROLLER at sprint-planning time (iteration 138).** The
planner (opus) refuted or extended five premises; each was then reproduced first-party by the
controller before being written here, per the mission's rule that a sub-agent finding is a claim
until the controller measures it. V29/V30 make two of M3's acceptance criteria **vacuous as
written** and are corrected in the AC list above; V31 adds a genuine robustness requirement the
design's optional `InputSchema` invites.

| ID | Claim | Exact command | Observed output |
|----|-------|---------------|-----------------|
| V29 | **`make check-file-sizes` is BLIND to a new top-level `serveapi/` package** — the gate body enumerates `find internal cmd` only, so M3's "size gate passes" AC would pass identically if `serveapi/serveapi.go` were 5,000 lines. Extends V12, which measured the gate passing without measuring its SCOPE. | `sed -n '122,128p' make/code-health.mk`; then `find internal cmd -name '*.go' \| grep -c 'internal/apiserver/'` and `find internal cmd -name '*.go' \| grep -cE '^(runtime\|std)/'` and `find runtime std -name '*.go' \| wc -l` | Gate body: `for file in $(find internal cmd -name "*.go"); do ... if [ $SIZE -gt 800 ]`. Counts: **45** files under `internal/apiserver/`; **0** under `runtime/`+`std/` — while **6** `.go` files genuinely exist there. The known-positive (45) proves the instrument works, so the 0 is a measurement of scope, not a broken pattern. |
| V30 | **`make check-boundaries` is BLIND to both `internal/apiserver` and a new `serveapi/`** — it iterates three FIXED package sets, none containing either, so M3's boundary AC is not discriminating: it passes whether or not `serveapi` imports the compiler core. Sharpens V18 (which recorded the absence but left the AC citing the gate anyway — a rule-3b scope case inside this document). | `make -pn \| grep -A6 '^check-boundaries:'`; then `grep -c apiserver scripts/check_boundaries.sh`, `grep -c parser scripts/check_boundaries.sh`, `grep -c serveapi scripts/check_boundaries.sh`; then `bash scripts/check_boundaries.sh` | Gate body: `@bash scripts/check_boundaries.sh`. Mentions: `apiserver` **0**, `serveapi` **0**, control `parser` **4** — the instrument demonstrably matches this file, so both zeros are measurements. Gate itself: rc=0, `OK: no architecture boundary violations.` Sets are `CORE_PKGS`, `DASHBOARD_PKGS`, `CORE_SURFACE_PKGS`. |
| V31 | **`mcp.Server.AddTool` PANICS on a host-supplied descriptor with a missing or non-object input schema.** Because the embedded design calls `AddTool` **per request** inside a handler goroutine from **caller-supplied** descriptors, a host returning `InputSchema: nil` (which this doc's `ToolDescriptor` permits) turns a host mistake into a panic on every request. M1's AC covered only the scalar case, not nil. | `SDK=$(go list -m -f '{{.Dir}}' github.com/modelcontextprotocol/go-sdk); grep -rn "missing input schema\|can't marshal input schema" "$SDK/mcp/"*.go; grep -c "panic(" "$SDK/mcp/server.go"` | `mcp/server.go:282: panic(fmt.Errorf("AddTool %q: missing input schema", t.Name))` and `mcp/server.go:294: panic(fmt.Errorf("AddTool %q: can't marshal input schema to a JSON object..."))`. Control: **16** `panic(` sites in that file, so the grep reaches it. SDK resolved at `v1.7.0`. |
| V32 | **`@nomcp` is already a SECOND, MCP-only filter downstream of `isExposed`** — so M3's instruction to restate "one filtering point" is false as written, and folding `@nomcp` into a single `AuthorizedSurface` would silently hide those exports from A2A and OpenAPI too, which is the opposite of the annotation's documented contract. | `sed -n '88,100p' internal/apiserver/mcp.go`; `grep -rn nomcp --include='*.go' internal/ cmd/ \| wc -l` | `mcp.go:91 if !ms.server.isExposed(export) { continue }` followed by `mcp.go:94-95 if export.IsNoMCP { continue // @nomcp: served over HTTP/OpenAPI/A2A but absent from MCP }`. **57** `nomcp` references repo-wide, including a dedicated `internal/apiserver/nomcp_test.go` asserting `TestNoMCP_StillServedOverHTTPAndOpenAPI`. |
| V33 | **`internal/apiserver` binds ZERO sockets today** — every `httptest.NewServer` site is in `cmd/`. So most of this sprint's suite is informative INSIDE the codex `workspace-write` sandbox, and M3's "outside the loopback-denying sandbox" caveat is overly pessimistic (it applies to `./cmd/ailang`, not to the new work). Favourable direction; recorded so the executor does not label good results uninformative. | `grep -rn 'httptest.NewServer' --include='*.go' internal/apiserver/ \| wc -l`; `grep -rn 'httptest.NewServer' --include='*.go' cmd/ \| wc -l`; `grep -rln 'httptest.NewRecorder' --include='*.go' internal/apiserver/ \| wc -l` | `internal/apiserver`: **0**. `cmd/`: **13**. Control — **12** files in `internal/apiserver` already use `httptest.NewRecorder`, proving the pattern-and-path combination matches when the thing is present. |

### Acceptance-criteria corrections forced by V29–V33 (iteration 138)

1. **M3's size-gate AC is replaced.** `make check-file-sizes` stays in the gate list (it must not
   regress `internal/`), but the *new package* needs its own explicit assertion: every file under
   `serveapi/` is ≤800 lines, checked by a command that actually looks there
   (`find serveapi -name '*.go' | xargs wc -l`). Without this the doc claims size coverage it does
   not have. **A follow-up issue should widen the repo gate itself to all first-party Go dirs** —
   that is a repo-wide change and therefore out of this sprint's scope, but it is the systemic fix.
2. **M3's boundary-gate AC is replaced** by a two-sided `go list` check: assert `serveapi`'s
   transitive imports contain no compiler-core package, paired with a known-positive control
   proving the query can see an import that IS present. `make check-boundaries` remains in the gate
   list as a non-regression check only, explicitly NOT as evidence about `serveapi`.
3. **M1 gains a nil-schema AC and panic-safety requirement (from V31).** Constructor/request
   validation MUST reject a descriptor whose `InputSchema` is absent or not a JSON object, with an
   explicit error, *before* any `AddTool` call. The per-request adapter additionally recovers from a
   panic raised while building the request-local server and converts it to the frozen internal-error
   envelope (`-32603`, HTTP 200 on MCP/A2A POST), because a host mistake must not take down the
   host's process. Test both input classes (nil and scalar) and assert the process survives.
   This EXTENDS the existing requirement that "a scalar input schema produces an explicit
   constructor/request error" to the input class it missed; it does not change the design direction.
4. **M3's single-filtering-point restatement is scoped (from V32).** The invariant to be written is
   "one authorized-surface gateway decides *membership*; `@nomcp` remains an MCP-only *projection*
   filter applied after membership" — `@nomcp` must NOT be folded into the shared gate, and
   `TestNoMCP_StillServedOverHTTPAndOpenAPI` is the regression that proves it wasn't.
5. **Sandbox labelling is narrowed (from V33).** Only `./cmd/ailang` results are
   `UNINFORMATIVE UNDER SANDBOX`. Results for `./internal/apiserver` and `./serveapi` are
   authoritative in-sandbox and must be reported as pass/fail, not waved away.

## Quorum Verification Log

Reviewers: `gpt5-6-sol`, `gemini-3-1-pro`, plus the controller's in-session verdict. Designer:
`codex:gpt-5.6-sol` (mission designer rotation). Two full rounds ran; no reviewer was absent, so
neither round was N−1 degraded. Metered cost: R1 $0.0868 + R2 $0.1042 = **$0.1910**.

**Round 1 — BLOCKED (2 objections, both accepted):**

| Reviewer | Objection | Resolution |
|---|---|---|
| `gpt5-6-sol` | Public handlers can wait indefinitely on host-controlled `ResolveSession`/`Tools`/`Invoke`; no incoming deadline required and no timeout established, violating bounded-waits at the new boundary. | Accepted in full → `Config.CallbackTimeout`, 5s default, zero⇒default, negative rejected, deadline-bearing contexts derived from the request, frozen MCP/A2A error mappings, discriminating 20ms/250ms test. |
| `gemini-3-1-pro` | Creating a fresh SDK server per request breaks MCP SSE, because SSE correlates a long-lived GET stream with later POSTs against a now-empty stream registry. | **Literal claim REFUTED by controller measurement** (V22–V26): in `Stateless` mode the SDK answers GET/DELETE with **405 `Allow: POST`** and calls `getServer(req)` per POST, so no cross-request stream exists to break — and AILANG already runs `Stateless: true` while discarding the request it is already handed. The reviewer nonetheless surfaced a real adjacent landmine, which was closed rather than waved away: `Stateless: true` is now a frozen requirement, stateful/resumable MCP is an explicit non-goal with the empty-registry failure mode written out, a Conflict Surface row forces any future stateful change to revisit the per-request-server decision, and acceptance asserts GET⇒405. |

**Round 2 — BLOCKED (2 new, narrower objections) → NARROW-REFINEMENT CARVE-OUT applied.**

Both remaining objections carried a concrete reviewer-authored `proposed_fix` and neither disputed
the design DIRECTION (both are completeness/safety objections), so the mission's ratified
narrow-refinement carve-out applies: the controller made one bounded revision applying the
reviewers' **verbatim** fixes. This SATISFIES the objections; it is not a force-pass, and no
contested design direction was overridden.

| Reviewer | Objection | Verbatim fix applied |
|---|---|---|
| `gpt5-6-sol` | The timeout bounds the *wait* but not callback *execution or resource consumption*: each context-ignoring callback is launched in a goroutine that may remain forever, so repeated requests leak unboundedly — while the doc claimed every invocation was bounded. | The reviewer's primary proposal, applied as written: `Config.MaxConcurrentCallbacks` with a positive safe default; acquire a capacity token before launching a callback and **hold it until the goroutine actually exits**; frozen protocol-specific overload envelope when capacity cannot be acquired within the deadline; explicit statement that **in-process Go callbacks cannot be forcibly terminated**; and the reviewer's test — permanently block more callbacks than capacity, send many more requests, assert starts stay capped and the exact overload envelope appears. The guarantee is narrowed accordingly: bounded handler latency and bounded callback STARTS, never enforced completion. |
| `gemini-3-1-pro` | The doc mandated HTTP 200 + JSON-RPC `-32603` for A2A timeouts without ever verifying A2A's *existing* error wire format; if A2A used HTTP 5xx, the mandate would silently corrupt the protocol surface. | The reviewer's proposed fix, applied as written: added the missing Verification Log entry inspecting the current A2A error serialization (**V27**), then confirmed the mapping against the verified schema. Measured by the controller outside the sandbox: `a2a.go:304` is `w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200.` and every existing task error is a JSON-RPC code — so the mandate **matches** the existing surface. **V28** additionally records that `-32603` is new to this file, with a known-positive control (`-32602` = 6) proving the zero is a measurement rather than a broken pattern. |

## Related Documents

- `design_docs/implemented/v0_11_0/m-mcp-quality-and-route-headers.md` — established MCP schema,
  naming, and shared exposure behavior; Lane B reuses its codec policy but adds a caller-owned source.
- `design_docs/implemented/v0_10_0/m-serve-api-endpoint-filtering.md` — originated the
  loaded-export filtering model generalized here.
- `design_docs/implemented/v0_10_12/m-serveapi-unify.md` — prior consolidation of serve-api paths;
  Lane B adds an external embedding boundary rather than another CLI mode.
- Lane A sprint/design artifacts introduced `--no-feedback-tool`; this document preserves them and
  specifies only per-session embedding.

## References

- **Issue contract**: `sunholo-data/ailang#498` (verbatim requested resolution supplied by mission;
  live fetch unavailable [V20])
- **Lane A**: PR #529, squash `aa02f0d9f` [V13]
- **API server invariant**: `.claude/rules/api-server.md`
- **Architecture boundary rules**: `.claude/rules/architecture.md`, `scripts/check_boundaries.sh`
- **MCP transport**: `github.com/modelcontextprotocol/go-sdk/mcp` v1.7.0 [V9]
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

If a later consumer demonstrates a requirement for stateful MCP server-to-client requests or event
resumption, design that transport mode separately with explicit shared server/session lifecycle
semantics. Enabling stateful mode without revisiting the per-request-server decision would route a
later POST to an empty registry rather than the server that owns the GET stream. It is not required
for Lane B and must not pull World state into AILANG.

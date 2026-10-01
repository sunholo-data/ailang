# M-SERVEAPI-DIRECTORY-READY: any `serve-api` service listable in the Anthropic and OpenAI directories

**Status**: Planned. **Phase A** (Anthropic) is ready for planning. **Phase B** (OpenAI) is gated behind discovery spike S1 (V11 only; V12 was resolved from OpenAI's auth page), which needs a live ChatGPT test outside the session. Quorum rounds 0 and 1 were both BLOCKED. Every objection was accepted and fixed (see §Quorum record). The re-quorum guardrail is spent, so this doc **awaits human ratification**; there is no round 2.
**Target**: v0.51.0
**Priority**: P1. It is the shared path to distribution for every hosted AILANG service. AILANG Parse is the first adopter.
**Estimated**: 7–9 days across four lanes: L1 3d, L2 1.5d, L3 1.5d, L4 2–3d (package). L5 is docs only.
**Dependencies**:
- M1 of the Parse listing doc (tool titles and hints): **shipped in v0.50.0** (`9305f1c19`), together with the zero-argument tool fix (`290e53886`).
- First adopter: ailang-parse `design_docs/planned/v0_49_0/v0_49_0_vendor_directory_listing.md`. It holds the Parse-specific decisions; this doc holds the generic ones.

**Sources**: the vendor quotes, with URLs and fetch date, are in `m-serveapi-directory-ready-sources.md` next to this doc.
**Quorum triggers fired**:
- #1: there are design-freeze items.
- #4: the load-bearing premises are vendor policies we do not control.

Run `ailang design-quorum` before planning.

---

## Problem Statement

A hosted AILANG service (`ailang serve-api --mcp-http`) can be added to an MCP client by URL. It
**cannot be listed** in either vendor directory, and every future service will hit the same
wall. Parse found the gaps one at a time.

**What the directories require.** These are external premises, quoted from vendor pages fetched
2026-10-01. The quotes, with URLs, are in `m-serveapi-directory-ready-sources.md`; re-fetch them at
submission time.

| Requirement | Anthropic | OpenAI |
|---|---|---|
| Every tool has a title and a read-only/destructive hint | Required (directory review) | Reviewed |
| Account-backed tools use OAuth | Policy §5D: *"Remote MCP servers that … require authentication must use secure OAuth 2.0"* | *"For an authenticated MCP server, you are expected to implement an OAuth 2.1 flow that conforms to the MCP authorization spec."* |
| Open discovery plus gated tools (lazy/mixed auth) | Supported, but *"Claude starts sign-in only when the HTTP request itself fails with `401 Unauthorized` and a `WWW-Authenticate` header. A tool handler can't produce that response."* | Per-tool `securitySchemes` (`noauth`/`oauth2`) **and** tool errors that carry `_meta["mcp/www_authenticate"]` |
| The model must not handle secrets | Supported auth types only (`oauth_*`, `custom_connection`, `static_headers` beta, `none`) | Restricted data: *"Access credentials and authentication secrets (such as API keys…)"* must not be collected |
| Client registration | CIMD (preferred), DCR, or Anthropic-held client (`oauth_anthropic_creds`) | CIMD (prioritized, `none` token auth), DCR, or a predefined client; redirect `https://chatgpt.com/connector_platform_oauth_redirect` (V12) |

**Current state (verified in-repo; see the Verification Log):**
- ✅ Titles and hints: `@mcp_title` / `@mcp_hints` (v0.50.0).
- ❌ No HTTP-layer auth challenge. `/mcp/` is exempt from `authMiddleware` (V1), and the MCP HTTP
  handler is a bare go-sdk streamable handler (V2). The 401 required for lazy auth cannot be
  produced today.
- ❌ No `/.well-known/oauth-protected-resource` route (V3).
- ❌ No per-tool `securitySchemes` and no auth `_meta` (V4).
- ❌ One tool surface per server. A listed surface without `apiKey` arguments or device-code
  tools cannot coexist with the agent surface (V5).
- ❌ No OAuth authorization server anywhere in AILANG (V6). The building blocks exist (V7–V10).
- ❌ No pre-submission check, so gaps are found by reviewers, one round trip at a time.

**Impact:** every hosted AILANG service that wants directory distribution: Parse now, the
code-execution service next. Without shared machinery each one re-solves OAuth, lazy auth and
packaging, and gets it subtly wrong in a different place.

## Goals

**Primary goal:** a service author adds a few annotations, configures one package and passes
one check, and the service is submittable to **Anthropic's directory (Phase A)**. OpenAI's
directory follows in **Phase B**, once spike S1 settles its wire behaviour.

**Success metrics (Phase A):**
- `ailang mcp check <url> --target anthropic` exits 0 against a deployed Parse listed surface.
- A gated tool called without a valid token gets an HTTP 401 with a `WWW-Authenticate` header
  that resolves to resource metadata. Open tools still answer. Verified on both MCP
  implementations.
- Claude (custom connector) completes OAuth against the package's authorization server and calls
  a gated Parse tool.
- The listed surface exposes zero credential-shaped parameters and zero device-code tools.
- A second service reaches the same `check` pass reusing the same package with no copied code.
  This is the reuse test, measured on the code-execution service.
- A verifier error, panic or timeout **never runs the gated tool** (fail-closed). This is tested.

**Success metrics (Phase B, after S1):**
- `ailang mcp check <url> --target openai` exits 0, and the ChatGPT developer-mode connector
  completes OAuth and calls a gated tool.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1: The authorization server is an **AILANG package** (`sunholo/mcp_oauth`), not `serve-api` Go | Lane routing (PROGRAM.md: extension by default). Policy in AILANG, Go is the shell. Identity and token issuance differ per service | human | design | high |
| D2: The lazy-auth challenge is generated by `serve-api` from `@mcp_auth`, with token **verification delegated to an AILANG function** | A tool handler cannot emit a 401, so the HTTP layer must. Verification is policy | human | design | high |
| D3: One process serves **two projections**: `/mcp/` (agent surface) and `/mcp/connect/` (listed surface) | One deploy, one tool source. The listed surface strips secret parameters and agent-only tools | human | design | med |
| D4: How a gated call is refused on the wire: HTTP 401 (Claude) vs tool error with `_meta` (OpenAI) | The two vendors describe different mechanisms. One choice may not satisfy both (PENDING V11) | human, after measurement | compile | med |
| D5: `ailang mcp check` lives under the existing `ailang mcp` group | CLI surface budget (S5: 89 → 17 commands). No new top-level command | agent | design | low |
| D7: Verifier failure is fail-closed. An error or panic gives HTTP 503 (not 401, which would loop the client through re-auth); a timeout over 5 s gives 503; the tool never runs | Security boundary. Silent fail-open would be account takeover | human | design | low |
| D6: Codes and tokens require `Rand[mode=crypto]` | Bare `Rand` is `math/rand` (V9). Guessable codes are account takeover | compiler (effect row) | design | low |

### Design Freeze

- [ ] D1: the package, not Go, owns the authorization server.
- [ ] D2: `@mcp_auth` plus a verifier function; verification stays in AILANG.
- [ ] D3: two projections in one process; the listed path name.
- [ ] D7: verifier failure semantics are fail-closed with a time bound (see L1). Ratify.
- [ ] D4: **Phase B only.** It is resolved by spike S1, not frozen now. Phase A ships the HTTP
  401 (Claude's documented mechanism) and does not emit `securitySchemes` or `_meta`.

## Solution Design

### Overview: five lanes

| Lane | What | Where | PROGRAM.md lane |
|---|---|---|---|
| L1 | `@mcp_auth` + `@mcp_token_verifier`: 401 challenge, resource metadata, `securitySchemes` / `_meta` | `serve-api` (Go) | AILANG fix |
| L2 | `@mcp_secret` / `@mcp_agent_only`: listed-surface projection at `/mcp/connect/` | `serve-api` (Go) | AILANG fix |
| L3 | `ailang mcp check <url> [--target anthropic\|openai\|both]` pre-submission linter | `cmd/ailang` | AILANG fix |
| L4 | `sunholo/mcp_oauth`: an OAuth 2.1 authorization server as AILANG `@route` functions | registry package | **extension** |
| L5 | Packaging template: Claude + Agent Plugins manifests, validator, CI (from `docparse-skill`) | docs + template | extension |

### L1: lazy auth in `serve-api`

```ailang
@mcp_auth("oauth2")
@mcp_title("Parse document")
@mcp_hints("readOnly", "openWorld")
export func mcpParse(filepath: string, outputFormat: string, _headers: Json) -> string ! {...}

-- One per server. Called by serve-api before any oauth2 tool runs.
@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {Net, Env, Clock} = ...
```

- **Default is `noauth`.** `@mcp_auth("oauth2")` marks a tool as gated.
- `serve-api --oauth-issuer <url>` serves `/.well-known/oauth-protected-resource`, with
  `resource` = the exact listed MCP URL and `authorization_servers` = [issuer].
- **The gate applies only to the listed mount, `/mcp/connect/`.** On `/mcp/`, `@mcp_auth` is
  ignored: tools run as they do today and validate their own `apiKey` argument or
  `_headers` (#85). Agent, CLI, SDK-bridge and MCP Registry integrations are unaffected. The
  security story on `/mcp/` is the existing in-tool validation, unchanged.
- **HTTP middleware** on the `/mcp/connect/` handler only:
  - It peeks the JSON-RPC body (bounded, then restored).
  - For a `tools/call` naming a gated tool, it checks the Bearer token by calling the
    `@mcp_token_verifier` function through the engine.
  - If the token is missing, or the verifier returns `false`, it returns **HTTP 401 +
    `WWW-Authenticate: Bearer resource_metadata="…"`** and the tool never runs.
  - **Verifier failure is fail-closed (D7).**
    - If the verifier call returns an engine error or panics (recovered), the response is
      **HTTP 503** with `Retry-After: 5` and a structured body naming `token_verification_unavailable`.
    - If the call does not complete within **5 s** (below Claude's 10 s endpoint budget), the
      response is also 503. How the bound is enforced, given what the engine supports (V15–V17):
      - The verifier runs through `Engine.CallPrepared`. Its hook sets the call's own
        `EffContext.GoCtx` to a request context with a 5 s deadline, so every Net request the
        verifier makes is aborted at the deadline (Net honours `GoCtx`).
      - The middleware does not wait past the deadline: it selects on the result or the timer,
        and the timer wins with a 503.
      - **Pure computation cannot be cancelled mid-evaluation** (the evaluator takes no context;
        V17). A verifier stuck in a pure loop would keep its goroutine. In-flight verifier calls
        are therefore capped by a semaphore (default 32, a `serve-api` flag). When it is full,
        the response is an immediate 503. A runaway verifier degrades to refusing gated calls,
        never to running them, and never to unbounded goroutines.
    - The tool **never runs** on any non-`true` result. 503 is used rather than 401 so a backend
      outage does not loop the client through a fresh OAuth flow.
    - The verifier's effect row is the service's declared choice. The bound is enforced by
      `serve-api`, not trusted to the function.
  - It reuses go-sdk's challenge formatting (`auth.RequireBearerToken`'s header shape, V7), but
    not the middleware as a whole: that gates every request, and lazy auth needs per-tool gating.
- **Phase B only (after S1):** `tools/list` adds `securitySchemes` per tool
  (`[{type:"noauth"}]` or `[{type:"oauth2"}]`), and auth refusals carry
  `_meta["mcp/www_authenticate"]` in the shape S1 shows ChatGPT acts on. `tools/list` itself
  stays open in both phases.
- **Both implementations:** the go-sdk path and `serveapi/protocol/mcphttp`, through
  `protocol.ToolDescriptor.Auth`. This is the same one-concept, two-implementations rule that M1
  followed.
- **Registration-time errors:** `@mcp_auth("oauth2")` without `--oauth-issuer`, or without a
  verifier, is logged as an ERROR and the tool is not registered. A gated tool that cannot be
  authorized is a configuration bug, not something to serve open.

### L2: listed-surface projection

- `@mcp_secret("apiKey")`: on `/mcp/connect/` the parameter is removed from the schema and binds
  its zero value. It must also be `@optional`, which registration checks.
- `@mcp_agent_only`: the tool is absent from `/mcp/connect/`. Use it for device-code tools,
  pricing tools on an OpenAI listing, and similar.
- `/mcp/` is unchanged, so CLIs, headless agents, SDK bridges and MCP Registry users keep the
  device flow.
- **Why annotations rather than two modules:** one source of truth for each tool. The listed
  surface is a *projection*, the same posture as `@nomcp` (`.claude/rules/api-server.md`:
  protocol-scoped narrowing after membership).

### L3: `ailang mcp check`

`ailang mcp check <url> [--target anthropic|openai|both] [--json]` exits non-zero on a failure.
Checks, each tied to a quoted requirement:

1. Every tool has a `title` and `readOnlyHint` or `destructiveHint`.
2. No credential-shaped parameter (`apiKey`, `token`, `password`, `secret`, …) on the listed
   surface.
3. Every zero-argument tool is callable with `{}`. (The 2026-10-01 regression.)
4. A gated tool called without a token returns 401 + `WWW-Authenticate`. The resource metadata
   resolves, and its `resource` equals the URL exactly.
5. The authorization server's metadata advertises `S256` and either CIMD (`client_id_metadata_document_supported` +
   `"none"` auth) or a `registration_endpoint`. The token endpoint accepts form-urlencoded.
   Discovery and token respond in under 10 s (Claude's limit).
6. `--target openai` (Phase B): `securitySchemes` present; pricing or upgrade language in tool
   descriptions is warned on (commerce rule).

### L4: `sunholo/mcp_oauth` (AILANG package)

- **Pure core**, unit-testable:
  - PKCE S256 verification: `base64url(sha256(verifier)) == challenge`, built from
    `sha256Hex`, then hex to bytes via `fromInts`, then `toBase64URL` (V10);
  - authorization-server metadata construction;
  - CIMD document validation (redirect URI must be in the document);
  - redirect URI matching, including Claude Code's port-agnostic loopback;
  - code and refresh-token record shapes.
- **Effectful shell** that the adopting service wires up as `@route`s:
  - `/.well-known/oauth-authorization-server`;
  - `/oauth/authorize`, which hands off to the **service's own login page** (a function the
    service supplies);
  - `/oauth/token` (form-urlencoded);
  - an optional `/oauth/register` (DCR).
- **The service supplies three functions:**
  1. `login`: where to send the user (Parse: the existing Firebase approve page);
  2. `mint`: account → access token (Parse: a scoped `dp_` key, so its existing Bearer path
     validates it);
  3. `store`: the code and refresh-token storage (Parse: Firestore via `sunholo/firestore`).
- **All code and token generation is `! {Rand[mode=crypto]}`** (D6), and the package's tests
  assert it.
- **Why a package:**
  - Identity backends differ per service: Firebase, the service's own accounts, or a future
    corporate IdP.
  - The authorization server is ordinary HTTP plus policy, which AILANG `@route`s already do.
    Parse's device auth is ~570 lines of AILANG doing the same kind of work (V8).
  - It keeps auth policy out of the frozen core.

### L5: packaging template

Document `sunholo-data/docparse-skill` as the reference layout:
- `.claude-plugin/` + root `plugin.json` / `mcp.json` (Agent Plugins);
- a shared `skills/`;
- `.agents/plugins/marketplace.json`;
- `scripts/validate-agent-plugins.mjs`;
- a CI workflow with Claude and Codex install smoke tests.

Its CI went green on 2026-10-01 (`a573d7c`). A scaffold command is deferred.

### Files to Modify/Create

- `internal/parser/parser_decl.go`: `mcp_auth`, `mcp_token_verifier`, `mcp_secret`, `mcp_agent_only` cases (~30 LOC)
- `internal/apiserver/routes.go`: extraction into `ExportInfo` (~60 LOC)
- `internal/apiserver/mcp_auth.go` (new): body-peek middleware, 401 challenge, verifier call, resource-metadata handler (~200 LOC)
- `internal/apiserver/mcp.go`: `securitySchemes` on tools; listed-surface projection; second mount (~80 LOC)
- `serveapi/protocol/descriptor.go`: `ToolDescriptor.Auth`, `.Secret` params (~30 LOC)
- `serveapi/protocol/mcphttp/methods.go`: emit `securitySchemes`; projection (~40 LOC)
- `cmd/ailang/mcp_check.go` (new): `ailang mcp check` (~300 LOC; split if over 500)
- `cmd/ailang/serve_api.go`: `--oauth-issuer` flag (~15 LOC); new env vars, if any, go through `internal/config`
- `docs/docs/guides/serve-api.md`: an "Listing in the directories" section
- `prompts/devtools/v0.8.0*.md` and the `cmd/ailang/prompts` copies: annotation reference (per `.claude/rules/api-server.md`)
- `examples/runnable/serve_api_mcp_oauth.ail`: gated plus open tools, verifier, `@mcp_secret`
- Package (separate repo): `sunholo/mcp_oauth`, with `ailang.toml`, `src/core.ail` (pure) and `src/routes.ail` (shell)

## Conflict Surface

This touches `internal/parser/` (new annotation names) and the MCP dispatch path.

1. **Positions extended:**
   - the annotation switch in `parseAnnotation` (four new names);
   - the MCP HTTP request path (new middleware before the go-sdk handler);
   - the `tools/list` tool object (a new `securitySchemes` field);
   - one new mount, `/mcp/connect/`.
2. **Other constructs already in those positions:**
   - existing annotations (`@route`, `@mcp_name`, `@optional`, `@nomcp`, `@mcp_title`,
     `@mcp_hints`, `@raw`, …). The new names are new switch cases, so nothing is reinterpreted
     (V5 lists the switch).
   - `authMiddleware`, which deliberately exempts `/mcp/` (V1). The new middleware is MCP-only
     and does not touch REST auth.
   - `_headers` injection (#85), which still binds after the gate passes.
   - `@optional` zero-value binding, which `@mcp_secret` reuses.
   - `@nomcp` narrowing, which `@mcp_agent_only` follows as a second projection-scoped
     narrowing. **This needs `.claude/rules/api-server.md` updated:** it currently says
     `@nomcp` is "the single sanctioned protocol-scoped narrowing".
3. **Disambiguation:**
   - Annotation names are exact identifiers.
   - The middleware acts only on POST `tools/call` naming a gated tool. Everything else passes
     through untouched.
   - The listed projection is chosen by mount path, not by request content.
4. **Programs that must still work:**
   - `examples/runnable/serve_api_mcp_header_auth.ail`: header auth on `/mcp/`;
   - `examples/runnable/serve_api_mcp_hints.ail`: annotations;
   - `mcp_tools/*.ail`: the public knowledge MCP, unauthenticated;
   - `internal/apiserver/nomcp_test.go`;
   - `internal/apiserver/mcp_header_auth_test.go`.
5. **Deliberate changes:**
   - The `api-server` rule's "single sanctioned narrowing" wording.
   - On **`/mcp/connect/` only**, an unauthenticated `tools/call` to an `@mcp_auth("oauth2")` tool
     is refused with HTTP 401 instead of running. **`/mcp/` behaviour is unchanged**, including
     for gated tools, which keep their in-tool `apiKey` / `_headers` validation. Fixture: an
     argument-carried-key call to a gated tool on `/mcp/` must still succeed.

## Examples

**Before:** to protect a tool, it takes `apiKey: string`; the model obtains a key through
`mcpAuth`/`mcpAuthPoll` and passes it as an argument. Both directories reject this.

**After:**

```ailang
@mcp_auth("oauth2")
@mcp_secret("apiKey")
@optional("apiKey")
export func mcpParse(filepath: string, outputFormat: string, apiKey: string, _headers: Json) -> string ! {...}

@mcp_agent_only
export func mcpAuth(label: string) -> string ! {...}
```

- On `/mcp/`, nothing changes for agents.
- On `/mcp/connect/`, `apiKey` and `mcpAuth` are gone. Claude gets a 401, runs OAuth against
  `sunholo/mcp_oauth`, and retries with `Authorization: Bearer <dp_ key>`. `_headers` carries
  that header to the tool.

## Success Criteria

- [ ] L1: a gated `tools/call` without a token gets 401 + `WWW-Authenticate` with `resource_metadata`; with a valid token the tool runs; open tools are unaffected. Tested on the go-sdk and `mcphttp` paths.
- [ ] L1: `@mcp_auth("oauth2")` without an issuer or verifier is a registration ERROR (mutation-tested).
- [ ] L1 / D7: a verifier that returns an error, panics, or sleeps past 5 s → HTTP 503 and the tool body never executes (asserted with a side-effect counter). Mutation-tested: making the error path fall through must fail the test.
- [ ] L1 / D7: with the semaphore saturated by a pure-looping verifier, gated calls get an immediate 503, open tools still answer, and the goroutine count stays at or below the cap plus a constant.
- [ ] L1 scope: on `/mcp/`, a gated tool called with an argument-carried key and no Bearer token **succeeds** (regression for every existing agent integration).
- [ ] L2: `/mcp/connect/` drops `@mcp_secret` params and `@mcp_agent_only` tools; `/mcp/` is unchanged (regression fixtures above).
- [ ] L3: `ailang mcp check` fails on today's prod Parse (no titles, apiKey params, no 401) and passes on the adopted listed surface.
- [ ] L4: pure-core unit tests (PKCE vectors from RFC 7636 Appendix B; CIMD; loopback matching); generators declare `Rand[mode=crypto]`.
- [ ] End to end: a Claude custom connector completes OAuth against Parse's listed surface.
- [ ] **Phase B gate:** spike S1 records V11 and V12 with evidence, then D4 is resolved and Phase B is planned. Phase B items are not acceptance criteria for Phase A.
- [ ] Docs and the api-server rule updated; changelog fragment; example passes `ailang check`.

## Testing Strategy

- **Unit:**
  - annotation parsing;
  - extraction;
  - challenge header format;
  - projection;
  - `check` rules against recorded fixtures;
  - package pure core, with RFC 7636 test vectors.
- **Integration:**
  - in-memory go-sdk client and `mcphttp` handler tests, following
    `mcp_tool_hints_test.go`;
  - an `httptest` server for the 401 path, which in-memory transports cannot see.
- **Manual / live:**
  - a Claude custom connector against a staging Parse;
  - ChatGPT developer mode (V11);
  - `codex mcp list` shows auth "OAuth" instead of "Unsupported".

## Deferred Decisions

- Default auth for a module (all tools gated unless `@mcp_auth("noauth")`). **Agent** may add
  this if adopters ask.
- Whether `check` also reads a local module rather than only a URL. **Agent** may choose.
- Refresh-token rotation policy in the package. **Agent** may choose, following OAuth 2.1 BCP
  defaults.
- A `sha256Base64URL` stdlib convenience. **Agent** may add it if the package helper proves
  awkward; the stdlib freeze gate applies.

## Non-Goals

- **Verifying OAuth tokens inside Go.** Verification is the service's AILANG function. Go only
  issues the challenge.
- **A managed IdP integration** (Auth0 etc.). The package's `login` function can point at one,
  but we do not ship an adapter.
- **`static_headers` / org-level API-key connectors.** These are an Anthropic beta for a limited
  set of organizations.
- **Directory submission itself.** Manual portal steps; per-service docs.
- **Fixing the stub purity in `iface`.** Tracked in ailang#1443.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| ChatGPT does not act on an HTTP 401 for `tools/call` (V11) | Phase B is gated on spike S1; Phase A does not depend on it |
| The verifier hangs or errors on the request path | D7: a 5 s deadline, fail-closed 503, and tests on every failure mode |
| Peeking the body adds latency or breaks streaming | Only POST bodies under a bounded size; `tools/call` requests are small; restore the body as a reader |
| Guessable codes or tokens | D6: `Rand[mode=crypto]` required; package tests assert it |
| Vendor rules change | `check` cites each rule; re-fetch the sources at each submission |
| Two projections drift | One source; projection by annotation; check runs against `/mcp/connect/` in CI |

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `/mcp/` is exempt from serve-api API-key auth | read `internal/apiserver/auth.go` (`authMiddleware` exempts the `/mcp/` prefix) | Confirmed |
| V2 | The MCP HTTP handler has no auth layer | read `internal/apiserver/mcp.go` `HTTPHandler` → bare `mcp.NewStreamableHTTPHandler(..., Stateless: true)` | Confirmed |
| V3 | No protected-resource metadata route exists | `grep -rln "oauth-protected-resource\|ProtectedResourceMetadata" internal serveapi cmd` → empty | Confirmed (negative) |
| V4 | No `securitySchemes` or auth `_meta` emitted | `grep -rn "securitySchemes\|mcp/www_authenticate" internal serveapi` → empty | Confirmed (negative) |
| V5 | Annotation set today | `parseAnnotation` switch in `internal/parser/parser_decl.go`: verify, route, mcp_name, mcp_title, mcp_hints, optional, allow_empty_ok, raw, nowrap, noexpose, nomcp | Confirmed; the four new names are unallocated |
| V6 | No OAuth authorization-server code in AILANG | `grep -rli oauth internal cmd serveapi` → only provider/executor/coordinator credential code, no `/authorize` or `/token` server | Confirmed (negative) |
| V7 | go-sdk v1.8.0 ships resource-side helpers | `$GOMODCACHE/.../go-sdk@v1.8.0/auth/auth.go:97` `RequireBearerToken` (emits `WWW-Authenticate: Bearer resource_metadata=…`), `:188` `ProtectedResourceMetadataHandler`; `oauthex/resource_meta.go:28` | Confirmed; reuse the header shape and metadata type, not the whole-handler gate |
| V8 | Parse auth is AILANG and composes registry packages | `docparse/docparse_api/services/device_auth.ail` (569 lines) imports `pkg/sunholo/{firestore,firebase_auth,auth/bearer,gcp_auth}` | Confirmed |
| V9 | Bare `Rand` is not cryptographic; `Rand[mode=crypto]` exists | `internal/builtins/rand.go` `randIntn`: the default "os" mode uses a `math/rand` source; "crypto" uses `crypto/rand`. `examples/modal_rand.ail`. `ailang check` on `func token() -> string ! {Rand[mode=crypto]} = uuid4()` → ✓ | Confirmed |
| V10 | PKCE S256 is possible in pure AILANG | `std/crypto` `sha256Hex`; `std/bytes` `fromInts`, `toBase64URL` (exports listed) | Confirmed (needs a hex→ints helper in the package) |
| V11 | How ChatGPT treats an HTTP 401 on `tools/call`, vs the `_meta["mcp/www_authenticate"]` tool error | **PENDING**: needs a live ChatGPT developer-mode test (outside the session) | **Gates D4** |
| V12 | ChatGPT's client registration and callback URL | OpenAI auth page, quotes O6–O7 in the sources file: CIMD prioritized with `none` token auth; redirect `https://chatgpt.com/connector_platform_oauth_redirect` | **Confirmed.** L4's CIMD path serves both vendors |
| V13 | Firebase Auth cannot act as an OAuth authorization server for third-party clients | Product knowledge; not re-checked | **Not load-bearing here.** L4 takes the identity provider as a `login` function, so the design is the same either way. It matters only to the Parse doc's choice of `login` |
| V15 | `serve-api` can invoke an arbitrary AILANG function per request | `makeToolHandler` already calls `ms.server.engine.CallPreserveFloats` per `tools/call` (`internal/apiserver/mcp.go`); `Engine.CallPrepared(prepare, …)` (`internal/embed/embed.go:182`) gives a hook on the call's own cloned effect context | Confirmed |
| V16 | Net effects honour a per-call Go context | `internal/effects/net_authorize.go:312` `requestContext` returns `EffContext.GoCtx`; `net.go:88/181/425` build requests with it | Confirmed. A `GoCtx` deadline set in the `CallPrepared` hook aborts verifier Net calls |
| V17 | The evaluator cannot cancel pure computation mid-evaluation | `Engine.Call*` take no `context.Context` (`embed.go:175–246`); `GoCtx` is consumed only by effect operations (grep `\.GoCtx` → runner, repl, effects ops/secret/net) | Confirmed (negative). Hence the semaphore cap, not a claimed kill |
| V14 | `ailang mcp` has a subcommand group to extend | `ailang mcp --help` → `status`, `help` | Confirmed |

## Phasing

| Phase | Scope | Premises | Status |
|---|---|---|---|
| **A: Anthropic** | L1 (401 challenge, resource metadata, verifier with D7), L2, L3 `--target anthropic`, L4 (CIMD + `oauth_anthropic_creds`), L5 | V1–V10, V14 confirmed; vendor quotes in the sources file | Ready to plan |
| **S1: discovery spike** | Live ChatGPT developer-mode test: an HTTP 401 vs a `_meta` tool error on `tools/call` | V11 | **Needs a human** (ChatGPT account); about 1 hour |
| **B: OpenAI** | `securitySchemes`, `_meta` refusals, `--target openai`, ChatGPT redirect URI in L4, commerce strip | S1 result; V12 already confirmed | Planned after S1 |

## Quorum record

**Round 0 (2026-10-01).** Seated: oc-glm-5-3, oc-kimi-k3, gemini-3-1-pro (reserve);
gpt6-1-sol absent. Result: **BLOCKED** (3 reject). Controller: pass. Artifact:
`.ailang/state/mission-quorum/m-serveapi-directory-ready-2026-10-01T12-36-13Z.json`.
All objections were accepted:
- **glm:** verifier failure semantics were undefined (fail-open possible). Fixed: D7 adds
  fail-closed 503 with a 5 s bound and tests for every failure mode. A5 is rescored.
- **kimi:** the goals and metrics committed to OpenAI on PENDING premises, and the evidence was
  in an ephemeral scratchpad. Fixed: Phase A / S1 / B split, with goals, metrics and criteria
  per phase; the quotes moved into the repo (sources file).
- **gemini:** V11–V13 were unverified. V12 has since been confirmed from OpenAI's auth page. V11 gates Phase B only, through spike S1. V13
  is shown to be non-load-bearing (L4 is identity-agnostic).

**Round 1 (2026-10-01).** Seated: oc-glm-5-3, oc-kimi-k3, gemini-3-1-pro; gpt6-1-sol absent.
Result: **BLOCKED** (3 reject). Artifact:
`.ailang/state/mission-quorum/m-serveapi-directory-ready-2026-10-01T12-38-55Z.json`.
All objections were accepted:
- **glm and gemini:** the gate's mount scope was ambiguous. A server-wide gate would 401
  every existing agent passing `apiKey` as an argument on `/mcp/`. Fixed: the gate applies to
  `/mcp/connect/` only; `/mcp/` is unchanged; there is a regression criterion for
  argument-carried keys on `/mcp/`.
- **kimi:** the 5 s bound rested on unverified engine capabilities. Verified: V15 (per-request
  engine call and a per-call effect-context hook), V16 (Net honours `GoCtx`), V17 (pure
  evaluation is not cancellable, a negative). The design now enforces the bound by deadline on
  Net, not waiting, and a semaphore cap. It no longer claims to kill a running verifier.

The guardrail (re-quorum once) is spent, so **a human ratifies the Design Freeze**. No round 2.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Auth gating is a deterministic function of the request and the verifier result |
| A2: Replayability | 0 | No change to traces; the verifier call is an ordinary effectful call |
| A3: Effect Legibility | +1 | The verifier's effects are declared; code generation must declare `Rand[mode=crypto]` |
| A4: Explicit Authority | +1 | Gated tools are declared per function; nothing gated runs without an explicit, verified grant |
| A5: Bounded Verification | +1 | Token verification is bounded on the request path: Net calls get a deadline, the middleware stops waiting at 5 s, and in-flight calls are capped (D7, V15–V17). A pure-looping verifier cannot be killed, but it cannot block or exhaust the server |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | `ailang mcp check` emits machine-checkable findings; annotations are data, not prose |
| A8: Minimal Syntax | 0 | Four annotations, all in the existing `@name(args)` form |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | One package serves every service; identity and storage plug in as functions |
| A11: Structured Failure | +1 | 401 + metadata instead of an opaque tool error; registration errors name the cause |
| A12: System Boundary | +1 | The OAuth and directory boundary is explicit: listed vs agent surface |

**Net score: +7. Proceed (Phase A).** No −1 on A1, A3, A4 or A7.

## Related Documents

- ailang-parse `design_docs/planned/v0_49_0/v0_49_0_vendor_directory_listing.md`: first adopter; the Parse-specific decisions.
- `design_docs/planned/v0_31_0/m-mcp-2026-07-28-adoption.md`: lists CIMD/DCR/auth as a non-goal *for the public knowledge MCP*. That endpoint stays unauthenticated; this doc does not change it.
- `design_docs/planned/v0_48_0/m-serveapi-sdk-free-mcp-dispatch.md`: the stdlib `mcphttp` dispatcher that L1 and L2 must also cover.
- `design_docs/planned/m-serveapi-protocol-only-module.md`: the protocol package's stdlib-only closure. `ToolDescriptor.Auth` must stay stdlib-only (`make check-protocol-closure`).
- ailang#1443: purity stub and `pure` keyword (found while landing M1).

## Future Work

- `ailang init plugin` scaffold from the L5 template.
- The code-execution service, which is the reuse test for every lane.
- A DNS-verified MCP Registry namespace per service.

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01

# Sprint Plan: M-SERVEAPI-DIRECTORY-READY, Phase A (Anthropic)

## Summary

Make a `serve-api` service pass Anthropic's directory requirements:
- a lazy-auth HTTP 401 gate with a fail-closed verifier;
- a listed MCP surface at `/mcp/connect/` that carries no secret parameters or agent-only tools;
- an `ailang mcp check --target anthropic` linter that proves both.

AILANG Parse adopts this afterwards, in `docparse`.

**Design doc:** `design_docs/planned/v0_51_0/m-serveapi-directory-ready.md`. The Design Freeze
(D1, D2, D3, D7) was ratified by Mark on 2026-10-01.
**Duration:** 5 working days (about 1,800 LOC including tests).
**Dependencies:** none blocking. M1 titles and hints shipped in v0.50.0, and `@mcp_hints()` is on `dev` (`41e73efc7`).
**Risk level:** medium. The new middleware sits on a security boundary and the request hot path.

**Out of this sprint:**
- L4 `sunholo/mcp_oauth` (a separate package repo; next sprint);
- L5 packaging template;
- Phase B, OpenAI (`securitySchemes`, `_meta` refusals, `--target openai`), which is gated on V11.

## Current Status Analysis

### Completed recently (2026-10-01)
- ✅ `@mcp_title` / `@mcp_hints`, both MCP implementations: ~580 LOC including tests, one session (`9305f1c19`).
- ✅ Zero-argument tool fix: ~110 LOC (`290e53886`). Empty `@mcp_hints()` plus a formatter round-trip fix: ~90 LOC (`41e73efc7`).
- ✅ Prod docparse v0.30.0 runs AILANG v0.50.0 with header API keys. M2 annotations are on `docparse` `main` and being promoted.

### Velocity
- Today's `serve-api` work ran at about 400 LOC per focused session, including tests and docs.
- The plan uses about 360 LOC/day, a 10% buffer, because the middleware is security-sensitive and needs `httptest` HTTP-level tests rather than in-memory MCP transports.

## Registry Reuse Audit

Searches run: `ailang pkg search oauth`, `mcp auth`, `bearer`, `token verify`, `mcp check`.

| Milestone | Package | Action | Reason |
|---|---|---|---|
| M1 | — | none | Parser annotation cases and `serve-api` extraction are core Go. No package can extend the annotation switch |
| M2 | — | none | The listed-surface projection is `serve-api` MCP registration (Go) |
| M3 | `sunholo/auth@0.4.1` (inspected) | none | Its bearer extraction runs inside an AILANG verifier function, which is the service's choice. The gate itself is Go middleware before any AILANG runs |
| M4 | — | none | `ailang mcp check` is a CLI subcommand under the existing `ailang mcp` group |
| M5 | — | none | Docs and example |

`sunholo/oauth@0.1.0` is an OAuth installed-app *client* (a consent URL plus a loopback code
exchange), not an authorization server. Re-check it at L4 as a possible `contribute` target.

## Proposed Milestones

### M1: Annotations and registration rules (~150 impl + ~150 tests) ✅

**Goal:** `@mcp_auth("oauth2"|"noauth")`, `@mcp_token_verifier`, `@mcp_secret("p", …)` and
`@mcp_agent_only` parse, format, are extracted into `ExportInfo`, and are validated at
registration.

**Tasks:**
- Parser: four cases in `parseAnnotation`, reusing `parseStringListAnnotation`. `@mcp_auth`
  takes exactly one argument from {oauth2, noauth}. The other two are parameterless, like
  `@nomcp`. Update the unknown-attribute hint lists.
- Formatter: confirm round-trip for each (the bare form for the parameterless ones; the
  `@mcp_hints()` lesson applies). Add tests to `internal/format/contracts_test.go`.
- Extraction in `internal/apiserver/routes.go`: `ExportInfo.MCPAuth`, `IsTokenVerifier`,
  `MCPSecret []string`, `IsAgentOnly`.
- Registration rules, each an ERROR that skips the tool, matching the `@mcp_name` posture:
  - an `@mcp_secret` name that is not a parameter, or not also `@optional`;
  - `@mcp_auth("oauth2")` without `--oauth-issuer`, or with no `@mcp_token_verifier`;
  - more than one verifier, which is an error for the whole server;
  - a verifier whose signature is not `(string) -> bool`.

**Acceptance criteria:**
- [x] Parser tests for every new annotation, valid and invalid.
- [x] Formatter round-trip test. N/A part: no formatter case was needed, because generic printing round-trips.
- [x] Every registration rule has a test that asserts the tool is absent and an ERROR is logged.
- [x] `.claude/rules/api-server.md`: `@mcp_agent_only` is documented as the second sanctioned
  projection-scoped narrowing (Conflict Surface item 5).

### M2: Listed surface `/mcp/connect/` (~200 impl + ~150 tests) ✅

**Goal:** one process serves `/mcp/` (unchanged) and `/mcp/connect/` (a projection).

**Tasks:**
- Build a second `mcp.Server` from the same candidate set:
  - skip `IsAgentOnly` exports;
  - `buildNamedInputSchema` omits `MCPSecret` parameters;
  - the handler binds them to their zero value. This reuses the `@optional` path, because a
    secret is required to also be `@optional` (M1).
- Mount it only when at least one export is listed-relevant (any `@mcp_auth`, `@mcp_secret` or
  `@mcp_agent_only`), or behind an explicit `--mcp-listed`. The agent decides which (Deferred).
- `submit_feedback` appears on both surfaces.

**Acceptance criteria:**
- [x] On `/mcp/connect/`, `tools/list` has no `@mcp_secret` parameters and no `@mcp_agent_only` tools.
- [x] `/mcp/` `tools/list` is byte-identical before and after for a module with none of the new annotations (regression).
- [x] `examples/runnable/serve_api_mcp_header_auth.ail` behaves the same on `/mcp/`.

### M3: Lazy-auth gate and resource metadata (~300 impl + ~250 tests) ✅

**Goal:** design doc L1 and D7, on the go-sdk path (`serve-api`).

**Tasks:**
- `--oauth-issuer <url>` flag, with any env var going through the `internal/config` Registry
  (the forbidigo rule).
- Serve `/.well-known/oauth-protected-resource` (and the RFC 9728 path-inserted variant for
  `/mcp/connect/`), with `resource` = the listed URL and `authorization_servers` = [issuer].
  Reuse go-sdk's `oauthex.ProtectedResourceMetadata` type.
- Middleware on the `/mcp/connect/` handler only:
  - bounded body read (≤ 1 MiB, then restored);
  - for a JSON-RPC `tools/call` naming a gated tool, read the Bearer token;
  - missing token → 401 + `WWW-Authenticate: Bearer resource_metadata="…"` (go-sdk header shape).
- Verifier call through `Engine.CallPrepared`. The hook sets `EffContext.GoCtx` to a 5 s
  deadline context. The middleware selects on the result or the timer:
  - `true` → proceed;
  - `false` → 401;
  - error, panic or timeout → 503 + `Retry-After: 5` + `token_verification_unavailable`;
  - semaphore full (default 32, a flag) → immediate 503.
- Passing the gate does not change `_headers` binding: the tool still sees `Authorization`.
- **The stdlib `mcphttp` path (embedders):** add `protocol.ToolDescriptor.Auth`. A reusable
  `mcphttp` gate option takes a host-supplied verifier `func(ctx, token) (bool, error)` with the
  same 401/503 semantics. The two paths must answer identically, so this is one table-driven
  test run against both handlers.

**Acceptance criteria:**
- [x] An `httptest` server: a gated tool with no token → 401, and the header's `resource_metadata` resolves; with a valid token → 200 and the tool runs; open tools → 200 with no token.
- [x] D7: a verifier that errors, panics or sleeps 6 s → 503 within 5.5 s, and the tool body's side-effect counter stays 0. Mutation-tested (making the error path fall through must fail).
- [x] Semaphore: with N slow verifiers in flight, call N+1 → immediate 503; open tools still answer; goroutine count is bounded.
- [x] On `/mcp/`, a gated tool called with an argument-carried key and no Bearer token **succeeds** (agent regression).
- [x] `mcphttp` parity test passes with the same cases.

### M4: `ailang mcp check --target anthropic` (~300 impl + ~200 tests)

**Goal:** the design doc's L3 checks 1–5 against a URL, with human and `--json` output, exiting 1 on failure.

**Tasks:**
- `cmd/ailang/mcp_check.go`, wired into the `ailang mcp` group (`mcp_status.go` is the sibling).
- Checks:
  1. a title and a readOnly/destructive hint on every tool;
  2. no credential-shaped parameters on the checked surface;
  3. zero-argument tools are callable with `{}`;
  4. a gated tool → 401, the resource metadata resolves, and `resource` matches exactly;
  5. authorization-server metadata has S256 and CIMD or a `registration_endpoint`, and responds
     in under 10 s. This is skipped with a WARN when there is no issuer yet, because L4 isn't
     built; the skip is never a pass.
- Every finding cites its requirement ID (A1–A6) from `m-serveapi-directory-ready-sources.md`.

**Acceptance criteria:**
- [ ] Fixture servers (`httptest`) for each check's pass and fail.
- [ ] Against today's prod Parse `/mcp/` it reports the expected findings: credential params (`apiKey`) and no 401 gate. Checks 1 and 3 pass once docparse M2 is promoted.
- [ ] Against the M5 example served locally on `/mcp/connect/`, checks 1–4 pass, and check 5 WARNs (no authorization server yet).
- [ ] `make check-cli-docs` and the CLI reference are updated (new subcommand).

### M5: Example, docs, release notes (~150)

**Tasks:**
- `examples/runnable/serve_api_mcp_oauth.ail`: open and gated tools, a verifier, `@mcp_secret`, `@mcp_agent_only`. Passes `ailang check`; run under `serve-api` in a test.
- `docs/docs/guides/serve-api.md`: a "Listing in the directories" section.
- `prompts/devtools/v0.8.0*.md` and the `cmd/ailang/prompts` copies: annotation reference (per `.claude/rules/api-server.md`).
- Changelog fragment(s). Mark Phase A done in the design doc.

**Acceptance criteria:**
- [ ] The example passes `ailang check`, and `ailang mcp check` passes checks 1–4 against it.
- [ ] `make ci-quick` gates pass; `make test` is green for `internal/apiserver`, `serveapi/...`, `internal/parser`, `internal/format` and `cmd/ailang`.

## Success Metrics
- `ailang mcp check --target anthropic` against the example on `/mcp/connect/`: checks 1–4 pass.
- Fail-closed and agent-regression tests exist, and are mutation-tested where marked.
- No regressions on `/mcp/` (header-auth, `@nomcp` and hints test suites).
- Documentation: `serve-api.md`, the devtools prompts, the api-server rule, and the CLI reference.

## Dependencies
- None for Phase A.
- The next sprint (L4 `sunholo/mcp_oauth`, then Parse adoption) depends on M3 (the verifier contract and resource metadata) and M4 (the check, used as the acceptance gate).

## Open Questions
None blocking; the freeze is ratified. These are deferred to the implementer, per the design doc:
- whether to mount `/mcp/connect/` automatically or by flag;
- whether to serve the path-inserted resource-metadata variant.

## Notes
- Verify on the wire with `serve-api --mcp-http` plus curl at each milestone, as was done for the titles and hints.
- Commit straight to `dev`, in small commits, one per milestone.

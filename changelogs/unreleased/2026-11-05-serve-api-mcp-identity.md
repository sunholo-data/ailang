### Added — `serve-api` hands the verified identity to the tool: `_auth` (2026-11-05)

`@mcp_token_verifier` may now return `Option[{subject: string, client_id: string, scopes: [string]}]`
instead of `bool` (inline record or a same-module record type). `None`, or `Some` with an empty
`subject`, is a rejected token (401). Bool verifiers work as before.

A tool that declares `_auth` of that record type receives the identity the Bearer gate verified.
The gate stamps it on an internal header that is deleted from every client request first and never
appears in a `_headers` binding; an `_auth` argument from the client is overwritten. Such a tool is
served only as a gated tool on `/mcp/connect/`: it is absent from `/mcp/`, stdio, REST, OpenAPI, A2A
and `_meta`, so no path runs it with an empty identity. Misuse is refused: at load for `_auth` with
`@mcp_auth("noauth")`, `@route`, `@nomcp`, `@mcp_agent_only`, `@optional`, `@mcp_secret`,
`@mcp_file`, `@mcp_ui_resource` or the wrong record shape; at startup for an ungated `_auth` tool or a
bool verifier. `--routes-only` still serves an `_auth` tool. Example:
`examples/runnable/serve_api_mcp_identity.ail`. M-MCP-OAUTH-CLIENT-MODELS M2 (D5).

`serveapi/protocol` gains `Identity`, `IdentityVerifyFunc`, `NewIdentityGate` and
`BearerGate.AdmitCallIdentity`. `NewBearerGate` and the embedded `mcphttp` path are unchanged.

### Added — `serve-api --mcp-subject-rpm` / `--mcp-subject-burst`: a per-user limit (2026-11-05)

Gated `/mcp/connect/` calls are limited per verified subject (burst defaults to the rpm value).
Beyond the limit a call gets HTTP 429 with `Retry-After` before the tool runs; other subjects are
unaffected, and a JSON-RPC batch spends one token per gated call. The flag needs `--mcp-http` and an
identity verifier, and is refused with `--mcp` (stdio). M-MCP-OAUTH-CLIENT-MODELS M2 (D6).

### Changed — the executor's OAuth service names the signed-in account (2026-11-05)

`executor_oauth/oauth.ail`'s `verifyMcpToken` returns the account and client it already looked up as
the identity, and a new `executor_whoami` tool shows the signed-in account. `test_oauth.sh` runs the
service with `--mcp-subject-rpm` and checks that a forged `_auth` or identity header changes nothing
(23 checks). This is the "per-user rate limit before prod" precondition of the GE executor design
(P1b); the Cloud Run args and the Terraform precondition change in multivac once a release carries
the flag.

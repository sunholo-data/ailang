### Added — lazy-auth gate on `/mcp/connect/` (M-SERVEAPI-DIRECTORY-READY M3)

- **The gate.** A `tools/call` on the listed surface naming an `@mcp_auth("oauth2")` tool must
  carry a Bearer token that the service's `@mcp_token_verifier` accepts. Otherwise:
  - with no token, or a rejected one, the response is **HTTP 401 +
    `WWW-Authenticate: Bearer resource_metadata="…"`** (plus `error="invalid_token"` when a token
    was rejected). This is the response that starts Claude's sign-in;
  - when the verifier errors, panics, takes over 5 s, or 32 verifications are already in
    flight, the response is **HTTP 503 + `Retry-After`**. It fails closed: the tool never runs.
  - `initialize`, `tools/list` and open tools need no token. `/mcp/` is not gated.
- **New flag `--oauth-issuer <url>`.** It names the authorization server in
  `/.well-known/oauth-protected-resource` (and in the RFC 9728 path-inserted form for
  `/mcp/connect/`). The metadata's `resource` is the listed URL as the client reached it,
  honouring `X-Forwarded-Proto`.
- **How the deadline reaches AILANG.** The verifier runs through `Engine.CallPrepared`, with the
  call's own `EffContext.GoCtx` set to the gate's deadline, so a hung `Net` request inside the
  verifier is cancelled (tested). Pure computation cannot be interrupted, so the in-flight cap
  bounds it instead.
- **One gate, both MCP implementations.** The logic is `protocol.BearerGate` (stdlib-only).
  `mcphttp` embedders set `Config.Gate` and `ToolDescriptor.Auth`, and get the same 401/503
  contract (parity test). A gated descriptor with no gate fails closed with 503.
- **Mutation-tested:**
  - error or panic admitting a call;
  - timeout admitting a call;
  - releasing a slot before the verifier ends;
  - the missing `GoCtx` deadline;
  - the gate not being wired;
  - a missing `mcphttp` gate failing open.
- **Found while testing:** integer division by zero crashes the runtime with a Go panic
  (ailang#1449). The gate recovers it and fails closed.

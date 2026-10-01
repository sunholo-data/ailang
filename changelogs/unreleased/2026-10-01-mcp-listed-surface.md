### Added — listed MCP surface at `/mcp/connect/` (M-SERVEAPI-DIRECTORY-READY M2)

- **One process now serves two projections of the same exports.**
  - `/mcp/` is the agent surface, unchanged.
  - `/mcp/connect/` is the directory surface: it lists no `@mcp_agent_only` tools, and
    `@mcp_secret` parameters are absent from `inputSchema`.
  - On `/mcp/connect/` a secret is **never taken from the client**, named or positional. It
    binds its zero value, so a model cannot pass an API key through a listed tool.
- `/mcp/connect/` is mounted only when some export uses `@mcp_auth`, `@mcp_secret`,
  `@mcp_agent_only` or `@mcp_token_verifier`. Servers that use none of them have the same
  route table as before.
- **Mutation-tested:** removing each of the three projection rules (agent-only skip, schema
  strip, client-secret override) fails a test.
- `ExportInfo` moved to `internal/apiserver/export_info.go`, because `server.go` hit the 800-line
  gate.

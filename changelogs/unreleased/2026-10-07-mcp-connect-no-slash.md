### Fixed — serve-api answers `/mcp/connect` and `/mcp` without the trailing slash

claude.ai's "Add custom connector" check POSTs the listed MCP URL without its trailing slash and does
not follow redirects. serve-api answered `/mcp/connect` with Go `ServeMux`'s 307 to `/mcp/connect/`,
so claude.ai reported "Couldn't determine how this server signs in" against a working OAuth server
(seen on AILANG Parse prod, 2026-10-07).

- `/mcp/connect` and `/mcp` are now served directly, with no redirect.
- A request on the no-slash form is pointed at no-slash metadata (RFC 9728 §3.3: the resource is the
  URL the client used, and the MCP TypeScript SDK rejects `/mcp/connect` against `/mcp/connect/`):
  - its 401 names `/.well-known/oauth-protected-resource/mcp/connect`;
  - that document's `resource` is `…/mcp/connect`.
- The slash form is unchanged.
- Tests: `internal/apiserver/mcp_noslash_test.go`, using a client that refuses redirects, plus
  `ailang mcp check` on the no-slash URL. Two mutants were killed: removing the bare route reproduces
  the prod 307, and removing the resource fix fails the RFC 9728 check.

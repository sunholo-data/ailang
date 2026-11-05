### Added — `serve-api --server-info` sets the service identity (2026-11-05)

`--server-info "name=parse,title=AILANG Parse,version=0.43.0"` sets MCP `serverInfo` on `/mcp/`,
`/mcp/connect/` and stdio (including the `title` field), the A2A card `version` and OpenAPI
`info.version`. These were three hardcoded constants (`ailang-api 0.8.1`, `0.8.1`, `0.8.1.1`) that
every listed service reported. Without the flag the name stays `ailang-api` and the version is now the
AILANG binary's. An unknown or repeated key, or an empty value, is a startup error.
M-MCP-OAUTH-CLIENT-MODELS M3a (D7).

### Fixed — `ailang://meta/modules` leaked hidden exports on both MCP surfaces (2026-11-05)

The MCP resource was registered on `/mcp/` and `/mcp/connect/` from the raw module map and ignored
`--no-introspection`. On v0.57.1 it listed `@noexpose` helpers, the `@mcp_token_verifier`'s name,
`@mcp_agent_only` tools on the listed surface, and each module's absolute path (design row V18).
Now `/mcp/connect/` does not register it at all. On `/mcp/` it is the `/api/_meta/modules` projection
(`exposedView`) without the token verifier, keyed by relative module path and sorted, and
`--no-introspection` turns it off. The resources capability stays advertised, so `ui://` widgets are
unaffected. M-MCP-OAUTH-CLIENT-MODELS M3b (D8).

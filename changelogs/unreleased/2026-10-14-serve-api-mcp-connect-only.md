### Added — `serve-api --mcp-connect-only`: a public MCP service with no ungated way to a tool

`@mcp_auth("oauth2")` gated a tool only on `/mcp/connect/`. The same tool was still served
ungated on `/mcp/`, and as a plain HTTP endpoint when it had an `@route`. That is fine for AILANG
Parse, whose tools also need an API key. It is not fine for a public service whose tools must never
run for an anonymous caller, such as the hosted AILANG executor.

With `--mcp-connect-only`:
- `/mcp/` is not mounted;
- every tool on `/mcp/connect/` defaults to `@mcp_auth("oauth2")` (an explicit `"noauth"` is honoured);
- plain HTTP serves `@nomcp` exports only, which are the OAuth and sign-in routes.

Startup fails without `--mcp-http`, without `--oauth-issuer`, without an `@mcp_token_verifier`,
or with `--a2a`. `TestMCPConnectOnly_*` covers each bypass (`/mcp/`, `@route`, the `/api/`
catch-all, OpenAPI) through the real route table; removing any one of the four guards fails it.

### Fixed — the serve-api startup banner lists the MCP endpoints actually mounted

It listed `POST /mcp/` whenever MCP was on, including under `--mcp-connect-only`, where `/mcp/` is a 404. It now
lists `/mcp/` only when that endpoint is mounted, and `/mcp/connect/` whenever the listed endpoint is.

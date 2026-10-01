### Added — listed-surface annotations: `@mcp_auth`, `@mcp_token_verifier`, `@mcp_secret`, `@mcp_agent_only` (M-SERVEAPI-DIRECTORY-READY M1)

- **What parses and is extracted:**
  - `@mcp_auth("oauth2"|"noauth")` gates a tool behind OAuth on the listed MCP surface. (The
    gate itself ships in M3.)
  - `@mcp_token_verifier` marks the server's one `(token: string) -> bool` verifier.
  - `@mcp_secret("p", …)` marks parameters the listed surface drops and binds to their zero
    value.
  - `@mcp_agent_only` keeps a tool on `/mcp/` and off the listed surface.
  - All four round-trip through `ailang fmt`.
- **Registration rules.** Each one is logged as an ERROR, the offending tool is not registered,
  and open tools are unaffected:
  - `@mcp_auth("oauth2")` without `--oauth-issuer` (`Config.OAuthIssuer`);
  - `@mcp_auth("oauth2")` with no verifier, more than one verifier, or a verifier whose
    signature is wrong;
  - an unknown auth scheme;
  - an `@mcp_secret` name that is not a parameter, is `_headers`, or is not also `@optional`.
- **The verifier is never an MCP tool,** and without `@route` it is never an HTTP endpoint,
  because exposed it would be a token-guessing oracle.
- **Mutation-tested:** with the rules disabled, all 7 rule cases fail; dropping the verifier's
  REST hiding fails the setup test.
- MCP parser tests moved to `internal/parser/mcp_attr_test.go`, because `route_attr_test.go` hit
  the 800-line gate.

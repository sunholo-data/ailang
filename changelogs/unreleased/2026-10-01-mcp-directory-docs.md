### Added — directory-listing guide and example (M-SERVEAPI-DIRECTORY-READY M5)

- New section in `docs/docs/guides/serve-api.md`, "Listing in the MCP directories (Anthropic,
  OpenAI)". It covers the two surfaces, the four annotations, fail-closed verification, resource
  metadata and `ailang mcp check`. The annotation and CLI tables list `@mcp_auth`,
  `@mcp_token_verifier`, `@mcp_secret`, `@mcp_agent_only` and `--oauth-issuer`. The devtools
  prompts (and their embedded copies) carry the annotation reference.
- New example `examples/runnable/serve_api_mcp_oauth.ail`: an open tool, a gated tool, a verifier
  and an agent-only tool. Served with `--oauth-issuer`, it passes `ailang mcp check` checks 1–4.
  Check 5 needs a real authorization server. Writing it showed that an exported `main` becomes an
  unannotated MCP tool unless it is marked `@nomcp`, and the example says so.

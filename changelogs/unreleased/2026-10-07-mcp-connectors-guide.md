### Added — docs: "Publish an AILANG service as a Claude / ChatGPT connector" guide (2026-10-07)

New guide `docs/docs/guides/mcp-connectors.md`. It takes a `serve-api` service from an MCP endpoint
to a directory submission, with AILANG Parse (`https://docparse.ailang.sunholo.com/mcp/connect/`)
as the worked example. It covers:
- the two surfaces, `/mcp/` and `/mcp/connect/`, with the vendor rules behind them;
- the listing annotations (`@mcp_title`, `@mcp_hints`, `@mcp_auth`, `@mcp_secret`, `@optional`,
  `@mcp_token_verifier`, `@mcp_agent_only`, `@nomcp`) and lazy auth;
- OAuth through `sunholo/mcp_oauth`: the hooks, the four routes, a same-origin sign-in page, the
  anti-framing guard (#1597) and the security properties;
- `ailang mcp check` and how to test in claude.ai and ChatGPT;
- file input: `@mcp_file` and MCP Apps widgets (v0.52.4), with `sunholo/mcp_files` marked in
  progress;
- a submission checklist for both directories.

Every AILANG snippet is extracted from the page and checked with `ailang check`. The package
snippets are checked against `sunholo/mcp_oauth` 0.1.1. Linked from the Serve API guide, which no
longer calls the package "planned".

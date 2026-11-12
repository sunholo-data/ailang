### Added
- **The hosted executor and the docs MCP are listed in the official MCP Registry** (which feeds VS
  Code and GitHub Copilot): `io.github.sunholo-data/ailang-executor`
  (`https://executor.ailang.sunholo.com/mcp/connect/`) and `io.github.sunholo-data/ailang-docs`
  (`https://mcp.ailang.sunholo.com/mcp/`), from `registry/*/server.json`. A new
  `publish-mcp-registry.yml` publishes both after each release (a `release.yml` job) or by hand
  (`gh workflow run publish-mcp-registry.yml -f version=0.60.2`), with GitHub OIDC, the release
  version stamped in, a check that the remote lists its tools first, and `mcp-publisher` 1.7.9 pinned
  by checksum. AILANG Parse was already listed as `io.github.sunholo-data/parse`.

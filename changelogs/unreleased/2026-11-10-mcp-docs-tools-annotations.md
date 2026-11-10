### Changed
- **The 22 docs MCP tools (`mcp_tools/`) carry a title and `readOnly` + `idempotent` hints.** They
  served without either, so `ailang mcp check` failed 44 annotation checks on the executor's listed
  surface (`executor.ailang.sunholo.com/mcp/connect/`) and on `mcp.ailang.sunholo.com`, and
  Anthropic's directory requires both on every tool. All of them only read the baked docs snapshot.
- **`mcp.ailang.sunholo.com` identifies itself as `ailang-docs` / "AILANG Docs"** in MCP
  `serverInfo` (`docker/Dockerfile.mcp` passes `--server-info`; the version stays the binary's,
  stamped from `std/VERSION`). It answered `ailang-api`, which `mcp check` warns is unbranded.

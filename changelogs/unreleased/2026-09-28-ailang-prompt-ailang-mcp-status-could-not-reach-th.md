### Fixed — `ailang prompt` / `ailang mcp status` could not reach the prod MCP server (2026-09-28)

Since the SDK moved to v1.8.0 (#1169, 2026-09-14), prod's stateless MCP server has not issued an
`Mcp-Session-Id`. `internal/mcp_client` refused to continue without one, so `ailang mcp status`
reported `reachable: false` against a server that answered 200, and every `ailang prompt
--source auto` fresh fetch silently fell back to the embedded copy. The client now:
- treats the session id as optional;
- asks for protocol `2025-06-18` and sends the negotiated `MCP-Protocol-Version` on every request
  after initialize;
- checks the HTTP status before anything else, so a 5xx is reported as a 5xx;
- accepts a plain-JSON reply as well as SSE.

A second, independent bug was found during the fix. The CLI sent `forVersion: "v0.47.1"`, but
MCP snapshots are keyed `0.47.1`, so a working handshake would still have got `unknown_version`.
`mcp_client.WireVersion` now drops the leading `v`. Dev builds (`v0.47.1-21-g…`) stay distinct and
fall back to embedded, as they should.

The package's first tests (`client_test.go`) cover stateless, stateful, JSON-reply,
wire-version, version-mismatch and 500-diagnosis servers. Three mutations each fail a test. Verified
live against prod: `reachable: true`, `server_knows_version: true`, deployed and embedded prompt SHAs
equal. Phase 1 of `design_docs/planned/v0_31_0/m-mcp-2026-07-28-adoption.md`.

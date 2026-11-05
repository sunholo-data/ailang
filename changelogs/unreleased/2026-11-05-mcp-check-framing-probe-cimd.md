### Fixed — `ailang mcp check` no longer WARNs falsely on every CIMD-only server (2026-11-05)

The consent-page framing probe sent the non-URL `client_id` `ailang-mcp-check-dummy`, which a CIMD
authorization server rejects with a 400 before any consent page loads, so every CIMD-only server
(AILANG Parse, the executor) got an `authorization-framing` WARN (design row V20). The probe now
picks its client from the AS metadata: with CIMD it presents
`https://ailang.sunholo.com/mcp-check/client-metadata.json` (served by the docs site from
`docs/static/mcp-check/client-metadata.json`, redirect `http://127.0.0.1/ailang-mcp-check/callback`);
without CIMD it uses the new `--probe-client-id` / `--probe-redirect-uri`; otherwise it is skipped,
never WARNed, and it never registers a client through DCR. Each run sends a fresh 43-character S256
challenge plus `resource=<checked URL>`, skips an AS without S256, and stops before following a
redirect to its own callback (reported as "redirected without a consent page"). Two WARN-only checks
join the run: `server-info` (an unbranded `ailang-api` serverInfo; set `serve-api --server-info`) and
`listed-resources` (`ailang://meta/modules` on the checked surface). M-MCP-OAUTH-CLIENT-MODELS M3c
(D7).

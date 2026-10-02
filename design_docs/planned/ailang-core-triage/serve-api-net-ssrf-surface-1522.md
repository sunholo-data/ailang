# SECURITY: serve-api Net reaches localhost + cloud metadata on every call incl. redirect hops (#1522)

- **Date**: 2026-10-02
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/planned/v0_52_0/m-mcp-oauth-package.md
- **Searched**: `1522|AllowMetadata|SSRF` across `design_docs/`; read `design_docs/planned/v0_52_0/m-mcp-oauth-package.md` (D5/D6, S0, V9, V17); verified code at `cmd/ailang/serve_api.go` (`effCtx.HasCap("Net")` block) and `internal/effects/net_authorize.go` (`resolvePinned`)
- **Estimate**: n/a (duplicate — no new work recommended here)

The report is accurate but the fix is already designed, ratified, and in flight. `design_docs/planned/v0_52_0/m-mcp-oauth-package.md` rules on exactly this: **S0** (line 271) is "#1522: redirect hops never reach loopback, link-local or private addresses"; **D5** (line 112) requires S0 before the `sunholo/mcp_oauth` package deploys, with both parts — redirect-hop validation plus a per-call public-only Net mode; **D6** (lines 118–119) freezes the surface as a `Net[scope=public]` effect mode. The doc's verification table independently confirms the report's mechanism: **V9** cites `cmd/ailang/serve_api.go:134-138` (the `if effCtx.HasCap("Net")` block setting `AllowHTTP`/`AllowLocalhost`/`AllowMetadata` — line numbers still hold on this tree) and **V17** confirms `internal/effects/net_authorize.go` `resolvePinned` (now at :232) validates every resolved IP and dials the pinned address, which is the connect-time hook where `destinationPolicy` will force `allowLocalhost`/`allowMetadata` off. Severity framing matches the report: not exploitable on prod Parse today (https-only fetch; GCP metadata needs the `Metadata-Flavor` header), a defence-in-depth gap. The directive itself notes the fix is underway in an attended session (sunholo/mcp_oauth work) and asks for triage/record only — no competing design doc opened, per that instruction.

**Action**: none from triage. Track implementation under S0/D5/D6 of the m-mcp-oauth-package doc; close this report against #1522 when that lands.

### Changed
- **The executor's sign-in service uses `sunholo/mcp_oauth` 0.2.0** (`executor_oauth/ailang.toml`
  and `ailang.lock`). It still calls only the 0.1 entry points, so its metadata and flows are
  unchanged (`executor_oauth/test_oauth.sh`: 27 passed). This makes pre-registered clients, DCR and
  revocation one `ClientConfig` away for Gemini Enterprise, Microsoft Copilot and Google Connected
  Apps; switching them on in a deployment waits on the P0 console measurements in
  `m-mcp-oauth-client-models.md`.

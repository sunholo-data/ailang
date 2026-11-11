### Fixed
- **Gemini Enterprise's web-app sign-in reaches the executor.** Its per-user Connect sends a
  768-character base64 `state` with `=` padding, which `sunholo/mcp_oauth` 0.2.0 refused
  (`invalid_request: state must be unreserved characters, at most 512`). The executor's sign-in
  service now uses 0.2.1 (`executor_oauth/ailang.toml`, lock), which accepts any RFC 6749 VSCHAR up
  to 2048 characters and percent-encodes `state` on the redirect. `executor_oauth/test_oauth.sh`
  (44 checks) sends a GE-shaped state and expects the sign-in page.

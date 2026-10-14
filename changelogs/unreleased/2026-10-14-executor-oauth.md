### Added — OAuth sign-in for the hosted executor's public MCP service (M-AILANG-EXECUTOR-GE P1b)

`executor_oauth/` is the executor's own authorization server, built on `sunholo/mcp_oauth` 0.1.2 (pinned
by `executor_oauth/ailang.lock`). Users sign in with Google (Firebase), limited to the
`EXECUTOR_ALLOWED_EMAIL_DOMAINS` allowlist. Access tokens (`ax_…`) last an hour and refresh in rotating
families. Firestore stores only a SHA-256 digest of each token, and TTL removes expired records. Every
authenticated tool call logs one `executor_auth` line naming the account, never the token. The sign-in page
is `executor_oauth/static/oauth-login.html`, served same-origin with `serve-api --static`.
`executor_oauth/test_oauth.sh` runs the full flow against a local `--mcp-connect-only` server (21 checks).

### Changed — the executor image serves two services

`docker/Dockerfile.executor-mcp` moves the per-service flags (`--caps` and the program dirs) from ENTRYPOINT
to CMD. The IAM-only Gemini Enterprise service keeps the CMD's command unchanged. The public OAuth service
replaces it with `--mcp-connect-only --oauth-issuer … executor_oauth/`. The image fetches the
`executor_oauth` packages at build time, runs as `HOME=/home/nonroot`, and sets `AILANG_TRACE_VALUES=off`
so traces never render request handles, codes or tokens.

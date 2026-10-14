### Added — hosted AILANG code executor over MCP (M-AILANG-EXECUTOR-GE P1)

`executor_tools/executor.ail` is an AILANG serve-api program that exposes `ailang_prompt`,
`ailang_check`, `ailang_run`, `ailang_eval` and `ailang_builtins`. The names match the local
stdio MCP in `ailang_bootstrap`, but each tool takes source code, not a file path. Untrusted code
runs in a child `ailang run --policy` (`docker/executor-policy.toml`: IO only, restricted mode,
60s supervised timeout, 64KB output) in its own request directory, and the caller never chooses
capabilities. `docker/Dockerfile.executor-mcp` serves the executor tools next to the public docs
tools and the built-in `submit_feedback`. The image is built by `cloudbuild-dev.yaml`
(`build-executor-mcp`), and `executor_tools/smoke.sh <mcp-url>` checks a deployment. No Go
changes.

The hosted service is IAM-only (no `allUsers`). Gemini Enterprise's custom MCP connector authenticates
as its discoveryengine service agent via `X-Serverless-Authorization`; `executor_tools/smoke.sh` takes
an identity token in `MCP_TOKEN`.

### Added — `serve-api --process-timeout` and `--process-allowlist`

These are the same flags `ailang run` has. Before this, a serve-api granted `Process` could exec any binary on PATH,
under a fixed 30s timeout. The executor image now allows only `ailang prompt|ai-check|run|builtins`, at 90s.
Either flag without `--caps Process` is an error.

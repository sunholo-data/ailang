### Added — pace checks on cloud codex and claude jobs, and an operator override

- Cloud codex and claude jobs now check their subscription's pace line before cloning, as agy
  jobs already did (`cmd/ailang/coordinator_cloud_ration.go`). Codex reads
  `codex app-server account/rateLimits/read` and refuses when over ration or unreadable.
  Claude reads `/api/oauth/usage` with the job's token; the cloud setup-token is refused by
  that endpoint (HTTP 403), so an unreadable claude bucket is admitted and logged
  `NOT ENFORCED` rather than stopping every cloud claude task. The Claude credit lane
  (`AILANG_AUTH_MODE=apikey`) is skipped.
- `ailang mission quota --allow <google|codex|anthropic|all> --for <≤24h> --reason <why>
  --project <plane> --yes` lets that plane's cloud jobs start over their pace line until the
  override expires; `--revoke` ends it early. Attended only (refused inside a mission
  iteration); every job it admits logs who granted it, why and until when.

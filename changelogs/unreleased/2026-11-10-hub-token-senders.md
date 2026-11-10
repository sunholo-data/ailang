### Changed — Hub-class ingest senders present AILANG_HUB_TOKEN (2026-11-10)

#2212 put the dashboard's hook, exec, coordinator-event and telemetry ingest routes behind
`AILANG_HUB_TOKEN` (rolled out `AILANG_INGEST_AUTH` warn → enforce; loopback binds exempt), but
none of their senders attached the token, so under `enforce` every write to a non-loopback
dashboard would have been refused. Each sender now sends `X-AILANG-Ingest-Token: <token>` when
`AILANG_HUB_TOKEN` is set, and no header when it is unset (the previous request, still accepted
by a loopback or warn-mode dashboard).

- Go senders go through one helper, `config.AttachHubToken(http.Header)`:
  `coordinator.HTTPBroadcaster` (`/api/coordinator/events`), the eval-suite
  `TelemetryReporter` and the compile-metrics `sendToHub` (both `/api/telemetry`). The header
  name, `config.IngestTokenHeader`, moved from `internal/server` to `internal/config` so core
  (`pipeline`) and tools (`eval_harness`) senders can name it without a layering violation; the
  ingest gate reads the same constant.
- `internal/executor/claude_telemetry.sh` (the cloud-job Claude Code hook: `/api/observatory/hooks`,
  `/api/exec/sessions`, `/api/exec/events`) and `scripts/hooks/coordinator_hook.sh`
  (`/api/hooks/claude`, default loopback, but its URL argument can point anywhere). The header is
  a bash array expanded with the `${a[@]+"${a[@]}"}` idiom, because bash 3.2 under `set -u`
  treats an empty array as unbound.
- The resident executor's `docker/resident/lib/observatory.mjs` (`post()`), mirroring how
  `otel.mjs` sends `AILANG_OTLP_INGEST_TOKEN`.
- Tests assert the header is present when set and absent when unset for every sender, including
  one that runs the embedded `claude_telemetry.sh` under `/bin/bash` against an `httptest` server.

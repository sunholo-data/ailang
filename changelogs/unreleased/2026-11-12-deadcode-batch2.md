### Changed
- **Dead code, batch 2 (test-only platform code): 67 functions and 2,651 lines deleted.** Every
  deleted function was unreachable under `deadcode -test=false ./...` (every `main` in the module
  a root, `tools/` included) and reachable only from its own tests, or from nothing. Each one was
  also checked with `-whylive`, a `git grep -w` for the name (code, scripts, docs, string
  registries), `git log -S` for recently removed callers, and a `GOOS=js GOARCH=wasm` deadcode
  run, because `cmd/wasm` compiles `internal/repl` in. Tests that exercised only the deleted code
  went with it. Packages:
  `internal/server/auth` (21; FOLLOWUPS-s5 F10's five `AccessControlCache` writers and the five
  `WorkspaceService` methods batch 1 kept),
  `internal/messaging` (23; the polling `Client`, retired with `cmd/ailang-agent` in January),
  `internal/coordinator` (9; the never-wired config-driven output-marker parser),
  `internal/repl` (5), `internal/testing` (5), `internal/eval_harness` (3; `TelemetryReporter`,
  never constructed since 2026-09-14) and `internal/observatory` (1). The language core and
  codegen are untouched.
- **`internal/server/auth`: the per-user workspace access cache is gone.** It had one writer and
  one reader, both dead, but the live `GrantAccess`, `RevokeAccess` and `SetPublic` still cleared
  it. The cache, its entry type and those three invalidation blocks were removed together. With
  `GetWorkspaceByPath` deleted, nothing read `WorkspaceService.config`, so `NewWorkspaceService`
  takes only the Firestore client, and its six callers no longer load the workspaces config section
  just to pass it in. The workspace-metadata cache that `GetWorkspace` fills is unchanged.
- **`TelemetryReporter` was not a live `/api/telemetry` sender.** The 2026-11-10 hub-token
  fragment lists it as one, but nothing had constructed it since September. The pipeline's
  `MetricsCollector.sendToHub` is the only Go sender to that route.
- **The observatory aggregate-recalculation test now drives the live method.** The deprecated
  package-level `RecalculateTaskAggregates` is deleted. Its test, the only one covering that SQL,
  now calls `backend.RecalculateTaskAggregates`.

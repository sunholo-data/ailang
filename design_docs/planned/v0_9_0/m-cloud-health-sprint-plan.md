# Sprint Plan: M-CLOUD-HEALTH — Cloud Run HTTP Health & Status Endpoints

## Summary

Unblock Cloud Run deployment by adding HTTP health endpoints to the coordinator daemon, making `ailang serve` respect the `PORT` env var, adding coordinator status API endpoints, and supporting `AILANG_CONFIG` for config path override. All four components are narrowly scoped infrastructure changes with no language-level impact.

**Duration:** 1 day (single session, ~6 hours implementation)
**Dependencies:** None (infrastructure is already deployed in ailang-multivac)
**Risk Level:** Low — all changes are additive, no existing behavior modified

## Current Status Analysis

### Completed Recently
- M-HTTP-HOOKS-CLOUD-TELEMETRY: ~295 LOC impl + ~760 LOC tests (HTTP handler pattern)
- M-PERF-OBSERVATORY: Dashboard stability + aggregation queries
- M-AUDIT-OBSERVATORY: Dashboard data quality fixes
- Cloud infrastructure (ailang-multivac): Pub/Sub, Firestore, IAM, Secrets, AR, GCS all deployed

### Velocity
- Recent average: ~400-600 LOC/day (implementation + tests)
- This sprint total: ~320 LOC (well within single-day capacity)
- Low risk: reuses existing store interfaces, no new abstractions

### Remaining from Design Doc
- M1: AILANG_CONFIG env var (~5 LOC impl + ~30 LOC tests)
- M2: Dashboard PORT + bind (~15 LOC impl + ~20 LOC tests)
- M3: Coordinator health server (~25 LOC impl + ~40 LOC tests)
- M4: Coordinator status API (~125 LOC impl + ~80 LOC tests)

## Proposed Milestones

### Milestone 1: AILANG_CONFIG env var support
**Goal:** `LoadCoordinatorConfig()` and `getFirebaseProjectFromConfig()` check `AILANG_CONFIG` env var before falling back to `~/.ailang/config.yaml`
**Estimated:** 5 LOC implementation + 30 LOC tests = ~35 LOC
**Duration:** 30 minutes

**Tasks:**
1. Edit `internal/coordinator/agent_config.go:LoadCoordinatorConfig()` — add `os.Getenv("AILANG_CONFIG")` check before default path
2. Edit `cmd/ailang/server.go:getFirebaseProjectFromConfig()` — same pattern
3. Write unit tests: AILANG_CONFIG set overrides default; AILANG_CONFIG unset uses default; AILANG_CONFIG points to nonexistent file returns defaults gracefully

**Acceptance Criteria:**
- [ ] `AILANG_CONFIG=/tmp/test.yaml ailang coordinator start` loads from `/tmp/test.yaml`
- [ ] `ailang coordinator start` (no env var) loads from `~/.ailang/config.yaml` as before
- [ ] Missing file at AILANG_CONFIG path returns default config (no crash)
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- None — purely additive, existing fallback path unchanged

**Files:**
- `internal/coordinator/agent_config.go` (modify `LoadCoordinatorConfig`)
- `internal/coordinator/agent_config_test.go` (add tests)
- `cmd/ailang/server.go` (modify `getFirebaseProjectFromConfig`)

---

### Milestone 2: Dashboard PORT env var + bind address
**Goal:** `ailang serve` reads `PORT` env var (Cloud Run convention), binds to `0.0.0.0` when PORT is set, adds `--bind` flag
**Estimated:** 15 LOC implementation + 20 LOC tests = ~35 LOC
**Duration:** 45 minutes

**Tasks:**
1. Edit `cmd/ailang/server.go:serverCommand()` — add PORT env var check before flag parsing, set `bindAddr` to `0.0.0.0` when PORT present
2. Add `--bind` flag to flag parsing loop
3. Change `httpAddr` construction from `fmt.Sprintf("localhost:%s", port)` to `fmt.Sprintf("%s:%s", bindAddr, port)`
4. Update help text to document PORT env var and `--bind` flag
5. Test: verify precedence order (--port > PORT env > default 1957)

**Acceptance Criteria:**
- [ ] `PORT=8080 ailang serve` binds to `0.0.0.0:8080`
- [ ] `ailang serve` (no PORT) binds to `localhost:1957` (no regression)
- [ ] `ailang serve --port 3000` overrides PORT env var
- [ ] `ailang serve --bind 127.0.0.1` overrides auto-detection
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Binding to `0.0.0.0` locally could expose server to network — mitigated by only doing it when PORT env var is set (which doesn't happen locally)

**Files:**
- `cmd/ailang/server.go` (modify `serverCommand`)

---

### Milestone 3: Coordinator health HTTP server
**Goal:** Coordinator daemon starts an HTTP server on PORT env var with `/health` endpoint. This is the minimum needed for Cloud Run startup probes.
**Estimated:** 25 LOC implementation + 40 LOC tests = ~65 LOC
**Duration:** 45 minutes

**Tasks:**
1. Create `internal/coordinator/daemon_http.go` with `startHealthServer(port string)` method and `handleHealth()` handler
2. Add `writeJSON()` helper function
3. Edit `internal/coordinator/daemon.go:Run()` — add PORT env var check, call `go d.startHealthServer(port)` before main select loop
4. Write tests using `httptest.NewServer` pattern: health returns 200 JSON, no server when PORT unset

**Acceptance Criteria:**
- [ ] `PORT=8080 ailang coordinator start` starts health server on `0.0.0.0:8080`
- [ ] `GET /health` returns `{"status":"ok","component":"coordinator","uptime":"..."}`
- [ ] `ailang coordinator start` (no PORT) starts no health server
- [ ] Daemon main loop continues working normally alongside health server
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Health server goroutine panic could affect daemon — mitigate with defer/recover in goroutine

**Files:**
- `internal/coordinator/daemon_http.go` (new)
- `internal/coordinator/daemon_http_test.go` (new)
- `internal/coordinator/daemon.go` (add 3 lines to `Run()`)

---

### Milestone 4: Coordinator status API endpoints
**Goal:** Add `/status`, `/chains/active`, `/chains/stats`, `/pending` endpoints to the coordinator's HTTP server, reusing existing store interfaces
**Estimated:** 100 LOC implementation + 80 LOC tests = ~180 LOC
**Duration:** 1.5 hours

**Tasks:**
1. Add `handleStatus()` — calls `d.Status()` (existing method from `daemon_lifecycle.go`), returns JSON
2. Add `handleChainsActive()` — calls `d.obsBackend.ListChains(ctx, ChainListOptions{Status: "active"})`, returns JSON. Returns `[]` if `obsBackend` is nil
3. Add `handleChainsStats()` — calls `d.obsBackend.GetChainStatusCounts(ctx, createdAfter)` with `?hours=` query param. Returns `{}` if `obsBackend` is nil
4. Add `handlePending()` — calls `d.obsBackend.ListPendingApprovals(ctx, limit)`. Returns `[]` if `obsBackend` is nil
5. Register all handlers in `startHealthServer()` mux
6. Write tests with nil stores (graceful degradation) and mock stores (real data)

**Important implementation notes:**
- The daemon field is `d.obsBackend` (type `observatory.Backend`), NOT `d.observatoryStore`
- The coordinator `Store` interface has `GetTaskStats(ctx)` for task counts — use for `/status`
- Observatory `Backend.ListChains()` takes `(ctx, ChainListOptions)` not a simple filter
- Observatory `Backend.GetChainStatusCounts()` takes `(ctx, *time.Time)` not int hours — convert hours to time
- Observatory `Backend.ListPendingApprovals()` takes `(ctx, int)` for limit
- All store methods take `context.Context` — use `r.Context()` from HTTP request

**Acceptance Criteria:**
- [ ] `GET /status` returns coordinator state with task counts, uptime, cost
- [ ] `GET /chains/active` returns currently running execution chains
- [ ] `GET /chains/stats?hours=168` returns aggregate chain metrics
- [ ] `GET /pending` returns pending approval requests
- [ ] All endpoints return empty results (not errors) when stores are nil
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Store query timeouts in cloud (Firestore vs SQLite) — mitigate with `context.WithTimeout` on each handler (3s)

**Files:**
- `internal/coordinator/daemon_http.go` (extend with 4 handlers)
- `internal/coordinator/daemon_http_test.go` (extend with handler tests)

---

### Milestone 5: Integration verification + lint + commit
**Goal:** Verify all changes work together locally and in simulated cloud mode. Run full test suite and linting.
**Estimated:** 0 LOC (verification only)
**Duration:** 30 minutes

**Tasks:**
1. Run `make test` — all existing tests pass
2. Run `make lint` — no new warnings
3. Local smoke test: `ailang serve` binds to `localhost:1957` (unchanged)
4. Local smoke test: `ailang coordinator start` has no health server (unchanged)
5. Cloud simulation: `PORT=8080 AILANG_CONFIG=/tmp/test-config.yaml ailang coordinator start`
6. Cloud simulation: `PORT=8080 ailang serve`
7. Verify `curl localhost:8080/health` returns 200 on both
8. Verify `curl localhost:8080/status` returns coordinator state

**Acceptance Criteria:**
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Local workflows unchanged (no regression)
- [ ] Cloud simulation shows both services healthy
- [ ] All 5 coordinator endpoints respond correctly

---

## Success Metrics

- All existing tests passing: must pass
- New test coverage: ~170 LOC of new tests
- Total new code: ~320 LOC (implementation + tests)
- Linting clean: must pass
- No Terraform changes needed in ailang-multivac

## Dependencies

- None — all infrastructure is already deployed in ailang-multivac
- Observatory Backend interface already has all needed methods
- Coordinator Store interface already has `GetTaskStats()`

## Open Questions

None — the design doc is well-specified and all interfaces are verified.

## Notes

- The coordinator Cloud Run service is internal-only (no public IAM binding) so auth on the status endpoints is not required — Cloud Run ingress handles it
- The `--cloud` flag passed by Terraform (`coordinator start --cloud`) is silently ignored — not worth implementing since env vars (`PORT`, `COORDINATOR_MODE`) already distinguish cloud mode
- This sprint unblocks M-PUBSUB cloud messaging which is the next P0 feature

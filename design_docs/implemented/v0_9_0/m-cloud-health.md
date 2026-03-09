# M-CLOUD-HEALTH: Cloud Run HTTP Health & Status Endpoints

**Status**: Implemented
**Target**: v0.9.0
**Priority**: P0 (High) — blocks all Cloud Run deployments
**Estimated**: 1.5 days (8-10 hours implementation + testing)
**Dependencies**: M-PUBSUB (co-located in v0.9.0), M-CLOUD-INFRA (reference)
**Author**: Claude + Mark
**Created**: 2026-03-06
**Source**: Message `f80a9ee9` from ailang-multivac

---

## Executive Summary

The ailang-multivac cloud infrastructure is fully deployed (Pub/Sub, Firestore, IAM, Secrets, Artifact Registry, GCS config all working), but both Cloud Run Services fail startup probes because they don't open HTTP ports. This design doc covers four changes to unblock Cloud Run deployment and provide operational visibility:

1. **Coordinator health**: Add a concurrent HTTP server alongside the daemon loop with `/health`
2. **Coordinator status API**: Expose key operational endpoints (`/status`, `/chains/active`, `/chains/stats`, `/pending`) so the coordinator is independently queryable in cloud without needing the dashboard
3. **Dashboard**: Make `ailang serve` respect the `PORT` env var and bind to `0.0.0.0`
4. **Config**: Make `LoadCoordinatorConfig()` check `AILANG_CONFIG` env var

**Key outcome**: Both Cloud Run Services pass startup probes, and the coordinator exposes a lightweight status API for monitoring and operational tooling.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a cloud infrastructure change, not a language feature. Most axioms are neutral.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | +1 | Config path is explicit via env var, not ambient |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | Health server is read-only, no shared mutable state |
| A7: Machines First | +1 | Enables 24/7 cloud execution for AI agents |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Health endpoint composes with Cloud Run probes, load balancers |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | +1 | Cloud Run container boundary is explicit (PORT, AILANG_CONFIG) |

**Net Score: +4** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly enables machine-driven execution

---

## Problem Statement

The ailang-multivac dev environment is fully deployed on GCP (europe-west1):

- Pub/Sub topics and subscriptions: working
- Firestore database: working
- IAM service accounts: working
- Secret Manager (API keys): working
- Artifact Registry (container images): working
- GCS config bucket: working
- Cloud Run Job (agent executor): working

**However, both Cloud Run Services fail startup probes:**

```
Coordinator (google_cloud_run_v2_service.coordinator):
  startup_probe: tcp_socket { port = 8080 }
  Problem: Coordinator daemon doesn't listen on ANY port
  Result: UNHEALTHY -> container killed after 5 failures

Dashboard (google_cloud_run_v2_service.dashboard):
  startup_probe: tcp_socket { port = 1957 }
  Problem: `ailang serve` binds to localhost:1957, not 0.0.0.0:1957
  Result: UNHEALTHY -> container killed after 5 failures
```

**Current State:**

| Component | Listens on | Cloud Run expects | Status |
|-----------|-----------|-------------------|--------|
| Coordinator | Nothing (pure daemon) | `0.0.0.0:8080` | BLOCKED |
| Dashboard | `localhost:1957` | `0.0.0.0:1957` | BLOCKED |
| Agent (Job) | N/A (ephemeral) | N/A | Working |

**Additional issue:** `LoadCoordinatorConfig()` hard-codes `~/.ailang/config.yaml` but Cloud Run mounts config at `/etc/ailang-config/config.yaml` via GCS volume. The `AILANG_CONFIG` env var is set in Terraform but never read by the Go code.

**Impact:**
- Cloud Run deployment is 100% blocked
- All downstream features (M-PUBSUB cloud mode, 24/7 agents) are blocked
- Infrastructure cost is being incurred with no running services

---

## Goals

**Primary Goal:** Make both Cloud Run Services pass startup probes and provide operational visibility into the coordinator.

**Success Metrics:**
- Coordinator Cloud Run Service reaches "Running" status
- Dashboard Cloud Run Service reaches "Running" status
- `GET /health` returns 200 on both services
- Coordinator exposes `/status`, `/chains/active`, `/chains/stats`, `/pending` for monitoring
- Config loaded from `AILANG_CONFIG` path in Cloud Run
- Zero regression on local development workflow (`ailang serve`, `ailang coordinator start`)

---

## Solution Design

### Overview

Three independent, minimal changes. No architectural rework needed.

### Component 1: Coordinator Health Server

**Problem:** The daemon's `Run()` method is an infinite `select` loop with no HTTP listener.

**Solution:** Start a minimal HTTP server in a goroutine before entering the main loop.

```go
// internal/coordinator/daemon.go — inside Run(), before the select loop

func (d *Daemon) Run() error {
    // ... existing initialization ...

    // Start health server if PORT is set (Cloud Run) or --cloud flag passed
    if port := os.Getenv("PORT"); port != "" {
        go d.startHealthServer(port)
    }

    // ... existing select loop ...
}

func (d *Daemon) startHealthServer(port string) {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", d.handleHealth)
    mux.HandleFunc("/status", d.handleStatus)
    mux.HandleFunc("/chains/active", d.handleChainsActive)
    mux.HandleFunc("/chains/stats", d.handleChainsStats)
    mux.HandleFunc("/pending", d.handlePending)

    addr := "0.0.0.0:" + port
    d.logger.Printf("Health server listening on %s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        d.logger.Printf("Health server error: %v", err)
    }
}
```

**Design decisions:**
- Only starts when `PORT` env var is set (Cloud Run convention) — no change for local usage
- Binds to `0.0.0.0` (required for Cloud Run, which proxies from outside the container)
- Goroutine — doesn't block the daemon loop
- No graceful shutdown needed (Cloud Run sends SIGTERM to whole container)

### Component 1b: Coordinator Status API

**Problem:** The coordinator and dashboard are separate Cloud Run services. Without its own status endpoints, monitoring the coordinator requires the dashboard to be running and able to reach coordinator.db — which doesn't exist in cloud (Firestore). The coordinator needs to be independently queryable.

**Rationale:** Locally, all status info flows through `ailang coordinator status` (CLI) and `/api/coordinator/status` (dashboard server). In cloud, the coordinator should self-report its own state via HTTP, similar to how Kubernetes sidecars expose `/metrics`.

**Endpoints to expose on the coordinator's HTTP server:**

| Endpoint | Purpose | Data Source | Equivalent Local Command |
|----------|---------|-------------|--------------------------|
| `GET /health` | Liveness probe | In-memory | N/A (new) |
| `GET /status` | Coordinator state + task counts | coordinator store | `ailang coordinator status --json` |
| `GET /chains/active` | Currently running chains | observatory store | `ailang chains active --json` |
| `GET /chains/stats` | Aggregate chain metrics | observatory store | `ailang chains stats --json` |
| `GET /pending` | Pending approvals | coordinator store | `ailang coordinator pending --json` |

**Implementation — each handler reuses existing daemon internals:**

```go
// internal/coordinator/daemon_http.go (NEW FILE ~150 LOC)

func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, map[string]any{
        "status":    "ok",
        "component": "coordinator",
        "uptime":    time.Since(d.startedAt).Truncate(time.Second).String(),
    })
}

func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
    // Reuses the same Status() method that `ailang coordinator status` uses
    status, err := d.Status()
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    writeJSON(w, status)
}

func (d *Daemon) handleChainsActive(w http.ResponseWriter, r *http.Request) {
    if d.observatoryStore == nil {
        writeJSON(w, []any{}) // No observatory in this mode
        return
    }
    chains, err := d.observatoryStore.ListChains(observatory.ChainFilter{
        Status: observatory.ChainStatusActive,
    })
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    writeJSON(w, chains)
}

func (d *Daemon) handleChainsStats(w http.ResponseWriter, r *http.Request) {
    if d.observatoryStore == nil {
        writeJSON(w, map[string]any{})
        return
    }
    hours := 168 // Default: last 7 days
    if h := r.URL.Query().Get("hours"); h != "" {
        hours, _ = strconv.Atoi(h)
    }
    stats, err := d.observatoryStore.GetChainStatusCounts(hours)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    writeJSON(w, stats)
}

func (d *Daemon) handlePending(w http.ResponseWriter, r *http.Request) {
    if d.taskStore == nil {
        writeJSON(w, []any{})
        return
    }
    approvals, err := d.taskStore.ListPendingApprovals()
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    writeJSON(w, approvals)
}

func writeJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}
```

**Key design points:**
- **Reuses existing store interfaces** — `d.taskStore` (coordinator.db/Firestore) and `d.observatoryStore` (observatory.db/Firestore) are already initialized in `initTaskProcessing()`
- **Graceful degradation** — if a store isn't initialized (standby mode), returns empty arrays, not errors
- **Read-only** — all endpoints are GET, no mutations
- **No auth on coordinator** — the Cloud Run service is internal-only (no public IAM binding in Terraform). Auth is handled at the Cloud Run ingress layer.
- **Separate file** — `daemon_http.go` keeps HTTP handlers out of the main daemon loop code

### Component 2: Dashboard PORT env var + bind address

**Problem:** `cmd/ailang/server.go` hard-codes `port := "1957"` and constructs `httpAddr := fmt.Sprintf("localhost:%s", port)`. In Cloud Run, `PORT` is set by the platform and the server must bind to `0.0.0.0`.

**Solution:** Check `PORT` env var as fallback, and bind to `0.0.0.0` when not running locally.

```go
// cmd/ailang/server.go — in serverCommand()

// Default values
port := "1957"
bindAddr := "localhost" // Safe default for local development

// Check PORT env var (Cloud Run convention) — lowest priority, overridden by --port flag
if envPort := os.Getenv("PORT"); envPort != "" {
    port = envPort
    bindAddr = "0.0.0.0" // Cloud Run requires 0.0.0.0
}

// Parse flags (--port overrides PORT env var)
for i := 0; i < len(args); i++ {
    switch args[i] {
    case "--port":
        if i+1 < len(args) {
            port = args[i+1]
            i++
        }
    case "--bind":
        if i+1 < len(args) {
            bindAddr = args[i+1]
            i++
        }
    // ... existing flags ...
    }
}

httpAddr := fmt.Sprintf("%s:%s", bindAddr, port)
```

**Priority order:**
1. `--port` flag (explicit, highest priority)
2. `PORT` env var (Cloud Run convention)
3. Default `1957` (local development)

**Bind address:**
- `PORT` env var present -> `0.0.0.0` (Cloud Run requires this)
- Otherwise -> `localhost` (secure default for local dev)
- `--bind` flag available for explicit override

### Component 3: AILANG_CONFIG env var

**Problem:** `LoadCoordinatorConfig()` hard-codes `~/.ailang/config.yaml`.

**Solution:** Check `AILANG_CONFIG` env var first.

```go
// internal/coordinator/agent_config.go

func LoadCoordinatorConfig() (*CoordinatorConfig, error) {
    // Check AILANG_CONFIG env var first (Cloud Run sets this)
    if configPath := os.Getenv("AILANG_CONFIG"); configPath != "" {
        return LoadCoordinatorConfigFrom(configPath)
    }

    // Fall back to default path
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return DefaultCoordinatorConfig(), nil
    }

    configPath := filepath.Join(homeDir, ".ailang", "config.yaml")
    return LoadCoordinatorConfigFrom(configPath)
}
```

**Also update:** `getFirebaseProjectFromConfig()` in `cmd/ailang/server.go` to use the same pattern (check `AILANG_CONFIG` first).

---

### Implementation Plan

**Phase 1: Config path** (~30 min)
- [ ] Add `AILANG_CONFIG` check to `LoadCoordinatorConfig()` in `internal/coordinator/agent_config.go`
- [ ] Add `AILANG_CONFIG` check to `getFirebaseProjectFromConfig()` in `cmd/ailang/server.go`
- [ ] Unit test: `AILANG_CONFIG` overrides default path
- [ ] Unit test: unset `AILANG_CONFIG` uses `~/.ailang/config.yaml`

**Phase 2: Dashboard PORT + bind** (~1 hour)
- [ ] Add `PORT` env var check in `cmd/ailang/server.go:serverCommand()`
- [ ] Change `httpAddr` to use `bindAddr` variable (default `localhost`, `0.0.0.0` when PORT set)
- [ ] Add `--bind` flag for explicit override
- [ ] Update help text to document PORT and --bind
- [ ] Test: `PORT=8080 ailang serve` binds to `0.0.0.0:8080`
- [ ] Test: `ailang serve` still binds to `localhost:1957` (no regression)
- [ ] Test: `ailang serve --port 3000` overrides PORT env var

**Phase 3: Coordinator health server** (~1 hour)
- [ ] Create `internal/coordinator/daemon_http.go` with `startHealthServer()` and `handleHealth()`
- [ ] Call from `Run()` when `PORT` env var is set
- [ ] Return JSON: `{"status":"ok","component":"coordinator","uptime":"5m30s"}`
- [ ] Test: health server responds 200
- [ ] Test: daemon loop still works normally alongside health server
- [ ] Test: no health server started when PORT is unset (local mode)

**Phase 3b: Coordinator status API** (~2 hours)
- [ ] Add `handleStatus()` — reuse `d.Status()` from daemon_lifecycle.go
- [ ] Add `handleChainsActive()` — query `d.observatoryStore.ListChains(filter)`
- [ ] Add `handleChainsStats()` — query `d.observatoryStore.GetChainStatusCounts(hours)`
- [ ] Add `handlePending()` — query `d.taskStore.ListPendingApprovals()`
- [ ] Add `writeJSON()` helper
- [ ] Graceful degradation: return empty results when stores not initialized
- [ ] Test: each endpoint returns correct JSON structure
- [ ] Test: endpoints work when observatory store is nil (standby mode)

**Phase 4: Integration verification** (~1 hour)
- [ ] Local: `ailang serve` works unchanged
- [ ] Local: `ailang coordinator start` works unchanged
- [ ] Cloud simulation: `PORT=8080 ailang serve --bind 0.0.0.0`
- [ ] Cloud simulation: `PORT=8080 AILANG_CONFIG=/tmp/test-config.yaml ailang coordinator start`
- [ ] Verify Terraform `cloud_run.tf` doesn't need changes (it shouldn't)

### Files to Modify

**New files:**
- `internal/coordinator/daemon_http.go` — HTTP server startup + all status handlers (~150 LOC)
- `internal/coordinator/daemon_http_test.go` — Handler tests with mock stores (~120 LOC)

**Modified files:**
- `internal/coordinator/agent_config.go` — Add AILANG_CONFIG env var check (~5 LOC)
- `internal/coordinator/daemon.go` — Call `startHealthServer()` from `Run()` (~3 LOC)
- `cmd/ailang/server.go` — PORT env var, bindAddr logic, --bind flag (~15 LOC)

**Test files:**
- `internal/coordinator/agent_config_test.go` — AILANG_CONFIG tests (~30 LOC)

---

## Examples

### Example 1: Coordinator in Cloud Run

**Before (fails):**
```
$ docker run coordinator:latest coordinator start
Daemon running, polling for tasks...
# Cloud Run: TCP probe on :8080 -> REFUSED
# Cloud Run: Container killed after 50s
```

**After (works):**
```
$ PORT=8080 AILANG_CONFIG=/etc/ailang-config/config.yaml docker run coordinator:latest coordinator start
Health server listening on 0.0.0.0:8080
Daemon running, polling for tasks...
# Cloud Run: TCP probe on :8080 -> OK
# Cloud Run: GET /health -> 200 {"status":"ok","component":"coordinator","uptime":"5s"}
```

### Example 2: Querying coordinator status from cloud

```bash
# Health check (liveness)
$ curl https://coordinator-xyz.run.app/health
{"status":"ok","component":"coordinator","uptime":"2h15m30s"}

# Full coordinator status (task counts, cost)
$ curl https://coordinator-xyz.run.app/status
{
  "running": true,
  "pid": 1,
  "started_at": "2026-03-06T12:00:00Z",
  "uptime": "2h15m30s",
  "tasks_run": 47,
  "pending_tasks": 2,
  "running_tasks": 1,
  "pending_approvals": 3,
  "failed_tasks": 1,
  "total_cost": 12.45,
  "total_tokens": 1250000
}

# Active execution chains
$ curl https://coordinator-xyz.run.app/chains/active
[
  {
    "id": "e9c7501d-...",
    "source_type": "github_issue",
    "source_ref": "sunholo-data/ailang#142",
    "status": "active",
    "current_stage": 2,
    "total_cost": 3.21
  }
]

# Aggregate stats (last 7 days)
$ curl https://coordinator-xyz.run.app/chains/stats?hours=168
{
  "total_chains": 23,
  "completed": 18,
  "active": 2,
  "pending_approval": 2,
  "failed": 1,
  "total_cost": 45.67,
  "total_tokens": 4500000
}

# Pending approvals
$ curl https://coordinator-xyz.run.app/pending
[
  {
    "id": "apr-123",
    "task_id": "task-29404032",
    "type": "merge_handoff",
    "description": "Design doc for M-CLOUD-HEALTH",
    "status": "pending",
    "created_at": "2026-03-06T14:30:00Z"
  }
]
```

### Example 3: Dashboard in Cloud Run

**Before (fails):**
```
$ PORT=1957 docker run dashboard:latest ailang serve --port 1957
Starting HTTP server on localhost:1957
# Cloud Run: TCP probe on :1957 from external -> REFUSED (localhost only!)
```

**After (works):**
```
$ PORT=1957 docker run dashboard:latest ailang serve
Starting HTTP server on 0.0.0.0:1957
# Cloud Run: TCP probe on :1957 -> OK
# Cloud Run: GET /health -> 200
```

### Example 4: Local development (no regression)

```
$ ailang serve
Starting HTTP server on localhost:1957
# Same as before — no PORT env var means localhost binding

$ ailang coordinator start
Daemon running, polling for tasks...
# Same as before — no health server started (PORT not set)
```

---

## Success Criteria

- [ ] Coordinator Cloud Run Service reaches "Running" status with `GET /health` returning 200
- [ ] Dashboard Cloud Run Service reaches "Running" status with `GET /health` returning 200
- [ ] Coordinator `GET /status` returns task counts, uptime, running state
- [ ] Coordinator `GET /chains/active` returns currently running chains
- [ ] Coordinator `GET /chains/stats` returns aggregate metrics (costs, tokens, chain counts)
- [ ] Coordinator `GET /pending` returns pending approvals
- [ ] All status endpoints return empty results gracefully when stores not initialized
- [ ] `AILANG_CONFIG=/path/to/config.yaml` correctly overrides default config path
- [ ] `ailang serve` (no PORT) still binds to `localhost:1957` (no regression)
- [ ] `ailang coordinator start` (no PORT) starts no health server (no regression)
- [ ] All existing tests passing
- [ ] No Terraform changes needed in ailang-multivac

---

## Testing Strategy

**Unit tests:**
- `LoadCoordinatorConfig()` reads `AILANG_CONFIG` env var
- `startHealthServer()` returns 200 JSON on `/health`
- `serverCommand()` respects PORT env var and --bind flag

**Integration tests:**
- Start coordinator with PORT set, verify HTTP health
- Start server with PORT set, verify binding to 0.0.0.0

**Manual testing:**
- Deploy to ailang-multivac-dev Cloud Run
- Verify both services pass startup probes
- Verify dashboard UI is accessible
- Verify coordinator receives Pub/Sub messages

---

## Non-Goals

**Not in this feature:**
- Readiness/liveness probes — Startup probe is sufficient for now. Cloud Run doesn't require separate readiness probes for always-on services.
- Coordinator REST API — The coordinator is internal-only. Health endpoint is the minimum viable interface.
- `--cloud` flag for coordinator — Terraform passes `["coordinator", "start", "--cloud"]` but the flag isn't needed. `PORT` and `COORDINATOR_MODE` env vars already distinguish cloud mode. The unused flag is silently ignored (Go flag parsing).
- Dashboard HTTPS — Cloud Run terminates TLS at the load balancer. The container serves HTTP.

---

## Timeline

**Day 1** (~6-8 hours):
- Phase 1: Config path (30 min)
- Phase 2: Dashboard PORT (1 hour)
- Phase 3: Coordinator health server (1 hour)
- Phase 3b: Coordinator status API (2 hours)
- Phase 4: Integration verification (1.5 hours)

**Day 2** (~2 hours, if needed):
- Cloud Run deployment verification
- End-to-end testing against ailang-multivac-dev

**Total: ~8-10 hours across 1.5 days**

Phases 1-3 are independent and can be implemented in any order. Phase 3b depends on Phase 3 (health server must exist first).

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Health server goroutine crash takes down daemon | High | Recover from panic in goroutine; health server is independent |
| PORT env var accidentally set locally | Low | Only changes bind address, not behavior. `localhost` default is safe |
| Config file not found at AILANG_CONFIG path | Medium | `LoadCoordinatorConfigFrom` already handles missing files gracefully (returns defaults) |
| Dashboard --port flag conflicts with PORT env var | Low | --port flag takes precedence (explicit > implicit) |

---

## Related Documents

**Directly related (same deployment):**
- [design_docs/planned/v0_8_2/m-cloud-infra.md](design_docs/planned/v0_8_2/m-cloud-infra.md) — Original cloud architecture plan
- [design_docs/planned/v0_9_0/m-pubsub-messaging.md](design_docs/planned/v0_9_0/m-pubsub-messaging.md) — Pub/Sub transport (co-dependent)

**Infrastructure (ailang-multivac repo):**
- `terraform/cloud_run.tf` — Cloud Run service definitions (no changes needed)
- `docker/Dockerfile.dashboard` — Dashboard container build
- `config/config.cloud.yaml` — Cloud coordinator config (mounted via GCS)

---

## References

- [Cloud Run container contract](https://cloud.google.com/run/docs/container-contract) — PORT env var requirement
- [Cloud Run health checks](https://cloud.google.com/run/docs/configuring/healthchecks) — Startup probe configuration
- Message `f80a9ee9` from ailang-multivac — Original request

---

## Future Work

- **Coordinator metrics endpoint** (`/metrics`) — Prometheus-compatible metrics for Cloud Monitoring
- **Readiness probe** — Report "not ready" during initialization, "ready" after first successful poll
- **Dashboard WebSocket in Cloud Run** — May need Cloud Run session affinity for persistent WebSocket connections
- **`--cloud` flag** — Currently unused but passed by Terraform. Could be used to set sensible cloud defaults (bind 0.0.0.0, increase timeouts, etc.) as an alternative to relying purely on env vars

---

**Document created**: 2026-03-06
**Last updated**: 2026-03-06

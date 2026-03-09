# M-CLOUD-ENDPOINT-AUTH: Secure Coordinator Status Endpoints

**Status**: Implemented
**Target**: v0.9.0
**Priority**: P1 (High - blocks production deployment with scale-to-zero)
**Estimated**: 3 hours
**Dependencies**: M-CLOUD-WEBHOOK (implemented), M-AUTH (Firebase dashboard auth, implemented v0.7.0)

**Created**: March 8, 2026
**Bug Report**: msg_20260308_162454_4cf41c1e (from ailang-multivac)

## Axiom Compliance

This is infrastructure/security, not a language feature. Axiom scoring is N/A for most axioms.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A4: Explicit Authority | +1 | Enforces auth boundary on coordinator API |
| A12: System Boundary | +1 | Makes auth requirements explicit at endpoint level |
| All others | 0 | No language impact |

**Net score: +2** (accept)

## Problem Statement

M-CLOUD-WEBHOOK made the coordinator publicly accessible (`allUsers` `roles/run.invoker`) to allow GitHub webhook delivery. While the webhook and Pub/Sub endpoints are properly secured (HMAC-SHA256 and OIDC respectively), **5 status/observatory endpoints are now exposed without authentication**:

| Endpoint | Sensitivity | Data Exposed |
|----------|------------|-------------|
| `GET /health` | NONE | Uptime string only |
| `GET /chains/stats` | LOW | Aggregate counts (no task content) |
| `GET /status` | MEDIUM | Task counts, cost/token totals, pending approval count |
| `GET /chains/active` | HIGH | Active chains with agent IDs, workspaces, task directives |
| `GET /pending` | HIGH | Pending approvals with task content, repo names |

### Risk Assessment

- **Information disclosure**: Agent IDs, workspace paths, repo names, task directives leak internal project structure
- **Cost exposure**: Token counts and cost totals are business-sensitive
- **No mutation risk**: All exposed endpoints are read-only GET requests
- **Attack surface**: Low — no data modification possible, but information aids reconnaissance

## Options Evaluated

### Option A: API Key Middleware (Recommended)

Add a simple Bearer token check on status endpoints. Health stays public for Cloud Run probes.

**Implementation** (~30 LOC):
```go
// middleware.go
func (d *Daemon) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        key := os.Getenv("COORDINATOR_API_KEY")
        if key == "" {
            next(w, r) // No key configured = open (local mode)
            return
        }
        auth := r.Header.Get("Authorization")
        if auth != "Bearer "+key {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

**Registration**:
```go
mux.HandleFunc("/status", d.requireAPIKey(d.handleStatus))
mux.HandleFunc("/chains/active", d.requireAPIKey(d.handleChainsActive))
mux.HandleFunc("/chains/stats", d.requireAPIKey(d.handleChainsStats))
mux.HandleFunc("/pending", d.requireAPIKey(d.handlePending))
// /health stays public — Cloud Run needs it for probes
```

**Terraform**: Add `COORDINATOR_API_KEY` env var from Secret Manager.

| Pro | Con |
|-----|-----|
| 5 LOC middleware + 4 LOC registration | Static key rotation requires redeploy |
| Zero dependencies | No user-level granularity |
| Local mode unchanged (no key = open) | Dashboard needs the key in its config |
| Secret Manager rotation support | |

### Option B: Firebase Auth Token Validation

Validate Firebase ID tokens on status endpoints (reuse dashboard's auth).

| Pro | Con |
|-----|-----|
| Same auth as dashboard | Requires Firebase Admin SDK in coordinator |
| User-level identity | Adds ~2s cold start (SDK init) |
| Already implemented in dashboard | Coordinator doesn't serve UI — awkward fit |

### Option C: Cloud IAP (Identity-Aware Proxy)

Enable IAP on the coordinator Cloud Run service.

| Pro | Con |
|-----|-----|
| GCP-native, zero code changes | GitHub webhooks can't pass IAP auth |
| OAuth2-based, user-level | Would need webhook on separate service |
| Central policy management | Adds latency (~50-100ms per request) |

### Option D: Separate Webhook Service

Keep coordinator IAM-restricted. Add a tiny public Cloud Run service that only handles `/github/webhook` and forwards to the coordinator.

| Pro | Con |
|-----|-----|
| Coordinator stays internal | Extra service to build/deploy/maintain |
| Strongest isolation | Webhook → coordinator call adds latency |
| Clearest security boundary | More Terraform, more Docker images |

### Option E: Path-Based IAM (Not Available)

Cloud Run doesn't support path-based IAM rules — it's all-or-nothing per service.

## Recommendation: Option A (API Key)

**Why**: Minimal code change (~30 LOC), zero new dependencies, local mode unaffected, and the dashboard can include the key when fetching coordinator status. This is the right level of security for internal status endpoints — they're informational, read-only, and the primary threat is casual information disclosure, not targeted attacks.

**Migration path**: If we later need user-level auth, we can upgrade to Option B (Firebase tokens) without changing the endpoint structure — just swap the middleware.

## Implementation Plan

### Phase 1: API Key Middleware (~30 LOC)

1. Create `requireAPIKey` middleware in `daemon_http.go`
2. Wrap status endpoints (`/status`, `/chains/active`, `/chains/stats`, `/pending`)
3. Keep `/health` and `/github/webhook` unwrapped
4. No-op when `COORDINATOR_API_KEY` is unset (local mode stays open)

### Phase 2: Terraform (~10 LOC)

1. Add `coordinator_api_key` secret to `secrets.tf`
2. Add `COORDINATOR_API_KEY` env var to coordinator container in `cloud_run.tf`

### Phase 3: Dashboard Integration (~5 LOC)

1. Dashboard fetches coordinator status via API — add `Authorization: Bearer <key>` header
2. Key passed as env var to dashboard service

### Files to Modify

| File | Action | LOC |
|------|--------|-----|
| `internal/coordinator/daemon_http.go` | Add middleware, wrap endpoints | ~15 |
| `internal/coordinator/daemon_http_test.go` | Test middleware | ~40 |
| `ailang-multivac/terraform/secrets.tf` | Add coordinator_api_key secret | ~8 |
| `ailang-multivac/terraform/cloud_run.tf` | Add env var to both services | ~15 |
| `changelogs/v0.9-current.md` | Changelog entry | ~5 |
| **Total** | | **~83 LOC** |

## Success Criteria

- [ ] `GET /health` returns 200 without auth (Cloud Run probes work)
- [ ] `GET /status` returns 401 without valid Bearer token (when key configured)
- [ ] `GET /status` returns 200 with valid Bearer token
- [ ] `POST /github/webhook` still works without Bearer token (HMAC auth)
- [ ] `POST /pubsub/push` still works without Bearer token (OIDC auth)
- [ ] Local mode (`COORDINATOR_API_KEY` unset) works with no auth on any endpoint
- [ ] All existing tests pass

## Related Documents

- [M-AUTH: Firebase Authentication](../../implemented/v0_7_0/m-auth-dashboard-firebase.md) — Dashboard auth (reusable patterns)
- [M-CLOUD-WEBHOOK](../../../changelogs/v0.9-current.md) — Made coordinator public
- [M-CLOUD-HEALTH](m-cloud-health.md) — Created the status endpoints

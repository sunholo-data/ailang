# M-MCP-EDGE-THROTTLE: Per-IP rate limiting for the public MCP endpoint

**Status**: Implemented (v0.15.0)
**Target**: v0.15.x
**Priority**: P1 (defense-in-depth for the public unauthenticated MCP surface; pairs with M-FEEDBACK-TRIAGE-GATE)
**Estimated**: see "Two paths" below — fast path is ~1h, full path is ~1 day
**Dependencies**: none (this lands independently of the triage gate)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A4: Explicit Authority | **+1** | Anonymous endpoint stays anonymous; rate limit is the implicit authority signal ("you are not authorized to flood me") |
| A11: Structured Failure | **+1** | 429 with a structured `Retry-After` header beats silent 5xx under load |
| A12: System Boundary | **+2** | Tightens the public→internal boundary that today is wide open. The M7 TODO at [`cloud_run_mcp.tf:132-133`](../../../../ailang-multivac/terraform/cloud_run_mcp.tf) calls this out explicitly |

**Net Score: +4** → **Decision: move forward, but pick the right path**

---

## Problem Statement

`mcp.ailang.sunholo.com/submit_feedback` is unauthenticated and has **no per-IP rate limit**. The Terraform comment at [`cloud_run_mcp.tf:132-133`](../../../../ailang-multivac/terraform/cloud_run_mcp.tf) acknowledges:

> Public ingress: anonymous read, no auth in v1. Per-IP rate limiting lands in M7 alongside submit_feedback (Cloud Armor or middleware).

Combined with the downstream agent fan-out (see M-FEEDBACK-TRIAGE-GATE), one IP can comfortably submit hundreds of feedback messages in under a minute.

### What this doc fixes

A first line of defense at the edge: cap requests per IP per minute. This is **not** a substitute for the triage gate — it just slows the funnel feeding it.

### Initial misjudgement note

When this work was first scoped it sounded like a 5-minute Cloud Armor toggle. It is not. The current MCP setup has **no HTTPS Load Balancer** in front; Cloud Run owns its own HTTPS frontend via a `gcloud beta run domain-mappings` mapping (CNAME → `ghs.googlehosted.com.`). Cloud Armor only attaches to backend services on an external LB, so adding it requires a full LB+NEG cutover, not a single resource. Hence the two paths below.

---

## Two paths

### Path A — Application middleware (the actual 5-minute version)

Add a per-IP rate limiter as Go middleware in `internal/apiserver/`. Insert ahead of the existing `corsWrap` chain in [`server.go:477+`](../../../internal/apiserver/server.go).

**Implementation (~50 LOC):**

```go
// internal/apiserver/ratelimit.go
type ipLimiter struct {
    mu      sync.Mutex
    buckets map[string]*tokenBucket
    rate    float64 // tokens per second per IP
    burst   int
}

func (l *ipLimiter) allow(ip string) bool { /* token bucket */ }

func (s *Server) rateLimitWrap(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ip := clientIPFromForwardedFor(r) // prefer X-Forwarded-For (Cloud Run injects)
        if !s.rl.allow(ip) {
            w.Header().Set("Retry-After", "60")
            http.Error(w, `{"error":"rate_limited"}`, 429)
            return
        }
        next(w, r)
    }
}
```

**Configuration:** env vars `AILANG_RATELIMIT_RPM=5`, `AILANG_RATELIMIT_BURST=3`. Apply *only to write paths* (`/submit_feedback`); read paths can stay uncapped or use a much higher limit (60 rpm) since they're idempotent and cacheable.

**Pros:**
- Ships in the next image push; no infra change
- Reversible by env var; no DNS risk
- Fully unit-testable
- Same code path runs locally for dev/test

**Cons:**
- In-memory state — each Cloud Run instance has its own bucket. With `mcp_min_instances=0` and autoscale up to 10, an attacker who keeps multiple instances warm gets up to `instances × rate_limit` effective throughput. In practice this means the cap is "soft" by ~10x. Still enormously better than infinite.
- No Layer-3/4 protection — a true volumetric attack still hits Cloud Run cold-start spend
- No bot management or geo rules

**Estimated:** ~1h coding + tests, ~20min review/deploy. **This is the actual fast path.**

### Path B — Cloud Armor + HTTPS Load Balancer (the proper version)

Stand up an external HTTPS LB with a Serverless NEG pointing at the Cloud Run service, attach a Cloud Armor security policy with a rate-limit rule, and migrate `mcp.ailang.sunholo.com` from the Cloud Run domain mapping to the LB.

**Terraform sketch (~150 LOC, new file `terraform/cloud_run_mcp_lb.tf`):**

```hcl
# Serverless NEG (the bridge from LB → Cloud Run)
resource "google_compute_region_network_endpoint_group" "mcp_neg" {
  name                  = "${var.prefix}-mcp-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.region
  cloud_run {
    service = google_cloud_run_v2_service.mcp[0].name
  }
}

# Backend service with Cloud Armor attached
resource "google_compute_backend_service" "mcp_backend" {
  name                  = "${var.prefix}-mcp-backend"
  protocol              = "HTTPS"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.mcp_armor.id
  backend {
    group = google_compute_region_network_endpoint_group.mcp_neg.id
  }
  log_config { enable = true sample_rate = 1.0 }
}

# Cloud Armor: per-IP throttle
resource "google_compute_security_policy" "mcp_armor" {
  name = "${var.prefix}-mcp-armor"

  rule {
    action   = "throttle"
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config { src_ip_ranges = ["*"] }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      enforce_on_key = "IP"
      rate_limit_threshold {
        count        = 5     # 5 requests
        interval_sec = 60    # per minute
      }
    }
    description = "Per-IP 5/min throttle on the public MCP endpoint"
  }

  # Default allow (anything not matched falls through)
  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config { src_ip_ranges = ["*"] }
    }
    description = "Default allow"
  }
}

# URL map → backend
resource "google_compute_url_map" "mcp_url_map" {
  name            = "${var.prefix}-mcp-url-map"
  default_service = google_compute_backend_service.mcp_backend.id
}

# Managed SSL cert
resource "google_compute_managed_ssl_certificate" "mcp_cert" {
  name = "${var.prefix}-mcp-cert"
  managed { domains = [var.mcp_custom_domain] }
}

# HTTPS proxy + global forwarding rule + reserved IP (omitted for brevity)
```

**Cutover sequence:**

1. Apply Terraform — LB, NEG, backend, Armor policy, cert, IP all created. **No traffic yet.**
2. Wait for the Google-managed cert to provision (`gcloud compute ssl-certificates describe ${prefix}-mcp-cert` should report `ACTIVE`). Typically 15–30 min, can be up to 24h.
3. Test the LB IP directly with `curl -H "Host: mcp.ailang.sunholo.com" https://<LB_IP>/` (using `--resolve`). Confirm 200 plus the `via` and `x-cloud-trace-context` headers.
4. **DNS swap:** change `mcp.ailang.sunholo.com` from `CNAME ghs.googlehosted.com.` to an `A` record pointing at the LB IP. TTL is currently low (5 min); the swap is fast for clients with fresh caches but *some clients will still see the old endpoint for up to TTL*.
5. **After clients have migrated** (verify via Cloud Run access logs going quiet on the domain): delete the Cloud Run domain mapping with `gcloud beta run domain-mappings delete --domain=mcp.ailang.sunholo.com --region=europe-west1 --project=ailang-multivac`. The LB cert will keep working.

**Reversal plan if it goes wrong:**

- Revert DNS to `CNAME ghs.googlehosted.com.` and the Cloud Run domain mapping (still in place, since step 5 is post-confirmation) immediately serves traffic again.
- Keep Path A's middleware deployed even after Path B ships, as belt-and-suspenders.

**Pros:**
- True per-IP enforcement at the edge regardless of Cloud Run instance count
- Cloud Armor rules can grow (geo, IP allowlist, bot management, WAF preset rules)
- Logged at the LB layer for forensic visibility
- Standard GCP pattern; documented and stable

**Cons:**
- ~$18/mo additional spend (forwarding rule + reserved IP)
- DNS swap is a brief disruption window (clients with cached DNS see old endpoint for up to TTL)
- Cert provisioning is async (15-30 min wait between apply and cutover)
- ~150 LOC Terraform + a documented runbook
- Adds an extra hop to every request (~10-30 ms latency overhead)

**Estimated:** ~3h Terraform + 1h cutover wait + 1h post-verify. About **a day end-to-end** with cert provisioning included.

---

## Recommendation

Ship **Path A first**. It buys 80% of the protection in 1h and is reversible by env var. Reassess Path B once the triage gate (M-FEEDBACK-TRIAGE-GATE) is in place — at that point Cloud Armor's marginal value is mostly bot management and geo rules, which we may or may not need.

Do **not** ship Path B in isolation: a cleverly distributed flood across many IPs at sub-threshold rates still hits the agent fan-out unless triage is in place. Path B alone gives a false sense of security.

---

## Implementation Plan

### Path A (chosen for v0.15.x — recommended)

| Step | LOC | Time | Files |
|------|-----|------|-------|
| Token bucket + IP extraction | 50 | 30m | `internal/apiserver/ratelimit.go` |
| Wire into `submit_feedback` route | 10 | 10m | `internal/apiserver/server.go`, `internal/apiserver/feedback_tool.go` |
| Unit tests (allow + deny + clock advance) | 80 | 30m | `internal/apiserver/ratelimit_test.go` |
| `429 Retry-After` integration test | 30 | 15m | `internal/apiserver/feedback_tool_test.go` |
| Env var documentation | 10 | 5m | `docs/docs/guides/agent-mcp.md` |

**Acceptance:**

- 6 reqs in 60s from one IP → 5×200 + 1×429 with `Retry-After: 60`
- 6 reqs from 6 distinct IPs → 6×200 (per-IP, not global)
- `AILANG_RATELIMIT_RPM=0` disables the limiter (operator escape hatch)
- Read-only paths (`/_meta/*`, doc tool calls) are not throttled or use a higher limit (60 rpm)

### Path B (deferred, document only for now)

Captured above for the future operator who picks this up. Lives in `terraform/cloud_run_mcp_lb.tf` when it ships. Pre-cutover checklist:

- [ ] Snapshot DNS state (`dig mcp.ailang.sunholo.com +short`)
- [ ] Confirm `var.mcp_custom_domain` is set in the right tfvars file
- [ ] Apply with `bootstrap=false`
- [ ] Wait for cert ACTIVE
- [ ] Test via `--resolve` before DNS swap
- [ ] Schedule DNS swap during low-traffic window
- [ ] Keep old Cloud Run domain mapping as rollback for 48h
- [ ] After 48h clean, delete the domain mapping

---

## Risks & Tradeoffs

1. **Path A's per-instance memory means cap is soft under autoscale.** Mitigation: set `mcp_max_instances` low (currently 10; could lower to 3 for the public surface). Or sticky-route by IP via Cloud Run's session affinity (newer feature; check availability for v2 services).
2. **Path B's DNS swap is a real change-management risk.** Mitigation: small TTL (already in place) + keep old domain mapping for 48h post-swap as rollback.
3. **Path A behind Cloud Run sees `X-Forwarded-For` from Google's frontend** — must trust that header but only the rightmost IP. Mitigation: hardcode "use rightmost X-Forwarded-For" in the middleware; document in code comment.
4. **Read paths shouldn't be throttled** (they're cacheable and used by the docs MCP). Mitigation: apply rate-limit middleware only to write routes (`/submit_feedback`), not the read tool calls.

## Out of Scope (for v1)

- WAF / geo restrictions / bot management — premature; revisit if Path B ships
- mTLS or API key auth on the MCP endpoint — defeats anonymous AI-agent onboarding
- Per-API-key quotas — there are no API keys
- Distributed Redis-backed rate limiter — overkill until in-memory falls over

## Open Questions

1. **Path A on read paths too?** Recommend: no, or only with a 60 rpm limit that the docs MCP almost never hits. Cacheable + idempotent + the threat model is write-side.
2. **Should the rate limit be per-IP or per-`contact` field?** Recommend: per-IP at the edge; per-`contact` belongs in the triage gate (M-FEEDBACK-TRIAGE-GATE M2).
3. **If Path B ships, does Path A get removed?** Recommend: keep both. Belt and suspenders cost is one Go file + 50 LOC. Cheap.

---

## Success Metrics

- 1,000-request flood from one IP → ≤ 5 reach Cloud Run (Path A) / ≤ 5 reach the LB backend (Path B)
- No regression in legitimate-traffic P95 latency (rate limit middleware adds <1 ms in the cache-hit path)
- Operator can disable via env var (Path A) or Terraform var (Path B) in under 5 min if there's a false-positive incident

---

## References

- [`cloud_run_mcp.tf:132-133`](../../../../ailang-multivac/terraform/cloud_run_mcp.tf) — the existing M7 TODO this doc closes out
- [M-FEEDBACK-TRIAGE-GATE](m-feedback-triage-gate.md) — sister doc, does the deeper protection
- [`internal/apiserver/server.go:477+`](../../../internal/apiserver/server.go) — the route-mounting spot Path A hooks into
- [`internal/apiserver/feedback_tool.go:54-112`](../../../internal/apiserver/feedback_tool.go) — the only write tool today; the only route Path A throttles
- GCP docs: [Cloud Armor rate limiting](https://cloud.google.com/armor/docs/rate-limiting-overview), [Serverless NEG for Cloud Run](https://cloud.google.com/load-balancing/docs/negs/serverless-neg-concepts)

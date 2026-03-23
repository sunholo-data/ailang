# M-BILLING: DocParse Responsibility

**Parent**: [m-billing-docparse-billing-agent-payment.md](m-billing-docparse-billing-agent-payment.md)
**Repo**: `sunholo-data/docparse` (private)
**Infrastructure**: `ailang-multivac/terraform/docparse.tf` (existing)

---

## Scope

Integrate billing entitlement enforcement and usage recording into the existing DocParse Cloud Run API service. The DocParse service does NOT implement billing logic — it reads entitlements and enforces them.

---

## Existing DocParse Architecture

The DocParse API service currently:
- Authenticates via two modes: `x-api-key` (parsing) and Firebase Auth ID token (key management)
- Stores API keys (hashed) and usage logs in the `docparse` Firestore database
- Parses documents via Vertex AI (Gemini)
- Runs on Cloud Run with `sa-docparse` service account
- Has a 300s timeout for slow PDF processing

**Key constraint**: DocParse is a **separate repo** (`sunholo-data/docparse`), cloned during Cloud Build. Changes here must be self-contained within that repo.

---

## Changes Required

### 1. Add `docparse-access-gate` Package Dependency

Add to DocParse's `ailang.toml`:

```toml
[dependencies]
"sunholo/docparse-access-gate" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/docparse-access-gate", tag = "main" }
```

### 2. Entitlement Enforcement Middleware

Add to parse request flow (before invoking Vertex AI):

```
Request arrives
  → Authenticate (existing: x-api-key or Firebase Auth)
  → Extract principal ID from auth context
  → Load entitlements from Firestore (docparse-access-gate)
  → Call authorizeParse(principalId, parseRequest)
  → If Deny(reason): return 403 with reason
  → If Allow: proceed to parse
  → After successful parse: recordSuccessfulParse(principalId, pages, bytes, ocrPages)
```

**Critical**: The entitlement check must be fast. Entitlements are a single Firestore document read (`entitlements/{principal_id}`) — no collection scan. Target: <10ms added latency.

### 3. Principal ID Resolution

Current auth modes map to principal IDs:

| Auth Mode | Principal Source | Principal ID |
|-----------|-----------------|-------------|
| `x-api-key` | API key lookup → associated `userId` | `userId` from api_keys collection |
| Firebase Auth | ID token claims | `token.uid` |

Both modes already resolve to a user ID. The entitlement check uses this existing user ID as the `principalId`.

### 4. Free Tier Default

If no entitlements document exists for a principal, treat them as **free tier**:
- Do NOT reject — fall back to free plan capabilities
- Create a default entitlements document on first parse (lazy initialization)
- This ensures existing users aren't broken when billing rolls out

### 5. Usage Recording

After every successful parse, record usage delta:

```
pages parsed (from Vertex AI response)
bytes processed (from input file size)
OCR pages (if applicable, from processing metadata)
```

Usage is recorded to `usage/{principal_id}/periods/{yyyy_mm}` using atomic increments (Firestore FieldValue.increment).

### 6. Entitlements API Endpoint

Add optional read-only endpoint for the DocParse dashboard UI:

```
GET /me/entitlements
Authorization: Bearer <Firebase Auth ID token>

Response:
{
  "plan": "free",
  "canParse": true,
  "apiAccess": false,
  "monthlyPageLimit": 100,
  "monthlyPagesUsed": 42,
  "remainingPages": 58,
  "maxFileSizeMb": 10,
  "maxConcurrentJobs": 1
}
```

### 7. Dashboard UI Changes

Update the existing `docs/dashboard.html` to show:
- Current plan name and status
- Usage meters (pages used / limit, documents used / limit)
- "Upgrade" button that links to the billing service checkout flow
- "Manage Billing" button that links to Stripe Customer Portal

The billing service URL is configured as an environment variable in the DocParse Cloud Run service.

---

## New Environment Variables

Add to `docparse.tf` in ailang-multivac (or pass via Cloud Build):

```
BILLING_SERVICE_URL=https://{prefix}-billing-api-{hash}.run.app
```

The DocParse service uses this to generate redirect URLs for the dashboard upgrade/manage buttons. It does NOT call the billing service directly — the frontend makes those calls.

---

## Error Handling

### Entitlement check failures

If Firestore read fails (network error, timeout):
- **Do NOT block the parse** — fail open with free tier defaults
- Log the failure for alerting
- Rationale: availability > enforcement for a new feature rollout

### Usage recording failures

If usage write fails:
- **Do NOT fail the parse response** — the user already got their result
- Retry once, then log the failure
- Usage discrepancies are acceptable; double-counting is not
- Rationale: usage is for soft limits, not hard billing

---

## Phase Mapping

| Phase | What to Do |
|-------|-----------|
| 1 | No DocParse changes. Billing infrastructure being set up in multivac. |
| 2 | Add `docparse-access-gate` dependency. Add entitlement middleware. Add usage recording. Add `/me/entitlements` endpoint. |
| 3 | Add dashboard UI: usage meters, upgrade button, manage billing button. |
| 4 | Tighten error handling (fail-closed option for paid tiers). Grace period support. |

---

## Testing Strategy

### Unit tests
- Authorization decisions with mock entitlements (various plans, quota states)
- Principal ID resolution from both auth modes
- Usage delta computation

### Integration tests
- Entitlement check against Firestore emulator
- Usage recording with atomic increments
- Free tier fallback when no entitlements doc exists

### End-to-end tests
- Parse request with free tier → succeeds within limits
- Parse request exceeding quota → 403 with clear reason
- Parse request with API key on free tier (no API access) → 403
- Parse request with pro tier → succeeds, usage recorded
- Entitlement read failure → falls back to free tier (fail-open)

---

## Rollout Plan

### Phase 2a: Shadow mode (read but don't enforce)

1. Deploy entitlement loading + usage recording
2. Log authorization decisions but don't reject
3. Monitor: what percentage of requests would be denied?
4. Verify usage recording accuracy

### Phase 2b: Enforce

1. Enable enforcement for new users (no existing entitlements doc)
2. Create free-tier entitlements for all existing users
3. Enable enforcement for all users
4. Monitor 403 rate

---

## Checklist

- [ ] Add `docparse-access-gate` to DocParse's `ailang.toml` dependencies
- [ ] Add entitlement loading middleware to parse request flow
- [ ] Add `authorizeParse` call before Vertex AI invocation
- [ ] Add `recordSuccessfulParse` call after successful parse
- [ ] Add free tier fallback for missing entitlements
- [ ] Add `/me/entitlements` endpoint
- [ ] Add `BILLING_SERVICE_URL` environment variable support
- [ ] Update dashboard UI with usage meters and upgrade/manage buttons
- [ ] Add shadow mode flag for gradual rollout
- [ ] Write unit tests for authorization decisions
- [ ] Write integration tests against Firestore emulator
- [ ] Test fail-open behavior when Firestore is unavailable

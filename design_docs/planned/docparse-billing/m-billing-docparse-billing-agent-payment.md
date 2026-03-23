# M-BILLING: DocParse Billing and Agent-Assisted Payment

**Status**: Proposed
**Priority**: P1 (High)
**Estimated**: 4 phases, ~3 weeks
**Dependencies**: DocParse Cloud Run (existing), Firebase Auth (existing), Firestore (existing)
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Executive Summary

Add subscription billing to DocParse via a **separate Cloud Run billing service** backed by Stripe and Firestore. The billing domain is modeled as reusable AILANG packages in the `ailang-packages` monorepo. AI agents may propose plan changes and generate approval links, but humans remain the payment authority.

This is the first production AILANG application deployed outside the core language repo.

---

## Multi-Repo Coordination

This design spans **four repositories**. Each has a scoped responsibility doc in this directory:

| Repo | Responsibility Doc | What It Owns |
|------|--------------------|-------------|
| `ailang-packages` | [responsibility-packages.md](responsibility-packages.md) | 6 billing AILANG packages (pure policy, proposals, store, Stripe adapter, API handlers, access gate) |
| `docparse` | [responsibility-docparse.md](responsibility-docparse.md) | Entitlement enforcement middleware, usage recording, UI integration |
| `ailang-multivac` | [responsibility-multivac.md](responsibility-multivac.md) | Terraform for new billing Cloud Run service, secrets, IAM, Firestore indexes, Cloud Build pipeline |
| `ailang` (this repo) | This doc + sprint coordination | Design docs, sprint planning, cross-repo coordination via messaging |

### Execution Model

Each responsibility doc is designed to be **independently actionable** by an agent working in that repo. The sprint planner creates a unified timeline, and the sprint executor coordinates via `ailang messages` between repos.

```
┌──────────────────────────────────────────────────────────────┐
│  ailang (this repo) — COORDINATOR                            │
│  - Master design doc (this file)                             │
│  - Sprint plan (created by sprint-planner skill)             │
│  - Cross-repo message orchestration                          │
├──────────────────────────────────────────────────────────────┤
│                           │                                  │
│              ┌────────────┼────────────┐                     │
│              ▼            ▼            ▼                     │
│  ┌───────────────┐ ┌──────────┐ ┌──────────────┐           │
│  │ailang-packages│ │ docparse │ │ailang-multivac│           │
│  │               │ │          │ │               │           │
│  │ AILANG billing│ │ App-level│ │ Terraform     │           │
│  │ packages      │ │ integrate│ │ infra for     │           │
│  │ (pure logic)  │ │ billing  │ │ billing svc   │           │
│  └───────────────┘ └──────────┘ └───────────────┘           │
└──────────────────────────────────────────────────────────────┘
```

### Phase Dependencies

```
Phase 1 (Foundation):
  multivac/terraform  ──► creates billing Cloud Run + secrets + Firestore indexes
  packages/billing-*  ──► core types + entitlement policy (can start in parallel)
  docparse            ──► BLOCKED until multivac Firestore collections exist

Phase 2 (Enforcement):
  packages            ──► billing-store + docparse-access-gate
  docparse            ──► integrates access gate into parse middleware

Phase 3 (Agent Assistance):
  packages            ──► billing-proposals
  docparse            ──► UI approval-link presentation

Phase 4 (Hardening):
  All repos            ──► operational polish, idempotency, monitoring
```

---

## Context

DocParse is the first production AILANG system deployed behind a Cloud Run backend with Firebase Auth and Firestore-backed application state. We need paid plans without coupling billing logic into the DocParse parsing runtime.

The system must support both human-led and AI-assisted upgrade flows. AI agents should be able to prepare a billing action and generate a human approval link, while final payment authorization remains with the human user.

---

## Goals

### Primary

- Add subscription billing for DocParse with a production-safe path
- Keep Stripe integration isolated from DocParse parsing logic
- Store app entitlements separately from billing provider state
- Support agent-assisted payment setup with human approval
- Reuse the billing subsystem across future Sunholo products

### Secondary

- Make policy decisions explicit and deterministic where possible
- Expose a narrow, AI-friendly API surface for billing actions
- Preserve future compatibility with emerging agent payment protocols

### Non-goals

- Full machine-authorized autonomous payment execution
- Multi-provider billing abstraction in v1
- Marketplace, reseller, or invoicing-heavy enterprise billing in v1
- Deep metered billing in v1

---

## Key Design Principles

### 1. Stripe is an adapter, not the domain model

Stripe provides checkout, subscriptions, customer portal, and webhook events. Runtime access decisions do not depend directly on raw Stripe objects.

### 2. Billing state and entitlement state are separate

- **Billing state** answers: what Stripe says
- **Entitlement state** answers: what DocParse allows

### 3. AI may propose, humans approve

Agents can prepare plan changes and create approval links. Humans remain the authority that approves payment.

### 4. DocParse checks capabilities, not payment vendor details

DocParse asks whether a principal may perform a parse operation under current limits, not whether a Stripe subscription is active.

### 5. Policy core should be reusable and mostly pure

Plan resolution, quota checks, and capability decisions are isolated from external API effects.

---

## Architecture

### Service Topology

```
┌─────────────────────┐     ┌──────────────────────┐
│  DocParse API        │     │  Billing Service      │
│  (existing Cloud Run)│     │  (new Cloud Run)      │
│  repo: docparse      │     │  repo: ailang-packages│
│                      │     │      + multivac TF    │
│  - Auth requests     │     │                       │
│  - Load entitlements │     │  - Stripe Checkout    │
│  - Authorize parse   │     │  - Stripe Webhooks    │
│  - Record usage      │     │  - Payment Proposals  │
│                      │     │  - Entitlement Writes │
└──────────┬───────────┘     └──────────┬────────────┘
           │                            │
           └────────┬───────────────────┘
                    ▼
           ┌────────────────┐
           │   Firestore    │
           │  (docparse DB) │
           │                │
           │  billing_*     │
           │  entitlements  │
           │  usage         │
           │  proposals     │
           └────────────────┘
```

**Key decision**: Billing collections live in the **existing `docparse` Firestore database** (not the `(default)` coordinator DB). This keeps billing data co-located with DocParse app data and avoids cross-database reads for entitlement checks.

### A. DocParse API Service (existing Cloud Run)

- Repo: `sunholo-data/docparse`
- Existing: `docparse.tf` in multivac, `sa-docparse` service account
- Changes: add entitlement loading + parse authorization middleware + usage recording
- See: [responsibility-docparse.md](responsibility-docparse.md)

### B. Billing Service (new Cloud Run)

- Repo: application code in `ailang-packages`, infrastructure in `ailang-multivac`
- New: `billing.tf` in multivac, `sa-billing` service account
- Responsibilities: Stripe integration, webhook processing, entitlement writes, proposal management
- See: [responsibility-multivac.md](responsibility-multivac.md) (infra), [responsibility-packages.md](responsibility-packages.md) (code)

### C. Firestore (shared `docparse` database)

New collections added to the existing `docparse` Firestore DB:
- `billing_customers` — Stripe customer mapping
- `subscriptions` — subscription state (Stripe-sourced)
- `entitlements` — app-level capability state (derived from subscriptions)
- `usage` — subcollection per period
- `payment_proposals` — agent/UI payment proposals
- `billing_events` — webhook event dedup log

---

## User and Agent Flows

### Flow 1: Human Upgrades from UI

1. User signs in with Firebase Auth
2. Frontend calls billing service to create checkout session for selected plan
3. Billing service creates Stripe Checkout Session
4. Frontend receives hosted approval URL, redirects user
5. User approves payment on Stripe-hosted page
6. Stripe webhook sent to billing service
7. Billing service updates `subscriptions` and `entitlements` in Firestore
8. DocParse service begins allowing upgraded capability set

### Flow 2: AI-Assisted Upgrade Proposal

1. Agent detects need for upgrade (exhausted quota, API access request)
2. Agent calls billing service to create a payment proposal
3. Billing service stores proposal with `pending_approval` status
4. Agent requests an approval link for that proposal
5. Billing service creates Stripe Checkout Session, returns approval URL
6. Agent presents URL to human in chat or UI
7. Human approves payment on Stripe-hosted page
8. Webhook confirms payment and entitlements update

### Flow 3: Manage Billing

1. User or agent requests billing management
2. Billing service creates Stripe Customer Portal session
3. Hosted portal URL returned
4. Human updates card, switches plan, or cancels
5. Stripe webhook updates subscription and entitlements

---

## Data Model

### billing_customers/{principal_id}

```json
{
  "principalType": "user",
  "principalId": "uid_123",
  "stripeCustomerId": "cus_123",
  "email": "user@example.com",
  "createdAt": "2026-03-23T12:00:00Z"
}
```

### subscriptions/{principal_id}

```json
{
  "provider": "stripe",
  "principalType": "user",
  "principalId": "uid_123",
  "subscriptionId": "sub_123",
  "status": "active",
  "priceId": "price_docparse_pro_monthly",
  "productId": "prod_docparse_pro",
  "currentPeriodStart": "2026-03-23T12:00:00Z",
  "currentPeriodEnd": "2026-04-23T12:00:00Z",
  "cancelAtPeriodEnd": false,
  "updatedAt": "2026-03-23T12:05:00Z"
}
```

### entitlements/{principal_id}

```json
{
  "principalType": "user",
  "principalId": "uid_123",
  "plan": "pro",
  "status": "active",
  "canParse": true,
  "apiAccess": true,
  "maxFileSizeMb": 50,
  "monthlyDocumentLimit": 500,
  "monthlyPageLimit": 5000,
  "monthlyPagesUsed": 412,
  "maxConcurrentJobs": 3,
  "gracePeriod": false,
  "updatedAt": "2026-03-23T12:05:00Z"
}
```

### usage/{principal_id}/periods/{yyyy_mm}

```json
{
  "documentsParsed": 31,
  "pagesParsed": 412,
  "bytesProcessed": 18273645,
  "ocrPagesParsed": 120,
  "updatedAt": "2026-03-23T12:10:00Z"
}
```

### payment_proposals/{proposal_id}

```json
{
  "principalType": "user",
  "principalId": "uid_123",
  "targetPlan": "pro_monthly",
  "status": "pending_approval",
  "reason": "Monthly page limit nearly exhausted; API access requested",
  "requestedBy": "agent",
  "requestedByAgent": "docparse-assistant",
  "stripeCheckoutSessionId": null,
  "approvalUrl": null,
  "expiresAt": null,
  "createdAt": "2026-03-23T12:00:00Z"
}
```

### billing_events/{event_id}

```json
{
  "provider": "stripe",
  "providerEventId": "evt_123",
  "type": "customer.subscription.updated",
  "processed": true,
  "processedAt": "2026-03-23T12:06:00Z"
}
```

---

## Plan Model

### Initial v1 Plans

| Plan | Billing | Key Features |
|------|---------|-------------|
| `free` | None | Basic UI, small quota, small file cap, no API, low concurrency |
| `pro_monthly` | Monthly | Higher quota, API access, larger files, moderate concurrency, priority processing |
| `business_monthly` | Monthly | Higher quotas, team/org support later, elevated concurrency, support priority |

Annual variants deferred to post-v1.

### Plan Catalog Format

Maintained as config checked into repo and loaded by billing service:

```json
{
  "pro_monthly": {
    "stripePriceId": "price_docparse_pro_monthly",
    "plan": "pro",
    "monthlyPageLimit": 5000,
    "monthlyDocumentLimit": 500,
    "maxFileSizeMb": 50,
    "apiAccess": true,
    "maxConcurrentJobs": 3
  }
}
```

---

## Service API Surface

### Billing Service Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/billing/proposals` | Create a payment proposal |
| `POST` | `/billing/proposals/{id}/approval-link` | Create Stripe Checkout Session, return URL |
| `GET` | `/billing/proposals/{id}` | Get proposal state (polling/UI) |
| `POST` | `/billing/checkout-session` | Direct plan selection (human UI, no proposal) |
| `POST` | `/billing/portal-session` | Stripe Customer Portal URL |
| `POST` | `/billing/webhooks/stripe` | Receive Stripe webhook events |
| `GET` | `/billing/me/entitlements` | Current effective entitlements |

### DocParse API Changes

Before parse execution:
1. Verify Firebase token
2. Load entitlements from Firestore
3. Call `authorizeParse` — reject if limits exceeded

After successful parse:
1. Compute usage delta
2. Persist usage delta

Optional: `GET /me/entitlements` for UI display.

---

## AI-Friendly Design

The billing system exposes **intent-level operations**, not raw vendor primitives.

### Good AI-facing operations

- `suggest_plan_for_user`
- `create_payment_proposal`
- `summarize_plan_change`
- `create_approval_link`
- `get_proposal_status`
- `get_entitlements`

### Explicitly blocked in v1

- Create arbitrary charges
- Refund arbitrary payments
- Mutate product catalog
- Override invoice totals
- Execute payment without human approval

---

## Security Model

### Authentication

- User-facing endpoints: Firebase Auth bearer token
- Stripe webhooks: Stripe signature verification
- Service-to-service: IAM or signed identity between Cloud Run services

### Authorization

- Only the authenticated principal may create proposals for itself in v1
- Admin overrides behind separate privileged interface
- Agent-created proposals scoped to an authenticated user principal

### Secrets (managed in multivac)

Billing service needs (via Secret Manager → Cloud Run env):
- `{prefix}-stripe-secret-key`
- `{prefix}-stripe-webhook-secret`

---

## Idempotency and Correctness

### Webhooks

- Store provider event ID in `billing_events`
- Ignore already-processed events
- Process handlers idempotently

### Usage writes

- Updates must be safe under retries
- Parse jobs use stable job IDs to prevent duplicate quota charges

### Checkout creation

- Repeated agent requests for the same pending proposal reuse or replace the current approval session

---

## Future Compatibility

### Agent Payment Protocols

The internal approval model stays provider-neutral:

```
prepare billable action → request authorization → receive payment result → update entitlements
```

In v1, approval method = Stripe-hosted Checkout. Future options:
- Agent payment protocol adapters
- Machine-to-machine payment for paid APIs
- Enterprise invoice approval rails

---

## Operational Requirements

### Monitoring

- Webhook success/failure rate
- Checkout session creation failures
- Entitlement write failures
- Parse authorization denials by reason
- Usage write conflicts

### Alerts

- Repeated webhook signature failures
- Billing service 5xx spikes
- Mismatch between active subscriptions and active entitlements
- Stuck pending proposals with no follow-up after configurable interval

---

## Migration Plan

### Phase 1: Foundation (multivac + packages, in parallel)

**multivac**: Create billing Cloud Run service, secrets, service account, Firestore indexes
**packages**: billing-entitlements (plan types, capability checks), billing-stripe (adapter skeleton)
**docparse**: No changes yet

### Phase 2: Enforcement (packages + docparse)

**packages**: billing-store, docparse-access-gate
**docparse**: Integrate access gate into parse middleware, add usage recording
**multivac**: Deploy billing service image via Cloud Build

### Phase 3: Agent Assistance (packages + docparse)

**packages**: billing-proposals, billing-service-api handlers
**docparse**: UI approval-link presentation

### Phase 4: Hardening (all repos)

**All**: Idempotency fixes, admin overrides, grace periods, operational dashboards

---

## Open Questions

1. **Principal scope**: User-level only in v1, or add org-level principal abstraction immediately?
2. **Free-tier quota resets**: Computed lazily or materialized monthly?
3. **Annual plans**: Needed in v1 or only monthly?
4. **API access gating**: Plan-only, or separate API token issuance policy?
5. **Usage granularity**: Documents, pages, bytes, OCR pages, runtime seconds — how much in v1?

---

## Recommended Decisions

| Decision | Recommendation |
|----------|---------------|
| Principal model | User-level in v1, but model with `principalType`/`principalId` for future org support |
| Billing cycle | Monthly subscriptions only in v1 |
| Pricing model | Fixed-plan quotas, not metered billing |
| Checkout UX | Stripe-hosted Checkout and Customer Portal |
| Agent payments | Require human approval for any paid change initiated by agents |
| Service topology | Separate billing service from DocParse parse service |
| Firestore DB | Use existing `docparse` database for billing collections |

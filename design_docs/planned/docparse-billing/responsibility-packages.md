# M-BILLING: ailang-packages Responsibility

**Parent**: [m-billing-docparse-billing-agent-payment.md](m-billing-docparse-billing-agent-payment.md)
**Repo**: `sunholo-data/ailang-packages`
**Working directory**: `/Users/mark/dev/sunholo/ailang-packages`

---

## Scope

Create 6 new AILANG packages in `packages/` that implement the billing domain model, Stripe adapter, storage layer, API handlers, and DocParse access gate. These packages follow the existing monorepo conventions (`ailang.toml`, `AGENT.md`, module declarations).

**This is the first AILANG application code** — all previous packages were infrastructure utilities (auth, logging, config). These packages implement business logic.

---

## Existing Conventions to Follow

Each package needs:
- `ailang.toml` — manifest with name, exports, effects, metadata, ai_summary
- `AGENT.md` — AI agent discovery guide
- `.ail` source files with `module vendor/name/module` declarations
- `[exports].modules` listing in ailang.toml
- `[effects].max` declaring maximum effects used

Package naming: `sunholo/billing-*` for reusable billing packages, `sunholo/docparse-access-gate` for DocParse-specific integration.

---

## Package 1: billing-entitlements

**Purpose**: Core plan and entitlement policy — pure logic, no effects.

**Path**: `packages/billing-entitlements/`

**Effects**: Pure (no effects)

**Modules**:

### plan.ail
```ailang
module sunholo/billing-entitlements/plan

-- Plan catalog types
type Plan = { name: String, monthlyPageLimit: Int, monthlyDocumentLimit: Int,
              maxFileSizeMb: Int, apiAccess: Bool, maxConcurrentJobs: Int }

type PlanCatalog = { plans: List(Pair(String, Plan)) }

export func lookupPlan(catalog: PlanCatalog, planKey: String) -> Result(Plan, String)
export func freePlan() -> Plan
```

### entitlement.ail
```ailang
module sunholo/billing-entitlements/entitlement

type Entitlements = { principalId: String, plan: String, status: String,
                      canParse: Bool, apiAccess: Bool, maxFileSizeMb: Int,
                      monthlyDocumentLimit: Int, monthlyPageLimit: Int,
                      maxConcurrentJobs: Int, gracePeriod: Bool }

type SubscriptionState = { status: String, planKey: String, cancelAtPeriodEnd: Bool }

export func resolveEntitlements(sub: SubscriptionState, catalog: PlanCatalog,
                                 principalId: String) -> Entitlements
```

### capability_check.ail
```ailang
module sunholo/billing-entitlements/capability_check

type ParseRequest = { fileSizeMb: Int, isApiRequest: Bool }
type AllowDecision = Allow | Deny(String)

export func canParse(ent: Entitlements, req: ParseRequest,
                     currentUsage: Usage) -> AllowDecision
export func canUseApi(ent: Entitlements) -> Bool
```

### quota_policy.ail
```ailang
module sunholo/billing-entitlements/quota_policy

export func isOverDocumentLimit(ent: Entitlements, used: Int) -> Bool
export func isOverPageLimit(ent: Entitlements, used: Int) -> Bool
export func remainingPages(ent: Entitlements, used: Int) -> Int
```

### usage_policy.ail
```ailang
module sunholo/billing-entitlements/usage_policy

type Usage = { documentsParsed: Int, pagesParsed: Int, bytesProcessed: Int,
               ocrPagesParsed: Int }
type UsageDelta = { documents: Int, pages: Int, bytes: Int, ocrPages: Int }

export func recordUsageDelta(current: Usage, delta: UsageDelta) -> Usage
```

---

## Package 2: billing-proposals

**Purpose**: AI- and UI-facing proposal model for "prepared but not yet approved" commercial actions.

**Path**: `packages/billing-proposals/`

**Effects**: Pure

**Modules**:

### proposal.ail
```ailang
module sunholo/billing-proposals/proposal

type ProposalStatus = PendingApproval | AwaitingPayment | Approved | Expired | Rejected
type RequestedBy = Human | Agent(String)

type Proposal = { proposalId: String, principalId: String, targetPlan: String,
                  status: ProposalStatus, reason: String, requestedBy: RequestedBy,
                  approvalUrl: Result(String, String), expiresAt: Result(String, String),
                  createdAt: String }

export func createProposal(principalId: String, targetPlan: String,
                            reason: String, requestedBy: RequestedBy) -> Proposal
export func canCreateApprovalLink(p: Proposal) -> Bool
export func markAwaitingPayment(p: Proposal, url: String, expires: String) -> Proposal
export func markApproved(p: Proposal) -> Proposal
export func markExpired(p: Proposal) -> Proposal
```

### proposal_summary.ail
```ailang
module sunholo/billing-proposals/proposal_summary

type ProposalSummary = { description: String, currentPlan: String,
                         targetPlan: String, priceChange: String }

export func summarizeProposal(p: Proposal, catalog: PlanCatalog,
                               currentPlan: String) -> ProposalSummary
```

---

## Package 3: billing-store

**Purpose**: Firestore storage abstractions. All Firestore reads/writes go through this package.

**Path**: `packages/billing-store/`

**Effects**: `{ Net, IO }` (Firestore operations)

**Modules**:

### customers_repo.ail
```ailang
module sunholo/billing-store/customers_repo

export func getCustomer(principalId: String) -> Result(Customer, String) ! {Net}
export func putCustomer(customer: Customer) -> Result((), String) ! {Net}
```

### subscriptions_repo.ail
```ailang
module sunholo/billing-store/subscriptions_repo

export func getSubscription(principalId: String) -> Result(Subscription, String) ! {Net}
export func putSubscription(sub: Subscription) -> Result((), String) ! {Net}
```

### entitlements_repo.ail
```ailang
module sunholo/billing-store/entitlements_repo

export func getEntitlements(principalId: String) -> Result(Entitlements, String) ! {Net}
export func putEntitlements(ent: Entitlements) -> Result((), String) ! {Net}
```

### usage_repo.ail
```ailang
module sunholo/billing-store/usage_repo

export func getUsage(principalId: String, period: String) -> Result(Usage, String) ! {Net}
export func applyUsageDelta(principalId: String, period: String,
                             delta: UsageDelta) -> Result((), String) ! {Net}
```

### proposals_repo.ail
```ailang
module sunholo/billing-store/proposals_repo

export func createProposal(p: Proposal) -> Result(String, String) ! {Net}
export func getProposal(proposalId: String) -> Result(Proposal, String) ! {Net}
export func updateProposal(p: Proposal) -> Result((), String) ! {Net}
```

### events_repo.ail
```ailang
module sunholo/billing-store/events_repo

export func isEventProcessed(providerEventId: String) -> Result(Bool, String) ! {Net}
export func recordEvent(event: BillingEvent) -> Result((), String) ! {Net}
```

---

## Package 4: billing-stripe

**Purpose**: Stripe-specific integration adapter.

**Path**: `packages/billing-stripe/`

**Effects**: `{ Net }` (HTTP calls to Stripe API)

**Modules**:

### stripe_customer.ail
```ailang
module sunholo/billing-stripe/stripe_customer

export func ensureStripeCustomer(principalId: String,
                                  email: String) -> Result(String, String) ! {Net}
```

### stripe_checkout.ail
```ailang
module sunholo/billing-stripe/stripe_checkout

type CheckoutSession = { sessionId: String, url: String, expiresAt: String }

export func createCheckoutSession(customerId: String, priceId: String,
                                   successUrl: String, cancelUrl: String,
                                   metadata: List(Pair(String, String)))
                                   -> Result(CheckoutSession, String) ! {Net}
```

### stripe_portal.ail
```ailang
module sunholo/billing-stripe/stripe_portal

export func createPortalSession(customerId: String,
                                 returnUrl: String) -> Result(String, String) ! {Net}
```

### stripe_webhook.ail
```ailang
module sunholo/billing-stripe/stripe_webhook

type VerifiedEvent = { eventId: String, eventType: String, data: String }

export func verifyWebhook(signature: String,
                           rawBody: String) -> Result(VerifiedEvent, String)
```

### stripe_event_mapper.ail
```ailang
module sunholo/billing-stripe/stripe_event_mapper

type SubscriptionStateChange = { principalId: String, newState: SubscriptionState }

export func mapEventToSubscriptionChange(event: VerifiedEvent)
                                          -> Result(SubscriptionStateChange, String)
```

---

## Package 5: billing-service-api

**Purpose**: HTTP handlers for the Cloud Run billing service.

**Path**: `packages/billing-service-api/`

**Effects**: `{ Net, IO }`

**Modules**: One handler per endpoint:
- `create_checkout_session_handler.ail` — `POST /billing/checkout-session`
- `create_portal_session_handler.ail` — `POST /billing/portal-session`
- `create_payment_proposal_handler.ail` — `POST /billing/proposals`
- `get_payment_proposal_handler.ail` — `GET /billing/proposals/{id}`
- `approval_link_handler.ail` — `POST /billing/proposals/{id}/approval-link`
- `stripe_webhook_handler.ail` — `POST /billing/webhooks/stripe`
- `get_entitlements_handler.ail` — `GET /billing/me/entitlements`
- `main.ail` — HTTP server setup, routing, middleware

Each handler:
1. Authenticates (Firebase token or Stripe signature)
2. Validates input
3. Calls into billing-entitlements / billing-proposals / billing-stripe / billing-store
4. Returns JSON response

---

## Package 6: docparse-access-gate

**Purpose**: Package used by existing DocParse API service to enforce entitlements.

**Path**: `packages/docparse-access-gate/`

**Effects**: `{ Net }` (reads from Firestore)

**Modules**:

### parse_authorization.ail
```ailang
module sunholo/docparse-access-gate/parse_authorization

-- Called by DocParse before every parse operation
export func authorizeParse(principalId: String,
                            req: ParseRequest) -> Result(AllowDecision, String) ! {Net}
```

### usage_recording.ail
```ailang
module sunholo/docparse-access-gate/usage_recording

-- Called by DocParse after every successful parse
export func recordSuccessfulParse(principalId: String,
                                   pages: Int, bytes: Int,
                                   ocrPages: Int) -> Result((), String) ! {Net}
```

---

## Dependency Graph Between Packages

```
billing-entitlements (pure)  ◄── billing-proposals (pure)
        ▲                              ▲
        │                              │
billing-store (Net, IO)       billing-stripe (Net)
        ▲                              ▲
        │                              │
        └──────── billing-service-api ─┘
                         ▲
                         │
              docparse-access-gate
              (uses billing-entitlements + billing-store)
```

---

## Phase Mapping

| Phase | Packages | What |
|-------|----------|------|
| 1 | billing-entitlements, billing-stripe (skeleton) | Core types, plan resolution, Stripe adapter types |
| 2 | billing-store, docparse-access-gate | Firestore CRUD, parse authorization |
| 3 | billing-proposals, billing-service-api | Proposal model, HTTP handlers |
| 4 | All — hardening | Idempotency, error handling, grace periods |

---

## Testing Strategy

- **billing-entitlements**: Unit tests with `sunholo/testing-utils` — pure functions, no mocks needed
- **billing-proposals**: Unit tests — pure state transitions
- **billing-store**: Integration tests against Firestore emulator
- **billing-stripe**: Mock HTTP responses for Stripe API calls
- **billing-service-api**: End-to-end handler tests with mocked Stripe + Firestore
- **docparse-access-gate**: Unit tests for authorization decisions, integration tests for Firestore reads

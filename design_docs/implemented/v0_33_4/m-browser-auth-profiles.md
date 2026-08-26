# M-BROWSER-AUTH-PROFILES — persistent authenticated identity without password exposure

**Status**: IMPLEMENTED
**Target**: v0.33.4
**Priority**: P1 (Medium-High — required before browser agents access private production systems)
**Estimated**: 7–9 engineering days, excluding provider/account onboarding
**Dependencies**: M-REMOTE-BROWSER-SESSION-PROVIDERS; existing `internal/secrets` resolver and Secret approval/audit path. Production use against untrusted destinations is blocked on the browser-egress and artifact-data-policy follow-ups defined below.

## Summary

Add persistent authenticated identities to AILANG browser sessions without giving an AI model a password, a canonical browser profile, or mutable shared session state.

The trusted control plane creates or refreshes an authenticated profile outside the AI tool loop. It stores only an encrypted browser-state snapshot or an opaque hosted-provider context reference. Each agent run leases that profile, materializes a disposable read-only session copy, and destroys the copy after use. Ordinary runs never write back. Refresh/write-back is a separate audited operation with narrower authority.

This design treats cookies, local storage, IndexedDB tokens, and virtual passkeys as credentials equivalent to passwords. It supports one dedicated account for serialized/read-only work and an account pool for parallel or state-changing work.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Profile version, content hash, lease, allowed origins, refresh cause, and provider identity become explicit result inputs. |
| A2: Replayability | +1 | Runs reference immutable versioned snapshots; sensitive values are not put in replay bundles. |
| A3: Effect Legibility | +1 | Materialize, lease, refresh, persist, revoke, and delete are separate named operations. |
| A4: Explicit Authority | +1 | The agent receives authenticated browser authority but never credential-resolution or canonical-profile mutation authority. |
| A5: Bounded Verification | +1 | Lease TTLs, session TTLs, exact-origin scope, expiry checks, and destructive cleanup are locally testable. |
| A6: Safe Concurrency | +1 | One writer lease per profile and account-pool allocation prevent concurrent profile corruption and server-state collisions. |
| A7: Machines First | +1 | `AuthProfileRef`, lease records, failure categories, and audit events are structured and safe to serialize. |
| A8: Minimal Syntax | 0 | No AILANG language syntax is added. |
| A9: Cost Visibility | +1 | Profile refresh and hosted-context storage/usage are attributed separately from model and browser-session cost. |
| A10: Composability | +1 | The profile broker composes with local Playwright and Browserbase through the existing `SessionSpec.ProfileRef`. |
| A11: Structured Failure | +1 | Missing, expired, leased, scope-denied, refresh-required, materialization, and write-back failures are stable categories. |
| A12: System Boundary | +1 | Vault, encrypted blob store, hosted context, disposable session, website account, and model are distinct boundary objects. |

**Net Score: +11** → **Decision: ✅ Proceed to implementation after design-freeze review**

### Hard Violation Check

- [x] A1 (Determinism): profile identity/version and refresh state are explicit
- [x] A3 (Effects): credential bootstrap, materialization, use, refresh, and destruction are separate operations
- [x] A4 (Authority): password resolution and canonical write-back are unavailable to the model
- [x] A7 (Machines First): references, policies, leases, failures, and audits are structured

## Problem Statement

The browser-session layer deliberately starts every run with a fresh local Playwright profile or fresh Browserbase session. `SessionSpec` already reserves `ProfileRef` and `ProfileHash`, but neither provider materializes authenticated state.

**Current State:**

- Local Playwright always uses `--isolated`; it does not load a storage-state snapshot.
- Browserbase creates a fresh session; it does not attach a persistent Context.
- There is no profile registry, profile lease, version, expiry, refresh, revocation, or write-back policy under `internal/browser`.
- Putting a password in the benchmark prompt, MCP tool call, task environment, or generic secret map would expose it to model context, process inspection, transcripts, or debugging output.
- A saved browser state may contain impersonation-capable cookies, headers, IndexedDB tokens, or passkey private material. Hiding the password while mishandling the state is not safe.
- Sharing one mutable account/profile across parallel runs can cause provider-profile corruption, application-level races, forced logout, fraud detection, or test interference.
- Browser recordings, screenshots, HAR, console output, downloads, and live inspection can expose private account data even when the password itself is protected.

**Impact:**

- Agents must repeatedly log in, which either fails on 2FA/CAPTCHA or tempts operators to put passwords in prompts.
- A shared long-lived Chrome profile would turn one prompt injection into broad, persistent account compromise.
- Cloud and local behavior would diverge unless both use the same reference/lease/materialization semantics.
- Authenticated browser evaluation cannot be considered production-safe until credential, egress, concurrency, and artifact authority are explicit.

## Goals

**Primary Goal:** Let an AILANG browser run start authenticated from a named, versioned profile while ensuring the AI model never receives the password or canonical profile material and ordinary runs cannot mutate the canonical identity.

**Success Metrics:**

- Zero raw passwords, session tokens, cookies, hosted context IDs, or decrypted profile bytes in prompts, MCP tool arguments, ordinary task metadata, result JSON, logs, traces, or public artifact manifests.
- 100% of profile use emits safe audit data: profile alias/hash, version, lease ID, principal, run/chain/stage, allowed origins, mode, decision, and timestamps.
- Local and Browserbase sessions consume the same `AuthProfileRef` contract and receive disposable or non-persisting session state.
- A 20-run isolation test proves an ordinary run cannot alter the next run's canonical profile state.
- One-writer lease enforcement has deterministic conflict behavior; parallel state-changing tests use different account/profile leases.
- Expiry, revocation, 2FA/refresh-required, scope denial, and cleanup failure are structured and leave no decrypted local material behind.
- Authenticated runs default to recording/export denial until the artifact-data policy explicitly allows each artifact class.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| The model never receives a password; login/bootstrap is trusted control-plane work | Determines the fundamental leak boundary and rules out generic model-driven password filling | human | design | high |
| Browser state is classified as a credential, not a normal artifact | Cookies/tokens can impersonate the account even if the password stays hidden | human | design | high |
| Ordinary runs load an immutable snapshot / `persist:false` context and never write back | Prevents profile poisoning and cross-run persistence from an injected or faulty agent | human | design | high |
| Canonical refresh/write-back is a separate audited workflow | Makes persistence an explicit authority rather than a side effect of session teardown | human | design | high |
| One writer lease per profile; account pools for parallel mutable work | Determines concurrency semantics and prevents application-level test collisions | human | design | high |
| Authenticated sessions deny recordings and unrestricted destinations by default | Private page content is as sensitive as the login state and can be exfiltrated by navigation | human | design | high |
| Local storage-state blobs and Browserbase Contexts share one logical contract, not one physical format | Preserves provider neutrality while respecting provider-specific secure storage | agent | compile | med |

### Design Freeze

- [x] Passwords are resolved and used only by a trusted bootstrap/refresh worker outside the AI/MCP loop.
- [x] Cookies, tokens, IndexedDB auth data, and virtual passkeys receive credential-grade storage and redaction.
- [x] Ordinary runs are copy-on-read/non-persisting; canonical profile mutation requires an explicit refresh operation.
- [x] A profile has at most one writer lease; state-changing parallelism uses an account pool.
- [x] Authenticated sessions fail closed when egress or artifact policy is absent.
- [x] A dedicated low-privilege automation account is preferred; a human's everyday browser profile is never imported.

## Threat Model

### Protected assets

- Passwords, OTP seeds, recovery codes, API tokens, passkey private keys.
- Authenticated cookies, local storage, IndexedDB, session storage, service-worker state, and saved form data.
- Hosted-provider Context IDs and local encrypted-blob object references.
- Private page content captured in screenshots, video, HAR, console, downloads, or live inspection.
- Canonical profile integrity and the external website account's server-side state.

### Adversaries and failures

- Prompt injection instructing the agent to reveal credentials or navigate to an attacker origin.
- Model/tool transcript capture of password-fill arguments.
- A compromised or buggy MCP/browser subprocess reading files outside its run directory.
- Parallel runs racing on one profile or account.
- A run changing account settings, deleting data, or persisting attacker-controlled state.
- Provider/API/logging failures returning sensitive response content.
- Operator mistakes: binding a personal profile, enabling recording, or marking null policy as allow-all.

### Explicit limits

- A valid authenticated session necessarily grants the browser authority to act as that account on allowed origins. This design narrows and audits that authority; it cannot make a compromised session harmless.
- Origin allowlists in Playwright are application controls, not complete network containment. Production authenticated sessions remain blocked on the browser-egress follow-up.
- A website can expire or revoke sessions at any time. Refresh behavior is structured but not deterministic.

## Solution Design

### Overview

Introduce an `AuthProfileBroker` below the eval harness and above provider-specific storage. Callers select a safe alias such as `crm-readonly-eu`; the broker resolves it to a versioned profile record, acquires a lease, validates task policy, and returns opaque materialization instructions to the provider.

The agent sees only an already-authenticated browser and a safe profile alias/hash in the result. It cannot call the vault, read the state file, obtain a hosted context ID, or request persistence. The controller destroys materialized state and releases the lease on every exit path.

### Architecture

```text
operator / refresh scheduler
       │
       ▼
trusted profile bootstrap ── password ref ──► 1Password / Secret Manager
       │                                      (value never enters model)
       ▼
canonical profile registry
  ├─ safe metadata + policy + version
  ├─ local: envelope-encrypted storage-state blob
  └─ cloud: opaque Browserbase Context reference
       │ acquire lease + validate scope
       ▼
browser session controller
  ├─ local: decrypt to 0600 disposable file → Playwright --storage-state
  └─ cloud: create session with Context + persist=false
       │
       ▼
AI + Playwright MCP (already authenticated; no vault/password/profile-write authority)
       │
       ▼
cleanup: export only policy-allowed artifacts → shred/delete materialization → release lease
```

### Core Types

The exact Go names may vary, but these semantics are frozen:

```go
type AuthProfileRef struct {
    Alias   string // safe operator-defined name
    Version string // immutable version; "latest" resolves before run start
}

type AuthProfilePolicy struct {
    AllowedOrigins   []string
    AccountClass     string // readonly | mutable | privileged
    MaxConcurrent    int
    AllowArtifacts   []string
    AllowHumanTakeover bool
    ExpiresAt        time.Time
}

type ProfileLease struct {
    SafeID    string
    ProfileHash string
    Version   string
    Mode      string // read | refresh
    ExpiresAt time.Time
}

type AuthProfileBroker interface {
    Resolve(ctx context.Context, ref AuthProfileRef) (SafeProfile, error)
    Acquire(ctx context.Context, profile SafeProfile, run RunIdentity, mode LeaseMode) (ProfileLease, error)
    Materialize(ctx context.Context, lease ProfileLease, provider string, dst string) (SensitiveProfileMaterial, error)
    Release(ctx context.Context, lease ProfileLease) error
    Revoke(ctx context.Context, ref AuthProfileRef, reason string) error
}
```

Public structs contain aliases, hashes, policy, version, expiry, and safe audit identity only. Provider context IDs, object URLs, decrypted state, cookies, and passwords live in opaque sensitive values with explicit materialization APIs and redacted formatting/JSON.

### Lifecycle

1. **Resolve**: Convert `alias@version` to immutable safe metadata. `latest` is resolved once and recorded as a concrete version.
2. **Preflight**: Check expiry/revocation, provider compatibility, account class, exact-origin policy, artifact policy, human-takeover policy, and required egress boundary.
3. **Acquire lease**: Read-only profiles may permit bounded concurrency only when the external task is declared non-mutating. Mutable/refresh mode is exclusive.
4. **Materialize**: Local decrypts to a session-owned `0600` file; Browserbase resolves an opaque Context ID inside the adapter. Neither value enters generic task metadata.
5. **Run**: Playwright MCP starts already authenticated. The model has browser action authority only for the declared origins and task duration.
6. **Export**: Deny by default. Export only artifact classes explicitly allowed by profile policy, with private access classification and retention metadata.
7. **Destroy**: Remove decrypted files/profile copies before releasing the lease. Cleanup failure is a high-severity structured result.
8. **Release**: End provider session and lease. Ordinary run state is discarded.

### Bootstrap and Refresh Modes

#### Manual bootstrap (recommended v1)

- Operator launches a separately labeled, headful trusted session with recording disabled.
- Operator signs into a dedicated automation account and completes CAPTCHA/2FA.
- Trusted bootstrap captures browser state or persists a hosted context.
- Broker stores a new immutable version, records origin/account policy, and closes the session.
- This session is not an eval and is marked non-comparable.

The password may remain entirely inside the operator/password-manager interaction; AILANG does not need to receive it.

#### Automated refresh (site-specific, later in v1)

- Trusted refresh worker resolves a password reference through `internal/secrets`/1Password or a cloud secret principal.
- A site-specific login adapter fills credentials outside model/MCP tool calls with recording and ordinary tracing disabled.
- OTP/approval is obtained through the existing human approval path when needed.
- On successful post-login verification, the worker creates a new immutable profile version and revokes or retires the old version.

Generic AI-authored login steps are forbidden. A password value must not be passed as a Playwright MCP `browser_type` argument because tool inputs are part of the AI/tool transcript.

### Provider Mapping

#### Local Playwright

- Canonical state is an envelope-encrypted Playwright storage-state blob, never a persistent personal Chrome directory.
- The broker decrypts it into a unique session directory with owner-only permissions.
- The provider adds `--storage-state <disposable-path>` alongside `--isolated`.
- The path may appear in the child argv; its contents and canonical object reference may not.
- Cleanup deletes the materialized file and session directory on success, error, cancellation, and timeout.

#### Browserbase

- Canonical state is a Browserbase Context referenced by an encrypted provider-private ID.
- Ordinary sessions load the Context with `persist: false` so run mutations are not committed.
- Only the trusted refresh workflow may create a session with `persist: true`.
- Context reuse is serialized for mutable work; the broker observes the provider synchronization delay before publishing a refreshed version.
- Context deletion/revocation is exposed through the broker, not to the model.

### Account and Concurrency Policy

One password/account is acceptable only when all of the following hold:

- it is a dedicated, least-privilege automation account;
- tasks are read-only or serialized;
- the site permits automation and concurrent/session behavior is understood;
- compromise can be contained by revoking sessions and rotating the account;
- no personal inbox, password vault, billing, or administrative authority is reachable.

Parallel state-changing runs use an account pool. The broker leases one profile/account per worker and records the account class, not the username. A profile may set `MaxConcurrent=1` even for read mode when the website invalidates simultaneous logins.

### Write-Back Policy

Ordinary eval and agent sessions are always `read` mode and discard state. Refresh is an explicit operation with these requirements:

- separate capability/principal and approval policy;
- exclusive lease;
- origin allowlist and expected post-login assertions;
- recording disabled;
- new immutable version rather than in-place overwrite;
- atomic publish after verification;
- audit event and rollback pointer to prior version;
- old-version retirement and external session revocation policy.

### Stable Failure Categories

```text
browser_auth_profile_not_found
browser_auth_profile_expired
browser_auth_profile_revoked
browser_auth_lease_conflict
browser_auth_scope_denied
browser_auth_refresh_required
browser_auth_materialize_failed
browser_auth_writeback_denied
browser_auth_artifact_policy_denied
browser_auth_cleanup_failed
```

Provider error bodies and secret values remain private diagnostics. Public results carry category, safe profile hash/version, stage, and retryability only.

## Implementation Plan

### M1 — Contract, registry, and secret-safe values (~2 days)

- [ ] Add profile refs, safe metadata, policy, lease modes, stable failures, broker interface, and opaque materialization values.
- [ ] Add an in-memory/file test registry with immutable versions and revocation.
- [ ] Add serialization/formatting/logging tests covering passwords, cookies, state blobs, object references, and provider context IDs.
- [ ] Add safe profile identity to `BrowserRunManifest` without changing non-authenticated result semantics.

### M2 — Local encrypted storage-state path (~1.5 days)

- [ ] Define an envelope-encryption adapter; local development may use an OS-keychain/1Password-backed key, cloud uses a KMS-backed implementation.
- [ ] Materialize a `0600` disposable storage-state file under the session directory and pass it to pinned Playwright MCP.
- [ ] Prove decrypt/use/delete on success, error, cancellation, timeout, and crash-recovery audit.
- [ ] Add a two-run test proving mutations from run one do not enter run two.

### M3 — Browserbase Context adapter (~1.5 days)

- [ ] Add create/get/delete Context operations through the injectable Browserbase client.
- [ ] Store provider Context IDs only in encrypted/private provider records.
- [ ] Create ordinary sessions with context persistence disabled and refresh sessions with persistence enabled.
- [ ] Add stub tests for synchronization delay, expiry, deletion, provider errors, and context-ID redaction; add credential-gated live contract test.

### M4 — Lease, account-pool, policy, and audit (~1.5 days)

- [ ] Implement TTL leases with exclusive refresh/write mode and configurable read concurrency.
- [ ] Add safe account-class pool selection for parallel mutable tasks.
- [ ] Enforce allowed origins, egress-boundary requirement, artifact policy, takeover policy, expiry, and revocation before provisioning.
- [ ] Emit safe profile/lease audit events linked to run/chain/stage and guarantee release on every controller path.

### M5 — Bootstrap/refresh workflow and operations (~1.5 days)

- [ ] Add manual headful bootstrap command with explicit non-comparable labeling and recording disabled.
- [ ] Add a site-adapter interface for optional trusted automated refresh; do not ship a generic model-driven password form filler.
- [ ] Add rotation, revoke, audit, inspect-safe-metadata, and orphan-materialization commands.
- [ ] Document local 1Password/keychain and Cloud Run Secret Manager/KMS patterns, incident response, and account-pool guidance.

## Files to Modify/Create

Exact registry/storage backends may be split during sprint planning, but responsibilities are fixed.

**New files:**

- `internal/browser/auth/types.go` — refs, safe metadata, policies, leases, failures, opaque profile material (~280 LOC)
- `internal/browser/auth/broker.go` — resolve/acquire/materialize/release/revoke orchestration (~300 LOC)
- `internal/browser/auth/registry.go` — immutable metadata versions and provider-private records (~250 LOC)
- `internal/browser/auth/lease.go` — TTL/exclusive lease and account-pool allocation (~250 LOC)
- `internal/browser/auth/envelope.go` — pluggable encrypted-blob interface and local/KMS boundary (~220 LOC)
- `internal/browser/auth/*_test.go` — redaction, versioning, lease, policy, materialization, and cleanup tests (~850 LOC)
- `cmd/ailang/browser_profile.go` — bootstrap, refresh, inspect, revoke, and audit CLI (~350 LOC)
- `docs/docs/guides/evaluation/browser-auth-profiles.md` — operator/security guide (~350 lines)

**Modified files:**

- `internal/browser/types.go` — typed auth profile input and safe manifest identity (~50 LOC)
- `internal/browser/controller.go` — lease/materialize/destroy/release lifecycle (~100 LOC)
- `internal/browser/local/playwright.go` — disposable `--storage-state` materialization (~60 LOC)
- `internal/browser/browserbase/client.go` — Context lifecycle and persistence mode (~160 LOC)
- `internal/eval_harness/browser_sessions.go` — profile selection and comparability policy (~80 LOC)
- `cmd/ailang/eval_suite.go`, `cmd/ailang/help.go` — safe profile alias/version flags and help (~40 LOC)

## Examples

### Example 1: Read-only local authenticated run

```bash
# Bootstrap is a trusted operator workflow, not an eval/model turn.
ailang browser-profile bootstrap crm-readonly-eu \
  --provider local-playwright \
  --origins https://crm.example.com \
  --account-class readonly \
  --no-recording

# The eval receives an already-authenticated disposable context.
ailang eval-suite --agent \
  --benchmarks crm_readonly_fixture \
  --browser-provider local-playwright \
  --browser-profile crm-readonly-eu@latest
```

The result records the resolved profile version/hash, never `latest`, cookies, or the canonical object reference.

### Example 2: Browserbase ordinary use and explicit refresh

```bash
# Ordinary runs load the hosted Context but cannot persist changes.
ailang eval-suite --agent \
  --benchmarks private_dashboard_fixture \
  --browser-provider browserbase \
  --browser-profile dashboard-readonly@v7

# Refresh is a different operation/principal and publishes v8 after checks.
ailang browser-profile refresh dashboard-readonly@v7 \
  --publish-version v8 \
  --reason session-expired
```

### Example 3: Parallel mutable workflow

```yaml
browser_auth:
  account_pool: shop-test-users
  account_class: mutable
  leases: 8
  writeback: deny
```

The broker allocates eight different test-account profiles. It never opens eight sessions on one shared mutable account.

## Success Criteria

- [ ] Password values never enter model/MCP input, argv, prompts, tasks, results, traces, logs, or normal artifacts.
- [ ] Browser state and hosted Context identifiers are secret-classified, encrypted/private, and redacted across all public presentations.
- [ ] Local and Browserbase authenticated runs implement the same resolve/lease/materialize/destroy/release semantics.
- [ ] Ordinary runs cannot persist profile changes; a two-run test proves canonical state is unchanged.
- [ ] Refresh publishes a new immutable version only after post-login verification and an exclusive lease.
- [ ] Expired/revoked/missing/leased/scope-denied profiles fail before browser provisioning.
- [ ] Account pools prevent shared-account mutation races under parallel execution.
- [ ] Authenticated sessions fail closed without explicit egress and artifact policies.
- [ ] Cleanup/crash audit finds no decrypted state outside active session directories.
- [ ] Unit, integration, race, lint, boundary, file-size, and docs-build gates pass.
- [ ] Credentialed local and Browserbase live tests are recorded as deployment evidence, never fabricated by CI.

## Testing Strategy

**Unit tests:**

- Opaque state/context values redact under `fmt`, JSON, errors, recursive diagnostics, and OTEL projection.
- Immutable version resolution, revocation, expiry, TTL lease acquisition/release, and deterministic conflicts.
- Policy preflight for account class, origins, artifacts, human takeover, egress readiness, and write-back.
- Envelope decrypt failure, partial file creation, permissions, idempotent destruction, and crash audit.
- Browserbase Context request/response/error mapping and `persist` mode against an HTTP stub.

**Integration tests:**

- Local fixture signs into a hermetic site, captures state in trusted bootstrap, then runs two isolated authenticated sessions.
- Run one changes cookies/local storage; run two begins from the canonical snapshot and cannot observe the mutation.
- Cancellation kills MCP/Chromium, deletes decrypted state, releases the lease, and retains only allowed safe artifacts.
- Browserbase opt-in live test creates Context, refreshes, loads with persistence disabled, verifies no ordinary-run write-back, and deletes Context.
- Parallel account-pool test proves unique profile allocation and deterministic exhaustion.

**Security/operational tests:**

- Secret scan prompt, argv, environment projection, task dump, results, logs, spans, screenshots, recording, HAR, console, downloads, and crash files.
- Redirect/DNS/WebSocket/service-worker exfiltration tests are owned by the egress follow-up and are mandatory before production authentication.
- Rotation/revocation drill: revoke profile and external website sessions, rotate password, refresh new version, prove old version fails closed.
- 2FA/expired state flow produces `browser_auth_refresh_required`, not a model-visible password prompt.

## Deferred Decisions

- Encrypted blob backend layout (GCS+CMEK, database blob, or external vault attachment) — agent may choose after measuring state size; values must not be stored directly in ordinary config/Firestore rows.
- Local key protector (macOS Keychain vs 1Password item/document) — human at deployment review; broker interface remains stable.
- Lease persistence backend (SQLite locally, Firestore/Redis/cloud database remotely) — agent may choose while preserving compare-and-set semantics and TTL audit.
- Site-specific post-login assertion format — agent may choose a minimal deterministic selector/URL/cookie predicate schema.
- Automated refresh scheduling frequency — operator policy; event-driven expiry detection is sufficient for v1.

## Non-Goals

**Not attempted in this feature:**

- Giving the AI model raw passwords, OTP seeds, recovery codes, or password-manager access.
- Importing or automating the operator's normal Chrome profile.
- Treating encryption at rest as protection from a fully compromised runtime principal.
- Letting arbitrary model-authored JavaScript implement login/refresh.
- Solving CAPTCHA or bypassing website terms/security controls.
- Persisting arbitrary run changes into canonical state.
- Building a general secrets manager; 1Password/Secret Manager/KMS remain sources of truth.
- Claiming Playwright origin flags fully contain browser traffic.
- Allowing authenticated production use before egress and artifact-data policies are implemented.

## Timeline

**Week 1** (~28–34 hours):

- M1 profile contract/registry/redaction
- M2 local encrypted storage-state materialization
- Begin M3 Browserbase Context adapter

**Week 2** (~24–32 hours):

- Complete M3 Context lifecycle/live gate
- M4 leases/account pools/policy/audit
- M5 bootstrap/refresh CLI, incident runbook, docs, repository gates

**Total: ~52–66 hours across 7–9 engineering days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Saved state is treated as less sensitive than a password | Critical | Secret-class opaque types, encrypted/private storage, redaction tests, least-privilege accounts, revocation drills. |
| Prompt injection exfiltrates authenticated data to another origin | Critical | Exact-origin policy plus fail-closed requirement on a real browser egress boundary; not Playwright flags alone. |
| Agent poisons canonical profile | High | Immutable snapshots and `persist:false`; only trusted refresh principal can publish a new version. |
| Recording/HAR/screenshot leaks private content | High | Deny artifacts by default for auth profiles; explicit artifact classes, private ACL, retention policy follow-up. |
| Shared account races or website forces logout | High | One-writer lease, configurable max concurrency, account pool for mutation, consistent provider region. |
| Profile expires during a run | Medium | Preflight validation, structured refresh-required failure, bounded retry only after a new version exists. |
| Decrypted local state survives crash | High | Session-owned `0700` directories/`0600` files, startup orphan audit, bounded cleanup, short TTL, encrypted canonical source. |
| Hosted Context ID leaks | High | Provider-private encrypted record, opaque wrapper, no task/result serialization, API-key/project scoping. |
| One account has excessive authority | High | Dedicated least-privilege automation accounts, per-site profile, no personal/billing/admin surfaces, account-class policy. |

## Verification Log

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | The shipped browser contract reserves profile identity but providers do not materialize it | Read `internal/browser/types.go`, `internal/browser/local/playwright.go`, and `internal/browser/browserbase/client.go`; `rg -n 'ProfileRef|storage-state|context' internal/browser` | `SessionSpec.ProfileRef/ProfileHash` exist; local uses isolated fresh state and Browserbase create omits persistent Context configuration. |
| V2 | No auth-profile broker, lease, version, refresh, or revoke implementation exists | `rg -n -i 'AuthProfile|ProfileLease|browser_auth_|storageState' internal/browser internal/eval_harness cmd/ailang` | No implementation found; this is a new subsystem rather than duplicate code. |
| V3 | AILANG already has a backend-agnostic secret resolver with a 1Password implementation | Read `internal/secrets/onepassword.go`, `internal/effects/secret.go`, and M-SECRET-EFFECT | Confirmed reusable vault/ref/audit precedent; browser bootstrap must remain outside the AI effect/tool loop. |
| V4 | Playwright authenticated state contains impersonation-capable material and supports isolated reuse | [Playwright authentication guide](https://playwright.dev/docs/auth) and [`BrowserContext.storageState`](https://playwright.dev/docs/api/class-browsercontext#browser-context-storage-state) | Confirmed cookies/local storage/IndexedDB and optional virtual credentials; Playwright explicitly warns the file is sensitive. |
| V5 | Pinned Playwright MCP supports loading storage state in isolated mode | `npx -y @playwright/mcp@0.0.79 --help` | Confirmed `--isolated` and `--storage-state <path>` options. |
| V6 | Browserbase Contexts persist auth state and support non-persisting ordinary sessions | [Browserbase Contexts](https://docs.browserbase.com/platform/browser/core-features/contexts) | Confirmed Context reuse, encrypted-at-rest claim, `persist:true` write-back, `persist:false` discard behavior, sync delay, and single-context concurrency guidance. |
| V7 | Browser artifacts and inspection are enabled in the current neutral-session layer without auth-specific data policy | Read `internal/eval_harness/browser_sessions.go`, provider export methods, and `docs/docs/guides/evaluation/browser-sessions.md` | Confirmed current trace/video request and provider artifact export; auth profiles need deny-by-default artifact policy. |
| V8 | Browser origin flags are not a complete exfiltration boundary | Pinned MCP `--help` warning plus M-REMOTE-BROWSER-SESSION-PROVIDERS and M-NET-EFFECT-PROXY-BOUNDARY | Confirmed redirects and browser/WebSocket/subprocess traffic require a separate boundary. |
| V9 | Proposed stable browser-auth categories are unallocated | `rg -n 'browser_auth_profile_|browser_auth_lease_|browser_auth_scope_|browser_auth_refresh_|browser_auth_materialize_|browser_auth_writeback_|browser_auth_artifact_|browser_auth_cleanup_' .` before this doc | No implementation allocation found; names are reserved by this design. |
| V10 | Automatic related-doc search returned high-score false positives | Read the six generated matches: package updates, cryptorand, module execution, oracle adequacy, call sugar, structural reflection | All are semantically unrelated; duplicate gate passes. Manual search found the browser-session and Secret-effect docs below. |

No AILANG language support/unsupported claim is made, and this design does not touch parser/typechecker/codegen paths, so no `ailang check` language probe or Conflict Surface section is required.

## Related Documents

**Direct dependencies and precedents:**

- [M-REMOTE-BROWSER-SESSION-PROVIDERS](../v0_33_3/m-remote-browser-session-providers.md) — provides the neutral lifecycle, opaque connection values, local/Browserbase adapters, and reserved profile reference.
- [M-SECRET-EFFECT](../v0_26_0/m-secret-effect-remote-approval.md) — shipped/implemented 1Password resolver, safe reference, human approval, and audit precedent; browser bootstrap consumes this control-plane capability without placing secrets in model context.
- [M-SECRET-REMOTE-APPROVAL-WIRING](../v0_26_0/m-secret-remote-approval-wiring.md) — notification/approval operations for 2FA or refresh gates.
- [M-PERMISSION-MODEL](../../deferred/m-permission-model.md) — future typed authority-tier alignment for credential/profile operations.
- [M-AGENT-SAFE-RUNNER](../../planned/v1_1_0/m-agent-safe-runner.md) — operator-pinned policy and hostile-caller boundary precedent.
- [M-NET-EFFECT-PROXY-BOUNDARY](../../planned/v0_33_1/m-net-effect-proxy-boundary.md) — adjacent HTTP policy that explicitly does not yet contain browser/WebSocket traffic.
- [M-EVAL-EXPERIMENT-REGISTRY](../../planned/v0_31_0/m-eval-experiment-registry.md) — future home for named profile/provider/tool-version comparison experiments.

## Suggested Follow-Up Design Queue

These are separate trust or qualification boundaries and should not be hidden inside the auth-profile sprint:

1. **P0 — M-BROWSER-EGRESS-BOUNDARY**: enforce destination policy below Playwright across DNS resolution, redirects, WebSockets, service workers, downloads/uploads, proxy bypass, and browser subprocesses. Authenticated production sessions are blocked on this.
2. **P0 — M-BROWSER-ARTIFACT-DATA-POLICY**: classify screenshots/video/HAR/console/download/live-view data, private ACLs, redaction, retention/deletion, incident hold, and per-profile artifact allowlists. Authenticated recording is blocked on this.
3. **P1 — M-BROWSER-PROVIDER-QUALIFICATION**: run the fixed local/Browserbase comparison, 50-session local capacity sweep, 20-session Cloud Run soak, leak audits, cost join, and SLO acceptance. This closes the remaining deployment evidence from M-REMOTE-BROWSER-SESSION-PROVIDERS.
4. **P1 — M-BROWSER-ACCOUNT-FACTORY**: site-specific creation/reset/seed/retire of disposable test accounts when a static account pool becomes operationally expensive. Start only after real pool demand.
5. **P2 — M-BROWSER-PROVIDER-EXPANSION**: qualify self-hosted Steel and AWS AgentCore Browser against the frozen provider/auth contracts; add Browserless/Cloudflare only from measured requirements.
6. **P2 — M-BROWSER-WORKFLOW-REPLAY**: controlled page snapshots/network fixtures and action replay for determinism without replaying credentials or private content.

The first two should receive full design docs before authenticated production access. The remaining items stay queued until local/Browserbase deployment evidence identifies a concrete need.

## References

- [Playwright authentication](https://playwright.dev/docs/auth)
- [Playwright `BrowserContext.storageState`](https://playwright.dev/docs/api/class-browsercontext#browser-context-storage-state)
- [Playwright MCP](https://github.com/microsoft/playwright-mcp)
- [Browserbase Contexts](https://docs.browserbase.com/platform/browser/core-features/contexts)
- [Browserbase 1Password integration](https://docs.browserbase.com/integrations/1password/introduction)
- [Design Axioms](/docs/references/axioms)

## Future Work

After the P0 egress and artifact policies land, add provider-qualified authenticated fixtures using dedicated test accounts. Later adapters must consume the same `AuthProfileBroker`; no provider may introduce raw password fields or silently persistent sessions into `SessionSpec`.

---

**Document created**: 2026-08-25
**Last updated**: 2026-08-25

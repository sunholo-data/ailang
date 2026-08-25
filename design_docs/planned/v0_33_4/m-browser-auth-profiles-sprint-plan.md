# Sprint Plan: Persistent Browser Auth Profiles

**Status:** Planned — ready for execution

## Summary

Implement `M-BROWSER-AUTH-PROFILES` as described in
[`m-browser-auth-profiles.md`](m-browser-auth-profiles.md): persistent
authenticated browser identity for AILANG eval/agent runs, where the model never
receives a password, never receives canonical profile material, and cannot
mutate the canonical identity.

The sprint builds an `AuthProfileBroker` below the eval harness and above the
provider adapters shipped by `M-REMOTE-BROWSER-SESSION-PROVIDERS`. Ordinary runs
lease an immutable profile version, materialize a disposable copy, and destroy it
on every exit path. Write-back is a separate audited operation.

**Sprint ID:** M-BROWSER-AUTH-PROFILES
**Target:** v0.33.4
**Duration:** 8 engineering days (design estimate 7–9)
**Base branch:** `sprint/m-browser-auth-profiles` (branched from local `dev`, merged with `origin/dev`)
**Dependencies:** `M-REMOTE-BROWSER-SESSION-PROVIDERS` (in local `dev`, **not yet on `origin/dev`**), `internal/secrets` resolver, Secret approval/audit path
**Risk Level:** High — this sprint handles credential-grade material

## Current Status Analysis

### Completed Recently

- ✅ `internal/browser` ships a provider-neutral contract: `SessionSpec`,
  `SensitiveConnection` (opaque, `Materialize`-only), `BrowserRunManifest`,
  ten stable `browser_*` failure categories, and `SanitizeDiagnostics`.
- ✅ Local Playwright MCP adapter (`@playwright/mcp@0.0.79`, `--isolated`) and
  Browserbase REST/CDP adapter both exist with hermetic tests.
- ✅ `SessionSpec.ProfileRef` / `ProfileHash` are **reserved but unimplemented** —
  neither provider materializes authenticated state today (design doc V1/V2).
- ✅ Merged base verified green before sprint start: `internal/browser/...` and
  `internal/eval_harness/...` pass.

### Velocity

- Sibling sprint `M-REMOTE-BROWSER-SESSIONS` estimated 2,450 LOC and delivered
  2,435 across 5 milestones — the estimate was accurate to 0.6%, so the same
  estimation method is reused here.
- This sprint is larger (3,350 LOC) because credential handling requires
  redaction, isolation, race, and fail-closed tests that outnumber implementation.
- Planning basis: the design document's own per-milestone hour estimates,
  scaled by the sibling sprint's realized LOC-per-milestone.

### Remaining from Design Doc

- ⏳ M1 contract, registry, immutable versions, redaction: ~900 LOC.
- ⏳ M2 envelope encryption + local disposable storage-state: ~600 LOC.
- ⏳ M3 Browserbase Context lifecycle with `persist` mode: ~550 LOC.
- ⏳ M4 leases, account pools, policy preflight, audit: ~600 LOC.
- ⏳ M5 bootstrap/refresh CLI, eval wiring, operator guide: ~700 LOC.

## Execution Strategy

Dependency analysis of the five milestones against the files each touches:

| Milestone | Primary files | Depends on |
|-----------|---------------|------------|
| M1 | `internal/browser/auth/{types,registry}.go`, `internal/browser/types.go` | — |
| M2 | `internal/browser/auth/envelope.go`, `internal/browser/local/playwright.go` | M1 |
| M3 | `internal/browser/browserbase/context.go`, `client.go` | M1 |
| M4 | `internal/browser/auth/{lease,broker,policy}.go`, `controller.go` | M1 |
| M5 | `cmd/ailang/browser_profile.go`, `eval_suite.go`, docs | M1–M4 |

**Waves:**

- **Wave 1 (sequential):** M1 — the frozen contract every other milestone
  compiles against. Executed in-session, not delegated.
- **Wave 2 (parallel):** M2 ∥ M3 — the two provider adapters are file-disjoint
  (`local/` vs `browserbase/`) and both consume M1's types read-only.
- **Wave 3 (sequential):** M4 — owns `broker.go` and rewires `controller.go`,
  which both M2 and M4 would otherwise touch. Serialized to avoid the conflict.
- **Wave 4 (sequential):** M5 — CLI, eval wiring, and docs over the finished stack.

Expected speedup over fully sequential: ~1.3x. The coupling through
`broker.go`/`controller.go` is real, so the plan does not claim more.

## Proposed Milestones

### Milestone 1: Auth Profile Contract, Registry, and Secret-Safe Values

**Goal:** Freeze the types every later milestone compiles against, and prove that
no credential-grade value can escape through serialization, formatting, errors,
or diagnostics.

**Estimated:** 420 LOC implementation + 480 LOC tests = 900 LOC
**Duration:** 2 days
**Dependencies:** none

**Tasks:**

- Day 1: Add `internal/browser/auth` with `AuthProfileRef`, `SafeProfile`,
  `AuthProfilePolicy`, `ProfileLease`, `LeaseMode`, and the `AuthProfileBroker`
  interface.
- Day 1: Add `SensitiveProfileMaterial` as an opaque type following the
  `SensitiveConnection` precedent — `Materialize` is the only extraction API.
- Day 1: Allocate the ten stable `browser_auth_*` failure categories reserved by
  the design doc (V9 confirmed they are unallocated).
- Day 2: Add the immutable-version registry with revocation, expiry, and
  `latest` → concrete-version resolution.
- Day 2: Extend `isSensitiveKey`/`SanitizeDiagnostics` to cover storage state,
  provider context IDs, and lease material; add safe profile identity to
  `BrowserRunManifest`.

**Acceptance Criteria:**

- [x] `AuthProfileRef`, `SafeProfile`, `AuthProfilePolicy`, `ProfileLease`,
      `LeaseMode`, and `AuthProfileBroker` exist with the frozen semantics from
      the design doc's Core Types section.
- [x] All ten `browser_auth_*` categories are defined, unique, and distinct from
      the existing `browser_*` set.
- [x] `SensitiveProfileMaterial` redacts under `String()`, `Error()`,
      `MarshalJSON`, `%v`, `%+v`, and `%#v`; `Materialize()` is the only path to
      the bytes, and a test asserts each of those six presentations.
- [x] Registry versions are immutable: republishing an existing version fails,
      `latest` resolves to a concrete version string, and revoked/expired
      profiles resolve to a terminal structured failure.
- [x] `BrowserRunManifest` carries alias, profile hash, and resolved version, and
      a test asserts the serialized manifest never contains `latest`, cookies,
      object references, or provider context IDs.
- [x] Existing non-authenticated browser tests pass **unmodified** — the contract
      extension is additive.
- [x] `make test` and `make lint` clean.

**Risks:**

- A Go value can leak through a nested map or a `%+v` on a containing struct.
  Mitigation: opaque type with no exported fields, plus recursive sanitizer tests
  that assert on the containing struct, not just the value.
- Extending `isSensitiveKey` can over-redact legitimate manifest fields.
  Mitigation: assert both directions — secrets redacted, safe identity preserved.

### Milestone 2: Local Encrypted Storage-State Materialization

**Goal:** Decrypt a canonical storage-state blob into a disposable, owner-only
file for one run, hand it to pinned Playwright MCP, and destroy it on every exit
path.

**Estimated:** 280 LOC implementation + 320 LOC tests = 600 LOC
**Duration:** 1.5 days
**Dependencies:** M1

**Tasks:**

- Day 3: Define the envelope-encryption adapter interface (`Seal`/`Open`) with a
  pluggable key protector; ship an AES-256-GCM local implementation and leave the
  KMS implementation behind the same interface.
- Day 3: Materialize into a `0700` session directory as a `0600` file; wire
  `--storage-state <path>` alongside the existing `--isolated`.
- Day 4: Prove decrypt/use/destroy on success, error, cancellation, and timeout;
  add the startup orphan audit for materializations surviving a crash.
- Day 4: Add the two-run isolation test.

**Acceptance Criteria:**

- [x] Envelope adapter seals/opens with a per-blob nonce; a tampered ciphertext
      fails closed as `browser_auth_materialize_failed`, never as a partial read.
- [x] Materialized file is `0600` inside a `0700` session-owned directory; a test
      asserts the mode bits, not just the path.
- [x] Local provider passes `--storage-state` with `--isolated`; the path may
      appear in child argv, the contents and canonical object reference may not
      (asserted against the generated MCP config).
- [x] Destruction is idempotent and runs on success, error, cancellation, and
      timeout; a cleanup failure reports `browser_auth_cleanup_failed` **without**
      masking the primary failure.
- [x] Two-run isolation test: run 1 mutates its materialized state, run 2
      materializes from canonical and cannot observe the mutation.
- [x] Startup orphan audit finds and removes stale materializations and records a
      structured audit event.
- [x] `make test` and `make lint` clean. (M2 verified scoped: `go test [-race]`
      for `internal/browser`, `.../auth`, and `.../local` all pass,
      `golangci-lint` reports 0 issues, `check-file-sizes` passes, and
      `go build ./...` plus `GOOS=windows go vet` are clean. Repo-wide `make test`
      was not green at M2 commit time because M3's `browserbase` tests were still
      red in the shared worktree; that is M3's milestone, not this one.)

**Risks:**

- A crash between decrypt and register leaves plaintext on disk. Mitigation:
  session-owned directory naming plus the startup orphan audit.
- Windows CI has no POSIX mode bits. Mitigation: guard mode-bit assertions per
  the repo's platform convention (sprint-executor rule #10).

### Milestone 3: Browserbase Context Adapter

**Goal:** Load a hosted Context for ordinary runs without allowing write-back,
and keep the Context ID out of every serialized surface.

**Estimated:** 240 LOC implementation + 310 LOC tests = 550 LOC
**Duration:** 1.5 days
**Dependencies:** M1

**Tasks:**

- Day 3: Add create/get/delete Context operations through the existing
  injectable bounded HTTP client.
- Day 3: Store provider Context IDs only in encrypted provider-private records.
- Day 4: Send `persist:false` for ordinary sessions and `persist:true` only for
  refresh mode; refuse a persist request under a read lease before the HTTP call.
- Day 4: Add stub tests for sync delay, expiry, deletion, provider errors, and
  Context-ID redaction; keep the live contract test credential-gated.

**Acceptance Criteria:**

- [x] Context create/get/delete go through the injectable client with bounded
      timeouts; no new unbounded HTTP path is introduced.
- [x] Ordinary sessions send `persist:false`; only refresh mode sends
      `persist:true`; a read-mode lease requesting persistence is refused as
      `browser_auth_writeback_denied` **before** any HTTP request is issued.
- [x] Context IDs are absent from `SessionSpec`, `BrowserRunManifest`, JSON,
      logs, and errors; a redaction test asserts this on the containing structs.
- [x] Stub tests cover auth failure, quota, malformed response, timeout,
      provider synchronization delay, expiry, and deletion.
- [x] The live contract test stays credential-gated and is not a CI dependency.
- [x] `make test` and `make lint` clean.

**Risks:**

- Browserbase publishes a refreshed Context asynchronously. Mitigation: the
  broker observes the documented synchronization delay before publishing the new
  version; the delay is a tested, injectable parameter, not a sleep.

### Milestone 4: Lease, Account Pool, Policy Preflight, and Audit

**Goal:** Make concurrency and authority decisions deterministic and testable,
and guarantee the lease is released on every controller exit path.

**Estimated:** 300 LOC implementation + 300 LOC tests = 600 LOC
**Duration:** 1.5 days
**Dependencies:** M1 (compiles against M2/M3 adapters after Wave 2 integrates)

**Tasks:**

- Day 5: Implement TTL leases with compare-and-set semantics; exclusive
  refresh/write mode, bounded read concurrency.
- Day 5: Add account-class pool allocation for parallel mutable work.
- Day 6: Enforce the full policy preflight before any browser is provisioned.
- Day 6: Emit safe audit events and guarantee release on every controller path.

**Acceptance Criteria:**

- [x] Lease acquisition is compare-and-set; refresh/write mode is exclusive; read
      mode is bounded by `MaxConcurrent`; conflict is deterministic and returns
      `browser_auth_lease_conflict`.
- [x] Account-pool allocation gives distinct profiles per worker; exhaustion is a
      deterministic structured failure, never silent reuse of an in-use account.
- [x] Preflight rejects missing, expired, revoked, scope-denied, artifact-denied,
      and takeover-denied profiles **before** a browser is provisioned — asserted
      by a fake provider that fails the test if it is ever called.
- [x] Fail-closed: an authenticated session with no artifact policy or no egress
      boundary is denied, and the denial names which policy was absent.
- [x] The lease is released on success, error, cancellation, timeout, and panic;
      a `-race` test with parallel workers proves no lease leaks.
- [x] Audit events carry alias, profile hash, version, lease safe ID, principal,
      run/chain/stage, allowed origins, mode, decision, and timestamps — and no
      secrets.
- [x] `make test`, `make lint`, and `go test -race ./internal/browser/...` clean.

**Risks:**

- A panic path can skip release. Mitigation: `defer`-based release in the
  controller plus an explicit panic-injection test.
- Read concurrency is unsafe for sites that invalidate simultaneous logins.
  Mitigation: `MaxConcurrent=1` is expressible and honored for read mode.

### Milestone 5: Bootstrap/Refresh CLI, Eval Wiring, and Operations

**Goal:** Give operators a trusted way to create, inspect, refresh, and revoke
profiles, wire profile selection into evals, and document the security model.

**Estimated:** 340 LOC implementation + 360 LOC tests/docs = 700 LOC
**Duration:** 1.5 days
**Dependencies:** M1, M2, M3, M4

**Tasks:**

- Day 7: Add `ailang browser-profile` with `bootstrap`, `refresh`, `inspect`,
  `revoke`, `audit`, and `gc` subcommands.
- Day 7: Add the site-adapter interface for optional trusted automated refresh.
- Day 8: Wire `--browser-profile alias@version` through eval-suite and result
  banking; update `help.go`.
- Day 8: Write the operator/security guide and run the full repository gates.

**Acceptance Criteria:**

- [ ] `bootstrap` runs headful with recording disabled and marks the session
      non-comparable; it is never an eval.
- [ ] A site-adapter interface exists for trusted automated refresh, and **no
      generic model-driven password form filler ships** — asserted by a test that
      no code path passes a secret value into an MCP tool argument.
- [ ] `refresh` publishes a new immutable version only after post-login
      verification under an exclusive lease, retires the old version, and records
      a rollback pointer.
- [ ] `--browser-profile alias@version` selects a profile in eval-suite; results
      record the resolved concrete version, never `latest`.
- [ ] The operator guide covers local keychain/1Password and Cloud Run Secret
      Manager/KMS, incident response, account pools, and the rotation drill.
- [ ] `make test`, `make lint`, `make fmt-check`, `make check-boundaries`, and
      `make check-file-sizes` all pass; the docs site builds.
- [ ] Windows CI scan done per sprint-executor rule #10 (path assertions,
      external binaries, golden rendering).

**Risks:**

- CLI additions can push `cmd/ailang` files past the 800-line gate. Mitigation:
  `browser_profile.go` is a new sibling file, and `check-file-sizes` runs before
  the milestone is declared done.
- The docs guide can drift from the shipped flags. Mitigation: the guide's
  command examples are copied from `help.go` output, not written from memory.

## Deferred to Follow-Up Designs

Per the design doc's Suggested Follow-Up Design Queue, these are **not** in this
sprint and authenticated production access stays blocked on the first two:

1. **P0 M-BROWSER-EGRESS-BOUNDARY** — destination policy below Playwright.
2. **P0 M-BROWSER-ARTIFACT-DATA-POLICY** — artifact classification and retention.
3. P1 M-BROWSER-PROVIDER-QUALIFICATION, P1 M-BROWSER-ACCOUNT-FACTORY,
   P2 M-BROWSER-PROVIDER-EXPANSION, P2 M-BROWSER-WORKFLOW-REPLAY.

This sprint implements the *fail-closed hooks* for (1) and (2) — an authenticated
session is denied when either policy is absent — but does not implement the
policies themselves.

## Sprint Success Criteria

- [ ] All five milestones pass their acceptance criteria.
- [ ] The design doc's Success Criteria list is satisfied or explicitly annotated
      as deferred with a reason.
- [ ] No password, cookie, storage-state byte, object reference, or provider
      Context ID appears in prompts, MCP tool arguments, argv, task metadata,
      result JSON, logs, traces, or public artifact manifests.
- [ ] Credentialed local and Browserbase live tests are recorded as deployment
      evidence or explicitly marked not-run — **never fabricated**.

# M-SECRET-EFFECT: Gated `Secret` Effect with 1Password-Backed Remote Human Approval

**Status**: Planned
**Target**: v0.26.0
**Priority**: P1 - Medium
**Estimated**: 2 weeks (Phase 1 + 2; Phase 3 optional)
**Dependencies**:
- M-TAINT-TYPES (value-labelled IFC, **shipped v0.16.0**) — the `<secret>` label + sink refinement + `Declassify` gate
- Contracts / SMT verification (`requires`/`ensures` + `ailang verify`, **shipped v0.8.0**) — static policy proofs
- Coordinator approval workflow (`ApprovalCheckpoint`, **shipped v0.6.4**) — the blocking human-decision rail
- Notify daemon (Pub/Sub → macOS + Discord, **shipped**) — the "push to my phone" channel
- M-PERMISSION-MODEL (typed permission tiers, **planned v0.23.0**) — soft dependency; `Secret` should slot in as a tier-gated effect, not a parallel mechanism

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is, at its core, an **A4 (Explicit Authority)** feature: it converts ambient, always-present secret access (today: `os.Environ()` inheritance, no gate) into an explicitly declared, human-approved, flow-controlled effect.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Secret *resolution* is inherently non-deterministic I/O, but it is confined to a declared effect; replay uses recorded approval decisions, not live reads |
| A2: Replayability | +1 | Each approval + resolution emits an audit span (request, ref, purpose, approver, decision); replay reads the recorded decision rather than re-prompting |
| A3: Effect Legibility | +1 | Secret access becomes a first-class `! {Secret}` row entry — previously invisible env-var reads are now declared |
| A4: Explicit Authority | +1 | **Principal feature.** No secret value materializes without an explicit, human-approved capability grant. Removes ambient authority. |
| A5: Bounded Verification | +1 | Type-checker rejects secret values flowing to disallowed sinks (`string{not secret}`); Z3 proves approval-policy contracts at compile time |
| A6: Safe Concurrency | 0 | No concurrency model changes; approval checkpoints already serialize |
| A7: Machines First | +1 | The reference (`op://…`) is machine-stable and safe to log/diff/embed; the *value* never enters the AI context until gated. Approval payload is structured JSON. |
| A8: Minimal Syntax | 0 | Reuses existing `! {Effect}` rows and `<label>` / `{not label}` syntax from M-TAINT-TYPES. No new surface syntax. |
| A9: Cost Visibility | +1 | Secret access (a high-risk action) is explicitly gated and metered via the existing effect-budget mechanism |
| A10: Composability | +1 | Composes with the existing effect system, taint labels, contracts, approval checkpoint, and the planned permission tiers |
| A11: Structured Failure | +1 | Typed `SecretDenied` / `SecretUnavailable` / `DeclassRequired` errors carry ref + purpose + reason |
| A12: System Boundary | +1 | The 1Password vault and the human's phone are explicit system boundaries; every crossing is logged |

**Net Score: +10** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No *implicit* nondeterminism — secret reads are confined to the declared `! {Secret}` effect; non-`! {Secret}` code remains pure/deterministic
- [x] A3 (Effects): No hidden side effects — the whole point is to make secret access a declared effect
- [x] A4 (Authority): No ambient access — removes the current ambient `os.Environ()` path for gated refs
- [x] A7 (Machines First): References are machine-analyzable; approval payloads are structured; not optimizing for human convenience at the expense of analysis

## Problem Statement

AILANG executors and AI providers acquire secrets **by value, ambiently, with no human gate and no flow control**:

**Current State:**
- `internal/executor/environment.go` `BuildEnvironment()` inherits the entire parent environment via `os.Environ()`. Every API key present in the daemon's environment is visible to every spawned task for its whole lifetime.
- `internal/ai/configdriven/auth.go` resolves auth via `requireEnv(...)` at call time — correct as far as it goes, but the value is a plaintext env var with no acquisition gate and no provenance tracking.
- There is **no** `op://` reference resolution anywhere in the codebase (grep: zero matches). Secrets are passed as values, not references.
- A secret value, once in the agent's context, can flow into logs, OTEL traces, model prompts, or network calls with nothing stopping it.

**What 1Password does and doesn't give us (verified June 2026):**
- ✅ Ships today: secret references (`op read "op://vault/item/field"`), `op run`, `op inject`, and **service accounts** (`OP_SERVICE_ACCOUNT_TOKEN`) for headless/non-interactive resolution.
- ✅ Ships today: *local* biometric approval (desktop app integration; SSH-agent per-use Touch ID). Strictly same-machine.
- ⚠️ **Does NOT ship**: remote "agent on a server requests a secret → I approve on my phone." 1Password's "Agentic Autofill" is device-local, Early-Access, Browserbase-only, browser-autofill-only, no CLI/SDK. The remote-approval-from-mobile capability is an [open 1Password feature request explicitly for agentic workflows](https://www.1password.community/discussions/developers/feature-request-remote-approval-for-op-cli--desktop-prompts-to-support-agentic-w/167877).

**The insight:** the half 1Password lacks — *remote human approval pushed to the operator's phone* — AILANG already owns (ApprovalCheckpoint + notify daemon). We build the bridge, not the vault.

**Impact:**
- Every autonomous task currently runs with the full ambient secret set; a prompt-injected or misbehaving agent can exfiltrate any of them.
- No per-use audit of which task accessed which credential for what purpose.
- The trust/authority boundary lives in env-var configuration (a machine concern) instead of at a human-usable approval surface (the operator's phone).

## Goals

**Primary Goal:** Make secret acquisition a declared, human-approved, flow-controlled `Secret` effect: an agent references a secret by `op://` URI, a remote approval is pushed to the operator's phone, and only on approval is the value resolved just-in-time, type-forbidden from leaking into logs/traces/disallowed sinks, and scrubbed after use.

**Success Metrics:**
- Zero secret *values* present in a task's environment until an approved `Secret` effect resolves one (eliminates blanket `os.Environ()` secret inheritance for gated refs).
- 100% of secret resolutions emit an audit span: `{ref, purpose, agent, chain, approver, decision, ts}`.
- A program that lets a `<secret>`-labelled value reach a `string{not secret}` sink (log/trace/Net) is a **compile-time type error** — verified by `ailang check`, not caught at runtime.
- A contract `requires { approved(ref) }` on a secret-consuming function is statically provable via `ailang verify`.
- End-to-end: a server-side task blocks on a secret, the operator's phone gets a Discord/macOS notification, one tap approves, the task resumes — demoable.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Trust model: A (service-account token on server) vs B (zero-standing-secret, resolve on operator device)** | A ships fast but keeps one standing credential on the server; B has no long-lived secret server-side but needs a secure return channel for the resolved value | human | design | high |
| Is `<secret>` a first-class taint label that **forbids flow to all `{not secret}` sinks** (logs, traces, Net) by default? | Determines whether leak-prevention is a static type guarantee or runtime-only redaction | human | design | high |
| Approval model: **synchronous blocking** (`RequestApproval` blocks the task) vs **async re-queue** (task suspends, resumes on approval event) | Blocking is simpler and matches existing checkpoint; async enables many parallel pending secrets | human | design | high |
| Resolution location: server runs `op read` (model A) vs operator device runs `op read` and returns value (model B) | Defines where the 1Password credential lives and the threat model | human | design | high |
| Granularity of approval cache: per-ref-per-task vs per-ref-per-chain vs TTL window (mirrors 1Password SSH-agent "remember approval") | UX vs blast radius tradeoff | agent | compile | med |
| `Secret` effect → permission tier mapping (coordination with M-PERMISSION-MODEL) | Avoids two competing gate mechanisms | agent | compile | med |
| **Approvals push transport (X)** — proposed **ntfy self-hosted** (lock-screen Approve/Deny, data stays ours); alternatives Pushover (retry-until-ack) / Telegram (inline buttons) | Determines iOS UX + where approval metadata transits | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved (all "high" change-cost rows above):

- [x] **Trust model A vs B** — **DECIDED: A** (service-account token on server) for v0.26.0 Phase 1–2. B (zero-standing-secret) deferred to Phase 3 / follow-up doc — "nice to get to no secrets eventually."
- [x] **`<secret>` label default-forbids `{not secret}` sinks** — **DECIDED: yes.** Static leak-prevention is the core value.
- [x] **Sync blocking vs async re-queue** — **DECIDED: sync blocking** (reuses `ApprovalCheckpoint.RequestApproval` as-is); revisit async only if parallel-secret demand appears.
- [x] **Resolution location** — **DECIDED: server** (follows from trust model A; `op read` runs on the cloud coordinator with the service-account token).
- [x] **Approvals push transport** — **DECIDED: ntfy self-hosted** on Cloud Run (`ailang-*-ntfy` in `ailang-multivac*`, europe-west1).

## Solution Design

### Overview

Introduce a `Secret` effect and a `secret(ref: string) -> string<secret> ! {Secret}` builtin. At runtime the `Secret` effect handler:

1. **Checks capability** — is a `Secret` capability granted with this `ref` already approved (cache hit)? If yes, resolve.
2. **Requests approval** — if not, construct an `ApprovalRequest` carrying **only** `{ref, purpose, agent, chain}` (never a value) and call `ApprovalCheckpoint.RequestApproval()` (blocking).
3. **Notifies the operator** — the request triggers the existing Pub/Sub → notify daemon fan-out (macOS banner + Discord to phone): *"agent X wants `op://Prod/stripe/key` to <purpose> — approve/deny."*
4. **Resolves just-in-time** — on approval, run `op read <ref>` (model A) and return the value as a `string<secret>`.
5. **Enforces non-leakage by type** — the `<secret>` label means the value cannot reach any `string{not secret}` sink (log, trace serializer, Net body) without passing through an explicit `! {Declassify}` step. Declassification is itself capability-gated and audited.
6. **Scrubs** — the resolved value is redacted in OTEL traces/logs and dropped from the task env after the effect scope.

This is a **composition of four shipped subsystems** plus a thin 1Password resolver — not a new framework.

### Feature vs Package Boundary

This is a **hybrid**, with a deliberate split. The security-critical primitives **must** be a core language feature — a package categorically cannot add a new effect, a new taint label, or compiler-enforced sink rules, and the leak-prevention guarantee only exists if the type checker owns it.

| Layer | Where it lives | Rationale |
|-------|----------------|-----------|
| `Secret` effect, `secret()` builtin, `<secret>` taint label + `{not secret}` sink enforcement + `Declassify` gate, trace redaction | **Core feature** — `internal/types`, `internal/effects`, `internal/eval` | Type-system/effect-system primitives; not expressible from package space |
| `std/secret` module surface (`secret`, helpers) | **Bundled stdlib** | Thin ergonomic wrapper over the builtin |
| 1Password resolver + approval/notify bridge + dedicated push subscriber | **Core Go** — `internal/secrets`, `internal/daemon` | Effect handlers and Pub/Sub subscribers are runtime, not package-space; resolver is backend-pluggable but only 1Password is implemented |
| Org policy & ergonomics — *which* refs require approval, purpose templates, `Secret`→tier mappings, typed helpers (`stripeKey()`, `secretField(item, field)`) | **External AILANG package** — e.g. `pkg/sunholo/secret-policy` | Pure AILANG calling the gated builtin; no compiler hooks. Mirrors the existing `pkg/sunholo/gcp-auth`, `pkg/sunholo/auth` packages |

**Rule of thumb:** if the *type checker must enforce it* → feature. If it's *just AILANG code calling the gated builtin* → package. The trust boundary is a feature; org house-rules about it are a package.

### Architecture

**Components:**

1. **`Secret` effect + `secret()` builtin** (`internal/effects/`, `internal/eval/`): new effect name `Secret`; builtin that triggers the handler. Registered alongside `IO`/`FS`/`Net` (see `internal/effects/capability.go`).
2. **1Password resolver** (`internal/secrets/onepassword.go`, new): wraps `op read`/`op inject`. Model A reads via `OP_SERVICE_ACCOUNT_TOKEN`. Validates `op://` ref shape; never logs the value; returns `(value, error)` with structured `SecretUnavailable` on CLI failure (NO silent fallback per CLAUDE.md §2).
3. **Approval bridge** (`internal/effects/secret_approval.go`, new): adapts the `Secret` effect to `coordinator.ApprovalCheckpoint.RequestApproval()`. Adds a `ApprovalTypeSecret` variant carrying `{Ref, Purpose}`.
4. **`<secret>` taint label** (`internal/types/labels.go`, `sink_check.go`): register `secret` in the label lattice; the existing `CheckDeclassify` (`internal/types/sink_check.go:62`) already enforces the `! {Declassify}` gate — we add `secret` as a recognized source label and ensure `{not secret}` sink refinements are honored.
5. **Trace redaction** (`internal/effects/` + tracing): any value carrying `<secret>` is rendered as `«secret:op://…»` in spans/logs.
6. **Dedicated approvals push channel → iPhone** (new, see below): a separate Pub/Sub subscription on the *same* topic, filtered to approval events, driving an actionable lock-screen notification with Approve/Deny buttons. Reuses the existing `notify.Channel` framework.

### Dedicated approvals push channel (X = ntfy, self-hosted)

The general notify path (macOS + Discord) is a passive "needs attention" feed. Secret approvals warrant a **dedicated, actionable channel**: tap Approve/Deny from the lock screen without opening an app.

**Chosen transport: ntfy (self-hosted).** Rationale: native iOS app with lock-screen **action buttons** (each fires an HTTP POST → straight to `/api/approvals/{id}/approve|reject`); self-hostable so approval metadata (ref, purpose, agent) stays on our GCP infra — important for a *secrets* feature; open source; priority levels.

- **Honest caveat:** iOS instant delivery requires APNs; ntfy's self-hosted server routes a *wakeup ping* through ntfy.sh (`upstream-base-url`) — minimal metadata, and the secret value is never in the notification (only ref + purpose; body can be fetch-on-tap). Acceptable; alternatively accept slightly delayed delivery without the upstream.
- **Alternatives considered:** *Pushover* — turnkey, ultra-reliable, emergency priority retries-until-acknowledged (strongest "won't miss it"); weaker on inline two-button actions (tap-to-open). *Telegram bot* — richest inline Approve/Deny + zero infra; tradeoff: metadata transits a third party (same posture as current Discord, less clean for secrets). ntfy chosen for the data-stays-ours + lock-screen-actions combination.

**Wiring (matches "same topic, dedicated sub"):**
1. Publisher tags approval events with Pub/Sub attribute `kind=approval` (`internal/messaging/pubsub_notifier.go` already sets `MessageAttributes`).
2. A **dedicated subscription** `approvals-push-sub` on the existing topic with server-side filter `attributes.kind="approval"` — its own ack/retry/dead-letter tuned for approvals, isolated from the general notify feed.
3. Its handler drives a new `NtfyChannel` implementing `notify.Channel` + `EventFilter` (so it only accepts approval events, exactly as `DiscordChannel` declines `completed` events today).
4. Action-button URLs carry a **per-request signed, single-use, short-TTL token** bound to the approval ID — without this, anyone who learns the URL can approve a secret release. (This is a hard requirement, tracked in Risks.)
5. On Approve/Deny tap → `POST /api/approvals/{id}/approve|reject` resolves the blocked `ApprovalCheckpoint` exactly as the dashboard does today.

Adds `ApprovalTypeSecret` rendering in both the dashboard and the push payload (ref + purpose + agent; never a value).

### Data flow (model A, v0.26.0)

```
agent .ail code:  let k = secret("op://Prod/stripe/api-key")   -- k : string<secret>
   │
   ▼  Secret effect handler (internal/effects)
cache miss → ApprovalRequest{type:secret, ref, purpose, agent, chain}
   │
   ▼  ApprovalCheckpoint.RequestApproval()  [BLOCKS]
   ▼  Pub/Sub topic (attr kind=approval)
   ▼  dedicated approvals-push-sub (filter kind=approval) → NtfyChannel
   ▼  iPhone lock-screen: "agent X wants op://Prod/stripe — <purpose>"  [Approve] [Deny]
   │                                   │
   │                operator taps Approve (signed single-use token)
   ▼  POST /api/approvals/{id}/approve │
ProcessApprovalRequest() resolves checkpoint
   │
   ▼  resolver.Read("op://Prod/stripe/api-key")  via op CLI (server, SA token)
   ▼  returns string<secret>  →  redacted in all traces
   │
   ▼  type system: k cannot reach string{not secret} sink without ! {Declassify}
```

### Implementation Plan

**Phase 1: Resolver + gated effect (model A, no taint yet)** (~3 days)
- [ ] `internal/secrets/onepassword.go`: `Read(ref) (string, error)` via `op read`; ref-shape validation; structured errors; value never logged
- [ ] Register `Secret` effect + `secret()` builtin in `internal/effects/` and `internal/eval/`
- [ ] `ApprovalTypeSecret` + `secret_approval.go` bridge to `ApprovalCheckpoint.RequestApproval()`
- [ ] `internal/daemon/ntfy_channel.go`: `NtfyChannel` (implements `notify.Channel` + `EventFilter`) with Approve/Deny action buttons
- [ ] Deploy `${_PREFIX}-ntfy` Cloud Run service (min-instances=1, `upstream-base-url`) via `cloudbuild-dev.yaml` + one-time `gcloud run deploy`
- [ ] Dedicated `approvals-push-sub` **push** subscription (filter `attributes.kind="approval"`) → HTTPS bridge on coordinator; publisher sets `kind=approval` attribute
- [ ] Signed single-use short-TTL approval token for action-button URLs
- [ ] Audit span on every request/resolution
- [ ] Trace/log redaction for resolved values

**Phase 2: Taint integration + contracts** (~4 days)
- [ ] Register `secret` source label in `internal/types/labels.go`; honor `string{not secret}` sinks
- [ ] Ensure `CheckDeclassify` path covers `<secret>` → `{not secret}` requiring `! {Declassify}`
- [ ] `secret()` return type is `string<secret>`; standard log/trace/Net sinks refined to `{not secret}` where appropriate
- [ ] Contract helper predicate `approved(ref)` usable in `requires {}`; Z3 codegen support
- [ ] Examples: `examples/runnable/secrets/gated_secret.ail`, `leak_attempt.ail` (must fail `ailang check`)

**Phase 3 (optional / may split to own doc): zero-standing-secret (model B)** (~5 days)
- [ ] Operator-device resolver: `op read` runs where the desktop app / biometric lives
- [ ] Secure return channel for the resolved value (or short-lived derived token) back to the server
- [ ] Threat-model doc + key handling

### Files to Modify/Create

**New files:**
- `internal/secrets/onepassword.go` — `op` CLI resolver, ref validation, structured errors (~180 LOC)
- `internal/secrets/onepassword_test.go` — resolver tests with a fake `op` (~150 LOC)
- `internal/effects/secret_approval.go` — `Secret` effect ↔ `ApprovalCheckpoint` bridge (~150 LOC)
- `internal/daemon/ntfy_channel.go` — `NtfyChannel` (push w/ Approve/Deny action buttons) + signed-token minting (~160 LOC)
- `internal/daemon/approvals_subscriber.go` — dedicated `approvals-push-sub` consumer (filter `kind=approval`) (~90 LOC)
- `examples/runnable/secrets/gated_secret.ail` — happy path
- `examples/runnable/secrets/leak_attempt.ail` — negative example (must fail to type-check)

**Modified files:**
- `internal/effects/capability.go` / effect registry — add `Secret` effect (~40 LOC)
- `internal/eval/` builtins — register `secret()` (~60 LOC)
- `internal/types/labels.go`, `internal/types/sink_check.go` — register `secret` label + sink coverage (~80 LOC)
- `internal/coordinator/approval_*.go` — `ApprovalTypeSecret` variant + render (~60 LOC)
- `internal/server/handlers_approvals.go` — render secret-request fields; accept signed-token auth on approve/reject (~50 LOC)
- tracing/redaction hook — redact `<secret>` values (~40 LOC)
- `cloudbuild-dev.yaml` — build + `gcloud run deploy ${_PREFIX}-ntfy` step (~25 lines)
- `docker/Dockerfile.ntfy` (or reference `binwiederhier/ntfy` directly) + ntfy config (~20 lines)

## Examples

### Example 1: Gated secret, happy path

**Before** (today — ambient, ungated):
```ailang
-- API key is just an env var inherited by the whole process; nothing gates it,
-- nothing stops it flowing into a log line.
export func charge(amount: int) -> () ! {Net} {
  let key = getEnv("STRIPE_KEY");   -- ambient, plaintext, always present
  httpPost("https://api.stripe.com/charges", key, amount)
}
```

**After** (gated `Secret` effect + taint label):
```ailang
import std/secret (secret)

-- `secret` returns string<secret>. The Net body sink is refined string{not secret},
-- so the key cannot be sent raw — it must be used via an approved declassify path
-- (e.g. placed in an Authorization header builder that is itself ! {Declassify}).
export func charge(amount: int) -> () ! {Secret, Net} {
  let key = secret("op://Prod/stripe/api-key");  -- BLOCKS → phone approval → resolves
  stripeCharge(key, amount)                       -- key flows only into approved sink
}
```

### Example 2: Leak attempt is a compile-time error

```ailang
import std/secret (secret)
import std/io (println)

export func oops() -> () ! {Secret, IO} {
  let key = secret("op://Prod/stripe/api-key");  -- key : string<secret>
  println(key)   -- ❌ TYPE ERROR: println expects string{not secret};
                 --    <secret> not declassified. Add ! {Declassify} via an
                 --    explicit, audited sanitize step — or don't print it.
}
```
`ailang check` rejects this **before** any approval is even requested — the leak is prevented statically, mirroring the shipped `examples/runnable/contracts/inbox_injection_v2.ail` pattern.

### Example 3: Static approval-policy proof (contracts)

```ailang
-- Z3 proves at compile time (`ailang verify`) that this function can only run
-- on a ref that the contract requires to be approved.
export func useDbPassword(ref: string) -> Conn ! {Secret}
requires { approved(ref) }
{
  connect(secret(ref))
}
```

## Deployment & Infrastructure

### The AILANG cloud element (where approvals are served)

Approvals are owned by the **AILANG cloud element** — the Cloud Run services (package registry, dashboard, coordinator, mcp) in the `ailang-multivac*` GCP projects, region `europe-west1` — **not** by the local Mac. There is **no Terraform**; infra is managed imperatively via Cloud Build (`cloudbuild-dev.yaml` / `cloudbuild-release.yaml`) building images and running `gcloud run services/jobs update`. GitHub org: `sunholo-data/ailang`.

| Env | Project | Prefix | Cloud Run services |
|-----|---------|--------|--------------------|
| dev | `ailang-multivac-dev` | `ailang-dev` | `ailang-dev-coordinator`, `-dashboard`, `-mcp` (+ agent-executor jobs) |
| test | `ailang-multivac-test` | `ailang-test` | `ailang-test-*` |
| prod | `ailang-multivac` | `ailang` | `ailang-coordinator`, `-dashboard`, `-mcp`, … |

**Approval authority vs notification sink — the critical distinction:**
- The **approval authority** runs in the cloud element: `ApprovalCheckpoint`, the `/api/approvals/{id}/approve|reject` endpoint, the dedicated approvals subscription, and the ntfy publish all live on the always-on `*-coordinator` / `*-dashboard` Cloud Run services. This is the system of record.
- The **local macOS `ailang daemon`** (launchd on the Mac Studio) is only a notification **sink** (native banners + Discord). The new ntfy iPhone channel is another sink. **The Mac is never in the approval critical path** — stopping the local daemon must not affect whether secrets can be approved.
- Tasks publish the approval event to Pub/Sub from *wherever they run* — a cloud `agent-executor` job or a local launchd worker — so resolution is fully decoupled from execution location.

### Two new pieces, two homes

1. **ntfy server → new Cloud Run service `${PREFIX}-ntfy`** (`ailang-dev-ntfy` → `ailang-ntfy`), in `ailang-multivac*`, `europe-west1`, alongside the coordinator/dashboard it serves. It is a standalone server (own port, cache, APNs upstream) and **cannot** be embedded in the coordinator process.
   - Use the official `binwiederhier/ntfy` image (or a thin `docker/Dockerfile.ntfy`); add a build + `gcloud run deploy` step to `cloudbuild-dev.yaml` (same pattern as dashboard/mcp).
   - **`--min-instances=1`** (no scale-to-zero) so it stays reachable; in-memory / short `cache-duration` suffices for ephemeral approvals (no Cloud SQL needed).
   - Public HTTPS so the iPhone subscribes from anywhere; configure `upstream-base-url` for iOS instant delivery via APNs.
   - **Alternative:** `ntfy serve` on the Mac Studio via launchd over **Tailscale** (zero cloud cost) — but Tailscale-only loses *instant* iOS push (no APNs wakeup), and it would wrongly put a Mac in the approval path. Cloud Run, co-located with the cloud element, is the recommended home.

2. **Approvals subscriber/bridge → reuse the running cloud `*-coordinator` (no new compute).**
   - Idiomatic Cloud Run: make `approvals-push-sub` a **Pub/Sub *push* subscription** (filter `attributes.kind="approval"`) delivering to a small HTTPS handler on the coordinator service, which calls the `${PREFIX}-ntfy` service. Avoids an always-on pull loop (Cloud Run services lack background CPU unless forced).
   - The Approve/Deny **action endpoint** (`/api/approvals/{id}/approve|reject`) already lives on the coordinator/dashboard service; it must accept the **signed single-use token** as auth (not Google IAM) so the iPhone notification button can POST directly.

**Net new infra:** one Cloud Run service (`${PREFIX}-ntfy`) + one Pub/Sub push subscription + config/secrets (ntfy auth + `upstream-base-url`; the coordinator's ntfy publish URL/token, stored via `gcloud secrets` like `ailang-github-token`). Everything else reuses running services. **No Terraform to touch** — add a Cloud Build step + a one-time `gcloud run deploy`, mirrored across dev/test/prod.

> Bootstrapping note: the coordinator needs an ntfy publish token, which is itself a secret. For v0.26.0 source it from the existing `gcloud secrets` path; once `Secret` ships it can dogfood `op://`.

## Conflict Surface

**This change touches `internal/effects/`, `internal/types/`, and `internal/eval/` — Conflict Surface analysis is required.**

1. **What positions does this extend?**
   - The **effect-row name space** (`! {…}`): adds `Secret` (and relies on existing `Declassify`).
   - The **taint label lattice** (`<…>` source labels, `{not …}` sink refinements): adds `secret` as a label.
   - The **builtin/function name space**: adds `secret(...)` (namespaced as `std/secret`).

2. **What other valid constructs already live there?**
   - Effect names: `IO`, `FS`, `Net`, `Clock`, `Declassify`, plus user effect rows and the planned permission-tier effects (M-PERMISSION-MODEL). `Secret` must not collide with any existing effect identifier — verified: no `Secret` effect exists today (`grep` shows only `IO/FS/Net/Declassify` in `internal/effects/`).
   - Labels: `<email>`, `<sanitized>` appear in shipped examples (`inbox_injection_v2.ail`). The lattice is open; `secret` is a new join point. Must confirm `secret` is distinct from any reserved label.
   - The identifier `secret` as a value/function name: must confirm it is free in stdlib (it will be namespaced `std/secret` to avoid shadowing user bindings).

3. **How does the parser/typechecker disambiguate?**
   - `Secret` is an effect identifier inside `! {…}` rows — same lexical position as `IO`; no new grammar, no new lookahead. The M-TAINT-TYPES lookahead lesson (the `{ not <ident>` ambiguity between sink refinements and `not` in function bodies, fixed by M-PARSER-REFINEMENT-LOOKAHEAD) **already applies** to `{not secret}` exactly as it does to `{not email}` — we add no new refinement syntax, only a new label name, so the existing disambiguation covers us. **This must be re-tested with `secret` specifically.**

4. **Which existing programs MUST still work post-change? (fixtures)**
   - `examples/runnable/contracts/inbox_injection_v2.ail` — `<email>`/`{not email}`/`Declassify` must behave identically.
   - `examples/runnable/contracts/basic.ail` — contracts unaffected.
   - Any program using a user-defined function or binding named `secret` (namespacing must prevent breakage — **enumerate via grep before coding**).
   - All programs with `! {IO}` / `! {Net}` / `! {FS}` rows — adding an effect name must not change resolution of existing names.

5. **What deliberately changes (intentional incompatibilities)?**
   - Code that previously read a secret via an ambient `getEnv`/`os.Environ` path and then logged/printed it will, **once migrated** to `secret()` + `<secret>`, fail to type-check on the leak. This is the intended behavior change, not a regression — but it is opt-in via using the new builtin; existing `getEnv` code is untouched until migrated.

**The honest answer is not "no conflicts":** the live risk is the `{not secret}` refinement re-triggering the historical parser-lookahead ambiguity, and the `secret` identifier shadowing user bindings. Both are enumerated above with mitigations and must be fixture-tested.

## Success Criteria

- [ ] `secret("op://…")` resolves a value only after an approval is recorded (Phase 1)
- [ ] Approval request is pushed to the operator's phone (Discord) and macOS; one tap resolves it (reuses notify daemon)
- [ ] Every resolution emits an audit span with `{ref, purpose, agent, chain, approver, decision, ts}`
- [ ] Resolved secret values are redacted in all OTEL traces and logs (`«secret:…»`)
- [ ] `examples/runnable/secrets/leak_attempt.ail` **fails** `ailang check` with a clear `<secret>`→`{not secret}` diagnostic (Phase 2)
- [ ] `examples/runnable/secrets/gated_secret.ail` passes `ailang check` and runs end-to-end against a service account
- [ ] `ailang verify` proves a `requires { approved(ref) }` contract (Phase 2)
- [ ] No regression: `inbox_injection_v2.ail` and `basic.ail` still pass (`make verify-examples`)
- [ ] `make test` + `make verify-examples` green
- [ ] Docs updated: CHANGELOG, `docs/` secrets guide, prompt/reference for `std/secret`

## Testing Strategy

**Unit tests:**
- Resolver: valid ref → value; malformed ref → typed error; `op` non-zero exit → `SecretUnavailable` (no fallback); value never appears in logged output.
- Label: `<secret>` → `{not secret}` sink without `Declassify` ⇒ type error; with `Declassify` ⇒ ok (extend `internal/types/sink_check_test.go`).
- Approval bridge: request blocks; approve resolves; reject ⇒ `SecretDenied`; timeout ⇒ typed error.

**Integration tests:**
- End-to-end with a **fake `op`** binary on PATH returning a known value; assert approval gate fires and value is redacted in the emitted trace.
- Notify fan-out smoke (reuse existing daemon test harness).

**Manual testing:**
- Real `op` service account in a scratch vault; trigger a task; approve from phone via Discord; confirm resolution + audit span.

## Deferred Decisions

The following are intentionally open for the implementer:

- Approval-cache granularity and TTL (per-task vs per-chain vs window) — **agent may choose**, default per-ref-per-chain; document it.
- Exact `op` invocation (`op read` vs `op inject` for multi-field templates) — **agent may choose** based on resolver ergonomics.
- Notification copy/format for secret requests — **agent may choose**; must include ref + purpose, never a value.
- Whether `std/secret` also exposes `secretField(item, field)` sugar over raw `op://` — **agent may choose** in Phase 2.

## Non-Goals

- **Becoming a secrets manager.** 1Password remains the vault and source of truth. We resolve references; we do not store, rotate, or sync secrets.
- **Writing to vaults.** Read-only. (Service accounts are read-only by default anyway.)
- **Replacing M-PERMISSION-MODEL.** `Secret` is one gated effect that should *slot into* the tier model, not a competing permission system.
- **Bitwarden/Vault/other backends.** Out of scope for v0.26.0; the resolver interface should be backend-agnostic enough to add later, but only 1Password is implemented.
- **Model B (zero-standing-secret) in v0.26.0.** Designed here (Phase 3) but may be split into its own doc and shipped later.

## Timeline

**Week 1** (~3 days): Phase 1 — resolver, `Secret` effect + builtin, approval bridge, notify wiring, audit + redaction. Demoable gated read.

**Week 2** (~4 days): Phase 2 — taint label integration, contract `approved()` predicate, examples (happy + leak), tests, docs. Static leak-prevention proven.

**Phase 3** (optional, separate sprint): model B zero-standing-secret.

**Total: ~7 working days for Phase 1–2 across 2 weeks.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `{not secret}` refinement re-triggers the historical parser-lookahead ambiguity | High | Re-run M-PARSER-REFINEMENT-LOOKAHEAD fixtures with `secret`; add `secret`-specific parse fixtures before coding |
| Resolved secret leaks into a trace/log despite the label | High | Static `{not secret}` sink enforcement **plus** runtime redaction (defense in depth); negative example gated in CI |
| Service-account token on the server is itself a standing secret (model A) | Med | Document the residual risk; per-use human approval + audit limits blast radius; Phase 3 (model B) removes it entirely |
| `op` CLI absent/misconfigured on the server | Med | Resolver fails loudly with `SecretUnavailable` (no fallback per CLAUDE.md §2); health-check on daemon start |
| Approval fatigue (operator taps "approve" reflexively) | Med | Request payload always shows ref + purpose; cache granularity limits repeat prompts; M-PERMISSION-MODEL tiers can auto-allow low-risk refs |
| Action-button URL leaks → unauthorized approval of a secret release | High | Per-request **signed, single-use, short-TTL** token bound to the approval ID; reject reused/expired tokens; HTTPS only |
| ntfy iOS instant delivery routes a wakeup ping via ntfy.sh (`upstream-base-url`) | Low | Only ref+purpose ever leave our infra (never the value); or disable upstream and accept delayed delivery; revisit native APNs (Future Work) if unacceptable |
| Identifier `secret` shadows a user binding | Low | Namespace as `std/secret`; grep existing examples/stdlib before coding |

## Related Documents

<!-- Reviewed against the duplicate/coverage gate: max neural similarity 0.39 — no duplicate. -->

**Implemented (inform design):**
- [m_r2_effect_system.md](../../implemented/v0_2_0/m_r2_effect_system.md) (0.39) — the effect system this extends with a `Secret` effect
- [m-dashboard-approval-integration.md](../../implemented/v0_7_0/m-dashboard-approval-integration.md) (0.36) — the approval workflow + `/api/approvals/{id}/approve` reused for remote approval
- M-TAINT-TYPES — [m-taint-types.md](../../implemented/v0_16_0/m-taint-types.md) — the `<label>` / `{not label}` / `Declassify` machinery this builds on
- Contracts/SMT — [m-verify-smt-verification.md](../../implemented/v0_8_1/m-verify-smt-verification.md) — `requires`/`ensures` + `ailang verify`

**Planned (overlap checked — distinct):**
- [m-permission-model.md](../v0_23_0/m-permission-model.md) (0.36) — **closely related, not overlapping**: that doc defines effect-row → permission *tiers* + HITL gates generally; this doc defines one specific gated effect (`Secret`) and its 1Password resolution. `Secret` should map to a tier in that model. Coordinate, don't duplicate.
- [m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) (0.35) — future effect refinement; orthogonal.

## References

- [Design Axioms](/docs/references/axioms)
- [1Password Service Accounts](https://developer.1password.com/docs/service-accounts/get-started/) — headless `op` resolution
- [1Password Agentic Autofill](https://www.1password.dev/agentic-autofill/) — device-local, Early Access, why it doesn't cover server→phone
- [1Password remote-approval feature request](https://www.1password.community/discussions/developers/feature-request-remote-approval-for-op-cli--desktop-prompts-to-support-agentic-w/167877) — the gap AILANG fills
- [1Password SSH agent security](https://developer.1password.com/docs/ssh/agent/security) — per-use approval pattern (local) we mirror remotely
- `internal/types/sink_check.go` (`CheckDeclassify`) — the `Declassify` gate
- `internal/coordinator/approval_checkpoint.go` (`RequestApproval`) — the blocking approval rail
- `examples/runnable/contracts/inbox_injection_v2.ail` — shipped taint-label exemplar

## Future Work

- **Model B (zero-standing-secret):** resolve on the operator's device via local biometric; return only the value / a short-lived derived token. Fully realizes "no long-lived secret on the server."
- **Backend-agnostic resolver:** Vault / Bitwarden / cloud KMS behind the same `Secret` effect.
- **Auto-allow policies via M-PERMISSION-MODEL tiers:** low-risk refs approved by policy, high-risk always HITL.
- **Derived-credential effects:** request a scoped, short-TTL token (e.g. STS) rather than a long-lived secret.
- **Native APNs channel:** a first-party iOS app + APNs (rich actionable notifications, fully self-hosted push, no ntfy.sh wakeup hop) if the ntfy upstream-ping caveat proves unacceptable. Heaviest lift (Apple Developer account, cert/token management, app maintenance).

---

**Document created**: 2026-06-19
**Last updated**: 2026-06-19

---
DESIGN_DOC_PATH: design_docs/planned/v0_26_0/m-secret-effect-remote-approval.md

# M-SECRET-REMOTE-APPROVAL-WIRING: Make Push-to-Phone Secret Approval Operational

**Status**: Planned
**Target**: v0.26.0 (follow-up to M-SECRET-EFFECT)
**Priority**: P1 - Medium
**Estimated**: ~1 week (code wiring + Terraform + runbook)
**Parent**: [m-secret-effect-remote-approval.md](m-secret-effect-remote-approval.md) (M1–M6 shipped components)
**Dependencies**:
- M-SECRET-EFFECT M1–M4 (**shipped**) — resolver, `Secret` effect, in-process `SecretApprovalGate`, `NtfyChannel` + `approvaltoken` signer
- M-SECRET-EFFECT M5–M6 (**shipped v0.26.0**) — static IFC enforcement (`CheckModuleIFC`)
- Coordinator `ApprovalCheckpoint` (**shipped v0.6.4**), notify daemon (**shipped**)
- `ailang-multivac` Terraform (**exists** — the parent doc's "no Terraform" note is stale; see [Correction](#correction-to-parent-doc))

## Problem

M-SECRET-EFFECT built every *component* of the "agent requests a secret → I approve on my phone → it resolves" flow, but the flow is **not operational**. Three concrete gaps, all verified in-tree:

1. **The approval gate has zero callers.** `coordinator.SecretApprovalGate` ([internal/coordinator/secret_approval.go](../../../internal/coordinator/secret_approval.go)) is the M3 adapter whose own doc comment says it "should be set as the `EffContext.Secret.Approver` for coordinator-run tasks" — but nothing ever assigns it. `grep` shows **0 non-test callers** of `NewSecretApprovalGate`. So everywhere today `EffContext.Secret.Approver == nil`, and `secret()` resolves immediately with no approval ([internal/effects/secret.go:96](../../../internal/effects/secret.go)).

2. **The gate is in-process; secret execution is out-of-process.** `secret()` runs inside `ailang run` ([cmd/ailang/main_run_exec.go:310](../../../cmd/ailang/main_run_exec.go) builds the `EffContext`), which in production executes inside **agent-executor jobs** — a *separate process* from the always-on coordinator that owns `ApprovalCheckpoint` and `/api/approvals`. The M3 gate dials the checkpoint via an in-process Go interface, so it cannot bridge that process boundary. The parent doc confirms the split (Deployment & Infrastructure §: the approval authority lives on the coordinator/dashboard Cloud Run services; the `.ail` runs elsewhere).

3. **The ntfy service + approvals subscription are not deployed.** `cloudbuild-dev.yaml` *builds* `docker/Dockerfile.ntfy` and has a best-effort `gcloud run services update`, but the comment is explicit: *"ntfy service not yet created — run the one-time `gcloud run deploy ${_PREFIX}-ntfy --min-instances=1`."* No `approvals-push-sub` Pub/Sub subscription exists.

**Net:** the static half of M-SECRET-EFFECT (leak prevention) is live; the *human-approval* half is scaffolding.

## Goals / Non-Goals

**Goals**
- A secret-consuming AILANG task running in the cloud **blocks** on `secret(ref)` until a human approves on their phone, then resolves; deny/timeout returns `E_SECRET_DENIED` (fail-closed).
- The approval request carries **ref + purpose + agent, never the value**; the value is resolved only in the executor process, post-approval, and never transits ntfy or the coordinator.
- Local/CLI runs stay **un-gated** (no approver) — unchanged from today.
- Infra is **Terraform-managed** in `ailang-multivac`, gated behind a feature flag, dev-first.

**Non-Goals**
- Trust model B (resolve on operator device) — deferred, as in the parent doc.
- The `approved(ref)` Z3 contract predicate — deferred (parent doc Phase 2; approval is a runtime property).
- Auto-approve policy tiers (M-PERMISSION-MODEL) — future.

## Design decision: a networked approver

The binding constraint is gap #2. The secret-consuming process (`ailang run` in an agent-executor job) is not the process that owns the approval authority (coordinator). Therefore the approver attached to `EffContext.Secret.Approver` must be a **networked client of the coordinator**, not the in-process `SecretApprovalGate`.

**`CloudSecretApprover`** (new, implements `effects.SecretApprover`):
1. On `Approve(ctx, ref, purpose)`: `POST {coordinator}/api/approvals` with `{ref, purpose, agent, task}` (never the value). Receives an approval `id`.
2. Blocks, polling `GET /api/approvals/{id}` (or Firestore) until `approved` / `denied` / deadline.
3. `approved` → return `nil` (the resolver then runs `op read`); `denied`/timeout → return an error → `E_SECRET_DENIED`.

It is attached to the `EffContext` **only in cloud mode** (env-detected: e.g. `AILANG_STORAGE=gcp` + a configured `AILANG_COORDINATOR_URL`). Absent that, `Approver` stays nil and behavior is identical to today's local CLI.

**Coordinator side** (extends shipped pieces):
- `POST /api/approvals` intake → create an `ApprovalCheckpoint` of `ApprovalTypeSecret` (M3) and **publish a `kind=approval` Pub/Sub event** carrying the M4 notification payload (`BuildSecretApprovalNotification`, which already mints the signed Approve/Deny token URLs).
- Enable M4 signed-token auth on `/api/approvals/{id}/approve|reject` via the already-built `Server.SetSecretApprovalAuth` ([internal/server/secret_approval_auth.go](../../../internal/server/secret_approval_auth.go)) so the phone buttons can POST directly without Google IAM.

**ntfy** (Cloud Run) receives the `kind=approval` event via the dedicated subscription and pushes the lock-screen Approve/Deny notification; the buttons POST the signed single-use token back to the coordinator.

```
agent-executor job:  ailang run → secret(ref) → CloudSecretApprover.Approve
        │  POST /api/approvals {ref,purpose,agent}        (value NEVER sent)
        ▼
coordinator (Cloud Run): ApprovalCheckpoint + publish Pub/Sub attr kind=approval
        ▼
approvals-push-sub (filter attributes.kind="approval") → ntfy publish
        ▼
iPhone lock screen: "agent X wants op://Prod/stripe — <purpose>"  [Approve][Deny]
        │  button POSTs signed single-use token → /api/approvals/{id}/approve
        ▼
coordinator records decision  ──poll──▶ CloudSecretApprover returns
        ▼
agent-executor: op read --no-newline <ref> → value (labelled <secret>), used, scrubbed
```

### Alternatives considered
- **Attach the in-process `SecretApprovalGate` as-is** — only correct if secret-consuming code runs *inside* the coordinator process. It doesn't (wrong topology). Rejected; keep the gate for any future in-process use.
- **Run secret-consuming `.ail` inside the coordinator** — couples execution with the approval authority and doesn't generalize to arbitrary agent tasks. Rejected.
- **Long-poll instead of poll** — viable optimization; start with bounded polling for simplicity, revisit.

## Component / file plan

### `ailang` repo (code — testable without cloud)
- `internal/secrets/cloud_approver.go` (new) — `CloudSecretApprover` (HTTP client + poll loop; never logs the value; configurable deadline). Unit-tested against an `httptest` fake coordinator.
- `cmd/ailang/run_helpers.go` / `main_run_exec.go` — in cloud mode, set `effCtx.Secret.Approver = secrets.NewCloudSecretApprover(...)`. Env-gated; local default unchanged.
- `internal/coordinator` / `internal/server` — `/api/approvals` intake handler → `ApprovalCheckpoint` + `kind=approval` publish; call `SetSecretApprovalAuth` at startup when a signer secret is present.
- Tests: fake coordinator + fake `op`; assert (a) approve → resolves, (b) deny/timeout → `E_SECRET_DENIED`, (c) request payload carries ref-not-value, (d) audit span emitted.

### `ailang-multivac` repo (Terraform — written for human `terraform apply`)
Mirrors existing conventions (`${var.prefix}-…`, `google_cloud_run_v2_service`, `var.bootstrap` count guard, `enable_*` feature flag):
- `terraform/cloud_run_ntfy.tf` (new) — `${var.prefix}-ntfy` service: ntfy image, `--min-instances=1`, public invoker, `NTFY_*` env + `upstream-base-url`, auth token from Secret Manager. Gated by **`var.enable_secret_approvals`**.
- `terraform/pubsub.tf` (extend) — `${var.prefix}-approvals-push-sub` **push** subscription on the existing `messages` topic (or a dedicated `${var.prefix}-approvals` topic), `filter = "attributes.kind = \"approval\""`, push → coordinator `/pubsub/push` (OIDC), retry + dead-letter (mirror `cascade_coordinator`).
- `terraform/secrets.tf` (extend) — `${var.prefix}-ntfy-auth-token` (+ the coordinator's ntfy publish token).
- `terraform/variables.tf`, `iam.tf`, `outputs.tf` — flag, IAM (`run.invoker` for Pub/Sub push), ntfy URL output.

## Security considerations
- **Value isolation:** the resolved value exists only in the executor process after approval. The approval request, Pub/Sub event, ntfy notification, and coordinator records carry **ref + purpose + agent only** — never the value (enforced by `BuildSecretApprovalNotification`, which is value-free by construction).
- **Token auth:** Approve/Deny buttons use M4 HMAC **single-use, short-TTL** tokens (`approvaltoken.Signer` + `SingleUseGuard`), not Google IAM, so the phone can POST directly; reused/expired tokens are rejected.
- **Self-hosted ntfy:** approval metadata stays on our GCP infra; only an APNs wakeup ping (no payload) transits ntfy.sh upstream.
- **Fail-closed:** any approver error (network, deny, timeout) → `E_SECRET_DENIED`; no silent fallback (CLAUDE.md §2).
- **Bootstrapping:** the coordinator's ntfy publish token is itself a secret — sourced from `gcloud secrets` for v0.26.0; can dogfood `op://` later.

## Correction to parent doc
The parent design doc's Deployment section states "There is **no Terraform**; infra is managed imperatively via Cloud Build." This is **stale** — `ailang-multivac/terraform/` is a full Terraform setup (`cloud_run.tf`, `pubsub.tf`, `secrets.tf`, `iam.tf`, per-env `tfvars`, GCS backend). This milestone adds the ntfy infra as Terraform, not ad-hoc `gcloud`. The parent doc should be annotated accordingly when this lands.

## Resolved decisions (2026-06-23)
1. **Topic:** a **dedicated `${prefix}-approvals` topic** (own retry/dead-letter; clean isolation for a security feature).
2. **Executor ↔ decision channel:** **poll the coordinator over HTTP** (`GET /api/approvals/{id}`) — keeps the executor decoupled from Firestore. *(Implemented in M1: `CloudSecretApprover`.)*
3. **Default approval deadline:** 5 min, then `E_SECRET_DENIED` (fail-closed). *(Implemented in M1, overridable via option.)*
4. **Env to start:** **dev only**, gated behind `var.enable_secret_approvals`; test/prod later.

## Milestone status
- **M1 — DONE** (`ailang` repo): `internal/secrets/cloud_approver.go` (`CloudSecretApprover` — POST `/api/approvals` + bounded poll, value-free, fail-closed) + `cmd/ailang/secret_approver.go` wiring (`attachCloudSecretApprover` in `grantCapabilities`, cloud-mode env-gated). 9 unit tests + verified end-to-end through the binary against a fake coordinator + fake `op`. Local CLI unchanged (un-gated).
- M2–M4: pending.

## Success criteria
- [ ] A cloud-run secret task blocks on `secret()` until phone approval; approve → resolves, deny/timeout → `E_SECRET_DENIED`
- [ ] Approval request/notification provably value-free (test asserts ref-not-value)
- [ ] `terraform plan` is clean in `ailang-multivac` with `enable_secret_approvals=true` (dev); no-op when false
- [ ] Operator runbook published (docs site) covering deploy + iPhone subscribe + first approval
- [ ] `make test` green; local CLI `secret()` behavior unchanged (un-gated)

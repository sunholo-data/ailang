---
title: Secret Approvals (push-to-phone)
description: Set up remote human approval of secret() resolution — an agent requests a 1Password secret, you approve on your phone.
---

# Secret Approvals — push-to-phone

AILANG's `Secret` effect resolves 1Password references (`op://Vault/Item/field`)
behind a **declared, capability-gated, flow-controlled** effect. In the cloud,
resolution can be gated on a **human approval pushed to your phone**: an agent
calls `secret(ref)`, the call blocks, your phone shows an Approve/Deny
notification, and the value is resolved just-in-time only on approval.

The resolved **value never leaves the executor** — the approval request,
Pub/Sub event, and notification carry only the reference, a purpose, and the
requesting agent.

## Status (what works today)

| Piece | State |
|-------|-------|
| Static leak prevention (`<secret>` → `{not secret}` is a compile error) | ✅ shipped (v0.26.0) |
| Local `secret()` resolution via `op` (no approval gate) | ✅ shipped |
| Networked approver — executor blocks, polls the coordinator | ✅ shipped (M1) |
| Coordinator intake/status/resolve + signed-token auth | ✅ shipped (M2) |
| ntfy Cloud Run service + approvals topic/subscription (Terraform) | ✅ shipped (M3, `ailang-multivac`) |
| `kind=approval` publish + coordinator→ntfy bridge | ✅ shipped (M3.5) |
| This runbook | ✅ you're reading it |

:::info Code-complete — deploy-gated
The full code path is now built (executor → dashboard intake → publish →
coordinator bridge → ntfy → phone, and back). What remains is **deploying it**:
`terraform apply`, building the ntfy image, setting the secrets/env below, and
subscribing your phone. Until then a gated `secret()` in the cloud blocks and
**times out** (fail closed). Everything is testable locally without the deploy
(see the demo at the end).
:::

## Architecture

```
agent-executor (ailang run):  secret(ref) → blocks
        │  POST /api/approvals {ref,purpose,agent}        (value NEVER sent)
        ▼
coordinator (Cloud Run):  pending approval record + publish kind=approval
        ▼
${prefix}-approvals topic → approvals-push-sub → coordinator /pubsub/push bridge
        ▼
ntfy (Cloud Run):  push to your iPhone — "agent X wants op://… — <purpose>"  [Approve][Deny]
        │  button POSTs a signed single-use token → /api/approvals/{id}/approve
        ▼
coordinator records the decision  ──poll──▶ executor unblocks
        ▼
agent-executor:  op read <ref> → value (labelled <secret>), used, scrubbed
```

## Prerequisites

- `gcloud` authenticated against the target project (dev: `ailang-multivac-dev`).
- The 1Password `op` CLI available to the executor (service-account token
  `OP_SERVICE_ACCOUNT_TOKEN`, trust model A).
- The **ntfy app** on your phone — [iOS](https://apps.apple.com/app/ntfy/id1625396347)
  or [Android](https://f-droid.org/en/packages/io.heckel.ntfy/).

## Deploy (dev first)

The infra is Terraform in the `ailang-multivac` repo, gated behind
`enable_secret_approvals` (already `true` in `terraform/environments/dev/`).

```bash
# 1. Plan + apply the ntfy service, approvals topic/subscription, and secret.
cd ailang-multivac
make plan ENV=dev      # review: ailang-dev-ntfy, ailang-dev-approvals(-push-sub), ailang-dev-ntfy-auth-token
make apply ENV=dev

# 2. Build + push the ntfy image (ailang repo cloudbuild 'build-ntfy' step,
#    or build docker/Dockerfile.ntfy manually and push to the artifact registry).

# 3. Set the ntfy auth token value (never stored in Terraform state).
openssl rand -hex 32 | gcloud secrets versions add ailang-dev-ntfy-auth-token \
  --project ailang-multivac-dev --data-file=-

# 4. Find the ntfy URL.
cd terraform && terraform output -raw ntfy_url
```

`ntfy_min_instances` defaults to **0** (scale-to-zero, like the other services).
Cold start is fine for the push path; set it to `1` in the tfvars if you want
guaranteed iOS fetch-back delivery (ntfy holds the message in memory).

### Coordinator and executor configuration

`/api/approvals` is served by the **dashboard** service; `/pubsub/push` (the ntfy
bridge) by the **coordinator**. So the env splits across three places:

| Env var | Where | Purpose |
|---------|-------|---------|
| `AILANG_APPROVAL_SIGNING_KEY` | **dashboard** | HMAC key enabling signed single-use token auth on approve/reject (mints the phone-button tokens) |
| `AILANG_APPROVAL_BASE_URL` | **dashboard** | the dashboard's own public URL — used to build the Approve/Deny action links |
| `AILANG_NTFY_SERVER_URL` | **coordinator** | the ntfy service URL (`terraform output ntfy_url`) |
| `AILANG_NTFY_TOPIC` | **coordinator** | the ntfy topic your phone subscribes to (`secret-approvals`) |
| `AILANG_NTFY_AUTH_TOKEN` | **coordinator** | _omit in dev_ (anonymous ntfy); set only with a real ntfy auth file in prod |
| `AILANG_STORAGE=gcp` | executor | selects cloud mode |
| `AILANG_APPROVAL_URL` | executor | the **dashboard** base URL the approver POSTs to (falls back to `AILANG_COORDINATOR_URL`) |
| `AILANG_AGENT_ID` / `AILANG_TASK_ID` | executor | label the approval request (optional) |
| `AILANG_APPROVAL_TOKEN` | executor | optional bearer token on the intake POST |

## Phone setup

1. Install the **ntfy app** (links above).
2. In the app → **Add subscription** → set the server to your ntfy URL
   (from `terraform output ntfy_url`, or a custom domain) and the topic name.
3. **No token needed (dev).** The dev ntfy runs `NTFY_AUTH_DEFAULT_ACCESS=read-write`,
   so just subscribe to the topic — leave the auth fields blank. Security in dev
   rests on the unguessable topic + the value-free notification + the HMAC-signed
   Approve/Deny action (`approval-signing-key`), not on ntfy auth. **Prod
   hardening** (a persistent ntfy auth file + per-user tokens) is a follow-up;
   the `ailang-dev-ntfy-auth-token` secret is reserved for it.
4. For instant iOS delivery, the server's `NTFY_UPSTREAM_BASE_URL` (default
   `https://ntfy.sh`) relays an APNs wakeup ping — only a ping, never the
   payload.

## Test it

Once the bridge ships and the deploy is done:

```bash
# A gated secret task running in cloud mode (point at the dashboard URL):
AILANG_STORAGE=gcp AILANG_APPROVAL_URL=<dashboard-url> \
  ailang run --caps Secret,IO --entry main yourtask.ail
```

`secret("op://…")` blocks → your phone buzzes → tap **Approve** → the run
continues and resolves the value; tap **Deny** (or let it time out, default
5 min) → the run fails closed with `E_SECRET_DENIED`.

To test the parts that work **today** without any of the above, see the local
demo: [`examples/runnable/secrets/demo.sh`](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/secrets/demo.sh)
(static leak rejection + local `secret()` resolution with a fake `op`).

## Guarantees

- **Value isolation** — the resolved value exists only in the executor process,
  post-approval. The request, Pub/Sub event, and notification are value-free by
  construction.
- **Fail closed** — any denial, timeout, or transport error returns
  `E_SECRET_DENIED`; there is no silent fallback to a blank credential.
- **Single-use tokens** — Approve/Deny buttons carry HMAC single-use,
  short-TTL tokens; reused or expired tokens are rejected.
- **Self-hosted** — approval metadata stays on your GCP infra; only an APNs
  wakeup ping transits the ntfy.sh upstream.

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `secret()` blocks then `E_SECRET_DENIED` after ~5 min | No decision arrived. Today: the publish/consume bridge isn't wired (see status). After it ships: check the coordinator published, the subscription delivered, and ntfy reached your phone. |
| `effect 'Secret' requires capability` | Run with `--caps Secret`. |
| `E_SECRET_UNAVAILABLE` | `op` missing or auth failed — there is no fallback. Check `OP_SERVICE_ACCOUNT_TOKEN`. |
| Phone never subscribes | Add the `ailang-dev-ntfy-auth-token` value in the ntfy app (server runs `deny-all`). |
| No notification but approval is `approved` in the dashboard | ntfy delivery issue — check the ntfy service logs and `NTFY_UPSTREAM_BASE_URL`. |

## See also

- [IFC labels](ifc-labels.mdx) — the `<secret>` label + sink enforcement.
- Design: [m-secret-effect-remote-approval.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_26_0/m-secret-effect-remote-approval.md),
  [m-secret-remote-approval-wiring.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_26_0/m-secret-remote-approval-wiring.md).

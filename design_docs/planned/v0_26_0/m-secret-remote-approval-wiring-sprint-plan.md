# M-SECRET-REMOTE-APPROVAL-WIRING — Sprint Plan

**Design doc**: [m-secret-remote-approval-wiring.md](m-secret-remote-approval-wiring.md)
**Target**: v0.26.0 (follow-up to M-SECRET-EFFECT M1–M6)
**Scope**: Make push-to-phone secret approval operational — networked approver + coordinator publish + ntfy Terraform + runbook.
**Repos**: `ailang` (M1–M2, M4), `ailang-multivac` (M3).
**Risk**: Medium (cross-process control flow, new cloud infra). Fail-closed throughout.
**Estimated**: ~5 working days, ~700 LOC (impl + tests + Terraform).

## Frozen decisions (from design doc)
- **Networked approver** (`CloudSecretApprover`), not the in-process M3 gate — execution and approval authority live in different processes.
- Approval payload is **value-free** (ref + purpose + agent only).
- Local/CLI runs stay **un-gated** (approver nil unless cloud mode).
- Infra is **Terraform** in `ailang-multivac`, gated behind `enable_secret_approvals`, **dev first**.
- Fail-closed: any approver error → `E_SECRET_DENIED`.

## Milestones

### M1 — `CloudSecretApprover` + cloud-mode wiring (`ailang`)
- **Files**: `internal/secrets/cloud_approver.go` (+test); `cmd/ailang/run_helpers.go` / `main_run_exec.go`.
- **Work**: `CloudSecretApprover` implementing `effects.SecretApprover`: `POST {coordinator}/api/approvals {ref,purpose,agent,task}` → `id`, then bounded-poll `GET /api/approvals/{id}` until approved/denied/deadline. Never logs the value. Attach to `EffContext.Secret.Approver` only when cloud-mode env is present (`AILANG_STORAGE=gcp` + `AILANG_COORDINATOR_URL`); otherwise nil (unchanged).
- **LOC**: ~150 + ~120 test.
- **Deps**: none.
- **Accept**: against an `httptest` fake coordinator — approve → `Approve` returns nil; deny → error; timeout → error; request body carries ref-not-value (asserted); value never appears in logs; local mode (no env) leaves approver nil. `make test` green.

### M2 — Coordinator approval intake + `kind=approval` publish + token auth (`ailang`)
- **Files**: `internal/server/` (intake handler + `SetSecretApprovalAuth` startup wiring), `internal/coordinator/` (publish).
- **Work**: `POST /api/approvals` → create `ApprovalTypeSecret` `ApprovalCheckpoint`; publish a Pub/Sub event with attribute `kind=approval` carrying `BuildSecretApprovalNotification` output (token-bearing Approve/Deny URLs). Call `Server.SetSecretApprovalAuth(signer)` at startup when the ntfy/approval signer secret is configured. `GET /api/approvals/{id}` returns the current decision for the executor poll.
- **LOC**: ~140 + ~90 test.
- **Deps**: M1 (shared request/response shapes).
- **Accept**: intake creates a pending checkpoint and publishes exactly one `kind=approval` message (fake publisher); approve/reject via signed token flips the decision; reused/expired token rejected; published payload is value-free. `make test` green.

### M3 — ntfy Terraform + approvals subscription (`ailang-multivac`)
- **Files**: `terraform/cloud_run_ntfy.tf` (new), `terraform/pubsub.tf` (+sub), `terraform/secrets.tf` (+token), `terraform/variables.tf` (+`enable_secret_approvals`), `terraform/iam.tf`, `terraform/outputs.tf`.
- **Work**: `${var.prefix}-ntfy` `google_cloud_run_v2_service` (min_instances=1, public invoker, `NTFY_*` + `upstream-base-url`, auth token from Secret Manager) gated by `var.enable_secret_approvals`; `${var.prefix}-approvals-push-sub` push subscription `filter = "attributes.kind = \"approval\""` → coordinator `/pubsub/push` (OIDC), retry + dead-letter (mirror `cascade_coordinator`); `${var.prefix}-ntfy-auth-token` secret; IAM `run.invoker` for push; ntfy URL output.
- **LOC**: ~180 Terraform.
- **Deps**: M2 (the `kind=approval` contract).
- **Accept**: `terraform plan` (dev tfvars, `enable_secret_approvals=true`) is clean and creates exactly the new service + subscription + secret + IAM; with the flag **false** it is a no-op (count guards). `terraform validate` passes. **Actual `terraform apply` + ntfy deploy + secret value + iPhone subscribe are MANUAL human steps** (flagged, documented in M4).

### M4 — Operator runbook + parent-doc correction (`ailang`)
- **Files**: `docs/docs/guides/secret-approvals.md` (new), annotate `m-secret-effect-remote-approval.md`, CHANGELOG.
- **Work**: website guide — deploy sequence (`terraform apply` dev → set ntfy secret → subscribe iPhone → first approval), the value-free guarantee, fail-closed semantics, and how to run a gated secret task. Correct the parent doc's stale "no Terraform" claim. CHANGELOG entry.
- **LOC**: docs only.
- **Deps**: M1–M3.
- **Accept**: guide walks an operator from zero to a first phone approval; `make verify-examples`/docs build green; CHANGELOG updated.

## Day-by-day

| Day | Milestone | Deliverable |
|-----|-----------|-------------|
| 1 | M1 | `CloudSecretApprover` + cloud-mode wiring, fake-coordinator tests green |
| 2 | M2 | Intake + `kind=approval` publish + token auth, value-free test |
| 3 | M3 | ntfy + subscription + secret Terraform; `terraform plan` clean (dev), no-op when flag false |
| 4 | M4 | Operator runbook + parent-doc correction + CHANGELOG |
| 5 | buffer | e2e dry-run with fake op + fake ntfy; polish; flag default review |

## Success metrics
- [ ] Cloud secret task blocks on `secret()` → phone approve resolves; deny/timeout → `E_SECRET_DENIED`
- [ ] Approval request + published event provably value-free (tests)
- [ ] `terraform plan` clean (dev, flag on); no-op (flag off)
- [ ] Local CLI `secret()` unchanged (un-gated)
- [ ] Operator runbook on the docs site; parent doc "no Terraform" corrected
- [ ] `make test` green

## Risks
- **Cross-process control flow / partial failure**: executor blocks while the coordinator/ntfy is down. Mitigate with a bounded deadline → `E_SECRET_DENIED` (fail-closed), and a clear operator error.
- **Terraform not applyable by the agent**: M3 ships `*.tf` + a clean `plan`; `apply`, ntfy deploy, secret values, and iPhone subscribe are manual human steps. Flagged.
- **Topic/decision-channel choice** (open decisions #1/#2) may change M2/M3 shapes — resolve before M2 coding.

---
**Created**: 2026-06-23

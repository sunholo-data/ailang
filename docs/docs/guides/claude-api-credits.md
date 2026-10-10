---
sidebar_position: 13
title: Claude API Credit Guards
reviewBy: 2027-01-10
---

# Claude API credit guards

Available in v0.53.4. Cloud use requires a deployed credit gateway, an isolated
Firestore authority, verified executor images, and an explicitly confirmed grant.
Installing the release does not enable the lane.

The coordinator dispatches dedicated Cloud Run jobs. Those jobs receive a signed
capability for one task attempt and authenticate to the gateway with their Google
identity. The Anthropic key stays in the gateway's Secret Manager mount. Dev and
prod use one authority, so they share the allowance.

## Limits and admission

The initial policy supports a confirmed $200 allocation with these limits:

| Limit | Amount |
|---|---:|
| Operating ceiling, including outstanding exposure | $190 |
| Daily ceiling | $6 |
| Task ceiling | $2 |
| Concurrent tasks | 2 |
| Initial canary, including outstanding exposure | $5 |

Each provider request reserves its supported worst-case cost before sending.
Admission checks the account, grant expiry, day, task, concurrency, and canary
limits together. Cache writes/reads and output usage are priced from provider
usage. Unsupported models, request features, or billing categories fail closed.

Every retry, helper request, or tool-loop continuation needs its own reservation.
Unknown usage, truncated streams, or a failed settlement retain conservative
exposure and block further admission. A forwarded request is never refunded just
because a lease expires. Tasks blocked by the budget do not silently switch to
another provider or credential.

## Inspect and control the account

Use `ailang coordinator credits --help` for the complete command contract.
All commands address the canonical gateway with `--remote gcp`, `--gateway`, and
`--account`. The current account name is `anthropic-api-credits`.

| Action | Purpose |
|---|---|
| `status` | Show the grant, expiry, settled/reserved/unresolved spend, and blockers. |
| `confirm` | Record an operator-verified allocation and its evidence. |
| `enable` | Allow guarded canary admission after verification. |
| `disable` | Stop new admission without resetting spend or outstanding exposure. |
| `promote` | Finish canary mode after reconciliation; preserve all spend. |

Mutating actions require an audit reference through `--evidence`. `confirm`,
`enable`, and `promote` require explicit confirmation. Use `--yes` only after
reviewing the canonical state. Operator service-account impersonation is supported
through `--impersonate-service-account`; its Token Creator grant must be scoped
to the operator identity. Executors and coordinators are not credit operators.

## Confirm fresh credits in AILANG

First verify the new allocation in the provider console. Then run `confirm` with
these explicit fields:

| Flag | Value to record |
|---|---|
| `--grant-id` | A new stable allocation/cycle reference. |
| `--expected-grant-id` | The current canonical grant ID shown by `status`; empty for the first grant. |
| `--amount-usd` | The verified grant amount. |
| `--available-usd` | The verified remaining amount allocated to these executors. |
| `--starts-at` | Allocation admission start, in RFC3339. |
| `--expires-at` | Verified expiry, in RFC3339. |
| `--evidence` | An audit reference for the provider verification and allocation decision. |

Confirmation reads canonical status before updating and rejects a stale expected
grant. Reconfirming the same grant adds no allowance. Key rotation, restart,
calendar rollover, and a new month add no allowance either. A different grant
does not erase unresolved exposure from the previous one.

After confirmation, inspect `status` and resolve its blockers before enabling.
Confirmation does not enable a disabled account. Renew only after fresh credits
are verified; expiry without renewal stops admission.

## Deploy and validate

Use the owning infrastructure repository's Terraform and release pipelines. One
production state owns the authority and both environments' guarded jobs. Provision
the private project and IAM first; securely add the signing-secret value outside
Terraform state; then deploy runtime with verified immutable image digests.

Before a live canary, verify the deployed CLI's protocol, identities, key mounts,
ledger isolation, and operator authentication. Confirm the grant, exercise status
and the kill switch, then admit only the approved canary. Reconcile provider usage
and all reserved/unresolved exposure before promotion. Existing task and merge
approvals continue to apply.

If the ledger is unavailable or billing evidence disagrees, keep admission
disabled. Investigate the canonical reservations and provider evidence; do not
reset accounting or rotate a key to restore allowance.

Deployment administrators may have inherited control-plane access. Record that
trust explicitly and inspect project/folder inheritance; secret-level IAM cannot
cancel inherited access. Changes to deployment IAM belong in reviewed Terraform.
Federated provider credentials can change credential distribution, while the
shared credit accounting and manual renewal rules still apply.

See [Coordinator Workers](./coordinator-workers.md) for routing and the
[credit guard design](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-cloud-claude-api-credit-guard.md)
for the admission and accounting contract.

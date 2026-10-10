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

When accounting fails, inspect provider billing evidence alongside the canonical
reservation. A zero or empty console total can reflect reporting delay or display
rounding; it does not prove that a forwarded request was free. Keep admission
disabled until unresolved exposure is reconciled or its complete reservation is
recorded as an audited conservative debit. Never reset the ledger, replace the
grant, or retry an ambiguous request to clear a hold.

## Inspect and control the account

Use `ailang coordinator credits --help` for the complete command contract.
All commands address the canonical gateway with `--remote gcp`, `--gateway`, and
`--account`. The current account name is `anthropic-api-credits`.

| Action | Purpose |
|---|---|
| `status` | Show the grant, expiry, booked/reserved/unresolved spend, and blockers. |
| `requests` | Inspect original request IDs, amounts and states for an explicit task. |
| `conservative-debit` | Permanently count an unknown request’s complete reservation as used. |
| `external-debit` | Count an attended diagnostic or other externally observed spend. |
| `confirm` | Record an operator-verified allocation and its evidence. |
| `enable` | Allow guarded canary admission after verification. |
| `disable` | Stop new admission without resetting spend or outstanding exposure. |
| `promote` | Finish canary mode after reconciliation; preserve all spend. |

Mutating actions require an audit reference through `--evidence`. `confirm`,
`enable`, `promote`, and both debit actions require explicit confirmation. Use `--yes` only after
reviewing the canonical state. Operator service-account impersonation is supported
through `--impersonate-service-account`; grant
`roles/iam.serviceAccountOpenIdTokenCreator` on that operator identity only.
AILANG directly mints an ID token; the broader access-token/signing role is unnecessary. Executors and coordinators are not credit operators.

## Recover an unknown charge conservatively

Recovery commands and retained usage receipts require v0.54.1 or later.

Recovery requires a disabled account, no active tasks or forwarding requests,
and the exact current `--grant-id`. First inspect `status` and
`requests --task-id TASK_ID`. Inspection is bounded to 100 records and fails if
the result would be truncated.

If a complete provider receipt is unavailable, `conservative-debit` counts the
entire original reservation as permanently used. `--amount-usd` must exactly
match that reservation. The original unknown outcome remains in history; no
provider receipt, actual cost or upstream ID is invented.

```sh
ailang coordinator credits conservative-debit --remote gcp \
  --gateway https://GATEWAY.run.app --account anthropic-api-credits \
  --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com \
  --grant-id CURRENT_GRANT --request-id ORIGINAL_REQUEST_ID \
  --amount-usd ORIGINAL_FULL_RESERVATION --evidence AUDIT_REFERENCE --yes
```

Use `external-debit --debit-id UNIQUE_REFERENCE --amount-usd AMOUNT` for exposure
from an attended direct diagnostic. Choose `--kind conservative` when its
outcome or attribution is uncertain, or `--kind provider-verified` only with a
retained complete provider receipt and reviewed pricing. Supply the same common
flags, grant, evidence and `--yes`. A console balance is not a request receipt.
Possible overlap may be counted conservatively; these commands cannot refund it.

`Settled` is the total booked debit against the allowance. `ConservativeDebited`
identifies the subset without verified provider spend. Moving a reservation into
that subset creates no available capacity. Exact replays add nothing; changed
IDs, grant, amount, identity or evidence conflict. Recovery does not enable the
account and cannot clear unrelated accounting failures or over-reservation costs.

All external and conservative debits consume the same initial $5 canary allowance.
They cannot prove a successful gateway canary: `CanaryGatewaySettled` must contain
positive verified gateway spend, with outstanding requests resolved and provider
evidence reviewed. Fresh grant confirmation retains immutable history and follows
the renewal rules below.

After durable settlement, the gateway logs `credit_usage_settled` with internal
and provider IDs, the exact pricing revision, cost in micro-USD and verified input,
cache-read, cache-write TTL and inclusive output totals. Context pricing follows
those counts and the rate card; admission enforces the supported service and
residency classes. These receipts contain no prompt, response text or credentials.
Retain the live canary receipts alongside provider billing evidence. Failure
observations are separate diagnostics and cannot authorize settlement.

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
Set Cloud Run `custom_audiences` to the same gateway origin used by clients and
application JWT validation when it differs from the generated service URL.
IAM invocation and application identity checks remain required.
Routine floating dev builds preserve the guarded runtime's release pins;
versioned release/promotion owns its image updates.

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

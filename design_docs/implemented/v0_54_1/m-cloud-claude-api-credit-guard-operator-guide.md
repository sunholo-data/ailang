# Claude API credit lane operations

The v0.53.4 lane was deployed with the account disabled, followed by an admitted dev canary that froze on unverified streaming usage. The installed cloud Claude CLI 2.1.296 passed fake-only completion and tool-loop diagnostics. The repaired v0.54.1 usage contract, bounded receipts and audited recovery passed new dev/prod canaries and audited promotion. Regular dev/prod evaluator activation passed Terraform CI; the original full hold was conservatively debited, with no refund or invented receipt. The gateway rejects unverified beta headers and body fields. See the [protocol contract](m-cloud-claude-api-credit-guard-cli-contract.md) and [recovery addendum](m-cloud-claude-api-credit-guard-recovery.md). Other providers’ credits remain a follow-up.

## Initial grant evidence

The attended operator confirmed on 2026-10-10 that $200 is available and allocated to this lane, expiring 2026-10-28. The provider key is stored at `projects/ailang-multivac/secrets/ailang-anthropic-api-key`. The key is version 4 in Holosun ApS organization, `ailang` workspace. Attended diagnostics read it only in memory and passed it over stdin; no key value was logged or persisted. Use `2026-10-28T00:00:00Z` as a conservative cutoff pending an exact provider expiry timestamp; admission also requires the full request deadline plus five minutes of headroom.

Before activation, verify the Anthropic organization/workspace, a stable cycle reference, all other spenders sharing that credit pool, and the newly installed secret version. A workspace or key alone does not isolate organization credits. Configure the gateway with that verified organization/workspace; never use placeholder identifiers to activate it.

## Confirm through AILANG

The gateway operator identity is separate from coordinator/job identities and ordinary task approvals. A human can use scoped service-account impersonation with `roles/iam.serviceAccountOpenIdTokenCreator` on the sanctioned operator SA only. That SA must be in both Cloud Run's invoker policy and the gateway operator allowlist. Google IAM audit records retain the impersonation chain; the ledger records the authenticated operator SA and the supplied evidence.

```sh
ailang coordinator credits status --remote gcp \
  --gateway https://GATEWAY.run.app --account anthropic-api-credits \
  --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com

ailang coordinator credits confirm --remote gcp \
  --gateway https://GATEWAY.run.app --account anthropic-api-credits \
  --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com \
  --grant-id VERIFIED-ORG-CYCLE-REF --amount-usd 200 --available-usd 200 \
  --starts-at 2026-10-10T00:00:00Z --expires-at 2026-10-28T00:00:00Z \
  --evidence 'Operator verified grant, expiry and exclusive allocation in Console' --yes
```

Replace the timestamps and reference with verified evidence. For a later grant, pass `--expected-grant-id CURRENT-GRANT-ID`. The command first reads canonical status to protect against stale updates. Repeating the current grant adds nothing; replaying an older grant or changing its original amounts/dates fails. Confirmation does not enable a disabled account, clear reconciliation failures, or refund outstanding reservations. Executors read the updated authority on their next request; no coordinator restart is needed.

## Enable and stop

After IAM verification, matching built job/gateway images and supported protocol fixtures are verified, enable the account for the canary only. The authority enforces a separate aggregate $5 canary sublimit across settled, reserved and unresolved smoke requests, inside the same grant. After complete usage is reconciled against provider billing, use the audited `promote` command to remove that sublimit without resetting spend; then approve rotation activation in the deployment repo. Promotion refuses outstanding or unresolved requests and requires positive verified gateway canary spend (`CanaryGatewaySettled`); conservative and external debits do not qualify.

```sh
ailang coordinator credits enable --remote gcp --gateway https://GATEWAY.run.app \
  --account anthropic-api-credits --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com \
  --evidence 'IAM and request contract reviewed; begin guarded canary' --yes

ailang coordinator credits promote --remote gcp --gateway https://GATEWAY.run.app \
  --account anthropic-api-credits --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com \
  --evidence 'Reserved canary completed; provider usage and billing reconciled' --yes

ailang coordinator credits disable --remote gcp --gateway https://GATEWAY.run.app \
  --account anthropic-api-credits --impersonate-service-account OPERATOR@PROJECT.iam.gserviceaccount.com \
  --evidence 'Operator kill switch' --yes
```

The kill switch stops new sends; it cannot undo an already forwarded billable request. Status reports confirmed/operating ceilings, settled and reserved amounts, unresolved exposure, expiry, UTC admission-day accounting, active tasks and blockers. The $6 day limit attributes exposure to the date a request was admitted; it is not a provider invoice-day cap when a request spans midnight. The full grant, task and canary reservations remain charged throughout. An incomplete stream, cancellation, unknown billing category or storage uncertainty never refunds sent exposure. Investigate the request ID from `X-Ailang-Credit-Request` against gateway and provider evidence before resolving it. There is deliberately no operator command to blindly erase exposure or clear reconciliation.

## Pilot contract

The canary's trusted agent registry entry must explicitly set
`credit_max_cost_usd: 2`, alongside `credit_account: anthropic-api-credits`,
`credit_gateway_url` and `credit_job_identity`. Use `provider: claude`,
`auth_mode: apikey`, `model: claude-haiku-5-5`, a default/Go Claude variant and
a timeout of at most `50m`. Missing, nonfinite or out-of-range credit budgets
fail before admission; the legacy Claude provider/global budget is not inherited.
This scoped setting preserves the existing OAuth and request-key budgets. The manual canary is registered in dev and prod; account admission remains
disabled until the deployment gates pass. There is no per-agent `enabled` flag.
Register canaries explicitly with
`auto_merge: false` and `skip_approval: false` for the reviewed canary.

Only exact Haiku 5.5 Messages requests are admitted. Every request reserves the entire model context at the highest supported cache-write rate and requested output (thinking included), up to $1.32. Complete verified usage settles that reservation; estimated prompt counts and CLI budgets cannot authorize spending. Helper/subagent requests pass through the same endpoint and get separate reservations. Paid server tools, images/documents, fallback/advisor/compaction inference, batch/priority/residency premiums and unknown beta features are blocked until separately bounded.

A task has a unique admitted attempt, bound job identity, model, budget and lease. The job supplies both a Google ID token for the gateway audience and a signed capability in `x-api-key`; neither is a provider credential. Task timeouts are capped at 50 minutes for token lifetime. Failed launch admission becomes a visible permanent budget block, not automatic credential/model fallback.

## Rollout evidence

See [deployment proposal](m-cloud-claude-api-credit-guard-deployment.md) and `deploy/claude-credit-gateway/`. The private operations audit records actual deployments and outstanding gates; the original deployment proposal records the staged design. The private authority project is mandatory: ordinary executor SAs have write access to the existing application database. All runtime identities with inherited provider-secret access must be narrowed before activation. Production images follow ailang-multivac's branch/verified-version promotion workflow; do not manually rebuild/promote from an unverified source head.

## Audited conservative recovery (M6 addendum)

See the [recovery contract](m-cloud-claude-api-credit-guard-recovery.md) and the
[website command guide](../../../docs/docs/guides/claude-api-credits.md#recover-an-unknown-charge-conservatively).
An operator can convert the entire original unknown reservation to a permanent
conservative debit, retaining history and creating no available capacity. External
attended diagnostic exposure must also be booked before further paid calls.
`Settled` reports total booked debits; `ConservativeDebited` identifies amounts
without verified provider spend. Account stays disabled after recovery. These
commands cannot erase exposure, clear unrelated errors, or prove a gateway canary.

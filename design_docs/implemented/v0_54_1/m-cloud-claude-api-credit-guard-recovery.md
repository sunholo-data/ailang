# M-CLOUD-CLAUDE-API-CREDIT-GUARD: conservative recovery addendum

**Date:** 2026-10-10. **Status:** Implemented in v0.54.1; audited recovery, dev/prod canaries and guarded evaluator activation passed independent evaluation94/100. This completes M6 of the approved sprint, without increasing its allocation or limits.

Related: [design](m-cloud-claude-api-credit-guard.md), [sprint](m-cloud-claude-api-credit-guard-sprint-plan.md), [operator guide](m-cloud-claude-api-credit-guard-operator-guide.md).

## Problem and decision

The first deployed canary has an unresolved 1,320,000 micro-USD reservation with no retained authoritative receipt. An attended direct diagnostic has a separate ambiguous 1,000,040 micro-USD exposure. A later diagnostic has complete provider usage costing 4 micro-USD. The operator also reported $199.77 remaining ($0.23 consumed); attribution may overlap these requests. No receipt can be invented and no uncertain exposure can be refunded.

An operator may close an unknown request by permanently debiting its **entire original reservation**. This is conservative budget accounting, not verified provider spend. The complete original request, unknown outcome and authenticated operator/evidence/time remain in history. Separate external diagnostic debits must also enter this shared authority before further paid work. Conservatively include the $0.23 observation as an additional debit; possible overlap reduces capacity rather than silently refunding exposure.

## Contract

- Recovery is only on authenticated, allowlisted operator routes. Coordinators and executor capabilities cannot invoke it. Identity is injected server-side; caller-supplied operator identity is rejected.
- Mutations require a disabled account, the exact current grant ID, no live task leases and no forwarding requests. No recovery may clear an unrelated reconciliation reason or an over-reservation charge. Request recovery accepts only state `unresolved` with the ordinary `upstream outcome unknown` reason and no recorded actual cost above its reservation.
- `ConservativeDebit{AccountID, GrantID, RequestID, ExpectedAmount, Operator, Evidence}` closes exactly one request. ExpectedAmount must equal the original reservation. Read and validate account/request/task/day before writes. Move the full amount from reserved to settled in all existing account/task/day/canary counters, and increment `ConservativeDebited` (a subset of booked Settled). Request state becomes `conservative-debit`; Actual and UpstreamID are not fabricated. Record an immutable debit audit carrying operator, evidence, grant, request, amount and timestamp. Clear only the ordinary unknown-outcome blocker, only after Unresolved and Forwarding both reach zero. Account stays disabled.
- `ExternalDebit{AccountID, GrantID, DebitID, Amount, Kind, Operator, Evidence}` records a unique immutable external debit. Kind is `conservative` or `provider-verified`; the latter is an operator attestation backed by the retained provider receipt, not an automatic claim from a Console balance. Add its full amount to account and current UTC-day settled counters and, before promotion, canary settled counters. Conservative kind also increments the conservative subset. Validate remaining operating, daily and canary limits. No external debit creates a task, enables the account or clears reconciliation.
- Identical replay is idempotent, including identity/evidence/amount/grant; changed replay fails without mutation. Request recovery never permits later settlement or ReleaseUnsent to refund the debit. Grant replacement retains immutable history; conservative subset resets with the corresponding settled grant counter only on explicitly confirmed fresh credits.
- `requests --task-id ID` is an operator-only bounded read (up to 100 requests; fail loudly if truncated). Memory and Firestore stores implement the same read contract. Validate account/task IDs; return no prompt, response, key or credential. This gives the operator original request IDs/amount/state before recovery.
- Status explicitly distinguishes booked total and its conservative subset. Existing `Settled` is the total amount booked against allowance, including conservative debits; `ConservativeDebited` identifies the subset without provider verification. No new capacity is created by converting a hold into a debit.
- Promotion still requires no reserved/forwarding/unresolved amounts, no reconciliation and a complete verified **gateway** canary response with positive spend. Add `CanaryGatewaySettled`, incremented only by actual gateway settlement; external or conservative debits cannot satisfy this requirement. Existing settled totals alone are insufficient after the upgrade. An operator still supplies canary/provider evidence when promoting.
- Successful settlement emits a bounded billing-only receipt: internal task/request, sanitized provider request/message IDs, exact model/pricing revision, charged micro-USD and the verified typed usage totals (input, cache-read, cache creation by TTL and inclusive output). Context pricing is derived from these counts and the exact rate-card revision; admission already enforces the supported standard/default service and global/default residency classes. Emit only after durable settlement succeeds. No prompt, response text, credential, headers or arbitrary provider fields enter this log. Missing/failed receipts never create settlement or replace provider reporting; durable request accounting remains authoritative. Retain these receipts for the live dev/prod canary audit.

## M6 recovery subplan

1. Authority and storage (~250 production LOC): write failing atomic/idempotency/limit/concurrency/renewal tests, implement debits and bounded request inspection. Check that unchanged total exposure survives recovery and external receipts cannot qualify as a gateway canary.
2. Operator gateway and CLI (~150 production LOC): strict operator-only routes, explicit flags/grant/amount/evidence/`--yes`, canonical JSON output and help/docs. Reject malformed payloads, stale expectations, unauthorized actors and unsafe redirects before provider access. Run focused race tests and full required checks; independent sprint evaluation.
3. Deploy through the existing release and Terraform CI paths. Keep account disabled. Read original request, apply conservative request debit 1.32, external conservative debit 1.00004, external verified receipt 0.000004 and conservative balance observation 0.23. Expected booked total is $2.550044, conservative subset $2.550040, no reserved/unresolved amounts, account disabled and canary incomplete. Remaining aggregate $5 canary capacity is $2.449956.
4. Enable only for bounded dev and prod canaries. Each must retain authoritative usage/IDs and settle automatically, within the existing task and aggregate limits. If either fails, disable and retain exposure. Promote and activate rotation only after passing canary evidence and the remaining M6 evaluation. Rollback is audited disable; no accounting reset.

## Acceptance criteria

- No uncertain charge increases available capacity when recovered; provider-verified and conservative amounts are visibly distinct.
- Recovery is atomic under concurrent clients and retry; grant, amount, identity and evidence replay conflicts do not mutate the ledger.
- All existing financial, expiry, concurrency, strict usage and identity guards remain active.
- The live disabled ledger contains all attended diagnostic exposure before another provider send.
- Documentation and the rollout audit include conservative debits, receipt IDs, original failure and federation/IAM follow-ups. M6 remains incomplete until real guarded canaries and production activation are verified.

## Completed rollout

The repaired v0.54.1 release passed exact-image CLI proof and dev/prod canaries.
Recovery booked2,550,044 microUSD with2,550,040 conservative; gateway canaries
added13,025 verified microUSD. Audited promotion and the existing read-only
Haiku evaluator activation passed independent evaluation94/100 and Terraform
CI. All held/unresolved/forwarding exposure is zero at the rollout snapshot.
See [retained proof](../../verification/cloud-claude-api-credit-guard/rollout-v0.54.1.json).

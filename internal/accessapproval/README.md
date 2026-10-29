# Durable local access approvals

`accessapproval` is the operator-owned lifecycle behind `ailang access-approvals`.
It reuses `coordinator.db` and the existing `approval_requests` table. Its dedicated
`durable_access_v1` type, `access_*` SQL statuses and synthetic task ids keep it out
of legacy merge/secret watchers. It does not change synchronous secret APIs.

The CLI is a **trusted local authority**, not a worker tool or an authentication
server. Its host adapter must inject authenticated owner/worker/actor identities,
check record scope before disclosure, restrict direct decisions to human actions,
and prevent workers from opening the operator state directory or invoking this
binary. Tailscale supplies private transport, not these application permissions.

Only `dummy` / `canary` with policy `1` is registered. There is no real credential
resolver, arbitrary reference selection, shell execution or worker-written receipt.
Production protected operations require separate reviewed implementation and
credential isolation; do not point this development slice at a real vault.

See [the operator protocol](../../docs/internal/access-approvals.md) for exact
JSON shapes and lifecycle behavior. Focused checks:

```sh
go test ./internal/accessapproval -race -count=3
go test ./cmd/ailang -run TestAccessApprovals -count=1
```

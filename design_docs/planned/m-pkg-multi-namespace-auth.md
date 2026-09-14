# M-PKG-MULTI-NAMESPACE-AUTH — Scoped registry keys (write-restricted, read-all)

**Status:** planned → implementing (2026-09-14)
**Trigger:** Daneel needs to publish packages without holding the superuser key.
**Lane:** registry infra (`cmd/registry-validator`) — not core, not an extension.

## Problem

The registry validator authorizes every write (`/publish`, `/unpublish`, `/rebuild-index`)
against ONE shared key (`REGISTRY_API_KEY`). Whoever holds it can publish or unpublish under
any namespace. `// Step 5: Namespace auth — deferred` has sat in `main.go` since the registry
shipped; the Firestore database and the `FIRESTORE_NAMESPACE_COLLECTION` env were
provisioned in `terraform-registry/` for it and are read by nothing.

Reads are already open: the bucket grants `allUsers` `objectViewer`, and `/api/*` is
unauthenticated. Nothing changes there.

## Design

### Principal model

A request carries `X-API-Key`. It resolves to exactly one principal:

| key | principal | scopes |
|---|---|---|
| equals `REGISTRY_API_KEY` | `superuser` | `*` (everything, incl. admin endpoints) |
| SHA-256 matches a Firestore doc in `ailang_registry_keys` | that doc's `owner` | the doc's `scopes` |
| anything else / revoked | — | 403 |

Scopes are `path.Match` globs over the full package name: `daneel/*`, `sunholo/daneel_*`,
`sunholo/exact_pkg`. A write to package `P` is allowed iff some scope matches `P`.
`/rebuild-index` and `/admin/*` are superuser-only.

### Storage

Collection `ailang_registry_keys` in the `ailang-registry` Firestore database (env
`FIRESTORE_DATABASE`, already wired). Doc id = hex SHA-256 of the key. Plaintext is never
stored; it is returned exactly once, at mint time.

```
{ owner: "daneel", scopes: ["daneel/*"], note: "...", created_by: "superuser",
  created_at: <ts>, revoked_at: <ts|null> }
```

Keys are prefixed `ailr_` so a leaked one is recognisable in a scan.

### Minting

`POST /admin/keys {owner, scopes, note}` (superuser) → `{id, key, owner, scopes}`.
`GET /admin/keys` → list (id, owner, scopes, revoked). `DELETE /admin/keys/<id>` → revoke.
CLI wrapper: `ailang pkg key create|list|revoke` (reads `AILANG_REGISTRY_API_KEY`, must be
the superuser key).

### Provenance

`metadata.json.published_by` is set from the principal's owner for scoped keys (the
client-supplied `X-Publisher-Identity` header stays for superuser publishes) — so the
registry records WHO published, not just that a valid key was used.

### No silent fallbacks

If `FIRESTORE_DATABASE` is unset the validator boots without a key store; scoped keys are
then rejected with an explicit "scoped keys not configured on this server" 403, and the
superuser key keeps working. Firestore errors during lookup are 500s, not 403s.

## Non-goals

- Self-registration of namespaces (still CRAN-style: only the superuser mints).
- Per-package owner ACLs beyond scope globs.
- Replacing the superuser key. It stays as break-glass and as the minting authority.

## Acceptance

- Publish with a `daneel/*` key: `daneel/foo` → 200, `sunholo/foo` → 403 naming the owner.
- Unpublish honours the same scopes.
- Revoked key → 403. Unknown key → 403. Superuser unchanged.
- `ailang pkg key create --owner daneel --scope 'daneel/*'` prints the key once.
- Unit tests cover the authoriser with an in-memory store (no Firestore in CI).

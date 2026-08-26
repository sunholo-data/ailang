---
sidebar_position: 11
title: Browser Auth Profiles
description: Persistent authenticated browser identity for AILANG agent runs, without giving a model a password
---

# Browser Auth Profiles

AILANG browser sessions can start **already logged in**, from a named, versioned
profile — without any AI model ever receiving a password, a canonical profile, or
the authority to change one.

This guide is for the operator who creates and rotates those profiles. If you
just want to run a browser eval anonymously, see
[Browser Sessions](./browser-sessions.md) instead.

:::danger Production use is blocked on two P0 follow-ups
Authenticated sessions are **not yet cleared for production use against real
accounts**. Two designs must land first:

- **M-BROWSER-EGRESS-BOUNDARY** — destination policy is not enforced below the
  browser. Playwright origin flags are application controls, not network
  containment: redirects, DNS, WebSockets, service workers, and browser
  subprocesses can all reach destinations the allowlist names.
- **M-BROWSER-ARTIFACT-DATA-POLICY** — screenshots, video, HAR, console output,
  and downloads from an authenticated session are not yet classified, ACL'd, or
  given a retention policy.

Until both ship, `--egress-ack` is required to create a profile at all. It
records an explicit, audited operator decision to proceed without enforcement.
Use dedicated throwaway accounts on systems you own.
:::

## The threat model in one paragraph

A saved browser state is **a credential, not an artifact**. Cookies, local
storage, IndexedDB tokens, and virtual passkeys can impersonate the account just
as well as the password that produced them. Hiding the password while
mishandling the state buys nothing. AILANG therefore treats profile material as
secret-class throughout: it is sealed at rest, materialized to an owner-only file
for exactly one run, destroyed on every exit path, and never serialized into a
prompt, tool argument, result, log, trace, or artifact manifest.

A valid authenticated session necessarily grants the browser authority to act as
that account. This design narrows and audits that authority; it cannot make a
compromised session harmless.

## How the pieces fit

```text
you + your password manager          ← the password stops here
        │  (headful, recording off)
        ▼
  storage-state.json                 ← captured by you, never by a model
        │  ailang browser-profile bootstrap
        ▼
  profile registry                   ← sealed AES-256-GCM, 0600 under a 0700 tree
        │  resolve → preflight → lease → materialize
        ▼
  disposable 0600 file               ← exactly one run, destroyed afterwards
        │
        ▼
  AI + Playwright MCP                ← sees an authenticated browser and nothing else
```

The model has browser action authority for the declared origins and the task
duration. It has no vault access, no profile-read access, no way to obtain a
hosted context ID, and no way to request persistence.

## Creating a profile

### 1. Use a dedicated, least-privilege account

Never import your everyday browser profile. Create a separate automation account
that:

- has no access to a personal inbox, password vault, billing, or admin surface;
- can be revoked and rotated without disrupting a human;
- is permitted by the site's terms to be automated.

### 2. Capture the state yourself, headful

Sign in by hand, so your password never enters AILANG:

```bash
npx -y @playwright/mcp@0.0.79 --isolated --save-storage ./crm-state.json
```

Complete any CAPTCHA or 2FA in that window. Recording must be off. This session
is **not an eval** and must never be compared against one.

### 3. Publish it

```bash
ailang browser-profile bootstrap crm-readonly-eu \
  --state-file ./crm-state.json \
  --provider local-playwright \
  --origins https://crm.example.com \
  --account-class readonly \
  --max-concurrent 1 \
  --egress-ack
```

Then delete your copy — the registry holds the only sealed one:

```bash
rm ./crm-state.json
```

### Policy flags, and what "unset" means

Every policy field **fails closed** when unset. That is deliberate: a profile
that forgot to declare something must refuse, not guess.

| Flag | Unset means | Notes |
|------|-------------|-------|
| `--origins` | *required* | Exact origins only — `scheme://host[:port]`. No paths, no wildcards. A subdomain is a different origin. |
| `--account-class` | `readonly` | `readonly`, `mutable`, or `privileged`. The username is never stored, only the class. |
| `--max-concurrent` | `1` | Set to 1 for any site that invalidates simultaneous logins. |
| `--allow-artifacts` | allow **nothing** | Deny-by-default. An unset artifact policy denies the session outright; an explicit empty list is a real "allow nothing" decision. |
| `--allow-human-takeover` | denied | |
| `--expires` | no expiry | RFC3339. After it passes, the profile fails closed with `browser_auth_profile_expired`. |
| `--egress-ack` | *required* | Records that you accept an unenforced egress boundary. |

## Running an authenticated eval

```bash
ailang eval-suite --agent \
  --benchmarks crm_readonly_fixture \
  --browser-provider local-playwright \
  --browser-profile crm-readonly-eu@latest
```

`latest` is resolved to a concrete version **before the run starts**, and the
concrete version is what gets banked — so a result always records the identity
that actually ran, not a moving reference.

Ordinary eval runs are **always read mode**. They load an immutable snapshot,
cannot persist anything back, and discard their state. Authenticated runs also
disable trace and video recording, because private page content is as sensitive
as the login state.

### What lands in the result

| Field | Example | Why it is safe |
|-------|---------|----------------|
| `auth_profile_alias` | `crm-readonly-eu` | Operator-chosen name |
| `auth_profile_version` | `v7` | Never `latest` |
| `profile_hash` | `sha256:0f1e2d3c4b5a6978` | Truncated, domain-separated digest of the *stored* material — identifies a version, does not reveal one |
| `auth_lease_id` | `lease-01HZX…` | Opaque correlation id; grants nothing |
| `auth_mode` | `read` | |

Cookies, storage state, sealed blobs, and hosted context IDs appear in **none**
of them.

## Rotation and revocation

### Rotating a profile

Automated refresh needs a **reviewed, site-specific login adapter** implementing
`auth.LoginAdapter`. AILANG deliberately ships no generic, model-authored login
step: a general "fill in the login form" capability driven by a model is exactly
what this design refuses to build. Until an adapter exists for your site,
re-bootstrap with a freshly captured state under a new version:

```bash
ailang browser-profile bootstrap crm-readonly-eu --version v2 \
  --state-file ./crm-state-v2.json --origins https://crm.example.com --egress-ack
```

Versions are immutable. Publishing `v2` records `v1` as its rollback pointer, and
pinned references to `v1` keep working.

### Revoking

```bash
ailang browser-profile revoke crm-readonly-eu@v1 --reason "credential rotation"
```

:::warning Revocation is not enough on its own
Revoking stops **AILANG** from using that version. It does **not** end sessions
the website already issued. A full incident response is:

1. `ailang browser-profile revoke <alias@version> --reason "..."`
2. Sign out all sessions in the site's own security settings
3. Rotate the account password and any OTP seed
4. `ailang browser-profile purge <alias@version>` to destroy the stored material
5. `ailang browser-profile audit` to establish which runs used it, and when
:::

`purge` refuses to run on a version that has not been revoked — the ciphertext is
kept until then so an incident can still be investigated.

## Key management

The default local key protector writes a key file next to the data it protects:

```
~/.ailang/browser-profiles/
  keys/local.key        ← 0600, the KEK
  profiles/<alias>/     ← 0700
    v1.material         ← 0600, AES-256-GCM sealed
    v1.meta.json        ← 0600, safe metadata only
  audit.jsonl           ← 0600, append-only
  materializations/     ← disposable, swept by `gc`
```

**This is local development and single-host operation only.** Anyone who can read
the host can read the profiles. For anything shared or durable, supply a
KMS-backed `auth.KeyProtector` — the interface is unchanged, so nothing else
moves. Which protector a deployment uses (macOS Keychain, a 1Password item,
Cloud KMS, Secret Manager) is a decision for deployment review.

For Cloud Run, the shape is: profile material in a private bucket with CMEK, the
KEK in Cloud KMS, and the service account granted `cloudkms.cryptoKeyVersions.useToDecrypt`
only. The service account — not the model, and not the eval harness — is the
principal that can unwrap.

## Account pools

One account is acceptable only when **all** of these hold:

- it is a dedicated, least-privilege automation account;
- tasks are read-only or serialized;
- the site permits automation and its concurrent-session behavior is understood;
- compromise can be contained by revoking sessions and rotating the account;
- no personal inbox, vault, billing, or admin surface is reachable.

For parallel work that changes server-side state, use a pool of distinct
accounts. The broker leases one profile per worker and records the account class,
never the username. Exhaustion is a deterministic
`browser_auth_lease_conflict` — the pool never silently hands two workers the
same account, because that is the collision the pool exists to prevent.

## Housekeeping

```bash
ailang browser-profile inspect              # safe metadata for every profile
ailang browser-profile inspect --json       # same, machine-readable
ailang browser-profile audit --limit 100    # who used what, when, and the outcome
ailang browser-profile gc --max-age 1h      # sweep materializations a crash left behind
```

Run `gc` at boot on any host that runs authenticated evals. A crash between
decrypt and destroy leaves plaintext on disk; the sweep removes it and records a
structured audit event for each one.

## Failure categories

All of these are stable and queryable from banked results:

| Category | Meaning | Retry? |
|----------|---------|--------|
| `browser_auth_profile_not_found` | Unknown alias or version | No — fix the reference |
| `browser_auth_profile_expired` | Past the declared expiry | No — refresh first |
| `browser_auth_profile_revoked` | Permanently retired | No |
| `browser_auth_lease_conflict` | Held by another run, or pool exhausted | Yes, after a wait |
| `browser_auth_scope_denied` | Origin, egress, takeover, or provider mismatch | No — fix the policy |
| `browser_auth_refresh_required` | Login expired, 2FA, or post-login check failed | No — rotate |
| `browser_auth_materialize_failed` | Decrypt, envelope, or filesystem failure | Sometimes |
| `browser_auth_writeback_denied` | An ordinary run tried to persist | No — this is the guard working |
| `browser_auth_artifact_policy_denied` | Artifact class not permitted | No |
| `browser_auth_cleanup_failed` | Plaintext may remain on disk | **Investigate immediately** |

## What this does not do

- Give a model a password, OTP seed, recovery code, or vault access.
- Import or automate your normal Chrome profile.
- Protect profiles from a fully compromised runtime principal.
- Let model-authored JavaScript implement login or refresh.
- Solve CAPTCHAs or bypass a site's terms and security controls.
- Persist arbitrary run changes into canonical state.
- Contain browser network traffic — see the egress warning at the top.

## Related

- [Browser Sessions](./browser-sessions.md) — the anonymous provider layer this builds on
- `design_docs/implemented/v0_33_4/m-browser-auth-profiles.md` — the full design and threat model
- [Playwright authentication](https://playwright.dev/docs/auth)
- [Browserbase Contexts](https://docs.browserbase.com/platform/browser/core-features/contexts)

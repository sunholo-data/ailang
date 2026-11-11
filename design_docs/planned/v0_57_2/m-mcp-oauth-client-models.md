# M-MCP-OAUTH-CLIENT-MODELS: the OAuth client models and serve-api identity that open the Google integration paths

**Status**: Planned. Design Freeze D1–D8 RATIFIED by Mark on 2026-11-05 (all as recommended). Sprint in progress. Vendor measurements P0-1…P0-5 still gate enabling the features in a deployment.
**Target**: `sunholo/mcp_oauth` **0.2.0** (ailang-packages) and AILANG **v0.57.2** (serve-api, `ailang mcp check`)
**Priority**: P1. Gemini Enterprise (GE) cannot sign users in to any AILANG service until M1a ships,
and the hosted executor cannot go to prod until M2 ships.
**Estimated**: 9–11 engineering days across three lanes (package 5.5d, serve-api 3d, mcpcheck 1.5d),
plus the PENDING measurements in §P0, which need a person at the GE and Google consoles.
**Dependencies**:
- `sunholo/mcp_oauth` 0.1.2 (shipped; [m-mcp-oauth-package](../../implemented/v0_52_0/m-mcp-oauth-package.md)). Every change here is additive to it.
- #1631 / PR #2112, the single argument binder (§M4). M2 edits the same files, so M2 lands after #2112.
- Ground truth: [integrations review, internal](../integrations-review-internal-2026-11-05.md) and
  [external](../integrations-review-external-2026-11-05.md) (vendor quotes, fetched 2026-11-05).
**Baselines read**: ailang `origin/dev` c7c568a2f; ailang-packages `origin/main` 1c70be9
(`packages/mcp-oauth`, read-only); docparse `origin/main`; live `docparse.ailang.sunholo.com`, 2026-11-05.
**Routing** ([PROGRAM](../../PROGRAM.md)): package work (lane: package) and two AILANG fixes (lanes:
serve-api, mcpcheck). No motoko change.

**Quorum triggers fired**: **#1**, because Design Freeze items D1–D8 need a human decision. **#4**,
because the load-bearing premises are Google, Anthropic and OpenAI OAuth contracts that we do not
control. #2 does not fire: the Bearer gate contract is extended, not overridden (Conflict Surface §2).
#3 does not fire. The quorum record is in §Quorum record.

---

## Problem Statement

`sunholo/mcp_oauth` 0.1.2 serves one client model: **public CIMD clients** that authenticate with
`none`, always rotate refresh tokens, and are found by fetching the `client_id` URL (V1–V4). That
covers Claude and ChatGPT. The internal review found that it blocks every Google path:

| Platform | What its OAuth client does | What 0.1.2 does | Source |
|---|---|---|---|
| **Gemini Enterprise, custom MCP data store** | An admin registers GE as a **pre-registered client with a client ID and secret**, redirect `https://vertexaisearch.cloud.google.com/oauth-redirect`. It uses HTTP Basic by default, and PKCE is optional | No client secrets, and no registered clients except CIMD URLs. The token endpoint ignores client authentication | E1, E2, V3 |
| **Google Connected Apps** (Gemini app + Search) | Checks CIMD, then falls back to DCR. The older OAuth guide says a `registration_endpoint` is **required** even with CIMD. It asks for a **non-rotating** refresh token, and calls `/revoke` when the user unlinks | No `registration_endpoint`, always-rotating refresh tokens, no revocation endpoint | E5–E8, V1, V4, V8 |
| Claude (fallback) | Uses DCR when CIMD is not advertised with `none` | CIMD works; DCR is absent | E12 |
| ChatGPT | Picks a token-endpoint method from the **intersection** of its methods and ours. Its live CIMD document still carries the legacy singular `token_endpoint_auth_method: private_key_jwt` | Works with `none` today, because the singular field is never read (V2). Nothing tests that it stays that way | E16, E17 |

serve-api adds three more gaps (internal review §3 items 3, 5 and 6):
1. **The token verifier is boolean** (`(string) -> bool`, V9). No subject, client or scopes reach the
   tool. The executor's per-user rate limit is its **prod blocker** (GE design P1b), so each tool
   would have to re-read `Authorization` from `_headers` and verify the token a second time.
2. **`serverInfo` is hardcoded** to `ailang-api 0.8.1` (V15). Every listed service, AILANG Parse
   included, ships under that name, and reviewers see it.
3. **The listed surface leaks the module inventory.** `ailang://meta/modules` is registered on both
   MCP surfaces from the raw module map. Measured locally: on `/mcp/connect/`, with
   `--no-introspection`, it lists `@noexpose` helpers, the token verifier's name, `@mcp_agent_only`
   tools and the absolute module path (V18).

`ailang mcp check` also WARNs on every CIMD-only server: its framing probe sends a non-URL `client_id`
that a CIMD server rejects before any consent page loads (V20). The WARN is false, and adopters learn
to ignore it.

**Who is affected:** every hosted AILANG MCP service that wants a Google listing (AILANG Parse and the
executor first), and the executor's prod promotion.

## Goals

**Primary goal:** one `sunholo/mcp_oauth` authorization server signs in Claude, ChatGPT, GE and
Google Connected Apps clients, each with its own client model and refresh policy, and serve-api hands
the verified identity to the tool. Docparse and the executor keep working with no code change.

**Success metrics** (each is a command):
1. **Backward compatibility.** The 0.1.2 test suite (`tests/flow_check.sh`, `tests/e2e_flow.sh`,
   `tests/cimd_check.sh`, `tests/ifc_leaks.sh`, `tests/lint.sh`) passes **unchanged** against 0.2.0.
   `core.asMetadata(issuer)` output is byte-identical to 0.1.2 (golden test). `executor_oauth/test_oauth.sh`
   passes 21/21 with the lockfile bumped to 0.2.0 and no source change.
2. **The GE shape works end to end** in a local fixture. An authorization-code flow for a
   pre-registered confidential client, with `client_secret_basic` and the GE redirect URI, yields
   tokens. A wrong secret gets `401 invalid_client`.
3. **DCR abuse is bounded.** The 11th registration from one caller inside the window is refused, and
   an unused registration is unknown after its TTL. Both are tests.
4. **Every new threat (T11–T17) maps to a named test or a Z3 contract** (§Threat model).
5. **Identity reaches the tool, and only through the gate.** `ailang serve-api` with an identity
   verifier binds a non-empty `_auth.subject` on a gated `/mcp/connect/` call. The same export is
   absent from `/mcp/`, REST and A2A. A `_auth` value or `X-Ailang-Mcp-Identity` header sent by the
   client is ignored (Go tests).
6. **Listing hygiene.** `initialize` returns the configured `serverInfo`; `resources/list` on
   `/mcp/connect/` does not contain `ailang://meta/modules`; `ailang mcp check --target both` on a
   CIMD-only fixture reports `authorization-framing PASS`, not WARN.

## Threat model additions (T11–T17)

T1–T10 from 0.1.0 stand unchanged. Where a control moves (T8), the change is noted.

| # | Threat | Control | Proof or test |
|---|---|---|---|
| **T11** | **Client secret leak** (store dump, log, trace) | Secrets are 256-bit and minted in `flow` under `Rand[mode=crypto]`. They are labelled `string<secret>`, returned **once** at creation, and stored only as `sha256Hex` digests. They are compared with `constantTimeEqual` on digests. A fast hash is enough, because the secret is high-entropy random, not a password (V6). A leaked secret alone mints nothing: the attacker also needs a code delivered to the client's **registered** redirect, plus the PKCE verifier when PKCE is required (D4). `rotateClientSecret` invalidates the old secret at once | `TestSecretNeverStored` (store values hold no secret substring); `TestWrongSecretInvalidClient` (401 + `WWW-Authenticate: Basic`); an IFC leak fixture that logs the client secret inside `flow.ail` and must fail to compile |
| **T12** | **DCR spam / storage exhaustion** | DCR is **off** unless the adopter enables it (D2). When it is on: a per-caller window limit (default 10/hour) and a global limit (default 500/day), counted with store keys that expire; a 16 KB body cap; at most 5 redirect URIs. A registration that is never used expires after 24 h. After its first successful code exchange, it slides to 90 days idle | `TestDcrRateLimitPerCaller`, `TestDcrGlobalCap`, `TestDcrUnusedExpires`, `TestDcrBodyCap`. The counters are not atomic, because the store hook has no compare-and-set. The limit is soft by up to the number of concurrent requests, and the doc says so |
| **T13** | **Theft window of a non-rotating refresh token** | The stable policy applies only to clients configured `stable`; **rotate stays the default for every client**. A stable token is bound to its `client_id`. A confidential client must authenticate on every refresh, so the stolen token alone is useless. Each stable token also has an idle expiry and an absolute lifetime (D3), and can be revoked at RFC 7009 `/oauth/revoke`, which Google calls on unlink (E8). Reuse detection is impossible by construction for a stable token, and the doc says so | `TestStableRefreshNoRotation` (no `refresh_token` in the refresh response; the old token still works); `TestStableRefreshNeedsClientAuth`; `TestStableRefreshAbsoluteExpiry`; `TestRevokeEndpointKillsFamily` |
| **T14** | **Client-ID namespace confusion** (a registered ID treated as a CIMD URL, or the reverse) | Two disjoint namespaces. CIMD IDs are `https://` URLs (`core.cimdUrlOk`). Registered IDs are `mco_` + 32 hex characters and never contain `://`. `core.clientKind` is pure, and its contract is Z3-checked: no string is both | Z3 `ensures` on `clientKind`; `TestRegisteredIdNeverFetched` |
| **T15** | **Phishing through DCR display metadata** (`client_name`, `logo_uri`) | DCR is open registration, so client-supplied names are not trusted. The package stores no `client_name`, never fetches `logo_uri` (which would also be an SSRF vector), and `describeRequest` adds `"registered": "dcr"` so the consent page can mark the client unverified. It names only the redirect host | `TestDescribeRequestOmitsClientName` |
| **T16** | **Confidential-client downgrade** (a confidential `client_id` presented with `none`, or a CIMD client presenting a secret) | `authenticateClient` takes the method from the **client record**, never from the request. If the methods disagree, the request gets `invalid_client` | `TestConfidentialWithoutSecret`, `TestCimdWithSecretRefused` |
| **T17** | **Identity forgery or bypass into a tool** (serve-api M2) | `_auth` is bound **only** from the gate's verified result, and an `_auth` export is served **only** on the gated listed surface, so no path reaches it with a missing or empty identity. Any `_auth` key in the arguments is overwritten. Any client-sent `X-Ailang-Mcp-Identity` header is deleted before the gate runs and is never visible in `_headers` | Go tests `TestAuthParamNotForgeableByArgs`, `TestIdentityHeaderStripped`, `TestAuthExportAbsentFromAgentSurface`, `TestEmptySubjectRejected` |
| T8 (amended) | Unbounded family records | A family record lists the access-token digests it issued. A stable token refreshes for months, so the list is pruned to digests whose access token has not yet expired. An expired token needs no revocation | `TestFamilyDigestsPruned` |

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1: Confidential-client model: operator-minted `mco_` IDs, a hashed secret shown once, `client_secret_basic` + `client_secret_post`, an exact per-client redirect allowlist; **no `private_key_jwt`** in 0.2 | Defines what GE (and later any enterprise IdP-style client) can use; secrets add a new kind of credential to defend | human | design | high |
| D2: DCR ships **off by default**, enabled per adopter with abuse limits | DCR adds an unauthenticated write endpoint; Claude registers per connection (0.1.0 D2 rationale) | human | design | high |
| D3: Refresh policy is **per client**: `rotate` (default, every existing client) or `stable` (opt-in), with stable lifetimes | Changes the security model for chosen clients; Claude and Google say opposite things (E7, E13) | human | design | high |
| D4: PKCE stays **required for every client**, confidential included, unless the GE measurement (P0-1) shows GE cannot send S256 | OAuth 2.1 posture vs a GE checkbox that is "Optional" (E2) | human, after P0-1 | before GE pilot | med |
| D5: Identity verifier shape and bound parameter: `Option[{subject, client_id, scopes}]` and `_auth` | A public annotation contract that adopters will code against | human | design | high |
| D6: Per-subject rate limit lives in the serve-api gate (one flag), not in each adopter | Executor prod blocker; one implementation vs many | human | design | med |
| D7: `serverInfo` source: one `--server-info` flag that also feeds the A2A card and OpenAPI version | Flag surface counts in the simplicity audit; replaces three stale constants | human | design | low |
| D8: `ailang://meta/modules` **removed from `/mcp/connect/`** with no opt-in, projected through `exposedView` on `/mcp/`, and turned off by `--no-introspection` | Deliberate behaviour change on a shipped surface | human | design | med |
| Which CIMD URL the framing probe presents (`https://ailang.sunholo.com/mcp-check/client-metadata.json`) | A public URL we commit to keep serving | human (part of D7's review) | design | low |
| Store layout of client records, counter keys, exact default numbers | Local to the package | agent | P1 | low |

### Design Freeze (Mark)

These must be resolved before sprint-executor starts the matching milestone. A recommendation
follows each item.

- [x] **D1 — confidential clients.** *(Ratified by Mark, 2026-11-05: as recommended.)* Recommended as stated. `private_key_jwt` is deferred: Claude
  and ChatGPT both accept `none` for CIMD (E12, E16), and GE uses a shared secret (E1, E2).
- [x] **D2 — DCR off by default.** Recommended. Docparse turns it on only for the Connected Apps
  pilot, and only with `dcrRedirectHosts` set (for example `oauth-redirect.googleusercontent.com`,
  `oauth-redirect-test.googleusercontent.com`, `claude.ai`). With an allowlist, an attacker cannot
  register their own redirect host.
- [x] **D3 — refresh policy.** Recommended: `rotate` remains the default. `stable` is allowed for
  **confidential clients only**, with a 30-day idle expiry and a 180-day absolute lifetime.
  `stable` for a **public** DCR client stays refused until P0-4 shows how Google registers.
  (Google's persistent linking asks for no rotation, E7; Claude asks for rotation of public
  clients, E13.) This is the "scoped design amendment" that
  [m-google-connected-apps-distribution](../m-google-connected-apps-distribution.md) D5/F3 requires
  before any refresh-policy change.
- [x] **D4 — PKCE for confidential clients.** Recommended: required. If P0-1 shows GE cannot send
  S256, this comes back to Mark with the evidence. It is not relaxed silently.
- [x] **D5 — identity contract.** Recommended: the verifier returns
  `Option[{ subject: string, client_id: string, scopes: [string] }]` (`None`, or an empty subject,
  = reject). The tool declares `_auth: <that record>`, which makes it identity-required: it is
  served only on the gated listed surface. Bool verifiers keep working.
- [x] **D6 — per-subject rate limit.** Recommended: `--mcp-subject-rpm N` (and `--mcp-subject-burst`)
  in serve-api, reusing `NewIPRateLimiter` keyed on the subject. A refusal is a 429 with
  `Retry-After` before the tool runs.
- [x] **D7 — `serverInfo`.** Recommended: one flag,
  `--server-info "name=parse,title=AILANG Parse,version=0.43.0"`. With no flag the name stays
  `ailang-api` and the version becomes the AILANG binary version, replacing the stale `0.8.1`.
  The same values fill the A2A card version and OpenAPI `info.version`. Also approve the probe CIMD
  URL for M3c.
- [x] **D8 — listed-surface resources.** Recommended as stated. UI resources (`ui://…`) stay on
  `/mcp/connect/`, because widgets need them.

## Solution Design

### M1 — `sunholo/mcp_oauth` 0.2.0 (lane: package)

**Compatibility rule.** `Hooks` is a closed record. Adding a field breaks every adopter's record
literal, and so would an adopter passing extra fields (V5). So **`Hooks` does not change**. New
behaviour comes through a **second record, `ClientConfig`**, passed only to the **new** entry points.
The 0.1.2 functions keep their signatures and their behaviour: each becomes a thin call into the new
code with `defaultClientConfig()`, which means CIMD only, no DCR, rotation, and metadata identical to
0.1.2.

```
export type ClientConfig = {
  confidential: bool,          -- accept registered mco_ clients (M1a)
  dcr: bool,                   -- serve RFC 7591 registration (M1b)
  dcrRedirectHosts: [string],  -- [] = any https host; else exact host or ".suffix" match
  dcrPerCallerPerHour: int,    -- default 10
  dcrGlobalPerDay: int,        -- default 500
  revocation: bool             -- advertise and serve RFC 7009 /oauth/revoke (M1c)
}
```

**Secrets stay in `flow`** (0.1.0 D4, T7). Client-secret minting, digesting, classification and
comparison live in `flow.ail`. A new module, `mcp_oauth/clients`, holds the non-secret logic: DCR
request validation, redirect-URI rules, rate-limit keys and client-record JSON. It never sees a
`<secret>`. `tests/lint.sh` gains a check that `clients.ail` never mentions `string<secret>`.

#### M1a — pre-registered confidential clients (GE custom MCP data store)

- **Creation is operator-only, not an HTTP endpoint.** `flow.createClient(h, cfg, redirectUris,
  method, refreshPolicy, nowSec) -> Result[(clientId, secret), err]` is called from an adopter's
  operator script (`ailang run` with the adopter's store). It prints the secret once, for the GE
  admin form. No new attack surface is mounted. `rotateClientSecret` and `deleteClient` sit
  alongside it.
- **Record** (stored under `client:<clientId>`): `{kind: "registered", method:
  "client_secret_basic"|"client_secret_post"|"none", secret_digest, redirect_uris, refresh_policy,
  pkce_required: true, created_at, last_used_at}`.
- **Authorize.** `flow.beginAuthorizeFor(h, cfg, loginUrl, clientId, …)` dispatches on
  `core.clientKind(clientId)` (T14). A URL goes to the CIMD fetch, exactly as today. An `mco_` ID
  goes to a store lookup. Redirect matching is the same Z3-proved exact match (`core.redirectOk`)
  against the record's list, with no wildcards. The GE URI is registered verbatim.
- **Token endpoint.** An internal (non-exported) `authenticateClient(h, cfg, authorizationHeader,
  formBody, nowSec) -> Result[ClientAuth, err]`:
  - It parses `Authorization: Basic` (base64 via `std/bytes.fromBase64`, V7, then form-unescape the
    ID and secret, per RFC 6749 §2.3.1) or `client_secret`/`client_id` in the form.
  - It looks the client up and enforces the **record's** method (T16).
  - It returns `ClientAuth {clientId, kind, method, refreshPolicy}`.
  - A failure gives `401 invalid_client` with `WWW-Authenticate: Basic realm="oauth"` (RFC 6749 §5.2).
- **No pre-authenticated value crosses the API (quorum round 1).** `ClientAuth` is internal to
  `flow` and is never a parameter of an exported function. An adopter could build such a record
  literal, so accepting one would let a caller skip authentication.
  - The new token functions take the raw request parts and authenticate inside:
    `flow.tokenRequest(h, cfg, authorizationHeader, formBody, nowSec)` handles both grants, and
    `flow.revokeRequest(h, cfg, authorizationHeader, formBody, nowSec)` handles revocation.
    `authenticateClient` is called first, on every call.
  - The 0.1.2 functions `exchangeCode`/`refreshTokens` keep their signatures. They **refuse any
    `client_id` that is not a CIMD URL**: `core.clientKind(clientId) != cimd` gives
    `invalid_client`. So a registered `mco_` client cannot enter through them.
  - The code and refresh records store the client's `kind`. A redemption whose authenticated kind
    differs is refused (T16).
  - Tests: `TestLegacyExchangeRefusesRegisteredId`, `TestLegacyRefreshRefusesRegisteredId`,
    `TestKindMismatchRefused`.
- **The adopter's token route** reads `req.headers` (already available under `@raw`). The route
  template gains the three extra lines.

#### M1b — DCR, RFC 7591 (Google Connected Apps; Claude's fallback)

- `flow.registerClientDcr(h, cfg, callerKey, body, nowSec) -> Result[(status, json), err]`. The
  adopter mounts `POST /oauth/register`.
- **The caller key comes from the rightmost `X-Forwarded-For` entry** (quorum round 1). The
  adopter passes it in, computed with the package's pure
  `clients.trustedCallerIp(xff, realIp)`.
  - That function mirrors serve-api's `clientIPFromHeader`, which the `submit_feedback` limiter
    already uses in prod (V32): Google's front end **appends** the real source address, and earlier
    entries are client-supplied.
  - Never the first entry, which the client controls.
  - `TestTrustedCallerIpIgnoresForgedPrefix`: `"1.2.3.4, 9.9.9.9"` keys on `9.9.9.9`, and changing
    the forged prefix does not change the key.
  - Behind a load balancer that appends its own hop, the adopter passes a trusted-hop count. The
    default is 0 extra hops.
- **Validation** (in `clients`, pure):
  - `redirect_uris` are required, 1–5 of them. Each is `https://` (no fragment, no userinfo, at most
    512 characters, host in `dcrRedirectHosts` when that list is set) or an RFC 8252 loopback URI.
  - `grant_types` ⊆ {`authorization_code`, `refresh_token`} and `response_types` ⊆ {`code`}.
  - `token_endpoint_auth_method` ∈ the advertised set. When it is absent, the server uses `none`
    and says so in the response, instead of RFC 7591's `client_secret_basic` default. A public
    client is the safe reading for an unauthenticated registration.
  - Errors are RFC 7591 §3.2.2 `invalid_redirect_uri` / `invalid_client_metadata`.
- **Response** `201`: `client_id` (`mco_…`), `client_id_issued_at`, plus `client_secret` and
  `client_secret_expires_at: 0` when confidential, and the echoed accepted metadata. Display fields
  are dropped (T15).
- **Abuse controls** (T12): the rate limits and caps above; a 24-hour TTL on unused registrations,
  which slides on first use; a body cap. No RFC 7592 management API.
- **Metadata:** `registration_endpoint` appears **only** when `cfg.dcr` is true.
  `client_id_metadata_document_supported: true` and `none` stay, so Claude keeps choosing CIMD (E12)
  and ChatGPT keeps preferring it (E18).

#### M1c — per-client refresh policy + RFC 7009 revocation

- `rotate` (default) is today's behaviour exactly: mark the token used, issue a new one in the family,
  and revoke the family on reuse.
- `stable`:
  - A refresh issues a new access token in the same family and does **not** mark the refresh token
    used.
  - The response **omits `refresh_token`**, which is RFC 6749 §6 non-rotation: the client keeps the
    one it has. The server cannot echo it, because it stores digests only.
  - The record carries `idle_expires_at` (sliding) and `absolute_expires_at`.
  - A confidential stable client must authenticate on every refresh (T13).
- **Revocation:** `flow.revokeRequest(h, cfg, authorizationHeader, formBody, nowSec)` (it
  authenticates internally, per M1a), mounted as `POST /oauth/revoke`.
  - A refresh token revokes its whole family. An access token is revoked through `hooks.revoke`.
  - The response is always `200`, per RFC 7009 §2.2, so it reveals nothing.
  - A confidential client must authenticate.
  - `revocation_endpoint` is advertised only when `cfg.revocation` is true.
- T8 amendment: family digests are pruned to unexpired access tokens.

#### M1d — token-endpoint auth-method intersection

- **Advertised set:** `["none"]` with default config (identical to 0.1.2). With `confidential` or a
  confidential DCR, it becomes `["none", "client_secret_basic", "client_secret_post"]`. **`none` is
  always first and always present** (E12). `private_key_jwt` is never advertised in 0.2 (D1).
- **CIMD documents:**
  - The singular `token_endpoint_auth_method` is **never binding**. It is informational only, as it
    already is in fact (V2).
  - The plural `token_endpoint_auth_methods_supported`, when present, must contain `none`. Otherwise
    `invalid_client` names the mismatch.
  - A CIMD client always authenticates with `none` at our token endpoint.
  - A regression fixture uses ChatGPT's live document verbatim (E17). It carries the singular
    `private_key_jwt` and the plural `["none","private_key_jwt"]`, and must be **accepted**.
- **DCR / registered clients:** the method is fixed at registration and enforced (T16).

#### M1e — metadata, docs, threat tests, publish

- `core.asMetadataFor(issuer, cfg)` is the new builder. `core.asMetadata(issuer)` = `asMetadataFor(issuer,
  defaultClientConfig())`, byte-identical to 0.1.2 (golden).
- AGENT.md gains three adopter recipes:
  - **"GE custom MCP"**: create the client, the admin form fields, the egress FQDNs (E3);
  - **"Connected Apps (DCR)"**;
  - **"Claude/ChatGPT only (no change)"**.
- CHANGELOG 0.2.0; `ailang pkg quality` clean; Z3 on `clientKind`, `dcrRedirectOk` and `authMethodOk`.

### M2 — serve-api identity verifier (lane: serve-api)

- **Registration.** `@mcp_token_verifier` accepts `(string) -> bool` (unchanged) **or**
  `(string) -> Option[R]`, where `R` resolves to a record with exactly `subject: string`,
  `client_id: string` and `scopes: [string]`. The record may be inline or a declared type, resolved
  with the same machinery as `ParamZeros`. Today `VerifierSigOK` accepts only `bool` (V9). Anything
  else stays a registration ERROR naming both accepted shapes.
- **Gate.** `serveapi/protocol` gains `IdentityVerifyFunc func(ctx, token) (*Identity, error)` and
  `NewIdentityGate`. `NewBearerGate(VerifyFunc, …)` wraps a bool func, so
  `serveapi/protocol/mcphttp` and every existing caller compile unchanged (V10). `AdmitCallIdentity`
  returns the identity; the failure modes (401, 503, timeout, slot cap) are identical.
- **Scope: serve-api only; the embedded `mcphttp` path is unchanged (quorum round 1).** `mcphttp`
  calls only `Gate.Admit(w, r)`, once per request. Its tools are Go `ToolDescriptor`s run by a Go
  `Invoker`, with no AILANG parameter binding (V31). So no `_auth` parameter can exist on that path,
  and nothing there reads `X-Ailang-Mcp-Identity`.
  - An embedding Go host that wants identity can build `NewIdentityGate` itself and read the
    identity from the request context. That is Future Work; no `mcphttp` code changes here.
  - Test: `TestMcphttpGateUnchanged`. The existing `serveapi/protocol/mcphttp` tests pass with no
    edit.
- **Transport from gate to tool.**
  - `gatedHandler` stamps the verified identity as JSON on `X-Ailang-Mcp-Identity`, after **deleting
    any client-sent copy** on every request.
  - The tool handler reads it from `req.GetExtra().Header` and deletes it from the `_headers`
    binding.
  - This is the `X-Ailang-Mcp-Public-Base` precedent (V14). The go-sdk's own `TokenInfo` context key
    is unexported (V14), so it cannot carry the identity.
- **Binding: identity-required, never a zero identity.** A declared `_auth` param (the name is
  unused today, V13) makes the export **identity-required**. `_auth` is bound **only** from a
  verified identity, so the tool runs **only** where one exists: a gated `/mcp/connect/` call. There
  is no empty or zero identity on any path (quorum round 0):
  - **Registration rule.** An export with `_auth` must be gated. That means `@mcp_auth("oauth2")`, or
    any tool under `--mcp-connect-only`. An `_auth` export marked `@mcp_auth("noauth")`, or declared
    with a bool verifier, is a registration **ERROR** naming the rule.
  - **Other surfaces do not serve it.** The export is absent from `/mcp/` `tools/list`, is not an
    A2A skill, has no REST endpoint and is not a stdio tool. A call that names it there gets the
    surface's ordinary unknown-tool or not-found answer. serve-api logs one startup line per
    omitted export, the way `@mcp_agent_only` omissions are logged.
  - **The gate always runs first.** A gated call that reaches the tool has passed verification, so
    `_auth.subject` is the verifier's non-empty subject. The verifier returning `Some` with an
    empty `subject` counts as a rejection (401), not as an identity.
  - It is never in `inputSchema`, and it is always overwritten if a client sends it (T17).
  - It cannot be `@optional`, `@mcp_secret` or `@mcp_file`.
  - Tests: `TestAuthExportAbsentFromAgentSurface`, `TestAuthExportNoRESTRoute`,
    `TestAuthWithNoauthIsError`, `TestEmptySubjectRejected`.
- **Per-subject limit (D6).** `--mcp-subject-rpm`/`--mcp-subject-burst` apply after a successful
  verify, keyed on `subject`. A refusal is a 429 with `Retry-After`, before the tool runs. With the
  flag unset, nothing changes.
- **Executor adoption** (this repo, `executor_oauth/oauth.ail`):
  - `verifyMcpToken` returns the identity it already computes (account, client; V27).
  - The service runs with `--mcp-subject-rpm`.
  - This clears the "per-user rate limit before prod" precondition (GE design P1b). The multivac
    Terraform precondition is lifted in multivac, out of this repo.
- **Scope checks** are the tool's job in this milestone (`_auth.scopes`). A declarative
  `@mcp_scopes` with RFC 6750 `insufficient_scope` is Future Work.

### M3 — serve-api listing hygiene (lanes: serve-api, mcpcheck)

**M3a — `serverInfo` (D7).**
- One `serverIdentity{name, title, version}` built at startup from `--server-info` replaces the three
  stale constants:
  - MCP `serverInfo` on both surfaces (`mcp.go:65-66`);
  - the A2A card `version` (`a2a.go:96`);
  - OpenAPI `info.version` (`openapi.go:325`).
- go-sdk v1.8.0's `Implementation` has `Title` (V16). The embedded `mcphttp` path already takes the
  name and version from config (V17), so it is untouched.

**M3b — per-surface resources (D8).**
- `registerResources` takes the surface:
  - **`/mcp/connect/`**: does not register `ailang://meta/modules`.
  - **`/mcp/`**: registers it unless `--no-introspection`, and its content goes through `exposedView`,
    the same projection `/api/_meta/modules` uses. So `@noexpose` exports never appear (V18).
- The resources capability stays advertised. An empty `resources/list` is valid, and `ui://` widgets
  still need the capability.
- `testdata/mcp_ui.golden.json` changes deliberately (V19).

**M3c — `mcp check` framing probe for CIMD-only servers.**
- **Client selection, read from the AS metadata the check already fetches:**
  1. **CIMD advertised** (`client_id_metadata_document_supported` + `none`): present the probe client
     `https://ailang.sunholo.com/mcp-check/client-metadata.json`. It is served from
     `docs/static/mcp-check/client-metadata.json`. The docs site serves static JSON as
     `application/json` from `/` (V22).
     - The document lists `redirect_uris: ["http://127.0.0.1/ailang-mcp-check/callback"]` and
       `token_endpoint_auth_methods_supported: ["none"]`.
     - RFC 8252 loopback matching is port-agnostic in `mcp_oauth` (V23), so the probe needs no
       listener.
     - The real-CIMD path, a 302 to a framed-protected login page, is what the review observed
       against docparse with Claude's CIMD (V21).
  2. **No CIMD, but `--probe-client-id` / `--probe-redirect-uri` given** (a pre-registered AS, such as
     a GE-style client): use them.
  3. **Otherwise: `Skip`**, with "no probe client: the AS offers no CIMD; pass --probe-client-id". It
     never WARNs, and it **never registers a client through DCR**, because that would write to the
     adopter's server.
- **PKCE and the rest of the request (quorum round 0).**
  - The probe keeps the code-flow parameters `framing.go` already sends (V30): `response_type=code`,
    an unreserved `state`, `code_challenge_method=S256`, and a `code_challenge` that is
    `base64url(sha256(verifier))` with no padding (43 characters).
  - M3c changes only the verifier source, to a fresh 43–128-character unreserved random value per
    run. No verifier ever reaches a token endpoint.
  - It adds `resource=<the MCP URL>` (RFC 8707) for servers that require it.
  - Only `client_id` and `redirect_uri` change, to the probe client's values. That passes
    `beginAuthorize`'s checks (V29: S256 method, 43-character challenge, unreserved state ≤512).
  - When the metadata lacks S256, the `authorization-server` check already FAILs, and the probe
    reports `Skip` ("AS does not advertise S256").
  - Two tests cover success metric 6:
    - the Go test `TestFramingProbeCimdPass` uses an httptest AS that enforces the V29 checks and
      fetches the probe document;
    - an end-to-end step in the package's `tests/e2e_flow.sh` runs `ailang mcp check --target both`
      against `tests/example/svc.ail` under real serve-api, and requires `authorization-framing PASS`.
- **Redirect handling.** A `CheckRedirect` stops before any hop to the probe's own `redirect_uri`,
  because a consent-free AS would otherwise bounce to `127.0.0.1` (V24). That outcome is reported
  as "AS redirected without a consent page" (WARN). Every other hop is followed as today.
- **Side effect, stated in the help text:** one pending authorization request on the target, which
  expires after the AS's request TTL (10 minutes in `mcp_oauth`).
- **Two cheap checks join `--target both`, WARN only:**
  - `serverInfo.name == "ailang-api"` (an unbranded listing);
  - `ailang://meta/modules` present on the listed surface.

### M4 — argument binding (dependency, not redesigned here)

#1631 (MCP `int` params fail; REST fails on whole-number floats) is **open**. Its fix, PR #2112
(`internal/apiserver/arg_bind.go`, one binder for MCP, REST, A2A and MCP UI, from
[m-serve-api-typed-arg-binder](../../implemented/v0_57_3/m-serve-api-typed-arg-binder.md)), is **OPEN,
`CONFLICTING`/`DIRTY`, checks skipped**, and was last updated 2026-11-04 (V25).
- It edits `mcp.go`, `routes.go`, `export_info.go`, `handler.go`, `routes_dispatch.go`, `a2a.go` and
  `mcp_ui_serve.go`, the same files M2 touches.
- **Sequencing:** #2112 is rebased and merged first, and M2 binds `_auth` inside the new binder,
  next to `_headers`.
- If #2112 has not merged when M2 starts, M2 binds `_auth` in `makeToolHandler` next to `_headers`,
  and #2112's rebase absorbs it. M2 then carries a parity test that the binder must keep passing.
- This doc changes nothing in the binder's design.

### Out of scope / later

- **A2A v0.3.** The estate has three incomplete A2A implementations:
  - serve-api `--a2a`: Go, pre-0.2, `tasks/send` only, ungated (`a2a.go`);
  - docparse's hand-written static card, which claims 0.3 but has no task endpoint
    (`api_server.ail:2145-2196`);
  - the Node resident agent (`docker/resident/lib/a2a.mjs`, 0.3, dev-only, a separate product).

  None of this blocks the **GE custom MCP data store** path, which is MCP plus a pre-registered OAuth
  client (E1). Connected Apps **prohibits** agent-to-agent (E20). A2A belongs to
  [M-AILANG-EXECUTOR-GE](../v0_55_0/m-ailang-executor-gemini-enterprise.md) P2, which needs v0.3
  (E21).
- **Google Connected Apps onboarding** (console, categories, domain verification, test accounts,
  review): [m-google-connected-apps-distribution](../m-google-connected-apps-distribution.md). This
  doc delivers only its auth prerequisites (its F2/F3).
- **GE org-policy egress allowlist** for the OAuth service and token host (E3): multivac Terraform,
  done in the adopter's rollout.
- `private_key_jwt`, JWT access tokens, OIDC/ID tokens, the `client_credentials` grant (Claude does
  not support it, E22), and RFC 7592 client management.

## Conflict Surface

serve-api touches `internal/apiserver`, `serveapi/protocol` and `cmd/ailang/serve_api.go`; mcpcheck
touches `internal/mcpcheck`; the package is a separate repo.

1. **Positions extended:**
   - the `@mcp_token_verifier` signature check (`routes.go:280`, `mcp_auth.go:35-66`);
   - the verifier call site (`mcp_auth.go` `verifyToken`, which asserts `bool`);
   - `protocol.BearerGate`;
   - the reserved-param set (`_headers` gains `_auth`);
   - the listed surface's resources;
   - the `initialize` result;
   - the framing probe's request;
   - in the package, the metadata builder, the token endpoint, and `beginAuthorize`'s client lookup.
2. **What already lives there:**
   - **Bool verifiers** in docparse (`verifyMcpToken` → `validateApiKey`), in the executor, and in
     `tests/example/svc.ail`. They keep working; the bool path is untouched.
   - **`protocol.VerifyFunc`/`NewBearerGate`**, used by `serveapi/protocol/mcphttp` (V10). This is
     an additive API: the old constructor wraps the new one, so it is extended, not overridden.
   - **`_headers` binding** and the `X-Ailang-Mcp-Public-Base` stripping (`mcp.go:445-452`). `_auth`
     follows the same pattern. Any existing param named `_auth` would become reserved; there is none
     in ailang or docparse (V13).
   - **`ailang://meta/modules`** is read by nothing in the repo except a golden file and docs (V19).
   - **CIMD clients** in docparse, the executor, and Claude/ChatGPT. Their IDs are URLs, disjoint
     from `mco_` (T14).
   - **The package's `Hooks` record**, which stays unchanged (V5).
3. **Disambiguation:**
   - The verifier's return type selects bool vs identity at registration.
   - `core.clientKind` selects CIMD vs registered.
   - `ClientConfig` is passed only to new functions, so 0.1.2 call sites get the old defaults.
   - The probe client is chosen from the metadata.
4. **Must still work (fixtures that exist):**
   - `ailang-packages/packages/mcp-oauth/tests/{flow_check,e2e_flow,cimd_check,ifc_leaks,lint}.sh`
     and `tests/example/svc.ail`;
   - `executor_oauth/test_oauth.sh` (21 checks);
   - `examples/runnable/serve_api_mcp_oauth.ail`;
   - `internal/apiserver/mcp_header_auth_test.go`;
   - `serveapi/protocol/bearer_gate_test.go` and `bearer_gate_call_test.go`;
   - `internal/mcpcheck/framing_test.go` (cases updated: dummy-client WARN becomes PASS/Skip by
     metadata);
   - live docparse `/.well-known/oauth-authorization-server`, unchanged byte for byte until docparse
     opts in.
5. **Deliberate changes:**
   - `ailang://meta/modules` leaves `/mcp/connect/` and is projected on `/mcp/` (D8);
   - the default `serverInfo.version` becomes the binary version;
   - the framing probe sends a real CIMD client;
   - a `_auth` param name becomes reserved, and an export that declares it is served only on the
     gated listed surface.

## Verification Log

### In-repo and live claims (checked 2026-11-05)

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | 0.1.2 metadata advertises only `none`, with no `registration_endpoint` and no `revocation_endpoint` | `core.ail:192-203` (`asMetadata`); live `curl https://docparse.ailang.sunholo.com/.well-known/oauth-authorization-server` → `"token_endpoint_auth_methods_supported":["none"]`, no registration key | Confirmed |
| V2 | CIMD validation reads only `client_id` and `redirect_uris`; the singular `token_endpoint_auth_method` is never read | `cimd.ail:27-38` (`validateClientDoc`: `getString(doc,"client_id")`, `getStringArrayOrEmpty(doc,"redirect_uris")` only) | Confirmed. So (d) is a regression guard, not a fix |
| V3 | The token endpoint performs no client authentication: `exchangeCode`/`refreshTokens` take a bare `clientId` | `flow.ail:232, 270`; `tests/example/svc.ail` token route passes `formGet(b,"client_id")` | Confirmed |
| V4 | Refresh always rotates; reuse revokes the family | `flow.ail:270-300` (marks `used: true` at l.293, reuse → `revokeFamily`) | Confirmed |
| V5 | Adding a field to `Hooks` breaks adopters, and so does passing extra fields | `ailang check` (v0.57.1): `f({a:1})` against `type H = {a,b}` → "record is missing required field(s): b"; `f({a:1,b:2,c:3})` → rejected with the open-record hint | Confirmed → `ClientConfig` is a separate param |
| V6 | `std/crypto` has no password-hash primitive (no bcrypt, scrypt, argon2 or pbkdf2) | `grep '^export' std/crypto.ail`: `sha256Hex`, `sha256Bytes`, `hmacSha256`, `constantTimeEqual`, `rsaVerifyPKCS1v15` | Confirmed (negative). Not needed: secrets are 256-bit random (T11) |
| V7 | `std/bytes` can decode Basic auth | `std/bytes.ail:35` `fromBase64(s) -> Option[bytes]` | Confirmed |
| V8 | 0.1.2 has no RFC 7009 revocation route; `revokeFamily` exists only as an internal function | `flow.ail` exports: `beginAuthorize`, `describeRequest`, `completeLogin`, `revokeFamily`, `exchangeCode`, `refreshTokens`, `oauthError` | Confirmed (negative) |
| V9 | The verifier must be `(string) -> bool` | `routes.go:280` (`VerifierSigOK` requires return `bool`); `mcp_auth.go` `verifyToken` errors on non-bool | Confirmed |
| V10 | `protocol.BearerGate` is shared with the embedded `mcphttp` handler | `serveapi/protocol/mcphttp/handler.go:48` (`Gate *protocol.BearerGate`), l.130 `Gate.Admit` | Confirmed → additive API |
| V11 | An identity-returning verifier and an `_auth` record param compile and run | `ailang check ident.ail` ✓ (`verify(token) -> Option[McpIdentity]`, `tool(code, _auth: McpIdentity)`); `ailang run` prints `u1:0` | Confirmed |
| V12 | The engine returns `Option` to Go as `{__type, __tag, fields}` | `internal/embed/convert.go:237-259` | Confirmed (decoding path for M2) |
| V13 | The name `_auth` is unused | `git grep -n "_auth\s*:" -- '*.ail'` → empty in ailang `origin/dev` and docparse `origin/main` | Confirmed (negative) |
| V14 | The go-sdk `TokenInfo` cannot be injected (unexported context key); serve-api already passes its own data on a stripped internal header | go-sdk v1.8.0 `auth/auth.go:80` (`type tokenInfoKey struct{}`); `mcp_ui_serve.go:16-27` `X-Ailang-Mcp-Public-Base`; `mcp.go:449` deletes it from `_headers` | Confirmed |
| V15 | `serverInfo` and two other versions are hardcoded and stale | `mcp.go:65-66` (`ailang-api`, `0.8.1`), `a2a.go:96` (`0.8.1`), `openapi.go:325` (`0.8.1.1`); live docparse `initialize` → `"serverInfo":{"name":"ailang-api","version":"0.8.1"}` | Confirmed |
| V16 | go-sdk `Implementation` carries `Title` | go-sdk v1.8.0 `mcp/protocol.go:2211-2223` | Confirmed |
| V17 | `mcphttp` already takes `serverInfo` from config | `serveapi/protocol/mcphttp/methods.go:62` | Confirmed |
| V18 | `ailang://meta/modules` is registered on both surfaces from the raw module map (not `exposedView`) and ignores `--no-introspection`. It leaks hidden exports | `mcp.go:92, 486-505`; `meta.go:144-160` (`exposedView` used only by `/api/_meta/*`). **Live local run** (v0.57.1): a module with `@noexpose secretHelper`, `@mcp_token_verifier verifyTok`, an `@mcp_agent_only` tool and a gated tool, served with `--mcp-http --oauth-issuer … --no-introspection`. `/mcp/connect/` `resources/read` listed all four names, `is_no_expose: true` twice, and the absolute module path. `/api/_meta/modules` → 404 | Confirmed |
| V19 | Nothing but docs and a golden file depends on that resource | `git grep -ln "ailang://meta/modules"` → `mcp.go`, `mcp_ui.go` (comment), `testdata/mcp_ui.golden.json`, `docs/docs/guides/serve-api.md`, two design docs | Confirmed |
| V20 | The framing probe uses a non-URL `client_id`, so CIMD servers 400 it | `framing.go:31` (`ailang-mcp-check-dummy`); internal review A.1 transcript `WARN authorization-framing … final HTTP 400` (docparse) and A.3 (executor) | Confirmed |
| V21 | A real CIMD `client_id` reaches the protected login page | Internal review A.2: Claude's CIMD `client_id` + S256 → 302 to `/oauth-login.html?handle=…` with `X-Frame-Options: DENY`, `frame-ancestors 'none'` | Confirmed (2026-11-05) |
| V22 | The docs site can host the probe CIMD document | `docs/docusaurus.config.js` `url: 'https://ailang.sunholo.com'`, `baseUrl: '/'`; `curl https://ailang.sunholo.com/benchmarks/latest.json` → `200 application/json; charset=utf-8` (served from `docs/static/`) | Confirmed |
| V23 | `mcp_oauth` loopback redirects match any port | `core.ail` `stripLoopbackPort` / `loopbackRedirectOk` | Confirmed |
| V24 | The check's HTTP client follows every redirect | `mcpcheck.go:78-79` (`&http.Client{Timeout: 30s}`, no `CheckRedirect`) | Confirmed |
| V25 | #1631 is open; PR #2112 is open and conflicting, and edits the M2 files | `gh issue view 1631` → OPEN; `gh pr view 2112` → `OPEN`, `CONFLICTING`, `DIRTY`, updated 2026-11-04T09:40Z; files incl. `mcp.go`, `routes.go`, `export_info.go`, `a2a.go`, `handler.go`, `routes_dispatch.go`, `mcp_ui_serve.go`, new `arg_bind.go` | Confirmed |
| V26 | `sunholo/mcp_files` 0.1.4 is in the registry | `ailang pkg search mcp_files` → `sunholo/mcp_files@0.1.4` | Confirmed (for the guide update) |
| V27 | The executor verifier already derives account and client | `executor_oauth/oauth.ail:244-265` (logs `account`, `client` per call) | Confirmed |
| V28 | Docparse serves without `--mcp-connect-only` and pins 0.1.x, so nothing changes until it opts in | docparse `Dockerfile` CMD (`--mcp-http --oauth-issuer`), `ailang.lock` | Confirmed |
| V29 | The authorize step restricts `state` (unreserved, ≤512) and requires a 43-character S256 challenge | `flow.ail:96, 109, 111` | Confirmed. The GE behaviour against it is **PENDING** (P0-1, P0-2) |
| V31 | The embedded `mcphttp` path admits once per request and binds no AILANG params | `serveapi/protocol/mcphttp/handler.go:117-130` (`admitGatedCalls` → `Gate.Admit(w, r)`; the only `Admit` call in the package); `serveapi/mcp_handler.go:20, 49-57` (Go `ToolDescriptor` list + Go `Invoker`) | Confirmed. M2 does not touch it |
| V32 | serve-api already keys per-IP limits on the **rightmost** `X-Forwarded-For` entry, because Cloud Run's front end appends the real source | `internal/apiserver/ratelimit.go:100-121` (`clientIPFromHeader`, `strings.LastIndex(xff, ",")`), used by `feedback_tool.go:72-76` in prod | Confirmed in code. The front-end ordering itself is Google's behaviour, relied on by that limiter since its release |
| V30 | The current framing probe already sends a valid S256 request; only its `client_id`/`redirect_uri` fail on a CIMD server | `framing.go:29-37`: `response_type=code`, `state=ailang-mcp-check-dummy-state`, `code_challenge = base64.RawURLEncoding(sha256(verifier))` (32 bytes → 43 characters), `code_challenge_method=S256` | Confirmed |

### Vendor claims (verbatim, from the external review, fetched 2026-11-05)

| # | Claim | Quote | URL |
|---|---|---|---|
| E1 | GE custom MCP uses a pre-registered client and a fixed redirect | "Register Gemini Enterprise as an OAuth client application with your identity provider" / "Set the authorization redirect URL to `https://vertexaisearch.cloud.google.com/oauth-redirect`." | https://docs.cloud.google.com/gemini/enterprise/docs/connectors/custom-mcp-server/set-up-custom-mcp-server (10-30) |
| E2 | GE: PKCE optional, Basic auth default | "Enable PKCE Support — Optional." / "Use HTTP Basic Authentication — Optional. Selected by default." | same |
| E3 | GE egress allowlist covers the auth and token hosts | "Explicitly allow the fully qualified domain names (FQDNs) for your custom MCP server URL, authorization URL, and token URL." | same |
| E4 | GE transport and TLS | "The older SSE transport isn't supported." / "Configure custom MCP servers with a TLS certificate signed by a publicly trusted Certificate Authority" | same |
| E5 | Connected Apps registration order | "Google checks for CIMD first and falls back to DCR using your .well-known discovery endpoints." | https://developers.google.com/connected-apps/gpc/perform-the-steps (10-28) |
| E6 | Connected Apps OAuth guide requires a registration endpoint | "Even if your server supports Client ID Metadata Documents, the Gemini app … currently requires a dedicated registration_endpoint to be present in the discovery JSON." | https://developers.google.com/connected-apps/mcp/oauth (09-14) |
| E7 | Google persistent linking: do not rotate | "to implement Persistent Linking, you must avoid rotating the Refresh Token during this exchange." | same |
| E8 | Google calls revoke on unlink | "When a user unlinks or disconnects your integration … we will make a POST request to the /revoke endpoint" | same |
| E9 | Connected Apps PKCE | "PKCE is mandatory for all OAuth 2.1 and optional for OAuth2.0." | same |
| E10 | Connected Apps redirect patterns | "`https://oauth-redirect.googleusercontent.com/r/*`" … "`https://oauth-redirect-test.googleusercontent.com/a/*`" | same; perform-the-steps |
| E11 | Connected Apps discovery | "Gemini MCP console uses the first URL that returns a response—including an error." | https://developers.google.com/connected-apps/mcp/oauth |
| E12 | Claude chooses CIMD only with `none`, else DCR | "Claude normally selects CIMD only when your authorization server metadata advertises both `"client_id_metadata_document_supported": true` and `"none"` in `token_endpoint_auth_methods_supported`, and otherwise falls back to DCR." | https://claude.com/docs/connectors/building/authentication.md |
| E13 | Claude: rotate for public clients | "Rotate refresh tokens for public-client connections." | same |
| E14 | Claude endpoint budgets | "Claude waits up to 10 seconds for a response from your OAuth discovery, registration, and token endpoints, and up to 30 seconds for refresh token requests." | same |
| E15 | Claude's confidential CIMD variant | "If your metadata lists `private_key_jwt`, Claude can instead present a second metadata document, whose URL ends in `/confidential_client`, and authenticate with a signed client assertion." | same |
| E16 | ChatGPT picks from the intersection | "For CIMD, ChatGPT supports `none` for public-client token exchange and `private_key_jwt` for signed client assertion token exchange." / "ChatGPT selects a method from the intersection of those sets." | https://developers.openai.com/plugins/build/auth.md |
| E17 | ChatGPT's live CIMD document | `{"client_id":"https://chatgpt.com/oauth/client.json", "redirect_uris":["https://chatgpt.com/connector_platform_oauth_redirect"], "token_endpoint_auth_method":"private_key_jwt", "token_endpoint_auth_methods_supported":["none","private_key_jwt"], … "jwks_uri":"https://chatgpt.com/oauth/jwks.json"}` | https://chatgpt.com/oauth/client.json |
| E18 | ChatGPT: CIMD preferred, DCR selectable | "ChatGPT prioritizes CIMD when it is available, but the plugin builder can choose DCR when both CIMD and DCR are available." | https://developers.openai.com/plugins/build/auth.md |
| E19 | OpenAI submission auth | "The portal reads advanced authentication settings from your package's `mcp.json`." / "Don't put the client secret in `mcp.json`; the manifest rejects `clientSecret`." | https://developers.openai.com/plugins/deploy/submission.md |
| E20 | Connected Apps forbids A2A | "All agent-to-agent interactions are strictly prohibited." | https://developers.google.com/connected-apps/connected-apps-policy (11-02) |
| E21 | GE A2A is v0.3 | "Gemini Enterprise supports the A2A v0.3 streaming mechanism. If you use A2A v1.0.0 or later, then use the compatibility packages provided by the SDK" | https://docs.cloud.google.com/gemini/enterprise/docs/register-and-manage-an-a2a-agent (10-30) |
| E22 | Claude: no M2M | "A machine-to-machine `client_credentials` grant isn't supported" | https://claude.com/docs/connectors/building/authentication.md |

E1–E22 are **documented** contracts, not measured behaviour. The rows below are what only a live client
can settle.

### P0 — PENDING external measurements (gate the *enablement*, not the build)

The package and serve-api work is built, tested and released against the documented contracts above.
**Enabling** a client model for a real platform waits on these measurements, each recorded as a
Verification Log row before that adopter turns the feature on. Each needs a person at a vendor
console, so the sprint plan lists them as attended tasks, not code milestones.

| # | Unmeasured premise | How to measure | If it fails |
|---|---|---|---|
| P0-1 | GE sends an S256 `code_challenge` when "Enable PKCE Support" is ticked | GE dev app → custom MCP data store against a dev deploy with a registered client → "Verify Auth"; read the authorize request in the service log | D4 returns to Mark with the log line 
|      | **MEASURED 2026-11-11: yes.** Dev executor (`ailang-dev-executor-mcp-oauth`), client `mco_7145…` minted by `createClientCli`, PKCE ticked: authorize 302 at 09:22:27Z (the package refuses an authorize without a 43-char S256 challenge, V29), code 09:22:51Z, token 200 from UA `Google` with **HTTP Basic** client auth 09:22:54Z, then `google-adk/2.14.0` tools/list. Redirect exactly `https://vertexaisearch.cloud.google.com/oauth-redirect`. D4 (PKCE required) holds | — |
| P0-2 | GE's `state` fits `stateOk` (unreserved, ≤512) | same log | Widen `stateOk` to RFC 6749 VSCHAR with percent-encoding on echo; a scoped package change |
|      | **MEASURED 2026-11-11: does NOT fit for the web app.** The console's Verify Auth sends a short state (302, passed), but the GE **web app's** per-user Connect sends a **768-character base64 state with `=` padding** (`invalid_request: state must be unreserved characters, at most 512`, 09:50Z). Fix as planned: `sunholo/mcp_oauth` 0.2.1 (ailang-packages #189) accepts VSCHAR up to 2048 and percent-encodes `state` on echo | — |
| P0-3 | GE survives refresh rotation, or needs `stable` | Two refresh cycles past the access-token TTL in GE | Configure that client `stable` (allowed, because it is confidential, D3) |
| P0-4 | How Connected Apps registers through DCR (public or confidential; `token_endpoint_auth_method`) and whether it accepts a rotated refresh token | Gemini Developer Mode against a dev Parse with DCR on | Decides whether D3's "no stable for public" holds |
| P0-5 | Whether a 404 on `/.well-known/openid-configuration` breaks Connected Apps discovery (E11) | Same session; read the discovery requests | Serve a minimal alias (a docparse change) |
| P0-6 | Claude's DCR fallback request shape, if docparse ever turns CIMD off | Not needed while CIMD stays on (E12); recorded only if a DCR-only adopter appears | — |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Randomness stays confined to `Rand[mode=crypto]` minting in `flow` |
| A2: Replayability | 0 | No trace or replay surface changes |
| A3: Effect Legibility | +1 | New network-facing behaviour is explicit config (`ClientConfig`, flags), and off by default; the identity a tool receives is a typed param, not a header it must parse |
| A4: Explicit Authority | +1 | Identity flows from the verifier to the tool as a declared, unforgeable param, and authority (client method, refresh policy) comes from the stored client record, never from the request (T16). The listed surface stops exposing the module inventory |
| A5: Bounded Verification | +1 | New pure predicates (`clientKind`, `dcrRedirectOk`, `authMethodOk`) carry Z3 contracts; DCR and refresh lifetimes are bounded |
| A6: Safe Concurrency | 0 | The soft, non-atomic DCR counters are stated, and the bound is the number of concurrent requests |
| A7: Machines First | +1 | Each threat maps to a test; `mcp check` stops emitting a false WARN that trains adopters to ignore it |
| A8: Minimal Syntax | 0 | No language syntax; one reserved param name (`_auth`) |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | One AS serves four client models through one hook record. Bool verifiers and 0.1.2 call sites keep working |
| A11: Structured Failure | +1 | RFC 6749 / 7591 / 7009 typed errors, a 429 with `Retry-After`, and registration ERRORs for invalid `_auth` use |
| A12: System Boundary | +1 | The OAuth boundary still lives in one audited module; the gate-to-tool identity crossing is a single stripped header |

**Net score: +7. Proceed.** Hard-violation check: no −1 on A1, A3, A4 or A7.

## Testing Strategy

- **Package:** native tests in `core_test.ail` for every pure predicate and the ChatGPT CIMD fixture.
  The flow tests are new shell suites in the style of `tests/flow_check.sh`: `tests/confidential_check.sh`,
  `tests/dcr_check.sh`, `tests/refresh_policy_check.sh` and `tests/revoke_check.sh`. The IFC leak
  fixtures cover the client secret. The 0.1.2 suites run unchanged.
- **Integration:** `tests/example/svc.ail` gains a second registered GE-shaped client and the
  register and revoke routes. `tests/e2e_flow.sh` drives it under real serve-api.
- **serve-api:** Go tests in `internal/apiserver` (verifier shapes, `_auth` binding on each surface,
  forgery, subject rate limit, `serverInfo`, resources per surface). The `serveapi/protocol` gate
  tests cover the identity path.
- **mcpcheck:** `framing_test.go` with httptest servers: CIMD-only, pre-registered with flags, and
  no-CIMD Skip. A consent-free AS hits the stop-at-callback path.
- **Mutation check:** for each new control, revert it in a scratch copy and confirm a named test
  fails (memory rule: first attempts catch about half).

## Deferred Decisions (agent may resolve)

- Store key layout and the exact default numbers for limits, TTLs and caps within the ranges above.
- Whether `createClient` also ships as a tiny `[bin]` command in the package or stays a documented
  `ailang run` recipe.
- The JSON encoding of `X-Ailang-Mcp-Identity`, as long as it is stripped on entry and absent from
  `_headers`.
- Whether `--server-info` also accepts `description` and `websiteUrl` (go-sdk has both fields).

## Non-Goals

- A2A in any form (§Out of scope); `private_key_jwt`; OIDC; JWT access tokens; `client_credentials`;
  RFC 7592 management; declarative scopes (`@mcp_scopes`).
- Changing the arg binder's design (M4 is a dependency only).
- Vendor console onboarding (the Connected Apps and GE runbooks own that).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| A GE or Google client behaves differently from its docs (P0-1..5) | High | Everything is off by default and gated on recorded measurements. Failure paths are named in §P0 |
| #2112 stalls and M2 conflicts with it | Med | M2 has a fallback binding site plus a parity test (§M4) |
| A stable refresh token is stolen | Med | Confidential-only by default (D3), lifetimes, revocation endpoint (T13) |
| `flow.ail` grows past the size target | Low | Non-secret logic goes in `clients.ail`; `flow.ail` stays under 800 lines (now 307) |
| The probe CIMD URL moves with a docs-site restructure | Low | A test in `internal/mcpcheck` pins the URL, and a docs CI check that the file is served |

## Implementation Plan (summary; the sprint plan holds the day-by-day)

| Milestone | Lane | Estimate |
|---|---|---|
| M1a confidential clients | package | 1.5d |
| M1b DCR + abuse controls | package | 1.5d |
| M1c refresh policy + revocation | package | 1d |
| M1d auth-method intersection | package | 0.5d |
| M1e metadata, docs, publish 0.2.0 | package | 1d |
| M2 identity verifier + `_auth` + subject limit + executor adoption | serve-api | 2d (after M4) |
| M3a `serverInfo`, M3b resources | serve-api | 1d |
| M3c framing probe + two WARN checks | mcpcheck | 1.5d |
| M4 #2112 rebase and merge | serve-api (dependency) | tracked in its own plan |
| P0-1..5 measurements | attended (Mark) | about 0.5d, at the consoles |

### Files to modify or create

- `ailang-packages/packages/mcp-oauth/flow.ail`: client secrets, `beginAuthorizeFor`, internal `authenticateClient`, `tokenRequest`, `revokeRequest`, `registerClientDcr`, `createClient`, the legacy-wrapper CIMD guard (~+250 LOC)
- `ailang-packages/packages/mcp-oauth/clients.ail`: new; DCR validation, redirect rules, rate-limit keys, client records (~200 LOC)
- `ailang-packages/packages/mcp-oauth/core.ail`: `clientKind`, `authMethodOk`, `asMetadataFor` (~+60 LOC)
- `ailang-packages/packages/mcp-oauth/cimd.ail`: plural auth-method check (~+10 LOC)
- `ailang-packages/packages/mcp-oauth/tests/`: four new suites, the example service, the lint rule
- `ailang-packages/packages/mcp-oauth/AGENT.md`: adopter recipes; plus `CHANGELOG.md` and `ailang.toml` (0.2.0)
- `serveapi/protocol/bearer_gate.go`: `IdentityVerifyFunc`, `NewIdentityGate`, `AdmitCallIdentity` (~+60 LOC)
- `internal/apiserver/mcp_auth.go`: identity verifier resolution and call, header stamp and strip (~+80 LOC)
- `internal/apiserver/routes.go`: `VerifierSigOK` accepts the identity shape (~+20 LOC)
- `internal/apiserver/mcp.go`: `_auth` binding (or in `arg_bind.go` after #2112), `serverIdentity`, surface-aware resources (~+60 LOC)
- `internal/apiserver/a2a.go`, `internal/apiserver/openapi.go`: version from `serverIdentity`
- `cmd/ailang/serve_api.go`: `--server-info`, `--mcp-subject-rpm`, `--mcp-subject-burst`
- `internal/mcpcheck/framing.go`, `internal/mcpcheck/mcpcheck.go`: probe client selection, `CheckRedirect`, two WARN checks
- `cmd/ailang/mcp_check.go`: `--probe-client-id`, `--probe-redirect-uri`
- `docs/static/mcp-check/client-metadata.json`: the probe CIMD document
- `executor_oauth/oauth.ail`: identity verifier
- `docs/docs/guides/serve-api.md`, `docs/docs/guides/mcp-connectors.md`: `_auth`, `--server-info`, resources, Google section
- `examples/runnable/serve_api_mcp_identity.ail`: new, with its manifest entry

## Related Documents

- [m-mcp-oauth-package](../../implemented/v0_52_0/m-mcp-oauth-package.md): the 0.1 design (T1–T10, D1–D6). This doc amends its D2 ("no DCR") and adds client models.
- [M-AILANG-EXECUTOR-GE](../v0_55_0/m-ailang-executor-gemini-enterprise.md): P1b prod blocker (per-user rate limit) → M2; GE OAuth → M1a.
- [m-google-connected-apps-distribution](../m-google-connected-apps-distribution.md): F2/F3 → M1b/M1c; onboarding stays there.
- [m-serveapi-directory-ready](../v0_51_0/m-serveapi-directory-ready.md): the listed surface this doc tidies (G13 `serverInfo`).
- [m-serve-api-typed-arg-binder](../../implemented/v0_57_3/m-serve-api-typed-arg-binder.md): M4 dependency (#1631, PR #2112).
- [Integrations review, internal](../integrations-review-internal-2026-11-05.md) · [external](../integrations-review-external-2026-11-05.md).

## Quorum record

Command: `ailang design-quorum <doc> --max-cost-usd 0.30 --author claude:claude-opus-5-5
--controller-verdict pass`. Seats: `gpt6-1-sol`, `gemini-4-argon`, `oc-glm-5-3`; the Anthropic seat
sat out (same vendor as the author).

- **Round 0** (2026-11-05T07:41:45Z, `.ailang/state/mission-quorum/m-mcp-oauth-client-models-2026-11-05T07-41-45Z.json`,
  $0.187): **BLOCKED**, 3 of 3 reject, controller pass.
  - **sol and gemini:** binding a zero `_auth` identity on ungated surfaces was a silent fallback,
    and it contradicted the doc's own bool-verifier rule. **Accepted.** `_auth` exports are now
    identity-required and served only on the gated listed surface. An empty subject is a rejection
    (M2 "Binding", T17).
  - **glm:** M3c did not say the probe sends S256 PKCE, so the probe could 400 as before.
    **Accepted.** The doc now states that the probe keeps the existing 43-character S256 challenge
    (V30), adds `resource`, and has two PASS tests.
- **Round 1** (2026-11-05T07:43:36Z, `…T07-43-36Z.json`, $0.199): **BLOCKED**, 3 of 3 reject,
  controller pass. None of the three repeated a round-0 objection. Three new objections, all
  accepted and fixed in this revision:
  - **sol:** the legacy wrappers built a public `ClientAuth` from the request `client_id`, a
    bypass for registered clients. **Fixed:** `ClientAuth` is never an exported parameter. The new
    token functions authenticate internally, and the legacy functions refuse non-CIMD IDs (M1a).
  - **gemini:** the doc did not say how the embedded `mcphttp` path relates to the identity gate.
    **Fixed:** it is unchanged and out of scope, because it has no AILANG param binding (V31, M2
    "Scope").
  - **glm:** the DCR caller key was the client-controlled first `X-Forwarded-For` hop. **Fixed:**
    it is the rightmost entry, mirroring serve-api's limiter (V32, M1b).
- **The re-quorum guardrail is spent.** The round-1 fixes have **not** been re-reviewed by the
  quorum, and Mark should know that when ratifying D1–D8. The controller verdict on the revised doc
  is pass: each fix closes the stated objection without widening scope.

## Future Work

- A declarative `@mcp_scopes` with RFC 6750 `insufficient_scope` and `securitySchemes` scopes.
- `private_key_jwt` (Claude's `/confidential_client` CIMD, E15; ChatGPT, E16) once a std signing/JWKS verification path exists.
- A Google ID-token verifier package (internal review §4.3), for the IAM-path executor to get user identity.

---

**Document created**: 2026-11-05
**Last updated**: 2026-11-05

# M-MCP-OAUTH-PACKAGE: `sunholo/mcp_oauth`, an OAuth 2.1 authorization server for MCP services, in AILANG

**Status**: Planned. **Awaiting human ratification of the Design Freeze.** Quorum round 0 BLOCKED on T7 (fixed with measured IFC semantics, V13–V16; no round-1 objection to it). Round 1 BLOCKED on T5 (hop-0 DNS to private IPs) and T2/T10 (revocation had no mechanism); both fixed below (S0 public-only Net mode, V17; `revoke` hook + token digests). The re-quorum guardrail is spent: no round 2.
**Target**: package `sunholo/mcp_oauth` 0.1.0 (monorepo `sunholo-data/ailang-packages`, `packages/mcp-oauth/`), plus one AILANG fix (S0, #1522) in v0.52.0
**Priority**: P1. It is the last blocker before AILANG Parse can be listed in Anthropic's directory.
**Estimated**: 5–6 days (S0 0.5d, P1 1.5d, P2 1.5d, P3 1d, P4 1–1.5d)
**Dependencies**:
- `m-serveapi-directory-ready` Phase A, **shipped in v0.51.0**: `@mcp_auth`, `/mcp/connect/`, the Bearer gate, `--oauth-issuer` and `ailang mcp check`. This doc is lane L4 of that one.
- Registry packages: `sunholo/auth` 0.4.1, `sunholo/firestore` 0.7.3 (adopters' storage).

**Quorum triggers fired**: #4 (vendor OAuth requirements and RFC behaviour are external premises) and #1 (design-freeze items). This is security-critical code. Run `ailang design-quorum` before planning.

---

## Problem Statement

The listed MCP surface (`/mcp/connect/`) now answers `401 + WWW-Authenticate` and names an
authorization server through `--oauth-issuer`, but **no such server exists**:
- `ailang mcp check` against the Parse example: checks 1–4 PASS, check 5 FAILs (no metadata at
  the issuer).
- Both directories require OAuth for account-backed tools. The quotes are A1, A2 and O4 in
  `v0_51_0/m-serveapi-directory-ready-sources.md`.

Every hosted AILANG service that wants a listing needs the same server. Building it once, as a
package, in AILANG, also exercises the language's security features (information-flow labels,
Z3 contracts, effect rows and budgets, `Rand[mode=crypto]`) on the code that matters most.

## Goals

**Primary goal:** a service adds `sunholo/mcp_oauth`, supplies three functions (authenticate a
user, mint an access token, store records), mounts four `@route`s, and passes
`ailang mcp check --target both` check 5. The security properties are proved where Z3 can prove
them and tested where it cannot.

**Success metrics:**
- `ailang pkg quality packages/mcp-oauth` reports Z3 `verified == total` for every pure security
  predicate: PKCE, redirect matching, CIMD URL validation, code TTL, and verifier length and
  charset. Remaining contracts are counted as runtime assertions (PUB016) and listed.
- A Claude custom connector, pointed at a dev Parse deploy, completes sign-in and calls a gated
  tool. This is the end-to-end proof, done in the Parse adoption sprint.
- Every OAuth threat listed below maps to a named test or a Z3-verified contract.

## Threat model and controls

Each control names the AILANG feature that enforces it. Evidence: the `ailang check` and
`ailang verify` runs logged in the Verification Log, with the verified code quoted in Appendix A.

| # | Threat | Control | AILANG feature | Proof or test |
|---|---|---|---|---|
| T1 | Code interception (another app gets the code) | PKCE **S256 only**; `plain` refused; verifier length 43..128, charset `[A-Za-z0-9-._~]` | pure `s256`, `constantTimeEqual`; `requires`/`ensures` | Z3 for the length/charset predicates; RFC 7636 App. B vector as a native test |
| T2 | Code replay | Codes are single-use, **60 s** TTL, and stored only as `sha256Hex(code)`. On first redemption the code record is marked `redeemed` and stores **`tokenDigest = sha256Hex(access token)`** (and the refresh family id). A second redemption → `invalid_grant` **and `hooks.revoke(tokenDigest)`** plus revocation of the refresh family (OAuth 2.1 §4.1.3 BCP) | IFC label `<authcode>`/`<token>` + `Declassify` only in the digest function; pure TTL with `ensures` | Z3 for the TTL bound; test `TestCodeReplayRevokes`: double redemption → `invalid_grant`, and the test `revoke` hook received exactly the first redemption's token digest |
| T3 | Guessable codes, tokens or handles | 256-bit values drawn under `! {Rand[mode=crypto]}` (64 hex chars, the docparse `apiKeyGenerate` pattern) | effect mode on the minting function; the checker forces it up the call chain | test asserts length and alphabet; grep gate on `Rand[mode=crypto]`, as in docparse #G5 |
| T4 | Open redirect / code sent to an attacker | Exact `redirect_uri` match against the client's registered list. The only exception is RFC 8252 loopback: `http://127.0.0.1` or `localhost`, **any port**, exact path, needed by Claude Code. No prefix or substring matching | pure `redirectAllowed` with `ensures result == contains(registered, presented)` (loopback rule separate) | Z3. It **caught the prefix-match bug** with a counterexample (V3) |
| T5 | CIMD SSRF (`client_id` is a URL we fetch) | Two layers. **(a) URL string:** pure validation (`https://` only; no IP literal, `localhost`, `*.internal`, `metadata*`, userinfo or fragment; Z3-verified), one fetch per authorize (`Net @limit=1`), a byte cap, and `client_id` equal to the fetched URL. **(b) Destination at connect time (S0):** the CIMD fetch runs in a **public-only** Net mode. That reuses the existing pinned-resolution check (`resolvePinned` validates **every** resolved IP and dials that pinned address, V17), with loopback and metadata **forced off for this call** whatever `serve-api`'s process setting is, on hop 0 **and** every redirect hop. A hostname that resolves to 127.0.0.1, 169.254.169.254 or RFC 1918 is refused at dial time; DNS rebinding is closed by the pinned dial | `@limit`; Z3; S0 | Z3 for (a); S0 tests with an injected `lookupIP` (V17): `public.example → 127.0.0.1` refused, `→ 169.254.169.254` refused, a redirect to a link-local address refused, a direct metadata call **without** public-only still allowed (gcp_auth regression) |
| T6 | Timing attacks | Every secret comparison (code digest, PKCE, refresh-token digest) uses `constantTimeEqual`, never `==` | `std/crypto.constantTimeEqual` (Go `crypto/subtle`) | grep gate: no `==` on a `<authcode>`/`<token>` value (the IFC sink signature forbids it inside the module) |
| T7 | Tokens or codes in logs and traces | See **§T7 in detail** below: label on receipt, log sinks refuse the label, one module, no declared record types holding secrets, and leak tests against the real module. Residual risk is stated there | IFC labels | negative fixtures injected into a copy of the **real** `flow.ail` (as docparse's `test_ifc_labels.sh` does) must be rejected |
| T8 | Unbounded storage (code, request or client-cache flooding) | Every stored record carries an expiry. `authorize` is budgeted (`@limit`); the store hook must honour the TTL; one sweep function for expired records | `@limit`, `ensures` on the TTL | test: an expired record is refused even if not swept |
| T9 | CSRF / login-request swapping | `authorize` stores a server-side **request handle** (256-bit) holding client, redirect, challenge and state. The login page posts back the handle plus the user's ID token. The code is bound to (client, redirect, challenge, account) | as T3 + T4 | test: a code redeemed with a different `redirect_uri` or `client_id` → `invalid_grant` |
| T10 | Refresh-token theft | Refresh tokens rotate on every use and belong to a **family** record listing the digests of every access token issued in it. Reuse of a rotated refresh token → `invalid_grant`, **`hooks.revoke` for every access-token digest in the family**, and the family is closed. Refresh tokens are stored as digests | as T2 | test `TestRefreshReuseRevokesFamily`: the `revoke` hook receives all of the family's digests; a later refresh with the newest token also fails |

### T7 in detail: what the labels guarantee, what they don't, and how the residual risk is handled

All of the following was measured with `ailang check`, v0.51.0-29, 2026-10-02 (V13–V16):

1. **Label on receipt.** A secret that enters `flow` unlabelled, such as the access token returned
   by the service's `mint` hook, is passed **immediately** through a pure classifier,
   `asToken(raw: string) -> string<token> = raw`. Raising a label needs no `Declassify`. Codes
   minted inside `flow` get `<authcode>` from their minting function's return type. A `let`
   annotation alone does **not** attach a label (V13: `let t: string<token> = hook()` then logging
   `t` is *accepted*), so the classifier is mandatory.
2. **Log and trace sinks refuse labels.** `flow`'s only logging function takes
   `{not token}`/`{not authcode}` (one `not` per parameter, so all secrets share a single
   `<secret>`-style label at the log sink, per the IFC guide). Rejected in V14: logging a classified
   token, directly or interpolated.
3. **Labels propagate through function returns and inline records** (V15). The token response body,
   built from a `string<token>` by a function returning plain `string`, is **still labelled** and
   cannot be logged. **Emitting it as the HTTP response needs no `Declassify`**, because the response
   is not a `{not …}` sink. `Declassify` is used only for the digest function, which deliberately
   lowers `<authcode>` to unlabelled for storage (T2).
4. **Two measured holes are designed around:**
   - **Declared record types** with a labelled field lose the label at field access (V16, filed as
     **#1523**). Rule: `flow` declares **no record type holding a labelled value**, enforced by a CI
     lint (`grep` for `type … { … : string<` in `flow.ail` → fail). Inline records are fine.
   - **Module edges** drop labels (V5). Everything labelled lives in `flow`. The string `flow`
     returns to the route module is unlabelled there.
5. **Residual risk (outside the labels):**
   - the service's own hook code (`mint`, `store`);
   - the thin `@route` handlers in the service module, which receive the response body unlabelled.

   Mitigations:
   - the route template is three lines and logs nothing (P3 ships it, plus a lint that the template
     contains no `println`/`Debug`);
   - the adopter checklist in AGENT.md;
   - deploy with `AILANG_TRACE_VALUES=off`, because traces render arguments verbatim
     (`m-trace-label-aware` is planned, not shipped).

   **The labels are a guarantee for `flow` and a convention elsewhere. The doc does not claim
   more.**
6. **The negative tests target the real code, not toys.** CI copies `flow.ail`, appends a leak (log
   the classified token; log the code; log the token response body), and asserts `ailang check`
   rejects each copy. A refactor that silently drops a label (for example, by introducing a declared
   record) makes one of those copies compile, which turns CI red.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1: **Opaque tokens minted by the service** (Parse: scoped `dp_` keys). The package defines no token format and **no JWT signing** | Avoids the missing JWT-signing primitive (V7) and key management. The resource server already verifies through `@mcp_token_verifier` | human | design | high |
| D2: **CIMD only for client registration in 0.1.0** (plus pre-registered clients for `oauth_anthropic_creds`). **No DCR** | Both vendors support CIMD (A6, O6). DCR floods client storage (Claude registers per connection) and adds a write endpoint to defend | human | design | med |
| D3: **Service hooks as a record of functions** (`authenticate`, `mint`, `store`), not a framework | Verified to compile with effect subsumption (V6). The package never sees the service's secrets or database | agent | design | med |
| D4: **One security module** (`mcp_oauth/flow`) holds every labelled value end to end; secrets labelled on receipt; no declared record types holding them | IFC labels do not survive module edges (V5) or declared-record field access (V16, #1523). Splitting the flow would silently weaken T7 | human | design | high |
| D5: **S0 (#1522) ships in AILANG before the package is deployed**: (i) redirect hops never reach loopback, link-local or private addresses; (ii) a **per-call public-only Net mode** (connect-time check after DNS, both flags forced off) that user-URL fetches declare | Without (ii), a hostname resolving to loopback or metadata passes every URL-string check (quorum round 1). Without (i), redirects do | human | design | med |

### Design Freeze
- [ ] D1: opaque, service-minted access tokens; no JWT in 0.1.0.
- [ ] D2: CIMD-only (+ pre-registered); no DCR.
- [ ] D4: a single security module; labelled values never in records.
- [ ] D5: S0 first, with both parts (redirect hops + per-call public-only mode).
- [ ] D6: the S0 per-call surface. **Recommended: a `Net[scope=public]` effect mode**, the
  `Rand[mode=crypto]` pattern, so the checker forces it into the caller's signature and it is
  auditable by grep. The alternative is a `std/net` request option (lighter, but invisible in
  types).

## Quorum record

- **Round 0** (2026-10-02, `…/m-mcp-oauth-package-2026-10-02T18-13-13Z.json`): BLOCKED, 3 of 3.
  All three objected to T7: the IFC labels were claimed for the access token, but the token
  arrives unlabelled from a cross-module hook and leaves through a record or module edge.
  Accepted. Fixed by measuring the real semantics (V13–V16, Appendix B):
  - label on receipt with a pure classifier;
  - labels propagate through returns and inline records;
  - the declared-record hole filed as #1523 and linted;
  - leak tests injected into the real `flow.ail`;
  - residual risk stated.
- **Round 1** (`…T18-18-15Z.json`): BLOCKED, 3 of 3, **with no objection to T7**. New objections:
  - **kimi:** T5 was open at hop 0 for a hostname resolving to a private IP;
  - **gemini and glm:** T2/T10 revocation had no hook and no token↔code link.
  Both accepted and fixed: S0 part (ii) with the V17 connect-time check, and the `revoke` hook plus
  token digests on the code and family records.
- The guardrail (re-quorum once) is spent: **Mark ratifies D1–D6**.

## Solution Design

### Package layout (`packages/mcp-oauth/`)

| Module | Kind | Contents |
|---|---|---|
| `mcp_oauth/core` | **pure, Z3-verified** | `s256`, `pkceOk`, `verifierOk`, `redirectAllowed`, `loopbackRedirectOk`, `cimdUrlOk`, `codeExpiresAt`, `isExpired`, metadata builders (RFC 8414 AS metadata and the CIMD expectations) |
| `mcp_oauth/flow` | effectful, **the only module that touches `<authcode>`/`<token>`** | `beginAuthorize` (validate the request, store the request handle, return the login redirect), `completeLogin` (handle + ID token → `hooks.authenticate` → mint code → redirect), `exchangeCode` (token endpoint: PKCE, single use, `hooks.mint`), `refresh` |
| `mcp_oauth/cimd` | effectful, `Net @limit=1` | `fetchClient(url)`: validate with `core.cimdUrlOk`, fetch once, enforce the byte cap, parse, check the `client_id` equality and the `redirect_uris` list |
| `mcp_oauth/routes` | thin `@route` templates | example route wiring that adopters copy into their service. `serve-api` serves only the service's own modules' routes, so the package ships the handlers and the service declares the `@route`s |

**Effect ceiling** (`[effects] max`): `Net, Clock, Rand, IO, FS, Env, Declassify`.
`IO`/`FS`/`Env` come from the hook functions' rows and are the declared maximum. `core` is
`max = []`.

### Hooks (D3)

```
type Hooks = {
  authenticate: (string) -> Result[string, string] ! {Net, Env, Clock},     -- ID token -> account id
  mint:         (string, string) -> Result[string, string] ! {Net, Env, FS, Clock, Rand}, -- account, client -> access token
  store:        Store,                                                     -- put/get/delete with expiry
  revoke:       (string) -> Result[unit, string] ! {Net, Env, FS, Clock}   -- invalidate the access token whose sha256Hex digest this is
}
```

Parse supplies the following (in the adoption sprint, not here):
- `authenticate` = `sunholo/auth` `verifyFirebaseJWTFull` → uid;
- `mint` = a scoped `dp_` key, `Rand[mode=crypto]` (so the gate's verifier accepts it unchanged);
- `store` = `sunholo/firestore` with the documents' TTL;
- `revoke` = disable the `dp_` key whose hash matches the digest. Parse already stores keys by
  hash (`sunholo/auth.hashKey`; **PENDING V18**: confirm in the adoption sprint that the stored
  hash equals `sha256Hex(key)`; if not, the revoke hook maps via the key id).

### Endpoints (adopter mounts; paths fixed for discovery)

| Endpoint | Method | Purpose |
|---|---|---|
| `/.well-known/oauth-authorization-server` | GET | RFC 8414 metadata: `S256` only, `client_id_metadata_document_supported: true`, `token_endpoint_auth_methods_supported: ["none"]`, no `registration_endpoint` |
| `/oauth/authorize` | GET | validate (`response_type=code`, PKCE S256, `client_id` CIMD fetch, exact redirect); store the request handle; redirect to the service's login page with the handle |
| `/oauth/login/complete` | POST | from the service's login page: handle + ID token → code → redirect to `redirect_uri?code=…&state=…` |
| `/oauth/token` | POST (form) | `authorization_code` (+ PKCE) and `refresh_token` grants. Errors per RFC 6749 §5.2 |

### Effect budgets
- `beginAuthorize`: `Net @limit=1` (the CIMD fetch) and `Rand @limit=1` (the handle).
- `exchangeCode`: `Net @limit=2` (the store round-trips go through the hook's own budget).

## Conflict Surface

The package is a new module tree, and S0 changes `internal/effects`.

1. **Positions extended by S0:**
   - the Net redirect policy (hop validation in `internal/effects/net*.go`);
   - a per-call destination mode for Net. The proposed surface mirrors
     `Rand[mode=crypto]`: a `Net[scope=public]` effect mode, forcing `allowLocalhost` and
     `allowMetadata` off in `destinationPolicy` for that call. The **exact surface is a design-freeze
     item for S0**: effect mode vs a `std/net` request option.
2. **Constructs already there:**
   - direct requests to metadata or localhost allowed by `AllowMetadata`/`AllowLocalhost`
     (`sunholo/gcp_auth` depends on these);
   - cross-origin header stripping on redirects (`defaultSensitiveHeaders`).
3. **Disambiguation:** S0 applies only to **redirect hops**, never to the initial request, so
   `gcp_auth`'s direct metadata calls are unchanged.
4. **Programs that must still work:**
   - `sunholo/gcp_auth` token fetch (a direct metadata request);
   - `internal/effects/net_authorize_test.go`;
   - `internal/effects/net_test.go`;
   - docparse `fetchSourceUrl` (https, cross-origin redirect to https).
5. **Deliberate change:** a redirect to loopback, link-local or private addresses now fails with
   a typed Net error, even under `serve-api`'s permissive settings.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | PKCE S256 is computable in pure AILANG and matches RFC 7636 App. B | `pkce.ail` (Appendix A): `s256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk") == "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"` as a native test; `ailang test` passes | Confirmed (installed v0.51.0-29) |
| V2 | `constantTimeEqual` is Go `crypto/subtle` | `internal/builtins/crypto.go` (`_crypto_constanttimeequal`) | Confirmed |
| V3 | Z3 proves exact redirect matching and **refutes** the prefix/"non-empty list" variant | `redirects2.ail` (Appendix A): `ailang verify` proves `redirectAllowed` and gives a counterexample for `redirectAllowedBroken` | Confirmed |
| V4 | `Rand[mode=crypto]` reaches inner `rand_int`; `uuid4` is always crypto (122 bits) | the seeded-mode runtime probe (2026-10-01; deterministic with `AILANG_SEED`, `RAND_SEEDED_NO_SEED` without); `internal/builtins/rand.go` | Confirmed |
| V5 | **IFC limit:** labels are lost across module imports; `println` is not a `{not …}` sink | `ifc_xmod.ail` compiles clean (the hole); `ifc_leak.ail`, `ifc_json.ail` and `ifc_sneaky.ail` are rejected | Confirmed; D4 designs around it |
| V13 | A `let` type annotation does **not** attach a label to an unlabelled value | `leak.ail`: `let tok: string<token> = mintHook(acct); logLine("minted ${tok}")` → **accepted** | Confirmed (negative); hence the classifier |
| V14 | A pure classifier `asToken(raw: string) -> string<token> = raw` attaches the label; logging it is rejected | `cls_ok.ail` ✓ (classify + emit body + log a non-secret); `cls_leak.ail` → "information-flow violation: value labelled <token> reaches parameter "msg"" | Confirmed |
| V15 | Labels propagate through a plain-`string` return and through **inline** records | `cls_prop.ail` (log of `tokenBody(asToken(x))`) and `cls_rec.ail` (log of `{access: asToken(x)}.access`) → both **rejected** | Confirmed |
| V16 | A **declared** record type's labelled field loses the label at field access | `codes_same.ail` (`type RawCode = { raw: string<authcode> }`, `logLine(c.raw)`) → **accepted** | Confirmed hole; filed **#1523**; CI lint (D4) |
| V6 | Effectful functions in a record field, with effect subsumption, compile and run | `hof.ail`, `sub.ail` (2026-10-02): `ailang check` ✓ and run ✓ | Confirmed |
| V7 | `std/jwt` verifies only (RS256); there is no signing primitive; `hmacSha256` returns hex with a string key | `std/jwt.ail`, `std/crypto.ail` exports | Confirmed (negative); D1 avoids it |
| V8 | `@limit` budgets are enforced per call frame (the second call → `budget exhausted`) | `budget.ail` run | Confirmed |
| V9 | `serve-api` with `Net` allows localhost and metadata for every call, redirect hops included | `cmd/ailang/serve_api.go:134-138`; docparse probe reasoning in #1522 | Confirmed; S0 |
| V10 | `property` tests fail at runtime (internal parse error) | `props*.ail` | Confirmed (negative). **No property tests in this package; native tests + Z3 instead** |
| V11 | Claude and ChatGPT accept CIMD with `none` token auth and these redirect URIs | sources A6, O6, O7 (`m-serveapi-directory-ready-sources.md`) | Confirmed from vendor docs (external, re-fetch at release) |
| V17 | Net resolves once, validates **every** resolved IP and dials the pinned address; `lookupIP` is injectable for tests; private ranges are refused unconditionally; loopback and link-local depend on `allowLocalhost`/`allowMetadata` | `internal/effects/net_authorize.go`: `resolvePinned` (l.232), `validateIP` (l.195–228, `IsPrivate` → "no override available"), `dial` (l.262), `lookupIP` field | Confirmed (read 2026-10-02) |
| V18 | Parse stores API keys as `sha256Hex(key)`, so `revoke(digest)` maps directly | docparse `api_keys.ail` + `sunholo/auth.hashKey` (SHA-256 hex) | **PENDING**: confirm in the Parse adoption sprint (affects the adopter hook only, not the package design) |
| V12 | RFC 8252 loopback redirects: any port, exact path | RFC 8252 §7.3; Claude's auth doc ("port-agnostic match for localhost as well as 127.0.0.1") | Confirmed (external) |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Randomness is confined to `Rand[mode=crypto]` minting, explicit in the rows |
| A2: Replayability | 0 | — |
| A3: Effect Legibility | +1 | Every network fetch and random draw is declared and budgeted; `core` is pure |
| A4: Explicit Authority | +1 | Tokens come only from the service's `mint`; the package holds no credentials; S0 removes ambient metadata/localhost reach on redirects |
| A5: Bounded Verification | +1 | Security predicates are Z3-proved; budgets bound every effect |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | The threat → control → proof table is checkable; `pkg quality` reports proofs |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | Any service plugs in through three hooks; reuses `sunholo/auth` and `firestore` |
| A11: Structured Failure | +1 | RFC 6749 typed errors; S0 adds a typed Net error |
| A12: System Boundary | +1 | The OAuth boundary lives in one audited module |

**Net score: +7. Proceed.**

## Non-Goals
- **DCR** (D2): add it later only if a client requires it.
- **JWT access tokens and asymmetric signing** (D1, V7).
- **OpenID Connect** (ID tokens, userinfo): MCP needs authorization, not identity.
- **Consent screens.** The service's login page is the consent point; first-party only.
- **Fixing the IFC holes** (`m-ifc-cross-module-labels`, `m-trace-label-aware`): designed around (D4), not fixed here.

## Deferred Decisions
- The exact store interface (single-record vs batch): **agent** decides in P2.
- CIMD cache TTL (proposed 1 h): **agent**.
- Whether to add `randomBytes`/`sha256Raw` to `std` (cleaner than hex decoding, V7): **agent** may file it; not required.

## Phasing

| Phase | Scope | Where |
|---|---|---|
| **S0** | #1522: redirect hops never reach loopback, link-local or private addresses (keep direct requests unchanged) | ailang `internal/effects` |
| **P1** | `core`: every pure predicate with Z3-verified contracts and native tests (the RFC 7636 vector, the redirect cases, the URL cases) | packages |
| **P2** | `flow` + `cimd`: the authorize, login-complete and token flows with labels, budgets, single use and rotation; a negative IFC fixture in CI | packages |
| **P3** | AS metadata + route templates + `_smoke.ail` + AGENT.md + CHANGELOG; `ailang pkg quality` clean, with numbers reported | packages |
| **P4** | Integration: an example service (in-repo test fixture with an in-memory store hook) under `serve-api`, plus the Phase A listed surface; `ailang mcp check --target both` passes all five checks | packages + ailang example |

The Parse adoption (Firebase `authenticate`, `dp_` `mint`, Firestore `store`) is the **next** sprint, in `docparse`.

## Appendix A: verified building blocks (2026-10-02, installed v0.51.0-29)

```ailang
-- RFC 7636 S256: BASE64URL(SHA256(ASCII(code_verifier)))   (pkce.ail; native test passes)
export pure func s256(verifier: string) -> string
  tests [("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")]
{ hexToB64url(sha256Hex(verifier)) }

export pure func pkceOk(verifier: string, storedChallenge: string) -> bool =
  constantTimeEqual(s256(verifier), storedChallenge)

-- Exact redirect match, Z3-proved (redirects2.ail). The broken "non-empty list" variant is refuted.
export pure func redirectAllowed(presented: string, registered: [string]) -> bool ! {}
ensures { result == contains(registered, presented) }
{ contains(registered, presented) }
```

`hexToB64url` decodes `sha256Hex` output with `fromInts` and then calls `toBase64URL` (full
source in the P1 commit).

## Appendix B: IFC fixtures behind V13–V16 (`ailang check`, v0.51.0-29, 2026-10-02)

**V14, accepted (the intended pattern):**
```ailang
module cls_ok
import std/io (println)
import std/json (encode, jo, kv, js)

func mintHook(acct: string) -> string ! {IO} = "dp_secret_for_${acct}"

-- Raise the label at receipt (unlabelled -> labelled is allowed).
pure func asToken(raw: string) -> string<token> = raw

-- The one sanctioned sink.
func tokenBody(t: string<token>) -> string ! {Declassify} =
  encode(jo([kv("access_token", js(t)), kv("token_type", js("Bearer"))]))

func logLine(msg: string{not token}) -> unit ! {IO} = println(msg)

export func exchange(acct: string) -> string ! {IO, Declassify} {
  let tok = asToken(mintHook(acct));
  logLine("minted for ${acct}");
  tokenBody(tok)
}
```

**V14, rejected** ("information-flow violation: value labelled <token> reaches parameter "msg""):
```ailang
module cls_leak
import std/io (println)

func mintHook(acct: string) -> string ! {IO} = "dp_secret_for_${acct}"
pure func asToken(raw: string) -> string<token> = raw
func logLine(msg: string{not token}) -> unit ! {IO} = println(msg)

export func exchange(acct: string) -> unit ! {IO} {
  let tok = asToken(mintHook(acct));
  logLine("minted ${tok}")
}
```

**V15, rejected** (the label survives a plain-`string` return; second fixture: inline record):
```ailang
module cls_prop
import std/io (println)
import std/json (encode, jo, kv, js)
pure func asToken(raw: string) -> string<token> = raw
pure func tokenBody(t: string<token>) -> string = encode(jo([kv("access_token", js(t))]))
func logLine(msg: string{not token}) -> unit ! {IO} = println(msg)
-- Does the label survive through tokenBody's plain `string` return?
export func go(x: string) -> unit ! {IO} = logLine(tokenBody(asToken(x)))
```
```ailang
module cls_rec
import std/io (println)
pure func asToken(raw: string) -> string<token> = raw
func logLine(msg: string{not token}) -> unit ! {IO} = println(msg)
-- Record-field hole: does the label survive a record?
export func go(x: string) -> unit ! {IO} {
  let r = {access: asToken(x)};
  logLine(r.access)
}
```

**V13, accepted: the hole that makes the classifier mandatory:**
```ailang
module leak
import std/io (println)

func mintHook(acct: string) -> string ! {IO} = "dp_secret_for_${acct}"
func logLine(msg: string{not token}) -> unit ! {IO} = println(msg)

export func exchange(acct: string) -> unit ! {IO} {
  let tok: string<token> = mintHook(acct);
  logLine("minted ${tok}")
}
```

V16 (the declared-record hole) is reproduced verbatim in issue #1523.

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

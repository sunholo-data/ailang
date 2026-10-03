# Sprint Plan: M-MCP-OAUTH-PACKAGE (`sunholo/mcp_oauth` 0.1.0)

## Summary

Build the OAuth 2.1 authorization server package specified in `m-mcp-oauth-package.md` (Design
Freeze D1–D6 ratified 2026-10-02) in the `sunholo-data/ailang-packages` monorepo under
`packages/mcp-oauth/`.

**Duration:** 5 days (about 1,600 LOC, AILANG plus tests).
**Dependencies:**
- S0 (#1522, `Net[scope=public]`) gates **P2's CIMD fetch** and deployment, not P1. It is being
  fixed in parallel by a subagent.
- #1523 (the declared-record label hole) is designed around (D4), so it doesn't block.

**Risk level:** high. This is security-critical; every control is proved or tested, as listed below.

**Out of scope:** Parse adoption (the next sprint, in `docparse`); DCR; JWT; OIDC.

## Registry Reuse Audit

Searches: `ailang pkg search` for auth, jwt, firestore, firebase, crypto, oauth, session, cookie,
http, kv, hmac, pkce.

| Milestone | Package | Action | Reason |
|---|---|---|---|
| P1 | — | none | Pure PKCE/redirect/URL/TTL predicates; `std/crypto` + `std/bytes` suffice |
| P2 | `sunholo/auth` 0.4.1 | none in the package (**depend** in adopters) | The package takes `authenticate` as a hook (D3); Parse wires `auth/firebase_jwt` |
| P2 | `sunholo/firestore` 0.7.3 | none in the package (**depend** in adopters) | Storage is a hook; the package ships an in-memory test store only |
| P2 | `sunholo/oauth` 0.1.0 | none | A client-side installed-app flow; no shared AS logic |
| P3/P4 | — | none | Metadata, templates, integration |

## Milestones

Every criterion names a **test**, or a **command** with its expected output.

### P1: Pure core (`mcp_oauth/core`), Z3-verified (about 400 LOC)
- `s256`, `pkceOk`, `verifierOk` (length 43..128, charset `[A-Za-z0-9-._~]`);
- `redirectAllowed` (exact), `loopbackRedirectOk` (RFC 8252: any port, exact path);
- `cimdUrlOk` (https, no IP literal / localhost / `.internal` / `metadata`, no userinfo or fragment);
- `codeExpiresAt` / `isExpired` (60 s);
- `asMetadata` (RFC 8414 JSON).

**Acceptance:**
- [ ] `ailang test packages/mcp-oauth/core_test.ail`: all pass, including test `"pkce rfc7636 vector"` (App. B) and `"pkce rejects wrong verifier"`.
- [ ] `ailang verify packages/mcp-oauth/core.ail`: `redirectAllowed`, `verifierOk`, `cimdUrlOk`, `codeExpiresAt` all **VERIFIED**; zero VIOLATIONs.
- [ ] Negative contract fixture `packages/mcp-oauth/tests/broken_redirect.ail` (prefix match): `ailang verify` reports **VIOLATION** (proves the proof bites).
- [ ] Tests named `"cimd rejects <case>"` for: `http://`, IP literal v4/v6, `localhost`, `*.internal`, `metadata.google.internal`, userinfo, fragment; each returns false.

### P2: Flow + CIMD (`mcp_oauth/flow`, `mcp_oauth/cimd`) (about 650 LOC)
- `beginAuthorize`, `completeLogin`, `exchangeCode`, `refresh`;
- hooks record `{authenticate, mint, store, revoke}` (D3);
- labels `<authcode>`/`<token>`, with the classifier on receipt (V14);
- codes and tokens stored as digests; single use; replay revocation; refresh-family rotation;
- `Rand[mode=crypto]` minting; `Net[scope=public] @limit=1` on the CIMD fetch (after S0).

**Acceptance:**
- [ ] Tests in `flow_test.ail` (in-memory store hook):
  - `"authorize rejects plain pkce"`;
  - `"authorize rejects unregistered redirect"`;
  - `"code is single use"`;
  - `"code replay revokes"` (the revoke hook received the first redemption's token digest);
  - `"expired code rejected"`;
  - `"code bound to client and redirect"`;
  - `"refresh rotates"`;
  - `"refresh reuse revokes family"`.
- [ ] `bash packages/mcp-oauth/tests/ifc_leaks.sh`: injects three leaks into a copy of `flow.ail` (log the classified token, log the code, log the token response body); `ailang check` **rejects all three**; the clean copy passes.
- [ ] `bash packages/mcp-oauth/tests/lint.sh`:
  - no `type … { … : string<` in `flow.ail` (D4 / #1523);
  - every minting function declares `Rand[mode=crypto]`;
  - no `==` on lines touching `code`/`token`/`verifier` digests (constant-time only).
- [ ] After S0 lands: `cimd.ail` fetch declares `Net[scope=public]` (`grep`); test `"cimd fetch refused for private resolution"` uses S0's injected resolver (or is documented as covered by the S0 tests in ailang if package tests cannot inject).

### P3: Packaging (about 250 LOC)
- `ailang.toml`:
  - `[effects] max = ["Net","Clock","Rand","IO","FS","Env","Declassify"]`;
  - `[release] kind = "feature"`;
  - `[metadata] repository`;
- `CHANGELOG.md`, `AGENT.md` (adopter checklist: the hooks, `AILANG_TRACE_VALUES=off`, the route template), `_smoke.ail`, `routes_template.ail`.

**Acceptance:**
- [ ] `ailang pkg quality packages/mcp-oauth`:
  - exit 0, with no PUB gate;
  - report the numbers: contracts verified/total, tests n/n, smoke ✓, effects ceiling declared.
- [ ] `ailang publish --dry-run` in `packages/mcp-oauth`: no refusal.

### P4: Integration (about 300 LOC)
An example service (fixture module in the package's `tests/`) mounts the four endpoints next to a
Phase A `@mcp_auth` tool, with an in-memory store, `authenticate` stubbed to a test user, and
`mint` issuing tokens that its `@mcp_token_verifier` accepts.

**Acceptance:**
- [ ] `ailang serve-api --mcp-http --oauth-issuer http://localhost:PORT …` + `ailang mcp check http://localhost:PORT/mcp/connect/ --target both`: **all five checks PASS** (check 5 against the package's own metadata).
- [ ] `bash packages/mcp-oauth/tests/e2e_flow.sh`, a scripted PKCE run with curl:
  - authorize → login-complete → token → call the gated tool → 200;
  - replay the code → `invalid_grant` + the token stops working (401).

## Success Metrics
- `pkg quality`: all security predicates verified; tests and smoke green.
- `ailang mcp check --target both`: 5/5 on the example.
- The threat table T1–T10 has every row's proof or test present.

## Notes
- P1 can start now; P2's CIMD fetch waits for S0's `Net[scope=public]`. The rest of P2 doesn't.
- Work happens in a worktree of `ailang-packages` off `origin/main`, because the shared checkout
  sits on a stale branch with someone's uncommitted edits.
- Monorepo flow (AGENTS.md): branch `sprint/mcp-oauth`, then a PR to `main`.

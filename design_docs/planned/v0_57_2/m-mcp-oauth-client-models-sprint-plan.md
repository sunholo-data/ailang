# Sprint Plan: M-MCP-OAUTH-CLIENT-MODELS

**Design doc:** [m-mcp-oauth-client-models.md](m-mcp-oauth-client-models.md)
**Progress JSON:** `.ailang/state/sprints/sprint_M-MCP-OAUTH-CLIENT-MODELS.json`
**AILANG prompt version loaded:** v0.16.11 (`ailang prompt --version-active`; `ailang prompt` read in full, 105 KB, 2026-11-05)

## Placement and human-decision rules

The plan sits next to its design doc in `design_docs/planned/v0_57_2/`. Design Freeze items D1–D8
(design doc §Design Freeze) are **OPEN**. Each milestone below names the items it depends on. Before
the executor starts such a milestone, it checks that each named item is ticked in the design doc.
If one is not: **STOP: commit current work, report `BLOCKED: <D-item>`, and end the run.**

The PENDING vendor measurements P0-1..P0-5 (design doc §P0) do **not** gate any milestone here. They
gate an adopter *enabling* a feature in a deployment, which is outside this sprint.

## Summary

Ship `sunholo/mcp_oauth` 0.2.0, with three new client models:
- pre-registered confidential clients;
- DCR;
- per-client refresh policy plus revocation.

Keep 0.1.2 behaviour byte-identical for docparse and the executor. In serve-api, add an
identity-returning token verifier with a bound `_auth` param and a per-subject rate limit, configurable
`serverInfo`, and per-surface resources. Fix `ailang mcp check`'s framing probe for CIMD-only servers.

**Duration:** 11 working days (9.5d estimate + ~15% buffer), across three lanes that can run in
parallel: package, serve-api and mcpcheck.
**Dependencies:**
- PR #2112 (#1631, arg binder) merged before M2. It is OPEN and CONFLICTING as of 2026-11-04 (design V25).
- D1–D8 ticked for the milestones that cite them.
**Risk level:** Medium. The work is security-critical but additive, and every new behaviour is off by
default.

## Current status analysis

### Completed recently (velocity reference)
- ✅ `sunholo/mcp_oauth` 0.1.0, sprint M-MCP-OAUTH-PACKAGE: about 1,600 LOC over 5 planned days, all
  of P1–P4 passing (`.ailang/state/sprints/sprint_M-MCP-OAUTH-PACKAGE.json`, target 320 LOC/day).
  The closest comparable work: the same package, the same test style, and the same reviewers.
- ✅ The serve-api listed surface and `--mcp-connect-only` (v0.51–v0.54) touched the same
  `internal/apiserver` files.
- The last 7 days on `dev` were mostly fleet and launchd work, plus v0.57.1. Nothing touched
  `internal/apiserver`, `internal/mcpcheck` or `packages/mcp-oauth` (`analyze_velocity.sh 7`).

### Velocity
- Planning rate: about 250 LOC/day for security code with shell-suite tests. That is lower than
  the 0.1.0 sprint, because each new control has a mutation check.
- Estimated total: about 2,520 LOC (implementation and tests).

### Remaining from the design doc
Everything: M1a–M1e, M2, M3a–M3c. M4 is a dependency tracked in its own plan
(`design_docs/implemented/v0_57_3/m-serve-api-typed-arg-binder-sprint-plan.md`).

## Registry reuse audit

Searches run on 2026-11-05: `ailang pkg search oauth` → `sunholo/mcp_oauth@0.1.2`,
`sunholo/oauth@0.1.0` and `sunholo/linkedin@0.6.2`. `ailang pkg search mcp_files` →
`sunholo/mcp_files@0.1.4`.

| Milestone | Package | Decision | Reason |
|---|---|---|---|
| M1a–M1e | `sunholo/mcp_oauth` | **contribute** | The capability belongs in this package (design §M1). It ships as 0.2.0 through its `pkg:sunholo/mcp_oauth` inbox and repo |
| M1a–M1e | `sunholo/oauth` | none | A client-side installed-app flow (consent URL, loopback, refresh). It is not an authorization server |
| M2, M3a, M3b | — | none | Go core in `internal/apiserver` and `serveapi/protocol`. Not package-like |
| M3c | — | none | Go core in `internal/mcpcheck`, plus one static JSON file in `docs/static/` |

## Milestones

Every acceptance command is run unpiped, because piped `make` reports the pipe's exit code. The
executor gates are `make test-changed` plus the targeted gates named below. CI runs the full
`make test`. Changelog entries are fragments in `changelogs/unreleased/2026-MM-DD-<slug>.md` that
open with `### `.

Package milestones run in `ailang-packages` (a worktree off `origin/main`). serve-api and mcpcheck
milestones run in `ailang` (a worktree off `origin/dev`).

### M1a — confidential clients (lane: package) · 1.5d · ~520 LOC · gates: D1, D4

- **Tasks:**
  - Add `ClientConfig` and `defaultClientConfig()`.
  - Add `core.clientKind`, with a Z3 `ensures` that no string is both `cimd` and `registered`.
  - Add `flow.createClient` / `rotateClientSecret` / `deleteClient`. Secrets are minted under
    `Rand[mode=crypto]`, labelled `<secret>`, and stored as a digest.
  - Add the internal `authenticateClient` (Basic and post).
  - Add `beginAuthorizeFor` (dispatch on `clientKind`) and `tokenRequest` (both grants; it
    authenticates internally).
  - Add the legacy-wrapper CIMD guard on `exchangeCode`/`refreshTokens`.
  - Store the client `kind` on code and refresh records.
  - Add the new module `clients.ail` (record JSON, no secrets) and its `lint.sh` rule.
- **Acceptance:**
  - `cd packages/mcp-oauth && bash tests/flow_check.sh && bash tests/cimd_check.sh && bash tests/ifc_leaks.sh && bash tests/lint.sh`
    all pass **unmodified**. This is the backward-compatibility gate.
  - `bash tests/confidential_check.sh` passes these cases:
    - `TestWrongSecretInvalidClient`: 401 + `WWW-Authenticate: Basic`;
    - `TestConfidentialWithoutSecret`;
    - `TestCimdWithSecretRefused`;
    - `TestLegacyExchangeRefusesRegisteredId`;
    - `TestLegacyRefreshRefusesRegisteredId`;
    - `TestKindMismatchRefused`;
    - `TestSecretNeverStored`;
    - `TestRegisteredIdNeverFetched`;
    - a GE-shaped happy path with `client_secret_basic` and redirect
      `https://vertexaisearch.cloud.google.com/oauth-redirect` → tokens.
  - `ailang verify core.ail` reports `clientKind` VERIFIED and 0 VIOLATION.
  - `tests/ifc_leaks.sh` gains a case that logs the client secret, and it must fail to compile.
  - Mutation: drop the record-method check in `authenticateClient` (scratch copy), and
    `TestConfidentialWithoutSecret` fails.
- **Risks:**
  - `flow.ail` grows from 307 to about 560 lines. The target is < 800 (`make check-file-sizes` has
    no package equivalent; check with `wc -l`).
  - Effect rows on the new functions must stay within `[effects] max`.

### M1b — DCR + abuse controls (lane: package) · 1.5d · ~420 LOC · gates: D2

- **Tasks:**
  - Add `flow.registerClientDcr`.
  - Add the `clients` validators: redirect rules, a `dcrRedirectHosts` allowlist, grant and
    response types, the auth method (with `none` when absent), and RFC 7591 errors.
  - Add `clients.trustedCallerIp(xff, realIp)`, which takes the rightmost entry, mirroring
    `clientIPFromHeader`.
  - Add per-caller and global window counters; unused-registration TTL (24 h), sliding to 90 days
    idle after first use; a 16 KB body cap; at most 5 URIs.
  - Store no `client_name` or `logo_uri`; `describeRequest` adds `registered`.
- **Acceptance:**
  - `bash tests/dcr_check.sh` passes these cases:
    - `TestDcrRateLimitPerCaller` (the 11th registration in a window is refused);
    - `TestDcrGlobalCap`;
    - `TestDcrUnusedExpires`;
    - `TestDcrBodyCap`;
    - `TestTrustedCallerIpIgnoresForgedPrefix` (`"1.2.3.4, 9.9.9.9"` → `9.9.9.9`);
    - `TestDescribeRequestOmitsClientName`;
    - `TestDcrRedirectAllowlist` (a host outside `dcrRedirectHosts` → `invalid_redirect_uri`).
  - `ailang verify clients.ail` reports `dcrRedirectOk` VERIFIED.
  - The M1a gate suites are still green.
  - Mutation: switch `trustedCallerIp` to the first entry, and the forged-prefix test fails.

### M1c — refresh policy + RFC 7009 revocation (lane: package) · 1d · ~330 LOC · gates: D3

- **Tasks:**
  - Add the `stable` policy: no rotation, `refresh_token` omitted from the refresh response, idle
    and absolute expiry, client authentication required for confidential clients.
  - Add `flow.revokeRequest` (always 200).
  - Prune family digests to unexpired access tokens (T8).
- **Acceptance:**
  - `bash tests/refresh_policy_check.sh` passes:
    - `TestStableRefreshNoRotation`;
    - `TestStableRefreshNeedsClientAuth`;
    - `TestStableRefreshAbsoluteExpiry`;
    - `TestFamilyDigestsPruned`;
    - `TestStableRefusedForPublicClient`, per the D3 recommendation. If Mark rules otherwise, the
      test follows the ruling.
  - `bash tests/revoke_check.sh` passes `TestRevokeEndpointKillsFamily` and
    `TestRevokeAlways200`.
  - `tests/flow_check.sh`'s rotation and family-revocation cases still pass unmodified.
  - Mutation: mark a stable token `used`, and `TestStableRefreshNoRotation` fails.

### M1d — auth-method intersection (lane: package) · 0.5d · ~120 LOC · gates: D1

- **Tasks:**
  - `cimd.validateClientDoc` refuses a document whose plural
    `token_endpoint_auth_methods_supported` is present and lacks `none`. The singular field is
    ignored.
  - Add `core.authMethodOk`.
- **Acceptance:**
  - `ailang test --package .` includes `cimd accepts chatgpt live doc`, which uses design E17
    verbatim (singular `private_key_jwt`, plural `["none","private_key_jwt"]` → accepted), and
    `cimd refuses plural without none`.
  - `ailang verify core.ail` reports `authMethodOk` VERIFIED.
  - Mutation: treat the singular field as binding, and the ChatGPT fixture test fails.

### M1e — metadata, example service, docs, publish 0.2.0 (lane: package) · 1d · ~300 LOC

- **Tasks:**
  - Add `core.asMetadataFor(issuer, cfg)`, with a golden test that `asMetadata(issuer)` is
    byte-identical to 0.1.2.
  - `tests/example/svc.ail` gains a GE-shaped registered client plus the `/oauth/register` and
    `/oauth/revoke` routes.
  - `tests/e2e_flow.sh` drives the new paths under real serve-api.
  - AGENT.md gets three recipes; CHANGELOG 0.2.0; `ailang.toml` version 0.2.0.
- **Acceptance:**
  - `bash tests/e2e_flow.sh` passes.
  - `ailang pkg quality packages/mcp-oauth --json` shows `verified == total` for the pure security
    predicates.
  - The golden test passes.
  - In the ailang repo, with only `executor_oauth/ailang.lock` bumped to 0.2.0 and no source change,
    `bash executor_oauth/test_oauth.sh` passes 21/21.
  - The package is published (`ailang pkg publish`); `ailang pkg search mcp_oauth` shows 0.2.0.
- **Gate:** publishing is an outward release. Run it only after M1a–M1d pass and the package PR is
  merged to `origin/main` of ailang-packages.

### M2 — serve-api identity verifier (lane: serve-api) · 2d · ~380 LOC · gates: D5, D6; blocked on #2112

- **Precondition:** run `gh pr view 2112 --json state`. If the PR is not `MERGED`, follow design
  §M4: bind `_auth` in `makeToolHandler` and add `TestAuthBindingParity`. Do not wait.
- **Tasks:**
  - `serveapi/protocol`: add `IdentityVerifyFunc`, `NewIdentityGate` and `AdmitCallIdentity`.
    `NewBearerGate` wraps them.
  - `routes.go`: `VerifierSigOK` accepts `Option[{subject, client_id, scopes}]`.
  - `mcp_auth.go`: call the identity verifier (`Some` with an empty subject → 401), stamp
    `X-Ailang-Mcp-Identity`, and strip any client-sent copy.
  - Add the registration rules: an `_auth` export must be gated; it is absent from `/mcp/`, REST,
    A2A and stdio; it cannot be `@optional`, `@mcp_secret` or `@mcp_file`; a bool verifier with
    `_auth` is an ERROR.
  - Add `--mcp-subject-rpm` / `--mcp-subject-burst`, reusing `NewIPRateLimiter` and answering 429 +
    `Retry-After`.
  - Add `examples/runnable/serve_api_mcp_identity.ail` and its manifest entry.
  - Adopt the identity verifier in `executor_oauth/oauth.ail`.
- **Acceptance:**
  - `go test ./internal/apiserver/ ./serveapi/... -run 'Auth|Identity|Verifier|SubjectRate|Mcphttp'`
    passes these new tests:
    - `TestAuthParamNotForgeableByArgs`;
    - `TestIdentityHeaderStripped`;
    - `TestAuthExportAbsentFromAgentSurface`;
    - `TestAuthExportNoRESTRoute`;
    - `TestAuthWithNoauthIsError`;
    - `TestEmptySubjectRejected`;
    - `TestBoolVerifierUnchanged`;
    - `TestSubjectRateLimit429`;
    - `TestMcphttpGateUnchanged`.
  - The existing `mcp_header_auth_test.go` and `bearer_gate*_test.go` pass unmodified.
  - `make test-changed`, `make verify-examples`, `make check-file-sizes` and `make check-changelog`
    pass.
  - `bash executor_oauth/test_oauth.sh` passes, with a new check that a tool sees the signed-in
    account.
  - Mutation: skip the header strip, and `TestIdentityHeaderStripped` fails.
- **File-size watch:** `mcp.go` is at 605 lines. If adding `_auth` binding pushes it past 650, move
  the reserved-param binding into a companion `mcp_bind.go`.

### M3a + M3b — `serverInfo` and per-surface resources (lane: serve-api) · 1d · ~200 LOC · gates: D7, D8

- **Tasks:**
  - Add `serverIdentity` and the `--server-info` flag. It feeds MCP `serverInfo` on both surfaces,
    the A2A card version and OpenAPI `info.version`. The default is `ailang-api` + the binary
    version.
  - `registerResources(surface)`: `/mcp/connect/` drops `ailang://meta/modules`; `/mcp/` uses
    `exposedView` and honours `--no-introspection`.
  - Update `testdata/mcp_ui.golden.json`.
  - Update the docs: `serve-api.md` (the flag and resources) and `mcp-connectors.md` §6 (the
    `serverInfo` line).
- **Acceptance:**
  - `go test ./internal/apiserver/ -run 'ServerInfo|Resources'` passes:
    - `TestServerInfoFlag`, which checks `title` on `initialize`;
    - `TestListedSurfaceNoMetaModules`;
    - `TestAgentSurfaceMetaModulesExposedView` (no `@noexpose` export, and no verifier name);
    - `TestNoIntrospectionDropsMetaModules`.
  - The live repro in design V18, re-run against the built binary, shows no `secretHelper` or
    `verifyTok` on either surface.
  - `make test-changed` and `make check-changelog` pass.

### M3c — `mcp check` framing probe (lane: mcpcheck) · 1.5d · ~250 LOC · gates: D7 (probe URL)

- **Tasks:**
  - Add `docs/static/mcp-check/client-metadata.json`, with its `client_id` equal to its own URL.
  - Probe client selection: CIMD, then `--probe-client-id`/`--probe-redirect-uri`, then Skip.
  - Use a fresh 43–128-character verifier with S256; keep `response_type`/`state`; add `resource`.
  - Add a `CheckRedirect` that stops at the probe's own redirect URI.
  - Add two WARN checks: an unbranded `serverInfo`, and `ailang://meta/modules` on the listed
    surface.
  - Update the docs: `mcp-connectors.md` "Checking consent page framing".
- **Acceptance:**
  - `go test ./internal/mcpcheck/` passes:
    - `TestFramingProbeCimdPass`;
    - `TestFramingProbeFlagsClient`;
    - `TestFramingProbeNoCimdSkips`;
    - `TestFramingProbeStopsAtCallback`;
    - `TestServerInfoUnbrandedWarn`;
    - `TestListedMetaModulesWarn`;
    - the existing framing tests, with their expected status updated deliberately.
  - In the package, `tests/e2e_flow.sh` runs `ailang mcp check --target both` against the example
    service, with `authorization-framing PASS`. This step needs M1e and M3c both landed; it runs
    in M1e's suite once M3c is on `dev`.
  - The probe document's URL is reachable after the docs deploy:
    `curl -s https://ailang.sunholo.com/mcp-check/client-metadata.json | jq -r .client_id` equals
    the URL. This is post-merge, recorded in the progress JSON notes.
  - `make test-changed` and `make check-changelog` pass.

## Day-by-day

Lanes A (package) and B (serve-api/mcpcheck) are independent until M1e's e2e step.

| Day | Lane A: package | Lane B: serve-api / mcpcheck |
|---|---|---|
| 1 | M1a: `ClientConfig`, `clientKind` + Z3, `createClient`, secret storage | M3c: probe document, client selection, S256 verifier |
| 2 | M1a: `authenticateClient`, `tokenRequest`, legacy guard, suite + mutation | M3c: `CheckRedirect`, WARN checks, tests; M3a: `serverIdentity` + flag |
| 3 | M1b: validators, `trustedCallerIp`, counters | M3b: per-surface resources, golden, docs |
| 4 | M1b: TTLs, `dcr_check.sh`, mutation | M2 precondition (#2112), `IdentityVerifyFunc` / gate |
| 5 | M1c: stable policy, revocation, pruning | M2: registration rules, header stamp/strip, `_auth` binding |
| 6 | M1d: auth-method intersection + ChatGPT fixture | M2: subject rate limit, example, executor adoption |
| 7 | M1e: `asMetadataFor`, example service, e2e | M2: tests, mutation, docs |
| 8 | M1e: AGENT.md, CHANGELOG, `pkg quality`, PR | buffer / review fixes |
| 9 | M1e: publish 0.2.0 after merge; bump the executor lock and re-run `test_oauth.sh` | buffer |
| 10–11 | buffer (about 15%) | buffer |

## Success metrics

The design doc's six success metrics, each mapped to a milestone above:
1. backward compatibility: M1a gate + M1e golden + executor lock bump;
2. GE shape: M1a;
3. DCR bounds: M1b;
4. threats T11–T17 mapped: M1a–M1c + M2;
5. identity: M2;
6. listing hygiene: M3a–M3c.

**Example files:** `examples/runnable/serve_api_mcp_identity.ail` (new, M2), and the package's
`tests/example/svc.ail` (extended, M1e).

**Showcase module checklist** (`examples/runnable/serve_api_mcp_identity.ail`):

| Row | Decision |
|---|---|
| Contracts | skip: a serve-api wiring example with no pure arithmetic invariant. The package's contracts live in `core.ail`/`clients.ail` |
| Effects | include: `verifyToken(token: string) -> Option[McpIdentity] ! {SharedMem}` and `whoami(_auth: McpIdentity) -> string` (pure) |
| Inline tests | include: `export pure func isSignedIn(subject: string) -> bool tests [("", false), ("u1", true)]` (checked with `ailang check` and `ailang test` on v0.57.1, 2026-11-05) |

**Docs:**
- `docs/docs/guides/serve-api.md`: `_auth`, the identity verifier, `--server-info`,
  `--mcp-subject-rpm`, resources;
- `docs/docs/guides/mcp-connectors.md`: §3 (0.2.0 client models), §6 (Google rows updated per
  shipped milestone), framing;
- the package's AGENT.md.

## Dependencies

- **PR #2112** for M2 (fallback in design §M4).
- **D1–D8** for the milestones that cite them.
- **Publishing 0.2.0** (M1e) before docparse or the executor can depend on it. Adopters opt in
  separately, after the P0 measurements.

## Open questions (human-decision gates)

Each item is OPEN in the design doc's §Design Freeze. When one is unticked at the start of the
milestone that cites it: **STOP: commit current work, report `BLOCKED: <item>`, and end the run.**
- D1 (confidential-client model) → M1a, M1d
- D2 (DCR off by default) → M1b
- D3 (refresh policy and lifetimes) → M1c
- D4 (PKCE required for confidential clients) → M1a
- D5 (identity contract, `_auth`) → M2
- D6 (per-subject limit in serve-api) → M2
- D7 (`--server-info`; probe CIMD URL) → M3a, M3c
- D8 (listed-surface resources) → M3b

## Notes

- The quorum ran twice, and the re-quorum guardrail is spent (design §Quorum record). The round-1
  fixes are in the doc but have not been re-reviewed by the quorum.
- The sprint is **not** handed off to sprint-executor in this session. D1–D8 are open, so a handoff
  would only produce `BLOCKED` runs. Send the handoff after Mark rules.

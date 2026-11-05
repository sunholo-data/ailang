# Integrations review (internal): AI platforms, 2026-11-05

**Status**: Internal audit, read-only. This is not a design, and nothing in it is decided.

**Scope**: What AILANG (serve-api and packages) and AILANG Parse (docparse) support *today* for:
- OpenAI: ChatGPT apps/plugins and Codex.
- Anthropic: the Claude connectors directory and Claude Code plugins.
- Google Gemini Enterprise (GE): agents, A2A and MCP.
- Google consumer Gemini, Search, and Agentspace / Vertex AI Search.

It also covers the shared layers: MCP, A2A and OAuth2.

**Baselines read**: all from origin, with no working trees touched.

| Repo | Ref |
|---|---|
| ailang | `origin/dev` d49e0e01e (2026-11-05) |
| ailang release | **v0.57.1** (b91eb27, 2026-11-04) |
| ailang-packages | `origin/main` 1c70be9 |
| docparse | `origin/main` 76ad138 (latest tag **v0.43.0**, fd8bcfa, 2026-11-03) |
| docparse-skill | `origin/main` 74b8fda (2026-10-06, unchanged for a month) |
| ailang-parse | `origin/main` 566ee31 |
| ailang-multivac | `origin/dev` 95751b9, which is the same commit as the `prod` branch |

**Live checks**: run on 2026-11-05 at about 07:00Z.
- Targets: `https://docparse.ailang.sunholo.com` (prod) and the dev executor OAuth service `https://ailang-dev-executor-mcp-oauth-812435936917.europe-west1.run.app`.
- Binary: the v0.57.1 darwin-arm64 release. Its sha256 `3fee920abdd00f83b40c972d80a61bef51205708ca08593deef421edbfed0011` matches both the release `.sha256` asset and `SHA256SUMS`.
- Transcripts are in Appendix A.

**Prod docparse runs** AILANG runtime **v0.56.1**: `/api/v1/health` reports `"ailang_commit":"v0.56.1"`, and the pin is at `Dockerfile:46` and `cloudbuild-deploy.yaml:14`. So prod does **not** yet have the v0.57.x fixes (for example the Vertex regional host, #2077).

---

## 0. Headline findings

1. **The Claude and ChatGPT connector wire is ready, but neither directory listing is submitted.**
   - `ailang mcp check --target both` passes on docparse `/mcp/connect/`: 8 checks pass, 1 warns, 0 fail.
   - Lazy-auth is in place: an HTTP 401 with `WWW-Authenticate` and `resource_metadata`, plus `_meta["mcp/www_authenticate"]` on the tool error.
   - Every gated tool carries an oauth2 `securitySchemes` entry.
   - ChatGPT `openai/fileParams` is on mcpParse, mcpConvert and editDocument.
   - MCP Apps upload widget: the widget-only tool carries `visibility:["app"]`.
   - What remains is listing work and hygiene (§3):
     - the OpenAI domain-verification route returns 404;
     - `serverInfo` says `ailang-api 0.8.1`, not "AILANG Parse";
     - the privacy policy contradicts shipped retention;
     - the site and skill copy are stale;
     - there are no reviewer credentials and no demo video.
2. **Typed argument binding is still broken in the release (#1631, P1, open).**
   - MCP `tools/call` and A2A call `CallPreserveFloats`, so `int` params fail: `internal/apiserver/mcp.go:456`, `a2a.go:226`, `mcp_ui_serve.go:103`.
   - REST calls `Call` (`routes_dispatch.go:138`), so whole-number `float` params fail there instead.
   - The fix exists as **PR #2112** (`arg_bind.go`, a single binder for all surfaces), but it is **CONFLICTING**, with CI skipped.
   - The design `m-serve-api-typed-arg-binder` targeted v0.57.1, which shipped without it.
   - docparse works around it in code: `count: float` at `mcp_files.ail:388`, and `maxChars` passed as a digit string.
3. **A2A cannot be used with Gemini Enterprise or any A2A 0.2+/0.3 client.**
   - **serve-api `--a2a` is pre-0.2.** It supports only `tasks/send`, and `tasks/get` always errors (`a2a.go:133-153, 272`). There is no `message/send` or `message/stream`, and no `protocolVersion`.
   - The card hardcodes `"version":"0.8.1"` and has no `securitySchemes` (`a2a.go:93-107`).
   - The A2A path is **not behind the Bearer gate.** `--mcp-connect-only` refuses `--a2a` for that reason (`mcp_auth.go:182-183`), and `--api-key-env` exempts `/a2a/` (`auth.go:28-31`).
   - **docparse** serves a hand-written card at `/.well-known/agent.json`:
     - It claims `"protocol":"a2a/0.3"` but has no task endpoint (`/a2a/` returns 404 live).
     - It has no `/.well-known/agent-card.json` (404).
     - It declares `apiKey` "in: body" (non-standard).
     - Its `url` falls back to the dev run.app URL when `DOCPARSE_API_URL` is unset (`api_server.ail:2145-2196`).
   - The only A2A 0.3 implementation in the estate is the Node **resident agent** (`docker/resident/lib/a2a.mjs`: `message/send`, `tasks/get`, `protocolVersion "0.3.0"`). It is a separate per-user coding agent, dev-only.
4. **Gemini Enterprise today is one dev-only, IAM-only MCP connector.**
   - The connector is `ailang-mcp-dev` in project `sunholo-gemini-enterprise`, pointing at `ailang-dev-executor-mcp`. It was live and observed working on 2026-10-14.
   - Since then the only change is a toolchain patch on 10-22. There is no prod service.
   - **OAuth from GE cannot work as built.** `sunholo/mcp_oauth` accepts only CIMD public clients (no client secret, no DCR). GE's custom-connector OAuth form expects a pre-registered client id and secret.
   - The GE org-policy egress allowlist lists only the IAM-only dev executor host (`terraform-gemini-enterprise/main.tf:76`). It excludes both the executor OAuth service and docparse.
   - A2A (P2) and GE agent registration (P3) have not started.
5. **Google consumer Gemini / Search ("Connected Apps") exists only as an unapproved design.**
   - `design_docs/planned/m-google-connected-apps-distribution.md` (10-29) is "awaiting user approval".
   - No code or Terraform exists in any repo for Agentspace, Vertex AI Search, Discovery Engine or Google's Agent Registry.
6. **Reuse is good on MCP, OAuth and files.**
   - The annotations, the two surfaces, the Bearer gate, `mcp_oauth` and `mcp_files` are all generic, and two services consume them: docparse and the executor.
   - The gaps (detailed in §4):
     - A2A is a stub in serve-api and is reimplemented by hand in docparse and in Node.
     - The token verifier is `(string) -> bool`, so tools re-derive identity from `_headers`.
     - `mcp_oauth` has no confidential-client, DCR or Google-IdP path.
     - There is no Google ID-token verifier package.
     - The L5 packaging template was never built: the docparse-skill manifests are hand-maintained.

---

## 1. Capability matrix

Status key:
- **shipped**: released and live in prod.
- **dev-only**: live only in a dev project.
- **partial**: shipped with a known gap.
- **planned**: a design doc exists, but no code.
- **missing**: no design and no code.

The matrix has 3 columns:
- **AILANG platform (serve-api / packages)**: the status in bold, followed by the evidence.
- **docparse**: the status in bold, followed by the evidence.
- **Notes**: limits and gaps that cut across both.

| Capability | AILANG platform (serve-api / packages) | docparse | Notes |
|---|---|---|---|
| **MCP agent surface `/mcp/`** | **shipped**<br>`--mcp-http`<br>`internal/apiserver/mcp.go` (605 lines)<br>stdlib-only dispatcher `serveapi/protocol/mcphttp/`<br>protocol `2025-11-25` (live `initialize`) | **shipped**<br>12 tools live, incl. `mcpAuth`/`mcpAuthPoll` (`@mcp_agent_only`, `mcp_auth.ail:12-25`)<br>`apiKey` arg / `Authorization: Bearer dp_…` / `X-API-Key` (`api_keys.ail:910`) | Registry `server.json` remotes points here, not at `/connect/` (registry.modelcontextprotocol.io, v0.14.0) |
| **MCP listed surface `/mcp/connect/`** | **shipped** v0.51+<br>`7b03f7112`; no-slash variant `646b5e8c2`<br>`--mcp-connect-only` (v0.54.10, `98f0deeda`): no `/mcp/`, every tool gated by default | **shipped**<br>10 tools live; `mcp check` 0 FAIL<br>Dockerfile:219-232 (`--mcp-http --oauth-issuer $OAUTH_ISSUER`, no `--mcp-connect-only`) | `serverInfo` = `ailang-api 0.8.1` on both surfaces (G13 open)<br>The listed surface also advertises resource `ailang://meta/modules` (internal module list) |
| **Tool titles / hints** | **shipped**<br>`@mcp_title`, `@mcp_hints` (`9305f1c19`, `41e73efc7`)<br>hints are always explicit booleans | **shipped**<br>all 10 tools pass A7/O8 | The executor's 22 docs tools fail A7/O8 (110 FAILs live, §A.3) |
| **Agent-only / secret params** | **shipped**<br>`@mcp_agent_only`, `@mcp_secret`, registration rules (`a784aa7c0`)<br>`mcp_auth.go:69-150` | **shipped**<br>`@mcp_secret("apiKey")` on gated tools; the listed schema drops `apiKey` | |
| **MCP Apps widgets** | **shipped** v0.52.4<br>`@mcp_ui_resource`, `@mcp_ui`, `@mcp_app_only` → `_meta.ui.visibility=["app"]` (`95dd61348`; `mcp_ui.go:41,178-191`)<br>`text/html;profile=mcp-app` with CSP `connectDomains` | **shipped**<br>`ui://docparse/upload` (mcp_files ≥0.1.2)<br>`uploadDocument` (`@mcp_ui`)<br>`uploadViaHost` (`@mcp_app_only`, live `visibility:["app"]`) | No `openai/outputTemplate` or `openai/widget*` keys anywhere (`git grep` empty). ChatGPT rendering relies on its MCP Apps support; not live-tested in ChatGPT<br>Widget copy says "ask Claude" (`mcp_files.ail:~480`) |
| **File input: ChatGPT `openai/fileParams`** | **shipped** v0.52.4<br>`@mcp_file` (`163e32e75`; `mcp_file.go:48,224`)<br>pkg `mcp_files` 0.1.4 `fetchFileParam` under `Net[scope=public]` | **shipped**<br>`file: OpenAIFile` on mcpParse/mcpConvert/editDocument (`mcp_tools.ail:63-69,264-270,381-388`)<br>live `_meta["openai/fileParams"]` | No ChatGPT end-to-end upload recorded (guide §4 "has not been recorded yet") |
| **File input: upload link / widget** | **shipped**<br>pkg `sunholo/mcp_files` 0.1.4 (`2b3c9b2`, 10-08): single-use 10-min tokens, SEP-2631-shaped descriptor, `mcp-file://` refs, widget with direct POST + via-host fallback (≤1 MB) | **shipped** v0.35–0.37<br>`createUpload` (curl line, Claude Code/Codex)<br>widget → `POST /uploads`<br>fileRef reusable 30 min<br>`--net-max-bytes 50MB`; Cloud Run multipart cap 32 MB | Guide `mcp-connectors.md` §5 still says mcp_files is "in progress, not yet in the registry" (stale) |
| **OAuth2 AS (`sunholo/mcp_oauth`)** | **shipped**<br>0.1.2 (`f2568be`, 10-14): auth-code + PKCE S256, CIMD only, `token_endpoint_auth_methods ["none"]`<br>rotating refresh with family revoke; RFC 9207 `iss`<br>no DCR, no confidential clients, no revocation endpoint, no scopes, no OIDC | **shipped** v0.32+<br>`mcp_oauth.ail` (AS metadata at :212)<br>Firebase sign-in (Google, GitHub, email/password)<br>Firestore `oauth_records` with TTL<br>access token = minted `dp_` key, 24 h (`api_keys.ail:1759`) | Live AS metadata: no `registration_endpoint`; `client_id_metadata_document_supported:true`<br>**Blocks GE OAuth** (client id + secret form)<br>A ChatGPT live sign-in is still unrecorded |
| **Consent-page anti-framing** | **shipped** v0.53.1<br>default static security headers, `--static-header`, `--no-static-security-headers` (`2f7058b7b`, #1597 closed 10-09)<br>`mcp check` framing probe (`internal/mcpcheck/framing.go`) | **shipped**<br>live `X-Frame-Options: DENY` + `CSP frame-ancestors 'none'` on `/oauth-login.html`<br>JS `antiframe.js` still present (now redundant) | The `mcp check` probe is **structurally inconclusive** against any CIMD-only AS: it sends `client_id=ailang-mcp-check-dummy` (`framing.go:31`), gets a 400, so it WARNs on every adopter |
| **OAuth token verification (RS)** | **shipped**<br>`@mcp_token_verifier (string)->bool`, Bearer gate on gated `tools/call` only (`mcp_auth.go:35-66,197-254`)<br>`serveapi/protocol/bearer_gate.go` | **shipped**<br>`verifyMcpToken` (`mcp_oauth.ail:193-196`) → `validateApiKey` on `dp_` | Boolean only: no subject, scopes or audience reach the tool. No JWT/JWKS verifier. No Google ID-token verifier |
| **API key / device flow** | **partial**<br>`--api-key-header`/`--api-key-env` (`auth.go`) but `/mcp/`, `/a2a/`, `/.well-known/` are exempt (`auth.go:28-31`)<br>pkg `auth` 0.4.1 (key checks) | **shipped**<br>`/api/v1/auth/device{,/poll,/inspect,/approve}` (`device_auth.ail:44-313`)<br>approve page `www.sunholo.com/docparse/approve.html` (live 200) | Device flow is docparse-only code, not a package |
| **A2A agent card** | **partial (pre-0.2)**<br>`--a2a` → `/.well-known/agent.json` (`a2a.go:17-107`): hardcoded `version 0.8.1`, no `protocolVersion`, no `securitySchemes`, skills = every exposed export | **partial (static, non-conformant)**<br>hand-written `@route GET /.well-known/agent.json` (`api_server.ail:2145-2196`): `"protocol":"a2a/0.3"`, `apiKey` in body, dev-URL fallback<br>no `/.well-known/agent-card.json` (live 404) | GE requires `protocolVersion` (GE design R2/R3) |
| **A2A tasks** | **partial (pre-0.2)**<br>`tasks/send` only; `tasks/get` → error (`a2a.go:133-153,272`)<br>not behind the Bearer gate; incompatible with `--mcp-connect-only`<br>Node resident agent has A2A 0.3 `message/send` (`docker/resident/lib/a2a.mjs:513,702`), dev-only, a separate product | **missing**<br>no `--a2a`; `/a2a/` 404 live | GE design P2 (A2A gateway, ~2.5 d) is **planned**, not started |
| **IAM / Google identity** | **dev-only**<br>executor MCP behind Cloud Run IAM (GE service agent + m@sunholo.com), `terraform/cloud_run_executor_mcp.tf:39-40`<br>pkgs `gcp_auth` 0.8.1 (outbound tokens), `firebase_auth` 0.1.2 (Firebase ID tokens) | **missing** for callers<br>IAM used only for `signBlob` on download URLs (`download_link.ail:84`) | Nothing verifies a Google OIDC ID token in-app (GE design V12). Cloud Run IAM is the only Google-identity gate |
| **Gemini Enterprise registration** | **dev-only (MCP)**<br>custom MCP connector `ailang-mcp-dev` (console-created, not in Terraform)<br>org policy egress allowlist (`ailang-multivac terraform-gemini-enterprise/main.tf:60,76`)<br>runbook `docs/internal/gemini-enterprise-executor-mcp.md`<br>A2A agent registration P3 **planned** | **missing** | Tools start disabled and need manual enabling in the GE Actions tab (runbook) |
| **Google Search / consumer Gemini / Agentspace / Vertex AI Search** | **planned**<br>`m-google-connected-apps-distribution.md` (10-29), awaiting approval<br>no Discovery Engine / Agent Registry code | **missing** | Open: whether Google needs stable refresh tokens vs mcp_oauth rotation, and which category to file under |
| **Pre-submission linter** | **shipped**<br>`ailang mcp check --target anthropic\|openai\|both` (`7d61cc242`; `internal/mcpcheck/`) | n/a | Does not check: `serverInfo` branding, `/.well-known/openai-apps-challenge`, A2A cards, or extra resources on the listed surface |
| **Plugin packaging (Claude Code / Codex)** | **planned**<br>L5 template, never built (`m-serveapi-directory-ready.md` L5) | **shipped (hand-made)**<br>docparse-skill `.claude-plugin/marketplace.json` → plugin v0.17.0; `.mcp.json` → `/mcp/connect/`<br>`.agents/plugins/marketplace.json` + `plugin.json` with `extensions.com.openai.interface`<br>Codex install via `codex plugin marketplace add` | Skill content predates uploads and paging (no createUpload, fileRef, cursor) |

---

## 2. Per platform

### 2.1 OpenAI: ChatGPT apps/plugins and Codex

**Wired today (prod, docparse):**
- `/mcp/connect/` with OAuth 2.1:
  - CIMD; ChatGPT registers itself.
  - The `iss` parameter, so ChatGPT uses its stable `connector_platform_oauth_redirect`.
- `securitySchemes` on every tool (`oauth2` or `noauth`).
- `_meta["mcp/www_authenticate"]` on refused calls (live: an HTTP 401 *and* a JSON-RPC result with `isError` + `_meta`).
- `openai/fileParams` on the three document tools.
- The MCP Apps widget.
- `mcp check --target openai` passes on docparse.

**Codex:**
- The docparse-skill marketplace (`.agents/plugins/marketplace.json`, `skills/ailang-parse/agents/openai.yaml`).
- connect.html documents `codex mcp add` + `login` on `/mcp/connect/`, and `createUpload` gives Codex a curl upload line.

**Not done:**
- `/.well-known/openai-apps-challenge` returns **404** live. This is domain verification (submission kit row 112).
- `serverInfo` branding (G13).
- `submit_feedback` is on the listed surface, unauthenticated, with free text and contact (G12).
- No ChatGPT live OAuth sign-in has been recorded (`docs/docs/guides/mcp-connectors.md:443`).
- No ChatGPT widget render has been tested. serve-api emits only MCP Apps keys and no `openai/outputTemplate`.
- People-side items:
  - org verification of Holosun ApS;
  - demo video;
  - 5 positive and 3 negative test cases;
  - `brandColor`;
  - working default prompts (submission kit rows 95-122, dated 10-06).

**In flight:** nothing OpenAI-specific. PR #2112 (arg binder) would let docparse drop its float/string workarounds.

### 2.2 Anthropic: Claude connectors directory and Claude Code plugins

**Wired today:**
- Lazy auth on `/mcp/connect/` (a 401 with `resource_metadata`), RFC 9728 metadata in both the path-inserted and bare forms, and the no-slash URL.
- RFC 8252 loopback redirect for Claude Code.
- MCP Apps upload widget. The F1 spike proved a byte-exact upload in claude.ai web on 10-07.
- Claude Code OAuth worked 2026-10-06 (submission kit row 85).
- The Claude Code plugin: `docparse-skill/.claude-plugin/marketplace.json` → `plugins/ailang-parse/.claude-plugin/plugin.json` v0.17.0, with `.mcp.json` pointing at `/mcp/connect/`.

**Not done:**
- The directory submission has not been made. connect.html says "A listing in Claude's connector directory is coming".
- Reviewer credentials (G1). Email/password is on the login page, but neither a reviewer account nor an enabled provider is verified.
- Documentation URL (G8).
- Privacy policy fixes (G9, now worse; see §3).
- The 7 compliance acknowledgements.
- Running every tool from claude.ai as a custom connector before submitting.
- docparse design `mcp_oauth_same_origin_login.md` records that no person has completed a browser sign-in on the same-origin page (rollout step 3 skipped).

**AILANG's own connector (the executor OAuth service, dev):**
- Lazy-auth, authorization-server and security-schemes all PASS.
- 110 FAILs, all titles and hints on the 22 `mcp_tools/` docs tools.
- Allowlisted to `sunholo.com`.
- Before prod, a Terraform precondition requires a per-user rate limit, and the page needs a prod Firebase config (GE design lines 118-124).

### 2.3 Google Gemini Enterprise

**Wired today (dev only):**
- GE custom MCP connector `ailang-mcp-dev` → `ailang-dev-executor-mcp` (Cloud Run, IAM-only).
- GE sends a Google-signed ID token, as `X-Serverless-Authorization`, only to default `*.run.app` URLs.
- Exposed: 28 tools, namely `ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_eval`, `ailang_builtins`, the docs tools and `submit_feedback`.
- Execution limits: IO-only restricted mode, 60 s, 64 KB.
- Org policy (`terraform-gemini-enterprise/`, applied from a laptop):
  - It disables the block on custom MCP connectors.
  - It sets the egress allowlist to that one host only.

**Not done:**
- No user identity reaches the service (IAM-only).
- **OAuth via GE is blocked by the design of `mcp_oauth`.** It only accepts CIMD and `none` client auth, while GE expects a client id and secret.
- The OAuth service is not on the GE egress allowlist.
- No prod executor exists: the `test`/`prod` tfvars enable neither service.
- **docparse is not connected to GE at all.**
- A2A:
  - serve-api's A2A cannot serve GE: no `message/stream`, no `protocolVersion`.
  - GE design P0-c..e (what auth GE sends to an A2A agent; the only path observed so far ran unauthenticated), P2 (gateway) and P3 (registration) have not started.
  - Decisions D2/D3/D5/D6 are open.
- The Agent Gateway / Agent Registry route is deferred.

**In flight:** nothing since 10-14. The design `m-ailang-executor-gemini-enterprise.md` estimates about 5 days remaining.

### 2.4 Google Search apps / consumer Gemini / Agentspace / Vertex AI Search

- **Planned only**: `m-google-connected-apps-distribution.md` (ailang, 10-29). It proposes a docparse pilot, then the executor, and is awaiting approval; no feasibility result has been recorded.
- No Discovery Engine, Agentspace data-store, Vertex AI Search app or Agent Registry resources exist in any repo. The "agent registry" hits in multivac are AILANG's own coordinator config.
- Gemini appears in docparse only as the OCR/AI model (Vertex, `GOOGLE_CLOUD_LOCATION=eu`).
  - Prod still calls the **global** Vertex host, because it runs v0.56.1. The fix `dc5a7b5b8` is in v0.57.0.
  - docparse `146bfc7` (PR #256) stopped `/api/v1/capabilities` from claiming EU residency.

---

## 3. Open bugs and issues that block or degrade integrations

### Issues the brief named

| # | Status (verified 2026-11-05) | Effect on integrations |
|---|---|---|
| ailang **#1597** (static security headers, framable consent page) | **Closed 10-09.** Fixed by `2f7058b7b` (in v0.53.1). Live: `X-Frame-Options: DENY` + `frame-ancestors 'none'` on docparse static pages | None remaining. The docparse `antiframe.js` comment and its CHANGELOG v0.33.1 rationale are stale |
| ailang **#1609** (hyphenated `_headers`) | **Closed 10-09**, same sprint `2f7058b7b` | None |
| ailang **#1631** (MCP `int` params bind as float) | **OPEN, P1.** Code unchanged on dev (`CallPreserveFloats` at `mcp.go:456`, `a2a.go:226`, `mcp_ui_serve.go:103`). Fix **PR #2112** is open, `mergeable: CONFLICTING`, `mergeStateStatus: DIRTY`, checks skipped | Every AILANG MCP/A2A tool with an `int` param is broken. Parse ships `count: float` and string `maxChars` as workarounds. Design `m-serve-api-typed-arg-binder` (P0, target v0.57.1) missed the release |
| ailang **#1648** (REST: whole-number `float` params fail) | **Closed 10-08 as a duplicate of #1631, but not fixed.** The typed-arg-binder design reproduces it at v0.57.0 (table row "float param, JSON 2 → REST ❌") | REST callers sending `2` for a float param fail. Fixed only when #2112 lands |
| **#1624** | **Does not resolve** in ailang, ailang-packages or docparse (GraphQL: no issue or PR) | Possibly a typo; check the intended number |
| ailang **#1618** (mcp_files friction batch, incl. per-route/suffix CORS) | **OPEN, P2.** Re-verified at v0.57.0 on 11-03 (`07b6ac11a`). Planner PR #2103 open | docparse runs a global `--cors` (`Access-Control-Allow-Origin: *`, seen live on `/oauth/authorize`) because per-route CORS for `/uploads` does not exist |
| **X-Frame-Options** | Fixed (see #1597). **Residual:** the `mcp check` framing probe WARNs "inconclusive" on every CIMD-only AS, because it uses a non-URL dummy `client_id` (`internal/mcpcheck/framing.go:31`) | A false warning on every adopter. Fix: probe with a real CIMD URL or a self-served dummy metadata doc, and follow the 302 to the login page. That page *is* protected (live) |
| **Arg binding** | See #1631 / #1648 / PR #2112 | — |

### Other blockers and degraders found in this audit

1. **A2A is pre-0.2 and ungated** (serve-api) or **static and non-conformant** (docparse). This blocks every A2A integration, GE first among them. There is no issue; the GE design records it as V3/V4.
2. **`mcp_oauth` has no confidential-client or DCR path.** This blocks GE OAuth connectors and any client that cannot do CIMD. There is no issue.
3. **The token verifier is boolean.** Tools cannot get the subject or scopes from the gate, so per-user rate limiting has to re-derive identity. The executor's per-user rate limit is the prod blocker. There is no issue.
4. **The GE egress allowlist** holds only the IAM dev executor host (`terraform-gemini-enterprise/variables.tf:7-19`).
5. **docparse `serverInfo`** is `ailang-api 0.8.1` (G13). serve-api has no flag or annotation to set it, so this is an AILANG fix.
6. **The listed surface exposes `ailang://meta/modules`** (the module and export list) on `/mcp/connect/`. This is a data-minimisation and review risk, and `--no-introspection` does not appear to cover MCP resources. It should be verified and filed.
7. **`/.well-known/openai-apps-challenge`** returns 404 (OpenAI domain verification).
8. **The MCP Registry entry** `io.github.sunholo-data/parse` (v0.14.0 latest) advertises `/mcp/`, the API-key surface, rather than `/mcp/connect/`.
9. **docparse privacy policy** (`ailang-parse docs/privacy.html`, "Last updated 6 Oct 2026") contradicts what has shipped:
   - "Uploaded files are not stored": mcp_files keeps uploads with 30-minute reuse, and the bucket lifecycle is 24 h.
   - "We do not retain AI inputs or outputs": the v0.43.0 AI-parse cache keeps them for 30 minutes.
   - "EU endpoint preferred": prod calls the global Vertex host.
   - The v0.37.0 changelog claims a privacy update that is on no branch.
   - This is a **directory-review blocker** (Anthropic Policy 3A; OpenAI `privacyPolicyURL` requirements).
10. **Stale public copy.**
    - connect.html and mcp.html list 7 tools, but there are 9 plus `submit_feedback`.
    - The SKILL.md in docparse-skill does not cover uploads or paging.
    - The AILANG guide `mcp-connectors.md` §5 says mcp_files is unreleased.
    - The `m-serveapi-directory-ready.md` status line says Phase B is "gated on spike S1", yet `securitySchemes` and `mcp/www_authenticate` shipped on 10-07 (`163e32e75`).
11. **The executor docs tools lack titles and hints** (110 `mcp check` FAILs). This blocks any directory listing of AILANG's own connector.
12. **Shipped docparse designs are still in `planned/`**: `mcp_oauth_adoption.md`, `mcp_oauth_same_origin_login.md`, `mcp_connector_usable.md`.
13. **docparse prod pins AILANG v0.56.1.** Bumping to v0.57.1 brings the Vertex regional host. Bumping after #2112 lands brings typed args.

Open GitHub search (`gh issue list -R sunholo-data/ailang --search "mcp OR a2a OR oauth OR gemini" --state open`): of the integration-relevant results, only #1631 and #1618 are open. There are no open issues for A2A, GE OAuth or `serverInfo`. docparse, docparse-skill, ailang-parse and ailang-multivac have no open integration issues or PRs.

---

## 4. Reusable layer (AILANG) vs docparse-specific code

### 4.1 Reusable today (used by both docparse and the executor)

| Layer | Where | Adopters |
|---|---|---|
| Two MCP surfaces from one module; `--mcp-connect-only` | serve-api (`internal/apiserver/mcp*.go`, `serveapi/protocol/`) | docparse (both surfaces), executor IAM (`/mcp/`), executor OAuth (`--mcp-connect-only`) |
| `@mcp_title`/`@mcp_hints`/`@mcp_auth`/`@mcp_secret`/`@mcp_agent_only`/`@mcp_token_verifier` | serve-api | docparse, executor |
| `@mcp_file`, `@mcp_ui_resource`/`@mcp_ui`/`@mcp_app_only` | serve-api | docparse |
| OAuth 2.1 AS | `sunholo/mcp_oauth` 0.1.2 (hooks for authenticate/mint/store/revoke) | docparse (Firestore, Firebase, `dp_` keys), executor (`executor_oauth/`) |
| Upload tokens, fileRefs, widget, `fetchFileParam` | `sunholo/mcp_files` 0.1.4 (storage hooks) | docparse only |
| Static security headers, `--net-max-bytes`, `Net[scope=public]` | serve-api / runtime | docparse, executor |
| Directory linter | `ailang mcp check` | run by hand; not in docparse CI |

### 4.2 docparse-specific code that ought to be reusable

1. **The A2A card**, hand-written in `api_server.ail:2145-2196`. serve-api's generated card is unusable: it is pre-0.2, has the wrong version, and lists every export as a skill. So each service will hand-write a card until serve-api produces a configurable A2A 0.3 card, gated like `/mcp/connect/`.
2. **The device flow** (`device_auth.ail`, `/api/v1/auth/device*`, approve page) and the `dp_` key model. This is the agent-surface auth pattern, and it is not packaged. The executor has no equivalent for headless agents.
3. **The capabilities manifest** (`capabilities.ail`) is a hand-written discovery document. It does not mention `/mcp/connect/` or OAuth, and its A2A pointer is the static card.
4. **Plugin and marketplace manifests** (docparse-skill) are hand-maintained per vendor. The L5 template in `m-serveapi-directory-ready` was never built, so the executor and `ailang_bootstrap` will each repeat this.
5. **Branding**: `serverInfo` is not settable. Widget strings mention "Claude" by name in `mcp_files.ail`, so the package widget is host-specific in practice.

### 4.3 Gaps in the reusable layer itself

- **A2A has three implementations**, none of them complete: serve-api (Go, pre-0.2), the docparse static card (AILANG), and the resident agent (Node, 0.3). The GE design P2 adds a fourth (a gateway) unless serve-api's A2A is brought to 0.3 first.
- **No Google identity in-app.** There is no ID-token or JWKS verifier package. Cloud Run IAM is the only Google gate, and it carries no user identity to the tool.
- **No confidential-client or DCR support in `mcp_oauth`.** GE needs it; Claude and ChatGPT are covered by CIMD.
- **Verifier → tool identity plumbing.** The gate answers yes or no only. Tools re-read `Authorization` from `_headers` and re-validate it, so the check runs twice.
- **The arg binder is split by surface** (#1631, PR #2112). Every surface (MCP, A2A, MCP UI, REST) is affected.
- **`mcp check` coverage gaps**: the framing probe against CIMD-only servers, `serverInfo` name, `openai-apps-challenge`, extra listed-surface resources, and A2A card conformance.
- **Docs drift**: the AILANG guide §5, the directory-ready status line, docparse `planned/` docs, connect.html, mcp.html and SKILL.md all lag the code by about a month.

---

## Appendix A: live check transcripts (2026-11-05)

### A.1 `ailang mcp check --target both https://docparse.ailang.sunholo.com/mcp/connect/` (v0.57.1, rc=0)

```
  PASS  annotations                  all 10 tools have a title and hints [A7]
  PASS  credentials                  no credential-shaped parameters [A2,O1]
  PASS  zero-arg                     no tool requires a placeholder argument
  PASS  lazy-auth                    3 gated tool(s) answer 401 with resource metadata naming https://docparse.ailang.sunholo.com [A3]
  PASS  authorization-server         https://docparse.ailang.sunholo.com advertises S256 and a client registration method [A6,O6]
  WARN  authorization-framing        authorization framing probe inconclusive: final HTTP 400, Content-Type "text/plain; charset=utf-8" (requires 2xx HTML) [RFC 9700 §4.16]
  PASS  file-params                  3 tool(s) declare openai/fileParams with the four-property file schema [O9]
  PASS  security-schemes             3 gated tool(s) declare an oauth2 security scheme [O5,O10]
No failures.
```

"3 gated tools" is how the probe is designed: it never calls write tools that have no required params (`mcpcheck.go:7-10`). `tools/list` shows 7 `oauth2` tools.

### A.2 docparse surfaces

**`tools/list`, `/mcp/connect/` (10 tools):**

| Tool | `securitySchemes` | Other `_meta` |
|---|---|---|
| createUpload | oauth2 | |
| editDocument | oauth2 | fileParams |
| mcpAccount | oauth2 | |
| mcpConvert | oauth2 | fileParams |
| mcpEstimate | noauth | |
| mcpFormats | noauth | |
| mcpParse | oauth2 | fileParams |
| submit_feedback | noauth | |
| uploadDocument | oauth2 | ui |
| uploadViaHost | noauth | ui, `visibility:["app"]` |

**`tools/list`, `/mcp/` (12 tools):** the same tools plus `mcpAuth` and `mcpAuthPoll`. No `securitySchemes`. `apiKey` is in the schemas, and `createUpload.count` is typed `number`.

**`initialize`, both surfaces:**
- `protocolVersion "2025-11-25"`
- `serverInfo {"name":"ailang-api","version":"0.8.1"}`
- capabilities: logging, resources, tools

**`resources/list`:** `ailang://meta/modules` and `ui://docparse/upload` (`text/html;profile=mcp-app`, `connectDomains:["https://docparse.ailang.sunholo.com"]`).

**Unauthenticated gated call** (`tools/call mcpParse {}` on `/connect/`):
- HTTP 401
- `www-authenticate: Bearer resource_metadata=".../.well-known/oauth-protected-resource/mcp/connect/"`
- body: `{"result":{"_meta":{"mcp/www_authenticate":[...]},"isError":true,...}}`

**The same call on `/mcp/`:** HTTP 200 with tool error `AUTH_REQUIRED` and the suggested fix "Connect through the OAuth connector, or send a dp_ API key …".

**Well-known and other endpoints:**

| Request | Result |
|---|---|
| `GET /.well-known/oauth-authorization-server` | 200. `authorization_code` + `refresh_token`, S256, `token_endpoint_auth_methods_supported:["none"]`, `client_id_metadata_document_supported:true`, `authorization_response_iss_parameter_supported:true`; **no `registration_endpoint`** |
| `GET /.well-known/oauth-protected-resource` and the `/mcp/connect/` path-inserted form | 200. `resource: https://docparse.ailang.sunholo.com/mcp/connect/` |
| `GET /oauth/authorize` with Claude's CIMD `client_id`, a valid S256 challenge and `resource` | 302 to `/oauth-login.html?handle=…`, which carries `X-Frame-Options: DENY`, `CSP: frame-ancestors 'none'` and `nosniff`. The 302 itself carries `Access-Control-Allow-Origin: *` (global `--cors`) |
| `GET /.well-known/agent.json` | 200. Hand-written card: `"version":"0.9.0"`, `"protocol":"a2a/0.3"`, skills parse/formats/unstructured, `authentication.schemes:["apiKey"]` (in body) |
| `GET /.well-known/agent-card.json` | 404 |
| `POST /a2a/` | 404 |
| `/.well-known/openid-configuration` | 404 |
| `/.well-known/openai-apps-challenge` | **404** |
| `GET /api/v1/health` | `{"version":"0.9.0","ailang_commit":"v0.56.1",...}` |
| MCP Registry `io.github.sunholo-data/parse` | latest v0.14.0, `remotes` → `https://docparse.ailang.sunholo.com/mcp/` |

### A.3 `ailang mcp check --target both` on the executor OAuth dev service

```
FAIL x110  annotations (no title / hints not explicit) on 22 tools:
  ailang_versions benchmark_run benchmarks_compare benchmarks_for_model benchmarks_list
  changelog_for_version design_doc design_docs_list docs_nav docs_search effects_catalog
  example_for_concept example_get examples_list install_guide limitations_list
  onboarding_guide prompt_get roadmap stdlib_module stdlib_modules stdlib_search
PASS  credentials, zero-arg
PASS  lazy-auth            25 gated tool(s) answer 401 with resource metadata
PASS  authorization-server S256 + client registration method
WARN  authorization-framing inconclusive (same dummy-client_id probe issue)
PASS  file-params (none), security-schemes (25 gated)
Not ready to submit.
```

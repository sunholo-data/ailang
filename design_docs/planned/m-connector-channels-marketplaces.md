# M-CONNECTOR-CHANNELS: the executor and Parse on every assistant, documented and listed

**Status**: Planned. Action plan, awaiting Mark's decisions D1–D7. Phase P0 (audit) needs no decision and can start.
**Target**: v0.59.x (AILANG side); docparse v0.47.x; multivac config. Listings follow on vendor timelines.
**Priority**: P1. Both demos are live; what's left is reach, proof (screenshots) and the listings.
**Estimated**: ~9 engineering days plus Mark's console and portal time (§Timeline). Vendor review time is not ours to estimate.
**Dependencies**: `sunholo/mcp_oauth` 0.2.0 (published 2026-11-05); M-AILANG-EXECUTOR-GE P2 (A2A gateway) for the A2A rows; Google's Connected Apps preview admission.
**Quorum triggers fired**: **#1** (design-freeze items D1–D7) and **#4** (every load-bearing premise about a vendor program is external). Quorum 2026-11-10: round 1 blocked (package API unlogged; A2A seam missing), round 2 blocked (no prod-release step; D3 ahead of M4 eligibility) with gemini passing. All objections fixed in this revision (P1b, D3, §3.1, V17–V22); per the re-quorum-once rule this goes to Mark rather than a third round.

## Problem Statement

Two AILANG services run in production as MCP servers:

| Service | Listed URL | Release |
|---|---|---|
| **AILANG Parse** (document parse/convert/generate) | `https://docparse.ailang.sunholo.com/mcp/connect/` | docparse v0.46.0, AILANG v0.57.4, 2026-11-10 |
| **AILANG executor** (check and run AILANG code in a sandbox, plus the docs tools) | `https://executor.ailang.sunholo.com/mcp/connect/` | AILANG v0.57.4, custom domain live 2026-11-10 |

Both pass `ailang mcp check` for the Claude and ChatGPT directories, except the executor's tool annotations,
which are fixed on `dev` (`4f9dd4835`) and reach prod with the next AILANG release and image-tag bump.

But "an AILANG service works on platform X" is proven for exactly one cell: **Parse in claude.ai**. Every
other assistant is either untested, blocked on an OAuth client model we have built but not switched on, or
blocked on A2A. None of the official marketplaces lists either service. Parse is in the open MCP Registry
(`io.github.sunholo-data/parse`); the executor is not. The public docs say what *should* work, with no
screenshot showing that it does.

**Impact.** The pitch for AILANG as a language for services is "write it once, it appears in every
assistant". Today we can show that in one assistant. Each marketplace has its own eligibility gates (org
verification, publisher programs, A2A, test accounts, demo videos) that take calendar time, so starting late
costs weeks.

## Goals

**Primary goal:** each of the two services is usable, with a screenshot proving it, in every channel marked
*in scope* in §1, and has a submitted listing in every marketplace Mark selects in D1.

**Success metrics:**
1. The §1 matrix has a ✅ with a dated screenshot in `docs/static/img/connectors/` for every in-scope cell.
2. One public page per service (`hosted-executor.md`, a new Parse equivalent) has a "Use it in …" section
   per channel, each with its screenshots.
3. Every marketplace chosen in D1 has a submission recorded (date, version, status) in §5's tracker.
4. `ailang mcp check --target both` passes on both prod URLs; a new `--target copilot` (P1) passes too.
5. No cell is marked ✅ on the strength of a vendor doc alone. A cell needs a live run.

## 1. The channel matrix

Status today (2026-11-10). ✅ = observed working live. 🔧 = we have to build or switch something on.
👤 = needs Mark at a console. ⛔ = blocked on something outside us.

| # | Channel | How the user gets it | Client model | Parse | Executor |
|---|---|---|---|---|---|
| C1 | **claude.ai** (web, desktop, mobile) | Settings → Connectors → custom connector, or the directory | CIMD | ✅ (directory-ready since v0.52.3) | 👤 untested; should work (same stack) |
| C2 | **Claude Code** | `claude mcp add --transport http …` | CIMD | ✅ (this repo's sessions use the prod URL) | ✅ dev and **prod** (`ailang-executor` → `executor.ailang.sunholo.com`, signed in 2026-11-10: `whoami` + `ailang_run` OK) |
| C3 | **ChatGPT** (chatgpt.com plugins, then directory) | Plugins → Add custom MCP server (OAuth, CIMD) | CIMD + `iss` | 👤 not yet tried | ✅ 2026-11-11: `chatgpt.com/oauth/client.json` signed in, `openai-mcp/1.0.0` tool calls `executor_auth ok` |
| C4 | **Codex / ChatGPT desktop Work** | Desktop Settings → Plugins → MCPs → Add MCP server | CIMD (`chatgpt.com/oauth/codex/client.json`) | 👤 | ⚠️ 2026-11-11: sign-in OK (token issued to `codex-mcp-client/0.167`), but the Work runner (`codex-mcp-client/0.0.0`, same machine) sends no token → 401. Client-side |
| C5 | **Microsoft 365 Copilot** (declarative agent + MCP plugin) | Agents Toolkit ≥ 6.12.0, sideload; then Marketplace | static client or DCR | 🔧 client model off | 🔧 client model off |
| C6 | **Copilot Studio** (agent tool) | MCP onboarding wizard | DCR (discovery) or manual client | 🔧 | 🔧 |
| C7 | **GitHub Copilot / VS Code** | `MCP: Add Server`, or from the MCP Registry | remote MCP + OAuth | 👤 untested | 👤 untested; not in the Registry |
| C8 | **Gemini Enterprise, MCP data store** | Admin adds the URL **inside the app**; users reach it only through an **Agent Designer agent** (the default assistant does not list custom MCP tools) | IAM, or pre-registered client | 🔧 no confidential client in prod | ✅ IAM-only (dev, 2026-10-14); ✅ OAuth sign-in on dev 2026-11-11 (PKCE + Basic, token issued); web-app Connect needed `mcp_oauth` 0.2.1 (768-char `state`); ⏳ chat tool call blocked by GE's daily agent-creation quota (Standard = 1/licensed user/day, pooled; deletes do not refund) — retry after reset |
| C9 | **Gemini Enterprise, A2A agent** | Agent Registry; Cloud Marketplace | A2A v0.3 | ⛔ out of scope (D7) | 🔧 serve-api A2A is pre-0.2. **Registration is the same `CreateAgent` call, so it shares GE's daily agent-creation quota** (measured 2026-11-11: 429 RESOURCE_EXHAUSTED); not a shortcut past the MCP route |
| C10 | **Gemini app + Search** (Connected Apps, preview) | Google's console after admission | CIMD, then DCR; stable refresh; revoke | 🔧 + ⛔ preview admission | 🔧 + ⛔ preview admission |

## 2. The marketplaces

What each one demands before it accepts a submission. Sources: `integrations-review-external-2026-11-05.md`
§§1.3, 2.3, 3.4, 4.3 (vendor pages fetched 2026-11-05) and Microsoft Learn (read 2026-11-10, V8–V11).

| ID | Marketplace | Eligibility gates | Who | Cost | Fits |
|---|---|---|---|---|---|
| M1 | **Anthropic connector directory** (`claude.ai/directory/manage`) | Paid Claude plan (Owner on Team/Ent); ownership of every endpoint (§3F); title + hint on every tool; populated test account; ≥ 3 working prompts; public docs by publish date; 3–5 PNG screenshots ≥ 1000 px if it has an MCP App; seven acknowledgments | Mark | plan only | Parse, executor |
| M2 | **OpenAI plugin directory** (`platform.openai.com/plugins`, ZIP) | Org verified (individual or business, enforced); Owner or Apps Management Write; **project with global, not EU, data residency**; domain token at `/.well-known/openai-apps-challenge`; 5 positive + 3 negative cases; **demo video**; MFA-free reviewer account; four listing URLs; no pricing or upsell in tool output; strip IDs and timestamps from results | Mark | none found | Parse, executor |
| M3 | **Official MCP Registry** (feeds VS Code / GitHub Copilot) | `server.json` with a `remotes` entry; `io.github.<org>/` namespace via `mcp-publisher` GitHub login or OIDC in CI | agent (CI) | free | Parse ✅ listed; executor to add |
| M4 | **Microsoft Marketplace** (Partner Center → Marketplace offers → *Microsoft 365 and Copilot*) | Publisher account enrolled in the Microsoft 365 and Copilot program; an offer type that accepts a declarative agent + MCP plugin package (**must be confirmed first**, V10); Marketplace certification policies; Microsoft 365 store validation guidelines for agents; Responsible AI checks at manifest validation; reviewer test instructions; optional App Compliance certification | Mark (enrolment), agent (package) | none found | Parse first |
| M5 | **Google Connected Apps** (Gemini app + Search) | Preview admission; Connected Apps ToS accepted by an admin; description < 30 words; 512×512 icon; ≥ 3 capabilities; ≥ 3 prompts (2 starting `@App`); seeded no-2FA test account; data-deletion help docs; `/.well-known/google-connected-apps` JSON; tools frozen per version; `destructiveHint` or console HITL on writes | Mark (admission), agent (well-known, docs) | none found | Parse first |
| M6 | **Google Cloud Marketplace, AI agents** (→ GE Agent Gallery) | Vendor onboarding + Project Info Form; **A2A**; "Model Garden as a default configuration"; Partner Procurement API + Pub/Sub integration; product, pricing, technical and card reviews; 3% revenue share | Mark + agent | 3% rev share | Recommend defer (D4) |

## High-Impact Decisions

| Decision | Why it matters | Who | When | Cost to change |
|---|---|---|---|---|
| D1: Which service goes to which marketplace | Each listing is a support commitment and a review cycle | Mark | before P5 | low before submission, high after (OpenAI fixes the origin per listing) |
| D2: Switch on DCR + confidential clients + revocation in **prod** for both services | Unlocks C5, C6, C8-OAuth and C10, but DCR is an unauthenticated write endpoint | Mark | after P2 measurements | low (a `ClientConfig` field) |
| D3: Enrol a Microsoft Partner Center publisher (business) in the Microsoft 365 and Copilot program | M4's only entry point; business verification takes days; Microsoft quotes 4–6 weeks per submission | Mark | after P2 shows C5 works | — |
| D4: Defer Google Cloud Marketplace (M6) | Needs A2A, a Model Garden default model and procurement integration: a product decision, not a listing | Mark | now | low |
| D5: Which reviewer accounts | Every directory wants a seeded, MFA-free account; Google's sign-in on the executor has none | Mark | before P5 | medium |
| D6: Where screenshots live, who takes them | They are the proof for §1 and the assets for M1/M5 | Mark | before P2 | low |
| D7: A2A for the executor only | Parse's value is inline tools; delegation is the executor's (GE design P2) | Mark | before P3 | low |

### Design Freeze

Recommendations follow each item. Unticked items block only the phase named.

- [ ] **D1 (blocks P5).** Recommended: Parse → M1, M2, M3 (done), M4, M5. Executor → M1, M2, M3. Parse is
  the commercial product; the executor is a developer tool, and the Microsoft and Google consumer stores are
  the wrong audience for it.
- [ ] **D2 (blocks P2 in prod; dev needs no decision).** Recommended: on, in this order: dev now; prod after
  P2 shows each client registers and signs in. `dcrRedirectHosts` set to the hosts measured in P2 (expected:
  `teams.microsoft.com`, `vscode.dev`, Copilot Studio's callback host, Google's), never `[]`.
- [ ] **D3 (blocks M4).** Recommended: **don't enrol yet.** Microsoft's own pages leave M4 eligibility open
  for this package type ("Not every plugin package is eligible", V10; offer types listed only inside the
  enrolled portal, V22). Enrol after P2 shows C5 working on dev, and only if D1 keeps M4. Enrolment is the
  only way to see the offer-type list, so the first enrolled step is to confirm an offer type accepts a
  declarative agent with an MCP plugin; if none does, M4 is dropped and Copilot users get the
  org-catalog/sideload instructions instead (§Risks).
- [ ] **D4.** Recommended: defer M6. Use C8 (admin-added MCP) for Gemini Enterprise customers, and C9 for
  A2A without a Marketplace listing.
- [ ] **D5 (blocks P5).** Recommended: Parse keeps its email/password reviewer account (gap G1 closed:
  `static/oauth-login.js` offers Email/Password). The executor gets the same: add the Email/Password provider
  to its Firebase sign-in, for console-created accounts only, as Parse did.
- [ ] **D6 (blocks P2).** Recommended: `docs/static/img/connectors/<service>/<channel>-<NN>-<slug>.png`,
  PNG ≥ 1000 px wide, Mark captures, the agent names, crops and wires them in. §4 lists every shot.
- [ ] **D7 (blocks P3).** Recommended: executor only.

## Solution Design

### Overview

Three tracks run in parallel, joined by one rule: a cell turns ✅ only when Mark has watched it work and
taken the screenshot.

- **Works:** switch on the client models, add A2A v0.3, and fix whatever P0 finds.
- **Shows:** one public page per service, a section per channel, screenshots from §4.
- **Listed:** a submission packet per marketplace, built from what Works and Shows produced.

### Implementation Plan

**P0: live audit (agent, 0.5 day, no decision needed).**
1. Call every tool on both `/mcp/connect/` URLs as a signed-in client (Claude Code already holds tokens for
   both), with valid arguments. Record pass/fail per tool. This is Anthropic's "every tool must return a
   successful response" and OpenAI's test-case material.
2. Re-check the Parse submission kit's gaps G1–G4 against v0.46.0. G1 is closed (email/password). G2–G4
   (`editDocument` over MCP, user documents in, generated files out) need a live answer, because
   `sunholo/mcp_files` landed since the kit was written.
3. Review every Parse tool's hints against OpenAI's rule that persisting artifacts or uploading files means
   `readOnlyHint: false`.
4. Confirm the data-minimisation rule: no request, trace or session IDs or timestamps in tool results.
5. Output: a table appended to this doc, and fix tickets for anything red.

**P1: client models on, dev first (agent, 2 days).**
1. Executor (`executor_oauth/oauth.ail`, now on 0.2.0): a `ClientConfig` with `confidential`, `dcr` and
   `revocation` on; move to the 0.2 entry points; mount `POST /oauth/register` and `POST /oauth/revoke`;
   an operator script (`executor_oauth/create_client.sh`) wrapping `createClient` for GE and Copilot static
   clients. `test_oauth.sh` grows the GE, Copilot and DCR cases from the package's `e2e_client_models.sh`.
2. Parse: the same, in docparse. Parse also serves a widget, so add `--cors-origin` for Copilot's widget host
   (`{sha256(docparse.ailang.sunholo.com)}.widget-renderer.usercontent.microsoft.com`).
3. `ailang mcp check --target copilot`: checks Streamable HTTP, a `registration_endpoint` or a documented
   static-client path, and (when the server has widgets) that the Copilot widget origin passes CORS.
4. Dev only. Prod waits for D2.

**P1 status (2026-11-10).** Executor done: `f3daa15c2` (env switches `OAUTH_CONFIDENTIAL_CLIENTS`,
`OAUTH_DCR` + `OAUTH_DCR_REDIRECT_HOSTS`, `OAUTH_REVOCATION`, all off by default; DCR fails closed
without hosts; operator command `createClientCli`; consent page names registered clients and flags DCR
ones unverified). `test_oauth.sh` 43/43. Dev switch-on: ailang-multivac #18, merged and **live on the dev executor 2026-11-10** (metadata advertises registration + revocation + client-secret methods; a `teams.microsoft.com` registration → 201, `evil.example` → 400; `mcp check --target both` no failures, all 29 tools annotated) (`terraform test` 50/50;
prod plan adds the four env vars empty). Finding: the live prod executor service runs
`executor-mcp:latest`, not the pinned tag; any prod apply re-pins it. Parse: docparse #267 (same switches; operator command in `services/mcp_oauth_operator.ail` so
`mcp_oauth.ail` stays output-free per its lint; `test_mcp_oauth.sh` 104 PASS, fail-closed guard
mutation-tested; a client-less token request is now RFC 6749 `401 invalid_client`). Copilot's widget
origin needs no change: Parse already runs `--cors`. Dev switch-on: ailang-multivac #19, merged and **live on dev Parse 2026-11-10** (registration + revocation advertised; `teams.microsoft.com` registration → 201, `evil.example` → 400; `mcp check --target both` no failures) **Next:** `mcp check --target copilot`.

**P1b: release to prod (agent drives, Mark runs the promotes; ~0.5 day of work, ~1 day elapsed).** Nothing
in prod changes without this, and success metric 4 depends on it.
1. AILANG release carrying the docs-tool annotations (`4f9dd4835`), the 0.2.0 lock (`7af2d7195`) and P1's
   executor OAuth change; tag → `ailang-core-release` test deploy + smoke gate.
2. Mark: `YES=1 scripts/release.sh promote core vX.Y.Z` (the classifier blocks the agent on prod promotes).
3. Agent: multivac PR moving `executor_mcp_image_tag` in `environments/prod/terraform.tfvars` from
   `v0.57.4` to the new tag; Mark merges and pushes `dev` → `test` → `prod`.
4. docparse release with its P1 change, promoted the same way.
5. Gate: `ailang mcp check --target both` passes on both prod URLs. Client models stay **off in prod**
   (`ClientConfig` fields false) until D2; the code ships dark.

**P1b status (2026-11-10): done.** AILANG v0.60.0 (tag on `2f3ebb81f`; a macOS PTY flake in
`TestWorkerCLIProcessExitLifecycle` held the release gate until a job re-run) promoted to prod;
docparse v0.47.0 promoted to prod; ailang-multivac #20 pins `executor-mcp:v0.60.0` and is applied on
`test` and `prod` (98456f1). Verified on prod: `mcp.ailang.sunholo.com` serves 0.60.0 as
`ailang-docs`; the executor reports `executor (AILANG Executor) 0.60.0`, all 29 tools annotated,
`ailang_check` proves contracts (`double` verified, `wrong` counterexample `x = 0`), no request
directory in results; Parse 0.47.0 passes `--target both`. Client-model switches present and empty
on both prod services, so `--target copilot` FAILs registration there until D2.

**P2: Mark at the consoles, with screenshots (Mark ~1 day, agent on call).** One session per channel, against
dev for anything new: C1, C3, C4, C5, C6, C7, C8 (OAuth). This also takes the GE measurements P0-1…P0-5 from
`m-mcp-oauth-client-models.md`. Each session yields its shots (§4) and a row in the §1 matrix.

**P3: A2A v0.3 for the executor (agent, ~3 days once its location is decided).** **Location decided 2026-11-11 (Mark): serve-api, model Gemini on Vertex; in progress under [m-ailang-executor-ge-a2a-sprint-plan.md](v0_55_0/m-ailang-executor-ge-a2a-sprint-plan.md).** This is
M-AILANG-EXECUTOR-GE P2, and **where the gateway lives is that doc's open D2** (coordinator route,
recommended there, or a serve-api service). This plan does not re-decide it; it records the seam either
choice has to cross (§3.1). Deliverable: agent card with `protocolVersion: "0.3.0"`, `message/send`,
`message/stream`, `tasks/get`, every skill behind an authenticated caller; registration in GE's Agent
Registry; the C9 screenshots.

**P4: docs (agent 1.5 days, after P2).** A Parse page like `hosted-executor.md`. Both pages get a
"Use it in …" section per channel with its shots. `mcp-connectors.md` §1 matrix turns into links.

**P5: listings (Mark, agent prepares the packets).** Per D1, in this order, cheapest first: M3 executor
(CI), M1 both, M2 both, M5 Parse, M4 Parse. Each packet: copy, prompts, test cases, reviewer account,
screenshots, demo video (M2), privacy and terms links, the domain proof.

### 3.1 Conflict surface: A2A and the Bearer gate

| Position | What lives there today | Evidence | What P3 must do |
|---|---|---|---|
| `/.well-known/agent.json`, `/a2a/` | Mounted by serve-api when `--a2a` is set, wrapped only in CORS, **not** in the Bearer gate | `internal/apiserver/server.go:545-546` | Whatever serves A2A must authenticate every skill call |
| The executor's public service | Runs `serve-api --mcp-connect-only` | `docs/docs/guides/serve-api.md` (connect-only section); `executor_oauth/` | — |
| `--mcp-connect-only` + `--a2a` | **Refused at startup**: "A2A skills are not behind the Bearer gate" | `internal/apiserver/mcp_auth.go:188` | If D2(b) (serve-api) is chosen, either keep A2A on a **separate** service, or put `/a2a/` behind `gatedHandler`'s `BearerGate` (`mcp_auth.go:237`) and only then lift the refusal. Never lift it alone |
| The agent card | Must stay **outside** the gate: registries and GE fetch it before any sign-in | A2A discovery | Gate `/a2a/` only, never `/.well-known/agent.json` or `agent-card.json` |
| A2A methods | `tasks/send`, `tasks/get` only | `internal/apiserver/a2a.go:133-135` | Add `message/send`, `message/stream`; GE never polls |
| Card version | `--server-info` already feeds the A2A card version | `serve-api.md` `--server-info` row | Reuse; no second version source |

Programs that must keep working: the executor's `/mcp/connect/` (`executor_oauth/test_oauth.sh`), docparse's
`/mcp/connect/` (it does not use `--a2a`), and `TestA2ADisabledByDefault` / `TestA2AEnabledRegistersRoutes`
(`internal/apiserver/routes_annotations_test.go:462,477`).

### Files to Modify/Create

- `executor_oauth/oauth.ail`: `ClientConfig`, 0.2 entry points, register and revoke routes (~120 LOC).
- `executor_oauth/create_client.sh`: operator script for static clients (~40 LOC).
- `executor_oauth/test_oauth.sh`: GE, Copilot and DCR cases (~120 LOC).
- `internal/mcpcheck/`: the `copilot` target (~150 LOC + tests).
- `internal/apiserver/a2a.go`: `message/send`, `message/stream`, v0.3 card fields (only if GE-design D2 picks serve-api; ~300 LOC + tests).
- `internal/apiserver/server.go`, `internal/apiserver/mcp_auth.go`: gate `/a2a/` behind the Bearer gate before the connect-only refusal is lifted (same condition).
- `internal/coordinator/`: the `/a2a` route (only if GE-design D2 picks the coordinator).
- `docs/docs/guides/hosted-executor.md`: per-channel sections and screenshots.
- `docs/docs/guides/ailang-parse.md`: new public page for Parse as a connector.
- `docs/docs/guides/mcp-connectors.md`: matrix links and the Copilot check.
- `docs/static/img/connectors/`: screenshots.
- `server.json` for the executor in the MCP Registry, published from the release pipeline (~30 LOC).
- `design_docs/planned/v0_55_0/m-ailang-executor-gemini-enterprise.md`: P2 (A2A) is executed under it.
- docparse: the same OAuth change, CORS origin, `/.well-known/google-connected-apps`.

## 4. Screenshot list

Mark captures; the agent crops and places them. PNG, ≥ 1000 px wide (Anthropic's floor for MCP App
screenshots). No real customer data, no email addresses other than the reviewer account's.

| Channel | Shots (per service unless noted) |
|---|---|
| C1 claude.ai | 01 add custom connector dialog with the URL · 02 the sign-in page · 03 a tool call result · 04 (Parse) the upload widget with a file chosen |
| C2 Claude Code | 01 `claude mcp add` and `/mcp` sign-in · 02 a tool call in the transcript |
| C3/C4 ChatGPT, Codex | 01 developer-mode connector form · 02 sign-in · 03 a result · 04 (Parse) a file attached in the chat reaching `fileParams` |
| C5 M365 Copilot | 01 Agents Toolkit "Start with an MCP Server" + auth type · 02 the agent in m365.cloud.microsoft/chat · 03 sign-in · 04 result · 05 (Parse) the widget inline |
| C6 Copilot Studio | 01 MCP onboarding wizard · 02 OAuth type chosen · 03 the tool in a test chat |
| C7 VS Code | 01 the server added (from the Registry for Parse) · 02 a tool call in Copilot chat |
| C8 GE MCP | 01 the data store with the URL and OAuth fields · 02 Actions tab enabled · 03 a call in the GE app |
| C9 GE A2A (executor) | 01 the registered agent · 02 a delegated task and its artifact |
| C10 Connected Apps | 01 the console listing · 02 `@App` prompt in the Gemini app |

## Examples

### Example 1: a Copilot user adds Parse (after P1 + D2)

1. In VS Code, Agents Toolkit → Create a New Agent → Declarative Agent → Add an Action → Start with an MCP
   Server → `https://docparse.ailang.sunholo.com/mcp/connect/` → **OAuth (with dynamic registration)**.
2. Provision. In `m365.cloud.microsoft/chat`, pick the agent and ask "parse this sample invoice".
3. Copilot registers through `POST /oauth/register` (redirect host `teams.microsoft.com`, on the allowlist),
   shows **Sign in to Parse**, the user signs in, and the result comes back.

### Example 2: a GE admin adds the executor with per-user sign-in (after P1 + D2)

1. Operator: `executor_oauth/create_client.sh ge https://vertexaisearch.cloud.google.com/oauth-redirect`
   prints an `mco_` client ID and a secret, once.
2. GE admin: new MCP data store, URL `https://executor.ailang.sunholo.com/mcp/connect/`, OAuth with that
   client, Basic auth, PKCE ticked. Enable the tools in the Actions tab.

## P0 audit results (2026-11-10; widget tools and executor prod pending)

**Blocked half:** every signed-in call on both services returned "needs you to sign in again" in Claude
Code. Prod logs show why: the 13:10 tool calls got 401 (the access token from the 07:20 refresh had
expired), and Claude Code did **not** attempt a refresh (no `/oauth/token` request after the 401s). The
07:20 refresh itself succeeded. So this is client-side token state, not an mcp_oauth 0.2.0 regression; the
re-sign-in (Mark, `/mcp`) doubles as the first live CIMD sign-in against Parse on 0.2.0. Note for docs:
hosts cache `tools/list` at connect, so this session still sees Parse's pre-v0.46.0 schemas
(`maxChars: string`, `count: number`). A client holding the old list sends `"maxChars": "2000"` and is
refused (`expected int`) until it reconnects.

| Service | Tool | Result | Finding |
|---|---|---|---|
| Parse | `mcpFormats` | ✅ open | — |
| Parse | `mcpEstimate` | ✅ open, but ❌ data minimisation | Returned the REST string, double-encoded, with `meta.request_id`. **Fixed:** docparse PR #265 (`mcpOutput`; test asserts the shape and fails on the old code) |
| Parse | `mcpParse` (sample, summary) | ✅ | — |
| Parse | `mcpParse` (`content` + `notes.md`, `maxChars` sent as the old schema's string `"2000"`) | ✅ | Accepted; whether the client coerced it or serve-api bound the digits was not isolated |
| Parse | `createUpload` → `curl` → `mcpParse(fileRef)` | ✅ | Real 3 KB DOCX: 7 headings, 1 table. **G3 closed** for the code path |
| Parse | `mcpConvert` (Markdown → docx) | ✅ | `download_url` (signed, 1 h) + base64. **G4 closed** |
| Parse | `editDocument` (sample, `deltas: []`) | ✅ | **G2 closed** (sample ids now resolve) |
| Parse | `mcpAccount status` | ✅ | No upgrade links (`manageUrl: ""`) |
| Parse | `uploadDocument`, `uploadViaHost` | ⏳ needs a widget host (claude.ai, P2) | — |
| Parse | `submit_feedback` | not called (files real feedback) | — |
| Executor (dev) | `executor_whoami` | ✅ | Reports `client: "claude.ai"`: Claude Code signs in with claude.ai's CIMD identity |
| Executor (dev) | `ailang_eval`, `ailang_run` | ✅ | — |
| Executor (dev) | `ailang_run` reading `/etc/passwd` (`! {IO, FS}`) | ✅ refused | `policy_violation`, `missing_from_policy: ["FS"]` |
| Executor (dev) | `ailang_check` with `requires`/`ensures` | ⚠️ | **`verify.available: false`: the image has no Z3**, so the description's "verify its contracts with Z3 where available" never applies hosted. Also returns the request dir (`/tmp/ailang-exec/<uuid>/main.ail`) and a MOD010 warning naming it: an internal identifier in a tool result (OpenAI data minimisation) |
| Executor (dev) | `ailang_builtins` | ✅ | — |
| Executor (dev) | `docs_search "record update"` (latest) | ⚠️ | ~30 KB result, top hits are **old prompt files** (v0.8.2, v0.11.4, v0.16.x) rather than current docs: too big and stale for a listed connector |
| Executor (prod) | `executor_whoami`, `ailang_run` | ✅ | Signed in through `executor.ailang.sunholo.com`; runs v0.57.4 (annotations, scrub and z3 arrive with the next release + tag bump) |

**Executor fix tickets from P0** (all on `dev` in `8fd2cbca9`; prod with the next release + image-tag bump):
(E1) **z3 in the image** (`6a6f4b9e6`, base `distroless/cc-debian12`, pin `tools/ci/install_z3.sh`); live on the dev executor 2026-11-10: `verify.available: true`, `double` verified in 0.3 s, `wrong` counterexample `x = 0`.
(E2) `stdout`/`stderr` scrubbed of the request directory (`scrubDir`); `smoke.sh` checks it and fails on the
old code. (E3) `docs_search` returns path, title, 10 headings and an ~800-character excerpt, docs before
`prompts/v*.md`: ~30 KB → 3–11 KB per search.

**Hints vs OpenAI's rule** ("Use `false` for … persisting artifacts, starting stateful jobs …, uploading
files"), from prod `tools/list`:

| Tool | readOnly | Assessment |
|---|---|---|
| `mcpConvert` | true | **Likely wrong.** It stores the generated file for an hour behind `download_url`: persisting an artifact. Recommend `false` (not destructive). Changing it also changes how Claude prompts before calling it, so it waits for Mark |
| `mcpParse` | true | Borderline: output is kept only when the user turned history on (`mcpAccount history_on`). Keep `true`; say so in the OpenAI submission notes |
| `editDocument` | true | OK: returns edited blocks, persists nothing |
| `createUpload`, `uploadDocument`, `uploadViaHost`, `mcpAccount`, `submit_feedback` | false | OK |
| `mcpFormats`, `mcpEstimate` | true | OK |

Submission-kit gaps (2026-10-06) re-checked so far: **G1 closed** (email/password reviewer login),
**G5/G7 closed** except `mcpEstimate` (#265), **G6 closed** (no upgrade links, `mcp_account.ail:31`).
**G2, G3, G4 closed** (signed-in calls above).

## Success Criteria

- [ ] P0 table appended; every tool on both services passes with valid arguments, or has a ticket.
- [ ] P1b done: both prod URLs run the new code and pass `mcp check --target both`.
- [ ] Client models on in dev for both services; `test_oauth.sh` and docparse's OAuth tests cover GE, Copilot
      static and DCR; `ailang mcp check --target copilot` passes on dev.
- [ ] Every in-scope §1 cell ✅ with a dated screenshot.
- [ ] A2A v0.3 round trip from GE to the executor, screenshotted.
- [ ] Two public service pages with per-channel sections.
- [ ] §5 tracker: a submission per D1 choice.
- [ ] Changelog fragments; `make check-doc-review` clean for the new pages.

## Testing Strategy

**Automated:** `test_oauth.sh` (executor) and docparse's OAuth suite run the full flow per client model on
stub storage. `mcp check --target both|copilot` runs against dev in each service's release smoke gate.
**Live:** P2 is the test. A vendor doc alone never turns a cell green.

## 5. Submission tracker

| Marketplace | Service | Version | Submitted | Status | Notes |
|---|---|---|---|---|---|
| M3 MCP Registry | Parse | 0.14.1 (ailang-parse) | yes | listed | `io.github.sunholo-data/parse`, CI-stamped |
| | | | | | |

## Deferred Decisions

- The exact Copilot Studio callback host, and whether Copilot Studio's "Dynamic discovery" reads RFC 9728
  resource metadata or only RFC 8414: P2 measures it, then `dcrRedirectHosts` is set.
- Whether the executor needs widgets at all (probably not).
- `private_key_jwt` for OpenAI: stays deferred, as in M-MCP-OAUTH-CLIENT-MODELS D1.

## Non-Goals

- Google Cloud Marketplace (M6) unless D4 says otherwise.
- Billing through any marketplace. OpenAI forbids digital subscriptions in plugins; Parse's `mcpAccount`
  already strips upgrade links (gap G6, `mcp_account.ail:31`).
- Entra SSO. The Microsoft path uses OAuth, which we already serve.
- Teams or Outlook as separate surfaces beyond what a declarative agent gets.

## Timeline

| Week | Agent | Mark |
|---|---|---|
| 1 | P0 audit (0.5 d); P1 executor + Parse client models in dev (2 d); `--target copilot`; P1b release (code ships dark) | D1, D2, D4–D7; core + docparse promotes; multivac merge; OpenAI org verification; Connected Apps preview application |
| 2 | P3 A2A v0.3 (3 d) | P2 console sessions + screenshots (~1 d) |
| 3 | P4 docs (1.5 d); packets for M1, M2, M5 | D2 prod switch (code already in prod from P1b); D3 if C5 worked; submissions M1, M2 |
| 4+ | packets for M4; follow-ups from review | M4, M5 submissions as gates clear |

**Total: ~9 agent days, ~2–3 Mark days, spread over 3–4 weeks.**

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| DCR spam on a public write endpoint | Package limits (10/hour per caller, 500/day), redirect-host allowlist, unused registrations expire in 24 h |
| Copilot Studio's discovery does not match our metadata | P2 measures on dev before any prod switch; Manual mode with a static client is the fallback |
| Anthropic §1G ("uploaded files") read against Parse | The listing states Parse acts only on files the user explicitly passes through its own upload widget or URL |
| OpenAI EU-residency block | Submit from a global-residency project |
| No Microsoft Marketplace offer type accepts a declarative agent with an MCP plugin | D3 waits for P2; first enrolled step confirms the offer type; fallback is documented org-catalog and sideload install, which Copilot customers' admins can do without a listing |
| Reviewers can't sign in to the executor (Google only) | D5: email/password for console-created accounts |
| Tool snapshot frozen per Connected Apps version | Ship tool changes before submission; treat tool renames as a new version |
| Docs drift from vendor consoles | Every vendor claim in the docs carries its read date; `reviewBy` three months out |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language or runtime change |
| A2: Replayability | 0 | No trace change |
| A3: Effect Legibility | 0 | The OAuth code keeps its declared rows; new routes declare theirs |
| A4: Explicit Authority | +1 | Every client is either operator-minted or rate-limited and allow-listed DCR; `_auth` stays bound only from the verified identity |
| A5: Bounded Verification | 0 | Reuses the package's Z3-proved predicates; adds none |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +1 | One MCP surface serves every assistant; `mcp check` targets make each vendor's rules machine-checkable |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | Per-subject rate limits unchanged |
| A10: Composability | +1 | Same services, more hosts, no per-host forks |
| A11: Structured Failure | 0 | Unchanged |
| A12: System Boundary | +1 | Each new host is an explicit, allow-listed client at the OAuth boundary |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: no nondeterminism introduced
- [x] A3: no hidden side effects (DCR writes go through the declared store hooks)
- [x] A4: no ambient access (no client is accepted without a registration path and limits)
- [x] A7: not human-convenience over machine analysis

## Verification Log (2026-11-10)

| # | Claim | How checked | Result |
|---|---|---|---|
| V1 | Parse prod is v0.46.0 on AILANG v0.57.4 and passes `mcp check --target both` | `curl …/api/v1/health`; `ailang mcp check https://docparse.ailang.sunholo.com/mcp/connect/ --target both` | `ailang_commit: v0.57.4`, `parse (AILANG Parse) 0.46.0`, no failures |
| V2 | Executor custom domain live; only annotations fail | `ailang mcp check https://executor.ailang.sunholo.com/mcp/connect/` | server-info, lazy-auth, AS, framing PASS; 44 annotation FAILs (fixed on dev `4f9dd4835`, local check: no failures) |
| V3 | Executor uses only 0.1 entry points (no client models) | `executor_oauth/oauth.ail:40-42` imports `asMetadata`, `beginAuthorize`, `exchangeCode`, `refreshTokens` | confirmed; lock now 0.2.0 (`7af2d7195`) |
| V4 | Parse has no `ClientConfig` | `git grep "ClientConfig\|asMetadataFor\|registerClientDcr" origin/main -- '*.ail'` in docparse | empty |
| V5 | serve-api A2A answers only `tasks/send` / `tasks/get` | `internal/apiserver/a2a.go:133-135` | confirmed; no `message/send` |
| V6 | Executor not in the MCP Registry; Parse is | `curl registry.modelcontextprotocol.io/v0/servers?search=executor|ailang|sunholo` | Parse `io.github.sunholo-data/parse` (latest 0.14.1 with `/mcp/connect/`); no executor entry |
| V7 | Parse reviewer login has email/password (G1 closed) | docparse `static/oauth-login.js:101-109` | `EmailAuthProvider` for console-created accounts |
| V8 | M365 Copilot: auth types static OAuth, DCR, Entra SSO, None; redirect `teams.microsoft.com/api/platform/v1.0/oAuthRedirect`; Toolkit ≥ 6.12.0 | learn.microsoft.com `build-mcp-plugins` (ms.date 2026-09-30) | quoted |
| V9 | Copilot widgets: CORS for `{sha256}.widget-renderer.usercontent.microsoft.com`; no `fileParams`; only `readOnlyHint` | learn.microsoft.com `plugin-mcp-apps` (2026-09-30) | quoted |
| V10 | Marketplace: Partner Center "Microsoft 365 and Copilot" program; "Not every plugin package is eligible"; store validation + RAI checks | learn.microsoft.com `publish-plugin-publicly` (2026-09-30, updated 10-12), `publish` (11-06) | quoted; offer-type eligibility for an MCP plugin is **PENDING** (P2/D3) |
| V11 | Copilot Studio: Streamable HTTP only; OAuth Dynamic discovery / Dynamic / Manual | learn.microsoft.com `mcp-add-existing-server-to-agent` (2026-10-30) | quoted |
| V12 | OpenAI, Anthropic, Google listing gates in §2 | `integrations-review-external-2026-11-05.md` §§1.3, 2.3, 3.4, 4.3 | quoted there with URLs |
| V13 | Parse tool output carries no upgrade links | docparse `mcp_account.ail:31-39` | `mcpOutputWithout(…, ["upgradeUrl", …])` |
| V15 | Claude Code reaches Parse prod and the executor's dev URL | `~/.claude.json` `mcpServers`: `docparse` → `https://docparse.ailang.sunholo.com/mcp/connect/`; `ailang-exec-oauth` → `https://ailang-dev-executor-mcp-oauth-…run.app/mcp/connect/` | confirmed; executor prod in Claude Code not yet observed |
| V16 | `internal/mcpcheck` is where `--target` checks live | `ls internal/mcpcheck/` | `mcpcheck.go`, `framing_test.go`, `mcpcheck_test.go` |
| V17 | `sunholo/mcp_oauth` 0.2.0 is published | `ailang pkg search mcp_oauth` | `sunholo/mcp_oauth@0.2.0`; source in `~/.ailang/cache/registry/sunholo/mcp_oauth/0.2.0/` |
| V18 | The 0.2 API P1 relies on exists | grep of that source: `core.ail:363 asMetadataFor`, `flow.ail:196 createClient`, `:357 beginAuthorizeFor`, `:629 tokenRequest`, `:655 revokeRequest`, `:696 registerClientDcr` | confirmed |
| V19 | DCR defaults and expiry used as the spam mitigation | `core.ail:247` `defaultClientConfig` = `{confidential: false, dcr: false, dcrRedirectHosts: [], dcrPerCallerPerHour: 10, dcrGlobalPerDay: 500, revocation: false}`; `clients.ail:65-67` `dcrUnusedTtlSec() = 86400` | confirmed: 10/h per caller, 500/day, unused registration lives 24 h; all off by default |
| V20 | The package's security predicates are Z3-verified | sprint JSON `M-MCP-OAUTH-CLIENT-MODELS` M1e notes: "contracts 21/21 verified (verified == total)", 47/47 mutants killed; 23 `requires`/`ensures` lines in `core.ail` + `clients.ail` | confirmed at publish time |
| V21 | `--mcp-connect-only` refuses `--a2a`, and `/a2a/` is ungated | `internal/apiserver/mcp_auth.go:188`; `server.go:545-546` | confirmed; drives §3.1 |
| V22 | Partner Center offer types and timing | learn.microsoft.com `partner-center/marketplace-offers/add-in-submission-guide` (updated 2026-10-01): "Apps and agents for Microsoft 365 and Copilot, the new AI Apps and Agents category"; "four to six weeks to complete an app submission"; tab visible only to program members | offer category exists; whether it accepts an MCP-plugin package stays **PENDING** (D3) |
| V14 | Copilot rendered MCP Apps widgets with both bridges | aitana-labs `docs/workshop/protocol-gotchas.md` §12 (live 2026-07-08) | confirmed for a Python server; **PENDING** for AILANG (P2) |

## Related Documents

- [integrations-review-external-2026-11-05.md](integrations-review-external-2026-11-05.md): vendor requirements, quoted.
- [m-serveapi-directory-ready-submission-kit.md](v0_51_0/m-serveapi-directory-ready-submission-kit.md): Parse's Claude/ChatGPT packet (2026-10-06); P0 refreshes it.
- [m-mcp-oauth-client-models.md](v0_57_2/m-mcp-oauth-client-models.md): the client models P1 switches on; its P0 measurements are taken in P2 here.
- [m-ailang-executor-gemini-enterprise.md](v0_55_0/m-ailang-executor-gemini-enterprise.md): P3 (A2A) runs as its P2.
- [m-google-connected-apps-distribution.md](m-google-connected-apps-distribution.md): the M5 onboarding detail.
- Public: `docs/docs/guides/mcp-connectors.md`, `docs/docs/guides/hosted-executor.md`.

**Document created**: 2026-11-10
**Last updated**: 2026-11-10

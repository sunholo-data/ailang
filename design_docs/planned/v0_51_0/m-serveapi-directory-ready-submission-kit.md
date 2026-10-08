# Submission kit: AILANG Parse in the Claude and ChatGPT directories

**Status:** prepared 2026-10-06, not yet submitted. Vendor pages re-fetched 2026-10-06.
**Server:** `https://docparse.ailang.sunholo.com/mcp/connect/` (OAuth 2.1, PKCE S256, CIMD; `ailang mcp check --target both` passes 5/5).
**Companion docs:** `m-serveapi-directory-ready.md` (generic design), `m-serveapi-directory-ready-sources.md` (vendor quotes A1–A7, O1–O8), ailang-parse `design_docs/planned/v0_49_0/v0_49_0_vendor_directory_listing.md` (Parse decisions D2c/D2d, M5).

This kit holds the non-code material: what each vendor asks for today, what the website already has, the drafted text, and the steps only Mark can do. Some gaps turned out to be code or tool-surface problems that would fail review. They are listed in §1 because they decide whether submitting now makes sense.

---

## 1. Gaps that matter (read first)

Ordered by how likely each one is to cause a rejection.

| # | Gap | Vendor | Lane | Evidence |
|---|---|---|---|---|
| G1 | **Reviewers have no password login.** The consent page (`docparse/static/oauth-login.js:98-101`) offers only Google and GitHub through Firebase. OpenAI requires credentials that "work immediately without MFA approval, email or SMS codes, magic links". Anthropic requires "test account credentials". A dedicated Google or GitHub account will hit Google's new-device challenge or GitHub's emailed device-verification code when a reviewer signs in from their network. | Both (OpenAI hard) | docparse code + Firebase console | Already flagged as D2d / open question 1b in the v0_49_0 doc. Still open. |
| G2 | **`editDocument` cannot succeed over MCP.** It takes `filepath` ("uploaded file path (multipart upload)"). `sample_docx_formatting` returns `INPUT_NOT_FOUND`, and it takes no URL. Anthropic: "Every tool must return a successful response when called with valid parameters." | Both | docparse code | Live call 2026-10-06 (`/mcp/` surface, same module): `File not found: sample_docx_formatting`. |
| G3 | **There is no way to give the connector the user's own document or text.** `mcpParse` accepts only a sample ID or server path. An https URL returns `INPUT_NOT_FOUND`, because `parseFileSecure` is called with `sourceUrl=""` (`mcp_tools.ail:46`). `mcpConvert` accepts https URLs but not inline Markdown, so "turn these notes into a deck" fails. Claude/ChatGPT attachments never reach the server. Only public URLs (convert only) and samples work. | Both (use-case accuracy, functional review) | docparse code | Live calls 2026-10-06. Fix: pass https input through to `sourceUrl` in `mcpParse`, and add an inline `content` (text or base64) argument to `mcpConvert`/`mcpParse`. |
| G4 | **Generated files come back as base64 in the tool result**, with no download link. In claude.ai and ChatGPT the user cannot save the DOCX or PPTX. The listing must not promise "create a Word document" until there is a signed download URL or an MCP resource. | Both (accurate description) | docparse code | `mcpConvert` description; live response carries `content` base64. |
| G5 | **Tool text still advertises the device-flow and API-key surface on `/mcp/connect/`.** `mcpParse`: "requires a valid dp_ API key — get one via mcpAuth". `editDocument`: "apiKey: dp_ API key". The `AUTH_REQUIRED` and `INVALID_API_KEY` fixes say "Call mcpAuth" or "POST /api/v1/auth/device". `mcpFormats` returns `service.auth_method: "RFC 8628 Device Authorization — call mcpAuth"`, `tools_available` lists `mcpAuth`/`mcpAuthPoll`, which are absent on this surface, and `runtime.mode: "local"`. Its result is also JSON double-encoded as a string. `mcpFormats`' description exposes internals ("Delegates to package implementation… pkg/sunholo/ailang_parse/…"). | Both ("descriptions must match behavior"; O1 credentials) | docparse code | `tools/list` and `tools/call mcpFormats` on prod, 2026-10-06. |
| G6 | **Pricing and upgrade content in tool output.** `mcpFormats.service.tiers` lists EUR prices. `mcpAccount` has a `pricing` action, `getUploadUrl` says "Business tier only", and the status output carries `upgradeUrl`. OpenAI: "must not display subscription plans, initiate new subscriptions, or promote upgrades" (O3). OpenAI also now says plugins may sell only physical goods. | OpenAI | docparse code (D2d, still open) | Live output. |
| G7 | **Response hygiene for OpenAI.** Every result carries `request_id`, and `editDocument` returns a `_headers` block (`X-Request-Id`, quota headers). OpenAI app-review: remove "telemetry/internal identifiers (for example, session, trace, or request IDs…)". | OpenAI | docparse code | Live output. |
| G8 | **No public page documents the connector.** `ailang-parse/docs/mcp.html` covers `/mcp/` (stdio, API keys, device flow) and never mentions `/mcp/connect/`, OAuth sign-in, adding it in Claude or ChatGPT, or revoking access. Anthropic requires public documentation "by your publish date" (3C: "how users can troubleshoot issues"). | Both | ailang-parse site (**deployable now**) | `grep mcp/connect ailang-parse/docs` finds nothing. |
| G9 | **Privacy policy is silent on the connector.** It does not mention OAuth sign-in from an AI assistant, what a connection stores, the 24 h access key labelled `oauth: <client>`, 30-day rotating refresh tokens kept as SHA-256 digests in Firestore `oauth_records`, revocation, GitHub as a sign-in provider, what the assistant vendor receives, or `submit_feedback`. Account deletion has no self-service path: Terms §10 says "via the dashboard", but the dashboard has only *Delete history* and key revoke. | Both | ailang-parse site (**deployable now**); dashboard is blocked (§5) | §4.3 has the drop-in text. |
| G10 | **ChatGPT auth wiring (Phase B).** Tools carry no `securitySchemes`, and protected calls return a bare HTTP 401 with no `_meta["mcp/www_authenticate"]` (O5). The AS metadata lacks `authorization_response_iss_parameter_supported` and does not return `iss`, so ChatGPT will use the callback-ID redirect `https://chatgpt.com/connector/oauth/{callback_id}` (new since 10-01). OpenAI domain verification needs `/.well-known/openai-apps-challenge` served as plain text on the MCP host. | OpenAI | docparse + AILANG `serve-api` (Phase B, spike S1/V11) | Live discovery docs. |
| G11 | **The consent page links neither privacy nor terms.** OpenAI: "the flow must be transparent and explicit". | OpenAI (good practice for both) | docparse static page (Cloud Run, deployable) | `oauth-login.html` has no privacy/terms link. |
| G12 | **`submit_feedback` is exposed on the listed surface.** Its description invites the model to set `auto_dispatch=true` to "authorize the package agent to act on your submission immediately", which mentions AILANG internals and autonomous agents. A reviewer can read this as steering. It also accepts a free-text `contact`. Either drop it from `/mcp/connect/` or rewrite it to say only what it does. | Both (prompt-injection and data-collection criteria) | AILANG `serve-api` built-in | `tools/list`. |
| G13 | Cosmetic: `serverInfo.name` is `ailang-api` 0.8.1, not "AILANG Parse". | Both | AILANG `serve-api` config | `initialize` response. |

**Recommendation:** an Anthropic submission is viable once G1, G2, G5, G8, G9 and G12 are closed. G3 and G4 limit what the listing may honestly claim; the drafted text in §4 already avoids those claims. OpenAI needs all of the above plus G6, G7, G10 and G11.

---

## 2. Vendor requirements, re-checked 2026-10-06

### 2.1 What changed since the 2026-10-01 sources file

| Quote | Still matches? | Change |
|---|---|---|
| A1 (Policy §5D OAuth) | Yes, verbatim | — |
| A2 (submission: OAuth or none) | Yes | The page now also lists the full portal flow (§2.2). |
| A3 (lazy auth, 401 + `WWW-Authenticate`) | Yes, verbatim | — |
| A4 (`static_headers` beta) | Yes | — |
| A5 (10 s / 30 s) | Yes | — |
| A6 (auth types table) | Yes | `none` now points to lazy auth for mixed servers. |
| A7 (title + hints) | Yes, verbatim | — |
| New, Anthropic | — | Submission is in the **developer portal `claude.ai/directory/manage`** (any paid plan: Pro, Max, Team or Enterprise; Owner role on Team/Ent). Connectors are **listed as "Community" by default after an automated scan**. Verified review, with a person running each tool, happens only if Anthropic escalates. Policy §3D: "standard testing account with sample data". §3E: "at least three working examples of prompts". §3F: verify you own every endpoint/domain. Review criteria: public docs "required by your publish date", tool names ≤ 64 chars, no catch-all read/write tools, no AI image/video/audio generation. |
| O1 (restricted data) | Yes | — |
| O2 (auth transparent) | Yes; continues "Users must be informed of all requested permissions…" | — |
| O3 (no subscriptions) | Yes | Added: "plugins may conduct commerce only for physical goods". |
| O4 (OAuth 2.1 per MCP spec) | Yes | — |
| O5 (mixed auth needs `securitySchemes` + `_meta["mcp/www_authenticate"]`) | Yes | The fetch summary says a bare 401 is insufficient. That was not quoted verbatim, so V11 stays open until spike S1. |
| O6 (CIMD) | Yes | CIMD token auth now also lists `private_key_jwt`. |
| O7 (redirect URI) | **Changed** | Two URIs now: the stable `https://chatgpt.com/connector_platform_oauth_redirect` **only if** the AS sets `authorization_response_iss_parameter_supported: true` and returns `iss` matching `issuer` on every authorization response; otherwise `https://chatgpt.com/connector/oauth/{callback_id}`. |
| O8 (explicit boolean hints) | Yes | — |
| New, OpenAI | — | "Apps" are now **"plugins"**. Submission is a **ZIP package** (`plugin.json` + `mcp.json`, assets) uploaded at `platform.openai.com/plugins`. It needs **5 positive + 3 negative test cases**, a **demo video URL (required for MCP plugins)**, release notes, **domain verification** (`/.well-known/openai-apps-challenge`), and **individual or business verification** of the OpenAI org. **Screenshots are no longer shown**; up to 3 default prompts (≤ 128 chars) replace them. Suitable for ages 13–17. Data minimisation: strip request/trace IDs and timestamps from results. |

Sources fetched 2026-10-06: `claude.com/docs/connectors/building/{submission,review-criteria,authentication,lazy-authentication}`, `claude.com/docs/directory/publish`, `support.claude.com/en/articles/13145358`, `developers.openai.com/plugins/{plugin-guidelines,deploy/submission,deploy/app-review}`, `developers.openai.com/apps-sdk/build/auth`. `help.openai.com/en/articles/20001040` returned 403 to the fetcher and was not checked.

### 2.2 Anthropic: Claude connectors directory (portal `claude.ai/directory/manage` → Submit new → MCP connector)

Status key: ✅ present and adequate · 🟡 present, needs work · ❌ missing · 👤 only Mark can do it.

| Portal step / requirement | Spec | Status | Notes |
|---|---|---|---|
| Account can submit | Paid plan; on Team/Ent, an Owner | 👤 | Submit from the org that should own the listing long term (Holosun ApS). |
| Connection | `https://` URL, universal | ✅ | `https://docparse.ailang.sunholo.com/mcp/connect/` |
| Tools | Title + `readOnlyHint`/`destructiveHint` on every tool | ✅ | 8/8 annotated, `mcp check` PASS. Text problems are G5/G12. |
| Listing: name | ≤ 100 chars | ✅ draft | "AILANG Parse" |
| Listing: one-liner | ≤ 200 chars | ✅ draft | §4.1 |
| Listing: description | ≤ 2,000 chars; Anthropic cannot edit it | ✅ draft | §4.1 |
| Listing: 1–5 categories | portal list | 👤 | Suggest Productivity, Developer tools / Data. |
| Listing: documentation URL | public; may be shared privately during review | ❌ | G8. Proposed: `https://www.sunholo.com/ailang-parse/mcp.html#connector` (new section) or a new `connector.html`. |
| Listing: privacy policy URL | Policy 3A: clear and accessible | 🟡 | `https://www.sunholo.com/ailang-parse/privacy.html` (200). Needs the §4.3 additions (G9). |
| Listing: support contact | Policy 3B: verified contact and support channel | ✅ | `docparse@sunholo.com` (on privacy, terms and pricing pages). Optional: a `support.html` page; `/ailang-parse/support.html` is 404 today. |
| Listing: icon | No size spec published | ✅ | `docparse-skill/plugins/ailang-parse/assets/logo.png` (512×512 PNG) or `ailang-parse/docs/img/docparse-logo.png` (512×512). |
| Listing: slug | permanent once published | 👤 | Suggest `ailang-parse`. |
| Use cases: primary use cases, prerequisites, read/write | free text | ✅ draft | §4.2. Prerequisites: an AILANG Parse account (free tier; created on first Google/GitHub sign-in). Reads and writes: the only write is `mcpAccount` history on/off, plus key mint/rotation on sign-in. |
| Use cases: ≥ 3 working example prompts (Policy 3E) | must work | 🟡 | §4.4 drafts five that work against today's surface. Do not use attachment-based prompts (G3). |
| Company | name, website, primary contact | 👤 | Holosun ApS (CVR 44324687), `https://www.sunholo.com/ailang-parse/`, contact address for review mail. |
| Authentication | choose type | ✅ | **OAuth with client ID metadata documents** (`oauth_cimd`). Flag "starts without auth, tools prompt on demand" (lazy auth): `mcpFormats`/`mcpEstimate` are public. |
| Data handling | own API / proxied / third party; health data; sponsored | ✅ | Own API. No personal health data by design (the user decides what to upload). No sponsored content. Gemini (Vertex AI) is used only for AI-routed formats, as a sub-processor. |
| Test & launch: reviewer credentials | "fully populated account"; every link, credential and step | ❌ | G1 blocks a password login. Draft instructions in §4.5. Account must be **Business tier** so `getUploadUrl` succeeds, and have **< cap active keys** (OAuth mint counts against 3/10/25). |
| Test & launch: confirm every tool run | MCP Inspector or custom connector | 🟡 | Claude Code OAuth worked 2026-10-06. `editDocument` cannot pass (G2). Run all tools in claude.ai as a custom connector before submitting. |
| Compliance: 7 acknowledgments | directory guidelines, first-party API, financial transactions, AI media generation, prompt injection, conversation data, public docs | 👤 | Fine once G8 and G12 are done. Parse generates documents, not AI media. |
| Ownership (Policy 3F) | own every domain | ✅ | `docparse.ailang.sunholo.com`, `www.sunholo.com` (Holosun). |
| Data minimisation (Policy 1D) | no extraneous conversation data | 🟡 | Tools take only file refs and format. `submit_feedback` accepts free text plus `contact` (G12). |
| Escalations | `mcp-review@anthropic.com` | — | — |

### 2.3 OpenAI: ChatGPT plugin directory (`platform.openai.com/plugins` → Upload new or existing plugin)

| Requirement | Spec | Status | Notes |
|---|---|---|---|
| Org verification | "verified individuals or organizations" via Platform settings | 👤 | Business verification of Holosun ApS (likely needs company documents or ID). |
| Role | Owner, or "Apps Management Write" | 👤 | — |
| Package ZIP | `plugin.json` + `mcp.json` + assets | 🟡 | Exists in `sunholo-data/docparse-skill/plugins/ailang-parse/` (`plugin.json` with `extensions.com.openai.interface`, `mcp.json` → `/mcp/connect/`). Update fields per below. |
| `displayName` ≤ 30 | — | ✅ | "AILANG Parse" |
| `shortDescription` ≤ 30 | — | ✅ | "Parse and create documents" (26). Change to "Parse and convert documents" (27) until G4 is fixed. |
| `longDescription` ≤ 4000 | no pricing or subscriptions | 🟡 | Current text promises generation and editing (G2/G4). Use §4.1. |
| `developerName` ≤ 80 | — | ✅ | Holosun ApS |
| `category` | dashboard list | ✅ | Productivity |
| `capabilities` ≤ 20 × 120 | — | 🟡 | Remove "Edit documents" until G2 is fixed. |
| `websiteURL` | HTTPS | ✅ | `https://www.sunholo.com/ailang-parse/` |
| `privacyPolicyURL` | categories of data, purposes, recipients, retention, user controls | 🟡 | Covers everything except the connector (G9). |
| `termsOfServiceURL` | HTTPS | 🟡 | `terms.html` (200, last updated 1 Apr 2026). §10 promises account deletion "via the dashboard", which does not exist. |
| `supportURL` | HTTPS | ✅ | `https://github.com/sunholo-data/ailang-parse/issues` (public repo). Alternative: a `support.html` page. |
| `defaultPrompt` ≤ 3 × 128 | must work | ❌ | Both current prompts ("…from this Word document", "Turn these notes into a PowerPoint deck") fail on the connector (G3). Use §4.4 #2–#4. |
| `logo` / `composerIcon` | square, ≥ 48×48, ≤ 5 MiB | 🟡 | `logo.png` 512×512 ✅. `composerIcon` is `icon.svg`. The spec says "same format as icons" without naming SVG, so ship a PNG to be safe. |
| `brandColor` | #RRGGBB, 2:1 vs white | ❌ | Not set. Add. |
| Screenshots | optional; no longer shown | — | Skip. |
| MCP URL + domain verification | token at `/.well-known/openai-apps-challenge` | ❌ | Token from portal 👤. Route on docparse API (Cloud Run; deployable). |
| Auth | OAuth 2.1 per MCP spec; `securitySchemes` + `_meta` www_authenticate; redirect URIs | ❌ | G10 (Phase B). |
| Tool annotations | explicit booleans for read-only, destructive, open-world | ✅ | All three explicit on all tools. |
| No restricted data | no API keys or credentials through the model | 🟡 | No `apiKey` parameter on `/connect/` ✅, but descriptions still mention it (G5). |
| No subscriptions or upgrades | O3 | ❌ | G6. |
| Data minimisation in results | no request/trace IDs | ❌ | G7. |
| 5 positive test cases | description, prompt, tools_triggered, expected_behavior | ✅ draft | §4.6. |
| 3 negative test cases | prompt, why not, expected refusal | ✅ draft | §4.6. |
| Demo video URL | required for MCP plugins; must show the test cases | 👤 | Record after G1–G7 are fixed. |
| Release notes | summary of version | ✅ draft | "Initial ChatGPT listing: parse, convert, estimate, formats, account and large-file upload over OAuth." |
| Test credentials | login URL, account, sign-in steps; no MFA, codes or magic links | ❌ | G1. |
| Audience | suitable for 13–17 | ✅ | No issue. |
| Countries | optional `publication.countries` | — | Leave open; EU users are first-class (EU hosting). |

---

## 3. Website audit

### 3.1 Where the pages come from

| URL | Source | Repo visibility | Deploy | Can it deploy today? |
|---|---|---|---|---|
| `www.sunholo.com/ailang-parse/*` (home, `privacy.html`, `terms.html`, `mcp.html`, `pricing.html`, `docs.html`) | `sunholo-data/ailang-parse` → `docs/` | **public** | Pages, build_type `workflow` (`pages.yml`) | **Yes.** "Deploy Docs to GitHub Pages" succeeded 2026-10-06 18:07Z (run 37508851820). Public repos are not affected by the billing block. |
| `www.sunholo.com/docparse/dashboard.html`, `approve.html`, `partners.html` | `sunholo-data/docparse` → `docs/` | private | Pages, workflow | **No.** "Deploy Billing Dashboard to GitHub Pages" failed 2026-10-06 09:33Z: *"The job was not started because recent account payments have failed or your spending limit needs to be increased."* `www.sunholo.com/docparse/` itself is **404** (no index), so never use it as a listing URL. |
| `www.sunholo.com/` (incl. `/privacy`) | `sunholo-data/sunholo-data.github.io` (local: `~/dev/sunholo-data/website`) | private | Pages, workflow, CNAME `www.sunholo.com` | **No.** Failed 2026-10-05 and 2026-10-06 with the same billing annotation. Last success 2026-09-28 (prod). |
| OAuth consent page, discovery docs, tools | `sunholo-data/docparse` `static/` + API, on Cloud Run | private | `deploy.sh` (gcloud, local) | **Yes.** Does not use GitHub Actions. (docparse **CI** is red for the same billing reason, so PRs there merge without CI.) |

The listing URLs (privacy, terms, docs, website) all live in the **public ailang-parse repo**, so every site fix this kit needs can ship today. Only the dashboard (account deletion UI, any OAuth-key UX tweaks) and the company site are blocked until the org billing is fixed.

### 3.2 Per-requirement audit

| Item | Status | What's there / what's missing |
|---|---|---|
| Privacy policy (`/ailang-parse/privacy.html`, updated 25 Sep 2026) | 🟡 | **Has:** operator (Holosun ApS, CVR), data by mode, request records (metadata plus 10 KB output, on by default, can be turned off, deleted after 12 months), files not stored, large files deleted after 1 day, sub-processors (GCP EU, Firebase Auth US, Vertex AI), GDPR basis and rights, contact, cookies. **Missing:** (a) connecting from Claude, ChatGPT or another MCP client via OAuth; (b) what a connection stores: an access key labelled `oauth: <client host>` that expires after 24 h and is replaced on each refresh, with usage counters carried over, plus refresh tokens kept only as SHA-256 digests in Firestore `oauth_records`, valid 30 days and rotated on use, removed by a Firestore TTL policy; (c) how to revoke (dashboard key list, or email); (d) that the AI assistant's vendor receives tool results under its own privacy policy; (e) Google **and GitHub** as Firebase providers and what Firebase stores (provider UID, email, display name, photo URL); (f) how to delete request history (dashboard *Delete history* / `POST /api/v1/requests/delete`). The policy only mentions turning it off; (g) account deletion: there is no self-service path, so state "email docparse@sunholo.com"; (h) `submit_feedback` stores the text and optional contact for human review; (i) a children/age line (optional; OpenAI audience rule). |
| Terms (`/ailang-parse/terms.html`, updated 1 Apr 2026) | 🟡 | Adequate overall. §10 "delete your account at any time via the dashboard" is not true today; change to "by contacting us" or build the feature. Add one line: use through third-party AI assistants is also subject to those assistants' terms. "15 input formats" is stale; `mcpFormats` and the plugin now say 17. |
| Support contact | ✅ | `docparse@sunholo.com` on privacy, terms, pricing and index. GitHub issues (public) for `supportURL`. No `support.html`, which is optional. |
| Connector documentation | ❌ | G8. `mcp.html` describes local stdio, `/mcp/` with API keys, and the device flow only. Needs a "Use it in Claude or ChatGPT" section: URL, Settings → Connectors → Add custom connector (until listed), sign-in with Google/GitHub, the tool list with what needs sign-in, file input limits (public URL or samples, G3), revoking access, troubleshooting (KEY_LIMIT_REACHED at the key cap, expired session → reconnect). |
| Company / developer identity | ✅ | Holosun ApS, CVR 44324687, Denmark, on privacy and terms. |
| Logo / icon | ✅ | 512×512 PNGs: `ailang-parse/docs/img/docparse-logo.png`, `docparse-skill/plugins/ailang-parse/assets/logo.png`; SVG icon. |
| Brand colour | ❌ | Not declared in `plugin.json`. |
| Example prompts | 🟡 | Two in `plugin.json`, both broken on the connector (G3). |
| Data-handling / retention statement | ✅ | Thorough in privacy plus the DPA, except for the OAuth items above. |
| Pricing page | ✅ (Anthropic) / ⚠ OpenAI | Fine to exist on the website. Must not surface inside ChatGPT tool output (G6). |
| Consent page (`docparse.ailang.sunholo.com/oauth/...`) | 🟡 | Shows the requesting client and redirect hosts, explains the access key and revocation, links the dashboard. **No privacy/terms links (G11). No password option (G1).** |
| Company privacy (`www.sunholo.com/privacy`, 18 Mar 2026) | n/a | Covers the marketing site and uses `multivac@sunholo.com`. Don't use it for the listing; use the product policy. It cannot be redeployed today anyway. |

---

## 4. Drafted text

### 4.1 Descriptions

**Name:** AILANG Parse

**Anthropic one-liner (≤ 200):**
> Read Word, PowerPoint, Excel, OpenDocument, EPUB, email and PDF files as clean Markdown or structured blocks, and convert documents between formats. Office files parse deterministically, without AI.

**OpenAI shortDescription (≤ 30):** `Parse and convert documents`

**Long description (Anthropic ≤ 2,000 / OpenAI ≤ 4,000; no pricing, per OpenAI):**
> AILANG Parse turns documents into something Claude can read precisely, and back again.
>
> Point it at a document and it returns the content as Markdown, HTML or structured JSON blocks. Headings, lists, tables (including merged cells), footnotes, comments, tracked insertions and deletions, speaker notes and spreadsheet formulas are all kept, not flattened into plain text. Word, PowerPoint, Excel, OpenDocument, HTML, Markdown, CSV, EPUB, email (EML/MBOX), LaTeX and RTF are parsed deterministically by AILANG Parse's own engine. The same file always gives the same output, and no AI model sees the content. PDFs and images use AI-assisted extraction, and the service tells you beforehand which formats count as AI requests.
>
> It also converts between formats. Turn a document at a public link into Markdown, HTML or Quarto, or into Word, PowerPoint, Excel or OpenDocument.
>
> Tools:
> • List formats and capabilities: what can be read, what needs AI, and built-in sample documents to try.
> • Estimate cost and latency before parsing a document.
> • Parse document: structured blocks, Markdown, HTML or A2UI.
> • Convert document: from a public https link or a sample to another format.
> • Account, quota and usage: your plan's usage, your access keys, and whether request history is kept.
> • Large-file upload URL: for files over 32 MB.
>
> You sign in with Google or GitHub. AILANG Parse never sees your Claude or ChatGPT conversation, only the file references and formats the assistant sends to a tool. Uploaded files are not stored. Each request keeps a short record for 12 months, and you can turn off stored output or delete your history at any time. Hosting is in the EU (Belgium), and Holosun ApS (Denmark) operates the service.
>
> For documents that must not leave your machine, the same engine runs locally as a CLI or in the browser: https://www.sunholo.com/ailang-parse/

(Once G3/G4 are fixed, add: "Generate Word, PowerPoint and Excel files from Markdown, and download them.")

### 4.2 Use cases (Anthropic "Use cases" step)

- **Read Office documents faithfully.** Extract the text, tables, tracked changes and comments of a DOCX, PPTX or XLSX so Claude can summarise, compare or answer questions without losing structure.
- **Convert formats.** Turn a document at a public link into Markdown or HTML for further work, or into DOCX, PPTX, XLSX or OpenDocument.
- **Plan before spending.** Check which formats need AI and estimate cost and latency before parsing.
- **Manage your account from the chat.** See quota and usage, list access keys (including the one this connector uses), and switch stored request output on or off.

**Prerequisites:** a free AILANG Parse account. It is created automatically the first time you sign in with Google or GitHub. Large-file upload requires the Business plan.
**Reads/writes:** both. Most tools only read. `mcpAccount` can turn request history on or off. Signing in creates an access key on the account.

### 4.3 Privacy-policy additions (drop into `ailang-parse/docs/privacy.html`)

Insert as a new subsection after §2's table, and bump "Last updated":

> **Using AILANG Parse from Claude, ChatGPT or another AI assistant**
>
> You can connect AILANG Parse to an AI assistant that supports the Model Context Protocol, at `https://docparse.ailang.sunholo.com/mcp/connect/`. When you connect:
>
> - **Sign-in.** You sign in to AILANG Parse on our own page with Google or GitHub (Firebase Authentication). The assistant never receives your Google or GitHub password or tokens. The sign-in page shows which app is asking and where it will send you back.
> - **What we store for a connection.** We issue the assistant an access key for your account, labelled `oauth: <app host>` (for example `oauth: claude.ai`). It expires after 24 hours. The assistant then uses a refresh token to get a new key, which replaces the old one and keeps its usage counters. Refresh tokens are valid for 30 days and are replaced each time they are used. We store keys and refresh tokens only as SHA-256 hashes, never in readable form, in Google Cloud Firestore (EU). Expired records are deleted automatically.
> - **What the assistant sends us.** Only the tool call: the document reference (a sample name or a link) and the output format you asked for. We do not receive your conversation, chat history or other files.
> - **What the assistant receives.** The tool's result: parsed or converted content, estimates, or account information. From then on that content is handled under the assistant provider's own privacy policy (for example Anthropic's or OpenAI's), not ours.
> - **Requests made through an assistant** are recorded like any other API request (see Request records above), and your request-history setting applies.
> - **Disconnecting.** Revoke the connection at any time from your dashboard (Account → API keys, keys labelled `oauth: …`) or by removing the connector in the assistant. Revoking stops access at once. You can also ask us at docparse@sunholo.com.
> - **Feedback tool.** If you or the assistant use "Send feedback", we store the report text and any contact detail you include, so a person can review it. Don't include personal or confidential content.

Edits to existing rows:

- *Account & auth* row: "Email address, display name and profile photo URL, and the sign-in provider (Google or GitHub) with its user ID, via Firebase Authentication; access keys issued to apps you connect (hashed)." Retention: "Until account deletion. Connection keys expire after 24 hours; refresh tokens after 30 days."
- *Request records* row: add "Delete all of your request history at any time from the dashboard (*Delete history*)."
- §8 Your Rights: add "To delete your account, email docparse@sunholo.com. We delete it, its keys and its request records within 30 days." (Matches reality; Terms §10 must change to match.)
- Optional §: "AILANG Parse is not directed at children under 13."

### 4.4 Example prompts (each works against today's surface)

| # | Prompt | Tool(s) | Auth | Why it works |
|---|---|---|---|---|
| 1 | "What document formats can AILANG Parse read, and which of them need AI?" | `mcpFormats` | none | Public tool. |
| 2 | "Parse the AILANG Parse sample Word document with tracked changes and list every insertion and deletion with its author." | `mcpParse` (`sample_docx_track_changes`, `blocks`) | sign-in | Sample IDs resolve server-side. |
| 3 | "Convert https://www.sunholo.com/ailang-parse/assets/sample.docx to Markdown and summarise it." | `mcpConvert` (URL, `md`) | sign-in | Verified live 2026-10-06. |
| 4 | "Before I parse a PDF, estimate the cost and latency, and tell me whether it counts as an AI request." | `mcpEstimate` (`sample_pdf`, `markdown`) | none | Verified live. |
| 5 | "How much of my AILANG Parse quota have I used this month, and which access keys are active?" | `mcpAccount` (`status`, `keys`) | sign-in | — |
| 6 (Business account) | "I need to parse a 200 MB PDF called annual-report.pdf. Get me an upload link." | `getUploadUrl` | sign-in, Business | — |

OpenAI `defaultPrompt` (≤ 3 × 128 chars): #3 (shortened: "Convert https://www.sunholo.com/ailang-parse/assets/sample.docx to Markdown"), #2 (shortened: "Show the tracked changes in the AILANG Parse sample Word document"), #1.

### 4.5 Reviewer test instructions (Anthropic "Test & launch" / OpenAI "test credentials")

> **Server:** `https://docparse.ailang.sunholo.com/mcp/connect/`. OAuth 2.1 with PKCE (S256) and client ID metadata documents. No client secret is needed.
>
> **Account:** a pre-provisioned Business-plan AILANG Parse account with sample request history.
> - Login: `<REVIEWER EMAIL>` / `<PASSWORD>` *(requires G1: Firebase email/password provider enabled on the consent page)*
> - No MFA and no email or SMS codes. If you are asked for anything else, contact `<REVIEW CONTACT>`.
>
> **Steps**
> 1. Add the connector. In Claude: Settings → Connectors → Add custom connector, and paste the URL. In ChatGPT: add it as the developer-mode connector.
> 2. Ask prompt #1 (§4.4). It runs without sign-in.
> 3. Ask prompt #2. A **Connect** prompt appears. Sign in on the AILANG Parse page (`docparse.ailang.sunholo.com`) with the email and password above, then click **Allow**. You return to the chat and the tool call completes.
> 4. Run prompts #3–#6. Every tool listed in the portal is covered: `mcpFormats`, `mcpEstimate`, `mcpParse`, `mcpConvert`, `mcpAccount`, `getUploadUrl`*(, `editDocument` once G2 is fixed)*.
> 5. Optional: `mcpAccount` `keys` shows a key labelled `oauth: claude.ai` (or `oauth: chatgpt.com`). It is valid for 24 hours and is refreshed automatically.
>
> **Sample inputs:** sample IDs from `mcpFormats` (e.g. `sample_docx_tables`, `sample_xlsx_formulas`, `sample_pptx_notes`), or any public https link to a document (convert only).
> **Not supported, by design:** local file paths, audio and video on the hosted service, AI image or video generation.
> **Support during review:** docparse@sunholo.com

Seed the account before review: Business tier; at least 5 past requests with history on; active keys well under the 25 cap (each connecting client mints one).

### 4.6 OpenAI test cases

**Positive (5).** Run each with the reviewer account before submitting.

| # | Scenario | Prompt | Tools triggered | Expected behaviour |
|---|---|---|---|---|
| P1 | Discover formats | Prompt #1 | `mcpFormats` | Lists formats; marks PDF/images as AI-assisted and Office formats as deterministic; no pricing (after G6). |
| P2 | Tracked changes | Prompt #2 | `mcpParse` | Lists the insertions, deletions and moves in the sample with authors. |
| P3 | Convert from URL | Prompt #3 | `mcpConvert` | Returns Markdown with the "DocParse Test Document" headings and a 3-row format table. |
| P4 | Estimate | Prompt #4 | `mcpEstimate` | Says PDF is AI-routed, counts as an AI request, ~2 s. |
| P5 | Usage | Prompt #5 | `mcpAccount` | Shows plan, requests used and remaining, and the active keys, including the `oauth: chatgpt.com` key. |

**Negative (3).**

| # | Prompt | Why it should not act | Expected |
|---|---|---|---|
| N1 | "Parse the file at /Users/me/Documents/contract.pdf" | The server cannot read the user's disk. | Explains it needs a public link (or the local CLI); does not call a tool with the path, or reports `INPUT_NOT_FOUND` clearly. |
| N2 | "Transcribe this podcast MP3 for me" | Audio is not available on the hosted service. | Says hosted AILANG Parse doesn't process audio or video and points to the local CLI. |
| N3 | "Delete all my AILANG Parse request history" | Deletion is deliberately not an MCP action. | Explains it can be done from the dashboard; offers to turn history off instead (`mcpAccount history_off`) only if the user asks. |

---

## 5. What only Mark can do

1. **Fix GitHub org billing** for `sunholo-data`. It blocks the docparse dashboard Pages deploy, the company site deploy and docparse CI. It is not needed for the listing pages (public ailang-parse repo), but it is needed for any dashboard change, such as account deletion or OAuth-key UI.
2. **Decide G1:** enable the Firebase **Email/Password** provider in the prod Firebase project, then create the reviewer account (`docparse-review@…`, strong password, no MFA). Upgrade it to **Business** and seed it with history. Someone owns the credentials (open question 4 in the v0_49_0 doc).
3. **Anthropic:** sign in to claude.ai on a paid plan as the org Owner (Holosun) → `claude.ai/directory/manage` → Submit new → MCP connector. Pick categories and the slug (permanent), enter company and primary contact, upload the icon, accept the 7 compliance acknowledgments, paste the §4 text and §4.5 credentials, and confirm every tool was run in Claude as a custom connector.
4. **OpenAI:** complete **business verification** of the OpenAI Platform org (Settings → General) and grant yourself Owner or "Apps Management Write". Upload the `docparse-skill` plugin ZIP at `platform.openai.com/plugins`. Copy the **domain-verification token** from the portal so the docparse API can serve it at `/.well-known/openai-apps-challenge`. Enter the 5+3 test cases and the release notes.
5. **Record the demo video** (OpenAI, required): about 2–3 min, screen capture of ChatGPT adding the connector, signing in with the reviewer account, and running P1–P5. Host it unlisted (YouTube or Drive with link access) and paste the URL. No Anthropic video is needed. Screenshots are not needed for either: Anthropic wants them only for MCP Apps with UI, and OpenAI no longer shows them.
6. **Approve the privacy and terms wording in §4.3** (legal/GDPR owner). In particular, "account deletion by email within 30 days" is a commitment.
7. **Pick the support channel** shown publicly: `docparse@sunholo.com` (Anthropic), and GitHub issues or a new support page (OpenAI `supportURL`).

## 6. Pre-submission work for agents (not Mark-only)

| Item | Repo | Deployable now? |
|---|---|---|
| Privacy and terms edits (§4.3, Terms §10, format count) | ailang-parse `docs/` | Yes (public Pages) |
| Connector docs section in `mcp.html` (G8) | ailang-parse `docs/` | Yes |
| Fix `plugin.json` prompts, capabilities, shortDescription; add `brandColor`; PNG composer icon | docparse-skill | n/a (packaged into ZIP) |
| G1 sign-in option, G11 privacy/terms links on the consent page | docparse `static/` | Yes (Cloud Run `deploy.sh`) |
| G2, G3, G5, G6, G7 tool fixes | docparse `docparse_api/` | Yes (Cloud Run); CI won't run until billing is fixed |
| G10 `securitySchemes`, `_meta` www_authenticate, `iss` in authorization responses, challenge route | AILANG `serve-api` + `sunholo/mcp_oauth` + docparse | Phase B |
| G12 `submit_feedback` on the listed surface, G13 serverInfo name | AILANG `serve-api` | ailang dev |

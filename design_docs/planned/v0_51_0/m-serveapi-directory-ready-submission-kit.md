# Submission packet: AILANG Parse in the Claude and ChatGPT directories

**Status:** re-audited 2026-11-12 against the live server, current code and the vendor pages (re-fetched the same day). Not yet submitted. Replaces the 2026-10-06 kit.
**Server:** `https://docparse.ailang.sunholo.com/mcp/connect/`. Prod runs docparse **v0.47.1** (`serverInfo: parse (AILANG Parse) 0.47.1`) on AILANG **v0.57.4**. docparse v0.48.0 is released but not promoted. `ailang mcp check <url> --target both`: 10/10 PASS.
**Milestones:** M1 = Anthropic connector directory. M2 = OpenAI ChatGPT plugin directory.
**Companion docs:** `m-serveapi-directory-ready.md` (generic design), `m-serveapi-directory-ready-sources.md` (vendor quotes A1–A7, O1–O8), docparse `design_docs/planned/mcp_connector_usable.md` (the G2–G7 fix design, now shipped).

---

## 0. Verdict

**M1 (Anthropic) is blocked only on Mark.** The tool surface, docs, privacy policy and consent page are ready. To submit:
1. Enable Email/Password sign-in in the prod Firebase project and create the reviewer account (G1).
2. Take the MCP App screenshots of the upload card (§2.1).
3. Make the portal entries (§6).

Merging and deploying the open PRs (§7) is strongly recommended first, because a reviewer would see each of these issues:
- the internal note in the `mcpEstimate` description;
- IP addresses in `mcpAccount keys`;
- the old `submit_feedback` text that mentions autonomous agents.

None of them is a hard rule break for Anthropic.

**M2 (OpenAI) is blocked on code and on Mark:**
- **Code.**
  - sunholo-data/ailang#2269 (refused calls must carry `error` + `error_description` in `_meta["mcp/www_authenticate"]`; OpenAI says this is what opens the linking UI). It needs an AILANG release and a docparse AILANG bump.
  - sunholo-data/docparse#274 (no IP / user agent / user id in results).
  - sunholo-data/docparse#269 (no "Upgrade your plan" in quota refusals).
  - All three must be deployed to prod.
- **Mark.**
  - Business verification of the OpenAI org, from a project **without EU data residency** (OpenAI: "projects with EU data residency cannot submit plugins with MCP servers").
  - The G1 reviewer account.
  - The demo video.
  - The domain token.
  - The portal upload.

---

## 1. Gap table (re-audited 2026-11-12)

Key: ✅ closed · 🟡 closed in code, waiting on merge or deploy · ❌ open · 👤 Mark-only.

| # | Gap (2026-10-06) | Now | Evidence (2026-11-12) |
|---|---|---|---|
| G1 | Reviewers have no password login | 👤 **Changed.** The code is done; the prod provider is off. | `docparse/static/oauth-login.js:101-111` offers `EmailAuthProvider` with `disableSignUp`, and the live `oauth-login.js` serves it. A `accounts:signInWithPassword` probe against prod Firebase `ailang-multivac` with a non-existent user returns **`PASSWORD_LOGIN_DISABLED`**. Mark enables the provider and creates the account. |
| G2 | `editDocument` cannot succeed over MCP | ✅ | Live `editDocument filepath=sample_docx_formatting deltas=[]` returns `status: success`, 6 blocks. It also takes `fileRef`, `file`, `content` and https URLs. |
| G3 | No way to give the connector the user's own document | ✅ | `mcpParse`/`mcpConvert`/`editDocument` take `fileRef` (from `createUpload` / `uploadDocument`), `file` (ChatGPT attachment, `openai/fileParams`), `content`+`filename`, and https URLs. Verified live: URL convert, `file` with a public `download_url`, inline Markdown. |
| G4 | Generated files only come back as base64 | ✅ | `mcpConvert` returns `download_url`, a V4-signed GCS link valid for 3600 s. A generated PPTX downloaded: HTTP 200, 6177 bytes. |
| G5 | Tool text advertises the device flow and API keys | 🟡 Mostly closed. | Descriptions and `mcpFormats` are clean: a JSON object, no `auth_method` / `tools_available` / runtime. **Residual:** (a) the `mcpEstimate` description publishes an internal note ("Returned through mcpOutput … which OpenAI review forbids…") and says "Accepts a file path", but only sample ids work; fixed in docparse#274. (b) The `INPUT_NOT_FOUND` `suggested_fix` on MCP still says "Upload the file as multipart (-F …), pass a sample id from GET /api/v1/samples, or use gcsRef / sourceUrl". ❌ open, low priority: it comes from REST `apiInputPathRefusedHint`, and the fix belongs in `mcpOutput`, which #269 also edits. |
| G6 | Pricing and upgrade content in tool output | 🟡 | `mcpFormats` has no tiers. `mcpAccount` has no `pricing` action and strips `upgradeUrl` (`mcp_account.ail:39`). `getUploadUrl` is gone. **Residual:** the AI-quota refusal says "Upgrade your plan…"; fixed in docparse#269 (open). |
| G7 | `request_id` / `_headers` in results | ✅ | `mcpOutput` strips `request_id`, `meta` and `_headers` (`mcp_input.ail:118-132`). `mcpEstimate` was fixed in v0.47.0. Live estimate/convert/parse results carry none. See G16 for the remaining personal data. |
| G8 | No public connector docs | ✅ | `https://www.sunholo.com/ailang-parse/connect.html` (200) covers setup per app, sign-in, tools, sending documents, large documents, revoking and troubleshooting. `mcp.html#connect` links to it. |
| G9 | Privacy silent on the connector | 🟡 | Live `privacy.html` (updated 9 Nov) has `#ai-assistants`, GitHub/email sign-in, keys and refresh tokens, revocation, feedback, and deletion by email in 30 days. **Residual:** key-creation IP, browser and referrer are not disclosed, and there is no under-13 line; fixed in ailang-parse#159. |
| G10 | ChatGPT auth wiring | 🟡 **Changed.** | ✅ per-tool `securitySchemes` (`oauth2` / `noauth`) in `_meta`. ✅ a refused call is HTTP 401 + `WWW-Authenticate` + `_meta["mcp/www_authenticate"]`. ✅ `authorization_response_iss_parameter_supported: true`, and `issuer` equals the PRM `authorization_servers`. ✅ `sunholo/mcp_oauth` 0.2.0 `core.ail:350` appends `iss` to the code redirect; no authorization error is sent by redirect (pre-redirect errors are HTTP 400, and Deny does not redirect, see G15). ✅ `/.well-known/openai-apps-challenge` is live (404 `not found` while the env var is empty; `api_server.ail:2218`). ❌ **The `_meta` value lacks `error` and `error_description`**, which OpenAI now requires; fixed in ailang#2269. The token itself is 👤 (multivac PR #9 holds the terraform variable). |
| G11 | Consent page links neither privacy nor terms | ✅ | `oauth-login.html:162-164` links Terms and Privacy. FirebaseUI `tosUrl`/`privacyPolicyUrl` are set (`oauth-login.js:96-97`). |
| G12 | `submit_feedback` text mentions autonomous agents | 🟡 **Changed.** | The rewrite (it says only what the tool does) is commit 72fe5d258 "fix(mcp): tool descriptions on listed connectors say only what the tool does". On 2026-11-12 it sat on the main checkout's local `dev`, **not pushed and in no release**. Prod is on AILANG v0.57.4 and still shows "…where its autonomous agent watches… `auto_dispatch=true` to authorize the package agent…". It closes when 72fe5d258 is pushed and released and docparse bumps AILANG, or now with `serve-api --no-feedback-tool` in the docparse Dockerfile. |
| G13 | `serverInfo` is `ailang-api 0.8.1` | ✅ | `{"name":"parse","title":"AILANG Parse","version":"0.47.1"}`. |
| G14 | **New:** tool text that does not match what the hosted service does | 🟡 / ❌ | (a) `mcpFormats` lists PDF as `ai_required: true`, but `mcpEstimate sample_pdf` says pdftotext, no AI (metering since `700b8a6`). (b) "Headings become slides in pptx" (`mcpConvert` description, `mcpFormats.generation.target_notes`, website `document-generation.html`) is false: `# A … # B` gives **one** slide, locally and on prod, because `pptxCollectSlides` splits only on slide sections. docparse#274 makes the hosted text true. ❌ The generator fix (headings start slides) is an ailang_parse change. The issue text is drafted at `…/scratchpad/issue_slides.md` in the 2026-11-12 session; filing it was refused by the permission gate, so Mark files it or rules on it. **Until it lands, the listing promises no slide decks.** |
| G15 | **New:** Deny on the consent page does not return to the client | ❌ low | `oauth-login.js:135-139` `deny()` only shows "Not connected. You can close this tab." The client gets no `error=access_denied` redirect and waits. Not a review rule, but a reviewer who clicks Deny sees the assistant hang. |
| G16 | **New:** `mcpAccount keys` returns personal data and internal ids | 🟡 | Live it returns, per key, `createdIp`, `createdUserAgent` and `createdReferer`, plus the top-level `userId`. OpenAI: no "internal identifiers" or tracking metadata "such as … IP addresses". Fixed in docparse#274: the MCP view drops them and REST keeps them. Privacy disclosure in ailang-parse#159. |
| G17 | **New:** review assets | 👤 | OpenAI: `demo_recording_url` "required for MCP review". Anthropic: `uploadDocument` is an MCP App (`ui://docparse/upload`), so the listing takes 3–5 carousel screenshots (PNG ≥ 1000 px wide, cropped to the app response, no prompt in the image; video and GIF not accepted). |

---

## 2. Vendor requirements (re-fetched 2026-11-12)

Sources: `claude.com/docs/connectors/building/{submission,review-criteria}`, `claude.com/docs/directory/publish`, `support.claude.com/en/articles/13145358`, `developers.openai.com/plugins/{plugin-guidelines,deploy/submission,deploy/app-review,build/plugins}`, `developers.openai.com/apps-sdk/build/auth`. All loaded with HTTP 200.

### 2.1 Anthropic (portal `claude.ai/directory/manage` → Submit new → MCP connector)

What changed since 10-06:
- The Claude Console form "is no longer supported".
- Listings are scanned and published as **Community** by default. "Verified" review is applied by Anthropic and cannot be requested.
- Ultra is now among the plans that can submit.
- MCP Apps need 3–5 carousel screenshots.
- New optional "Allowed link URIs".
- Escalations: `mcp-review@anthropic.com`; migration questions: `directory@anthropic.com`.

| Step / requirement | Status | Value or note |
|---|---|---|
| Account | 👤 | Paid plan (Pro, Max, Ultra, Team, Enterprise); Owner on Team/Ent. Submit from the Holosun org. |
| Connection | ✅ | `https://docparse.ailang.sunholo.com/mcp/connect/` (Streamable HTTP). |
| Tools: `title` + `readOnlyHint`/`destructiveHint` on every tool; names ≤ 64; no catch-all tools | ✅ | 10/10 annotated (`mcp check` annotations PASS). The longest name is `uploadDocument`. |
| Listing: name ≤ 100 · one-liner ≤ 200 · description ≤ 2,000 · 1–5 categories · docs URL · privacy URL · support contact · icon · slug (permanent) | ✅ text in §4 · 👤 categories, slug | Docs `https://www.sunholo.com/ailang-parse/connect.html`. Privacy `https://www.sunholo.com/ailang-parse/privacy.html`. Support `docparse@sunholo.com`. Icon `docparse-skill/plugins/ailang-parse/assets/logo.png` (512×512). Suggested slug `ailang-parse`. Categories: Productivity; Developer tools. |
| Use cases: primary uses, prerequisites, reads/writes | ✅ | §4.2. |
| ≥ 3 working example prompts (Policy §3E) | ✅ | §4.3: seven prompts, six verified live. |
| Company | 👤 | Holosun ApS (CVR 44324687), `https://www.sunholo.com/ailang-parse/`. |
| Authentication | ✅ | OAuth (CIMD, no client secret), with "Required when the server asks": `mcpFormats`, `mcpEstimate` and `submit_feedback` need no sign-in. |
| Data handling | ✅ | Own API. No personal health data by design. No sponsored content. Vertex AI (EU) is a sub-processor for images and for scanned PDFs on request. |
| Test & launch: credentials for a "fully populated account"; every link and step; confirm every tool was run | ❌ G1 · 👤 | §5. Run all 10 tools in claude.ai as a custom connector first; `uploadDocument` (card) and `createUpload` were not run in the 2026-11-12 audit. |
| Screenshots (MCP Apps only) | 👤 G17 | 3–5 PNGs of the upload card, ≥ 1000 px wide, with the prompt for each entered separately. |
| Compliance: 7 acknowledgments | 👤 | Nothing blocks them: no money transfer and no AI media generation (Parse converts documents). Public docs are live. |
| Domain ownership (§3F) | ✅ | `docparse.ailang.sunholo.com`, `www.sunholo.com` (Holosun). |
| No extraneous conversation data (§1E) | ✅ | Tools take file references, formats and options only. The `submit_feedback` text is G12. |

### 2.2 OpenAI (portal `platform.openai.com/plugins` → upload ZIP)

What changed since 10-06:
- Test cases, `commerce` and the demo URL can live in the ZIP (`extensions["com.openai"].review`). Credentials and reviewer instructions are **refused** in the ZIP and go in the portal's Review details form.
- `developerName` is taken from the verified identity.
- SVG icons are allowed, but need a square viewBox of at least 48.
- The `mcp/www_authenticate` value must contain `error` and `error_description`.
- EU-data-residency projects cannot submit MCP plugins.
- After publication, tool changes are rescanned daily.
- The MCP origin cannot change after publication.

| Requirement | Status | Note |
|---|---|---|
| Org verification (business, Holosun ApS); project **not** EU data residency; role `api.apps.write` ("Apps Management Write") | 👤 | "Publishing under an unverified individual or business name will result in rejection." |
| ZIP = `plugin.json` + `mcp.json` + assets | 🟡 | `docparse-skill` `scripts/build-openai-submission.sh`. `openai-submission/mcp.json` declares `auth: {type: "mixed", client: {mode: "cimd"}}`. docparse-skill#6 adds the review block. |
| `displayName` ≤ 30 · `shortDescription` ≤ 30 · `longDescription` ≤ 4000 · `category` · 4 URLs | ✅ | "AILANG Parse" · "Parse and create documents" (26) · §4.1 / plugin.json · Productivity. Website, privacy, terms; support = `https://github.com/sunholo-data/ailang-parse/issues`. |
| `capabilities` ≤ 20 × 120 | ✅ | Parse, convert, generate Office documents, edit, summarise long documents. All are true now (G2–G4 closed). |
| `defaultPrompt` ≤ 3 × 128, unique, no @mentions | 🟡 #6 | The old two needed an attachment, and the deck prompt gave a one-slide deck. Replaced with §4.3 #2, #4, #1. |
| `brandColor` `#RRGGBB`, ≥ 2:1 vs white (dark ≥ 2:1 vs #212121) | 🟡 #6 | `#2563eb`: 5.2:1 on white, 3.2:1 on #212121. |
| `logo` / `composerIcon`: square, ≥ 48, ≤ 5 MiB; SVG needs a square viewBox ≥ 48 | 🟡 #6 | `logo.png` 512² ✅. `icon.svg` has a 32-unit viewBox ❌, so the composer icon is now `assets/composer-icon.png` (96²). |
| Exactly 5 positive + 3 negative test cases | 🟡 #6 | §4.4, in `openai-submission/review.json`. |
| `review.commerce` | 🟡 #6 | `false`. |
| Demo video (required for MCP review) | 👤 G17 | `DEMO_RECORDING_URL=https://… bash scripts/build-openai-submission.sh`, or enter it in the portal. |
| Domain verification: `https://docparse.ailang.sunholo.com/.well-known/openai-apps-challenge`, the exact token as plain text | ✅ route · 👤 token | Route live (404 until set). Token → terraform var `docparse_openai_apps_challenge` (multivac PR #9, open) → apply. |
| OAuth: S256 ✅, CIMD + `none` ✅, RFC 9207 `iss` ✅, `securitySchemes` per tool ✅, `_meta["mcp/www_authenticate"]` with `error` + `error_description` ❌ | ❌ | ailang#2269, then an AILANG release, then the docparse bump and deploy. The redirect URI is the stable `https://chatgpt.com/connector_platform_oauth_redirect`, because `iss` is advertised and returned. |
| Annotations: explicit `readOnlyHint` / `destructiveHint` / `openWorldHint` | ✅ | All three explicit on all 10 tools. |
| No restricted data (credentials through the model) | ✅ | No `apiKey` parameter on `/connect/` (`mcp check` credentials PASS). |
| No plans, subscriptions or upgrade prompts | 🟡 | #269 (quota refusal). Everything else is clean (G6). |
| Data minimisation (no request/trace ids, timestamps, IPs) | 🟡 | #274 (G16). The `mcpAccount keys` `created` (epoch ms) is kept: it is how a user tells keys apart. |
| Every user data type returned is in the privacy policy | 🟡 | ailang-parse#159. |
| Test credentials: no MFA, codes, magic links or private network | ❌ G1 · 👤 | §5. |
| Audience 13–17 | ✅ | Under-13 line in #159. |
| Release notes | ✅ | "Initial listing: read Word, PowerPoint, Excel, OpenDocument, PDF and more with structure intact; create documents from Markdown with a download link; account and usage." |

---

## 3. Public pages (all live, `www.sunholo.com/ailang-parse/`, source: public repo `sunholo-data/ailang-parse` `docs/`)

| Page | Status |
|---|---|
| `connect.html` (documentation URL) | ✅ 200. Setup for Claude Code, claude.ai/Desktop, Codex and ChatGPT; sign-in; tools; documents; revoking; troubleshooting. |
| `privacy.html` | ✅ 200, updated 9 Nov. #159 adds key provenance and the under-13 line. |
| `terms.html` | ✅ 200, updated 5 Nov. §10 deletion by email (fixed); assistant use covered. #159 corrects the PDF/AI wording. |
| `mcp.html` | ✅ 200. Links `connect.html`. |
| `document-generation.html` | ❌ claims "Headings become slides in PPTX" (G14 generator decision). |
| `support.html` | 404, not needed: support = `docparse@sunholo.com` (Anthropic) and GitHub issues (OpenAI `supportURL`). |
| Consent page `docparse.ailang.sunholo.com/oauth-login.html` | ✅ Shows the requesting app and redirect host, plus terms and privacy links. Email/password is offered once the Firebase provider is on (G1). |

---

## 4. Listing text (final)

### 4.1 Descriptions

**Name:** AILANG Parse

**One-liner (Anthropic ≤ 200; 189 chars):**
> Read Word, PowerPoint, Excel, OpenDocument, EPUB, email and PDF files with their tables, comments and tracked changes intact, and turn Markdown into Word, Excel and other files to download.

**OpenAI shortDescription (≤ 30):** `Parse and create documents`

**Description (Anthropic ≤ 2,000; no pricing; 1,757 chars):**
> AILANG Parse reads documents the way they were written, and writes new ones.
>
> Give it a Word, PowerPoint, Excel or OpenDocument file and it returns Markdown, HTML or structured blocks with headings, lists, tables (including merged cells), footnotes, comments, tracked changes, speaker notes and spreadsheet formulas kept. Office, OpenDocument, HTML, Markdown, CSV, EPUB, email, LaTeX and RTF are read without AI, so the same file always gives the same result. PDFs are read from their text layer. Images, and scanned PDFs when you ask, are read with an AI model.
>
> It also writes documents. Ask for a report, a brief or a table and it produces a Word, Excel, OpenDocument, HTML, Markdown or Quarto file from Markdown, or converts an existing document, and gives you a private download link valid for one hour.
>
> To give it a file, pick one in the AILANG Parse card in Claude, attach it in ChatGPT, give a public link, or try one of the built-in sample documents. Long documents are read in parts, with a short summary first.
>
> Tools: parse a document · convert or create a document · edit a document · upload a document (card) · upload links for files in a code sandbox · formats and samples · estimate time and AI use · account, usage and history setting · send feedback.
>
> You sign in with Google or GitHub. AILANG Parse acts only on the files you pass to it and never sees the rest of your conversation. Uploaded files are deleted within a day, and request records follow your plan's history period. You can turn off stored output or delete your history. Hosting is in the EU (Belgium), operated by Holosun ApS (Denmark).
>
> For documents that must not leave your machine, the same engine runs locally as the docparse CLI: https://www.sunholo.com/ailang-parse/

The OpenAI `longDescription` is the shorter variant in docparse-skill#6 `plugin.json`. It makes the same claims.

Not claimed until G14's generator fix lands: slide decks with one slide per heading.

### 4.2 Use cases (Anthropic "Use cases" step)

- **Read Office documents faithfully.** Text, tables, tracked changes and comments of a DOCX, PPTX, XLSX or OpenDocument file, so Claude can summarise, compare or answer questions without losing structure.
- **Write documents.** Turn what Claude drafted into a Word, Excel or OpenDocument file with a download link.
- **Convert formats.** A document at a public link, or one the user uploaded, to Markdown, HTML or Quarto, or to another Office format.
- **Plan before parsing.** See which formats use AI, and how long a parse takes.
- **Manage the account from the chat.** Plan, usage this month, access keys, and the request-history setting.

**Prerequisites:** a free AILANG Parse account, created on first sign-in with Google or GitHub.
**Reads/writes:** both. Most tools only read. `mcpAccount` can switch request history on or off. Uploads and created files are stored for up to a day. Signing in creates an access key on the account.

### 4.3 Example prompts

Verified against prod on 2026-11-12:
- **Signed-in calls** went from a Claude Code session connected to `/mcp/connect/` by OAuth. The account was on the Business plan.
- **Public calls** went by curl `tools/call` with no token.

| # | Prompt | Tool(s) | Sign-in | How verified (result) |
|---|---|---|---|---|
| 1 | "Which document formats can AILANG Parse read without AI?" | `mcpFormats` | none | curl: 17 input formats with `ai_required`/`deterministic`, 26 samples, `self_host_only` mp3/wav/mp4. No tiers or prices. (Until #274 deploys, PDF is shown as `ai_required: true`.) |
| 2 | "Show the comments and tracked changes in the AILANG Parse sample Word document." | `mcpParse` (`sample_docx_comments`, `sample_docx_track_changes`) | yes | Five comments by Jesse Rosenthal (e.g. "I left a comment.", 2016-05-09) and a move-from/move-to of "Here is the text to be moved." |
| 3 | "Convert https://www.sunholo.com/ailang-parse/assets/sample.docx to Markdown and summarise it." | `mcpConvert` (`input` URL, `md`) | yes | "# DocParse Test Document", 4 headings, the Format/Extension/Status table, plus `download_url`. |
| 4 | "Write a one-page project brief for a website relaunch as a Word document I can download." | `mcpConvert` (`content`, `filename` `brief.md`, `docx`) | yes | `status: success`, 3582-byte DOCX, `download_url` (3600 s). The same mechanism's PPTX link downloaded with HTTP 200. |
| 5 | "How much of my AILANG Parse allowance have I used this month?" | `mcpAccount` (`status`) | yes | Plan, `monthlyRequestsUsed`, `remainingRequests`, AI allowance. `manageUrl` is empty; no upgrade link. |
| 6 | "Before I parse a PDF, tell me whether it counts as an AI request and how long it takes." | `mcpEstimate` (`sample_pdf`) | none | curl: `strategy: pdftotext`, `counts_as_ai_request: false`, `estimated_ms: 2000`. |
| 7 | "Let me pick a Word file from my computer and summarise it." (claude.ai) | `uploadDocument` → `mcpParse` | yes | **Not verified.** It needs the claude.ai card UI. Mark runs it while taking the screenshots. |

For the Anthropic portal, use #2, #3, #4 and #1. OpenAI `defaultPrompt` uses #2, #4 and #1 (docparse-skill#6).

### 4.4 OpenAI test cases (5 positive + 3 negative)

These are in `docparse-skill/openai-submission/review.json` (#6). Each was run live with the tool the case names.

| # | Prompt | Tools | Expected | Verified by |
|---|---|---|---|---|
| P1 | "Which document formats can AILANG Parse read without AI?" | `mcpFormats` | Lists the formats read without AI and says images need AI. No prices. | curl, as §4.3 #1 |
| P2 | "Show the comments and tracked changes in the AILANG Parse sample Word document." | `mcpParse` | The comments with author and date, and the moved paragraph. | §4.3 #2 |
| P3 | "Summarise the structure of the Word document I attached: its headings and tables." (attachment `https://www.sunholo.com/ailang-parse/assets/sample.docx`) | `mcpParse` (`file`) | 4 headings (Introduction, Features, Sample Data Table, Conclusion) and 1 table. | `mcpParse file={download_url: <that URL>, file_id}` `summary`: headings 4, tables 1, the outline as listed |
| P4 | "Write a one-page project brief for a website relaunch as a Word document I can download." | `mcpConvert` | Markdown converted to docx; the `download_url` is given to the user. | §4.3 #4 |
| P5 | "How much of my AILANG Parse allowance have I used this month?" | `mcpAccount` | Plan, used and remaining, AI allowance. No upgrade prompt. | §4.3 #5 |
| N1 | "Parse the file at /Users/me/Documents/contract.pdf" | — | Explains it cannot reach the user's disk and asks for an attachment or link. If a call is made, it is `INPUT_NOT_FOUND`. | `mcpParse filepath=/Users/me/…` → `INPUT_NOT_FOUND` (path guard) |
| N2 | "Transcribe this podcast MP3 for me with AILANG Parse." | — | The hosted service does not process audio; points to the local CLI. No invented transcript. | `mcpParse content=<mp3 bytes> filename=podcast.mp3` → `FORMAT_NOT_AVAILABLE`; `mcpFormats.self_host_only` lists mp3 |
| N3 | "Delete all of my AILANG Parse request history." | — | No delete tool exists. Points to the dashboard (*Delete history*); offers `history_off` only if asked. | `tools/list`: there is no delete action, and `mcpAccount` actions are status/keys/history/history_on/history_off |

---

## 5. Reviewer instructions (paste into both portals; fill the placeholders)

> **Server:** `https://docparse.ailang.sunholo.com/mcp/connect/`. Streamable HTTP. OAuth 2.1 with PKCE (S256) and client ID metadata documents: leave any client ID and secret fields empty.
>
> **Test account:** a pre-provisioned AILANG Parse account (Business plan) with sample request history.
> - Login page: opens automatically at `https://docparse.ailang.sunholo.com/oauth-login.html` when a tool first needs sign-in.
> - Choose **Sign in with email**. Email: `<REVIEWER_EMAIL>` · Password: `<REVIEWER_PASSWORD>`
> - No MFA, no email or SMS codes, no magic links, no private network. If anything else is asked, write to `<REVIEW_CONTACT_EMAIL>`.
>
> **Steps**
> 1. Add the connector. Claude: Customize → Connectors → Add custom connector, and paste the URL. ChatGPT: the plugin under review.
> 2. Ask: "Which document formats can AILANG Parse read without AI?" It runs without sign-in.
> 3. Ask: "Show the comments and tracked changes in the AILANG Parse sample Word document." A sign-in prompt appears. Sign in with the email and password above and click **Allow**. You return to the chat and the answer lists the comments and the moved paragraph.
> 4. Ask: "Write a one-page project brief for a website relaunch as a Word document I can download." The reply contains a download link, valid for one hour.
> 5. Ask: "Convert https://www.sunholo.com/ailang-parse/assets/sample.docx to Markdown and summarise it."
> 6. Ask: "How much of my AILANG Parse allowance have I used this month?"
> 7. In Claude: "Let me pick a Word file from my computer and summarise it." An AILANG Parse card with a **Choose file** button appears. Pick any .docx.
>
> Every tool is covered by these steps, by the built-in samples (any id from step 2, e.g. `sample_xlsx_formulas`, `sample_pptx_notes`), or by your own attachment: `mcpFormats`, `mcpEstimate`, `mcpParse`, `mcpConvert`, `editDocument`, `uploadDocument`, `createUpload` (sandboxes with a shell), `mcpAccount`, `submit_feedback`. `uploadViaHost` is called only by the upload card.
> **Not supported, by design:** files on your own disk (use the card, an attachment or a link), audio and video on the hosted service, and AI image or video generation.
> **Support during review:** docparse@sunholo.com

Before review, seed the account:
- Business tier.
- At least 5 past requests with history on.
- Active keys well under the 25-key cap, since each connecting client mints one.

---

## 6. Mark-only steps, in order

1. **Approve merges and deploys (§7).** Agents opened the PRs; Mark merges them and promotes prod. This includes promoting docparse **v0.48.0** and the following release, which carries #269 + #274 and the AILANG bump.
2. **Firebase (G1):** in the prod project `ailang-multivac`, Authentication → Sign-in method → enable **Email/Password**. Sign-up stays off in the page.
   - Create `<REVIEWER_EMAIL>` (strong password, no MFA) in the console.
   - Sign in once through `/oauth-login.html` so the account exists in AILANG Parse.
   - Set it to **Business** and seed history (§5).
   - Decide who holds the credentials.
3. **Legal sign-off** on ailang-parse#159 (key provenance, under-13 line). File, or rule on, the G14 generator issue (headings → slides). The text is drafted in the 2026-11-12 session's scratchpad `issue_slides.md`.
4. **Anthropic (M1):**
   - Run every tool in claude.ai as a custom connector with the reviewer account.
   - Take 3–5 upload-card screenshots (PNG ≥ 1000 px wide, cropped, no prompt in the image).
   - In `claude.ai/directory/manage` → Submit new → MCP connector: categories, slug `ailang-parse` (permanent), company and contact, icon `logo.png`, §4 text, §5 credentials, the 7 acknowledgments, and confirm every tool was run.
5. **OpenAI org:** business verification of Holosun ApS (Settings). Use a Platform project **without EU data residency**. Grant yourself Owner or Apps Management Write.
6. **Domain token:**
   - Start the plugin in `platform.openai.com/plugins` and copy the domain-verification token.
   - Set `docparse_openai_apps_challenge` in multivac (PR #9) and apply. Check `curl https://docparse.ailang.sunholo.com/.well-known/openai-apps-challenge` returns exactly the token.
   - Click verify.
7. **Demo video (OpenAI):**
   - After the deploys in step 1, record 2–3 min: adding the plugin in ChatGPT, signing in with the reviewer account, then P1–P5 and one negative case.
   - Host it unlisted and reviewer-accessible.
   - Build with `DEMO_RECORDING_URL=<url> bash scripts/build-openai-submission.sh` in docparse-skill.
8. **OpenAI submission:**
   - Upload `dist/ailang-parse-0.19.0-openai.zip`.
   - Enter the §5 credentials and instructions in Review details (they are refused in the ZIP).
   - Check that the imported 5 + 3 test cases and the release notes (§2.2) are present.
   - Submit.

---

## 7. Agent PRs (opened 2026-11-12, none merged or deployed)

| PR | Closes | Merge/deploy note |
|---|---|---|
| sunholo-data/docparse#274 | G5(a), G14 hosted text, G16 | Cloud Run deploy. Merge with ailang-parse#159, whose privacy text says the fields are not returned to assistants. |
| sunholo-data/docparse#269 (pre-existing) | G6 residual | Cloud Run deploy. |
| sunholo-data/ailang#2269 | G10 `error` / `error_description` | Needs an AILANG release, then the docparse `AILANG_VERSION` bump (4 pins, see docparse CHANGELOG v0.46.0). The same bump closes G12 once 72fe5d258 (local `dev`, unpushed) is in that release. |
| sunholo-data/docparse-skill#6 | OpenAI test cases, `brandColor`, composer icon, prompts, `commerce` | Packaged into the ZIP; no deploy. |
| sunholo-data/ailang-parse#159 | G9 residual, under-13 line, PDF wording in terms | GitHub Pages deploys on merge. Legal sign-off first. |
| sunholo-data/ailang-multivac#9 (pre-existing) | G10 token plumbing | Terraform apply once the token is known. |

Still open with no PR: G5(b) (MCP `INPUT_NOT_FOUND` hint is REST-speak), G14 generator, G15 (Deny does not redirect).

---

## 8. Verification log (2026-11-12)

| Claim | Instrument | Result |
|---|---|---|
| Server identity | `initialize` | `parse` / "AILANG Parse" / 0.47.1 |
| Tool list | `tools/list` (no token) | 10 tools: createUpload, editDocument, mcpAccount, mcpConvert, mcpEstimate, mcpFormats, mcpParse, submit_feedback, uploadDocument, uploadViaHost (app-only) |
| Lazy auth | `tools/call mcpParse` without a token | HTTP 401, `WWW-Authenticate: Bearer resource_metadata="…/.well-known/oauth-protected-resource/mcp/connect/"`, `_meta["mcp/www_authenticate"]` = the same string, with no `error` / `error_description` |
| AS metadata | `GET /.well-known/oauth-authorization-server` | S256, `none`, CIMD true, `authorization_response_iss_parameter_supported: true` |
| Challenge route | `GET /.well-known/openai-apps-challenge` | 404 `not found` (route present, env empty) |
| Password sign-in | Firebase `signInWithPassword`, unknown user, prod web key | `PASSWORD_LOGIN_DISABLED` |
| Readiness | `ailang mcp check … --target both` | 10/10 PASS (does not check `error_description`) |
| Slides | `docparse n.md --convert n.pptx` (local) and `mcpConvert` (prod) with two `#` headings | 1 slide each |
| Key-list contents | `mcpAccount keys` (prod) | per key `createdIp`, `createdUserAgent`, `createdReferer`; top-level `userId` |
| Vendor pages | 9 pages fetched | All 200. The changes are folded into §2. |

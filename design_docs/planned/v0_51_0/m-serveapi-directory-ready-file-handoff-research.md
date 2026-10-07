# M-SERVEAPI-DIRECTORY-READY — file hand-off research (user upload → AILANG Parse MCP tool)

**Fetched:** 2026-10-07 (every URL below was fetched or read via `gh api` on this date).
**Rule:** primary sources only (vendor docs, vendor source code, spec repos). Where only a
third-party source says something, the cell says **not found (primary)**; any secondary claim
is labelled as such and is not relied on.

**Trigger incident.** A user uploaded a 14 KB `.docx` in claude.ai and asked Claude to parse it with
AILANG Parse. Claude had the file in its code-execution sandbox but the tool takes base64
`content`; 19 KB of base64 hit Claude's output limit, so it parsed locally instead.

**What we already ship (relevant).** The docparse MCP server already exposes `getUploadUrl`:
*"Request a pre-authenticated GCS upload URL for direct file upload. Business tier only. The
returned URL allows the client to PUT file content directly to GCS, bypassing the 32MB Cloud Run
request limit. After upload, pass the gcs_ref to POST /api/v1/parse."* (tool description as served
by `docparse.ailang.sunholo.com/mcp`). The upload host is GCS, not `docparse.ailang.sunholo.com`.

---

## 1. Claude (claude.ai web / Desktop) — code-execution sandbox

| Capability | Supported? | Quote | URL |
|---|---|---|---|
| Network settings exist (4 levels) | Yes (Team/Enterprise documented) | "Allow network egress toggled off: Claude operates with pre-installed packages only, with no internet access." / "Allow network egress to package managers only (default)" / "Allow network egress to package managers and specific domains: Claude can access package managers plus additional domains you specify. Add domains individually to whitelist specific resources your organization needs" / "All domains: Claude has full internet access except for domains on Anthropic's legal blocklist." | https://support.claude.com/en/articles/12111783 ("Create and edit files with Claude", dated August 6, 2026) |
| Who changes it — Team/Enterprise | Org owner | "Team and Enterprise organization owners can control network access settings in Organization settings > Capabilities." | same |
| Who changes it — Free/Pro/Max | The user, toggle only documented | "Enable file creation from Settings > Capabilities by toggling Code execution and file creation on. To give Claude access to external data sources, toggle Allow network egress on" | same |
| Free/Pro/Max: can the user add a specific domain (e.g. docparse) or pick "All domains"? | **Not found (primary).** The article documents the 4-level picker only under "Configuring network access (Team and Enterprise plans)". (Secondary only: an MCP GitHub issue by a connector author says individual users go to "Settings → Capabilities → Additional allowed domains", https://github.com/modelcontextprotocol/modelcontextprotocol/issues/2240; a third-party blog says "All domains" is the Pro/Max default — unverified.) | — | — |
| Default — Free/Pro/Max | Network on (scope not stated) | "Code execution and file creation is enabled by default" / "Network access is enabled, allowing Claude to install packages from approved sources" | same |
| Default — Team | **Contradictory within the article** | Availability section: "Network access is disabled by default; owners can enable it in organization settings". Getting-started section: "enabled by default at the organization level with Allow network egress toggled on with access to package managers only." | same |
| Default — Enterprise | Off | "enabled by default at the organization level with Allow network egress toggled off for new Enterprise organizations." | same |
| Approved (package-manager) domains | Fixed list; ours not on it; GCS not on it | "Anthropic Services (Explicit): api.anthropic.com, statsig.anthropic.com · GitHub: github.com · NPM: registry.npmjs.org, npmjs.com, npmjs.org · Python: pypi.org, files.pythonhosted.org, pythonhosted.org · Rust: crates.io, index.crates.io, static.crates.io · Ubuntu: archive.ubuntu.com, security.ubuntu.com · Yarn: yarnpkg.com, registry.yarnpkg.com" | same |
| HTTPS PUT/POST from sandbox to an allowlisted domain like `docparse.ailang.sunholo.com` | Implied yes when the domain is allowlisted / "All domains" — the method is not specified anywhere. The doc's own threat model describes exactly this: "using the sandbox environment to make an external network request to leak the data" | same |
| MCP traffic vs egress setting | MCP is separate from the sandbox | "If MCP (Model Context Protocol) integrations are enabled, network communication remains possible through those connections regardless of the network egress setting." | same |
| Uploaded files reachable in sandbox | Yes (stated generally); **path not found (primary)** | "perform advanced analyses on uploaded data" / "For PDFs larger than 30MB, Claude can process them through its computing environment without loading them into the context window." No page states `/mnt/user-data/uploads`. | same |
| File size limit | 30 MB | "The maximum file size is 30MB per file for both uploads and downloads." | same |
| Tool result → sandbox | Yes, for large results | "When a tool result exceeds approximately 150,000 characters and Claude's code execution sandbox is active, Claude writes the result to the sandbox filesystem instead of passing it inline to the conversation." | https://claude.com/docs/connectors/building/mcp-apps/troubleshooting.md |
| Sandbox code → call an MCP tool directly | **Not found.** No doc describes sandbox code invoking a connector. Chaining is model-mediated: the model can call tool A, run code, then call tool B, but every tool argument passes through model output (the root cause of the 19 KB failure). | — | — |
| Limits on connector tools | Hard | "Maximum tool result size: ~150,000 characters (claude.ai and Desktop) … Tool call timeout: 240 seconds per tool call" | https://claude.com/docs/connectors/building/index.md |

## 2. Claude / Anthropic + MCP spec — connector receives a user file

| Capability | Supported? | Quote | URL |
|---|---|---|---|
| Latest MCP spec version | 2026-07-28 | "The current protocol version is 2026-07-28." | https://modelcontextprotocol.io/specification/versioning |
| File input to tools in the spec | **No.** Not in 2026-07-28; nothing in its changelog adds file input. Roots deprecated, with "pass directories or files via tool parameters, resource URIs, or server configuration instead of Roots" | https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/changelog.mdx |
| `resource_link` content type | Server → client only (tool results) | "A tool MAY return links to Resources, to provide additional context or data. In this case, the tool will return a URI that can be subscribed to or fetched by the client" | https://modelcontextprotocol.io/specification/2026-07-28/server/tools |
| Blob resources | Server → client only | Claude supports "Text and binary resources"; WG charter: "Server-to-client file delivery, which is already covered by Resources and `BlobResourceContents`." | https://claude.com/docs/connectors/building/index.md ; https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/community/working-groups/file-uploads.mdx |
| SEP-2356 (declarative file inputs, `data:` URIs) | **Closed, not accepted** (2026-06-26) | "Closing in favor of #2631 which will likely be the more durable solution." (localden) | https://github.com/modelcontextprotocol/modelcontextprotocol/pull/2356 |
| SEP-2631 (File Objects and Transfer; `files/authorizeUpload`) | **Open draft** (PR is a draft, last updated 2026-10-02; author caseychow-oai) | "`files/authorizeUpload` and `files/authorizeDownload` authorize HTTPS transfer descriptors while keeping bytes out of JSON-RPC payloads." Server returns `upload: {transport:"https", method:"POST", url, headers, multipart:{fileField, fields:{token}}, expiresAt}`; client "uploads bytes out-of-band using the provided descriptor, then passes the returned file URI string in `tools/call`". | https://github.com/modelcontextprotocol/modelcontextprotocol/pull/2631 |
| WG status | File Uploads + Filesystems WGs being merged into a Files WG (open PR) | "first work item is deciding whether transferred files are addressed as Resource URIs" | https://github.com/modelcontextprotocol/modelcontextprotocol/pull/3389 |
| Claude support for draft features | No | "Claude doesn't yet support these MCP features … Resource subscriptions · Sampling · Advanced or draft capabilities" / "Claude implements a subset of the MCP specification" | https://claude.com/docs/connectors/building/index.md |
| Anthropic connector mechanism to hand a user upload to a connector (file ref / Files API hand-off) | **Not found.** No page in the claude.com/docs index (https://claude.com/docs/llms.txt) describes one. | — | — |
| Directory guidelines on user files | Restrictive | "Don't query Claude's memory, chat history, conversation summaries, or user files" | https://claude.com/docs/connectors/building/review-criteria.md |
| Proposal to auto-allowlist connector domains for sandbox egress | Issue closed as completed 2026-02-13 with no comment; **no spec or product change found** | "When a connector provides a presigned upload URL … the client's code execution sandbox cannot reach the URL unless the user has manually added the domain to their egress allowlist." | https://github.com/modelcontextprotocol/modelcontextprotocol/issues/2240 |

## 3. OpenAI ChatGPT apps/plugins (Apps SDK → "plugins") and Codex

| Capability | Supported? | Quote | URL |
|---|---|---|---|
| Field name | `_meta["openai/fileParams"]` | "To let ChatGPT pass files to a tool, list each top-level file input in `_meta["openai/fileParams"]`. Each listed field must resolve to a file object or an array of file objects." | https://developers.openai.com/plugins/reference (redirected from learn.chatgpt.com/apps-sdk/reference) |
| Schema the tool must declare | 4 properties; 2 required | "Every file object schema must declare all four supported properties: download_url (required), file_id (required), mime_type, file_name … The Scan Tools step and plugin submission reject a file schema that omits any of the four properties, does not require download_url and file_id, marks either optional property as required … You can declare extra optional properties." Example: `"_meta": { "openai/fileParams": ["file"] }` with `"file": {"$ref": "#/$defs/OpenAIFile"}` | same |
| What the tool receives | URL + id | "At runtime, ChatGPT passes file values with snake case fields: `{ "download_url": "https://...", "file_id": "file_...", "mime_type": "image/png", "file_name": "input.png" }` ChatGPT always includes download_url and file_id; it may omit mime_type and file_name." | same |
| URL expiry / auth | **Not found** for `download_url`. Only the widget helper is described as temporary: "Request a temporary download URL for a file." | same |
| Size limit (ChatGPT) | **Not found.** | — | — |
| Multiple files | Yes | "To accept more than one file, define the top-level field as an array and use the same file object schema in items." | same |
| Works in ChatGPT developer mode (custom, unlisted connector)? | **Not found in OpenAI docs** (neither the reference nor https://developers.openai.com/plugins/deploy/connect-chatgpt says). Secondary only: a third-party GitHub issue reports ChatGPT "never hydrates openai/fileParams" for its dev-mode connector (https://github.com/thaynes43/cigar-journal/issues/202); OpenAI's examples repo has an open, unanswered report that mobile sends `{"images": ["chat_upload://image_0"]}` instead of file objects (https://github.com/openai/openai-apps-sdk-examples/issues/185, opened 2026-01-20). | — | — |
| Codex CLI | **Only for OpenAI's own `codex_apps` server; custom MCP servers are excluded** | `// Disallow custom MCPs from uploading files via fileParams.` → `(server == CODEX_APPS_MCP_SERVER_NAME).then(...)` with `CODEX_APPS_MCP_SERVER_NAME = "codex_apps"`. Mechanism for codex_apps: "read those files from the primary environment, upload them to OpenAI file storage, and rewrite only the declared arguments into the provided-file payload shape". Limit `OPENAI_FILE_UPLOAD_LIMIT_BYTES: u64 = 512 * 1024 * 1024`. | openai/codex @ b17c74cf (2026-10-07): `codex-rs/core/src/mcp_tool_call.rs`, `codex-rs/core/src/mcp_openai_file.rs`, `codex-rs/codex-mcp/src/mcp/mod.rs`, `codex-rs/codex-api/src/files.rs` |
| Codex CLI shell network | Off by default | "For the ChatGPT desktop app, Codex CLI, or IDE extension, the default `workspace-write` sandbox mode keeps network access turned off unless you enable it in your configuration" | https://developers.openai.com/codex/agent-approvals-security.md (redirects to learn.chatgpt.com/docs/agent-approvals-security.md) |

## 4. Other vendor-recommended patterns for large inputs

| Pattern | Source status | Quote | URL |
|---|---|---|---|
| Out-of-band upload descriptor (presigned-style URL + token + expiry), then pass a file URI to `tools/call` | **Draft SEP** (SEP-2631), not in any spec version, no host implements it per any primary doc | see §2 row SEP-2631 | https://github.com/modelcontextprotocol/modelcontextprotocol/pull/2631 |
| Presigned upload URLs | Explicitly left open by the WG | "The WG may evaluate approaches such as streaming, chunked transfer, or presigned upload URLs as part of its design work" | https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/community/working-groups/file-uploads.mdx |
| Inline base64 is the status quo the WG wants to replace | — | "Today, servers that need a file from the user resort to prose instructions asking for base64 strings or local paths, which produces inconsistent UX" | same |
| Host-managed file → URL (ChatGPT) | Shipped (listed plugins) | `openai/fileParams` → `download_url` | https://developers.openai.com/plugins/reference |
| Anthropic recommendation for large *inputs* | **Not found.** Only guidance for large *outputs*: "return identifiers or previews in the initial result and provide a separate tool to retrieve the full content when needed" | https://claude.com/docs/connectors/building/mcp-apps/troubleshooting.md |

## 5. Embedded UI (MCP Apps / ChatGPT widgets): can a widget take a file and POST it to our server?

| Capability | Supported? | Quote | URL |
|---|---|---|---|
| MCP Apps spec version | Stable `2026-01-26` + `draft` | spec dirs `specification/2026-01-26/apps.mdx`, `specification/draft/apps.mdx` | https://github.com/modelcontextprotocol/ext-apps (main @ 82221c0c, 2026-09-25) |
| Widget network calls | Only to declared `connectDomains`; default none | "Origins for network requests (fetch/XHR/WebSocket) - Empty or omitted = no external connections (secure default) - Maps to CSP `connect-src` directive" / restrictive default includes `connect-src 'none';` / "No Loosening: Host MAY further restrict but MUST NOT allow undeclared domains" | https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx |
| Sandbox attributes | `allow-scripts allow-same-origin` required | "The Sandbox MUST have the following permissions: `allow-scripts`, `allow-same-origin`." | same |
| File *input* / picker / upload API in MCP Apps | **Not found** in stable or draft spec. Draft adds download only: "`ui/download-file` - Request host to download a file" and "MCP Apps run in sandboxed iframes where direct file downloads are blocked (`allow-downloads` is not set)." The File Uploads WG charter lists only intent: "embedded app UIs may surface their own file pickers". Whether `<input type=file>` works inside a host's iframe: **not found** in any host doc. | https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/draft/apps.mdx |
| Which Claude surfaces render MCP Apps | claude.ai web, Desktop (iOS has dev-tools docs); not Claude Code | "claude.ai on the web: remote connectors, including MCP Apps that display interactive content in the conversation" / "Rendering the UI takes a host that supports MCP Apps, such as the Claude desktop app. Claude Code calls the tool as text and doesn't render the UI." / "On iOS, the Claude app renders your MCP App inside a `WKWebView`." | https://claude.com/docs/connectors/overview.md ; https://claude.com/docs/connectors/building/mcp-apps/quickstart.md ; https://claude.com/docs/connectors/building/mcp-apps/troubleshooting.md |
| Claude widget → own API directly | Yes (documented as a normal case) | "The missing `Referer` header affects requests your app makes directly from the user's device, such as loading bundles and images or calling your own API from client-side code." / "Configure your infrastructure to allow requests whose `Origin` matches `*.claudemcpcontent.com` and return a corresponding `Access-Control-Allow-Origin` header." / Claude's `ui.domain` = "the first 32 hex characters of the SHA-256 of your server URL, followed by `.claudemcpcontent.com`" | same troubleshooting page; https://claude.com/docs/connectors/building/mcp-apps/getting-started.md |
| ChatGPT widget file API | **Yes** — host-mediated | "`window.openai.uploadFile(file, { library?: boolean })` Upload a user-selected file and receive a `fileId`." / "`window.openai.getFileDownloadUrl({ fileId })` … Works for files uploaded by the widget, selected from the file library, passed via file params, or returned by tool file references." / "`window.openai.selectFiles()` Open the file library picker for existing files … Feature-detect this helper" | https://developers.openai.com/plugins/reference |
| ChatGPT `uploadFile` non-image files | Yes since 2026-03-09 | "2026-03-09 Non-image file uploads … `window.openai.uploadFile` now supports non-image file types." | https://developers.openai.com/plugins/changelog |
| ChatGPT widget CSP | Declared domains | "`connectDomains`: string[]. Domains the widget may contact via fetch/XHR." (`_meta.ui.csp`; legacy `_meta["openai/widgetCSP"].connect_domains`) / "The CSP controls standard `fetch` requests." / "The plugin review process checks the declared policy against the UI behavior." | https://developers.openai.com/plugins/reference ; https://developers.openai.com/plugins/guides/security-privacy ; https://developers.openai.com/plugins/build/chatgpt-ui |
| ChatGPT renders MCP Apps resources | Yes | "2026-02-22 MCP Apps compatibility" / "ChatGPT also honors `_meta["openai/outputTemplate"]` as a compatibility alias" (of `_meta.ui.resourceUri`) | changelog; chatgpt-ui page |
| Widget file APIs in developer-mode connectors | **Not found.** | — | — |

---

## Recommended design

Ground truth that drives the ranking:

- **No MCP host today hands a user-uploaded file to a third-party connector by URL or reference
  except ChatGPT's `openai/fileParams`** — and only for listed plugins for sure; dev mode is
  undocumented, and Codex excludes custom servers by code.
- Every Claude path that moves bytes goes either through **model output** (base64, capped — the
  incident) or through **sandbox egress** (user/admin network setting) or through **a widget the
  user drives** (MCP Apps, `connectDomains`).
- The MCP-native fix (SEP-2631 `files/authorizeUpload`) is a draft; design so it can slot in later.

Server changes common to all ranks (one primitive, three clients):

1. **`getUploadUrl` for every tier, hosted on our own domain.** Return `{upload_url, method, headers,
   expires_at, file_ref}` where `upload_url` is `https://docparse.ailang.sunholo.com/u/<token>`
   (we proxy/redirect to GCS). One domain to allowlist instead of `storage.googleapis.com`, and the
   shape is already SEP-2631's `upload` descriptor, so a future `files/authorizeUpload` is a rename.
2. **`mcpParse` accepts `file_ref` and `url`** (https URL fetched server-side — this also covers
   ChatGPT `download_url`), alongside the existing base64 `content` for small files.
3. Tool descriptions state the decision rule in one line: "file in a sandbox or on disk → getUploadUrl
   then curl PUT; file larger than ~4 KB → never base64".

### (a) claude.ai web

1. **Sandbox PUT to `getUploadUrl`** → `mcpParse(file_ref)`. Works when egress reaches our domain.
   Proven only for "All domains" / allowlisted domain; the per-user domain allowlist for Free/Pro/Max
   is **not documented by Anthropic**, Team defaults are contradictory, Enterprise is off. On failure the tool
   must return a clear error that names the setting to change (Settings > Capabilities > network egress /
   Organization settings > Capabilities for Team/Enterprise owners).
2. **MCP App upload widget** (`parseUpload` tool renders a drop zone; widget `fetch` POSTs to our
   `connectDomains` origin with a short-lived token minted by the tool call; result returns via
   `tools/call` from the widget). Not affected by sandbox egress. Risk: `<input type=file>` inside
   Claude's iframe is **not documented** — needs a live test before committing.
3. Base64 `content` for small files only (fallback that exists today).

### (b) Claude Desktop

Same as (a); Desktop renders MCP Apps (documented) and uses the same sandbox. Extra option:
an MCPB local extension reads local paths directly — out of scope for a remote connector.

### (c) Claude Code

1. **Local file path → `getUploadUrl` + `curl` PUT from Bash** (Claude Code sandbox: "allowed domains,
   which start empty", first use prompts for approval) → `mcpParse(file_ref)`.
2. Plugin / local stdio server that reads the path itself (the existing `ailang-parse` plugin's
   `mcpParse(filepath)` shape). MCP Apps do not render in Claude Code (documented).

### (d) ChatGPT web

1. **`_meta["openai/fileParams"]: ["file"]`** on `mcpParse` with the exact 4-property `OpenAIFile`
   schema; server fetches `download_url`. Expiry/auth of `download_url` is undocumented → fetch
   immediately on call, never store the URL. Dev-mode hydration is unverified → test in dev mode first;
   expect it to work for the listed plugin.
2. **Widget with `window.openai.uploadFile`** → `fileId` → `callTool` with a fileParams arg (or
   `getFileDownloadUrl`, then pass the URL). Host-mediated, no CSP gymnastics.
3. Base64 for tiny files.

### (e) Codex CLI

1. **Local path → `getUploadUrl` + `curl` PUT**. Needs network on (`[sandbox_workspace_write]
   network_access = true`) or approval, since the default is "network access turned off".
2. `fileParams` will **not** help: Codex only rewrites file params for the `codex_apps` server.
3. Base64 for small files.

### Cross-cutting order of work

1. Our-domain `getUploadUrl` + `file_ref`/`url` on `mcpParse` (unblocks a, b, c, e and is SEP-2631-shaped).
2. `openai/fileParams` schema on `mcpParse` (d, a few lines).
3. MCP App upload widget (a, b, d) — only after a live `<input type=file>` test in claude.ai.
4. Track SEP-2631 / Files WG; adopt `files/authorizeUpload` when a host ships it.

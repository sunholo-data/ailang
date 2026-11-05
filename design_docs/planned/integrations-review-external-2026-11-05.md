# External integrations review: OpenAI, Anthropic, Google (2026-11-05)

**Subject:** AILANG Parse (docparse). It is a remote MCP server (Streamable HTTP, OAuth 2.1 + PKCE +
CIMD, an MCP Apps widget, and ChatGPT `openai/fileParams`), plus REST, and possibly A2A.

**Method.** Every row comes from a vendor page, vendor source code or a spec repository, fetched on
2026-11-05. OpenAI and Claude pages were fetched as their published Markdown twins (`.md`). Google pages
were fetched as HTML and converted to text. Quotes are verbatim, with link markup removed. Google pages
show a "Last updated" date, and it is given where it matters. Blogs were used only to find primary
pages. The exception is Google's own announcement post (blog.google), which counts as a vendor primary.
Anything that wasn't verified is marked **not found (primary)**.

**Baseline for "changed since early October".** The baseline is the earlier research in
`design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md` (fetched 2026-10-01),
`…-submission-kit.md` §2 (re-checked 2026-10-06) and `…-file-handoff-research.md` (fetched 2026-10-07).
It is supplemented by Internet Archive snapshots of the vendor pages themselves, diffed sentence by
sentence against today's text:

| Page | Snapshot used |
|---|---|
| developers.openai.com/plugins/build/auth | 2026-09-29 |
| developers.openai.com/plugins/plugin-guidelines | 2026-10-05 |
| developers.openai.com/plugins/deploy/submission | 2026-10-05 |
| developers.openai.com/plugins/deploy/app-review | 2026-09-27 |
| developers.openai.com/plugins/reference | 2026-09-27 |
| claude.com/docs/connectors/building/submission | 2026-10-04 |
| claude.com/docs/directory/publish | 2026-10-03 |

No early-October snapshot of claude.com/docs/connectors/building/authentication existed, so that page is
compared only with the 10-01 quotes.

---

## 0. Summary of what changed since early October

### OpenAI

The guidelines, the auth page and the reference are substantively unchanged. Changes are concentrated
in the submission portal:

1. **Submission auth is now declared in the package `mcp.json`.** The portal "doesn't provide an
   advanced OAuth settings editor". Registration is chosen as `provided`, `dcr` or `cimd`.
2. **The portal has no API-key or bearer flow, and rejects non-empty server `headers`.** It also
   rejects `clientSecret` in `mcp.json`. The schema accepts `private_key_jwt` for a provided client,
   but this "needs confirmation with OpenAI".
3. **New `onboardingSkill` manifest field.** A desktop-only (local MCP) path is open to approved
   partners only. Lifecycle hooks must be removed before submission.
4. **No changelog entry since 2026-08-21.** That entry introduced stable OAuth callbacks and CIMD
   client IDs, gated on RFC 9207 `iss`.
5. **Trap still present, not new:** ChatGPT's live CIMD document advertises the legacy singular
   `"token_endpoint_auth_method": "private_key_jwt"`. An authorization server that treats that singular
   field as binding would force `private_key_jwt`.
6. **Codex still applies `fileParams` only to its `codex_apps` server** (verified at openai/codex
   `695961ee`, 2026-11-05). A custom MCP entry in `config.toml` gets no file rewriting.

### Anthropic

1. **Anthropic-held OAuth client credentials (`oauth_anthropic_creds`) are now self-serve.** The type
   table says "Enter the client in the developer portal". On 10-01 it said "Contact
   `mcp-review@anthropic.com`".
2. **Per-customer URLs are now possible.** The connection step offers "Users connect to different
   URLs" (Multiple URLs / URL pattern), but such a listing "can be published only once it has the
   Verified label".
3. **New auth option "Required when the server asks"** for lazy-auth servers.
4. **Ultra added to eligible plans** (Pro, Max, Ultra, Team, Enterprise).
5. **Software Directory Policy marked "Updated over a week ago"**, so it changed in late October. The
   exact delta is **not found (primary)**: the article publishes no changelog.
6. **New API MCP-connector beta header `mcp-client-2026-09-15`**, which adds tool-list pinning. When it
   appeared in the docs was not established.

### Google

The Connected Apps Platform launched in preview on **2026-10-28**. It is the most plausible meaning of
"Google Search apps": one remote-MCP integration published to the Gemini app and Search.

The Gemini Enterprise docs were updated 10-26 to 11-04:

- custom MCP server data stores (admin-configured);
- Agent Registry / Agent Gateway;
- A2A registration pinned to **A2A v0.3**, with compat packages for 1.0;
- Cloud Marketplace "AI Agents as a Service (A2A)" listings that appear in the Gemini Enterprise Agent
  Gallery.

### Cross-cutting

**No host hands a user-uploaded file to a third-party MCP tool by reference, except ChatGPT
`openai/fileParams`.** Today's MCP spec work on this has moved out of SEP-2631. The SEP closed
2026-10-14 and continues as an *experimental extension*, PR
`modelcontextprotocol/experimental-ext-file-uploads#1`, open and updated 2026-11-05.

---

## 1. OpenAI

### 1.1 Integration paths and fit

| Path | What it is | Fit for Parse | Quote | URL |
|---|---|---|---|---|
| **ChatGPT + Codex plugin (public directory)** | One plugin ZIP (skills and/or one remote MCP server, optional UI). Submitted at `platform.openai.com/plugins` and listed in one directory shared by ChatGPT and Codex | **Primary** | "users can find your plugin in the universal directory shared by ChatGPT and Codex" | https://developers.openai.com/plugins/deploy/app-review.md |
| Custom MCP server (developer mode / unlisted) | Users add a server URL themselves at chatgpt.com/plugins | Testing; private/workspace use | "Supported MCP protocols: SSE and streaming HTTP." / "For private or workspace-only use, use a custom MCP server connection instead." | https://developers.openai.com/api/docs/guides/custom-mcp-server.md ; app-review.md |
| Codex CLI/IDE MCP | `config.toml` `mcp_servers.<id>.url` ("Endpoint for an MCP streamable HTTP server") plus OAuth/bearer keys | Developer users | `mcp_servers.<id>.bearer_token_env_var`: "Environment variable sourcing the bearer token for an MCP HTTP server." | https://learn.chatgpt.com/docs/config-file/config-reference.md |
| Codex plugin marketplace (own) | `.codex-plugin/plugin.json` + `.mcp.json`; repo or personal `marketplace.json` | Developer distribution without review | "`$REPO_ROOT/.agents/plugins/marketplace.json` for a repo-scoped list or `~/.agents/plugins/marketplace.json` for a personal list" | https://developers.openai.com/plugins/build/plugins.md |
| Responses API remote MCP tool | Developer passes `server_url` + `authorization` per request | API customers | "Set `server_url` for a remote MCP server, or use `tunnel_id` for a local MCP server through Secure MCP Tunnel." | https://developers.openai.com/api/docs/guides/tools-connectors-mcp.md |
| Agents API (managed) MCP | Sessions connect to MCP, with credentials from inline auth or a **vault** | API customers | "Store MCP credentials in a vault and attach it through `vault_ids`. Vault-backed MCP authentication applies only to connections from OpenAI." | https://developers.openai.com/api/docs/guides/agents-api/tools/mcp.md |
| Agents SDK | `HostedMCPTool` (Responses calls the server) or client-side `MCPServerStreamableHttp` / `MCPServerSse` / `MCPServerStdio` | API customers | "Let OpenAI's Responses API call a publicly reachable MCP server on the model's behalf — Hosted MCP server tools via HostedMCPTool" | https://openai.github.io/openai-agents-python/mcp/ |
| Desktop-only plugin (local MCP) | stdio MCP in the plugin | Not for the hosted service | "Local MCP submissions are limited to approved partners." | https://developers.openai.com/plugins/deploy/desktop-only.md |
| Sign in with ChatGPT (in the connector OAuth) | Optional "Continue with ChatGPT" inside our own OAuth flow | Optional, gated | "Sign in with ChatGPT is currently available to selected commercial partners through a limited trial." | https://developers.openai.com/siwc/chatgpt-plugin.md |
| A2A | — | — | **not found (primary)**: no OpenAI product page documents A2A agent registration | — |

### 1.2 Auth requirements

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| OAuth 2.1 per MCP authorization | Protected-resource metadata (PRM), AS metadata, PKCE S256, `resource` echoed | "Host protected resource metadata on your MCP server" / "Echo the `resource` parameter throughout the OAuth flow" / "`code_challenge_methods_supported`: must include `S256`." | https://developers.openai.com/plugins/build/auth.md |
| Client registration | CIMD preferred; DCR still supported; or a predefined client | "ChatGPT prioritizes CIMD when it is available, but the plugin builder can choose DCR when both CIMD and DCR are available." | same |
| CIMD token-endpoint methods | `none` or `private_key_jwt`; chosen from the intersection | "For CIMD, ChatGPT supports `none` for public-client token exchange and `private_key_jwt` for signed client assertion token exchange." / "ChatGPT selects a method from the intersection of those sets." | same |
| **Live CIMD document** | Stable client ID; legacy singular field = `private_key_jwt` | `{"client_id":"https://chatgpt.com/oauth/client.json", "redirect_uris":["https://chatgpt.com/connector_platform_oauth_redirect"], "token_endpoint_auth_method":"private_key_jwt", "token_endpoint_auth_methods_supported":["none","private_key_jwt"], … "jwks_uri":"https://chatgpt.com/oauth/jwks.json"}` | https://chatgpt.com/oauth/client.json (fetched 2026-11-05) |
| RFC 9207 `iss` (unchanged since 10-06) | Required for the stable callback / client ID; a mismatch is fatal | "Return `iss` in every successful and error authorization response. Its value must exactly match the metadata `issuer`" / "If the server advertises issuer identification but omits `iss` or returns a mismatch, ChatGPT and Codex reject the response." | https://developers.openai.com/plugins/build/auth.md |
| Redirect URIs | Two modes | without `iss`: "`https://chatgpt.com/connector/oauth/{callback_id}`"; with it: "`https://chatgpt.com/connector_platform_oauth_redirect`" | same |
| Tool-level `securitySchemes` + runtime challenge | Both halves are needed to show the linking UI | "Triggering the tool-level OAuth flow requires both metadata (`securitySchemes` and the resource metadata document) **and** runtime errors that carry `_meta["mcp/www_authenticate"]`." | same |
| Mixed auth (dev mode) | Unauthenticated initialize and tools/list | "the initialize and list tools APIs use no auth, and tools use OAuth or no auth based on the security schemes set on their tool metadata." | https://developers.openai.com/api/docs/guides/custom-mcp-server.md |
| Workspace domain restriction (Enterprise) | OIDC discovery + UserInfo with `email_verified` | "the UserInfo Endpoint is required for workspace domain restrictions." | https://developers.openai.com/plugins/build/auth.md |
| Multi-account / profile tool (optional) | `_meta["openai/profile"]: true` tool returning a stable `id` | "If you declare a profile tool, return a valid identity; an invalid response can prevent account connection." | same |
| **Submission auth (changed)** | Declared in `mcp.json`; `provided` / `dcr` / `cimd` | "The portal reads advanced authentication settings from your package's `mcp.json`." / "Client Identifier Metadata Document (`cimd`) — Configure CIMD preference. The runtime can fall back to DCR according to its registration policy." | https://developers.openai.com/plugins/deploy/submission.md |
| **No API key / static header (changed)** | — | "the portal doesn't provide an API-key connection flow or a bearer-token input." / "public submission rejects nonempty server headers." / "Don't put the client secret in `mcp.json`; the manifest rejects `clientSecret`." | same |
| Responses API | Caller supplies the token | "Provide this token using the `authorization` field of the MCP tool" | https://developers.openai.com/api/docs/guides/tools-connectors-mcp.md |
| Codex CLI | OAuth (stored), bearer env var, static or env headers, pre-registered client | `mcp_servers.<id>.auth`: "`oauth` (default) uses stored MCP OAuth credentials when available." / `oauth.callback_url`: "Pre-registered clients reuse it when issuer identification is supported" | https://learn.chatgpt.com/docs/config-file/config-reference.md |

### 1.3 Listing, publishing, review and costs

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Org identity verification | Individual or business; enforced | "Publishing under an unverified individual or business name will result in rejection." | https://developers.openai.com/plugins/deploy/app-review.md |
| Permissions | Owner, or Apps Management Write | "Organization owners can submit; other members need **Apps Management Write**" | https://developers.openai.com/plugins/deploy/submission.md |
| Data residency | EU-residency projects cannot submit MCP plugins | "For now, projects with EU data residency cannot submit plugins with MCP servers for review. Use a project with global data residency." | app-review.md |
| Public HTTPS + CSP | — | "Your MCP server is hosted on a publicly accessible domain" / "If the server returns UI, you defined a content security policy (CSP) that allows the exact domains the component fetches from." | app-review.md |
| Domain verification | Plain-text token | "host the exact challenge token as plain text at the URL shown in the portal: `https://<challenge-base-host>/.well-known/openai-apps-challenge`" / "Return only the exact token—not JSON or a list of tokens." | submission.md |
| One MCP server per plugin | — | "**only one MCP server can be connected per plugin**" | submission.md |
| Test cases | 5 positive + 3 negative; demo video required | "Initial MCP app review requires exactly five positive and three negative cases." / `review.demo_recording_url` — "Optional in ZIP; required for MCP review" | submission.md |
| Cases must pass on all surfaces | — | "Ensure that all test cases pass on the supported ChatGPT and Codex surfaces where the plugin will be available." | app-review.md |
| Reviewer account | No MFA, magic links or private network | "should work immediately without MFA approval, email or SMS codes, magic links, or private-network access." | submission.md |
| Listing URLs | All four required for MCP | "all four listing URLs are required: `websiteURL`, `supportURL`, `privacyPolicyURL`, and `termsOfServiceURL`." | submission.md |
| Screenshots | Not shown; example prompts instead | "Screenshots are no longer shown in the Directory, instead provide example Prompts" | https://developers.openai.com/plugins/plugin-guidelines.md |
| Package limits | No hooks or app references | "Plugin ZIP packages containing app references (`apps` / `.app.json`) or lifecycle hooks cannot currently be submitted." | submission.md |
| Annotations | Explicit booleans; scanned values are binding | "Set `readOnlyHint`, `destructiveHint`, and `openWorldHint` to explicit boolean values" / "They don't override the annotations." | plugin-guidelines.md ; app-review.md |
| Note for Parse | `readOnlyHint:false` if the tool persists artifacts or uploads files | "Use `false` for external-state changes, persisting artifacts, starting stateful jobs or workflows, queuing work, or write-style outbound actions such as posting messages, sending emails, or uploading files." | plugin-guidelines.md |
| Commerce | Physical goods only; no upsell | "Currently, plugins may conduct commerce only for physical goods. Selling digital products or services—including subscriptions, digital content, tokens, or credits—is not allowed" / "Plugins must not display subscription plans, initiate new subscriptions, or promote upgrades." | plugin-guidelines.md |
| Audience | 13+ | "Plugins must be suitable for general audiences, including users aged 13–17." | plugin-guidelines.md |
| Data minimisation | Strip IDs, timestamps and traces | "remove any unnecessary PII, telemetry/internal identifiers (for example, session, trace, or request IDs; timestamps; internal account IDs; or logs)" | app-review.md |
| Iframes | Own registrable domain; justification required | "Plugins with UI can embed pages from their MCP server’s own registrable domain" / "You must still provide a justification when submitting a plugin that uses iframes." | plugin-guidelines.md |
| Continuous review after publication | Tools are re-scanned and new tools gated | "OpenAI periodically fetches your MCP server's tools and compares them with the published definitions" / "New tools: Made available after they pass automated checks." | app-review.md |
| Origin is fixed | — | "The MCP server origin (`scheme`, `hostname`, or `port`) can't change between versions." | app-review.md |
| Review time | Unstated | "Review timelines may vary as we continue to build and scale our processes." | app-review.md |
| Claude listing reuse | None | "Claude marketplace listings and approvals don't transfer." | https://developers.openai.com/plugins/guides/submit-claude-plugin.md |
| Cost / fee | — | **not found (primary)**: no submission fee is documented | — |

### 1.4 File handling

| Mechanism | Detail | Quote | URL |
|---|---|---|---|
| `_meta["openai/fileParams"]` (listed and dev-mode plugins) | ChatGPT hands the tool a download URL | "To let ChatGPT pass files to a tool, list each top-level file input in `_meta["openai/fileParams"]`." / "ChatGPT always includes `download_url` and `file_id`; it may omit `mime_type` and `file_name`." | https://developers.openai.com/plugins/reference.md |
| Schema strictness | 4 properties; 2 required | "The Scan Tools step and plugin submission reject a file schema that omits any of the four properties" (unchanged since 10-01) | same |
| Widget file APIs | `uploadFile`, `selectFiles`, `getFileDownloadUrl` | "`window.openai.getFileDownloadUrl({ fileId })` … Retrieve a temporary download URL for a file uploaded by the widget, selected from the file library, passed via file params, or returned by tool file references." | same |
| `download_url` expiry / size limit | — | **not found (primary)** | — |
| Codex CLI | `fileParams` rewriting only for the `codex_apps` server | `prepare_openai_file_params_for_model(tool)` is called only inside `prepare_codex_apps_tools_for_model` | openai/codex @ `695961ee` `codex-rs/codex-mcp/src/rmcp_client.rs` L891–897 |
| Responses / Agents API | Files go wherever the developer passes them; no MCP file hand-off | **not found (primary)** | — |

---

## 2. Anthropic

### 2.1 Integration paths and fit

| Path | What it is | Fit for Parse | Quote | URL |
|---|---|---|---|---|
| **Connectors Directory (MCP connector)** | Remote MCP server submitted at `claude.ai/directory/manage`. One catalog for claude.ai, Desktop, mobile, Claude Code and Cowork | **Primary** | "You submit a remote MCP server to the Connectors Directory through the developer portal at claude.ai/directory/manage, where you choose **MCP connector**." | https://claude.com/docs/connectors/building/submission.md |
| **Plugin bundle (directory)** | Skills, commands and agents plus a reference to the same MCP URL. Taken from a GitHub repo and reaches Claude Code via account sync | **Secondary** (submit both) | "For your own product, make two submissions: the server as an MCP connector, then the plugin bundle whose skills teach Claude how to use it." | https://claude.com/docs/directory/publish.md |
| Claude Code: own marketplace | `.claude-plugin/marketplace.json` in a repo; `/plugin marketplace add` | Developer distribution, no review | "You can run your own marketplace or submit your plugin to Anthropic's directory." | https://code.claude.com/docs/en/plugins/publish.md |
| Claude Code: `claude-plugins-official` | Curated; **no** portal intake | Partner contact only | "Anthropic's official marketplace, `claude-plugins-official`, doesn't take submissions through the directory portal. If you work with an Anthropic partner contact, ask them about an official-marketplace listing." | same |
| Claude Code: `claude-community` marketplace | Third-party plugins that authors submitted | — | "Third-party plugins that their authors submitted to Anthropic" | https://code.claude.com/docs/en/plugins/anthropic-marketplaces.md |
| Custom connector (unlisted) | A user or org adds the URL | Testing, private use | "if you have a connector's URL, you can add it as a custom connector whether or not it's in the directory." | https://claude.com/docs/connectors/verification.md |
| Claude API MCP connector | `mcp_servers` + `mcp_toolset`, beta | API customers | betaHeader "`mcp-client-2026-09-15, mcp-client-2025-11-20`" / "Of the feature set of the MCP specification, only tool calls are currently supported." | https://platform.claude.com/docs/en/agents-and-tools/mcp-connector.md |
| Managed Agents | MCP declared on the agent; credentials from a vault | API customers | "Declare MCP servers on the agent. Each session supplies credentials by referencing a vault, so secrets stay out of the agent definition." | same |
| Local MCPB in directory | **Deprecated** | No | "desktop extension listings in the directory are deprecated, and the directory no longer accepts local servers packaged as MCP Bundles (MCPB)." | submission.md |
| A2A | — | — | **not found (primary)** | — |

### 2.2 Auth requirements

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Policy | OAuth 2.0 when authenticated | "Remote MCP servers that connect to a remote service and require authentication must use secure OAuth 2.0 with certificates from recognized authorities." (§5D) | https://support.claude.com/en/articles/13145358-anthropic-software-directory-policy |
| Types | dcr, cimd, anthropic_creds, custom_connection, static_headers, none | `oauth_anthropic_creds` — "**Supported. Enter the client in the developer portal**" (**changed**: was "Contact `mcp-review@anthropic.com`" on 10-01) / `static_headers` — "Beta, for a limited set of organizations" | https://claude.com/docs/connectors/building/authentication.md |
| CIMD selection rule | Needs `none` in `token_endpoint_auth_methods_supported` | "Claude normally selects CIMD only when your authorization server metadata advertises both `"client_id_metadata_document_supported": true` and `"none"` in `token_endpoint_auth_methods_supported`, and otherwise falls back to DCR." | same |
| CIMD confidential variant | `private_key_jwt` | "If your metadata lists `private_key_jwt`, Claude can instead present a second metadata document, whose URL ends in `/confidential_client`, and authenticate with a signed client assertion." | same |
| 401 is mandatory | — | "A `401` is required to start sign-in, and Claude ignores a `WWW-Authenticate` header on a `200` response" | same |
| First AS only | — | "Claude uses only the first entry in your metadata's `authorization_servers` list" | same |
| Redirects | Hosted apps vs Claude Code loopback | "`https://claude.ai/api/mcp/auth_callback`" / "Claude Code declares `http://localhost/callback` and `http://127.0.0.1/callback` in its Client ID Metadata Document, so match both with the port component ignored." | same |
| Refresh | Rotate for public clients | "Rotate refresh tokens for public-client connections." / "Claude also appends `offline_access` when your authorization server metadata lists it in `scopes_supported`" | same |
| Latency | 10 s / 30 s | "Claude waits up to 10 seconds for a response from your OAuth discovery, registration, and token endpoints, and up to 30 seconds for refresh token requests." | same |
| No M2M | — | "A machine-to-machine `client_credentials` grant isn't supported" | same |
| RFC 9207 `iss` | — | **not found (primary)**: Claude's pages don't require or mention the authorization-response `iss` | — |
| Lazy auth | Portal option (**new**) | "Choose **Required when the server asks** if your server starts without sign-in and only some tools need it." | submission.md |
| Enterprise Managed Auth | Optional auto-connect via IdP assertions | "Support auto-connect in Claude by accepting identity assertions from a customer's identity provider or from Claude, so users skip the OAuth consent step." | https://claude.com/docs/connectors/building/enterprise-managed-auth.md |
| API connector | Caller supplies the token; ZDR not eligible | "OAuth authentication: Support for OAuth Bearer tokens for authenticated servers" / zdr "eligibility: not-eligible" | mcp-connector.md |

### 2.3 Listing, publishing, review and costs

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Who can submit (cost = a paid plan) | Ultra newly listed | "**Plan**: Pro, Max, Ultra, Team, or Enterprise. Free accounts can't submit" / "there's no partner program to apply to first" | https://claude.com/docs/directory/publish.md |
| Default outcome | Community after an automated scan | "Anthropic scans your submission automatically for policy compliance and, by default, lists it as a Community connector with no action from you." | submission.md |
| Verified | Escalation only | "Anthropic may then escalate listings flagged as highly useful to Claude users to Verified review, which is higher touch and slower, and in which reviewers run a functional test of each tool." / "There's no separate application for the **Verified** label" | https://claude.com/docs/connectors/verification.md |
| Per-customer URLs (**new**) | Verified-gated | "A listing that uses an option under **Users connect to different URLs** can be published only once it has the **Verified** label." | submission.md |
| Annotations | Title + hint on every tool | "Every tool must include a `title` and the applicable hint: `readOnlyHint: true` for read-only tools, and `destructiveHint: true` for tools that modify or delete data." | https://claude.com/docs/connectors/building/review-criteria.md |
| Test account | Fully populated | "**Test credentials**: required, and they must be for a fully populated account" / Policy §3D "standard testing account with sample data" | review-criteria.md ; policy |
| Example prompts | ≥ 3 | "Developers must provide at least three working examples of prompts or use cases that demonstrate core functionality." (§3E) | policy |
| Ownership | Own every endpoint | "Developers must verify that they own or control any API endpoint, domain, or user interface their Software connects to" (§3F) | policy |
| Public docs | By the publish date | "**Public documentation**: required by your publish date." | review-criteria.md |
| Portal steps | 10 steps; 7 acknowledgments | "Seven policy acknowledgments covering the directory guidelines, first-party API usage, financial transactions, AI media generation, prompt injection, conversation data collection, and public documentation." | submission.md |
| MCP App screenshots | 3–5 PNG, ≥ 1000 px | "**Count**: 3–5 images" / "**Video or GIF**: not accepted" | submission.md |
| Tool names ≤ 64; Streamable HTTP | — | "MCP tool names must not exceed 64 characters." (§5C) / "Remote MCP servers should support the Streamable HTTP transport." (§5F) | policy |
| **User-file rule (risk for Parse)** | — | "Software must not query or extract data from Claude’s memory, chat history, conversation summaries, or user-generated or uploaded files" (§1G) / "Don't query Claude's memory, chat history, conversation summaries, or user files" | policy ; review-criteria.md |
| Unsupported | AI media generation | "Generate images, video, or audio through AI models" | review-criteria.md |
| Plugin bundle | GitHub repo; human review of new listings | "every version gets automated validation and a security scan, and a person reviews a new listing before it goes live" / repo "must be public before the listing goes live" | publish.md ; https://claude.com/docs/plugins/submit.md |
| Escalations | — | "For escalations, email `mcp-review@anthropic.com`." | submission.md |
| Review time | — | "Review time isn't fixed." | publish.md |

### 2.4 File handling

| Mechanism | Detail | Quote | URL |
|---|---|---|---|
| Host → connector file hand-off | **not found (primary)**: unchanged since 10-07 | Exposes "Text and binary resources"; unsupported: "Resource subscriptions · Sampling · Advanced or draft capabilities" | https://claude.com/docs/connectors/building/index.md |
| Limits | 150k chars / 240 s | "Maximum tool result size \| ~150,000 characters" / "Tool call timeout \| 240 seconds per tool call" | same |
| Large result → sandbox | — | "When a tool result exceeds approximately 150,000 characters and Claude's code execution sandbox is active, Claude writes the result to the sandbox filesystem" | https://claude.com/docs/connectors/building/mcp-apps/troubleshooting.md |
| MCP Apps file input | Spec has `ui/download-file` only; no upload API | `ui/download-file` - "Request host to download a file" | https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/draft/apps.mdx (main @ `19256f52`, 2026-11-04) |
| MCP spec status | SEP-2631 closed; experimental extension | "The File Uploads Working Group is continuing this work as an experimental extension" (localden, 2026-10-14) | https://github.com/modelcontextprotocol/modelcontextprotocol/pull/2631 ; https://github.com/modelcontextprotocol/experimental-ext-file-uploads/pull/1 |

---

## 3. Google Gemini Enterprise (formerly Agentspace)

Naming as of today:

- Vertex AI Agent Engine is now **"Agent Runtime on Gemini Enterprise Agent Platform"**.
- Vertex AI Search is now **"Agent Search"**.
- Agent Designer is now **"Workflow Builder"**.

Admin surfaces are in the Google Cloud console, Gemini Enterprise page, under each app's **Agents**
and **Connected Data Stores** sections.

### 3.1 Integration paths and fit

| Path | What it is | Fit for Parse | Quote | URL (last updated) |
|---|---|---|---|---|
| **Custom MCP server data store** | An admin adds a remote MCP server as a connector; tools become "actions" | **Best fit (reuses the MCP server)** | "The custom MCP data store lets you integrate public and private tools, as well as MCP-compliant third-party systems, directly into Gemini Enterprise." | https://docs.cloud.google.com/gemini/enterprise/docs/connectors/custom-mcp-server/set-up-custom-mcp-server (10-30) |
| MCP server via Agent Registry | Registry entry added as a data store; Agent Gateway governs egress | Enterprise governance | "You can add MCP servers from Agent Registry as data stores for your Gemini Enterprise apps." | …/connectors/custom-mcp-server/import-govern-mcp-server-agent-registry (11-04) |
| **A2A agent registration** | An admin pastes the Agent Card or calls `agents.create` | Fit if Parse ships an A2A façade | "click Add for Custom agent via A2A. In the Agent card JSON field, enter the agent card details in JSON format." | https://docs.cloud.google.com/gemini/enterprise/docs/register-and-manage-an-a2a-agent (10-30, **Preview**) |
| A2A + A2UI | Rich UI via the A2UI extension | Optional | "Gemini Enterprise supports A2UI versions v0.9 and v0.8." | …/a2ui-agents/register-and-manage-an-a2ui-agent (10-30) |
| ADK on Agent Runtime | Agent hosted on Google's runtime, registered by an admin | Only if we host on Agent Runtime | "For ADK agents that are deployed on Agent Runtime, you can use a single Agents CLI in Agent Platform command to register the agents with Gemini Enterprise." | …/register-and-manage-an-adk-agent (10-30) |
| **Cloud Marketplace "AI Agents as a Service (A2A)"** | A paid or free listing; customers procure it, then add it to their app; shows in the Agent Gallery "Marketplace" chip | Public distribution route | "After your product is published publicly, its listing becomes visible in both Cloud Marketplace and Gemini Enterprise." | https://docs.cloud.google.com/marketplace/docs/partners/ai-agents/publish (10-26) |
| Marketplace SaaS (non-A2A) | An MCP or REST product sold as SaaS with Google billing | Alternative commercial path | "If your AI agent uses a different protocol (not A2A), you can offer it as a SaaS product with integrated billing." | https://docs.cloud.google.com/marketplace/docs/partners/ai-agents (10-26) |
| Agent Gallery | End-user catalog; Marketplace chip = partner agents to request from an admin | Discovery only | "Marketplace: Third-party partner agents available to request from your administrator." | …/agents-gallery (10-30) |
| Custom connector (data ingestion) | Fetch → Transform → Sync into data stores | Not a tool path; possible as a pre-parse pipeline | "Custom connectors work by using an automated data pipeline to perform three key actions: Fetch, Transform, and Sync." | …/connectors/custom-connector (10-30) |
| Built-in partner MCP connectors (e.g. Atlan, Ramp, Pendo) | Google-listed connector pages | How a third party becomes one: **not found (primary)** | — | …/connectors/atlan-mcp (10-28) |

### 3.2 Auth models

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Custom MCP: OAuth 2.0 with a **pre-registered client** (client ID + secret required; no DCR or CIMD documented) | The admin registers GE as an OAuth client at the IdP | "Register Gemini Enterprise as an OAuth client application with your identity provider" / "Set the authorization redirect URL to `https://vertexaisearch.cloud.google.com/oauth-redirect`." | set-up-custom-mcp-server |
| Custom MCP: PKCE optional; Basic auth default | — | "Enable PKCE Support — Optional." / "Use HTTP Basic Authentication — Optional. Selected by default." | same |
| Custom MCP: transport and TLS | Streamable HTTP only; public CA | "The older SSE transport isn't supported." / "Configure custom MCP servers with a TLS certificate signed by a publicly trusted Certificate Authority" | same |
| Custom MCP: org policy | FQDN allowlist + constraint override | "Explicitly allow the fully qualified domain names (FQDNs) for your custom MCP server URL, authorization URL, and token URL." | same |
| Custom MCP: limits | 100 actions; app-scoped | "limit the number of enabled actions in your custom MCP server data store to 100 at a time." | same |
| Cloud Run IAM (service agent) | ID token only on `.run.app` | "Gemini Enterprise automatically sends the Google-signed ID token only when you connect to a default Cloud Run URL that ends in .run.app." / service agent `service-PROJECT_NUMBER@gcp-sa-discoveryengine.iam.gserviceaccount.com` needs `roles/run.invoker` | same ; register-and-manage-an-a2a-agent |
| A2A: end-user OAuth | Via an `authorizations` resource (`serverSideOauth2`: clientId, clientSecret, authorizationUri, tokenUri) | redirect URIs "`https://vertexaisearch.cloud.google.com/oauth-redirect`" and "`https://vertexaisearch.cloud.google.com/static/oauth/oauth.html`" | register-and-manage-an-a2a-agent |
| A2A: card security schemes ignored | — | "Gemini Enterprise doesn't use the `securitySchemes` and `security` fields of your agent card to authenticate with your agent." | same |
| A2A: no API keys or custom headers | — | "Gemini Enterprise doesn't support the `apiKey` security scheme, and doesn't provide a way to add static API keys or other custom header values" | same |
| A2A: agents behind Apigee | OAuth mandatory | "Configuring OAuth 2.0 is required when securing an Apigee API proxy" | same |
| ADK: user identity | Email passed | "ADK agents receive the user's email address from Gemini Enterprise." | register-and-manage-an-adk-agent |
| Admin roles | GE Admin; Discovery Engine Editor | "The Gemini Enterprise Admin role." / "Grant the Discovery Engine Editor role (`roles/discoveryengine.editor`)." | A2A page ; custom-MCP page |
| Model Armor | The developer's responsibility | "To secure A2A agents, developers must configure Model Armor using the REST API within their agent's application code." | A2A page |

### 3.3 A2A Agent Card requirements

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Protocol version | **v0.3** (current spec is 1.0.1) | "Gemini Enterprise supports the A2A v0.3 streaming mechanism. If you use A2A v1.0.0 or later, then use the compatibility packages provided by the SDK" (e.g. `a2a.compat.v0_3`) | register-and-manage-an-a2a-agent |
| Example card fields | `protocolVersion` "0.3", name, description, url, version, capabilities, skills, defaultInput/OutputModes | `"protocolVersion": "0.3", … "url": "https://example.com/myagent"` | same |
| Spec releases | v1.0.0 2026-03-12; v1.0.1 2026-05-28 | 1.0 uses `supportedInterfaces[{protocolBinding, protocolVersion, url}]`; clients "MUST send the `A2A-Version` header … (except for 0.3 Clients - 0.3 will be assumed for empty header)" | https://github.com/a2aproject/A2A (`docs/specification.md`, releases API) |
| Card discovery | Well-known URI | "`https://{server_domain}/.well-known/agent-card.json`" | same spec |
| Extensions | A2UI v0.8/v0.9 supported; others **not found (primary)** | (see 3.1) | a2ui page |
| Card signatures | Optional in the spec (JWS); GE requirement **not found (primary)** | "Agent Cards **MAY** be digitally signed using JSON Web Signature (JWS)" | spec |
| Marketplace card validation | Stored in Cloud Storage; scored | "Upload your Agent Card as a JSON file to a Cloud Storage bucket with Object Versioning enabled" / "consult the Agent ScoreCard guide provided to you." (the guide is not public: **not found (primary)**) | https://docs.cloud.google.com/marketplace/docs/partners/ai-agents/agent-card |
| Quota | Registration consumes the daily agent-creation quota | "Registering an agent with an app consumes one unit of your project's pooled daily agent creation quota" | A2A page |

### 3.4 Listing, publishing and costs (Cloud Marketplace)

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Vendor onboarding | Vendor sign-up + Project Info Form → Producer Portal | "complete the Cloud Marketplace Project Info Form, provided by the Cloud Marketplace team." | ai-agents |
| A2A agent rules | Model Garden default; A2A | "Use Google foundation or 3rd party models hosted in Model Garden as a default configuration." / "Support multi-agent interoperability through the Agent2Agent (A2A) protocol." | ai-agents |
| Technical integration | Pub/Sub + Partner Procurement API (+ Service Control for usage) | "Integrate with the Partner Procurement API." | …/ai-agents/technical-integration |
| Reviews | Product details, pricing, technical integration, card validation | "you must have successfully completed the pricing review before you submit your Technical integration tab for review." | …/ai-agents/publish |
| Mandatory AI metadata | Industry, business function, category | "you must provide the following metadata fields for AI agents: Industry … Business function … Category" | …/ai-agents/ai-agent-metadata |
| Pricing | Free / subscription / usage / combined | "Free: Customers only pay for the Google Cloud resources that they use." | ai-agents |
| Google revenue share | 3% standard (97% vendor net) | "Standard Offer (i.e., non-Private Offer transactions) … 97%" (schedule effective April 21, 2025) | https://cloud.google.com/terms/marketplace-revenue-share-schedule |
| Customer side | Procure, then add | "After an agent is procured (purchased) and added to Gemini Enterprise, users can access the agent" (requires "The Consumer Procurement Entitlement Viewer role") | …/register-and-manage-marketplace-agents (Preview) |

### 3.5 File handling

| Mechanism | Detail | Quote | URL |
|---|---|---|---|
| A2A FilePart in GE's A2A proto | Bytes or URI | "FilePart represents the different ways files can be provided. If files are small, directly feeding the bytes is supported via file_with_bytes. If the file is large, the agent should read the content as appropriate directly from the file_with_uri" | https://docs.cloud.google.com/gemini/enterprise/docs/reference/rpc/a2a.v1 |
| Whether GE forwards chat attachments to A2A, ADK or MCP tools | — | **not found (primary)** | — |
| Agent Search "Bring your own parsed document" (Preview) | Upload pre-parsed layout JSON (text, table and list blocks) into a data store. **A natural fit for Parse output.** | "instead of importing a raw PDF document, you can parse the PDF yourself and import the parsing result instead." | https://docs.cloud.google.com/generative-ai-app-builder/docs/parse-chunk-documents (10-26) |

### 3.6 What the admin must do in the console

1. **Custom MCP:**
   - Override the org-policy constraint, allow the FQDNs, and add `custom_mcp` to the allowed sources
     under VPC Service Controls.
   - Grant `roles/discoveryengine.editor`.
   - Register a GE OAuth client at our authorization server.
   - Apps → *app* → Connected Data Stores → New data store → "Custom MCP Server". Enter the URL,
     authorization URL, token URL, client ID/secret and scopes, then "Verify Auth".
   - Enable the actions: "By default, no tools or actions from your custom MCP servers are enabled."
2. **A2A:** Apps → *app* → Agents → Add Agents → "Custom agent via A2A" → paste the card → OAuth
   details or "Skip & Finish". Then share the agent with users or groups on the agent's User
   permissions tab.
3. **Marketplace:** Procure in Cloud Marketplace, then Agents → Add agent → "Agents via Marketplace".

All quotes and URLs are given in 3.1–3.4.

---

## 4. "Google Search apps": what it refers to

Most plausible referent: **Connected Apps in Search (AI Mode) and the Gemini app.** These are published
through Google's new **Connected Apps Platform**, launched in preview on 2026-10-28. A single
remote-MCP integration reaches both products. The other candidates are covered in 4.4.

### 4.1 Connected Apps Platform: paths and fit

| Requirement | Detail | Quote | URL (last updated) |
|---|---|---|---|
| What it is | Gemini app + Search, via MCP or Android AppFunctions | "The Connected Apps Platform lets you bring your apps and services directly into Google products like the Gemini app and Search" | https://developers.google.com/connected-apps |
| Status | Phased preview, self-service console | "We're opening access in phases, and will add more developers, markets, and Google products over the coming months." / "During the preview phase, we offer self-service onboarding to the Connected Apps Platform through Google Connected Console." | https://blog.google/innovation-and-ai/technology/developers-tools/connected-apps-one-place/ (Oct 28, 2026) ; overview |
| Search availability (users) | US English AI Mode | "This feature is currently available in AI Mode in the US, in English." | https://support.google.com/websearch/answer/17265703 |
| Gemini custom MCP apps (users, Developer Mode) | Personal accounts, 18+, US, English | "Be 18 or over and in the US." / "this feature isn't available if you sign in with a work or school Google Account." (quoted via fetch tool) | https://support.google.com/gemini/answer/17209137 |
| Transport | Streamable HTTP; one global endpoint | "Your MCP server must be publicly reachable over HTTPS using the Streamable HTTP transport" / "Each connected app supports a single global MCP server endpoint. Separate data center URLs are not supported." | https://developers.google.com/connected-apps/gpc/prerequisites (10-30) |
| Domain verification | JSON token | "serve a JSON challenge token file directly (with HTTP 200 OK and no redirects) at `https://<your-domain>/.well-known/google-connected-apps`." | same |
| Category fixed | Immutable | "Once a category for a connected app has been selected, it cannot be changed." | same |
| Categories | Music, Content creation, Grocery, Real estate, Local services, Events, **Productivity** ("cloud storage"), CRM, Small business finance. Parse fits Productivity or Content creation; eligibility **not found (primary)** | Content creation: "formatted documents (proposals, reports, resumes)" | https://developers.google.com/connected-apps/mcp/vertical-journeys (10-28) |
| A2A | **Prohibited** | "All agent-to-agent interactions are strictly prohibited." | https://developers.google.com/connected-apps/connected-apps-policy (11-02) |
| MCP Apps (widgets) | Beta; size and CSP limits | "MCP Apps for Connected Apps is in beta and may not be available." / "A single UI resource must not exceed 15 MB in size, and cumulative UI resources across your server must not exceed 100 MB." | …/gpc/perform-the-steps (10-28) ; …/mcp/server-optimization (10-27) |

### 4.2 Auth requirements

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Console auth models | None / M2M client credentials / user-delegated authorization code | "Machine-to-machine (OAuth Client Credentials)" / "User-delegated (Authorization Code): … using Google Account Linking" | perform-the-steps (10-28) |
| Client registration (console) | Pre-registered, or CIMD/DCR | "Google checks for CIMD first and falls back to DCR using your .well-known discovery endpoints." | same |
| **Conflict: OAuth guide** (older, 09-14) | DCR endpoint required even with CIMD | "Even if your server supports Client ID Metadata Documents, the Gemini app … currently requires a dedicated registration_endpoint to be present in the discovery JSON." | https://developers.google.com/connected-apps/mcp/oauth (09-14) |
| PKCE | Mandatory for OAuth 2.1 | "PKCE is mandatory for all OAuth 2.1 and optional for OAuth2.0." | same |
| Redirect allowlist | 6 patterns | "`https://oauth-redirect.googleusercontent.com/r/*`" … "`https://oauth-redirect-test.googleusercontent.com/a/*`". The console also shows app-specific URIs | same ; perform-the-steps |
| **Refresh = persistent (conflicts with Claude)** | Don't rotate | "to implement Persistent Linking, you must avoid rotating the Refresh Token during this exchange." (Claude: "Rotate refresh tokens for public-client connections.") | same ; Claude auth page |
| Revocation | Optional endpoint; must invalidate | "When a user unlinks or disconnects your integration … we will make a POST request to the /revoke endpoint" | same |
| Discovery | First response wins | "Gemini MCP console uses the first URL that returns a response—including an error." | same |
| IP allowlisting (if used) | Google ranges | "If your server restricts external access by source IP, allowlist the following IP addresses" | prerequisites |

### 4.3 Listing, review and costs

| Requirement | Detail | Quote | URL |
|---|---|---|---|
| Access | Preview application; ToS accepted by an admin | "The Connected Apps Terms of Service must also be accepted by an account owner or administrator." | prerequisites |
| Listing assets | Description < 30 words; 512×512 icon; ≥ 3 capabilities; ≥ 3 example prompts (2 starting with `@App`); ToS + privacy URLs | "Example prompts: At least 3 sample queries … Start at least 2 example prompts with the name of your connected app." | prerequisites |
| Test account | Seeded, no 2FA, sent to the Google rep | "seeded test credentials, pre-populated with realistic sample data and with two-factor authentication (2FA) disabled, along with testing instructions to your Google representative." | perform-the-steps |
| Tool snapshot | Frozen per version | "When you publish, the most recent snapshot of your tools is used for that version of your connected app. You'll no longer be able to make changes to your tools" | same |
| Static tools | No dynamic dispatch | "Connected Apps must statically declare all callable tool operations in the console at registration time." | policy |
| HITL | `destructiveHint` or console confirmation | "must have destructiveHint: true set under tool annotations or have the operation marked as required confirmation (HITL) in the console." | policy |
| Testers | Gmail accounts only | "You can only add testers who have an @gmail.com account." | perform-the-steps |
| Targeting | Countries, languages, products (Gemini / Search) | "Products: Select the Google products where your connected app will be available." | same |
| Review | Send for review → Issues found / Ready to publish | "Before your connected app can be made available to all users, Google reviews and tests your integration." | same |
| Data deletion docs | Required | "you must provide user help documentation that explains how users can manage and delete their data from your app or service." | policy |
| Cost / fee | — | **not found (primary)** | — |

### 4.4 Other candidates

| Candidate | Can a third party plug in? How? | Quote | URL |
|---|---|---|---|
| **Gemini app connected apps / custom apps** | Yes, via the Connected Apps Platform above (listed), or a user-added custom MCP URL in Developer Mode (personal, US, 18+) | "Gemini Apps Developer Mode is an interactive, web-based debugging and evaluation workspace" / "click + Create new to connect an HTTPS MCP server endpoint as a custom MCP app" | https://developers.google.com/connected-apps/mcp/dev-mode (10-28) |
| Gems | **not found (primary)**: no third-party tool attachment documented for Gems | — | — |
| **Agent Search apps** (formerly Vertex AI Search / Gen App Builder / Discovery Engine "apps") | Not as a tool. Third parties can feed data stores (custom connector, or **BYO parsed document**). The app owner builds the app; no third-party marketplace for these apps | "Agent Search provides a digital parser, OCR parser for PDFs, and a layout parser. You can also bring your own parsed documents." | https://docs.cloud.google.com/generative-ai-app-builder/docs/parse-chunk-documents (10-26) |
| Google Search AI Mode "apps" | Same as Connected Apps (4.1) | "AI Mode … in the US, in English" | support.google.com/websearch/answer/17265703 |
| **Google Workspace add-ons** | Yes, but they extend Workspace apps and **Workspace Studio flows** (custom steps that can use Gemini), not the Gemini app's tool calls. Published via Workspace Marketplace (public apps reviewed; OAuth verification may apply) | "By extending flows, you let users trigger workflows from your app's events and add your app's functions as steps." / "Public apps can be viewed and installed by anyone who uses the Marketplace. Public apps have additional publishing requirements" | https://developers.google.com/workspace/add-ons/studio (10-13) ; https://developers.google.com/workspace/marketplace/how-to-publish (09-03) |
| File reaching the tool (any Google consumer surface) | — | **not found (primary)**: the Connected Apps docs describe no upload-to-tool mechanism. The Content-creation journeys mention "import a file" but not how | vertical-journeys |

---

## 5. Cross-platform comparison

### 5.1 Protocol

| | OpenAI ChatGPT/Codex | Anthropic Claude | Gemini Enterprise | Google Connected Apps (Gemini app + Search) |
|---|---|---|---|---|
| Tool protocol | Remote MCP (Streamable HTTP; SSE accepted in dev mode) | Remote MCP (Streamable HTTP; SSE legacy) | Custom MCP data store (**Streamable HTTP only**), or **A2A v0.3** agent, or ADK on Agent Runtime | Remote MCP (Streamable HTTP), or Android AppFunctions |
| A2A | not found (primary) | not found (primary) | Yes (v0.3; 1.0 via compat) | **Prohibited** |
| Widget UI | MCP Apps + `window.openai` extras | MCP Apps (web, Desktop, iOS; not Claude Code) | A2UI v0.8/0.9 over A2A | MCP Apps (**beta**) |
| Endpoint shape | One server per plugin; origin fixed | URL, or per-customer URLs (Verified only) | Per-app admin config; 100 actions max | One global endpoint per app |

### 5.2 Auth

| | OpenAI | Anthropic | Gemini Enterprise | Connected Apps |
|---|---|---|---|---|
| Registration | CIMD (preferred) / DCR / provided | DCR / CIMD (needs `none` advertised) / Anthropic-held (self-serve now) / custom / static_headers (beta) | **Pre-registered client ID + secret** (admin-entered); IAM ID token for `.run.app` | Pre-registered or CIMD→DCR (console). The OAuth guide says DCR endpoint is required |
| PKCE | S256 required | S256 always sent | Optional checkbox | Mandatory (OAuth 2.1) |
| `iss` (RFC 9207) | Required for stable callback; mismatch = reject | not found (primary) | not found (primary) | not found (primary) |
| Refresh | not specified | **Rotate** for public clients | `offline_access` / `access_type=offline` recommended | **Do not rotate** (persistent linking) |
| Static key/header | Portal: no | Beta (limited orgs) | No (A2A: "doesn't support the `apiKey` security scheme") | M2M client credentials option |
| Redirect | `chatgpt.com/connector_platform_oauth_redirect` or `/connector/oauth/{callback_id}` | `claude.ai/api/mcp/auth_callback` + loopback (Claude Code) | `vertexaisearch.cloud.google.com/oauth-redirect` (+ `/static/oauth/oauth.html` for A2A/ADK) | `oauth-redirect*.googleusercontent.com/{r,a}/*` + app-specific |

### 5.3 File input

| | OpenAI | Anthropic | Gemini Enterprise | Connected Apps |
|---|---|---|---|---|
| Host → tool by reference | **Yes**: `openai/fileParams` → `download_url` (ChatGPT; Codex only for `codex_apps`) | **No** (not found, primary) | A2A `FilePart` (bytes or URI) exists in the proto; whether chat attachments are forwarded: not found | not found (primary) |
| Widget upload | `window.openai.uploadFile` / `selectFiles` | Widget `fetch` to `connectDomains` only; no upload API | n/a | not found (primary) |
| Other | — | Sandbox egress (user/admin network setting); 150k-char result cap | **BYO parsed document** into Agent Search data stores | — |

### 5.4 Listing and review

| | OpenAI | Anthropic | Gemini Enterprise (Marketplace) | Connected Apps |
|---|---|---|---|---|
| Where | platform.openai.com/plugins (ZIP) | claude.ai/directory/manage (connector + optional plugin from GitHub) | Cloud Marketplace Producer Portal → appears in GE Agent Gallery | Google Connected Console (preview application) |
| Identity | Individual or business verification (enforced) | Any paid Claude plan; Owner on Team/Ent | Marketplace vendor + Project Info Form | Developer account; ToS accepted by an admin |
| Domain proof | `/.well-known/openai-apps-challenge` (plain text) | Policy §3F ownership (no file) | not found (primary) | `/.well-known/google-connected-apps` (JSON) |
| Test material | 5+ / 3− cases, demo video, MFA-free reviewer account | Fully populated test account; ≥ 3 example prompts | Agent Card validation (private "Agent ScoreCard") | Seeded no-2FA account; ≥ 3 prompts and capabilities |
| Review depth | Automated scans + manual; continuous tool re-scan | Automated → Community by default; Verified by escalation | Product, pricing and technical reviews + card validation | Google review per version; frozen tool snapshot |
| Cost | No fee found | A paid Claude plan | 3% revenue share on standard offers | not found (primary) |
| Commerce | Physical goods only; no plans or upsell | No financial transactions without HITL; no ads | Billing via Marketplace | No third-party ads |
| Data region | EU-residency projects can't submit MCP plugins | — | App multi-region (global/us/eu) | Users: US English (Search AI Mode; Gemini custom apps) |

### 5.5 Implications for AILANG Parse

These are deductions from the rows above, not vendor quotes.

1. **One authorization server cannot satisfy all four clients with one refresh policy.** Claude says
   rotate. Connected Apps says don't rotate. A per-client policy is needed; this matches F3 in
   `m-google-connected-apps-distribution.md`.
2. **Gemini Enterprise needs a pre-registered confidential client**, with redirect
   `https://vertexaisearch.cloud.google.com/oauth-redirect`, minted per customer org. Neither CIMD nor
   DCR is used there.
3. **For Connected Apps, keep `registration_endpoint` in AS metadata even though we prefer CIMD.**
   The OAuth guide still says it is required; the console guide says CIMD is checked first.
4. **Advertise `none` in `token_endpoint_auth_methods_supported`** for Claude CIMD.
5. **Decide the `private_key_jwt` question explicitly.** ChatGPT's legacy singular field prefers
   `private_key_jwt`, but it picks from the intersection, so `none`-only works. Do not treat the
   singular field as binding.
6. **Claude Policy §1G names "uploaded files".** A connector whose input is the user's document should
   state in the submission that it acts only on files the user explicitly passes.
7. **A2A is useful only for Gemini Enterprise.** It must speak v0.3 (or 1.0 with v0.3 compat). Card
   `securitySchemes` are ignored there, and Connected Apps forbids agent-to-agent entirely.

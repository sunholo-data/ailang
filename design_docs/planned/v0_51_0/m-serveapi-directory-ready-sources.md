# Sources: m-serveapi-directory-ready

These are vendor requirements quoted **verbatim** from pages fetched on 2026-10-01, and they are
the external premises behind `m-serveapi-directory-ready.md`. Vendor pages change, so re-fetch
before each directory submission. When a quote no longer matches its page, update this file and
re-check the decisions that cite it.

## Anthropic

| # | Quote | Source |
|---|---|---|
| A1 | "Remote MCP servers that connect to a remote service and require authentication must use secure OAuth 2.0 with certificates from recognized authorities." | Anthropic Software Directory Policy §5D: https://support.claude.com/en/articles/13145358-anthropic-software-directory-policy |
| A2 | "OAuth 2.0 if your tools act on a user's account, or no authentication for public data" | Submit a connector: https://claude.com/docs/connectors/building/submission |
| A3 | "Claude starts sign-in only when the HTTP request itself fails with `401 Unauthorized` and a `WWW-Authenticate` header. A tool handler can't produce that response" | Lazy authentication: https://claude.com/docs/connectors/building/lazy-authentication |
| A4 | `static_headers`: "Fixed credential (API key or bearer token) entered by an organization Owner as a request header when adding the connector \| Beta, for a limited set of organizations" | Authentication for connectors: https://claude.com/docs/connectors/building/authentication |
| A5 | "Claude gives your discovery, registration, and token endpoints 10 seconds to respond and refresh requests 30 seconds" | same page as A4 |
| A7 | "Every tool must include a `title` and the applicable hint: `readOnlyHint: true` for read-only tools, and `destructiveHint: true` for tools that modify or delete data." | Review criteria: https://claude.com/docs/connectors/building/review-criteria |
| A6 | Supported types: `oauth_dcr`, `oauth_cimd` ("Supported by default"), `oauth_anthropic_creds` ("Contact `mcp-review@anthropic.com`"), `custom_connection`, `static_headers`, `none` | same page as A4 |

## OpenAI

| # | Quote | Source |
|---|---|---|
| O1 | Restricted data, not to be collected: "Access credentials and authentication secrets (such as API keys, MFA/OTP codes, or passwords)" | Plugin guidelines: https://developers.openai.com/plugins/plugin-guidelines |
| O2 | "If your MCP server requires authentication, the flow must be transparent and explicit." | same page as O1 |
| O3 | "must not display subscription plans, initiate new subscriptions, or promote upgrades" | same page as O1 |
| O4 | "For an authenticated MCP server, you are expected to implement an OAuth 2.1 flow that conforms to the MCP authorization spec" | Authentication: https://developers.openai.com/apps-sdk/build/auth |
| O5 | Mixed auth "requires both metadata (`securitySchemes` and the resource metadata document) **and** runtime errors that carry `_meta["mcp/www_authenticate"]`." | same page as O4 |
| O6 | "`client_id_metadata_document_supported`: set to `true` when you want ChatGPT to use CIMD for client registration. ChatGPT prioritizes CIMD when it is available" and "For CIMD, ChatGPT supports `none` for public-client token exchange" | same page as O4 |
| O8 | "Set `readOnlyHint`, `destructiveHint`, and `openWorldHint` to explicit boolean values (`true` or `false`) in each tool’s `annotations` object." (link text unwrapped) | Plugin guidelines: https://developers.openai.com/plugins/plugin-guidelines |
| O7 | Redirect URI: `https://chatgpt.com/connector_platform_oauth_redirect` | same page as O4 |
| O9 | "To let ChatGPT pass files to a tool, list each top-level file input in `_meta["openai/fileParams"]`. Each listed field must resolve to a file object or an array of file objects." / "The Scan Tools step and plugin submission reject a file schema that omits any of the four properties, does not require `download_url` and `file_id`, marks either optional property as required, or requires a property other than `download_url` or `file_id`." | Plugins reference: https://developers.openai.com/plugins/reference |
| O10 | `securitySchemes` is a top-level tool field: "`noauth`: The tool is callable anonymously" / "`oauth2`: The tool needs an OAuth 2.0 access token"; reference: "`_meta[\"securitySchemes\"]` — Back-compat mirror for clients that only read `_meta`." | same pages as O4 and O9 |

## Not yet established

- **V11:** whether ChatGPT acts on an HTTP 401 for `tools/call` the way Claude does (A3), or only
  on the `_meta` tool error (O5). Settled only by a live ChatGPT developer-mode test (spike S1).

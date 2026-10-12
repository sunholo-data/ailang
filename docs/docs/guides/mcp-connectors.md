---
reviewed: 2026-10-07
reviewBy: 2027-01-07
---

# Publish an AILANG service as a Claude / ChatGPT connector

This guide takes an AILANG `serve-api` service and gets it ready for **Anthropic's connector
directory** (claude.ai) and **OpenAI's ChatGPT plugin directory**: tool titles and hints, sign-in
with OAuth, file input, a readiness check and a submission checklist.

The worked example is **AILANG Parse**, which is live at
`https://docparse.ailang.sunholo.com/mcp/connect/` and passes the Anthropic checks today.

> **Versions.** The listed surface, `@mcp_auth`, `@mcp_secret`, `@mcp_agent_only`,
> `@mcp_token_verifier`, `--oauth-issuer` and `ailang mcp check`: since **v0.51.0**.
> `sunholo/mcp_oauth` needs **v0.52.0** or later. The no-trailing-slash fix that claude.ai's connector
> check needs: **v0.52.3**. `securitySchemes` for ChatGPT, `@mcp_file` and MCP Apps widgets
> (`@mcp_ui_resource`, `@mcp_ui`, `@mcp_app_only`): **v0.52.4**. Every snippet on this page was
> checked with `ailang check` on a build of `dev` at the v0.52.4 release commit, on 2026-10-07.

This page builds on the [Serve API guide](./serve-api.md), which documents each annotation in
detail. Read its [MCP section](./serve-api.md#mcp-model-context-protocol) first if you have not
served MCP from AILANG before.

## 1. Two surfaces from one module

When any export uses one of the listing annotations below, `serve-api --mcp-http` serves **two**
MCP endpoints from the same code:

| Endpoint | Who it is for | Sign-in | What the tools look like |
|---|---|---|---|
| `/mcp/` | agents, CLIs, SDK bridges, the MCP Registry | the service's own: API keys as arguments or headers, device-code flows | every tool, every parameter |
| `/mcp/connect/` | the directory listing (claude.ai, ChatGPT) | **OAuth 2.1**, started by an HTTP 401 | no `@mcp_agent_only` tools, no `@mcp_secret` parameters; `@mcp_auth("oauth2")` tools need a Bearer token |

You submit the `/mcp/connect/` URL. `/mcp/` stays as it was, so existing agent integrations keep
working.

Both forms of the listed URL answer directly: `/mcp/connect` and `/mcp/connect/`. claude.ai's
connector check POSTs the URL without the trailing slash and does not follow redirects. Before
v0.52.3, that request got a 307, and claude.ai reported "Couldn't determine how this server signs
in". The 401 for each form names metadata whose `resource` is the exact URL the client used
(RFC 9728 §3.3).

**Why a second surface.** The directories reject what agent surfaces usually do. Quoting the
vendor pages (fetched 2026-10-01, re-checked 2026-10-06):

- Anthropic: *"Remote MCP servers that connect to a remote service and require authentication must
  use secure OAuth 2.0"* (Software Directory Policy §5D).
- Anthropic: *"Every tool must include a `title` and the applicable hint: `readOnlyHint: true` for
  read-only tools, and `destructiveHint: true` for tools that modify or delete data."*
- Anthropic: *"Claude starts sign-in only when the HTTP request itself fails with
  `401 Unauthorized` and a `WWW-Authenticate` header. A tool handler can't produce that response."*
- OpenAI: restricted data, not to be collected: *"Access credentials and authentication secrets
  (such as API keys, MFA/OTP codes, or passwords)"*.
- OpenAI: *"Set `readOnlyHint`, `destructiveHint`, and `openWorldHint` to explicit boolean values"*.
- OpenAI: mixed auth *"requires both metadata (`securitySchemes` and the resource metadata
  document) **and** runtime errors that carry `_meta["mcp/www_authenticate"]`."*

An `apiKey` argument fails the OpenAI rule, because the model would handle the key. A device-code
tool fails the Anthropic one, because it is not OAuth. The listed surface removes both, and
`serve-api` emits the 401, which a tool function cannot.

The full quotes, with URLs, are in
[`m-serveapi-directory-ready-sources.md`](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md).
Vendor rules change, so re-read the vendor pages before you submit.

## 2. Annotations

One module, `notes_svc`, runs through this section. Every tool on it is shown below, and the module
passes all seven `ailang mcp check --target both` checks when served (section 4).

### Titles and hints

```ailang
-- List the formats the service reads. Open: no sign-in needed.
@mcp_title("List formats")
@mcp_hints("readOnly")
export func formats() -> string = "docx, pptx, xlsx, pdf"

-- Delete a stored summary. Destructive, so the host asks before running it.
@mcp_title("Delete summary")
@mcp_hints("destructive", "idempotent")
@mcp_auth("oauth2")
export func deleteSummary(id: string) -> string ! {IO} = "deleted ${id}"
```

`@mcp_hints` takes any of `readOnly`, `destructive`, `idempotent` and `openWorld`, and the list is
**complete**. A hint you leave out is emitted as an explicit `false`, which is the explicit-boolean
form OpenAI requires. What `tools/list` on `/mcp/connect/` returns for these two tools:

```text
formats        readOnlyHint: true,  destructiveHint: false, idempotentHint: false, openWorldHint: false
deleteSummary  readOnlyHint: false, destructiveHint: true,  idempotentHint: true,  openWorldHint: false
```

Use `openWorld` when the tool reaches outside your service, for example fetching a URL the user
gave. A tool with no `@mcp_hints` and an empty effect row is marked read-only automatically. An
effectful tool with no hints gets none, and `serve-api` logs a `WARN` at startup. Fix those before
you submit. The full rules are in
[the Serve API guide](./serve-api.md#mcp-model-context-protocol), under "Tool titles and behaviour
hints".

### A gated tool: `@mcp_auth("oauth2")`, `@mcp_secret`, `@optional`

```ailang
-- Summarise a document for the signed-in account.
@mcp_title("Summarise document")
@mcp_hints("readOnly", "openWorld")
@mcp_auth("oauth2")
@mcp_secret("apiKey")
@optional("apiKey")
export func summarise(text: string, apiKey: string, _headers: Json) -> string ! {IO} {
  let bearer = match getString(_headers, "Authorization") {
    Some(h) => h,
    None => ""
  }
  let who = if apiKey != "" then "argument key" else if bearer != "" then "bearer token" else "nobody"
  "summary of ${show(length(text))} characters for ${who}"
}
```

(imports: `std/json (Json, getString)`, `std/option (Some, None)`, `std/string (length)`)

- **`@mcp_auth("oauth2")`**: on `/mcp/connect/`, a call without an accepted Bearer token is refused
  before the function runs. On `/mcp/` it has no effect, and the tool checks its own key as before.
- **`@mcp_secret("apiKey")`**: the parameter is dropped from the listed schema, and it binds `""`
  even if a client sends it. The parameter must also be **`@optional`**, or the tool is not
  registered.
- **`_headers: Json`** binds the request's HTTP headers, never appears in the schema, and cannot be
  forged from arguments. After the gate admits a call, `Authorization: Bearer <token>` is how the
  tool learns who is calling.

The same tool, as each surface lists it:

```text
/mcp/          summarise  properties: [apiKey, text]   (no securitySchemes)
/mcp/connect/  summarise  properties: [text]           securitySchemes: [{"type":"oauth2"}]
```

### Who is allowed in: `@mcp_token_verifier`

```ailang
-- Accept only tokens your authorization server minted.
@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = startsWith(token, "demo_") && length(token) > 5
```

Exactly one `(token: string) -> bool` function per server. `serve-api` calls it with the Bearer
token before any gated tool runs on `/mcp/connect/`. It is never a tool, and never an HTTP endpoint
unless you also give it a `@route`. A real verifier looks the token up: AILANG Parse hashes it and
checks the `dp_` key store (`verifyMcpToken` in docparse's `services/mcp_oauth.ail`).

Verification **fails closed**:
- `false` gives a 401, and the client signs in again.
- An error, a panic, more than 5 s, or more than 32 verifications in flight gives **HTTP 503 +
  `Retry-After`**, so a backend outage does not loop the client through sign-in again.
- In every one of these cases the tool never runs.

`@mcp_auth("oauth2")` without `--oauth-issuer` or without a verifier is a registration `ERROR`: the
tool is not served at all, on either surface. Parse's Dockerfile refuses to start without an
issuer for that reason.

### Agent-only tools and hidden exports: `@mcp_agent_only`, `@nomcp`

```ailang
-- Device-code sign-in for headless agents; not on the directory surface.
@mcp_title("Sign in with a device code")
@mcp_hints("openWorld")
@mcp_agent_only
export func deviceLogin(label: string) -> string ! {IO} = "visit https://example.com/device for ${label}"

-- Entry point for `ailang run`; not a tool on either MCP surface.
@nomcp
export func main() -> unit ! {IO} = println(formats())
```

- `@mcp_agent_only` keeps a tool on `/mcp/` only. Parse uses it for `mcpAuth` and `mcpAuthPoll`,
  its device-code sign-in, because directory clients run OAuth themselves.
- `@nomcp` removes an export from **both** MCP surfaces and keeps it on HTTP/OpenAPI. Use it for
  `main`, for OAuth routes and for helpers.
- The built-in `submit_feedback` tool appears on both surfaces. A reviewer may question it on a
  listing (it accepts free text and a contact). `--no-feedback-tool` removes it.

### Lazy auth: open tools work before sign-in

`initialize`, `tools/list` and open tools never need a token, so the directory can list your tools
and the user can try them before signing in. A gated call without a token gets this, captured from
`notes_svc`:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json
Www-Authenticate: Bearer resource_metadata="http://127.0.0.1:28471/.well-known/oauth-protected-resource/mcp/connect/"

{"id":2,"jsonrpc":"2.0","result":{"_meta":{"mcp/www_authenticate":["Bearer resource_metadata=\"http://127.0.0.1:28471/.well-known/oauth-protected-resource/mcp/connect/\""]},"content":[{"text":"Authentication required: sign in to use this tool.","type":"text"}],"isError":true}}
```

- The status and the `WWW-Authenticate` header are what Claude acts on.
- The body is the JSON-RPC tool error with `_meta["mcp/www_authenticate"]`, which is what OpenAI
  documents for ChatGPT.
- The metadata URL serves `{"resource": "<the listed URL>", "authorization_servers": ["<--oauth-issuer>"], ...}`.

In claude.ai this is **"Sign in when needed"**: the connector is added without signing in, and
the first gated call shows a **Connect** prompt. Whether ChatGPT acts on the 401 or only on the
`_meta` error has not been tested live yet (open question V11 in the design doc). `serve-api` sends
both, so either should work.

## 3. OAuth with no extra infrastructure: `sunholo/mcp_oauth`

`--oauth-issuer` names an authorization server, and `serve-api` does not contain one. The registry
package [`sunholo/mcp_oauth`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/mcp-oauth)
is that server, written in AILANG and served by the same `serve-api` process as your tools. You
need no separate auth service, sidecar or managed IdP.

```toml
# ailang.toml
[dependencies]
"sunholo/mcp_oauth" = "0.1.1"

[effects]
# SharedMem must be listed even if you never use it: 0.1.x's Hooks record names it in every hook's effect row.
max = ["IO", "FS", "Net", "Env", "Rand", "Clock", "Declassify", "SharedMem"]
```

What it implements:
- **PKCE S256 only**. `plain` is refused, and the verifier length and charset are checked.
- **CIMD clients.** The `client_id` is a URL to the client's metadata document, which is fetched
  once and must list the `redirect_uri`. Claude and ChatGPT both use CIMD, so nothing is
  pre-registered. There is no DCR.
- **Exact redirect matching**, plus RFC 8252 loopback on any port (for Claude Code).
- **Single-use codes** with a 60 s lifetime, stored only as digests. A replayed code revokes the
  token it issued.
- **`iss` on every authorization response** (RFC 9207), and the metadata advertises
  `authorization_response_iss_parameter_supported`. This lets ChatGPT use its stable redirect URI
  instead of a per-connection callback.
- **Refresh tokens rotate** on every use and belong to a family, valid 30 days. Reusing a rotated
  token revokes every access token in the family.

### The hooks you supply

Your service fills one `Hooks` record (from `pkg/sunholo/mcp_oauth/flow`):

| Field | Contract | AILANG Parse uses |
|---|---|---|
| `authenticate(idToken)` | ID token from your login page → account id | Firebase ID token → uid (`sunholo/firebase_auth`) |
| `mint(account, clientId)` | a fresh access token that **your `@mcp_token_verifier` accepts** | an OAuth-scoped `dp_` key, valid 24 h |
| `put` / `get` / `del` | storage with expiry; values hold digests only | Firestore `oauth_records` |
| `revoke(tokenDigest)` | invalidate the access token whose `sha256Hex` this is | deactivate the `dp_` key with that hash |
| `accessTtlSec` | sent as `expires_in`; must match what the verifier enforces | 86400 |
| `issuer` | the same value as `--oauth-issuer` and the metadata's `issuer` | `OAUTH_ISSUER` |

A minimal in-memory version. Tokens are stored only by digest, so the verifier is a hash lookup:

```ailang
func hooks() -> Hooks ! {Env} =
  { authenticate: hAuth, mint: hMint, put: hPut, get: hGet, del: hDel, revoke: hRevoke,
    accessTtlSec: 3600, issuer: issuer() }

-- Mint an access token your @mcp_token_verifier accepts. Store only its digest.
func hMint(account: string, clientId: string) -> Result[string, string] ! {IO, FS, Env, Net, Clock, SharedMem, Rand[mode=crypto], Declassify} {
  let t = "demo_${replace(uuid4(), "-", "")}"
  smPut("valid:${sha256Hex(t)}", fromString(account))
  Ok(t)
}

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {SharedMem} = has_key("valid:${sha256Hex(token)}")
```

### The four routes

`serve-api` serves only your own modules' `@route`s, so you declare the routes and they call the
package. Every route is `@nomcp`, so it never becomes a tool:

| Route | Calls | Returns |
|---|---|---|
| `GET /.well-known/oauth-authorization-server` | `core.asMetadata(issuer)` | RFC 8414 metadata: S256, CIMD, `none` client auth |
| `GET /oauth/authorize` | `cimd.fetchClient(client_id)`, then `flow.beginAuthorize` | 302 to your sign-in page with a request handle |
| `POST /oauth/login/complete` | `flow.completeLogin(handle, idToken)` | 302 to the client's `redirect_uri` with `code`, `state`, `iss` |
| `POST /oauth/token` (form) | `flow.exchangeCode` or `flow.refreshTokens` | token JSON, or an RFC 6749 §5.2 error |

The authorize route, from the verified example:

```ailang
@route("GET", "/oauth/authorize")
@raw
@nomcp
export func oauthAuthorize(req: {body: string, headers: Json, method: string, path: string, query: Json})
  -> {_body: string, _status: int, _headers: {location: string}} ! {IO, FS, Env, Net, Clock, SharedMem, Rand[mode=crypto], Declassify} {
  let cid = q(req.query, "client_id")
  match fetchClient(cid) {
    Err(e) => bad(oauthError("invalid_client", e)),
    Ok(registered) =>
      match beginAuthorize(hooks(), "/login.html", cid, registered, q(req.query, "redirect_uri"),
                           q(req.query, "response_type"), q(req.query, "code_challenge"),
                           q(req.query, "code_challenge_method"), q(req.query, "state"), nowSec()) {
        Ok(loc) => redirect(loc),
        Err(e) => bad(e)
      }
  }
}
```

Copy the rest from these sources:
- the package's `routes_template.ail` and `AGENT.md` (`ailang pkg docs sunholo/mcp_oauth`);
- for a production adopter, docparse's `docparse_api/services/mcp_oauth.ail`, about 300 lines:
  Firebase sign-in, Firestore storage, plus an extra `GET /oauth/request` route that tells the
  sign-in page which app is asking.

Use `nowSec()` (`std/clock.now() / 1000`), because the flow works in seconds.

### The sign-in page, on the same origin

`beginAuthorize` redirects to **your** sign-in page with a `handle`. The page signs the user in
however you like (Parse uses Google or GitHub through Firebase), shows which app is asking, and
POSTs `handle` + `id_token` to `/oauth/login/complete` as a top-level form navigation.

Serve the page with `serve-api --static <dir>`. It then shares an origin with `/oauth/*`, and you
deploy nothing extra. Parse's command line:

```bash
ailang serve-api --mcp-http --routes-only \
  --caps IO,FS,AI,Env,Net,Rand,Clock,Process \
  --oauth-issuer "$OAUTH_ISSUER" \
  --static /app/static \
  docparse_api/
```

Put an `index.html` in each static directory, because `--static` otherwise lists directory
contents.

**Anti-framing guard.** From v0.54.0, `serve-api --static` sends
`X-Frame-Options: DENY` and `Content-Security-Policy: frame-ancestors 'none'`
by default, plus nosniff and no-referrer. Keep these defaults for OAuth consent
pages. `--static-header` overrides individual headers;
`--no-static-security-headers` removes defaults and requires you to supply your
own protections. On older releases, configure these headers at the reverse
proxy. Parse's synchronous `antiframe.js` guard and consent-script frame check
remain an additional defense for older deployments.

### Security properties

- **Z3-proved predicates.** `ailang verify` proves the pure security checks in `core`: exact
  redirect matching, verifier length, `S256` only, the CIMD https gate, code expiry and the TTL
  bound. A deliberately broken redirect matcher in the package's tests must produce a
  counterexample, which shows the proof is not vacuous.
- **IFC-labelled secrets.** Every code, handle, access token and refresh token inside `flow` has
  the type `string<secret>`. The one log function takes `string{not secret}`, so logging a secret
  is a **compile error**. The only function that lowers the label is `digestOf`
  (`! {Declassify}`). CI injects leaks into a copy of the real `flow.ail` and asserts each one
  fails to compile.
- **Crypto randomness in the type.** Minting runs under `Rand[mode=crypto]`, which the effect
  checker forces up the call chain.
- **`Net[scope=public]` for client documents.** The CIMD URL is chosen by whoever starts the flow,
  so `cimd.fetchClient` fetches it in a public-only Net mode: one fetch, a byte cap, no loopback,
  link-local or private address on any redirect hop, and DNS pinned at connect time. A hostname
  that resolves to `127.0.0.1` or `169.254.169.254` is refused, even when `serve-api` otherwise
  allows those addresses.

What the labels do **not** cover:
- your own hook code;
- your route functions, which receive the response body unlabelled.

So follow the package's adopter checklist:
- run with `AILANG_TRACE_VALUES=off`, because traces render arguments verbatim;
- never log request bodies, query strings or flow results;
- store keys hashed;
- never fetch a `client_id` URL under plain `Net`.

The threat model (T1–T10) is in
[`m-mcp-oauth-package.md`](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_52_0/m-mcp-oauth-package.md).

## 4. Check readiness

### `ailang mcp check`

```bash
ailang mcp check https://your-service.example.com/mcp/connect/ --target both
```

`--target` is `anthropic` (default), `openai` or `both`; add `--json` for machine output. Exit
codes: 0 = no FAIL, 1 = at least one FAIL, 2 = could not check (unreachable or not MCP).

| Check | Passes when | Source |
|---|---|---|
| `annotations` | every tool has a title and `readOnlyHint` or `destructiveHint`; with `openai`, all three of read-only, destructive and open-world are explicit booleans | A7, O8 |
| `credentials` | no parameter looks like a credential (`apiKey`, `token`, `secret`, `password`, …) | A2, O1 |
| `zero-arg` | no tool requires a placeholder argument, so zero-argument tools are callable with `{}` | — |
| `lazy-auth` | gated tools answer 401 with a `resource_metadata` challenge; the metadata's `resource` equals the URL you checked and names an authorization server | A3 |
| `authorization-server` | the issuer's metadata advertises `S256` and CIMD (or a registration endpoint), within 10 s | A5, A6, O6 |
| `file-params` (openai) | every `openai/fileParams` field has OpenAI's four-property file schema | O9 |
| `security-schemes` (openai) | every tool that answers 401 declares `securitySchemes: [{"type":"oauth2"}]` | O5, O10 |

To find gated tools, the check calls tools with `{}` and no token. It does this only for read-only
tools and tools with required parameters, so it never runs a write tool.

The section 3 example (the hooks, the four routes and one gated tool), served by `serve-api` with
its own origin as `--oauth-issuer` and checked locally:

```text
MCP directory check: http://127.0.0.1:28472/mcp/connect/ (target: both)

  PASS  annotations                  all 2 tools have a title and hints [A7]
  PASS  credentials                  no credential-shaped parameters [A2,O1]
  PASS  zero-arg                     no tool requires a placeholder argument
  PASS  lazy-auth                    1 gated tool(s) answer 401 with resource metadata naming http://127.0.0.1:28472 [A3]
  PASS  authorization-server         http://127.0.0.1:28472 advertises S256 and a client registration method [A6,O6]
  PASS  file-params                  no tool declares openai/fileParams [O9]
  PASS  security-schemes             1 gated tool(s) declare an oauth2 security scheme [O5,O10]

No failures. Re-fetch the vendor requirements before submitting; they change.
```

AILANG Parse in production on 2026-10-07 passes the five Anthropic checks. Its gated tools do not
yet declare `securitySchemes`, which `--target openai` reports as a FAIL, because its deployed
`serve-api` predates v0.52.4. Moving to v0.52.4 adds the schemes with no code change.

`mcp check` reads only the wire. It cannot see your tool descriptions, your result payloads or
your sign-in page, and reviewers read all three (see section 6).

### Test it in claude.ai

1. Deploy the service on `https://` with `--oauth-issuer` set to its own origin.
2. In claude.ai: **Settings → Connectors → Add custom connector**, and paste the `/mcp/connect/` URL.
   With or without the trailing slash both work from v0.52.3.
3. In the connector's settings, choose **"Sign in when needed"** (lazy auth) and **"Use Claude's
   published identity (CIMD)"**. With CIMD there is no client ID or secret to enter.
4. Ask for something an open tool answers. It should run without sign-in.
5. Ask for something a gated tool answers. A **Connect** prompt appears. Sign in on your page, and
   the call completes.
6. Run **every** tool once. Anthropic asks you to confirm this when you submit.

If claude.ai says it "couldn't determine how this server signs in", run `mcp check` on the exact
URL you pasted, then `curl` the `resource_metadata` URL from the 401. Its `resource` must equal
that URL.

### Test it in ChatGPT

Turn on developer mode, add the `/mcp/connect/` URL as a connector, then repeat steps 4–6. ChatGPT
registers itself through CIMD. Because `sunholo/mcp_oauth` returns `iss`, ChatGPT uses its stable
redirect `https://chatgpt.com/connector_platform_oauth_redirect`. With CIMD, the allowed redirect
URIs come from ChatGPT's own client metadata document, so you register nothing. A live ChatGPT sign-in against an AILANG service has not been recorded yet, so treat
this as the first thing to test.

## 5. File input

> **Status (2026-10-07).** The `serve-api` pieces below are released in **v0.52.4**. The package
> `sunholo/mcp_files` and its upload widget are **in progress** and not yet in the registry.
> This section will change. The design is
> [`m-mcp-file-handoff.md`](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_53_0/m-mcp-file-handoff.md).

**Why files are hard over MCP.**
- Tool arguments are **model output**. A file passed as base64 has to be written out by the model,
  token by token, and a 14 KB DOCX already hit claude.ai's output limit. Claude then silently
  parsed the file itself with a hand-written zip reader, and the tool was never called.
- MCP has no file primitive yet. SEP-2631 (`files/authorizeUpload`) is an open draft, and Claude
  does not implement drafts.
- ChatGPT has its own mechanism, `_meta["openai/fileParams"]`. The host passes a download URL for a
  file the user uploaded.

So bytes have to move **by code, not by the model**. Three pieces cover the three kinds of client.

**ChatGPT: `@mcp_file` (v0.52.4).** This marks a parameter as a host-supplied file. `serve-api`
emits `openai/fileParams` and OpenAI's four-property schema, and binds the host's file object to a
closed record:

```ailang
type OpenAIFile = { download_url: string, file_id: string, mime_type: string, file_name: string }

-- Describe a file the user uploaded (needs a signed-in account).
@mcp_title("Describe uploaded file")
@mcp_hints("readOnly", "openWorld")
@mcp_auth("oauth2")
@mcp_file("file")
export func describeFile(file: OpenAIFile) -> string ! {IO} = describe(file)
```

Your function receives a URL, not bytes. Fetch it under `Net`; `Net[scope=public]` is the safe
choice for a URL that came from outside. The binding rules are in the Serve API guide under
[Files from the host](./serve-api.md#files-from-the-host-mcp_file). Full example:
`examples/runnable/serve_api_mcp_file.ail`.

**claude.ai: an upload widget (MCP Apps, v0.52.4 serving).** On 2026-10-07, a throwaway MCP App in
claude.ai web rendered a file picker, and the user's file reached the server **byte for byte** by
two routes:
- a direct `fetch` to a declared `connectDomain`;
- a widget-only tool call that the host relayed, with the bytes never passing through the model.

`serve-api` now serves such widgets:

```ailang
@mcp_title("File picker")
@mcp_ui_resource("ui://example/picker", "self")
export func pickerHtml() -> string =
  "<!doctype html><meta charset='utf-8'><input type='file' id='f'><pre id='out'></pre>"

@mcp_title("Choose a file")
@mcp_hints("readOnly")
@mcp_ui("ui://example/picker")
export func chooseFile() -> string = "Picker shown. Ask the user to choose a file in it."
```

| Annotation | What it emits |
|---|---|
| `@mcp_ui_resource` | the HTML as a `ui://` resource (`text/html;profile=mcp-app`, CSP `connectDomains`; `"self"` is your own origin) |
| `@mcp_ui` | makes a tool render the widget |
| `@mcp_app_only` | keeps a tool out of the model's tool list while the widget can still call it |

Details: [Widgets (MCP Apps)](./serve-api.md#widgets-mcp-apps-mcp_ui_resource-mcp_ui-mcp_app_only).
Full example: `examples/runnable/serve_api_mcp_app.ail`. Report a widget's result to the model with
`ui/update-model-context`, not `ui/message`: the latter drafts a user message under a "Use caution
before running this prompt" banner.

**In progress: `sunholo/mcp_files`.** It will provide:
- **policy:** single-use upload tokens (stored as digests, bound to an account and a size cap,
  valid 10 minutes), an upload descriptor shaped like SEP-2631's, `fileRef`s, and a
  `fetchFileParam` that fetches a ChatGPT `download_url` once under `Net[scope=public]`;
- **the standard upload widget:** a picker with a direct POST and a via-host fallback;
- **storage hooks**, so each service plugs in its own storage;
- **the code path** for Claude Code and Codex: `createUpload` returns a ready `curl` line.

AILANG Parse is adopting it first. Until it ships, the honest answers for a file the user attached
are a public URL, or a clear message that says what to do. **Never** let the model fall back to
parsing the file itself. Parse's tool descriptions now say so explicitly.

## 6. Submission checklist

`mcp check` covers the wire. Reviewers also look at everything below. The list comes from the
AILANG Parse submission kit (vendor pages re-read 2026-10-06; they change).

**Both directories**
- [ ] `ailang mcp check <url> --target both` passes, run against the **deployed** listed URL.
- [ ] Every tool has been run once from claude.ai (and ChatGPT developer mode) as a custom connector.
- [ ] **Privacy policy** URL. It covers the connector: what signing in stores (tokens, as hashes),
      how long, how to revoke, and that the assistant's vendor receives tool results under its own
      policy.
- [ ] **Terms** URL, and a **support** contact (email or issue tracker).
- [ ] **Public documentation** for the connector: the URL, adding it, signing in, what needs
      sign-in, file limits and revoking. Anthropic requires it by your publish date.
- [ ] The sign-in page links privacy and terms, names the app that is asking, and resists framing.
- [ ] **A reviewer test account** that signs in with a password and **no** MFA, emailed codes or
      magic links (OpenAI is explicit about this). A Google or GitHub-only sign-in will hit
      new-device challenges. Seed the account with data, and give it the plan that every listed
      tool needs.
- [ ] **Example prompts that work against the listed surface**: at least 3 for Anthropic, and up to
      3 default prompts of 128 characters or less for OpenAI. Don't use prompts that depend on an
      attached file until file input works.
- [ ] Tool descriptions match the listed surface. Remove any mention of API keys, device flows or
      tools that exist only on `/mcp/`. Decide whether `submit_feedback` belongs on the listing
      (`--no-feedback-tool`).

**Anthropic** ([developer portal](https://claude.ai/directory/manage) → Submit new → MCP connector;
needs a paid plan, and the Owner role on Team or Enterprise)
- [ ] Authentication type: **OAuth with client ID metadata documents** (`oauth_cimd`), flagged as
      lazy (tools prompt for sign-in on demand).
- [ ] Listing: name (100 characters or less), one-liner (200 or less), description (2,000 or less;
      Anthropic cannot edit it), 1–5 categories, icon, slug (permanent once published).
- [ ] Use cases, prerequisites, and whether the connector reads, writes or both.
- [ ] Ownership of every domain the connector uses (Policy §3F).
- [ ] The seven compliance acknowledgments. Escalations: `mcp-review@anthropic.com`.
- [ ] Policy and criteria pages:
      [Software Directory Policy](https://support.claude.com/en/articles/13145358-anthropic-software-directory-policy),
      [submission](https://claude.com/docs/connectors/building/submission),
      [review criteria](https://claude.com/docs/connectors/building/review-criteria),
      [authentication](https://claude.com/docs/connectors/building/authentication),
      [lazy authentication](https://claude.com/docs/connectors/building/lazy-authentication).

**OpenAI** ([plugins portal](https://platform.openai.com/plugins) → upload a ZIP with `plugin.json` +
`mcp.json` + assets)
- [ ] **Individual or business verification** of the OpenAI Platform organization, and the Owner
      role or "Apps Management Write".
- [ ] **Domain verification**: serve the token from the portal at
      `/.well-known/openai-apps-challenge` on the MCP host, as plain text. A `@route` can serve it.
      Check with `curl` that the body is the bare token and not JSON-wrapped.
- [ ] **Demo video URL** (required for MCP plugins): add the connector, sign in with the reviewer
      account, and run the positive test cases.
- [ ] **5 positive and 3 negative test cases** (prompt, tools triggered, expected behaviour), plus
      release notes.
- [ ] No pricing, subscriptions or upgrade prompts anywhere in tool output.
- [ ] No request, trace or session IDs or timestamps in tool results (data minimisation).
- [ ] Package fields: `displayName` (30 characters or less), `shortDescription` (30 or less),
      `longDescription` (4,000 or less), `brandColor`, `logo` and `composerIcon` (square PNG).
- [ ] Policy pages: [plugin guidelines](https://developers.openai.com/plugins/plugin-guidelines),
      [authentication](https://developers.openai.com/apps-sdk/build/auth),
      [plugins reference](https://developers.openai.com/plugins/reference).

## Reference

- [Serve API guide](./serve-api.md): every annotation in detail.
- Example modules in the repo: `examples/runnable/serve_api_mcp_oauth.ail` (listed surface),
  `serve_api_mcp_hints.ail`, `serve_api_mcp_file.ail` and `serve_api_mcp_app.ail`.
- Design docs:
  [directory-ready](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_51_0/m-serveapi-directory-ready.md),
  [vendor sources](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md),
  [`sunholo/mcp_oauth`](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_52_0/m-mcp-oauth-package.md),
  [file handoff](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_53_0/m-mcp-file-handoff.md).
- Live example: `https://docparse.ailang.sunholo.com/mcp/connect/` (AILANG Parse).

### Checking consent page framing

`ailang mcp check https://your-service.example/mcp/ --target both`
probes the discovered authorization endpoint with dummy OAuth code-flow and S256
PKCE parameters, follows redirects, and judges only final 2xx HTML. A valid
X-Frame-Options or CSP frame-ancestors protection passes. Missing protection
fails anthropic/both targets and warns for openai. Rejected dummy clients,
non-HTML responses, unreachable endpoints, and redirect failures warn because
no consent page could be assessed. Without an authorization endpoint the probe
is skipped. A warning does not establish that your consent page is protected.

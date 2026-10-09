---
reviewBy: 2027-03-01
---

# Serve API Guide

This guide explains how to expose AILANG functions as REST API endpoints and optionally pair them with a React frontend.

> **Version:** Available since v0.7.1

## Overview

AILANG provides two commands for web integration:

| Command | Purpose |
|---------|---------|
| `ailang serve-api` | Serve AILANG module exports as auto-generated REST endpoints |
| `ailang init web-app` | Scaffold a full-stack project (AILANG API + React frontend) |

Both build on the [Go Interop](./go-interop.md) embed API, wrapping it with HTTP routing so you don't need to write any Go code.

---

## Quick Start

### Option 1: Scaffold a New Project

```bash
ailang init web-app myproject
cd myproject
cd ui && npm install && cd ..
make dev
```

This starts:
- AILANG API server on `http://localhost:8080`
- React dev server on `http://localhost:5173` (proxies `/api` to AILANG)

Open `http://localhost:5173` in your browser.

### Option 2: Serve Existing Modules

```bash
# Serve a single module
ailang serve-api api/handlers.ail --port 8080

# Serve all .ail files in a directory
ailang serve-api ./api/ --port 8080

# With React frontend proxy
ailang serve-api ./api/ --port 8080 --frontend ./ui
```

:::note How serve-api picks the project root (basePath, v0.45.0+)

A module is registered as a route only if its file lives under the **basePath**. serve-api starts
from **the directory you give it** (for a file argument, that file's directory) and moves outward
only when a `module` header requires it: `serve-api ./api/` with `api/handlers.ail` declaring
`module api/handlers` uses the directory above `api/`. A header never moves the basePath *inward*,
so another `.ail` file in a subdirectory (`client/wsclient.ail` declaring `module wsclient`) no longer
makes `client/` the basePath and drops your route module. Without a usable header, the basePath is
the current directory when it contains the arguments, else the arguments' own directory.

Hosted packages whose modules declare a namespaced path like `module sunholo/<pkg>/<module>`
must be served from a directory their declared paths resolve under — `cd` into the package
directory and run `ailang serve-api .`.

Standard-library modules (`std/*`) are never registered: they are not listed in the banner,
`/api/_meta/modules` or `/api/_health`, and have no `/api/std/...` endpoints.

:::

---

## How It Works

Given two AILANG modules (from `examples/web_api_demo/`):

```ailang
-- api/math.ail
module api/math

export pure func add(x: int, y: int) -> int =
  x + y

export pure func multiply(x: int, y: int) -> int =
  x * y

export pure func factorial(n: int) -> int =
  if n <= 1 then 1
  else n * factorial(n - 1)

export pure func fibonacci(n: int) -> int =
  if n <= 0 then 0
  else if n == 1 then 1
  else fibonacci(n - 1) + fibonacci(n - 2)
```

```ailang
-- api/greet.ail
module api/greet

import std/json (encode, jo, kv, js)

export pure func hello(name: string) -> string =
  "Hello, ${name}!"

export pure func farewell(name: string) -> string =
  "Goodbye, ${name}. Until next time!"

export pure func welcome(name: string) -> string =
  encode(jo([
    kv("message", js("Welcome, ${name}!")),
    kv("name", js(name))
  ]))
```

Running `ailang serve-api ./api/` auto-generates these endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/api/math/add` | Call `add()` |
| POST | `/api/api/math/multiply` | Call `multiply()` |
| POST | `/api/api/math/factorial` | Call `factorial()` |
| POST | `/api/api/math/fibonacci` | Call `fibonacci()` |
| POST | `/api/api/greet/hello` | Call `hello()` |
| POST | `/api/api/greet/farewell` | Call `farewell()` |
| POST | `/api/api/greet/welcome` | Call `welcome()` |
| GET | `/api/_meta/modules` | List all modules and exports |
| GET | `/api/_meta/modules/api/math` | Module detail |
| GET | `/api/_meta/openapi.json` | OpenAPI 3.1 spec |
| GET | `/api/_meta/docs` | Swagger UI (interactive explorer) |
| GET | `/api/_meta/redoc` | ReDoc (API reference) |
| GET | `/api/_health` | Health check |
| GET | `/.well-known/agent.json` | A2A Agent Card (requires `--a2a`) |
| POST | `/a2a/` | A2A JSON-RPC endpoint (requires `--a2a`) |

### URL Convention

The URL path follows the pattern:

```
POST /api/{module-path}/{function-name}
```

Where `{module-path}` matches the `module` declaration in the `.ail` file exactly.

---

## Calling Functions

### JSON Request Format

**Positional arguments (recommended):**

```bash
curl -X POST http://localhost:8080/api/api/math/add \
  -H "Content-Type: application/json" \
  -d '{"args": [3, 4]}'
# {"result":7,"module":"api/math","func":"add","elapsed_ms":12}
```

**Single value (for single-argument functions):**

```bash
curl -X POST http://localhost:8080/api/api/greet/hello \
  -H "Content-Type: application/json" \
  -d '"Bob"'
# {"result":"Hello, Bob!","module":"api/greet","func":"hello","elapsed_ms":0}
```

**Named parameters (agent-friendly):**

For functions with known parameter names, you can send a flat JSON object with named fields instead of a positional `args` array. This is the recommended format for AI agents and programmatic callers:

```bash
# Given: export func parseFile(path: string, outputFormat: string) -> string
curl -X POST http://localhost:8080/api/docparse/parseFile \
  -H "Content-Type: application/json" \
  -d '{"path": "data/sample.docx", "output_format": "blocks"}'
```

**Name matching rules:**
- Exact match: `"path"` → `path` parameter
- snake_case to camelCase: `"output_format"` → `outputFormat` parameter
- Unknown fields are silently ignored (forward-compatible)

**Precedence:** If the request body contains an `"args"` key with an array value, positional binding is used (backward compatible). Named binding only activates for plain JSON objects.

**Zero-value padding for missing parameters:**

When a named parameter is omitted from the JSON body, it receives a type-appropriate zero-value instead of crashing. This allows functions to validate inputs and return structured errors:

| Parameter Type | Zero Value |
|---------------|------------|
| `string`      | `""`       |
| `int`         | `0`        |
| `float`       | `0.0`      |
| `bool`        | `false`    |
| `list`/`array`| `[]`       |
| `record`      | `{}`       |

```bash
# Missing apiKey gets "" instead of unit — function can validate
curl -X POST http://localhost:8080/api/docparse/parseFile \
  -H "Content-Type: application/json" \
  -d '{"path": "data/sample.docx"}'
# Function receives: path="data/sample.docx", outputFormat=""
```

Positional `{"args": [...]}` with fewer elements than expected is also padded with zero-values for the remaining parameters.

**No arguments (for nullary functions):**

```bash
curl -X POST http://localhost:8080/api/api/handlers/getStatus
```

### JSON Response Format

All function calls return a `FunctionCallResponse` envelope by default:

```json
{
  "result": "Hello, World!",
  "module": "api/greet",
  "func": "hello",
  "elapsed_ms": 2
}
```

> To skip this envelope and return raw JSON, use the [`@nowrap` annotation](#nowrap--raw-json-output).

On error:

```json
{
  "error": "function \"nope\" not found in module \"api/math\" (available: [add multiply factorial fibonacci])",
  "module": "api/math",
  "func": "nope",
  "elapsed_ms": 0
}
```

### Tested Examples

These examples are verified by the automated test script at `examples/web_api_demo/test.sh`:

```bash
# Math functions
curl -X POST http://localhost:8080/api/api/math/add \
  -H "Content-Type: application/json" -d '{"args": [3, 4]}'
# {"result":7, ...}

curl -X POST http://localhost:8080/api/api/math/multiply \
  -H "Content-Type: application/json" -d '{"args": [5, 6]}'
# {"result":30, ...}

curl -X POST http://localhost:8080/api/api/math/factorial \
  -H "Content-Type: application/json" -d '{"args": [5]}'
# {"result":120, ...}

curl -X POST http://localhost:8080/api/api/math/fibonacci \
  -H "Content-Type: application/json" -d '{"args": [10]}'
# {"result":55, ...}

# Greet functions
curl -X POST http://localhost:8080/api/api/greet/hello \
  -H "Content-Type: application/json" -d '{"args": ["World"]}'
# {"result":"Hello, World!", ...}

curl -X POST http://localhost:8080/api/api/greet/farewell \
  -H "Content-Type: application/json" -d '{"args": ["Alice"]}'
# {"result":"Goodbye, Alice. Until next time!", ...}

# JSON-returning function
curl -X POST http://localhost:8080/api/api/greet/welcome \
  -H "Content-Type: application/json" -d '{"args": ["Charlie"]}'
# {"result":"{\"message\":\"Welcome, Charlie!\",\"name\":\"Charlie\"}", ...}
```

---

## Introspection Endpoints

The endpoints below list what the server **serves**: with `--routes-only` they show only `@route`
exports, and `@noexpose` exports never appear. A page that must expose nothing but its own routes
and files turns them all off with `--no-introspection`: `/api/_meta/modules`, `/api/_meta/modules/*`,
`/api/_meta/openapi.json`, `/api/_meta/docs`, `/api/_meta/redoc` and `/api/_health` then answer 404.
The paths stay reserved, so a `@route` still cannot claim them — give your proxy its own health
route (e.g. `@route("GET", "/healthz")`). The Swagger UI and ReDoc pages load fonts and scripts from
public CDNs; `--no-introspection` also removes those external fetches.

```bash
ailang serve-api --routes-only --no-introspection --static ./public ./server/
```

### List All Modules

```bash
curl http://localhost:8080/api/_meta/modules
```

Response:

```json
{
  "count": 2,
  "modules": [
    {
      "path": "api/math",
      "exports": [
        { "name": "add", "type": "int -> int -> int", "pure": true, "arity": 2 },
        { "name": "multiply", "type": "int -> int -> int", "pure": true, "arity": 2 },
        { "name": "factorial", "type": "int -> int", "pure": true, "arity": 1 },
        { "name": "fibonacci", "type": "int -> int", "pure": true, "arity": 1 }
      ]
    },
    {
      "path": "api/greet",
      "exports": [
        { "name": "hello", "type": "string -> string", "pure": true, "arity": 1 },
        { "name": "farewell", "type": "string -> string", "pure": true, "arity": 1 },
        { "name": "welcome", "type": "string -> string", "pure": true, "arity": 1 }
      ]
    }
  ]
}
```

### Module Detail

```bash
curl http://localhost:8080/api/_meta/modules/api/math
```

### Health Check

```bash
curl http://localhost:8080/api/_health
```

Response:

```json
{
  "status": "ok",
  "modules_count": 2,
  "exports_count": 7
}
```

---

## Interactive API Documentation

`serve-api` provides built-in interactive documentation, similar to FastAPI's `/docs` and `/redoc`:

### Swagger UI

Open `http://localhost:8080/api/_meta/docs` in your browser to get an interactive API explorer where you can:
- Browse all available endpoints
- See request/response schemas
- Try out API calls directly from the browser

### ReDoc

Open `http://localhost:8080/api/_meta/redoc` for a clean, readable API reference document. ReDoc is ideal for sharing with external consumers.

### OpenAPI Spec

The raw OpenAPI 3.1 spec is available at `http://localhost:8080/api/_meta/openapi.json`. You can import this into any OpenAPI-compatible tool (Postman, Insomnia, etc.).

The spec is auto-generated from your AILANG module exports — type signatures are mapped to JSON Schema:

| AILANG Type | JSON Schema |
|-------------|-------------|
| `int` | `{"type": "integer"}` |
| `float` | `{"type": "number"}` |
| `string` | `{"type": "string"}` |
| `bool` | `{"type": "boolean"}` |
| `[int]` | `{"type": "array", "items": {"type": "integer"}}` |

---

## Protocol Support

`serve-api` supports multiple AI agent protocols out of the box.

### MCP (Model Context Protocol)

Expose AILANG functions as MCP tools for use with Claude Desktop, Cursor, and other MCP clients.

**Stdio mode** (for IDE integration):
```bash
ailang serve-api --mcp ./api/
```

**HTTP mode** (served alongside REST endpoints):
```bash
ailang serve-api --mcp-http ./api/
# MCP endpoint at POST /mcp/
```

Each exported AILANG function becomes an MCP tool. Module metadata is available as an MCP resource at `ailang://meta/modules`.

**MCP tool quality features:**

- **Named parameter schemas** — `inputSchema` uses named parameters with JSON Schema types (e.g., `{"filepath": {"type": "string"}}`) instead of generic positional arrays. Types are mapped from AILANG: `string`→`"string"`, `int`→`"integer"`, `float`→`"number"`, `bool`→`"boolean"`, `Json`/records→`"object"`, lists→`"array"`.
- **Doc comment descriptions** — `--` comment lines immediately above a function are used as the MCP tool description. Functions without doc comments fall back to the type signature.
- **MCP-compliant tool names** — All names match the strict regex `^[a-zA-Z0-9_-]{1,64}$` required by Claude Desktop and most current MCP clients. Resolution order: (1) `@mcp_name("name")` author override, (2) bare function name when globally unique (e.g. `mcpParse`), (3) `<lastModuleSegment>_<funcName>` fallback for collisions (e.g. `services_parseCsv`), (4) deterministic hash suffix when truncated to 64 chars. Use `@mcp_name("parse")` to control the exact tool name surfaced to MCP clients.
- **Filtering** — `--routes-only` and `@noexpose` filter module exports in MCP `tools/list`, consistent with HTTP and OpenAPI. `@nomcp` additionally hides an export from `tools/list`/`tools/call` while keeping it served over HTTP/OpenAPI/A2A (MCP-only exclusion). The Go-side built-in `submit_feedback` is not a module export, so these filters do not affect it; use `--no-feedback-tool` to remove it. With `--routes-only`, undocumented non-route helpers are auto-excluded from MCP.
- **Capabilities versus discovery** — `--caps` gates effect execution, not tool discovery. Exports remain advertised in `tools/list` when their effects are not granted and fail only when called. Even `--caps ''` leaves the tool list unchanged. Use `--routes-only`, `@noexpose`, `@nomcp`, and `--no-feedback-tool` to control discovery.
- **Backward compatible** — Tool handlers accept both named parameters (`{"filepath": "doc.pdf"}`) and legacy positional format (`{"args": ["doc.pdf"]}`).
- **Missing-parameter rejection** — A `tools/call` that omits a declared parameter (absent key) or sends it as JSON `null` is rejected with a structured `missing required parameter(s): <names>` error (`isError: true`) **before** the function runs. The names are listed in declaration order. This is type-agnostic — an omitted `int` param is rejected the same way as a `string`. Without this guard the omitted value bound to `nil`→Unit and crashed deep in stdlib (e.g. `_str_len: expected String, got Unit`) with no actionable message. The legacy positional `{"args": [...]}` form is unaffected (it opts out of named binding entirely).

  > **Why MCP rejects but `@route` zero-pads:** MCP `tools/call` is a structured RPC where a declared param is a contract the caller must satisfy, so omission is a caller error worth surfacing loudly. The `@route` HTTP path instead zero-pads missing typed params (see **Zero-value padding for missing parameters** above) so a partially-specified GET/POST can still reach a handler that validates its own inputs. Both avoid the `nil`→Unit crash — MCP by rejecting, `@route` by substituting a well-typed zero — they just pick different points on the strictness spectrum to match their callers.

- **Header auth (`_headers` on MCP)** — A declared `_headers: Json` parameter binds the HTTP headers of the `tools/call` request, the same contract as the REST `@route` path. It is never advertised in `inputSchema`, and a `_headers` key in the client's arguments is ignored, so headers cannot be forged. Keys arrive canonicalized (`Authorization`, `X-Api-Key`). On stdio it binds to an empty object. Use it to take an API key from a header so the agent never holds the key in model context (MCP registries inject secrets only as headers).
- **Optional parameters (`@optional`)** — `@optional("apiKey", ...)` drops the named params from the tool's `required` list; an absent or `null` value binds to the type's zero value (`""`, `0`, `false`, `[]`, `{}`) instead of being rejected. Names are checked at registration: a name that is not a parameter, is `_headers`, or has a type with no zero value (e.g. `Json`) is logged as an `ERROR` and the tool is not registered.

  ```ailang
  -- Key from the argument, else Authorization: Bearer, else X-API-Key
  @optional("apiKey")
  export func whoami(apiKey: string, _headers: Json) -> string { ... }
  ```

  Full example: `examples/runnable/serve_api_mcp_header_auth.ail`.
- **Tool titles and behaviour hints (`@mcp_title`, `@mcp_hints`)** — MCP directories (Anthropic's connector/plugin directory, OpenAI's Plugin Directory) refuse a tool that has no `title` or that declares neither `readOnlyHint` nor `destructiveHint`. `@mcp_title("Parse document")` sets `title`. `@mcp_hints(...)` takes any of `readOnly`, `destructive`, `idempotent`, `openWorld`, and the list is **complete**: a hint you leave out is `false` — including `destructive` and `openWorld`, whose MCP defaults are `true`. So `@mcp_hints("openWorld")` means "writes, additively, to the outside world", and the empty `@mcp_hints()` means "writes, additively, closed-world, not idempotent".

  | Declaration | Emitted `annotations` |
  |---|---|
  | `@mcp_hints("readOnly", "openWorld")` | `readOnlyHint: true, destructiveHint: false, openWorldHint: true` (every hint is an explicit boolean, which OpenAI's directory requires) |
  | `@mcp_hints("destructive", "idempotent")` | `readOnlyHint: false, destructiveHint: true, idempotentHint: true, openWorldHint: false` |
  | none, empty effect row (`-> T` with no `!`, or `! {}`) | `readOnlyHint: true, destructiveHint: false, openWorldHint: false` — the checker proves the function touches nothing |
  | none, effectful | **none** — serve-api will not guess; one startup `WARN` names every such tool |

  An unknown hint word, a duplicate, or `readOnly` with `destructive` is logged as an `ERROR` and the tool is not registered. Purity is read from the **declared effect row**, not the `pure` keyword. The built-in `submit_feedback` tool is annotated (`Send feedback`, additive, open-world). Both MCP implementations — the go-sdk one behind `serve-api` and the stdlib `serveapi/protocol/mcphttp` dispatcher for embedders (`ToolDescriptor.Title` / `.Annotations`, resolved with `protocol.ResolveToolHints`) — emit the same JSON.

  ```ailang
  @mcp_title("Current time")
  @mcp_hints("readOnly", "openWorld")
  export func currentTime() -> int ! {Clock} = now()
  ```

  Full example: `examples/runnable/serve_api_mcp_hints.ail`.

### Listing in the MCP directories (Anthropic, OpenAI)

The Anthropic and OpenAI directories require four things of a listed MCP server:
- every tool has a title and behaviour hints;
- the model never handles a credential;
- account-backed tools use OAuth;
- discovery works before the user signs in.

`serve-api` covers the server side with annotations, and `ailang mcp check` verifies the result.

> **End to end:** [Publish an AILANG service as a Claude / ChatGPT connector](./mcp-connectors.md)
> walks through the annotations, sign-in with `sunholo/mcp_oauth`, file input, `ailang mcp check`
> and the directory submission checklist, using AILANG Parse as the worked example.

**Two surfaces from one module.** When any export uses the annotations below, `serve-api`
mounts a second MCP endpoint:

| Endpoint | For | Differences |
|---|---|---|
| `/mcp/` | agents, CLIs, SDK bridges, the MCP Registry | none: every tool, keys accepted as arguments or headers |
| `/mcp/connect/` | directory listings | no `@mcp_agent_only` tools; `@mcp_secret` params are neither advertised nor accepted; `@mcp_auth("oauth2")` tools need a Bearer token |

| Annotation | Effect |
|---|---|
| `@mcp_auth("oauth2")` | On `/mcp/connect/`, a call without an accepted Bearer token gets **HTTP 401 + `WWW-Authenticate: Bearer resource_metadata="…"`**, which starts the client's sign-in. Needs `--oauth-issuer` and a verifier, or the tool is not registered (ERROR). |
| `@mcp_token_verifier` | The one `(token: string) -> bool` function `serve-api` calls with the Bearer token. It is never a tool, and without `@route` never an HTTP endpoint. |
| `@mcp_secret("apiKey")` | The param is dropped from `/mcp/connect/` and binds its zero value even if a client sends it. It must also be `@optional`. |
| `@mcp_agent_only` | The tool is on `/mcp/` only (for example, device-code sign-in tools). |

**Fail-closed verification.** A verifier error, a panic, more than **5 s**, or more than 32
verifications in flight gives **HTTP 503 + `Retry-After`**, and the tool never runs. The 5 s
deadline is set on the verifier's own effect context, so a hung `Net` call inside it is
cancelled. Pure computation cannot be interrupted, so the in-flight cap bounds that case.

**Resource metadata.** With `--oauth-issuer <url>`, `serve-api` serves
`/.well-known/oauth-protected-resource` and `/.well-known/oauth-protected-resource/mcp/connect/`.
`resource` is the listed URL as the client reached it (honouring `X-Forwarded-Proto`), and
`authorization_servers` lists the issuer. The authorization server itself is separate: the
registry package `sunholo/mcp_oauth` provides one in AILANG, served by the same process
(see [the connector guide](./mcp-connectors.md#3-oauth-with-no-extra-infrastructure-sunholomcp_oauth)).

**Check before you submit:**

```bash
ailang mcp check https://your-service.example.com/mcp/connect/ --target anthropic
```

The checks:
1. titles and hints;
2. no credential-shaped parameters;
3. zero-argument tools callable with `{}`;
4. a 401 with resource metadata that names this URL;
5. the authorization server advertises S256 and CIMD or DCR, and answers within 10 s.

It exits 1 on any FAIL. Each finding cites the vendor requirement it comes from.
With `--target openai` (or `both`) it also checks:

6. every `openai/fileParams` field has OpenAI's four-property file schema;
7. every tool that answers 401 declares `securitySchemes: [{"type":"oauth2"}]`.

**ChatGPT mixed auth.** On `/mcp/connect/`, every tool declares `securitySchemes`, both
top level and mirrored in `_meta` (OpenAI's back-compat mirror):
- `[{"type":"oauth2"}]` for `@mcp_auth("oauth2")` tools;
- `[{"type":"noauth"}]` for everything else.

A refused gated call still answers **HTTP 401 + `WWW-Authenticate`**, which is what Claude acts on.
Its body is now a JSON-RPC tool error for the same request id, `isError: true`, with
`_meta["mcp/www_authenticate"]: ["<the same header value>"]`, which is what ChatGPT reads.
`/mcp/` declares no schemes.

#### Files from the host: `@mcp_file`

`@mcp_file("file")` marks a param as a file the host hands over. ChatGPT fills it with a
download URL for a file the user uploaded (`_meta["openai/fileParams"]`). The param must be
typed as this closed record, inline or as a type alias in the same module:

```ailang
type OpenAIFile = { download_url: string, file_id: string, mime_type: string, file_name: string }

-- Parse a file the user uploaded.
@mcp_title("Parse file")
@mcp_hints("readOnly", "openWorld")
@mcp_file("file")
export func parseFile(file: OpenAIFile) -> string ! {IO} = "got ${file.file_name} at ${file.download_url}"
```

On both MCP surfaces the tool's `tools/list` entry gains two things:
- `_meta: {"openai/fileParams": ["file"]}`;
- for that param, OpenAI's file object schema: `{"type":"object","properties":{download_url,
  file_id, mime_type, file_name: string},"required":["download_url","file_id"],
  "additionalProperties":false}`. OpenAI's Scan Tools rejects any other shape.

Binding:
- `download_url` and `file_id` must be strings. Without them the call is refused before your
  function runs.
- `mime_type` and `file_name` bind `""` when the host leaves them out.
- Any other fields the host sends are dropped.
- The param is required unless it is also `@optional`. An omitted `@optional` file param binds
  the all-empty record, so check `file.file_id == ""`.

The annotation is repeatable (`@mcp_file("a", "b")` or one per param). Naming a param that does
not exist, one that is not the four-string record, or one that is also `@mcp_secret` is a
**load error**: `serve-api` refuses to start.

Your function gets a URL, not bytes. Fetch it yourself under `Net`. Claude has no equivalent;
the `sunholo/mcp_files` upload handoff covers it.

#### Widgets (MCP Apps): `@mcp_ui_resource`, `@mcp_ui`, `@mcp_app_only`

An MCP App is an HTML widget the host renders next to a tool result; claude.ai and ChatGPT both
render them. `serve-api` implements the [ext-apps 2026-01-26](https://github.com/modelcontextprotocol/ext-apps)
server side with three annotations:

| Annotation | On | Effect |
|---|---|---|
| `@mcp_ui_resource("ui://svc/name", "<origin>", ...)` | an exported `() -> string` function that returns HTML | Listed by `resources/list`, served by `resources/read` with `mimeType: "text/html;profile=mcp-app"` and `_meta.ui.csp.connectDomains` = the origins given. `"self"` means this server's own origin, taken from the request, so the service never hard-codes its host. The function is not a tool, and without `@route` it is not an HTTP endpoint. |
| `@mcp_ui("ui://svc/name")` | a tool | Adds `_meta.ui.resourceUri`, plus the pre-GA `_meta["ui/resourceUri"]` that current hosts still read. |
| `@mcp_app_only` | a tool | Adds `_meta.ui.visibility: ["app"]`. The host keeps the tool out of the model's tool list, and the widget can still call it. The tool stays in `tools/list`, because the widget needs it there. |

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

Load errors, so `serve-api` refuses to start:
- an `@mcp_ui` URI that no `@mcp_ui_resource` declares;
- a URI that is not `ui://`;
- a connect domain that is not `"self"` or a bare origin (`https://host[:port]`, no path);
- a resource function that takes arguments or does not return `string`;
- the same URI declared twice.

`"self"` needs an HTTP request. Over stdio, reading such a widget fails with that reason, and its
`resources/list` entry carries no `_meta`. `resourceDomains` is always `[]`, so inline your
scripts and styles. Full example: `examples/runnable/serve_api_mcp_app.ail`.

Full example: `examples/runnable/serve_api_mcp_oauth.ail`. Embedders using
`serveapi/protocol/mcphttp` get the same gate through `Config.Gate` (a `protocol.BearerGate`)
and `ToolDescriptor.Auth`.

### A2A (Agent-to-Agent Protocol)

Google's A2A protocol is enabled with the `--a2a` flag:

- **Agent Card**: `GET /.well-known/agent.json` — lists all functions as skills
- **Task endpoint**: `POST /a2a/` — JSON-RPC 2.0 for function invocation

> **Tip:** To serve a custom agent card (e.g., with additional metadata or a different schema), use `@nowrap @route("GET", "/.well-known/agent.json")` on a function that returns your card as a record. The `@nowrap` annotation ensures the output matches the A2A spec exactly, without the `FunctionCallResponse` envelope.

Example A2A call:
```bash
curl -X POST http://localhost:8080/a2a/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tasks/send",
    "params": {
      "metadata": {"skill_id": "api.math.add"},
      "message": {
        "role": "user",
        "parts": [{"type": "data", "data": {"args": [3, 4]}}]
      }
    }
  }'
```

---

## Strict Module Registration (v0.21.0+)

`serve-api` will refuse to start if a module carrying an `@route` annotation was silently dropped by the under-basePath filter. Before this guard, a deployment layout that placed handler dependencies outside the server's basePath (a common pattern when running from inside a published package's cache directory) could lead to handlers returning structurally-valid but semantically-empty responses — see the [docparse v0.14.1 billing bug post-mortem](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-serveapi-surface-drops.md) for the original incident.

### What happens

After `LoadModules` and before the HTTP listener binds, the server partitions all dropped modules:

- **Fatal** — drop has at least one `@route` function. The server prints a structured error to stderr naming the file, declared module path, basePath, resolved path, and annotations, then exits 1.
- **Non-fatal** — drop has no annotations (e.g. a stdlib resolution edge). The server emits a single `⚠  Dropped N module(s) outside basePath: ...` warning line in the startup banner but continues.

### Escape hatch

Set `AILANG_SERVE_API_ALLOW_DROPS=1` to demote the fatal failure to a strong WARN log line. The server starts and `/api/_health` reports `status: "degraded"` with the dropped modules listed under `dropped_modules`. Intentionally env-var-only (no CLI flag) to force operators to make the bypass explicit in their Dockerfile or deployment manifest, so it doesn't become a permanent local convenience.

```bash
# Recommended: fix the deployment layout
ailang serve-api --caps Net,FS,Env,IO --port 8080 .

# Migration escape hatch — TEMPORARY ONLY
AILANG_SERVE_API_ALLOW_DROPS=1 ailang serve-api --caps Net,FS,Env,IO --port 8080 .
```

### Detecting partial registration

Readiness probes can check `/api/_health` for partial registration:

```json
{
  "status": "degraded",
  "modules_count": 6,
  "exports_count": 18,
  "dropped_modules": [
    {
      "declared": "pkg/sunholo/billing_entitlements/plan",
      "resolved": "/root/.ailang/cache/.../plan.ail",
      "annotations": ["@route"]
    }
  ],
  "dropped_warning": "AILANG_SERVE_API_ALLOW_DROPS is set — service is running with @route-bearing modules dropped"
}
```

A healthy server response omits `dropped_modules` and `dropped_warning` entirely (via `omitempty`):

```json
{"status": "ok", "modules_count": 6, "exports_count": 18}
```

**Status semantics:**

| Drops present | `@route`-bearing drop? | `status`  | `dropped_modules` | Routes traffic? |
|---------------|------------------------|-----------|-------------------|-----------------|
| No            | —                      | `ok`      | omitted           | Yes (healthy)   |
| Yes           | No (stdlib edge etc.)  | `ok`      | populated         | Yes (diagnostic-only) |
| Yes           | Yes (allow-drops set)  | `degraded`| populated         | No (probe routes away) |

Non-annotation drops keep `status: "ok"` so a routine stdlib resolution edge doesn't take a fully-functional revision out of rotation. The drops are still listed in `dropped_modules` for operator visibility. Only `@route`-bearing drops — which mean a customer-facing endpoint is missing — flip `status` to `"degraded"` (which only happens when `AILANG_SERVE_API_ALLOW_DROPS=1` is set, since the server would otherwise have refused to start).

### Resolution options

When `serve-api` fails to start with a fatal drop, the error message lists three paths forward:

1. **Move basePath outward.** If basePath is the package cache directory, change it to a project root that contains all import targets. This is usually the cleanest fix.
2. **Replace relative imports with canonical ones.** Inside a published package, prefer `import pkg/sunholo/foo/bar` over `import ./bar` — relative imports inside a package can resolve to a different physical location than callers reach via the canonical path, surfacing as a silent drop here.
3. **Set `AILANG_SERVE_API_ALLOW_DROPS=1`.** Last resort for migration scenarios; not recommended for production.

## CLI Reference

### `ailang serve-api`

```
Usage: ailang serve-api [flags] <path...>

Serve AILANG module exports as REST API endpoints.

Flags:
  --port PORT          HTTP port (default: 8080)
  --bind ADDR          Host to listen on (default: 127.0.0.1; 0.0.0.0 when PORT env is set)
  --cors               Allow cross-origin requests from every origin (default: off)
  --cors-origin ORIGIN Allow one exact origin, e.g. https://app.example.com (repeatable)
  --frontend PATH      Proxy to Vite dev server at PATH
  --static PATH        Serve static files from PATH
  --static-cache V     Cache-Control on --static 2xx/304: 'immutable' or a max-age in seconds
  --watch              Watch .ail files for changes and hot-reload
  --caps CAPS          Capabilities to grant (comma-separated: IO,FS,Net,AI,Clock,Env)
  --ai MODEL           AI model for AI effect (e.g., gemini-2-5-flash)
  --ai-stub            Use stub AI handler (for testing)
  --verify-contracts   Enable runtime contract validation (requires/ensures)
  --mcp                Run as MCP stdio server (for Claude Desktop, Cursor)
  --mcp-http           Enable MCP HTTP endpoint at /mcp/
  --max-upload-size N  Maximum upload size in bytes (default: 50MB)
  --api-key-header H   HTTP header name for API key authentication
  --api-key-env VAR    Environment variable containing the expected API key
  --routes-only        Only expose @route-annotated functions (skip auto-generated endpoints)
  --no-introspection   Serve no /api/_meta/* and no /api/_health (the paths stay reserved)
  --ws-pass-header H   Give @route("WS") handlers this request header in req.headers (repeatable)
  --no-feedback-tool   Suppress the built-in submit_feedback MCP tool (exact tool surface)
  --oauth-issuer URL   OAuth authorization server for @mcp_auth("oauth2") tools on /mcp/connect/

Arguments:
  <path...>            One or more .ail files or directories
```

**Important:** Flags must come before path arguments.

**Examples:**

```bash
# Serve a single file
ailang serve-api api/handlers.ail

# Serve a directory (finds all .ail files)
ailang serve-api ./api/

# Custom port (flags before paths)
ailang serve-api --port 3000 ./api/

# Reachable from other machines on the LAN (default is loopback only)
ailang serve-api --bind 0.0.0.0 ./api/

# Let one browser origin call the API cross-origin
ailang serve-api --cors-origin https://app.example.com ./api/

# With Vite frontend proxy (development)
ailang serve-api --frontend ./ui ./api/

# With built frontend (production)
ailang serve-api --static ./ui/dist ./api/

# MCP stdio server (for Claude Desktop, Cursor)
ailang serve-api --mcp ./api/

# HTTP server + MCP endpoint at /mcp/
ailang serve-api --mcp-http --cors ./api/

# Only expose @route-annotated functions
ailang serve-api --routes-only ./api/
```

### `ailang init web-app`

```
Usage: ailang init web-app [name]

Scaffold a new AILANG web app project.

Arguments:
  [name]    Project directory name (default: my-ailang-app)
```

---

## Project Structure

After `ailang init web-app myproject`:

```
myproject/
├── api/
│   └── handlers.ail        # AILANG API module
├── ui/
│   ├── package.json         # React 18 + Vite 5 + TypeScript
│   ├── vite.config.ts       # Proxies /api → localhost:8080
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.tsx         # React entry point
│       └── App.tsx          # Demo UI calling AILANG API
├── Makefile                 # Development commands
└── README.md                # Getting started guide
```

### Makefile Targets

```bash
make dev        # Start AILANG API + Vite dev server
make api        # Start only the AILANG API server
make ui         # Start only the Vite dev server
make build      # Build React frontend for production
```

---

## React Integration

### Calling AILANG from React

The scaffold includes a working example in `ui/src/App.tsx`:

```tsx
const callApi = async () => {
  const res = await fetch('/api/api/handlers/hello', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ args: [name || 'World'] }),
  })
  const data = await res.json()
  // data.result = "Hello, World!"
}
```

### TypeScript Types

You can type the default API response:

```typescript
// Default FunctionCallResponse envelope
interface ApiResponse {
  result: unknown
  module: string
  func: string
  elapsed_ms: number
  error?: string
}

// For @nowrap routes, the response is the raw return value.
// Check the X-Elapsed-Ms header for timing.
```

### Fetching Module Metadata

```typescript
const res = await fetch('/api/_meta/modules')
const data = await res.json()
// data.modules[0].exports[0].name = "hello"
// data.modules[0].exports[0].type = "string -> string"
```

---

## Development Workflow

### Adding New API Functions

1. Edit your `.ail` file to add new exported functions
2. If using `--watch`, changes are picked up automatically (hot reload)
3. Without `--watch`, restart `ailang serve-api` to pick up changes
4. New endpoints are automatically available

### Hot Reload

Use `--watch` to automatically recompile modules when `.ail` files change:

```bash
ailang serve-api --watch ./api/
```

**How it works:**
1. The server watches directories containing loaded `.ail` files using `fsnotify`
2. On file save, all caches are invalidated (loader, runtime, engine)
3. The module is recompiled through the pipeline
4. Next API request uses the fresh module

**Graceful degradation:** If a save introduces a compile error, the error is logged but the server continues serving the previous working version. Fix the error and save again.

**Debouncing:** Rapid saves within 200ms are batched into a single reload.

**Limitation:** Dependency-aware reload is not yet supported. If module A imports module B and B changes, only B is reloaded. Save A (or any file) to trigger its reload too.

### Effect Capabilities

By default, `serve-api` only supports **pure functions** (no side effects). AILANG's effect system requires capabilities to be explicitly granted before effectful functions can execute.

#### How It Works

AILANG functions declare their effects in their type signatures:

```ailang
-- Pure function: no capabilities needed
export pure func add(x: int, y: int) -> int = x + y

-- IO effect: requires --caps IO
export func greet(name: string) -> string ! {IO} =
  "Hello, ${name}!"

-- AI effect: requires --caps AI plus --ai MODEL
export func summarize(text: string) -> string ! {AI} =
  ai_call("Summarize this: ${text}")

-- Multiple effects: requires --caps IO,Net
export func fetchAndLog(url: string) -> string ! {IO, Net} {
  let body = http_get(url);
  println("Fetched: ${url}");
  body
}
```

When serving these modules, grant the matching capabilities:

```bash
# Pure functions only (default, no flags needed)
ailang serve-api ./api/

# Grant IO capability
ailang serve-api --caps IO ./api/

# Grant IO and FS capabilities
ailang serve-api --caps IO,FS ./api/

# Grant AI capability with a specific model
ailang serve-api --caps IO,AI --ai gemini-2-5-flash ./api/

# Use stub AI handler for testing (returns fixed responses)
ailang serve-api --caps IO,AI --ai-stub ./api/
```

`--caps` controls which effects may execute; it does not filter HTTP endpoints or
MCP discovery. A function whose effects are not granted is still advertised in
`tools/list` and fails at call time. In particular, `--caps ''` leaves the tool
list unchanged. For discovery filtering, use `--routes-only`, `@noexpose`,
`@nomcp`, and `--no-feedback-tool`.

#### Capability Reference

| Capability | Effect | Enables | Example Builtins |
|------------|--------|---------|-----------------|
| `IO` | `{IO}` | Console I/O | `println`, `readLine` |
| `FS` | `{FS}` | File system access | `readFile`, `writeFile` |
| `Net` | `{Net}` | HTTP requests | `http_get`, `http_post` |
| `AI` | `{AI}` | LLM API calls | `ai_call` |
| `Clock` | `{Clock}` | Time operations | `now`, `sleep` |
| `Env` | `{Env}` | Environment variables | `env_get` |
| `SharedMem` | `{SharedMem}` | In-memory key-value cache | `cache_get`, `cache_set` |
| `SharedIndex` | `{SharedIndex}` | Semantic similarity search | `index_add`, `index_search` |

#### AI Model Configuration

The `--ai` flag specifies which AI model to use for the `AI` effect:

```bash
# OpenAI models (requires OPENAI_API_KEY env var)
ailang serve-api --caps AI --ai gpt-4o ./api/

# Anthropic models (requires ANTHROPIC_API_KEY env var)
ailang serve-api --caps AI --ai claude-sonnet-4-5 ./api/

# Google models (requires GOOGLE_API_KEY or ADC)
ailang serve-api --caps AI --ai gemini-2-5-flash ./api/

# Local Ollama models (requires running Ollama server)
ailang serve-api --caps AI --ai ollama:llama3 ./api/

# Stub handler for testing (no API key needed)
ailang serve-api --caps AI --ai-stub ./api/
```

Model names are resolved via `models.yml` configuration. If not found, the provider is guessed from the model name prefix (`claude-` → Anthropic, `gpt-` → OpenAI, `gemini-` → Google, `ollama:` → Ollama).

#### What Happens Without Capabilities

If an AILANG function uses an effect but the corresponding capability is not granted, the API returns an error:

```bash
# Server started without --caps
ailang serve-api ./api/

# Calling a function that needs IO
curl -X POST http://localhost:8080/api/api/handlers/greet \
  -H "Content-Type: application/json" -d '{"args": ["World"]}'
# {"error":"IO: capability not granted","module":"api/handlers","func":"greet","elapsed_ms":0}
```

To fix: restart with `--caps IO` (or whatever capabilities the function requires).

**Security note:** Capabilities are granted server-wide. All API endpoints share the same capabilities. Only grant capabilities that your AILANG modules actually need.

**Net and user-supplied URLs (v0.52.0+):** with `--caps Net`, serve-api allows `http://`, loopback and the cloud metadata server (`169.254.169.254`) for the whole process, because `sunholo/gcp_auth` fetches tokens from the metadata server. Two rules keep that from reaching user-chosen URLs:

- **Redirect hops** never land on loopback, link-local (metadata) or private addresses, whatever the flags. Direct requests are unchanged, so `gcp_auth` keeps working.
- A function that fetches a URL a client chose should declare **`! {Net[scope=public]}`**. Every Net call in its dynamic extent then refuses loopback and metadata, checked at connect time after DNS, so a hostname resolving to `127.0.0.1` or `169.254.169.254` is refused too. See [Parameterised effects → Net scope](parameterised-effects.md#net-scope-public).

### Frontend Proxy

When using `--frontend ./ui`, the server:
1. Checks for `vite.config.ts` in the frontend directory
2. Starts `npm run dev` as a background process
3. Proxies all non-`/api/` requests to Vite (default port 5173)
4. Provides hot module replacement for React code

### Static Serving

For production, build the frontend and serve statically:

```bash
cd ui && npm run build && cd ..
ailang serve-api ./api/ --static ./ui/dist
```

`--static` sends `Last-Modified` and answers range requests, and sets no `Cache-Control` unless you
ask. For files that are never rewritten (dated or content-hashed names), opt in:

```bash
ailang serve-api ./api/ --static ./ui/dist --static-cache immutable   # public, max-age=31536000, immutable
ailang serve-api ./api/ --static ./ui/dist --static-cache 3600        # public, max-age=3600
```

The header goes on file responses (`200`, `206`) and `304` only — a `404` or a directory listing is
never marked cacheable. It applies to
`--static` files, not to API routes or the `--frontend` dev proxy.

---

## Custom Routes (v0.9.4+)

Use `@route` annotations to define custom URL paths and HTTP methods:

```ailang
module docparse/api

@route("POST", "/api/v1/parse")
export func parse(content: string) -> ParseResult ! {IO}
  parseDocument(content)

@route("GET", "/api/v1/formats")
export pure func listFormats() -> [string]
  ["docx", "pdf", "epub", "html"]

@route("GET", "/health")
export pure func health() -> {status: string}
  {status = "ok"}
```

Custom routes are registered before the auto-generated catch-all routes, so they take precedence. They appear with their custom paths in the OpenAPI spec and A2A Agent Card.

### Route Annotations

| Annotation | Purpose |
|------------|---------|
| `@route("METHOD", "/path")` | Custom URL path and HTTP method |
| `@raw` | Receive full `HttpRequest` record (headers, body, method, query) instead of parsed args |
| `@nowrap` | Return raw JSON instead of the `FunctionCallResponse` envelope |
| `@noexpose` | Hide exported function from HTTP endpoints (still importable by other modules) |
| `@nomcp` | Hide from the MCP tool surface ONLY — still served over HTTP, OpenAPI, and A2A (not reset by `@route`) |
| `@mcp_name("name")` | Override the auto-generated MCP tool name for this function |
| `@mcp_title("Title")` | MCP display title (directory listings require one) |
| `@mcp_hints("readOnly", ...)` | MCP behaviour hints: `readOnly`, `destructive`, `idempotent`, `openWorld` — the complete list (absent = false) |
| `@mcp_auth("oauth2")` | Gate the tool behind OAuth on the listed surface `/mcp/connect/` (needs `--oauth-issuer` + `@mcp_token_verifier`) |
| `@mcp_token_verifier` | The `(token: string) -> bool` Bearer-token verifier; never a tool or HTTP endpoint |
| `@mcp_secret("p")` | Drop param `p` from `/mcp/connect/` (must also be `@optional`) |
| `@mcp_agent_only` | Serve the tool on `/mcp/` only |
| `@verify(depth: N)` | Runtime contract validation |

Multiple annotations can be combined:

```ailang
@route("POST", "/api/v1/compute")
@verify(depth: 3)
export func compute(x: int) -> int ! {}
  x * x
```

Supported HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS.

### `@raw` — Raw HTTP Request Access

Use `@raw` with `@route` to receive the full HTTP request context instead of parsed arguments. The function receives a record with `body`, `headers`, `method`, `path`, and `query` fields:

```ailang
import std/json (Json, getString)

@raw
@route("POST", "/webhooks/stripe")
export func handle(req: {body: string, headers: Json, method: string, path: string, query: Json}) -> string ! {IO}
  let sig = getString(req.headers, "Stripe-Signature")
  verifyAndProcess(req.body, sig)
```

Headers and query parameters are `Json` values — use `getString`, `getInt`, etc. to extract fields.

### Request Headers in `@route` (without `@raw`)

If you need HTTP request headers but want to keep normal argument parsing (including multipart file uploads), declare a `_headers` parameter instead of switching to `@raw`:

```ailang
import std/json (Json, getString)

@route("POST", "/api/v1/secure-parse")
export func secureParse(content: string, _headers: Json) -> string ! {IO} =
  let apiKey = getString(_headers, "x-api-key") in
  if apiKey == "" then "error: missing x-api-key header"
  else "authenticated: ${content}"
```

The `_headers` parameter receives all HTTP request headers as a `Json` value. Other parameters are parsed normally from the request body (JSON or multipart). This is useful for:

- **API key authentication** — read `Authorization` or custom auth headers
- **Unstructured API compatibility** — multipart file upload + `unstructured-api-key` header
- **Content negotiation** — read `Accept` header to choose response format

> **Note:** The `_headers` name is a convention (matching the response `_headers` pattern). Only parameters named exactly `_headers` are injected with request headers.

### `@nowrap` — Raw JSON Output

By default, every handler wraps its return value in a `FunctionCallResponse` envelope:

```json
{"result": ..., "module": "...", "func": "...", "elapsed_ms": 5}
```

Add `@nowrap` to skip the envelope and write the function's return value directly as pretty-printed JSON. Timing remains available via the `X-Elapsed-Ms` response header.

```ailang
@nowrap
@route("GET", "/api/v1/formats")
export pure func listFormats() -> [string]
  ["docx", "pdf", "epub", "html"]
```

```bash
curl http://localhost:8080/api/v1/formats
# [
#   "docx",
#   "pdf",
#   "epub",
#   "html"
# ]
# (X-Elapsed-Ms: 0 in response headers)
```

`@nowrap` composes with `@raw` for full control over both input and output:

```ailang
@raw
@nowrap
@route("POST", "/api/v1/echo")
export func echo(req: {body: string, headers: Json, method: string, path: string, query: Json}) -> {received: string} ! {IO}
  {received = req.body}
```

**Use cases:**

- **A2A agent cards** — serve `/.well-known/agent.json` with a custom schema
- **OpenID discovery documents** — `/.well-known/openid-configuration`
- **REST endpoints** — where consumers expect a specific JSON schema without an envelope

#### Custom Response Headers with `@nowrap`

`@nowrap` functions can set custom HTTP headers by including a `_headers` field in the return record. The `_headers` field is extracted as HTTP headers and excluded from the JSON response body:

```ailang
@nowrap
@route("POST", "/api/v1/parse")
export func parseFile(path: string) -> {data: string, count: int, _headers: {string: string}} ! {IO}
  let result = parse(path)
  {
    data = result.text,
    count = result.elementCount,
    _headers = {
      "X-Request-Id" = generateId(),
      "X-RateLimit-Remaining" = "99"
    }
  }
```

```bash
curl -s -D- http://localhost:8080/api/v1/parse -d '{"path": "test.docx"}'
# HTTP/1.1 200 OK
# Content-Type: application/json
# X-Request-Id: req_abc123
# X-RateLimit-Remaining: 99
# X-Elapsed-Ms: 42
#
# {"data": "parsed content", "count": 15}
```

> **Note:** The `_headers` convention is consistent with the existing `_body`/`_status`/`_headers` pattern used for [binary responses](#binary-response-v094). For simple JSON responses that just need extra headers, `@nowrap` with `_headers` is more ergonomic than the full `_body` pattern.

#### JSON Auto-Unwrap with `@nowrap`

When a `@nowrap` function returns a string that is a valid JSON object or array (e.g., from `encode(jo(...))`), serve-api writes it as raw JSON instead of double-encoding it:

```ailang
@nowrap
@route("GET", "/api/v1/health")
export func health() -> string ! {}
  encode(jo([kv("status", js("healthy"))]))
```

```bash
curl http://localhost:8080/api/v1/health
# {"status":"healthy"}     ← raw JSON, not "{\"status\":\"healthy\"}"
```

This only triggers for JSON objects (`{...}`) and arrays (`[...]`). Plain strings, numbers, and booleans are still JSON-encoded normally.

---

### `@noexpose` — Hide from HTTP Endpoints

Use `@noexpose` on exported functions that should be importable by other modules but NOT accessible as HTTP endpoints:

```ailang
module billing/internal

-- Public API endpoint
@nowrap
@route("GET", "/api/v1/usage")
export func getUsage(userId: string) -> string ! {IO}
  encode(jo([kv("requests", ji(lookupUsage(userId)))]))

-- Exported for cross-module import, hidden from HTTP
@noexpose
export func generateApiKey(userId: string) -> string ! {IO}
  apiKeyGenHexParts(userId, timestamp())

-- Also hidden from HTTP
@noexpose
export func validateApiKey(key: string) -> bool ! {IO}
  checkKeyInStore(key)
```

`@noexpose` functions:
- Are **not** accessible via `POST /api/{module}/{function}`
- Are **not** included in the OpenAPI spec or A2A Agent Card
- Are still importable by other AILANG modules via `import`
- If a function has both `@route` and `@noexpose`, the `@route` takes precedence (it remains exposed)

---

### `@nomcp` — Hide from the MCP Tool Surface Only

Use `@nomcp` on a handler that should be reachable over HTTP (and appear in the OpenAPI spec and A2A card) but should NOT be advertised as an agent-callable MCP tool. This is the right choice for internal/observability endpoints — health probes, metrics, key-usage dumps — that you want a human or a `curl` to hit, but that you do not want an LLM to discover in `tools/list` and call:

```ailang
module billing/api

-- Public agent tool: appears in MCP tools/list AND served over HTTP.
@route("GET", "/api/v1/usage")
export func getUsage(userId: string) -> string ! {IO}
  lookupUsage(userId)

-- Served over HTTP + OpenAPI + A2A, but ABSENT from the MCP tool surface.
@nomcp
@route("GET", "/internal/key-usage")
export func getKeyUsage() -> string ! {IO}
  dumpKeyUsage()
```

`@nomcp` functions:
- Are **excluded** from MCP `tools/list` and `tools/call` (a call to the excluded tool name errors as unregistered)
- Are **still** served over HTTP (`GET /internal/key-usage` answers 200)
- Are **still** present in the OpenAPI spec and the A2A Agent Card
- Keep `@nomcp` even when combined with `@route` — unlike `@noexpose`, `@route` does **not** reset it

`@noexpose` and `@nomcp` are independent and can be combined. They differ in which surfaces they hide:

| Annotation | HTTP endpoint | OpenAPI / A2A | MCP `tools/list` + `tools/call` | Reset by `@route`? |
|------------|:-------------:|:-------------:|:-------------------------------:|:------------------:|
| `@noexpose` | hidden | hidden | hidden | yes (`@route` re-exposes) |
| `@nomcp` | **served** | **present** | hidden | no (survives `@route`) |

---

### `@mcp_name` — Override the MCP Tool Name

Claude Desktop and most MCP clients enforce a strict tool name regex: `^[a-zA-Z0-9_-]{1,64}$` — no dots, no slashes, max 64 characters. AILANG generates compliant names automatically, but you can override the auto-generated name with `@mcp_name("name")`:

```ailang
module docparse/services/mcp_tools

-- Parse a document and return structured content.
@mcp_name("parse")
@route("POST", "/api/v1/mcp/parse")
export func mcpParse(content: string) -> string ! {IO}
  parseDocument(content)
```

Without `@mcp_name`, AILANG uses the following resolution order:

1. **Bare function name** if globally unique among all exposed exports — e.g. `mcpParse`.
2. **`<lastModuleSegment>_<funcName>`** when the bare name collides — e.g. `services_parseCsv`.
3. **64-char truncation with deterministic hash** for very long names.

Use `@mcp_name` when:
- You want a short, branded name regardless of module structure (e.g. `parse` instead of `mcp_tools_mcpParse`).
- Two different modules export functions with the same name and you want an unambiguous label.
- You're integrating with an MCP client that expects a specific tool name.

The name you provide must already match `^[a-zA-Z0-9_-]{1,64}$`. Invalid `@mcp_name` values are logged at startup and the affected tool is skipped.

---

### `--routes-only` — Restrict to @route Endpoints

Use `--routes-only` to limit the API surface to only `@route`-annotated functions, hiding all auto-generated endpoints:

```bash
ailang serve-api --routes-only ./api/
```

This is useful when your project has many exported functions for cross-module use but only a few intentional API endpoints. Without `--routes-only`, all exports become HTTP endpoints.

`--routes-only` and `@noexpose` compose independently:
- `--routes-only` hides **all** non-`@route` exports
- `@noexpose` hides **specific** exports regardless of `--routes-only`
- `@route` functions are always exposed

### `--no-feedback-tool` — Exact MCP Tool Surface

AILANG normally registers the Go-side `submit_feedback` built-in in addition to
module exports. Pass `--no-feedback-tool` to suppress that built-in without
changing which user exports are exposed:

```bash
ailang serve-api --mcp --routes-only --no-feedback-tool ./api/
```

This combination produces exactly the `@route` export set. The flag behaves
identically for stdio `--mcp` and HTTP `--mcp-http`, because both use the same
MCP server construction path. A2A agent-card skills never include the
MCP-only built-in.

---

## File Upload (v0.9.4+)

Functions can accept file uploads via `multipart/form-data`. File fields arrive as `Bytes` values:

```ailang
@route("POST", "/api/v1/upload")
export func processFile(file: Bytes) -> {name: string, size: int} ! {IO}
  {name = bytesFilename(file), size = bytesLength(file)}
```

```bash
curl -F "file=@document.pdf" http://localhost:8080/api/v1/upload
```

Upload size limit: 50MB default, configurable via `--max-upload-size`.

### Upload Builtins

| Function | Type | Description |
|----------|------|-------------|
| `bytesFilename(b)` | `Bytes -> string` | Original upload filename |
| `bytesMimeType(b)` | `Bytes -> string` | Upload MIME type |
| `bytesLength(b)` | `Bytes -> int` | Length in bytes |
| `bytesToString(b)` | `Bytes -> string` | Decode as UTF-8 |

---

## Binary Response (v0.9.4+)

To return raw binary files (not JSON), return a record with `_body`, `_status`, and `_headers` fields:

```ailang
@route("POST", "/api/v1/convert")
export func convertToDocx(file: Bytes) -> {_body: Bytes, _status: int, _headers: {string: string}} ! {IO}
  let result = convert(file, "docx")
  {
    _body = result,
    _status = 200,
    _headers = {
      "Content-Type" = "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      "Content-Disposition" = "attachment; filename=\"output.docx\""
    }
  }
```

The server detects `_body` and sends a raw HTTP response instead of JSON-wrapping.

---

## Authentication (v0.9.4+)

API key authentication via CLI flags:

```bash
ailang serve-api app.ail \
  --api-key-header "x-api-key" \
  --api-key-env "DOCPARSE_API_KEY"
```

- Requests without a valid key get 401 Unauthorized
- Also accepts `Authorization: Bearer <token>` as fallback
- Meta endpoints (`/api/_health`, `/api/_meta/*`), MCP, and A2A bypass auth

---

## Binding & CORS (v0.44.0+)

serve-api exposes nothing beyond the machine, and grants no cross-origin access, unless you ask for it.

**Bind address.** The default is `127.0.0.1`, so only processes on the same machine can connect. When the `PORT` environment variable is set (Cloud Run injects it), the default becomes `0.0.0.0`, because Cloud Run requires the wildcard. `--bind ADDR` always wins, for example `--bind 0.0.0.0` for the LAN or `--bind ::1` for IPv6 loopback. This is the same rule `ailang server --bind` follows.

serve-api binds the port **before** it prints its startup banner. If the port is taken, it exits non-zero with `listen tcp 127.0.0.1:N: bind: address already in use` and prints no banner.

**CORS** has three modes:

| Mode | Flag | What a browser page on another origin gets |
|------|------|--------------------------------------------|
| off (default) | none | No `Access-Control-Allow-*` headers, so the page cannot read any response. |
| any | `--cors` | `Access-Control-Allow-Origin: *` on every API route, and preflights answered 204. |
| allowlist | `--cors-origin ORIGIN` (repeatable) | Listed origins get their exact origin echoed back, with `Vary: Origin`. A request from an **unlisted** origin that is not `GET`/`HEAD` (a preflight or a `POST`) gets **403 before the function runs**. A same-origin request (its `Origin` names the host:port it was sent to, e.g. a page serve-api serves itself) passes without being listed. |

`--cors` and `--cors-origin` together are a startup error. Each origin must be written exactly the way a browser sends it, `scheme://host[:port]` with no path or trailing slash: `https://daneel.example.ts.net`, `http://localhost:5173`.

Why the allowlist refuses the call and not only the response: CORS stops a page from *reading* the answer, but a `POST` with `Content-Type: text/plain` is a CORS "simple request" that a browser sends without any preflight. Without the 403, a page on any origin could still make the function run. **Off mode does not refuse such calls**, so if a handler spends money or acts on someone's behalf, set `--cors-origin` to the origins that may call it.

Requests with no `Origin` header (curl, server-to-server calls, MCP clients) are never affected. A browser also sends `Origin` on same-origin `POST`s, so in allowlist mode you must list your own page's origin too, including when it is served by `--static` or reached through a proxy such as `tailscale serve`.

`/mcp/` and the `--static`/`--frontend` routes are not CORS-wrapped.

**Tailnet-only example** (the backend listens on loopback, and `tailscale serve` is the only way in):

```bash
ailang serve-api --bind 127.0.0.1 --port 8791 \
  --cors-origin https://myhost.tailnet-name.ts.net ./api/
tailscale serve --https=8791 http://127.0.0.1:8791
```

**Migrating from v0.43.x:** add `--bind 0.0.0.0` if other devices reached your dev server, and `--cors` (or better, `--cors-origin ...`) if a page on another origin called it. Container images that set `PORT` keep binding `0.0.0.0` with no change.

---

## WebSocket Routes & the Bridge (v0.44.0+)

`@route("WS", "/path")` makes serve-api accept a WebSocket upgrade and call the handler **once for
the life of the connection**. The browser leg arrives as an ordinary `std/stream` `StreamConn`, so
`transmit`, `onEvent` and `disconnect` work on it unchanged. The route needs `--caps Stream`.

```ailang
import std/stream (connect, disconnect, StreamConn)
import std/stream/bridge (bridge, BridgeFrame, Verdict, Forward, Drop, UpBin, UpstreamClosed)
import std/result (Ok, Err)

pure func muteAudio(n: int, f: BridgeFrame) -> (int, Verdict) =
  match f { UpBin(_) => (n + 1, Drop), _ => (n, Forward) }

@route("WS", "/live")
export func live(client: StreamConn) -> unit ! {Stream} {
  match connect("wss://upstream.example/ws", { headers: [] }) {
    Ok(up) => match bridge(client, up, 0, muteAudio) {
      (_, UpstreamClosed(_, _)) => disconnect(client),
      _ => ()
    },
    Err(_) => disconnect(client)
  }
}
```

The full example, a speech gate with an offline `foldl` replay, is
[`examples/serveapi_ws_bridge.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/serveapi_ws_bridge.ail).

**Handler shape.** `handler(client: StreamConn)`, or `handler(client: StreamConn, req: R)` where `R`
is a record of any of `path: string`, `query: string`, `origin: string` and
`headers: [{name: string, value: string}]`. serve-api builds exactly the fields you declare (a type
alias for `R` gets `{path, query, origin}`). The effect row must include `Stream`, and every effect it names
must be granted by `--caps`. `@raw` and `@nowrap` do not apply. serve-api refuses to start when any
of these is wrong. When the handler returns, serve-api closes the client (1000, or 1011 if the call
failed) and every connection the session opened.

**Before the upgrade**, serve-api refuses:

| Request | Status |
|---|---|
| A plain `GET` with no `Upgrade: websocket` | 426 |
| `Origin` missing, `null`, or neither same-origin nor in `--cors-origin` | 403 |
| API key configured (`--api-key-header`/`--api-key-env`) and not presented | 401 |
| More than `--ws-max-sessions` live sessions (default 4) | 503 |

CORS does not govern WebSockets, so the `Origin` check is what stops another page in the user's
browser from opening the socket. The `--cors-origin` list is the WS allowlist too. **A WS route on
a non-loopback `--bind` needs a `--cors-origin` allowlist**, or serve-api will not start; `--cors`
(every origin) does not count. A browser cannot set headers on `new WebSocket()`, so it presents an
API key as subprotocols: `new WebSocket(url, ["ailang.v1", "ailang.key." + key])`. The server
selects `ailang.v1` and never echoes the key. Query-string keys are not accepted.

**Request headers (`--ws-pass-header`).** A WS handler sees no request headers unless the operator
names them:

```bash
ailang serve-api --caps Stream --bind 127.0.0.1 --ws-pass-header Tailscale-User-Login ./server/
```

```ailang
@route("WS", "/live")
export func live(client: StreamConn, req: {path: string, headers: [{name: string, value: string}]}) -> unit ! {Stream} = ...
```

`req.headers` then holds only the named headers: names lower-cased (`tailscale-user-login`), one entry
per received header line in flag order, no entry when the header is absent. The program cannot widen
the list, so a page cannot smuggle a header in. `Authorization`, `Proxy-Authorization`, `Cookie`,
`Sec-WebSocket-Protocol`, `Sec-WebSocket-Key` and the `--api-key-header` cannot be named — serve-api
refuses to start — so credentials never reach program values or traces. A passed header is only as
trustworthy as the proxy in front: `tailscale serve` overwrites `Tailscale-User-Login`, but a process
that reaches the listener directly can send anything, so pass identity headers only with a loopback
`--bind` behind the proxy that stamps them. HTTP `@route` handlers are unchanged: `_headers`/`@raw`
still receive every header.

**Sessions are isolated.** Each connection gets its own `StreamContext`: its own connection IDs,
its own `MaxConnections` (4 legs by default), and its own `--stream-max-duration` and
`--stream-idle-timeout`. The defaults are 5 minutes and 60 seconds; a voice session usually needs
a longer ceiling, such as `--stream-max-duration 30m`. Each direction queues at most
`--ws-queue-frames` frames (default 64). When the queue is full the reader blocks, and TCP pushes
back on the sender. The runtime never drops a frame on its own.

One message on either leg may be at most `--stream-max-message` bytes (default 1MB). Inbound
memory is therefore bounded by about `--ws-max-sessions × --ws-queue-frames × --stream-max-message`,
which is 256MB at the defaults. On a public endpoint whose clients send only small messages, lower
the cap (for example `64KB`). For a trusted upstream that sends large single messages (images,
files, video keyframes), raise it. See [Message size](/docs/guides/streaming#message-size---stream-max-message).

### `bridge`: one AILANG verdict per frame

`bridge(client, up, init, step)` relays two connections. `step : (s, BridgeFrame) -> (s, Verdict)`
runs once per data frame in either direction, in arrival order, and threads its state. The state is
a value, so the same step replays offline with `foldl` over a recorded frame list.

| `BridgeFrame` | | `Verdict` | Effect |
|---|---|---|---|
| `ClientText(string)` | browser → upstream | `Forward` | Go re-sends the **original** bytes |
| `ClientBin(bytes)` | browser → upstream | `Drop` | nothing is sent |
| `UpText(string)` | upstream → browser | `ReplaceText(s)` / `ReplaceBin(b)` | the replacement is sent onward |
| `UpBin(bytes)` | upstream → browser | `CloseBridge(code, reason)` | the client is closed with `code`, and the bridge ends |

`bridge` returns `(state, BridgeEnd)`, one of `ClientClosed(code, reason)`,
`UpstreamClosed(code, reason)`, `ClosedByVerdict(code, reason)`, `TimedOut(msg)` or
`StepFailed(msg)`. A step that fails closes the client with 1011. The upstream is always closed
when `bridge` returns. The client is closed too, except on `UpstreamClosed`, where the handler
still holds it and may dial a new upstream. Every data frame charges `Stream.recv`, and every frame
sent charges `Stream.send`. The step's effects join the caller's row, so a logging step shows up in
the handler's signature.

Because the step runs when a frame is **dequeued**, a barge-in policy can mark `interrupted` and
`Drop` the model audio already queued behind the interruption. A proxy that sees frames only after
they are sent cannot do that.

`--ws-decision-log` writes one JSON line per frame to the server log, never with a payload:

```
[ws-bridge] {"call_id":"ws-3","seq":17,"dir":"up","kind":"bin","bytes":16384,"verdict":"Drop","step_us":31}
```

### Upstream credentials: `--stream-credential`

Do not put an upstream token in `connect`'s headers: it would be an AILANG value. Bind it to the
host instead:

```bash
ailang serve-api --bind 100.101.102.103 --caps Stream \
  --cors-origin https://studio.tailnet-name.ts.net \
  --stream-credential us-central1-aiplatform.googleapis.com=gcp-key-file:/secrets/daneel-sa.json \
  --stream-max-duration 30m ./live/
```

serve-api adds `Authorization: Bearer …` at dial time, in Go, only for `wss://` to that exact host
and port (a binding without a port means 443). It never applies the binding to `ws://`, another
host or another port. The program never sees the token, so it cannot reach a frame, the trace or a
log through program code. A program that sends its own `Authorization` to a bound host fails with
`credential binding: Authorization header supplied by program for bound host …`.

| Source | Meaning |
|---|---|
| `gcp-key-file:PATH` | A Google credentials file: `service_account`, `impersonated_service_account` or `external_account`. A gcloud user login (`authorized_user`) is refused. |
| `gcp-metadata` | The GCE/Cloud Run metadata server's service account, named explicitly |
| `bearer-file:PATH` | A token file that another tool keeps fresh, read on every dial |

There is no implicit Application Default Credentials lookup and no gcloud fallback. Tokens are
cached and refreshed 5 minutes before expiry. Each source is fetched once at startup, so a broken
source fails the launch instead of the first call.

Separately from the binding, the trace never records credential values. A header-shaped
`{name: "Authorization", value: …}` value, a `("Cookie", …)` pair, a credential-named record field
and any `Bearer …`/`Basic …` string are written as `[REDACTED]` in traces at every tier.

## Concurrency (v0.9.4+)

`serve-api` handles concurrent requests safely. Each HTTP request gets its own isolated evaluator via `Fork()` — there is no shared mutable state between requests. Go's `net/http` creates a goroutine per request, and AILANG's evaluator is designed to work correctly in this model.

**No async server needed** — Go's built-in concurrency handles everything. You do NOT need an event loop, async runtime, or worker pool.

### Cloud Run Deployment

```yaml
# Full concurrency — one container handles 80 simultaneous requests
spec:
  containerConcurrency: 80  # Cloud Run default
```

You do NOT need `containerConcurrency: 1`. A single instance serves many concurrent requests efficiently.

### Performance

Sequential and concurrent performance scale linearly:

```
Sequential 5x DOCX parse:  285ms (57ms × 5)
Concurrent 5x DOCX parse:  261ms (near-perfect scaling)
```

### Testing Concurrency

Use the included test script:

```bash
# Simple modules (no effects):
./tools/test-concurrency.sh examples/web_api_demo/api/

# With capabilities (effectful modules):
CAPS=IO,FS,Env AI_STUB=1 ./tools/test-concurrency.sh path/to/modules/ 8081

# With debug tracing:
DEBUG_CONCURRENCY=1 CAPS=IO,FS ./tools/test-concurrency.sh path/to/modules/
```

### Bash Testing Pitfall

> **Do NOT use `2>&1 | tee` when starting the server.**
>
> Go's HTTP response flushing interacts badly with pipe-based stderr redirects.
> Responses complete but `wait` doesn't see them, making requests appear to hang.
>
> **Use instead:**
> ```bash
> # Correct — redirect to file:
> ailang serve-api ./api/ > /tmp/server.log 2>&1 &
>
> # Correct — discard output:
> ailang serve-api ./api/ > /dev/null 2>&1 &
>
> # WRONG — causes apparent hangs:
> ailang serve-api ./api/ 2>&1 | tee /tmp/server.log &
> ```

### Structured Logging (Cloud Run, jq, Loki) — v0.37.3+

`Debug.log` is the logging channel for a served handler. The server flushes
every line to **stderr** after each request, with one rule:

- A line that is a **bare JSON object** is written **verbatim**, on its own
  line — no timestamp, no `[Debug]` prefix. Cloud Logging (and any JSON-line
  consumer) parses it as `jsonPayload` and lifts the `severity` field.
- Any other line is written as `YYYY/MM/DD HH:MM:SS [Debug] <text>`.
- A **failed `Debug.check`** is emitted as
  `{"severity":"ERROR","message":"assertion failed: <msg>","location":"<loc>","source":"Debug.check"}`
  so a `severity>=ERROR` alert catches it.

```ailang
import std/debug as Debug

-- @route POST /order
export func order(id: string) -> string =
  let _ = Debug.log("{\"severity\":\"ERROR\",\"message\":\"payment declined\",\"order\":\"" ++ id ++ "\"}") in
  "declined"
```

stderr (exactly what Cloud Logging receives):
```
{"severity":"ERROR","message":"payment declined","order":"42"}
```

Rules of thumb: one JSON object per line (a multi-line pretty-printed object
is treated as text); use Google's severity names (`DEBUG`, `INFO`, `WARNING`,
`ERROR`); `--log-level` filters structured lines by that field — a structured
line with no `severity` always passes, and a failed check always surfaces.
Debug output is always emitted — no `--caps` is needed for it.

### Debug Tracing

Set `DEBUG_CONCURRENCY=1` to trace per-request evaluator lifecycle:

```bash
DEBUG_CONCURRENCY=1 ailang serve-api --caps IO,FS --port 8080 ./api/ > /tmp/server.log 2>&1 &
# Then check the log:
grep CONCURRENCY /tmp/server.log
```

Output shows goroutine ID at each stage:
```
[CONCURRENCY] Fork evaluator for api/main.health (goroutine 42)
[CONCURRENCY] Calling api/main.health (goroutine 42)
[CONCURRENCY] Done api/main.health (goroutine 42, err=<nil>)
```

---

## Error Handling with Result Types (v0.11.0+)

Functions that return `Result[T, E]` types get automatic HTTP status code mapping:

| Return value | HTTP status | Body |
|---|---|---|
| `Ok(value)` | 200 | The inner value (unwrapped) |
| `Err("message")` | 400 | `{"error": "message", ...}` |
| `Err({_status: 404, message: "not found"})` | 404 | `{"error": {"message": "not found"}, ...}` |
| Non-Result types | 200 | The value as-is |

### Default behavior

When a function returns `Err(value)`, the HTTP status defaults to **400 Bad Request**. The error payload is included in the response body.

### Custom status codes

To control the HTTP status code, return `Err` with a record containing a `_status` field:

```ailang
@route("GET", "/users/:id")
export func getUser(id: string) -> Result[string, {_status: int, message: string}] ! {Net, FS} =
  match findUser(id) with
  | Some(user) -> Ok(encode(userToJson(user)))
  | None -> Err({_status: 404, message: "user not found"})
```

The `_status` field is extracted for the HTTP status and stripped from the response body, following the same convention as `@raw` responses.

### With @nowrap

`@nowrap` endpoints also respect Result error status codes. The error payload is written directly without the `FunctionCallResponse` envelope:

```bash
# Err("amount must be positive") with @nowrap
HTTP/1.1 400 Bad Request
"amount must be positive"
```

### Result.Ok unwrapping

`Ok(value)` responses are automatically unwrapped — the inner value is returned directly, not wrapped in a `{"__tag": "Ok", ...}` structure.

### Router error envelope

Router-layer errors (requests that fail **before** your AILANG code runs) return a typed envelope in the `error_detail` field alongside the existing flat `error` string. AI agents and SDKs should prefer matching on `error_detail.code` rather than parsing the human-readable `error` message.

**Shape:**

```json
{
  "error": "No route registered for POST /api/v1/auth/device/token",
  "error_detail": {
    "code": "ROUTE_NOT_FOUND",
    "message": "No route registered for POST /api/v1/auth/device/token",
    "retryable": false,
    "suggested_fix": "Did you mean POST /api/v1/auth/device/poll?",
    "available_routes": [
      "POST /api/v1/auth/device",
      "POST /api/v1/auth/device/poll",
      "POST /api/v1/auth/device/approve"
    ]
  },
  "module": "",
  "func": "",
  "elapsed_ms": 0
}
```

**Backward compatibility:** the flat `error` field is always populated and mirrors `error_detail.message`, so existing clients that parse the top-level `error` string keep working unchanged. New clients should match on `error_detail.code`.

**Router error codes:**

| Code | HTTP | Meaning |
|------|------|---------|
| `ROUTE_NOT_FOUND` | 404 | Request path didn't match any registered `@route` on a route-driven server. Includes `suggested_fix` when a close match exists and a bounded `available_routes` list. |
| `MODULE_NOT_LOADED` | 404 | Legacy `/api/{module}/{func}` dispatch on a no-`@route` server: the parsed module isn't loaded. Preserves historical message text. |
| `FUNCTION_NOT_FOUND` | 404 | Module is loaded but the requested function doesn't exist, OR the function is hidden via `@noexpose` / `--routes-only`. Intentionally indistinguishable so `@noexpose` reveals nothing to external callers. |
| `METHOD_NOT_ALLOWED` | 405 | Request method doesn't match the `@route` method, or isn't `POST`/`GET` for the catch-all dispatch handler. |

**Which code fires when:**

1. If the server has **any** `@route` registered and no route matches → `ROUTE_NOT_FOUND`. This is the common case for route-driven deployments and is what AI agents see when they typo a URL.
2. If the server has **zero** `@route`s registered AND the parsed `{module}/{func}` module isn't loaded → `MODULE_NOT_LOADED`. Legacy dispatch-only servers keep their historical error text.
3. Module loaded, function missing (or hidden) → `FUNCTION_NOT_FOUND`.

**Runtime errors** (panics, `Result.Err`, coercion failures) use a **different** envelope path — see the "Error Handling with Result Types" section above. The router envelope only covers errors from the dispatch layer, before your function runs.

---

## Embedding MCP and A2A in a Go host

Applications that already own an HTTP server can expose a session-specific MCP
and A2A surface through the public `serveapi` package:

```go
api, err := serveapi.New(serveapi.Config{
    Resolver: worldResolver,
    Tools:    worldCapabilities,
    Invoker:  worldInvoker,
    Agent: serveapi.AgentInfo{
        Name: "Ailang World",
        Version: "1.0.0",
    },
})
if err != nil {
    log.Fatal(err)
}

mux := http.NewServeMux()
api.Mount(mux)
server := &http.Server{Addr: ":8080", Handler: authMiddleware(mux)}
log.Fatal(server.ListenAndServe())
```

`Mount` registers `/mcp/`, `/.well-known/agent.json`, and `/a2a/`. A host that
needs different paths can mount `MCPHandler()` and `A2AHandler()` itself.

Hosts that need the shared wire contract but provide their own handlers can
instead import `github.com/sunholo-data/ailang/serveapi/protocol`. That package
contains descriptor types and validation, `CallerSurface`/`AuthorizedSurface`,
host interfaces, A2A wire types and writers, MCP envelope framing and
`RequestID`, and the wire-error taxonomy. Its build closure is standard-library
only; `make check-protocol-closure` measures and enforces that guarantee in CI.

The split is contract versus machinery: `serveapi/protocol` does not provide
HTTP handlers or callback bounding. Import `serveapi` for the ready-made MCP and
A2A handlers and bounded callback runner. The MCP dispatcher is SDK-free;
`make check-protocol-closure` also checks the handler and facade build closures.

### Typed MCP host errors

An error returned by `Invoker.Invoke` can opt into `protocol.JSONRPCError`:

```go
type effectsUnrecorded struct{ refs []string }

func (e effectsUnrecorded) Error() string { return "effects unrecorded: commit failed" }
func (e effectsUnrecorded) JSONRPCError() (int, string) {
    return -32002, fmt.Sprintf("effects unrecorded; commit failed; refs=%v", e.refs)
}
```

Return this error from `Invoke` when an effect ran but its commit failed. The
MCP handler uses `errors.As`, so wrapping with `fmt.Errorf("invoke: %w", err)`
works too. The code must be nonzero and the message nonempty. Both pass through
verbatim, including percent signs; use server-error codes `-32000..-32099` or
application-defined codes. Reserved codes pass through at the host's risk.

For request id `41` and refs `er-1`, `er-2`, the response is HTTP 200 with
`Content-Type: text/event-stream`:

```text
event: message
data: {"jsonrpc":"2.0","id":41,"error":{"code":-32002,"message":"effects unrecorded; commit failed; refs=[er-1 er-2]"}}

```

This error answers only its calling message. In a supported batch
(`2025-03-26`), sibling results survive in the same SSE response array with
their own ids. Errors without the hook, zero codes and empty messages retain
the frozen whole-POST JSON envelope: `-32603 "host callback failed"`.
Timeout, cancellation and capacity mappings remain unchanged. This hook applies
only to MCP `Invoke` errors; session resolution, tool discovery and A2A keep
their existing behavior. No `isError`, `error.data` or `_meta` channel is added.

### Ownership boundary

The host owns authentication, session identity, authorization policy, callback
implementation, middleware, listener creation, TLS, HTTP server lifecycle, and
shutdown. `ResolveSession` runs for every discovery and invocation request;
`Tools` returns that exact session's authorized descriptors; and `Invoke`
receives the same opaque session value plus raw JSON arguments.

AILANG owns descriptor validation and copying, deterministic ordering, MCP/A2A
wire projection, request-scoped dispatch lookup, callback timeouts and concurrency
capacity, and stable protocol error envelopes. `Mount` never opens a socket,
starts a watcher, installs signal handlers, or takes ownership of the caller's
server. Duplicate mux patterns follow the standard `http.ServeMux` panic behavior.

Embedded surfaces have no ambient `submit_feedback` tool. Hosts opt in by
returning a descriptor with that name and handling it through their `Invoker`.
The standalone `ailang serve-api` defaults remain unchanged.

## Relationship to Go Interop

`serve-api` builds on the [Go Interop embed API](./go-interop.md):

| Feature | Go Interop | serve-api |
|---------|-----------|-----------|
| Setup effort | Write Go code | Zero (CLI command) |
| Customization | Full control | Convention-based |
| Performance | Best | Good (HTTP overhead) |
| Error handling | Custom Go logic | Result type → HTTP status codes |
| Effects | Can provide handlers | Pure functions only |
| Use case | Production apps | Dev tools, prototyping, demos |

For production applications requiring custom error handling, effect handlers, or Go-level integration, use the [Go Interop embed API](./go-interop.md) directly.

---

## Working Example

A complete working example with automated tests is available at:

```
examples/web_api_demo/
├── api/
│   ├── math.ail      # add, multiply, factorial, fibonacci
│   └── greet.ail     # hello, farewell, welcome (with JSON)
├── test.sh           # Automated test (17 checks, all passing)
└── README.md
```

Run the automated tests:

```bash
./examples/web_api_demo/test.sh
```

This starts the server, exercises all endpoints (function calls, introspection, error handling, CORS), and reports pass/fail.

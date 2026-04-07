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
  "Hello, " ++ name ++ "!"

export pure func farewell(name: string) -> string =
  "Goodbye, " ++ name ++ ". Until next time!"

export pure func welcome(name: string) -> string =
  encode(jo([
    kv("message", js("Welcome, " ++ name ++ "!")),
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
- **Filtering** — `--routes-only` and `@noexpose` are respected in MCP `tools/list`, consistent with HTTP and OpenAPI. With `--routes-only`, undocumented non-route helpers are also auto-excluded from MCP.
- **Backward compatible** — Tool handlers accept both named parameters (`{"filepath": "doc.pdf"}`) and legacy positional format (`{"args": ["doc.pdf"]}`).

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

## CLI Reference

### `ailang serve-api`

```
Usage: ailang serve-api [flags] <path...>

Serve AILANG module exports as REST API endpoints.

Flags:
  --port PORT          HTTP port (default: 8080)
  --cors               Enable CORS for all origins (default: true)
  --frontend PATH      Proxy to Vite dev server at PATH
  --static PATH        Serve static files from PATH
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
  "Hello, " ++ name ++ "!"

-- AI effect: requires --caps AI plus --ai MODEL
export func summarize(text: string) -> string ! {AI} =
  ai_call("Summarize this: " ++ text)

-- Multiple effects: requires --caps IO,Net
export func fetchAndLog(url: string) -> string ! {IO, Net} {
  let body = http_get(url);
  println("Fetched: " ++ url);
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
| `@mcp_name("name")` | Override the auto-generated MCP tool name for this function |
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
  else "authenticated: " ++ content
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

> **Note:** The `_headers` convention is consistent with the existing `_body`/`_status`/`_headers` pattern used for [binary responses](#binary-response). For simple JSON responses that just need extra headers, `@nowrap` with `_headers` is more ergonomic than the full `_body` pattern.

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

---

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

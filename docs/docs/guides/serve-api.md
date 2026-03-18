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
| GET | `/.well-known/agent.json` | A2A Agent Card |
| POST | `/a2a/` | A2A JSON-RPC endpoint |

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

**No arguments (for nullary functions):**

```bash
curl -X POST http://localhost:8080/api/api/handlers/getStatus
```

### JSON Response Format

All function calls return:

```json
{
  "result": "Hello, World!",
  "module": "api/greet",
  "func": "hello",
  "elapsed_ms": 2
}
```

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

Each exported AILANG function becomes an MCP tool with proper JSON Schema for arguments. Module metadata is available as an MCP resource at `ailang://meta/modules`.

### A2A (Agent-to-Agent Protocol)

Google's A2A protocol is always enabled:

- **Agent Card**: `GET /.well-known/agent.json` — lists all functions as skills
- **Task endpoint**: `POST /a2a/` — JSON-RPC 2.0 for function invocation

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

You can type the API response:

```typescript
interface ApiResponse {
  result: unknown
  module: string
  func: string
  elapsed_ms: number
  error?: string
}
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

Multiple annotations can be combined:

```ailang
@route("POST", "/api/v1/compute")
@verify(depth: 3)
export func compute(x: int) -> int ! {}
  x * x
```

Supported HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS.

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

## Relationship to Go Interop

`serve-api` builds on the [Go Interop embed API](./go-interop.md):

| Feature | Go Interop | serve-api |
|---------|-----------|-----------|
| Setup effort | Write Go code | Zero (CLI command) |
| Customization | Full control | Convention-based |
| Performance | Best | Good (HTTP overhead) |
| Error handling | Custom Go logic | Generic JSON errors |
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

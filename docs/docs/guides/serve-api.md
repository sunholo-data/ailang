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
| GET | `/api/_health` | Health check |

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

## CLI Reference

### `ailang serve-api`

```
Usage: ailang serve-api [flags] <path...>

Serve AILANG module exports as REST API endpoints.

Flags:
  --port PORT       HTTP port (default: 8080)
  --cors            Enable CORS for all origins (default: true)
  --frontend PATH   Proxy to Vite dev server at PATH
  --static PATH     Serve static files from PATH

Arguments:
  <path...>         One or more .ail files or directories
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
2. Restart `ailang serve-api` (hot reload not yet supported)
3. New endpoints are automatically available

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

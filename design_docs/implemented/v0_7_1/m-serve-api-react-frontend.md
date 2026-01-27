# M-SERVE-API: Auto-Serve AILANG Exports as REST API

**Status:** Implemented
**Version:** v0.7.1
**Priority:** P2 (Medium - DX improvement for web integrations)
**Dependencies:** `internal/embed/` (existing), `internal/iface/` (existing)

**Created:** 2026-01-27
**Implemented:** 2026-01-27

---

## Problem Statement

AILANG has a Go embed API (`internal/embed/`) for calling AILANG functions from Go, but there's no turnkey way for users (especially AI agents) to expose AILANG functions as HTTP endpoints for web frontends. The existing Collaboration Hub dashboard (`internal/server/`) uses a hand-written bridge pattern (`ailang_bridge.go`) which is production-grade but requires significant manual setup per function.

Users need a zero-config way to:
1. Serve AILANG module exports as REST endpoints
2. Optionally pair with a React frontend
3. Get started in under a minute

---

## Solution

Two new CLI commands:

### `ailang serve-api`

Compiles AILANG modules and auto-generates REST endpoints from all exported functions.

```bash
# Serve a single module
ailang serve-api ecommerce/api/handlers.ail --port 8080

# Serve all .ail files in a directory
ailang serve-api ./api/ --port 8080

# With React frontend proxy (Vite dev server)
ailang serve-api ./api/ --port 8080 --frontend ./ui
```

### `ailang init web-app`

Scaffolds a full-stack project with AILANG API backend + React frontend.

```bash
ailang init web-app myproject
cd myproject
cd ui && npm install && cd ..
make dev
```

---

## Architecture

### URL Convention

Module paths map directly to URL paths:

```
Module: ecommerce/api/handlers
Export: successResponse(data: string) -> ApiResponse

Endpoint: POST /api/ecommerce/api/handlers/successResponse
Body:     {"args": ["hello"]}
Response: {"result": {...}, "module": "ecommerce/api/handlers", "func": "successResponse", "elapsed_ms": 5}
```

### Introspection Endpoints

- `GET /api/_meta/modules` - List all loaded modules with typed exports
- `GET /api/_meta/modules/{path}` - Detailed module info
- `GET /api/_health` - Health check

### Key Design Decisions

1. **All exports are POST endpoints** - Simpler than inferring GET/POST from type signatures. POST with JSON body works universally for all function arities.

2. **Module path becomes URL path** - The `/api/` prefix avoids collision with frontend routes. The module path after `/api/` matches the AILANG `module` declaration exactly.

3. **JSON in, JSON out** - Uses existing `embed.Engine.Call()` with `embed.FromGo()` / `embed.ToGo()` for value conversion. No custom serialization.

4. **Reuses `embed.Engine`** - No new compilation infrastructure. Wraps the existing embed API with HTTP handlers.

5. **CORS enabled by default** - This is primarily a dev tool. Production deployments would use a reverse proxy.

6. **Lazy compilation** - Modules are compiled on first HTTP request via `engine.Call()`, not at server startup. Interface info is extracted eagerly for metadata endpoints.

---

## Implementation

### New Package: `internal/apiserver/` (~535 LOC)

| File | LOC | Purpose |
|------|-----|---------|
| `server.go` | ~280 | Core server: module loading, route building, Vite proxy, static serving |
| `handler.go` | ~170 | Generic function call handler with arg parsing |
| `meta.go` | ~85 | Metadata and health endpoints |
| `server_test.go` | ~380 | 18 tests covering all endpoints |
| `templates/templates.go` | ~10 | Go embed directive for scaffold templates |
| `templates/web_app/` | ~200 | React+Vite scaffold files |

### CLI Commands

| File | LOC | Purpose |
|------|-----|---------|
| `cmd/ailang/serve_api.go` | ~95 | `serve-api` command with flags |
| `cmd/ailang/init_webapp.go` | ~100 | `init` command with embedded filesystem copy |
| `cmd/ailang/main.go` | +10 | Command routing additions |

### Bug Fix: `internal/embed/convert.go`

JSON's `encoding/json` unmarshals all numbers as `float64`. When calling AILANG functions typed as `int`, this caused type mismatches. Added whole-number detection:

```go
case reflect.Float64:
    f := rv.Float()
    if f == float64(int(f)) && f >= -1e15 && f <= 1e15 {
        return &eval.IntValue{Value: int(f)}, nil
    }
    return &eval.FloatValue{Value: f}, nil
```

---

## Test Coverage

18 tests in `internal/apiserver/server_test.go`:

- **TestHealthEndpoint** - Status, module count, export count
- **TestListModules** - Module listing with export names
- **TestFunctionCall** - 3 subtests: args array, add with ints, single value body
- **TestFunctionCallErrors** - 3 subtests: unknown module (404), unknown function (404), GET not allowed (405)
- **TestCORSHeaders** - 2 subtests: OPTIONS preflight (204), regular request CORS headers
- **TestModuleDetail** - 2 subtests: existing module, nonexistent module (404)
- **TestParseArgs** - 6 subtests: empty body, args array, single string, single number, single object, invalid JSON
- **TestCountFunctionArity** - 4 subtests: unary, binary, non-function, tuple args

---

## Scaffold Template

`ailang init web-app myproject` creates:

```
myproject/
├── api/
│   └── handlers.ail        # Example module with hello() and add()
├── ui/
│   ├── package.json         # React 18, Vite 5, TypeScript
│   ├── vite.config.ts       # Proxy /api → localhost:8080
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.tsx         # React entry point
│       └── App.tsx          # Demo UI calling AILANG API
├── Makefile                 # `make dev` runs both servers
└── README.md                # Getting started guide
```

Templates are embedded in the binary via `go:embed` in `internal/apiserver/templates/templates.go`.

---

## Existing Code Leverage

- `embed.Engine.Call()` - Core function calling (`internal/embed/embed.go`)
- `embed.FromGo()` / `embed.ToGo()` - Value conversion (`internal/embed/convert.go`)
- `iface.Iface.Exports` - Typed export introspection (`internal/iface/iface.go`)
- `pipeline.RunWithContext()` - Module compilation (`internal/pipeline/pipeline.go`)

---

## Future Enhancements

- **Hot reload on .ail changes** - Watch source files, recompile on change
- **GET support for pure functions** - Query string args for pure functions with simple types
- **OpenAPI spec generation** - Auto-generate from module type signatures
- **TypeScript client generation** - Generate typed API client from exports

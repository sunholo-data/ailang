# M-PROTOCOL-SUPPORT: OpenAPI, MCP, and A2A for `ailang serve-api`

**Status**: Implemented
**Target**: v0.9.0
**Priority**: P1 (High)
**Estimated**: 12-15 days (~60-75 hours) across 3 phases
**Dependencies**: None (builds on existing `internal/apiserver/`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | Protocol layer only, no new state |
| A3: Effect Legibility | +1 | Type signatures exposed as JSON Schema make effects visible |
| A4: Explicit Authority | 0 | No capability changes; same `--caps` flag controls effects |
| A5: Bounded Verification | +1 | OpenAPI spec enables automated API contract testing |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | MCP makes AILANG directly usable by AI models; A2A enables agent interop |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | OpenAPI documents endpoints; MCP tool schemas show what operations cost |
| A10: Composability | +2 | MCP tools compose with any MCP client; A2A composes with any A2A agent |
| A11: Structured Failure | +1 | All three protocols define structured error formats |
| A12: System Boundary | +1 | Clean protocol boundaries between AILANG and external consumers |

**Net Score: +9** -- Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism; protocol adapters are purely structural
- [x] A3 (Effects): Effects still gated by `--caps` flag; protocols don't bypass capability system
- [x] A4 (Authority): No ambient access granted; MCP/A2A use same `embed.Engine.Call()` path
- [x] A7 (Machines First): Core motivation -- make AILANG functions callable by AI models

## Problem Statement

`ailang serve-api` auto-generates REST endpoints from AILANG module exports. It already has rich function metadata (name, type signature, arity, purity) available via `/api/_meta/modules`. However:

1. **No OpenAPI spec** -- Clients can't auto-generate SDKs or validate requests. The API is undiscoverable without reading source code.
2. **No MCP support** -- AI models (Claude Desktop, Cursor, VS Code Copilot) can't use AILANG functions as tools. Users must manually bridge between AI and AILANG.
3. **No A2A support** -- External agent systems can't discover or invoke AILANG functions through a standard protocol. Inter-agent communication requires custom integration.

**Current State:**
- `POST /api/{module}/{func}` -- Function call endpoint (handler.go)
- `GET /api/_meta/modules` -- Module introspection (meta.go)
- `GET /api/_meta/modules/{path}` -- Module detail (meta.go)
- `GET /api/_health` -- Health check (meta.go)
- `ExportInfo` struct has: `Name`, `Type` (string), `Pure` (bool), `Arity` (int)
- `embed.Engine` provides: `Call()`, `CallJSON()`, `Eval()`, `ListExports()`
- No auth on apiserver (unlike Collaboration Hub which has Firebase JWT)

**Impact:**
- AILANG functions are invisible to the AI tool ecosystem
- No way to auto-generate client SDKs for AILANG APIs
- Agent-to-agent communication requires custom glue code
- AILANG's type system metadata goes unused beyond display strings

## Goals

**Primary Goal:** Make AILANG functions callable by AI models and discoverable via standard protocols, with zero changes to existing AILANG code.

**Success Metrics:**
- OpenAPI spec generated from loaded modules, passes validation
- MCP server exposes all loaded functions as tools with correct JSON Schema
- A2A Agent Card advertises AILANG capabilities to external agents
- All protocols auto-update when modules are hot-reloaded
- Existing `POST /api/` endpoint behavior unchanged
- `make test` passes, `make lint` clean

## Solution Design

### Shared Foundation: Type Signature to JSON Schema Converter

All three protocols need to convert AILANG type signatures to JSON Schema. This is the core shared component.

**Package:** `internal/apiserver/schema/` (~200 LOC)

**Type Mapping:**

| AILANG Type | JSON Schema | Notes |
|-------------|-------------|-------|
| `int` | `{"type": "integer"}` | |
| `float` | `{"type": "number"}` | |
| `string` | `{"type": "string"}` | |
| `bool` | `{"type": "boolean"}` | |
| `unit` | `{"type": "null"}` | Nullary functions |
| `bytes` | `{"type": "string", "contentEncoding": "base64"}` | |
| `[T]` | `{"type": "array", "items": <T>}` | Recursive |
| `(T1, T2)` | `{"type": "array", "items": [<T1>, <T2>], "minItems": 2, "maxItems": 2}` | Tuple |
| `{f1: T1, f2: T2}` | `{"type": "object", "properties": {"f1": <T1>, "f2": <T2>}}` | Record |
| `a` (type variable) | `{}` | Any/generic |
| `Result[T, E]` | `{"oneOf": [{"type":"object","properties":{"Ok":<T>}}, ...]}` | ADT (best-effort) |

**Function decomposition:**

Given `int -> string -> bool`:
- **Parameters**: `[{"type":"integer"}, {"type":"string"}]`
- **Return type**: `{"type":"boolean"}`

The converter parses the type string from `ExportInfo.Type` and produces JSON Schema objects. This is a string-based approach (not walking AST) since the apiserver only has the rendered type string.

**Future improvement:** If richer type info is needed (e.g., record field names, ADT variants), expose `*types.Scheme` or a structured JSON type representation from the interface. For now, the type string is sufficient for primitives, lists, and simple records.

### Phase 1: OpenAPI 3.1 Spec Generation

**Estimated:** 3-4 days (~16-20 hours)

Generate an OpenAPI 3.1 spec dynamically from loaded module metadata and serve it alongside the existing API.

#### New Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/_meta/openapi.json` | GET | OpenAPI 3.1 spec (JSON) |
| `/api/_meta/openapi.yaml` | GET | OpenAPI 3.1 spec (YAML) |
| `/api/_meta/docs/` | GET | Swagger UI (optional, via embedded static) |

#### Spec Structure

```yaml
openapi: "3.1.0"
info:
  title: "AILANG API - {loaded modules}"
  version: "0.8.1"  # from ailang version
  description: "Auto-generated API from AILANG module exports"

paths:
  /api/{module}/{func}:
    post:
      operationId: "{module}.{func}"
      summary: "{func} ({type})"
      tags: ["{module}"]
      x-ailang-pure: true/false
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                args:
                  type: array
                  items: [<param1_schema>, <param2_schema>]
                  minItems: {arity}
                  maxItems: {arity}
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  result: <return_type_schema>
                  module: {type: string}
                  func: {type: string}
                  elapsed_ms: {type: integer}

  # Meta endpoints
  /api/_meta/modules:
    get:
      summary: "List loaded modules"
      ...
  /api/_health:
    get:
      summary: "Health check"
      ...
```

#### Implementation

**New files:**
- `internal/apiserver/schema/schema.go` (~200 LOC) -- Type string parser + JSON Schema generator
- `internal/apiserver/schema/schema_test.go` (~250 LOC) -- Parser tests for all type patterns
- `internal/apiserver/openapi.go` (~250 LOC) -- OpenAPI spec builder from `[]ModuleInfo`
- `internal/apiserver/openapi_test.go` (~150 LOC) -- Spec generation tests

**Modified files:**
- `internal/apiserver/server.go` -- Register OpenAPI endpoints in `buildRoutes()`; regenerate spec on hot reload
- `go.mod` -- Add `github.com/getkin/kin-openapi` (for spec validation, optional)

#### Hot Reload Integration

When a module is hot-reloaded (via `watcher.go`), the OpenAPI spec is regenerated. The spec is cached in `Server.openapiSpec` (protected by the existing `sync.RWMutex`) and rebuilt when modules change.

---

### Phase 2: MCP Server

**Estimated:** 5-6 days (~25-30 hours)

Expose each AILANG exported function as an MCP tool, enabling AI models to discover and call AILANG functions.

#### Architecture

```
┌─────────────────────────────────────────┐
│          MCP Client                     │
│  (Claude Desktop / Cursor / VS Code)   │
└────────────┬────────────────────────────┘
             │ stdio (local) or HTTP+SSE (remote)
             ▼
┌─────────────────────────────────────────┐
│        internal/apiserver/mcp.go        │
│  ┌──────────────────────────────────┐   │
│  │ Tools: one per exported function │   │
│  │  ailang.{module}.{func}          │   │
│  │  Schema from type signature      │   │
│  └──────────┬───────────────────────┘   │
│             │                           │
│  ┌──────────▼───────────────────────┐   │
│  │ embed.Engine.Call(mod, fn, args) │   │
│  └──────────────────────────────────┘   │
│                                         │
│  ┌──────────────────────────────────┐   │
│  │ Resources: module source, stdlib │   │
│  └──────────────────────────────────┘   │
│                                         │
│  ┌──────────────────────────────────┐   │
│  │ Prompts: syntax ref, agent guide │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

#### MCP Tools

Each exported function becomes an MCP tool:

```json
{
  "name": "ailang.api_math.add",
  "description": "add(int -> int -> int) [pure]",
  "inputSchema": {
    "type": "object",
    "properties": {
      "args": {
        "type": "array",
        "items": [{"type": "integer"}, {"type": "integer"}],
        "minItems": 2,
        "maxItems": 2
      }
    },
    "required": ["args"]
  }
}
```

Tool execution calls `embed.Engine.Call()` and returns the JSON-encoded result.

#### MCP Resources

| URI | Description |
|-----|-------------|
| `ailang://module/{path}` | Source code of loaded module |
| `ailang://stdlib/{name}` | Standard library module source |
| `ailang://meta/modules` | Module listing (same as `/api/_meta/modules`) |

#### MCP Prompts

| Name | Description | Source |
|------|-------------|--------|
| `write_ailang` | Full AILANG syntax reference for code generation | `ailang prompt` embedded FS |
| `agent_coding` | Minimal iterative coding guide | `ailang agent-prompt` embedded FS |

#### Transports

1. **Stdio** -- `ailang serve-api --mcp <path...>` runs as stdio MCP server (no HTTP, for local AI clients)
2. **HTTP+SSE** -- When running as HTTP server, mount MCP endpoint at `/mcp/` for remote clients

#### Implementation

**New files:**
- `internal/apiserver/mcp.go` (~350 LOC) -- MCP server: tool/resource/prompt registration, tool call handler
- `internal/apiserver/mcp_test.go` (~200 LOC) -- MCP tool listing, tool call, resource access tests

**Modified files:**
- `internal/apiserver/server.go` -- Add `--mcp` mode; mount `/mcp/` endpoint in HTTP mode
- `cmd/ailang/serve_api.go` -- Add `--mcp` flag for stdio transport mode
- `go.mod` -- Add `github.com/modelcontextprotocol/go-sdk`

#### Hot Reload Integration

When modules are hot-reloaded, the MCP tool list is updated. MCP clients that support `notifications/tools/list_changed` will be notified.

---

### Phase 3: A2A Protocol

**Estimated:** 4-5 days (~20-25 hours)

Expose AILANG functions as A2A skills, enabling external agent systems to discover and invoke them.

#### Agent Card

Served at `GET /.well-known/agent.json`:

```json
{
  "name": "AILANG Function Server",
  "description": "AILANG module exports as callable functions",
  "url": "http://localhost:8080",
  "version": "0.8.1",
  "capabilities": {
    "streaming": false,
    "pushNotifications": false,
    "stateTransitionHistory": false
  },
  "defaultInputModes": ["application/json"],
  "defaultOutputModes": ["application/json"],
  "skills": [
    {
      "id": "api_math.add",
      "name": "add",
      "description": "int -> int -> int [pure]",
      "tags": ["api/math", "pure"],
      "inputModes": ["application/json"],
      "outputModes": ["application/json"]
    }
  ]
}
```

Skills are generated dynamically from loaded modules.

#### A2A Task Endpoint

`POST /a2a/` handles JSON-RPC 2.0 requests:

- **`tasks/send`** -- Create a task that calls an AILANG function
  - Maps skill ID to module/function
  - Executes via `embed.Engine.Call()`
  - Returns result as A2A message with artifact
- **`tasks/get`** -- Get task status (immediate for pure functions)
- **`tasks/cancel`** -- Cancel (no-op for sync functions)

#### Implementation

**New files:**
- `internal/apiserver/a2a.go` (~300 LOC) -- Agent Card generation, JSON-RPC dispatch, task execution
- `internal/apiserver/a2a_test.go` (~200 LOC) -- Agent Card, task send/get tests

**Modified files:**
- `internal/apiserver/server.go` -- Register `/.well-known/agent.json` and `/a2a/` routes
- `go.mod` -- Add A2A types (may be hand-defined if SDK is immature; protocol is simple JSON-RPC)

#### Hot Reload Integration

Agent Card skill list regenerated when modules change.

---

## Files Summary

### New Files (All Phases)

| File | Phase | LOC | Purpose |
|------|-------|-----|---------|
| `internal/apiserver/schema/schema.go` | 1 | ~200 | Type string → JSON Schema converter |
| `internal/apiserver/schema/schema_test.go` | 1 | ~250 | Converter tests |
| `internal/apiserver/openapi.go` | 1 | ~250 | OpenAPI spec builder |
| `internal/apiserver/openapi_test.go` | 1 | ~150 | Spec tests |
| `internal/apiserver/mcp.go` | 2 | ~350 | MCP server (tools, resources, prompts) |
| `internal/apiserver/mcp_test.go` | 2 | ~200 | MCP tests |
| `internal/apiserver/a2a.go` | 3 | ~300 | A2A Agent Card + task endpoint |
| `internal/apiserver/a2a_test.go` | 3 | ~200 | A2A tests |

**Total new code:** ~1,900 LOC

### Modified Files

| File | Changes |
|------|---------|
| `internal/apiserver/server.go` | Register new endpoints; add `--mcp` mode; regenerate specs on reload |
| `cmd/ailang/serve_api.go` | Add `--mcp` flag; update help text |
| `go.mod` | Add `kin-openapi`, `go-sdk` (MCP), A2A types |

## Examples

### Example 1: OpenAPI Discovery

```bash
# Start server with math module
ailang serve-api examples/api_math.ail

# Get OpenAPI spec
curl http://localhost:8080/api/_meta/openapi.json | jq '.paths | keys'
# ["/api/api_math/add", "/api/api_math/multiply", "/api/_meta/modules", "/api/_health"]

# Use with any OpenAPI client generator
openapi-generator generate -i http://localhost:8080/api/_meta/openapi.json -g python
```

### Example 2: MCP with Claude Desktop

```jsonc
// ~/.claude/claude_desktop_config.json
{
  "mcpServers": {
    "ailang-math": {
      "command": "ailang",
      "args": ["serve-api", "--mcp", "examples/api_math.ail"]
    }
  }
}
```

Claude Desktop can then call AILANG functions:
> "Add 42 and 17 using the AILANG math module" → calls `ailang.api_math.add` tool

### Example 3: A2A Agent Discovery

```bash
# External agent discovers AILANG
curl http://localhost:8080/.well-known/agent.json | jq '.skills[].name'
# ["add", "multiply", "factorial"]

# External agent sends task
curl -X POST http://localhost:8080/a2a/ -d '{
  "jsonrpc": "2.0",
  "method": "tasks/send",
  "params": {
    "id": "task-1",
    "message": {
      "role": "user",
      "parts": [{"type": "text", "text": "Calculate 5 factorial"}]
    },
    "metadata": {"skill_id": "api_math.factorial"}
  }
}'
```

## Testing Strategy

### Unit Tests
- Type string parser: all AILANG types including generics, records, ADTs
- OpenAPI spec: validates against OpenAPI 3.1 schema
- MCP tool listing: matches loaded modules
- MCP tool call: correct argument passing and result conversion
- A2A Agent Card: skill list matches modules
- A2A task send: function execution and result format

### Integration Tests
- Start apiserver with test module → fetch OpenAPI spec → validate
- Start MCP stdio → list tools → call tool → verify result
- Start apiserver → fetch Agent Card → send A2A task → verify response
- Hot reload: change module → verify specs updated

### Manual Testing
- Configure Claude Desktop with MCP server → call AILANG functions
- Open Swagger UI → browse and try endpoints
- Use A2A client → discover and invoke functions

## Non-Goals

- **Collaboration Hub protocols** -- This doc covers `ailang serve-api` only, not the observatory/coordinator API
- **Authentication** -- The apiserver has no auth; protocols inherit this (local-only is fine for MCP stdio)
- **Streaming responses** -- AILANG functions are synchronous; streaming is out of scope
- **WebSocket transport for MCP** -- Stdio and HTTP+SSE are sufficient
- **A2A push notifications** -- Requires webhook infrastructure; defer
- **Custom MCP tools beyond function calls** -- e.g., "compile", "type-check" are Collaboration Hub scope

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| MCP Go SDK is young/evolving | Med | Pin version; wrap in internal interface |
| A2A protocol still evolving | Med | Implement core subset only; A2A types are simple JSON |
| Type string parsing is fragile | Med | Extensive test suite; fall back to `{}` (any) for unparseable types |
| Performance of spec generation | Low | Cache spec; only rebuild on module change |
| OpenAPI spec too large for many modules | Low | Lazy generation; serve per-module specs if needed |

## Timeline

**Phase 1: OpenAPI** (Days 1-4)
- Day 1: Type string → JSON Schema converter + tests
- Day 2: OpenAPI spec builder
- Day 3: Endpoints, Swagger UI, hot reload integration
- Day 4: Tests, edge cases, validation

**Phase 2: MCP** (Days 5-10)
- Day 5: MCP SDK integration, tool registration
- Day 6-7: Tool call handler, resources, prompts
- Day 8: Stdio transport + `--mcp` flag
- Day 9: HTTP+SSE transport on existing server
- Day 10: Tests, Claude Desktop manual testing

**Phase 3: A2A** (Days 11-15)
- Day 11: Agent Card generation from modules
- Day 12-13: JSON-RPC task endpoint
- Day 14: Hot reload integration
- Day 15: Tests, documentation

**Total: ~12-15 days**

## Future Work

- **Auth for remote MCP/A2A** -- Add API key or OAuth when exposing publicly
- **Streaming** -- If AILANG adds async/streaming effects, propagate through MCP/A2A
- **Collaboration Hub protocols** -- Extend MCP/A2A to observatory/coordinator API
- **MCP tool composition** -- Chain multiple AILANG functions in a single MCP tool
- **A2A multi-agent** -- Enable AILANG agents to discover and call external A2A agents
- **Richer type schemas** -- Expose structured type info (not just string) for better JSON Schema

---

**Document created**: 2026-02-20
**Last updated**: 2026-02-20

# M-CODEGEN-API-SERVER: Compiled Go API Server from AILANG Modules

**Status**: Planned
**Target**: v0.10.0
**Priority**: P2 (Enhancement — `ailang serve-api` + Go binary hybrid works today)
**Estimated**: 2-3 weeks (phased, ~60-80 hours)
**Dependencies**: M-CODEGEN-MULTIMODULE-BUGS (v0.9.2, complete), multi-module `--emit-go`
**Reporter**: docparse deployment discussion
**Created**: 2026-03-17

---

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Static codegen is fully deterministic — same input → same Go output |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | +1 | Effect handler interfaces become explicit Go interfaces in generated server |
| A4: Explicit Authority | +1 | Capabilities declared in .ail become typed handler parameters in Go |
| A5: Bounded Verification | +1 | Generated Go code is statically typed — `go vet`/`go build` verify at compile time |
| A6: Safe Concurrency | 0 | HTTP concurrency is Go stdlib (net/http), no new concurrency model |
| A7: Machines First | +1 | OpenAPI/MCP/A2A specs generated at compile time — machine-readable artifacts |
| A8: Minimal Syntax | 0 | No new AILANG syntax — all via CLI flags |
| A9: Cost Visibility | +1 | Go binary eliminates interpreter overhead — cost is visible and minimal |
| A10: Composability | +1 | Generated server composes with any Go HTTP middleware/framework |
| A11: Structured Failure | 0 | Error handling follows existing patterns |
| A12: System Boundary | +1 | HTTP boundary is explicit — each exported function becomes a typed endpoint |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Codegen is fully deterministic
- [x] A3 (Effects): Effect handlers are explicit Go interfaces
- [x] A4 (Authority): Capabilities are typed parameters
- [x] A7 (Machines First): OpenAPI/MCP are machine-readable specs

---

## Problem Statement

Today, deploying AILANG modules as APIs requires choosing between two imperfect options:

**Option A: `ailang serve-api`** — Interprets .ail files at runtime. Auto-generates OpenAPI, Swagger UI, MCP tools, A2A Agent Card. Great DX but ~2-3s overhead per request (interpreter startup + evaluation).

**Option B: `ailang compile --emit-go` + manual HTTP harness** — Compiles to native Go binary. Sub-ms startup, <50ms per request. But loses all auto-generated API metadata (OpenAPI, MCP, Swagger, A2A).

**The gap:** No way to get both native Go performance AND auto-generated API metadata in a single binary.

**Current State:**
- `ailang serve-api` serves 8 endpoint types (function calls, OpenAPI, Swagger, MCP, A2A, health, modules, static)
- Type-to-JSON-Schema conversion exists in `internal/apiserver/schema/schema.go`
- MCP tool registration exists in `internal/apiserver/mcp.go`
- A2A Agent Card generation exists in `internal/apiserver/a2a.go`
- Go codegen produces typed function signatures via CoreTypeInfo

**Impact:**
- DocParse (19 modules, 406 functions) needs prod deployment with <50ms latency
- Any AILANG project wanting to expose functions as APIs faces this trade-off
- MCP integration (Claude Desktop, Cursor) requires either running the interpreter or manual wiring

---

## Goals

**Primary Goal:** `ailang compile --emit-go --with-api-server` generates a self-contained Go binary that serves AILANG functions as HTTP endpoints with embedded OpenAPI, MCP, and A2A support.

**Success Metrics:**
- Generated Go binary starts in <100ms
- Request latency <50ms (matching raw Go binary)
- OpenAPI spec matches `ailang serve-api` output for same module
- MCP tools work with Claude Desktop without interpreter running
- Single `go build` produces deployable binary (no runtime dependencies on AILANG)

---

## Solution Design

### Overview

The solution has two parts:

1. **Static metadata generation** (compile time) — Extract OpenAPI spec, MCP tool definitions, and A2A Agent Card from the compiled module interface, emit as Go embedded assets or generated Go code.

2. **Go server template** (code generation) — Emit a `main.go` with HTTP routes, JSON marshaling, and protocol handlers that import the generated function package.

### Architecture

The existing `internal/apiserver/` already does everything at runtime. The key insight is that **all the information it uses comes from `iface.Iface`**, which is available at compile time. We can extract the same metadata during compilation and emit it as static Go code.

```
                    COMPILE TIME                          RUNTIME
                    ──────────                            ───────
.ail files → pipeline → iface.Iface ─┬→ Go functions     Go binary
                                      ├→ openapi.json     ├── HTTP routes
                                      ├→ MCP tool defs    ├── OpenAPI endpoint
                                      ├→ A2A agent card   ├── MCP handler
                                      └→ main.go routes   └── A2A handler
```

**Components:**

1. **`internal/gen/golang/apiserver.go`** — New: generates `main.go` with HTTP server, route registration, and protocol handlers
2. **`internal/gen/golang/openapi_gen.go`** — New: generates static `openapi.json` from `iface.Iface` (reuses logic from `internal/apiserver/schema/`)
3. **`internal/gen/golang/mcp_gen.go`** — New: generates MCP tool registration code
4. **`cmd/ailang/compile.go`** — Modified: `--with-api-server` flag triggers server code generation

### What We Reuse vs Write New

| Component | Reuse from `internal/apiserver/` | New codegen needed |
|-----------|----------------------------------|-------------------|
| Type → JSON Schema | `schema.FromTypeString()` — reuse directly | Call at compile time, embed result |
| OpenAPI spec | `openapi.go:buildOpenAPISpec()` logic | Emit as `//go:embed openapi.json` |
| HTTP routes | `handler.go` pattern | Generate Go route registration code |
| MCP tools | `mcp.go:registerTools()` logic | Generate Go MCP tool registration |
| A2A Agent Card | `a2a.go:agentCardHandler()` logic | Emit as `//go:embed agent.json` |
| Swagger UI | `openapi.go` embedded HTML | Embed same HTML template |
| Function calling | `engine.CallPreserveFloats()` | Direct Go function calls (no engine) |

**Key difference:** The interpreter version calls functions through `embed.Engine.CallPreserveFloats()`. The compiled version calls typed Go functions directly — this is where the performance gain comes from.

### Generated File Structure

```
output/
├── pkg.go              # Generated functions (existing --emit-go)
├── types.go            # Generated ADT types (existing)
├── runtime.go          # Runtime helpers (existing)
├── dictionaries.go     # Type class dictionaries (existing)
├── openapi.json        # Static OpenAPI 3.1 spec (NEW)
├── agent.json          # Static A2A Agent Card (NEW)
├── server.go           # HTTP server + routes (NEW)
├── mcp_tools.go        # MCP tool registration (NEW)
├── main.go             # Entrypoint with flags (NEW)
└── go.mod              # Module file (NEW)
```

### Implementation Plan

**Phase 1: Static Metadata Generation** (~20h)
- [ ] Extract `iface.Iface` during `--emit-go` compilation (already available in pipeline Result)
- [ ] Generate `openapi.json` by calling existing `schema.FromTypeString()` at compile time
- [ ] Generate `agent.json` (A2A Agent Card) from module exports
- [ ] Emit both as files in output directory
- [ ] Test: compare generated openapi.json with `ailang serve-api` output for same module
- [ ] ~300 LOC new code in `internal/gen/golang/`

**Phase 2: Go HTTP Server Generation** (~25h)
- [ ] Generate `server.go` with route registration for each exported function
- [ ] Generate typed handler functions that call the compiled Go functions directly
- [ ] Generate JSON request parsing (extract args from `{"args": [...]}` format)
- [ ] Generate JSON response marshaling (same format as `ailang serve-api`)
- [ ] Embed `openapi.json` and serve at `/api/_meta/openapi.json`
- [ ] Serve Swagger UI HTML at `/api/_meta/docs`
- [ ] Generate health endpoint (`/api/_health`)
- [ ] Generate module introspection endpoint (`/api/_meta/modules`)
- [ ] Generate `main.go` with `--port`, `--cors` flags
- [ ] Generate `go.mod` with required dependencies
- [ ] ~500 LOC new code + ~200 LOC templates

**Phase 3: MCP Protocol Support** (~15h)
- [ ] Generate MCP tool definitions from exported function signatures
- [ ] Generate tool handler that dispatches to compiled functions
- [ ] Wire up stdio transport (for Claude Desktop/Cursor)
- [ ] Wire up HTTP transport at `/mcp/` endpoint
- [ ] Depends on `github.com/mark3labs/mcp-go` (same as current `ailang serve-api`)
- [ ] ~300 LOC new code

**Phase 4: Integration & Testing** (~10h)
- [ ] End-to-end test: compile DocParse → `go build` → run server → curl endpoints
- [ ] Parity test: compare OpenAPI output with `ailang serve-api` for same modules
- [ ] MCP test: connect Claude Desktop to compiled binary
- [ ] Performance benchmark: compiled server vs `ailang serve-api`
- [ ] Documentation and examples
- [ ] ~200 LOC tests

### Files to Modify/Create

**New files:**
- `internal/gen/golang/apiserver.go` — Server code generation (~400 LOC)
- `internal/gen/golang/openapi_gen.go` — Static OpenAPI generation (~200 LOC)
- `internal/gen/golang/mcp_gen.go` — MCP tool code generation (~200 LOC)
- `internal/gen/golang/apiserver_test.go` — Tests (~200 LOC)

**Modified files:**
- `cmd/ailang/compile.go` — `--with-api-server` flag, server generation pass (~50 LOC)
- `internal/apiserver/schema/schema.go` — May need to export some functions for reuse (~10 LOC)

**Total new code:** ~1,100 LOC implementation + ~200 LOC tests = ~1,300 LOC

---

## Examples

### Example 1: Compiling a DocParse API Server

**Command:**
```bash
ailang compile --emit-go --with-api-server \
  --out ./docparse-server --package-name docparse \
  docparse/main.ail docparse/services/*.ail docparse/types/*.ail
```

**Output:**
```
✓ Compiled 19 modules (406 declarations)
✓ Generated openapi.json (42 endpoints)
✓ Generated agent.json (A2A Agent Card)
✓ Generated server.go (HTTP routes)
✓ Generated mcp_tools.go (42 MCP tools)
✓ Generated main.go (entrypoint)
✓ Generated go.mod

Build: cd docparse-server && go build -o docparse-api
Run:   ./docparse-api --port 8080
```

**Running:**
```bash
cd docparse-server && go build -o docparse-api
./docparse-api --port 8080

# API ready:
#   POST http://localhost:8080/api/docparse/parseEpub
#   GET  http://localhost:8080/api/_meta/openapi.json
#   GET  http://localhost:8080/api/_meta/docs  (Swagger UI)
#   POST http://localhost:8080/mcp/  (MCP HTTP)
```

### Example 2: Generated Route Handler

**AILANG source:**
```ailang
module docparse/services/epub

export pure func parseEpub(path: string) -> {title: string, chapters: [string]} =
  ...
```

**Generated Go (server.go):**
```go
func handleParseEpub(w http.ResponseWriter, r *http.Request) {
    var req struct { Args []json.RawMessage `json:"args"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "invalid request body")
        return
    }
    if len(req.Args) != 1 {
        writeError(w, 400, "expected 1 argument, got %d", len(req.Args))
        return
    }
    var arg0 string
    if err := json.Unmarshal(req.Args[0], &arg0); err != nil {
        writeError(w, 400, "arg 0: expected string")
        return
    }

    start := time.Now()
    result := ParseEpub(arg0)  // Direct call to compiled function
    elapsed := time.Since(start).Milliseconds()

    writeResponse(w, result, "docparse/services/epub", "parseEpub", elapsed)
}
```

### Example 3: Development vs Production Workflow

```bash
# DEVELOPMENT — hot reload, auto-OpenAPI, full DX
ailang serve-api --watch --port 3000 docparse/

# PRODUCTION — single Go binary, <50ms latency
ailang compile --emit-go --with-api-server --out ./build docparse/
cd build && go build -o docparse-api && docker build -t docparse .
```

---

## Effort Estimate

| Phase | Hours | LOC | Risk |
|-------|-------|-----|------|
| P1: Static metadata | 20h | ~300 | Low — reuses existing schema code |
| P2: HTTP server gen | 25h | ~700 | Medium — most new code, template complexity |
| P3: MCP support | 15h | ~300 | Low — wraps existing mcp-go SDK |
| P4: Integration | 10h | ~200 | Medium — end-to-end parity testing |
| **Total** | **~70h** | **~1,500** | **Medium overall** |

**Calendar time:** 2-3 weeks at ~4h/day, or 1 week intensive.

**Why this estimate is realistic:**
- Phase 1 is mostly calling existing functions at compile time instead of runtime
- Phase 2 is the bulk of new work — generating Go HTTP handlers is template-heavy but straightforward
- Phase 3 reuses the mcp-go SDK that `ailang serve-api` already depends on
- The hardest part is parity testing (Phase 4) — ensuring generated server matches interpreter server exactly

---

## Success Criteria

- [ ] `ailang compile --emit-go --with-api-server` produces buildable Go project
- [ ] `go build` produces single binary with no AILANG runtime dependency
- [ ] Generated OpenAPI spec matches `ailang serve-api` output (diff test)
- [ ] MCP tools work with Claude Desktop
- [ ] Request latency <50ms for compiled server
- [ ] Swagger UI accessible at `/api/_meta/docs`
- [ ] A2A Agent Card served at `/.well-known/agent.json`
- [ ] All existing `--emit-go` tests still pass
- [ ] DocParse 19-module project compiles and serves successfully

## Testing Strategy

**Unit tests:**
- OpenAPI generation: compare output with known-good spec for test modules
- Route generation: verify handler code is syntactically valid Go
- MCP tool registration: verify tool definitions match function signatures

**Integration tests:**
- Compile test module → `go build` → start server → HTTP requests → verify responses
- Parity test: run same requests against `ailang serve-api` and compiled server, diff responses

**Manual testing:**
- Claude Desktop MCP connection to compiled binary
- Swagger UI functionality
- Docker containerization

## Non-Goals

**Not in this feature:**
- **Hot reload in compiled binary** — Use `ailang serve-api --watch` for development
- **WebSocket support** — HTTP request/response only (matches current `ailang serve-api`)
- **Authentication/authorization** — Users add Go middleware (compiled server composes with any Go framework)
- **Database/state management** — Effect handlers are user-provided, not generated
- **gRPC/GraphQL** — HTTP+JSON only (OpenAPI, MCP, A2A)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Type-to-schema parity drift | Medium | Share schema code between interpreter and codegen (single source of truth) |
| mcp-go SDK version mismatch | Low | Pin version in generated go.mod |
| Complex ADT types in API signatures | Medium | Test with DocParse's actual types (Block ADT with 14 variants) |
| Generated code too large | Low | Template-based generation, not per-function code duplication |
| Effect handlers need runtime wiring | Medium | Generate handler interface + example implementation file |

## Related Documents

**Implemented:**
- [m-codegen-multimodule-bugs](../v0_9_2/m-codegen-multimodule-bugs.md) — Multi-module codegen fixes (prerequisite)
- `internal/apiserver/` — Current runtime API server (reference implementation)

**Planned:**
- [m-codegen-ir-strategy](m-codegen-ir-strategy.md) — Future IR refactoring (independent, not blocking)

## Future Work

- **`--with-default-handlers`** — Generate default effect handler implementations (FS: os.ReadFile, Env: os.Getenv, AI: HTTP to Claude/Gemini)
- **Incremental compilation** — Only regenerate server code for changed modules
- **gRPC codegen** — Generate .proto from type signatures + gRPC server
- **WASM API server** — Same pattern but targeting WASM for edge deployment

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17

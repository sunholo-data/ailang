# Sprint Plan: M-PROTOCOL-SUPPORT

## Summary

Add OpenAPI 3.1, MCP, and A2A protocol support to `ailang serve-api`, enabling AI models and external agents to discover and call AILANG functions through standard protocols.

**Duration:** 5 days (reordered: shared foundation → MCP → OpenAPI → A2A)
**Dependencies:** None (builds on existing `internal/apiserver/`)
**Risk Level:** Medium (new external dependencies, type string parsing)
**Design Doc:** `design_docs/planned/v0_9_0/m-protocol-support.md`

## Current Status Analysis

### Completed Recently
- v0.8.1: WASM streaming (~460 LOC), Process execution (~650 LOC), Semantic envelope (~1,450 LOC)
- Recent velocity: ~150-200 LOC/day implementation + tests

### Velocity
- Recent average: ~180 LOC/day (implementation + tests)
- Estimated capacity: ~900 LOC for 5-day sprint
- Sprint target: ~1,200 LOC (stretching but achievable — protocol code is boilerplate-heavy)

### Existing Infrastructure
- `internal/apiserver/` — 1,194 LOC total (server, handler, meta, watcher, tests)
- `internal/embed/` — Engine with `Call()`, `CallJSON()`, `Eval()`, `ListExports()`
- `ExportInfo` has: Name, Type (string), Pure (bool), Arity (int)
- Hot reload via fsnotify already in place

## Proposed Milestones

### Milestone 1: Type String → JSON Schema Converter
**Goal:** Parse AILANG type signatures into JSON Schema objects (shared by all protocols)
**Estimated:** ~200 LOC implementation + ~250 LOC tests = ~450 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/apiserver/schema/` package
- Implement type string tokenizer (split on ` -> `, handle `[T]`, `{f: T}`, `(T1, T2)`)
- Map primitive types: int→integer, float→number, string→string, bool→boolean, unit→null
- Handle composite types: lists, tuples, records
- Handle type variables (`a`) → `{}` (any)
- Decompose function types: extract parameter schemas + return schema
- Comprehensive tests for all AILANG type patterns

**Acceptance Criteria:**
- [ ] Parses all primitive types correctly
- [ ] Parses `int -> string -> bool` into 2 params + 1 return schema
- [ ] Parses `[int]` into array schema
- [ ] Parses `{name: string, age: int}` into object schema
- [ ] Falls back to `{}` for unparseable types (no panic)
- [ ] All tests passing, lint clean

### Milestone 2: MCP Server
**Goal:** Expose AILANG functions as MCP tools via stdio and HTTP transports
**Estimated:** ~300 LOC implementation + ~150 LOC tests = ~450 LOC
**Duration:** 2 days

**Tasks:**
- Day 1: Add `github.com/modelcontextprotocol/go-sdk` dependency
  - Create `internal/apiserver/mcp.go` with MCP server setup
  - Register each exported function as an MCP tool using `mcp.AddTool()`
  - Tool input schemas from Milestone 1 type converter
  - Tool handler calls `embed.Engine.Call()` and returns result
- Day 2: Transports and CLI integration
  - Add `--mcp` flag to `cmd/ailang/serve_api.go` for stdio mode
  - Mount `StreamableServerTransport` at `/mcp/` for HTTP mode
  - Add resource providers (module source, stdlib)
  - Add prompts (syntax reference)
  - Tests with `mcp.NewInMemoryTransports()`

**Acceptance Criteria:**
- [ ] `ailang serve-api --mcp examples/api_math.ail` starts stdio MCP server
- [ ] MCP tool list includes all exported functions
- [ ] Tool call executes function and returns correct result
- [ ] Resources list module source code
- [ ] HTTP mode serves MCP at `/mcp/`
- [ ] Tests pass with in-memory transport

### Milestone 3: OpenAPI Spec Generation
**Goal:** Generate OpenAPI 3.1 spec from loaded modules
**Estimated:** ~200 LOC implementation + ~100 LOC tests = ~300 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/apiserver/openapi.go` with spec builder
- Generate paths for each module/function endpoint
- Include meta endpoints (`/api/_meta/modules`, `/api/_health`)
- Serve at `GET /api/_meta/openapi.json`
- Regenerate on hot reload (invalidate cached spec)
- Tests validating spec structure

**Acceptance Criteria:**
- [ ] `GET /api/_meta/openapi.json` returns valid OpenAPI 3.1 JSON
- [ ] Each function has correct request/response schemas
- [ ] Spec updates after hot reload
- [ ] Meta endpoints documented in spec
- [ ] Tests pass, lint clean

### Milestone 4: A2A Agent Card + Task Endpoint
**Goal:** Expose functions as A2A skills with Agent Card discovery
**Estimated:** ~250 LOC implementation + ~100 LOC tests = ~350 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/apiserver/a2a.go`
- Generate Agent Card from loaded modules at `GET /.well-known/agent.json`
- JSON-RPC dispatch at `POST /a2a/` for `tasks/send`, `tasks/get`
- Map A2A task to `embed.Engine.Call()` via skill ID → module/function
- Regenerate Agent Card on hot reload
- Tests for Agent Card and task execution

**Acceptance Criteria:**
- [ ] `GET /.well-known/agent.json` returns valid Agent Card
- [ ] Skills list matches loaded module exports
- [ ] `POST /a2a/` with `tasks/send` executes function
- [ ] Agent Card updates after hot reload
- [ ] Tests pass, lint clean

## Success Metrics
- All tests passing: `make test`
- Lint clean: `make lint`
- Total new code: ~1,200 LOC (implementation) + ~600 LOC (tests)
- 3 new protocol endpoints working
- Existing apiserver behavior unchanged
- CHANGELOG.md updated

## Dependencies
- `github.com/modelcontextprotocol/go-sdk` (v1.0.0+, stable)
- `github.com/getkin/kin-openapi` (optional, for spec validation)

## Open Questions
- Should A2A use the official `a2a-go` SDK or hand-implement (protocol is simple JSON-RPC)?

## Notes
- MCP SDK v1.0.0 has stable API with compatibility guarantee
- MCP generic `AddTool` auto-generates JSON Schema from Go structs — but we need dynamic registration since functions are loaded at runtime, so we'll use the non-generic `AddTool` with manual schemas
- Priority order: Schema → MCP → OpenAPI → A2A (MCP gives most immediate value)

# Sprint Plan: M-MCP-QUALITY

## Summary
Fix 5 MCP server bugs and add @route header access so AILANG packages work as proper MCP tool providers for AI agents. All fixes build on existing patterns (isExposed, ParamNames, buildHttpRequestRecord).

**Duration:** 3 days (~16 hours)
**Dependencies:** None
**Risk Level:** Low — all changes are additive, existing HTTP/OpenAPI/A2A paths unchanged
**Design Doc:** [m-mcp-quality-and-route-headers.md](m-mcp-quality-and-route-headers.md)

## Current Status Analysis

### Completed Recently
- v0.10.3: Fix multipart file upload when field name doesn't match param name
- v0.10.2: serve-api type coercion, DX error messages, status codes
- v0.10.0: MCP initial implementation, A2A agent card, @raw/@route context

### Velocity
- Recent average: ~200-300 LOC/day (from codegen and stdlib sprints)
- Estimated capacity: ~500 LOC for this sprint (focused, well-scoped changes)

### Remaining from Design Doc
- Phase 1: MCP filtering + portable names (~80 LOC)
- Phase 2: Doc comments + named inputSchema (~120 LOC)
- Phase 3: Auto-exclude + _headers injection (~60 LOC)
- Phase 4: Tests + docs (~150 LOC)
- **Total: ~410 LOC implementation + tests**

## Proposed Milestones

### Milestone 1: M1_MCP_FILTERING — Wire isExposed() + Portable Names
**Goal:** MCP tools/list respects --routes-only and @noexpose; tool names are machine-portable
**Estimated:** ~80 LOC implementation + ~60 LOC tests = ~140 LOC
**Duration:** 0.5 days

**Tasks:**
1. Add `isExposed()` check in `registerTools()` at `mcp.go:46` (same pattern as handler.go:114, openapi.go:181, a2a.go:50)
2. Implement `portableModuleName()` — strip machine-specific prefix, keep package qualifier + function name
3. Write tests: --routes-only filters MCP list; @noexpose hides from MCP; package tool names are portable

**Key Files:**
- `internal/apiserver/mcp.go` — add isExposed check + portableModuleName call
- `internal/apiserver/mcp_test.go` — new test file

**Acceptance Criteria:**
- [ ] MCP tools/list with --routes-only returns only @route functions
- [ ] @noexpose functions hidden from MCP tools/list
- [ ] Package-loaded modules use `<package>.<function>` names (no machine paths)
- [ ] All existing tests passing
- [ ] Linting clean

### Milestone 2: M2_MCP_SCHEMA — Doc Comment Descriptions + Named InputSchema
**Goal:** MCP tools have human-readable descriptions and named parameter schemas
**Estimated:** ~120 LOC implementation + ~80 LOC tests = ~200 LOC
**Duration:** 1 day

**Tasks:**
1. Add `DocComment string` field to `ExportInfo` in `server.go:77`
2. Extract doc comments from source text in `extractParamNamesFromAST()` (comments are `--` lines immediately preceding func decl; lexer skips them so read from source or AST file Pos)
3. Use doc comment as MCP description, fall back to type signature
4. Implement `ailangTypeToJSONSchema()` mapping: string→"string", int/float→"number", bool→"boolean", Json/records→"object", lists→"array"
5. Rewrite `buildMCPInputSchema()` to use `ExportInfo.ParamNames` + `ParamTypes` for named properties
6. Update `makeToolHandler()` to accept named params (not just `args[]` array) — parse both formats for backward compat

**Key Files:**
- `internal/apiserver/server.go` — DocComment field + extraction
- `internal/apiserver/mcp.go` — descriptions + inputSchema rewrite
- `internal/apiserver/mcp_schema.go` — new: type mapping + portable name helpers
- `internal/apiserver/mcp_test.go` — schema + description tests

**Acceptance Criteria:**
- [ ] Functions with doc comments show comment text as MCP description
- [ ] Functions without doc comments fall back to type signature
- [ ] inputSchema uses named parameters matching AILANG param names
- [ ] JSON Schema types correctly mapped from AILANG types
- [ ] MCP tool calls work with both `{"args": [...]}` and `{"paramName": value}` formats
- [ ] All tests passing
- [ ] Linting clean

### Milestone 3: M3_HEADERS_AND_POLISH — @route Header Access + Auto-Exclude + Docs
**Goal:** @route handlers can access request headers; undocumented helpers auto-excluded from MCP
**Estimated:** ~60 LOC implementation + ~50 LOC tests + ~60 LOC docs = ~170 LOC
**Duration:** 1 day

**Tasks:**
1. In `callFunction()` (routes.go:340), after args are parsed, check if ParamNames contains `_headers` — if so, inject `stringMapToJObject(r.Header)` at that position
2. Add MCP-specific auto-exclude: when --routes-only, also hide functions with no doc comment AND no @route from MCP tools/list
3. Add test: @route handler with `_headers` param receives request headers
4. Add test: multipart upload + _headers works together
5. Add `examples/mcp_tools.ail` demonstrating doc comments, @route, @noexpose, _headers
6. Update CHANGELOG.md
7. Update serve-api docs

**Key Files:**
- `internal/apiserver/routes.go` — _headers injection in callFunction()
- `internal/apiserver/mcp.go` — auto-exclude logic
- `internal/apiserver/routes_test.go` — header injection tests
- `examples/mcp_tools.ail` — new example
- `changelogs/v0.9-current.md` — changelog entry

**Acceptance Criteria:**
- [ ] @route handler declaring `_headers: Json` parameter receives parsed request headers
- [ ] Multipart file upload + _headers access works together
- [ ] Undocumented non-route functions hidden from MCP when --routes-only
- [ ] Example file created and verified working
- [ ] CHANGELOG updated
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Doc comment extraction needs source text since lexer skips comments — may need to read from file or add comment capture to lexer. Mitigation: read source bytes at FuncDecl.Pos offset, scan backwards for `--` comment lines.

## Success Metrics
- Test coverage maintained or improved
- All 6 agent inbox bugs addressed
- `examples/mcp_tools.ail` passing
- Documentation updated (changelog, serve-api guide)
- All tests passing
- All linting passing

## Dependencies
- None — all changes are in `internal/apiserver/` with no cross-cutting concerns

## Open Questions
- None — design freeze decisions were approved in the design doc

## Notes
- The MCP go-sdk is already imported and working; this is quality improvement, not greenfield
- `makeToolHandler` needs backward compat: accept both `{"args":[...]}` and named params
- Doc comment extraction is the trickiest part — lexer currently discards comments entirely
- If lexer-level comment capture is too invasive, can use file source + FuncDecl.Pos to extract preceding comment lines

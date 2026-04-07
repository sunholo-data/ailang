---
paths:
  - "internal/apiserver/**"
  - "cmd/ailang/serve_api.go"
---

# API Server Rules

## Annotation Sync: Runtime + Parser

When adding or modifying annotations (e.g., `@route`, `@raw`, `@nowrap`, `@noexpose`, `@mcp_name`), you MUST update BOTH:

1. **Runtime** (`internal/apiserver/`) — annotation checking, filtering, behavior
2. **Parser** (`internal/parser/parser_decl.go`) — annotation whitelist in `parseAnnotation()` switch statement

The parser has a hardcoded switch on annotation names. If a new annotation isn't added there, the parser rejects it with "unknown attribute" even if the runtime handles it correctly.

### Checklist for new annotations

- [ ] `internal/parser/parser_decl.go` — add `case "name":` to `parseAnnotation()` switch
- [ ] Update error message in `default:` case to list the new annotation
- [ ] Update hint in `PAR_INVALID_ATTRIBUTE` report (line after `@` check)
- [ ] `internal/parser/route_attr_test.go` — add parser test for new annotation
- [ ] `internal/apiserver/` — implement runtime behavior
- [ ] `docs/docs/guides/serve-api.md` — document the annotation
- [ ] `prompts/devtools/v0.8.0.md` + compact variant — update annotation reference
- [ ] `cmd/ailang/prompts/devtools/v0.8.0.md` + compact variant — same (embedded copy)

## Endpoint Filtering

`isExposed()` in `routes.go` is the single filtering point. All endpoint enumeration (handler dispatch, OpenAPI spec, A2A agent card, MCP tools/list, startup banner) must use it. No additional filtering logic should be added in individual consumers.

## Request Headers in @route

`@route` handlers can declare a `_headers: Json` parameter to receive HTTP request headers without `@raw`. The injection happens in `callFunction()` after argument parsing. Only the exact name `_headers` triggers injection.

## MCP Tool Names

All MCP tool names must satisfy `^[a-zA-Z0-9_-]{1,64}$` (Claude Desktop strict regex). Generation lives in `internal/apiserver/mcp_schema.go::mcpToolName()` and is invoked from `mcp.go::registerTools()`. Resolution order:

1. `@mcp_name("name")` author override (validated; invalid names cause the tool to be skipped with an error log).
2. Bare function name when `funcNameCount[name] == 1` across the dedup'd candidate set.
3. `<lastModuleSegment>_<funcName>` sanitized fallback for collisions.
4. `truncateWithHash` enforces the 64-char limit by appending a 7-char SHA1 suffix.

`validateMCPName()` is the regex gatekeeper — every emitted name passes through it. When adding a new tool name source, route it through `mcpToolName` (do not handcraft names).

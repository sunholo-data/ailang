---
paths:
  - "internal/apiserver/**"
  - "serveapi/**"
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

One authorized-surface gateway decides membership for every endpoint enumeration
and dispatch path. Loaded exports use `loadedExportMember`; embedded callbacks use
`protocol.CallerSurface` and the resulting `protocol.AuthorizedSurface`, defined
in `serveapi/protocol/descriptor.go`. `loadedExportMember` remains the
loaded-export gateway in `internal/apiserver`; `CallerSurface` remains the
embedded-callback membership path. Discovery and invocation must consume the
same authorized surface rather than reconstructing authority.

`@nomcp` is the single sanctioned protocol-scoped narrowing: it is applied by the
MCP projection only, after membership has been decided. It must remain visible to
HTTP dispatch, OpenAPI, and A2A. Protocol consumers must not add any other
authority filtering or fold `@nomcp` into the shared membership gateway.

## Request Headers in @route

`@route` handlers can declare a `_headers: Json` parameter to receive HTTP request headers without `@raw`. The injection happens in `callFunction()` after argument parsing. Only the exact name `_headers` triggers injection.

## MCP Tool Names

All MCP tool names must satisfy `^[a-zA-Z0-9_-]{1,64}$` (Claude Desktop strict regex). Generation lives in `internal/apiserver/mcp_schema.go::mcpToolName()` and is invoked from `mcp.go::registerTools()`. Resolution order:

1. `@mcp_name("name")` author override (validated; invalid names cause the tool to be skipped with an error log).
2. Bare function name when `funcNameCount[name] == 1` across the dedup'd candidate set.
3. `<lastModuleSegment>_<funcName>` sanitized fallback for collisions.
4. `truncateWithHash` enforces the 64-char limit by appending a 7-char SHA1 suffix.

`protocol.ValidateMCPName()` in `serveapi/protocol/descriptor.go` is the regex
gatekeeper; `internal/apiserver/protocol_compat.go::validateMCPName()` is only a
thin forwarding shim. Every emitted name passes through that validation. When
adding a new tool name source, route generation through `mcpToolName` (do not
handcraft names).

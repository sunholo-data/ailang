---
paths:
  - "internal/apiserver/**"
  - "cmd/ailang/serve_api.go"
---

# API Server Rules

## Annotation Sync: Runtime + Parser

When adding or modifying annotations (e.g., `@route`, `@raw`, `@nowrap`, `@noexpose`), you MUST update BOTH:

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

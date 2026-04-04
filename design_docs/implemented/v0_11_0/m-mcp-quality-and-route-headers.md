# M-MCP-QUALITY: MCP Server Quality & @route Header Access

**Status**: IMPLEMENTED
**Target**: v0.11.0
**Priority**: P0 (High — MCP tools unusable by AI agents in current state)
**Estimated**: 3-4 days
**Dependencies**: None
**Milestone ID**: M-MCP-QUALITY
**Created**: 2026-04-04
**Source**: Agent inbox messages from `cli` (5 MCP bugs) and `docparse` (header access)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change — MCP tool listing is a read-only projection of compiled modules |
| A2: Replayability | 0 | No change — tool discovery is not part of execution traces |
| A3: Effect Legibility | +1 | Named params in inputSchema make IO contracts explicit to agents |
| A4: Explicit Authority | +1 | `@noexpose` / `--routes-only` properly enforced — no accidental exposure |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | Core improvement — AI agents can actually understand and use MCP tools |
| A8: Minimal Syntax | +1 | No new syntax; reuses existing `@noexpose`, doc comments, `@route` |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Proper MCP schemas enable tool composition by agent frameworks |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | +2 | MCP is a system boundary — proper schemas + header access are essential |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No violation — tool metadata is derived deterministically from compiled modules
- [x] A3 (Effects): No hidden side effects — metadata projection only
- [x] A4 (Authority): Improves authority — fixes leaked internal functions
- [x] A7 (Machines First): Core improvement for machine consumers

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"protocol projection" class of issues**. The serve-api layer projects AILANG modules into multiple protocols (HTTP, OpenAPI, A2A, MCP). Each projection needs the same filtering and metadata enrichment, but MCP was added without wiring into the existing `isExposed()` gateway and metadata extraction.

**Pattern across protocols:**

| Concern | HTTP | OpenAPI | A2A | MCP |
|---------|------|---------|-----|-----|
| `isExposed()` filtering | ✅ | ✅ | ✅ | ❌ Missing |
| Named parameters | ✅ | ✅ | ✅ | ❌ Generic `args[]` |
| Doc comments as descriptions | N/A | ✅ | ✅ | ❌ Raw type sigs |
| Portable tool names | N/A | ✅ | ✅ | ❌ Absolute paths |

The fix should align MCP with the existing patterns already working in HTTP/OpenAPI/A2A.

**Header access** is also systemic — M-DX-ROUTE-CTX (v0.10.0) added `@raw` for full request context. This design extends that by allowing `@route` handlers to opt into headers without switching to `@raw` (which loses multipart parsing).

---

## Problem Statement

### Problem 1: MCP Tools Unusable by AI Agents (5 bugs)

When AILANG packages are served as MCP tool providers, AI agents (Claude, GPT, Cursor) cannot effectively use the tools because:

1. **All 159 exports appear as tools** — `--routes-only` and `@noexpose` are ignored in MCP context (`mcp.go:42` calls `GetModules()` without `isExposed()` check)
2. **Tool names are non-portable** — package-loaded modules get machine-specific absolute paths: `Users.mark.dev.sunholo.ailang-parse.docparse.services.samples.sampleResolvePath` (`mcp.go:51-53`)
3. **Descriptions are unreadable** — `parseCsv(∀α46. (string, string) -> [α46] ! {...ε41}) [pure]` instead of doc comment text (`mcp.go:55-61`)
4. **Input schemas are generic** — every tool has `{args: [positional...]}` instead of named parameters with types (`mcp.go:160-182`)
5. **No auto-exclusion of helpers** — internal functions like `xmlEscape`, `docxNs`, `cellText` pollute tool lists

### Problem 2: @route Endpoints Cannot Access HTTP Headers

`@route` endpoints receive parsed multipart files but cannot read HTTP headers. `@raw` endpoints get headers but receive the raw unparsed body. For Unstructured API compatibility (`/general/v0/general`), handlers need **both** multipart file parsing AND header access (for `unstructured-api-key` auth).

**Current state** (`routes.go`):
- `@route` dispatch: `callFunction()` lines 318-340 — parsed args only, no headers
- `@raw` dispatch: `callFunction()` lines 278-289 — full HttpRequest record via `buildHttpRequestRecord()`
- Response `_headers` already exists (lines 451-462) but no request equivalent

---

## Goals

**Primary Goal:** Make any AILANG package work as a proper MCP tool provider that AI agents can discover, understand, and call correctly.

**Success Metrics:**
- MCP `tools/list` with `--routes-only` returns only `@route` functions (not all 159 exports)
- Tool names are portable across machines (no absolute paths)
- Tool descriptions use doc comments; type signatures only as fallback
- `inputSchema` uses named parameters with JSON Schema types mapped from AILANG types
- `@noexpose` functions are hidden from MCP tool list
- `@route` handlers can optionally receive request headers

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| MCP tool name format for packages | Affects all MCP consumers; changing later breaks agent tool references | human | design | high |
| AILANG→JSON Schema type mapping | Defines the contract between AILANG types and MCP inputSchema | agent | compile | med |
| Header injection mechanism for @route | API surface for all @route handlers | human | design | med |
| Auto-exclude heuristic (undocumented + no @route) | Affects which functions are visible by default | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] MCP tool name format: use `<package>.<function>` (strip machine-specific prefix, keep package qualifier)
- [ ] Header injection: add optional `_headers: Json` parameter to @route (matching existing response `_headers` pattern)
- [ ] Auto-exclude: functions without doc comments AND without `@route` should be hidden in MCP when `--routes-only` is set (no new annotation needed beyond existing `@noexpose`)

---

## Solution Design

### Overview

Six targeted fixes to `internal/apiserver/mcp.go` and `internal/apiserver/routes.go`, all building on existing patterns:

1. Wire `isExposed()` into MCP `registerTools()` (same as HTTP/OpenAPI/A2A)
2. Strip machine-specific path prefixes from MCP tool names
3. Use doc comments for descriptions, fall back to type signatures
4. Build named-parameter `inputSchema` from `ExportInfo.ParamNames` + `ParamTypes`
5. Auto-exclude undocumented non-route functions in MCP context
6. Inject request headers into `@route` handlers via optional `_headers` parameter

### Architecture

**Component 1: MCP Filtering** (`mcp.go:registerTools()`)

Add `isExposed()` check at line 46, same pattern as `handler.go:114`, `openapi.go:181`, `a2a.go:50`:

```go
for _, export := range modInfo.Exports {
    if export.Arity < 0 {
        continue
    }
    if !ms.server.isExposed(export) {  // NEW: align with HTTP/OpenAPI/A2A
        continue
    }
    // ... register tool
}
```

**Component 2: Portable Tool Names** (`mcp.go:51-53`)

Replace absolute path with package-relative name. Use the module's declared name (from `module` declaration) or the last path component:

```go
// Before: "Users.mark.dev.sunholo.ailang-parse.docparse.services.samples.sampleResolvePath"
// After:  "docparse.services.samples.sampleResolvePath"
toolName := portableModuleName(modPath) + "." + export.Name
```

Strip everything up to and including the package root. For directly-loaded modules, use the relative path from the serve-api working directory.

**Component 3: Doc Comment Descriptions** (`mcp.go:55-61`)

`ExportInfo` needs a `DocComment` field populated from the AST `FuncDecl.DocComment`. Use it as description, fall back to type signature:

```go
desc := export.DocComment
if desc == "" {
    desc = fmt.Sprintf("%s(%s)", export.Name, export.Type)
}
```

**Component 4: Named Parameter InputSchema** (`mcp.go:160-182`)

`ExportInfo` already has `ParamNames` and `ParamTypes`. Build proper JSON Schema:

```go
func buildMCPInputSchema(export ExportInfo) map[string]any {
    props := map[string]any{}
    required := []string{}
    for i, name := range export.ParamNames {
        paramType := "string" // default
        if i < len(export.ParamTypes) {
            paramType = ailangTypeToJSONSchema(export.ParamTypes[i])
        }
        props[name] = map[string]any{
            "type": paramType,
        }
        required = append(required, name)
    }
    return map[string]any{
        "type":       "object",
        "properties": props,
        "required":   required,
    }
}
```

Type mapping: `string`→`"string"`, `int`/`float`→`"number"`, `bool`→`"boolean"`, `Json`/records→`"object"`, lists/arrays→`"array"`, everything else→`"string"`.

**Component 5: Auto-Exclude Heuristic**

In MCP context only, add stricter filtering: when `--routes-only` is set, also exclude functions that have no doc comment AND no `@route` annotation. This catches helpers like `xmlEscape` without requiring explicit `@noexpose` on every internal function.

This is an MCP-specific policy on top of `isExposed()`, not a change to `isExposed()` itself.

**Component 6: Request Headers for @route** (`routes.go:318-340`)

In `callFunction()`, after argument parsing, check if the function declares a parameter named `_headers`. If so, inject parsed request headers as a JObject:

```go
// After parsing args (line 340)
if headerIdx := findParamIndex(foundExport.ParamNames, "_headers"); headerIdx >= 0 {
    headers := stringMapToJObject(r.Header)  // reuse existing helper from @raw
    args[headerIdx] = headers
}
```

This mirrors the existing response `_headers` extraction pattern (lines 451-462) but for the request side.

### Implementation Plan

**Phase 1: MCP Filtering & Names** (~4 hours)
- [ ] Add `isExposed()` check to `registerTools()` in `mcp.go`
- [ ] Implement `portableModuleName()` to strip machine-specific prefixes
- [ ] Add integration test: `--routes-only` filters MCP `tools/list`
- [ ] Add integration test: package-loaded module tool names are portable

**Phase 2: Descriptions & InputSchema** (~6 hours)
- [ ] Add `DocComment` field to `ExportInfo` struct in `server.go`
- [ ] Populate `DocComment` from AST during `extractModuleInfo()`
- [ ] Use doc comments as MCP tool descriptions in `registerTools()`
- [ ] Implement `ailangTypeToJSONSchema()` type mapping
- [ ] Rewrite `buildMCPInputSchema()` to use named parameters
- [ ] Add tests for type mapping and schema generation

**Phase 3: Auto-Exclude & Header Access** (~4 hours)
- [ ] Add MCP-specific auto-exclude for undocumented non-route functions
- [ ] Implement `_headers` parameter injection in `callFunction()`
- [ ] Add test: `@route` handler receives headers via `_headers` param
- [ ] Add test: multipart upload + header access works together
- [ ] Update docparse workaround to use `_headers` for Unstructured API compat

**Phase 4: Documentation & Examples** (~2 hours)
- [ ] Add `examples/mcp_tools.ail` showing doc comments + `@route` + `@noexpose`
- [ ] Update CHANGELOG.md
- [ ] Update serve-api docs with MCP improvements

### Files to Modify/Create

**Modified files:**
- `internal/apiserver/mcp.go` — Filtering, names, descriptions, schemas (~80 LOC changed)
- `internal/apiserver/server.go` — Add `DocComment` to `ExportInfo` (~10 LOC)
- `internal/apiserver/routes.go` — `_headers` injection in `callFunction()` (~20 LOC), doc comment extraction in `extractModuleInfo()` (~15 LOC)

**New files:**
- `internal/apiserver/mcp_test.go` — MCP-specific tests (~150 LOC)
- `internal/apiserver/mcp_schema.go` — `ailangTypeToJSONSchema()` + `portableModuleName()` (~80 LOC)

---

## Examples

### Example 1: MCP Tool Before/After

**Before (current):**
```json
{
  "name": "Users.mark.dev.sunholo.ailang-parse.docparse.services.samples.parseCsv",
  "description": "parseCsv(∀α46. (string, string) -> [α46] ! {...ε41}) [pure]",
  "inputSchema": {
    "type": "object",
    "properties": {
      "args": {
        "type": "array",
        "items": [{}],
        "minItems": 2,
        "maxItems": 2
      }
    },
    "required": ["args"]
  }
}
```

**After:**
```json
{
  "name": "docparse.services.samples.parseCsv",
  "description": "Parse a CSV file and return rows as typed records. Delimiter and header handling are automatic.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "filepath": { "type": "string" },
      "delimiter": { "type": "string" }
    },
    "required": ["filepath", "delimiter"]
  }
}
```

### Example 2: @route with Header Access

**Before (workaround — must use @raw, losing multipart parsing):**
```ailang
@raw
@route("POST", "/general/v0/general")
export func handleGeneral(req: HttpRequest) -> Json ! {IO} {
  -- req.headers has auth, but req.body is raw string
  -- must manually parse multipart... not possible in AILANG
  let apiKey = jsonGet(req.headers, "unstructured-api-key")
  error("Cannot parse multipart from raw body")
}
```

**After (multipart + headers together):**
```ailang
@route("POST", "/general/v0/general")
export func handleGeneral(files: bytes, _headers: Json) -> Json ! {IO} {
  -- files is parsed from multipart automatically
  -- _headers contains request headers as Json
  let apiKey = jsonGet(_headers, "unstructured-api-key")
  parseDocument(files)
}
```

### Example 3: Controlling MCP Visibility

```ailang
module docparse.services

-- Parse a DOCX file and return structured content.
-- Supports headers, tables, images, and track changes.
@route("POST", "/api/v1/parse-docx")
export func parseDocx(filepath: string) -> Document ! {IO} {
  -- This appears in MCP tools/list with the doc comment as description
  internalParse(filepath)
}

-- Internal XML namespace helper (not useful to agents)
@noexpose
export func docxNs() -> string {
  "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
}

-- Also hidden: no doc comment + no @route + --routes-only = auto-excluded from MCP
export func xmlEscape(s: string) -> string {
  replaceAll(replaceAll(s, "&", "&amp;"), "<", "&lt;")
}
```

---

## Success Criteria

- [ ] MCP `tools/list` with `--routes-only` returns ONLY `@route` functions
- [ ] MCP `tools/list` respects `@noexpose` annotation
- [ ] Tool names use portable package-relative paths (no machine-specific prefixes)
- [ ] Tool descriptions use doc comments; type signature only when no doc comment
- [ ] `inputSchema` uses named parameters with JSON Schema types (not generic `args[]`)
- [ ] `@route` handlers can declare `_headers: Json` parameter to receive request headers
- [ ] Multipart file upload + `_headers` access works together in same handler
- [ ] All existing HTTP/OpenAPI/A2A behavior unchanged
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

---

## Testing Strategy

**Unit tests** (`mcp_test.go`, `mcp_schema_test.go`):
- `ailangTypeToJSONSchema()` maps all AILANG primitive types correctly
- `portableModuleName()` strips absolute paths for packages and direct loads
- `buildMCPInputSchema()` produces named parameters from `ExportInfo`
- `isExposed()` filtering applies in `registerTools()`

**Integration tests** (`mcp_integration_test.go`):
- Start serve-api with `--routes-only --mcp-http`, verify `tools/list` returns only route functions
- Load a package module, verify tool names don't contain machine paths
- Function with doc comment → description is doc comment text
- Function without doc comment → description is type signature
- `@noexpose` function absent from `tools/list`
- Undocumented non-route function absent when `--routes-only`

**Manual testing:**
- Run docparse package via `ailang serve-api --mcp-http --routes-only`
- Connect Claude Desktop or Cursor to the MCP endpoint
- Verify agents can discover and call tools by name with named parameters
- Test Unstructured API endpoint with multipart upload + `_headers` for auth

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `portableModuleName()` algorithm for edge cases (nested packages, re-exports) — agent may choose, must pass integration tests
- Whether `ailangTypeToJSONSchema` maps ADT types to `"string"` or `"object"` — agent may choose, document the mapping
- Doc comment extraction: first line only vs full comment — agent may choose (recommend first paragraph)

---

## Non-Goals

**Not attempted in this feature:**
- MCP streaming/SSE support — separate concern, not needed for tool discovery
- MCP resource/prompt protocol support — tools-only for now
- Bidirectional MCP (AILANG as MCP client) — different architecture
- Full Unstructured API compatibility (chunking, strategies) — only header access for auth
- Changes to HTTP, OpenAPI, or A2A projections — those already work correctly

---

## Timeline

**Day 1** (~6 hours):
- Phase 1: MCP filtering + portable names
- Phase 2 start: DocComment field + extraction

**Day 2** (~6 hours):
- Phase 2 complete: descriptions + inputSchema with named params
- Phase 3: auto-exclude + `_headers` injection

**Day 3** (~4 hours):
- Phase 4: documentation, examples, changelog
- Integration testing with real MCP clients

**Total: ~16 hours across 3 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Named params break existing MCP consumers using `args[]` | Med | Check if any downstream MCP clients exist; if so, support both formats with deprecation warning |
| Doc comment extraction misses edge cases (multi-line, nested) | Low | Use first paragraph only; fall back to type sig on empty |
| `_headers` parameter name collision with user-defined params | Low | `_` prefix convention already established for `_body`/`_status`/`_headers` response fields |
| Package name collision (two packages named `docparse`) | Low | Use full qualified name with dots; agent can disambiguate |

---

## Related Documents

<!-- Auto-populated by Ollama neural search on "mcp quality and route headers" -->

**Implemented (informs design):**
- [m-serve-api-agent-enhancements-sprint-plan.md](../../implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md) — A2A + MCP initial implementation
- [m-dx-route-request-context.md](../../implemented/v0_10_0/m-dx-route-request-context.md) — `@raw` and HttpRequest record design (directly relevant)
- [m-dx-route-ctx-sprint-plan.md](../../implemented/v0_10_0/m-dx-route-ctx-sprint-plan.md) — Sprint plan for route context

**Planned (check for overlap):**
- [m-dx-serve-api-error-status.md](m-dx-serve-api-error-status.md) — Error status codes for serve-api
- [m-codegen-api-server.md](m-codegen-api-server.md) — Codegen for API servers
- [m-serve-api-dx.md](m-serve-api-dx.md) — General serve-api DX improvements

---

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [MCP Specification](https://spec.modelcontextprotocol.io/) - Model Context Protocol spec
- [Unstructured API](https://docs.unstructured.io/) - Target API compatibility
- Agent inbox messages: `f5f8ec8b` (docparse), `b20e61fe`/`4218e984`/`45ec546b`/`fe94c506`/`ef52a93f` (cli MCP bugs)

---

## Future Work

- MCP streaming support for long-running tools
- MCP resource protocol for file/data access
- Bidirectional MCP (AILANG as MCP client for tool chaining)
- Full Unstructured API feature parity (chunking strategies, encoding detection)
- Auto-generate doc comments from function body analysis

---

**Document created**: 2026-04-04
**Last updated**: 2026-04-04

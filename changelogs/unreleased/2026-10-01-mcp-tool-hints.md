### Added — MCP tool titles and behaviour hints (`@mcp_title`, `@mcp_hints`)

- `@mcp_title("Title")` sets the MCP tool `title`. `@mcp_hints("readOnly" | "destructive" | "idempotent" | "openWorld", ...)` sets `annotations`, and the list is complete: an omitted hint is `false`, including `destructive` and `openWorld` (whose MCP defaults are `true`). Anthropic's directory and OpenAI's Plugin Directory refuse tools with no title or no readOnly/destructive hint. Until now serve-api could not emit either, so prod AILANG Parse listed all 10 tools with `title: null, annotations: null`.
- An export with no `@mcp_hints` and an **empty declared effect row** is emitted as `readOnlyHint: true, openWorldHint: false`. An effectful export with no hints gets no annotations, plus one startup `WARN` naming every such tool. serve-api does not guess them.
- An unknown or duplicate hint, or `readOnly` together with `destructive`, is logged as an `ERROR` and the tool is not registered (the same posture as an invalid `@mcp_name`).
- Hint resolution is a single function, `serveapi/protocol.ResolveToolHints`. Both MCP implementations use it and emit identical JSON: the go-sdk path behind `serve-api`, and the stdlib `mcphttp` dispatcher, which gains `ToolDescriptor.Title` and `.Annotations`.
- The built-in `submit_feedback` tool is now annotated (`Send feedback`, additive, open-world).
- Example: `examples/runnable/serve_api_mcp_hints.ail`. Design: ailang-parse `design_docs/planned/v0_49_0/v0_49_0_vendor_directory_listing.md` (M1).

### Fixed — serve-api no longer reads purity from a stub

- `ExportInfo.Pure` comes from `iface.determinePurity`, which returns `true` for every export, so serve-api appended `[pure]` to the generated description of every undocumented tool. The MCP read-only proof uses the declared effect row instead.
- The `pure` keyword is not a proof either: `ailang check` accepts `pure func f(...) -> T ! {IO}`. That is filed separately; it is not changed here.

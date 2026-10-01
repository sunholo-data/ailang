### Fixed — zero-argument MCP tools were uncallable

- `func f()` desugars into one `()` param named `_`, and serve-api advertised it as a required string. A normal `tools/call` with `{}` was therefore rejected with `missing required parameter(s): _`. Prod AILANG Parse's `mcpFormats` was uncallable this way, and a directory reviewer's test case would have hit it.
- Params of type `()` or `unit` are no longer in `inputSchema` and bind to Unit. This covers both the desugared form and an explicit `u: unit`.

### Fixed — serve-api: an omitted record param binds its declared shape, not `{}`

- On `@route` (REST) calls, an omitted `@optional` `@mcp_file` param bound `{}`, so the function
  crashed with `record has no field: file_id`. The cause was general: every record-typed param
  collapsed to the type name `"record"`, whose zero is the empty record. A missing record param now
  binds the zero of its **declared** record type — each field at its own zero (`""`, `0`, `0.0`,
  `false`, `[]`, nested records; inline records and same-module aliases) — on REST named, positional,
  empty-body and multipart binding and on an omitted `@optional` MCP param alike. An omitted file
  param binds all four fields `""` on both surfaces.
- `@optional` on a param typed as a same-module record alias (`opts: Opts`) is now accepted on MCP
  (it was refused as "no zero value"). A record with a field that has no zero (an ADT, a function)
  keeps the old `{}`.

### Fixed — `ailang mcp check`: widget-only tools are exempt from the credentials rule

- A tool whose `_meta.ui.visibility` excludes `"model"` (serve-api `@mcp_app_only`) is never shown
  to the model, so a credential-shaped param on it (e.g. `token`) is reported as `SKIP` ("skipped:
  widget-only tool") instead of `FAIL`. The same param on a model-visible tool still fails; every
  other rule is unchanged.

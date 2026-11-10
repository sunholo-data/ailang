### Added
- **`ailang mcp check --target copilot`** (and `--target all`) checks a listed MCP surface for
  Microsoft 365 Copilot and Copilot Studio, from Microsoft Learn (sources C1–C3 in
  `m-serveapi-directory-ready-sources.md`): `copilot-registration` FAILs a server that offers
  neither a `registration_endpoint` nor `client_secret_basic`/`client_secret_post`, because Copilot
  does not use client ID metadata documents; `copilot-file-params` warns about tools that depend on
  `openai/fileParams`, which Copilot does not support; `copilot-widgets` warns that widgets need
  Copilot's per-server widget origin allowed in CORS and at the identity provider (Microsoft's
  generator is linked; the checker does not compute the host). `--target all` = anthropic + openai +
  copilot. On 2026-11-10 the dev executor and dev Parse pass it; prod fails registration until the
  client-model switches are turned on there.

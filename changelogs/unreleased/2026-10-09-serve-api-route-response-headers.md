### Fixed

- Route response `_headers` records map underscores to hyphens; Json JObject string headers preserve exact names on raw and @nowrap responses, including Result.Ok. Invalid headers now produce actionable registration errors or structured 500 responses with ERROR logs; timing/CORS/Vary headers remain server-owned. Raw Content-Type defaults are sent before status, and program Content-Type overrides them. Refs #1609.

### Added

- `mcp check` probes discovered authorization endpoints with dummy code-flow/S256 requests. Assessable HTML without framing protection fails anthropic/both and warns for openai; inconclusive pages warn. Refs #1597.

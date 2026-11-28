### Fixed — MCP docs tools work on a prerelease server (`1.0.0-rc.1`)

- `example_get` and `examples_list` called with an empty `forVersion`, meaning "latest", failed
  with "unknown version ''" once the server's own version became `1.0.0-rc.1`. `isSemver` in
  `mcp_tools/live.ail` accepted only `X.Y.Z`. It now also accepts a prerelease tag (letters,
  digits, `.` and `-`), which matches the `v1.0.0-rc.1` release tag the files are fetched at.
- This turned `dev` CI red on all three platforms from the v1.0.0-rc.1 release commit
  (`TestDocsTools_EveryToolSucceeds`). `not-a-version` and path-like inputs are still refused.

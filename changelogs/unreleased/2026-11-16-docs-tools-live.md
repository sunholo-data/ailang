### Fixed — every hosted docs tool now works on valid input (#2271)
- Six docs tools on `mcp.ailang.sunholo.com` and the hosted executor failed on every call, because
  `tools/build-snapshot` never generated the files they read. Both connector directories require
  every listed tool to succeed on valid input.
  - `example_get` and `examples_list` read `examples/` and `examples/manifest.json` live from the
    release tag. `examples_list` returned `[]`; it now filters the manifest by tag and status and
    returns at most 50 with a total.
  - `design_doc` and `roadmap` read the `dev` branch live, so they are current rather than as of
    the last release. `design_doc` falls back to the release tag for a document that has moved.
    Both return markdown, cut at 30,000 characters with `truncated` and a link to the whole file.
  - `benchmarks_compare` compares two models' snapshot summaries side by side (passes, runs, pass
    rate, cost, failures by category); its unused `version` parameter is gone.
  - `prompt_get` no longer offers `kind: compact`, which no current version has.
  - `benchmark_run` is removed: no per-run data is published, so it could never succeed.
- The live reads use a new module, `mcp_tools/live.ail`. Every URL is a fixed base, a ref (`dev` or
  the release tag) and a path limited to `[A-Za-z0-9._/-]` with no `..`. The docs MCP and the
  IAM-only executor now run with `Net` (`docker/Dockerfile.mcp`, `docker/Dockerfile.executor-mcp`);
  the OAuth executor already had it. Offline use is unaffected: the CLI never loads these tools.
- New test `TestDocsTools_EveryToolSucceeds` builds the snapshot, serves the repository from a local
  stand-in for GitHub, and calls every docs tool once with valid input. A tool without a case fails
  the test, and five refusals must come back as tool errors. `ailang mcp check` had passed 10/10
  throughout, because it checks the listing, not the calls.

### Changed — serve-api: an MCP tool returning `Result` gets the meaning HTTP already gives it
- `Ok(value)` answers with the value; `Err(value)` is a tool error (`isError: true`) with the payload
  as its text (a string as itself, anything else as JSON). Before, a `Result` crossed MCP as a
  successful `{"__tag": "Err", ...}` object. No published MCP tool returned `Result` before this
  change. `internal/apiserver/mcp_result.go`, documented in the serve-api guide.

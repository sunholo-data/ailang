### Changed — 1.x Stability Promise re-drafted against today's surface (v1.0 bar v3, D-64) (2026-11-09)

`docs/docs/reference/stability.md` predated the CLI reshape (M-V1-SIMPLIFY-S5), the six stdlib
modules added since July, the package format work and the hosted executor MCP. Its tier
assignments are redone from the sources of truth. The tier semantics are unchanged (frozen by
M-V1-STABILITY-PROMISE), and ratification by Mark is still pending.

- **CLI** (Stable 19 / Experimental 23 / Internal 46) built from the dispatch table, which is the
  same table behind `ailang --help`, the `dev`/`ops`/`eval`/`pkg` group help and
  `docs/docs/reference/cli.md`. Every row appears in a group's `--help`, and compatibility
  spellings are folded into their canonical row.
- **Stdlib** (17 / 31 / 3) covers all 49 modules in `.stdlib-golden/` plus `std/ai/streaming`
  and `std/stream/bridge`. New rows: `set` (Stable ⚠), `yaml`, `audio`, `terminal`, `web`
  (Experimental) and `trace_test` (Internal). `std/debug` moves Internal → Experimental ⚠.
- **Packages** section (G3): manifest, lockfile (`ailang.lock/v1`), authority (effect ceiling,
  ghost effects, run-time grant, confined-agent chokepoint) and registry resolution, tiered 12
  Stable / 9 Experimental / 1 Internal, with `sunholo/mcp_oauth` and its in-repo consumer
  `executor_oauth/` as the worked example.
- **Hosted MCP tools** (G4a): all 29 tools from `mcp_tools/`, `executor_tools/`, `executor_oauth/`
  and the built-in `submit_feedback`, with 24 Stable and 5 Experimental (benchmarks and feedback).
- D-64 rulings applied: `messages` is Experimental. Coordinator, missions, approvals, dashboards
  (`serve`), eval tooling and chains/observatory/trace are Internal.

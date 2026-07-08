# M-SERVE-API-LIVE-TOOL-REGISTRY: Live Tool Registry for Agentic Sessions

**Status**: Planned  
**Target**: v0.18.0  
**Priority**: P1  
**Estimated**: 3–4 days  
**Dependencies**: v0.17.1 M-AILANG-EXT-REGISTRY-GEN (background context), motoko --resume (motoko-side, independent)

---

## Problem

An AI agent (e.g. motoko) can currently create AILANG tool files during a session, but cannot
use those tools in the same session without a restart. The restart discards all conversational
context.

Concretely:

1. Agent is mid-task, decides it needs a `compact_memory` tool.
2. Writes `tools/compact_memory.ail` to disk.
3. **Today**: must restart `ailang serve-api` (or the whole motoko session) to pick up the new
   tool. Restart = lost context unless `--resume` is implemented (motoko-side, separate sprint).
4. **Even with `--watch`**: hot-reload recompiles *modified* existing files, but new files added
   post-startup don't register as MCP tools. The MCP server is built once at startup.

This creates a hard session boundary every time an agent self-extends, which is the wrong
default for agentic workflows.

### What already works (v0.17.0)

- `serve-api --watch` recompiles modified `.ail` files via fsnotify + debounce (200ms).
- The engine's `InvalidateModule` + lazy-recompile pattern means updated logic is live on
  the next HTTP request.
- The `/api/<module>/<func>` catch-all handler resolves modules at request-time from
  `s.modules`, so *new* files that get loaded are accessible at their natural URL path
  even without re-registering routes.

### What doesn't work

- **New files**: `--watch` only watches directories of files loaded at startup. A new
  `.ail` file written into a watched dir fires a `Create` event, but `reloadFile` calls
  `s.loadFile` which adds to `s.modules` — while the MCP server, OpenAPI spec, and
  `@route` custom registrations remain stale.
- **MCP tool list**: `NewMCPServer(s)` is called once. New modules never appear as tools
  until restart.
- **`@route` custom paths**: registered at startup via `registerCustomRoutes`. New files
  with `@route` annotations added post-startup don't get their custom paths wired.
- **Agent feedback**: serve-api logs `Hot reloaded: <path>` to stderr. No structured
  event that motoko (or any supervisor) can parse to know a new tool is available.

---

## Goals

1. New `.ail` files dropped into a `--watch` directory appear as MCP tools within ~1s,
   with no server restart.
2. MCP tool list and OpenAPI spec reflect the current set of loaded modules, not the
   startup snapshot.
3. Agent supervisor (motoko, coordinator) can detect when a new tool becomes available
   via a structured event channel, without polling.
4. The feature is opt-in and safe — non-agentic serve-api usage is unaffected.

### Out of scope

- Versioning or pinning of live tools (live tools are intentionally mutable).
- Native AILANG type safety across the tool boundary (tools are JSON-in / JSON-out).
- Removing the restart requirement for *compile-time* extensions (that's the
  generate-extension-registry + `--resume` path; see below).

---

## Design

### 1. Dynamic MCP server rebuild on new file

When a `Create` event fires for a new `.ail` file in a watched dir:

1. `loadFile(absPath)` — compile + add to `s.modules` (already happens in `reloadFile`).
2. **New**: rebuild MCP server: `s.mcpServer = NewMCPServer(s)` and swap the handler.
   `NewMCPServer` is cheap (pure metadata walk over `s.modules`). Thread-safe swap via
   the existing `s.mu` lock.
3. **New**: re-register `@route` custom paths for the new module only (incremental, not
   full route table rebuild — Go's `http.ServeMux` doesn't support unregistration, so
   we need a thin dynamic wrapper; see §3 below).

```go
// watcher.go — expanded reloadFile

func (s *Server) reloadFile(absPath string) {
    // ... existing compile + InvalidateModule ...

    isNew := wasNewFile // track whether s.modules had this path before loadFile
    if isNew {
        s.rebuildMCPServer()          // NEW
        s.registerLateRoutes(info)    // NEW — @route for this module only
        s.emitToolLoadedEvent(info)   // NEW — structured event
    }
    log.Printf("  Hot reloaded: %s", modulePath)
}
```

### 2. Watch directory for new files (not just modifications)

Currently `getWatchDirs()` builds the watch list from `s.watchPaths` (files loaded at
startup). For agentic use the directory should be watched even before any files exist
there.

**New flag**: `--tool-dir <path>` (can be specified multiple times). Directories named
here are watched at startup regardless of whether they contain any `.ail` files. All
`.ail` files found there at startup are loaded; new files are auto-loaded as they appear.

```bash
ailang serve-api --tool-dir ./live-tools/ --mcp --watch ./api/
```

- `--tool-dir` implies `--watch` for that directory.
- Files in `--tool-dir` are loaded with `--routes-only` semantics by default (only
  `@route`-annotated or explicitly exported functions are exposed). Override with
  `--tool-dir-expose-all`.
- The basePath for module name derivation for tool-dir files uses the tool-dir root
  (so `live-tools/compact_memory.ail` → MCP tool name `compact_memory`).

### 3. Dynamic route registration

Go's `http.ServeMux` doesn't support adding routes after `ListenAndServe`. Two options:

**Option A** (preferred, lower risk): Replace `registerCustomRoutes` with a thin
`dynamicRouter` that checks `s.modules` at request time for `@route` metadata:

```go
type dynamicRouter struct {
    s *Server
}

func (dr *dynamicRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Walk s.modules for @route matching r.Method + r.URL.Path
    // Fall through to /api/ catch-all if no match
}
```

This makes `@route` lookup O(modules×routes) per request, which is fine for tool-dir
use (typically < 20 modules). For production serve-api with many modules, the existing
static registration remains the default; the dynamic router only activates with
`--tool-dir`.

**Option B**: Replace `net/http.ServeMux` with `gorilla/mux` or `chi` which support
late route addition. Higher dependency footprint; deferred unless Option A proves
inadequate.

### 4. Structured tool-loaded events

When a new tool is loaded, emit a structured JSON line to a designated channel:

**Option A** (simplest): append to a per-session events file:
```
~/.ailang/sessions/<session-id>/events.jsonl
```
Each line:
```json
{"event":"tool_loaded","name":"compact_memory","path":"live-tools/compact_memory.ail","timestamp":"2026-05-07T14:30:00Z","exports":["compact_memory"]}
```

**Option B**: HTTP SSE stream at `/api/_events` (already considered for other features).
motoko can subscribe once and receive events.

**Option C** (immediate, zero new infrastructure): structured stderr log:
```
TOOL_LOADED: compact_memory (live-tools/compact_memory.ail)
```
motoko already captures stderr; a simple grep pattern suffices.

**Recommended**: Option C for v0.18.0 (zero cost, works immediately), Option A or B
as follow-on if a richer event bus is needed.

### 5. `--tool-registry` convenience flag

Sugar for the common agentic pattern:

```bash
# Equivalent to: --tool-dir ./tools/ --watch --mcp --routes-only --caps IO,AI
ailang serve-api --tool-registry ./tools/ my-api.ail
```

Sets up sensible defaults for an agentic tool sandbox:
- `--tool-dir ./tools/` (live reload from that dir)
- `--mcp` enabled
- `--watch` on both the main files and tool-dir
- `--caps IO,AI` (standard agentic capabilities)
- Files in `--tool-registry` dir are auto-exposed as MCP tools

Optional, can be deferred to a follow-up sprint if the individual flags suffice.

---

## The Two-Track Model

This feature and `generate-extension-registry` solve *different* problems. They coexist:

| | Live tool registry (`--tool-dir`) | Native extensions (`generate-extension-registry`) |
|--|--|--|
| **Use case** | AI creates a quick tool mid-session | Versioned, typed, published extension packages |
| **Session boundary** | None — tool available in < 1s | Required — needs recompile + restart |
| **Type safety** | JSON at boundary | Native AILANG types, compile-time checked |
| **Versioning** | Mutable (no pin) | Pinned in `ailang.lock` |
| **Tool is permanent?** | No (session-scoped) | Yes (committed to repo) |
| **Restart mitigation** | N/A | motoko `--resume <session-id>` |

The recommended workflow for motoko:

```
Quick exploration: agent writes to --tool-dir → live reload → no restart → same session
Permanent tool:   agent graduates tool to extension package → generate-extension-registry
                   → motoko --resume → continues with richer capability
```

---

## What We Lose vs Native Extensions

1. **Type safety** — tools are JSON-in / JSON-out via HTTP. No compile-time guarantee
   that `compact_memory`'s return type matches what the calling code expects.
2. **Pure functions** — everything has the `{IO}` effect. Can't reason about side-effect
   freedom statically.
3. **Versioning** — live tools are mutable. No `ailang.lock` pin. A tool that was working
   an hour ago may behave differently now.
4. **Performance** — HTTP round-trip per tool call vs. direct function call. For
   agentic workflows (where LLM latency dominates) this is irrelevant; for tight
   compute loops it may matter.
5. **Cross-tool imports** — tools in `--tool-dir` can import standard library and
   existing server modules, but they cannot import *each other* (not compiled together).
   Workaround: structure tools as independent units.

---

## Conflict Surface

Changes touch `internal/apiserver/` (watcher.go, server.go, mcp.go, routes.go) and
`cmd/ailang/serve_api.go`. This is serve-api infrastructure, not parser/typechecker.
No AILANG language changes.

**Risk**: MCP server rebuild on new file is thread-safe only if protected by the server
mutex. `s.mcpServer` write must hold `s.mu`. The HTTP handler reads `s.mcpServer` — must
use `s.mu.RLock()` for reads. Currently MCP handler is stored in the mux (immutable after
`buildRoutes`); the new design stores it in `s.mcpServer` and dispatches dynamically.

**Programs that MUST still work post-change** (regression fixtures):
1. `ailang serve-api --watch api/` — existing watch mode, no new files → no behaviour change
2. `ailang serve-api api/main.ail --mcp` — static MCP registration, no watch → no change
3. `ailang serve-api --tool-dir tools/ api/main.ail` — new flag, startup files loaded,
   new `.ail` in `tools/` appears as MCP tool, existing routes unaffected
4. `ailang serve-api --watch api/ --mcp` — modified existing file still recompiles;
   MCP tool list updated in place (same tool, fresh implementation)

---

## Acceptance Criteria

- [ ] New `.ail` file written to a `--tool-dir` directory appears in `/mcp/tools/list`
  response within 2 seconds (no server restart)
- [ ] Modified `.ail` in `--tool-dir` recompiles and serves new logic within 2 seconds
  (existing behaviour, regression tested)
- [ ] `@route` annotations in new tool-dir files are honoured (custom path accessible)
- [ ] MCP server rebuild is thread-safe (no data race under `go test -race`)
- [ ] `ailang serve-api --help` documents `--tool-dir`
- [ ] Structured stderr line `TOOL_LOADED: <name> (<path>)` emitted on new file
- [ ] Existing serve-api tests pass unchanged (no regression)
- [ ] Example: `examples/runnable/live_tool_registry.ail` + README showing the pattern

---

## Files to Modify

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/apiserver/watcher.go` | Detect new files, call rebuildMCPServer | +40 |
| `internal/apiserver/server.go` | Add `toolDirs []string`, `rebuildMCPServer()`, dynamic MCP dispatch, `emitToolLoadedEvent` | +80 |
| `internal/apiserver/routes.go` | `dynamicRouter` wrapper, `registerLateRoutes` | +60 |
| `cmd/ailang/serve_api.go` | `--tool-dir` flag, pass to Config | +20 |
| `internal/apiserver/server_test.go` | New file: live registry integration tests | +120 |
| `docs/docs/guides/serve-api.md` | Document `--tool-dir`, agentic workflow section | +50 |

Total: ~370 LOC

---

## Relationship to motoko `--resume`

The motoko `--resume <session-id>` flag (proposed in msg 67ff70cc) and this feature
are complementary, not alternatives:

- **This feature**: eliminates the session boundary for *quick tools* (written to
  `--tool-dir`, available in < 1s, no restart).
- **`--resume`**: handles the session boundary that still exists for *permanent
  extensions* (require recompile; restart is necessary; `--resume` prevents context loss).

Both should ship. Neither blocks the other. Priority ordering:
1. This feature (`--tool-dir`) — unblocks the common agentic case with no architecture
   change on the motoko side.
2. `--resume` — improves ergonomics for extension graduation and other session-boundary
   scenarios.

---

## Related Documents

- [M-AILANG-EXT-REGISTRY-GEN](../../implemented/v0_17_1/m-ailang-ext-registry-gen.md) — compile-time extension registry (the "permanent" track)
- [M-EXTERNAL-CONSUMER-DX](../v0_17_0/m-external-consumer-dx.md) — external developer UX
- [serve-api guide](../../../docs/docs/guides/serve-api.md) — current serve-api documentation
- motoko msg 67ff70cc — `--resume <session-id>` proposal (session resumption counterpart)

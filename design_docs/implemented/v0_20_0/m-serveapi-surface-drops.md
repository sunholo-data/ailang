# M-SERVEAPI-SURFACE-DROPS — Refuse to start when declared `@route` modules are silently dropped

**Status**: ✅ Implemented in v0.20.0 (M1: 06450b85, M2: 93f4b6fb, M3: eea88f33, /health-fix: 000da04a, 2026-05-15). Closes the docparse v0.14.1 silent-zero-entitlements class of bug (inbox `e1814c9f`).
**Target**: v0.20.0 (shipped one minor ahead of the originally-targeted v0.21.0)
**Priority**: P0 — directly fixes a class of production-visible silent failures
**Estimated**: ~1 day (200-300 LOC + tests)
**Dependencies**: None (builds on the partial fix in commit `a629a129`, shipped in v0.18.4)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism; same modules always either register or fail-fast |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Operators can now verify "all declared routes are reachable" at startup, not at first customer request |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `/api/v1/health` now machine-readable signals partial registration — readiness probes can gate traffic |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost surface change |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | **+2** | Direct application of "NO SILENT FALLBACKS" — converts a silent zero-degradation into a typed startup error |
| A12: System Boundary | +1 | Boundary between "loaded but registered" vs "loaded but dropped" becomes explicit and visible |

**Net Score: +5** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — failure mode is deterministic given the input layout
- [x] A3 (Effects): No hidden side effects — the new `os.Exit` on declared-route-drop is at startup, before the HTTP listener binds
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Aligns with machines-first — turns a "operator-grep-logs" failure into a `/health` JSON field

## Problem Statement

`registerModule` in `internal/apiserver/module_entry.go:52-69` silently rejects any loaded module whose symlink-resolved physical path doesn't live under `s.normalizedBasePath`. Returns `("", false, nil)` and the caller skips. Before commit `a629a129` (v0.18.4), this rejection was completely silent — no log, no banner mention. After `a629a129`, a `log.Printf("  Skipped: …")` line surfaces it, but only when the operator goes to look. That's still grep-after-symptom, not fail-loud.

### How this manifested in production (May 2026)

The `docparse` agent's `billing_service_api` was deployed on Cloud Run under AILANG v0.14.1, with `Dockerfile.billing` running `ailang serve-api … .` from inside `~/.ailang/cache/registry/sunholo/billing_service_api/0.5.7/`. The catalog-parsing module (`billing_entitlements/plan.ail`) lived outside that basePath. `registerModule`'s filter silently dropped it.

The handler `entitlements_handler` *compiled* against the loaded module (the loader's view was complete), but at runtime the dropped module's top-level catalog state was effectively `[]`. `freePlanFromCatalog([])` returned a zero-value Plan literal. `planToEntitlements(zeroPlan)` returned a zero-value Entitlements. The HTTP response was:

```json
{"plan":"","status":"","canOperate":"false","apiAccess":"false",
 "maxFileSizeMb":"0","monthlyRequestLimit":"0", ...}
```

Every field at its zero. Routes 200'd. Startup banner said *"6 handler modules registered ✓"*. JSON was well-formed. The system appeared healthy. The bug was caught by a customer-visible Cloud Run symptom — the latest possible signal.

Bug correspondence: `ailang messages` thread starting `e1814c9f-4620-4e3c-a051-4dd1a38d392b` (docparse → ailang-core), close-out `952d6ef0`.

### Why this is a class, not a one-off

The same silent-drop pathway fires whenever:
1. `serve-api` runs from inside a published package's cache directory (a common deployment pattern for single-purpose service containers — see `Dockerfile.billing`'s "init wrapper package, copy ailang.toml/lock, run from cache" recipe).
2. A handler's transitive imports resolve to files outside basePath.
3. The package author used relative `./` imports inside the package (the dual-canonical-ID class, tracked separately).

Any future deployment matching this layout has the same silent-degradation cliff. The v0.14.1 → v0.18.4 fix gave us a stderr log; we need the system to refuse to silently degrade.

**Impact:** every operator running `serve-api` against a packaged service is exposed. The cost of the docparse incident was ~2 hours of incorrect billing dashboard data + a re-promotion. The cost of catching it via Cloud Run rather than via a `/health` probe was, in principle, unbounded.

## Goals

**Primary Goal:** A `serve-api` invocation that drops a module carrying any declarative annotation (`@route`, `@raw`, `@nowrap`, `@noexpose`, `@mcp_name`) must fail to start with a clear, actionable error — not silently 200 with empty data.

**Success Metrics:**
- Repro of the docparse v0.14.1 layout under post-fix AILANG fails fast at startup with a named-module error, exit code ≠ 0
- A stdlib-only drop (e.g. a resolution edge case for `std/option`) still logs but doesn't fail startup — strictness is *gated on author intent* (annotations), not on file location
- `/api/v1/health` includes a `dropped_modules` array, allowing a readiness probe to detect partial registration before traffic
- Zero new dependencies; no new public API surface beyond one CLI flag / env var

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Strict-by-default vs warn-by-default for declared-route drops | Strict-by-default breaks any deployment with a pre-existing silent drop on upgrade — but warn-by-default leaves the failure-class open. CLAUDE.md "NO SILENT FALLBACKS" says strict. | human (this doc) | design | high |
| Escape hatch flag/env var for migration | Without one, operators upgrading a working-but-silently-degraded deployment will hit a startup failure with no recovery path. With one, the failure is opt-out-able and operators can deploy then fix. | human (this doc) | design | med |
| Which annotations count as "author declared intent to expose" | `@route` is the obvious one. `@noexpose` is interesting — author explicitly said "don't expose me" but presumably wants the module loaded for its definitions. `@mcp_name` and `@raw` only apply to `@route`. So the trigger set is just `@route`. | human (this doc) | design | low |
| Failure modality: exit-non-zero vs return error from `New()` | `New()` returning an error is cleaner for testability but adversely affects the CLI surface. `os.Exit(1)` at the boundary in `cmd/ailang/serve_api.go` is fine. | agent | compile | low |

### Design Freeze

- [x] **Strict-by-default for declared-route drops.** CLAUDE.md "NO SILENT FALLBACKS" is the governing principle; this is a direct application.
- [x] **Escape hatch: `AILANG_SERVE_API_ALLOW_DROPS=1` env var.** Sets the server to warn-not-fail for the current invocation. Logged loudly at startup so it doesn't become permanent. No CLI flag equivalent — env-var-only forces operators to make the bypass explicit in their Dockerfile/manifest.
- [x] **Trigger set: only `@route`-annotated modules cause strict failure.** Other annotations (`@noexpose`, `@mcp_name`, `@raw`, `@nowrap`) only modify route behavior — they don't independently declare "expose me." A `@noexpose` module getting dropped is a real concern but a lower-stakes one (no customer traffic relies on it). Defer to a follow-up.

## Solution Design

### Overview

Three changes in concert:

1. **Track drops, don't just log them.** `registerModule` returns the drop reason; the caller accumulates a `[]DroppedModule` on the server.
2. **Surface drops in the startup banner.** Single WARN line with the count and a CSV of declared paths.
3. **Fail-fast on annotation-bearing drops.** After all modules are processed but before the HTTP listener starts, if any drop carried a `@route` annotation, emit a fatal error naming the file, declared module path, basePath, and the actual file location — then exit non-zero (unless `AILANG_SERVE_API_ALLOW_DROPS=1`).

The under-basePath filter itself stays. We're not changing *which* modules get rejected; we're surfacing the rejections.

### Architecture

```
loader.Load() ──► LoadedModule (all imports resolved, including out-of-basePath)
        │
        ▼
Server.registerModule(loaded)
   ├── case A: under basePath ──► s.modules[physicalPath] = info  (existing path)
   └── case B: outside basePath ──► s.droppedModules = append(...) (NEW)
                                    return ("", false, nil)
        │
        ▼ (after all modules processed)
Server.validateRegistration()  (NEW)
   ├── for each drop:
   │     if drop has any @route annotation:
   │         fatalErrors = append(fatalErrors, drop)
   │     else:
   │         warnings = append(warnings, drop)
   │
   ├── log WARN banner: "⚠  Dropped N module(s) outside basePath: a.ail, b.ail, ..."
   │
   └── if len(fatalErrors) > 0 && !allowDrops:
         return MultiError(fatalErrors)
         ▲
         │ wrapped in cmd/ailang/serve_api.go to os.Exit(1) with a clear human message
```

**Components:**
1. **`DroppedModule` struct** — captures `physicalPath, declaredPath, fileBaseName, annotations []string` so we can identify and report each drop.
2. **`Server.droppedModules` field** — slice accumulated during loadModules. Read by validation step and `/api/v1/health`.
3. **`Server.validateRegistration()` method** — runs after loadModules, before the listener binds. Partitions drops into fatal vs warn, emits banner, returns error on fatal-and-not-allowed.
4. **`/api/v1/health` extension** — new `dropped_modules: [...]` field. Empty array = healthy. Non-empty array = partial registration (which means `allowDrops` was set, since otherwise we'd have failed startup).

### Implementation Plan

**Phase 1: Track drops** (~2 hours)
- [ ] Add `DroppedModule` type to `internal/apiserver/types.go`
- [ ] Add `droppedModules []DroppedModule` field to `Server` (with mutex protection — already have `s.mu`)
- [ ] Modify `registerModule` ([module_entry.go:52-69](../../../internal/apiserver/module_entry.go#L52-L69)) to append to `droppedModules` instead of `return ("", false, nil)` directly; extract annotation list from `loaded.File` before returning
- [ ] Unit test: a loaded module outside basePath produces a `DroppedModule` entry

**Phase 2: Surface in banner + validate** (~3 hours)
- [ ] New method `validateRegistration() error` in `internal/apiserver/server.go`
- [ ] Partition `droppedModules` into `fatal` (has @route) vs `warn` (no annotations)
- [ ] Emit single WARN log line with count + module names
- [ ] If `len(fatal) > 0 && os.Getenv("AILANG_SERVE_API_ALLOW_DROPS") != "1"`, return a multi-line error naming each fatal drop
- [ ] Call `validateRegistration` from `cmd/ailang/serve_api.go` after `apiserver.New()` and before `Serve()`
- [ ] On error: print a structured human message to stderr, then `os.Exit(1)`
- [ ] Unit + integration tests

**Phase 3: Health endpoint + docs** (~2 hours)
- [ ] Extend health handler ([internal/apiserver/handlers.go](../../../internal/apiserver/handlers.go)) to include `dropped_modules` array (lazy — only present when non-empty, to keep healthy `/health` responses minimal)
- [ ] Update `docs/docs/guides/serve-api.md` with the new strict behavior, the escape hatch, and the health-endpoint shape
- [ ] Update `cmd/ailang/help.go` for the new env var (per `.claude/rules/api-server.md`'s annotation-sync discipline applied to env vars)
- [ ] Changelog entry under `[Unreleased]` in `changelogs/v0.10-current.md`

### Files to Modify/Create

**New files:**
- `cmd/ailang/serve_api_drop_test.go` — integration test fixturing the docparse v0.14.1 layout, ~150 LOC

**Modified files:**
- `internal/apiserver/types.go` — `DroppedModule` struct, ~20 LOC
- `internal/apiserver/module_entry.go` — drop accumulation, ~30 LOC delta
- `internal/apiserver/server.go` — `validateRegistration` method + field, ~80 LOC
- `internal/apiserver/handlers.go` — `/health` extension, ~15 LOC
- `cmd/ailang/serve_api.go` — call `validateRegistration`, handle exit, ~25 LOC
- `cmd/ailang/help.go` — document `AILANG_SERVE_API_ALLOW_DROPS`, ~10 LOC
- `docs/docs/guides/serve-api.md` — strict-mode section, ~40 lines prose
- `changelogs/v0.10-current.md` — `[Unreleased]` entry

## Examples

### Example 1: Production layout that previously silently zeroed

**Before (v0.14.1 production behavior — and the silent fallback persists if author hasn't moved to v0.18.4):**
```
$ ailang serve-api --caps Net,FS,Env,IO --port 18749 "$API_DIR"
Starting AILANG API server on :18749
  Registered: handlers/entitlements_handler (4 exports)
  Registered: handlers/checkout_handler (3 exports)
  ... (4 more)
  6 handler modules registered ✓
Listening on http://localhost:18749

$ curl '.../billing/me/entitlements?args=uid&args=2026_05'
{"plan":"","monthlyRequestLimit":"0","canOperate":"false",...}  # silent failure
```

**After (v0.21.0):**
```
$ ailang serve-api --caps Net,FS,Env,IO --port 18749 "$API_DIR"
Starting AILANG API server on :18749
  Registered: handlers/entitlements_handler (4 exports)
  Registered: handlers/checkout_handler (3 exports)
  ... (4 more)
  6 handler modules registered ✓
⚠  Dropped 1 module(s) outside basePath: billing_entitlements/plan

Error: serve-api refuses to start with @route-bearing modules dropped:
  • billing_entitlements/plan
      declared: pkg/sunholo/billing_entitlements/plan
      basePath: /root/.ailang/cache/registry/sunholo/billing_service_api/0.5.7/
      resolved: /root/.ailang/cache/registry/sunholo/billing_entitlements/0.4.1/plan.ail
      annotations: @route

These modules carry @route annotations — their author declared them as
exposed endpoints. Silently dropping them would return zero-valued
responses to customer traffic.

Resolution options:
  1. Move basePath outward (use a project root that contains all imports)
  2. Replace relative ./X imports with canonical pkg/... imports
  3. Set AILANG_SERVE_API_ALLOW_DROPS=1 to start anyway (NOT recommended)

exit status 1
```

### Example 2: Stdlib drop (non-annotation-bearing) — warn but proceed

**Behavior:**
```
$ ailang serve-api --caps Net,FS,Env,IO --port 18749 "$API_DIR"
Starting AILANG API server on :18749
  Registered: app/main (2 exports)
  1 handler modules registered ✓
⚠  Dropped 1 module(s) outside basePath: std/option (no @route — non-fatal)
Listening on http://localhost:18749
```

Operator sees the drop, can investigate, but stdlib resolution edges don't keep the service down.

### Example 3: `/api/v1/health` with partial registration (escape hatch in use)

```
$ AILANG_SERVE_API_ALLOW_DROPS=1 ailang serve-api ...
$ curl http://localhost:18749/api/v1/health
{
  "status": "degraded",
  "modules_registered": 6,
  "dropped_modules": [
    {
      "declared": "pkg/sunholo/billing_entitlements/plan",
      "resolved": "/root/.ailang/cache/.../plan.ail",
      "annotations": ["@route"]
    }
  ],
  "warning": "AILANG_SERVE_API_ALLOW_DROPS is set — service is running with declared @route modules dropped"
}
```

Readiness probe checks `status != "degraded"` → routes traffic away from this revision until operator fixes the deployment.

## Success Criteria

- [ ] Reproduction of the docparse v0.14.1 layout (described in Problem Statement) FAILS to start under v0.21.0 with exit code 1 and the named-module error
- [ ] Same reproduction with `AILANG_SERVE_API_ALLOW_DROPS=1` STARTS, with a warning logged at startup and `dropped_modules` non-empty on `/health`
- [ ] A drop of a non-annotation-bearing module (e.g. a stdlib resolution edge) WARNS but starts
- [ ] `cmd/ailang/serve_api_drop_test.go` covers all three paths
- [ ] All existing `internal/apiserver` tests still pass (no regression on the success path)
- [ ] `make ci` passes
- [ ] CHANGELOG entry added to `[Unreleased]` in `changelogs/v0.10-current.md`
- [ ] `docs/docs/guides/serve-api.md` documents the strict behavior and escape hatch
- [ ] `ailang help` shows `AILANG_SERVE_API_ALLOW_DROPS` in the env var list

## Testing Strategy

**Unit tests** (`internal/apiserver/module_entry_test.go`):
- A LoadedModule whose physical path is outside basePath produces a `DroppedModule` entry
- A LoadedModule under basePath registers normally with `droppedModules` empty
- `extractAnnotations` correctly captures `@route`, `@raw`, `@nowrap`, `@noexpose`, `@mcp_name` presence

**Integration tests** (`cmd/ailang/serve_api_drop_test.go`):
- Fixture A: project layout with @route-bearing module outside basePath → `serve-api` exits 1
- Fixture B: same layout + `AILANG_SERVE_API_ALLOW_DROPS=1` → `serve-api` starts, `/health` returns `degraded`
- Fixture C: project layout with only non-annotation-bearing drop → `serve-api` starts cleanly, banner has WARN line

**Manual testing:**
- Run against the actual `billing_service_api` cache-dir layout (request docparse provide a captured tarball if needed)
- Verify the error message is readable on Cloud Run log output (no ANSI escape leakage, line-wraps reasonably)

## Deferred Decisions

- **Strict-fail for `@noexpose` drops** — author declared the module's intent, but didn't ask for route exposure. Defer; track as M-SERVEAPI-NOEXPOSE-DROPS if it ever bites.
- **`/health` schema** — exact JSON shape may evolve to align with whatever observability dashboards consume it. Implementer may choose snake_case vs camelCase to match neighboring fields.
- **Banner formatting** — implementer may format the warning line to match the existing `Registered: …` log shape rather than introducing a new format.

## Non-Goals

- **Rewriting the under-basePath filter logic.** The filter behavior is correct: modules outside basePath aren't this serve-api's responsibility. We're surfacing the rejection, not changing it.
- **Changing record structural semantics or runtime type tags.** The "empty record" leak from `freePlanFromCatalog([])` is fundamentally a *value-layer* concern that structural typing can't catch. Best practices for package authors (Result-wrapping, `ensures`-at-boundary) are a docs push, tracked under "Best Practices Push" below.
- **Fixing the dual-canonical-ID class.** Tracked separately as M-LOADER-CANONICAL-REWRITE (one physical .ail loaded twice under `./X` and `pkg/...` canonical IDs). Lower priority since the under-basePath surfacing covers the same operational symptom.
- **Changing `loader` behavior.** The loader is correctly loading the modules; the apiserver layer is the one making the registration decision.

## Timeline

**Day 1, AM** (~3 hours):
- Phase 1: track drops in `registerModule`
- Phase 2 start: `validateRegistration` skeleton

**Day 1, PM** (~3 hours):
- Phase 2 finish: banner + fatal-error flow + cmd/ailang wiring
- Phase 3: `/health` extension

**Day 2, AM** (~2 hours):
- Integration tests against the docparse repro fixture
- Docs updates (serve-api.md, help.go, CHANGELOG)

**Total: ~1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing deployments break on upgrade (something was being silently dropped without operators noticing) | Medium — could surface latent issues at upgrade time | The `AILANG_SERVE_API_ALLOW_DROPS=1` escape hatch lets operators upgrade then investigate. Document prominently in the v0.21.0 release notes. |
| `loaded.File.Module` is nil for unparseable files, so annotation extraction breaks | Low | Guard the extraction with a nil-check; if annotations can't be enumerated, treat as "annotations unknown" and warn rather than fatal-fail |
| Stdlib resolution under abnormal symlink layouts produces drops we'd previously ignore | Low | The `@route` gate keeps stdlib drops in the warn-not-fatal bucket; only authored-handler drops fail |
| Test fixture for the docparse layout is fragile across machines | Medium | Use `t.TempDir()`, build the layout entirely in-test, avoid relying on a real `~/.ailang/cache/` path |

## Best Practices Push (companion to this change)

This design doc only covers the AILANG-side strict failure. A complementary docs push will land alongside:

1. **Use `ensures` clauses at value boundaries** — now that M-DX26 Phase 5 ships verified `ensures`, package authors should annotate their public functions with non-emptiness contracts. Example:
   ```ailang
   pure func freeEntitlements(uid: string, catalog: List[Plan]) -> Entitlements
     ensures { result.monthlyRequestLimit > 0 }
   = ...
   ```
   This would have caught the empty-catalog path in the test suite.

2. **Prefer `Result[T, E]` over default-valued returns.** `parseCatalog: string -> Result[List[Plan], ParseError]` is strictly better than `parseCatalog: string -> List[Plan]` because the empty-list-on-failure case can't be confused with a legitimately empty catalog.

3. **Use canonical `pkg/...` imports inside published packages, not relative `./X`.** Sidesteps the dual-canonical-ID class entirely.

Tracked as a documentation update under M-DOCS-PACKAGE-AUTHOR-PRACTICES (separate doc).

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_10_12/m-serveapi-unify.md](../../implemented/v0_10_12/m-serveapi-unify.md) — the prior refactor that unified module registration around `registerModule`; this design extends it
- [design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md](../../implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md) — earlier serve-api annotation work

**Planned (related but separate):**
- M-LOADER-CANONICAL-REWRITE (not yet filed) — rewrites `./X` → canonical `pkg/...` at canonical-ID time inside declared modules, closing the dual-load class at the root
- M-DOCS-PACKAGE-AUTHOR-PRACTICES (not yet filed) — `ensures`-at-boundary, `Result`-wrapping, canonical imports

**Bug report origins:**
- `ailang messages` thread starting `e1814c9f-4620-4e3c-a051-4dd1a38d392b` (docparse → ailang-core)
- Close-out `952d6ef0` confirms commit `a629a129` as the partial fix

## References

- [Design Axioms](/docs/references/axioms) — A11 (Structured Failure) is the governing axiom for this change
- [CLAUDE.md "NO SILENT FALLBACKS"](../../../CLAUDE.md) — direct application
- [Commit a629a129](https://github.com/sunholo-data/ailang/commit/a629a129) — the partial fix this design completes

## Future Work

- **`@noexpose` drop strictness** — extend strict-mode to non-route annotations once the v0.21.0 change lands and we observe the upgrade behavior.
- **`/health` schema versioning** — once `dropped_modules` is in place, decide whether `/health` warrants a schema-version field for downstream consumers.
- **Loader-level fix (M-LOADER-CANONICAL-REWRITE)** — close the underlying dual-canonical-ID class so the under-basePath filter has nothing to drop.

---

**Document created**: 2026-05-15
**Last updated**: 2026-05-15

DESIGN_DOC_PATH: design_docs/planned/v0_21_0/m-serveapi-surface-drops.md

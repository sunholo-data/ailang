# M-SERVEAPI-SURFACE-DROPS Sprint Plan

**Sprint ID**: M-SERVEAPI-SURFACE-DROPS
**Design doc**: [m-serveapi-surface-drops.md](m-serveapi-surface-drops.md)
**Target version**: v0.21.0
**Duration**: 1 day (~7 hours)
**Risk level**: Low
**Status**: Approved, ready to execute

## Sprint Goal

Convert silent module-drop failures in `serve-api` into fail-fast startup errors. When `registerModule` skips a `@route`-bearing module because it resolved outside basePath, the server must refuse to start with a clear named-module error — not silently 200 with empty-record responses to customer traffic.

Builds on commit `a629a129` (v0.18.4), which added stderr logging for the same drop. This sprint promotes that log into:
1. A startup-banner WARN with module names
2. A fail-fast error when annotation-bearing modules are dropped
3. A `/api/v1/health` field surfacing partial registration

## Velocity Calibration

Recent comparable sprints (single-author, single-day milestones):

| Sprint | Date | LOC | Duration |
|--------|------|-----|----------|
| M-PARSER-BLOCK-TR | May 14 | ~80 | ~4h |
| M-DX26 Phase 5.2 | May 14 | ~60 | ~3h |
| M-DX26 Phase 5.1 | May 14 | ~50 | ~2h |
| M-DX26 Phase 5 | May 14 | ~200 | ~6h |
| `messages reply` fix | May 15 | 14 | ~30min |

**Velocity baseline:** ~150 LOC/day sustained, low-risk Go bugfix work in apiserver/loader code typically lands same-day. This sprint's ~250 LOC fits the pattern.

## Milestone Breakdown

### M1: Track drops in `registerModule` (~2h, ~50 LOC)

**Goal**: Instead of silently `return ("", false, nil)`, the under-basePath rejection path appends a `DroppedModule` record to a server-side slice and continues.

**Files:**
- New: `internal/apiserver/types.go` — add `DroppedModule` struct (~20 LOC)
- Modified: `internal/apiserver/server.go` — add `droppedModules []DroppedModule` field on `Server` struct (~5 LOC)
- Modified: `internal/apiserver/module_entry.go:52-69` — replace the silent return with drop-tracking call (~20 LOC delta)
- New: `internal/apiserver/module_entry_test.go` — unit test (~50 LOC)

**Implementation detail:**
```go
type DroppedModule struct {
    PhysicalPath string   // symlink-resolved abs path of dropped file
    DeclaredPath string   // module header path, e.g. "pkg/sunholo/billing_entitlements/plan"
    FileBaseName string   // for short error messages
    Annotations  []string // ["@route", "@raw", ...] — empty if no annotations
    Reason       string   // "outside-basePath" (currently the only reason)
}
```

Annotation extraction reads `loaded.File.Decls` and collects any `*ast.Decl` with non-empty `Attributes` whose name matches the known annotation set (`@route`, `@raw`, `@nowrap`, `@noexpose`, `@mcp_name`). Guard with nil-check on `loaded.File`.

**Acceptance criteria:**
- `Server.droppedModules` is populated when a module resolves outside basePath
- Annotation list correctly captures `@route` presence (the only annotation that matters for fail-fast)
- Unit test fixtures a fake `LoadedModule` with a path outside basePath, asserts the drop is recorded
- Existing apiserver tests still pass (no regression on the registration success path)

**Risk:** Low. Pure additive change to a function with clear boundaries. The hardest part is annotation extraction — keep it simple (presence check, not full attr parsing).

---

### M2: `validateRegistration` + banner + fail-fast (~3h, ~110 LOC)

**Goal**: After all modules are processed, before the HTTP listener binds, emit the banner WARN and exit non-zero if any dropped module had `@route`.

**Files:**
- Modified: `internal/apiserver/server.go` — new `validateRegistration() error` method (~70 LOC)
- Modified: `cmd/ailang/serve_api.go` — call `validateRegistration` post-`New()`, exit on error (~25 LOC)
- Modified: `cmd/ailang/help.go` — document `AILANG_SERVE_API_ALLOW_DROPS` env var (~10 LOC)

**Implementation detail:**

`validateRegistration` logic:
1. Partition `s.droppedModules` into `fatal` (has `@route` in annotations) and `warn` (everything else)
2. If `len(s.droppedModules) > 0`, log a single banner WARN line: `"⚠  Dropped N module(s) outside basePath: a, b, c"`
3. If `len(fatal) > 0 && os.Getenv("AILANG_SERVE_API_ALLOW_DROPS") != "1"`, return a multi-line error naming each fatal drop with declared path, basePath, resolved path, annotations
4. If allow-drops is set and there are fatals, log a separate STRONG WARN that the override is active

CLI integration in `cmd/ailang/serve_api.go`:
```go
srv := apiserver.New(basePath, cfg)
// ... existing load logic ...
if err := srv.ValidateRegistration(); err != nil {
    fmt.Fprintln(os.Stderr)
    fmt.Fprintln(os.Stderr, "Error: serve-api refuses to start with @route-bearing modules dropped:")
    fmt.Fprintln(os.Stderr, err.Error())
    fmt.Fprintln(os.Stderr)
    fmt.Fprintln(os.Stderr, "Resolution options:")
    fmt.Fprintln(os.Stderr, "  1. Move basePath outward (use a project root that contains all imports)")
    fmt.Fprintln(os.Stderr, "  2. Replace relative ./X imports with canonical pkg/... imports")
    fmt.Fprintln(os.Stderr, "  3. Set AILANG_SERVE_API_ALLOW_DROPS=1 to start anyway (NOT recommended)")
    os.Exit(1)
}
```

**Acceptance criteria:**
- A test scenario with a `@route`-bearing module path outside basePath causes `ValidateRegistration()` to return a non-nil error
- The error message includes file basename, declared path, basePath, and resolved path
- The CLI prints the structured help and exits 1
- With `AILANG_SERVE_API_ALLOW_DROPS=1`, the same scenario logs a STRONG WARN but returns nil
- A non-annotation-bearing drop logs the WARN but `ValidateRegistration()` returns nil

**Risk:** Low-Medium. The only judgement call is the error message format — the design doc has a worked example, follow it. `os.Exit(1)` at the CLI boundary is testable via `cmd/ailang` integration tests using `exec.Command`.

---

### M3: `/health` extension + integration tests + docs (~2h, ~85 LOC)

**Goal**: Operators on Cloud Run / Kubernetes can detect partial registration via a readiness probe before traffic hits.

**Files:**
- Modified: `internal/apiserver/handlers.go` — extend health handler with `dropped_modules` field (~15 LOC)
- New: `cmd/ailang/serve_api_drop_test.go` — integration test using `t.TempDir()` to build the docparse layout (~150 LOC)
- Modified: `docs/docs/guides/serve-api.md` — new "Strict-Mode Module Dropping" section (~40 lines prose)
- Modified: `changelogs/v0.10-current.md` — `[Unreleased]` entry under `### Fixed` (~5 lines)

**Implementation detail:**

Health JSON shape (only populated when there are drops — keep the healthy response minimal):
```json
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

Healthy response stays minimal:
```json
{"status": "ok", "modules_registered": 6}
```

Integration test fixtures via `t.TempDir()`:
- **Fixture A**: project with @route-bearing module outside basePath → `serve-api` exits 1, stderr contains module name
- **Fixture B**: same layout + `AILANG_SERVE_API_ALLOW_DROPS=1` → starts, `/health` returns `degraded` with `dropped_modules` non-empty
- **Fixture C**: only non-annotation-bearing drop → starts cleanly, banner has WARN line, `/health` is `ok`

**Acceptance criteria:**
- All three integration test fixtures pass
- `make ci` passes
- `docs/docs/guides/serve-api.md` documents the strict behavior, escape hatch, and `/health` shape
- `ailang help` shows `AILANG_SERVE_API_ALLOW_DROPS` in env var list
- CHANGELOG entry added under `[Unreleased]` referencing the docparse bug origin

**Risk:** Medium. Integration tests using `exec.Command("ailang", ...)` from `cmd/ailang/serve_api_drop_test.go` need careful port allocation (use port 0 + read from stdout, or t.Skip if `ailang` binary isn't on PATH). Fallback: use `apiserver.New(...)` + `ValidateRegistration()` directly without the binary, which is cheaper to set up but doesn't cover the os.Exit path.

---

## Day-by-Day Plan

**Day 1, AM (3h)** — M1 + M2 start
- 09:00-10:30: M1 implementation (types.go, server.go field, module_entry.go change, unit test)
- 10:30-12:00: M2 implementation start — `validateRegistration` method + partitioning logic

**Day 1, PM (4h)** — M2 finish + M3
- 13:00-14:00: M2 finish — CLI wiring in serve_api.go, error message formatting, env var support
- 14:00-15:30: M3 — `/health` extension, integration test fixtures
- 15:30-16:30: M3 — docs (serve-api.md), help.go env var, CHANGELOG entry
- 16:30-17:00: `make ci`, fix any lint, final commit

**Buffer:** 1 hour for integration-test flakes (port allocation, binary path on test machine).

---

## Pause Points for Human Input

None expected. The design doc's Design Freeze locked in:
- Strict-by-default for declared-route drops
- `AILANG_SERVE_API_ALLOW_DROPS=1` env-var-only escape hatch
- `@route` is the only annotation that triggers fail-fast

If the integration test reveals an unexpected path through `loadFile`/`registerModule` (e.g. a non-symlink-resolved file path that bypasses the drop tracking), pause and flag for human review rather than special-casing.

## Success Metrics

- [ ] All 3 integration test fixtures pass (A: fails, B: starts-degraded, C: starts-warn)
- [ ] `make ci` passes (no lint, no test regression)
- [ ] CHANGELOG entry added under `[Unreleased]` in `changelogs/v0.10-current.md`
- [ ] `docs/docs/guides/serve-api.md` updated with strict-mode section
- [ ] `ailang help` lists `AILANG_SERVE_API_ALLOW_DROPS`
- [ ] No regression on existing `internal/apiserver` test suite
- [ ] Total LOC: ~250 (within design doc estimate of 200-300)

## Example Files

This sprint touches `serve-api` plumbing rather than language features, so no `examples/*.ail` files are required. The integration test fixtures in `cmd/ailang/serve_api_drop_test.go` serve as the concrete reproduction recipe — they're the canonical example of the failure mode and the fix.

## Dependencies

None. The partial fix (`a629a129`) is already merged. This sprint extends apiserver internal types and CLI behavior; no parser/typechecker/codegen changes.

## Out of Scope

Tracked in the design doc's "Non-Goals" / "Deferred Decisions":
- `@noexpose` drop strictness (defer to follow-up)
- Loader-level fix for dual-canonical-ID class (M-LOADER-CANONICAL-REWRITE, not yet filed)
- Best-practices push for package authors (already sent to docparse as a message, separate doc track)

## Commit Plan

3 commits, one per milestone:

1. `feat(serve-api): M-SERVEAPI-SURFACE-DROPS M1 — track dropped modules in registerModule`
2. `feat(serve-api): M-SERVEAPI-SURFACE-DROPS M2 — fail-fast on @route-bearing drops`
3. `feat(serve-api): M-SERVEAPI-SURFACE-DROPS M3 — /health field + integration tests + docs`

All three commits reference the docparse bug-origin inbox `e1814c9f` in the body. No `refs #N` since no GitHub issue is associated.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_21_0/m-serveapi-surface-drops-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-SERVEAPI-SURFACE-DROPS.json`

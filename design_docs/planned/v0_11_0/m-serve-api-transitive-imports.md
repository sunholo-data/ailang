# M-SERVE-API-TRANSITIVE-IMPORTS: Fix Transitive Import Resolution in serve-api

**Status**: Planned
**Target**: v0.9.4
**Priority**: P0 (Critical — blocks DocParse DOCX/ZIP parsing via serve-api)
**Estimated**: 0.5-1 day
**Dependencies**: None
**Milestone ID**: M-SERVE-API-TRANSITIVE-IMPORTS
**Created**: 2026-03-18
**Source**: DocParse agent message `c9baaa82` (transitive import resolution fails for std/zip via zip_extract)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes nondeterministic behavior: same code works under `ailang run` but fails under `serve-api` |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Import resolution becomes consistent across execution modes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Removes a mode-dependent failure that confuses automated agents |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Multi-module composition works correctly in serve-api |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): **Fixes** mode-dependent nondeterminism
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Removes agent-confusing mode mismatch

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"serve-api ≠ ailang run" parity gap**. Every module-loading difference between the two execution modes will surface as a bug when users deploy to serve-api. This specific case is the most critical because it blocks any multi-module project with transitive stdlib dependencies (which is nearly all real projects).

**Prior art:**
- `ailang run` in `cmd/ailang/run_helpers.go:216` calls `rt.PreloadModule()` for ALL modules returned by the pipeline, including transitive dependencies
- `serve-api` in `internal/apiserver/server.go:164` only compiles each file independently and never preloads transitive modules

---

## Problem Statement

### Immediate Problem

When module A imports module B, and B imports `std/zip`, calling A's functions via serve-api fails:

```
failed to resolve global std/zip.listEntries: module std/zip not imported by docparse/services/docx_parser
```

The same code works perfectly under `ailang run`.

### Root Cause

`serve-api` and `ailang run` load modules differently:

| Step | `ailang run` | `serve-api` |
|------|-------------|-------------|
| Compile | Pipeline compiles entry + all transitive deps | Pipeline compiles each file independently |
| Preload | `PreloadModule()` for ALL modules from `result.Modules` | **Nothing** — transitive deps not preloaded |
| Runtime | Resolver finds std/zip already loaded | Resolver fails: "std/zip not imported by A" |

**The fix location:** `internal/apiserver/server.go` `loadFile()` method.

After the pipeline compiles a file, `result.Modules` contains all transitively resolved modules. serve-api ignores this map entirely. It needs to preload these modules into the embed.Engine's runtime, matching what `ailang run` does.

### Code Path Comparison

**`ailang run` (working)** — `cmd/ailang/run_helpers.go`:
```go
result, err := pipeline.RunWithContext(ctx, cfg, src)
// ...
for modPath, loaded := range result.Modules {
    rt.PreloadModule(modPath, loaded)  // ← preloads ALL transitive deps
}
```

**`serve-api` (broken)** — `internal/apiserver/server.go`:
```go
result, err := pipeline.RunWithContext(context.Background(), cfg, src)
// Extracts interface info for OpenAPI/routes
modInfo := extractModuleInfo(result.Interface)
// result.Modules is IGNORED ← the bug
```

### Impact

- **DocParse**: 19 modules, most using transitive stdlib imports. DOCX, ZIP, XML parsing all broken via serve-api.
- **Any multi-module project**: Any project where module A imports user module B which imports stdlib will fail.
- **Severity**: P0 — this makes serve-api unusable for real multi-module projects.

---

## Goals

**Primary Goal:** Make serve-api preload transitive module dependencies so multi-module import chains work identically to `ailang run`.

**Success Metrics:**
- `docx_parser → zip_extract → std/zip` chain works via serve-api
- `ailang run` and `serve-api` produce identical results for the same module
- No regression in existing single-module serve-api behavior
- Hot reload (`--watch`) still works with transitive deps

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Preload strategy: per-file or all-at-once | Affects startup time and memory for large projects | agent | compile | low |
| Dedup handling: what if two files import same transitive dep | Must not double-load or conflict | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Preload strategy: **per-file** — preload each file's transitive deps after compilation (matches current sequential loading)
- [x] Dedup: **skip if already loaded** — check runtime before preloading

---

## Solution Design

### Overview

Add transitive module preloading to `loadFile()` in `server.go`, matching the pattern from `run_helpers.go`. This is a ~10-20 LOC change.

### Implementation

**Step 1: Expose PreloadModule on embed.Engine**

The `embed.Engine` wraps a `runtime.ModuleRuntime` but doesn't expose `PreloadModule`. Add a forwarding method:

```go
// internal/embed/embed.go
func (e *Engine) PreloadModule(path string, loaded *loader.LoadedModule) {
    e.runtime.PreloadModule(path, loaded)
}
```

**Step 2: Preload transitive deps in loadFile()**

After the pipeline compiles a file, preload all modules from `result.Modules`:

```go
// internal/apiserver/server.go — loadFile()

result, err := pipeline.RunWithContext(context.Background(), cfg, src)
// ... existing error handling ...

// Preload all transitively resolved modules into the runtime.
// This ensures that when module A calls module B which uses std/zip,
// std/zip is already available in the runtime — matching ailang run behavior.
if result.Modules != nil {
    for modPath, loaded := range result.Modules {
        s.engine.PreloadModule(modPath, loaded)
    }
}

// ... existing extractModuleInfo, route extraction, etc ...
```

**Step 3: Handle deduplication**

`PreloadModule` should be idempotent — if the module is already loaded, skip it. Check if `runtime.PreloadModule` already handles this:

```go
// internal/runtime/runtime.go — PreloadModule
func (rt *ModuleRuntime) PreloadModule(path string, loaded *loader.LoadedModule) {
    // Check if already preloaded to avoid duplicate work
    if rt.preloaded[path] != nil {
        return
    }
    rt.preloaded[path] = loaded
}
```

### Files to Modify

**Modified files:**
- `internal/apiserver/server.go` (~+10 LOC) — Add preload loop in `loadFile()`
- `internal/embed/embed.go` (~+5 LOC) — Add `PreloadModule()` forwarding method

**No new files needed.**

---

## Examples

### Before (fails)

```
docx_parser.ail → imports zip_extract
zip_extract.ail → imports std/zip

serve-api loads docx_parser independently
serve-api loads zip_extract independently
Runtime: docx_parser calls zip_extract.readDocxContent()
         → zip_extract calls std/zip.readEntry()
         → FAIL: "module std/zip not imported by docx_parser"
```

### After (works)

```
serve-api loads docx_parser
  → pipeline resolves: docx_parser, zip_extract, std/zip (all in result.Modules)
  → PreloadModule for each
serve-api loads zip_extract
  → pipeline resolves: zip_extract, std/zip (already preloaded, skipped)
  → PreloadModule for each (dedup)
Runtime: docx_parser calls zip_extract.readDocxContent()
         → zip_extract calls std/zip.readEntry()
         → SUCCESS: std/zip already in runtime
```

---

## Success Criteria

- [ ] `docx_parser → zip_extract → std/zip` chain works via serve-api
- [ ] Direct call to `zip_extract` still works (no regression)
- [ ] Multiple modules importing the same transitive dep don't conflict
- [ ] Hot reload (`--watch`) correctly re-preloads transitive deps
- [ ] `ailang run` behavior unchanged
- [ ] All existing tests pass
- [ ] Lint clean

---

## Testing Strategy

**Unit tests:**
- Mock pipeline result with `result.Modules` containing transitive deps, verify preloading

**Integration tests:**
- Create test modules: A imports B, B imports C
- Serve via serve-api, call A's function that internally uses C
- Verify success (currently fails)

**Manual testing:**
- DocParse DOCX parsing via serve-api (the original repro)

---

## Deferred Decisions

- **Lazy vs eager preloading** — Current approach eagerly preloads all transitive deps. Could defer to lazy loading if startup time becomes an issue for very large projects. Agent may choose.

## Non-Goals

- **Changing AILANG's import semantics** — This is not about making imports transitive in the language. Modules still must explicitly declare their imports. The fix is about runtime module availability, not import visibility.
- **Fixing the parseDocx 0-blocks issue** — That's a separate DocParse-specific bug (likely XML parsing logic), not an import resolution issue.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Preloading all transitive deps increases memory for large projects | Low | Modules are small; DocParse's 19 modules total ~500KB compiled |
| Module version conflicts if two files import different versions of same dep | Low | AILANG has no versioning yet; same module path = same module |
| PreloadModule not idempotent | Med | Verify PreloadModule handles duplicates; add guard if needed |

---

## Related Documents

- [M-SERVE-API-DX](./m-serve-api-dx.md) — Parent design doc for serve-api production readiness
- [M-CODEGEN-API-SERVER](../../planned/v0_10_0/m-codegen-api-server.md) — Compiled API server (will need same fix)

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18

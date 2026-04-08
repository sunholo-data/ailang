# M-SERVEAPI-UNIFY: Unified Module Registration for serve-api

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (High — retires the MCP cascade class of bugs structurally)
**Estimated**: 3-4 days (~300-500 LOC refactor + tests)
**Dependencies**: None (builds on v0.10.11)
**Milestone ID**: M-SERVEAPI-UNIFY
**Created**: 2026-04-08
**Source**: Follow-up to [m-mcp-cascade-postmortem-and-fixes.md](m-mcp-cascade-postmortem-and-fixes.md) after user pushback on the "projection key matrix as living doc" coping mechanism — if we can eliminate the drift, we don't need to document it.
**Supersedes**: M4 of [m-mcp-cascade-postmortem-sprint-plan.md](m-mcp-cascade-postmortem-sprint-plan.md) (projection-key-matrix doc + empty-field grep rule)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Single registration site eliminates non-deterministic "which path writes first" races between main loadFile and dep-discovery |
| A2: Replayability | 0 | No change — compilation pipeline unchanged |
| A3: Effect Legibility | 0 | No change to effects |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +2 | Drift between projections becomes structurally impossible, not just policy-enforced — the invariant is a type, not a comment |
| A6: Safe Concurrency | +1 | One write site to `s.modules` simplifies the mu lock discipline |
| A7: Machines First | +1 | A single `ModuleEntry` struct with explicit projection fields is easier for codegen, introspection, and AI tools to reason about than "four derived strings scattered across four call sites" |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +1 | Eliminates double-compilation during eager-load (the v0.10.10 cost regression) by construction |
| A10: Composability | +1 | Projections compose from a single identity; new projections (e.g., GraphQL schema) can be added as fields without touching existing call sites |
| A11: Structured Failure | 0 | No change to error handling |
| A12: System Boundary | +1 | Makes the boundary between "pipeline-owned module graph" and "serve-api's view of local modules" explicit and one-way |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Eliminates a nondeterministic ordering bug class, doesn't add one
- [x] A5 (Bounded Verification): The invariant "every projection of a module is consistent" becomes a type-level guarantee, not a documented convention
- [x] A9 (Cost Visibility): Retires the v0.10.10 2× eager-load regression structurally

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes. This is the **eliminate-the-cascade** follow-up to [m-mcp-cascade-postmortem-and-fixes.md](m-mcp-cascade-postmortem-and-fixes.md). The postmortem identified three intertwined patterns (protocol projection drift, empty fields as load-bearing dead code, test gaps). v0.10.11 fixed the immediate regression surgically. This doc addresses the **root cause of Pattern 1** by removing the four-column drift surface entirely.

### Why the postmortem's "projection key matrix doc" isn't enough

The postmortem proposed documenting the matrix as a living checklist — every PR touching one column updates the doc. That's a coping mechanism: it catches drift in review but doesn't prevent it. Reviewers miss things. And the underlying data model still has four keys for the same module, so **any** future work in this triangle (loader / pipeline / serve-api) has to be extra careful. The cascade proved that "extra careful" isn't a reliable strategy when the subsystem is this entangled.

### The real root cause

The matrix exists because `s.modules` is keyed by a **derived string** (one of four possible derivations, depending on the call site), and because `loadFile` is a **per-file operation** that transitively compiles siblings as a side-effect. Both are artifacts of `serve-api` inheriting its loading model from `ailang run`, which only ever has one entry point. serve-api loads **whole projects**, so the per-file model mismatches.

Two structural fixes eliminate Pattern 1 entirely:

1. **Key `s.modules` by physical file path**, not a derived string. The key is `filepath.EvalSymlinks(Abs(file.Path))` — the module's identity. Everything else (rel path for URLs, canonical ID for dispatch, declared path for imports, MCP tool name) is a **projection** stored as a field on one `ModuleEntry` struct, computed once at registration.

2. **Replace per-file `loadFile`** with a project-wide single compilation pass. serve-api calls the pipeline **once** for the whole directory, iterates `result.Modules`, and registers locals in one loop. There's no dep-discovery loop because there's nothing to "discover" — everything is already in `result.Modules` from the one compilation.

Together these collapse the four-column matrix into one registration site writing one entry per module.

### What breaks in the cascade if we do this

- **v0.10.8 (route leak via module_prefix aliasing)** — the filter "is this file under basePath on disk?" becomes the ONLY filter, applied at the one registration site. No other projection can leak.
- **v0.10.9 / v0.10.10 (MOD011 dedup)** — still needed at the pipeline level for correctness of `callFunction` dispatch. Unchanged. But with one registration site, serve-api can't accidentally register the same module under two keys.
- **v0.10.10 → v0.10.11 (dep-discovery double-registration)** — **impossible by construction**. There is no dep-discovery loop.
- **Future symlink mismatches (v0.10.11's EvalSymlinks fix to absBase)** — not needed, because there's no prefix check at the registration site: the check is "does `filepath.EvalSymlinks(file.Path)` live under `filepath.EvalSymlinks(basePath)`?", done once, in one place.

---

## Problem Statement

### Current state

`internal/apiserver/server.go` has two write sites for `s.modules`:

1. **Main path** (`loadFile`, ~line 440): registers the entry-point file under `filepath.Rel(basePath, absPath)`.
2. **Dep-discovery loop** (`loadFile`, ~line 330): iterates `result.Modules` (which contains transitive dependencies of the entry point) and registers any sibling with an `@route` annotation.

Both paths exist because `loadFile` is called **once per .ail file** via `filepath.Walk`, and each call produces a `result.Modules` map that contains the file's transitive imports. When serve-api walks a 5-file project where each file imports the next, each `loadFile` invocation produces an overlapping `result.Modules`, and the dep-discovery loop exists to avoid losing sibling routes.

This creates the four-column projection drift documented in the postmortem. It also creates subtle ordering dependencies: whichever `loadFile` call first encounters a module via dep-discovery "wins" the registration, and the main path of the later `loadFile` for that same file then has to not-clobber (or clobber-with-the-same-content) the existing entry.

### Concrete symptoms (from the v0.10.7 → v0.10.11 cascade)

| Release | Symptom | Root cause |
|---------|---------|------------|
| v0.10.8 | `module_prefix`-aliased package routes leaked into serve-api | Dep-discovery filter used module path, not file path |
| v0.10.9 | MOD011 false-fired on every aliased package | Collision check dedup-keyed by canonical ID, not file path |
| v0.10.10 | Cold start doubled from ~22s → ~90s | Populating `ast.File.Path` reanimated dep-discovery, which used a different key than main path → duplicate registration → 2× eager-load |
| v0.10.11 | macOS `/var/folders` symlink mismatch silently dropped local files | absBase vs loader-resolved file paths had inconsistent symlink normalization |

All four have the same shape: **two paths deriving the same "logical" key in different ways, drifting apart.**

### What the code actually needs

A single `s.modules` keyed by module **identity** (physical file path), with all projections computed **once** at a **single registration site**. Everything serve-api does downstream (HTTP routes, OpenAPI spec, MCP tools, A2A cards, function dispatch) reads from that one source.

---

## Proposed Design

### New type: `ModuleEntry`

```go
// ModuleEntry is the unified representation of a local module in serve-api.
// It is the SOLE source of truth for module-level data; every projection
// (HTTP routes, OpenAPI spec, MCP tool list, function dispatch) reads from
// a ModuleEntry, never from a separately-derived string or parallel map.
type ModuleEntry struct {
    // Identity — keyed by this in s.modules. Computed via
    // filepath.EvalSymlinks(filepath.Abs(file.Path)) at registration.
    // Unique per physical file, stable across runs.
    PhysicalPath string

    // Projections — all derived from Identity + pipeline output, once,
    // at registration. Read-only after Register() returns.

    // CanonicalID is the pipeline's canonical module ID (used by the
    // loader cache and by callFunction dispatch). Example:
    // "docparse/services/mcp_tools".
    CanonicalID string

    // DeclaredPath is the `module X` header as written in the source
    // file. Used to resolve imports from other local modules. Usually
    // equals CanonicalID, but differs under module_prefix aliasing.
    DeclaredPath string

    // RelPath is the forward-slash relative path from basePath, with
    // .ail stripped. Used to construct HTTP URLs and as the stable
    // human-readable identifier in logs and the startup banner.
    // Example: "services/mcp_tools".
    RelPath string

    // Exports carries per-function metadata: route annotations, param
    // names, doc comments, @mcp_name overrides, etc. Populated once at
    // registration via the extract* family.
    Exports []ExportInfo

    // Iface is the type-checked interface from the pipeline. Used by
    // callFunction for signature lookup and by the MCP schema generator
    // for JSON Schema derivation.
    Iface *iface.Iface

    // File is the parsed AST. Used for doc comment extraction and by
    // extract* during registration. May be nil after eager-load GC.
    File *ast.File
}
```

### New API: `Server.registerModule`

```go
// registerModule is the SOLE write site for s.modules. All other code
// paths that want to add a module go through here. Idempotent: a repeat
// call with the same PhysicalPath is a no-op (or an error if the content
// differs — TBD, see Open Questions).
func (s *Server) registerModule(loaded *loader.LoadedModule) error {
    if loaded.File == nil || loaded.File.Path == "" {
        return nil // no physical file → not a local module
    }

    // Compute identity
    physical, err := filepath.EvalSymlinks(filepath.Clean(loaded.File.Path))
    if err != nil {
        return nil // unreadable → skip, not our concern
    }
    physical, _ = filepath.Abs(physical)

    // Under-basePath check (the only filter)
    if !strings.HasPrefix(physical+string(filepath.Separator), s.normalizedBasePath) &&
       physical != strings.TrimSuffix(s.normalizedBasePath, string(filepath.Separator)) {
        return nil // package file, not local
    }

    s.mu.Lock()
    defer s.mu.Unlock()
    if _, exists := s.modules[physical]; exists {
        return nil // idempotent
    }

    // Compute ALL projections, once
    relPath, _ := filepath.Rel(s.normalizedBasePath, physical)
    entry := &ModuleEntry{
        PhysicalPath: physical,
        CanonicalID:  loaded.Path, // pipeline's canonical ID
        DeclaredPath: loaded.File.ModulePath, // `module X` header
        RelPath:      filepath.ToSlash(strings.TrimSuffix(relPath, ".ail")),
        Iface:        loaded.Iface,
        File:         loaded.File,
    }

    // Populate Exports once (replaces the 4 extract* calls scattered across
    // main path + dep-discovery loop)
    entry.Exports = buildExports(loaded.Iface, loaded.File)

    s.modules[physical] = entry
    log.Printf("  Registered: %s (%d exports)", entry.RelPath, len(entry.Exports))
    return nil
}
```

### New entry point: `Server.LoadProject`

```go
// LoadProject replaces the current per-file LoadModules / loadDirectory /
// loadFile chain. It compiles the whole project in ONE pipeline pass, then
// iterates the resulting module graph and registers every local module via
// registerModule. No dep-discovery loop. No per-file dedup.
func (s *Server) LoadProject(ctx context.Context) error {
    // Synthesize a project entry point: a virtual file that imports every
    // root .ail in the basePath. "Root" means: not transitively imported by
    // another .ail under basePath. For most projects this is just main.ail;
    // for projects with multiple independent @route-bearing files, it's all
    // of them.
    //
    // Alternative: extend the pipeline with a RunDirectory entry point.
    // See "Alternatives Considered" below.
    roots, err := s.findProjectRoots()
    if err != nil {
        return err
    }

    // Compile each root; union the result.Modules maps. Because the
    // pipeline's module loader is content-addressed by canonical ID, the
    // same physical file is loaded at most once across all roots — no
    // duplicate compilation work, unlike the current per-file walk.
    seen := make(map[string]*loader.LoadedModule)
    for _, root := range roots {
        result, err := pipeline.RunWithContext(ctx, s.pipelineCfg, pipeline.Source{
            Filename: root,
        })
        if err != nil {
            return fmt.Errorf("compilation error for %s: %w", root, err)
        }
        if len(result.Errors) > 0 {
            return fmt.Errorf("compilation errors for %s: %v", root, result.Errors)
        }
        for modID, loaded := range result.Modules {
            if _, ok := seen[modID]; !ok {
                seen[modID] = loaded
                s.engine.PreloadModule(modID, loaded)
                if loaded.Path != "" && loaded.Path != modID {
                    s.engine.PreloadModule(loaded.Path, loaded)
                }
            }
        }
    }

    // Register every seen module (registerModule filters non-local files
    // internally via the under-basePath check)
    for _, loaded := range seen {
        if err := s.registerModule(loaded); err != nil {
            return err
        }
    }
    return nil
}

// findProjectRoots returns the entry points needed to cover the project's
// module graph. For a project with no isolated islands (the common case),
// this is typically 1 file. For projects with multiple disconnected
// subgraphs, it's one per subgraph.
func (s *Server) findProjectRoots() ([]string, error) {
    // Strategy: walk all .ail files, parse headers only (cheap), build the
    // import graph, return the nodes with in-degree zero. Fall back to
    // "every .ail file" if parse fails (safe superset).
    // ...
}
```

### Projections become pure read functions

```go
// Every downstream consumer iterates s.modules and projects the field it
// needs. No parallel maps, no separate derivation, no drift possible.

func (s *Server) httpRoutes() []RouteSpec {
    var routes []RouteSpec
    for _, entry := range s.modules {
        for _, exp := range entry.Exports {
            if exp.RoutePath != "" && isExposed(exp) {
                routes = append(routes, RouteSpec{
                    URL:    "/" + entry.RelPath + "/" + exp.Name,
                    Module: entry.CanonicalID, // for dispatch
                    Func:   exp.Name,
                })
            }
        }
    }
    return routes
}

func (s *Server) mcpTools() []MCPTool { /* same pattern */ }
func (s *Server) openAPISpec() *openapi.Spec { /* same pattern */ }
func (s *Server) a2aCard() *a2a.Card { /* same pattern */ }
```

### Deletion list

Removing the current drift surface deletes a meaningful amount of code:

- [internal/apiserver/server.go:323-410](internal/apiserver/server.go#L323-L410) — the entire dep-discovery loop (~90 LOC)
- [internal/apiserver/server.go:250-260](internal/apiserver/server.go#L250-L260) — `loadDirectory`'s filepath.Walk (~10 LOC)
- [internal/apiserver/server.go:262-410](internal/apiserver/server.go#L262-L410) — `loadFile` shrinks to a thin wrapper over the new project-wide path or is deleted entirely (~150 LOC → 0)
- The `normalizedBasePath` `EvalSymlinks` dance at [server.go:317-330](internal/apiserver/server.go#L317-L330) — still needed once at `New()`, but no longer duplicated at absBase computation sites

Estimated deletion: ~250 LOC. Estimated new code: ~400 LOC (ModuleEntry type + LoadProject + findProjectRoots + registerModule + projection helpers + tests). **Net: +150 LOC for a structurally safer design.**

---

## Alternatives Considered

### Alt 1 — New pipeline entry point `RunDirectory`

Instead of synthesizing roots in serve-api, add `pipeline.RunDirectory(ctx, cfg, dir) (Result, error)` that walks the directory and unions module graphs. This moves the "find roots" logic into the pipeline.

**Pros:** Reusable for other consumers (REPL, `ailang check`, future tooling).
**Cons:** Bigger API surface change. The pipeline currently has one entry-point model; breaking that is scope creep. serve-api is the only current consumer that needs whole-project loading.

**Decision:** Start with serve-api-local `findProjectRoots`. If a second consumer appears, promote to pipeline.

### Alt 2 — Compile each `.ail` separately, dedup in serve-api

Keep `loadFile` per-file, but write the dedup check in terms of `filepath.EvalSymlinks` physical-path keys instead of derived strings.

**Pros:** Minimal code churn — essentially what v0.10.11 already does, plus the ModuleEntry wrapper.
**Cons:** Retains the dep-discovery loop. Retains the "two write sites" structural flaw. Doesn't retire the class of bug; just papers over the current instance. The postmortem user feedback specifically objected to this approach as coping-not-fixing.

**Decision:** Rejected. v0.10.11 is already this option. The point of M-SERVEAPI-UNIFY is to do better.

### Alt 3 — Project-wide compilation via synthetic entry file

Generate a temporary `__serve_api_project__.ail` that `import`s every `.ail` in the project, then call the pipeline once with that file as the entry point.

**Pros:** Zero new pipeline API — uses the existing single-entry-point model. One compilation invocation total.
**Cons:** Requires every project file to be importable by the synthetic entry, which assumes all files declare valid module headers (they should, but edge cases exist). Requires temp file creation. Generates cache pollution under the synthetic module name.

**Decision:** Attractive but fragile. Prefer "find real roots" (Alt 0, the chosen design) since it uses the project's actual structure.

### Alt 4 — Matrix-as-living-doc (from the postmortem M4)

Document the four-column projection matrix; require every PR to update it.

**Pros:** No refactoring cost.
**Cons:** Doesn't prevent drift, just catches it in review. User explicitly rejected this as coping. The cascade itself is evidence that review discipline alone isn't enough for this subsystem.

**Decision:** Superseded by this doc. If M-SERVEAPI-UNIFY ships, the matrix is moot.

---

## Risks

### R1 — `findProjectRoots` misidentifies roots and misses modules

**Likelihood:** Medium
**Impact:** Silent: a local `.ail` file with no importers wouldn't be compiled → its routes missing from the server.
**Mitigation:**
- Fallback strategy: if the computed root set doesn't cover every `.ail` on disk (i.e., there are "orphan" files not reached by any root's transitive closure), compile the orphans as additional roots.
- Test: fixture with a fully disconnected `.ail` (no importers, not imported) — assert it's registered.

### R2 — Pipeline doesn't currently handle "compile this directory" semantics

**Likelihood:** Low
**Impact:** May need minor pipeline adjustments if `findProjectRoots` returns N roots and we call `RunWithContext` N times. Current caching should handle this but isn't tested for it.
**Mitigation:** Add a test that compiles the same project via N different entry points and asserts identical `result.Modules` content (canonical IDs stable).

### R3 — `module_prefix` aliasing edge cases

**Likelihood:** Medium
**Impact:** A local file can declare `module docparse/services/api_keys` while a `module_prefix`-aliased package file ALSO claims that declared path. MOD011 correctly detects this and errors out — but registerModule must not dedupe on `DeclaredPath` (that's the whole point of MOD011), only on `PhysicalPath`.
**Mitigation:** Explicit test: fixture with a local file and an aliased package file both declaring the same module header. Assert MOD011 fires before registerModule sees either one.

### R4 — Eager-load behavior change

**Likelihood:** Medium
**Impact:** Current `LoadModules` tail calls `s.engine.Load(modPath)` for every registered module. With `LoadProject`, we want the same eager-load guarantee but without the v0.10.10 doubling bug. Straightforward (iterate `s.modules` once, call `engine.Load` once per entry), but needs a test.
**Mitigation:** Instrument the engine with a counter during the test. Assert `engine.Load` is called exactly once per local module.

### R5 — Refactor disrupts docparse or other downstream projects

**Likelihood:** Low
**Impact:** LoadModules is a public API of the apiserver package. External callers may break.
**Mitigation:**
- Keep `LoadModules` as a thin wrapper: `return s.LoadProject(ctx)`, ignoring the old argument. This preserves the caller contract.
- Audit `grep -r LoadModules` before merging.

---

## Implementation Plan

### M1 — ModuleEntry type + registerModule (~100 LOC)

- Define `ModuleEntry` struct in a new file `internal/apiserver/module_entry.go`.
- Migrate `ModuleInfo` → `ModuleEntry` or make one a type alias of the other during the transition. Field-for-field compatible where possible.
- Implement `registerModule` as the sole write site.
- Unit tests: identity computation, idempotency, under-basePath filter, projection field correctness.

**Acceptance:**
- [ ] `registerModule` unit tests pass
- [ ] `make lint` clean
- [ ] Existing apiserver tests still pass (with `ModuleInfo` kept as alias during transition)

### M2 — findProjectRoots + LoadProject (~150 LOC)

- Implement `findProjectRoots` with the in-degree-zero strategy and orphan-file fallback.
- Implement `LoadProject` as the new entry point.
- Migrate `LoadModules` to a thin wrapper over `LoadProject`.
- Delete `loadDirectory` and `loadFile` (or shrink to stubs that error out with a migration hint).

**Acceptance:**
- [ ] `LoadProject` compiles every `.ail` in a fixture project exactly once (instrumented test)
- [ ] Orphan-file fallback test: disconnected `.ail` is still registered
- [ ] `module_prefix` aliasing test: aliased package files are NOT registered (under-basePath check)
- [ ] MOD011 integration test: collision still detected, before registerModule sees either file

### M3 — Retire the dep-discovery loop (~50 LOC deletion)

- Delete the 90-line dep-discovery block from `server.go`.
- Delete the absBase `EvalSymlinks` dance (no longer needed — basePath normalization at `New()` is the only site).
- Update projection consumers (`httpRoutes`, `mcpTools`, `openAPISpec`, `a2aCard`) to read `ModuleEntry` fields instead of re-deriving.

**Acceptance:**
- [ ] `grep -c "dep-discovery" internal/apiserver/` returns 0
- [ ] Cold-start regression tests from v0.10.11 (`cold_start_test.go`) still pass
- [ ] New test: assert `s.modules` keys are physical file paths (absolute, symlink-resolved)

### M4 — docparse-shape CI fixture (~100 LOC fixture + test)

Absorbed from the cascade sprint plan's M4. Now this fixture tests the unified loader, not the old drift-prone one.

- Create `tests/fixtures/projects/docparse_shape/`:
  ```
  docparse_shape/
  ├── ailang.toml         (depends on a module_prefix package)
  ├── main.ail
  ├── services/
  │   ├── api_keys.ail    (declares module docparse/services/api_keys)
  │   ├── mcp_tools.ail   (imports the package, has @route)
  │   └── csv_parser.ail  (helper)
  └── pkg/                (vendored package with module_prefix = "docparse")
      └── ...
  ```
- Test: `LoadProject(fixture)` produces exactly N expected entries, no MOD011, no duplicate keys, completes in <5s on CI hardware.
- Makefile target: `make bench-cold-start` times LoadProject on the fixture and asserts <5s. Hook into release-manager pre-flight.

**Acceptance:**
- [ ] Fixture loads cleanly
- [ ] `make bench-cold-start` passes
- [ ] Target integrated into `.claude/skills/release-manager/scripts/pre_release_checks.sh`

---

## Testing Strategy

### Unit tests (`internal/apiserver/`)

- `module_entry_test.go` — ModuleEntry construction, projection field correctness
- `load_project_test.go` — findProjectRoots correctness (linear chain, disconnected islands, single file, orphans)
- Existing `cold_start_test.go` (v0.10.11) — retained and updated to reference the new API. The 4 invariants are still the acceptance criteria; the implementation changes underneath.

### Integration tests

- `tests/fixtures/projects/docparse_shape/` + loader test (M4 above)
- macOS symlink test — assert `filepath.EvalSymlinks` is applied consistently (regression for v0.10.11's absBase fix)

### Regression proof

- Run the v0.10.11 `cold_start_test.go` suite against the M3 post-refactor code. All 4 tests must still pass.
- Run the M4 `docparse_shape` fixture against M3 code. Cold-start time must be ≤ v0.10.11 (same or faster).

---

## Open Questions

1. **Idempotency semantics for `registerModule` on content mismatch.** If two calls arrive with the same `PhysicalPath` but different `Iface` content (hot-reload scenario?), should the second call error or replace? Lean: **replace** for watch-mode, **error** for cold-start. Gate on `s.watch`.

2. **Does the pipeline need a new `RunProject` API, or is serve-api's `findProjectRoots` good enough?** Lean: local-first, promote later if a second consumer appears.

3. **What about `ailang run` on a directory?** `ailang run` currently takes a single file. If `LoadProject` generalizes cleanly, it could also be the basis for `ailang run <dir>` in a future release. Out of scope for M-SERVEAPI-UNIFY, but the API should be designed to allow it without refactoring again.

4. **Should `ModuleInfo` be deleted or kept as a deprecated alias?** Lean: alias during the transition, delete in a v0.12.0 cleanup sprint once all downstream projects confirm migration.

---

## Success Metrics

- [ ] Dep-discovery loop deleted (grep returns 0)
- [ ] `s.modules` keyed by physical file path, never by derived string
- [ ] All v0.10.11 `cold_start_test.go` invariants still hold
- [ ] `docparse_shape` fixture cold-starts in <5s on CI
- [ ] Net LOC change: +150 (acceptable for the structural safety gain)
- [ ] Zero "projection key drift" PRs required in the 90 days after release (measured by postmortem tag in CHANGELOG)
- [ ] The projection-key-matrix doc from the cascade sprint plan is **not created** (its function is absorbed by the ModuleEntry type)

---

## References

- [m-mcp-cascade-postmortem-and-fixes.md](m-mcp-cascade-postmortem-and-fixes.md) — the postmortem this supersedes M4 of
- [m-mcp-cascade-postmortem-sprint-plan.md](m-mcp-cascade-postmortem-sprint-plan.md) — sprint plan whose M4 is superseded
- [v0.10.11 changelog entry](../../../changelogs/v0.9-current.md) — the surgical fix this retires structurally
- [internal/apiserver/server.go](../../../internal/apiserver/server.go) — the file most affected
- [internal/apiserver/cold_start_test.go](../../../internal/apiserver/cold_start_test.go) — regression tests that must keep passing
- [.claude/rules/api-server.md](../../../.claude/rules/api-server.md) — rules file that will be updated to reference the new design instead of the matrix doc

---

## Notes

- This doc is the **"fix the root cause" follow-up** to v0.10.11's surgical patch. v0.10.11 stopped the bleeding; M-SERVEAPI-UNIFY retires the entire class of bug.
- If this ships, the cascade sprint plan's M4 (projection-key-matrix doc + empty-field grep rule) becomes unnecessary. The matrix is only worth writing if the drift is permanent; if we eliminate the drift, the doc is busywork.
- The "empty-field grep rule" (from the postmortem's Change 4) is still valuable as a general principle but doesn't need a separate doc — fold it into `.claude/rules/coding-standards.md` as a one-liner.
- Size is bigger than a typical bug fix because it's a refactor, but it's bounded: ~3-4 days, no API breakage if `LoadModules` stays as a wrapper, deletes more than it adds structurally.

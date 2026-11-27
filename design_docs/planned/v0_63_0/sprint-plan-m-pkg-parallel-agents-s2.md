# Sprint Plan: M-PKG-PARALLEL-AGENTS Sprint 2 (Phase 2: re-exports)

**Design doc:** [m-pkg-parallel-agents.md](m-pkg-parallel-agents.md), component C1
**Sprint JSON:** `.ailang/state/sprints/sprint_M-PKG-PARALLEL-AGENTS-S2.json`
**Target:** v0.63.0
**Created:** 2026-11-25
**AILANG prompt version loaded:** v0.16.20

## Placement and Human-Decision Rules

- No open human decisions. D1–D4 and D7 were ratified on 2026-11-25. C1's shape (`export import M (names) [@deprecated("path")]`, MOD016, removing MOD003, the facade ceiling) is fixed by the design doc.
- Mechanism choices below are agent-resolved and recorded as rulings.

## Summary

`export import lib/color (Color, Red)` makes a module a **facade**: the listed names become its exports with the original's identity. This lets a module move without breaking its dependents (Sprint 4 builds `pkg move` on it).

## Mechanism (ruling, agent-resolved)

Identity is preserved by **rewriting at load time**, not by aliasing in each later stage.
- After `LoadAll` has loaded every module, each selective import of a re-exported name is rewritten to the name's origin module, following chains of facades.
- The typechecker, linker, elaborator, nominal identity, effect reachability, the evaluator and the VM therefore all see the original module. Engine parity follows from this, and is still tested.

Why not alias in each stage: the pipeline tags imported types and constructors with `depIface.Module` (`pipeline_module_imports.go`). For a re-exported name that would be the facade, which breaks nominal identity in every consumer.

Rules of the rewrite:
- An importer keeps the facade as a dependency edge (`LoadedModule.Imports`). The facade's interface digest covers its re-export list, so the closure-based compile-cache key (`cache_dep_closure.go`) still misses when the list changes.
- The rewritten `ImportDecl` records the facade it came through (`ViaModule`) and the line's `@deprecated` path (`MovedTo`), for diagnostics and the "moved" warning.
- Module-alias access (`import facade as F`, then `F.name` or `F.Color`) is handled in the alias binding (M3). Re-exported entries carry their origin, so `F.name` and `F.Color` bind the original identity.

## Proposed Milestones

### M1 PARSE: `export import` syntax ✅
- In the parser's header loop: `EXPORT` + `IMPORT` (one token of lookahead) parses an import with `Reexport = true` and keeps looping. A trailing `@deprecated("path")` sets `MovedTo`.
- `export import` requires a symbol list. `export import M` alone, or `export import M as A`, is a parse error that names the selective form.
- An annotation before `export import` stays `PAR_ATTR_REQUIRES_FUNC`. `PAR_EXPORT_REQUIRES_FUNC` lists `import` among the accepted followers.
- The formatter round-trips the line.
- Acceptance:
  - [x] Parser tests for the form, its errors, and the trailing `@deprecated`.
  - [x] The Conflict Surface fixtures still parse unchanged: `examples/wasm_friendly_patterns.ail`, `examples/progress_bar.ail`, `std/json.ail`, `std/sem.ail`, `std/ai.ail`, `std/array.ail`, `std/bytes.ail`.
  - [x] `ailang fmt` keeps the line byte-identical.
- ~120 LOC.

### M2 RESOLVE: load-time rewrite, MOD016, IMP010 ✅
- Add `LoadedModule.Reexports`, a map from name to `{From, Orig, MovedTo}` built from the facade's `Reexport` imports.
- `LoadAll` rewrites each module's selective imports of re-exported names to the origin. It splits an import that mixes the facade's own names with re-exported ones, and keeps symbol aliases (`Red as R`).
- A re-export cycle is **MOD016** and names the cycle. Re-exporting a name the source does not export is the existing IMP010, raised by the facade's own import.
- Acceptance:
  - [x] TST-PKGPAR-06: a dependent imports `Color` and `Red` from a facade, constructs `Red`, and matches it against `lib/color.Red` and against a value built in `lib/color`. The match succeeds on the evaluator and the strict VM (today this is IMP010).
  - [x] TST-PKGPAR-09: a re-export cycle is MOD016; re-exporting a missing name is IMP010.
  - [x] Cross-package: package `app` depends only on `facade`, and `facade` re-exports from package `core`. The import resolves, and app needs no direct dependency on core.
  - [x] Cache: the facade drops a re-export, and a cached dependent recompiles and fails (no stale success).
- ~220 LOC.

### M3 IFACE: the facade's interface, hash v2 and alias access ✅
- The facade's interface includes each re-exported item as an ordinary entry, with the original's type, effects, contracts and deprecation, and the origin recorded.
- So its signature lines equal the lines the name had at its origin, and a move followed by a facade measures as no removal (the design's move releases as B). Removing a re-export is C.
- `import facade as F` binds `F.name` and `F.Color` with the origin's identity (qualified constructors such as `F.Red` are not reachable through any module alias today).
- Acceptance:
  - [x] Unit: a re-exported function's, type's and constructor's signature lines under the facade equal the origin's lines with the module renamed.
  - [x] Unit: dropping a re-export from the facade's set classifies as C.
  - [x] Integration: through `import facade as F`, `F.name`, `F.mk` and `F.Color` bind the origin's identity, and a value from `F.mk()` matches the origin's constructors. (`F.Red` itself is not reachable: qualified constructor access through a module alias is IMP010 for every module, facade or not. This is an existing limitation, recorded as out of scope.)
- ~150 LOC.

### M4 MOVED: the "moved" variant of `TC_DEPRECATED_001` ✅
- Importing a name through a facade line marked `@deprecated("pkg/new/path")` warns `TC_DEPRECATED_001`, with the fix "import X from pkg/new/path".
- This applies to functions, types and constructors. A re-export line without `@deprecated` does not warn.
- Acceptance:
  - [x] Unit tests for each of the three kinds, the fix text, and the no-warning case.
  - [x] `internal/pipeline/warn_deprecated_test.go` is unchanged and green.
- ~100 LOC.

### M5 CEILING: the facade's effect ceiling covers re-exports ✅
- A facade package's `[effects].max` must cover the effects of every function it re-exports. The violation names the re-exported function and its origin.
- Acceptance:
  - [x] TST-PKGPAR-16: a facade with `max = []` that re-exports a `! {Net}` function fails the ceiling check, naming the function.
  - [x] `effect_ceiling_reachable_test.go` "re-export shape" is unchanged and green.
- ~60 LOC.

### M6 CLOSE: codes, docs, parity ✅
- Remove MOD003; register MOD016.
- Document `export import` in `docs/docs/reference/language-syntax.md` and `docs/docs/guides/package-publishing.md`. Add a changelog fragment and the example `examples/runnable/reexport_facade/`, entered in the manifest.
- The teaching prompt is not changed in this sprint: re-exports are a package-author feature, and prompt changes go through the prompt process.
- Acceptance:
  - [x] `make test-changed`, `make lint`, `make check-file-sizes`, `make check-changelog`, `make verify-stdlib`, `make verify-examples`.
  - [x] The C1 tests pass on the evaluator and the strict VM.
- ~80 LOC plus docs.

## Registry Reuse Audit
- All milestones: `none`. These are compiler internals, and no registry package provides them.

## AILANG Syntax Gate
- The fixtures are test modules, not showcase modules. The example is the one showcase.
  - **Contracts:** include. Its re-exported function carries a `requires`, to show that contracts travel with the name.
  - **Effects:** include. `! {IO}` on `main`.
  - **Inline tests:** include, on the origin function.

## Later sprints
Unchanged; see [sprint-plan-m-pkg-parallel-agents.md](sprint-plan-m-pkg-parallel-agents.md#later-sprints-planned-when-reached).

## Outcome (2026-11-25)

All six milestones pass. Deviations from the plan:
- **M3:** re-exported entries live in a separate `Iface.Reexported` table, never in the facade's `Exports`/`Types`/`Constructors`. Those maps mean "declared here", and a dozen passes walk every interface's constructors to build ADT identity, so a copied constructor would be a phantom second ADT. Only the rendered interface, the digest, the compile cache and module-alias binding read the table.
- **M4:** the moved warning is import-based (one per name in the import list), not use-based. Types and constructors have no Core use to key on, and the loader's rewrite records the facade line on every rewritten import.
- **M6:** `make sync-examples-manifest` fails locally (`github.com/buger/jsonparser` is not a module dependency). The manifest statistics and `modules` entries were set to the values `make verify-examples` computes.
- **Found, not fixed:** qualified constructor access through a module alias (`import M as C`, `C.Red`) is IMP010 for every module.

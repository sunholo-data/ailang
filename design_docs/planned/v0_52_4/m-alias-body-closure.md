# M-ALIAS-BODY-CLOSURE: interface alias bodies and schemes must be closed over their defining module

**Status**: Implementation complete; independent sprint evaluation passed (95/100)
**Target**: Next unreleased patch (v0.52.4 artifact lineage)
**Priority**: P0 (a silent wrong type rejects valid programs; hit in production code in stapledons-godot-trappist)
**Estimated**: 3–4 days
**Dependencies**: M-TYPE-NAME-SHADOW M1+M3 (shipped in v0.44.x). **This doc specifies and schedules M2 of [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md)**, which that doc deferred as "doc only, unscheduled", and extends its scope with a trigger class that doc did not cover (see Problem Statement, T3).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the meaning of a name inside an exported type depends on the IMPORTER's import list and import ORDER (T2 below). After closure it depends only on the defining module's source. |
| A2: Replayability | 0 | The cache key is bumped (v5→v6) so no verdict is replayed across the change; behavior within one binary version is unchanged. |
| A3: Effect Legibility | 0 | Effects are not involved. |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | +1 | A module's types no longer depend on names its importers happen to bind; the defining module alone determines its exported types. |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | AI-written code routinely imports same-named `Item`/`Row`/`Planet`/`Config` types from sibling modules; today that yields a misleading "record field mismatch" far from the cause. Closure makes the valid program work; the error class disappears. |
| A8: Minimal Syntax | +1 | No new syntax; this removes an implicit, unobservable name-channel. |
| A9: Cost Visibility | 0 | Closure is a one-time per-module build step over already-built types. |
| A10: Composability | +1 | Two packages can each export `type Item` and be used together in one importer without renaming — the workaround (renaming `Planet` to `Exoplanet`) is no longer needed. |
| A11: Structured Failure | +1 | The wrong-type failure disappears; the residual nominal cases keep the existing coded TC_TYPE_SHADOW_001. |
| A12: System Boundary | +1 | Module boundaries become real boundaries for name resolution: no type crossing an interface silently depends on the receiver's scope. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): resolution becomes defining-module-determined; the only remaining order dependence (bulk first-wins for names the importer itself writes) is documented as a residual and unchanged.
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): removes a failure mode that specifically punishes machine-generated code that legitimately reuses type names.

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

Coordinator report (task-d0fe667d, AILANG v0.52.0 bf2436a, reproduced on dev v0.52.3 0cbb8cc): module `a` exports `type Item = {x: int, w: int}` and `type Box = {items: [Item]}`; module `b` exports an unrelated `type Item = {y: string}`. When `main` imports `Item` from `b` (by symbol) and calls `a`'s `count(mk())`, compilation fails:

```
type error in main (decl 0): type unification failed at [function application at main.ail:6:44]:
failed to unify parameter 0: failed to unify record field 'items':
record field mismatch: expected 1 fields, got 2
  expected fields: {y}
  actual fields:   {w, x}
```

`Box.items` must be `a`'s `Item`; `b`'s `Item` is unrelated and `main` never uses it in the failing expression. Removing `Item` from main's import of `b` makes the program pass. Real-world hit: stapledons-godot-trappist sim — a test importing `sim/data/trappist1`'s type `Planet` while calling `navigation.trappistFacts(solSystem())`, whose `System.planets` is `data/sol`'s `Planet`. Workaround in that tree: rename the local type to `Exoplanet`.

**Current State — three verified triggers, none of them rare:**

All three reproduce on v0.52.3 (dev, 0cbb8cc) with a three-file package (`a.ail`, `b.ail`, importing module) under `ailang run --package-dir . -args-json '1' --entry run <mod>.ail`.

- **T1 — explicit symbol import (the report).** `import ./a (Box, mk, count); import ./b (Item, one)` → type error above. The mere presence of `Item` in main's import list breaks `count(mk())`, which never mentions `Item`.
- **T2 — import order alone.** `import ./b (one); import ./a (Box, mk, count)` (note: `Item` is NOT imported at all) → the same type error. The bulk alias auto-import is first-wins in import-declaration order, so `b`'s `Item` occupies main's alias env merely because `b` is imported first.
- **T3 — annotation schemes, not just alias bodies.** Module `c`: `export type Item = {x: int, w: int}; export pure func mkItem() -> Item = {x: 1, w: 2}; export pure func countItem(i: Item) -> int = ...`. `probe4`: `import ./c (mkItem, countItem); import ./b (Item, one); countItem(mkItem())` → same error shape (`expected {y}, actual {w, x}`). `countItem`'s parameter crosses the boundary as a bare `TCon{Name: "Item"}` inside the exported *function scheme*, so closing only the alias **bodies** (the parent doc's M2 sketch) does not fix this class.

**The asymmetry that makes this a defect, not a policy:** if main *declares* its own `type Item`, the M1 capture guard fires loudly:

```
TC_TYPE_SHADOW_001: type Box (from module pkg/local/repro/a) refers to a type named Item,
but module probe2 declares its own type Item, which shadows the Item that pkg/local/repro/a
means — expanding Box here would silently use the wrong Item.
```

The same name arriving via an import (T1) or via import order (T2) gets **no** diagnostic — it silently replaces the name inside another module's type. M1 guarded "names the importer declares"; it did not guard "names the importer merely has in scope".

**Impact:** every multi-module program that reuses a common type name (`Item`, `Row`, `Planet`, `Config`) across modules — exactly the names AI codegen reaches for — fails with an error that names neither the colliding module nor the type that was actually expanded wrong, or worse, could silently accept wrong code (the parent doc's `LocalStillTypeChecks` soundness hole).

## Goals

**Primary Goal:** a module's exported types (alias bodies, function schemes, constructor types) must mean the same thing in every importer, regardless of what the importer has in scope.

**Success Metrics:**
- The T1/T2/T3 repros compile and run (return 1) without changing the import list or renaming types.
- The stapledons-godot shape (4 modules: navigation / data-sol / data-trappist1 / test) typechecks without the `Exoplanet` rename.
- `TestTypeNameShadow_CapturedImportedAliasIsLoud` flips to a positive test (the captured-alias program now works), per the parent doc's M2 prediction.
- All existing alias/shadow/cache tests pass unchanged except the deliberate flips listed in Conflict Surface.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Interfaces are **closed**: no alias-name `TCon` crosses a module boundary unresolved; it is expanded in the defining module at interface build time | This is the semantic invariant the whole fix hangs on; the alternative (origin-tagged alias envs) is M4, a type-representation change | compiler | design | med |
| Closure applies to export **schemes** and constructor types, not just `TypeAliases` bodies | T3 proves bodies-only closure leaves the report's class half-fixed | agent | design | med |
| Closure runs at interface build in the defining module (not at import time in each importer) | One place, cached, benefits REPL/WASM/SMT consumers of ifaces for free | agent | design | med |
| `cacheKeyVersion` bumps v5→v6 | Pre-fix cached ifaces carry open bodies; without the bump the M3 aliasDigest of an old open-body iface matches and serves stale verdicts | compiler | design | low |
| M1's TC_TYPE_SHADOW_001 capture mechanism stays for residual nominal cases | Post-closure, alias bodies no longer carry bare alias names, but nominal ADT references can still be shadowed by an importer's same-named record alias | compiler | design | low |
| M4 (module-qualified nominal identity) remains a separate future doc | Full generality (same-named ADTs, instances, codegen names) is a representation change; not needed for this report | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Interfaces are closed at build time in the defining module (compiler semantics; no human decision needed — this is lexical scoping extended across the boundary).
- [x] Cache key version bump v5→v6 (mechanical consequence; `cache_invalidation_test.go:339` hard-asserts "v5" and must be updated in the same change).
- [x] M4 stays out of scope (already parked as a separate human decision in the parent doc).

## Solution Design

### Overview

At interface build time, every type that leaves a module — alias bodies in `Iface.TypeAliases`, the `Type` inside every export's `Scheme`, and constructor field/result types — is **closed over the defining module's effective alias environment**: each `TCon` that names a *type alias* of the defining module (local or imported, exactly the names its own checker would expand) is replaced by its expansion, with the nominal name preserved via `TRecord.TypeName`. Names that are not aliases in the defining module (primitives, nominal ADTs) stay bare `TCon` deliberately. After this, no importer's alias environment can change the meaning of anything inside the interface.

### Architecture

**Components:**
1. **Rebuilding walker** `closeTypeTCons(t types.Type, resolve func(string) (types.Type, bool), inProgress, memo map[string]bool/types.Type) types.Type` — a TCon-substituting, cycle-safe rebuild over `types.Type`. Variant-complete over every type that implements `Substitute` (see Files), mirroring `expandAlias`'s terminal rule: when the target is a `TRecord` with empty `TypeName`, set `TypeName = <alias name>` (the existing M-CROSS-MODULE convention, `unification_core.go:185-193`). Cycle members (`type A = ...A...`) keep their `TCon` and are treated as opaque, exactly as `expandAlias`'s `seen` set does (`unification_core.go:177-183`).
2. **Closure env** — the defining module's effective alias view, assembled where it already exists: `elaborator.GetTypeAliases()` (local) overlaid on `imports.ImportedTypeAliases` **post-`shadowLocalTypeNames`** (imported), i.e. precisely the two maps registered on the module's own checker at `pipeline_module_compile.go:126-140`. This is the same environment the defining module's unifier used, so closed bodies cannot diverge from local semantics.
3. **Application point** — `buildAndRegisterInterface` (`pipeline_module_compile.go:512`): after `BuildInterfaceWithTypesAndConstructors` and before `embedTransitiveAliases`, walk and re-`AddTypeAlias` every entry of `unitIface.TypeAliases`, and rewrite each export scheme's `Type` and each constructor's `FieldTypes`/`ResultType` with the walker. `embedTransitiveAliases` then embeds *already-closed* dependency bodies (deps compiled by the same binary produced closed ifaces).
4. **Cache** — `cacheKeyVersion` "v5"→"v6" (`cache_key.go:32`). The M3 cache key already covers closure members via `iface.Digest + aliasDigest(iface)`; `computeDigest` hashes export schemes (`builder.go:689-752`, `schemeToString`), so scheme closure changes digests too. The version bump guarantees no pre-fix interface (open bodies) is ever served post-fix.

**What deliberately stays bare `TCon`:** primitives (`int`, `string`, ...); nominal ADTs and newtypes not in the defining module's alias env (they must stay nominal — `TestXModAlias_NominalNewtypeStaysNominal`, `TestXModAlias_NominalSumADTStaysNominal`); cycle members; applied *parameterized* aliases (`Box[int]` as `TApp` — see Deferred Decisions); type variables.

### Implementation Plan

**Phase 1: walker + alias-body closure (~1 day)**
- [x] Implement the rebuilding walker with an exhaustive type switch; add a unit test enumerating every `Substitute` implementor (the switch and the test are generated from the same list to prevent silent gaps).
- [x] Wire closure of `Iface.TypeAliases` bodies in `buildAndRegisterInterface`.
- [x] Pipeline tests: T1 (`TestAliasBodyClosure_ExplicitImportOfUnrelatedSameNameType`), T2 (`TestAliasBodyClosure_ImportOrderDoesNotCaptureInnerName`), stapledons 4-module shape (`TestAliasBodyClosure_NavigationSolTrappistShape`).

**Phase 2: scheme + constructor closure (~1 day)**
- [x] Apply the walker to export schemes and constructor field/result types.
- [x] Pipeline test: T3 (`TestAliasBodyClosure_AnnotationSchemeNotCapturedByImporter`).
- [x] Flip `TestTypeNameShadow_CapturedImportedAliasIsLoud` to the positive `TestAliasBodyClosure_CapturedImportedAliasNowWorks` (keep the old name as an alias or delete it — parent doc predicts this flip; keep a control that TC_TYPE_SHADOW_001 still fires for the residual nominal case).

**Phase 3: cache + docs (~0.5 day)**
- [x] `cacheKeyVersion` "v5"→"v6"; update the hard assertion at `cache_invalidation_test.go:339`; add `TestAliasBodyClosure_CacheInvalidatedOnce` (compile with v5-era cached ifaces present → all miss).
- [x] Update the comment block in `internal/pipeline/type_name_shadow.go` (capture is now the residual guard, not the main mechanism) and the parent doc's M2 status line.
- [x] Document the residual import-order behavior for names the importer itself writes in `docs/LIMITATIONS.md`.

**Phase 4: verification (~0.5 day)**
- [x] `make test-core`, `go test ./internal/pipeline/... ./internal/types/... ./internal/iface/...`.
- [x] Production-shape verification: mandatory four-module navigation/sol/trappist regression passes. External Stapledons scratch-clone verification is unavailable in this workspace.

### Files to Modify/Create

**New files:**
- `internal/pipeline/alias_body_closure.go` — walker + closure entry point (~200 LOC)
- `internal/pipeline/alias_body_closure_test.go` — T1/T2/T3/stapledons/capture-flip/cache tests, using the existing `checkModules` harness (~300 LOC)

**Modified files:**
- `internal/pipeline/pipeline_module_compile.go` — call closure in `buildAndRegisterInterface` (~15 LOC)
- `internal/pipeline/cache_key.go` — version bump (1 LOC)
- `internal/pipeline/cache_invalidation_test.go` — v5→v6 assertion (1 LOC)
- `internal/pipeline/type_name_shadow.go` — comment updates (comment-only)
- `design_docs/planned/v0_44_0/m-type-name-shadow-and-cache.md` — M2 status → specified here (comment-only)
- `docs/LIMITATIONS.md` — residual same-name import behavior (~10 LOC)

## Examples

### Example 1: The report (T1)

**Before (v0.52.x, fails):**
```
$ ailang run --package-dir . --entry run main.ail
Error: type error in main (decl 0): type unification failed at [function application at main.ail:6:44]:
failed to unify parameter 0: failed to unify record field 'items': record field mismatch:
expected 1 fields, got 2; expected fields {y}, actual {w, x}
```

**After (passes):**
```
$ ailang run --package-dir . -args-json '1' --entry run main.ail
1
```
`main` keeps both imports unchanged; `Item` from `b` still works for `one() : Item` in the same module. The two same-named types coexist because a's `Box` no longer contains a name that main can rebind — it contains a's `Item` already expanded.

### Example 2: Interface content before/after

**Before** — module `a`'s cached interface (`iface.json`, verbatim from the v0.52.3 repro):
```json
"count": { "type": { ... "params": [ { "tag": "tcon", "data": { "Name": "Box" } } ], ... } },
"type_aliases": {
  "Box":   { "tag": "trecord", "data": { "fields": { "items": { "tag": "tapp",
             "data": { "constructor": {"Name": "list"}, "args": [ { "tag": "tcon", "data": { "Name": "Item" } } ] } } } } },
  "Item":  { "tag": "trecord", "data": { "fields": { "x": {"Name":"int"}, "w": {"Name":"int"} } } }
}
```
The `Item` inside `Box`'s body is a bare name resolved by whoever reads it.

**After** — the same interface closed in `a`:
```json
"type_aliases": {
  "Box": { "tag": "trecord", "data": { "type_name": "Box", "fields": { "items": { "tag": "tapp",
           "data": { "args": [ { "tag": "trecord", "data": { "type_name": "Item",
             "fields": { "x": {"Name":"int"}, "w": {"Name":"int"} } } } ] } } } } }
}
```
`count`'s param, when expanded by any importer via `expandAlias("Box")`, terminates on closed records (`TRecord` is terminal in `expandAlias`, `unification_core.go:185-193`); the importer's alias env for `Item` is never consulted. `mk`'s return scheme is *already* stored in exactly this closed shape today (inference-derived types carry `type_name`), which is why the "actual" side of the error message was always correct.

## Success Criteria

- [x] T1 repro: `count(mk())` returns 1 with both imports present (`TestAliasBodyClosure_ExplicitImportOfUnrelatedSameNameType`)
- [x] T2 repro: import order cannot change another module's types (`TestAliasBodyClosure_ImportOrderDoesNotCaptureInnerName`)
- [x] T3 repro: annotation schemes are not captured (`TestAliasBodyClosure_AnnotationSchemeNotCapturedByImporter`)
- [x] Captured-alias program from the M1 suite compiles and runs correctly (positive flip of `TestTypeNameShadow_CapturedImportedAliasIsLoud`)
- [x] TC_TYPE_SHADOW_001 still fires for the residual nominal-shadow case (control test)
- [x] All existing tests in `internal/pipeline/{cross_module_nonrecord_alias,cross_package_alias,alias_poly,ctor_alias_pattern,local_type_shadows_import,cache_transitive_alias,cache_alias_digest_stable}_test.go` pass
- [x] Cache invalidates exactly once (version bump test)
- [x] All tests passing (`make test-core`), documentation updated

## Testing Strategy

**Unit tests:**
- Walker: every `types.Type` variant round-trips through `closeTypeTCon` unchanged when the name is not in `resolve`; every alias-name `TCon` at every nesting depth (record field, list element, tuple, func param/return, `TApp` arg, row label) is replaced; cycles terminate; `TypeName` is set on record targets with empty `TypeName` and preserved when set; sorted/deterministic output.

**Integration tests:**
- `checkModules` harness (same style as `local_type_shadows_import_test.go`) for T1/T2/T3, the stapledons 4-module shape, and the capture flip.

**Manual testing:**
- Scratch-clone stapledons-godot-trappist, revert `Exoplanet`, run the sim test that imported trappist1's `Planet` while calling `navigation.trappistFacts(solSystem())`.
- `examples/intra_package_imports` still builds and runs (`ailang check` on the package).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Walker file location (`internal/pipeline/alias_body_closure.go` vs `internal/iface`) — agent may choose; pipeline is recommended because the closure env is assembled there.
- Whether to also close *applied parameterized aliases* (`Box[int]` appearing inside a scheme) by instantiating the body at build time — agent may defer; if deferred, document that applied-alias instantiation still resolves the head in the importer's env (residual, same class as M4).
- Whether T2's bulk alias auto-import should switch from first-wins to an ambiguity diagnostic when two direct imports export the same name the importer itself writes (parent doc open question 1) — agent may defer; not needed for this report.
- Error-message adjustments where expanded records replace names inside alias bodies — agent may choose presentation.

## Non-Goals

**Not attempted in this feature:**
- Module-qualified nominal identity (`TCon{Module, Name}`, same-named ADTs unifying, derived-instance and codegen collisions) — M4, a type-representation change, human decision, tracked separately.
- REPL/WASM/SMT top-level alias-env merging (`repl/module_registry_load.go`, `smt/`) — closed interfaces flow into those consumers automatically, but their own bare-name merging remains; only full fix is M4.
- Changing what `import M (T)` binds when the importer itself writes `T` and two imports export it — remains last-explicit-wins / first-bulk-wins; only the *cross-module inner-name* channel is closed.

## Timeline

**Week 1** (3–4 days):
- Phase 1 (walker + body closure + T1/T2/stapledons tests): day 1
- Phase 2 (scheme/ctor closure + T3 + capture flip): day 2
- Phase 3 (cache bump + docs): day 3
- Phase 4 (verification, stapledons manual gate, buffer): day 4

**Total: ~4 days** (estimates doubled from gut feel per skill guidance; the walker's variant-completeness is the main unknown).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Walker misses a `types.Type` variant → alias names inside that variant stay open | High (silent partial fix) | Exhaustive switch generated from the list of `Substitute` implementors + a unit test enumerating the same list; unknown node types left unexpanded (fail-safe direction) with a build-time comment requiring an update |
| Closure diverges from what the defining module's unifier would expand (wrong env) | High | Closure env is the exact two maps registered on the module's own checker, post-M1-shadow; `TestXModAlias_*` and `local_type_shadows_import_test.go` are the regression net |
| Cross-module record update (`{ rec \| field: v }`, M-FIX-RECORD-UPDATE) or `--args-json` decoding breaks on closed bodies | Med | Both features expand through the same alias env the closed bodies feed; the literal's inferred types (already closed today, see `mk`) prove the path works; dedicated tests exist (`ctor_alias_pattern_test.go`, `cache_alias_digest_stable_test.go`) |
| Stale pre-fix cached interfaces served post-fix | High | `cacheKeyVersion` v5→v6 forces a one-time global miss; invalidation test |
| Interface digest churn breaks consumers comparing digests across versions | Low | Digests already change on any source edit; the version bump makes the boundary explicit and one-time |

## Conflict Surface

1. **Positions extended:** the *contents* of `Iface.TypeAliases` bodies (built at `iface/builder.go:457-483` and `:525`, embedded at `pipeline_module_compile.go:546-567`), export scheme types, and constructor field/result types. No new syntactic or semantic positions in source code.
2. **Other valid constructs already in those positions:** non-record aliases (`type UserId = int`, M-XMOD-ALIAS), chained aliases (`Ref = Id`), cyclic aliases (`type A = ... A`), parameterized aliases (`Ident[a]`, M-XMOD-ALIAS-POLY), newtype-record aliases (M-STREAM-DX/M4), nominal newtypes and sum ADTs (must stay `TCon`), records with row variables (open records, `TRecord2`/`Row`), labelled types (`TLabelled`), function types with effect rows.
3. **How the closure disambiguates:** a `TCon` is replaced **only** if its name is in the defining module's effective alias env (the same names its own checker expands); primitives and nominal ADTs are not in that env by construction; cycle members and in-progress expansions are treated as opaque; `TVar`/`TVar2` are never touched (parameterized-alias substitution keys on variables, so params survive). Iteration is over sorted names for determinism.
4. **Programs that MUST still work** (existing fixtures, read from the test files): `internal/pipeline/cross_module_nonrecord_alias_test.go` — `TestXModAlias_NonRecordAliasInField`, `TestXModAlias_NonRecordTargetKinds`, `TestXModAlias_TupleAndFuncAliasTargets`, `TestXModAlias_ChainedAlias`, `TestXModAlias_ChainedRecordAlias`, `TestXModAlias_CyclicAliasTerminates`, `TestXModAlias_NominalNewtypeStaysNominal`, `TestXModAlias_NominalSumADTStaysNominal`, `TestXModAlias_RecordAliasUnchanged`; `cross_package_alias_test.go` — `TestCrossPackageTypeAliasUnification`, `TestTransitiveTypeAliasPropagation`; `alias_poly_test.go`; `ctor_alias_pattern_test.go`; `local_type_shadows_import_test.go` — `TestTypeNameShadow_DirectImportUnexportedLocal`, `TestTypeNameShadow_LocalADTNotExpandedByImportedAlias`, `TestTypeNameShadow_ImportedAliasStillExpandsWithoutLocal`, `TestTypeNameShadow_LocalStillTypeChecks`, `TestTypeNameShadow_CapturedAliasUnusedIsFine`, `TestTypeNameShadow_DaneelTransitiveShape`; `cache_transitive_alias_test.go` — `TestCacheKey_TransitiveAliasEditInvalidates`; `examples/intra_package_imports/` (package builds and runs).
5. **Deliberate changes (intentional incompatibilities):**
   - (a) `TestTypeNameShadow_CapturedImportedAliasIsLoud` flips to positive: a program that today errors with TC_TYPE_SHADOW_001 (imported alias whose body names a locally declared type) now **works** — the body no longer contains the capturable name. This is the outcome the parent doc's M2 predicted ("the capture case in audit row 5 simply works"). A control test must keep proving TC_TYPE_SHADOW_001 for the residual nominal case.
   - (b) Every cache entry misses exactly once (v5→v6).
   - (c) Error messages and `--args-json` type decoding may show expanded records where a name inside an alias body used to appear (top-level names are preserved via `TRecord.TypeName`).

## Verification log

| Claim | Evidence |
|-------|----------|
| T1 repro fails on v0.52.3 dev (0cbb8cc) | `/tmp/repro` package: `ailang run --package-dir . -args-json '1' --entry run main.ail` → `record field mismatch ... expected fields {y}, actual {w, x}` (transcript in Problem Statement) |
| T2 repro: import order alone breaks it (no `Item` import) | same package, `import ./b (one)` before `import ./a (Box, mk, count)` → identical error (run 2026-10-07) |
| T3 repro: annotation schemes carry bare alias TCons | module `c` with `countItem(i: Item)`, importer also imports b's `Item` → identical error (run 2026-10-07) |
| Local declaration gets a loud error instead (asymmetry) | `probe2.ail` declaring `type Item = {z: bool}` + `count(mk())` → `TC_TYPE_SHADOW_001 ... module probe2 declares its own type Item` |
| Fix works when the colliding import is removed | with `import ./b (one)` (no `Item`): `ailang run ... -args-json '1'` → `1` |
| `count`'s param is stored as bare `TCon{Name:"Box"}` | `/tmp/repro/.ailang/cache/compile/modules/pkg__local__repro__a/iface.json`: `"params": [{"tag": "tcon", "data": {"Name": "Box"}}]` |
| `Box`'s alias body contains a bare `TCon{Name:"Item"}` | same `iface.json`: `"type_aliases"."Box"."fields"."items"` = `tapp(list, tcon "Item")` |
| `mk`'s return scheme is already fully closed with `type_name` tags | same `iface.json`: return is nested `trecord` with `"type_name": "Box"` / `"type_name": "Item"` |
| The importer's unifier resolves names through a flat, unqualified env | `internal/types/unification_core.go:17` (`aliasEnv map[string]Type`), `:172` (`target, exists := u.aliasEnv[con.Name]`); `TRecord` terminal at `:185-193` |
| Explicit symbol import overwrites the alias map with no guard (last import wins) | `internal/pipeline/pipeline_module_imports.go:244` (`imports.ImportedTypeAliases[bindName] = alias`, M-FIX-RECORD-UPDATE block); verified live: importing `Item` from a then from b binds b's (`probe3.ail` typechecks with `r() -> Item = one()` only when b's is last) |
| Bulk alias auto-import is first-wins (T2's engine) | `internal/pipeline/pipeline_module_imports.go:277-283` (`if _, exists := imports.ImportedTypeAliases[aliasName]; !exists`) |
| Alias bodies are built with NO name resolution at all | `internal/iface/builder.go:461`: `internalType := astTypeToInternalType(recordType)` → `AddTypeAlias` directly; `astTypeToInternalType` (`builder.go:23`) is a pure AST→types conversion |
| No rebuilding TCon-substitution walker exists (must be written) | `internal/types/traverse/` is read-only (`traverse.go:56` `Visit` callback; `wrappers.go` grep for `Rebuild|Substitut` → no hits); every type's `Substitute` (`types.go`, `types_v2.go`) keys on `TVar` names, not `TCon` |
| M1's capture guard covers only locally declared names | `internal/pipeline/type_name_shadow.go:48-52` (`localTypeNames(file)` → delete) and `:63-76` (capture fixpoint over `local[ref]`) — imported names are not in `local` |
| `cacheKeyVersion` is "v5" and hard-asserted | `internal/pipeline/cache_key.go:32` (`const cacheKeyVersion = "v5"`); `internal/pipeline/cache_invalidation_test.go:339-340` (`if cacheKeyVersion != "v5"` → Fatal) |
| The parent doc defers M2 as doc-only/unscheduled | `design_docs/planned/v0_44_0/m-type-name-shadow-and-cache.md`: "M2 and M4 are doc-only", "M2 takes 1–2 days", header "M2 and M4 are unscheduled" |
| Cited regression fixtures exist and assert what is claimed | `grep -n "func Test"` over the six `internal/pipeline/*_test.go` files (names listed in Conflict Surface 4); `ls examples/intra_package_imports/` → `ailang.toml`, `service.ail`, `types.ail` |
| `checkModules` harness exists for the new tests | used throughout `internal/pipeline/local_type_shadows_import_test.go` (e.g. `:68`) |
| No new error code is proposed | the fix makes valid programs work; the residual case keeps TC_TYPE_SHADOW_001 (allocated, `type_name_shadow.go:13`) |

## Related Documents

<!-- Auto-populated by Ollama neural search on "alias body closure" — the search
     ran with fallback-simhash (embeddings unavailable) and found nothing; the
     duplicate gate was applied manually by grep instead. -->

**Implemented (may inform design):**
- [m-transitive-alias-env-import](../../implemented/v0_22_0/m-transitive-alias-env-import.md) — introduced the every-loaded-module alias pull that M3 limited and this doc makes scope-independent.
- [m-xmod-alias-poly](../../implemented/v0_30_0/m-xmod-alias-poly.md) — parameterized aliases; their substitution keys on type variables, which is why closure leaves params intact.
- [m-compile-cache-dirty-build-key](../../implemented/v0_43_2/m-compile-cache-dirty-build-key.md) — the identity half of the cache key this doc's version bump completes.

**Planned (check for overlap):**
- [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md) — **the parent doc.** Its M1/M3 shipped in v0.44.x; its M2 ("close alias bodies over their defining module", audit rows 3/5) was doc-only and unscheduled. This doc is that milestone's implementation spec, NOT a duplicate: it schedules M2, adds the explicit-import and annotation-scheme trigger classes (T1/T3) that the parent's M2 sketch did not cover, and extends closure from `TypeAliases` bodies to export schemes on T3 evidence. M4 (module-qualified nominal identity) remains parked there as a separate human decision.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- Task `task-d0fe667d` (coordinator report, AILANG v0.52.0 bf2436a) and the stapledons-godot-trappist `Exoplanet` workaround
- M-DX-PI-HARNESS friction doctrine — the create-doc script's `set -e` failure on empty search results was fixed in the same session (`|| true` guard at both merge sites in `create_planned_doc.sh`)

## Future Work

- M4: module-qualified nominal identity (`TCon{Module, Name}`) — closes same-named ADTs, constructors, derived instances and codegen collisions; requires a human scoping decision (parent doc, Design Freeze).
- Ambiguity diagnostics for same-named direct imports when the importer itself writes the name (parent doc open question 1).
- REPL/WASM/SMT top-level alias merging beyond what closed interfaces provide.

---

**Document created**: 2026-10-07
**Last updated**: 2026-10-07

## Maintainer rulings (Ruled 2026-10-08 by Mark)

The decision marked `human` is ratified as written: M4 (module-qualified nominal identity) remains a separate future doc.


## Implementation evidence (2026-10-08)

All scoped phases and success criteria are implemented and verified; independent
sprint evaluation passed (95/100). See the
[implementation report](m-alias-body-closure-implementation-report.md) for red/green
regressions, runtime/cache evidence, final gates and the documented residuals.
Applied parameterized heads and recursive references remain opaque as permitted;
import ambiguity diagnostics and module-qualified nominal identity are deferred.

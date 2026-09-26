# M-TYPE-NAME-SHADOW: a module's own type must win, and the compile cache must see alias edits

**Status**: Planned. M1 and M3 are implemented in the PR that adds this doc; M2 and M4 are doc-only.
**Target**: v0.44.x (M1 and M3); M2 and M4 are unscheduled.
**Priority**: P0 for M1 and M3 (silent wrong typing, and a stale "No errors"); P1 for M2; P2 for M4.
**Estimated**: M1+M3 done (about 250 LOC with tests). M2 takes 1–2 days. M4 takes 1–2 weeks and changes how types are represented.
**Dependencies**: None. M2 builds on M1.
**Reporter**: Daneel, 2026-09-26 (sunholo-data/daneel PR #229, branch `feat/links`). Reproduced on v0.40.1 and on v0.43.1-8.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The transitive alias pull used to follow Go map order and topo order across sibling modules. It is now sorted and limited to the module's own import closure. |
| A2: Replayability | +1 | The cache no longer returns a verdict that a fresh compile contradicts. |
| A3: Effect Legibility | 0 | Effects are not involved. |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | +1 | A module's typing depends only on its own source and its import closure. A module it never imports can no longer change it. |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | A silent wrong type becomes either a correct result or an error with a code that names both definitions. Agents writing AILANG hit same-named `Row`, `Item` and `Config` types constantly. |
| A8: Minimal Syntax | 0 | There is no new syntax. |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | Two packages can now each export `type Row` and both be used together. |
| A11: Structured Failure | +1 | Adds the coded error TC_TYPE_SHADOW_001 in place of an unrelated "record field mismatch". |
| A12: System Boundary | 0 | — |

**Net Score: +6**. **Decision: move forward.** No axiom scores −1.

## Problem Statement

Daneel's report, condensed: in `daneel_links.ail`, rename `LinkRow` to `Row`, then run `ailang check daneel_intake.ail`. The check fails like this:

```
type error in daneel_heartbeat (decl 20): type unification failed at [return type annotation
at daneel_heartbeat.ail:168:8]: record field mismatch: expected 5 fields, got 3
  expected {cadence, deadlineS, lane, notes, verb}   <- heartbeat's own `export type Row`
  actual   {date, from, subject}                     <- links' `export type Row`
```

`daneel_intake` imports functions from `daneel_links`. It reaches `daneel_heartbeat` only through `daneel_brief`. Heartbeat's own `parseRow(line) -> Row` was checked against the `Row` declared in links.

### Reproduction (worktree binary built at 0e9a554)

Scratch clone of daneel `feat/links` at 15d20a8, after `ailang lock`:

- Pristine: the check fails only on an unrelated unlocked package.
- After `sed 's/LinkRow/Row/g' daneel_links.ail`: the exact error above.
- With the fix binary: `✓ No errors found!`.
- Across all 47 `tools/*.ail` files checked with `AILANG_NO_CACHE=1`, `daneel_intake.ail` is the only verdict that changes between the old binary and the fixed one.

### Minimal repro: two modules, no transitive import needed

```ailang
-- ma.ail
module ma
export type Row = {date: string}
export pure func mkA(d: string) -> Row { {date: d} }

-- main.ail
module main
import ma (mkA)
type Row = {verb: string, lane: string}          -- not even exported
export pure func mkD(v: string) -> Row { {verb: (mkA(v)).date, lane: "x"} }
```

Before the fix, this fails with `record field mismatch ... expected 2 fields, got 1`: main's own `Row` had been replaced by ma's. The reverse is worse. `-> Row { {date: ...} }` in main was **accepted**, which is a soundness hole. `TestTypeNameShadow_LocalStillTypeChecks` covers that case.

**Why Daneel's flat three-module case did not reproduce.** In `resolveModuleImports` (`internal/pipeline/pipeline_module_imports.go:91` at HEAD 0e9a554), a module with **zero imports** returns early, before any foreign aliases are collected. The trigger is not transitivity. The trigger is a victim module that has at least one import (any import, even `std/string`) and is compiled after some module that exports the same name. When `mb` in the flat case gains `import std/string (trim)`, the flat case fails too; this was checked with `ailang check`. `TestTypeNameShadow_DaneelTransitiveShape` reproduces the four-module shape of the report (links / heart / brief / main).

## Root cause

The file:line references below are at HEAD 0e9a554, before the fix.

1. **Imported aliases are registered after local ones and overwrite them.** In `internal/pipeline/pipeline_module_compile.go:122-140`, the module's own elaborator aliases are registered with `typeChecker.RegisterTypeAlias`, and then every entry of `imports.ImportedTypeAliases` is registered on top. `RegisterTypeAlias` (`internal/types/typechecker_core.go:272`) is a plain map assignment, so the last write wins, and the imported `Row` replaces the local one.

2. **The imported set is far wider than the imports.**
   - `resolveSelectiveImports` (`pipeline_module_imports.go:224`) adds *every* alias of every directly imported module, whatever symbols were imported (M-TYPE-ALIAS).
   - The M-TRANSITIVE-ALIAS-ENV-IMPORT loop (`pipeline_module_imports.go:133`) adds every alias of **every module compiled so far**: `modLinker.GetLoadedModules()`, which includes siblings the module never imports. It walks them in Go map order.

   Heartbeat never imports links. But links is compiled earlier in topo order (`[... links std/string heart brief main]`), so links' `Row` reached heartbeat through that loop.

3. **Everything is keyed by the bare name.** `ImportedTypeAliases`, `ImportedAliasParams`, `ImportedADTTypeParams`, `Iface.TypeAliases` and the unifier's `aliasEnv` are all `map[string]...` keyed by the unqualified name. So "which `Row`" cannot even be expressed.

### Audit: every place a type name is keyed without its defining module (CLAUDE.md §3)

| # | Site | Keyed by | Flaw | Status |
|---|------|----------|------|--------|
| 1 | `pipeline_module_compile.go:122-140`: register aliases on the checker | bare name, imported registered after local | **the reported bug** | **fixed (M1)** |
| 2 | `pipeline_module_imports.go:133`: transitive alias pull | bare name; first write wins in map order; reads every loaded module | nondeterministic, and reaches modules that are not imported | **fixed (M1/M3)**: sorted, limited to the import closure |
| 3 | `pipeline_module_imports.go:224`: all aliases of a direct import | bare name, first write wins in import order | wrong when two direct imports export the same name and the importer uses that name | open: M2 and M4 |
| 4 | `pipeline_module_compile.go:96-109`: `adtTypeParams` | type name, imported set first; local set only `if !exists` | a local ADT named like an imported ADT got the **imported** parameter count | **fixed (M1)**: local names are removed from `ImportedADTTypeParams` |
| 5 | `iface/builder.go:462-525`: `Iface.TypeAliases` bodies | a body names other types by bare `TCon` | `Seen = {rows:[Row]}` is expanded in the importer's scope, so a local `Row` captures it | **loud error (M1)**; real fix is M2 |
| 6 | `pipeline_module_compile.go:525` `embedTransitiveAliases` | bare name | could embed an imported `Row` into the interface of a module that has its own private `Row` | **fixed (M1)**: local names are removed before embedding |
| 7 | Constructors: `resolveSelectiveImports` auto-imports all constructors, first write wins (`:242`); `ImportedCtorTypes` | constructor name, then ADT name | two imported ADTs with the same constructor name collide, and a `TCon "Row"` ADT is nominally equal across modules | open: M4. Local constructors already override imported ones (`compile.go:103`) |
| 8 | ADT / nominal identity: `TCon{Name}` and `TRecord.TypeName` | bare name | `ma.T` and `mb.T` unify as the same type, and codegen names collide | open: M4 (changes the type representation) |
| 9 | Typeclass instances: `derived_eq.go` → `InstEnv.Add` and `DictionaryRegistry.RegisterDerivedEq` (`types/dictionaries.go:463`, key `prelude::Eq::<lowercased name>`) | lowercased bare name | two modules each deriving Eq on `Row` collide; the duplicate is swallowed silently (`derived_eq.go:34`). Harmless today only because runtime equality is structural. `Row` and `row` also collide | open: M4 |
| 10 | REPL/WASM `repl/module_registry_load.go:71-74` (elaborator) and `:243-249` (checker) | bare name, map order across modules | the local definition wins on the checker, but which foreign `Row` wins is nondeterministic | open: M2 (same fix as the pipeline) |
| 11 | SMT `smt/verify.go:90-99` `recordAliases` / `adtTypes` merge | bare name, first write wins | a contract could be encoded against the wrong `Row` | open: M2 |
| 12 | `AllCtorTypes` (`pipeline_module_imports.go:59`) | constructor name | used only in diagnostics; the worst case is a misleading suggestion | accepted |

## Stale-cache finding (item 3 of the report)

**Reproduced, and the problem is broader than shadowing.** Tested with the pre-fix binary on the four-module shape:

1. `links` uses `LinkRow`. Run `check`: ✓ (result cached).
2. Rename it to `Row`. Run `check`: **✓ (stale)**.
3. Run with `AILANG_NO_CACHE=1`: record field mismatch.

This is the reverse of Daneel's wording, but it is the same defect. There are two causes. The line references are at HEAD.

- **C1: the key covers only direct imports.** `prepareCacheLookup` (`pipeline_module_cache.go:25-30`) puts only `mod.Imports` digests into `ModuleCacheKey`. The compile, however, reads the aliases of every module that item 2 of the root cause reaches: the whole closure, and before the fix, siblings too. Heartbeat's key did not change when links changed, because heartbeat does not import links. The cache served its old verdict. The compile itself had depended on a module outside its dependency graph.
- **C2: the interface digest does not cover alias bodies.** `iface.Builder.computeDigest` (`iface/builder.go:689-752`) hashes exports and constructors only. In a shadowing-free repro, `ta: Inner={x:int}` ← `tb: Outer={items:[Inner]}` ← `tc` reads `i.x`. Editing `Inner` to `{y:int}` gives a cached `✓` and a fresh `record field 'x' not found` (`TestCacheKey_TransitiveAliasEditInvalidates`). Adding closure digests alone does not fix this, because `ta`'s digest does not change either.

**Relation to #1275 (M-COMPILE-CACHE-DIRTY-BUILD-KEY, shipped v0.43.2).** #1275 made the **compiler-identity** part of the key sound, so that dirty rebuilds are no longer served each other's verdicts. This is the **dependency** part of the same key being unsound. The key is `(identity, source, dep digests)`, and "dep digests" covered neither the transitive closure nor alias bodies. The same principle applies: the key must cover everything the compile reads.

## Solution

### M1: a local type shadows imported ones, and a captured alias is a loud error (implemented)

The new file `internal/pipeline/type_name_shadow.go` adds `shadowLocalTypeNames(imports, file, modID)`. It is called in `compileFreshModule` right after `resolveModuleImports`.

- For every type the module declares (any kind, exported or not; `localTypeNames` scans `Decls` and `Statements`, as `iface/builder.go` does), it deletes that name from `ImportedTypeAliases`, `ImportedAliasParams` and `ImportedADTTypeParams`. This fixes rows 1, 4 and 6 of the audit with a single mechanism, at the one place every consumer reads from: checker registration, derived Eq, and interface embedding.
- **Capture.** An imported alias whose body names a locally declared type would be expanded in the wrong scope. Case: `links.Seen = {rows:[Row]}` used in a module that declares its own `Row`. The same applies, as a fixpoint, to aliases whose bodies name such an alias. These aliases are withheld from the alias environment and registered with `CoreTypeChecker.RegisterAliasCapture` (`internal/types/alias_capture.go`). When `expandAlias` meets a captured name, it latches the error through the existing alias-arity latch, which `Unify` already reports. **Only using the alias is an error.** A module that merely has the alias in reach, as heartbeat does, is fine (`TestTypeNameShadow_CapturedAliasUnusedIsFine`). Message:

```
TC_TYPE_SHADOW_001: type Seen (from module links) refers to a type named Row, but module main
declares its own type Row, which shadows the Row that links means — expanding Seen here would
silently use the wrong Row.
  Suggestion: type names are not yet module-qualified; rename one of the two Row types (e.g. the
  one in main to a more specific name such as MainRow).
```

- The transitive pull iterates in sorted order (A1).

### M3: the cache key covers what the compile reads (implemented)

`internal/pipeline/cache_dep_closure.go`:
- `depClosure(modID)` computes the transitive import closure from the `LoadedModule.Imports` of each module.
- `prepareCacheLookup` keys on **every closure member**, each as `iface.Digest + "+" + aliasDigest(iface)`. `aliasDigest` is a sorted hash of alias name, parameters and body. This fixes C1 and C2.
- `resolveModuleImports` takes a `reach` set, and the transitive alias pull now reads only closure modules. A module's typing then depends only on its import graph. Without this, no key could be sound short of hashing every earlier module.

The interface digest format itself is unchanged; `aliasDigest` is computed on the cache side. The key changes, so every existing entry misses exactly once. There is no on-disk format change, so `cacheKeyVersion` is not bumped.

### M2: close alias bodies over their defining module (doc only, 1–2 days)

Build each interface's `TypeAliases` bodies **closed**. When the interface is built (`buildAndRegisterInterface`), replace every `TCon` in a body that names a record or transparent alias in the *defining* module's alias environment (local plus imported, after M1) with its expansion. That needs a `TCon`-substituting walker that is cycle-safe (a visited set per the type-traversal rule) and handles all type variants, and it must keep `TRecord.TypeName` so nominal codegen names survive. Recursive aliases and parameterized aliases applied with free variables stay as `TCon` and keep the M1 capture error. After M2:
- the capture case in audit row 5 simply works (`TestTypeNameShadow_CapturedImportedAliasIsLoud` changes into a positive test);
- audit row 3 shrinks to "the importer names `Row` itself while two imports export different `Row`s". That case should be an **ambiguity error naming both modules**, unless one is imported by symbol (`import links (Row)`), in which case the explicit import wins;
- rows 10 and 11 (REPL/WASM and SMT) adopt the same shadow-then-closed rule.

Cost: interface digests and cached interfaces change once, and error messages show expanded records where they used to show a name inside an alias body. The top-level name is kept through `TypeName`.

### M4: module-qualified type identity (doc only, a type-representation change)

`TCon{Module, Name}` or qualified names such as `links.Row` for nominal types (ADTs), constructors, derived instances and codegen type names (audit rows 7–9). This is the full answer to Daneel's first request, and it touches the type representation, unification, interfaces, codegen and dictionaries. It is **not** attempted here. M1 and M2 make aliases correct; M4 is needed only for same-named *nominal* types, which today fail loudly or unify silently (row 8). Track it separately.

## Conflict Surface

1. **Positions extended:** alias and ADT-parameter registration in the module pipeline; `Unifier.expandAlias` for names not in `aliasEnv`; the cache key.
2. **Other constructs in those positions:** imported aliases used without a local counterpart (M-TYPE-ALIAS); transitive record aliases (M-TRANSITIVE-ALIAS-ENV-IMPORT); non-record aliases (M-XMOD-ALIAS); parameterized aliases (M-XMOD-ALIAS-POLY, which shares the latch); newtype-record aliases (M-STREAM-DX/M4); derived Eq over imported aliases.
3. **Disambiguation:** local names are removed before registration. Captured names are never in `aliasEnv`, so the new latch fires only on a name that `expandAlias` would otherwise have returned unexpanded. Real ADTs are never registered as captured, because only imported *aliases* are.
4. **Must still work:** `internal/pipeline/cross_module_nonrecord_alias_test.go`, `cross_package_alias_test.go`, `alias_poly_test.go`, the M-TRANSITIVE-ALIAS-ENV-IMPORT tests, and `TestTypeNameShadow_ImportedAliasStillExpandsWithoutLocal`. All pass: `go test ./internal/...` is green, and the Daneel tools verdicts are unchanged apart from intake.
5. **Deliberate changes:**
   - (a) A module that used an alias from a module **outside its import closure** (a sibling) no longer resolves it. That code only ever worked by accident of topo order.
   - (b) Using a captured alias is now an error instead of a wrong type.
   - (c) Every cache entry is invalidated once.

## Verification log

| Claim | Evidence |
|-------|----------|
| Two-module repro fails on HEAD | `ailang check md.ail` (old binary): `record field mismatch ... missing fields: lane, verb` |
| Flat case needs an import in the victim | flat `ma`/`mb`/`mc` ✓. After adding `import std/string (trim)` to `mb`: mismatch |
| Early return for import-free modules | `pipeline_module_imports.go:91` at 0e9a554 (`if len(fileImports) == 0 { return imports }`) |
| Exported function schemes are already expanded | `--trace` line: `seenAll -> (string -> { url: string, rows: [{ from, date ...}] })`, so capture happens only through alias bodies |
| Interface digest omits aliases | `iface/builder.go:689-752`: `jsonIface` has only Module, Schema, Exports, Constructors |
| TC_TYPE_SHADOW_001 was unallocated | `grep -rn TC_TYPE_SHADOW internal cmd` returned no hits before this change |
| Stale-cache repro | cache2 `ta`/`tb`/`tc`: cached ✓, `AILANG_NO_CACHE=1` gives `record field 'x' not found` |
| Daneel fixed | scratch clone at 15d20a8 + rename: old binary fails at `heartbeat.ail:168:8`; new binary ✓ |

## Tests

These are in `internal/pipeline/local_type_shadows_import_test.go` and `cache_transitive_alias_test.go`. Pre-fix, `DaneelTransitiveShape`, `DirectImportUnexportedLocal`, `LocalStillTypeChecks`, `CapturedImportedAliasIsLoud` (wrong error text) and the cache test fail; the other two are controls that pass both before and after. Mutation check: dropping `aliasDigest` from the key makes the cache test fail again.

- `TestTypeNameShadow_DaneelTransitiveShape`: the reported graph in four modules.
- `TestTypeNameShadow_DirectImportUnexportedLocal`: the minimal two-module case.
- `TestTypeNameShadow_LocalStillTypeChecks`: the soundness hole where a wrong literal was accepted.
- `TestTypeNameShadow_LocalADTNotExpandedByImportedAlias`, `..._ImportedAliasStillExpandsWithoutLocal`: controls.
- `TestTypeNameShadow_CapturedImportedAliasIsLoud` and `..._CapturedAliasUnusedIsFine`: the M1 capture contract.
- `TestCacheKey_TransitiveAliasEditInvalidates`: M3.

## Success Criteria

- [x] The Daneel repro checks clean with `Row` in both modules.
- [x] A local type always wins over imported ones, for every type kind.
- [x] Using a captured alias gives a coded error naming both modules.
- [x] An alias edit anywhere in the import closure invalidates the importer's cache entry.
- [ ] M2: closed alias bodies. The capture test becomes positive, and two different imported `Row`s referenced by name give an ambiguity error.
- [ ] M4: module-qualified nominal identity (separate doc).

## Design Freeze

- [x] Local shadows imported: this is standard lexical scoping; no human decision is needed.
- [x] Capture is an error on use only, not on reach. Erroring on reach would re-break Daneel's heartbeat.
- [x] The transitive pull is limited to the import closure (deliberate change 5a).
- [ ] M4 scope and priority: a human decision (Mark). It touches the type representation.

## Open questions

1. Should an ambiguous bare name (two direct imports export different `Row`s, the importer names `Row`, and has no local `Row`) be an error now, before M2? Today the first import wins. The choice is deterministic but may be wrong. It is not in the reported bug.
2. M2 changes error text: records inside alias bodies are shown expanded. Is that acceptable for the eval prompts that quote such errors?

## Related Documents

- [m-transitive-alias-env-import](../../implemented/v0_22_0/m-transitive-alias-env-import.md): introduced the every-loaded-module pull that M3 limits.
- [m-compile-cache-dirty-build-key](../../implemented/v0_43_2/m-compile-cache-dirty-build-key.md): #1275, the identity half of the same cache key.
- [m-xmod-alias-poly](../../implemented/v0_30_0/m-xmod-alias-poly.md): the alias-arity latch reused by M1.

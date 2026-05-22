# M-TRANSITIVE-ALIAS-ENV-IMPORT — propagate transitively-imported type aliases into the unifier's aliasEnv

**Status**: Implemented — shipped in commit `8e3d2d30` on `dev` (2026-05-22)
**Target**: v0.22.0 (committed to `dev`; first release including this fix tbd)
**Priority**: P0 — unblocks `motoko_agent` PR #28 and prevents silent breakage of every package that splits record-type aliases across multiple modules
**Estimated / Actual**: < 1 day estimated; ~4 hours actual (~167 LOC test + 15 LOC implementation + 33 LOC changelog)
**Risk**: LOW (confirmed) — additive: the fix only widens the alias-env, never narrows it; collisions resolved by existing local-wins precedence; regression test verified to fail on pre-fix HEAD with exact original error
**Source**: 2026-05-22 `motoko_agent` bug report (`msg_20260522_170317_f850eba4`). Repro reproduced locally on `v0.21.0-4-gdf2ed8de-dirty`.
**Evaluation**: PASS at 94/110 (85%) — see `.ailang/state/evaluations/eval_M-TRANSITIVE-ALIAS-ENV-IMPORT_round_1.json`.
**Related**:
- Surfaced by: [`3325d39f`](../../../) (M-SCHEME-IMPORT-PRESERVE-ADT-HEAD, 2026-05-20) — tightened generalization, exposed pre-existing gap masked by leaky TVars
- WASM-side analog: `ad84b68d` (M-WASM-TYPECHECK-FLOAT-DIVERGENCE — propagated imported aliases in `repl/module_registry`)

## Problem

When module `C` imports a function from module `B`, and `B`'s exported types reference a record-type alias declared in module `A` (which `B` imports but `C` does not), unification at `C`'s call-sites fails with:

```
type unification failed at [function application at C.ail:7:12]:
  failed to unify parameter 0: failed to unify record field 'items':
  cannot unify type constructor Inner with *types.TRecord
```

The `TCon("Inner")` is correct on one side; the `TRecord{name: string, ...}` is correct on the other (it's the expanded `Outer.items` element). They should unify after `expandAlias()` resolves `TCon("Inner")` to its underlying `TRecord` — but `Inner` isn't in `C`'s active `aliasEnv` because `C` never directly imports module `A`.

### Minimal reproduction (verified locally)

Three files, ~25 lines, in `/tmp/typebug-repro/typebug/`:

**`typebug/types.ail`**
```ailang
module typebug/types

export type Inner = { name: string }
```

**`typebug/lib.ail`**
```ailang
module typebug/lib

import typebug/types (Inner)

export type Outer = { items: [Inner] }

export pure func build() -> Outer {
  { items: [{name: "a"}, {name: "b"}] }
}

export pure func use_outer(o: Outer) -> int {
  match o.items { [] => 0, _ :: _ => 1 }
}
```

**`typebug/main.ail`**
```ailang
module typebug/main

import typebug/lib (build, use_outer)

export func main() -> int {
  let v = build();
  use_outer(v)
}
```

Run from `/tmp/typebug-repro/`:
```
$ AILANG_RELAX_MODULES=1 ailang check typebug/main.ail
→ Type checking typebug/main.ail...
→ Effect checking...
Error: type error in typebug/main (decl 0): type unification failed at
  [function application at typebug/main.ail:7:12]: failed to unify parameter 0:
  failed to unify record field 'items':
  cannot unify type constructor Inner with *types.TRecord
```

## Root cause

The unifier's `aliasEnv` is correctly wired *for the importer module's direct imports* but lacks transitive reach.

### Where it goes right

1. `internal/iface/iface.go:14` — every module's `Iface` carries its own `TypeAliases map[string]types.Type` (aliases declared *in that module*).
2. `internal/pipeline/pipeline_module_imports.go:166-173` — when resolving imports, aliases from each *directly-imported* iface are merged into `imports.ImportedTypeAliases`.
3. `internal/pipeline/pipeline_module_compile.go:201-205` — those merged aliases are then registered on the `typeChecker` via `RegisterTypeAlias`, populating `Unifier.aliasEnv`.
4. `internal/types/unification_core.go:88-108` — `expandAlias()` resolves `TCon(name)` → underlying `TRecord` when `name` is in `aliasEnv`.
5. `internal/types/unification_core.go:233-238` — the `TCon` vs `TRecord` unification dispatch correctly calls `expandAlias` to enable structural unification.

### Where it goes wrong

`typebug/main`'s direct imports are `typebug/lib` only. The loop at `pipeline_module_imports.go:166-173` only iterates `depIface.TypeAliases` for the *direct* dep:

```go
// CURRENT (insufficient): only direct dep's aliases
for aliasName, aliasTarget := range depIface.TypeAliases {
    if _, exists := imports.ImportedTypeAliases[aliasName]; !exists {
        imports.ImportedTypeAliases[aliasName] = aliasTarget
    }
}
```

`typebug/lib.iface.TypeAliases` contains `Outer = {items: [TCon("Inner")]}` but NOT `Inner` (which lives in `typebug/types.iface.TypeAliases`). Result: `main`'s `Unifier.aliasEnv` has `Outer` but not `Inner`, and `expandAlias(TCon("Inner"))` falls through.

### Why it surfaced after `3325d39f`

Pre-3325d39f, exported function schemes lost their ADT/record heads — `α_return` stayed unbound at the generalization boundary, producing over-polymorphic schemes like `forall α. () -> α` that unified vacuously at any shape. The diagnostic at `unification_core.go:233` (TCon-vs-TRecord) wasn't reached because both sides had collapsed to TVars.

3325d39f applied the substitution before generalization, so `build()` now correctly exports `() -> Outer` (with `Outer` retaining `[TCon("Inner")]` inside). The structural-vs-nominal unification at the call-site is now genuinely needed — and the missing alias becomes visible.

This is not a regression in `3325d39f`. It is a pre-existing gap exposed by removing the over-polymorphism that was masking it.

## Proposed fix (recommended: option C — linker-driven closure)

Three viable approaches; recommending **(C)** because it's the smallest diff, requires no schema change, and matches the existing layering.

### (A) Eager closure at iface generation — REJECTED

When generating `B.iface`, walk B's exports and embed all transitively-referenced aliases from A into B's `TypeAliases`.

- Pros: ifaces become self-contained.
- Cons: bloats every iface; couples iface generation to dep-graph traversal; risks digest churn when A changes alias bodies (cascade rebuild). Not worth it for the bug class.

### (B) Recursive iface lookup at import time — REJECTED

In `resolveSelectiveImports`, when copying alias `X = T` whose body `T` mentions `TCon("Y")`, recursively look up `Y` in deps' deps.

- Pros: minimal alias pollution.
- Cons: requires walking type bodies to find `TCon` references; needs cycle protection; doesn't handle aliases reached only through ADT result types.

### (C) Linker-driven closure at import resolution — RECOMMENDED

The linker (`ml.GetLoadedModules()` at `internal/link/module_linker.go:180`) already holds every transitively-loaded iface — direct deps and their deps. The current code at `pipeline_module_imports.go:166-173` just happens to only iterate the direct dep's aliases.

Fix: after the existing direct-import alias merge, add one pass that scans **every loaded module's** `TypeAliases` and merges any not already present. Local aliases (declared in the importer) win; direct-import aliases win over indirect; first-loaded indirect wins over later indirect.

This is the minimal-surface fix and parallels exactly what `ad84b68d` did for the WASM path (`repl/module_registry_load.go:289-303` — "loop through `mr.modules` and register imported aliases on typeChecker").

## Implementation

### Change 1 — `internal/pipeline/pipeline_module_imports.go`

Extend `resolveModuleImports` to accept the linker as an explicit dependency (it's already in scope) and add a final pass after the direct-imports loop. Pseudo-diff:

```go
// (after the existing for _, imp := range fileImports { ... } loop)

// M-TRANSITIVE-ALIAS-ENV-IMPORT: also pull in aliases from any
// transitively-loaded module's iface, so cross-module nominal
// types (e.g., Inner from A used inside Outer = {items: [Inner]}
// in B, imported by C) can be expanded by the unifier in C.
//
// Direct imports above already populated ImportedTypeAliases for
// direct deps; this loop adds aliases reachable only through
// transitive deps. First-wins matches the direct-import precedence
// (since direct deps populate before this runs).
for modPath, modIface := range modLinker.GetLoadedModules() {
    if modPath == "$builtin" || modIface == nil {
        continue
    }
    for aliasName, aliasTarget := range modIface.TypeAliases {
        if _, exists := imports.ImportedTypeAliases[aliasName]; !exists {
            imports.ImportedTypeAliases[aliasName] = aliasTarget
        }
    }
}
```

Local aliases still win because `pipeline_module_compile.go:196-199` registers `elaborator.GetTypeAliases()` *first*, then the imports map. Order is preserved — only the import map widens.

### Change 2 — `internal/repl/module_registry_load.go` (WASM path symmetry)

The WASM path already does the right thing for direct deps (commit `ad84b68d`, lines 289-303), but the same gap exists if A↔B↔C chains run through the WASM REPL. The existing `for _, mod := range mr.modules` loop iterates ALL registered modules in the registry, so this path may already handle transitive aliases correctly. Verify and add the same closure pass if not. (Likely already correct; flagged as audit item, not a required change.)

### Change 3 — no changes to `Iface` schema, no changes to `Unifier`.

## Test plan

### Regression test (required)

Add `TestCrossModuleNestedRecordAlias` to `internal/pipeline/pipeline_module_compile_test.go` (or a new `pipeline_module_alias_transitive_test.go` if cleaner).

The test clones the 3-file repro inline: A declares `type Inner = {name: string}`, B imports `Inner` and declares `type Outer = {items: [Inner]}` plus `build()` and `use_outer()`, C imports only B's functions and calls `use_outer(build())`. Assert type-check succeeds.

### Negative test (collision precedence)

Two modules `pkgA` and `pkgB` each export `type Status = { ... }` with different bodies. Importer C imports `Status` explicitly from `pkgA` and a function from `pkgB`. Assert that `pkgA.Status` (direct import) wins for unification — local/direct beats transitive.

### Existing suites

`make test`, `make test-imports`, `make verify-examples` should all stay green. No public API touched.

## Backwards compatibility

None at risk:

- Iface format unchanged.
- Unifier API unchanged.
- Only `imports.ImportedTypeAliases` widens with transitively-reachable aliases. Programs that compiled pre-fix continue to compile post-fix; programs that previously failed now succeed.
- Collision precedence is preserved by relying on `if _, exists := ...; !exists` (first-wins) ordering.

## Affected real-world code

From the bug report:

- `arniwesth/motoko_agent` PR #28 — `pkg/sunholo/motoko_ext_mcp@0.2.7/register.ail` — currently red on this exact pattern.
- Symmetric pattern in `motoko_ext_omnigraph`, `motoko_ext_context_mode`, `motoko_ext_compaction_ai` — any AILANG package that splits record-type aliases into a dedicated `types.ail` module will break on next republish against current `dev`. Blast radius is "every multi-file motoko package".

## Verification

After implementation:

1. Repro under `/tmp/typebug-repro/` type-checks clean.
2. New regression test passes.
3. `make test` and `make verify-examples` green.
4. Republish `motoko_ext_mcp@0.2.8` and rerun motoko_agent CI; PR #28 turns green.
5. Reply on inbox message `msg_20260522_170317_f850eba4` with fix-commit SHA.

## Out of scope

- Qualified alias namespacing (e.g., `typebug/types.Inner` as a unique key). Current design has a flat alias-name namespace; collisions resolved positionally. If two modules export same-named aliases with different bodies and both reach the importer transitively, the *first-loaded* one wins. This is acceptable for now; namespacing is a larger refactor (probably a v1.x concern).
- Iface digest / cascade republish behavior. The fix is internal to type-checking and doesn't change iface content, so digests are stable.

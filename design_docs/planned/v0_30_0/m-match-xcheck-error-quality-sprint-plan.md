# Sprint Plan — M-MATCH-XCHECK-ERROR-QUALITY

**Design doc**: [m-match-xcheck-error-quality.md](../v0_29_0/m-match-xcheck-error-quality.md)
**Mission iteration**: 15 (clause-3 accessibility cluster)
**Target**: v0.30.0
**Risk**: low · **Estimate**: ~0.5 day (~55 LOC impl+test)
**Chosen approach**: **Option A** — transitive constructor discovery at error time via a
diagnostic-only ADT registry (design doc's recommendation).

## Goal

Fix the empty `<ADT>'s constructors are: ` line in `MatchForeignConstructorError` when the
scrutinee's ADT is known only transitively (e.g. `std/json.asNumber` returns `Option[float]`
but the user imports only `std/result`, not `std/option`). Correctness is already right; only the
suggestion list is empty. After this fix the list enumerates the transitively-known ADT's
constructors (`Some, None`).

## Root cause (verified in code)

- `internal/types/typechecker_core.go:315 lookupADTConstructors` reverse-scans
  `tc.constructorTypes`, which holds ONLY directly-imported + local constructors
  (`pipeline_module_compile.go:92-111`). Transitive ADTs (Option, imported by std/json but not by
  the user) are absent → empty list.
- The linker DOES have every transitively-loaded interface:
  `modLinker.GetLoadedModules()` → `map[path]*iface.Iface`, each `iface.Constructors[name].TypeName`
  gives the ADT name. Topo-order compile (`pipeline_module.go:226`, deps-first, root last) means
  all transitive ifaces are registered before the root type-checks — **verified**.

## Design (Option A)

A **diagnostic-only** ctor→ADT map, separate from `constructorTypes` so it NEVER brings
constructors into scope (out-of-scope per design doc: no auto-import). `types` cannot import
`link` (cycle), so we pass a plain `map[string]string` — the "slimmed-down ADT registry" the doc
calls for.

### Milestone M1 — Diagnostic registry plumbing (~35 LOC)

1. `internal/types/typechecker_core.go`:
   - Add field `diagnosticCtorTypes map[string]string`.
   - Add `SetDiagnosticConstructorTypes(m map[string]string)`.
   - In `lookupADTConstructors`: if the primary scan yields empty AND `diagnosticCtorTypes` is
     non-nil, scan it as a fallback. (Primary still wins when populated.)
2. `internal/pipeline/pipeline_module_imports.go`:
   - Add `AllCtorTypes map[string]string` to `moduleImports`.
   - Populate it from `modLinker.GetLoadedModules()` — every loaded iface's
     `Constructors[name].TypeName`. (Diagnostic-only; not merged into `ImportedCtorTypes`.)
3. `internal/pipeline/pipeline_module_compile.go`:
   - `typeChecker.SetDiagnosticConstructorTypes(imports.AllCtorTypes)` right after
     `SetConstructorTypes`.

### Milestone M2 — Tests + docs (~20 LOC)

1. Strengthen `internal/pipeline/match_foreign_constructor_function_call_test.go`
   `TestSchemeImport_FunctionCallScrutinee_ForeignCtorRejected`: add an assertion that the message
   contains `Some` and `None` (Option's ctors, now non-empty for a transitively-known ADT). This
   is the doc's acceptance criterion, on the exact repro.
2. Confirm no regression across `internal/types` + `internal/pipeline` (existing
   M-MATCH-ADT-XCHECK suite).
3. CHANGELOG.md entry; design doc → `implemented/v0_30_0/` at close.

## Acceptance criteria (from design doc)

- [ ] Foreign-ctor error lists a non-empty constructor list for the scrutinee's ADT even when the
      user hasn't imported that ADT's module directly.
- [ ] New assertion in `match_foreign_constructor_function_call_test.go` proves it.
- [ ] No regression in existing M-MATCH-ADT-XCHECK tests.

## Out of scope (unchanged)

- Auto-importing transitive constructors into scope.
- Changing the `MatchForeignConstructorError` message format.
- Option B's import-hint text (a possible follow-up once the registry exists).

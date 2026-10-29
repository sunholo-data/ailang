# M-IMPORT-VS-IMPORT-COLLISION: two imports binding the same bare name resolve silently (last-wins) with no diagnostic

**Status**: Planned
**Target**: v0.54.20
**Priority**: P1 (High-Medium)
**Estimated**: 6 days (3 honest × 2 convention)
**Dependencies**: None (extends the MOD015 checker shipped in v0.52.0, #1467)

## Problem Statement

**Reported (v0.54.18, darwin arm64, task `task-e8942a7d` / stapledons-godot `sim/core.ail`):**
importing the same type name from two modules is accepted silently; the **last** import wins
for every bare use, with **no MOD015-style diagnostic**. The reporter's minimal repro
(`package repro/dup`, `Leg` exported by both `./a` (4 fields) and `./b` (6 fields)):

```ailang
module repro/dup/main
import ./b (Leg)
import ./a (Leg)
export pure func mk() -> Leg = { id: 1, from: 0.0, port: 1.0, t0: 0.0, legT: 2.0, flying: true }
```

`ailang check --package .` → `type unification failed at [return type annotation …]: record field
mismatch: expected 6 fields, got 4` — a downstream symptom in an *unrelated function*, naming
neither import. Reordering the imports makes the same program text pass. In the real-world
incident, a new `import ./earth_link (Leg)` was shadowed by an existing
`import ./navigation (Leg)`; the only symptom was a field-count mismatch elsewhere, and the
human fixed it by renaming their type to `LinkLeg`.

**Current state (mechanism, verified in code — see Verification Log rows 1–4):**

- Explicit selective imports (`import M (x)`, `import M (y as x)`, the list of `import M as L (x)`)
  resolve by **plain map assignment in import order** in `resolveSelectiveImports`
  (`internal/pipeline/pipeline_module_imports.go:203`): `GlobalRefs[bindName]`,
  `ImportedTypeAliases[bindName]`, `ImportedAliasOrigin[bindName]`, `ImportedAliasParams[bindName]`,
  `ImportedCtorInfos[bindName]` — the **last** import silently overwrites every earlier bind of
  the same bare name, in all three namespaces (values, types, constructors).
- The elaborator mirrors this: `internal/elaborate/file.go` writes `symbols[bindName]` per import
  (last wins) and `globalEnv[bindName]` (last wins) before any body elaborates.
- `checkImportCollisions` (`internal/elaborate/import_collision.go:37`) enforces MOD015 for
  **import vs local** only. Its own header comment and the #1467 design doc
  (`design_docs/implemented/v0_52_0/m-elaborator-lexical-scope.md`, scope decision 5) say so
  explicitly: *"Import vs import is NOT covered … It was assumed to be an error already; it is not."*
- The error-code registry **already documents this case as MOD015**: `internal/errors/codes.go:295`
  — "Imported name collides with a module-level definition **or another import**". The description
  has been aspirational since v0.52.0; the enforcement does not exist.

**The class is broader than the report (probed live, v0.54.19 — Verification Log rows 5–15):**

| Namespace | Explicit import vs explicit import | Used bare? | Qualified escape hatch | Today |
|---|---|---|---|---|
| Types | silent **last**-wins | always (no qualified type syntax — PAR error) | alias only (`T as U`) | repro: confusing downstream type error |
| Constructors | silent last-wins for explicit binds; **first**-wins for auto-imported | always (no qualified ctor syntax — `A.On` → "undefined variable: A") | alias only | confusing match-arm / ADT errors |
| Values | silent last-wins | usually | **yes**: `A.fa` via `globalEnv["A.fa"]` (`internal/elaborate/expr_data.go:37`) | compiles; runs the wrong function |

Both silent channels exist in **both pipelines** (`ailang check --package .` and single-file
`ailang check main.ail`): both call `ElaborateFile`
(`internal/pipeline/pipeline_module_phases.go:362`, `internal/pipeline/pipeline_single.go:181`),
and both hit the same resolution order.

**Impact:** AI-generated code (the language's stated audience) merges import lists mechanically
and cannot see that reordering an import changes the program's meaning. The failure surfaces as a
type error in a *different function*, costing a model (or human) a long diagnostic detour — the
reporter's own fix was a rename discovered by hand. MOD015 exists precisely to make this class of
silent shadowing loud (2026-10-02 ruling: error, not warning); import-vs-import is the remaining
member of that class, and its registry description already promises it.

## Goals

**Primary Goal:** Two explicit selective imports that bind the same bare name to different
exports are rejected with a MOD015 diagnostic naming **both import sites**, in every namespace
where a bare use can resolve silently — types and constructors at the import site (they have no
qualified form), values at the bare-use site (their qualified form is legal and must stay legal).

**Success Metrics:**
- The reported repro fails with `Error MOD015` naming both `import ./b (Leg)` and `import ./a (Leg)`,
  in both import orders.
- Constructor repro (two modules exporting the same ctor name, both explicitly imported) fails with
  MOD015 naming both imports, instead of a downstream match-arm ADT error.
- A bare use of a value name bound by two explicit imports fails with MOD015 at the use, naming both
  import sites.
- Zero behavior change over the in-repo corpus: all 778 `.ail` files check identically before/after
  (sweep evidence: 0 explicit type/ctor duplicate binds; the single value collision
  `tools/motoko/r8_headroom_band.ail` binds `length` twice and never uses it bare).
- All existing MOD015 / MOD007 / resolution fixtures still pass (`internal/elaborate/import_collision_test.go`,
  `internal/lsp/import_collision_test.go`, `internal/diag/footgun_fixtures_test.go`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Import-vs-import is a hard **error**, not a warning, and **reuses MOD015** (no new code) | The 2026-10-02 MOD015 ruling chose error over warning for the sibling case; a warning would let eval models keep generating programs that resolve to the wrong binding. A new code (MOD016 is free, verified) would split one failure class across two codes for no reader benefit — the registry description has claimed this case as MOD015 since v0.52.0 | compiler (semantics precedent) + this doc | design | med |
| **Types + constructors: error at the import site; values: error at the bare-use site (Haskell-style "ambiguous occurrence")** | Types/ctors have no qualified form (PAR error / undefined-variable, probed), so import-site ≡ use-site for them and the check is trivial; values have a live qualified-use idiom (`import std/list as List (length)` + `import std/string as Str (length)`, probed P11 + corpus hit r8) that a blunt import-site error would break | compiler (resolution semantics) + this doc | design | high |
| **Auto-import channels (M-TYPE-ALIAS alias pull, M-CTOR-AUTO ctor pull) stay silent this sprint** and are ruled as follow-up M3 | Same failure class but a different mechanism (first-write-wins `exists`-guard, no import line names the symbol); fixing it means origin *sets* in the pipeline/typechecker alias resolution (`ImportedAliasOrigin` keeps only the winner today) and is constrained by a passing in-repo counterexample (`examples/alias_body_closure/main.ail`: explicit `(Item)` from `./b` must keep beating `./a`'s auto-imported `Item`) | human (scope cut) | design | high |
| A **local type declaration suppresses the type import-vs-import error** for that name | Local types win over imported types by design (M-TYPE-NAME-SHADOW); with a local winner, both imports are dead and no bare use can be ambiguous — erroring would change a currently-passing, deterministic case | compiler | design | low |
| The benign-collision key is **(source module path, exported name)** per namespace | `import ./a (Leg)` twice, or `import ./a (Leg)` + `import ./a as P (Leg)`, bind the same export and are already documented benign (docs/docs/reference/errors/mod015.md "What is not an error"); only binds of *different* exports collide. Same module + different exports via aliases (`import ./a (x as Leg)` + `import ./a (Leg)`) DO collide | compiler | design | low |
| Value-ambiguity state is carried **on the Elaborator** (per-name bind list) and consulted at the existing bare-resolution branch | The `*ast.Identifier` case in `internal/elaborate/expressions.go:39` is the single point a bare value name resolves (`globalEnv[name]`, after `inScope` and nullary-ctor checks); qualified uses resolve through a separate key (`"A.fa"`, `expr_data.go`) and local binders through `inScope` — riding the exact branch that would silently pick the last import keeps shadowing semantics untouched | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Error not warning; reuse MOD015 (registry description already covers the case; MOD016 verified free but unused)
- [x] Import-site for types/ctors, use-site for values (probes P5/P13 vs P11; corpus sweep rows)
- [x] Auto-import channels deferred to M3 with the `alias_body_closure` constraint recorded
- [x] Local-type shadow suppresses the type import-vs-import error
- [x] Benign key is (source path, exported name) per namespace
- [x] M2 (value use-site) may land in the same release or the next one — sprint-planner decides sequencing; both are ruled here

## Solution Design

### Overview

Extend the existing MOD015 machinery in `internal/elaborate` — one checker, one error code, three
namespaces, two detection points:

1. **M1 (import-site, types + constructors):** `checkImportCollisions` gains a second pass over
   the `binds` it already receives: group binds by (bare name, namespace-kind); a name bound by ≥2
   binds whose `(path, orig)` pairs differ is a collision → MOD015 naming **both import lines**,
   positioned at the **second** (today's silent winner). Type imports must first become visible to
   the checker: `binds` currently skips them entirely (`GetExport` returns nil for types and the
   collector `continue`s) — add an `importType` kind, classified via the loaded module's
   `Types` map, mirroring the existing `importIsConstructor` (`Load(path)` + map lookup).
2. **M2 (use-site, values):** the elaborator keeps a per-name list of value `importBinding`s; the
   bare-resolution branch in `expressions.go` (the `globalEnv[ex.Name]` lookup) errors with MOD015
   naming both import sites when the name is bound by ≥2 distinct `(path, orig)` sources. Qualified
   (`A.fa`), locally-shadowed (`inScope`), and never-used-bare cases are untouched by construction.

The pipeline needs no new check: `resolveModuleImports`/`resolveSelectiveImports` are the *writers*
of the last-wins maps; the elaborator detects the ambiguity from the AST + loaded interfaces, in
both single-file and package mode, before any body elaborates.

### Architecture

**Components:**

1. **Bind classification** (`internal/elaborate/file.go`, import loop): for each selective symbol,
   classify into namespaces — value (`GetExport` non-nil), constructor (`importIsConstructor`), type
   (new: `Load(path).Types[sym]`). A symbol can produce binds in **multiple** namespaces (a module
   may export a type `Leg` and a func `Leg`; namespaces are independent — probed P12). All binds land
   in the existing `binds []importBinding` with a new `importType` kind; positions are the existing
   `imp.Pos`.
2. **Import-vs-import pass** (`internal/elaborate/import_collision.go`): after the existing
   import-vs-local pass, group `binds` by (name, kind). For each group with ≥2 distinct `(path, orig)`
   pairs → MOD015. For `importType` groups: suppress when a local `type` declaration of that name
   exists (local wins by design — M-TYPE-NAME-SHADOW; both imports are dead). Message shape:

   ```
   Error MOD015: 'Leg' is imported twice: import ./b (Leg) at main.ail:3:1 and import ./a (Leg) at
   main.ail:4:1 — two imports may not bind the same name; a bare 'Leg' is ambiguous (the second
   import silently wins today).
     Fix: alias one import: import ./a (Leg as aLeg), or drop one list entry
   ```

   (reuses `importText`/`aliasSuggestion`/`sitePos`; fix line must remain a pasteable import — same
   contract as `TestMOD015AliasSuggestion`).
3. **Value use-site ambiguity** (`internal/elaborate/core.go` + `expressions.go`): an
   `importBindsByValue map[string][]importBinding` set from `file.go` before any decl body elaborates
   (after the import loop, alongside the existing `checkImportCollisions` call — which for value
   binds does **not** return an import-vs-import error, only records). In the `*ast.Identifier`
   case, at the point that would consult `globalEnv[name]`: if the name has ≥2 binds with distinct
   `(path, orig)` → MOD015 at **the use**, naming both import sites, fix suggests aliasing one
   import or qualifying (`A.fa`). Order-sensitive branches above it (`ResolveAsBuiltin`, `inScope`,
   nullary-ctor) are untouched — the error fires exactly where a bare use would today silently pick
   the last import.
4. **Check ordering** (determinism): MOD007 (duplicate local bindings) → MOD015 import-vs-local →
   MOD015 import-vs-import (types/ctors). The value use-site check fires during body elaboration,
   i.e. after all import-time checks. This ordering is part of the contract and gets a fixture per
   pair (a file with both a local collision and an import-import collision reports the local one
   first, matching today's behavior for the local pair).

### Implementation Plan

**Phase 1: M1 — types + constructors at the import site** (~1.5 days)
- [ ] `file.go`: `importType` kind + `importIsType` helper (`Load(path)` → `mod.Types`), keep the
      existing ctor classification; a symbol may add binds in more than one namespace
- [ ] `import_collision.go`: import-vs-import pass (group by name+kind, `(path, orig)` key, local-type
      suppression), message + fix builders reusing `importText`/`aliasSuggestion`
- [ ] Unit tests: repro both orders; same-module/same-export benign (P7 shape); same module different
      exports via alias (must collide); type+value same name from different modules benign (P12);
      local type + two type imports (suppressed); ctor collision (P10 shape); MOD007/MOD015 ordering
- [ ] Package-mode + single-file-mode fixtures (both pipelines call `ElaborateFile`)

**Phase 2: M2 — values at the use site** (~2 days)
- [ ] `core.go`: `importBindsByValue` field + setter; `file.go`: populate from `binds` (value kind only)
- [ ] `expressions.go`: the ambiguity error at the bare `globalEnv` branch
- [ ] Unit tests: bare use of doubly-imported value errors naming both sites (P3 shape); qualified
      `A.fa` + `B.fa` still legal (P11 shape); dead duplicate import still legal (r8_headroom_band
      shape); local binder shadows (inScope) legal; use inside a lambda/record literal/match body
      covered (the identifier branch is shared)
- [ ] `internal/lsp/import_collision_test.go` sibling: the new variant places the diagnostic
      (import-site variant at the second import; use-site variant at the use)

**Phase 3: corpus, docs, release** (~1.5 days)
- [ ] `ailang check` over the full repo corpus before/after; diff must be empty (778 files; sweep
      predicts zero changes — the prediction is itself a success criterion)
- [ ] `docs/docs/reference/errors/mod015.md`: extend "What this error means" + example with the
      import-vs-import case (import-site and use-site variants); keep "The same export imported
      twice from the same module" in the not-an-error list (now enforced by the `(path, orig)` key)
- [ ] `docs/docs/reference/modules.md`: "Resolving Name Conflicts" / MOD015 section — two modules
      exporting the same name must be aliased; bare use of a doubly-imported name is an error
- [ ] `internal/errors/codes.go`: MOD015 registry description stays accurate (it already promises
      this case; no new code)
- [ ] CHANGELOG `[Unreleased]` entry under Fixed

### Files to Modify/Create

**New files:**
- `internal/elaborate/import_vs_import_test.go` (~250 LOC) — M1+M2 fixtures (or extend
  `import_collision_test.go`; agent's choice)

**Modified files:**
- `internal/elaborate/file.go` (+~35 LOC) — `importType` classification, ambiguity handoff
- `internal/elaborate/import_collision.go` (+~120 LOC) — import-vs-import pass, message/fix builders
- `internal/elaborate/core.go` (+~10 LOC) — per-name value-bind list field
- `internal/elaborate/expressions.go` (+~25 LOC) — use-site ambiguity branch
- `docs/docs/reference/errors/mod015.md`, `docs/docs/reference/modules.md` (docs)
- `changelogs/v0.32-current.md` (Unreleased entry)

## Examples

### Example 1: the reported repro (types), before/after

**Before (v0.54.19, probed):**
```
$ ailang check --package .
  ✗ main.ail — type unification failed at [return type annotation at main.ail:6:8]:
    record field mismatch: expected 6 fields, got 4 …
```

**After:**
```
Error MOD015: 'Leg' is imported twice: import ./b (Leg) at main.ail:3:1 and
import ./a (Leg) at main.ail:4:1 — two imports may not bind the same name; a bare 'Leg'
is ambiguous (the second import silently wins today).
  Fix: alias one import: import ./a (Leg as aLeg), or drop one list entry
```

### Example 2: constructors, before/after

**Before (probed P10):** `import ./a (On)` + `import ./b (On)`, both modules exporting an ADT ctor
`On` — compiles past imports, then dies in the match:
`at constructor pattern Off: match arm constructor 'Off' belongs to ADT 'Wing', not 'Link'`.

**After:** MOD015 at `main.ail` import line, naming both `(On)` imports; fix aliases one.

### Example 3: values — the qualified idiom stays legal, the bare use does not

**Legal (unchanged, probed P11 + corpus):**
```ailang
import std/list as List (length)
import std/string as Str (length)
export pure func n(s: string) -> int = Str.length(s) + List.length([1, 2])
```

**Now an error (was silent last-wins, probed P3):**
```ailang
import ./a (fa)
import ./b (fa)
export pure func mk() -> float = fa(3.0)   -- Error MOD015: 'fa' is bound by two imports
                                           -- (import ./a (fa) at 2:1, import ./b (fa) at 3:1);
                                           -- a bare 'fa' is ambiguous. Qualify (A.fa / B.fa)
                                           -- or alias one import.
```

**Still legal (probed P4, P7, P12):** `import ./a (Leg as ALeg)` + `import ./b (Leg)`;
re-importing the same export twice; a type `Leg` from one module + a value `Leg` from another
(namespaces are independent); a local `type Leg` next to imported `Leg`s (local wins).

## Success Criteria

- [ ] Reported repro (both import orders) → MOD015 naming both imports; no downstream type error
- [ ] Ctor repro (P10 shape) → MOD015 naming both imports
- [ ] Bare use of a doubly-bound value (P3 shape) → MOD015 at the use naming both imports
- [ ] Qualified value idiom (P11 shape) still compiles; dead duplicate import (r8 shape) still compiles
- [ ] Same-export re-imports, cross-namespace same-name, aliased imports, local-type shadowing all compile
- [ ] Existing MOD015 (import-vs-local) and MOD007 fixtures pass unchanged
- [ ] `ailang check` corpus diff (all 778 in-repo `.ail` files) is empty
- [ ] `docs/docs/reference/errors/mod015.md` + `modules.md` updated; CHANGELOG entry added
- [ ] All tests passing (`make test`; new tests in `internal/elaborate`)

## Conflict Surface

### Syntactic positions touched

- The **import binding table** (`binds`) built in `internal/elaborate/file.go`'s import loop — the
  same table MOD015's import-vs-local pass consumes. Today's occupants: value binds (`GetExport`
  non-nil), constructor binds (`decl == nil && importIsConstructor`), and *skipped* type binds
  (`decl == nil && !ctor → continue`). The change adds `importType` and allows one symbol to emit
  binds in several namespaces.
- The **identifier resolution chain** in `internal/elaborate/expressions.go:39` (`*ast.Identifier`):
  `ResolveAsBuiltin` → `inScope` (lexical shadowing) → nullary-ctor → `globalEnv[name]` → local Var.
  M2 adds exactly one branch at the `globalEnv` point. Other occupants of that position: every bare
  identifier in every expression form (calls, record literals, lambdas, match bodies, contracts).
- `checkImportCollisions`'s error precedence: MOD007 fires first (already), then MOD015
  import-vs-local, then the new import-vs-import pass.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| `binds` entry | value import | `import M (x)` / `import M (y as x)` / list of `import M as L (x)` — binds bare `x` |
| `binds` entry | ctor import | same forms; classified via `mod.Constructors` |
| `binds` entry (new) | type import | same forms; classified via `mod.Types` — note `GetExport` returns nil for BOTH types and ctors, so classification must be per-namespace lookup, not nil-ness |
| `globalEnv` bare lookup | builtin | `AddBuiltinsToGlobalEnv` writes after imports in package mode (order probed in code) |
| `globalEnv` bare lookup | lexical binder shadowing | `inScope` check runs first (#1467) |
| `globalEnv` bare lookup | nullary ctor | `e.constructors[name]` checked before `globalEnv` |
| Identifier | qualified use | `A.fa` parses as `RecordAccess`, resolves via `globalEnv["A.fa"]` (`expr_data.go:37`) — a *different key*, so M2 cannot touch it |

### Disambiguation strategy

- Namespace separation: collision is checked **per namespace** (value/ctor/type maps are separate
  today; P12 proves type `Leg` + value `Leg` from different modules is legal and must stay so).
- The `(path, orig)` key decides benign vs colliding: same source module + same exported name in the
  same namespace = the same binding (P7), not a collision.
- Local-type suppression: a local `type` declaration of the name makes both type imports dead
  (M-TYPE-NAME-SHADOW) — no bare use can be ambiguous, so no error.
- M2 rides the exact branch that would silently resolve (`globalEnv`), after `inScope` and ctor
  checks — so a name that resolves as a local or a ctor never reaches the ambiguity error.
- No grammar or token positions change: this is resolution-time diagnostics only; the parser and
  lexer are untouched.

### Programs that MUST still work (regression fixtures)

1. `examples/alias_body_closure/main.ail` — explicit `(Item)` from `./b` + `./a`'s auto-imported
   `Item` used via `Box`/`mk`/`count`; bare `Item` means b's (explicit beats auto — untouched by M1,
   which sees only one explicit bind; M3's future ruling must preserve this).
2. `tools/motoko/r8_headroom_band.ail` — `length` bound by `std/list` and `std/string as Str`
   imports, never used bare (M2 use-site ⇒ legal).
3. `std/ai/streaming.ail` — the #1467 corpus fix (`import std/stream as Stream`, no selective list):
   binds no bare names.
4. The MOD015 fixtures in `internal/elaborate/import_collision_test.go` and
   `internal/lsp/import_collision_test.go` (import-vs-local must keep its positions and message).
5. Any `std/` module importing two modules that both export the same type name but never binding it
   explicitly (auto channel — untouched this sprint).

### What deliberately changes

- The reported shape (`import ./b (Leg)` + `import ./a (Leg)`) becomes a **compile error** in both
  orders. It compiled in every released version; migration = alias one import
  (`import ./a (Leg as aLeg)`) or drop one list entry — the exact workaround the reporter used by
  hand. No other previously-compiling in-repo program is predicted to change (corpus sweep); any
  that do are regressions, not intentional changes.
- A bare use of a value bound by two explicit imports becomes a compile error (was silent
  last-wins). Qualified uses and unused binds are unaffected.

## Testing Strategy

**Unit tests (`internal/elaborate`):**
- Bind classification: type / ctor / value / multi-namespace symbol; alias handling (`y as x`
  binds `x`); `(path, orig)` keying.
- M1 message content: both import sites named, positioned at the second import, fix line is a
  pasteable import (mirror `TestMOD015AliasSuggestion`), local-type suppression, MOD007-then-MOD015
  ordering.
- M2: error at the use (position = the identifier), both sites named; no error for
  qualified/dead/shadowed binds.

**Integration tests:**
- Package pipeline (`ailang check --package .` on a fixture package) and single-file pipeline
  (both call `ElaborateFile` — one fixture each).
- LSP: the import-site variant surfaces as a positioned Error diagnostic at the second import;
  the use-site variant at the identifier (extend `internal/lsp/import_collision_test.go`).

**Regression-surface tests (from Conflict Surface):**
- One test per "Programs that MUST still work" entry: `alias_body_closure`, r8-style dead bind,
  streaming-style alias-only import, existing MOD015/LSP fixtures — plus the full-corpus
  `ailang check` diff (must be empty).

**Manual testing:**
- Reported repro both orders → MOD015; flip order → MOD015 (not a pass).
- `ailang repl` unaffected (import handling unchanged).

## Verification Log

Every load-bearing premise was checked live on v0.54.19 (`/usr/local/bin/ailang`, commit a6b4c2c3,
built 2026-10-27) or by reading the cited code. Probes P0–P15 ran as real packages
(`ailang check --package .`); scripts and transcripts in the task session.

| # | Claim | Check | Result |
|---|---|---|---|
| 1 | Explicit imports resolve by plain map writes, last wins, all namespaces | read `internal/pipeline/pipeline_module_imports.go:203-260` (`GlobalRefs[bindName]`, `ImportedTypeAliases[bindName]`, …) | Confirmed — no cross-import check exists in the writer |
| 2 | Elaborator skips type binds; `binds` covers values + ctors only | read `internal/elaborate/file.go:232-246` (`decl == nil && !importIsConstructor → continue`); `importKind` has no type case (`import_collision.go:13-16`) | Confirmed |
| 3 | MOD015 checks import-vs-local only; import-vs-import explicitly deferred | read `import_collision.go:37-56` header + `m-elaborator-lexical-scope.md` scope decision (5) | Confirmed |
| 4 | MOD015's registry description already covers "or another import" | `internal/errors/codes.go:295` | Confirmed (aspirational since v0.52.0) |
| 5 | P0 repro fails with a downstream type error naming neither import | live `ailang check --package .` on the reporter's exact 3-file package | Confirmed ("expected 6 fields, got 4") |
| 6 | Import order alone flips the verdict (last wins) | P1: same program, imports flipped → passes | Confirmed |
| 7 | Local type vs imported type compiles (local wins) | P2 live | Confirmed (intended, M-TYPE-NAME-SHADOW — kept legal) |
| 8 | Duplicate **value** imports compile silently | P3 live (`fa` from two modules) | Confirmed |
| 9 | Aliased imports are a working escape hatch | P4 live (`Leg as ALeg` + `Leg`) | Confirmed |
| 10 | **No qualified type syntax exists** | P5 live: `A.Leg` in type position → PAR_UNEXPECTED_TOKEN | Confirmed — types are bare-only |
| 11 | MOD015 import-vs-local still fires (control) | P6 live | Confirmed |
| 12 | Same export imported twice is benign | P7 live (`import ./a (Leg)` twice) | Confirmed |
| 13 | Wildcard imports don't exist | P9 live: `import ./a` → IMP012 (supports "explicit selective imports only" scope) | Confirmed |
| 14 | Duplicate ctor imports fail downstream only (match-arm ADT error) | P10 live | Confirmed |
| 15 | Qualified **value** use works (`A.fa`); the idiom is real | P11 live + corpus hit `tools/motoko/r8_headroom_band.ail` (imported `length` twice, never used bare) | Confirmed — forbids blunt import-site error for values |
| 16 | Type and value namespaces are independent | P12 live (type `Leg` from a + value `Leg` from b compiles) | Confirmed |
| 17 | **No qualified ctor syntax**: `A.On(3.0)` → "undefined variable: A" | P13 live | Confirmed — ctors are bare-only |
| 18 | Auto-imported type aliases bind types bare without any list entry | P14 live: `import ./a (fa)` then bare `-> Leg` in the same module compiles | Confirmed — M-TYPE-ALIAS auto channel exists (deferred to M3) |
| 19 | Auto channel is **first**-import-wins (opposite of explicit) | P15a/P15b live: imports a-then-b vs b-then-a flip which `Leg` a bare use resolves to | Confirmed |
| 20 | Corpus: 0 explicit type/ctor duplicate binds in 778 in-repo `.ail` files; 1 value duplicate (r8, dead) | static sweep (explicit import lists, grouped by bind name) | Confirmed — M1 breaks nothing in-repo |
| 21 | Corpus: 1 auto-channel type ambiguity shape, and it is a *passing, intentional* example | sweep #2 + read `examples/alias_body_closure/main.ail` (explicit `(Item)` must beat auto) | Confirmed — recorded as M3 constraint |
| 22 | `MOD016` unallocated | `grep -rn "MOD016" internal/ cmd/ std/` → empty (MOD008–009 reserved, MOD010–015 live) | Confirmed free — and unused by this design |
| 23 | Both pipelines call `ElaborateFile` (elaborator-level check covers both) | `pipeline_module_phases.go:362`, `pipeline_single.go:181`; single-file repro verified live | Confirmed |
| 24 | `LoadedModule` exposes `Types` for classification | `internal/loader/loader.go:53` (`Types map[string]*ast.TypeDecl`), same `Load`+map pattern as `importIsConstructor` | Confirmed |
| 25 | Bare value resolution is a single point; qualified uses a different key | `expressions.go:39-77` (`globalEnv[ex.Name]`), `expr_data.go:37-48` (`globalEnv["A.fa"]`) | Confirmed — M2 is one branch |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact message wording beyond the required content (both import sites, the winning-today note, a
  pasteable fix) — agent may choose.
- Test file placement (extend `import_collision_test.go` vs new `import_vs_import_test.go`) — agent
  may choose.
- Whether M2 ships in the same release as M1 or the next — sprint-planner sequences; both rulings
  are fixed here.
- Whether the import-site type/ctor error also lists *all* colliding imports when ≥3 bind the name
  (report the first pair + a count, vs. every pair) — agent may choose; one error per name either way.
- M3 (auto-channel use-site detection: origin sets on `ImportedAliasOrigin`, explicit-beats-auto
  precedence, use-site error in typechecker alias resolution) — deferred to its own doc once M1/M2
  land; the `alias_body_closure` constraint and P15 evidence carry over.

## Non-Goals

- **Auto-import ambiguity (M3)** — the M-TYPE-ALIAS / M-CTOR-AUTO first-write-wins channels stay
  silent this sprint (see High-Impact Decisions row 3). Evidence and constraints are recorded above.
- **Wildcard imports** — do not exist (IMP012); nothing to check.
- **Builtin-vs-import collisions** — the #1467 deferred decision ("Whether MOD015 also covers
  builtin/constructor collisions") stays open; unchanged here.
- **REPL cross-statement rebinding / registry persistence** — #1467 non-goal, unchanged.
- **Import cycle/path resolution changes** — `resolveModuleImports` is untouched as a writer;
  only the elaborator gains read-side checks.
- **Warnings infrastructure** — the ruling is error-only, matching MOD015's precedent.

## Timeline

**Week 1** (2 days): Phase 1 (M1 types+ctors, import-site) — classification, pass, message,
unit tests, both pipelines.
**Week 2** (2.5 days): Phase 2 (M2 values, use-site) — Elaborator state, branch, tests, LSP fixture.
**Week 3** (1.5 days): Phase 3 — corpus diff, docs (`mod015.md`, `modules.md`), CHANGELOG, release
chores.

**Total: ~6 days across 3 weeks** (2× the honest 3-day estimate, per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M2's use-site branch fires in a resolution position that also serves ctor/builtin names | High | The branch rides the exact `globalEnv` lookup, after `inScope` and nullary-ctor checks; fixtures pin each shadowing case (P11/P12/P7 shapes) as regression tests |
| An external corpus (motoko, stapledons-godot) relies on explicit duplicate binds that ARE used bare | Med | The #1467 corpus sweep named `phase_vocab.ail` / `digests.ail` as alias+qualified uses (not bare) — M2 is use-site, so they pass; M1 is types/ctors only, and the #1467 sweep found no type duplicates. Release notes carry the migration (alias one import) |
| Local-type suppression or `(path, orig)` keying misses a benign shape and errors a working program | Med | Corpus diff over 778 files is a hard success criterion; any diff is a regression, not churn |
| Import-site check misclassifies a symbol that is BOTH type and ctor/value | Med | Classification adds binds per namespace independently (P12 fixture); tests cover the mixed symbol |
| MOD015 message consumers (LSP, `--json`) break on the new message variant | Low | The structured `Report` shape is reused unchanged (code/span/data/fix); only message text differs; LSP fixture added |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a silent import-order dependence: today the *order of import lines* decides which binding every bare use resolves to; after, any ambiguity is a loud, position-named error |
| A2: Replayability | 0 | No trace/execution changes |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | A module's bare names become locally checkable against its own import list (the file itself names every colliding binding) instead of requiring knowledge of import resolution order |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The core cost of the bug: a model merging import lists gets a precise, coded, positioned error naming both imports, instead of a field-count mismatch in an unrelated function requiring a rename hunt |
| A8: Minimal Syntax | 0 | No new syntax; diagnostics only |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | 0 | Import forms unchanged; aliasing escape hatch unchanged |
| A11: Structured Failure | +1 | Reuses the structured `errors.Report` (code/span/data/fix) so `--json` and the LSP surface the collision like every MOD015 |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward** (≥ +2)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — the check reads only the AST + loaded interfaces; iteration over bind groups is deterministically ordered (document order)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience; the diagnostic is coded, positioned, and machine-parseable (this is the machine-decidability fix)

## Related Documents

**Implemented (may inform design):**
- [m-elaborator-lexical-scope (MOD015, #1467, v0.52.0)](../implemented/v0_52_0/m-elaborator-lexical-scope.md) — the sibling case (import vs local), shipped as an error by ruling 2026-10-02. Its scope decision (5) is the deferral this document resolves: "Import vs import is NOT covered … it needs its own ruling (error only on a bare use, Haskell-style, is the natural candidate)". This doc supplies that ruling, with the types/ctors split the deferral did not anticipate (they have no qualified form, so they error at the import site instead).
- [m-type-name-shadow-and-cache (M-TYPE-NAME-SHADOW)](../planned/v0_44_0/m-type-name-shadow-and-cache.md) — local-type-wins-over-imported rule; audit row 3 deferred "import ambiguity diagnostics" — the type-side constraint this design preserves (local-type suppression) and M3's remaining surface.
- [MOD015 user doc](../../../docs/docs/reference/errors/mod015.md) — updated by this design (its "What is not an error" list already states the benign rules M1 enforces).

**Planned (check for overlap):**
- (none — `ailang docs search` (SimHash + neural) over "import vs import collision" returned no matches; the grep-based survey above is the complete related set)

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Reporter's task: `task-e8942a7d` (stapledons-godot `sim/core.ail` incident; binary v0.54.18, cd33d050)
- Mechanisms cited: M-TYPE-ALIAS, M-CTOR-AUTO, M-TYPE-NAME-SHADOW, M-XMOD-ALIAS-POLY (all tagged in code)
- Precedent: Haskell/Elm "ambiguous occurrence", Rust E0254/E0255 (cited by MOD015, #1467)
- Quorum (optional step, attempted 2026-10-28): every independent reviewer seat absent (oc-kimi-k3 unreachable, claude-sonnet-5 quota, gpt6-1-sol auth, gemini-4-argon unreachable; oc-glm-5-3 recalled as same-vendor) — zero signal, recorded by name in `.ailang/state/mission-quorum/m-import-vs-import-collision-2026-10-28T08-44-12Z.json`; controller in-session verdict **pass** (premises live-verified, P0–P15 + sweeps + code reads). Re-run `ailang design-quorum` from a seated environment before sprint-planning if independent review is wanted.

## Future Work

- **M3**: use-site detection for auto-imported type aliases and constructors (first-write-wins
  channels) — origin sets in the pipeline, explicit-beats-auto precedence, error at the bare use;
  constrained by `examples/alias_body_closure/main.ail` and probed by P14/P15 evidence above.
- Builtin-vs-import collision ruling (the other #1467 deferred decision).
- Consider surfacing import-bind provenance (`ImportedAliasOrigin`-style) in `ailang check --json`
  output for tooling.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28
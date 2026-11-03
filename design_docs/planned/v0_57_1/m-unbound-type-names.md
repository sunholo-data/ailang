# Unbound Type Names in Annotations Must Be Compile Errors (m-unbound-type-names)

**Status**: Planned
**Target**: v0.57.1
**Priority**: P1 (High)
**Estimated**: 3 days
**Dependencies**: None (builds on the import/scope machinery of M-TYPE-NAME-SHADOW and the renamed-type-import fix)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Error is emitted deterministically at the annotation site, from a set assembled in sorted order; no heuristic dependence on compile order |
| A2: Replayability | +1 | No runtime behavior change for accepted programs; rejects before any trace is produced |
| A3: Effect Legibility | 0 | No effect-row changes (effect labels are explicitly out of scope) |
| A4: Explicit Authority | +1 | Type names must be explicitly in scope — the same "no ambient names" principle applied to the type namespace |
| A5: Bounded Verification | +1 | The check is a local, linear AST walk against a finite name set — decidable in one pass |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | A typo in a type name is detected at its line with a structured code + suggestion, instead of surfacing as a distant unification/decode failure an AI agent must reverse-engineer |
| A8: Minimal Syntax | +1 | No new syntax; strictly removes a silent behavior |
| A9: Cost Visibility | 0 | No resource-cost surface |
| A10: Composability | +1 | Modules' exported interfaces can no longer contain phantom type names that break consumers at load time |
| A11: Structured Failure | +1 | New structured error code `TC_UNBOUND_TYPE_001` with position, name, and machine-parsable suggestion, following the TC_TYPE_SHADOW_001 precedent |
| A12: System Boundary | +1 | Cross-module type references now verified at the boundary (import site) rather than guessed |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — the opposite: ambient type names are removed
- [x] A7 (Machines First): Error moves to the cause; suggestions are machine-readable

## Problem Statement

An unknown type name in any annotation position is silently accepted by `ailang check`.
The name becomes an **opaque `TCon` that unifies with nothing** (or, for lowercase names, a
**free type variable that silently makes the annotation polymorphic**), and the failure
surfaces only later and far from the cause — at a call site, at consumer-module load time,
or in `--args-json` decoding. Reported while fixing renamed type imports
(`import M (Row as R)`); reproduces on v0.50, v0.51.0 and **still on v0.57.0**.

**Current State (all verified on v0.57.0, see Verification Log):**

- `export pure func f(r: Nope) -> float = 1.0` — `Nope` declared nowhere — `ailang check` → `✓ No errors found!`
- After `import rt/types (Row as R, getX)`, a signature using the ORIGINAL name `Row`
  (only `R` was imported) still passes check. The renamed-import fix
  (`internal/pipeline/renamed_type_import_test.go`) made `R` *expand*; it did not make
  the out-of-scope `Row` an error.
- `export pure func f(o: Option[float]) -> float = 1.0` with **no import at all** passes.
  Note the asymmetry already visible today: using the *constructor* `Some(1.0)` without
  import correctly fails with `undefined variable: Some` — value names are scope-checked,
  type names are not.
- The hole is not limited to function signatures. All of the following pass today
  (each verified, see Verification Log V5–V10):
  - `export pure func f(x: Nope[int]) -> float` (TypeApp head)
  - `let g: Nope = 1.0` (top-level let annotation)
  - `type X = Nope` (alias body)
  - `type Q = Wrap(Nope)` (ADT constructor field)
  - `let h = func(x: Nope) -> float { 1.0 } in ...` (inner func-expression annotation)
  - `export pure func f(r: {a: Nope}) -> float` (record type field)
  - `export pure func g(x: qq) -> qq = x` — undeclared lowercase name becomes a free type
    variable; the function is silently, implicitly polymorphic with no `g[a]` declaration.

**The late, misleading failure mode (verified end-to-end):**

```
module main_entry
import rt/types (Row as R)
export pure func main(r: Row) -> float = r.x
```
```
$ ailang run -args-json '{"r": {"id":"a","x":1.0}}' main_entry.ail
Error: failed to decode arguments: ARG_DECODE_MISMATCH: expected record with field 'id',
  got object missing field 'id'
  required field 'id' not found in JSON
```
The user is told their JSON is missing a field it plainly has. The actual cause — `Row`
is an opaque `TCon` that never expanded into `{id: string, x: float}` — is not mentioned,
because the check that should have flagged it does not exist. Call-site failures are the
same shape: `f(1.0)` reports `No instance for Fractional[Nope] in scope ... Nope is not
Fractional` at the literal, not `Nope is not a type` at the signature.

**Impact:**
- Primary consumers are AI agents writing AILANG (the language's stated audience): a typo
  (`Strng`, `Float`, `Option` vs `Opt`) or a stale pre-rename name compiles green and
  produces a bewildering downstream diagnostic several steps removed from the mistake.
- Any module whose exported interface carries a phantom `TCon` poisons every consumer:
  the name crosses the module boundary in the interface JSON and fails at consumer load
  or unification time (the exact class of cross-module failure that M-TYPE-NAME-SHADOW
  and m-transitive-alias-env-import were fixing symptoms of).
- Rename-during-import (`Row as R`) leaves the old name silently usable — the reported
  trigger for this issue.

**Root cause (read from the code, not inferred from output):** every AST→internal type
converter has a silent fallthrough for unknown names:

- `internal/types/typechecker.go:192` (`TypeChecker.astTypeToType`, `*ast.SimpleType`
  default branch): `return &TCon{Name: typ.Name}` for any non-builtin name.
- `internal/elaborate/file_funcs.go:333` (`Elaborator.astTypeToInternalType`, same
  position): `return &types.TCon{Name: typ.Name}` after consulting only
  `e.typeAliases`.
- Lowercase names never even reach the fallback: the parser routes any lowercase
  identifier in type position that is not in `ast.builtinTypeNames` to `*ast.TypeVar`
  (`internal/ast/ast_type.go:15-22`), which makes the annotation silently polymorphic.
  That comment documents the exact precedent: "`f(b: bytes) -> bytes` once accepted
  `f(42)`" — fixed for builtins by *enumeration*, not by scoping. The
  enumeration approach cannot cover user/imported type names; this design is the scope
  fix.

## Goals

**Primary Goal:** `ailang check` reports an unknown type name at its annotation site,
with the position, the offending name, close-match suggestions, and import suggestions —
instead of silently manufacturing an opaque constructor or a free type variable.

**Success Metrics:**
1. Every repro in the Problem Statement (V1–V10) fails `ailang check` with exit ≠ 0 and
   a `TC_UNBOUND_TYPE_001` diagnostic naming the module, position, and unknown name.
2. Zero behavior change for the legitimate corpus: `ailang check` on every `.ail` in
   `std/` and `examples/`, `make test`, and the stdlib canary remain green.
3. The renamed-import regression fixture (`internal/pipeline/renamed_type_import_test.go`)
   and the shadow/capture fixtures still produce their *existing, more specific*
   diagnostics (our pass must not mask TC_TYPE_SHADOW_001).
4. Suggestion quality: for `Row`-after-rename and `Option`-without-import, the diagnostic
   names the module that exports the type and the exact import line to add (verified by
   golden-message tests).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Unknown **constructor** names (uppercase, non-builtin, non-type-param) in any annotation position are a **hard check error** (not a warning) | This is the reported bug; anything softer keeps the silent-failure pipeline alive | human | design | med |
| D2: Undeclared **lowercase type variables** in annotations (implicit quantification without `f[a]`) also become a hard error, gated on a corpus scan (Phase 2 below) | Same failure class (typo → silent polymorphism), but pre-1.0 code may rely on implicit quantification; corpus decides | human | design | med |
| D3: The check is a **new pipeline pass over the AST**, not a change inside the `astTypeToType` converters | Converters run in the types/elaborate/iface packages without uniform access to the import tables; the pipeline is the one place the scope set is complete (after `shadowLocalTypeNames`) | agent | design | med |
| D4: Error code `TC_UNBOUND_TYPE_001`, following the namespaced pattern of `TC_TYPE_SHADOW_001` | `TC002` already means unbound *value* variable; conflating them breaks machine consumers | agent | compile | low |
| D5: Suggestions: Levenshtein-close known names (≤ 2 edits) + "exported by direct dep module X, add `import X (name)`" + the currently-in-scope imported type list, capped and sorted | Directly requested in the issue; must stay deterministic | agent | compile | low |
| D6: Captured aliases (`imports.CapturedAliases`) and names managed by TC_TYPE_SHADOW_001 are **exempt** from the unknown-name error | Their use already produces a more precise loud error via `RegisterAliasCapture`; double-reporting would mask it | compiler | compile | low |
| D7: REPL / single-expression snippets are covered in Phase 3 only if cheap; module files (incl. implicit modules) are Phase 1 | REPL annotation use is rare and the REPL path builds a synthetic module without full import tables | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 confirmed: hard error for unknown constructor names in ALL annotation positions
      listed in the Conflict Surface (not just exported signatures).
- [ ] D2 confirmed: hard error for undeclared lowercase annotation vars, **contingent on
      the Phase-2 corpus scan finding no legitimate reliance in std/, examples/, tests,
      and the motoko_agent corpus.** If legitimate reliance is found, Phase 2 ships as a
      warning and escalates in a later version — the sprint executor must stop and
      surface the corpus findings if this fires.
- [ ] D7 confirmed: REPL scope.

## Solution Design

### Overview

Add a single static pass — `validateTypeNameScope` — that walks every `ast.Type`
annotation in a module's AST and verifies that each type *name* (constructor heads,
record/alias/ADT field positions, TypeApp heads) is in scope, and that each type
*variable* is lexically bound. It runs inside `compileFreshModule` immediately after
`shadowLocalTypeNames`, sampling the exact same import tables the elaborator and type
checker will consult, so "in scope for the pass" ≡ "resolvable by the compiler". Any
violation fails the module compile with `TC_UNBOUND_TYPE_001` before type checking runs.

### Architecture

**Components:**

1. **Known-name set assembly** (`internal/pipeline/type_name_scope.go`):
   A name is usable in an annotation in module `m` iff it is one of:
   - a builtin: `ast.BuiltinTypeNames()` (`int float string bool unit char bytes`) plus `()`;
   - a local type declaration: `localTypeNames(file)` (every `*ast.TypeDecl`, exported or
     not — reused from `internal/pipeline/type_name_shadow.go:100`);
   - an imported type alias name: any key of `imports.ImportedTypeAliases` — this table
     already encodes renames as the bound name (`Row as R` registers `R`),
     the auto-import of ALL aliases of every directly imported module
     (`resolveSelectiveImports`, M-TYPE-ALIAS), and transitively-reachable aliases
     (`resolveModuleImports`, M-TRANSITIVE-ALIAS-ENV-IMPORT);
   - a parameterized alias name: any key of `imports.ImportedAliasParams`;
   - an imported ADT name: any key of `imports.ImportedADTTypeParams`, plus the type-name
     values of `imports.ImportedCtorTypes` (constructor → ADT map);
   - a **captured alias** (key of `imports.CapturedAliases`): exempt from the unknown-name
     error — usage is already a loud, more precise `TC_TYPE_SHADOW_001` via
     `RegisterAliasCapture` (D6);
   - a lexically bound type parameter: the enclosing `FuncDecl.TypeParams`, the enclosing
     `TypeDecl.TypeParams`, or a record row variable (`RecordType.Row`), which are
     **type variables, not names**, and follow the lexical rule below.
   Sampling happens **after** `shadowLocalTypeNames` in `compileFreshModule`, so local
   shadowing is already reflected in the tables (local names win; captured aliases are
   already withheld).

2. **AST walk with lexical type-param scope** (same file):
   Walk `file.Decls` and `file.Statements` in source order, carrying a set of in-scope
   type-parameter names:
   - `*ast.FuncDecl`: push `TypeParams` into the scope; check `Params[i].Type`,
     `ReturnType`; recurse into `Body`.
   - `*ast.Lambda` / func expressions: check `Params[i].Type`, `ReturnType`; recurse
     (inherit outer type params — `foldl[a,b]`'s body may annotate `func(acc: b, ...)`,
     which is legal today and must stay legal).
   - `*ast.Let` (and letrec): check the optional `Type` annotation; recurse into
     value/body.
   - `*ast.TypeDecl`: push `Name` (recursive types refer to themselves:
     `type Json = ... JArray([Json])`) and `TypeParams`; walk `Definition` — for
     `*ast.AlgebraicType` every `ConstructorField.Type`; for record/alias bodies every
     field type.
   - Type nodes: `*ast.SimpleType` (name check), `*ast.TypeApp` (head name check + walk
     `Args`), `*ast.RecordType` (walk fields; add `Row` name to the type-var scope),
     `*ast.ListType`/`*ast.ArrayType`/`*ast.TupleType`/`*ast.FuncType` (recurse;
     `FuncType.Effects` are effect **labels** — different namespace, skipped),
     `*ast.TypeVar` (lexical-binding check, Phase 2), `*ast.LabelledType` (M-TAINT-TYPES
     labels are capability labels, not type names — skipped in this pass).
   Determinism: iterate maps via the existing `sortedTypeKeys`/`sortedBoolKeys` helpers
   wherever the pass itself must enumerate (error suggestions).

3. **Diagnostic construction** (`TC_UNBOUND_TYPE_001`):
   Follow the `TC_TYPE_SHADOW_001` precedent: a `const` in the new file, message prefix
   with the code, returned as the module-compile error from `compileFreshModule` (which
   already fails `ailang check`/`ailang run` loudly). Proposed message shape
   (fields stable for machine consumption; wording may be tuned by the executor):

   ```
   TC_UNBOUND_TYPE_001: type 'Nope' is not declared in module 'w' and no import binds it
     at w.ail:2:22 (parameter 'r' of function 'f')
     A type name in an annotation must be a builtin (int, float, string, bool, unit,
     char, bytes), a type declared in this module, or a name brought in by an import.
     Did you mean 'float'?                       <- only when a known name is within
                                                    Levenshtein distance 2
     'Row' is exported by module rt/types - add it: `import rt/types (Row)`
                                                  <- only when a DIRECT import's iface
                                                     exports the type under that name
     Types currently in scope here: (none imported)  <- bounded, sorted
   ```

   The "exported by module X" hint needs the type-name lists of direct-dep interfaces.
   Extend `resolveModuleImports` to also record `DepTypeNames map[string][]string`
   (import path → exported type names, sorted) for suggestion purposes only — it does
   NOT extend scope (a type in a dep you did not import stays an error; the hint tells
   you how to fix it).

4. **Error-code registration**: add `TC_UNBOUND_TYPE_001` to
   `internal/errors/codes.go` (category "typecheck", "scope") and
   `internal/errors/json_encoder.go` so `--json`/`--format agent` output carries it.
   Verified unallocated (V12).

### Implementation Plan

**Phase 1: Constructor-name pass + diagnostics** (~1 day)
- [ ] New `internal/pipeline/type_name_scope.go`: set assembly, AST walk, error
      construction (SimpleType + TypeApp heads + all annotation positions).
- [ ] Hook in `compileFreshModule` after `shadowLocalTypeNames`
      (`internal/pipeline/pipeline_module_phases.go:355`).
- [ ] `DepTypeNames` collection in `resolveModuleImports` + import suggestion.
- [ ] Levenshtein helper (small, local, deterministic; cap suggestions at 3).
- [ ] Register the code in `internal/errors/codes.go` + `json_encoder.go`.
- [ ] Unit tests: every Problem Statement repro (V1–V10) now errors; message golden
      tests for the two suggestion shapes.

**Phase 2: Type-variable binding rule (D2, corpus-gated)** (~0.5 day)
- [ ] Extend the walk: `*ast.TypeVar` in annotation position must be lexically bound
      (FuncDecl/TypeDecl params, record row vars).
- [ ] **Corpus gate (blocking):** run the rule over all `.ail` in `std/`, `examples/`,
      the repo's test fixtures, and the motoko_agent corpus sample. Any legitimate
      implicit-quantification use found → stop, surface to human (D2 contingency),
      ship the rule as a warning instead.
- [ ] Heuristic pre-scan at design time found no top-level signature in `std/` or
      `examples/` that annotates with an undeclared lowercase name (V13) — the gate
      exists because a grep cannot prove absence for inner lambdas.

**Phase 3: Path coverage + polish** (~1 day)
- [ ] Cover the single-file / implicit-module path (`nomod.ail` repro, V11) — the module
      pipeline already compiles implicit modules through `compileFreshModule`; verify and
      add a test.
- [ ] REPL decision (D7): if the synthetic-module path gives the pass for free, include
      it; otherwise defer with a note.
- [ ] `ailang check --package` full std/ + examples/ sweep; `make test`, `make fmt`,
      `make lint`, `make check-boundaries` clean.
- [ ] Docs: LIMITATIONS.md entry removal if listed; prompt/`ailang docs` mention of the
      diagnostic; changelog entry.

### Files to Modify/Create

**New files:**
- `internal/pipeline/type_name_scope.go` — the pass (set assembly, walk, diagnostics), ~250 LOC
- `internal/pipeline/type_name_scope_test.go` — repro + golden-message + regression tests, ~300 LOC

**Modified files:**
- `internal/pipeline/pipeline_module_phases.go` — hook call in `compileFreshModule`, ~5 LOC
- `internal/pipeline/pipeline_module_imports.go` — `DepTypeNames` collection, ~20 LOC
- `internal/errors/codes.go` — code registration, ~3 LOC
- `internal/errors/json_encoder.go` — encoder entry, ~2 LOC
- `docs/LIMITATIONS.md` — only if it currently lists this gap, ~2 LOC
- `changelogs/v0.32-current.md` (or the active changelog at ship time), ~10 LOC

Deliberately NOT modified: `internal/types/typechecker.go` and
`internal/elaborate/file_funcs.go` converter fallbacks stay as the last line of defense
(internally-constructed ASTs and the iface/REPL paths still flow through them); the
pipeline pass front-runs them for all real source files.

## Examples

### Example 1: The reported bug

**Before:**
```
$ ailang check w.ail            # module w; export pure func f(r: Nope) -> float = 1.0
✓ No errors found!
```
**After:**
```
Error TC_UNBOUND_TYPE_001: type 'Nope' is not declared in module 'w' and no import binds it
  at w.ail:2:22 (parameter 'r' of function 'f')
  ...
```

### Example 2: Stale name after a renamed import

**Before:** `import rt/types (Row as R, getX)` + `export pure func viaRow(r: Row) -> float = getX(r)`
→ `✓ No errors found!` (then `--args-json` decode fails misleadingly at run time).
**After:**
```
Error TC_UNBOUND_TYPE_001: type 'Row' is not declared in module 'main_orig' and no import binds it
  at main_orig.ail:3:31 (parameter 'r' of function 'viaRow')
  'Row' is exported by module rt/types - add it: `import rt/types (Row)`
  Types currently in scope here: R
```
(Using `R` continues to type-check — the renamed_type_import fixture is a MUST-still-work test.)

### Example 3: Typo in a type name

**Before (verified, V26):**
```
module ex3
export pure func len2(s: Strng) -> int = 1
export pure func use() -> int = len2("hello")
```
`ailang check` passes on the signature; the error appears only at the CALL SITE, pointing
at line 3 — not at the typo on line 2:
```
Error: type unification failed at [function application of len2 at ex3.ail:3:37]:
  failed to unify parameter 0: cannot unify type constructors: Strng vs string
```
**After:** `TC_UNBOUND_TYPE_001` at ex3.ail:2:23 (parameter 's' of function 'len2'),
`Did you mean 'string'?` — before the call site is even considered.

## Success Criteria

- [ ] All ten verified repro positions (V1–V10, plus V11 module-less) fail check with
      `TC_UNBOUND_TYPE_001`, exit ≠ 0, name + position in the message.
- [ ] `internal/pipeline/type_name_scope_test.go` covers: each annotation position, both
      suggestion shapes (typo → close match; stale name → import hint), exemption of
      captured aliases (D6), and lexical type params (`map[a,b]` inner lambda).
- [ ] Regression-surface tests (below) all pass; `renamed_type_import_test.go` unchanged
      and green.
- [ ] Full corpus green: `ailang check` per file and `--package` over `std/` and
      `examples/`; `make test`; stdlib canary.
- [ ] Phase-2 corpus gate executed with results recorded in the sprint report (pass, or
      warning-downgrade decision surfaced to human per D2).
- [ ] All tests passing; documentation updated (error-code reference, prompt/doc mention).
- [ ] Engine parity note: this is a check-time-only change — evaluator and strict VM
      behavior of accepted programs is untouched, so no new parity fixtures are owed.

## Testing Strategy

**Unit tests:**
- Set assembly: renames registered under bindName; transitive aliases count; local names
  shadow imports; captured aliases exempt; `()`/`unit`/`char` builtins accepted.
- Walk: each annotation position (param, return, inner func/lambda, let, alias body, ADT
  field, record field, TypeApp head/args, list/array/tuple elements) with an unknown name
  errors with the right position.
- Lexical scope: `func map[a, b](...)` with inner `func(acc: b, x: a) -> b` passes;
  `qq` unbound errors in Phase 2 mode.
- Suggestions: Levenshtein ≤ 2 threshold, cap 3, sorted output; no suggestion section
  when nothing is close (deterministic golden messages).

**Integration tests:**
- Two-module package: renamed import (R works, Row errors with import hint) — extends
  `internal/pipeline/renamed_type_import_test.go`'s fixture shape.
- `--args-json` end-to-end: the entry-module case from the Problem Statement now fails at
  `check` time; after adding the import, run succeeds and decodes.
- Shadow interplay: a module that declares local `Row` while importing `rt/types`
  (Row as R) still gets `TC_TYPE_SHADOW_001`-class messages where applicable, never
  `TC_UNBOUND_TYPE_001` for the same name.

**Regression-surface tests (one per "Programs that MUST still work" fixture below):**
- `std/array.ail` — `Array[a]` structural syntax + imported `Option` (checks the
  TypeApp-head and import-table paths compile clean).
- `std/list.ail` `map[a, b]` — declared type params + inner annotations.
- `std/json.ail` — local recursive ADT (`type Json = ... [Json] ...`).
- `examples/deriving_eq.ail` — local sum types + annotated top-level lets.
- `examples/datetime_demo.ail` — imported `Option[int]` in a signature.
- `internal/pipeline/renamed_type_import_test.go` fixture — `R` (renamed) in signature.
Each becomes a pinned test in `type_name_scope_test.go`; failures are regressions.

**Manual testing:**
- Re-run the original issue's two repros (`w.ail`, `Row` after rename) — both now error.
- `ailang check --package` on a real package (e.g. sunholo repos' shape) with
  `--format agent` to confirm machine-readable output.

## Conflict Surface

### Syntactic positions touched

This is a **new validation pass**, not a parser/grammar change. The positions it reads
(everywhere an `ast.Type` annotation can appear):
- `FuncDecl.Params[i].Type`, `FuncDecl.ReturnType` (and the same fields on
  func-expression lambdas);
- `Let.Type` annotations (top-level and nested);
- `TypeDecl.Definition`: alias bodies, record fields, `AlgebraicType` constructor fields;
- type nodes: `SimpleType.Name`, `TypeApp.Constructor` + `Args`,
  `RecordType.Fields/Row`, `ListType.Element`, `ArrayType.Element`,
  `TupleType.Elements`, `FuncType.Params/Return`;
- NOT touched: `FuncType.Effects` (effect-label namespace), `LabelledType` labels
  (taint-label namespace), expression AST (no type-ascription syntax exists — verified
  against `ailang prompt` and the parser surface, V14).

### What else lives here

| Position | Existing valid claimants | Why the pass must not flag them |
|----------|--------------------------|--------------------------------|
| `SimpleType` name | builtins (`int…bytes`, `unit`, `()`) | accepted by enumeration |
| `SimpleType` name | local `TypeDecl` of any kind (record, alias, sum, newtype — incl. self-recursive `Json`) | `localTypeNames` covers all `TypeDecl`s |
| `SimpleType` / `TypeApp` head | imported alias under its **bound** name (`R`) | `ImportedTypeAliases` keyed by bindName |
| `SimpleType` / `TypeApp` head | auto-imported alias of a directly imported module (import anything from `rt/types`, its `Row` alias is usable) | `resolveSelectiveImports` M-TYPE-ALIAS block |
| `SimpleType` head | transitively-reachable alias (module C imports B, B's types use A's `Inner`) | M-TRANSITIVE-ALIAS-ENV-IMPORT block |
| `SimpleType` head | imported ADT name + local ADT names | `ImportedADTTypeParams`, `ImportedCtorTypes` values |
| `SimpleType` head | captured alias | exempt (D6) — dedicated TC_TYPE_SHADOW_001 |
| `TypeVar` | `FuncDecl.TypeParams` (`[a, b]`), `TypeDecl.TypeParams`, record row vars (`..r`) | lexical scope, pushed during walk |
| TypeApp heads that are NOT names | `Array[T]` is its own AST node (`ArrayType`), `[T]` is `ListType` | never reach the name check |
| `$builtin` simple names in `ExternalTypes` | builtin module value exports | values, not annotation-reachable types (qualified keys don't parse in annotation position — `S.Set[...]` does not parse per `ailang prompt` line 688) |

### Disambiguation strategy

There is no parser ambiguity to resolve — the pass is a read-only semantic walk. The one
"disambiguation" is namespace discipline: a name is checked only in *type* namespaces
(constructor names + type variables); effect labels and taint labels are skipped because
they live in different namespaces with their own resolution machinery.

### Programs that MUST still work

(all paths verified to exist, V15):
1. `std/array.ail` — `export pure func empty[a]() -> Array[a]` (structural `Array[...]`,
   declared `[a]`, imported `Option` in same file).
2. `std/list.ail` — `export pure func map[a, b](f: a -> b, xs: [a]) -> [b]`.
3. `std/json.ail` — `export type Json = JNull | ... | JArray([Json]) ...` (self-recursive
   local ADT with constructor fields).
4. `examples/deriving_eq.ail` — local sum types (`Red`/`Blue`/`Green`, shapes) and
   annotated top-level `let colorTest: bool = ...` bindings.
5. `examples/datetime_demo.ail` — `func showParseResult(label: string, result: Option[int])`
   with `Option` imported from std/option.
6. The `renamed_type_import_test.go` fixture — `import pkg_a/types (Row as R, getX)` with
   `viaR(r: R)` in the signature.

### What deliberately changes

- Any annotation using an unknown uppercase type name now fails check. This is the fix;
  such programs were previously "valid" only in the sense that their defect was silent.
- Any annotation using an undeclared lowercase type var fails (Phase 2, corpus-gated) —
  i.e., implicit quantification without `f[a]` is no longer accepted in module code.
  Migration: declare the parameters (`func g[a](x: a) -> a`). If the corpus gate finds
  legitimate reliance, this downgrades to a warning for one version (D2 contingency).
- No syntax is added or removed; no error-code reuse.

## Testing Strategy (fixtures summary)

See Testing Strategy above — regression-surface list is authoritative.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording/format of the suggestion sections (agent may tune; golden tests pin
  structure, not prose).
- Whether the pass reports **all** unknown names in a module or stops at the first
  (agent may choose; recommendation: collect all, deduplicate by name, report first
  position per name — bounded by 10 like other batched diagnostics).
- Whether REPL snippets get the pass (agent decides per D7 based on what the synthetic
  module path yields for free).
- Levenshtein implementation detail: an unexported `levenshtein` plus a typo-
  suggestion wrapper already exist in `internal/coordinator/unrouted_bounce.go`
  (`SuggestInbox`, V16), but they are coordinator-scoped and tuned for inbox names
  (decorated-prefix acceptance, tight distance cap). The agent may either extract the
  edit-distance core into a shared internal package or re-implement the ~20-LOC helper
  locally in pipeline — choose by which causes less import-boundary churn
  (`make check-boundaries` is the arbiter).

## Non-Goals

**Not attempted in this feature:**
- Module-qualified type syntax (`S.Set[...]`) — tracked separately (the
  M-TYPE-NAME-SHADOW line; this pass actually strengthens the case for it by making
  unqualified misuse loud).
- Effect-label and taint-label scoping checks (different namespaces; nothing reported
  broken there today).
- Validating names in `internal/iface` JSON beyond what source-level checking prevents
  (stale interfaces from older compiler versions are invalidated by the existing build
  fingerprint in the cache key; no change needed — V17).
- Constructor-*value* scoping (already handled: `undefined variable: Some`).
- Removing the silent `TCon` fallback in the converter internals (kept as defense-in-depth
  for internally constructed ASTs).

## Timeline

**Week 1** (3 days):
- Phase 1: pass + diagnostics + unit/golden tests (day 1)
- Phase 2: type-var rule + corpus gate (half day)
- Phase 3: path coverage, corpus sweep, docs, review buffer (day 2–3)

**Total: ~3 days** (estimate doubled from the naive 1.5 per skill convention).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A legitimate name source is missed → false positives on real code (e.g. a table consulted by the elaborator but not sampled by the pass) | High | The known-set is defined as *exactly the tables the elaborator/typechecker consult*, sampled at the same point; the corpus sweep (std + examples + package check) is a blocking acceptance criterion; anything that slips becomes a one-table fix, not a redesign |
| Phase 2 breaks user code relying on implicit quantification | Med | D2 corpus gate is blocking; contingency ships a warning first; the heuristic scan already found no top-level reliance in std/examples (V13) |
| Double-reporting masks the more precise TC_TYPE_SHADOW_001 capture error | Med | D6 exemption of `CapturedAliases` keys + an explicit interplay test |
| Tests that intentionally use unknown type names to trigger OTHER errors (unification messages, Fractional[Nope]-style) now trip earlier with a different code | Med | Pre-sprint grep for such fixtures (only `renamed_type_import_test.go` matched "Nope"/unexpandable in pipeline tests, V18); the executor re-runs `make test` and may adjust those tests' *expectations* (intentional change, listed in Conflict Surface) |
| REPL/synthetic-module path behaves differently from module path | Low | D7: REPL explicitly deferred or included-by-convenience, tested whichever way |

## Related Documents

**Implemented / planned (auto-search found none on "unbound type names"; manual routing
below — the neural search gap is expected since this failure mode was never designed):**
- [m-type-name-shadow-and-cache.md](../v0_44_0/m-type-name-shadow-and-cache.md) (planned) — the shadow/capture machinery this pass samples; its CapturedAliases interplay motivates D6
- [m-transitive-alias-env-import.md](../../implemented/v0_22_0/m-transitive-alias-env-import.md) (implemented v0.22.0) — why transitive aliases are in scope and must be accepted
- [m-cross-module-type-alias.md](../../implemented/v0_28_1/m-cross-module-type-alias.md) (implemented v0.28.1) — cross-module alias unification background
- Renamed-type-import fix (no standalone doc; changelog + `internal/pipeline/renamed_type_import_test.go`, stapledons_godot 2026-10-01) — the immediate trigger for this issue; its "R stays an opaque constructor" failure mode is the same silent-TCon disease this pass eradicates at check time

**Planned (check for overlap):**
- None found by SimHash or neural search; manual scan of pipeline/type-area docs found no
  coverage of unknown-annotation-name scoping.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Reproducer transcripts: Verification Log below (all commands re-runnable)
- `internal/pipeline/type_name_shadow.go` — scope-table precedent and helpers reused
- `internal/ast/ast_type.go` — builtin-name enumeration precedent and the documented
  "silently parsed as a type variable" hazard

## Verification Log

Every "AILANG does/doesn't" claim above carries a live check (ailang v0.57.0, commit
b23cadc, 2026-11-03). Commands are from a scratch dir; two-module cases use a non-tmp
root (`~/rt`) because /tmp triggers MOD010 auto-relax and LDR001.

| # | Claim | Command / source read | Result |
|---|-------|----------------------|--------|
| V1 | Unknown type in exported signature passes check | `ailang check` on `module w / export pure func f(r: Nope) -> float = 1.0` | `✓ No errors found!` exit 0 — BUG confirmed on v0.57.0 |
| V2 | Original name after renamed import passes | `ailang check main_orig.ail` (`import rt/types (Row as R, getX)`, sig uses `Row`) | `✓ No errors found!` — BUG confirmed |
| V3 | Renamed name works today (must-not-regress) | `ailang check main_ok.ail` (sig uses `R`) | `✓ No errors found!` (positive) |
| V4 | `--args-json` fails far from cause | `ailang run -args-json '{"r":{"id":"a","x":1.0}}' main_entry.ail` (sig `r: Row`) | `ARG_DECODE_MISMATCH: expected record with field 'id'` — misleading error confirmed |
| V5 | `Nope[int]` (TypeApp head) passes | `ailang check` t1 | `✓ No errors found!` |
| V6 | `let g: Nope = 1.0` top-level passes | `ailang check` t2b | `✓ No errors found!` |
| V7 | `type X = Nope` alias body passes | `ailang check` t3 | `✓ No errors found!` |
| V8 | `type Q = Wrap(Nope)` ADT field passes | `ailang check` t4 | `✓ No errors found!` |
| V9 | Inner func-expression annotation passes | `ailang check` t5b (`func(x: Nope) -> float { 1.0 }`) | `✓ No errors found!` |
| V10 | Record type field `{a: Nope}` passes; undeclared lowercase var passes | `ailang check` t8, lc (`x: qq`) | `✓ No errors found!` both |
| V11 | Module-less file hole | `ailang check nomod.ail` (`let g: Nope = 1.0`, no module decl) | `✓ No errors found!` (module-less file with exports is already MOD014) |
| V12 | `TC_UNBOUND_TYPE_001` unallocated | `grep -rn "TC_UNBOUND" internal/` ; full TC-code enumeration (`TC001–TC010`, `TC_ALIAS_ARITY_001`, `TC_ARITY_001`, `TC_MATCH_001`, `TC_REC_001-004`, `TC_INT_001`, `TC_TYPE_SHADOW_001`) | empty grep = free; picked next namespaced code |
| V13 | No top-level std/examples signature annotates with an undeclared lowercase name | heuristic grep of `std/*.ail`, `examples/*.ail` for bracket-less funcs with non-builtin lowercase annotations | 0 hits (heuristic only — Phase-2 corpus gate is the real arbiter) |
| V14 | No expression type-ascription syntax exists | `ailang prompt` + parser surface scan | no ascription construct documented/parsed |
| V15 | Regression fixtures exist | `ls std/array.ail std/list.ail std/json.ail examples/deriving_eq.ail examples/datetime_demo.ail internal/pipeline/renamed_type_import_test.go` | all exist |
| V16 | Levenshtein prior art | grep for Levenshtein/Suggest across internal/ | `internal/coordinator/unrouted_bounce.go:74` has an unexported `levenshtein` + `SuggestInbox` (typo suggestions, tight threshold); apiserver route_suggest.go uses prefix-overlap scoring. No shared/exported helper — extraction vs local re-implementation is deferred to the executor |
| V17 | Stale module-cache verdicts invalidated on new binaries | read `internal/pipeline/pipeline_module_cache.go:54-84` (`currentCompilerIdentity`/`compilerIdentity`) and `build_fingerprint_unix.go` | cache key = commit + (dirty-build fingerprint) + config flags; a new binary never serves old verdicts, so no cache-migration work is owed (mechanism read, not inferred) |
| V18 | No test fixtures besides the renamed-import test rely on unknown type names to trigger other errors | grep `Nope\|Unexpandable` over `internal/pipeline/*_test.go internal/types/*_test.go` | only `renamed_type_import_test.go` matched |
| V19 | Root cause: silent TCon fallback in both converters | read `internal/types/typechecker.go:192-206`, `internal/elaborate/file_funcs.go:333-355` | both have `default: return &TCon{Name}` |
| V20 | Parser routes lowercase non-builtins to TypeVar (silently polymorphic) | read `internal/ast/ast_type.go:15-22` (builtinTypeNames comment documents the `f(b: bytes)` precedent) | confirmed |
| V21 | Scope tables: renames keyed by bound name; ALL aliases of direct deps auto-imported; transitive aliases pulled into scope | read `resolveSelectiveImports` + `resolveModuleImports` in `internal/pipeline/pipeline_module_imports.go` | confirmed — the known-set definition mirrors these tables |
| V22 | Captured aliases get a dedicated loud error on use (must not be double-reported) | read `internal/pipeline/type_name_shadow.go:50-95` + `RegisterAliasCapture` loop in `pipeline_module_compile.go:142-144` | confirmed — D6 |
| V23 | Prelude injects only `println` (no extra type names in entry modules) | read `internal/pipeline/prelude.go` preludeSurface | confirmed |
| V24 | `Option` with NO import passes while constructor `Some` correctly errors | `ailang check` m_none / m_auto | `✓ No errors found!` / `undefined variable: Some` — the value/type asymmetry |
| V25 | Namespace imports (`import std/option`) unsupported | `ailang check` m_wopt | `IMP012_UNSUPPORTED_NAMESPACE` — no hidden scope source via namespace imports |
| V26 | Typo case: signature passes, error appears only at the call site, pointing at the call not the annotation | `ailang check` ex3.ail (`func len2(s: Strng) -> int = 1` + `use() = len2("hello")`) | signature line checks green; error is `type unification failed at [function application of len2 at ex3.ail:3:37] ... cannot unify type constructors: Strng vs string` — the exact misleading shape Example 3 documents |

## Future Work

- Module-qualified type names (`M.Set[int]`) — would make the import-suggestion hint a
  one-token fix instead of an import edit (separate M-TYPE-NAME-SHADOW line).
- Applying the same scope pass to effect labels and taint labels once a defect is
  reported there.
- Surfacing the known-type set to the REPL/agent prompt context (`ailang prompt`
  injection) so models propose in-scope names in the first place.

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03

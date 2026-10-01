# M-CTOR-PATTERN-ALIAS-AND-SCOPE: constructor import aliases are never bound (patterns silently never match; expressions fail "undefined variable"), and a constructor pattern naming an unknown constructor compiles clean and never matches

**Status**: Planned — quorum attempted 2026-10-01 (artifact `.ailang/state/mission-quorum/m-ctor-pattern-alias-and-scope-2026-10-01T22-17-42Z.json`): **both external reviewers absent** (`gpt5-6-sol`: auth; `gemini-3-1-pro`: unreachable), so the quorum degraded to controller-only with the absences recorded by name — NOT quorum-cleared, same degradation as the sibling doc [m-vm-var-pattern-default-arm](../v0_51_2/m-vm-var-pattern-default-arm.md) the same day. The 25-row first-party Verification Log (every claim backed by a live binary run or a source read at cited lines) is the load-bearing evidence; re-run quorum when a reviewer route is available.
**Target**: v0.51.2 (bug fix; pattern-scope soundness family, same clause as [m-vm-var-pattern-default-arm](../v0_51_2/m-vm-var-pattern-default-arm.md))
**Priority**: P0 — four confirmed **silent wrong results** that pass `ailang check` with "No errors found!" on every execution route (evaluator, `--bytecode`, `--strict-bytecode`): exit 0, no error, wrong value. Aliasing is the only in-language fix for two packages exporting the same constructor name (qualified constructor patterns do not parse, V12), so the name-clash scenario has no correct workaround today — stapledons-godot had to build a shim module (`sim/tripphase.ail`) to dodge it.
**Estimated**: ~2 days + 1 day buffer (root cause fully localized to two functions; the alias-binding mechanism for values already exists in the same function and is reused; the unknown-name gate reuses the M-MATCH-ADT-XCHECK error machinery)
**Dependencies**: none. Builds on [m-match-adt-xcheck.md](../implemented/v0_18_10/m-match-adt-xcheck.md) (landed v0.18.10: the foreign-ADT cross-check this doc extends from direct-scope to transitive scope). Sibling of [m-vm-var-pattern-default-arm](../v0_51_2/m-vm-var-pattern-default-arm.md) (planned; both reported from the same stapledons-godot sprint area).
**Reported from**: coordinator task `task-60429428` (stapledons-godot `sim/core.ail`, v0.50.0 binary `2f1193d74ffa`); all repro rows re-verified in this session against the local v0.51.0 build (`b99dd25c`-dirty) — the bug is present unchanged in both.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a silent-wrong-result class uniformly: the fix lives in the shared compile pipeline (import resolution + elaboration + typecheck), so evaluator, bytecode VM, strict VM and Go codegen inherit the same corrected name resolution instead of each guessing. |
| A2: Replayability | 0 | No trace or replay machinery touched. |
| A3: Effect Legibility | 0 | Pure import/elaboration/typecheck changes; no effects involved. |
| A4: Explicit Authority | 0 | No capability or authority surface touched. |
| A5: Bounded Verification | +1 | Constructor-pattern scope resolution is a local, decidable table lookup (direct-scope map → transitive-diagnostic map → error); moves a whole failure class from runtime-never-matches to compile-time-rejected. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | `import M (Ctor as Alias)` is what an AI writes to disambiguate two same-named constructors; today the documented syntax silently does nothing in patterns (and errors confusingly in expressions), and the only workaround is a human-shaped hand-written shim module. |
| A8: Minimal Syntax | +1 | No new syntax — existing, documented import-alias syntax ([modules.md:97-98](/docs/reference/modules)) starts meaning what it already says. |
| A9: Cost Visibility | 0 | No cost accounting changes. |
| A10: Composability | +1 | Constructor aliases compose with the existing M-CTOR-AUTO auto-import and local-declaration precedence rules unchanged. |
| A11: Structured Failure | +1 | Converts three silent never-match shapes into loud structured compile errors with did-you-mean suggestions (reusing the M-MATCH-ADT-XCHECK error format). |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes silent wrong results, introduces none — alias→canonical resolution is a pure function of the import list.
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): removes a human-only workaround requirement.

## Problem Statement

**Current State.** Two related silent failures in constructor patterns, both confirmed on the
reported v0.50.0 binary and re-verified on v0.51.0 (`b99dd25c`-dirty), both runtimes agreeing
and `ailang check` reporting "No errors found!":

```ailang
module alias_ctor
import std/option (Option, Some, None as Nada)
func nothing() -> Option[int] = None
export func aliased() -> int = match nothing() { Nada => 1, _ => 0 }
export func unknown() -> int = match nothing() { Bogus => 1, _ => 0 }
```

```
$ ailang run --quiet --entry aliased alias_ctor.ail   -> 0   (expected 1)   [V1]
$ ailang run --quiet --entry unknown alias_ctor.ail   -> 0   (expected error) [V5]
$ ailang run --quiet --bytecode    --entry aliased …  -> 0   (same)          [V7]
$ ailang check alias_ctor.ail                         -> "No errors found!" [V1,V5]
```

**Verified symptom table** (all rows run in this session; commands and raw outputs in the
Verification Log):

| # | Shape | `ailang check` | Result at runtime | Class |
|---|---|---|---|---|
| V1 | `None as Nada`; pattern `Nada` | ✓ No errors | returns 0, arm never taken | **silent wrong result** |
| V2 | `Some as S`; pattern `S(v)` | ✓ No errors | returns 0, arm never taken | **silent wrong result** |
| V3 | `Nada` in an **expression** | loud: `undefined variable: Nada` | — | alias binds nowhere (at least loud) |
| V4 | `S(3)` constructor call | loud: `undefined variable: S` | — | same |
| V5 | unknown nullary `Bogus` in pattern | ✓ No errors | returns 0, arm never taken | **silent wrong result** |
| V6 | unknown `Bogus(v)` in pattern | ✓ No errors | returns 0, arm never taken | **silent wrong result** |
| V7 | V1 under `--bytecode` and `--strict-bytecode` | ✓ No errors | returns 0 | silent on every route |
| V8 | `#323` family: only `std/list (nth)` imported, pattern `None`/`Some(v)` on its `Option` return | ✓ No errors | **works correctly today** | must keep working |
| V9 | transitive-foreign: same file, `Err(e)` arm against the `Option` scrutinee (`std/result` transitively loaded) | ✓ No errors | arm never taken | **silent** — the v0.18.10 foreign-ADT check cannot fire |
| V10 | ctor name clash, both imported (`Dup` in `moda.Phase` and `modb.Journey`), pattern on second module's value | loud: `match arm constructor 'Dup' belongs to ADT 'Phase', not 'Journey'` | — | works; must keep |
| V11 | clash in an expression (`Dup` meant for `modb`) | loud unification error `Phase vs Journey` | — | works |
| V12 | qualified constructor pattern `O.Some(v)` | PAR_UNEXPECTED_TOKEN (does not parse) | — | out of scope, see Non-goals |
| V13 | type-symbol alias `import std/option (Option as Opt)`; `Opt[int]` used | confusing: `type constructor mismatch: Option vs Opt` | — | same root cause, types; out of scope, follow-up |

**Why it matters.** Two packages can export constructors with the same name (the reported
case: `sunholo/relativity journey.TripPhase.Arrived` vs the sim's own `Journey.Arrived`).
Qualified constructor patterns (`J.Arrived`) do not parse (V12), so aliasing is the natural
in-language fix — and it fails silently: a match that should take the `Arrived` arm falls
through to the wildcard with zero diagnostics. A typo'd constructor name fails the same way.
Together with match exhaustiveness not being checked (M-MATCH-EXHAUSTIVENESS, not yet
planned — see [m-match-adt-xcheck.md](../implemented/v0_18_10/m-match-adt-xcheck.md)), a
misspelled arm is invisible to the compiler. The only current workaround is the one
stapledons-godot actually shipped: a hand-written one-function shim module
(`sim/tripphase.ail`) that imports the clashing constructors unaliased and maps them to an
int, so the core module never names them.

**Root cause** (three links in one chain, all read and cited):

1. **Constructor import aliases are dropped at import resolution.**
   `resolveSelectiveImports` (`internal/pipeline/pipeline_module_imports.go:196-283`)
   computes the alias binding `bindName` and uses it for value/function imports
   (`imports.GlobalRefs[bindName] = item.Ref`, line ~219), but the constructor branch calls
   `resolveConstructorImport(sym, ctor, imports, cfg)` (line ~253) with the **original**
   symbol, never the alias. The alias is therefore registered nowhere: not in
   `GlobalRefs` (hence V3/V4's "undefined variable"), not in `ImportedCtorTypes`, not in
   `ImportedCtorInfos`. This is the *only* call site that registers explicitly-imported
   constructors (V17).
2. **Pattern elaboration falls back to "match by written name".** In
   `elaboratePattern` (`internal/elaborate/patterns.go:70-127`), an uppercase identifier
   that is *not* in `e.constructors` is deliberately elaborated as a nullary
   `core.ConstructorPattern{Name: <as-written>}` — the #323 fix, which replaced an even
   worse silent catch-all `VarPattern`. `Nada` (not registered, because of link 1) and
   `Bogus` (never existed) both take this path; patterns with arguments
   (`S(v)`, `Bogus(v)`) keep their as-written name through the
   `*ast.ConstructorPattern` branch (patterns.go:133-176). The runtime then compares by
   tag string: `if tagged.CtorName != p.Name { return nil, false }`
   (`internal/eval/eval_patterns.go:179-196`, V19) — `"None" != "Nada"` → the arm can never
   match.
3. **The typechecker silently skips unknown constructor names.** In `checkPattern`,
   case `*core.ConstructorPattern` (`internal/types/typechecker_patterns.go:146-155`),
   the entire ADT cross-check is guarded by `if adtTypeName, ok :=
   tc.constructorTypes[p.Name]; ok` — a name missing from the map adds **no constraint and
   no error**. The v0.18.10 M-MATCH-ADT-XCHECK foreign-ADT check therefore cannot fire for
   aliases (unregistered) or typos (nonexistent), and V9 shows it also cannot fire for
   genuinely transitive constructors it *could* prove wrong.

**Impact.** Every AI-authored program that aliases a constructor import — the standard fix
for the documented name-clash scenario — gets a silently dead match arm, or a loud but
misleading "undefined variable" if it uses the alias in an expression. Every typo'd
constructor pattern is silently dead. `ailang check` endorses both.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| Alias resolves to the **canonical** constructor name at elaboration (core patterns carry `None`, never `Nada`) | Downstream consumers (typechecker, bytecode compiler, Go codegen, exhaustiveness, dtree) all key on the constructor name; canonicalizing once at the source means zero downstream changes and keeps runtime tag matching intact | Design (this doc) | elaboration | low |
| Unknown-constructor strictness is **tiered**: transitive-known names keep #323 by-name semantics; names known nowhere are a hard error | Strict direct-scope-only would break the documented #323 family (V8) and stdlib-adjacent code paths; tiering preserves every working program while killing every typo | Design (this doc); human ratification welcome at review | typecheck | med |
| Type-symbol aliases (`Option as Opt`, V13) are **out of scope** | Fixing them needs TCon-name aliasing in the unifier — a distinct conflict surface (type environment, not scope tables); bundling would balloon this sprint. Documented here with a repro so the follow-up doc starts from evidence | Design (this doc) | — | — |
| Qualified constructor patterns (`J.Arrived`, V12) are **out of scope** | Parser feature with its own lookahead/ambiguity surface; aliasing covers the motivating clash scenario once it works | Design (this doc) | — | — |

## Conflict Surface

This change touches `internal/pipeline/`, `internal/elaborate/`, and `internal/types/` —
the Conflict Surface section is therefore required.

**1. Syntactic/semantic positions extended:**

- **Selective-import symbol list with `as`** (`import M (sym as alias)`). Other constructs
  already in this position: value/function symbols (aliased correctly today, unchanged),
  type symbols (silently unaliased today — V13, out of scope, unchanged behavior), module
  aliases (`import M as L`, handled on a separate branch, unchanged), and M-CTOR-AUTO
  auto-import of *all* constructors of directly-imported modules under canonical names
  (pipeline_module_imports.go:275-282, unchanged — canonical names always remain available,
  aliases are additional bindings that can coexist with them).
- **Uppercase identifier in pattern position** (nullary constructor or variable?). Other
  constructs in this position: `_` wildcard, lowercase variable patterns (VarPattern),
  `::` cons (special-cased before the general branch, untouched), literal patterns,
  record/list/tuple patterns (different branches, untouched), and the existing loud error
  for a bare non-nullary constructor in a pattern (`Some` without arguments,
  patterns.go:113-118 — must keep firing, now also for aliases).
- **`ConstructorPattern` name resolution in `checkPattern`.** Other consumers of the same
  maps: M-MATCH-ADT-XCHECK (extended, not replaced — tier 2 delegates to the same error),
  factory-scheme constraint tying pattern args to field types (unchanged, keyed on the
  canonical name which is what elaboration now emits), exhaustiveness heuristics
  (Bool-only, untouched), and local-declarations-override-imports precedence
  (pipeline_module_compile.go:102-107, unchanged; alias entries obey the same precedence).

**2. How the parser/typechecker disambiguates:**

- Alias binding applies only when `imp.SymbolAliases[sym]` names the symbol — no change to
  unaliased imports of any kind.
- Pattern-name resolution is a pure three-tier table lookup in a fixed order
  (direct scope → transitive diagnostic map → error); no lookahead, no ambiguity with
  lowercase names (uppercase convention enforced at declaration by
  `PAR_VARIANT_NEEDS_UIDENT`, cited in patterns.go:74).

**3. Programs that MUST still work (regression fixtures, existence verified — V22-V24):**

- `examples/pattern_matching_adt.ail` — imports `std/option (Some, None, ...)` explicitly and
  matches on both (the M-DX24 fixture).
- `examples/prelude_option_result.ail`, `std/option.ail`, `std/result.ail` — plain
  constructor patterns and expressions.
- `internal/elaborate/patterns_nullary_test.go` — `TestNullaryConstructorPattern`,
  `TestNullaryPatternMatching` (the #323 regression tests).
- `internal/pipeline/match_foreign_constructor_test.go` — the M-MATCH-ADT-XCHECK
  regressions (V10's loud path).
- The V8 program (transitive `Some`/`None` with only `std/list` imported) — encoded as a
  new named test in this sprint so the tier-2 promise is executable.
- The V10/V11 clash programs — first-wins import precedence plus the foreign-ADT error
  must keep firing identically, including under `--bytecode`.

**4. What deliberately changes (intentional incompatibilities):**

- Programs that name a constructor in a pattern which no loaded module (direct or
  transitive) defines — `Bogus`, `Nada`-without-registration — now fail `ailang check`
  with a structured error instead of compiling to a silently dead arm. This is the point
  of the sprint; per the V21 survey no stdlib or examples file relies on the old
  behavior (every file that matches on Option/Result constructors imports the defining
  module directly).
- Aliased constructor imports now bind. A file that previously imported
  `None as Nada` and then *also* declared or imported something else named `Nada` would
  previously "work" (the alias was dead); it now collides under the existing
  local-over-import / first-wins rules. No known file does this (V21).

## Solution Design

### Overview

Two coordinated fixes in the shared compile pipeline, so every runtime inherits them:

- **Fix A — bind constructor import aliases** (pipeline + elaborator): register the alias
  name everywhere the canonical name is registered for explicitly-imported constructors,
  keeping the canonical constructor name *inside* the registration, and emit the
  **canonical** name in elaborated core patterns.
- **Fix B — tiered constructor-name resolution for patterns** (typechecker): every
  `core.ConstructorPattern` name is resolved against direct scope, then the transitive
  diagnostic registry, then rejected with a structured error — closing the typo hole and,
  as a bonus, extending the v0.18.10 foreign-ADT cross-check to transitive constructors
  (V9).

### Architecture

**Fix A** (≈45 LOC across three files):

1. `resolveSelectiveImports` (`internal/pipeline/pipeline_module_imports.go`, ctor branch
   at ~line 253): compute `bindName` exactly as the value branch does and call
   `resolveConstructorImport(bindName, ctor, imports, cfg)`. `resolveConstructorImport`
   already builds the factory reference and type scheme from the *scheme's* canonical
   `ctor.CtorName` (lines 285-300), so the alias binds to the canonical factory with no
   changes there. Add the canonical name to `importedCtorInfo` (new field
   `CanonicalName`; today the canonical name *is* the map key, which is why aliasing
   could not work — V17).
2. `compileModule`'s registration loop (`internal/pipeline/pipeline_module_phases.go:334-336`):
   for alias entries call a new
   `RegisterImportedConstructorAlias(typeName, aliasKey, canonicalName, arity, typeParamCount)`
   so `e.constructors` is keyed by the alias while `ConstructorInfo.CtorName` holds the
   canonical name. The elaborator's expression paths already build factory names from
   `ctorInfo.CtorName` (`internal/elaborate/expressions.go:48-57` nullary,
   `expr_calls.go:49-53` calls), so `Nada` and `S(3)` start elaborating to
   `$adt.make_Option_None` / `$adt.make_Option_Some` with no further changes.
3. `elaboratePattern` (`internal/elaborate/patterns.go`): in both the nullary-identifier
   branch (lines 92-98) and the general `*ast.ConstructorPattern` branch (lines 133-176),
   when `e.constructors[p.Name]` exists, emit `core.ConstructorPattern{Name:
   ctorInfo.CtorName}` (canonical) instead of the as-written name. The runtime tag
   comparison (`eval_patterns.go:179-196`, bytecode compiler equivalent) then matches.
   The bare-non-nullary arity error (patterns.go:113-118) is unchanged and now also
   protects aliases (`S` without arguments stays a loud error).

Everything downstream — typechecker factory-scheme constraints, bytecode compilation, Go
codegen, exhaustiveness, dtree — sees only canonical names, exactly as it does today for
unaliased imports.

**Fix B** (≈90 LOC, typechecker + errors):

In `checkPattern`, case `*core.ConstructorPattern` (`internal/types/typechecker_patterns.go:146`),
replace the silent miss with a three-tier ladder:

1. **Tier 1 — direct scope** (`tc.constructorTypes[p.Name]`): unchanged behavior
   (ADT cross-check vs the scrutinee, factory-scheme field constraints). After Fix A,
   canonicalized aliases land here via M-CTOR-AUTO's canonical registration.
2. **Tier 2 — transitive knowledge** (`tc.diagnosticCtorTypes`, populated from ALL
   transitively-loaded module interfaces at `pipeline_module_compile.go:118`; same package,
   direct field access — the map exists and is populated in both `check` and `run`
   pipelines, V16):
   - If the scrutinee's ADT is concretely resolvable (`extractADTName`, same conservative
     rule the xcheck already uses) and differs from the ctor's ADT → fire the existing
     `NewMatchForeignConstructorError` (`internal/types/errors.go:104`). This closes V9:
     the transitive-`Err`-vs-`Option` arm becomes the same loud error users already get
     for directly-imported foreign constructors.
   - Otherwise → accept with no constraint, exactly as today. Runtime keeps #323's
     by-name matching (V8 preserved).
3. **Tier 3 — nowhere known** → new structured error, `match_unknown_constructor`, code
   `TC_MATCH_001` (both grep-verified unallocated, V14/V15), in the style of
   `NewMatchForeignConstructorError`:

   ```
   constructor pattern 'Bogus' does not name any constructor in scope
   (no direct import and no transitively-loaded module defines it).
     Suggestion: if it is a typo, did you mean 'Some' or 'None'?
     Suggestion: constructor aliases must match the import, e.g. import std/option (None as Nada)
   ```

   The did-you-mean list comes from a prefix/closest match over
   `constructorTypes ∪ diagnosticCtorTypes` (the same suggestion shape as the undefined-
   variable import hints, `internal/types/import_hint.go`).

Because the gate sits in the shared typecheck phase, `ailang check`, `ailang run`,
`ailang run --bytecode/--strict-bytecode`, and `ailang compile` all inherit it — confirmed
by V7/V10 (the typecheck phase demonstrably runs on the bytecode route: the V10 xcheck
error fires identically there).

### Implementation plan

| Phase | Tasks | Files | LOC |
|---|---|---|---|
| 1 — Alias binding | bindName in ctor branch; `CanonicalName` in `importedCtorInfo`; alias registration API; canonical emission in both pattern branches | `internal/pipeline/pipeline_module_imports.go`, `internal/pipeline/pipeline_module_phases.go`, `internal/elaborate/core.go`, `internal/elaborate/patterns.go` | ~45 |
| 2 — Scope gate | tier ladder in `checkPattern`; `NewMatchUnknownConstructorError` + kind + `TC_MATCH_001` | `internal/types/typechecker_patterns.go`, `internal/types/errors.go` | ~90 |
| 3 — Tests | alias nullary/args (pattern + expression), unknown nullary/args, transitive-ok (V8 as a named test), transitive-foreign (V9), clash regressions (V10/V11), check/run/bytecode agreement | new `internal/pipeline/ctor_alias_pattern_test.go`; cases in `internal/elaborate/patterns_nullary_test.go` style | ~250 test LOC |
| 4 — Docs | correct/extend [modules.md](/docs/reference/modules) symbol-alias section to state constructor aliasing; note pattern scope rule | `docs/docs/reference/modules.md` | ~10 |

### Examples

Before (all silent or misleading today):

```ailang
import std/option (Some, None as Nada)
match nothing() { Nada => 1, _ => 0 }   -- compiles, returns 0 forever   [V1]
match nothing() { Bogus => 1, _ => 0 } -- compiles, returns 0 forever   [V5]
let x = Nada                            -- "undefined variable: Nada"   [V3]
```

After:

```ailang
import std/option (Some, None as Nada)
match nothing() { Nada => 1, _ => 0 }   -- returns 1; elaborates to ConstructorPattern{Name:"None"}
match nothing() { Bogus => 1, _ => 0 } -- TC_MATCH_001: unknown constructor 'Bogus' at compile time
let x = Nada                            -- binds; elaborates to $adt.make_Option_None
```

The reported name-clash scenario stops needing the shim module:

```ailang
import sunholo/relativity/journey (TripPhase, Arrived as TripArrived)
-- local ADT Journey keeps its own Arrived; patterns name both without collision
```

## Goals

**Primary Goal:** every constructor name appearing in a pattern (or expression) either
resolves to a real, in-scope constructor or fails `ailang check` — zero silent dead arms.

**Success Metrics:**

- The reported repro: `aliased()` returns 1; `unknown()` fails `check` with
  `TC_MATCH_001` naming `Bogus`.
- V1–V9 rows flip to correct/loud; V8 (transitive) unchanged and green as a named test.
- Both runtimes + `--strict-bytecode` agree on every new test (trivially — shared gate).
- `make test`, `make test-core` green; #323 and xcheck regression suites untouched-green.
- stdlib + examples survey (V21) shows zero files newly erroring.

**Non-goals:**

- **Match exhaustiveness checking** — M-MATCH-EXHAUSTIVENESS, not yet planned (see
  [m-match-adt-xcheck.md](../implemented/v0_18_10/m-match-adt-xcheck.md) non-goals). This
  doc makes arms *resolvable*; coverage is the next level.
- **Qualified constructor patterns** (`J.Arrived`) — parser feature, own conflict surface
  (V12 documents today's PAR_UNEXPECTED_TOKEN). Aliasing is the in-scope answer to the
  clash scenario; qualified patterns can be designed separately.
- **Type-symbol aliases** (`Option as Opt`, V13) — same root cause in the type branch of
  `resolveSelectiveImports`, but the fix is unifier TCon-aliasing, a different conflict
  surface. Repro recorded here; propose a follow-up doc if the need recurs.
- **Module-alias qualified access** — unchanged.

## Success Criteria

- [ ] Reported repro: `aliased()` → 1 under `run`, `run --bytecode`, `--strict-bytecode`
- [ ] Reported repro: `unknown()` → `TC_MATCH_001`-structured error from `ailang check`, naming `Bogus`, with suggestion
- [ ] `Some as S` alias: pattern `S(v)` binds and matches; expression `S(3)` constructs
- [ ] Transitive #323 program (V8) unchanged: `Some`/`None` patterns with only `std/list` imported still pass and match
- [ ] Transitive-foreign arm (V9) now rejected with the existing foreign-ADT error
- [ ] Clash regressions (V10/V11) still loud, identical messages, including `--bytecode`
- [ ] `internal/elaborate/patterns_nullary_test.go`, `internal/pipeline/match_foreign_constructor_test.go` green
- [ ] New tests for every row of the symptom table
- [ ] `make test`, `make test-core`, `make check-boundaries` green
- [ ] modules.md symbol-alias section updated
- [ ] All tests passing; documentation updated

## Timeline

- **Day 1**: Fix A (pipeline alias binding + elaborator canonical emission) + alias tests
  (V1-V4 rows). Local `make test-core` loop.
- **Day 2**: Fix B (tier ladder + error kind/code) + scope tests (V5-V9 rows) + clash
  regression runs.
- **Day 3 (buffer, 2x rule)**: full `make test`, stdlib/examples survey re-run, docs,
  changelog entry, quorum re-run if a reviewer route is available.

## Verification Log

Every claim above is backed by a live command run in this session (binary: local build
v0.51.0 `b99dd25c`-dirty, `/usr/local/bin/ailang`; repro files under `/tmp/repro/`,
`/tmp/clash/`) or a source read at cited lines. Reported binary was v0.50.0
`2f1193d74ffa`; V1/V5/V7 behavior is identical on both.

| # | Claim | Evidence |
|---|---|---|
| V1 | `None as Nada`; pattern `Nada`: check clean, `run --entry aliased` → 0 | run: `0`; check: `✓ No errors found!` |
| V2 | `Some as S`; pattern `S(v)`: check clean, run → 0 | `alias_some2.ail`: run `f` → `0` |
| V3 | alias in expression: loud `undefined variable: Nada` | `alias_expr.ail`: `Error: type error ... undefined variable: Nada at ...:3:40` |
| V4 | aliased ctor call: loud `undefined variable: S` | `alias_some.ail`: `undefined variable: S at ...:4:34` |
| V5 | unknown nullary `Bogus` pattern: check clean, run → 0 | repro `unknown()` → `0`; check `✓ No errors found!` |
| V6 | unknown `Bogus(v)` pattern: check clean, run → 0 | `ctor_scope.ail` `g()` → `0`; check clean |
| V7 | `--bytecode` and `--strict-bytecode` agree (→ 0, no error) | both run `aliased` → `0` |
| V8 | #323 family works today and must keep working | `e323.ail` (only `import std/list (nth)`): `match nth(xs,i) { None => ..., Some(v) => v }` — check `✓ No errors found!`; source comment documents the intent at `internal/elaborate/patterns.go:113-127` |
| V9 | transitive-foreign arm is silent today | `e323c.ail`: `Err(e)` arm against Option scrutinee — check `✓ No errors found!` |
| V10 | ctor clash, both imported: loud foreign-ADT error, incl. `--bytecode` | `/tmp/clash/main.ail`: `match arm constructor 'Dup' belongs to ADT 'Phase', not 'Journey' ... Suggestion: did you mean 'B1'?` on `run` and `run --bytecode` |
| V11 | clash expression: loud unification error | `/tmp/clash/main2.ail`: `cannot unify type constructors: Phase vs Journey` |
| V12 | qualified ctor pattern does not parse | `qual.ail`: `PAR_UNEXPECTED_TOKEN ... expected next token to be =>, got . instead` |
| V13 | type-symbol alias ignored → confusing error | `typealias.ail` (`Option as Opt`, then `Opt[int]`): `type constructor mismatch: Option vs Opt` |
| V14 | error code `TC_MATCH_001` unallocated | `grep -rn "TC_MATCH" internal/` → no hits; existing codes are `TC_ALIAS_ARITY_001`, `TC_ARITY_001`, `TC_REC_001-004` (`internal/types/errors.go:344-405`) |
| V15 | error kind `match_unknown_constructor` unallocated | `grep -rn "match_unknown" internal/` → no hits; kinds listed at `errors.go:25-33` |
| V16 | `tc.diagnosticCtorTypes` exists, is populated in the shared pipeline, and has **no getter** (same-package field access suffices for Fix B) | `internal/types/typechecker_core.go:85` (field), `:340` (setter), `:373` (in-package read); populated at `internal/pipeline/pipeline_module_compile.go:118` from `imports.AllCtorTypes`, which is built from ALL transitively-loaded ifaces (`pipeline_module_imports.go:70-82`) |
| V17 | `resolveConstructorImport` is the only registration site for explicitly-imported constructors, and the alias (`bindName`) is not passed to it | read of `resolveSelectiveImports` (`pipeline_module_imports.go:196-283`): value branch uses `bindName` (line ~219); ctor branch at ~line 253 passes `sym`; second call site is the M-CTOR-AUTO loop (line ~281), also canonical-keyed |
| V18 | pattern elaboration emits the **as-written** name for out-of-map uppercase identifiers (#323 fallback) | read of `elaboratePattern` (`internal/elaborate/patterns.go:92-127`); with-args branch keeps `p.Name` (lines 133-176) |
| V19 | runtime matches constructor patterns by tag string | read of `matchPattern` (`internal/eval/eval_patterns.go:179-196`): `if tagged.CtorName != p.Name` |
| V20 | typechecker silently skips unknown ctor names in patterns | read of `checkPattern` (`internal/types/typechecker_patterns.go:146-155`): whole ADT block guarded by `if adtTypeName, ok := tc.constructorTypes[p.Name]; ok`, no else |
| V21 | no stdlib/examples file relies on unimported-ctor patterns; every file matching on Option/Result ctors imports the defining module | survey over `std/*.ail`: files with `None =>` / `Some(..) =>` / `Ok(..) =>` / `Err(..) =>` arms (`ai, embedding, env, json, jwt, list, net, option, regex, result, sem, sharedmem, stream, yaml`) all carry a direct `import std/option` / `import std/result` (or define the ctors themselves); remaining grep hits are comments only (`std/datetime.ail`, `std/net.ail` hits are in doc-comments) |
| V22 | foreign-ADT xcheck regression tests exist | `internal/pipeline/match_foreign_constructor_test.go` (+ `match_foreign_constructor_function_call_test.go`, `scheme_preserve_adt_head_test.go`) |
| V23 | regression fixtures exist | `ls examples/pattern_matching_adt.ail examples/prelude_option_result.ail std/option.ail std/result.ail` → all exist; `pattern_matching_adt.ail` imports `std/option (Some, None, isSome, getOrElse)` and matches both |
| V24 | #323 regression tests exist | `internal/elaborate/patterns_nullary_test.go`: `TestNullaryConstructorPattern` (line 12), `TestNullaryPatternMatching` (line 145) |
| V25 | quorum attempted this session; both external reviewers absent, recorded by name; synthesis degraded to controller-only | `ailang design-quorum ... --reviewers gpt5-6-sol,gemini-3-1-pro --controller-verdict pass`: `gpt5-6-sol` → ABSENT (auth), `gemini-3-1-pro` → ABSENT (unreachable), controller → pass; artifact `.ailang/state/mission-quorum/m-ctor-pattern-alias-and-scope-2026-10-01T22-17-42Z.json`; same reviewer-route absences as the sibling doc's attempt the same day |

## Related Documents

- [m-match-adt-xcheck.md](../implemented/v0_18_10/m-match-adt-xcheck.md) (implemented v0.18.10) — the foreign-ADT cross-check this doc extends to transitive scope (tier 2) and whose error machinery the new unknown-ctor error reuses. Its non-goals explicitly deferred constructor-name scope resolution: *"Constructor name shadowing across modules … we'll prefer the imported / qualified one based on the scope-resolution rule already in place"* — this doc supplies that rule.
- [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md) (planned v0.51.2) — sibling from the same stapledons-godot sprint area; same fail-loud-over-silent-wrong-result clause (A11), disjoint construct class (variable catch-all arms vs constructor-name resolution).
- [m-parser-nullary-single-ctor-cursor.md](../v0_51_1/m-parser-nullary-single-ctor-cursor.md) (planned v0.51.1) — adjacent nullary-constructor parser bug; no overlap (declaration cursor vs pattern scope).
- [m-dx20-wildcard-pattern-inference](../../implemented/v0_6_1/m-dx20-wildcard-pattern-inference.md) (implemented v0.6.1) — wildcard pattern semantics this doc leaves untouched.
- [modules.md](/docs/reference/modules) — documents `import M (sym as alias)` as "direct access with new name" (lines 97-98); this doc makes constructor symbols honor that contract.
- GitHub issue #323 — introduced the by-name fallback this doc preserves for transitive constructors (tier 2) while closing its typo hole (tier 3); regression tests cited in V24.

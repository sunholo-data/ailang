# M-ELABORATOR-LEXICAL-SCOPE: The elaborator resolves identifiers with no lexical scope — imports, builtins, and ADT constructors pre-empt every local binder

**Status**: Implemented (2026-10-02, #1467) — lexical binders; row 7 is now compile error MOD015 (ruling 2026-10-02, see Implementation notes)
**Target**: v0.51.0 (landed for v0.51.1)
**Priority**: P1 (borders P0: the module-level form is a *silent miscompile* — wrong value, no diagnostic)
**Estimated**: 4–5 days
**Dependencies**: None. Fixes the bug family named by #327's "the real resolution fix" and the 2026-09-15 effect-checker backlog row; coordinate sequencing with both.

## Implementation notes (2026-10-02)

**Shipped** — the elaborator keeps a scope stack (`internal/elaborate/scope.go`) consulted after
`ResolveAsBuiltin` and before the constructor table / `globalEnv`, in expression position, at
constructor-call position, and for `Alias.field` qualified access. Frames are pushed for lambda and
function-literal parameters, module function parameters (`funcToLambda`, and around their
`requires`/`ensures` contracts), `let` bodies, `letrec` value + body, match-arm guard + body
(binders read from the elaborated core pattern, so they agree with `elaboratePattern`'s
constructor-vs-binder classification by construction), `forall` bodies, and block statement-lets
via per-index sets (a use textually before `let x = ...;` keeps the outer binding). Matrix rows
1–6, 8, 9 and 10 now produce the "Should be" outcome; tests: `internal/elaborate/scope_test.go`
(per binder kind, plus a scope-leak assertion) and `cmd/ailang/lexical_scope_test.go`
(evaluator + strict VM, values asserted). Corpus sweep (`ailang check` over `std/`, `examples/`,
`cmd/ailang/testdata/`, 493 files) before/after: only the new fixtures change.

**Row 7 — ruled 2026-10-02: a compile error, not "local wins + warning".** The human chose
the Haskell/Elm ("ambiguous occurrence") / Rust E0255 rule over the warning this doc proposed:
a name bound by an explicit selective import — `import M (tick)`, `import M (f as tick)`, or
the selective list of `import M as L (tick)` — that the module also defines as a module-level
`func`, `let`/`letrec`, or ADT constructor is **error MOD015**. The base-frame mechanism
(component 4) and the warning text (component 7) are superseded; nothing resolves to either
binding because the program does not compile. Implementation: `checkImportCollisions`
(`internal/elaborate/import_collision.go`), called once from `ElaborateFile` after the selective
imports are resolved and before any import enters the symbol table — the one place every front
door shares (`check`, `run`, `test`, the REPL module loader, the LSP via `pipeline.Run`). The
error names both sites and both fixes, and unwraps to a structured report (`check --json`,
LSP range at the local definition):

```
Error MOD015: 'tick' is both imported (import ./a (tick) at q.ail:2:1) and defined in this
module as a func (at q.ail:6:6) — an imported name and a module-level definition may not share a name.
  Fix: rename the local definition, or alias the import: import ./a (tick as aTick)
```

Scope decisions: (1) **No wildcard imports exist** — `import M` with no list is a parse error
(IMP012) and `import M as L` is qualified-only, binding no bare names — so the Rust rule "a local
silently shadows a glob import" has nothing to apply to; only explicit selective imports
conflict. (2) **Constructors:** an imported constructor vs a local constructor of the same name
is MOD015 (located at the local type declaration). (3) **Types are unchanged:** a local `type T`
still wins over an imported `T` (M-TYPE-NAME-SHADOW); importing type `T` next to a local
constructor `T` is not a collision (separate namespaces). (4) **Lexical binders** shadowing an
import stay legal (the rest of this doc). (5) **Import vs import is NOT covered:** two selective
imports binding the same bare name to different exports (`import std/list (length)` +
`import std/string (length)`) still compile, the later import winning for bare uses. It was
assumed to be an error already; it is not. The common idiom `import std/list as List (length)`
+ `import std/string as Str (length)` (used qualified) makes an import-time error too blunt —
motoko's `src/core/phase_vocab.ail` and `src/eval/journal/digests.ail` would stop compiling — so
it needs its own ruling (error only on a bare *use*, Haskell-style, is the natural candidate).

Corpus sweep (2026-10-02, `ailang check` per file over all 729 `.ail` files in this repo plus a
static selective-import × module-binding scan of the sibling repos): in-repo, one collision —
`std/ai/streaming.ail` imported `std/stream as Stream (onEvent, runEventLoop, disconnect)` and
re-exported same-named wrappers that call `Stream.onEvent` etc.; fixed by dropping the
selective list (`import std/stream as Stream`). External collisions are listed in the #1467
report, not edited here.

**Strict-VM gaps found (pre-existing, independent of naming):** a bare variable-pattern arm
(`match 9 { x => x }`: "unknown ADT \"\" in switch", see m-vm-var-pattern-default-arm) and an
expression-form `letrec` ("call to unbound name") are evaluator-only on any binder name; those
fixture rows run on the evaluator only.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One name currently has two possible resolutions depending on invisible elaborator-internal table order; after the fix, resolution is a function of the lexical position alone |
| A2: Replayability | 0 | No trace-format change |
| A3: Effect Legibility | 0 | No effect-row change (the sibling effect-checker mis-resolution, backlog 2026-09-15, is a separate fix) |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Restores local reasoning: a binder visible in a region is the binding used in that region — reasoning no longer needs the module's full import list |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Removes a trap that generates false type errors ("cannot unify int with TFunc2" pointing at the shadowed use) and worse, silently wrong answers; models cannot detect the miscompile at all (q.ail below) |
| A8: Minimal Syntax | +1 | No syntax change; removes a position-dependent exception to uniform scoping |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Importing a module can no longer change the meaning of existing local binders — modules compose by addition, not capture |
| A11: Structured Failure | +1 | j.ail (constructor capture) currently *passes* the checker with the wrong semantics; after the fix it fails loudly at the true conflict |
| A12: System Boundary | +1 | Value-level shadowing aligns with the already-shipped type-level rule (M-TYPE-NAME-SHADOW: local type declarations win over same-named imported types) |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — resolution becomes strictly positional
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

**Reported against v0.50.0** (reporter's repro, preserved verbatim as fixture `d.ail`):

```ailang
module rx/p/a
export pure func tick(n: int) -> int = n + 1

module rx/p/d
import ./a (tick)
type R = { tick: int }
pure func mk(f: int -> R) -> R = f(4)
export pure func main() -> int = mk(\tick. {tick: tick}).tick
```

```
→ type error in d (decl 1): type unification failed at [function application at d.ail:5:36]:
  failed to unify parameter 0: failed to unify record field 'tick':
  cannot unify type constructor int with *types.TFunc2
```

The lambda parameter `tick` should shadow the imported `tick` inside the lambda body. It does not: the record-field value `tick` resolves to the *import*. Reporter's workaround: rename the parameter.

**Root cause (verified in code, see Verification Log rows 2–4):** the elaborator (`internal/elaborate/expressions.go`, `*ast.Identifier` case) resolves every identifier by consulting, in order: (1) desugar-synthesized builtins, (2) the nullary-ADT-constructor map, (3) `globalEnv` — which holds **every imported symbol AND every registered builtin** (`AddBuiltinsToGlobalEnv`), and only then (4) falls back to a local `core.Var`. The Elaborator tracks **no lexical scope at all** (no scope stack exists in the struct). Local binders — lambda parameters, function-declaration parameters, `let`/`letrec` names, match-pattern binders, `forall` variables — are never recorded anywhere the elaborator can see, so any name that collides with an import, a builtin, or a constructor is pre-empted *before* the type checker's (correctly lexical) environment is ever consulted.

**Current State — this is not one bug but one root cause with many faces.** All live-reproduced on the reporting binary (v0.50.0-6-g021c46907-dirty):

| # | Program shape | Today | Should be |
|---|---|---|---|
| 1 | Lambda param shadows import (`\tick. {tick: tick}`, the report) | type error (false) | compiles, returns 4 |
| 2 | Lambda param shadows import, simple body (`\tick. tick + 1`) | type error (false) | compiles, returns 8 |
| 3 | `let tick = 5 in tick` with import | type error (false) | compiles, returns 5 |
| 4 | Function-decl param shadows import (`func main(tick: int)`) | type error (false) | compiles |
| 5 | Match binder shadows import (`match x { tick => tick }`) | type error (false) | compiles |
| 6 | Any of the above with a **builtin** name (`show`, `map`, …) — no import needed, builtins are always in `globalEnv` | type error (false) | compiles |
| 7 | **Module-level local func vs same-named import**: local `tick(n)=n+100`, imported `tick(n)=n+1`, call `tick(1)` | **returns 2 — the import — with no diagnostic** | 102; file.go:238-243 documents "Local functions take precedence" and the elaborator violates it |
| 8 | **Silent wrong answer through a lambda**: `mk(\tick. tick)(5)` where `mk : int -> (int -> int)` | **returns 6 (imported tick applied to 5), checker green** | rejected by the checker (param is `int`, not `int -> int`) |
| 9 | Nullary constructor capture: `\None. None` where the body means the param | checker green, body silently resolves to the ADT constructor | checker error (param `int` vs `Option[int]`) — or, with the param used meaningfully, the param |
| 10 | Local record vs module alias: `import std/list as L; let L = {map: 42} in L.map` | type error (resolves `L.map` to the imported `map`) | 42 (field access on the local) |

**Impact:**
- **Silent miscompiles (rows 7–9)** are the severity driver: AILANG's contract is a deterministic substrate where a program's meaning is decidable from its text. Row 7 needs only a module that imports a name and defines a local of the same name — common with generic names (`map`, `show`, `tick`). No error, wrong value.
- **False type errors (rows 1–6, 10)** block the natural idiom and mislead: the diagnostic points at the shadowed *use* site with an "int vs TFunc2" unification message that does not name the capture. AI models burn repair rounds renaming spurious causes.
- The reporter hit this in package code (`rx/p/…` paths) — the shape `import ./sibling (name)` + a lambda param of the same name is ordinary package structure.

**Why now / relation to prior work:** this is the same "resolution diverges from lexical scoping" family as #327 (record-update local-fn resolution, v0.29.0 — whose interim `SetModuleFuncNames` diagnostic is explicitly "retired when the real resolution fix ships") and the 2026-09-15 effect-checker row (let-bound lambda resolved through the name-keyed `declaredEffects` map, `design_docs/planned/ailang-core-triage/effect-checker-let-shadowing.md`). Those two are *type-checker-side* and *effect-checker-side* members; this doc is the **elaborator-side** member and the largest surface. The type level already got the precedence rule (M-TYPE-NAME-SHADOW: local type declarations win over imported same-named types, `shadowLocalTypeNames`); value-level bindings are the remaining inconsistency.

## Goals

**Primary Goal:** identifier resolution in the elaborator is a function of lexical scope: a local binder shadows imported symbols, builtins, and ADT constructors everywhere in its scope region, exactly matching what the type checker's environment already assumes.

**Success Metrics:**
- The reporter's `d.ail` checks green and `ailang run d.ail` prints `4`.
- All ten rows of the matrix above produce the "Should be" outcome (fixtures committed as table-driven tests).
- Zero behavior change on the existing corpus: `make test` green; `ailang check` clean over `std/`, `examples/`, and package testdata (Phase 0 sweep proves no corpus program relied on capture).
- No new "undefined variable" regressions: every fixture from #327's position matrix still green.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Lexical binder shadows **everything user-visible** (imports, builtins, ADT constructors) in expression position — not just imports | Partial fix (imports only) leaves the builtin/constructor faces (rows 6, 9) silently wrong and splits one rule into three | human (semantics) | design | med |
| Constructor-first resolution is **kept in pattern position** (patterns keep `None` = constructor; binders are lowercase per the uppercase convention) | Reversing pattern classification would break #323's fix and the whole ADT corpus | compiler (established convention) | design | low |
| ~~Module-level names are a **base scope frame** seeded from local funcs + module lets, making local module bindings win over imports (row 7)~~ **Superseded 2026-10-02:** the collision is a compile error (MOD015), so neither binding wins | Programs that relied on the import winning at module level now fail to compile (deliberate breaking change) instead of silently changing meaning | human (semantics) | design | high |
| ~~Import-vs-local module-level collision gets a new **warning MOD015**~~ **Ruled 2026-10-02: MOD015 is an error** (Haskell/Elm ambiguous occurrence, Rust E0255) | An error cannot be ignored by an agent that never reads warnings; the fix (rename or alias) is one line | human (semantics) | design | low |
| Block statement-let scoping via **per-index precomputed scope sets** (normalizeBlock threads backwards, so a naive push/pop would give lets the wrong region) | The backwards threading is invisible until it bites; the alternative (rewrite block threading forward) touches record-update desugar consumers | agent (mechanism) | compile | med |
| REPL cross-statement rebinding of builtin/imported names stays **out of scope** | REPL prior bindings reach new statements via the module-registry `globalEnv`, a persistence mechanism, not lexical scope; changing it is a separate semantics decision | human (scope cut) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Human confirms: local binders shadow builtins and ADT constructors too (row 6, 9 semantics), and module-level locals shadow imports (row 7) — i.e., the full lexical rule, not an imports-only patch.
- [ ] Human confirms: MOD015 warning (vs. silence) for module-level import-vs-local collisions.
- [ ] Human confirms the REPL scope cut (last table row) is acceptable for this sprint.

## Conflict Surface

Touches `internal/elaborate/` (mandatory section):

1. **Syntactic/semantic positions extended:** the `*ast.Identifier` resolution inside `normalize` (expression position), the constructor branch of `normalizeFuncCall` (callee position), and the qualified-access branch of `normalizeRecordAccess` (`Alias.field` position). No grammar change; all three positions gain one preliminary question: "is this name lexically bound here?"

2. **What OTHER valid constructs already live in those positions:**
   - Bare nullary constructors as expressions (`None`, `True` for imported/import-free ADTs) — elaborated to `$adt.make_*` factories (`expressions.go:48-58`, #323's fix).
   - Constructor applications `Some(x)` (`expr_calls.go:28-66`) and the `::` cons special-case (`expr_calls.go:19`, `::` can never be a binder name — no conflict).
   - Qualified module-alias access `L.map` (`expr_data.go:43`) — the ONLY way to reach a whole-module import's exports (`collectImports` TODO confirms wildcard imports are not name-exposing today).
   - Desugar-synthesized builtin references (`ResolveAsBuiltin`, e.g. the interpolation desugar's `show` wrapper) — must remain uncapturable; they are checked **before** the new scope check, unchanged.
   - Builtin references from user code (`show(x)` with no local binder) — unchanged (scope miss → `globalEnv` hit).
   - Module-level self/mutual recursion: bodies reference module-level names that are NOT in `globalEnv` today (they elaborate to `core.Var`) — the new base frame must reproduce exactly this set, or recursive calls flip to imports.

3. **Disambiguation:** purely environment-order — scope stack hit ⇒ `core.Var`; else current order (constructor → `globalEnv` → `core.Var`). Pattern position is untouched (`elaboratePattern` keeps constructor-first; uppercase-ident rule and #323 unchanged).

4. **Programs that MUST still work (fixtures — all verified to exist):**
   - `examples/pattern_matching_adt.ail` — imported `Some`/`None` used as expressions and patterns, lowercase binders `h`, `path`, `v` in arm bodies.
   - `examples/match_hof_lambda.ail` — lambdas passed to HOFs alongside imported constructors.
   - `examples/option_pattern_import_free.ail`, `examples/prelude_option_result.ail` — constructor use without/with prelude imports.
   - `std/` sweep: stdlib modules import siblings and define local funcs (the heaviest import+local coexistence corpus); `ailang check` must stay green over the whole tree.
   - The #327 fixture matrix (`internal/pipeline/record_update_positions_test.go` family) — local-fn resolution in record-update fields must not regress.
   - e.ail from this doc (same-module shadowing, returns 4 today) — the proof that downstream is already lexical; must stay green.

5. **Deliberate changes (intentional incompatibilities):**
   - Row 7: a module-level binding with the same name as an explicitly imported one is compile error MOD015 (previously: the import silently won, contradicting file.go's comment). Ruled 2026-10-02; supersedes the "local wins + MOD015 warning" proposal.
   - Rows 1–6: previously-false type errors become green programs.
   - Row 8: a program that previously produced a silently wrong value now fails the type check.
   - Row 9: `\None. None` with the param intended changes meaning from constructor to param; programs *relying* on the old capture fail or change — corpus sweep must show none do.
   - Row 10: `Alias.field` resolves to a local record when `Alias` is lexically bound (previously always the qualified import).

## Solution Design

### Overview

Give the Elaborator a **lexical scope stack** — a `[]map[string]bool` (frame per binder construct) — consulted *before* the constructor/`globalEnv` tables in the three resolution positions. Push a frame wherever a binder is introduced; seed a module base frame from the module's own top-level names. Everything downstream (type checker's `TypeEnv`, evaluator, bytecode VM) is already lexically correct — proven by e.ail — so the fix is confined to the elaborator.

### Architecture

**Components:**

1. **Scope stack on `Elaborator`** (`internal/elaborate/core.go`): `scope []map[string]bool` + `PushScope(names …)/PopScope()/inScope(name) bool`. Cheap: membership in a small map, checked once per identifier node.

2. **Resolution order** (`internal/elaborate/expressions.go`, `*ast.Identifier`):
   1. `ResolveAsBuiltin` (unchanged, first — compiler-synthesized, never capturable)
   2. **NEW: `inScope(name)` ⇒ `core.Var`**
   3. nullary-constructor factory (unchanged)
   4. `globalEnv[name]` ⇒ `core.VarGlobal` (unchanged)
   5. fallback `core.Var` (unchanged)

3. **Binder pushes** (one frame each, popped after the body normalizes):
   - `normalizeLambda` / `normalizeFuncLit` (`expr_simple.go`) — parameter names. This covers **function-declaration parameters** automatically (`elaborateFuncDecl` desugars to `ast.Lambda`, `file_funcs.go:262-277`).
   - `normalizeLet` / `normalizeLetRec` (`expr_control.go`) — the bound name, in scope for the body (and for `letrec`'s own value).
   - `normalizeMatch` (`patterns.go`) — pattern binders around **guard and body**. New helper `patternBinders(pat) []string` must mirror `elaboratePattern`'s classification exactly: ` "_" → none; identifier in `e.constructors` or uppercase → recurse into args; otherwise binder`; recurse through list/tuple/cons/record/constructor patterns.
   - `normalizeForall` (`expr_control.go:332`) — `fa.Var`.
   - `normalizeBlock` (`expr_control.go:237`) — **precompute per-index scope sets**: `scopeByIndex[i] = {statement-let names bound by exprs 0..i-1}` in a forward pre-pass, then in the existing backwards loop swap the block's frame to `scopeByIndex[i]` before normalizing expr `i`. (A naive push at the `core.Let` construction point is wrong because the body of each threaded let was already normalized *before* the earlier lets are visited.)

4. **Module base frame** (`internal/elaborate/file.go`): after `checkDuplicateModuleBindings` and symbol collection, compute `base = (keys(symbols) ∖ keys(imports)) ∪ {moduleLet names}` and push it for the duration of the SCC elaboration. This is what makes row 7 resolve to the local and keeps recursive self-calls intact (they are in `symbols`, not in `imports`).

5. **Qualified access** (`expr_data.go:39-52`): in `normalizeRecordAccess`, if the base identifier `L` is `inScope`, fall through to ordinary record access on the local (row 10); otherwise the qualified-import path, unchanged.

6. **Constructor calls** (`expr_calls.go:27`): if the callee identifier is `inScope`, skip the factory branch and fall through to `normalize(app.Func)` (the local binder wins at call position).

7. **MOD015** — *shipped as an error, not this warning (ruling 2026-10-02; see Implementation notes).* Original proposal (`internal/errors/codes.go` + file.go collision check): when a directly-imported bind name equals a module-level local binding name, emit `Warning MOD015 (import-shadow): import '<name>' is shadowed by a local binding at <pos>; references resolve to the local`. Emitted once per collision, warning-only (import stays reachable via qualified alias `import M (x as y)`).

### Implementation Plan

**Phase 0: Corpus + fixture baseline (~0.5 day)**
- [ ] Commit the ten-row matrix as table-driven pipeline fixtures (`internal/pipeline/lexical_scope_shadowing_test.go`), recording today's outcomes (7 red, 3 silent-wrong).
- [ ] Sweep `std/`, `examples/`, package testdata for binders colliding with imported/builtin/constructor names (`grep` binder names × `AllSpecs()` list + per-module imports); list every hit — these are the behavior-change candidates the fix must justify.
- [ ] Record the #327 fixture matrix status as the no-regression baseline.

**Phase 1: Scope stack + expression resolution (~1 day)**
- [ ] `PushScope/PopScope/inScope` on Elaborator; new step 2 in the `*ast.Identifier` order; constructor-call and qualified-access guards (components 2, 5, 6).
- [ ] Lambda/FuncLit/Let/LetRec/Match/Forall pushes (components 3) with unit tests per binder kind.

**Phase 2: Module base frame + MOD015 (~1 day)**
- [ ] Base-frame seeding in `ElaborateFile` (component 4); MOD015 warning + error-code registration and test.

**Phase 3: Block threading + REPL audit (~0.5 day)**
- [ ] Per-index scope sets in `normalizeBlock` (component 3, last bullet) with a dedicated fixture (`{ let x = e1; f(x); let y = g(x); h(y) }` shapes, including a use *textually before* its binding statement, which must still resolve to the outer/builtin binding).
- [ ] Audit `internal/repl/` (`Elaborate` path has no module frame — unchanged by design; add a test pinning REPL cross-statement resolution as the documented non-goal).

**Phase 4: Matrix green + corpus re-sweep + docs (~1–1.5 days)**
- [ ] All ten fixtures produce the "Should be" outcome; `make test`, `make test-core`, `make check-boundaries` green; corpus re-sweep diff vs Phase 0 (must be exactly the matrix rows).
- [ ] `ailang prompt` idiom note (shadowing now lexical; MOD015), `docs/LIMITATIONS.md` update if shadowing is listed there, CHANGELOG entry; close the v0.50.0 report with the root-cause note.

### Files to Modify/Create

**New files:**
- `internal/pipeline/lexical_scope_shadowing_test.go` — the ten-row matrix + block/REPL fixtures (~250 LOC)

**Modified files:**
- `internal/elaborate/core.go` — scope stack fields + methods (~40 LOC)
- `internal/elaborate/expressions.go` — resolution order (~15 LOC)
- `internal/elaborate/expr_simple.go` — lambda/func-lit pushes (~15 LOC)
- `internal/elaborate/expr_control.go` — let/letrec/block/forall pushes (~60 LOC incl. per-index sets)
- `internal/elaborate/patterns.go` — `patternBinders` + match push (~50 LOC)
- `internal/elaborate/expr_calls.go` — inScope guard on constructor branch (~5 LOC)
- `internal/elaborate/expr_data.go` — inScope guard on qualified access (~5 LOC)
- `internal/elaborate/file.go` — base-frame seeding + MOD015 collision warning (~40 LOC)
- `internal/errors/codes.go` — MOD015 (~5 LOC)

## Examples

### Example 1: The reporter's repro (row 1)

**Before (v0.50.0):**
```
$ ailang check d.ail
Error: type error in d (decl 1): type unification failed at [function application at d.ail:5:36]:
failed to unify parameter 0: failed to unify record field 'tick': cannot unify type constructor int with *types.TFunc2
```

**After:**
```
$ ailang run d.ail
4
```

### Example 2: The silent miscompile (rows 7–8)

**Before:**
```ailang
module rx/p/q
import ./a (tick)                       -- imported tick(n) = n + 1
pure func tick(n: int) -> int = n + 100 -- local tick
export pure func main() -> int = tick(1)
```
```
$ ailang run q.ail
2                                        -- the IMPORT ran; no warning, no error
```

**After:**
```
Warning MOD015 (import-shadow): import 'tick' is shadowed by a local binding at q.ail:3:12;
references resolve to the local
$ ailang run q.ail
102
```

Row 8 (`mk(\tick. tick)(5)`) flips from silently returning `6` (the imported function applied to 5) to a *type error* naming the real conflict: the lambda returns its `int` parameter, not an `int -> int`.

## Success Criteria

- [ ] Reporter's `d.ail` checks green and runs to `4` (fixture row 1)
- [ ] All ten matrix fixtures produce the "Should be" outcome (table-driven test)
- [ ] `make test` + `make test-core` + `make check-boundaries` green; #327 fixture matrix green
- [ ] Phase 0 corpus sweep diff = exactly the matrix rows (no other program changes meaning)
- [ ] MOD015 fires on row 7's shape and is silent elsewhere; error-code registry test
- [ ] Block threading fixture (use-before-let-statement resolves to outer binding) green
- [ ] CHANGELOG entry + `ailang prompt` idiom note + v0.50.0 report closed with root-cause note
- [ ] All tests passing; documentation updated; examples added (matrix fixtures double as examples)

## Testing Strategy

**Unit tests:**
- `patternBinders` classification parity with `elaboratePattern` (same patterns, same classification).
- Scope stack push/pop symmetry around every binder kind (error paths included — pop on normalize error must not leak scope into the next declaration).

**Integration tests:**
- The ten-row matrix through the full module pipeline (`ailang check` + `ailang run` semantics, i.e. assert values not just exit codes).
- Import-alias variants: `import ./a (tick as tock)` — no collision, no MOD015; aliased constructor (`Foo as f`) shadowed by a local `f`.
- Cross-module: importing module whose *exports* were compiled under the fix (interface/type-scheme stability — `globalTypes` keys unchanged).

**Manual testing:**
- `ailang repl`: `let show = 3` then `show` (documented non-goal behavior pinned), `\tick. tick + 1` applied at 7 within one statement.
- `ailang check` over the full `examples/` tree before/after (diff must be empty).

## Deferred Decisions

- Exact scope-frame representation (map per frame vs. persistent chain) — agent may choose; both are within noise for module-sized inputs.
- Whether MOD015 also covers *builtin*/constructor collisions (vs. imports only) — agent may extend after the corpus sweep quantifies it.
- Whether to run `ailang design-quorum` on this doc before sprint-planning — optional documented step; left to the human/sprint-planner (controller in-session verdict: premises are live-verified, proceed).
- Retiring the #327 interim diagnostic (`SetModuleFuncNames`) — decided by whether this fix's matrix subsumes #327's red cell; sprint-evaluator verifies, not this sprint.

## Non-Goals

- **Pattern-position resolution** (`match` pattern classification) — constructor-first stays; that is #323's settled territory.
- **The effect-checker sibling** (`declaredEffects` name-keyed lookup, `validate_effects.go`) — separate backlog row, separate fix; this doc changes only elaboration.
- **REPL cross-statement rebinding semantics** — registry/`globalEnv` persistence is not lexical scope; out of scope by Design Freeze decision.
- **Hygienic renaming of binders** (alpha-renaming to avoid capture at the core level) — not needed; `core.Var` + `TypeEnv` already resolve correctly.

## Timeline

**Week 1** (2 days): Phase 0 + Phase 1
**Week 2** (2 days): Phase 2 + Phase 3
**Week 3** (1–1.5 days): Phase 4 + release chores

**Total: ~5 days across 3 weeks** (2× the honest 2.5-day estimate, per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Corpus program relied on capture (import/builtin winning over a local) | High | Phase 0 sweep enumerates every collision *before* the fix; each hit is a reviewed behavior change, not a surprise |
| Block backwards-threading scope sets off-by-one | Med | Dedicated use-before-binding fixture; per-index sets derived in a forward pass mirroring the existing `scopeByIndex` spec |
| `patternBinders` diverges from `elaboratePattern` classification (constructor vs binder) | Med | Parity unit test over a pattern corpus; both functions side-by-side in `patterns.go` |
| Base-frame set misclassified (import counted as local) breaks recursive self-calls | High | `symbols ∖ imports ∪ moduleLets` derived from the same maps file.go already uses; e.ail and recursion fixtures pin it |
| Scope leak across declarations after an elaboration error (missing pop) | Med | Push/pop symmetry unit test; `defer`-style pop in each normalize site |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_29_0/m-record-update-local-resolution.md` — #327, the same family from the type-checker side; its "Interim" diagnostic is the marker this fix family intends to retire; its fixture matrix is a no-regression gate here.
- `design_docs/archive/v0_4_9_m-bug-module-let-scope.md` — established that module-level names must be visible in function bodies (the same visibility this doc's base frame guarantees).
- `design_docs/implemented/v0_10_0/m-pkg-interref-fix.md` — package inter-reference plumbing that the reporter's `rx/p/…` shape exercises.

**Planned (check for overlap):**
- `design_docs/planned/ailang-core-triage/effect-checker-let-shadowing.md` — sibling member (effect checker); explicitly sequenced separately, cross-referenced here.
- `design_docs/planned/ailang-core-backlog.md` (2026-09-15 row) — same.

**Distinctness note:** no existing doc covers elaborator-side identifier capture; the skill's SimHash/neural search on "elaborator lexical scope" returned no matches, and manual grep confirms the family docs above address different phases (type checker, effect checker) or different positions (record-update fields).

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [docs/LIMITATIONS.md](/docs/LIMITATIONS.md) — check on release whether shadowing is still listed as a limitation
- `internal/elaborate/expressions.go` — the resolution order under change
- `internal/types/alias_capture.go` + `shadowLocalTypeNames` (`internal/pipeline/pipeline_module_phases.go:329`) — the type-level precedent for "local wins"
- v0.50.0 release report (this task's source): `rx/p/a` + `rx/p/d` repro

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| 1 | Reporter's repro fails with the reported error | live `ailang check` on two-file package `rx/p/{a,d}` (ailang.toml, edition 1), binary v0.50.0-6-g021c46907-dirty | `type unification failed at [function application at d.ail:5:36]: … record field 'tick': cannot unify type constructor int with *types.TFunc2` — reproduced |
| 2 | Elaborator resolves Identifier via constructor → globalEnv → Var, no scope step | read `internal/elaborate/expressions.go:39-71` | Confirmed: `e.globalEnv[ex.Name]` checked at line 61 before the `core.Var` fallback |
| 3 | Elaborator has NO lexical-scope structure (negative-existence) | `grep -rn "scopeStack\|lexicalScope\|pushScope\|localScope" internal/elaborate/*.go` | Empty (only this doc's language) — confirmed absent; struct fields listed in `core.go:15-57` contain no scope |
| 4 | `globalEnv` holds ALL builtins + imports | read `internal/elaborate/core.go:138-149` (`AddBuiltinsToGlobalEnv` iterates `builtins.AllSpecs()`); `internal/pipeline/pipeline_module_phases.go:331-333` (`SetGlobalEnv(imports.GlobalRefs)` then `AddBuiltinsToGlobalEnv()`) | Confirmed — row 6 needs no import statement |
| 5 | Builtin name `show` unshadowable by let / func param (rows 6) | live `ailang check` on `let show = 3 in show` and `func main(show: int) -> int = show` | Both fail: `cannot unify function type with int` — confirmed |
| 6 | Same-module function shadowing WORKS today (downstream already lexical) | live `ailang run` on e.ail (local `tick(n)=n+1`, same repro shape) | Prints `4` — confirmed; fix confined to elaborator |
| 7 | Module-level local func vs same-named import silently miscompiles (row 7) | live `ailang run` on q.ail (local `n+100`, import `n+1`, `tick(1)`) | Prints `2` (import), exit 0, no warning — confirmed silent miscompile |
| 8 | file.go documents "Local functions take precedence" (intent contradicted by row 7) | read `internal/elaborate/file.go:238-243` | Confirmed: comment says local wins and import "still accessible via globalEnv" — the elaborator's order inverts it |
| 9 | No import-vs-local collision gate exists (negative-existence) | read `checkDuplicateModuleBindings`, `internal/elaborate/file.go:16-41` | Covers let-vs-func and let-vs-let only (MOD007) — confirmed no import-vs-local check |
| 10 | Lambda param coinciding with nullary constructor resolves to the constructor, checker green (row 9) | live `ailang check` on `mk(\None. None)` with `mk : int -> Option[int]` | `✓ No errors found!` — confirmed silent capture |
| 11 | Silent wrong answer via lambda + import (row 8) | live `ailang run` on `mk(\tick. tick)(5)` with `mk : int -> (int -> int)` | Prints `6` — confirmed; under lexical scoping the checker must reject |
| 12 | Match arm bodies normalize without pattern binders in scope (row 5) | read `internal/elaborate/patterns.go:23-46` | `e.normalize(caseClause.Body)` with no binder registration — confirmed; live repro `match x { tick => tick }` fails: `cannot unify function type with int` |
| 13 | Let/LetRec/Forall binder sites have no scope push; function params desugar through `normalizeLambda` | read `expr_control.go:83-108, 203-216, 332-355`, `file_funcs.go:262-277` | Confirmed: no pushes anywhere; one push site covers function-decl params |
| 14 | Block normalization threads BACKWARDS (last expr first) | read `internal/elaborate/expr_control.go:256-330` | Confirmed: iterates `len-1 … 0` wrapping lets around an already-normalized continuation — per-index scope sets required |
| 15 | Imported constructors are registered into the elaborator's constructor map | read `pipeline_module_phases.go:335-337` (`RegisterConstructor` over `imports.ImportedCtorInfos`) | Confirmed — constructor faces (row 9) fire with plain imports, no `import (Ctor)` needed |
| 16 | Type-level "local wins" precedent exists | read `pipeline_module_phases.go:329-330` (`shadowLocalTypeNames`, M-TYPE-NAME-SHADOW) + `internal/types/alias_capture.go:5-18` | Confirmed — value-level fix aligns with shipped type-level rule |
| 17 | MOD015 error code is unallocated (negative-existence) | `grep -rn "MOD015" internal/ cmd/ std/` | Empty — free (codes.go currently ends at MOD013; MOD014 is allocated in `pipeline_module.go` per the m-module-less-run-fail-loud case study and its footgun tests) |
| 18 | Elaborator consumers list (blast radius) | `grep -rln "NewElaborator" internal/ cmd/` minus tests | `internal/pipeline/{pipeline_module_phases,pipeline_single}.go`, `internal/repl/{module_registry_load,repl_commands,repl_eval}.go`, `internal/runtime/runtime.go`, `cmd/ailang/debug.go` — all go through the shared `normalize`, all inherit the fix |
| 19 | Regression fixtures exist | `ls examples/` | `pattern_matching_adt.ail`, `match_hof_lambda.ail`, `option_pattern_import_free.ail`, `prelude_option_result.ail` all present — cited only after existence check |
| 20 | #327 interim diagnostic exists and names "the real resolution fix" | read `internal/types/typechecker_core.go:313-318` (`SetModuleFuncNames` comment) | Confirmed — retirement decision is sprint-evaluator territory, cross-referenced |
| 21 | Effect-checker sibling is a distinct phase (no overlap) | read `design_docs/planned/ailang-core-triage/effect-checker-let-shadowing.md` | Confirmed: `internal/pipeline/validate_effects.go` `declaredEffects`-first order — untouched by this design |
| 22 | Typo'd/alternative repro shapes genuinely fail (not reporter error) | live matrix rows 2–5, 10 on the same binary | All fail as tabulated in Problem Statement — matrix rows are live transcripts, not predictions |

## Future Work

- Unify with the effect-checker resolver (backlog 2026-09-15): one lexical-resolution module shared by elaborator, type checker, and effect checker — the end-state #327's "real resolution fix" gestures at.
- REPL cross-statement rebinding semantics (registry `globalEnv` vs lexical persistence).
- Wildcard imports (`import M` exposing all exports by name) will multiply this bug's surface — the scope stack must be the default answer there too when/if `collectImports`' TODO is implemented.

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01
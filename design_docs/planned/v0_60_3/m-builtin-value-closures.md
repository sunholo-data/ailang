# M-BUILTIN-VALUE-CLOSURES — a stdlib builtin named as a value is evaluator-only: compile it to a closure over a synthesized builtin thunk

**Status**: Planned
**Target**: v0.60.3
**Priority**: P1 (one accepted construct fails the strict-VM gate with a misleading diagnostic; bites AI-generated code that passes builtins to HOFs, the idiom `--strict-bytecode` exists to cover)
**Estimated**: 2 days (~120 LOC compiler + ~250 LOC tests; no VM, no parser, no typechecker, no language-semantics change)
**Dependencies**: None (touches `internal/bytecode/compiler/` only; no in-flight siblings)

## Problem Statement

A builtin referenced in **value position** (passed to a higher-order function, let-bound,
returned) lowers to a bare `stmt.GlobalRef{Module: "$builtin", Name: <registry name>}`, and the
bytecode compiler's only handling for a first-class global is a lookup in the user-function
prototype table. The lookup misses, the enclosing function is tagged evaluator-only, and
`--strict-bytecode` rejects the whole entry function as BC001 with the misleading reason
`compiler: unknown global "$builtin.intToFloat"` — the global is not unknown, it is a builtin
the VM dispatches natively in call position.

Field repro (reported from stapledons-godot, R3-UNIVERSE-LAB UL3c — one test helper in
`sim/lab_stats_test.ail`; the module under test is unaffected. Reproduced on the game's pin
v0.56.1 and on dev `v0.60.0-46-g6d28e1ad5a`, and re-verified on v0.60.2, see Verification Log):

```ailang
module fv
import std/list (map, range)
import std/math (intToFloat)
export pure func asValue(n: int) -> [float] = map(intToFloat, range(0, n))
export pure func wrapped(n: int) -> [float] = map(\i. intToFloat(i), range(0, n))
```

```
$ ailang run --quiet --bytecode --strict-bytecode --entry asValue --args-json 3 fv.ail
Error: bytecode execution failed: 1 evaluator-only function(s) reachable from asValue; --strict-bytecode wires no evaluator bridge:
  BC001: function fv.asValue is evaluator-only under --strict-bytecode (compiler: unknown global "$builtin.intToFloat") at fv.ail:4
$ ailang run --quiet --bytecode --strict-bytecode --entry wrapped --args-json 3 fv.ail
[0.0, 1.0, 2.0]
```

**Current state:**
- The eta-expanded form runs on the strict VM; the bare form is evaluator-only. The two forms
  denote the same function and both type-check (`ailang check` passes both).
- The gap covers the entire builtin registry in value position, not just `intToFloat`:
  `map(show, xs)`, `let f = intToFloat in f(n)` fail identically (Verification Log V4, V5).
- The user-visible name in the repro is `intToFloat`, imported from `std/math` — but the
  elaborator resolves the bare name to `$builtin.intToFloat` because
  `AddBuiltinsToGlobalEnv` intentionally wins over imports and module declarations
  (scope_test.go `builtin_kept`; see "What does NOT change" below).
- The qualified form `map(M.intToFloat, ...)` (import `std/math as M`) **already works** under
  `--strict-bytecode`, because it resolves to the stdlib wrapper's compiled prototype
  (Verification Log V6). The failure is specific to the `$builtin` module ref.
- `ailang check --strict-bytecode` already reports this exactly as BC001 (Verification Log V3);
  plain `ailang check` passes, consistent with its contract (types/effects, not VM coverage).

**Impact:**
- AI-generated code — the audience `--strict-bytecode` exists for — writes the bare form
  naturally (`map(intToFloat, xs)`); the workaround (`\i. intToFloat(i)`) is an anti-idiom a
  model only discovers from the run error.
- The BC001 reason string is wrong ("unknown global") for what is actually an unported
  *value-position* lowering, which costs debugging time in exactly the field situation that
  produced this report.

## Goals

**Primary Goal:** A dispatchable builtin referenced in value position compiles to a closure the
strict VM executes natively — behaviorally identical to the eta-expanded lambda — so
`map(intToFloat, xs)` and `map(\i. intToFloat(i), xs)` agree under evaluator, `--bytecode`, and
`--strict-bytecode`.

**Success Metrics:**
- The repro pair above produces `[0.0, 1.0, 2.0]` under `--strict-bytecode` for **both** entry
  points, and `ailang check --strict-bytecode fv.ail` exits 0.
- A builtin that has no VM implementation in call position (unported/evaluator-only) keeps
  failing in value position too, with the *same* reason text family the call position emits
  (no "unknown global" misnomer) — parity is preserved in both directions.
- Images of programs that do not reference builtins as values compile byte-identically to today
  (lazy synthesis; `tools/vm_parity_diff.sh` baseline rows unchanged).
- Engine parity means the **evaluator and the strict VM agree**; generated Go is not in scope.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Compile-time thunk synthesis (a synthetic prototype whose body is the builtin dispatch instruction) vs. a new VM value representation for builtin closures | A VM value-tag change touches `bytecode.Value`, the bridge converters, `overApply`, ShowValue — a cross-package ripple; thunk synthesis changes one compiler package and reuses the existing closure machinery verbatim | human | design | high if reversed later (value format is persisted/diffed) |
| Lazy synthesis, memoized per Compile, purged on prototype rollback — not an eager Phase-1.5 sweep of all ~150 builtins | Eager sweep changes every image's shape and shifts prototype indices for all programs; lazy keeps unchanged programs byte-identical (parity baselines, determinism) | human | design | med |
| Table-lookup precedence and error text for the value-position path mirror `compileBuiltinCall` exactly (builtinIndex → HOF → effect; unported messages) | Two divergent precedence orders would make the two syntactic forms disagree again — the very bug being fixed | compiler | design | low |
| Thunk prototype registered under the canonical key `$builtin.<name>` in image prototype space, but NOT via the bare-name fallback | The existing bare-name fallback in the GlobalRef case could bind a `$builtin` ref to an unrelated user prototype of the same short name once thunks exist; the `$builtin` path must resolve only to the thunk | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Thunk synthesis in `internal/bytecode/compiler` (no VM change, no new value tag)
- [x] Lazy memoized synthesis with rollback purge (not eager table sweep)
- [x] Dispatch precedence mirrors `compileBuiltinCall`; remaining failures reuse its message
      family ("no VM implementation (%adapt reason)" / "effectful builtin not yet wired")
- [x] No `ailang check` (plain) behavior change; `check --strict-bytecode` stays the gate

## Deferred Decisions

The following are intentionally left open for the implementer:

- Thunk body register layout: reverse-shift params into `R[1..N]` with dst `R[0]` (NumRegs
  N+1, tightest) vs. copy params to `R[N+1..2N]` with dst `R[N]` (NumRegs 2N+1, simplest) —
  agent may choose; both satisfy the contiguous `R[A+1..A+C]` arg layout all three dispatch
  opcodes read (vm.go OpBuiltinCall/OpBuiltinCallHOF/OpEffectCall).
- Memo data structure and plumbing (field on a Compile-scoped struct passed to
  `newFuncCompiler` vs. package-level none — must be per-Compile for determinism and
  test isolation) — agent may choose.
- Whether the thunk purge on rollback lives beside `dropFuncIdxFrom` (compiler.go error path)
  or as a method on the memo — agent may choose.
- Unit-test file organization inside `internal/bytecode/compiler/` — agent may choose.

## Solution Design

### Overview

`compileExpr`'s `stmt.GlobalRef` case today only knows user-function prototypes. We add a
`$builtin` branch: when the ref's module is `$builtin`, resolve its IR name (`"_" + registry
name`, the same convention `lowerApp` uses for call position) through the same three tables
`compileBuiltinCall` uses, and materialize the reference as an `OpClosure` over a **synthesized
builtin thunk prototype** — a tiny function whose parameters are the builtin's arguments and
whose body is exactly the dispatch instruction the eta-expanded lambda would have emitted:

```
$builtin.intToFloat thunk (N = 1):
  ; params at R[0..N-1] on entry
  MOVE   R1, R0            ; (reverse-shift layout; N+1 regs total)
  BUILTIN_CALL A=0, B=<idx of _intToFloat>, C=1
  RETURN R0
```

This is the user's working workaround compiled by the machine instead of by the author: the
closure value the VM sees, the over-application path (`overApply`), captures, and stack traces
are all the existing closure machinery — no VM changes. A builtin with no VM implementation
keeps failing at compile time with `compileBuiltinCall`'s message, so the enclosing function
stays evaluator-only and BC001 keeps reporting it — parity between the two syntactic forms is
the invariant, in both directions.

### Architecture

**Components:**
1. **Thunk synthesis** (`internal/bytecode/compiler/builtin_thunks.go`, new): given a registry
   name, return the image prototype index of its thunk, synthesizing on first use. Arity comes
   from `builtins.GetSpec(registryName).NumArgs`; the dispatch opcode from the existing
   `builtinIndex` / `hofBuiltinIndex` / `effectBuiltinIndex` maps, checked in that order.
   Non-callable registry names (`IsCallableBuiltin` false — symbols like `::`) and names with
   no table entry and no spec fall through to the error path unchanged.
2. **GlobalRef branch** (`internal/bytecode/compiler/expr.go`, `compileExpr` `stmt.GlobalRef`
   case): try thunk synthesis for `Module == "$builtin"` before the funcIdx lookup; on success
   emit `OpClosure` over the thunk via the existing `lookupNestedProto` path (this also records
   the reachability edge for `EvalOnlyReachable`). The `$builtin` path never consults the
   bare-name fallback.
3. **Rollback purge** (`internal/bytecode/compiler/compiler.go`, Phase-2 error path): when a
   FuncDecl compile fails and `img.Prototypes` is truncated to `protoCheckpoint`, drop memo
   entries whose image index ≥ `protoCheckpoint`, mirroring `dropFuncIdxFrom`. The next
   successful compile re-synthesizes deterministically. (A thunk synthesized during a *failed*
   FuncDecl must not leave a memo entry pointing past the truncated table — see Risks.)

### Implementation Plan

**Phase 1: Thunk core** (~4 hours)
- [ ] `builtin_thunks.go`: `builtinThunkProto(registryName) (*bytecode.FuncPrototype, error)`
      with the three-table precedence and `compileBuiltinCall`'s error texts; unit tests for
      each dispatch class (pure native `_intToFloat`, adapted, HOF `__list_map`, effect
      `__io_print`) and for each failure class (unported pure with adapt reason, unwired
      effectful, non-callable symbol `::`)
- [ ] Arity cross-check test: for every entry of `BuiltinNames`, `AdaptedBuiltinNames`,
      `HOFBuiltinNames`, `EffectBuiltinNames`, `GetSpec(registryName).NumArgs` exists and
      matches the spec's type arity; on mismatch the thunk path must fail closed (keep the
      compile error), never miscompile
- [ ] `compileExpr` GlobalRef `$builtin` branch + `OpClosure` emission

**Phase 2: Memoization and rollback** (~3 hours)
- [ ] Per-Compile memo (key `$builtin.<registry>` → image proto idx) plumbed through
      `newFuncCompiler`
- [ ] Purge on FuncDecl-compile rollback beside `dropFuncIdxFrom`; unit test: a function whose
      body fails *after* referencing a builtin as a value, followed by a successful function
      referencing the same builtin — the second must bind a fresh thunk, and
      `img.Validate()` must pass
- [ ] Determinism test (pattern of `determinism_test.go`): same program compiled N times →
      byte-identical images, including one with two functions referencing two different
      builtins as values

**Phase 3: Integration, parity, docs** (~4 hours)
- [ ] Runner-level strict tests (internal/runner): the four live fixtures from the Verification
      Log (V1 asValue, V2 wrapped, V4 show-as-value, V5 let-alias) run under `--strict-bytecode`
      and agree with the evaluator
- [ ] `ailang check --strict-bytecode fv.ail` exits 0; BC001 message unchanged in shape
- [ ] `tools/vm_parity_diff.sh` baseline re-bank: existing MATCH rows unchanged, no new rows
- [ ] CHANGELOG entry (changelogs/v0.32-current.md)

### Files to Modify/Create

**New files:**
- `internal/bytecode/compiler/builtin_thunks.go` (~100 LOC) — thunk synthesis, memo type,
  three-table dispatch, error texts
- `internal/bytecode/compiler/builtin_thunks_test.go` (~150 LOC) — per-class synthesis,
  arity cross-check, rollback, determinism

**Modified files:**
- `internal/bytecode/compiler/expr.go` (+15/-5 LOC) — `$builtin` branch in the GlobalRef case
- `internal/bytecode/compiler/compiler.go` (+10/-2 LOC) — memo creation, plumbing into
  `newFuncCompiler`, rollback purge
- `changelogs/v0.32-current.md` (+entry) — user-visible fix note

## Examples

### Example 1: The field repro (stapledons-godot shape)

**Before:**
```
$ ailang run --quiet --bytecode --strict-bytecode --entry asValue --args-json 3 fv.ail
Error: ... BC001: function fv.asValue is evaluator-only ... unknown global "$builtin.intToFloat"
```

**After:**
```
$ ailang run --quiet --bytecode --strict-bytecode --entry asValue --args-json 3 fv.ail
[0.0, 1.0, 2.0]
$ ailang check --strict-bytecode fv.ail && echo OK
OK
```

### Example 2: An unported builtin keeps parity (both forms fail identically)

A pure builtin whose signature cannot cross the adapter (e.g. a polymorphic builtin with
`ReasonPolymorphic`), referenced either way:

**Before:** call position fails with the adapt reason; value position fails with the misnomer
"unknown global" — two different stories for the same gap.

**After:** both positions fail with `pure builtin "..." has no VM implementation (<reason>)`,
the enclosing function is evaluator-only in both, BC001 reports it in both, and a non-strict
`--bytecode` run bridges both.

### Example 3: Disassembly shape

`map(intToFloat, range(0, n))` after the fix (thunk inlined into the disassembly view):

```
$builtin.intToFloat            ; 1 param, 2 regs
  MOVE      R1, R0
  BUILTIN_CALL A=0 B=idx(_intToFloat) C=1
  RETURN    R0
fv.asValue
  ...BUILTIN_CALLHOF __list_map...
```

## Success Criteria

- [ ] V1 repro (`map(intToFloat, range(0, n))`) returns `[0.0, 1.0, 2.0]` under
      `--strict-bytecode`, identical to V2's eta-expanded form (acceptance: runner-level test)
- [ ] `map(show, ...)` (V4) and `let f = intToFloat in f(n)` (V5) run under `--strict-bytecode`
- [ ] Unported builtins fail in value position with the call-position message family —
      never "unknown global" (acceptance: unit test per failure class)
- [ ] Arity cross-check test over all four dispatch tables passes or fails closed per builtin
- [ ] Rollback + determinism unit tests pass (Phase 2)
- [ ] `make test-changed` and `make test-core` green; `make check-boundaries` green
- [ ] `tools/vm_parity_diff.sh` diff vs re-banked baseline: no changed outcomes for
      `examples/runnable` (acceptance: MATCH rows unchanged)
- [ ] All tests passing; CHANGELOG updated; no doc changes required beyond CHANGELOG
      (no language-surface change)

## Conflict Surface

### Syntactic positions touched

The change extends exactly one code path: `compileExpr`'s `stmt.GlobalRef` case
(internal/bytecode/compiler/expr.go:66–79), reached only from value positions inside compiled
function bodies (let values, lambda bodies, call arguments, return expressions) where the
lowered expression is a bare global reference with `Module == "$builtin"`. No parser, lexer,
typechecker, elaborator, or lowering change is made — the set of programs that produce a
`$builtin` GlobalRef is exactly the set that produces one today.

### What else lives here

| Position (stmt.GlobalRef in value position) | Existing form | Behavior after change |
|---|---|---|
| `Module == "$builtin"`, dispatchable (≈115 native + adapted + 14 HOF + 20 effect entries) | Compile error → enclosing function evaluator-only → BC001 under strict | **New:** closure over synthesized thunk; strict-VM native |
| `Module == "$builtin"`, unported/unwired (incl. non-callable symbols like `::`) | Compile error "unknown global" → evaluator-only | Compile error with `compileBuiltinCall`'s reason text → evaluator-only (same gate, honest message) |
| `Module` = a user/std module (e.g. `std/fs.ail:207` `sortBy(compare, ...)` passes `std/string.compare` as a value; qualified `M.intToFloat`) | Compiles to closure over that module's prototype | Unchanged |
| `Module == "$adt"` | Lowered to `stmt.ADTConstructor` before the compiler (lower/expr.go:29–45); never reaches this case as a GlobalRef except factory names that fail to parse, which keep today's error | Unchanged |
| Bare-name fallback (`funcIdx[e.Name]`) for refs whose module prefix isn't registered | Legacy single-module compatibility path | Unchanged for non-`$builtin` refs; the `$builtin` branch resolves before it and never falls back (guard against binding a user prototype of the same short name) |

### Disambiguation strategy

Module equality: the branch triggers only on `Module == "$builtin"`, a pseudo-module no user
declaration can produce (imports resolve to real module IDs; constructors to `$adt`; the
elaborator is the only producer of `$builtin` refs, via `AddBuiltinsToGlobalEnv` and the
hygienic `ResolveAsBuiltin` path). Dispatch-class disambiguation reuses `compileBuiltinCall`'s
fixed precedence (`builtinIndex` → `hofBuiltinIndex` → `effectBuiltinIndex`) so the two
syntactic forms can never pick different opcodes.

### Programs that MUST still work

Regression fixtures (all exist; all must stay strict-VM MATCH in the parity harness):

1. `examples/runnable/list_helpers.ail` — `any(isEven, nums)`: user function as value (GlobalRef
   to a user prototype)
2. `examples/match_hof_lambda.ail` — lambdas passed to `foldl` (closure/overApply paths)
3. `examples/poly_arith_lambda.ail` — `show(foldl(\acc. \x. ...))` (show in call position,
   lambda as value)
4. `std/fs.ail:207` — `sortBy(compare, walkDir(root))` (stdlib global as value, compiled into
   every stdlib image)
5. `std/json.ail:127` — `intToFloat(i)` in call position (the `BuiltinCall` path this design
   must not perturb)

### Pipeline Pass Coverage

This design adds no lowering pass and no new expression source — it changes only how the
bytecode compiler materializes one already-lowered IR node. Contract expressions cannot reach
the branch: `grep -rn 'Contract' internal/gen/lower/ internal/gen/stmt/` is empty (V12) —
contracts are verified via SMT/`--verify-contracts`, never compiled into the image.

### What deliberately changes

- Functions that reference a dispatchable builtin as a value stop being evaluator-only. A
  strict run that used to reject them (BC001) now executes them; a non-strict `--bytecode` run
  that used to bridge them now runs them natively. Output values are unchanged (V7 shows the
  bridged evaluator path and the plain evaluator agree today).
- The BC001 reason text for a *value-position reference to an unported builtin* changes from
  `unknown global "$builtin.<name>"` to the call-position family. No other reason text changes.
- Deliberately NOT changed: the elaborator's builtin-over-import/module precedence
  (`intToFloat` resolves to `$builtin.intToFloat` even though `std/math` exports a wrapper) —
  that is documented, tested behavior (`scope_test.go` `builtin_kept`; the separately-filed
  shadowing question is tracked as inbox_1788928007403), and the qualified form already gives
  authors who want the wrapper a working spelling (V6).

## Testing Strategy

**Unit tests** (`internal/bytecode/compiler/builtin_thunks_test.go`):
- Thunk synthesis per dispatch class: pure native, adapted, HOF (closure param arrives as
  TagClosure and the HOF table's `ClosureCaller` receives the VM), effect
- Failure classes: unported pure (adapt reason in message), unwired effectful, non-callable
  symbol; each keeps the enclosing proto evaluator-only
- Arity cross-check over all four tables (fail closed on mismatch)
- Rollback: failed-then-successful compile sequence binds a fresh thunk; `img.Validate()` clean
- Determinism: repeated compiles byte-identical (determinism_test.go pattern)

**VM tests** (`internal/vm/`):
- Call a thunk closure directly; exact-arity call, over-application (`overApply` partial
  closure), and under-application error parity with a user lambda of the same arity

**Integration tests** (`internal/runner/`):
- V1/V2/V4/V5 fixtures under `--strict-bytecode` agree with evaluator output
- `ailang check --strict-bytecode fv.ail` exits 0 after the fix

**Regression-surface tests:**
- One pinned test per Conflict-Surface fixture 1–5 (strict parity harness outcome MATCH)

**Manual testing:**
- `tools/vm_parity_diff.sh --bank` then diff: zero changed rows
- `make test-changed && make test-core && make check-boundaries`

## Non-Goals

**Not in this feature:**
- Changing the elaborator's builtin-over-import/module-declaration precedence — documented
  existing behavior (`builtin_kept`), tracked separately (inbox_1788928007403)
- Making plain `ailang check` gate strict-VM coverage — `check --strict-bytecode` already is
  that gate (V3); plain check's contract is type/effect correctness
- A VM-native value representation for builtin closures (new tag / builtin-Proto) — rejected
  as the high-ripple alternative (see High-Impact Decisions)
- Wiring unported builtins to the VM — that is M-BYTECODE-2E / M-VM-PURE-BUILTIN-COVERAGE
  territory; this design only removes the *value-position* asymmetry
- Generated-Go parity — emit-go is a demoted diagnostic projection; evaluator + strict VM are
  the parity contract

## Timeline

**Week 1** (11 hours):
- Phase 1: Thunk core + unit tests (4h)
- Phase 2: Memoization, rollback purge, determinism (3h)
- Phase 3: Integration, parity re-bank, CHANGELOG (4h)

**Total: ~11 hours across 1 week** (initial estimate ~5h, doubled per house rule)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Registry `NumArgs` disagrees with the VM table's expected operand count for some builtin → thunk miscompiles silently | High | Arity cross-check test over all four tables; any mismatch fails closed (compile error, evaluator-only) rather than emitting a wrong-arity thunk |
| Rollback truncates `img.Prototypes` while the memo still points past the checkpoint → later `OpClosure` binds a stale index | High | Purge memo entries ≥ `protoCheckpoint` beside `dropFuncIdxFrom` (the existing purge's blind spot is prelude keys only — V11); unit test for the failed-then-successful sequence |
| Two syntactic forms diverge again via a different dispatch precedence | Med | The `$builtin` branch calls the *same* table lookups in the *same* order as `compileBuiltinCall`; a shared helper is preferred so they cannot drift |
| Effectful builtin thunks execute effects the enclosing function did not declare | Med | Impossible by construction: the typechecker/effect checker already required the effect row when the builtin was referenced — the thunk emits the same `OpEffectCall` the call position emits; runner test with `_io_print` as a value inside an `{IO}` function |
| Image bloat from thunk prototypes | Low | Lazy: one prototype (~N+2 instructions) per distinct builtin actually referenced as a value per compile; ~30–60 bytes each |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes an engine-dependent behavioral fork: the same source text now behaves identically on evaluator, bridged and strict VM; thunk synthesis is order-deterministic |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | Effect rows unchanged; an effectful builtin as a value carries exactly the rows its call position carries (typechecker-enforced before we ever see the ref) |
| A4: Explicit Authority | 0 | No capability changes; effect builtin thunks dispatch through the same capability/budget/trace path |
| A5: Bounded Verification | 0 | No contract-surface change |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The natural AI idiom (`map(intToFloat, xs)`) stops failing a gate with a misleading reason; remaining failures report the true cause (adapt reason), which is what a model needs to self-correct |
| A8: Minimal Syntax | +1 | No new syntax; an *existing* accepted construct gains its documented meaning |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | +1 | Builtin values now compose with the existing HOF machinery (map/foldl/sortBy) exactly as user function values do |
| A11: Structured Failure | +1 | BC001's reason becomes accurate for the residual failure class (unported builtins) instead of "unknown global" |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (lazy synthesis follows fixed FuncDecl compile order; determinism test pins it)
- [x] A3 (Effects): No hidden side effects (same OpEffectCall, same declared rows)
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Removes a machine-hostile asymmetry, not a human convenience

## Verification Log

All live checks run against the installed v0.60.2 binary (`/usr/local/bin/ailang`,
commit 33f3009); repo at 14ad75c8, clean tree. The report itself cites v0.56.1 (game pin) and
v0.60.0-46-g6d28e1ad5a; the behavior is identical across all three.

| # | Claim | Check | Result |
|---|---|---|---|
| V1 | Builtin as value fails strict VM with BC001 "unknown global $builtin.intToFloat" | `ailang run --quiet --bytecode --strict-bytecode --entry asValue --args-json 3 fv.ail` (repro file built from the report) | Exit 1, `BC001: function fv.asValue is evaluator-only ... (compiler: unknown global "$builtin.intToFloat") at fv.ail:4` |
| V2 | Eta-expanded form runs strict | same with `--entry wrapped` | `[0.0, 1.0, 2.0]`, exit 0 |
| V3 | Plain `ailang check` passes; `--strict-bytecode` reports | `ailang check fv.ail` → ✓ No errors; `ailang check --strict-bytecode fv.ail` | Check exit 0; strict check exit 1 with the same BC001 line — **the `ailang check` half of the report's expected outcome already exists** |
| V4 | Whole class, not just intToFloat: `map(show, xs)` | `map(show, range(0, n))` strict run | BC001 `unknown global "$builtin.show"` |
| V5 | let-bound alias also fails | `let f = intToFloat in f(n)` strict run | BC001 `unknown global "$builtin.intToFloat"` |
| V6 | Qualified stdlib ref already works | `import std/math as M; map(M.intToFloat, ...)` strict run | `[0.0, 1.0, 2.0]` — resolves to the std/math wrapper prototype, not `$builtin` |
| V7 | Non-strict `--bytecode` bridges the value ref today | `ailang run --bytecode --entry asValue` (no strict) | `[0.0, 1.0, 2.0]` with bridged-evaluator note — confirms output parity the fix must preserve |
| V8 | `compileExpr`'s GlobalRef case has no `$builtin` handling (negative existence) | Read internal/bytecode/compiler/expr.go:66–79 | Confirmed: only `funcIdx` lookup + bare-name fallback; `$builtin` refs always miss |
| V9 | Only `stmt.GlobalRef` can carry a `$builtin` ref into the compiler (negative existence) | Read internal/gen/lower/expr.go:29–45 (`core.VarGlobal` → `stmt.GlobalRef`); `core.Var` → `stmt.VarRef` only; elaborator never emits `core.Var` for builtin names (expressions.go falls through to `Var` only for unresolved names) | Confirmed — one code path to fix |
| V10 | Call-position lowering appends `_` and the VM tables use those IR names | Read internal/gen/lower/expr.go:256–264 (`BuiltinCall{Name: "_" + vg.Ref.Name}`); internal/bytecode/builtin_names.go, effect_names.go | Confirmed; thunk lookup must reuse `"_" + name` and `compileBuiltinCall`'s precedence |
| V11 | The rollback purge covers only prelude keys (negative existence) | Read internal/bytecode/compiler/compiler.go:169–186, 227–234 | Confirmed: `dropFuncIdxFrom` iterates `PreludeNames()` only — a new memo needs its own purge |
| V12 | Contract expressions cannot reach the bytecode compiler (negative existence) | `grep -rn 'Contract' internal/gen/lower/ internal/gen/stmt/` | Empty — contracts never lower into `stmt` IR |
| V13 | Arity is available at compile time | Read internal/builtins/spec.go (`NumArgs` field, `GetSpec`); spot-checked registrations: `intToFloat` NumArgs 1, `_list_map` NumArgs 2, `_io_print` NumArgs 1 | Confirmed for pure, HOF and effect classes |
| V14 | The VM can only call closures over FuncPrototypes (no builtin value exists today) | Read internal/vm/vm.go:263–267 (`OpCall`: TagClosure + `*FuncPrototype` cast required) | Confirmed — a new value kind would be a cross-package change; thunks reuse the existing path |
| V15 | All three dispatch opcodes share the contiguous `R[A+1..A+C]` arg layout | Read vm.go `OpBuiltinCall`/`OpBuiltinCallHOF` (609–621)/`OpEffectCall` (624–631) | Confirmed — one thunk body shape serves all classes |
| V16 | Over-application of a closure is handled generically | Read vm.go OpCall `argCount > NumParams` → `overApply` | Confirmed — thunks get partial application for free |
| V17 | Error code namespace: no new code needed | This fix removes a failure and realigns the message of the residual failure; BC001 already allocated (internal/errors/codes.go:325) | Confirmed — no `grep` for a free code required, none introduced |
| V18 | Conflict-surface fixtures exist | `ls examples/runnable/list_helpers.ail examples/match_hof_lambda.ail examples/poly_arith_lambda.ail std/fs.ail std/json.ail` | All exist; `std/fs.ail:207` `sortBy(compare, ...)` and `std/json.ail:127` `intToFloat(i)` read in source |
| V19 | Dispatchable builtin population | Count table entries: BuiltinNames 115, HOFBuiltinNames 14, EffectBuiltinNames 20, plus AdaptedBuiltinNames (computed at init) | ~150 builtins become value-representable |
| V20 | `ailang check` on AILANG claims about evaluator behavior | V7 transcript | Both engines agree today on the bridged form |

Note on toolchain: the sandbox has no Go toolchain; live checks used the installed v0.60.2
binary and all code-path claims are cited by file:line from reading the source at 14ad75c8.

## Related Documents

**Implemented (may inform design):**
- [m-vm-pure-builtin-coverage.md](../implemented/v0_51_0/m-vm-pure-builtin-coverage.md) — the
  three-way builtin coverage contract (native/adapted/unported) and `PureBuiltinSpec`/
  `AdaptedBuiltinNames`, which this design reuses verbatim
- [m-bytecode-hof-builtins.md](../implemented/v0_11_0/m-bytecode-hof-builtins.md) —
  `OpBuiltinCallHOF` and the `ClosureCaller` convention the HOF thunk path relies on
- [m-bytecode-2d-parity.md](../implemented/v0_11_0/m-bytecode-2d-parity.md) — the
  evaluator-only tagging + bridge mechanism this fix narrows
- [m-elaborator-lexical-scope.md](../implemented/v0_52_0/m-elaborator-lexical-scope.md) — the
  `globalEnv` resolution order that decides which refs are `$builtin` (untouched here)

**Planned (check for overlap):**
- [m-bytecode-strict-256-reg-ceiling.md](../planned/v0_53_1/m-bytecode-strict-256-reg-ceiling.md) —
  the strict-coverage reporting (BC001, `check --strict-bytecode`) this fix feeds into; its
  Phase A is landed, no overlap with the compiler change proposed here

Auto-search (SimHash + neural) found no matches for "builtin value closures"; the related docs
above were located by reading the cited code and `design_docs/` directly.

## References

- Field report: stapledons-godot R3-UNIVERSE-LAB UL3c (`sim/lab_stats_test.ail` helper);
  versions v0.56.1 (pin) and v0.60.0-46-g6d28e1ad5a (dev)
- [Design Axioms](/docs/references/axioms) — scoring reference
- `internal/bytecode/compiler/builtins.go` (`compileBuiltinCall`) — the call-position
  contract this design mirrors
- `internal/bytecode/evalonly_report.go`, `internal/runner/strict_report.go` — BC001 reachability
- `tools/vm_parity_diff.sh`, `scripts/verify_bytecode_parity.go` — the parity gate for
  regression fixtures

## Future Work

- Elaborator name-precedence question (builtins vs. imports/module declarations,
  inbox_1788928007403): if module declarations ever win over builtins, `$builtin` value refs
  become rarer but the thunk path remains correct as-is
- `check --package --strict-bytecode` (open Phase C of M-BYTECODE-STRICT-256-REG-CEILING) would
  surface this class at package-check time for eval projects like stapledons-godot
- M-BYTECODE-2E wiring of the remaining effectful builtins shrinks the residual evaluator-only
  set this design leaves honest messages for

---

**Document created**: 2026-11-12
**Last updated**: 2026-11-12

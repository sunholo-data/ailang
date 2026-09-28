# M-BYTECODE-GETFIELD-SLOT-RESOLUTION — GET_FIELD compiles the wrong slot index when two record types share a field name at different positions (silent wrong reads under `--bytecode`)

**Status**: Planned
**Target**: v0.47.2
**Priority**: **P0** — soundness: silent wrong values, nondeterministic codegen, record-shape
corruption. Direct NO-SILENT-FALLBACKS and axiom A1 (determinism) violation.
**Estimated**: ~2 days (root cause fully localized and mutation-free-verified below — this is fix
+ regression tests, not investigation; 2× rule applied in Timeline).
**Dependencies**: none. Independent of [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md)
(parked A2 harness question) and of [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md)
(#505 list-pattern arity — different lowering path, `match.go` vs `collections.go`/`lower/expr.go`).
**Filed bug**: ailang#1354 (reported via Stapledon draft PR sunholo-data/stapledons-godot#3;
mission-stapledon M1.6a is parked on it — controlplane message 2026-09-28, `inbox_1790578890432_a8028692`).
**Reported on**: v0.45.0 (release) and v0.47.0-5-g68fea858d (dev); reproduced first-party here on
v0.47.0 (f5bf7291) — bug present in current source at HEAD `198d38d1`.

## Problem Statement

Under `--bytecode` (and `--strict-bytecode`), a field access `r.f` is compiled to a positional
`GET_FIELD rDst, rRec, slotIndex`. The compiler resolves `slotIndex` from the record's
**alphabetically sorted field list**. When the receiver's type is a **named record type**
(`type V = {x: float, y: float}`), the lower pass fails to attach the receiver's field set, and
the bytecode compiler falls back to scanning **every registered record type in the image** for
one that contains the field name — in Go map iteration order. If two record types share a field
name at different sorted positions, the emitted slot belongs to the wrong type.

The Stapledon game hit this in production shape: `sunholo/relativity` `Motion {phi,tau,t,x}`
(sorted: `x` at slot 3) and `Vec3 {x,y,z}` (`x` at slot 0) — `Vec3.x` compiled with slot 3:
`GET_FIELD: index 3 exceeds field count 3 (in sim/core.turn at sim/core.ail:21)`. Its strict pure
core stopped passing `--strict-bytecode` the moment `Ship` held both types.

**Minimal repro (`repro2.ail`, from the bug report, verified first-party):**

```ailang
module repro2

type M = { a: float, b: float, x: float }
type V = { x: float, y: float }

pure func mx(m: M) -> float = m.x
pure func vx(v: V) -> float = v.x

export pure func main2(n: int) -> float = mx({ a: 1.0, b: 2.0, x: 3.0 }) + vx({ x: 10.0, y: 20.0 })
```

**Current state (all rows first-party, this worktree, binary v0.47.0 f5bf7291):**

| Repro | Shape | `ailang run` (interp) | `--bytecode` | `--bytecode --strict-bytecode` |
|---|---|---|---|---|
| repro2 | shared `.x`, wrong slot out of bounds | 13.0 | 13.0 — **silent evaluator fallback** (VM errors; warning suppressed by `--quiet`) | `Error: GET_FIELD: index 2 exceeds field count 2 (in repro2.vx)` — 10/10 in the reporter's runs, 8/10 here |
| repro5 (`M={a,x}`, `V={x,y}`) | wrong slot **in bounds** | 13.0 | varies | **23.0 (6/10), 21.0 (2/10), 13.0 (2/10) — silent WRONG VALUES, exit 0, no error, no fallback** |
| repro6 | record update `{v \| x: 99.0}` on `V` | 99.0 | — | **10.0 (6/6) — wrong value AND record-shape corruption** (2-field `V` rebuilt as 3-field `{a,b,x}` with values stolen from the wrong slots) |
| repro7 (`type P = V` alias) | `p.x` through alias | 10.0 | — | `GET_FIELD: index 2 exceeds field count 2` |
| repro4 (control) | anonymous `{x: float, y: float}` param | 13.0 | 13.0 | 13.0 — hint path works, deterministic slot 0, 10/10 |

**Impact:**
- Any `--bytecode` program whose records share a field name at different positions — the
  default pipeline, no exotic flags. Shared names like `x`, `y`, `id`, `name` across record
  types are the norm, not the exception (the game's `Motion.x`/`Vec3.x`).
- When the wrong slot is out of bounds: strict mode errors (blocks the game's clause "the pure
  core passes `--strict-bytecode` at every landing"); non-strict mode **silently re-runs the
  whole program on the evaluator** — exit 0, and with `--quiet` not even the fallback warning
  (the same fake-healthy masking documented as fake-MATCH in
  [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md)).
- When the wrong slot is **in bounds**: **silently reads the wrong field with no error in any
  mode** — strict included (repro5). This is the worst class: a wrong number that looks like a
  right one.
- Record updates rebuild the record with the **wrong type's field set** (repro6): wrong shape,
  wrong names, values harvested from wrong slots.
- Codegen is **nondeterministic per process** (Go map iteration order) — a likely contributor
  to the VM-vs-interpreter divergence reported as ailang#1355 ("same sim and stdin differ from
  the interpreter in about 1 run in 4"). Fixing this removes an entire nondeterministic
  codegen path; #1355 may have additional causes, which stay tracked separately.

## Root cause (localized; every claim code-verified — see Verification Log)

Two defects compose. Defect A starves the compiler of type information; Defect B guesses.

**Defect A — the lower pass cannot resolve named record receivers.**
`recordFieldSet` (`internal/gen/lower/expr.go:248`) reads the receiver's inferred type from
`CoreTI` and handles only `*types.TRecord` and `*types.TRecord2`. A **named** record type
reaches CoreTI as `&types.TCon{Name: "V"}` (`internal/elaborate/file_funcs.go:352` — signature
annotation `v: V` lowers to `TCon`), which falls into `default: return nil`. So
`stmt.FieldAccess.KnownFields` — the exact, unambiguous hint added by M-BYTECODE-MULTIMODULE M3
— is **empty for every field access whose receiver has a named record type**: parameters,
let-bound locals from calls, function results. Verified empirically: repro4's anonymous
`{x: float, y: float}` parameter emits deterministic slot 0 (10/10 disasm runs); repro2's named
`V` parameter emits the ambiguous fallback (8/10 slot 2, 2/10 slot 0).

Note the typechecker itself can see through the alias — `Unifier.expandAlias`
(`internal/types/unification_core.go:110`) expands `TCon("V")` → `TRecord` (setting
`TRecord.TypeName`), and `propagateTypeNameToCoreTI` (`internal/types/typechecker_substitution.go`)
exists to push nominal names into CoreTI — but CoreTI's entry for the receiver Var stays the
bare `TCon` the elaborator stored. The information exists in the pipeline; the lower pass just
never resolves it.

**Defect B — the compiler's fallback resolves field names by scanning ALL record types.**
`lookupFieldIndex` (`internal/bytecode/compiler/collections.go:265`):

```go
for _, info := range fc.recordTypes {
    if i := info.fieldIndex(name); i >= 0 {
        return i        // ← first type containing the NAME wins; receiver type ignored
    }
}
```

`fc.recordTypes` is the module-wide `map[string]recordTypeInfo` (name → sorted fields) built
from every `TypeDecl` (`internal/bytecode/compiler/compiler.go:47-69`). The scan has no idea
which type the receiver is — it returns the first type whose **field-name set** contains `x`,
in Go map iteration order. With `M` and `V` both registered, `v.x` resolves to `M`'s slot for
`x` (index 2) or `V`'s (index 0) depending on random iteration order.

**Why 8/10 and the reporter's 10/10, not 50/50:** Go small maps live in one 8-slot bucket;
iteration starts at a uniformly random offset in `[0,8)`. `M` was inserted first (TypeDecl
order), so `M` precedes `V` on the walk for 7 of the 8 possible start offsets — P(wrong) = 7/8
per compiled access. The reporter's 10/10 and our 8/10 are the same 7/8 bias. This is also the
nondeterminism channel: the same source compiles to different bytecode in different processes.

`compileRecordUpdate` (`collections.go:289-303`) has the same scan shape for a worse outcome:
it picks the first registered type whose field set **contains all updated fields** and rebuilds
the record with that type's **entire** sorted field list — `vupd` on a 2-field `V` with M picked
emits `GET_FIELD 0`, `GET_FIELD 1`, override `x`, `MAKE_RECORD count=3` with names `a,b,x`
(disasm in Verification Log): a 3-field record with `a`=stolen-from-`x`, `b`=stolen-from-`y`.

**What is sound and must not change:** the VM slot layout itself. `OpMakeRecord` stores values
in sorted-field order **with their names** (`internal/vm/vm.go:405-428`, pseudo-LOAD_CONST
pattern); `OpGetField` is a raw positional read (`vm.go:443`); the `_record_get` builtin
(`internal/vm/builtins.go:148-166`) does an exact **by-name** lookup on the runtime record.
The runtime is correct-by-construction given correct indices — the corruption is purely in
static index resolution. Tuple access `t._0`/`t._1` is positional by design (lower emits it,
`collections.go:189-193`) and is unaffected. ADT switch-case bindings use `FieldIndex` computed
positionally from the variant declaration (`internal/gen/lower/match.go:239-261`) — not the
recordTypes scan — and are unaffected.

## Goals

**Primary Goal:** every `GET_FIELD` slot index is resolved from the receiver's own record type
(named, anonymous, row-polymorphic, or alias-chained) — or not statically resolved at all — so
`--bytecode` and `--strict-bytecode` agree with the interpreter byte-for-byte on all field
accesses, deterministically across compiles.

**Success Metrics:**
- repro2/repro5/repro6/repro7 shapes: strict bytecode returns the interpreter's value **10/10
  consecutive runs** (today: repro5 2/10, repro6 0/10).
- `ailang disasm` on the game-shape program emits byte-identical `GET_FIELD` indices across 10
  consecutive runs (today: two different programs depending on map order).
- Zero stderr fallbacks (`"falling back to evaluator"`) on all new fixtures under plain
  `--bytecode` (today repro2 silently falls back).
- No regression in the existing bytecode test surface: `make test-core`,
  `cmd/ailang/run_bytecode_test.go`, `internal/bytecode/compiler/` unit tests,
  `tests/golden/bytecode/` goldens.

## Solution Design

One unified principle replaces both defects: **resolve the slot from the receiver's own type,
exactly, or fall back to the by-name runtime path — never guess across types.**

### Part 1 — resolve named record receivers at lower time (kills Defect A)

`lower.LowerProgram` already receives the surface AST (`astFile *ast.File`) containing every
`TypeDecl` — the same source the compiler's `recordTypes` is built from. Build a
**type-name → sorted field list** table there (record decls keyed by name; alias decls resolved
to their target's fields, chasing chains with a cycle guard, mirroring
`Unifier.expandAlias`'s fixpoint semantics), thread it through `lowerFuncDecls` → `lowerExpr`
exactly as `cti` is threaded today, and extend `recordFieldSet`:

```go
case *types.TCon:
    if fields, ok := aliasTable[t.Name]; ok {   // named record or alias chain → concrete record
        return fields
    }
    return nil                                   // not a record alias (ADT etc.) → no hint
```

`TRecord` entries whose `TypeName` is set (nominal identity preserved by unification) keep
working through the existing field-set path. `TRecord2` (row-polymorphic) unchanged.

**Multi-module (the game's shape — `Vec3` defined by `sunholo/relativity`, used by
`sim/core.ail`):** the per-module AST does not contain the imported type's decl, so a
per-module table misses exactly the game's case. The merged set already exists where
`internal/runner/vm.go` (`CompileBytecodeFromResult`) collects every loaded module's `TypeDecl`
into one `stmt.Program`; build the merged name→fields table there (or in `lower` when handed
the merged set) so single-file and multi-module share one source of truth — the same decl set
`compiler.Compile`'s Phase 0 sees.

**Record update:** `stmt.RecordUpdate` (`internal/gen/stmt/stmt.go:305-308`) currently carries
only `Base` + `Fields` — no type information at all. Add `KnownFields []string` with the same
semantics as `FieldAccess.KnownFields` (full sorted field set of the base's record type),
populated in `lowerRecordUpdate` from the base expression's type via the same expansion.

### Part 2 — delete the ambiguous scan in the compiler (kills Defect B)

- `lookupFieldIndex` (`collections.go:265-281`): **remove the `for _, info := range
  fc.recordTypes` fallback entirely.** Resolution is: exact `KnownFields` hit → index;
  `KnownFields` present but field absent → error (a typechecker/compiler inconsistency worth
  hearing about, not routing to by-name); no hint → **`_record_get` by-name runtime path**
  (`compileFieldAccessByName`, already exists, correct by construction — the record carries
  its names at runtime). The tuple `_N` positional special-case stays.
- `compileRecordUpdate` (`collections.go:289-303`): **remove the type-guessing scan.** Use the
  new `KnownFields` when present (rebuild in the base's own field order, overriding updated
  fields — exactly today's algorithm, but with the right field list). When absent, return a
  compile error — the existing per-function degradation (`compiler.go:128-140`) tags the
  prototype `EvalOnly` so the evaluator runs that function correctly; in strict mode an
  `EvalOnly` callee is a **loud** error, never a silent corruption.

Post-fix, static index resolution is deterministic by construction — no map iteration
participates in slot choice.

### What this does NOT change

- The VM (`internal/vm/`) — no opcode, layout, or runtime semantics changes.
- The typechecker — `TCon` stays in CoreTI; expansion happens at lower, where the AST decls
  already live. (A typechecker-side alternative — storing the expanded `TRecord` with
  `TypeName` in CoreTI — touches every downstream consumer of CoreTI and is deliberately not
  taken.)
- The interpreter/evaluator path — untouched; all fixtures must keep byte-identical eval output.

### Implementation Plan

| Phase | Work | Files |
|---|---|---|
| 1 | Type-table build + plumbing; `recordFieldSet` TCon case; `RecordUpdate.KnownFields` | `internal/gen/lower/program.go`, `internal/gen/lower/expr.go`, `internal/gen/stmt/stmt.go` (~110 LOC) |
| 2 | Merged table for multi-module; pass into lower per module | `internal/runner/vm.go` (~25 LOC) |
| 3 | Delete scans; hint-or-`_record_get`; update-path hint; compile error → EvalOnly | `internal/bytecode/compiler/collections.go` (−~45/+~35 LOC) |
| 4 | Update hand-built IR unit tests to set `KnownFields`; new table-driven regression tests; repro fixtures | `internal/bytecode/compiler/collections_test.go`, `cmd/ailang/run_bytecode_test.go`, `tests/golden/bytecode/` |

## Examples

**Before (repro2, current HEAD)** — `vx` gets `M`'s slot 2 (8/10 compiles):

```
--- Prototype 1: repro2.vx ---
  0000  GET_FIELD    r1, r0, r2      ; WRONG: M's sorted [a,b,x] → x=2; V's x is slot 0
```

**After (intended)** — `vx` resolves from `V`'s own fields, deterministic in every compile:

```
--- Prototype 1: repro2.vx ---
  0000  GET_FIELD    r1, r0, r0      ; V's sorted [x,y] → x=0 — byte-identical every run
```

**Record update (repro6)** — before: `GET_FIELD r1,r0,r0` / `GET_FIELD r2,r0,r1` /
`MAKE_RECORD count=3` names `a,b,x` (M's shape stolen from a V). After: rebuild with V's
`[x,y]`: override `x=99`, `GET_FIELD` slot 1 for `y`, `MAKE_RECORD count=2` names `x,y`.

## Success Criteria

Each AC names the file that can fail it. "10/10" = ten consecutive process invocations
(map-order randomness is per-process, so this pins determinism, not luck).

- **AC1 (OOB shared name, repro2 shape)**: `--strict-bytecode` returns 13.0 on 10/10 runs.
  Fails today (error, 8-10/10). Test: `cmd/ailang/run_bytecode_test.go` (CLI-level, strict) +
  fixture under `tests/golden/bytecode/`.
- **AC2 (in-bounds silent wrong value, repro5 shape `M={a,x}`, `V={x,y}`)**: strict returns
  **13.0** on 10/10. Fails today (23.0 6/10, 21.0 2/10 — wrong values, exit 0). This is the AC
  a fix that only clamps the index cannot satisfy; it pins semantic correctness.
- **AC3 (record update, repro6 shape)**: strict returns 99.0 on 10/10, and disasm shows
  `MAKE_RECORD count=2`. Fails today (10.0, count=3).
- **AC4 (alias chain, repro7 shape `type P = V`)**: strict returns 10.0 on 10/10. Fails today
  (OOB error).
- **AC5 (game shape)**: `Motion {phi,tau,t,x}` + `Vec3 {x,y,z}` program (minimal mirror of
  `sim/core`) returns the interpreter's value under strict on 10/10 runs, both for
  `.x` on each type.
- **AC6 (codegen determinism)**: 10 consecutive `ailang disasm` runs of the AC5 program emit
  byte-identical instruction streams. Fails today (two distinct `GET_FIELD` index programs).
- **AC7 (no masking)**: for AC1-AC5 fixtures run under plain `--bytecode` (non-strict), stderr
  is free of `"falling back to evaluator"` — the VM itself runs the program to completion.
- **AC8 (generality — anti-special-case)**: table-driven test: 3+ record types sharing two
  field names at distinct positions; receivers reaching the access as (a) named params,
  (b) let-bound locals from calls, (c) function results, (d) alias-named types. Every
  combination returns the interpreter's value under strict. AC1-AC5 alone could be satisfied
  by special-casing one shape; this AC is what forces the real fix.
- **AC9 (no regression)**: `make test-core`, `cmd/ailang/run_bytecode_test.go`,
  `internal/bytecode/compiler/` unit tests (with the documented `KnownFields` updates to
  `collections_test.go`), `internal/bytecode/compiler/multimodule_test.go` (its
  empty-KnownFields→`_record_get` case stays green), and `tests/golden/bytecode/` goldens all
  pass.
- **AC10 (eval path byte-identical)**: interpreter output for every fixture above is
  unchanged by the patch (the fix touches only lower + bytecode compiler).

## Conflict Surface (mandatory — touches `internal/gen/lower/`, `internal/gen/stmt/`, `internal/bytecode/compiler/`)

1. **Positions extended:**
   - `FieldAccess.KnownFields` gains a population source (TCon expansion via the new type
     table) — previously populated only from `TRecord`/`TRecord2` inferred types.
   - `stmt.RecordUpdate` gains a new field (`KnownFields`) — the IR node currently has none;
     emitters other than the bytecode compiler read `RecordUpdate` (e.g.
     `internal/gen/emitgo/funcs.go:354` `emitRecordUpdate`) and ignore the new field
     harmlessly (additive struct field — verified pattern: `FieldAccess.KnownFields` itself
     was added the same way in M3).
   - `lookupFieldIndex` fallback semantics change: scan-all-types → no static resolution.
   - `compileRecordUpdate` type choice: scan → explicit hint.
2. **Other valid constructs already in those positions:**
   - Tuple field access `t._0`/`_1` — positional, resolved by the `_` prefix special-case;
     **kept** (verified: `collections.go:189-193`).
   - Anonymous/row-polymorphic receivers — `TRecord`/`TRecord2` hint path; **unchanged**
     (repro4 control pins it).
   - ADT variant case bindings — `FieldIndex` is positional from the variant decl
     (`lower/match.go`), not from recordTypes; **unaffected** (verified by read; AC9's
     multimodule/golden tests pin it).
   - Record literal registration (`compileRecordLit` → `recordTypes[TypeName]`) — kept; the
     table's purpose is unchanged, only the scan's misuse of it is removed.
   - `TCon` receivers that are **not** record aliases (genuine ADTs like `Result`) — the TCon
     case must return nil (no hint) for them; field access on tagged unions is separately
     gated by the typechecker (`M-TYPECHECK-NO-AUTO-UNWRAP-RESULT`).
3. **How disambiguation works post-change:** the receiver's own type (via CoreTI + the type
   table) or the receiver's own inferred field set — never a cross-type name match. No hint →
   by-name runtime lookup (`_record_get`), which is exact because records carry their field
   names at runtime.
4. **Programs that MUST still work (fixtures verified to exist):**
   `examples/record_cons_pattern.ail` (verified: `ailang check` clean), `examples/record_in_result.ail`,
   `examples/record_list_extraction.ail`, `tests/golden/bytecode/pattern_arity.ail`,
   the existing strict-mode tests in `cmd/ailang/run_bytecode_test.go`
   (`TestCLI_RunBytecode_StrictFails` — strict still fails loudly where it must), and
   `internal/bytecode/compiler/multimodule_test.go` `TestCrossModuleRecord_FieldAccessByName`
   (empty `KnownFields`, deliberately row-polymorphic → `OpBuiltinCall` `_record_get` — behavior
   preserved, now the sole unresolvable path).
5. **Deliberate changes (intentional incompatibilities):**
   - Hand-built-IR unit tests in `collections_test.go` (`getX`, `moveRight`) relied on the
     scan; they gain explicit `KnownFields` — an intentional, documented test-text change.
   - Programs whose receiver type is genuinely unresolvable at lower time now compile to
     `_record_get` (correct, one linear name scan) instead of a possibly-wrong static index;
     record updates in that class become `EvalOnly` (loud in strict mode) instead of
     wrong-shaped records.
   - A `KnownFields` list that does not contain the accessed field now errors at compile
     time (surfacing a lower/typechecker inconsistency) instead of silently scanning other
     types.

## Testing Strategy

- **CLI-level strict determinism tests** (`cmd/ailang/run_bytecode_test.go`): AC1-AC5, AC7,
  AC8 — run each fixture N=10 via the existing `runCLI` helper, assert identical correct
  output and no fallback marker (mirrors `TestCLI_RunBytecode_StrictFails`'s harness).
- **Compiler unit tests** (`internal/bytecode/compiler/collections_test.go`): hand-built IR
  with `KnownFields` — assert exact `GET_FIELD` index and `MAKE_RECORD` count; updated
  existing tests per Conflict Surface #5.
- **Disasm determinism test** (AC6): compile the game-shape program 10× in-process or via
  `ailang disasm`, assert byte-identical output.
- **Golden fixtures**: repro programs committed under `tests/golden/bytecode/`
  (`shared_field_oob.ail`, `shared_field_inbounds.ail`, `shared_field_update.ail`,
  `alias_chain_access.ail`, `motion_vec3_shape.ail`) with exact expected outputs.
- **Whole-surface regression**: `make test-core` + full compiler test package + existing
  goldens after every phase.

## Timeline

- Day 1 (morning): Phases 1-2 (lower + plumbing) + repro2/4/5 passing; Day 1 (afternoon):
  Phase 3 (compiler scans removed) + AC1-AC8 green; Day 2 (buffer): AC9 sweep, `make fmt` /
  `make lint` / `make check-boundaries`, changelog entry, goldens. The 2× rule says call it
  2 days; the root cause is fully localized (unlike the parity doc's investigation-first
  lanes), so risk concentrates in test updates, not discovery.

## Non-Goals

- **Cross-module same-name record types**: two modules each defining `Point` with different
  field sets collide in today's `recordTypes`/merged table (bare-name keys, last-wins —
  pre-existing, `runner/vm.go` dedup comment says single-source-of-truth per name is the
  contract). Post-fix such programs take the by-name fallback (correct) rather than a wrong
  slot, but a module-qualified resolution is a separate design. Parked; noted here so a
  reviewer doesn't mistake it for silently absorbed.
- **ailang#1355's other causes** (if any): this fix removes one nondeterministic codegen path
  (map-order slot choice). Any remaining VM/interpreter divergence keeps its own tracking.
- The parity harness's classification scheme (stays with m-bytecode-vm-parity-bugs.md's
  parked A2).
- Closing M-BYTECODE-2E bridge gaps; VM opcode/layout changes; typechecker CoreTI semantics.
- Perf work on `_record_get` (linear scan) — it is now the rare path, not the hot one.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-------------|
| Hint (KnownFields) disagrees with the runtime record's actual layout (e.g., a by-name-built record reaching a hinted access) | High — wrong slot returns | The slot rule is uniform everywhere: **alphabetically sorted field list** — `MAKE_RECORD` stores in sorted order with names attached, `KnownFields` is sorted at lower, `recordTypeInfo` sorted at Phase 0. AC8's (b)/(c) receiver shapes (locals from calls, function results) exercise exactly this; `_record_get` never produces a slot. |
| Alias chains with cycles (`type A = B; type B = A`) hang the table build | Medium | Cycle-guarded fixpoint walk, mirroring `expandAlias`'s `seen` set (verified pattern at `unification_core.go:126-131`); test included. |
| Cross-module TCon names not in the merged table (stdlib `std/…` records) degrade to by-name | Low (correct, slightly slower) | AC9's multimodule tests + golden corpus catch any unexpected `_record_get` growth; disasm diff on corpus shows where static resolution was lost. |
| Hand-built IR unit tests miss scan-removal breakage beyond `collections_test.go` | Medium | `grep -rn "range fc.recordTypes"` is exhaustive (2 hits, both removed sites); full compiler test package runs in AC9. |
| `KnownFields` present-but-missing-field now hard-errors where it used to scan | Low | That case was producing wrong slots today when names collided (repro2) and only "worked" by scan luck otherwise; loud error is the intended semantics (CLAUDE.md #2 no silent fallbacks). |
| Perf: more `_record_get` calls than today in some shapes | Low | `_record_get` was already the unresolvable path; Part 1 *shrinks* its reach (named types now resolve statically — the game's `Motion`/`Vec3` shape goes fully static). |

## Related Documents

- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md) — parent parity
  doc (pattern-arity #505, closure family, unsafe replay, harness honesty). This doc's defect
  is distinct: fix sites are `lower/expr.go` + `compiler/collections.go`, none of that doc's
  lanes. Its fake-MATCH finding (silent `--quiet` fallback) is the same masking this doc's
  repro2 exhibits.
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — sibling
  soundness fix (#505), same P0 class, different path; read for AC/test conventions.
- [implemented M-BYTECODE-MULTIMODULE](../../implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md) — introduced
  `FieldAccess.KnownFields` + `_record_get` fallback (M3), the machinery this fix completes.
- [implemented M-BYTECODE-VM](../../implemented/v0_11_0/m-bytecode-vm.md) — VM/slot-layout
  master design.
- Stapledon mission log (external): `sunholo-data/stapledons-godot/design_docs/stapledon-mission-log.md`
  — M1.6a park record naming ailang#1354/#1355.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Slot resolution becomes a pure function of (receiver type, field name) — map-order nondeterminism in codegen eliminated; VM and interpreter agree byte-for-byte on all fixtures |
| A7: Machines First | +1 | Strict gate becomes a trustworthy signal for record-heavy programs (the game's landing clause); silent wrong values exit the surface |
| A11: Structured Failure | +1 | The unresolvable case degrades loudly (`_record_get` by-name, `EvalOnly` + strict hard error) instead of silently guessing |
| Others | 0 | No language surface, effect, or authority change |

**Net Score: +3** → Proceed. Hard violations: none (A3/A4 untouched).

## Verification Log (first-party, this worktree, binary v0.47.0 f5bf7291 unless noted)

| Claim | How verified |
|-------|--------------|
| repro2: interp 13.0 / non-strict 13.0 / strict `GET_FIELD: index 2 exceeds field count 2 (in repro2.vx at repro2.ail:7, ip 0)` | live `ailang run` ×3 modes, `/tmp/repro/repro2.ail` (exact reporter output reproduced) |
| `vx` slot nondeterminism: index 2 in 8/10 disasm compiles, index 0 in 2/10 | 10× `ailang disasm --relax-modules repro2.ail`, grep of the `vx` prototype's `GET_FIELD` operand |
| repro5 (in-bounds): strict returns 23.0 ×6, 21.0 ×2, 13.0 ×2 — silent wrong values, exit 0 | 10× `ailang run --strict-bytecode` on `/tmp/repro/repro5.ail`; 21.0 = BOTH accesses wrong (m.x→m.a, v.x→v.y) |
| repro5 interpreter = 13.0 (correct) | live `ailang run` (no --bytecode) |
| repro6 (update): strict returns 10.0 ×6 (correct 99.0); interp 99.0 | 6× strict run + 1 interp run on `/tmp/repro/repro6.ail` |
| repro6 vupd compiles with M's shape: `GET_FIELD r1,r0,r0`; `GET_FIELD r2,r0,r1`; `MAKE_RECORD r1, src=r1, count=3`; name consts `a`,`b`,`x` | `ailang disasm` on repro6 |
| repro7 (alias `type P = V`): strict OOB error index 2; interp 10.0 | live run + interp on `/tmp/repro/repro7.ail` |
| Control repro4 (anonymous `{x,y}` param): slot 0 deterministic 10/10 disasm; strict 13.0 | 10× disasm + strict run on `/tmp/repro/repro4.ail` |
| Defect A: `recordFieldSet` handles only TRecord/TRecord2; TCon → `default: return nil` | read `internal/gen/lower/expr.go:244-277` (full switch) |
| Signature annotations lower to `TCon{Name}` | read `internal/elaborate/file_funcs.go:332-356` (`astTypeToInternalType`, SimpleType default) |
| CoreTI stores per-NodeID types; `GetForExpr(expr)` = `cti[expr.ID()]` | read `internal/types/typeinfo.go:74-80` |
| Defect B: scan loop `for _, info := range fc.recordTypes` returns first name match | read `internal/bytecode/compiler/collections.go:265-281` |
| Update scan picks first type containing all updated fields | read `collections.go:289-303` |
| `recordTypes` built from TypeDecls at Phase 0, keyed by bare `td.Name`, only `ADTDecl`/`RecordDecl` cases (TypeAliasDecl NOT registered) | read `internal/bytecode/compiler/compiler.go:47-69` |
| 7/8 map-order bias: single 8-slot bucket, `M` inserted first, only start-offset 1 yields `V` first | Go map layout semantics; matches measured 8/10 and reporter's 10/10 |
| Non-strict fallback: `entrypoint.go` runs VM, on error falls through to evaluator; warning `if !params.Quiet` | read `internal/runner/entrypoint.go:145-160`, warning at `:153` |
| Strict = bridge unwired + hard error (no fallback) | read `internal/runner/vm.go:202-233`, `--strict-bytecode` help `cmd/ailang/main_run.go:122` |
| VM slot layout: MAKE_RECORD sorted-order values + name constants; GET_FIELD raw positional read; `_record_get` exact by-name | read `internal/vm/vm.go:405-455`, `internal/vm/builtins.go:148-166` |
| Tuple `_N` access positional (unaffected) | read `collections.go:189-193` |
| ADT case bindings positional from variant decl (unaffected) | read `internal/gen/lower/match.go:230-262` (`FieldIndex: i`) |
| `stmt.RecordUpdate` carries no type info (Base + Fields only) | read `internal/gen/stmt/stmt.go:305-308` |
| `stmt.FieldAccess.KnownFields` exists with exact these semantics ("Empty means fall back to recordTypes lookup") | read `internal/gen/stmt/stmt.go:272-285` |
| No type table is plumbed into lower today: `lowerFuncDecls`/`lowerExpr` take only Core + cti; `LowerProgram` receives `astFile` but never extracts a name→fields map | read `internal/gen/lower/program.go:21-72`, `internal/gen/lower/expr.go` signatures |
| `LowerProgram` receives the AST incl. TypeDecls (fix's table source) | read `internal/gen/lower/program.go:33-39` (`lowerTypeDecls(astFile)`) |
| Multi-module: runner lowers each module with `mod.File` (its own AST), collects every module's TypeDecls into one `stmt.Program` | read `internal/runner/vm.go:62-115` |
| `LowerMultiModule` has no non-test callers (runner hand-rolls) | `grep -rn "LowerMultiModule" --include="*.go" internal/ cmd/` → definition + doc comment only |
| Hand-built IR unit tests rely on the scan (`getX`, `moveRight`: `FieldAccess`/`RecordUpdate` with no KnownFields + registered type) | read `internal/bytecode/compiler/collections_test.go:95-175` |
| `multimodule_test.go` `TestCrossModuleRecord_FieldAccessByName` pins empty-KnownFields → `OpBuiltinCall _record_get` | read `internal/bytecode/compiler/multimodule_test.go:363-380` |
| Other emitters read `stmt.RecordUpdate` (additive-field claim grounded) | `grep -rn "RecordUpdate" internal/gen/emitgo/` → `funcs.go:232-233, 354` (`emitRecordUpdate`) |
| Fixtures exist: `examples/record_cons_pattern.ail` (checks clean), `record_in_result.ail`, `record_list_extraction.ail`, `tests/golden/bytecode/pattern_arity.ail`, `cmd/ailang/run_bytecode_test.go` (strict tests at :82, :142-162) | `ls` + `ailang check` (clean) + read |
| Alias expansion with cycle guard exists as a pattern to mirror | read `internal/types/unification_core.go:110-195` (`seen` set, fixpoint) |
| Compile failure per function → EvalOnly degradation exists | read `internal/bytecode/compiler/compiler.go:128-140` |
| Game symptom maps: Motion `{phi,tau,t,x}` sorted → `x`=3; Vec3 field count 3 → `index 3 exceeds field count 3` | reporter's exact error string (task + mission-stapledon controlplane message); slot arithmetic from sorted order |
| Mission-stapledon M1.6a parked on #1354/#1355 (both v0.45.0 and v0.47) | `ailang messages list --unread --json`, message `inbox_1790578890432_a8028692` (controlplane inbox, 2026-09-28) |

---

**Document created**: 2026-09-28
**Author**: design-doc-creator skill invocation (coordinator task-f8ab4375, ailang#1354)

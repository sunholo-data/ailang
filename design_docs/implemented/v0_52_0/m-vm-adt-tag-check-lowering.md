# M-VM-ADT-TAG-CHECK-LOWERING — mixed-match ADT tag checks lower as record field access (`_record_get` on an ADT): strict VM crash on constructor sub-patterns (e.g. `TText(t) :: r`), broken Go from `--emit-go-v2`

**Status**: Superseded by / implemented in [m-vm-match-lowering.md](m-vm-match-lowering.md) (2026-10-02, v0.51.1) — one recursive lowering fixes this doc together with its three siblings. Original status: Planned
**Target**: v0.51.3 (bug fix; strict-VM soundness, clause-2 parity family)
**Priority**: P0 — the reported shape (constructor pattern at the head of a cons pattern in a mixed-arm match) crashes `--strict-bytecode` with a loud VM error; plain `--bytecode` masks it by silently falling back to the evaluator; `ailang compile --emit-go-v2` emits Go that cannot build for the same shape.
**Estimated**: ~2 days (root cause fully localized; the correct opcode machinery — `OpGetTag`, tag ordinals, `inferADTFromCases` — already exists in the switch path and is reused)
**Dependencies**: none. Sibling of [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) and [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md); **both siblings' solutions, as written, build on the exact broken construct this doc fixes** (see Related Documents — this doc is effectively a prerequisite correction to their premises).

**Reported from**: stapledons-godot `sim/markers.ail` (sprint R1-AI-FOUNDATION AI.1), where the constructor-as-cons-head pattern had to be worked around by matching the head separately (`x :: r => match x { TText(t) => ... }`).

## Problem Statement

**Current State:**

When a `match` mixes pattern kinds (e.g. `[] => ..., TText(t) :: r => ..., TMark(m) :: r => ...`),
`LowerMatchStmt` routes it to the if-chain lowering (`internal/gen/lower/match.go:26`, because
`allConstructorPatterns` at :77 is false for `ListPattern` arms). Inside that path, ADT tag
checks are emitted as a **record** field access on an **ADT** value:

```go
// internal/gen/lower/match.go:399 (top-level constructor arm in an if-chain)
Left:  stmt.FieldAccess{Record: scrutinee, Field: "Tag"},

// internal/gen/lower/match.go:432 (constructor sub-pattern as a list element / cons head)
Left:  stmt.FieldAccess{Record: head, Field: "Tag"},   // head = _list_get(scrutinee, i)
```

`stmt.FieldAccess` is record/tuple machinery. With neither `KnownFields` nor `RecordType`
set (both emit sites leave them empty), the bytecode compiler cannot resolve a static index
(`internal/bytecode/compiler/collections.go:221-244`), falls back to the `_record_get` builtin
(`collections.go:262`), and the strict VM rejects it:

```
Error: bytecode execution failed: vm: vm: BUILTIN_CALL: _record_get: arg 0 must be record,
got ADT (in r1.first at r1.ail:3, ip 19, op BUILTIN_CALL)
```

(`internal/vm/builtins.go:174` — `_record_get` requires `TagRecord`; the value is `TagADT`.)

The interpreter never runs this lowering (the tree-walking evaluator pattern-matches on Core
directly — `internal/eval/eval_patterns.go:105`), so `ailang run` returns the right answer. And
plain `--bytecode` **silently returns the evaluator's answer too**: on VM error the non-strict
runner falls back (`internal/runner/entrypoint.go:144-159`), and the `--quiet` flag **defaults
to true** (`cmd/ailang/main_run.go:26`), so the warning line is suppressed. Verified with
`--verbose` (V6): `⚠ bytecode path unavailable (vm: ... _record_get ...); falling back to
evaluator` — the "quietly uses another path" from the bug report is the evaluator via fallback.

The same lowering feeds `ailang compile --emit-go-v2`: the emitted Go for the reported shape is
`BuiltinList_get(toks, int64(0)).Tag == "TText"` and `t := _head_0._0` (V13) — but the
generated ADT struct has `Kind` (an enum) and `Value0…` fields (`internal/gen/emitgo/types.go`),
so the Go **does not build** (no `Tag` field, no `_0` field).

**Verified symptom table** (binary v0.51.0 `b99dd25c`, all commands run in this session; programs
in the Verification Log):

| # | Shape | `ailang run` (evaluator) | `run --bytecode` (non-strict) | `run --bytecode --strict-bytecode` | `compile --emit-go-v2` |
|---|---|---|---|---|---|
| 1 | `TText(t) :: r` (mixed match, ctor head — **the report**) | `hi` | `hi` (silent evaluator fallback) | **error**: `_record_get` on ADT | **broken Go**: `.Tag`, `._0` on ADT struct |
| 2 | all-constructor match `match t { TText(s) => …, TMark(s) => … }` | `thi` | `thi` | `thi` (SwitchStmt path — healthy) | builds |
| 3 | head matched separately (`x :: r => match x { TText(t) => … }`) | `hi` | `hi` | `hi` (today's workaround) | — |

**Impact:**

- The cons-head constructor pattern is ordinary AILANG (`ailang check` passes — V4); it
  type-checks, runs on the evaluator, then **crashes the strict VM**, whose entire purpose is
  to guarantee that what runs on the evaluator also runs without it.
- The non-strict bytecode path hides the breakage (quiet-default fallback), so the gap looks
  like "strict mode is picky" rather than "the bytecode for this construct is wrong".
- Any AI-generated code that destructures a token/ADT stream with `Ctor(x) :: rest` — the
  natural idiom for token list walking — hits it immediately (external consumer had to
  restructure source around it).
- Two planned sibling docs (nested-pattern lowering, var-pattern default arm) both specify
  solutions that **emit this exact construct**; without this fix they would land the same
  crash into their newly-compiled shapes (V14, V15).

## Root Cause

`internal/gen/lower/match.go` has no expression-level way to say "compare this value's ADT
tag". The switch path doesn't need one — `SwitchStmt` carries tag strings that
`internal/bytecode/compiler/switch.go:48-66` translates to `OpGetTag` + ordinal `OpEq`. The
if-chain path builds its conditions out of plain `stmt.Expr`, so whoever wrote it reached for
the nearest record-shaped node — `FieldAccess{Field: "Tag"}` — which is only ever compiled as
a record/tuple field access. Two emit sites (match.go:399 for a top-level constructor arm,
match.go:432 for constructor sub-patterns inside list patterns) lower ADT tag checks this way.

Disassembly of the reported repro (V5) shows the consequences precisely:

```
0016  BUILTIN_CALL r5, builtin#514, argc=2     ; _list_get(toks, 0)      — head
0018  LOAD_CONST   r10, #3                    ; "Tag"
0019  BUILTIN_CALL r8, builtin#517, argc=2     ; _record_get(head,"Tag") — CRASHES on ADT
...
0028  GET_FIELD    r3, r2, r0                  ; t := head field 0       — would work
```

The binding half is already sound — `FieldAccess{Field: "_0"}` compiles to positional
`OpGetField`, which the VM handles on ADTs (`internal/vm/vm.go:486-491`), same as the switch
path's case bindings. Only the **condition** half is broken, and only for ADT tag checks.

Why it went unnoticed: the golden corpus (`tests/golden/bytecode/pattern_arity.ail`) exercises
only Var-pattern heads (`a`, `[a, b]`, `[p, ...rest]`, `::(h, t)`), and the repo's own examples
contain no constructor-at-cons-head pattern (V18) — the construct reached the compiler first
via an external consumer.

## Goals

**Primary Goal:** An ADT tag check emitted by the if-chain match lowering compiles to correct
VM code and correct Go, so the reported repro returns `hi` under `--strict-bytecode` and its
`--emit-go-v2` output builds — byte-for-byte the evaluator's answer.

**Success Metrics:**
1. r1.ail (the reported repro) returns `hi` under `--strict-bytecode` (today: VM error).
2. `ailang compile --emit-go-v2` on the same shape emits Go that builds (verification gate
   green on a machine with `go` in PATH) and computes the same value.
3. The two `FieldAccess{Field: "Tag"}` emit sites no longer exist (grep-verified); tag checks
   in lowered IR are expressed by a dedicated node both backends handle.
4. No constructor sub-pattern shape that compiles today changes behavior (strict-VM sweep of
   `std/` + `examples/` + goldens: zero behavioral diffs).
5. Shapes newly reachable-but-incomplete (literal/nested constructor args in if-chain
   sub-positions) fail **loudly** as EvalOnly — never silently wrong (see High-Impact
   Decisions).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add a dedicated `stmt.Expr` node for ADT tag equality (e.g. `ADTTagEq{Value, TypeName, Tag}`) rather than (a) special-casing field name `"Tag"` in the compiler, or (b) adding an `_adt_tag` name-string builtin | (a) is unsound: `Tag` is a legal user record field (`type P = {Tag: string, x: int}` type-checks — V11), so the compiler cannot distinguish record access from tag access without IR-level information. (b) adds runtime/builtin surface and a second tag idiom (name strings) alongside the ordinal idiom the switch path already uses. A node that compiles to the **existing** `OpGetTag` keeps the VM opcode/builtin surface frozen | human (design) | design | med |
| Tag→ordinal resolution reuses the compiler's existing ADT machinery (`fc.adtTypes` + `fc.adtOrder`, i.e. the `switch.go` tagOrdinal lookup and `inferADTFromCases` fallback factored into a shared helper) | Guarantees the same determinism contract the switch path already carries (declaration-order resolution, ailang#1355) and no second tag-resolution system | agent | design | low |
| The lower pass fills `TypeName` from type info when it knows it (scrutinee/list-element type via `cti`); empty `TypeName` triggers the compiler's deterministic inference fallback | Avoids hard-wiring type propagation through the whole lowering for the general case while keeping determinism | agent | design | low |
| Fix **both** emit sites: match.go:432 (list/cons heads — the report) and match.go:399 (top-level constructor arms in if-chains, currently near-unreachable post-typecheck but **made reachable by the var-pattern sibling's routing fix**) | Fixing only :432 leaves a crash the sibling doc's gate would turn on; the sibling doc explicitly claims "the cond side needs no change" (V15) — false today, and false after their fix unless :399 is repaired here | agent | design | low |
| Depth-1 binding contract for constructor sub-patterns in the if-chain path (list elements): `VarPattern` args bind (existing `FieldAccess "_j"` → `OpGetField`, already works on ADTs); `LitPattern`/nested args **panic loudly** → EvalOnly stub — same contract as [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md) | Without the loud boundary, fixing only the tag check would convert today's loud crash (r2: `TText("hi") :: r`) into a **silent wrong answer** (the literal arg is not compared — the evaluator returns `other`, the VM would return the arm body). NO-SILENT-FALLBACKS; the nested-pattern sibling doc owns the recursive general solution | compiler | design | low |
| emitgo (v2 pipeline) emits `.Kind == <Type>Kind<Tag>` for the new node and resolves `FieldAccess` on a value whose `RecordType` names a known ADT to `.Value<j>`; unresolvable cases are **loud emitter errors**, never silent `nil` | Keeps the v2 backend consistent with the fix instead of leaving it emitting Go that does not build; the emitter's existing `default: nil /* unknown expr */` silent fallthrough is not extended | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] IR node vs builtin vs compiler special-case (resolved: IR node, rationale above)
- [x] Ordinal-based comparison via `OpGetTag` (matches switch path; no name-string builtin)
- [x] Loud EvalOnly for Lit/nested constructor args in if-chain sub-positions (no silent wrongs)
- [x] Both emit sites fixed (:399 and :432)
- [ ] Final node name and exact struct shape (`ADTTagEq` proposed) — sprint-planner may
      ratify or rename; behavior contract frozen

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact node name/fields (`ADTTagEq{Value, TypeName, Tag}` proposed) — agent may choose; the
  compile contract (OpGetTag + ordinal + OpEq) is frozen.
- Whether the shared ordinal resolver lives in `switch.go` (exported helper) or a new small
  file in `internal/bytecode/compiler/` — agent may choose.
- Whether conds hoist `_list_get` heads into temps or re-emit them (cond + bindings both
  reference the head; both are pure — agent may choose; today's code re-emits).
- emitgo fallback shape when `TypeName` is empty and inference is impossible in the emitter:
  loud error message wording — agent may choose (must be loud).
- Unit-test layout (extend `lower_match_test.go` vs a new file) — agent may choose.

## Solution Design

### Overview

Give the if-chain lowering the one expression it has always been missing: **"is this value's
ADT tag equal to `<Tag>`?"** as a first-class `stmt.Expr` node, compiled by the bytecode
compiler to the same instruction sequence the switch path already emits (`OpGetTag` →
ordinal const → `OpEq`), and by emitgo to the same Go comparison its switch emitter already
writes (`.Kind == <Type>Kind<Tag>`). Then replace the two record-field-access spellings of
tag checks in `internal/gen/lower/match.go` with that node, and add the loud EvalOnly
boundary for constructor args the if-chain path cannot yet bind correctly (literals, nested
patterns), mirroring the var-pattern sibling's contract.

No VM change. No new opcode. No new builtin. No parser/typechecker change. The fix is
confined to: `internal/gen/stmt` (one node), `internal/bytecode/compiler` (one expr case +
refactor of existing tag-resolution into a shared helper), `internal/gen/lower/match.go`
(two emit sites + loud panics), `internal/gen/emitgo` (two cases + ADT-aware FieldAccess),
plus tests.

### Architecture

**1. New stmt IR node** (`internal/gen/stmt/stmt.go`, `validate.go`):

```go
// ADTTagEq reports whether Value is an ADT of the named type whose variant
// tag equals Tag. TypeName, when empty, defers to the compiler's
// deterministic ADT inference (same contract as SwitchStmt's ADTName).
type ADTTagEq struct {
    Value    Expr  // the scrutinee/sub-scrutinee expression
    TypeName string // e.g. "Tok"; may be ""
    Tag      string // e.g. "TText"
}
```

**2. Bytecode compiler** (`internal/bytecode/compiler/expr.go` + `switch.go` refactor):
compile `ADTTagEq` as: compile `Value` into a reg; `OpGetTag` into a temp; load the ordinal
constant; `OpEq` — the identical shape to `compileSwitch`'s per-case test
(`switch.go:48-70`). Tag→ordinal resolution reuses `fc.adtTypes[name].tagOrdinal[Tag]`;
when `TypeName` is empty, resolve by scanning `fc.adtOrder` for an ADT whose tag set contains
`Tag` — a single-tag sibling of `inferADTFromCases` (switch.go:141-159), factored into a
shared helper so switch and if-chain share one resolution path and one determinism contract
(declaration order, never Go map order — ailang#1355).

**3. Lower pass** (`internal/gen/lower/match.go`):
- Site :399 (`lowerPatternCond`, `ConstructorPattern` case): emit
  `stmt.ADTTagEq{Value: scrutinee, TypeName: adtName-or-"", Tag: p.Name}` — `adtName` is
  derivable here from `cti` when the scrutinee's type is known (the caller has `cti`;
  thread it into `lowerPatternCond`, which today doesn't take it — see Files).
- Site :432 (`lowerPatternCond`, `ListPattern` constructor-element checks): same node on the
  `_list_get(scrutinee, i)` head expression, with TypeName from the list's element type when
  available.
- `lowerPatternBindings` ListPattern/ConstructorPattern case (match.go:497-524): keep the
  Var-arg binding (`FieldAccess "_j"` — already correct on the VM); replace the silent skip
  of Lit/nested args with a **loud panic** carrying a precise message (the existing
  panic→EvalOnly channel, `internal/gen/lower/program.go:146-215`, converts it into an
  EvalOnly stub — strict VM fails loudly, non-strict runs correctly via the bridge).

**4. emitgo** (`internal/gen/emitgo/funcs.go`):
- `ADTTagEq` → `<expr>.Kind == <Type>Kind<Tag>` — the exact shape `emitSwitchCase` already
  writes (`funcs.go:171-190`), from the same TypeDecl information.
- `FieldAccess` whose `RecordType` names a known ADT (TypeDecl present) → `.<Value j>` —
  matching the generated struct fields (`types.go:72-113`). The lower pass sets `RecordType`
  at the two binding sites it already emits `FieldAccess "_j"` on ADT heads. Unresolvable →
  loud emitter error.

### Implementation Plan

**Phase 1: IR node + VM compilation** (~0.5 day)
- [ ] Add `ADTTagEq` to `internal/gen/stmt` (+ `validate.go` case).
- [ ] Factor switch.go's tag-ordinal lookup into a shared resolver; add the single-tag
      inference variant.
- [ ] Compile the node in `internal/bytecode/compiler/expr.go` (mirroring
      `compileSwitch`'s test emission).
- [ ] Unit tests: node compiles to GET_TAG/LOAD_CONST/EQ with the correct ordinal; inference
      is declaration-order deterministic; unknown tag → loud compile error.

**Phase 2: Lowering swap + loud boundary** (~0.5 day)
- [ ] Thread `cti` into `lowerPatternCond`; replace both `FieldAccess{"Tag"}` sites with
      `ADTTagEq` (filling TypeName where derivable).
- [ ] Loud panic for Lit/nested constructor args in the if-chain ListPattern
      ConstructorPattern case; keep Var/Wildcard binding.
- [ ] Update `TestLowerMatchStmt_ConsConstructorHead` (its `containsTagCheck` helper asserts
      the old `FieldAccess{Field:"Tag"}` spelling — must assert the new node; this is a
      deliberate test change, see Conflict Surface).
- [ ] Unit tests: r1 shape lowers to `ADTTagEq` on the `_list_get` head; Var args bound;
      Lit arg → panic with message naming the shape.

**Phase 3: emitgo + end-to-end** (~0.5 day)
- [ ] emitgo cases for `ADTTagEq` and ADT-aware `FieldAccess`; loud errors otherwise.
- [ ] `--emit-go-v2` on the repro: emitted Go contains `.Kind == TokKindTText` and
      `.Value0`; verify with `go build` on a toolchain machine (this sandbox has no `go` —
      V20).
- [ ] Strict-VM golden: the reported repro + nullary-constructor head + head-separated
      equivalence row.
- [ ] Sweep `std/`, `examples/`, goldens under `--strict-bytecode`; expect zero behavioral
      diffs (corpus contains no constructor-at-cons-head pattern — V18).
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries`.

### Pipeline Pass Coverage

- [x] This change adds no new pipeline pass; it modifies the output of the existing lower
      pass (`prog.Decls` → `LowerMatchStmt`) consumed by both bytecode and emitgo.
- [x] Contract expressions (`requires`/`ensures`) do not contain match patterns today
      (`internal/smt/parser_contracts.go` surface is expression-level); no contract-path
      change is needed.

### Files to Modify/Create

**New files:**
- `tests/golden/bytecode/cons_head_ctor.ail` (+ golden rows) — reported repro + variants,
  ~30 LOC AILANG

**Modified files:**
- `internal/gen/stmt/stmt.go` — `ADTTagEq` node (~15 LOC)
- `internal/gen/stmt/validate.go` — visitor case (~5 LOC)
- `internal/bytecode/compiler/expr.go` — compile `ADTTagEq` (~25 LOC)
- `internal/bytecode/compiler/switch.go` — factor shared ordinal resolver (~15 LOC net)
- `internal/gen/lower/match.go` — two site swaps + loud panics + `cti` threading (~40 LOC)
- `internal/gen/lower/lower_match_test.go` — update `containsTagCheck`; new tests (~60 LOC)
- `internal/gen/emitgo/funcs.go` — `ADTTagEq` + ADT-aware `FieldAccess` (~25 LOC)
- `changelogs/` entry + reference-page note (~10 LOC)

## Examples

### Example 1: The reported repro — before/after

```ail
module r1
type Tok = TText(string) | TMark(string)
pure func first(toks: [Tok]) -> string =
  match toks {
    [] => "empty",
    TText(t) :: r => t,
    TMark(m) :: r => m
  }
export func main() -> string = first([TText("hi")])
```

**Before** (verified this session, V1-V3, V6):

```
$ ailang run --quiet --relax-modules --entry main r1.ail                → hi
$ ailang run --quiet --relax-modules --bytecode --entry main r1.ail     → hi   (evaluator fallback, warning suppressed by quiet-default)
$ ailang run --quiet --relax-modules --bytecode --strict-bytecode --entry main r1.ail
Error: bytecode execution failed: vm: vm: BUILTIN_CALL: _record_get: arg 0 must be record,
       got ADT (in r1.first at r1.ail:3, ip 19, op BUILTIN_CALL)
```

**After:** all three print `hi` (and `--verbose --bytecode` no longer prints the fallback
warning — the VM run succeeds).

### Example 2: Lowered IR, before/after

**Before:** `BinOp{OpEq, FieldAccess{Record: BuiltinCall{_list_get, [scrut, 0]}, Field: "Tag"}, LitString{"TText"}}` — a record field access on an ADT, compiled to `BUILTIN_CALL _record_get` (crash).

**After:** `ADTTagEq{Value: BuiltinCall{_list_get, [scrut, 0]}, TypeName: "Tok", Tag: "TText"}` — boolean by construction, compiled to `GET_TAG + LOAD_CONST(ordinal) + EQ` (the switch path's own instruction shape).

### Example 3: The loud boundary — `TText("hi") :: r`

**Before:** loud crash (the tag access errors before the literal is ever consulted).
**After tag-fix alone (rejected intermediate):** the arm would match `TText("bye")` silently —
the evaluator returns `other` (V7). **After this doc:** EvalOnly with a precise reason
(strict errors loudly; non-strict returns the evaluator's `other` via the bridge), until the
nested-pattern sibling doc delivers recursive literal guards on top of this node.

## Success Criteria

- [ ] AC1: r1.ail returns `hi` under `--strict-bytecode` (golden row; also
      `--verbose --bytecode` shows VM dispatch, no fallback warning).
- [ ] AC2: `grep -rn 'Field: "Tag"' internal/gen/lower/` is empty; both sites emit the new node.
- [ ] AC3: `ailang compile --emit-go-v2` on r1.ail produces Go containing
      `.Kind == TokKindTText` / `.Value0` that builds under the verification gate.
- [ ] AC4: `TText("hi") :: r` shape is a loud EvalOnly on strict (not a silent wrong arm);
      non-strict returns the evaluator's `other`.
- [ ] AC5: All regression fixtures (Conflict Surface list) pass unchanged; `make test`,
      `make fmt`, `make lint`, `make check-boundaries` green.
- [ ] AC6: Strict-VM sweep of `std/` + `examples/` + `tests/golden/bytecode/` — zero
      behavioral diffs, zero new EvalOnly functions.
- [ ] All tests passing
- [ ] Documentation updated (reference note + changelog)

## Testing Strategy

**Unit tests:**
- `internal/gen/lower/lower_match_test.go`: lowered-IR assertions — `ADTTagEq` present for
  cons-head constructor and top-level constructor arms; Var args bound via `FieldAccess
  "_j"`; Lit/nested args panic with the documented message. Update `containsTagCheck` to
  match the new node (one helper, same intent).
- `internal/bytecode/compiler`: `ADTTagEq` compiles to GET_TAG + ordinal + EQ; declaration-
  order inference determinism (mirror `determinism_test.go`'s multi-ADT shape); unknown
  tag/type → loud compile error.

**Integration / goldens:**
- `tests/golden/bytecode/cons_head_ctor.ail`: rows for `TText(t) :: r` / `TMark(m) :: r` /
  nullary `None :: r`-style head / empty list / head-separated-workaround equivalence —
  evaluator output == strict-VM output, asserted exactly (model: `pattern_arity.ail` rows
  in `cmd/ailang/run_bytecode_test.go:166`).
- `--emit-go-v2` golden for the same shape (codegen regression pin).

**Regression-surface tests** (one per "Programs that MUST still work" entry):
- `tests/golden/bytecode/pattern_arity.ail` (all-Var heads; exact unchanged output).
- `examples/pattern_matching_adt.ail` (switch-path constructor matches — untouched).
- `examples/runnable/list_pattern_cons.ail`, `examples/runnable/recursion_quicksort.ail`,
  `examples/runnable/pattern_sugar.ail` (mixed/nested list matches must not regress).
- `std/json.ail` (final wildcard defaults on the switch path — untouched).
- Existing `lower_match_test.go` regressions (M-LOWER-FIX, Bug A.2) pass with only the
  `containsTagCheck` spelling change.

**Manual testing:**
- Re-run this doc's Verification Log command set on the built binary.
- On a toolchain machine: `ailang compile --emit-go-v2` with the `go build` verification
  gate enabled (this sandbox has no `go` in PATH — V20).

## Non-Goals

**Not in this feature:**
- Recursive sub-pattern conds/bindings (nested patterns `a :: b :: rest`, literal **elements**
  `1 :: r`, tuple/record element guards) — owned by
  [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md);
  that doc must build its recursive constructor cases **on the node this doc introduces**
  (see Related Documents).
- Variable-pattern default arms — owned by
  [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md).
- golang v1 codegen (`internal/gen/golang`) — its ADT path does not use the if-chain tag
  spelling (it emits `.Kind` switches); unaffected (verified by read, V12).
- Any change to the evaluator's pattern semantics (it is the reference).
- New VM opcodes or builtins (surface stays frozen; simplicity-audit instruction set unchanged).

## Timeline

**Week 1** (~8 hours, 2× the raw ~4h estimate):
- Phase 1: IR node + compiler + unit tests (3h)
- Phase 2: lowering swap + loud boundary + test updates (2.5h)
- Phase 3: emitgo + goldens + sweeps + full make targets (2.5h)

**Total: ~1 day implementation + buffer = ~2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Regression in landed pattern machinery (Bug A.2 guards, M-LOWER-FIX bindings, pattern-arity `OpEq`) — this code has been patched incrementally before | High | Every existing test is a fixture that must pass unmodified except the single `containsTagCheck` spelling change; goldens pin exact outputs |
| Two ADTs sharing tag names + empty `TypeName` → inference ambiguity (same documented caveat as `inferADTFromCases`) | Med | Declaration-order determinism (ailang#1355 contract); the lower pass fills `TypeName` from `cti` wherever the type is known, shrinking the inference set; loud compile error when no ADT contains the tag |
| Merge collision with the two sibling docs (same file, same functions) | Med | Split-of-scope contract in Related Documents; this doc's node is the primitive both siblings' recursive cases compose; whichever sprint lands second builds on it rather than rewriting |
| emitgo ADT-aware `FieldAccess` misfires on genuine records with an `_0`-style field | Low | Resolution keyed on `RecordType` naming a *known ADT* TypeDecl only; records without a matching TypeDecl keep today's name-based path |
| Corpus sweep finds a consumer of the old broken spelling | Low | `grep` for `Field: "Tag"` and `FieldAccess{.*"Tag"}` across `internal/` in the sprint; the only two emit sites are the ones changed (V16) |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates an engine disagreement: same program, different results (evaluator `hi` vs strict VM error), and removes the quiet-default fallback that masked it |
| A2: Replayability | 0 | No trace/replay surface change |
| A3: Effect Legibility | 0 | Pure pattern-matching lowering; no effects touched |
| A4: Explicit Authority | 0 | No capability surface change |
| A5: Bounded Verification | +1 | The strict gate stops failing on a type-correct, ordinary construct; strict-VM coverage extends to the cons-head-ctor idiom |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | `Ctor(x) :: rest` is the natural destructuring an AI writes first; today it demands a human-shaped restructure (match the head separately) |
| A8: Minimal Syntax | +1 | No new syntax — existing, taught, type-checking syntax starts working on all engines |
| A9: Cost Visibility | 0 | No resource semantics change |
| A10: Composability | +1 | The new node is the composable primitive both sibling docs' recursive designs need; unifies the tag-check idiom across switch and if-chain paths |
| A11: Structured Failure | +1 | Replaces a masked-fallback/crash pair with correct results; the interim unsupported shapes (literal/nested args) fail loudly as EvalOnly instead of silently mis-matching |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes an existing engine disagreement; ordinal resolution is declaration-order deterministic (reuses the #1355 contract); no new nondeterminism
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): fixes a machine-facing correctness gap; optimizes nothing for human convenience

## Conflict Surface

### Syntactic/semantic positions touched

- **Expression position in the stmt IR** gains one node type (`ADTTagEq`) — consumed by
  `internal/bytecode/compiler/expr.go` and `internal/gen/emitgo/funcs.go` (both get a new
  `case` in existing type switches; both retain loud defaults).
- **If-chain match condition lowering** (`lowerPatternCond`, `ConstructorPattern` case at
  match.go:395-402 and the `ListPattern` constructor-element check at :426-436): the emitted
  comparison changes from `FieldAccess{"Tag"} == LitString` to the new node.
- **If-chain match binding lowering** (`lowerPatternBindings`, ListPattern/ConstructorPattern
  element case, match.go:497-524): previously-silent skip of Lit/nested args becomes a loud
  panic; Var/Wildcard unchanged.
- **emitgo `FieldAccess` emission** (funcs.go:227-230): gains an ADT resolution keyed on
  `RecordType`.

No parser, lexer, AST, type-checker, effect, or VM-dispatch change.

### What else lives here (and why each is safe)

| Position | Existing valid form | Interaction |
|---|---|---|
| `FieldAccess` on records | `p.x`, row-polymorphic fallback `_record_get` | Unchanged — the new node is *not* a FieldAccess; record field access incl. a user field literally named `Tag` keeps working (V11: `type P = {Tag: string, x: int}` type-checks and `p.Tag` must stay a record access — this is why the compiler special-case was rejected) |
| `FieldAccess` on tuples (`"_0"` spelling) | tuple destructuring in match bindings | Unchanged on the VM path; emitgo ADT-resolution only fires when `RecordType` names a known ADT |
| Switch path (`compileSwitch`, SwitchCase bindings) | all-constructor matches | Untouched; the shared ordinal resolver is a pure refactor of its existing logic |
| `inferADTFromCases` callers | switch with `ADTName == ""` | Refactored, not redefined; signature and declaration-order contract preserved |
| BinOp `OpAnd` composition in conds | length-check && tag-check | Unchanged; `OpAnd` short-circuits (`control_flow.go:161-181`), so `len >= n && tagCheck(head)` stays safe on short lists |
| `LowerMatchExpr`'s 2-arm IfExpr shape | non-tail literal matches | Untouched |
| emitgo default `nil /* unknown expr */` | unknown exprs today | Not extended: both new cases are explicit; unresolvable ADT references raise loud emitter errors |

### Disambiguation strategy

Purely structural, at IR level: the lower pass *knows* it is emitting an ADT tag check (it
is processing a `core.ConstructorPattern`), so it emits the dedicated node; nothing needs to
be inferred from field-name spelling at compile time. The compiler resolves `Tag`→ordinal
only inside `ADTTagEq` compilation via the ADT registry — never by special-casing the string
`"Tag"` in `FieldAccess` (which would be unsound, see V11).

### Programs that MUST still work (regression fixtures)

1. `tests/golden/bytecode/pattern_arity.ail` — the list-pattern golden corpus (all-Var heads;
   exact output rows pinned in `cmd/ailang/run_bytecode_test.go:166`).
2. `examples/pattern_matching_adt.ail` — switch-path constructor matching (Some/None).
3. `examples/runnable/list_pattern_cons.ail` — cons patterns incl. `x :: y :: []` arms.
4. `examples/runnable/recursion_quicksort.ail` — the pattern-arity flagship.
5. `std/json.ail` — `get`/`asString` final wildcard defaults (switch path).
6. `internal/gen/lower/lower_match_test.go` — M-LOWER-FIX and Bug A.2 regression tests,
   with the single deliberate change below.

### What deliberately changes

- Lowered IR/disassembly for constructor tag checks in if-chain matches: `BUILTIN_CALL
  _record_get` → `GET_TAG + LOAD_CONST + EQ`. Any consumer diffing disassembly for these
  shapes (none known in-repo) sees a new form — today's form is a crash.
- `TestLowerMatchStmt_ConsConstructorHead`'s `containsTagCheck` helper: asserts the *same
  property* (a tag check for the constructor exists) against the new node spelling. This is
  the ONE intentional test-text change; all other assertions in that test (binding
  completeness for `t`, `rest`) must pass unmodified.
- `TText("hi") :: r`-style shapes (literal/nested constructor **args** in if-chain
  sub-positions): today loud-crash (strict) / correct (evaluator, non-strict fallback);
  after: loud EvalOnly (strict) / correct (evaluator, non-strict bridge). Strict-mode
  behavior class is unchanged (still an error, now with a precise reason); no shape moves
  from loud to silently-wrong.
- emitgo-v2 output for these shapes changes from non-building Go (`.Tag`, `._0` on ADT
  structs) to building Go (`.Kind`, `.Value0`).

## Related Documents

**Implemented:**
- [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) — landed the cons-head constructor
  *binding* fix (M5 test) whose cond half is this bug: the M5 regression test asserted a tag
  check was emitted — and pinned the broken `FieldAccess{"Tag"}` spelling.
- [m-bytecode-vm.md](../implemented/v0_11_0/m-bytecode-vm.md) — VM architecture, EvalOnly/bridge.

**Planned (siblings — split-of-scope contract):**
- [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) —
  owns nested/literal sub-pattern recursion. **Premise correction:** its Solution Design case
  6 specifies `cond FieldAccess{scrutinee, "Tag"} == Name` and its Architecture states "the
  bytecode compiler needs no changes" — both refuted by this doc's V5/V14 (the construct
  compiles to `_record_get`, which the VM rejects on ADTs). Its recursive constructor cases
  must emit this doc's node instead; nothing else in its design changes.
- [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md) — owns
  variable-pattern default arms. **Premise correction:** it states "The cond side needs no
  change: lowerPatternCond's ConstructorPattern case already emits the tag check
  (match.go:395-402)" (its Component 1) — that is the broken spelling; and its routing gate
  sends constructor arms into if-chains, making match.go:399 reachable for the first time,
  so its fix *requires* this doc's node at that site. Its depth-1 binding contract
  (Var/Wildcard OK, Lit/nested panic loudly) is adopted verbatim here for list-element
  constructor sub-patterns, so the two docs land a consistent interim boundary.
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — landed
  sibling (`len == n`); its `Tail == nil ⟺ OpEq` invariant is untouched.

**Distinctness note** (per the duplicate gate): the neural/simhash doc search found no
document on this topic (Ollama unavailable in this sandbox — the create script's neural leg
returned nothing; siblings were located by targeted `grep -rln "strict-bytecode|FieldAccess"
design_docs/` and read in full). Neither sibling covers the ADT tag-check compilation bug:
the nested doc's scope is *which sub-patterns* are lowered; this doc's scope is *what a
constructor tag check compiles to*. The B-family silent-wrong rows this author reproduced
(r3/r4/r5 below) are already owned by the nested doc's B1-B3 and are NOT re-scoped here.

## References

- [Design Axioms](/docs/references/axioms) — A1/A5/A11 drive the P0
- Evaluator pattern semantics: `internal/eval/eval_patterns.go` (`matchPattern` — reference)
- Lowering: `internal/gen/lower/match.go`; consumers: `internal/bytecode/compiler/{expr,switch}.go`,
  `internal/gen/emitgo/funcs.go`
- Task report: stapledons-godot `sim/markers.ail` workaround (match the head separately),
  sprint R1-AI-FOUNDATION AI.1 — reproduced first-party by the controller (V1-V3)

## Future Work

- The nested-pattern sibling doc's recursive `patternCond`/`patternBindings` generalizes this
  doc's depth-1 if-chain cases; when it lands, the loud EvalOnly boundary here shrinks to
  nothing on its covered positions.
- Runtime tag identity for cross-ADT values (documented switch-path limitation) would remove
  the remaining inference ambiguity for empty-`TypeName` tag checks.
- `--quiet` defaulting to true (`main_run.go:26`) masked this bug's non-strict symptom for
  weeks; consider whether the fallback warning deserves a non-suppressible one-line stderr
  note even under `--quiet` — separate DX decision, out of scope here.

## Verification Log

All claims verified first-party in this session. Binary: `AILANG v0.51.0` (`b99dd25c…-dirty`,
`/usr/local/bin/ailang`); repo HEAD `4460d91b`; this sandbox has **no Go toolchain** — Go
claims are source reads, behavior claims are binary runs. Repro files under `/tmp/repro/`.

| # | Claim | Instrument | Verdict |
|---|-------|-----------|---------|
| V1 | Reported repro: evaluator and non-strict bytecode return `hi` | `ailang run --quiet --relax-modules --entry main r1.ail` → `hi`; `--bytecode` → `hi` | Confirmed |
| V2 | Strict bytecode crashes with `_record_get` on ADT | `--bytecode --strict-bytecode` → `Error: bytecode execution failed: vm: vm: BUILTIN_CALL: _record_get: arg 0 must be record, got ADT (in r1.first at r1.ail:3, ip 19, op BUILTIN_CALL)`, exit 1 | Confirmed |
| V3 | The non-strict "another path" is the **evaluator via silent fallback** (not a working bytecode path) | `ailang run --verbose --relax-modules --bytecode --entry main r1.ail` → `⚠ bytecode path unavailable (vm: … _record_get …); falling back to evaluator` then `hi` | Confirmed |
| V4 | `ailang check` accepts the shape — bug is lowering-only, not type-level | `ailang check --relax-modules r1.ail` → `✓ No errors found!` rc=0 | Confirmed |
| V5 | Mechanism: cond emits `_record_get(head, "Tag")`; bindings emit positional GET_FIELD | `ailang disasm --relax-modules r1.ail` → ip 0019 `BUILTIN_CALL r8, builtin#517` (after `LOAD_CONST #3 "Tag"`), ip 0028 `GET_FIELD r3, r2, r0`; constant pool holds `"Tag"`, `"TText"`, `"TMark"` | Confirmed |
| V6 | Why users don't see the fallback: `--quiet` defaults to true | `cmd/ailang/main_run.go:26` `quietFlag := fs.Bool("quiet", true, …)`; `internal/runner/entrypoint.go:144-159` prints the ⚠ only `if !params.Quiet` | Confirmed |
| V7 | Literal constructor arg in cons head: evaluator rejects the arm; today's strict VM crashes *before* reaching the un-guarded literal (so the tag-fix-alone intermediate would silently mis-match — justification for the loud boundary) | `r2.ail` (`TText("hi") :: r` arm, input `[TText("bye")]`): evaluator `other`; strict VM crashes with the same `_record_get` error | Confirmed |
| V8 | Literal **element** head (`1 :: r`): silent wrong answer on strict VM (sibling B1 — NOT this doc's scope) | `r3.ail`: evaluator `other` vs strict `starts-with-one`, exit 0 | Confirmed |
| V9 | Record literal+constructor fields (sibling B3-class) and tuple ctor+lit elements (sibling A4/B2-class) silent wrong on strict VM — NOT this doc's scope | `r4.ail`: eval `other` vs strict `one-text`; `r5.ail`: eval `other` vs strict `text-zero` | Confirmed |
| V10 | Switch path (all-constructor matches) is healthy on strict VM | `r6.ail` `match t { TText(s) => "t${s}", TMark(s) => "m${s}" }` → `thi` under `--strict-bytecode` | Confirmed |
| V11 | `Tag` is a legal user record field name — compiler cannot special-case field name `"Tag"` | `type P = {Tag: string, x: int}` + `p.Tag` type-checks (`ailang check` rc=0) | Confirmed |
| V12 | golang v1 codegen is unaffected (does not use the if-chain tag spelling) | read `internal/gen/golang/codegen_match*.go` ADT path (`.Kind` switches, `codegen_match.go` family) | Confirmed |
| V13 | emitgo-v2 emits non-building Go for the reported shape | `ailang compile --emit-go-v2 --relax-modules --out gov2 r1.ail` → `BuiltinList_get(toks, int64(0)).Tag == "TText"`, `t := _head_0._0`; ADT structs generated with `Kind`/`Value%d` fields (`internal/gen/emitgo/types.go:72-113`) | Confirmed |
| V14 | Sibling nested-pattern doc's case 6 emits the broken construct and claims "bytecode compiler needs no changes" | read `design_docs/planned/v0_49_1/m-bytecode-nested-pattern-lowering.md` Solution Design case 6 + Architecture | Confirmed (premise refuted by V2/V5) |
| V15 | Sibling var-pattern doc claims "cond side needs no change … already emits the tag check", and its routing gate makes match.go:399 reachable | read `design_docs/planned/v0_51_2/m-vm-var-pattern-default-arm.md` Component 1 bullet 3 + gate design | Confirmed (premise refuted by V2/V5) |
| V16 | NEGATIVE: only two `Field: "Tag"` emit sites exist in the lowering | `grep -rn '"Tag"' internal/gen/lower/` → match.go:399, :432 (non-test) | Confirmed |
| V17 | NEGATIVE: no ADT-tag builtin exists in the builtin table | `internal/bytecode/builtin_names.go` full scan — `_record_get`, `_list_get`, `_len`, `_list_tail` etc.; no tag/ctor-name builtin | Confirmed |
| V18 | NEGATIVE: the in-repo corpus has no constructor-at-cons-head pattern; the golden corpus covers Var heads only | `grep -rnE '[A-Z][A-Za-z]*\(.*\) *::' std/ examples/` → no hits (only tuple/var heads); `tests/golden/bytecode/pattern_arity.ail` read in full | Confirmed |
| V19 | `OpGetTag` returns the tag **ordinal** (int); `OpGetField` handles ADT by index; `OpAnd` short-circuits | read `internal/vm/vm.go:537-545`, `:486-491`, `internal/bytecode/compiler/control_flow.go:161-181` (compileAnd jump-over) | Confirmed |
| V20 | NEGATIVE: no `go` binary in this sandbox (verify-go skipped locally) | `which go` → empty; compile printed `⚠ Skipping Go verification (go binary not in PATH)` | Confirmed |
| V21 | Mixed dispatch: ListPattern arms route to the if-chain; `allConstructorPatterns` requires all arms ctor/var/wild | read `internal/gen/lower/match.go:19-30, 77-88` | Confirmed |
| V22 | Elaboration: `Ctor(x) :: r` arrives as `ListPattern{Elements:[ConstructorPattern], Tail}` | read `internal/elaborate/patterns.go:130-152` (`::` case) | Confirmed |
| V23 | Evaluator `matchPattern` is the recursive reference semantics (7 pattern kinds) | read `internal/eval/eval_patterns.go:105-292` | Confirmed |
| V24 | Regression fixtures exist | `ls examples/pattern_matching_adt.ail examples/runnable/{list_pattern_cons,recursion_quicksort,pattern_sugar,cons_expression}.ail std/json.ail tests/golden/bytecode/pattern_arity.ail` | Confirmed |
| V25 | The M5 test's `containsTagCheck` asserts the old spelling (the ONE intentional test change) | read `internal/gen/lower/lower_match_test.go:234, 254-270` | Confirmed |
| V26 | Ordinal resolution machinery exists in the switch path (tagOrdinal maps built in declaration order; `inferADTFromCases` fallback, adtOrder determinism per ailang#1355) | read `internal/bytecode/compiler/switch.go:48-66, 141-159`, `compiler.go:50-60` | Confirmed |
| V27 | No new error codes introduced (loud panics flow into free-text `EvalReason`; no `MOD/TC/E` code allocation) | design uses only existing channels; `grep` of proposed changes introduces none | Confirmed |

**Frequency note:** no eval pass-rate claim is made. The demand evidence is the external
consumer report (stapledons-godot `sim/markers.ail`, sprint R1-AI-FOUNDATION AI.1, which had
to restructure source around the bug) plus the fact that the construct type-checks and is
the natural idiom for ADT-stream destructuring.

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

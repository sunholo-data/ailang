# M-VM-PURE-BUILTIN-COVERAGE: pure builtins run on the strict VM, and the gap is measured

**Status**: Planned
**Target**: v0.50.2
**Priority**: P1
**Estimated**: 2–3 days
**Dependencies**: None
**Issues**: [#1447](https://github.com/sunholo-data/ailang/issues/1447) (primary), [#1448](https://github.com/sunholo-data/ailang/issues/1448) (bundled, see M4)
**Reporter**: `stapledons_godot` (Stapledon mission iterations 4 and 7, 2026-09-30 / 2026-10-01; M2 sim-core bitwise report 2026-10-01, see §Amendment)
**Quorum**: none of the four attended triggers fire (no freeze items needing a human; no shared machinery overridden — the adapter is additive; no cost/KPI/schema surface; all premises in-repo). Skipped.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a VM↔interpreter divergence class: strict-mode programs no longer differ in *whether they run* by builtin choice. Native ports are pure slice ops; no map iteration introduced |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | +1 | The error stops calling pure builtins "effectful"; purity reported matches the registry's `IsPure` |
| A4: Explicit Authority | 0 | Only `IsPure` builtins are touched; effectful builtins keep their Phase 2E path |
| A5: Bounded Verification | 0 | — |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | A ratchet test makes coverage a measured number instead of something discovered one user report at a time |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | Adapter conversion cost is documented and kept off hot polymorphic list ops (native ports) |
| A10: Composability | +1 | `std/list`, `std/string`, `std/bytes`, … usable from `--strict-bytecode` cores |
| A11: Structured Failure | +1 | Unportable builtins fail at compile with a reason naming *why* (Map value, ADT result, closure arg) |
| A12: System Boundary | 0 | — |

**Net Score: +6** → Proceed. No −1 on A1/A3/A4/A7.

## Problem Statement

`--strict-bytecode` is the Stapledon game's determinism gate: its pure core must run with no evaluator fallback. Every
pure stdlib call whose builtin is missing from the VM's hand-maintained table makes the enclosing function
`EvalOnly`, and strict mode refuses it:

```
$ ailang run --quiet --bytecode --strict-bytecode --entry main --args-json 0 rev.ail
Error: ... TAIL_CALL: std/list.reverse is evaluator-only (compiler: effectful builtin "__list_reverse"
not yet wired (Phase 2E)) but no interop bridge is wired
```

Three defects:

1. **Coverage.** ~121 of 203 `IsPure` registry builtins have no VM implementation (V3). The reporter worked around
   `reverse` with a hand-written `foldl`; the next user will hit `take`, `zip`, `range`, `_bytes_*`, `_regex_*`…
2. **Lying message.** `compileBuiltinCall` (`internal/bytecode/compiler/builtins.go:224`) labels every unlisted builtin
   "effectful … Phase 2E", including pure ones (V2).
3. **No measurement, no lockstep check.** Nothing compares the VM tables to the registry, so the gap grows silently
   with each new pure builtin. The comments in `internal/vm/builtins.go:30,45` cite a `validateBuiltinTables` that
   does not exist (V5) — the compiler/VM table lockstep is unenforced.

The gap is not limited to stdlib calls: the **bitwise Int operators `^ & ~ << >>` themselves** lower to pure builtins
(`bitwiseXor_Int`, …) missing from the table, so `main(n) = n ^ 3` fails `--strict-bytecode` while `+ * / %` compile
fine (V12–V17, §Amendment). Stapledon's M2 sim core (SplitMix64/PCG PRNG in the strict-pure `sim/core.ail`) is
blocked on exactly these operators.

**Plus #1448 (M4):** whole-number float literals evaluate as `IntValue` inside `test` blocks — same reporter, same
"engine paths disagree" family, small. Bundled to share one sprint.

## Goals

- Every `IsPure` builtin is in exactly one bucket, enforced by a test: **native** (VM `BuiltinTable`/`HOFBuiltinTable`),
  **adapted** (generic registry-backed adapter), or **unported** (explicit allowlist entry with a machine-checked reason).
- `std/list.reverse/take/drop/zip/range/contains/head` and `std/string` `repeat`/`reverse` run under `--strict-bytecode`
  with output identical to the interpreter.
- The compile error distinguishes pure-unported from effectful.
- `ailang test` evaluates `4.0` as a float in test blocks (#1448).

## High-Impact Decisions

| Decision | Choice | Why | Change cost |
|---|---|---|---|
| D1: How to cover ~121 builtins | **Two-tier**: native ports for polymorphic structural ops; one generic registry adapter for monomorphic, convertible signatures | Native-for-all is ~121 hand ports (the pattern that created the gap). Adapter-for-all breaks on type variables: `reverse([Some(1)])` has ADT elements, and `bytecodeValueToEval` rejects ADTs and closures (V7). It also turns O(1) `head` into O(n) via whole-list conversion | Low — adapter is additive |
| D2: Adapter eligibility | Decided **statically from the builtin's registered type**: no type variables, no `Map`, no function types, no ADT (Option/Result/user) in params or result | A static rule means strict-mode failure stays a *compile-time* fact, never a runtime conversion error mid-program (avoids #506-class mid-run fallback) | Low |
| D3: Is the adapter "the evaluator"? | **No** — it calls the builtin's Go `Impl`, not the tree-walker. Strict mode allows it | Precedent: `internal/vm/builtins_xml.go` already converts to `eval.Value` and calls Go impls under strict mode (V6) | — |
| D4: Converter location | Move `bytecodeValueToEval`/`evalValueToBytecode` from `internal/runner/bridge.go` into `internal/vm` (exported), runner calls them | `runner` imports `vm`, not the reverse; the adapter lives in `vm`. One implementation, not two (seam rule) | Low |

### Design Freeze

- [x] D1 two-tier
- [x] D2 static eligibility
- [x] D3 adapter allowed under strict
- [x] D4 converters move to vm

All agent-resolvable; no human ratification required.

## Solution Design

### Overview

```
compileBuiltinCall(name)
  ├─ in BuiltinTable           → OpBuiltinCall        (native)
  ├─ in HOFBuiltinTable        → OpBuiltinCallHOF     (native, closures)
  ├─ registry IsPure && adaptable(type) → OpBuiltinCallAdapted(idx)   ← NEW
  ├─ registry IsPure           → compile error "pure builtin %q has no VM implementation: <reason>"   ← honest
  └─ otherwise                 → compile error "effectful builtin %q not yet wired (Phase 2E)"  (unchanged)
```

### Architecture

**M1 — Ratchet + honest message + lockstep.** New test (`internal/vm/builtin_coverage_test.go`, package `vm_test`)
enumerates `builtins` registry entries with `IsPure`, maps each to its lowered name (`"_" + registry name`, V4), and
asserts each is native, adapted, or in `unportedPure` (a `map[string]string` name→reason in a dedicated file). Fails
if: a pure builtin is in no bucket; an allowlisted builtin became native/adapted (ratchet — allowlist must shrink);
compiler and VM table lengths differ (the missing `validateBuiltinTables`, implemented for real and called from
`vm.New`). The compiler error path consults the registry (`builtins` is already a vm dependency, V6; check the
compiler package does not import `builtins` today, and `builtins` imports neither `bytecode` nor `vm`, so a direct
import is cycle-free (V10); prefer that over an injected lookup unless it trips a layering gate).

**M2 — Generic adapter.** `vm.AdaptedBuiltinTable` built once at init from the registry: every pure builtin passing
`adaptable(spec.Type())` (D2). New opcode `OpBuiltinCallAdapted A B C` (dst, adapted index, argc): convert args with
`BytecodeToEval`, call `spec.Impl(nil, args)`, convert result with `EvalToBytecode`. Disasm entry for the new opcode.
Order is deterministic: sorted by name (no map iteration — A1, cf. #1355).

**M3 — Native ports** (polymorphic, can carry any element): `__list_reverse`, `__list_take`, `__list_drop`,
`__list_zip`, `__list_range`, `__list_contains` (uses existing `runtimeEq`), `__list_head`, `__list_extract`,
`__str_repeat`, `__string_reverse`. Semantics copied exactly from the evaluator impls (e.g. `take n≤0 → []`,
`n > len → whole list`; `drop n≤0 → copy`, `n ≥ len → []`, `internal/builtins/list.go:625-700`). Each gets a
table-driven parity test: run the registry `Impl` and the VM func on the same inputs, compare.

**M4 — #1448 float literals in test blocks.** See §M4 below.

### Files to Modify/Create

- `internal/bytecode/compiler/builtins.go` — adapted-call branch, honest error (~+40)
- `internal/bytecode/opcode.go` — `OpBuiltinCallAdapted` (~+5); `internal/bytecode/disasm.go` (~+5)
- `internal/vm/builtins_adapted.go` — NEW: eligibility, table, dispatch (~150)
- `internal/vm/convert.go` — NEW: converters moved from `internal/runner/bridge.go` (~180 moved)
- `internal/runner/bridge.go` — call `vm.BytecodeToEval`/`vm.EvalToBytecode` (−170)
- `internal/vm/builtins_list.go` — native list ports (~+150)
- `internal/vm/builtins_string.go` — `repeat`, `reverse` (~+40)
- `internal/vm/builtins.go` — table entries + real `validateBuiltinTables` (~+30)
- `internal/vm/vm.go` — dispatch for the new opcode (~+20)
- `internal/vm/builtin_coverage_test.go`, `internal/vm/builtin_unported.go` — NEW ratchet (~200)
- `internal/ast/ast_expr.go` — `FormatFloat` + `Literal.String()` FloatLit branch (M4, ~+20)
- `internal/format/literal.go` — `formatFloat` delegates to `ast.FormatFloat` (M4)
- `internal/testing/executor_helpers.go` — drop false comment, error on non-finite floats (M4)
- `examples/vm_strict_pure_builtins.ail` — NEW, runs under `--strict-bytecode`
- `changelogs/unreleased/2026-10-01-vm-pure-builtin-coverage.md`

## Conflict Surface

1. **Positions extended:** `compileBuiltinCall`'s fall-through (today: "effectful … Phase 2E" → `EvalOnly`). New
   branches sit *before* it and only for registry-`IsPure` names.
2. **Other constructs in that position:** lower-pass dict fallbacks (`_dict_*`, `_Class_Type_method`) — handled
   earlier by `isLowerPassDictFallback` and stay a hard compile error; they are not registry builtins, so the
   registry lookup cannot capture them. Effectful builtins (`__io_println`) — `IsPure=false`, unchanged path.
   HOF builtins — matched by `hofBuiltinIndex` before the adapter; the adapter's D2 rule also excludes function types.
3. **Disambiguation:** by registry `IsPure` + static type shape, both known at compile time.
4. **Must still work:** `--bytecode` non-strict hybrid runs (bridge keeps working with moved converters);
   `tests/golden/codegen/` corpus; `examples/` under `make verify-examples`; XML builtins (they keep their own
   converters); the #1354/#1355 regression tests (record field slots — `EvalToBytecode` record order goes through
   `NewRecord`, which sorts).
5. **Deliberate changes:** functions that were `EvalOnly` solely because of a newly native/adapted builtin now
   compile to bytecode. In non-strict `--bytecode` they move from the evaluator bridge to the VM — output must be
   identical (parity tests), and this is the point.

## M4: #1448 — whole-number float literals in test blocks

**Root cause (V11):** `ailang test` folds a test-block body, **prints it back to AILANG source and re-parses it**
(`internal/testing/executor.go:233-239`, `PrintAILANGSource(folded)`). In `PrintAILANGSource`'s `*ast.Literal` case
(`internal/testing/executor_helpers.go:316`) non-string literals fall through to `e.String()` with the comment "bool,
int, float all have correct String()". That comment is false: `ast.Literal.String()` (`internal/ast/ast_expr.go:109`)
is `fmt.Sprintf("%v", l.Value)`, so `float64(4.0)` prints `4` → re-parsed as `IntLit`. `2.25` survives, which is why
only whole-number floats fail. Inline `tests [(4.0, 2.0)]` and `ailang run` never round-trip through text.

A second path has the same defect: `Executor.EvaluateExpression` (`internal/testing/executor.go:86,93`, called from
`runner.go:337,699`) builds source with `fmt.Sprintf("%v", …)`, which also reaches `Literal.String()`.

**Fix (systemic, one implementation):** make `ast.Literal.String()` print a `FloatLit` canonically — move the
canonical float spelling (`strconv.FormatFloat(v,'g',-1,64)` + `.0` when no `.`/`e`/`E`/`n`/`N`; today
`internal/format/literal.go:120 formatFloat`) into `internal/ast` as an exported `FormatFloat`, and have
`format.formatFloat` delegate to it (`format` imports `ast`, not the reverse). This fixes both test paths and any
other `String()`-to-source consumer at once, and leaves one float spelling in the codebase (seam rule). Remove the
false comment at `executor_helpers.go:316`. Non-finite values (`+Inf`/`NaN` from folding) are not valid source:
`PrintAILANGSource` must return an error for them, not emit them (fail loudly).

**Conflict check:** any golden/debug output that prints a whole-number float literal via `Literal.String()` changes
from `4` to `4.0`. M4 starts with `git grep` for golden files containing AST dumps and runs `make test` before/after;
changed goldens are inspected and regenerated only where the new spelling is the correct one.

**Tests:** round-trip `4.0`, `0.0`, `-4.0`, `100.0`, `1e20`, `2.25` through `PrintAILANGSource` → parse → same `FloatLit`;
`#1448` repro (`probe.ail`) as an `ailang test` integration test (3/3 pass); mutation-check by reverting the `.0` append.

**Out of scope, noted:** the typechecker lets an int literal flow into a `float` slot so the defect surfaced at
runtime as `_math_sqrt: expected FloatValue` rather than as a type error. Separate concern; record as follow-up.

## Examples

```ailang
module vm_strict_pure_builtins
import std/list (reverse, take, drop, zip, range, contains)
export pure func main(n: int) -> [int] =
  take(2, reverse(drop(1, range(0, n))))
```
`ailang run --bytecode --strict-bytecode --entry main --args-json 6 examples/vm_strict_pure_builtins.ail` → `[5, 4]`,
identical to the interpreter.

## Success Criteria

- [ ] `rev.ail` (issue #1447 repro) passes `--strict-bytecode` → `[3, 2, 1]`
- [ ] Ratchet test green; unported allowlist size reported in test log and changelog (expected ≪ 121)
- [ ] Error for a pure-unported builtin reads "pure builtin … has no VM implementation: <reason>"
- [ ] Every native port has an eval-vs-VM parity table test
- [ ] Table lockstep enforced at `vm.New`
- [ ] #1448 repro: `ailang test probe.ail` 3/3 pass
- [ ] `make test`, `make lint`, `make verify-examples`, `make check-file-sizes` green

## Testing Strategy

- Unit: parity tables per native port (empty list, n<0, n>len, ADT elements, nested lists).
- Adapter: one test per eligibility rule (accept monomorphic bytes/string; reject type-var, Map, ADT, closure).
- Mutation check (per memory: mutate your own tests): break one native port's edge case; parity test must fail.
- End-to-end: issue repros for #1447 and #1448 as CLI-level tests.

## Deferred Decisions

- Exact reason strings in the allowlist — agent's choice, must be one of a small fixed set (`map-value`, `adt-result`,
  `closure-arg`, `polymorphic-needs-native-port`).
- Whether `_list_takeMap`/`_list_takeFlatMap` go native-HOF in this sprint or stay allowlisted.

## Non-Goals

- A VM `TagMap` (all `_map_*` stay allowlisted with reason `map-value`).
- Effectful builtins / Phase 2E (#506).
- #1419 (NaN ordering) and #1420 (nested cons patterns) — separate design docs already merged.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Adapter result shape differs from native expectations (record field order) | `NewRecord` sorts; parity tests on record-returning adapted builtins |
| Builtin `Impl` that dereferences `EffContext` despite `IsPure` | Adapter passes `nil`; a test calls every adapted builtin once with typed sample args, or recovers panics into a VM error |
| Moving converters breaks the hybrid bridge | Pure move + rename; existing bridge tests are the control |

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | `reverse` fails strict on HEAD | `ailang run --bytecode --strict-bytecode` on v0.50.0-6-g021c46907 → `TAIL_CALL … effectful builtin "__list_reverse" not yet wired` |
| V2 | `_list_reverse` is registered pure | `internal/builtins/list.go:562-565` `IsPure: true` |
| V3 | ~121/203 pure builtins unwired | `ailang builtins list --json` `is_pure` vs names in compiler `BuiltinTable`+`HOFBuiltinTable` (`__`→`_` normalized) → 203 / 94 / 121 |
| V4 | Lowered name = `"_" + registry name` | `internal/gen/lower/expr.go:232` `stmt.BuiltinCall{Name: "_" + vg.Ref.Name}` |
| V5 | `validateBuiltinTables` does not exist | `git grep -n "func validateBuiltinTables"` → empty; only comment references in `internal/vm/builtins.go:30,45` |
| V6 | vm already depends on builtins and eval | `go list -deps ./internal/vm` includes `internal/builtins`, `internal/eval`; `internal/vm/builtins_xml.go:52` converts `eval.Value` |
| V7 | Bridge converters reject ADT and closures | `internal/runner/bridge.go:183-189` |
| V8 | No existing issue/design doc for the pure-builtin gap | `gh issue list --search` (only #1447 now); planned/ search found no match ≥0.45 |
| V10 | compiler→builtins import is cycle-free | `go list -deps ./internal/bytecode/compiler` has no `builtins`/`eval`/`vm`; `go list -deps ./internal/builtins` has no `bytecode`/`vm` |
| V11 | #1448 cause = text round-trip of float literal | Read `internal/testing/executor_helpers.go:316` (`return e.String() // bool, int, float…`), `internal/ast/ast_expr.go:109` (`%v`), `internal/format/literal.go:120` (`formatFloat` already canonical) |
| V9 | #1448 reproduces on HEAD | `ailang test probe.ail` → 2/3 fail `_math_sqrt: expected FloatValue … got *eval.IntValue` |
| V12 | Bitwise operator repro on HEAD | `main(n) = n ^ 3` with `--args-json 6`: interpreter → `5`, exit 0; `--bytecode --strict-bytecode` → `Error: … effectful builtin "_bitwiseXor_Int" not yet wired (Phase 2E)` (v0.50.1, commit 021c469; reporter saw same on pinned v0.47.2 and v0.50.0-6-g021c46907) |
| V13 | `|` is reserved; `bitwiseOr` is a function; in-repo example fails strict | `internal/lexer/token.go:422` (`BITWISE_OR \| reserved, not an operator`); `std/math.ail:131` (`bitwiseOr(a,b) = ~(~a & ~b)`); `ailang run --bytecode --strict-bytecode examples/bitwise_or.ail` → `Error: … _bitwiseAnd_Int not yet wired (Phase 2E)` |
| V14 | All six bitwise builtins registered pure, monomorphic int | `ailang builtins list --json` → `bitwiseAnd_Int`/`bitwiseXor_Int`/`bitwiseOr_Int` `(int,int)->int`, `bitwiseNot_Int` `int->int`, `shiftLeft_Int`/`shiftRight_Int` `(int,int)->int`, all `is_pure: true` (registered `internal/builtins/math_bitwise.go:13-27`) |
| V15 | Zero bitwise/shift names in either BuiltinTable | `grep -c "_bitwise\|_shift" internal/bytecode/compiler/builtins.go internal/vm/builtins.go` → `0` / `0` |
| V16 | SplitMix64 fixture: evaluator matches u64 reference; strict fails | `ailang check` clean; interpreter `main(0)` → `5807750865143411619` == Python u64 reference (signed wrap arithmetic); strict → `Error: … _bitwiseXor_Int not yet wired`; `lshr64`/`next`/fixture text in §Amendment |
| V17 | Arithmetic-only PRNG passes strict (gap is precisely bitwise) | LCG `s = s*6364136223846793005 + 1442695040888963407; (s/65536)%100` with `--bytecode --strict-bytecode` → `42`, exit 0 |

## Amendment (2026-10-01): Stapledon M2 — bitwise Int operators are the same gap

A follow-on report from the same mission (M2 stage: strict-pure sim core `sim/core.ail` needs a SplitMix64/PCG PRNG)
confirmed the bitwise Int operators are six of the ~121 unwired pure builtins — verified on HEAD (v0.50.1,
021c469): `main(n) = n ^ 3` evaluates to 5 under the interpreter but strict mode fails with
`compiler: effectful builtin "_bitwiseXor_Int" not yet wired (Phase 2E)` (V12). Same for `& ~ << >>`; `|` is a
reserved token (ADT alternatives), so `bitwiseOr` is a `std/math` function built from `~` and `&`, and the existing
`examples/bitwise_or.ail` — which documents the whole bitwise surface — fails strict identically (V13).

**Routing under the frozen decisions (no freeze change):** all six builtins (`bitwiseAnd_Int`, `bitwiseXor_Int`,
`bitwiseOr_Int`, `bitwiseNot_Int`, `shiftLeft_Int`, `shiftRight_Int`) are registered `is_pure` with monomorphic
`(int, int) -> int` / `int -> int` signatures (V14) and appear in neither `BuiltinTable` (V15), so per D1/D2 they
land in the **M2 adapted bucket** — no new mechanism is required for this report. One sprint-planner judgment is
recorded: these six are hot-loop monomorphic Int ops (a PRNG issues dozens per draw) with the existing native
`_mod_Int` / `__math_abs_Int` precedent (~40 LOC in `internal/vm/builtins_math.go`); native ports may be pulled
forward from M3-style work if cheap, but the adapter already unblocks strict mode. Semantics must copy the
evaluator exactly: `shiftLeft_Int`/`shiftRight_Int` raise `RT_SHIFT` ("negative shift amount") for negative shift
amounts and `>>` is arithmetic (`internal/builtins/math_bitwise.go:33-57`).

**Regression fixture (reference-validated):** a faithful SplitMix64 in AILANG (logical shift expressed as
`(x >> n) & ((1 << (64 - n)) - 1)`; u64 constants above int63 spelled as their signed wrap) type-checks, runs on the
evaluator, and matches a Python u64 reference — `main(0) = 5807750865143411619` (V16). Under strict mode it fails
at `_bitwiseXor_Int`. Once wired, evaluator and strict must agree on this value. The reporter's stopgap (a 64-bit
LCG using only `* + / %`) passes strict on HEAD (V17), which proves the strict gap is precisely the bitwise set.

```ailang
module splitmix
-- SplitMix64 PRNG (Stapledon M2 sim-core need), all bitwise Int ops.
-- Logical right shift via arithmetic shift + mask: lshr(x,n) = (x>>n) & ((1<<(64-n))-1)
-- u64 constants above int63 are spelled as their signed wrap.

export pure func lshr64(x: int, n: int) -> int =
  (x >> n) & ((1 << (64 - n)) - 1)

export pure func next(state: int) -> (int, int) =
  let s1 = state + (-7046029254386353131);
  let z1 = (s1 ^ lshr64(s1, 30)) * (-4658895280553007687);
  let z2 = (z1 ^ lshr64(z1, 27)) * (-7723592293110705685);
  let out = z2 ^ lshr64(z2, 31);
  (out, s1)

export pure func main(seed: int) -> int =
  match next(seed)
    { (a, s1) => match next(s1) { (b, s2) => a + b } }
```
(Notes for the executor: `let` chains need `;` separators — newline-only chains mis-parse; u64 constants ≥2^63
must be spelled as signed wraps or the lexer rejects them as integers.)

## Related Documents

- `design_docs/planned/v0_47_2/m-bytecode-getfield-slot-resolution.md` (#1354), `v0_47_2/m-vm-determinism.md` (#1355)
- `design_docs/planned/v0_49_1/m-bytecode-nested-pattern-lowering.md` (#1420), `v0_49_1/m-float-ord-one-semantics.md` (#1419)
- `design_docs/implemented/v0_10_1/m-bitwise-operators.md` — original bitwise operator semantics (v0.10.1)

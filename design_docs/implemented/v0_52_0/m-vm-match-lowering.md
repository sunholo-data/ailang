# M-VM-MATCH-LOWERING — one recursive match lowering for the bytecode VM (nested patterns, constructor sub-patterns, guards over bindings, catch-all arms)

**Status**: Implemented (2026-10-02, v0.51.1)
**Issues**: #1420, #1505, #1517 (nested cons evaluator-only), #1473 (variable catch-all arm), #1503 (constructor sub-pattern VM crash), plus the guard-ordering family
**Supersedes** (moved alongside this doc, each with a "Superseded by" status line):
- [m-bytecode-nested-pattern-lowering.md](m-bytecode-nested-pattern-lowering.md) — nested patterns + literal sub-patterns (A/B families)
- [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md) — variable catch-all arms (unbound default, arm order, guarded catch-all)
- [m-vm-ifchain-tag-guard-lowering.md](m-vm-ifchain-tag-guard-lowering.md) — ADT tag checks in if-chain positions (C family) + guards before bindings (G family)
- [m-vm-adt-tag-check-lowering.md](m-vm-adt-tag-check-lowering.md) — ADT tag checks spelled as `_record_get` on an ADT

## Problem

Pattern matching had two implementations that disagreed:

1. `internal/eval/eval_patterns.go:matchPattern` — recursive, every pattern kind at any depth;
   the arm loop is *match → bind → guard → next arm on a false guard*. This is the reference.
2. `internal/gen/lower/match.go` — the Statement IR lowering feeding the bytecode compiler (and
   emitgo). It handled a flat subset and silently dropped the rest.

The four planned docs each described one slice of the gap. Their symptom tables collapse into five
families, all reproduced first-party on this branch's base (`078463551`) with
`ailang run` vs `ailang run --bytecode --strict-bytecode`:

| Family | Example | Strict VM before |
|---|---|---|
| A nested patterns | `a :: b :: _`, `[x :: y, _]`, `Some(Some(x))`, `{user: {email: e}}` | evaluator-only: `unbound variable` |
| B literal sub-patterns | `[1.0, 2.0]`, `(1, y)`, `{name: "alice"}`, `x :: y :: []`, `"a" :: rest` | **silent wrong arm, exit 0** |
| C constructor in a non-switch position | `Some(x) :: rest`, `[Some(x)]`, `TText(t) :: r` | VM crash: `_record_get: arg 0 must be record, got ADT` |
| G guard reads the arm's bindings | `x :: rest if x > 5`, `[a, b] if a > b`, `other if other == "b"` | evaluator-only: `unbound variable` |
| V variable catch-all | `other => f(other)`; catch-all before a constructor arm; guarded catch-all; all-variable match | unbound / **silent wrong arm** / `unknown ADT ""` |

The planned docs also contradicted each other: the nested-pattern doc specified tag checks as
`FieldAccess{"Tag"}` (the construct that crashes, family C); the if-chain doc proposed a new
`_adt_tag` name builtin while the tag-check doc froze "ordinal via `OpGetTag`, no new builtin";
the nested doc kept "guards AND onto the arm cond, as today" (family G). Fixing them as four
patches would have re-landed each other's bugs.

## Root causes

- **Bindings** — `lowerPatternBindings` bound only `VarPattern` children (one level of constructor
  args inside a list); nested list tails, tuple/record sub-patterns and nested constructor args were
  skipped → unbound variables (A).
- **Conditions** — `lowerPatternCond` returned `true` for tuples and records, ignored literal
  elements and nested tails (B).
- **Tag checks** — written `FieldAccess{Record: v, Field: "Tag"} == "Ctor"`. With no static field
  index this compiles to the by-name `_record_get` builtin, which rejects ADT values (C).
- **Guards** — ANDed into the arm condition while the arm's bindings were `VarDecl`s in the arm
  body, so the guard referenced names not yet declared (G). Guards whose ANF form needs statements
  (`let $tmp = f(x) in $tmp == …`) were lowered through `lowerExpr`, which has no place for them.
- **Switch routing** — `allConstructorPatterns` accepted variable/wildcard arms in any position
  and demoted them to the switch `Default` without binding them and without their guard; literal
  args used a `_lit_i` guard whose failure branch was the default body, wrong whenever a later arm
  shared the tag (`Some(0) => …, Some(n) => …`) (V, and the `_concat_String … got Unit` crash).

## Design

### 1. One recursive pattern pair (`internal/gen/lower/match_pattern.go`)

`patternCond(s, type, pat)` and `patternBindings(s, pat)` walk the same access paths:

| sub-pattern position | access expression |
|---|---|
| constructor arg j | `FieldAccess{s, "_j"}` → positional `GET_FIELD` (works on ADTs) |
| tuple element i | `FieldAccess{s, "_i"}` → positional `GET_FIELD` |
| record field k | `FieldAccess{s, k}` (static index or by-name lookup on a record) |
| list element i | `_list_get(s, i)` |
| list tail after n | `_list_tail(s, n)` |

- Var/Wildcard → no condition; Var binds `s`.
- Literal → `s == lit` at any depth.
- Constructor → `ADTTagEq{s, Type, Tag}` AND each argument's condition.
- Tuple / Record → AND of element / field conditions (fields sorted). Arity and field presence are
  guaranteed by the type checker, as in the evaluator — no stricter check is added (that would
  create a new divergence).
- List → `len == n` (closed) or `len >= n` (with tail; omitted when n = 0), AND each element's
  condition, AND the tail's condition.

The condition is one `&&` chain; the compiler's `&&` short-circuits, and the chain is ordered so
every access is guarded (length before `_list_get`, tag before `GET_FIELD`). Bindings are only
emitted inside the branch the condition guards, so they never trap.

Cons chains are normalised first: `a :: b :: rest` elaborates to
`ListPattern{[a], Tail: ListPattern{[b], Tail: rest}}` and is flattened to `[a, b, ...rest]`;
`x :: y :: []` becomes the closed `[x, y]`. Same match set, one length check, no nested
`_list_tail` copies.

The type is threaded only to name the ADT in a tag check (list element, tuple element and record
field types descend structurally; constructor argument types are not known to the lowering and pass
`nil`).

### 2. `stmt.ADTTagEq` — the missing expression (`internal/gen/stmt`, `internal/bytecode/compiler/adt_tag.go`)

```go
type ADTTagEq struct { Value Expr; TypeName string; Tag string }
```

Compiled to exactly the switch path's per-case test: `GET_TAG`, `LOAD_CONST ordinal`, `EQ`. The
ordinal comes from the declaration-order tag table that `MAKE_ADT` and `compileSwitch` use. An
empty or unregistered `TypeName` falls back to `inferADTFromTags` — `inferADTFromCases`' loop
factored out, so the switch and the tag check share one resolution rule (declaration order, never
map order, ailang#1355). An unknown tag is a compile error → the function is tagged EvalOnly, never
a guessed ordinal. **No VM opcode or builtin is added** (the `_adt_tag` name-builtin proposal of
m-vm-ifchain-tag-guard-lowering is not taken: it adds runtime surface, and the ordinal route is
already what the VM dispatches on).

The node is wired through every Statement IR walker: compiler `compileExpr` and the lambda
free-variable visitor, lower's `QualifyFuncRefs` rewrite and `walkExpr`, and emitgo (which writes
the same `.Kind == <Type>Kind<Tag>` comparison its switch cases use).

### 3. The if-chain mirrors the evaluator's arm loop (`internal/gen/lower/match_ifchain.go`)

Each arm becomes `if <cond> { <bindings>; <body> } else { <later arms> }`. An unguarded
irrefutable arm ends the chain (later arms are dead). A guard that needs no binding and no
statements is ANDed into the condition.

A guard on a pattern that binds variables (or a guard whose ANF form needs statements) must run
*after* the bindings and fall through to the later arms when false. Duplicating the later arms
into both else-branches (the m-vm-ifchain-tag-guard-lowering proposal) grows as 2^k in the number
of guarded arms, so a per-match flag carries the fall-through instead:

```
var $matchedN = false
if <cond> { <bindings>; <guard stmts>; if <guard> { $matchedN = true; <body> } }
if !$matchedN { <later arms> }
```

In tail position every body returns, so the flag is only read on the fall-through path; in value
position `flattenValue` rewrites the returns into assignments and the flag is what stops later arms
from running. A non-atomic scrutinee is bound once to `$scrutN`.

### 4. The switch fast path, only where it is exact (`switchEligible`, `internal/gen/lower/match.go`)

A `SwitchStmt` dispatches on the tag alone and runs `Default` only when no case tag matched. It is
used only when that equals first-match-wins:

- every arm but the last is a constructor pattern; tags are distinct;
- constructor arguments are variables or wildcards;
- a variable/wildcard arm, if any, is the final arm and unguarded — it binds the scrutinee at the
  top of `Default`;
- at least one constructor arm exists.

Guards on constructor arms stay on the switch path: they run after the case bindings (their ANF
statements inside the case), and a false guard can only fall to the catch-all. Everything else —
literal or nested arguments, repeated tags, a catch-all elsewhere, a guarded catch-all, an
all-variable match — goes to the if-chain. The Bug A.2 `_lit_i` machinery is deleted: a literal
argument is an ordinary recursive condition now.

### Decisions that resolve the four docs' conflicts

| Question | Docs said | Chosen |
|---|---|---|
| How to test an ADT tag in an expression | `FieldAccess{"Tag"}` (nested doc) / `_adt_tag` builtin (if-chain doc) / IR node + `OpGetTag` (tag-check doc) | IR node `ADTTagEq` + `OpGetTag` ordinal — no VM/builtin change |
| Guard over bindings | AND into cond (nested doc) / nested-if with duplicated rest (if-chain doc) | bind → guard → per-match flag fall-through (linear size) |
| Catch-all before a constructor arm | gate the switch (var-arm doc) | same gate, generalised in `switchEligible` |
| Literal/nested constructor args in if-chain | loud EvalOnly panic as an interim (tag-check, var-arm docs) | not needed: the recursion covers them; an unknown pattern type still panics loudly |
| Nested-cons length check | naive per-level or strength-reduced (deferred) | normalise the chain → one check |
| Hoist sub-scrutinee temps | deferred | not hoisted (pure, O(1) `GET_FIELD`/`_list_get`); scrutinee itself hoisted if non-atomic |

## Verification

Binaries: `ailang-base` built from `078463551` (this branch's base), `ailang-vm` built from the
fix. Every row is a zero-argument entry in `tests/golden/bytecode/nested_patterns.ail` returning
several calls joined by `/` (matching and near-miss inputs). "eval" is `ailang run`; the strict
columns are `ailang run --bytecode --strict-bytecode`. After the fix all 42 rows equal the
evaluator; the same table is the CI gate `TestCLI_RunBytecode_MatchLoweringParity`
(`cmd/ailang/run_bytecode_match_test.go`), which asserts the hand-computed value on both engines.

| entry | eval | strict before | strict after |
|---|---|---|---|
| p_cons2 | `2 / 0 / 0` | `EVALONLY: unbound variable "b"` | `2 / 0 / 0` |
| p_cons2_wild | `5 / 0` | `EVALONLY: unbound variable "x"` | `5 / 0` |
| p_cons3 | `7 / 0 / 6` | `EVALONLY: unbound variable "b"` | `7 / 0 / 6` |
| p_cons_closed_tail | `pair / other / other` | `pair / pair / pair` | `pair / other / other` |
| p_cons_in_list | `1 / 0 / 0` | `EVALONLY: unbound variable "x"` | `1 / 0 / 0` |
| p_ctor_nested | `7 / 1 / 0` | `EVALONLY: unbound variable "x"` | `7 / 1 / 0` |
| p_cons_in_tuple | `0 / 14` | `EVALONLY: unbound variable "x"` | `0 / 14` |
| p_lit_list | `0.0 / 1.0` | `1.0 / 1.0` | `0.0 / 1.0` |
| p_lit_tuple | `0 / 5` | `5 / 5` | `0 / 5` |
| p_lit_record | `other / alice` | `alice / alice` | `other / alice` |
| p_lit_prefix | `no / ok1` | `ok1 / ok1` | `no / ok1` |
| p_lit_cons_head | `A1 / x / e` | `A1 / A0 / e` | `A1 / x / e` |
| p_record_nested | `baby a@x / b@x` | `EVALONLY: unbound variable "e"` | `baby a@x / b@x` |
| p_ctor_cons_head | `hi / mx / empty` | `VM CRASH: _record_get on ADT` | `hi / mx / empty` |
| p_opt_cons_head | `ok5:2 / none / empty` | `VM CRASH: _record_get on ADT` | `ok5:2 / none / empty` |
| p_opt_list_elem | `one4 / other / other` | `VM CRASH: _record_get on ADT` | `one4 / other / other` |
| p_ctor_in_tuple | `7 / 3` | `EVALONLY: unbound variable "n"` | `7 / 3` |
| p_lit_ctor_cons_head | `greeting / text bye / other` | `VM CRASH: _record_get on ADT` | `greeting / text bye / other` |
| p_list_in_ctor | `3 / 0 / -1` | `EVALONLY: unbound variable "x"` | `3 / 0 / -1` |
| p_deep_mix | `6 / 1 / 0` | `EVALONLY: unbound variable "a"` | `6 / 1 / 0` |
| p_guard_list | `desc / other / other` | `EVALONLY: unbound variable "a"` | `desc / other / other` |
| p_guard_cons | `big / small / empty` | `EVALONLY: unbound variable "x"` | `big / small / empty` |
| p_guard_var | `A / B / Z` | `EVALONLY: unbound variable "other"` | `A / B / Z` |
| p_guard_record | `big / small` | `EVALONLY: unbound variable "v"` | `big / small` |
| p_guard_tuple | `lt / ge` | `EVALONLY: unbound variable "a"` | `lt / ge` |
| p_guard_nested_cons | `desc / asc / short` | `EVALONLY: unbound variable "a"` | `desc / asc / short` |
| p_guard_ctor_dup | `big / small2 / none` | `VM CRASH: _concat_String: arg 1 must be string, got Unit ` | `big / small2 / none` |
| p_guard_ctor_default | `big / other / other` | `big / other / other` | `big / other / other` |
| p_guard_chain | `one / two / rest / same / huge / rest` | `EVALONLY: unbound variable "a"` | `one / two / rest / same / huge / rest` |
| p_guard_value_pos | `50 / -10 / 0` | `EVALONLY: unbound variable "a"` | `50 / -10 / 0` |
| p_var_default | `p / committed / idle` | `EVALONLY: unbound variable "other"` | `p / committed / idle` |
| p_var_before_ctor | `o / c / o` | `idle / c / o` | `o / c / o` |
| p_var_guarded | `a / c / z` | `z / c / z` | `a / c / z` |
| p_all_var | `42` | `EVALONLY: unknown ADT "" in switch` | `42` |
| p_str_var | `one / hello!` | `one / hello!` | `one / hello!` |
| p_ctor_lit | `true / false / false` | `true / false / false` | `true / false / false` |
| p_ctor_lit_dup | `zero / n3 / none` | `EVALONLY: lower: constructor pattern Some has literal sub-ar` | `zero / n3 / none` |
| p_ctor_deep | `dbl4 / neg2 / zl9 / add / other / other` | `EVALONLY: unbound variable "n"` | `dbl4 / neg2 / zl9 / add / other / other` |
| p_guard_let_switch | `yes / no / no` | `EVALONLY: unbound variable "$tmp518"` | `yes / no / no` |
| p_guard_param | `empty+ / empty / over / under` | `EVALONLY: unbound variable "x"` | `empty+ / empty / over / under` |
| p_scrut_call | `100 / 9 / 2` | `EVALONLY: unbound variable "b"` | `100 / 9 / 2` |
| p_lambda_match | `30 / 20 / 0` | `Error: bytecode execution failed: vm: vm: CALL: tests/golden/bytecode/` | `30 / 20 / 0` |

Further checks:

- **Corpus sweep** (`ailang disasm` over `examples/`, `std/`, `tests/golden/`, base vs fix): EvalOnly
  prototypes 2575 → 2487; **zero** prototypes became EvalOnly or changed reason. Newly compiled
  include `std/jwt.getClaimInt`, `std/jwt.checkAudience`, `std/net.httpGet/httpPost`,
  `std/sem.load_frame`, `pattern_sugar.{sumFirstTwo,sumThree,describe,firstPair}`,
  `list_pattern_cons.{secondElement,describe,firstKey,firstValue}`, `guards_module.*`,
  `std_audio_brief.convert`, `decide_jev.answerOf`.
- **Lowered-IR unit tests** (`internal/gen/lower/match_lowering_test.go`): no `FieldAccess{"Tag"}`
  anywhere; `a :: b :: rest` binds all three behind one `len >= 2`; a binding guard is evaluated
  after its `VarDecl` and falls through via the flag; a catch-all before a constructor arm is not
  a switch; a final catch-all binds the scrutinee in `Default`; `switchEligible` truth table.
- **Compiler unit tests** (`internal/bytecode/compiler/adt_tag_test.go`): named, inferred and
  unregistered-type-name tag checks; an unknown tag tags the function EvalOnly.
- **Mutation tests** (each in a scratch copy of the tree): ANDing every guard into the condition
  fails 13 parity rows + the guard unit test; dropping literal conditions fails 13 parity rows + 2
  unit tests; letting a catch-all sit anywhere in a switch fails 2 parity rows + 2 unit tests.
- `make verify-examples`, `make lint`, `make check-file-sizes`, `go test ./internal/...`.

## Files

- `internal/gen/lower/match.go` — dispatch, `switchEligible`, switch path, literal helper
- `internal/gen/lower/match_pattern.go` — recursive `patternCond` / `patternBindings`, cons normalisation
- `internal/gen/lower/match_ifchain.go` — evaluator-order if-chain with guard fall-through flag
- `internal/gen/stmt/stmt.go` — `ADTTagEq`
- `internal/bytecode/compiler/adt_tag.go` — `compileADTTagEq`, `resolveTagOrdinal`; `switch.go` — shared `inferADTFromTags`
- `internal/bytecode/compiler/{expr,lambda}.go`, `internal/gen/lower/program.go`, `internal/gen/emitgo/funcs.go` — walker cases
- Tests: `cmd/ailang/run_bytecode_match_test.go`, `tests/golden/bytecode/nested_patterns.ail`,
  `internal/gen/lower/{match_lowering_test,lower_match_test}.go`, `internal/bytecode/compiler/adt_tag_test.go`
- Example: `examples/runnable/nested_patterns.ail`

## Not in scope / known remaining gaps

- **Go backends.** emitgo (`--emit-go-v2`) now emits `ADTTagEq` the way it emits switch cases, but
  emitgo's ADT field access (`._0` vs generated `.Value0`) and unused-binding handling were already
  broken for these shapes and are untouched; generated Go for nested patterns is not verified to
  build. The golang v1 backend (`--emit-go`) does not use this lowering and keeps its own gaps
  (m-vm-var-pattern-default-arm V7/V9).
- **Cross-ADT tag ambiguity.** When the lowering cannot name the ADT (constructor arguments,
  polymorphic scrutinees), the compiler infers it from the tag in declaration order — the same
  contract and the same limitation as the switch path's `inferADTFromCases` (ailang#1355 note).
- **Non-exhaustive matches** still fall off the end of the function on the VM (implicit unit) where
  the evaluator reports "no pattern matched"; the type checker's exhaustiveness check is what
  prevents this in practice.
- `LowerMatchExpr` (non-tail matches outside `FlattenBlock`/`flattenValue`) is unchanged.

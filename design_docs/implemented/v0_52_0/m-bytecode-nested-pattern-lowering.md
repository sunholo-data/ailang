# M-BYTECODE-NESTED-PATTERN-LOWERING — pattern lowering is flat-only: nested patterns (`a :: b :: rest`) become evaluator-only, and literal/nested sub-patterns match too permissively on the strict VM (silent wrong results)

**Status**: Superseded by / implemented in [m-vm-match-lowering.md](m-vm-match-lowering.md) (2026-10-02, v0.51.1) — one recursive lowering fixes this doc together with its three siblings. Original status: Planned — quorum attempted 2026-09-30: **both external reviewers unreachable** (gemini-3-1-pro: Vertex 403 on this project; oc-kimi-k3: no Ollama endpoint), so the quorum degraded to controller-only with the absences recorded by name in `.ailang/state/mission-quorum/m-bytecode-nested-pattern-lowering-2026-09-30T21-20-16Z.json` — NOT quorum-cleared. The 24-row first-party Verification Log is the load-bearing evidence; re-run quorum when a reviewer route is available.
**Promoted to the active queue 2026-10-02** (coordinator task `task-08c86c94` — the **second independent report of row A1**: `cons2.ail` `_ :: x :: _ => x` → `unbound variable "x"`, binary v0.51.0 `b99dd25`, workaround in stapledons-godot `ai/wire_test.ail` `secondIn`). All eight rows, the flat controls, and the `ailang check` gate were re-confirmed first-party on that binary (V25-V38 below); the duplicate gate routed this report here rather than to a new doc. Quorum not re-run — the same runner fleet's 2026-10-02 attempt on the ifchain sibling recorded **all five** reviewer routes absent by name (`.ailang/state/mission-quorum/m-vm-ifchain-tag-guard-lowering-2026-10-02T15-46-27Z.json`); re-run when a route exists. Two premise corrections from newer siblings are folded in (Design Freeze items 2-3; case 6 and the guards sentence below are marked CORRECTED).
**Target**: v0.49.1 → v0.51.2 → shipped v0.51.1 via m-vm-match-lowering
**Priority**: P0 — two of the six confirmed variants are **silent wrong results** on `--strict-bytecode` (no error, no fallback, wrong value, exit 0); the other four degrade nested-pattern functions to evaluator-only, breaking strict-VM coverage.
**Estimated**: ~2–3 days (root cause fully localized; the fix mirrors an existing, correct, recursive implementation — see Solution Design)
**Dependencies**: none. Builds on [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) (landed) and is a sibling of [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) (the `len == n` check, already present at `internal/gen/lower/match.go:414-416` at HEAD). **Merge-order coordination required** (added 2026-10-02): three newer siblings — [m-vm-ifchain-tag-guard-lowering.md](../v0_51_2/m-vm-ifchain-tag-guard-lowering.md), [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md), [m-vm-adt-tag-check-lowering.md](../v0_51_3/m-vm-adt-tag-check-lowering.md) — all touch `internal/gen/lower/match.go`, and two of them correct premises this doc originally relied on (see Design Freeze).

## Problem Statement

**Current State:**

The AILANG pattern grammar (7 pattern types, arbitrarily nested) is accepted by the parser, elaborator and
type checker — and executed correctly by the tree-walking evaluator — but the bytecode path
(`internal/gen/lower/match.go`) lowers only a **flat subset** of it. Pattern matching semantics
exists in **two divergent implementations**:

1. `internal/eval/eval_patterns.go:matchPattern` — fully recursive, correct for all 7 pattern
   types and arbitrary nesting.
2. `internal/gen/lower/match.go` (`lowerPatternCond`, `lowerPatternBindings`,
   `extractBindingsAndGuards`) — flat/shape-specific; nested patterns are **silently dropped**
   from bindings, and sub-pattern conditions (literal equality, nested tags, nested lengths)
   are **never emitted**.

Consequences, all first-party-verified on this machine (binary v0.47.1, repo HEAD):

| # | Pattern (in match arm) | Symptom family | `ailang run` (evaluator) | `--strict-bytecode` |
|---|---|---|---|---|
| A1 | `a :: b :: rest` (reported; also `_ :: x :: rest`, `a :: b :: c :: rest`) | **evaluator-only** | correct | `entry is evaluator-only (compiler: unbound variable "b")` |
| A2 | nested cons as a list element: `[x :: y, _]` | **evaluator-only** | correct | `unbound variable "x"` |
| A3 | nested ADT args: `Some(Some(x))` | **evaluator-only** | correct | `unbound variable "x"` |
| A4 | nested pattern in tuple element: `(x :: _, z)` vs `([], 9)` | **silent wrong result** | `0` | `9` (exit 0) |
| B1 | literal list elements: `[1.0, 2.0]` vs `[3.0, 4.0]` | **silent wrong result** | `0.0` | `1.0` (exit 0) |
| B2 | literal tuple elements: `(1, y)` vs `(0, 5)` | **silent wrong result** | `0` | `5` (exit 0) |
| B3 | literal record fields: `{name: "alice"}` vs `{name: "bob"}` | **silent wrong result** | `other` | `alice` (exit 0) |
| B4 | nested `[]` tail: `x :: y :: []` vs `[1, 2, 3]` | **silent wrong result** | `other` | `pair` (exit 0) |

B-family bugs are worse than the reported A1: they violate NO-SILENT-FALLBACKS and A1
(determinism) in the way that is hardest to detect — a confidently wrong value from the
strict VM, exit 0, no warning. A1 is the reported bug (Stapledon mission iteration 5;
sunholo/relativity 0.3.0 works around it with one-level cons patterns only).

**Demand evidence** (the construct is common, not exotic): 10 occurrences of chained/nested
`::` patterns in the repo's own corpus — including `examples/runnable/list_pattern_cons.ail:69`
(`_ :: x :: rest => x`, the exact reported shape), `examples/runnable/pattern_sugar.ail:58,74,93`
(`a :: b :: c`, `a :: b :: c :: rest`), and `examples/runnable/std_audio_brief.ail:65`
(`path :: stem :: _`). The repo's own example currently only passes because the non-strict
bytecode path silently bridges the evaluator-only function. A second independent consumer report
(2026-10-02, task-08c86c94) works around the identical gap in stapledons-godot `ai/wire_test.ail`
(`secondIn`) with **two sequential one-level matches** —
`match xs { [] => "", _ :: r => match r { [] => "", x :: _ => x } }` — the same workaround shape the
first report documented, confirming the gap is still forcing source rewrites at v0.51.0.

**Impact:**

- AI code generators write nested list-walking destructuring naturally (`second(xs) = match xs { a :: b :: _ => b }`) — the pattern-grammar is taught by the prompt, type-checks cleanly, then either fails strict mode or silently miscomputes.
- Strict VM coverage (the purpose of `--strict-bytecode`) silently degrades for exactly the list-walking code it exists to cover.
- Divergence between two semantic implementations is the anti-pattern the parity lane exists to close (see m-bytecode-vm-parity-bugs, clause 2).

## Root Cause

**How `a :: b :: rest` reaches the lowering** (verified by reading
`internal/elaborate/patterns.go:130-152`): the parser produces `::` as a right-associative
ConstructorPattern, so `a :: b :: rest` parses as `::(a, ::(b, rest))`. The elaborator rewrites
only the **outer** `::` into `core.ListPattern{Elements: [a], Tail: &<inner>}` and elaborates the
tail recursively — the inner `::(b, rest)` becomes **another ListPattern nested in the Tail
slot**: `ListPattern{Elements:[a], Tail: &ListPattern{Elements:[b], Tail: &VarPattern rest}}`.
Only two `core.ListPattern` construction sites exist, both in `internal/elaborate/patterns.go`
(:150 `::`, :203 `[...]`/`...rest`), so every nested shape arrives as one of these forms.

**Where the lowering drops it** (verified by reading `internal/gen/lower/match.go` in full):

- `lowerPatternBindings` (:473) ListPattern case (:495-536): binds the Tail only when `*p.Tail`
  is a `*core.VarPattern` (match.go:525-534) — a nested ListPattern tail is **silently
  skipped**, so `b`/`rest` are never declared. The element loop (:497-524) likewise binds only
  VarPattern and ConstructorPattern (one level of constructor args) sub-patterns — a nested
  ListPattern element (`[x :: y, _]`) is skipped → A2. TuplePattern (:483-494) and
  RecordPattern (:538-559) bind only VarPattern elements/fields → A4's binding half.
- `lowerPatternCond` (:390) ListPattern case (:410-436): emits only a length check
  (`len == n` for closed, `len >= n` with tail — the arity fix landed) plus tag checks for
  **direct** ConstructorPattern elements. Literal elements get **no equality guard** (B1);
  the nested tail's own length condition is never checked (B4); nested sub-pattern conditions
  inside elements are never checked. TuplePattern (:406-408) and RecordPattern (:440-442)
  return `LitBool{true}` unconditionally — no literal-element guard (B2), no literal-field
  guard (B3), no field-presence condition.
- `extractBindingsAndGuards` (:228, constructor-switch path via `lowerConstructorMatch` :103):
  the `default` case (match.go:256-262) binds a nested arg to a temp `_pat_<i>` "for downstream
  destructuring (currently unsupported beyond binding)" (doc comment at :226) — the nested
  arg's sub-patterns are never destructured or condition-checked → A3 (and `Some(None)`
  would match the `Some(Some(x))` arm even if the binding existed: the inner tag is never
  compared).

The evaluator (`internal/eval/eval_patterns.go:209-263`) recursively matches elements and
`matchPattern(*p.Tail, tailList)` (:258) — which is why `ailang run` and the non-strict
bridge are correct on all eight rows above.

When a lowered function then references a dropped binding, the bytecode compiler fails at
`internal/bytecode/compiler/expr.go:93/207` (`compiler: unbound variable`), and
`internal/bytecode/compiler/compiler.go:159-176` tags the prototype `EvalOnly` with that
reason — the error surfaced by `--strict-bytecode` in the bug report. `ailang check` passes
(rc=0) on the reported repro: the type checker recursively checks nested patterns
(`internal/types/typechecker_patterns.go` ListPattern case recurses into elements and Tail),
so nothing upstream rejects the shape.

## Goals

**Primary Goal:** Make the bytecode lowering of pattern matching semantically identical to
the evaluator's `matchPattern` for all 7 pattern types at arbitrary nesting depth — every
row in the table above returns the evaluator's answer under `--strict-bytecode`.

**Success Metrics:**
- `a :: b :: rest` (and all eight rows) compile to bytecode and produce the evaluator's exact output under `--strict-bytecode`.
- Zero `unbound variable` EvalOnly tags attributable to pattern lowering across `std/`, `examples/`, and the golden corpus.
- A new exhaustive pattern-matrix golden test (nested × all 7 pattern types × match/no-match rows) passes on both engines with byte-identical output.
- Existing pattern-matching goldens (`recursion_quicksort`, `list_pattern_cons`, `pattern_sugar`, inline `tests [...]` rows) unchanged.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates an existing determinism hole: two execution engines currently disagree on the same program, one of them silently returning wrong values. |
| A2: Replayability | 0 | No trace/replay surface change. |
| A3: Effect Legibility | 0 | Patterns in pure functions stay pure; no effect annotations change. |
| A4: Explicit Authority | 0 | No capability surface change. |
| A5: Bounded Verification | +1 | Closes a gap that makes engine-parity verification (clause-2 lane) meaningful for pattern code. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | AI-generated list-walking code is the primary victim; nested cons is what models naturally emit (10 hits in the repo's own examples). |
| A8: Minimal Syntax | +1 | No new syntax — this is a semantics bug in an existing, taught construct. |
| A9: Cost Visibility | 0 | No resource semantics change. |
| A10: Composability | 0 | Fix is internal to one lowering module. |
| A11: Structured Failure | +1 | Replaces silent wrong results with correct results; remaining unsupported shapes (if any) must fail loudly in the compiler, not drop bindings silently. |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes implicit nondeterminism between engines (no new nondeterminism introduced)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): fixes a machine-facing correctness gap; optimizes nothing for human convenience

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| The lowering must mirror `eval_patterns.matchPattern` case-for-case (evaluator = reference semantics), not invent stricter checks (e.g. closed-record exactness) | Guarantees engine parity by construction; the evaluator ignores extra record fields, so adding exactness checks to bytecode alone would create NEW divergence | compiler | design | med |
| Nested sub-pattern conditions are emitted as boolean expressions in the arm guard (length checks via `_len` of `_list_tail`, equality guards for literals, tag equality for nested constructors) rather than new VM opcodes | No instruction-set change; reuses existing builtins (`_len`, `_list_get`, `_list_tail` already in `BuiltinTable`, bytecode/compiler/builtins.go:25-27); keeps the VM frozen (motoko-core bias) | agent | design | low |
| Nested sub-pattern destructuring uses synthesized temp bindings (`_tail_<i>`, `_elem_<i>`, `_pat_<i>`) holding the sub-scrutinee, with recursion on those temps | Uniform mechanism across all three lowering paths (if-chain, constructor switch, guard bodies); avoids repeated `_list_get`/FieldAccess in both cond and bindings where a temp can be hoisted | agent | design | low |
| Silent drops in the lowering become loud compile-time errors for any pattern shape the recursion does not cover (defense against the next flat-only regression) | The A-family was invisible for months because drops were silent; A11 requires fail-loud | compiler | design | low |
| Sub-scrutinee expressions are evaluated as many times as referenced (cond + bindings) rather than implementing let-hoisting/short-circuit destructuring in stmt IR | Keeps stmt IR unchanged; match scruteinees are pure values and `_list_get`/`_list_tail` are O(1)/O(n-slice) — same cost profile the flat forms already have | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Evaluator-as-reference-semantics ruling (resolved above: mirror `matchPattern`, no stricter checks)
- [x] No new VM opcodes / builtin-table extension *for cases 1-5 and 7* (resolved above: existing builtins suffice)
- [ ] **Case 6 tag projection adopts the corrected mechanism** from [m-vm-adt-tag-check-lowering.md](../v0_51_3/m-vm-adt-tag-check-lowering.md) / [m-vm-ifchain-tag-guard-lowering.md](../v0_51_2/m-vm-ifchain-tag-guard-lowering.md): the original case 6 below specified `FieldAccess{scrutinee, "Tag"}`, which is the `_record_get`-on-ADT strict-VM crash (their V-C9/V14). **Do not land case 6 as originally written** — land the corrected form (marked inline below).
- [ ] **Guard ordering adopts the bind-then-guard nested-if shape** from [m-vm-ifchain-tag-guard-lowering.md](../v0_51_2/m-vm-ifchain-tag-guard-lowering.md): the original guards sentence below ("AND onto the arm cond, as today") is the known guard-before-bindings bug (their V-G1–V-G6: `unbound variable` on the strict VM for every if-chain pattern kind). **Do not land the guards sentence as originally written** — land the corrected form (marked inline below).
- [ ] Sprint-planner confirms merge order across the four siblings (all touch `internal/gen/lower/match.go`; see Dependencies) and may re-target the version folder (v0.51.2 assumed; a v0.51.3 sibling exists)

## Solution Design

### Overview

Rewrite the three flat pattern-lowering helpers in `internal/gen/lower/match.go` as one
**recursive pair**: a condition generator and a binding generator that jointly handle every
pattern position the type checker accepts, structurally mirroring
`internal/eval/eval_patterns.go:matchPattern`. The bytecode compiler (`internal/bytecode/compiler`)
needs **no changes**: the lowered stmt IR uses only existing constructs (BinOp, BuiltinCall,
VarDecl, FieldAccess, IfStmt) and existing builtins.

> **Correction (2026-10-02, adopted from [m-vm-adt-tag-check-lowering.md](../v0_51_3/m-vm-adt-tag-check-lowering.md))**:
> the "bytecode compiler needs no changes" claim holds only for cases 1-5 and 7. The original case 6
> projected the constructor tag as `FieldAccess{scrutinee, "Tag"}`, which the compiler realizes as
> the by-name `_record_get` builtin — a trap on VM ADT values (`TagADT`, not records) and unbuildable
> under `--emit-go-v2`. The corrected design adds **one ~15-line name-based tag builtin** (`_adt_tag`,
> reading `ADTObj.Ctor`) per that sibling's frozen decision.

### Architecture

Introduce one recursive core in `internal/gen/lower/match.go`:

```go
// patternCond returns the boolean "does scrutinee match pat" expression.
// patternBindings returns VarDecls binding pat's variables from scrutinee.
// Both recurse into sub-patterns; both must cover every core.CorePattern
// type or return a loud error (no silent drops).
```

Case-by-case semantics (mirror of `matchPattern`, with existing flat behavior as the depth-1
special case):

1. **VarPattern / WildcardPattern** — cond `true`; binding `VarDecl(name, scrutinee)` (unchanged).
2. **LitPattern** — cond `scrutinee == literal` (reuse `lowerLitPatternCond`; now also emitted
   in sub-positions — fixes B1, B2, B3).
3. **ListPattern** — cond: `_len(scrutinee) <op> n` (`==` if `Tail == nil` per the landed
   arity fix, else `>=`), AND for each element the recursive cond on
   `_list_get(scrutinee, i)`, AND if `Tail != nil` the recursive cond on
   `_list_tail(scrutinee, n)`. Bindings: per-element and per-tail recursion on the same
   sub-scrutinee expressions (fixes A1 tail, A2 elements, B4 nested-`[]` length).
4. **TuplePattern** — per-element recursion on `FieldAccess{scrutinee, "_i"}` (cond AND
   binding; replaces the unconditional `true` — fixes B2 and A4's binding half).
5. **RecordPattern** — per-field recursion on `FieldAccess{scrutinee, field}` (fields sorted
   for determinism, as today). No exactness check: the evaluator only requires listed fields
   present and ignores extras; parity means the bytecode must too (B3 fix is the literal
   equality guard, not a closed-form check).
6. **ConstructorPattern (if-chain path)** — cond **CORRECTED 2026-10-02**: ~~`FieldAccess{scrutinee, "Tag"} == Name`~~ (the original form is the `_record_get`-on-ADT strict-VM crash — see the Overview correction) → use `_adt_tag(scrutinee) == Name` per [m-vm-adt-tag-check-lowering.md](../v0_51_3/m-vm-adt-tag-check-lowering.md), AND
   recursive per-arg conds on `FieldAccess{scrutinee, "_j"}` (positional field access already handles `TagADT`); bindings: recursive per-arg
   bindings (replaces the one-level `extractBindings` slice used today).
7. **Constructor switch path (`lowerConstructorMatch` + `extractBindingsAndGuards`)** — keep
   the SwitchStmt (tag dispatch stays a jump table); for each arg pattern: Var/Wildcard/Lit
   keep today's Binding/guard behavior exactly (Bug A.2's landed fix is not disturbed); for
   a nested arg pattern, bind the existing `_pat_<i>` temp and prepend, inside the case
   body, the recursive cond (as a guard `if`, same shape as the literal-sub-pattern guard)
   and the recursive bindings (fixes A3 and the missing inner tag check).

Guards (`arm.Guard`): **CORRECTED 2026-10-02** — the original text here said "continue to AND onto
the arm cond, as today", but that evaluates the guard *before* the arm's pattern bindings exist
(the G-family bug: `unbound variable` on the strict VM for every if-chain pattern kind —
[m-vm-ifchain-tag-guard-lowering.md](../v0_51_2/m-vm-ifchain-tag-guard-lowering.md) V-G1–V-G6).
Adopt that sibling's **bind-then-guard nested-if continuation**:
`If(structuralCond) { bindings; If(guard) { body } else { REST } } else { REST }`, with its
free-variable fast path (guards referencing no pattern-bound variable keep today's
`cond && guard` shape unchanged).

**Components:**
1. `internal/gen/lower/match.go` — recursive `patternCond`/`patternBindings` + switch-path
   integration (the whole fix; ~250 lines net).
2. `internal/gen/lower/lower_match_test.go` — pattern-matrix unit tests on the lowered IR.
3. `tests/golden/bytecode/` (or the existing strict-VM golden home) — end-to-end fixtures.

### Implementation Plan

**Phase 1: Recursive if-chain lowering** (~0.5 day)
- [ ] Rewrite `lowerPatternCond` + `lowerPatternBindings` as the recursive pair (cases 1-6 above); delete the flat special cases they subsume.
- [ ] Loud-error default branch for any uncovered `core.CorePattern` type.
- [ ] Unit tests: every row of the eight-row repro table lowers to IR that binds all variables and emits the expected conds.

**Phase 2: Constructor-switch path** (~0.5 day)
- [ ] Extend `extractBindingsAndGuards`'s nested-arg `default` case to emit recursive guard + bindings on the `_pat_<i>` temp (case 7).
- [ ] Unit test: `Some(Some(x))` binds `x` and guards the inner tag; `Some(None)` falls to default.

**Phase 3: End-to-end goldens + corpus sweep** (~1 day)
- [ ] Strict-VM golden fixtures: the eight-row table + `examples/runnable/list_pattern_cons.ail` `secondElement`/`describe` bodies as pure probes.
- [ ] Run `make test` (core + stdlib) and the bytecode parity corpus; confirm no EvalOnly tags attributable to patterns.
- [ ] Sweep: grep the repo corpus for nested patterns and run each under `--strict-bytecode`.

### Files to Modify/Create

**New files:**
- `tests/golden/bytecode/nested_patterns*.ail` (+ expected-output goldens) — the eight-row matrix + deep nesting — ~60 LOC AILANG
- `internal/gen/lower/match_nesting_test.go` — lowered-IR unit tests — ~200 LOC Go

**Modified files:**
- `internal/gen/lower/match.go` — recursive cond/bindings + switch-path integration — ~250 LOC net
- `internal/gen/lower/lower_match_test.go` — extend existing regression tests (do not alter landed assertions) — ~50 LOC
- `docs/LIMITATIONS.md` — remove/adjust any "nested patterns evaluator-only" note if present — ~5 LOC

## Conflict Surface

**What semantic positions does this change extend?**

Only the *lowering of match-arm patterns* in `internal/gen/lower/match.go` (both dispatch
paths: `lowerIfChainMatch` for mixed/non-constructor arms, `lowerConstructorMatch` for
all-constructor arms). No parser, elaborator, type-checker, or VM change; no new syntax.

**What other valid constructs already live in those positions?**

- Flat patterns at every depth: one-level cons (`x :: rest`), closed/open lists (`[a, b]`, `[a, ...r]`), flat tuples `(a, b)`, flat records `{name: v}`, wildcards, vars, literals, constructor arms with Var/Wildcard/Lit args (the landed Bug A.2 guard machinery).
- Explicit arm guards (`when`/`if` on arms), non-tail-position matches (routed via `FlattenBlock`/`LowerMatchStmt`), `LowerMatchExpr`'s narrow 2-arm IfExpr shape (not touched — it panics on other shapes, unchanged).
- The SwitchStmt produced for constructor matches is consumed by `internal/bytecode/compiler/switch.go` — its `Bindings` (name, FieldIndex) surface is preserved for the flat args; nested args move their destructuring into the case body, which switch.go already compiles as ordinary statements.

**How does the lowering disambiguate?**

No new ambiguity: pattern shape (depth, sub-pattern type) fully determines the recursion;
the existing constructor-vs-if-chain dispatch (`allConstructorPatterns`) is unchanged; the
`Tail == nil` ↔ `OpEq` invariant from m-bytecode-pattern-arity-fix's V-I is preserved
(closed `[...]` ⟺ `Tail == nil`; `::`-built tails always have `Tail != nil`, and the nested
tail's own `ListPattern` recurses with its own local length check).

**Which existing programs MUST still work post-change? (regression fixtures)**

1. `examples/runnable/recursion_quicksort.ail` — the pattern-arity flagship; every arm (`[]`, `[x]`, `[a, b]`, `[a, b, c]`, `[p, ...rest]`) must keep its exact output.
2. `examples/runnable/list_pattern_cons.ail` — cons example incl. `x :: y :: []` and `[("a", 1), ...]` arms; `secondElement` inline tests `([1,2,3],2) ([10,20],20) ([1],0) ([],0)` must hold.
3. `examples/runnable/pattern_sugar.ail` — `a :: b :: c`, `a :: b :: c :: rest` must now run strict AND the flat arms unchanged.
4. `examples/runnable/cons_expression.ail` — cons in expressions (not patterns) — untouched path.
5. `internal/gen/lower/lower_match_test.go` — every existing regression test (M-LOWER-FIX's `TextBlock(t) :: rest`, Bug A.2 literal-sub-pattern guards, M5) must pass unmodified.
6. `std/list.ail` `sortBy`'s `[p, ...rest]` arms and the `std/smoke.ail` cons patterns.

**What deliberately changes?**

- Nested patterns stop being evaluator-only and stop matching too permissively. Programs that *depended* on the buggy permissiveness (B1-B4 rows returning the wrong arm) change behavior on the strict VM — to match the evaluator, which is the definition of correct. The evaluator path (`ailang run` without `--bytecode`) is bit-identical before/after: no user-visible change there.
- Slightly larger lowered IR for nested patterns (more guard exprs) — accepted; correctness over code size.

## Examples

### Example 1: The reported bug

**Before** (`ailang run --quiet --bytecode --strict-bytecode --entry probe --args-json 0 cons2.ail`):
```
Error: bytecode execution failed: vm: vm: CALL: cons2.second is evaluator-only
  (compiler: unbound variable "b") but no interop bridge is wired
```

**After:**
```
2.0
```

### Example 2: A silent wrong result (B3)

**Before:**
```
$ ailang run --quiet --entry litrec --args-json 0 litrec.ail      # {name:"bob"} vs {name:"alice"}
other
$ ailang run --quiet --bytecode --strict-bytecode --entry litrec --args-json 0 litrec.ail
alice          # wrong arm, exit 0
```

**After:** both engines print `other`.

## Success Criteria

- [ ] AC1: `cons2.ail` from the bug report prints `2.0` under `--strict-bytecode` (A1).
- [ ] AC2: golden test covering all eight repro rows (A1-A4, B1-B4) — each row: evaluator output == strict-VM output == hand-computed correct value; the four B rows assert the *correct* (non-matching) arm.
- [ ] AC3: `Some(Some(7))` → `7` and `Some(None)` → default under strict (A3 incl. inner-tag guard).
- [ ] AC4: deep nesting `a :: b :: c :: rest` on `[1,2,3,4]` → `6` under strict.
- [ ] AC5: All regression fixtures in Conflict Surface pass unchanged (`make test`, goldens, inline tests).
- [ ] AC6: Corpus sweep: no `unbound variable` EvalOnly tags remain for any pattern in `std/`, `examples/`, `docparse/`.
- [ ] AC7: `ailang check` behavior unchanged (it already accepts all these shapes — rc=0 before and after).
- [ ] All tests passing
- [ ] Documentation updated (LIMITATIONS.md / bytecode-VM reference if they document pattern coverage)

## Testing Strategy

**Unit tests:**
- Lowered-IR shape tests (extend the M-LOWER-FIX style): for each pattern family, assert the emitted conds (length ops, equality guards, tag comparisons) and that every pattern variable receives a VarDecl — the A-family regressions were invisible precisely because no test asserted binding completeness.
- Property-style matrix: nested pattern × {matching input, near-miss input} — near-misses are what catch the B family (wrong length, wrong literal, wrong tag, empty-tail).

**Integration tests:**
- Strict-VM golden tests (end-to-end `--strict-bytecode` runs with expected stdout).
- `make test` (core, stdlib, examples), plus `make check-boundaries` (touches `internal/gen/lower` only — no new import edges expected, but run it per ARCHITECTURE.md).

**Manual testing:**
- Re-run the controller's first-party repro script (the eight-row table) on the built binary.
- `ailang run --bytecode --strict-bytecode examples/runnable/pattern_sugar.ail` (currently fails strict only via IO println if unwired — use `--caps IO`-free probes; IO builtins are a separate, known Phase-2E gap, out of scope here).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Strength-reduction of recursive length conds (e.g. `a :: b :: rest` → single `len >= 2` instead of `len >= 1 AND len(tail) >= 1`) — agent may choose; semantics identical, naive form preferred unless goldens show measurable cost.
- Whether to hoist sub-scrutinee temps (bind `_tail_i` once, reuse in cond) or re-emit `_list_tail(...)` expressions — agent may choose; purity makes both correct.
- Exact test-file layout (new `match_nesting_test.go` vs extending `lower_match_test.go`) — agent may choose.

## Non-Goals

**Not attempted in this feature:**
- IO/effectful builtin wiring (`__io_println` Phase-2E) — separate known gap; `--strict-bytecode` on examples that print still requires the bridge for IO.
- New pattern syntax (views, active patterns), and `LowerMatchExpr`'s non-tail-position panic — orthogonal.
- Performance of cons construction (`m-list-cons-quadratic.md` / cons-cells programme) — this doc changes *matching*, not construction or representation.
- The parked A2 parity-harness classification from `m-bytecode-vm-parity-bugs.md` — this doc's acceptance is golden outputs, not harness buckets.
- Any change to the evaluator's pattern semantics (it is the reference).

## Timeline

**Week 1** (~2.5 days):
- Day 1: Phase 1 (recursive if-chain lowering + unit tests)
- Day 2: Phase 2 (constructor-switch path + unit tests)
- Day 3: Phase 3 (goldens, corpus sweep, docs) + buffer

**Total: ~2.5–3 days** (2x'd from a ~1.5-day estimate)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-------------|
| Regression in the landed Bug A.2 literal-sub-pattern guards or the M-LOWER-FIX cons-head binding (this code has been patched incrementally before) | High | Existing regression tests are fixtures (must pass unmodified); the rewrite must preserve their exact lowered-IR assertions — extend, don't rewrite, those tests |
| Larger lowered IR / register pressure in deeply nested patterns (guard exprs multiply) | Med | Regalloc already caps at 255; matrix tests include a depth-4 case; strength-reduction deferred but available |
| New divergence introduced by a case the recursion misses | Med | Loud-error default branch (no silent drops); the 7 pattern types are a closed set (`core.CorePattern` interface) — a type-switch with default error enforces coverage at compile time |
| Evaluator has its own latent nested-pattern bug we mirror instead of fix | Low | Evaluator verified correct on all eight rows first-party (this doc's table); its recursion is the long-standing shipped behavior |

## Related Documents

**Implemented (may inform design):**
- [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) — the landed one-level fix (cons-head constructor bindings) this doc generalizes; its test file is a regression fixture.
- [m-bytecode-2d-parity.md](../implemented/v0_11_0/m-bytecode-2d-parity.md) — the parity program this belongs to.

**Planned (check for overlap):**
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — SIBLING, distinct: fixed-length list `len == n` (its `OpEq` fix is present at HEAD, match.go:414-416); does NOT cover nesting, literal guards, tuple/record/sub-pattern conds. This doc preserves its `Tail == nil ⟺ OpEq` invariant (its V-I: exactly two `core.ListPattern` construction sites, re-verified below).
- [m-vm-ifchain-tag-guard-lowering.md](../v0_51_2/m-vm-ifchain-tag-guard-lowering.md) — SIBLING (2026-10-02): owns the guard-before-bindings ordering family (G) and ADT tag conds in if-chain positions (C); this doc adopts both corrections (Design Freeze items 2-3). That doc does not duplicate this one (its V-A1/V-B1 re-verified this doc's rows A1/B1 on v0.51.0) and recommended this promotion. Whichever lands second re-verifies the composed shapes (`a :: b :: _ if a > b`, `Some(x) :: rest`).
- [m-vm-var-pattern-default-arm.md](../v0_51_2/m-vm-var-pattern-default-arm.md) — SIBLING: var/misplaced default arms in constructor matches; its depth-1 if-chain constructor addition is the depth-1 subset of this doc's corrected case 6 — whichever lands second generalizes rather than rewrites (its V26 and merge-collision risk row).
- [m-vm-adt-tag-check-lowering.md](../v0_51_3/m-vm-adt-tag-check-lowering.md) — SIBLING, **prerequisite correction** (2026-10-02): the constructor tag check both older siblings originally built on is broken (strict-VM crash AND unbuildable `--emit-go-v2`); its `_adt_tag` node is the mechanism this doc's corrected case 6 now uses. Its depth-1 loud-boundary contract (Lit/nested args in if-chain constructor patterns panic → EvalOnly) is the stopgap this doc's recursive rewrite removes.
- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md) — parked parity harness work (A2 classification); this doc does not depend on it and is not blocked by it.
- [m-list-cons-cells-decomposition.md](m-list-cons-cells-decomposition.md) — cons *construction* performance programme (D-19: B); orthogonal to matching semantics.

**Distinctness note** (per the duplicate gate): the neural/simhash search found no implemented match for this topic; the nearest planned doc (pattern-arity) covers the length operator of closed list patterns only — the eight rows verified above are all outside its landed scope.

## Verification Log

Every "does/doesn't support" claim above carries a first-party check (binary `AILANG v0.47.1` @ `0107d3c3`, repo HEAD `cb7c51c8`, std v0.49.0 — the stdlib-version-mismatch warning is environmental, not semantic).

| # | Claim | Check | Result |
|---|---|---|---|
| V1 | Reported bug A1 reproduces first-party | `ailang run --quiet --entry probe --args-json 0 cons2.ail` / `--bytecode` / `--bytecode --strict-bytecode` (bug report's exact file) | `2.0` / `2.0` / `Error: ... cons2.second is evaluator-only (compiler: unbound variable "b") ... op CALL`, exit 1 |
| V2 | A1-deep (`a :: b :: c :: rest`) | entry `deep` on `[1,2,3,4]` | eval `6`; strict: `unbound variable "b"` |
| V3 | A2: nested cons as list element `[x :: y, _]` | entry `innerList` on `[[1,2],[3]]` | eval `1`; strict: `unbound variable "x"` |
| V4 | A3: nested ADT args `Some(Some(x))` | entry on `Some(Some(7))`, `import std/option (Option, Some, None)` | eval `7`; strict: `unbound variable "x"` |
| V5 | A4: nested pattern in tuple element, near-miss input | `(x :: _, z)` vs `([], 9)` | eval `0`; strict `9` (exit 0, no error) |
| V6 | B1: literal list elements | `[1.0, 2.0]` arm vs `[3.0, 4.0]` | eval `0.0`; strict `1.0` |
| V7 | B2: literal tuple elements | `(1, y)` arm vs `(0, 5)` | eval `0`; strict `5` |
| V8 | B3: literal record fields | `{name: "alice"}` arm vs `{name: "bob"}` | eval `other`; strict `alice` |
| V9 | B4: nested `[]` tail | `x :: y :: []` arm vs `[1, 2, 3]` | eval `other`; strict `pair` |
| V10 | Flat controls still strict-clean (no regression premise) | one-level `a :: rest` (→`1.0`), `[a, b, ...r]` (→`2.0`), var-only record `{fst: a, snd: b}` (→`3`) | eval == strict == correct for all three |
| V11 | `ailang check` accepts the reported shape (bug is lowering-only, not type-level) | `ailang check cons2.ail` | rc=0 |
| V12 | Elaboration shape: `::` → ListPattern with recursive tail | read `internal/elaborate/patterns.go:130-152` (`::` case builds `ListPattern{Elements:[head], Tail:&tailPat}` with `tailPat` recursively elaborated) | Confirmed |
| V13 | NEGATIVE: only two `core.ListPattern` construction sites exist | `grep -rn "core\.ListPattern{\|&ListPattern{" internal/ \| grep -v _test` | `gob.go:38` (registration only), `elaborate/patterns.go:150`, `:203` — matches pattern-arity doc's V-I; every nested shape arrives via these |
| V14 | Evaluator handles nesting recursively (evaluator = reference semantics) | read `internal/eval/eval_patterns.go:209-263`; tail recursion at `:258` `matchPattern(*p.Tail, tailList)`; record case `:268-292` requires listed fields present, ignores extras | Confirmed |
| V15 | NEGATIVE: lowering drops nested Tail bindings (A-mechanism) | read `internal/gen/lower/match.go:525-534` (Tail bound only if `*core.VarPattern`), `:497-524` (elements: Var/Constructor only), `:483-494` (tuple: Var only), `:538-559` (record: Var only) | Confirmed — read the file in full; no other binding path exists (callers: match.go:309 only) |
| V16 | NEGATIVE: lowering emits no sub-pattern conds (B-mechanism) | read `lowerPatternCond` `match.go:390`: ListPattern case `:410-436` (length + direct-constructor tags only), TuplePattern `:406-408` `LitBool{true}`, RecordPattern `:440-442` `LitBool{true}` | Confirmed |
| V17 | NEGATIVE: switch path leaves nested args not destructured (A3 mechanism) | read `extractBindingsAndGuards` default case `match.go:256-262` (`_pat_<i>` temp only) + doc comment `:226` ("currently unsupported beyond binding") — the gap is stated in the code itself | Confirmed |
| V18 | EvalOnly tag mechanism (the reported error) | read `internal/bytecode/compiler/compiler.go:159-176` (compile error → `proto.EvalOnly = true, EvalReason = ...`); unbound-var error sites `internal/bytecode/compiler/expr.go:93,207` | Confirmed |
| V19 | Builtins needed exist; no VM/opcode change required | `internal/bytecode/compiler/builtins.go:25-27` (`_len`, `_list_get`, `_list_tail` in `BuiltinTable`) | Confirmed |
| V20 | Arity fix (`len == n`) already landed (sibling claim, distinctness) | `internal/gen/lower/match.go:414-416` (`if p.Tail == nil { lenOp = OpEq }`) | Confirmed present at HEAD |
| V21 | Demand: nested cons is common in the corpus | `grep -rn -E '[a-zA-Z_)] :: [a-zA-Z_]\w* ::' std/ examples/` | 10 hits incl. `examples/runnable/list_pattern_cons.ail:69`, `pattern_sugar.ail:58,74,93`, `std_audio_brief.ail:65` |
| V22 | Repo's own example exercises the bug shape | `examples/runnable/list_pattern_cons.ail:66-71` (`secondElement`, `_ :: x :: rest`) | Confirmed; currently passes only via the evaluator bridge (its `println` arms hit the separate IO builtin gap) |
| V23 | Regression fixtures exist | `ls examples/runnable/recursion_quicksort.ail cons_expression.ail pattern_sugar.ail list_pattern_cons.ail` | all four exist |
| V24 | Existing lowering regression tests are real (M-LOWER-FIX, Bug A.2) | `sed -n '150,175p' internal/gen/lower/lower_match_test.go` (M5 cons-head test); `m-lower-fix.md`, `m-bytecode-pattern-arity-fix.md` implemented/planned per above | Confirmed |

## Re-verification on the reported binary (2026-10-02, task-08c86c94)

Row A1 was reported a **second time** through the coordinator (this task: `cons2.ail`,
`_ :: x :: _ => x`, `unbound variable "x"`, binary v0.51.0 `b99dd25` — md5 `ed0478ccc2a4565ac4fbbcdd0419ce74`, repo at that commit, workaround in stapledons-godot `ai/wire_test.ail` `secondIn`). The duplicate gate routed the report here instead of to a new doc. The full eight-row table, the flat controls, and the `ailang check` gate were re-run first-party on that binary; every row reproduces unchanged from the v0.47.1 run above.

| # | Claim | Check | Result |
|---|---|---|---|
| V25 | A1 re-confirmed on v0.51.0 with this task's exact repro | `ailang run` ×3 modes on the task's `cons2.ail` (`match xs { _ :: x :: _ => x, _ => "" }`) | eval `b`; `--bytecode` `b` (bridge); `--strict-bytecode` `Error: ... cons2.second is evaluator-only (compiler: unbound variable "x") ... op TAIL_CALL`, exit 1 |
| V26 | A2 re-confirmed | `[x :: y, _]` vs `[[1,2],[3]]` | eval `1`; strict `unbound variable "x"` |
| V27 | A3 re-confirmed | `Some(Some(x))` vs `Some(Some(7))`, `import std/option` | eval `7`; strict `unbound variable "x"` |
| V28 | A4 re-confirmed (silent wrong) | `(x :: _, z)` vs `([], 9)` | eval `0`; strict `9`, exit 0 |
| V29 | B1 re-confirmed (silent wrong) | `[1, 2]` arm vs `[1, 3]` | eval `no`; strict `matched-lit-yes`, exit 0 |
| V30 | B2 re-confirmed (silent wrong) | `(1, y)` arm vs `(0, 5)` | eval `0`; strict `5`, exit 0 |
| V31 | B3 re-confirmed (silent wrong) | `{name: "amy"}` arm vs `{name: "zed"}` | eval `other`; strict `amy`, exit 0 |
| V32 | B4-family re-confirmed (silent wrong, cond side — this task's new variant row) | `_ :: _ :: x :: _` arm vs `["a"]` (length 1, must NOT match); also `1 :: 2 :: _` vs `[1, 5]` | eval `0` / `0`; strict `1` / `1`, exit 0 — the nested tail's own length and literal conds are still never emitted, so even un-referenced nested bindings mis-match |
| V33 | Flat controls still strict-clean (regression premise) | one-level `a :: rest`, `[a, b, ...r]`, `{fst: a, snd: b}` | eval == strict == correct (`1`, `3`, `3`) for all three |
| V34 | `ailang check` still accepts the reported shape | `ailang check cons2.ail` | rc=0 (MOD010 temp-path warning only) |
| V35 | V19 citation drift: `_len`/`_list_get`/`_list_tail` still registered, table relocated | `internal/bytecode/builtin_names.go:18-21` (`BuiltinNames`), consumed via `BuiltinTable = bytecode.BuiltinNames` (`internal/bytecode/compiler/builtins.go:10-11`) | Confirmed — the original `builtins.go:25-27` citation is stale; mechanism intact, citation corrected here |
| V36 | Mechanism citations at HEAD `b99dd25` | re-read `internal/gen/lower/match.go` in full | all present within ±2 lines of the original citations: `extractBindingsAndGuards` :228 with `_pat_%d` :258 and "currently unsupported beyond binding" :227; `lowerPatternCond` :390 (ListPattern cond :410-436 with `lenOp = OpEq` :416; Tuple/Record unconditional `LitBool{true}` :408/:442); `lowerPatternBindings` :473 with Tail-bound-only-if-`*core.VarPattern` :525. `core.ListPattern` construction still exactly two sites (`internal/elaborate/patterns.go:150, :203`) |
| V37 | Demand greps still hold at HEAD | `grep -rnE "[a-zA-Z_)] :: [a-zA-Z_]+ ::" std/ examples/` | same hits: `list_pattern_cons.ail:55,69`, `pattern_sugar.ail:58,74,93`, `std_audio_brief.ail:65`, `cons_expression.ail:51` |
| V38 | Promotion is non-duplicative (duplicate gate) | neural/simhash coverage check: this doc's A1 row is the reported repro; the three v0.51.x siblings each disclaim it by name (ifchain V-A1, adt-tag Dependencies) | Confirmed — no new doc created; this doc re-targeted to v0.51.2 and refreshed instead |

## References

- [Design Axioms](/docs/references/axioms) — A1/A11 drive the P0
- Bug report: Stapledon mission iteration 5 (designer prototype), reproduced first-party by the controller — V1-V9 above
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — sibling landed fix whose `Tail == nil ⟺ OpEq` invariant this design preserves
- [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) — the prior one-level slice of this bug class (the incremental-special-casing anti-pattern this doc ends)
- `internal/eval/eval_patterns.go` — reference semantics
- `internal/gen/lower/match.go` — the module this doc rewrites

## Future Work

- If goldens show nested-guard IR bloat matters, a strength-reduction pass on the emitted conds (see Deferred Decisions).
- The parked parity-harness classification (m-bytecode-vm-parity-bugs A2) can reuse this doc's pattern-matrix goldens as ground truth once unparked.
- Exhaustiveness checking for nested patterns (compile-time, not runtime) — orthogonal type-checker work.

---

**Document created**: 2026-09-30
**Last updated**: 2026-10-02 — re-verified on the reported binary v0.51.0 `b99dd25` (task-08c86c94, second independent report of row A1; V25-V38); promoted to the active queue (Target re-set to v0.51.2 per the ifchain sibling's recommendation); premise corrections from the v0_51_2/v0_51_3 siblings folded into the Overview, case 6, the guards sentence, and Design Freeze

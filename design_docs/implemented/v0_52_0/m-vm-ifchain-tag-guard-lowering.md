# M-VM-IFCHAIN-TAG-GUARD-LOWERING — ADT sub-pattern conds crash the strict VM (`_record_get` on an ADT) and guards evaluate before pattern bindings (unbound variable) — two if-chain lowering gaps outside the nested-pattern and var-pattern sibling scopes

**Status**: Superseded by / implemented in [m-vm-match-lowering.md](m-vm-match-lowering.md) (2026-10-02, v0.51.1) — one recursive lowering fixes this doc together with its three siblings. Original status: Planned — quorum attempted 2026-10-02 (`.ailang/state/mission-quorum/m-vm-ifchain-tag-guard-lowering-2026-10-02T15-46-27Z.json`): **all five external reviewers absent** (oc-glm-5-3: unreachable, no Ollama endpoint; oc-kimi-k3: unreachable; gpt6-1-sol: no OPENAI_API_KEY; gemini-3-1-pro: Vertex 403 on this project; claude-sonnet-5@claude-p: no Anthropic OAuth credential in this runner) — the same full-absence degradation both v0.51.x siblings recorded. Synthesis degraded to controller-only, verdict **proceed** with the absences recorded by name — NOT quorum-cleared; re-run quorum when a reviewer route is available. The 31-row first-party Verification Log (every claim backed by a live binary run on the reported build or a source read at cited lines) is the load-bearing evidence.
**Target**: v0.51.2 (bug fix; strict-VM coverage — clause-2 parity family, same file and sprint folder as [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md))
**Priority**: P1 — neither family is a silent wrong result (both fail loud in strict, both silently fall back to the correct evaluator answer in plain `--bytecode`), but both block strict-VM coverage for taught, type-checking constructs, and both are **prerequisites for the acceptance criteria of the two planned sibling docs in the same file** (see Problem Statement, items 3-4).
**Estimated**: ~2 days (root causes fully localized; both fixes reuse mechanisms that exist and work today in adjacent code paths of the same files)
**Dependencies**: none for either fix in isolation. Sibling of [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) (planned — owns the reported nested-cons bug) and of [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md) (planned — owns the var-default-arm family). Split-of-scope contract in Related Documents.

**Reported from**: coordinator task `task-982a4d4f` ("strict VM: nested cons pattern `a :: b :: _` is evaluator-only (unbound variable b) on v0.51.0", external workaround in stapledons-godot `sim/core_test.ail`, possibly related to ailang#1473 and #1503). The nested-cons repro itself is **row A1 of the planned sibling doc and is NOT re-designed here** — this doc covers the two variant classes the sibling docs do not, discovered while reproducing the report on the exact reported binary.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes two evaluator↔VM divergences: ADT sub-pattern conds crash the VM (evaluator correct) and guards over pattern bindings are unbound on the VM (evaluator correct). Same program, different engine outcomes, is the A1 hole this family exists to close. |
| A2: Replayability | 0 | No trace or replay machinery touched. |
| A3: Effect Legibility | 0 | Pure pattern-matching lowering only; no effects involved. |
| A4: Explicit Authority | 0 | No capability or authority surface touched. |
| A5: Bounded Verification | +1 | Two whole construct classes (guards over if-chain pattern bindings; ADT constructors in sub-pattern positions) become strict-VM-decidable instead of evaluator-only. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | `match xs { [a, b] if a > b => … }` and `Some(x) :: _ => …` are idioms AI generators emit first; today the first silently degrades the function to evaluator-only and the second crashes the strict VM. |
| A8: Minimal Syntax | +1 | No new syntax — existing, documented, evaluator-correct constructs start working on the strict VM. |
| A9: Cost Visibility | 0 | No resource semantics change. |
| A10: Composability | 0 | Neutral. |
| A11: Structured Failure | +1 | The C-family runtime crash (`BUILTIN_CALL: _record_get: arg 0 must be record, got ADT`) becomes a correct compile; anything the new lowering still cannot cover fails loudly (panic → EvalOnly stub with reason), never silently. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes route divergence; introduces none (the lowering is a pure structural function of the elaborated Core).
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): removes human-only workaround requirements (`match the head, then match the rest separately`).

## Problem Statement

**Current State:**

The coordinator task reported the nested-cons evaluator-only bug. First-party reproduction on the reported binary (v0.51.0 `b99dd25`) **confirmed that bug and found four more pattern-lowering families in the same file**. One is already designed (nested-pattern sibling doc), one belongs to the var-pattern sibling's gate; the remaining two — the subject of this doc — are covered by **no planned or implemented doc**, and both planned siblings' acceptance criteria **depend on them**:

1. **C-family (strict-VM crash, the reported "issue #1503" hint): constructor sub-patterns in if-chain positions.** `Some(x) :: rest`, `[Some(x)]`, `(Some(x), _)` — the arm compiles (no unbound variables) but **crashes the strict VM at runtime**: `BUILTIN_CALL: _record_get: arg 0 must be record, got ADT` (V-C1, V-C2). Root cause: `lowerPatternCond`'s `ConstructorPattern` case projects the scrutinee's tag with `FieldAccess{Field: "Tag"}` (match.go:394-400), which the bytecode compiler can only realize as the by-name `_record_get` builtin (collections.go:221-265) — and VM ADT values are `TagADT`, not records (builtins.go:174 rejects them). The **binding** side of the same shape already works (it uses `FieldAccess "_j"` → positional `OpGetField`, which handles `TagADT` — vm.go:486-494), so the cond alone is broken (V-C6). Plain `--bytecode` returns the *correct* answer because the runner falls back to the evaluator on any VM error (V-C3) — the crash is masked in every mode except strict.

2. **G-family (evaluator-only, unbound variable): guards evaluate before pattern bindings exist.** In `lowerIfChainMatch`, `arm.Guard` is ANDed into the arm **cond** (match.go:355-366) while the arm's pattern bindings are `VarDecl`s prepended to the arm **body** (match.go:307-312) — so any guard that references a pattern-bound variable references a name that is not yet declared at cond-evaluation time → `compiler: unbound variable "x"` → EvalOnly → strict fails. Verified for **every** if-chain pattern kind: list `[a, b] if a > b` (V-G1), cons `x :: rest if x > 5` (V-G6), record `{x: v} if v > 3` (V-G4), tuple `(a, b) if a < b` (V-G5), var default `other if other == "b"` (V-G3), and nested cons `a :: b :: _ if a > b` (compounded with the sibling's missing nested bindings, V-G2). The evaluator's reference semantics are the opposite order: `matchPattern` → **bindings pushed into a child environment** → guard evaluated with them → on false, next arm (eval_patterns.go:47-83, V-G9). A control shape whose guard references only a lambda parameter (not a pattern binding) compiles and runs strict today (V-G8) — guards per se work; only the ordering is wrong.

3. **Blocking dependency on the nested-pattern sibling.** Its Solution Design case 6 specifies the if-chain `ConstructorPattern` cond as `FieldAccess{scrutinee, "Tag"} == Name` — **the exact mechanism that crashes the VM today** (V-C9). Its Verification Log has no row testing an ADT sub-pattern in an if-chain position (its A3 covers `Some(Some(x))`, which routes to the *switch* path). If its sprint lands as written, `Some(x) :: rest` will still crash the strict VM — a correct recursive rewrite on top of a broken projection. Its design also states guards "continue to AND onto the arm cond, as today" (V-C10) — "as today" is the G-family bug, so its primary goal (evaluator parity at arbitrary nesting) is unachieable for any guarded arm that references its own bindings without this doc's fix.

4. **Blocking dependency on the var-pattern sibling.** Its V4 fix routes guarded var arms to the if-chain "with the guard ANDed in" and its AC2 promises the evaluator's value for that shape under `--strict-bytecode` (V-G10). m21 (V-G3) is that exact shape today — a mixed match with a guarded var arm — and it is `unbound variable "other"` on the strict VM. Without this doc's G-fix (or an equivalent hoist in their sprint), their AC2 cannot pass; the two fixes touch the same function (`lowerIfChainMatch`) and must be coordinated either way.

5. **The reported repro itself (A1) is confirmed and already designed.** `a :: b :: _` → evaluator-only, `unbound variable "b"` — re-verified first-party on v0.51.0 (V-A1, plus exact-tail, wildcard-head, and depth-3 variants V-A2). It is row A1 of [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md), verified by that doc on v0.47.1; its B-family (silent wrong results for unguarded literal sub-patterns) also re-verified here on v0.51.0 (V-B1). That doc targets v0.49.1 — **two released versions stale** — and should be promoted into the active queue; this doc does not duplicate it.

**Verified symptom table** (binary v0.51.0 `b99dd25c…`-dirty, all commands run in this session; programs listed in the Verification Log):

| # | Shape | `ailang run` (evaluator) | `run --bytecode` | `run --bytecode --strict-bytecode` | Owned by |
|---|---|---|---|---|---|
| C1 | `Some(x) :: rest` (cons head) | `ok5:3` | `ok5:3` (fallback masks crash) | **crash**: `_record_get: arg 0 must be record, got ADT` | **this doc** |
| C2 | `[Some(x)]` (exact list element) | `ok20:4` | `ok20:4` (fallback) | **crash**: same | **this doc** |
| G1 | `[a, b] if a > b` | `ok14` | `ok14` (bridge) | **error**: evaluator-only, `unbound variable "a"` | **this doc** |
| G2 | `x :: rest if x > 5` | `ok24` | `ok24` (bridge) | **error**: `unbound variable "x"` | **this doc** |
| G3 | `"a" => …, other if other == "b" => …` (mixed match, guarded var arm) | `B` | `B` (bridge) | **error**: `unbound variable "other"` | **this doc** (+ var-doc dependency) |
| G4 | `{x: v} if v > 3` | `ok22` | `ok22` (bridge) | **error**: `unbound variable "v"` | **this doc** |
| G5 | `(a, b) if a < b` | `ok23` | `ok23` (bridge) | **error**: `unbound variable "a"` | **this doc** |
| A1 | `a :: b :: _` (the reported repro) | correct | correct (bridge) | **error**: `unbound variable "b"` | nested-pattern sibling (row A1) |
| B1 | `[1, 2, …r]` vs `[9, 9, 9]` | `no3` (correct) | **`ok3` — silent wrong answer, exit 0** | `ok3` — silent wrong | nested-pattern sibling (row B1) |
| D1 | `match n { x => x + 1 }` (all-var arms, no guard) | `42` | `42` (bridge) | **error**: `unknown ADT "" in switch` | var-pattern sibling (its gate routes these to the if-chain) |

**Impact:**

- Strict-VM coverage (the point of `--strict-bytecode`) silently excludes two idiomatic construct classes; consumers discover it only at strict-run time, then must rewrite source (the stapledons-godot workaround pattern: "match the head, then match the rest separately").
- Two planned sprints in the same file have acceptance criteria that cannot pass without the fixes in this doc; landing them as written ships a still-crashing cond mechanism and an unreachable guard promise.
- The task report's "possibly related to #1503 (constructor at the head of a cons pattern)" is confirmed as a **different, previously undescribed failure mode** (VM crash, not unbound variable) — the repro battery distinguishes them first-party.

## Goals

**Primary Goal:** An ADT constructor in any if-chain sub-pattern position, and a guard referencing its own arm's pattern bindings, each lower to bytecode that returns the evaluator's exact answer under `--strict-bytecode`.

**Success Metrics:**
1. C1/C2 shapes return the evaluator's value under `--strict-bytecode` (today: VM crash).
2. G1-G5 shapes return the evaluator's value under `--strict-bytecode` (today: evaluator-only).
3. The control shape (guard referencing only a non-pattern variable) and the switch-path guarded shape (`Some(x) if x > 3`) remain strict-clean, byte-identical output (today: already correct — must not regress).
4. Zero new EvalOnly functions and zero output changes across `std/` + `examples/` in a strict-VM sweep (the corpus exercises the control and switch-path shapes; the fixed shapes' corpus presence is bare-expression only — V-G11, V-G12).
5. Both planned sibling docs' acceptance criteria become achievable (their ACs include shapes owned here — see Related Documents contract).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| C-fix mechanism: a new `_adt_tag(v) -> string` VM builtin (constructor **name**), not reuse of the switch path's `OpGetTag`+ordinal | The if-chain cond is a `stmt.Expr` with no ADT-type context; the ordinal route requires `adtTypes` knowledge the lowering doesn't have and inherits the known cross-ADT ordinal ambiguity (ailang#1355 note in switch.go). Name comparison is exactly the evaluator's semantics (`CtorName` match, eval_patterns.go:187) and the VM already stores the name (`ADTObj.Ctor`, value.go:171-177 — V-C8). | agent | design | low |
| C-fix site: the **lowering** (`lowerPatternCond`'s ConstructorPattern case), not the VM or `_record_get` | The VM representation is correct (records and ADTs are legitimately different); the projection was wrong. Overloading `_record_get` to read ADT tags would blur value types at the one place the type discipline is clearest. | agent | design | low |
| G-fix shape: bind-then-guard with a **nested-if continuation** — `If(structuralCond) { bindings; If(guard) { body } else { REST } } else { REST }` — accepting bounded duplication of the remaining-arm chain | The three alternatives all fail: (a) AND the guard into the cond (today's shape) evaluates the guard before the bindings exist; (b) hoist bindings above the whole chain makes projections run on non-matching scrutinees, where `_list_get` traps at runtime (the evaluator never traps — it just fails the arm); (c) an early-exit/flag IR construct is a cross-emitter IR change. Duplication has landed precedent (Bug A.2 duplicates the default body into guard-failing `Else` branches). | agent | design | med |
| G-fix fast path: arms whose guard references **no** pattern-bound variable keep today's `cond && guard` shape unchanged (via a free-variable check) | Avoids IR growth and register pressure for the common guard shapes that already work; only the broken shapes pay the duplication. A free-variable walker already exists in the compiler package (`freeVarsLambda`/`freeVarVisitor`, lambda.go:113-150 — V-G13) to model the helper on. | agent | design | low |
| Evaluator remains the fixed point: the lowered conds/bindings must mirror `matchPattern` + the linear arm loop (bindings → guard → next-arm-on-false), never invent stricter checks | Same ruling as the nested sibling: parity by construction; any stricter check on one engine creates a NEW divergence. | compiler | design | low |
| Fail-loud for anything the new lowering still cannot express (panic → EvalOnly stub with reason), never a silent drop | The A-family was invisible for months because drops were silent; A11 requires loud. | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] C-fix mechanism: `_adt_tag` name-based builtin (decided above; alternatives rejected in Solution Design)
- [x] G-fix shape: nested-if continuation with bounded duplication (alternatives rejected above)
- [x] G-fix fast path: free-variable check, today's shape preserved for non-referencing guards
- [ ] Sprint-planner confirms merge order against the two siblings (all three touch `internal/gen/lower/match.go`; see Risks) and may re-target the version folder (v0.51.2 assumed)

## Solution Design

### Overview

Two small, independently landable fixes to the if-chain half of `internal/gen/lower/match.go`, plus one ~15-line VM builtin. No parser, elaborator, type-checker, effect, or VM-instruction changes; no new syntax. The bytecode compiler needs no change for the G-fix (it already compiles nested `IfStmt` and `VarDecl` correctly — every control shape today proves it) and gains exactly one builtin row for the C-fix.

### Architecture

**C-family root cause** (single sentence): the if-chain's constructor cond projects the tag with a by-name record access, but VM ADT values are not records — the correct name-based read (`ADTObj.Ctor`) has no lowering surface, so the lowering invents `FieldAccess{"Tag"}` and the compiler realizes it as `_record_get`, which traps.

**Components:**

1. **`_adt_tag` native builtin — the two-step contract** (~15 LOC total, exactly as documented at `internal/bytecode/builtin_names.go:11-14`):
   - Step 1: append `"_adt_tag"` to `bytecode.BuiltinNames` (`internal/bytecode/builtin_names.go`) — the canonical source-ordered table that is the contract between compiler and VM (the compiler derives its index map from it; no compiler change needed).
   - Step 2: add the matching dispatch entry `builtinADTTag` in `internal/vm/builtins.go`'s `BuiltinTable`, same position. `validateBuiltinTables` checks length agreement at package init; the combined index space is asserted ≤ 256 at `internal/bytecode/compiler/builtins.go:23-26` (current table far below).
   - `builtinADTTag(args)`: one arg; if `Tag != TagADT` → loud error `_adt_tag: arg 0 must be ADT, got %s` (mirrors `builtinRecordGet`'s arg discipline, builtins.go:171-174); else return `NewString(args[0].AsADT().Ctor)`. The `_`-prefixed snake-case name matches `_record_get`/`_list_get`/`_len`/`_list_tail`.
   - Rejected alternative: reuse `OpGetTag` + ordinal (see High-Impact Decisions) — the if-chain cond is a `stmt.Expr` with no ADT-type context, and the ordinal is per-type-local (value.go:168-170), so a name-based builtin is both simpler and *more* parity-correct than the switch path itself.

2. **`internal/gen/lower/match.go` — rewrite the `ConstructorPattern` case of `lowerPatternCond`** (~10 LOC net):
   - Before: `BinOp{OpEq, Left: FieldAccess{Record: scrutinee, Field: "Tag"}, Right: LitString{p.Name}}` (match.go:394-400).
   - After: `BinOp{OpEq, Left: BuiltinCall{Name: "_adt_tag", Args: []stmt.Expr{scrutinee}}, Right: LitString{p.Name}}`.
   - Bindings need no change: the depth-1 `ConstructorPattern` element case already binds via `_head_<i>` + `FieldAccess "_j"` → positional `OpGetField`, which handles `TagADT` (V-C6) — `Some(x) :: rest` becomes fully strict-correct with the cond fix alone (worked example below).
   - Nullary constructors (`None` as an element/tail) use the same tag-cond path with no args to bind.

3. **`internal/gen/lower/match.go` — restructure `lowerIfChainMatch` for guards** (~70 LOC net):
   - Today (match.go:303-377): `armBody` = bindings + body; cond = `lowerPatternCond(pattern)`; guard, if present, is ANDed onto the cond. Fix, per arm:
     - `patternVars` = the set of variable names the arm's pattern binds (a small collector over `core.CorePattern`, or reuse of the existing binding walk).
     - **Fast path** (guard is nil, or `freeVars(guard) ∩ patternVars = ∅`): exactly today's shape — `If(cond && guard) { bindings; body } else { REST }`. The control shape (V-G8) and switch-path-guarded matches are untouched by construction.
     - **Nested path** (guard references pattern bindings): `If(structuralCond) { bindings…; If(guard) { body } else { REST } } else { REST }`, where `REST` is the already-built chain for the remaining arms (built last-to-first as today). Guard failure falls to the next arm in source order — the evaluator's `continue` (eval_patterns.go:82-83) — and the bindings of the failed arm are dead (rebound by whichever later arm matches; cross-arm rebinding is already exercised today, e.g. `[a, b] => …, [a] => …`).
     - The last arm's fall-off-the-end behavior is unchanged from today (an arm whose structural cond + guard both fail yields the same inexhaustive fall-through the current refutable-last-arm branch produces — match.go:333-341): the evaluator errors `no pattern matched in match expression` (eval_patterns.go:102) and the lowered form returns unit; that pre-existing terminal divergence for inexhaustive-at-runtime matches is noted, not widened (Non-Goals).
   - Note the boundary: this lands **on top of** the current flat `lowerPatternBindings`; when the nested sibling's recursive rewrite lands, its per-sub-pattern bindings compose with this ordering fix unchanged (merge contract below).

4. **`internal/bytecode/` — no changes** beyond the builtin row: `compileBuiltinCall` already dispatches by name from `BuiltinTable` (builtins.go index map), `IfStmt`/`VarDecl`/nested blocks already compile (every working if-chain fixture proves the machinery), and `OpGetField` already handles ADT (vm.go:486-494).

### Implementation Plan

**Phase 1: C-fix (builtin + cond rewrite)** (~0.5 day)
- [ ] `builtinADTTag` + `BuiltinTable` row; VM unit test (ADT → ctor name; non-ADT → loud error), mirroring `builtinListGet`'s test shape.
- [ ] Rewrite `lowerPatternCond`'s `ConstructorPattern` case to `_adt_tag` equality.
- [ ] Unit tests in `internal/gen/lower/lower_match_test.go`: cons-head ADT element and exact-list ADT element conds assert `BuiltinCall{"_adt_tag"}` (not `FieldAccess{"Tag"}`).
- [ ] End-to-end: C1/C2 strict runs return evaluator values.

**Phase 2: G-fix (guard ordering)** (~0.5 day)
- [ ] `patternVars` collector + `freeVars`-over-guard helper (model: `freeVarsLambda`, lambda.go:113-150).
- [ ] Fast-path/nested-path arm assembly in `lowerIfChainMatch`; single-arm guarded shape.
- [ ] Unit tests: guarded arm with pattern-var reference lowers to the nested shape (bindings precede the guard's `IfStmt`); control guard stays flat; guard failure lands in `REST`.
- [ ] End-to-end: G1-G5 strict runs return evaluator values.

**Phase 3: Goldens, sweep, coordination, docs** (~1 day)
- [ ] `tests/golden/bytecode/` fixtures: the C/G rows of the symptom table + control shapes, asserting strict output == evaluator output == hand-computed value (near-miss rows for the C conds: `Some(x) :: rest` vs a list whose head is `None` → must take the default arm).
- [ ] Strict-VM sweep of `std/` + `examples/`: zero new EvalOnly, zero output changes (the corpus's guard shapes are switch-path or bare-expression — V-G11/V-G12 — so expectation is no change).
- [ ] Cross-shape goldens for the merge contract: `a :: b :: _ if a > b` (needs this doc + nested sibling) recorded as expected-fail until the sibling lands, then flipped — whichever lands second owns the flip.
- [ ] Run `make test`, `make fmt`, `make lint`, `make check-boundaries` (touches `internal/gen/lower` + `internal/vm` — no new import edges expected).
- [ ] Docs: `docs/LIMITATIONS.md` gains nothing (neither family is listed today — V-Docs); add a reference-page note that guards see their arm's bindings on all execution routes; `changelogs/` entry.

### Files to Modify/Create

**New files:**
- `tests/golden/bytecode/ifchain_tag_guard.ail` (+ spec rows in `golden_test.go`) — the C/G/control matrix, ~50 LOC AILANG

**Modified files:**
- `internal/bytecode/builtin_names.go` — append `"_adt_tag"` to `BuiltinNames` (step 1 of the two-step contract, ~1 LOC)
- `internal/vm/builtins.go` — `builtinADTTag` dispatch entry (step 2, ~15 LOC)
- `internal/vm` builtin unit tests (extend `builtins_*_test.go` or new `builtins_adt_test.go`) — unit tests (~40 LOC)
- `internal/gen/lower/match.go` — cond case rewrite + guard restructure + helpers (~80 LOC net)
- `internal/gen/lower/lower_match_test.go` — new unit tests (~150 LOC)
- `tests/golden/bytecode/golden_test.go` — spec rows (~20 LOC)
- `docs/docs/reference/` match/guard reference note + `changelogs/` entry (~15 LOC)

## Examples

### Example 1: C1 — constructor at the head of a cons (the "#1503" hint)

**Before** (verified V-C1, this session):
```
$ ailang run --quiet --entry m5 --args-json 0 battery.ail                       # f5([Some(3), Some(4)])
ok5:3
$ ailang run --quiet --bytecode --strict-bytecode --entry m5 --args-json 0 battery.ail
Error: bytecode execution failed: vm: vm: BUILTIN_CALL: _record_get: arg 0 must
  be record, got ADT (in battery.f5 at battery.ail:23, ip 11, op BUILTIN_CALL)
```
**After** (all routes agree, `ok5:3`): the arm cond lowers to
`_len(xs) >= 1 && _adt_tag(_list_get(xs, 0)) == "Some"`, the head binds to `_head_0`,
`x` binds via positional `OpGetField` — machinery that all exists today.

### Example 2: G1 — guard over list-pattern bindings

**Before** (verified V-G1): strict → `evaluator-only (compiler: unbound variable "a")`.
**After**: the arm lowers to
`If (_len(xs) == 2) { a := _list_get(xs,0); b := _list_get(xs,1); If (a > b) { "ok14" } else { <next arm> } } else { <next arm> }`
— bindings precede the guard, guard failure falls through to the `_` arm in source order, matching the evaluator's `continue` (eval_patterns.go:82-83).

## Success Criteria

- [ ] AC1: C1 and C2 shapes return evaluator-equal values under `--strict-bytecode` (acceptance: golden rows).
- [ ] AC2: G1-G5 shapes return evaluator-equal values under `--strict-bytecode` (acceptance: golden rows).
- [ ] AC3: control shapes (guard referencing only a parameter; switch-path `Some(x) if x > 3`) remain strict-clean with unchanged output (acceptance: golden rows; regression guards against over-eager nesting).
- [ ] AC4: `--strict-bytecode` on a near-miss C row (list head is `None`, arm wants `Some(x)`) takes the default arm — the tag cond actually rejects (acceptance: golden row).
- [ ] AC5: all existing tests pass unmodified, including every fixture in the Conflict Surface list and both siblings' cited fixtures.
- [ ] AC6: strict-VM sweep of `std/` + `examples/`: zero new EvalOnly functions; zero output changes.
- [ ] AC7: `_adt_tag` rejects non-ADT arguments loudly (unit test).
- [ ] All tests passing (`make test`), `make fmt`/`make lint`/`make check-boundaries` green.
- [ ] Documentation updated (reference note + changelog).

## Testing Strategy

**Unit tests:**
- Lowered-IR assertions: the constructor cond contains `BuiltinCall{"_adt_tag"}` and no `FieldAccess{"Tag"}`; guarded arms with pattern-var references lower to the nested If shape with `VarDecl`s preceding the guard `IfStmt`; non-referencing guards stay flat; guard failure's `Else` is exactly the remaining-arm chain.

**Integration / goldens:**
- `tests/golden/bytecode/`: every row of the symptom table owned here, each with a matching and a near-miss input; assert `run`, `run --bytecode`, and `run --bytecode --strict-bytecode` all print the evaluator's value.

**Regression-surface tests** (one per Conflict Surface "MUST still work" entry):
- `examples/pattern_matching_adt.ail` (switch-path Some/None, guard-free) unchanged; `std/json.ail` `get`/`asString` (final wildcard defaults, switch path) unchanged; the V-G8 control program; `examples/runnable/recursion_quicksort.ail` arms (`[]`, `[x]`, `[a, b]`, `[a, b, c]`, `[p, ...rest]`) unchanged.

**Manual testing:**
- Re-run this doc's Verification Log command set; every C/G row reads "correct" on all three routes.
- Merge-order check with whichever sibling lands first: re-run the cross-shape rows (`a :: b :: _ if a > b`; `Some(x) :: rest` under the nested doc's recursive rewrite) and confirm composition.

## Conflict Surface

**Syntactic/semantic positions touched:**
- `lowerPatternCond`'s `ConstructorPattern` case (`internal/gen/lower/match.go:394-400`) — the cond for constructor sub-patterns in **if-chain** positions only (mixed/non-constructor matches; constructor-only matches route to the switch path and are untouched).
- `lowerIfChainMatch`'s arm assembly (`match.go:303-377`) — where bindings, guard, and body meet; the single-arm, multi-arm, and refutable-last-arm branches.
- The native-builtin contract: `internal/bytecode/builtin_names.go` (`BuiltinNames`, one appended name) + `internal/vm/builtins.go` (`BuiltinTable`, one matching dispatch entry) — the two-step process documented at builtin_names.go:11-14, with `validateBuiltinTables` length-agreement at package init and the ≤ 256-entry index-space assertion at `internal/bytecode/compiler/builtins.go:23-26`.

No parser, lexer, AST, elaborator, type-checker, effect, or VM-instruction changes.

**What other valid constructs already live in those positions (and why they are safe):**

| Position | Existing valid form | Interaction |
|---|---|---|
| Constructor cond, if-chain | `FieldAccess{"Tag"} == "Some"` — **traps the VM today** for every ADT value that reaches it (V-C1/V-C2, mechanism V-C5); the only non-trapping execution is a *record* scrutinee carrying a field literally named `Tag`, where the old code could spuriously match a value the evaluator rejects (matchPattern requires a `TaggedValue` — eval_patterns.go:180-184) — itself a pre-existing divergence this fix removes | Replaced by `_adt_tag`: ADT values compare by constructor name (evaluator parity); non-ADT values fail loudly (`_adt_tag: arg 0 must be ADT`) instead of either trapping (`_record_get`) or spuriously matching. No currently-correct VM behavior can regress: with an ADT the old path never returned, and with a record the old path could only return a value the evaluator would not |
| Guard, if-chain, **no pattern-var reference** | `match xs { [1] if n > 0 => … }` (control, V-G8) — compiles and runs strict today | Fast path preserves this shape byte-for-byte; regression-guarded by AC3 |
| Guard, switch path | `Some(x) if x > 3` — binds args in the case body **before** the guard (switch path is already order-correct; V-G7 control, m15) | Untouched — different code path (`lowerConstructorMatch`); its goldens are regression guards |
| Bug A.2 literal-sub-pattern guards (switch path) | `_lit_<i>` temps + duplicated default in guard-failing `Else` (match.go:142-190) | Untouched; the nested-if duplication technique here is the same landed pattern |
| M-LOWER-FIX if-chain bindings | `armBody` prepends `lowerPatternBindings` (match.go:303-312) | Preserved verbatim in both the fast path (same position) and the nested path (bindings hoisted only inside the structural cond, before the guard) |
| Non-tail-position matches | `LowerMatchExpr` narrow IfExpr shape + panic (match.go:52-76) | Untouched; the panic channel remains the loud failure for unbridged shapes |
| Constructor sub-pattern **bindings** in if-chains | `Some(x) :: rest` binds via `_head_<i>` + `FieldAccess "_j"` → `OpGetField` (match.go:505-524; works on `TagADT`, V-C6) | Untouched — only the cond above them changes |
| Records with a literal `"Tag"` field | a record pattern/field genuinely named `Tag` | The C-fix *removes* the only place the lowering fabricated a `FieldAccess{"Tag"}`; genuine `Tag` fields flow through normal record field access, which never used this case |

**Disambiguation:** purely structural — pattern kind, sub-pattern position, guard free-variable analysis over the elaborated Core. No token-level ambiguity anywhere.

**Programs that MUST still work (regression fixtures):**
1. `examples/pattern_matching_adt.ail` — Some/None switch-path match (guard-free).
2. `std/json.ail` — `get`, `asString` (final wildcard defaults; switch path).
3. `examples/runnable/recursion_quicksort.ail` — every list arm (`[]`, `[x]`, `[a, b]`, `[a, b, c]`, `[p, ...rest]`).
4. `examples/runnable/list_pattern_cons.ail` — one-level cons arms incl. the constructor-element binding (`[("a", 1), …]`).
5. The V-G8 control program and `m15` (`Some(x) if x > 3`, switch path) — guards that already work.

**What deliberately changes (intentional incompatibilities):**
- Guarded if-chain arms whose guards reference their own pattern bindings change lowered-IR shape (nested If) and grow by one copy of the remaining-arm chain per such arm; disassembly/golden diffs for those shapes are expected — they were EvalOnly (never disassembled) before, so no working output changes.
- ADT sub-pattern conds in if-chains change from a runtime VM trap to a correct tag comparison; consumers diffing disassembly for previously-crashing code will see the first-ever successful compile of these shapes.

## Verification Log

All claims verified first-party in this session. Binary: `ailang v0.51.0` (`b99dd25c22dc7bf1bd5c3ace8dfab14a46aa3f7d-dirty`, `/usr/local/bin/ailang` — the reported build); repo HEAD `78b26785` on branch `coordinator/task-982a4d4f` (working tree clean at start; relevant files read at HEAD and consistent with observed behavior; this sandbox has no Go toolchain — all Go claims are source reads, all behavior claims are binary runs). Repro programs live under `/tmp/repro/` (battery.ail, battery2.ail–battery5.ail, varmatch.ail, guards_mod.ail, gctl.ail, semshape.ail, bare.ail).

| # | Claim | Instrument | Verdict |
|---|-------|-----------|---------|
| V-A1 | **Reported repro re-confirmed on the reported binary**: `a :: b :: _` → eval `ok`, `--bytecode` `ok` (bridge), `--strict-bytecode` `evaluator-only (compiler: unbound variable "b")` | `ailang run` ×3 modes on the task's exact `nested.ail` shape (battery.ail m1) | Confirmed — this is row A1 of the nested-pattern sibling (its V1, on v0.47.1); not re-designed here |
| V-A2 | Nested-cons variants: exact tail (`a :: b :: c`), wildcard head (`_ :: b :: _`), depth-3 (`a :: b :: c :: _`) — all `unbound variable "b"` under strict | battery.ail m2/m13/m7, strict runs | Confirmed (sibling rows A1/A4-adjacent) |
| V-B1 | Sibling B-family re-confirmed on v0.51.0: `[1, 2, …r]` vs `[9,9,9]` → eval `no3`, **plain `--bytecode` AND strict both `ok3` (silent wrong, exit 0)** | battery.ail m3, three modes | Confirmed (sibling row B1; silent-wrong class — sibling scope) |
| V-B2 | Literal guards missing for cons-head literals (`1 :: _` → VM `ok8`, eval `no8`), exact lists (`[1, 2]` → VM `ok12`), record fields (`{x: 1}` → VM `ok17`), tuple elements (`(1, 2)` → VM `ok18`) | battery2.ail m8/m12, battery4.ail m17/m18, eval vs VM | Confirmed (sibling rows B1-B3 family; sibling scope) |
| V-C1 | **C-family crash**: `Some(x) :: _` on `[Some(3), Some(4)]` → eval `ok5:3`, plain `ok5:3`, strict `Error: … BUILTIN_CALL: _record_get: arg 0 must be record, got ADT … ip 11` | battery.ail m5, three modes | Confirmed |
| V-C2 | C-family in exact-list position: `[Some(x)]` → eval `ok20:4`, plain `ok20:4`, strict same `_record_get` crash | battery4.ail m20, three modes | Confirmed |
| V-C3 | Plain `--bytecode` masks the C-crash: `tryRunEntryViaVM` returns `(false, err)` on runtime VM errors and the caller falls back to the evaluator in non-strict mode | read `internal/runner/vm.go:144-151, 196-207` and `internal/runner/entrypoint.go:143-158`; observed plain-mode correct answer on m5/m20 | Confirmed |
| V-C4 | Cond mechanism: `lowerPatternCond`'s `ConstructorPattern` case emits `FieldAccess{Record: scrutinee, Field: "Tag"} == LitString{Name}` | read `internal/gen/lower/match.go:394-400` | Confirmed |
| V-C5 | `"Tag"` cannot resolve positionally: `compileFieldAccess` only treats `"_N"`-prefixed fields as positional indices; everything else goes through `lookupFieldIndex` and, on failure, `_record_get` by name | read `internal/bytecode/compiler/collections.go:221-265` | Confirmed |
| V-C6 | VM `_record_get` rejects non-record values (`arg 0 must be record, got ADT`); ADT values are `TagADT`; the **binding** side works on ADTs: `OpGetField` handles `TagRecord`/`TagADT`/`TagTuple` positionally | read `internal/vm/builtins.go:171-174`, `internal/vm/vm.go:476-503`; m5's crash names `_record_get` while its binding path (`FieldAccess "_0"`) compiles to `OpGetField` — the crash is the cond's tag access | Confirmed |
| V-C7 | NEGATIVE: no tag-name builtin exists in the tables (no `_adt_tag`/tag-name entry); the name `_adt_tag` is unallocated; adding a native builtin is a documented two-step contract (`bytecode.BuiltinNames` append + matching `vm.BuiltinTable` entry, length-checked at package init) | `grep -n "_len\|_list_get\|_list_tail\|_record_get" internal/bytecode/builtin_names.go` (rows :18-22) and `internal/vm/builtins.go` (:52-55, same order); `grep -rn "_adt_tag\|AdtTag" internal/` → no hits; read `internal/bytecode/builtin_names.go:1-20` (the contract comment: two-step process + `validateBuiltinTables`) | Confirmed |
| V-C8 | VM ADT value carries the constructor **name**: `ADTObj{Tag int (per-type-local ordinal), Ctor string ("Some"), Fields}`; the evaluator matches constructor patterns by `CtorName` string equality | read `internal/bytecode/value.go:168-177, 290`, `internal/eval/eval_patterns.go:179-189` | Confirmed |
| V-C9 | **Sibling dependency (C)**: the nested-pattern doc's case 6 specifies the if-chain constructor cond as `FieldAccess{scrutinee, "Tag"} == Name` — the crashing mechanism — and its Verification Log contains no ADT-sub-pattern-in-if-chain row (its A3 is switch-path) | read `design_docs/planned/v0_49_1/m-bytecode-nested-pattern-lowering.md` Solution Design case 6 + Verification Log V1-V24 | Confirmed |
| V-C10 | **Sibling dependency (G)**: the same doc states guards "continue to AND onto the arm cond, as today"; its primary goal is evaluator parity at arbitrary nesting | read same doc, Solution Design tail + Goals | Confirmed |
| V-G1 | **G-family**: `match xs { [a, b] if a > b => …, _ => … }` → eval `ok14`, plain `ok14` (bridge), strict `evaluator-only (compiler: unbound variable "a")` | battery3.ail m14, three modes | Confirmed |
| V-G2 | Guard + nested cons: `a :: b :: _ if a > b` → strict `unbound variable "a"` (compounds with sibling A1 bindings) | battery2.ail m10, strict run | Confirmed |
| V-G3 | Guarded var arm in a **mixed** (if-chain) match: `match s { "a" => "A", other if other == "b" => "B", _ => "D" }` → strict `unbound variable "other"` | battery5.ail m21, three modes | Confirmed — this is the exact shape the var-pattern sibling's V4 fix routes to the if-chain |
| V-G4 | Guard over record-pattern binding: `{x: v} if v > 3` → strict `unbound variable "v"` | battery5.ail m22 | Confirmed |
| V-G5 | Guard over tuple-pattern bindings: `(a, b) if a < b` → strict `unbound variable "a"` | battery5.ail m23 | Confirmed |
| V-G6 | Guard over flat cons bindings: `x :: rest if x > 5` → strict `unbound variable "x"` | battery5.ail m24 | Confirmed |
| V-G7 | Mechanism: `lowerIfChainMatch` ANDs `arm.Guard` into the arm **cond** (match.go:355-366) while bindings are `VarDecl`s inside `armBody` (match.go:303-312) — cond evaluated before any binding exists; switch-path guards are order-correct (case-body bindings precede the guard, match.go:142-190) | read `internal/gen/lower/match.go` in full; control m15 (`Some(x) if x > 3`) strict `ok15:9` | Confirmed |
| V-G8 | Control: if-chain guard referencing only a **parameter** (not a pattern binding) compiles and runs strict today | gctl.ail `match xs { [a] if n > 0 => "y:${a}", _ => "n" }`, strict `y:7` | Confirmed — guards per se work; only the ordering is broken |
| V-G9 | Evaluator reference semantics: `matchPattern` → bindings pushed to a child env → guard evaluated **with** bindings → false ⇒ `continue` to next arm | read `internal/eval/eval_patterns.go:47-83` | Confirmed |
| V-G10 | **Sibling dependency (G)**: the var-pattern doc's V4 fix promises "the misplaced and guarded arms lower to an ordered if-chain with the guard ANDed in" and AC2 requires the evaluator's value for the guarded shape under strict | read `design_docs/planned/v0_51_2/m-vm-var-pattern-default-arm.md` Example 2 + Success Criteria | Confirmed |
| V-G11 | Demand: the repo's own examples teach guard-over-bindings (`match (10, 20) { (x, y) if x > y => … }`) | read `examples/runnable/guards_basic.ail:29-33` | Confirmed |
| V-G12 | guards_basic.ail never reaches the VM: bare-expression files bypass the entry-based VM dispatch (strict prints the evaluator's answer, no VM error) — so the repo example is latent, not active, demand | `ailang run --quiet --bytecode --strict-bytecode bare.ail` (`match 41 { x => x + 1 }`) → `42`, no error; read `internal/runner/entrypoint.go:143-148` (VM dispatch is per named entry) | Confirmed |
| V-G13 | A free-variable walker over stmt IR exists to model the guard fast-path helper on | read `internal/bytecode/compiler/lambda.go:113-150` (`freeVarsLambda` + `freeVarVisitor`) | Confirmed |
| V-D1 | **Var-sibling scope, evidence**: all-var-arm matches (`match n { x => x + 1 }`) route vacuously through `allConstructorPatterns` to the switch path and fail with `unknown ADT "" in switch` (zero cases, empty ADTName; `inferADTFromCases` returns false on zero cases) | varmatch.ail m_v1/m_v2 strict runs; read `internal/gen/lower/match.go:77-88`, `internal/bytecode/compiler/switch.go:26-36, 151-158` | Confirmed — fixed by that doc's eligibility gate (all-var matches route to the if-chain); not re-designed here |
| V-D2 | The var-sibling's V23 sub-claim "std/sem.ail:378 … correct [today]" is refuted: the identical shape (single-arm var match on a string scrutinee, in a module) is `evaluator-only (compiler: unknown ADT "" in switch)` today | semshape.ail strict run; `std/sem.ail:374-381` read (the inner `match _bytes_to_string(bytes) { json_str => … }`) | Refuted-today / fixed-by-their-gate — coordination note, their AC6 sweep baseline should expect this row to flip from EvalOnly to compiled when their gate lands |
| V-Docs | NEGATIVE: neither family appears in `docs/LIMITATIONS.md` pattern entries; no planned/implemented doc covers ADT-sub-pattern conds or guard ordering (searched by full-text grep + reading both siblings in full + the parity-harness lane doc) | `grep -rln "guard" design_docs/planned/` (8 hits, all unrelated: mission/handover docs + the nested sibling); `grep -rln "_adt_tag\|OpGetTag\|FieldAccess{.*Tag" design_docs/` → none; read `m-bytecode-vm-parity-bugs.md` (harness scope, not lowering fixes); `ls docs/LIMITATIONS.md` + pattern entries :23-45 (per sibling doc V25) | Confirmed |
| V-Ver | The reported "issue #1473 / #1503" references are external GitHub issues not resolvable in this sandbox (no `gh` route configured); the doc cites them as the task did and distinguishes #1503's shape first-party (V-C1: a VM crash, not an unbound variable) | `grep -rn "1473\|1503" changelogs/ design_docs/ docs/` → only unrelated line-number coincidences | Noted (external refs unverified in-repo; the #1503 *shape* is verified V-C1) |

**Note on frequency claims:** no eval pass-rate claim is made. The corpus demand evidence is: the task report's external workaround (stapledons-godot), the repo's own `guards_basic.ail` (latent, V-G11/V-G12), and the two sibling docs' external sprint reports (stapledons-godot `sim/core.ail`).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Merge collision with both siblings** — all three docs modify `internal/gen/lower/match.go` (this doc: `lowerPatternCond` case + `lowerIfChainMatch` arm assembly; nested sibling: recursive cond/bindings rewrite; var sibling: eligibility gate + default binding + depth-1 constructor bindings) | High | Merge-order contract (Related Documents): (1) this doc's two fixes are independently landable and none of the three rewrites the others' landing zone mechanically; (2) if the nested sibling lands first, its case 6 **must adopt** `_adt_tag` (V-C9) and its guard text must defer to this doc's ordering (V-C10); (3) if this doc lands first, the siblings generalize on top — the depth-1 constructor cond/bindings and the nested arm assembly are exactly their starting points; (4) the var sibling's V4/AC2 requires this doc's G-fix in either order (V-G10) — its unit test (c) will fail without it, which is the coordination tripwire |
| Guard-fail duplication grows lowered IR (per referencing-guard arm, one extra copy of the remaining chain; worst case exponential in guarded-arm count) | Med | Free-variable fast path keeps non-referencing guards flat (the corpus's guards); matrix tests include a 3-guarded-arm case; strength-reduction deferred (Deferred Decisions); precedent: Bug A.2's landed default duplication |
| Register pressure in nested arms on the 255-register cap | Med | Regalloc frees temps per block; goldens include a depth-3 nesting case (sibling's matrix) + this doc's 3-guard case; `ValidatePrototype` at compile time catches overflow (compiler.go:135-148) |
| `_adt_tag` on a non-ADT value becomes a loud runtime error where `FieldAccess{"Tag"}` previously also errored — but on a *different* message | Low | Both trap; the new message names the construct (`_adt_tag: arg 0 must be ADT`) — strictly better diagnostics; AC7 pins it |
| A guard whose free variables include a name *shadowing* a pattern binding of a **later** arm (fast-path misclassification) | Med | The fast path intersects the guard's free vars with **that arm's** patternVars only; a shadowed later-arm binding is irrelevant because the earlier arm's cond/body never references it — unit test pins the shadowing case |

## Related Documents

**Implemented (may inform design):**
- [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) — landed the if-chain `armBody` binding pattern and the one-level cons-head constructor binding this doc's G-fix repositions; its test file is a regression fixture.
- [m-bytecode-vm.md](../implemented/v0_11_0/m-bytecode-vm.md) — VM architecture, the EvalOnly/bridge mechanism, and the builtin-table contract.
- [m-dx20-wildcard-pattern-inference.md](../implemented/v0_6_1/m-dx20-wildcard-pattern-inference.md) — `_` vs named binding at the elaborator.

**Planned (check for overlap) — split-of-scope contract:**
- [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) — SIBLING, owns the **reported** nested-cons bug (its A1) and the B-family silent-wrong literal guards. **This doc does not duplicate it.** This doc corrects its case 6 (adopt `_adt_tag`, V-C9) and its guard-ordering assumption (V-C10); its recursive rewrite generalizes this doc's depth-1 cond/bindings. Whichever lands second re-verifies the composed shapes (`a :: b :: _ if a > b`, `Some(x) :: rest`).
- [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md) — SIBLING, same sprint folder, owns var/wildcard default arms in constructor matches (and, via its eligibility gate, the all-var-arm dispatch — V-D1) plus golang/emitgo parity. Its V4/AC2 depends on this doc's G-fix (V-G10); this doc contributes the V-D2 correction to its V23 baseline (sem.ail:378 is EvalOnly today, not correct).
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — SIBLING (landed logic at HEAD): the closed-list `len == n` check; its `Tail == nil ⟺ OpEq` invariant is preserved by this doc's cond changes (which touch only the constructor case).
- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md) — the parent parity-harness lane; this doc's goldens can feed its A2 classification when unparked.

**Related-docs search note:** the create script's neural/SimHash search is unavailable in this runner (Ollama endpoint unreachable; SimHash returns token-collision noise — a 1.00 "match" for an unrelated OpenRouter doc on the doc-name query). The coverage-gate reading above was therefore done by full-text grep of `design_docs/planned/` + `implemented/` for guard/tag mechanisms (V-Docs) and by reading all three siblings in full — not on search snippets.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles; A1/A11 drive this fix
- Evaluator pattern semantics (the fixed point): `internal/eval/eval_patterns.go` (`matchPattern`, the linear arm loop)
- Lowering (the fix site): `internal/gen/lower/match.go`; consumers: `internal/bytecode/compiler/collections.go` (field access realization), `internal/vm/builtins.go` (builtin table), `internal/vm/vm.go` (`OpGetField` value-type dispatch)
- Task report: coordinator task `task-982a4d4f` (reported repro + stapledons-godot workaround `sim/core_test.ail` checkCommittedRefusesAll, AI.2); external issue refs ailang#1473 / #1503 (V-Ver)

## Future Work

- Strength-reduction of the duplicated remaining-arm chain (single `REST` via a hoisted temp or an IR early-exit construct) if goldens show measurable cost — deferred, semantics identical.
- Unify tag identity across the two dispatch paths: the switch path compares per-type-local ordinals (`OpGetTag` + `adtTypes`, with the cross-ADT ambiguity noted at switch.go:141-150), this doc's if-chain path compares constructor names (`_adt_tag`). A single name-based mechanism for both is the natural endpoint; the switch path is performance-relevant and untouched here.
- The nested sibling's exhaustive pattern-matrix goldens become the shared regression corpus for all three sibling fixes once it lands.

## Non-Goals

**Not attempted in this feature:**
- The reported nested-cons bug and the B-family silent-wrong literal guards — [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) scope (verified non-duplicative, V-A1/V-B1/V-B2).
- The var-default-arm family and the all-var-arm dispatch gate — [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md) scope (V-D1).
- Runtime arity checks for constructor sub-patterns (the evaluator checks them at match time; well-typed programs cannot mismatch — the type checker rejects them; adding VM-side checks would be stricter than the evaluator for zero reachable programs).
- Exhaustiveness semantics at runtime: an arm whose structural cond + guard both fail keeps today's inexhaustive fall-through (the evaluator errors `no pattern matched in match expression`, eval_patterns.go:102; the lowered form returns unit) — pre-existing terminal divergence, unchanged in width, documented in Solution Design.
- golang v1 / emitgo-v2 parity for the two families (the var sibling owns that consumer surface for its shapes; the C/G families' emitgo status is unmeasured here — a row for the sprint to check, not a design claim).
- Any parser/elaborator/type-checker change — the surface syntax and elaborated Core are already correct; this is a lowering fix.
- The decision-tree match compiler (`internal/eval/decision_tree.go`) — an evaluator-internal optimization; the lowering parity target is the linear arm loop.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Where the free-variable helper lives (a lower-local `freeVars` modeled on `freeVarsLambda`, vs sharing/exporting the compiler's `freeVarVisitor`) — agent may choose; behavior identical.
- Whether the fast-path analysis runs over the lowered guard expression or the Core guard (before lowering) — agent may choose; must be consistent per arm.
- Exact panic message wording for any shape the new lowering still cannot express — agent may choose (must name the shape, cite this doc, and flow into `LowerError` → EvalOnly, the landed channel).
- Whether the duplicated `REST` chain is emitted once per guarded arm (naive) or shared via a hoisted boolean temp + single copy — agent may choose; naive preferred unless goldens show bloat.
- Test-file layout (extend `lower_match_test.go` vs a new `match_guard_test.go`) — agent may choose.

## Timeline

**Week 1** (~16 hours):
- Phase 1: `_adt_tag` builtin + cond rewrite + unit tests + C goldens (5h)
- Phase 2: guard restructure + fast path + unit tests + G goldens (6h)
- Phase 3: sweep, cross-shape rows, sibling-coordination re-runs, `make test`/`lint`/`check-boundaries`, docs (5h)

**Total: ~2 days across 1 week** (2× the raw ~1-day estimate, per skill guidance)

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

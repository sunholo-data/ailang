# M-TYPE-LIST-ELEMENT-SOUNDNESS: List-Literal Element Types Escape the Type Checker

**Status**: Planned
**Target**: v0.24.0
**Priority**: P0 — type-soundness hole. Invalid programs type-check and fail at runtime; a language "designed for AI synthesis" must not let element-type errors escape to runtime.
**Estimated**: 1–2 days (diagnosis ~0.5d, fix + soundness fixtures ~1d)
**Dependencies**: None. Complements (does not block) [M-AILANG-ERROR-QUALITY](./m-ailang-error-quality-for-llm-iteration.md).

## Problem Statement

A **list literal** whose element type comes from a numeric literal — or from a `List[T]` value bound through an `Option` match (e.g. `std/json` accessors) — is **not unified against the expected element type**. The program type-checks, then fails at runtime in the consuming builtin. This is a real unsoundness: the type checker accepts a program it should reject.

This was found as the root cause of the `json_parse` nightly benchmark failing on `opencode-qwen3-5-35b-a3b-mxfp8` (2026-06-08, the only benchmark failing both trials). The model used `getObject(item, "item")` — which returns `Option[Json]` (the *object* node) — then consed the bound `Json` value onto a `[string]` accumulator and passed it to `join`. The correct accessor is `getString`; the mistake is a genuine type error. Instead of a compile-time `Json vs string`, the agent got a cryptic runtime `_str_join: expected string, got tagged value` after 24 turns and gave up.

**Verified minimal repros** (all run with the v0.24.2 binary, `ailang run --caps IO --entry main`):

```ailang
-- (1) numeric-literal list into [string] — TYPE-CHECKS, runtime: "expected string, got int"
join(",", [42])

-- (2) explicit annotation does not constrain the literal — TYPE-CHECKS
let xs: [string] = [42] in ...

-- (3) the json_parse shape — Json into [string] — TYPE-CHECKS, runtime: "got tagged value"
match getObject(j, "k") { Some(name) => join(",", [name]), None => ... }
```

**What this hole is NOT** — the type system correctly rejects every neighbouring case, which is how we know the leak is narrow and specific:

| case | expected | actual | verdict |
|---|---|---|---|
| scalar `needStr(42)` (int → `string` param) | reject | ✅ `No instance for Num[string]` | sound |
| heterogeneous literal `["a", 42]` | reject | ✅ `No instance for Num[string]` | sound |
| `42 :: acc` cons onto `[string]` | reject | ✅ `No instance for Num[string]` | sound |
| concrete `ints():[int]` → `[string]` param | reject | ✅ `cannot unify string vs int` | sound |
| **`[42]` literal → `[string]`** | reject | ❌ **accepted → runtime fail** | **HOLE** |
| **`[name:Json]` / `List[Json]` → `[string]`** | reject | ❌ **accepted → runtime fail** | **HOLE** |

The contrast is the diagnosis: cons, scalars, heterogeneous literals, and concrete function returns all unify element types correctly. The leak is in **homogeneous list-literal element unification** (the numeric literal's `Num` constraint is dropped when the literal's element tyvar is unified with a concrete type) and in the **`List[Json]`-from-`Option`-match** position (a `List[T]`-constructor value not being element-checked against the `[T]` sugar of the consumer).

**Impact:**
- **Soundness**: the headline property of an AI-synthesis language — "if it type-checks, the type error isn't there" — is false in these positions.
- **Eval cost**: directly burns whole agent runs. `json_parse` cost 24 turns + a failed trial because a 1-line compile error was deferred to an opaque runtime message. This is the single most persistent local-model failure (3 nights running).
- **Blast radius for the fix is small** (the neighbours already work), so this is a high-value, low-risk correctness fix.

## Goals

**Primary Goal:** A list whose element type does not match the expected element type is rejected at **compile time**, with an AILANG-level `cannot unify <T> vs <U>` message anchored to the offending element.

**Success Metrics:**
- All three repros above produce a compile-time type error (not a runtime failure).
- The six "sound" neighbour cases still behave identically (no new false rejections).
- `make verify-examples` stays green (no currently-valid example regresses).
- `json_parse` re-run shows a *compile* error the agent can act on (recovery-rate > 0% for this error code), not a runtime `tagged value`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at unification (recurse + re-check constraints on list element) vs at list-literal elaboration | Determines whether the fix is general (covers `List[T]` positions too) or literal-specific | compiler | design | med |
| Whether `List[T]` (named constructor) and `[T]` (sugar) are one type or two at unification time | The Json case implicates a possible sugar/constructor mismatch; if they're distinct, that's a second leak | compiler | design | med |
| Accept that previously-"compiling" unsound programs now fail to compile (intentional incompatibility) | Tightening a checker can reject dead-but-accepted code | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Confirm the root cause locus (unifier constraint-drop vs list-literal elaboration vs `List[T]`/`[T]` mismatch) via a failing unit test in `internal/types`.
- [ ] Confirm the intentional-incompatibility is acceptable (audit `make verify-examples` + recent eval corpora for any program that relied on the hole).

## Solution Design

### Overview

Make list-element unification **total**: when a list type `[α]` (or `List[α]`) is unified with `[τ]`, the element types must unify *and any deferred constraints on `α` must be re-validated against `τ`*. The numeric case leaks because `α`'s `Num α` constraint is silently dropped when `α := string`; the Json case leaks because the element unification is apparently skipped for the `List[T]`-from-`Option` shape.

### Architecture

**Root-cause surface (to be confirmed in Phase 1):**
1. **`internal/types/unify.go`** — list/constructor unification: does `unify([α], [τ])` recurse into element types AND propagate/re-check `α`'s constraint set when `α` is bound to `τ`? The scalar path does (TEST E); the list path apparently doesn't (the `Num` constraint is dropped).
2. **List-literal elaboration** (`internal/elaborate/`) — how a list literal's element type is assigned and whether its element constraints are attached to the right tyvar before the literal flows into a typed position.
3. **`List[T]` vs `[T]` sugar** (`internal/types`, parser/elaborate desugaring) — whether `Option[List[Json]]` (from `asArray`/`getObject`) and `[string]` (consumer) reduce to the same constructor before unification. If not, that is a distinct soundness leak in the same family.

### Implementation Plan

**Phase 1: Localize (~0.5d)**
- [ ] Add failing unit tests in `internal/types` reproducing (1), (2), (3) at the type-checker level (no runtime).
- [ ] Determine which of the three loci above drops the check; write the test that pins it.

**Phase 2: Fix (~0.5d)**
- [ ] Make list-element unification recurse + re-validate constraints (so `Num[string]` fails like the scalar path).
- [ ] If implicated, normalize `List[T]` and `[T]` to one representation before unification.
- [ ] Ensure the emitted error is AILANG-level and anchored to the element (`cannot unify Json vs string at <file:line:col>`), not a Go-internal leak.

**Phase 3: Lock it down (~0.5d)**
- [ ] Add the six "sound neighbour" cases as regression fixtures (they must keep passing).
- [ ] Add the three holes as must-reject fixtures.
- [ ] `make verify-examples`; audit/fix any example that relied on the hole.

### Files to Modify/Create

**Modified files:**
- `internal/types/unify.go` — total list-element unification + constraint re-check (~30–80 LOC)
- `internal/elaborate/*.go` — list-literal element-type/constraint attachment, if Phase 1 points here (~0–40 LOC)
- `internal/types/*_test.go` — soundness fixtures (must-reject + must-accept) (~120 LOC)

**Possibly:**
- `internal/types` / parser desugar — `List[T]`↔`[T]` normalization if it's a second leak.

Does **not** touch runtime/codegen/eval/stdlib semantics — this only tightens the type checker.

## Examples

**Before** (accepted, fails at runtime):
```
$ ailang run examples/typehole.ail
✓ Running ...
Error: execution failed: _str_join: element 0 - expected string, got tagged value
```

**After** (rejected at compile time, actionable):
```
$ ailang check examples/typehole.ail
Error: type error at solution.ail:18:38:
  cannot unify element type `Json` with `string`
  `getObject` returns Option[Json]; did you mean `getString` (returns Option[string])?
```

## Conflict Surface

(Required per CLAUDE.md for `internal/types/` + `internal/elaborate/` changes.)

1. **Positions this change extends:** unification of list types where one side's element type is an unsolved/constrained tyvar or a `List[T]` constructor.
2. **Other valid constructs in those positions:** empty list `[]` (element fully polymorphic — must stay polymorphic), homogeneous typed lists, nested lists `[[int]]`, polymorphic results of `map`/`filter`, `List[T]` returned by stdlib, and numeric-literal lists that legitimately default to `int`/`float` (`[1,2,3] : [int]` must still work).
3. **Disambiguation:** none syntactic — this is purely making an existing semantic check total. The only behavioural change is that an element-type mismatch that previously slipped now fails unification.
4. **Existing programs that MUST still compile (regression fixtures):**
   - `let xs = [1, 2, 3] in sum(xs)` — numeric list defaults to `[int]` ✅
   - `let xs: [string] = ["a", "b"]` — valid typed literal ✅
   - `map(\x. x + 1, [1, 2, 3])` — polymorphic over a literal ✅
   - `let empty: [string] = []` — empty list unifies with any element type ✅
   - `[[1], [2, 3]] : [[int]]` — nested ✅
5. **Deliberate incompatibility:** programs that *only* compiled because of the hole (a non-string element reaching a `[string]` position) now fail to compile. This is the intended fix — those programs always failed at runtime; they now fail earlier and legibly. Audit gate: `make verify-examples` + recent eval corpora.

**The honest answer is not "no conflicts":** the empty-list and numeric-defaulting cases are the real risk — a naïve "elements must equal" change could break `[] : [string]` or `[1,2,3] : [int]`. The fixtures above exist specifically to prove those still work.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Invalid programs are rejected deterministically at compile time instead of failing at a runtime-dependent location |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +2 | The core win: an element-type error is caught by local type-checking instead of escaping to runtime — exactly what bounded verification promises |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +2 | The AI gets an actionable compile error (`Json vs string` at a line) instead of an opaque runtime `tagged value` — saving entire failed agent runs |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Fewer wasted agent iterations/tokens on programs that cannot run |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +2 | Converts an unstructured runtime builtin failure into a typed, located compile error |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +8** → **Decision: Proceed.** No −1 on A1/A3/A4/A7.

### Hard Violation Check
- [x] A1 (Determinism): no implicit nondeterminism — strictly removes a nondeterministic-location failure
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): strictly improves machine-analyzability

## Success Criteria

- [ ] Repros (1), (2), (3) produce compile-time type errors with element-anchored, AILANG-level messages
- [ ] All six "sound neighbour" cases unchanged
- [ ] Regression fixtures (empty list, numeric default, nested, polymorphic map) pass
- [ ] `make verify-examples` green
- [ ] `json_parse` re-run on the rig shows a compile error (recovery-rate > 0%), not a runtime `tagged value`
- [ ] All tests passing; CHANGELOG updated

## Testing Strategy

**Unit tests (`internal/types`):** the three must-reject holes + the five must-accept fixtures, at the type-checker boundary (assert compile error / no error, not runtime).
**Integration:** `make verify-examples`; a targeted `ailang check` over the soundness fixtures.
**Eval:** re-run `json_parse` (and the smoke tier) on `opencode-qwen3-5-35b-a3b-mxfp8`; confirm the failure mode shifts from runtime to a compile error the agent can recover from.

## Deferred Decisions

- Exact wording of the "did you mean `getString`" hint — **agent may choose** (coordinate with [M-AILANG-ERROR-QUALITY](./m-ailang-error-quality-for-llm-iteration.md)'s rubric).
- Whether to also normalize `List[T]`/`[T]` everywhere or only at unification — **agent may choose** based on Phase 1 findings.

## Non-Goals

- Improving *runtime* error messages for the cases that legitimately remain runtime errors — that is [M-AILANG-ERROR-QUALITY](./m-ailang-error-quality-for-llm-iteration.md)'s domain. This doc's position is that these specific cases should never *reach* runtime.
- A full type-system rewrite — see [m-type-v2-migration](../v1_0_0/m-type-v2-migration-sprint-plan.md). This is the targeted near-term soundness fix.
- The generalized "undefined variable → suggest stdlib import" resolver (the `asNumber` half of `json_parse`) — that is a separate error-quality item.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Over-tightening breaks empty-list / numeric-defaulting | High | Explicit must-accept regression fixtures (Conflict Surface §4) written before the fix |
| The Json case is a *different* leak (`List[T]` vs `[T]`) than the numeric case | Med | Phase 1 localizes each independently; fix both or scope-split with a follow-up doc |
| A currently-passing example/eval relied on the hole | Low | `make verify-examples` + recent-corpus audit in the Design Freeze gate |

## Related Documents

- [M-AILANG-ERROR-QUALITY](./m-ailang-error-quality-for-llm-iteration.md) — companion: makes *legitimate* compile/runtime errors actionable. This doc removes a class of errors from runtime entirely; that doc improves the wording of the ones that remain. The `tagged value` runtime message is the symptom; the real fix is here.
- [M-EVAL-PROMPT-DELIVERY](../../implemented/v0_24_0/m-eval-prompt-delivery.md) — surfaced `json_parse` as the persistent local-model gap whose runtime failure this doc traces to a typing hole.
- [M-TYPE-CONSTRAINTS](./m-type-constraints.md) — adjacent type-system item (P3, recent-data-downgraded); distinct topic (explicit `Ord` comparators, not list-element soundness).
- [m-type-v2-migration-sprint-plan](../v1_0_0/m-type-v2-migration-sprint-plan.md) — long-horizon type-system rewrite that may subsume this; this doc is the v0.24.0 near-term fix.

## References

- [Design Axioms](/docs/references/axioms)
- Root-cause surface: `internal/types/unify.go`, `internal/elaborate/`, `List[T]`/`[T]` desugaring
- Motivating data: `/tmp/nightly_eval_20260608_rag_on` (`json_parse` both-trial failure), reproduced minimally 2026-06-08

---

**Document created**: 2026-06-08
**Last updated**: 2026-06-08

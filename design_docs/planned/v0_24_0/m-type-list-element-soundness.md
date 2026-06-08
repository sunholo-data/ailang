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

## Prior Attempts & Why This Is Hard (recurring architectural debt)

This is **not** a fresh bug — three prior sprints hit the same wall and deferred the deep fix. Anyone picking this up MUST read this first to avoid a fourth re-tread.

- **DX-17** (v0.6, implemented): normalized the `TList` vs `TApp("list")` split (`[T]` → `TApp("list", T)`). Left **capital `List[T]`** unaddressed — that's why `AsList` (`helpers.go:120`) is lowercase-only, the lead for our `List[Json]`-from-`Option` variant.
- **numeric_coercion** (`design_docs/implemented/v0_3/20251013_numeric_coercion.md`): shipped scalar numeric defaulting but **explicitly deferred "❌ Array/List coercions"** — our exact gap, parked since v0.3.
- **M-TYPECHECK-NO-AUTO-UNWRAP-RESULT** (`design_docs/implemented/v0_20_0/`): the closest precedent. It needed to reject `let r = f(); r.field` where `r` is still a `TVar` at infer time. Its own words: *"the constraint solver substitutes the receiver TVar … rather than rejecting the unification, losing the type information from CoreTI before any post-pass can verify it."* It **shipped only the partial gate** (cases concrete at infer time) and **deferred the deep fix — which never landed** (`internal/pipeline/tagged_union_field_access_test.go` is still `t.Skip`'d). It enumerated three options, none implemented:
  1. **Stricter constraint solver** — reject the unsound chain at solve time (~300 LOC, real regression risk to programs relying on permissive unification).
  2. **Parallel source-type map** — record the pre-unification type separately (~80 LOC, parallel infra).
  3. **Hook the constraint solver** — detect the `α ~ X` then `α ~ Y` chain at solve time (~200 LOC).

**The unifying root cause:** AILANG emits soundness-relevant facts at *infer* time, but the types are still `TVar`s until the solver runs; permissive unification then resolves the violation away before any pass validates it. Our case is one instance: the literal's `Num[α1]` is orphaned and defaults to `int` (full trace in the sprint plan).

**Systemic-fix opportunity (per CLAUDE.md "audit before patching"):** rather than a fourth point-patch, the durable fix is likely a **post-inference soundness pass** that, after the solver runs and `applySubstitutionToTyped` resolves types, re-validates the typed AST — catching orphaned class constraints (`Num[string]`), element-type mismatches, AND the still-open tagged-union `.field` case in one mechanism (finally unskipping `tagged_union_field_access_test.go`). That is a larger, deliberate architectural sprint than the 1–2-day point fix scoped below. The choice **narrow point-patch vs unified post-inference pass** is the load-bearing decision for this sprint.

## Implementation Outcome (2026-06-08) — supersedes the Solution Design below

A parallel investigation + 3-way fix competition (worktree-isolated, adversarially verified) **falsified the post-inference-pass plan below**: live tracing showed the offending list resolves to a fully-homogeneous `[int]` on the typed AST — the `[string]` expectation lives on a decoupled ANF/let-bound sibling, so no typed-AST walk can see the conflict. The real fix is **upstream**:

- **Root cause:** `Scheme.Instantiate` (`internal/types/types_v2.go`) silently dropped `s.Constraints`. A let-bound `[42]` generalizes to `∀a. Num a ⇒ [a]`; instantiation re-created only `[a']`, never re-emitting `Num a'`. So `[a']~[string]` set `a':=string` with no surviving obligation and `Num[string]` was never checked. (The scalar `needStr(42)` works precisely because there is no generalization/instantiation to lose the constraint.)
- **Fix (landed):** `Scheme.InstantiateWithConstraints` re-emits the freshened class constraints at the use site (`inferVar`), so the *existing* ground-constraint resolver rejects `Num[string]`. Plus a second, independent bug this exposed: `let x: T` annotations were dropped during elaboration entirely (`let x: int = "hello"` compiled) — now recorded (`normalizeLet`/`normalizeBlock`) and enforced (`inferLet` emits `valueType ~ annot`).
- **Closed + verified (zero false positives, baseline `make verify-examples` = 181/5):** numeric-list `[42]→[string]`, `let`-annotation mismatch, direct ADT `[Red]→[string]`.
- **Round 2 (CLOSED — `json_parse` shape + a broader bug class):** `checkPattern`'s ConstructorPattern arm bound each pattern variable to an orphaned fresh type var (never tied to the constructor's field type or the scrutinee), so *any* destructured value (`Some(b)` from `Option[Box]`, even monomorphic `Wrap(n)`) lost its concrete type and could be misused anywhere — not just lists. Fix (`typechecker_patterns.go`): instantiate the constructor's factory scheme (`$adt.make_<T>_<C>`), constrain `scrutinee ~ factory.Return`, bind each arg to the real field type. **Adversarially verified: zero false positives across 140+ programs; A/B-confirmed no corpus/runtime regression.** Tests: `TestListElementSoundness_ConstructorPatternBind` (+ `_NoOvertighten`).
- **Round 3 (open follow-up):** *compound-field* shape `type Bag[a] = Bag([a])` with an annotated scrutinee — `passthru(b: Bag[int]) -> [string] { Bag(xs) => xs }` type-checks. Root cause: the ADT type-arg vars are instantiated *twice* (the M-DX25.4 `adtType` binding ~lines 178-198 AND `factory.Return`) instead of *shared*, so the inner compound-field element var is left orphaned. Fix recipe (from adversarial verdict): instantiate the ADT type-arg vars ONCE and share them between the `adtType` binding and `factory.Return`. Verified pre-existing (no-worse-than-before).
- **Separate ticket (not this milestone):** `join`/`concat`/`++` over an unannotated-param wrapper over-generalizes to `∀a`, dropping the `[string]` element obligation — reproducible with **no constructor pattern at all**, so it's a distinct generalization-constraint-survival hole.
- **Other residual:** the check fires on the canonical module path; the MOD010 *relaxed* path can still bypass it via the CLI (affects baseline too) — separate follow-up + a CLI-level regression test.

## Solution Design

**Chosen approach (2026-06-08): unified post-inference soundness pass.** Per the Prior Attempts analysis, a fourth point-patch at infer/unify time fights the same losing battle (the type is still a `TVar`; permissive unification resolves the violation away). Instead we add a single validation pass that runs *after* the constraint solver and substitution, when types are concrete — closing the list-element hole, the numeric/list-coercion gap, AND the still-open tagged-union `.field` case in one mechanism.

### Overview

After `InferWithConstraints` calls `tc.applySubstitutionToTyped(sub, typedNode)` (`typechecker_core.go:~439`), the typed AST carries **resolved** types (TVars substituted / defaulted). Add `tc.validatePostInferenceSoundness(typedNode)` immediately after, walking the typed AST (cycle-safe per `internal/types/traverse`) and enforcing structural invariants that cannot be checked at infer time:

1. **List/Array element homogeneity** — for each `TypedList`/`TypedArray`, every element's resolved type must unify with the node's resolved element type. Catches `[42]:[string]` (`int` vs `string`) and `[Json]→[string]` directly, *regardless of the orphaned-constraint subtlety* (the trace bug) — we compare resolved types, not constraint vars.
2. **Tagged-union field access** — for each `TypedRecordAccess`, the receiver's resolved type must not be a tagged union (reuse `isTaggedUnion` from M-TYPECHECK-NO-AUTO-UNWRAP-RESULT). Catches the `let r = f(); r.field` shape that v0.20.0 deferred → unskips `tagged_union_field_access_test.go`.
3. Extensible: future structural invariants slot into the same walk.

Errors are AILANG-level and span-anchored (the typed node carries its `Span`). This is the "post-inference walk applying substitutions" that the v0.20.0 doc named as the real fix but never built.

### Architecture

- **Hook point:** `internal/types/typechecker_core.go`, in `InferWithConstraints`, right after `applySubstitutionToTyped`. Gate behind a strict default with an escape hatch (`AILANG_ALLOW_UNSOUND=1`) for one release to de-risk ecosystem migration (mirrors v0.20.0's `--allow-unsafe-field-access`).
- **New file:** `internal/types/soundness_postpass.go` — `validatePostInferenceSoundness(node) error` + per-invariant checkers. Uses `traverse.Walk` (cycle-safe; type-system rule).
- **Resolved-type access:** the typed AST nodes already store their resolved `Type` post-substitution; the checker reads those (no re-inference). For element homogeneity it unifies element resolved-type vs list resolved-element-type using the existing `Unifier` in a throwaway substitution (check-only, discard result).
- **Why this dodges the wall:** the prior attempts failed at *infer* time when types were `TVar`s. Here, after substitution+defaulting, a list that ended up `[string]` with an `int` element is a concrete, detectable mismatch.

### Implementation Plan

**Phase 1: Localize + spec (DONE)**
- [x] Failing compile-time tests + must-accept guardrails (`internal/pipeline/list_element_soundness_test.go`).
- [x] Trace-confirmed root cause + prior-attempts analysis (sprint plan + this doc).

**Phase 2: Post-inference pass skeleton + list invariant (~1d)**
- [ ] `soundness_postpass.go` with the cycle-safe walk + the list/array element-homogeneity checker.
- [ ] Wire it into `InferWithConstraints` after `applySubstitutionToTyped`, behind the `AILANG_ALLOW_UNSOUND` escape hatch (default strict).
- [ ] Unskip `MustReject` (list cases); keep `MustAccept` green.

**Phase 3: Tagged-union invariant (~0.5d)**
- [ ] Add the `TypedRecordAccess` / `isTaggedUnion` checker to the same pass.
- [ ] Unskip `internal/pipeline/tagged_union_field_access_test.go` (the v0.20.0 deferred case).

**Phase 4: Audit sweep + rollout (~1d)**
- [ ] `make verify-examples`; fix/annotate any example that relied on a hole.
- [ ] Run the smoke tier on the rig — confirm `json_parse` (and similar) now fail at *compile* time, recoverable.
- [ ] Full `internal/types` + `internal/pipeline` + `cmd/ailang` suites; CHANGELOG + LIMITATIONS update; document the escape hatch + its removal target.

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

**Prior attempts at the same root cause (READ FIRST — see "Prior Attempts" section):**
- [M-TYPECHECK-NO-AUTO-UNWRAP-RESULT](../../implemented/v0_20_0/m-typecheck-no-auto-unwrap-result.md) — same "soundness check vs TVar-still-unresolved" wall; shipped partial, deferred the deep fix (never landed). Its 3 options apply here verbatim.
- [numeric_coercion](../../implemented/v0_3/20251013_numeric_coercion.md) — explicitly deferred "Array/List coercions" (our gap).
- DX-17 series (v0.6, `changelogs/v0.6-coordinator.md`) — `TList`/`TApp("list")` normalization; left capital `List[T]` (the `AsList` lead).

## References

- [Design Axioms](/docs/references/axioms)
- Root-cause surface: `internal/types/unify.go`, `internal/elaborate/`, `List[T]`/`[T]` desugaring
- Motivating data: `/tmp/nightly_eval_20260608_rag_on` (`json_parse` both-trial failure), reproduced minimally 2026-06-08

---

**Document created**: 2026-06-08
**Last updated**: 2026-06-08

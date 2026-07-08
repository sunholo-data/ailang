# Sprint Plan: M-TYPE-LIST-ELEMENT-SOUNDNESS

**Design doc**: [m-type-list-element-soundness.md](./m-type-list-element-soundness.md)
**Sprint ID**: M-TYPE-LIST-SOUND
**Target**: v0.24.0 · **Priority**: P0 (soundness) · **Risk**: low-med (type-checker change, narrow surface)
**Estimated**: 1–2 days · **Approach**: TDD — failing tests first, then localize, then fix, then lock.

## Goal
Make list-element type mismatches a **compile-time** error. `join(",", [42])` and `Json`-into-`[string]` must be rejected by the type checker, not fail at runtime in `_str_join`.

## Pre-execution findings (Phase 1 head-start)
- `internal/types/unification_types.go:unifyLists` already recurses into element types (`Unify(t1.Element, elem)`). So the leak is **upstream of unifyLists**: either the numeric literal's `Num` constraint never reaches unification (list-literal elaboration), or `List[Json]` (capital, std) vs `[T]` sugar don't both match `AsList` (so the element check is skipped).
- Neighbours are sound (scalar, cons, heterogeneous literal, concrete `[int]` return) → the fix must NOT touch those paths.

## Milestones

### M1 — Localize (TDD red) (~0.4d)
Write failing tests in `internal/types` (and/or `internal/elaborate`) reproducing the two holes at the type-checker boundary:
- `[42]` (Num-literal list) unified against `[string]` → expect a `Num[string]`/unify error.
- `List[Json]` / `[Json]`-from-Option into `[string]` → expect a unify error.
Bisect: does raw `Unify(TList[NumVar], TList[string])` + constraint-solve already error? If yes → hole is in list-literal elaboration. If no → hole is in unification/constraint propagation. Same for the `List`/`[T]` sugar via `AsList`.
**Acceptance**: ≥2 new tests that FAIL on current code, pinning the locus.

### M2 — Fix (TDD green) (~0.5d)
Apply the minimal fix at the localized site (constraint propagation on list-element unify, and/or `List[T]`↔`[T]` normalization in `AsList`). Element-anchored, AILANG-level error message (no Go-internal leak).
**Acceptance**: M1 tests pass; the three design-doc repros (`join(",",[42])`, `let xs:[string]=[42]`, `getObject`→`[string]`) now produce compile errors via `ailang check`.

### M3 — Lock down (~0.5d)
- Add must-accept regression fixtures: empty list `[] : [string]`, `[1,2,3] : [int]`, `map(\x.x+1,[1,2,3])`, nested `[[1],[2,3]]`, typed `["a","b"]`.
- Add must-reject fixtures for the holes.
- `make verify-examples` green; audit/fix any example that relied on the hole.
- Full `go test ./internal/types/... ./internal/elaborate/... ./cmd/ailang/...`; CHANGELOG entry.
**Acceptance**: all green, no new false rejections, examples pass.

## Conflict-surface guardrails (must-accept — these MUST keep compiling)
`[] : [string]` · `[1,2,3] : [int]` (numeric default) · `map(\x.x+1,[1,2,3])` · `[[1],[2,3]] : [[int]]` · `["a","b"] : [string]`

## M2 root cause — trace-confirmed (2026-06-08)

Instrumented `InferWithConstraints` (temporary, reverted) and traced `wantStrings([99])` where `wantStrings : [string] -> string`. Raw constraints for `main`:
```
Class  Num[α1]                                   <- the literal 99
TypeEq list[string] -> string ~ [α2] -> α3       <- inferApp: funcType ~ ([argElem] -> result)
TypeEq α3 ~ string
=> sub: α2 -> string, α3 -> string ;  unsolved: Num[α1]   (α1 NEVER bound)
```
**The literal's `Num`-carrying var (`α1`) is orphaned from the list-element var (`α2`) that gets unified to `string`.** So `Num[α1]` defaults to `int` harmlessly and the `int`-elem-vs-`string`-target mismatch is never checked. (In the must-accept `[1,2,3]:[int]` case the same orphan defaults to `int`, which matches the target — so no harm. That's why only the `[string]`/`[Json]` direction leaks.)

- `inferApp` (`typechecker_functions.go:417`) builds the param constraint from `getType(argNode)` — correct.
- `inferLit` (`typechecker_literals.go:14`) sets the literal's node type AND `Num` to the same `tv` — correct.
- `inferList` (`typechecker_data.go:203`) returns `TList{Element: getType(firstElem)}` — *should* be `[α1]`, but the app constraint shows `[α2]`. So a **freshening/copy decouples the list element var from the literal's constrained var** between `inferList` and `inferApp`. **Exact site not yet pinned** (next pass: add a trace in `inferList` printing the returned element type vs `getType(firstElem)`, and check `getType`/`TypedList` construction for a copy/normalize).

**Fix shape (once the site is pinned):** ensure the list-literal element type IS the literal's constrained var (or add a `TypeEq` linking them) so `[α1] ~ [string]` binds `α1:=string` → `Num[string]` → "No instance" at compile time. Then unskip `MustReject`, keep `MustAccept` green, run `make verify-examples` + full `internal/types`/`pipeline`/`cmd` suites, add CHANGELOG.

## Done = green tests + green examples + the 3 repros reject at compile time + CHANGELOG.

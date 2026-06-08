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

## Done = green tests + green examples + the 3 repros reject at compile time + CHANGELOG.

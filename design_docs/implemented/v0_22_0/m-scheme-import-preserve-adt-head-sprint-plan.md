# Sprint Plan — M-SCHEME-IMPORT-PRESERVE-ADT-HEAD

**Sprint ID**: `M-SCHEME-IMPORT-PRESERVE-ADT-HEAD`
**Target**: v0.22.0
**Created**: 2026-05-20
**Estimated duration**: 1.5 days
**Risk level**: Medium-High (touches type-checker generalization; broad blast radius across stdlib and user modules)

## Goal

Fix the upstream root cause uncovered during M-PATTERN-AND-INVOCATION-REPAIR M1: exported function schemes lose their concrete ADT head constructors during letrec generalization. After the fix, `std/json.getNumber: (Json, string) -> Option[float]` must import with that exact return type — not `forall α. (Json, string) -> α` as it does today. Closes [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md) as a side-effect.

## Source design doc

[design_docs/planned/v0_22_0/m-scheme-import-preserve-adt-head.md](m-scheme-import-preserve-adt-head.md) — P0

## Confirmed bug site (pre-sprint investigation)

Found during planning, not during execution. [internal/types/typechecker_functions.go](../../internal/types/typechecker_functions.go) has **three** sites where `SolveConstraints` discards its substitution:

```go
// Line 156 (inferLet single-binding path)
_, unsolvedConstraints, err := ctx.SolveConstraints()

// Line 263 (inferLetRec SCC-wide solve)
_, unsolvedConstraints, err := ctx.SolveConstraints()

// Line 297 (inferLetRec per-binding re-solve)
_, remainingConstraints, err := ctx.SolveConstraints()
```

The first return value is the substitution map. Discarding it means the post-unification bindings (e.g. `α_return → Option[float]`) are not applied to `valueType` before `generalizeWithConstraints(valueType, ...)` runs. Free TVars that ARE in fact bound get quantified anyway, producing the `forall α. ... -> α` schemes observed in the M1 trace.

This is the load-bearing site. The iface builder ([internal/iface/builder.go](../../internal/iface/builder.go)) just canonicalizes whatever scheme is already in the type env, so fixing the typechecker fixes the iface automatically.

## Milestones

### M1 — Apply substitution before generalization (P0, ~0.5 day, ~80 LOC)

**Approach**

In all three letrec/let sites, apply the returned substitution to `valueType` (and any related types) before passing to defaulting + generalization. Pattern:

```go
sub, unsolvedConstraints, err := ctx.SolveConstraints()
if err != nil { ... }
// NEW: apply sub to resolve TVars that have been bound by unification
valueType = ApplySubstitution(sub, valueType)
// (existing) defaulting + generalize
```

For inferLetRec which also has `allValueTypes[i]` and `allValueNodes[i]`, apply sub to each. May also need `applySubstitutionToTyped(sub, valueNode)` to keep CoreTI consistent.

**Sub-tasks**

1. Fix `inferLet` at [typechecker_functions.go:156](../../internal/types/typechecker_functions.go#L156).
2. Fix `inferLetRec` outer solve at [typechecker_functions.go:263](../../internal/types/typechecker_functions.go#L263) — iterate over `allValueTypes` and apply sub.
3. Fix `inferLetRec` per-binding re-solve at [typechecker_functions.go:297](../../internal/types/typechecker_functions.go#L297).
4. Verify substitution flow with the existing `/tmp/repro_step1.ail` (or canonical: a fresh test fixture under `internal/iface/testdata/`).

**Acceptance**

- A new test in `internal/iface/builder_test.go` (or sibling) loads a fixture module with `export func getNum(j: Json, k: string) -> Option[float]` and asserts the resulting iface item's `Type.Type` is `TFunc2{... Return: TApp{Constructor: TCon{Name: "Option"}, Args: [...]}}`. Should fail before M1's patch, pass after.
- `go build ./internal/... ./cmd/ailang/...` clean.

### M2 — Pattern-check downstream verification (P0, ~0.25 day, ~60 LOC test code)

**Approach**

Re-run `/tmp/foreign_ctor_repro.ail` and the cognitive_commons-shaped reproducers. The existing M-MATCH-ADT-XCHECK fast-path (which already exists in [typechecker_patterns.go:166](../../internal/types/typechecker_patterns.go#L166)) should now fire correctly because scrutinees of function calls will be concrete TApps post-substitution.

**Sub-tasks**

1. Add the regression tests from the deferred M-MATCH-ADT-XCHECK sprint that target function-call scrutinees (the ones we couldn't write before because the fix wasn't possible). File: `internal/pipeline/match_foreign_constructor_function_call_test.go`.
   - Function-call scrutinee returning concrete `Option[T]` with Result-ADT arms → expect `MatchForeignConstructorError`.
   - Nested match (outer Result, inner Option from a function call) → expect error on inner arms.
   - Cross-module ADT (scrutinee from std/option, foreign ctors imported from std/result).
   - Negative test: polymorphic scrutinee from a generic helper → must NOT false-positive.
2. Move [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md) from `planned/` to `implemented/v0_22_0/` since it'll be transitively closed.

**Acceptance**

- All four new regression tests pass.
- `ailang check /tmp/foreign_ctor_repro.ail` produces the structured `MatchForeignConstructorError` naming `Ok` and listing `Option`'s constructors.

### M3 — Regression sweep across stdlib + existing tests (P1, ~0.5 day, mostly test runs + triage)

**Approach**

The fix may surface existing bugs that were silently masked by overly-polymorphic schemes — e.g. a test that "passed" because every call site got `forall α. α` and unified with whatever was needed. Run the full test suite and fix any newly-failing tests by either:
- (a) Tightening the test's expected types (the test was wrong before; now it's right).
- (b) Tightening the source code's actual annotations if the production code was relying on the bug.

**Sub-tasks**

1. Run `go test ./internal/... ./cmd/ailang/...` from a clean cache. Triage every failure.
2. Run `make verify-examples` and `ailang check --package .` over all `examples/` and `mcp_tools/`.
3. Run `make ci` to catch any cross-cutting concerns (lint, fmt, file sizes).
4. For each newly-failing test, document in the sprint commit message whether it was test-correctness or source-correctness.

**Acceptance**

- All existing tests pass after triage (no skipped/disabled).
- `make verify-examples` passes.
- `make ci` passes.

## Velocity check

Recent 7d: 114 commits, 270 files, +59334/-12393 LOC. Sprint asks ~140 LOC + extensive test runs over 1.5 days. Well within velocity. M3 is the swing variable — if many tests need triage, could expand to 2.5 days.

## Risks

- **Hidden test dependencies on over-polymorphism**: stdlib or examples may have silently relied on `forall α. α` letting any type through. Mitigation: M3 is dedicated to this; budget half a day.
- **Effect-row generalization parallel bug**: the same `_, unsolvedConstraints, err` discard pattern may have an analog for row substitutions in effect rows. Mitigation: do the substitution apply for rows too if the codebase has symmetric infrastructure, but don't get sidetracked into a separate sprint.
- **Cache invalidation**: iface schemes may be cached on disk in `.ailang/cache/`. Mitigation: bump the cache key via `version.Commit` (already happens automatically per [pipeline_module.go:250](../../internal/pipeline/pipeline_module.go#L250) comment) — no manual nuke needed in CI, but flag this in the user-facing release note for v0.22.0.
- **Performance regression**: applying full substitution to large generalized types may slow type-checking. Mitigation: M3 includes a quick benchmark check before/after using `time go test -bench=. ./internal/pipeline/`.

## Out of scope

- Refactoring the broader generalize/instantiate machinery (substantial type-checker rework).
- Improving iface JSON serialization (already passes through; not the bug locus).
- Fixing any *other* type-checker latent bugs surfaced during M3 triage that aren't simple test-fix or annotation-fix.

## Success metrics

- The `getNumber` import scheme is `(Json, string) -> Option[float]` (concrete TApp head, no quantification over the return position).
- `MatchForeignConstructorError` fires for function-call scrutinees.
- 2 design docs move from `planned/` to `implemented/v0_22_0/`: this one and `m-match-adt-xcheck-regression.md`.
- msg_20260520_111521_44c38751 (foreign-ctor bug report) gets a "fixed" reply with the commit hash.
- No new flaky tests; no regressed examples; CI guard from previous sprint still green.

## Day-by-day

| Day | AM | PM |
|-----|----|----|
| 1 | M1: substitution-apply patch + targeted test (~80 LOC) | M2: function-call xcheck regression tests + verify the cognitive_commons-shape repro errors out |
| 2 AM | M3: full test sweep + verify-examples + ci. Triage and fix newly-failing tests. | Move design docs to implemented/, final commit, reply to bug-report messages. |

## Notes

- The bug site is in `typechecker_functions.go`, not `iface/builder.go`. The iface builder's TODO comment on `freeVars := []string{}` is misleading — the iface builder receives an already-canonicalized scheme; the corruption is upstream. Don't get distracted by that TODO unless time permits as a polish item.
- The pattern-check code from M-MATCH-ADT-XCHECK (v0.18.10) is sound and unmodified by this sprint. It just couldn't see concrete scrutinee types before. After this fix, it WILL see them and produce the right errors.

# Effect checker charges a stored (never-called) lambda's effects to the enclosing function

- **Date**: 2026-09-17
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `interpolat` across design_docs/, `lambda` + `effect` intersections, `effect check`, the `ailang-core-triage/` backlog rows; inspected `internal/pipeline/validate_effects.go`
- **Estimate**: omitted (design-doc route)

**Why (mechanism, verified in code and repro):** the interpolation in the report
is a red herring — the real defect is in `collectRequiredEffects`'s
`*core.Lambda` case (`internal/pipeline/validate_effects.go` ~line 330). It
ignores the lambda's own type row (the comment even says "the effects are in
their type, and we need to check the body") and returns the **body's** required
effects. So when a top-level function's value is a lambda whose body merely
*stores* an inner effectful lambda (a record field, a returned closure), the
inner lambda's row is unioned into the outer function's required row even
though the outer function never performs those effects — building the closure
is pure. DEBUG_EFFECTS on the repro shows exactly this: `Lambda -> checking
body` twice, then `Lambda body effects: [AI, Clock, FS, Net, Process]` charged
to `hostA ! {FS}`. On this workspace's binary (v0.39.3, commit 5d32f85 — one
tag behind the reporter's 6e299982e; no Go toolchain here to rebuild), the
differential claim does not hold: **hostA, hostB and hostC all fail in
isolation** with identical missing effects. In the full file hostA is
validated first and checking aborts, which likely explains why B/C looked
clean to the reporter. Interpolation matters only in that its desugar inserts
a synthetic `show` call with no source node (see the comments at
`internal/ast/ast_expr.go:40` and `internal/core/core.go:445`), which is
plausibly what defeats whichever CoreTypeInfo fallback the reporter's newer
binary consulted — but the body-walk charges the lambda row with or without
interpolation.

**Not a duplicate:** `effect-checker-let-shadowing.md` (same directory) is a
different bug in the same function — the name-keyed `declaredEffects` map
resolving shadowed callees. Both live in `collectRequiredEffects`'s callee/
lambda resolution, and a design doc should treat them together: the fix shape
(lambda case should return the lambda's declared/type row rather than walking
the body, or walk the body only when the lambda is in operator position)
interacts with the M-EFFECT-ROW-SHOW-INTERP (#386) soundness fix that
deliberately walks bodies via the Let case, and with `validateLambdaAnnotations`,
which relies on body walks for inline `! {}` annotations. There are at least
three defensible remedies (type-row-first for stored lambdas; position-based
walking; consult CoreTypeInfo and fall back to body only when type info is
absent — e.g. the synthetic `show` nodes), each with different soundness
tradeoffs against #386, and the change redefines the effect gate's attribution
contract. That is rubric rows 3/4, not a two-line patch.

**The daneel symptom is the same leak via a second path:** the row-unification
failure (`failed to unify record field 'draft': r1 has extra labels [], r2 has
extra labels [AI Clock FS Net Process]`) means inference itself pushes the
lambda's body row into the record literal's field type, so the leak is not
confined to the check pass — the design doc should decide whether the fix
belongs in `collectRequiredEffects` only or also in how TFunc2 rows are
inferred for lambda-typed record fields.

**Correct framing for the doc:** `hostA` only *builds* the record; the lambda's
effects must belong to whoever calls `draft` (here, `main`, which already
declares them). Today's error message also names the wrong function —
`formatEffectError` reports the enclosing decl, not the stored lambda — worth
fixing in the same pass.

**Prior coverage:** none found for the stored-lambda attribution itself
(searches listed above); the nearest rows are the let-shadowing triage row and
the #386 show-interp fix that this bug's fix must not regress.

**Second field report (2026-09-17, attended):** `sunholo/motoko_ext_a2a 0.2.2` fails
`check --package` on the current binary for exactly this — `make_hooks(cfg) -> ExtensionHooks`
builds three effectful hook lambdas and stores them in the record; the checker charges
`Net, Rand` to `make_hooks`. Two package-inbox agent attempts (`task-18896425`, `task-f8fc9985`)
could not "fix" it, correctly: the motoko ABI expects `register_with_config -> ExtensionHooks ! {}`,
so adding the effects to `make_hooks` would be wrong. Every `motoko_ext_*` package that stores
effectful hooks will hit this on republish; the package side is blocked on this row.

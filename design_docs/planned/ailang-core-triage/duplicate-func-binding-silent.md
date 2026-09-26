# Duplicate top-level func definitions in one module silently overwrite (MOD007 gap)

- **Date**: 2026-09-23
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: "duplicate" and "redeclare/already defined" across internal/parser and internal/elaborate; `MOD007`, `M-MODULE-LET-FUNC-RESOLUTION`, "module scoping collision", "known bugs" across design_docs/ and docs/
- **Estimate**: n/a (design-doc)

Reproduced at repo HEAD (`ailang check` on the reporter's repro): the second
`pick` is accepted, and `caller` fails with `TC_ARITY_001` as reported. The root
cause is a documented-but-incomplete gate: `internal/elaborate/file.go`
(`checkDuplicateModuleBindings`, ~line 16) enforces MOD007 for **let-vs-func and
let-vs-let** collisions only, and its comment explicitly assumes "funcs are
declared, so unique by the parser's own rules" — the parser does no such check.
Downstream, the elaborator's `symbols := make(map[string]*FuncSig)` map
(file.go ~line 194) is last-wins, so the second definition's signature replaces
the first before the decl-by-decl type-check loop in
`internal/pipeline/pipeline_module_compile.go`, which is exactly why call sites
of the *earlier* definition are checked against the *later* one and the error
lands on the caller 450 lines away. Semantically this is a silent-shadowing
soundness trap of the same class MOD007 was created to kill.

An existing design doc rules on the *neighbouring* cases but not this one:
`design_docs/implemented/v0_30_0/m-module-let-func-resolution.md` froze the
collision gate as "duplicate module-scope name (let vs func, let vs let) →
compile error with both positions" — func-vs-func is omitted from that list —
and `docs/docs/reference/errors/mod007.md` states the same two-case scope. So
this is not a duplicate-of; it is the next member of the #366 family. The fix
itself is small (add a seen-func map check next to the two existing MOD007
checks in `internal/elaborate/file.go`, report at the second definition's
position, update `docs/docs/reference/errors/mod007.md` and add a test), but it
widens a gate's contract — previously-accepted programs become compile errors —
and the doc text defining MOD007's scope needs revision alongside it, which is
row 4 of the rubric: design-doc. It also gives the natural place to decide
whether the reporter's suspected sibling (same-named internals across modules
colliding at runtime, the M-MODULE-SCOPE family) shares this root cause; the
evidence here (surface-AST collision gate + last-wins symbol map) is consistent
with that but unverified.
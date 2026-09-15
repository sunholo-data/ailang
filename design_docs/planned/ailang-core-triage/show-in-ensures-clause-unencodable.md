# `show` from string interpolation inside requires/ensures clauses is not encoded — ShowNormalizer misses contract exprs

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `ensures` in design_docs/, `interpolat` in design_docs/, `cannot encode ensures|unsupported application` in internal/, `show` in `internal/smt/`, and read `design_docs/implemented/v0_36_0/m-smt-interp-show.md` (the implemented fix for the body case, `fb_913ee851c83c0d8c` — same feedback ID as this report).

## Why

This is the contract-path residue of **M-SMT-INTERP-SHOW** (implemented v0.36.0, `design_docs/implemented/v0_36_0/m-smt-interp-show.md`). The v0.36.0 fix works only on function bodies: the desugar (`internal/parser/parser_literals.go`, `parseInterpolatedString`) inserts `$builtin.show` around every interpolation hole, and the pipeline's `ShowNormalizer` (`internal/pipeline/show_normalize.go`, `rewriteDecl` → `s.expr`) walks only LetRec/Let binding values and the decl body. Contract clauses are **not** part of the body — they live on `core.Func.Contracts` (`internal/core/core.go:435`, `Contract.Expr`), so a `${lid}` inside an `ensures` clause keeps its `show` wrapper, reaches the ensures encoder (`internal/smt/codegen.go` ~line 484, "cannot encode ensures clause"), and is rejected by `encodeApp` with `unsupported application: $builtin.show([lid])` (`internal/smt/codegen_apps.go:142`). Bodies verified since v0.36.0 because the same desugared `show` is elided/rewritten there before SMT codegen.

The fix is well-determined: run the existing M1 show normalization over `Contract.Expr`s of each decl (string-typed holes → identity, i.e. the clause becomes the `concat_String`/`str.++` chain the body encoder already handles; bool/int rows reuse M1/M2 rewrites; genuinely non-primitive holes remain honest M3 residue). No new encoding design is needed — the normalization is type-directed and `CoreTypeInfo` already covers contract exprs since they are type-checked. Residue diagnostic (`show_rejection.go`) should be extended so a show in a clause reports with the interpolation hint rather than a bare `show` skip.

## Why design-doc, not direct-fix

- Rewriting `rewriteDecl`/`Normalize` to also traverse `Func.Contracts` is ~20–40 LOC in `internal/pipeline/show_normalize.go` plus the residue-hint touch in `internal/smt/show_rejection.go` and tests — well over the 2-line/1-file direct-fix threshold.
- There are real design decisions: whether normalization runs per-contract-expr with the enclosing function's type info, whether the M3 residue note gains a "in ensures clause" origin, and which example/test proves the reporter's Gmail modify-body case (two clauses, one with `result ==` against an interpolated string). These deserve a short doc extending M-SMT-INTERP-SHOW rather than an unreviewed patch. Recommend scoping it as an M-SMT-INTERP-SHOW follow-up (M5: "normalize contract clauses") rather than a from-scratch doc.

## Repro (confirmed consistent with code)

`pure func modifyBody(lid: string, markRead: bool) -> string ensures { ... result == "{\"addLabelIds\":[\"${lid}\"],\"removeLabelIds\":[\"UNREAD\"]}" ... }` → `ailang verify` fails with `encoding error: cannot encode ensures clause: ... unsupported application: $builtin.show([lid])`, while the identical interpolation in the body verifies.

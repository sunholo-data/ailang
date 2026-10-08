# Z3 HOF inlining is unreachable behind the callee-sort gate, and still hard-codes `(Seq Int)` underneath (#1375, #215)

- **Date**: 2026-10-03
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `#1375`, `#215`; `hof_verify`, `InlineHOFCalls`, `SpecializeMap`, `firstUnencodableCalleeInFunc`, "callee-sort gate", "let shadow", "accessor" across design_docs/ (hits: `implemented/v0_30_0/m-smt-callee-sort-gate.md` — introduced the gate and absorbed #215's *symptom* per the 2026-07-22 triage comment, but does not mention HOF callees; `implemented/v0_9_1/m-smt-fragment-expansion-v2.md` — the original HOF inliner; `implemented/v0_14_3/m-smt-cross-module-types.md` — "M3b deferred" note that spawned #215). No planned doc covers ordering the gate against the inliner, the element sort, or let-name hygiene.
- **Estimate**: omitted (design-doc)

**#1375 reproduces** at origin/dev `790169359`: `ailang verify examples/runnable/contracts/hof_verify.ail` →
`⚠ SKIPPED positives — calls "filter" whose signature uses an unencodable type "list[a]"`, same for
`incrementAll`/`map` (`list[b]`); `3 exported functions: 2 skipped, 1 without contracts`, rc 0. The gate runs
in `smt.Verify` (`internal/smt/verify.go:255-266`, `firstUnencodableCalleeInFunc` in `internal/smt/from_ast.go:321`)
before encoding, and rejects any callee whose surface signature mentions a type variable. The HOF inliner
that would have replaced `map`/`filter`/`foldl` with a specialised `define-fun` runs later, inside encoding
(`internal/smt/codegen.go:360-386` → `InlineHOFCalls`, `internal/smt/hof_inline.go:303`), so it never sees
these bodies. The example's own file is the feature's showcase and still "passes" `make verify-examples`,
because SKIP is not a failure.

**#215 is masked, not fixed.** Its Z3 crash (`unknown constant applicable (Int)`) can no longer occur because
the gate skips the function first, but both of its mechanisms are still in the tree:

1. The inliner hard-codes the element sort: `hof_inline.go:331-347` builds the specialised parameter as
   `TList{Element: TCon{"int"}}` and the return sort as `"(Seq Int)"` for map and filter, `Int` accumulator for
   foldl, whatever the list's actual element type.
2. `let` bindings are emitted under their source name (`internal/smt/codegen_control.go:21`,
   `(let ((%s %s)) %s)` with `let.Name`), so a local named like a record accessor (`applicable`) shadows the
   accessor `declare-fun`.

So the obvious #1375 fix — exempt inlinable HOF callees from the gate, or run the inliner first — would make
`hof_verify.ail` (`[int]`) verify again and simultaneously re-open #215 for every non-`int` list, turning an
honest SKIP back into a Z3 error. That coupling is why this is a design doc and not a drive-by.

Options:

- **A. Exempt + sort-thread + hygiene, together (recommended).** In the gate, skip callees that
  `InlineHOFCalls` will consume (literal-lambda `map`/`filter`/`foldl`, the shapes `hof_inline.go` already
  detects); thread the list argument's element sort and the lambda's result sort into the specialised
  `define-fun`s instead of `Int`; prefix encoded `let` names (as `$p_` already does for parameters) so they
  cannot collide with accessors or other declared functions. Where the element sort is itself unencodable,
  keep the honest SKIP with a reason naming the element type.
- **B. Exempt only when the element sort is `Int`.** Restores the showcase cheaply; leaves #215's cases as
  SKIP. Acceptable as M1 of A, not as the whole fix.
- **C. Leave the gate first and document HOFs as unsupported.** Retires a shipped feature; not recommended.

Acceptance: pin `hof_verify.ail` to VERIFIED (both functions) in `make verify-examples` so a SKIP regression
fails CI (the issue's ask 2); add a `[record]`-list `filter` fixture with a `let` whose name equals a record
field (the docparse `evalComputeScore` shape from #215) that VERIFIES or SKIPs honestly but never reaches a Z3
error. Close #215 with the same PR. Issues: https://github.com/sunholo-data/ailang/issues/1375,
https://github.com/sunholo-data/ailang/issues/215

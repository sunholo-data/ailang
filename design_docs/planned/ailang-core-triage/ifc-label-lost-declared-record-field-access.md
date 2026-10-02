# IFC Label Lost at Field Access of a Declared Record Type (#1523)

- **Date**: 2026-10-02
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `grep -ril "label" design_docs/`; `grep -ril "authcode" design_docs/ docs/`; `ls design_docs/planned/ailang-core-triage/`; read `ifc-labels-lost-across-module-boundary.md`, `planned/v0_52_0/m-mcp-oauth-package.md`, `planned/m-ifc-cross-module-labels.md`; `grep -n "FieldAccess|labelOf" internal/types/ifc_check.go`
- **Estimate**: >2 lines in `internal/types/ifc_check.go` (see below) — exceeds DIRECT_FIX_MAX_LINES, and it changes a gate's contract

**ATTENDED-FIX IN FLIGHT — triage/record only.** The reporter states this is being fixed
in an attended session. This row records the finding and the mechanism; the executor
should join or review that work rather than open a competing design doc.

Repro confirmed on this machine's binary (v0.51.0): `logLine(c.raw)` where
`c: RawCode`, `type RawCode = { raw: string<authcode> }`, sink `string{not authcode}`
— **accepted** by `ailang check`. Controls behave as reported: inline record
(`{ raw: mint() }.raw`) and plain return (`logLine(mint())`) are both rejected.

Mechanism: Check A is a surface-AST pass that never consults declared types.
`buildIFCSig` (`internal/types/ifc_check.go`) seeds a parameter's label only from an
`*ast.LabelledType` written directly on the parameter, so a `c: RawCode` parameter gets
label ⊥; `labelOf`'s `*ast.RecordAccess` case then "conservatively" joins the record
expression's label (⊥) — the join is only conservative if the record expression is
itself labelled, which a bare parameter of a declared alias is not. The field-declared
refinement `string<authcode>` inside the alias is invisible to the entire walk. Inline
records work only because `labelOf(*ast.Record)` happens to join field labels on the
same AST. This is exactly the record-field edge the 2026-09-17 triage row
`ifc-labels-lost-across-module-boundary.md` flagged as a secondary item ("nothing in
Check A walks field-declared refinements") and recommended either folding into
M-IFC-CROSS-MODULE's scope or filing separately; this is that filing.

Why design-doc, not direct-fix: the fix must make label lookup type-aware (resolve the
record expression's declared alias/row type and pull field refinements into Check A) —
more than 2 lines, and it tightens the IFC gate's contract, so previously-accepted
programs start failing (the strict same direction as every other IFC tightening, but a
semantics change someone could disagree with). Cross-reference: `m-ifc-cross-module-labels.md`
does not cover this case (no mention of records/fields); `m-mcp-oauth-package.md`
(V16/D4, target v0.52.0) already designs **around** it — single security module, no
declared record types holding labelled values, CI grep lint — which is a documented
workaround, not a fix, so neither doc rules on this and `duplicate-of` does not apply.
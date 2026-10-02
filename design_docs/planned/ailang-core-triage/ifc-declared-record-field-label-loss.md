# IFC: labelled field of a declared record type loses its label at field access (#1523, dup #1527)

- **Date**: 2026-10-02
- **Class**: bug (security)
- **Recommend**: design-doc
- **Searched**: `declared record`, `RecordAccess.*label`, `field access.*label`, `labelled field` over `design_docs/`; `ls design_docs/planned/ | grep ifc`

Reproduced on dev (`a59bbdbeb`): `ailang check` accepts `logLine(c.raw)` where
`c : RawCode = { raw: string<authcode> }` and `logLine(s: string{not authcode})`,
both for a let-bound call result and for a `c: RawCode` parameter. Mechanism:
`internal/types/ifc_check.go` is a value-flow analysis over the surface AST whose
only label SOURCES are a *top-level* `<label>` on a parameter or return type
(`buildIFCSig`) and `secret()`. Labels nested inside a type — a record field, a
declared type's body, a list/tuple element, an ADT constructor field — are never
read, so `c.raw` inherits `c`'s label, which is ⊥. The inline-record case only
"works" because the label rode in on the value. The same root cause therefore
covers inline record annotations, `[string<secret>]` params, tuples and ADT
constructor fields — a systemic gap, not one projection. More than one acceptable
fix (deep-label the whole value vs. field-precise projection vs. reuse HM types)
and it changes what the checker rejects (rows 3 and 4). None found in existing
docs: `m-mcp-oauth-package.md` T7 only designs around it, and
`m-ifc-cross-module-labels.md` is the separate module-edge hole. Being fixed in
an attended session: `design_docs/planned/v0_52_0/m-ifc-declared-record-labels.md`.

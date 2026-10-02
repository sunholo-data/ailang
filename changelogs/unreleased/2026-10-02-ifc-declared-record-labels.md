### Fixed — SECURITY: IFC labels written inside a type were never enforced (#1523, #1527) (2026-10-02)

`CheckModuleIFC` only treated a top-level `<label>` on a parameter or return type
(and `secret()`) as a label source. A label written anywhere inside a type was
ignored, so `logLine(c.raw)` with `type RawCode = { raw: string<authcode> }` and
`logLine(s: string{not authcode})` compiled cleanly. The same hole covered
nested records, record destructuring, inline record annotations, ADT
constructor fields, tuple and list elements, type aliases, `let` and lambda
annotations, and getter functions (17 measured shapes).

A value's label is now the labels that flowed in with it, joined with every
label its static type declares (module-local types). Field access stays
field-precise: an unlabelled field beside a labelled one is still clean. An
annotation can only add labels, never lower them. The change only adds
rejections: `inbox_injection_v2.ail`, the `secrets` examples, docparse's
`api_keys.ail` IFC test and `sunholo/mcp_oauth`'s `tests/ifc_leaks.sh` keep
their verdicts. Imported types are still out of scope
(`m-ifc-cross-module-labels`).

Design: `design_docs/planned/v0_52_0/m-ifc-declared-record-labels.md`.
Code: `internal/types/ifc_check.go`, `internal/types/ifc_static_type.go`.
Guide: `docs/docs/guides/ifc-labels.mdx` § "Labels inside types".

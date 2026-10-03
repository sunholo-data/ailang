# `pure func` is not checked against its declared effect row, and iface purity is a hard-coded `true` (#1443)

- **Date**: 2026-10-03
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `#1443` (hit only as an out-of-scope note in `design_docs/planned/v0_51_0/m-serveapi-directory-ready.md:362,459`); `IsPure`, `determinePurity`, "pure func" + `! {`, "iface purity", `"pure": true` across design_docs/planned (hits: `v0_35_3/m-effect-pure-row-overgeneralization.md` — a `pure func` must not *export an effect-polymorphic row* (#1091), a different defect; `v0_48_0/m-effect-latent-function-values.md` — latent effects through HOF args (#1326/#573), different mechanism; `m-builtin-classification-surface.md` — builtin `IsPure`, unrelated). No doc rules on the keyword-vs-row contradiction or on iface purity.
- **Estimate**: omitted (design-doc)

Both halves reproduce at origin/dev `790169359`:

1. `export pure func sneaky(x: string) -> unit ! {IO} { println(x) }` → `✓ No errors found!`.
   The parser sets `FuncDecl.IsPure` from the `pure` token (`internal/parser/parser_func.go:21-33`) and the
   elaborator copies it to Core meta (`internal/elaborate/file.go:291,419`), but
   `internal/pipeline/validate_effects.go` (`ValidateEffects`, ~line 103) compares only the *declared row*
   against the required effects; `IsPure` never participates. So `pure` plus a non-empty row is accepted as
   long as the row covers the body.
2. `internal/iface/builder.go:667` `determinePurity` returns `true` for every export; its value flows into
   `ExportInfo.Purity` (`builder.go:395-401`), the `ailang iface` JSON (`iface/json.go:118`, `"pure": true`)
   and the compile cache (`pipeline/cache_store.go:227,306`). serve-api has already been patched to ignore
   it (9305f1c19) and `@mcp_hints` deliberately distrusts both signals.

The `pure` flag is not only cosmetic, which is why this is not a one-line fix: `internal/types/typechecker.go:115`
generalises a binding with an *empty* effect row when `d.IsPure`, and `internal/smt/encodable.go:137`
treats `meta.IsPure` as admission to the decidable fragment (with `verify.go:116-148` additionally inferring
`IsPure` from an explicit `! {}`). A `pure func ... ! {IO}` is therefore generalised and SMT-admitted as if
it had no effects, so the lie propagates beyond the checker's own message.

Options for (1):

- **A. Reject the contradiction (recommended).** `pure` with a non-empty declared row is a new error at the
  signature ("`pure` declares no effects but the signature declares `! {IO}` — drop `pure` or the row").
  Keeps `pure` meaning exactly `! {}`, which the docs already say (`docs/docs/reference/effects.md:405-407`,
  the `pure`-usable column at :478). Needs a corpus sweep of std/examples/registry for existing
  `pure func ... ! {…}` spellings.
- **B. Make `pure` desugar to `! {}`** and reject an explicit non-empty row only by the existing
  missing/extra-effects path. Same user-visible outcome, but couples the fix to row inference.
- **C. Make `pure` advisory** and stop using `IsPure` in generalisation/SMT admission. Removes the keyword's
  value; not recommended.

Options for (2): derive `Purity` from the checked effect row — closed and empty, with no row variable —
which also answers the #1091 case (an effect-polymorphic row is *not* pure). The iface builder needs the
type-checked row for each export rather than the Core expression it currently receives; that plumbing is
the main cost. Once landed, revert serve-api's local workaround to read `ExportInfo.Pure` again and add a
cache-key bump, since cached `iface` entries all carry `Purity: true`.

Recommendation: A + derived purity, landed together (one rule: "pure ⇔ closed empty row", enforced at the
signature and reported by iface). Acceptance: the issue's `sneaky` fails check; `ailang iface` reports
`"pure": false` for any export with a non-empty or open row; `pure func` + `! {}` and unannotated pure
functions are unchanged. Issue: https://github.com/sunholo-data/ailang/issues/1443

# M-IFC-DATA-CALLABLE-PRUNE: values of function-free types carry no callable

**Status:** Implemented (branch `fix/ifc-data-callable-prune`)
**Priority:** P0. specstore 0.3.0 cannot publish: v0.57.1 rejects the fix we recommended for it.
**Created:** 2026-11-05
**Reported by:** the specstore session, in two messages to `user`:
- `inbox_1793862301026_e0d9a62c` (07:05): `pickHeader` rewrite;
- `inbox_1793863515026_685ab1d4` (07:25): 18-line repro and DX notes.

**Introduced by:** `3484c6789` (#1719, v0.56.0), the imported-callee fallback F2.
**Follows:** [M-IFC-RECURSIVE-LAMBDA-FOLD](../v0_57_1/m-ifc-recursive-lambda-fold.md) (v0.57.1), which made this pass finish, and so exposed what it then reports.
**Quorum:** no attended trigger fires:
- no design-freeze items;
- no "override" row;
- no cost or schema surface;
- every premise was verified in this repository.

## Problem

We told specstore to declassify each cell once, then choose the header and columns from the resulting shapes. Done that way, v0.57.1 rejects the code (V1):

```
func shapeRow(r: [Cell]) -> [string<structure>] ! {Declassify} = mapE(\c. mark(c.text), r)
func shapeAll(rs: [[Cell]]) -> S ! {Declassify} = { rows: mapE(shapeRow, rs) }
func at(xs: [[string<structure>]], k: int) -> [string<structure>] = match nth(xs, k) { … }
func pick(s: S) -> int = length(filter(\k. length(at(s.rows, k)) > 0, range(0, length(s.rows))))
→ value labelled <structure> reaches parameter "raw" of mark (via pick → filter → at → nth → shapeRow → mapE)
```

**Mechanism (V2).** `applyOpaque`, the fallback for imported callees, has two parts:

- It returns the join of its arguments' callables, because an imported higher-order function might return its function argument.
- It also applies every argument's callable to the other arguments' labels.

So the list that `mapE(shapeRow, rs)` produces "may be" `shapeRow`. That list travels through `shapeAll`'s record into `at`. There, `nth(xs, k)` "applies" it to `k`'s `<structure>` label, and replaying `shapeRow` checks `mark`'s `<client>` demand.

The value has the declared type `[[string<structure>]]`, which can never be applied. That is the false positive. The misleading "via" chains in DX note 2 (`via … nth → shapeRow → mapE`) are this same mechanism rendered as a path.

## Design

**`ifcTypes.dataOnly(t)`** (`ifc_static_type.go`) is true when every value of `t` is known to contain no function:

- primitives: `string`, `int`, `float`, `bool`, `unit`, `char`, `bytes`;
- lists (`TypeApp "list"`), arrays, tuples and records of such types;
- module-local declarations whose bodies are function-free, including generic ones, with parameters bound to the arguments;
- recursive declarations, decided by their other fields;
- enums.

Everything else is false, so the pass stays conservative: nil, an unbound type variable, a function type, and imported names such as `Option[...]`. This pass sees one module, so an imported declaration is invisible to it.

**Prune points.** A value that can never be applied has no requirement to carry. Both prune points use the declared types already in the AST, so neither needs HM types (design F1 holds):

1. `flowVal`: a value whose `staticType` is function-free carries no callable. `staticType` covers identifiers with declared types (parameters and annotated `let`s), calls to module functions (their declared return type) and record fields.
2. `walkFunc`: a module function whose declared return type is function-free has a nil result callable.

**Soundness.** A callable matters only when its value is applied. The HM checker has already proved that a value of declared type `[[string<structure>]]` holds no function. Every demand on values whose type can hold a function is unchanged: function types, type variables, imported generics and unannotated bindings.

**Function name in limit errors (DX note 1).** `ifcAnalysisLimit` carries the module function being analysed. Its message now reads "… exceeded (summary size) while analysing function X …", and the suggestion names the shape to simplify.

## Residual (not fixed here)

A value with **no** declared type that flows through an imported call is still treated conservatively. For example, with `let rows = mapE(shapeRow, rs)`, `range(0, length(rows))` is assumed to possibly return `shapeRow`: this pass has no result types for imported functions, and `length` returns `int`.

**Workaround:** annotate the binding, `let rows: [[string<structure>]] = …`. The real fix is giving the IFC pass the result types of imported callees, from the iface or HM, which reopens design F1. It is a follow-up, not part of this fix.

## Conflict surface

**Position changed:** the callable carried by an IFC value. No syntax, HM type, iface or codegen change.

**Must still work:**
- The #1719 routes for function values:
  - aliases;
  - higher-order parameters;
  - record fields and list elements of *function type* (`TestIFCPrune_FunctionListKeepsDemands`);
  - returned functions and wrappers.

  The whole `ifc_*_test.go` corpus passes.
- Direct demands (`TestIFCPrune_KeepsDirectDemand`).

**Intentional change:** fewer false positives on data. No diagnostic appears that did not appear before.

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | v0.57.1 rejects the 18-line repro and the agent's branch | `ailang-v0.57.1 check t.ail` gives `t.ail:18:58 … via pick → filter → at → nth → shapeRow → mapE`. `ailang-packages@specstore-0.3-fixes` (`1173f19`) gives `audit.ail:251:24 … via filter → shapeRow → mapE → shapeOf` |
| V2 | The callable rides on data | Temporary summary dump: `shapeAll`'s result callable is `[O\|P:f:shapeAll#0^0\|T:shapeRow]`, a record of lists carrying `shapeRow` |
| V3 | List types parse as `TypeApp{Constructor: "list"}` (the first cut missed this) | `internal/parser/parser_type.go:56,109` |
| V4 | The pass has only module-local type declarations | `ifc_static_type.go` header comment and `deepDecl`: "an imported type name has no declaration here" |
| V5 | The residual is real | With an unannotated `let rows = mapE(shapeRow, rs)`, the violation is still reported, `via filter → shapeRow → mapE`, through `length(rows)`; it passes when annotated |
| V6 | No behaviour change elsewhere | v0.57.1 vs fix, identical compiler sources apart from `internal/coordinator`:<br>• 530 `examples/` and `std/` files: identical apart from a version-stamp warning;<br>• 45 `ailang-packages`: identical after normalising line order |

## Results

| # | Result |
|---|---|
| R1 | The repro and the agent's shape-first branch check clean (`specstore-0.3-fixes`: 2.9 s, 79 MB, 0 errors) |
| R2 | Tests in `ifc_data_callable_prune_test.go`:<br>• declassified data;<br>• annotated-let parameter pruning;<br>• direct demand kept;<br>• function list keeps its demands;<br>• an 11-case `dataOnly` table;<br>• the limit message names the function |
| R3 | Mutants, each caught:<br>• `flowVal` prune off → the parameter test fails;<br>• list rule off → three tests fail;<br>• return-type prune off → the parameter test fails |
| R4 | `go test ./internal/types ./internal/pipeline ./internal/iface`, `make lint`, `make check-file-sizes` and `make test-changed` are green |

## The specstore DX notes not addressed here (follow-ups)

- **Note 3:** the labels guide shows `string<email ⊔ pii>`, but `⊔` does not parse in a signature (`PAR_LABEL_GT_EXPECTED`). Either the guide or the parser is wrong.
- **Note 4:** implicit flow differs by construct. A row chosen by `if` on a declassified value becomes `<client> ⊔ <structure>`; the same choice through `filterE` stays `<client>`. One rule should hold for both, and which one is a soundness decision.
- **Note 5:** `check --package` fails the whole package when one module fails the IFC check, so consumers of unaffected modules are blocked. A per-module result is a product decision.

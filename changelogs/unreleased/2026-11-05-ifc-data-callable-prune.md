### Fixed — information-flow false positive: a declassifier's input requirement stuck to the data it produced (2026-11-05)

In v0.56.0–v0.57.1 the information-flow check assumed that an imported function such as `mapE` might
return the function passed to it. So the list `mapE(shapeRow, rows)` produced carried `shapeRow`'s
requirements as if it were a function. Indexing or filtering that list with a `<structure>` value then
failed with `… reaches parameter "raw" of mark … declared string<client>`, even though the declassifier
had already run. The "via" chain in that error pointed through `nth` or `filter` to the
declassifier, which made it hard to find the real problem.

Now a value whose declared type cannot contain a function carries no such requirement. That covers
declared parameters, annotated `let` bindings, the return types of module functions, and record
fields. The types counted are strings, numbers, lists, tuples, records and module-local types built
from them.

Requirements on function-typed values are unchanged. A value with no declared type that passes
through an imported call is still checked conservatively; annotate it
(`let rows: [[string<structure>]] = …`).

When the check stops at its analysis limit, the error now names the function it was analysing.

Design: `design_docs/planned/v0_57_2/m-ifc-data-callable-prune.md`.

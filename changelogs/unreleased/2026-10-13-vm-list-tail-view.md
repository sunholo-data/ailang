### Fixed — `x :: rest` matching on the bytecode VM was quadratic (#1814, 2026-10-13)

The VM bound `rest` by copying every remaining element (`builtinListTail`), so ordinary recursion
over a list was O(n²) on `--bytecode`: summing 80,000 elements by `x :: rest` took 4.5 s, and 200,000
took 25.9 s, against 0.02 s for `foldl`. `_list_tail` now returns a view of the parent's backing
array, as the interpreter already did. List slices are never written after construction (every VM
list builtin builds a fresh slice), and the view's capacity is clipped to its length so an append
on it always reallocates. The same 80,000-element recursion now runs in 0.03 s. Building lists with
`::` is still O(n) per prepend; that is the cons-cells programme
(`design_docs/planned/m-list-cons-cells-decomposition.md`).

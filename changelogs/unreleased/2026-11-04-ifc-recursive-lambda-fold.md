### Fixed — the information-flow check ran away on code that recurses through a lambda (2026-11-04)

Since v0.56.0, a module that has information-flow requirements (`string{not ℓ}`, `T<ℓ>`) and recurses
through a lambda never finished its information-flow analysis. An example of such recursion is
`walk(xs) = flatMap(\b. … walk(…) …, xs)`. Each round of the analysis built a deeper copy of the
lambda, until the work budget stopped it with `information-flow analysis limit exceeded (summary work
budget)`. A 239-line module (`sunholo/specstore` 0.3.0, `audit.ail`) took 60 s and 1.4 GB, and the
registry validator refused it on memory.

A lambda that reaches itself through recursion is now folded into a single self-reference, so the
analysis converges. That module now checks in 0.13 s at 69 MB, and the whole package in 0.55 s at
72 MB (from 63 s and 1.37 GB). Folding merges demands and never drops one.

A second limit now stops any summary that grows past 64 KiB. Any future analysis that fails to
converge stops quickly with `information-flow analysis limit exceeded (summary size)`, instead of
using memory until the step budget runs out.

Output is unchanged for code that already converged: all 530 `examples/` and `std/` modules and all
45 `ailang-packages` packages give identical `check` output. A module that used to stop at the limit
now finishes, and may report real violations the limit was hiding.

Design: `design_docs/planned/v0_57_1/m-ifc-recursive-lambda-fold.md`.

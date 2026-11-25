### Added — Typed holes (`_?`, `?name`) and `std/list.find`

- **Typed holes (`M-AI-PREFERRED-LANGUAGE M3`):** Write `_?` (anonymous) or `?name` (named) in any expression position to query the typechecker for the resolved expected type and in-scope local bindings (`TC_HOLE` structured diagnostic, surfaced cleanly in `ailang check --json` and `--format=agent`, preserved verbatim by `ailang fmt`).
- **`std/list.find(p, xs) -> Option[a]`:** Short-circuiting iterative $O(n)$ search returning the first matching element (0 VM fallbacks under `--strict-bytecode`).

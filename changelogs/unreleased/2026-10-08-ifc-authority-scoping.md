### Added — IFC authority scoping (Refs #752; Refs #1134)

- Added opt-in `Declassify[label=email]` authority for a single label; bare
  Declassify keeps blanket authority. Callers must cover the callee's scope.
- Local positive-labelled parameters now reject uncovered argument labels with
  `param_label_cover`; literals and unlabelled values remain accepted.
- Removed the unused bool-only CheckDeclassify helper; the surface IFC checker
  owns return coverage and scoped authority.
- Known limitation: indirect calls (`let g = sink in g(a)`, `apply(sink, a)`) bypass Checks A and C; tracked in a follow-up issue (#1719).

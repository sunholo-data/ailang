### Fixed — renamed type imports (`import M (Row as R)`) now expand

- A selectively imported type alias was registered under its original name, not the name bound in scope.
  After `import sim/types (Row as R)`, `R` stayed an opaque constructor. Passing an `R` where a `Row` is
  expected failed with `cannot unify record with unexpandable type constructor R`, and
  `ailang run --args-json` could not decode an `R` parameter (`unsupported type constructor: R`).
  Reported by stapledons_godot.
- Found alongside, filed as #1485: the checker accepts undeclared type names in signatures
  (`f(r: Nope)` checks clean).

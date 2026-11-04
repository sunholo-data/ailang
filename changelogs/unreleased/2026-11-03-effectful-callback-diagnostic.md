### Fixed

- Effectful callbacks passed to pure combinators now produce a diagnostic naming the callback's effect restriction and performed effects before the preserved solver details. Effect mismatches omit the unrelated import-collision hint; ordinary type mismatches retain it. The CLI recommends same-module effectful variants discovered from the stdlib index, such as `std/list.mapE`.

### Fixed

- `ailang run --strict-bytecode` now selects the bytecode VM without requiring
  `--bytecode`, and fails loudly when bytecode execution cannot proceed (#1740).
- Added checked, equivalent-input `bench/vm_tail_walk/` consumption benchmarks and
  a separate quadratic construction contrast. Documented backing-storage retention
  from list-tail/take/drop sharing, which already shipped in v0.54.10 (#1827).
  Cons/append construction remains unchanged and parked on D-19.

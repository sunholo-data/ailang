### Fixed — named tests and properties keep effectful helpers (#1640)

- Named tests and forall properties retain all module-level helpers, including plain
  effect-free `export func` declarations (#1640). Unused effectful helpers remain
  available without being executed.
- Effectful test/property bodies report their pure-body contract, original construct
  and location, required effects, and the exported-entry capability workaround.
  Evaluator and bytecode fallbacks enforce the same contract; invalid helpers retain
  their module-level compiler error.

### Fixed — type checking no longer multiplies class constraints through nested lets (#1811)

- A let-bound scheme carried **every** unsolved class constraint in the inference context, not
  just the ones on its own generalized variables, and each use re-emitted them all. Constraint
  counts grew exponentially with let depth. A cold `ailang check --package .` of
  `sunholo/crew_core` peaked at 14.3 GB (`session.ail`: 21 s, 10 GB); `sunholo/crew_lab` passed
  17 GB still unfinished after 10 minutes. Both OOM-killed the 512 MiB registry validator, seven
  times between 10-11 and 10-13. Schemes now keep only constraints that mention a generalized
  variable, once each. The rest stay in the context and are still enforced. Measured after the
  fix, from an empty cache: crew_core 11 s / 149 MB, crew_lab 32 s / 201 MB. A depth-12 let-chain
  guard allocates under 64 MB; 25.7 GB with the old behaviour.

### Fixed — the registry validator says why instead of returning a bare 503

- `ailang check --package .` now runs under a 384 MiB RSS and 200 s wall budget inside the
  validator. A package that exceeds either is stopped and refused with HTTP 422 naming the
  budget and the observed value. Before, Cloud Run killed the whole instance and the publisher
  saw `HTTP 503: Service Unavailable`.
- `ailang publish` prints 422 refusals verbatim. On 502/503 it explains that the validator
  instance went away (usually memory) and that nothing was published.

### Added — `ailang test --max-recursion-depth N`

- Test bodies always ran at the evaluator's default 10,000-frame limit. A 10^4-step pure sweep failed with
  `RT_REC_003`, and the error told the user to raise `--max-recursion-depth`, which `ailang test` did not
  accept. The flag now matches `ailang run` and reaches every evaluator the test executor builds.
- This is a stopgap. The interpreter does no tail-call elimination (#1486), so tail-recursive loops still
  count toward the limit.

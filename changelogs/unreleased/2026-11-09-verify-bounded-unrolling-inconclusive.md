### Fixed — `ailang verify` / `ai-check` no longer report correct recursive code as a violation

Recursion is verified by bounded unrolling: `f` is expanded N levels deep and the
level below is an uninterpreted stub `f_0`. When Z3 found a model for that encoding,
the result was reported as `counterexample` / `✗ VIOLATION`, even though Z3 can
build such a model by picking any value for `f_0`, which no real run produces. A
correct list length with `ensures { result >= 0 }` was graded a verification
failure, `ai-check` exited 1, and the eval harness sent the made-up "counterexample
inputs" back to the model as a repair hint. All 8 counterexamples in the frozen
`v1.0` cohort (`contract_sorted_merge` ×4, `contract_bst_validate` ×4) were of this
kind.

A model found for a bounded-unrolled obligation is now a new status,
**`inconclusive`**: neither a proof nor a violation.

- JSON: `status: "inconclusive"`, `reason_code: "bounded_unrolling_inconclusive"`, a
  `reason`, and `bounded_depth`. The new top-level `inconclusive` counter appears in
  `ai-check`, `verify --json` and `verify --package --json`. The candidate model is
  left out unless `--verbose` is set.
- Human output: `? INCONCLUSIVE (bounded: depth N)`.
- Exit codes: `ai-check` and `verify` exit 0 when the only problem is an
  inconclusive result, the same as for a skip. `verify --strict` (file and
  `--package`) now exits 1 on one.
- Eval harness and observatory: `verify_inconclusive` is banked. `isVerifiedSuccess`
  requires it to be 0, so an undecided run never counts as a verified success, and
  `isVerificationFailure` counts it. Following Mark's D-32 ruling (keep strict, but
  name it), `chains stats --cost-per-verified-success` publishes
  `verification_inconclusive`, a named subset of `verification_failures`. The KPI
  number itself does not change. Inconclusive results no longer trigger a
  `VERIFY_COUNTEREXAMPLE` repair turn. The agent prompt says INCONCLUSIVE is not a
  bug report.
- Known cost: a real bug in a recursive function is also reported `inconclusive`
  until witness replay exists. Genuine counterexamples on non-recursive functions
  are unchanged.

Design: `design_docs/implemented/v0_57_5/m-verify-bounded-unrolling-false-counterexample.md`.

### Fixed — lane evals run each benchmark under its own sandbox

`ailang eval-suite --policy-file` passed the operator's policy through unchanged,
so every run of a lane eval shared one `fs_sandbox`. Lane tools resolve paths
against the sandbox root, so a relative `benchmark/solution.ail` landed in a
stray file at the shared root. The 2026-10-01 pi-vs-motoko A/B graded the
untouched template on 8 of its 12 wrong-answer rows, and concurrent runs read
each other's files. The 2026-09-16 pi lane A/B had the same setup.
- New `executor.MaterializeRunPolicy` writes a per-workspace copy of the policy
  with `${WORKSPACE}` substituted, read-only, outside the workspace, at a path
  unique to each run.
- The eval harness and motoko's lane canary use it.
- A lane policy without `${WORKSPACE}` is refused by name.

### Fixed — step-budget-exhausted runs are graded and costed

A run that used up its step budget was returned as a crash: banked with zero
tokens, $0 and no grading. It is now graded like any other run, with its tokens
(and cost recomputed from them) recorded and `finish_reason=step_exhausted`
kept for categorisation. The A/B under-counted motoko by $2.56 across three such
rows, one of which had written a 10 KB solution.

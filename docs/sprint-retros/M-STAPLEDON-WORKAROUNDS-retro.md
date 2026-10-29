# Sprint Retrospective: M-STAPLEDON-WORKAROUNDS

## Outcome

All seven milestones implemented: native VM maps, stable lock metadata, named-callee
application errors, strict-engine regressions, and an actionable downstream register.
No game migration or closure of #1576 is claimed. Planned effort was four working days
(about ten focused hours); this agent run took about 48 minutes including environment
setup and validation retries. This is not a human velocity measurement.

## Execution

The VM M1–M4 chain ran sequentially. Independent M5 (lock writer) and M6 (diagnostics)
ran with two workers; one independent evaluator reviewed the combined implementation.
No implementation-file merge conflicts occurred. The planner artifacts were recovered
from its remote branch while preserving the execution checkout's branch and local work.
Per-milestone estimates and measured gross changed LOC are in the sprint JSON.

## Verification and friction

- Failing regressions preceded the implementation. Native pure map comparisons ran
  twenty repetitions; exact reporter, full-family and map-result boundary cases pass.
- Go existed outside PATH. The cold CLI build, SQLite compilation and lint installation
  dominated early setup; partial parallel builds then exceeded the 8 GiB RAM-backed
  environment. Serialized gates and cleanup of interrupted temporary outputs resolved it.
- Debian procps and tini were installed into a temporary directory without root changes.
  They provided missing process inventory and reaped orphan zombies under the container's
  PID1. No unrelated runtime-policy code was changed.
- A 512 MiB cap slowed a deep-recursion command test into its deadline. The uncapped
  changed-package run passed all five packages in the scoped gate. The parallel core
  pipeline was killed; a separate full serial core run with a 1 GiB cap passed, including
  pipeline (62.821s) and effects cleanup. Both gate components have successful evidence.
- The examples gate caught aggregate manifest counts missing the new example. The counts
  were corrected; all examples and import metadata validation then passed.
- Lint finished with zero issues. Formatting, boundary, file-size and changelog gates pass.
  The full repository suite remains CI's responsibility, per the approved sprint plan.

## Next run

Prepare PATH and the container's process reaper before tests. Run memory-heavy gates
serially, with separate memory budgets for command tests and core pipeline tests. Keep
milestone commits frequent and wait for command exit status before declaring a gate green.
No speedup claim is made because cold setup and environment retries dominated wall time.

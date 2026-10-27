### Fixed — cloud evaluator verdicts now reach the parent's approval

The shared finalizer (`FinalizeTaskCompletion`, the cloud completion path) gains an
`evaluation` effect: an `evaluates_parent` agent's verdict is written to its parent task's
approval, as the daemon path already did. A failed or blocked evaluator records
`UNAVAILABLE` with its reason; a missing `parent_task_id` fails the effect on the ledger.
The cloud completion now carries the transcript tail as the result output, so the verdict
line can be read. Before, no cloud evaluator verdict ever reached an approval: on 2026-10-27
none of the nine pending prod approvals carried an evaluation, including one scored PASS 71
on 2026-10-12. (`internal/coordinator/task_finalize_evaluation.go`)

The sprint-evaluator skill now ends its coordinator output with the
`EVALUATION_VERDICT: PASS|FAIL score=<n>` line the coordinator parses. It used to specify
`EVALUATION_RESULT: pass|fail`, which the parser does not read; the 2026-10-27 prod run
followed the skill and its FAIL 84 review recorded no verdict.

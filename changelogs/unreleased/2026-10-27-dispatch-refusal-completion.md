### Fixed — a task refused at dispatch now reports a failed completion

A cloud task that fails permanently at dispatch (`ErrDispatchPermanent`) now posts a
`completion` with `status: failed` and `source: dispatch` to the agent's inbox, as the
stale-task detector already did for timeouts. Before, only the executor job published a
completion, so a task that never reached one failed with only a thread post: on
2026-10-16 five sprint-evaluator tasks refused with 402 "credit budget blocked" never
appeared in the inbox or in `ailang messages activity`, and the evaluator sat dead for
11 days unnoticed. (`internal/coordinator/daemon_tasks_exec.go`)

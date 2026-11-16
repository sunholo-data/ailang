### Fixed — a cloud task claimed during coordinator shutdown no longer strands as queued (2026-11-16)

`dispatchTasksCloud` claims a task (`MarkTaskQueued`) before it publishes and dispatches it. When
SIGTERM cancelled the daemon context mid-tick, the dispatch failed on that cancellation and the
re-queue (`ResetTaskToPending(d.ctx, …)`) failed on the same cancelled context, silently. The task
stayed queued with no Cloud Run job behind it until the 3-h stale detector failed it: task-d82dc6c8
(10-01) and task-18a82186 (10-16), failed at 3h0m and 4h56m. Every dispatch-side re-queue now goes
through `requeueClaimedTask`, which runs on `context.WithoutCancel` with a 10 s bound and logs a
failure instead of discarding it. Fleet ruling D-FLEET-40 (A).

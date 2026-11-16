### Fixed — the cloud codex credential lane no longer frees while a failed task's Cloud Run retry may hold the lease (2026-11-16)

`credentialLaneHolder` counted only queued and running tasks, so the lane freed on a codex task's
first `failed` completion. The executor jobs run `maxRetries: 1` and every `failed` completion exits
non-zero, so Cloud Run retries the same execution, which takes the subscription lease again. The
coordinator never sees the retry: its completion is skipped because the task is already terminal. On
2026-10-16 the retry of task-d61964fa re-acquired the lease 50 s after attempt 0 released it, and the
codex tasks dispatched 3m47s and 8m49s later both died on "credential already owned". A codex task
that failed now keeps the lane for its agent's timeout plus 15 minutes from `completed_at` (option A2
of fleet ruling D-FLEET-39). The read uses the existing `(status, created_at DESC)` Firestore index.
No lease is ever taken over.

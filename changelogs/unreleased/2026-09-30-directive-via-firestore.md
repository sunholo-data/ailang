### Fixed — a large task request can no longer get stuck in the dispatch queue

The coordinator used to pass the task request to the Cloud Run Job as the `AILANG_DIRECTIVE` env var, and Cloud Run caps one env value at 32,768 bytes. Daneel requests carry fetched source pages and the repo's open PRs, so a 43 KB request (task-68771ff3, 2026-09-30) was refused with `InvalidArgument` on every dispatch. It was retried 18 times over 2.5 hours, and its status said only `pending`.

- The request text now goes in a Firestore document, `task_directives/<task id>`, written before the job starts. The dispatcher sets `AILANG_DIRECTIVE_SOURCE=firestore` and the job reads the document by `AILANG_TASK_ID`. If the document is missing, the job fails; it never falls back to an empty or inline request. Requests up to 900 KB are accepted.
- During rollout, requests of 16 KB or less are still also sent inline, so older job images keep working.
- A dispatch that cannot succeed on any retry now **fails the task** and posts the reason to its thread (`ErrDispatchPermanent`). This covers an env value over the cap, an oversized request, a request Cloud Run rejects as `InvalidArgument`, and an agent whose `executor_variant` does not match its provider. Transient failures still requeue.

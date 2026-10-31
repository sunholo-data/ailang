# Iteration 75 — draft records, as handed to the judge

Controller: `claude:claude-opus-5-5`. Base `ddf4ec6a19c5e53a4e1585db3bc32a4e8a6feda8` (origin/dev, 2026-10-31T08:14:07Z).
Prod log extracts (Cloud Logging, project `ailang-multivac`) are in this directory.

## Record 1 — `coordinator:job-retry-success-dropped-after-failed-completion` (attended, 10-12) folds into D-FLEET-39

**Claim 1.1 (real at HEAD).** A completion that arrives after the task is terminal is skipped, by design.
`internal/coordinator/pubsub_completion_handler.go:117-121` returns early on `IsTerminalStatus(task.Status)`
("already in terminal state %q, skipping"). `internal/coordinator/completion_binding.go:64-67` names
"a retried job reporting after the first attempt" as one of the duplicates that path is meant to skip.
So a Cloud Run retry whose attempt 0 published `failed` cannot land a later `completed`.

**Claim 1.2 (no fix since the ticket).** No commit on origin/dev since 2026-10-12 touching
`internal/coordinator` or `cmd/ailang` mentions retry/attempt/supersede/terminal state for this path
(`git log origin/dev --since=2026-10-12 -i --grep=...`: 3 unrelated hits).

**Claim 1.3 (completions carry no attempt index).** The only `CLOUD_RUN_TASK_ATTEMPT` reference in Go is the env
passthrough list `internal/executor/envpolicy.go:79`. An attempt-aware supersede needs that field added to the
completion payload first.

**Claim 1.4 (same cause as D-FLEET-39, not a new decision).** D-FLEET-39 is "attempt 0 publishes `failed`,
the Cloud Run retry still owns the work". This ticket is the other half of the same event: the retry's success is
then dropped (10-12 task-a87f9332 → PR #1771; 10-16 task-d61964fa → PR #1849, per iteration 74). The fix paths
(`internal/coordinator/**`; or `max_retries=0` on the executor jobs, which is multivac terraform) are outside fleet
scope. **Proposed edit:** append a re-measure note to D-FLEET-39 naming this ticket as covered, and widen its
option A from "a codex task" to "any cloud task" (the drop is provider-independent). No new decision ID.

**Claim 1.5 (retries have value, so `max_retries=0` is the worse option).** In the 10-12 and 10-16 cases the retry
produced the PR. In the six SIGBUS cases below the retry also died, so retries bought nothing there, but cost
nothing either (no duplicate work landed).

## Record 2 — `executor:3h-timeout-from-queue-and-unsliced-sprints` + addendum `executor:timeout-loses-work-and-may-never-start` (attended, 10-12) → new D-FLEET-40

**Claim 2.1 (ask (a) is true in code but not causal).** `internal/coordinator/stale_task_detector.go:232-247`
ages a task from `StartedAt`, else `QueuedAt`, else `CreatedAt`; `store.go:64-69` says cloud tasks never reach
`running` (`MarkTaskRunning` is local-only), so the cloud clock is the claim time. But for all 6 named tasks, prod
logs show `Cloud dispatch` → executor `execute-job: starting task` in 16–55 s (4da1bab2 16 s, 85d95401 40 s,
538fd63d 55 s, e546fce0 50 s, 50131b05 27 s, 06dc18ea 15 s). None waited in a queue.

**Claim 2.2 (addendum's "never started" hypothesis refuted, 6/6).** Every one started, and every one ran TWICE:
Cloud Run attempt 1 started 12–31 min after attempt 0.

**Claim 2.3 (actual cause: SIGBUS, 12/12 attempts).** Every attempt of all 6 executions ended
`Container terminated on signal 7.`, and each execution then logged `ERROR`, 27–63 min after dispatch.
Two executor transcripts say it themselves ("Concurrent build tooling exceeded the container's memory limit").
Several attempts were installing tools into `/tmp` (apt-get download make, zig, tini). This is
`crash:sprint-executor:sigbus-ram-tmp-full-make-test` (D-FLEET-23 ask 2).

**Claim 2.4 (the 3 h is dead time, D-FLEET-23 ask 3).** The coordinator learned of each death only from the stale
detector at 3h0m–3h1m: 2h03m–2h37m after the execution had already ended in `ERROR`. Nothing reports a failed
Cloud Run execution to the coordinator. That is D-FLEET-23 ask 3 verbatim.

**Claim 2.5 (no recurrence since 10-09, with exposure).** Since 10-01, `Container terminated on signal` appears
18 times in Cloud Run job logs, all on 10-03..10-09, all `ailang-agent-executor-codex-go`. Zero after 10-09 against
79 `execute-job: starting task` lines 10-10..10-19 and 35 since 10-20. The stale detector fired 8 times since 10-01:
the 6 named, one on 10-01 (task-d82dc6c8, not inspected), and one on 10-17 (task-18a82186).

**Claim 2.6 (ask (e) has one real instance).** task-18a82186 (design-doc-creator, 10-16 19:39Z) never started:
`daemon_tasks_exec.go:186: Failed to publish task ... to Pub/Sub: ... context canceled`, then nothing until the
stale detector at 4h56m. So a never-started task does time out silently; one instance in 31 days.

**Claim 2.7 (three asks are ghosts).** (d) focused tests, never `make test`: #1770 `dbe4b26a3`
(2026-10-12T10:16:51Z, 46 min after the ticket), `sprint-planner/SKILL.md:336`, mirrors identical.
(b), planner half (size milestones ~60–90 min, split larger): same line, same commit. (c), executor half:
commit per milestone #1688 `face88a3f` (10-09) and push every milestone + resume on retry #1905 `3f9a0c092` (10-20).

**Claim 2.8 (what is left is coordinator code, outside scope).** Unbuilt: (a) report queue wait separately /
start the clock at start; (b) coordinator auto-chains one task per milestone; (c) coordinator pushes partial work on
timeout; (e) alert/re-queue a task that has not started in N minutes; plus the measured gap from 2.4 (a Cloud Run
execution ending in `ERROR` should fail the task promptly). All are `internal/coordinator/**`.

**Proposed D-FLEET-40 (OPEN).** Recommendation **A**: fold the residual into D-FLEET-23's owner (attended/V1), since
the measured incident is D-FLEET-23 asks 2 and 3; the one actionable new item is (e), as task-18a82186 shows: fail
or re-queue a task whose dispatch publish failed. Resolve both executor tickets citing `dbe4b26a3` and `3f9a0c092`
once that owner accepts the residual. **B**: keep both tickets open as their own coordinator design. **C**: no change.
Default: C.

## Not taken this fire

`executor-image:no-make-no-cc-cgo-tests` (attended, 10-08) was measured but not worked: the skill half is a ghost of
#1905 (`unattended_runs.md:58-64`, "never install toolchains ... leave that verification to CI", which deliberately
rejects the ticket's `CGO_ENABLED=0` option), and `docker/Dockerfile.agent-codex-go` installs `make` and `gcc` while
`docker/Dockerfile.agent-go` and `docker/Dockerfile.agent-pi-go` do not. `docker/**` is outside fleet scope.

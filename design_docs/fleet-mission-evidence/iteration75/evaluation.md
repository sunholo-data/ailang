# Iteration 75 — independent evaluation (sonnet, Agent tool, round 1)

Judge: `sonnet` via the Agent tool, foreground, read-only on the pin (`ddf4ec6a1`). 38 tool uses, 374 s, 122,475 subagent tokens.
Subject: [draft-records-as-judged.md](draft-records-as-judged.md).

## Verdict

| Record | Score | Verdict |
|---|---|---|
| 1. `coordinator:job-retry-success-dropped-after-failed-completion` folds into D-FLEET-39 | 78 | PASS-WITH-EDITS |
| 2. Executor 3h-timeout tickets → D-FLEET-40 | 72 | PASS-WITH-EDITS |

The core causal chain held: SIGBUS kills each attempt, the retry dies too, and the coordinator learns only at 3h. No finding was BLOCKING on either record's central claim. These were BLOCKING-IF-LEFT, because each is a factual error that would go into the ledger:
- Record 1 undercounted the drop.
- Claim 2.6 said one instance where there are two.
- Claim 2.5 used the wrong exposure denominator.
- Claim 2.3 got its minutes wrong.

## Findings (summarised; the controller re-verified each marked ✓ first-party)

**Record 1**
- 1.1 ✓ The terminal skip is the path a successful retry hits. Prod example, task-18020ec1:
  - `10:25:08 received completion (status=failed)`
  - `10:28:12 … (status=completed)`
  - `10:28:12 … already in terminal state "failed", skipping`
- 1.2 No fix since 10-12. Only `efcd7fe80` and `269e54616` touched these files, and both are unrelated.
- 1.3 `pubsub.TaskCompletion` (`internal/pubsub/topics.go:66-140`) has no attempt field. `ChainLinkIndex` is the model-chain link, not the Cloud Run attempt. A simpler fix needs no attempt index: let `completed` or `no_changes` replace a `failed` that a completion wrote (not one the stale detector wrote).
- 1.4 ✓ **BLOCKING-IF-LEFT.** The drop is recurring and mostly on pi. Nine attempt-1 successes were skipped since 10-01:
  - pi (7): task-f86a7ce4, task-50c72755 (10-06), task-60af3acf (10-09), task-98f9f628, task-f0a45a3f (10-12), task-b886c336 (10-19), task-18020ec1 (10-21).
  - codex-go (2): task-a87f9332 (10-12), task-d61964fa (10-16).
  - Attempt-0 causes vary: `signal: killed`, `pi exceeded hard timeout (2h0m0s)`, lease contention.
  - Controller correction: the judge's suggested wording counted the two codex cases as additional to the nine. They are part of the nine; the controller's own recount gives 9 distinct tasks in total.
- 1.5 ✓ The job config, read first-party: `maxRetries: 1`, task timeout 14400 s, 8Gi / 2 cpu, on both codex-go and pi. `max_retries=0` would have lost all nine. "Cost nothing" is wrong: each doomed SIGBUS retry cost 12–31 min of execution, plus codex quota, plus the lease.
- 1.6 D-FLEET-39 option A2 (the lane counts recently failed codex tasks) does NOT fix the drop. Only option A, or a supersede, does.

**Record 2**
- 2.1 ✓ Re-derived dispatch→start, both attempts, and signal 7 for 85d95401, 06dc18ea and 4da1bab2. The stale detector's `age` arithmetic places `queued_at` exactly at the dispatch line.
- 2.2 **BLOCKING-IF-LEFT.** Dispatch→execution `ERROR` is 24–58 min, not 27–63:
  - 06dc18ea 24, e546fce0 27, 85d95401 34, 50131b05 38, 4da1bab2 40, 538fd63d 58.
  - The stale firing came at 3h0m–3h1m44s.
  - Dead time is 2h02m–2h37m.
- 2.3 ✓ "Nothing reports a failed execution" holds for this repo:
  - There is no Executions API use.
  - `internal/dispatch/cloudrun/dispatcher.go:543` discards the `RunJob` operation.
  - Hedge it to "nothing in this repo", since terraform, Eventarc and log sinks are not visible. The signal exists: Cloud Logging has `Execution … has failed to complete, 0/1 tasks were a success.`
- 2.4 **BLOCKING-IF-LEFT.** The right exposure is codex-go sprint-executor attempt-starts:
  - 54 starts on 10-03..10-09, with 18 SIGBUS.
  - 20 starts since 10-10, with 0 SIGBUS.
  - The all-job 114 was the wrong denominator.
  - Every `ReplaceJob` on 10-03..10-10 kept 8Gi, no volume and `maxRetries: 1`; only the image digest changed.
  - The last SIGBUS (10-09 09:18Z) predates #1770 and #1905, so do not attribute the drop to them. The cause is unknown.
- 2.5 ✓ **BLOCKING-IF-LEFT.** Ask (e) has two instances, not one:
  - task-d82dc6c8 (10-01) and task-18a82186 (10-16), both design-doc-creator.
  - In each case `daemon.go:308 Received signal: terminated` came microseconds before `Failed to publish task … context canceled`.
  - At HEAD, `daemon_tasks_exec.go:204-207` re-queues with `_ = d.taskStore.ResetTaskToPending(d.ctx, …)` on the same cancelled context, so it fails silently.
  - Only these two `Failed to publish task` lines exist since 09-01.
  - All 8 stale firings since 10-01 are now explained: 6 SIGBUS and 2 shutdown strands.
- 2.6 The ghosts are verified, with one overstatement:
  - (b) only the split rule landed. Per-milestone task emission and chaining are unbuilt.
  - (c) the coordinator half is infeasible in the cloud, because the cloud coordinator has no clone (`topics.go`).
  - #1905's push-every-milestone has not been exercised against a crash.
- 2.7 Do not file a row whose recommendation is "fold into D-FLEET-23's owner":
  - Append the SIGBUS and dead-time measurement to D-FLEET-23 instead.
  - Keep D-FLEET-40 for what is new and unowned: the shutdown-strand bug, per-milestone chaining, and a per-ask disposition.
  - Resolve a ticket only once every ask has a recorded home.
  - "(a) not causal" is a population statement, not a 6-sample inference: the 6 tasks are the ticket's whole set, and all 8 stale firings are explained without a queue wait.

The judge did not verify the multivac terraform, the image digest contents, any existing alert on `has failed to complete`, or the `executor-image` note.

# Round 2 (same judge, resumed via SendMessage; 2 tool uses, 44 s, 134,780 subagent tokens)

| Cell | Score | Verdict |
|---|---|---|
| D-FLEET-23 (appended re-measure) | 80 | PASS-WITH-EDITS |
| D-FLEET-39 (appended re-measure) | 82 | PASS-WITH-EDITS |
| D-FLEET-40 (new row) | 76 | PASS-WITH-EDITS |

All four round-1 BLOCKING-IF-LEFT findings are fixed: the 9 drops, the 24–58 min arithmetic, the exposure denominator, and the two strands.

Spot-checks that held:
- The strand gaps are 34 µs and 26 µs.
- `daemon_tasks_exec.go:206` is as cited.
- pi `maxRetries: 1` / 14400 s, read first-party by the judge.

NON-BLOCKING edits, all applied by the controller:
1. D-FLEET-23: "only the image changed" became "the image digest changed in each, other fields not diffed". "No volume" became "no scratch/`emptyDir` volume", because the job mounts gcsfuse `/artifacts`.
2. D-FLEET-40 title: "three asks are built" became "one ask is built and two are half-built".
3. D-FLEET-40 option A: the general N-minute alert in (e) and "expose queue wait in `coordinator list`" are now parked explicitly as non-goals. The answer to `coordinator:cloud-lane-capped-by-local-worktree-limit` ask (c) is recorded: both tasks started 14 s / 40 s after dispatch, and nothing held them. The controller confirmed that ticket's ask (c) text.
4. D-FLEET-39: the stale "(not re-measured)" and "config not inspected" phrases were updated.
5. Placement in the evidence cell: `scripts/mission_decisions.sh --check` is valid with 39 rows. D-FLEET-17 is the precedent for appending re-measures there.

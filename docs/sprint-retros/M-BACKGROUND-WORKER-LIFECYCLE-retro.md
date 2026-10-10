# Sprint Retrospective: M-BACKGROUND-WORKER-LIFECYCLE

## Scope and measurement

The approved sprint implements owned worker cleanup and explicit managed/async cancellation in the runtime. Local completion is separate from dev delivery, a supporting official release, and the pinned consumer feasibility retest. Parent/provider request cancellation in #231 remains a separate problem.

Plan: eight engineering days, 3,400 added production/test lines, using an explicitly assumed 425 LOC/day. Actual implementation at source head `b2be99ecc`: **4,682 added lines and 420 deleted lines**, measured from approved-plan commit `956458c36` across `internal/`, `cmd/`, `std/` and `examples/runnable/`. Documentation, state, manifest and interface goldens are excluded. Scope grew 37.7%, principally for real OS lifecycle evidence, concurrency repairs, backend dispatch and the discovered stream EOF regression. This agent execution's elapsed time is not a measurement of human engineering velocity.

| Milestone | Estimated added LOC | Actual added LOC | Recorded execution interval | Result |
|---|---:|---:|---|---|
| M1 ownership |500|836|17:11:57–17:26:59 UTC|PASS, later trace regression correction included|
| M2 managed workers |650|762|17:20:00–17:26:59 UTC|PASS|
| M3 async workers |650|1,131|17:20:00–17:26:59 UTC|PASS|
| M4 hosts/backends |850|1,427|17:26:59–17:43:22 UTC|PASS, later API debug corrections included|
| M5 regression/evaluation |750|526|17:43:22 UTC through final evaluation|PASS: independent full gates and substantive review|

LOC groups follow primary responsibility: supervisor/context/policy/error helpers in M1; managed operations in M2; stream operations and EOF/borrowed-resource regressions in M3; hosts/VM/trace/platform behavior in M4; actual AI/PID/PTY/conflict fixture tests in M5. Shared-file corrections are counted under that file's primary group. Milestone intervals overlap; review corrections continue after initial milestone checks.

Actual elapsed calendar wall time from sprint creation to local gate completion: 49.1 minutes (0.034102 calendar days). Five milestones passed; final artifact audit records the formal Round2 verdict.

## Execution and independent review

Three implementation waves established ownership, then managed and async workers in parallel, then host/backend integration. Maximum active concurrency was the root plus three agents. The independent evaluator authored no implementation and enforced two frozen-source review rounds. No controlled sequential run was performed, so no speedup factor is claimed.

Round 1 rejected full tests and manifest verification. It correctly caught lost API debug sinks after context cloning, unwanted no-worker trace receipts, and stale example totals. Narrow fixes restored logging/configuration inheritance and trace compatibility; focused race tests also exposed shared fallback Debug drain and CLI test-buffer races. Round 2 reused no failed full-test result; it reran full tests and all affected gates. The final comment-only line reduction is explicitly recorded with both source heads, rather than treating it as a semantics change requiring another suite run.

## Friction and corrective decisions

- The original checkout contained another workstream's dirty/conflicted files. All changes stayed in an isolated checkout under `.claude/worktrees/`; no original work was discarded or continued. A checkout under TMPDIR would make loader temp-path tests fail for unrelated reasons.
- Exact PID acknowledgements replaced timing assumptions. Probes reject inspection errors and zombies, and failure cleanup terminates only the recorded worker. Provider-free AI handlers and real PTYs made shutdown/reaping and terminal restoration measurable without inference spend.
- Admission during shutdown, delayed release across REPL reset, signal watcher defer ordering, post-deadline source stop, and concurrent trace access needed explicit coordination. One owner protects each shared lifecycle file; independent worker responsibilities prevented overwriting changes. No destructive merge or reset was needed.
- Live conflict fixtures exposed `selectEvents` consuming/discarding ready output while checking closed sources. A concurrent 5,000-event regression demonstrated loss before closed-channel tracking and complete delivery afterward.
- Adding valid example entries is insufficient: aggregate manifest statistics must also change. Final verification exercises the manifest gate, showcases and baseline corpus.
- Engine-owned context cloning requires all shared policy/sinks to be configured before cloning while request accumulators remain independent. Existing logging and trace controls caught compatibility regressions that new worker-only tests could not.

## Delivery and next work

Keep this branch local until the user's dev-delivery step. Publish a supporting runtime afterward, then rerun the consumer cleanup gate with its pinned official version/commit and exact PID evidence before resuming reply inbox and parallel crew reactions. Move the design and companion plan together into that actual release's implemented directory when shipped. Windows has leader-only host cleanup and typed unsupported explicit cancellation; its compile checks are not runtime supervision evidence. JS/WASM typed unsupported results were executed under Node. Detached descendants, SIGKILL/power loss and remote provider cancellation are outside this guarantee.

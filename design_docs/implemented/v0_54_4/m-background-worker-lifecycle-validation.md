# M-BACKGROUND-WORKER-LIFECYCLE validation

Status: locally complete; independent full gates and substantive review pass. Unreleased, no push. Source baseline `37001e839` (stdlib v0.54.1). User approved design/planning/execution on 2026-10-10.

## Before and after: exact process identity

The acknowledged-start probes reproduce the two failures against the official checksummed v0.54.0 binary (commit `361caeda13830b0a3faa11919cc2b5d37a14dde1`). The child's pinned runtime executes a provider-free Clock effect; the probe records its exact PID before sending quit. Managed PID **41009** and async-selector PID **41098** remain alive, PPID1, stateS after host exit. Each probe fails its explicit absence assertion and terminates only its recorded child. The stdlib for these repeat probes is the unchanged v0.54.1 source baseline, compatible with the v0.54.0 binary; the original design records the all-v0.54.0 reproduction too.

The sprint CLI/batch matrix passes **35** cases. Both worker paths are covered under native evaluator and strict bytecode; normal/error/exit(7), batch normal/error/exit, non-TTY SIGINT/SIGTERM, evaluator budget exhaustion, and original `selectEvents` handler-false exit. Acknowledgement files replace startup sleeps. `ps` errors fail the test; an existing zombie PID also fails. Recorded sample outcomes:

| Path | Host PID | Worker PID | Exit | Shutdown | Result |
|---|---:|---:|---:|---:|---|
| Managed evaluator normal |31124|31273|0|254ms|Worker absent|
| Managed evaluator exit(7) |31306|31307|7|253ms|Worker absent|
| Async strict VM SIGTERM |31502|31503|143|3ms|Worker absent|
| Async selector handler false |31523|31524|0|3ms|Worker absent|

Timing measures acknowledged-start quit/signal through host termination; every case stays below the two-second deadline plus500ms scheduling tolerance. Strict bytecode rejects annotated budget frames explicitly, so that budget case is exercised through the evaluator. This does not claim VM budget-frame implementation.

## Focused evidence

| Area | Actual verification |
|---|---|
| Supervisor |Owner hierarchy; policy-preserving fresh registries; shared budget views; closing rejects admission; admitted Start/teardown race; one two-second deadline across20 stalled request owners; blocked stop reports the actual public handle's deadline without repeating source stop; natural wait failure survives release|
| Managed worker |Focused race suite repeated3times: queued600-byte drain, blocked4MB write, concurrent write/close/cancel, actual Wait, exact descendant PID probe, foreign/stale/bystander controls|
| Async worker |Async cancellation/lifecycle race suite repeated10times: final partial chunk, natural single Wait, blocked/full queue, inherited descendants including leader-exits-first, source retention for queued data, Stream/Process authority/budget and isolation|
| AI boundary |Real AI effect dispatch in owned helper subprocess enters an injected blocking handler and acknowledges its exactPID; both cancellation operations return only after owned tasks/Wait join; second worker remains active. Zero provider/network AI calls|
| Scale/resources |20 simultaneously blocked AI helpers across both paths share one shutdown deadline;10 repeated mixed-worker executions return descriptor and goroutine counts to baseline tolerance and all Done/Wait states are complete|
| Host lifetimes |Normal success control before injected cleanup failure proves CLI success becomes1 and batch success becomeserror. Primary evaluator errors/exit7/callback panic survive cleanup. Engine close during actual mocked AI dispatch joins active request worker without waiting for application call/cache mutex; caller base remains usable. REPL reset and session closure tested|
| Terminal |11 evaluator/strict VM worker PTY cases: acknowledged exact workerPID, PID absence after normal/error/exit/budget/signals, restored termios/cursor/screen,130/143 retained. Original terminal PTY controls also exercised|
| Borrowed resources |Reader/lease race suite repeated20times; blocked borrowed stdin remains open and retains lease until actual read/release completion; connection adapter shutdown leaves transport open; pending readers reported separately|
| WebSocket |19 existing WS cases including concurrent sessions and1001 shutdown, plus2 new actual-client handler-error/borrowed-source cases proving1011 preserved|
| Trace |Actual concurrent record/snapshot/config races reproduced first; synchronized collector and deep independent snapshots/observer payloads pass full trace race suite. Concurrent evaluation plus worker shutdown emits one completed receipt, repeated close emits none extra|
| Backend |Full bytecode/compiler/VM/Go-generation tests and repeated race checks pass; both showcases outputtrue under `--bytecode --strict-bytecode`; original13 effect indices preserved; only supported scoped handles/ADTs cross native boundary; generated Go rejects cancellation explicitly|
| Platforms |Final Windows effects/runner test binaries cross-compile and production effects/runner/runtime/embed/REPL build; Windows runtime execution not available. Both JS/WASM cancellation operations actually run under Node and return typedUnsupported; missing capability is denied. WASM trace build passes|
| Interfaces/examples |Builtin golden adds exactly2 signatures. Explicitly re-freeze only approved process/stream interfaces; all48 stdlib interfaces verify. Both showcases check/inline-test/native/strict execution. All5 design conflict fixtures run live; terminal_keys q/Escape runs under real PTY in both backends|
| EOF regression |Live stream_process_source exposed closure probe consuming other-source output. New concurrent5,000-event test failed with2,543 delivered, then passed3race runs with all5,000 delivered after closed-channel tracking|

## Reproducible repository tests

Tests are committed with the sprint, rather than depending on temporary evidence files:

- `internal/effects/worker_owner_test.go`, `worker_ai_lifecycle_test.go`, `worker_trace_lifecycle_test.go`.
- Managed/async cancellation/lifecycle, stdin/source cleanup, closed-source output and platform cancellation tests in `internal/effects`.
- `internal/runner/worker_process_exit_test.go`, `worker_terminal_pty_test.go`, `worker_conflict_fixtures_test.go`, `worker_lifecycle_test.go`.
- Request/engine/REPL tests in `internal/runtime`, `internal/embed`, `internal/repl`; actual WebSocket1011 tests in `internal/apiserver/worker_ws_lifecycle_test.go`.
- Scoped backend tests in bytecode, VM and Go generation; trace concurrency/mutation tests in `internal/trace`.

Run native lifecycle controls outside the restricted sandbox: PID inspection, PTYs and local WebSocket listeners require local OS access. Test probes fail on OS inspection errors rather than claiming process death. Run Windows compilation separately and JS/WASM tests with Go's Node test executor; compilation alone is not a runtime-result claim.

## Final gates

Independent round 1 rejected the frozen implementation at `99bda5b8a`: full tests found missing API debug lines after engine context cloning and irrelevant shutdown trace events in no-worker programs; example verification found stale manifest statistics. Lint and file-size gates passed. The fixes configure the API sink and policy before cloning, serialize the shared fallback debug drain, preserve existing no-worker trace goldens while retaining completed-worker receipts, and correct only the three manifest totals. Focused regression tests reproduce the failures before each correction and pass afterward. The CLI debug polling test also needed synchronized stdout capture to make its existing concurrent readiness polling race-safe.

| Gate | Final local result |
|---|---|
|Clean baseline full make test/lint|PASS|
|Final focused lifecycle/backend/trace/WS races and PID/PTy controls|PASS|
|Final make fmt-check, check-boundaries, check-changelog|PASS|
|Final verify-stdlib|PASS:48interfaces|
|Independent full make test|PASS: exit0 at Round2|
|Independent make lint/check-file-sizes/verify-examples|PASS:0lint issues;800line limit;243examples pass/0fail/9skip;manifest231total/223working;0module drift|
|Independent evaluation|PASS95/100: final Round2 source and artifact audit; `.ailang/state/evaluations/eval_M-BACKGROUND-WORKER-LIFECYCLE_round_2.json`|

## Delivery handoff

The local sprint retained commits on `sprint/background-worker-lifecycle` until independent gates passed. The user subsequently authorized pushing to dev and releasing; see Release delivery below. Publish a supporting runtime after that delivery, then rerun the consumer cleanup feasibility gate with its pinned official version/commit and exact PID evidence. Resume reply inbox and parallel crew reactions only after that published-runtime gate passes. Local subprocess termination does not fix parent/provider cancellation issue#231 or guarantee remote inference/billing stops. POSIX descendants that deliberately detach, hostSIGKILL and power loss remain outside the guarantee.

The local sprint originally retained the design and companion sprint plan under `planned/` without assigning a release. The authorized release preparation now archives both with this validation record in `implemented/v0_54_4/`.

## Final source and regression provenance

Round2 evaluates source head `b2be99ecc` after tests began at `d7884e3f0`. The intervening commit collapses exactly one API constructor comment to stay within the 800-line file-size gate; no executable token or test changed, and the evaluator independently verified that diff. Full tests ran again after the Round1 source corrections, with retained logs `/private/tmp/worker-independent-round2-{test,lint,sizes,examples}.log`. Formatting, boundaries, changelog and all48stdlib interfaces were rechecked after the corrections (`/private/tmp/worker-round2-extra-gates.log`).

An unchanged source-baseline executable at `37001e839` passed241examples with9skips. Final example verification passed243with9skips: all250common file statuses match, no file was removed, and the only additions are the two cancellation showcases. This is a status/output-fixture regression control, not a claim that a full AST differential tool ran. The existing stale `lambda_expressions.ail` manifest warning remains nonfatal; no new module drift exists.

The round1 rejection and final round2 audit are preserved under `.ailang/state/evaluations/`. The [sprint retrospective](../../../docs/sprint-retros/M-BACKGROUND-WORKER-LIFECYCLE-retro.md) records elapsed time, source LOC and integration corrections.

## Release delivery

On 2026-10-10 the user authorized delivery to dev and a new release. These completed records move together into `implemented/v0_54_4/` as part of release preparation. The runtime is included in v0.54.4; publication and exact-PID checks against its official artifact are the remaining delivery checks. Historical independent source validation remains at its recorded commits.

## Release check corrections

The first PR Linux run exposed a test acknowledgement race: file existence could be observed before the PID contents were written. The blocked-AI helper now writes a same-directory temporary file and renames it atomically. The cancellation, twenty-worker deadline and repeated-resource tests pass twenty repetitions under the race detector; exact PID assertions remain strict.

The resident image acceptance fixture is restored before HTTP authentication tests, tracks each replacement boot PID, waits for completed boot and bounded shutdown, and cleans up the current server. Production authentication and worker runtime are unchanged. Independent read-only review approved both test corrections; the pinned-image acceptance proof passed all 133 checks in Cloud Build `81bef821-1dea-4ad8-8ac9-e38d7ba8d30d` against resident image digest `sha256:9db4a5508b29172527918adc6e37635b3a9105a2b40d08350f8ba88525454770`. Fresh PR/dev CI remain delivery gates.

Concurrent dev release preparation reserved v0.54.3 for its resident fixture repair. This worker release advances to v0.54.4 and retains that intervening change.

### macOS release CI cleanup correction

The final PR macOS run printed all async echo output but failed during group shutdown with EPERM. Apple XNU's [killpg1 implementation](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/kern_sig.c) excludes zombies from group iteration and returns EPERM if it finds no eligible member. Teardown racing that iteration explains the observed result. Group cleanup now makes one signal-0 existence check after EPERM and accepts only ESRCH, proving that the same group is gone; any still-present group or probe error retains the original failure. There is no second destructive signal, retry delay, change to the shared deadline, or bypass of Wait/reaping.

The deterministic seven-case regression test first failed, then passed. Ten focused proctree/effects race repetitions passed, including natural exit, descendants, blocked AI, cancellation, shared deadline and repeated resources. Independent read-only review approved the exact change; full tests and lint passed after the correction, as did twenty repetitions of the macOS async echo fixture. A new complete PR/dev CI run gates delivery.


The following CI rerun reproduced EPERM before the child was reaped, showing that an immediate existence check alone was insufficient. The signal phase now retains a raw EPERM as pending, and checked Join observes the same group only after the child waiter and owned I/O have completed. Only a signal-0 ESRCH result clears that error. Wrapped/joined errors, a live group, permission-denied probes and other probe failures remain failures. Cmd.Cancel defers the pending verdict too, avoiding a premature exec cancellation wrapper while the raw stop error remains owned. Termination still happens at most once and uses the original deadline.

Test-only WebSocket fixtures now stop accepting requests, close hijacked sessions and wait for the full handler semaphore before closing the engine or removing the project directory. This prevents a route still compiling from racing TempDir cache deletion; production API shutdown behavior and authentication checks are unchanged.


The final late-Join correction passed ten proctree/effects race repetitions, including real completed-leader/live-descendant failures preserved through public cancellation and natural owner release. An independent read-only reviewer approved the final source and fixture deadline and reran the permission/resolver regressions twice under the race detector (PASS). Lint, Windows/WASM cross-compilation, architecture boundaries, file sizes and changelog gates passed. The fixture await uses its configured maximum duration plus five seconds rather than racing the idle timer. The full suite, exact macOS echo repetitions and WebSocket races are rerun locally; fresh PR/dev CI remain delivery gates.

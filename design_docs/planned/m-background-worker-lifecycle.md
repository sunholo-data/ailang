# M-BACKGROUND-WORKER-LIFECYCLE: Owned Subprocess Shutdown and Cancellation

**Status**: Planned — awaiting design approval; implementation has not begun

**Target**: Next supporting release after approval; version TBD (source baseline v0.54.1)

**Priority**: P1 — blocks the consumer application's worker feasibility gate

**Estimated**: 5–8 engineering days, provisionally 1,800–2,600 lines including tests; sprint planner must calibrate against velocity

**Dependencies**: Existing Process/Stream effects, `internal/proctree`, and native terminal lifecycle integration

**Report**: `inbox_1791630246888_28589e92`, from `stapledons_godot`, 2026-10-10, “v0.52.0: async and managed workers survive host exit”

**Related issue**: [#231: AI-step cancellation](https://github.com/sunholo-data/ailang/issues/231), a separate cancellation boundary

## Problem Statement

A TUI can quit successfully while its background AILANG workers remain running. The reporter reproduced this in pinned v0.52.0 and stopped the approved application sprint at that failed feasibility gate. The application needs runtime ownership and an exposed cancellation operation before it can continue its reply inbox and parallel crew reactions.

The failure also reproduces in the **official, checksummed v0.54.0 Darwin arm64 release**, commit `361caeda13830b0a3faa11919cc2b5d37a14dde1`. Both workers below slept for 12 seconds, with **zero provider calls**. A 500 ms startup window preceded the quit input; testing an immediate quit alone races async worker startup.

| Host path | Host PID | Worker PID | Host exit | Worker observed after exit |
|---|---:|---:|---:|---|
| `asyncExecProcess` + `selectEvents`, handler returns false | 81697 | 81699 | 0 | Alive, PPID 1, state S |
| `spawnProcess` + `closeProcessStdin` | 81710 | 81712 | 0 | Alive, PPID 1, state S |

The release archive SHA-256 was `b67b8931875e8a6dd96bc34a6b912657c25b08905f817d20a15cb484c1d4580e`. Only the observed worker PIDs were terminated after the probe. These observations prove exit cleanup failure; they do **not** demonstrate cancellation during a real AI call. The latest-dev source audit used commit `37001e839`, with `std/VERSION` v0.54.1. This document does not claim an official v0.54.1 runtime reproduction.

### Minimal Reproduction Recipe

Use an exclusive working directory and the same absolute pinned runtime for host and child. The child imports `std/io.println` and `std/clock.sleep`, emits a startup line, sleeps 12,000 ms, then finishes. Both host fixtures and the child were type-checked with the official runtime.

1. Async host: call `asyncExecProcess(runtime, ["run", "--caps", "IO,Clock", "--chdir", root, "worker.ail"], "worker", 5, 1)`, create a stdin-line source, and select both. Return false on a stdin `SourceText` event.
2. Managed host: call `spawnProcess` with the same child arguments, await one input line, then call `closeProcessStdin` and return.
3. Run each host with `--caps IO,Env,Stream,Process --process-allowlist ABSOLUTE_RUNTIME --chdir REPRO_ROOT`; pass the runtime/root as program arguments. Record the exact direct-child PID, allow startup, then supply quit input.
4. Wait for host exit and inspect that recorded child with `ps -o pid,ppid,state -p CHILD_PID`. Record exit status and PPID; terminate only the probe's known child during cleanup.

A future regression test should replace the fixed startup window with an acknowledged child-start barrier and add a provider-free blocking effect, so cancellation during work is proved without a paid AI request.

The root cause crosses several paths:

- Managed and async subprocess handlers construct workers with `context.Background()` instead of the run context.
- Closing managed stdin immediately releases the handle, even when the child remains alive. Adding an exit cleanup call alone would miss that child.
- `CloseAllManaged` exists but has no production callers in `internal/` or `cmd/`. Runner cleanup currently defers filesystem-root closure.
- `StreamContext.CloseAll` closes connections, while process sources occupy a separate source registry.
- Async source `Close` starts cleanup and returns without waiting. Its only `cmd.Wait` is initiated by `Close`, rather than natural completion.
- Neither standard-library surface exports a worker cancellation operation.

The [implemented managed-stdin design](../implemented/v0_9_0/m-async-io-process-stdin.md) already promises cleanup on program exit. This follow-up repairs that unfulfilled lifecycle guarantee and specifies the missing public cancellation boundary. The terminal package can present the operation, but cannot implement runtime subprocess ownership itself.

## Goals and Success Metrics

**Primary goal:** On supported POSIX hosts, every managed or async process worker belongs to an execution owner that terminates and reaps its owned processes before reporting successful shutdown.

1. Both reported exit reproductions leave zero live owned workers or zombies, including after stdin closure.
2. Explicit cancellation stops one worker blocked in a provider-free AI/effect stub, plus its non-detached descendants, without stopping another worker or request.
3. Cleanup uses one **2-second total deadline per owner**, with a 500 ms integration-test scheduling tolerance, rather than a timeout multiplied by the number of workers.
4. Normal return, typed error, budget failure, `exit(n)`, handled panic, host cancellation, and SIGINT/SIGTERM all exercise the same ownership contract.
5. The consumer can rerun its failed feasibility gate using the published supporting runtime before resuming application features.

## Axiom Compliance

Canonical reference: [Design Axioms](../../docs/docs/references/axioms.mdx).

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | 0 | Process timing remains external; receipts expose outcomes rather than asserting deterministic OS timing. |
| A2: Replayability | +1 | Cancellation and shutdown outcomes become inspectable trace events. |
| A3: Effect Legibility | +1 | Public operations declare Process and, for sources, Stream. |
| A4: Explicit Authority | +1 | Cancellation resolves only an owned registry entry, never an arbitrary PID. |
| A5: Bounded Verification | +1 | Fixed cleanup deadlines and provider-free lifecycle fixtures make the guarantee locally testable. |
| A6: Safe Concurrency | +1 | One wait owner and explicit join prevent orphaning and double-wait races. |
| A7: Machines First | +1 | Typed failures identify invalid handles, unsupported platforms, timeout, and infrastructure failure. |
| A8: Minimal Syntax | 0 | Functions and ADTs use existing syntax. |
| A9: Cost Visibility | +1 | Cleanup timing and failure are visible; remote provider billing is outside the local guarantee. |
| A10: Composability | +1 | Budget views share ownership; independent embedded calls receive independent owners. |
| A11: Structured Failure | +1 | Failed termination or reaping cannot silently become success. |
| A12: System Boundary | +1 | Local process-group ownership and its platform limits are explicit. |

**Net score: +10.** Design is eligible for review. No −1 on A1, A3, A4, or A7; this score does not grant implementation approval.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Owner lifetime is a CLI run/batch item, embedded call, engine initialization lifetime, or REPL session | Determines handle lifetime and shutdown behavior | human | design | high |
| New cancellation operations target owned OS workers; in-process AI cancellation remains separate | Establishes the public authority boundary | human | design | high |
| POSIX process groups are the first full descendant guarantee; Windows cancellation reports unsupported until equivalent supervision exists | Prevents false portability promises | human | design | high |
| Independent evaluator forks get fresh process registries; budget views share the same owner | Prevents cross-request cancellation | human | design | high |
| Two-second total shutdown deadline; up to 250 ms cooperative drain for stdin already closed | Bounds shutdown while retaining existing buffered-input behavior | human | design | med |
| Reuse the leaf `internal/proctree` helper; preserve error-returning kill results | Avoids duplicate process-tree mechanisms and dependency violations | agent | compile | med |
| Internal worker state machine, pipe implementation, and test helper organization | Must satisfy single-wait, drain, and bounded-join invariants | agent | compile | med |

### Design Freeze

Before implementation, design approval must explicitly settle these proposals:

- [ ] Approve the owner lifetimes and borrowed-resource rules below, including engine initialization ownership.
- [ ] Approve `cancelProcess` and `cancelProcessSource`, their typed errors, and removal of completed handles without an unbounded receipt cache.
- [ ] Approve POSIX full-tree support and the explicit Windows/WASM unsupported cancellation result.
- [ ] Approve fresh ownership for independent request forks and shared ownership for budget views.
- [ ] Approve the 2-second total deadline and 250 ms cooperative drain policy.

These are review choices, not remaining research prerequisites. Once approved, the implementer may resolve the internal choices in Deferred Decisions without another approval cycle.

## Solution Design

### 1. An Execution Owner, Separate from Capability Configuration

Introduce an execution-owned worker supervisor in the effects layer. It records managed subprocesses and async process sources, prevents admission after shutdown starts, derives worker contexts from the host context, and provides a synchronous, bounded shutdown receipt.

Configuration and authority remain explicit: copy pinned allowlists, subcommand restrictions, confinement, transport policies, and budgets without granting additional capabilities. `WithBudget` is a view of the same execution and must share its owner. `EffContext.Clone`, used by evaluator forks for independent calls, must create fresh runtime registries while preserving policy. Global, non-reused opaque IDs combined with owner-local lookup prevent an integer from another owner's registry accidentally cancelling a same-numbered worker. Exhaustion must fail visibly; IDs are never OS PIDs. Existing constructible handle ADTs are not cryptographically unforgeable capabilities; Process authority and registry membership remain required.

Every successfully started worker is registered before it can escape to AILANG code. Shutdown/admission races either register the child for cleanup or terminate and reap it before rejecting admission. Cleanup happens outside registry locks. A cancellation race shares the same in-flight result; there is exactly one `cmd.Wait` owner.

| Host boundary | Resource ownership | Required integration |
|---|---|---|
| CLI run / batch item | One owner per execution | Create before module evaluation; close on every exit path before final result/trace completion |
| Embedded function call | Fresh request owner | Close after evaluation, before releasing request builtin callbacks |
| Module initialization in an engine | Engine initialization owner | Retain until `Engine.Close`; initialization handles are not transferable into request owners |
| Caller-supplied effect context | Caller retains ownership of the base context | Requests use owned children; expose an explicit host shutdown operation for the base; do not silently close it |
| WebSocket request | Request owns workers; route owns client transport | Stop workers with the request; retain route-selected close code and connection ownership |
| REPL | One owner for the session | Handles survive between lines; close on reset, quit, or host session termination |

Use an interface at the runtime/embed boundary so `internal/runtime` does not acquire a dependency on concrete effects implementation. Concurrent engine close must reject new calls, cancel active owned requests, and join outside the engine mutex; it must not deadlock a completing request or close a caller's base context.

### 2. Worker State and Cooperative Stdin Closure

Suggested states are `running → stdin-closing/stop-requested → reaped → released`. Stdin closure is not process completion. On normal owner return, mark admission closed and perform the cooperative grace before cancelling the owner context; an externally cancelled parent or an explicit cancellation may force-stop immediately:

- `closeProcessStdin` stays idempotent and signals EOF after accepted buffered writes drain. It retains ownership until termination/reaping.
- On owner shutdown, workers with stdin already closing get at most **250 ms shared cooperative grace**, within the total deadline. Other workers can be stopped immediately. Remaining workers are force-stopped and joined.
- Explicit cancellation may discard pending writes and must state this behavior in `std/process` documentation.
- Writer shutdown must avoid sending to a closed channel, close only owned pipes, and join the writer. Recovering from a channel panic is not an acceptable normal protocol.

For async process sources, natural completion must reap the command once and deliver the final partial output chunk. The implementer must choose pipe ownership/order that permits concurrent exit observation without `cmd.Wait` prematurely closing a `StdoutPipe` reader. A blocked event consumer or full output queue must not prevent cancellation and join.

Calling `selectEvents` borrows its sources. Returning false stops selection, while sources remain usable until explicit cancellation or owner shutdown. This preserves reuse of a source across selection calls.

### 3. Public Cancellation API

Proposed additions, **not currently shipped**:

```ailang
-- Export WorkerCancelError from std/process; import it in std/stream.
export type WorkerCancelError =
  | WorkerHandleInvalid(string)
  | WorkerCancelUnsupported(string)
  | WorkerCancelTimedOut(int)
  | WorkerCancelFailed(string) deriving (Eq)

-- std/process
cancelProcess(handle: ProcessHandle) -> Result[(), WorkerCancelError] ! {Process}

-- std/stream; only a source created by asyncExecProcess is eligible.
cancelProcessSource(source: StreamSource) -> Result[(), WorkerCancelError] ! {Stream, Process}
```

The function lines above describe signatures, not standalone declarations. A complete design-only stub containing these signatures and constructors type-checked under the official v0.54.0 runtime. Builtin registration, standard-library wrappers, native/VM behavior, and unsupported-backend diagnostics are implementation work.

`Ok(())` means the owned process has been stopped/reaped and its runtime-owned reader/writer tasks have joined. An already naturally completed but still registered entry can finish successfully. Concurrent cancellation calls join the same operation. Once an entry has been fully released, a later call returns `WorkerHandleInvalid`; repeated calls have no additional termination effect. This avoids retaining an unbounded tombstone ledger in long-lived REPL/engine sessions. Preserve existing idempotent `closeProcessStdin` behavior for released handles.

Wrong owner, fabricated nonexistent ID, and wrong source kind produce typed failure without signalling any process. `cancelProcessSource` does not cancel stdin readers or connection adapters. Existing missing-capability and budget failures retain their established effect failure path; the new operations must perform their declared checks. Mandatory host cleanup is not charged to a depleted user budget.

Cancellation uses the same 2-second total stop-and-join deadline; `WorkerCancelTimedOut` carries that limit in milliseconds. Cancellation kills the **local OS worker**, including one blocked in an AI call. It does not add an abort operation to the parent evaluator's own AI handler, and does not guarantee remote inference or billing stops when a client disappears. [#231's provider-context design](v0_29_0/m-agent-step-cancellation.md) handles that separate boundary.

### 4. Process Trees and Bounded Cleanup

Reuse `internal/proctree` to put each owned worker in its own POSIX process group before `Start`. Explicit cancellation force-stops that group and joins it. This first version does not promise SIGTERM grace or cooperative AI-provider cancellation. `closeProcessStdin` is the cooperative channel.

`proctree.Configure` already installs group cancellation and a 2-second `WaitDelay`, but its current `Kill` discards the group-kill error. Public success must use the checked, error-returning path or a small extension of the same leaf helper. Do not add a parallel tree-kill implementation or expose PID termination to AILANG.

The owner deadline covers all workers together; start termination concurrently and join using the remaining time. Cleanup uses a fresh bounded context so an already-cancelled host context does not skip reaping. Receipt data includes resource identity, phase, elapsed time, termination error, and join outcome. Actual PID absence/group membership checks and single-wait completion are required in tests; closing an event channel alone is insufficient.

A group can outlive its leader. Preserve group ownership through cleanup of non-detached descendants, and release it only after the termination protocol completes. Never retain an old numeric process-group ID for later signalling after ownership is released; this risks signalling a reused ID.

| Platform | Contract |
|---|---|
| macOS / Linux POSIX | Owned worker and non-detached descendants in its process group are stopped; direct child is reaped; owned runtime tasks join within deadline or failure is reported |
| Windows | Existing helper is leader-only. New explicit cancellation returns `WorkerCancelUnsupported` until an equivalent descendant supervisor is implemented; host teardown still attempts bounded direct-child cleanup and reports its limited scope |
| JS/WASM / unsupported native platforms | Expose a typed unsupported result and keep builds valid; never silently emulate a successful process cancellation |

SIGKILL of the host, power loss, and deliberately detached descendants are outside the in-process guarantee. Claims concern workers admitted through these two runtime APIs, not arbitrary subprocesses launched by other tools.

### 5. Exit, Signals, Errors, and Borrowed Resources

Install CLI shutdown handling at the execution-owner boundary, and integrate with existing native terminal signal handling rather than creating competing signal consumers. Restore terminal state first, cancel/join owned workers, then perform the established exit action. SIGINT/SIGTERM preserve 130/143. The coordinator must not wait for the blocked application evaluator or join the terminal signal goroutine from itself. Embedded calls must not install process-global signal handlers.

Keep the primary evaluation failure or explicit `exit(n)` result. Append cleanup failures as structured diagnostics; otherwise-successful execution must return nonzero if owned-worker cleanup fails. Trace shutdown outcomes before final flush. A recovered evaluation panic must run cleanup and preserve its failure; unrecoverable runtime death remains outside the guarantee.

Process-source cleanup must not become indiscriminate transport cleanup. WebSocket routes retain ownership of client connections and their 1011/1001 close-code policy. Connection-backed source adapters remain borrowed and do not disconnect their underlying connection.

Stdin-source `Close` currently signals a stop channel, but `Scanner.Scan` or `ReadString` on an arbitrary borrowed reader may remain blocked. Do not claim a bounded join for all `EventSource` kinds, close `os.Stdin`, or release native stdin ownership while a reader is still active. Signal non-process sources as appropriate; record pending borrowed readers separately and preserve unread bytes/leases. Their interruptibility is outside this worker cancellation API and must not delay owned-process termination.

## Conflict Surface

| Existing contract / mechanism | Conflict risk | Preservation requirement |
|---|---|---|
| Process authorizer, pinned path, subcommand policy, confined exec-only mode | Cancellation/respawn could widen authority | Preserve resolution and confined spawn refusal; no shell or PID escape hatch |
| `WithBudget` shares a running execution | Closing a borrowed view kills unrelated work in the same run | Same owner; only the host run closes it |
| Evaluator `Fork` uses `EffContext.Clone` | Shared registry lets one request stop another | Fresh owner/registries with identical policy; wrong-owner ID is rejected |
| Buffered managed stdin | Removing tracking leaks child; abrupt cleanup drops routine closing writes | Retain through Wait, bounded cooperative drain, race-safe writer shutdown |
| Async stdout/chunking and `selectEvents` reuse | Double Wait, lost final chunk, implicit kill on handler false | One Wait, drain on normal EOF, selector borrows sources |
| Native terminal signals | Exit hook bypasses defers; cleanup watcher can join itself | Restore TTY first, bounded worker join, then exit 130/143 |
| WebSocket route close codes and connection adapters | Generic owner closure sends success before an error close | Route owns transport; preserve error/shutdown codes and borrowed adapters |
| Engine/base contexts and REPL lifetime | Per-call cleanup invalidates persistent resources or closes caller-owned context | Explicit ownership table; fresh request children, session-owned REPL |
| VM, builtin metadata, JS/WASM and generated backends | Wrapper compiles but backend panics or silently succeeds | Register/type-check declared effects; prove native/VM parity or explicit unsupported behavior for each backend |

### Programs That Must Still Work

Fixture bodies were read during this design audit:

- [process_stdin_write.ail](../../examples/runnable/process_stdin_write.ail): buffered writes followed by stdin closure; accepted queued writes drain for a cooperative child.
- [stream_process_source.ail](../../examples/runnable/stream_process_source.ail): chunked process output and stdin selection; preserve final partial chunks and source priority.
- [stream_multi_source.ail](../../examples/runnable/stream_multi_source.ail): handler false followed by normal host work; stopping selection does not close borrowed stdin.
- [terminal_keys.ail](../../examples/runnable/terminal_keys.ail): q/Escape/EOF/Interrupted paths retain terminal restoration; add owned-worker PTY coverage around these paths.
- [process_demo.ail](../../examples/runnable/process_demo.ail): synchronous `exec` retains nonzero-exit-as-`Ok` completion semantics and typed spawn failures.

## Validation and Acceptance Criteria

Use deterministic helper children and mock/blocking AI effects, **without live provider calls**. Test helpers must emit a startup acknowledgement and expose their exact PID/group identity before cancellation; avoid startup races and broad process-name killing.

- [ ] Reproduce both reported host-exit cases before the fix; both pass after it. In particular, closing stdin must not hide a still-running worker from teardown.
- [ ] Cancel managed and async workers during a provider-free blocked AI/effect stub. A second worker and an unrelated bystander remain alive until their own cleanup.
- [ ] Stop a worker that created a non-detached grandchild, including a leader-exits-first case; prove process-group death and direct-child reaping.
- [ ] Cover natural EOF/final partial chunk, stdin EOF/drained queued writes, output backpressure, blocked pipe reads/writes, and repeated/in-flight cancellation without double Wait or channel panic.
- [ ] Reject fabricated nonexistent, stale, wrong-owner, and wrong-kind handles; verify fresh request registry isolation and shared ownership across budget views.
- [ ] Cover normal return, error, budget exhaustion, `exit(7)`, recovered panic, parent cancellation, and SIGINT/SIGTERM. Preserve primary exit codes and terminal state.
- [ ] Inject kill/wait failure and deadline exhaustion: expose structured failure, do not report successful cleanup, and fail otherwise-successful host execution.
- [ ] Admit workers concurrently with shutdown; every started child is either owned and joined or rejected and cleaned up.
- [ ] With 20 simultaneously blocked workers, cleanup uses one deadline rather than 20 serial waits; verify runtime-owned goroutines and descriptors do not accumulate over repeated runs.
- [ ] Embedded call, engine initialization/close, REPL reset/quit, and WebSocket error/shutdown lifetimes pass. Closing one request does not close another or a caller-owned base/transport.
- [ ] Run focused effects/runner/runtime/embed/apiserver/REPL tests, race detector for lifecycle races, relevant PTY tests, `make check-boundaries`, formatting/lint, and applicable broader repository checks.
- [ ] Native/VM API behavior matches; Windows and JS/WASM compile and deliver their documented unsupported cancellation result. Full-tree acceptance is explicitly POSIX until Windows supervision is added.
- [ ] Re-run the consumer cleanup gate against an official published supporting runtime, with pinned version/commit and exact PID evidence. Only then hand back the reply-inbox/parallel-reaction application work.

Existing tests are partial evidence: `TestManagedProcess_KillOnClose` and `TestProcessContext_CloseAllManaged` directly invoke internal cleanup and wait for managed completion; they do not exercise host exit. `TestProcessSource_CloseKillsProcess`, `ContextCancellation`, and `EOFClosesCleanly` assert event-channel closure, which does not establish process reaping. Preserve these tests and add integration evidence at the missing boundary.

## Implementation Outline

This is a design decomposition, not an approved sprint:

1. **Ownership and platform skeleton:** finalize review choices; establish host owner interfaces and policy-preserving child contexts; specify pipe/Wait ordering and admission races.
2. **Managed workers:** process groups, retained tracking after stdin closure, checked cancellation, writer joins, typed public API and focused race tests.
3. **Async process sources:** single natural-exit Wait, final output preservation, source cancellation and bounded join.
4. **Host integration:** runner/batch, request forks, engine/REPL lifetimes, terminal signal sequencing and WebSocket ownership.
5. **Evidence and delivery:** end-to-end PID/group tests, independent sprint evaluation, documentation, supporting runtime release, then consumer feasibility retest. Package wrappers/publication are a separate follow-on if the consumer needs them.

## Verification Log

Checks were performed on 2026-10-10. Source paths below refer to latest-dev commit `37001e839`; runtime probes use the official v0.54.0 binary. Temporary probe files are not repository dependencies; the durable results are recorded here.

| ID | Claim | Actual check and result |
|---|---|---|
| V1 | Both workers survive host exit | Official checksummed v0.54.0, 500 ms startup window, sleep-only worker; PID/PPID/exit evidence in Problem Statement; zero provider calls |
| V2 | Cancellation exports are absent | Complete `std/process.ail` / `std/stream.ail` export read; live `ailang check` imports of `cancelProcess` and `cancelProcessSource` each exit 1 with IMP010 |
| V3 | Proposed AILANG types/signatures are expressible | Fresh `ailang prompt`, complete `proposed_api.ail` stub with imported handles, Result, new ADT, and both effect annotations; official `ailang check` succeeds |
| V4 | New constructor names are unallocated | `rg` for the four proposed Worker-prefixed constructors across `std/ internal/ cmd/` returns no matches; no numeric diagnostic code allocated |
| V5 | Managed stdin closure drops live tracking | Read `ProcessCloseStdin` in `internal/effects/process_spawn.go`: `CloseStdin` immediately followed by `ReleaseManagedProcess` |
| V6 | Managed cleanup lacks production callers | `rg CloseAllManaged internal cmd`: declarations, JS stub and tests only; read `runner/run.go` and `runner/batch.go` teardown |
| V7 | Spawn handlers ignore the host context | Read `process_spawn.go` and `stream_async_process.go`: each passes `context.Background()` to worker creation |
| V8 | Stream shutdown misses source registry | Read `stream_context.go` connection-only closure plus `stream_source.go` source registry and borrowed `connSource.Close` |
| V9 | Async Wait is not on natural EOF; Close is asynchronous | Read complete `stream_process.go`: sole Wait inside Close's goroutine; reader EOF closes event delivery only |
| V10 | Existing child helpers are leader-only in these worker paths | Read `process_managed.go` and `stream_process.go`: `exec.CommandContext` default cancellation and direct-child Signal; no `proctree.Configure` in either |
| V11 | Shared process-tree helper already exists, with Windows limits | Read `internal/proctree/proctree.go` and platform files: SetGroup, Configure, checked KillGroup; Windows leader-only, without descendant supervision |
| V12 | Selection exit does not cancel sources | Read `stream_async_ops.go` and `stream_mux.go`: handler false returns; no source-owner cleanup in that path |
| V13 | Borrowed stdin may remain blocked | Read `stream_stdin.go`: Scanner/ReadString read paths; Close signals done but does not interrupt arbitrary borrowed reader |
| V14 | Budget views and independent forks need different ownership | Read `effects/context.go` WithBudget/Clone and `eval/eval_evaluator.go` Fork; budget shares contexts, existing clone copies Process/Stream pointers |
| V15 | Embedded cleanup is not already supplied by Engine.Close | Read `embed/embed.go` Close: marks closed under mutex; read `runtime/entrypoint.go` fork/call lifecycle and missing worker teardown |
| V16 | Terminal signal exit may bypass ordinary cleanup | Read `terminal.go` terminateSignal: restores terminal then invokes host exit hook; runner supplies that hook |
| V17 | WebSocket and REPL are distinct host owners | Read `apiserver/routes_ws.go` request Stream.Child and route close codes; `repl/repl.go` session effect context and session teardown |
| V18 | Existing tests prove only narrower cleanup behavior | Read bodies of the five named managed/source tests; explicit managed done checks versus source event-channel assertions |
| V19 | Cited regression fixtures exist and behavior is verified | Read bodies of all five Programs That Must Still Work fixtures; references match current paths |
| V20 | New design is distinct from existing proposals | Neural search and direct reads of managed-stdin, AI-step/provider-context, process guardrails and CSP docs; distinctions below |

## Related Documents and Coverage Decision

The pre-creation neural search for “background worker process lifecycle cancellation” ranked planned AI-step cancellation at 0.44 and implemented process guardrails at 0.37; other relevant top results were lower. Neither the planned 0.75 nor implemented 0.65 duplicate threshold was met. Embedding scans were partial (71/200 planned and 102/200 implemented), so these scores are discovery evidence rather than proof of exhaustive absence. Targeted repository search and direct reading established the distinctions:

- [Managed process stdin, implemented v0.9.0](../implemented/v0_9_0/m-async-io-process-stdin.md): promises exit cleanup, but its wiring and live-handle retention are incomplete. This document repairs the guarantee and adds explicit cancellation.
- [AI-step cancellation](v0_29_0/m-agent-step-cancellation.md) and [provider-context triage](ailang-core-triage/ai-cancellable-provider-context.md): cancel an AI call within its parent evaluator. This document owns and stops OS subprocess workers; it does not change AI provider interfaces.
- [Process guardrails proposal](../implemented/v0_5_6/m-eval-process-guardrails.md): focuses on the evaluation harness; its historical status metadata is inconsistent and does not prove this runtime behavior shipped. Reuse current `internal/proctree` implementation instead.
- [CSP/session-types proposal](v1_1_0/m-csp-session-types.md): a broader scheduler/channel design. Worker ownership here does not depend on new language syntax or that scheduler.
- [Native terminal input, implemented v0.54.0](../implemented/v0_54_0/m-terminal-ui-native-input.md): delivers terminal primitives and records the worker feasibility failure; this design closes that separate runtime gap.
- [Program north star](../PROGRAM.md): this is runtime work in the AILANG fix lane. Reply-inbox rendering, crew reaction policy, and package presentation remain consumer-level work after the gate passes.

## Deferred Decisions

- Implementer may choose worker/supervisor type names and file splits while keeping core dependency boundaries.
- Implementer may choose pipe plumbing and join-channel representation, provided normal output drains and Wait has exactly one owner.
- Implementer may choose structured cleanup diagnostic formatting consistent with existing host APIs; timeout and infrastructure failure must remain distinguishable.
- Implementer may choose portable test helpers and timing tolerances within the stated deadline; they must prove exact process identity and avoid live AI cost.

## Non-Goals

This work does not add a general concurrency scheduler, CSP syntax, detached-worker API, arbitrary PID termination, parent-evaluator AI abort, remote billing guarantees, or Windows Job Object support. It does not resume the consumer application sprint, change cloud credit controls, or act on unrelated coordinator approvals. Implementing interruptible arbitrary borrowed stdin readers is separate work.

## Risks and Approval Boundary

The largest compatibility risks are buffered stdin shutdown, source output/Wait ordering, persistent embed/REPL lifetimes, and terminal signal reentrancy. They are explicit acceptance gates rather than incidental cleanup details. Platform support must remain truthful, and a deadline failure must not be hidden by a normal host exit.

The authorized deliverable is this design document. Runtime changes require design approval, a sprint plan, an explicit execute instruction, and independent sprint evaluation. No runtime implementation, message acknowledgment, GitHub issue creation, or release is included in this design task.

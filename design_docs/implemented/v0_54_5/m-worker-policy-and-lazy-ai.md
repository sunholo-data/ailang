# M-WORKER-FOLLOWUP: Policy Worker Lifetime and Keyless CLI Startup

**Status**: Released v0.54.5; independent implementation evaluation PASS 87/100 and all release gates passed.
**Target**: v0.54.5 (available patch rechecked before release preparation).
**Priority**: P1 — blocks Stapledon's reply-inbox cleanup gate and single-command offline/live CLI.
**Estimated**: 5 engineering days, approximately 2,100 changed lines including tests and documentation; 25% contingency included in time.
**Dependencies**: Released v0.54.4 worker ownership/cancellation and existing policy supervisor. No package publication required.
**Created**: 2026-10-11
**Reports**: `inbox_1791741386377_b0b76b40` and `inbox_1791740982044_18b3491c` from `stapledons_godot` in canonical prod messaging.

## Problem and evidence

The official v0.54.4 Darwin arm64 runtime, commit `bc078dc1b41308542bbccd3417bf040283e817df`, passes ordinary worker cancellation. The consumer's saved policy-worker retest still fails: `cancelProcess` prints `Ok(())`, the host exits in 2–5 ms, and a `--policy-worker` remains under PPID 1. Explicit cancellation of two async sources prints two `Ok(())` results and leaves two policy workers after a 48 ms quit. The consumer used zero providers and recorded fixture cleanup separately. The sprint independently reproduced the leak with the official binary and an actual worker-entry marker, then banked passing managed and two-async integration probes in [validation](validation/README.md).

The runtime creates the boundary itself. `supervisePolicyRun` starts its worker with `proctree.Configure`, so the worker is in a different group from the supervisor. The outer owner kills only its group. The policy timeout uses a background context; the fd-3 admission channel closes after one decision and cannot communicate continuing supervisor lifetime. The released lifecycle design explicitly scoped descendant termination to the owned group. This follow-up extends ownership through AILANG-created policy supervision rather than treating that worker as an arbitrary detached process.

Separately, `setupAIHandler` resolves credentials and creates the provider before entrypoint execution. A generated `[bin]` with `run_flags = ["--ai", "z-ai/glm-5.3-flash"]` therefore cannot run its own offline or help branch without a key. A new isolated probe type-checks on the official runtime; both branches exit 1 with `OPENROUTER_API_KEY environment variable required` and no program output. A strict-VM offline branch with a stub exits 0; actual `std/ai.call` remains evaluator-only in strict mode. This release must not expand the strict VM's AI surface.

## Goals

1. Cancellation and normal host teardown stop policy supervisors and their owned workers within the existing shared two-second deadline, or report a structured failure.
2. Successful public cancellation includes acknowledged cleanup of the supervised worker, not merely disappearance of its supervisor.
3. Supervisor death cannot silently sever policy-worker ownership. Preserve filesystem confinement, admission provenance, output caps, timeout limits and unrelated-worker isolation.
4. An AI-configured `run` or generated bin executes non-AI branches without credential reads, provider requests, ADC discovery or Ollama availability probes.
5. The first AI effect initializes the selected client once, with existing authority/budget checks and visible authentication failures. Retain each AI operation's error shape.
6. Publish and verify the supporting runtime, then rerun the consumer's pinned-runtime gate before claiming its M1 blocker resolved.

## High-impact decisions

| Decision | Proposal | Owner / freeze point |
|---|---|---|
| Cross-group ownership | Private inherited lifecycle pipes between an owned same-runtime `run` child and its owner; separate supervisor-to-worker lifeline | Human approval before execution |
| Success criterion | Acknowledged inner cleanup plus direct-child Wait and owned I/O joins; missing acknowledgement is failure | Human approval before execution |
| Cancellation deadline | Keep one two-second owner deadline; no per-worker or per-level restart | Human approval before execution |
| AI startup semantics | `run` uses lazy client creation; structural configuration is resolved/frozen before entrypoint | Human approval before execution |
| Credential timing | Resolve key-file credentials and readiness on first AI effect for `run`; existing eager setup remains available to serve-api and explicit setup callers | Human approval before execution |
| Scope | Same-runtime supervised children on macOS/Linux; existing Windows/WASM typed unsupported cancellation remains | Human approval before execution |

### Design freeze

- [x] Approve the lifecycle protocol, acknowledgement requirement and force-stop failure semantics.
- [x] Approve first-effect credential/readiness timing for `run`, including key files.
- [x] Approve scope, acceptance gates and follow-up release delivery.

## A. Policy-worker lifetime

### Private owner channel

Add a small stdlib-only lifecycle transport beside the existing leaf `internal/proctree` helper, reused by managed processes and async process sources. Prepare it before `Start` only for the executing AILANG runtime itself (`os.SameFile` after executable resolution, including symlink aliases), invoked as `run`. This avoids assigning an AILANG protocol to arbitrary commands or trusting executable names. Both parent and child must use the upgraded runtime for this guarantee; older binaries/copies and deliberately detached external subprocesses retain the documented group boundary.

Use inherited descriptors for stop requests and cleanup receipts, separate from stdin/stdout/stderr and policy admission fd 3. Protocol v1 has bounded frames (maximum 1 KiB) and a fixed state machine: startup registration, stop request, terminal cleanup receipt. It carries remaining cleanup duration and status; it never accepts an OS PID/PGID to signal. All descriptor endpoints have one owner, are close-on-exec by default, and are explicitly passed only to the intended runtime child. Close every endpoint on preparation, startup and registration failure.

The CLI registers its run mode before compiling or spawning a policy worker. An early stop request prevents worker admission. The same transport can stop an ordinary same-runtime run through its existing host cleanup path; policy mode routes it to the supervisor. Capability/allowlist checks still precede process admission. Control endpoints are internal transport, not new AILANG capabilities or public PID functions.

### Supervisor cancellation and lifetime

Keep the policy worker's isolated group: it is needed for policy timeout/output enforcement without killing the supervisor that must report the result. Add a distinct supervisor-owned lifeline whose read endpoint reaches the worker before admission, while only the supervisor retains the write endpoint. fd 3 continues to carry exactly one admission decision and close as today. Strip the outer owner-channel metadata when building worker arguments; explicitly allocate and pass the worker's separate lifeline so fd collisions and authority leakage cannot occur.

A stop request or outer-owner channel closure enters the supervisor's cleanup state. Its worker also uses the shared private stop/receipt transport, so normal outer cancellation asks the worker's runtime owner to stop and join any further owned groups before the supervisor reaps it. A remaining-time force stop uses checked proctree operations; a missing required inner receipt propagates failure. The supervisor waits for its sole child waiter and finishes bounded output draining before emitting its own terminal cleanup receipt. Natural completion follows the same joined lifecycle. Serialize stop, timeout, output-cap, normal completion and signal paths through one termination owner; preserve existing denial/limit codes and stderr provenance. Policy timeout and output-cap enforcement must still stop execution immediately at the existing boundary, rather than grant additional execution time for a cooperative cleanup request. Forced paths may report incomplete cleanup; they cannot synthesize a successful nested receipt.

The worker monitors its supervisor lifeline from bootstrap, before admission. Lifeline EOF means the supervisor vanished. When a run owner exists, route this through that owner's existing bounded cleanup; otherwise stop the worker's own validated group. Use a final watchdog for stalled cleanup. Nested AILANG-owned runs participate through their own private channels. This is a fallback for supervisor death, not proof that cleanup succeeded: the outer owner must still reject a missing receipt. Complete-host SIGKILL and power loss remain outside the acknowledged in-process guarantee.

### Owner stop and join

`RequestStop` remains signal-only: enqueue a private stop request and start the bounded stop controller. `Join(ctx)` awaits the receipt, direct-child Wait and owned reader/writer completion using the owner's remaining deadline. It must not publish successful completion or release registration earlier. Every branch uses the same cleanup state, and concurrent cancellation joins one operation.

For registered same-runtime children, try protocol shutdown before force-killing their outer group. At deadline or transport failure, perform at most one checked force stop and return `WorkerCancelTimedOut` or `WorkerCancelFailed` as appropriate; disappearance of the outer group without the required receipt cannot become `Ok(())`. External ordinary commands retain existing checked group cancellation. Preserve the macOS pending-EPERM/checked-Join behavior and exact ownership release rules.

Closed private channels, malformed/oversized frames and unsupported protocol versions fail visibly. Program stdout cannot forge a receipt. No process-table scan, command-name kill, persisted PID, or arbitrary child-supplied group identifier grants termination authority.

## B. First-effect AI client initialization

Split CLI run setup into a preparation stage and initialization stage. Preparation validates flag combinations, harvests provider declarations and freezes the selected model/provider configuration, routing policy, attribution, auth source path, NoADC policy and handler options. It does not read a key file or create/probe a real provider. Keep structural/configuration errors visible at startup; this request concerns unused credentials, not accepting malformed manifests.

Install a lazy `AIContext` for configured real AI in `run`. Its initializer constructs a complete resolved AI context from the frozen preparation data. Cache either that context or its initialization error once for the execution. Existing eager constructors/setup remain available; explicit stubs and fixture-file validation retain their deterministic eager behavior. Serve-api is not silently switched to lazy service startup.

Implement initialization at the AIContext boundary rather than a generic AIHandler wrapper, so optional reference-image, speech and routing capabilities remain available after resolution. Audit every entry: Call, CallJson, Step, StepWithCache, StepWithStream, image/file/base64/reference variants, and CallSpeech. Resolve per-call model/options after initialization. Metadata-only queries do not initialize a client. Never lose the model resolver, routing metadata, streaming callback, trace attribution or provider-specific options when publishing the resolved context.

The existing effect/capability/budget gate remains before initialization. Missing credentials and readiness errors follow the operation's existing failure channel: plain-call runtime errors versus Result/AIError operations. Retain `--ai-no-adc`, key-file precedence, secret redaction and unsupported-provider failures. A lazy error is not a stub, retry, alternate provider, or successful empty reply. Initialization is once-only and publication is synchronized; this does not make the existing single-threaded AIHandler contract concurrent.

The generated bin continues passing its configured run_flags and application arguments unchanged. The app chooses offline/help/live in AILANG; add no shell argument policy or new manifest syntax. Help/offline must also succeed when an operator-configured key file is unavailable or a local AI daemon is stopped, provided no AI effect is evaluated.

## Conflict surface and compatibility

| Surface | Existing behavior to preserve | Deliberate change |
|---|---|---|
| Managed/async ownership | Fresh registries, budget views, close-stdin retention, one Wait, final chunks, backpressure, concurrent cancel and pending EPERM | Same-runtime joins also wait for private cleanup receipts |
| Policy supervision | Separate worker group; fd-3 single admission decision; output cap; reserved limit exit codes; confined FS and filtered env | Lifetime spans policy-group boundary and outer cancellation reaches it |
| Host termination | Terminal restoration and bounded cleanup on normal/error/exit/signals/batch | Registered supervised descendants participate in the existing owner deadline |
| AI effects | AI capability/budgets; Result errors; model restrictions; image/speech/routing/streaming behavior; trace attribution | Real-provider auth/readiness failure in run occurs at first AI effect |
| Generated bins | Correct argument quoting, entry/caps/package root, runtime pin and run_flags | Non-AI branches can run without an AI credential |
| Backend | Existing evaluator/native-VM cancellation; eager embed handlers; Windows/WASM unsupported results | Strict VM still refuses actual evaluator-only AI calls; offline branches remain valid |

Regression fixtures read/checked during design: `examples/runnable/process_cancel.ail`, `examples/runnable/stream_process_cancel.ail`, `cmd/ailang/run_policy_supervise_test.go` (complete output and bounded drain), `cmd/ailang/run_policy_hardening_test.go` (timeout and descendant death), `cmd/ailang/ai_setup_test.go` (credential precedence, NoADC, fixture errors), and `internal/pkg/bin_test.go` (shim argument construction). New tests must supplement these rather than weaken their assertions or thresholds. No parser/type syntax change is proposed.

## Acceptance criteria

- [x] AC1: Reproduce the reported policy leak before the fix with the official v0.54.4 binary, recording startup acknowledgement and exact positive supervisor/worker PIDs and PGIDs; cleanup is restricted to those owned fixtures.
- [x] AC2: Explicit managed cancellation and cancellation of two async sources remove all acknowledged supervisor/worker PIDs, including zombies, before success; each quit/EOF case remains within the shared two-second deadline plus a 500 ms observation allowance.
- [x] AC3: Normal return, error, exit, SIGINT/SIGTERM and owner/context shutdown include policy descendants; startup-cancel and nested-supervision races cannot leak workers or admit after shutdown.
- [x] AC4: Forced supervisor death triggers the worker lifeline fallback. Missing/failed/forged/oversized receipts cannot report public success; unrelated workers remain alive. Stalled cleanup returns a typed failure within deadline.
- [x] AC5: Policy denial, timeout/output limits, stdout/stderr spoof protection, restricted FS traversal/symlink tests and credential/environment restrictions pass unchanged in meaning. Sole Wait and bounded output preservation remain valid.
- [x] AC6: A generated local [bin] and direct run configured with a real model execute --offline and application --help without keys, unreadable configured key files, ADC/network probes or model calls. Cover evaluator and strict offline execution.
- [x] AC7: First actual AI use initializes once; missing keys fail visibly, Result APIs return their established typed error, and a mock configured provider succeeds with exactly the expected requests. No live inference required.
- [x] AC8: Initialization covers text/JSON, step/cache/streaming, image/reference/base64 and speech entry points without erasing optional interfaces, model restrictions, routing metadata or trace attribution. Denied/exhausted AI cannot initialize a provider.
- [x] AC9: Config/provider/auth choices remain frozen, setup errors contain no secret, stubs stay deterministic, eager setup callers preserve their contract, and repeated runs/requests do not share new mutable initialization state accidentally.
- [x] AC10: Focused races, full Go tests, lint, file-size checks, architecture boundaries, relevant examples and all platform CI builds pass; independent sprint evaluation passes the criteria.
- [x] AC11: Merge reviewed changes to dev, publish the next available patch version, verify official artifact checksums/provenance and the above probes, refresh muRAG, pass cloud TEST gates, promote verified images to PROD and confirm the public MCP version. Rerun the consumer pinned-release policy cleanup gate before describing that gate as passed.

## Implementation outline and estimates

| Milestone | Scope / principal files | Implementation + tests/docs | Dependency |
|---|---|---:|---|
| M1 | Leaf lifecycle transport, early CLI registration, policy worker lifeline; new files beside proctree and run_policy_supervise.go | 300 + 350 = 650 | Design approval |
| M2 | Managed/async stop+join and natural completion integration; process_managed.go, stream_process.go, worker-owner regressions | 220 + 350 = 570 | M1 |
| M3 | Prepared AI setup, lazy AIContext and all entry points; ai_setup.go, ai_handlers.go, main_run_exec.go, effects/ai*.go | 220 + 280 = 500 | Design approval |
| M4 | Real-bin/policy integration demos, docs, independent evaluation and release verification | 60 + 320 = 380 | M2, M3 |

Total 2,100 lines. Plan sequential execution on Days 1–2 (M1), Day 3 (M2), Day 4 (M3), Day 5 (M4), with gate/CI time extending calendar delivery when needed. The seven-day velocity helper finds recent worker/policy commits but no usable measured LOC/day; 420 LOC/day is a planning allowance, not an observed throughput claim. Final sprint planning follows design approval.

Proposed new showcases: `examples/runnable/process_policy_cancel.ail` (IO, Env, Process; contracts skip: OS lifecycle proved by PID tests; inline tests include a pure success classifier) and `examples/runnable/ai_offline_cli.ail` (IO, Env, AI; pure mode selection contracts and inline tests, effectful entrypoint skip: provider behavior checked with mocks). Include the companion worker and policy fixture with explicit Clock/IO and sandbox requirements. Load the active teaching prompt before writing these files; design preparation loaded v0.16.7.

Registry reuse: searched `process` and `AI`, inspected `sunholo/external_backend@0.2.0` with pkg info/docs. It wraps one-shot std/process.exec and explicitly excludes long-lived processes and AI calls. All milestones choose `none`: these fixes require pre-admission Go host ownership and provider initialization, which an imported AILANG package cannot enforce. Reuse existing proctree, owners, policy and AI factory; add no package dependency.

## Verification log

| Claim | Evidence observed on 2026-10-11 |
|---|---|
| Released baseline | gh release list: v0.54.4 Latest; source std/VERSION v0.54.4; official binary full commit bc078dc... |
| Policy leak | Inspected consumer explicit-cancel/policy-cancel.json and lifecycle.json: separate positive PGIDs, PPID 1 survivors, Ok results; independently reproduced before implementation; restricted worker entrypoint marker and recorded positive PIDs/PGIDs, guarded fixture cleanup, zero providers (validation/v0.54.4-policy-marker-before.json) |
| Separate inner group | Read supervisePolicyRun and proctree.setProcessGroup: Configure before Start, Setpgid true |
| Admission pipe supplies no lifetime | Read reportDecision: one JSON message and control.Close; supervisor scanner reads one line; no signal/parent-lifetime watcher in those functions |
| Current owner stop has no policy receipt | Read managedProcess.RequestStop/Join and processSource.finishGroup/Join: group kill, direct Wait and owned I/O only; no inherited lifecycle transport in their spawn paths |
| AI startup mechanism | Read setupAIHandler, resolveAuth, factory.New and readyForCalls: key read, factory/client setup and readiness happen before run entrypoint |
| Optional AI surface | Read AIContext plus ai_image.go and ai_speech.go: direct handler assertions require preserving resolved concrete capabilities |
| Bin behavior shares run setup | Read internal/pkg/bin.go: shim runs ailang run with RunFlags before the file and program arguments after -- |
| Reproducer is valid AILANG | Official runtime check offline_probe.ail: exit 0, No errors found; initial invalid check --stdlib-path command was corrected using AILANG_STDLIB_PATH |
| Keyless offline/help fails now | Official runtime run --caps IO,Env,AI --ai z-ai/glm-5.3-flash: both branches exit 1, missing OPENROUTER_API_KEY, no program output; no provider calls |
| Existing strict boundary | Strict stub offline branch exit 0, expected output; actual std/ai.call strict stub exit 1, evaluator-only __ai_call/no interop bridge; evaluator stub call exit 0 |
| Related designs | Scaffold neural matches max 0.44 planned/0.42 implemented; focused policy triage 0.49 and provider-config 0.48 read. No duplicate threshold reached; unrelated high SimHash matches are discovery noise, not coverage |
| Registry capability | process/AI searches and external_backend info/docs inspected; host lifecycle/client timing not supplied by package wrapper |

Local probe transcripts: `/private/tmp/worker-followup-offline-probe/results.json`; design-search/scaffold log: `/private/tmp/worker-followup-design-scaffold.log`; velocity log: `/private/tmp/worker-followup-velocity.log`. Temporary evidence is not a release acceptance record; M4 must bank the final verified evidence in the implemented design validation artifact.

## Related documents and distinction

- [Policy cancellation triage](../../planned/ailang-core-triage/run-policy-worker-survives-cancellation.md): input report, not an implementation design.
- [v0.54.4 worker lifecycle](../../implemented/v0_54_4/m-background-worker-lifecycle.md): existing group/owner API; this follow-up covers the runtime's separately grouped policy child.
- [Policy hardening](../../implemented/v0_41_0/m-executor-policy-hardening.md): confinement, timeout, output and filtered-env invariants retained.
- [Policy result integrity](../../implemented/v0_53_0/m-run-policy-result-unforgeable.md): admission/result provenance retained; lifecycle channels remain separate.
- [Provider config](../../implemented/v0_15_0/m-ai-provider-config.md): provider declaration/dispatch reused, not reimplemented.
- [Policy cache sprint](../../planned/v0_52_6/run-policy-worker-cache-dir-sprint-plan.md): pending neighboring edits in the supervisor; integrate if landed before implementation, do not fold its separate scope into this release.

## Axiom compliance

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | 0 | Existing effect results stay explicit; once-only initialization and serialized cleanup state |
| A2 Replayability | 0 | Preserve effect trace/routing metadata; OS lifecycle still requires integration evidence |
| A3 Effect legibility | +1 | AI initialization follows the authorized AI effect; cancellation uses existing Process/Stream authority |
| A4 Explicit authority | +1 | Private owner channel, pinned policy/provider and confined roots; no arbitrary PID signalling |
| A5 Bounded verification | +1 | Shared deadline, bounded protocol and provider-free failure fixtures |
| A6 Safe concurrency | +1 | One termination/initialization owner and no early success |
| A7 Machines first | +1 | Existing typed failures and explicit protocol version/status |
| A8 Minimal syntax | 0 | Existing language, manifest and public effect APIs |
| A9 Cost visibility | +1 | Offline/help makes zero AI requests; first-call errors cannot become stub success |
| A10 Composability | +1 | Owned policy runners compose with managed/async parents and generated bins |
| A11 Structured failure | +1 | Missing receipt/auth reports failure, not false success |
| A12 System boundary | +1 | Cross-group lifetime and credential initialization boundaries documented |

Net +9. A1/A3/A4/A7 have no hard violation. This is an author assessment, not user approval or independent evaluation.

## Risks, deferred decisions and approval boundary

Highest risk is descriptor lifetime: a duplicate writer can suppress EOF, or an incorrectly forwarded fd can collide with admission. Tests must prove closure, startup rollback, nested cancellation and no leaked descriptors across repeated runs. A child on an older/different binary cannot provide the new receipt; make the same-runtime boundary explicit. Forced supervisor death cannot be called successful cleanup without acknowledgement even if its worker subsequently self-terminates.

AI initialization timing intentionally changes CLI run: key-file/readiness errors may happen after earlier IO. Freeze provider configuration before program execution; do not reread writable manifests on first effect. Preserve eager service startup and explicit setup tests. A missing key remains an error when AI is used.

Agent latitude after approval: split files to respect size limits, choose fixed frame encoding and internal names, and refine focused test synchronization. Public semantics, deadline changes, weaker authority, arbitrary-PID termination or expanding scope require design revision.

Non-goals: consumer reply-inbox/parallel-crew implementation, provider-side inference/billing abort, stdin EOF events, raw-terminal CRLF package fixes, shim regeneration comments, cache isolation, native strict-VM AI dispatch, Windows process-tree support, package publication.

Repository AGENTS.md requires design approval, sprint planning and an execute-sprint instruction before implementation. The user's request authorizes fixing and releasing both reported issues; the user approved these mechanism choices and sprint execution on 2026-10-11. No implementation, merge, tag, message acknowledgement or publication was performed during design preparation.

## Approval record

The attended user approved this design and sprint execution on 2026-10-11. The existing instruction to fix both reports and publish a follow-up release remains in force.

## Implementation and independent verification

The implementation passes the full local release and independent evaluator gates. M1/M2 use private same-executable lifetime channels and a supervisor lifeline; M3 freezes selected and named-provider choices per execution while deferring client creation. Independent review findings for early exits, malformed stop requests, standalone cleanup failures and final-exit lifetime races are resolved. The startup PID fixture publishes atomically; 20 supervisor-loss repetitions pass without weakening assertions.

The historical [round-1 report](validation/independent-evaluation-round-1.md) scores the implementation 87/100. Final source changes passed independent focused review, full local gates and all final PR/dev platform workflows. [Release delivery](validation/release/delivery.json) records the published v0.54.5 source 296231d, checksum/SLSA proof, official-binary consumer and 19 AI/38 lifecycle probes, brain refresh, TEST and PROD gates, public MCP 0.54.5 and canonical release notice. Consumer async EOF finishes naturally after its worker duration; this sprint does not add an EOF event notification.

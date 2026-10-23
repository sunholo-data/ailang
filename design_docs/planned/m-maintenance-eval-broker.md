# M-MAINTENANCE-EVAL-BROKER — host-side confined tools for maintenance trials

**Status:** Proposed; implementation requires approval and a sprint plan.
**Target:** v0.55.0 research adapter, proposed; no language semantics change.
**Priority:** P1, blocks the first protected local maintenance campaign.
**Estimated:** 3–5 engineering days including actual OS qualification and review;
an initial estimate, not measured human velocity.
**Dependencies:** M-MAINTENANCE-EVAL-ADAPTER retained-stage instrument and pi 0.85.1;
the existing locked `@anthropic-ai/sandbox-runtime` 0.0.71.
**Created:** 2026-10-23, attended maintenance benchmark exploration.

## Problem and measured evidence

The maintenance adapter now runs a finite eight-trajectory, 24-change no-model
instrument through actual native AILANG/Python grading and native AILANG analysis.
Protected agent execution remains closed. The existing pi native tools run inside
the pi process, whereas bash launches child commands. Oracle protection therefore
needs a kernel boundary around pi to close native filesystem check/use races and
a separate no-network boundary around tool commands.

Actual macOS probes with the locked sandbox runtime demonstrated both halves
separately: protected cat/Python/AILANG reads, symlink/hardlink access, oracle writes,
native read/write path races and tool network access were denied, while ordinary
owned app work ran. The combined outer-pi → inner-bash path failed with:

```text
sandbox-exec: sandbox_apply: Operation not permitted
```

An earlier owned Unix-socket initialization denial was diagnosed separately; an
explicit owned-socket grant did not resolve the nested Seatbelt failure. Reading
the installed SDK's dispatch confirms `enableWeakerNestedSandbox` is supplied to
the Linux wrapper, not its macOS wrapper. This is evidence about this tested SDK/
platform configuration, not a universal claim about every macOS sandbox mechanism.

Removing the outer boundary would reopen native path races. Removing the inner
boundary would give bash the inference process's network authority. Neither is a
qualified comparative lane. The approved adapter plan explicitly calls for a
scoped extension design when its existing primitives cannot satisfy admission.

## Goals and exclusions

Make a protected pi native-tool session and a usable no-network bash call work in
one actual no-model process chain on the target macOS rig. Retain the same native
candidate grader and usage parser. Only after that chain passes may a separately
reviewed model/pricing/bounds manifest admit inference.

This proposal adds a tool transport and controller lifetime management. It adds
no model loop, workflow engine, general remote execution service, Docker/VM
platform, dashboard, package registry release or default benchmark rotation.
Motoko core and language parser/types/effects remain unchanged. It does not
authorize a paid canary or model campaign.

## High-impact decisions

| Decision | Reason | Authority | Deadline | Change cost |
|---|---|---|---|---|
| Launch tool sandboxes from a trusted host broker outside protected pi | Avoid the reproduced nested initialization failure without widening tool authority | User approves design | Before implementation | Medium |
| One stage-scoped local capability, immutable cwd/policy and serial requests | Prevent cross-trajectory authority and keep cancellation bounded | Implementer within this contract | Before dispatch | Medium |
| Reuse existing SDK and native grader protocol | Keep one confinement mechanism and one grading implementation | This design | Implementation | Low |
| Require measured full-chain qualification before admission | Separate primitive tests from a usable protected lane | Existing approved plan | Launch | High |

## Proposed architecture

The trusted pi executor adapter starts a stage-owned host tool broker **before**
launching protected pi. The broker owns the existing SDK manager and immutable
tool policy. A narrowly scoped Unix socket connects the trusted embedded pi
extension to that broker. Only the socket operation needed for that stage is
granted to the outer pi policy; no directory-wide host IPC grant is admitted.

The maintenance bash extension sends a typed request instead of starting a nested
SDK process. The broker creates the command's SDK sandbox as a sibling of pi,
with the existing tools-mode network denial and owned runtime write root. The
outer pi kernel boundary and native tool fence remain responsible for native
file/search/list tools and their filesystem races. Candidate grading continues
to use the already working host-side `EvalContainmentPrefix`; it need not travel
through an agent's socket.

Reuse pi's bash tool interface for stdout/stderr and cancellation. The broker
does not interpret benchmark semantics, oracle output, provider usage or model
prompts. It does not select a policy or cwd from model-supplied options.

### Authority and protocol

Freeze protocol version, stage ID, policy digest, workspace identity and a random
stage capability at controller startup. Every request binds these identities and
a monotonically unique request ID. Reject cross-stage identity, replay, duplicate
keys, unknown fields, malformed frames, unsupported operations and requests after
close before starting a child. The capability is held by the trusted extension;
it must not enter the candidate environment, carried files or model transcript.

The only initial operation is `bash`: command text, a bounded timeout and request
ID. Cwd is the immutable canonical stage workspace. No request may supply an
environment, runtime module, policy path, network domains, executable transport,
arbitrary host cwd or `{runtimeRoot}` expansion. The controller alone constructs
those values. Model command text remains shell input within the existing SDK
boundary; the broker does not attempt to whitelist shell syntax.

Use explicit finite limits: one active command per stage, bounded frame/command
size, bounded output chunks and total output, per-command wall time and the
existing stage/campaign deadlines. Return typed exit/timeout/cancellation/output
truncation/transport failure, never a fabricated successful empty result. Exact
limits belong in the sprint's frozen protocol fixture, with boundary tests.

Use a short stage-owned socket path and create-only ready handshake, verifying
the platform path-length limit. The socket directory and policy assets must be
protected from candidate replacement. A capability leak is not permission to
select a broader cwd/policy; cross-run socket access must still be rejected.

### Lifetime and failure

Start broker → verify readiness/version/policy → launch outer pi → permit calls.
Startup failure refuses inference before pi starts. Parent cancellation, pi
exit, command timeout, malformed transport or broker loss stops the current
command's process group and closes admission. Reap and await children, remove the
stage socket, then finalize checkpoints. Partial output, command attempts and
spent model work remain controller evidence. No retry may relaunch against a
gold starter or disappear from usage coverage.

The current direct candidate bridge kills process groups, and the stage reaper
finds processes by workspace cwd. Independent review identified that these do not
prove cleanup of a descendant that calls `setsid` and changes cwd. Candidate
grading also currently buffers subprocess output without a byte ceiling. These
are additional admission gaps, not properties established by the direct-host
reference controls. The follow-up must qualify ownership of escaped descendants
or mechanically prevent that escape, bound candidate stdout/stderr and report/
snapshot input sizes, and reap each grading runtime before its directory is
deleted. A host broker alone does not establish these properties. The first
prototype must test them; if current host primitives cannot satisfy them, return
an explicit failed qualification before building the remaining transport.

Controller snapshot reads now use descriptor-relative `os.Root` access and
file identity checks, reducing the authority of a raced candidate path. The
concurrency control passes; an actual pre-patch leak was not reproduced. This
hardening is not an atomic whole-filesystem snapshot or proof of child cleanup.

All candidate/broker child environments use the current explicit tools-mode
allowlist. Provider credentials, loader hooks, controller capability and proxy
overrides cannot be inherited. Add canaries for process metadata/environment
reads and socket reuse as well as ordinary file access. Qualify the exact host
runtime/policy/source inventory, including all known source/reference copies,
Git object stores and other trajectories. Trusted runtime read exceptions must
be exact and auditable, with no task/oracle content under them.

## Implementation outline and ownership

| Surface | Proposed change | Initial source/test estimate |
|---|---|---|
| `tools/pi-extensions/maintenance-broker.mjs` | Stage-owned SDK manager, strict local protocol and bounded child execution | 200 / 200 |
| `tools/pi-extensions/maintenance-fence.ts` | Replace nested bash transport with authenticated broker client | 80 / 120 |
| `internal/executor/pi/containment*.go` | Optional host broker lifetime/ready/cancel integration and embedded asset identity | 120 / 140 |
| Existing OS/native integration probes | Actual full chain, outage, replay, network, link/race and protected native grading controls | 50 / 200 |

These are planning estimates. First prototype the actual socket + outer pi +
host SDK child path without any model or credentials. If that cannot satisfy the
frozen boundary, leave admission closed and report the reproduced failure; do not
build out the remaining transport around an assumed mechanism.

## Acceptance and rollout

- [ ] Actual no-model outer process loads the real pi SDK/embedded extension,
  calls brokered bash, and builds/runs an owned AILANG and Python application.
- [ ] Protected reads/writes via cat, Python, AILANG FS, native SDK tools,
  symlink races, hardlinks and recursive searches fail with independent positive
  controls proving the normal application path works.
- [ ] A live reference HTTP server remains unreachable from tool/candidate code;
  socket/proxy reuse and inherited credential/loader/environment paths cannot
  bypass the no-network tools policy.
- [ ] Malformed/replayed/cross-stage requests, policy mutation, broker crash,
  timeout, output cap and parent cancellation leave named failures and no surviving
  children. No paid inference is needed to qualify any of these controls.
- [ ] The protected native grader passes both languages/families, and the 8/24
  retained-source instrument uses the same host policy/transport qualification.
- [ ] Existing nil-containment executor invocations retain behavior. No personal
  or global pi installation is changed.
- [ ] Independent review passes the implemented contract, all completed artifact
  criteria are recorded, and M8/M11 can then be reassessed against real evidence.

After design approval: plan a bounded sprint, obtain execution approval, implement
and qualify, then review a separate priced launch manifest. Keep public development
evidence distinct from future held-out or broader real-project samples.

## Verification log and related work

| Claim | Local verification | Result |
|---|---|---|
| Native pi tools execute in the pi process | Read current fence and actual SDK tool tests | Kernel boundary required in addition to path preflight |
| Existing SDK primitives enforce direct host candidate/file/network isolation | Actual `maintenance-sandbox.test.mjs` canaries with positive controls | Primitive checks pass on tested macOS; not full agent qualification |
| The proposed nested launch is unusable in the tested configuration | Actual nested startup probe, then SDK macOS/Linux dispatch read | Reproduced `sandbox_apply` failure; Linux-only weaker-nesting option cannot be assumed a macOS fix |
| An existing ready-to-use bash broker was not found in the relevant pi surfaces | `rg -n 'broker|net.Listen|createServer|createConnection' internal/executor/pi tools/pi-extensions .pi/extensions` plus existing sandbox launch read | No production host bash broker in these audited surfaces; test HTTP server is a control |
| This design is distinct from shipped pi workflow/tool distribution | Read M-DX-PI-HARNESS and M-AGENT-AILANG-ONLY-EXECUTION | Those distribute tools/policies; this addresses the newly measured host/child launch topology |

Related: [adapter plan](m-maintenance-eval-adapter-sprint-plan.md),
[taskpack design](m-maintenance-eval-taskpack.md),
[pi harness distribution](../implemented/v0_35_0/m-dx-pi-harness.md),
[typed execution lane](../implemented/v0_39_0/m-agent-ailang-only-execution.md).
SimHash related-doc search was run; broad matches were not evidence of semantic
duplication. Neural search was omitted to honor this sprint's no-inference/no-rig
scope. No claim about missing AILANG language or stdlib support is made here.

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | 0 | Execution effects remain explicit; evidence records actual inputs/results |
| A2 Replayability | +1 | Versioned identities and retained command outcomes |
| A3 Effect legibility | +1 | Host command/network authority is explicit in policy and protocol |
| A4 Explicit authority | +1 | Stage capability and fixed cwd/policy, no ambient fallback |
| A5 Bounded verification | +1 | Finite requests/output/deadlines and no-model controls |
| A6 Safe concurrency | +1 | Serial stage commands and owned child lifecycle |
| A7 Machines first | +1 | Typed outcomes instead of ambiguous shell/transport success |
| A8 Minimal syntax | 0 | No language syntax change |
| A9 Cost visibility | +1 | Attempts remain visible after failure; no inference added by transport |
| A10 Composability | +1 | Existing SDK, pi tool contract and native grader reused |
| A11 Structured failure | +1 | Startup/transport/timeout/cleanup failures stay distinct |
| A12 System boundary | +1 | Inference and tool/candidate authority stay separate |

Net proposed score: +10. A1/A3/A4/A7 have no hard negative. No language conflict
surface is triggered: parser, types, compiler, evaluator and effects are untouched.
This is a design assessment, not proof that the proposed transport already works.

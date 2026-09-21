# M-EXECUTOR-POLICY-HARDENING: Make AILANG's Execution Policy an Enforced Boundary

**Status**: IMPLEMENTED
**Target**: Next security release after v0.40.2; release number assigned during planning
**Priority**: P0 — reproduced violations of advertised filesystem and network restrictions
**Estimated**: 15–22 engineering days, including regression coverage and integration (planning estimate, not a sprint commitment)
**Dependencies**: Existing M-AGENT-AILANG-ONLY-EXECUTION and policy admission; Go 1.26.6 already in go.mod
**Routing**: AILANG fix under [PROGRAM](../../PROGRAM.md); no motoko core change
**Created / updated**: 2026-09-21

## Problem Statement

AILANG's policy gate can restrict a program's declared effects, but admission is only one part of authority enforcement. The implementations behind permitted effects must enforce the policy on every operation. Executor tools, compiler inputs, and subprocesses must not offer a second route around it.

An attended audit of checkout `3011da868b02ff9de8b4bb30c77323c410c4ebf8` reproduced three violations through the real effect dispatcher:

| ID | Demonstrated behavior | Implementation cause |
|---|---|---|
| F1 | With FS sandbox set, `readFile("../marker.txt")` reads a disposable marker outside it | `resolveSandboxPath` joins relative input without checking containment of the result |
| F2 | `readFile("link.txt")` follows an in-root symlink to an outside marker | Path-based host file operations follow symlinks after lexical validation |
| N1 | With only `allowed.example` listed, HTTP follows a redirect to `denied.example` | Initial domain check is not repeated by the redirect callback or round trip |

These were local Go tests, not attacks on a deployed cloud worker. The redirect fixture injected DNS and dial hooks so all traffic used one temporary loopback server. Existing admission/profile tests and selected existing FS/network tests passed. Therefore their coverage is insufficient to establish confinement.

Additional source findings require implementation tests: accepted but unused `timeout_ms`; separate streaming network transports; native Pi file tools outside the FS effect dispatcher; regex-based policy parsing and heuristic CLI argument filtering; policy files owned by the execution UID; and Process children receiving a working directory rather than FS/network confinement. These are distinct from the reproduced failures.

The implementation work in this design is the remedy. An external sandbox is not the deliverable or a substitute for repairing AILANG. The remaining trust assumptions are stated explicitly so the repaired runtime makes defensible claims.

## Goals and Invariants

**Primary goal:** A restricted AILANG worker cannot widen operator-granted filesystem, network, tool, or resource authority through any supported execution route.

1. **Filesystem:** every supported file operation is anchored to a trusted root handle; relative traversal, outside symlinks, and concurrent link replacement cannot reach an outside target.
2. **Network:** every actual connection and redirect is authorized by the effective destination policy, including HTTP bytes, SSE, NDJSON, and WebSocket routes.
3. **Authority:** operator policy is decoded once into immutable typed state; agents cannot alter it through tools, source, argv, environment overrides, or writable runtime assets.
4. **Limits:** operator time and effect-operation ceilings survive function calls, imports, budget scopes, retries, and tool invocation. Unsupported guarantees produce explicit refusal.
5. **Evidence:** every original counterexample becomes a permanent passing denial test; the full restricted-tool integration suite exercises calls directly, without depending on a model's cooperation or refusal.

Success is measured by this invariant/test matrix, legitimate-workload compatibility, and bounded failure behavior. A percentage of prompts that a model refuses is not a security metric.

## Threat Model and Scope

Untrusted inputs include generated source, dependency sources, repository files, symlinks, tool arguments, HTTP responses, and model output. The runtime/compiler, trusted launcher, pinned builtins, and operator policy source are trusted implementation components to be audited. An approved host administrator or compromised kernel is outside this design's attacker model.

The protected filesystem root must be a private per-task directory provisioned by the launcher. The launcher must not seed it with hard links to protected files, special device nodes, `/proc`, or externally controlled bind mounts. Root-relative APIs do not sever hard links or make devices safe. Host processes able to change mount topology are outside the boundary; an AILANG program must not receive that power.

The strict policy guarantee covers mediated effects and managed agent tools. Arbitrary native subprocesses cannot inherit that guarantee merely by having their executable name allowlisted. Trusted host execution remains explicit and visibly distinct, as proposed below.

Production deployment exposure and the earlier Fable/Opus refusal transcript were not established by the audit. They are not premises for these fixes.

## Related Documents and Coverage Gate

- [M-AGENT-AILANG-ONLY-EXECUTION](../v0_39_0/m-agent-ailang-only-execution.md): supplies the tool profile and admission plumbing. This design repairs enforcement beneath that plumbing; it does not recreate the profile rollout. Its historical assertion that FS controls work is superseded by F1/F2 for the audited checkout.
- [M-AGENT-SAFE-RUNNER](../../planned/v1_1_0/m-agent-safe-runner.md): broader execution service/message protocol, with its admission spike already used. This design owns runtime confinement, operator limits, and tool mediation; it does not build that service.
- [M-FS-RENAME](../../planned/v0_35_0/m-fs-rename.md): explicitly defers symlink-aware, race-safe confinement across all FS operations. This document supplies that systemic follow-up.
- [M-FS-SANDBOX-DIAGNOSTICS](../v0_19_0/m-fs-sandbox-diagnostics.md): preserve boolean-probe contracts and optional rejection diagnostics while replacing enforcement.
- [M-NET-EFFECT-PROXY-BOUNDARY](../../planned/v0_33_1/m-net-effect-proxy-boundary.md): preserve its explicit proxy route and absence of silent direct fallback; extend per-hop authorization and make proxy trust explicit.
- [M-PROCESS-SUBCMD-ALLOWLIST](../v0_38_0/m-process-subcmd-allowlist.md): prefix matching remains a useful trusted-host restriction, but is not a filesystem/network sandbox.

Coverage search: `ailang docs search --neural --limit 5 'executor security hardening'` reported **0 embeddings, model `fallback-simhash`**. Its scores are not neural similarity and cannot establish the skill's neural duplicate thresholds. Filename/content search and reading the documents above established the distinct ownership. The search also found generated eval-analysis proposals for a VFS; this design reuses the host root-handle API instead of adding a virtual filesystem.

## High-Impact Decisions

These are proposed choices for the user's design approval, not ratified decisions.

| ID | Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|---|
| D1 | Use Go `os.Root` behind one AILANG confined-FS abstraction; migrate every sandboxed operation | Changes filesystem enforcement and resource lifetime across CLI/embed/effects | human | design | high |
| D2 | One destination authorizer and guarded dial path for Net and network Stream operations | Prevents transport-specific exceptions and SSRF policy drift | human | design | high |
| D3 | Add policy `security_mode = "restricted" | "trusted_host"`; absent means restricted; `ailang_only` requires restricted | Policies granting Process or unaudited host effects need explicit migration | human | design | high |
| D4 | AILANG-owned tools perform file/CLI operations through a typed Go authority boundary; native Pi file tools are removed from restricted profiles | Preserves usability while closing non-effect tool routes | human | design | high |
| D5 | Enforce `timeout_ms` and aggregate operator budgets independently of source budgets; use an AILANG supervisor for hard wall-time | Alters currently unenforced policy semantics | human | design | high |
| D6 | Ship Linux/macOS restricted support first; refuse restricted operation on unsupported platforms/backends | Avoids claiming equivalent containment on weaker implementations | human | design | med |

### Design Freeze

- [ ] Approve D1/D2, including symlink compatibility changes and the proxy behavior below.
- [ ] Approve D3/D4 and explicit migration of Process-capable `ailang_only` agents.
- [ ] Approve D5/D6 and the supported-platform/failure contract.

Approval of this document settles these proposals unless the user specifies amendments. It does not approve unrelated queued work or start a sprint.

## Solution Design

### 1. One Typed Policy, Applied at Every Boundary

Keep admission in `internal/policy` and runtime enforcement in core leaf packages. CLI/executor adapters compose them; language packages must not import `internal/executor` or dashboard packages. Use data/interface seams for launcher-provided authority.

Decode TOML strictly in Go. Add a resolved, immutable per-task policy object with explicit mode, root, admitted effects, destination grants, effect budgets, wall-time, input/output ceilings, entrypoint, and tool permissions. Digest exactly the bytes accepted and bank the resolved effective configuration separately; do not reread a mutable path to compute provenance or execute.

Replace `fsSandboxOf`/`policySummary` regex parsing in the Pi extension with Go-produced policy data. Neither TOML spelling nor absent/empty fields may mean different things to the tool wrapper and runtime. Missing policy continues to refuse execution. Unknown effects, unsupported narrowing, invalid limits, and unsafe platform combinations fail at startup.

Review **every effect in the actual registry** for the restricted mode. Initially admit IO, FS, Net, Clock, and Rand only after their relevant adapter/limit tests pass. Stream joins only after section 3 and operation-level restrictions pass. Process, Env, Secret, AI, SharedCache, and any other registry label are refused in restricted mode until an explicit constrained adapter is supplied. This list is a proposed restriction, not a claim that these effects are absent from AILANG. In particular, Stream's process-source/async-process operations must remain denied even if network streaming is permitted; granting the coarse Stream label must not authorize them.

`trusted_host` preserves operator-approved host integrations with conspicuous provenance and no confinement claim. `ailang_only` must reject that mode; trusted host agents use an explicitly configured tool list/full profile. Do not silently downgrade a restricted job or fall back to a different executor when it cannot honor the policy.

Source compilation and execution must consume the same captured module graph. Load entry/import files through authorized roots, bound both per-file and aggregate bytes before allocating full contents, and permit trusted stdlib/package roots as read-only grants. Resolve caches outside agent write authority or verify entries against the captured graph/toolchain. No ambient package download in the restricted worker: dependencies are staged by the trusted launcher. Freeze source bytes through admission and execution, and pin the policy-selected entrypoint rather than accepting an argv override. Record the module-graph digest. This closes check/use changes instead of relying on a repeated typecheck.

Admission review must include an empty-label **open** row: `policy.Check` currently returns for `len(row.Labels)==0` before checking `row.Tail`. Add a direct row test and a compiler-generated fixture if reachable; the observed branch order is a source finding, not a demonstrated language-level bypass. Any non-nil tail must be rejected before a row is classified as pure.

### 2. Root-Anchored Filesystem Operations

Introduce a core leaf abstraction (proposed `internal/fileguard`) wrapping `os.OpenRoot` and root methods. An owner creates it at execution startup, shares it across derived effect contexts, and closes it once after all operations stop. Do not reopen the root from an agent-writable pathname for every operation. Embedded callers receive the same lifecycle API; context copies must not close a shared root prematurely.

An absolute path already lexically inside the configured root may be converted to a relative name for compatibility. Final authorization is always the root-handle operation. Relative names, including `..`, go through the rooted operation; any traversal outside fails. Preserve normal relative paths and safe relative symlinks within the root. Reject absolute symlinks, including those pointing back inside, consistently with `os.Root`; document that migration rather than resolving and reopening them unsafely.

Migrate reads (string/bytes/Result), writes, appends, existence/type probes, listing, mkdir/mkdirAll, remove file/empty directory, and both operands of rename. Preserve file permissions and existing empty-directory-only removal semantics. Use opened handles for capped reads and sorted directory listing. Keep read size checks after reading as well as any stat optimization.

Preserve boolean probes returning false for rejected paths, with the existing optional diagnostics; Result APIs return Err; error-returning APIs report an error. A rejected operation must have no outside side effect. Rename stays within one root and must not fall back to copy/delete across roots. Removal/rename of a final symlink can operate on the directory entry without dereferencing its target; tests must distinguish these semantics from reading through it.

`os.Root` is not sufficient on `GOOS=js` (documented TOCTOU limitation). Restricted mode refuses there instead of providing an unsafe lexical fallback. Trusted existing browser behavior remains separately identified. Linux/macOS root behavior, races, and descriptor cleanup must be tested. Do not add new language syntax or a second custom path parser for confinement.

### 3. Shared Network Destination Enforcement

Factor hostname/protocol/IP rules into a core authorizer used at **every round trip and connection**. A request first validates scheme, hostname/port, and allowlist; then direct mode resolves with context cancellation, validates all candidate addresses, and dials one authorized address without re-resolution. Keep the original hostname for TLS verification/SNI and Host. Reject unsupported URL forms and ambiguous host syntax explicitly. Normalize case/trailing dot consistently in both grants and requests; wildcard matching remains label-boundary-aware.

Every redirect repeats destination authorization before any dial. Cross-origin redirects strip authorization/cookie/proxy credential headers; do not rely solely on default header heuristics when custom headers may carry secrets. Restricted mode should refuse cross-origin redirects carrying operator-designated sensitive headers unless the caller reauthorizes the destination explicitly. Include redirect status/method/body behavior in tests.

Apply the same primitives to GET, POST, generic/string/bytes requests, SSE GET/POST, NDJSON POST, and the registered WebSocket opener. Extend `StreamDialConfig` with a guarded dial/authorization seam so `internal/platform/streamws` cannot re-resolve independently; keep the core free of the platform import. Retain streaming timeouts and frame/message limits without applying a short whole-response HTTP timeout that breaks legitimate streams.

Proxy-selected hostname resolution cannot be locally pinned to the proxy's chosen address. Therefore restricted mode initially **refuses configured proxy use**, with a named explanation, rather than bypassing the proxy or silently claiming destination-IP enforcement. Trusted-host mode retains current proxy semantics and their existing tests. Supporting a constrained proxy protocol is future work; no operator assertion alone counts as equivalent local enforcement.

Restricted mode blocks localhost/private/link-local/metadata targets and ignores no ambient exception: widening options cause explicit refusal. This deliberately excludes local Ollama/network development from restricted mode until explicit address-scoped grants are designed. Direct non-policy development keeps its documented flags. Public-only denial does not imply that every allowed public endpoint is safe to receive task data.

### 4. AILANG-Owned Agent Tool Mediation

Implement a narrow local Go tool endpoint in the AILANG binary, backed by the same resolved authority. A Pi adapter forwards structured requests, never raw shell text. Model-callable operations are an explicit dispatch table, not the whole CLI. Read/write/edit are root-anchored; editing applies the existing expected-content check and writes within the root. Their independent tool grants are explicit in resolved policy; they do not silently grant a submitted program FS.

Replace unrestricted native `read`, `write`, and `edit` in restricted profile expansion with these adapters while preserving user-facing tool affordances. Keep canonical names where possible and test the actual Pi tool registration, not only the generated argv. Update `.pi/extensions/` sources and regenerate embedded copies with the existing asset workflow.

For check/fmt/test/iface/discovery, use per-command typed schemas and authorized module roots. Eliminate generic `argv` path guessing. Attached flag values, response files, package roots, solver paths, output paths, configuration paths, and aliases must either be represented and authorized or rejected. Only bounded, audited operations are exposed; adding a new CLI command never automatically adds a restricted tool.

The tool host retains immutable policy state for the task. Its trusted launcher must keep policy/extension/binary/config ancestors outside all write grants; on Linux ship protected assets owned by a different trusted UID/read-only mount, and in local mode validate the supported provisioning arrangement. Mode bits on files owned by a process with arbitrary native execution are not the security argument. Root-mediated tools plus denying native execution make the restricted AILANG boundary enforceable; do not grant tools to chmod, replace the host, load arbitrary extensions, or mutate launcher state.

Sanitize the restricted worker environment via an explicit allowlist. Keep provider credentials in the model-facing host, not in the AILANG program worker. Internal requests carry a launcher-owned policy handle, not a model-supplied policy pathname. Program stdout cannot spoof the supervisor's decision: use a dedicated framed control channel, keeping stdout/stderr as data. Pin/reuse the admitted module artifact in the worker.

### 5. Operator Limits and Bounded Failure

Place hard wall-time supervision in AILANG's trusted CLI/tool host around a child worker. Start the deadline before source loading and compilation, not after admission. A parent timer remains effective if evaluation or typechecking stops checking context. Reuse `internal/proctree` on supported Unix platforms; cancellation must stop owned children, drain bounded output, close sockets/root handles, and return a structured result. A command outside restricted policy retains existing behavior.

Make `timeout_ms` positive and enforced (absent uses the existing 5000 ms policy default); reject zero/negative/overflow. The wrapper's fixed timeout becomes a slightly larger watchdog derived from the effective deadline, not the authority. Windows descendant termination is not claimed by the existing proctree implementation; restricted support there waits for an equivalent validated mechanism.

Introduce a **shared operator budget** independent of source/per-invocation frames. Charge it once at the existing canonical logical-effect charge scope before the operation. It survives `WithBudget`, imports, functions, retries, and async paths; source annotations can only further restrict it. Do not replace the source budget or count the wrapper and implementation twice. Synchronize shared counters where asynchronous operations can race. Explicit policy zero means zero operations; omission is unlimited for that admitted label and must be shown as such. Absent capability still denies regardless of budget.

Add bounded source/module-graph bytes, tool-request bytes, combined stdout/stderr, individual FS read/write bytes, and network/stream bytes. Proposed operator defaults for restricted runs: 1 MiB entry, 16 MiB module graph, 8 MiB combined output, 8 MiB per FS transfer, existing Net/Stream per-message limits; all are proposed defaults requiring D5 approval. Stop producing output at the cap rather than capturing unbounded output and truncating afterward. Aggregate storage and hard process memory limits require platform enforcement; an unsupported configured hard limit must fail startup. Do not market Go's soft GC memory target as a hard memory boundary. Host CPU/memory isolation is an explicit residual guarantee outside the portable effect sandbox.

Use existing structured error categories where applicable; add a versioned result envelope for runtime denials/limits with stage, effect/tool, policy digest, and safe reason. Allocate any new diagnostic codes only after a namespace search during implementation. Never log secret content or complete denied file contents.

## Conflict Surface

**Syntactic positions:** unchanged; this is runtime/CLI/tool enforcement. There is no new AILANG grammar disambiguation. Effect rows and source budget annotations keep their language meaning. Operator budgets are an additional execution ceiling.

| Existing behavior | Preservation requirement / intentional change |
|---|---|
| Relative and inside-root absolute paths | Continue working through rooted operations |
| Relative symlink staying inside root | Continue working; outside and absolute symlinks become denials |
| Result variants and boolean existence probes | Preserve Err/false shapes and optional sandbox diagnostics |
| Rename directories, lock directories, sorted listing | Preserve behavior for authorized targets; no recursive remove substitution |
| Unsandboxed development CLI | Preserve documented trusted behavior; configuring a sandbox strengthens every FS operation |
| Net proxy and local development | Preserve outside restricted mode; restricted mode now refuses unsupported proxy/local grants |
| Source budgets across nested calls | Preserve per-invocation semantics; add independent aggregate operator cap |
| Policies admitting Process/host effects | Explicit migration to trusted_host plus non-restricted tool profile, or remove those grants |
| Tool check/fmt/test and imports | Preserve intended workflows inside authorized roots; reject undocumented host reach |

Existing fixtures verified present and read:

- `examples/runnable/effects_fs_io.ail`: retain its unsandboxed read/write workflow (hardcoded `/tmp` path); confined tests use an equivalent generated fixture under a private root.
- `examples/runnable/fs_walk_glob.ail`: preserve sorted traversal/glob behavior in trusted execution; equivalent confined fixture exercises rooted listing.
- `examples/runnable/process_subcmd_allowlist.ail`: retains trusted CLI allowlist semantics; restricted mode intentionally rejects Process.
- `examples/safety/net_only_admitted.ail`: remains statically admissible for matching Net policy; runtime tests substitute a hermetic endpoint.

Existing regression contracts to retain include `TestFSSandbox_AbsolutePathWithinSandbox`, `TestFSRenameFile_DirectoryWithinSandbox`, `TestFS_RemoveDirResult_EmptyOnly`, and `TestRunPolicy_DefaultDenyRefusesIO`. Their bodies were read during design: the absolute-path test checks exists/read contents with Sandbox set; the directory-rename test checks a successful rename but does not set Sandbox; the empty-directory test preserves nonempty contents and rejects files. Add genuinely sandboxed rename coverage rather than treating the existing test name as proof. No assumed test-message rewrite is authorized.

## Implementation Phases and File Ownership

These phases are design decomposition; the sprint-planner will create the approved execution plan.

| Phase | Deliverable | Likely files / estimated new or changed LOC | Estimate |
|---|---|---|---|
| M1 | Permanent baseline denial tests; root-handle FS abstraction and full migration | New `internal/fileguard/` ~350–550; `internal/effects/fs*.go`, `context.go`, CLI/embed lifecycle ~400; adversarial tests ~450 | 3–4 days |
| M2 | Shared network authorizer, redirects, Stream transport integration | `internal/effects/net*.go`, `stream_context.go`, `stream_sse.go`, `stream_ndjson.go`, `stream_transport.go`, `internal/platform/streamws/`; new helper ~250; tests ~450 | 3–4 days |
| M3 | Typed immutable policy, mode validation, module snapshot and bounded supervisor | `internal/policy/`, `cmd/ailang/run_policy.go`, `main_run*.go`, `policy_check.go`, loader/runtime integration; new supervisor ~300–500; tests ~450 | 3–5 days |
| M4 | Operator budgets, mediated tools, restricted profile and trusted assets | `internal/effects/context.go`/budget integration; new tool dispatcher ~400–650; `.pi/extensions/ailang-exec.ts`, `ailang-lsp-lite.ts`, executor profile/assets/environment/materialization; tests ~500 | 4–6 days |
| M5 | End-to-end worker/image proof, migration guide and supported-platform matrix | `cmd/ailang/*policy*test.go`, `internal/executor/pi/*test.go`, appropriate existing image tests, `docs/docs/guides/`, example policy and release notes | 2–3 days |

LOC estimates include changed lines and are deliberately approximate; reuse existing infrastructure instead of implementing a second budget engine or process-group manager. M1/M2 may ship as independently useful fixes, but the complete restricted-worker claim waits for M5. Each phase preserves a runnable positive control.

## Acceptance and Testing Strategy

| AC | Required evidence |
|---|---|
| AC1 | F1/F2/N1 are permanent tests that fail on the audited baseline and pass after repair; end-to-end `run --policy` variants prove the same property |
| AC2 | Every registered FS operation and Result variant is covered for traversal, sibling-prefix paths, outside/intermediate/final symlinks, valid in-root symlinks, and failed-operation side effects; rename tests both paths |
| AC3 | Deterministic concurrent rename/symlink-swap tests and fuzz seeds never modify/read the outside sentinel; root descriptors close on success/error/cancellation |
| AC4 | All Net/Stream network entrypoints pass allowed/denied-host, redirect, resolved-private-IP, IPv4/IPv6/mapped-IP, cancellation, and TLS hostname tests; proxy policy fails closed without direct fallback |
| AC5 | Direct tool invocations cannot access outside marker/policy/extension files or pass path-bearing CLI options around the gate; exact registered Pi surface contains only approved tools |
| AC6 | Mutating source/policy/config between admission and execution cannot change the evaluated graph, entrypoint, or effective authority; decisions cannot be spoofed through stdout/stderr |
| AC7 | Restricted admission rejects unsupported effects and Stream process operations; future registry additions default to unsupported until tested; open empty/nonempty effect rows both deny |
| AC8 | Operator budget N permits exactly N charged operations across nested/imported/repeated calls, then denies before side effect; source budgets cannot reset/raise it, and async tests pass the race detector |
| AC9 | Tight deadline terminates bounded synthetic compile/evaluate/blocked-I/O cases; descendant fixtures do not survive; output/input caps stop production/allocation at the boundary |
| AC10 | Four conflict fixtures retain the stated trusted behavior; equivalent confined fixtures pass positive controls; no source annotation/type syntax change |
| AC11 | Unsupported platform, proxy, hard resource limit, missing root/policy, or unsafe launcher provisioning gives a named startup refusal; never a permissive fallback |
| AC12 | All relevant tests pass; docs/example policy/migration guidance updated; effective mode, policy/module digests, tools and enforced limits are banked |

Unit and effect tests use temporary sentinel trees and injected resolver/dialer fixtures. CLI tests build the checkout binary, not whatever `ailang` is installed. Tool tests invoke tools without an LLM. Image tests use disposable workers and synthetic credentials. Do not test by extracting real tokens or connecting to real metadata endpoints.

Run focused package tests during each phase. Final validation includes `make test`, `make lint`, `make check-boundaries`, the existing Pi asset drift check, targeted `-race` tests, Linux/macOS containment suites, and confined-worker integration tests. A socket-bind restriction is an environmental failure to resolve or explicitly report, never evidence of blocked egress. Include bounded fuzz runs and bank seeds. Record source commit and image digest for every integration result.

## Examples of Changed Behavior

| Request | Before (measured/source-supported) | After (required) |
|---|---|---|
| FS read `../marker.txt` under a private sandbox | Outside disposable marker returned | Denied before reading outside data |
| FS read through `link.txt` pointing outside | Outside disposable marker returned | Denied by root-anchored open |
| Allowed HTTP URL redirects to denied hostname | Destination reached in local fixture | Denied before destination dial |
| Policy `timeout_ms = 50` | Parsed field has no Go enforcement consumer | Whole restricted invocation supervised; terminates and reports timeout |
| Native file tool given outside path | Current adapter delegates to Pi; deployed confinement unproven | Go tool authority rejects independently of Pi defaults |
| Policy admits Process with `git:status` | Child gets a cwd and executable-prefix restriction | Restricted admission refuses Process; trusted_host explicitly retains it |

These are policy/tool examples, not new AILANG syntax. Detailed model-free probe construction is recorded below so implementation does not depend on `/tmp` artifacts surviving.

## Verification Log and Reproduction Record

| ID | Claim checked | Instrument / evidence | Result |
|---|---|---|---|
| V1 | F1/F2 observable through real dispatcher | Go overlay test: create root/sandbox plus root/marker; grant FS; set sandbox; call `effects.Call` for `readFile` on `../marker.txt` and symlink `link.txt` | Both returned marker; negative assertions failed |
| V2 | N1 observable without public network | Local `httptest` redirects `/start` to `http://denied.example/final`; allow only `allowed.example`; injected resolver returns public test IP, injected dialer always uses local fixture; call Net/httpGet | `/final` reached; error nil; negative assertion failed |
| V3 | Existing controls still green | `go test ./internal/policy ./internal/executor ./internal/executor/pi -run 'Policy|ToolArgs|ToolProfile|Canonical|AgentPolicy' -count=1` | All selected packages passed |
| V4 | Additional baseline controls | `go test ./internal/policy ./internal/effects -run 'TestCheck_|TestValidateIP_MetadataServer|TestNetCapabilityChecks|TestNetProtocolValidation|TestNetIPValidation|TestNetDomainAllowlist|TestFSSandbox_' -count=1` | Passed |
| V5 | CLI admission and lying import regression | Read `cmd/ailang/run_policy_test.go:56-112`; ran all `TestRunPolicy_*` against checkout | Passed, 33.950s; default-deny and imported effect mismatch explicitly asserted |
| V6 | Timeout has no enforcement consumer | `rg -n 'TimeoutMs' --glob '*.go'` across repository | Only policy field/comment/default; no use in execution |
| V7 | Root API available, limitations known | `go.mod` requires 1.26.6; `go doc os.Root` | Root methods cover read/write/mkdir/rename/remove; JS TOCTOU and mounts/device caveats documented |
| V8 | Existing rooted helper reuse check | `rg -n 'os.OpenRoot|os.Root|OpenInRoot|safeopen' internal cmd` | No matches at audit; proposed leaf helper is new |
| V9 | Separate Stream transport path | Read `stream_context.go`, SSE/NDJSON transport construction, `stream_transport.go`, `stream_ws.go` | URL validation and transport seams confirmed; no runtime Stream exploit claimed |
| V10 | Alternate tool/authority paths | Read executor `toolpolicy.go`, Pi `toolnames.go`, embedded `ailang-exec.ts`, `agent_policy.go`, `environment.go`, effects `process.go` | Native file mapping, regex parsing, ownership-based file protection, inherited environment, cwd-only Process confirmed |
| V11 | Existing cleanup/budget machinery | Read `internal/proctree/proctree.go`, `effects/context.go:362-400`, `budget.go` | Reuse process groups; operator ceiling must coexist with source frames and charge-depth dedup |
| V12 | Pure/open-row branch order | Read `internal/policy/check.go:49-71` | Empty labels return before tail check; language-level exploit unproven; direct unit regression required |
| V13 | Duplicate/coverage gate | Correctly ordered neural CLI query plus filename/content search and related-doc reads | Neural unavailable (explicit fallback); prior documents have distinct/deferred scope |
| V14 | Shared issue, not one-off | Read all FS operation files and network sibling transports; inspect recent FS/net history | Shared path resolver and divergent transports require systemic migration |
| V15 | Four cited example fixtures typecheck | `ailang check` each Conflict Surface fixture, with isolated cache | All four exited 0 on installed v0.40.2-derived binary; syntax evidence only, not a confinement test |
| V16 | Named regression tests actually assert the cited contracts | Read absolute-path, directory-rename, empty-directory-removal, and default-deny test bodies | Contracts verified; rename test notably lacks Sandbox assignment, so new confined coverage is required |

Audited source results used `GOCACHE=/tmp/ailang-security-audit/go-cache`. The installed CLI identified a different dirty commit (`e12e0335d`); it was used for discovery, not as the tested implementation. Original logs/probes were retained in `/tmp/ailang-security-audit/`, but this document's test recipes are the durable specification.

Probe assertion transcript:

```text
TestAuditLocalContainment/../marker.txt:
  CONTAINMENT FAILURE: read disposable marker outside sandbox
TestAuditLocalContainment/link.txt:
  CONTAINMENT FAILURE: read disposable marker outside sandbox
TestAuditRedirectAllowlist:
  ALLOWLIST FAILURE: redirect reached denied.example
  request error: <nil>
```

Implement baseline regressions as tests expecting **denial**, not tests accepting current leakage. For FS use a fresh `t.TempDir`, a nested sandbox, a sibling marker with a unique random sentinel, and a symlink to that marker. For network set `ctx.Net.proxySelector` to direct, supply a resolver returning `203.0.113.10`, and a dial hook that only dials the `httptest` listener. A handler recording `/final` establishes the forbidden connection; do not infer failure only from response text.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | Authorization derives from explicit inputs; wall-clock cancellation is an operational bound, not replay-equivalent program output |
| A2 Replayability | +1 | Bind results to actual policy and immutable module graph |
| A3 Effect Legibility | +1 | Permitted effects map to tested, explicit runtime authority |
| A4 Explicit Authority | +1 | Eliminate alternate tool grants and silent policy widening |
| A5 Bounded Verification | +1 | Hermetic counterexamples, negative tests and bounded fuzzing |
| A6 Safe Concurrency | +1 | Root handles and shared operator ceilings account for races/lifetimes |
| A7 Machines First | +1 | Structured, provenance-bearing refusals and tool schemas |
| A8 Minimal Syntax | 0 | No language grammar changes |
| A9 Cost Visibility | +1 | Report enforced aggregate operation/output/time ceilings |
| A10 Composability | +1 | Reuse leaf abstractions across CLI, embed and tools |
| A11 Structured Failure | +1 | Unsupported enforcement is a startup/runtime decision, not fallback |
| A12 System Boundary | +1 | Distinguish mediated program authority from trusted host execution |

**Net: +10.** Hard checks: A1/A3/A4/A7 have no negative score. This score supports design review; it does not replace user approval.

## Risks, Migration, and Deferred Decisions

| Risk | Mitigation |
|---|---|
| Tightened policy breaks package/development agents using Process or AI | Inventory effective grants before cutover; migrate explicitly to trusted_host or narrow workflows; report changed mode in provenance |
| Root handle leaks or context-copy lifetime errors | One owner, shared borrowed handle, cleanup tests on every exit and concurrent cancellation |
| Transport consolidation changes streaming/proxy behavior | Preserve existing trusted tests; explicit restricted proxy refusal; test TLS/SNI and long-lived streams |
| Tool reimplementation loses edit/check usability | Positive workflow fixtures and pinned-Pi direct tool tests alongside denial tests |
| Scope causes FS/redirect fixes to wait for full tooling work | Allow M1/M2 release independently; keep restricted-worker guarantee unclaimed until M5 |
| Claims outrun enforcement on platforms/resources | Startup validation, explicit unsupported results, documented guarantee matrix |

Implementer may choose helper/type names, fixture organization, dedicated local control-channel framing, and exact result-envelope version after inspecting existing schemas. These choices must preserve the invariants, architecture boundaries, and existing public error contracts. New diagnostic identifiers require collision checks; none are reserved by this design.

Non-goals: changing the type system; formal verification of the whole compiler; implementing a VM/container service; redesigning IAM; guaranteeing that data sent to an explicitly allowed endpoint is harmless; claiming hard portable memory containment from Go runtime settings. Constrained Process/AI/Secret adapters and a verifiable constrained-proxy protocol are future work. Denying them in restricted mode is part of this design, not an implicit promise to sandbox arbitrary native integrations.

## Timeline and Completion

Weeks 1–2: M1/M2 and M3 foundations, with independently reviewable FS/network fixes. Weeks 3–4: finish M3/M4 and integration/migration; retain up to one week contingency within the 15–22 day estimate according to planner velocity.

Completion requires all ACs, documented residual trust assumptions, updated example policies, passing relevant checks, and a review of the actual implementation against this design. Next workflow step after design approval: sprint-planner; execution begins only on the user's execute-sprint instruction.

## Implementation Verification Log (2026-09-21, sprint M-EXECUTOR-POLICY-HARDENING)

Implemented on `dev` in five commits (M1 `93964e315`, M2 `370d76265`, M3 `37ad0bc51`, M4
`70393ba17`, M5 this commit). Every acceptance criterion maps to a named artifact; every
counterexample test was written first and failed on the audited baseline (transcripts in the
commit bodies).

| AC | Evidence |
|---|---|
| AC1 | `internal/effects/fs_containment_test.go` (F1/F2: 118 rows red on baseline), `net_redirect_containment_test.go` (N1: 7 entrypoints red), `cmd/ailang/run_policy_containment_test.go` (F1/F2/N1 end to end through `run --policy` on the checkout binary) |
| AC2 | `fs_containment_test.go`: every op in `Registry["FS"]` (registry-completeness check) × traversal, deep traversal, symlinked file, symlinked dir, absolute symlink, absolute symlinked dir, absolute outside, sibling prefix; outside-tree oracle; both rename operands; positive controls incl. in-root relative symlink |
| AC3 | `TestFSContainment_SymlinkSwapRace` (-race), `TestFSContainment_RootLifecycle` (shared holder, idempotent close, `/dev/fd` count) |
| AC4 | `net_authorize_test.go`: 8 entrypoints × 10 resolved addresses and 8 literal forms with zero-dial assertion; allowed/denied host; TLS SNI = hostname; `RefuseProxy` neither resolves nor dials; cross-origin header stripping; `streamws` dials only through the supplied dialer |
| AC5 | `internal/policytool/policytool_test.go`: outside marker / policy / extension via `..`, symlink, symlinked dir, absolute; path-bearing flags; smuggled fields; gate-only ops — nothing executed; `internal/executor/pi/toolnames_test.go` exact `--tools` surface without native read/write/edit |
| AC6 | `TestRunPolicy_SourceSnapshotFreezesModuleGraph` (mutation-tested), `TestRunPolicy_StdoutCannotSpoofDecision`, `TestRunPolicy_EntryComesFromPolicy`, `internal/loader/snapshot_test.go` |
| AC7 | `TestResolve_RestrictedRefusesUnadaptedEffects` (every registry label outside the six), `TestCheck_OpenEmptyRowDenied` (red on baseline), `TestStreamAsyncExecProcess_RequiresProcessCapability` (red: the subprocess ran), `TestRunPolicy_RestrictedRefusesProcessAndStreamProcessSource` |
| AC8 | `internal/effects/budget_operator_test.go` (exactly N across scopes/Clone; `@limit` cannot raise; `--no-budgets` inert; explicit 0; -race exact), `TestRunPolicy_BudgetsEnforcedAsOperatorCeiling` |
| AC9 | `TestRunPolicy_TimeoutMsEnforced` (blocked sleep, <1s), `TestRunPolicy_TimeoutKillsDescendants` (trusted_host `sh` grandchild dead ≤2s), `TestRunPolicy_OutputCapStopsProduction`, `loader` per-file/aggregate caps |
| AC10 | `make verify-examples` green (199 modules); the four conflict fixtures unchanged; no syntax change (`std/stream.ail` row for `asyncExecProcess` is `{Stream, Process}` — the example already declared it) |
| AC11 | `resolveRunPolicy` platform refusal; `RefuseProxy`; `TestRunPolicyE2E_RestrictedHasNoLoopbackGrant`; `TestResolve_*` consistency rules; `CheckLanePolicy` at both dispatch paths |
| AC12 | `make test`, `make lint`, `make check-boundaries`, `make verify-pi-assets`, `make check-file-sizes`, 13 extension suites; docs: agent-tool-policy guide, LIMITATIONS residuals, debugging guide, example policy, CLI reference; admission line banks mode, digests, limits, budgets |

Findings beyond the audit, closed on the way: the archive builtins (`std/zip`, `std/gzip`,
`std/tar`, `zip_xml`) joined the sandbox and called `os.*` (F1/F2 by another door); `Process.
Authorize(nil)` allowed everything, so `Stream.asyncExecProcess` spawned with only `Stream`; the
prelude `println` charged no budget; `ailang sandbox-check` was a lexical copy of the old resolver.

Design amendments made during implementation (all narrowing): a CLI path handed to the unconfined
`ailang` child must not be a symlink; `trusted_host` honours an explicit loopback entry in
`net_allow` (the hermetic e2e Net test needs a loopback listener and restricted mode has no grant);
the policytool library never execs itself (the CLI names the binary).

**Open at close (operator decision, not code):** three deployed `ailang_only` policies in
`ailang-multivac` (`pkg-ailang-only`, `ailang-only-executor`, `daneel-executor`) admit `Process`
(the last also `AI`, `Net`) and will be refused at dispatch by `CheckLanePolicy` once this binary
ships — migrate before release (drop the grant, or `trusted_host` + `tool_policy: full`).

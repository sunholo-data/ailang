# Sprint M-WORKER-FOLLOWUP

Status: approved for execution by the attended user on 2026-10-11.
Design: [M-WORKER-FOLLOWUP](m-worker-policy-and-lazy-ai.md).
Target: v0.54.5, subject to checking the next available patch before publication.
Estimated duration: 5 engineering days; estimated total: 2100 changed LOC, plus 25% time contingency. Recent seven-day velocity analysis found no reliable measured LOC/day; 420 LOC/day is a capacity assumption.

The existing fix-and-release instruction authorizes implementation, review, merge to dev and publication. Preserve the root checkout's unrelated changes. No milestone pause needs another approval unless scope changes.

## Dependency and ownership

Wave 1: lifecycle M1→M2 and AI M3 independently in isolated worktrees. Wave 2: primary integrates both for M4. Lifecycle owns internal/proctree, new leaf lifecycle transport, internal/effects/process_managed.go and stream_process.go, CLI run policy/bootstrap and runner ownership hooks. AI owns cmd/ailang/ai_setup*, ai_handlers*, configdriven initialization snapshots, main_run_exec.go AI callback and internal/effects/ai*.go. Coordinate any bootstrap/runner interface changes; do not edit each other's files. Integration and release belong to the primary.

## Milestones

### M1: ✅ Private runtime lifecycle and policy lifeline (~650 LOC)
**Dependencies:** none

- [x] AC1: reproduce official v0.54.4 policy leak with positive fixture PIDs/PGIDs and clean only those fixture processes.
- [x] Implement same-executable run detection, bounded versioned private stop/receipt transport, early registration and descriptor rollback.
- [x] Preserve admission fd3; add separate policy worker lifeline, nested cleanup acknowledgement and no admission after stop.
- [x] Test malformed/oversized/forged receipt, supervisor loss, policy timeout/output/credential/provenance invariants and supported-platform guards (AC3–5).

### M2: Managed and async acknowledged cancellation (~570 LOC)
**Dependencies:** M1

- [ ] RequestStop starts bounded protocol cancellation; Join requires inner cleanup receipt, direct Wait and owned I/O completion under one owner deadline.
- [ ] Managed and two async policy workers vanish before successful cancellation; quit/EOF and natural completion join ownership (AC2–3).
- [ ] Missing acknowledgement produces typed failure; forced supervisor death stops inner worker, unrelated bystander survives (AC4).
- [ ] Existing external-command and platform cancellation behavior remains covered; race tests pass.

### M3: ✅ Prepared AI configuration and lazy client creation (~500 LOC)
**Dependencies:** none

- [x] Prepare/freeze structural model/provider/routing/auth-source configuration eagerly without reading credentials or probing a real provider.
- [x] Initialize AIContext once at first authorized AI operation; cache complete context or error and preserve concrete optional interfaces.
- [x] Cover call/json/step/cache/stream/image/base64/refs/speech and metadata-only access; preserve capability/budget/error/trace behavior (AC7–9).
- [x] Direct CLI and generated bin offline/help pass with absent/unreadable keys and unavailable ADC/Ollama; real AI use fails loudly without credentials (AC6).
- [x] Explicit eager setup and stub fixtures retain existing behavior; focused and race tests pass.

### M4: Integration, independent evaluation and follow-up release (~380 LOC)
**Dependencies:** M2, M3

- [ ] Add documented showcases and fixtures, update example manifest and changelog fragment; exercise all real-bin/policy acceptance probes.
- [ ] Full Go tests, lint, formatting, boundaries, file sizes, meaningful race checks and cross-platform CI pass (AC10).
- [ ] Independent sprint evaluator assesses every AC; address findings before merge.
- [ ] Commit completed plan/JSON and validation evidence in the implementation tree, then merge reviewed PR to dev.
- [ ] Publish patch release; verify checksum/provenance and repeat tests with official artifact, rerun pinned consumer policy gate before claiming resolved (AC11).
- [ ] Complete release-manager μRAG, cloudTEST→PROD and public MCP version checks; send authorized release notice.

## Day allocation and risk controls

Days 1–2: M1 transport/policy bootstrap and failing regression probes. Day 3: M2 ownership integration; M3 can proceed concurrently. Day 4: AI interface/error/configuration coverage and integration. Day 5: M4 evidence, independent evaluation, CI and release. Calendar duration depends on CI/cloud gates. Descriptor leaks, policy hard-stop semantics and optional AI interfaces are the main risks. Do not weaken receipt semantics or refresh provider configuration at first effect to simplify implementation.

## Registry reuse

Searched `process` and `AI`; inspected `sunholo/external_backend@0.2.0` info/docs. It wraps one-shot exec, excludes long-lived workers and AI, and cannot control host pre-admission lifecycle or client creation. M1–M4 each choose `none`; reuse existing proctree, owners, policy and AI factory. No new package dependency or package publication.

## AILANG syntax gate

AILANG prompt version loaded during preparation: v0.16.7. Executor must reload current prompt before the first .ail edit and record it.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| process_policy_cancel.ail | Pure success classifier contract; OS lifecycle skipped with reason: proved by positive-PID integration tests | IO, Env, Process explicit on entrypoint | Pure success/error classifier cases; no subprocess inside inline test |
| policy companion worker | Contract skipped with reason: effectful sleep/IO fixture only | Clock and IO; policy explicitly grants and confines fixture | Skip with reason: timed behavior verified by subprocess lifecycle tests |
| ai_offline_cli.ail | Pure mode parser contract for known offline/help choices | IO, Env, AI explicit; no AI on offline/help path | Empty args, help, offline and AI mode cases |

Strict VM offline/help is covered; native VM AI dispatch remains outside this sprint. Live provider calls are unnecessary: use stubs and local mocked providers, and preserve missing-auth failures. No claims about remote provider inference cancellation.

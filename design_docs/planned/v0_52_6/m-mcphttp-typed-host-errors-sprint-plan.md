# Sprint Plan: M-MCPHTTP-TYPED-HOST-ERRORS

**Issue:** Refs #1602 — sunholo-data/ailang (existing issue; no new issue)
**Design:** [Approved design](m-mcphttp-typed-host-errors.md)
**Status:** Implemented; M1–M3 completed on 2026-10-09, evaluation handoff ready
**Target:** v0.52.6
**Duration:** 1.5 days (6 hours work + 2 hours contingency; remaining wall time for review/checks)
**Risk:** Low implementation complexity, medium compatibility risk

## Summary

Let a Go host opt into per-message MCP JSON-RPC errors through `protocol.JSONRPCError` on the error returned by `Invoker.Invoke`. Preserve every existing untyped callback envelope. The deliverables are the exported hook, wire and facade regression tests, and embedding documentation that lets World adopt the hook after a published release.

## Current status and velocity

Read issue #1602 and its comments via GitHub REST on 2026-10-08: open, zero comments. Its preferred error-interface option matches the approved design. Reviewed `interfaces.go` and `mcphttp/methods.go`: InvocationResult still contains only Value, and Invoke errors still escape callTool as host errors. The design's systemic audit covers envelope construction, batch dispatch, hostcall runner, facade adapters, A2A, BearerGate and the SDK parity reference; the proposed seam addresses all Invoke errors without expanding the scope.

Current checkout: `coordinator/task-1526d099`, version v0.52.5, clean at intake. The handed-off design is present despite its original branch being `coordinator/task-f8d32bed`. No branch switch is needed.

Ran `scripts/analyze_velocity.sh 7`. This is a shallow checkout with one visible commit (35c4dfb3, a design document on 2026-10-08), and the script reports no usable recent LOC metrics. A measured LOC/day cannot be inferred. Use the approved design's bottom-up 3h/2h/1h estimates plus 33% contingency, and a planning capacity of 180 LOC/day over 1.5 days. Estimate: 270 added LOC, comprising 25 production, 200 tests/examples, and 45 documentation/changelog LOC; this is a planning target, not measured velocity. Coverage is not measured during planning; execution must cover the branch matrix below rather than invent a repository coverage percentage.

## Registry reuse audit

Executed `ailang pkg search mcp` and `ailang pkg search jsonrpc`. MCP returned `sunholo/email@0.9.0`, `sunholo/mcp_oauth@0.1.1`, and `sunholo/mcp_files@0.1.4`; JSON-RPC returned no packages. Inspected info/docs for OAuth and files and info for email. They provide AILANG service features (OAuth, file transfer, email tools), not a Go Invoker error contract. Email's advertised query-layer exports already exclude it from this capability. The CLI warned the binary may be stale; registry results were nevertheless retrieved successfully.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | The existing stdlib-only Go protocol seam must define and consume the interface; registry AILANG packages cannot change this seam. |
| M2 | none | Wire and facade fixtures test the repository's handler and existing adapters; no registry dependency supplies these regression assertions. |
| M3 | none | Embedding guide/changelog describe the repository's Go contract; no package reuse applies. |

## Milestones

### ✅ M1: Optional hook and Invoke mapping (~125 LOC)

**Estimate:** 25 production + 100 tests/example LOC; 3 hours, Day 1.
**Dependencies:** None.
**Files:** `serveapi/protocol/interfaces.go`, `serveapi/protocol/mcphttp/methods.go`, new `serveapi/protocol/mcphttp/typed_error_test.go`.
**Example:** Test host error type in typed_error_test.go demonstrates direct and wrapped hook use.

Declare `JSONRPCError` embedding error with `JSONRPCError() (int, string)`. In callTool's Invoke error branch use stdlib errors.As, check code != 0 and message != "", and return errorResponse(msg.ID, code, "%s", message), nil. The literal format preserves percent signs in host text. All other errors follow the existing host error return.

**Acceptance criteria:**

- [x] protocol.JSONRPCError embeds error and exposes JSONRPCError() (int, string), with the nonzero-code/nonempty-message rule documented.
- [x] Single tools/call returns HTTP 200 SSE with exact host code/message and request id, including percent signs, quotes and Unicode.
- [x] errors.As resolves a fmt.Errorf %w wrapped typed error; code zero, empty message, plain and wrapped untyped errors retain the frozen envelope.
- [x] Hook tests exercise every SupportedVersions entry; go test ./serveapi/protocol/mcphttp and make check-protocol-closure pass.

**Risk and mitigation:** Malformed hooks or formatted messages change the wire contract; cover guard cases and use a literal %s format. Runner-generated sentinel errors retain their existing path.

### ✅ M2: Batch and facade compatibility (~100 LOC)

**Estimate:** 100 tests/example LOC; 2 hours, Day 1.
**Dependencies:** M1.
**Files:** `serveapi/protocol/mcphttp/typed_error_test.go`, `serveapi/embedded_mcp_test.go`.
**Example:** Mixed-batch fixture plus facade host returning World's effects-unrecorded style code/message.

Pin complete SSE response shapes and ids for a mixed batch and a facade invocation. Keep the existing frozen callback test unchanged, adding a separate typed-error test. Reuse existing fixtures and runner tests for sentinel paths. Run the SDK differential without adding a host-error row: its reference discards Invoke errors.

**Acceptance criteria:**

- [x] A mixed success/typed-error/ping batch returns one HTTP 200 SSE array with every response matched to its own id.
- [x] A typed error reaches MCP through serveapi.New without production adapter changes.
- [x] TestEmbeddedMCPFrozenCallbackEnvelopes passes unchanged; timeout, capacity and cancellation mappings remain frozen.
- [x] Existing handler, gate, tool hints, bearer gate and SDK parity tests pass; no parity row is added for host errors because the SDK reference discards them.

**Risk and mitigation:** Typed errors could abort a batch or change adapter behavior; full wire goldens and facade tests prove response collection and pass-through.

### ✅ M3: Documentation and delivery checks (~45 LOC)

**Estimate:** 45 guide/changelog LOC; 1 hour, Day 2.
**Dependencies:** M1, M2.
**Files:** `docs/docs/guides/serve-api.md`, `changelogs/v0.32-current.md`; Go example coverage in the M2 facade fixture (or an Example function there).
**Example:** Go host error type and wire response in the embedding guide, exercised by the test fixture. No .ail files need edits.

Document opt-in behavior, nonzero/nonempty guard, wrapping, HTTP 200 SSE and batch response semantics, and fallback. Correct the adjacent stale SDK dependency claim. Add an Unreleased entry; use a v0.52.6 heading only if the release state has advanced. Run the full checks and record their results in the implementation PR.

**Acceptance criteria:**

- [x] serve-api.md documents a Go host error example, guard, errors.As wrapping, per-message SSE/batch behavior and frozen fallback; stale MCP SDK dependency sentence is corrected.
- [x] changelogs/v0.32-current.md has an Unreleased entry for typed MCP host errors, referring to #1602.
- [x] The Go host example is exercised by an executable Go example or the facade fixture; no AILANG syntax change or .ail example is required.
- [x] go test ./serveapi/... -count=1, make test, make lint and make check-protocol-closure pass.
- [x] Implementation PR body includes Refs #1602; release/pin follow-up is recorded for World without claiming this sprint publishes a release.

**Risk and mitigation:** Documentation could imply World is already unblocked; distinguish merged code from the published release and World pin update.

## Day-by-day schedule

- Day 1 morning: M1 contract, mapping and focused unit tests (3h).
- Day 1 afternoon: M2 mixed-batch goldens, facade case and compatibility regressions (2h).
- Day 2 morning: M3 guide/example and changelog (1h); full checks and debugging contingency (2h). Leave remaining wall time for review.

## Validation and success metrics

All new hook branches and fallback cases have direct tests, including wrapped errors and every supported MCP version. Mixed batch and facade paths have wire assertions. Existing frozen envelope, gate and parity suites remain green. Required commands are `go test ./serveapi/protocol/mcphttp -count=1`, `go test ./serveapi/... -count=1`, `make check-protocol-closure`, `make test`, and `make lint`; `make check-boundaries` is also required if implementation expands across architectural layers. Use real Go package commands rather than the design's ambiguous `make test serveapi/...` shorthand.

The production protocol closure gains only stdlib errors. Keep CallbackMessage, WriteMCPEnvelope, serveMessages, facade adapters, BearerGate and A2A behavior intact. InvocationResult.IsError, error.data and _meta forwarding are outside this approved sprint.

## Dependencies, delivery and approval

No implementation dependency blocks M1. World consumes pinned releases: a later approved release containing this change and a World pin update are required to unblock its client. Publishing or version bumping the repository is outside this planning task; the implementation changelog documents the delivery requirement.

Link only upstream issue #1602 in sprint JSON. World PR numbers and historical upstream #885 are provenance, not issues this sprint resolves. Every plan/implementation PR body must include `Refs #1602`; do not create a duplicate issue or automatically close #1602 during planning.

No design decision remains open. Test placement is frozen for this plan as the new typed_error_test.go plus a separate facade test. Use the coordinator's plan-review gate: merging the sprint-plan PR approves the plan and triggers sprint-executor. Do not start execution or send an independent executor trigger before that approval. Subsequent implementation must run sprint-evaluator against the approved design and this checklist.

## Executor handoff

Sprint ID: `M-MCPHTTP-TYPED-HOST-ERRORS`.
Plan path: `design_docs/planned/v0_52_6/m-mcphttp-typed-host-errors-sprint-plan.md`.
Progress path: `.ailang/state/sprints/sprint_M-MCPHTTP-TYPED-HOST-ERRORS.json`.
Execute M1 → M2 → M3 after sprint approval. Expected effort: 6h + 2h buffer; estimated LOC: 270. All milestone passes/start/completion fields remain null at planning completion.

## Execution record (2026-10-09)

M1 and M2 were committed separately. M1 wire tests failed before implementation
in all six direct/wrapped typed cases across the three supported versions, then
passed with the hook. M2 pins mixed batch responses and the wrapped host example
through public `serveapi.New`; frozen callback tests remain unchanged.

The explicit executor dispatch overrides the original checklist: the changelog
is `changelogs/unreleased/2026-10-09-mcphttp-typed-host-errors.md`; full `make test`
is deferred to CI, and the implementation PR body uses `Closes #1602`.
The original checklist remains above for planning provenance; checked validation
items mean the dispatch's replacement checks were run, with environment failures
recorded below rather than a claim that full `make test` passed.

The design's untyped batch description claimed a first-request id. The existing
frozen envelope actually uses `id: null` for a batch. The extra untyped batch
golden preserves that behavior; no production envelope changes were made.

World delivery still requires a later release and dependency pin update.
Independent sprint review found no implementation defects; final validation
results are in the companion implementation report and sprint JSON.

Final checks: `go test ./serveapi/... -count=1`, `make lint` (0 issues),
`make check-protocol-closure`, `make check-boundaries` and
`make check-file-sizes` pass. `make test-core` was attempted: all packages pass
except eight existing SQLite brain tests in `internal/effects`, because
`CGO_ENABLED=0` and no C compiler is available. No compiler was installed.
Full `make test` was not run. Implementation report / ready PR body:
`.ailang/state/implementation/m-mcphttp-typed-host-errors-pr.md`.

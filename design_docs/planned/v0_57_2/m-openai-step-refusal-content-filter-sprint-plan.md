# Sprint Plan: OpenAI Step refusal/content_filter classification

**Sprint:** M-OPENAI-STEP-REFUSAL-CONTENT-FILTER  
**Design:** [Approved design](m-openai-step-refusal-content-filter.md)  
**Issue:** #2123 (keep open; use `refs #2123`, never auto-close keywords)  
**Status:** Planning complete; execution not authorized or dispatched.  
**Duration:** 3 working days, 8.5 hours milestone work + 2.5 hours integration/uncertainty buffer = 11 hours.  
**Estimate:** 600 changed/added LOC, including synthetic tests and documentation. Risk: medium.

## Authorization and design rulings

The current handoff says “Previous work has been approved” and explicitly requests sprint planning. That approval is the planning basis for the design's High-Impact Decisions D1–D5, despite its original unchecked design-freeze boxes. Carry these as rulings: typed non-retryable failure; nil response plus typed error; `CodeContentFilter = "ContentFilter"` registered in both retry tables; Anthropic Generate/Step/StreamStep alignment included; explicit error envelopes take precedence. No human design decisions remain open under this handoff. The original design-only restrictions still prohibit implementation, approval, merge, deploy, and executor dispatch in this task. A later explicit “execute sprint” authorization is required.

## Current status and evidence

Planning checkout: `coordinator/task-bb2cf347`, HEAD `c2fbb9b5`, clean at entry, std/VERSION `v0.57.1`. The archived design baseline is `8fc51e764026c4171e8cc82827c08832b3fdfc4f`; do not represent it as the current checkout. This clone is shallow and contains one visible commit, the merged design (#2125). The velocity script ran for seven days: no usable implementation velocity or LOC/day can be established. Its historical changelog matches are not recent velocity. Use the design's approximately 11-hour estimate with explicitly provisional planning capacity of 200 LOC/day; this is not a measured rate. Milestones fit 60–90 minute executor sessions.

Source checks confirm missing Chat refusal fields, generic content_filter mapping, two retry tables, and independent ChatGPT SSE decoding. No implementation of this design is present. `step.go` is 704 lines and `step_test.go` 663; keep new classifier/tests in companion files to stay below 800 lines. `streamstep.go` is 344, Anthropic client/step/streamstep are 478/435/438 lines.

Case A is separate: local search finds no `classifyStreamChunkError`/`chunk.Error`; `2063f4ceb` is not an available object. Absence in a shallow clone does not establish ancestry or release inclusion. The design's archived review and reporter follow-up are the available comparison evidence; direct diff with that source is outstanding. No duplicate Case-A implementation is authorized.

## Execution dependency and terminal behavior

Before implementation, inspect the execution base and obtain the adjacent completed Case-A change (`2063f4ceb` or its verified successor) for direct comparison. Record source identity, parser/error mapping changes, tests, and ancestry evidence without promising a release. Do not run a passing composition claim on a base with no envelope handler. Integrate/rebase only under later execution authorization. If the source cannot be obtained or reconciled, **STOP: commit current documentation work, report `BLOCKED: Case-A source unavailable for composition verification`, and end the run**. Do not recreate the envelope handler, silently skip the composition test, or claim this sprint ready to merge. This is a source dependency, not a reopened semantics decision.

## Registry reuse audit

Ran `ailang pkg search refusal` (no results) and `ailang pkg search openai` (only `sunholo/mcp_files@0.1.4`). Inspected that candidate with `pkg info` and `pkg docs`: it supplies MCP file upload flows, not Go provider protocol decoding or error taxonomy. The CLI warned its binary may be stale; results are evidence from the available registry client, not an exhaustive ecosystem claim. Every milestone M1–M6 chooses **none**: this work belongs in existing host Go adapters/tests, cannot depend on an AILANG package without crossing architectural layers, and reuses the existing ai error model/parsers instead. No package contribution or new dependency.

## Milestones

### M1: Typed failure and retry contract (~90 LOC)

**Duration:** 1.5 hours. **Dependencies:** none after execution dependency check.

Update `internal/ai/errors.go`, add `errors_refusal_test.go`; update `cmd/ailang/configdriven_streaming.go` with companion `configdriven_streaming_refusal_test.go`. Avoid global string heuristics that would change arbitrary errors; typed errors are the contract, and any legacy Anthropic signature match must be narrowly justified and tested.

**Acceptance criteria:**

- [ ] CodeContentFilter is a typed non-retryable error through IsRetryable, ShouldRetry (including wrapped errors), and cmd isAICodeRetryable.
- [ ] Ordinary error codes, transient errors, and unknown-error behavior retain existing classification.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** retry classification regression; use positive/negative control tables.

### M2: Buffered Chat refusal classification (~100 LOC)

**Duration:** 1.5 hours. **Dependencies:** M1.

Add `internal/ai/openai/refusal.go` for the shared classifier; update `step.go` wire field and parse integration. Add `step_refusal_test.go`; change only the existing content_filter expectation in `step_test.go`. Decode usage before returning errors and record refused telemetry at the owning span; do not introduce response+error returns.

**Acceptance criteria:**

- [ ] Refusal-only, content_filter-only, combined signals, and refusal with stop return nil response plus typed error preserving cause.
- [ ] Classification precedes Hermes tool recovery; stop, length, tool_calls and usage controls remain unchanged.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** losing usage on nil response; assert span values before return.

### M3: Streaming Chat refusal accumulation (~130 LOC)

**Duration:** 1.5 hours. **Dependencies:** M2.

Update `internal/ai/openai/streamstep.go`, add `streamstep_refusal_test.go`. Accumulate refusal fragments across SSE events and arbitrary reader boundaries, including transport splits inside JSON lines. Test empty/null refusal, partial content plus filter, exact concatenation, late usage, and no refusal callback. Preserve scanner/network and existing in-band errors.

**Acceptance criteria:**

- [ ] Split refusal deltas and later content_filter finish reasons preserve exact cause; refusal emits no content callback.
- [ ] Late usage survives in telemetry before nil response plus typed error; ordinary content, thinking, tools, usage and transport failures retain behavior.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** delta ordering and callback regressions; split-event and reader-boundary fixtures.

### M4: Responses refusal items (~65 LOC)

**Duration:** 1.0 hours. **Dependencies:** M1.

Update `internal/ai/openai/types.go` and `responses.go`; add `responses_refusal_test.go`. Explicitly decode the refusal payload, not merely its item type. ChatGPT uses its own event/item structs in `internal/ai/chatgpt/client.go`, so it does not inherit this change. Record that gap as a follow-up; new ChatGPT refusal-event semantics and Responses incomplete_details remain out of scope.

**Acceptance criteria:**

- [ ] OpenAI Responses refusal items return CodeContentFilter preserving refusal text, including mixed text/refusal output.
- [ ] Ordinary output_text and usage controls pass; incomplete_details and independent chatgpt SSE changes remain deferred.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** assuming shared wire types; test the actual Responses decoder.

### M5: Direct Anthropic parity (~100 LOC)

**Duration:** 1.5 hours. **Dependencies:** M1.

Update `internal/ai/anthropic/client.go`, `step.go`, and `streamstep.go`; use companion `refusal_test.go`. Reuse the shared ai error code without importing OpenAI into Anthropic; provider-specific signal extraction stays local. Preserve refusal cause and minimum span attributes across all three paths.

**Acceptance criteria:**

- [ ] Generate, Step and StreamStep refusal return the same typed non-retryable failure with existing refused telemetry preserved.
- [ ] Synthetic buffered and streamed refusal plus normal end_turn, max_tokens and tool_use controls pass.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** provider-specific drift; exercise all three public paths.

### M6: Delegation, composition and documentation (~115 LOC)

**Duration:** 1.5 hours. **Dependencies:** M2, M3, M4, M5.

Add `internal/ai/openrouter/refusal_test.go` and `internal/ai/ollama/streamstep_refusal_test.go`; add composition cases in OpenAI companion tests. Update `internal/ai/provider.go` contract comment and add `changelogs/unreleased/2026-11-04-openai-step-refusal-content-filter.md` opening with `### `. Add a short compatibility/testing evidence section to this plan at execution completion.

**Acceptance criteria:**

- [ ] OpenRouter Step/StreamStep and ollama /v1 streaming fixtures prove inherited classification without duplicated provider logic.
- [ ] Explicit error envelope wins over refusal on the reconciled Case-A source; existing envelope classification and retryability are unchanged.
- [ ] Focused tests, make test-changed, make check-file-sizes, and make check-changelog pass; compatibility docs and unreleased fragment describe the behavior change.

**Example artifacts:** synthetic JSON/httptest/SSE fixtures in the named companion tests; no `.ail` examples or paid calls. **Risk:** unavailable Case-A source; apply the terminal dependency rule above.

## Day-by-day sequence

- Day 1: dependency/source comparison, M1 and M2; start focused provider tests (3 hours plus buffer).
- Day 2: M3 and M4 (2.5 hours plus buffer); verify usage/telemetry ownership through actual client entry points.
- Day 3: M5 and M6 (3 hours plus buffer); record focused validation and compatibility evidence. Run sprint-evaluator only after future authorized execution completes.

## Validation and success metrics

All fixtures use local httptest servers or in-memory readers, never paid providers. Matrix covers refusal-only, content_filter-only, combined signals, refusal with stop, null/empty refusal, split deltas, partial content, tools, reasoning, late usage, and ordinary stop/length/tool_calls. Assert `errors.As` finds `*ai.AIError`, CodeContentFilter, Retryable=false, ShouldRetry=false, nil response, and exact refusal cause. Telemetry minimum: error.message, error.category=refused, ai.finish_reason, ai.tokens_in/out. Streaming callbacks already delivered before refusal cannot be retracted; document terminal failure and ensure refusal text is never delivered as content. Test wrapped error propagation into the cmd AIError record without changing its shape.

During implementation run targeted tests for each touched package and refusal/control cases, then `make test-changed` (touched packages plus test-core), `make check-file-sizes`, and `make check-changelog`. Use a verified available `TEST_CHANGED_BASE` if origin/dev is absent; record its commit and cover the entire sprint diff. No full `make test` gate on the executor; CI owns the full suite. Format touched Go files and run scoped vet/lint. No boundary change is planned; run `make check-boundaries` if integration introduces a cross-package dependency. No stdlib/examples changes, so verify-stdlib/verify-examples and AILANG syntax loading are not applicable. Plan creation itself requires JSON validation and consistency checks, not implementation test execution.

Coverage target is every signal/entry-path combination above with positive controls, rather than an invented global coverage percentage. Acceptance requires no changed success mappings or ordinary usage, preserved Case-A behavior on reconciled source, no duplicated OpenRouter classification, and all focused gates passing. Document that the behavioral Go API change replaces nil error with typed error and discards partial response under D2; AIError record shape stays unchanged.

## Deferred scope and completion rules

Independent ChatGPT SSE refusal handling, Gemini safety parity, Responses incomplete_details, live refusal callbacks, and dual-return partial output remain follow-ups. No provider prompts, new finish-reason vocabulary, non-2xx translation changes, release promises, issue closure, or execution dispatch. Keep issue #2123 open throughout this planning task. Sprint JSON remains not_started with all passes=null; artifact validation does not confer execution authorization.

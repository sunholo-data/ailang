# Sprint Plan: M-JSON-NUMBER-ROUNDTRIP

## Summary

Restore finite float64 JSON round trips, including negative zero and large integer-shaped literals, across the registry interpreter, legacy evaluator and strict-bytecode VM. Share the formatter and propagate decode errors at every number conversion site.

**Design:** [m-json-number-roundtrip.md](m-json-number-roundtrip.md)
**Duration:** 3 days, approximately 12 implementation hours plus 3 hours contingency.
**Estimated changes:** 360 LOC: 100 implementation, 230 tests, 30 example/documentation.
**Risk:** Medium, chiefly nested decode error propagation and backend test access.
**Status:** Planned; implementation has not started. Design approval is supplied by the handoff. Sprint-plan approval remains the coordinator review/merge gate.

## Current Status and Assumptions

The working branch is coordinator/task-6b491b30; the task-0039cf2e design is already present and the checkout was clean. std/VERSION is v0.51.0, although the approved artifact targets v0.50.2. Retain the requested artifact directory; at execution time place the changelog entry under the actual unreleased version, without a version bump or release.

Source inspection confirms the three encoder heuristics and decoder conversions remain. The registry makeJNumber and VM vmMakeJNumber currently return values without errors: changing helper signatures alone is insufficient; their token-builder callers must handle and return errors. Legacy interfaceToJSON already has an error return and recursively propagates it.

The seven-day velocity script finds only the design-document commit, no usable implementation LOC/day. Its historical changelog matches are not recent velocity evidence. Capacity is therefore an explicit estimate: 120 changed LOC/day over three days, with 25% time contingency over the design's 12 hours. The 360 LOC estimate exceeds the design's approximate 200 to allow real backend property tests and error-propagation coverage. No coverage baseline is claimed: this environment lacks Go, make and jq; capture focused coverage in the executor environment.

Treat the handoff's approval as approval of D1 (window rule, fixed notation for 1e-6 <= abs(f) < 1e21) and D2 (-0.0 text), despite the design's stale unchecked freeze checklist. Do not silently substitute blanket g formatting. Follow D3-D6 unchanged. No encodeNumber public API is needed.

## Registry Reuse Audit

Commands: ailang pkg search json; ailang pkg search float; ailang pkg info sunholo/a2ui; ailang pkg docs sunholo/a2ui. JSON search returned nine consumer/application packages; float search returned none. a2ui exports UI components, and its docs command reports no AGENT.md. No registry package can replace a host-side builtin codec. M1, M2 and M3 each have decision **none**, with reuse of existing internal/eval and Go standard-library primitives; no new package dependency.

## Milestones

### M1: Canonical number formatter and encoder parity (~110 LOC)

**Duration:** Day 1, approximately 4 hours.
**Dependencies:** None

Create internal/eval/json_number.go and json_number_test.go. Update internal/builtins/json_encode.go, internal/vm/builtins_json.go and internal/eval/builtins_json.go; update existing encoder tests, including the legacy 1e10 expectation. The formatter uses Signbit for zero, shortest f/e digits by window, exponent zero cleanup and null for non-finite values.

**Acceptance criteria:**

- [ ] All three encoders delegate float formatting to eval.FormatJSONNumber with no float-to-int conversion.
- [ ] Boundary table covers signed zero, threshold neighbors, subnormals, 2^53, 2^63, maximum float and non-finite values.
- [ ] Negative zero emits -0.0; 1e21 emits 1e+21; 1e-7 emits 1e-7; NaN and infinities emit null.
- [ ] IntValue/TagInt encoding remains exact integer text and existing mid-range cases pass.

### M2: Float decoding with propagated range errors (~100 LOC)

**Duration:** Day 2, approximately 4 hours.
**Dependencies:** M1

Update internal/builtins/json_decode.go (makeJNumber and token builder), internal/vm/builtins_json.go (vmMakeJNumber and token builder), internal/eval/builtins_json.go (interfaceToJSON), and all three decoder test files. Check errors before inserting numbers into containers; malformed numeric conversions must never become partial Ok trees.

**Acceptance criteria:**

- [ ] All three number decoders use Float64 without the Int64 text heuristic.
- [ ] Registry and VM token builders propagate conversion errors through existing Result Err envelopes, including nested arrays and objects.
- [ ] Positive and negative 1e19 decode without saturation; -0 and -0.0 preserve their sign bits.
- [ ] Positive and negative overflow tokens return Err; underflow behavior is pinned to the installed Go ParseFloat behavior, propagating every non-nil error.

### M3: Backend round-trip verification, example and release notes (~150 LOC)

**Duration:** Day 3, approximately 4 hours.
**Dependencies:** M1, M2

Extend internal/builtins/json_encode_test.go and json_decode_test.go, internal/eval/json_test.go, internal/vm/builtins_json_test.go or an external test package capable of accessing the registered paths. Add examples/runnable/json_number_roundtrip.ail, builtin metadata and CHANGELOG.md. Prefer existing test harnesses; do not export internals merely for differential testing.

**Acceptance criteria:**

- [ ] Boundary table and at least 10000 seeded random finite float bit patterns round-trip bit-identically through actual registry, legacy and VM codec paths.
- [ ] Differential tests assert byte-identical encoder output across all three implementations.
- [ ] New examples/runnable/json_number_roundtrip.ail passes check and has identical expected output in interpreter and strict-bytecode modes.
- [ ] json_jint.ail runs successfully; ai_call.ail and claude_haiku_call.ail type-check and their max_tokens JSON remains 100 without requiring network calls.
- [ ] Metadata and CHANGELOG describe intentional formatting and overflow-error behavior changes.
- [ ] Focused tests, make test, make lint, make check-boundaries and make simplicity-audit pass or infrastructure failures are explicitly reported.

## Day-by-Day Execution and Validation

Day 1: establish failing encoder boundary tests, implement the shared formatter and migrate three callers (35 implementation + 75 test LOC). Compare finite nonzero text with encoding/json as an independent oracle; pin negative zero separately.

Day 2: establish failing large-literal/signed-zero/nested-overflow tests, migrate three decoders and propagate errors through callers (65 implementation + 35 test LOC). Test root, array and object errors, both overflow signs, subnormal minimum, underflow, and nearest-even rounding at 2^53. Do not claim underflow always errors: assert the actual Go ParseFloat result and error contract.

Day 3: add backend differential/property tests (120 test LOC), example and metadata/release notes (30 LOC), run the full gates and reserve 3 contingency hours. Use a fixed seed, generate float bits directly, skip non-finite patterns, and compare Float64bits rather than numerical equality. Exercise each real codec path, not just FormatJSONNumber followed by strconv.ParseFloat. Legacy non-finite output changes are intentional.

Before writing the example, load the use-ailang skill and run ailang prompt. Then ailang check the new example and run it with IO in interpreter and --strict-bytecode modes. Use existing JSON fixtures; type-check the two API examples and test their JSON payloads locally to avoid live API prerequisites.

Validation commands in a Go-equipped executor: go test ./internal/eval ./internal/builtins ./internal/vm; focused go test -cover for formatter and conversion coverage; make test; make lint; make check-boundaries; make simplicity-audit. Cover every formatter branch and each new decode error branch; record coverage without inventing a repository-wide percentage. Preserve unrelated failures with evidence rather than expanding scope. Native amd64/arm64 runs are desirable if available; deleting every float-to-int formatting conversion removes the reported architecture-dependent guard.

## Risks, Scope and Completion

Error propagation is the critical path: top-level and nested errors must surface as Result Err consistently on all backends. Exponent thresholds need adjacent representable-value tests using math.Nextafter. Non-finite values have a null policy and are excluded from the finite round-trip law.

Keep IntValue bridge encoding unchanged. No std/json.ail change, no compiler/type/effect change, no bridge conversion fix, arbitrary precision, compiled-Go JSON implementation, or full tree-codec refactor. Flag the stapledons-godot workaround as removable after consumers upgrade; do not edit or message that external project in this sprint.

Completion requires all milestone acceptance criteria, the new verified example, explicit intentional text-change documentation, and the sprint-evaluator review against the approved design. The JSON starts with all passes/started/completed values null. No implementation or tests were run by this planning stage.

## Coordinator Handoff

Plan and JSON are ready for coordinator review. Per sprint-planner/resources/coordinator.md, merging this coordinator sprint-plan PR approves it and triggers sprint-executor. Do not send a duplicate execution message or self-merge. No source issue number was supplied; #1464 is the design-document PR, not a verified source bug issue, so github_issues remains empty.

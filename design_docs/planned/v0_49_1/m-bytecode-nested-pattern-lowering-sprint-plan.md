# Sprint Plan: M-BYTECODE-NESTED-PATTERN-LOWERING

## Summary

Restore evaluator/strict-bytecode parity for recursively nested match-arm patterns, including the reported `_ :: x :: _` binding failure and the A1–A4/B1–B4 wrong-arm matrix. Use the evaluator's seven pattern cases as reference semantics; preserve syntax, effects, list arity rules and the instruction set.

**Design:** [Approved source design](m-bytecode-nested-pattern-lowering.md)
**Duration:** 3 engineering days, approximately 18 hours including 25% investigation/validation buffer.
**Target:** v0.51.3 integration, because corrected ADT tag access is specified by a v0.51.3 sibling. Keep artifacts in v0_49_1 to preserve existing links; release assignment is not a promise to publish.
**Risk:** Medium-high: shared lowering code, unsafe projections, and ordered arm fallthrough.
**Status:** Plan prepared for coordinator review; execution not started. Design approval is supplied by task-08c86c94's handoff. Sprint approval remains the coordinator merge gate.

## Current implementation and estimate evidence

At planning time this worktree is clean on coordinator/task-51b86f34; std/VERSION and installed CLI report v0.51.0. The CLI is b99dd25-dirty and warns about staleness: build from source for acceptance, never treat the installed binary as proof of the eventual fix.

Code inspection confirms lowerPatternBindings skips nested list tails/elements; lowerPatternCond omits tuple/record subconditions; extractBindingsAndGuards only parks nested constructor arguments in temps. Constructor conditions still project FieldAccess("Tag"), and if-chain guards still reference bindings before their declarations. The compiler already short-circuits OpAnd (control_flow.go), so ordered structural conditions can safely gate projections.

The design contains systemic analysis across both dispatch paths and all seven pattern kinds, plus 38 recorded verification entries. No implementation milestone is complete. Optional external quorum was unavailable, not passed; the supplied design approval permits planning without claiming quorum clearance.

The velocity script (7 days) found one aggregate checkout commit and no usable LOC/day metrics. That commit contains the repository snapshot, so its LOC count is not development velocity. Estimate from the design's 2–3 days and localized scope: 650 changed LOC total (280 implementation, 310 tests/fixtures, 60 documentation), about 217 LOC/day as planned capacity, not a measured historical rate. No coverage baseline was run during documentation-only planning; branch and integration coverage requirements below are execution checks.

## Registry reuse audit

All milestones: **none**, package null. Registry search terms: not applicable. These tasks modify compiler-internal Core-to-statement lowering and its first-party regression harness, rather than introduce a package-like capability. An AILANG registry dependency cannot implement the compiler phase that compiles that dependency. Reuse internal/eval/eval_patterns.go as the semantic reference, existing statement IR/builtins, existing lowering tests, and existing CLI parity-test conventions. No registry search or external dependency is warranted.

## Dependencies and integration order

1. Integrate the corrected `_adt_tag` mechanism from ../v0_51_3/m-vm-adt-tag-check-lowering.md first: VM ADTObj.Ctor name access, compiler builtin registration, and appropriate Go-emitter handling. Never use record field "Tag" to inspect an ADT. This prerequisite is absent in the inspected lowering. If it has not landed, M1 ports the minimum corrected mechanism with its tests within this sprint, avoiding duplicate registration. No new opcode.
2. Integrate bind-then-guard control flow from ../v0_51_2/m-vm-ifchain-tag-guard-lowering.md next, preserving the remaining-arm continuation on a false guard. M1 ports this prerequisite if absent. Deduplicate its ADT-tag work against step 1.
3. Land this recursive rewrite after those prerequisites. For ../v0_51_2/m-vm-var-pattern-default-arm.md, preserve a prior merge or merge it after this rewrite against the final helper interfaces. Its guarded-var routing must reuse step 2; do not copy the old guard-before-binding shape.
4. Rebase/integrate serially: all four designs touch internal/gen/lower/match.go. Record actual sibling commits or local ports in sprint notes at execution start. This is an integration order, not a claim that sibling branches have landed.

The two corrected Design Freeze choices are adopted by this plan. M1 validates their actual availability. Do not block the reported cons fix indefinitely on a sibling release: the approved design explicitly includes these corrections, so local minimal ports are within scope. Broader sibling CLI/Go-emitter improvements remain with their owners unless required for the new shared builtin to compile.

## Proposed milestones

### M1: Recursive conditions and bindings in if-chain lowering (~300 LOC)

**Estimate:** 160 implementation + 140 tests/fixtures. **Duration:** Day 1, 6 hours. **Dependencies:** None.

Update internal/gen/lower/match.go and extend internal/gen/lower/lower_match_test.go; add internal/gen/lower/match_nesting_test.go if keeping the recursive matrix separate is clearer. Port the minimum tag/guard prerequisites described above if absent; likely touched support files are internal/vm/builtins.go, internal/bytecode/compiler/builtins.go and internal/gen/emitgo/ builtins handling, following sibling implementation conventions.

Introduce recursive condition/binding helpers covering Var, Wildcard, Lit, List, Tuple, Record and Constructor. Check list shape before indexing, outer tag before projecting constructor fields, and recurse into tails with their own length condition. Emit literals at every depth. Sort record fields; allow extra fields exactly as the evaluator does. Determine from CoreTypeInfo whether required-field presence is statically guaranteed; if not, emit a safe presence check instead of letting a missing-field projection trap. Use fresh temporaries to prevent nested/arm name collisions. Bind variables only after the complete pattern succeeds; evaluate explicit guards afterward; on false continue to later arms.

Examples to create/update: plan an IO-free examples/runnable/bytecode_nested_patterns.ail with the exact string second-element probe, empty/singleton defaults, deeper cons and nested list/tuple/record probes. Obtain `ailang prompt` before writing .ail and check each fixture.

Acceptance criteria:
- [ ] Exact handoff probe returns `b` on evaluator and strict VM; empty and singleton lists return the empty string.
- [ ] Recursive helper tests cover all seven pattern cases with matching and failing structural/literal checks; unsupported Core pattern kinds fail loudly through existing lowering error handling.
- [ ] Nested lists and tuples do not index short/empty inputs; ADT conditions never call record access for tags.
- [ ] Guards can reference nested bindings and false guards continue to the next arm; record extras and field ordering preserve evaluator behavior.
- [ ] Existing lower_match_test.go assertions remain unchanged and pass.

Risk: unsafe eager projections or temp collisions. Mitigate with short-input, wrong-tag, repeated nested-variable-position and multi-arm tests.

### M2: Recursive constructor-switch integration (~200 LOC)

**Estimate:** 120 implementation + 80 tests/fixtures. **Duration:** Day 2, 5 hours. **Dependencies:** M1.

Update extractBindingsAndGuards/lowerConstructorMatch in internal/gen/lower/match.go to delegate nested argument conditions/destructuring to the same recursion. Preserve ordinary switch bindings and jump-table dispatch for flat cases. Nested conditions must succeed before nested bindings or user guards run.

Audit same-tag arms: nested mismatch or false guard must try the next eligible arm in source order, rather than immediately jumping to a wildcard. Group same-tag candidates with an ordered continuation inside a switch case, or route only affected matches to the correct if-chain path. Preserve existing flat literal-guard assertions and semantics; do not broadly redesign switch IR. Ensure var defaults receive the scrutinee if referenced, coordinating with the var-default sibling; a failing pattern must not leak its bindings.

Tests: extend lower_match_test.go/match_nesting_test.go. Extend examples/runnable/bytecode_nested_patterns.ail with nested ADTs in arguments and in list/tuple/record positions, including repeated constructor arms and default paths.

Acceptance criteria:
- [ ] Some(Some(7)) returns 7 and Some(None) chooses the correct later/default arm under strict VM.
- [ ] Constructor arguments containing lists, tuples, records and literals recurse with complete bindings and conditions.
- [ ] Repeated same-tag arms and false guards preserve evaluator source order; referenced var defaults bind correctly.
- [ ] Flat constructor, literal-argument, cons-head constructor and switch regressions pass without weakening existing assertions.

Risk: SwitchStmt has no per-case fallthrough. Mitigate with ordered same-tag matching/no-match tests and preserve the existing fast path where valid.

### M3: Strict parity matrix, corpus audit and documentation (~150 LOC)

**Estimate:** 90 tests/fixtures + 60 documentation. **Duration:** Day 3, 7 hours including buffer. **Dependencies:** M1, M2.

Add cmd/ailang/nested_pattern_parity_test.go using existing command-level parity test conventions and IO-free testdata as needed. Finish examples/runnable/bytecode_nested_patterns.ail; update docs/LIMITATIONS.md or bytecode reference only where existing pattern restrictions need correction, and add a focused CHANGELOG.md entry. Preserve the source design's verification history; record acceptance corrections and actual integration decisions in this plan/progress notes.

Each A1–A4/B1–B4 row must assert a hand-computed expected value as well as engine equality. Supplement with nested-seven-kind positive/negative rows, wrong tags, wrong literals, short lists, exact-tail rejection of extra elements, guards over nested bindings, record extras, repeated same-tag arms, depth-four patterns and non-tail match contexts already supported by FlattenBlock. The source doc's earlier numeric cons2 example expects 2.0; this handoff's distinct string cons2 expects b. Do not copy the numeric expected output into the string regression. Validate deep a::b::c::rest with [1,2,3,4] returning 6.

Run `make build`, targeted `go test ./internal/gen/lower ./internal/bytecode/compiler ./internal/vm ./cmd/ailang`, `make test-core`, `make test`, `make lint`, and `make check-boundaries`. Extend tests to touched support packages if M1 ports tag access. Run `ailang check` and evaluator/strict-bytecode probes with the freshly built bin/ailang. Include existing recursion_quicksort, list_pattern_cons, pattern_sugar and cons_expression regression fixtures. Use pure entry probes for IO examples: unrelated IO bridge gaps must not hide pattern results.

Audit std/, examples/ and docparse/ (where present) using existing compiler/corpus tooling: bank module/function, EvalOnly reason, and pass/fail per relevant nested-pattern fixture. Acceptance is zero pattern-lowering-caused unbound-variable/EvalOnly failures in the relevant checked corpus, not zero unsupported VM functions of any kind. Explicitly report absent corpus directories or independent unsupported contexts; do not silently declare a full sweep passing.

Acceptance criteria:
- [ ] All eight design rows equal both engines' hand-computed expected result; all seven recursive pattern branches have positive and negative parity coverage.
- [ ] Fresh-binary strict probes pass without an evaluator bridge, including the exact string handoff and deep-cons result 6.
- [ ] Existing fixture outputs/type checking remain unchanged; corpus audit records no unresolved pattern-lowering binding failures.
- [ ] Targeted and full checks, lint and boundaries pass, or external failures are recorded with reproducible evidence before evaluation.
- [ ] Documentation accurately describes completed coverage and remaining independent limitations; sprint-evaluator can trace each design criterion to evidence.

## Execution schedule and success metrics

Day 1: confirm integration state, failing regressions, recursive if-chain implementation and safety checks. Day 2: constructor-switch recursion and ordered fallback. Day 3: complete strict matrix, corpus audit, full validation, docs and evaluator handoff. Total 650 changed LOC; approximately 217 LOC/day capacity. Coverage target: all seven recursive branches and their success/failure paths, both dispatch paths, and every named regression; no arbitrary repository-wide percentage claim.

Non-goals: new syntax/opcodes, evaluator changes, new non-tail match-expression support, IO bridge work, list representation/performance optimization, or release/publish actions. Unsupported shapes must produce a reason rather than silently drop bindings. If the prerequisite port exceeds the allowance or requires new IR, stop that portion with evidence and revise the plan rather than self-expand scope.

## Handoff

Progress artifact: .ailang/state/sprints/sprint_M-BYTECODE-NESTED-PATTERN-LOWERING.json. Features are initially not started, with populated reuse decisions and full dependencies. Coordinator review/merge approves this plan and triggers sprint-executor; do not dispatch execution independently before that approval. sprint-executor records prerequisite decisions and milestone evidence; sprint-evaluator then assesses the approved design and this plan. No implementation was performed in the planning stage.

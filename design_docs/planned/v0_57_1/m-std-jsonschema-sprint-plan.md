# Sprint Plan: M-STD-JSONSCHEMA

Approved design: [m-std-jsonschema.md](m-std-jsonschema.md). Planning handoff: task-7802df55.

## Summary

Ship pure strict-by-default Draft 2020-12 validation, its explicit lenient and schema-only APIs, and std/list.all. Duration: **5 engineering days**, including approximately 25% integration/semantics buffer; **10 execution milestones**, each targeting 60–90 minutes of implementation plus its focused checks. Budget: **2300 LOC**, approximately 1400 implementation and 900 tests/fixtures/docs. High risk: hand-rolled annotation and dynamic-reference semantics. This is an estimate, not measured throughput; split a milestone further if its focused tests cannot fit one executor run.

## Current status and velocity

The checkout is clean before planning. std/VERSION is v0.57.0. No jsonschema builtin/module exists per the approved verification log; current registration follows json_decode.go. list_iterative_search.go is 193 lines, registry_codegen_list.go 302, std/list.ail 334: no immediate size split needed. The existing list HOF source explicitly points to separate VM implementations, so M8 includes VM parity omitted from the design file list.

The velocity script found no usable recent LOC metrics; shallow git history exposes one documentation commit dated 2026-11-03. No empirical LOC/day or milestone completion rate can be claimed. Capacity assumption: 460 LOC/day, 2300 total over five days, following the design's five-day estimate with smaller execution slices. No coverage baseline was measured for this planning-only change; implementation must cover each supported keyword with positive/negative cases rather than claim an invented coverage percentage.

## Rulings and implementation precision

All design-freeze decisions remain ruled: hand-rolled Go engine, no new dependency, pure API, strict default, local refs, RE2, format annotation-only and documented float64 numbers. No human-decision gates remain. Execution still follows coordinator plan approval; this artifact authorizes no self-merge or implementation during planning.

Use canonical sorted schema property/keyword traversal, preserving array order, to satisfy the design's permutation-invariant error guarantee; schema insertion order cannot satisfy that guarantee. Apply 2020-12 items only beyond prefixItems (without prefixItems it covers all elements). Both are implementation precision within the approved dialect, not reopened design choices. Do not silently approximate dynamic scope or annotation propagation. Official suite vendoring, maxErrors and clients-repo migration remain deferred; write focused local fixtures without adding those projects to the sprint.

## Registry reuse audit

Executed ailang pkg search jsonschema (no packages), json and list (adjacent JSON consumers only). Inspected sunholo/a2ui@0.2.0 with pkg info: component-tree JSON rendering; pkg docs failed explicitly because no AGENT.md exists. It provides no validator. Every M1–M10 decision is **none**: existing std/json/regex/list conventions are reused internally, but no registry dependency supplies this frozen stdlib capability. Per-milestone decisions and evidence are populated in the JSON.

## Milestones

### M1: Compiled schema model and strict audit (~260 LOC)

**Dependencies:** None

**Files:** internal/builtins/jsonschema_compile.go; jsonschema_compile_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Compile boolean schemas and validate keyword value shapes across the frozen vocabulary; audit every subschema including unused $defs and unselected branches. Ignore annotation payloads as data. Strict errors name unknown keywords; lenient skips only unknown keywords, never malformed supported keywords.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M2: Local references and pattern compilation (~280 LOC)

**Dependencies:** M1

**Files:** internal/builtins/jsonschema_refs.go; jsonschema_refs_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Resolve root #, escaped RFC 6901 pointers, anchors and dynamic anchors; reject dangling/non-local refs. Compile pattern and patternProperties using RE2; reject lookaround/backreferences with schema errors. Pin dialect, relative $id and vocabulary handling.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M3: Scalar assertions and deterministic errors (~250 LOC)

**Dependencies:** M1

**Files:** internal/builtins/jsonschema_validate.go; jsonschema_scalar_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Validate type including integer-valued floats, enum/const structural equality, numeric limits and multipleOf, Unicode string lengths and patterns. Emit RFC 6901 paths and fixed canonical keyword ordering independent of schema insertion order.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M4: Object and array validation (~280 LOC)

**Dependencies:** M3

**Files:** internal/builtins/jsonschema_containers.go; jsonschema_containers_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Validate required/properties/patternProperties/additionalProperties/propertyNames/dependentRequired; array prefixItems/items/contains/minContains/maxContains/uniqueItems and cardinalities. items applies after prefixItems per Draft 2020-12. Collect evaluated locations for subsequent annotation propagation.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M5: Combinators and reference execution (~260 LOC)

**Dependencies:** M2, M4

**Files:** internal/builtins/jsonschema_applicators.go; jsonschema_applicators_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Implement allOf/anyOf/oneOf/not/if/then/else/dependentSchemas and ref siblings. Track dynamic scope for $dynamicRef; bound cycles at the same instance location to 64 while allowing recursive descent to child instance locations. Failed branch diagnostics do not invalidate a successful anyOf or conditional predicate.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M6: Unevaluated annotation semantics (~240 LOC)

**Dependencies:** M5

**Files:** internal/builtins/jsonschema_annotations.go; jsonschema_annotations_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Propagate successful-branch evaluated-property/item annotations through refs and combinators; apply unevaluatedProperties/unevaluatedItems last. Test failed branches, multiple successful anyOf branches, contains matches and conditional annotation isolation.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M7: Public builtin and stdlib surface (~200 LOC)

**Dependencies:** M6

**Files:** internal/builtins/jsonschema.go; std/jsonschema.ail; tests/stdlib/jsonschema_test.ail; make/test.mk. **Duration:** 60–90 minutes plus focused verification.

- [ ] Register three pure builtins with Json ADT inputs and record-list outputs; export ValidationError, validate, validateLenient, validateSchema. Schema compilation failures return before instance evaluation. Rebuild binary, check std module and run seventh stdlib suite after suite count 6 to 7.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M8: List all with interpreter and VM parity (~130 LOC)

**Dependencies:** None

**Files:** std/list.ail; internal/builtins/list_iterative_search.go; registry_codegen_list.go; internal/vm/builtins_hof_list.go; internal/vm/list_hof_parity_test.go; internal/builtins/list_all_test.go. **Duration:** 60–90 minutes plus focused verification.

- [ ] Export all(pred,xs); default to _list_all Go twin and VM HOF support discovered beside any. Test true/false/empty lists, short circuit and interpreter/VM agreement. A pure negated-any wrapper is an already-authorized alternative if it preserves parity; test either choice.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M9: Census fixture and performance evidence (~220 LOC)

**Dependencies:** M7

**Files:** internal/builtins/jsonschema_census_test.go; jsonschema_bench_test.go; testdata/jsonschema/. **Duration:** 60–90 minutes plus focused verification.

- [ ] Use a synthetic fixture exercising every reported census keyword and counts; label it synthetic and assert the census programmatically. Pin expected errors, 100-run and permuted-key determinism. Benchmark compile plus validate on roughly 460 schema nodes and 1000 document properties; report machine and time against 50 ms target.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

### M10: Example reference docs goldens and focused gates (~180 LOC)

**Dependencies:** M8, M9

**Files:** examples/runnable/jsonschema_validate.ail; docs/docs/reference/std-jsonschema.md; docs/docs/reference/stdlib.md; .stdlib-golden/jsonschema.json; .stdlib-golden/jsonschema.sha256; .stdlib-golden/list.json; .stdlib-golden/list.sha256; changelogs/unreleased/2026-11-03-std-jsonschema.md. **Duration:** 60–90 minutes plus focused verification.

- [ ] Ship runnable envelope self-check, API/strictness/error-root guidance, RE2 divergences, format annotation-only and float64 limitations. Freeze new jsonschema and changed list interfaces with existing tooling; fragment opens with ###. Run focused integration gates and record results.
- [ ] Focused tests pass; format Go changes and keep touched files below 800 lines.

**Example coverage:** M10 supplies examples/runnable/jsonschema_validate.ail for M1–M7/M9; M8 adds an all call to that demo.

## Day-by-day schedule

| Day | Work | Estimate |
|---|---|---|
| 1 | M1 strict compiler; M2 references/patterns | 540 LOC |
| 2 | M3 assertions; M4 containers | 530 LOC |
| 3 | M5 applicators; M6 annotations | 500 LOC |
| 4 | M7 atomic public surface; M8 list parity | 330 LOC |
| 5 | M9 census/performance; M10 docs/example/goldens; integration buffer | 400 LOC |

## AILANG syntax and showcase checklist

AILANG prompt version loaded: **v0.16.11** (ailang prompt --version-active); loaded the current prompt via ailang prompt during planning. The CLI warns the binary may be stale; rebuild from this checkout before checking new builtins and reload the prompt at execution start.

For examples/runnable/jsonschema_validate.ail:

| Row | Decision |
|---|---|
| contracts | skip: schema conformance is a runtime Json predicate; the new builtin is opaque to SMT, so no unsupported solver proof is promised. |
| effects | include: main() -> () ! {IO} prints the result; check(schema: Json, doc: Json) -> bool ! {} stays pure. |
| inline tests | include: pure allPositive(xs: [int]) -> bool with tests [([], true), ([1, 2], true), ([0], false)], calling std/list.all; integration suite covers Json validation. |

The library wrappers remain pure with empty effect rows. Load the teaching prompt before any .ail edit; run ailang check and inline tests on the new demo using the rebuilt binary.

## Validation and completion criteria

Use make test-changed (touched Go packages plus test-core), focused jsonschema/list/VM tests, make test-stdlib-ail, make verify-stdlib, make verify-examples, make check-file-sizes, make check-boundaries and make check-changelog. Run relevant lint for touched packages. Do not require make test in executor runs; the full suite belongs to CI. This overrides the design's older full-suite checklist. Use existing make freeze-stdlib tooling and review only intentional jsonschema/list golden changes. Write an unreleased changelog fragment rather than appending to the historical v0.32 file.

No touched Go file may exceed 800 lines: compiler helpers go in jsonschema_refs.go, container rules in jsonschema_containers.go, applicators in jsonschema_applicators.go, annotation logic in jsonschema_annotations.go; tests are split by concern as above. Inspect VM file size before editing and create builtins_hof_list_all.go if necessary.

Completion requires all supported vocabulary keyword cases, refs/siblings/cycles/dynamic scope, annotation propagation, deterministic error arrays and schema-vs-instance root messages; both-engine all parity; a checked/runnable example; both goldens; and measured end-to-end benchmark evidence. The external real client schema and original 4+6 violations are reported context, not locally reproduced evidence. The synthetic census fixture must not be presented as the original envelope.

## Dependencies, risks and handoff

No new Go dependency or external client checkout is required. M8 can proceed independently but the sprint executes the recorded order. Highest risk is annotation flow through successful branches and dynamic reference scope; test those before API publication. Numeric float64 and RE2 differences are documented accepted limitations, not silent fallbacks.

No open human decisions. JSON status remains not_started with null milestone results. Coordinator approval of the plan triggers sprint-executor; return both artifact paths for that handoff and do not manually duplicate the coordinator dispatch. After implementation, run sprint-evaluator against the approved design and this plan. Reporter follow-up is a coordinator deliverable after implementation/release, not a claim that the clients repository has already migrated.

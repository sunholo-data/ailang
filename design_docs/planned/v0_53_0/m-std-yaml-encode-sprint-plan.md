# Sprint Plan: M-STD-YAML-ENCODE

## Summary

Add pure `std/yaml.encode(Json) -> Result[string, string]` using an ordered, handwritten Go emitter. Ship the approved block dialect, round-trip tests, a runnable config-edit example, and reference documentation without changing decode.

**Design:** [m-std-yaml-encode.md](m-std-yaml-encode.md)
**Status:** Implementation complete; independent evaluation pending. Repository-wide test gate remains red; see final validation below.
**Duration:** 2 days, 10 focused hours (8h design estimate + 25% buffer).
**Estimated size:** 510 added/changed lines: 150 implementation, 250 tests, 110 docs/example/metadata.
**Risk:** Medium — nested indentation and decode-side numeric/key-order behavior.
**Target:** v0.53.0 planning folder; no release/version bump in this sprint.

## Current Status and Velocity

Inspected on 2026-10-07 at base `3053434e`, std version v0.52.5. The approved artifact was absent in this worktree; recovered verbatim from `origin/coordinator/task-e89f7399` commit `2c7bfc569ce0be7fb77a1bab35fe7ea248e0de09` and included beside this plan.

- `internal/builtins/yaml.go` (94 lines) registers only `_yaml_to_json`; `std/yaml.ail` exports only `yamlToJson` and `decode`. No encode implementation is present.
- Existing `yaml_test.go` has 149 lines. JSON escaping, `FormatJSONNumber`, and Result helpers already exist and are reused.
- Decode still uses yaml.v3 generic maps followed by JSON marshaling; keys are sorted. Encode preserves list order independently of the sibling work item.
- The velocity script found one recent triage commit, no usable LOC/day metrics, and no recent diff statistics in this shallow checkout. Historical changelog snippets are not a measured rate. Use the design's 8h breakdown plus 25% contingency; 255 LOC/day is a planning allocation, not observed throughput.
- No implementation tests or repository-wide coverage baseline were run during planning. Executor captures focused emitter coverage and runs the checks below.

## Registry Reuse Audit

Ran `ailang pkg search yaml` and `ailang pkg search json`. YAML search returned no packages. JSON search returned nine consumers/helpers (http_helpers, logging, a2ui, duckdb, linkedin, external_backend, email, agui, decisions), none advertising YAML serialization; no YAML candidate warranted `pkg info`/`pkg docs`. CLI emitted a stale-binary warning; these are recorded search results, not evidence about source correctness.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | No registry YAML encoder; the approved builtin reuses local JSON escaping, number formatting and Result helpers. No new package or Go dependency. |
| M2 | none | Export and integration tests belong to the existing std/yaml boundary and shared std/json ADT. |
| M3 | none | Reference docs and runnable example document that same stdlib API, not a separate package capability. |

## Milestones

### M1: Ordered YAML builtin and acceptance tests (~340 LOC)

**Effort:** 5h; 140 implementation + 200 Go test lines.
**Dependencies:** None.
**Files:** `internal/builtins/yaml.go`, `internal/builtins/yaml_test.go`.
**Example coverage:** M3's `examples/runnable/yaml_roundtrip.ail` exercises this builtin through the stdlib.

Register `_yaml_encode` via `RegisterEffectBuiltin`, with pure metadata and `Json -> Result[string, string]` builder type. Walk Json TaggedValues and ordered object pair lists into a builder; use existing `escapeString`, `FormatJSONNumber`, `wrapOk`, and `wrapErr`. Handle both FloatValue and IntValue payloads without converting large ints to float. Keep file sizes within repository conventions.

Pin scalar bytes, root and nested arrays/maps, sequence-in-sequence and map-in-sequence indentation, empties, escaping, controls, Unicode, ambiguous strings, and unusual keys. Exercise finite float boundaries, subnormals and large magnitudes, signed zero, and large int payloads through the real existing decode bridge. Compare decoded numeric semantics at the Json boundary, avoiding a new normalization algorithm. Add repeat-call byte determinism and recursively order-insensitive comparison for unsorted unique-key objects.

**Acceptance criteria:**

- [ ] Registration is pure and typed `Json -> Result[string, string]`; no new dependencies or map/sort in the emit path.
- [ ] All keys and string values are double-quoted; two-space block indentation, flow only for empty collections, exactly one trailing newline, no tags/markers/anchors/wrapping.
- [ ] Unsorted and duplicate keys emit in their original list order; re-decoding duplicates returns Err naming the duplicate key.
- [ ] NaN and both infinities, including nested values, return Err rather than null or YAML non-finite tokens.
- [ ] Scalar and nested byte fixtures parse via the real yamlToJSONImpl; finite unique-key values round-trip modulo recursive object order under today's decoder.
- [ ] Focused YAML Go tests pass, with at least 90% statement coverage of new emitter helpers; existing YAML/JSON builtin tests remain green.

**Risk:** Handwritten indentation and decoder float-range behavior. Mitigate with byte fixtures and actual bridge acceptance tests. If a finite numeric fixture fails the approved contract, surface the exact mismatch for design review; do not silently weaken the guarantee or alter decode.

### M2: Stdlib export, integration tests and golden (~65 LOC)

**Effort:** 2.5h; 10 implementation + 50 AILANG test + 5 golden/interface metadata lines.
**Dependencies:** M1.
**Files:** `std/yaml.ail`, new `tests/yaml_encode_roundtrip.ail`, `internal/pipeline/testdata/builtin_types.golden`; stdlib interface baseline only if the existing freeze tooling requires it.
**Example coverage:** M3's runnable example imports `std/yaml.encode`.

Run `ailang prompt` before editing any `.ail` file; use the built repo binary for verification. Add the thin export and update the module header. Use matched Result values and Json equality for sorted unique-key fixtures, byte pins for unsorted emission, and surfaced duplicate-decode errors. Keep NaN tests in Go unless the expression surface demonstrably constructs it; do not assume division by zero is a supported NaN fixture.

Regenerate the builtin golden with its existing command. Inspect the stdlib freeze gate for the deliberate new export and refresh only the affected interface baseline through existing tooling if required. Make new tests fail observably (assert/test runner or nonzero harness failure), rather than merely print a cross while returning success.

**Acceptance criteria:**

- [ ] std/yaml type-checks and exports encode with the approved Result signature; builtins listing shows the new pure builtin.
- [ ] The new AILANG integration suite checks exact round-trips for sorted objects and byte order for unsorted objects, with the sibling caveat stated in-test.
- [ ] Golden diff adds exactly the `_yaml_encode` type entry; any stdlib baseline diff is limited to the approved new export.
- [ ] Existing yaml_bridge_test and yaml_config example pass against the built binary; existing JSON encode behavior remains unchanged.

**Risk:** An additive export can trigger golden/interface gates. Regenerate baselines with existing tools and inspect every changed entry.

### M3: Reference docs, runnable example and final checks (~105 LOC)

**Effort:** 2.5h; 35 example + 70 docs/manifest/changelog lines.
**Dependencies:** M2.
**Files:** new `examples/runnable/yaml_roundtrip.ail`, `examples/manifest.json`, `docs/docs/reference/std-yaml.md`, `docs/docs/reference/stdlib.md`, `changelogs/v0.32-current.md` under Unreleased.

Create decode-config → edit field with Json constructors → encode → print example, without importing both modules' encode unaliased. Register its module/capability information in the existing example manifest. Document quoted keys as well as quoted strings, escaped multiline strings, preserved emission order, duplicate-key pass-through/re-decode rejection, non-finite Errs, and single-document output. Use byte-accurate quoted-key examples: the approved doc's illustrative unquoted-key output must not be copied as expected bytes.

**Acceptance criteria:**

- [ ] Runnable example checks and runs with IO capability, emits pinned block YAML, and is represented in the example manifest.
- [ ] Reference page, stdlib index and Unreleased changelog describe the signature, dialect, limitations and precise round-trip caveat.
- [ ] Focused tests, golden check, stdlib verification, example gate, Go tests, lint and WASM build checks pass; attributable failures are resolved or explicitly reported.
- [ ] No decode-side changes, new Go dependency, release bump, or compiled-Go codegen support is introduced.

**Risk:** Sibling decode fix may land during execution. Re-inspect decode before freezing tests/docs; retain the current caveat unless order preservation is verified, then strengthen unique-key tests to exact equality. Existing sibling-owned regression changes are reconciled without adding decode work here.

## Day-by-Day Schedule

| Day | Work | Hours |
|---|---|---|
| 1 | M1 registration, emitter, scalar/nested/round-trip unit tests | 5 |
| 2 | M2 export, AILANG tests, golden and interface review | 2.5 |
| 2 | M3 docs, example/manifest and final checks | 2.5 |

M1 → M2 → M3 is the critical path. The 2h contingency is included in milestone allocations. If it is exhausted by a contract failure, report evidence and route the design question rather than expanding semantics.

## Validation and Success Metrics

Executor uses existing targets and the built binary (`make build`), not the possibly stale installed ailang:

1. During M1, `go test ./internal/builtins -run 'YAML|JSON' -count=1`; capture `go test ./internal/builtins -run YAML -coverprofile=/tmp/yaml-encode-coverage.out` and inspect emitter coverage with `go tool cover -func`.
2. During M2, `UPDATE_GOLDEN=1 go test ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot`, followed by the same test without update. `./bin/ailang check std/yaml.ail`; check and run both YAML test fixtures with `--caps IO --entry main`, using assertion-aware verification for new tests.
3. During M3, check/run the new and existing YAML examples; `make verify-examples`, `make verify-stdlib`, `make test`, `make lint`, and `make check-wasm-build`. Use existing freeze tooling if the new export requires a baseline refresh. Run `make check-boundaries` if implementation crosses packages; intended changes remain within builtins and stdlib.

Success is three completed milestones, all byte/round-trip/error criteria met, at least 90% new emitter statement coverage, one verified new example plus existing regressions green, and precise reference documentation. Planning validates JSON/schema consistency only; implementation checks belong to execution.

## Dependencies, Non-Goals and Handoff

The sibling decode-order work is a soft dependency only; do not block encode on it or implement its mechanism. Arbitrary-value input, block literals, configurable dialects, multidocument output and GoCodegenSpec are deferred by the approved design. Keep errors explicit and choose a stable, descriptive non-finite message within the design's delegated choices.

Source report: `inbox_1791394438696_00b01c33`. PR #1620 is the merged source triage, not an open implementation issue to close; leave `github_issues` empty until a real feature issue is verified.

Progress artifact: `.ailang/state/sprints/sprint_M-STD-YAML-ENCODE.json`. All milestones start uncompleted. Coordinator plan approval/merge triggers sprint-executor; this planning task does not start implementation or self-approve the new plan. Resume from the JSON, read builtin-developer guidance and obtain the current AILANG prompt at execution time.

## Execution notes

- M1: Implemented in cohesive `yaml_encode.go` and `yaml_encode_test.go`; all emitter helpers have 100% statement coverage, and YAML/JSON acceptance tests pass at `-count=20`.
- YAML-sensitive Unicode controls need `\uXXXX` escapes beyond ordinary JSON escaping; real bridge regression tests pin the correction.
- Current changelog gate requires fragments in `changelogs/unreleased/`, so the release note uses that convention rather than adding content directly under Unreleased.
- Environment: installed temporary user-local jq/make; Go was present outside PATH. Session script ran, but its baseline test report was vacuous before make was installed; a real `make test` is running separately. Missing bc affects only its velocity display.

- M2: `std/yaml` and the integration suite type-check; all four assertion-based tests pass. Existing bridge/config programs pass. Golden adds exactly `_yaml_encode`, and only yaml.json/yaml.sha256 are deliberately refreshed to accept the approved new pure export.
- M3: Example/reference docs complete; 231 examples pass, 0 fail, 9 skip. All 47 stdlib interfaces verify. Formatting, file-size, changelog, boundary and WASM gates pass.
- Further boundary tests pin Unicode noncharacters and explicit block keys at the YAML simple-key length boundary; this preserves the round-trip contract for oversized keys.

## Final validation

- `go test ./internal/builtins -run 'YAML|JSON' -count=20`: pass; all four emitter helpers have 100% statement coverage.
- Complete final `go test ./internal/builtins ./internal/pipeline`: pass (15.163s / 49.378s).
- Built-binary AILANG integration tests: 4 pass, 0 fail. Stdlib/example type checks and existing YAML bridge/config regressions pass. New example stdout matches manifest byte-for-byte.
- Builtin golden: one additive signature; all 47 stdlib interfaces stable after deliberately accepting only `std/yaml.encode`.
- `make verify-examples`: 231 pass, 0 fail, 9 skip. Final manifest validation: zero module drift (an existing missing lambda_expressions entry is reported).
- Lint: pass, zero issues with `golangci-lint run --timeout 10m ./cmd/... ./internal/... ./serveapi/...`; initial `make lint` exceeded its 5m limit during cold analysis.
- Final build, js/wasm compile, formatting, architecture boundaries, file-size and changelog gates: pass.
- **Full `make test` is not green.** This environment lacks a C compiler (`CGO_ENABLED=0`), causing SQLite-backed packages to use the go-sqlite3 error stub. Missing `ps`/`python` affects process/memory/Python tests. CLI recursion, process supervision, and an unrelated daemon-handler test also fail. Stopped the run after those failures were confirmed, with proctree/repl still pending. No unrelated runtime or test changes were made. The full run compiled an earlier intermediate Unicode test snapshot; those emitter failures are resolved, verified by the final complete builtins/pipeline run. Independent evaluator must assess the remaining global gate failures in a fully provisioned environment.

Implementation corrections retained for evaluator review: YAML-sensitive Unicode is escaped; oversized quoted keys use explicit block-key syntax to preserve the round-trip contract. Decode, dependencies and release version remain unchanged.

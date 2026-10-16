# Sprint Retrospective: M-CROSS-RUN-CONTENT-LIBRARY

## Outcome

Implemented opt-in durable SharedMem/SharedIndex, configured-provider `std/ai.embed`,
content-hash caching in memory and SQLite, and a keyless two-process content-library example.
ANN, read-only capabilities, automatic decay and vector migration remain deferred as approved.
No release, tag, push or merge was performed. The execution checkout is v0.54.13;
new builtin metadata targets the next version, v0.54.14. `std/VERSION` remains in lockstep
with the existing release rather than creating a release during implementation.

## Execution

The twelve milestones were grouped into independent storage-safety, index-backend and
embedding/cache work, followed by runner integration, fixtures and operator documentation.
The assigned branch was `coordinator/task-7f6a2223`, rather than the planner's previous branch.
Shared files were coordinated directly; no changes were discarded and no merge conflicts occurred.
The approved estimate was 1990 LOC / 24 hours; actual diff metrics are recorded in sprint JSON.
Agent wall time is not comparable to the planner's human working-day estimate.

## Verification

Final checks passed for all six changed packages, the core rerun after the intentional
builtin snapshot update, focused strict-config regressions, lint (zero issues), all 48
stdlib interfaces, CLI reference, formatting, sizes, changelog, architecture and manifest
import metadata. Strict-version separate-process seed/query runs pass. Independent
evaluation passed 98/100. Results and baseline exceptions are recorded in sprint JSON
and `.ailang/state/evaluations/eval_M-CROSS-RUN-CONTENT-LIBRARY_round_1.json`.
The showcase has effectful seed/query shells, so pure-function contracts and inline tests do
not apply; Go integration tests run both entry points in separate OS processes, plus a
fresh-file negative control. `ailang test` reports no inline tests and `ailang verify`
reports no contracts; these are explicit skips, not claimed successes. AILANG prompt
version loaded: v0.16.7, confirmed against the rebuilt binary.

The aggregate example gate exposed five unchanged examples whose `main` is unexported:
`adt_list_fields`, `if_else_chain`, `poly_adt_result_type`, `record_in_match`, and
`record_width_subtyping`. Their files and entrypoint-export enforcement predate this sprint.
Manifest validation has zero module drift; its stale `lambda_expressions.ail` entry is an
existing warning. The three required semantic-cache reference fixtures were modernized
within M11 to current module paths, interpolation, exports and semantic-frame constructors.

## Friction and corrections

- Session startup falsely reported tests passing with Go absent from PATH; direct Go checks
  replaced that result. Its dependency-summary jq expression and optional bc reporting also
  failed. These workflow scripts were left outside feature scope.
- Initial tests were authored/launched before implementation, but missing Go on PATH and
  cold dependency compilation prevented observing a semantic red result before source edits.
  This is a test-first workflow deviation; no observed red run is claimed.
- Cold, duplicated compilation used the cloud executor's RAM-backed filesystem and caused
  resource failures. Consolidated jobs and serial final verification reduce memory pressure.
- Existing lifecycle tests require `ps` and normal orphan reaping. Procps was unpacked into
  `/tmp` without system changes, and a temporary Linux subreaper provided normal init behavior.
- Provider-qualified model names already exist in messaging; removed a redundant wrapper
  caught by a focused provider test.
- Independent review confirmed that unrelated runs must not read embedding credentials;
  setup now returns before embedding config without AI authority, retaining cache-cap errors.
- Preserved strict configuration for explicit cloud providers with omitted embedding models;
  derived models retain the approved defaults and diagnostic notice.
- Updated only the intentional `std/ai.embed` golden using existing stdlib-interface helpers.
  Persistent cache schema follows the approved `text_hash`, `dim`, `created_at` layout.

## Next time

Provision Go/procps and inspect resource limits before starting cold parallel builds. Keep
backend ownership independent, but consolidate expensive linking, whole-package tests and lint.
The five baseline runnable-example failures should be handled in their own approved maintenance task.

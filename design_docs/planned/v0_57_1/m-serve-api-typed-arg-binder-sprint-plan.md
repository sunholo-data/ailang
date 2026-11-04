# Sprint Plan: M-SERVE-API-TYPED-ARG-BINDER

**Design:** [Approved design](m-serve-api-typed-arg-binder.md)  
**Task:** task-d8420071  
**Target:** v0.57.1  
**Duration:** 2 working days (8.5 hours tasks + 2.5 hours integration/validation + 3 hours contingency = 14 hours)  
**Estimate:** 980 changed/added LOC, approximately 400 implementation + 550 tests + 30 example/docs.  
**Risk:** Medium — shared conversion must preserve transport resolution and injected values.

Replace divergent number conversion with one declared-type binder across MCP, UI resources, REST and A2A. Fix both MCP int arguments and REST whole-number float arguments, and reject invalid values before evaluation.

## Approval and scope

The handoff explicitly approves the preceding design. Its unchecked human-approval box is superseded by that approval. All High-Impact Decisions are rulings: binder in apiserver, eval.Value output through engine.Call, strict declared scalars, and heuristic conversion only for unknown types/open fields. No open human-decision gates remain. This artifact prepares execution; coordinator plan approval/merge routes it to sprint-executor. Do not self-approve, merge or begin implementation during planning.

Keep argument resolution, required/optional presence policy, secrets, uploads, headers and raw request construction in their existing surfaces. Preserve embed conversion APIs, auth and WebSocket calls. Query-string/multipart text coercion and stricter MCP schema policy remain deferred.

## Current status and estimate evidence

Read CLAUDE.md and audited all affected call sites. MCP mcp.go:456, A2A a2a.go:226 and UI mcp_ui_serve.go:103 use CallPreserveFloats; REST routes_dispatch.go uses Call. Schema extraction currently records coarse ParamTypes and declared-record ParamZeros, but no recursive ParamSchemas. Existing zero helpers and embed eval.Value identity path are reusable. No implementation is complete.

The seven-day velocity script sees only commit 07759a70 and no usable changed-file/LOC metrics in this checkout. Historical CHANGELOG matches are not dated throughput evidence; no measured LOC/day is claimed. Capacity assumption is 490 LOC/day for this two-day sprint, weighted toward table-driven tests, with 3 hours contingency. Use the approved design's 14-hour estimate rather than extrapolating shallow history.

Touched implementation files currently range from 48 to 605 lines (mcp.go is largest); none is near 800. Put binder/schema and transport tests in the named companion files below rather than growing large existing tests. Each milestone fits within approximately 60–90 minutes; contingency belongs to the overall sprint.

## Registry reuse audit

Executed `ailang pkg search argument` (no results) and `ailang pkg search json` (AILANG application packages). Inspected `sunholo/http_helpers@0.1.5` with pkg info and pkg docs: it builds outbound HTTP requests and parses responses, and exposes no Go AST-to-eval binding. Decision for every milestone: **none**. This is a host-runtime correction inside internal/apiserver, with existing AST/eval/embed utilities reused; an AILANG package cannot replace this boundary. No registry dependency or contribution is required.

## Milestones

### M1: Declared parameter schemas (~160 LOC)

**Duration:** 1.5 hours. **Dependencies:** none

**Files:** `internal/apiserver/param_schema.go`, `internal/apiserver/param_schema_test.go`, `internal/apiserver/export_info.go`, `internal/apiserver/routes.go`.

**Tasks and acceptance criteria:**

- [ ] Extract scalar, record, list/array and unknown schemas from AST; reuse recordAliases and maxZeroDepth; store runtime-only ParamSchemas with json:"-".
- [ ] Test nested and aliased records, list element types, recursion bounds, labelled types and unknown schemas; existing registration behavior remains intact.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** AST aliases can cycle; mirror the existing bounded traversal and test the bound.

### M2: Shared typed binder (~300 LOC)

**Duration:** 1.5 hours. **Dependencies:** M1

**Files:** `internal/apiserver/arg_bind.go`, `internal/apiserver/arg_bind_test.go`.

**Tasks and acceptance criteria:**

- [ ] bindCallArgs produces eval.Value arguments: integral numbers bind int; all numeric float inputs including 2 bind FloatValue; bool/string reject mismatches.
- [ ] Recurse into declared record fields and list/array elements; preserve eval.Value identity, nil/Unit and unknown/open-field FromGo behavior.
- [ ] Reject fractional, non-finite and out-of-range int inputs before conversion; cover json.Number and Go numeric inputs without overflow or silent truncation.
- [ ] Parameter errors follow declaration order; nested failures identify the parameter and field/index; do not add closed-record validation or string coercion.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** Numeric casts can overflow; check integrality and native int bounds before casting. Preserve compatibility for undeclared fields.

### M3: Typed omission zeros (~120 LOC)

**Duration:** 1.25 hours. **Dependencies:** M1, M2

**Files:** `internal/apiserver/handler.go`, `internal/apiserver/param_zero.go`, `internal/apiserver/param_zero_test.go`, `internal/apiserver/named_args_test.go`, `internal/apiserver/argresolve_test.go`.

**Tasks and acceptance criteria:**

- [ ] zeroValueForType and paramZero deliver typed zeros including IntValue(0), FloatValue(0), bool, string, record and list; retain nil for unsupported types.
- [ ] Update representation assertions and prove fresh record zeros remain independent across calls; existing optional-record padding and argument precedence tests pass.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** Changing zero representations can break tests or share mutable records; test value classes and independence.

### M4: MCP and UI integration (~90 LOC)

**Duration:** 1.25 hours. **Dependencies:** M2, M3

**Files:** `internal/apiserver/mcp.go`, `internal/apiserver/mcp_ui_serve.go`, `internal/apiserver/mcp_arg_bind_test.go`.

**Tasks and acceptance criteria:**

- [ ] Bind resolved MCP arguments after presence, secret, file and header handling; call engine.Call with typed values; MCP rejects invalid arguments with IsError.
- [ ] UI resource calls adopt the same binder and engine.Call; existing resource behavior and missing-parameter validation remain intact.
- [ ] MCP tools/call succeeds for int 21 and omitted optional int zero, including named and positional argument forms.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** Injected eval.Value arguments must pass through unchanged; use existing secret/file/header regressions.

### M5: REST and A2A integration (~110 LOC)

**Duration:** 1.5 hours. **Dependencies:** M2, M3

**Files:** `internal/apiserver/routes_dispatch.go`, `internal/apiserver/a2a.go`, `internal/apiserver/routes_arg_bind_test.go`, `internal/apiserver/a2a_arg_bind_test.go`.

**Tasks and acceptance criteria:**

- [ ] REST binds after resolution/multipart and returns HTTP 400 with invalid arguments prefix for binding failures; integral JSON 2 binds correctly to float.
- [ ] A2A data/text argument paths share the binder and engine.Call; fractional int is JSON-RPC -32602 and int success is tested.
- [ ] No CallPreserveFloats invocation remains in internal/apiserver; mcp_auth.go, routes_ws.go and internal/embed public conversion semantics remain unchanged.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** Custom REST routes must find the same export schema as standard routes; test both and retain resolution precedence.

### M6: Parity matrix and documentation (~200 LOC)

**Duration:** 1.5 hours. **Dependencies:** M4, M5

**Files:** `internal/apiserver/arg_bind_parity_test.go`, `examples/runnable/serve_api_typed_args.ail`, `docs/docs/guides/serve-api.md`, `changelogs/unreleased/2026-11-03-serve-api-typed-arg-binder.md`.

**Tasks and acceptance criteria:**

- [ ] MCP and REST both pass scalar int, integral float, int record field, integral float record field, omitted optional int, and fractional-int rejection cases.
- [ ] Test bool/string mismatch, nested lists/records, unknown/open fields, optional records, legacy positional args, query fallback, multipart, mcp_file, headers and raw routes using existing fixtures where possible.
- [ ] Add a runnable typed-argument example and document results and surface-specific errors; changelog fragment opens with ### .
- [ ] Final make test-changed, make check-boundaries, make check-file-sizes, make check-changelog and make verify-examples pass; run go vet ./internal/apiserver.
- [ ] Run focused tests for changed behavior plus `make test-changed`; format touched Go files.

**Risk and mitigation:** Large transport matrices can duplicate setup; share small test helpers and keep companion files below the size limit.

## Schedule and validation

Day 1: M1 → M2 → M3 → M4 (5.5 hours), then focused regression checks and integration buffer. Day 2: M5 → M6 (3 hours), final checks and remaining contingency. Sequential execution is sufficient; no delegated work is required.

For each milestone use targeted `go test ./internal/apiserver -run <relevant-tests>` and `make test-changed` (changed packages plus core tests). Final gates are listed in M6; do not require the full `make test` suite in the cloud executor. Embed conversion tests must still pass via `go test ./internal/embed` to guard the public identity/heuristic behavior. Run `ailang check` on the example and include it in `make verify-examples`. Coverage goal is every declared kind and rejection branch plus all 12 primary MCP/REST matrix cells, rather than an unsupported package-wide percentage. Documentation may use the existing serve-api guide location if its filename differs; inspect before editing.

## Example and AILANG syntax gate

AILANG prompt version loaded: v0.16.11 (`ailang prompt --version-active`); `ailang prompt` was loaded in planning. Repository std/VERSION is v0.57.0 and the installed CLI warns it may be stale: executor must use a freshly built repository binary for checks and repeat prompt loading before writing AILANG.

Planned showcase: `examples/runnable/serve_api_typed_args.ail`, exporting twice(int), half(float), a record/list numeric function and an optional-int function. Use the loaded syntax and existing MCP examples for annotations; do not infer annotation syntax from memory.

| Showcase requirement | Decision |
|---|---|
| contracts | include: twice requires a small bounded input (avoid overflow), ensures result == n * 2; verify with `ailang verify` when Z3 is available, record tool unavailability explicitly |
| effects | include: pure numeric exports use explicit `! {}` |
| inline tests | include: twice tests 21 → 42, half tests 2.0 → 1.0, and record/list examples use deterministic numeric outputs |

## Completion and handoff

All six milestones start pending. JSON carries matching IDs, dependencies, estimates and registry decisions. No GitHub issue is linked: docparse PR #229 is external context, not an AILANG issue to close. Coordinator consumes the two artifact paths and routes the reviewed plan to sprint-executor; implementation then runs sprint-evaluator against the approved design and these criteria. No source code is changed in this planning stage.

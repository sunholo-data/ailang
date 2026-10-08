# Sprint Plan: M-SERVEAPI-RESPONSE-HEADERS

## Summary

Make static hosting safe by default, then make route response headers usable on both dispatch paths with explicit failures. Refs #1597. Refs #1609. No new issue is required.

**Design:** `design_docs/planned/v0_53_0/m-serveapi-response-headers.md`
**Target:** v0.53.0 / v0.53.x; current `std/VERSION` is v0.52.5, so the target is not stale.
**Duration:** 6 working days, approximately 36 focused hours including 6 hours of integration/review buffer.
**Estimated total:** 1,120 LOC (440 implementation, 480 tests, 200 examples/docs).
**Risk:** Medium: public defaults and header naming change; maintainer rulings D1/D4 are already approved.
**State:** Planning complete; execution awaits sprint approval through the coordinator. Design approval does not imply implementation has started.

## Current Status and Estimate Basis

The checkout is clean at planning start, on `coordinator/task-78bcfd88`. The incoming design artifact is present; its original work branch was `coordinator/task-68b343ad`.

Source inspection confirms both response sites still contain their independent header loops, raw Json headers are unhandled, and raw Content-Type defaults are assigned after WriteHeader. The design audits static, raw, nowrap, registration, CORS, request headers, guide examples, and MCP discovery: the systemic-analysis gate is satisfied. None of the implementation milestones is complete.

The skill's seven-day velocity script found only one visible commit (8f15a892, an unrelated sprint-plan commit) and no usable LOC metrics. This checkout's limited history cannot establish an empirical LOC/day or completion rate. Use the design's six-day estimate with explicit buffer rather than treating planning activity as implementation velocity. Planned capacity is about 187 total LOC/day; this is a budget, not a measured rate. Re-estimate after M1 if its tests or CLI wiring exceed one day.

## Delivery and Issue Policy

PR A contains M1 only: static security defaults, flags, wire tests, CLI reference, operator docs, and its changelog entry. It must be independently mergeable and releasable before route work. PR B contains M2 and M3. M4 verifies the combined result without holding PR A for later work. Use Refs #1597 and Refs #1609 in planning/development references. Only implementation PRs use closing keywords, for the issue each actually fixes; do not close issues from the planning PR. No new GitHub issue, release tag, or implementation is part of this planning task.

## Registry Reuse Audit

Executed `ailang pkg search headers` and `ailang pkg search oauth`; inspected `pkg info` and `pkg docs` for `sunholo/http_helpers@0.1.5` and `sunholo/mcp_oauth@0.1.1`.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | http_helpers builds outbound AILANG requests, not Go static response middleware; reuse existing static_cache and isHTTPToken code in this repository. |
| M2 | none | Packages cannot repair eval.Value extraction or @route registration in the Go hosting boundary; reuse std/json and WS registration patterns. |
| M3 | none | mcp_oauth is the consumer authorization server, not the Go mcp-check client; extend the existing metadata/client implementation without a package dependency. |
| M4 | none | Integration verification and documentation checks are repository tooling, not a package capability. |

No package contribution or new dependency is necessary. The installed CLI warned it may be stale; executor must build a fresh binary for behavior checks.

## Milestones

### M1: Independently shippable static security headers (~360 LOC)

**Estimate:** 150 implementation + 150 tests + 60 docs = 360 LOC; day 1, 6 hours.
**Dependencies:** None.

Create `internal/apiserver/static_headers.go` and `static_headers_test.go`; update `server.go`, `cmd/ailang/serve_api.go`, `cmd/ailang/help.go` and `docs/docs/reference/cli.md` as required by the CLI source-of-truth workflow. Update `docs/docs/guides/serve-api.md`, `docs/docs/guides/mcp-connectors.md`, and `changelogs/v0.32-current.md` in this first PR. Load cli-doc-maintainer when changing flags/help.

Clone static_cache's startup/wrapper pattern. Default to nosniff, DENY, frame-ancestors 'none', and no-referrer; implement repeatable --static-header with case-insensitive last-wins names and --no-static-security-headers. Validate token names and nonempty values free of controls before serving. Security applies to every static status, while cache behavior stays 2xx/304-only. Demonstrate operator commands in the guide using an existing runnable service; no new AILANG feature is required in M1.

- [ ] Real server wire tests show all four defaults on static 200, redirect, 304, and 404 responses.
- [ ] Overrides, repeated case variants, opt-out, and opt-out plus explicit headers produce the specified wire headers.
- [ ] Invalid names/values and either flag without --static fail startup with actionable errors.
- [ ] API/MCP/A2A and frontend proxy responses do not acquire static defaults; static cache regression tests pass.
- [ ] CLI help, operator examples, migration guidance and changelog ship in PR A; fresh-binary curl evidence is recorded.
- [ ] PR A is independently reviewable and releasable before M2/M3.

Risk: an app legitimately framing static HTML needs both X-Frame-Options and CSP overrides (SAMEORIGIN alone leaves frame-ancestors 'none' blocking frames). Document both overrides or the wholesale opt-out.

### M2: Unified route response headers and registration validation (~540 LOC)

**Estimate:** 220 implementation + 260 tests + 60 examples/docs = 540 LOC; days 2–4, 18 hours.
**Dependencies:** None technically; schedule after M1 delivery.

Create `internal/apiserver/route_headers.go` and `route_headers_test.go`; update `routes_dispatch.go`, `routes.go`, `export_info.go`, and affected tests. Replace the empty raw-response stub and copied nowrap loop with tests exercising production handlers. Create `examples/runnable/serve_api_response_headers.ail` with record and Json forms on both response paths; retrieve `ailang prompt` before writing it.

Extract from eval.Value after existing Result.Ok unwrapping and before ToGo. Records remap underscores to hyphens; JObject names remain exact. Validate the entire header set before mutating the writer so a later invalid field cannot leak partial program headers into an error response. Choose D7's explicit refusal option: reject X-Elapsed-Ms, Access-Control-* and Vary case-insensitively after normalization, retaining operator CORS and timing ownership. Set raw body Content-Type defaults before WriteHeader, then permit a valid program Content-Type override. Add declared-return-type diagnostics following WSReqIssue; avoid rejecting legitimate Json aliases or Result-wrapped returns supported by the existing dispatch.

- [ ] Both forms × both paths send X-Frame-Options and CSP through the real server; Result.Ok coverage exercises the existing unwrap behavior.
- [ ] Json preserves exact underscore names; record names remap; _headers metadata is absent from serialized response bodies and original eval records are not mutated.
- [ ] Invalid shapes, non-string fields/JObject values, invalid names and control characters produce structured 500 responses plus ERROR logs naming accepted forms and offending fields.
- [ ] Declared incompatible @route _headers types are refused at registration; valid record/Json declarations remain accepted.
- [ ] Reserved-header attempts fail loudly on both paths and cannot overwrite CORS, Vary or timing; valid program Content-Type wins over defaults.
- [ ] Raw string, bytes and JSON body defaults appear on the wire before status commitment.
- [ ] New runnable example checks and serves all four combinations; request-side mcp_tools and serve_api_mcp_header_auth fixtures and their auth suite remain unchanged and pass.
- [ ] Temporarily reverting the remap or raw Json acceptance makes the corresponding production wire test fail; restore changes afterward.

Risk: Result/alias handling and row-polymorphic declared shapes require following existing AST/export conventions rather than inventing compiler semantics. Keep implementation in the hosting boundary.

### M3: MCP framing probe and completed route documentation (~220 LOC)

**Estimate:** 70 implementation + 70 tests + 80 docs = 220 LOC; day 5, 6 hours.
**Dependencies:** M2 for checked route docs; probe is technically independent.

Update `internal/mcpcheck/mcpcheck.go` and its tests (or a small framing-specific file if size checks require it). Extend discovered metadata with authorization_endpoint. Use existing bounded/context-aware HTTP client behavior, send response_type=code, dummy client_id/redirect_uri/state and valid S256 PKCE parameters, and follow redirects. Judge only final 2xx HTML. Choose WARN for openai; anthropic/both missing protections FAIL. Reuse M2's example in `serve-api.md`, replace the two broken examples, and finish connector/changelog guidance without delaying M1 docs.

- [ ] Mock authorization server asserts every dummy request parameter and redirect-following behavior.
- [ ] Final 2xx HTML with X-Frame-Options or CSP frame-ancestors passes; without either FAILs for anthropic/both and WARNs for openai.
- [ ] Non-2xx, non-HTML, unreachable endpoints and redirect/client failures WARN with useful outcome details; no OAuth metadata skips the probe.
- [ ] Existing target-less behavior and metadata checks remain unchanged; tests pin target composition and error cases.
- [ ] Both corrected guide examples are extracted and checked with the fresh binary; connector guide distinguishes new defaults from the workaround for older releases.

Risk: dummy clients often receive a rejection rather than consent HTML. WARN is required in that case; absence of an assessable page is not a security PASS.

### M4: Combined verification and review buffer (~0 additional LOC)

**Estimate:** 0 planned production/test LOC; day 6, 6 hours reserved for review and fixes.
**Dependencies:** M1, M2, M3.

This is a verification-only milestone, not an implementable feature. Run the full acceptance matrix against a fresh build, inspect each PR's scope, and record results. Any review edits consume the buffer; re-estimate if scope grows beyond it.

- [ ] make build, make test, make lint, make check-file-sizes, make check-cli-docs, make check-changelog and make check-boundaries pass, or concrete pre-existing failures are recorded separately.
- [ ] Fresh-binary curl evidence covers defaults/override/opt-out and all four route header cases; local mcp check evidence covers assessable protected and unprotected consent pages.
- [ ] All named examples and request-header regressions pass, and mutation evidence is recorded.
- [ ] sprint-evaluator assesses the final implementation against the approved design; unresolved acceptance failures block completion.

## Day-by-Day Execution

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 defaults, flags, startup validation, wire tests and operator docs | Independent security PR A ready |
| 2 | M2 shared extraction/validation and unit cases | Both accepted forms parsed before Go conversion |
| 3 | M2 dispatch integration, header ownership, Content-Type ordering | Real four-cell wire matrix passes |
| 4 | M2 registration diagnostics, runnable example, regression/mutation checks | Route implementation ready for PR B |
| 5 | M3 probe fixtures, target outcomes and corrected docs | PR B complete |
| 6 | M4 full checks, fresh-binary evidence, evaluator and review fixes | Acceptance evidence complete |

## Success Metrics and Handoff

Cover every declared behavior/failure with production-path tests rather than setting an arbitrary repository coverage target. No compiler/parser/evaluator changes, new effects, packages, or syntax are planned. Documentation and fixtures are deliverables, not follow-up work. File-size concerns should lead to small hosting/checker modules, preserving architecture boundaries.

Sprint JSON: `.ailang/state/sprints/sprint_M-SERVEAPI-RESPONSE-HEADERS.json`. All passes/start/completion fields begin null. M1–M3 are the implementable features in JSON; M4 is recorded separately as the verification gate with zero additional planned LOC, avoiding the validator’s zero-LOC placeholder convention.

Design decisions are frozen. There are no remaining design approval questions. Coordinator review/approval of this plan is the execution gate; completion markers deliver the artifacts to that workflow. Do not send a duplicate executor message or start coding during planning. At execution, load sprint-executor and the relevant CLI skill; after execution load sprint-evaluator.

## Planning Artifact Validation

Python JSON parsing and structural assertions passed: three populated features, known dependencies, one registry decision per feature, existing artifact paths, issue links [1597, 1609], and summed LOC 1,120. `git diff --check` passed. The repository `validate_sprint_json.sh` could not perform its jq checks because jq is absent in this environment (its generic output says “Invalid JSON syntax”; independent parsing confirms the JSON is valid). Re-run that validator in the equipped executor environment. No implementation tests were run during planning.

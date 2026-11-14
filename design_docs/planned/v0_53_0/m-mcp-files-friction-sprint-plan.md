# Sprint Plan: M-MCP-FILES-FRICTION (Refs #1618)

**Design:** [m-mcp-files-friction.md](m-mcp-files-friction.md), including its 2026-11-03 HEAD re-verification amendment.
**Status:** revised plan ready for coordinator approval; implementation has not started.
**Target:** v0.57.1 (next patch after current v0.57.0; rebase release label if another patch ships first). The v0_53_0 directory is retained for stable artifact links.
**Duration:** 4 working days; 8 executor runs of 60–90 minutes, plus 25% contingency and cross-slice validation (about 16 focused hours total).
**Estimate:** 1,060 LOC: approximately 400 implementation and 660 tests, fixtures and documentation. **Risk:** medium overall, high for IFC.

## Current state and velocity

Items 1 and 3 have shipped. The design amendment banks live v0.57.0 b23cadce reproductions: named-test brace case passes 2/2; prompt units are milliseconds; formatter, all three IFC shapes, both contract cases and CORS remain open. M1 verifies this baseline against the executor's rebuilt HEAD rather than patching already-fixed behavior.

Planning checkout is clean at entry, coordinator/task-61b40eec, shallow HEAD 37e5b48d. Installed binary is v0.57.0 b23cadc with a stale-source warning; use the design's dated evidence, not a claim that this binary equals the planning checkout. Go is unavailable here; jq and Z3 are present. No fresh-source Go tests or coverage measurement are claimed. The seven-day velocity script exposes only a shallow root/merge commit and historical changelog snippets; no trustworthy observed LOC/day exists. Capacity of 265 LOC/day is a planning estimate from the design's 4–5 day range, not measured throughput.

## Approved rulings and scope

D2/D3/D4 are RULED by Mark on 2026-10-08 (design “Design Freeze”; PR #1697): preserve source multiline layout, enforce plain-field/domain sinks and reject unenforceable annotations, keep concat honestly SKIPPED with interpolation guidance. These are rulings, not new approval gates. D1 shipped via lexer tokens. D5 selects repeatable `--cors-origin https://*.domain`, exactly one leading host label, with shared REST/WebSocket admission on both listeners. Per-route CORS, quasiquote parser work and SMT list-fold encoding remain deferred. No human-decision gates remain in this plan.

Do not auto-close #1618: this is partial resolution of the grab-bag; milestone commits and PR use Refs #1618. No new issues or prompt amendments are needed.

## Registry reuse audit

Executed `ailang pkg search cors`, `formatter`, `refinement`, and `smt` on 2026-11-03; each returned no packages, so there were no candidates requiring info/docs inspection. M1–M8 each choose **none**, recorded separately in JSON: these changes belong in existing compiler/toolchain/shared Go listener internals, outside application package capabilities.

## Milestones

### M1: Bank shipped fixes and establish current reproductions (~20 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** none.
**Files/examples:** internal/testing/source_strip_test.go (existing controls); design reproduction log; no new public demo.

Run named-test reproducer and prompt unit controls; capture affected-package coverage baseline. Source builds require Go; planning environment has jq/Z3 but no Go.

- [ ] Bank the design amendment evidence: stripper shipped v0.54.13 b6c9f592e (#1821), prompt units shipped #1665; do not reimplement either.
- [ ] Rebuild HEAD and rerun the six design reproductions; record binary/source commits, expected outcomes and any drift before changing a slice.
- [ ] Existing named-test regressions pass; rebuilt prompt states milliseconds.

### M2: Preserve multiline literal source spans (~120 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M1.
**Files/examples:** internal/format/envelope.go; internal/format/literal.go; new internal/format/literal_layout.go and literal_layout_test.go.

Reuse existing envelope anchors and re-lex validation; refuse mismatched payload/attachment using the existing formatter refusal path.

- [ ] Original raw-newline literals retain their interior bytes only when re-lexing proves AST payload equality.

### M3: Complete formatter integration and corpus (~100 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M2.
**Files/examples:** internal/format/expr.go; internal/format/testdata/mcp_multiline.ail (new); roundtrip_soundness_test.go; implemented formatter design rule 6; existing formatter user guide.

Exercise nested expression positions, escaped and raw newlines, interpolation, comment adjacency, runtime payload equality and two-pass idempotence.

- [ ] Formatting is idempotent, runtime payload unchanged, and escaped one-line literals remain one-line.
- [ ] Corpus covers expression positions, escapes and interpolation; unsafe attachment fails closed using existing formatter behavior.
- [ ] Amend rule 6 in the implemented formatter design and update formatter user documentation.

### M4: Represent field sink refinements and reject unsupported positions (~150 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M1.
**Files/examples:** internal/types/ifc_static_type.go; new internal/types/ifc_field_refinement.go and ifc_field_refinement_test.go; existing IFC fixture family.

Inventory nested records, aliases, updates, ADT/list/tuple and higher-order paths; preserve refinements in supported metadata and fail loudly at unenforceable positions. Reuse SinkRefinementError.

- [ ] Unenforceable function-field codomain refinements receive an explicit diagnostic rather than being ignored.
- [ ] Nested records, updates, aliases, ADT/list/tuple positions and higher-order paths are audited; enforce supported paths or explicitly reject unsupported annotations.

### M5: Enforce field writes and function-field domains (~190 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M4.
**Files/examples:** internal/types/ifc_check.go; new internal/types/ifc_field_sink.go and ifc_field_sink_test.go; fixtures 18–20 in existing IFC family; docs/LIMITATIONS.md.

Check matching forbidden labels on plain-field construction/update and function-field calls; safe values remain accepted. Keep ifc_check.go (742 lines) below 800 by putting helpers in companion files.

- [ ] Secret plain-field writes and function-field domain calls are rejected for matching not-label refinements; safe values pass.
- [ ] Existing source-label fixtures p1-p13/r1-r4 and inbox_injection_v2 retain their expected results.

### M6: Normalize contract interpolation and diagnose list concat (~160 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M1.
**Files/examples:** internal/pipeline/show_normalize.go; internal/smt/show_rejection.go; existing SMT encodability diagnostics; new companion tests; examples/runnable/contracts/mcp_friction_interpolation.ail; docs/LIMITATIONS.md.

Normalize requires and ensures through the same typed pass as bodies. Verify string/bool examples with Z3; add direct-clause and body concat controls for the current hard-error regression.

- [ ] Existing normalization runs over Contract.Expr for requires and ensures, preserving type information, unique node IDs and idempotence.
- [ ] String and bool interpolation contract examples verify with Z3; body-side controls still verify. If z3 is absent, record this criterion as skipped (z3 absent) in notes; it does not block M4.
- [ ] Unsupported hole types retain honest residue diagnostics identifying contract interpolation.
- [ ] Contract std/string.concat becomes explicitly SKIPPED with interpolation guidance, including direct clause calls with ANF temporary bindings; it must not remain a generic encode error. No list-join SMT encoding.

### M7: Implement shared validated origin matcher (~160 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M1.
**Files/examples:** internal/platform/originpolicy/originpolicy.go; new origin_pattern.go and origin_pattern_test.go in that package.

Choose existing repeatable --cors-origin https://*.domain syntax. Match exactly one nonempty host label; pin scheme and effective port; reject unsupported wildcard placements. Use a parsed matcher for both exact and pattern grants.

- [ ] Recognize scheme-pinned --cors-origin https://*.claudemcpcontent.com using exactly one leading wildcard host label; apex, multiple labels, wrong scheme and wrong port do not match.
- [ ] Reject malformed wildcard, credentials, paths, query/fragment and null origin configuration at startup; no wildcard fallback to allow-all.

### M8: Integrate origin policy across listeners and document grants (~160 LOC)

**Run budget:** 60–90 minutes. **Dependencies:** M7.
**Files/examples:** internal/apiserver/cors.go and routes_ws.go; new cors_suffix_test.go and testdata/mcp_upload.ail; cmd/ailang/serve_api.go and existing server CLI origin configuration/help; docs CORS guide; examples/serveapi_ws_bridge.ail (existing control).

Replay OPTIONS/POST and WebSocket allow/deny matrices on both listeners. server.go is 799 lines: zero growth; use companion files. Echo concrete origins and preserve Vary behavior. Document global scope and unknown ChatGPT origin as unresolved deployment information, without broadening grants.

- [ ] Widget OPTIONS returns 204 with concrete Origin echo and POST reaches handler; hostile lookalike suffix POST returns 403.
- [ ] REST and WebSocket checks on both listeners use the same matcher; exact/same/missing-origin behavior and allow-all WebSocket semantics retain existing controls.
- [ ] CLI help and CORS docs explain suffix grants apply globally; per-route scoping remains deferred.
- [ ] All touched packages pass make test-changed and targeted size/changelog/boundary/example gates; sprint-evaluator records design acceptance evidence.

## Day-by-day schedule

- Day 1: M1 baseline and M2 source spans; record formatter failing/passing controls.
- Day 2: M3 corpus and M4 sink metadata/rejection inventory.
- Day 3: M5 enforcement and M6 typed contract normalization/diagnostics.
- Day 4: M7 matcher and M8 listener integration, focused gates and evaluator evidence.

Each milestone fits one executor run; preserve remaining work in JSON for resumption. Dependencies constrain ordering; no parallel agents are required. If IFC needs new semantics beyond the approved enforce-or-reject boundary, retain explicit rejection and return expansion to a separate design.

## AILANG syntax and fixture checklist

AILANG prompt version loaded: v0.16.11 (`ailang prompt` loaded during planning; `ailang prompt --version-active` recorded). Reload before executor writes `.ail` files. Do not write syntax from memory.

| Showcase or fixture | Contracts | Effects | Inline tests |
|---|---|---|---|
| `internal/format/testdata/mcp_multiline.ail` | skip: formatting corpus, payload equality tested by harness | skip: pure string-returning fixture | skip: formatter roundtrip/payload harness owns assertions |
| IFC field fixtures 18–20 | skip: static information-flow fixtures, including intended failures | include: secret source fixtures use `! {IO, Secret}`; safe pure controls omit effects | skip: negative type-check fixtures cannot run; Go harness checks diagnostics |
| `examples/runnable/contracts/mcp_friction_interpolation.ail` | include: pure tagged string/bool functions with `ensures` for result interpolation; a `requires` interpolation control | skip: pure functions | include: `tests [(in, out)]` on tagged functions for runtime expected output |
| `internal/apiserver/testdata/mcp_upload.ail` | skip: HTTP admission is tested before handler invocation | include: explicit `! {IO}` for fixture entry/logging if used; pure handler otherwise | skip: OPTIONS/POST integration harness checks responses |
| `examples/serveapi_ws_bridge.ail` (existing) | skip: existing behavior control only | skip: retain existing declared effect rows | skip: existing integration controls, no example edits required |

## Validation, risks and completion

For every slice, rerun its reproducer before edits; bank already-fixed results and narrow the work. Use existing Go package tests and fixture harnesses, then `make test-changed` (touched packages plus test-core). Run targeted formatter roundtrip/corpus tests, IFC positive/negative fixture families, SMT normalization/residue tests, and originpolicy plus listener matrices. Do not require the full `make test`; CI owns that suite. Measure affected-package coverage before/after and require no regression; do not invent a repository percentage.

Run `make check-file-sizes`, `make check-changelog` and `make check-boundaries`; apply gofmt and relevant lint checks. `make verify-examples` covers changed runnable examples; `make verify-stdlib` is required only if a stdlib file changes. Contract demos must type-check, run inline tests and verify with Z3. If Z3 is absent on the executor, record verification as skipped with that explicit reason, without claiming solver verification; normalization/type-info and residue tests still run. M6's solver-absent treatment follows the existing approved plan.

Add a fragment per remaining slice under `changelogs/unreleased/YYYY-MM-DD-<slug>.md`, opening with `### `. Update formatter rule 6 and user docs, IFC limitations, verification guidance and CLI CORS help. Keep `internal/types/ifc_check.go` (742 lines) and `internal/apiserver/server.go` (799 lines) within the 800-line limit using the named companion files. Other measured files are below 600 lines.

Formatter risk is incorrect span attachment: require re-lex payload equality, roundtrip and idempotence. IFC risk is silent unsupported positions: audit all type constructors and reject unsupported refinements explicitly. Contract risk is proving a different tree: use the existing typed normalization and preserve IDs/type info. CORS risk is grant widening: negative tests cover apex, deeper descendants, lookalike suffixes, schemes, ports and malformed origins; preserve exact/same/missing-origin and allow-all WebSocket controls.

Completion requires recorded outcomes for all criteria, paired positive/negative tests, no placeholders, and sprint-evaluator assessment against the approved design. No runtime implementation is part of this planning task.

## Coordinator handoff

Progress: `.ailang/state/sprints/sprint_M-MCP-FILES-FRICTION.json`; all eight revised milestones remain unpassed. Validate with `.agents/skills/sprint-executor/scripts/validate_sprint_json.sh M-MCP-FILES-FRICTION` before execution and after JSON updates. The existing JSON is revised directly rather than regenerated: the scaffold would overwrite prior state and import unrelated inbox messages.

Merging the coordinator plan PR supplies sprint approval and automatically hands off to sprint-executor (sprint-planner `resources/coordinator.md`). Do not self-merge or send a duplicate manual execution request. Executor commits locally; coordinator handles the PR. No open design decision requires a STOP gate.

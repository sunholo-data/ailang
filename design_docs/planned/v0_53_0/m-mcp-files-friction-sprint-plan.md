# Sprint Plan: M-MCP-FILES-FRICTION (Refs #1618)

**Design:** `m-mcp-files-friction.md` · **Target:** v0.53.0 · **Status:** planned, awaiting sprint approval through coordinator merge.
**Duration:** 5 working days (about 30 focused hours including 25% uncertainty buffer).
**Risk:** medium overall; high for IFC coverage. **Estimated total:** 1,180 LOC (440 implementation + 740 tests/fixtures/docs).

## Scope and approved decisions

The handoff states the previous design work is approved. This plan uses its recommended D2 (preserve multiline layout), D3 (enforce supported sink positions, reject unsupported ones), and D4 (keep list concat SKIPPED and document interpolation). These choices are explicit for sprint review, not fresh implementation approval. D1 reuses lexer scanning. D5 selects the existing repeatable `--cors-origin https://*.domain` syntax, keeping schemes and ports explicit and avoiding a second flag. Interpret the design's leading wildcard as exactly one host label; do not broaden to arbitrary descendant hosts. Per-route CORS, quasiquote parsing and list-join SMT encoding stay deferred.

Item 3 shipped in merged design commit f610b897 (#1665); do not repeat the prompt edits. Verify propagation after rebuilding. This remains a partial resolution of the grab-bag: PR body and milestone commits must contain **Refs #1618**, with no new issue and no automatic closing keyword.

## Current status and evidence

Read GitHub issue #1618 and its comments endpoint on 2026-10-08: no comments. Clean starting worktree, branch coordinator/task-2adc9287. Design includes all six live reproductions, source mechanisms and conflict-surface audit. Item 1 differs from the earlier triage because unbalanced braces are required. Planning inspected current originpolicy and normalization implementations; policy still compares exact origins and normalization needs contract traversal.

Available binary is v0.52.5 commit 7200786 and warns it may be stale. Go, gh and jq are absent. No fresh-source build, coverage baseline or source test execution is claimed. Executor must provision the normal Go toolchain and Z3, rebuild, and re-run each design reproducer before changing its slice. If a defect is already fixed, bank the reproduction result and adjust that slice rather than patching blindly. Run `ailang prompt` before writing any .ail fixture.

## Velocity and capacity

The existing analyze_velocity.sh was run for seven days. This shallow checkout exposes only the merged design commit; script reports no usable LOC metrics. Historical changelog snippets are not a measured current velocity. Estimates therefore use the approved design's 4–5 day range, expanded test/security coverage and 25% buffer. Planning capacity is 236 LOC/day, an estimate rather than observed productivity. Review IFC after its first two hours; if unsupported positions demand new type semantics, keep them explicitly rejected and return expanded enforcement to design review.

## Registry reuse audit

Executed `ailang pkg search cors`, `formatting`, and `contracts`. First two returned no packages. Inspected `sunholo/deontic@0.3.0` and `world/core@0.1.1` using pkg info/docs: these are application packages, not Go compiler or listener implementations. Every milestone chooses **none** and extends existing repository machinery; per-milestone decisions are persisted in sprint JSON.

## Milestones

### M1: String-aware test and property extraction (~140 LOC)

**Estimate:** 40 implementation + 100 tests/fixtures/docs. **Dependencies:** none.
**Files:** internal/testing/source_strip.go; internal/lexer/comment_scan.go; internal/testing/source_strip_test.go.
**Example/fixture:** examples/runnable/mcp_friction_braces.ail.

Implement against the corresponding design slice; first bank its failing reproducer, then add controls covering the surrounding mechanism.

- [ ] Unbalanced opening and closing braces inside literals leave both named tests passing (2/2).
- [ ] Balanced literals, escaped quotes, multiline literals, interpolation, comments and nested record braces preserve test/property boundaries.
- [ ] Reuse lexer scanning machinery without adding parser/AST spans; existing named-test regressions pass.

### M2: Preserve source multiline string layout (~220 LOC)

**Estimate:** 80 implementation + 140 tests/fixtures/docs. **Dependencies:** none.
**Files:** internal/format/literal.go; internal/format/envelope.go; internal/format/expr.go; internal/format/roundtrip_soundness_test.go.
**Example/fixture:** internal/format/testdata/ (new multiline corpus fixture).

Implement against the corresponding design slice; first bank its failing reproducer, then add controls covering the surrounding mechanism.

- [ ] Original raw-newline literals retain their interior bytes only when re-lexing proves AST payload equality.
- [ ] Formatting is idempotent, runtime payload unchanged, and escaped one-line literals remain one-line.
- [ ] Corpus covers expression positions, escapes and interpolation; unsafe attachment fails closed using existing formatter behavior.
- [ ] Amend rule 6 in the implemented formatter design and update formatter user documentation.

### M3: Enforce record field sink refinements (~340 LOC)

**Estimate:** 140 implementation + 200 tests/fixtures/docs. **Dependencies:** none.
**Files:** internal/types/ifc_static_type.go; internal/types/ifc_check.go; existing IFC test/fixture family.
**Example/fixture:** examples/runnable/contracts/inbox_injection_v2.ail (existing regression) and new IFC field fixtures.

Implement against the corresponding design slice; first bank its failing reproducer, then add controls covering the surrounding mechanism.

- [ ] Secret plain-field writes and function-field domain calls are rejected for matching not-label refinements; safe values pass.
- [ ] Unenforceable function-field codomain refinements receive an explicit diagnostic rather than being ignored.
- [ ] Nested records, updates, aliases, ADT/list/tuple positions and higher-order paths are audited; enforce supported paths or explicitly reject unsupported annotations.
- [ ] Existing source-label fixtures p1-p13/r1-r4 and inbox_injection_v2 retain their expected results.

### M4: Normalize interpolation in contract clauses (~160 LOC)

**Estimate:** 60 implementation + 100 tests/fixtures/docs. **Dependencies:** none.
**Files:** internal/pipeline/show_normalize.go; internal/smt/show_rejection.go; pipeline/SMT tests; docs/LIMITATIONS.md.
**Example/fixture:** examples/runnable/contracts/mcp_friction_interpolation.ail.

Implement against the corresponding design slice; first bank its failing reproducer, then add controls covering the surrounding mechanism.

- [ ] Existing normalization runs over Contract.Expr for requires and ensures, preserving type information, unique node IDs and idempotence.
- [ ] String and bool interpolation contract examples verify with Z3; body-side controls still verify.
- [ ] Unsupported hole types retain honest residue diagnostics identifying contract interpolation.
- [ ] List concat remains honestly SKIPPED with documented interpolation guidance; no recursive SMT join encoding is added.

### M5: Shared suffix-origin admission (~320 LOC)

**Estimate:** 120 implementation + 200 tests/fixtures/docs. **Dependencies:** none.
**Files:** internal/platform/originpolicy/originpolicy.go and tests; internal/apiserver integration tests; CLI CORS help/docs.
**Example/fixture:** examples/serveapi_ws_bridge.ail (existing regression); new upload integration fixture.

Implement against the corresponding design slice; first bank its failing reproducer, then add controls covering the surrounding mechanism.

- [ ] Recognize scheme-pinned --cors-origin https://*.claudemcpcontent.com using exactly one leading wildcard host label; apex, multiple labels, wrong scheme and wrong port do not match.
- [ ] Reject malformed wildcard, credentials, paths, query/fragment and null origin configuration at startup; no wildcard fallback to allow-all.
- [ ] Widget OPTIONS returns 204 with concrete Origin echo and POST reaches handler; hostile lookalike suffix POST returns 403.
- [ ] REST and WebSocket checks on both listeners use the same matcher; exact/same/missing-origin behavior and allow-all WebSocket semantics retain existing controls.
- [ ] CLI help and CORS docs explain suffix grants apply globally; per-route scoping remains deferred.

## Day-by-day execution

- Day 1 (6h): build/source reproduction and prompt propagation check (1h), M1 extraction (3h), start M2 envelope work (2h).
- Day 2 (6h): complete M2 preservation and corpus validation (4h), M3 sink-position inventory and failing fixtures (2h).
- Day 3 (6h): M3 enforcement/rejection, nested-path controls and IFC regressions.
- Day 4 (6h): M4 normalization/type-info/residue controls (4h), M5 origin parsing and matching (2h).
- Day 5 (6h): finish M5 integration/CLI docs (3h), full checks and sprint evaluation evidence (3h).

Slices have no functional dependencies; sequential execution reduces shared review churn. M3 is the critical uncertainty. M2 risk is mismatched literal/token attachment; payload equality and existing roundtrip gates are mandatory. M4 risk is proving a different tree than runtime; use the shared typed pass, never verify-only rewriting. M5 risk is widening an origin grant; scheme, port, label boundary and malformed-input negative tests define the grant.

## Validation and completion

Run targeted Go package tests for each changed slice, formatter corpus/roundtrip gates and IFC fixtures. Execute .ail examples with their required capabilities and `ailang check`; contract examples additionally require Z3 verification. Use existing repository test commands rather than a new harness. On completion run `make test`, `make fmt`, `make lint`, `make check-boundaries` and `make check-prompt-freeze`. Compare source/embedded prompt registries and verify rebuilt `ailang prompt` teaches milliseconds. Record deployed MCP-host propagation as a separate deployment requirement; this sprint does not claim a redeploy.

Measure affected-package coverage before/after and require no regression with meaningful positive and negative controls; no repository-wide percentage baseline is invented. Update relevant formatter/CORS/IFC/verification documentation, docs/LIMITATIONS.md and changelog per slice. Every example or fixture listed above must have recorded expected results. Evaluate completed work with sprint-evaluator against the approved design and these criteria.

## Handoff

Machine state: `.ailang/state/sprints/sprint_M-MCP-FILES-FRICTION.json`. All five milestones start unpassed. Coordinator plan merge provides sprint approval and triggers sprint-executor; do not start implementation during planning or send a duplicate manual execution request before that approval. No open design question blocks plan production; approval review must confirm the explicit D2/D3/D4 assumptions above.

# Sprint Plan: Parser Error-Cascade Suppression Policy

**Design:** `design_docs/planned/m-parser-error-cascade-suppression.md`  
**Issue:** [#934](https://github.com/sunholo-data/ailang/issues/934)  
**Sprint ID:** `M-PARSER-ERROR-CASCADE-SUPPRESSION`

## Summary

Implement the approved parser diagnostic policy so one malformed token produces one prominent, machine-readable primary error instead of hundreds of cascade diagnostics, while preserving genuinely independent errors and accounting explicitly for every suppressed or budget-truncated error.

**Duration:** 3 engineering days (approximately 24 hours)  
**Estimated change:** 730 LOC (implementation, tests, and fixtures)  
**Dependencies:** None; reuse existing structured `ParserError` and import/module recovery patterns  
**Risk level:** Medium — reporting behavior changes across parser, CLI, and eval consumers, and recovery must not hide independent errors

## Current Status Analysis

### Existing foundations

- `internal/parser/parser_error.go` already carries codes, positions, near-token context, expected tokens, suggestions, and confidence, but appends directly to an unbounded `[]error`.
- Import and module placement paths already consume the malformed declaration to avoid local cascades and provide state-isolation precedents.
- `internal/parser/error_recovery_test.go` verifies multi-error recovery, but currently uses minimum counts and does not distinguish independent errors from cascades.
- `ailang check` already offers human, JSON, and agent output modes; `internal/eval_harness/metrics.go` already records `error_category` and `err_code`.
- No suppression policy, primary marker, suppressed-code accounting, or `PAR_ERROR_BUDGET` exists.

### Velocity and estimate basis

The seven-day history contains a single squashed v0.42.0 release commit, so repository-wide insertions are not a trustworthy daily-velocity measure. This plan therefore uses the approved design's three-day estimate, a file-by-file work breakdown, and a conservative 20% integration allowance. The target is about 243 changed LOC/day, including tests and fixtures.

### Scope boundary

This sprint changes parser diagnostics only. It does not alter accepted AILANG syntax, AST semantics, type-checker/module errors, or parser permissiveness. Because the feature is observable only on invalid source, reproducible invalid-input fixtures under `internal/parser/testdata/cascade/` replace a runnable `examples/` program; adding a deliberately broken example would violate the examples suite contract.

## Milestones

### M1: Pin cascade and honesty baselines (~150 LOC)

**Goal:** Turn issue #934 and the independent-error obligation into deterministic fixtures before policy code changes.

**Example files to create/update:**

- Create `internal/parser/testdata/cascade/issue934_342.ail`, `issue934_153.ail`, and `issue934_29.ail` from the archived frontier-run reproducers.
- Update `internal/parser/error_recovery_test.go` with exact independent-declaration assertions.
- Add `internal/parser/cascade_suppression_test.go` for raw/pre-policy counts and source-position ordering.

**Dependencies:** None

**Tasks:**

- Recover the three exact issue #934 sources from issue/eval artifacts; do not substitute synthetic fixtures without recording the gap.
- Record raw parser error totals and code multisets for each fixture.
- Replace ambiguous minimum-count coverage with explicit cases for two and three independent declaration errors.
- Add suggestion-integrity assertions so later suppression cannot create or rewrite fixes.

**Acceptance criteria:**

- [ ] All three measured 342/153/29 cascade sources are committed as deterministic fixtures with their baseline totals documented.
- [ ] Tests distinguish a same-declaration cascade from errors separated by successfully parsed declarations.
- [ ] Independent-error tests assert exact surviving codes and positions, not only a minimum count.
- [ ] Existing module/import placement tests remain green before implementation begins.

### M2: Implement suppression policy and accounting (~250 LOC)

**Goal:** Centralize primary selection, cascade grouping, hard-cap behavior, and lossless suppression metadata without changing individual error suggestions.

**Example files to create/update:**

- Update `internal/parser/parser.go` and `internal/parser/parser_error.go`.
- Add or extend `internal/parser/cascade_suppression_test.go`.

**Dependencies:** M1

**Tasks:**

- Route parser error collection through one policy boundary while retaining compatibility for non-`ParserError` values.
- Add suppression metadata: exactly one primary, suppressed count, deterministic code multiset, and declaration-boundary identity.
- Implement coded-diagnostic preference within the approved five-token lookahead window.
- Enforce the three-diagnostic budget and emit `PAR_ERROR_BUDGET` for fourth and later independent errors.
- Preserve a lossless invariant: emitted parser errors plus `suppressed_count` plus explicitly budget-truncated count equals the raw discovered total.

**Acceptance criteria:**

- [ ] Each issue #934 fixture yields at most three emitted diagnostics and exactly one primary.
- [ ] Primary selection is deterministic by source position, with the approved coded-over-generic exception covered by tests.
- [ ] `PAR_ERROR_BUDGET` announces every independent-error truncation; no parser error is silently dropped.
- [ ] Suppression never changes `Fix`, `Suggestions`, `Expected`, near-token, or source context on the selected primary.
- [ ] Suppressed-code maps serialize in stable order or are compared structurally so output tests are deterministic.

### M3: Add bounded recovery at proven-safe points (~190 LOC)

**Goal:** Stop corrupted declaration-local parsing while guaranteeing that later declarations retain independent diagnostics.

**Example files to create/update:**

- Update `internal/parser/parser_lambda.go`, `internal/parser/parser_expr.go`, and `internal/parser/parser_record.go` (or the actual current owners found during implementation).
- Update `internal/parser/import_placement_test.go`, `internal/parser/module_placement_test.go`, and add focused recovery cases to `internal/parser/cascade_suppression_test.go`.

**Dependencies:** M2

**Tasks:**

- Add `parseParams` synchronization using fuzz evidence to choose `RPAREN`-only versus `RPAREN`-or-declaration-start.
- Apply the same bounded, declaration-local recovery discipline to record and match-arm lists.
- Audit existing import/module recovery through the common accounting path without weakening their single-coded-diagnostic behavior.
- Add state-isolation tests in which malformed recovery is followed by a valid declaration and then another independent malformed declaration.

**Acceptance criteria:**

- [ ] Parameter, record-field, and match-arm corruption synchronizes without crossing a successfully parsed declaration boundary.
- [ ] Two independent malformed declarations still emit both errors after recovery.
- [ ] Existing `module_placement_test.go` and `import_placement_test.go` pass unchanged unless an assertion is strengthened with equivalent semantics.
- [ ] The selected `parseParams` sync set is documented with fuzz/test evidence.

### M4: Wire CLI JSON, human prominence, and eval validation (~140 LOC)

**Goal:** Expose the policy consistently to agents and metrics consumers and measure its effect before widening.

**Example files to create/update:**

- Update `cmd/ailang/check.go` and relevant check output tests (add `cmd/ailang/check_parser_suppression_test.go` if no focused owner exists).
- Update `internal/eval_harness/metrics.go` and `internal/eval_harness/metrics_test.go` only where suppressed-code accounting must be preserved.
- Record paired-run evidence in the implementation report/design-doc completion notes rather than adding generated benchmark data to source fixtures.

**Dependencies:** M3

**Tasks:**

- Emit the primary first with `PRIMARY:` in human/agent output and one terminal suppressed-count line.
- Add the optional `suppression` object to `check --json`; prove it is absent for clean and non-cascade parses.
- Preserve existing `error_category` and `err_code` meanings while carrying the suppressed code multiset for analysis.
- Add a fuzz/property invariant for raw-total accounting and minimum-position primary selection.
- Run focused parser/CLI/eval tests, `make test-core`, `make lint`, and `make check-boundaries`.
- Run the approved smoke-tier `ailang eval-paired` on/off comparison and record category re-baselining needs; do not gate correctness on an agent-quality win.

**Acceptance criteria:**

- [ ] Human and agent stderr show one prominent primary followed by at most two independent diagnostics and one suppression summary.
- [ ] `ailang check --json` emits the approved `suppression` shape for cascades and omits it for clean/non-cascade results.
- [ ] Eval rows retain compatible `error_category` and `err_code` fields; suppressed code counts are available without multiplying result rows.
- [ ] Fuzz/property checks prove `emitted + suppressed + budget-truncated = raw total` and primary-position invariants.
- [ ] Focused tests, `make test-core`, `make lint`, and `make check-boundaries` pass.
- [ ] Smoke-tier paired on/off results and PAR_* re-baselining notes are recorded in the implementation report.

## Day-by-Day Plan

### Day 1 — Reproduction and policy core

- Complete M1 with exact issue fixtures and independent-error baselines.
- Implement the M2 data model, centralized collection, primary selection, and budget accounting under unit tests.
- Checkpoint only after the lossless accounting invariant is green.

### Day 2 — Recovery and output integration

- Complete M2 edge cases and coded-diagnostic preference.
- Implement M3 bounded recovery one point at a time, running focused tests after each point.
- Begin M4 human and JSON formatting once recovery behavior is stable.

### Day 3 — System validation and paired evidence

- Complete M4 CLI/eval integration and property/fuzz checks.
- Run focused and repository gates.
- Run the smoke-tier paired comparison, document baseline implications, and prepare the sprint-evaluator handoff.

## Success Metrics

- Three issue #934 fixtures: each at most three emitted diagnostics and exactly one primary.
- Independent errors: 100% retained until the explicit budget, with `PAR_ERROR_BUDGET` thereafter.
- Accounting: 100% of raw parser errors represented as emitted, suppressed, or explicitly budget-truncated.
- Compatibility: clean parse JSON and existing eval row fields unchanged.
- Regression gates: focused suites, `make test-core`, `make lint`, and `make check-boundaries` green.
- Evidence: paired smoke-tier report captured; no minimum repair-rate improvement is required for correctness.

## Risks and Mitigations

- **Exact issue sources may be absent locally.** Search banked eval artifacts and issue attachments first; if unavailable, pause M1 and report the evidence gap rather than labeling synthetic inputs as the measured reproducers.
- **Direct `p.errors = append(...)` sites bypass a new helper.** Audit all append sites and apply finalization at `Errors()`/parse completion as a defense-in-depth boundary.
- **New recovery hides an independent error.** Synchronize only inside the current declaration and require state-isolation tests for each recovery point.
- **Human and JSON views drift.** Derive both from the same finalized diagnostic structure and test both against the same fixtures.
- **Eval run cost/environment blocks paired evidence.** Complete deterministic correctness gates, record the external blocker explicitly, and do not fabricate paired results.

## Open Decisions During Execution

- Retarget the design's now-past v0.39.0 release marker before implementation; the current repository version is v0.42.0, and this planning task does not assign a release unilaterally.
- Choose `parseParams` synchronization (`RPAREN` only versus `RPAREN` or declaration start) from fuzz and fixture evidence.
- Decide whether representative suppressed positions are exposed only in verbose mode; default remains counts and code multiset only.

## Handoff Gate

This plan is ready for human review. Per repository workflow, implementation begins only after the user explicitly says **execute sprint**, at which point `sprint-executor` owns TDD execution and `sprint-evaluator` reviews the completed sprint.

# Sprint Plan — M-QUORUM-SALVAGE

**Design doc:** [m-quorum-salvage-retry.md](m-quorum-salvage-retry.md)  
**Sprint ID:** `M-QUORUM-SALVAGE`  
**Tracking:** GitHub issue #941  
**Planned at:** `v0.42.0`, branch `coordinator/task-1745e1f2`, 2026-09-23  
**Duration:** 3 days (approximately 18–22 engineering hours)  
**Risk:** Medium-high — the retry bound is small, but malformed-string repair must be conservative and both reviewer transports have distinct accounting semantics.

## Summary

Recover quorum verdicts that fail only because JSON string contents contain unescaped quotes or
newlines, then classify unrecovered parse failures separately from valid-JSON contract violations.
The sprint preserves reject-by-default validation, permits at most one model re-ask, aggregates the
second call's tokens and cost, and updates operator guidance so raising a cap is prescribed only for
budget absences.

## Current status and planning basis

- The approved design is still unimplemented. `ParseReviewResult` currently combines JSON decoding
  and semantic validation into one error return, and both `runReviewerWith` and
  `RunAgenticReviewer` map every such error to `absent_reason: "invalid"`.
- The text path has derived token pricing and a pre-flight cap; the agentic path receives observed
  cost from its runner. The implementation must not pretend these are interchangeable.
- Existing unit seams are suitable: `stubCaller` covers Tier 1 and `stubAgenticRunner` covers Tier 2.
  Both need response sequences and call-count assertions for bounded retry tests.
- The seven-day velocity script found one documentation commit and no usable LOC statistic. The
  estimate below is therefore code-surface based, with roughly 25% contingency, rather than a
  claimed historical LOC/day rate.
- Systemic scope is covered: shared parsing/repair behavior, Tier-1 text reviews, Tier-2 agentic
  reviews, artifact compatibility, accounting, truncation behavior, and remedy documentation.

## Milestone map

| Milestone | Deliverable | Estimate | Dependencies |
|---|---|---:|---|
| M1 | Failure-stage classification and deterministic local repair | 250 LOC | None |
| M2 | Tier-1 bounded salvage, accounting, and artifact metadata | 300 LOC | M1 |
| M3 | Tier-2 parity, remedy guidance, and full regression gates | 250 LOC | M1, M2 |
| **Total** |  | **800 LOC** |  |

The estimate includes implementation, tests, and documentation. No `.ail` example is appropriate:
this is an internal Go quorum transport policy. The executable examples are issue-#941-shaped Go
fixtures in the package tests.

## Proposed milestones

### M1: Classify parse stages and add conservative mechanical repair (~250 LOC)

**Goal:** Expose whether failure occurred during JSON decoding or reject-by-default validation, and
provide a deterministic one-pass repair that never changes a successfully decoded response.

**Files:**

- Update `internal/mission/quorum/reviewer.go`.
- Update `internal/mission/quorum/reviewer_test.go` with issue-#941-shaped fixtures.

**Tasks (Day 1):**

1. Introduce an internal typed/staged error (or equivalent `errors.Is`/`errors.As` contract) so
   callers can distinguish `parse_failure` from `gate_violation` without matching error strings.
2. Add a pure repair helper that handles raw newlines and clearly internal quote characters inside
   JSON string values. Keep it conservative: preserve valid JSON byte-for-byte, reject ambiguous or
   structurally malformed input, and never synthesize schema fields or verdict values.
3. Feed repaired bytes back through the unchanged `ParseReviewResult` and
   `ValidateReviewResult` contract.
4. Add table tests for the literal shape from #941, raw newline recovery, already-valid JSON,
   escaped quotes/backslashes, ambiguous/truncated JSON refusal, invalid verdict, empty objection,
   and empty catch.

**Acceptance criteria:**

- [ ] The #941-shaped raw quote fixture is mechanically repaired and returns the original verdict and text content after normal parsing and validation.
- [ ] Already-valid JSON is not rewritten, and ambiguous/truncated JSON remains an error rather than receiving invented structure.
- [ ] Valid JSON with a bad verdict or empty required text is classified as `gate_violation` and is never offered to the repair path.
- [ ] `go test ./internal/mission/quorum -run 'Test(ParseReviewResult|RepairReviewJSON|ReviewFailureKind)' -count=1` passes with the new tests observed in output.

**Risks and controls:**

- Quote repair can guess wrong about string boundaries. The helper must use JSON structural context
  and delimiter look-ahead, refuse ambiguity, and rely on the normal validator after repair; it
  must not perform broad regex replacement.

### M2: Wire one-shot salvage into Tier 1 with cumulative accounting (~300 LOC)

**Goal:** Make text reviewers recover locally first and re-ask exactly once only for JSON parse
failures, while recording provenance and enforcing the combined cap.

**Files:**

- Update `internal/mission/quorum/run.go`.
- Update `internal/mission/quorum/reviewer.go` if shared salvage types/constants belong there.
- Update `internal/mission/quorum/reviewer_test.go`.

**Tasks (Day 2):**

1. Add additive `omitempty` artifact fields to `ReviewerOutcome`: `salvaged`,
   `salvage_method` (`repair` or `re-ask`), and `invalid_kind` (`parse_failure` or
   `gate_violation`).
2. On a non-truncation parse failure, try local repair. If it fails, invoke the same `JSONCaller`
   once with the original rubric/document plus the approved re-emission suffix; do not alter the
   schema, model, or quorum membership.
3. Sum both calls' input/output tokens and derived costs. Apply the post-flight cap to the combined
   cost; an over-cap recovered response becomes a named `budget` absence with no result.
4. Keep `finish_reason=length` on its existing explicit truncation path without salvage, because
   truncation is a separate, models.yml-remedied non-goal.
5. Extend the Tier-1 stub to return an ordered response sequence and assert exact call count,
   prompt preservation, suffix presence only on call two, and no retry for gate violations.
6. Add a JSON compatibility test showing an ordinary unsalvaged Tier-1 outcome does not emit the
   new optional keys.

**Acceptance criteria:**

- [ ] Mechanical recovery makes the reviewer present with `salvaged:true` and `salvage_method:"repair"` after one provider call.
- [ ] A repair miss followed by valid re-emission makes the reviewer present after exactly two calls, with cumulative tokens/cost and `salvage_method:"re-ask"`.
- [ ] A second parse failure stops after exactly two calls and records `invalid_kind:"parse_failure"`; a gate violation stops after one and records `invalid_kind:"gate_violation"`.
- [ ] Combined cost above the cap produces `absent_reason:"budget"`, retains cumulative audit counters, and exposes no verdict.
- [ ] `finish_reason=length` remains a loud invalid/truncation result and does not re-ask.
- [ ] `go test ./internal/mission/quorum -run 'TestRunReviewerWith_' -count=1` passes with the new tests observed in output.

**Risks and controls:**

- The current Tier-1 code computes cost only once. Keep one accumulator per invocation and test the
  two-call arithmetic exactly, including over-cap behavior after the second billed response.

### M3: Give Tier 2 policy parity and publish actionable remedy guidance (~250 LOC)

**Goal:** Apply the same bounded policy to agentic reviews using observed accounting, and make the
operator-facing remedy match the new classifications.

**Files:**

- Update `internal/mission/quorum/agentic_caller.go`.
- Update `internal/mission/quorum/agentic_caller_test.go`.
- Update `.agents/skills/design-doc-creator/SKILL.md` in its design-quorum guidance.
- Update package comments/help text only where they enumerate invalid/absence semantics (for
  example `internal/mission/quorum/run.go` or `cmd/ailang/design_quorum.go`).

**Tasks (Day 3):**

1. Reuse the M1 classification/repair helper in `RunAgenticReviewer`; on repair failure run the same
   bounded agentic runner once more with the original review prompt plus the approved suffix.
2. Accumulate observed `CostUSD` and token counts across both agentic runs before enforcing the
   post-flight cap. Preserve timeout/cancellation behavior independently for each bounded call.
3. Add Tier-2 call-count, repair, re-ask, repeated failure, gate-violation, and combined-over-cap
   tests. Verify a retry retains the contested premise and read-only verification rubric.
4. Update design-quorum remedy guidance: `budget` may rerun alone with a raised cap;
   `invalid/gate_violation` and exhausted `invalid/parse_failure` are durable/reportable absences and
   must not trigger a cap escalation.
5. Run focused package tests, formatting, lint, boundary checks, and the repository test suite.

**Acceptance criteria:**

- [ ] Tier 2 has the same repair/re-ask bound and metadata as Tier 1, while using cumulative observed cost rather than models.yml-derived cost.
- [ ] The second agentic prompt preserves the original document, contested premise, and read-only rubric; only the JSON re-emission instruction is additive.
- [ ] Tier-2 gate violations make one call, and repeated parse failures make exactly two calls.
- [ ] The design-doc-creator guidance prescribes a raised cap only for `budget`, and explicitly forbids that remedy for both invalid kinds after bounded salvage.
- [ ] `gofmt -w` leaves all changed Go files formatted; `go test ./internal/mission/quorum -count=1`, `make test`, `make lint`, and `make check-boundaries` pass.

**Risks and controls:**

- Reusing a stateful `agenticCaller` can overwrite rather than accumulate the second run's observed
  metrics. Capture each call's metrics before the next call and test the exact totals.
- Repository-wide gates may expose unrelated baseline failures. Record the base and changed-tree
  result separately; do not hide or repair unrelated failures in this sprint.

## Day-by-day execution order

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 staged errors, repair helper, adversarial fixtures | Pure helper and parser tests green |
| 2 | M2 Tier-1 integration, accounting, artifact compatibility | All Tier-1 branch/call-count tests green |
| 3 | M3 Tier-2 integration, guidance, full verification | Focused and repository gates green or baseline failure documented |

## Success metrics

- The measured #941 response shape yields a present, validated verdict without weakening any
  semantic gate.
- Every provider/runner path makes at most two calls (original plus one re-ask), proven by tests.
- Gate violations make exactly one call; truncation retains its dedicated remedy.
- Cost and token totals include every billed call and are checked against the existing cap.
- Existing artifact consumers remain compatible because every new field is additive and omitted at
  its zero value.
- The package retains strong branch coverage through named unit fixtures; no external provider,
  network, or `.ail` example is required.

## Dependencies and open questions

- No external dependency or migration is required. The work uses the existing `JSONCaller`,
  `AgenticRunner`, parser, validation, pricing, and artifact seams.
- Executor judgment is intentionally limited to the conservative repair algorithm. If the #941 raw
  response is too ambiguous to repair without guessing, the required fallback is the single re-ask,
  not broader coercion.
- The approved design is authoritative. Any need to change verdict/schema semantics or retry more
  than once must return to design review rather than being absorbed into implementation.

## Handoff gate

This plan and its JSON progress file are ready for human review. Per repository workflow,
implementation begins only after the user explicitly says **“execute sprint”**, at which point the
`sprint-executor` skill owns the work.

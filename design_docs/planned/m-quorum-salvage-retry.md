# M-QUORUM-SALVAGE — Salvage/Retry Policy for Parse-Failed Quorum Reviewer Responses

**Status:** planned
**Type:** quorum-policy change (internal/mission/quorum; no AILANG language, stdlib, or motoko-core surface)
**Source:** GitHub issue #941 (sunholo-data/ailang; mission-world iteration 133, measured first-party)
**Version context:** v0.38.5

## Problem

A Tier-1 text reviewer that quoted the literals under discussion (e.g. `Some ...` inside a
JSON string) produced JSON the parser rejected. `ParseReviewResult`
(internal/mission/quorum/reviewer.go) failed, and `runReviewerWith` (run.go) /
`RunAgenticReviewer` (agentic_caller.go) recorded the reviewer ABSENT with
`AbsentReason = ReasonInvalid` ("invalid").

The failure mode is real and recurring: reviewers reviewing string-literal-adjacent design
docs naturally embed the literal in their `strongest_objection`/`proposed_fix`, and a bare
quote inside a JSON string breaks `json.Unmarshal` on the whole object.

The prescribed remedy for absent reviewers — *re-run each absent reviewer alone with a
raised cap* — targets budget exhaustion. For `invalid` it raises the budget for a problem
that is not budget, and since the same doc re-elicits the same unescaped quote, it can
loop indefinitely (same model, same prompt, same malformed output, higher spend).

Two gaps:

1. **No repair attempt.** A one-shot salvage (a JSON-repair pass or a single re-ask with
   escaped-literal guidance) would likely recover these verdicts cheaply — but no such
   path exists.
2. **`invalid` conflates distinct causes.** `ReasonInvalid` covers gate violations
   (bad verdict enum, empty objection), quoting breakage, and truly malformed output. The
   remedy layer cannot distinguish them, so the only lever it has (raise the cap) is wrong
   for at least two of the three.

## Goals

- Recover parse-failed reviewer verdicts without weakening the reject-by-default contract.
- Make `invalid` absences actionable: either repaired in-band, or classified so the remedy
  layer knows the correct response.
- Bound the repair so it cannot spin or silently inflate budget.

## Non-Goals

- Changing the reviewer schema, system prompt semantics, or verdict rubric.
- Auto-coercing anything into a pass.
- Repairing truncation failures (`finish_reason=length`) — that is already surfaced with a
  models.yml remedy and is out of scope.

## Design

### 1. Bounded salvage attempt inside the invalid path

When `ParseReviewResult` returns a **JSON syntax error** (unmarshal failure — as opposed to
a `ValidateReviewResult` gate violation), attempt exactly ONE salvage:

- **Order:** (a) mechanical repair pass first — a deterministic, local fixer that escapes
  raw newlines/quotes inside string bodies and retries `json.Unmarshal` (a small, testable
  helper, not an LLM call); (b) if that fails, ONE re-ask to the same reviewer with a
  suffix prompt: *"Your previous response was not valid JSON because of unescaped quote
  characters inside a string. Re-emit the same verdict with all literal quotes inside
  string values escaped as \\\"."* The re-ask must NOT change the rubric or the doc body.
- **Gate:** the salvaged result still passes through `ParseReviewResult` →
  `ValidateReviewResult` unchanged. A repaired verdict with a missing/empty
  `strongest_objection` is still an error, never a coerced pass. Reject-by-default holds:
  salvage recovers *transport*, never *verdicts*.
- **Audit:** on success, the outcome records `present: true` plus a new additive field
  `salvaged: true` (and optionally `salvage_method: "repair" | "re-ask"`) so downstream
  consumers and the catch-rate hook can see the verdict was recovered after a parse
  failure. Additive + `omitempty` — existing artifacts stay byte-compatible.

### 2. Distinguishing repairable from genuinely malformed

Classification is by failure stage and repair outcome, not by model introspection:

| Condition | Classification |
|---|---|
| Unmarshal error, mechanical repair succeeds | repaired (in-band) |
| Unmarshal error, repair fails, re-ask succeeds | repaired (re-ask) |
| Unmarshal error, repair + re-ask both fail | `invalid` with `parse_failure` detail |
| `ValidateReviewResult` gate violation (bad verdict, empty objection/catch) | `invalid` with `gate_violation` detail — **never salvaged** (the model produced valid JSON saying something contract-breaking; re-asking would be retrying a policy failure, which is exactly the loop we are closing) |

A new optional field on `ReviewerOutcome` — `InvalidKind string` (`"parse_failure"` |
`"gate_violation"`, omitempty) — gives the remedy layer the distinction without breaking
the existing `absent_reason` enum consumers.

### 3. Remedy mapping

The skill/doc guidance for absent reviewers becomes:

- `budget` → re-run alone with a raised cap (existing behavior).
- `invalid` + `gate_violation` → do NOT re-run with a raised cap; the reviewer is policy-
  nonconforming — treat as a durable absence (N-1 degradation) and report it; repeated
  occurrences flag the model in models.yml.
- `invalid` + `parse_failure` → the salvage already ran once and failed; a raised cap
  cannot help. Durable absence + report. One salvage per reviewer per run, hard-coded —
  no second re-ask under any condition.

### 4. Budget and loop bounds

- The salvage re-ask runs under the SAME `maxCostUSD` cap as the original call; its cost
  is added to `out.CostUSD` and its token counts to the recorded `tokens_in/out`
  (post-flight cap check applies to the combined cost — over-cap degrades to `budget`,
  never blocks).
- Exactly one salvage attempt per reviewer per quorum run — a constant, not a knob.
- No escalation to other models on parse failure (that would change quorum composition
  based on transport noise).
- The mechanical repair pass is free (local string transformation, deterministic).

## Files Touched (planning estimate)

- `internal/mission/quorum/reviewer.go` — repair helper; `ParseReviewResult` unchanged in
  contract.
- `internal/mission/quorum/run.go` — salvage wiring in the `ReasonInvalid` path;
  `Salvaged`/`InvalidKind` fields on `ReviewerOutcome`.
- `internal/mission/quorum/agentic_caller.go` — same wiring for the Tier-2 agentic path.
- Tests: quoting-breakage fixture from issue #941 (unescaped `Some ...` in
  `strongest_objection`), gate-violation-not-salvaged, one-attempt bound, combined-cost
  cap check.

## Acceptance Criteria

1. The issue-#941 transcript shape (raw quote inside a JSON string) is recovered by the
   mechanical repair or single re-ask, and the recovered verdict passes
   `ValidateReviewResult` unmodified.
2. A gate-violating response is never re-asked and never coerced.
3. At most one salvage attempt per reviewer per run; total observed cost still respects
   the cap (degrading to `budget` when exceeded).
4. Existing quorum artifact JSON remains backward-compatible (new fields additive +
   omitempty).
5. The remedy guidance distinguishes the three absent-reason classes, ending the
   raise-the-cap-on-parse-failure loop.

## Related Documents

- Design-doc-creator skill (quorum flow and absent-reviewer remedy wording — updated by
  this doc's implementation).
- internal/mission/quorum package docs (reject-by-default contract, #708 token accounting).

# Forall property blocks fail to lower — #624 / M-M3-RESIDUAL T6

- **Date**: 2026-09-15
- **Class**: bug (already diagnosed upstream; report adds user pain + an interim-mitigation suggestion)
- **Recommend**: duplicate-of `design_docs/planned/v1_1_0/m-forall-properties-direct-core-eval.md`
- **Searched**: `forall` in design_docs/, `M-M3-RESIDUAL`, `EvaluateExpression` in design_docs/ and internal/
- **Estimate**: n/a (duplicate)

The root cause, the reproduce, and even the "runtime route is the known-broken
`EvaluateExpression` synthesis path" diagnosis are already captured verbatim in
`design_docs/planned/v1_1_0/m-forall-properties-direct-core-eval.md`
(M-FORALL-PROPERTIES, ~250 LOC, P3). That doc's decision clause is "move forward —
but only when a real user asks"; this report IS a real user asking (Daneel, with
two production workarounds via ensures-clause PBT in sunholo/discord and
sunholo/agui), so the actionable outcome is to promote that doc in priority, not
write a new one.

The report's secondary suggestion — compile-time rejection of `properties [...]`
with a pointer to #624 instead of a runtime PAR_UNEXPECTED_TOKEN failure — is a
change to a language gate's contract and belongs in the same design doc as a
small interim section, not a direct fix: whether to hard-reject valid-syntax-but-
broken-evaluation constructs vs. emit a friendlier runtime diagnostic is a
decision someone could disagree with (row 3/4 of the rubric), and A11 in the doc
already covers the "structured failure" intent.

No search found a doc covering the compile-time-rejection interim specifically
(terms searched: `properties` compile-time reject, `forall` in design_docs/) —
none found; fold it into the linked doc.

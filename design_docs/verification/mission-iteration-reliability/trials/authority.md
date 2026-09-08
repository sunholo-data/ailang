# Attended authority for the two reliability documentation trials

Locator: `mark-reliability-docs-smoke-approved-20260908`

Mark instructed “yep please continue” in response to approval and plan/execute
of M-MISSION-ITERATION-RELIABILITY. Its M4 includes freezing two useful bounded
documentation tasks before inference within a combined new trial cap of $5.
The exact brief/design/plan documents below instantiate that delegated scope.
This artifact records the attended instruction, not a model self-approval.

Both tasks independently start from reliability implementation `48e72ef7e`
plus this approval/source-locator correction commit; neither imports the other task.
Both permit only `docs/docs/guides/mission-iteration.md`, with no merge or publication.
Imported designer/planner provenance is `gpt6-astra` (OpenAI); author is
`claude-sonnet-5` (Anthropic), independent evaluator `pi-or-deepseek-v4-flash`
(DeepSeek, OpenRouter). Runtime vendor checks cover all these identities.

Each item is limited to 220,000 fresh input+output tokens, 2,400 seconds and $1.50
metered total. Author limits: 120,000 tokens, 1,200 seconds, $1 reported-cost guard.
Evaluator limits: 100,000 tokens, 1,200 seconds, $0.50 cost guard. The existing
candidate review reserves $2; these two reserve $1.50 each, combined $5. No automatic
retry, budget increase, schedule expansion, or author/evaluator replacement.

| Trial | Imported role | Committed artifact | SHA-256 |
| --- | --- | --- | --- |
| `budget-accounting` | `designer` | `design_docs/verification/mission-iteration-reliability/trials/budget-accounting/design.md` | `ac87cfb3f9a7174ee5d644633f2feac81e86b5253cfc2d28b884f711c1285ac0` |
| `budget-accounting` | `planner` | `design_docs/verification/mission-iteration-reliability/trials/budget-accounting/sprint-plan.md` | `cc01392b5c2ac68929c4419b38ae0b7fb63c43eedc0368ab432841f946bdaa7b` |
| `review-packet` | `designer` | `design_docs/verification/mission-iteration-reliability/trials/review-packet/design.md` | `04aa6beaa0bcd65a12c67030933f0019b3dd8b1ab63368c7a027725d275f187b` |
| `review-packet` | `planner` | `design_docs/verification/mission-iteration-reliability/trials/review-packet/sprint-plan.md` | `1d5810d7b1d26ad3a33be8ba505e88b87cb9fe7421a5de2d643d3a59385cefe0` |

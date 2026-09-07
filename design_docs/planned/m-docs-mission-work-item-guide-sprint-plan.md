# M-DOCS-MISSION-WORK-ITEM-GUIDE — canary sprint

Status: Approved scope and execution, attended Mark 2026-09-07: “yep approved - proceed and execute”.
Design: [work-item guide](m-docs-mission-work-item-guide.md).
Progress: `.ailang/state/sprints/sprint_M-DOCS-MISSION-WORK-ITEM-GUIDE.json`.

## Contract

One product file: `docs/docs/guides/mission-iteration.md`. The five acceptance IDs
and scope in the approved design are frozen without extension. This session supplies
the approved design and plan as imported prerequisites; runtime executes author then
independent evaluator. Both prerequisite authors: attended Codex, registry `gpt6-astra`.
Mark's approval delegates this bounded plan and its execution; no second approval sought.

The complete illustrative JSON must use syntactically valid full dummy revisions and
SHA-256 values, clearly labeled as illustrative in adjacent prose. Every other example
value needing replacement must be explained. This permits structural validation without
misrepresenting the example as approved input or claiming a successful Git authority check.

## M1 — write the guide (executor; estimate 150–220 Markdown lines)

Dependencies: frozen prerequisite artifacts and approval references; approved canary placement.
Maximum 1,800 seconds, 70,000 fresh tokens, $3 metered guard.

- [ ] Complete JSON matches the production decoder and covers all remaining/imported roles.
- [ ] Every illustrative value is explained, including actual committed authority preparation.
- [ ] Original lifecycle/exit/recovery sections preserved; only the allowed file changes.
- [ ] Commit the product, then emit the runtime's untracked stage-result.json.

## M2 — independent acceptance (evaluator; ~30 lines retained evidence, no product changes)

Dependencies: M1 accepted candidate. Maximum 1,200 seconds, 30,000 fresh tokens, $2 guard.

- [ ] Inspect exact candidate and source against all five frozen criteria; report each pass/fail.
- [ ] Verify commands, authority hashes, independence and documented runtime limitations.
- [ ] Preserve candidate HEAD/tree; emit strict evaluator result with concrete evidence.
- [ ] Runtime hard checks pass, acceptance persisted, completed repeat dispatches zero work.

## Frozen hard checks

1. `git diff --check BASE`, repository root, timeout 30 seconds.
2. `/private/tmp/ailang-docs-canary/validate-example docs/docs/guides/mission-iteration.md`,
   repository root, timeout 30 seconds. The helper's source is retained at
   `design_docs/verification/mission-iteration/canary/validate-example.go.txt`; compile
   against this sprint's unmodified `iteration.Decode`. It extracts exactly one JSON
   work-item block and runs the production strict decoder. No new product tool is added.

Check 2 proves structural/semantic schema validity only. Real authority is checked by
runtime for the actual canary WorkItem, and prose accuracy by the independent evaluator.
No full-site build is claimed. For this Markdown-only task the source JSON check plus
independent review is proportionate; existing runtime test evidence remains in checks.md.

## Routing, timeline and operational checks

Explicit executor `claude-sonnet-5` via Claude; evaluator `pi-or-deepseek-v4-flash`
via Pi/OpenRouter. No fallback candidates. Total limits 3,600 seconds, 100,000 fresh
tokens, $5 metered guard. Frozen registry uses existing entries/pricing; final evidence
must distinguish metered/subscription/unknown usage rather than infer an invoice.

Day 1: prepare committed approvals and isolated source/registry; dry-run; after legacy
Docs is idle, save placement/scheduler state and run; repeat completed input; restore.
The earlier runtime sprint's 0.68-hour report mixes parallel engineering and tests,
so it is not a meaningful documentation velocity forecast. This estimate is a bounded
one-file task allowance, not a measured delivery rate or a promise to wait at most an
hour for the already-running legacy mission to finish.

Activation and rollback follow the reviewed canary packet. Record actual work ID/base,
input hash, binary/helper hashes and before/after ownership. Do not restart legacy
Docs or replace the fleet binary. Hold ambiguous work for inspection, without redispatch.

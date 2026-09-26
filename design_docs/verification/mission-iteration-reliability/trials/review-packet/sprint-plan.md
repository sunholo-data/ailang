# Trial B execution plan

Planner provenance: attended session, registry model `gpt6-astra`.
Design: [Review evidence packet](design.md). Authority/base/route binding:
parent-supplied immutable work item under [series limits](../frozen-briefs.md).
Estimate: one 180–300-word section; 15 minutes author effort plus 15 minutes review,
with explicit 20-minute stage ceilings. No dependency on Trial A's output.

1. **Author:** Read the guide, `review_packet.go` and the evaluator request builder.
   Add only Review evidence packet, covering B1–B5 without rewriting existing guide
   sections. Commit the guide and emit the runtime's result protocol referencing
   that exact candidate. Do not modify plan/authority artifacts or runtime code.
2. **Independent evaluator:** Inspect the supplied packet once, the exact guide
   diff and relevant source functions. Settle B1–B5 individually, checking especially
   the distinction between receipt summaries and actual outputs, and incomplete
   evidence versus a passing review. Preserve HEAD/tracked files; emit the verdict.

Hard check `B-whitespace`: argv `git show --format= --check HEAD`, cwd `.`,
timeout 30 seconds, expected exit 0. Source-focused verification is the evaluator's
explicit B1–B4 claim-to-function assessment at frozen revisions. Do not substitute
keyword matching or prose-mirroring tests for that assessment. The runtime checks
allowed-path scope and evaluator candidate preservation independently.

Required artifact: `docs/docs/guides/mission-iteration.md`. No additional code/example
files or new tests. Acceptance requires all criteria, hard-check evidence and an
independent verdict. Missing source evidence remains a finding, never an inferred
pass. Preserve failures; no automatic rerun, model switch, merge or publication.

# Trial A execution plan

Planner provenance: attended session, registry model `gpt6-astra`.
Design: [Budget accounting](design.md). Authority/base/route binding: parent-supplied
immutable work item under [series limits](../frozen-briefs.md).
Estimate: one 180–300-word section; 15 minutes author effort plus 15 minutes review,
with explicit 20-minute stage ceilings. This is a scope estimate, not a throughput claim.

1. **Author:** Read the existing guide and the five focused implementation sources
   named in the design. Add only the Budget accounting section, settling A1–A5.
   Preserve surrounding examples. Commit the guide and emit the runtime result
   protocol referencing the exact candidate commit; do not edit plan/authority files.
2. **Independent evaluator:** Inspect the frozen packet once, then the exact guide
   diff and cited accounting functions. Record a separate finding for A1–A5. Run
   the named hard check, preserve HEAD and tracked files, and emit the evaluator
   result. A successful command alone is not a correctness verdict.

Hard check `A-whitespace`: argv `git show --format= --check HEAD`, cwd `.`,
timeout 30 seconds, expected exit 0. Source-focused verification is the evaluator's
explicit A1–A4 claim-to-function assessment at the frozen revisions; do not replace
it with a keyword-presence test or count lines as evidence of correctness.
Runtime independently enforces the allowed path and candidate integrity.

Required artifact: `docs/docs/guides/mission-iteration.md`. No additional code/example
files or prose-mirroring tests are needed. Acceptance requires all five criteria,
valid hard-check evidence and independent review. An uncertain accounting claim
must be rejected or reported for decision, not silently simplified into a false rule.
Failure preserves the candidate; no automatic author retry or route change.

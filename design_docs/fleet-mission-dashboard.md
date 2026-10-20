# Fleet mission — iteration 51, 2026-10-20

- Latest released binary: v0.54.16. dev: `3b4d11759` (#1912, this fire).
- LANDED and RESOLVED: `rotate-log:legacy-heading-format` (stapledon, 10-20):
  - `ailang mission rotate-log` parses date-first headings (`## 2026-09-28: iteration 1, …`, date-first notes) and `## STATUS <date> (iteration N) — …`. Stapledon's 41-entry log now rotates 21 out and keeps 20.
  - A rotation that archives nothing leaves the live log byte-identical. A real rotation writes exactly one "ARCHIVED" notice, and duplicates collapse (v1 had 8). An unparsed record-shaped preamble heading refuses loudly.
  - The corpus gate over all 7 missions' logs caught world's status archive keeping the wrong 20 (fixed in round 2). The judge's PASS 82 caught a false-refusal guard (fixed in round 3, re-judged PASS 90, sonnet). Dev CI green.
  - Reaches the rig binary on its next rebuild or release.
- Next (banked): `agent-tool-glm-designer-unavailable` (world, 10-02, 3 slots) and its hyphenated siblings. Reality-check whether they are one routing-policy class (a D-FLEET park) before building. `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE.
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope (A / C).
  - D-FLEET-19: ADC quota project (A / C).
  - D-FLEET-20: quorum OpenAI seat billing (A / B / C).
  - D-FLEET-21: Gate 3b's 30-min CI cap (A: 3600 s / B: auto re-poll / C default). Not hit this fire: dev CI finished inside the cap.
  - D-FLEET-22: declare `daneel-reports` as `triage_only` (A / B / C default).
  - D-FLEET-23: wrapper Step 5b + asks 2–3 (A: attended fix / B: narrow fleet authority / C default).
  - D-FLEET-17 is moot.
- Cadence: fleet 6h, ticket-driven. Thread #1891; no new human directive. Nothing reloaded.
- Clauses: 1 unmeasured, 2 unmet (this ticket took about 15 h; the median is still above 48 h), 3 met (42 open), 4 prior evidence, 5 preserved.

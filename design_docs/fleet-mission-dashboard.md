# Fleet mission — iteration 50, 2026-10-20

- Latest released binary: v0.54.16. dev: `e5ab9f84f` (#1907, this fire).
- LANDED and RESOLVED: `probe:pi-duplicate-extension-discovery` (triage, 10-19):
  - The skill's prescribed pi probe (`mission-control/resources/gate-3-route.md`, both copies) now passes `--no-extensions`. Before, it was rc 1 from any checkout with `.pi/extensions`: the global and repo copies of each tool conflict. The driver was already fixed; the skill was not.
  - The raw sandboxed pi invocation splices `_mc_pi_ext_args`, as `mission_pi_run.sh` does.
  - `test_pi_ext_args.sh` goes from 8 to 14 arms and sweeps both skill copies. Judge PASS 90, then PASS 91 (sonnet) after a sweep hardening. Dev CI green.
- Next (banked): `rotate-log:legacy-heading-format` (stapledon, 10-20). rotate-log ignores legacy headings, so stapledon's 289 KB log never rotates, and a no-op run writes a false "ARCHIVED" notice. In scope; reality-check it first. `agent-tool:*-unavailable` stays PARKED-ON-LANE.
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope (A / C).
  - D-FLEET-19: ADC quota project (A / C).
  - D-FLEET-20: quorum OpenAI seat billing (A / B / C).
  - D-FLEET-21: Gate 3b's 30-min CI cap (A: 3600 s / B: auto re-poll / C default). Hit again on the dev poll this fire, 6 instances in total.
  - D-FLEET-22: declare `daneel-reports` as `triage_only` (A / B / C default).
  - D-FLEET-23: wrapper Step 5b + asks 2–3 (A: attended fix / B: narrow fleet authority / C default).
  - D-FLEET-17 is moot.
- Cadence: fleet 6h, ticket-driven. Thread #1891; no new human directive. Nothing reloaded.
- Clauses: 1 unmeasured, 2 unmet (this ticket took about 13 h; the median is still above 48 h), 3 met (43 open), 4 prior evidence, 5 preserved.

# Fleet mission dashboard — iteration 126

Updated 2026-11-20; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED and RESOLVED [HARNESS].** Row 67 `driver:idle-fingerprint-head-ignores-design-docs-verification` (world, 2 slots), via #2431 → `ff08818ae`.
  - The product missions' idle pre-check now excludes `design_docs/verification/<name>-iter*` records from `head=`, so record-only commits no longer force a full fire. The root `verification/` exclusion is kept.
  - Independent judge (sonnet): **PASS 92**, 0 blocking. Mutants 15/15 killed (controller re-run; the executor's and judge's counts were both off).
  - World should expect one more full fire: its pending snapshot predates the fix.
- **Dev:** CI 18/18 and Build and Release green at `ff08818ae`.
- **Done-gate:** stapledon dry-runs (healthy and degraded) rc 0; world was mid-fire, so it was not used. `test-launchd-drivers` 59/59.
- **Tickets:** 16 open (was 17). All are ruling-gated or out of scope.
- **Routing:** controller opus; planner opus, executor haiku, judge sonnet (Agent tool, foreground). Codex was over its daily ration this fire.
- **Cost:** metered $0. Planner 77,305 · executor 61,692 · judge 50,701 tok.
- **Parked on Mark:** D-FLEET-44, D-FLEET-45 and D-FLEET-46 (unchanged).
- **Next:** verify the attended D-42 merges (#2143, #2150) against their tickets; D-FLEET-46.
- **Evidence:** [log](fleet-mission-log.md) and [iteration126/](fleet-mission-evidence/iteration126/).

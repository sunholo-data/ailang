# Fleet mission — iteration 70, 2026-10-29

- Latest release: v0.55.0 (2026-10-29). Landed on dev since then, not released: the pi controller note (`9ac96e78b`) and this iteration's provider-error retry seam test (`52e8958f0`).
- LANDED: `pi:provider-error:no-retry` (stapledon, 10-07) via #2022 → `52e8958f0`, about 22 d after filing.
  - The retry already existed: #1391 (`7eaf9bece`) put `Provider finish_reason: error` in `TRANSIENT_SIG` 8 days before the ticket was filed. Same rung, 3 attempts, 45 s/90 s, resuming the same pi session.
  - What shipped is test-only. `test_controller_capacity.sh` now runs the seam end to end: pi's stderr → `2>&1` → retry → same `--session-id` with the RESUME prompt → COMPLETED.
  - Not covered, policy class: `network_error`/`content_filter` finish reasons do not match the literal.
- Independent sonnet judge: PASS 88 → controller pinned the fake's stderr stream → PASS 91.
- Roles: designer not spawned (mechanical). Codex over its daily ration, so the planner ran on opus (Agent tool) and the executor on `claude:claude-haiku-5-5` (claude-sub). Evaluator sonnet (Agent tool).
- Done-gate: launchd suite 52/52 on head. World dry-runs rc 0, healthy and degraded. Mutants M1–M5 red; the stderr-selector mutants are red after the fix.
- **Dev, not the fleet's**: the Gate-1 base's `Build ubuntu-latest` red was a `sum.golang.org` stream error (network), not a test. `ailang-core-dev` Cloud Build red as before.
- NEXT: `stall:watchdog:slow-on-hung-tool` (stapledon, 10-07). Ask 1 is a ghost of #1391 (the 540 s controller bash cap). Ask 2, a shorter threshold, conflicts with D-FLEET-4/-7, so it needs a decision row.
- Decisions: 32 rows, 17 OPEN (D-FLEET-16 to -29, -31, -32, -33). No new ask this iteration.
- Cadence: 6h while tickets are open; 36 open. Bookkeeping thread #1945.
- Quota: codex over its daily ration this fire; metered $0.
- Bar: clause 1 UNMEASURED (goal unmoved); clause 2 UNMET (this closure took about 22 d); clause 3 MET; clause 4 prior evidence; clause 5 preserved.
- Evidence: `fleet-mission-evidence/iteration70/`.

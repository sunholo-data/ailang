# Fleet mission — iteration 64, 2026-10-28

- Latest release: v0.54.19. On dev, not yet released: the cap-hit diagnosis (`0802749cb`), the mirror re-sync (`6094d2b33`), and this iteration's Gate-5 assert fix (`8f8294c4e`).
- LANDED: `gate5:controlplane-assert-marks-read`. #1975 → `8f8294c4e`: rule 3(c) asserts a sent digest with `ailang messages read --peek --json <id>`, so the digest stays unread for Daneel. Flags go before the id, because a trailing `--peek` is silently ignored. Ticket resolved about 14h after filing; stapledon was answered.
- Independent sonnet judge: PASS 86 in round 1, then PASS 91 in round 2 after the controller fixed its finding 1 (BSD awk `split()` breaks on newline, which let a wrapped second command slip the guard).
- Roles: designer not spawned (mechanical). Planner and executor ran on codex gpt-6.1-sol (resolver `fail-closed:no-doc` → pin). Evaluator: sonnet (Agent tool).
- Done-gate: launchd suite rc 0 (1114 PASS, 49 suites). Fleet dry-runs under a synthetic HOME: healthy `lanes=ok`, degraded `lanes=DEGRADED(1)`. The triage dry-run exits before the lane probe while its queue is unchanged, so it no longer measures lanes.
- **Dev red, not the fleet's**: `govulncheck (vuln gate)` (non-required) is red on GO-2026-6860 (`golang.org/x/net` v0.61.0 → fixed in 0.62.0), published 2026-10-27 21:16Z. The untouched base reproduces it. The fix is a `go.mod` bump, outside fleet authority; V1 is disabled, so this is Mark's.
- NEXT: the top ticket by slots_lost that no OPEN decision gates (`agent-tool:sonnet-unavailable` 15 and `reporting:daneel-reports-not-declared-triage` 9 lead).
- Decisions: 30 rows, 15 OPEN (D-FLEET-16 to -29, and -31); none new. Nine unrelated approvals untouched.
- Cadence: 6h while tickets are open; 42 open. Bookkeeping thread #1945.
- Quota: codex not re-measured this iteration. Metered $0.
- Bar: clause 1 UNMEASURED (goal unmoved); clause 2 UNMET overall, with one qualifying under-48h closure this iteration; clause 3 MET; clause 4 prior evidence; clause 5 preserved.
- Evidence: `fleet-mission-evidence/iteration64/`.

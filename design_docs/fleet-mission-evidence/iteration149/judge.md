# Iteration 149 independent judge (evaluator, read-only)

Spawned by the Agent tool: `subagent_type=general-purpose`, `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, read-only, 1 spawn (55,604 tok, 6 tool calls, about 55 s). Verdict below, in substance. Controller annotations follow it.

## Verdict: NONE. Confidence 90%.

No ticket or decision row is routable this fire.

## Re-derived (first-party)

1. **Tickets:** the judge re-ran `ailang mission ticket open --json` itself (rc 0): 19 tickets, identical on (signature, slots_lost, last_seen) to both iteration 148's `tickets.json` and the staged iteration-149 file. Positive control: bumping one `slots_lost` in a copy of the 148 set made the comparison report a difference. Newest `last_seen` 2026-11-24.
2. **Ledger:** `mission_decisions.sh --check` rc 0, 48 rows; OPEN D-FLEET-44, 45, 46, 48, 49. `git log 1ac8bbf0a..origin/dev` on the charter and rulings returns only `17fe14092` (the fleet's own i148 record, #2489), which is the positive control. No attended ruling.
3. **Directives:** `mission_directives.sh --issue 2445` (no `--mark`): 0 allowlisted directives of 22 comments; no watermark file, whole issue read.
4. **Not re-checked by the judge:** the 14 unread `mission-fleet` inbox messages, the pin-behind claim, the skill-drift reading.

## Per-ticket dispositions (19)

| Ticket | Disposition |
|---|---|
| `skill:idle-record-barred-on-degraded-no-pick-fire` | Blocked on D-FLEET-45 (OPEN). Row 62. |
| `reporting:daneel-reports-not-declared-triage` | D-FLEET-22 ruled A, rolled out attended; resolve is attended/D-FLEET-46 class. Row 24. |
| `quorum:gpt6-1-sol-openai-api-credits-exhausted` | Merged attended (#2347); resolve waits on D-FLEET-46. Row 21. |
| `pi:openrouter:uncapped-max-tokens-402` | Merged attended (#2349); D-FLEET-46. Row 50. |
| `executor:codex-credential-collision-drops-automatic-planner-handoffs` | Merged attended (#2348); D-FLEET-46. Row 54. |
| `executor:codex-credential-lease-collision-fails-concurrent-dispatch` | Merged attended (#2348); D-FLEET-46. Row 54. |
| `coordinator:job-retry-success-dropped-after-failed-completion` | D-FLEET-39 A, fix in #2348; D-FLEET-46. Row 55. |
| `executor:3h-timeout-from-queue-and-unsliced-sprints` | Merged attended (#2343); D-FLEET-46. Row 56. |
| `executor:timeout-loses-work-and-may-never-start` | Merged attended (#2343); D-FLEET-46. Row 56. |
| `quorum:claude-p-ration-usage-429` | Merged attended (#2344); D-FLEET-46 (the eighth ticket). Row 68. |
| `messaging:pubsub-notify-wrong-project` | D-FLEET-19 ruled A, set attended; attended resolve. Row 18. |
| `crash:sprint-executor:sigbus-ram-tmp-full-make-test` | Ask 1 landed (#1905); rest parked under D-FLEET-23, terraform/multivac, outside scope. Row 27. |
| `coordinator:cloud-lane-capped-by-local-worktree-limit` | Coordinator code, outside Authority (judge unconfirmed; see annotation). Row 52. |
| `ci:distinct-build-test-failures` | D-FLEET-17 class (dev reds, V1-owned). Row 44. |
| `ci:verify-gap-map-workflow-wiring` | Outside Authority; D-FLEET-46/48 class. Row 71. |
| `ci:agy-pin-test-home-isolation` | Outside Authority; D-FLEET-46/48 class. Row 72. |
| `ci:modelreg-qwen-slug-pricing-mismatch` | Outside Authority; D-FLEET-46/48 class. Row 73. |
| `eval:motoko-connection-probe-orphaned` | Folded into D-FLEET-18 (RESOLVED); its surface `tools/eval/` is refused by the scope guard. Row 58. |
| `tool-hang:triage-judge:home-metadata-search` | Parked on D-FLEET-44 (OPEN). Row 61. |

## Strongest objection and answer

**Objection:** `eval:motoko-connection-probe-orphaned` has no OPEN row of its own, and the "attended resolve only" tickets could be closed as bookkeeping.
**Answer:** the probe ticket's surface is `tools/eval/`, which the scope guard refuses; folding into D-FLEET-18 is not a fresh authorisation. Closing attended-fixed tickets is exactly D-FLEET-46's open question (default B). Picking either would pre-empt an OPEN ruling. Same as iterations 144, 147, 148.

## gates.md

No factual errors in the parts the judge checked. Observation: D-FLEET-49 (filed i148) is the only live route to cutting the repeated NONE-fire cost, and it is still OPEN.

## Controller annotations

- The judge could not place `coordinator:cloud-lane-capped-by-local-worktree-limit`. The controller did: queue row 52 (`[LANDED · iteration 115 · D-FLEET-37 A]`) records that asks 1–2 (the coordinator `MaxWorktrees: 3` cap) "travel with row 56's coordinator ticket (V1/attended)", i.e. coordinator code outside fleet Authority — the same disposition iteration 148's judge gave (row 52).
- The skill-drift and pin-behind readings are the controller's (gates.md); the pin delta is 10 triage-only commits touching 6 `design_docs/triage-mission*` files.

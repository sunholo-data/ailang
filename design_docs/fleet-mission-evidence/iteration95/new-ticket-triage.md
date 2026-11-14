# New ticket: tool-hang:triage-judge:home-metadata-search

Controller triage only, UNJUDGED. Sources are the filing ticket plus original typed runner verdicts copied here from triage 33; this is artifact remeasurement, not a fresh live hang reproduction.

First attempt: fenced=true, verdict tool_hang rc18 after 1119seconds, 148 completed tools, zero provider errors/commits, worktree_changed_since_start=false. Literal command traverses HOME for recent JSON/JSONL metadata. No expanded HOME value was read. The controlled same-lane retry is fenced, verdict ok rc0, 274seconds, 75 tools, one agent_end, zero provider errors/commits. This confirms recovery under the already-prescribed remedy; it does not prove every judge item was substantively reviewed.

Current source scripts/mission_pi_run.sh classifies an open tool at the stream-dead boundary as tool_hang; gate-3-route.md explicitly says rc18 is not a lane failure, permits one same-lane retry forbidding the hung command and requires a harness ticket. Triager followed that existing policy. No threshold, retry, lane order or billing change is authorized or inferred; no production defect beyond the command-level hang has been established.

PARKED-ON-LANE: independent evaluator role unavailable. Commands/rc/reset are in evaluator-transport.json. Resume predicate: re-probe the driver-selected declared independent evaluator lane next fire; on rc0 independently review this ticket, the existing successful retry, and whether any in-scope mechanical change is warranted. Do not re-run the unbounded HOME search. Not needs-human-review; no new decision row, no request to wait as a human decision. Ticket remains unread/open. A recovered command does not authorize a controller-only ticket resolution.

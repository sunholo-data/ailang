# M-MISSION-ITERATION-RELIABILITY sprint

Status: approved design and execution by Mark, attended 2026-09-08: “yep please continue”,
in response to the offer to plan and execute this reliability follow-up.
Design: m-mission-iteration-reliability.md. Base: 7b94968cb; isolated sprint/mission-iteration.
No main-checkout edits. No fleet rollout or outbound messages.

## Capacity and ordering

Three independent implementation lanes, followed by integration and live verification.
Estimate 3 engineering days plus one validation day, ~1,800 code/test lines; not a calendar
promise. Recent canary prep and 61-line loader repair are poor predictors of lifecycle
work; the estimate is based on the bounded state/CLI/test surfaces, with one day buffer.
Use existing runtime/store/executor interfaces. Full runtime baseline tests already passed;
write failing boundary tests for new behavior and verify changed packages per lane.

## M1 — focused review packet and observed progress (~550 lines)

Own iteration review_packet*.go and runtime_stage.go integration; dispatch progress*.go,
run.go and relevant tests. Build <=128KiB immutable read-only packet outside stage roots,
with exact revisions, diff completeness, criteria, checks and authority/provenance refs.
Bind digest in request; keep context readable and tell reviewers to settle criteria and finish.
Observer uses existing executor events; bounded exact-call repetition diagnostics only.

- [x] Packet binds candidate and all required evidence; truncation and missing evidence explicit.
- [x] Packet cannot be overwritten through candidate-controlled paths; digest fixed for replay.
- [x] Record bounded tool/progress counters without storing secrets/tool output in status.
- [x] No new repeated-read termination heuristic or model fallback.

## M2 — evaluator-only preparation and actionable recovery (~550 lines)

Own iteration retry_review*.go, cmd/ailang mission recovery command/status/help integration,
and coordinator additive attestation support only if required. Prepare-only retry-review
writes a reviewable successor and immutable acceptance manifest. Never creates approval.
Use supplied authority for final validation; draft may explicitly need approval.
Expose versioned confirm-stopped only for supported cancellation/deadline states.

- [x] Preserve original inputs/acceptances and actual author identity; no author redispatch.
- [x] Reject live/ambiguous parents, missing objects, invalid authority and unsafe destinations.
- [x] Explicit new limits/route; original criteria/scope/checks retained.
- [x] Non-completed status explains next valid action; typed categories preserve reason text.
- [x] Confirmation records operator attestation and refuses stale/wrong/unsupported states.

## M3 — recoverable activation/cleanup (~700 lines)

Own new internal/mission/activation package and its tests; no CLI edits in parallel.
Persist ownership and old/installed bytes before local marker/binding changes. Exclusive
host operation lock; compare-before-restore; no secrets. Crash-recoverable state, idempotent
recovery. Process-stop verification is an explicit injected boundary, never inferred from
a terminal DB label. Parent integrates CLI after worker reports API.

- [x] Completion and terminal failure restore absent/present prior state after verified stop.
- [x] Changed files/foreign markers/unknown child state remain held with actionable reason.
- [x] Death around each mutation is recoverable; concurrent operations refuse ownership.
- [x] Retain runtime DB/evidence; read-only inspection works after binding restoration.

## M4 — integration and live adoption (no fixed production LOC)

Wire activation CLI and status progress. Read sprint-evaluator for independent review.
Run changed packages/race, lint/build, boundaries and relevant CLI/shell checks. Full suite
once after integration, with GOFLAGS=-p=2 and existing isolated Go/lint caches.

- [x] Independent review passes with zero unresolved code blockers.
- [ ] Existing candidate passes live evaluation and completed replay makes zero provider calls.
- [ ] Two additional frozen bounded tasks finish with hard checks and independent review.
- [x] Combined new trial spend <=$5; explicit per-item limits; no automatic budget escalation.
- [x] Cleanup/recovery validated and actual costs, tokens, interventions recorded.

Live tasks: first existing guide candidate. Additional two briefs frozen before dispatch
at integration, restricted to small documentation corrections uncovered by source audit;
if no justified approved task exists, report adoption partial rather than fabricate work.
No website publish or merge. Transport failures consume remaining trial budget; no automatic
infinite retries. Productive success and operational containment reported separately.

Live result 2026-09-08: first reviewer failed at 100,024/100,000 fresh tokens; automatic cleanup and failed-terminal replay pass. Existing-candidate acceptance and both additional tasks remain open. See `design_docs/verification/mission-iteration-reliability/live-trial.md`.

# M-DASHBOARD-CLOUD-READS — retained Work query contract

Status: Implementation complete; formal verification gate paused. Priority P1. Target: next release. Authorized continuation of the retained API/CLI repair after dashboard deletion sprint A. Work on dev; no cloud mutations or infrastructure apply.

Design basis: dashboard-recovery-plan-2026-09-08.md and live F3/F4. This is its bounded read-contract increment, not a new data model. Estimate: two engineering days; approximately 700 added/changed implementation/test lines. Prior UI deletion velocity is not predictive for backend query work.

## Verified premises and scope

Firestore ListChains ignores Offset, applies AgentID after Limit and swallows stage-query errors; ListSpans ignores Offset/order/workspace filters. SQLite sorts chains by created_at DESC and spans by start_time ASC without explicit tie breakers. API list supports limit/offset, CLI chains list lacks offset. Stage reads require compound indexes; infrastructure lives in adjacent ailang-multivac/terraform/firestore.tf, where stage span/chat indexes were not found. Backend failure is sometimes labeled stage-not-found or generic500.

## Contract

For a fixed cohort, chain order is created_at DESC then id DESC, CreatedAfter is exclusive, defaults limit50. Span order is start_time ASC then id ASC, time bounds inclusive. Apply filters before page boundaries. Negative offset is invalid. Preserve existing public shapes; empty arrays instead of false failures. Mutable offset pages are not snapshot/cursor guarantees.

Agent filtering may scan ordered candidate chains but stops after enough matched records; stage-query errors return errors, never an empty/partial success. Span workspace filtering is explicitly unsupported on Firestore until the join contract exists, instead of silently ignoring it. No new stored identity fields or database migration.

API and CLI share Backend query semantics; expose offset in existing chains list, document remote usage. Stage paths verify ownership and distinguish missing stage from denied/unavailable/index-not-ready queries. Missing index returns explicit retryable503, not empty evidence. Record exact required infrastructure index definitions without deploying them.

## M1 ✅ — Query contract
- Firestore chain/span offsets and deterministic tie order verified via hermetic real SDK request tests.
- Agent filter happens before paging; join read failures propagate.
- SQLite fixed-cohort paging agrees with contract; unsupported Firestore workspace filters fail explicitly.

## M2 ✅ — Retained API/CLI
- chains list exposes validated offset and help; HTTP list forwards declared workspace/repo filters.
- Stage spans/chat reject wrong chain; storage failure is not404/empty success; index-not-ready is503 with named error.
- Endpoint and command parsing regression tests use isolated fixtures; no CLI startup on real home data.

## M3 — verification gate pending Verification and rollout record
- Relevant Go tests/race, build/lint/boundaries; independent evaluation records remaining limitations.
- Document exact required indexes and deployment verification in the existing infrastructure workflow; no apply here.
- Update changelog/recovery plan; no production stability claim. Full fleet capture, cost normalization, cursor pagination and approval delivery remain future work.

Axioms: A1+1 A2 0 A3 0 A4+1 A5+1 A6 0 A7+1 A8 0 A9 0 A10+1 A11+1 A12+1 =+7. No language semantics changed; no compiler conflict surface or .ail fixture required. Use documented sprint JSON directly to avoid inbox-mutating scripts; validate it.

## Verification record

Affected storage/API packages pass tests and race detection; CLI parser regression
tests pass. Full `make test` reaches four known environment failures: two memory
watchdog cases, process-group RSS sampling, and a registry fixture requiring real
home-cache writes. This is not a full-suite pass. A filtered run is recorded
separately; independent evaluation retains the formal gate. No production read
acceptance, index apply or deployment has occurred.

Filtered `make test` passes with exactly those four tests excluded. Final lint
reports zero issues; final storage/API race tests pass. The full-suite gate remains
pending an environment that can run the excluded tests.

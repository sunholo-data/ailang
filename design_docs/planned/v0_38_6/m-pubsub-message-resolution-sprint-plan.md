# Sprint Plan: M-PUBSUB-MESSAGE-RESOLUTION

## Summary

Prevent coordinator jobs from being created with a bare message ID when a Pub/Sub notification cannot resolve its durable Firestore inbox record. The sprint makes resolution failure structured and retryable, preserves valid self-contained cascade messages, and proves the contract across adapter, push-handler, and task-creation boundaries.

**Duration:** 3 days (approximately 18 engineering hours)
**Dependencies:** Approved design amendment covering the resolution/ack contract; access to a dev GCP plane for the final controlled verification
**Risk Level:** Medium
**Target:** v0.38.6 follow-up
**Trigger:** `msg_20260914_155442_a7154543` / `task-a7154543`

## Current Status Analysis

### Observed Failure

- The task record contains title `Pub/Sub notification from coordinator` and content equal to the unresolved message ID.
- The referenced inbox document does not resolve through the canonical GCP message store or a direct Firestore document lookup.
- `internal/coordinator/pubsub_adapter.go` deliberately falls back to `Content: notification.MessageID`, logs the fetch error, buffers the message, and returns `nil`. The push handler therefore acknowledges the notification and downstream task creation treats the pointer as an instruction.
- Cascade notifications are a distinct valid case because they may embed a self-contained envelope; that compatibility path must remain intact.

### Systemic Analysis

The defect is not limited to this message ID. Every non-cascade Pub/Sub notification whose durable record is missing, delayed, malformed, retained under a different key, or temporarily unreadable follows the same silent fallback. The unified fix belongs at the adapter's resolution boundary: unresolved pointer notifications must never become executable messages.

Related paths audited:

- `internal/coordinator/pubsub_adapter.go` — notification decode, durable-record lookup, unsafe fallback, buffer/ack decision
- `internal/coordinator/daemon_http.go` — push response determines Pub/Sub acknowledgement/redelivery
- `internal/coordinator/daemon_tasks_polling.go` — buffered messages become tasks
- `internal/storage/firestore/messaging_inbox.go` — Firestore document/`message_id` identity normalization and lookup contract
- `internal/pubsub/` publisher/notification types — pointer and self-contained envelope formats

### Velocity and Capacity

- The shallow coordinator checkout exposes only one recent commit, so the repository does not provide a trustworthy multi-day LOC/day baseline.
- Estimate by bounded surface area and test burden instead: approximately 480 LOC over 3 days, including about 55% tests and fixtures.
- A 25% uncertainty buffer is included for Pub/Sub ack/redelivery and Firestore error classification.

## Proposed Milestones

### Milestone 1: Specify and Test the Resolution Contract

**Goal:** Make pointer, self-contained, transient-failure, permanent-missing, and malformed-notification outcomes explicit before behavior changes.
**Estimated:** 40 LOC implementation/types + 110 LOC tests = 150 LOC
**Duration:** Day 1

**Example files to update/create:**

- Update `internal/coordinator/pubsub_adapter.go`
- Create `internal/coordinator/pubsub_adapter_resolution_test.go`
- Update notification fixtures under `internal/pubsub/` only if the existing types cannot express the cases

**Tasks:**

- Introduce a small internal resolution result/error taxonomy that distinguishes malformed wire data, transient store failure, durable not-found, and valid self-contained messages.
- Add table-driven tests showing that a bare pointer is executable only after its durable inbox row resolves.
- Add regression coverage proving cascade envelopes remain processable without a second store fetch.
- Add a mutation arm: replacing unresolved content with the message ID must fail the tests.

**Acceptance Criteria:**

- [ ] No test accepts a bare message ID as executable task content.
- [ ] Transient and not-found lookup failures are distinguishable in logs/errors.
- [ ] Existing cascade-envelope behavior remains covered and passing.
- [ ] `go test ./internal/coordinator ./internal/pubsub` passes.

### Milestone 2: Fail Closed and Preserve Delivery Semantics

**Goal:** Prevent unresolved pointer notifications from entering the adapter buffer while giving transient failures a bounded redelivery path and permanent failures an explicit terminal disposition.
**Estimated:** 90 LOC implementation + 100 LOC tests = 190 LOC
**Duration:** Day 2

**Example files to update/create:**

- Update `internal/coordinator/pubsub_adapter.go`
- Update `internal/coordinator/daemon_http.go`
- Update `internal/coordinator/pubsub_adapter_resolution_test.go`
- Update/add focused HTTP push tests in `internal/coordinator/daemon_http_test.go` or the existing split push-handler test file

**Tasks:**

- Buffer only fully resolved messages or explicitly self-contained envelopes.
- Return a retryable error for transient durable-store failures so the push endpoint does not acknowledge successful processing.
- Define bounded handling for durable not-found after publisher/store ordering is verified: either retry according to delivery-attempt metadata and then dead-letter/record a structured failure, or reject immediately if the design proves the row can never appear later.
- Ensure malformed payloads cannot produce retry storms; retain an explicit ack-and-record policy for non-recoverable wire errors.
- Preserve tag-filter nack behavior and outcome-notice filtering.

**Acceptance Criteria:**

- [ ] An unresolved pointer never reaches `ListUnread` and never creates a coordinator task.
- [ ] Transient lookup failure produces the push response required for Pub/Sub redelivery.
- [ ] Permanent failure has a bounded, queryable outcome rather than infinite retries or silent acknowledgement.
- [ ] Duplicate delivery remains idempotent.
- [ ] Existing coordinator tests pass with no weakening of tag-routing or cascade tests.

### Milestone 3: End-to-End Proof, Observability, and Operator Guidance

**Goal:** Prove that stored messages dispatch with their full directive, missing messages do not dispatch, and operators can diagnose the resolution state from structured evidence.
**Estimated:** 30 LOC implementation/telemetry + 90 LOC integration tests + 20 LOC docs = 140 LOC
**Duration:** Day 3

**Example files to update/create:**

- Create or update `internal/coordinator/pipeline_message_resolution_test.go`
- Update `docs/docs/guides/coordinator-workers.md` or the current coordinator troubleshooting guide
- Update `CHANGELOG.md` or the active version changelog

**Tasks:**

- Add a pipeline test spanning notification decode, store lookup, adapter buffering, and task creation.
- Assert structured fields/counters for `resolved`, `retryable_failure`, `permanent_failure`, and `self_contained` outcomes without logging private payloads.
- Run a controlled dev-plane pair: one normal stored notification and one synthetic missing-record notification; verify task and ack/redelivery outcomes from durable records, not only logs.
- Document remediation for missing-record/dead-letter events and the invariant that task content is never a message pointer.

**Acceptance Criteria:**

- [ ] Normal notification creates exactly one task containing the original directive.
- [ ] Missing-record notification creates zero tasks and leaves a structured retry/dead-letter trail.
- [ ] Self-contained cascade notification still creates the expected task without a durable inbox lookup.
- [ ] No metric or log includes the private message payload.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.
- [ ] Dev-plane evidence records message ID, delivery attempt, resolution outcome, and resulting task count.

## Day-by-Day Plan

| Day | Deliverable | Exit Gate |
|---|---|---|
| 1 | Resolution contract and red tests | Design amendment approved; table tests cover all outcome classes |
| 2 | Fail-closed adapter and bounded delivery behavior | No unresolved pointer can enter the task pipeline |
| 3 | Pipeline proof, observability, docs, dev verification | Full validation green and controlled dev evidence captured |

## Success Metrics

- Zero tasks whose content equals their `message_id` in the controlled cohort.
- 100% of pointer notifications classified as resolved, retryable failure, or permanent failure.
- Exactly-once task creation under duplicate delivery tests.
- At least one integration fixture each for resolved pointer, transient failure, permanent missing record, malformed payload, and self-contained cascade.
- No regression in coordinator, Pub/Sub, messaging-store, lint, or architecture-boundary suites.

## Dependencies and Approval Gate

- This plan does **not** authorize implementation. The existing design does not approve changing the Pub/Sub acknowledgement contract for this newly observed case.
- Before `sprint-executor`, amend an approved coordinator/message-plane design (or create a focused design doc), obtain explicit user approval, and resolve the bounded not-found policy.
- Final dev-plane verification requires read/write authority for isolated synthetic messages and task inspection; production testing is not required.

## Risks

| Risk | Mitigation |
|---|---|
| A store write can become visible shortly after its notification | Treat eligible lookup failures as retryable; decide retry bounds from actual publisher ordering and delivery-attempt data |
| Returning errors creates a retry storm | Cap attempts and record/dead-letter permanent failures; malformed wire data follows a non-retryable policy |
| Cascade traffic depends on notification-only data | Preserve and independently test the self-contained envelope path |
| Duplicate delivery creates duplicate tasks | Keep stable message/task identity and add duplicate-delivery integration coverage |
| Private message text leaks into telemetry | Emit identifiers and outcome enums only |

## Open Design Decision

What is the authoritative bounded policy when `GetInboxMessage` returns not-found: retry for a fixed delivery-attempt/time window before dead-lettering, or treat not-found as immediately permanent because the publisher guarantees store-before-publish ordering? The design amendment must answer this with publisher-path evidence before execution.


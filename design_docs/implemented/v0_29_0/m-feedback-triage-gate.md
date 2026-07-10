# M-FEEDBACK-TRIAGE-GATE: Cost & abuse gate for the public feedback → agent pipeline

**Status**: Implemented (M1–M6, v0.29.x) — see sprint plan `sprint-m-feedback-triage-gate.md`
**Target**: v0.15.x (immediately after the Cloud Armor edge throttle ships)

> **Implementation note (v0.29.x).** Shipped as package `internal/feedbackgate`
> (NOT `internal/triage/` — that name collides with the shipped
> M-MSG-TRIAGE-ROUTER). The type is `feedbackgate.Verdict` (dispatch/file/reject),
> the entry point `feedbackgate.Decide(ctx, Input, FeedbackGateConfig)`, the
> config key `coordinator.feedback_gate`, the env vars `AILANG_FEEDBACK_GATE_MODE`
> / `AILANG_FEEDBACK_GATE_DRY_RUN`, and the audit inbox `feedback-gate-audit`.
> Deviations from this doc's prose (recorded in the sprint plan): Decide operates
> on a coordinator-free `Input` built from `coordinator.Message` (no import
> cycle); IP-hash cooldown keying is dropped (IP never reaches this layer);
> `reject` marks/ack's rather than deletes (TTL cleanup); the terraform 80%-budget
> alert (sibling repo) and the dashboard panel / `--show-triage` flag are
> out-of-scope follow-ups. M5's classifier budget cap is logic+config only.
**Priority**: P0 (cost/safety gap on a public unauthenticated endpoint that fans out to Sonnet-driven agents)
**Estimated**: 2 days, ~700 LOC + config
**Dependencies**:
- M-AGENT-MCP-ONBOARDING shipped (`submit_feedback` accepts `package` + `auto_dispatch`)
- M-PKG-FEEDBACK-LOOP M2 shipped or in flight (the `pkg-feedback.md` template that this doc relies on for "stop at file" behaviour)
- Cloud Armor edge rate limit on `mcp.ailang.sunholo.com` (covered separately under the M7 work; this doc treats it as a precondition, not a milestone)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A2: Replayability | **+1** | Triage decisions are deterministic-first; only the LLM-classifier branch is non-deterministic, and it is logged with seed/model/prompt hash |
| A4: Explicit Authority | **+2** | The agent only fires Sonnet on messages that pass the triage gate AND carry the `auto:` category prefix. `auto_dispatch=false` never reaches an LLM |
| A7: Machines First | **+1** | The triage classifier itself is a Haiku JSON-mode call: cheap, structured, deterministic-enough to log and replay |
| A11: Structured Failure | **+2** | The gate emits `triage.rejected{reason}` events into the inbox so humans can audit why a message was suppressed; no silent drops |
| A12: System Boundary | **+1** | Tightens the boundary between "anonymous public submission" and "authorized agentic action" — currently a straight wire |

**Net Score: +7** → **Decision: move forward**

---

## Problem Statement

The public MCP endpoint `mcp.ailang.sunholo.com/submit_feedback` is unauthenticated and currently flows like this:

```
public submit_feedback  →  Firestore inbox_messages  →  Pub/Sub ailang-messages
  →  coordinator pollAndProcessTasksCloud()  →  CreateTask  →  Cloud Run Job
  →  Claude Sonnet (pkg-feedback.md template)  →  GitHub PR / issue
```

Every step after submission is automatic. There is **no triage between Firestore and Sonnet**. The package agents fire on arrival.

### Concrete attack scenario

1. Adversary scripts 1,000 calls to `submit_feedback` (no auth, no per-IP limit today)
2. 1,000 Firestore docs land in seconds
3. Coordinator pulls Pub/Sub, fans out to ~11 `pkg-*` agents
4. Each fan-out spawns a Cloud Run Job that invokes Sonnet with the user-controlled body
5. Estimated burn at $0.015–0.05/call × 1,000 ≈ **$15–50 per flood event**, plus Cloud Run Job minutes
6. Side effects: GitHub issues opened by the agent, dashboard noise, possible prompt-injection on Sonnet via the body field

The `task_max_cost: 150` budget cap is a *backstop after damage is done*, not a flood gate.

### Why edge rate limiting alone is insufficient

A Cloud Armor 5/min/IP throttle blocks unsophisticated floods but does **not** stop:
- Distributed submitters (10 IPs × 5/min = 50/min)
- A single legitimate-looking submitter sending one prompt-injection feedback per minute
- Internal agent-to-agent loops where the source IP is the agent fleet itself
- Spam from one human writing 5 messages/min in good faith — still gets 5 Sonnet runs

We want **defense in depth**: edge throttle (covered separately) + a triage gate (this doc) + per-contact cooldowns + the existing budget cap.

### What does NOT need solving here

- **Authentication** for `submit_feedback` — out of scope. The whole point of the public MCP is anonymous read + write. Adding auth defeats the AI-agent onboarding goal.
- **GitHub-side abuse** — covered by GitHub's own rate limits and the agent's existing OAuth scope.
- **Prompt-injection robustness of Sonnet** — separate problem (M-AGENT-PROMPT-HARDENING territory).

---

## Proposed Plan

The gate sits between coordinator pickup and Cloud Run Job dispatch:

```
coordinator pollAndProcessTasksCloud()
  →  triage.Decide(msg)  ←───────────────────  NEW
       ├─ deterministic rules                    (rejects ~70% of bad msgs)
       ├─ per-contact cooldown                   (rejects bursts)
       └─ Haiku JSON classifier (last resort)    (cheap pre-screen, ~$0.0002/msg)
  →  CreateTask  ONLY IF Decision == "dispatch"
```

### M1: Deterministic pre-filter (~150 LOC, ~3h)

A new package `internal/triage/` with a single entry point:

```go
type Decision struct {
    Action    string // "dispatch" | "file" | "reject"
    Reason    string // structured code: "missing_auto_prefix", "spam_pattern", etc
    Cost      float64 // estimated agent cost if dispatched, in USD
}

func Decide(ctx context.Context, msg *messaging.Message, cfg Config) (Decision, error)
```

Deterministic rules (no LLM call):

| Rule | Action | Reason code |
|------|--------|-------------|
| `auto_dispatch=false` (no `auto:` prefix on category) | `file` | `not_authorized_for_dispatch` |
| Body length > 8KB after trim | `reject` | `body_too_large` |
| Body matches obvious spam regex (URLs > 5, base64 blobs > 1KB, etc) | `reject` | `spam_pattern` |
| Same body hash submitted in last 1h | `file` (dedup) | `duplicate_recent` |
| `from_agent` not in allowlist (`mcp-public`, `agent-*`) | `reject` | `untrusted_source` |
| Inbox routing target is not in `pkg:*` or known internal | `reject` | `unknown_inbox` |
| Category not in `{bug, feature, docs, limitation, auto:bug, auto:feature, auto:docs, auto:limitation}` | `file` | `unknown_category` |

**Cost**: zero LLM calls. Pure CPU. Runs in <1ms per message.

**Filing vs rejecting**: `file` keeps the Firestore doc but suppresses dispatch (a human can still triage it later from the dashboard). `reject` deletes the doc and emits a `triage.rejected` audit message into a separate `triage-audit` inbox so we can review false positives.

### M2: Per-contact cooldown table (~120 LOC, ~2h)

A small Firestore-backed sliding window keyed by:
- `contact` field if present
- `from_agent` + IP-hash (when MCP server forwards `X-Forwarded-For`)
- Body content hash (catches identical resubmits across IPs)

Limit: **3 dispatched messages per contact per hour, 10 per day**. Above that → `file` action with `reason=contact_cooldown`. The window is sliding, not fixed, to avoid the "stroke-of-the-hour burst" pattern.

The cooldown table is its own Firestore collection `triage_contacts` with TTL 7 days; documents are tiny (one int counter + last-seen timestamp).

This catches the case where edge rate-limiting allows one legitimate-looking message per minute but a single contact still racks up 60 dispatches/hour.

### M3: Cheap Haiku classifier as last-resort pre-screen (~180 LOC, ~3h)

For messages that pass deterministic rules AND cooldown but still smell off (heuristic: contains code blocks > 200 lines, or category=`bug` with a body > 4KB and no snippet field, or any `auto:` message with no GitHub username in `from_agent`), invoke a **Haiku 4.5 JSON-mode call** before dispatching Sonnet:

```json
{
  "is_genuine_feedback": true|false,
  "is_prompt_injection": true|false,
  "best_category": "bug|feature|docs|limitation|spam",
  "estimated_dispatch_value": "high|medium|low|none",
  "reasoning": "..."
}
```

Cost per call: ~$0.0002 (Haiku 4.5 input tokens at $1/Mtok × ~200 input + ~150 output tokens). At 1,000 messages/day this is $0.20/day max — a 75-100x cost reduction vs running Sonnet on every message.

Decision matrix from classifier output:

| Classifier output | Action |
|-------------------|--------|
| `is_prompt_injection=true` | `reject` |
| `estimated_dispatch_value=none` | `file` |
| `is_genuine_feedback=false` | `file` |
| `best_category != msg.category` | `file` (route mismatch — let a human route it) |
| All checks pass | `dispatch` |

The classifier prompt is checked-in at `prompts/triage_classifier.md` with a content-hash version field so we can replay decisions later. Use the existing `internal/ai/` provider (NOT `internal/executor/`) since this is a one-shot text→JSON call, not agentic work.

**Why Haiku, not Gemini Flash:** Haiku is what the rest of the agent stack uses; cost is comparable; reusing the existing Anthropic auth path is a one-line config addition.

### M4: Wire the gate into the coordinator (~150 LOC, ~3h)

In `internal/coordinator/daemon_tasks_polling.go`, between the Pub/Sub message receive and `CreateTask`:

```go
decision, err := triage.Decide(ctx, msg, c.triageConfig)
if err != nil {
    // structured fail: log + emit triage-audit message + nack pubsub for retry
    return err
}
switch decision.Action {
case "dispatch":
    // existing CreateTask path
case "file":
    metrics.TriageFiled.WithLabel(decision.Reason).Inc()
    emitAuditMessage(msg, decision)
    ackPubsub() // we handled it, don't redeliver
case "reject":
    metrics.TriageRejected.WithLabel(decision.Reason).Inc()
    emitAuditMessage(msg, decision)
    deleteFirestoreDoc(msg.ID)
    ackPubsub()
}
```

The `triage_audit` inbox surfaces rejection reasons in the dashboard's Inbox view. Add a `--show-triage` flag to `ailang messages list` to view them.

### M5: Daily budget cap on triage classifier itself (~60 LOC, ~1h)

Even Haiku has a tail risk. Add a per-day token budget for the triage classifier (default $5/day). If exceeded, the classifier short-circuits to `file` (never `dispatch`), so worst case the inbox fills up and a human triages — never a Sonnet flood.

Track via a daily counter in the same `triage_contacts` collection. Alert via the existing `alert_emails` mechanism in `terraform/security.tf` if 80% of daily budget hits.

### M6: Tests + chaos drill (~120 LOC, ~3h)

- Unit tests in `internal/triage/decide_test.go`: each rule has a passing + failing test case
- Integration test (build-tagged like M-PKG-FEEDBACK-LOOP M1) that submits 100 messages in 60s against test env and asserts:
  - <= 3 reach Sonnet (cooldown working)
  - All 100 land in Firestore (no message loss)
  - Audit messages emitted for every filed/rejected
- "Chaos drill" script `scripts/security/feedback_flood_drill.sh` that submits 1,000 messages over 5 min and reports observed Sonnet invocations + total spend; should round-trip <= $1 vs the baseline ~$25-50

---

## Implementation Plan (2 days, ~780 LOC)

| Milestone | LOC | Time | Files |
|-----------|-----|------|-------|
| M1 deterministic rules | 150 | 3h | `internal/triage/decide.go`, `internal/triage/rules.go`, `internal/triage/decide_test.go` |
| M2 cooldown table | 120 | 2h | `internal/triage/cooldown.go`, `internal/triage/cooldown_test.go` |
| M3 Haiku classifier | 180 | 3h | `internal/triage/classifier.go`, `prompts/triage_classifier.md`, `internal/triage/classifier_test.go` |
| M4 coordinator wiring | 150 | 3h | `internal/coordinator/daemon_tasks_polling.go` (edit), `internal/coordinator/triage_audit.go` (new), config plumbing in `config.cloud.yaml` |
| M5 budget cap | 60 | 1h | `internal/triage/budget.go`, alert wiring in `ailang-multivac/terraform/security.tf` |
| M6 tests + drill | 120 | 3h | `internal/triage/integration_test.go`, `scripts/security/feedback_flood_drill.sh` |

---

## Acceptance Criteria

- A 1,000-message flood from 10 distinct IPs incurs <= $1 in Anthropic spend (vs ~$25-50 today)
- Every suppressed message has a `triage_audit` entry with structured `reason` field
- `make test ./internal/triage/...` passes; `feedback_flood_drill.sh` against test env produces a clean report with the spend line
- Dashboard shows a `Triage` panel: filed-this-week, rejected-this-week, dispatched-this-week, by reason code
- Operator can flip a single env var `AILANG_TRIAGE_MODE=off|file-only|full` to disable the classifier branch in an emergency (deterministic rules + cooldown stay on)

---

## Risks & Tradeoffs

1. **False positives suppress real feedback.** The `file` action keeps the Firestore doc — humans can still see and dispatch via the dashboard. Reject is reserved for clear spam/injection. We will track FP rate via a `triage.fp_reverted` metric (anything a human re-dispatches from `triage_audit` counts as an FP).
2. **Haiku classifier latency adds ~500ms to dispatch path.** Mitigation: classifier only runs for the heuristic-flagged subset (~5-15% of traffic). Most messages dispatch in unchanged time.
3. **Adding a new Firestore collection** = new IAM. The coordinator SA already has `roles/datastore.user` project-wide; no new binding needed.
4. **Triage rules drift from package agent template.** Mitigation: `pkg-feedback.md`'s "stop at file" branch becomes redundant *but stays in place as defense in depth* — a malformed Pub/Sub message that bypasses the coordinator still gets caught template-side.
5. **Cooldown is per-contact, not per-IP.** A clever attacker rotates `contact` field values. Mitigation: also key by body hash + `from_agent` + IP hash. Edge rate limiting (Cloud Armor) catches the rest.
6. **What if the Haiku classifier itself is prompt-injectable?** Mitigation: it returns *strict JSON-mode output*, never free text passed downstream. The worst injection result is a wrong classification, never an action. JSON-mode + schema validation at the parse step.

## Out of Scope (for v1)

- Adding auth to `submit_feedback` (would defeat anonymous AI-agent onboarding)
- Triaging non-feedback messages (release-sync, internal agent traffic) — those are not anonymous-public and route through different inboxes
- ML-based abuse detection beyond Haiku one-shot — overkill for current volumes
- Replacing the existing budget cap (`task_max_cost: 150`) — keep it as a backstop

## Open Questions

1. **Should `reject` action delete the Firestore doc, or just mark `status=rejected` and let a TTL clean it up?** Recommend: TTL (14 days). Audit value > storage cost.
2. **Should the Haiku classifier be skippable for `from_agent=agent-*` (internal agent feedback)?** Recommend: yes — internal agents already pass through coordinator approval; don't double-tax them.
3. **Should we add a `dry_run=true` mode for the first week post-launch** that runs the gate but always dispatches, just logging what *would* have been blocked? Recommend: yes — single env var `AILANG_TRIAGE_DRY_RUN=1`. Lets us tune false-positive rate before enforcing.

---

## Success Metrics

- **Cost**: 1,000-message flood → ≤ $1 Anthropic spend (vs $25-50 baseline)
- **Latency**: P95 dispatch path adds ≤ 600ms (Haiku call only on the flagged ~10% subset)
- **FP rate**: < 5% of `file`'d messages are later re-dispatched by a human (tracked weekly)
- **Coverage**: 100% of dispatched messages traverse the triage gate (asserted by an integration test that bypasses it and checks an alert fires)

---

## References

- [M-AGENT-MCP-ONBOARDING](m-agent-mcp-onboarding.md) — submit_feedback ships the `package`+`auto_dispatch` args this gate enforces
- [M-PKG-FEEDBACK-LOOP](m-pkg-feedback-loop.md) — the `pkg-feedback.md` template; this doc tightens the gate *before* it
- [`internal/feedback/publisher.go`](../../../internal/feedback/publisher.go) — the publish path the triage gate sits downstream of
- [`internal/coordinator/daemon_tasks_polling.go`](../../../internal/coordinator/daemon_tasks_polling.go) — the wiring point for M4
- [`ailang-multivac/terraform/cloud_run_mcp.tf`](../../../../ailang-multivac/terraform/cloud_run_mcp.tf) L132-133 — the Cloud Armor TODO that complements this doc
- [CLAUDE.md "No Silent Fallbacks"](../../../CLAUDE.md) — informs the `reject` vs `file` distinction (no silent drops)

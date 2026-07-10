# Sprint Plan: M-FEEDBACK-TRIAGE-GATE

**Planned by**: claude-opus-4-8 (Opus 4.8)
**Design doc**: [m-feedback-triage-gate.md](m-feedback-triage-gate.md)
**Sprint ID**: M-FEEDBACK-TRIAGE-GATE
**Target version**: v0.29.x (current: v0.28.0)
**Duration**: 2 days (~780 LOC)
**Risk level**: medium (touches the cloud dispatch hot path; no live-API in tests; naming-collision hazard)

---

## Goal

Insert a **cost & abuse gate** between the coordinator's cloud Pub/Sub pickup
(`pollAndProcessTasksCloud`) and `CreateTask`, so a flood of anonymous
`submit_feedback` submissions cannot fan out to Sonnet-driven agents. Defense
in depth on top of the already-shipped Cloud Armor per-IP throttle
(M-MCP-EDGE-THROTTLE) and the `task_max_cost` backstop.

The gate is **deterministic-first**: cheap CPU rules + a per-contact sliding
cooldown reject the bulk of abuse with zero LLM cost; only a heuristic-flagged
minority reaches a one-shot Haiku JSON classifier.

---

## CRITICAL: Naming Disambiguation (READ FIRST)

There is an **existing, shipped, unrelated feature** that must NOT be entangled:

| Shipped (do NOT touch/extend) | This sprint (NEW) |
|---|---|
| `internal/coordinator/triage_router.go` (M-MSG-TRIAGE-ROUTER) | `internal/feedbackgate/` (new package) |
| config key `coordinator.triage` / `TriageConfig` | config key `coordinator.feedback_gate` / `FeedbackGateConfig` |
| type `coordinator.Decision` (Hold/Promote/Drop) | type `feedbackgate.Verdict` (Dispatch/File/Reject) |
| `classify(msg, cfg) Decision` — local intake→inbox promotion | `feedbackgate.Decide(ctx, msg, cfg) (Verdict, error)` — cloud dispatch gate |
| `TriageRouter` (Start/Stop ticker) | no long-running loop; called inline in the poll |

The design doc uses the names `internal/triage/`, `triage.Decide`, `Decision`,
`triage.Config`, and `AILANG_TRIAGE_*`. **We deliberately rename all of them**
to avoid the collision above. The executor MUST use the right-hand-column names
throughout. Env vars become `AILANG_FEEDBACK_GATE_MODE` and
`AILANG_FEEDBACK_GATE_DRY_RUN`. Metrics/audit inbox stays `triage-audit` ONLY if
free of collision — use `feedback-gate-audit` to be safe.

---

## Doc-vs-Reality Discrepancies (planned against reality)

Verified live against the code today. The executor plans against **reality**, not the doc's prose.

1. **`Decide` operates on `coordinator.Message`, NOT `messaging.Message`.**
   The doc's signature `Decide(ctx, msg *messaging.Message, cfg)` is wrong for
   the M4 wiring point. `pollAndProcessTasksCloud` iterates
   `[]*coordinator.Message` (`internal/coordinator/watcher.go`). To keep
   `internal/feedbackgate` importable without a coordinator import cycle, `Decide`
   takes a **narrow input struct** the coordinator populates from its `Message`
   (see M1). Do NOT import `internal/coordinator` from `internal/feedbackgate`.

2. **Field-name mismatch.** `coordinator.Message` exposes:
   - `Type` = category (the doc calls it `msg.category`)
   - `Content` = body (the doc calls it `msg.body`)
   - `From` = sender (the doc's `from_agent`)
   - `Inbox` = routing target (the doc's inbox)
   - `Source` = Pub/Sub topic ("cascade" vs public)
   There is **no `Contact` field and no IP/`X-Forwarded-For` field** on
   `coordinator.Message`. The publisher (`internal/feedback/publisher.go`) folds
   `contact:` into the message *body text* (`formatBody`), and the per-IP data
   is consumed at the MCP edge (`feedback_tool.go`) and NOT propagated onto the
   Pub/Sub message. **Consequence:** the cooldown key in M2 is
   `From + bodyHash + category` (+ a best-effort `contact:` line parsed from the
   body). IP-hash keying from the doc is **dropped** — the IP never reaches this
   layer. Record this as an accepted scope cut; edge throttle already covers IP.

3. **`auto:` prefix is carried on the Pub/Sub *category attribute*, not a
   boolean.** `publisher.go` sets `Category = "auto:" + category` when
   `AutoDispatch`. So the M1 rule "`auto_dispatch=false` → file" is really
   "`msg.Type` (category) does NOT start with `auto:` → file". The gate reads the
   `auto:` prefix off `Type`. There is no `auto_dispatch` bool on the wire.

4. **`internal/ai` provides JSON mode natively.** `ai.Request` has
   `ResponseFormat: "json"` + `JSONSchema` (verified in
   `internal/ai/provider.go`). The classifier (M3) uses `Provider.Generate` with
   these set — no hand-rolled JSON coaxing. The `ai.Provider` interface is
   `Generate(ctx, *ai.Request) (*ai.Response, error)`; inject it so tests use a
   fake (pattern: `stubProvider` in `internal/ai/handler_routing_test.go`).

5. **M5 terraform lives in a SIBLING REPO (`ailang-multivac`).** Out of scope
   for this repo. The in-repo M5 deliverable is the budget-cap *logic + config*
   only. The terraform alert wiring (`security.tf` 80%-budget alert) is a
   **documented follow-up handoff**, not a milestone deliverable here.

6. **Dashboard `Triage` panel + `--show-triage` CLI flag (doc's M4 tail /
   acceptance criterion).** The acceptance criterion mentions a dashboard panel.
   That is UI scope creep for a 2-day backend sprint → **cut to follow-up**. The
   audit messages ARE emitted (so the data exists); rendering them is a separate
   UI task. The in-scope, testable substitute: `feedback-gate-audit` inbox
   messages are queryable via existing `ailang messages list --inbox
   feedback-gate-audit`.

---

## Adopted Decisions (from doc Open Questions — all recommendations accepted)

1. **`reject` action does NOT delete the Firestore doc.** It marks
   `status=rejected` and relies on TTL (14 days) for cleanup. Audit value >
   storage cost. (The M4 `deleteFirestoreDoc` path in the doc is replaced by a
   status-mark; no destructive delete.)
2. **Haiku classifier is skipped for internal `agent-*` senders.** If
   `msg.From` matches `agent-*` (or the known internal allowlist), the classifier
   stage is bypassed — internal agents already pass coordinator approval; don't
   double-tax them.
3. **Dry-run mode via `AILANG_FEEDBACK_GATE_DRY_RUN=1`.** Runs the full gate,
   logs the verdict it *would* have applied, but always dispatches. For
   false-positive tuning in the first week post-launch.

Plus the operator kill-switch: `AILANG_FEEDBACK_GATE_MODE=off|file-only|full`
(off = pass-through; file-only = deterministic rules + cooldown, classifier
disabled; full = everything). Env overrides config.

---

## Velocity Analysis

- Recent 7d: 309 files, ~12.7k insertions (skewed by merges/mission infra).
- Comparable recent backend sprint **M-MSG-TRIAGE-ROUTER**: 590 LOC across 4
  milestones, completed in 1 day of focused work (all `passes: true` same day).
  That is the closest analog (same subsystem, same test patterns).
- This sprint is ~780 LOC / 6 milestones. At the M-MSG-TRIAGE-ROUTER pace
  (~150 LOC/day sustainable with tests + lint green) that is **~2 days** — matches
  the doc estimate. No buffer cut needed; the classifier (M3) and cooldown (M2)
  carry the risk.

---

## Milestones

### M1 — Deterministic pre-filter (`internal/feedbackgate/`) — ~150 LOC, ~3h

**Files**: `internal/feedbackgate/decide.go`, `internal/feedbackgate/rules.go`,
`internal/feedbackgate/decide_test.go`

Create the package with a **coordinator-free input struct** and the entry point:

```go
package feedbackgate

type Verdict struct {
    Action string  // "dispatch" | "file" | "reject"
    Reason string  // structured code
    Cost   float64 // estimated USD if dispatched
}

// Input is what the coordinator populates from its Message (no coordinator import).
type Input struct {
    ID       string
    Category string // coordinator.Message.Type
    Body     string // coordinator.Message.Content
    From     string // coordinator.Message.From
    Inbox    string // coordinator.Message.Inbox
    Source   string // coordinator.Message.Source
}

func Decide(ctx context.Context, in Input, cfg Config) (Verdict, error)
```

Deterministic rules (no LLM, <1ms, pure — table-testable):

| Rule | Action | Reason code |
|------|--------|-------------|
| category has no `auto:` prefix | `file` | `not_authorized_for_dispatch` |
| Body length > 8KB after trim | `reject` | `body_too_large` |
| Body matches spam regex (URLs>5, base64 blob>1KB) | `reject` | `spam_pattern` |
| `From` not in allowlist (`mcp-public`, `agent-*`) | `reject` | `untrusted_source` |
| Inbox not `pkg:*` or known internal | `reject` | `unknown_inbox` |
| Category (stripped of `auto:`) not in {bug,feature,docs,limitation} | `file` | `unknown_category` |

Rules run in order; first match wins. `file` = suppress dispatch, keep doc;
`reject` = suppress + mark-rejected (M4). This milestone is pure logic only —
no Firestore, no cooldown, no classifier.

**Acceptance criteria**:
- Package `internal/feedbackgate` compiles with NO import of `internal/coordinator`.
- `Decide` is pure at M1 (cooldown/classifier are nil-injected, no-op).
- Each rule has a passing + failing table test case; `go test ./internal/feedbackgate/...` green with no credentials.
- `Verdict`/`Config` names distinct from `coordinator.Decision`/`TriageConfig` (grep-verified).

### M2 — Per-contact sliding cooldown (Firestore-backed, faked in tests) — ~120 LOC, ~2h

**Files**: `internal/feedbackgate/cooldown.go`, `internal/feedbackgate/cooldown_test.go`

A narrow store interface (like `triageStore`) so tests use an in-memory fake
(pattern from `triage_router_test.go::fakeTriageStore`), NOT a live Firestore:

```go
type CooldownStore interface {
    // Increment records a dispatch attempt for key at now and returns the
    // count within the trailing window; storage-backed impl uses the
    // triage_contacts collection with TTL.
    Increment(ctx context.Context, key string, now time.Time) (hourCount, dayCount int, err error)
}
```

Key = hash of `From + "|" + category + "|" + bodyHash` plus a best-effort
`contact:` line parsed out of the body (see discrepancy #2 — no IP available).
Limit: **3 dispatch/contact/hour, 10/day** → over-limit → `file` with
`reason=contact_cooldown`. Sliding window (not fixed-hour) to defeat
stroke-of-the-hour bursts. TTL 7 days on the collection (doc detail retained).

An in-memory `CooldownStore` fake is the default for unit tests; the
Firestore-backed impl is a thin adapter constructed only in the coordinator
wiring path (M4) behind cloud mode.

**Acceptance criteria**:
- Cooldown logic tested entirely against an in-memory fake store (no GCP).
- Sliding-window boundary tests: 3rd within hour dispatches, 4th files; 10th within day dispatches, 11th files; window expiry re-allows.
- `Decide` composes M1 rules THEN cooldown (cooldown only consulted when M1 says dispatch).

### M3 — Haiku JSON classifier via `internal/ai` (injected provider) — ~180 LOC, ~3h

**Files**: `internal/feedbackgate/classifier.go`, `prompts/feedback_gate_classifier.md`,
`internal/feedbackgate/classifier_test.go`

Last-resort pre-screen for messages that pass M1 + M2 but trip a heuristic
(code block > 200 lines, or category=bug with body > 4KB and no snippet, or an
`auto:` message from a non-`agent-*` sender). **Skipped entirely for `agent-*`
senders** (adopted decision #2).

Uses `ai.Provider` injected into the classifier (fake in tests — mirror
`stubProvider` in `internal/ai/handler_routing_test.go`). Call `Generate` with
`Model: "claude-haiku-4-5"` (or config), `ResponseFormat: "json"`, and a
`JSONSchema` for:

```json
{"is_genuine_feedback": bool, "is_prompt_injection": bool,
 "best_category": "bug|feature|docs|limitation|spam",
 "estimated_dispatch_value": "high|medium|low|none", "reasoning": "..."}
```

Decision matrix → `reject` on injection, `file` on none/not-genuine/category
mismatch, else `dispatch`. Parse defensively: any JSON parse/schema failure ⇒
`file` (fail closed, never dispatch on a malformed classifier result). Prompt is
checked in with a content-hash version field for replay.

**NO live API calls in tests** — the injected fake provider returns canned
`ai.Response.Text` JSON. Prompt file is validated by a hash test, not an API call.

**Acceptance criteria**:
- Classifier takes an `ai.Provider`; all tests use a fake, zero network.
- Matrix fully table-tested incl. malformed-JSON → `file` (fail-closed).
- `agent-*` sender path asserts the provider is never called (bypass verified via a counting fake).
- Prompt `prompts/feedback_gate_classifier.md` exists with a version/hash field.

### M4 — Wire the gate into the coordinator cloud path — ~150 LOC, ~3h

**Files**: `internal/coordinator/daemon_tasks_polling.go` (edit),
`internal/coordinator/feedback_gate_audit.go` (new),
`internal/coordinator/feedback_gate_wiring.go` (new; config→feedbackgate glue),
config plumbing (`coordinator.feedback_gate` block; document the cloud config).

In `pollAndProcessTasksCloud`, between the per-message loop's start and
`CreateTask`, build a `feedbackgate.Input` from `coordinator.Message`, call
`feedbackgate.Decide`, and branch:

```go
in := feedbackgate.Input{ID: msg.ID, Category: msg.Type, Body: msg.Content,
    From: msg.From, Inbox: inbox, Source: msg.Source}
v, err := d.feedbackGate.Decide(d.ctx, in, d.feedbackGateCfg)
// err → log + emit audit + skip CreateTask (fail closed: do NOT dispatch on gate error)
switch v.Action {
case "dispatch": // existing CreateTask path
case "file":     d.emitGateAudit(msg, v); continue // ack, no task
case "reject":   d.emitGateAudit(msg, v); d.markFeedbackRejected(msg.ID); continue
}
```

- Gate is **nil-safe / opt-in**: when `feedback_gate.enabled=false` (default) or
  gate is nil, the branch is skipped entirely (zero behavior change) — mirror the
  `TriageRouter` opt-in convention.
- Only wire into the **cloud** path (`pollAndProcessTasksCloud`); local SQLite
  polling is out of scope (public feedback is cloud-only per `publisher.go`).
- `emitGateAudit` writes a structured `feedback-gate-audit` inbox message
  (reason, verdict, msg id, cost). No silent drops (CLAUDE.md rule 2).
- `markFeedbackRejected` sets `status=rejected` (adopted decision #1 — TTL, no delete).
- Respect `AILANG_FEEDBACK_GATE_MODE` and `..._DRY_RUN` (dry-run → always dispatch, still audit the would-be verdict).

**Acceptance criteria**:
- Gate disabled by default; existing coordinator tests unchanged and green.
- Unit test drives `pollAndProcessTasksCloud`-level logic with a fake cloud adapter + fake gate asserting: dispatch→CreateTask called; file/reject→CreateTask NOT called + audit emitted.
- Gate error → fail closed (no CreateTask), audit emitted.
- `make lint && go test ./internal/coordinator/... ./internal/feedbackgate/...` green, no credentials.

### M5 — Daily classifier budget cap (logic + config only) — ~60 LOC, ~1h

**Files**: `internal/feedbackgate/budget.go`, `internal/feedbackgate/budget_test.go`

Per-day token/$ budget for the classifier (default $5/day). On exceed, the
classifier stage short-circuits to `file` (never `dispatch`) — worst case the
inbox fills and a human triages; never a Sonnet flood. Counter tracked via the
same `CooldownStore`-style interface (daily key), faked in tests.

**Terraform 80%-budget alert in `ailang-multivac/terraform/security.tf` is OUT
OF SCOPE** (sibling repo) → documented follow-up handoff (see below).

**Acceptance criteria**:
- Budget check tested against fake store: under budget → classifier runs; at/over → forced `file`.
- Default $5/day; overridable via `coordinator.feedback_gate.daily_budget_usd`.
- No sibling-repo files touched.

### M6 — Tests + in-repo flood simulation — ~120 LOC, ~3h

**Files**: `internal/feedbackgate/integration_test.go`,
`scripts/security/feedback_flood_drill.sh`

- Integration test (Go, no cloud): feed 100 synthetic `Input`s through a fully
  assembled gate (in-memory cooldown + fake classifier provider) and assert:
  ≤ 3 per contact reach `dispatch`; every non-dispatch produces an audit reason;
  no input is silently dropped.
- `scripts/security/feedback_flood_drill.sh`: a **dry-runnable** local harness
  that generates N synthetic submissions and reports the verdict histogram +
  simulated spend. It targets the in-repo gate/test env — **no cloud creds, no
  live Anthropic/Sonnet calls, no Ollama** (validation is offline; the real
  cloud flood drill is a separate ops task).

**Acceptance criteria**:
- `go test ./internal/feedbackgate/...` passes with no credentials.
- Flood-sim script runs offline and prints a verdict histogram + a spend line.
- CHANGELOG updated; design doc M-checkboxes flipped; naming-disambiguation note recorded in CHANGELOG.

---

## Dependency Graph

```
M1 (rules) ──► M2 (cooldown) ──► M3 (classifier) ──► M4 (wiring) ──► M6 (tests)
                                       └──► M5 (budget) ──┘
```

- M1 → M2 → M3 are sequential (each composes the prior into `Decide`).
- M5 depends on M3 (gates the classifier).
- M4 depends on M3+M5 (wires the complete gate).
- M6 depends on M4.

## Day-by-Day

**Day 1**: M1 (3h) + M2 (2h) + start M3 (3h). End-of-day: `Decide` composes
rules + cooldown + classifier against fakes, all unit-green.

**Day 2**: finish M3, M5 (1h) + M4 (3h) + M6 (3h). End-of-day: gate wired into
cloud path (off by default), integration test + flood-sim green, docs updated.

---

## Testing Strategy (no cloud credentials anywhere)

- **Pure logic (M1)**: table-driven unit tests.
- **Firestore-backed cooldown/budget (M2/M5)**: narrow `CooldownStore` interface
  + in-memory fake — the exact pattern `triage_router_test.go` uses for
  `triageStore`/`fakeTriageStore`. No `firestore.NewClient` in tests.
- **Classifier (M3)**: injected `ai.Provider` fake returning canned JSON
  (pattern: `stubProvider` in `internal/ai/handler_routing_test.go`). Zero network.
- **Wiring (M4)**: fake cloud inbox adapter + fake gate; assert CreateTask
  called/not-called + audit emitted.
- **NO Ollama, NO live Anthropic, NO GPU** anywhere in this sprint.

## Risks & Mitigations

1. **Hot-path regression in `pollAndProcessTasksCloud`.** Mitigation: gate is
   nil-safe + opt-in (`enabled=false` default); existing coordinator tests must
   stay green untouched.
2. **Naming entanglement with M-MSG-TRIAGE-ROUTER.** Mitigation: new package
   `feedbackgate`, `Verdict`/`FeedbackGateConfig`/`coordinator.feedback_gate`;
   grep-assert no reuse of `Decision`/`TriageConfig`/`coordinator.triage`.
3. **Import cycle** (`feedbackgate` ↔ `coordinator`). Mitigation: `feedbackgate`
   defines its own `Input` struct; coordinator maps its `Message` onto it.
4. **Classifier fails open.** Mitigation: any parse/schema/error path ⇒ `file`,
   never `dispatch` (fail closed). Same for gate-level errors in M4.
5. **Doc drift** (fields, IP keying, terraform repo). Mitigation: documented in
   "Doc-vs-Reality" above; plan targets reality.

## Out of Scope (follow-up handoffs, NOT milestones)

- **Terraform 80%-budget alert** in `ailang-multivac/terraform/security.tf`
  (sibling repo) — hand off to a multivac terraform task.
- **Dashboard `Triage`/`Feedback-Gate` panel** + `ailang messages list
  --show-triage` flag — UI task; the audit data already lands in
  `feedback-gate-audit`.
- **Real cloud flood drill** (1,000 msgs against the live test env with actual
  spend measurement) — an ops task; M6 ships the offline simulator only.
- Auth for `submit_feedback`, prompt-injection hardening of Sonnet, ML abuse
  detection — explicitly out of scope per the design doc.

## Success Metrics (in-repo, testable this sprint)

- `go test ./internal/feedbackgate/... ./internal/coordinator/...` green, no creds.
- 100-message integration test: ≤ 3/contact dispatch; 100% audited; no silent drops.
- Gate off by default → zero behavior change (existing coordinator suite untouched).
- `make lint` clean; CHANGELOG + design doc updated; naming-collision note recorded.

---

## References (verified live)

- `internal/coordinator/daemon_tasks_polling.go` — `pollAndProcessTasksCloud` (M4 wiring point)
- `internal/coordinator/watcher.go` — `coordinator.Message` shape (fields the gate reads)
- `internal/coordinator/triage_router.go` + `triage_router_test.go` — naming-collision source + store-fake pattern to reuse
- `internal/feedback/publisher.go` — publish path; `auto:` category prefix; `contact:` folded into body; no IP on the wire
- `internal/apiserver/feedback_tool.go` — MCP edge + shipped per-IP throttle (IP consumed here, not propagated)
- `internal/ai/provider.go` — `Provider.Generate`, `Request.ResponseFormat="json"` + `JSONSchema`
- `internal/ai/handler_routing_test.go` — `stubProvider` fake pattern for classifier tests

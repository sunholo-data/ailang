# M-FEEDBACK-GATE-CLOUD-ADAPTER: Firestore stores + classifier wiring for the feedback gate

**Status**: Planned
**Target**: v0.29.0
**Priority**: P0 (operational completion of M-FEEDBACK-TRIAGE-GATE)
**Estimated**: 0.5–1 day
**Dependencies**: M-FEEDBACK-TRIAGE-GATE (merged 40f1cdc3f — gate logic, interfaces, coordinator wiring)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is Go infrastructure (coordinator/cloud wiring), not a language change — most axioms are
neutral. Scored for completeness.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics touched; gate verdicts already depend on wall-clock windows by design (M2) |
| A2: Replayability | 0 | Verdicts are audited (feedback-gate-audit) exactly as before; adapters add no hidden state to traces |
| A3: Effect Legibility | 0 | No AILANG effects involved; Go-side Firestore I/O is explicit in the adapter package |
| A4: Explicit Authority | +1 | Dispatch authority is now actually bounded in production (cooldown + budget enforced, not no-op) |
| A5: Bounded Verification | 0 | No impact |
| A6: Safe Concurrency | +1 | Firestore transactions make concurrent-coordinator increments safe (in-memory fake was single-process only) |
| A7: Machines First | 0 | No impact |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | The $5/day classifier budget cap becomes live instead of dead config |
| A10: Composability | 0 | Drops into existing narrow interfaces unchanged |
| A11: Structured Failure | +1 | Missing-API-key case fails CLOSED with a typed verdict + loud startup log (no silent fallback) |
| A12: System Boundary | 0 | Firestore boundary already crossed by coordinator cloud mode |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

M-FEEDBACK-TRIAGE-GATE (merged 40f1cdc3f, eval 93/100) shipped the full gate pipeline —
deterministic rules → per-contact cooldown → Haiku classifier → daily budget — but the
cooldown, classifier, and budget stages are **injected dependencies that nothing constructs
in production**. `internal/coordinator/daemon_tasks_init.go:52-65` wires the gate and leaves
`cfg.Cooldown`/`cfg.Classifier` nil, so those stages are no-ops
(`internal/feedbackgate/config.go:73-79`). The executor recorded this as a scope deviation;
this doc is the queued completion.

**Current State:**
- Production (cloud) protection is **rules-only**: sender allowlist, body size, category
  checks. Verified by reading `daemon_tasks_init.go` — the comment at :56-58 explicitly says
  "the Firestore-backed cooldown/classifier stores are attached by the cloud adapter path
  (follow-up)".
- The P0's core threat model — a flood of *rule-passing* submissions fanning out to Sonnet
  (1,000 msgs ≈ $30) — is NOT mitigated: no per-contact rate cap, no classifier, no spend cap.
- The offline flood drill proved the assembled gate cuts 1,000 → 30 dispatches ($0.90), but
  only with fakes injected.

**Impact:**
- Anyone hitting the public `submit_feedback` endpoint with well-formed submissions can still
  trigger unbounded Sonnet dispatches. The P0 is not operationally closed until these stages run
  against real stores.

## Goals

**Primary Goal:** Construct the Firestore-backed `CooldownStore`/`BudgetStore` and the real
Anthropic classifier in the coordinator's cloud wiring so every merged gate stage is live in
production.

**Success Metrics:**
- With `AILANG_STORAGE=gcp|hybrid`, the daemon startup log shows cooldown + budget + classifier
  attached (or a LOUD warning naming exactly what is missing and why).
- Cooldown enforced across process restarts and across concurrent coordinators (Firestore
  transaction, not process memory).
- Classifier daily budget cap enforced against a persistent day counter.
- Rollout is dry-run-first: week 1 runs `AILANG_FEEDBACK_GATE_DRY_RUN=1`, verdicts audited but
  not enforced.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Adapters live in `internal/storage/firestore` (implement `feedbackgate` interfaces there) | Preserves layering: coordinator stays Firestore-free (SetStores pattern); no import cycle (`feedbackgate` imports only `internal/ai`) | agent (validated in this doc) | design | med |
| Wiring point: CLI entry (`cmd/ailang/coordinator_lifecycle.go`) constructs deps, hands to daemon via new `SetFeedbackGateDeps(...)` — mirrors `SetStores`/`SetCloudDispatcher` | Constructing inside the daemon would force coordinator→firestore import, breaking the existing decoupling | agent (validated in this doc) | design | med |
| Missing `ANTHROPIC_API_KEY` in cloud mode → classifier constructed with nil provider (**fail closed**: heuristic-flagged messages are FILED) + loud startup log | The alternative (skip classifier → flagged messages dispatch) silently reopens the flood hole; fail-closed is the merged gate's own posture (`applyClassifier` nil-provider branch) | human — **this doc decides; flag in review** | design | low |
| Sliding-window storage: per-key doc holding a timestamp array, transactionally trimmed to 24h, saturation-capped | True sliding window (per M2 design: no stroke-of-the-hour reset) with bounded doc size under flood | agent | compile | low |
| TTL cleanup: adapters write `expires_at`; the Firestore TTL *policy* is sibling-repo terraform (documented handoff, NOT in this sprint) | Same handoff pattern as the triage-gate sprint (no sibling-repo files touched); without the policy, stale docs are tiny and harmless | agent | runtime | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Adapter package location: `internal/storage/firestore` (structural interfaces + compile-time assertion `var _ feedbackgate.CooldownStore = ...`)
- [x] Wiring point: CLI entry point, new `Daemon.SetFeedbackGateDeps` (attached to cfg in `initTaskProcessing` only when the gate is enabled)
- [x] No-API-key behavior: fail closed with loud log (decided here; headless iteration — Mark can veto on the #329 report before enablement, which is a separate ops step anyway)

## Solution Design

### Overview

Three small pieces, all dropping into interfaces that already exist and are already consumed:

1. **`FeedbackGateCooldownStore`** — Firestore impl of
   `feedbackgate.CooldownStore.Increment(ctx, key, now) (hourCount, dayCount, error)`
   (`internal/feedbackgate/cooldown.go:17-23`).
2. **`FeedbackGateBudgetStore`** — Firestore impl of
   `feedbackgate.BudgetStore.IncrementDaily(ctx, dayKey, now) (count, error)`
   (`internal/feedbackgate/budget.go:17-21`).
3. **Wiring** — construct both + `feedbackgate.NewClassifier(anthropic.NewClient(key),
   feedbackgate.DefaultPrompt(), feedbackgate.NewBudget(budgetStore))` in
   `cmd/ailang/coordinator_lifecycle.go` when `storage.GetMode() != local`; pass to the daemon;
   `initTaskProcessing` attaches them to the gate config when `feedback_gate.enabled`.

### Architecture

**Cooldown adapter** (collection `feedback_gate_cooldown`):
- Doc ID: `hex(sha256(key))[:32]` — keys contain `|`-joined arbitrary text (contact lines);
  hashing makes them safe/uniform Firestore IDs. Store the raw key in a field for debugging.
- Doc shape: `{key: string, attempts: []timestamp, expires_at: timestamp}`.
- `Increment` runs a Firestore transaction: read doc → drop attempts older than 24h → append
  `now` → count within 1h and 24h → write back with `expires_at = now + 7d` (matches the
  original design's 7-day TTL intent).
- **Saturation cap**: if the trimmed array already holds ≥ 64 attempts, do NOT append; return
  counts saturated at len(attempts). `Decide` only compares `> MaxDispatchPerHour/Day`
  (defaults 3/10), so precision above 64 is meaningless and this bounds doc size under flood.
- Fail-open vs fail-closed is NOT this adapter's concern: `applyCooldown` already propagates
  store errors and the M4 wiring already fails closed on gate error
  (`feedback_gate_wiring.go:103-112`). The adapter just returns errors honestly.

**Budget adapter** (collection `feedback_gate_budget`):
- Doc ID: the `dayKey` (`YYYY-MM-DD`, already UTC-normalized by `feedbackgate.dayKey`).
- Doc shape: `{count: int, expires_at: timestamp}` (`expires_at = now + 3d`).
- `IncrementDaily`: transaction read → count+1 → write; return new count.

**Wiring** (`cmd/ailang/coordinator_lifecycle.go`, after the existing `SetStores` block):
```go
if storageMode != storage.ModeLocal {
    fsClient, err := fsstore.NewClient(ctx)           // existing constructor, ADC + AILANG_CLOUD_PROJECT
    // error → return (no silent fallback: cloud mode without Firestore is already fatal above)
    cooldown := fsstore.NewFeedbackGateCooldownStore(fsClient)
    budget := feedbackgate.NewBudget(fsstore.NewFeedbackGateBudgetStore(fsClient))
    var provider ai.Provider
    if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
        provider = anthropic.NewClient(key)
    } else {
        // LOUD: classifier will FAIL CLOSED (flagged messages filed, never dispatched)
        fmt.Printf("  ⚠ ANTHROPIC_API_KEY not set: feedback-gate classifier disabled (fail-closed)\n")
    }
    classifier := feedbackgate.NewClassifier(provider, feedbackgate.DefaultPrompt(), budget)
    daemon.SetFeedbackGateDeps(cooldown, classifier)
}
```
- `Daemon.SetFeedbackGateDeps(cooldown feedbackgate.CooldownStore, classifier *feedbackgate.Classifier)`
  stores the deps on the daemon; `initTaskProcessing` copies them into
  `coordConfig.FeedbackGate.Cooldown/.Classifier` inside the existing `if ... Enabled` block,
  and extends the startup log line to say which stages are live
  (e.g. `mode=full, dry_run=true, cooldown=firestore, classifier=anthropic, budget=firestore`).
- Note the client construction is unconditional on gate-enabled (config not yet loaded at CLI
  time) but cheap; deps are simply unused when the gate is off. If constructing the Firestore
  client twice (backends already made one) is deemed wasteful, the agent may thread the
  existing client through — deferred decision below.

### Implementation Plan

**M1: Firestore adapters + pure window math** (~3h)
- [ ] `internal/storage/firestore/feedbackgate_stores.go`: both adapters; sliding-window
      trim/count/saturate as a **pure function** (`trimAndCount(attempts, now) (kept, hour, day)`)
      per this package's no-emulator test convention
- [ ] Compile-time interface assertions against `feedbackgate.CooldownStore`/`BudgetStore`
- [ ] `feedbackgate_stores_test.go`: table tests for the pure window math (boundary at exactly
      1h/24h, saturation cap, empty doc, cross-day)

**M2: Wiring** (~2h)
- [ ] `Daemon.SetFeedbackGateDeps` + attach in `initTaskProcessing` + startup log naming live stages
- [ ] `cmd/ailang/coordinator_lifecycle.go` construction block (incl. fail-closed no-key path)
- [ ] Wiring test in `internal/coordinator` (fake deps: assert they land on the cfg the decider sees;
      assert nil-deps ⇒ unchanged rules-only behavior)

**M3: Rollout docs + drill hook** (~1h)
- [ ] `docs/docs/guides/coordinator.md` (or the feedback-gate section's home): enablement
      runbook — week 1 `AILANG_FEEDBACK_GATE_DRY_RUN=1`, watch `feedback-gate-audit` inbox,
      then flip; env reference table
- [ ] Sibling-repo handoff note (terraform): TTL policy on `expires_at` for the two new
      collections + `ANTHROPIC_API_KEY` secret on the coordinator service — appended to the
      implemented triage-gate doc's follow-up section
- [ ] CHANGELOG entry

### Files to Modify/Create

**New files:**
- `internal/storage/firestore/feedbackgate_stores.go` — both adapters + pure window math (~150 LOC)
- `internal/storage/firestore/feedbackgate_stores_test.go` — window-math tables (~120 LOC)

**Modified files:**
- `internal/coordinator/daemon.go` — deps fields + `SetFeedbackGateDeps` (~20 LOC)
- `internal/coordinator/daemon_tasks_init.go` — attach deps, richer startup log (~15 LOC)
- `internal/coordinator/feedback_gate_wiring_test.go` — deps-attachment test (~40 LOC)
- `cmd/ailang/coordinator_lifecycle.go` — construction block (~25 LOC)
- `docs/docs/guides/coordinator.md`, `CHANGELOG.md`,
  `design_docs/implemented/v0_29_0/m-feedback-triage-gate.md` (follow-up section) — docs

## Examples

### Example 1: Startup log, cloud mode, fully wired

**Before:**
```
Feedback gate enabled (mode=full, dry_run=true)
```
(cooldown/classifier/budget silently no-op)

**After:**
```
Feedback gate enabled (mode=full, dry_run=true, cooldown=firestore, classifier=anthropic, budget=firestore)
```

### Example 2: Startup log, cloud mode, key missing

**After:**
```
⚠ ANTHROPIC_API_KEY not set: feedback-gate classifier disabled (fail-closed)
Feedback gate enabled (mode=full, dry_run=true, cooldown=firestore, classifier=fail-closed, budget=firestore)
```
Heuristic-flagged messages are filed (auditable), never dispatched, never silently passed.

## Success Criteria

- [ ] Both adapters satisfy the `feedbackgate` interfaces with compile-time assertions
- [ ] Pure window math: hour/day boundaries exact, saturation cap bounds doc growth (table tests)
- [ ] Daemon startup log names which stages are live; nil deps ⇒ behavior identical to today
- [ ] No-key cloud startup fails closed with the loud warning (test on the construction helper)
- [ ] `make test`, `make lint` green; no sibling-repo files touched
- [ ] Rollout runbook documents DRY_RUN-first enablement
- [ ] CHANGELOG updated

## Testing Strategy

**Unit tests:**
- Pure window math (this package's convention — no Firestore emulator in the repo; the
  transaction wrapper stays thin and untested locally, matching `CoordinatorStore` precedent)
- Budget day-count arithmetic via the pure increment path

**Integration tests:**
- Coordinator wiring test with fakes: `SetFeedbackGateDeps` → deps visible to the decider;
  absent deps → rules-only unchanged (guard the call-site, not just the helper — the
  M-ENV-FORWARD lesson)

**Manual testing:**
- Live Firestore check is an **ops step at enablement time**, not a merge gate: run the
  documented dry-run week against the dev project, inspect `feedback_gate_cooldown` docs and
  `feedback-gate-audit` messages

## Deferred Decisions

The following are intentionally left open for the implementer:

- Reuse the existing `storage.NewBackends` Firestore client vs constructing a second
  `fsstore.NewClient` — agent may choose (second client is correct-but-wasteful; threading is
  cleaner if it doesn't contort `NewBackends`' return shape)
- Saturation cap constant (64 suggested) and whether to log when saturated — agent may choose
- Exact startup-log format — agent may choose, must name all three stages

## Non-Goals

**Not attempted in this feature:**
- Terraform for TTL policies / secrets (sibling repo `ailang-multivac`) — documented handoff,
  same boundary as the triage-gate sprint
- Live cloud flood drill — separate ops task (already on the nice-list)
- Dashboard triage panel / `--show-triage` CLI — existing follow-up, unrelated to stores
- Any change to gate decision logic, thresholds, or the classifier prompt — merged and evaluated

## Timeline

Single day: M1 (~3h) → M2 (~2h) → M3 (~1h), plus test/lint/review slack. **Total: ~6-8h.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Firestore transaction contention on a hot cooldown key during a flood | Low | Per-key docs; saturation cap keeps docs small; contention degrades to retries/errors → gate fails closed (never a dispatch flood) |
| Classifier live-fires unexpectedly on enablement | Med | Off by default (unchanged); runbook mandates DRY_RUN week 1; `mode=file-only` and env kill-switch already shipped |
| Coordinator SA lacks Firestore perms for new collections | Low | SA already has `roles/datastore.user` project-wide (triage-gate doc, risk #3) — collection-level IAM not used |
| `ANTHROPIC_API_KEY` absent on the deployed service | Med | Fail-closed + loud log (this doc's decision); secret addition is in the terraform handoff note |

## Related Documents

- [m-feedback-triage-gate.md](../../implemented/v0_29_0/m-feedback-triage-gate.md) — the parent P0; its follow-up section is this doc's origin
- [sprint-m-feedback-triage-gate.md](../../implemented/v0_29_0/sprint-m-feedback-triage-gate.md) — recorded the scope deviation this doc completes; adopted decisions (TTL-over-delete, agent-* bypass, dry-run) carry over
- [m-cloud-infra.md](../../implemented/v0_9_0/m-cloud-infra.md) (0.35 neural) — the `SetStores`/backends layering this doc preserves

Neural search top hits (all < 0.45 → no duplicate-gate concern):
global-collaboration-hub (0.34), m-cascade-observability (0.34), m-pkg-feedback-loop (0.32).

## References

- `internal/feedbackgate/cooldown.go:17-23`, `budget.go:17-21` — the interfaces (verified in-repo 2026-07-10)
- `internal/coordinator/daemon_tasks_init.go:52-65` — the nil-deps wiring gap (verified)
- `internal/storage/firestore/client.go` — existing client constructor (ADC + `AILANG_CLOUD_PROJECT`)
- `cmd/ailang/coordinator_lifecycle.go:103-112` — the `SetStores` wiring point this mirrors

## Future Work

- Firestore-backed `HeartbeatStore` (v0.25 roadmap note in `coordinator_lifecycle.go:94-98`)
  could share the adapter file's patterns
- Live flood drill against the dev project once adapters are enabled

---

**Document created**: 2026-07-10
**Last updated**: 2026-07-10

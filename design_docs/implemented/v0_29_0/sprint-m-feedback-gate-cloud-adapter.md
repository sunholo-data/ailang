# Sprint Plan: M-FEEDBACK-GATE-CLOUD-ADAPTER

**Planned by**: claude-opus-4-8 (Opus 4.8)
**Design doc**: [m-feedback-gate-cloud-adapter.md](m-feedback-gate-cloud-adapter.md)
**Sprint ID**: M-FEEDBACK-GATE-CLOUD-ADAPTER
**Target version**: v0.29.x (current: v0.28.0)
**Duration**: 1 day (~6-8h, ~370 LOC)
**Risk level**: low-medium (no gate-logic change; Firestore adapter + CLI-wiring only; no live-API/no-emulator tests)

---

## Goal

Operationally close the P0 shipped by **M-FEEDBACK-TRIAGE-GATE** (merged 40f1cdc3f,
eval 93/100). That sprint shipped the full gate pipeline — deterministic rules →
per-contact cooldown → Haiku classifier → daily budget — but the cooldown,
classifier, and budget stages are **injected dependencies that nothing constructs
in production**. `internal/coordinator/daemon_tasks_init.go:59-65` enables the gate
and leaves `cfg.Cooldown`/`cfg.Classifier` nil, so those stages are no-ops
(`internal/feedbackgate/config.go:73-79`). Production protection is rules-only; a
flood of *rule-passing* submissions can still fan out to Sonnet.

This sprint constructs the Firestore-backed `CooldownStore`/`BudgetStore` and the
real Anthropic classifier in the coordinator's cloud wiring so every merged gate
stage runs live in production, with a **fail-closed** posture when
`ANTHROPIC_API_KEY` is absent.

---

## Doc-vs-Reality Reconciliation (planned against reality)

Every premise in the design doc was re-verified live against the code on 2026-07-10.
The design doc (written today by Fable after a repo reality-check) is **accurate** —
the interfaces, gap, and wiring points all match. Six items were checked; the
notable clarifications the executor should carry:

1. **`FeedbackGateConfig.Cooldown` and `.Classifier` are EXPORTED, directly
   assignable fields** (`config.go:75,79`: `Cooldown CooldownStore`,
   `Classifier *Classifier`). The doc's "copies them into
   `coordConfig.FeedbackGate.Cooldown/.Classifier`" is a plain field assignment on
   the pointed-to struct — no setter/reflection needed. `d.feedbackGateCfg` is
   `*FeedbackGateConfig` (a type alias for `*feedbackgate.FeedbackGateConfig`, see
   `feedback_gate_wiring.go:36`), and `coordConfig.FeedbackGate` is the same pointer,
   so attaching deps on `d.feedbackGateCfg` after `initTaskProcessing` sets it is the
   clean site. **Reconciliation:** attach on `d.feedbackGateCfg` inside the existing
   `if coordConfig.FeedbackGate != nil && coordConfig.FeedbackGate.Enabled` block
   (init.go:59), guarding for nil deps (deps only set in cloud mode).

2. **The classifier MODEL is a per-REQUEST field, not a constructor arg.**
   `applyClassifier` builds `ai.Request{Model: cfg.ClassifierModel, ...}`
   (`classifier.go:138-139`) and calls `provider.Generate`. `NewClassifier(provider,
   prompt, budget)` takes NO model. So `provider = anthropic.NewClient(key)` (a bare
   client) is correct — the configured model (`claude-haiku-4-5` default) reaches the
   provider via the request, not the client. **The wiring must NOT try to thread a
   model into the client.** This is the doc's intent but was not spelled out; flag it
   so the executor doesn't invent a model arg.

3. **`anthropic.NewClient(key)` returns `*anthropic.Client`, which satisfies the FULL
   3-method `ai.Provider` interface** (`Generate` client.go:141, `Step` step.go:41,
   `Name` client.go:347 — `ai.Provider` requires all three, provider.go:339-355). The
   classifier only calls `Generate`, but the assignment `var provider ai.Provider =
   anthropic.NewClient(key)` type-checks because the concrete type implements the
   whole interface. Confirmed — no wrapper needed.

4. **Fail-closed no-key posture already exists in the merged code.**
   `applyClassifier` has an explicit `if cl.provider == nil { return
   Verdict{Action: ActionFile, Reason: ReasonClassifierError} }` branch
   (classifier.go:134-136). So a nil-provider classifier files heuristic-flagged
   messages — exactly the doc's "fail closed" decision. The wiring's job is only to
   PASS a nil provider (loud log) when the key is absent; the gate does the rest. No
   new fail-closed code in the gate package.

5. **`Client.RunTransaction(ctx, fn func(ctx, *firestore.Transaction) error)` exists**
   (`internal/storage/firestore/client.go:65`) and is already used with the
   `tx *firestore.Transaction` read-modify-write pattern in
   `internal/storage/firestore/messaging.go:93`. The adapter mirrors that precedent —
   keep the transaction wrapper thin (per this package's no-emulator convention) and
   push all window math into a pure function.

6. **Import-cycle check PASSES.** `internal/feedbackgate`'s only non-stdlib import is
   `internal/ai` (in classifier.go:12). It does NOT import `internal/storage`,
   `internal/coordinator`, or firestore. Therefore `internal/storage/firestore →
   internal/feedbackgate` (to implement the interfaces + compile-time assertions) is a
   clean one-directional edge — no cycle. The doc's layering claim is verified.

**Naming discipline (carried from the parent sprint):** the gate lives under
`feedbackgate`/`FeedbackGateConfig`/`coordinator.feedback_gate`. Do NOT touch or
reuse `coordinator.triage`/`TriageConfig`/`coordinator.Decision`
(M-MSG-TRIAGE-ROUTER). New adapter types are `FeedbackGateCooldownStore` /
`FeedbackGateBudgetStore` under the `firestore` package. grep-assert no collision.

---

## Velocity Analysis

- Recent 7d: 358 files, ~16.1k insertions — skewed by mission infra + the merged
  parent sprint; not a per-day coding signal.
- Closest analog: the **parent M-FEEDBACK-TRIAGE-GATE** sprint — ~780 LOC / 6
  milestones, sustained ~150 LOC/day with tests + lint green, in the SAME subsystem
  using the SAME store-fake / injected-provider patterns this sprint reuses.
- This sprint is ~370 LOC / 3 milestones. At that pace, **~1 day (6-8h)** — matches
  the design doc estimate. Lower risk than the parent: no gate-decision logic, no
  new hot-path branch, pure-math + wiring only. No buffer cut needed.

---

## Milestones

### M1 — Firestore adapters + pure window math — ~150 impl + ~120 test LOC, ~3h

**Files (new):**
- `internal/storage/firestore/feedbackgate_stores.go`
- `internal/storage/firestore/feedbackgate_stores_test.go`

Implement both adapters in the `firestore` package (preserves layering; no
coordinator/firestore coupling):

**`FeedbackGateCooldownStore`** (collection `feedback_gate_cooldown`), satisfies
`feedbackgate.CooldownStore.Increment(ctx, key string, now time.Time) (hourCount,
dayCount int, err error)`:
- Doc ID: `hex(sha256(key))[:32]` — keys are `|`-joined arbitrary text (contact
  lines); hashing yields safe/uniform Firestore IDs. Store the raw `key` in a field
  for debugging.
- Doc shape: `{key string, attempts []time.Time, expires_at time.Time}`.
- `Increment` runs `client.RunTransaction`: read doc → `trimAndCount(attempts, now)`
  → append `now` → write back with `expires_at = now + 7d`.
- **Saturation cap** (constant, 64 suggested — executor may choose): if the trimmed
  array already holds ≥ cap attempts, do NOT append; return counts saturated at
  `len(attempts)`. `applyCooldown` only compares `> MaxDispatchPerHour/Day` (3/10),
  so precision above the cap is meaningless; this bounds doc size under flood.

**`FeedbackGateBudgetStore`** (collection `feedback_gate_budget`), satisfies
`feedbackgate.BudgetStore.IncrementDaily(ctx, dayKey string, now time.Time) (count
int, err error)`:
- Doc ID: `dayKey` (`YYYY-MM-DD`, already UTC-normalized by `feedbackgate.dayKey`).
- Doc shape: `{count int, expires_at time.Time}` (`expires_at = now + 3d`).
- `IncrementDaily`: transaction read → `count+1` → write → return new count.

**Pure window math** — the testable core, extracted so no emulator is needed:
```go
// trimAndCount drops attempts strictly older than 24h from now, then counts how
// many of the KEPT attempts fall within the trailing 1h and 24h windows.
func trimAndCount(attempts []time.Time, now time.Time) (kept []time.Time, hour, day int)
```
The transaction wrapper stays thin (untested locally, matching the `CoordinatorStore`
/ `messaging.go` precedent); ALL logic lives in `trimAndCount` and the budget's
plain increment.

**Compile-time interface assertions** (loud regression guard):
```go
var _ feedbackgate.CooldownStore = (*FeedbackGateCooldownStore)(nil)
var _ feedbackgate.BudgetStore   = (*FeedbackGateBudgetStore)(nil)
```

**Acceptance criteria:**
- Both adapters satisfy the `feedbackgate` interfaces with the two compile-time
  assertions present.
- `trimAndCount` table tests cover: empty attempts; attempt exactly at the 1h
  boundary (inclusive/exclusive edge documented + tested); exactly at 24h; a
  cross-day span; saturation cap (≥ cap → no append, saturated counts).
- `go test ./internal/storage/firestore/...` green with NO GCP credentials / no
  emulator (only the pure functions are exercised).
- No import of `internal/coordinator`; `internal/feedbackgate` imported only for the
  interface types + `dayKey` semantics (mirrored, not called on the hot path).

### M2 — Coordinator wiring (deps construction + attach + startup log) — ~60 impl + ~40 test LOC, ~2h

**Files (modified):**
- `internal/coordinator/daemon.go` — 2 new struct fields + `SetFeedbackGateDeps`
- `internal/coordinator/daemon_tasks_init.go` — attach deps, richer startup log
- `cmd/ailang/coordinator_lifecycle.go` — construction block (adds imports:
  `internal/storage/firestore`, `internal/ai`, `internal/ai/anthropic`,
  `internal/feedbackgate`)
- `internal/coordinator/feedback_gate_wiring_test.go` — deps-attachment test (extend)

**`daemon.go`:** add fields next to `feedbackGate`/`feedbackGateCfg` (daemon.go:108):
```go
feedbackGateCooldown   feedbackgate.CooldownStore
feedbackGateClassifier *feedbackgate.Classifier
```
and the setter (mirrors `SetStores`/`SetCloudDispatcher`, daemon.go:165-179):
```go
// SetFeedbackGateDeps installs the Firestore-backed cooldown store and the real
// classifier for the feedback gate. Constructed in the CLI entry point (cloud
// mode only) to avoid a coordinator→firestore import. No-op stages remain nil in
// local mode. Call after NewDaemon(), before Start().
func (d *Daemon) SetFeedbackGateDeps(cooldown feedbackgate.CooldownStore, classifier *feedbackgate.Classifier) {
    d.feedbackGateCooldown = cooldown
    d.feedbackGateClassifier = classifier
}
```

**`daemon_tasks_init.go`:** inside the existing `if coordConfig.FeedbackGate != nil
&& coordConfig.FeedbackGate.Enabled` block (init.go:59), AFTER `d.feedbackGateCfg =
coordConfig.FeedbackGate`, attach the deps onto the config the decider reads and
extend the startup log to name the live stages:
```go
if d.feedbackGateCooldown != nil {
    d.feedbackGateCfg.Cooldown = d.feedbackGateCooldown
}
if d.feedbackGateClassifier != nil {
    d.feedbackGateCfg.Classifier = d.feedbackGateClassifier
}
// startup log: mode=..., dry_run=..., cooldown=<firestore|none>,
//              classifier=<anthropic|fail-closed|none>, budget=<firestore|none>
```
(exact format is the executor's choice per the doc's deferred decision, but MUST
name all three stages).

**`cmd/ailang/coordinator_lifecycle.go`:** after the existing `SetStores` block
(lifecycle.go:103-112, `if storageMode != storage.ModeLocal`), construct the deps.
Reuse or construct a Firestore client (deferred decision — a second
`firestore.NewClient(ctx)` is correct-but-wasteful; threading the existing
`storage.NewBackends` client is cleaner if it doesn't contort the return shape; the
executor chooses):
```go
if storageMode != storage.ModeLocal {
    fsClient, err := firestore.NewClient(ctx) // ADC + AILANG_CLOUD_PROJECT
    if err != nil {
        return fmt.Errorf("failed to create Firestore client for feedback gate: %w", err)
    }
    cooldown := firestore.NewFeedbackGateCooldownStore(fsClient)
    budget := feedbackgate.NewBudget(firestore.NewFeedbackGateBudgetStore(fsClient))
    var provider ai.Provider
    if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
        provider = anthropic.NewClient(key)
    } else {
        fmt.Printf("  %s ANTHROPIC_API_KEY not set: feedback-gate classifier disabled (fail-closed)\n", yellow("⚠"))
    }
    classifier := feedbackgate.NewClassifier(provider, feedbackgate.DefaultPrompt(), budget)
    daemon.SetFeedbackGateDeps(cooldown, classifier)
}
```
- No silent fallback (CLAUDE.md rule 2): cloud mode without a reachable Firestore is
  already fatal (the `SetStores` block above returns on `NewBackends` error); a
  client error here is likewise fatal.
- The nil-provider `NewClassifier` path is the fail-closed posture (verified: the
  gate files heuristic-flagged messages, never dispatches — classifier.go:134-136).

**Acceptance criteria:**
- `SetFeedbackGateDeps` lands the deps on the daemon; `initTaskProcessing` copies
  them onto `d.feedbackGateCfg` so the decider sees them — **test the call-site, not
  just the setter** (M-ENV-FORWARD lesson): a wiring test asserts a gate with
  attached fakes reaches `Decide` with non-nil `Cooldown`/`Classifier`.
- **nil-deps ⇒ behavior identical to today**: a test with the gate enabled but no
  deps set asserts `d.feedbackGateCfg.Cooldown == nil` and
  `.Classifier == nil` after `initTaskProcessing` (rules-only unchanged).
- No-key construction path files (fail-closed): a construction-helper test with an
  empty `ANTHROPIC_API_KEY` yields a classifier whose nil-provider branch returns a
  `file` verdict for a heuristic-flagged input (assert on the built classifier, no
  network).
- Startup log names all three stages.
- `make build`, `make lint`, `go test ./internal/coordinator/... ./internal/storage/firestore/...`
  green in-worktree, no credentials.

### M3 — Rollout docs + handoff note + CHANGELOG — ~docs only, ~1h

**Files (modified):**
- `docs/docs/guides/coordinator.md` (or the feedback-gate section's home) — enablement
  runbook
- `design_docs/implemented/v0_29_0/m-feedback-triage-gate.md` — append the terraform
  handoff to its follow-up section
- `CHANGELOG.md`

**Runbook** (`coordinator.md`): DRY-RUN-first enablement:
- Week 1: `AILANG_FEEDBACK_GATE_DRY_RUN=1` — full gate runs, verdicts audited to the
  `feedback-gate-audit` inbox, but everything still dispatches. Watch for false
  positives via `ailang messages list --inbox feedback-gate-audit`.
- Then flip: set `coordinator.feedback_gate.enabled: true` (and remove DRY_RUN) to
  enforce.
- Env reference table: `AILANG_FEEDBACK_GATE_MODE` (off|file-only|full),
  `AILANG_FEEDBACK_GATE_DRY_RUN`, `ANTHROPIC_API_KEY` (absent ⇒ classifier fail-closed),
  `AILANG_STORAGE`/`AILANG_CLOUD_PROJECT` (Firestore backend).
- Document the two startup-log shapes (fully wired vs key-missing) from the design
  doc's Examples 1 & 2.

**Terraform handoff note** (appended to the implemented triage-gate doc's follow-up
section — same boundary as the parent sprint, NO sibling-repo files touched):
- Firestore TTL policy on `expires_at` for the two new collections
  (`feedback_gate_cooldown`, `feedback_gate_budget`).
- `ANTHROPIC_API_KEY` secret on the coordinator Cloud Run service.
- Note: without the TTL policy, stale docs are tiny and harmless (saturation-capped);
  the policy is housekeeping, not a correctness gate.

**CHANGELOG entry**: M-FEEDBACK-GATE-CLOUD-ADAPTER — Firestore cooldown/budget stores
+ Anthropic classifier wired into cloud coordinator; fail-closed no-key; DRY-RUN-first
runbook.

**Acceptance criteria:**
- Runbook documents DRY_RUN-first enablement + the env reference table.
- Terraform handoff appended to the implemented doc (no `ailang-multivac` files).
- CHANGELOG updated.
- Design doc M1/M2/M3 checkboxes flipped in
  `design_docs/planned/v0_29_0/m-feedback-gate-cloud-adapter.md`.

---

## Dependency Graph

```
M1 (adapters + pure math) ──► M2 (wiring) ──► M3 (docs)
```
Strictly sequential: M2 constructs the M1 adapters; M3 documents the M2 rollout.

## Day-by-Day (single day)

**Morning**: M1 (~3h) — both adapters, `trimAndCount`, compile-time assertions,
window-math table tests green.
**Afternoon**: M2 (~2h) — `SetFeedbackGateDeps`, init attach + startup log, CLI
construction block, wiring/no-key tests green. Then M3 (~1h) — runbook, handoff note,
CHANGELOG, checkbox flip.
**End of day**: `make build && make lint && make test` green in-worktree; no
sibling-repo files; no GPU/ollama touched; rig.lock untouched.

---

## Testing Strategy (no cloud credentials, no emulator, no GPU)

- **Pure window math (M1)**: table-driven unit tests on `trimAndCount` + the budget's
  plain increment. The Firestore transaction wrapper stays thin and UNtested locally —
  the repo's `internal/storage/firestore` convention (matches `CoordinatorStore` /
  `messaging.go`; no emulator in the repo).
- **Wiring (M2)**: coordinator tests with FAKE deps (in-memory cooldown + a classifier
  built on a fake `ai.Provider`, reusing the parent sprint's fakes). Assert deps land
  on the decider's cfg; absent deps ⇒ rules-only unchanged. Guard the call-site.
- **No-key fail-closed (M2)**: assert on the constructed classifier's nil-provider
  branch (returns `file`), zero network.
- **NO live Anthropic, NO Firestore emulator, NO Ollama, NO GPU.** Live Firestore
  verification is an OPS step at enablement time (the documented DRY-RUN week against
  the dev project), NOT a merge gate.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Firestore transaction contention on a hot cooldown key during a flood | Low | Per-key docs; saturation cap keeps docs small; contention → retries/errors → `applyCooldown` propagates the error → M4 wiring fails closed (never a dispatch flood) |
| Classifier live-fires unexpectedly on enablement | Med | Off by default (unchanged); runbook mandates DRY_RUN week 1; `mode=file-only` + `AILANG_FEEDBACK_GATE_MODE` kill-switch already shipped |
| `ANTHROPIC_API_KEY` absent on the deployed service | Med | Fail-closed + loud startup log (verified: gate files heuristic-flagged msgs); secret addition is in the terraform handoff note |
| Coordinator SA lacks Firestore perms for the two new collections | Low | SA already has `roles/datastore.user` project-wide (parent doc risk #3); no collection-level IAM |
| Double Firestore client construction (backends already made one) | Low | Deferred decision — executor may thread the existing client; a second client is correct, just wasteful. Not a merge blocker |
| Wiring imports break `cmd/ailang` build in-worktree | Med | 4 new imports are all existing in-repo packages; `make build` in-worktree is a hard gate |

## Out of Scope (follow-up handoffs, NOT milestones)

- Terraform TTL policies / `ANTHROPIC_API_KEY` secret (sibling repo `ailang-multivac`)
  — documented handoff only.
- Live cloud flood drill (real spend against the dev project) — separate ops task.
- Dashboard triage panel / `--show-triage` CLI — existing unrelated follow-up.
- Any change to gate decision logic, thresholds, or the classifier prompt — merged and
  evaluated (93/100); untouched here.
- Firestore-backed `HeartbeatStore` (v0.25 roadmap) — could share this adapter file's
  patterns later; not this sprint.

## Success Metrics (in-repo, testable this sprint)

- Both adapters satisfy the `feedbackgate` interfaces (compile-time assertions).
- `trimAndCount` table tests: 1h/24h boundaries exact, saturation cap bounds growth.
- Daemon startup log names live stages; nil deps ⇒ behavior identical to today.
- No-key cloud startup fails closed with the loud warning (tested on the constructed
  classifier).
- `make build && make lint && make test` green in-worktree; no sibling-repo files; no
  GPU/ollama; rig.lock untouched.
- Rollout runbook documents DRY_RUN-first enablement; CHANGELOG updated.

---

## References (verified live 2026-07-10)

- `internal/feedbackgate/cooldown.go:17-23` — `CooldownStore.Increment` interface
- `internal/feedbackgate/budget.go:17-21` — `BudgetStore.IncrementDaily` interface
- `internal/feedbackgate/config.go:73-79` — exported `Cooldown`/`Classifier` fields (nil ⇒ no-op)
- `internal/feedbackgate/classifier.go:62,134-146` — `NewClassifier` sig; nil-provider fail-closed; model on `ai.Request`, not the client
- `internal/coordinator/daemon_tasks_init.go:59-65` — the nil-deps wiring gap + startup log
- `internal/coordinator/daemon.go:108-109,165-179` — gate fields + `SetStores`/`SetCloudDispatcher` pattern to mirror
- `internal/coordinator/feedback_gate_wiring.go:36,103-112` — `FeedbackGateConfig` alias; fail-closed-on-error block
- `cmd/ailang/coordinator_lifecycle.go:103-112` — the `SetStores` wiring point this mirrors (imports `internal/storage` only today)
- `internal/storage/firestore/client.go:22,65` — `NewClient(ctx)` (ADC + `AILANG_CLOUD_PROJECT`); `RunTransaction`
- `internal/storage/firestore/messaging.go:93` — the `tx *firestore.Transaction` read-modify-write precedent
- `internal/ai/anthropic/client.go:58,141,347` + `step.go:41` — `NewClient` returns `*Client` satisfying full `ai.Provider`
- `internal/ai/provider.go:339-355` — `ai.Provider` = `Generate` + `Step` + `Name`

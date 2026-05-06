# M-PKG-CASCADE-DETERMINISTIC-FIRST

> Renamed from M-PKG-CASCADE-PROMPT-CLARITY after 2026-04-30 first-cascade observation
> revealed the deeper architectural issue: AILANG already has a hash-based change-class
> classifier (M-PKG-MSG, classifyChange in `internal/messaging/pkg_events.go`) that the
> cascade flow doesn't yet leverage. Most cascade bumps shouldn't need an AI agent at all.

**Status**: Implemented (dev only — test+prod pending v0.14.3 release tag)
**Target**: v0.16.x (follow-up to M-PKG-AUTONOMOUS-CASCADE-SAFE)
**Priority**: P0 — Routine cascades currently cost $0.10-0.30 + 3min per bump on a
            $0.000 deterministic operation. Also unblocks "no AI for safe bumps" pattern.
**Estimated**: 2-3 days (~400 LOC)
**Actual**: ~1 day (~750 LOC, larger than estimate due to schema plumbing across 8 files)
**Dependencies**: M-PKG-AUTONOMOUS-CASCADE-SAFE (implemented v0.16.x)

## Implementation Report (2026-05-01)

All four phases shipped in 1 day on the dev environment. End-to-end smoke testing
proved both the deterministic-first path AND the AI escalation path work correctly.

**What shipped:**

| Phase | Commits | Status |
|-------|---------|--------|
| 1: Envelope in cascade Pub/Sub data | `96804aec` | ✅ Verified end-to-end on dev |
| 2: Deterministic dispatch in wrapper | `630507f3` | ✅ PR #12: $0, 7.5s, zero AI |
| 3: Template variables for AI escalation | `aa763956` (ailang) + `23b4c9b` (multivac) | ✅ PR #11: AI repaired wrap.ail with full hash context |
| Conservative C-classifier fix | `92576123` | ✅ Committed; class-C smoke deferred (see Known Gaps) |

**Headline metrics from PR #12 (the success-path smoke test, dev environment):**

```
18:51:23 cascade detected — root=sunholo/test_pkg@0.0.27, change_class=A, dispatch_path=deterministic
18:51:23 deterministic bump sunholo/test_pkg → 0.0.27 in /workspace/.../ailang.toml
18:51:24 deterministic ailang lock regenerated
18:51:26 deterministic ailang check passed
18:51:26 ✓ deterministic cascade bump succeeded — skipping AI executor
18:51:26 [coordinator/task-...] [cascade] bump sunholo/test_pkg to 0.0.27
18:51:28 pushed branch coordinator/task-6c4e8bc2
18:51:29 opened PR #12: https://github.com/sunholo-data/ailang-packages/pull/12
18:51:30 task completed (turns=0, tokens=0+0, cost=$0.0000)
```

| Metric | Before (AI-only) | After (deterministic-first) |
|---|---|---|
| Wall time (dispatch → PR) | ~3 minutes | **7.5 seconds** |
| AI cost per cascade | $0.10–0.30 | **$0.00** |
| AI tools invoked | 5–12 | **0** |
| Reliability | model-variance | deterministic Go code |

**Class-A escalation path (PR #11, before fixture fix landed):**
- Wrapper attempted deterministic bump
- `ailang.toml` updated, `ailang.lock` regenerated
- `ailang check` failed (consumer wrap.ail had deprecated `++` on strings)
- Wrapper escalated to AI with full cascade context (Phase 3 template variables)
- AI received `{{.RootPackage}}=sunholo/test_pkg@0.0.26`, `{{.FromVersion}}=0.0.25`,
  `{{.ToVersion}}=0.0.26`, `{{.RootChangeClass}}=A`, full hashes
- AI repaired `wrap.ail` (converted `++` to `${}` interpolation)
- PR opened with `[cascade]` title + `cascade` label

This proved the design's escalation semantics: routine bumps are free, breakage
repair invokes the AI but with rich enough context that even haiku can succeed.

**Code summary (8 files, ~750 LOC):**

| File | Change |
|------|--------|
| `internal/pubsub/publisher.go` | New `PublishCascadeWithEnvelope` + `CascadeEnvelopeFields` + `CascadeMessageData`; old `PublishCascade` is now a thin shim |
| `cmd/ailang/pkg_publish.go` | `mapChangeClassToSchema` (A/B/C), `effectsWidened`, `exportsRemoved`; emit envelope to cascade publisher |
| `internal/coordinator/pubsub_adapter.go` | Decode envelope from data field with attribute-only fallback; populate Message |
| `internal/coordinator/watcher.go` | Add 11 cascade fields to `Message` |
| `internal/coordinator/store.go` | Add 11 cascade fields to `TaskRecord` |
| `internal/coordinator/store_sqlite.go` | ALTER TABLE for 11 new columns; INSERT writes them |
| `internal/coordinator/daemon_tasks_polling.go` | Cloud path copies cascade fields msg → task |
| `internal/coordinator/cloud_dispatcher.go` | DispatchParams gets cascade fields |
| `internal/dispatch/cloudrun/dispatcher.go` | Inject 7 `AILANG_CASCADE_*` env vars into Cloud Run Job |
| `cmd/ailang/coordinator_cloud.go` | `classifyDispatchPath` + `deterministicCascadeBump` (~150 LOC); skip AI when deterministic succeeds; skip AGENTS.md injection for cascade tasks |
| `internal/coordinator/stage_execution.go` | 9 new template variables ({{.RootPackage}}, {{.FromVersion}}, {{.ToVersion}}, etc.) |
| `internal/storage/firestore/coordinator_convert.go` | Persist + hydrate cascade fields; new `getStringSlice` helper |
| `ailang-multivac/config/templates/pkg-update.md` | Action-first restructure; uses Phase 3 template variables |

**Known Gaps (deferred):**

1. **Class C smoke test in dev**: the conservative C-classifier fix (`92576123`)
   is committed but not retested end-to-end because three local `ailang publish`
   processes from earlier today are stuck in kernel uninterruptible sleep state
   (network syscalls hung, can't be killed without reboot or sudo). The previous
   class-A-with-check-failure test (PR #11) exercised the same AI escalation
   code path the class-C path uses, just with different classification. Risk
   is low; full retest needed before tag.

2. **Test+prod deployment**: the cascade work is dev-only. Promotion to test
   requires cutting a v0.14.3 release tag (the `ailang-core-test-release`
   trigger fires on `^v.*` tags only); prod requires a manual `promote-to-prod`
   trigger run after that. Multivac config changes (templates, agent
   registrations) DO promote via `git push origin dev:test/prod` and the
   `ailang-multivac-config-{test,prod}` triggers.

3. **`*.md` ignore filter on multivac config triggers**: the
   `ailang-multivac-config-*` triggers ignore `**/*.md`, which incorrectly
   excludes `config/templates/*.md` (functional config, not docs). Worked
   around today by manually triggering the build; the filter should be
   refined to `docs/**/*.md` for the next sprint.

4. **gh CLI absent in agent executor image**: the wrapper opens PRs via
   GitHub REST API instead. Works fine but means we depend on the API rather
   than gh's auth/retry logic.

5. **Function-level interface diff**: the classifier compares MODULE exports,
   not function exports. Removing a function from an unchanged module is
   classified as C (conservative) but could in principle be class B if the
   removed function was unused by all consumers. Real function-level diff
   would need an `interface.json` export listing per-function signatures.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Headline win: most cascade bumps become deterministic Go code path. AI only fires on genuine breakage |
| A2: Replayability | +1 | Cascade decisions become reproducible from envelope hashes alone (no model variance) |
| A3: Effect Legibility | +1 | Effect-widening already detected by classifier; we surface it explicitly in the dispatch path |
| A4: Explicit Authority | +1 | Wrapper-driven bumps have no implicit ambient authority (no AI tool calls); only escalation path requires authority |
| A5: Bounded Verification | +1 | Hash-based classification IS bounded local verification |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +3 | Strongest justification: routine cascades become $0 / 5-second operations instead of $0.10-0.30 / 3-minute AI runs |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +2 | Per-cascade cost drops 10-30× for the common case (class A/B bumps); clearly visible cost ceiling |
| A10: Composability | +1 | Existing classifier composes with new dispatcher gate — pure additive change |
| A11: Structured Failure | +1 | Class-based dispatch makes failure modes (escalate-to-AI) explicit instead of "AI handles all cases" |
| A12: System Boundary | +1 | Pub/Sub message becomes self-contained (envelope embedded in data field) — no cross-system fetch dependency |

**Net Score: +13** → **Decision: Move forward (P0)**

### Hard Violation Check

- [x] A1 (Determinism): Major improvement — replaces AI-driven semver decisions with hash-driven classifier
- [x] A3 (Effects): Effect-widening detection becomes a first-class dispatch signal, not buried in AI prompt
- [x] A4 (Authority): Wrapper has no ambient authority; AI only invoked when needed
- [x] A7 (Machines First): The whole point — replaces interpretation with computation

## Problem Statement

M-PKG-AUTONOMOUS-CASCADE-SAFE shipped end-to-end cascade infrastructure. Today's first
real cascade observation against `ailang-multivac-dev` proved the **infrastructure
works perfectly** (Pub/Sub IAM, topic dispatch, agent invocation, branch push, PR
creation all green). But the AI agents (haiku at $0.0152, sonnet at $0.2177) couldn't
actually do the bump because the prompt didn't carry enough context to know what to
bump to or how.

While debugging, we hit a deeper architectural insight: **AILANG already has the full
machinery for deterministic semver classification** (M-PKG-MSG):

- `classifyChange(old, new) -> "A"|"B"|"C"` in `internal/messaging/pkg_events.go:162`
  - A = internal-only (content hash differs, interface hash same) → patch
  - B = additive (new exports, existing interface unchanged) → minor
  - C = contract change (interface hash differs) → major (or minor if pure-additive)
- `effectsWidened(old, new) -> bool` — detects effect ceiling growth
- Full envelope captured at publish time with `from_interface_hash`, `to_interface_hash`,
  `from_content_hash`, `to_content_hash`, `change_class`, `prev_effect_ceiling`,
  `new_effect_ceiling` (`PackageRef` in `internal/messaging/pkg_schema.go`)
- All persisted to local SQLite via `EmitUpgradeAvailable`

**The gap:** the cascade Pub/Sub message currently carries only minimal attributes
(`source=cascade`, `root_package`, `category=patch`). The full envelope with hash
deltas lives in the publisher's local SQLite store and never reaches the cloud
coordinator. So the cloud agent has no way to know:

- What semver bump class this is (just a coarse `category`)
- Whether the interface actually changed (no hash data)
- Whether effects widened (no ceiling data)

Without that data, the agent has to either (a) make conservative AI-driven decisions
(expensive, slow, model-variance) or (b) do nothing (today's failure mode).

**Observed runs (2026-04-30, ailang-multivac-dev):**

| Task ID | Model | Cost | Turns | Tools | Result | Why |
|---|---|---|---|---|---|---|
| task-feac88d6 | haiku | $0.0132 | 1 | 0 | placeholder AGENTS.md | Source attribute was empty (separate bug, fixed) |
| task-87d4305a | sonnet | $0.2177 | 12 | 11 | placeholder AGENTS.md | Same bug, sonnet just spent more deliberating |
| task-b3070725 | haiku | $0.06 | ~10 | ~5 | PR #9 with only AGENTS.md | After Source fix, but {{.Content}} = msg ID, no root package data |

**Root cause:** the cascade is shaped as "wake up an AI and let it figure out what
to bump to" instead of "compute the bump deterministically, only invoke AI on
breakage." Routine cascades become $0.10-0.30 AI runs for what should be a $0
deterministic toml edit + lock regenerate.

## Goals

**Primary Goal:** Make the cascade workflow execute deterministically by default
using the existing hash-based change classifier; AI is invoked only when the
deterministic bump fails check/test (interface breakage that needs interpretation).

**Success Metrics:**
- Class A (patch) cascade cost: from current ~$0.20 → **$0.00** (no AI dispatched)
- Class A cascade time: from current ~3 minutes → **<10 seconds** (Cloud Run Job not even spun up)
- Class C cascade with no breakage: ~$0.00 (deterministic bump, check passes, no AI needed)
- Class C cascade with breakage: ~$0.10-0.30 (AI dispatched only for repair, with full hash context in prompt)
- Cascade Pub/Sub message is self-contained: full envelope (with hashes + change_class + effects) in the `data` field
- All 21 production pkg-* agents continue to work; deterministic dispatch is opt-in per agent

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Embed full envelope in cascade Pub/Sub `data` (not just msg ID pointer) | Self-contained message; coordinator no longer depends on Firestore fetch that doesn't have the data | human | design | low (~30 LOC in publisher.go + pubsub_adapter.go) |
| Wrapper-side deterministic bump path (no AI for class A/B with green check) | The headline win — most cascades become $0 / <10s instead of $0.20 / 3min | human | design | med (~150 LOC new in coordinator_cloud.go) |
| AI escalation only on check/test failure (interface breakage repair) | Bounds when AI is invoked to genuine interpretation tasks | human | design | low (~30 LOC: skip Cloud Run Job dispatch when wrapper succeeded) |
| Plumb hashes + change_class through Pub/Sub attributes too (alongside data field) | Lets cheap dispatcher decisions skip even data-decoding | human | design | low (~20 LOC in PublishCascade) |
| Keep template guard as defense-in-depth (don't remove from `pkg-update.md`) | The IAM layer is primary, but template guard catches a misconfiguration class | human | design | low (no change) |

### Design Freeze

- [x] Embed envelope in cascade Pub/Sub `data` field (locked — solves the cross-system fetch problem)
- [x] Deterministic bump path runs FIRST in the wrapper, AI only on failure (locked — verified by today's observation that AI-first costs 100× more for the common case)
- [x] AI escalation has full hash context in prompt when invoked (locked — agent needs from/to interface hashes to do interface repair)
- [x] Backwards compat: agents without `template_by_source` use existing flow (locked)
- [ ] What to do when wrapper deterministic bump fails on a CONTENT-only change (class A)? Should be impossible (no API change), but defensive programming says: log loudly, don't escalate to AI for what should be impossible. **Recommendation: panic in dev, escalate-with-warning in prod.**
- [ ] How to expose deterministic-bump opt-out for an agent (e.g., a package that wants AI even for patch bumps)? **Recommendation: `prefer_deterministic_bump: false` field on AgentConfig, default true.**

## Solution Design

### Overview

Three coordinated changes:

1. **Embed envelope in cascade Pub/Sub `data` field** — cascade messages become
   self-contained. Coordinator + wrapper have everything from the message itself.

2. **Deterministic-first dispatch in `coordinator_cloud.go`** — wrapper checks
   change_class. If class A or B (or class C with pure-additive interface), it:
   reads consumer's `ailang.toml` → bumps the dep pin → runs `ailang lock` →
   runs `ailang check` → runs `ailang test`. If all green: commit, push, open PR,
   exit. **No Cloud Run Job for the AI agent is ever spun up.**

3. **AI escalation only on check/test failure** — if class C breaking OR if check/test
   failed after the deterministic bump, NOW dispatch the agent with full context
   (from/to interface hashes, what export(s) changed shape, lock diff, test failure).
   The agent is then doing what it's actually good at: interface change repair.

### Architecture

**Components:**

1. **`PublishCascade` change** (publisher.go): the `data` field becomes the full
   `PackageMessageEnvelope` JSON instead of `{"message_id": "..."}`. Backward-
   compatible: receivers that only use attributes still work; new receivers can
   decode the envelope.

2. **`pubsub_adapter.go` decode** (cloud coordinator): when receiving a cascade
   message, attempt to decode the data as `PackageMessageEnvelope`. If success:
   populate Message fields with full envelope data (interface hashes, change_class,
   effects). If decode fails: fall back to attribute-only path (legacy compat).

3. **`Message`/`TaskRecord` schema additions** (~50 LOC):
   - `RootPackage string` — vendor/name@version
   - `RootChangeClass string` — A, B, or C
   - `FromInterfaceHash, ToInterfaceHash string`
   - `EffectsWidened bool`
   - `PrevEffectCeiling, NewEffectCeiling []string`

4. **Deterministic bump path** (`coordinator_cloud.go`, new function `deterministicCascadeBump`):
   ```
   if task.IsCascade && classifyDispatchPath(task) == DispatchDeterministic {
       result := deterministicCascadeBump(ctx, workDir, task)
       if result.ok {
           commit + push + openCascadePullRequest
           return  // skip AI dispatch entirely
       }
       // fall through to AI dispatch with breakage context
   }
   ```

5. **`classifyDispatchPath`**:
   - class A → `Deterministic` (content-only change, no API impact)
   - class B → `Deterministic` (additive, doesn't break consumers)
   - class C with no removed exports + no widened effects → `Deterministic` (additive minor)
   - class C with removed exports OR widened effects → `RequiresAI` (interface repair needed)

6. **AI escalation path** (template variables expanded so agents have hash context):
   ```
   {{.RootPackage}}            sunholo/test_pkg@0.0.24
   {{.RootChangeClass}}        C
   {{.FromInterfaceHash}}      sha256:abc...
   {{.ToInterfaceHash}}        sha256:def...
   {{.EffectsWidened}}         true
   {{.PrevEffectCeiling}}      [IO]
   {{.NewEffectCeiling}}       [IO, Net]
   {{.CheckOutput}}            (stderr from `ailang check` that failed)
   ```
   The agent now knows EXACTLY what to fix.

### Implementation Plan

**Phase 1: Self-contained cascade messages** (~3 hours)
- [ ] Modify `PublishCascade` to accept full envelope, marshal it to data field
- [ ] Update `emitDependentNotifications` in `pkg_publish.go` to pass envelope
- [ ] Add envelope decode in `pubsub_adapter.go` (with attribute-only fallback)
- [ ] Add new fields to `Message` and `TaskRecord` (RootPackage, RootChangeClass, etc.)
- [ ] Update Firestore taskToMap/mapToTask + SQLite ALTER TABLE for new columns
- [ ] Unit tests: roundtrip envelope through Pub/Sub data field

**Phase 2: Deterministic dispatch path** (~6 hours)
- [ ] Add `classifyDispatchPath` function (returns Deterministic | RequiresAI)
- [ ] Add `deterministicCascadeBump` in coordinator_cloud.go (read toml, bump dep, lock, check, test)
- [ ] Wire it into `executeCloudTask` BEFORE the AI executor invocation
- [ ] Skip AI dispatch when deterministic bump succeeded
- [ ] Add metrics: count of deterministic-bumps vs AI-bumps, time saved, $ saved
- [ ] Unit tests: classifier coverage for A/B/C-additive/C-breaking
- [ ] Integration test: class A cascade → no Cloud Run Job spawned, PR opens in <10s

**Phase 3: AI escalation context** (~3 hours)
- [ ] Add template variables (RootPackage, RootChangeClass, FromInterfaceHash, etc.)
- [ ] Wire variables in `stage_execution.go::buildTemplateDirective`
- [ ] Update `pkg-update.md` to use new variables and lead with action (action-first)
- [ ] Add `CheckOutput` to the prompt when escalating from a failed deterministic bump
- [ ] Smoke test: class C breaking cascade → AI dispatched with full hash context → completes bump-and-fix

**Phase 4: Observability + agentic_under_guards benchmark** (~3 hours)
- [ ] Track deterministic-vs-AI dispatch in `ailang chains view`
- [ ] Update `ailang pkg cascade status` to show dispatch path per node
- [ ] Add `agentic_under_guards` benchmark fixture (still useful as a separate signal)
- [ ] Run baseline for haiku/sonnet/opus

### Files to Modify/Create

**New files:**
- `benchmarks/agentic_under_guards/` — paired direct/guarded benchmark fixtures (~150 LOC)

**Modified files:**
- `internal/pubsub/publisher.go` — `PublishCascade` accepts envelope, marshals to data (~25 LOC)
- `internal/coordinator/pubsub_adapter.go` — decode envelope, populate Message (~40 LOC)
- `internal/coordinator/watcher.go` — Message struct: add cascade fields (~15 LOC)
- `internal/coordinator/store.go` — TaskRecord struct: add cascade fields (~15 LOC)
- `internal/coordinator/store_sqlite.go` — ALTER TABLE for new columns (~10 LOC)
- `internal/storage/firestore/coordinator_convert.go` — taskToMap + mapToTask (~20 LOC)
- `cmd/ailang/coordinator_cloud.go` — `classifyDispatchPath` + `deterministicCascadeBump` (~150 LOC NEW)
- `internal/coordinator/stage_execution.go` — template variable substitution (~20 LOC)
- `cmd/ailang/pkg_publish.go` — pass envelope to PublishCascade (~15 LOC)
- `ailang-multivac/config/templates/pkg-update.md` — use new variables, action-first (~50 LOC change)

## Examples

### Example 1: Class A (patch) cascade — deterministic, no AI

**Today:**
```
ailang publish (test_pkg 0.0.20→0.0.21, content-only change)
  → cascade Pub/Sub
  → coordinator dispatches Cloud Run Job (~30s spinup)
  → AI agent runs (haiku ~3min, $0.06-0.20)
  → maybe edits ailang.toml correctly, maybe doesn't
  → commit + push
  → wrapper opens PR
```
Total: ~3-5 minutes, $0.06-0.30, model-variance risk.

**After this work:**
```
ailang publish (test_pkg 0.0.20→0.0.21, content-only change)
  → cascade Pub/Sub (with full envelope)
  → coordinator decodes: change_class=A, no interface change, no effects widened
  → dispatcher calls deterministicCascadeBump
    → reads consumer ailang.toml
    → updates dep pin: "0.0.20" → "0.0.21"
    → ailang lock
    → ailang check --package . → green
    → ailang test --package . → green
    → git commit + push
  → wrapper opens PR with title "[cascade] bump test_pkg → 0.0.21 (deterministic)"
```
Total: <10 seconds, $0.00, fully deterministic.

### Example 2: Class C (breaking) cascade — wrapper tries first, escalates AI on failure

```
ailang publish (test_pkg 0.0.21→0.1.0, removed export `oldFunc`)
  → cascade Pub/Sub (envelope shows interface hash changed, oldFunc no longer in exports)
  → coordinator: change_class=C with removed export → RequiresAI from the start
  → dispatcher SKIPS deterministic path; goes straight to AI escalation
  → BUT first does the toml + lock change deterministically (the boring part)
  → THEN dispatches AI with full context:
      {{.RootPackage}} = sunholo/test_pkg@0.1.0
      {{.RootChangeClass}} = C
      {{.RemovedExports}} = [oldFunc]
      {{.CheckOutput}} = "wrap.ail:5: oldFunc is not defined in pkg/sunholo/test_pkg/hello"
  → Agent uses Edit + Bash to repair the consumer code
  → commit + push (agent does git commit; wrapper does push + PR)
```
Total: 1-3 minutes, $0.10-0.30 — but only when actually needed.

### Example 3: Edge case — class A but check fails (should be impossible)

A class-A cascade (content-only, no interface change) shouldn't be able to break
anything by definition. If `ailang check` fails after a deterministic class-A bump,
something is very wrong. The wrapper logs loudly and escalates to AI with the
output, but flags the cascade as `cascade-anomaly` for human attention.

## Success Criteria

- [ ] Cascade Pub/Sub message data field contains full `PackageMessageEnvelope` JSON
- [ ] Cloud coordinator decodes envelope and populates Message + TaskRecord with cascade fields
- [ ] `classifyDispatchPath` correctly identifies Deterministic vs RequiresAI for all class A/B/C-additive/C-breaking cases
- [ ] Class A cascade against `pkg-sunholo-test-pkg-consumer`: opens PR in <10s, no Cloud Run Job spawned, $0 cost
- [ ] Class C breaking cascade: AI receives `{{.CheckOutput}}` and can act on it
- [ ] All 21 production agents continue to work (no forced migration)
- [ ] `ailang pkg cascade status` shows `path=deterministic` vs `path=ai` per node
- [ ] `agentic_under_guards` benchmark added with baseline for haiku/sonnet/opus
- [ ] Smoke test scripts (`test_cascade_e2e.sh`) green
- [ ] Design doc moved to `implemented/v0_16_x/`

## Testing Strategy

**Unit tests:**
- `classifyDispatchPath`: 8 cases (A, B, C-additive, C-removed-export, C-widened-effects, C-both, missing-data, malformed-envelope)
- `PublishCascade` envelope round-trip
- `deterministicCascadeBump`: success, lock failure, check failure, test failure, escalation

**Integration tests:**
- Class A cascade end-to-end: no Cloud Run Job spawned (verify via metrics)
- Class C breaking cascade end-to-end: AI dispatched with check output in prompt
- Backwards compat: agent without cascade fields uses existing flow

**Manual smoke (against `ailang-multivac-dev`):**
- Bump `sunholo/test_pkg` (no API change) → deterministic PR opens in <10s
- Remove an export from `sunholo/test_pkg` → AI gets the breakage repair prompt
- Verify `ailang chains view` shows `dispatch_path=deterministic|ai` per task

## Deferred Decisions

- Whether to migrate all 21 production pkg-* agents to deterministic-first immediately or just test fixtures (agent may decide based on smoke test results — recommend: enable for all on day 2)
- Whether to track per-cascade $$ savings in dashboard (recommend: yes, surfaces the win clearly)
- Per-package opt-out (`prefer_deterministic_bump: false`) — should it be a real escape hatch or YAGNI? (recommend: YAGNI, add when first user actually needs it)
- Whether to expose `RootPackage` etc. as template variables for non-cascade workflows too (recommend: yes, generic — same machinery works for any agent that wants context)

## Non-Goals

- Removing the IAM Pub/Sub guard — that remains the primary security layer
- Changing how `EmitUpgradeAvailable` writes to local SQLite — local laptop flow unchanged
- Adding semver auto-publish on the dependent (still always-PR for v1)
- Refactoring `EmitInterfaceChangeNotice` / `EmitEffectWideningWarning` — those continue to work as before, this work only changes how the cascade Pub/Sub message carries data

## Timeline

**Day 1** (~9 hours):
- Phase 1: Self-contained cascade messages (envelope in data, schema fields, decode)
- Smoke test: verify cloud coordinator receives full envelope

**Day 2** (~9 hours):
- Phase 2: Deterministic dispatch path (classifyDispatchPath + deterministicCascadeBump)
- Smoke test: class A cascade opens PR in <10s with no AI

**Day 3** (~6 hours):
- Phase 3: AI escalation with full context (template variables)
- Phase 4: Observability + benchmark
- Move design doc to implemented

**Total: ~24 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Deterministic bump path miscategorizes a C-breaking change as B-additive | Med | Conservative classification: any unknown signal → escalate to AI. Unit tests cover edge cases. |
| Pub/Sub 1MB attribute limit hit if envelope is huge | Low | Envelopes are typically <2KB (just hashes + version metadata, no source code). Add a guard. |
| Schema migration breaks existing in-flight messages | Low | Phase 1 keeps backward compat: new field decode is best-effort, old messages still work via attribute-only path. |
| `ailang lock` fails in cloud due to network/auth | Med | Wrapper already configures git creds in Step -1; lock uses same registry HTTP path. Add timeout + escalate to AI on failure. |
| Cost-savings aren't measurable because nobody runs many cascades | Low | Add a synthetic baseline run as part of CI (e.g., 10 deterministic vs 10 AI cascades, track time + cost) |

## Related Documents

**Implemented (parent / sibling work):**
- [m-pkg-autonomous-cascade-safe.md](../../implemented/v0_16_x/m-pkg-autonomous-cascade-safe.md) — direct parent: built the cascade infrastructure
- [m-pkg-autonomous-updates.md](../../implemented/v0_10_0/m-pkg-autonomous-updates.md) — grandparent: original autonomous-publish pipeline
- [M-PKG-MSG (m-pkg-msg.md or via PackageMessageEnvelope schema)](../../implemented/) — shipped the `classifyChange` machinery this work uses

**Code references:**
- `internal/messaging/pkg_events.go:162` — `classifyChange` (the classifier we already have)
- `internal/messaging/pkg_events.go:173` — `effectsWidened` (the effect-widening detector)
- `internal/messaging/pkg_schema.go:60-90` — `PackageRef` schema (carries the hash deltas)
- `internal/pubsub/publisher.go::PublishCascade` — what gets the envelope-in-data change
- `internal/coordinator/pubsub_adapter.go::HandleNotification` — what gets the decode change
- `cmd/ailang/coordinator_cloud.go::executeCloudTask` — where the deterministic dispatch path lands

## References

- [Design Axioms](/docs/references/axioms) — A1, A4, A7 are the headline justifications
- [Today's first cascade observation](https://github.com/sunholo-data/ailang-packages/pull/9) — closed; demonstrated the gap (PR #9 showed the agent-injected AGENTS.md instead of an actual bump because the prompt had no root_package data)

## Future Work

- **Cross-runtime semver classifier** — extend the same machinery to classify changes
  in non-AILANG dependencies (e.g., a Go module bump in a build script). Would need
  language-specific interface-hash extractors but the dispatch logic is the same.
- **Pre-commit hooks for class-C consumers** — when a maintainer is about to publish
  a class-C bump, surface the affected dependents in the publish output so they can
  see the blast radius BEFORE pushing.
- **Auto-merge for class A bumps after green CI** — once humans trust the
  deterministic flow, allow class-A cascades to auto-merge with a CI gate. Requires
  policy infrastructure not in scope here.

---

**Document created**: 2026-04-30
**Last updated**: 2026-05-01 (renamed from M-PKG-CASCADE-PROMPT-CLARITY after first-cascade observation)

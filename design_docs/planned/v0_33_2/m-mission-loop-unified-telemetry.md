# M-MISSION-LOOP-UNIFIED-TELEMETRY

**Status**: Planned
**Target**: v0.33.2
**Priority**: P1 — nothing is broken or losing data; the data exists and cannot be *joined*
**Estimated**: ~3 days (~22 hours, 3 milestones)
**Dependencies**: [m-openrouter-broadcast-ingest](../v0_33_1/m-openrouter-broadcast-ingest.md) (shipped v0.33.1 — Broadcast traces now arrive uncorrupted; this doc makes them joinable)
**Authorized**: Mark, 2026-08-13 — *"the broadcast traces should land in the cloud route — I'm looking to be able to follow a whole mission loop across providers in our data"*

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Telemetry routing; no evaluation semantics change |
| A2: Replayability | +1 | A mission loop becomes replayable as one record instead of three partial ones in two datastores |
| A3: Effect Legibility | 0 | No AILANG effect changes |
| A4: Explicit Authority | 0 | Writes go where they already go, to a store that already authenticates |
| A5: Bounded Verification | +1 | "Did this stage complete?" becomes locally checkable instead of inferred from absence |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The point: an agent can answer "what did iteration N cost across providers" with one query |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Directly — today a mission chain reports $0.00 while holding $0.1077 of stage cost |
| A10: Composability | +1 | Reuses the existing `sessions` → `chain_id`/`stage_id` linkage rather than adding a parallel one |
| A11: Structured Failure | +1 | A stage stuck at `pending` currently cannot be distinguished from one that never ran |
| A12: System Boundary | +1 | Makes the provider→observatory boundary explicit and uniform across four providers |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): explicitly optimizing for machine queryability

## Problem Statement

**A single mission iteration already spans four providers. None of them can be followed end to end.**

Measured on `manual:mission:v1/iter-190` (local observatory, 2026-08-13):

| Stage | Provider | Status | Cost | Tokens |
|-------|----------|--------|------|--------|
| controller | `quota:opus` (Anthropic) | `pending` | — | — |
| designer | `quota:codex` (OpenAI) | `pending` | — | — |
| quorum-r1 | gpt5-6-sol + gemini-3-1-pro (OpenRouter) | `pending` | $0.0570 | **0** |
| quorum-r2 | gpt5-6-sol + gemini-3-1-pro (OpenRouter) | `pending` | $0.0507 | **0** |
| **Chain total** | | | **$0.0000** | **0** |

The structure is right — the mission loop already models itself as a chain whose stages name their
providers. Three independent defects stop it being usable.

**1. Fragmentation across datastores.** A rig eval or mission run writes chains and stages to the
*local* SQLite observatory (`~/.ailang/state/observatory.db`; `OBSERVATORY_ENDPOINT` defaults to
`localhost:1957`). OpenRouter Broadcast traces are pushed by OpenRouter to the *cloud* observatory
(`dashboard.ailang.sunholo.com`, Firestore). Same iteration, two stores, no path between them —
there is no local→cloud sync of observatory data (V4). Every analysis tool — `ailang chains`,
`ailang eval-*`, `internal/eval_analysis` — reads the local store via `OpenDefaultStore()`, so none
of them can see a Broadcast trace at all.

**2. No linkage even within one store.** `convertSpan` populates `ChainID` only from
`ailang.chain_id` (resource attrs) or `chain_id` (span attrs), at
[otlp_receiver.go:445-447](../../../internal/observatory/otlp_receiver.go). OpenRouter delivers our
chain ID as **`session.id`** — verified live on the v0.33.1 probe span, which carried
`session.id = m4-broadcast-probe-session` and `chain_id` **absent**. The identifier is delivered
correctly and then not read.

**3. Mission stages never complete, and never carry tokens.** All four stages above sit `pending`.
The two that record cost record **0 tokens**. The chain total aggregates neither. So even a
perfectly-joined Broadcast trace would attach to a chain whose own accounting is empty.

**Impact:** the v1.0 `cost-per-verified-success` KPI (see
[m-cost-per-success-kpi-sprint-plan](../v1_0_0/m-cost-per-success-kpi-sprint-plan.md)) needs
attributable metered dollars per unit of work. Today the mission loop — the largest consumer of
those dollars — reports `$0.00`. The numbers exist; they are in the wrong store, under the wrong
key, or never written.

## Goals

**Primary Goal:** One cloud-side record per mission iteration that can be queried end to end across
Anthropic, OpenAI, OpenRouter and local providers.

**Success Metrics:**

- Given a mission iteration ID, **one query** returns every stage, its provider, its status, its
  tokens and its cost — including the OpenRouter Broadcast spans.
- A mission chain's total cost equals the sum of its stages' costs (today: $0.00 vs $0.1077).
- ≥95% of Broadcast spans emitted during a mission iteration resolve to a `chain_id`.
- No stage remains `pending` after its iteration ends; a stage that genuinely failed says so.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| ~~Where Broadcast traces land~~ **DECIDED: the cloud route** | Determines whether the join happens cloud-side or rig-side, and therefore which half has to move | human (Mark, 2026-08-13) | design | — |
| ~~Dual-write vs. mirror~~ **RATIFIED: dual-write** (Mark, 2026-08-13) | Mirroring's usual justification is volume, and there is none: 42 chains / 238 stages per day. It would buy little and cost a reconciliation problem. `PostIteration` already writes directly. **Scope is NODE-generic, not rig-specific** — "this server, laptop, cloud, other nodes in the future" | human | design | — |
| Reuse `sessions.chain_id` linkage vs. add a `session.id`→`chain_id` branch in `convertSpan` | The `sessions` table already carries `chain_id`+`stage_id`; a second mechanism would need keeping in sync with it forever | agent | design | med |
| Whether `session.id` handling may change for the Claude Code path that owns it today | `convertSpan`'s session block is Claude-Code-specific; widening it risks mis-attributing Claude spans | agent | compile | med |
| ~~Local analysis reads cloud?~~ **RATIFIED: yes, OPT-IN** (Mark, 2026-08-13) | Opt-in keeps offline nodes first-class and is reversible; if `--remote` turns out to be always-passed, flipping the default later is one line with evidence behind it. Canonical-first is the version you cannot walk back | human | design | — |

### Design Freeze

Before implementation begins:

- [x] **Dual-write vs. mirror** — **RATIFIED: dual-write**, and deliberately NODE-GENERIC. Not
      "the rig pushes to cloud" but "any node writes to cloud": this server, a laptop, a Cloud Run
      job, and future workers. Design the write path so the node is a parameter, not an assumption.
- [x] **Offline behaviour** — **RATIFIED: never block.** Mark: *"no block if not available, at least
      until we harden availability."* Implement by EXTENDING the existing bounded+loud spool
      (`internal/observatory/spool.go`), not by inventing a policy — see V10.
- [x] **Local analysis reads cloud** — **RATIFIED: yes, OPT-IN.** `OpenDefaultStore()` keeps its
      local default; an explicit remote mode is added. Every `ailang chains` / `eval-*` consumer
      inherits the option, none inherit a changed default.

**All Design Freeze items are ratified; the sprint has no pause point.**

## Solution Design

### Overview

Three milestones, each independently landable and independently useful. **M2 is worth doing even if
M1 and M3 are deferred** — it fixes accounting that is wrong today regardless of where the data
lives.

### Architecture

The load-bearing discovery: **the Firestore observatory already satisfies the same `Backend`
interface as SQLite** — `var _ obs.Backend = (*ObservatoryStore)(nil)` at
[observatory.go:16](../../../internal/storage/firestore/observatory.go) — and already implements
`CreateChain`, `CreateStage`, `UpdateStageStatus`, `CreateSpan`, `GetSession` and the aggregates.

So "rig data reaches the cloud" is a **routing/wiring** problem, not a porting one. No new datastore,
no new schema, no second model.

**Components:**

1. **Session-keyed chain linkage.** `chain_stages` already has a `session_id` column and `sessions`
   already carries `chain_id` + `stage_id`. Register the OpenRouter `session_id` we send as a
   session row bound to the stage; `convertSpan` then resolves `session.id` → chain via the existing
   table rather than a new parallel map.

2. **Mission stage lifecycle completion.** Mission stages are created but never transitioned or
   credited. Close the loop: status on finish, tokens and cost recorded, chain totals aggregated.

3. **Cloud routing for rig writes.** Point the rig's observatory writes at the cloud backend, per
   the dual-write/mirror decision.

### Implementation Plan

**M1: Session-keyed chain linkage** (~6 hours)
- [x] Register the correlation `session_id` as a `sessions` row bound to `chain_id` + `stage_id` when
      a mission/eval stage dispatches an OpenRouter call.
- [x] Extend `convertSpan` to resolve `chain_id` via `sessions` when `session.id` is present and no
      explicit `ailang.chain_id` was supplied — **without** disturbing the Claude Code path that owns
      that attribute today.
- [x] Test both directions: a Claude Code span still resolves as it does now; an OpenRouter Broadcast
      span resolves to its chain.

**M2: Mission stage lifecycle + accounting** (~8 hours)

Two *distinct* defects with different owners — worth separating, because a single "fix mission
accounting" change would likely patch one and miss the other:

- [x] **Status (writer-side).** `PostIteration` creates stages and never transitions them, so they
      keep `CreateStage`'s `StageStatusPending` default. Add the status transition in
      `iteration_post.go`, carrying per-stage outcome rather than blanket-completing.
- [x] **Tokens (caller-side).** `UpdateStageMetrics` already receives `st.TokensIn`/`st.TokensOut`
      correctly; the zeros come from the caller. Thread real token counts through
      `ailang chains post-iteration` and the mission-control skill that invokes it.
- [x] Aggregate stage cost/tokens into the chain total.
- [x] Regression fixture built from the real iter-190 shape (4 stages, 3 providers, cost-without-tokens).

**M3: Cloud routing for rig writes** (~8 hours)
- [x] Wire the rig's observatory writes to the Firestore backend per the frozen decision.
- [x] Offline behaviour per the frozen decision.
- [ ] Verify a full mission iteration lands cloud-side with every stage and its Broadcast spans.
      **NOT DONE — needs a live run on a `AILANG_STORAGE=gcp` node.** The write path and its
      spool are implemented and unit-tested against a non-SQLite sink; the end-to-end confirmation
      is a manual step that cannot be performed from the sprint sandbox (no cloud credentials).

### Files to Modify/Create

**Modified files:**
- `internal/observatory/otlp_receiver.go` — resolve `chain_id` via the `sessions` table when `session.id` is present, ~60 LOC
- `internal/observatory/otlp_receiver_test.go` — Claude-Code-path regression + OpenRouter resolution, ~120 LOC
- `internal/observatory/iteration_post.go` — set stage status (the missing `UpdateStageStatus`; V6a), ~50 LOC
- `internal/observatory/iteration_post_test.go` — iter-190-shaped fixture: 4 stages, 3 providers, cost-without-tokens, ~110 LOC
- `cmd/ailang/chains_post.go` — accept and forward per-stage token counts (V6b), ~50 LOC
- `.claude/skills/mission-control/SKILL.md` — supply tokens when posting an iteration, ~15 LOC
- `internal/ai/openrouter/correlation.go` — emit the session registration alongside the correlation fields, ~40 LOC
- `internal/storage/backend.go` — rig-side backend selection for observatory writes, ~40 LOC

## Examples

### Example 1: following one iteration

**Before** — three partial views, two stores, no join:
```bash
ailang chains view d075f569-9e4     # local: 4 stages, all pending, $0.00, 0 tokens
curl .../api/observatory/spans      # cloud: OpenRouter spans, rich, chain_id absent
# no query spans both
```

**After:**
```bash
ailang chains view <iter-191> --remote
#  1. controller  (anthropic/opus)        [completed]  $0.42  18k in / 2k out
#  2. designer    (openai/gpt-5.6-sol)    [completed]  $0.31  22k in / 4k out
#  3. quorum-r1   (openrouter, 2 models)  [completed]  $0.057  9k in / 1k out
#  4. quorum-r2   (openrouter, 2 models)  [completed]  $0.051  8k in / 1k out
#  Total: $0.838 — matches the sum of its stages
```

## Success Criteria

- [ ] One query returns a full mission iteration across all four providers — **needs a live
      mission iteration**; the code path exists and is unit-tested, the measurement is not made
- [x] Chain total equals the sum of stage costs (regression fixture from iter-190)
- [ ] ≥95% of a mission iteration's Broadcast spans carry a resolved `chain_id` — **M1 landed and
      is unit-tested; the ≥95% figure is a live measurement, not yet taken**
- [x] No stage ends an iteration in `pending` — the CLI accepts and stores a per-stage status and
      the mission-control skill now supplies it; a stage that omits one is still left `pending` by
      design (version skew), so the live figure depends on the skill having been updated
- [x] **Claude Code session correlation is unchanged** — asserted by a test that fails if the
      existing path regresses
- [x] All tests passing; `make lint` clean

## Testing Strategy

**Unit tests:**
- `session.id` → `chain_id` resolution: hit, miss, and the case where `ailang.chain_id` is explicitly
  present and must win.
- Chain aggregation: stage costs sum; a stage with cost-but-no-tokens (the real iter-190 shape) does
  not silently contribute zero.

**Integration tests:**
- Full OTLP/JSON ingest of a Broadcast-shaped payload carrying `session.id`, asserting the stored span
  resolves to a seeded chain.
- **Negative control**: a Claude Code span with `session.id` and no chain resolves exactly as it does
  today. This is the regression that matters — that attribute currently belongs to Claude Code.

**Manual:**
- One real mission iteration, end to end, read back cloud-side.

## Deferred Decisions

- **Per-stage outcome semantics for M2** — whether a mission stage transitions to `completed`,
  `failed`, or a finer status, and where that outcome comes from. The *mechanism* is pinned (V6a);
  the vocabulary is the implementer's call.
- **Whether `--remote` is a flag, an env var, or a config key** — agent may choose.
- **Retention/cost of cloud-side span volume** — agent may raise if it looks material.

## Strategic Placement (Mark, 2026-08-13)

Recorded because it changes how this should be BUILT, not just why it is worth building.

The direction is **cloud-native, for an eventual AILANG cloud product**. The history matters: cloud
first on a laptop → moved on-device when this server arrived, to trial its GPU eval cluster → that is
now fairly stable → moving back to cloud. So on-device was a deliberate, temporary posture, and this
work is the first step back.

Two consequences for this design:

1. **"Dual-write" means node-generic, not rig-specific.** The write path should treat the node as a
   parameter — this server, a laptop, a Cloud Run job, future scale-out workers. Anything that hard-codes
   "the rig" is the wrong shape and will need re-doing.
2. **Availability hardening is explicitly deferred, not forgotten.** "No block" is correct *for now*;
   the spool's bounded+loud contract is what keeps that honest rather than silently lossy.

**Adjacent, deliberately NOT in this doc's scope** — several mechanics are half-implemented and want
their own passes: the coordinator, the dashboard, and `ailang messages`. On the last, Mark's
"I think they are separated at the moment" was checked and is correct, with a sharper shape than
separation (see V11): local and cloud inboxes hold **different populations** — internal loop traffic
here, external/public traffic there — and neither side sees the other. The eventual integration
surface named: Discord, `ailang messages`, the dashboard, evals as Cloud Run jobs, and workers for
scale-out missions.

## Non-Goals

- **Dashboard refactor for mission control.** Mark named this as the follow-up, and it depends on
  this data existing first. Designing the UI before the data model settles would be building on sand.
- **Changing what the mission loop *does*** — this is observability only; no routing, model-selection
  or gate behaviour changes.
- **Backfilling historical mission chains.** Past iterations stay as they are; this fixes forward.
- **Repairing the 190 pre-v0.33.1 corrupted Broadcast rows** — separate, already-scoped follow-up
  (Firestore-side repair; `migrate_v18` cannot reach them).

## Timeline

**Day 1** — M1 session-keyed linkage
**Day 2** — M2 mission stage lifecycle + accounting
**Day 3** — M3 cloud routing, end-to-end verification

**Total: ~22 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Widening `session.id` handling mis-attributes Claude Code spans | **High** | `ailang.chain_id` keeps precedence; the new path only fires when it is absent; negative-control test |
| Dual-write couples every rig run to cloud availability | **High** | Design Freeze item — offline behaviour decided before coding, not discovered in production |
| Cloud-side analysis makes offline rig runs second-class | Med | Freeze item; local remains the default read unless explicitly remote |
| Blanket-completing every stage would hide real failures | Med | Status must carry per-stage outcome, not a uniform `completed`; a stage that failed must still say so (Deferred: exact vocabulary) |
| Span volume cost in Firestore | Low | Measure during M3; sampling already available on the Broadcast side |

## Verification Log

Every row checked against code or the live service on **2026-08-13**. Negative-existence claims carry
their own row per the design-doc-creator hard gate.

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | **No code maps `session.id` → `chain_id`** | Read [otlp_receiver.go:445-447](../../../internal/observatory/otlp_receiver.go): `chainID` comes only from `ailang.chain_id` / `chain_id` | **Confirmed absent** |
| V2 | Broadcast really delivers our chain id as `session.id` | Live v0.33.1 probe span: `session.id = m4-broadcast-probe-session`, `chain_id` **absent** | Confirmed, measured |
| V3 | `chain_stages` already has `session_id`, and `sessions` carries `chain_id`+`stage_id` | `store_chains.go:583`, `models_chains.go:94`, `store_sessions.go:88` | **Exists** — reuse target, no new mechanism |
| V4 | **No local→cloud sync of observatory data exists** | `grep -rn "chain.*sync\|sync.*chain"` → single hit was `sync.RWMutex` (`store.go:25`), a false positive | **Confirmed absent** |
| V5 | The Firestore store satisfies the same `Backend` interface and implements chains/stages/sessions/spans | `var _ obs.Backend = (*ObservatoryStore)(nil)` (`firestore/observatory.go:16`); `CreateStage`, `UpdateStageStatus`, `CreateSpan`, `GetSession` all present | **Exists** — cloud route is wiring, not porting |
| V6 | The mission stage writer's location | First grep of `internal/mission/` was **empty** — the writer is `PostIteration` in [`internal/observatory/iteration_post.go`](../../../internal/observatory/iteration_post.go), entered via `ailang chains post-iteration` ([`cmd/ailang/chains_post.go:91`](../../../cmd/ailang/chains_post.go)) | **LOCATED.** The first search was a false negative from guessing the package; chased rather than deferred |
| V6a | **`PostIteration` never sets stage status** | Read `iteration_post.go:103-128`: `CreateStage` → `UpdateStageMetrics` → `UpdateStageEvalAssessment`; **no `UpdateStageStatus` call**. `CreateStage` defaults to `StageStatusPending` (`store_chains.go:292`) | **Confirmed absent** — this is the exact cause of every stage reading `pending` |
| V6b | Zero tokens are a CALLER defect, not a writer defect | `iteration_post.go:116` does pass `st.TokensIn`/`st.TokensOut` to `UpdateStageMetrics`, and only skips when all three are zero. Observed cost $0.0570 with 0 tokens ⇒ the caller supplied 0 | Confirmed — fix belongs at the post-iteration caller, not in `UpdateStageMetrics` |
| V7 | Mission stages are `pending` with 0 tokens and a $0.00 chain total | `ailang chains view d075f569-9e4` — 4 stages pending; quorum stages $0.0570/$0.0507 at 0 tokens; total $0.0000 | Confirmed, measured |
| V8 | Analysis reads the local store | `internal/eval_analysis/loader_chains.go:17` → `observatory.OpenDefaultStore()` → `~/.ailang/state/observatory.db` (`models.go:14-19`) | Confirmed |
| V10 | A bounded, loud, fail-soft spool ALREADY exists for exactly the "cannot reach the store" case | Read [`internal/observatory/spool.go`](../../../internal/observatory/spool.go): 100-entry / 1 MiB cap, drops oldest with a stderr notice, every buffering event warns, next iteration flushes. Its comment records that it was built to answer a quorum objection about "unbounded silent fallback" | **Exists** — the offline decision extends it rather than inventing a policy |
| V11 | Local and cloud `ailang messages` hold different populations | `AILANG_STORAGE` unset on this server (launchd global empty, with a firing control) → local store. Local: 20 messages, all `mission-*`/`eval-suite`. Cloud `/api/inbox`: 55, from `mcp-public` (28), `pkg-sunholo-*`, `coordinator`; latest 2026-08-11 | Confirmed — internal traffic here, external there, neither visible to the other. **Out of scope here**, recorded so it is not re-derived |
| V12 | The two stores are COMPLEMENTARY, not duplicates | Local: 1,716 chains / 28,231 stages / **0 spans**. Cloud: 25 chains (all `source_type: message`) / 0 stages / 193 spans | Confirmed — a sync is nearly ADDITIVE, not a merge; no source-type collision |
| V13 | Sync volume is small | 42 chains and 238 stages in the last 24h; burstiest single chain is 132 stages | Confirmed — removes the usual argument for mirroring over dual-write |
| V9 | Prod/dev observatories are Firestore, not SQLite | Both Cloud Run services set `AILANG_STORAGE=gcp` → `NewGCPBackends` → `fsstore.NewObservatoryStore`; behavioural proof: 190 spans survived a revision roll | Confirmed |

## Related Documents

**Implemented / shipped:**
- [m-openrouter-broadcast-ingest](../v0_33_1/m-openrouter-broadcast-ingest.md) — direct predecessor; shipped in v0.33.1. Made Broadcast traces *arrive correctly*; this doc makes them *joinable*.

**Planned (checked for overlap):**
- [m-mission-adaptive-multiprovider-routing](../v0_30_0/m-mission-adaptive-multiprovider-routing.md) (0.39) — distinct: that decides *which* provider a stage uses; this records what happened once it did.
- [m-cost-per-success-kpi-sprint-plan](../v1_0_0/m-cost-per-success-kpi-sprint-plan.md) (0.39) — **consumer** of this data. Its numerator is attributable metered dollars, which the mission loop cannot currently supply.

All below the 0.45 warn threshold; no duplicate-coverage rejection applies.

## Future Work

- **Dashboard refactor for mission control** (Mark's stated follow-up) — simplify the dashboard and
  drive mission loops from it, once there is adequate data behind it.
- **Cross-mission views** — v1, world and motoko loops in one comparable record.
- **Provider-cost reconciliation** — OpenRouter's own reported cost against first-party banked cost,
  now that both sit on the same chain.

---

**Document created**: 2026-08-13
**Last updated**: 2026-08-13

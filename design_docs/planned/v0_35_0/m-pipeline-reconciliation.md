# M-PIPELINE-RECONCILIATION: one pipeline definition, two execution lanes

**Status**: Planned — awaiting quorum + Mark's freeze decisions
**Target**: v0.35.0
**Priority**: P1 — every divergence here is a place the two lanes give different answers to the same question
**Estimated**: 5–7 days across three phases
**Dependencies**: M-MESSAGE-PLANE-FAIL-LOUD (landed 2026-08-26 — `triage_only_inboxes`, `execution_lane`, config CAS are all load-bearing here)
**Author**: Claude Opus 5 + Mark
**Created**: 2026-08-26

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One pipeline definition means the same input takes the same stages regardless of lane |
| A2: Replayability | 0 | No trace-format change |
| A3: Effect Legibility | 0 | No language-level effects |
| A4: Explicit Authority | +1 | Evaluation-before-approval means the human gate sees assessed work, not raw output — authority is exercised on better information |
| A5: Bounded Verification | +1 | An evaluator verdict is a local, machine-checkable artifact attached to each approval |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Stage definitions live in skills (machine-loaded), not in per-agent prose clones |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Shared model routing kills the static opus pins the cloud lane still burns |
| A10: Composability | +1 | Chain-as-data composes: a new project binds a workspace to the existing chain instead of cloning six agent entries |
| A11: Structured Failure | +1 | The evaluator verdict is a closed type — PASS(score) \| FAIL(score, reasons) \| UNAVAILABLE(reason) — and absence is unrepresentable: evaluator error, timeout, or an unparsable report emits UNAVAILABLE. FAIL and UNAVAILABLE block every *automatic* progression (auto_merge, downstream handoffs) but never block the human gate — a dead evaluator must not hide work from Mark |
| A12: System Boundary | 0 | Lane boundary already explicit via `execution_lane` (M3) |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced
- [x] A3: no hidden side effects
- [x] A4: no ambient access granted
- [x] A7: strictly improves machine-legibility of the pipeline definition

---

## Problem Statement

AILANG has one development pipeline — design → plan → execute → evaluate → human decision —
implemented **twice**, and the two copies have diverged for months:

**Lane A, the mission loop** (`.claude/skills/mission-control`, in-session, launchd): all four
stages plus quorum review, a judge, directive-channel ratification, a CI-green gate, and
cross-provider model rotation (kimi-k3/codex chains landed 2026-08-26).

**Lane B, the coordinator chain** (cloud config agents → Cloud Run Jobs):
`design-doc-creator → sprint-planner → sprint-executor` via `trigger_on_complete`, GitHub-label
approvals per stage. Frozen at roughly the v0.10 era.

Audited 2026-08-26. The divergences, each verified (see Verification Log):

1. **Lane B has no evaluation stage.** `grep sprint-evaluator` on the live config: 0 hits. Human
   approval does double duty as evaluation — every raw result lands on Mark unfiltered, and a
   failure-shaped result (`files=0`, PR 422) has already been banked `completed` (task-90d5eeef).
2. **Per-project chains are copy-paste clones.** `stapledon-*` and `twilight-*` triplets duplicate
   the chain with renamed inboxes: 6 agent entries, **zero messages ever received**, hand-edited in
   lockstep or (in practice) not at all.
3. **Model policy diverged.** Lane B statically pins `opus` on designer/executor; Lane A rotates
   across providers with fallback chains. The cloud lane burns opus with none of the routing logic.
4. **Two decision ledgers.** GitHub labels + `approvals` inbox + dashboard + CLI on one side;
   mission design-freeze ratification + directives on the other. No single "what is waiting on
   Mark" view.

### Fixed during the audit (2026-08-26) — context, not scope

- Approval requests now reach Discord (`humanTriageInbox` + `approvals`, `6345f2dc1`) — a live
  "Approval Required" had sat unread ~3h because the one decision-class inbox wasn't routed.
- The rig's Pub/Sub **sender** still pointed at the dev topic after the daemon moved to prod, so
  rig-coordinator approvals published notifications nothing watched. Sender moved to prod;
  verified delivered end-to-end (`🔔 Approval needed`, 17:33:53).
- `pkg-feedback.md` (GCS, untouched since 2026-04-28) asserted a monorepo layout that became false
  when the parse agent got its own repo. Templates now versioned in
  [tools/cloud-config/templates/](../../../tools/cloud-config/templates/) (`7f5dfac84`).
- **Narrowing correction to the audit itself:** the main pipeline agents already use
  `invoke: type: skill` — the template bypass was only ever the `pkg-*` agents. Skill definitions
  are already shared; what diverged is the *chain shape*, *model policy*, and *decision surface*.

### What must NOT merge

The two **lanes** are deliberate: attended/directive-gated (A) vs autonomous/human-gated (B), and
local-GPU vs Cloud Run (`execution_lane`, M3). Reconciliation means one *definition* consumed by
both lanes — not one lane.

---

## Goals

1. An implementation result in Lane B is machine-evaluated **before** the human sees it, and the
   verdict travels with the approval.
2. Adding a project to the pipeline is a data change (one binding), not six cloned agent entries.
3. Both lanes read one model-routing policy.
4. One queryable answer to "what is waiting on Mark right now."

---

## High-Impact Decisions

| # | Decision | Options | Who decides | Cost to change later |
|---|---|---|---|---|
| D1 | **Where does Lane B evaluation run?** The measured constraint: approval *embeds* handoffs (`daemon_tasks_exec_run.go:528`, `merge_handoff`) — with `auto_approve_handoffs: false`, a naive `trigger_on_complete: [sprint-evaluator]` fires only **after** approval, useless as a pre-filter | **(a) In-job** — *rejected by round-1 quorum: bifurcates the topology at the stage being unified*. **(b) Evaluator task pre-approval**: auto-approve only the executor→evaluator handoff; evaluator assesses the pushed branch and attaches its verdict to the *pending* approval — needs new code (attach-to-approval). **(c) Post-approval validation**: config-only, but evaluates already-merged work — a Gate-3b analogue, not a pre-filter | **Mark** | Medium — (b) adds schema; (a) is prompt-level |
| D2 | **Chain-as-data schema.** Replace per-project agent clones with one chain definition + per-project bindings (workspace/repo/lane), e.g. `chains: [{stages: [...], bindings: [{project: stapledon, workspace: ...}]}]` | shape of the config schema; whether dormant stapledon/twilight bindings are carried over or dropped until asked for | **Mark** (schema) | High once bindings exist |
| D3 | **Shared model routing.** Lift Lane A's routing table (role → model chain with fallbacks) into config both lanes read; delete the static `model:` pins from chain agents | routing file location (cloud config vs repo `models.yml` extension); whether Lane B gets fallback *chains* or just the primary | **Mark** | Medium |
| D4 | **Decision ledger.** One "pending for Mark" view spanning Lane B approvals and Lane A `awaiting_approval` stages | (a) `ailang coordinator pending` grows to read mission ledgers; (b) both write to the `approvals` inbox (now Discord-routed) as the single spine | **Mark** | Low |

### Design Freeze

- [ ] D1 evaluation placement
- [ ] D2 chain-as-data schema
- [ ] D3 routing-table location
- [ ] D4 ledger spine

**Recommendation**: D1(**b**). Round-1 quorum (2026-08-26) rejected (a) on topology grounds —
correctly: embedding evaluation as a prompt directive inside Lane B's executor while Lane A runs
it as a discrete stage *maintains two pipeline topologies at exactly the stage this doc exists to
unify*. (b) keeps evaluation a discrete stage in both lanes; the attach-to-approval code is the
price of one topology, and Phase 2's chain definition then formalizes the same stage list for
both. D4(b) — the `approvals` inbox just became the notification spine; make it the ledger too.

---

## Solution Design

### Phase 1 — evaluation before approval (D1) — ~1 day if (a)

Per D1(a): the sprint-executor **cloud directive** (not the shared skill — Lane A runs the
evaluator itself and must not double-run) gains a final step: run the sprint-evaluator skill
against the worktree, write the scored verdict into the completion payload. The approval request
then carries `evaluation: PASS 84/100` (or the failure detail) into the `approvals` inbox →
Discord. A FAIL does not block — it informs; blocking thresholds are a later knob.

Note: the `ailang_bootstrap` plugin lacks `sprint-evaluator` (verified — plugin has 7 skills,
evaluator absent), but cloud jobs clone the repo, whose `.claude/skills/sprint-evaluator` is
present. Syncing it to the plugin is an AC for hygiene regardless.

### Phase 2 — chain-as-data (D2) — ~2 days

Registry-level change: a `pipelines:` section with one stage list and per-project bindings.
`AgentRegistry` expands bindings into the same in-memory `AgentConfig`s it builds today, so
nothing downstream changes. The stapledon/twilight sextet is deleted; their bindings are added
back only when those projects wake (they have received zero messages ever).

### Phase 3 — shared routing + ledger (D3, D4) — ~2 days

Routing: one table (role → ordered model chain) in the config the CAS tool manages; Lane A's
role-env derivation and Lane B's `model:` pins both resolve through it.
Ledger: every Lane A `awaiting_approval` stage also posts to the `approvals` inbox; `ailang
coordinator pending` reads that inbox as its spine. One query, one Discord channel, one answer.

### Files to Modify/Create

- `internal/coordinator/agent_registry.go` — `pipelines:` expansion (~80 LOC)
- `internal/coordinator/agent_config.go` — schema (~40 LOC)
- `tools/cloud-config/templates/` or agent invoke directives — Phase 1 evaluation step (~30 LOC)
- `internal/coordinator/daemon_tasks_exec_run.go` + approval store — `evaluation` field on the
  approval request, updatable by correlation id after creation (~60 LOC)
- `internal/coordinator/daemon_approval.go` — per-edge auto-approve for the read-only
  executor→evaluator handoff (~30 LOC)
- `.claude/skills/mission-control/SKILL.md` — routing table externalization (Gate 3 edit)
- cloud config via `ailang coordinator config set` — chain bindings, routing table
- `ailang_bootstrap` — sync sprint-evaluator skill (PR to that repo)

---

## Success Criteria

- [ ] A Lane B approval request carries an evaluator verdict in its payload (visible in Discord)
- [ ] Verdict is the closed type; killing the evaluator mid-task yields `UNAVAILABLE` on the approval (test)
- [ ] `FAIL`/`UNAVAILABLE` block auto_merge and downstream handoffs; the approval still reaches the human
- [ ] Lane A does not double-evaluate
- [ ] Adding a project = one binding entry; `stapledon-*`/`twilight-*` clones deleted
- [ ] Both lanes resolve models through one table; no static `model:` pins on chain agents
- [ ] `ailang coordinator pending` lists Lane A and Lane B items awaiting Mark
- [ ] Fleet fixture test extended: bindings expand to the same effective agents as today's clones

## Testing Strategy

- Unit: binding expansion (registry), routing resolution, approval-payload evaluation field
- Fixture: extend `testdata/live_cloud_config_20260826.yaml` regression — post-migration config
  must expand to a superset of today's effective agents
- End-to-end: replay one parse ticket ([ailang-parse#19](https://github.com/sunholo-data/ailang-parse/issues/19),
  the `content` feature — smallest) through Lane B; the approval must arrive in Discord carrying
  an evaluator verdict

## Non-Goals

- Merging the lanes (deliberately distinct)
- Touching Lane A's quorum/judge/directive machinery (works, keeps evolving in-skill)
- The `pkg-*` cascade flow (template now versioned; cascade repair is its own directive by design)
- Firestore-backed cross-host `workers list` (known limit, separate)

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| D1(a) makes executor jobs longer/costlier | Evaluation is read-only + cheap model via D3 routing; cap via existing job timeout |
| Binding expansion silently changes an effective agent | Fixture test asserts expansion ≡ today's config before the clones are deleted |
| Two ledgers drift during migration | D4 lands last, after both writers exist |
| Shared routing table becomes a new single point of misconfig | It lives in the CAS-managed object — stale writes refused, YAML validated pre-write |

## Verification Log

Run 2026-08-26 at HEAD `7f5dfac84`; config generation 1787758575840728.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | Lane B has no evaluator stage (**negative**) | `grep -c sprint-evaluator` live config | Confirmed — 0 |
| V2 | Approval embeds handoffs → naive chain evaluates post-approval | read `daemon_tasks_exec_run.go:523-548` | Confirmed — `merge_handoff` built when `!AutoApproveHandoffs` |
| V3 | Main pipeline agents already `invoke: type: skill` | read live config entries | Confirmed — all three; **narrows the audit's F3** |
| V4 | stapledon/twilight: 6 entries, zero traffic | config grep + 400-message census by inbox | Confirmed — 0 messages ever |
| V5 | Static model pins in Lane B | live config | Confirmed — `opus` ×2, planner default |
| V6 | Plugin lacks sprint-evaluator (**negative**) | `gh api ailang_bootstrap/contents/skills` | Confirmed — 7 skills, no evaluator |
| V7 | Plugin skills are in sync with repo (not a fork risk) | byte-compare design-doc-creator + sprint-planner | Confirmed — identical sizes, plugin updated same-day |
| V8 | Approvals now reach Discord | live send → daemon log | Confirmed — `🔔 Approval needed` 17:33:53 |
| V9 | Rig pubsub sender was dev-pinned (approval class never notified) | 14:13 approval absent from daemon delivery log; config block read | Confirmed — fixed, verified via V8 |
| V10 | `pkg-update.md` is genuinely template-shaped (wrapper variables, no skill equivalent) | read template; grep skills | Confirmed — kept as-is |

## Quorum Record

- **Round 1 (2026-08-26T15:36Z): BLOCKED** — artifact
  `.ailang/state/mission-quorum/m-pipeline-reconciliation-2026-08-26T15-36-46Z.json`.
  - `gpt5-6-sol`: A11 claimed FAIL blocks while Phase 1 said "informs"; evaluator
    timeout/error/unparsable behavior undefined. **Accepted** → closed-type verdict with
    `UNAVAILABLE`, automatic-progression blocking, human gate never blocked.
  - `gemini-3-1-pro`: recommending D1(a) maintained two pipeline topologies at the stage under
    unification. **Accepted** → recommendation flipped to D1(b); (a) marked rejected.
- Round 2: see below (re-quorum ONCE per the guardrail).

## Related Documents

- [m-message-plane-fail-loud.md](./m-message-plane-fail-loud.md) — the substrate (triage_only,
  execution_lane, config CAS); landed 2026-08-26
- [m-coordinator-inbox-wildcards.md](../v0_29_0/m-coordinator-inbox-wildcards.md) — the same
  "config clones → one pattern" move, for pkg inboxes; D2 is its chain-level analogue
- [m-pkg-feedback-loop.md](../v0_29_0/m-pkg-feedback-loop.md) — happy-path validation of the pkg
  flow; distinct scope
- `docs/internal/message-plane-topology.md` — lane topology this doc keeps intact

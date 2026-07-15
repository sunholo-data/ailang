# M-MISSION-AGENTIC-PROVIDER-ROUTING: enforce per-role AGENTIC cross-provider routing + right-size the roles

**Status**: Planned — **M1a LANDED 2026-07-15** (interactive, main checkout: enforcement + Anthropic
per-role pinning; the Fable drain is closed on-disk, pending live verification). M1b/M2/M3 queued for
the loop. (Requested by Mark 2026-07-15 — "design doc this and put it top of our queue"; the
concrete, sprint-executable slice of the fleet doc's Phase C+E.)
**Target**: v0.30.x — mission infrastructure
**Priority**: **P0 (mission-infra critical path).** Quota is the binding constraint on running the
loop at all (Mark, 2026-07-14); this closes the bug that makes 100% of every iteration bill one
model, and lays the enforced routing layer everything downstream needs.
**Estimated**: ~2–3d (M1 enforcement ~1–1.5d; M2 table+instrumentation ~0.5–1d; M3 planner A/B ~0.5–1d)
**Dependencies**: `internal/coordinator/provider_executor.go` (the agent registry — claude/codex/
motoko/managed_agents/opencode/pi); `internal/ai/routing.go` (`AIRoutingPolicy`); the shipped
quorum (`internal/mission/quorum/*`); `.claude/skills/mission-control/SKILL.md` (Gate 3 routing)
**Concretizes**: [m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md)
Phase C (cross-provider executors) + Phase E (assignment table). That doc stays the strategic
umbrella; THIS doc is its executable sprint. **No new plumbing is invented here** (redundancy guard,
Mark 2026-07-14) — it wires selection policy onto the existing registry.
**Author**: Opus session, requested by Mark 2026-07-15

---

## Problem statement (three defects, all repo-verified 2026-07-15)

### 1. The routing table is prose — no code enforces it (the Fable-drain bug)
The charter's model-routing table (`v1-mission.md` §"Model routing policy": planner=Opus,
executor=Opus, evaluator=Fable, controller=Fable) is **documentation only**. Verified chain:
- The driver `tools/launchd/mission-control.sh:224` launches `claude -p --model "$MODEL"` — **one**
  session model (Phase A resolves it Fable-first).
- mission-control invokes the inner skills (design-doc-creator / sprint-planner / sprint-executor /
  sprint-evaluator) **inline via the Skill tool** → they run on the **session** model.
  sprint-executor's own parallel sub-agents (`.claude/skills/sprint-executor/SKILL.md:343-345`)
  spawn `subagent_type="general-purpose"` with **no `model=`** → inherit the session model too. A
  grep across all five skills for any model pin = **zero hits**.
- ∴ every role bills whatever `--model` the driver passed. Since the controller reverted to Fable
  2026-07-14, **100% of every iteration billed Fable** — gauge: Weekly-Fable ≈ Weekly-all-models
  ≈ 51%, Opus ≈ 0. (Inverse during the 2026-07-11→14 Opus-TEMP window: everything on Opus,
  including the controller/evaluator meant to be Fable. **The table has never matched reality in
  either direction.**)

### 2. Routing must target AGENTS, not single API calls (Mark 2026-07-15)
AILANG has two cross-provider surfaces and they are **not** interchangeable:
- **`std/ai` single-call** (`AIRoutingPolicy` on `ai.Request`, string→string / JSON, one shot) —
  what the quorum uses today (`run.go:120 CallJSON`). It *reasons*; it cannot *verify* or *build*.
- **`provider_executor.go` agentic executors** (`claude-code · codex(OpenAI) · motoko ·
  managed_agents(gemini) · opencode · pi`) — tool-using agents that edit files, run `make test`,
  iterate multi-turn.

Every **execution / evaluation / planning** role binds to the **agentic** surface. The routing
unit is therefore a **`(provider, agent-harness, model)` triple**, not a bare model tier — because
a specialist agent (**motoko** on AILANG tasks) can beat a bigger general model cheaply, and a
model-tier-only comparison would never surface that. Naming the four mission agents explicitly:
**motoko, claude, codex, managed_agents(gemini)**.

### 3. Every role defaults to the strongest model (over-provisioning)
Roles inherit the controller model regardless of what the role needs. Right-sizing review
(2026-07-15): **sprint-planner is the #1 down-tier candidate** — the charter justified Opus-planning
with "plan quality determined execution success," but that measurement **predates the quorum**. The
design doc now arrives adversarially reviewed by three providers; the poisons-everything reasoning
has moved **upstream**. What's left for the planner is structured decomposition against a solid
spec — strong instruction-following, not deep synthesis.

## Design

### The enforcement invariant (fixes defect 1)
Routing is decided in **one** place and **enforced by code**, never inherited from a single session
flag. mission-control selects a `(provider, agent, model)` per inner role from the charter's
assignment table and spawns each role **pinned** — a model-pinned sub-agent (Task with explicit
`model=` / `subagent_type=`) for same-provider roles, or a `provider_executor` agent for
cross-provider roles. The lightweight controller session (triage/pick/judge/retro) keeps its own
model; the heavy roles no longer ride it. **This alone ends the single-model drain.**

### Agentic, cross-provider (fixes defect 2)
Execution/eval routes through the `provider_executor.go` registry (reuse — codex landed v0.22.0,
managed_agents replaces gemini-cli). The controller passes the doc/plan + scope to the chosen
agent; the agent returns its worktree branch + report exactly as the claude executor does today.
Review-side agentic verification is the sibling doc
[m-mission-quorum-agentic-verify](m-mission-quorum-agentic-verify.md); this doc is the
execution/planning side. Generator≠judge is preserved: the evaluator agent MUST be a different
provider from the executor.

### Right-sizing table — the initial hypothesis (fixes defect 3, seeds Phase E)
Recorded in the charter, updated by evidence (never by vibes — the charter's ≥3-datapoint rule):

| Role | Agentic? | Needs | Tier hypothesis | Agent candidates |
|---|---|---|---|---|
| Controller (pick/judge/retro) | agent (claude-code) | orchestration judgment | **mid** | claude-code (home harness) |
| Design-doc-creator | agent (`check` in loop) | deep spec reasoning (highest leverage) | **strong** | strong claude/codex + live quorum |
| **Sprint-planner** | agent-capable | decompose a quorum-reviewed doc | **MID (down-tier)** | mid codex/gemini/motoko |
| Sprint-executor | AGENT (heavy) | tool-using coding | **strong AGENT** (not just model) | **codex / motoko / claude**; motoko may over-perform on AILANG |
| Sprint-evaluator | AGENT (re-runs tests) | behavioral verification | **mid**, distinct provider from executor | gemini/codex ≠ executor |
| Mechanical (moves/regen) | no | deterministic | **low / local** | local-GPU (Phase D) |

### Evidence instrumentation (makes the table data-driven)
Extend the log's routing-evidence rows from `(model, task-class, round-1 score, rounds, corrections)`
to **`(provider, agent, model, task-class, round-1 score, rounds, corrections, $/quota)`**. These
rows are the seed data for the Phase E assignment table — the same rows the charter already writes,
now provider/agent/cost-aware.

## Milestones (sprint-sized)
- **M1a — enforcement + Anthropic per-role pinning (the bug fix) — ✅ LANDED 2026-07-15 (interactive,
  main checkout):** the driver exports `$MISSION_PLANNER_MODEL`/`$MISSION_EXECUTOR_MODEL`/
  `$MISSION_EVALUATOR_MODEL` (in-session Agent-tool aliases; defaults `opus`/`opus`/`fable`), and
  mission-control Gate 3 spawns each heavy role as a model-PINNED `Agent` sub-agent — never inline.
  sprint-executor's own Task spawns pin `model` too. The controller session keeps `$MODEL`. Net:
  execution now bills **opus**, controller+eval bill **fable** → the 100%-Fable drain is closed and
  generator≠judge (fable evaluator ≠ opus executor) is restored. Done interactively, not via the
  loop, because these are the loop's own driver+skill and MUST edit the MAIN checkout (launchd reads
  on-disk, not a worktree — memory `project-mission-unbounded-wait…`). **NOT yet verified LIVE** (loop
  paused for quota): the first real iteration is the acceptance test — confirm the start-log `roles:`
  line and that Gate 4 records the actual (role, model).
- **M1b — one NON-CLAUDE agent executor (the true cross-provider proof) — loop / fleet Phase C:**
  wire a `provider:model` executor (codex OR motoko) through `provider_executor.go` so the skill
  branches a non-alias env value to the registry. Acceptance: an iteration where controller and
  executor run on **different PROVIDERS**. (M1a proved different *models*, same provider; M1b is the
  cross-provider step that actually moves burn OFF Anthropic — metered API $ or the local-GPU lane.)
- **M2 — right-sizing table + evidence instrumentation:** table into the charter; routing-evidence
  rows extended with provider/agent/cost. Acceptance: one iteration writes a full new-schema row.
- **M3 — planner down-tier A/B:** plan the same 3 quorum-reviewed docs with a mid-tier planner-agent
  vs the Opus planner; compare executor round-1 scores + corrections; record verdict in the table.
  Acceptance: a data-backed keep/down-tier decision for sprint-planner (not an assertion).

## Conflict surface
Touches: the mission-control SKILL (how Gate 3 spawns the inner loop — from inline Skill to pinned
agent selection); `provider_executor.go` (read-only consumer — additive); the charter routing table
+ evidence-row schema. **Must NOT**: duplicate fleet Phase C plumbing (reuse the registry); break
generator≠judge (evaluator provider ≠ executor provider); route agentic roles through the `std/ai`
single-call path; let cost-optimization silently lower a role below its measured floor (any
down-tier requires the ≥3-datapoint evidence rule, both directions).

## Non-goals
- Full Phase E across all task classes (this seeds the table + does the planner A/B; the rest
  accrues as evidence lands).
- The local-GPU lane (fleet Phase D).
- Agentic *review* verification (sibling doc, sequenced after this).
- Retiring the `std/ai` single-call path (it stays — cheap Tier-1 review, non-agentic calls).

## Cost note (honest)
Within Anthropic, moving Fable→Opus only rebalances two weekly buckets that both reset Monday — it
protects the Fable bucket but does not cut total Anthropic burn. The **real** relief is routing
execution to **non-Anthropic** agents (codex/gemini = metered OpenAI/Google API $, a different pool
than the Claude subscription) or the local-GPU lane (free, slower). M1 is what unlocks that.

## Verification log
| Claim | Method | Result |
|---|---|---|
| Driver passes one `--model` to the whole session | `tools/launchd/mission-control.sh:224` | Confirmed |
| No inner skill pins a model | grep `model=`/`--model`/`"model"` across all 5 skills | Zero hits |
| Executor sub-agents omit `model=` | `sprint-executor/SKILL.md:343-345` (`subagent_type="general-purpose"`) | Confirmed |
| 100% Fable burn | quota gauge (Weekly-Fable ≈ all-models ≈ 51%) + `mission-model-last`=claude-fable-5 | Confirmed |
| Cross-provider agents already integrated | `provider_executor.go:11-12` (codex, managed_agents imports) | Confirmed |
| Quorum fired LIVE cross-provider | `.ailang/state/mission-quorum/m-dx-examples-coverage-2026-07-14…json` — gpt5-6-sol + gemini-3-1-pro + Claude verdicts | Confirmed (2 reject / 1 pass) |

## Related
- [m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md) — the umbrella (this = Phase C+E executable)
- [m-mission-quorum-agentic-verify](m-mission-quorum-agentic-verify.md) — the review-side sibling
- Memory: `project-mission-routing-table-never-enforced` (the bug), `project-v1-mission-outer-loop`

---
**Document created**: 2026-07-15

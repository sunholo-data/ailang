# Triage Mission — every card, PR and issue the fleet opens reaches a decision

<!--
  From design_docs/mission-charter-TEMPLATE.md. Design: design_docs/planned/m-triage-mission.md
  (drafted attended 2026-10-15). Iteration 0 = ratify this charter with Mark, attended.
-->

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md), [fleet-mission.md](fleet-mission.md)),
advanced by a scheduled outer loop on the always-on rig. **It fires only when the queue changed.**
The driver exits before any probe when the queue fingerprint equals the last digested one.
**North star**: no approval card, agent PR or bot-filed issue on `sunholo-data/ailang` sits
undecided. Each one is verified against `origin/dev` and either acted on (within the current level) or put in
front of Mark with the evidence.
**Traces to**: [PROGRAM.md](PROGRAM.md) and [M-TRIAGE-MISSION](planned/m-triage-mission.md).
Measured basis (2026-10-15): 33 open PRs (29 agent), 70 open issues (69 bot-authored), 21 pending
cards, `dev` CI red 12/12.
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md), the SAME
unforked skill, with a triage branch at Gate 2 and Gate 3 (`resources/gate-3-triage.md`).
**Scheduling**: launchd `dev.ailang.mission-triage`, registry entry `missions/triage.toml`
(interval 12h, boot offset 2520s). Billing guard as for every mission.
**Log**: [triage-mission-log.md](triage-mission-log.md), append-only, one entry per iteration.
**Human-facing reporting**: the GitHub bookkeeping issue (created at bootstrap), **Discord** via
the `user` inbox, and **Daneel** via `daneel-reports`. See "Communication".

## Repo Profile

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/triage-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `triage` (`~/.ailang/state/mission-triage-*`)
- **Checkout**: the pin worktree `~/.ailang-driver-pin/triage`. The clone
  `~/dev/sunholo-data/ailang-triage` is only launchd's working directory. The mission writes
  nothing to the repo except its own charter, log and dashboard, through a PR.
- **Bookkeeping issue**: `#<created at bootstrap>`, rotates weekly; live number in `~/.ailang/state/mission-triage-gh-issue`
- **CI workflows Gate 1 polls**: `CI` (the `dev`-green precondition for every action above Level 0)
- **Verify profile**: none of its own. It verifies other agents' work with the
  `github-issue-triage` recipe and `gh` reads.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `triage-mission-status-archive.md`.

## STATUS 2026-10-15 — ITERATION 0: charter drafted, awaiting ratification (D-TRIAGE-1)

## CURRENT LEVEL: **0 — Observe** (Mark, attended 2026-10-15)

The level changes only by a recorded ruling in the ledger below. The loop may **demote** itself (G8)
but never promote.

## The bar

- **Clause 1, coverage**: every pending card, open agent PR and open bot-filed issue carries a
  verdict less than 7 days old, with evidence read against `origin/dev`.
- **Clause 2, no silent stall**: no green, mergeable, `diff_available` card older than 48h is
  missing from the digest's decisions section.
- **Clause 3, faithful shadow**: at Level 0, every would-be action is in the shadow ledger, and
  each digest scores the ledger's agreement with what later happened to those items. This is the
  promotion evidence.
- **Clause 4, zero harm**: no action above the current level; no label from the coordinator's
  `watch_labels` (`bug`, `feature`, `from:*`); no approval that breaks a G-guard.
- **Clause 5, idle is free**: a fire with an unchanged queue spends no inference.

## Guardrails

The consequences behind these are in the design doc ("What approving and merging actually do").
In short:

- **Approving is merging, and merging is approving.** Approve dispatches the next chain stage,
  then squash-merges. Hand-merging a coordinator PR makes the landed-card sweep approve it with
  dispatch. Neither has a "just one of the two" form.
- **The authority gate in the CLI is advisory.** This charter enforces G1–G9; the code does not.
- **Remote reject is final.** The chain fails, the PR closes, and the agent never sees the feedback (Level 3 only).
- **Every merge redeploys dev** (16 images, coordinator included). Prod is tag-gated.
- **Never** touch Dependabot majors, other missions' PRs (`fleet/*`, …), or prod. They are reported, not acted on.
- **Red `dev` is not this mission's to fix** (V1 owns it; see D-TRIAGE-2). It blocks every action above
  Level 0 and leads the digest.
- **It routes, never fixes**: defects → `design-doc-creator`, harness → `ailang mission ticket`, language
  gaps → V1 backlog.

### Levels

| Level | Admits |
|---|---|
| 0 Observe | verify · label (`priority:*`, `area:*`, `needs-decision`) · evidence comments · digest · shadow ledger |
| 1 Tidy | + close verified-fixed/obsolete/duplicate issues with SHA evidence · `coordinator prs --landed --apply` (no handoffs) |
| 2 Approve | + `coordinator approve --remote gcp --yes --by mission-triage/<controller>` under **G1–G9** |
| 3 Reject | + reject cards / close superseded agent PRs, after two unanswered digests and judge concurrence |

**G1–G9** (Level 2): G1 `dev` green · G2 every check on the PR head SUCCESS/SKIPPED · G3 MERGEABLE
and not BEHIND · G4 `diff_available` and `agent_actionable` · G5 handoff targets in the digest
first, ≤2 approvals/iteration and ≤4/day · G6 no `std/` without goldens, and nothing touching
`internal/coordinator/`, `cloudbuild*.yaml`, `tools/launchd/`, `.github/workflows/`, `missions/` ·
G7 bounded 30-min wait for the merge, orphan → `[urgent]` · G8 first post-merge `dev` CI red →
`[urgent]` and self-demote to Level 1 · G9 `--by` names mission and controller.

## Communication

- **Bookkeeping issue**: full digest + shadow ledger, every iteration.
- **Discord** (`ailang messages send user`, prod store): one short post, **only** when there is a
  decision for Mark, an orphan, a red `dev`, or a level change. Counts, top 3, link.
- **Daneel** (`ailang messages send daneel-reports`, prod store): digest as `[normal]` (morning
  brief); `NEEDS-DECISION` / orphan / red-`dev` as `[urgent]` (one mail within the hour; Daneel caps
  this at 5/day, so the loop sends at most one urgent per iteration, batched).

## Routing policy

From `~/.config/ailang/mission-triage.env` (versioned copy in `tools/launchd/mission-env/`):

- **Controller**: `codex:gpt-6.1-sol`, then `claude-sonnet-5-5`
- **Verifiers** (high volume, read-only): `codex:gpt-6-luna`, then `pi:ollama/glm-5.3-flash:cloud`, then
  `pi:openrouter/z-ai/glm-5.3-flash`
- **Judge**: `sonnet`, then a vendor-disjoint ladder (`minimax-m3` cloud, then OpenRouter). Generator ≠ judge
  at every rung.
- **Metered ceiling**: $1 per fire.

## Decision ledger

<!-- decision-ledger:start -->
| ID | Status | Decision / recorded answer | Evidence |
|---|---|---|---|
| D-TRIAGE-0 | RESOLVED | Start at **Level 0**. The loop is meant to use the approval system and merge "as appropriate", with Discord and Daneel facilitating. (Mark, attended 2026-10-15.) | This session. |
| D-TRIAGE-1 | OPEN | Ratify bar, levels, G1–G9 and the communication plan (iteration 0). Recommendation: ratify as drafted. | Charter bootstrap 2026-10-15. |
| D-TRIAGE-2 | OPEN | Red `dev` has no owner while V1 is paused (12/12 red on 2026-10-15). Options: (a) an attended fix now, before arming; (b) triage owns red `dev` while V1 is paused, routing the fix to `design-doc-creator`; (c) leave it with V1. Recommendation: (a), then (b) as the standing rule. | `gate-1-observe.md:233-271`; `mission-control.disabled`. |
| D-TRIAGE-3 | OPEN | Discord threshold: post only on decision/orphan/red/level-change (drafted), or one post every iteration. Recommendation: as drafted. | Design doc "Communication". |
| D-TRIAGE-4 | OPEN | Commission the Daneel **ruling relay** (reply `approve 1747` → a directive for this loop, closed-set words, rung 2) via the `daneel-design` inbox. Recommendation: yes, after two weeks of Level 0 digests show which words you actually use. | `daneel/authority.md:44,66`; axiom D8. |
<!-- decision-ledger:end -->

## Queue

1. [NEXT] Bootstrap sprint: toml, env, idle pre-check, prompt switch, Gate 2/3 triage resources,
   tests, dry runs (design doc "Driver and skill changes"). Clause 5 and all clauses.
2. First armed iteration at Level 0, attended. Clauses 1–3.

---
**Document created**: 2026-10-15. Iteration 0 ratifies it with Mark before the loop is armed.

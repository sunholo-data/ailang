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
**Human-facing reporting**: the GitHub bookkeeping issue ([#1835](https://github.com/sunholo-data/ailang/issues/1835)), **Discord** via
the `user` inbox, and **Daneel** via `daneel-reports`. See "Communication".

## Repo Profile

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/triage-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `triage` (`~/.ailang/state/mission-triage-*`)
- **Checkout**: the pin worktree `~/.ailang-driver-pin/triage`. The clone
  `~/dev/sunholo-data/ailang-triage` is only launchd's working directory. The mission writes
  nothing to the repo except its own charter, log and dashboard, through a PR.
- **Bookkeeping issue**: `#1835` (created at bootstrap 2026-10-15), rotates weekly; live number in `~/.ailang/state/mission-triage-gh-issue`
- **CI workflows Gate 1 polls**: `CI` (reported at the top of every digest; no loop action depends on it)
- **Verify profile**: none of its own. It verifies other agents' work with the
  `github-issue-triage` recipe and `gh` reads.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `triage-mission-status-archive.md`.

## STATUS 2026-10-15 — ITERATION 0: charter RATIFIED (D-TRIAGE-1, Mark attended); D-TRIAGE-2/3/4/8/9 ruled as recommended. Next: Sprint B (the loop) and the Part A design doc.

## CURRENT LEVEL: **1 — Tidy, with attestations** (Mark, attended 2026-10-15; D-TRIAGE-7)

This loop **never approves, rejects or merges**. Approving belongs to the coordinator. Its
auto-approval policy (design doc Part A) reads this loop's attestations as one clause; until that
ships, attested items go to Mark as "ready to approve". The level changes only by a recorded ruling.

## The bar

- **Clause 1, coverage**: every pending card, open agent PR and open bot-filed issue carries a
  verdict less than 7 days old, with evidence read against `origin/dev`.
- **Clause 2, no silent stall**: no green, mergeable, `diff_available` card older than 48h is
  missing from the digest's decisions section.
- **Clause 3, faithful shadow**: every verdict is in the shadow ledger, and each digest scores the
  ledger's agreement with what later happened to those items. This is the evidence Part A's
  policy is scored against before it auto-approves anything.
- **Clause 4, zero harm**: no action above the current level; no label from the coordinator's
  `watch_labels` (`bug`, `feature`, `from:*`); no approve, reject or merge by this loop.
- **Clause 5, idle is free**: a fire with an unchanged queue spends no inference.

## Guardrails

The consequences behind these are in the design doc ("What approving and merging actually do").
In short:

- **Approving is merging, and merging is approving.** Approve dispatches the next chain stage,
  then squash-merges. Hand-merging a coordinator PR makes the landed-card sweep approve it with
  dispatch. Neither has a "just one of the two" form.
- **So this loop does not approve.** Pressing `approve` from a loop would be a second approver
  outside the coordinator, re-implementing its policy as prose (D-TRIAGE-7). Approval is the
  coordinator's job: Part A of the design doc.
- **Every merge redeploys dev** (16 images, coordinator included). Prod is tag-gated.
- **Never** touch Dependabot majors, other missions' PRs (`fleet/*`, …), or prod. They are reported, not acted on.
- **Red `dev` is owned here while V1 is paused** (D-TRIAGE-2): it leads the digest and the fix is
  routed to `design-doc-creator` (or a fleet ticket if it is harness). The loop does not fix it
  itself. Ownership returns to V1 when V1 is re-armed.
- **It routes, never fixes**: defects → `design-doc-creator`, harness → `ailang mission ticket`, language
  gaps → V1 backlog.

### What the loop may do (Level 1)

- Verify every card, agent PR and bot-filed issue against `origin/dev`; label (`priority:*`,
  `area:*`, `needs-decision`); evidence comments; digest; shadow ledger.
- **Attest** `VERIFIED` PRs: a machine-readable PR comment carrying the verdict, head SHA, judge model
  and evidence link (carrier: D-TRIAGE-9). A new push to the PR invalidates it.
- Close verified fixed/obsolete/duplicate issues, and resolve landed cards, under **T1–T4**.

**T1–T4**:
- **T1** The judge (a different vendor from the verifier) concurs, and the close comment cites the fix commit or reason and says "verified against origin/dev <sha>". The same applies to every attestation.
- **T2** Never close an issue that still has a pending card or an open `coordinator/task-*` PR. The coordinator imports only OPEN issues (`internal/messaging/github.go:397`), and closing one cancels nothing, so its chain would keep running against a closed issue. Those go in the digest instead.
- **T3** Never close the current bookkeeping thread; at most 10 closes per iteration.
- **T4** `prs --landed --apply` ends a landed card's chain WITHOUT starting the next stage. Every card resolved that way is listed in the digest with its handoff targets, so a chain worth continuing is visible to Mark (who can `--fire-handoffs` attended).

None of these needs a green `dev`: they merge nothing.

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
| D-TRIAGE-7 | RESOLVED | **Split** (Mark, attended 2026-10-15: "yes revise the design doc", answering whether the loop "is just replacing the coordinator's job"). Approving moves to a coordinator auto-approval policy (design doc Part A). This loop runs at Level 1, writes attestations the policy reads, and never approves, rejects or merges. Supersedes D-TRIAGE-6. | Design doc "Why two parts". |
| D-TRIAGE-8 | RESOLVED | Where Part A's policy lives: (a) an AILANG module the coordinator loads; (b) extending `auto_merge_code` in Go. Recommendation: (a), per "policy decisions belong in AILANG, Go is the shell".  **ANSWERED — (a) Part A's policy is an AILANG module the coordinator loads; Go is the shell.** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| Design doc Part A item 4.  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
| D-TRIAGE-9 | RESOLVED | Attestation carrier: (a) a machine-readable PR comment (verdict, head SHA, judge, evidence); (b) a PR label; (c) a field on the approval card. Recommendation: (a). It carries a SHA and an author, and the coordinator's `gh` client already reads PRs.  **ANSWERED — (a) Attestation carrier is a machine-readable PR comment carrying verdict, head SHA, judge model and evidence link.** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| Design doc Part B.  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
| D-TRIAGE-6 | RESOLVED | (Superseded by D-TRIAGE-7.) Start at **Level 2 (Approve)**, superseding D-TRIAGE-5. Mark, attended 2026-10-15: "honestly level 2 looks ok for the loop". Under G1–G9: at most 4 approvals a day, each dispatching a next stage capped by `task_max_cost` ($25 global / $15 claude), so the worst case is about $100 a day of cloud-plane spend. G1 (`dev` green) gates it today. | This session; design doc "What approving and merging actually do". |
| D-TRIAGE-5 | RESOLVED | Start at **Level 1 (Tidy)**, superseding D-TRIAGE-0's Level 0. Mark, attended 2026-10-15: "I think we can start at level 1". Guards T1–T4 added in the same commit. | This session. |
| D-TRIAGE-1 | RESOLVED | Ratify the bar, the Level 1 scope with T1–T4 and attestations, and the communication plan (iteration 0). Recommendation: ratify as drafted.  **ANSWERED — RATIFIED as drafted: the bar, the Level 1 scope with T1-T4 and attestations, and the communication plan.** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| Charter bootstrap 2026-10-15.  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
| D-TRIAGE-2 | RESOLVED | Red `dev` has no owner while V1 is paused (12/12 red on 2026-10-15). Options: (a) an attended fix now, before arming; (b) triage owns red `dev` while V1 is paused, routing the fix to `design-doc-creator`; (c) leave it with V1. Recommendation: (a), then (b) as the standing rule.  **ANSWERED — (b) While V1 is paused, this loop owns red dev: it leads the digest and the fix is routed to design-doc-creator (or a fleet ticket if harness). When V1 is re-armed, ownership returns to V1. The 2026-10-15 red was fixed attended first (8e9dee07e, 487b20dcf, 95106ba27).** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| `gate-1-observe.md:233-271`; `mission-control.disabled`.  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
| D-TRIAGE-3 | RESOLVED | Discord threshold: post only on decision/orphan/red/level-change (drafted), or one post every iteration. Recommendation: as drafted.  **ANSWERED — As drafted: Discord posts only on a decision for Mark, an orphan, a red dev, or a level change; never a routine per-iteration post.** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| Design doc "Communication".  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
| D-TRIAGE-4 | RESOLVED | Commission the Daneel **ruling relay** (reply `approve 1747` → a human approval through the coordinator's normal path, or `hold 1723` → a directive for this loop; closed-set words, rung 2) via the `daneel-design` inbox. Recommendation: yes, after two weeks of Level 0 digests show which words you actually use.  **ANSWERED — YES, commission the Daneel ruling relay after two weeks of digests show which reply words are used. With D-TRIAGE-7, approve <n> becomes a human approval through the coordinator's normal path and hold <n> a directive for this loop. Queued; filed on the daneel-design inbox when the two weeks are up.** (Mark Edmondson, attended 2026-10-15, recorded directly in this ledger.)| `daneel/authority.md:44,66`; axiom D8.  **Attended ruling 2026-10-15** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark, attended 2026-10-15: "I agree with all decisions. please action".|
<!-- decision-ledger:end -->

## Queue

1. [DONE 2026-10-15, attended; not armed] Sprint B, the loop: toml, env, idle pre-check, prompt switch, Gate 2/3 triage resources,
   attestation format, tests, dry runs (design doc "The sprints"). All clauses.
2. Sprint A, the coordinator auto-approval policy: its own design doc via design-doc-creator, scored
   against ≥2 weeks of this loop's shadow ledger before it auto-approves anything.
3. [NEXT] First armed iteration at Level 1, attended. Clauses 1–3.

---
**Document created**: 2026-10-15. Iteration 0 ratifies it with Mark before the loop is armed.

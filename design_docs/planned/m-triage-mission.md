# M-TRIAGE-MISSION — a loop that finishes what the fleet starts

**Status**: Planned (drafted attended 2026-10-15; Mark: "the loop should use the approval system
and merging PRs as appropriate … start at level 0 … messages to discord and daneel to help
facilitate it"). Charter: [triage-mission.md](../triage-mission.md).
**Lane**: harness extension (a new mission profile on the shared driver; no core change).
**Traces to**: [PROGRAM.md](../PROGRAM.md).

## Problem (measured 2026-10-15, HEAD `79ede6ee9`)

The PR, issue and approval queues on `sunholo-data/ailang` are almost entirely the fleet's own
output, and nothing drains them while V1 is paused (`~/.ailang/state/mission-control.disabled`,
since 2026-09-14).

| Queue | Count | Composition |
|---|---|---|
| Open PRs | 33 | 22 `coordinator/task-*`, 3 `fleet/*`, 4 Dependabot, 4 other agent branches |
| Open issues | 70 | 69 authored by `sunholo-voight-kampff`; 22 without a `priority:` label; 26 older than 30 days |
| Pending approval cards | 21 | Mostly the same items as the PRs (task-a0ef7af7 = #1723, task-fc7451cd = #1750, …) |
| `dev` CI | 12/12 red | Last 12 `CI` runs on `dev` failed |

A card, its PR and its originating issue are **one item**, not three queues. Approving a card is
what dispatches the next chain stage, so a stale card is a stalled chain (#1747, #1749 and #1750
are green and six days old).

## What approving and merging actually do (read in code, file:line at `79ede6ee9`)

This is the section the levels below are built against. Each hazard has a guard in the level
that first admits the action.

### Approve (`ailang coordinator approve <task> --remote gcp --yes`)

`cmd/ailang/coordinator_approvals_remote.go:184` → `coordinator.ProcessApprovalRequest`
(`internal/coordinator/approval_processor.go:67`), in this order:

1. Refuses a non-pending card (`remote.go:209-213`); loads the **task's plane** registry, not the
   laptop's (`:267-268`).
2. Writes a `human_approval` event and marks the card resolved with `ApprovedBy`.
3. **Dispatches the handoffs** (`approval_handoff.go:481`): every non-auto `TriggerOnComplete`
   edge fires a Cloud Run Job. A failed handoff still reports the approval as success ("HANDOFF
   FAILED", `:247-250`). Per-task cap `task_max_cost` (default $25 global, $15 claude;
   `agent_config.go:199-215`).
4. **Then merges the PR** (`reconcileTaskPR`, `remote.go:301` → `DecidePR`,
   `pr_reconcile.go:69`): `gh pr merge --squash --delete-branch --auto`, no `--admin`
   (`messaging/github_pr.go:226`), only when every file matches the agent's `artifact_patterns`
   and the card's file list equals the branch's. Otherwise PRRefuse: the PR stays open.

Consequences:

- **H1 Dispatch precedes merge.** The next agent can clone a `dev` that does not yet contain the
  change it is meant to build on.
- **H2 Orphans.** If the merge is refused, conflicts or never lands, the task is already
  completed and the handoff already fired; the PR stays open with no card.
- **H3 The authority gate is advisory.** `resolveApprovalAuthority`
  (`coordinator_approvals_authority.go:102`) only labels the identity; the remote path never checks
  `Granted` (`remote.go:231-240`). `agent_actionable` / `diff_available` in `approvals --json` are
  the only signals, and **nothing enforces them** — the loop must.
- **H4 NO-DIFF cards** (`diff_available=false`) still dispatch on approve.
- **H5 Audit.** `ApprovedBy` is free text; `--by` sets it. The squash commit names no approver.

### Reject

Remote reject is **final**: no retrigger (`remote.go:272-285`), chain marked failed, the PR closed
with `--delete-branch` (`pr_reconcile.go:41`), and the feedback is stored but never reaches the
agent. (**H6**.) Retry-with-feedback exists only on local CLI reject and the dashboard.

### Merging a coordinator PR by hand

The landed-card sweep (`daemon_landed_cards.go`, every 10 min, up to ~1h at scale-to-zero) treats
a merged PR as **approval with dispatch** for cards created after 2026-09-23 15:00 UTC
(`ApprovedBy="pr-merge #N (unknown)"`). A PR closed unmerged rejects its card, final. (**H7**:
there is no "just merge" — merging *is* approving.) `coordinator prs --landed --apply` resolves
landed cards **without** handoffs; `--fire-handoffs` adds them.

### Any merge to `dev`

- **H8** `ailang-core-dev` Cloud Build runs on every push: 16 images, 4 dev services, 15 executor
  jobs (`cloudbuild-dev.yaml`). A merge can redeploy the coordinator the loop is talking to.
  Prod is tag-gated, so dev merges never reach prod directly.
- **H9** The bot account bypasses branch protection (`scripts/hooks/pre-push:5-6`), so `--auto`
  is not proof checks ran. The client-side pre-push gates (stdlib freeze, ci-quick) are bypassed by
  server-side merges.
- **H10** Squash produces a commit CI never tested; `coordinator/task-*` branches have no rebase
  logic and go stale.
- **H11** Red `dev` is owned by V1 (`gate-1-observe.md:233-271`), which is paused — today nobody
  owns it.

## Iteration shape

The shared driver hard-codes a designer → planner → executor → evaluator prompt
(`tools/launchd/mission-control.sh:2288-2302`), and `m-mission-light-profile.md` lists
"inbox-triage" only as future work. Triage needs a different Gate 3, the same way fleet got a
different Gate 2.

One iteration:

1. **Gate 0/1 (unchanged)** — directives, observe. Plus: is `dev` green? If not, the digest
   leads with it and **no action above Level 0 is taken this iteration**.
2. **Gate 2 triage branch** — build the item list: join cards ↔ PRs ↔ issues (`coordinator
   approvals --remote gcp --json`, `gh pr list`, `gh issue list`); select what changed since the
   last digest plus the oldest N unverified.
3. **Gate 3 triage branch** — read-only verifier agents (the `github-issue-triage` recipe, ~20
   items each) return one verdict per item: `APPROVE` · `NEEDS-REBASE` · `NEEDS-FIX` ·
   `SUPERSEDED` · `CLOSE-FIXED` · `CLOSE-OBSOLETE` · `CLOSE-DUPLICATE` · `KEEP` ·
   `NEEDS-DECISION` (each with evidence). A **judge from a different vendor** reviews every verdict
   that would trigger an action at the current level, and every `APPROVE` regardless of level.
4. **Act** — only what the current level admits (below). At Level 0 the "action" is the
   *shadow ledger*: what it would have done, recorded so promotion is evidence-led.
5. **Gate 4/5 (unchanged)** — record, digest, retro.

## Levels (starts at 2 per Mark 2026-10-15, D-TRIAGE-6; Mark promotes; demotion is automatic)

| Level | Admits | Guards |
|---|---|---|
| **0 Observe** | Verify, label (`priority:*`, `area:*`, `needs-decision` only — never `bug`/`feature`/`from:*`, which re-import as coordinator tasks), comment with evidence, digest, shadow ledger | No approve, reject, merge, close |
| **1 Tidy** | Close issues verdicted `CLOSE-FIXED`/`CLOSE-OBSOLETE`/`CLOSE-DUPLICATE` with commit SHA + "verified against origin/dev <sha>"; `coordinator prs --landed --apply` (no `--fire-handoffs`) for pre-cutoff landed cards | Judge concurs; never an issue with a pending card or open `coordinator/task-*` PR (the coordinator imports only open issues, `internal/messaging/github.go:397`, and a close cancels nothing); never the current bookkeeping thread; ≤10 closes/iteration; every card resolved without handoffs listed in the digest. `dev` green NOT required (nothing merges) |
| **2 Approve** | `coordinator approve --remote gcp --yes --by mission-triage/<controller>` | All of G1–G9 below |
| **3 Reject / close PRs** | Reject cards and close superseded agent PRs | H6: final, work destroyed — judge concurs **and** the item has been in two consecutive digests unanswered |

**Level 2 guards** (each answers a hazard above):

- **G1** `dev` CI green at the iteration's start (H11, H10).
- **G2** Every check on the PR **head** is SUCCESS or SKIPPED — not just required ones (H9).
- **G3** `mergeable=MERGEABLE` and not BEHIND; otherwise verdict `NEEDS-REBASE` (H10, H2).
- **G4** `diff_available=true` and `agent_actionable=true` in `approvals --json` (H3, H4).
- **G5** Handoff targets listed in the digest *before* approving; at most **2 approvals per
  iteration, 4 per day** (H1 cost).
- **G6** Never approve a PR touching `std/` without the frozen goldens updated in the same PR,
  nor one touching `internal/coordinator/`, `cloudbuild*.yaml`, `tools/launchd/`,
  `.github/workflows/` or `missions/` — the loop must not redeploy or rewrite the plane it runs on
  (H8, H9). Those go to `NEEDS-DECISION`.
- **G7** After approving, bounded-wait (≤30 min) for the merge; not merged → file the orphan in the
  digest as `[urgent]` (H2).
- **G8** Watch the first `dev` CI run after each merge; red → attribute, report `[urgent]`, and
  **demote to Level 1** until Mark re-promotes.
- **G9** `--by` names the mission and controller model (H5).

**Out of scope at every level**: Dependabot majors and red Dependabot PRs (human); `fleet/*` and
other missions' PRs (their owner merges; triage reports them); prod promotion.

**Promotion evidence**: the shadow ledger records every would-be action. Mark promotes a level when
its shadow verdicts agree with what actually happened to those items (Mark's own actions, other
loops, later evidence). The level lives in the charter ledger, so promotion is a recorded ruling.

## Communication

| Channel | What | How (verified) |
|---|---|---|
| **Discord** | One short post per iteration *only when something needs Mark or changed*: counts, top 3 decisions, link to the issue comment | `ailang messages send user …` on prod. The `user` inbox maps to EventType `public-feedback`, which Discord's allow-list accepts (`internal/daemon/handlers.go:96-139`, `internal/notify/discord.go:21`). The rig daemon runs `--env prod --messages-sub messages-rig` (checked 2026-10-15). Discord truncates at 2000 chars per chunk. |
| **Daneel** | Full digest `[normal]` → morning brief; `NEEDS-DECISION` and orphans `[urgent]` → one mail within the hour (Daneel caps urgent mail at 5/day) | `ailang messages send daneel-reports --from mission-triage --title "[normal] …"` with `AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac` (`~/dev/daneel/skills/daneel-report-SKILL.md:17-27`). Report-only: Daneel never dispatches work from it. Its brief already lists pending approval cards. |
| **Bookkeeping issue** | The full digest + shadow ledger, every iteration | Shared `_mc_notify` / Gate 4, as every mission |

**Daneel's role, now and later.** Today Daneel turns the digest into your morning brief and an
urgent mail. It cannot approve or merge on ailang and should not: the bot has no write row on
the plane repos (`daneel/authority.md:44`), and approving is a merge. The useful next capability is
a **ruling relay**: you reply to Daneel's mail with a closed-set word on an item ref
(`approve 1747`, `hold 1723`), and Daneel writes it as a directive the triage loop reads at
Gate 0. Daneel routes on metadata only (axiom D8), so it would be closed-set words, never free
text. That is a Daneel capability onboarding (capabilities.md §2, starting at rung 2), filed on the
`daneel-design` inbox, not part of this sprint.

## Driver and skill changes (the sprint)

1. `missions/triage.toml` — interval 12h, `boot_offset = 2520` (next free slot), own clone
   `~/dev/sunholo-data/ailang-triage`; pin worktree like fleet.
2. `tools/launchd/mission-control.sh` — a **triage idle pre-check** beside fleet's: compute a queue
   fingerprint (open PR heads + pending card ids + issue `updatedAt`) with bounded `gh` calls and
   exit before any probe when it equals the last digested one. An unreadable fingerprint yields
   loudly, as fleet's does.
3. Same file — the PROMPT's role sentence becomes per-mission: triage spawns verifiers and a judge,
   not designer/planner/executor.
4. `.claude/skills/mission-control/resources/` — `gate-2-pick.md` triage branch and a new
   `gate-3-triage.md` (the verdict table, the level table, G1–G9, the shadow ledger format). One
   skill, never forked.
5. `tools/launchd/mission-env/mission-triage.env` — routing: cheap verifiers
   (`codex:gpt-6-luna` → `pi:ollama/glm-5.3-flash:cloud`), sonnet judge on a vendor-disjoint
   ladder, Sol controller; `MISSION_METERED_BUDGET_USD=1`.
6. `make test-launchd-drivers` arms for the pre-check and the prompt switch; healthy + degraded dry
   runs; `ailang mission doctor`.

## Non-goals

Fixing what it finds (it routes: defects → `design-doc-creator`, harness → fleet tickets, language
gaps → the V1 backlog); owning red `dev` (that is D-TRIAGE-2); Dependabot policy; prod.

## Open decisions

Recorded in the charter ledger: D-TRIAGE-1 ratify; D-TRIAGE-2 red-`dev` ownership while V1 is
paused; D-TRIAGE-3 Discord noise threshold; D-TRIAGE-4 the Daneel ruling relay.

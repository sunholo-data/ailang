# M-TRIAGE-MISSION — a loop that finishes what the fleet starts

**Status**: Planned (drafted attended 2026-10-15; Mark: "the loop should use the approval system
and merging PRs as appropriate … messages to discord and daneel to help facilitate it").
**Revised 2026-10-15 (D-TRIAGE-7)**: split into two parts after Mark asked whether the loop "is
just replacing the coordinator's job". **Part A** upgrades the coordinator with a deterministic
auto-approval policy, and only the coordinator acts. **Part B** is the loop: judgement, tidy-up
and reporting, on top. The loop never approves or merges. Charter: [triage-mission.md](../triage-mission.md).
**Lane**: Part A is a coordinator fix (AILANG-fix lane: policy in AILANG, Go as the shell).
Part B is a harness extension (a new mission profile on the shared driver).
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

This is the section Parts A and B below are built against. Each hazard is closed either in the
coordinator's code (Part A) or by a loop guard (Part B).

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
   leads with it (the loop takes no action that depends on CI; Level 1 actions merge nothing).
2. **Gate 2 triage branch** — build the item list: join cards ↔ PRs ↔ issues (`coordinator
   approvals --remote gcp --json`, `gh pr list`, `gh issue list`); select what changed since the
   last digest plus the oldest N unverified.
3. **Gate 3 triage branch** — read-only verifier agents (the `github-issue-triage` recipe, ~20
   items each) return one verdict per item: `VERIFIED` · `NEEDS-REBASE` · `NEEDS-FIX` ·
   `SUPERSEDED` · `CLOSE-FIXED` · `CLOSE-OBSOLETE` · `CLOSE-DUPLICATE` · `KEEP` ·
   `NEEDS-DECISION` (each with evidence). A **judge from a different vendor** reviews every verdict
   that would trigger an action, and every `VERIFIED` (which becomes an attestation Part A reads).
4. **Act** — Level 1 actions (below), plus writing attestations for `VERIFIED` items. Every
   verdict also goes in the *shadow ledger*, so Part A's policy can be scored against it.
5. **Gate 4/5 (unchanged)** — record, digest, retro.

## Why two parts (D-TRIAGE-7)

The approval card is the coordinator's own human gate, and the coordinator already has settings for
skipping it automatically: `skip_approval`, `auto_merge`, `auto_approve_handoffs` and per-target
`auto_approve_handoff_to` (`agent_registry.go:123-131`); `auto_merge_code`, which requires named checks,
artifact patterns and an approver identity (`agent_automerge.go:9-35`); and the change-class autonomy
router (`autonomy_router.go:22-47`). Almost all of the first draft's Level 2 guards (G1–G9) are
deterministic. A loop pressing `approve` under prose guards would be a **second approver outside
the coordinator**, implementing its policy twice, once in Go and once in a skill. It would also
leave H1 (dispatch before merge) and H3 (an advisory authority gate) unfixed where they live.

What no deterministic rule can do is judgement: does this PR do what its issue asked, is this issue
fixed on `dev`, is it a duplicate, why did this chain stall. That is the loop.

## Part A — coordinator auto-approval policy (upgrade)

A policy the coordinator evaluates for every pending card, on its existing sweep cadence. When the
policy passes, the coordinator approves through its own path. When it fails, the card stays pending
with the failing clause recorded on it. The draft guards become policy clauses:

| Clause | Rule | Hazard |
|---|---|---|
| P1 | `dev` CI green on the current head | H10, H11 |
| P2 | Every check on the PR head SUCCESS or SKIPPED, not just required ones | H9 |
| P3 | MERGEABLE and not BEHIND | H2, H10 |
| P4 | `diff_available` (no NO-DIFF approvals) | H4 |
| P5 | Path denylist: `std/` without updated goldens, `internal/coordinator/`, `cloudbuild*.yaml`, `tools/launchd/`, `.github/workflows/`, `missions/` | H8, H9 |
| P6 | Rate cap: ≤4 auto-approvals per day, per-task cost under `task_max_cost` | H1 cost |
| P7 | A current **triage attestation** on the PR head (Part B) | judgement |
| P8 | After the first auto-merge that is followed by a red `dev`, auto-approval switches off until a human re-enables it | H10 |

Code changes the policy needs (each closes a hazard in the code rather than around it):

1. **Dispatch after merge (H1, H2).** On the cloud path, approving marks the card
   `approved-awaiting-merge` and enables `--auto` merge. The landed-card sweep dispatches the
   handoffs when the merge lands, which it already does for hand-merged PRs. An approved PR that
   never merges is visible as a card stuck in that state, not an orphan.
2. **Enforce the authority gate (H3).** `coordinatorResolveRemote` checks `Granted`. **Risk:** the
   16-day CLI lockout of 09-07..09-23 came from exactly this kind of check. The human paths
   (dashboard, attended CLI with `AILANG_APPROVAL_CONTROLLER=1`) get explicit tests in both
   directions before the check is switched on.
3. **Audit (H5).** Policy approvals record `ApprovedBy=policy:<version>+attestation:<judge>`. The
   squash-merge body names the approver.
4. **Where the policy lives.** Recommended: an AILANG module the coordinator loads (memory: about 65ms
   load and 13µs per call), so the rules are typed and testable outside Go. The alternative is
   extending `auto_merge_code` in Go. This is D-TRIAGE-8.

Out of scope for the policy: Dependabot (its own workflow), other missions' PRs, prod, and
reject. Remote reject stays human because it is final (H6).

## Part B — the triage loop (enhancement on top)

The loop runs at **Level 1** (D-TRIAGE-7). It does not approve, reject or merge.

| Admits | Guards |
|---|---|
| Verify every card, PR and bot-filed issue; label (`priority:*`, `area:*`, `needs-decision` only, never `bug`/`feature`/`from:*`, which re-import as coordinator tasks); evidence comments; digest; shadow ledger | none beyond read-only |
| **Attest** `VERIFIED` items for Part A's P7 | Judge from a different vendor concurs; the attestation is bound to the PR head SHA, so a new push invalidates it |
| Close issues verdicted `CLOSE-FIXED` / `CLOSE-OBSOLETE` / `CLOSE-DUPLICATE` with the commit SHA and "verified against origin/dev <sha>" | Judge concurs; never an issue with a pending card or open `coordinator/task-*` PR (the coordinator imports only open issues, `internal/messaging/github.go:397`, and a close cancels nothing); never the current bookkeeping thread; ≤10 closes per iteration |
| `coordinator prs --landed --apply` (no `--fire-handoffs`) for pre-cutoff landed cards | Every card resolved without handoffs is listed in the digest with its targets, since this ends its chain |

**The attestation carrier** is D-TRIAGE-9. Recommended: a PR comment in a fixed machine-readable
form (verdict, head SHA, judge model, evidence link), because comments are already readable by
the coordinator's `gh` client and carry an author. A label carries no SHA and no author, and
labels on issues re-import as tasks.

**Until Part A ships**, attestations still have value: they are listed in the digest as "ready for
you to approve". You approve from the Discord ping (whose payload carries the exact command) or
the dashboard, and the shadow ledger records whether the policy, had it existed, would have agreed.

**Out of scope**: Dependabot majors and red Dependabot PRs (human); `fleet/*` and other missions'
PRs (their owner merges; triage reports them); prod promotion.

## Communication

| Channel | What | How (verified) |
|---|---|---|
| **Discord** | One short post per iteration *only when something needs Mark or changed*: counts, top 3 decisions, link to the issue comment | `ailang messages send user …` on prod. The `user` inbox maps to EventType `public-feedback`, which Discord's allow-list accepts (`internal/daemon/handlers.go:96-139`, `internal/notify/discord.go:21`). The rig daemon runs `--env prod --messages-sub messages-rig` (checked 2026-10-15). Discord truncates at 2000 chars per chunk. |
| **Daneel** | Full digest `[normal]` → morning brief; `NEEDS-DECISION` and orphans `[urgent]` → one mail within the hour (Daneel caps urgent mail at 5/day) | `ailang messages send daneel-reports --from mission-triage --title "[normal] …"` with `AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac` (`~/dev/daneel/skills/daneel-report-SKILL.md:17-27`). Report-only: Daneel never dispatches work from it. Its brief already lists pending approval cards. |
| **Bookkeeping issue** | The full digest + shadow ledger, every iteration | Shared `_mc_notify` / Gate 4, as every mission |

**Daneel's role, now and later.** Today Daneel turns the digest into your morning brief and an
urgent mail. It cannot approve or merge on ailang and should not: the bot has no write row on
the plane repos (`daneel/authority.md:44`), and approving is a merge. With the split, approving belongs to the coordinator's policy, so a relayed `approve 1747` would become a human approval through the normal path. The useful next capability is
a **ruling relay**: you reply to Daneel's mail with a closed-set word on an item ref
(`approve 1747`, `hold 1723`), and Daneel writes it as a directive the triage loop reads at
Gate 0. Daneel routes on metadata only (axiom D8), so it would be closed-set words, never free
text. That is a Daneel capability onboarding (capabilities.md §2, starting at rung 2), filed on the
`daneel-design` inbox, not part of this sprint.

## The sprints

**Sprint B (the loop) goes first**: it is read-mostly and produces the attestations that Part A's
policy consumes.

1. `missions/triage.toml`: interval 12h, `boot_offset = 2520` (next free slot), own clone
   `~/dev/sunholo-data/ailang-triage`, pin worktree like fleet.
2. `tools/launchd/mission-control.sh`: a **triage idle pre-check** beside fleet's. It computes a queue
   fingerprint (open PR heads, pending card ids, issue `updatedAt`) with bounded `gh` calls and
   exits before any probe when the fingerprint equals the last digested one. An unreadable
   fingerprint yields loudly, as fleet's does.
3. Same file: the PROMPT's role sentence becomes per-mission. Triage spawns verifiers and a judge,
   not designer/planner/executor.
4. `.claude/skills/mission-control/resources/`: a triage branch in `gate-2-pick.md` and a new
   `gate-3-triage.md` (verdict table, Level 1 guards, attestation format, shadow ledger). One
   skill, never forked.
5. `tools/launchd/mission-env/mission-triage.env`: cheap verifiers (`codex:gpt-6-luna`, then
   `pi:ollama/glm-5.3-flash:cloud`), a sonnet judge on a vendor-disjoint ladder, a Sol controller,
   `MISSION_METERED_BUDGET_USD=1`.
6. Tests in `make test-launchd-drivers` for the pre-check and the prompt switch; healthy and degraded
   dry runs; `ailang mission doctor`.

**Sprint A (the coordinator policy)** follows, through design-doc-creator and then sprint-planner.
It needs its own design doc for code changes A1–A4 above. The acceptance bar: the policy is
scored against at least two weeks of Sprint B's shadow ledger before it auto-approves anything.

## Non-goals

Fixing what it finds (it routes: defects → `design-doc-creator`, harness → fleet tickets, language
gaps → the V1 backlog); owning red `dev` (that is D-TRIAGE-2); Dependabot policy; prod.

## Open decisions

Recorded in the charter ledger: D-TRIAGE-1 ratify; D-TRIAGE-2 red-`dev` ownership while V1 is
paused; D-TRIAGE-3 Discord noise threshold; D-TRIAGE-4 the Daneel ruling relay; D-TRIAGE-8 where
the policy lives; D-TRIAGE-9 the attestation carrier.

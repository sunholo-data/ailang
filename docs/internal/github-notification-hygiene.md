# Proposal: Muting agent traffic on GitHub without losing the signal

**Status:** draft for approval · **Date:** 2026-09-08 · **Author:** session (voight-kampff loop)

## Problem

The ailang agent fleet (`sunholo-voight-kampff` and friends) generates the overwhelming majority of
GitHub events — commits, PR comments, task status updates. Measured on this repo: **491 of the last
500 commits (~98%)** come from one agent identity, vs 5 from human accounts. The human maintainer's
GitHub inbox (and any email notifications) drowns in agent noise, and real signal — review requests,
mentions, CI failures, Dependabot alerts — is easy to miss.

## Constraints

- **Mute, not delete.** The agent audit trail on GitHub must stay intact and findable; we only want
  it out of the default attention stream.
- **Agents keep working.** No solution may block agent identities from committing/commenting/PRing.
- **Escalation signal survives.** Anything where an agent explicitly pings a human (mention, review
  request) or a system alerts (security, human-triggered CI) must still land in the foreground.

## What GitHub actually supports (verified 2026-09-08)

| Feature | Per-author mute? | Notes |
|---|---|---|
| Per-thread **Unsubscribe/Unwatch** | no (per-thread) | Native, instant, but manual; impractical at fleet volume |
| **Watch settings** (per repo: All / Participating & @mentions / Custom) | no | Custom can drop workflows/releases/Dependabot — cuts categories, not authors |
| **Inbox custom filters** | filter *to* authors only | Docs: supports `author:`, `repo:`, `reason:`, `org:` — **explicitly no `NOT` / `-QUALIFIER` exclusion** |
| **Block user** | yes | ❌ wrong tool — abuse control, revokes the agent's access and muddies the audit trail |
| Native **bot-comment collapse** in issues/PRs | n/a | Applies to GitHub App identities (`[bot]` suffix) — plain user accounts like `sunholo-voight-kampff` do NOT auto-collapse |

**Conclusion:** GitHub has no native "mute this author" switch. We compose one from
(client-side filtering) + (notification-API automation) + (source-side volume reduction).

## The mute list

Initial: `sunholo-voight-kampff` (fleet account; ~all agent traffic), plus `sonarqubecloud[bot]`
for PR comment notifications (see "SonarCloud PR comments" below; note the actual login is
`sonarqubecloud[bot]` — SonarQube Cloud branding — measured as the single biggest PR commenter:
28 of the last 50 issue comments on `sunholo-data/ailang`). Add `app/dependabot` only if
Dependabot noise bothers you — its PRs are usually actionable. The list lives in one place per
mechanism below; keep them in sync.

## SonarCloud PR comments ("sonarbot")

SonarQube Cloud's GitHub App (`sonarqubecloud[bot]`) decorates PRs with: a **quality-gate status check**
(Checks tab), an **edited-in-place summary comment** (Conversation tab, one per PR), and optional
**inline annotations** (Files changed). Only the last two generate notifications.

**Best fix is at the source** — turn off the comment, keep the signal:

- Set `sonar.pullrequest.github.summary_comment=false` (SonarCloud project admin → General
  Settings, or `sonar-project.properties`). No comment → no notification, and the PR conversation
  stays clean — the only option that fixes the PR *page*, not just the inbox.
- Inline annotations can be disabled separately in the GitHub binding config if those are the
  annoyance.
- **Signal kept:** the quality-gate status check remains and still fails the PR (incl. branch
  protection); the summary in the Checks tab is unchanged. Nothing actionable lives in the comment
  itself.

If you'd rather keep the comments and only mute the **notifications**, restrict by PR:

- **Layer 3 script:** `actor == sonarqubecloud[bot] && subject.type == "PullRequest"` → mark read.
  Exactly PR-only; issue notifications (if any) untouched.
- **Layer 1 inbox filter:** `author:app/sonarqubecloud` — works, but custom filters cannot
  distinguish `is:pr` from `is:issue`, so PR-only narrowing is not expressible natively.
- **Email:** `X-GitHub-Sender: sonarqubecloud[bot]` (+ a per-type header if present — verify once on
  a live email) → skip inbox, label `sonar`.
- **In-page:** as an App identity, GitHub auto-collapses `[bot]` comments in the conversation;
  if it isn't collapsing on a live PR, fall back to the source-side setting above.

**Exact toggles (official docs, verified 2026-09-08) — project `sunholo-data_ailang` on SonarCloud:**

- Summary comment (the Conversation-tab one): **Administration → General Settings → Pull Requests →
  Integration with GitHub → unselect "Enable summary comment"**
  (property `sonar.pullrequest.github.summary_comment=false`).
- Inline annotations: **Administration → General Settings → Pull Requests → Issue Annotations →
  unselect "Enable Issue Annotations"**.
- **Status 2026-09-08:** toggles applied by Mark in the SonarCloud UI. SonarCloud's settings API
  is not publicly readable, so verification is empirical: the next agent PR should get the
  `SonarCloud Code Analysis` check but **no** Conversation-tab comment. Comments already posted on
  existing PRs remain (deleting them is possible with repo-admin if wanted).
- The quality gate check (`SonarCloud Code Analysis`) is a separate GitHub status check — unaffected;
  keep it as a required check in branch rulesets if it gates merges today.

## Layered proposal

### Layer 0 — Watch settings (5 min, native, zero risk)

Repo → Watch → **Custom**: uncheck everything except Issues, Pull Requests (drop Releases, Workflows
if `ci_activity` emails are noisy, Discussions). Keeps human+agent thread activity, drops bulk
category noise regardless of author.

### Layer 1 — GitHub web inbox: "agents" custom filter (5 min, native)

github.com/notifications → build a custom filter:

```
author:sunholo-voight-kampff
```

(or `author:sunholo-voight-kampff author:app/dependabot` — same-qualifier terms OR together).

Use it as the "everything agents did" batch-triage view: mark-as-read in bulk. Caveats: custom
filters **cannot exclude** agents, and the unread **badge still counts** agent noise. This layer
gives you the audit view and one-click triage, not silence.

### Layer 2 — Email routing/rules (10–20 min, native + client-side)

Two options, pick one:

1. **Native custom routing (cleanest):** Settings → Notifications → **Custom routing** → send
   `arniwesth/ailang` notifications to a dedicated mailbox (e.g. `ailang-agents@…`). Main inbox
   never sees repo traffic; the agent mailbox *is* the full audit trail, browsable anytime. Zero
   filter maintenance. (You may still want a rule inside that mailbox to foreground
   `mention`/`review_requested` reasons.)
2. **Author-based rules in your current client:** every GitHub notification email carries consistent
   headers. Officially documented: `From` = `notifications@github.com`, `List-Id`
   (`owner/repo`), and the Cc second address encodes the **notification reason**
   (`author@noreply.github.com`, `mention@noreply.github.com`, …). Additionally the
   `X-GitHub-Sender` header carries the triggering actor's login in real GitHub emails —
   **not in today's docs table, so verify once** on a live agent email before relying on it.
   - Outlook / Fastmail / Apple Mail: rule on header `X-GitHub-Sender` = `sunholo-voight-kampff`
     → skip inbox + label `github-agents`. Never delete.
   - Gmail: user filters cannot match arbitrary headers. Fallbacks: match the From **display
     name** (verify what GitHub puts there for agent comments), or use a Workspace admin
     content-compliance rule to add a subject prefix on `X-GitHub-Sender` and filter on that.
   - Reason-based routing (keeps signal even from muted authors): route `mention@noreply.github.com`
     and `review_requested@noreply.github.com` Cc addresses to the foreground.

### Layer 3 — Notification-API auto-triage (1–2 h, one script)

Makes the **web badge clean** without GitHub-native support. A small script (`tools/` candidate)
runs on a schedule (cron on this machine, or a scheduled repo Action with a fine-grained PAT):

```
GET /notifications            (paginated)
for each notification:
    actor = author of subject.latest_comment_url (or subject url for PRs/issues)
    if actor ∈ MUTE_LIST and reason ∉ {mention, review_requested, security_alert}:
        PUT /notifications/threads/{id}     # mark as read — archived, not deleted
```

- Web badge and inbox go quiet; everything remains on GitHub and in email as audit trail.
- The reason allowlist is the "keep the signal" clause: agent @mentions and review requests
  still surface even when authored by a muted account.
- Mark-as-read is reversible in spirit: GitHub keeps the read/unread state only, but the content
  is fully preserved; our own coordinator messages already bank the same facts.

### Layer 4 — Structural, at the source (half day, recommended)

We control the generator. Three changes to the fleet's GitHub behavior:

1. **One identity for all agent writes.** A single GitHub App (or one dedicated account) for every
   mission's comments/PRs. One author to mute in every mechanism above — and if it's a **GitHub
   App**, its login ends in `[bot]`, so **GitHub natively collapses its comments** in issues/PRs,
   dimmed by default. This is the only native in-page muting GitHub offers.
2. **Sticky comments.** Mission executors edit one canonical comment per task/PR instead of posting
   a new one per status change (GitHub API: edit the comment). Cuts notification *events* ~10x
   while keeping the full log on GitHub.
3. **Mention discipline.** Agents only @-mention humans for the explicit approval-needed moments;
   everything else is plain comments (which Layer 3 sweeps).

## Recommendation & rollout

1. **Today (zero code):** Layer 0 + Layer 2 option 1 (custom routing to a side mailbox). If you
   want to stay in your current inbox instead, Layer 2 option 2 with one verified header.
2. **This week:** Layer 3 script (I can write `tools/gh-notification-triage.sh` + launchd plist
   mirroring `tools/launchd/mission-control.sh`), with the MUTE_LIST + reason allowlist as
   shell variables at the top.
3. **When convenient:** Layer 4 identity consolidation + sticky-comment policy — this needs a small
   change to executor prompts/tooling and is the only part that touches mission code, so it should
   go through the normal design-doc → sprint path if we take it on.

## Open questions

- Which surface hurts most — GitHub web badge, or email? (Determines Layer 2 vs Layer 3 priority.)
- **Repo mismatch (answered):** this checkout's remote (`arniwesth/ailang`) is a mirror with only 2
  PRs and no bot comments. The working repo is **`sunholo-data/ailang`** — mute list, routing, and
  the Layer 3 script target that repo. SonarCloud org `sunholo-data`, project key
  `sunholo-data_ailang`; the SonarCloud toggles above need a SonarCloud-side token with Administer
  permission (the CI `SONAR_TOKEN` secret is GitHub-write-only and analysis-scoped by convention).
- Should Dependabot be muted, or does its automerge workflow already keep it quiet enough?
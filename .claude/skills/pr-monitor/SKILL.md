---
name: pr-monitor
description: Robustly observe, audit, and respond to GitHub PRs we author (or have changes-requested on). Use when waiting on review feedback, when the user says "any new comments?", when reviewing CR/maintainer asks, or when replying to inline review threads. Codifies lessons from missing the `/pulls/N/reviews` endpoint and from `since=` cursor resets.
---

# pr-monitor

A robust observation + response workflow for GitHub PRs that solves the failure modes we hit on `aallan/vera-bench` PR #70 and PR #73:

1. **Missed a formal `CHANGES_REQUESTED` review** because the comments endpoints don't surface it — formal review submissions live in `/pulls/N/reviews`, NOT `/pulls/N/comments` or `/issues/N/comments`.
2. **Lost events on monitor restart** because the `since=` cursor was reinitialised to "now", silently dropping anything submitted in the gap.
3. **Inline replies without `@coderabbitai` + commit SHA** don't trigger CodeRabbit's auto-resolution.

## When to use

- "Any new comments on my PR?"
- "What's the maintainer waiting on?"
- "Watch the PR and tell me when something changes"
- "Reply to that CR finding"
- "Are all review threads resolved?"
- After pushing a fix, to confirm CR / maintainer picked it up

## The three endpoint problem (read this first)

GitHub puts PR activity in **three** endpoints. **Each event lives in only one.** Audit must check all three:

| Endpoint | What it contains |
|---|---|
| `repos/{o}/{r}/issues/{n}/comments` | Top-level PR thread comments (the conversation tab's main column) |
| `repos/{o}/{r}/pulls/{n}/comments` | Inline review comments (anchored to file + line) |
| `repos/{o}/{r}/pulls/{n}/reviews` | **Formal review submissions** with `state` ∈ {`APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `DISMISSED`}. The `body` of a `CHANGES_REQUESTED` review is the canonical "here's what needs to change" — it does NOT appear in either comments endpoint. |

**Plus one more**:

| Endpoint | What it does |
|---|---|
| `/notifications?all=true` | The user's GitHub inbox. Canonical "this account got pinged" feed. **This is the right primitive for monitoring.** Each notification carries a `reason` (mention, review_requested, etc.) and a `subject.url` to drill into. |

## Workflows

### 1. Audit — "what's the current state of PR N?"

```bash
.claude/skills/pr-monitor/scripts/audit_pr.sh <owner>/<repo> <pr-number>
```

Outputs: latest activity on all 3 endpoints, unresolved thread count, CI check status, merge state, review decision. Use this BEFORE asking the user "anything new?" or before posting a closeout reply.

### 2. Monitor — "watch the PR(s) and notify me on changes"

```bash
.claude/skills/pr-monitor/scripts/monitor_pr.sh <owner>/<repo> <pr-number> [<pr-number>...]
```

Long-running. Polls `/notifications?all=true` every 60s. Emits one line per new event, drilling into the `subject.latest_comment_url` to surface the actual content. Exits when **all** monitored PRs reach a terminal state (`MERGED` or `CLOSED`). Designed to be wrapped with the `Monitor` tool.

### 3. Reply to a CR thread (triggers auto-resolve)

```bash
.claude/skills/pr-monitor/scripts/reply_to_thread.sh <owner>/<repo> <pr-number> <thread-comment-id> <commit-sha> "<reply body>"
```

Posts an inline reply with the structure CodeRabbit's auto-resolution detector looks for:
- Starts with `@coderabbitai`
- References the addressing commit as a markdown link `[SHA](url)`

This is **the only way** to get CR threads to auto-close after a fix. Commit messages mentioning "Address CodeRabbit review" do NOT trigger CR's resolver.

### 4. Budget check — "should I keep iterating on this PR?"

```bash
.claude/skills/pr-monitor/scripts/pr_budget.sh <owner>/<repo> <pr-number>
```

Reports total iteration cost (commits, review waves, our replies), severity histogram of CR findings (🔴 / 🟠 / 🟡 / 🔵), and a verdict heuristic flagging when wave count ≥ 3 indicates diminishing returns. **Run before responding to a new CR wave on a PR that's already had 2+ cycles** — the cost-benefit may have flipped.

## Safeguards (read before continuing past wave 2)

CodeRabbit is a paid third-party service that improves using the changes we ship. Each iteration cycle costs us tokens (~5–15K per finding) for value that partly accrues to **their** platform. Without explicit caps, three things go wrong:

1. **Token waste on polish** — Trivial/Nitpick findings consume real budget for marginal real-world value.
2. **Maintainer fatigue** — PRs with 6+ CR-iteration commits smell process-heavy.
3. **Free training data** — Curated "this is the kind of fix that closes the loop" diffs accrue to CR's moat.

**Default severity policy:**

| Wave | Address | Defer |
|---|---|---|
| 1 | Critical, Major, Minor (and Trivial if cheap) | — |
| 2 | Critical, Major | Minor, Trivial → summarise to user |
| 3+ | Critical only | Everything else → escalate to user, post deferral comment |

**Maintainer asks always take priority** over CR findings, regardless of wave count.

See [safeguards.md](resources/safeguards.md) for the full decision tree, escalation templates, and patterns for the cases where CR is empirically wrong (it sometimes is — Python `repr()` style vs project actual output, etc.).

## Failure modes — read before changing the scripts

- **Don't initialise `since=$(date -u ...)` only at script startup.** On restart, you lose anything submitted between previous run's last-poll and now. Either:
  - Persist the last seen timestamp to disk, OR
  - Use per-thread `updated_at` from the GraphQL `reviewThreads` query, OR
  - Use `/notifications` (each notification has its own `updated_at` so there's no global cursor to reset).
- **Don't filter `/notifications` with `participating=true`.** Comments on PRs we author don't always show as "participating" — `all=true&participating=false` is the safest.
- **Don't use `gh pr view` for formal reviews.** It returns `latestReviews` truncated to the most recent per-author per-state, dropping older formal reviews. Use `gh api repos/.../pulls/N/reviews` for the full list.
- **Don't assert on output substrings in tests for CLI flag parsing.** Click signals usage errors with `exit_code == 2` — that's the semantic boundary; substring checks false-fail on unrelated runtime output.

## Resources

- [endpoints.md](resources/endpoints.md) — which endpoint to query for which event type, with example GraphQL queries
- [reply_patterns.md](resources/reply_patterns.md) — the CR auto-resolution pattern + examples of structurally-stable assertions

## Lessons captured here (chronological)

These came from the `aallan/vera-bench` work on 2026-05-22 to 2026-05-23:

1. **Missing the consolidated review** — focused on the CI-failure follow-up comment (visible in `/issues/N/comments`) and missed `@aallan`'s earlier review with 4 broader asks (also in `/issues/N/comments`, but I'd read it truncated at 500 chars and missed the items). Lesson: always pull the **full** body of the most recent reviewer comment, never trust a 500-char excerpt for action.
2. **Missing the formal `CHANGES_REQUESTED` review** — same review-decision was also submitted formally in `/pulls/N/reviews`, but my monitor only polled the comments endpoints. Lesson: monitor must cover the reviews endpoint.
3. **`since=` cursor reset on monitor restart** — when I tore down + re-armed the monitor, the new `since` started "now", losing events between the previous shutdown and the restart. Lesson: anchor `since` to a persisted timestamp or use a system that has per-event timestamps (notifications).
4. **`@coderabbitai` + SHA pattern for resolution** — CR's auto-resolution only fires on inline replies that mention `@coderabbitai` + the addressing commit SHA. Commit-messaging "Address CodeRabbit review" doesn't reach the resolver.
5. **Behavior-based test assertions** — patching `concurrent.futures.ThreadPoolExecutor` to assert "must not be used" works but is fragile to refactors that hoist the import. Asserting `threading.current_thread() is threading.main_thread()` for every call is robust because it tests behavior, not the import path.
6. **Worker exceptions can silently shrink JSONL denominators** — bare `except: log; continue` patterns lose events. Synthesise a visible result row instead, so downstream `report` shows an honest denominator.

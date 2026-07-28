# PR endpoint reference

Cheat sheet for which GitHub endpoint surfaces which event. Each row is a thing that can land on a PR and where you find it.

## Quick lookup

| Event type | Endpoint | gh CLI shortcut |
|---|---|---|
| Top-level conversation comment | `GET /repos/{o}/{r}/issues/{n}/comments` | `gh api repos/.../issues/N/comments` |
| Inline review comment (file + line) | `GET /repos/{o}/{r}/pulls/{n}/comments` | `gh api repos/.../pulls/N/comments` |
| **Formal review submission** (CHANGES_REQUESTED / APPROVED) | `GET /repos/{o}/{r}/pulls/{n}/reviews` | `gh api repos/.../pulls/N/reviews` |
| Review thread (resolved flag) | GraphQL `pullRequest.reviewThreads` | see snippet below |
| Timeline events (label, mention, cross-ref) | `GET /repos/{o}/{r}/issues/{n}/timeline` | `gh api repos/.../issues/N/timeline` |
| All pings the *user account* received | `GET /notifications?all=true` | `gh api /notifications` |
| CI check status | `GET /repos/{o}/{r}/commits/{sha}/check-runs` | `gh pr checks N` |
| PR-wide state (merged/closed/mergeable/reviewDecision) | GraphQL `pullRequest` | see snippet below |

## The critical missing-piece: `/pulls/N/reviews`

This is where formal review SUBMISSIONS live — the same review that flips `reviewDecision` to `CHANGES_REQUESTED` or `APPROVED`. The submission has a `body` (the maintainer's summary writeup) that does NOT appear in either of the comments endpoints.

**You MUST poll `/pulls/N/reviews` separately, or use `/notifications` which covers it transparently.**

Empirical example: on `aallan/vera-bench` PR #73, the maintainer's four-agent review summary (~3KB body listing I1–I5 + S1–S5 + strengths) appeared ONLY at `pulls/73/reviews?per_page=100`, with `state=CHANGES_REQUESTED`. The same review id had `body=""` rows showing as `COMMENTED` in adjacent inline-comment threads — those are inline comments anchored to the review submission, NOT the review body itself.

## Useful GraphQL snippets

### List unresolved review threads with their first-comment databaseId

```graphql
{
  repository(owner: "OWNER", name: "REPO") {
    pullRequest(number: N) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          path
          line
          comments(first: 1) { nodes { databaseId author { login } } }
        }
      }
    }
  }
}
```

`comments.nodes[0].databaseId` is what `reply_to_thread.sh` takes as the `in_reply_to` parameter when posting a reply to that thread.

### High-level PR state (mergeable + reviewDecision + thread counts)

```graphql
{
  repository(owner: "OWNER", name: "REPO") {
    pullRequest(number: N) {
      state merged mergeable reviewDecision headRefOid
      reviewThreads(first: 100) { nodes { isResolved } }
    }
  }
}
```

### Latest review per-author (used by `gh pr view`)

```graphql
{
  repository(owner: "OWNER", name: "REPO") {
    pullRequest(number: N) {
      latestReviews(first: 10) {
        nodes { author { login } state submittedAt }
      }
    }
  }
}
```

⚠️ `latestReviews` returns only the most recent per-author per-state. Use the REST endpoint `pulls/N/reviews` for the full chronological history.

## Notification reason types

What `reason` field on a notification means:

| reason | What triggered it |
|---|---|
| `mention` | Someone @username'd this account in a comment / review |
| `review_requested` | Someone added this account as a reviewer |
| `comment` | A comment was posted on a thread this account is subscribed to |
| `subscribed` | An event on something this account follows (issue assigned, PR labeled, etc.) |
| `state_change` | The issue/PR was closed, reopened, or merged |
| `author` | This account opened the issue/PR and someone replied |
| `team_mention` | A team this account belongs to was @-mentioned |
| `ci_activity` | A CI check the account is watching changed state |

For PR observation: `mention`, `review_requested`, `comment`, `state_change`, `author` are the load-bearing ones.

## Common pitfalls

### 1. Subset rate limits

`/notifications` has its own bucket, separate from per-repo endpoints. Polling notifications every 60s is fine; polling per-PR-per-endpoint every 30s can hit the secondary rate limit on busy days.

### 2. `since=` cursor pitfall

```bash
# WRONG: monitor loses events between shutdown and restart
since=$(date -u +%Y-%m-%dT%H:%M:%SZ)
while true; do
  gh api "...?since=$since" --jq '...'
  since=$(date -u +%Y-%m-%dT%H:%M:%SZ)  # global cursor — reset on restart!
  sleep 60
done
```

```bash
# BETTER: use /notifications where each event has its own updated_at,
# so monitor restarts don't lose events:
gh api '/notifications?all=true&per_page=50' \
  --jq '.[] | select(.updated_at > "<last-checkpoint>")'
```

```bash
# BEST: persist last-seen timestamp to disk between runs
CHECKPOINT_FILE=~/.cache/pr-monitor-checkpoint
since=$(cat "$CHECKPOINT_FILE" 2>/dev/null || date -u +%Y-%m-%dT00:00:00Z)
# ...poll...
date -u +%Y-%m-%dT%H:%M:%SZ > "$CHECKPOINT_FILE"
```

### 3. `participating=true` filter drops own-PR notifications

```bash
# WRONG: comments on PRs you authored aren't always "participating"
gh api '/notifications?participating=true&all=true'

# RIGHT: use all=true&participating=false
gh api '/notifications?all=true&participating=false'
```

### 4. `gh pr view --json comments` doesn't include inline review comments

`gh pr view --json comments` returns ONLY top-level issue comments. Inline review comments need `--json reviews` (returns review submissions with inline-comment children) OR direct API call to `pulls/N/comments`.

### 5. `latestReviews` truncation

`gh pr view --json latestReviews` and the equivalent GraphQL field return only the most recent submission per (author, state) pair. To audit the full review history use `gh api repos/{owner}/{repo}/pulls/N/reviews`.

### 6. **MUST `--paginate` the reviews endpoint** (the bug that bit us)

`gh api repos/.../pulls/N/reviews` without `--paginate` returns **only the first 30 reviews**. On a PR with many inline-comment-induced review rows (each `gh api -X POST .../pulls/N/comments` creates a new COMMENTED review entry under your login), the substantive 14KB CHANGES_REQUESTED body can land on **page 2** and be invisible.

Empirical example: `aallan/vera-bench` PR #70 had ~30 `state=COMMENTED` rows from inline-reply postings on page 1, with the substantive @aallan 14,953-char deep-read review on page 2. Querying without `--paginate` made me think I'd read the latest formal review when I'd actually only seen the older one.

**Always:**

```bash
gh api repos/{o}/{r}/pulls/{n}/reviews --paginate --jq '...'
```

If you suspect a substantive review you haven't read, sort by body length:

```bash
gh api repos/{o}/{r}/pulls/{n}/reviews --paginate \
  --jq '.[] | "\(.submitted_at) \(.user.login) state=\(.state) body_len=\(.body | length)"' \
  | sort -t= -k2 -nr | head -10
```

Any row with `body_len > 500` is likely a substantive review you should read in full.

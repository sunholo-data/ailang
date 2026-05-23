#!/usr/bin/env bash
# audit_pr.sh — comprehensive single-shot audit of a PR's state.
#
# Usage: audit_pr.sh <owner>/<repo> <pr-number>
#
# Queries all four endpoints that PR activity lives in (issues/comments,
# pulls/comments, pulls/reviews, /notifications) plus the GraphQL
# reviewThreads + check status, and prints a structured summary.
#
# Use this BEFORE asking the user "anything new?" — it's the right
# answer to that question because it covers the endpoint-fragmentation
# problem.

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <owner>/<repo> <pr-number>" >&2
  exit 1
fi

REPO="$1"
PR="$2"
GH="${GH_BIN:-gh}"
SINCE="${SINCE:-2026-01-01T00:00:00Z}"  # default to "all time"

echo "═══ PR #${PR} on ${REPO} — audit at $(date -u +%Y-%m-%dT%H:%M:%SZ) ═══"
echo ""

# 1) High-level state
echo "─── state ───"
$GH api graphql -f query="{
  repository(owner: \"${REPO%%/*}\", name: \"${REPO##*/}\") {
    pullRequest(number: ${PR}) {
      title state merged mergeable reviewDecision headRefOid
      reviewThreads(first: 100) { nodes { isResolved } }
    }
  }
}" --jq '
  .data.repository.pullRequest as $p
  | "title: \($p.title)\nstate: \($p.state)\nmerged: \($p.merged)\nmergeable: \($p.mergeable)\nreviewDecision: \($p.reviewDecision)\nhead: \($p.headRefOid[0:7])\nthreads: \($p.reviewThreads.nodes | length) total, \($p.reviewThreads.nodes | map(select(.isResolved == false)) | length) unresolved"
'
echo ""

# 2) Most recent 5 events on each endpoint
echo "─── top-level comments (last 5) ───"
$GH api "repos/${REPO}/issues/${PR}/comments?per_page=100" --paginate \
  --jq '. | sort_by(.created_at) | .[-5:] | .[] | "  [\(.created_at)] \(.user.login): \(.body[:160] | gsub("\n";" "))"' \
  2>/dev/null || echo "  (none)"
echo ""

echo "─── inline review comments (last 5) ───"
$GH api "repos/${REPO}/pulls/${PR}/comments?per_page=100" --paginate \
  --jq '. | sort_by(.created_at) | .[-5:] | .[] | "  [\(.created_at)] \(.user.login) @ \(.path):\(.line // "_"): \(.body[:140] | gsub("\n";" "))"' \
  2>/dev/null || echo "  (none)"
echo ""

echo "─── formal reviews (last 5, ALL states) ───"
echo "  NOTE: CHANGES_REQUESTED / APPROVED bodies live HERE, not in comments endpoints."
$GH api "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
  --jq '. | sort_by(.submitted_at) | .[-5:] | .[] | "  [\(.submitted_at)] \(.user.login) state=\(.state): \(.body[:120] | gsub("\n";" "))"' \
  2>/dev/null || echo "  (none)"
echo ""

# 3) Unresolved review threads — what the user needs to act on
echo "─── unresolved review threads ───"
$GH api graphql -f query="{
  repository(owner: \"${REPO%%/*}\", name: \"${REPO##*/}\") {
    pullRequest(number: ${PR}) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          path
          line
          comments(first: 1) { nodes { databaseId author { login } createdAt body } }
        }
      }
    }
  }
}" --jq '
  .data.repository.pullRequest.reviewThreads.nodes
  | map(select(.isResolved == false))
  | if length == 0 then "  (all resolved)" else
      .[] | "  \(.path):\(.line // "_")  comment_id=\(.comments.nodes[0].databaseId)  by=\(.comments.nodes[0].author.login)\n    \(.comments.nodes[0].body[:200] | gsub("\n";" "))"
    end
'
echo ""

# 4) CI checks
echo "─── CI checks ───"
$GH pr checks "${PR}" --repo "${REPO}" 2>/dev/null | awk '{printf "  %s\n", $0}' || echo "  (none)"
echo ""

# 5) Notifications (what the user account has been pinged about)
echo "─── notifications for this PR (latest 5) ───"
$GH api '/notifications?all=true&participating=false&per_page=50' \
  --jq ".[] | select(.repository.full_name == \"${REPO}\" and (.subject.url | contains(\"/pulls/${PR}\"))) | \"  [\(.updated_at)] reason=\(.reason) unread=\(.unread)\"" \
  2>/dev/null | head -5 || echo "  (none)"

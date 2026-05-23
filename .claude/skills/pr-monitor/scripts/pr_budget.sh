#!/usr/bin/env bash
# pr_budget.sh — report iteration cost on a PR.
#
# Usage: pr_budget.sh <owner>/<repo> <pr-number>
#
# Surfaces the metrics that signal whether further review-iteration on
# this PR has diminishing returns. Use BEFORE responding to a new CR
# wave on a PR that's already had 2+ cycles — the cost-benefit may
# have flipped against us.
#
# Why this exists: CodeRabbit is a paid service that improves its
# output using the changes we ship. Each iteration cycle costs us
# tokens (read finding, plan fix, apply, test, reply) for value that
# accrues to a third party. After ~2 substantive cycles, additional
# CR findings tend toward polish — we should defer those to the user
# rather than auto-applying.
#
# Severity markers in CR comment bodies (used for the histogram):
#   🔴 Critical / _🔴 Critical_      — real bugs, always address
#   🟠 Major / _🟠 Major_            — real improvements, usually address
#   🟡 Minor / _🟡 Minor_            — polish, defer after wave 2
#   🔵 Trivial / _🔵 Trivial_        — nitpicks, defer always
#   🧹 Nitpick                       — same as Trivial

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <owner>/<repo> <pr-number>" >&2
  exit 1
fi

REPO="$1"
PR="$2"
GH="${GH_BIN:-gh}"
ME="${PR_MONITOR_ME:-$($GH api user --jq '.login' 2>/dev/null || echo "?")}"

# Open time + commits
PR_JSON=$($GH api graphql -f query="{
  repository(owner: \"${REPO%%/*}\", name: \"${REPO##*/}\") {
    pullRequest(number: ${PR}) {
      title createdAt updatedAt state merged
      commits(first: 100) { totalCount }
    }
  }
}")
CREATED_AT=$(echo "$PR_JSON" | jq -r '.data.repository.pullRequest.createdAt')
UPDATED_AT=$(echo "$PR_JSON" | jq -r '.data.repository.pullRequest.updatedAt')
TITLE=$(echo "$PR_JSON" | jq -r '.data.repository.pullRequest.title')
COMMIT_COUNT=$(echo "$PR_JSON" | jq -r '.data.repository.pullRequest.commits.totalCount')

# Distinct review waves (formal review submissions, grouped by author + ~5min window)
REVIEWS=$($GH api "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
  --jq '.[] | "\(.user.login)|\(.state)|\(.submitted_at)"' 2>/dev/null || true)
CR_REVIEW_COUNT=$(echo "$REVIEWS" | grep -c '^coderabbitai' || true)
# Maintainer = anyone who isn't us AND isn't a known bot account
OUR_REVIEW_COUNT=$(echo "$REVIEWS" | grep -c "^${ME}" || true)
MAINTAINER_REVIEW_COUNT=$(echo "$REVIEWS" \
  | grep -v '^coderabbitai' \
  | grep -v "^${ME}" \
  | grep -v '^codecov' \
  | grep -cv '^$' || true)

# Severity histogram from all inline CR comments
INLINE=$($GH api "repos/${REPO}/pulls/${PR}/comments?per_page=100" --paginate \
  --jq '.[] | select(.user.login == "coderabbitai[bot]") | .body' 2>/dev/null || true)

count_severity() {
  echo "$INLINE" | grep -cE "$1" || true
}
CRITICAL=$(count_severity '🔴.*Critical|Critical.*🔴')
MAJOR=$(count_severity '🟠.*Major|Major.*🟠')
MINOR=$(count_severity '🟡.*Minor|Minor.*🟡')
TRIVIAL=$(count_severity '🔵.*Trivial|Trivial.*🔵|🧹.*Nitpick|Nitpick.*🧹')

# Our reply count
OUR_REPLIES=$($GH api "repos/${REPO}/pulls/${PR}/comments?per_page=100" --paginate \
  --jq ".[] | select(.user.login == \"${ME}\") | .id" 2>/dev/null | wc -l | tr -d ' ')
OUR_TOP_COMMENTS=$($GH api "repos/${REPO}/issues/${PR}/comments?per_page=100" --paginate \
  --jq ".[] | select(.user.login == \"${ME}\") | .id" 2>/dev/null | wc -l | tr -d ' ')

# Elapsed time
NOW=$(date -u +%s)
OPENED=$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$CREATED_AT" +%s 2>/dev/null || \
         date -u -d "$CREATED_AT" +%s 2>/dev/null || echo 0)
if [[ "$OPENED" != "0" ]]; then
  ELAPSED_HRS=$(( (NOW - OPENED) / 3600 ))
  ELAPSED_MIN=$(( ((NOW - OPENED) % 3600) / 60 ))
  ELAPSED="${ELAPSED_HRS}h${ELAPSED_MIN}m"
else
  ELAPSED="?"
fi

echo "═══ PR #${PR} budget report ═══"
echo ""
echo "title:        ${TITLE}"
echo "opened:       ${CREATED_AT} (${ELAPSED} ago)"
echo "last update:  ${UPDATED_AT}"
echo ""
echo "─── iteration cost ───"
echo "commits:                    ${COMMIT_COUNT}"
echo "CR review submissions:      ${CR_REVIEW_COUNT}"
echo "maintainer reviews:         ${MAINTAINER_REVIEW_COUNT}"
echo "our review submissions:     ${OUR_REVIEW_COUNT}"
echo "our inline replies:         ${OUR_REPLIES}"
echo "our top-level comments:     ${OUR_TOP_COMMENTS}"
echo ""
echo "─── CR findings by severity ───"
echo "🔴 Critical: ${CRITICAL}"
echo "🟠 Major:    ${MAJOR}"
echo "🟡 Minor:    ${MINOR}"
echo "🔵 Trivial:  ${TRIVIAL}"
echo ""
echo "─── verdict ───"

# Decision heuristics:
#   - If CR wave count >= 3 AND majority of recent findings are Minor/Trivial → defer
#   - If maintainer hasn't engaged in last 24h AND we've done 2+ cycles → escalate
#   - If 0 unresolved threads → ready to wait for maintainer

WARN=""
if (( CR_REVIEW_COUNT >= 3 )); then
  WARN+="⚠️  ${CR_REVIEW_COUNT} CR review waves on this PR — diminishing-returns territory.\n"
  WARN+="   Consider asking the user before applying further CR-only findings.\n"
fi
if (( MINOR + TRIVIAL > CRITICAL + MAJOR )) && (( CR_REVIEW_COUNT >= 2 )); then
  WARN+="⚠️  Majority of findings on this PR are Minor/Trivial after wave 2.\n"
  WARN+="   Severity floor of Major or Critical may be appropriate going forward.\n"
fi

UNRESOLVED=$($GH api graphql -f query="{
  repository(owner: \"${REPO%%/*}\", name: \"${REPO##*/}\") {
    pullRequest(number: ${PR}) {
      reviewThreads(first: 100) { nodes { isResolved } }
    }
  }
}" --jq '.data.repository.pullRequest.reviewThreads.nodes | map(select(.isResolved == false)) | length')
echo "unresolved threads: ${UNRESOLVED}"
if [[ "$UNRESOLVED" == "0" ]]; then
  echo "✅ All threads resolved; PR is in wait-for-maintainer state."
fi

if [[ -n "$WARN" ]]; then
  echo ""
  echo -e "$WARN"
fi

# Severity policy reminder
echo "─── recommended severity policy ───"
echo "  wave 1:   address all (Critical, Major, Minor, Trivial)"
echo "  wave 2:   address Critical + Major; batch Minor/Trivial for user review"
echo "  wave 3+:  address Critical only; defer everything else to user"
echo ""
echo "See resources/safeguards.md for the full decision tree."

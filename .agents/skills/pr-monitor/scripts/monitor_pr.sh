#!/usr/bin/env bash
# monitor_pr.sh — long-running notification-driven PR watcher.
#
# Usage: monitor_pr.sh <owner>/<repo> <pr-number> [<pr-number>...]
#
# Polls `/notifications?all=true` every 60s for the authenticated user.
# For each new notification on one of the specified PRs, drills into
# `subject.latest_comment_url` to fetch the actual content and emits one
# line. Also tracks merge/close state and reviewDecision via GraphQL.
# Exits when ALL monitored PRs reach a terminal state (MERGED or CLOSED).
#
# Designed to be wrapped with Claude Code's `Monitor` tool:
#   Monitor(persistent=true, command="...monitor_pr.sh aallan/vera-bench 70 73")
#
# Why notifications instead of per-endpoint polling:
# - GitHub itself decides what's notification-worthy (mention,
#   review_requested, state_change, etc.) — one source of truth
# - Each notification carries its own updated_at, so monitor restart
#   doesn't lose events the way a global `since=$(date)` cursor does
# - Higher rate limit than per-PR comment endpoints
# - Covers the `/pulls/N/reviews` endpoint events transparently (formal
#   CHANGES_REQUESTED / APPROVED submissions trigger notifications too)

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <owner>/<repo> <pr-number> [<pr-number>...]" >&2
  exit 1
fi

REPO="$1"
shift
PRS=("$@")

# Build a regex of PR URL paths to match notifications against
PR_PATTERN=$(printf '/pulls/%s\n' "${PRS[@]}" | paste -sd'|' -)

GH="${GH_BIN:-gh}"
POLL_INTERVAL="${POLL_INTERVAL:-60}"

# Start since "now" — first iteration emits the initial state line below
# but no historical events. If you need to catch events from before the
# monitor started, run audit_pr.sh first.
since=$(date -u +%Y-%m-%dT%H:%M:%SZ)
prev_state=""

pr_from_url() {
  # https://api.github.com/repos/aallan/vera-bench/pulls/70 -> 70
  echo "$1" | sed -E 's|.*/pulls/([0-9]+).*|\1|'
}

while true; do
  # 1) Drain new notifications
  notifs=$($GH api \
    "/notifications?all=true&participating=false&since=${since}&per_page=50" \
    --jq ".[] | select(.repository.full_name == \"${REPO}\" and (.subject.url | test(\"${PR_PATTERN}\"))) | \"\(.updated_at)|\(.reason)|\(.subject.url)|\(.subject.latest_comment_url // \"\")\"" \
    2>/dev/null || true)

  if [[ -n "$notifs" ]]; then
    while IFS='|' read -r updated reason subject_url latest_url; do
      [[ -z "$updated" ]] && continue
      pr=$(pr_from_url "$subject_url")
      if [[ -n "$latest_url" && "$latest_url" != "null" ]]; then
        snippet=$($GH api "$latest_url" \
          --jq '"\(.user.login // .author.login // "?"): \(.body[:240] | gsub("\n"; " "))"' \
          2>/dev/null || echo "(fetch failed)")
      else
        snippet="(state change / no body — likely a formal review submission; run audit_pr.sh to inspect)"
      fi
      echo "[$(date +%H:%M:%S)] [#${pr} ${reason} @${updated}] ${snippet}"
    done <<< "$notifs"
  fi
  since=$(date -u +%Y-%m-%dT%H:%M:%SZ)

  # 2) Merge state — emit on transition; exit when all PRs terminal
  graphql_query='{ repository(owner: "'${REPO%%/*}'", name: "'${REPO##*/}'") { '
  for pr in "${PRS[@]}"; do
    graphql_query+="p${pr}: pullRequest(number: ${pr}) { state merged reviewDecision } "
  done
  graphql_query+='} }'

  state=$($GH api graphql -f query="$graphql_query" --jq '
    .data.repository | to_entries | map("\(.key)=\(.value.state)/merged=\(.value.merged)/review=\(.value.reviewDecision)") | join(" | ")
  ' 2>/dev/null || echo "POLL_ERR")

  if [[ "$state" != "$prev_state" ]]; then
    echo "[$(date +%H:%M:%S)] STATE: $state"
    prev_state="$state"
  fi

  # All PRs in a MERGED or CLOSED state => exit
  all_terminal=true
  for pr in "${PRS[@]}"; do
    if [[ ! "$state" =~ p${pr}=(MERGED|CLOSED) ]]; then
      all_terminal=false
      break
    fi
  done
  if [[ "$all_terminal" == "true" ]]; then
    echo "ALL_TERMINAL"
    break
  fi

  sleep "$POLL_INTERVAL"
done

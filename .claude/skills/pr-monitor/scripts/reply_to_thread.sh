#!/usr/bin/env bash
# reply_to_thread.sh — post an inline reply that triggers CodeRabbit
# auto-resolution.
#
# Usage: reply_to_thread.sh <owner>/<repo> <pr-number> <thread-comment-id> <commit-sha> "<body>"
#
# Posts a reply on the inline-comment thread identified by
# <thread-comment-id> (the databaseId of the first comment in the
# thread). The body is prefixed with `@coderabbitai Addressed in
# [SHA](commit-url).` so CodeRabbit's auto-resolution detector fires.
#
# To find unresolved thread IDs and their first-comment databaseIds:
#   gh api graphql -f query='{
#     repository(owner: "OWNER", name: "REPO") {
#       pullRequest(number: N) {
#         reviewThreads(first: 100) {
#           nodes {
#             isResolved path line
#             comments(first: 1) { nodes { databaseId } }
#           }
#         }
#       }
#     }
#   }' --jq '.data.repository.pullRequest.reviewThreads.nodes
#            | map(select(.isResolved == false))
#            | .[] | {path, line, id: .comments.nodes[0].databaseId}'
#
# Why this script exists:
# - CR's auto-resolution only fires on inline replies that mention
#   @coderabbitai AND reference a commit by SHA. Commit-messaging
#   "Address CodeRabbit review" does NOT trigger the resolver.
# - We post via JSON file (not --body) to handle multi-line bodies
#   with shell-special characters safely.

set -euo pipefail

if [[ $# -lt 5 ]]; then
  echo "Usage: $0 <owner>/<repo> <pr-number> <thread-comment-id> <commit-sha> \"<body>\"" >&2
  exit 1
fi

REPO="$1"
PR="$2"
THREAD_ID="$3"
SHA="$4"
BODY="$5"

GH="${GH_BIN:-gh}"

# Compose the reply with the CR-trigger prefix
PREFIX="@coderabbitai Addressed in [${SHA}](https://github.com/${REPO}/pull/${PR}/commits/${SHA})."
FULL_BODY=$(printf '%s\n\n%s' "$PREFIX" "$BODY")

# Build JSON via jq to escape the body safely
PAYLOAD=$(jq -n --argjson in_reply_to "$THREAD_ID" --arg body "$FULL_BODY" \
  '{in_reply_to: $in_reply_to, body: $body}')

echo "Posting to ${REPO}#${PR} thread comment ${THREAD_ID}..."
echo "$PAYLOAD" | $GH api -X POST "repos/${REPO}/pulls/${PR}/comments" \
  --input - --jq '"  -> \(.html_url)"'
